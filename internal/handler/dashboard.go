package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/andreyo/pos/internal/repository"
)

// ─── /dashboard/today ────────────────────────────────────────────────────────

// JSON item types of /dashboard/today. They alias the repository types, which
// carry the same JSON tags, so the payload stays byte-for-byte the same.
type (
	topMarginItem   = repository.MarginItem
	lowStockItem    = repository.LowStockItem
	agingReceivable = repository.AgingReceivable
)

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

const dashboardListLimit = 5

// getDashboardToday computes today's window [00:00, next 00:00) in local time
// and assembles the response from MetricsRepository. Any error → 500 with no
// partial figures.
func (h *Handler) getDashboardToday(w http.ResponseWriter, r *http.Request) {
	if err := h.writeDashboardToday(w, r); err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo métricas")
	}
}

func (h *Handler) writeDashboardToday(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	now := time.Now()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := todayStart.Add(24 * time.Hour)

	var resp todayResponse
	var err error

	if resp.SoldToday, _, err = h.metrics.Sold(ctx, todayStart, todayEnd); err != nil {
		return err
	}

	if resp.CreditNewToday, resp.CreditCollectedToday, err = h.metrics.CreditFlow(ctx, todayStart, todayEnd); err != nil {
		return err
	}

	cashSales, cashOut, err := h.metrics.CashFlow(ctx, todayStart, todayEnd)
	if err != nil {
		return err
	}
	resp.CashEstimated = cashSales - cashOut

	if resp.TopProductName, resp.TopProductQty, err = h.metrics.TopSeller(ctx, todayStart, todayEnd); err != nil {
		return err
	}

	if resp.GrossProfit, resp.ExpensesToday, resp.NetProfit, resp.UnconfiguredCostCount, err =
		h.metrics.Profit(ctx, todayStart, todayEnd); err != nil {
		return err
	}

	margin, err := h.metrics.TopByMargin(ctx, todayStart, todayEnd, dashboardListLimit)
	if err != nil {
		return err
	}
	// Presentation rule: the margin ranking is only meaningful with 2+ products.
	resp.TopByMargin = []topMarginItem{}
	if len(margin) >= 2 {
		resp.TopByMargin = margin
	}

	if resp.LowStock, err = h.metrics.LowStock(ctx, dashboardListLimit); err != nil {
		return err
	}

	if resp.AgingReceivables, err = h.metrics.AgingReceivables(ctx, dashboardListLimit); err != nil {
		return err
	}

	respond(w, http.StatusOK, resp)
	return nil
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
