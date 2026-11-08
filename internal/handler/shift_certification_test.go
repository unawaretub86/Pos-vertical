package handler

// US-091 contract §4, T-1 .. T-4: one scripted shift run end to end through
// h.Routes() with real JWTs, certifying the register close and the customer
// fiado (store credit) balance to the peso.
//
// Fixture (contract §4, D-5): a file-backed DB under t.TempDir(), not
// setupTestDB(t). POST /registers/{id}/close starts sendRegisterSummaryEmail on
// a second pooled connection, and with ":memory:" every pooled connection is a
// separate empty database, so the request after the close could land on one.
//
// Cashier token for every step except voids (admin). Movements share one
// created_at second, so they are compared sorted by id.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

const shiftNeedsRegisterMsg = "Debe haber una caja abierta para registrar un abono"

// shiftSetupDB opens a file-backed DB and seeds cashier 2, products 1 Arroz
// $30,000, 2 Aceite $50,000, 3 Panela $12,000 (stock 10 each, tax_rate 0 by
// default) and customer 1 Dora. Admin id 1 comes from migration 1.
func shiftSetupDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.New(filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatalf("open file db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	abonoExec(t, db, `INSERT INTO users (id, username, password_hash, role) VALUES (2, 'cajero1', 'h', 'cashier')`)
	abonoExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Arroz', 'A', 30000, 20000, 'unit', 1, 10)`)
	abonoExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (2, 'Aceite', 'B', 50000, 35000, 'unit', 1, 10)`)
	abonoExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (3, 'Panela', 'C', 12000, 8000, 'unit', 1, 10)`)
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	return db
}

type shiftEnv struct {
	db      *database.DB
	h       *Handler
	admin   string
	cashier string
}

func newShiftEnv(t *testing.T) *shiftEnv {
	t.Helper()
	db := shiftSetupDB(t)
	return &shiftEnv{
		db:      db,
		h:       New(db),
		admin:   tokenFor(t, abonoAdminID, "admin"),
		cashier: tokenFor(t, abonoCashierID, "cashier"),
	}
}

