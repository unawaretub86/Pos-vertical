package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getFiscalConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.fiscalRepo.Get(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo configuración fiscal")
		return
	}
	respond(w, http.StatusOK, cfg)
}

func (h *Handler) updateFiscalConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.fiscalRepo.Get(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo configuración fiscal")
		return
	}
	if err := decode(r, cfg); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	if err := h.fiscalRepo.Update(r.Context(), cfg); err != nil {
		respondError(w, http.StatusInternalServerError, "error actualizando configuración fiscal")
		return
	}
	respond(w, http.StatusOK, cfg)
}

func (h *Handler) getSaleFactura(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if h.factelect == nil {
		respondError(w, http.StatusBadRequest, "facturación electrónica no habilitada")
		return
	}
	doc, err := h.factelect.GetDocBySaleID(r.Context(), id)
	if err != nil || doc == nil {
		respondError(w, http.StatusNotFound, "documento no encontrado")
		return
	}
	respond(w, http.StatusOK, doc)
}

func (h *Handler) getSaleFacturaPDF(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if h.factelect == nil {
		respondError(w, http.StatusNotFound, "facturación electrónica no habilitada")
		return
	}
	doc, err := h.factelect.GetDocBySaleID(r.Context(), id)
	if err != nil || doc == nil || doc.PDFURL == "" {
		respondError(w, http.StatusNotFound, "PDF no disponible aún")
		return
	}
	http.Redirect(w, r, doc.PDFURL, http.StatusTemporaryRedirect)
}
