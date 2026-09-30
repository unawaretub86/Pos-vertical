package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/andreyo/pos/internal/model"
	"modernc.org/sqlite"
)

var (
	// ErrOrderNotOpen is returned when changing an order that is not 'open'.
	ErrOrderNotOpen = errors.New("order not open")
	// ErrTableInactive is returned when opening an order on a deactivated table.
	ErrTableInactive = errors.New("table inactive")
	// ErrItemVoid is returned when changing or voiding an already void item.
	ErrItemVoid = errors.New("item void")
	// ErrProductInactive is returned when adding an inactive product.
	ErrProductInactive = errors.New("product inactive")
	// ErrBusy wraps SQLITE_BUSY: the caller should retry.
	ErrBusy = errors.New("database busy")
)

// ItemPatch holds the optional fields of an item update; nil means "unchanged".
type ItemPatch struct {
	Quantity *int
	Note     *string
	Status   *string
}

type OrderRepository interface {
	Open(ctx context.Context, tableID, userID int64) (*model.Order, error)
	AddItem(ctx context.Context, orderID, productID int64, quantity int, note string) (*model.OrderItem, error)
	UpdateItem(ctx context.Context, orderID, itemID int64, p ItemPatch) (*model.OrderItem, error)
	VoidItem(ctx context.Context, orderID, itemID, userID int64) (*model.OrderItem, error)
	Cancel(ctx context.Context, orderID, userID int64, reason string) error
	Get(ctx context.Context, id int64) (*model.Order, error)
	ListOpen(ctx context.Context, status string) ([]*model.OrderSummary, error)
}

type orderRepo struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepo{db: db}
}

// mapBusy converts a SQLITE_BUSY driver error into ErrBusy.
func mapBusy(err error) error {
	var se *sqlite.Error
	if errors.As(err, &se) && se.Code()&0xff == 5 {
		return fmt.Errorf("%w: %v", ErrBusy, err)
	}
	return err
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

const itemSelect = `
	SELECT i.id, i.order_id, i.product_id, p.name, i.quantity, i.unit_price,
	       i.quantity * i.unit_price, COALESCE(i.note, ''), i.status
	FROM order_items i JOIN products p ON p.id = i.product_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanItem(s rowScanner) (*model.OrderItem, error) {
	it := &model.OrderItem{}
	if err := s.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Quantity,
		&it.UnitPrice, &it.Subtotal, &it.Note, &it.Status); err != nil {
		return nil, err
	}
	return it, nil
}

func getItem(ctx context.Context, q queryer, orderID, itemID int64) (*model.OrderItem, error) {
	it, err := scanItem(q.QueryRowContext(ctx, itemSelect+` WHERE i.id = ? AND i.order_id = ?`, itemID, orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order item get: %w", err)
	}
	return it, nil
}

// requireOpen checks the order exists and is open, and returns its table id.
func requireOpen(ctx context.Context, q queryer, orderID int64) (sql.NullInt64, error) {
	var status string
	var tableID sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT status, table_id FROM orders WHERE id = ?`, orderID).Scan(&status, &tableID)
	if errors.Is(err, sql.ErrNoRows) {
		return tableID, ErrNotFound
	}
	if err != nil {
		return tableID, fmt.Errorf("order get status: %w", err)
	}
	if status != "open" {
		return tableID, ErrOrderNotOpen
	}
	return tableID, nil
}

