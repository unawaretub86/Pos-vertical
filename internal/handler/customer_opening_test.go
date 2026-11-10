package handler

// US-092 contract §4, T-1, T-4 .. T-11: the opening fiado balance (deuda
// anterior) through h.Routes() with real JWTs.
//
// Seed (contract §4): admin id 1 (migration 1), cashier id 2, customers 7 Dora
// and 8 Beto (active), 9 Eva (active = 0). D45/D50 = today (server-local)
// minus 45/50 days as YYYY-MM-DD. setupTestDB(t) everywhere except the
// concurrency case and the register close, which use the file-backed
// shiftSetupDB(t) (US-091).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

const (
	obAmountMsg    = "el monto debe ser un valor entero entre $1 y $100.000.000"
	obDateMsg      = "fecha de la deuda inválida"
	obFutureMsg    = "la fecha de la deuda no puede ser futura"
	obDuplicateMsg = "este cliente ya tiene una deuda anterior registrada"
	obNoneMsg      = "el cliente no tiene deuda anterior"
	obNotFoundMsg  = "cliente no encontrado"
	obBadIDMsg     = "id inválido"
	obInternalMsg  = "error registrando deuda anterior"
	obCreateAction = "opening_balance_create"
	obUpdateAction = "opening_balance_update"
	obPath7        = "/customers/7/opening-balance"
	obPath8        = "/customers/8/opening-balance"
)

// obDate returns today (server-local) plus days as YYYY-MM-DD.
func obDate(days int) string {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+days, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}

type obEnv struct {
	db      *database.DB
	h       *Handler
	admin   string
	cashier string
	clock   time.Time // injected handler clock (zero = real time)
}

