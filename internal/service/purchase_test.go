package service

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

func newPurchaseTestEnv(t *testing.T) *PurchaseService {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.DB.ExecContext(context.Background(),
		`INSERT INTO suppliers (id, name) VALUES (1, 'Lácteos El Establo')`); err != nil {
		t.Fatalf("seed supplier: %v", err)
	}
	if _, err := db.DB.ExecContext(context.Background(),
		`INSERT INTO products (id, name, sale_price, cost, stock) VALUES (1, 'Leche', 3000, 7500, 5)`); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	if _, err := db.DB.ExecContext(context.Background(),
		`INSERT INTO users (id, username, password_hash, role) VALUES (7, 'u7', 'h', 'admin')`); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	purchaseRepo := repository.NewPurchaseRepository(db.DB)
	supplierRepo := repository.NewSupplierRepository(db.DB)
	return NewPurchaseService(purchaseRepo, supplierRepo)
}

// TestPurchaseService_Create_RejectsZeroQuantity is a DoD-pinned case:
// quantity=0 must be rejected with a 400-equivalent service error, not
// silently accepted or turned into a panic/500.
func TestPurchaseService_Create_RejectsZeroQuantity(t *testing.T) {
	svc := newPurchaseTestEnv(t)
	_, err := svc.Create(context.Background(), 1, &model.Purchase{
		SupplierID: 1,
		Items:      []model.PurchaseItem{{ProductID: 1, Quantity: 0, UnitCost: 8000}},
	})
	if err != ErrPurchaseInvalidQty {
		t.Errorf("expected ErrPurchaseInvalidQty, got %v", err)
	}
}

// TestPurchaseService_Create_RejectsNegativeUnitCost is the DoD-pinned
// counterpart for a negative unit_cost.
func TestPurchaseService_Create_RejectsNegativeUnitCost(t *testing.T) {
	svc := newPurchaseTestEnv(t)
	_, err := svc.Create(context.Background(), 1, &model.Purchase{
		SupplierID: 1,
		Items:      []model.PurchaseItem{{ProductID: 1, Quantity: 10, UnitCost: -100}},
	})
	if err != ErrPurchaseInvalidCost {
		t.Errorf("expected ErrPurchaseInvalidCost, got %v", err)
	}
}

func TestPurchaseService_Create_RejectsEmptyItems(t *testing.T) {
	svc := newPurchaseTestEnv(t)
	_, err := svc.Create(context.Background(), 1, &model.Purchase{SupplierID: 1})
	if err != ErrPurchaseNoItems {
		t.Errorf("expected ErrPurchaseNoItems, got %v", err)
	}
}

func TestPurchaseService_Create_RejectsUnknownSupplier(t *testing.T) {
	svc := newPurchaseTestEnv(t)
	_, err := svc.Create(context.Background(), 1, &model.Purchase{
		SupplierID: 999,
		Items:      []model.PurchaseItem{{ProductID: 1, Quantity: 1, UnitCost: 100}},
	})
	if err != ErrPurchaseSupplierGone {
		t.Errorf("expected ErrPurchaseSupplierGone, got %v", err)
	}
}

// TestPurchaseService_Create_Success confirms a valid purchase recalculates
// subtotal/total server-side and persists.
func TestPurchaseService_Create_Success(t *testing.T) {
	svc := newPurchaseTestEnv(t)
	created, err := svc.Create(context.Background(), 7, &model.Purchase{
		SupplierID: 1,
		Reference:  "FAC-001",
		Items:      []model.PurchaseItem{{ProductID: 1, Quantity: 10, UnitCost: 8000}},
	})
	if err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	if created.Total != 80000 {
		t.Errorf("expected total 80000, got %.0f", created.Total)
	}
	if created.UserID != 7 {
		t.Errorf("expected user_id 7, got %d", created.UserID)
	}
}
