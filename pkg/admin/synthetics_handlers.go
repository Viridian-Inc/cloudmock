package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/synthetics"
)

// SetSyntheticsEngine sets the synthetics engine on the admin API and registers routes.
func (a *API) SetSyntheticsEngine(engine *synthetics.Engine) {
	a.syntheticsEngine = engine

	a.mux.HandleFunc("/api/synthetics/tests", a.handleSyntheticsTests)
	a.mux.HandleFunc("/api/synthetics/tests/", a.handleSyntheticsTestByID)
	a.mux.HandleFunc("/api/synthetics/status", a.handleSyntheticsStatus)
}

// handleSyntheticsTests handles GET (list) and POST (create) for synthetic tests.
func (a *API) handleSyntheticsTests(w http.ResponseWriter, r *http.Request) {
	if a.syntheticsEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "synthetics engine not available")
		return
	}

	switch r.Method {
	case http.MethodGet:
		tests := a.syntheticsEngine.Store().ListTests()
		writeJSON(w, http.StatusOK, tests)

	case http.MethodPost:
		var input struct {
			Name       string            `json:"name"`
			Type       string            `json:"type"`
			URL        string            `json:"url"`
			Method     string            `json:"method"`
			Headers    map[string]string `json:"headers"`
			Body       string            `json:"body"`
			Assertions []synthetics.Assertion `json:"assertions"`
			IntervalS  int               `json:"interval_seconds"`
			TimeoutS   int               `json:"timeout_seconds"`
			Enabled    bool              `json:"enabled"`
			Locations  []string          `json:"locations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if input.URL == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}

		interval := time.Duration(input.IntervalS) * time.Second
		timeout := time.Duration(input.TimeoutS) * time.Second

		test := a.syntheticsEngine.Store().AddTest(synthetics.SyntheticTest{
			Name:       input.Name,
			Type:       input.Type,
			URL:        input.URL,
			Method:     input.Method,
			Headers:    input.Headers,
			Body:       input.Body,
			Assertions: input.Assertions,
			Interval:   interval,
			Timeout:    timeout,
			Enabled:    input.Enabled,
			Locations:  input.Locations,
		})

		if test.Enabled {
			a.syntheticsEngine.ScheduleTest(test)
		}

		writeJSON(w, http.StatusCreated, test)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSyntheticsTestByID handles operations on a specific test.
// GET  /api/synthetics/tests/:id          — get test
// GET  /api/synthetics/tests/:id/results  — get results
// POST /api/synthetics/tests/:id/run      — run now
// DELETE /api/synthetics/tests/:id        — delete
func (a *API) handleSyntheticsTestByID(w http.ResponseWriter, r *http.Request) {
	if a.syntheticsEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "synthetics engine not available")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/synthetics/tests/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusBadRequest, "test id is required")
		return
	}

	subpath := ""
	if len(parts) > 1 {
		subpath = parts[1]
	}

	switch {
	case subpath == "results" && r.Method == http.MethodGet:
		results := a.syntheticsEngine.Store().GetResults(id, 50)
		writeJSON(w, http.StatusOK, results)

	case subpath == "run" && r.Method == http.MethodPost:
		result := a.syntheticsEngine.RunTest(id)
		if result == nil {
			writeError(w, http.StatusNotFound, "test not found")
			return
		}
		a.syntheticsEngine.Store().AddResult(*result)
		writeJSON(w, http.StatusOK, result)

	case subpath == "" && r.Method == http.MethodGet:
		test, ok := a.syntheticsEngine.Store().GetTest(id)
		if !ok {
			writeError(w, http.StatusNotFound, "test not found")
			return
		}
		writeJSON(w, http.StatusOK, test)

	case subpath == "" && r.Method == http.MethodDelete:
		if !a.syntheticsEngine.Store().DeleteTest(id) {
			writeError(w, http.StatusNotFound, "test not found")
			return
		}
		a.syntheticsEngine.StopTest(id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSyntheticsStatus returns a summary of all synthetic tests.
func (a *API) handleSyntheticsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.syntheticsEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "synthetics engine not available")
		return
	}
	writeJSON(w, http.StatusOK, a.syntheticsEngine.Status())
}
