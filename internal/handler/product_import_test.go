package handler

// Tests for POST /products/import — contract US-094 §4, T-1, T-2, T-5,
// T-10..T-16, T-20..T-22. Requests go through h.Routes() so the Auth and
// RequireAdmin middleware are exercised (AC-1). Fixture: setupTestDB
// (dashboard_test.go:14) + literal INSERTs; tokenFor (assistant_test.go:22).
//
// Report and result field names come from AC-19 / AC-21. The contract does
// not fix the shape of a `preview` row; T-10 assumes it uses the AC-6 field
// keys (`barcode`), the same keys as `mapping`.

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
)

// ─── helpers ──────────────────────────────────────────────────────────────

type importRowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

type importReport struct {
	FileName       string            `json:"file_name"`
	Encoding       string            `json:"encoding"`
	Delimiter      string            `json:"delimiter"`
	Headers        []string          `json:"headers"`
	Mapping        map[string]string `json:"mapping"`
	IgnoredHeaders []string          `json:"ignored_headers"`
	TotalRows      int               `json:"total_rows"`
	ValidRows      int               `json:"valid_rows"`
	ErrorCount     int               `json:"error_count"`
	Errors         []importRowError  `json:"errors"`
	NewCategories  []string          `json:"new_categories"`
	Valid          bool              `json:"valid"`
	Preview        []map[string]any  `json:"preview"`
}

type importResult struct {
	ProductsCreated     int   `json:"products_created"`
	CategoriesCreated   int   `json:"categories_created"`
	StockEntriesCreated int   `json:"stock_entries_created"`
	FirstProductID      int64 `json:"first_product_id"`
	LastProductID       int64 `json:"last_product_id"`
}

// importCSV joins lines with CRLF, as Excel writes them.
func importCSV(lines ...string) []byte {
	return []byte(strings.Join(lines, "\r\n") + "\r\n")
}

// importBody builds the AC-1 JSON body. A nil mapping is sent as null.
func importBody(t *testing.T, content []byte, mapping map[string]string, dryRun bool) string {
	t.Helper()
	body := map[string]any{
		"file_name":      "productos.csv",
		"content_base64": base64.StdEncoding.EncodeToString(content),
		"mapping":        nil,
		"dry_run":        dryRun,
	}
	if mapping != nil {
		body["mapping"] = mapping
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal import body: %v", err)
	}
	return string(b)
}

func doImport(h *Handler, token, body string) *httptest.ResponseRecorder {
	return doImportReq(h, http.MethodPost, "/products/import", token, body)
}

func doImportReq(h *Handler, method, path, token, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

func expectStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, want, w.Body.String())
	}
}

