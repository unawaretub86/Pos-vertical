package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

// A credit sale (fiado) cannot be charged to a deleted or missing customer;
// a cash sale that merely names an inactive customer still goes through.
func TestSaleRepo_Create_CreditToInactiveCustomer(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '', '')`)
	metricsExec(t, db, `INSERT INTO customers (id, name, phone, address, active) VALUES (2, 'Beto', '', '', 0)`)
	repo := NewSaleRepository(db.DB)

	sale := func(customerID int64, method string) *model.Sale {
		return &model.Sale{
			UserID: 1, CustomerID: &customerID, Subtotal: 10000, Total: 10000,
			PaymentMethod: method, Status: "completed",
			Payments: []model.Payment{{Method: method, Amount: 10000}},
		}
	}

	if _, err := repo.Create(ctx, sale(1, "credit")); err != nil {
		t.Errorf("credit sale to active customer: %v", err)
	}
	if _, err := repo.Create(ctx, sale(2, "credit")); !errors.Is(err, ErrCustomerInactive) {
		t.Errorf("credit sale to inactive customer: err = %v, want ErrCustomerInactive", err)
	}
	if _, err := repo.Create(ctx, sale(999, "credit")); !errors.Is(err, ErrCustomerInactive) {
		t.Errorf("credit sale to missing customer: err = %v, want ErrCustomerInactive", err)
	}
	if _, err := repo.Create(ctx, sale(2, "cash")); err != nil {
		t.Errorf("cash sale naming an inactive customer: %v", err)
	}
}
