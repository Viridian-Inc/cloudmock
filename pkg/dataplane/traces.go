package dataplane

import (
	"context"
	"time"
)

type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Service      string
	Action       string
	Method       string
	Path         string
	StartTime    time.Time
	EndTime      time.Time
	DurationNs   uint64
	StatusCode   int
	Error        string
	TenantID     string
	OrgID        string
	UserID       string
	MemAllocKB   float64
	Goroutines   uint32
	Metadata     map[string]string
	ReqHeaders   map[string]string
	ReqBody      string
	RespBody     string
}

type TraceContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Service      string
	Action       string
	Method       string
	Path         string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	DurationMs   float64
	StatusCode   int
	Error        string
	Children     []*TraceContext
	Metadata     map[string]string
}

type TraceSummary struct {
	TraceID     string
	RootService string
	RootAction  string
	Method      string
	Path        string
	DurationMs  float64
	StatusCode  int
	SpanCount   int
	HasError    bool
	StartTime   time.Time
}

type TimelineSpan struct {
	SpanID        string
	ParentSpanID  string
	Service       string
	Action        string
	StartOffsetMs float64
	DurationMs    float64
	StatusCode    int
	Error         string
	Depth         int
	Metadata      map[string]string
}

type TraceFilter struct {
	Service  string
	HasError *bool
	TenantID string
	Limit    int
}

type TraceReader interface {
	Get(ctx context.Context, traceID string) (*TraceContext, error)
	Search(ctx context.Context, filter TraceFilter) ([]TraceSummary, error)
	Timeline(ctx context.Context, traceID string) ([]TimelineSpan, error)
}

type TraceWriter interface {
	WriteSpans(ctx context.Context, spans []*Span) error
}
