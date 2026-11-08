package web

// US-091 contract §4, T-5 (AC-7, AC-8): every page.tsx under frontend/app must
// resolve to its own exported file through newFileServer. Runs in CI's Go job,
// which has frontend/app in the checkout but no real export, so the export
// layout is rebuilt in an fstest.MapFS from the page.tsx files found.

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// appRoute is one page.tsx: its route pattern ("/", "/pos", "/customers/[id]")
// and, for a dynamic route, the prefix before the first [segment].
type appRoute struct {
	pattern string
	file    string // page.tsx path on disk
	dynamic bool
	prefix  string
}

func findAppRoutes(t *testing.T) []appRoute {
	t.Helper()
	root := filepath.Join("..", "frontend", "app")
	var routes []appRoute
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "page.tsx" {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(p))
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		r := appRoute{pattern: "/", file: p}
		if rel != "." {
			r.pattern = "/" + rel
			segs := strings.Split(rel, "/")
			for i, s := range segs {
				if strings.HasPrefix(s, "[") {
					r.dynamic = true
					r.prefix = strings.Join(segs[:i], "/")
					break
				}
			}
		}
		routes = append(routes, r)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].pattern < routes[j].pattern })
	return routes
}

// exportName is the file the static export writes for a static route.
func exportName(pattern string) string {
	if pattern == "/" {
		return "index.html"
	}
	return strings.TrimPrefix(pattern, "/") + ".html"
}

func serveRoute(h http.Handler, target string) (int, string, string) {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	res := w.Result()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b), res.Header.Get("Content-Type")
}

func TestFileServer_T18_EveryAppRouteResolves(t *testing.T) {
	routes := findAppRoutes(t)

	// The 21 page routes of contract §2.
	want := []string{
		"/", "/audit", "/barra", "/catalogo/[token]", "/change-password", "/customers",
		"/customers/[id]", "/inventory", "/login", "/mesas", "/mesas/[orderId]", "/onboarding",
		"/portfolio", "/pos", "/purchases", "/register", "/reports", "/settings", "/setup",
		"/suppliers", "/users",
	}
	found := make([]string, 0, len(routes))
	for _, r := range routes {
		found = append(found, r.pattern)
	}
	for _, w := range want {
		if !slices.Contains(found, w) {
			t.Errorf("route %s: no page.tsx found under frontend/app (found %v)", w, found)
		}
	}
	if len(routes) < len(want) {
		t.Errorf("found %d page.tsx routes, want at least %d", len(routes), len(want))
	}

	// Export layout with distinct content per file.
	mfs := fstest.MapFS{}
	for _, r := range routes {
		if r.dynamic {
			mfs[r.prefix+"/_.html"] = &fstest.MapFile{Data: []byte("PAGE " + r.prefix + "/_")}
		} else {
			name := exportName(r.pattern)
			mfs[name] = &fstest.MapFile{Data: []byte("PAGE " + name)}
		}
	}
	h := newFileServer(mfs)

	for _, r := range routes {
		if r.dynamic {
			if !slices.Contains(dynamicRoutes, r.prefix) {
				t.Errorf("route %s: prefix %q is not in dynamicRoutes %v (web/static.go)", r.pattern, r.prefix, dynamicRoutes)
			}
			src, err := os.ReadFile(r.file)
			if err != nil {
				t.Fatalf("route %s: read %s: %v", r.pattern, r.file, err)
			}
			if !strings.Contains(string(src), "generateStaticParams") || !strings.Contains(string(src), "'_'") {
				t.Errorf("route %s: %s does not bake '_' through generateStaticParams", r.pattern, r.file)
			}
			wantBody := "PAGE " + r.prefix + "/_"
			for _, target := range []string{"/" + r.prefix + "/42", "/" + r.prefix + "/42/"} {
				code, body, ct := serveRoute(h, target)
				if code != http.StatusOK || body != wantBody || !strings.HasPrefix(ct, "text/html") {
					t.Errorf("route %s: GET %s = %d %q %q, want 200 %q text/html", r.pattern, target, code, body, ct, wantBody)
				}
			}
			continue
		}

		name := exportName(r.pattern)
		wantBody := "PAGE " + name
		targets := []string{r.pattern, r.pattern + "?x=1"}
		if r.pattern != "/" {
			targets = append(targets, r.pattern+"/")
		}
		for _, target := range targets {
			code, body, ct := serveRoute(h, target)
			if code != http.StatusOK || body != wantBody || !strings.HasPrefix(ct, "text/html") {
				t.Errorf("route %s: GET %s = %d %q %q, want 200 %q text/html", r.pattern, target, code, body, ct, wantBody)
			}
		}
	}
}