func obSeedCustomers(t *testing.T, db *database.DB) {
	t.Helper()
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (7, 'Dora', '', '')`)
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (8, 'Beto', '', '')`)
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address, active) VALUES (9, 'Eva', '', '', 0)`)
}

func obWrap(t *testing.T, db *database.DB) *obEnv {
	t.Helper()
	return &obEnv{
		db:      db,
		h:       New(db),
		admin:   tokenFor(t, abonoAdminID, "admin"),
		cashier: tokenFor(t, abonoCashierID, "cashier"),
	}
}

// newObEnv is the in-memory fixture.
func newObEnv(t *testing.T) *obEnv {
	t.Helper()
	db := setupTestDB(t)
	abonoExec(t, db, `INSERT INTO users (id, username, password_hash, role) VALUES (2, 'cajero1', 'h', 'cashier')`)
	obSeedCustomers(t, db)
	return obWrap(t, db)
}

// newObFileEnv is the file-backed fixture (shiftSetupDB already seeds the
// cashier, products 1..3 and customer 1).
func newObFileEnv(t *testing.T) *obEnv {
	t.Helper()
	db := shiftSetupDB(t)
	obSeedCustomers(t, db)
	return obWrap(t, db)
}

func (e *obEnv) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	return abonoRequest(t, e.h, method, path, token, body)
}

func (e *obEnv) mustStatus(t *testing.T, method, path, token, body string, want int) *httptest.ResponseRecorder {
	t.Helper()
	w := e.do(t, method, path, token, body)
	if w.Code != want {
		t.Fatalf("%s %s %s = %d, want %d; body %s", method, path, body, w.Code, want, w.Body.String())
	}
	return w
}

func obBody(amount any, debtDate, note string) string {
	return fmt.Sprintf(`{"amount":%v,"debt_date":%q,"note":%q}`, amount, debtDate, note)
}

// obErr returns the "error" field of a JSON body, or "" when the body is not
// a JSON error envelope (e.g. chi's plain-text 404 for an unknown route).
func obErr(w *httptest.ResponseRecorder) string {
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		return ""
	}
	return env.Error
}

func obExpectError(t *testing.T, where string, w *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	if w.Code != status {
		t.Errorf("%s: status = %d, want %d; body %s", where, w.Code, status, w.Body.String())
		return
	}
	if got := obErr(w); got != msg {
		t.Errorf("%s: error = %q, want %q", where, got, msg)
	}
}

type obJSON struct {
	ID         int64   `json:"id"`
	CustomerID int64   `json:"customer_id"`
	Amount     float64 `json:"amount"`
	DebtDate   string  `json:"debt_date"`
	Note       string  `json:"note"`
	CreatedBy  int64   `json:"created_by"`
	CreatedAt  string  `json:"created_at"`
}

func (e *obEnv) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.db.DB.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v\nSQL: %s", err, q)
	}
	return n
}

func (e *obEnv) openingRows(t *testing.T) int {
	t.Helper()
	return e.count(t, `SELECT COUNT(*) FROM customer_opening_balances`)
}

func (e *obEnv) auditRows(t *testing.T, action string) int {
	t.Helper()
	return e.count(t, `SELECT COUNT(*) FROM audit_log WHERE action = ?`, action)
}

func (e *obEnv) balance(t *testing.T, id int64) float64 {
	t.Helper()
	w := e.mustStatus(t, http.MethodGet, fmt.Sprintf("/customers/%d/balance", id), e.cashier, "", http.StatusOK)
	var b struct {
		Balance *float64 `json:"balance"`
	}
	abonoData(t, w, &b)
	if b.Balance == nil {
		t.Fatalf("GET /customers/%d/balance has no balance key; body %s", id, w.Body.String())
	}
	return *b.Balance
}

type obDetailJSON struct {
	Balance        float64         `json:"balance"`
	Payments       json.RawMessage `json:"payments"`
	CreditSales    json.RawMessage `json:"credit_sales"`
	OpeningBalance json.RawMessage `json:"opening_balance"`
}

func (e *obEnv) detail(t *testing.T, id int64) obDetailJSON {
	t.Helper()
	w := e.mustStatus(t, http.MethodGet, fmt.Sprintf("/customers/%d/detail", id), e.cashier, "", http.StatusOK)
	var d obDetailJSON
	abonoData(t, w, &d)
	return d
}

type obPOSSummaryJSON struct {
	Balance     float64         `json:"balance"`
	DaysOverdue *int            `json:"days_overdue"`
	LastVisitAt json.RawMessage `json:"last_visit_at"`
	TopProducts json.RawMessage `json:"top_products"`
}

func (e *obEnv) posSummary(t *testing.T, id int64) obPOSSummaryJSON {
	t.Helper()
	w := e.mustStatus(t, http.MethodGet, fmt.Sprintf("/customers/%d/pos-summary", id), e.cashier, "", http.StatusOK)
	var s obPOSSummaryJSON
	abonoData(t, w, &s)
	return s
}

func emptyList(raw json.RawMessage) bool {
	s := string(raw)
	return s == "null" || s == "[]"
}

func (e *obEnv) createDora(t *testing.T, amount float64, debtDate string) obJSON {
	t.Helper()
	w := e.mustStatus(t, http.MethodPost, obPath7, e.admin, obBody(amount, debtDate, ""), http.StatusCreated)
	var o obJSON
	abonoData(t, w, &o)
	return o
}

// ─── T-1: AC-1, AC-8, AC-13 ──────────────────────────────────────────────────

func TestCustomerOpening_T1_CreateAddsToBalanceAndDetail(t *testing.T) {
	e := newObEnv(t)
	d45 := obDate(-45)

	w := e.do(t, http.MethodPost, obPath7, e.admin, obBody(150000, d45, "  Cuaderno pág. 12  "))
	if w.Code != http.StatusCreated {
		t.Fatalf("POST %s = %d, want 201; body %s", obPath7, w.Code, w.Body.String())
	}
	var o obJSON
	abonoData(t, w, &o)
	if o.CustomerID != 7 || o.Amount != 150000 || o.DebtDate != d45 || o.Note != "Cuaderno pág. 12" || o.CreatedBy != 1 {
		t.Errorf("created = %+v, want customer_id 7, amount 150000, debt_date %s, note %q (trimmed), created_by 1",
			o, d45, "Cuaderno pág. 12")
	}
	if o.ID <= 0 || o.CreatedAt == "" {
		t.Errorf("created id = %d, created_at = %q, want a positive id and a timestamp", o.ID, o.CreatedAt)
	}

	if got := e.balance(t, 7); got != 150000 {
		t.Errorf("GET /customers/7/balance = %.0f, want 150000", got)
	}

	d := e.detail(t, 7)
	if d.Balance != 150000 {
		t.Errorf("detail balance = %.0f, want 150000", d.Balance)
	}
	var ob obJSON
	if len(d.OpeningBalance) == 0 || string(d.OpeningBalance) == "null" {
		t.Fatalf("detail opening_balance = %s, want the created object", d.OpeningBalance)
	}
	if err := json.Unmarshal(d.OpeningBalance, &ob); err != nil {
		t.Fatalf("decode opening_balance: %v; raw %s", err, d.OpeningBalance)
	}
	if ob.Amount != 150000 || ob.DebtDate != d45 || ob.CustomerID != 7 {
		t.Errorf("detail opening_balance = %+v, want amount 150000, debt_date %s, customer_id 7", ob, d45)
	}
	if !emptyList(d.CreditSales) {
		t.Errorf("detail credit_sales = %s, want []", d.CreditSales)
	}
	if !emptyList(d.Payments) {
		t.Errorf("detail payments = %s, want []", d.Payments)
	}

	// AC-13: a customer with none gets opening_balance: null (key present).
	w = e.mustStatus(t, http.MethodGet, "/customers/8/detail", e.cashier, "", http.StatusOK)
	var raw map[string]json.RawMessage
	abonoData(t, w, &raw)
	if v, ok := raw["opening_balance"]; !ok || string(v) != "null" {
		t.Errorf("Beto detail opening_balance = %s (present %v), want null", v, ok)
	}
}

// ─── T-4: AC-2, AC-3 ─────────────────────────────────────────────────────────

func TestCustomerOpening_T4_Validation(t *testing.T) {
	e := newObEnv(t)
	d45 := obDate(-45)
	cases := []struct {
		name string
		body string
		msg  string
	}{
		{"amount_0", obBody(0, d45, ""), obAmountMsg},
		{"amount_negative", obBody(-5000, d45, ""), obAmountMsg},
		{"amount_fraction", obBody(1500.5, d45, ""), obAmountMsg},
		{"amount_above_cap", obBody(100000001, d45, ""), obAmountMsg},
		{"date_missing", `{"amount":1000}`, obDateMsg},
		{"date_dd_mm_yyyy", obBody(1000, "25/09/2026", ""), obDateMsg},
		{"date_feb_30", obBody(1000, "2026-02-30", ""), obDateMsg},
		{"date_tomorrow", obBody(1000, obDate(1), ""), obFutureMsg},
	}
	for _, c := range cases {
		w := e.do(t, http.MethodPost, obPath7, e.admin, c.body)
		obExpectError(t, c.name, w, http.StatusBadRequest, c.msg)
	}
	if n := e.openingRows(t); n != 0 {
		t.Errorf("opening rows after rejected requests = %d, want 0", n)
	}
	if n := e.auditRows(t, obCreateAction); n != 0 {
		t.Errorf("audit rows after rejected requests = %d, want 0", n)
	}

	// Boundaries, each on its own customer.
	abonoExec(t, e.db, `INSERT INTO customers (id, name, phone, address) VALUES (10, 'Fito', '', '')`)
	boundaries := []struct {
		name string
		path string
		body string
	}{
		{"amount_1", obPath7, obBody(1, d45, "")},
		{"amount_cap", obPath8, obBody(100000000, d45, "")},
		{"date_today", "/customers/10/opening-balance", obBody(1000, obDate(0), "")},
	}
	for _, b := range boundaries {
		w := e.do(t, http.MethodPost, b.path, e.admin, b.body)
		if w.Code != http.StatusCreated {
			t.Errorf("%s: POST %s %s = %d, want 201; body %s", b.name, b.path, b.body, w.Code, w.Body.String())
		}
	}
	if n := e.openingRows(t); n != 3 {
		t.Errorf("opening rows after boundaries = %d, want 3", n)
	}
}

// ─── T-5: AC-4 ───────────────────────────────────────────────────────────────

func TestCustomerOpening_T5_OnePerCustomer(t *testing.T) {
	t.Run("sequential", func(t *testing.T) {
		e := newObEnv(t)
		d45 := obDate(-45)
		e.createDora(t, 150000, d45)

		w := e.do(t, http.MethodPost, obPath7, e.admin, obBody(999, d45, ""))
		obExpectError(t, "second POST", w, http.StatusConflict, obDuplicateMsg)

		if got := e.balance(t, 7); got != 150000 {
			t.Errorf("balance after second POST = %.0f, want 150000", got)
		}
		if n := e.openingRows(t); n != 1 {
			t.Errorf("opening rows = %d, want 1", n)
		}
		if n := e.auditRows(t, obCreateAction); n != 1 {
			t.Errorf("audit rows %s = %d, want 1", obCreateAction, n)
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		e := newObFileEnv(t)
		body := obBody(20000, obDate(-45), "")
		ws := ccConcurrent(t, e.h, e.admin, ccRepeat(10, ccReq{http.MethodPost, obPath8, body}))
		codes, samples := ccCodes(ws, http.StatusCreated)
		if codes[http.StatusCreated] != 1 || codes[http.StatusConflict] != 9 {
			t.Errorf("status codes = %v, want exactly one 201 and nine 409; samples %v", codes, samples)
		}
		if n := e.count(t, `SELECT COUNT(*) FROM customer_opening_balances WHERE customer_id = 8`); n != 1 {
			t.Errorf("opening rows for Beto = %d, want 1", n)
		}
		if n := e.auditRows(t, obCreateAction); n != 1 {
			t.Errorf("audit rows %s = %d, want 1", obCreateAction, n)
		}
	})
}

// ─── T-6: AC-5, AC-7 ─────────────────────────────────────────────────────────

func TestCustomerOpening_T6_Correction(t *testing.T) {
	e := newObEnv(t)
	d45, d50 := obDate(-45), obDate(-50)
	e.createDora(t, 150000, d45)

	w := e.do(t, http.MethodPut, obPath7, e.admin, fmt.Sprintf(`{"amount":120000,"debt_date":%q}`, d50))
	if w.Code != http.StatusOK {
		t.Fatalf("PUT %s = %d, want 200; body %s", obPath7, w.Code, w.Body.String())
	}
	var o obJSON
	abonoData(t, w, &o)
	if o.CustomerID != 7 || o.Amount != 120000 || o.DebtDate != d50 {
		t.Errorf("PUT response = %+v, want customer_id 7, amount 120000, debt_date %s", o, d50)
	}
	if got := e.balance(t, 7); got != 120000 {
		t.Errorf("balance after PUT = %.0f, want 120000", got)
	}
	if s := e.posSummary(t, 7); s.DaysOverdue == nil || *s.DaysOverdue != 50 {
		t.Errorf("pos-summary days_overdue after PUT = %v, want 50", s.DaysOverdue)
	}
	var updatedAt *string
	if err := e.db.DB.QueryRow(`SELECT updated_at FROM customer_opening_balances WHERE customer_id = 7`).Scan(&updatedAt); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if updatedAt == nil || *updatedAt == "" {
		t.Errorf("updated_at after PUT = %v, want set", updatedAt)
	}

	var userID, entityID int64
	var entity, details string
	err := e.db.DB.QueryRow(`SELECT user_id, entity, entity_id, details FROM audit_log WHERE action = ?`, obUpdateAction).
		Scan(&userID, &entity, &entityID, &details)
	if err != nil {
		t.Fatalf("read %s audit row: %v", obUpdateAction, err)
	}
	if userID != 1 || entity != "customer" || entityID != 7 {
		t.Errorf("update audit row = user %d, entity %q #%d, want user 1, customer #7", userID, entity, entityID)
	}
	var det map[string]any
	if err := json.Unmarshal([]byte(details), &det); err != nil {
		t.Fatalf("decode update details %q: %v", details, err)
	}
	wantDet := map[string]any{"from_amount": 150000.0, "to_amount": 120000.0, "from_date": d45, "to_date": d50}
	if len(det) != len(wantDet) {
		t.Errorf("update details = %v, want exactly %v", det, wantDet)
	}
	for k, v := range wantDet {
		if det[k] != v {
			t.Errorf("update details[%s] = %v, want %v (details %s)", k, det[k], v, details)
		}
	}

	// Beto has none → 404.
	w = e.do(t, http.MethodPut, obPath8, e.admin, obBody(5000, d45, ""))
	obExpectError(t, "PUT Beto", w, http.StatusNotFound, obNoneMsg)

	// Amount 0 → 400, row unchanged, no extra audit row.
	w = e.do(t, http.MethodPut, obPath7, e.admin, obBody(0, d45, ""))
	obExpectError(t, "PUT amount 0", w, http.StatusBadRequest, obAmountMsg)
	var ob obJSON
	if err := json.Unmarshal(e.detail(t, 7).OpeningBalance, &ob); err != nil {
		t.Fatalf("decode opening_balance: %v", err)
	}
	if ob.Amount != 120000 || ob.DebtDate != d50 {
		t.Errorf("opening after rejected PUT = %+v, want amount 120000, debt_date %s", ob, d50)
	}
	if n := e.auditRows(t, obUpdateAction); n != 1 {
		t.Errorf("audit rows %s = %d, want 1", obUpdateAction, n)
	}
}

// ─── T-7: AC-6 ───────────────────────────────────────────────────────────────

func TestCustomerOpening_T7_RoleIDAndCustomerChecks(t *testing.T) {
	e := newObEnv(t)
	body := obBody(150000, obDate(-45), "")

	for _, m := range []string{http.MethodPost, http.MethodPut} {
		w := e.do(t, m, obPath7, e.cashier, body)
		if w.Code != http.StatusForbidden {
			t.Errorf("cashier %s %s = %d, want 403; body %s", m, obPath7, w.Code, w.Body.String())
		} else if got := obErr(w); got != "forbidden" {
			t.Errorf("cashier %s error = %q, want forbidden", m, got)
		}

		obExpectError(t, m+" abc", e.do(t, m, "/customers/abc/opening-balance", e.admin, body), http.StatusBadRequest, obBadIDMsg)
		obExpectError(t, m+" 999", e.do(t, m, "/customers/999/opening-balance", e.admin, body), http.StatusNotFound, obNotFoundMsg)
		obExpectError(t, m+" inactive Eva", e.do(t, m, "/customers/9/opening-balance", e.admin, body), http.StatusNotFound, obNotFoundMsg)
	}

	if n := e.openingRows(t); n != 0 {
		t.Errorf("opening rows = %d, want 0", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM audit_log`); n != 0 {
		t.Errorf("audit rows = %d, want 0", n)
	}
}

