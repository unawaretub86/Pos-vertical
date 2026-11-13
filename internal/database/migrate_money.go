package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
)

// migrationBalanceSQL is a customer's outstanding balance: opening balance +
// credit lines of completed sales - abonos. Arguments: the customer id twice
// (?1 and ?2 are the same id).
const migrationBalanceSQL = `
	SELECT
		COALESCE((SELECT amount FROM customer_opening_balances WHERE customer_id = ?1), 0)
		+ COALESCE((
			SELECT SUM(sp.amount) FROM sale_payments sp
			JOIN sales s ON s.id = sp.sale_id
			WHERE s.customer_id = ?2 AND sp.method = 'credit' AND s.status = 'completed'
		), 0)
		- COALESCE((SELECT SUM(amount) FROM customer_payments WHERE customer_id = ?1), 0)`

// migration24 (US-102, R-133) converts every stored peso amount to a whole
// peso. Rounding is half away from zero (SQLite ROUND); sale lines and sales
// subtotal/tax/total are recomputed with the per-line rule using integer
// arithmetic. The rule is copied here on purpose: a migration is frozen and
// must not follow later changes of the service code. It never fails because
// of a data inconsistency.
func migration24(ctx context.Context, tx *sql.Tx) error {
	// Balances before, for every customer (inactive ones included: their
	// balances stay in reports and history).
	before, err := customerBalances(ctx, tx)
	if err != nil {
		return fmt.Errorf("balances before: %w", err)
	}

	// Tables whose CHECK demands amount > 0: a value under 0.5 would round to
	// 0 and abort the startup, so it is clamped to 1 peso.
	for _, t := range []string{"sale_payments", "expenses", "customer_opening_balances"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET amount = MAX(1, ROUND(amount)) WHERE amount IS NOT NULL`, t)); err != nil {
			return fmt.Errorf("round %s.amount: %w", t, err)
		}
	}
	for _, c := range []struct{ table, col string }{
		{"products", "sale_price"}, {"products", "wholesale_price"}, {"products", "cost"},
		{"registers", "opening_amount"}, {"registers", "expected_close"},
		{"registers", "actual_close"}, {"registers", "difference"},
		{"sales", "discount"}, {"sales", "cash_tendered"}, {"sales", "change_amount"},
		{"sale_items", "unit_price"}, {"sale_items", "original_price"}, {"sale_items", "discount"},
		{"customer_payments", "amount"}, {"register_movements", "amount"},
		{"purchases", "total"}, {"purchase_items", "unit_cost"}, {"purchase_items", "subtotal"},
		{"order_items", "unit_price"}, {"business_config", "default_opening_amount"},
	} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET %s = ROUND(%s) WHERE %s IS NOT NULL`, c.table, c.col, c.col, c.col)); err != nil {
			return fmt.Errorf("round %s.%s: %w", c.table, c.col, err)
		}
	}

	sales, err := recomputeSales(ctx, tx)
	if err != nil {
		return err
	}

	after, err := customerBalances(ctx, tx)
	if err != nil {
		return fmt.Errorf("balances after: %w", err)
	}

	// Drift: a balance that moved by a whole peso or more beyond the rounding
	// of the old balance. A sub-peso residue is not drift.
	var drifted []int64
	for id, b := range before {
		if math.Abs(after[id]-math.Round(b)) >= 1 {
			drifted = append(drifted, id)
		}
	}
	if len(drifted) == 0 {
		return nil
	}
	sort.Slice(drifted, func(i, j int) bool { return drifted[i] < drifted[j] })
	details, err := json.Marshal(map[string]any{
		"sales_recomputed":  sales,
		"customers_drifted": drifted,
	})
	if err != nil {
		return fmt.Errorf("audit details: %w", err)
	}
	var adminID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE role = 'admin' AND active = 1 ORDER BY id LIMIT 1`).Scan(&adminID)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("migration 24: no active admin, audit row skipped: %s", details)
		return nil
	}
	if err != nil {
		return fmt.Errorf("find admin for audit row: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO audit_log (user_id, action, entity, entity_id, details) VALUES (?, 'money_rounding_migration', 'system', 0, ?)`,
		adminID, string(details)); err != nil {
		return fmt.Errorf("audit row: %w", err)
	}
	return nil
}

