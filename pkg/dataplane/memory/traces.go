package memory

import (
	"context"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

// TraceStore wraps gateway.TraceStore to satisfy the dataplane.TraceReader
// and dataplane.TraceWriter interfaces for local (in-memory) mode.
type TraceStore struct {
	store *gateway.TraceStore
}

// NewTraceStore creates a TraceStore wrapping the given gateway.TraceStore.
func NewTraceStore(store *gateway.TraceStore) *TraceStore {
	return &TraceStore{store: store}
}

// Get returns the trace with the given ID, converting from gateway types.
func (s *TraceStore) Get(_ context.Context, traceID string) (*dataplane.TraceContext, error) {
	tc := s.store.Get(traceID)
	if tc == nil {
		return nil, dataplane.ErrNotFound
	}
	return convertTraceContext(tc), nil
}

// Search returns recent traces matching the filter.
func (s *TraceStore) Search(_ context.Context, filter dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
	gw := s.store.Recent(filter.Service, filter.HasError, filter.Limit)
	out := make([]dataplane.TraceSummary, len(gw))
	for i, ts := range gw {
		startTime, _ := time.Parse(time.RFC3339Nano, ts.StartTime)
		out[i] = dataplane.TraceSummary{
			TraceID:     ts.TraceID,
			RootService: ts.RootService,
			RootAction:  ts.RootAction,
			Method:      ts.Method,
			Path:        ts.Path,
			DurationMs:  ts.DurationMs,
			StatusCode:  ts.StatusCode,
			SpanCount:   ts.SpanCount,
			HasError:    ts.HasError,
			StartTime:   startTime,
		}
	}
	return out, nil
}

// Timeline returns a flattened waterfall view of the trace.
func (s *TraceStore) Timeline(_ context.Context, traceID string) ([]dataplane.TimelineSpan, error) {
	gw := s.store.Timeline(traceID)
	if gw == nil {
		return nil, dataplane.ErrNotFound
	}
	out := make([]dataplane.TimelineSpan, len(gw))
	for i, ts := range gw {
		out[i] = dataplane.TimelineSpan{
			SpanID:        ts.SpanID,
			ParentSpanID:  ts.ParentSpanID,
			Service:       ts.Service,
			Action:        ts.Action,
			StartOffsetMs: ts.StartOffsetMs,
			DurationMs:    ts.DurationMs,
			StatusCode:    ts.StatusCode,
			Error:         ts.Error,
			Depth:         ts.Depth,
			Metadata:      ts.Metadata,
		}
	}
	return out, nil
}

// WriteSpans writes spans to the underlying trace store.
func (s *TraceStore) WriteSpans(_ context.Context, spans []*dataplane.Span) error {
	for _, sp := range spans {
		tc := &gateway.TraceContext{
			TraceID:      sp.TraceID,
			SpanID:       sp.SpanID,
			ParentSpanID: sp.ParentSpanID,
			Service:      sp.Service,
			Action:       sp.Action,
			Method:       sp.Method,
			Path:         sp.Path,
			StartTime:    sp.StartTime,
			EndTime:      sp.EndTime,
			Duration:     time.Duration(sp.DurationNs),
			DurationMs:   float64(sp.DurationNs) / 1e6,
			StatusCode:   sp.StatusCode,
			Error:        sp.Error,
			Metadata:     sp.Metadata,
		}
		s.store.Add(tc)
	}
	return nil
}

// convertTraceContext recursively converts a gateway TraceContext to a dataplane TraceContext.
func convertTraceContext(tc *gateway.TraceContext) *dataplane.TraceContext {
	out := &dataplane.TraceContext{
		TraceID:      tc.TraceID,
		SpanID:       tc.SpanID,
		ParentSpanID: tc.ParentSpanID,
		Service:      tc.Service,
		Action:       tc.Action,
		Method:       tc.Method,
		Path:         tc.Path,
		StartTime:    tc.StartTime,
		EndTime:      tc.EndTime,
		Duration:     tc.Duration,
		DurationMs:   tc.DurationMs,
		StatusCode:   tc.StatusCode,
		Error:        tc.Error,
		Metadata:     tc.Metadata,
	}
	if len(tc.Children) > 0 {
		out.Children = make([]*dataplane.TraceContext, len(tc.Children))
		for i, child := range tc.Children {
			out.Children[i] = convertTraceContext(child)
		}
	}
	return out
}

// Compile-time interface checks.
var (
	_ dataplane.TraceReader = (*TraceStore)(nil)
	_ dataplane.TraceWriter = (*TraceStore)(nil)
)
