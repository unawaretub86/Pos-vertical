package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MarginItem is a product ranked by gross margin percentage in a date range.
type MarginItem struct {
	Name      string  `json:"name"`
	MarginPct float64 `json:"margin_pct"`
	UnitsSold float64 `json:"units_sold"`
}

// SellerItem is a product ranked by units sold in a date range (US-061,
// AC-21/D-2). It backs the assistant's TOP_PRODUCTS body — a ranking by
// volume, deliberately separate from ReportRepository.TopProducts, which
// stays ordered by revenue for /reports/top-products.
type SellerItem struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Revenue  float64 `json:"revenue"`
}

// LowStockItem is an active product whose stock is below its configured minimum.
// It includes velocity metrics (AvgDaily and DaysRemaining) if the product
// has been sold in the last 7 days.
type LowStockItem struct {
	ProductID     int64    `json:"product_id"`
	Name          string   `json:"name"`
	Stock         float64  `json:"stock"`
	MinStock      float64  `json:"min_stock"`
	AvgDaily      *float64 `json:"avg_daily"`
	DaysRemaining *float64 `json:"days_remaining"`
}

// Debtor is a customer with an outstanding credit balance.
type Debtor struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone"`
	Balance      float64    `json:"balance"`
	OldestDebtAt *time.Time `json:"oldest_debt_at"`
}

// AgingReceivable is a customer debt ranked by days since the oldest credit sale.
type AgingReceivable struct {
	CustomerID  int64   `json:"customer_id"`
	Name        string  `json:"name"`
	Balance     float64 `json:"balance"`
	DaysOverdue int     `json:"days_overdue"`
}

// MetricsRepository computes the business metrics shown by the dashboard (and
// consumed by the local assistant). Flow methods filter by the half-open range
// [from, to). Lists are never nil.
type MetricsRepository interface {
	Sold(ctx context.Context, from, to time.Time) (total float64, count int, err error)
	Profit(ctx context.Context, from, to time.Time) (gross, expenses, net float64, unconfiguredCost int, err error)
	CashFlow(ctx context.Context, from, to time.Time) (cashSales, cashOut float64, err error)
	CreditFlow(ctx context.Context, from, to time.Time) (newCredit, collected float64, err error)
	TopByMargin(ctx context.Context, from, to time.Time, limit int) ([]MarginItem, error)
	TopSeller(ctx context.Context, from, to time.Time) (name string, qty float64, err error)
	TopSellers(ctx context.Context, from, to time.Time, limit int) ([]SellerItem, error)
	LowStock(ctx context.Context, limit int) ([]LowStockItem, error)
	Debtors(ctx context.Context, limit int) ([]Debtor, error)
	AgingReceivables(ctx context.Context, limit int) ([]AgingReceivable, error)
}

type metricsRepo struct {
	db       *sql.DB
	expenses ExpenseRepository
}

// NewMetricsRepository returns a MetricsRepository backed by db.
func NewMetricsRepository(db *sql.DB) MetricsRepository {
	return &metricsRepo{db: db, expenses: NewExpenseRepository(db)}
}

// Sold returns the total and count of completed sales in [from, to).
func (r *metricsRepo) Sold(ctx context.Context, from, to time.Time) (float64, int, error) {
	var total float64
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total), 0), COUNT(*)
		FROM sales
		WHERE status = 'completed' AND created_at >= ? AND created_at < ?`,
		from, to,
	).Scan(&total, &count)
	if err != nil {
		return 0, 0, fmt.Errorf("metrics sold: %w", err)
	}
	return total, count, nil
}

// Profit returns gross profit (only products with cost > 0), the expenses of the
// range, net = gross − expenses, and how many distinct sold products have cost = 0.
func (r *metricsRepo) Profit(ctx context.Context, from, to time.Time) (float64, float64, float64, int, error) {
	var gross float64
	var unconfigured int
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN p.cost > 0 THEN (si.unit_price - p.cost) * si.quantity ELSE 0 END), 0),
			COUNT(DISTINCT CASE WHEN p.cost = 0 THEN si.product_id END)
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed' AND s.created_at >= ? AND s.created_at < ?`,
		from, to,
	).Scan(&gross, &unconfigured)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("metrics profit: %w", err)
	}

	expenses, err := r.expenses.SumRange(ctx, from, to)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("metrics profit: %w", err)
	}
	return gross, expenses, gross - expenses, unconfigured, nil
}

