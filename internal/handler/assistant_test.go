package handler

// Tests for POST /ai/ask — contrato US-061 §4, T-2, T-12..T-14, T-21. Unlike
// the rest of this package's tests (which call the handler method directly,
// bypassing auth), these go through h.Routes() so the real appmw.Auth/
// appmw.RequireAdmin middleware chain is exercised (AC-12/AC-13 are
// middleware-level guarantees, not handler-level ones).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
)

func tokenFor(t *testing.T, userID int64, role string) string {
	t.Helper()
	tok, err := appmw.GenerateToken(&model.Claims{UserID: userID, Username: "u", Role: role})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return tok
}

// askAskEndpoint performs POST /ai/ask against the full middleware chain
// (h.Routes()). token == "" omits the Authorization header entirely (AC-12).
func askAskEndpoint(h *Handler, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/ai/ask", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

// askEnvelope mirrors respond()'s {"data": AskResult} wrapper, with AskResult's
// own "data" payload narrowed to the sales-shaped fields T-2/T-14 need.
type askEnvelope struct {
	Data struct {
		Intent string `json:"intent"`
		Text   string `json:"text"`
		Data   struct {
			Total float64 `json:"total"`
			Count int     `json:"count"`
			Items []struct {
				Name    string  `json:"name"`
				Balance float64 `json:"balance"`
			} `json:"items"`
		} `json:"data"`
		Suggestions []string `json:"suggestions"`
		Tier        string   `json:"tier"`
	} `json:"data"`
}

func decodeAskResponse(t *testing.T, w *httptest.ResponseRecorder) askEnvelope {
	t.Helper()
	var env askEnvelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode /ai/ask response: %v\nbody: %s", err, w.Body.String())
	}
	return env
}

// ─── T-2 (AC-1, handler) ──────────────────────────────────────────────────

func TestAskAssistant_T2_SalesTotalMatchesDashboard(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	hoy0 := time.Now()
	hoy0 = time.Date(hoy0.Year(), hoy0.Month(), hoy0.Day(), 0, 0, 0, 0, hoy0.Location())
	raw := db.DB
	exec := func(q string, args ...any) {
		if _, err := raw.ExecContext(context.Background(), q, args...); err != nil {
			t.Fatalf("seed exec: %v", err)
		}
	}
	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 28000, 28000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour))
	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (2, 1, 12000, 12000, 'completed', 'cash', ?)`, hoy0.Add(2*time.Hour))

	token := tokenFor(t, 1, "admin")
	w := askAskEndpoint(h, token, `{"query":"cuanto vendi hoy"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	env := decodeAskResponse(t, w)
	if env.Data.Data.Total != 40000 {
		t.Errorf("data.total = %.0f, want 40000", env.Data.Data.Total)
	}
	if env.Data.Data.Count != 2 {
		t.Errorf("data.count = %d, want 2", env.Data.Data.Count)
	}

	// Same figure as /dashboard/today.sold_today for the same seed.
	dashReq := httptest.NewRequest(http.MethodGet, "/dashboard/today", nil)
	dashReq.Header.Set("Authorization", "Bearer "+token)
	dashW := httptest.NewRecorder()
	h.Routes().ServeHTTP(dashW, dashReq)
	if dashW.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200", dashW.Code)
	}
	dashResp := decodeTodayResponse(t, dashW)
	if dashResp.SoldToday != env.Data.Data.Total {
		t.Errorf("dashboard sold_today = %.0f, assistant total = %.0f, want equal", dashResp.SoldToday, env.Data.Data.Total)
	}
}

// ─── T-12 (AC-12) ─────────────────────────────────────────────────────────

func TestAskAssistant_T12_NoTokenReturns401(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	w := askAskEndpoint(h, "", `{"query":"cuanto vendi hoy"}`)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), `{"error":"unauthorized"}`) {
		t.Errorf("body = %q, want it to contain {\"error\":\"unauthorized\"}", w.Body.String())
	}
}

// ─── T-13 (AC-13) ─────────────────────────────────────────────────────────

