package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/andreyo/pos/internal/model"
)

// ErrTableNameTaken is returned when a table name already exists (active or not).
var ErrTableNameTaken = errors.New("table name taken")

// ErrTableHasOpenOrder is returned when deactivating a table with an open order.
var ErrTableHasOpenOrder = errors.New("table has open order")

type DiningTableRepository interface {
	List(ctx context.Context) ([]*model.TableWithState, error)
	GetByID(ctx context.Context, id int64) (*model.DiningTable, error)
	Create(ctx context.Context, t *model.DiningTable) (*model.DiningTable, error)
	Update(ctx context.Context, t *model.DiningTable) (*model.DiningTable, error)
	Deactivate(ctx context.Context, id int64) error
}

type diningTableRepo struct {
	db *sql.DB
}

func NewDiningTableRepository(db *sql.DB) DiningTableRepository {
	return &diningTableRepo{db: db}
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (r *diningTableRepo) List(ctx context.Context) ([]*model.TableWithState, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.zone, t.seats, t.sort_order, t.active,
		       o.id,
		       CAST(COALESCE((julianday('now') - julianday(o.created_at)) * 1440, 0) AS INTEGER),
		       COALESCE((SELECT SUM(quantity * unit_price) FROM order_items
		                 WHERE order_id = o.id AND status != 'void'), 0),
		       (SELECT COUNT(*) FROM order_items
		        WHERE order_id = o.id AND status IN ('pending','preparing')),
		       (SELECT COUNT(*) FROM order_items
		        WHERE order_id = o.id AND status IN ('ready','delivered'))
		FROM dining_tables t
		LEFT JOIN orders o ON o.id = (
			SELECT MAX(id) FROM orders WHERE table_id = t.id AND status = 'open')
		WHERE t.active = 1
		ORDER BY t.sort_order, t.name`)
	if err != nil {
		return nil, fmt.Errorf("dining table list: %w", err)
	}
	defer rows.Close()

	var out []*model.TableWithState
	for rows.Next() {
		t := &model.TableWithState{}
		var orderID sql.NullInt64
		var pending, done int
		if err := rows.Scan(&t.ID, &t.Name, &t.Zone, &t.Seats, &t.SortOrder, &t.Active,
			&orderID, &t.OpenedMinutes, &t.OpenTotal, &pending, &done); err != nil {
			return nil, fmt.Errorf("dining table scan: %w", err)
		}
		switch {
		case !orderID.Valid:
			t.State = "libre"
			t.OpenedMinutes = 0
			t.OpenTotal = 0
		case pending == 0 && done > 0:
			t.State = "por_cobrar"
		default:
			t.State = "ocupada"
		}
		if orderID.Valid {
			id := orderID.Int64
			t.OpenOrderID = &id
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *diningTableRepo) GetByID(ctx context.Context, id int64) (*model.DiningTable, error) {
	t := &model.DiningTable{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, zone, seats, sort_order, active FROM dining_tables WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Zone, &t.Seats, &t.SortOrder, &t.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("dining table get: %w", err)
	}
	return t, nil
}

func (r *diningTableRepo) Create(ctx context.Context, t *model.DiningTable) (*model.DiningTable, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO dining_tables (name, zone, seats, sort_order) VALUES (?, ?, ?, ?)`,
		t.Name, t.Zone, t.Seats, t.SortOrder)
	if isUniqueViolation(err) {
		return nil, ErrTableNameTaken
	}
	if err != nil {
		return nil, fmt.Errorf("dining table create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("dining table create id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *diningTableRepo) Update(ctx context.Context, t *model.DiningTable) (*model.DiningTable, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE dining_tables SET name = ?, zone = ?, seats = ?, sort_order = ? WHERE id = ?`,
		t.Name, t.Zone, t.Seats, t.SortOrder, t.ID)
	if isUniqueViolation(err) {
		return nil, ErrTableNameTaken
	}
	if err != nil {
		return nil, fmt.Errorf("dining table update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("dining table update rows: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, t.ID)
}

func (r *diningTableRepo) Deactivate(ctx context.Context, id int64) error {
	if _, err := r.GetByID(ctx, id); err != nil {
		return err
	}
	var open int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders WHERE table_id = ? AND status = 'open'`, id).Scan(&open); err != nil {
		return fmt.Errorf("dining table open orders: %w", err)
	}
	if open > 0 {
		return ErrTableHasOpenOrder
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE dining_tables SET active = 0 WHERE id = ?`, id); err != nil {
		return fmt.Errorf("dining table deactivate: %w", err)
	}
	return nil
}
