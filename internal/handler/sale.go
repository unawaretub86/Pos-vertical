package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) createSale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		model.Sale
		BuyerEmail string `json:"buyer_email"`
	}
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	claims := appmw.GetClaims(r)
	if claims != nil {
		req.Sale.UserID = claims.UserID
	}

	var subtotal float64
	for i := range req.Sale.Items {
		item := &req.Sale.Items[i]
		item.Subtotal = item.UnitPrice*item.Quantity - item.Discount
		subtotal += item.Subtotal
	}
	req.Sale.Subtotal = subtotal
	req.Sale.Total = subtotal - req.Sale.Discount + req.Sale.Tax

	created, err := h.sales.Create(r.Context(), &req.Sale)
	if errors.Is(err, repository.ErrBusy) {
		respondError(w, http.StatusServiceUnavailable, msgBusy)
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.factelect != nil {
		go h.factelect.EmitFromSale(r.Context(), created.ID, req.BuyerEmail)
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
	var cnErr *repository.ErrRequiresCreditNote
	if errors.As(err, &cnErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error":                "sale_has_accepted_invoice",
			"message":              cnErr.Error(),
			"document_id":          cnErr.DocumentID,
			"cude":                 cnErr.CUDE,
			"requires_credit_note": true,
		})
		return
	}
	if errors.Is(err, repository.ErrBusy) {
		respondError(w, http.StatusServiceUnavailable, msgBusy)
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, sale)
}

func (h *Handler) voidSaleWithCreditNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Motive string `json:"motive"`
	}
	if err := decode(r, &body); err != nil || body.Motive == "" {
		respondError(w, http.StatusBadRequest, "motive requerido")
		return
	}
	if h.factelect == nil {
		respondError(w, http.StatusBadRequest, "facturación electrónica no habilitada")
		return
	}
	if _, err := h.factelect.CreateCreditNote(r.Context(), id, body.Motive); err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	claims := appmw.GetClaims(r)
	var userID int64
	if claims != nil {
		userID = claims.UserID
	}
	sale, err := h.sales.Void(r.Context(), id, userID)
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

	ticket, err := h.buildTicket(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "venta no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo venta")
		return
	}
	respond(w, http.StatusOK, ticket)
}

// buildTicket assembles the ticket data of a sale. It is shared by the JSON
// ticket endpoint and the ESC/POS printer so both show the same content.
func (h *Handler) buildTicket(ctx context.Context, id int64) (*model.TicketData, error) {
	sale, err := h.sales.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sale %d: %w", id, err)
	}

	// Build enriched ticket items with product names
	ticketItems := make([]model.TicketItem, 0, len(sale.Items))
	for _, item := range sale.Items {
		name := "Producto"
		unit := "unit"
		if prod, err := h.products.GetByID(ctx, item.ProductID); err == nil {
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
		if cust, err := h.customers.GetByID(ctx, *sale.CustomerID); err == nil {
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
		Payments:      sale.Payments,
		Customer:      customer,
		Business:      business,
	}
	return &ticket, nil
}
