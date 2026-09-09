package repository

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

// TestSupplierRepo_Delete_BlockedWithPurchases mirrors the category-in-use
// rule: a supplier with purchase history cannot be deleted (US-052 AC).
func TestSupplierRepo_Delete_BlockedWithPurchases(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	raw := db.DB

	exec := func(q string, args ...any) {
		if _, err := raw.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("seed exec: %v\nSQL: %s", err, q)
		}
	}
	exec(`INSERT INTO suppliers (id, name) VALUES (1, 'Lácteos El Establo')`)
	exec(`INSERT INTO purchases (id, supplier_id, user_id, total) VALUES (1, 1, 1, 10000)`)

	repo := NewSupplierRepository(raw)
	if err := repo.Delete(ctx, 1); err == nil {
		t.Fatal("expected delete to be rejected for a supplier with purchases, got nil error")
	}

	var count int
	if err := raw.QueryRowContext(ctx, `SELECT COUNT(*) FROM suppliers WHERE id = 1`).Scan(&count); err != nil {
		t.Fatalf("query suppliers: %v", err)
	}
	if count != 1 {
		t.Error("supplier was deleted despite having purchases")
	}
}

// TestSupplierRepo_Delete_NoPurchases confirms a clean supplier deletes fine.
func TestSupplierRepo_Delete_NoPurchases(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	raw := db.DB

	if _, err := raw.ExecContext(ctx, `INSERT INTO suppliers (id, name) VALUES (1, 'Sin compras')`); err != nil {
		t.Fatalf("seed supplier: %v", err)
	}

	repo := NewSupplierRepository(raw)
	if err := repo.Delete(ctx, 1); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
}
