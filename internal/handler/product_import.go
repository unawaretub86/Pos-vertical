package handler

import (
	"encoding/base64"
	"errors"
	"net/http"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
)

// maxImportBody bounds the JSON body of POST /products/import: a 2 MiB file
// grows by 4/3 in base64, plus the other fields.
const maxImportBody = 3 << 20

const msgImportTooLarge = "el archivo supera 2 MB"

// importProducts validates (dry_run) or imports a product catalog CSV (US-094).
func (h *Handler) importProducts(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBody)
	var body struct {
		FileName      string            `json:"file_name"`
		ContentBase64 string            `json:"content_base64"`
		Mapping       map[string]string `json:"mapping"`
		DryRun        bool              `json:"dry_run"`
	}
	if err := decode(r, &body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, msgImportTooLarge)
			return
		}
		respondError(w, http.StatusBadRequest, "archivo inválido")
		return
	}
	content, err := base64.StdEncoding.DecodeString(body.ContentBase64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "archivo inválido")
		return
	}
	req := service.ProductImportRequest{FileName: body.FileName, Content: content, Mapping: body.Mapping}

	if body.DryRun {
		report, err := h.prodImport.DryRun(r.Context(), req)
		if err != nil {
			respondImportError(w, err)
			return
		}
		respond(w, http.StatusOK, report)
		return
	}
	res, err := h.prodImport.Commit(r.Context(), appmw.GetClaims(r).UserID, req)
	if err != nil {
		respondImportError(w, err)
		return
	}
	respond(w, http.StatusCreated, res)
}

func respondImportError(w http.ResponseWriter, err error) {
	var (
		ve           *service.ValidationError
		rejected     *service.ImportRejectedError
		barcodeRace  *repository.ErrCatalogBarcodeTaken
		categoryRace *repository.ErrCatalogCategoryTaken
	)
	switch {
	case errors.Is(err, service.ErrImportTooLarge):
		respondError(w, http.StatusRequestEntityTooLarge, msgImportTooLarge)
	case errors.As(err, &ve):
		respondError(w, http.StatusBadRequest, ve.Msg)
	case errors.As(err, &rejected):
		respondError(w, http.StatusUnprocessableEntity, rejected.Error())
	case errors.As(err, &barcodeRace):
		respondError(w, http.StatusConflict, barcodeRace.Error())
	case errors.As(err, &categoryRace):
		respondError(w, http.StatusConflict, categoryRace.Error())
	case errors.Is(err, repository.ErrBusy):
		respondError(w, http.StatusServiceUnavailable, msgBusy)
	default:
		respondError(w, http.StatusInternalServerError, "error importando productos")
	}
}
