//go:build export

package web

// US-091 contract §4, T-6 (AC-7) and T-7 (AC-10), against the real embedded
// export. Excluded from the default run by the export build tag. Run:
//
//	cd frontend && pnpm run build && cd .. && go run scripts/copy-frontend.go && go test -tags export ./web/

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// exportRouter mounts FileServer() like cmd/pos/main.go:63-72.
func exportRouter() http.Handler {
	r := chi.NewRouter()
	r.Mount("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "api", http.StatusTeapot)
	}))
	r.Handle("/*", FileServer())
	return r
}

func exportFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := fs.ReadFile(staticFiles, "out/"+name)
	if err != nil {
		t.Fatalf("read embedded out/%s: %v", name, err)
	}
	return b
}

func TestFileServer_T19_RealExportEveryRoute(t *testing.T) {
	if _, err := fs.Stat(staticFiles, "out/index.html"); err != nil {
		t.Fatalf("embedded export has no out/index.html (%v): run pnpm run build and go run scripts/copy-frontend.go first", err)
	}
	h := exportRouter()

	cases := []struct {
		method, target, file string
	}{
		{"GET", "/", "index.html"},
		{"GET", "/audit", "audit.html"},
		{"GET", "/barra", "barra.html"},
		{"GET", "/change-password", "change-password.html"},
		{"GET", "/customers", "customers.html"},
		{"GET", "/inventory", "inventory.html"},
		{"GET", "/login", "login.html"},
		{"GET", "/mesas", "mesas.html"},
		{"GET", "/onboarding", "onboarding.html"},
		{"GET", "/portfolio", "portfolio.html"},
		{"GET", "/pos", "pos.html"},
		{"GET", "/purchases", "purchases.html"},
		{"GET", "/register", "register.html"},
		{"GET", "/reports", "reports.html"},
		{"GET", "/settings", "settings.html"},
		{"GET", "/setup", "setup.html"},
		{"GET", "/suppliers", "suppliers.html"},
		{"GET", "/users", "users.html"},
		{"GET", "/customers/42", "customers/_.html"},
		{"GET", "/customers/42/", "customers/_.html"},
		{"GET", "/customers/", "customers.html"},
		{"GET", "/mesas/7", "mesas/_.html"},
		{"GET", "/catalogo/abc123", "catalogo/_.html"},
		{"GET", "/pos?x=1", "pos.html"},
		{"HEAD", "/customers/42", ""},
		{"GET", "/catalogo", "index.html"},
		{"GET", "/nope", "index.html"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(c.method, c.target, nil))
		res := w.Result()
		body, _ := io.ReadAll(res.Body)
		ct := res.Header.Get("Content-Type")
		if res.StatusCode != http.StatusOK || !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s %s = %d %q, want 200 text/html", c.method, c.target, res.StatusCode, ct)
			continue
		}
		if c.file == "" {
			continue
		}
		if want := exportFile(t, c.file); string(body) != string(want) {
			t.Errorf("%s %s: body (%d bytes) is not out/%s (%d bytes)", c.method, c.target, len(body), c.file, len(want))
		}
	}

	m := regexp.MustCompile(`/_next/static/chunks/[^"'\s]+\.js`).Find(exportFile(t, "index.html"))
	if m == nil {
		t.Fatal("index.html references no /_next/static/chunks/*.js")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, string(m), nil))
	if ct := w.Header().Get("Content-Type"); w.Code != http.StatusOK || !strings.Contains(ct, "javascript") {
		t.Errorf("GET %s = %d %q, want 200 javascript", m, w.Code, ct)
	}
}

func TestCopyFrontend_T20_KeepsGitkeep(t *testing.T) {
	info, err := fs.Stat(staticFiles, "out/.gitkeep")
	if err != nil {
		t.Fatalf("embedded export has no out/.gitkeep after go run scripts/copy-frontend.go: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("out/.gitkeep size = %d, want 0", info.Size())
	}
}
