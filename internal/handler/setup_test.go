package handler

// Contrato US-055 §4, T-3 y T-4 (restaurant_mode / lan_access_enabled en /setup).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// doAuth performs a request through the full middleware chain (h.Routes()).
func doAuth(h *Handler, token, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

func getSetupFlags(t *testing.T, h *Handler, token string) (restaurant, lan bool) {
	t.Helper()
	w := doAuth(h, token, http.MethodGet, "/setup", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /setup status = %d, body %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			RestaurantMode   bool `json:"restaurant_mode"`
			LanAccessEnabled bool `json:"lan_access_enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, w.Body.String())
	}
	return env.Data.RestaurantMode, env.Data.LanAccessEnabled
}

func TestSetup_T3_PersistsAndClearsFlags(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	w := doAuth(h, tok, http.MethodPut, "/setup", `{"name":"Café Uno","restaurant_mode":true,"lan_access_enabled":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %s", w.Code, w.Body.String())
	}
	if rm, lan := getSetupFlags(t, h, tok); !rm || !lan {
		t.Errorf("after PUT true/true: restaurant_mode=%v lan=%v, want true/true", rm, lan)
	}

	w = doAuth(h, tok, http.MethodPut, "/setup", `{"name":"Café Uno","restaurant_mode":false,"lan_access_enabled":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("second PUT status = %d, body %s", w.Code, w.Body.String())
	}
	if rm, lan := getSetupFlags(t, h, tok); rm || lan {
		t.Errorf("after PUT false/false: restaurant_mode=%v lan=%v, want false/false", rm, lan)
	}
}

func TestSetup_T4_AbsentLanFlagDefaultsFalse(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	tok := tokenFor(t, 1, "admin")

	w := doAuth(h, tok, http.MethodPut, "/setup", `{"name":"Café Uno","restaurant_mode":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %s", w.Code, w.Body.String())
	}
	rm, lan := getSetupFlags(t, h, tok)
	if !rm {
		t.Error("restaurant_mode = false, want true")
	}
	if lan {
		t.Error("lan_access_enabled = true, want false (absent in PUT)")
	}
}

// Contrato US-054 T-5 (AC-6, AC-7, P-7). Se espera el campo no exportado
// `localIP func() string` en Handler (inyectable desde el test); nil o "" => sin IP.
func TestSetup_US054_T5_LanInfo(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)
	admin := tokenFor(t, 1, "admin")

	type lanInfo struct {
		IP  string `json:"ip"`
		URL string `json:"url"`
	}
	fetch := func(tok string) (int, lanInfo) {
		w := doAuth(h, tok, http.MethodGet, "/setup/lan-info", "")
		var env struct {
			Data lanInfo `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		return w.Code, env.Data
	}

	h.localIP = func() string { return "192.168.1.15" }
	code, got := fetch(admin)
	if code != http.StatusOK || got.IP != "192.168.1.15" || got.URL != "http://192.168.1.15:8080" {
		t.Errorf("with IP: %d %+v, want 200 {192.168.1.15 http://192.168.1.15:8080}", code, got)
	}

	h.localIP = func() string { return "" }
	code, got = fetch(admin)
	if code != http.StatusOK || got.IP != "" || got.URL != "" {
		t.Errorf("without IP: %d %+v, want 200 {\"\" \"\"}", code, got)
	}

	h.localIP = func() string { return "192.168.1.15" }
	if code, _ := fetch(tokenFor(t, 2, "cashier")); code != http.StatusForbidden {
		t.Errorf("cashier status = %d, want 403", code)
	}
}

// US-076: the email App Password is write-only from the API.

const (
	us076Secret    = "abcd efgh ijkl mnop"
	us076NewSecret = "wxyz wxyz wxyz wxyz"
)

func putSetup(t *testing.T, h *Handler, body string) string {
	t.Helper()
	w := doAuth(h, tokenFor(t, 1, "admin"), http.MethodPut, "/setup", body)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /setup status = %d, body %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}

func getSetupRaw(t *testing.T, h *Handler, token string) string {
	t.Helper()
	w := doAuth(h, token, http.MethodGet, "/setup", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /setup status = %d, body %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}

type setupView struct {
	Data struct {
		OwnerEmail        string `json:"owner_email"`
		Address           string `json:"address"`
		HasEmailAppPasswd *bool  `json:"has_email_app_password"`
	} `json:"data"`
}

func decodeSetupView(t *testing.T, raw string) setupView {
	t.Helper()
	var v setupView
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("decode: %v\n%s", err, raw)
	}
	return v
}

func storedPassword(t *testing.T, h *Handler) string {
	t.Helper()
	cfg, err := h.setup.Get()
	if err != nil {
		t.Fatalf("setup.Get: %v", err)
	}
	return cfg.OwnerEmailAppPassword
}

func TestSetup_US076_T1_CashierGetDoesNotLeakPassword(t *testing.T) {
	h := New(setupTestDB(t))
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"`+us076Secret+`"}`)

	raw := getSetupRaw(t, h, tokenFor(t, 2, "cashier"))
	if strings.Contains(raw, "owner_email_app_password") {
		t.Errorf("GET body contains key owner_email_app_password: %s", raw)
	}
	if strings.Contains(raw, us076Secret) {
		t.Errorf("GET body leaks the secret: %s", raw)
	}
	v := decodeSetupView(t, raw)
	if v.Data.HasEmailAppPasswd == nil || !*v.Data.HasEmailAppPasswd {
		t.Errorf("has_email_app_password = %v, want true", v.Data.HasEmailAppPasswd)
	}
	if v.Data.OwnerEmail != "dueno@gmail.com" {
		t.Errorf("owner_email = %q, want dueno@gmail.com", v.Data.OwnerEmail)
	}
}

func TestSetup_US076_T2_NoPasswordStoredReportsFalse(t *testing.T) {
	h := New(setupTestDB(t))
	putSetup(t, h, `{"name":"Tienda Uno"}`)

	raw := getSetupRaw(t, h, tokenFor(t, 2, "cashier"))
	v := decodeSetupView(t, raw)
	if v.Data.HasEmailAppPasswd == nil || *v.Data.HasEmailAppPasswd {
		t.Errorf("has_email_app_password = %v, want false (present in body: %s)", v.Data.HasEmailAppPasswd, raw)
	}
}

func TestSetup_US076_T3_OmittedOrEmptyPasswordKeepsStored(t *testing.T) {
	h := New(setupTestDB(t))
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"`+us076Secret+`"}`)

	putSetup(t, h, `{"name":"Tienda Uno","address":"Cra 7 # 10-20","owner_email":"dueno@gmail.com"}`)
	if got := storedPassword(t, h); got != us076Secret {
		t.Errorf("password after omitted key = %q, want %q", got, us076Secret)
	}
	if v := decodeSetupView(t, getSetupRaw(t, h, tokenFor(t, 1, "admin"))); v.Data.Address != "Cra 7 # 10-20" {
		t.Errorf("address = %q, want Cra 7 # 10-20", v.Data.Address)
	}

	putSetup(t, h, `{"name":"Tienda Uno","address":"Cra 7 # 10-20","owner_email":"dueno@gmail.com","owner_email_app_password":""}`)
	if got := storedPassword(t, h); got != us076Secret {
		t.Errorf("password after empty string = %q, want %q", got, us076Secret)
	}
}

func TestSetup_US076_T4_NonEmptyPasswordReplacesStored(t *testing.T) {
	h := New(setupTestDB(t))
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"`+us076Secret+`"}`)
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com"}`)

	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"`+us076NewSecret+`"}`)
	if got := storedPassword(t, h); got != us076NewSecret {
		t.Errorf("password after new value = %q, want %q", got, us076NewSecret)
	}

	// The replacement must also survive a later save that omits the password.
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com"}`)
	if got := storedPassword(t, h); got != us076NewSecret {
		t.Errorf("password after replacement then omitted save = %q, want %q", got, us076NewSecret)
	}
}

