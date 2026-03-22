package gateway

import (
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/gateway/traceid"
)

// TraceContext represents a single span in a distributed trace.
type TraceContext struct {
	TraceID      string          `json:"trace_id"`
	SpanID       string          `json:"span_id"`
	ParentSpanID string          `json:"parent_span_id,omitempty"`
	Service      string          `json:"service"`
	Action       string          `json:"action"`
	Method       string          `json:"method,omitempty"`
	Path         string          `json:"path,omitempty"`
	StartTime    time.Time       `json:"start_time"`
	EndTime      time.Time       `json:"end_time"`
	Duration     time.Duration   `json:"duration_ns"`
	DurationMs   float64         `json:"duration_ms"`
	StatusCode   int             `json:"status_code"`
	Error        string          `json:"error,omitempty"`
	Children     []*TraceContext `json:"children,omitempty"`
}

// TraceSummary is a lightweight representation for listing traces.
type TraceSummary struct {
	TraceID     string  `json:"trace_id"`
	RootService string  `json:"root_service"`
	RootAction  string  `json:"root_action"`
	Method      string  `json:"method"`
	Path        string  `json:"path"`
	DurationMs  float64 `json:"duration_ms"`
	StatusCode  int     `json:"status_code"`
	SpanCount   int     `json:"span_count"`
	HasError    bool    `json:"has_error"`
	StartTime   string  `json:"start_time"`
}

// TimelineSpan is a flattened span for waterfall rendering.
type TimelineSpan struct {
	SpanID       string  `json:"span_id"`
	ParentSpanID string  `json:"parent_span_id,omitempty"`
	Service      string  `json:"service"`
	Action       string  `json:"action"`
	StartOffsetMs float64 `json:"start_offset_ms"`
	DurationMs   float64 `json:"duration_ms"`
	StatusCode   int     `json:"status_code"`
	Error        string  `json:"error,omitempty"`
	Depth        int     `json:"depth"`
}

// TraceStore is a thread-safe circular buffer of recent traces, indexed by TraceID.
type TraceStore struct {
	mu      sync.RWMutex
	traces  []*TraceContext
	index   map[string]int // traceID -> position in traces slice
	pos     int
	size    int
	count   int
}

// NewTraceStore creates a TraceStore with the given capacity.
func NewTraceStore(capacity int) *TraceStore {
	if capacity <= 0 {
		capacity = 500
	}
	return &TraceStore{
		traces: make([]*TraceContext, capacity),
		index:  make(map[string]int, capacity),
		size:   capacity,
	}
}

// Add stores a trace in the circular buffer.
func (ts *TraceStore) Add(trace *TraceContext) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// If overwriting an existing slot, remove old index entry.
	if ts.traces[ts.pos] != nil {
		delete(ts.index, ts.traces[ts.pos].TraceID)
	}

	ts.traces[ts.pos] = trace
	ts.index[trace.TraceID] = ts.pos
	ts.pos = (ts.pos + 1) % ts.size
	if ts.count < ts.size {
		ts.count++
	}
}

// Get returns the trace with the given ID, or nil if not found.
func (ts *TraceStore) Get(traceID string) *TraceContext {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	idx, ok := ts.index[traceID]
	if !ok {
		return nil
	}
	return ts.traces[idx]
}

// Recent returns up to limit traces, newest first.
// Supports filtering by service and status.
func (ts *TraceStore) Recent(service string, hasError *bool, limit int) []TraceSummary {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if limit <= 0 {
		limit = ts.count
	}

	var result []TraceSummary
	for i := 0; i < ts.count && len(result) < limit; i++ {
		idx := (ts.pos - 1 - i + ts.size) % ts.size
		t := ts.traces[idx]
		if t == nil {
			continue
		}

		if service != "" && t.Service != service {
			continue
		}

		traceHasError := t.Error != "" || t.StatusCode >= 400
		if hasError != nil && *hasError != traceHasError {
			continue
		}

		spanCount := countSpans(t)

		result = append(result, TraceSummary{
			TraceID:     t.TraceID,
			RootService: t.Service,
			RootAction:  t.Action,
			Method:      t.Method,
			Path:        t.Path,
			DurationMs:  t.DurationMs,
			StatusCode:  t.StatusCode,
			SpanCount:   spanCount,
			HasError:    traceHasError,
			StartTime:   t.StartTime.Format(time.RFC3339Nano),
		})
	}
	return result
}

// Timeline returns a flattened waterfall view of the trace.
func (ts *TraceStore) Timeline(traceID string) []TimelineSpan {
	t := ts.Get(traceID)
	if t == nil {
		return nil
	}

	var spans []TimelineSpan
	flattenSpans(t, t.StartTime, 0, &spans)
	return spans
}

// countSpans counts the total spans (including children) in a trace.
func countSpans(t *TraceContext) int {
	count := 1
	for _, child := range t.Children {
		count += countSpans(child)
	}
	return count
}

// flattenSpans recursively flattens a trace tree into a list of timeline spans.
func flattenSpans(t *TraceContext, traceStart time.Time, depth int, out *[]TimelineSpan) {
	offsetMs := float64(t.StartTime.Sub(traceStart).Nanoseconds()) / 1e6
	if offsetMs < 0 {
		offsetMs = 0
	}

	*out = append(*out, TimelineSpan{
		SpanID:        t.SpanID,
		ParentSpanID:  t.ParentSpanID,
		Service:       t.Service,
		Action:        t.Action,
		StartOffsetMs: offsetMs,
		DurationMs:    t.DurationMs,
		StatusCode:    t.StatusCode,
		Error:         t.Error,
		Depth:         depth,
	})

	for _, child := range t.Children {
		flattenSpans(child, traceStart, depth+1, out)
	}
}

// GenerateTraceID returns a new unique trace ID.
func GenerateTraceID() string {
	return traceid.New()
}

// GenerateSpanID returns a new unique span ID.
func GenerateSpanID() string {
	return traceid.New()
}
