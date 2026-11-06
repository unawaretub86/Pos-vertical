package database

// Contrato US-055 §4, T-1 y T-2.

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigration18_T1_TablesIndexesAndChecks(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var v int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 18`).Scan(&v); err != nil {
		t.Fatalf("version 18: %v", err)
	}
	if v != 1 {
		t.Errorf("schema_migrations rows with version 18 = %d, want 1", v)
	}

	for _, name := range []string{
		"dining_tables", "orders", "order_items",
		"idx_orders_status", "idx_orders_table", "idx_orders_updated",
		"idx_order_items_order", "idx_order_items_status",
	} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = ?`, name).Scan(&n); err != nil {
			t.Fatalf("sqlite_master %s: %v", name, err)
		}
		if n != 1 {
			t.Errorf("sqlite_master count for %q = %d, want 1", name, n)
		}
	}

	if _, err := db.Exec(`INSERT INTO products(id, name, sale_price) VALUES(1, 'P1', 1000)`); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO orders(type, user_id) VALUES('bogus', 1)`); err == nil {
		t.Error("orders.type 'bogus' should violate CHECK")
	}
	if _, err := db.Exec(`INSERT INTO orders(id, type, user_id) VALUES(1, 'table', 1)`); err != nil {
		t.Fatalf("valid order insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO order_items(order_id, product_id, quantity, unit_price, status) VALUES(1, 1, 1, 1000, 'cooking')`); err == nil {
		t.Error("order_items.status 'cooking' should violate CHECK")
	}
	if _, err := db.Exec(`INSERT INTO order_items(order_id, product_id, quantity, unit_price, status) VALUES(1, 1, 1, 1000, 'ready')`); err != nil {
		t.Errorf("valid order_item insert: %v", err)
	}
}

func TestMigration18_T2_BusinessConfigDefaults(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`INSERT INTO business_config(id, name, currency) VALUES(1, 'Cafe', 'COP')`); err != nil {
		t.Fatalf("insert business_config: %v", err)
	}
	var rm, lan int
	if err := db.QueryRow(`SELECT restaurant_mode, lan_access_enabled FROM business_config WHERE id = 1`).Scan(&rm, &lan); err != nil {
		t.Fatalf("select new columns: %v", err)
	}
	if rm != 0 || lan != 0 {
		t.Errorf("restaurant_mode=%d lan_access_enabled=%d, want 0 and 0", rm, lan)
	}
}

// US-080 T-16 (AC-2): migration 20 adds the printer settings with safe defaults
// to an existing business_config row.
func TestMigration20_T16_PrinterSettingsDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), "v19.db")
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
		if m.version > 19 {
			break
		}
		if err := old.runMigration(m); err != nil {
			t.Fatalf("v%d: %v", m.version, err)
		}
	}
	if _, err := raw.Exec(`INSERT INTO business_config(id, name, currency) VALUES(1, 'Cafe', 'COP')`); err != nil {
		t.Fatalf("seed business_config: %v", err)
	}
	raw.Close()

	db, err := New(p)
	if err != nil {
		t.Fatalf("upgrade v19 -> latest: %v", err)
	}
	defer db.Close()

	var v int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 20`).Scan(&v); err != nil || v != 1 {
		t.Errorf("schema_migrations rows with version 20 = %d err=%v, want 1", v, err)
	}
	var method, host, name string
	var width, port int
	err = db.QueryRow(`SELECT print_method, paper_width_mm, printer_host, printer_port, printer_name FROM business_config WHERE id = 1`).
		Scan(&method, &width, &host, &port, &name)
	if err != nil {
		t.Fatalf("select printer columns: %v", err)
	}
	if method != "browser" || width != 58 || port != 9100 || host != "" || name != "" {
		t.Errorf("defaults = %q/%d/%q/%d/%q, want browser/58/''/9100/''", method, width, host, port, name)
	}
}

// ─── US-082 contract §4, T-1 (AC-1) ──────────────────────────────────────────

// customerBalanceSQL mirrors repository.customerRepo.Balance (credit sale
// payments minus abonos), so the balance is measured the same way before and
// after migration 21.
const customerBalanceSQL = `
SELECT
	COALESCE((
		SELECT SUM(sp.amount) FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.customer_id = ? AND sp.method = 'credit' AND s.status = 'completed'
	), 0)
	- COALESCE((SELECT SUM(amount) FROM customer_payments WHERE customer_id = ?), 0)`

func TestMigration21_T1_AbonoMethodAndRegisterBackfill(t *testing.T) {
	p := filepath.Join(t.TempDir(), "v20.db")
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
		if m.version > 20 {
			break
		}
		if err := old.runMigration(m); err != nil {
			t.Fatalf("v%d: %v", m.version, err)
		}
	}
	mustExec := func(q string) {
		t.Helper()
		if _, err := raw.Exec(q); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	// Customer with a fiado of $200,000 and two abonos: $20,000 and $1,000.
	mustExec(`INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	mustExec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method) VALUES (1, 1, 1, 200000, 200000, 'completed', 'credit')`)
	mustExec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 200000)`)
	mustExec(`INSERT INTO customer_payments (id, customer_id, amount, note) VALUES (1, 1, 20000, 'primer abono')`)
	mustExec(`INSERT INTO customer_payments (id, customer_id, amount, note) VALUES (2, 1, 1000, '')`)

	var before float64
	if err := raw.QueryRow(customerBalanceSQL, 1, 1).Scan(&before); err != nil {
		t.Fatalf("balance before: %v", err)
	}
	if before != 179000 {
		t.Fatalf("fixture balance before migration = %.0f, want 179000", before)
	}
	raw.Close() // flush before reopening through New, which runs migration 21

	db, err := New(p)
	if err != nil {
		t.Fatalf("upgrade v20 -> v21 failed: %v", err)
	}
	defer db.Close()

	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("schema version: %v", err)
	}
	if version < 21 {
		t.Fatalf("schema version = %d, want >= 21", version)
	}

	rows, err := db.Query(`SELECT id, amount, method, register_id FROM customer_payments ORDER BY id`)
	if err != nil {
		t.Fatalf("select method/register_id: %v", err)
	}
	type row struct {
		id       int64
		amount   float64
		method   string
		register sql.NullInt64
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.amount, &r.method, &r.register); err != nil {
			rows.Close()
			t.Fatalf("scan: %v", err)
		}
		got = append(got, r)
	}
	rows.Close()
	if len(got) != 2 {
		t.Fatalf("rows after migration = %d, want 2", len(got))
	}
	wantAmounts := []float64{20000, 1000}
	for i, r := range got {
		if r.amount != wantAmounts[i] {
			t.Errorf("row %d amount = %.0f, want %.0f", r.id, r.amount, wantAmounts[i])
		}
		if r.method != "cash" {
			t.Errorf("row %d method = %q, want cash", r.id, r.method)
		}
		if r.register.Valid {
			t.Errorf("row %d register_id = %d, want NULL", r.id, r.register.Int64)
		}
	}

	var after float64
	if err := db.QueryRow(customerBalanceSQL, 1, 1).Scan(&after); err != nil {
		t.Fatalf("balance after: %v", err)
	}
	if after != 179000 {
		t.Errorf("balance after migration = %.0f, want 179000", after)
	}

	if _, err := db.Exec(`INSERT INTO customer_payments (customer_id, amount, method) VALUES (1, 500, 'nequi')`); err == nil {
		t.Error("insert with method='nequi' succeeded, want CHECK constraint failure")
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM customer_payments`).Scan(&n); err != nil || n != 2 {
		t.Errorf("row count after rejected insert = %d err=%v, want 2", n, err)
	}
}