// ─── T-8: AC-7 ───────────────────────────────────────────────────────────────

func TestCustomerOpening_T8_Audit(t *testing.T) {
	t.Run("create_writes_one_row", func(t *testing.T) {
		e := newObEnv(t)
		d45 := obDate(-45)
		e.mustStatus(t, http.MethodPost, obPath7, e.admin, obBody(150000, d45, "Cuaderno pág. 12"), http.StatusCreated)

		if n := e.count(t, `SELECT COUNT(*) FROM audit_log`); n != 1 {
			t.Fatalf("audit_log rows = %d, want exactly 1", n)
		}
		var userID, entityID int64
		var action, entity, details string
		if err := e.db.DB.QueryRow(`SELECT user_id, action, entity, entity_id, details FROM audit_log`).
			Scan(&userID, &action, &entity, &entityID, &details); err != nil {
			t.Fatalf("read audit row: %v", err)
		}
		if userID != 1 || action != obCreateAction || entity != "customer" || entityID != 7 {
			t.Errorf("audit row = user %d, %q, %q #%d, want user 1, %q, customer #7", userID, action, entity, entityID, obCreateAction)
		}
		var det map[string]any
		if err := json.Unmarshal([]byte(details), &det); err != nil {
			t.Fatalf("decode details %q: %v", details, err)
		}
		if len(det) != 2 || det["amount"] != 150000.0 || det["debt_date"] != d45 {
			t.Errorf("details = %s, want {\"amount\":150000,\"debt_date\":%q}", details, d45)
		}

		w := e.mustStatus(t, http.MethodGet, "/audit?action="+obCreateAction, e.admin, "", http.StatusOK)
		var entries []struct {
			UserID   int64  `json:"user_id"`
			Action   string `json:"action"`
			Entity   string `json:"entity"`
			EntityID int64  `json:"entity_id"`
		}
		abonoData(t, w, &entries)
		if len(entries) != 1 || entries[0].Action != obCreateAction || entries[0].EntityID != 7 || entries[0].UserID != 1 || entries[0].Entity != "customer" {
			t.Errorf("GET /audit?action=%s = %+v, want one row for customer 7 by user 1", obCreateAction, entries)
		}
	})

	t.Run("audit_failure_rolls_back", func(t *testing.T) {
		e := newObEnv(t)
		abonoExec(t, e.db, `ALTER TABLE audit_log RENAME TO audit_log_x`)

		w := e.do(t, http.MethodPost, obPath7, e.admin, obBody(150000, obDate(-45), ""))
		obExpectError(t, "POST with audit_log missing", w, http.StatusInternalServerError, obInternalMsg)
		if n := e.openingRows(t); n != 0 {
			t.Errorf("opening rows after failed audit = %d, want 0 (rolled back)", n)
		}
	})
}

