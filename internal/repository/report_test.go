package repository

import (
	"context"
	"testing"
	"time"

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

// TestReportRepo_T22_BuildDateWhereIsHalfOpen covers US-061 AC-20 (R-06):
// buildDateWhere must build [from, to) — a sale created exactly at the
// midnight boundary T belongs to "mañana" ([T, T+1d)), never to "hoy"
// ([T-1d, T)), and never to both or neither.
func TestReportRepo_T22_BuildDateWhereIsHalfOpen(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	raw := db.DB

	now := time.Now()
	loc := now.Location()
	boundary := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	hoyFrom, hoyTo := boundary.Add(-24*time.Hour), boundary
	mananaFrom, mananaTo := boundary, boundary.Add(24*time.Hour)

	if _, err := raw.ExecContext(ctx,
		`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 1000, 500, 'unit', 1)`,
	); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	if _, err := raw.ExecContext(ctx,
		`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1000, 1000, 'completed', 'cash', ?)`,
		boundary,
	); err != nil {
		t.Fatalf("seed sale at boundary: %v", err)
	}
	if _, err := raw.ExecContext(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 1000, 1000)`,
	); err != nil {
		t.Fatalf("seed sale item: %v", err)
	}

	repo := NewReportRepository(raw)

	hoySummary, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &hoyFrom, To: &hoyTo, GroupBy: "day"})
	if err != nil {
		t.Fatalf("SalesSummary(hoy): %v", err)
	}
	if hoySummary.TotalCount != 0 {
		t.Errorf("SalesSummary(hoy).TotalCount = %d, want 0 (boundary sale belongs to mañana)", hoySummary.TotalCount)
	}

	mananaSummary, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &mananaFrom, To: &mananaTo, GroupBy: "day"})
	if err != nil {
		t.Fatalf("SalesSummary(mañana): %v", err)
	}
	if mananaSummary.TotalCount != 1 {
		t.Errorf("SalesSummary(mañana).TotalCount = %d, want 1", mananaSummary.TotalCount)
	}

	hoyProducts, err := repo.TopProducts(ctx, &hoyFrom, &hoyTo, 10)
	if err != nil {
		t.Fatalf("TopProducts(hoy): %v", err)
	}
	if len(hoyProducts) != 0 {
		t.Errorf("TopProducts(hoy) len = %d, want 0 (boundary sale belongs to mañana): %+v", len(hoyProducts), hoyProducts)
	}

	mananaProducts, err := repo.TopProducts(ctx, &mananaFrom, &mananaTo, 10)
	if err != nil {
		t.Fatalf("TopProducts(mañana): %v", err)
	}
	if len(mananaProducts) != 1 || mananaProducts[0].Quantity != 1 {
		t.Errorf("TopProducts(mañana) = %+v, want exactly 1 row of qty 1", mananaProducts)
	}
}
