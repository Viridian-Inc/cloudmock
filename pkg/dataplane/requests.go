package dataplane

import (
	"context"
	"time"
)

type RequestEntry struct {
	ID             string
	TraceID        string
	SpanID         string
	Timestamp      time.Time
	Service        string
	Action         string
	Method         string
	Path           string
	StatusCode     int
	Latency        time.Duration
	LatencyMs      float64
	CallerID       string
	Error          string
	Level          string
	MemAllocKB     float64
	Goroutines     int
	RequestHeaders map[string]string
	RequestBody    string
	ResponseBody   string
	TenantID       string
	OrgID          string
	UserID         string
}

type RequestFilter struct {
	Service      string
	Path         string
	Method       string
	CallerID     string
	Action       string
	ErrorOnly    bool
	TraceID      string
	Level        string
	Limit        int
	TenantID     string
	OrgID        string
	UserID       string
	MinLatencyMs float64
	MaxLatencyMs float64
	From         time.Time
	To           time.Time
}

type RequestReader interface {
	Query(ctx context.Context, filter RequestFilter) ([]RequestEntry, error)
	GetByID(ctx context.Context, id string) (*RequestEntry, error)
}

type RequestWriter interface {
	Write(ctx context.Context, entry RequestEntry) error
}
