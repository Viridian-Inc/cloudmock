package monitor

import (
	"context"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

// GatewayProvider reads live metrics from the gateway SLO engine and request
// stats to produce ServiceSnapshots for the evaluator.
type GatewayProvider struct {
	slo   *gateway.SLOEngine
	stats *gateway.RequestStats
}

// NewGatewayProvider creates a MetricsProvider backed by gateway components.
func NewGatewayProvider(slo *gateway.SLOEngine, stats *gateway.RequestStats) *GatewayProvider {
	return &GatewayProvider{slo: slo, stats: stats}
}

// Snapshot collects the current metrics from the SLO engine windows and request stats.
func (g *GatewayProvider) Snapshot(_ context.Context) ([]ServiceSnapshot, error) {
	// Aggregate SLO window data by service.
	sloStatus := g.slo.Status()
	serviceMap := make(map[string]*ServiceSnapshot)

	for _, w := range sloStatus.Windows {
		snap, ok := serviceMap[w.Service]
		if !ok {
			snap = &ServiceSnapshot{Service: w.Service}
			serviceMap[w.Service] = snap
		}

		// Use the worst (highest) error rate seen across actions.
		if w.ErrorRate > snap.ErrorRate {
			snap.ErrorRate = w.ErrorRate
		}
		// Use the worst (highest) latency across actions.
		if w.P50Ms > snap.LatencyP50 {
			snap.LatencyP50 = w.P50Ms
		}
		if w.P95Ms > snap.LatencyP95 {
			snap.LatencyP95 = w.P95Ms
		}
		if w.P99Ms > snap.LatencyP99 {
			snap.LatencyP99 = w.P99Ms
		}
	}

	// Enrich with request-rate data from RequestStats.
	if g.stats != nil {
		counts := g.stats.Snapshot()
		for svc, count := range counts {
			snap, ok := serviceMap[svc]
			if !ok {
				snap = &ServiceSnapshot{Service: svc}
				serviceMap[svc] = snap
			}
			// RequestStats.Snapshot() returns cumulative counts.
			// Throughput is approximate: count is total, not per-second.
			// A more accurate approach would track deltas, but this provides
			// a useful baseline for threshold monitors.
			snap.Throughput = float64(count)
		}
	}

	results := make([]ServiceSnapshot, 0, len(serviceMap))
	for _, snap := range serviceMap {
		results = append(results, *snap)
	}
	return results, nil
}

// Compile-time interface check.
var _ MetricsProvider = (*GatewayProvider)(nil)
