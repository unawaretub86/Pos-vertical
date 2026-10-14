package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// An installation older than v12 that already holds sales must still upgrade
// now that foreign keys are enforced: migration 12 rebuilds the sales table.
func TestMigrate_UpgradeFromV11KeepsSalesAndItems(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", p) // no pragmas: foreign keys off, like an old install
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	old := &DB{DB: raw, path: p}
	if _, err := raw.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at DATETIME DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	for _, m := range dbMigrations {
		if m.version > 11 {
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
	mustExec(`INSERT INTO registers (id, user_id, opening_amount) VALUES (1, 1, 0)`)
	mustExec(`INSERT INTO products (id, name, sale_price, cost, stock) VALUES (1, 'p', 1000, 500, 10)`)
	mustExec(`INSERT INTO sales (id, user_id, register_id, subtotal, total, payment_method) VALUES (1, 1, 1, 1000, 1000, 'cash')`)
	mustExec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 1000, 1000)`)
	raw.Close() // flush before reopening through New

	db, err := New(p)
	if err != nil {
		t.Fatalf("upgrade with data and FK on failed: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sale_items`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("sale_items after upgrade = %d err=%v, want 1", n, err)
	}
}
