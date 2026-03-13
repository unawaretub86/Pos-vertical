package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

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

	updated, err := h.products.Update(r.Context(), &p)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "producto no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, updated)
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
