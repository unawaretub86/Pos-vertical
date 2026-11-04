package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// R-62 — login attempt limit: 10 failures from one IP lock it for 5 minutes.

func newTestLimiter() (*LoginLimiter, *time.Time) {
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	l := NewLoginLimiter()
	l.now = func() time.Time { return now }
	return l, &now
}

func TestLoginLimiter_T1_LocksAfterTenFailures(t *testing.T) {
	l, _ := newTestLimiter()
	for i := 0; i < 9; i++ {
		l.Fail("10.0.0.5")
	}
	if ok, _ := l.Allowed("10.0.0.5"); !ok {
		t.Fatalf("9 failures: locked, want allowed")
	}
	l.Fail("10.0.0.5")
	ok, retry := l.Allowed("10.0.0.5")
	if ok {
		t.Fatalf("10 failures: allowed, want locked")
	}
	if retry != 5*time.Minute {
		t.Errorf("retry after = %v, want 5m", retry)
	}
}

func TestLoginLimiter_T2_UnlocksAfterFiveMinutes(t *testing.T) {
	l, now := newTestLimiter()
	for i := 0; i < 10; i++ {
		l.Fail("10.0.0.5")
	}
	*now = now.Add(5*time.Minute + time.Second)
	if ok, _ := l.Allowed("10.0.0.5"); !ok {
		t.Errorf("after 5m: locked, want allowed")
	}
}

func TestLoginLimiter_T3_IPsAreIndependent(t *testing.T) {
	l, _ := newTestLimiter()
	for i := 0; i < 10; i++ {
		l.Fail("10.0.0.5")
	}
	if ok, _ := l.Allowed("10.0.0.6"); !ok {
		t.Errorf("other IP locked, want allowed")
	}
}

func TestLoginLimiter_T4_SuccessResetsCounter(t *testing.T) {
	l, _ := newTestLimiter()
	for i := 0; i < 9; i++ {
		l.Fail("10.0.0.5")
	}
	l.Reset("10.0.0.5")
	l.Fail("10.0.0.5")
	if ok, _ := l.Allowed("10.0.0.5"); !ok {
		t.Errorf("1 failure after reset: locked, want allowed")
	}
}

func TestLoginLimiter_T5_OldFailuresExpire(t *testing.T) {
	l, now := newTestLimiter()
	for i := 0; i < 9; i++ {
		l.Fail("10.0.0.5")
	}
	*now = now.Add(11 * time.Minute) // outside the 10-minute window
	l.Fail("10.0.0.5")
	if ok, _ := l.Allowed("10.0.0.5"); !ok {
		t.Errorf("stale failures counted: locked, want allowed")
	}
}

func TestClientIP_T6_UsesRemoteAddrNotForwardedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "192.168.1.20:51234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := ClientIP(req); got != "192.168.1.20" {
		t.Errorf("ClientIP = %q, want 192.168.1.20", got)
	}
}

// CORS allowlist: only the dev server origins are echoed back.

func corsRequest(origin string) *httptest.ResponseRecorder {
	h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestCORS_T7_AllowsDevOrigins(t *testing.T) {
	for _, origin := range []string{"http://localhost:3000", "http://127.0.0.1:3000"} {
		if got := corsRequest(origin).Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %s: Allow-Origin = %q, want %q", origin, got, origin)
		}
	}
}

func TestCORS_T8_RejectsForeignOrigin(t *testing.T) {
	w := corsRequest("http://evil.example")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("foreign origin: Allow-Origin = %q, want empty", got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("same-origin style request must still be served, status %d", w.Code)
	}
}

func TestCORS_T9_NoOriginSendsNoHeader(t *testing.T) {
	if got := corsRequest("").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("no Origin: Allow-Origin = %q, want empty", got)
	}
}
