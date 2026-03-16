package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andreyo/pos/internal/model"
)

// SaleFilter controls listing behavior.
type SaleFilter struct {
	From  *time.Time
	To    *time.Time
	Limit int
}

// SaleRepository defines operations for sales.
type SaleRepository interface {
	Create(ctx context.Context, sale *model.Sale) (*model.Sale, error)
	List(ctx context.Context, filter SaleFilter) ([]*model.Sale, error)
	GetByID(ctx context.Context, id int64) (*model.Sale, error)
}

type saleRepo struct {
	db *sql.DB
}

// NewSaleRepository returns a SQLite-backed SaleRepository.
func NewSaleRepository(db *sql.DB) SaleRepository {
	return &saleRepo{db: db}
}

func (r *saleRepo) Create(ctx context.Context, sale *model.Sale) (*model.Sale, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sale create begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Insert sale
	res, err := tx.ExecContext(ctx, `
		INSERT INTO sales (user_id, customer_id, register_id, subtotal, discount, tax, total, payment_method, cash_tendered, change_amount, reference, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sale.UserID, sale.CustomerID, sale.RegisterID,
		sale.Subtotal, sale.Discount, sale.Tax, sale.Total,
		sale.PaymentMethod, sale.CashTendered, sale.ChangeAmount,
		sale.Reference, sale.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("sale insert: %w", err)
	}
	saleID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("sale last id: %w", err)
	}

	// Insert items and update stock
	for i := range sale.Items {
		item := &sale.Items[i]
		item.SaleID = saleID

		_, err := tx.ExecContext(ctx, `
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, discount, subtotal)
			VALUES (?, ?, ?, ?, ?, ?)`,
			item.SaleID, item.ProductID, item.Quantity, item.UnitPrice, item.Discount, item.Subtotal,
		)
		if err != nil {
			return nil, fmt.Errorf("sale_item insert: %w", err)
		}

		// Only decrement stock for 'product' type items
		var productType, productName string
		var productStock float64
		err = tx.QueryRowContext(ctx, `SELECT type, name, stock FROM products WHERE id = ?`, item.ProductID).Scan(&productType, &productName, &productStock)
		if err != nil {
			return nil, fmt.Errorf("sale_item product type: %w", err)
		}
		if productType == "product" {
			res, err := tx.ExecContext(ctx,
				`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`,
				item.Quantity, item.ProductID, item.Quantity,
			)
			if err != nil {
				return nil, fmt.Errorf("sale stock update: %w", err)
			}
			affected, err := res.RowsAffected()
			if err != nil {
				return nil, fmt.Errorf("sale stock rows affected: %w", err)
			}
			if affected == 0 {
				return nil, fmt.Errorf("stock insuficiente: hay %.0f unidades de %s, intentas vender %.0f", productStock, productName, item.Quantity)
			}
		}
	}

	// Register movement for this sale (same transaction)
	if sale.RegisterID != nil {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO register_movements (register_id, type, amount, description) VALUES (?, 'sale', ?, ?)`,
			*sale.RegisterID, sale.Total, fmt.Sprintf("Venta #%d", saleID),
		)
		if err != nil {
			return nil, fmt.Errorf("register movement insert: %w", err)
		}
	}

	// For credit sales, record the debt in customer_payments (same transaction)
	if sale.PaymentMethod == "credit" && sale.CustomerID != nil {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO customer_payments (customer_id, amount, note) VALUES (?, ?, ?)`,
			*sale.CustomerID, sale.Total, fmt.Sprintf("Deuda venta #%d", saleID),
		)
		if err != nil {
			return nil, fmt.Errorf("customer payment insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("sale commit: %w", err)
	}

	return r.GetByID(ctx, saleID)
}

func (r *saleRepo) List(ctx context.Context, filter SaleFilter) ([]*model.Sale, error) {
	query := `SELECT id, user_id, customer_id, register_id, subtotal, discount, tax, total,
		payment_method, cash_tendered, change_amount, reference, status, created_at
		FROM sales WHERE 1=1`
	args := []any{}

	if filter.From != nil {
		query += " AND created_at >= ?"
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		query += " AND created_at <= ?"
		args = append(args, *filter.To)
	}
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sale list: %w", err)
	}
	defer rows.Close()

	var sales []*model.Sale
	for rows.Next() {
		s, err := scanSale(rows)
		if err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

func (r *saleRepo) GetByID(ctx context.Context, id int64) (*model.Sale, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, customer_id, register_id, subtotal, discount, tax, total,
		payment_method, cash_tendered, change_amount, reference, status, created_at
		FROM sales WHERE id = ?`, id)
	s, err := scanSale(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sale get: %w", err)
	}

	// Load items
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, sale_id, product_id, quantity, unit_price, discount, subtotal
		FROM sale_items WHERE sale_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("sale items list: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.SaleItem
		if err := rows.Scan(&item.ID, &item.SaleID, &item.ProductID,
			&item.Quantity, &item.UnitPrice, &item.Discount, &item.Subtotal); err != nil {
			return nil, fmt.Errorf("sale item scan: %w", err)
		}
		s.Items = append(s.Items, item)
	}
	return s, rows.Err()
}

func scanSale(s scanner) (*model.Sale, error) {
	sale := &model.Sale{}
	return sale, s.Scan(
		&sale.ID, &sale.UserID, &sale.CustomerID, &sale.RegisterID,
		&sale.Subtotal, &sale.Discount, &sale.Tax, &sale.Total,
		&sale.PaymentMethod, &sale.CashTendered, &sale.ChangeAmount,
		&sale.Reference, &sale.Status, &sale.CreatedAt,
	)
}