// ─── T-9: AC-10 ──────────────────────────────────────────────────────────────

func TestCustomerOpening_T9_AbonosPayItDown(t *testing.T) {
	e := newObEnv(t)
	e.createDora(t, 150000, obDate(-45))
	e.mustStatus(t, http.MethodPost, "/registers", e.cashier, `{"opening_amount":50000}`, http.StatusCreated)

	pay := func(amount float64, method string) *httptest.ResponseRecorder {
		return e.do(t, http.MethodPost, "/customers/7/payments", e.cashier, fmt.Sprintf(`{"amount":%.0f,"method":%q}`, amount, method))
	}

	if w := pay(100000, "cash"); w.Code != http.StatusCreated {
		t.Fatalf("cash abono 100000 = %d, want 201; body %s", w.Code, w.Body.String())
	}
	if got := e.balance(t, 7); got != 50000 {
		t.Errorf("balance after abono 100000 = %.0f, want 50000", got)
	}

	obExpectError(t, "abono 60000", pay(60000, "cash"), http.StatusBadRequest, "el pago ($60000) supera la deuda actual ($50000)")

	w := e.do(t, http.MethodDelete, "/customers/7", e.admin, "")
	if w.Code != http.StatusConflict {
		t.Errorf("DELETE /customers/7 with balance 50000 = %d, want 409; body %s", w.Code, w.Body.String())
	}

	if w := pay(50000, "transfer"); w.Code != http.StatusCreated {
		t.Fatalf("transfer abono 50000 = %d, want 201; body %s", w.Code, w.Body.String())
	}
	if got := e.balance(t, 7); got != 0 {
		t.Errorf("balance after last abono = %.0f, want 0", got)
	}
}

