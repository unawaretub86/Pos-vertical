package handler

// US-090 contract §4, T-1 .. T-7. Every sale flow must work with DIAN
// e-invoicing turned off (fiscal_config.enabled = 0, the migration default),
// so h.factelect is nil. These go through h.Routes() with real JWTs, and each
// test captures the standard logger (the slog default handler writes through
// it) to prove that no flow logs anything (AC-6).

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

const dianOffMsg = "facturación electrónica no habilitada"

// dianOffHandler builds the handler on a fresh DB, asserts that the factelect
// service was not constructed, and redirects the standard logger to the
// returned buffer until the test ends.
func dianOffHandler(t *testing.T, db *database.DB) (*Handler, *bytes.Buffer) {
	t.Helper()
	h := New(db)
	if h.factelect != nil {
		t.Fatalf("h.factelect must be nil with fiscal_config.enabled = 0")
	}
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return h, &buf
}

func dianOffAssertNoLog(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	if buf.Len() != 0 {
		t.Errorf("standard logger received output, want none:\n%s", buf.String())
	}
}

func dianOffCount(t *testing.T, db *database.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.DB.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v\nSQL: %s", err, q)
	}
	return n
}

func dianOffStock(t *testing.T, db *database.DB, productID int64) float64 {
	t.Helper()
	var s float64
	if err := db.DB.QueryRowContext(context.Background(), `SELECT stock FROM products WHERE id = ?`, productID).Scan(&s); err != nil {
		t.Fatalf("stock of product %d: %v", productID, err)
	}
	return s
}

type dianOffSaleJSON struct {
	ID     int64   `json:"id"`
	Status string  `json:"status"`
	Total  float64 `json:"total"`
}

// dianOffSeedSale seeds Arroz ($50,000, stock 20), opens a register with
// $100,000 and posts sale 1: 1 × Arroz, cash $50,000, with a buyer email.
func dianOffSeedSale(t *testing.T, db *database.DB, h *Handler, admin string) {
	t.Helper()
	voidSeedCatalog(t, db)
	voidOpenRegister(t, h, admin, 100000)
	body := `{"items":[{"product_id":1,"quantity":1,"unit_price":50000}],"total":50000,"payments":[{"method":"cash","amount":50000}],"buyer_email":"a@b.co"}`
	w := abonoRequest(t, h, http.MethodPost, "/sales", admin, body)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sales = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var s dianOffSaleJSON
	abonoData(t, w, &s)
	if s.ID != 1 || s.Total != 50000 {
		t.Fatalf("created sale = id %d total %.0f, want id 1 total 50000", s.ID, s.Total)
	}
}

func dianOffGetSale(t *testing.T, h *Handler, token string, id int64) dianOffSaleJSON {
	t.Helper()
	w := abonoRequest(t, h, http.MethodGet, fmt.Sprintf("/sales/%d", id), token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /sales/%d = %d, want 200; body %s", id, w.Code, w.Body.String())
	}
	var s dianOffSaleJSON
	abonoData(t, w, &s)
	return s
}

