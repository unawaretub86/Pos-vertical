package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CatalogProductRef is an existing product as the catalog import sees it:
// every product, active or deleted, because the barcode index covers both.
type CatalogProductRef struct {
	Name    string
	Barcode string // "" when the product has no barcode
	Active  bool
}

// CatalogCategoryRef is an existing category.
type CatalogCategoryRef struct {
	ID   int64
	Name string
}

// CatalogLookup is the database state the import validates against.
// Categories are ordered by id.
type CatalogLookup struct {
	Products   []CatalogProductRef
	Categories []CatalogCategoryRef
}

// CatalogImportProduct is one validated row ready to insert.
type CatalogImportProduct struct {
	Name       string
	Code       string
	Barcode    string // "" is stored as NULL
	CategoryID *int64 // existing category
	// NewCategory names a category of CatalogImportPlan.NewCategories that this
	// import creates; it wins over CategoryID when set.
	NewCategory string
	Unit        string
	SalePrice   float64
	Cost        float64
	Stock       float64
	MinStock    float64
	TaxRate     float64
}

// CatalogImportPlan is everything one import writes.
type CatalogImportPlan struct {
	FileName      string
	NewCategories []string
	Products      []CatalogImportProduct
}

// CatalogImportResult is what one committed import created.
type CatalogImportResult struct {
	ProductsCreated     int   `json:"products_created"`
	CategoriesCreated   int   `json:"categories_created"`
	StockEntriesCreated int   `json:"stock_entries_created"`
	FirstProductID      int64 `json:"first_product_id"`
	LastProductID       int64 `json:"last_product_id"`
}

// catalogImportStockNote is the stock_entries note of the imported stock.
const catalogImportStockNote = "Importación de catálogo"

// ErrCatalogBarcodeTaken is returned when another product with the same
// barcode was created after the validation and before the insert.
type ErrCatalogBarcodeTaken struct{ Barcode string }

func (e *ErrCatalogBarcodeTaken) Error() string {
	return fmt.Sprintf("otro usuario creó un producto con el código de barras %s mientras importabas; vuelve a revisar el archivo", e.Barcode)
}

// ErrCatalogCategoryTaken is returned when another category with the same
// name was created after the validation and before the insert.
type ErrCatalogCategoryTaken struct{ Name string }

func (e *ErrCatalogCategoryTaken) Error() string {
	return fmt.Sprintf("otro usuario creó la categoría «%s» mientras importabas; vuelve a revisar el archivo", e.Name)
}

type productImportRepo struct {
	db *sql.DB
}

// NewProductImportRepository returns a SQLite-backed ProductImportRepository.
func NewProductImportRepository(db *sql.DB) ProductImportRepository {
	return &productImportRepo{db: db}
}

type rowsQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (r *productImportRepo) Lookup(ctx context.Context) (*CatalogLookup, error) {
	return loadCatalogLookup(ctx, r.db)
}

func loadCatalogLookup(ctx context.Context, q rowsQueryer) (*CatalogLookup, error) {
	lk := &CatalogLookup{}

	rows, err := q.QueryContext(ctx, `SELECT name, COALESCE(barcode, ''), active FROM products ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("catalog lookup products: %w", mapBusy(err))
	}
	for rows.Next() {
		var p CatalogProductRef
		if err := rows.Scan(&p.Name, &p.Barcode, &p.Active); err != nil {
			rows.Close()
			return nil, fmt.Errorf("catalog lookup product scan: %w", err)
		}
		lk.Products = append(lk.Products, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("catalog lookup products: %w", err)
	}
	rows.Close()

	rows, err = q.QueryContext(ctx, `SELECT id, name FROM categories ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("catalog lookup categories: %w", mapBusy(err))
	}
	defer rows.Close()
	for rows.Next() {
		var c CatalogCategoryRef
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, fmt.Errorf("catalog lookup category scan: %w", err)
		}
		lk.Categories = append(lk.Categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("catalog lookup categories: %w", err)
	}
	return lk, nil
}

