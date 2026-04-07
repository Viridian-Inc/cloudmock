// Package ingest provides HTTP handlers for receiving trace spans.
//
// Two endpoints are exposed:
//   - POST /v1/ingest  – CloudMock native JSON format (simple, low-overhead)
//   - POST /v1/traces  – OTLP/HTTP JSON format (standard OpenTelemetry)
//
// Received spans are buffered in memory and flushed to the store every second
// or when the buffer reaches 1000 spans, whichever comes first.
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/services/cloud-ingest/internal/store"
)

const (
	flushInterval = time.Second
	flushSize     = 1000
)

// Handler buffers incoming spans and batch-inserts them into the store.
type Handler struct {
	store *store.SpanStore

	mu     sync.Mutex
	buffer []store.Span

	flushCh chan struct{}
}

// New creates a Handler and starts the background flush goroutine.
// Call the returned stop function to drain the buffer on shutdown.
func New(ss *store.SpanStore) (*Handler, func()) {
	h := &Handler{
		store:   ss,
		buffer:  make([]store.Span, 0, flushSize),
		flushCh: make(chan struct{}, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				h.flush(context.Background())
			case <-h.flushCh:
				h.flush(context.Background())
			case <-ctx.Done():
				h.flush(context.Background())
				return
			}
		}
	}()

	stop := func() {
		cancel()
		<-done
	}
	return h, stop
}

// RegisterRoutes attaches the ingest endpoints to mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/ingest", h.handleNative)
	mux.HandleFunc("POST /v1/traces", h.handleOTLP)
}

// ---------------------------------------------------------------------------
// Native format
// ---------------------------------------------------------------------------

// nativeRequest is the body for POST /v1/ingest.
type nativeRequest struct {
	Spans []store.Span `json:"spans"`
}

func (h *Handler) handleNative(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		http.Error(w, `{"error":"missing X-Api-Key header"}`, http.StatusUnauthorized)
		return
	}

	var req nativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if len(req.Spans) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.enqueue(req.Spans)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": len(req.Spans)})
}

// ---------------------------------------------------------------------------
// OTLP/HTTP JSON format
// ---------------------------------------------------------------------------

// otlpRequest mirrors the top-level OTLP ExportTraceServiceRequest JSON shape.
// Only the fields we need are decoded; the rest are ignored.
type otlpRequest struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   otlpResource    `json:"resource"`
	ScopeSpans []otlpScopeSpan `json:"scopeSpans"`
}

type otlpResource struct {
	Attributes []otlpKV `json:"attributes"`
}

type otlpScopeSpan struct {
	Spans []otlpSpan `json:"spans"`
}

type otlpSpan struct {
	TraceID           string   `json:"traceId"`
	SpanID            string   `json:"spanId"`
	ParentSpanID      string   `json:"parentSpanId"`
	Name              string   `json:"name"`
	StartTimeUnixNano string   `json:"startTimeUnixNano"`
	EndTimeUnixNano   string   `json:"endTimeUnixNano"`
	Status            otlpStatus `json:"status"`
	Attributes        []otlpKV `json:"attributes"`
}

type otlpStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type otlpKV struct {
	Key   string    `json:"key"`
	Value otlpValue `json:"value"`
}

type otlpValue struct {
	StringValue string `json:"stringValue"`
	IntValue    string `json:"intValue"`
}

func (h *Handler) handleOTLP(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		http.Error(w, `{"error":"missing X-Api-Key header"}`, http.StatusUnauthorized)
		return
	}

	var req otlpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid OTLP JSON"}`, http.StatusBadRequest)
		return
	}

	spans := convertOTLP(req, apiKey)
	if len(spans) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.enqueue(spans)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": len(spans)})
}

