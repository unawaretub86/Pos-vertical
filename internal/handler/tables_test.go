package handler

// Contrato US-055 §4, T-5 a T-15 (API de mesas). Las FK no se aplican (R-56):
// ningún test depende de errores de FK.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type tableJSON struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Zone          string  `json:"zone"`
	Seats         int     `json:"seats"`
	SortOrder     int     `json:"sort_order"`
	Active        bool    `json:"active"`
	State         string  `json:"state"`
	OpenOrderID   *int64  `json:"open_order_id"`
	OpenTotal     float64 `json:"open_total"`
	OpenedMinutes float64 `json:"opened_minutes"`
}

func decodeTable(t *testing.T, w *httptest.ResponseRecorder) tableJSON {
	t.Helper()
	var env struct {
		Data tableJSON `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, w.Body.String())
	}
	return env.Data
}

func listTablesJSON(t *testing.T, h *Handler, tok string) ([]tableJSON, string) {
	t.Helper()
	w := doAuth(h, tok, http.MethodGet, "/tables", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /tables status = %d, body %s", w.Code, w.Body.String())
	}
	var env struct {
		Data []tableJSON `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, w.Body.String())
	}
	return env.Data, w.Body.String()
}

func errMsg(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
		t.Fatalf("decode error body: %v\n%s", err, w.Body.String())
	}
	return e.Error
}

func byName(list []tableJSON, name string) tableJSON {
	for _, x := range list {
		if x.Name == name {
			return x
		}
	}
	return tableJSON{}
}

func TestTables_T5_CreateDefaultsAndValues(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	w := doAuth(h, tok, http.MethodPost, "/tables", `{"name":"Mesa 1"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeTable(t, w)
	if got.Name != "Mesa 1" || got.Zone != "" || got.Seats != 0 || got.SortOrder != 0 || !got.Active || got.ID == 0 {
		t.Errorf("got %+v, want Mesa 1, zone '', seats 0, sort 0, active true, id>0", got)
	}

	w = doAuth(h, tok, http.MethodPost, "/tables", `{"name":"Mesa 2","zone":"Terraza","sort_order":2}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got = decodeTable(t, w)
	if got.Name != "Mesa 2" || got.Zone != "Terraza" || got.SortOrder != 2 || !got.Active {
		t.Errorf("got %+v, want Mesa 2 / Terraza / sort 2 / active", got)
	}
}

func TestTables_T6_EmptyNameRejected(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	w := doAuth(h, tok, http.MethodPost, "/tables", `{"name":"   "}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %s", w.Code, w.Body.String())
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM dining_tables`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("COUNT(*) = %d, want 0", n)
	}
}

func TestTables_T7_DuplicateNameConflict(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	if w := doAuth(h, tok, http.MethodPost, "/tables", `{"name":"Mesa 1"}`); w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, body %s", w.Code, w.Body.String())
	}
	w := doAuth(h, tok, http.MethodPost, "/tables", `{"name":"Mesa 1"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409; body %s", w.Code, w.Body.String())
	}
	msg := errMsg(t, w)
	if msg == "" || strings.Contains(msg, "UNIQUE constraint") {
		t.Errorf("message %q must be non-empty and not expose SQL", msg)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM dining_tables`).Scan(&n)
	if n != 1 {
		t.Errorf("COUNT(*) = %d, want 1", n)
	}

	// A deactivated table still blocks its name (UNIQUE persists).
	if _, err := db.Exec(`UPDATE dining_tables SET active = 0 WHERE name = 'Mesa 1'`); err != nil {
		t.Fatal(err)
	}
	w = doAuth(h, tok, http.MethodPost, "/tables", `{"name":"Mesa 1"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("create over deactivated status = %d, want 409", w.Code)
	}
}

func TestTables_T8_Update(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)
	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (2, 'Mesa 2')`)

	w := doAuth(h, tok, http.MethodPut, "/tables/1", `{"name":"Barra 1","zone":"Interior","sort_order":5,"seats":4}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeTable(t, w)
	if got.ID != 1 || got.Name != "Barra 1" || got.Zone != "Interior" || got.SortOrder != 5 || got.Seats != 4 {
		t.Errorf("got %+v", got)
	}

	w = doAuth(h, tok, http.MethodPut, "/tables/1", `{"name":"Mesa 2"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("rename to existing status = %d, want 409", w.Code)
	}

	w = doAuth(h, tok, http.MethodPut, "/tables/999", `{"name":"Nada"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown id status = %d, want 404", w.Code)
	}
}

func TestTables_T9_DeleteIsLogical(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)
	w := doAuth(h, tok, http.MethodDelete, "/tables/1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var active, n int
	db.QueryRow(`SELECT active FROM dining_tables WHERE id = 1`).Scan(&active)
	db.QueryRow(`SELECT COUNT(*) FROM dining_tables`).Scan(&n)
	if active != 0 || n != 1 {
		t.Errorf("active=%d count=%d, want 0 and 1", active, n)
	}

	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (2, 'Mesa 2')`)
	db.Exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (1, 'table', 2, 'closed', 1)`)
	db.Exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (2, 'table', 2, 'cancelled', 1)`)
	w = doAuth(h, tok, http.MethodDelete, "/tables/2", "")
	if w.Code != http.StatusOK {
		t.Errorf("delete with closed+cancelled orders status = %d, want 200; body %s", w.Code, w.Body.String())
	}
}

