package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/andreyo/pos/internal/repository"
)

// Catalog import from a CSV file (US-094). The file is decoded, parsed and
// mapped to product fields here; every row is validated against the
// database lookup; a commit writes everything or nothing.

const (
	importMaxFileBytes = 2 << 20 // 2 MiB decoded file
	importMaxRows      = 5000
	importMaxErrors    = 200
	importPreviewRows  = 5
	importMaxMoney     = 100_000_000
	importMaxQty       = 1_000_000
)

// ErrImportTooLarge is returned when the decoded file exceeds 2 MiB.
var ErrImportTooLarge = errors.New("el archivo supera 2 MB")

// ImportRejectedError is returned by a commit whose file has row errors.
type ImportRejectedError struct{ Count int }

func (e *ImportRejectedError) Error() string {
	if e.Count == 1 {
		return "el archivo tiene 1 error; no se importó nada"
	}
	return fmt.Sprintf("el archivo tiene %d errores; no se importó nada", e.Count)
}

// ProductImportRequest is one uploaded file. A nil Mapping means automatic
// column recognition; a non-nil one is {field: header}.
type ProductImportRequest struct {
	FileName string
	Content  []byte
	Mapping  map[string]string
}

// ProductImportError is one problem in one cell (or row) of the file.
type ProductImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

// ProductImportReport is the dry-run answer.
type ProductImportReport struct {
	FileName       string               `json:"file_name"`
	Encoding       string               `json:"encoding"`
	Delimiter      string               `json:"delimiter"`
	Headers        []string             `json:"headers"`
	Mapping        map[string]string    `json:"mapping"`
	IgnoredHeaders []string             `json:"ignored_headers"`
	TotalRows      int                  `json:"total_rows"`
	ValidRows      int                  `json:"valid_rows"`
	ErrorCount     int                  `json:"error_count"`
	Errors         []ProductImportError `json:"errors"`
	NewCategories  []string             `json:"new_categories"`
	Valid          bool                 `json:"valid"`
	Preview        []map[string]any     `json:"preview"`
}

// ProductImportService validates and imports product catalogs.
type ProductImportService struct {
	repo repository.ProductImportRepository
}

// NewProductImportService returns a ProductImportService.
func NewProductImportService(repo repository.ProductImportRepository) *ProductImportService {
	return &ProductImportService{repo: repo}
}

// DryRun validates the file against the current database and writes nothing.
func (s *ProductImportService) DryRun(ctx context.Context, req ProductImportRequest) (*ProductImportReport, error) {
	file, err := parseCatalogFile(req.Content, req.Mapping)
	if err != nil {
		return nil, err
	}
	lk, err := s.repo.Lookup(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog import lookup: %w", err)
	}
	report, _ := validateCatalog(file, lk)
	report.FileName = req.FileName
	return report, nil
}

// Commit re-validates the file inside the import transaction and writes it
// only when no row has an error (*ImportRejectedError otherwise).
func (s *ProductImportService) Commit(ctx context.Context, userID int64, req ProductImportRequest) (*repository.CatalogImportResult, error) {
	file, err := parseCatalogFile(req.Content, req.Mapping)
	if err != nil {
		return nil, err
	}
	res, err := s.repo.ImportCatalog(ctx, userID, func(lk *repository.CatalogLookup) (*repository.CatalogImportPlan, error) {
		report, plan := validateCatalog(file, lk)
		if report.ErrorCount > 0 {
			return nil, &ImportRejectedError{Count: report.ErrorCount}
		}
		plan.FileName = req.FileName
		return plan, nil
	})
	if err != nil {
		return nil, fmt.Errorf("catalog import: %w", err)
	}
	return res, nil
}

// ─── fields and header mapping ────────────────────────────────────────────

type importField struct {
	key      string
	label    string
	synonyms []string
}

