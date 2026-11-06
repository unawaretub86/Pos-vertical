package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andreyo/pos/internal/model"
)

type customerRepo struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepo{db: db}
}

// List returns all active customers with their on-the-fly credit balance.
// Optional search filters by name or phone (LIKE).
func (r *customerRepo) List(ctx context.Context, search string) ([]*model.CustomerWithBalance, error) {
	query := `
		SELECT
			c.id, c.name, c.phone, c.address, COALESCE(c.notes,''), c.credit_limit, c.created_at,
			` + customerBalanceExpr + ` AS balance,
			sp_sum.oldest_at AS oldest_debt_at` + customerBalanceFrom

	query += ` WHERE c.active = 1`
	args := []any{}
	if strings.TrimSpace(search) != "" {
		query += ` AND (c.name LIKE ? OR c.phone LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	query += ` GROUP BY c.id ORDER BY c.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("customer list: %w", err)
	}
	defer rows.Close()

	var customers []*model.CustomerWithBalance
	for rows.Next() {
		var cwb model.CustomerWithBalance
		var oldestDebt sql.NullString
		if err := rows.Scan(
			&cwb.ID, &cwb.Name, &cwb.Phone, &cwb.Address, &cwb.Notes, &cwb.CreditLimit, &cwb.CreatedAt,
			&cwb.Balance, &oldestDebt,
		); err != nil {
			return nil, fmt.Errorf("customer scan: %w", err)
		}
		cwb.OldestDebtAt = parseOldestDebt(oldestDebt)
		customers = append(customers, &cwb)
	}
	return customers, rows.Err()
}

func (r *customerRepo) GetByID(ctx context.Context, id int64) (*model.Customer, error) {
	c := &model.Customer{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(phone,''), COALESCE(address,''), COALESCE(notes,''), credit_limit, created_at FROM customers WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Phone, &c.Address, &c.Notes, &c.CreditLimit, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customer get: %w", err)
	}
	return c, nil
}

func (r *customerRepo) Create(ctx context.Context, c *model.Customer) (*model.Customer, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customers (name, phone, address, notes, credit_limit) VALUES (?, ?, ?, ?, ?)`,
		c.Name, c.Phone, c.Address, c.Notes, c.CreditLimit)
	if err != nil {
		return nil, fmt.Errorf("customer create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("customer last id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *customerRepo) Update(ctx context.Context, c *model.Customer) (*model.Customer, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE customers SET name = ?, phone = ?, address = ?, notes = ?, credit_limit = ? WHERE id = ?`,
		c.Name, c.Phone, c.Address, c.Notes, c.CreditLimit, c.ID)
	if err != nil {
		return nil, fmt.Errorf("customer update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, c.ID)
}

// Delete deactivates an active customer (active = 0). The row is kept because
// sales and payments reference it; an unknown or already inactive id is
// ErrNotFound.
func (r *customerRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE customers SET active = 0 WHERE id = ? AND active = 1`, id)
	if err != nil {
		return fmt.Errorf("customer delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Balance calculates the on-the-fly balance:
// SUM(credit sales total) - SUM(customer_payments amount)
func (r *customerRepo) Balance(ctx context.Context, id int64) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((
				SELECT SUM(sp.amount) FROM sale_payments sp
				JOIN sales s ON s.id = sp.sale_id
				WHERE s.customer_id = ? AND sp.method = 'credit' AND s.status = 'completed'
			), 0)
			- COALESCE((SELECT SUM(amount) FROM customer_payments WHERE customer_id = ?), 0)
	`, id, id).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("customer balance: %w", err)
	}
	return balance, nil
}

