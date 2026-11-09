package handler

// US-096 contract §4, T-0 .. T-3: property-based money invariants.
//
// A moneyEnv drives the real router (h.Routes(), real JWTs) on a file-backed
// DB under a temp dir (contract D-4: ":memory:" gives every pooled connection
// its own empty database). Next to it runs an independent in-test model of the
// money: registers, sales, payments, abonos, movements, expenses and stock.
// Every action computes the status it expects from the model, sends the
// request, and on success updates the model; check() compares the system with
// the model (AC-2 .. AC-7). Rejected steps are compared against a snapshot of
// the tables taken before the request (AC-6).
//
// Actions whose precondition does not hold return early (a no-op step). They
// never call rapid's skip: the pos-guard hook blocks it (contract §2).

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"pgregory.net/rapid"
)

const (
	moneyTol = 0.01 // one centavo (AC-1..AC-7 "Tolerance")
	stockTol = 1e-9

	msgNoOpenRegister  = "no hay caja abierta"
	msgRegisterClosed  = "la caja ya está cerrada"
	msgCloseError      = "error cerrando caja"
	msgAlreadyVoided   = "la venta ya fue anulada"
	moneyCashierUserID = 2
)

// moneyTB is what both *testing.T and *rapid.T offer.
type moneyTB interface {
	Helper()
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Cleanup(func())
}

type moneyProduct struct {
	id      int64
	name    string
	typ     string // product | service
	unit    string // unit | kg
	price   float64
	taxRate float64
	stock   float64
}

// moneyFullCatalog is the AC-1 catalog.
func moneyFullCatalog() []moneyProduct {
	return []moneyProduct{
		{1, "Arroz", "product", "unit", 2500, 0, 500},
		{2, "Gaseosa", "product", "unit", 1001, 0.19, 500},
		{3, "Leche", "product", "unit", 4200, 0.05, 500},
		{4, "Queso", "product", "kg", 9999, 0, 200},
		{5, "Domicilio", "service", "unit", 3000, 0, 0},
	}
}

// moneyZeroTaxCatalog is the T-2 sub-domain: 0% tax, unit products, whole-peso prices.
func moneyZeroTaxCatalog() []moneyProduct {
	return []moneyProduct{
		{1, "Arroz", "product", "unit", 2500, 0, 500},
		{2, "Panela", "product", "unit", 3000, 0, 500},
		{3, "Aceite", "product", "unit", 12500, 0, 500},
	}
}

var moneyKgQuantities = []float64{0.1, 0.125, 0.25, 0.5, 1.5}

// ---- model ----

type moneyLine struct {
	productID int64
	qty       float64
}

type moneyPay struct {
	method string
	amount float64
}

type mSale struct {
	id         int64
	registerID int64
	customerID int64 // 0 = none
	lines      []moneyLine
	pays       []moneyPay
	voided     bool
}

type mReg struct {
	id            int64
	opening       float64
	open          bool
	expectedClose float64
	actualClose   float64
}

type mAbono struct {
	customerID int64
	registerID int64 // 0 = no open register (admin abono)
	method     string
	amount     float64
}

type mMove struct {
	registerID int64
	typ        string // cash_in | cash_out
	amount     float64
}

type mCustomer struct {
	id           int64
	name         string
	voidedCredit bool
}

type moneyEnv struct {
	db        *database.DB
	router    http.Handler
	admin     string
	cashier   string
	wholePeso bool // AC-7 / AC-11: every stored money column must be an integer

	products  []*moneyProduct
	customers []*mCustomer
	registers []*mReg
	sales     []*mSale
	abonos    []mAbono
	moves     []mMove
	expenses  int
}

// moneyTemplate migrates one empty file-backed DB and closes it. Sequences
// copy it instead of migrating from scratch, which saves about 50 ms each and
// keeps both properties inside the AC-8 budget. The copy is still a fresh
// file-backed DB seeded only by the literal INSERTs of newMoneyEnv.
func moneyTemplate(tb moneyTB, dir string) string {
	tb.Helper()
	path := filepath.Join(dir, "template.db")
	db, err := database.New(path)
	if err != nil {
		tb.Fatalf("open template db: %v", err)
	}
	if err := db.Close(); err != nil {
		tb.Fatalf("close template db: %v", err)
	}
	return path
}