// importFields is in the AC-6 table order, which is also the error sort
// order and the order of the "missing columns" message.
var importFields = []importField{
	{"name", "nombre", []string{"nombre", "producto", "descripcion", "nombre del producto", "articulo", "name", "product"}},
	{"sale_price", "precio de venta", []string{"precio de venta", "precio venta", "precio", "pvp", "valor", "valor venta", "price", "sale price"}},
	{"barcode", "código de barras", []string{"codigo de barras", "codigo barras", "cod barras", "barcode", "ean", "ean13", "upc"}},
	{"category", "categoría", []string{"categoria", "linea", "grupo", "departamento", "familia", "category"}},
	{"unit", "unidad", []string{"unidad", "unidad de medida", "um", "medida", "unit"}},
	{"cost", "costo", []string{"costo", "precio de compra", "precio compra", "costo unitario", "cost"}},
	{"stock", "stock", []string{"stock", "existencia", "existencias", "cantidad", "inventario", "stock actual", "qty"}},
	{"min_stock", "stock mínimo", []string{"stock minimo", "minimo", "existencia minima", "min stock"}},
	{"tax_rate", "IVA", []string{"iva", "impuesto", "tax", "porcentaje iva"}},
	{"code", "código interno", []string{"codigo", "codigo interno", "referencia", "ref", "sku", "code"}},
}

var (
	importFieldOrder = map[string]int{}
	importSynonyms   = map[string]string{}
)

func init() {
	for i, f := range importFields {
		importFieldOrder[f.key] = i
		for _, s := range f.synonyms {
			importSynonyms[s] = f.key
		}
	}
}

var importNormReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	".", " ", "_", " ", "-", " ", "/", " ", "%", " ", "(", " ", ")", " ", ":", " ",
)

// normalizeImportText is the AC-6 normalization, also used to compare
// product and category names.
func normalizeImportText(s string) string {
	s = importNormReplacer.Replace(strings.ToLower(strings.TrimSpace(s)))
	return strings.Join(strings.Fields(s), " ")
}

// collapseSpaces trims and turns every run of whitespace into one space.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// ─── file decoding and parsing ────────────────────────────────────────────

type catalogRow struct {
	line  int
	cells []string
}

type catalogFile struct {
	encoding  string
	delimiter rune
	headers   []string
	rows      []catalogRow
	columns   map[string]int    // field → header index
	mapping   map[string]string // field → header text
	ignored   []string
}

// win1252 maps bytes 0x80-0x9F; every other byte is its Latin-1 code point.
var win1252 = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

func decodeWindows1252(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b) + len(b)/8)
	for _, c := range b {
		if c >= 0x80 && c <= 0x9F {
			sb.WriteRune(win1252[c-0x80])
		} else {
			sb.WriteRune(rune(c))
		}
	}
	return sb.String()
}

// detectDelimiter counts ;, comma and tab outside quotes in the first
// non-blank line; ties go to ; then comma then tab.
func detectDelimiter(text string) rune {
	start := 0
	for start < len(text) {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text) - start
		}
		if strings.TrimSpace(text[start:start+end]) != "" {
			break
		}
		start += end + 1
	}
	counts := map[rune]int{}
	inQuote := false
	for _, c := range text[min(start, len(text)):] {
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		if c == '\n' {
			break
		}
		counts[c]++
	}
	best := ';'
	for _, d := range []rune{',', '\t'} {
		if counts[d] > counts[best] {
			best = d
		}
	}
	return best
}

