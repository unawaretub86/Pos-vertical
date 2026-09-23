package handler

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

// US-059 — equivalence tests. T-20, T-21 and T-23 pin the numbers of the
// pre-refactor code and must pass both before and after extracting
// MetricsRepository. T-24, T-28 and T-29 pin approved behavior changes (Q6, Q7).
// US-063 — T-22 pins the corrected aging balance (D-4 fixed: each customer
// payment is subtracted once); T-36 pins the single-balance invariant across
// /dashboard/today, /reports/portfolio and /customers.

func eqToday0() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func eqNear(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func callDashboardToday(h *Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/today", nil)
	w := httptest.NewRecorder()
	h.getDashboardToday(w, req)
	return w
}

func callPortfolio(h *Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/reports/portfolio", nil)
	w := httptest.NewRecorder()
	h.getPortfolio(w, req)
	return w
}

// seedGoldenT20 is the golden seed of T-20 (reused by T-28).
func seedGoldenT20(t *testing.T, db *database.DB) {
	t.Helper()
	hoy0 := eqToday0()

	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (1, 'Arroz', 'ARZ', 3000, 2500, 'unit', 1, 50, 0)`)
	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (2, 'Aceite', 'ACE', 12000, 8000, 'unit', 1, 3, 10)`)
	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock, min_stock) VALUES (3, 'Gaseosa', 'GAS', 4000, 0, 'unit', 1, 1, 6)`)
	execSeed(t, db, `INSERT INTO registers (id, user_id, opening_amount) VALUES (1, 1, 0)`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '3001111111', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '3002222222', '')`)

	// S1 today 10:00 cash: 4 Arroz + 1 Aceite = 24.000.
	execSeed(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 24000, 24000, 'completed', 'cash', ?)`, hoy0.Add(10*time.Hour))
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 4, 3000, 12000)`)
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 1, 12000, 12000)`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 24000)`)

	// S2 today 11:00 mixed, Ana: 2 Aceite + 2 Gaseosa = 32.000 (cash 12.000 + credit 20.000).
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 32000, 32000, 'completed', 'mixed', ?)`, hoy0.Add(11*time.Hour))
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (2, 2, 2, 12000, 24000)`)
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (2, 3, 2, 4000, 8000)`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 12000)`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)

	// S3 today 12:00 voided: 1 Aceite = 12.000.
	execSeed(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 12000, 12000, 'voided', 'cash', ?)`, hoy0.Add(12*time.Hour))
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (3, 2, 1, 12000, 12000)`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'cash', 12000)`)

	// S4 yesterday 15:00 credit, Beto: 10 Arroz = 30.000.
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 2, 30000, 30000, 'completed', 'credit', ?)`, hoy0.Add(-9*time.Hour))
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (4, 1, 10, 3000, 30000)`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 30000)`)

	// Customer payments: Ana 5.000 today, Beto 10.000 yesterday.
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 5000, ?)`, hoy0.Add(13*time.Hour))
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 10000, ?)`, hoy0.Add(-8*time.Hour))

	// Register movements: cash_out 3.000 today, cash_out 1.000 yesterday, cash_in 50.000 today.
	execSeed(t, db, `INSERT INTO register_movements (register_id, type, amount, created_at) VALUES (1, 'cash_out', 3000, ?)`, hoy0.Add(14*time.Hour))
	execSeed(t, db, `INSERT INTO register_movements (register_id, type, amount, created_at) VALUES (1, 'cash_out', 1000, ?)`, hoy0.Add(-7*time.Hour))
	execSeed(t, db, `INSERT INTO register_movements (register_id, type, amount, created_at) VALUES (1, 'cash_in', 50000, ?)`, hoy0.Add(9*time.Hour))

	// Expenses: 3.000 today, 7.000 yesterday.
	execSeed(t, db, `INSERT INTO expenses (register_id, user_id, category, amount, created_at) VALUES (1, 1, 'otro', 3000, ?)`, hoy0.Add(15*time.Hour))
	execSeed(t, db, `INSERT INTO expenses (register_id, user_id, category, amount, created_at) VALUES (1, 1, 'otro', 7000, ?)`, hoy0.Add(-6*time.Hour))
}

