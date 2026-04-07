package memory

import (
	"context"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

// RequestStore wraps gateway.RequestLog to satisfy the dataplane.RequestReader
// and dataplane.RequestWriter interfaces for local (in-memory) mode.
type RequestStore struct {
	log *gateway.RequestLog
}

// NewRequestStore creates a RequestStore wrapping the given gateway.RequestLog.
func NewRequestStore(log *gateway.RequestLog) *RequestStore {
	return &RequestStore{log: log}
}

// Query returns request entries matching the filter.
func (s *RequestStore) Query(_ context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
	gf := gateway.RequestFilter{
		Service:      filter.Service,
		Path:         filter.Path,
		Method:       filter.Method,
		CallerID:     filter.CallerID,
		Action:       filter.Action,
		ErrorOnly:    filter.ErrorOnly,
		TraceID:      filter.TraceID,
		Level:        filter.Level,
		Limit:        filter.Limit,
		TenantID:     filter.TenantID,
		OrgID:        filter.OrgID,
		UserID:       filter.UserID,
		MinLatencyMs: filter.MinLatencyMs,
		MaxLatencyMs: filter.MaxLatencyMs,
		From:         filter.From,
		To:           filter.To,
	}
	entries := s.log.RecentFiltered(gf)
	out := make([]dataplane.RequestEntry, len(entries))
	for i, e := range entries {
		out[i] = convertRequestEntry(e)
	}
	return out, nil
}

// GetByID returns the request entry with the given ID.
func (s *RequestStore) GetByID(_ context.Context, id string) (*dataplane.RequestEntry, error) {
	e := s.log.GetByID(id)
	if e == nil {
		return nil, dataplane.ErrNotFound
	}
	de := convertRequestEntry(*e)
	return &de, nil
}

// Write adds a request entry to the log.
func (s *RequestStore) Write(_ context.Context, entry dataplane.RequestEntry) error {
	ge := gateway.RequestEntry{
		ID:             entry.ID,
		TraceID:        entry.TraceID,
		SpanID:         entry.SpanID,
		Timestamp:      entry.Timestamp,
		Service:        entry.Service,
		Action:         entry.Action,
		Method:         entry.Method,
		Path:           entry.Path,
		StatusCode:     entry.StatusCode,
		Latency:        entry.Latency,
		LatencyMs:      entry.LatencyMs,
		CallerID:       entry.CallerID,
		Error:          entry.Error,
		Level:          entry.Level,
		MemAllocKB:     int64(entry.MemAllocKB),
		Goroutines:     entry.Goroutines,
		RequestHeaders: entry.RequestHeaders,
		RequestBody:    entry.RequestBody,
		ResponseBody:   entry.ResponseBody,
	}
	s.log.Add(ge)
	return nil
}

// convertRequestEntry converts a gateway.RequestEntry to a dataplane.RequestEntry.
func convertRequestEntry(e gateway.RequestEntry) dataplane.RequestEntry {
	return dataplane.RequestEntry{
		ID:             e.ID,
		TraceID:        e.TraceID,
		SpanID:         e.SpanID,
		Timestamp:      e.Timestamp,
		Service:        e.Service,
		Action:         e.Action,
		Method:         e.Method,
		Path:           e.Path,
		StatusCode:     e.StatusCode,
		Latency:        e.Latency,
		LatencyMs:      e.LatencyMs,
		CallerID:       e.CallerID,
		Error:          e.Error,
		Level:          e.Level,
		MemAllocKB:     float64(e.MemAllocKB),
		Goroutines:     e.Goroutines,
		RequestHeaders: e.RequestHeaders,
		RequestBody:    e.RequestBody,
		ResponseBody:   e.ResponseBody,
		TenantID:       e.RequestHeaders["X-Tenant-Id"],
		OrgID:          e.RequestHeaders["X-Enterprise-Id"],
		UserID:         e.RequestHeaders["X-User-Id"],
	}
}

// Compile-time interface checks.
var (
	_ dataplane.RequestReader = (*RequestStore)(nil)
	_ dataplane.RequestWriter = (*RequestStore)(nil)
)
