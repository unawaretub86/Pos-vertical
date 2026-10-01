package handler

import (
	"net/http"
)

func (h *Handler) kdsGetActiveOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	res, err := h.kds.GetActiveOrders(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, res)
}
