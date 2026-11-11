package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type backupStatusResponse struct {
	ConfiguredDir string  `json:"configured_dir"`
	DefaultDir    string  `json:"default_dir"`
	EffectiveDir  string  `json:"effective_dir"`
	LastAttemptAt *string `json:"last_attempt_at"`
	LastSuccessAt *string `json:"last_success_at"`
	LastOK        bool    `json:"last_ok"`
	FellBack      bool    `json:"fell_back"`
	LastError     string  `json:"last_error"`
	LastFile      string  `json:"last_file"`
}

func rfc3339Local(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.In(time.Local).Format(time.RFC3339)
	return &s
}

// getBackupStatus returns the folder setting and the report of the last backup.
func (h *Handler) getBackupStatus(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.setup.GetBackupConfig()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error al obtener la configuración de copias")
		return
	}
	respond(w, http.StatusOK, backupStatusResponse{
		ConfiguredDir: cfg.Dir,
		DefaultDir:    h.db.DefaultBackupDir(),
		EffectiveDir:  cfg.EffectiveDir,
		LastAttemptAt: rfc3339Local(cfg.LastAttemptAt),
		LastSuccessAt: rfc3339Local(cfg.LastSuccessAt),
		LastOK:        cfg.LastOK,
		FellBack:      cfg.FellBack,
		LastError:     cfg.LastError,
		LastFile:      cfg.LastFile,
	})
}

// updateBackupConfig validates and stores the backup folder. "" clears it back
// to the default; a missing or non-string "dir" never clears it.
func (h *Handler) updateBackupConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := decode(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	raw, ok := body["dir"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "Falta la carpeta")
		return
	}
	dir := strings.TrimSpace(raw)
	if dir != "" {
		if !filepath.IsAbs(dir) {
			respondError(w, http.StatusBadRequest, `Escriba la ruta completa de la carpeta, por ejemplo D:\Respaldos`)
			return
		}
		info, err := os.Stat(dir)
		if err != nil {
			respondError(w, http.StatusBadRequest, "La carpeta no existe")
			return
		}
		if !info.IsDir() {
			respondError(w, http.StatusBadRequest, "La ruta no es una carpeta")
			return
		}
		if err := h.canWrite(dir); err != nil {
			respondError(w, http.StatusBadRequest, "No se puede escribir en la carpeta")
			return
		}
	}
	if err := h.setup.SaveBackupDir(dir); err != nil {
		respondError(w, http.StatusInternalServerError, "Error al guardar la carpeta de copias")
		return
	}
	h.getBackupStatus(w, r)
}
