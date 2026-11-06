package handler

// US-082 contract §4, T-2 .. T-6 and the handler side of T-7. These go through
// h.Routes() with real JWTs (tokenFor, assistant_test.go) so the role rules
// and the route group of POST /customers/{id}/payments are exercised.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

const (
	abonoAdminID   = 1
	abonoCashierID = 2
)

func abonoExec(t *testing.T, db *database.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.DB.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

// seedAbonoBase relies on the admin (id 1) seeded by migration 1 and adds a
// cashier (id 2) and customer 1 (Dora) with a fiado of $200,000, enough room
// for every abono these tests post.
func seedAbonoBase(t *testing.T, db *database.DB) {
	t.Helper()
	abonoExec(t, db, `INSERT INTO users (id, username, password_hash, role) VALUES (2, 'cajero1', 'h', 'cashier')`)
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	abonoExec(t, db, `INSERT INTO sales (id, user_id, customer_id, subtotal, total, status, payment_method) VALUES (1, 1, 1, 200000, 200000, 'completed', 'credit')`)
	abonoExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'credit', 200000)`)
}

// seedOpenRegister opens register 1 with an opening float of $100,000 and one
// completed cash sale of $30,000 linked to it.
func seedOpenRegister(t *testing.T, db *database.DB) {
	t.Helper()
	abonoExec(t, db, `INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 100000, 'open')`)
	abonoExec(t, db, `INSERT INTO sales (id, user_id, register_id, subtotal, total, status, payment_method) VALUES (2, 1, 1, 30000, 30000, 'completed', 'cash')`)
	abonoExec(t, db, `INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 30000)`)
}

func abonoRequest(t *testing.T, h *Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

// paymentJSON is one abono as returned by the API. RegisterID is a raw
// message so that an explicit JSON null can be told apart from a missing key.
type paymentJSON struct {
	ID         int64           `json:"id"`
	Amount     float64         `json:"amount"`
	Method     *string         `json:"method"`
	RegisterID json.RawMessage `json:"register_id"`
}

func abonoData(t *testing.T, w *httptest.ResponseRecorder, into any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v; body %s", err, w.Body.String())
	}
	if err := json.Unmarshal(env.Data, into); err != nil {
		t.Fatalf("decode data: %v; body %s", err, w.Body.String())
	}
}

func abonoError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error body: %v; body %s", err, w.Body.String())
	}
	return env.Error
}

func assertPayment(t *testing.T, where string, p paymentJSON, amount float64, method, registerID string) {
	t.Helper()
	if p.Amount != amount {
		t.Errorf("%s: amount = %.0f, want %.0f", where, p.Amount, amount)
	}
	if p.Method == nil {
		t.Errorf("%s: method missing from JSON, want %q", where, method)
	} else if *p.Method != method {
		t.Errorf("%s: method = %q, want %q", where, *p.Method, method)
	}
	if len(p.RegisterID) == 0 {
		t.Errorf("%s: register_id missing from JSON, want %s", where, registerID)
	} else if string(p.RegisterID) != registerID {
		t.Errorf("%s: register_id = %s, want %s", where, p.RegisterID, registerID)
	}
}

type registerSummaryJSON struct {
	CashSales  float64  `json:"cash_sales"`
	CashAbonos *float64 `json:"cash_abonos"`
	Expected   float64  `json:"expected"`
}

func getRegisterSummary(t *testing.T, h *Handler, token string) registerSummaryJSON {
	t.Helper()
	w := abonoRequest(t, h, http.MethodGet, "/registers/1", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /registers/1 status = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var s registerSummaryJSON
	abonoData(t, w, &s)
	return s
}

func countPayments(t *testing.T, db *database.DB) int {
	t.Helper()
	var n int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM customer_payments`).Scan(&n); err != nil {
		t.Fatalf("count customer_payments: %v", err)
	}
	return n
}

// ─── T-2: AC-4, AC-8, AC-10, AC-11 ───────────────────────────────────────────

