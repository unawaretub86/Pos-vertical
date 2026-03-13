// Package repository defines data access interfaces and SQLite implementations.
package repository

import (
	"context"

	"github.com/andreyo/pos/internal/model"
)

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
	Update(ctx context.Context, p *model.Product) (*model.Product, error)
	Delete(ctx context.Context, id int64) error
}

// ProductFilter controls listing behavior.
type ProductFilter struct {
	Search     string
	CategoryID *int64
	ActiveOnly bool
}
