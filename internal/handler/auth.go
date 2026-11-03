package handler

import (
	"net/http"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username y contraseña requeridos")
		return
	}

	var user model.User
	err := h.db.QueryRow(
		`SELECT id, username, password_hash, role, active FROM users WHERE username = ? AND active = 1`,
		req.Username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.Active)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	forcePasswordChange := req.Password == "admin"

	token, err := appmw.GenerateToken(&model.Claims{
		UserID:              user.ID,
		Username:            user.Username,
		Role:                user.Role,
		ForcePasswordChange: forcePasswordChange,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "error generando token")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"token":                 token,
		"force_password_change": forcePasswordChange,
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	claims := appmw.GetClaims(r)
	respond(w, http.StatusOK, map[string]any{
		"id":       claims.UserID,
		"username": claims.Username,
		"role":     claims.Role,
	})
}
