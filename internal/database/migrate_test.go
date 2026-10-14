package database

// Contrato US-055 §4, T-1 y T-2.

import (
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