// CashFlow returns cash payments of completed sales and cash_out movements in
// [from, to). The estimated cash (cashSales − cashOut) is computed by the caller.
func (r *metricsRepo) CashFlow(ctx context.Context, from, to time.Time) (float64, float64, error) {
	var cashSales, cashOut float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(sp.amount), 0)
		FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.status = 'completed' AND sp.method = 'cash' AND s.created_at >= ? AND s.created_at < ?`,
		from, to,
	).Scan(&cashSales)
	if err != nil {
		return 0, 0, fmt.Errorf("metrics cash flow: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM register_movements
		WHERE type = 'cash_out' AND created_at >= ? AND created_at < ?`,
		from, to,
	).Scan(&cashOut)
	if err != nil {
		return 0, 0, fmt.Errorf("metrics cash flow: %w", err)
	}
	return cashSales, cashOut, nil
}

// CreditFlow returns new credit granted (credit payments of completed sales)
// and credit collected (customer payments) in [from, to).
func (r *metricsRepo) CreditFlow(ctx context.Context, from, to time.Time) (float64, float64, error) {
	var newCredit, collected float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(sp.amount), 0)
		FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.status = 'completed' AND sp.method = 'credit' AND s.created_at >= ? AND s.created_at < ?`,
		from, to,
	).Scan(&newCredit)
	if err != nil {
		return 0, 0, fmt.Errorf("metrics credit flow: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM customer_payments
		WHERE created_at >= ? AND created_at < ?`,
		from, to,
	).Scan(&collected)
	if err != nil {
		return 0, 0, fmt.Errorf("metrics credit flow: %w", err)
	}
	return newCredit, collected, nil
}

// TopByMargin returns up to limit products with cost > 0 sold in [from, to),
// ordered by margin percentage descending.
func (r *metricsRepo) TopByMargin(ctx context.Context, from, to time.Time, limit int) ([]MarginItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.name,
			((AVG(si.unit_price) - p.cost) / AVG(si.unit_price)) * 100 AS margin_pct,
			SUM(si.quantity) AS units_sold
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed'
			AND s.created_at >= ? AND s.created_at < ?
			AND p.cost > 0
		GROUP BY p.id, p.name, p.cost
		ORDER BY margin_pct DESC
		LIMIT ?`,
		from, to, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("metrics top by margin: %w", err)
	}
	defer rows.Close()

	items := []MarginItem{}
	for rows.Next() {
		var item MarginItem
		if err := rows.Scan(&item.Name, &item.MarginPct, &item.UnitsSold); err != nil {
			return nil, fmt.Errorf("metrics top by margin: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metrics top by margin: %w", err)
	}
	return items, nil
}

// TopSeller returns the product with the highest sold quantity in [from, to),
// or "" and 0 when there are no sales.
func (r *metricsRepo) TopSeller(ctx context.Context, from, to time.Time) (string, float64, error) {
	var name sql.NullString
	var qty sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT p.name, COALESCE(SUM(si.quantity), 0) AS qty
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed' AND s.created_at >= ? AND s.created_at < ?
		GROUP BY p.id, p.name
		ORDER BY qty DESC
		LIMIT 1`,
		from, to,
	).Scan(&name, &qty)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("metrics top seller: %w", err)
	}
	if !name.Valid {
		return "", 0, nil
	}
	return name.String, qty.Float64, nil
}