// catalogWriteError maps a failed insert: SQLITE_BUSY to ErrBusy, a unique
// failure on the given constraint to conflict, anything else wrapped.
func catalogWriteError(err error, constraint string, conflict error, what string) error {
	if mapped := mapBusy(err); errors.Is(mapped, ErrBusy) {
		return mapped
	}
	if isUniqueViolation(err) && strings.Contains(err.Error(), constraint) {
		return conflict
	}
	return fmt.Errorf("%s: %w", what, err)
}

// ImportCatalog opens one transaction, loads the lookup inside it, lets build
// validate the file against it and, when build returns a plan, writes the new
// categories, the products (in plan order), one stock entry per product with
// stock > 0 and one product_import audit row. Every read and write uses the
// transaction; any error rolls everything back.
func (r *productImportRepo) ImportCatalog(ctx context.Context, userID int64, build func(*CatalogLookup) (*CatalogImportPlan, error)) (*CatalogImportResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("catalog import begin: %w", mapBusy(err))
	}
	defer tx.Rollback()

	lk, err := loadCatalogLookup(ctx, tx)
	if err != nil {
		return nil, err
	}
	plan, err := build(lk)
	if err != nil {
		return nil, err
	}
	if len(plan.Products) == 0 {
		return nil, errors.New("catalog import: empty plan")
	}

	res := &CatalogImportResult{}
	newCatIDs := make(map[string]int64, len(plan.NewCategories))
	for _, name := range plan.NewCategories {
		out, err := tx.ExecContext(ctx, `INSERT INTO categories (name, description) VALUES (?, '')`, name)
		if err != nil {
			return nil, catalogWriteError(err, "categories.name", &ErrCatalogCategoryTaken{Name: name}, "catalog import category insert")
		}
		id, err := out.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("catalog import category id: %w", err)
		}
		newCatIDs[name] = id
		res.CategoriesCreated++
	}

	prodStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO products (name, code, barcode, category_id, type, sale_price, wholesale_price, cost, unit, stock, min_stock, tax_rate, active)
		VALUES (?, ?, ?, ?, 'product', ?, NULL, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return nil, fmt.Errorf("catalog import prepare product: %w", mapBusy(err))
	}
	defer prodStmt.Close()
	entryStmt, err := tx.PrepareContext(ctx, `INSERT INTO stock_entries (product_id, quantity, note) VALUES (?, ?, ?)`)
	if err != nil {
		return nil, fmt.Errorf("catalog import prepare stock entry: %w", mapBusy(err))
	}
	defer entryStmt.Close()

	for _, p := range plan.Products {
		categoryID := p.CategoryID
		if p.NewCategory != "" {
			id, ok := newCatIDs[p.NewCategory]
			if !ok {
				return nil, fmt.Errorf("catalog import: category %q not in plan", p.NewCategory)
			}
			categoryID = &id
		}
		out, err := prodStmt.ExecContext(ctx, p.Name, p.Code, toNullString(p.Barcode), categoryID,
			p.SalePrice, p.Cost, p.Unit, p.Stock, p.MinStock, p.TaxRate)
		if err != nil {
			return nil, catalogWriteError(err, "products.barcode", &ErrCatalogBarcodeTaken{Barcode: p.Barcode}, "catalog import product insert")
		}
		id, err := out.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("catalog import product id: %w", err)
		}
		if res.ProductsCreated == 0 {
			res.FirstProductID = id
		}
		res.LastProductID = id
		res.ProductsCreated++

		// products.stock already holds the quantity; the entry only traces it.
		if p.Stock > 0 {
			if _, err := entryStmt.ExecContext(ctx, id, p.Stock, catalogImportStockNote); err != nil {
				return nil, fmt.Errorf("catalog import stock entry insert: %w", mapBusy(err))
			}
			res.StockEntriesCreated++
		}
	}

	details, err := json.Marshal(map[string]any{
		"file_name":             plan.FileName,
		"rows":                  len(plan.Products),
		"products_created":      res.ProductsCreated,
		"categories_created":    res.CategoriesCreated,
		"stock_entries_created": res.StockEntriesCreated,
		"first_product_id":      res.FirstProductID,
		"last_product_id":       res.LastProductID,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog import audit details: %w", err)
	}
	if err := InsertAuditLog(ctx, tx, userID, "product_import", "product", res.FirstProductID, string(details)); err != nil {
		return nil, mapBusy(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("catalog import commit: %w", mapBusy(err))
	}
	return res, nil
}
