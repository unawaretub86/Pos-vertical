package handler

import (
	"net/http"
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