func customerBalances(ctx context.Context, tx *sql.Tx) (map[int64]float64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM customers`)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	out := make(map[int64]float64, len(ids))
	for _, id := range ids {
		var b float64
		if err := tx.QueryRowContext(ctx, migrationBalanceSQL, id, id).Scan(&b); err != nil {
			return nil, err
		}
		out[id] = b
	}
	return out, nil
}

// mDivRound returns num/den rounded half away from zero (den > 0).
func mDivRound(num, den int64) int64 {
	if num < 0 {
		return -((-num + den/2) / den)
	}
	return (num + den/2) / den
}

// recomputeSales recomputes sale_items.subtotal and sales.subtotal/tax/total
// from the already rounded unit prices and discounts, and returns how many
// sales it visited. A total that differs from the sum of the sale's payments
// by at most $1 is set to that sum.
func recomputeSales(ctx context.Context, tx *sql.Tx) (int, error) {
	type item struct {
		id             int64
		qty, unit, dsc float64
		rate           float64
	}
	type saleRow struct {
		id       int64
		discount float64
		items    []item
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, discount FROM sales ORDER BY id`)
	if err != nil {
		return 0, fmt.Errorf("list sales: %w", err)
	}
	var sales []*saleRow
	byID := map[int64]*saleRow{}
	for rows.Next() {
		s := &saleRow{}
		if err := rows.Scan(&s.id, &s.discount); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan sale: %w", err)
		}
		sales = append(sales, s)
		byID[s.id] = s
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	irows, err := tx.QueryContext(ctx, `SELECT id, sale_id, quantity, unit_price, discount, tax_rate FROM sale_items ORDER BY id`)
	if err != nil {
		return 0, fmt.Errorf("list sale_items: %w", err)
	}
	for irows.Next() {
		var saleID int64
		var it item
		if err := irows.Scan(&it.id, &saleID, &it.qty, &it.unit, &it.dsc, &it.rate); err != nil {
			irows.Close()
			return 0, fmt.Errorf("scan sale_item: %w", err)
		}
		if s := byID[saleID]; s != nil {
			s.items = append(s.items, it)
		}
	}
	if err := irows.Err(); err != nil {
		irows.Close()
		return 0, err
	}
	irows.Close()

	for _, s := range sales {
		var subtotal, tax int64
		for _, it := range s.items {
			base := mDivRound(int64(math.Round(it.qty*1000))*int64(math.Round(it.unit)), 1000)
			line := base - int64(it.dsc)
			if line < 0 {
				line = 0
			}
			subtotal += line
			tax += mDivRound(line*int64(math.Round(it.rate*10000)), 10000)
			if _, err := tx.ExecContext(ctx, `UPDATE sale_items SET subtotal = ? WHERE id = ?`, line, it.id); err != nil {
				return 0, fmt.Errorf("update sale_item %d: %w", it.id, err)
			}
		}
		total := subtotal - int64(s.discount) + tax
		if total < 0 {
			total = 0
		}
		var paid sql.NullFloat64
		if err := tx.QueryRowContext(ctx, `SELECT SUM(amount) FROM sale_payments WHERE sale_id = ?`, s.id).Scan(&paid); err != nil {
			return 0, fmt.Errorf("sum payments of sale %d: %w", s.id, err)
		}
		if paid.Valid && math.Abs(paid.Float64-float64(total)) <= 1 {
			total = int64(paid.Float64)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sales SET subtotal = ?, tax = ?, total = ? WHERE id = ?`, subtotal, tax, total, s.id); err != nil {
			return 0, fmt.Errorf("update sale %d: %w", s.id, err)
		}
	}
	return len(sales), nil
}
