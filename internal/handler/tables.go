package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
	"github.com/go-chi/chi/v5"
)

const (
	msgTableNameTaken = "Ya existe una mesa con ese nombre (puede estar desactivada)"
	msgTableNotFound  = "mesa no encontrada"
)

type tableBody struct {
	Name      string `json:"name"`
	Zone      string `json:"zone"`
	Seats     int    `json:"seats"`
	SortOrder int    `json:"sort_order"`
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	tables, err := h.tables.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando mesas")
		return
	}
	if tables == nil {
		tables = []*model.TableWithState{}
	}
	respond(w, http.StatusOK, tables)
}

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	var body tableBody
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	t, err := h.tables.Create(r.Context(), body.Name, body.Zone, body.Seats, body.SortOrder)
	if errors.Is(err, repository.ErrTableNameTaken) {
		respondError(w, http.StatusConflict, msgTableNameTaken)
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, t)
}

func (h *Handler) updateTable(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body tableBody
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	t, err := h.tables.Update(r.Context(), id, body.Name, body.Zone, body.Seats, body.SortOrder)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		respondError(w, http.StatusNotFound, msgTableNotFound)
	case errors.Is(err, repository.ErrTableNameTaken):
		respondError(w, http.StatusConflict, msgTableNameTaken)
	case err != nil:
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respond(w, http.StatusOK, t)
	}
}

func (h *Handler) deleteTable(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	err = h.tables.Deactivate(r.Context(), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		respondError(w, http.StatusNotFound, msgTableNotFound)
	case errors.Is(err, repository.ErrTableHasOpenOrder):
		respondError(w, http.StatusConflict, "La mesa tiene un pedido abierto")
	case err != nil:
		respondError(w, http.StatusInternalServerError, "error desactivando la mesa")
	default:
		respond(w, http.StatusOK, map[string]string{"message": "mesa desactivada"})
	}
}
