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

// List returns all customers with their on-the-fly credit balance.
// Optional search filters by name or phone (LIKE).
func (r *customerRepo) List(ctx context.Context, search string) ([]*model.CustomerWithBalance, error) {
	query := `
		SELECT
			c.id, c.name, c.phone, c.address, c.created_at,
			COALESCE(SUM(CASE WHEN s.payment_method = 'credit' AND s.status = 'completed' THEN s.total ELSE 0 END), 0)
			- COALESCE(SUM(DISTINCT COALESCE(cp_sum.amount, 0)), 0) AS balance,
			MIN(CASE WHEN s.payment_method = 'credit' AND s.status = 'completed' THEN s.created_at ELSE NULL END) AS oldest_debt_at
		FROM customers c
		LEFT JOIN sales s ON s.customer_id = c.id
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
			&cwb.ID, &cwb.Name, &cwb.Phone, &cwb.Address, &cwb.CreatedAt,
			&cwb.Balance, &oldestDebt,
		); err != nil {
			return nil, fmt.Errorf("customer scan: %w", err)
		}
		if oldestDebt.Valid && oldestDebt.String != "" {
			t, err := time.Parse("2006-01-02T15:04:05Z", oldestDebt.String)
			if err != nil {
				// try alternate SQLite datetime format
				t, err = time.Parse("2006-01-02 15:04:05", oldestDebt.String)
			}
			if err == nil {
				cwb.OldestDebtAt = &t
			}
		}
		customers = append(customers, &cwb)
	}
	return customers, rows.Err()
}

func (r *customerRepo) GetByID(ctx context.Context, id int64) (*model.Customer, error) {
	c := &model.Customer{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(phone,''), COALESCE(address,''), created_at FROM customers WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Phone, &c.Address, &c.CreatedAt)
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
		`INSERT INTO customers (name, phone, address) VALUES (?, ?, ?)`,
		c.Name, c.Phone, c.Address)
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
		`UPDATE customers SET name = ?, phone = ?, address = ? WHERE id = ?`,
		c.Name, c.Phone, c.Address, c.ID)
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
			COALESCE((SELECT SUM(total) FROM sales WHERE customer_id = ? AND payment_method = 'credit' AND status = 'completed'), 0)
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
			COALESCE((SELECT SUM(total) FROM sales WHERE payment_method = 'credit' AND status = 'completed'), 0)
			- COALESCE((SELECT SUM(amount) FROM customer_payments), 0)
	`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("portfolio balance: %w", err)
	}
	return total, nil
}