func decodeImportReport(t *testing.T, w *httptest.ResponseRecorder) importReport {
	t.Helper()
	expectStatus(t, w, http.StatusOK)
	var env struct {
		Data importReport `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode import report: %v; body: %s", err, w.Body.String())
	}
	return env.Data
}

func decodeImportResult(t *testing.T, w *httptest.ResponseRecorder) importResult {
	t.Helper()
	expectStatus(t, w, http.StatusCreated)
	var env struct {
		Data importResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode import result: %v; body: %s", err, w.Body.String())
	}
	return env.Data
}

// expectImportError checks status and the exact {"error": msg} text.
func expectImportError(t *testing.T, w *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	expectStatus(t, w, status)
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error body: %v; body: %s", err, w.Body.String())
	}
	if env.Error != msg {
		t.Fatalf("error = %q, want %q", env.Error, msg)
	}
}

func countRows(t *testing.T, db *database.DB, table string) int {
	t.Helper()
	var n int
	// table is a constant chosen by the test, never user input.
	if err := db.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func seedExec(t *testing.T, db *database.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.DB.Exec(q, args...); err != nil {
		t.Fatalf("seed exec: %v\nSQL: %s", err, q)
	}
}

// expectCounts asserts products, categories, stock_entries, audit_log.
func expectCounts(t *testing.T, db *database.DB, products, categories, stock, audit int) {
	t.Helper()
	got := [4]int{
		countRows(t, db, "products"),
		countRows(t, db, "categories"),
		countRows(t, db, "stock_entries"),
		countRows(t, db, "audit_log"),
	}
	want := [4]int{products, categories, stock, audit}
	if got != want {
		t.Fatalf("counts (products, categories, stock_entries, audit_log) = %v, want %v", got, want)
	}
}

func errorAtRow(r importReport, row int) (importRowError, bool) {
	for _, e := range r.Errors {
		if e.Row == row {
			return e, true
		}
	}
	return importRowError{}, false
}

func expectRowError(t *testing.T, r importReport, row int, field, msg string) {
	t.Helper()
	e, ok := errorAtRow(r, row)
	if !ok {
		t.Fatalf("no error at row %d; errors: %+v", row, r.Errors)
	}
	if field != "" && e.Field != field {
		t.Errorf("row %d field = %q, want %q", row, e.Field, field)
	}
	if e.Message != msg {
		t.Errorf("row %d message = %q, want %q", row, e.Message, msg)
	}
}

func expectNoRowError(t *testing.T, r importReport, row int) {
	t.Helper()
	if e, ok := errorAtRow(r, row); ok {
		t.Errorf("row %d unexpectedly has error %+v", row, e)
	}
}

// ─── T-1 (AC-1) ───────────────────────────────────────────────────────────

func TestProductImport_T1_AdminOnly(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	body := importBody(t, importCSV("Nombre;Precio de venta", "Arroz;2.500"), nil, true)

	w := doImport(h, tokenFor(t, 2, "cashier"), body)
	expectImportError(t, w, http.StatusForbidden, "forbidden")

	w = doImport(h, "", body)
	expectStatus(t, w, http.StatusUnauthorized)

	r := decodeImportReport(t, doImport(h, tokenFor(t, 1, "admin"), body))
	if !r.Valid {
		t.Fatalf("admin dry run valid = false, want true; errors: %+v", r.Errors)
	}
	if r.TotalRows != 1 || r.ValidRows != 1 || r.ErrorCount != 0 {
		t.Fatalf("total/valid/errors = %d/%d/%d, want 1/1/0", r.TotalRows, r.ValidRows, r.ErrorCount)
	}
}

// ─── T-2 (AC-2) ───────────────────────────────────────────────────────────

func TestProductImport_T2_SizeAndMalformedLimits(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	// Body of exactly 3,145,729 bytes (3 MiB + 1).
	prefix := `{"file_name":"big.csv","dry_run":false,"mapping":null,"content_base64":"`
	suffix := `"}`
	pad := 3145729 - len(prefix) - len(suffix)
	bigBody := prefix + strings.Repeat("A", pad) + suffix
	if len(bigBody) != 3145729 {
		t.Fatalf("test setup: body length %d", len(bigBody))
	}
	expectImportError(t, doImport(h, admin, bigBody), http.StatusRequestEntityTooLarge, "el archivo supera 2 MB")

	// Decoded file of 2,097,153 bytes (2 MiB + 1); its body stays under 3 MiB.
	file := append([]byte("Nombre;Precio de venta\r\n"), []byte(strings.Repeat("a", 2097153-24))...)
	if len(file) != 2097153 {
		t.Fatalf("test setup: file length %d", len(file))
	}
	expectImportError(t, doImport(h, admin, importBody(t, file, nil, false)), http.StatusRequestEntityTooLarge, "el archivo supera 2 MB")

	bad := `{"file_name":"x.csv","content_base64":"%%%","mapping":null,"dry_run":false}`
	expectImportError(t, doImport(h, admin, bad), http.StatusBadRequest, "archivo inválido")

	empty := `{"file_name":"x.csv","content_base64":"","mapping":null,"dry_run":false}`
	expectImportError(t, doImport(h, admin, empty), http.StatusBadRequest, "archivo inválido")

	if n := countRows(t, db, "products"); n != 0 {
		t.Fatalf("products = %d, want 0", n)
	}
}

// ─── T-5 (AC-7, AC-8) ─────────────────────────────────────────────────────

func TestProductImport_T5_ManualMapping(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	file := importCSV("Col A;Col B", "Arroz;2.500")

	expectImportError(t, doImport(h, admin, importBody(t, file, nil, true)),
		http.StatusBadRequest, "faltan columnas obligatorias: nombre, precio de venta")

	r := decodeImportReport(t, doImport(h, admin,
		importBody(t, file, map[string]string{"name": "Col A", "sale_price": "Col B"}, true)))
	if !r.Valid || r.ValidRows != 1 {
		t.Fatalf("explicit mapping: valid=%v valid_rows=%d, want true/1; errors: %+v", r.Valid, r.ValidRows, r.Errors)
	}

	expectImportError(t, doImport(h, admin,
		importBody(t, file, map[string]string{"name": "Col A", "sale_price": "Col A"}, true)),
		http.StatusBadRequest, "la columna «Col A» está asignada a dos campos")

	expectImportError(t, doImport(h, admin,
		importBody(t, file, map[string]string{"name": "Nope", "sale_price": "Col B"}, true)),
		http.StatusBadRequest, "la columna «Nope» no existe en el archivo")

	expectImportError(t, doImport(h, admin,
		importBody(t, file, map[string]string{"precio": "Col B"}, true)),
		http.StatusBadRequest, "campo desconocido: precio")
}

// ─── T-10 (AC-14, AC-15) ──────────────────────────────────────────────────

func TestProductImport_T10_BarcodeRules(t *testing.T) {
	db := setupTestDB(t)
	seedExec(t, db, `INSERT INTO products (id, name, barcode, sale_price, cost, unit, active) VALUES (1, 'Arroz', '7701', 2500, 1800, 'unit', 1)`)
	seedExec(t, db, `INSERT INTO products (id, name, barcode, sale_price, cost, unit, active) VALUES (2, 'Avena', '7709', 4000, 3000, 'unit', 0)`)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	file := importCSV(
		"Nombre;Código de barras;Precio de venta", // line 1
		"P2;7,70123E+12;1000",                     // line 2
		`P3;="0077001";1000`,                      // line 3
		"P4;7702;1000",                            // line 4
		"P5;7701;1000",                            // line 5
		"P6;7709;1000",                            // line 6
		"P7;7702;1000",                            // line 7
	)
	r := decodeImportReport(t, doImport(h, admin, importBody(t, file, nil, true)))

	expectRowError(t, r, 2, "barcode", "Excel convirtió el código de barras a notación científica: formatea la columna como Texto y vuelve a escribirlo")
	expectNoRowError(t, r, 3)
	expectNoRowError(t, r, 4)
	expectRowError(t, r, 5, "barcode", "el código de barras ya existe en el producto «Arroz»")
	expectRowError(t, r, 6, "barcode", "el código de barras ya existe en el producto «Avena» (eliminado)")
	expectRowError(t, r, 7, "barcode", "código de barras repetido en la fila 4")
	if r.ErrorCount != 4 || r.ValidRows != 2 || r.TotalRows != 6 || r.Valid {
		t.Fatalf("error_count/valid_rows/total_rows/valid = %d/%d/%d/%v, want 4/2/6/false",
			r.ErrorCount, r.ValidRows, r.TotalRows, r.Valid)
	}

	found := false
	for _, p := range r.Preview {
		if p["barcode"] == "0077001" {
			found = true
		}
	}
	if !found {
		t.Errorf("preview has no row with barcode \"0077001\" (Excel =\"…\" unwrapped); preview: %+v", r.Preview)
	}

	expectImportError(t, doImport(h, admin, importBody(t, file, nil, false)),
		http.StatusUnprocessableEntity, "el archivo tiene 4 errores; no se importó nada")

	var name, barcode string
	var price, cost float64
	var active int
	if err := db.DB.QueryRow(`SELECT name, barcode, sale_price, cost, active FROM products WHERE id = 1`).
		Scan(&name, &barcode, &price, &cost, &active); err != nil {
		t.Fatalf("read Arroz: %v", err)
	}
	if name != "Arroz" || barcode != "7701" || price != 2500 || cost != 1800 || active != 1 {
		t.Fatalf("Arroz changed: %q %q %v %v %d", name, barcode, price, cost, active)
	}
	if n := countRows(t, db, "products"); n != 2 {
		t.Fatalf("products = %d, want 2", n)
	}
}

// ─── T-11 (AC-9, AC-16, AC-18) ────────────────────────────────────────────

func TestProductImport_T11_NameAndCodeRules(t *testing.T) {
	db := setupTestDB(t)
	seedExec(t, db, `INSERT INTO products (id, name, sale_price, unit, active) VALUES (1, 'Leche Alpina 1 L', 3200, 'unit', 1)`)
	seedExec(t, db, `INSERT INTO products (id, name, sale_price, unit, active) VALUES (2, 'Yogurt', 2800, 'unit', 0)`)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	long := strings.Repeat("a", 201)
	code65 := strings.Repeat("C", 65)
	file := importCSV(
		"Nombre;Precio de venta;Código interno", // line 1
		";1000;",                                // line 2: blank name
		long+";1000;",                           // line 3
		"leche alpina  1 l;1000;",               // line 4
		"Yogurt;1000;",                          // line 5: deleted name does not block
		"Pan;1000;",                             // line 6
		"Sal;1000;"+code65,                      // line 7
		"PAN;1000;",                             // line 8
	)
	r := decodeImportReport(t, doImport(h, admin, importBody(t, file, nil, true)))

	expectRowError(t, r, 2, "name", "el nombre es obligatorio")
	expectRowError(t, r, 3, "name", "el nombre supera 200 caracteres")
	expectRowError(t, r, 4, "name", "ya existe un producto con este nombre")
	expectNoRowError(t, r, 5)
	expectNoRowError(t, r, 6)
	expectRowError(t, r, 7, "code", "el código interno supera 64 caracteres")
	expectRowError(t, r, 8, "name", "producto repetido en la fila 6")
	if r.ErrorCount != 5 || r.ValidRows != 2 || r.TotalRows != 7 {
		t.Fatalf("error_count/valid_rows/total_rows = %d/%d/%d, want 5/2/7", r.ErrorCount, r.ValidRows, r.TotalRows)
	}
}

// ─── T-12 (AC-17, AC-21) ──────────────────────────────────────────────────

func productCategory(t *testing.T, db *database.DB, name string) sql.NullInt64 {
	t.Helper()
	var id sql.NullInt64
	if err := db.DB.QueryRow(`SELECT category_id FROM products WHERE name = ?`, name).Scan(&id); err != nil {
		t.Fatalf("read category of %q: %v", name, err)
	}
	return id
}

func TestProductImport_T12_CategoryMatching(t *testing.T) {
	db := setupTestDB(t)
	seedExec(t, db, `INSERT INTO categories (id, name) VALUES (1, 'Bebidas')`)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	file := importCSV(
		"Nombre;Categoría;Precio de venta",
		"P1;bebidas;1000",
		"P2;BEBIDAS ;1000",
		"P3;Lácteos;1000",
		"P4;lacteos;1000",
		"P5;Aseo;1000",
		"P6;;1000",
	)
	r := decodeImportReport(t, doImport(h, admin, importBody(t, file, nil, true)))
	if len(r.NewCategories) != 2 || r.NewCategories[0] != "Lácteos" || r.NewCategories[1] != "Aseo" {
		t.Fatalf("new_categories = %q, want [Lácteos Aseo]", r.NewCategories)
	}

	res := decodeImportResult(t, doImport(h, admin, importBody(t, file, nil, false)))
	if res.CategoriesCreated != 2 {
		t.Fatalf("categories_created = %d, want 2", res.CategoriesCreated)
	}
	if n := countRows(t, db, "categories"); n != 3 {
		t.Fatalf("categories = %d, want 3", n)
	}
	for _, p := range []string{"P1", "P2"} {
		if c := productCategory(t, db, p); !c.Valid || c.Int64 != 1 {
			t.Errorf("%s category_id = %+v, want 1", p, c)
		}
	}
	c3, c4 := productCategory(t, db, "P3"), productCategory(t, db, "P4")
	if !c3.Valid || c3 != c4 || c3.Int64 == 1 {
		t.Fatalf("P3/P4 category_id = %+v/%+v, want one shared new id", c3, c4)
	}
	var catName string
	if err := db.DB.QueryRow(`SELECT name FROM categories WHERE id = ?`, c3.Int64).Scan(&catName); err != nil {
		t.Fatalf("read category %d: %v", c3.Int64, err)
	}
	if catName != "Lácteos" {
		t.Fatalf("shared category name = %q, want Lácteos", catName)
	}
	if c := productCategory(t, db, "P6"); c.Valid {
		t.Fatalf("P6 category_id = %d, want NULL", c.Int64)
	}
}

func TestProductImport_T12_CategoryLowestIDWins(t *testing.T) {
	db := setupTestDB(t)
	seedExec(t, db, `INSERT INTO categories (id, name) VALUES (1, 'Bebidas')`)
	seedExec(t, db, `INSERT INTO categories (id, name) VALUES (2, 'bebidas')`)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	file := importCSV("Nombre;Categoría;Precio de venta", "Gaseosa;BEBIDAS;2000")
	res := decodeImportResult(t, doImport(h, admin, importBody(t, file, nil, false)))
	if res.CategoriesCreated != 0 {
		t.Fatalf("categories_created = %d, want 0", res.CategoriesCreated)
	}
	if c := productCategory(t, db, "Gaseosa"); !c.Valid || c.Int64 != 1 {
		t.Fatalf("Gaseosa category_id = %+v, want 1", c)
	}
}

// ─── T-13 (AC-19, AC-20) ──────────────────────────────────────────────────

func TestProductImport_T13_DryRunReportAndRejectedCommit(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	lines := []string{"Nombre;Precio de venta;Unidad"}
	for i := 2; i <= 11; i++ {
		price, unit := "1000", "unidad"
		if i == 7 {
			price = "abc"
		}
		if i == 9 {
			unit = "caja"
		}
		lines = append(lines, fmt.Sprintf("Producto %d;%s;%s", i, price, unit))
	}
	file := importCSV(lines...)

	r := decodeImportReport(t, doImport(h, admin, importBody(t, file, nil, true)))
	if r.Valid || r.ErrorCount != 2 || len(r.Errors) != 2 || r.ValidRows != 8 || r.TotalRows != 10 {
		t.Fatalf("valid/error_count/len(errors)/valid_rows/total_rows = %v/%d/%d/%d/%d, want false/2/2/8/10",
			r.Valid, r.ErrorCount, len(r.Errors), r.ValidRows, r.TotalRows)
	}
	e0, e1 := r.Errors[0], r.Errors[1]
	if e0.Row != 7 || e0.Field != "sale_price" || e0.Column != "Precio de venta" {
		t.Errorf("errors[0] = %+v, want row 7 sale_price «Precio de venta»", e0)
	}
	if e1.Row != 9 || e1.Field != "unit" || e1.Column != "Unidad" {
		t.Errorf("errors[1] = %+v, want row 9 unit «Unidad»", e1)
	}

	expectImportError(t, doImport(h, admin, importBody(t, file, nil, false)),
		http.StatusUnprocessableEntity, "el archivo tiene 2 errores; no se importó nada")
	expectCounts(t, db, 0, 0, 0, 0)
}

// ─── T-14 (AC-22) ─────────────────────────────────────────────────────────

func TestProductImport_T14_RollbackOnFailure(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	seedExec(t, db, `CREATE TRIGGER boom BEFORE INSERT ON products WHEN NEW.name = 'Boom' BEGIN SELECT RAISE(ABORT, 'boom'); END`)

	file := importCSV(
		"Nombre;Categoría;Precio de venta;Stock",
		"Arroz;Granos;2.500;24",
		"Boom;;1.000;",
		"Sal;;800;",
	)
	expectImportError(t, doImport(h, admin, importBody(t, file, nil, false)),
		http.StatusInternalServerError, "error importando productos")
	expectCounts(t, db, 0, 0, 0, 0)
}

// ─── T-21 (AC-22) category race ───────────────────────────────────────────

func TestProductImport_T21_CategoryRace(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	file := importCSV("Nombre;Categoría;Precio de venta", "Arroz;Granos;2.500")

	r := decodeImportReport(t, doImport(h, admin, importBody(t, file, nil, true)))
	if len(r.NewCategories) != 1 || r.NewCategories[0] != "Granos" {
		t.Fatalf("new_categories = %q, want [Granos]", r.NewCategories)
	}

	seedExec(t, db, `CREATE TRIGGER race_cat BEFORE INSERT ON categories WHEN NEW.name = 'Granos' BEGIN INSERT INTO categories (name) VALUES ('Granos'); END`)
	w := doImport(h, admin, importBody(t, file, nil, false))
	expectImportError(t, w, http.StatusConflict,
		"otro usuario creó la categoría «Granos» mientras importabas; vuelve a revisar el archivo")
	if strings.Contains(w.Body.String(), "código de barras") {
		t.Fatalf("category conflict body mentions a barcode: %s", w.Body.String())
	}
	expectCounts(t, db, 0, 0, 0, 0)
}

// ─── T-22 (AC-22) barcode race ────────────────────────────────────────────

func TestProductImport_T22_BarcodeRace(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	file := importCSV("Nombre;Código de barras;Categoría;Precio de venta", "Arroz;7702;Granos;2.500")

	seedExec(t, db, `CREATE TRIGGER race_bc BEFORE INSERT ON products WHEN NEW.barcode = '7702' BEGIN INSERT INTO products (name, sale_price, barcode) VALUES ('Otro', 1000, '7702'); END`)
	w := doImport(h, admin, importBody(t, file, nil, false))
	expectImportError(t, w, http.StatusConflict,
		"otro usuario creó un producto con el código de barras 7702 mientras importabas; vuelve a revisar el archivo")
	if strings.Contains(w.Body.String(), "categoría") {
		t.Fatalf("barcode conflict body mentions a category: %s", w.Body.String())
	}
	expectCounts(t, db, 0, 0, 0, 0)
}

// ─── T-15 (AC-21, AC-23) ──────────────────────────────────────────────────

const templateHeader = "Nombre;Código de barras;Categoría;Unidad;Costo;Precio de venta;Stock;Stock mínimo;IVA;Código interno"

func TestProductImport_T15_CommitWritesEverything(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	file := importCSV(
		templateHeader,
		"Arroz Diana 500 g;7702001001234;Granos;und;$1.800;$2.500;24;6;0;ARZ-500",
		"Queso campesino;;Lácteos;kg;14.000;18.000;3,5;1;19;",
		"Bolsa;;;;0;200;0;;;",
	)

	res := decodeImportResult(t, doImport(h, admin, importBody(t, file, nil, false)))
	if res.ProductsCreated != 3 || res.CategoriesCreated != 2 || res.StockEntriesCreated != 2 {
		t.Fatalf("result = %+v, want products 3, categories 2, stock entries 2", res)
	}
	if res.LastProductID-res.FirstProductID != 2 {
		t.Fatalf("last-first product id = %d, want 2", res.LastProductID-res.FirstProductID)
	}

	type row struct {
		id                               int64
		name, code, unit, typ            string
		barcode                          sql.NullString
		categoryID                       sql.NullInt64
		sale, cost, stock, minStock, tax float64
		wholesale                        sql.NullFloat64
		active                           int
	}
	read := func(name string) row {
		var r row
		err := db.DB.QueryRow(`SELECT id, name, code, unit, type, barcode, category_id, sale_price, cost, stock, min_stock, tax_rate, wholesale_price, active FROM products WHERE name = ?`, name).
			Scan(&r.id, &r.name, &r.code, &r.unit, &r.typ, &r.barcode, &r.categoryID, &r.sale, &r.cost, &r.stock, &r.minStock, &r.tax, &r.wholesale, &r.active)
		if err != nil {
			t.Fatalf("read %q: %v", name, err)
		}
		return r
	}

	a := read("Arroz Diana 500 g")
	if a.id != res.FirstProductID {
		t.Errorf("Arroz id = %d, want first_product_id %d", a.id, res.FirstProductID)
	}
	if a.sale != 2500 || a.cost != 1800 || a.stock != 24 || a.minStock != 6 || a.unit != "unit" ||
		a.tax != 0 || a.code != "ARZ-500" || a.barcode.String != "7702001001234" || a.typ != "product" ||
		a.active != 1 || a.wholesale.Valid || !a.categoryID.Valid {
		t.Errorf("Arroz row = %+v", a)
	}
	q := read("Queso campesino")
	if q.stock != 3.5 || q.unit != "kg" || q.tax != 0.19 || q.barcode.Valid || q.sale != 18000 || q.cost != 14000 || q.minStock != 1 {
		t.Errorf("Queso row = %+v", q)
	}
	if q.id != res.FirstProductID+1 {
		t.Errorf("Queso id = %d, want %d (file order)", q.id, res.FirstProductID+1)
	}
	b := read("Bolsa")
	if b.categoryID.Valid || b.stock != 0 || b.cost != 0 || b.sale != 200 || b.unit != "unit" || b.id != res.LastProductID {
		t.Errorf("Bolsa row = %+v", b)
	}

	rows, err := db.DB.Query(`SELECT product_id, quantity, note FROM stock_entries ORDER BY id`)
	if err != nil {
		t.Fatalf("read stock_entries: %v", err)
	}
	type se struct {
		pid  int64
		qty  float64
		note string
	}
	var entries []se
	for rows.Next() {
		var e se
		if err := rows.Scan(&e.pid, &e.qty, &e.note); err != nil {
			t.Fatalf("scan stock entry: %v", err)
		}
		entries = append(entries, e)
	}
	rows.Close()
	if len(entries) != 2 ||
		entries[0] != (se{a.id, 24, "Importación de catálogo"}) ||
		entries[1] != (se{q.id, 3.5, "Importación de catálogo"}) {
		t.Fatalf("stock_entries = %+v, want Arroz 24 and Queso 3.5 with note Importación de catálogo", entries)
	}

	if n := countRows(t, db, "audit_log"); n != 1 {
		t.Fatalf("audit_log rows = %d, want 1", n)
	}
	var action, entity, details string
	var entityID, userID int64
	if err := db.DB.QueryRow(`SELECT action, entity, entity_id, user_id, details FROM audit_log`).
		Scan(&action, &entity, &entityID, &userID, &details); err != nil {
		t.Fatalf("read audit_log: %v", err)
	}
	if action != "product_import" || entity != "product" || entityID != res.FirstProductID || userID != 1 {
		t.Errorf("audit row = %s %s %d user %d", action, entity, entityID, userID)
	}
	var d struct {
		FileName            string `json:"file_name"`
		Rows                int    `json:"rows"`
		ProductsCreated     int    `json:"products_created"`
		CategoriesCreated   int    `json:"categories_created"`
		StockEntriesCreated int    `json:"stock_entries_created"`
		FirstProductID      int64  `json:"first_product_id"`
		LastProductID       int64  `json:"last_product_id"`
	}
	if err := json.Unmarshal([]byte(details), &d); err != nil {
		t.Fatalf("audit details not JSON: %v (%s)", err, details)
	}
	if d.FileName != "productos.csv" || d.Rows != 3 || d.ProductsCreated != 3 || d.CategoriesCreated != 2 ||
		d.StockEntriesCreated != 2 || d.FirstProductID != res.FirstProductID || d.LastProductID != res.LastProductID {
		t.Errorf("audit details = %+v", d)
	}

	w := doImportReq(h, http.MethodGet, "/products/barcode/7702001001234", admin, "")
	expectStatus(t, w, http.StatusOK)
	var one struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &one); err != nil || one.Data.Name != "Arroz Diana 500 g" {
		t.Fatalf("GET by barcode name = %q (err %v), want Arroz Diana 500 g", one.Data.Name, err)
	}

	w = doImportReq(h, http.MethodGet, "/products", admin, "")
	expectStatus(t, w, http.StatusOK)
	var list struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Data) != 3 {
		t.Fatalf("GET /products len = %d (err %v), want 3", len(list.Data), err)
	}

	w = doImportReq(h, http.MethodGet, "/categories", admin, "")
	expectStatus(t, w, http.StatusOK)
	var cats struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cats); err != nil || len(cats.Data) != 2 {
		t.Fatalf("GET /categories len = %d (err %v), want 2", len(cats.Data), err)
	}

	w = doImport(h, admin, importBody(t, file, nil, false))
	expectStatus(t, w, http.StatusUnprocessableEntity)
	expectCounts(t, db, 3, 2, 2, 1)
}

