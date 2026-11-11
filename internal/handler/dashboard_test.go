package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/repository"
)

// utcStamp renders an instant the way SQLite CURRENT_TIMESTAMP stores it:
// UTC text YYYY-MM-DD HH:MM:SS (US-098). Fixtures must never bind a zoned
// time.Time into created_at, because the driver would store the local offset.
func utcStamp(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05") }

// utcArgs converts every time.Time argument to its UTC text form.
func utcArgs(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		if tm, ok := a.(time.Time); ok {
			out[i] = utcStamp(tm)
		} else {
			out[i] = a
		}
	}
	return out
}

func setupTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seedProfitData(t *testing.T, db *database.DB) *Handler {
	t.Helper()
	ctx := context.Background()
	rawDB := db.DB

	now := time.Now()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	saleTime := todayStart.Add(time.Hour)

	exec := func(q string, args ...any) {
		_, err := rawDB.ExecContext(ctx, q, args...)
		if err != nil {
			t.Fatalf("seed exec: %v\nSQL: %s", err, q)
		}
	}

	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1)`)
	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Prod B', 'B', 6000, 0, 'unit', 1)`)
	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (3, 'Prod C', 'C', 12000, 8000, 'unit', 1)`)

	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, utcStamp(saleTime))

	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 10000, 10000)`)
	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 1, 6000, 6000)`)
	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 3, 1, 12000, 12000)`)

	return New(db)
}

