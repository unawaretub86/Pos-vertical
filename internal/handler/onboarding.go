package handler

import (
	"net/http"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
)

type onboardingRequest struct {
	FullName     string `json:"full_name"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	BusinessName string `json:"business_name"`
	NIT          string `json:"nit"`
}

// getSetupStatus GET /setup/status — public, used by the frontend to decide
// whether to route to /onboarding or /login.
func (h *Handler) getSetupStatus(w http.ResponseWriter, r *http.Request) {
	completed, err := h.setup.IsSetupCompleted()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error verificando estado de configuración")
		return
	}
	respond(w, http.StatusOK, map[string]bool{"completed": completed})
}

// submitOnboarding POST /onboarding — public, first-run wizard. Rejects with
// 409 once setup_completed is already true; never mutates state in that case.
func (h *Handler) submitOnboarding(w http.ResponseWriter, r *http.Request) {
	completed, err := h.setup.IsSetupCompleted()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error verificando estado de configuración")
		return
	}
	if completed {
		respondError(w, http.StatusConflict, "la configuración inicial ya fue completada")
		return
	}

	var req onboardingRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	if req.BusinessName == "" {
		respondError(w, http.StatusBadRequest, "el nombre del negocio es requerido")
		return
	}
	if req.NIT == "" {
		respondError(w, http.StatusBadRequest, "el NIT es requerido")
		return
	}

	user, err := h.users.Onboard(r.Context(), 1, req.FullName, req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.setup.CompleteOnboarding(req.BusinessName, req.NIT); err != nil {
		respondError(w, http.StatusInternalServerError, "error guardando configuración del negocio")
		return
	}

	token, err := appmw.GenerateToken(&model.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error generando token")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
