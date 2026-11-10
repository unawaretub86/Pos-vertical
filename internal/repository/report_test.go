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
		utcStamp(boundary),
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

// ─── US-098: day windows and stored UTC timestamps agree ────────────────────

// T-4 (AC-4): two contiguous Bogota day windows split the five sales S5, S1,
// S2, S3, S4 with none counted twice and none dropped; S0 is in neither.
func TestReportRepo_T4_ContiguousDayWindows(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedDayWindowSales(t, db)
	repo := NewReportRepository(db.DB)

	nov9From, nov10From, nov11From := bogotaMidnight(2026, 11, 9), bogotaMidnight(2026, 11, 10), bogotaMidnight(2026, 11, 11)

	nov9, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &nov9From, To: &nov10From, GroupBy: "day"})
	if err != nil {
		t.Fatalf("SalesSummary(nov9): %v", err)
	}
	nov10, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &nov10From, To: &nov11From, GroupBy: "day"})
	if err != nil {
		t.Fatalf("SalesSummary(nov10): %v", err)
	}
	if nov9.TotalRevenue != 87000 || nov9.TotalCount != 4 {
		t.Errorf("Nov 9 = %.0f / %d, want 87000 / 4 (S5 7000 + S1 10000 + S2 50000 + S3 20000)", nov9.TotalRevenue, nov9.TotalCount)
	}
	if nov10.TotalRevenue != 3000 || nov10.TotalCount != 1 {
		t.Errorf("Nov 10 = %.0f / %d, want 3000 / 1 (S4)", nov10.TotalRevenue, nov10.TotalCount)
	}
	if sum, cnt := nov9.TotalRevenue+nov10.TotalRevenue, nov9.TotalCount+nov10.TotalCount; sum != 90000 || cnt != 5 {
		t.Errorf("Nov 9 + Nov 10 = %.0f / %d, want 90000 / 5 (S0 in neither)", sum, cnt)
	}

	// Same windows through TopProducts' filter (buildDateWhere): one product, sold in each sale.
	metricsExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 1000, 500, 'unit', 1)`)
	for _, id := range []int{10, 15, 11, 12, 13, 14} {
		metricsExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (?, 1, 1, 1000, 1000)`, id)
	}
	p9, err := repo.TopProducts(ctx, &nov9From, &nov10From, 10)
	if err != nil {
		t.Fatalf("TopProducts(nov9): %v", err)
	}
	if len(p9) != 1 || p9[0].Quantity != 4 {
		t.Errorf("TopProducts(nov9) = %+v, want one row of qty 4", p9)
	}
	p10, err := repo.TopProducts(ctx, &nov10From, &nov11From, 10)
	if err != nil {
		t.Fatalf("TopProducts(nov10): %v", err)
	}
	if len(p10) != 1 || p10[0].Quantity != 1 {
		t.Errorf("TopProducts(nov10) = %+v, want one row of qty 1", p10)
	}
}

