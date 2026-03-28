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
	From          *time.Time
	To            *time.Time
	Limit         int
	IncludeVoided bool // when false (default), only 'completed' sales are returned
}

// SaleRepository defines operations for sales.
type SaleRepository interface {
	Create(ctx context.Context, sale *model.Sale) (*model.Sale, error)
	List(ctx context.Context, filter SaleFilter) ([]*model.Sale, error)
	GetByID(ctx context.Context, id int64) (*model.Sale, error)
	Void(ctx context.Context, saleID int64, userID int64) (*model.Sale, error)
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
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, discount, subtotal, tax_rate)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.SaleID, item.ProductID, item.Quantity, item.UnitPrice, item.Discount, item.Subtotal, item.TaxRate,
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

	// Register movement for all sales; credit sales use a distinct description prefix
	if sale.RegisterID != nil {
		desc := fmt.Sprintf("Venta #%d", saleID)
		if sale.PaymentMethod == "credit" {
			desc = fmt.Sprintf("Crédito #%d", saleID)
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO register_movements (register_id, type, amount, description) VALUES (?, 'sale', ?, ?)`,
			*sale.RegisterID, sale.Total, desc,
		)
		if err != nil {
			return nil, fmt.Errorf("register movement insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("sale commit: %w", err)
	}

	return r.GetByID(ctx, saleID)
}

func (r *saleRepo) List(ctx context.Context, filter SaleFilter) ([]*model.Sale, error) {
	query := `SELECT id, user_id, customer_id, register_id, subtotal, discount, tax, total,
		payment_method, cash_tendered, change_amount, reference, status, voided_by, voided_at, created_at
		FROM sales WHERE 1=1`
	args := []any{}

	if !filter.IncludeVoided {
		query += " AND status = 'completed'"
	}
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
		payment_method, cash_tendered, change_amount, reference, status, voided_by, voided_at, created_at
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
		SELECT id, sale_id, product_id, quantity, unit_price, discount, subtotal, tax_rate
		FROM sale_items WHERE sale_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("sale items list: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.SaleItem
		if err := rows.Scan(&item.ID, &item.SaleID, &item.ProductID,
			&item.Quantity, &item.UnitPrice, &item.Discount, &item.Subtotal, &item.TaxRate); err != nil {
			return nil, fmt.Errorf("sale item scan: %w", err)
		}
		s.Items = append(s.Items, item)
	}
	return s, rows.Err()
}

// Void marks a sale as voided, restores stock for product items, and records
// a negative register movement — all in a single transaction.
func (r *saleRepo) Void(ctx context.Context, saleID int64, userID int64) (*model.Sale, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("void begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Verify sale exists and is not already voided
	var status string
	var total float64
	var registerID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT status, total, register_id FROM sales WHERE id = ?`, saleID).
		Scan(&status, &total, &registerID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("void get sale: %w", err)
	}
	if status == "voided" {
		return nil, fmt.Errorf("la venta ya fue anulada")
	}

	// Restore stock for 'product' type items
	rows, err := tx.QueryContext(ctx, `
		SELECT si.product_id, si.quantity, p.type
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = ?`, saleID)
	if err != nil {
		return nil, fmt.Errorf("void get items: %w", err)
	}
	type itemRow struct {
		productID   int64
		quantity    float64
		productType string
	}
	var items []itemRow
	for rows.Next() {
		var ir itemRow
		if err := rows.Scan(&ir.productID, &ir.quantity, &ir.productType); err != nil {
			rows.Close()
			return nil, fmt.Errorf("void item scan: %w", err)
		}
		items = append(items, ir)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("void items rows: %w", err)
	}

	for _, item := range items {
		if item.productType == "product" {
			if _, err := tx.ExecContext(ctx,
				`UPDATE products SET stock = stock + ? WHERE id = ?`,
				item.quantity, item.productID); err != nil {
				return nil, fmt.Errorf("void restore stock: %w", err)
			}
		}
	}

	// Mark sale as voided
	now := time.Now()
	if _, err := tx.ExecContext(ctx,
		`UPDATE sales SET status = 'voided', voided_by = ?, voided_at = ? WHERE id = ?`,
		userID, now, saleID); err != nil {
		return nil, fmt.Errorf("void update sale: %w", err)
	}

	// Register reversal movement
	if registerID.Valid {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO register_movements (register_id, type, amount, description) VALUES (?, 'withdrawal', ?, ?)`,
			registerID.Int64, total, fmt.Sprintf("Anulación venta #%d", saleID)); err != nil {
			return nil, fmt.Errorf("void register movement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("void commit: %w", err)
	}

	return r.GetByID(ctx, saleID)
}

func scanSale(s scanner) (*model.Sale, error) {
	sale := &model.Sale{}
	return sale, s.Scan(
		&sale.ID, &sale.UserID, &sale.CustomerID, &sale.RegisterID,
		&sale.Subtotal, &sale.Discount, &sale.Tax, &sale.Total,
		&sale.PaymentMethod, &sale.CashTendered, &sale.ChangeAmount,
		&sale.Reference, &sale.Status, &sale.VoidedBy, &sale.VoidedAt, &sale.CreatedAt,
	)
}
