package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// R-71: the LAN URL shown in Settings follows the port the POS is served on.
func TestLanInfo_R71_UsesRequestPort(t *testing.T) {
	h := New(setupTestDB(t))
	h.localIP = func() string { return "192.168.1.15" }

	req := httptest.NewRequest(http.MethodGet, "/setup/lan-info", nil)
	req.Host = "localhost:9090"
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, 1, "admin"))
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)

	var env struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if w.Code != http.StatusOK || env.Data.URL != "http://192.168.1.15:9090" {
		t.Errorf("status %d url %q, want 200 http://192.168.1.15:9090", w.Code, env.Data.URL)
	}
}
