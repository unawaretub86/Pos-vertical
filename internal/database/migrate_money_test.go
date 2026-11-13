package database

// US-102 (R-133) contract §5, T-12 .. T-16 and T-24: migration 24 rounds every
// money value to a whole peso (half away from zero), recomputes sale lines and
// totals with the §3 rule, settles sub-peso fiado residues and writes one audit
// row when a customer balance drifts by a whole peso or more.
//
// Every test builds a database migrated to v23, seeds it with literal INSERTs
// (FK enforcement off, like the other upgrade tests), closes it and reopens it
// through New, which applies migration 24.

import (
	"database/sql"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

// buildV23 creates <tmp>/v23.db at schema version 23, runs seed on it and
// closes it, ready for New.
func buildV23(t *testing.T, seed func(exec func(q string, args ...any))) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "v23.db")
	raw, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	old := &DB{DB: raw, path: p}
	if _, err := raw.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	for _, m := range dbMigrations {
		if m.version > 23 {
			break
		}
		if err := old.runMigration(m); err != nil {
			t.Fatalf("v%d: %v", m.version, err)
		}
	}
	var v int
	if err := raw.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil || v != 23 {
		t.Fatalf("fixture version = %d err=%v, want 23", v, err)
	}
	seed(func(q string, args ...any) {
		t.Helper()
		if _, err := raw.Exec(q, args...); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	})
	raw.Close() // flush before reopening through New
	return p
}

