package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/monitor"
)

// handleMonitors serves the /api/monitors collection endpoints:
//
//	GET  /api/monitors       — list monitors with optional filters
//	POST /api/monitors       — create a new monitor
//	GET  /api/monitors/{id}  — get a single monitor by ID
//	PUT  /api/monitors/{id}  — update a monitor
//	DELETE /api/monitors/{id} — delete a monitor
//	POST /api/monitors/{id}/mute — mute a monitor for a duration
//	POST /api/monitors/{id}/test — trigger a test alert evaluation
func (a *API) handleMonitors(w http.ResponseWriter, r *http.Request) {
	if a.monitorService == nil {
		writeError(w, http.StatusServiceUnavailable, "monitor service not available")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/monitors")
	path = strings.TrimPrefix(path, "/")

	// POST /api/monitors/{id}/mute
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/mute") {
		id := strings.TrimSuffix(path, "/mute")
		a.handleMonitorMute(w, r, id)
		return
	}

	// POST /api/monitors/{id}/test
	if r.Method == http.MethodPost && strings.HasSuffix(path, "/test") {
		id := strings.TrimSuffix(path, "/test")
		a.handleMonitorTest(w, r, id)
		return
	}

	// Collection endpoints (no ID in path).
	if path == "" {
		switch r.Method {
		case http.MethodGet:
			a.handleListMonitors(w, r)
		case http.MethodPost:
			a.handleCreateMonitor(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// Single-resource endpoints (ID in path, no sub-resource).
	if !strings.Contains(path, "/") {
		switch r.Method {
		case http.MethodGet:
			a.handleGetMonitor(w, r, path)
		case http.MethodPut:
			a.handleUpdateMonitor(w, r, path)
		case http.MethodDelete:
			a.handleDeleteMonitor(w, r, path)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

// handleListMonitors serves GET /api/monitors.
func (a *API) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	filter := monitor.MonitorFilter{
		Service: r.URL.Query().Get("service"),
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = monitor.MonitorType(t)
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = monitor.MonitorStatus(s)
	}
	if e := r.URL.Query().Get("enabled"); e != "" {
		b, err := strconv.ParseBool(e)
		if err == nil {
			filter.Enabled = &b
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	results, err := a.monitorService.Monitors().List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// handleCreateMonitor serves POST /api/monitors.
func (a *API) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	var mon monitor.Monitor
	if err := json.NewDecoder(r.Body).Decode(&mon); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if mon.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if mon.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if mon.Service == "" {
		writeError(w, http.StatusBadRequest, "service is required")
		return
	}
	if mon.Operator == "" {
		mon.Operator = "gt" // default to "greater than"
	}

	mon.Status = monitor.MonitorStatusNoData

	if err := a.monitorService.Monitors().Save(r.Context(), &mon); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, mon)
}

// handleGetMonitor serves GET /api/monitors/{id}.
func (a *API) handleGetMonitor(w http.ResponseWriter, r *http.Request, id string) {
	mon, err := a.monitorService.Monitors().Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mon)
}

// handleUpdateMonitor serves PUT /api/monitors/{id}.
func (a *API) handleUpdateMonitor(w http.ResponseWriter, r *http.Request, id string) {
	existing, err := a.monitorService.Monitors().Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var update monitor.Monitor
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Preserve immutable fields.
	update.ID = existing.ID
	update.CreatedAt = existing.CreatedAt
	if update.Name == "" {
		update.Name = existing.Name
	}
	if update.Type == "" {
		update.Type = existing.Type
	}
	if update.Service == "" {
		update.Service = existing.Service
	}
	if update.Operator == "" {
		update.Operator = existing.Operator
	}

	if err := a.monitorService.Monitors().Update(r.Context(), &update); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, update)
}

// handleDeleteMonitor serves DELETE /api/monitors/{id}.
func (a *API) handleDeleteMonitor(w http.ResponseWriter, r *http.Request, id string) {
	if err := a.monitorService.Monitors().Delete(r.Context(), id); err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleMonitorMute serves POST /api/monitors/{id}/mute.
func (a *API) handleMonitorMute(w http.ResponseWriter, r *http.Request, id string) {
	mon, err := a.monitorService.Monitors().Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var body struct {
		Duration string `json:"duration"` // e.g. "1h", "30m"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	dur, err := time.ParseDuration(body.Duration)
	if err != nil || dur <= 0 {
		writeError(w, http.StatusBadRequest, "invalid duration; use Go duration format (e.g. '1h', '30m')")
		return
	}

	muteUntil := time.Now().Add(dur)
	mon.MutedUntil = &muteUntil
	mon.Status = monitor.MonitorStatusMuted

	if err := a.monitorService.Monitors().Update(r.Context(), mon); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mon)
}

// handleMonitorTest serves POST /api/monitors/{id}/test.
// It creates a synthetic alert event to verify webhook and incident integration.
func (a *API) handleMonitorTest(w http.ResponseWriter, r *http.Request, id string) {
	mon, err := a.monitorService.Monitors().Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, monitor.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	alert := monitor.AlertEvent{
		MonitorID:   mon.ID,
		MonitorName: mon.Name,
		Status:      monitor.MonitorStatusCritical,
		Value:       mon.Critical + 1, // synthetic breach
		Threshold:   mon.Critical,
		Service:     mon.Service,
		Action:      mon.Action,
		Message:     "[TEST] " + mon.Name + " critical threshold breached",
		CreatedAt:   time.Now(),
	}

	if err := a.monitorService.Alerts().SaveAlert(r.Context(), &alert); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, alert)
}

// handleAlerts serves the /api/alerts endpoints:
//
//	GET /api/alerts       — list alert events with optional filters
//	GET /api/alerts/{id}  — get a single alert event by ID
func (a *API) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if a.monitorService == nil {
		writeError(w, http.StatusServiceUnavailable, "monitor service not available")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/alerts")
	path = strings.TrimPrefix(path, "/")

	// GET /api/alerts — list
	if path == "" {
		filter := monitor.AlertFilter{
			MonitorID: r.URL.Query().Get("monitor_id"),
			Service:   r.URL.Query().Get("service"),
		}
		if s := r.URL.Query().Get("status"); s != "" {
			filter.Status = monitor.MonitorStatus(s)
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			filter.Limit, _ = strconv.Atoi(l)
		}
		if filter.Limit == 0 {
			filter.Limit = 100
		}

		results, err := a.monitorService.Alerts().ListAlerts(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, results)
		return
	}

	// GET /api/alerts/{id} — detail
	if !strings.Contains(path, "/") {
		alert, err := a.monitorService.Alerts().GetAlert(r.Context(), path)
		if err != nil {
			if errors.Is(err, monitor.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, alert)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
