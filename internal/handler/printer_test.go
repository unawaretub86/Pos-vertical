package handler

// US-080 contract section 4, T-7, T-8, T-9, T-10, T-13 (print endpoints).
// Hardware is replaced by a local TCP listener that records what it receives.

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andreyo/pos/internal/database"
)

type fakePrinter struct {
	port int
	mu   sync.Mutex
	buf  bytes.Buffer
	ln   net.Listener
}

func startFakePrinter(t *testing.T) *fakePrinter {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	fp := &fakePrinter{ln: ln, port: ln.Addr().(*net.TCPAddr).Port}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				b, _ := io.ReadAll(c)
				fp.mu.Lock()
				fp.buf.Write(b)
				fp.mu.Unlock()
			}(c)
		}
	}()
	return fp
}

// received waits briefly for bytes to arrive and returns what was captured.
// With wantAny=false it waits a short grace period to prove nothing arrives.
func (fp *fakePrinter) received(wantAny bool) []byte {
	deadline := time.Now().Add(1500 * time.Millisecond)
	if !wantAny {
		deadline = time.Now().Add(300 * time.Millisecond)
	}
	for time.Now().Before(deadline) {
		fp.mu.Lock()
		n := fp.buf.Len()
		fp.mu.Unlock()
		if wantAny && n > 0 {
			time.Sleep(50 * time.Millisecond) // let the rest of the stream land
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return append([]byte(nil), fp.buf.Bytes()...)
}

// closedPort returns a TCP port on 127.0.0.1 with nothing listening.
func closedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	p := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return p
}

// seedPrintData seeds business config with the given print settings plus two
// sales: id 1 (completed, total 10200, cash 20000, change 9800) and id 2 (voided).
func seedPrintData(t *testing.T, db *database.DB, method, host string, port int, name string) {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("seed exec: %v\nSQL: %s", err, q)
		}
	}
	exec(`INSERT INTO business_config (id, name, currency, address, nit, print_method, paper_width_mm, printer_host, printer_port, printer_name)
	      VALUES (1, 'Tienda Uno', 'COP', 'Calle 1 # 2-3', '900123456-7', ?, 58, ?, ?, ?)`, method, host, port, name)
	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Gaseosa 1.5L', 'G', 4500, 2000, 'unit', 1)`)
	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Pan', 'P', 1200, 500, 'unit', 1)`)
	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method, cash_tendered, change_amount) VALUES (1, 1, 10200, 10200, 'completed', 'cash', 20000, 9800)`)
	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 1, 2, 4500, 9000)`)
	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (1, 2, 1, 1200, 1200)`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (1, 'cash', 10200)`)
	exec(`INSERT INTO sales (id, user_id, subtotal, total, status, payment_method) VALUES (2, 1, 1200, 1200, 'voided', 'cash')`)
	exec(`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES (2, 2, 1, 1200, 1200)`)
	exec(`INSERT INTO sale_payments (sale_id, method, amount) VALUES (2, 'cash', 1200)`)
}

