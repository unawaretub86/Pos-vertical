package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

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

	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, saleTime)

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