func allBlank(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// parseCatalogFile decodes and parses the file and resolves the column
// mapping. Its errors are file-level: *ValidationError or ErrImportTooLarge.
func parseCatalogFile(content []byte, mapping map[string]string) (*catalogFile, error) {
	if len(content) == 0 {
		return nil, invalid("archivo inválido")
	}
	if len(content) > importMaxFileBytes {
		return nil, ErrImportTooLarge
	}
	if bytes.HasPrefix(content, []byte{0xFF, 0xFE}) || bytes.HasPrefix(content, []byte{0xFE, 0xFF}) {
		return nil, invalid("formato no compatible: guarda el archivo como CSV UTF-8 o CSV (delimitado por comas)")
	}
	if bytes.HasPrefix(content, []byte("PK\x03\x04")) {
		return nil, invalid("es un archivo de Excel (.xlsx): ábrelo en Excel y guárdalo como CSV")
	}
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	f := &catalogFile{encoding: "utf-8"}
	var text string
	if utf8.Valid(content) {
		text = string(content)
	} else {
		f.encoding = "windows-1252"
		text = decodeWindows1252(content)
	}
	f.delimiter = detectDelimiter(text)

	r := csv.NewReader(strings.NewReader(text))
	r.Comma = f.delimiter
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	haveHeader := false
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, invalid("archivo inválido")
		}
		if allBlank(rec) {
			continue
		}
		if !haveHeader {
			haveHeader = true
			f.headers = make([]string, len(rec))
			for i, h := range rec {
				f.headers[i] = strings.TrimSpace(h)
			}
			continue
		}
		line, _ := r.FieldPos(0)
		f.rows = append(f.rows, catalogRow{line: line, cells: rec})
		if len(f.rows) > importMaxRows {
			return nil, invalid("el archivo supera 5.000 productos; divídelo en partes")
		}
	}
	if len(f.rows) == 0 {
		return nil, invalid("el archivo no tiene filas de productos")
	}

	if err := f.resolveMapping(mapping); err != nil {
		return nil, err
	}
	var missing []string
	for _, fd := range importFields[:2] { // name and sale_price are required
		if _, ok := f.columns[fd.key]; !ok {
			missing = append(missing, fd.label)
		}
	}
	if len(missing) > 0 {
		return nil, invalid("faltan columnas obligatorias: " + strings.Join(missing, ", "))
	}
	return f, nil
}

