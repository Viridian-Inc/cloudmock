package memory

import (
	"context"
	"sort"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/regression"
)

// MetricSource implements regression.MetricSource by computing WindowMetrics
// from gateway in-memory stores.
type MetricSource struct {
	log   *gateway.RequestLog
	trace *gateway.TraceStore
}

// NewMetricSource creates a MetricSource backed by the given gateway stores.
func NewMetricSource(log *gateway.RequestLog, trace *gateway.TraceStore) *MetricSource {
	return &MetricSource{log: log, trace: trace}
}

// WindowMetrics computes metrics for a service+action within the given time window.
func (m *MetricSource) WindowMetrics(_ context.Context, service, action string, window regression.TimeWindow) (*regression.WindowMetrics, error) {
	entries := m.log.RecentFiltered(gateway.RequestFilter{
		Service: service,
		Action:  action,
		From:    window.Start,
		To:      window.End,
	})

	wm := computeMetrics(entries, service, action)
	m.enrichFromTraces(wm, service, window)
	return wm, nil
}

// TenantWindowMetrics computes metrics for a service+tenant within the given time window.
func (m *MetricSource) TenantWindowMetrics(_ context.Context, service, tenantID string, window regression.TimeWindow) (*regression.WindowMetrics, error) {
	entries := m.log.RecentFiltered(gateway.RequestFilter{
		Service:  service,
		TenantID: tenantID,
		From:     window.Start,
		To:       window.End,
	})

	wm := computeMetrics(entries, service, "")
	m.enrichFromTraces(wm, service, window)
	return wm, nil
}

// FleetWindowMetrics computes metrics for all actions of a service within the given time window.
func (m *MetricSource) FleetWindowMetrics(_ context.Context, service string, window regression.TimeWindow) (*regression.WindowMetrics, error) {
	entries := m.log.RecentFiltered(gateway.RequestFilter{
		Service: service,
		From:    window.Start,
		To:      window.End,
	})

	wm := computeMetrics(entries, service, "")
	m.enrichFromTraces(wm, service, window)
	return wm, nil
}

// ListServices returns unique service names from recent requests.
func (m *MetricSource) ListServices(_ context.Context) ([]string, error) {
	entries := m.log.RecentFiltered(gateway.RequestFilter{})

	seen := make(map[string]struct{})
	var services []string
	for _, e := range entries {
		if e.Service == "" {
			continue
		}
		if _, ok := seen[e.Service]; !ok {
			seen[e.Service] = struct{}{}
			services = append(services, e.Service)
		}
	}
	sort.Strings(services)
	return services, nil
}

// ListTenants returns unique tenant IDs for a service from request headers.
func (m *MetricSource) ListTenants(_ context.Context, service string) ([]string, error) {
	entries := m.log.RecentFiltered(gateway.RequestFilter{Service: service})

	seen := make(map[string]struct{})
	var tenants []string
	for _, e := range entries {
		tid := e.RequestHeaders["X-Tenant-Id"]
		if tid == "" {
			continue
		}
		if _, ok := seen[tid]; !ok {
			seen[tid] = struct{}{}
			tenants = append(tenants, tid)
		}
	}
	sort.Strings(tenants)
	return tenants, nil
}

// computeMetrics derives latency percentiles, error rate, and request count
// from a slice of request entries.
func computeMetrics(entries []gateway.RequestEntry, service, action string) *regression.WindowMetrics {
	wm := &regression.WindowMetrics{
		Service: service,
		Action:  action,
	}

	n := len(entries)
	if n == 0 {
		return wm
	}

	wm.RequestCount = int64(n)

	// Collect latencies and compute percentiles.
	latencies := make([]float64, n)
	var errorCount int
	var totalRespSize float64
	for i, e := range entries {
		latencies[i] = e.LatencyMs
		if e.StatusCode >= 400 {
			errorCount++
		}
		totalRespSize += float64(len(e.ResponseBody))
	}

	sort.Float64s(latencies)
	wm.P50Ms = percentile(latencies, 0.50)
	wm.P95Ms = percentile(latencies, 0.95)
	wm.P99Ms = percentile(latencies, 0.99)
	wm.ErrorRate = float64(errorCount) / float64(n)
	wm.AvgRespSize = totalRespSize / float64(n)

	return wm
}

// enrichFromTraces adds cache miss rate and average span count from trace data.
func (m *MetricSource) enrichFromTraces(wm *regression.WindowMetrics, service string, window regression.TimeWindow) {
	if m.trace == nil {
		return
	}

	// Get recent traces for the service (no time filter on TraceStore.Recent,
	// so we filter manually).
	summaries := m.trace.Recent(service, nil, 0)
	if len(summaries) == 0 {
		return
	}

	var cacheMissCount, cacheTotal int
	var totalSpanCount int
	var traceCount int

	for _, s := range summaries {
		tc := m.trace.Get(s.TraceID)
		if tc == nil {
			continue
		}
		// Filter by time window.
		if tc.StartTime.Before(window.Start) || tc.StartTime.After(window.End) {
			continue
		}
		traceCount++

		// Cache miss from metadata.
		if cs, ok := tc.Metadata["x-cache-status"]; ok {
			cacheTotal++
			if cs == "MISS" {
				cacheMissCount++
			}
		}

		// Span count (1 for root + children).
		totalSpanCount += 1 + countChildren(tc)
	}

	if cacheTotal > 0 {
		wm.CacheMissRate = float64(cacheMissCount) / float64(cacheTotal)
	}
	if traceCount > 0 {
		wm.AvgSpanCount = float64(totalSpanCount) / float64(traceCount)
	}
}

// countChildren recursively counts all children of a trace context.
func countChildren(tc *gateway.TraceContext) int {
	count := len(tc.Children)
	for _, c := range tc.Children {
		count += countChildren(c)
	}
	return count
}

// percentile computes the p-th percentile from a sorted slice using nearest-rank.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := int(p * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

// Compile-time interface check.
var _ regression.MetricSource = (*MetricSource)(nil)
