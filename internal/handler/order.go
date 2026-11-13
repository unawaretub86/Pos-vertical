package handler

import (
	"errors"
	"net/http"
	"strconv"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
	"github.com/go-chi/chi/v5"
)

// msgBusy is sent with 503 when SQLITE_BUSY prevented a write; a 503 means
// nothing was committed, so the client may retry.
const msgBusy = "El sistema está ocupado, reintenta en un momento"

// respondOrderError maps repository/service errors to HTTP statuses.
func respondOrderError(w http.ResponseWriter, err error, fallback string) {
	var ve *service.ValidationError
	switch {
	case errors.As(err, &ve):
		respondError(w, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, repository.ErrBusy):
		respondError(w, http.StatusServiceUnavailable, msgBusy)
	case errors.Is(err, repository.ErrNotFound), errors.Is(err, repository.ErrTableInactive):
		respondError(w, http.StatusNotFound, "no encontrado")
	case errors.Is(err, repository.ErrProductInactive):
		respondError(w, http.StatusBadRequest, "el producto está inactivo")
	case errors.Is(err, repository.ErrTableHasOpenOrder):
		respondError(w, http.StatusConflict, "La mesa ya tiene un pedido abierto")
	case errors.Is(err, repository.ErrOrderNotOpen):
		respondError(w, http.StatusConflict, "El pedido no está abierto")
	case errors.Is(err, repository.ErrItemVoid):
		respondError(w, http.StatusConflict, "El ítem ya está anulado")
	default:
		respondError(w, http.StatusInternalServerError, fallback)
	}
}

func idParam(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	return id, err == nil
}

func (h *Handler) openOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TableID int64  `json:"table_id"`
		Type    string `json:"type"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	o, err := h.orders.Open(r.Context(), body.TableID, body.Type, appmw.GetClaims(r).UserID)
	if err != nil {
		respondOrderError(w, err, "error abriendo el pedido")
		return
	}
	respond(w, http.StatusCreated, o)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "open"
	}
	list, err := h.orders.List(r.Context(), status)
	if err != nil {
		respondOrderError(w, err, "error listando pedidos")
		return
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	o, err := h.orders.Get(r.Context(), id)
	if err != nil {
		respondOrderError(w, err, "error obteniendo el pedido")
		return
	}
	respond(w, http.StatusOK, o)
}

func (h *Handler) addOrderItem(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		ProductID int64  `json:"product_id"`
		Quantity  int    `json:"quantity"`
		Note      string `json:"note"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	it, err := h.orders.AddItem(r.Context(), id, body.ProductID, body.Quantity, body.Note)
	if err != nil {
		respondOrderError(w, err, "error agregando el ítem")
		return
	}
	respond(w, http.StatusCreated, it)
}

func (h *Handler) updateOrderItem(w http.ResponseWriter, r *http.Request) {
	id, ok1 := idParam(r, "id")
	itemID, ok2 := idParam(r, "itemId")
	if !ok1 || !ok2 {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Quantity *int    `json:"quantity"`
		Note     *string `json:"note"`
		Status   *string `json:"status"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	it, err := h.orders.UpdateItem(r.Context(), id, itemID,
		repository.ItemPatch{Quantity: body.Quantity, Note: body.Note, Status: body.Status})
	if err != nil {
		respondOrderError(w, err, "error actualizando el ítem")
		return
	}
	respond(w, http.StatusOK, it)
}

func (h *Handler) voidOrderItem(w http.ResponseWriter, r *http.Request) {
	id, ok1 := idParam(r, "id")
	itemID, ok2 := idParam(r, "itemId")
	if !ok1 || !ok2 {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	it, err := h.orders.VoidItem(r.Context(), id, itemID, appmw.GetClaims(r).UserID)
	if err != nil {
		respondOrderError(w, err, "error anulando el ítem")
		return
	}
	respond(w, http.StatusOK, it)
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	if err := h.orders.Cancel(r.Context(), id, appmw.GetClaims(r).UserID, body.Reason); err != nil {
		respondOrderError(w, err, "error cancelando el pedido")
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "pedido cancelado"})
}

func (h *Handler) checkoutOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "id")
	if !ok {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req struct {
		model.Sale
		BuyerEmail string `json:"buyer_email"`
	}
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	order, err := h.orders.Get(r.Context(), id)
	if err != nil {
		respondOrderError(w, err, "error obteniendo el pedido")
		return
	}
	if order.Status != "open" {
		respondError(w, http.StatusBadRequest, "el pedido no está abierto")
		return
	}

	var saleItems []model.SaleItem
	for _, it := range order.Items {
		if it.Status == "void" {
			continue
		}
		saleItems = append(saleItems, model.SaleItem{
			ProductID: it.ProductID,
			Quantity:  float64(it.Quantity),
			UnitPrice: it.UnitPrice,
		})
	}
	if len(saleItems) == 0 {
		respondError(w, http.StatusBadRequest, "el pedido no tiene ítems válidos para cobrar")
		return
	}
	req.Sale.Items = saleItems
	req.Sale.OrderID = &id

	claims := appmw.GetClaims(r)
	if claims != nil {
		req.Sale.UserID = claims.UserID
	}

	// The service prices the sale (including IVA) from the products' current
	// tax_rate; the order total is not trusted from the client.
	req.Sale.NoClientTotal = true

	created, err := h.sales.Create(r.Context(), &req.Sale)
	if errors.Is(err, repository.ErrBusy) {
		respondError(w, http.StatusServiceUnavailable, msgBusy)
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if h.factelect != nil && req.BuyerEmail != "" {
		go h.factelect.EmitFromSale(r.Context(), created.ID, req.BuyerEmail)
	}

	respond(w, http.StatusCreated, map[string]any{
		"order_id": id,
		"sale_id":  created.ID,
		"total":    created.Total,
	})
}