// convertOTLP transforms an OTLP JSON payload into store.Span values.
// We extract well-known CloudMock / AWS attributes by convention.
func convertOTLP(req otlpRequest, apiKey string) []store.Span {
	var spans []store.Span

	for _, rs := range req.ResourceSpans {
		// Pull resource-level attributes.
		resAttrs := kvSliceToMap(rs.Resource.Attributes)

		orgID := attrStr(resAttrs, "cloudmock.org_id", "org_id")
		appID := attrStr(resAttrs, "cloudmock.app_id", "app_id")
		env := attrStr(resAttrs, "deployment.environment", "environment")
		if env == "" {
			env = "production"
		}
		region := attrStr(resAttrs, "cloud.region", "aws.region")
		accountID := attrStr(resAttrs, "cloud.account.id", "aws.account_id")

		for _, ss := range rs.ScopeSpans {
			for _, os := range ss.Spans {
				spanAttrs := kvSliceToMap(os.Attributes)

				service := attrStr(spanAttrs, "aws.service", "rpc.service", "db.system", "service.name")
				if service == "" {
					service = "unknown"
				}
				action := attrStr(spanAttrs, "aws.operation", "rpc.method", "db.operation", "faas.invoked_name")
				if action == "" {
					action = os.Name
				}

				t := parseNanoTime(os.StartTimeUnixNano)
				endT := parseNanoTime(os.EndTimeUnixNano)
				durationMs := 0.0
				if !t.IsZero() && !endT.IsZero() {
					durationMs = float64(endT.Sub(t).Microseconds()) / 1000.0
				}
				if t.IsZero() {
					t = time.Now().UTC()
				}

				// Merge resource + span attributes into a single map.
				merged := make(map[string]any, len(resAttrs)+len(spanAttrs))
				for k, v := range resAttrs {
					merged[k] = v
				}
				for k, v := range spanAttrs {
					merged[k] = v
				}

				sp := store.Span{
					Time:         t,
					TraceID:      os.TraceID,
					SpanID:       os.SpanID,
					ParentSpanID: os.ParentSpanID,
					OrgID:        orgID,
					AppID:        appID,
					Environment:  env,
					Source:       "otlp",
					Service:      service,
					Action:       action,
					Region:       region,
					AccountID:    accountID,
					RequestID:    attrStr(spanAttrs, "aws.request_id", "request_id"),
					DurationMs:   durationMs,
					StatusCode:   os.Status.Code,
					Attributes:   merged,
				}
				spans = append(spans, sp)
			}
		}
	}
	return spans
}

// ---------------------------------------------------------------------------
// Buffer management
// ---------------------------------------------------------------------------

func (h *Handler) enqueue(spans []store.Span) {
	h.mu.Lock()
	h.buffer = append(h.buffer, spans...)
	shouldFlush := len(h.buffer) >= flushSize
	h.mu.Unlock()

	if shouldFlush {
		select {
		case h.flushCh <- struct{}{}:
		default:
		}
	}
}

func (h *Handler) flush(ctx context.Context) {
	h.mu.Lock()
	if len(h.buffer) == 0 {
		h.mu.Unlock()
		return
	}
	batch := h.buffer
	h.buffer = make([]store.Span, 0, flushSize)
	h.mu.Unlock()

	if err := h.store.InsertBatch(ctx, batch); err != nil {
		log.Printf("ingest: flush error (dropped %d spans): %v", len(batch), err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func kvSliceToMap(kvs []otlpKV) map[string]string {
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		v := kv.Value.StringValue
		if v == "" {
			v = kv.Value.IntValue
		}
		m[kv.Key] = v
	}
	return m
}

// attrStr returns the value of the first key found in m, or "".
func attrStr(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

// parseNanoTime parses a Unix nanosecond timestamp string.
func parseNanoTime(s string) time.Time {
	if s == "" || s == "0" {
		return time.Time{}
	}
	var ns int64
	if _, err := fmt.Sscanf(s, "%d", &ns); err != nil {
		return time.Time{}
	}
	return time.Unix(0, ns).UTC()
}

