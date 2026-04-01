package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/uptime"
)

// SetUptimeEngine wires the uptime engine to the admin API.
func (a *API) SetUptimeEngine(engine *uptime.Engine) {
	a.uptimeEngine = engine
}

// handleUptimeChecks handles GET /api/uptime/checks (list) and POST /api/uptime/checks (create).
func (a *API) handleUptimeChecks(w http.ResponseWriter, r *http.Request) {
	if a.uptimeEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "uptime monitoring not enabled")
		return
	}

	switch r.Method {
	case http.MethodGet:
		checks, err := a.uptimeEngine.Store().ListChecks()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, checks)

	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read body")
			return
		}
		defer r.Body.Close()

		var req checkRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}

		check := req.toCheck()
		if err := a.uptimeEngine.CreateCheck(check); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Re-fetch to get the generated ID.
		created, _ := a.uptimeEngine.Store().GetCheck(check.ID)
		if created != nil {
			writeJSON(w, http.StatusCreated, created)
		} else {
			writeJSON(w, http.StatusCreated, check)
		}

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleUptimeCheckByID handles PUT /api/uptime/checks/:id and DELETE /api/uptime/checks/:id
// and GET /api/uptime/checks/:id/results.
func (a *API) handleUptimeCheckByID(w http.ResponseWriter, r *http.Request) {
	if a.uptimeEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "uptime monitoring not enabled")
		return
	}

	// Parse path: /api/uptime/checks/{id} or /api/uptime/checks/{id}/results
	path := strings.TrimPrefix(r.URL.Path, "/api/uptime/checks/")
	parts := strings.SplitN(path, "/", 2)
	checkID := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = parts[1]
	}

	if checkID == "" {
		writeError(w, http.StatusBadRequest, "check ID required")
		return
	}

	if subPath == "results" {
		a.handleUptimeResults(w, r, checkID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		check, err := a.uptimeEngine.Store().GetCheck(checkID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, check)

	case http.MethodPut:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read body")
			return
		}
		defer r.Body.Close()

		var req checkRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}

		check := req.toCheck()
		check.ID = checkID
		if err := a.uptimeEngine.UpdateCheck(check); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, check)

	case http.MethodDelete:
		if err := a.uptimeEngine.DeleteCheck(checkID); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": checkID})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleUptimeResults handles GET /api/uptime/checks/:id/results.
func (a *API) handleUptimeResults(w http.ResponseWriter, r *http.Request, checkID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	results, err := a.uptimeEngine.Store().Results(checkID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// handleUptimeStatus handles GET /api/uptime/status — returns summary of all checks.
func (a *API) handleUptimeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.uptimeEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "uptime monitoring not enabled")
		return
	}

	summaries, err := a.uptimeEngine.Summary()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summaries)
}

// checkRequest is the JSON body for creating/updating a check.
type checkRequest struct {
	ID             string            `json:"id,omitempty"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	ExpectedStatus int               `json:"expected_status"`
	IntervalSec    int               `json:"interval_sec"`
	TimeoutSec     int               `json:"timeout_sec"`
	Headers        map[string]string `json:"headers"`
	Enabled        bool              `json:"enabled"`
}

func (r *checkRequest) toCheck() uptime.Check {
	interval := time.Duration(r.IntervalSec) * time.Second
	if interval == 0 {
		interval = 60 * time.Second
	}
	timeout := time.Duration(r.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return uptime.Check{
		ID:             r.ID,
		Name:           r.Name,
		URL:            r.URL,
		Method:         r.Method,
		ExpectedStatus: r.ExpectedStatus,
		Interval:       interval,
		Timeout:        timeout,
		Headers:        r.Headers,
		Enabled:        r.Enabled,
	}
}
