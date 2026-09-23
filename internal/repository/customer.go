package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/andreyo/pos/internal/model"
)

type customerRepo struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) CustomerRepository {
	return &customerRepo{db: db}
}

// List returns all customers with their on-the-fly credit balance.
// Optional search filters by name or phone (LIKE).
func (r *customerRepo) List(ctx context.Context, search string) ([]*model.CustomerWithBalance, error) {
	query := `
		SELECT
			c.id, c.name, c.phone, c.address, COALESCE(c.notes,''), c.credit_limit, c.created_at,
			COALESCE(sp_sum.amount, 0) - COALESCE(cp_sum.amount, 0) AS balance,
			sp_sum.oldest_at AS oldest_debt_at
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

	args := []any{}
	if strings.TrimSpace(search) != "" {
		query += ` WHERE (c.name LIKE ? OR c.phone LIKE ?)`
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

func (r *customerRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM customers WHERE id = ?`, id)
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

// TotalPortfolioBalance returns the sum of all outstanding customer balances.
func (r *customerRepo) TotalPortfolioBalance(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((
				SELECT SUM(sp.amount) FROM sale_payments sp
				JOIN sales s ON s.id = sp.sale_id
				WHERE sp.method = 'credit' AND s.status = 'completed'
			), 0)
			- COALESCE((SELECT SUM(amount) FROM customer_payments), 0)
	`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("portfolio balance: %w", err)
	}
	return total, nil
}

// ListPayments returns all payments for a given customer, ordered by most recent first.
func (r *customerRepo) ListPayments(ctx context.Context, customerID int64) ([]*model.CustomerPayment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, amount, COALESCE(note,''), created_at
		 FROM customer_payments WHERE customer_id = ? ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*model.CustomerPayment
	for rows.Next() {
		p := &model.CustomerPayment{}
		if err := rows.Scan(&p.ID, &p.CustomerID, &p.Amount, &p.Note, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// CreatePayment inserts a new customer payment (abono).
func (r *customerRepo) CreatePayment(ctx context.Context, p *model.CustomerPayment) (*model.CustomerPayment, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO customer_payments (customer_id, amount, note) VALUES (?, ?, ?)`,
		p.CustomerID, p.Amount, p.Note)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("payment last id: %w", err)
	}
	created := &model.CustomerPayment{}
	err = r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, amount, COALESCE(note,''), created_at FROM customer_payments WHERE id = ?`, id).
		Scan(&created.ID, &created.CustomerID, &created.Amount, &created.Note, &created.CreatedAt)
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
