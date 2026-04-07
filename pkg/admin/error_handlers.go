package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	errs "github.com/Viridian-Inc/cloudmock/pkg/errors"
)

// SetErrorStore wires the error store to the admin API and registers routes.
func (a *API) SetErrorStore(store errs.ErrorStore) {
	a.errorStore = store
}

// handleErrors handles GET /api/errors — list error groups.
func (a *API) handleErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.errorStore == nil {
		writeError(w, http.StatusServiceUnavailable, "error tracking not enabled")
		return
	}

	status := r.URL.Query().Get("status")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	groups, err := a.errorStore.GetGroups(status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, groups)
}

// handleErrorByID handles GET /api/errors/:id and PUT /api/errors/:id/status.
func (a *API) handleErrorByID(w http.ResponseWriter, r *http.Request) {
	if a.errorStore == nil {
		writeError(w, http.StatusServiceUnavailable, "error tracking not enabled")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/errors/")

	// PUT /api/errors/:id/status
	if strings.HasSuffix(path, "/status") {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id := strings.TrimSuffix(path, "/status")
		a.handleErrorUpdateStatus(w, r, id)
		return
	}

	// GET /api/errors/:id/events
	if strings.HasSuffix(path, "/events") {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id := strings.TrimSuffix(path, "/events")
		a.handleErrorEvents(w, r, id)
		return
	}

	// GET /api/errors/:id — group detail + recent events
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	group, err := a.errorStore.GetGroup(path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	events, _ := a.errorStore.GetEvents(path, 10)

	writeJSON(w, http.StatusOK, map[string]any{
		"group":  group,
		"events": events,
	})
}

// handleErrorUpdateStatus handles PUT /api/errors/:id/status.
func (a *API) handleErrorUpdateStatus(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Status != "unresolved" && req.Status != "resolved" && req.Status != "ignored" {
		writeError(w, http.StatusBadRequest, "status must be 'unresolved', 'resolved', or 'ignored'")
		return
	}

	if err := a.errorStore.UpdateGroupStatus(id, req.Status); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// handleErrorEvents handles GET /api/errors/:id/events.
func (a *API) handleErrorEvents(w http.ResponseWriter, r *http.Request, groupID string) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	events, err := a.errorStore.GetEvents(groupID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, events)
}

// handleErrorIngest handles POST /api/errors/ingest — ingest error events from SDK.
func (a *API) handleErrorIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.errorStore == nil {
		writeError(w, http.StatusServiceUnavailable, "error tracking not enabled")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Try batch first, fall back to single.
	var events []errs.ErrorEvent
	if err := json.Unmarshal(body, &events); err != nil {
		var single errs.ErrorEvent
		if err := json.Unmarshal(body, &single); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		events = []errs.ErrorEvent{single}
	}

	accepted := 0
	for _, ev := range events {
		if err := a.errorStore.IngestError(ev); err == nil {
			accepted++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accepted": accepted,
		"total":    len(events),
	})
}
