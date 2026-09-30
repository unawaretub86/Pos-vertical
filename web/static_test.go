package web

// Contrato US-054 T-6..T-10. Firma esperada: newFileServer(fs.FS) http.Handler,
// recibiendo la raiz ya "sub-eada" (index.html en la raiz del FS).

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":               {Data: []byte("INDEX")},
		"login.html":               {Data: []byte("LOGIN")},
		"customers.html":           {Data: []byte("CUST")},
		"customers/_.html":         {Data: []byte("CUSTID")},
		"catalogo/_.html":          {Data: []byte("CAT")},
		"_next/static/chunks/a.js": {Data: []byte("js")},
	}
}

func get(t *testing.T, path string) (int, string, string) {
	t.Helper()
	h := newFileServer(testFS())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	res := w.Result()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b), res.Header.Get("Content-Type")
}

func TestFileServer_T6_RootServesIndex(t *testing.T) {
	code, body, _ := get(t, "/")
	if code != 200 || body != "INDEX" {
		t.Errorf("GET / = %d %q, want 200 INDEX", code, body)
	}
}

func TestFileServer_T7_HtmlExtensionWithoutRedirect(t *testing.T) {
	code, body, ct := get(t, "/login")
	if code != 200 || body != "LOGIN" || !strings.HasPrefix(ct, "text/html") {
		t.Errorf("GET /login = %d %q %q, want 200 LOGIN text/html", code, body, ct)
	}
	code, body, _ = get(t, "/customers")
	if code != 200 || body != "CUST" {
		t.Errorf("GET /customers = %d %q, want 200 CUST (no 301)", code, body)
	}
}

func TestFileServer_T8_AssetContentType(t *testing.T) {
	code, body, ct := get(t, "/_next/static/chunks/a.js")
	if code != 200 || body != "js" {
		t.Errorf("GET a.js = %d %q, want 200 js", code, body)
	}
	if !strings.HasPrefix(ct, "text/javascript") && !strings.HasPrefix(ct, "application/javascript") {
		t.Errorf("Content-Type = %q, want javascript", ct)
	}
}

func TestFileServer_T9_DynamicRoutes(t *testing.T) {
	if code, body, _ := get(t, "/customers/5"); code != 200 || body != "CUSTID" {
		t.Errorf("GET /customers/5 = %d %q, want 200 CUSTID", code, body)
	}
	if code, body, _ := get(t, "/catalogo/abc123"); code != 200 || body != "CAT" {
		t.Errorf("GET /catalogo/abc123 = %d %q, want 200 CAT", code, body)
	}
	if code, body, _ := get(t, "/customers"); code != 200 || body != "CUST" {
		t.Errorf("GET /customers = %d %q, want 200 CUST", code, body)
	}
}

func TestFileServer_T10_FallbackAndAsset404(t *testing.T) {
	if code, body, _ := get(t, "/nada"); code != 200 || body != "INDEX" {
		t.Errorf("GET /nada = %d %q, want 200 INDEX", code, body)
	}
	if code, _, _ := get(t, "/_next/static/chunks/no-existe.js"); code != 404 {
		t.Errorf("GET missing chunk = %d, want 404", code)
	}
}
