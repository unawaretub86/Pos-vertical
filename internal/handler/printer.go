package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
	"github.com/go-chi/chi/v5"
)

// transportFor builds the transport of the saved config. ok is false when the
// method is "browser" (nothing to send to a printer).
func transportFor(cfg *repository.BusinessConfig) (service.Transport, bool) {
	switch cfg.PrintMethod {
	case service.PrintMethodNetwork:
		return service.NetworkTransport{Host: cfg.PrinterHost, Port: cfg.PrinterPort}, true
	case service.PrintMethodWindows:
		return service.WindowsTransport{Name: cfg.PrinterName}, true
	}
	return nil, false
}

// sendToPrinter prints data and writes the HTTP response for every outcome.
func (h *Handler) sendToPrinter(ctx context.Context, w http.ResponseWriter, tr service.Transport, data []byte) {
	err := h.printer.Print(ctx, tr, data)
	switch {
	case errors.Is(err, service.ErrUnsupported):
		respondError(w, http.StatusNotImplemented, "La impresión por cola de Windows solo está disponible en Windows")
	case err != nil:
		respondError(w, http.StatusBadGateway, "No se pudo imprimir: "+err.Error())
	default:
		respond(w, http.StatusOK, map[string]bool{"printed": true})
	}
}

func (h *Handler) printTicket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "sale_id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	cfg, err := h.setup.Get()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener configuración")
		return
	}
	tr, ok := transportFor(cfg)
	if !ok {
		respondError(w, http.StatusConflict, "La impresión directa no está activada en la configuración")
		return
	}
	ticket, err := h.buildTicket(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "venta no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo venta")
		return
	}
	h.sendToPrinter(r.Context(), w, tr, service.BuildTicket(*ticket, cfg.PaperWidthMM))
}

func (h *Handler) printTest(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.setup.Get()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener configuración")
		return
	}
	tr, ok := transportFor(cfg)
	if !ok {
		respondError(w, http.StatusConflict, "La impresión directa no está activada en la configuración")
		return
	}
	data := service.BuildTestTicket(cfg.Name, cfg.PrintMethod, cfg.PaperWidthMM, time.Now())
	h.sendToPrinter(r.Context(), w, tr, data)
}
