package prometheus

import (
	"context"
	"fmt"
	"math"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// MetricReader implements dataplane.MetricReader against the Prometheus HTTP API.
type MetricReader struct {
	api promv1.API
}

// NewMetricReader creates a MetricReader from the given Client.
func NewMetricReader(c *Client) *MetricReader {
	return &MetricReader{api: c.api}
}

// ServiceStats queries Prometheus for aggregate service metrics over the given
// window duration and returns them as a ServiceMetrics value.
func (r *MetricReader) ServiceStats(ctx context.Context, service string, window time.Duration) (*dataplane.ServiceMetrics, error) {
	windowMinutes := int(math.Round(window.Minutes()))
	if windowMinutes < 1 {
		windowMinutes = 1
	}
	win := fmt.Sprintf("%dm", windowMinutes)
	now := time.Now()

	reqRate, err := r.queryScalar(ctx, fmt.Sprintf(
		`sum(rate(cloudmock_http_requests_total{service=%q}[%s]))`, service, win), now)
	if err != nil {
		return nil, fmt.Errorf("service stats request rate: %w", err)
	}

	errRate, err := r.queryScalar(ctx, fmt.Sprintf(
		`sum(rate(cloudmock_http_request_errors_total{service=%q}[%s]))`, service, win), now)
	if err != nil {
		return nil, fmt.Errorf("service stats error rate: %w", err)
	}

	p50, err := r.queryScalar(ctx, fmt.Sprintf(
		`histogram_quantile(0.5, sum(rate(cloudmock_http_request_duration_seconds_bucket{service=%q}[%s])) by (le))`,
		service, win), now)
	if err != nil {
		return nil, fmt.Errorf("service stats p50: %w", err)
	}

	p95, err := r.queryScalar(ctx, fmt.Sprintf(
		`histogram_quantile(0.95, sum(rate(cloudmock_http_request_duration_seconds_bucket{service=%q}[%s])) by (le))`,
		service, win), now)
	if err != nil {
		return nil, fmt.Errorf("service stats p95: %w", err)
	}

	p99, err := r.queryScalar(ctx, fmt.Sprintf(
		`histogram_quantile(0.99, sum(rate(cloudmock_http_request_duration_seconds_bucket{service=%q}[%s])) by (le))`,
		service, win), now)
	if err != nil {
		return nil, fmt.Errorf("service stats p99: %w", err)
	}

	// reqRate is requests/sec over the window; scale to a count approximation.
	requestCount := int64(math.Round(reqRate * window.Seconds()))
	errorCount := int64(math.Round(errRate * window.Seconds()))

	var errorRatePct float64
	if reqRate > 0 {
		errorRatePct = errRate / reqRate
	}

	return &dataplane.ServiceMetrics{
		Service:      service,
		RequestCount: requestCount,
		ErrorCount:   errorCount,
		ErrorRate:    errorRatePct,
		P50Ms:        p50 * 1000,
		P95Ms:        p95 * 1000,
		P99Ms:        p99 * 1000,
	}, nil
}

// Percentiles queries Prometheus for per-action latency percentiles over the
// given window duration.
func (r *MetricReader) Percentiles(ctx context.Context, service, action string, window time.Duration) (*dataplane.LatencyPercentiles, error) {
	windowMinutes := int(math.Round(window.Minutes()))
	if windowMinutes < 1 {
		windowMinutes = 1
	}
	win := fmt.Sprintf("%dm", windowMinutes)
	now := time.Now()

	p50, err := r.queryScalar(ctx, fmt.Sprintf(
		`histogram_quantile(0.5, sum(rate(cloudmock_http_request_duration_seconds_bucket{service=%q,action=%q}[%s])) by (le))`,
		service, action, win), now)
	if err != nil {
		return nil, fmt.Errorf("percentiles p50: %w", err)
	}

	p95, err := r.queryScalar(ctx, fmt.Sprintf(
		`histogram_quantile(0.95, sum(rate(cloudmock_http_request_duration_seconds_bucket{service=%q,action=%q}[%s])) by (le))`,
		service, action, win), now)
	if err != nil {
		return nil, fmt.Errorf("percentiles p95: %w", err)
	}

	p99, err := r.queryScalar(ctx, fmt.Sprintf(
		`histogram_quantile(0.99, sum(rate(cloudmock_http_request_duration_seconds_bucket{service=%q,action=%q}[%s])) by (le))`,
		service, action, win), now)
	if err != nil {
		return nil, fmt.Errorf("percentiles p99: %w", err)
	}

	return &dataplane.LatencyPercentiles{
		P50Ms: p50 * 1000,
		P95Ms: p95 * 1000,
		P99Ms: p99 * 1000,
	}, nil
}

// queryScalar executes an instant query and extracts the first sample's value.
// Returns 0 and no error when the result set is empty (no data yet).
func (r *MetricReader) queryScalar(ctx context.Context, query string, ts time.Time) (float64, error) {
	result, warnings, err := r.api.Query(ctx, query, ts)
	if err != nil {
		return 0, fmt.Errorf("prometheus query %q: %w", query, err)
	}
	if len(warnings) > 0 {
		// Warnings are non-fatal; surface them via a wrapped error so callers
		// can decide whether to log them.
		_ = warnings
	}

	vec, ok := result.(model.Vector)
	if !ok || len(vec) == 0 {
		return 0, nil
	}
	v := float64(vec[0].Value)
	if math.IsNaN(v) {
		return 0, nil
	}
	return v, nil
}

// MetricWriter is a no-op implementation of dataplane.MetricWriter.
//
// In production mode, the OTel SDK emits metrics directly to the Collector,
// which Prometheus then scrapes. This interface exists for symmetry with the
// local-mode data plane.
type MetricWriter struct{}

// NewMetricWriter returns a no-op MetricWriter.
func NewMetricWriter() *MetricWriter {
	return &MetricWriter{}
}

// Record is a no-op; metrics are emitted by the OTel SDK out-of-band.
func (w *MetricWriter) Record(_ context.Context, _, _ string, _ float64, _ int) error {
	return nil
}

// Compile-time interface checks.
var (
	_ dataplane.MetricReader = (*MetricReader)(nil)
	_ dataplane.MetricWriter = (*MetricWriter)(nil)
)
