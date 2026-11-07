package handler

// Contrato US-056 §4: T-1, T-2, T-4..T-13, T-20, T-21, T-22 (pedidos por mesa).
// Las FK no se aplican (R-56); los fixtures son INSERT literales.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

const (
	cafeID    = 1 // $4.500
	tortaID   = 2 // $8.000
	seedStamp = "2026-01-01 00:00:00"
)

type orderFixture struct {
	t   *testing.T
	db  *database.DB
	h   *Handler
	tok string // admin, user 1
}

func newOrderFixture(t *testing.T) *orderFixture {
	t.Helper()
	return fixtureOn(t, setupTestDB(t))
}

func fixtureOn(t *testing.T, db *database.DB) *orderFixture {
	t.Helper()
	f := &orderFixture{t: t, db: db, h: New(db), tok: tokenFor(t, 1, "admin")}
	f.exec(`INSERT OR IGNORE INTO users (id, username, password_hash, role) VALUES (1, 'admin', 'x', 'admin')`)
	f.exec(`INSERT OR IGNORE INTO users (id, username, password_hash, role) VALUES (2, 'caja', 'x', 'cashier')`)
	f.exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Café', 'CAF', 4500, 1000, 'unit', 1)`)
	f.exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Torta', 'TOR', 8000, 2000, 'unit', 1)`)
	f.exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (3, 'Viejo', 'OLD', 1000, 500, 'unit', 0)`)
	return f
}

func (f *orderFixture) exec(q string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		f.t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

func (f *orderFixture) count(q string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRow(q, args...).Scan(&n); err != nil {
		f.t.Fatalf("count: %v\nSQL: %s", err, q)
	}
	return n
}

func (f *orderFixture) do(method, path, body string) *httptest.ResponseRecorder {
	return doAuth(f.h, f.tok, method, path, body)
}

func (f *orderFixture) mkTable(name string) int64 {
	f.t.Helper()
	w := f.do(http.MethodPost, "/tables", fmt.Sprintf(`{"name":%q}`, name))
	if w.Code != http.StatusCreated {
		f.t.Fatalf("create table: %d %s", w.Code, w.Body.String())
	}
	return decodeTable(f.t, w).ID
}

type orderItemJSON struct {
	ID          int64   `json:"id"`
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
	Note        string  `json:"note"`
	Status      string  `json:"status"`
}

type orderJSON struct {
	ID      int64           `json:"id"`
	Type    string          `json:"type"`
	TableID int64           `json:"table_id"`
	Status  string          `json:"status"`
	Total   float64         `json:"total"`
	Items   []orderItemJSON `json:"items"`
}

type orderSummaryJSON struct {
	ID            int64   `json:"id"`
	TableID       int64   `json:"table_id"`
	TableName     string  `json:"table_name"`
	Total         float64 `json:"total"`
	ItemsPending  int     `json:"items_pending"`
	OpenedMinutes float64 `json:"opened_minutes"`
}

func decodeInto[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var env struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, w.Body.String())
	}
	return env.Data
}

func (f *orderFixture) openOrder(tableID int64) int64 {
	f.t.Helper()
	w := f.do(http.MethodPost, "/orders", fmt.Sprintf(`{"table_id":%d}`, tableID))
	if w.Code != http.StatusCreated {
		f.t.Fatalf("open order: %d %s", w.Code, w.Body.String())
	}
	return decodeInto[orderJSON](f.t, w).ID
}

func (f *orderFixture) addItem(orderID, productID int64, qty int) int64 {
	f.t.Helper()
	w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/items", orderID),
		fmt.Sprintf(`{"product_id":%d,"quantity":%d}`, productID, qty))
	if w.Code != http.StatusCreated {
		f.t.Fatalf("add item: %d %s", w.Code, w.Body.String())
	}
	return decodeInto[orderItemJSON](f.t, w).ID
}

func (f *orderFixture) getOrder(orderID int64) orderJSON {
	f.t.Helper()
	w := f.do(http.MethodGet, fmt.Sprintf("/orders/%d", orderID), "")
	if w.Code != http.StatusOK {
		f.t.Fatalf("get order: %d %s", w.Code, w.Body.String())
	}
	return decodeInto[orderJSON](f.t, w)
}

func (f *orderFixture) patch(orderID, itemID int64, body string) *httptest.ResponseRecorder {
	return f.do(http.MethodPatch, fmt.Sprintf("/orders/%d/items/%d", orderID, itemID), body)
}

func (f *orderFixture) voidItem(orderID, itemID int64) *httptest.ResponseRecorder {
	return f.do(http.MethodDelete, fmt.Sprintf("/orders/%d/items/%d", orderID, itemID), "")
}

func (f *orderFixture) itemStatus(itemID int64) string {
	f.t.Helper()
	var s string
	if err := f.db.QueryRow(`SELECT status FROM order_items WHERE id = ?`, itemID).Scan(&s); err != nil {
		f.t.Fatalf("item status: %v", err)
	}
	return s
}

func (f *orderFixture) reseedUpdatedAt(orderID int64) {
	f.t.Helper()
	f.exec(`UPDATE orders SET updated_at = ? WHERE id = ?`, seedStamp, orderID)
}

func (f *orderFixture) updatedAtChanged(orderID int64) bool {
	f.t.Helper()
	return f.count(`SELECT COUNT(*) FROM orders WHERE id = ? AND updated_at = ?`, orderID, seedStamp) == 0
}

func TestOrder_T1_AddItemsTotalAndVoid(t *testing.T) {
	f := newOrderFixture(t)
	mesa1 := f.mkTable("Mesa 1")
	oid := f.openOrder(mesa1)
	f.addItem(oid, cafeID, 2)
	torta := f.addItem(oid, tortaID, 1)

	o := f.getOrder(oid)
	if o.Total != 17000 {
		t.Errorf("total = %v, want 17000", o.Total)
	}
	if len(o.Items) != 2 || o.Items[0].Subtotal != 9000 || o.Items[1].Subtotal != 8000 {
		t.Errorf("items = %+v, want subtotals 9000 and 8000", o.Items)
	}

	if w := f.voidItem(oid, torta); w.Code != http.StatusOK {
		t.Fatalf("void status = %d, body %s", w.Code, w.Body.String())
	}
	o = f.getOrder(oid)
	if o.Total != 9000 {
		t.Errorf("total after void = %v, want 9000", o.Total)
	}
	if len(o.Items) != 2 {
		t.Fatalf("items after void = %d, want 2 (void included)", len(o.Items))
	}
	if got := f.itemStatus(torta); got != "void" {
		t.Errorf("torta status = %q, want void", got)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ?`, torta); n != 1 {
		t.Errorf("torta rows = %d, want 1 (no physical delete)", n)
	}

	if n := f.count(`SELECT COUNT(*) FROM audit_log WHERE action='order_item_void'`); n != 1 {
		t.Fatalf("audit rows = %d, want 1", n)
	}
	var entity, details string
	var entityID, userID int64
	if err := f.db.QueryRow(`SELECT entity, entity_id, user_id, details FROM audit_log WHERE action='order_item_void'`).
		Scan(&entity, &entityID, &userID, &details); err != nil {
		t.Fatalf("scan audit: %v", err)
	}
	if entity != "order_item" || entityID != torta || userID != 1 {
		t.Errorf("audit = %s/%d/user %d, want order_item/%d/user 1", entity, entityID, userID, torta)
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(details), &d); err != nil {
		t.Fatalf("details not JSON: %v\n%s", err, details)
	}
	want := map[string]any{
		"order_id": float64(oid), "product_id": float64(tortaID), "product_name": "Torta",
		"quantity": float64(1), "unit_price": float64(8000), "note": "", "status": "pending",
		"table_id": float64(mesa1), "user_id": float64(1),
	}
	if len(d) != len(want) {
		t.Errorf("details has %d keys, want exactly %d: %s", len(d), len(want), details)
	}
	for k, v := range want {
		if d[k] != v {
			t.Errorf("details[%q] = %v, want %v", k, d[k], v)
		}
	}

	// Second void: 409 and no second audit row.
	if w := f.voidItem(oid, torta); w.Code != http.StatusConflict {
		t.Errorf("second void status = %d, want 409", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM audit_log WHERE action='order_item_void'`); n != 1 {
		t.Errorf("audit rows after second void = %d, want 1", n)
	}
}

// Q-8 (decision PO): details.status is the item state right before the void.
func TestOrder_T1b_VoidDetailsStatusIsPreVoidState(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	item := f.addItem(oid, cafeID, 1)
	if w := f.patch(oid, item, `{"status":"ready"}`); w.Code != http.StatusOK {
		t.Fatalf("patch ready: %d %s", w.Code, w.Body.String())
	}
	if w := f.voidItem(oid, item); w.Code != http.StatusOK {
		t.Fatalf("void: %d %s", w.Code, w.Body.String())
	}
	var details string
	if err := f.db.QueryRow(`SELECT details FROM audit_log WHERE action='order_item_void'`).Scan(&details); err != nil {
		t.Fatalf("scan: %v", err)
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(details), &d); err != nil {
		t.Fatalf("details: %v", err)
	}
	if d["status"] != "ready" {
		t.Errorf("details.status = %v, want ready (state before void)", d["status"])
	}
}

func TestOrder_T2_SecondOpenConflicts(t *testing.T) {
	f := newOrderFixture(t)
	mesa := f.mkTable("Mesa 1")
	body := fmt.Sprintf(`{"table_id":%d}`, mesa)
	if w := f.do(http.MethodPost, "/orders", body); w.Code != http.StatusCreated {
		t.Fatalf("first open = %d, want 201; %s", w.Code, w.Body.String())
	}
	if w := f.do(http.MethodPost, "/orders", body); w.Code != http.StatusConflict {
		t.Fatalf("second open = %d, want 409; %s", w.Code, w.Body.String())
	}
	if n := f.count(`SELECT COUNT(*) FROM orders WHERE table_id = ? AND status = 'open'`, mesa); n != 1 {
		t.Errorf("open orders = %d, want 1", n)
	}
}

func TestOrder_T4_CancelOrder(t *testing.T) {
	f := newOrderFixture(t)
	mesa := f.mkTable("Mesa 1")
	oid := f.openOrder(mesa)
	f.addItem(oid, cafeID, 2)
	f.addItem(oid, tortaID, 1)

	w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/cancel", oid), `{"reason":"cliente se fue"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, body %s", w.Code, w.Body.String())
	}
	if n := f.count(`SELECT COUNT(*) FROM orders WHERE id = ? AND status = 'cancelled' AND closed_at IS NOT NULL`, oid); n != 1 {
		t.Errorf("cancelled order with closed_at = %d rows, want 1", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM audit_log WHERE action='order_cancel'`); n != 1 {
		t.Fatalf("audit rows = %d, want 1", n)
	}
	var entity, details string
	var entityID, userID int64
	if err := f.db.QueryRow(`SELECT entity, entity_id, user_id, details FROM audit_log WHERE action='order_cancel'`).
		Scan(&entity, &entityID, &userID, &details); err != nil {
		t.Fatalf("scan audit: %v", err)
	}
	if entity != "order" || entityID != oid || userID != 1 || details != "cliente se fue" {
		t.Errorf("audit = %s/%d/user %d/%q, want order/%d/user 1/'cliente se fue'", entity, entityID, userID, details, oid)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE order_id = ?`, oid); n != 2 {
		t.Errorf("items = %d, want 2 kept", n)
	}
	tables, _ := listTablesJSON(t, f.h, f.tok)
	got := byName(tables, "Mesa 1")
	if got.State != "libre" || got.OpenOrderID != nil {
		t.Errorf("Mesa 1 = state %q open_order_id %v, want libre / nil", got.State, got.OpenOrderID)
	}
}

func TestOrder_T5_CancelRejections(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))

	w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/cancel", oid), `{"reason":"   "}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("blank reason status = %d, want 400", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM orders WHERE id = ? AND status = 'open'`, oid); n != 1 {
		t.Errorf("order no longer open after rejected cancel")
	}
	if n := f.count(`SELECT COUNT(*) FROM audit_log`); n != 0 {
		t.Errorf("audit rows = %d, want 0", n)
	}

	for _, st := range []string{"cancelled", "closed"} {
		f.exec(`UPDATE orders SET status = ? WHERE id = ?`, st, oid)
		w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/cancel", oid), `{"reason":"x"}`)
		if w.Code != http.StatusConflict {
			t.Errorf("cancel on %s status = %d, want 409", st, w.Code)
		}
	}
	if n := f.count(`SELECT COUNT(*) FROM audit_log`); n != 0 {
		t.Errorf("audit rows after 409s = %d, want 0", n)
	}
	if w := f.do(http.MethodPost, "/orders/9999/cancel", `{"reason":"x"}`); w.Code != http.StatusNotFound {
		t.Errorf("cancel missing order status = %d, want 404", w.Code)
	}
}

