package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/andreyo/pos/internal/model"
)

type categoryRepo struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryRepo{db: db}
}

func (r *categoryRepo) Create(ctx context.Context, name string) (*model.Category, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (name, description) VALUES (?, ?)`, name, "")
	if err != nil {
		return nil, fmt.Errorf("category create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("category last id: %w", err)
	}
	return &model.Category{ID: id, Name: name, Description: "", CreatedAt: time.Now()}, nil
}

func (r *categoryRepo) CreateWithDescription(ctx context.Context, name, description string) (*model.Category, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (name, description) VALUES (?, ?)`, name, description)
	if err != nil {
		return nil, fmt.Errorf("category create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("category last id: %w", err)
	}
	return &model.Category{ID: id, Name: name, Description: description, CreatedAt: time.Now()}, nil
}

func (r *categoryRepo) List(ctx context.Context) ([]*model.Category, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, COALESCE(description,''), created_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("category list: %w", err)
	}
	defer rows.Close()

	var cats []*model.Category
	for rows.Next() {
		c := &model.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("category scan: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *categoryRepo) GetByID(ctx context.Context, id int64) (*model.Category, error) {
	c := &model.Category{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(description,''), created_at FROM categories WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("category get: %w", err)
	}
	return c, nil
}

func (r *categoryRepo) Update(ctx context.Context, id int64, name string) (*model.Category, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return nil, fmt.Errorf("category update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *categoryRepo) UpdateFull(ctx context.Context, id int64, name, description string) (*model.Category, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name = ?, description = ? WHERE id = ?`, name, description, id)
	if err != nil {
		return nil, fmt.Errorf("category update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *categoryRepo) Delete(ctx context.Context, id int64) error {
	// Check if any products reference this category
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM products WHERE category_id = ?`, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("category delete check: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("la categoría tiene %d producto(s) asignado(s); reasígnalos antes de eliminar", count)
	}

	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("category delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
