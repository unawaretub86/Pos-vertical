package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Server-side enforcement of the forced password change: a session opened with
// the default admin/admin password can only read /auth/me and change the
// password; every other protected route answers 403 until the password changes.

func doJSON(t *testing.T, h http.Handler, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func loginToken(t *testing.T, h http.Handler, user, pass string) (string, bool) {
	t.Helper()
	code, out := doJSON(t, h, http.MethodPost, "/auth/login", "", `{"username":"`+user+`","password":"`+pass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("login %s status %d, body %v", user, code, out)
	}
	data := out["data"].(map[string]any)
	return data["token"].(string), data["force_password_change"].(bool)
}

func TestForcePassword_T1_DefaultPasswordBlocksProtectedRoutes(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	token, force := loginToken(t, h, "admin", "admin")
	if !force {
		t.Fatalf("force_password_change = false for admin/admin, want true")
	}
	for _, path := range []string{"/products", "/users", "/dashboard/today"} {
		if code, _ := doJSON(t, h, http.MethodGet, path, token, ""); code != http.StatusForbidden {
			t.Errorf("GET %s with default-password token = %d, want 403", path, code)
		}
	}
	if code, _ := doJSON(t, h, http.MethodGet, "/auth/me", token, ""); code != http.StatusOK {
		t.Errorf("GET /auth/me = %d, want 200", code)
	}
}

func TestForcePassword_T2_ChangeIssuesCleanToken(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	old, _ := loginToken(t, h, "admin", "admin")
	code, out := doJSON(t, h, http.MethodPut, "/users/me/password", old, `{"current_password":"admin","new_password":"Nueva-2026"}`)
	if code != http.StatusOK {
		t.Fatalf("change password = %d, body %v", code, out)
	}
	fresh, _ := out["data"].(map[string]any)["token"].(string)
	if fresh == "" {
		t.Fatalf("response carries no new token: %v", out)
	}
	if code, _ := doJSON(t, h, http.MethodGet, "/products", fresh, ""); code != http.StatusOK {
		t.Errorf("GET /products with the new token = %d, want 200", code)
	}
	// The old token keeps the claim and stays blocked.
	if code, _ := doJSON(t, h, http.MethodGet, "/products", old, ""); code != http.StatusForbidden {
		t.Errorf("GET /products with the old token = %d, want 403", code)
	}
	if _, force := loginToken(t, h, "admin", "Nueva-2026"); force {
		t.Errorf("force_password_change = true after changing the password, want false")
	}
}

func TestForcePassword_T3_NewPasswordCannotBeAdmin(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	old, _ := loginToken(t, h, "admin", "admin")
	code, _ := doJSON(t, h, http.MethodPut, "/users/me/password", old, `{"current_password":"admin","new_password":"admin"}`)
	if code != http.StatusBadRequest {
		t.Errorf("new password \"admin\" = %d, want 400", code)
	}
}

func TestForcePassword_T4_NonDefaultPasswordIsNotBlocked(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	old, _ := loginToken(t, h, "admin", "admin")
	doJSON(t, h, http.MethodPut, "/users/me/password", old, `{"current_password":"admin","new_password":"Nueva-2026"}`)

	token, force := loginToken(t, h, "admin", "Nueva-2026")
	if force {
		t.Fatalf("force_password_change = true, want false")
	}
	if code, _ := doJSON(t, h, http.MethodGet, "/products", token, ""); code != http.StatusOK {
		t.Errorf("GET /products = %d, want 200", code)
	}
}
