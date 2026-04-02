// Package dashboard provides a single-page web dashboard for cloudmock,
// served on the dashboard port and talking to the admin API.
package dashboard

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed all:dist
var distFS embed.FS

// Handler serves the cloudmock web dashboard as a Vite-built SPA.
// When an admin API handler is attached via SetAdminHandler, requests to
// /api/* are delegated to it, making the dashboard port a single-origin
// server for both the UI and the admin API (eliminates CORS).
type Handler struct {
	fileServer   http.Handler
	adminPort    int
	adminHandler http.Handler
	bootID       string
}

// New creates a dashboard Handler. The adminPort parameter is kept for API
// compatibility but is no longer injected into the HTML (the SPA discovers
// the admin port at runtime).
func New(adminPort int) *Handler {
	dist, _ := fs.Sub(distFS, "dist")
	return &Handler{
		fileServer: http.FileServer(http.FS(dist)),
		adminPort:  adminPort,
		bootID:     fmt.Sprintf("%d", time.Now().UnixMilli()),
	}
}

// SetAdminHandler attaches an admin API handler so that /api/* requests
// are served on the same port as the SPA (single-origin, no CORS needed).
func (h *Handler) SetAdminHandler(handler http.Handler) {
	h.adminHandler = handler
}

// ServeHTTP implements http.Handler. It serves static assets from the embedded
// dist directory. For any path that does not match a static file, it serves
// index.html (SPA fallback for client-side routing). Requests to /api/* are
// forwarded to the admin API handler when one is attached.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Delegate /api/* to the admin API handler if attached.
	if h.adminHandler != nil && strings.HasPrefix(r.URL.Path, "/api/") {
		h.adminHandler.ServeHTTP(w, r)
		return
	}

	// Force cache-bust: redirect bare / to /?v=<bootID> so the browser
	// can't serve a stale HTML page from disk cache.
	if r.URL.Path == "/" && r.URL.Query().Get("v") != h.bootID {
		http.Redirect(w, r, "/?v="+h.bootID, http.StatusTemporaryRedirect)
		return
	}

	// Try to open the requested file in the embedded FS.
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	f, err := distFS.Open("dist/" + path)
	if err != nil {
		// File not found — serve index.html for SPA client-side routing.
		h.serveIndex(w, r)
		return
	}
	f.Close()

	// HTML files: no caching (ensures fresh JS/CSS references after rebuilds).
	// Hashed assets (JS/CSS): immutable, cache forever.
	if strings.HasSuffix(path, ".html") || path == "index.html" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	// File exists — serve it with the embedded file server.
	h.fileServer.ServeHTTP(w, r)
}

func (h *Handler) serveIndex(w http.ResponseWriter, _ *http.Request) {
	f, err := distFS.Open("dist/index.html")
	if err != nil {
		http.Error(w, "dashboard not built", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, f)
}