func TestTables_T10_DeleteBlockedByOpenOrder(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)
	db.Exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (1, 'table', 1, 'open', 1)`)
	w := doAuth(h, tok, http.MethodDelete, "/tables/1", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body %s", w.Code, w.Body.String())
	}
	if msg := errMsg(t, w); msg != "La mesa tiene un pedido abierto" {
		t.Errorf("message = %q", msg)
	}
	var active int
	db.QueryRow(`SELECT active FROM dining_tables WHERE id = 1`).Scan(&active)
	if active != 1 {
		t.Errorf("active = %d, want 1", active)
	}
}

func TestTables_T11_ListOrderAndEmpty(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	list, raw := listTablesJSON(t, h, tok)
	if len(list) != 0 || !strings.Contains(raw, `"data":[]`) {
		t.Errorf("empty list: got %d items, body %s; want data []", len(list), raw)
	}

	db.Exec(`INSERT INTO dining_tables (id, name, sort_order) VALUES (1, 'Mesa 2', 2)`)
	db.Exec(`INSERT INTO dining_tables (id, name, sort_order) VALUES (2, 'Mesa 1', 1)`)
	db.Exec(`INSERT INTO dining_tables (id, name, sort_order) VALUES (3, 'Mesa 3', 1)`)
	db.Exec(`INSERT INTO dining_tables (id, name, sort_order, active) VALUES (4, 'Mesa X', 0, 0)`)

	list, raw = listTablesJSON(t, h, tok)
	if len(list) != 3 {
		t.Fatalf("got %d tables, want 3; %s", len(list), raw)
	}
	want := []string{"Mesa 1", "Mesa 3", "Mesa 2"}
	for i, w := range want {
		if list[i].Name != w {
			t.Errorf("pos %d = %q, want %q", i, list[i].Name, w)
		}
		if list[i].State != "libre" || list[i].OpenOrderID != nil {
			t.Errorf("%s: state=%q open_order_id=%v, want libre/null", list[i].Name, list[i].State, list[i].OpenOrderID)
		}
	}
	if !strings.Contains(raw, `"open_order_id":null`) {
		t.Errorf("open_order_id must serialize as null: %s", raw)
	}
}

func seedRestaurantProducts(t *testing.T, h *Handler, exec func(string, ...any)) {
	t.Helper()
	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (1, 'Cafe', 'C', 4500, 0, 'unit', 1)`)
	exec(`INSERT INTO products (id, name, code, sale_price, cost, unit, active) VALUES (2, 'Torta', 'T', 8000, 0, 'unit', 1)`)
}

