package handler

import (
	"net/http"
	"strings"

	appmw "github.com/andreyo/pos/internal/middleware"
)

// askAssistantRequest is decoded from POST /ai/ask. It deliberately does not
// declare a user_id field: even if a client sends one, json.Decode ignores
// unknown-to-the-struct fields, so userID always comes from the caller's own
// JWT claims below (AC-15, decision GATE 1).
type askAssistantRequest struct {
	Query string `json:"query"`
}

// askAssistant handles POST /ai/ask (admin only, see Routes()). It resolves
// the free-form question against real data via AssistantService and always
// logs the attempt, understood or not.
func (h *Handler) askAssistant(w http.ResponseWriter, r *http.Request) {
	var body askAssistantRequest
	if err := decode(r, &body); err != nil || strings.TrimSpace(body.Query) == "" {
		respondError(w, http.StatusBadRequest, "la pregunta no puede estar vacía")
		return
	}

	claims := appmw.GetClaims(r)
	result, err := h.assistant.Ask(r.Context(), claims.UserID, body.Query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error procesando la consulta")
		return
	}
	respond(w, http.StatusOK, result)
}
