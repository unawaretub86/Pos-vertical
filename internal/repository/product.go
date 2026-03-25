package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/andreyo/pos/internal/model"
)

type productRepo struct {
	db *sql.DB
}

// NewProductRepository returns a SQLite-backed ProductRepository.
func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *model.Product) (*model.Product, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO products (name, code, category_id, type, sale_price, wholesale_price, cost, unit, stock, min_stock, tax_rate, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Code, p.CategoryID, p.Type, p.SalePrice, p.WholesalePrice,
		p.Cost, p.Unit, p.Stock, p.MinStock, p.TaxRate, true,
	)
	if err != nil {
		return nil, fmt.Errorf("product create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("product last id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *productRepo) List(ctx context.Context, filter ProductFilter) ([]*model.Product, error) {
	query := `SELECT id, name, code, category_id, type, sale_price, wholesale_price, cost, unit, stock, min_stock, tax_rate, active, created_at FROM products WHERE 1=1`
	args := []any{}

	if filter.ActiveOnly {
		query += " AND active = 1"
	}
	if filter.CategoryID != nil {
		query += " AND category_id = ?"
		args = append(args, *filter.CategoryID)
	}
	if filter.Search != "" {
		query += " AND (name LIKE ? OR code LIKE ?)"
		s := "%" + strings.ToLower(filter.Search) + "%"
		args = append(args, s, s)
	}
	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("product list: %w", err)
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *productRepo) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, code, category_id, type, sale_price, wholesale_price, cost, unit, stock, min_stock, tax_rate, active, created_at
		FROM products WHERE id = ?`, id)
	p, err := scanProduct(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("product get: %w", err)
	}
	return p, nil
}

func (r *productRepo) Update(ctx context.Context, p *model.Product) (*model.Product, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE products SET name=?, code=?, category_id=?, type=?, sale_price=?, wholesale_price=?,
		cost=?, unit=?, stock=?, min_stock=?, tax_rate=?, active=?
		WHERE id=?`,
		p.Name, p.Code, p.CategoryID, p.Type, p.SalePrice, p.WholesalePrice,
		p.Cost, p.Unit, p.Stock, p.MinStock, p.TaxRate, p.Active, p.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("product update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, p.ID)
}

// Delete soft-deletes a product by setting active = 0.
func (r *productRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE products SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("product delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// LowStock returns all active products where stock <= min_stock.
func (r *productRepo) LowStock(ctx context.Context) ([]*model.Product, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, code, category_id, type, sale_price, wholesale_price, cost, unit, stock, min_stock, tax_rate, active, created_at
		FROM products
		WHERE active = 1 AND type = 'product' AND stock <= min_stock
		ORDER BY stock ASC`)
	if err != nil {
		return nil, fmt.Errorf("product low stock: %w", err)
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// scanner abstracts *sql.Row and *sql.Rows for scanProduct.
type scanner interface {
	Scan(dest ...any) error
}

func scanProduct(s scanner) (*model.Product, error) {
	p := &model.Product{}
	return p, s.Scan(
		&p.ID, &p.Name, &p.Code, &p.CategoryID, &p.Type,
		&p.SalePrice, &p.WholesalePrice, &p.Cost, &p.Unit,
		&p.Stock, &p.MinStock, &p.TaxRate, &p.Active, &p.CreatedAt,
	)
}