func TestOrder_T6_ClosedOrderRejectsItemChanges(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	item := f.addItem(oid, cafeID, 1)
	f.exec(`UPDATE orders SET status = 'closed' WHERE id = ?`, oid)

	if w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/items", oid), `{"product_id":1,"quantity":1}`); w.Code != http.StatusConflict {
		t.Errorf("POST item on closed = %d, want 409", w.Code)
	}
	if w := f.patch(oid, item, `{"quantity":5}`); w.Code != http.StatusConflict {
		t.Errorf("PATCH on closed = %d, want 409", w.Code)
	}
	if w := f.voidItem(oid, item); w.Code != http.StatusConflict {
		t.Errorf("DELETE on closed = %d, want 409", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE order_id = ?`, oid); n != 1 {
		t.Errorf("items = %d, want 1 (none added)", n)
	}
	if got := f.itemStatus(item); got != "pending" {
		t.Errorf("item status = %q, want pending", got)
	}
	if w := f.do(http.MethodPost, "/orders/9999/items", `{"product_id":1,"quantity":1}`); w.Code != http.StatusNotFound {
		t.Errorf("POST item on missing order = %d, want 404", w.Code)
	}
}

func TestOrder_T7_ValidationOfQuantityStatusProduct(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	item := f.addItem(oid, cafeID, 1)
	base := f.count(`SELECT COUNT(*) FROM order_items`)

	for _, q := range []string{"0", "-1", "0.5", "2.0", `"2"`} {
		body := fmt.Sprintf(`{"product_id":1,"quantity":%s}`, q)
		if w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/items", oid), body); w.Code != http.StatusBadRequest {
			t.Errorf("POST quantity %s = %d, want 400", q, w.Code)
		}
		if w := f.patch(oid, item, fmt.Sprintf(`{"quantity":%s}`, q)); w.Code != http.StatusBadRequest {
			t.Errorf("PATCH quantity %s = %d, want 400", q, w.Code)
		}
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items`); n != base {
		t.Errorf("items = %d, want %d (no new rows)", n, base)
	}

	w := f.do(http.MethodPost, fmt.Sprintf("/orders/%d/items", oid), `{"product_id":3,"quantity":1}`)
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("inactive product = %d, want 404 or 400", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items`); n != base {
		t.Errorf("inactive product created an item")
	}

	for _, st := range []string{"void", "cancelled"} {
		if w := f.patch(oid, item, fmt.Sprintf(`{"status":%q}`, st)); w.Code != http.StatusBadRequest {
			t.Errorf("PATCH status %s = %d, want 400", st, w.Code)
		}
	}
	for _, st := range []string{"ready", "preparing", "delivered"} {
		if w := f.patch(oid, item, fmt.Sprintf(`{"status":%q}`, st)); w.Code != http.StatusOK {
			t.Errorf("PATCH status %s = %d, want 200; %s", st, w.Code, w.Body.String())
		}
		if got := f.itemStatus(item); got != st {
			t.Errorf("persisted status = %q, want %q", got, st)
		}
	}
}

func TestOrder_T8_PriceSnapshot(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	f.addItem(oid, cafeID, 2)
	f.exec(`UPDATE products SET sale_price = 5000 WHERE id = 1`)

	o := f.getOrder(oid)
	if o.Items[0].UnitPrice != 4500 || o.Total != 9000 {
		t.Errorf("after price change: unit_price %v total %v, want 4500 / 9000", o.Items[0].UnitPrice, o.Total)
	}
	f.addItem(oid, cafeID, 1)
	o = f.getOrder(oid)
	if len(o.Items) != 2 || o.Items[1].UnitPrice != 5000 {
		t.Errorf("new item = %+v, want unit_price 5000", o.Items)
	}
}

func TestOrder_T9_TouchesUpdatedAt(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))

	f.reseedUpdatedAt(oid)
	item := f.addItem(oid, cafeID, 1)
	if !f.updatedAtChanged(oid) {
		t.Errorf("POST item did not touch updated_at")
	}
	f.reseedUpdatedAt(oid)
	if w := f.patch(oid, item, `{"note":"sin azúcar"}`); w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	if !f.updatedAtChanged(oid) {
		t.Errorf("PATCH note did not touch updated_at")
	}
	f.reseedUpdatedAt(oid)
	if w := f.voidItem(oid, item); w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	if !f.updatedAtChanged(oid) {
		t.Errorf("DELETE item did not touch updated_at")
	}
}

func TestOrder_T10_PatchQuantityAndNote(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	item := f.addItem(oid, cafeID, 2)

	w := f.patch(oid, item, `{"quantity":3,"note":"sin azúcar"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	o := f.getOrder(oid)
	it := o.Items[0]
	if it.Quantity != 3 || it.Subtotal != 13500 || it.Note != "sin azúcar" || it.Status != "pending" || o.Total != 13500 {
		t.Errorf("item %+v total %v, want qty 3 subtotal 13500 note 'sin azúcar' pending total 13500", it, o.Total)
	}

	// Fields not sent stay intact.
	if w := f.patch(oid, item, `{"quantity":4}`); w.Code != http.StatusOK {
		t.Fatalf("patch qty: %d", w.Code)
	}
	it = f.getOrder(oid).Items[0]
	if it.Quantity != 4 || it.Note != "sin azúcar" {
		t.Errorf("after qty-only patch: %+v, want qty 4 and note kept", it)
	}

	f.exec(`UPDATE order_items SET status = 'void' WHERE id = ?`, item)
	if w := f.patch(oid, item, `{"note":"x"}`); w.Code != http.StatusConflict {
		t.Errorf("PATCH void item = %d, want 409", w.Code)
	}
}

func TestOrder_T11_OpenValidation(t *testing.T) {
	f := newOrderFixture(t)
	inactive := f.mkTable("Vieja")
	if w := f.do(http.MethodDelete, fmt.Sprintf("/tables/%d", inactive), ""); w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("deactivate table: %d %s", w.Code, w.Body.String())
	}
	cases := []struct {
		name, body string
		want       int
	}{
		{"missing table", `{"table_id":999}`, http.StatusNotFound},
		{"inactive table", fmt.Sprintf(`{"table_id":%d}`, inactive), http.StatusNotFound},
		{"empty body", `{}`, http.StatusBadRequest},
		{"takeaway type", `{"table_id":1,"type":"takeaway"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		if w := f.do(http.MethodPost, "/orders", c.body); w.Code != c.want {
			t.Errorf("%s: status = %d, want %d; %s", c.name, w.Code, c.want, w.Body.String())
		}
	}
	if n := f.count(`SELECT COUNT(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestOrder_T12_ListOpen(t *testing.T) {
	f := newOrderFixture(t)

	w := f.do(http.MethodGet, "/orders?status=open", "")
	var raw struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil || w.Code != http.StatusOK || string(raw.Data) != "[]" {
		t.Errorf("empty list = %d %s, want 200 with data []", w.Code, w.Body.String())
	}

	m1 := f.mkTable("Mesa 1")
	m2 := f.mkTable("Mesa 2")
	o1 := f.openOrder(m1)
	f.addItem(o1, cafeID, 2)
	torta := f.addItem(o1, tortaID, 1)
	if w := f.patch(o1, torta, `{"status":"ready"}`); w.Code != http.StatusOK {
		t.Fatalf("patch ready: %d", w.Code)
	}
	f.openOrder(m2)

	w = f.do(http.MethodGet, "/orders?status=open", "")
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	rows := decodeInto[[]orderSummaryJSON](t, w)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2: %s", len(rows), w.Body.String())
	}
	by := map[string]orderSummaryJSON{}
	for _, r := range rows {
		by[r.TableName] = r
	}
	r1, r2 := by["Mesa 1"], by["Mesa 2"]
	if r1.Total != 17000 || r1.ItemsPending != 1 || r1.OpenedMinutes != 0 {
		t.Errorf("Mesa 1 = %+v, want total 17000 pending 1 minutes 0", r1)
	}
	if r2.Total != 0 || r2.ItemsPending != 0 {
		t.Errorf("Mesa 2 = %+v, want total 0 pending 0", r2)
	}
	if w := f.do(http.MethodGet, "/orders?status=foo", ""); w.Code != http.StatusBadRequest {
		t.Errorf("status=foo = %d, want 400", w.Code)
	}
}

func TestOrder_T13_CashierAllowedAnonymousRejected(t *testing.T) {
	f := newOrderFixture(t)
	mesa := f.mkTable("Mesa 1")
	cashier := tokenFor(t, 2, "cashier")

	w := doAuth(f.h, cashier, http.MethodPost, "/orders", fmt.Sprintf(`{"table_id":%d}`, mesa))
	if w.Code != http.StatusCreated {
		t.Fatalf("cashier open = %d, want 201; %s", w.Code, w.Body.String())
	}
	oid := decodeInto[orderJSON](t, w).ID
	w = doAuth(f.h, cashier, http.MethodPost, fmt.Sprintf("/orders/%d/items", oid), `{"product_id":1,"quantity":1}`)
	if w.Code != http.StatusCreated {
		t.Errorf("cashier add item = %d, want 201", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM orders WHERE id = ? AND user_id = 2`, oid); n != 1 {
		t.Errorf("order user_id != 2")
	}

	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	rec := httptest.NewRecorder()
	f.h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", rec.Code)
	}
}

func TestOrder_T20_EditReadyAndDeliveredItems(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	a := f.addItem(oid, cafeID, 2)
	b := f.addItem(oid, cafeID, 2)
	if w := f.patch(oid, a, `{"status":"ready"}`); w.Code != http.StatusOK {
		t.Fatalf("patch ready: %d", w.Code)
	}
	if w := f.patch(oid, b, `{"status":"delivered"}`); w.Code != http.StatusOK {
		t.Fatalf("patch delivered: %d", w.Code)
	}

	for _, c := range []struct {
		id int64
		st string
	}{{a, "ready"}, {b, "delivered"}} {
		if w := f.patch(oid, c.id, `{"quantity":3,"note":"sin azúcar"}`); w.Code != http.StatusOK {
			t.Fatalf("patch %s item: %d %s", c.st, w.Code, w.Body.String())
		}
		var q float64
		var note, status string
		if err := f.db.QueryRow(`SELECT quantity, note, status FROM order_items WHERE id = ?`, c.id).Scan(&q, &note, &status); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if q != 3 || note != "sin azúcar" || status != c.st {
			t.Errorf("item %d = qty %v note %q status %q, want 3 / 'sin azúcar' / %s", c.id, q, note, status, c.st)
		}
	}
}

func TestOrder_T21_ReadyAtSetAndPreserved(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	a := f.addItem(oid, cafeID, 1)
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ? AND ready_at IS NULL`, a); n != 1 {
		t.Fatalf("ready_at should start NULL")
	}

	f.reseedUpdatedAt(oid)
	if w := f.patch(oid, a, `{"status":"ready"}`); w.Code != http.StatusOK {
		t.Fatalf("patch ready: %d", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ? AND ready_at IS NOT NULL`, a); n != 1 {
		t.Errorf("ready_at not set after pending -> ready")
	}
	if !f.updatedAtChanged(oid) {
		t.Errorf("updated_at not touched by status change")
	}

	f.exec(`UPDATE order_items SET ready_at = ? WHERE id = ?`, seedStamp, a)
	if w := f.patch(oid, a, `{"status":"ready"}`); w.Code != http.StatusOK {
		t.Fatalf("patch ready again: %d", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ? AND ready_at = ?`, a, seedStamp); n != 1 {
		t.Errorf("ready_at overwritten by repeated ready")
	}

	b := f.addItem(oid, tortaID, 1)
	if w := f.patch(oid, b, `{"status":"preparing"}`); w.Code != http.StatusOK {
		t.Fatalf("patch preparing: %d", w.Code)
	}
	if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ? AND ready_at IS NULL`, b); n != 1 {
		t.Errorf("ready_at set by pending -> preparing")
	}
}

// Q-8 (decision PO): leaving ready for pending/preparing clears ready_at.
func TestOrder_T21b_ReadyAtClearedWhenLeavingReady(t *testing.T) {
	f := newOrderFixture(t)
	oid := f.openOrder(f.mkTable("Mesa 1"))
	for _, st := range []string{"pending", "preparing"} {
		item := f.addItem(oid, cafeID, 1)
		if w := f.patch(oid, item, `{"status":"ready"}`); w.Code != http.StatusOK {
			t.Fatalf("patch ready: %d", w.Code)
		}
		if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ? AND ready_at IS NOT NULL`, item); n != 1 {
			t.Fatalf("ready_at not set")
		}
		if w := f.patch(oid, item, fmt.Sprintf(`{"status":%q}`, st)); w.Code != http.StatusOK {
			t.Fatalf("patch %s: %d", st, w.Code)
		}
		if n := f.count(`SELECT COUNT(*) FROM order_items WHERE id = ? AND ready_at IS NULL`, item); n != 1 {
			t.Errorf("ready_at not NULL after ready -> %s", st)
		}
	}
}

// T-22: SQLITE_BUSY while opening an order must map to 503 (US-081 AC-5, was
// 409), never 500. A real BUSY is produced by holding the write lock on another
// connection (the DSN busy_timeout is 5s, so this test takes about 5s).
func TestOrder_T22_BusyMapsTo503(t *testing.T) {
	db, err := database.New(filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	f := fixtureOn(t, db)
	mesa := f.mkTable("Mesa 1")

	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	if _, err := conn.ExecContext(t.Context(), `BEGIN IMMEDIATE`); err != nil {
		t.Fatalf("begin immediate: %v", err)
	}
	t.Cleanup(func() {
		conn.ExecContext(context.Background(), `ROLLBACK`)
		conn.Close()
	})

	w := f.do(http.MethodPost, "/orders", fmt.Sprintf(`{"table_id":%d}`, mesa))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status under BUSY = %d, want 503; %s", w.Code, w.Body.String())
	}
	if msg := errMsg(t, w); msg != "El sistema está ocupado, reintenta en un momento" {
		t.Errorf("503 error = %q, want msgBusy", msg)
	}
	if _, err := conn.ExecContext(t.Context(), `ROLLBACK`); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if n := f.count(`SELECT COUNT(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}