// newMoneyEnv opens a fresh file-backed DB in dir (a copy of template when it
// is not empty), seeds it with literal INSERTs (admin id 1 from migration 1,
// cashier id 2, Dora and Beto, the given catalog) and builds the router once.
func newMoneyEnv(tb moneyTB, dir, template string, catalog []moneyProduct) *moneyEnv {
	tb.Helper()
	path := filepath.Join(dir, "money.db")
	if template != "" {
		raw, err := os.ReadFile(template)
		if err != nil {
			tb.Fatalf("read template db: %v", err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			tb.Fatalf("copy template db: %v", err)
		}
	}
	db, err := database.New(path)
	if err != nil {
		tb.Fatalf("open file db: %v", err)
	}
	tb.Cleanup(func() { db.Close() })
	e := &moneyEnv{db: db}
	// One transaction for the whole seed: one commit (one fsync) instead of eight.
	tx, err := db.DB.BeginTx(context.Background(), nil)
	if err != nil {
		tb.Fatalf("seed begin: %v", err)
	}
	seed := func(q string, args ...any) {
		if _, err := tx.ExecContext(context.Background(), q, args...); err != nil {
			tx.Rollback() //nolint:errcheck
			tb.Fatalf("seed exec: %v; SQL: %s", err, q)
		}
	}
	seed(`INSERT INTO users (id, username, password_hash, role) VALUES (2, 'cajero1', 'h', 'cashier')`)
	seed(`INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	seed(`INSERT INTO customers (id, name, phone, address) VALUES (2, 'Beto', '', '')`)
	e.customers = []*mCustomer{{id: 1, name: "Dora"}, {id: 2, name: "Beto"}}
	for _, p := range catalog {
		p := p
		seed(`INSERT INTO products (id, name, code, type, sale_price, cost, unit, stock, tax_rate, active) VALUES (?, ?, ?, ?, ?, 0, ?, ?, ?, 1)`,
			p.id, p.name, fmt.Sprintf("P%d", p.id), p.typ, p.price, p.unit, p.stock, p.taxRate)
		e.products = append(e.products, &p)
	}
	if err := tx.Commit(); err != nil {
		tb.Fatalf("seed commit: %v", err)
	}
	e.admin = moneyToken(tb, 1, "admin")
	e.cashier = moneyToken(tb, moneyCashierUserID, "cashier")
	e.router = New(db).Routes()
	return e
}

func moneyToken(tb moneyTB, userID int64, role string) string {
	tb.Helper()
	tok, err := appmw.GenerateToken(&model.Claims{UserID: userID, Username: role, Role: role})
	if err != nil {
		tb.Fatalf("GenerateToken: %v", err)
	}
	return tok
}

func (e *moneyEnv) exec(tb moneyTB, q string, args ...any) {
	tb.Helper()
	if _, err := e.db.DB.ExecContext(context.Background(), q, args...); err != nil {
		tb.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

func (e *moneyEnv) count(tb moneyTB, q string, args ...any) int {
	tb.Helper()
	var n int
	if err := e.db.DB.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		tb.Fatalf("count: %v\nSQL: %s", err, q)
	}
	return n
}

func (e *moneyEnv) float(tb moneyTB, q string, args ...any) float64 {
	tb.Helper()
	var v float64
	if err := e.db.DB.QueryRowContext(context.Background(), q, args...).Scan(&v); err != nil {
		tb.Fatalf("query: %v\nSQL: %s", err, q)
	}
	return v
}

// send performs a mutating request and logs it, so a failing (shrunk) rapid
// run prints every step's request body and status code (AC-8).
func (e *moneyEnv) send(tb moneyTB, method, path, token, body string) (int, []byte) {
	tb.Helper()
	code, resp := e.raw(method, path, token, body)
	short := string(resp)
	if len(short) > 300 {
		short = short[:300] + "..."
	}
	tb.Logf("%s %s %s -> %d %s", method, path, body, code, strings.TrimSpace(short))
	return code, resp
}

func (e *moneyEnv) raw(method, path, token, body string) (int, []byte) {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

// get performs an admin read and decodes its data envelope.
func (e *moneyEnv) get(tb moneyTB, path string, into any) {
	tb.Helper()
	code, body := e.raw(http.MethodGet, path, e.admin, "")
	if code != http.StatusOK {
		tb.Fatalf("GET %s = %d, want 200; body %s", path, code, body)
	}
	moneyData(tb, body, into)
}

func moneyData(tb moneyTB, body []byte, into any) {
	tb.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		tb.Fatalf("decode envelope: %v; body %s", err, body)
	}
	if err := json.Unmarshal(env.Data, into); err != nil {
		tb.Fatalf("decode data: %v; body %s", err, body)
	}
}

func moneyErrMsg(body []byte) string {
	var env struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &env)
	return env.Error
}

func moneyClose(a, b float64) bool { return math.Abs(a-b) <= moneyTol }

func (e *moneyEnv) product(id int64) *moneyProduct {
	for _, p := range e.products {
		if p.id == id {
			return p
		}
	}
	return nil
}

func (e *moneyEnv) openReg() *mReg {
	for _, r := range e.registers {
		if r.open {
			return r
		}
	}
	return nil
}

func (e *moneyEnv) lastReg() *mReg {
	if len(e.registers) == 0 {
		return nil
	}
	return e.registers[len(e.registers)-1]
}

func (e *moneyEnv) reg(id int64) *mReg {
	for _, r := range e.registers {
		if r.id == id {
			return r
		}
	}
	return nil
}

func (e *moneyEnv) sale(id int64) *mSale {
	for _, s := range e.sales {
		if s.id == id {
			return s
		}
	}
	return nil
}

func (e *moneyEnv) modelCashSales(regID int64) float64 {
	var sum float64
	for _, s := range e.sales {
		if s.registerID != regID || s.voided {
			continue
		}
		sum += s.cash()
	}
	return sum
}

func (s *mSale) cash() float64   { return s.methodSum("cash") }
func (s *mSale) credit() float64 { return s.methodSum("credit") }

func (s *mSale) methodSum(method string) float64 {
	var sum float64
	for _, p := range s.pays {
		if p.method == method {
			sum += p.amount
		}
	}
	return sum
}

func (e *moneyEnv) modelCashAbonos(regID int64) float64 {
	var sum float64
	for _, a := range e.abonos {
		if a.registerID == regID && a.method == "cash" {
			sum += a.amount
		}
	}
	return sum
}

func (e *moneyEnv) modelMoves(regID int64, typ string) float64 {
	var sum float64
	for _, m := range e.moves {
		if m.registerID == regID && m.typ == typ {
			sum += m.amount
		}
	}
	return sum
}

func (e *moneyEnv) modelExpected(r *mReg) float64 {
	return r.opening + e.modelCashSales(r.id) + e.modelCashAbonos(r.id) + e.modelMoves(r.id, "cash_in") - e.modelMoves(r.id, "cash_out")
}

func (e *moneyEnv) modelBalance(customerID int64) float64 {
	var credit, abonos float64
	for _, s := range e.sales {
		if s.customerID == customerID && !s.voided {
			credit += s.credit()
		}
	}
	for _, a := range e.abonos {
		if a.customerID == customerID {
			abonos += a.amount
		}
	}
	return credit - abonos
}

// modelMovementRows is the expected register_movements row count: one
// opening per register, one 'sale' row per sale (voided included, a void adds
// none), and every cash_in / cash_out (expenses with an open register too).
func (e *moneyEnv) modelMovementRows() int {
	return len(e.registers) + len(e.sales) + len(e.moves)
}

func (e *moneyEnv) modelSalePaymentRows() int {
	n := 0
	for _, s := range e.sales {
		n += len(s.pays)
	}
	return n
}

// ---- snapshot (AC-6) ----

type moneySnapshot struct {
	values []float64
}

func (e *moneyEnv) snapshot(tb moneyTB) moneySnapshot {
	tb.Helper()
	queries := []string{
		`SELECT COUNT(*) FROM sales`,
		`SELECT COALESCE(SUM(total), 0) FROM sales`,
		`SELECT COUNT(*) FROM sales WHERE status = 'voided'`,
		`SELECT COUNT(*) FROM sale_payments`,
		`SELECT COALESCE(SUM(amount), 0) FROM sale_payments`,
		`SELECT COUNT(*) FROM customer_payments`,
		`SELECT COALESCE(SUM(amount), 0) FROM customer_payments`,
		`SELECT COUNT(*) FROM register_movements`,
		`SELECT COALESCE(SUM(amount), 0) FROM register_movements`,
		`SELECT COUNT(*) FROM registers`,
		`SELECT COUNT(*) FROM registers WHERE status = 'closed'`,
		`SELECT COALESCE(SUM(opening_amount + COALESCE(actual_close, 0) + COALESCE(expected_close, 0)), 0) FROM registers`,
		`SELECT COUNT(*) FROM expenses`,
		`SELECT COALESCE(SUM(amount), 0) FROM expenses`,
		`SELECT COUNT(*) FROM audit_log`,
	}
	s := moneySnapshot{values: e.scalars(tb, queries)}
	stocks := e.stocks(tb)
	for _, p := range e.products {
		s.values = append(s.values, stocks[p.id])
	}
	return s
}

// scalars runs several scalar subqueries as one SELECT: every query costs file
// syscalls on Windows, and check() runs after every step (AC-8 budget).
func (e *moneyEnv) scalars(tb moneyTB, queries []string) []float64 {
	tb.Helper()
	parts := make([]string, len(queries))
	for i, q := range queries {
		parts[i] = "(" + q + ")"
	}
	vals := make([]float64, len(queries))
	dest := make([]any, len(queries))
	for i := range vals {
		dest[i] = &vals[i]
	}
	if err := e.db.DB.QueryRowContext(context.Background(), "SELECT "+strings.Join(parts, ", ")).Scan(dest...); err != nil {
		tb.Fatalf("scalars: %v", err)
	}
	return vals
}

// stocks reads the stock of every product in one query.
func (e *moneyEnv) stocks(tb moneyTB) map[int64]float64 {
	tb.Helper()
	rows, err := e.db.DB.QueryContext(context.Background(), `SELECT id, stock FROM products`)
	if err != nil {
		tb.Fatalf("stocks: %v", err)
	}
	defer rows.Close()
	out := map[int64]float64{}
	for rows.Next() {
		var id int64
		var v float64
		if err := rows.Scan(&id, &v); err != nil {
			tb.Fatalf("stocks scan: %v", err)
		}
		out[id] = v
	}
	if err := rows.Err(); err != nil {
		tb.Fatalf("stocks rows: %v", err)
	}
	return out
}

func (e *moneyEnv) assertUnchanged(tb moneyTB, step string, before moneySnapshot) {
	tb.Helper()
	after := e.snapshot(tb)
	for i := range before.values {
		if before.values[i] != after.values[i] {
			tb.Errorf("AC-6: rejected step %s changed stored data: snapshot[%d] %v -> %v", step, i, before.values[i], after.values[i])
		}
	}
}

// expectRejection checks the status code, the error message (when given) and
// that nothing changed.
func (e *moneyEnv) expectRejection(tb moneyTB, step string, code int, body []byte, wantCode int, wantMsg string, before moneySnapshot) {
	tb.Helper()
	if code != wantCode {
		tb.Fatalf("%s: status %d, want %d; body %s", step, code, wantCode, body)
	}
	if wantMsg != "" && moneyErrMsg(body) != wantMsg {
		tb.Errorf("%s: error %q, want %q", step, moneyErrMsg(body), wantMsg)
	}
	e.assertUnchanged(tb, step, before)
}

// ---- actions with explicit parameters ----

func (e *moneyEnv) doOpen(tb moneyTB, amount float64) int {
	tb.Helper()
	body := fmt.Sprintf(`{"opening_amount":%s}`, moneyNum(amount))
	if e.openReg() != nil {
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, "/registers", e.admin, body)
		e.expectRejection(tb, "second open", code, resp, http.StatusConflict, "", before)
		return code
	}
	code, resp := e.send(tb, http.MethodPost, "/registers", e.admin, body)
	if code != http.StatusCreated {
		tb.Fatalf("open register %v: status %d, want 201; body %s", amount, code, resp)
	}
	var reg model.Register
	moneyData(tb, resp, &reg)
	e.registers = append(e.registers, &mReg{id: reg.ID, opening: amount, open: true})
	return code
}

type moneySaleReq struct {
	CustomerID *int64            `json:"customer_id,omitempty"`
	Items      []moneyItemReq    `json:"items"`
	Discount   float64           `json:"discount"`
	Tax        float64           `json:"tax"`
	Total      *float64          `json:"total,omitempty"`
	Payments   []moneyPayReqJSON `json:"payments"`
}

type moneyItemReq struct {
	ProductID int64   `json:"product_id"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Discount  float64 `json:"discount"`
}

type moneyPayReqJSON struct {
	Method string  `json:"method"`
	Amount float64 `json:"amount"`
}

// clientTotals mirrors frontend/lib/saleCalc.ts calcSubtotal / calcTax with no
// discounts, in the same summation order.
func (e *moneyEnv) clientTotals(lines []moneyLine) (subtotal, tax float64) {
	for _, l := range lines {
		p := e.product(l.productID)
		subtotal += p.price*l.qty - 0
	}
	for _, l := range lines {
		p := e.product(l.productID)
		tax += (p.price*l.qty - 0) * p.taxRate
	}
	return subtotal, tax
}

// doSale posts a sale shaped as the UI sends it (cashier token, product price,
// client tax as calcTax) with the given payment lines.
func (e *moneyEnv) doSale(tb moneyTB, customerID int64, lines []moneyLine, pays []moneyPay) int {
	tb.Helper()
	subtotal, tax := e.clientTotals(lines)
	total := subtotal + tax
	req := moneySaleReq{Tax: tax, Total: &total}
	if customerID != 0 {
		id := customerID
		req.CustomerID = &id
	}
	for _, l := range lines {
		req.Items = append(req.Items, moneyItemReq{ProductID: l.productID, Quantity: l.qty, UnitPrice: e.product(l.productID).price})
	}
	for _, p := range pays {
		req.Payments = append(req.Payments, moneyPayReqJSON{Method: p.method, Amount: p.amount})
	}
	raw, err := json.Marshal(req)
	if err != nil {
		tb.Fatalf("marshal sale: %v", err)
	}
	body := string(raw)

	reg := e.openReg()
	if reg == nil {
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, "/sales", e.cashier, body)
		e.expectRejection(tb, "sale with no open register", code, resp, http.StatusBadRequest, msgNoOpenRegister, before)
		return code
	}
	demand := map[int64]float64{}
	for _, l := range lines {
		if e.product(l.productID).typ == "product" {
			demand[l.productID] += l.qty
		}
	}
	over := false
	for _, p := range e.products {
		if demand[p.id] > p.stock+stockTol {
			over = true
		}
	}
	if over {
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, "/sales", e.cashier, body)
		e.expectRejection(tb, "sale above stock", code, resp, http.StatusBadRequest, "", before)
		if !strings.HasPrefix(moneyErrMsg(resp), "stock insuficiente") {
			tb.Errorf("sale above stock: error %q, want prefix %q", moneyErrMsg(resp), "stock insuficiente")
		}
		return code
	}

	code, resp := e.send(tb, http.MethodPost, "/sales", e.cashier, body)
	if code != http.StatusCreated {
		tb.Fatalf("sale: status %d, want 201; body %s", code, resp)
	}
	var created model.Sale
	moneyData(tb, resp, &created)
	e.sales = append(e.sales, &mSale{id: created.ID, registerID: reg.id, customerID: customerID, lines: lines, pays: pays})
	for _, l := range lines {
		p := e.product(l.productID)
		if p.typ == "product" {
			p.stock -= l.qty
		}
	}
	return code
}

