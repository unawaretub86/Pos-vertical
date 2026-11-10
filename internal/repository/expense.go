package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andreyo/pos/internal/model"
)

// ExpenseFilter controls listing behavior for GET /api/expenses.
type ExpenseFilter struct {
	Category string
	From     *time.Time
	To       *time.Time
}

// ExpenseRepository persists business expenses (US-049).
type ExpenseRepository interface {
	Create(ctx context.Context, e *model.Expense) (*model.Expense, error)
	List(ctx context.Context, filter ExpenseFilter) ([]*model.Expense, error)
	// SumRange returns the sum of expenses created in [from, to), used to
	// compute net_profit on the dashboard against the same today-window
	// already computed there (server local time).
	SumRange(ctx context.Context, from, to time.Time) (float64, error)
}

type expenseRepo struct {
	db *sql.DB
}

// NewExpenseRepository returns a SQLite-backed ExpenseRepository.
func NewExpenseRepository(db *sql.DB) ExpenseRepository {
	return &expenseRepo{db: db}
}

func (r *expenseRepo) Create(ctx context.Context, e *model.Expense) (*model.Expense, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO expenses (register_id, user_id, category, amount, note) VALUES (?, ?, ?, ?, ?)`,
		e.RegisterID, e.UserID, e.Category, e.Amount, e.Note,
	)
	if err != nil {
		return nil, fmt.Errorf("expense insert: %w", err)
	}
	id, _ := res.LastInsertId()
	row := r.db.QueryRowContext(ctx,
		`SELECT id, register_id, user_id, category, amount, COALESCE(note, ''), created_at FROM expenses WHERE id = ?`, id)
	ex := &model.Expense{}
	if err := row.Scan(&ex.ID, &ex.RegisterID, &ex.UserID, &ex.Category, &ex.Amount, &ex.Note, &ex.CreatedAt); err != nil {
		return nil, fmt.Errorf("expense scan: %w", err)
	}
	return ex, nil
}

func (r *expenseRepo) List(ctx context.Context, filter ExpenseFilter) ([]*model.Expense, error) {
	query := `
		SELECT e.id, e.register_id, e.user_id, u.username, e.category, e.amount, COALESCE(e.note, ''), e.created_at
		FROM expenses e
		JOIN users u ON u.id = e.user_id
		WHERE 1=1`
	args := []any{}

	if filter.Category != "" {
		query += " AND e.category = ?"
		args = append(args, filter.Category)
	}
	if filter.From != nil {
		query += " AND e.created_at >= ?"
		args = append(args, dbTimePtr(filter.From))
	}
	if filter.To != nil {
		query += " AND e.created_at <= ?"
		args = append(args, dbTimePtr(filter.To))
	}
	query += " ORDER BY e.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("expense list: %w", err)
	}
	defer rows.Close()

	var list []*model.Expense
	for rows.Next() {
		e := &model.Expense{}
		if err := rows.Scan(&e.ID, &e.RegisterID, &e.UserID, &e.Username, &e.Category, &e.Amount, &e.Note, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("expense scan: %w", err)
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *expenseRepo) SumRange(ctx context.Context, from, to time.Time) (float64, error) {
	var sum float64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE created_at >= ? AND created_at < ?`,
		dbTime(from), dbTime(to),
	).Scan(&sum)
	return sum, err
}