type shiftMovementJSON struct {
	ID          int64   `json:"id"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type shiftMethodJSON struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
	Count  int     `json:"count"`
}

type shiftSummaryJSON struct {
	Register struct {
		ID            int64    `json:"id"`
		Status        string   `json:"status"`
		ExpectedClose *float64 `json:"expected_close"`
		ActualClose   *float64 `json:"actual_close"`
		Difference    *float64 `json:"difference"`
	} `json:"register"`
	CashSales       float64             `json:"cash_sales"`
	CashAbonos      float64             `json:"cash_abonos"`
	CashIn          float64             `json:"cash_in"`
	CashOut         float64             `json:"cash_out"`
	Expected        float64             `json:"expected"`
	Movements       []shiftMovementJSON `json:"movements"`
	ByPaymentMethod []shiftMethodJSON   `json:"by_payment_method"`
}

type shiftCustomersJSON struct {
	Customers []struct {
		ID           int64           `json:"id"`
		Name         string          `json:"name"`
		Balance      float64         `json:"balance"`
		OldestDebtAt json.RawMessage `json:"oldest_debt_at"`
	} `json:"customers"`
	TotalPortfolio float64 `json:"total_portfolio"`
}

func (e *shiftEnv) expectStatus(t *testing.T, method, path, token, body string, want int) []byte {
	t.Helper()
	w := abonoRequest(t, e.h, method, path, token, body)
	if w.Code != want {
		t.Fatalf("%s %s %s = %d, want %d; body %s", method, path, body, w.Code, want, w.Body.String())
	}
	return w.Body.Bytes()
}

func (e *shiftEnv) openRegister(t *testing.T, opening float64) {
	t.Helper()
	e.expectStatus(t, http.MethodPost, "/registers", e.cashier, fmt.Sprintf(`{"opening_amount":%.0f}`, opening), http.StatusCreated)
}

// postSale sells 1 unit of productID at price with the given payments JSON
// array, as the cashier. customerID 0 omits customer_id.
func (e *shiftEnv) postSale(t *testing.T, productID int64, price float64, payments string, customerID int64) {
	t.Helper()
	customer := ""
	if customerID != 0 {
		customer = fmt.Sprintf(`"customer_id":%d,`, customerID)
	}
	body := fmt.Sprintf(`{%s"items":[{"product_id":%d,"quantity":1,"unit_price":%.0f}],"total":%.0f,"payments":%s}`,
		customer, productID, price, price, payments)
	e.expectStatus(t, http.MethodPost, "/sales", e.cashier, body, http.StatusCreated)
}

func (e *shiftEnv) postAbono(t *testing.T, token string, amount float64, method string) paymentJSON {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodPost, "/customers/1/payments", token, fmt.Sprintf(`{"amount":%.0f,"method":%q}`, amount, method))
	if w.Code != http.StatusCreated {
		t.Fatalf("POST abono %.0f %s = %d, want 201; body %s", amount, method, w.Code, w.Body.String())
	}
	var p paymentJSON
	abonoData(t, w, &p)
	return p
}

func (e *shiftEnv) addMovement(t *testing.T, registerID int64, typ string, amount float64) {
	t.Helper()
	e.expectStatus(t, http.MethodPost, fmt.Sprintf("/registers/%d/movements", registerID), e.cashier,
		fmt.Sprintf(`{"type":%q,"amount":%.0f,"description":"test"}`, typ, amount), http.StatusCreated)
}

func (e *shiftEnv) summary(t *testing.T, registerID int64) shiftSummaryJSON {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodGet, fmt.Sprintf("/registers/%d", registerID), e.cashier, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /registers/%d = %d, want 200; body %s", registerID, w.Code, w.Body.String())
	}
	var s shiftSummaryJSON
	abonoData(t, w, &s)
	sort.Slice(s.Movements, func(i, j int) bool { return s.Movements[i].ID < s.Movements[j].ID })
	return s
}

func (e *shiftEnv) balance(t *testing.T) float64 {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodGet, "/customers/1/balance", e.cashier, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /customers/1/balance = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var b struct {
		Balance *float64 `json:"balance"`
	}
	abonoData(t, w, &b)
	if b.Balance == nil {
		t.Fatalf("GET /customers/1/balance has no balance key; body %s", w.Body.String())
	}
	return *b.Balance
}

func (e *shiftEnv) customers(t *testing.T) shiftCustomersJSON {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodGet, "/customers", e.cashier, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /customers = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var c shiftCustomersJSON
	abonoData(t, w, &c)
	return c
}

func (e *shiftEnv) num(t *testing.T, q string, args ...any) float64 {
	t.Helper()
	var v float64
	if err := e.db.DB.QueryRowContext(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("query: %v\nSQL: %s", err, q)
	}
	return v
}

func (e *shiftEnv) stock(t *testing.T, productID int64) float64 {
	t.Helper()
	return e.num(t, `SELECT stock FROM products WHERE id = ?`, productID)
}

func (e *shiftEnv) paymentCount(t *testing.T) int {
	t.Helper()
	return int(e.num(t, `SELECT COUNT(*) FROM customer_payments WHERE customer_id = 1`))
}

type shiftStoredRegister struct {
	status                                 string
	expectedClose, actualClose, difference sql.NullFloat64
}

func (e *shiftEnv) storedRegister(t *testing.T, id int64) shiftStoredRegister {
	t.Helper()
	var r shiftStoredRegister
	if err := e.db.DB.QueryRowContext(context.Background(),
		`SELECT status, expected_close, actual_close, difference FROM registers WHERE id = ?`, id).
		Scan(&r.status, &r.expectedClose, &r.actualClose, &r.difference); err != nil {
		t.Fatalf("read register %d: %v", id, err)
	}
	return r
}

// runShiftUntilSale3 runs the contract §3 shift up to (not including) the void:
// open $100,000; sale #1 Arroz cash $30,000; sale #2 Aceite credit $50,000 to
// Dora; abonos $20,000 cash, $10,000 transfer, $5,000 card; cash_in $15,000;
// cash_out $8,000; sale #3 Panela cash $12,000.
func (e *shiftEnv) runShiftUntilSale3(t *testing.T) {
	t.Helper()
	e.openRegister(t, 100000)
	e.postSale(t, 1, 30000, `[{"method":"cash","amount":30000}]`, 0)
	e.postSale(t, 2, 50000, `[{"method":"credit","amount":50000}]`, 1)
	e.postAbono(t, e.cashier, 20000, "cash")
	e.postAbono(t, e.cashier, 10000, "transfer")
	e.postAbono(t, e.cashier, 5000, "card")
	e.addMovement(t, 1, "cash_in", 15000)
	e.addMovement(t, 1, "cash_out", 8000)
	e.postSale(t, 3, 12000, `[{"method":"cash","amount":12000}]`, 0)
}

// runShiftWithVoid runs runShiftUntilSale3 and the admin void of sale #3.
func (e *shiftEnv) runShiftWithVoid(t *testing.T) {
	t.Helper()
	e.runShiftUntilSale3(t)
	voidSale(t, e.h, e.admin, 3)
}

func assertMethods(t *testing.T, where string, got []shiftMethodJSON, want []shiftMethodJSON) {
	t.Helper()
	sorted := append([]shiftMethodJSON(nil), got...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Method < sorted[j].Method })
	if len(sorted) != len(want) {
		t.Errorf("%s: by_payment_method = %+v, want exactly %+v", where, got, want)
		return
	}
	for i := range want {
		if sorted[i] != want[i] {
			t.Errorf("%s: by_payment_method = %+v, want exactly %+v", where, got, want)
			return
		}
	}
}

func assertNullFloat(t *testing.T, where string, got sql.NullFloat64, want float64) {
	t.Helper()
	if !got.Valid || got.Float64 != want {
		t.Errorf("%s = %v (valid %v), want %.0f", where, got.Float64, got.Valid, want)
	}
}

// ─── T-1: AC-1, AC-3, AC-4 ───────────────────────────────────────────────────

func TestShiftCert_T1_FullShift(t *testing.T) {
	e := newShiftEnv(t)
	e.runShiftUntilSale3(t)

	// Before the void: 100,000 + 42,000 + 20,000 + 15,000 − 8,000 = 169,000.
	before := e.summary(t, 1)
	if before.Expected != 169000 {
		t.Errorf("expected before void = %.0f, want 169000", before.Expected)
	}
	if before.CashSales != 42000 {
		t.Errorf("cash_sales before void = %.0f, want 42000", before.CashSales)
	}
	assertMethods(t, "before void", before.ByPaymentMethod, []shiftMethodJSON{
		{"cash", 42000, 2}, {"credit", 50000, 1},
	})
	if got := e.stock(t, 3); got != 9 {
		t.Errorf("Panela stock after sale #3 = %.0f, want 9", got)
	}
	if len(before.Movements) != 6 {
		t.Errorf("movements before void = %d, want 6", len(before.Movements))
	}

	// Cashier void → 403 and nothing changes.
	w := abonoRequest(t, e.h, http.MethodDelete, "/sales/3/void", e.cashier, "")
	if w.Code != http.StatusForbidden {
		t.Errorf("cashier DELETE /sales/3/void = %d, want 403; body %s", w.Code, w.Body.String())
	}
	if got := e.summary(t, 1).Expected; got != 169000 {
		t.Errorf("expected after cashier void attempt = %.0f, want 169000", got)
	}
	if got := e.stock(t, 3); got != 9 {
		t.Errorf("Panela stock after cashier void attempt = %.0f, want 9", got)
	}

	// Admin void → 200.
	voidSale(t, e.h, e.admin, 3)

	s := e.summary(t, 1)
	if s.CashSales != 30000 {
		t.Errorf("cash_sales = %.0f, want 30000", s.CashSales)
	}
	if s.CashAbonos != 20000 {
		t.Errorf("cash_abonos = %.0f, want 20000", s.CashAbonos)
	}
	if s.CashIn != 15000 {
		t.Errorf("cash_in = %.0f, want 15000", s.CashIn)
	}
	if s.CashOut != 8000 {
		t.Errorf("cash_out = %.0f, want 8000", s.CashOut)
	}
	// 100,000 + 30,000 + 20,000 + 15,000 − 8,000 = 157,000.
	if s.Expected != 157000 {
		t.Errorf("expected = %.0f, want 157000", s.Expected)
	}
	assertMethods(t, "after void", s.ByPaymentMethod, []shiftMethodJSON{
		{"cash", 30000, 1}, {"credit", 50000, 1},
	})

	wantMovements := []struct {
		typ    string
		amount float64
		desc   string // "" = not asserted
	}{
		{"opening", 100000, ""},
		{"sale", 30000, "Venta #1"},
		{"sale", 50000, "Crédito #2"},
		{"cash_in", 15000, ""},
		{"cash_out", 8000, ""},
		{"sale", 12000, "Venta #3"},
	}
	if len(s.Movements) != len(wantMovements) {
		t.Errorf("movements after void = %d (%+v), want %d", len(s.Movements), s.Movements, len(wantMovements))
	} else {
		for i, want := range wantMovements {
			got := s.Movements[i]
			if got.Type != want.typ || got.Amount != want.amount || (want.desc != "" && got.Description != want.desc) {
				t.Errorf("movement %d by id = %s %.0f %q, want %s %.0f %q", i+1, got.Type, got.Amount, got.Description, want.typ, want.amount, want.desc)
			}
		}
	}
	if n := voidCountMovements(t, e.db, 1); n != 6 {
		t.Errorf("stored movements of register 1 = %d, want 6", n)
	}

	for _, p := range []struct {
		id    int64
		name  string
		stock float64
	}{{1, "Arroz", 9}, {2, "Aceite", 9}, {3, "Panela", 10}} {
		if got := e.stock(t, p.id); got != p.stock {
			t.Errorf("%s stock = %.0f, want %.0f", p.name, got, p.stock)
		}
	}

	// Balance = 50,000 − 20,000 − 10,000 − 5,000 = 15,000.
	if got := e.balance(t); got != 15000 {
		t.Errorf("GET /customers/1/balance = %.0f, want 15000", got)
	}
	c := e.customers(t)
	if len(c.Customers) != 1 || c.Customers[0].Name != "Dora" || c.Customers[0].Balance != 15000 {
		t.Errorf("GET /customers rows = %+v, want one row Dora 15000", c.Customers)
	}
	if c.TotalPortfolio != 15000 {
		t.Errorf("total_portfolio = %.0f, want 15000", c.TotalPortfolio)
	}

	rows, err := e.db.DB.QueryContext(context.Background(),
		`SELECT amount, method, register_id FROM customer_payments WHERE customer_id = 1 ORDER BY id`)
	if err != nil {
		t.Fatalf("query customer_payments: %v", err)
	}
	defer rows.Close()
	type payRow struct {
		amount float64
		method string
		reg    sql.NullInt64
	}
	var got []payRow
	for rows.Next() {
		var r payRow
		if err := rows.Scan(&r.amount, &r.method, &r.reg); err != nil {
			t.Fatalf("scan customer_payments: %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate customer_payments: %v", err)
	}
	want := []payRow{
		{20000, "cash", sql.NullInt64{Int64: 1, Valid: true}},
		{10000, "transfer", sql.NullInt64{Int64: 1, Valid: true}},
		{5000, "card", sql.NullInt64{Int64: 1, Valid: true}},
	}
	if len(got) != len(want) {
		t.Fatalf("customer_payments of customer 1 = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("customer_payments row %d = %+v, want %+v", i+1, got[i], want[i])
		}
	}
}

// ─── T-2: AC-2, AC-6 ─────────────────────────────────────────────────────────

func TestShiftCert_T2_CloseAndAfterClose(t *testing.T) {
	e := newShiftEnv(t)
	e.runShiftWithVoid(t)

	// Cashier closes with $157,000 counted.
	body := e.expectStatus(t, http.MethodPost, "/registers/1/close", e.cashier, `{"actual_close":157000}`, http.StatusOK)
	var closed shiftSummaryJSON
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode close envelope: %v; body %s", err, body)
	}
	if err := json.Unmarshal(env.Data, &closed); err != nil {
		t.Fatalf("decode close data: %v; body %s", err, body)
	}
	if closed.Expected != 157000 {
		t.Errorf("close response expected = %.0f, want 157000", closed.Expected)
	}

	r1 := e.storedRegister(t, 1)
	if r1.status != "closed" {
		t.Errorf("register 1 status = %q, want closed", r1.status)
	}
	assertNullFloat(t, "register 1 expected_close", r1.expectedClose, 157000)
	assertNullFloat(t, "register 1 actual_close", r1.actualClose, 157000)
	assertNullFloat(t, "register 1 difference", r1.difference, 0)

	// Cashier abono with no open register → 403, nothing stored.
	w := abonoRequest(t, e.h, http.MethodPost, "/customers/1/payments", e.cashier, `{"amount":1000,"method":"cash"}`)
	if w.Code != http.StatusForbidden {
		t.Errorf("cashier abono after close = %d, want 403; body %s", w.Code, w.Body.String())
	} else if msg := abonoError(t, w); msg != shiftNeedsRegisterMsg {
		t.Errorf("cashier abono after close error = %q, want %q", msg, shiftNeedsRegisterMsg)
	}
	if n := e.paymentCount(t); n != 3 {
		t.Errorf("customer_payments count after cashier attempt = %d, want 3", n)
	}
	if got := e.balance(t); got != 15000 {
		t.Errorf("balance after cashier attempt = %.0f, want 15000", got)
	}

	// Admin abono $1,000 cash with no open register → 201, register_id null.
	p := e.postAbono(t, e.admin, 1000, "cash")
	assertPayment(t, "admin abono after close", p, 1000, "cash", "null")
	if got := e.balance(t); got != 14000 {
		t.Errorf("balance after admin abono = %.0f, want 14000", got)
	}
	if n := e.paymentCount(t); n != 4 {
		t.Errorf("customer_payments count after admin abono = %d, want 4", n)
	}
	r1 = e.storedRegister(t, 1)
	assertNullFloat(t, "register 1 expected_close after admin abono", r1.expectedClose, 157000)
	assertNullFloat(t, "register 1 difference after admin abono", r1.difference, 0)

	// A second close of register 1 → 409.
	w = abonoRequest(t, e.h, http.MethodPost, "/registers/1/close", e.cashier, `{"actual_close":157000}`)
	if w.Code != http.StatusConflict {
		t.Errorf("second close of register 1 = %d, want 409; body %s", w.Code, w.Body.String())
	}
}

// ─── T-3: AC-5 ───────────────────────────────────────────────────────────────

func TestShiftCert_T3_CreditVoidLowersBalance(t *testing.T) {
	t.Run("credit_no_abonos", func(t *testing.T) {
		e := newShiftEnv(t)
		e.openRegister(t, 100000)
		e.postSale(t, 2, 50000, `[{"method":"credit","amount":50000}]`, 1)
		if got := e.balance(t); got != 50000 {
			t.Fatalf("balance before void = %.0f, want 50000", got)
		}
		if got := e.stock(t, 2); got != 9 {
			t.Errorf("Aceite stock before void = %.0f, want 9", got)
		}
		if got := e.summary(t, 1).Expected; got != 100000 {
			t.Errorf("expected before void = %.0f, want 100000", got)
		}

		voidSale(t, e.h, e.admin, 1)

		if got := e.balance(t); got != 0 {
			t.Errorf("balance after void = %.0f, want 0", got)
		}
		if got := e.stock(t, 2); got != 10 {
			t.Errorf("Aceite stock after void = %.0f, want 10", got)
		}
		if got := e.summary(t, 1).Expected; got != 100000 {
			t.Errorf("expected after void = %.0f, want 100000", got)
		}
		c := e.customers(t)
		if len(c.Customers) != 1 {
			t.Fatalf("GET /customers rows = %d, want 1", len(c.Customers))
		}
		if od := string(c.Customers[0].OldestDebtAt); od != "null" {
			t.Errorf("Dora oldest_debt_at after void = %s, want null", od)
		}
		w := abonoRequest(t, e.h, http.MethodGet, "/customers/1/detail", e.cashier, "")
		if w.Code != http.StatusOK {
			t.Fatalf("GET /customers/1/detail = %d, want 200; body %s", w.Code, w.Body.String())
		}
		var detail struct {
			CreditSales json.RawMessage `json:"credit_sales"`
		}
		abonoData(t, w, &detail)
		if cs := string(detail.CreditSales); cs != "null" && cs != "[]" {
			t.Errorf("detail credit_sales after void = %s, want null or []", cs)
		}
	})

	t.Run("credit_after_abonos", func(t *testing.T) {
		e := newShiftEnv(t)
		e.openRegister(t, 100000)
		e.postSale(t, 2, 50000, `[{"method":"credit","amount":50000}]`, 1)
		e.postAbono(t, e.cashier, 20000, "cash")
		e.postAbono(t, e.cashier, 10000, "transfer")
		e.postAbono(t, e.cashier, 5000, "card")
		if got := e.balance(t); got != 15000 {
			t.Fatalf("balance before void = %.0f, want 15000", got)
		}
		if got := e.summary(t, 1).Expected; got != 120000 {
			t.Errorf("expected before void = %.0f, want 120000", got)
		}

		voidSale(t, e.h, e.admin, 1)

		// 0 completed credit − 35,000 abonos = −35,000.
		if got := e.balance(t); got != -35000 {
			t.Errorf("balance after void = %.0f, want -35000", got)
		}
		if got := e.customers(t).TotalPortfolio; got != 0 {
			t.Errorf("total_portfolio after void = %.0f, want 0", got)
		}
		if got := e.stock(t, 2); got != 10 {
			t.Errorf("Aceite stock after void = %.0f, want 10", got)
		}
		if got := e.summary(t, 1).Expected; got != 120000 {
			t.Errorf("expected after void = %.0f, want 120000", got)
		}
	})

	t.Run("mixed_cash_credit", func(t *testing.T) {
		e := newShiftEnv(t)
		e.openRegister(t, 100000)
		e.postSale(t, 2, 50000, `[{"method":"cash","amount":30000},{"method":"credit","amount":20000}]`, 1)
		if got := e.balance(t); got != 20000 {
			t.Fatalf("balance before void = %.0f, want 20000", got)
		}
		if got := e.summary(t, 1).Expected; got != 130000 {
			t.Errorf("expected before void = %.0f, want 130000", got)
		}

		voidSale(t, e.h, e.admin, 1)

		if got := e.balance(t); got != 0 {
			t.Errorf("balance after void = %.0f, want 0", got)
		}
		if got := e.summary(t, 1).Expected; got != 100000 {
			t.Errorf("expected after void = %.0f, want 100000", got)
		}
		if got := e.stock(t, 2); got != 10 {
			t.Errorf("Aceite stock after void = %.0f, want 10", got)
		}
	})
}

// ─── T-4: AC-5 ───────────────────────────────────────────────────────────────

func TestShiftCert_T4_CreditVoidOnClosedRegister(t *testing.T) {
	e := newShiftEnv(t)
	e.runShiftWithVoid(t)
	e.expectStatus(t, http.MethodPost, "/registers/1/close", e.cashier, `{"actual_close":157000}`, http.StatusOK)
	e.postAbono(t, e.admin, 1000, "cash")
	if got := e.balance(t); got != 14000 {
		t.Fatalf("balance before credit void = %.0f, want 14000", got)
	}

	e.openRegister(t, 50000)
	voidSale(t, e.h, e.admin, 2)

	// 0 completed credit − (20,000 + 10,000 + 5,000 + 1,000) = −36,000.
	if got := e.balance(t); got != -36000 {
		t.Errorf("balance after credit void = %.0f, want -36000", got)
	}
	if got := e.stock(t, 2); got != 10 {
		t.Errorf("Aceite stock after credit void = %.0f, want 10", got)
	}
	s2 := e.summary(t, 2)
	if s2.Expected != 50000 {
		t.Errorf("register 2 expected = %.0f, want 50000", s2.Expected)
	}
	if len(s2.Movements) != 1 {
		t.Errorf("register 2 movements = %d (%+v), want 1", len(s2.Movements), s2.Movements)
	}
	r1 := e.storedRegister(t, 1)
	assertNullFloat(t, "register 1 expected_close after credit void", r1.expectedClose, 157000)
}
