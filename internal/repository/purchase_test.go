package repository

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/model"
)

// TestPurchaseRepo_Create_UpdatesStockAndCost is the DoD-pinned case: a
// purchase of 10 units at $8.000 for a product that had stock 5 and cost
// $7.500 must leave stock 15 and cost exactly $8.000 (last-cost, not a
// weighted average — see US-052 scope note).
func TestPurchaseRepo_Create_UpdatesStockAndCost(t *testing.T) {
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
	exec(`INSERT INTO products (id, name, sale_price, cost, stock) VALUES (1, 'Leche', 3000, 7500, 5)`)

	repo := NewPurchaseRepository(raw)
	created, err := repo.Create(ctx, &model.Purchase{
		SupplierID: 1,
		UserID:     1,
		Reference:  "FAC-001",
		Total:      80000,
		Items: []model.PurchaseItem{
			{ProductID: 1, Quantity: 10, UnitCost: 8000, Subtotal: 80000},
		},
	})
	if err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	if created.Total != 80000 {
		t.Errorf("expected total 80000, got %.0f", created.Total)
	}

	var stock, cost float64
	if err := raw.QueryRowContext(ctx, `SELECT stock, cost FROM products WHERE id = 1`).Scan(&stock, &cost); err != nil {
		t.Fatalf("query product: %v", err)
	}
	if stock != 15 {
		t.Errorf("expected stock 15, got %.0f", stock)
	}
	if cost != 8000 {
		t.Errorf("expected cost 8000 (last-cost, not average), got %.0f", cost)
	}

	var entryCount int
	if err := raw.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stock_entries WHERE product_id = 1 AND note LIKE '%Compra #%'`,
	).Scan(&entryCount); err != nil {
		t.Fatalf("query stock_entries: %v", err)
	}
	if entryCount != 1 {
		t.Errorf("expected 1 stock_entries row referencing the purchase, got %d", entryCount)
	}
}

// TestPurchaseRepo_Create_UnknownSupplier confirms a purchase against a
// non-existent supplier is rejected instead of silently inserted with a
// dangling foreign key.
func TestPurchaseRepo_Create_UnknownSupplier(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	raw := db.DB

	repo := NewPurchaseRepository(raw)
	_, err = repo.Create(ctx, &model.Purchase{
		SupplierID: 999,
		UserID:     1,
		Items:      []model.PurchaseItem{{ProductID: 1, Quantity: 1, UnitCost: 100, Subtotal: 100}},
	})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for unknown supplier, got %v", err)
	}
}
