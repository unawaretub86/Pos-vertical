package service_test

// Parsing and validation tests for the catalog import — contract US-094 §4,
// T-3, T-4, T-6..T-9, T-17..T-19.
//
// The contract (§5) leaves the Go parse entry point to the implementer, so
// these tests do not assume a service signature: they drive the parser
// through POST /products/import (handler.New(db).Routes()) on
// database.New(":memory:"), as §5 allows ("or through the handler only").
// This is an external test package (service_test) so importing the handler
// does not create an import cycle. Parsed values are read back from the
// products table after a commit; errors are read from the dry-run report.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/handler"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
)

// ─── fixture ──────────────────────────────────────────────────────────────

type importEnv struct {
	db     *database.DB
	routes http.Handler
	token  string
}

func newImportEnv(t *testing.T) *importEnv {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	tok, err := appmw.GenerateToken(&model.Claims{UserID: 1, Username: "admin", Role: "admin"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	return &importEnv{db: db, routes: handler.New(db).Routes(), token: tok}
}

type rowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

type report struct {
	Encoding       string            `json:"encoding"`
	Delimiter      string            `json:"delimiter"`
	Headers        []string          `json:"headers"`
	Mapping        map[string]string `json:"mapping"`
	IgnoredHeaders []string          `json:"ignored_headers"`
	TotalRows      int               `json:"total_rows"`
	ValidRows      int               `json:"valid_rows"`
	ErrorCount     int               `json:"error_count"`
	Errors         []rowError        `json:"errors"`
	Valid          bool              `json:"valid"`
}

func csvLines(sep string, lines ...string) []byte {
	return []byte(strings.Join(lines, sep) + sep)
}

func (e *importEnv) post(t *testing.T, content []byte, dryRun bool) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"file_name":      "productos.csv",
		"content_base64": base64.StdEncoding.EncodeToString(content),
		"mapping":        nil,
		"dry_run":        dryRun,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/products/import", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.token)
	w := httptest.NewRecorder()
	e.routes.ServeHTTP(w, req)
	return w
}

func (e *importEnv) dryRun(t *testing.T, content []byte) report {
	t.Helper()
	w := e.post(t, content, true)
	if w.Code != http.StatusOK {
		t.Fatalf("dry run status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var env struct {
		Data report `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode report: %v; body: %s", err, w.Body.String())
	}
	return env.Data
}

func (e *importEnv) commit(t *testing.T, content []byte) {
	t.Helper()
	w := e.post(t, content, false)
	if w.Code != http.StatusCreated {
		t.Fatalf("commit status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func expectHTTPError(t *testing.T, w *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, status, w.Body.String())
	}
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error: %v; body: %s", err, w.Body.String())
	}
	if env.Error != msg {
		t.Fatalf("error = %q, want %q", env.Error, msg)
	}
}

// expectSingleError asserts the report has exactly one error, on row 2.
func expectSingleError(t *testing.T, r report, field, msg string) {
	t.Helper()
	if r.ErrorCount != 1 || len(r.Errors) != 1 || r.Valid {
		t.Fatalf("error_count=%d len=%d valid=%v, want 1/1/false; errors: %+v", r.ErrorCount, len(r.Errors), r.Valid, r.Errors)
	}
	e := r.Errors[0]
	if e.Row != 2 || e.Field != field || e.Message != msg {
		t.Fatalf("error = %+v, want row 2 field %q message %q", e, field, msg)
	}
}

func expectMapping(t *testing.T, r report, want map[string]string) {
	t.Helper()
	got := map[string]string{}
	for k, v := range r.Mapping {
		if v != "" {
			got[k] = v
		}
	}
	if len(got) != len(want) {
		t.Fatalf("mapping = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("mapping[%s] = %q, want %q (full %v)", k, got[k], v, got)
		}
	}
}

// ─── T-3 (AC-3, AC-4) ─────────────────────────────────────────────────────

func TestProductImport_T3_EncodingAndDelimiter(t *testing.T) {
	t.Run("a UTF-8 with BOM and semicolon", func(t *testing.T) {
		e := newImportEnv(t)
		file := append([]byte{0xEF, 0xBB, 0xBF}, csvLines("\r\n",
			"Nombre;Categoría;Precio",
			"Café molido 500 g;Lácteos;8.900",
			"Pan tajado;Panadería;6.500")...)
		r := e.dryRun(t, file)
		if r.Encoding != "utf-8" || r.Delimiter != ";" {
			t.Fatalf("encoding/delimiter = %q/%q, want utf-8/;", r.Encoding, r.Delimiter)
		}
		if len(r.Headers) != 3 || r.Headers[0] != "Nombre" {
			t.Fatalf("headers = %q, want header[0] Nombre without BOM", r.Headers)
		}
		if !r.Valid || r.ValidRows != 2 {
			t.Fatalf("valid=%v valid_rows=%d, want true/2; errors %+v", r.Valid, r.ValidRows, r.Errors)
		}
	})

	t.Run("b Windows-1252", func(t *testing.T) {
		e := newImportEnv(t)
		file := []byte("Nombre;Categor\xeda;Precio\r\nCaf\xe9 molido 500 g;L\xe1cteos;8.900\r\nPan tajado;Panader\xeda;6.500\r\n")
		r := e.dryRun(t, file)
		if r.Encoding != "windows-1252" || r.Delimiter != ";" {
			t.Fatalf("encoding/delimiter = %q/%q, want windows-1252/;", r.Encoding, r.Delimiter)
		}
		e.commit(t, file)
		var cat string
		err := e.db.DB.QueryRow(`SELECT c.name FROM products p JOIN categories c ON c.id = p.category_id WHERE p.name = ?`,
			"Café molido 500 g").Scan(&cat)
		if err != nil {
			t.Fatalf("product «Café molido 500 g» not found decoded: %v", err)
		}
		if cat != "Lácteos" {
			t.Fatalf("category = %q, want Lácteos", cat)
		}
	})

	t.Run("c comma with quoted semicolon", func(t *testing.T) {
		e := newImportEnv(t)
		file := csvLines("\r\n",
			"Nombre,Categoría,Precio",
			`"Aceite; 1 L",Lácteos,8.900`,
			"Pan tajado,Panadería,6.500")
		r := e.dryRun(t, file)
		if r.Encoding != "utf-8" || r.Delimiter != "," {
			t.Fatalf("encoding/delimiter = %q/%q, want utf-8/,", r.Encoding, r.Delimiter)
		}
		e.commit(t, file)
		var n int
		if err := e.db.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE name = 'Aceite; 1 L' AND sale_price = 8900`).Scan(&n); err != nil || n != 1 {
			t.Fatalf("products named «Aceite; 1 L» at 8900 = %d (err %v), want 1", n, err)
		}
	})

	t.Run("d tab", func(t *testing.T) {
		e := newImportEnv(t)
		file := csvLines("\r\n",
			"Nombre\tCategoría\tPrecio",
			"Café molido 500 g\tLácteos\t8.900",
			"Pan tajado\tPanadería\t6.500")
		r := e.dryRun(t, file)
		if r.Delimiter != "\t" || !r.Valid || r.ValidRows != 2 {
			t.Fatalf("delimiter=%q valid=%v valid_rows=%d, want tab/true/2; errors %+v", r.Delimiter, r.Valid, r.ValidRows, r.Errors)
		}
	})

	t.Run("e UTF-16 refused", func(t *testing.T) {
		e := newImportEnv(t)
		file := append([]byte{0xFF, 0xFE}, []byte("N\x00o\x00m\x00b\x00r\x00e\x00")...)
		expectHTTPError(t, e.post(t, file, true), http.StatusBadRequest,
			"formato no compatible: guarda el archivo como CSV UTF-8 o CSV (delimitado por comas)")
	})

	t.Run("f xlsx refused", func(t *testing.T) {
		e := newImportEnv(t)
		file := append([]byte("PK\x03\x04"), []byte("\x14\x00\x06\x00rest-of-zip")...)
		expectHTTPError(t, e.post(t, file, true), http.StatusBadRequest,
			"es un archivo de Excel (.xlsx): ábrelo en Excel y guárdalo como CSV")
	})
}