func (e *moneyEnv) doAbono(tb moneyTB, customerID int64, amount float64, method, role string) int {
	tb.Helper()
	token := e.admin
	if role == "cashier" {
		token = e.cashier
	}
	body := fmt.Sprintf(`{"amount":%s,"method":%q}`, moneyNum(amount), method)
	path := fmt.Sprintf("/customers/%d/payments", customerID)
	open := e.openReg()
	balance := e.modelBalance(customerID)
	switch {
	case open == nil && role == "cashier":
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, path, token, body)
		e.expectRejection(tb, "cashier abono with no open register", code, resp, http.StatusForbidden, "", before)
		return code
	case amount > balance+1e-6:
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, path, token, body)
		e.expectRejection(tb, "abono above balance", code, resp, http.StatusBadRequest, "", before)
		return code
	}
	code, resp := e.send(tb, http.MethodPost, path, token, body)
	if code != http.StatusCreated {
		tb.Fatalf("abono %v (balance %v): status %d, want 201; body %s", amount, balance, code, resp)
	}
	var regID int64
	if open != nil {
		regID = open.id
	}
	e.abonos = append(e.abonos, mAbono{customerID: customerID, registerID: regID, method: method, amount: amount})
	return code
}

func (e *moneyEnv) doMovement(tb moneyTB, regID int64, typ string, amount float64) int {
	tb.Helper()
	body := fmt.Sprintf(`{"type":%q,"amount":%s,"description":"prueba"}`, typ, moneyNum(amount))
	path := fmt.Sprintf("/registers/%d/movements", regID)
	r := e.reg(regID)
	if r == nil {
		tb.Fatalf("doMovement on unknown register %d is not generated", regID)
	}
	if !r.open {
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, path, e.cashier, body)
		e.expectRejection(tb, typ+" on a closed register", code, resp, http.StatusConflict, msgRegisterClosed, before)
		return code
	}
	code, resp := e.send(tb, http.MethodPost, path, e.cashier, body)
	if code != http.StatusCreated {
		tb.Fatalf("%s %v: status %d, want 201; body %s", typ, amount, code, resp)
	}
	e.moves = append(e.moves, mMove{registerID: regID, typ: typ, amount: amount})
	return code
}

