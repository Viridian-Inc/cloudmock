package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/rum"
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

	// Add computed fields expected by the DevTools frontend.
	totalEvents := overview.LCP.Good + overview.LCP.NeedsImprovement + overview.LCP.Poor +
		overview.FID.Good + overview.FID.NeedsImprovement + overview.FID.Poor +
		overview.CLS.Good + overview.CLS.NeedsImprovement + overview.CLS.Poor +
		overview.TTFB.Good + overview.TTFB.NeedsImprovement + overview.TTFB.Poor +
		overview.FCP.Good + overview.FCP.NeedsImprovement + overview.FCP.Poor

	// Calculate UX score (0-100) based on percentage of "good" ratings.
	totalRated := 0
	totalGood := 0
	for _, v := range []rum.VitalRating{overview.LCP, overview.FID, overview.CLS, overview.TTFB, overview.FCP} {
		total := v.Good + v.NeedsImprovement + v.Poor
		totalRated += total
		totalGood += v.Good
	}
	score := 0.0
	if totalRated > 0 {
		score = float64(totalGood) / float64(totalRated) * 100.0
	}

	// Add rating strings based on P75 thresholds.
	addRating := func(v rum.VitalRating, goodThresh, poorThresh float64) map[string]any {
		rating := "poor"
		if v.P75 <= goodThresh {
			rating = "good"
		} else if v.P75 <= poorThresh {
			rating = "needs-improvement"
		}
		return map[string]any{
			"p75": v.P75, "good": v.Good, "needs_improvement": v.NeedsImprovement,
			"poor": v.Poor, "rating": rating,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"lcp":          addRating(overview.LCP, 2500, 4000),
		"fid":          addRating(overview.FID, 100, 300),
		"cls":          addRating(overview.CLS, 0.1, 0.25),
		"ttfb":         addRating(overview.TTFB, 800, 1800),
		"fcp":          addRating(overview.FCP, 1800, 3000),
		"score":        score,
		"total_events": totalEvents,
	})
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

	// Transform to match DevTools expected format.
	result := make([]map[string]any, len(pages))
	for i, p := range pages {
		result[i] = map[string]any{
			"route":        p.Route,
			"avg_load_ms":  p.AvgDurationMs,
			"p75_lcp_ms":   p.P75DurationMs,
			"avg_cls":      0.0, // Not tracked per-page; use 0
			"sample_count": p.Views,
		}
	}
	writeJSON(w, http.StatusOK, result)
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

	// Transform to match DevTools expected format.
	result := make([]map[string]any, len(groups))
	for i, g := range groups {
		result[i] = map[string]any{
			"fingerprint":       g.Fingerprint,
			"message":           g.Message,
			"count":             g.Count,
			"affected_sessions": g.Sessions,
			"last_seen":         g.LastSeen.Format("2006-01-02T15:04:05Z"),
			"sample_stack":      g.Stack,
		}
	}
	writeJSON(w, http.StatusOK, result)
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

	// Transform to match DevTools expected format.
	result := make([]map[string]any, len(sessions))
	for i, s := range sessions {
		durationSec := s.LastSeen.Sub(s.StartedAt).Seconds()
		if durationSec < 0 {
			durationSec = 0
		}
		result[i] = map[string]any{
			"session_id":   s.SessionID,
			"start":        s.StartedAt.Format("2006-01-02T15:04:05Z"),
			"end":          s.LastSeen.Format("2006-01-02T15:04:05Z"),
			"page_count":   s.PageViews,
			"error_count":  s.ErrorCount,
			"duration_sec": durationSec,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// handleRUMClicks handles GET /api/rum/clicks — returns rage click events.
func (a *API) handleRUMClicks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	minutes := 60
	if v := r.URL.Query().Get("minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minutes = n
		}
	}

	clicks, err := a.rumEngine.Store().RageClicks(minutes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, clicks)
}

// handleRUMJourneys handles GET /api/rum/journeys/:sessionId — returns navigation journey for a session.
func (a *API) handleRUMJourneys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.rumEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "RUM not enabled")
		return
	}

	// Extract session ID from path: /api/rum/journeys/{sessionId}
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/rum/journeys/")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session ID required")
		return
	}

	journeys, err := a.rumEngine.Store().UserJourneys(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, journeys)
}
