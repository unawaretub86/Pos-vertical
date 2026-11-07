package handler

// Tests for GET /customers/{id}/pos-summary — contract US-043 §4, T-14, T-15.
// They go through h.Routes() with a cashier JWT so the route's role gate
// (authenticated group, outside RequireAdmin) is exercised, not only the
// handler method.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

func posSummaryExec(t *testing.T, db *database.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.DB.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

func getPOSSummary(h *Handler, token, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/customers/"+id+"/pos-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

type posSummaryEnvelope struct {
	Data struct {
		LastVisitAt *string `json:"last_visit_at"`
		TopProducts []struct {
			ProductID   int64  `json:"product_id"`
			Name        string `json:"name"`
			TimesBought int    `json:"times_bought"`
		} `json:"top_products"`
		Balance     float64 `json:"balance"`
		DaysOverdue *int    `json:"days_overdue"`
		CreditLimit *int64  `json:"credit_limit"`
	} `json:"data"`
}

// ─── T-14: AC-20 ─────────────────────────────────────────────────────────────

func TestPOSSummary_T14_CashierGetsSummaryAnd404And400(t *testing.T) {
	db := setupTestDB(t)
	// Pedro: limit $500,000, one credit sale of $450,000 26 days ago (Arroz).
	posSummaryExec(t, db, `INSERT INTO customers (id, name, phone, address, credit_limit) VALUES (1, 'Pedro', '', '', 500000)`)
	posSummaryExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Arroz', 'ARZ', 450000, 0, 'unit', 1)`)
	posSummaryExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 450000, 450000, 'completed', 'credit', datetime('now','-26 days'))`)
	posSummaryExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 450000)`)
	posSummaryExec(t, db, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 1, 450000, 450000)`)
	h := New(db)
	token := tokenFor(t, 2, "cashier")

	t.Run("known customer as cashier", func(t *testing.T) {
		w := getPOSSummary(h, token, "1")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
		}
		var env posSummaryEnvelope
		if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
			t.Fatalf("decode: %v", err)
		}
		d := env.Data
		if d.Balance != 450000 {
			t.Errorf("balance = %v, want 450000", d.Balance)
		}
		if d.DaysOverdue == nil || *d.DaysOverdue != 26 {
			t.Errorf("days_overdue = %v, want 26", d.DaysOverdue)
		}
		if d.CreditLimit == nil || *d.CreditLimit != 500000 {
			t.Errorf("credit_limit = %v, want 500000", d.CreditLimit)
		}
		if d.LastVisitAt == nil || *d.LastVisitAt == "" {
			t.Errorf("last_visit_at = %v, want the 26-days-ago sale timestamp", d.LastVisitAt)
		}
		if len(d.TopProducts) != 1 || d.TopProducts[0].Name != "Arroz" || d.TopProducts[0].ProductID != 1 || d.TopProducts[0].TimesBought != 1 {
			t.Errorf("top_products = %+v, want [{1 Arroz 1}]", d.TopProducts)
		}
	})

	t.Run("unknown id is 404", func(t *testing.T) {
		w := getPOSSummary(h, token, "999")
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404; body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-numeric id is 400", func(t *testing.T) {
		w := getPOSSummary(h, token, "abc")
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400; body: %s", w.Code, w.Body.String())
		}
	})
}

// ─── T-15: AC-20 ─────────────────────────────────────────────────────────────

func TestPOSSummary_T15_CustomerWithNoSales(t *testing.T) {
	db := setupTestDB(t)
	posSummaryExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Nuevo', '', '')`)
	h := New(db)

	w := getPOSSummary(h, tokenFor(t, 2, "cashier"), "1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (not 404); body: %s", w.Code, w.Body.String())
	}

	// Decode raw so `top_products: []` can be told apart from `null`.
	var env struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	checks := map[string]string{
		"last_visit_at": "null",
		"top_products":  "[]",
		"balance":       "0",
		"days_overdue":  "null",
	}
	for _, key := range []string{"last_visit_at", "top_products", "balance", "days_overdue"} {
		raw, ok := env.Data[key]
		if !ok {
			t.Errorf("%s missing from response", key)
			continue
		}
		if got := string(raw); got != checks[key] {
			t.Errorf("%s = %s, want %s", key, got, checks[key])
		}
	}
}