// TopSellers returns up to limit products sold in [from, to), ordered by
// quantity descending (US-061 AC-21/D-2) — unlike ReportRepository.TopProducts,
// which orders by revenue for /reports/top-products and is left unchanged.
func (r *metricsRepo) TopSellers(ctx context.Context, from, to time.Time, limit int) ([]SellerItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.name,
			COALESCE(SUM(si.quantity), 0) AS qty,
			COALESCE(SUM(si.subtotal), 0) AS revenue
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed'
			AND s.created_at >= ? AND s.created_at < ?
		GROUP BY p.id, p.name
		ORDER BY qty DESC
		LIMIT ?`,
		from, to, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("metrics top sellers: %w", err)
	}
	defer rows.Close()

	items := []SellerItem{}
	for rows.Next() {
		var item SellerItem
		if err := rows.Scan(&item.Name, &item.Quantity, &item.Revenue); err != nil {
			return nil, fmt.Errorf("metrics top sellers: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metrics top sellers: %w", err)
	}
	return items, nil
}

// LowStock returns up to limit active products below their minimum stock,
// ordered by days_remaining ascending (urgency). Products without recent sales
// (days_remaining = NULL) appear at the end.
func (r *metricsRepo) LowStock(ctx context.Context, limit int) ([]LowStockItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.id,
			p.name,
			p.stock,
			p.min_stock,
			v.avg_daily,
			ROUND(p.stock / v.avg_daily) AS days_remaining
		FROM products p
		LEFT JOIN (
			SELECT
				si.product_id,
				SUM(si.quantity) / 7.0 AS avg_daily
			FROM sale_items si
			JOIN sales s ON s.id = si.sale_id
			WHERE s.status = 'completed'
			  AND julianday('now', 'localtime') - julianday(s.created_at) <= 7
			GROUP BY si.product_id
		) v ON v.product_id = p.id AND v.avg_daily > 0
		WHERE p.active = 1 AND p.min_stock > 0 AND p.stock < p.min_stock
		ORDER BY
			days_remaining IS NULL,
			days_remaining ASC,
			(p.min_stock - p.stock) DESC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("metrics low stock: %w", err)
	}
	defer rows.Close()

	items := []LowStockItem{}
	for rows.Next() {
		var item LowStockItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Stock, &item.MinStock, &item.AvgDaily, &item.DaysRemaining); err != nil {
			return nil, fmt.Errorf("metrics low stock: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metrics low stock: %w", err)
	}
	return items, nil
}

// customerBalanceExpr is the customer balance (spec/domain/fiado.md): credit
// lines of completed sales minus customer payments. It must be used together
// with customerBalanceFrom, which aggregates both sides per customer before
// joining so that no payment is counted more than once. It is the same formula
// as CustomerRepository.List.
const customerBalanceExpr = `COALESCE(sp_sum.amount, 0) - COALESCE(cp_sum.amount, 0)`

// customerBalanceFrom exposes c (customers), sp_sum (credit per customer, with
// oldest_at = MIN(created_at) of its completed credit sales) and cp_sum
// (payments per customer).
const customerBalanceFrom = `
		FROM customers c
		LEFT JOIN (
			SELECT s2.customer_id, SUM(sp.amount) AS amount, MIN(s2.created_at) AS oldest_at
			FROM sale_payments sp
			JOIN sales s2 ON s2.id = sp.sale_id
			WHERE sp.method = 'credit' AND s2.status = 'completed'
			GROUP BY s2.customer_id
		) sp_sum ON sp_sum.customer_id = c.id
		LEFT JOIN (
			SELECT customer_id, SUM(amount) AS amount FROM customer_payments GROUP BY customer_id
		) cp_sum ON cp_sum.customer_id = c.id`

// Debtors returns customers with balance > 0 ordered by balance descending and
// name ascending. The balance uses the same formula as CustomerRepository.List.
// limit <= 0 means no limit.
func (r *metricsRepo) Debtors(ctx context.Context, limit int) ([]Debtor, error) {
	if limit <= 0 {
		limit = -1 // SQLite: negative LIMIT means no limit
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id, c.name, c.phone,
			`+customerBalanceExpr+` AS balance,
			sp_sum.oldest_at AS oldest_debt_at`+
		customerBalanceFrom+`
		WHERE `+customerBalanceExpr+` > 0
		ORDER BY balance DESC, c.name ASC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("metrics debtors: %w", err)
	}
	defer rows.Close()

	debtors := []Debtor{}
	for rows.Next() {
		var d Debtor
		var oldestDebt sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Phone, &d.Balance, &oldestDebt); err != nil {
			return nil, fmt.Errorf("metrics debtors: %w", err)
		}
		d.OldestDebtAt = parseOldestDebt(oldestDebt)
		debtors = append(debtors, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metrics debtors: %w", err)
	}
	return debtors, nil
}

// AgingReceivables returns up to limit customers with balance > 0 ordered by
// days since their oldest completed credit sale, descending. The balance is
// the same one returned by Debtors and CustomerRepository.List. Rows whose
// days_overdue is NULL (unparseable created_at) are excluded explicitly. When
// two or more customers tie on days_overdue, they are broken by the same
// criterion as Debtors: balance descending and, if that also ties, name
// ascending.
func (r *metricsRepo) AgingReceivables(ctx context.Context, limit int) ([]AgingReceivable, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.name,
			`+customerBalanceExpr+` AS balance,
			CAST(julianday('now') - julianday(sp_sum.oldest_at) AS INTEGER) AS days_overdue`+
		customerBalanceFrom+`
		WHERE `+customerBalanceExpr+` > 0
			AND julianday(sp_sum.oldest_at) IS NOT NULL
		ORDER BY days_overdue DESC, balance DESC, c.name ASC
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("metrics aging receivables: %w", err)
	}
	defer rows.Close()

	items := []AgingReceivable{}
	for rows.Next() {
		var item AgingReceivable
		if err := rows.Scan(&item.CustomerID, &item.Name, &item.Balance, &item.DaysOverdue); err != nil {
			return nil, fmt.Errorf("metrics aging receivables: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("metrics aging receivables: %w", err)
	}
	return items, nil
}

// parseOldestDebt parses the MIN(created_at) of credit sales as returned by
// SQLite. Unparseable values yield nil.
func parseOldestDebt(v sql.NullString) *time.Time {
	if !v.Valid || v.String == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", v.String)
	if err != nil {
		// try alternate SQLite datetime format
		t, err = time.Parse("2006-01-02 15:04:05", v.String)
	}
	if err != nil {
		return nil
	}
	return &t
}
