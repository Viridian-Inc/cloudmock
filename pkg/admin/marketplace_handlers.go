package admin

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/marketplace"
)

// SetMarketplace sets the marketplace registry on the admin API and registers routes.
func (a *API) SetMarketplace(registry *marketplace.Registry) {
	a.marketplace = registry

	a.mux.HandleFunc("/api/marketplace", a.handleMarketplace)
	a.mux.HandleFunc("/api/marketplace/", a.handleMarketplaceByID)
}

// handleMarketplace handles GET /api/marketplace — search/list plugins.
func (a *API) handleMarketplace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.marketplace == nil {
		writeError(w, http.StatusServiceUnavailable, "marketplace not available")
		return
	}

	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")

	if query != "" || category != "" {
		results := a.marketplace.Search(query, category)
		writeJSON(w, http.StatusOK, results)
		return
	}

	writeJSON(w, http.StatusOK, a.marketplace.List())
}

// handleMarketplaceByID handles operations on a specific marketplace listing.
// GET    /api/marketplace/:id          — plugin detail
// POST   /api/marketplace/:id/install  — install (placeholder)
// DELETE /api/marketplace/:id          — uninstall (placeholder)
func (a *API) handleMarketplaceByID(w http.ResponseWriter, r *http.Request) {
	if a.marketplace == nil {
		writeError(w, http.StatusServiceUnavailable, "marketplace not available")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/marketplace/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusBadRequest, "plugin id is required")
		return
	}

	subpath := ""
	if len(parts) > 1 {
		subpath = parts[1]
	}

	switch {
	case subpath == "install" && r.Method == http.MethodPost:
		if err := a.marketplace.Install(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "installed", "id": id})

	case subpath == "" && r.Method == http.MethodGet:
		listing, ok := a.marketplace.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "plugin not found")
			return
		}
		writeJSON(w, http.StatusOK, listing)

	case subpath == "" && r.Method == http.MethodDelete:
		if err := a.marketplace.Uninstall(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "uninstalled", "id": id})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
