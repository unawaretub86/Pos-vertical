package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) listCustomers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	customers, err := h.customers.List(r.Context(), search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando clientes")
		return
	}

	// Filter to only customers with balance > 0 when has_balance=true
	if r.URL.Query().Get("has_balance") == "true" {
		filtered := customers[:0]
		for _, c := range customers {
			if c.Balance > 0 {
				filtered = append(filtered, c)
			}
		}
		customers = filtered
	}

	total, err := h.customers.TotalPortfolioBalance(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error calculando cartera")
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"customers":       customers,
		"total_portfolio": total,
	})
}

func (h *Handler) createCustomer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Phone       string `json:"phone"`
		Address     string `json:"address"`
		Notes       string `json:"notes"`
		CreditLimit *int64 `json:"credit_limit"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	c, err := h.customers.Create(r.Context(), body.Name, body.Phone, body.Address, body.Notes, body.CreditLimit)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, c)
}

func (h *Handler) getCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	c, err := h.customers.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo cliente")
		return
	}
	respond(w, http.StatusOK, c)
}

func (h *Handler) updateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Phone       string `json:"phone"`
		Address     string `json:"address"`
		Notes       string `json:"notes"`
		CreditLimit *int64 `json:"credit_limit"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	c, err := h.customers.Update(r.Context(), id, body.Name, body.Phone, body.Address, body.Notes, body.CreditLimit)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, c)
}

func (h *Handler) deleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	err = h.customers.Delete(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	}
	if errors.Is(err, service.ErrCustomerHasBalance) {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error eliminando cliente")
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "cliente eliminado"})
}

func (h *Handler) getCustomerBalance(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	// Verify customer exists
	if _, err := h.customers.GetByID(r.Context(), id); errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo cliente")
		return
	}
	balance, err := h.customers.Balance(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error calculando saldo")
		return
	}
	respond(w, http.StatusOK, map[string]float64{"balance": balance})
}

func (h *Handler) listCustomerPayments(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if _, err := h.customers.GetByID(r.Context(), id); errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo cliente")
		return
	}
	payments, err := h.customers.ListPayments(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando pagos")
		return
	}
	respond(w, http.StatusOK, payments)
}

func (h *Handler) createCustomerPayment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Amount float64 `json:"amount"`
		Note   string  `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	payment, err := h.customers.CreatePayment(r.Context(), id, body.Amount, body.Note)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, payment)
}

func (h *Handler) getCustomerDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	customer, err := h.customers.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo cliente")
		return
	}
	balance, err := h.customers.Balance(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error calculando saldo")
		return
	}
	payments, err := h.customers.ListPayments(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando pagos")
		return
	}
	creditSales, err := h.customers.ListCreditSales(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando ventas a crédito")
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"customer":     customer,
		"balance":      balance,
		"payments":     payments,
		"credit_sales": creditSales,
	})
}

func (h *Handler) getCustomerPOSSummary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	summary, err := h.customers.GetPOSSummary(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "cliente no encontrado")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo cliente")
		return
	}
	respond(w, http.StatusOK, summary)
}
