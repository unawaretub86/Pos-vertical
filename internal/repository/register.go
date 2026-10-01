package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/andreyo/pos/internal/model"
)

// RegisterRepository defines operations for cash registers.
type RegisterRepository interface {
	Create(ctx context.Context, r *model.Register, mv *model.RegisterMovement) (*model.Register, error)
	List(ctx context.Context) ([]*model.Register, error)
	GetByID(ctx context.Context, id int64) (*model.Register, error)
	GetOpen(ctx context.Context) (*model.Register, error)
	Close(ctx context.Context, id int64, actualClose, expectedClose, difference float64) (*model.Register, error)
	AddMovement(ctx context.Context, m *model.RegisterMovement) (*model.RegisterMovement, error)
	ListMovements(ctx context.Context, registerID int64) ([]*model.RegisterMovement, error)
	// SumCashSales returns the sum of cash sales for a register.
	SumCashSales(ctx context.Context, registerID int64) (float64, error)
	// SumCashOut returns the sum of cash_out movements for a register.
	SumCashOut(ctx context.Context, registerID int64) (float64, error)
	// SumCashIn returns the sum of cash_in movements for a register.
	SumCashIn(ctx context.Context, registerID int64) (float64, error)
	// SumByPaymentMethod returns completed sales totals for a register, grouped
	// by payment method — used by the register close ticket (US-051), which
	// any authenticated cashier can see even though GET /reports/sales is
	// admin-only.
	SumByPaymentMethod(ctx context.Context, registerID int64) ([]PaymentMethodTotal, error)
	CountOpenOrders(ctx context.Context) (int, error)
}

type registerRepo struct {
	db *sql.DB
}

// NewRegisterRepository returns a SQLite-backed RegisterRepository.
func NewRegisterRepository(db *sql.DB) RegisterRepository {
	return &registerRepo{db: db}
}

func (r *registerRepo) Create(ctx context.Context, reg *model.Register, mv *model.RegisterMovement) (*model.Register, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO registers (user_id, opening_amount, status) VALUES (?, ?, 'open')`,
		reg.UserID, reg.OpeningAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("register create: %w", err)
	}
	id, _ := res.LastInsertId()

	if mv != nil {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO register_movements (register_id, type, amount, description) VALUES (?, ?, ?, ?)`,
			id, mv.Type, mv.Amount, mv.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("register movement insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *registerRepo) List(ctx context.Context) ([]*model.Register, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, opening_amount, expected_close, actual_close, difference, status, opened_at, closed_at
		 FROM registers ORDER BY opened_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("register list: %w", err)
	}
	defer rows.Close()
	var list []*model.Register
	for rows.Next() {
		reg, err := scanRegister(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, reg)
	}
	return list, rows.Err()
}

func (r *registerRepo) GetByID(ctx context.Context, id int64) (*model.Register, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, opening_amount, expected_close, actual_close, difference, status, opened_at, closed_at
		 FROM registers WHERE id = ?`, id)
	reg, err := scanRegister(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return reg, err
}

func (r *registerRepo) GetOpen(ctx context.Context) (*model.Register, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, opening_amount, expected_close, actual_close, difference, status, opened_at, closed_at
		 FROM registers WHERE status = 'open' ORDER BY opened_at DESC LIMIT 1`)
	reg, err := scanRegister(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return reg, err
}

func (r *registerRepo) Close(ctx context.Context, id int64, actualClose, expectedClose, difference float64) (*model.Register, error) {
	_, err := r.db.ExecContext(ctx,
		`UPDATE registers SET status='closed', actual_close=?, expected_close=?, difference=?, closed_at=CURRENT_TIMESTAMP WHERE id=?`,
		actualClose, expectedClose, difference, id,
	)
	if err != nil {
		return nil, fmt.Errorf("register close: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *registerRepo) AddMovement(ctx context.Context, m *model.RegisterMovement) (*model.RegisterMovement, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO register_movements (register_id, type, amount, description) VALUES (?, ?, ?, ?)`,
		m.RegisterID, m.Type, m.Amount, m.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("register movement insert: %w", err)
	}
	id, _ := res.LastInsertId()
	row := r.db.QueryRowContext(ctx,
		`SELECT id, register_id, type, amount, description, created_at FROM register_movements WHERE id = ?`, id)
	mv := &model.RegisterMovement{}
	err = row.Scan(&mv.ID, &mv.RegisterID, &mv.Type, &mv.Amount, &mv.Description, &mv.CreatedAt)
	return mv, err
}

func (r *registerRepo) ListMovements(ctx context.Context, registerID int64) ([]*model.RegisterMovement, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, register_id, type, amount, description, created_at
		 FROM register_movements WHERE register_id = ? ORDER BY created_at ASC`, registerID)
	if err != nil {
		return nil, fmt.Errorf("register movements list: %w", err)
	}
	defer rows.Close()
	var list []*model.RegisterMovement
	for rows.Next() {
		mv := &model.RegisterMovement{}
		if err := rows.Scan(&mv.ID, &mv.RegisterID, &mv.Type, &mv.Amount, &mv.Description, &mv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, mv)
	}
	return list, rows.Err()
}

func (r *registerRepo) SumCashSales(ctx context.Context, registerID int64) (float64, error) {
	var sum float64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(sp.amount), 0)
		 FROM sale_payments sp
		 JOIN sales s ON s.id = sp.sale_id
		 WHERE s.register_id = ? AND sp.method = 'cash' AND s.status = 'completed'`,
		registerID).Scan(&sum)
	return sum, err
}

func (r *registerRepo) SumByPaymentMethod(ctx context.Context, registerID int64) ([]PaymentMethodTotal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sp.method, COALESCE(SUM(sp.amount), 0), COUNT(*)
		FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.register_id = ? AND s.status = 'completed'
		GROUP BY sp.method
		ORDER BY sp.method`, registerID)
	if err != nil {
		return nil, fmt.Errorf("register sum by payment method: %w", err)
	}
	defer rows.Close()

	var totals []PaymentMethodTotal
	for rows.Next() {
		var t PaymentMethodTotal
		if err := rows.Scan(&t.Method, &t.Total, &t.Count); err != nil {
			return nil, fmt.Errorf("register payment method scan: %w", err)
		}
		totals = append(totals, t)
	}
	return totals, rows.Err()
}

func (r *registerRepo) SumCashOut(ctx context.Context, registerID int64) (float64, error) {
	var sum float64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM register_movements WHERE register_id = ? AND type = 'cash_out'`,
		registerID).Scan(&sum)
	return sum, err
}

func (r *registerRepo) CountOpenOrders(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE status = 'open'`).Scan(&count)
	return count, err
}

func (r *registerRepo) SumCashIn(ctx context.Context, registerID int64) (float64, error) {
	var sum float64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM register_movements WHERE register_id = ? AND type = 'cash_in'`,
		registerID).Scan(&sum)
	return sum, err
}

func scanRegister(s scanner) (*model.Register, error) {
	reg := &model.Register{}
	return reg, s.Scan(
		&reg.ID, &reg.UserID, &reg.OpeningAmount,
		&reg.ExpectedClose, &reg.ActualClose, &reg.Difference,
		&reg.Status, &reg.OpenedAt, &reg.ClosedAt,
	)
}
