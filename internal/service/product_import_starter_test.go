package service_test

// Starter catalog tests — contract R-173 (US-103) §4, T-1..T-5b.
//
// The starter file frontend/public/catalogo-base-tienda.csv is read from disk
// (so `go test` needs no frontend build) and driven through the real importer
// via the handler, reusing newImportEnv from product_import_test.go.

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

const starterPath = "../../frontend/public/catalogo-base-tienda.csv"

const starterHeader = "Nombre;Código de barras;Categoría;Unidad;Costo;Precio de venta;Stock;Stock mínimo;IVA;Código interno"

func readStarter(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(starterPath)
	if err != nil {
		t.Fatalf("read starter catalog: %v", err)
	}
	return b
}

// starterRows parses the file independently of the importer: strip the BOM,
// split on CRLF, split on ";". Returns the header line and the data rows.
func starterRows(t *testing.T, b []byte) (string, [][]string) {
	t.Helper()
	s := strings.TrimPrefix(string(b), "\uFEFF")
	lines := strings.Split(s, "\r\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) < 2 {
		t.Fatalf("starter file has %d lines, want header plus data rows", len(lines))
	}
	rows := make([][]string, 0, len(lines)-1)
	for i, l := range lines[1:] {
		cells := strings.Split(l, ";")
		if len(cells) != 10 {
			t.Fatalf("data row %d has %d cells, want 10: %q", i+1, len(cells), l)
		}
		rows = append(rows, cells)
	}
	return lines[0], rows
}

func (e *importEnv) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := e.db.DB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func TestStarterCatalog_T1_DryRunIsValid(t *testing.T) {
	e := newImportEnv(t)
	r := e.dryRun(t, readStarter(t))
	if !r.Valid || r.ErrorCount != 0 {
		t.Fatalf("valid=%v error_count=%d, want true/0; errors: %+v", r.Valid, r.ErrorCount, r.Errors)
	}
	if r.TotalRows != r.ValidRows {
		t.Fatalf("total_rows=%d valid_rows=%d, want equal", r.TotalRows, r.ValidRows)
	}
	if r.TotalRows < 140 {
		t.Fatalf("total_rows=%d, want at least 140", r.TotalRows)
	}
	if len(r.IgnoredHeaders) != 0 {
		t.Fatalf("ignored_headers=%v, want empty", r.IgnoredHeaders)
	}
	if r.Encoding != "utf-8" || r.Delimiter != ";" {
		t.Fatalf("encoding=%q delimiter=%q, want utf-8 and ;", r.Encoding, r.Delimiter)
	}
}

func TestStarterCatalog_T2_CommitCreatesProductsAndCategories(t *testing.T) {
	e := newImportEnv(t)
	file := readStarter(t)
	_, rows := starterRows(t, file)
	cats := map[string]bool{}
	for _, c := range rows {
		if c[2] != "" {
			cats[c[2]] = true
		}
	}
	e.commit(t, file)

	if got := e.count(t, `SELECT COUNT(*) FROM products`); got != len(rows) {
		t.Fatalf("products = %d, want %d (one per data row)", got, len(rows))
	}
	if got := e.count(t, `SELECT COUNT(*) FROM products WHERE active = 1 AND type = 'product'`); got != len(rows) {
		t.Fatalf("active product-type products = %d, want %d", got, len(rows))
	}
	if got := e.count(t, `SELECT COUNT(*) FROM categories`); got != len(cats) {
		t.Fatalf("categories = %d, want %d", got, len(cats))
	}
	if got := e.count(t, `SELECT COUNT(*) FROM stock_entries`); got != 0 {
		t.Fatalf("stock_entries = %d, want 0", got)
	}
	if got := e.count(t, `SELECT COUNT(*) FROM audit_log WHERE action = 'product_import'`); got != 1 {
		t.Fatalf("product_import audit rows = %d, want 1", got)
	}
}

