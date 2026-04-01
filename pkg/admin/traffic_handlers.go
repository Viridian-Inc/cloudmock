package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/traffic"
)

// --- handler: GET/POST /api/traffic/recordings ---

func (a *API) handleTrafficRecordings(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		recs, err := a.trafficEngine.Store().ListRecordings(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, recs)

	case http.MethodPost:
		// Create a recording directly (e.g. import).
		body, _ := io.ReadAll(r.Body)
		var rec traffic.Recording
		if err := json.Unmarshal(body, &rec); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := a.trafficEngine.Store().SaveRecording(r.Context(), &rec); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, rec)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- handler: GET/DELETE /api/traffic/recordings/{id} ---

func (a *API) handleTrafficRecordingByID(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}

	// Extract ID from path: /api/traffic/recordings/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/traffic/recordings/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "recording id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		rec, err := a.trafficEngine.Store().GetRecording(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "recording not found")
			return
		}
		writeJSON(w, http.StatusOK, rec)

	case http.MethodDelete:
		if err := a.trafficEngine.Store().DeleteRecording(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, "recording not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- handler: POST /api/traffic/record (start) ---

func (a *API) handleTrafficRecordStart(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Name        string `json:"name"`
		DurationSec int    `json:"duration_sec"`
		Filter      traffic.RecordingFilter `json:"filter"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Name == "" {
		req.Name = "untitled"
	}

	// Use background context — the capture goroutine must outlive the HTTP request
	rec, err := a.trafficEngine.StartRecording(context.Background(), req.Name, req.DurationSec, req.Filter)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

// --- handler: POST /api/traffic/record/stop ---

func (a *API) handleTrafficRecordStop(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rec, err := a.trafficEngine.StopRecording(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// --- handler: POST /api/traffic/replay (start) ---

func (a *API) handleTrafficReplayStart(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		RecordingID string  `json:"recording_id"`
		Speed       float64 `json:"speed"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.RecordingID == "" {
		writeError(w, http.StatusBadRequest, "recording_id required")
		return
	}

	// Use background context — replay goroutine must outlive the HTTP request
	run, err := a.trafficEngine.StartReplay(context.Background(), req.RecordingID, req.Speed)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// --- handler: GET /api/traffic/replay/{id} (status) ---
// --- handler: POST /api/traffic/replay/{id}/pause|resume|cancel ---

func (a *API) handleTrafficReplayByID(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}

	// Path: /api/traffic/replay/{id} or /api/traffic/replay/{id}/{action}
	rest := strings.TrimPrefix(r.URL.Path, "/api/traffic/replay/")
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if id == "" {
		writeError(w, http.StatusBadRequest, "run id required")
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		run, err := a.trafficEngine.Store().GetRun(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, run)

	case r.Method == http.MethodPost && action == "pause":
		if err := a.trafficEngine.PauseReplay(r.Context(), id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "paused", "id": id})

	case r.Method == http.MethodPost && action == "resume":
		if err := a.trafficEngine.ResumeReplay(r.Context(), id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "resumed", "id": id})

	case r.Method == http.MethodPost && action == "cancel":
		if err := a.trafficEngine.CancelReplay(r.Context(), id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled", "id": id})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- handler: GET /api/traffic/runs ---

func (a *API) handleTrafficRuns(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	runs, err := a.trafficEngine.Store().ListRuns(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// --- handler: POST /api/traffic/synthetic ---

func (a *API) handleTrafficSynthetic(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, _ := io.ReadAll(r.Body)
	var scenario traffic.SyntheticScenario
	if err := json.Unmarshal(body, &scenario); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	rec, err := a.trafficEngine.GenerateSynthetic(r.Context(), scenario)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

// --- handler: GET /api/traffic/compare?run_a=X&run_b=Y ---

func (a *API) handleTrafficCompare(w http.ResponseWriter, r *http.Request) {
	if a.trafficEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "traffic engine not configured")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	runA := r.URL.Query().Get("run_a")
	runB := r.URL.Query().Get("run_b")
	if runA == "" || runB == "" {
		writeError(w, http.StatusBadRequest, "run_a and run_b query params required")
		return
	}

	comparison, err := a.trafficEngine.CompareRuns(r.Context(), runA, runB)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}
