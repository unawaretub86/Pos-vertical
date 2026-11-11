package resetdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
)

// clearStep is one physical DELETE. The statements are constants (one per
// table), listed in dependency order (children before parents, contract V-6).
type clearStep struct {
	table    string
	count    string // SELECT COUNT(*) for the plan and the audit details
	del      string
	sequence bool // delete the sqlite_sequence row so ids restart at 1
}

var clearSteps = []clearStep{
	{"electronic_documents", `SELECT COUNT(*) FROM electronic_documents`, `DELETE FROM electronic_documents`, false},
	{"order_items", `SELECT COUNT(*) FROM order_items`, `DELETE FROM order_items`, true},
	{"orders", `SELECT COUNT(*) FROM orders`, `DELETE FROM orders`, true},
	{"sale_items", `SELECT COUNT(*) FROM sale_items`, `DELETE FROM sale_items`, true},
	{"sale_payments", `SELECT COUNT(*) FROM sale_payments`, `DELETE FROM sale_payments`, true},
	{"sales", `SELECT COUNT(*) FROM sales`, `DELETE FROM sales`, true},
	{"customer_payments", `SELECT COUNT(*) FROM customer_payments`, `DELETE FROM customer_payments`, true},
	{"expenses", `SELECT COUNT(*) FROM expenses`, `DELETE FROM expenses`, true},
	{"register_movements", `SELECT COUNT(*) FROM register_movements`, `DELETE FROM register_movements`, true},
	{"registers", `SELECT COUNT(*) FROM registers`, `DELETE FROM registers`, true},
	{"assistant_queries", `SELECT COUNT(*) FROM assistant_queries`, `DELETE FROM assistant_queries`, true},
}

// Audit rows that point to ids of deleted records. Kept: product_import,
// product_price_change, opening_balance_create/update, earlier data_reset.
const (
	auditPointerWhere = `action IN ('void', 'discount', 'price_override', 'cash_out', 'order_item_void', 'order_cancel')`
	auditCountSQL     = `SELECT COUNT(*) FROM audit_log WHERE ` + auditPointerWhere
	auditDeleteSQL    = `DELETE FROM audit_log WHERE ` + auditPointerWhere
)

const (
	deleteSequenceSQL = `DELETE FROM sqlite_sequence WHERE name = ?`
	// Stock rule B: stock = sum of stock_entries, product-type products only
	// (services untouched), inactive products included.
	stockRebuildSQL = `UPDATE products SET stock = COALESCE((SELECT SUM(quantity) FROM stock_entries WHERE product_id = products.id), 0) WHERE type = 'product'`
	stockReportSQL  = `
		SELECT p.id, p.name, p.active, p.stock,
		       COALESCE((SELECT SUM(e.quantity) FROM stock_entries e WHERE e.product_id = p.id), 0),
		       COALESCE((SELECT SUM(si.quantity) FROM sale_items si JOIN sales s ON s.id = si.sale_id
		                 WHERE si.product_id = p.id AND s.status = 'completed'), 0)
		FROM products p WHERE p.type = 'product' ORDER BY p.id`
	insertAuditSQL = `INSERT INTO audit_log (user_id, action, entity, entity_id, details) VALUES (?, 'data_reset', 'system', 0, ?)`
)

// StockChange is one product-type product whose stock changes under rule B.
// Untraced = Now + Sold - Entries: stock that no entry explains.
type StockChange struct {
	ProductID int64
	Name      string
	Active    bool
	Now       float64
	After     float64
	Entries   float64
	Sold      float64
	Untraced  float64
}

// PlanReport is what a reset would delete and change. Counts holds the rows to
// delete per table (audit_log counts only the pointer rows).
type PlanReport struct {
	Counts map[string]int
	Stock  []StockChange
}

// Options tunes Apply.
type Options struct {
	Force         bool
	DiscardFiscal bool
	BackupName    string         // recorded in the data_reset details
	Fingerprint   map[string]int // table counts of the backup; nil skips the recheck
	BeforeLock    func()         // test hook: runs after the backup, right before the lock
}

// Result describes a completed reset.
type Result struct {
	Cleared         map[string]int
	Stock           []StockChange
	DiscardedFiscal map[string]int
	AdminID         int64
	Warnings        []string // post-commit steps that failed (the reset itself is committed)
}

