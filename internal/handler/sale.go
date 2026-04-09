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
	if r.URL.Query().Get("include_voided") == "true" {
		filter.IncludeVoided = true
	}
	sales, err := h.sales.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando ventas")
		return
	}
	respond(w, http.StatusOK, sales)
}

func (h *Handler) voidSale(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	claims := appmw.GetClaims(r)
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}
	sale, err := h.sales.Void(r.Context(), id, userID)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "venta no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, sale)
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

func (h *Handler) getTicket(w http.ResponseWriter, r *http.Request) {
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

	// Build enriched ticket items with product names
	ticketItems := make([]model.TicketItem, 0, len(sale.Items))
	for _, item := range sale.Items {
		name := "Producto"
		unit := "unit"
		if prod, err := h.products.GetByID(r.Context(), item.ProductID); err == nil {
			name = prod.Name
			unit = prod.Unit
		}
		ticketItems = append(ticketItems, model.TicketItem{
			ProductName: name,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Discount:    item.Discount,
			Subtotal:    item.Subtotal,
			Unit:        unit,
		})
	}

	// Get customer name for credit sales
	var customer *model.Customer
	if sale.CustomerID != nil {
		if cust, err := h.customers.GetByID(r.Context(), *sale.CustomerID); err == nil {
			customer = cust
		}
	}

	// Business config
	var business model.TicketBusiness
	if cfg, err := h.setup.Get(); err == nil && cfg != nil {
		business = model.TicketBusiness{
			Name:         cfg.Name,
			Address:      cfg.Address,
			NIT:          cfg.NIT,
			TicketHeader: cfg.TicketHeader,
			TicketFooter: cfg.TicketFooter,
		}
	}

	ticket := model.TicketData{
		SaleID:        sale.ID,
		CreatedAt:     sale.CreatedAt,
		PaymentMethod: sale.PaymentMethod,
		CashTendered:  sale.CashTendered,
		ChangeAmount:  sale.ChangeAmount,
		Reference:     sale.Reference,
		Status:        sale.Status,
		Subtotal:      sale.Subtotal,
		Discount:      sale.Discount,
		Tax:           sale.Tax,
		Total:         sale.Total,
		Items:         ticketItems,
		Customer:      customer,
		Business:      business,
	}
	respond(w, http.StatusOK, ticket)
}
