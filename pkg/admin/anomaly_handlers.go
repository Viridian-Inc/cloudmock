package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/Viridian-Inc/cloudmock/pkg/anomaly"
)

// SetAnomalyDetector wires the anomaly detector to the admin API and registers routes.
func (a *API) SetAnomalyDetector(d *anomaly.Detector) {
	a.anomalyDetector = d

	a.mux.HandleFunc("/api/anomalies", a.handleAnomalies)
	a.mux.HandleFunc("/api/anomalies/baselines", a.handleAnomalyBaselines)
	a.mux.HandleFunc("/api/anomalies/check", a.handleAnomalyCheck)
}

// handleAnomalies handles GET /api/anomalies — lists recent anomalies.
func (a *API) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.anomalyDetector == nil {
		writeError(w, http.StatusServiceUnavailable, "anomaly detection not enabled")
		return
	}

	minutes := 60
	if v := r.URL.Query().Get("minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minutes = n
		}
	}

	anomalies := a.anomalyDetector.GetAnomalies(minutes)
	writeJSON(w, http.StatusOK, anomalies)
}

// handleAnomalyBaselines handles GET /api/anomalies/baselines — shows learned baselines.
func (a *API) handleAnomalyBaselines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.anomalyDetector == nil {
		writeError(w, http.StatusServiceUnavailable, "anomaly detection not enabled")
		return
	}

	baselines := a.anomalyDetector.GetBaselines()
	writeJSON(w, http.StatusOK, baselines)
}

// handleAnomalyCheck handles POST /api/anomalies/check — manually checks a value.
func (a *API) handleAnomalyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.anomalyDetector == nil {
		writeError(w, http.StatusServiceUnavailable, "anomaly detection not enabled")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var req struct {
		Service string  `json:"service"`
		Metric  string  `json:"metric"`
		Value   float64 `json:"value"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Service == "" || req.Metric == "" {
		writeError(w, http.StatusBadRequest, "service and metric are required")
		return
	}

	result := a.anomalyDetector.Check(req.Service, req.Metric, req.Value)
	if result == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"anomaly":  false,
			"service":  req.Service,
			"metric":   req.Metric,
			"value":    req.Value,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"anomaly":  true,
		"detail":   result,
	})
}