// ─── T-4 (AC-6, AC-28) ────────────────────────────────────────────────────

func TestProductImport_T4_AutoMapping(t *testing.T) {
	t.Run("synonyms and ignored", func(t *testing.T) {
		e := newImportEnv(t)
		r := e.dryRun(t, csvLines("\r\n",
			"PRODUCTO;EAN;Línea;Precio Venta;Existencia;Proveedor",
			"Arroz;7702;Granos;2.500;10;Acme"))
		expectMapping(t, r, map[string]string{
			"name": "PRODUCTO", "barcode": "EAN", "category": "Línea",
			"sale_price": "Precio Venta", "stock": "Existencia",
		})
		if len(r.IgnoredHeaders) != 1 || r.IgnoredHeaders[0] != "Proveedor" {
			t.Fatalf("ignored_headers = %q, want [Proveedor]", r.IgnoredHeaders)
		}
	})

	t.Run("leftmost wins", func(t *testing.T) {
		e := newImportEnv(t)
		r := e.dryRun(t, csvLines("\r\n", "Nombre;Precio;Valor", "Arroz;2.500;3.000"))
		expectMapping(t, r, map[string]string{"name": "Nombre", "sale_price": "Precio"})
		if len(r.IgnoredHeaders) != 1 || r.IgnoredHeaders[0] != "Valor" {
			t.Fatalf("ignored_headers = %q, want [Valor]", r.IgnoredHeaders)
		}
	})

	t.Run("template header maps all 10 fields", func(t *testing.T) {
		e := newImportEnv(t)
		r := e.dryRun(t, csvLines("\r\n",
			"Nombre;Código de barras;Categoría;Unidad;Costo;Precio de venta;Stock;Stock mínimo;IVA;Código interno",
			"Arroz Diana 500 g;7702001001234;Granos;unidad;1.800;2.500;24;6;0;ARZ-500"))
		expectMapping(t, r, map[string]string{
			"name": "Nombre", "barcode": "Código de barras", "category": "Categoría",
			"unit": "Unidad", "cost": "Costo", "sale_price": "Precio de venta",
			"stock": "Stock", "min_stock": "Stock mínimo", "tax_rate": "IVA", "code": "Código interno",
		})
		if len(r.IgnoredHeaders) != 0 {
			t.Fatalf("ignored_headers = %q, want none", r.IgnoredHeaders)
		}
	})
}

