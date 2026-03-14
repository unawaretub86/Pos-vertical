package handler

import (
	"errors"
	"net/http"
	"strconv"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) createSale(w http.ResponseWriter, r *http.Request) {
	var sale model.Sale
	if err := decode(r, &sale); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	// Set the user from JWT claims
	claims := appmw.GetClaims(r)
	if claims != nil {
		sale.UserID = claims.UserID
	}

	// Compute subtotals for items
	var subtotal float64
	for i := range sale.Items {
		item := &sale.Items[i]
		item.Subtotal = item.UnitPrice*item.Quantity - item.Discount
		subtotal += item.Subtotal
	}
	sale.Subtotal = subtotal
	sale.Total = subtotal - sale.Discount + sale.Tax

	created, err := h.sales.Create(r.Context(), &sale)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, created)
}

func (h *Handler) listSales(w http.ResponseWriter, r *http.Request) {
	filter := repository.SaleFilter{}
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil {
			filter.Limit = l
		}
	}
	sales, err := h.sales.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando ventas")
		return
	}
	respond(w, http.StatusOK, sales)
}

func (h *Handler) getSale(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	sale, err := h.sales.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "venta no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo venta")
		return
	}
	respond(w, http.StatusOK, sale)
}