// Plan reads what Apply would do. It writes nothing.
func Plan(ctx context.Context, db queryer) (*PlanReport, error) {
	p := &PlanReport{Counts: map[string]int{}}
	for _, s := range clearSteps {
		var n int
		if err := db.QueryRowContext(ctx, s.count).Scan(&n); err != nil {
			return nil, fmt.Errorf("%w: contar %s: %w", ErrDatabase, s.table, err)
		}
		p.Counts[s.table] = n
	}
	var n int
	if err := db.QueryRowContext(ctx, auditCountSQL).Scan(&n); err != nil {
		return nil, fmt.Errorf("%w: contar audit_log: %w", ErrDatabase, err)
	}
	p.Counts["audit_log"] = n

	rows, err := db.QueryContext(ctx, stockReportSQL)
	if err != nil {
		return nil, fmt.Errorf("%w: leer productos: %w", ErrDatabase, err)
	}
	defer rows.Close()
	for rows.Next() {
		var sc StockChange
		var active int
		if err := rows.Scan(&sc.ProductID, &sc.Name, &active, &sc.Now, &sc.Entries, &sc.Sold); err != nil {
			return nil, fmt.Errorf("%w: leer productos: %w", ErrDatabase, err)
		}
		sc.Active = active != 0
		sc.After = sc.Entries
		sc.Untraced = sc.Now + sc.Sold - sc.Entries
		if math.Abs(sc.Now-sc.After) > 1e-9 {
			p.Stock = append(p.Stock, sc)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: leer productos: %w", ErrDatabase, err)
	}
	return p, nil
}

func isBusy(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "sqlite_busy") || strings.Contains(s, "locked") || strings.Contains(s, "busy")
}

// Apply runs the whole reset in one BEGIN EXCLUSIVE transaction: guards,
// fingerprint recheck, deletes, stock rebuild, sequence deletes, data_reset
// audit row and foreign_key_check. Any failure rolls everything back.
func Apply(ctx context.Context, db *sql.DB, opts Options) (res *Result, err error) {
	if opts.BeforeLock != nil {
		opts.BeforeLock()
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: abrir conexión: %w", ErrDatabase, err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `BEGIN EXCLUSIVE`); err != nil {
		if isBusy(err) {
			return nil, fmt.Errorf("%w: %w", ErrLocked, err)
		}
		return nil, fmt.Errorf("%w: BEGIN EXCLUSIVE: %w", ErrDatabase, err)
	}
	committed := false
	defer func() {
		if !committed {
			// A fresh context: ctx may be the reason we are here.
			conn.ExecContext(context.Background(), `ROLLBACK`) //nolint:errcheck
		}
	}()

	if err := checkGuards(ctx, conn, opts); err != nil {
		return nil, err
	}
	if opts.Fingerprint != nil {
		now, err := TableCounts(ctx, conn)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(now, opts.Fingerprint) {
			return nil, fmt.Errorf("%w: la base cambió después del respaldo; no se borró nada", ErrDatabase)
		}
	}
	adminID, err := firstActiveAdmin(ctx, conn)
	if err != nil {
		return nil, err
	}
	plan, err := Plan(ctx, conn)
	if err != nil {
		return nil, err
	}
	discarded, err := fiscalCounts(ctx, conn)
	if err != nil {
		return nil, err
	}

	for _, s := range clearSteps {
		if _, err := conn.ExecContext(ctx, s.del); err != nil {
			return nil, fmt.Errorf("%w: borrar %s: %w", ErrDatabase, s.table, err)
		}
	}
	if _, err := conn.ExecContext(ctx, auditDeleteSQL); err != nil {
		return nil, fmt.Errorf("%w: borrar audit_log: %w", ErrDatabase, err)
	}
	if _, err := conn.ExecContext(ctx, stockRebuildSQL); err != nil {
		return nil, fmt.Errorf("%w: recalcular stock: %w", ErrDatabase, err)
	}
	for _, s := range clearSteps {
		if !s.sequence {
			continue
		}
		if _, err := conn.ExecContext(ctx, deleteSequenceSQL, s.table); err != nil {
			return nil, fmt.Errorf("%w: reiniciar numeración de %s: %w", ErrDatabase, s.table, err)
		}
	}

	details := map[string]any{
		"backup":     opts.BackupName,
		"tool":       "reset-data",
		"cleared":    plan.Counts,
		"stock_rule": "entries",
	}
	if opts.DiscardFiscal && len(discarded) > 0 {
		details["discarded_fiscal"] = discarded
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("%w: detalles de auditoría: %w", ErrDatabase, err)
	}
	if _, err := conn.ExecContext(ctx, insertAuditSQL, adminID, string(raw)); err != nil {
		return nil, fmt.Errorf("%w: registrar auditoría: %w", ErrDatabase, err)
	}

	fk, err := conn.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return nil, fmt.Errorf("%w: foreign_key_check: %w", ErrDatabase, err)
	}
	violations := 0
	for fk.Next() {
		violations++
	}
	ferr := fk.Err()
	fk.Close()
	if ferr != nil {
		return nil, fmt.Errorf("%w: foreign_key_check: %w", ErrDatabase, ferr)
	}
	if violations > 0 {
		return nil, fmt.Errorf("%w: foreign_key_check encontró %d violación(es); no se borró nada", ErrDatabase, violations)
	}

	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return nil, fmt.Errorf("%w: commit: %w", ErrDatabase, err)
	}
	committed = true

	res = &Result{Cleared: plan.Counts, Stock: plan.Stock, DiscardedFiscal: discarded, AdminID: adminID}
	if _, err := conn.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("checkpoint: %v", err))
	}
	if _, err := conn.ExecContext(ctx, `VACUUM`); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("VACUUM: %v", err))
	}
	return res, nil
}
