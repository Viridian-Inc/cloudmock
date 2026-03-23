package dataplane

import (
	"context"
	"time"
)

type ServiceMetrics struct {
	Service      string
	RequestCount int64
	ErrorCount   int64
	ErrorRate    float64
	P50Ms        float64
	P95Ms        float64
	P99Ms        float64
}

type LatencyPercentiles struct {
	P50Ms float64
	P95Ms float64
	P99Ms float64
}

type MetricReader interface {
	ServiceStats(ctx context.Context, service string, window time.Duration) (*ServiceMetrics, error)
	Percentiles(ctx context.Context, service, action string, window time.Duration) (*LatencyPercentiles, error)
}

type MetricWriter interface {
	Record(ctx context.Context, service, action string, latencyMs float64, statusCode int) error
}