// ─── T-10, T-11: AC-12 ───────────────────────────────────────────────────────

type obSnapshot struct {
	report    map[string]json.RawMessage
	today     map[string]json.RawMessage
	dashboard map[string]json.RawMessage
	register  map[string]json.RawMessage
}

func (e *obEnv) dataMap(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	w := e.mustStatus(t, http.MethodGet, path, e.admin, "", http.StatusOK)
	var m map[string]json.RawMessage
	abonoData(t, w, &m)
	return m
}

// obTodayReportPath is the "today" report window of the given clock:
// [local midnight, next local midnight) in the clock's own zone, as RFC3339.
func obTodayReportPath(now time.Time) string {
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 0, 1)
	return "/reports/sales?from=" + url.QueryEscape(from.Format(time.RFC3339)) + "&to=" + url.QueryEscape(to.Format(time.RFC3339))
}

func (e *obEnv) snapshot(t *testing.T) obSnapshot {
	t.Helper()
	return obSnapshot{
		report:    e.dataMap(t, obTodayReportPath(e.clock)),
		today:     e.dataMap(t, "/dashboard/today"),
		dashboard: e.dataMap(t, "/dashboard"),
		register:  e.dataMap(t, "/registers/1"),
	}
}

func obNum(t *testing.T, where string, m map[string]json.RawMessage, key string) float64 {
	t.Helper()
	raw, ok := m[key]
	if !ok {
		t.Errorf("%s: key %q missing", where, key)
		return -1
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Errorf("%s: %s = %s, not a number", where, key, raw)
		return -1
	}
	return v
}

