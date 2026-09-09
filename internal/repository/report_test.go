package repository

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

// TestReportRepo_SalesSummary_SplitsMixedPayments verifies a mixed sale
// contributes each portion to its own payment method in the report instead
// of lumping the whole total under 'mixed' (US-046 AC).
func TestReportRepo_SalesSummary_SplitsMixedPayments(t *testing.T) {
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

	// Mixed sale: $60.000 total, $30.000 cash + $20.000 transfer + $10.000 credit.
	exec(`INSERT INTO customers (id, name) VALUES (1, 'Cliente')`)
	exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method) VALUES (1, 1, 1, 60000, 60000, 'completed', 'mixed')`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 30000)`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'transfer', 20000)`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 10000)`)

	// Plain cash sale: $20.000.
	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method) VALUES (2, 1, 20000, 20000, 'completed', 'cash')`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 20000)`)

	repo := NewReportRepository(raw)
	summary, err := repo.SalesSummary(ctx, SalesSummaryParams{GroupBy: "day"})
	if err != nil {
		t.Fatalf("sales summary: %v", err)
	}

	totals := map[string]float64{}
	for _, pm := range summary.ByPaymentMethod {
		totals[pm.Method] = pm.Total
	}

	if totals["cash"] != 50000 {
		t.Errorf("expected cash total 50000, got %.0f", totals["cash"])
	}
	if totals["transfer"] != 20000 {
		t.Errorf("expected transfer total 20000, got %.0f", totals["transfer"])
	}
	if totals["credit"] != 10000 {
		t.Errorf("expected credit total 10000, got %.0f", totals["credit"])
	}
	if _, hasMixed := totals["mixed"]; hasMixed {
		t.Errorf("expected no 'mixed' bucket in by_payment_method, got one")
	}
	if summary.TotalRevenue != 80000 {
		t.Errorf("expected total revenue 80000, got %.0f", summary.TotalRevenue)
	}
}
