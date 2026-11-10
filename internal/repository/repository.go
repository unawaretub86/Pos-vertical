// Package repository defines data access interfaces and SQLite implementations.
package repository

import (
	"context"
	"time"

	"github.com/andreyo/pos/internal/model"
)

// UserRepository defines operations for user management.
type UserRepository interface {
	List(ctx context.Context) ([]*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Create(ctx context.Context, u *model.User) (*model.User, error)
	Update(ctx context.Context, u *model.User) (*model.User, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	// OnboardAdmin sets full_name, username and password_hash on an existing user in place.
	OnboardAdmin(ctx context.Context, id int64, fullName, username, passwordHash string) (*model.User, error)
	// Deactivate sets active=false; never hard-deletes.
	Deactivate(ctx context.Context, id int64) error
	// CountAdmins returns the number of active admin users.
	CountAdmins(ctx context.Context) (int, error)
}

// CategoryRepository defines CRUD operations for categories.
type CategoryRepository interface {
	Create(ctx context.Context, name string) (*model.Category, error)
	CreateWithDescription(ctx context.Context, name, description string) (*model.Category, error)
	List(ctx context.Context) ([]*model.Category, error)
	GetByID(ctx context.Context, id int64) (*model.Category, error)
	Update(ctx context.Context, id int64, name string) (*model.Category, error)
	UpdateFull(ctx context.Context, id int64, name, description string) (*model.Category, error)
	Delete(ctx context.Context, id int64) error
}

// ProductRepository defines CRUD operations for products.
type ProductRepository interface {
	Create(ctx context.Context, p *model.Product) (*model.Product, error)
	List(ctx context.Context, filter ProductFilter) ([]*model.Product, error)
	GetByID(ctx context.Context, id int64) (*model.Product, error)
	GetByBarcode(ctx context.Context, barcode string) (*model.Product, error)
	Update(ctx context.Context, p *model.Product) (*model.Product, error)
	Delete(ctx context.Context, id int64) error
	LowStock(ctx context.Context) ([]*model.Product, error)
}

// ProductFilter controls listing behavior.
type ProductFilter struct {
	Search     string
	CategoryID *int64
	ActiveOnly bool
}

// ProductImportRepository reads the state the catalog import validates
// against and writes a validated import (US-094).
type ProductImportRepository interface {
	// Lookup returns every product (active or not) and every category.
	Lookup(ctx context.Context) (*CatalogLookup, error)
	// ImportCatalog runs build on the lookup read inside one transaction and
	// writes the plan it returns in that same transaction; an error from
	// build is returned unchanged and nothing is written.
	ImportCatalog(ctx context.Context, userID int64, build func(*CatalogLookup) (*CatalogImportPlan, error)) (*CatalogImportResult, error)
}

// StockEntryRepository handles stock entry persistence.
type StockEntryRepository interface {
	Create(ctx context.Context, entry *model.StockEntry) (*model.StockEntry, error)
}

// ElectronicDocRepository handles persistence of electronic documents.
type ElectronicDocRepository interface {
	Create(ctx context.Context, doc *model.ElectronicDocument) error
	GetBySaleID(ctx context.Context, saleID int64) (*model.ElectronicDocument, error)
	GetByProviderID(ctx context.Context, providerID string) (*model.ElectronicDocument, error)
	Update(ctx context.Context, doc *model.ElectronicDocument) error
	ListByStatus(ctx context.Context, status string, limit int) ([]*model.ElectronicDocument, error)
}

// CustomerRepository defines CRUD operations for customers.
type CustomerRepository interface {
	List(ctx context.Context, search string) ([]*model.CustomerWithBalance, error)
	GetByID(ctx context.Context, id int64) (*model.Customer, error)
	Create(ctx context.Context, c *model.Customer) (*model.Customer, error)
	Update(ctx context.Context, c *model.Customer) (*model.Customer, error)
	Delete(ctx context.Context, id int64) error
	Balance(ctx context.Context, id int64) (float64, error)
	TotalPortfolioBalance(ctx context.Context) (float64, error)
	ListPayments(ctx context.Context, customerID int64) ([]*model.CustomerPayment, error)
	CreatePayment(ctx context.Context, p *model.CustomerPayment) (*model.CustomerPayment, error)
	ListCreditSales(ctx context.Context, customerID int64) ([]*model.Sale, error)
	GetPOSSummary(ctx context.Context, id int64) (*model.CustomerPosSummary, error)
	// IsActive reports whether the customer exists with active = 1.
	IsActive(ctx context.Context, id int64) (bool, error)
	// GetOpeningBalance returns the customer's opening balance or ErrNotFound.
	GetOpeningBalance(ctx context.Context, customerID int64) (*model.CustomerOpeningBalance, error)
	// CreateOpeningBalance inserts the row and its audit_log row in one
	// transaction; ErrOpeningBalanceExists when the customer already has one.
	CreateOpeningBalance(ctx context.Context, ob *model.CustomerOpeningBalance) (*model.CustomerOpeningBalance, error)
	// UpdateOpeningBalance replaces amount, date and note and audits the change
	// as userID in one transaction; ErrNotFound when the customer has none.
	UpdateOpeningBalance(ctx context.Context, ob *model.CustomerOpeningBalance, userID int64) (*model.CustomerOpeningBalance, error)
}

// SupplierRepository defines CRUD operations for suppliers.
type SupplierRepository interface {
	List(ctx context.Context) ([]*model.Supplier, error)
	GetByID(ctx context.Context, id int64) (*model.Supplier, error)
	Create(ctx context.Context, s *model.Supplier) (*model.Supplier, error)
	Update(ctx context.Context, s *model.Supplier) (*model.Supplier, error)
	Delete(ctx context.Context, id int64) error
}

// PurchaseFilter controls listing behavior for GET /api/purchases.
type PurchaseFilter struct {
	SupplierID *int64
	From       *time.Time
	To         *time.Time
}

// PurchaseRepository persists supplier purchases (US-052). Create updates
// products.stock and products.cost and records a stock_entries row in the
// same transaction, so stock history stays a single source of truth.
type PurchaseRepository interface {
	Create(ctx context.Context, p *model.Purchase) (*model.Purchase, error)
	List(ctx context.Context, filter PurchaseFilter) ([]*model.Purchase, error)
}
