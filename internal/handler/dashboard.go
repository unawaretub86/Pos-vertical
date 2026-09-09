package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/andreyo/pos/internal/repository"
)

// ─── /dashboard/today ────────────────────────────────────────────────────────

type topMarginItem struct {
	Name      string  `json:"name"`
	MarginPct float64 `json:"margin_pct"`
	UnitsSold float64 `json:"units_sold"`
}

type lowStockItem struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Stock     float64 `json:"stock"`
	MinStock  float64 `json:"min_stock"`
}

type agingReceivable struct {
	CustomerID  int64   `json:"customer_id"`
	Name        string  `json:"name"`
	Balance     float64 `json:"balance"`
	DaysOverdue int     `json:"days_overdue"`
}

type todayResponse struct {
	SoldToday             float64           `json:"sold_today"`
	CreditNewToday        float64           `json:"credit_new_today"`
	CreditCollectedToday  float64           `json:"credit_collected_today"`
	CashEstimated         float64           `json:"cash_estimated"`
	TopProductName        string            `json:"top_product_name"`
	TopProductQty         float64           `json:"top_product_qty"`
	GrossProfit           float64           `json:"gross_profit"`
	ExpensesToday         float64           `json:"expenses_today"`
	NetProfit             float64           `json:"net_profit"`
	UnconfiguredCostCount int               `json:"unconfigured_cost_count"`
	TopByMargin           []topMarginItem   `json:"top_by_margin"`
	LowStock              []lowStockItem    `json:"low_stock"`
	AgingReceivables      []agingReceivable `json:"aging_receivables"`
}

