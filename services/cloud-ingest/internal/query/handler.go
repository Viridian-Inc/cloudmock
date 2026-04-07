// Package query provides HTTP handlers for reading stored spans and metrics.
package query

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Viridian-Inc/cloudmock/services/cloud-ingest/internal/store"
)

// Handler holds the span store used by all query endpoints.
type Handler struct {
	store *store.SpanStore
}

// New creates a new query Handler.
func New(ss *store.SpanStore) *Handler {
	return &Handler{store: ss}
}

// RegisterRoutes attaches query endpoints to mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/spans", h.handleSpans)
	mux.HandleFunc("GET /v1/traces/{traceID}", h.handleTrace)
	mux.HandleFunc("GET /v1/metrics", h.handleMetrics)
	mux.HandleFunc("GET /v1/topology", h.handleTopology)
}

// ---------------------------------------------------------------------------
// GET /v1/spans?org_id=&env=&since=&limit=
// ---------------------------------------------------------------------------

func (h *Handler) handleSpans(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	orgID := q.Get("org_id")
	if orgID == "" {
		jsonError(w, "org_id is required", http.StatusBadRequest)
		return
	}
	env := q.Get("env")

	since := time.Now().Add(-1 * time.Hour)
	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}

	limit := 100
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	spans, err := h.store.QueryByOrg(r.Context(), orgID, env, since, limit)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, spans)
}

// ---------------------------------------------------------------------------
// GET /v1/traces/{traceID}
// ---------------------------------------------------------------------------

func (h *Handler) handleTrace(w http.ResponseWriter, r *http.Request) {
	traceID := r.PathValue("traceID")
	if traceID == "" {
		jsonError(w, "traceID is required", http.StatusBadRequest)
		return
	}

	spans, err := h.store.QueryByTrace(r.Context(), traceID)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, spans)
}

// ---------------------------------------------------------------------------
// GET /v1/metrics?org_id=&env=&service=&start=&end=
// ---------------------------------------------------------------------------

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	orgID := q.Get("org_id")
	if orgID == "" {
		jsonError(w, "org_id is required", http.StatusBadRequest)
		return
	}
	env := q.Get("env")
	service := q.Get("service")

	end := time.Now().UTC()
	start := end.Add(-1 * time.Hour)

	if s := q.Get("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if s := q.Get("end"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			end = t
		}
	}

	pts, err := h.store.QueryMetrics(r.Context(), orgID, env, service, start, end)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, pts)
}

// ---------------------------------------------------------------------------
// GET /v1/topology?org_id=&env=&window=
// ---------------------------------------------------------------------------

func (h *Handler) handleTopology(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	orgID := q.Get("org_id")
	if orgID == "" {
		jsonError(w, "org_id is required", http.StatusBadRequest)
		return
	}
	env := q.Get("env")

	window := 5 * time.Minute
	if ws := q.Get("window"); ws != "" {
		if d, err := time.ParseDuration(ws); err == nil && d > 0 {
			window = d
		}
	}

	edges, err := h.store.QueryTopology(r.Context(), orgID, env, window)
	if err != nil {
		jsonError(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, edges)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