func TestStarterCatalog_T3_FormatShapeAndPrices(t *testing.T) {
	file := readStarter(t)
	if !strings.HasPrefix(string(file), "\uFEFF") {
		t.Fatalf("file does not start with a UTF-8 BOM")
	}
	if len(file) >= 2<<20 {
		t.Fatalf("file is %d bytes, want < 2 MiB", len(file))
	}
	if strings.Contains(strings.ReplaceAll(string(file), "\r\n", ""), "\n") {
		t.Fatalf("file has a bare LF; every line break must be CRLF")
	}
	header, rows := starterRows(t, file)
	if header != starterHeader {
		t.Fatalf("header = %q, want %q", header, starterHeader)
	}
	if len(rows) < 140 || len(rows) > 170 {
		t.Fatalf("data rows = %d, want between 140 and 170", len(rows))
	}
	cats := map[string]bool{}
	names := map[string]int{}
	for i, c := range rows {
		if c[2] != "" {
			cats[c[2]] = true
		}
		key := strings.Join(strings.Fields(strings.ToLower(c[0])), " ")
		if key == "" {
			t.Fatalf("row %d has an empty name", i+1)
		}
		if prev, ok := names[key]; ok {
			t.Fatalf("row %d name %q duplicates row %d", i+1, c[0], prev)
		}
		names[key] = i + 1
		if c[5] != "1000" {
			t.Fatalf("row %d (%s) price cell = %q, want exactly 1000", i+1, c[0], c[5])
		}
	}
	if len(cats) != 12 {
		t.Fatalf("distinct categories = %d, want 12: %v", len(cats), cats)
	}

	e := newImportEnv(t)
	e.commit(t, file)
	if got := e.count(t, `SELECT COUNT(*) FROM products WHERE sale_price != 1000`); got != 0 {
		t.Fatalf("products with sale_price != 1000 = %d, want 0", got)
	}
	if got := e.count(t, `SELECT COUNT(*) FROM products WHERE sale_price = 1000`); got != len(rows) {
		t.Fatalf("products at 1000 = %d, want %d", got, len(rows))
	}
}

func TestStarterCatalog_T4_NoBarcodeNoIVA(t *testing.T) {
	file := readStarter(t)
	_, rows := starterRows(t, file)
	for i, c := range rows {
		if c[1] != "" {
			t.Fatalf("row %d (%s) has barcode %q, want empty", i+1, c[0], c[1])
		}
		if c[8] != "" {
			t.Fatalf("row %d (%s) has IVA %q, want empty", i+1, c[0], c[8])
		}
	}
	e := newImportEnv(t)
	e.commit(t, file)
	if got := e.count(t, `SELECT COUNT(*) FROM products WHERE barcode IS NOT NULL AND barcode != ''`); got != 0 {
		t.Fatalf("products with a barcode = %d, want 0", got)
	}
	if got := e.count(t, `SELECT COUNT(*) FROM products WHERE tax_rate != 0`); got != 0 {
		t.Fatalf("products with tax_rate != 0 = %d, want 0", got)
	}
}

func TestStarterCatalog_T5_ActiveSameNameRejectsAll(t *testing.T) {
	e := newImportEnv(t)
	file := readStarter(t)
	_, rows := starterRows(t, file)
	name := rows[0][0]
	if _, err := e.db.DB.Exec(`INSERT INTO products (name, sale_price, cost, unit, active) VALUES (?, 5000, 0, 'unit', 1)`, name); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	w := e.post(t, file, false)
	expectHTTPError(t, w, http.StatusUnprocessableEntity, "el archivo tiene 1 error; no se importó nada")
	if got := e.count(t, `SELECT COUNT(*) FROM products`); got != 1 {
		t.Fatalf("products = %d, want 1 (nothing imported)", got)
	}
}

func TestStarterCatalog_T5b_InactiveSameNameDoesNotBlock(t *testing.T) {
	e := newImportEnv(t)
	file := readStarter(t)
	_, rows := starterRows(t, file)
	name := rows[0][0]
	if _, err := e.db.DB.Exec(`INSERT INTO products (name, sale_price, cost, unit, active) VALUES (?, 5000, 0, 'unit', 0)`, name); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	e.commit(t, file)
	if got := e.count(t, `SELECT COUNT(*) FROM products`); got != len(rows)+1 {
		t.Fatalf("products = %d, want %d (starter rows + 1 inactive)", got, len(rows)+1)
	}
}
