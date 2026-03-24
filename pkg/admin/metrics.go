package admin

import (
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/gateway"
)

// ServiceMetrics holds computed latency percentiles and error rate for a single service.
type ServiceMetrics struct {
	Service    string  `json:"service"`
	P50Ms      float64 `json:"p50ms"`
	P95Ms      float64 `json:"p95ms"`
	P99Ms      float64 `json:"p99ms"`
	AvgMs      float64 `json:"avgMs"`
	ErrorRate  float64 `json:"errorRate"`
	TotalCalls int     `json:"totalCalls"`
	ErrorCalls int     `json:"errorCalls"`
}

// MetricsTimeBucket is a single time bucket for the timeline endpoint.
type MetricsTimeBucket struct {
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]*BucketMetric `json:"services"`
}

// BucketMetric holds per-service stats within a time bucket.
type BucketMetric struct {
	Calls    int     `json:"calls"`
	AvgMs    float64 `json:"avgMs"`
	Errors   int     `json:"errors"`
	// latencies is used internally for percentile computation on timeline buckets.
	latencies []float64
	P50Ms     float64 `json:"p50ms"`
	P95Ms     float64 `json:"p95ms"`
	P99Ms     float64 `json:"p99ms"`
}

// handleMetrics serves GET /api/metrics — per-service latency percentiles computed
// from the last 15 minutes of request log entries.
func (a *API) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	minutes := 15
	if m := r.URL.Query().Get("minutes"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 {
			minutes = n
		}
	}

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	entries := a.log.Recent("", 10000)

	// Group entries by service.
	type svcData struct {
		latencies []float64
		errors    int
	}
	byService := map[string]*svcData{}

	for _, e := range entries {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.Service == "" {
			continue
		}
		sd, ok := byService[e.Service]
		if !ok {
			sd = &svcData{}
			byService[e.Service] = sd
		}
		sd.latencies = append(sd.latencies, e.LatencyMs)
		if e.StatusCode >= 400 {
			sd.errors++
		}
	}

	result := make([]ServiceMetrics, 0, len(byService))
	for svc, sd := range byService {
		sort.Float64s(sd.latencies)
		total := len(sd.latencies)
		var sum float64
		for _, v := range sd.latencies {
			sum += v
		}
		errRate := 0.0
		if total > 0 {
			errRate = float64(sd.errors) / float64(total)
		}
		result = append(result, ServiceMetrics{
			Service:    svc,
			P50Ms:      percentile(sd.latencies, 50),
			P95Ms:      percentile(sd.latencies, 95),
			P99Ms:      percentile(sd.latencies, 99),
			AvgMs:      math.Round(sum/float64(total)*100) / 100,
			ErrorRate:  math.Round(errRate*10000) / 10000,
			TotalCalls: total,
			ErrorCalls: sd.errors,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCalls > result[j].TotalCalls
	})

	writeJSON(w, http.StatusOK, result)
}

// handleMetricsTimeline serves GET /api/metrics/timeline?minutes=15&bucket=1m
// Returns time-bucketed metrics for charting.
func (a *API) handleMetricsTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	minutes := 15
	if m := r.URL.Query().Get("minutes"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n > 0 {
			minutes = n
		}
	}

	bucketDur := time.Minute
	if b := r.URL.Query().Get("bucket"); b != "" {
		bucketDur = parseBucketDuration(b)
	}

	now := time.Now()
	cutoff := now.Add(-time.Duration(minutes) * time.Minute)

	// Align start to bucket boundaries.
	startBucket := cutoff.Truncate(bucketDur)
	numBuckets := int(now.Sub(startBucket)/bucketDur) + 1

	// Pre-create buckets.
	buckets := make([]MetricsTimeBucket, numBuckets)
	for i := range buckets {
		buckets[i] = MetricsTimeBucket{
			Timestamp: startBucket.Add(time.Duration(i) * bucketDur),
			Services:  map[string]*BucketMetric{},
		}
	}

	entries := a.log.Recent("", 10000)
	for _, e := range entries {
		if e.Timestamp.Before(cutoff) || e.Service == "" {
			continue
		}
		idx := int(e.Timestamp.Sub(startBucket) / bucketDur)
		if idx < 0 || idx >= numBuckets {
			continue
		}
		bm, ok := buckets[idx].Services[e.Service]
		if !ok {
			bm = &BucketMetric{}
			buckets[idx].Services[e.Service] = bm
		}
		bm.Calls++
		bm.latencies = append(bm.latencies, e.LatencyMs)
		if e.StatusCode >= 400 {
			bm.Errors++
		}
	}

	// Compute averages and percentiles per bucket per service.
	for i := range buckets {
		for _, bm := range buckets[i].Services {
			if bm.Calls > 0 {
				var sum float64
				for _, v := range bm.latencies {
					sum += v
				}
				bm.AvgMs = math.Round(sum/float64(bm.Calls)*100) / 100
				sort.Float64s(bm.latencies)
				bm.P50Ms = percentile(bm.latencies, 50)
				bm.P95Ms = percentile(bm.latencies, 95)
				bm.P99Ms = percentile(bm.latencies, 99)
			}
			bm.latencies = nil // clear internal field before JSON serialization
		}
	}

	writeJSON(w, http.StatusOK, buckets)
}

// AllSince returns all entries newer than the given cutoff. This is a helper
// that leverages the existing Recent method on RequestLog.
func allSince(log *gateway.RequestLog, cutoff time.Time) []gateway.RequestEntry {
	all := log.Recent("", 10000)
	var result []gateway.RequestEntry
	for _, e := range all {
		if !e.Timestamp.Before(cutoff) {
			result = append(result, e)
		}
	}
	return result
}

// percentile returns the p-th percentile of sorted data.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return math.Round(sorted[0]*100) / 100
	}
	rank := p / 100 * float64(n-1)
	lower := int(math.Floor(rank))
	upper := lower + 1
	if upper >= n {
		upper = n - 1
	}
	weight := rank - float64(lower)
	val := sorted[lower]*(1-weight) + sorted[upper]*weight
	return math.Round(val*100) / 100
}

// parseBucketDuration parses a bucket duration string like "1m", "5m", "30s".
func parseBucketDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Minute
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		// Try appending common suffixes.
		if d2, err2 := time.ParseDuration(s + "m"); err2 == nil {
			return d2
		}
		return time.Minute
	}
	if d < time.Second {
		return time.Second
	}
	return d
}