// ─── T-16 (AC-5) ──────────────────────────────────────────────────────────

func manyRows(n int) []byte {
	var sb strings.Builder
	sb.WriteString("Nombre;Precio de venta\r\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "Producto %04d;1000\r\n", i)
	}
	return []byte(sb.String())
}

func TestProductImport_T16_RowLimits(t *testing.T) {
	admin := tokenFor(t, 1, "admin")

	t.Run("5000 rows commit", func(t *testing.T) {
		db := setupTestDB(t)
		h := New(db)
		res := decodeImportResult(t, doImport(h, admin, importBody(t, manyRows(5000), nil, false)))
		if res.ProductsCreated != 5000 {
			t.Fatalf("products_created = %d, want 5000", res.ProductsCreated)
		}
		if n := countRows(t, db, "products"); n != 5000 {
			t.Fatalf("products = %d, want 5000", n)
		}
	})

	t.Run("5001 rows rejected", func(t *testing.T) {
		db := setupTestDB(t)
		h := New(db)
		expectImportError(t, doImport(h, admin, importBody(t, manyRows(5001), nil, false)),
			http.StatusBadRequest, "el archivo supera 5.000 productos; divídelo en partes")
		if n := countRows(t, db, "products"); n != 0 {
			t.Fatalf("products = %d, want 0", n)
		}
	})

	t.Run("header only", func(t *testing.T) {
		db := setupTestDB(t)
		h := New(db)
		expectImportError(t, doImport(h, admin, importBody(t, importCSV("Nombre;Precio de venta"), nil, true)),
			http.StatusBadRequest, "el archivo no tiene filas de productos")
	})
}

// ─── T-20 (AC-20) ─────────────────────────────────────────────────────────

func TestProductImport_T20_CommitRevalidates(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")
	file := importCSV("Nombre;Código de barras;Precio de venta", "Arroz;7702;2.500")

	r := decodeImportReport(t, doImport(h, admin, importBody(t, file, nil, true)))
	if !r.Valid {
		t.Fatalf("dry run valid = false; errors: %+v", r.Errors)
	}

	w := doImportReq(h, http.MethodPost, "/products", admin,
		`{"name":"Otro","barcode":"7702","sale_price":1000,"unit":"unit","type":"product"}`)
	expectStatus(t, w, http.StatusCreated)

	w = doImport(h, admin, importBody(t, file, nil, false))
	expectStatus(t, w, http.StatusUnprocessableEntity)
	if n := countRows(t, db, "products"); n != 1 {
		t.Fatalf("products = %d, want 1 (only Otro)", n)
	}
	var name string
	if err := db.DB.QueryRow(`SELECT name FROM products`).Scan(&name); err != nil || name != "Otro" {
		t.Fatalf("remaining product = %q (err %v), want Otro", name, err)
	}
}
