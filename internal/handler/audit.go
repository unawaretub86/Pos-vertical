package handler

import (
	"net/http"
	"strconv"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// listAudit returns audit_log entries, optionally filtered by user_id,
// action and a created_at date range (from/to, RFC3339).
func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.AuditFilter{Action: q.Get("action")}

	if s := q.Get("user_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'user_id' inválido")
			return
		}
		filter.UserID = &id
	}
	if s := q.Get("from"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'from' inválido (use RFC3339)")
			return
		}
		filter.From = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := parseTime(s)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parámetro 'to' inválido (use RFC3339)")
			return
		}
		filter.To = &t
	}

	entries, err := h.audit.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando auditoría")
		return
	}
	if entries == nil {
		entries = []*model.AuditLogEntry{}
	}
	respond(w, http.StatusOK, entries)
}
