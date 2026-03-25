package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/andreyo/pos/internal/repository"
)

// getPortfolio returns all customers with outstanding credit balance (balance > 0),
// ordered by balance descending. Reuses the CustomerRepository.List which already
// computes on-the-fly balances.
func (h *Handler) getPortfolio(w http.ResponseWriter, r *http.Request) {
	customers, err := h.customers.List(r.Context(), "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo cartera")
		return
	}

	// Filter to only customers with outstanding balance
	type portfolioEntry struct {
		ID           int64   `json:"id"`
		Name         string  `json:"name"`
		Phone        string  `json:"phone"`
		Balance      float64 `json:"balance"`
		OldestDebtAt any     `json:"oldest_debt_at"`
	}

	var entries []portfolioEntry
	for _, c := range customers {
		if c.Balance > 0 {
			entries = append(entries, portfolioEntry{
				ID:           c.ID,
				Name:         c.Name,
				Phone:        c.Phone,
				Balance:      c.Balance,
				OldestDebtAt: c.OldestDebtAt,
			})
		}
	}

	// Sort by balance descending (already done in most cases; ensure consistent output)
	// Simple insertion sort — portfolio lists are small
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Balance > entries[j-1].Balance; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	respond(w, http.StatusOK, entries)
}

// getSalesReport returns aggregated sales data for the given date range.
// Query params: from (RFC3339), to (RFC3339), group_by (day|week|month)
func (h *Handler) getSalesReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := repository.SalesSummaryParams{
		GroupBy: q.Get("group_by"),
	}
	if params.GroupBy == "" {
		params.GroupBy = "day"
	}
	if s := q.Get("from"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'from' inválido (use RFC3339)")
			return
		}
		params.From = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'to' inválido (use RFC3339)")
			return
		}
		params.To = &t
	}

	summary, err := h.reports.SalesSummary(r.Context(), params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error generando reporte de ventas")
		return
	}
	respond(w, http.StatusOK, summary)
}

// getTopProducts returns the top-selling products by revenue for the given date range.
// Query params: from (RFC3339), to (RFC3339), limit (default 10)
func (h *Handler) getTopProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 10
	if s := q.Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}

	var from, to *time.Time
	if s := q.Get("from"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'from' inválido (use RFC3339)")
			return
		}
		from = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'to' inválido (use RFC3339)")
			return
		}
		to = &t
	}

	products, err := h.reports.TopProducts(r.Context(), from, to, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error generando top productos")
		return
	}
	if products == nil {
		products = []*repository.TopProduct{}
	}
	respond(w, http.StatusOK, products)
}

// getRegistersReport returns all registers ordered by opened_at DESC.
// Reuses the existing RegisterService.List.
func (h *Handler) getRegistersReport(w http.ResponseWriter, r *http.Request) {
	registers, err := h.registers.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo historial de cajas")
		return
	}
	respond(w, http.StatusOK, registers)
}

// parseTime parses RFC3339 with or without sub-second precision.
// JavaScript's toISOString() always includes milliseconds (.000Z) which
// time.RFC3339 does not handle — time.RFC3339Nano covers both cases.
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
