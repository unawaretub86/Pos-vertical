package handler

import (
	"errors"
	"net/http"
	"strconv"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
)

func (h *Handler) createPurchase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SupplierID int64  `json:"supplier_id"`
		Reference  string `json:"reference"`
		Items      []struct {
			ProductID int64   `json:"product_id"`
			Quantity  float64 `json:"quantity"`
			UnitCost  float64 `json:"unit_cost"`
		} `json:"items"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	purchase := &model.Purchase{
		SupplierID: body.SupplierID,
		Reference:  body.Reference,
	}
	for _, item := range body.Items {
		purchase.Items = append(purchase.Items, model.PurchaseItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitCost:  item.UnitCost,
		})
	}

	claims := appmw.GetClaims(r)
	created, err := h.purchases.Create(r.Context(), claims.UserID, purchase)
	switch {
	case errors.Is(err, service.ErrPurchaseNoItems),
		errors.Is(err, service.ErrPurchaseInvalidQty),
		errors.Is(err, service.ErrPurchaseInvalidCost),
		errors.Is(err, service.ErrPurchaseSupplierGone):
		respondError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		respondError(w, http.StatusInternalServerError, "error registrando compra")
		return
	}
	respond(w, http.StatusCreated, created)
}

func (h *Handler) listPurchases(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.PurchaseFilter{}

	if s := q.Get("supplier_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'supplier_id' inválido")
			return
		}
		filter.SupplierID = &id
	}
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

	list, err := h.purchases.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando compras")
		return
	}
	if list == nil {
		list = []*model.Purchase{}
	}
	respond(w, http.StatusOK, list)
}