func obMethods(t *testing.T, where string, raw json.RawMessage) []shiftMethodJSON {
	t.Helper()
	var ms []shiftMethodJSON
	if err := json.Unmarshal(raw, &ms); err != nil {
		t.Errorf("%s: by_payment_method = %s: %v", where, raw, err)
		return nil
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].Method < ms[j].Method })
	return ms
}

func obSameKeys(t *testing.T, where string, before, after map[string]json.RawMessage, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if string(before[k]) != string(after[k]) {
			t.Errorf("%s.%s changed: before %s, after %s", where, k, before[k], after[k])
		}
	}
}

func (e *obEnv) tableCounts(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, tbl := range []string{"sales", "sale_payments", "customer_payments", "register_movements"} {
		out[tbl] = e.count(t, `SELECT COUNT(*) FROM `+tbl)
	}
	return out
}

// obBogota is the fixed UTC-5 zone used by the pinned-clock harness.
var obBogota = time.FixedZone("COT", -5*3600)

// obPinClock makes the run independent of the wall clock (US-098 T-12): every
// created_at written through the API on the money tables is forced to pinUTC
// (UTC text, as CURRENT_TIMESTAMP would write it) by AFTER INSERT triggers,
// and the handler clock is set to clock.
func (e *obEnv) obPinClock(t *testing.T, pinUTC string, clock time.Time) {
	t.Helper()
	for _, tbl := range []string{"sales", "sale_payments", "customer_payments", "register_movements", "expenses"} {
		abonoExec(t, e.db, fmt.Sprintf(
			`CREATE TRIGGER pin_%[1]s AFTER INSERT ON %[1]s BEGIN UPDATE %[1]s SET created_at = '%[2]s' WHERE id = NEW.id; END`,
			tbl, pinUTC))
	}
	e.clock = clock
	e.h.now = func() time.Time { return clock }
}

