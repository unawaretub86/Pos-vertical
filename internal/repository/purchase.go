package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/andreyo/pos/internal/model"
)

type purchaseRepo struct {
	db *sql.DB
}

func NewPurchaseRepository(db *sql.DB) PurchaseRepository {
	return &purchaseRepo{db: db}
}

// Create inserts the purchase and its items, and — per item — bumps the
// product's stock and overwrites its cost with the purchase's unit_cost
// (last-cost, not weighted average; see US-052 scope note). A stock_entries
// row is added per item so the existing stock-history view picks it up.
// All in one transaction: a bad item must not leave stock partially updated.
func (r *purchaseRepo) Create(ctx context.Context, p *model.Purchase) (*model.Purchase, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("purchase begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var supplierExists bool
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM suppliers WHERE id = ?`, p.SupplierID).Scan(&supplierExists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("purchase supplier check: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO purchases (supplier_id, user_id, reference, total) VALUES (?, ?, ?, ?)`,
		p.SupplierID, p.UserID, p.Reference, p.Total)
	if err != nil {
		return nil, fmt.Errorf("purchase insert: %w", err)
	}
	purchaseID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("purchase last id: %w", err)
	}

	for _, item := range p.Items {
		var productExists bool
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM products WHERE id = ?`, item.ProductID).Scan(&productExists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("producto %d no encontrado", item.ProductID)
			}
			return nil, fmt.Errorf("purchase item product check: %w", err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO purchase_items (purchase_id, product_id, quantity, unit_cost, subtotal) VALUES (?, ?, ?, ?, ?)`,
			purchaseID, item.ProductID, item.Quantity, item.UnitCost, item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("purchase item insert: %w", err)
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE products SET stock = stock + ?, cost = ? WHERE id = ?`,
			item.Quantity, item.UnitCost, item.ProductID,
		); err != nil {
			return nil, fmt.Errorf("purchase item update product: %w", err)
		}

		note := fmt.Sprintf("Compra #%d", purchaseID)
		if p.Reference != "" {
			note += " · " + p.Reference
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO stock_entries (product_id, quantity, note) VALUES (?, ?, ?)`,
			item.ProductID, item.Quantity, note,
		); err != nil {
			return nil, fmt.Errorf("purchase stock entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("purchase commit: %w", err)
	}

	return r.getByID(ctx, purchaseID)
}

func (r *purchaseRepo) getByID(ctx context.Context, id int64) (*model.Purchase, error) {
	p := &model.Purchase{}
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.supplier_id, s.name, p.user_id, COALESCE(p.reference,''), p.total, p.created_at
		FROM purchases p JOIN suppliers s ON s.id = p.supplier_id
		WHERE p.id = ?`, id,
	).Scan(&p.ID, &p.SupplierID, &p.SupplierName, &p.UserID, &p.Reference, &p.Total, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("purchase get: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT pi.id, pi.purchase_id, pi.product_id, pr.name, pi.quantity, pi.unit_cost, pi.subtotal
		FROM purchase_items pi JOIN products pr ON pr.id = pi.product_id
		WHERE pi.purchase_id = ?`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("purchase items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item := model.PurchaseItem{}
		if err := rows.Scan(&item.ID, &item.PurchaseID, &item.ProductID, &item.ProductName, &item.Quantity, &item.UnitCost, &item.Subtotal); err != nil {
			return nil, fmt.Errorf("purchase item scan: %w", err)
		}
		p.Items = append(p.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *purchaseRepo) List(ctx context.Context, filter PurchaseFilter) ([]*model.Purchase, error) {
	query := `
		SELECT p.id, p.supplier_id, s.name, p.user_id, COALESCE(p.reference,''), p.total, p.created_at
		FROM purchases p JOIN suppliers s ON s.id = p.supplier_id
		WHERE 1=1`
	args := []any{}

	if filter.SupplierID != nil {
		query += " AND p.supplier_id = ?"
		args = append(args, *filter.SupplierID)
	}
	if filter.From != nil {
		query += " AND p.created_at >= ?"
		args = append(args, dbTimePtr(filter.From))
	}
	if filter.To != nil {
		query += " AND p.created_at <= ?"
		args = append(args, dbTimePtr(filter.To))
	}
	query += " ORDER BY p.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("purchase list: %w", err)
	}
	defer rows.Close()

	var purchases []*model.Purchase
	for rows.Next() {
		p := &model.Purchase{}
		if err := rows.Scan(&p.ID, &p.SupplierID, &p.SupplierName, &p.UserID, &p.Reference, &p.Total, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("purchase scan: %w", err)
		}
		purchases = append(purchases, p)
	}
	return purchases, rows.Err()
}
