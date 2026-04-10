// Package repository defines data access interfaces and SQLite implementations.
package repository

import (
	"context"

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

// StockEntryRepository handles stock entry persistence.
type StockEntryRepository interface {
	Create(ctx context.Context, entry *model.StockEntry) (*model.StockEntry, error)
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
}
