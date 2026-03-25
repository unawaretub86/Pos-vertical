package handler

import (
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/model"
	"github.com/go-chi/chi/v5"
)

// listUsers GET /users — admin only
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando usuarios")
		return
	}
	if users == nil {
		users = []*model.User{}
	}
	respond(w, http.StatusOK, users)
}

// createUser POST /users — admin only
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	user, err := h.users.Create(r.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, user)
}

// updateUser PUT /users/:id — admin only
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Active   *bool  `json:"active"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	user, err := h.users.Update(r.Context(), id, req.Username, req.Role, active, req.Password)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, user)
}

// deactivateUser DELETE /users/:id — admin only
func (h *Handler) deactivateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.users.Deactivate(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"message": "usuario desactivado"})
}