func (h *Handler) getDashboardToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := h.db.DB

	now := time.Now()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := todayStart.Add(24 * time.Hour)

	var resp todayResponse

	// Vendido hoy (all completed sales)
	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total), 0)
		FROM sales
		WHERE status = 'completed' AND created_at >= ? AND created_at < ?`,
		todayStart, todayEnd,
	).Scan(&resp.SoldToday) //nolint:errcheck

	// Fiado nuevo hoy
	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(sp.amount), 0)
		FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.status = 'completed' AND sp.method = 'credit' AND s.created_at >= ? AND s.created_at < ?`,
		todayStart, todayEnd,
	).Scan(&resp.CreditNewToday) //nolint:errcheck

	// Cobrado de fiado hoy (customer_payments)
	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM customer_payments
		WHERE created_at >= ? AND created_at < ?`,
		todayStart, todayEnd,
	).Scan(&resp.CreditCollectedToday) //nolint:errcheck

	// Efectivo en caja estimado = cash sales today − cash_out movements today
	var cashSales, cashOut float64
	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(sp.amount), 0)
		FROM sale_payments sp
		JOIN sales s ON s.id = sp.sale_id
		WHERE s.status = 'completed' AND sp.method = 'cash' AND s.created_at >= ? AND s.created_at < ?`,
		todayStart, todayEnd,
	).Scan(&cashSales) //nolint:errcheck

	db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM register_movements
		WHERE type = 'cash_out' AND created_at >= ? AND created_at < ?`,
		todayStart, todayEnd,
	).Scan(&cashOut) //nolint:errcheck

	resp.CashEstimated = cashSales - cashOut

	// Producto más vendido hoy (by quantity)
	var name sql.NullString
	var qty sql.NullFloat64
	db.QueryRowContext(ctx, `
		SELECT p.name, COALESCE(SUM(si.quantity), 0) AS qty
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed' AND s.created_at >= ? AND s.created_at < ?
		GROUP BY p.id, p.name
		ORDER BY qty DESC
		LIMIT 1`,
		todayStart, todayEnd,
	).Scan(&name, &qty) //nolint:errcheck

	if name.Valid {
		resp.TopProductName = name.String
		resp.TopProductQty = qty.Float64
	}

	// Ganancia bruta hoy + conteo de productos sin costo configurado
	db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN p.cost > 0 THEN (si.unit_price - p.cost) * si.quantity ELSE 0 END), 0),
			COUNT(DISTINCT CASE WHEN p.cost = 0 THEN si.product_id END)
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed' AND s.created_at >= ? AND s.created_at < ?`,
		todayStart, todayEnd,
	).Scan(&resp.GrossProfit, &resp.UnconfiguredCostCount) //nolint:errcheck

	// Gastos de hoy → ganancia neta (US-049)
	resp.ExpensesToday, _ = h.expenses.SumRange(ctx, todayStart, todayEnd)
	resp.NetProfit = resp.GrossProfit - resp.ExpensesToday

	// Top 5 productos por margen (solo cost > 0)
	marginRows, err := db.QueryContext(ctx, `
		SELECT
			p.name,
			((AVG(si.unit_price) - p.cost) / AVG(si.unit_price)) * 100 AS margin_pct,
			SUM(si.quantity) AS units_sold
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		JOIN sales s ON s.id = si.sale_id
		WHERE s.status = 'completed'
			AND s.created_at >= ? AND s.created_at < ?
			AND p.cost > 0
		GROUP BY p.id, p.name, p.cost
		ORDER BY margin_pct DESC
		LIMIT 5`,
		todayStart, todayEnd,
	)
	if err == nil {
		defer marginRows.Close()
		var items []topMarginItem
		for marginRows.Next() {
			var item topMarginItem
			if scanErr := marginRows.Scan(&item.Name, &item.MarginPct, &item.UnitsSold); scanErr == nil {
				items = append(items, item)
			}
		}
		if len(items) >= 2 {
			resp.TopByMargin = items
		}
	}
	if resp.TopByMargin == nil {
		resp.TopByMargin = []topMarginItem{}
	}

	// US-042b: productos bajo stock mínimo (top 5 por déficit)
	lowRows, err := db.QueryContext(ctx, `
		SELECT id, name, stock, min_stock
		FROM products
		WHERE active = 1 AND min_stock > 0 AND stock < min_stock
		ORDER BY (min_stock - stock) DESC
		LIMIT 5`)
	if err == nil {
		defer lowRows.Close()
		for lowRows.Next() {
			var item lowStockItem
			if scanErr := lowRows.Scan(&item.ProductID, &item.Name, &item.Stock, &item.MinStock); scanErr == nil {
				resp.LowStock = append(resp.LowStock, item)
			}
		}
	}
	if resp.LowStock == nil {
		resp.LowStock = []lowStockItem{}
	}

	// US-041: cartera envejecida — top 5 clientes con deuda, ordenados por días (más urgente primero)
	agingRows, err := db.QueryContext(ctx, `
		SELECT
			c.id,
			c.name,
			COALESCE(SUM(sp.amount), 0) - COALESCE(SUM(cp.paid), 0) AS balance,
			CAST(julianday('now') - julianday(MIN(s.created_at)) AS INTEGER) AS days_overdue
		FROM customers c
		JOIN sales s ON s.customer_id = c.id AND s.status = 'completed'
		JOIN sale_payments sp ON sp.sale_id = s.id AND sp.method = 'credit'
		LEFT JOIN (
			SELECT customer_id, SUM(amount) AS paid FROM customer_payments GROUP BY customer_id
		) cp ON cp.customer_id = c.id
		GROUP BY c.id, c.name
		HAVING balance > 0
		ORDER BY days_overdue DESC
		LIMIT 5`)
	if err == nil {
		defer agingRows.Close()
		for agingRows.Next() {
			var item agingReceivable
			if scanErr := agingRows.Scan(&item.CustomerID, &item.Name, &item.Balance, &item.DaysOverdue); scanErr == nil {
				resp.AgingReceivables = append(resp.AgingReceivables, item)
			}
		}
	}
	if resp.AgingReceivables == nil {
		resp.AgingReceivables = []agingReceivable{}
	}

	respond(w, http.StatusOK, resp)
}

// dashboardResponse is the aggregated payload for the dashboard screen.
type dashboardResponse struct {
	Today       salesSnap          `json:"today"`
	Yesterday   salesSnap          `json:"yesterday"`
	Register    any                `json:"register"` // null | RegisterSummary
	Portfolio   portfolioSnap      `json:"portfolio"`
	RecentSales []recentSaleRow    `json:"recent_sales"`
}