func (e *moneyEnv) doExpense(tb moneyTB, category string, amount float64) int {
	tb.Helper()
	body := fmt.Sprintf(`{"category":%q,"amount":%s,"note":""}`, category, moneyNum(amount))
	code, resp := e.send(tb, http.MethodPost, "/expenses", e.cashier, body)
	if code != http.StatusCreated {
		tb.Fatalf("expense %v: status %d, want 201; body %s", amount, code, resp)
	}
	e.expenses++
	if r := e.openReg(); r != nil {
		e.moves = append(e.moves, mMove{registerID: r.id, typ: "cash_out", amount: amount})
	}
	return code
}

func (e *moneyEnv) registerExpected(tb moneyTB, regID int64) float64 {
	tb.Helper()
	var s struct {
		Expected float64 `json:"expected"`
	}
	e.get(tb, fmt.Sprintf("/registers/%d", regID), &s)
	return s.Expected
}

func (e *moneyEnv) customerBalance(tb moneyTB, customerID int64) float64 {
	tb.Helper()
	var b struct {
		Balance float64 `json:"balance"`
	}
	e.get(tb, fmt.Sprintf("/customers/%d/balance", customerID), &b)
	return b.Balance
}

func (e *moneyEnv) doVoid(tb moneyTB, saleID int64, role string) int {
	tb.Helper()
	token := e.admin
	if role == "cashier" {
		token = e.cashier
	}
	path := fmt.Sprintf("/sales/%d/void", saleID)
	s := e.sale(saleID)
	if s == nil {
		tb.Fatalf("doVoid on unknown sale %d is not generated", saleID)
	}
	if role == "cashier" {
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodDelete, path, token, "")
		e.expectRejection(tb, "cashier void", code, resp, http.StatusForbidden, "", before)
		return code
	}
	if s.voided {
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodDelete, path, token, "")
		e.expectRejection(tb, "second void", code, resp, http.StatusBadRequest, msgAlreadyVoided, before)
		return code
	}

	// AC-5: values right before the void.
	expBefore := e.registerExpected(tb, s.registerID)
	var balBefore float64
	if s.customerID != 0 {
		balBefore = e.customerBalance(tb, s.customerID)
	}
	stockBefore := e.stocks(tb)
	movesBefore := e.count(tb, `SELECT COUNT(*) FROM register_movements`)

	code, resp := e.send(tb, http.MethodDelete, path, token, "")
	if code != http.StatusOK {
		tb.Fatalf("void sale %d: status %d, want 200; body %s", saleID, code, resp)
	}

	if got, want := expBefore-e.registerExpected(tb, s.registerID), s.cash(); !moneyClose(got, want) {
		tb.Errorf("AC-5(a): void of sale %d lowered expected of register %d by %v, want %v (its cash lines)", saleID, s.registerID, got, want)
	}
	if s.customerID != 0 {
		if got, want := balBefore-e.customerBalance(tb, s.customerID), s.credit(); !moneyClose(got, want) {
			tb.Errorf("AC-5(b): void of sale %d lowered customer %d balance by %v, want %v (its credit lines)", saleID, s.customerID, got, want)
		}
	}
	rise := map[int64]float64{}
	for _, l := range s.lines {
		if e.product(l.productID).typ == "product" {
			rise[l.productID] += l.qty
		}
	}
	stockAfter := e.stocks(tb)
	for _, p := range e.products {
		after := stockAfter[p.id]
		if math.Abs(after-stockBefore[p.id]-rise[p.id]) > stockTol {
			tb.Errorf("AC-5(c): void of sale %d moved %s stock %v -> %v, want +%v", saleID, p.name, stockBefore[p.id], after, rise[p.id])
		}
	}
	if got := e.count(tb, `SELECT COUNT(*) FROM register_movements`); got != movesBefore {
		tb.Errorf("AC-5(d): void of sale %d changed register_movements rows %d -> %d", saleID, movesBefore, got)
	}

	s.voided = true
	for _, l := range s.lines {
		if p := e.product(l.productID); p.typ == "product" {
			p.stock += l.qty
		}
	}
	if s.customerID != 0 && s.credit() > 0 {
		for _, c := range e.customers {
			if c.id == s.customerID {
				c.voidedCredit = true
			}
		}
	}
	return code
}