func decodeTodayResponse(t *testing.T, w *httptest.ResponseRecorder) todayResponse {
	t.Helper()
	var envelope struct {
		Data todayResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return envelope.Data
}

func TestGetDashboardToday_GrossProfit(t *testing.T) {
	db := setupTestDB(t)
	h := seedProfitData(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/today", nil)
	w := httptest.NewRecorder()
	h.getDashboardToday(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}

	resp := decodeTodayResponse(t, w)

	// Profit: (10000-5000)*1 + (12000-8000)*1 = 5000 + 4000 = 9000
	if resp.GrossProfit != 9000 {
		t.Errorf("GrossProfit = %.0f, want 9000", resp.GrossProfit)
	}

	// Prod B has cost=0 → 1 unconfigured
	if resp.UnconfiguredCostCount != 1 {
		t.Errorf("UnconfiguredCostCount = %d, want 1", resp.UnconfiguredCostCount)
	}

	// TopByMargin: Prod A 50%, Prod C 33.3% — both cost>0
	if len(resp.TopByMargin) < 2 {
		t.Fatalf("TopByMargin len = %d, want >= 2", len(resp.TopByMargin))
	}
	if resp.TopByMargin[0].Name != "Prod A" {
		t.Errorf("top margin product = %q, want Prod A", resp.TopByMargin[0].Name)
	}
}

func TestGetDashboardToday_NoSales(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/today", nil)
	w := httptest.NewRecorder()
	h.getDashboardToday(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}

	resp := decodeTodayResponse(t, w)

	if resp.GrossProfit != 0 {
		t.Errorf("expected 0 profit with no sales, got %.0f", resp.GrossProfit)
	}
	if resp.UnconfiguredCostCount != 0 {
		t.Errorf("expected 0 unconfigured, got %d", resp.UnconfiguredCostCount)
	}
	if len(resp.TopByMargin) != 0 {
		t.Errorf("expected empty top_by_margin with no sales")
	}
}

// ─── US-098: day windows and stored UTC timestamps agree ────────────────────

// dayWindowBogota is the fixed UTC-5 zone of the injected clock.
var dayWindowBogota = time.FixedZone("COT", -5*3600)

// seedDayWindowSales seeds the S0..S5 completed cash sales of contract §3
// (literal UTC text, as CURRENT_TIMESTAMP stores them):
//
//	S0 Nov 8 23:59:59 Bogota  2026-11-09 04:59:59  $5.000
//	S5 Nov 9 00:00:00 Bogota  2026-11-09 05:00:00  $7.000
//	S1 Nov 9 18:59:00 Bogota  2026-11-09 23:59:00  $10.000
//	S2 Nov 9 20:18:41 Bogota  2026-11-10 01:18:41  $50.000
//	S3 Nov 9 23:59:59 Bogota  2026-11-10 04:59:59  $20.000
//	S4 Nov 10 00:00:00 Bogota 2026-11-10 05:00:00  $3.000
//
// Day Nov 9 = 87.000 / 4 sales; Nov 10 = 3.000 / 1; Nov 8 = 5.000 / 1.
func seedDayWindowSales(t *testing.T, db *database.DB) {
	t.Helper()
	rows := []struct {
		id    int
		total int
		at    string
	}{
		{10, 5000, "2026-11-09 04:59:59"},
		{15, 7000, "2026-11-09 05:00:00"},
		{11, 10000, "2026-11-09 23:59:00"},
		{12, 50000, "2026-11-10 01:18:41"},
		{13, 20000, "2026-11-10 04:59:59"},
		{14, 3000, "2026-11-10 05:00:00"},
	}
	for _, r := range rows {
		execSeed(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (?, 1, ?, ?, 'completed', 'cash', ?)`,
			r.id, r.total, r.total, r.at)
		execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'cash', ?)`, r.id, r.total)
	}
}

// T-1 (AC-1): at 2026-11-09 20:30 Bogota the dashboard "today" includes the
// 20:18:41 sale (stored 2026-11-10 01:18:41 UTC) and excludes S0 and S4.
func TestGetDashboardToday_T1_EveningSaleIsToday(t *testing.T) {
	db := setupTestDB(t)
	seedDayWindowSales(t, db)
	h := New(db)
	h.now = func() time.Time { return time.Date(2026, 11, 9, 20, 30, 0, 0, dayWindowBogota) }

	w := httptest.NewRecorder()
	h.getDashboardToday(w, httptest.NewRequest(http.MethodGet, "/api/dashboard/today", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	resp := decodeTodayResponse(t, w)
	if resp.SoldToday != 87000 {
		t.Errorf("sold_today = %.0f, want 87000 (S5 7000 + S1 10000 + S2 50000 + S3 20000; S0 and S4 excluded)", resp.SoldToday)
	}
}

// T-2 (AC-2): /dashboard at the same instant: today 87000 / 4, yesterday 5000 / 1.
func TestGetDashboard_T2_TodayAndYesterdayAtEvening(t *testing.T) {
	db := setupTestDB(t)
	seedDayWindowSales(t, db)
	h := New(db)
	h.now = func() time.Time { return time.Date(2026, 11, 9, 20, 30, 0, 0, dayWindowBogota) }

	w := httptest.NewRecorder()
	h.getDashboard(w, httptest.NewRequest(http.MethodGet, "/api/dashboard", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var env struct {
		Data dashboardResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Today.TotalRevenue != 87000 || env.Data.Today.TotalCount != 4 {
		t.Errorf("today = %.0f / %d, want 87000 / 4", env.Data.Today.TotalRevenue, env.Data.Today.TotalCount)
	}
	if env.Data.Yesterday.TotalRevenue != 5000 || env.Data.Yesterday.TotalCount != 1 {
		t.Errorf("yesterday = %.0f / %d, want 5000 / 1 (S0 only)", env.Data.Yesterday.TotalRevenue, env.Data.Yesterday.TotalCount)
	}
}

// T-3 (AC-3, AC-12): the Nov 9 Bogota window sent as RFC3339 with -05:00, and
// the same instants in the browser form (UTC, .000Z), both give 87000 / 4:
// S5 exactly at "from" is in, S4 exactly at "to" is out.
func TestGetSalesReport_T3_WindowBoundsAnyOffset(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
	}{
		{"offset -05:00", "2026-11-09T00:00:00-05:00", "2026-11-10T00:00:00-05:00"},
		{"browser UTC .000Z", "2026-11-09T05:00:00.000Z", "2026-11-10T05:00:00.000Z"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			seedDayWindowSales(t, db)
			h := New(db)

			w := httptest.NewRecorder()
			h.getSalesReport(w, httptest.NewRequest(http.MethodGet, "/api/reports/sales?from="+url.QueryEscape(tc.from)+"&to="+url.QueryEscape(tc.to), nil))
			if w.Code != http.StatusOK {
				t.Fatalf("status %d, body %s", w.Code, w.Body.String())
			}
			var env struct {
				Data repository.SalesSummary `json:"data"`
			}
			if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if env.Data.TotalRevenue != 87000 || env.Data.TotalCount != 4 {
				t.Errorf("total = %.0f / %d, want 87000 / 4 (S5 at from in, S4 at to out)", env.Data.TotalRevenue, env.Data.TotalCount)
			}
		})
	}
}

// T-9 (AC-8): a sale created through POST /sales keeps the stored text format
// YYYY-MM-DD HH:MM:SS (no mixed formats), and no migration was added (22).
func TestPostSales_T9_StoredTimestampFormatAndMigrationCount(t *testing.T) {
	e := newShiftEnv(t)
	e.openRegister(t, 100000)
	e.postSale(t, 1, 30000, `[{"method":"cash","amount":30000}]`, 0)

	var typ, stored string
	if err := e.db.DB.QueryRowContext(context.Background(),
		`SELECT typeof(created_at), CAST(created_at AS TEXT) FROM sales WHERE id = 1`).Scan(&typ, &stored); err != nil {
		t.Fatalf("read stored created_at: %v", err)
	}
	if typ != "text" {
		t.Errorf("typeof(created_at) = %q, want text", typ)
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(stored) {
		t.Errorf("stored created_at = %q, want YYYY-MM-DD HH:MM:SS (UTC text, no zone suffix)", stored)
	}

	var migrations int
	if err := e.db.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 23 {
		t.Errorf("schema_migrations rows = %d, want 23 (US-089 adds migration 23)", migrations)
	}
}
