package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/logstore"
)

// SetLogStore wires the log store to the admin API and registers routes.
func (a *API) SetLogStore(store logstore.LogStore) {
	a.logStore = store
}

// handleLogs handles GET /api/logs — query log entries.
func (a *API) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.logStore == nil {
		writeError(w, http.StatusServiceUnavailable, "log management not enabled")
		return
	}

	q := r.URL.Query()
	opts := logstore.QueryOpts{
		Search:  q.Get("search"),
		Level:   q.Get("level"),
		Service: q.Get("service"),
		TraceID: q.Get("traceId"),
	}

	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.Limit = n
		}
	}
	if v := q.Get("start"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			opts.StartTime = t
		}
	}
	if v := q.Get("end"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			opts.EndTime = t
		}
	}

	entries, err := a.logStore.Query(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

// handleLogStream handles GET /api/logs/stream — SSE live tail.
func (a *API) handleLogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.logStore == nil {
		writeError(w, http.StatusServiceUnavailable, "log management not enabled")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	q := r.URL.Query()
	filter := logstore.TailFilter{
		Level:   q.Get("level"),
		Service: q.Get("service"),
		Search:  q.Get("search"),
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch := a.logStore.Tail(ctx, filter)

	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleLogIngest handles POST /api/logs/ingest — ingest log entries from SDK.
func (a *API) handleLogIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.logStore == nil {
		writeError(w, http.StatusServiceUnavailable, "log management not enabled")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Try batch first, fall back to single.
	var entries []logstore.LogEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		var single logstore.LogEntry
		if err := json.Unmarshal(body, &single); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		entries = []logstore.LogEntry{single}
	}

	accepted := 0
	for _, e := range entries {
		if err := a.logStore.Write(e); err == nil {
			accepted++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accepted": accepted,
		"total":    len(entries),
	})
}

// handleLogServices handles GET /api/logs/services — list services.
func (a *API) handleLogServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.logStore == nil {
		writeError(w, http.StatusServiceUnavailable, "log management not enabled")
		return
	}

	svcs, err := a.logStore.Services()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, svcs)
}

// handleLogLevels handles GET /api/logs/levels — count by level.
func (a *API) handleLogLevels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.logStore == nil {
		writeError(w, http.StatusServiceUnavailable, "log management not enabled")
		return
	}

	counts, err := a.logStore.LevelCounts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, counts)
}