func (e *moneyEnv) doClose(tb moneyTB, regID int64, actual float64) int {
	tb.Helper()
	body := fmt.Sprintf(`{"actual_close":%s}`, moneyNum(actual))
	path := fmt.Sprintf("/registers/%d/close", regID)
	r := e.reg(regID)
	switch {
	case r == nil:
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, path, e.admin, body)
		e.expectRejection(tb, fmt.Sprintf("close of unknown register %d", regID), code, resp, http.StatusInternalServerError, msgCloseError, before)
		return code
	case !r.open:
		before := e.snapshot(tb)
		code, resp := e.send(tb, http.MethodPost, path, e.admin, body)
		e.expectRejection(tb, "close of a closed register", code, resp, http.StatusConflict, msgRegisterClosed, before)
		return code
	}
	code, resp := e.send(tb, http.MethodPost, path, e.admin, body)
	if code != http.StatusOK {
		tb.Fatalf("close register %d: status %d, want 200; body %s", regID, code, resp)
	}
	r.open = false
	r.expectedClose = e.modelExpected(r)
	r.actualClose = actual
	var summary struct {
		Register model.Register `json:"register"`
	}
	moneyData(tb, resp, &summary)
	if summary.Register.ExpectedClose == nil || !moneyClose(*summary.Register.ExpectedClose, r.expectedClose) {
		tb.Errorf("AC-3: close of register %d returned expected_close %v, model %v", regID, summary.Register.ExpectedClose, r.expectedClose)
	}
	return code
}

// moneyNum formats a float as JSON.
func moneyNum(v float64) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

// ---- invariants ----

// checkStoredRows checks AC-2 and the positivity part of AC-7 on every stored
// row, and, with wholePeso, that every stored money column is an integer.
func (e *moneyEnv) checkStoredRows(tb moneyTB) {
	tb.Helper()
	ctx := context.Background()
	rows, err := e.db.DB.QueryContext(ctx, `
		SELECT s.id, s.total, COALESCE(SUM(sp.amount), 0), COUNT(sp.id)
		FROM sales s LEFT JOIN sale_payments sp ON sp.sale_id = s.id
		GROUP BY s.id ORDER BY s.id`)
	if err != nil {
		tb.Fatalf("AC-2 query: %v", err)
	}
	for rows.Next() {
		var id int64
		var total, paid float64
		var n int
		if err := rows.Scan(&id, &total, &paid, &n); err != nil {
			rows.Close()
			tb.Fatalf("AC-2 scan: %v", err)
		}
		if n == 0 || !(math.Abs(paid-total) <= 0.5) {
			tb.Errorf("AC-2: sale %d has %d payment lines summing %v for total %v (gap %v, max 0.5)", id, n, paid, total, paid-total)
		}
	}
	rows.Close()

	positivity := []struct{ what, q string }{
		{"sale_payments.amount <= 0", `SELECT COUNT(*) FROM sale_payments WHERE NOT (amount > 0)`},
		{"customer_payments.amount <= 0", `SELECT COUNT(*) FROM customer_payments WHERE NOT (amount > 0)`},
		{"cash_in/cash_out amount <= 0", `SELECT COUNT(*) FROM register_movements WHERE type IN ('cash_in', 'cash_out') AND NOT (amount > 0)`},
		{"expenses.amount <= 0", `SELECT COUNT(*) FROM expenses WHERE NOT (amount > 0)`},
		{"registers.opening_amount < 0", `SELECT COUNT(*) FROM registers WHERE NOT (opening_amount >= 0)`},
	}
	pq := make([]string, len(positivity))
	for i, p := range positivity {
		pq[i] = p.q
	}
	pn := e.scalars(tb, pq)
	for i, p := range positivity {
		if n := pn[i]; n != 0 {
			tb.Errorf("AC-7: %v row(s) with %s", n, p.what)
		}
	}

	if e.wholePeso {
		e.checkWholePesos(tb)
	}
}

// checkWholePesos is the whole-peso part of AC-7 (zero-tax sub-domain) and AC-11.
func (e *moneyEnv) checkWholePesos(tb moneyTB) {
	tb.Helper()
	cols := []struct{ table, col string }{
		{"sales", "subtotal"}, {"sales", "discount"}, {"sales", "tax"}, {"sales", "total"},
		{"sale_items", "unit_price"}, {"sale_items", "discount"}, {"sale_items", "subtotal"},
		{"sale_payments", "amount"},
		{"customer_payments", "amount"},
		{"register_movements", "amount"},
		{"registers", "opening_amount"}, {"registers", "expected_close"}, {"registers", "actual_close"}, {"registers", "difference"},
		{"expenses", "amount"},
	}
	// One UNION ALL query for all columns (AC-8 budget); every value is
	// checked in Go.
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = `SELECT '` + c.table + `.` + c.col + `', id, ` + c.col + ` FROM ` + c.table + ` WHERE ` + c.col + ` IS NOT NULL`
	}
	rows, err := e.db.DB.QueryContext(context.Background(), strings.Join(parts, " UNION ALL ")+` ORDER BY 1, 2`)
	if err != nil {
		tb.Fatalf("whole-peso query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var col string
		var id int64
		var v float64
		if err := rows.Scan(&col, &id, &v); err != nil {
			tb.Fatalf("whole-peso scan: %v", err)
		}
		if v != math.Trunc(v) {
			tb.Errorf("whole pesos: %s of row %d = %v is not an integer", col, id, v)
		}
	}
	if err := rows.Err(); err != nil {
		tb.Fatalf("whole-peso rows: %v", err)
	}
}

