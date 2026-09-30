package handler

import (
	"net/http"

	"github.com/andreyo/pos/internal/mailer"
	"github.com/andreyo/pos/internal/repository"
)

func (h *Handler) getSetup(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.setup.Get()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener configuración")
		return
	}
	respond(w, http.StatusOK, cfg)
}

func (h *Handler) updateSetup(w http.ResponseWriter, r *http.Request) {
	var body repository.BusinessConfig
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if body.Name == "" {
		respondError(w, http.StatusBadRequest, "El nombre del negocio es requerido")
		return
	}
	if body.Currency == "" {
		body.Currency = "COP"
	}
	if err := h.setup.Upsert(&body); err != nil {
		respondError(w, http.StatusInternalServerError, "Error al guardar configuración")
		return
	}
	cfg, _ := h.setup.Get()
	respond(w, http.StatusOK, cfg)
}

func (h *Handler) testEmailConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.setup.Get()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener configuración")
		return
	}
	if cfg.OwnerEmail == "" || cfg.OwnerEmailAppPassword == "" {
		respondError(w, http.StatusBadRequest, "Credenciales de correo no configuradas")
		return
	}

	smtpCfg := mailer.SMTPConfig{
		Email:       cfg.OwnerEmail,
		AppPassword: cfg.OwnerEmailAppPassword,
	}

	err = mailer.Send(smtpCfg, "Prueba de conexión POS", "Este es un mensaje de prueba para confirmar que el correo está configurado correctamente.")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al enviar correo: "+err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]string{"message": "Correo enviado con éxito"})
}