func TestDianOff_T1_CreateReadTicket(t *testing.T) {
	db := setupTestDB(t)
	h, buf := dianOffHandler(t, db)
	admin := tokenFor(t, 1, "admin")

	dianOffSeedSale(t, db, h, admin)

	s := dianOffGetSale(t, h, admin, 1)
	if s.Status != "completed" || s.Total != 50000 {
		t.Errorf("GET /sales/1 = status %q total %.0f, want completed 50000", s.Status, s.Total)
	}

	w := abonoRequest(t, h, http.MethodGet, "/sales/1/ticket", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /sales/1/ticket = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var ticket struct {
		Total float64 `json:"total"`
		Items []struct {
			ProductName string `json:"product_name"`
		} `json:"items"`
	}
	abonoData(t, w, &ticket)
	if ticket.Total != 50000 {
		t.Errorf("ticket total = %.0f, want 50000", ticket.Total)
	}
	if len(ticket.Items) != 1 || ticket.Items[0].ProductName != "Arroz" {
		t.Errorf("ticket items = %+v, want 1 item Arroz", ticket.Items)
	}

	if n := dianOffCount(t, db, `SELECT COUNT(*) FROM electronic_documents`); n != 0 {
		t.Errorf("electronic_documents rows = %d, want 0", n)
	}
	dianOffAssertNoLog(t, buf)
}

func TestDianOff_T2_ReprintToNetworkPrinter(t *testing.T) {
	fp := startFakePrinter(t)
	db := setupTestDB(t)
	abonoExec(t, db, `INSERT INTO business_config (id, name, currency, address, nit, print_method, paper_width_mm, printer_host, printer_port, printer_name)
	      VALUES (1, 'Tienda Uno', 'COP', 'Calle 1 # 2-3', '900123456-7', 'escpos_network', 58, '127.0.0.1', ?, '')`, fp.port)
	h, buf := dianOffHandler(t, db)
	admin := tokenFor(t, 1, "admin")

	dianOffSeedSale(t, db, h, admin)

	w := postPrint(h, tokenFor(t, 2, "cashier"), "/printer/print-ticket/1")
	if w.Code != http.StatusOK {
		t.Fatalf("POST /printer/print-ticket/1 = %d, want 200; body %s", w.Code, w.Body.String())
	}
	if !printedFlag(t, w) {
		t.Errorf("printed flag not true: %s", w.Body.String())
	}
	if got := fp.received(true); !bytes.Contains(got, []byte("$50.000")) {
		t.Errorf("printer bytes must contain $50.000, got %q", got)
	}
	dianOffAssertNoLog(t, buf)
}

func TestDianOff_T3_VoidRestoresStock(t *testing.T) {
	db := setupTestDB(t)
	h, buf := dianOffHandler(t, db)
	admin := tokenFor(t, 1, "admin")

	dianOffSeedSale(t, db, h, admin)
	if s := dianOffStock(t, db, 1); s != 19 {
		t.Fatalf("stock after sale = %.0f, want 19", s)
	}

	w := abonoRequest(t, h, http.MethodDelete, "/sales/1/void", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /sales/1/void = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var s dianOffSaleJSON
	abonoData(t, w, &s)
	if s.Status != "voided" {
		t.Errorf("void status = %q, want voided", s.Status)
	}
	if st := dianOffStock(t, db, 1); st != 20 {
		t.Errorf("stock after void = %.0f, want 20", st)
	}

	w = abonoRequest(t, h, http.MethodDelete, "/sales/1/void", admin, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("second void = %d, want 400; body %s", w.Code, w.Body.String())
	}
	if msg := errorMessage(t, w); msg != "la venta ya fue anulada" {
		t.Errorf("second void error = %q, want %q", msg, "la venta ya fue anulada")
	}
	dianOffAssertNoLog(t, buf)
}

func TestDianOff_T4_TableCheckout(t *testing.T) {
	db := setupTestDB(t)
	h, buf := dianOffHandler(t, db)
	f := &orderFixture{t: t, db: db, h: h, tok: tokenFor(t, 1, "admin")}
	f.exec(`INSERT OR IGNORE INTO users (id, username, password_hash, role) VALUES (1, 'admin', 'x', 'admin')`)
	f.exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (1, 'Café', 'CAF', 4500, 1000, 'unit', 1, 100)`)
	f.exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active, stock) VALUES (2, 'Torta', 'TOR', 8000, 2000, 'unit', 1, 100)`)
	f.exec(`INSERT INTO registers (id, user_id, opening_amount, status) VALUES (1, 1, 0, 'open')`)

	orderID := f.openOrder(f.mkTable("Mesa 1"))
	f.addItem(orderID, cafeID, 2)
	f.addItem(orderID, tortaID, 1)

	w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/checkout", orderID),
		`{"payments":[{"method":"cash","amount":17000}],"buyer_email":"a@b.co"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("checkout = %d, want 201; body %s", w.Code, w.Body.String())
	}
	var res struct {
		OrderID int64   `json:"order_id"`
		SaleID  int64   `json:"sale_id"`
		Total   float64 `json:"total"`
	}
	abonoData(t, w, &res)
	if res.OrderID != orderID || res.SaleID != 1 || res.Total != 17000 {
		t.Errorf("checkout = %+v, want order_id %d sale_id 1 total 17000", res, orderID)
	}
	if n := dianOffCount(t, db, `SELECT COUNT(*) FROM electronic_documents`); n != 0 {
		t.Errorf("electronic_documents rows = %d, want 0", n)
	}

	w = f.do(http.MethodDelete, fmt.Sprintf("/sales/%d/void", res.SaleID), "")
	if w.Code != http.StatusOK {
		t.Errorf("void checkout sale = %d, want 200; body %s", w.Code, w.Body.String())
	}
	dianOffAssertNoLog(t, buf)
}

func TestDianOff_T5_CreditNoteRefused(t *testing.T) {
	db := setupTestDB(t)
	h, buf := dianOffHandler(t, db)
	admin := tokenFor(t, 1, "admin")

	dianOffSeedSale(t, db, h, admin)

	w := abonoRequest(t, h, http.MethodPost, "/sales/1/void-with-credit-note", admin, `{"motive":"error de digitación"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("void-with-credit-note = %d, want 400; body %s", w.Code, w.Body.String())
	}
	if msg := errorMessage(t, w); msg != dianOffMsg {
		t.Errorf("void-with-credit-note error = %q, want %q", msg, dianOffMsg)
	}
	if s := dianOffGetSale(t, h, admin, 1); s.Status != "completed" {
		t.Errorf("sale status after refused credit note = %q, want completed", s.Status)
	}

	w = abonoRequest(t, h, http.MethodPost, "/sales/1/void-with-credit-note", admin, `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("void-with-credit-note {} = %d, want 400; body %s", w.Code, w.Body.String())
	}
	if msg := errorMessage(t, w); msg != "motive requerido" {
		t.Errorf("void-with-credit-note {} error = %q, want %q", msg, "motive requerido")
	}

	w = abonoRequest(t, h, http.MethodPost, "/sales/1/void-with-credit-note", tokenFor(t, 2, "cashier"), `{"motive":"error de digitación"}`)
	if w.Code != http.StatusForbidden {
		t.Errorf("cashier void-with-credit-note = %d, want 403; body %s", w.Code, w.Body.String())
	}
	dianOffAssertNoLog(t, buf)
}

func TestDianOff_T6_FacturaEndpointsRefused(t *testing.T) {
	db := setupTestDB(t)
	h, buf := dianOffHandler(t, db)
	admin := tokenFor(t, 1, "admin")

	dianOffSeedSale(t, db, h, admin)

	for _, tc := range []struct {
		path string
		want int
	}{
		{"/sales/1/factura", http.StatusBadRequest},
		{"/sales/1/factura/pdf", http.StatusNotFound},
	} {
		w := abonoRequest(t, h, http.MethodGet, tc.path, admin, "")
		if w.Code != tc.want {
			t.Errorf("GET %s = %d, want %d; body %s", tc.path, w.Code, tc.want, w.Body.String())
		}
		if w.Code >= 500 {
			t.Errorf("GET %s answered 5xx", tc.path)
		}
		if msg := errorMessage(t, w); msg != dianOffMsg {
			t.Errorf("GET %s error = %q, want %q", tc.path, msg, dianOffMsg)
		}
	}
	dianOffAssertNoLog(t, buf)
}

func TestDianOff_T7_FiscalConfigRoute(t *testing.T) {
	db := setupTestDB(t)
	h, buf := dianOffHandler(t, db)
	admin := tokenFor(t, 1, "admin")

	w := abonoRequest(t, h, http.MethodGet, "/fiscal/config", admin, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /fiscal/config = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var cfg struct {
		Enabled *bool `json:"enabled"`
	}
	abonoData(t, w, &cfg)
	if cfg.Enabled == nil || *cfg.Enabled {
		t.Errorf("fiscal config enabled = %v, want false; body %s", cfg.Enabled, w.Body.String())
	}

	w = abonoRequest(t, h, http.MethodGet, "/fiscal-config", admin, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /fiscal-config = %d, want 404 (old frontend path must not exist)", w.Code)
	}
	dianOffAssertNoLog(t, buf)
}
