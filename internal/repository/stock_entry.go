package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andreyo/pos/internal/model"
)

type stockEntryRepo struct {
	db *sql.DB
}

func NewStockEntryRepository(db *sql.DB) StockEntryRepository {
	return &stockEntryRepo{db: db}
}

// Create inserts a stock entry and increments the product's stock atomically.
func (r *stockEntryRepo) Create(ctx context.Context, entry *model.StockEntry) (*model.StockEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("stock entry begin tx: %w", err)
	}
	defer tx.Rollback()

	// Verify product exists
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM products WHERE id = ? AND active = 1`, entry.ProductID).Scan(&exists); err != nil {
		return nil, ErrNotFound
	}

	// Insert entry
	res, err := tx.ExecContext(ctx,
		`INSERT INTO stock_entries (product_id, quantity, note) VALUES (?, ?, ?)`,
		entry.ProductID, entry.Quantity, entry.Note,
	)
	if err != nil {
		return nil, fmt.Errorf("stock entry insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("stock entry last id: %w", err)
	}

	// Bump product stock
	if _, err := tx.ExecContext(ctx,
		`UPDATE products SET stock = stock + ? WHERE id = ?`,
		entry.Quantity, entry.ProductID,
	); err != nil {
		return nil, fmt.Errorf("stock entry update product: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("stock entry commit: %w", err)
	}

	// Fetch created entry
	created := &model.StockEntry{}
	if err := r.db.QueryRowContext(ctx,
		`SELECT id, product_id, quantity, note, created_at FROM stock_entries WHERE id = ?`, id,
	).Scan(&created.ID, &created.ProductID, &created.Quantity, &created.Note, &created.CreatedAt); err != nil {
		return nil, fmt.Errorf("stock entry fetch: %w", err)
	}
	return created, nil
}
