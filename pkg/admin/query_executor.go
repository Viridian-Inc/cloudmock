package admin

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

// ExecuteQuery runs a MetricQuery against the request log in either timeseries
// or scalar mode. The minutes parameter controls the lookback window. For
// timeseries mode, bucketDur controls bucket width (defaults to 1m).
func ExecuteQuery(log *gateway.RequestLog, q MetricQuery, mode string, minutes int, bucketDur time.Duration) QueryResult {
	if minutes <= 0 {
		minutes = 15
	}
	if bucketDur <= 0 {
		bucketDur = time.Minute
	}

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	entries := filterEntries(log.Recent("", 10000), cutoff, q.Filters)

	result := QueryResult{
		Query: formatQuery(q),
		Mode:  mode,
	}

	if mode == "timeseries" {
		result.Buckets = computeTimeseries(entries, cutoff, bucketDur, q)
	} else {
		v := computeScalar(entries, q)
		result.Scalar = &v
	}

	return result
}

// filterEntries selects request entries that match the time cutoff and all
// MetricQuery filters (service, action, method, status).
func filterEntries(entries []gateway.RequestEntry, cutoff time.Time, filters map[string]string) []gateway.RequestEntry {
	var result []gateway.RequestEntry
	for _, e := range entries {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.Service == "" {
			continue
		}
		if v, ok := filters["service"]; ok && e.Service != v {
			continue
		}
		if v, ok := filters["action"]; ok && e.Action != v {
			continue
		}
		if v, ok := filters["method"]; ok && e.Method != v {
			continue
		}
		if v, ok := filters["status"]; ok {
			if strconv.Itoa(e.StatusCode) != v {
				continue
			}
		}
		result = append(result, e)
	}
	return result
}

// computeScalar reduces all matched entries into a single numeric value.
func computeScalar(entries []gateway.RequestEntry, q MetricQuery) float64 {
	values := extractValues(entries, q.Metric)
	return aggregate(values, q.Aggregation)
}

// computeTimeseries buckets entries by time and computes the aggregation per bucket.
func computeTimeseries(entries []gateway.RequestEntry, cutoff time.Time, bucketDur time.Duration, q MetricQuery) []QueryBucket {
	now := time.Now()
	startBucket := cutoff.Truncate(bucketDur)
	numBuckets := int(now.Sub(startBucket)/bucketDur) + 1

	// Collect values per bucket.
	bucketValues := make([][]float64, numBuckets)
	for _, e := range entries {
		idx := int(e.Timestamp.Sub(startBucket) / bucketDur)
		if idx < 0 || idx >= numBuckets {
			continue
		}
		bucketValues[idx] = append(bucketValues[idx], extractValue(e, q.Metric))
	}

	result := make([]QueryBucket, numBuckets)
	for i := range result {
		result[i] = QueryBucket{
			Timestamp: startBucket.Add(time.Duration(i) * bucketDur),
			Value:     aggregate(bucketValues[i], q.Aggregation),
		}
	}
	return result
}

// extractValues extracts the relevant float64 slice from entries for a given metric.
func extractValues(entries []gateway.RequestEntry, metric string) []float64 {
	values := make([]float64, 0, len(entries))
	for _, e := range entries {
		values = append(values, extractValue(e, metric))
	}
	return values
}

// extractValue extracts a single float64 from an entry for a given metric name.
func extractValue(e gateway.RequestEntry, metric string) float64 {
	switch metric {
	case "latency_ms":
		return e.LatencyMs
	case "request_count":
		return 1
	case "error_count":
		if e.StatusCode >= 400 {
			return 1
		}
		return 0
	case "error_rate":
		if e.StatusCode >= 400 {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// aggregate computes the final value from a slice using the given aggregation.
func aggregate(values []float64, agg string) float64 {
	if len(values) == 0 {
		return 0
	}

	switch agg {
	case "count":
		return float64(len(values))
	case "sum":
		var s float64
		for _, v := range values {
			s += v
		}
		return round2(s)
	case "avg":
		var s float64
		for _, v := range values {
			s += v
		}
		return round2(s / float64(len(values)))
	case "max":
		m := values[0]
		for _, v := range values[1:] {
			if v > m {
				m = v
			}
		}
		return round2(m)
	case "min":
		m := values[0]
		for _, v := range values[1:] {
			if v < m {
				m = v
			}
		}
		return round2(m)
	case "p50":
		return percentileSorted(values, 50)
	case "p95":
		return percentileSorted(values, 95)
	case "p99":
		return percentileSorted(values, 99)
	default:
		return 0
	}
}

// percentileSorted sorts the slice and delegates to the existing percentile function.
func percentileSorted(values []float64, p float64) float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	return percentile(sorted, p)
}

// round2 rounds to 2 decimal places.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// formatQuery reconstructs a query string from a MetricQuery.
func formatQuery(q MetricQuery) string {
	s := q.Aggregation + ":" + q.Metric
	if len(q.Filters) > 0 {
		s += "{"
		first := true
		// Sort keys for deterministic output.
		keys := make([]string, 0, len(q.Filters))
		for k := range q.Filters {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !first {
				s += ","
			}
			s += k + ":" + q.Filters[k]
			first = false
		}
		s += "}"
	}
	return s
}
