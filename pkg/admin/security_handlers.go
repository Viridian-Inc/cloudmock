package admin

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/security"
)

// SetSecurityScanner sets the security scanner on the admin API and registers routes.
func (a *API) SetSecurityScanner(scanner *security.Scanner) {
	a.securityScanner = scanner

	a.mux.HandleFunc("/api/security/scan", a.handleSecurityScan)
	a.mux.HandleFunc("/api/security/checks", a.handleSecurityChecks)
	a.mux.HandleFunc("/api/security/findings", a.handleSecurityFindings)
}

// handleSecurityScan runs or retrieves a security scan.
// GET  — return last scan results (run if none cached)
// POST — trigger a new scan
func (a *API) handleSecurityScan(w http.ResponseWriter, r *http.Request) {
	if a.securityScanner == nil {
		writeError(w, http.StatusServiceUnavailable, "security scanner not available")
		return
	}

	switch r.Method {
	case http.MethodGet:
		result := a.securityScanner.LastScan()
		if result == nil {
			result = a.securityScanner.Scan()
		}
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		result := a.securityScanner.Scan()
		writeJSON(w, http.StatusOK, result)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSecurityChecks returns the list of available security checks.
func (a *API) handleSecurityChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.securityScanner == nil {
		writeError(w, http.StatusServiceUnavailable, "security scanner not available")
		return
	}
	writeJSON(w, http.StatusOK, a.securityScanner.Checks())
}

// handleSecurityFindings returns cached findings from the last scan.
func (a *API) handleSecurityFindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.securityScanner == nil {
		writeError(w, http.StatusServiceUnavailable, "security scanner not available")
		return
	}

	result := a.securityScanner.LastScan()
	if result == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"findings": []any{},
			"note":     "No scan has been run yet. POST /api/security/scan to trigger one.",
		})
		return
	}

	category := r.URL.Query().Get("category")
	if category != "" {
		filtered := security.FindingsByCategory(result.Findings, a.securityScanner.Checks(), category)
		writeJSON(w, http.StatusOK, filtered)
		return
	}

	writeJSON(w, http.StatusOK, result.Findings)
}
