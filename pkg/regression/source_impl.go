package regression

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type prodMetricSource struct {
	prom promv1.API
	db   *sql.DB
}

// NewMetricSource returns a MetricSource backed by Prometheus (for
// latency/error/count aggregates) and DuckDB (for cache miss, fanout, and
// payload metrics derived from spans).
func NewMetricSource(prom promv1.API, db *sql.DB) MetricSource {
	return &prodMetricSource{prom: prom, db: db}
}

// WindowMetrics queries Prometheus recording rules and DuckDB for the
// given service+action over the supplied time window.
func (s *prodMetricSource) WindowMetrics(ctx context.Context, service, action string, window TimeWindow) (*WindowMetrics, error) {
	wm := &WindowMetrics{
		Service: service,
		Action:  action,
	}

	ts := window.End

	p50, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:service_p50_5m{service=%q,action=%q}`, service, action), ts)
	if err != nil {
		return nil, fmt.Errorf("window metrics p50: %w", err)
	}
	p95, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:service_p95_5m{service=%q,action=%q}`, service, action), ts)
	if err != nil {
		return nil, fmt.Errorf("window metrics p95: %w", err)
	}
	p99, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:service_p99_5m{service=%q,action=%q}`, service, action), ts)
	if err != nil {
		return nil, fmt.Errorf("window metrics p99: %w", err)
	}
	errRate, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:service_error_rate_5m{service=%q,action=%q}`, service, action), ts)
	if err != nil {
		return nil, fmt.Errorf("window metrics error rate: %w", err)
	}
	reqCount, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:service_request_count_5m{service=%q,action=%q}`, service, action), ts)
	if err != nil {
		return nil, fmt.Errorf("window metrics request count: %w", err)
	}

	// Prometheus stores latency in seconds; convert to milliseconds.
	wm.P50Ms = p50 * 1000
	wm.P95Ms = p95 * 1000
	wm.P99Ms = p99 * 1000
	wm.ErrorRate = errRate
	wm.RequestCount = int64(math.Round(reqCount))

	if s.db != nil {
		if err := s.enrichFromDB(ctx, wm, service, window); err != nil {
			// Trace-based metrics are best-effort; don't fail the whole call.
			_ = err
		}
	}

	return wm, nil
}

// TenantWindowMetrics queries the per-tenant P99 recording rule for the given
// service and tenant.
func (s *prodMetricSource) TenantWindowMetrics(ctx context.Context, service, tenantID string, window TimeWindow) (*WindowMetrics, error) {
	wm := &WindowMetrics{
		Service: service,
	}

	p99, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:tenant_p99_5m{service=%q,tenant_id=%q}`, service, tenantID), window.End)
	if err != nil {
		return nil, fmt.Errorf("tenant window metrics p99: %w", err)
	}
	wm.P99Ms = p99 * 1000

	return wm, nil
}

// FleetWindowMetrics queries the fleet-level P99 recording rule for the given
// service.
func (s *prodMetricSource) FleetWindowMetrics(ctx context.Context, service string, window TimeWindow) (*WindowMetrics, error) {
	wm := &WindowMetrics{
		Service: service,
	}

	p99, err := s.queryScalar(ctx, fmt.Sprintf(`cloudmock:fleet_p99_5m{service=%q}`, service), window.End)
	if err != nil {
		return nil, fmt.Errorf("fleet window metrics p99: %w", err)
	}
	wm.P99Ms = p99 * 1000

	return wm, nil
}

// ListServices returns distinct service names seen in spans within the last
// hour.
func (s *prodMetricSource) ListServices(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT service_name FROM spans WHERE start_time > current_timestamp - INTERVAL '1 hour'`)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("list services scan: %w", err)
		}
		services = append(services, name)
	}
	return services, rows.Err()
}

// ListTenants returns tenant IDs for a service ordered by request volume
// (descending), limited to 100 results.
func (s *prodMetricSource) ListTenants(ctx context.Context, service string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tenant_id FROM spans WHERE service_name = ? AND start_time > current_timestamp - INTERVAL '1 hour' GROUP BY tenant_id ORDER BY count(*) DESC LIMIT 100`,
		service)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []string
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err != nil {
			return nil, fmt.Errorf("list tenants scan: %w", err)
		}
		tenants = append(tenants, tid)
	}
	return tenants, rows.Err()
}

// enrichFromDB populates the cache-miss rate, average fanout span
// count, and average response payload size from the spans table.
func (s *prodMetricSource) enrichFromDB(ctx context.Context, wm *WindowMetrics, service string, window TimeWindow) error {
	start := window.Start
	end := window.End

	// Cache miss rate: metadata is stored as JSON, so we use json_extract_string.
	var cacheMissRate sql.NullFloat64
	if err := s.db.QueryRowContext(ctx,
		`SELECT CAST(count(*) FILTER (WHERE json_extract_string(metadata, '$.x-cache-status') = 'MISS') AS DOUBLE) / count(*) FROM spans WHERE service_name = ? AND start_time BETWEEN ? AND ?`,
		service, start, end,
	).Scan(&cacheMissRate); err == nil && cacheMissRate.Valid {
		wm.CacheMissRate = cacheMissRate.Float64
	}

	// Average fanout (spans per trace).
	var avgFanout sql.NullFloat64
	if err := s.db.QueryRowContext(ctx,
		`SELECT avg(cnt) FROM (SELECT trace_id, count(*) as cnt FROM spans WHERE service_name = ? AND start_time BETWEEN ? AND ? GROUP BY trace_id)`,
		service, start, end,
	).Scan(&avgFanout); err == nil && avgFanout.Valid {
		wm.AvgSpanCount = avgFanout.Float64
	}

	// Average response payload size.
	var avgPayload sql.NullFloat64
	if err := s.db.QueryRowContext(ctx,
		`SELECT avg(length(response_body)) FROM spans WHERE service_name = ? AND start_time BETWEEN ? AND ?`,
		service, start, end,
	).Scan(&avgPayload); err == nil && avgPayload.Valid {
		wm.AvgRespSize = avgPayload.Float64
	}

	return nil
}

// queryScalar executes a Prometheus instant query and returns the first sample
// value. Returns 0 (not an error) when the result set is empty or NaN.
func (s *prodMetricSource) queryScalar(ctx context.Context, query string, ts time.Time) (float64, error) {
	result, _, err := s.prom.Query(ctx, query, ts)
	if err != nil {
		return 0, fmt.Errorf("prometheus query %q: %w", query, err)
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

// Compile-time interface check.
var _ MetricSource = (*prodMetricSource)(nil)
