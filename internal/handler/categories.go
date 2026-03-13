package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.categories.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando categorías")
		return
	}
	respond(w, http.StatusOK, cats)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	cat, err := h.categories.Create(r.Context(), body.Name)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, cat)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	cat, err := h.categories.Update(r.Context(), id, body.Name)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "categoría no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, cat)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.categories.Delete(r.Context(), id); errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "categoría no encontrada")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "error eliminando categoría")
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "categoría eliminada"})
}