func touchOrder(ctx context.Context, ex execer, orderID int64) error {
	if _, err := ex.ExecContext(ctx, `UPDATE orders SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, orderID); err != nil {
		return fmt.Errorf("order touch: %w", err)
	}
	return nil
}

func (r *orderRepo) Open(ctx context.Context, tableID, userID int64) (*model.Order, error) {
	var active bool
	err := r.db.QueryRowContext(ctx, `SELECT active FROM dining_tables WHERE id = ?`, tableID).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order open table: %w", mapBusy(err))
	}
	if !active {
		return nil, ErrTableInactive
	}

	// Single atomic statement: active table AND no open order.
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO orders (type, table_id, status, user_id)
		SELECT 'table', ?, 'open', ?
		WHERE EXISTS (SELECT 1 FROM dining_tables WHERE id = ? AND active = 1)
		  AND NOT EXISTS (SELECT 1 FROM orders WHERE table_id = ? AND status = 'open')`,
		tableID, userID, tableID, tableID)
	if err != nil {
		return nil, fmt.Errorf("order open: %w", mapBusy(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("order open rows: %w", err)
	}
	if n == 0 {
		if err := r.db.QueryRowContext(ctx, `SELECT active FROM dining_tables WHERE id = ?`, tableID).Scan(&active); err == nil && !active {
			return nil, ErrTableInactive
		}
		return nil, ErrTableHasOpenOrder
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("order open id: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *orderRepo) AddItem(ctx context.Context, orderID, productID int64, quantity int, note string) (*model.OrderItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("order add item begin: %w", mapBusy(err))
	}
	defer tx.Rollback()

	if _, err := requireOpen(ctx, tx, orderID); err != nil {
		return nil, mapBusy(err)
	}
	var price float64
	var active bool
	err = tx.QueryRowContext(ctx, `SELECT sale_price, active FROM products WHERE id = ?`, productID).Scan(&price, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order add item product: %w", mapBusy(err))
	}
	if !active {
		return nil, ErrProductInactive
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO order_items (order_id, product_id, quantity, unit_price, note, status)
		 VALUES (?, ?, ?, ?, ?, 'pending')`,
		orderID, productID, quantity, price, note)
	if err != nil {
		return nil, fmt.Errorf("order add item insert: %w", mapBusy(err))
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("order add item id: %w", err)
	}
	if err := touchOrder(ctx, tx, orderID); err != nil {
		return nil, mapBusy(err)
	}
	it, err := getItem(ctx, tx, orderID, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("order add item commit: %w", mapBusy(err))
	}
	return it, nil
}

func (r *orderRepo) UpdateItem(ctx context.Context, orderID, itemID int64, p ItemPatch) (*model.OrderItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("order update item begin: %w", mapBusy(err))
	}
	defer tx.Rollback()

	if _, err := requireOpen(ctx, tx, orderID); err != nil {
		return nil, mapBusy(err)
	}
	cur, err := getItem(ctx, tx, orderID, itemID)
	if err != nil {
		return nil, mapBusy(err)
	}
	if cur.Status == "void" {
		return nil, ErrItemVoid
	}
	if p.Quantity != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE order_items SET quantity = ? WHERE id = ?`, *p.Quantity, itemID); err != nil {
			return nil, fmt.Errorf("order update item quantity: %w", mapBusy(err))
		}
	}
	if p.Note != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE order_items SET note = ? WHERE id = ?`, *p.Note, itemID); err != nil {
			return nil, fmt.Errorf("order update item note: %w", mapBusy(err))
		}
	}
	if p.Status != nil {
		var q string
		switch {
		case *p.Status == "ready" && cur.Status != "ready":
			q = `UPDATE order_items SET status = ?, ready_at = CURRENT_TIMESTAMP WHERE id = ?`
		case *p.Status != "ready" && cur.Status == "ready":
			q = `UPDATE order_items SET status = ?, ready_at = NULL WHERE id = ?`
		default:
			q = `UPDATE order_items SET status = ? WHERE id = ?`
		}
		if _, err := tx.ExecContext(ctx, q, *p.Status, itemID); err != nil {
			return nil, fmt.Errorf("order update item status: %w", mapBusy(err))
		}
	}
	if err := touchOrder(ctx, tx, orderID); err != nil {
		return nil, mapBusy(err)
	}
	it, err := getItem(ctx, tx, orderID, itemID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("order update item commit: %w", mapBusy(err))
	}
	return it, nil
}

func (r *orderRepo) VoidItem(ctx context.Context, orderID, itemID, userID int64) (*model.OrderItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("order void item begin: %w", mapBusy(err))
	}
	defer tx.Rollback()

	tableID, err := requireOpen(ctx, tx, orderID)
	if err != nil {
		return nil, mapBusy(err)
	}
	cur, err := getItem(ctx, tx, orderID, itemID)
	if err != nil {
		return nil, mapBusy(err)
	}
	if cur.Status == "void" {
		return nil, ErrItemVoid
	}
	details := map[string]any{
		"order_id": orderID, "product_id": cur.ProductID, "product_name": cur.ProductName,
		"quantity": cur.Quantity, "unit_price": cur.UnitPrice, "note": cur.Note,
		"status": cur.Status, "table_id": nil, "user_id": userID,
	}
	if tableID.Valid {
		details["table_id"] = tableID.Int64
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("order void item details: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE order_items SET status = 'void' WHERE id = ?`, itemID); err != nil {
		return nil, fmt.Errorf("order void item: %w", mapBusy(err))
	}
	if err := InsertAuditLog(ctx, tx, userID, "order_item_void", "order_item", itemID, string(raw)); err != nil {
		return nil, mapBusy(err)
	}
	if err := touchOrder(ctx, tx, orderID); err != nil {
		return nil, mapBusy(err)
	}
	it, err := getItem(ctx, tx, orderID, itemID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("order void item commit: %w", mapBusy(err))
	}
	return it, nil
}

func (r *orderRepo) Cancel(ctx context.Context, orderID, userID int64, reason string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("order cancel begin: %w", mapBusy(err))
	}
	defer tx.Rollback()

	if _, err := requireOpen(ctx, tx, orderID); err != nil {
		return mapBusy(err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE orders SET status = 'cancelled', closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		orderID); err != nil {
		return fmt.Errorf("order cancel: %w", mapBusy(err))
	}
	if err := InsertAuditLog(ctx, tx, userID, "order_cancel", "order", orderID, reason); err != nil {
		return mapBusy(err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("order cancel commit: %w", mapBusy(err))
	}
	return nil
}

func (r *orderRepo) Get(ctx context.Context, id int64) (*model.Order, error) {
	o := &model.Order{Items: []model.OrderItem{}}
	var tableID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, type, table_id, status, user_id, created_at FROM orders WHERE id = ?`, id).
		Scan(&o.ID, &o.Type, &tableID, &o.Status, &o.UserID, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order get: %w", err)
	}
	if tableID.Valid {
		v := tableID.Int64
		o.TableID = &v
	}
	rows, err := r.db.QueryContext(ctx, itemSelect+` WHERE i.order_id = ? ORDER BY i.id`, id)
	if err != nil {
		return nil, fmt.Errorf("order items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("order item scan: %w", err)
		}
		if it.Status != "void" {
			o.Total += it.Subtotal
		}
		o.Items = append(o.Items, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order items rows: %w", err)
	}
	return o, nil
}

func (r *orderRepo) ListOpen(ctx context.Context, status string) ([]*model.OrderSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.table_id, COALESCE(t.name, ''),
		       COALESCE((SELECT SUM(quantity * unit_price) FROM order_items
		                 WHERE order_id = o.id AND status != 'void'), 0),
		       (SELECT COUNT(*) FROM order_items
		        WHERE order_id = o.id AND status IN ('pending','preparing')),
		       CAST((julianday('now') - julianday(o.created_at)) * 1440 AS INTEGER)
		FROM orders o LEFT JOIN dining_tables t ON t.id = o.table_id
		WHERE o.status = ?
		ORDER BY o.id`, status)
	if err != nil {
		return nil, fmt.Errorf("order list: %w", err)
	}
	defer rows.Close()
	out := []*model.OrderSummary{}
	for rows.Next() {
		s := &model.OrderSummary{}
		var tableID sql.NullInt64
		if err := rows.Scan(&s.ID, &tableID, &s.TableName, &s.Total, &s.ItemsPending, &s.OpenedMinutes); err != nil {
			return nil, fmt.Errorf("order list scan: %w", err)
		}
		if tableID.Valid {
			v := tableID.Int64
			s.TableID = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