// runNotASale runs T-10 with the default pinned clock and returns the env
// with register 1 closed: rows at 2026-11-10 01:18:41 UTC (20:18:41 Bogota),
// handler clock 2026-11-09 20:30 Bogota.
func runNotASale(t *testing.T) *obEnv {
	t.Helper()
	return runNotASaleAt(t, "2026-11-10 01:18:41", time.Date(2026, 11, 9, 20, 30, 0, 0, obBogota))
}

func runNotASaleAt(t *testing.T, pinUTC string, clock time.Time) *obEnv {
	t.Helper()
	e := newObFileEnv(t)
	e.obPinClock(t, pinUTC, clock)
	abonoExec(t, e.db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (4, 'Pan', 'P', 10000, 6000, 'unit', 1, 10)`)
	abonoExec(t, e.db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (5, 'Queso', 'Q', 40000, 30000, 'unit', 1, 10)`)
	shift := &shiftEnv{db: e.db, h: e.h, admin: e.admin, cashier: e.cashier}

	shift.openRegister(t, 50000)
	shift.postSale(t, 4, 10000, `[{"method":"cash","amount":10000}]`, 0)
	shift.postSale(t, 5, 40000, `[{"method":"credit","amount":40000}]`, 7)

	before := e.snapshot(t)
	countsBefore := e.tableCounts(t)

	e.mustStatus(t, http.MethodPost, obPath8, e.admin, obBody(150000, obDate(-45), ""), http.StatusCreated)
	e.mustStatus(t, http.MethodPut, obPath8, e.admin, fmt.Sprintf(`{"amount":120000,"debt_date":%q}`, obDate(-50)), http.StatusOK)

	after := e.snapshot(t)

	obSameKeys(t, "reports/sales", before.report, after.report,
		"total_revenue", "total_cost", "gross_margin", "total_count", "by_payment_method", "grouped")
	obSameKeys(t, "dashboard/today", before.today, after.today,
		"sold_today", "credit_new_today", "credit_collected_today", "cash_estimated", "gross_profit", "net_profit")
	obSameKeys(t, "dashboard", before.dashboard, after.dashboard, "today", "yesterday", "recent_sales")
	obSameKeys(t, "registers/1", before.register, after.register, "cash_sales", "expected")
	bm, am := obMethods(t, "register before", before.register["by_payment_method"]), obMethods(t, "register after", after.register["by_payment_method"])
	if fmt.Sprint(bm) != fmt.Sprint(am) {
		t.Errorf("register by_payment_method changed: before %+v, after %+v", bm, am)
	}

	// Absolute values after the opening balance was recorded and corrected.
	if v := obNum(t, "reports/sales", after.report, "total_revenue"); v != 50000 {
		t.Errorf("total_revenue = %.0f, want 50000", v)
	}
	if v := obNum(t, "reports/sales", after.report, "total_count"); v != 2 {
		t.Errorf("total_count = %.0f, want 2", v)
	}
	assertMethods(t, "reports/sales", obMethods(t, "reports/sales", after.report["by_payment_method"]),
		[]shiftMethodJSON{{"cash", 10000, 1}, {"credit", 40000, 1}})
	for key, want := range map[string]float64{"sold_today": 50000, "credit_new_today": 40000, "credit_collected_today": 0} {
		if v := obNum(t, "dashboard/today", after.today, key); v != want {
			t.Errorf("dashboard/today %s = %.0f, want %.0f", key, v, want)
		}
	}
	if v := obNum(t, "registers/1", after.register, "cash_sales"); v != 10000 {
		t.Errorf("register cash_sales = %.0f, want 10000", v)
	}
	if v := obNum(t, "registers/1", after.register, "expected"); v != 60000 {
		t.Errorf("register expected = %.0f, want 60000", v)
	}

	countsAfter := e.tableCounts(t)
	for tbl, want := range map[string]int{"sales": 2, "sale_payments": 2, "customer_payments": 0} {
		if countsAfter[tbl] != want {
			t.Errorf("%s rows = %d, want %d", tbl, countsAfter[tbl], want)
		}
	}
	if countsAfter["register_movements"] != countsBefore["register_movements"] {
		t.Errorf("register_movements rows = %d, want unchanged %d", countsAfter["register_movements"], countsBefore["register_movements"])
	}

	s := e.posSummary(t, 8)
	if string(s.LastVisitAt) != "null" {
		t.Errorf("Beto last_visit_at = %s, want null", s.LastVisitAt)
	}
	if string(s.TopProducts) != "[]" {
		t.Errorf("Beto top_products = %s, want []", s.TopProducts)
	}
	if s.Balance != 120000 {
		t.Errorf("Beto pos-summary balance = %.0f, want 120000", s.Balance)
	}

	// Close with actual 60,000 → expected_close 60,000, difference 0.
	shift.expectStatus(t, http.MethodPost, "/registers/1/close", e.cashier, `{"actual_close":60000}`, http.StatusOK)
	r1 := shift.storedRegister(t, 1)
	assertNullFloat(t, "register 1 expected_close", r1.expectedClose, 60000)
	assertNullFloat(t, "register 1 difference", r1.difference, 0)
	return e
}

