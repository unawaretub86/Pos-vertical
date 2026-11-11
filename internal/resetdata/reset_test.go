package resetdata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
	_ "modernc.org/sqlite"
)

// ── shared fixture helpers (used by every *_test.go in this package) ──────────

// openRaw opens the database the way the tool does: raw sql.Open, no migrations,
// busy_timeout plus enforced foreign keys.
func openRaw(t *testing.T, path string, busyMs int) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout("+strconv.Itoa(busyMs)+")&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newFixtureF creates a migrated temp-file database named data.db, seeds
// fixture F (contract section 3) and returns a raw connection plus the path.
func newFixtureF(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	d, err := database.New(path)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	db := openRaw(t, path, 5000)
	seedFixtureF(t, db)
	return db, path
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

// seedFixtureF loads fixture F from the contract with literal INSERTs.
func seedFixtureF(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`INSERT OR REPLACE INTO business_config (id, name, jwt_secret, catalog_token, setup_completed) VALUES (1, 'Tienda Prueba', 'jwt-secret-xyz', 'cat-token-1', 1)`,
		`INSERT OR REPLACE INTO users (id, username, password_hash, role, active) VALUES (1, 'admin', 'hash-admin', 'admin', 1)`,
		`INSERT INTO users (id, username, password_hash, role, active) VALUES (2, 'cashier', 'hash-cashier', 'cashier', 1)`,
		`UPDATE fiscal_config SET enabled = 0, ambiente = 'test' WHERE id = 1`,
		`INSERT INTO categories (id, name) VALUES (1, 'Bebidas'), (2, 'Granos')`,
		`INSERT INTO products (id, name, category_id, type, sale_price, cost, stock) VALUES
			(1, 'Leche', 1, 'product', 3000, 800, 106),
			(2, 'Arroz', 2, 'product', 5000, 1500, 90),
			(3, 'Domicilio', 1, 'service', 2000, 0, 0),
			(4, 'Panela', 2, 'product', 2200, 1000, 20)`,
		`INSERT INTO stock_entries (id, product_id, quantity, note) VALUES
			(1, 1, 100, 'Importación de catálogo'),
			(2, 1, 10, 'Entrada de inventario'),
			(3, 2, 40, 'Importación de catálogo'),
			(4, 2, 60, 'Compra #1')`,
		`INSERT INTO suppliers (id, name) VALUES (1, 'Proveedor Uno')`,
		`INSERT INTO purchases (id, supplier_id, user_id, total) VALUES (1, 1, 1, 90000)`,
		`INSERT INTO purchase_items (id, purchase_id, product_id, quantity, unit_cost, subtotal) VALUES (1, 1, 2, 60, 1500, 90000)`,
		`INSERT INTO customers (id, name) VALUES (1, 'Dora'), (2, 'Beto')`,
		`INSERT INTO customer_opening_balances (id, customer_id, amount, debt_date, note, created_by) VALUES (1, 1, 150000, '2026-11-01 05:00:00', '', 1)`,
		`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 50000, 'closed'), (2, 1, 20000, 'open')`,
		`INSERT INTO register_movements (id, register_id, type, amount, description) VALUES
			(1, 1, 'opening', 50000, 'Apertura'),
			(2, 1, 'sale', 12000, 'Venta #1'),
			(3, 1, 'cash_out', 5000, 'Retiro'),
			(4, 2, 'opening', 20000, 'Apertura'),
			(5, 2, 'sale', 10000, 'Venta #4'),
			(6, 2, 'cash_in', 30000, 'Abono')`,
		`INSERT INTO expenses (id, register_id, user_id, category, amount) VALUES
			(1, 1, 1, 'servicios', 8000),
			(2, 2, 1, 'arriendo', 20000)`,
		`INSERT INTO sales (id, user_id, customer_id, register_id, subtotal, discount, total, payment_method, status, voided_by, created_at) VALUES
			(1, 1, NULL, 1, 12000, 0, 12000, 'cash', 'completed', NULL, '2026-11-09 15:00:00'),
			(2, 1, 1, 2, 50000, 0, 50000, 'credit', 'completed', NULL, '2026-11-09 16:00:00'),
			(3, 1, NULL, 2, 18000, 0, 18000, 'cash', 'voided', 1, '2026-11-09 17:00:00'),
			(4, 1, NULL, 2, 11000, 1000, 10000, 'cash', 'completed', NULL, '2026-11-09 18:00:00')`,
		`INSERT INTO sale_items (id, sale_id, product_id, quantity, unit_price, discount, subtotal) VALUES
			(1, 1, 1, 4, 3000, 0, 12000),
			(2, 2, 2, 10, 5000, 0, 50000),
			(3, 3, 1, 6, 3000, 0, 18000),
			(4, 4, 4, 5, 2200, 1000, 11000)`,
		`INSERT INTO sale_payments (id, sale_id, method, amount) VALUES
			(1, 1, 'cash', 12000), (2, 2, 'credit', 50000), (3, 3, 'cash', 18000), (4, 4, 'cash', 10000)`,
		`INSERT INTO customer_payments (id, customer_id, amount, note, method, register_id) VALUES (1, 1, 30000, 'abono', 'cash', 2)`,
		`INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`,
		`INSERT INTO orders (id, type, table_id, status, user_id, sale_id) VALUES (1, 'table', 1, 'closed', 1, 1)`,
		`INSERT INTO order_items (id, order_id, product_id, quantity, unit_price) VALUES (1, 1, 1, 4, 3000), (2, 1, 2, 1, 5000)`,
		`INSERT INTO assistant_queries (id, user_id, query, intent, tier) VALUES
			(1, 1, 'cuánto vendí ayer', 'sales_yesterday', 'regex'),
			(2, 1, 'quién me debe', 'receivables', 'regex'),
			(3, 1, 'asdf', 'none', 'none')`,
		`INSERT INTO electronic_documents (id, sale_id, status) VALUES ('ed-1', 1, 'rejected')`,
		`INSERT INTO audit_log (id, user_id, action, entity, entity_id, details) VALUES
			(1, 1, 'product_import', 'product', 1, '{}'),
			(2, 1, 'opening_balance_create', 'customer', 1, '{}'),
			(3, 1, 'product_price_change', 'product', 1, '{}'),
			(4, 1, 'void', 'sale', 3, '{}'),
			(5, 1, 'discount', 'sale', 4, '{}'),
			(6, 1, 'cash_out', 'register_movement', 3, '{}')`,
	}
	for _, s := range stmts {
		mustExec(t, db, s)
	}
}

