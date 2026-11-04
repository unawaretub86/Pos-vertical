package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func rawLogin(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// R-62 — the login endpoint locks an IP after 10 failures.

func TestLoginRateLimit_T1_LocksAfterTenFailures(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	bad := `{"username":"admin","password":"wrong"}`
	for i := 0; i < 10; i++ {
		if code, _ := doJSON(t, h, http.MethodPost, "/auth/login", "", bad); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401", i+1, code)
		}
	}
	// Even the right password is refused while locked.
	code, _ := doJSON(t, h, http.MethodPost, "/auth/login", "", `{"username":"admin","password":"admin"}`)
	if code != http.StatusTooManyRequests {
		t.Errorf("attempt 11 with the right password = %d, want 429", code)
	}
}

func TestLoginRateLimit_T2_SuccessResetsCounter(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	bad := `{"username":"admin","password":"wrong"}`
	for i := 0; i < 9; i++ {
		doJSON(t, h, http.MethodPost, "/auth/login", "", bad)
	}
	if code, _ := doJSON(t, h, http.MethodPost, "/auth/login", "", `{"username":"admin","password":"admin"}`); code != http.StatusOK {
		t.Fatalf("right password after 9 failures = %d, want 200", code)
	}
	for i := 0; i < 9; i++ {
		if code, _ := doJSON(t, h, http.MethodPost, "/auth/login", "", bad); code != http.StatusUnauthorized {
			t.Fatalf("failure %d after reset = %d, want 401", i+1, code)
		}
	}
}

func TestLoginRateLimit_T3_RetryAfterHeader(t *testing.T) {
	t.Setenv("POS_JWT_SECRET", "test-secret")
	h := New(setupTestDB(t)).Routes()

	bad := `{"username":"admin","password":"wrong"}`
	for i := 0; i < 10; i++ {
		doJSON(t, h, http.MethodPost, "/auth/login", "", bad)
	}
	w := rawLogin(h, bad)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if secs, err := strconv.Atoi(w.Header().Get("Retry-After")); err != nil || secs <= 0 || secs > 300 {
		t.Errorf("Retry-After = %q, want 1..300 seconds", w.Header().Get("Retry-After"))
	}
}
