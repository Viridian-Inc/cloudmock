package gateway

import (
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway/traceid"
)

// TraceContext represents a single span in a distributed trace.
type TraceContext struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	Service      string            `json:"service"`
	Action       string            `json:"action"`
	Method       string            `json:"method,omitempty"`
	Path         string            `json:"path,omitempty"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration_ns"`
	DurationMs   float64           `json:"duration_ms"`
	StatusCode   int               `json:"status_code"`
	Error        string            `json:"error,omitempty"`
	Children     []*TraceContext   `json:"children,omitempty"`
	// Context propagation: feature flags, cache, policy decisions
	Metadata     map[string]string `json:"metadata,omitempty"`
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
	SpanID        string            `json:"span_id"`
	ParentSpanID  string            `json:"parent_span_id,omitempty"`
	Service       string            `json:"service"`
	Action        string            `json:"action"`
	StartOffsetMs float64           `json:"start_offset_ms"`
	DurationMs    float64           `json:"duration_ms"`
	StatusCode    int               `json:"status_code"`
	Error         string            `json:"error,omitempty"`
	Depth         int               `json:"depth"`
	Metadata      map[string]string `json:"metadata,omitempty"`
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

// Add stores a trace span. If a trace with the same TraceID already exists
// and the span has a ParentSpanID, it is attached as a child of the parent
// span in the existing trace tree. Otherwise, a new root trace is created.
func (ts *TraceStore) Add(trace *TraceContext) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// If this span belongs to an existing trace, merge it as a child.
	if trace.ParentSpanID != "" {
		if idx, ok := ts.index[trace.TraceID]; ok {
			root := ts.traces[idx]
			if root != nil {
				attachChild(root, trace)
				return
			}
		}
	}

	// New root trace — store in circular buffer.
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

// attachChild recursively searches the trace tree for the parent span and
// appends the child. If the parent is not found, attaches to root.
func attachChild(root *TraceContext, child *TraceContext) {
	if parent := findSpan(root, child.ParentSpanID); parent != nil {
		parent.Children = append(parent.Children, child)
		return
	}
	// Parent not found — attach to root as fallback
	root.Children = append(root.Children, child)
}

// findSpan searches the trace tree for a span with the given SpanID.
func findSpan(t *TraceContext, spanID string) *TraceContext {
	if t.SpanID == spanID {
		return t
	}
	for _, c := range t.Children {
		if found := findSpan(c, spanID); found != nil {
			return found
		}
	}
	return nil
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

// CountInRange returns the number of traces with StartTime in [start, end).
func (ts *TraceStore) CountInRange(start, end time.Time) int64 {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var count int64
	for _, t := range ts.traces {
		if t != nil && !t.StartTime.Before(start) && t.StartTime.Before(end) {
			count++
		}
	}
	return count
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

// GenerateTraceID returns a new unique W3C-compatible trace ID (32 hex chars).
func GenerateTraceID() string {
	return traceid.GenerateW3CTraceID()
}

// GenerateSpanID returns a new unique W3C-compatible span ID (16 hex chars).
func GenerateSpanID() string {
	return traceid.GenerateW3CSpanID()
}
