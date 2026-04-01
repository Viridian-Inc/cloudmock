package admin

import (
	"fmt"
	"strings"
)

// Supported aggregation functions.
var validAggregations = map[string]bool{
	"avg": true, "sum": true, "max": true, "min": true,
	"count": true, "p50": true, "p95": true, "p99": true,
}

// Supported metric names.
var validMetrics = map[string]bool{
	"latency_ms":    true,
	"request_count": true,
	"error_count":   true,
	"error_rate":    true,
}

// Supported filter keys.
var validFilters = map[string]bool{
	"service": true, "action": true, "method": true, "status": true,
}

// ParseMetricQuery parses a DSL string like "avg:latency_ms{service:dynamodb,method:POST}"
// into a MetricQuery. The grammar is:
//
//	<aggregation>:<metric>{<key>:<value>,...}
//
// The filter block "{...}" is optional.
func ParseMetricQuery(raw string) (MetricQuery, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return MetricQuery{}, fmt.Errorf("empty query")
	}

	// Split aggregation from the rest at the first colon.
	colonIdx := strings.Index(raw, ":")
	if colonIdx < 0 {
		return MetricQuery{}, fmt.Errorf("invalid query %q: missing aggregation separator ':'", raw)
	}

	agg := raw[:colonIdx]
	rest := raw[colonIdx+1:]

	if !validAggregations[agg] {
		return MetricQuery{}, fmt.Errorf("unsupported aggregation %q", agg)
	}

	// Separate the metric name from the optional filter block.
	var metricName, filterBlock string
	if braceIdx := strings.Index(rest, "{"); braceIdx >= 0 {
		metricName = rest[:braceIdx]
		if !strings.HasSuffix(rest, "}") {
			return MetricQuery{}, fmt.Errorf("invalid query %q: unclosed filter block", raw)
		}
		filterBlock = rest[braceIdx+1 : len(rest)-1]
	} else {
		metricName = rest
	}

	if !validMetrics[metricName] {
		return MetricQuery{}, fmt.Errorf("unsupported metric %q", metricName)
	}

	filters := map[string]string{}
	if filterBlock != "" {
		pairs := strings.Split(filterBlock, ",")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
				return MetricQuery{}, fmt.Errorf("invalid filter %q in query %q", pair, raw)
			}
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			if !validFilters[key] {
				return MetricQuery{}, fmt.Errorf("unsupported filter key %q", key)
			}
			filters[key] = val
		}
	}

	return MetricQuery{
		Aggregation: agg,
		Metric:      metricName,
		Filters:     filters,
	}, nil
}
