package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) createStockEntry(w http.ResponseWriter, r *http.Request) {
	var entry model.StockEntry
	if err := decode(r, &entry); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	if entry.ProductID == 0 || entry.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "product_id y quantity son requeridos")
		return
	}
	created, err := h.stockEntries.Create(r.Context(), &entry)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "producto no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, created)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.ProductFilter{
		Search:     q.Get("search"),
		ActiveOnly: q.Get("active") != "false",
	}
	if catStr := q.Get("category_id"); catStr != "" {
		if catID, err := strconv.ParseInt(catStr, 10, 64); err == nil {
			filter.CategoryID = &catID
		}
	}

	products, err := h.products.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando productos")
		return
	}
	respond(w, http.StatusOK, products)
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var p model.Product
	if err := decode(r, &p); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	// Defaults
	if p.Type == "" {
		p.Type = "product"
	}
	if p.Unit == "" {
		p.Unit = "unit"
	}
	if p.WholesalePrice != nil && *p.WholesalePrice > 0 && *p.WholesalePrice < p.Cost {
		respondError(w, http.StatusBadRequest, "el precio mayorista no puede ser menor al costo")
		return
	}

	created, err := h.products.Create(r.Context(), &p)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, created)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.products.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "producto no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo producto")
		return
	}
	respond(w, http.StatusOK, p)
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var p model.Product
	if err := decode(r, &p); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	p.ID = id
	if p.WholesalePrice != nil && *p.WholesalePrice > 0 && *p.WholesalePrice < p.Cost {
		respondError(w, http.StatusBadRequest, "el precio mayorista no puede ser menor al costo")
		return
	}

	before, beforeErr := h.products.GetByID(r.Context(), id)

	updated, err := h.products.Update(r.Context(), &p)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "producto no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if beforeErr == nil {
		if claims := appmw.GetClaims(r); claims != nil {
			logPriceFieldChange(r, h, claims.UserID, updated.ID, "sale_price", before.SalePrice, updated.SalePrice)
			logPriceFieldChange(r, h, claims.UserID, updated.ID, "cost", before.Cost, updated.Cost)
		}
	}

	respond(w, http.StatusOK, updated)
}

// logPriceFieldChange writes a product_price_change audit entry when a
// tracked field actually changed value (no-op saves must not create noise).
func logPriceFieldChange(r *http.Request, h *Handler, userID, productID int64, field string, from, to float64) {
	if from == to {
		return
	}
	detailsBytes, _ := json.Marshal(map[string]any{"field": field, "from": from, "to": to})
	_ = repository.InsertAuditLog(r.Context(), h.db, userID, "product_price_change", "product", productID, string(detailsBytes))
}

func (h *Handler) getProductByBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := chi.URLParam(r, "barcode")
	if barcode == "" {
		respondError(w, http.StatusBadRequest, "barcode requerido")
		return
	}
	p, err := h.products.GetByBarcode(r.Context(), barcode)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "producto no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error buscando producto")
		return
	}
	respond(w, http.StatusOK, p)
}

func (h *Handler) listLowStockProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.LowStock(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo productos bajo stock")
		return
	}
	respond(w, http.StatusOK, products)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.products.Delete(r.Context(), id); errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "producto no encontrado")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "error eliminando producto")
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "producto eliminado"})
}

func (h *Handler) getPublicCatalog(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		respondError(w, http.StatusNotFound, "not found")
		return
	}

	cfg, err := h.setup.Get()
	if err != nil || !cfg.CatalogEnabled || cfg.CatalogToken != token {
		respondError(w, http.StatusNotFound, "not found")
		return
	}

	products, err := h.products.List(r.Context(), repository.ProductFilter{ActiveOnly: true})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando productos")
		return
	}

	var catalog []map[string]any
	for _, p := range products {
		if p.SalePrice > 0 {
			catalog = append(catalog, map[string]any{
				"id":         p.ID,
				"name":       p.Name,
				"code":       p.Code,
				"sale_price": p.SalePrice,
				"unit":       p.Unit,
				"type":       p.Type,
				"category":   "",
			})
		}
	}

	respond(w, http.StatusOK, map[string]any{
		"business_name": cfg.Name,
		"logo":          cfg.Logo,
		"currency":      cfg.Currency,
		"products":      catalog,
	})
}

