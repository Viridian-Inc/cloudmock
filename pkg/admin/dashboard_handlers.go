package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// handleDashboards handles GET /api/dashboards (list) and POST /api/dashboards (create).
func (a *API) handleDashboards(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.dashboardsMu.RLock()
		out := make([]Dashboard, len(a.dashboards))
		copy(out, a.dashboards)
		a.dashboardsMu.RUnlock()
		writeJSON(w, http.StatusOK, out)

	case http.MethodPost:
		var d Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if d.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		now := time.Now().UTC()
		d.ID = fmt.Sprintf("dash-%d", now.UnixNano())
		d.CreatedAt = now
		d.UpdatedAt = now
		// Assign IDs to widgets that lack them.
		for i := range d.Widgets {
			if d.Widgets[i].ID == "" {
				d.Widgets[i].ID = fmt.Sprintf("w-%d-%d", now.UnixNano(), i)
			}
		}

		a.dashboardsMu.Lock()
		a.dashboards = append(a.dashboards, d)
		a.persistDashboards()
		a.dashboardsMu.Unlock()

		a.auditLog(r.Context(), "dashboard.created", "dashboard:"+d.ID, map[string]any{"name": d.Name})
		writeJSON(w, http.StatusCreated, d)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleDashboardByID handles GET/PUT/DELETE /api/dashboards/{id}.
func (a *API) handleDashboardByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/dashboards/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing dashboard id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.dashboardsMu.RLock()
		d, idx := a.findDashboard(id)
		a.dashboardsMu.RUnlock()
		if idx < 0 {
			writeError(w, http.StatusNotFound, "dashboard not found")
			return
		}
		writeJSON(w, http.StatusOK, d)

	case http.MethodPut:
		var update Dashboard
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}

		a.dashboardsMu.Lock()
		_, idx := a.findDashboard(id)
		if idx < 0 {
			a.dashboardsMu.Unlock()
			writeError(w, http.StatusNotFound, "dashboard not found")
			return
		}
		// Preserve immutable fields.
		update.ID = id
		update.CreatedAt = a.dashboards[idx].CreatedAt
		update.UpdatedAt = time.Now().UTC()
		// Assign IDs to new widgets.
		for i := range update.Widgets {
			if update.Widgets[i].ID == "" {
				update.Widgets[i].ID = fmt.Sprintf("w-%d-%d", update.UpdatedAt.UnixNano(), i)
			}
		}
		a.dashboards[idx] = update
		a.persistDashboards()
		a.dashboardsMu.Unlock()

		a.auditLog(r.Context(), "dashboard.updated", "dashboard:"+id, map[string]any{"name": update.Name})
		writeJSON(w, http.StatusOK, update)

	case http.MethodDelete:
		a.dashboardsMu.Lock()
		_, idx := a.findDashboard(id)
		if idx < 0 {
			a.dashboardsMu.Unlock()
			writeError(w, http.StatusNotFound, "dashboard not found")
			return
		}
		a.dashboards = append(a.dashboards[:idx], a.dashboards[idx+1:]...)
		a.persistDashboards()
		a.dashboardsMu.Unlock()

		a.auditLog(r.Context(), "dashboard.deleted", "dashboard:"+id, nil)
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// findDashboard returns the dashboard and its index, or an empty Dashboard and -1.
// Caller must hold at least a read lock on dashboardsMu.
func (a *API) findDashboard(id string) (Dashboard, int) {
	for i, d := range a.dashboards {
		if d.ID == id {
			return d, i
		}
	}
	return Dashboard{}, -1
}

// handleMetricQuery handles POST /api/metrics/query — executes a metric query DSL.
func (a *API) handleMetricQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Query   string `json:"query"`
		Mode    string `json:"mode"`    // "timeseries" or "scalar"
		Minutes int    `json:"minutes"` // lookback window, default 15
		Bucket  string `json:"bucket"`  // bucket duration for timeseries, e.g. "1m"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	mq, err := ParseMetricQuery(req.Query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Mode == "" {
		req.Mode = "scalar"
	}
	if req.Mode != "scalar" && req.Mode != "timeseries" {
		writeError(w, http.StatusBadRequest, "mode must be 'scalar' or 'timeseries'")
		return
	}
	if req.Minutes <= 0 {
		req.Minutes = 15
	}

	bucketDur := parseBucketDuration(req.Bucket)
	if a.log == nil {
		writeError(w, http.StatusServiceUnavailable, "request log not available")
		return
	}

	result := ExecuteQuery(a.log, mq, req.Mode, req.Minutes, bucketDur)
	writeJSON(w, http.StatusOK, result)
}

// seedDefaultDashboard adds a "Service Overview" dashboard if the list is empty.
func (a *API) seedDefaultDashboard() {
	a.dashboardsMu.Lock()
	defer a.dashboardsMu.Unlock()

	if len(a.dashboards) > 0 {
		return
	}

	// Will be persisted at the end of this function via defer.
	defer a.persistDashboards()

	now := time.Now().UTC()
	a.dashboards = []Dashboard{
		{
			ID:          "dash-default",
			Name:        "Service Overview",
			Description: "Default dashboard showing key service metrics",
			CreatedAt:   now,
			UpdatedAt:   now,
			Widgets: []Widget{
				{
					ID:       "w-default-1",
					Title:    "Average Latency",
					Type:     "timeseries",
					Query:    "avg:latency_ms",
					Position: GridPosition{X: 0, Y: 0},
					Size:     GridSize{W: 6, H: 3},
				},
				{
					ID:       "w-default-2",
					Title:    "Request Count",
					Type:     "timeseries",
					Query:    "count:request_count",
					Position: GridPosition{X: 6, Y: 0},
					Size:     GridSize{W: 6, H: 3},
				},
				{
					ID:       "w-default-3",
					Title:    "P99 Latency",
					Type:     "timeseries",
					Query:    "p99:latency_ms",
					Position: GridPosition{X: 0, Y: 3},
					Size:     GridSize{W: 6, H: 3},
				},
				{
					ID:       "w-default-4",
					Title:    "Error Rate",
					Type:     "scalar",
					Query:    "avg:error_rate",
					Position: GridPosition{X: 6, Y: 3},
					Size:     GridSize{W: 6, H: 3},
				},
			},
		},
	}
}
