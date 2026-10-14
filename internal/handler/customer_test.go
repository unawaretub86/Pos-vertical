package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

// seedCustomersT5 seeds the US-064 fixture (contract §4, shared with
// internal/repository/customer_test.go's T-1):
//
//	Dora: fiado $40.000, abono $10.000  -> saldo  $30.000  (en cartera)
//	Eli:  fiado $20.000, abono  $5.000  -> saldo  $15.000  (en cartera)
//	Fer:  fiado $20.000, abono $20.000  -> saldo      $0   (fuera de cartera)
//	Gus:  fiado $10.000, abono $30.000  -> saldo -$20.000  (saldo a favor, fuera de cartera)
func seedCustomersT5(t *testing.T, db *database.DB) *Handler {
	t.Helper()
	ctx := context.Background()
	rawDB := db.DB

	exec := func(q string, args ...any) {
		if _, err := rawDB.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("seed exec: %v\nSQL: %s", err, q)
		}
	}

	exec(`INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	exec(`INSERT INTO customers (id, name, phone, address) VALUES (2, 'Eli', '', '')`)
	exec(`INSERT INTO customers (id, name, phone, address) VALUES (3, 'Fer', '', '')`)
	exec(`INSERT INTO customers (id, name, phone, address) VALUES (4, 'Gus', '', '')`)

	exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 1, 40000, 40000, 'completed', 'credit', datetime('now','-5 days'))`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 40000)`)
	exec(`INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (1, 10000, datetime('now','-1 days'))`)

	exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 2, 20000, 20000, 'completed', 'credit', datetime('now','-4 days'))`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'credit', 20000)`)
	exec(`INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (2, 5000, datetime('now','-1 days'))`)

	exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (3, 1, 3, 20000, 20000, 'completed', 'credit', datetime('now','-3 days'))`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (3, 'credit', 20000)`)
	exec(`INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (3, 20000, datetime('now','-1 days'))`)

	exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (4, 1, 4, 10000, 10000, 'completed', 'credit', datetime('now','-2 days'))`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (4, 'credit', 10000)`)
	exec(`INSERT INTO customer_payments (customer_id, amount, created_at) VALUES (4, 30000, datetime('now','-1 days'))`)

	return New(db)
}

// ─── T-5: AC-5, AC-7 ─────────────────────────────────────────────────────────

func TestListCustomers_T5_TotalPortfolioMatchesPositiveBalances(t *testing.T) {
	db := setupTestDB(t)
	h := seedCustomersT5(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	w := httptest.NewRecorder()
	h.listCustomers(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode top-level: %v", err)
	}
	dataRaw, ok := raw["data"]
	if !ok {
		t.Fatalf("response has no top-level %q key; body %s", "data", w.Body.String())
	}

	var body struct {
		Customers []struct {
			ID      int64   `json:"id"`
			Balance float64 `json:"balance"`
		} `json:"customers"`
		TotalPortfolio float64 `json:"total_portfolio"`
	}
	if err := json.Unmarshal(dataRaw, &body); err != nil {
		t.Fatalf("decode data: %v", err)
	}

	// Exactly the keys the contract fixes (AC-7): customers and total_portfolio.
	var dataKeys map[string]json.RawMessage
	if err := json.Unmarshal(dataRaw, &dataKeys); err != nil {
		t.Fatalf("decode data keys: %v", err)
	}
	if len(dataKeys) != 2 {
		t.Fatalf("data has %d top-level keys, want 2 (customers, total_portfolio); keys: %v", len(dataKeys), dataKeys)
	}
	if _, ok := dataKeys["customers"]; !ok {
		t.Error(`data missing "customers" key`)
	}
	if _, ok := dataKeys["total_portfolio"]; !ok {
		t.Error(`data missing "total_portfolio" key`)
	}

	var sumPositive float64
	for _, c := range body.Customers {
		if c.Balance > 0 {
			sumPositive += c.Balance
		}
	}

	const want = 45000.0 // Dora 30.000 + Eli 15.000
	if sumPositive != want {
		t.Fatalf("sum of customers[].balance > 0 = %v, want %v", sumPositive, want)
	}
	if body.TotalPortfolio != want {
		t.Errorf("total_portfolio = %v, want %v (= sum of positive balances)", body.TotalPortfolio, want)
	}
	if body.TotalPortfolio != sumPositive {
		t.Errorf("total_portfolio (%v) != sum of customers[].balance > 0 (%v)", body.TotalPortfolio, sumPositive)
	}
}

// ─── Customer logical delete ────────────────────────────────────────────────

func TestDeleteCustomer_LogicalDelete(t *testing.T) {
	db := setupTestDB(t)
	h := seedCustomersT5(t, db)

	del := func(id string) int {
		req := withURLParam(httptest.NewRequest(http.MethodDelete, "/api/customers/"+id, nil), "id", id)
		w := httptest.NewRecorder()
		h.deleteCustomer(w, req)
		return w.Code
	}

	// Fer (saldo $0) and Gus (saldo -$20.000) have history and are deactivated.
	if code := del("3"); code != http.StatusOK {
		t.Errorf("delete Fer = %d, want 200", code)
	}
	if code := del("4"); code != http.StatusOK {
		t.Errorf("delete Gus = %d, want 200", code)
	}
	// Dora owes $30.000 → 409; unknown and already inactive ids → 404.
	if code := del("1"); code != http.StatusConflict {
		t.Errorf("delete Dora = %d, want 409", code)
	}
	if code := del("999"); code != http.StatusNotFound {
		t.Errorf("delete 999 = %d, want 404", code)
	}
	if code := del("3"); code != http.StatusNotFound {
		t.Errorf("delete Fer twice = %d, want 404", code)
	}

	var rows, sales int
	if err := db.QueryRow(`SELECT COUNT(*) FROM customers`).Scan(&rows); err != nil || rows != 4 {
		t.Errorf("customer rows = %d (err %v), want 4", rows, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sales WHERE customer_id IN (3, 4)`).Scan(&sales); err != nil || sales != 2 {
		t.Errorf("sales kept = %d (err %v), want 2", sales, err)
	}

	// List keeps Dora and Eli only; portfolio stays $30.000 + $15.000 = $45.000.
	w := httptest.NewRecorder()
	h.listCustomers(w, httptest.NewRequest(http.MethodGet, "/api/customers", nil))
	var resp struct {
		Data struct {
			Customers []struct {
				ID int64 `json:"id"`
			} `json:"customers"`
			TotalPortfolio float64 `json:"total_portfolio"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(resp.Data.Customers) != 2 || resp.Data.Customers[0].ID != 1 || resp.Data.Customers[1].ID != 2 {
		t.Errorf("list = %+v, want ids [1 2]", resp.Data.Customers)
	}
	if resp.Data.TotalPortfolio != 45000 {
		t.Errorf("total_portfolio = %v, want 45000", resp.Data.TotalPortfolio)
	}

	// Search does not find the inactive customer.
	w = httptest.NewRecorder()
	h.listCustomers(w, httptest.NewRequest(http.MethodGet, "/api/customers?q=Fer", nil))
	resp.Data.Customers = nil
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if len(resp.Data.Customers) != 0 {
		t.Errorf("search Fer = %+v, want none", resp.Data.Customers)
	}

	// History still resolves the name of the inactive customer.
	w = httptest.NewRecorder()
	h.getCustomer(w, withURLParam(httptest.NewRequest(http.MethodGet, "/api/customers/3", nil), "id", "3"))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"Fer"`) {
		t.Errorf("get Fer = %d %s, want 200 with name", w.Code, w.Body.String())
	}
}