// ─── T-6 (AC-10) ──────────────────────────────────────────────────────────

const (
	msgCents    = "el valor debe ser en pesos enteros, sin centavos"
	msgNegative = "el valor no puede ser negativo"
	msgInvalid  = "valor inválido: escribe solo números, por ejemplo 2.500"
)

func TestProductImport_T6_MoneyParsing(t *testing.T) {
	ok := []struct {
		in   string
		want float64
	}{
		{"2500", 2500}, {"2.500", 2500}, {"$ 2.500", 2500}, {"$2.500,00", 2500},
		{"2,500", 2500}, {"2500.00", 2500}, {"1.234.567", 1234567},
		{"COP 12.000", 12000}, {"12.000 COP", 12000}, {"1", 1}, {"100.000.000", 100000000},
	}
	for _, c := range ok {
		t.Run("sale_price "+c.in, func(t *testing.T) {
			e := newImportEnv(t)
			e.commit(t, csvLines("\r\n", "Nombre;Precio de venta", "Arroz;"+c.in))
			var got float64
			if err := e.db.DB.QueryRow(`SELECT sale_price FROM products WHERE name = 'Arroz'`).Scan(&got); err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != c.want {
				t.Fatalf("sale_price(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}

	bad := []struct{ in, msg string }{
		{"2.500,50", msgCents}, {"2.5", msgCents},
		{"-2.500", msgNegative}, {"-$2.500", msgNegative}, {"$ -2.500", msgNegative}, {"-0", msgNegative},
		{"-abc", msgInvalid}, {"-2.500,50", msgCents}, {"--2500", msgInvalid}, {"abc", msgInvalid},
		{"100.000.001", "el valor supera $100.000.000"},
		{"0", "el precio de venta debe ser mayor que 0"},
		{"", "el precio de venta es obligatorio"},
	}
	for _, c := range bad {
		t.Run("sale_price error "+c.in, func(t *testing.T) {
			e := newImportEnv(t)
			r := e.dryRun(t, csvLines("\r\n", "Nombre;Precio de venta;Costo", "Arroz;"+c.in+";100"))
			expectSingleError(t, r, "sale_price", c.msg)
		})
	}

	for _, in := range []string{"", "0"} {
		t.Run("cost "+in, func(t *testing.T) {
			e := newImportEnv(t)
			e.commit(t, csvLines("\r\n", "Nombre;Precio de venta;Costo", "Arroz;1000;"+in))
			var got float64
			if err := e.db.DB.QueryRow(`SELECT cost FROM products WHERE name = 'Arroz'`).Scan(&got); err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != 0 {
				t.Fatalf("cost(%q) = %v, want 0", in, got)
			}
		})
	}
}

// ─── T-7 (AC-11) ──────────────────────────────────────────────────────────

const (
	msgQtyInvalid  = "cantidad inválida: escribe solo números, por ejemplo 24 o 2,5"
	msgQtyNegative = "la cantidad no puede ser negativa"
)

func TestProductImport_T7_QuantityParsing(t *testing.T) {
	ok := []struct {
		unit, in string
		want     float64
	}{
		{"unidad", "24", 24}, {"unidad", "1.500", 1500}, {"unidad", "", 0}, {"unidad", "1.000.000", 1000000},
		{"kg", "3,5", 3.5}, {"kg", "3.5", 3.5}, {"kg", "0,250", 0.25},
		{"lb", "1,125", 1.125}, {"litro", "1,125", 1.125},
	}
	for _, c := range ok {
		t.Run(c.unit+" "+c.in, func(t *testing.T) {
			e := newImportEnv(t)
			e.commit(t, csvLines("\r\n", "Nombre;Precio de venta;Unidad;Stock", "Arroz;1000;"+c.unit+";"+c.in))
			var got float64
			if err := e.db.DB.QueryRow(`SELECT stock FROM products WHERE name = 'Arroz'`).Scan(&got); err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != c.want {
				t.Fatalf("stock(%s %q) = %v, want %v", c.unit, c.in, got, c.want)
			}
		})
	}

	bad := []struct{ unit, in, msg string }{
		{"unidad", "2,5", "un producto por unidad necesita una cantidad entera"},
		{"unidad", "-1", msgQtyNegative},
		{"unidad", "-doce", msgQtyInvalid},
		{"unidad", "1.000.001", "la cantidad supera 1.000.000"},
		{"unidad", "doce", msgQtyInvalid},
		{"kg", "2,5555", "máximo 3 decimales"},
		{"kg", "-0,5", msgQtyNegative},
	}
	for _, c := range bad {
		t.Run("error "+c.unit+" "+c.in, func(t *testing.T) {
			e := newImportEnv(t)
			r := e.dryRun(t, csvLines("\r\n", "Nombre;Precio de venta;Unidad;Stock", "Arroz;1000;"+c.unit+";"+c.in))
			expectSingleError(t, r, "stock", c.msg)
		})
	}
}

// ─── T-8 (AC-12) ──────────────────────────────────────────────────────────

func TestProductImport_T8_UnitParsing(t *testing.T) {
	cases := []struct{ in, want string }{
		{"und", "unit"}, {"UNIDAD", "unit"}, {"u", "unit"}, {"", "unit"},
		{"Kilo", "kg"}, {"kilogramos", "kg"}, {"libra", "lb"},
		{"Litro", "liter"}, {"lt", "liter"}, {"l", "liter"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			e := newImportEnv(t)
			e.commit(t, csvLines("\r\n", "Nombre;Precio de venta;Unidad", "Arroz;1000;"+c.in))
			var got string
			if err := e.db.DB.QueryRow(`SELECT unit FROM products WHERE name = 'Arroz'`).Scan(&got); err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != c.want {
				t.Fatalf("unit(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	t.Run("caja", func(t *testing.T) {
		e := newImportEnv(t)
		r := e.dryRun(t, csvLines("\r\n", "Nombre;Precio de venta;Unidad", "Arroz;1000;caja"))
		expectSingleError(t, r, "unit", "unidad inválida: usa unidad, kg, lb o litro")
	})
}

// ─── T-9 (AC-13) ──────────────────────────────────────────────────────────

func TestProductImport_T9_TaxParsing(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{{"19", 0.19}, {"19%", 0.19}, {"5", 0.05}, {"0", 0}, {"", 0}}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			e := newImportEnv(t)
			e.commit(t, csvLines("\r\n", "Nombre;Precio de venta;IVA", "Arroz;1000;"+c.in))
			var got float64
			if err := e.db.DB.QueryRow(`SELECT tax_rate FROM products WHERE name = 'Arroz'`).Scan(&got); err != nil {
				t.Fatalf("read: %v", err)
			}
			if got != c.want {
				t.Fatalf("tax_rate(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
	for _, in := range []string{"16", "0,19"} {
		t.Run("error "+in, func(t *testing.T) {
			e := newImportEnv(t)
			r := e.dryRun(t, csvLines("\r\n", "Nombre;Precio de venta;IVA", "Arroz;1000;"+in))
			expectSingleError(t, r, "tax_rate", "IVA inválido: usa 0, 5 o 19")
		})
	}
}

// ─── T-17 (AC-5) ──────────────────────────────────────────────────────────

func TestProductImport_T17_RowWidth(t *testing.T) {
	header := "Nombre;Precio de venta;Código interno"

	for _, row := range []string{"Sal;1.000", "Sal;1.000;;"} {
		t.Run("valid "+row, func(t *testing.T) {
			e := newImportEnv(t)
			file := csvLines("\r\n", header, row)
			r := e.dryRun(t, file)
			if !r.Valid || r.ErrorCount != 0 || r.ValidRows != 1 {
				t.Fatalf("valid=%v error_count=%d valid_rows=%d, want true/0/1; errors %+v", r.Valid, r.ErrorCount, r.ValidRows, r.Errors)
			}
			e.commit(t, file)
			var code string
			var price float64
			if err := e.db.DB.QueryRow(`SELECT code, sale_price FROM products WHERE name = 'Sal'`).Scan(&code, &price); err != nil {
				t.Fatalf("read: %v", err)
			}
			if code != "" || price != 1000 {
				t.Fatalf("code/price = %q/%v, want \"\"/1000", code, price)
			}
		})
	}

	t.Run("extra non-blank cell", func(t *testing.T) {
		e := newImportEnv(t)
		r := e.dryRun(t, csvLines("\r\n", header, "Sal;1.000;;extra"))
		if r.ErrorCount != 1 || len(r.Errors) != 1 {
			t.Fatalf("error_count=%d, want 1; errors %+v", r.ErrorCount, r.Errors)
		}
		if r.Errors[0].Row != 2 || r.Errors[0].Message != "la fila tiene más columnas que el encabezado" {
			t.Fatalf("error = %+v, want row 2 «la fila tiene más columnas que el encabezado»", r.Errors[0])
		}
	})
}

// ─── T-18 (AC-5, AC-19) ───────────────────────────────────────────────────

func TestProductImport_T18_PhysicalLineNumbers(t *testing.T) {
	e := newImportEnv(t)
	file := []byte("Nombre;Precio de venta\r\n" + // line 1
		";;\r\n" + // line 2: blank record, skipped
		"\"Pan\r\n" + // line 3: record starts here
		"tajado\";abc\r\n" + // line 4
		"Sal;xyz\r\n") // line 5
	r := e.dryRun(t, file)
	if r.TotalRows != 2 || r.ErrorCount != 2 || len(r.Errors) != 2 {
		t.Fatalf("total_rows=%d error_count=%d, want 2/2; errors %+v", r.TotalRows, r.ErrorCount, r.Errors)
	}
	if r.Errors[0].Row != 3 || r.Errors[1].Row != 5 {
		t.Fatalf("error rows = %d, %d, want 3, 5", r.Errors[0].Row, r.Errors[1].Row)
	}
	for _, er := range r.Errors {
		if er.Field != "sale_price" || er.Message != msgInvalid {
			t.Errorf("error = %+v, want sale_price %q", er, msgInvalid)
		}
	}
}

// ─── T-19 (AC-19) ─────────────────────────────────────────────────────────

func TestProductImport_T19_ErrorListCappedAt200(t *testing.T) {
	e := newImportEnv(t)
	lines := []string{"Nombre;Precio de venta"}
	for i := 1; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("Producto %03d;abc", i))
	}
	r := e.dryRun(t, csvLines("\r\n", lines...))
	if r.ErrorCount != 300 || len(r.Errors) != 200 || r.Valid || r.TotalRows != 300 || r.ValidRows != 0 {
		t.Fatalf("error_count=%d len(errors)=%d valid=%v total=%d valid_rows=%d, want 300/200/false/300/0",
			r.ErrorCount, len(r.Errors), r.Valid, r.TotalRows, r.ValidRows)
	}
	if r.Errors[0].Row != 2 || r.Errors[199].Row != 201 {
		t.Fatalf("first/last listed rows = %d/%d, want 2/201 (sorted by row)", r.Errors[0].Row, r.Errors[199].Row)
	}
}

// ─── R-157 T-4 (AC-7) ─────────────────────────────────────────────────────

func TestProductImport_R157_T4_RejectedCommitSingularMessage(t *testing.T) {
	e := newImportEnv(t)
	file := csvLines("\r\n", "Nombre;Precio de venta", "Pan;abc")
	w := e.post(t, file, false)
	expectHTTPError(t, w, http.StatusUnprocessableEntity, "el archivo tiene 1 error; no se importó nada")
}

func TestProductImport_R157_T4_RejectedCommitPluralMessageUnchanged(t *testing.T) {
	e := newImportEnv(t)
	file := csvLines("\r\n", "Nombre;Precio de venta", "Pan;abc", "Sal;xyz")
	w := e.post(t, file, false)
	expectHTTPError(t, w, http.StatusUnprocessableEntity, "el archivo tiene 2 errores; no se importó nada")
}
