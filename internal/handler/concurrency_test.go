package handler

// US-081 contract §4, T-2 .. T-9: several devices on the LAN charging into the
// one open register at the same moment.
//
// Every test uses a file-backed database under t.TempDir(): with ":memory:"
// each pooled connection gets its own empty database (contract §2), so no
// real lock contention could happen. Requests go through h.Routes() with no
// Recoverer, so a handler panic crashes the test binary; a test that finishes
// is the AC-7 "no panic" check.
//
// Concurrent tests release all goroutines on one closed channel. Lock tests
// (T-6, T-7, T-9) hold BEGIN IMMEDIATE plus one INSERT on a second raw handle
// for the whole request, which waits out the 5 s busy_timeout, and release it
// right after the request returns.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

const ccBusyMsg = "El sistema está ocupado, reintenta en un momento"

type ccSummaryJSON struct {
	CashSales       float64 `json:"cash_sales"`
	Expected        float64 `json:"expected"`
	ByPaymentMethod []struct {
		Method string  `json:"method"`
		Total  float64 `json:"total"`
		Count  int     `json:"count"`
	} `json:"by_payment_method"`
}

func (s ccSummaryJSON) methodTotal(method string) float64 {
	for _, m := range s.ByPaymentMethod {
		if m.Method == method {
			return m.Total
		}
	}
	return 0
}

type ccEnv struct {
	path  string
	db    *database.DB
	h     *Handler
	admin string
}

// ccSetup opens a file-backed DB, seeds product 1 Arroz ($50,000, given
// stock) and customer 1 Dora, and opens register 1 with $100,000.
func ccSetup(t *testing.T, stock int) *ccEnv {
	t.Helper()
	path := filepath.Join(t.TempDir(), "c.db")
	db, err := database.New(path)
	if err != nil {
		t.Fatalf("open file db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	abonoExec(t, db, `INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Arroz', 'A', 50000, 30000, 'unit', 1, ?)`, stock)
	abonoExec(t, db, `INSERT INTO customers (id, name, phone, address) VALUES (1, 'Dora', '', '')`)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	voidOpenRegister(t, h, admin, 100000)
	return &ccEnv{path: path, db: db, h: h, admin: admin}
}

func (e *ccEnv) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.db.DB.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v\nSQL: %s", err, q)
	}
	return n
}

func (e *ccEnv) sum(t *testing.T, q string, args ...any) float64 {
	t.Helper()
	var v float64
	if err := e.db.DB.QueryRowContext(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatalf("sum: %v\nSQL: %s", err, q)
	}
	return v
}

func (e *ccEnv) stock(t *testing.T) float64 {
	t.Helper()
	return e.sum(t, `SELECT stock FROM products WHERE id = 1`)
}