func TestSetup_US076_T5_PutResponseDoesNotLeakPassword(t *testing.T) {
	h := New(setupTestDB(t))
	raw := putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"`+us076Secret+`"}`)

	if strings.Contains(raw, "owner_email_app_password") {
		t.Errorf("PUT body contains key owner_email_app_password: %s", raw)
	}
	if strings.Contains(raw, us076Secret) {
		t.Errorf("PUT body leaks the secret: %s", raw)
	}
	v := decodeSetupView(t, raw)
	if v.Data.HasEmailAppPasswd == nil || !*v.Data.HasEmailAppPasswd {
		t.Errorf("has_email_app_password = %v, want true", v.Data.HasEmailAppPasswd)
	}
}

func TestSetup_US076_T6_InternalConsumersStillSeeSecret(t *testing.T) {
	h := New(setupTestDB(t))

	// Unconfigured: email test answers 400.
	w := doAuth(h, tokenFor(t, 1, "admin"), http.MethodPost, "/settings/email/test", "")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Credenciales de correo no configuradas") {
		t.Errorf("unconfigured email test = %d %s, want 400 Credenciales de correo no configuradas", w.Code, w.Body.String())
	}

	// Saved, then re-saved with the password omitted: repo.Get still returns the real secret.
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com","owner_email_app_password":"`+us076Secret+`"}`)
	putSetup(t, h, `{"name":"Tienda Uno","owner_email":"dueno@gmail.com"}`)
	if got := storedPassword(t, h); got != us076Secret {
		t.Errorf("setup.Get password = %q, want %q", got, us076Secret)
	}
}
