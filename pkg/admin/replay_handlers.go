package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/neureaux/cloudmock/pkg/replay"
)

// SetReplayStore wires the replay session store to the admin API and registers routes.
func (a *API) SetReplayStore(store replay.Store) {
	a.replayStore = store

	a.mux.HandleFunc("/api/replay/sessions", a.handleReplaySessions)
	a.mux.HandleFunc("/api/replay/sessions/", a.handleReplaySessionByID)
}

// handleReplaySessions handles POST and GET /api/replay/sessions.
func (a *API) handleReplaySessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.handleReplaySessionCreate(w, r)
	case http.MethodGet:
		a.handleReplaySessionList(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleReplaySessionCreate handles POST /api/replay/sessions — saves a recorded session.
func (a *API) handleReplaySessionCreate(w http.ResponseWriter, r *http.Request) {
	if a.replayStore == nil {
		writeError(w, http.StatusServiceUnavailable, "replay not enabled")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	var session replay.Session
	if err := json.Unmarshal(body, &session); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if session.ID == "" {
		writeError(w, http.StatusBadRequest, "session id is required")
		return
	}

	if err := a.replayStore.SaveSession(session); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     session.ID,
		"events": len(session.Events),
	})
}

// handleReplaySessionList handles GET /api/replay/sessions — lists sessions (newest first).
func (a *API) handleReplaySessionList(w http.ResponseWriter, r *http.Request) {
	if a.replayStore == nil {
		writeError(w, http.StatusServiceUnavailable, "replay not enabled")
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	sessions, err := a.replayStore.ListSessions(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return sessions without events in the list view for performance.
	type sessionSummary struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		UserAgent string `json:"user_agent"`
		StartedAt string `json:"started_at"`
		Duration  int64  `json:"duration_ms"`
		EventCount int   `json:"event_count"`
		ErrorIDs  []string `json:"error_ids"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}

	result := make([]sessionSummary, len(sessions))
	for i, s := range sessions {
		result[i] = sessionSummary{
			ID:         s.ID,
			URL:        s.URL,
			UserAgent:  s.UserAgent,
			StartedAt:  s.StartedAt.Format("2006-01-02T15:04:05Z"),
			Duration:   s.Duration,
			EventCount: len(s.Events),
			ErrorIDs:   s.ErrorIDs,
			Width:      s.Width,
			Height:     s.Height,
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// handleReplaySessionByID handles GET /api/replay/sessions/:id.
func (a *API) handleReplaySessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.replayStore == nil {
		writeError(w, http.StatusServiceUnavailable, "replay not enabled")
		return
	}

	// Extract session ID: /api/replay/sessions/{id} or /api/replay/sessions/{id}/events
	path := strings.TrimPrefix(r.URL.Path, "/api/replay/sessions/")
	parts := strings.SplitN(path, "/", 2)
	sessionID := parts[0]

	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session ID required")
		return
	}

	session, err := a.replayStore.GetSession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// Check if this is the /events sub-path for streaming.
	if len(parts) > 1 && parts[1] == "events" {
		a.handleReplaySessionEvents(w, r, session)
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// handleReplaySessionEvents streams events for a session using newline-delimited JSON.
func (a *API) handleReplaySessionEvents(w http.ResponseWriter, r *http.Request, session *replay.Session) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	for _, event := range session.Events {
		if err := enc.Encode(event); err != nil {
			return // client disconnected
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}
