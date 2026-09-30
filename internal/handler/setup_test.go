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