func (f *catalogFile) resolveMapping(mapping map[string]string) error {
	f.columns = map[string]int{}
	f.mapping = map[string]string{}
	used := map[int]bool{}

	if mapping == nil {
		for i, h := range f.headers {
			if h == "" {
				continue
			}
			key, ok := importSynonyms[normalizeImportText(h)]
			if ok {
				if _, taken := f.columns[key]; !taken {
					f.columns[key] = i
					f.mapping[key] = h
					used[i] = true
				}
			}
		}
	} else {
		var unknown []string
		for k := range mapping {
			if _, ok := importFieldOrder[k]; !ok {
				unknown = append(unknown, k)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			return invalid("campo desconocido: " + unknown[0])
		}
		assigned := map[string]bool{}
		for _, fd := range importFields {
			h := strings.TrimSpace(mapping[fd.key])
			if h == "" {
				continue
			}
			idx := -1
			for i, fh := range f.headers {
				if fh == h {
					idx = i
					break
				}
			}
			if idx < 0 {
				return invalid(fmt.Sprintf("la columna «%s» no existe en el archivo", h))
			}
			if assigned[h] {
				return invalid(fmt.Sprintf("la columna «%s» está asignada a dos campos", h))
			}
			assigned[h] = true
			f.columns[fd.key] = idx
			f.mapping[fd.key] = h
			used[idx] = true
		}
	}

	f.ignored = []string{}
	for i, h := range f.headers {
		if h != "" && !used[i] {
			f.ignored = append(f.ignored, h)
		}
	}
	return nil
}

// ─── cell parsers (each returns "" or the Spanish error message) ──────────

var (
	reDigits         = regexp.MustCompile(`^\d+$`)
	reMoneyGroupDot  = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
	reMoneyGroupCom  = regexp.MustCompile(`^\d{1,3}(,\d{3})+$`)
	reMoneyDecimal   = regexp.MustCompile(`^(\d+|\d{1,3}(?:\.\d{3})+|\d{1,3}(?:,\d{3})+)[.,](\d{1,2})$`)
	reQtyThousands   = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
	reQtyComma       = regexp.MustCompile(`^(\d+|\d{1,3}(?:\.\d{3})+),(\d+)$`)
	reQtyDot         = regexp.MustCompile(`^(\d+)\.(\d+)$`)
	reScientificCode = regexp.MustCompile(`^\d+([.,]\d+)?[eE]\+?\d+$`)
)

const (
	msgMoneyInvalid  = "valor inválido: escribe solo números, por ejemplo 2.500"
	msgMoneyCents    = "el valor debe ser en pesos enteros, sin centavos"
	msgMoneyNegative = "el valor no puede ser negativo"
	msgMoneyTooBig   = "el valor supera $100.000.000"
	msgQtyInvalid    = "cantidad inválida: escribe solo números, por ejemplo 24 o 2,5"
)

// parseImportMoney reads a non-blank peso amount (AC-10).
func parseImportMoney(raw string) (int64, string) {
	s := strings.NewReplacer(" ", "", " ", "", "\t", "").Replace(raw)
	up := strings.ToUpper(s)
	if strings.HasPrefix(up, "COP") {
		s = s[3:]
	} else if strings.HasSuffix(up, "COP") {
		s = s[:len(s)-3]
	}
	s = strings.ReplaceAll(s, "$", "")

	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	var digits string
	switch {
	case reDigits.MatchString(s):
		digits = s
	case reMoneyGroupDot.MatchString(s), reMoneyGroupCom.MatchString(s):
		digits = strings.NewReplacer(".", "", ",", "").Replace(s)
	default:
		m := reMoneyDecimal.FindStringSubmatch(s)
		if m == nil {
			return 0, msgMoneyInvalid
		}
		if strings.Trim(m[2], "0") != "" {
			return 0, msgMoneyCents
		}
		digits = strings.NewReplacer(".", "", ",", "").Replace(m[1])
	}
	if negative {
		return 0, msgMoneyNegative
	}
	v, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || v > importMaxMoney {
		return 0, msgMoneyTooBig
	}
	return v, ""
}

// parseImportQty reads a quantity (AC-11); blank is 0.
func parseImportQty(raw, unit string) (float64, string) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	if s == "" {
		return 0, ""
	}
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	var intPart, decPart string
	switch {
	case reDigits.MatchString(s):
		intPart = s
	case reQtyThousands.MatchString(s):
		intPart = strings.ReplaceAll(s, ".", "")
	default:
		if m := reQtyComma.FindStringSubmatch(s); m != nil {
			intPart, decPart = strings.ReplaceAll(m[1], ".", ""), m[2]
		} else if m := reQtyDot.FindStringSubmatch(s); m != nil {
			intPart, decPart = m[1], m[2]
		} else {
			return 0, msgQtyInvalid
		}
	}
	if negative {
		return 0, "la cantidad no puede ser negativa"
	}
	decPart = strings.TrimRight(decPart, "0")
	if len(decPart) > 3 {
		return 0, "máximo 3 decimales"
	}
	num := intPart
	if decPart != "" {
		num += "." + decPart
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v > importMaxQty {
		return 0, "la cantidad supera 1.000.000"
	}
	if unit == "unit" && decPart != "" {
		return 0, "un producto por unidad necesita una cantidad entera"
	}
	return v, ""
}

var importUnits = map[string]string{
	"": "unit", "unit": "unit", "unidad": "unit", "und": "unit", "un": "unit", "u": "unit", "unidades": "unit",
	"kg": "kg", "kilo": "kg", "kilos": "kg", "kilogramo": "kg", "kilogramos": "kg",
	"lb": "lb", "libra": "lb", "libras": "lb",
	"liter": "liter", "litro": "liter", "litros": "liter", "lt": "liter", "l": "liter",
}

// parseImportTax reads IVA 0, 5 or 19 (optionally with %) as a fraction.
func parseImportTax(raw string) (float64, string) {
	s := strings.TrimSuffix(strings.NewReplacer(" ", "", " ", "").Replace(raw), "%")
	switch s {
	case "", "0":
		return 0, ""
	case "5":
		return 0.05, ""
	case "19":
		return 0.19, ""
	}
	return 0, "IVA inválido: usa 0, 5 o 19"
}

// parseImportBarcode trims and unwraps Excel's ="…" text form.
func parseImportBarcode(raw string) (string, string) {
	s := strings.TrimSpace(raw)
	if len(s) >= 3 && strings.HasPrefix(s, `="`) && strings.HasSuffix(s, `"`) {
		s = strings.TrimSpace(s[2 : len(s)-1])
	}
	if s == "" {
		return "", ""
	}
	if reScientificCode.MatchString(s) {
		return "", "Excel convirtió el código de barras a notación científica: formatea la columna como Texto y vuelve a escribirlo"
	}
	if utf8.RuneCountInString(s) > 64 {
		return "", "el código de barras supera 64 caracteres"
	}
	return s, ""
}

// ─── row validation ───────────────────────────────────────────────────────

// validateCatalog checks every row against the lookup and returns the
// report and, when no row has an error, the plan to write.
func validateCatalog(f *catalogFile, lk *repository.CatalogLookup) (*ProductImportReport, *repository.CatalogImportPlan) {
	report := &ProductImportReport{
		Encoding:       f.encoding,
		Delimiter:      string(f.delimiter),
		Headers:        f.headers,
		Mapping:        f.mapping,
		IgnoredHeaders: f.ignored,
		TotalRows:      len(f.rows),
		Errors:         []ProductImportError{},
		NewCategories:  []string{},
		Preview:        []map[string]any{},
	}

	existingBarcodes := map[string]repository.CatalogProductRef{}
	activeNames := map[string]bool{}
	for _, p := range lk.Products {
		if p.Barcode != "" {
			if _, ok := existingBarcodes[p.Barcode]; !ok {
				existingBarcodes[p.Barcode] = p
			}
		}
		if p.Active {
			activeNames[normalizeImportText(p.Name)] = true
		}
	}
	existingCats := map[string]int64{} // normalized name → lowest id
	for _, c := range lk.Categories {
		key := normalizeImportText(c.Name)
		if _, ok := existingCats[key]; !ok {
			existingCats[key] = c.ID
		}
	}

	newCats := map[string]string{} // normalized name → spelling of first row
	fileBarcodes := map[string]int{}
	fileNames := map[string]int{}
	var allErrors []ProductImportError
	plan := &repository.CatalogImportPlan{}

	for _, row := range f.rows {
		var rowErrs []ProductImportError
		addErr := func(field, msg string) {
			col := ""
			if idx, ok := f.columns[field]; ok {
				col = f.headers[idx]
			}
			rowErrs = append(rowErrs, ProductImportError{Row: row.line, Field: field, Column: col, Message: msg})
		}
		cell := func(field string) string {
			idx, ok := f.columns[field]
			if !ok || idx >= len(row.cells) {
				return ""
			}
			return row.cells[idx]
		}

		p := repository.CatalogImportProduct{}
		category := "" // as written in the file, for the preview

		// name
		p.Name = collapseSpaces(cell("name"))
		switch {
		case p.Name == "":
			addErr("name", "el nombre es obligatorio")
		case utf8.RuneCountInString(p.Name) > 200:
			addErr("name", "el nombre supera 200 caracteres")
		default:
			key := normalizeImportText(p.Name)
			if activeNames[key] {
				addErr("name", "ya existe un producto con este nombre")
			} else if first, ok := fileNames[key]; ok {
				addErr("name", fmt.Sprintf("producto repetido en la fila %d", first))
			}
		}
		if p.Name != "" {
			if key := normalizeImportText(p.Name); key != "" {
				if _, ok := fileNames[key]; !ok {
					fileNames[key] = row.line
				}
			}
		}

		// sale_price
		if raw := strings.TrimSpace(cell("sale_price")); raw == "" {
			addErr("sale_price", "el precio de venta es obligatorio")
		} else if v, msg := parseImportMoney(raw); msg != "" {
			addErr("sale_price", msg)
		} else if v == 0 {
			addErr("sale_price", "el precio de venta debe ser mayor que 0")
		} else {
			p.SalePrice = float64(v)
		}

		// barcode
		if bc, msg := parseImportBarcode(cell("barcode")); msg != "" {
			addErr("barcode", msg)
		} else if bc != "" {
			p.Barcode = bc
			if ex, ok := existingBarcodes[bc]; ok {
				m := fmt.Sprintf("el código de barras ya existe en el producto «%s»", ex.Name)
				if !ex.Active {
					m += " (eliminado)"
				}
				addErr("barcode", m)
			} else if first, ok := fileBarcodes[bc]; ok {
				addErr("barcode", fmt.Sprintf("código de barras repetido en la fila %d", first))
			}
			if _, ok := fileBarcodes[bc]; !ok {
				fileBarcodes[bc] = row.line
			}
		}

		// category
		if cat := collapseSpaces(cell("category")); cat != "" {
			category = cat
			key := normalizeImportText(cat)
			if id, ok := existingCats[key]; ok {
				p.CategoryID = &id
			} else {
				spelled, ok := newCats[key]
				if !ok {
					spelled = cat
					newCats[key] = cat
					report.NewCategories = append(report.NewCategories, cat)
				}
				p.NewCategory = spelled
			}
		}

		// unit
		unit, ok := importUnits[normalizeImportText(cell("unit"))]
		if !ok {
			addErr("unit", "unidad inválida: usa unidad, kg, lb o litro")
		}
		p.Unit = unit

		// cost
		if raw := strings.TrimSpace(cell("cost")); raw != "" {
			if v, msg := parseImportMoney(raw); msg != "" {
				addErr("cost", msg)
			} else {
				p.Cost = float64(v)
			}
		}

		// stock, min_stock
		if v, msg := parseImportQty(cell("stock"), unit); msg != "" {
			addErr("stock", msg)
		} else {
			p.Stock = v
		}
		if v, msg := parseImportQty(cell("min_stock"), unit); msg != "" {
			addErr("min_stock", msg)
		} else {
			p.MinStock = v
		}

		// tax_rate
		if v, msg := parseImportTax(cell("tax_rate")); msg != "" {
			addErr("tax_rate", msg)
		} else {
			p.TaxRate = v
		}

		// code
		p.Code = strings.TrimSpace(cell("code"))
		if utf8.RuneCountInString(p.Code) > 64 {
			addErr("code", "el código interno supera 64 caracteres")
		}

		// a non-blank cell beyond the header width
		for i := len(f.headers); i < len(row.cells); i++ {
			if strings.TrimSpace(row.cells[i]) != "" {
				rowErrs = append(rowErrs, ProductImportError{Row: row.line, Message: "la fila tiene más columnas que el encabezado"})
				break
			}
		}

		if len(rowErrs) > 0 {
			allErrors = append(allErrors, rowErrs...)
			continue
		}
		report.ValidRows++
		plan.Products = append(plan.Products, p)
		if len(report.Preview) < importPreviewRows {
			report.Preview = append(report.Preview, previewRow(p, category))
		}
	}

	sort.SliceStable(allErrors, func(i, j int) bool {
		if allErrors[i].Row != allErrors[j].Row {
			return allErrors[i].Row < allErrors[j].Row
		}
		return errorFieldOrder(allErrors[i].Field) < errorFieldOrder(allErrors[j].Field)
	})
	report.ErrorCount = len(allErrors)
	if len(allErrors) > importMaxErrors {
		allErrors = allErrors[:importMaxErrors]
	}
	report.Errors = append(report.Errors, allErrors...)
	report.Valid = report.ErrorCount == 0
	plan.NewCategories = report.NewCategories
	return report, plan
}

// errorFieldOrder sorts row-level errors (no field) after the field errors.
func errorFieldOrder(field string) int {
	if i, ok := importFieldOrder[field]; ok {
		return i
	}
	return len(importFields)
}

func previewRow(p repository.CatalogImportProduct, category string) map[string]any {
	return map[string]any{
		"name":       p.Name,
		"sale_price": p.SalePrice,
		"barcode":    p.Barcode,
		"category":   category,
		"unit":       p.Unit,
		"cost":       p.Cost,
		"stock":      p.Stock,
		"min_stock":  p.MinStock,
		"tax_rate":   p.TaxRate,
		"code":       p.Code,
	}
}
