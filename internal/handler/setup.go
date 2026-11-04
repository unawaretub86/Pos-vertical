package handler

import (
	"net"
	"net/http"

	"github.com/andreyo/pos/internal/mailer"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/repository"
)

// setupResponse is the API representation of the business config. The shallower
// OwnerEmailAppPassword (always nil, omitted) shadows the embedded one so the
// secret never leaves the server; HasEmailAppPassword says whether one is saved.
type setupResponse struct {
	repository.BusinessConfig
	OwnerEmailAppPassword *string `json:"owner_email_app_password,omitempty"`
	HasEmailAppPassword   bool    `json:"has_email_app_password"`
}

func newSetupResponse(cfg *repository.BusinessConfig) setupResponse {
	return setupResponse{BusinessConfig: *cfg, HasEmailAppPassword: cfg.OwnerEmailAppPassword != ""}
}

func (h *Handler) getSetup(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.setup.Get()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener configuración")
		return
	}
	resp := newSetupResponse(cfg)
	// The catalog token grants access to the public catalog: admin only (R-69).
	if claims := appmw.GetClaims(r); claims == nil || claims.Role != "admin" {
		resp.CatalogToken = ""
	}
	respond(w, http.StatusOK, resp)
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
	cfg, err := h.setup.Get()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener configuración")
		return
	}
	respond(w, http.StatusOK, newSetupResponse(cfg))
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


// getLanInfo suggests the URL other devices on the WiFi can use (admin only).
func (h *Handler) getLanInfo(w http.ResponseWriter, r *http.Request) {
	ip := ""
	if h.localIP != nil {
		ip = h.localIP()
	}
	url := ""
	if ip != "" {
		// The admin reaches this endpoint through the port the POS listens on
		// (--port / POS_PORT), so take it from the Host header.
		port := "8080"
		if _, p, err := net.SplitHostPort(r.Host); err == nil && p != "" {
			port = p
		}
		url = "http://" + ip + ":" + port
	}
	respond(w, http.StatusOK, map[string]string{"ip": ip, "url": url})
}
