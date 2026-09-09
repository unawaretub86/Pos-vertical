package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/andreyo/pos/internal/model"
)

type supplierRepo struct {
	db *sql.DB
}

func NewSupplierRepository(db *sql.DB) SupplierRepository {
	return &supplierRepo{db: db}
}

func (r *supplierRepo) List(ctx context.Context) ([]*model.Supplier, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, COALESCE(phone,''), COALESCE(notes,''), created_at FROM suppliers ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("supplier list: %w", err)
	}
	defer rows.Close()

	var suppliers []*model.Supplier
	for rows.Next() {
		s := &model.Supplier{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Phone, &s.Notes, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("supplier scan: %w", err)
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, rows.Err()
}

func (r *supplierRepo) GetByID(ctx context.Context, id int64) (*model.Supplier, error) {
	s := &model.Supplier{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(phone,''), COALESCE(notes,''), created_at FROM suppliers WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Phone, &s.Notes, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("supplier get: %w", err)
	}
	return s, nil
}

func (r *supplierRepo) Create(ctx context.Context, s *model.Supplier) (*model.Supplier, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO suppliers (name, phone, notes) VALUES (?, ?, ?)`, s.Name, s.Phone, s.Notes)
	if err != nil {
		return nil, fmt.Errorf("supplier create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("supplier last id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *supplierRepo) Update(ctx context.Context, s *model.Supplier) (*model.Supplier, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE suppliers SET name = ?, phone = ?, notes = ? WHERE id = ?`, s.Name, s.Phone, s.Notes, s.ID)
	if err != nil {
		return nil, fmt.Errorf("supplier update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, s.ID)
}

func (r *supplierRepo) Delete(ctx context.Context, id int64) error {
	var count int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM purchases WHERE supplier_id = ?`, id).Scan(&count); err != nil {
		return fmt.Errorf("supplier delete check: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("el proveedor tiene %d compra(s) registrada(s); no se puede eliminar", count)
	}

	res, err := r.db.ExecContext(ctx, `DELETE FROM suppliers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("supplier delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
