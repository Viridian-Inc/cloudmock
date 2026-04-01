package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/neureaux/cloudmock/pkg/rum"
)

// SetRUMEngine wires the RUM engine to the admin API.
func (a *API) SetRUMEngine(engine *rum.Engine) {
	a.rumEngine = engine
}

// handleRUMIngest handles POST /api/rum/events — ingests RUM events from the browser SDK.
func (a *API) handleRUMIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Try batch first (array of events), fall back to single event.
	var events []rum.RUMEvent
	if err := json.Unmarshal(body, &events); err != nil {
		// Try single event.
		var single rum.RUMEvent
		if err := json.Unmarshal(body, &single); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		events = []rum.RUMEvent{single}
	}

	stored := a.rumEngine.IngestBatch(events)
	writeJSON(w, http.StatusOK, map[string]any{
		"accepted": stored,
		"total":    len(events),
	})
}

// handleRUMVitals handles GET /api/rum/vitals — returns web vitals overview.
func (a *API) handleRUMVitals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	overview, err := a.rumEngine.Store().WebVitalsOverview()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, overview)
}

// handleRUMPages handles GET /api/rum/pages — returns per-route performance.
func (a *API) handleRUMPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	pages, err := a.rumEngine.Store().PageLoads()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, pages)
}

// handleRUMErrors handles GET /api/rum/errors — returns error groups.
func (a *API) handleRUMErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	groups, err := a.rumEngine.Store().ErrorGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, groups)
}

// handleRUMSessions handles GET /api/rum/sessions — returns session list.
func (a *API) handleRUMSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	sessions, err := a.rumEngine.Store().Sessions(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sessions)
}
