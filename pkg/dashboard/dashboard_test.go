package dashboard_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/dashboard"
)

// getHTML fetches the dashboard HTML via an httptest.Server so redirects
// (the cache-busting /?v=<bootID> redirect) are followed automatically.
func getHTML(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestHandler_StatusOK(t *testing.T) {
	h := dashboard.New(4599)
	code, _ := getHTML(t, h, "/")
	if code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", code)
	}
}

func TestHandler_ContentType(t *testing.T) {
	h := dashboard.New(4599)
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
}

func TestHandler_ContainsExpectedElements(t *testing.T) {
	h := dashboard.New(4599)
	_, body := getHTML(t, h, "/")

	checks := []struct {
		desc    string
		snippet string
	}{
		{"HTML doctype", "<!DOCTYPE html>"},
		{"app mount point", `id="app"`},
		{"module script", `type="module"`},
	}

	for _, c := range checks {
		if !strings.Contains(body, c.snippet) {
			t.Errorf("missing %s: expected to find %q in HTML", c.desc, c.snippet)
		}
	}
}

func TestHandler_AllPathsServed(t *testing.T) {
	h := dashboard.New(4599)

	paths := []string{"/", "/anything", "/some/deep/path"}
	for _, p := range paths {
		code, _ := getHTML(t, h, p)
		if code != http.StatusOK {
			t.Errorf("path %q: expected 200, got %d", p, code)
		}
	}
}

func TestHandler_HTMLStructure(t *testing.T) {
	h := dashboard.New(4599)
	_, body := getHTML(t, h, "/")

	if !strings.HasPrefix(strings.TrimSpace(body), "<!DOCTYPE html>") {
		t.Error("expected HTML to start with <!DOCTYPE html>")
	}
	if !strings.Contains(body, "<html") {
		t.Error("expected <html> element")
	}
	if !strings.Contains(body, "</html>") {
		t.Error("expected closing </html> tag")
	}
}

func TestHandler_StaticAssets(t *testing.T) {
	h := dashboard.New(4599)
	_, body := getHTML(t, h, "/")

	if !strings.Contains(body, "assets/") {
		t.Error("expected asset references in built HTML")
	}
}

func TestHandler_CacheBustRedirect(t *testing.T) {
	h := dashboard.New(4599)
	// Direct request to / without following redirects should get 307
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307 redirect, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/?v=") {
		t.Fatalf("expected redirect to /?v=<bootID>, got %q", loc)
	}
}
