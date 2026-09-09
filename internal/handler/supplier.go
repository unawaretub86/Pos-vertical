package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) listSuppliers(w http.ResponseWriter, r *http.Request) {
	suppliers, err := h.suppliers.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando proveedores")
		return
	}
	if suppliers == nil {
		suppliers = []*model.Supplier{}
	}
	respond(w, http.StatusOK, suppliers)
}

func (h *Handler) createSupplier(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Notes string `json:"notes"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	supplier, err := h.suppliers.Create(r.Context(), body.Name, body.Phone, body.Notes)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, supplier)
}

func (h *Handler) updateSupplier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Notes string `json:"notes"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	supplier, err := h.suppliers.Update(r.Context(), id, body.Name, body.Phone, body.Notes)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "proveedor no encontrado")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, supplier)
}

func (h *Handler) deleteSupplier(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.suppliers.Delete(r.Context(), id); errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "proveedor no encontrado")
		return
	} else if err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "proveedor eliminado"})
}