func TestTables_T12_DerivedStates(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")
	exec := func(q string, args ...any) {
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	seedRestaurantProducts(t, h, exec)

	for i, n := range []string{"A", "B", "C", "D"} {
		exec(`INSERT INTO dining_tables (id, name, sort_order) VALUES (?, ?, ?)`, i+1, "Mesa "+n, i)
	}
	// A: 2 ready items -> por_cobrar, 4500*2 + 8000 = 17000
	exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (1, 'table', 1, 'open', 1)`)
	exec(`INSERT INTO order_items (order_id, product_id, quantity, unit_price, status) VALUES (1, 1, 2, 4500, 'ready')`)
	exec(`INSERT INTO order_items (order_id, product_id, quantity, unit_price, status) VALUES (1, 2, 1, 8000, 'ready')`)
	// B: pending + ready -> ocupada
	exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (2, 'table', 2, 'open', 1)`)
	exec(`INSERT INTO order_items (order_id, product_id, quantity, unit_price, status) VALUES (2, 1, 1, 4500, 'pending')`)
	exec(`INSERT INTO order_items (order_id, product_id, quantity, unit_price, status) VALUES (2, 2, 1, 8000, 'ready')`)
	// D: open, no items -> ocupada, total 0
	exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (3, 'table', 4, 'open', 1)`)

	list, raw := listTablesJSON(t, h, tok)
	a, b, c, d := byName(list, "Mesa A"), byName(list, "Mesa B"), byName(list, "Mesa C"), byName(list, "Mesa D")
	if a.State != "por_cobrar" || a.OpenTotal != 17000 || a.OpenOrderID == nil || *a.OpenOrderID != 1 {
		t.Errorf("A = %+v, want por_cobrar/17000/order 1; %s", a, raw)
	}
	if b.State != "ocupada" {
		t.Errorf("B state = %q, want ocupada", b.State)
	}
	if c.State != "libre" || c.OpenOrderID != nil {
		t.Errorf("C = %+v, want libre/null", c)
	}
	if d.State != "ocupada" || d.OpenTotal != 0 {
		t.Errorf("D = %+v, want ocupada/0", d)
	}
}

func TestTables_T13_VoidAndMixedStatuses(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")
	exec := func(q string, args ...any) {
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	seedRestaurantProducts(t, h, exec)

	for i, n := range []string{"V", "DR", "DP", "AV"} {
		exec(`INSERT INTO dining_tables (id, name, sort_order) VALUES (?, ?, ?)`, i+1, n, i)
		exec(`INSERT INTO orders (id, type, table_id, status, user_id) VALUES (?, 'table', ?, 'open', 1)`, i+1, i+1)
	}
	ins := func(order, prod int, qty, price float64, status string) {
		exec(`INSERT INTO order_items (order_id, product_id, quantity, unit_price, status) VALUES (?, ?, ?, ?, ?)`, order, prod, qty, price, status)
	}
	// V: void torta + ready cafe x2 -> total 9000, por_cobrar
	ins(1, 2, 1, 8000, "void")
	ins(1, 1, 2, 4500, "ready")
	// DR: delivered + ready -> por_cobrar
	ins(2, 1, 1, 4500, "delivered")
	ins(2, 2, 1, 8000, "ready")
	// DP: delivered + preparing -> ocupada
	ins(3, 1, 1, 4500, "delivered")
	ins(3, 2, 1, 8000, "preparing")
	// AV: all void -> ocupada, total 0 (P-4)
	ins(4, 1, 1, 4500, "void")

	list, raw := listTablesJSON(t, h, tok)
	if v := byName(list, "V"); v.OpenTotal != 9000 || v.State != "por_cobrar" {
		t.Errorf("V = %+v, want total 9000 por_cobrar; %s", v, raw)
	}
	if x := byName(list, "DR"); x.State != "por_cobrar" {
		t.Errorf("DR state = %q, want por_cobrar", x.State)
	}
	if x := byName(list, "DP"); x.State != "ocupada" {
		t.Errorf("DP state = %q, want ocupada", x.State)
	}
	if x := byName(list, "AV"); x.State != "ocupada" || x.OpenTotal != 0 {
		t.Errorf("AV = %+v, want ocupada/0", x)
	}
}

func TestTables_T14_OpenedMinutesUTC(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)
	db.Exec(`INSERT INTO orders (id, type, table_id, status, user_id, created_at) VALUES (1, 'table', 1, 'open', 1, datetime('now','-25 minutes'))`)

	list, raw := listTablesJSON(t, h, tok)
	if len(list) != 1 {
		t.Fatalf("got %d tables; %s", len(list), raw)
	}
	if m := list[0].OpenedMinutes; m < 24 || m > 25 {
		t.Errorf("opened_minutes = %v, want between 24 and 25", m)
	}
}

func TestTables_T15_Permissions(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	cash := tokenFor(t, 2, "cashier")
	admin := tokenFor(t, 1, "admin")

	db.Exec(`INSERT INTO dining_tables (id, name) VALUES (1, 'Mesa 1')`)

	cases := []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/tables", "", http.StatusOK},
		{http.MethodGet, "/setup", "", http.StatusOK},
		{http.MethodPost, "/tables", `{"name":"Mesa 9"}`, http.StatusForbidden},
		{http.MethodPut, "/tables/1", `{"name":"Otra"}`, http.StatusForbidden},
		{http.MethodDelete, "/tables/1", "", http.StatusForbidden},
	}
	for _, c := range cases {
		if w := doAuth(h, cash, c.method, c.path, c.body); w.Code != c.want {
			t.Errorf("cashier %s %s = %d, want %d", c.method, c.path, w.Code, c.want)
		}
	}
	if w := doAuth(h, admin, http.MethodPost, "/tables", `{"name":"Mesa 9"}`); w.Code != http.StatusCreated {
		t.Errorf("admin POST /tables = %d, want 201", w.Code)
	}
	var active int
	db.QueryRow(`SELECT active FROM dining_tables WHERE id = 1`).Scan(&active)
	if active != 1 {
		t.Errorf("cashier DELETE must not deactivate: active = %d", active)
	}
}