func TestAskAssistant_T13_CashierReturns403WithoutLeakingFigures(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	hoy0 := time.Now()
	hoy0 = time.Date(hoy0.Year(), hoy0.Month(), hoy0.Day(), 0, 0, 0, 0, hoy0.Location())
	if _, err := db.DB.ExecContext(context.Background(),
		`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, created_at) VALUES (1, 1, 40000, 40000, 'completed', 'cash', ?)`, hoy0.Add(time.Hour),
	); err != nil {
		t.Fatalf("seed exec: %v", err)
	}

	token := tokenFor(t, 2, "cashier")
	w := askAskEndpoint(h, token, `{"query":"cuanto vendi hoy"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `{"error":"forbidden"}`) {
		t.Errorf("body = %q, want it to contain {\"error\":\"forbidden\"}", body)
	}
	if strings.Contains(body, "$") || strings.Contains(body, "40000") || strings.Contains(body, "40.000") {
		t.Errorf("body = %q, must not leak any figure to a non-admin", body)
	}
}

// ─── T-14 (AC-14) ─────────────────────────────────────────────────────────

func TestAskAssistant_T14_DebtorsTruncatesAtFiveWithRemainder(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	raw := db.DB
	exec := func(q string, args ...any) {
		if _, err := raw.ExecContext(context.Background(), q, args...); err != nil {
			t.Fatalf("seed exec: %v", err)
		}
	}
	balances := []float64{60000, 50000, 40000, 30000, 20000, 10000}
	for i, bal := range balances {
		id := int64(i + 1)
		exec(`INSERT INTO customers (id, name, phone, address) VALUES (?, ?, '', '')`, id, "Cliente"+string(rune('A'+i)))
		exec(`INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method, created_at) VALUES (?, 1, ?, ?, ?, 'completed', 'credit', ?)`,
			id, id, bal, bal, time.Now().Add(-time.Duration(i+1)*time.Hour))
		exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (?, 'credit', ?)`, id, bal)
	}

	token := tokenFor(t, 1, "admin")
	w := askAskEndpoint(h, token, `{"query":"quien me debe"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
	env := decodeAskResponse(t, w)
	if len(env.Data.Data.Items) != 5 {
		t.Fatalf("items len = %d, want 5 (truncated from 6): %+v", len(env.Data.Data.Items), env.Data.Data.Items)
	}
	if !strings.Contains(env.Data.Text, "y 1 más") {
		t.Errorf("text = %q, want it to mention the remaining debtor (\"y 1 más\")", env.Data.Text)
	}
}

// ─── T-21 (AC-15, user_id always from JWT) ────────────────────────────────

func TestAskAssistant_T21_UserIDAlwaysFromJWTNeverFromBody(t *testing.T) {
	db := setupTestDB(t)
	if _, err := db.DB.ExecContext(context.Background(),
		`INSERT INTO users (id, username, password_hash, role) VALUES (3, 'u3', 'h', 'admin'), (7, 'u7', 'h', 'admin')`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	h := New(db)

	token3 := tokenFor(t, 3, "admin")
	token7 := tokenFor(t, 7, "admin")

	w1 := askAskEndpoint(h, token3, `{"query":"cuanto vendi hoy"}`)
	if w1.Code != http.StatusOK {
		t.Fatalf("request 1 status = %d, want 200", w1.Code)
	}
	// Spurious "user_id" field in the body must be ignored: askAssistantRequest
	// does not declare it, so json.Decode drops it silently (AC-15, GATE 1).
	w2 := askAskEndpoint(h, token7, `{"query":"cuanto vendi hoy","user_id":999}`)
	if w2.Code != http.StatusOK {
		t.Fatalf("request 2 status = %d, want 200", w2.Code)
	}

	rows, err := db.DB.QueryContext(context.Background(), `SELECT user_id FROM assistant_queries ORDER BY id ASC`)
	if err != nil {
		t.Fatalf("query bitacora: %v", err)
	}
	defer rows.Close()
	var userIDs []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			t.Fatalf("scan: %v", err)
		}
		userIDs = append(userIDs, uid)
	}
	if len(userIDs) != 2 {
		t.Fatalf("assistant_queries rows = %v, want exactly 2", userIDs)
	}
	if userIDs[0] != 3 {
		t.Errorf("first row user_id = %d, want 3 (from token, not body)", userIDs[0])
	}
	if userIDs[1] != 7 {
		t.Errorf("second row user_id = %d, want 7 (from token, not the spurious 999 in body)", userIDs[1])
	}
}
