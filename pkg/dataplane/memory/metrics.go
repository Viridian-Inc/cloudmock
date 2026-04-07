package memory

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

// MetricStore wraps gateway.RequestStats and RequestLog to satisfy the
// dataplane.MetricReader and dataplane.MetricWriter interfaces for local mode.
type MetricStore struct {
	stats *gateway.RequestStats
	log   *gateway.RequestLog

	mu      sync.Mutex
	samples map[string][]sample // key: "service" or "service:action"
}

type sample struct {
	latencyMs  float64
	statusCode int
	timestamp  time.Time
}

// NewMetricStore creates a MetricStore.
func NewMetricStore(stats *gateway.RequestStats, log *gateway.RequestLog) *MetricStore {
	return &MetricStore{
		stats:   stats,
		log:     log,
		samples: make(map[string][]sample),
	}
}

// Record records a metric sample and increments the request counter.
func (s *MetricStore) Record(_ context.Context, service, action string, latencyMs float64, statusCode int) error {
	if s.stats != nil {
		s.stats.Increment(service)
	}

	sm := sample{
		latencyMs:  latencyMs,
		statusCode: statusCode,
		timestamp:  time.Now(),
	}

	s.mu.Lock()
	s.samples[service] = append(s.samples[service], sm)
	if action != "" {
		s.samples[service+":"+action] = append(s.samples[service+":"+action], sm)
	}
	s.mu.Unlock()

	return nil
}

// ServiceStats returns aggregate metrics for a service over the given window.
func (s *MetricStore) ServiceStats(_ context.Context, service string, window time.Duration) (*dataplane.ServiceMetrics, error) {
	s.mu.Lock()
	raw := s.samples[service]
	s.mu.Unlock()

	cutoff := time.Now().Add(-window)
	var latencies []float64
	var errorCount int64
	var total int64

	for _, sm := range raw {
		if !sm.timestamp.Before(cutoff) {
			total++
			latencies = append(latencies, sm.latencyMs)
			if sm.statusCode >= 400 {
				errorCount++
			}
		}
	}

	// Fall back to RequestStats snapshot for request count if no samples recorded via Record.
	if total == 0 && s.stats != nil {
		snap := s.stats.Snapshot()
		if cnt, ok := snap[service]; ok {
			total = cnt
		}
	}

	var errorRate float64
	if total > 0 {
		errorRate = float64(errorCount) / float64(total)
	}

	p50, p95, p99 := percentiles(latencies)

	return &dataplane.ServiceMetrics{
		Service:      service,
		RequestCount: total,
		ErrorCount:   errorCount,
		ErrorRate:    errorRate,
		P50Ms:        p50,
		P95Ms:        p95,
		P99Ms:        p99,
	}, nil
}

// Percentiles returns latency percentiles for a service/action over the given window.
func (s *MetricStore) Percentiles(_ context.Context, service, action string, window time.Duration) (*dataplane.LatencyPercentiles, error) {
	key := service
	if action != "" {
		key = service + ":" + action
	}

	s.mu.Lock()
	raw := s.samples[key]
	s.mu.Unlock()

	cutoff := time.Now().Add(-window)
	var latencies []float64
	for _, sm := range raw {
		if !sm.timestamp.Before(cutoff) {
			latencies = append(latencies, sm.latencyMs)
		}
	}

	p50, p95, p99 := percentiles(latencies)
	return &dataplane.LatencyPercentiles{
		P50Ms: p50,
		P95Ms: p95,
		P99Ms: p99,
	}, nil
}

// percentiles computes P50, P95, P99 from a slice of values.
func percentiles(vals []float64) (p50, p95, p99 float64) {
	if len(vals) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	p50 = pctl(sorted, 50)
	p95 = pctl(sorted, 95)
	p99 = pctl(sorted, 99)
	return
}

func pctl(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// Compile-time interface checks.
var (
	_ dataplane.MetricReader = (*MetricStore)(nil)
	_ dataplane.MetricWriter = (*MetricStore)(nil)
)