func (e *ccEnv) summary(t *testing.T) ccSummaryJSON {
	t.Helper()
	w := abonoRequest(t, e.h, http.MethodGet, "/registers/1", e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /registers/1 = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var s ccSummaryJSON
	abonoData(t, w, &s)
	return s
}

// ccSaleBody is one unit of product 1 at $50,000. customerID 0 omits it.
func ccSaleBody(customerID int64, payments string) string {
	customer := ""
	if customerID != 0 {
		customer = fmt.Sprintf(`"customer_id":%d,`, customerID)
	}
	return fmt.Sprintf(`{%s"items":[{"product_id":1,"quantity":1,"unit_price":50000}],"total":50000,"payments":%s}`, customer, payments)
}

const (
	ccCash   = `[{"method":"cash","amount":50000}]`
	ccCredit = `[{"method":"credit","amount":50000}]`
	ccMixed  = `[{"method":"cash","amount":30000},{"method":"card","amount":20000}]`
)

type ccReq struct {
	method, path, body string
}

// ccConcurrent fires every request at once (one closed channel releases all
// goroutines) and returns the responses in request order.
func ccConcurrent(t *testing.T, h *Handler, token string, reqs []ccReq) []*httptest.ResponseRecorder {
	t.Helper()
	out := make([]*httptest.ResponseRecorder, len(reqs))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, r := range reqs {
		wg.Add(1)
		go func(i int, r ccReq) {
			defer wg.Done()
			<-start
			out[i] = abonoRequest(t, h, r.method, r.path, token, r.body)
		}(i, r)
	}
	close(start)
	wg.Wait()
	return out
}

// ccCodes counts responses per status and returns up to 3 sample bodies of
// responses whose status is not want.
func ccCodes(ws []*httptest.ResponseRecorder, want int) (map[int]int, []string) {
	codes := map[int]int{}
	var samples []string
	for _, w := range ws {
		codes[w.Code]++
		if w.Code != want && len(samples) < 3 {
			samples = append(samples, fmt.Sprintf("%d %s", w.Code, strings.TrimSpace(w.Body.String())))
		}
	}
	return codes, samples
}

func ccRepeat(n int, r ccReq) []ccReq {
	out := make([]ccReq, n)
	for i := range out {
		out[i] = r
	}
	return out
}

// ccHoldWriteLock opens a second raw handle on the same file and holds
// BEGIN IMMEDIATE plus one INSERT on it. The returned release is idempotent
// and is also registered with t.Cleanup, so the lock is always freed.
func ccHoldWriteLock(t *testing.T, e *ccEnv) (release func()) {
	t.Helper()
	ctx := context.Background()
	abonoExec(t, e.db, `CREATE TABLE IF NOT EXISTS cc_lock_probe (id INTEGER PRIMARY KEY)`)
	raw, err := sql.Open("sqlite", e.path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("open lock holder: %v", err)
	}
	conn, err := raw.Conn(ctx)
	if err != nil {
		raw.Close()
		t.Fatalf("lock holder conn: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		conn.Close()
		raw.Close()
		t.Fatalf("lock holder BEGIN IMMEDIATE: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO cc_lock_probe DEFAULT VALUES`); err != nil {
		conn.ExecContext(ctx, `ROLLBACK`)
		conn.Close()
		raw.Close()
		t.Fatalf("lock holder INSERT: %v", err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			conn.ExecContext(ctx, `ROLLBACK`)
			conn.Close()
			raw.Close()
		})
	}
	t.Cleanup(release)
	return release
}

func ccAssertBusy(t *testing.T, where string, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("%s under a held write lock = %d, want 503; body %s", where, w.Code, w.Body.String())
		return
	}
	if msg := abonoError(t, w); msg != ccBusyMsg {
		t.Errorf("%s 503 error = %q, want %q", where, msg, ccBusyMsg)
	}
	body := w.Body.String()
	if strings.Contains(body, "SQLITE") || strings.Contains(body, "locked") {
		t.Errorf("%s 503 body leaks driver text: %s", where, body)
	}
}

// ccSeedOrder seeds dining table 1 and open order 1 with the given number of
// items, each 1 unit of product 1 at $50,000.
func ccSeedOrder(t *testing.T, e *ccEnv, items int) {
	t.Helper()
	abonoExec(t, e.db, `INSERT INTO dining_tables (id, name, active) VALUES (1, 'Mesa 1', 1)`)
	abonoExec(t, e.db, `INSERT INTO orders (id, type, table_id, status, user_id) VALUES (1, 'table', 1, 'open', 1)`)
	for i := 0; i < items; i++ {
		abonoExec(t, e.db, `INSERT INTO order_items (order_id, product_id, quantity, unit_price, status) VALUES (1, 1, 1, 50000, 'pending')`)
	}
}

// T-2: 20 concurrent cash sales of 1 x $50,000 all commit.
func TestConcurrency_T2_TwentyCashSales(t *testing.T) {
	e := ccSetup(t, 100)
	ws := ccConcurrent(t, e.h, e.admin, ccRepeat(20, ccReq{http.MethodPost, "/sales", ccSaleBody(0, ccCash)}))

	codes, samples := ccCodes(ws, http.StatusCreated)
	if codes[http.StatusCreated] != 20 {
		t.Errorf("201 responses = %d of 20, want 20; codes %v; samples %v", codes[http.StatusCreated], codes, samples)
	}
	if got := e.stock(t); got != 80 {
		t.Errorf("stock = %.0f, want 80", got)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 20 {
		t.Errorf("sales = %d, want 20", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1 AND type = 'sale'`); n != 20 {
		t.Errorf("sale movements = %d, want 20", n)
	}
	s := e.summary(t)
	if s.CashSales != 1000000 {
		t.Errorf("cash_sales = %.0f, want 1000000", s.CashSales)
	}
	if s.Expected != 1100000 {
		t.Errorf("expected = %.0f, want 1100000", s.Expected)
	}
}

// T-3: 50 concurrent credit (fiado) sales of $50,000 to Dora all commit.
// Today their first statement inside the deferred tx is a read, so the
// read-to-write upgrade fails for many of them with SQLITE_BUSY.
func TestConcurrency_T3_FiftyCreditSales(t *testing.T) {
	e := ccSetup(t, 100)
	ws := ccConcurrent(t, e.h, e.admin, ccRepeat(50, ccReq{http.MethodPost, "/sales", ccSaleBody(1, ccCredit)}))

	codes, samples := ccCodes(ws, http.StatusCreated)
	if codes[http.StatusCreated] != 50 {
		t.Errorf("201 responses = %d of 50, want 50; codes %v; samples %v", codes[http.StatusCreated], codes, samples)
	}
	if got := e.stock(t); got != 50 {
		t.Errorf("stock = %.0f, want 50", got)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 50 {
		t.Errorf("sales = %d, want 50", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1 AND type = 'sale'`); n != 50 {
		t.Errorf("sale movements = %d, want 50", n)
	}

	w := abonoRequest(t, e.h, http.MethodGet, "/customers/1/balance", e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /customers/1/balance = %d; body %s", w.Code, w.Body.String())
	}
	var bal struct {
		Balance float64 `json:"balance"`
	}
	abonoData(t, w, &bal)
	if bal.Balance != 2500000 {
		t.Errorf("Dora balance = %.0f, want 2500000", bal.Balance)
	}

	s := e.summary(t)
	if s.CashSales != 0 {
		t.Errorf("cash_sales = %.0f, want 0", s.CashSales)
	}
	if got := s.methodTotal("credit"); got != 2500000 {
		t.Errorf("by_payment_method credit = %.0f, want 2500000", got)
	}
}

// T-4: 30 concurrent sales, 10 cash + 10 credit + 10 mixed (cash $30,000 +
// card $20,000), all commit with exact register totals.
func TestConcurrency_T4_MixedPaymentsThirtySales(t *testing.T) {
	e := ccSetup(t, 100)
	var reqs []ccReq
	for i := 0; i < 10; i++ {
		reqs = append(reqs,
			ccReq{http.MethodPost, "/sales", ccSaleBody(0, ccCash)},
			ccReq{http.MethodPost, "/sales", ccSaleBody(1, ccCredit)},
			ccReq{http.MethodPost, "/sales", ccSaleBody(0, ccMixed)},
		)
	}
	ws := ccConcurrent(t, e.h, e.admin, reqs)

	codes, samples := ccCodes(ws, http.StatusCreated)
	if codes[http.StatusCreated] != 30 {
		t.Errorf("201 responses = %d of 30, want 30; codes %v; samples %v", codes[http.StatusCreated], codes, samples)
	}
	if got := e.stock(t); got != 70 {
		t.Errorf("stock = %.0f, want 70", got)
	}
	s := e.summary(t)
	if s.CashSales != 800000 {
		t.Errorf("cash_sales = %.0f, want 800000", s.CashSales)
	}
	if s.Expected != 900000 {
		t.Errorf("expected = %.0f, want 900000", s.Expected)
	}
	if got := s.methodTotal("card"); got != 200000 {
		t.Errorf("by_payment_method card = %.0f, want 200000", got)
	}
	if got := s.methodTotal("credit"); got != 500000 {
		t.Errorf("by_payment_method credit = %.0f, want 500000", got)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1 AND type = 'sale'`); n != 30 {
		t.Errorf("sale movements = %d, want 30", n)
	}
	if got := e.sum(t, `SELECT COALESCE(SUM(amount), 0) FROM register_movements WHERE register_id = 1 AND type = 'sale'`); got != 1500000 {
		t.Errorf("sale movements sum = %.0f, want 1500000", got)
	}
}

// T-5: stock 10, 30 concurrent cash sales of 1 unit: exactly 10 commit, 20 are
// rejected with "stock insuficiente", and the rejected ones leave no rows.
func TestConcurrency_T5_StockContention(t *testing.T) {
	e := ccSetup(t, 10)
	ws := ccConcurrent(t, e.h, e.admin, ccRepeat(30, ccReq{http.MethodPost, "/sales", ccSaleBody(0, ccCash)}))

	created, rejected := 0, 0
	for _, w := range ws {
		switch w.Code {
		case http.StatusCreated:
			created++
		case http.StatusBadRequest:
			if msg := abonoError(t, w); strings.HasPrefix(msg, "stock insuficiente") {
				rejected++
			} else {
				t.Errorf("400 error = %q, want prefix %q", msg, "stock insuficiente")
			}
		default:
			t.Errorf("unexpected status %d; body %s", w.Code, w.Body.String())
		}
	}
	if created != 10 {
		t.Errorf("201 responses = %d, want 10", created)
	}
	if rejected != 20 {
		t.Errorf("400 stock insuficiente responses = %d, want 20", rejected)
	}
	if got := e.stock(t); got != 0 {
		t.Errorf("stock = %.0f, want 0", got)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != 10 {
		t.Errorf("sales = %d, want 10", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_items`); n != 10 {
		t.Errorf("sale_items = %d, want 10", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM sale_payments`); n != 10 {
		t.Errorf("sale_payments = %d, want 10", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM register_movements WHERE register_id = 1 AND type = 'sale'`); n != 10 {
		t.Errorf("sale movements = %d, want 10", n)
	}
}

type ccSnapshot struct {
	sales, payments, movements int
	stock                      float64
}

func (e *ccEnv) snapshot(t *testing.T) ccSnapshot {
	t.Helper()
	return ccSnapshot{
		sales:     e.count(t, `SELECT COUNT(*) FROM sales`),
		payments:  e.count(t, `SELECT COUNT(*) FROM sale_payments`),
		movements: e.count(t, `SELECT COUNT(*) FROM register_movements`),
		stock:     e.stock(t),
	}
}

// T-6: a write that cannot get the lock within busy_timeout answers 503 with
// msgBusy, never the driver text, and commits nothing.
func TestConcurrency_T6_BusyWriteAnswers503(t *testing.T) {
	cases := []struct {
		name   string
		seed   func(t *testing.T, e *ccEnv)
		method string
		path   string
		body   string
		after  func(t *testing.T, e *ccEnv)
	}{
		{
			name:   "POST /sales cash",
			seed:   func(t *testing.T, e *ccEnv) {},
			method: http.MethodPost,
			path:   "/sales",
			body:   ccSaleBody(0, ccCash),
		},
		{
			name:   "POST /orders/1/checkout",
			seed:   func(t *testing.T, e *ccEnv) { ccSeedOrder(t, e, 2) },
			method: http.MethodPost,
			path:   "/orders/1/checkout",
			body:   `{"payments":[{"method":"cash","amount":100000}]}`,
			after: func(t *testing.T, e *ccEnv) {
				if n := e.count(t, `SELECT COUNT(*) FROM orders WHERE id = 1 AND status = 'open' AND sale_id IS NULL`); n != 1 {
					t.Errorf("order 1 is no longer open with no sale after a 503 checkout")
				}
			},
		},
		{
			name:   "DELETE /sales/1/void",
			seed:   func(t *testing.T, e *ccEnv) { voidPostSale(t, e.h, e.admin, ccCash, 0) },
			method: http.MethodDelete,
			path:   "/sales/1/void",
			after: func(t *testing.T, e *ccEnv) {
				if n := e.count(t, `SELECT COUNT(*) FROM sales WHERE id = 1 AND status = 'completed'`); n != 1 {
					t.Errorf("sale 1 is no longer completed after a 503 void")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := ccSetup(t, 20)
			tc.seed(t, e)
			before := e.snapshot(t)

			release := ccHoldWriteLock(t, e)
			w := abonoRequest(t, e.h, tc.method, tc.path, e.admin, tc.body)
			release()

			ccAssertBusy(t, tc.name, w)
			if after := e.snapshot(t); after != before {
				t.Errorf("rows changed after a 503: before %+v, after %+v", before, after)
			}
			if tc.after != nil {
				tc.after(t, e)
			}
		})
	}
}

// T-7: ErrBusy on POST /orders/{id}/items answers 503 msgBusy (was 409) and
// adds no item.
func TestConcurrency_T7_OrderItemBusyAnswers503(t *testing.T) {
	e := ccSetup(t, 20)
	ccSeedOrder(t, e, 0)

	release := ccHoldWriteLock(t, e)
	w := abonoRequest(t, e.h, http.MethodPost, "/orders/1/items", e.admin, `{"product_id":1,"quantity":2}`)
	release()

	ccAssertBusy(t, "POST /orders/1/items", w)
	if n := e.count(t, `SELECT COUNT(*) FROM order_items WHERE order_id = 1`); n != 0 {
		t.Errorf("order_items of order 1 = %d, want 0", n)
	}
	if n := e.count(t, `SELECT COUNT(*) FROM orders WHERE id = 1 AND status = 'open'`); n != 1 {
		t.Errorf("order 1 is no longer open")
	}
}

// T-9: ErrBusy on DELETE /tables/{id} answers 503 msgBusy (was 409) and leaves
// the table active; once the lock is gone the same delete succeeds.
func TestConcurrency_T9_DeleteTableBusyAnswers503(t *testing.T) {
	e := ccSetup(t, 20)
	abonoExec(t, e.db, `INSERT INTO dining_tables (id, name, active) VALUES (1, 'Mesa 1', 1)`)

	release := ccHoldWriteLock(t, e)
	w := abonoRequest(t, e.h, http.MethodDelete, "/tables/1", e.admin, "")
	release()

	ccAssertBusy(t, "DELETE /tables/1", w)
	if n := e.count(t, `SELECT active FROM dining_tables WHERE id = 1`); n != 1 {
		t.Errorf("table 1 active = %d after a 503, want 1", n)
	}

	w = abonoRequest(t, e.h, http.MethodDelete, "/tables/1", e.admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("second DELETE /tables/1 = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var msg struct {
		Message string `json:"message"`
	}
	abonoData(t, w, &msg)
	if msg.Message != "mesa desactivada" {
		t.Errorf("second DELETE message = %q, want %q", msg.Message, "mesa desactivada")
	}
	if n := e.count(t, `SELECT active FROM dining_tables WHERE id = 1`); n != 0 {
		t.Errorf("table 1 active = %d after the second DELETE, want 0", n)
	}
}

// T-8: two concurrent voids of one sale void it once; two concurrent checkouts
// of one order create exactly one sale.
func TestConcurrency_T8_DoubleVoidAndDoubleCheckout(t *testing.T) {
	e := ccSetup(t, 20)
	voidPostSale(t, e.h, e.admin, ccCash, 0)
	if got := e.stock(t); got != 19 {
		t.Fatalf("stock after the sale = %.0f, want 19", got)
	}

	voids := ccConcurrent(t, e.h, e.admin, ccRepeat(2, ccReq{http.MethodDelete, "/sales/1/void", ""}))
	sort.Slice(voids, func(i, j int) bool { return voids[i].Code < voids[j].Code })
	if voids[0].Code != http.StatusOK || voids[1].Code != http.StatusBadRequest {
		t.Errorf("void statuses = [%d %d], want [200 400]; bodies %s | %s",
			voids[0].Code, voids[1].Code, voids[0].Body.String(), voids[1].Body.String())
	} else if msg := abonoError(t, voids[1]); msg != "la venta ya fue anulada" {
		t.Errorf("losing void error = %q, want %q", msg, "la venta ya fue anulada")
	}
	if got := e.stock(t); got != 20 {
		t.Errorf("stock after the voids = %.0f, want 20", got)
	}

	ccSeedOrder(t, e, 2)
	salesBefore := e.count(t, `SELECT COUNT(*) FROM sales`)
	checkouts := ccConcurrent(t, e.h, e.admin, ccRepeat(2, ccReq{http.MethodPost, "/orders/1/checkout", `{"payments":[{"method":"cash","amount":100000}]}`}))
	sort.Slice(checkouts, func(i, j int) bool { return checkouts[i].Code < checkouts[j].Code })
	if checkouts[0].Code != http.StatusCreated || checkouts[1].Code != http.StatusBadRequest {
		t.Fatalf("checkout statuses = [%d %d], want [201 400]; bodies %s | %s",
			checkouts[0].Code, checkouts[1].Code, checkouts[0].Body.String(), checkouts[1].Body.String())
	}
	var res struct {
		SaleID int64 `json:"sale_id"`
	}
	abonoData(t, checkouts[0], &res)

	if n := e.count(t, `SELECT COUNT(*) FROM sales`); n != salesBefore+1 {
		t.Errorf("sales after the checkouts = %d, want %d", n, salesBefore+1)
	}
	var status string
	var saleID sql.NullInt64
	if err := e.db.DB.QueryRow(`SELECT status, sale_id FROM orders WHERE id = 1`).Scan(&status, &saleID); err != nil {
		t.Fatalf("read order 1: %v", err)
	}
	if status != "closed" {
		t.Errorf("order 1 status = %q, want closed", status)
	}
	if !saleID.Valid || saleID.Int64 != res.SaleID {
		t.Errorf("order 1 sale_id = %v, want %d", saleID, res.SaleID)
	}
	if got := e.stock(t); got != 18 {
		t.Errorf("stock after the checkout = %.0f, want 18", got)
	}
}
