package handler

import (
	"errors"
	"net/http"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
)

func (h *Handler) createExpense(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
		Note     string  `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	claims := appmw.GetClaims(r)
	expense, err := h.expenses.Create(r.Context(), claims.UserID, body.Category, body.Amount, body.Note)
	if errors.Is(err, service.ErrInvalidExpenseAmount) || errors.Is(err, service.ErrNotWholePeso) || errors.Is(err, service.ErrInvalidExpenseCategory) {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error registrando gasto")
		return
	}
	respond(w, http.StatusCreated, expense)
}

func (h *Handler) listExpenses(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.ExpenseFilter{Category: q.Get("category")}

	if s := q.Get("from"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'from' inválido (use RFC3339)")
			return
		}
		filter.From = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'to' inválido (use RFC3339)")
			return
		}
		filter.To = &t
	}

	list, err := h.expenses.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando gastos")
		return
	}
	if list == nil {
		list = []*model.Expense{}
	}

	var total float64
	for _, e := range list {
		total += e.Amount
	}

	respond(w, http.StatusOK, map[string]any{"expenses": list, "total": total})
}