// seedAgingDatetime seeds T-16 + T-17 with real app timestamps (datetime('now', ...)).
func seedAgingDatetime(t *testing.T, db *database.DB) {
	t.Helper()
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Carla', '', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (4, 'Dora', '', '')`)

	// Beto: 30.000 ten days ago, payment 10.000 → 20.000.
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 2, 30000, 30000, 'completed', 'credit', datetime('now','-10 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 30000)`)
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 10000, datetime('now','-1 days'))`)
	// Ana: 20.000 three days ago, payment 5.000 → 15.000.
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 1, 20000, 20000, 'completed', 'credit', datetime('now','-3 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 5000, datetime('now','-1 days'))`)
	// Carla: 10.000 five days ago, payment 10.000 → 0.
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 10000, 10000, 'completed', 'credit', datetime('now','-5 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 10000)`)
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 10000, datetime('now','-1 days'))`)
	// Dora: 2 × 25.000 (7 and 6 days ago), payment 10.000.
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 4, 25000, 25000, 'completed', 'credit', datetime('now','-7 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 25000)`)
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (5, 1, 4, 25000, 25000, 'completed', 'credit', datetime('now','-6 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (5, 'credit', 25000)`)
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (4, 10000, datetime('now','-1 days'))`)
}

// seedPortfolioT23 seeds T-12 + T-14 with datetime('now', ...) timestamps.
// Ana 150.000; Beto 2×50.000 − 20.000 = 80.000; Carla 30.000 − 30.000 = 0; Dora 80.000.
// Dora has a lower id than Beto so the tie is broken by name, not by id.
func seedPortfolioT23(t *testing.T, db *database.DB) {
	t.Helper()
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Ana', '3001111111', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (2, 'Dora', '3004444444', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (3, 'Beto', '3002222222', '')`)
	execSeed(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (4, 'Carla', '3003333333', '')`)

	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 150000, 150000, 'completed', 'credit', datetime('now','-4 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 150000)`)

	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 80000, 80000, 'completed', 'credit', datetime('now','-3 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 80000)`)

	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 50000, 50000, 'completed', 'credit', datetime('now','-5 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 50000)`)
	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 3, 50000, 50000, 'completed', 'credit', datetime('now','-2 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 50000)`)
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 20000, datetime('now','-1 days'))`)

	execSeed(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (5, 1, 4, 30000, 30000, 'completed', 'credit', datetime('now','-6 days'))`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (5, 'credit', 30000)`)
	execSeed(t, db, `INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (4, 30000, datetime('now','-1 days'))`)
}

// ─── T-20: golden /dashboard/today ───────────────────────────────────────────

func TestDashboardToday_T20_Golden(t *testing.T) {
	db := setupTestDB(t)
	seedGoldenT20(t, db)
	h := New(db)

	w := callDashboardToday(h)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body %s", w.Code, w.Body.String())
	}
	raw := w.Body.Bytes()
	var resp todayResponse
	if err := json.Unmarshal(raw, &struct {
		Data *todayResponse `json:"data"`
	}{&resp}); err != nil {
		t.Fatalf("decode: %v", err)
	}

	scalars := []struct {
		name      string
		got, want float64
	}{
		{"sold_today", resp.SoldToday, 56000},
		{"credit_new_today", resp.CreditNewToday, 20000},
		{"credit_collected_today", resp.CreditCollectedToday, 5000},
		{"cash_estimated", resp.CashEstimated, 33000},
		{"top_product_qty", resp.TopProductQty, 4},
		{"gross_profit", resp.GrossProfit, 14000},
		{"expenses_today", resp.ExpensesToday, 3000},
		{"net_profit", resp.NetProfit, 11000},
	}
	for _, s := range scalars {
		if s.got != s.want {
			t.Errorf("%s = %.2f, want %.0f", s.name, s.got, s.want)
		}
	}
	if resp.TopProductName != "Arroz" {
		t.Errorf("top_product_name = %q, want Arroz", resp.TopProductName)
	}
	if resp.UnconfiguredCostCount != 1 {
		t.Errorf("unconfigured_cost_count = %d, want 1", resp.UnconfiguredCostCount)
	}

	if len(resp.TopByMargin) != 2 {
		t.Fatalf("top_by_margin len = %d, want 2: %+v", len(resp.TopByMargin), resp.TopByMargin)
	}
	if m := resp.TopByMargin[0]; m.Name != "Aceite" || !eqNear(m.MarginPct, 33.33) || m.UnitsSold != 3 {
		t.Errorf("top_by_margin[0] = %+v, want Aceite 33.33%% / 3 u.", m)
	}
	if m := resp.TopByMargin[1]; m.Name != "Arroz" || !eqNear(m.MarginPct, 16.67) || m.UnitsSold != 4 {
		t.Errorf("top_by_margin[1] = %+v, want Arroz 16.67%% / 4 u.", m)
	}

	if len(resp.LowStock) != 2 {
		t.Fatalf("low_stock len = %d, want 2: %+v", len(resp.LowStock), resp.LowStock)
	}
	if l := resp.LowStock[0]; l.ProductID != 2 || l.Name != "Aceite" || l.Stock != 3 || l.MinStock != 10 {
		t.Errorf("low_stock[0] = %+v, want Aceite (id 2) 3/10", l)
	}
	if l := resp.LowStock[1]; l.ProductID != 3 || l.Name != "Gaseosa" || l.Stock != 1 || l.MinStock != 6 {
		t.Errorf("low_stock[1] = %+v, want Gaseosa (id 3) 1/6", l)
	}

	// Seeded with time.Time → julianday() is NULL → rows drop out (D-5). Pinned as-is.
	if !bytes.Contains(raw, []byte(`"aging_receivables":[]`)) {
		t.Errorf("aging_receivables must be [] (current behavior, D-5); body %s", raw)
	}
}

// ─── T-21: top_by_margin ≥ 2 rule ────────────────────────────────────────────

func TestDashboardToday_T21_TopByMarginNeedsTwo(t *testing.T) {
	db := setupTestDB(t)
	hoy0 := eqToday0()
	execSeed(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Prod A', 'A', 10000, 5000, 'unit', 1)`)
	execSeed(t, db, `INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 10000, 10000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	execSeed(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 10000, 10000)`)
	execSeed(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 10000)`)

	w := callDashboardToday(New(db))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
	raw := w.Body.Bytes()
	if !bytes.Contains(raw, []byte(`"top_by_margin":[]`)) {
		t.Errorf("top_by_margin must be [] with a single costed product; body %s", raw)
	}
	// Sanity: the sale itself is counted, so the empty list is the rule, not missing data.
	if !bytes.Contains(raw, []byte(`"gross_profit":5000`)) {
		t.Errorf("gross_profit must be 5000; body %s", raw)
	}
}

// ─── T-22: aging_receivables ─────────────────────────────────────────────────

func TestDashboardToday_T22_AgingReceivables(t *testing.T) {
	db := setupTestDB(t)
	seedAgingDatetime(t, db)

	w := callDashboardToday(New(db))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
	var env struct {
		Data struct {
			Aging []agingReceivable `json:"aging_receivables"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []agingReceivable{
		{CustomerID: 2, Name: "Beto", Balance: 20000, DaysOverdue: 10},
		{CustomerID: 4, Name: "Dora", Balance: 40000, DaysOverdue: 7},
		{CustomerID: 1, Name: "Ana", Balance: 15000, DaysOverdue: 3},
	}
	got := env.Data.Aging
	if len(got) != len(want) {
		t.Fatalf("aging_receivables = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("aging_receivables[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// ─── T-36: one balance definition across dashboard, portfolio and customers ─

func TestDashboardToday_T36_AgingBalanceMatchesPortfolioAndCustomers(t *testing.T) {
	db := setupTestDB(t)
	seedAgingDatetime(t, db)
	h := New(db)

	// /dashboard/today → aging_receivables (raw rows, to check the JSON keys too).
	wd := callDashboardToday(h)
	if wd.Code != http.StatusOK {
		t.Fatalf("dashboard status %d, want 200; body %s", wd.Code, wd.Body.String())
	}
	var dash struct {
		Data struct {
			Aging []map[string]json.RawMessage `json:"aging_receivables"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wd.Body.Bytes(), &dash); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}

	// /reports/portfolio.
	wp := callPortfolio(h)
	if wp.Code != http.StatusOK {
		t.Fatalf("portfolio status %d, want 200; body %s", wp.Code, wp.Body.String())
	}
	var port struct {
		Data []struct {
			ID      int64   `json:"id"`
			Balance float64 `json:"balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wp.Body.Bytes(), &port); err != nil {
		t.Fatalf("decode portfolio: %v", err)
	}
	portBal := map[int64]float64{}
	for _, p := range port.Data {
		portBal[p.ID] = p.Balance
	}

	// /customers.
	reqC := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	wc := httptest.NewRecorder()
	h.listCustomers(wc, reqC)
	if wc.Code != http.StatusOK {
		t.Fatalf("customers status %d, want 200; body %s", wc.Code, wc.Body.String())
	}
	var cust struct {
		Data struct {
			Customers []struct {
				ID      int64   `json:"id"`
				Balance float64 `json:"balance"`
			} `json:"customers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wc.Body.Bytes(), &cust); err != nil {
		t.Fatalf("decode customers: %v", err)
	}
	custBal := map[int64]float64{}
	for _, c := range cust.Data.Customers {
		custBal[c.ID] = c.Balance
	}

	if len(dash.Data.Aging) != 3 {
		t.Fatalf("aging_receivables len = %d, want 3 (Beto, Dora, Ana); body %s", len(dash.Data.Aging), wd.Body.String())
	}
	wantKeys := []string{"customer_id", "name", "balance", "days_overdue"}
	for i, row := range dash.Data.Aging {
		if len(row) != len(wantKeys) {
			t.Errorf("aging_receivables[%d] has %d keys, want exactly %v: %v", i, len(row), wantKeys, row)
		}
		for _, k := range wantKeys {
			if _, ok := row[k]; !ok {
				t.Errorf("aging_receivables[%d] missing key %q", i, k)
			}
		}
		var id int64
		var bal float64
		if err := json.Unmarshal(row["customer_id"], &id); err != nil {
			t.Fatalf("aging_receivables[%d].customer_id: %v", i, err)
		}
		if err := json.Unmarshal(row["balance"], &bal); err != nil {
			t.Fatalf("aging_receivables[%d].balance: %v", i, err)
		}
		pb, ok := portBal[id]
		if !ok || bal != pb {
			t.Errorf("customer %d: aging balance %.0f != portfolio balance %.0f (found %v)", id, bal, pb, ok)
		}
		cb, ok := custBal[id]
		if !ok || bal != cb {
			t.Errorf("customer %d: aging balance %.0f != /customers balance %.0f (found %v)", id, bal, cb, ok)
		}
		if id == 4 && bal != 40000 {
			t.Errorf("Dora (id 4) aging balance = %.0f, want 40000", bal)
		}
	}
	if portBal[4] != 40000 || custBal[4] != 40000 {
		t.Errorf("Dora (id 4): portfolio %.0f / customers %.0f, want 40000 / 40000", portBal[4], custBal[4])
	}
}

// ─── T-23: /reports/portfolio ────────────────────────────────────────────────

func TestPortfolio_T23_OrderAndFields(t *testing.T) {
	db := setupTestDB(t)
	seedPortfolioT23(t, db)

	w := callPortfolio(New(db))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
	var env struct {
		Data []struct {
			ID           int64   `json:"id"`
			Name         string  `json:"name"`
			Phone        string  `json:"phone"`
			Balance      float64 `json:"balance"`
			OldestDebtAt *string `json:"oldest_debt_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []struct {
		id      int64
		name    string
		phone   string
		balance float64
	}{
		{1, "Ana", "3001111111", 150000},
		{3, "Beto", "3002222222", 80000},
		{2, "Dora", "3004444444", 80000},
	}
	if len(env.Data) != len(want) {
		t.Fatalf("portfolio len = %d, want 3 (Carla at 0 excluded); body %s", len(env.Data), w.Body.String())
	}
	for i, wnt := range want {
		g := env.Data[i]
		if g.ID != wnt.id || g.Name != wnt.name || g.Phone != wnt.phone || g.Balance != wnt.balance {
			t.Errorf("portfolio[%d] = {%d %s %s %.0f}, want {%d %s %s %.0f}",
				i, g.ID, g.Name, g.Phone, g.Balance, wnt.id, wnt.name, wnt.phone, wnt.balance)
		}
		if g.OldestDebtAt == nil || *g.OldestDebtAt == "" {
			t.Errorf("portfolio[%d] (%s) oldest_debt_at is null, want a timestamp", i, g.Name)
		}
	}
}

// ─── T-24: empty portfolio → [] ──────────────────────────────────────────────

func TestPortfolio_T24_EmptyIsArray(t *testing.T) {
	db := setupTestDB(t)

	w := callPortfolio(New(db))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); body != `{"data":[]}` {
		t.Errorf("body = %s, want {\"data\":[]}", body)
	}
}

// ─── T-28, T-29: DB errors → 500 ─────────────────────────────────────────────

func TestDashboardToday_T28_DBErrorReturns500(t *testing.T) {
	db := setupTestDB(t)
	seedGoldenT20(t, db)
	h := New(db)
	db.Close()

	w := callDashboardToday(h)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want 500", w.Code)
	}
	body := w.Body.String()
	if strings.TrimSpace(body) != `{"error":"error obteniendo métricas"}` {
		t.Errorf("body = %s, want {\"error\":\"error obteniendo métricas\"}", body)
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err == nil {
		if _, has := m["data"]; has {
			t.Errorf("body must not contain key data: %s", body)
		}
	}
	if strings.Contains(body, "56000") {
		t.Errorf("body leaks a partial figure (56000): %s", body)
	}
}

func TestPortfolio_T29_DBErrorReturns500(t *testing.T) {
	db := setupTestDB(t)
	seedPortfolioT23(t, db)
	h := New(db)
	db.Close()

	w := callPortfolio(h)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want 500", w.Code)
	}
	body := w.Body.String()
	if strings.TrimSpace(body) != `{"error":"error obteniendo cartera"}` {
		t.Errorf("body = %s, want {\"error\":\"error obteniendo cartera\"}", body)
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err == nil {
		if _, has := m["data"]; has {
			t.Errorf("body must not contain key data: %s", body)
		}
	}
}
