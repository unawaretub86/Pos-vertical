package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type SalesSummaryParams struct {
	From    *time.Time
	To      *time.Time
	GroupBy string // "day" | "week" | "month"
}

type PaymentMethodTotal struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
	Count  int     `json:"count"`
}

type SalesPeriod struct {
	Period string  `json:"period"`
	Total  float64 `json:"total"`
	Count  int     `json:"count"`
}

type SalesSummary struct {
	TotalRevenue    float64              `json:"total_revenue"`
	TotalCost       float64              `json:"total_cost"`
	GrossMargin     float64              `json:"gross_margin"`
	TotalCount      int                  `json:"total_count"`
	ByPaymentMethod []PaymentMethodTotal `json:"by_payment_method"`
	Grouped         []SalesPeriod        `json:"grouped"`
}

type TopProduct struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	Revenue   float64 `json:"revenue"`
}

// ─── Interface ────────────────────────────────────────────────────────────────

type ReportRepository interface {
	SalesSummary(ctx context.Context, p SalesSummaryParams) (*SalesSummary, error)
	TopProducts(ctx context.Context, from, to *time.Time, limit int) ([]*TopProduct, error)
}

// ─── Implementation ───────────────────────────────────────────────────────────

type reportRepo struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) ReportRepository {
	return &reportRepo{db: db}
}

func (r *reportRepo) SalesSummary(ctx context.Context, p SalesSummaryParams) (*SalesSummary, error) {
	where, args := buildDateWhere(p.From, p.To, "s")

	// Total revenue, cost, count — use a derived table for cost to avoid
	// correlated-subquery issues in older SQLite versions.
	var summary SalesSummary
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT
			COALESCE(SUM(s.total), 0),
			COALESCE(SUM(c.sale_cost), 0),
			COUNT(DISTINCT s.id)
		FROM sales s
		LEFT JOIN (
			SELECT si.sale_id, SUM(si.quantity * p.cost) AS sale_cost
			FROM sale_items si
			JOIN products p ON p.id = si.product_id
			GROUP BY si.sale_id
		) c ON c.sale_id = s.id
		WHERE s.status = 'completed' %s`, where), args...,
	).Scan(&summary.TotalRevenue, &summary.TotalCost, &summary.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("report sales summary: %w", err)
	}
	summary.GrossMargin = summary.TotalRevenue - summary.TotalCost

	// By payment method — read from sale_payments so a mixed sale contributes
	// each portion to its own method instead of lumping the whole total under 'mixed'.
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT sp.method, COALESCE(SUM(sp.amount), 0), COUNT(*)
		FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.status = 'completed' %s
		GROUP BY sp.method
		ORDER BY SUM(sp.amount) DESC`, where), args...,
	)
	if err != nil {
		return nil, fmt.Errorf("report by payment method: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pm PaymentMethodTotal
		if err := rows.Scan(&pm.Method, &pm.Total, &pm.Count); err != nil {
			return nil, err
		}
		summary.ByPaymentMethod = append(summary.ByPaymentMethod, pm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Grouped by period. strftime() only parses a handful of well-known
	// text layouts (https://www.sqlite.org/lang_datefunc.html); rows whose
	// created_at was bound from a Go time.Time via a driver that doesn't
	// write ISO-8601 (no `_time_format=sqlite` DSN param, see roadmap R-05)
	// make strftime return NULL instead of erroring. COALESCE to '' so that
	// case groups under an empty period label instead of failing the Scan.
	groupFmt := groupFormat(p.GroupBy)
	groupRows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT COALESCE(strftime('%s', s.created_at), ''), COALESCE(SUM(s.total), 0), COUNT(*)
		FROM sales s
		WHERE s.status = 'completed' %s
		GROUP BY COALESCE(strftime('%s', s.created_at), '')
		ORDER BY COALESCE(strftime('%s', s.created_at), '') ASC`,
		groupFmt, where, groupFmt, groupFmt,
	), args...)
	if err != nil {
		return nil, fmt.Errorf("report grouped: %w", err)
	}
	defer groupRows.Close()
	for groupRows.Next() {
		var sp SalesPeriod
		if err := groupRows.Scan(&sp.Period, &sp.Total, &sp.Count); err != nil {
			return nil, err
		}
		summary.Grouped = append(summary.Grouped, sp)
	}

	return &summary, groupRows.Err()
}

func (r *reportRepo) TopProducts(ctx context.Context, from, to *time.Time, limit int) ([]*TopProduct, error) {
	if limit <= 0 {
		limit = 10
	}
	where, args := buildDateWhere(from, to, "s")
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			p.id,
			p.name,
			COALESCE(SUM(si.quantity), 0) AS qty,
			COALESCE(SUM(si.subtotal), 0) AS revenue
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed' %s
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
		LIMIT ?`, where), args...,
	)
	if err != nil {
		return nil, fmt.Errorf("report top products: %w", err)
	}
	defer rows.Close()

	var list []*TopProduct
	for rows.Next() {
		tp := &TopProduct{}
		if err := rows.Scan(&tp.ProductID, &tp.Name, &tp.Quantity, &tp.Revenue); err != nil {
			return nil, err
		}
		list = append(list, tp)
	}
	return list, rows.Err()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildDateWhere builds a WHERE fragment for date filtering. The range is
// half-open [from, to) (R-06, AC-20 of US-061 contract): "to" is exclusive
// so two contiguous ranges sharing an exact boundary (e.g. "hoy" and
// "mañana") never double-count nor drop the row created exactly at the
// boundary instant.
// tableAlias (optional) qualifies the column to avoid ambiguity in JOINs
// e.g. buildDateWhere(from, to, "s") → " AND s.created_at >= ?"
func buildDateWhere(from, to *time.Time, tableAlias ...string) (string, []any) {
	col := "created_at"
	if len(tableAlias) > 0 && tableAlias[0] != "" {
		col = tableAlias[0] + ".created_at"
	}
	where := ""
	var args []any
	if from != nil {
		where += " AND " + col + " >= ?"
		args = append(args, *from)
	}
	if to != nil {
		where += " AND " + col + " < ?"
		args = append(args, *to)
	}
	return where, args
}

func groupFormat(groupBy string) string {
	switch groupBy {
	case "week":
		return "%Y-W%W"
	case "month":
		return "%Y-%m"
	default: // "day"
		return "%Y-%m-%d"
	}
}