type moneyRegisterSummary struct {
	Register   model.Register `json:"register"`
	CashSales  float64        `json:"cash_sales"`
	CashAbonos float64        `json:"cash_abonos"`
	CashIn     float64        `json:"cash_in"`
	CashOut    float64        `json:"cash_out"`
	Expected   float64        `json:"expected"`
}

// check is the rapid "" action: every invariant, after every step.
func (e *moneyEnv) check(tb moneyTB) {
	tb.Helper()
	e.checkStoredRows(tb)

	// Row counts against the model (AC-6 on accepted steps too).
	counts := []struct {
		what string
		q    string
		want int
	}{
		{"sales", `SELECT COUNT(*) FROM sales`, len(e.sales)},
		{"sale_payments", `SELECT COUNT(*) FROM sale_payments`, e.modelSalePaymentRows()},
		{"customer_payments", `SELECT COUNT(*) FROM customer_payments`, len(e.abonos)},
		{"register_movements", `SELECT COUNT(*) FROM register_movements`, e.modelMovementRows()},
		{"registers", `SELECT COUNT(*) FROM registers`, len(e.registers)},
		{"expenses", `SELECT COUNT(*) FROM expenses`, e.expenses},
	}
	cq := make([]string, len(counts))
	for i, c := range counts {
		cq[i] = c.q
	}
	cn := e.scalars(tb, cq)
	for i, c := range counts {
		if got := int(cn[i]); got != c.want {
			tb.Errorf("row count %s = %d, model %d", c.what, got, c.want)
		}
	}

	// AC-3: every register against the model.
	for _, r := range e.registers {
		var s moneyRegisterSummary
		e.get(tb, fmt.Sprintf("/registers/%d", r.id), &s)
		fields := []struct {
			name      string
			got, want float64
		}{
			{"cash_sales", s.CashSales, e.modelCashSales(r.id)},
			{"cash_abonos", s.CashAbonos, e.modelCashAbonos(r.id)},
			{"cash_in", s.CashIn, e.modelMoves(r.id, "cash_in")},
			{"cash_out", s.CashOut, e.modelMoves(r.id, "cash_out")},
			{"expected", s.Expected, e.modelExpected(r)},
		}
		for _, f := range fields {
			if !moneyClose(f.got, f.want) {
				tb.Errorf("AC-3: register %d %s = %v, model %v", r.id, f.name, f.got, f.want)
			}
		}
		if r.open {
			if s.Register.Status != "open" {
				tb.Errorf("AC-3: register %d status %q, model open", r.id, s.Register.Status)
			}
			continue
		}
		reg := s.Register
		if reg.Status != "closed" || reg.ExpectedClose == nil || reg.ActualClose == nil || reg.Difference == nil {
			tb.Errorf("AC-3: register %d is closed in the model but reads %+v", r.id, reg)
			continue
		}
		if !moneyClose(*reg.ExpectedClose, r.expectedClose) {
			tb.Errorf("AC-3: register %d expected_close = %v, model %v (frozen at close)", r.id, *reg.ExpectedClose, r.expectedClose)
		}
		if !moneyClose(*reg.ActualClose, r.actualClose) {
			tb.Errorf("AC-3: register %d actual_close = %v, model %v", r.id, *reg.ActualClose, r.actualClose)
		}
		if !moneyClose(*reg.Difference, *reg.ActualClose-*reg.ExpectedClose) {
			tb.Errorf("AC-3: register %d difference = %v, want actual_close - expected_close = %v", r.id, *reg.Difference, *reg.ActualClose-*reg.ExpectedClose)
		}
	}

	// AC-4: three balance endpoints, the model and total_portfolio.
	var list struct {
		Customers []struct {
			ID      int64   `json:"id"`
			Balance float64 `json:"balance"`
		} `json:"customers"`
		TotalPortfolio float64 `json:"total_portfolio"`
	}
	e.get(tb, "/customers", &list)
	var portfolio float64
	for _, c := range e.customers {
		want := e.modelBalance(c.id)
		if want > 0 {
			portfolio += want
		}
		balance := e.customerBalance(tb, c.id)
		listed, found := 0.0, false
		for _, lc := range list.Customers {
			if lc.ID == c.id {
				listed, found = lc.Balance, true
			}
		}
		if !found {
			tb.Errorf("AC-4: customer %d missing from GET /customers", c.id)
		}
		var pos struct {
			Balance float64 `json:"balance"`
		}
		e.get(tb, fmt.Sprintf("/customers/%d/pos-summary", c.id), &pos)
		if !moneyClose(balance, want) || !moneyClose(listed, want) || !moneyClose(pos.Balance, want) {
			tb.Errorf("AC-4: %s balance: /balance %v, /customers %v, pos-summary %v, model %v", c.name, balance, listed, pos.Balance, want)
		}
		if !c.voidedCredit && balance < -1e-6 {
			tb.Errorf("AC-4: %s balance %v < 0 with no voided credit sale", c.name, balance)
		}
	}
	if !moneyClose(list.TotalPortfolio, portfolio) {
		tb.Errorf("AC-4: total_portfolio = %v, sum of positive model balances %v", list.TotalPortfolio, portfolio)
	}

	// Stock of every product.
	stocks := e.stocks(tb)
	for _, p := range e.products {
		got := stocks[p.id]
		if math.Abs(got-p.stock) > stockTol {
			tb.Errorf("stock of %s = %v, model %v", p.name, got, p.stock)
		}
	}
}

// ---- generators ----

var moneyCustomerIDs = []int64{1, 2}

func (e *moneyEnv) drawLines(t *rapid.T) []moneyLine {
	n := rapid.IntRange(1, 3).Draw(t, "items")
	var lines []moneyLine
	for i := 0; i < n; i++ {
		p := rapid.SampledFrom(e.products).Draw(t, "product")
		qty := float64(rapid.IntRange(1, 5).Draw(t, "qty"))
		if p.unit == "kg" {
			qty = rapid.SampledFrom(moneyKgQuantities).Draw(t, "kg")
		}
		lines = append(lines, moneyLine{productID: p.id, qty: qty})
	}
	// Now and then one line asks for more than the stock (AC-6: 400).
	if rapid.IntRange(0, 19).Draw(t, "overStockRoll") == 0 {
		var goods []*moneyProduct
		for _, p := range e.products {
			if p.typ == "product" {
				goods = append(goods, p)
			}
		}
		p := rapid.SampledFrom(goods).Draw(t, "overStockProduct")
		lines = append(lines, moneyLine{productID: p.id, qty: math.Floor(p.stock) + 1})
	}
	return lines
}