// allCounts returns COUNT(*) of every user table (sqlite_* excluded).
func allCounts(t *testing.T, db *sql.DB) map[string]int {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	rows.Close()
	out := map[string]int{}
	for _, n := range names {
		var c int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + n + `"`).Scan(&c); err != nil {
			t.Fatal(err)
		}
		out[n] = c
	}
	return out
}

func count(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	return n
}

func stockOf(t *testing.T, db *sql.DB, id int) float64 {
	t.Helper()
	var s float64
	if err := db.QueryRow(`SELECT stock FROM products WHERE id = ?`, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// dump renders every row of a query as text, for byte-identical comparisons.
func dump(t *testing.T, db *sql.DB, q string) string {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var sb strings.Builder
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = fmt.Sprint(v)
		}
		sb.WriteString(strings.Join(parts, " "))
		sb.WriteString("\n")
	}
	return sb.String()
}

func balance(t *testing.T, db *sql.DB, customerID int) float64 {
	t.Helper()
	var b float64
	err := db.QueryRow(`
		SELECT COALESCE((SELECT amount FROM customer_opening_balances WHERE customer_id = ?), 0)
		     + COALESCE((SELECT SUM(total) FROM sales WHERE customer_id = ? AND payment_method = 'credit' AND status = 'completed'), 0)
		     - COALESCE((SELECT SUM(amount) FROM customer_payments WHERE customer_id = ?), 0)`,
		customerID, customerID, customerID).Scan(&b)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func findStock(p *PlanReport, id int64) *StockChange {
	for i := range p.Stock {
		if p.Stock[i].ProductID == id {
			return &p.Stock[i]
		}
	}
	return nil
}

// ── tests ─────────────────────────────────────────────────────────────────────

// T-1: Plan on F returns the contract counts and writes nothing.
func TestPlan_T1_CountsAndNoWrites(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()

	before := allCounts(t, db)
	if len(before) != 25 {
		t.Fatalf("expected 25 user tables, got %d: %v", len(before), before)
	}

	p, err := Plan(ctx, db)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	want := map[string]int{
		"sales": 4, "sale_items": 4, "sale_payments": 4, "customer_payments": 1,
		"registers": 2, "register_movements": 6, "expenses": 2, "orders": 1,
		"order_items": 2, "electronic_documents": 1, "assistant_queries": 3, "audit_log": 3,
	}
	for table, n := range want {
		if p.Counts[table] != n {
			t.Errorf("Plan.Counts[%s] = %d, want %d", table, p.Counts[table], n)
		}
	}

	// Stock report: P1 106->110, P2 90->100, P4 20->0 (untraced = 20 + 5 sold - 0 = 25).
	cases := []struct {
		id             int64
		now, after, un float64
	}{{1, 106, 110, 0}, {2, 90, 100, 0}, {4, 20, 0, 25}}
	for _, c := range cases {
		sc := findStock(p, c.id)
		if sc == nil {
			t.Errorf("product %d missing from stock report", c.id)
			continue
		}
		if sc.Now != c.now || sc.After != c.after || sc.Untraced != c.un {
			t.Errorf("product %d: now %v after %v untraced %v, want %v %v %v",
				c.id, sc.Now, sc.After, sc.Untraced, c.now, c.after, c.un)
		}
	}
	if findStock(p, 3) != nil {
		t.Errorf("service product 3 must not appear in the stock report")
	}

	after := allCounts(t, db)
	if !reflect.DeepEqual(before, after) {
		t.Errorf("Plan changed row counts:\nbefore %v\nafter  %v", before, after)
	}
	if got := stockOf(t, db, 1); got != 106 {
		t.Errorf("Plan changed P1 stock to %v", got)
	}
}

// T-2: Apply leaves the "After" column and keeps business_config/fiscal_config/users byte-identical.
func TestApply_T2_AfterCountsAndKeptData(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()

	cfgBefore := dump(t, db, `SELECT * FROM business_config`)
	fiscalBefore := dump(t, db, `SELECT * FROM fiscal_config`)
	usersBefore := dump(t, db, `SELECT * FROM users ORDER BY id`)

	if _, err := Apply(ctx, db, Options{BackupName: "data.db.pre-reset-20261110-143005"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got := allCounts(t, db)
	want := map[string]int{
		"sales": 0, "sale_items": 0, "sale_payments": 0, "customer_payments": 0,
		"registers": 0, "register_movements": 0, "expenses": 0, "orders": 0,
		"order_items": 0, "electronic_documents": 0, "assistant_queries": 0,
		"audit_log": 4,
		"users":     2, "categories": 2, "products": 4, "customers": 2,
		"customer_opening_balances": 1, "suppliers": 1, "purchases": 1,
		"purchase_items": 1, "stock_entries": 4, "dining_tables": 1,
		"business_config": 1, "fiscal_config": 1,
	}
	for table, n := range want {
		if got[table] != n {
			t.Errorf("%s = %d after Apply, want %d", table, got[table], n)
		}
	}

	if name := dump(t, db, `SELECT name FROM business_config`); name != "Tienda Prueba\n" {
		t.Errorf("business_config.name = %q", name)
	}
	if v := dump(t, db, `SELECT * FROM business_config`); v != cfgBefore {
		t.Errorf("business_config changed:\n%s\nvs\n%s", cfgBefore, v)
	}
	if v := dump(t, db, `SELECT * FROM fiscal_config`); v != fiscalBefore {
		t.Errorf("fiscal_config changed:\n%s\nvs\n%s", fiscalBefore, v)
	}
	if v := dump(t, db, `SELECT * FROM users ORDER BY id`); v != usersBefore {
		t.Errorf("users changed:\n%s\nvs\n%s", usersBefore, v)
	}
	// Kept audit rows are exactly the three contract rows plus data_reset.
	if n := count(t, db, `SELECT COUNT(*) FROM audit_log WHERE action IN ('product_import','opening_balance_create','product_price_change')`); n != 3 {
		t.Errorf("kept audit rows = %d, want 3", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM audit_log WHERE action IN ('void','discount','cash_out')`); n != 0 {
		t.Errorf("pointer audit rows left = %d, want 0", n)
	}
}

// T-3: stock rule B on F.
func TestApply_T3_StockRuleB(t *testing.T) {
	db, _ := newFixtureF(t)
	if _, err := Apply(context.Background(), db, Options{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for id, want := range map[int]float64{1: 110, 2: 100, 4: 0, 3: 0} {
		if got := stockOf(t, db, id); got != want {
			t.Errorf("product %d stock = %v, want %v", id, got, want)
		}
	}
	if n := count(t, db, `SELECT COUNT(*) FROM products`); n != 4 {
		t.Errorf("products = %d, want 4", n)
	}
	var cost float64
	if err := db.QueryRow(`SELECT cost FROM products WHERE id = 2`).Scan(&cost); err != nil {
		t.Fatal(err)
	}
	if cost != 1500 {
		t.Errorf("P2 cost = %v, want 1500", cost)
	}
	if got := dump(t, db, `SELECT id, sale_price, cost FROM products ORDER BY id`); got != "1 3000 800\n2 5000 1500\n3 2000 0\n4 2200 1000\n" {
		t.Errorf("prices/costs changed: %q", got)
	}
}

// T-4: stock rule B variants: multiple entries, inactive product, product with no entries.
func TestApply_T4_StockRuleVariants(t *testing.T) {
	db, _ := newFixtureF(t)
	mustExec(t, db, `INSERT INTO products (id, name, category_id, type, sale_price, stock, active) VALUES
		(5, 'P5', 1, 'product', 1000, 5, 1),
		(6, 'P6 inactivo', 1, 'product', 1000, 12, 0),
		(7, 'P7 sin entradas', 1, 'product', 1000, 7, 1)`)
	mustExec(t, db, `INSERT INTO stock_entries (product_id, quantity, note) VALUES (5, 30, 'a'), (5, 20, 'b'), (6, 8, 'c')`)

	p, err := Plan(context.Background(), db)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	p5, p6, p7 := findStock(p, 5), findStock(p, 6), findStock(p, 7)
	if p5 == nil || p6 == nil || p7 == nil {
		t.Fatalf("plan missing products: p5=%v p6=%v p7=%v", p5, p6, p7)
	}
	if p5.After != 50 {
		t.Errorf("P5 after = %v, want 50", p5.After)
	}
	if p6.After != 8 || p6.Active {
		t.Errorf("P6 after = %v active = %v, want 8 and inactive", p6.After, p6.Active)
	}
	if p7.After != 0 || p7.Untraced != 7 {
		t.Errorf("P7 after = %v untraced = %v, want 0 and 7", p7.After, p7.Untraced)
	}

	if _, err := Apply(context.Background(), db, Options{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for id, want := range map[int]float64{5: 50, 6: 8, 7: 0} {
		if got := stockOf(t, db, id); got != want {
			t.Errorf("product %d stock = %v, want %v", id, got, want)
		}
	}
}

// T-5: ids of cleared tables restart at 1, kept tables continue.
func TestApply_T5_SequencesRestart(t *testing.T) {
	db, _ := newFixtureF(t)
	if _, err := Apply(context.Background(), db, Options{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	insert := func(q string) int64 {
		t.Helper()
		res, err := db.Exec(q)
		if err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	if id := insert(`INSERT INTO sales (user_id, subtotal, total, payment_method) VALUES (1, 100, 100, 'cash')`); id != 1 {
		t.Errorf("next sale id = %d, want 1", id)
	}
	if id := insert(`INSERT INTO registers (user_id, opening_amount) VALUES (1, 0)`); id != 1 {
		t.Errorf("next register id = %d, want 1", id)
	}
	if id := insert(`INSERT INTO orders (type, user_id) VALUES ('takeaway', 1)`); id != 1 {
		t.Errorf("next order id = %d, want 1", id)
	}
	if id := insert(`INSERT INTO stock_entries (product_id, quantity, note) VALUES (1, 1, 'x')`); id != 5 {
		t.Errorf("next stock_entries id = %d, want 5", id)
	}
	if id := insert(`INSERT INTO products (name, type, sale_price) VALUES ('Nuevo', 'product', 1)`); id != 5 {
		t.Errorf("next product id = %d, want 5", id)
	}
}

// T-6: customer balances return to the opening balance.
func TestApply_T6_BalancesEqualOpening(t *testing.T) {
	db, _ := newFixtureF(t)
	if got := balance(t, db, 1); got != 170000 {
		t.Fatalf("fixture Dora balance before = %v, want 170000", got)
	}
	if _, err := Apply(context.Background(), db, Options{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := balance(t, db, 1); got != 150000 {
		t.Errorf("Dora balance = %v, want 150000", got)
	}
	if got := balance(t, db, 2); got != 0 {
		t.Errorf("Beto balance = %v, want 0", got)
	}
	if got := dump(t, db, `SELECT customer_id, amount, debt_date FROM customer_opening_balances`); got != "1 150000 2026-11-01 05:00:00\n" {
		t.Errorf("opening balance row = %q", got)
	}
}

// T-10: audit row, second-run guard and --force.
func TestApply_T10_AuditRowAndSecondRun(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()
	backup := "data.db.pre-reset-20261110-143005"

	if _, err := Apply(ctx, db, Options{BackupName: backup}); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM audit_log WHERE action = 'data_reset'`); n != 1 {
		t.Fatalf("data_reset rows = %d, want exactly 1", n)
	}
	var userID, entityID int
	var entity, details string
	if err := db.QueryRow(`SELECT user_id, entity, entity_id, details FROM audit_log WHERE action = 'data_reset'`).Scan(&userID, &entity, &entityID, &details); err != nil {
		t.Fatal(err)
	}
	if userID != 1 || entity != "system" || entityID != 0 {
		t.Errorf("data_reset row = user %d entity %q entity_id %d, want 1 system 0", userID, entity, entityID)
	}
	if !strings.Contains(details, `"sales":4`) {
		t.Errorf("details %q missing \"sales\":4", details)
	}
	if !strings.Contains(details, backup) {
		t.Errorf("details %q missing backup name", details)
	}

	// Second run without force: refused, nothing changes.
	mustExec(t, db, `INSERT INTO sales (user_id, subtotal, total, payment_method) VALUES (1, 100, 100, 'cash')`)
	_, err := Apply(ctx, db, Options{})
	if !errors.Is(err, ErrAlreadyReset) || !errors.Is(err, ErrRefused) {
		t.Fatalf("second Apply err = %v, want ErrAlreadyReset (refused)", err)
	}
	if ExitCode(err) != 1 {
		t.Errorf("ExitCode = %d, want 1", ExitCode(err))
	}
	if n := count(t, db, `SELECT COUNT(*) FROM sales`); n != 1 {
		t.Errorf("sales = %d after refused second run, want 1", n)
	}

	// With force: runs and writes a second data_reset row.
	if _, err := Apply(ctx, db, Options{Force: true}); err != nil {
		t.Fatalf("forced Apply: %v", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM sales`); n != 0 {
		t.Errorf("sales = %d after forced run, want 0", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM audit_log WHERE action = 'data_reset'`); n != 2 {
		t.Errorf("data_reset rows = %d, want 2", n)
	}
}

// T-12a: a failing step rolls everything back.
func TestApply_T12_FailingStepRollsBack(t *testing.T) {
	db, _ := newFixtureF(t)
	mustExec(t, db, `DROP TABLE expenses`)
	before := allCounts(t, db)

	_, err := Apply(context.Background(), db, Options{})
	if err == nil {
		t.Fatal("Apply succeeded with expenses table dropped")
	}
	if ExitCode(err) != 4 {
		t.Errorf("ExitCode = %d, want 4 (err %v)", ExitCode(err), err)
	}
	if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
		t.Errorf("counts changed after rollback:\nbefore %v\nafter  %v", before, after)
	}
	if got := stockOf(t, db, 1); got != 106 {
		t.Errorf("P1 stock = %v after rollback, want 106", got)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM audit_log WHERE action = 'data_reset'`); n != 0 {
		t.Errorf("data_reset rows = %d after rollback, want 0", n)
	}
}

// T-12b: a foreign key violation that survives the deletes aborts the commit.
func TestApply_T12_ForeignKeyViolationRollsBack(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()

	// A product pointing to a missing category survives the reset (products are kept),
	// so foreign_key_check still reports it before commit.
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO products (id, name, category_id, type, sale_price, stock) VALUES (9, 'Huerfano', 99, 'product', 1, 3)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	before := allCounts(t, db)

	_, err = Apply(ctx, db, Options{})
	if err == nil {
		t.Fatal("Apply succeeded despite a foreign key violation")
	}
	if ExitCode(err) != 4 {
		t.Errorf("ExitCode = %d, want 4 (err %v)", ExitCode(err), err)
	}
	if after := allCounts(t, db); !reflect.DeepEqual(before, after) {
		t.Errorf("counts changed after rollback:\nbefore %v\nafter  %v", before, after)
	}
	if got := stockOf(t, db, 1); got != 106 {
		t.Errorf("P1 stock = %v after rollback, want 106", got)
	}
}

// T-13: a row inserted between the backup and the lock changes the fingerprint.
func TestApply_T13_FingerprintMismatchAborts(t *testing.T) {
	db, _ := newFixtureF(t)
	ctx := context.Background()

	fp, err := TableCounts(ctx, db)
	if err != nil {
		t.Fatalf("TableCounts: %v", err)
	}
	if fp["sales"] != 4 {
		t.Fatalf("fingerprint sales = %d, want 4", fp["sales"])
	}

	opts := Options{
		Fingerprint: fp,
		BeforeLock: func() {
			mustExec(t, db, `INSERT INTO sales (user_id, subtotal, total, payment_method) VALUES (1, 100, 100, 'cash')`)
		},
	}
	_, err = Apply(ctx, db, opts)
	if err == nil {
		t.Fatal("Apply succeeded although the database changed after the backup")
	}
	if ExitCode(err) != 4 {
		t.Errorf("ExitCode = %d, want 4 (err %v)", ExitCode(err), err)
	}

	want := map[string]int{}
	for k, v := range fp {
		want[k] = v
	}
	want["sales"] = 5
	if got := allCounts(t, db); !reflect.DeepEqual(got, want) {
		t.Errorf("rows deleted despite mismatch:\nwant %v\ngot  %v", want, got)
	}
	if got := stockOf(t, db, 1); got != 106 {
		t.Errorf("P1 stock = %v, want 106", got)
	}
}
