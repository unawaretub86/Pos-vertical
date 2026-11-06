package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/andreyo/pos/internal/mailer"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) openRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OpeningAmount float64 `json:"opening_amount"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	claims := appmw.GetClaims(r)
	reg, err := h.registers.Open(r.Context(), claims.UserID, body.OpeningAmount)
	if errors.Is(err, service.ErrRegisterAlreadyOpen) {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error abriendo caja")
		return
	}
	respond(w, http.StatusCreated, reg)
}

func (h *Handler) listRegisters(w http.ResponseWriter, r *http.Request) {
	list, err := h.registers.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando cajas")
		return
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) getRegister(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	summary, err := h.registers.Summary(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, "caja no encontrada")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo caja")
		return
	}
	respond(w, http.StatusOK, summary)
}

func (h *Handler) getOpenRegister(w http.ResponseWriter, r *http.Request) {
	reg, err := h.registers.GetOpen(r.Context())
	if errors.Is(err, service.ErrRegisterNotOpen) || reg == nil {
		respond(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		respond(w, http.StatusOK, nil)
		return
	}
	summary, err := h.registers.Summary(r.Context(), reg.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error obteniendo caja")
		return
	}
	respond(w, http.StatusOK, summary)
}

func (h *Handler) closeRegister(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		ActualClose float64 `json:"actual_close"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	summary, err := h.registers.Close(r.Context(), id, body.ActualClose)
	if errors.Is(err, service.ErrRegisterClosed) {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error cerrando caja")
		return
	}

	cashier := "—"
	if claims := appmw.GetClaims(r); claims != nil {
		cashier = claims.Username
	}
	go h.sendRegisterSummaryEmail(summary, cashier)

	respond(w, http.StatusOK, summary)
}

// sendRegisterSummaryEmail auto-sends the close summary to the owner's inbox
// (US-053). Fire-and-forget: the register is already closed and persisted by
// the time this runs, so a missing/misconfigured mailbox or offline network
// must never affect the close response — only logged server-side.
func (h *Handler) sendRegisterSummaryEmail(summary *service.RegisterSummary, cashier string) {
	cfg, err := h.setup.Get()
	if err != nil || cfg.OwnerEmail == "" || cfg.OwnerEmailAppPassword == "" {
		return
	}

	closedAt := time.Now()
	if summary.Register.ClosedAt != nil {
		closedAt = *summary.Register.ClosedAt
	}
	actualClose := 0.0
	if summary.Register.ActualClose != nil {
		actualClose = *summary.Register.ActualClose
	}
	difference := 0.0
	if summary.Register.Difference != nil {
		difference = *summary.Register.Difference
	}
	lines := make([]mailer.PaymentLine, len(summary.ByPaymentMethod))
	for i, m := range summary.ByPaymentMethod {
		lines[i] = mailer.PaymentLine{Method: m.Method, Total: m.Total}
	}

	subject, body := mailer.BuildRegisterSummaryEmail(mailer.RegisterSummaryData{
		BusinessName:    cfg.Name,
		Cashier:         cashier,
		ClosedAt:        closedAt,
		ByPaymentMethod: lines,
		CashAbonos:      summary.CashAbonos,
		CashIn:          summary.CashIn,
		CashOut:         summary.CashOut,
		Expected:        summary.Expected,
		ActualClose:     actualClose,
		Difference:      difference,
	})

	smtpCfg := mailer.SMTPConfig{Email: cfg.OwnerEmail, AppPassword: cfg.OwnerEmailAppPassword}
	if err := mailer.Send(smtpCfg, subject, body); err != nil {
		log.Printf("register summary email: %v", err)
	}
}

func (h *Handler) listMovements(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	movements, err := h.registers.ListMovements(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error listando movimientos")
		return
	}
	respond(w, http.StatusOK, movements)
}

func (h *Handler) addMovement(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Type        string  `json:"type"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "request inválido")
		return
	}
	if body.Type != "cash_in" && body.Type != "cash_out" {
		respondError(w, http.StatusBadRequest, "tipo debe ser cash_in o cash_out")
		return
	}
	mv, err := h.registers.AddMovement(r.Context(), id, body.Type, body.Amount, body.Description)
	if errors.Is(err, service.ErrRegisterClosed) {
		respondError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error agregando movimiento")
		return
	}

	if body.Type == "cash_out" {
		if claims := appmw.GetClaims(r); claims != nil {
			detailsBytes, _ := json.Marshal(map[string]any{"amount": body.Amount, "description": body.Description})
			_ = repository.InsertAuditLog(r.Context(), h.db, claims.UserID, "cash_out", "register_movement", mv.ID, string(detailsBytes))
		}
	}

	respond(w, http.StatusCreated, mv)
}