// drawPays splits total over the given methods: whole-peso amounts, the last
// line taking the exact remainder, every line > 0.
func drawPays(t *rapid.T, methods []string, total float64) []moneyPay {
	remaining := total
	var pays []moneyPay
	for i, m := range methods {
		if i == len(methods)-1 {
			pays = append(pays, moneyPay{method: m, amount: remaining})
			break
		}
		maxAmount := int(math.Floor(remaining)) - (len(methods) - 1 - i)
		if maxAmount < 1 {
			pays = append(pays, moneyPay{method: methods[len(methods)-1], amount: remaining})
			break
		}
		a := float64(rapid.IntRange(1, maxAmount).Draw(t, "split"))
		pays = append(pays, moneyPay{method: m, amount: a})
		remaining -= a
	}
	return pays
}

func (e *moneyEnv) genSale(t *rapid.T, kind string) {
	lines := e.drawLines(t)
	subtotal, tax := e.clientTotals(lines)
	total := subtotal + tax
	var methods []string
	switch kind {
	case "mixed":
		n := rapid.IntRange(2, 3).Draw(t, "payLines")
		methods = rapid.Permutation([]string{"cash", "transfer", "card", "credit"}).Draw(t, "methods")[:n]
	default:
		methods = []string{kind}
	}
	hasCredit := false
	for _, m := range methods {
		if m == "credit" {
			hasCredit = true
		}
	}
	var customerID int64
	if hasCredit {
		customerID = rapid.SampledFrom(moneyCustomerIDs).Draw(t, "customer")
	} else {
		customerID = int64(rapid.IntRange(0, 2).Draw(t, "customerOrNone"))
	}
	e.doSale(t, customerID, lines, drawPays(t, methods, total))
}

func (e *moneyEnv) actions() map[string]func(*rapid.T) {
	return map[string]func(*rapid.T){
		"openRegister": func(t *rapid.T) {
			e.doOpen(t, float64(rapid.IntRange(0, 200).Draw(t, "openingThousands")*1000))
		},
		"cashSale":     func(t *rapid.T) { e.genSale(t, "cash") },
		"transferSale": func(t *rapid.T) { e.genSale(t, "transfer") },
		"cardSale":     func(t *rapid.T) { e.genSale(t, "card") },
		"creditSale":   func(t *rapid.T) { e.genSale(t, "credit") },
		"mixedSale":    func(t *rapid.T) { e.genSale(t, "mixed") },
		"abono": func(t *rapid.T) {
			customerID := rapid.SampledFrom(moneyCustomerIDs).Draw(t, "customer")
			method := rapid.SampledFrom([]string{"cash", "transfer", "card"}).Draw(t, "method")
			role := rapid.SampledFrom([]string{"cashier", "admin"}).Draw(t, "role")
			balance := e.modelBalance(customerID)
			// Amounts within 1e-6 of the balance are never drawn, so a float
			// difference between the model and SQL cannot flip the outcome.
			var amount float64
			if !rapid.Bool().Draw(t, "aboveBalance") && balance-1e-6 >= 1 {
				amount = float64(rapid.IntRange(1, int(math.Floor(balance-1e-6))).Draw(t, "amount"))
			} else {
				amount = math.Ceil(math.Max(balance, 0)+1e-6) + float64(rapid.IntRange(0, 1000).Draw(t, "excess"))
			}
			e.doAbono(t, customerID, amount, method, role)
		},
		"cashIn": func(t *rapid.T) {
			r := e.lastReg()
			if r == nil {
				return
			}
			e.doMovement(t, r.id, "cash_in", float64(rapid.IntRange(1, 50000).Draw(t, "amount")))
		},
		"cashOut": func(t *rapid.T) {
			r := e.lastReg()
			if r == nil {
				return
			}
			e.doMovement(t, r.id, "cash_out", float64(rapid.IntRange(1, 50000).Draw(t, "amount")))
		},
		"expense": func(t *rapid.T) {
			category := rapid.SampledFrom([]string{"servicios", "arriendo", "nomina", "proveedor", "otro"}).Draw(t, "category")
			e.doExpense(t, category, float64(rapid.IntRange(1, 50000).Draw(t, "amount")))
		},
		"voidSale": func(t *rapid.T) {
			if len(e.sales) == 0 {
				return
			}
			s := e.sales[rapid.IntRange(0, len(e.sales)-1).Draw(t, "sale")]
			role := "admin"
			if rapid.IntRange(0, 4).Draw(t, "cashierRoll") == 0 {
				role = "cashier"
			}
			e.doVoid(t, s.id, role)
		},
		"closeRegister": func(t *rapid.T) {
			kind := rapid.IntRange(0, 5).Draw(t, "closeKind")
			delta := float64(rapid.IntRange(-5000, 5000).Draw(t, "countDelta"))
			open := e.openReg()
			var target int64
			switch {
			case kind <= 3 && open != nil:
				target = open.id
			case kind <= 4 && e.lastClosed() != nil:
				target = e.lastClosed().id
			default:
				target = int64(len(e.registers)) + 1 // unknown id; 1 when no register exists
				if rapid.Bool().Draw(t, "farId") {
					target = 99
				}
			}
			actual := delta
			if r := e.reg(target); r != nil {
				actual = math.Round(e.modelExpected(r)) + delta
			}
			e.doClose(t, target, actual)
		},
		// The whole-peso properties (T-2, T-6) check the stored rows only
		// (AC-2, AC-7); the HTTP invariants AC-3 .. AC-5 are T-1's job. Running
		// them in both properties put T-1 + T-2 above the AC-8 budget of 30 s.
		// The actions still assert every status code against the model.
		"": func(t *rapid.T) {
			if e.wholePeso {
				e.checkStoredRows(t)
				return
			}
			e.check(t)
		},
	}
}

func (e *moneyEnv) lastClosed() *mReg {
	for i := len(e.registers) - 1; i >= 0; i-- {
		if !e.registers[i].open {
			return e.registers[i]
		}
	}
	return nil
}

