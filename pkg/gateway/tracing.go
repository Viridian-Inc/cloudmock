package gateway

import (
	"github.com/Viridian-Inc/cloudmock/pkg/observability"
)

// Trace types live in pkg/observability — these aliases preserve the previous
// gateway.* API for the ~50 service tests, the admin API, and dataplane stores
// that already import them by their gateway-package names.
type (
	TraceContext = observability.TraceContext
	TraceSummary = observability.TraceSummary
	TimelineSpan = observability.TimelineSpan
	TraceStore   = observability.TraceStore
)

// NewTraceStore constructs a TraceStore. Forwards to pkg/observability.
func NewTraceStore(capacity int) *TraceStore {
	return observability.NewTraceStore(capacity)
}

// GenerateTraceID returns a new W3C-compatible trace ID.
func GenerateTraceID() string {
	return observability.GenerateTraceID()
}

// GenerateSpanID returns a new W3C-compatible span ID.
func GenerateSpanID() string {
	return observability.GenerateSpanID()
}