func TestCustomerOpening_T10_NotASale(t *testing.T) {
	runNotASale(t)
}

// T-12 (US-098 AC-13): the same flow is independent of the wall clock. Pins
// are 20:18:41 (the evening case that broke after 19:00), 18:59:00 (control)
// and 23:59:59 Bogota, each with the handler clock at the same Bogota day.
func TestCustomerOpening_T12_PinnedClockAnyHour(t *testing.T) {
	cases := []struct {
		name  string
		pin   string
		clock time.Time
	}{
		{"20_18_41_bogota", "2026-11-10 01:18:41", time.Date(2026, 11, 9, 20, 30, 0, 0, obBogota)},
		{"18_59_00_bogota_control", "2026-11-09 23:59:00", time.Date(2026, 11, 9, 19, 0, 0, 0, obBogota)},
		{"23_59_59_bogota", "2026-11-10 04:59:59", time.Date(2026, 11, 9, 23, 59, 59, 0, obBogota)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runNotASaleAt(t, tc.pin, tc.clock)
		})
	}
}

func TestCustomerOpening_T11_AbonoAgainstOpeningIsRealMoney(t *testing.T) {
	e := runNotASale(t)
	shift := &shiftEnv{db: e.db, h: e.h, admin: e.admin, cashier: e.cashier}
	shift.openRegister(t, 50000)

	e.mustStatus(t, http.MethodPost, "/customers/8/payments", e.cashier, `{"amount":20000,"method":"cash"}`, http.StatusCreated)

	if v := obNum(t, "dashboard/today", e.dataMap(t, "/dashboard/today"), "credit_collected_today"); v != 20000 {
		t.Errorf("credit_collected_today = %.0f, want 20000", v)
	}
	if got := shift.summary(t, 2).CashAbonos; got != 20000 {
		t.Errorf("register 2 cash_abonos = %.0f, want 20000", got)
	}
	if got := e.balance(t, 8); got != 100000 {
		t.Errorf("Beto balance = %.0f, want 100000 (120000 − 20000)", got)
	}
}
