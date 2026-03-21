package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/dashboard"
)

func TestHandler_StatusOK(t *testing.T) {
	h := dashboard.New(4599)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestHandler_ContentType(t *testing.T) {
	h := dashboard.New(4599)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
}

func TestHandler_ContainsExpectedElements(t *testing.T) {
	h := dashboard.New(4599)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	body := w.Body.String()

	checks := []struct {
		desc    string
		snippet string
	}{
		{"cloudmock branding", "cloudmock"},
		{"services table", `id="services-table"`},
		{"services tbody", `id="services-tbody"`},
		{"requests table", `id="requests-table"`},
		{"requests tbody", `id="requests-tbody"`},
		{"health badge element", `id="health-badge"`},
		{"health dot element", `id="health-dot"`},
		{"service filter dropdown", `id="service-filter"`},
		{"admin API health endpoint", "/api/health"},
		{"admin API services endpoint", "/api/services"},
		{"admin API stats endpoint", "/api/stats"},
		{"admin API requests endpoint", "/api/requests"},
	}

	for _, c := range checks {
		if !strings.Contains(body, c.snippet) {
			t.Errorf("missing %s: expected to find %q in HTML", c.desc, c.snippet)
		}
	}
}

func TestHandler_AdminPortEmbedded(t *testing.T) {
	h := dashboard.New(9999)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "9999") {
		t.Error("expected admin port 9999 to appear in dashboard HTML")
	}
}

func TestHandler_AllPathsServed(t *testing.T) {
	h := dashboard.New(4599)

	paths := []string{"/", "/anything", "/some/deep/path"}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("path %q: expected 200, got %d", p, w.Code)
		}
	}
}

func TestHandler_HTMLStructure(t *testing.T) {
	h := dashboard.New(4599)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body := w.Body.String()

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
