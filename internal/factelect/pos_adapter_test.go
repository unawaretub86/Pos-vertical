package factelect

// US-102 (R-133) contract §5, T-17 (AC-11): the DIAN adapter computes
// TaxAmount per line with the §3 rule (round half away from zero), so
// TotalIVA = sum of the line TaxAmount = sale.tax. The sale is seeded with
// literal INSERTs holding the whole-peso values the service now stores.

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/repository"
)

func adapterFixture(t *testing.T) *PosSaleAdapter {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, q := range []string{
		`INSERT INTO business_config (id, name, nit, address) VALUES (1, 'Tienda Uno', '900123456-7', 'Calle 1')`,
		`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (2, 'Cafe', 'P2', 'product', 1001, 0, 'unit', 20, 0.19, 1)`,
		`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (3, 'Taza', 'P3', 'product', 1050, 0, 'unit', 20, 0.19, 1)`,
		// Sale 1: 3 x Cafe $1.001 at 19%: tax 190 x 3 = 570, total 3573.
		`INSERT INTO sales (id, user_id, subtotal, tax, total, status, payment_method) VALUES (1, 1, 3003, 570, 3573, 'completed', 'cash')`,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (1, 2, 1, 1001, 1001, 0.19)`,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (1, 2, 1, 1001, 1001, 0.19)`,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (1, 2, 1, 1001, 1001, 0.19)`,
		`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 3573)`,
		// Sale 2: 1 x Taza $1.050 at 19%: 199.5 rounds half up to 200, total 1250.
		`INSERT INTO sales (id, user_id, subtotal, tax, total, status, payment_method) VALUES (2, 1, 1050, 200, 1250, 'completed', 'cash')`,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, tax_rate) VALUES (2, 3, 1, 1050, 1050, 0.19)`,
		`INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 1250)`,
	} {
		if _, err := db.DB.Exec(q); err != nil {
			t.Fatalf("seed: %v\nSQL: %s", err, q)
		}
	}
	return NewPosSaleAdapter(
		repository.NewSaleRepository(db.DB),
		repository.NewProductRepository(db.DB),
		repository.NewSetupRepository(db.DB),
		NewFiscalConfigRepository(db.DB),
	)
}

// T-17: 3 x P2: items TaxAmount 190, 190, 190; TotalIVA 570; Total 3573.
func TestPosAdapter_T17_TaxAmountRoundedPerLine(t *testing.T) {
	a := adapterFixture(t)
	req, err := a.ToBillRequest(context.Background(), 1, "a@b.co")
	if err != nil {
		t.Fatalf("ToBillRequest: %v", err)
	}
	if len(req.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(req.Items))
	}
	var sum float64
	for i, it := range req.Items {
		if it.TaxAmount != 190 {
			t.Errorf("item %d TaxAmount = %v, want 190", i, it.TaxAmount)
		}
		sum += it.TaxAmount
	}
	if req.Totals.TotalIVA != 570 || sum != 570 {
		t.Errorf("TotalIVA = %v, sum of line TaxAmount = %v, want 570 both", req.Totals.TotalIVA, sum)
	}
	if req.Totals.Total != 3573 {
		t.Errorf("Total = %v, want 3573", req.Totals.Total)
	}
	if req.Totals.Subtotal != 3003 {
		t.Errorf("Subtotal = %v, want 3003", req.Totals.Subtotal)
	}
}

// T-17: a half rounds up on the line, not down: $1.050 at 19% -> 200.
func TestPosAdapter_T17_HalfRoundsUpOnTheLine(t *testing.T) {
	a := adapterFixture(t)
	req, err := a.ToBillRequest(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("ToBillRequest: %v", err)
	}
	if len(req.Items) != 1 || req.Items[0].TaxAmount != 200 {
		t.Fatalf("items = %+v, want one line with TaxAmount 200", req.Items)
	}
	if req.Totals.TotalIVA != 200 || req.Totals.Total != 1250 {
		t.Errorf("TotalIVA/Total = %v/%v, want 200/1250", req.Totals.TotalIVA, req.Totals.Total)
	}
}