// T-8 (AC-7): grouped by day, the Nov 9 Bogota window is one period
// 2026-11-09 with 87000 / 4; S2 and S3 do not spill into 2026-11-10.
//
// The period label is produced by SQLite strftime(..., 'localtime'), which
// follows the OS zone. The contract values (one period) hold when the host
// zone is UTC-5; on another host the expected periods are derived from
// time.Local, so the test stays meaningful (and CI-safe) in any zone.
func TestReportRepo_T8_GroupByDayLocalPeriod(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	seedDayWindowSales(t, db)
	repo := NewReportRepository(db.DB)

	from, to := bogotaMidnight(2026, 11, 9), bogotaMidnight(2026, 11, 10)
	got, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &from, To: &to, GroupBy: "day"})
	if err != nil {
		t.Fatalf("SalesSummary(nov9, day): %v", err)
	}

	type bucket struct {
		total float64
		count int
	}
	want := map[string]bucket{}
	for _, s := range []struct {
		at    time.Time
		total float64
	}{
		{time.Date(2026, 11, 9, 5, 0, 0, 0, time.UTC), 7000},   // S5
		{time.Date(2026, 11, 9, 23, 59, 0, 0, time.UTC), 10000}, // S1
		{time.Date(2026, 11, 10, 1, 18, 41, 0, time.UTC), 50000}, // S2
		{time.Date(2026, 11, 10, 4, 59, 59, 0, time.UTC), 20000}, // S3
	} {
		label := s.at.In(time.Local).Format("2006-01-02")
		b := want[label]
		b.total += s.total
		b.count++
		want[label] = b
	}
	if _, off := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local).Zone(); off == -5*3600 {
		if len(got.Grouped) != 1 || got.Grouped[0].Period != "2026-11-09" || got.Grouped[0].Total != 87000 || got.Grouped[0].Count != 4 {
			t.Errorf("Grouped = %+v, want exactly [2026-11-09 87000 / 4]", got.Grouped)
		}
	}
	if len(got.Grouped) != len(want) {
		t.Fatalf("Grouped = %+v, want %d periods %+v", got.Grouped, len(want), want)
	}
	for _, p := range got.Grouped {
		w, ok := want[p.Period]
		if !ok || p.Total != w.total || p.Count != w.count {
			t.Errorf("period %s = %.0f / %d, want %+v (all: %+v)", p.Period, p.Total, p.Count, w, want)
		}
	}
	if got.TotalRevenue != 87000 || got.TotalCount != 4 {
		t.Errorf("window total = %.0f / %d, want 87000 / 4", got.TotalRevenue, got.TotalCount)
	}
}

// T-13 (AC-12): a row stored exactly at a bound instant belongs to exactly one
// of two adjacent windows, whether the bound is a local-zone time.Time or its
// .UTC() form (the +00:00 suffix used to sort after the zone-less stored text).
func TestReportRepo_T13_BoundInstantBelongsToOneWindow(t *testing.T) {
	db := openMetricsDB(t)
	ctx := context.Background()
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1000, 1000, 'completed', 'cash', '2026-11-09 05:00:00')`) // Nov 9 00:00 Bogota
	metricsExec(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2000, 2000, 'completed', 'cash', '2026-11-10 05:00:00')`) // Nov 10 00:00 Bogota
	repo := NewReportRepository(db.DB)

	b9, b10, b11 := bogotaMidnight(2026, 11, 9), bogotaMidnight(2026, 11, 10), bogotaMidnight(2026, 11, 11)
	cases := []struct {
		name         string
		b9, b10, b11 time.Time
	}{
		{"local bounds", b9, b10, b11},
		{"UTC bounds", b9.UTC(), b10.UTC(), b11.UTC()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// [Nov 8 00:00, Nov 9 00:00): neither row.
			b8 := c.b9.Add(-24 * time.Hour)
			before, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &b8, To: &c.b9, GroupBy: "day"})
			if err != nil {
				t.Fatalf("SalesSummary(before): %v", err)
			}
			first, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &c.b9, To: &c.b10, GroupBy: "day"})
			if err != nil {
				t.Fatalf("SalesSummary(first): %v", err)
			}
			second, err := repo.SalesSummary(ctx, SalesSummaryParams{From: &c.b10, To: &c.b11, GroupBy: "day"})
			if err != nil {
				t.Fatalf("SalesSummary(second): %v", err)
			}
			if before.TotalCount != 0 {
				t.Errorf("window ending at Nov 9 00:00 = %d rows, want 0 (row 1 sits exactly at its end)", before.TotalCount)
			}
			if first.TotalRevenue != 1000 || first.TotalCount != 1 {
				t.Errorf("[Nov 9, Nov 10) = %.0f / %d, want 1000 / 1 (row at from in, row at to out)", first.TotalRevenue, first.TotalCount)
			}
			if second.TotalRevenue != 2000 || second.TotalCount != 1 {
				t.Errorf("[Nov 10, Nov 11) = %.0f / %d, want 2000 / 1 (row at from in)", second.TotalRevenue, second.TotalCount)
			}
		})
	}
}