func postPrint(h *Handler, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

// printedFlag accepts both {"printed":true} and the project envelope {"data":{"printed":true}}.
func printedFlag(t *testing.T, w *httptest.ResponseRecorder) bool {
	t.Helper()
	var env struct {
		Printed *bool `json:"printed"`
		Data    struct {
			Printed *bool `json:"printed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, w.Body.String())
	}
	if env.Data.Printed != nil {
		return *env.Data.Printed
	}
	return env.Printed != nil && *env.Printed
}

func errorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return env.Error
}

func TestPrintTicket_T7_NetworkPrintsSale(t *testing.T) {
	fp := startFakePrinter(t)
	db := setupTestDB(t)
	h := New(db)
	seedPrintData(t, db, "escpos_network", "127.0.0.1", fp.port, "")

	w := postPrint(h, tokenFor(t, 2, "cashier"), "/printer/print-ticket/1")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if !printedFlag(t, w) {
		t.Errorf("printed flag not true: %s", w.Body.String())
	}
	got := fp.received(true)
	if !bytes.HasPrefix(got, []byte{0x1B, 0x40}) {
		t.Errorf("printer bytes must start with ESC @, got % X", got)
	}
	if !bytes.Contains(got, []byte("$10.200")) {
		t.Errorf("printer bytes must contain $10.200")
	}
}

func TestPrintTicket_T8_BrowserModeReturns409(t *testing.T) {
	fp := startFakePrinter(t)
	db := setupTestDB(t)
	h := New(db)
	seedPrintData(t, db, "browser", "127.0.0.1", fp.port, "")

	w := postPrint(h, tokenFor(t, 2, "cashier"), "/printer/print-ticket/1")
	if w.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409; body %s", w.Code, w.Body.String())
	}
	if errorMessage(t, w) == "" {
		t.Errorf("409 must carry a Spanish error message")
	}
	if got := fp.received(false); len(got) != 0 {
		t.Errorf("printer received %d bytes in browser mode, want 0", len(got))
	}
}

func TestPrintTicket_T9_IdValidationAndVoided(t *testing.T) {
	fp := startFakePrinter(t)
	db := setupTestDB(t)
	h := New(db)
	seedPrintData(t, db, "escpos_network", "127.0.0.1", fp.port, "")
	cashier := tokenFor(t, 2, "cashier")

	for _, tc := range []struct {
		id   string
		want int
	}{{"abc", 400}, {"0", 400}, {"-5", 400}, {"999", 404}} {
		w := postPrint(h, cashier, "/printer/print-ticket/"+tc.id)
		if w.Code != tc.want {
			t.Errorf("id %q: status %d, want %d", tc.id, w.Code, tc.want)
		}
	}
	if w := postPrint(h, "", "/printer/print-ticket/1"); w.Code != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", w.Code)
	}
	if got := fp.received(false); len(got) != 0 {
		t.Errorf("rejected requests wrote %d bytes to the printer, want 0", len(got))
	}

	w := postPrint(h, cashier, "/printer/print-ticket/2")
	if w.Code != http.StatusOK {
		t.Fatalf("voided sale: status %d, want 200; body %s", w.Code, w.Body.String())
	}
	if got := fp.received(true); !bytes.Contains(got, []byte("*** ANULADO ***")) {
		t.Errorf("voided sale bytes must contain *** ANULADO ***")
	}
}

func TestPrintTicket_T10_UnreachablePrinterReturns502(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	seedPrintData(t, db, "escpos_network", "127.0.0.1", closedPort(t), "")

	w := postPrint(h, tokenFor(t, 2, "cashier"), "/printer/print-ticket/1")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("status %d, want 502; body %s", w.Code, w.Body.String())
	}
	if errorMessage(t, w) == "" {
		t.Errorf("502 must carry a message naming the cause")
	}
	var total float64
	if err := db.QueryRow(`SELECT total FROM sales WHERE id = 1`).Scan(&total); err != nil || total != 10200 {
		t.Errorf("sale 1 must survive a print failure with total 10200, got %v err=%v", total, err)
	}
}

func TestPrintTest_T13_AdminTestPrint(t *testing.T) {
	fp := startFakePrinter(t)
	db := setupTestDB(t)
	h := New(db)
	seedPrintData(t, db, "escpos_network", "127.0.0.1", fp.port, "")

	// A cashier is forbidden and nothing is printed.
	if w := postPrint(h, tokenFor(t, 2, "cashier"), "/printer/test"); w.Code != http.StatusForbidden {
		t.Errorf("cashier: status %d, want 403", w.Code)
	}
	if got := fp.received(false); len(got) != 0 {
		t.Errorf("cashier test print wrote %d bytes, want 0", len(got))
	}

	w := postPrint(h, tokenFor(t, 1, "admin"), "/printer/test")
	if w.Code != http.StatusOK {
		t.Fatalf("admin: status %d, body %s", w.Code, w.Body.String())
	}
	if !printedFlag(t, w) {
		t.Errorf("printed flag not true: %s", w.Body.String())
	}
	got := fp.received(true)
	if !bytes.HasPrefix(got, []byte{0x1B, 0x40}) || !bytes.Contains(got, []byte("PRUEBA DE IMPRESION")) {
		t.Errorf("test ticket must start with ESC @ and contain PRUEBA DE IMPRESION, got % X", got)
	}

	// Browser mode -> 409.
	if _, err := db.Exec(`UPDATE business_config SET print_method = 'browser' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if w := postPrint(h, tokenFor(t, 1, "admin"), "/printer/test"); w.Code != http.StatusConflict {
		t.Errorf("browser mode: status %d, want 409", w.Code)
	}

	// Closed port -> 502.
	if _, err := db.Exec(`UPDATE business_config SET print_method = 'escpos_network', printer_port = ? WHERE id = 1`, closedPort(t)); err != nil {
		t.Fatal(err)
	}
	if w := postPrint(h, tokenFor(t, 1, "admin"), "/printer/test"); w.Code != http.StatusBadGateway {
		t.Errorf("closed port: status %d, want 502", w.Code)
	}
}