// TotalPortfolioBalance returns the sum of the outstanding balances of
// customers whose balance is > 0 (spec/domain/glosario.md: cartera). It
// aggregates the same per-customer formula as List/Balance/Debtors/
// AgingReceivables (customerBalanceExpr/customerBalanceFrom in metrics.go),
// so a customer who overpaid (negative balance) never reduces what the
// others owe.
func (r *customerRepo) TotalPortfolioBalance(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance), 0) FROM (
			SELECT `+customerBalanceExpr+` AS balance`+customerBalanceFrom+`
		) balances
		WHERE balance > 0
	`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("portfolio balance: %w", err)
	}
	return total, nil
}

// ListPayments returns all payments for a given customer, ordered by most recent first.
func (r *customerRepo) ListPayments(ctx context.Context, customerID int64) ([]*model.CustomerPayment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, amount, method, register_id, COALESCE(note,''), created_at
		 FROM customer_payments WHERE customer_id = ? ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.CustomerPayment
	for rows.Next() {
		p := &model.CustomerPayment{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.Amount, &p.Method, &p.RegisterID, &p.Note, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// CreatePayment inserts a new customer payment (abono).
func (r *customerRepo) CreatePayment(ctx context.Context, p *model.CustomerPayment) (*model.CustomerPayment, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customer_payments (customer_id, amount, method, register_id, note) VALUES (?, ?, ?, ?, ?)`,
		p.CustomerID, p.Amount, p.Method, p.RegisterID, p.Note)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("payment last id: %w", err)
	}
	created := &model.CustomerPayment{}
	err = r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, amount, method, register_id, COALESCE(note,''), created_at FROM customer_payments WHERE id = ?`, id).
		Scan(&created.ID, &created.CustomerID, &created.Amount, &created.Method, &created.RegisterID, &created.Note, &created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get created payment: %w", err)
	}
	return created, nil
}

// ListCreditSales returns all credit sales for a customer (completed only), most recent first.
func (r *customerRepo) ListCreditSales(ctx context.Context, customerID int64) ([]*model.Sale, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, customer_id, COALESCE(register_id, 0), subtotal, discount, tax, total,
		        payment_method, status, created_at
		 FROM sales
		 WHERE customer_id = ? AND status = 'completed'
		   AND id IN (SELECT sale_id FROM sale_payments WHERE method = 'credit')
		 ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list credit sales: %w", err)
	}
	defer rows.Close()

	var sales []*model.Sale
	for rows.Next() {
		s := &model.Sale{}
		var regID int64
		if err := rows.Scan(&s.ID, &s.UserID, &s.CustomerID, &regID,
			&s.Subtotal, &s.Discount, &s.Tax, &s.Total,
			&s.PaymentMethod, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan credit sale: %w", err)
		}
		if regID != 0 {
			s.RegisterID = &regID
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// GetPOSSummary returns the customer context for the POS checkout.
func (r *customerRepo) GetPOSSummary(ctx context.Context, id int64) (*model.CustomerPosSummary, error) {
	summary := &model.CustomerPosSummary{}

	c, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	summary.CreditLimit = c.CreditLimit

	var balance float64
	var oldestDebt sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT
			`+customerBalanceExpr+` AS balance,
			sp_sum.oldest_at AS oldest_debt_at
		`+customerBalanceFrom+`
		WHERE c.id = ?`, id).Scan(&balance, &oldestDebt)
	if err != nil {
		return nil, fmt.Errorf("customer summary balance: %w", err)
	}
	summary.Balance = balance
	oldest := parseOldestDebt(oldestDebt)
	if balance > 0 && oldest != nil {
		days := int(time.Since(*oldest).Hours() / 24)
		summary.DaysOverdue = &days
	}

	var lastVisit sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM sales WHERE customer_id = ? AND status = 'completed'`, id).Scan(&lastVisit)
	if err == nil && lastVisit.Valid {
		summary.LastVisitAt = &lastVisit.String
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.id, p.name, COUNT(DISTINCT si.sale_id) AS times_bought
		FROM sale_items si
		JOIN sales s ON s.id = si.sale_id
		JOIN products p ON p.id = si.product_id
		WHERE s.customer_id = ? AND s.status = 'completed'
		GROUP BY p.id, p.name
		ORDER BY times_bought DESC, p.name ASC
		LIMIT 3`, id)
	if err != nil {
		return nil, fmt.Errorf("customer summary top products: %w", err)
	}
	defer rows.Close()

	summary.TopProducts = []model.CustomerTopProduct{}
	for rows.Next() {
		var tp model.CustomerTopProduct
		if err := rows.Scan(&tp.ProductID, &tp.Name, &tp.TimesBought); err != nil {
			return nil, fmt.Errorf("scan top product: %w", err)
		}
		summary.TopProducts = append(summary.TopProducts, tp)
	}

	return summary, nil
}
