package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	var envelope struct {
		Data any `json:"data"`
	}
	body := w.Body.Bytes()
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	raw, err := json.Marshal(envelope.Data)
	if err != nil {
		t.Fatalf("remarshal data: %v", err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("decode into target: %v", err)
	}
}

func onboardingBody(overrides map[string]string) []byte {
	body := map[string]string{
		"full_name":     "Andrea Gomez",
		"username":      "andrea",
		"password":      "s3cret",
		"business_name": "Tienda Chévere",
		"nit":           "900123456",
	}
	for k, v := range overrides {
		body[k] = v
	}
	b, _ := json.Marshal(body)
	return b
}

func TestGetSetupStatus_BeforeAndAfter(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	w := httptest.NewRecorder()
	h.getSetupStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var status struct {
		Completed bool `json:"completed"`
	}
	decodeEnvelope(t, w, &status)
	if status.Completed {
		t.Fatalf("expected completed=false on fresh DB")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewReader(onboardingBody(nil)))
	w = httptest.NewRecorder()
	h.submitOnboarding(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("onboarding status %d, body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	w = httptest.NewRecorder()
	h.getSetupStatus(w, req)
	decodeEnvelope(t, w, &status)
	if !status.Completed {
		t.Fatalf("expected completed=true after onboarding")
	}
}

func TestSubmitOnboarding_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	req := httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewReader(onboardingBody(nil)))
	w := httptest.NewRecorder()
	h.submitOnboarding(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
		User  struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	decodeEnvelope(t, w, &resp)

	if resp.Token == "" {
		t.Errorf("expected non-empty token")
	}
	if resp.User.ID != 1 {
		t.Errorf("user.id = %d, want 1", resp.User.ID)
	}
	if resp.User.Username != "andrea" {
		t.Errorf("user.username = %q, want andrea", resp.User.Username)
	}
	if resp.User.Role != "admin" {
		t.Errorf("user.role = %q, want admin", resp.User.Role)
	}

	// Password must never be echoed back.
	if bytes.Contains(w.Body.Bytes(), []byte("s3cret")) {
		t.Errorf("response body leaks plaintext password: %s", w.Body.String())
	}

	// Business config was updated.
	cfg, err := h.setup.Get()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if cfg.Name != "Tienda Chévere" || cfg.NIT != "900123456" || cfg.Currency != "COP" {
		t.Errorf("business_config not updated as expected: %+v", cfg)
	}
}

func TestSubmitOnboarding_GuardAlreadyCompleted(t *testing.T) {
	db := setupTestDB(t)
	h := New(db)

	req := httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewReader(onboardingBody(nil)))
	w := httptest.NewRecorder()
	h.submitOnboarding(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first onboarding status %d, body=%s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewReader(onboardingBody(map[string]string{
		"username": "someone-else",
	})))
	w2 := httptest.NewRecorder()
	h.submitOnboarding(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("second onboarding status = %d, want 409", w2.Code)
	}

	user, err := h.users.GetByID(req2.Context(), 1)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Username != "andrea" {
		t.Errorf("username was modified by rejected onboarding call: %q", user.Username)
	}
}

func TestSubmitOnboarding_ValidationErrors(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]string
	}{
		{"weak password", map[string]string{"password": "ab"}},
		{"empty username", map[string]string{"username": ""}},
		{"empty business name", map[string]string{"business_name": ""}},
		{"empty nit", map[string]string{"nit": ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t)
			h := New(db)

			req := httptest.NewRequest(http.MethodPost, "/api/onboarding", bytes.NewReader(onboardingBody(tc.overrides)))
			w := httptest.NewRecorder()
			h.submitOnboarding(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body=%s", w.Code, w.Body.String())
			}

			completed, err := h.setup.IsSetupCompleted()
			if err != nil {
				t.Fatalf("IsSetupCompleted: %v", err)
			}
			if completed {
				t.Errorf("setup_completed should remain false after validation error")
			}
		})
	}
}