func TestCustomerPayment_T2_CashAbonoLinkedAndCountedInExpected(t *testing.T) {
	db := setupTestDB(t)
	seedAbonoBase(t, db)
	seedOpenRegister(t, db)
	h := New(db)
	admin := tokenFor(t, abonoAdminID, "admin")

	w := abonoRequest(t, h, http.MethodPost, "/customers/1/payments", admin, `{"amount":20000,"method":"cash"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var created paymentJSON
	abonoData(t, w, &created)
	assertPayment(t, "201 response", created, 20000, "cash", "1")

	// expected = 100,000 opening + 30,000 cash sale + 20,000 cash abono = 150,000
	s := getRegisterSummary(t, h, admin)
	if s.CashAbonos == nil {
		t.Fatalf("GET /registers/1 has no cash_abonos; body fields cash_sales=%.0f expected=%.0f", s.CashSales, s.Expected)
	}
	if *s.CashAbonos != 20000 {
		t.Errorf("cash_abonos = %.0f, want 20000", *s.CashAbonos)
	}
	if s.Expected != 150000 {
		t.Errorf("expected = %.0f, want 150000", s.Expected)
	}

	// GET /customers/1/payments (G-3)
	w = abonoRequest(t, h, http.MethodGet, "/customers/1/payments", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET payments status = %d; body %s", w.Code, w.Body.String())
	}
	var list []paymentJSON
	abonoData(t, w, &list)
	if len(list) != 1 {
		t.Fatalf("GET payments len = %d, want 1", len(list))
	}
	assertPayment(t, "GET /customers/1/payments", list[0], 20000, "cash", "1")

	// GET /customers/1/detail payments[] (G-3)
	w = abonoRequest(t, h, http.MethodGet, "/customers/1/detail", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET detail status = %d; body %s", w.Code, w.Body.String())
	}
	var detail struct {
		Payments []paymentJSON `json:"payments"`
	}
	abonoData(t, w, &detail)
	if len(detail.Payments) != 1 {
		t.Fatalf("detail payments len = %d, want 1", len(detail.Payments))
	}
	assertPayment(t, "GET /customers/1/detail", detail.Payments[0], 20000, "cash", "1")
}

// ─── T-3: AC-13 ──────────────────────────────────────────────────────────────

func TestCustomerPayment_T3_TransferAndCardAbonosDoNotChangeExpected(t *testing.T) {
	db := setupTestDB(t)
	seedAbonoBase(t, db)
	seedOpenRegister(t, db)
	h := New(db)
	admin := tokenFor(t, abonoAdminID, "admin")

	for _, body := range []string{
		`{"amount":20000,"method":"cash"}`,
		`{"amount":50000,"method":"transfer"}`,
		`{"amount":10000,"method":"card"}`,
	} {
		w := abonoRequest(t, h, http.MethodPost, "/customers/1/payments", admin, body)
		if w.Code != http.StatusCreated {
			t.Fatalf("POST %s status = %d, want 201; body %s", body, w.Code, w.Body.String())
		}
	}

	s := getRegisterSummary(t, h, admin)
	if s.CashAbonos == nil {
		t.Fatal("GET /registers/1 has no cash_abonos")
	}
	if *s.CashAbonos != 20000 {
		t.Errorf("cash_abonos = %.0f, want 20000 (transfer 50,000 and card 10,000 excluded)", *s.CashAbonos)
	}
	if s.Expected != 150000 {
		t.Errorf("expected = %.0f, want 150000", s.Expected)
	}
}

// ─── T-4: AC-6, AC-4 ─────────────────────────────────────────────────────────

func TestCustomerPayment_T4_CashierNeedsOpenRegister(t *testing.T) {
	db := setupTestDB(t)
	seedAbonoBase(t, db)
	seedOpenRegister(t, db)
	h := New(db)
	cashier := tokenFor(t, abonoCashierID, "cashier")

	w := abonoRequest(t, h, http.MethodPost, "/customers/1/payments", cashier, `{"amount":20000,"method":"cash"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("cashier POST with open register status = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var created paymentJSON
	abonoData(t, w, &created)
	assertPayment(t, "cashier 201 response", created, 20000, "cash", "1")

	abonoExec(t, db, `UPDATE registers SET status = 'closed', closed_at = CURRENT_TIMESTAMP WHERE id = 1`)
	before := countPayments(t, db)
	if before != 1 {
		t.Fatalf("payments before second POST = %d, want 1", before)
	}

	w = abonoRequest(t, h, http.MethodPost, "/customers/1/payments", cashier, `{"amount":5000,"method":"cash"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cashier POST with no open register status = %d, want 403; body %s", w.Code, w.Body.String())
	}
	msg := abonoError(t, w)
	if msg == "" || msg == "forbidden" {
		t.Errorf("403 error = %q, want the service's Spanish message (not the RequireAdmin middleware's)", msg)
	}
	if after := countPayments(t, db); after != before {
		t.Errorf("payments after rejected POST = %d, want %d", after, before)
	}
}

// ─── T-5: AC-5 ───────────────────────────────────────────────────────────────

func TestCustomerPayment_T5_AdminWithoutRegisterStoresUnattributed(t *testing.T) {
	db := setupTestDB(t)
	seedAbonoBase(t, db) // no register at all
	h := New(db)
	admin := tokenFor(t, abonoAdminID, "admin")

	w := abonoRequest(t, h, http.MethodPost, "/customers/1/payments", admin, `{"amount":20000,"method":"cash"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var created paymentJSON
	abonoData(t, w, &created)
	assertPayment(t, "201 response", created, 20000, "cash", "null")
}

// ─── T-6: AC-2, AC-3 ─────────────────────────────────────────────────────────

func TestCustomerPayment_T6_MethodDefaultAndValidation(t *testing.T) {
	db := setupTestDB(t)
	seedAbonoBase(t, db)
	h := New(db)
	admin := tokenFor(t, abonoAdminID, "admin")

	storedMethod := func(id int64) string {
		t.Helper()
		var m string
		if err := db.DB.QueryRow(`SELECT method FROM customer_payments WHERE id = ?`, id).Scan(&m); err != nil {
			t.Fatalf("read method of payment %d: %v", id, err)
		}
		return m
	}

	// No method -> cash.
	w := abonoRequest(t, h, http.MethodPost, "/customers/1/payments", admin, `{"amount":10000}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST without method status = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var p1 paymentJSON
	abonoData(t, w, &p1)
	if got := storedMethod(p1.ID); got != "cash" {
		t.Errorf("stored method without method in body = %q, want cash", got)
	}

	// Unknown method -> 400, nothing stored.
	before := countPayments(t, db)
	w = abonoRequest(t, h, http.MethodPost, "/customers/1/payments", admin, `{"amount":10000,"method":"nequi"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST method=nequi status = %d, want 400; body %s", w.Code, w.Body.String())
	}
	if after := countPayments(t, db); after != before {
		t.Errorf("payments after method=nequi = %d, want %d", after, before)
	}

	// transfer -> stored transfer.
	w = abonoRequest(t, h, http.MethodPost, "/customers/1/payments", admin, `{"amount":10000,"method":"transfer"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST method=transfer status = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var p3 paymentJSON
	abonoData(t, w, &p3)
	if got := storedMethod(p3.ID); got != "transfer" {
		t.Errorf("stored method = %q, want transfer", got)
	}
}

// ─── T-7 (handler side): AC-12, G-2 ──────────────────────────────────────────

func TestRegisterSummary_T7_FailingAbonoSumReturns500(t *testing.T) {
	db := setupTestDB(t)
	seedAbonoBase(t, db)
	seedOpenRegister(t, db)
	h := New(db)
	admin := tokenFor(t, abonoAdminID, "admin")

	// Make the cash-abono sum fail; every other register sum still works.
	abonoExec(t, db, `DROP TABLE customer_payments`)

	w := abonoRequest(t, h, http.MethodGet, "/registers/1", admin, "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /registers/1 status = %d, want 500; body %s", w.Code, w.Body.String())
	}
	w = abonoRequest(t, h, http.MethodGet, "/registers/open", admin, "")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("GET /registers/open status = %d, want 500; body %s", w.Code, w.Body.String())
	}
	// An unknown register is still 404, not 500.
	w = abonoRequest(t, h, http.MethodGet, "/registers/999", admin, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /registers/999 status = %d, want 404; body %s", w.Code, w.Body.String())
	}
}