// runMoneyMachine is one rapid sequence on a fresh DB.
func runMoneyMachine(t *testing.T, catalog []moneyProduct, wholePeso bool) {
	template := moneyTemplate(t, t.TempDir())
	rapid.Check(t, func(rt *rapid.T) {
		e := newMoneyEnv(rt, t.TempDir(), template, catalog)
		e.wholePeso = wholePeso
		if rapid.Bool().Draw(rt, "openAtStart") {
			e.doOpen(rt, float64(rapid.IntRange(0, 200).Draw(rt, "openingThousands")*1000))
		}
		rt.Repeat(e.actions())
	})
}

// T-1 (AC-1 .. AC-8).
func TestMoneyProperties_StateMachine(t *testing.T) {
	runMoneyMachine(t, moneyFullCatalog(), false)
}

// T-2 (AC-7, whole pesos on the zero-tax sub-domain).
func TestMoneyProperties_WholePesoZeroTax(t *testing.T) {
	runMoneyMachine(t, moneyZeroTaxCatalog(), true)
}

// ---- scripted sequences ----

func assertMoney(t *testing.T, what string, got, want float64) {
	t.Helper()
	if !moneyClose(got, want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func assertCode(t *testing.T, step string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status %d, want %d", step, got, want)
	}
}

// T-0: the oracle self-check. The same model and invariants as T-1, driven by
// a fixed list whose numbers are worked out by hand in contract §4.
func TestMoneyModel_T0_ScriptedSequence(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	e.check(t)
	steps := []struct {
		name string
		run  func() int
		want int
	}{
		{"open $50.000", func() int { return e.doOpen(t, 50000) }, 201},
		{"cash sale 1 x Gaseosa", func() int {
			return e.doSale(t, 0, []moneyLine{{2, 1}}, []moneyPay{{"cash", 1191.19}})
		}, 201},
		{"credit sale 0.5 kg Queso to Dora", func() int {
			return e.doSale(t, 1, []moneyLine{{4, 0.5}}, []moneyPay{{"credit", 4999.5}})
		}, 201},
		{"mixed sale 4 x Arroz", func() int {
			return e.doSale(t, 0, []moneyLine{{1, 4}}, []moneyPay{{"cash", 4000}, {"card", 6000}})
		}, 201},
		{"transfer abono $2.000 from Dora", func() int { return e.doAbono(t, 1, 2000, "transfer", "cashier") }, 201},
		{"cash out $5.000", func() int { return e.doMovement(t, 1, "cash_out", 5000) }, 201},
		{"expense servicios $3.000", func() int { return e.doExpense(t, "servicios", 3000) }, 201},
		{"void the mixed sale", func() int { return e.doVoid(t, e.sales[2].id, "admin") }, 200},
	}
	for _, s := range steps {
		assertCode(t, s.name, s.run(), s.want)
		e.check(t)
	}

	var sum moneyRegisterSummary
	e.get(t, "/registers/1", &sum)
	reg := e.reg(1)
	assertMoney(t, "cash_sales", sum.CashSales, 1191.19)
	assertMoney(t, "model cash_sales", e.modelCashSales(1), 1191.19)
	assertMoney(t, "cash_abonos", sum.CashAbonos, 0)
	assertMoney(t, "model cash_abonos", e.modelCashAbonos(1), 0)
	assertMoney(t, "cash_out", sum.CashOut, 8000)
	assertMoney(t, "model cash_out", e.modelMoves(1, "cash_out"), 8000)
	assertMoney(t, "expected", sum.Expected, 43191.19)
	assertMoney(t, "model expected", e.modelExpected(reg), 43191.19)
	assertMoney(t, "Dora balance", e.customerBalance(t, 1), 2999.5)
	assertMoney(t, "model Dora balance", e.modelBalance(1), 2999.5)
	if got := e.float(t, `SELECT stock FROM products WHERE id = 1`); got != 500 {
		t.Errorf("Arroz stock = %v, want 500", got)
	}
	if got := e.float(t, `SELECT stock FROM products WHERE id = 4`); math.Abs(got-199.5) > stockTol {
		t.Errorf("Queso stock = %v, want 199.5", got)
	}
	if got := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1`); got != 6 {
		t.Errorf("register movements = %d, want 6 (opening, 3 x sale, cash_out, Gasto: servicios)", got)
	}
	if got := e.modelMovementRows(); got != 6 {
		t.Errorf("model register movements = %d, want 6", got)
	}
	if got := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1 AND description = 'Gasto: servicios'`); got != 1 {
		t.Errorf("'Gasto: servicios' movements = %d, want 1", got)
	}
}

// T-3: every AC-6 rejection by a fixed list; each do* call asserts the code
// and that the snapshot of the tables is unchanged.
func TestMoneyModel_T3_Rejections(t *testing.T) {
	e := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())

	assertCode(t, "sale with no register", e.doSale(t, 0, []moneyLine{{1, 1}}, []moneyPay{{"cash", 2500}}), 400)
	assertCode(t, "open", e.doOpen(t, 50000), 201)
	assertCode(t, "second open", e.doOpen(t, 10000), 409)
	assertCode(t, "credit sale 0.5 kg Queso to Dora", e.doSale(t, 1, []moneyLine{{4, 0.5}}, []moneyPay{{"credit", 4999.5}}), 201)
	assertCode(t, "abono $2.000", e.doAbono(t, 1, 2000, "cash", "cashier"), 201)
	assertMoney(t, "Dora balance", e.customerBalance(t, 1), 2999.5)
	assertCode(t, "abono $3.000 on $2.999,50", e.doAbono(t, 1, 3000, "cash", "cashier"), 400)
	assertCode(t, "501 x Arroz with stock 500", e.doSale(t, 0, []moneyLine{{1, 501}}, []moneyPay{{"cash", 1252500}}), 400)
	e.check(t)
	assertCode(t, "close register 1", e.doClose(t, 1, 50000), 200)
	assertCode(t, "close register 1 again", e.doClose(t, 1, 50000), 409)
	assertCode(t, "cash in $1.000 on the closed register", e.doMovement(t, 1, "cash_in", 1000), 409)
	assertCode(t, "cashier abono with no register", e.doAbono(t, 1, 1000, "cash", "cashier"), 403)
	e.check(t)

	fresh := newMoneyEnv(t, t.TempDir(), "", moneyFullCatalog())
	assertCode(t, "close register 1 with no register", fresh.doClose(t, 1, 0), 500)
	assertCode(t, "close register 99", fresh.doClose(t, 99, 0), 500)
	fresh.check(t)
}