func openMigrated(t *testing.T, p string) *DB {
	t.Helper()
	db, err := New(p)
	if err != nil {
		t.Fatalf("upgrade v23 -> v24 failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func qFloat(t *testing.T, db *DB, q string, args ...any) float64 {
	t.Helper()
	var v float64
	if err := db.QueryRow(q, args...).Scan(&v); err != nil {
		t.Fatalf("query: %v\nSQL: %s", err, q)
	}
	return v
}

func qInt(t *testing.T, db *DB, q string, args ...any) int {
	t.Helper()
	var v int
	if err := db.QueryRow(q, args...).Scan(&v); err != nil {
		t.Fatalf("query: %v\nSQL: %s", err, q)
	}
	return v
}

func wantFloat(t *testing.T, what string, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func assertVersion24(t *testing.T, db *DB) {
	t.Helper()
	if v := qInt(t, db, `SELECT MAX(version) FROM schema_migrations`); v != 24 {
		t.Errorf("schema version = %d, want 24", v)
	}
}

// seedProducts: P1 Item $10.001 at 19% (id 1), P4 Queso $9.999 per kg (id 4).
func seedMoneyProducts(exec func(q string, args ...any)) {
	exec(`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (1, 'Item', 'P1', 'product', 10001, 0, 'unit', 20, 0.19, 1)`)
	exec(`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (4, 'Queso', 'P4', 'product', 9999, 0, 'kg', 200, 0, 1)`)
}

// seedFiadoSale inserts a single-line credit sale of one P1-like line.
func seedCreditSale(exec func(q string, args ...any), saleID, customerID int64, unit, tax, total float64) {
	exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, tax, total, status, payment_method) VALUES (?, 1, ?, ?, ?, ?, 'completed', 'credit')`,
		saleID, customerID, unit, tax, total)
	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (?, 1, 1, ?, ?, ?)`,
		saleID, unit, unit, taxRateOf(tax))
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'credit', ?)`, saleID, total)
}

func taxRateOf(tax float64) float64 {
	if tax == 0 {
		return 0
	}
	return 0.19
}

// T-12 (AC-8): S1 a taxed line, S2 a 0.5 kg line; values rounded, lines
// recomputed, quantity and tax_rate untouched, second open is a no-op.
func TestMigration24_T12_RoundsAndRecomputesSales(t *testing.T) {
	p := buildV23(t, func(exec func(q string, args ...any)) {
		exec(`INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
		seedMoneyProducts(exec)
		// S1: 1 x $10.001 at 19%, tax 1900.19, total 11901.19, on credit.
		exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, tax, total, status, payment_method) VALUES (1, 1, 1, 10001, 1900.19, 11901.19, 'completed', 'credit')`)
		exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (1, 1, 1, 10001, 10001, 0.19)`)
		exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 11901.19)`)
		// S2: 0.5 kg Queso at $9.999, subtotal and total 4999.5, cash.
		exec(`INSERT INTO sales (id, user_id, subtotal, tax, total, status, payment_method) VALUES (2, 1, 4999.5, 0, 4999.5, 'completed', 'cash')`)
		exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (2, 4, 0.5, 9999, 4999.5, 0)`)
		exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 4999.5)`)
	})

	db := openMigrated(t, p)
	assertVersion24(t, db)
	check := func(label string) {
		t.Helper()
		wantFloat(t, label+" S1 subtotal", qFloat(t, db, `SELECT subtotal FROM sales WHERE id = 1`), 10001)
		wantFloat(t, label+" S1 tax", qFloat(t, db, `SELECT tax FROM sales WHERE id = 1`), 1900)
		wantFloat(t, label+" S1 total", qFloat(t, db, `SELECT total FROM sales WHERE id = 1`), 11901)
		wantFloat(t, label+" S1 payment", qFloat(t, db, `SELECT amount FROM sale_payments WHERE sale_id = 1`), 11901)
		wantFloat(t, label+" S1 line subtotal", qFloat(t, db, `SELECT subtotal FROM sale_items WHERE sale_id = 1`), 10001)
		wantFloat(t, label+" S2 subtotal", qFloat(t, db, `SELECT subtotal FROM sales WHERE id = 2`), 5000)
		wantFloat(t, label+" S2 total", qFloat(t, db, `SELECT total FROM sales WHERE id = 2`), 5000)
		wantFloat(t, label+" S2 payment", qFloat(t, db, `SELECT amount FROM sale_payments WHERE sale_id = 2`), 5000)
		wantFloat(t, label+" S2 line subtotal", qFloat(t, db, `SELECT subtotal FROM sale_items WHERE sale_id = 2`), 5000)
		wantFloat(t, label+" S2 quantity", qFloat(t, db, `SELECT quantity FROM sale_items WHERE sale_id = 2`), 0.5)
		wantFloat(t, label+" S1 tax_rate", qFloat(t, db, `SELECT tax_rate FROM sale_items WHERE sale_id = 1`), 0.19)
		wantFloat(t, label+" P1 tax_rate", qFloat(t, db, `SELECT tax_rate FROM products WHERE id = 1`), 0.19)
	}
	check("after migration:")
	db.Close()

	// Second open: nothing to do, same values, still 24 migrations applied once.
	db2 := openMigrated(t, p)
	db = db2
	check("after second open:")
	assertVersion24(t, db2)
	if n := qInt(t, db2, `SELECT COUNT(*) FROM schema_migrations WHERE version = 24`); n != 1 {
		t.Errorf("schema_migrations rows for version 24 = %d, want 1", n)
	}
}

// T-13 (AC-8, AC-9): sub-peso residues end at exactly 0; no drift, no audit row.
func TestMigration24_T13_ResiduesSettleToZero(t *testing.T) {
	p := buildV23(t, func(exec func(q string, args ...any)) {
		exec(`INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
		exec(`INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '', '')`)
		seedMoneyProducts(exec)
		seedCreditSale(exec, 1, 1, 10001, 1900.19, 11901.19)
		seedCreditSale(exec, 2, 2, 10001, 1900.19, 11901.19)
		// Dora: abonos 5000 and 6901.19 (settled with Total). Beto: abono 11901.
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (1, 1, 5000, 'cash')`)
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (2, 1, 6901.19, 'cash')`)
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (3, 2, 11901, 'cash')`)
	})

	// Before: Dora about 0, Beto 0.19 (read on the v23 file).
	raw, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	var dora, beto float64
	if err := raw.QueryRow(customerBalanceSQL, 1, 1).Scan(&dora); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRow(customerBalanceSQL, 2, 2).Scan(&beto); err != nil {
		t.Fatal(err)
	}
	raw.Close()
	if math.Abs(dora) > 1e-6 {
		t.Fatalf("fixture Dora balance before = %v, want about 0", dora)
	}
	if math.Abs(beto-0.19) > 1e-6 {
		t.Fatalf("fixture Beto balance before = %v, want 0.19", beto)
	}

	db := openMigrated(t, p)
	assertVersion24(t, db)
	wantFloat(t, "Dora balance after", qFloat(t, db, customerBalanceSQL, 1, 1), 0)
	wantFloat(t, "Beto balance after", qFloat(t, db, customerBalanceSQL, 2, 2), 0)
	wantFloat(t, "Dora abono 2", qFloat(t, db, `SELECT amount FROM customer_payments WHERE id = 2`), 6901)
	wantFloat(t, "Beto credit line", qFloat(t, db, `SELECT amount FROM sale_payments WHERE sale_id = 2`), 11901)
	if n := qInt(t, db, `SELECT COUNT(*) FROM audit_log WHERE action = 'money_rounding_migration'`); n != 0 {
		t.Errorf("audit rows with no drift = %d, want 0 (a residue under $1 is not drift)", n)
	}
}

// T-14 (AC-9): drift. Eva (id 7) has two credit sales of 100.4 and an abono of
// 200.8: balance 0 before; after rounding 100 + 100 - 201 = -1.
func TestMigration24_T14_DriftWritesOneAuditRow(t *testing.T) {
	seed := func(exec func(q string, args ...any)) {
		exec(`INSERT INTO customers (id, name, phone, address) VALUES (7, 'Eva', '', '')`)
		seedMoneyProducts(exec)
		for _, id := range []int64{1, 2} {
			exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, tax, total, status, payment_method) VALUES (?, 1, 7, 100.4, 0, 100.4, 'completed', 'credit')`, id)
			exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (?, 1, 1, 100.4, 100.4, 0)`, id)
			exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'credit', 100.4)`, id)
		}
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (1, 7, 200.8, 'cash')`)
	}
	p := buildV23(t, seed)
	db := openMigrated(t, p)
	assertVersion24(t, db)
	wantFloat(t, "Eva balance after", qFloat(t, db, customerBalanceSQL, 7, 7), -1)

	if n := qInt(t, db, `SELECT COUNT(*) FROM audit_log WHERE action = 'money_rounding_migration'`); n != 1 {
		t.Fatalf("audit rows = %d, want 1", n)
	}
	var userID, entityID int64
	var entity, details string
	if err := db.QueryRow(`SELECT user_id, entity, entity_id, COALESCE(details, '') FROM audit_log WHERE action = 'money_rounding_migration'`).
		Scan(&userID, &entity, &entityID, &details); err != nil {
		t.Fatal(err)
	}
	if userID != 1 || entity != "system" || entityID != 0 {
		t.Errorf("audit row user/entity/entity_id = %d/%q/%d, want 1/system/0", userID, entity, entityID)
	}
	if !strings.Contains(details, "7") {
		t.Errorf("audit details %q do not list Eva's id 7", details)
	}

	// No active admin: the migration still succeeds and writes no row.
	p2 := buildV23(t, func(exec func(q string, args ...any)) {
		seed(exec)
		exec(`UPDATE users SET active = 0 WHERE role = 'admin'`)
	})
	db2 := openMigrated(t, p2)
	assertVersion24(t, db2)
	wantFloat(t, "Eva balance after (no admin)", qFloat(t, db2, customerBalanceSQL, 7, 7), -1)
	if n := qInt(t, db2, `SELECT COUNT(*) FROM audit_log WHERE action = 'money_rounding_migration'`); n != 0 {
		t.Errorf("audit rows with no active admin = %d, want 0", n)
	}
}

// T-15 (AC-8): register figures and the other money columns are whole.
func TestMigration24_T15_RegisterAndOtherColumns(t *testing.T) {
	p := buildV23(t, func(exec func(q string, args ...any)) {
		exec(`INSERT INTO registers (id, user_id, opening_amount, expected_close, actual_close, difference, status) VALUES (1, 1, 100000.5, 157000.4, 157000, -0.4, 'closed')`)
		exec(`INSERT INTO register_movements (register_id, type, amount, description) VALUES (1, 'cash_in', 1000.5, 'x')`)
		exec(`INSERT INTO expenses (register_id, user_id, category, amount, note) VALUES (1, 1, 'servicios', 3000.5, '')`)
		exec(`INSERT INTO products (id, name, code, type, sale_price, wholesale_price, cost, unit, stock, tax_rate, active) VALUES (1, 'Item', 'P1', 'product', 2500.5, 2400.4, 1200.4, 'unit', 20, 0.19, 1)`)
	})
	db := openMigrated(t, p)
	assertVersion24(t, db)
	wantFloat(t, "expected_close", qFloat(t, db, `SELECT expected_close FROM registers WHERE id = 1`), 157000)
	wantFloat(t, "actual_close", qFloat(t, db, `SELECT actual_close FROM registers WHERE id = 1`), 157000)
	wantFloat(t, "difference", qFloat(t, db, `SELECT difference FROM registers WHERE id = 1`), 0)
	wantFloat(t, "opening_amount 100000.5 (half away from zero)", qFloat(t, db, `SELECT opening_amount FROM registers WHERE id = 1`), 100001)
	wantFloat(t, "cash_in 1000.5", qFloat(t, db, `SELECT amount FROM register_movements WHERE type = 'cash_in'`), 1001)
	wantFloat(t, "expense 3000.5", qFloat(t, db, `SELECT amount FROM expenses`), 3001)
	wantFloat(t, "sale_price 2500.5", qFloat(t, db, `SELECT sale_price FROM products WHERE id = 1`), 2501)
	wantFloat(t, "wholesale_price 2400.4", qFloat(t, db, `SELECT wholesale_price FROM products WHERE id = 1`), 2400)
	wantFloat(t, "cost 1200.4", qFloat(t, db, `SELECT cost FROM products WHERE id = 1`), 1200)
	wantFloat(t, "tax_rate untouched", qFloat(t, db, `SELECT tax_rate FROM products WHERE id = 1`), 0.19)
	wantFloat(t, "stock untouched", qFloat(t, db, `SELECT stock FROM products WHERE id = 1`), 20)
}

// T-16 (AC-8): whole values and credit_limit are untouched; an empty database
// migrates.
func TestMigration24_T16_WholeValuesUntouchedAndEmptyDB(t *testing.T) {
	p := buildV23(t, func(exec func(q string, args ...any)) {
		exec(`INSERT INTO customers (id, name, phone, address, credit_limit) VALUES (1, 'Dora', '', '', 500000)`)
		exec(`INSERT INTO customer_opening_balances (customer_id, amount, debt_date, note, created_by) VALUES (1, 150000, '2026-09-25 05:00:00', '', 1)`)
	})
	db := openMigrated(t, p)
	assertVersion24(t, db)
	wantFloat(t, "opening balance", qFloat(t, db, `SELECT amount FROM customer_opening_balances WHERE customer_id = 1`), 150000)
	wantFloat(t, "credit_limit", qFloat(t, db, `SELECT credit_limit FROM customers WHERE id = 1`), 500000)
	wantFloat(t, "Dora balance", qFloat(t, db, `SELECT amount FROM customer_opening_balances WHERE customer_id = 1`), 150000)

	empty := openMigrated(t, buildV23(t, func(exec func(q string, args ...any)) {}))
	assertVersion24(t, empty)
	if n := qInt(t, empty, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales in an empty migrated database = %d, want 0", n)
	}
	if n := qInt(t, empty, `SELECT COUNT(*) FROM audit_log WHERE action = 'money_rounding_migration'`); n != 0 {
		t.Errorf("audit rows in an empty database = %d, want 0", n)
	}
}

// T-24 (AC-9): audit row user is the lowest-id ACTIVE admin; inactive
// customers are scanned and settled.
func TestMigration24_T24_AuditRowAdminAndInactiveCustomers(t *testing.T) {
	p := buildV23(t, func(exec func(q string, args ...any)) {
		// Admin 1 (seeded by migration 1) becomes inactive; admin 2 is active.
		exec(`UPDATE users SET active = 0 WHERE id = 1`)
		exec(`INSERT INTO users (id, username, password_hash, role, active) VALUES (2, 'admin2', 'x', 'admin', 1)`)
		exec(`INSERT INTO users (id, username, password_hash, role, active) VALUES (3, 'admin3', 'x', 'admin', 1)`)
		seedMoneyProducts(exec)
		// Eva (id 7, active): the drift case of T-14.
		exec(`INSERT INTO customers (id, name, phone, address) VALUES (7, 'Eva', '', '')`)
		// Gus (id 9, inactive): same drift, and a residue customer (id 41, inactive).
		exec(`INSERT INTO customers (id, name, phone, address, active) VALUES (9, 'Gus', '', '', 0)`)
		exec(`INSERT INTO customers (id, name, phone, address, active) VALUES (41, 'Hugo', '', '', 0)`)
		sid := int64(0)
		for _, cid := range []int64{7, 9} {
			for i := 0; i < 2; i++ {
				sid++
				exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, tax, total, status, payment_method) VALUES (?, 2, ?, 100.4, 0, 100.4, 'completed', 'credit')`, sid, cid)
				exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (?, 1, 1, 100.4, 100.4, 0)`, sid)
				exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'credit', 100.4)`, sid)
			}
		}
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (1, 7, 200.8, 'cash')`)
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (2, 9, 200.8, 'cash')`)
		// Hugo (inactive): credit 11901.19, abono 11901 -> residue 0.19.
		seedCreditSale(exec, 50, 41, 10001, 1900.19, 11901.19)
		exec(`INSERT INTO customer_payments (id, customer_id, amount, method) VALUES (3, 41, 11901, 'cash')`)
	})
	db := openMigrated(t, p)
	assertVersion24(t, db)

	if n := qInt(t, db, `SELECT COUNT(*) FROM audit_log WHERE action = 'money_rounding_migration'`); n != 1 {
		t.Fatalf("audit rows = %d, want 1", n)
	}
	var userID int64
	var entity, details string
	if err := db.QueryRow(`SELECT user_id, entity, COALESCE(details, '') FROM audit_log WHERE action = 'money_rounding_migration'`).
		Scan(&userID, &entity, &details); err != nil {
		t.Fatal(err)
	}
	if userID != 2 || entity != "system" {
		t.Errorf("audit row user/entity = %d/%q, want 2/system (lowest-id active admin)", userID, entity)
	}
	for _, id := range []string{"7", "9"} {
		if !strings.Contains(details, id) {
			t.Errorf("audit details %q do not list drifted customer %s", details, id)
		}
	}
	if strings.Contains(details, "41") {
		t.Errorf("audit details %q list the residue-only customer 41", details)
	}
	wantFloat(t, "inactive Gus balance", qFloat(t, db, customerBalanceSQL, 9, 9), -1)
	wantFloat(t, "inactive Hugo balance (residue settled)", qFloat(t, db, customerBalanceSQL, 41, 41), 0)
}
