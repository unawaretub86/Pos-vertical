package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Contrato US-056 T-16 (AC-16): CORS permite PATCH.
func TestCORS_T16_AllowsPatch(t *testing.T) {
	h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/orders/1/items/1", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", w.Code)
	}
	allow := w.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allow, "PATCH") {
		t.Errorf("Access-Control-Allow-Methods = %q, want to contain PATCH", allow)
	}
}