type salesSnap struct {
	TotalRevenue float64 `json:"total_revenue"`
	TotalCount   int     `json:"total_count"`
}

type portfolioSnap struct {
	Total float64 `json:"total"`
	Count int     `json:"count"`
}

type recentSaleRow struct {
	ID            int64    `json:"id"`
	CreatedAt     string   `json:"created_at"`
	CustomerID    *int64   `json:"customer_id"`
	CustomerName  string   `json:"customer_name"`
	ItemCount     int      `json:"item_count"`
	Total         float64  `json:"total"`
	PaymentMethod string   `json:"payment_method"`
	Status        string   `json:"status"`
}

func (h *Handler) getDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := h.db.DB

	// ── Date ranges ──────────────────────────────────────────────────────────
	now := time.Now()
	loc := now.Location()

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := todayStart.Add(24 * time.Hour)
	yesterdayStart := todayStart.Add(-24 * time.Hour)
	yesterdayEnd := todayStart

	// ── Today's sales snapshot ───────────────────────────────────────────────
	today, _ := h.reports.SalesSummary(ctx, repository.SalesSummaryParams{
		From:    &todayStart,
		To:      &todayEnd,
		GroupBy: "day",
	})

	// ── Yesterday's sales snapshot ───────────────────────────────────────────
	yesterday, _ := h.reports.SalesSummary(ctx, repository.SalesSummaryParams{
		From:    &yesterdayStart,
		To:      &yesterdayEnd,
		GroupBy: "day",
	})

	// ── Open register ────────────────────────────────────────────────────────
	var registerData any
	reg, err := h.registers.GetOpen(ctx)
	if err == nil && reg != nil {
		if summary, err := h.registers.Summary(ctx, reg.ID); err == nil {
			registerData = summary
		}
	}

	// ── Portfolio snapshot ───────────────────────────────────────────────────
	var portfolio portfolioSnap
	customers, err := h.customers.List(ctx, "")
	if err == nil {
		for _, c := range customers {
			if c.Balance > 0 {
				portfolio.Total += c.Balance
				portfolio.Count++
			}
		}
	}

	// ── Recent sales (last 8) with item count and customer name ──────────────
	rows, err := db.QueryContext(ctx, `
		SELECT
			s.id,
			s.created_at,
			s.customer_id,
			COALESCE(c.name, '') AS customer_name,
			COUNT(si.id)         AS item_count,
			s.total,
			s.payment_method,
			s.status
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
		LEFT JOIN sale_items si ON si.sale_id = s.id
		WHERE s.status != 'voided'
		GROUP BY s.id
		ORDER BY s.created_at DESC
		LIMIT 8
	`)
	var recentSales []recentSaleRow
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var row recentSaleRow
			var customerID sql.NullInt64
			var createdAt time.Time
			if scanErr := rows.Scan(
				&row.ID, &createdAt, &customerID, &row.CustomerName,
				&row.ItemCount, &row.Total, &row.PaymentMethod, &row.Status,
			); scanErr == nil {
				if customerID.Valid {
					v := customerID.Int64
					row.CustomerID = &v
				}
				row.CreatedAt = createdAt.Format(time.RFC3339)
				recentSales = append(recentSales, row)
			}
		}
	}
	if recentSales == nil {
		recentSales = []recentSaleRow{}
	}

	// ── Assemble response ─────────────────────────────────────────────────────
	resp := dashboardResponse{
		Register:    registerData,
		Portfolio:   portfolio,
		RecentSales: recentSales,
	}
	if today != nil {
		resp.Today = salesSnap{TotalRevenue: today.TotalRevenue, TotalCount: today.TotalCount}
	}
	if yesterday != nil {
		resp.Yesterday = salesSnap{TotalRevenue: yesterday.TotalRevenue, TotalCount: yesterday.TotalCount}
	}

	respond(w, http.StatusOK, resp)
}
