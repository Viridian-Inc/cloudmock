package cost

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// Engine computes costs for recorded requests using configurable pricing.
type Engine struct {
	requests dataplane.RequestReader
	pricing  PricingConfig
}

// New creates a cost Engine with the given request reader and pricing config.
func New(requests dataplane.RequestReader, pricing PricingConfig) *Engine {
	return &Engine{
		requests: requests,
		pricing:  pricing,
	}
}

// RequestCost computes the estimated cost of a single request entry.
func (e *Engine) RequestCost(entry dataplane.RequestEntry) float64 {
	var cost float64
	service := strings.ToLower(entry.Service)

	switch {
	case strings.Contains(service, "lambda"):
		durationSec := entry.LatencyMs / 1000.0
		memGB := e.pricing.Lambda.DefaultMemoryMB / 1024.0
		cost = durationSec * memGB * e.pricing.Lambda.PerGBSecond

	case strings.Contains(service, "dynamodb"):
		method := strings.ToUpper(entry.Method)
		switch method {
		case "GET", "QUERY":
			cost = e.pricing.DynamoDB.PerRCU
		case "PUT", "UPDATE", "DELETE":
			cost = e.pricing.DynamoDB.PerWCU
		}

	case strings.Contains(service, "s3"):
		method := strings.ToUpper(entry.Method)
		switch method {
		case "GET", "HEAD":
			cost = e.pricing.S3.PerGET
		case "PUT", "POST":
			cost = e.pricing.S3.PerPUT
		}

	case strings.Contains(service, "sqs"):
		cost = e.pricing.SQS.PerRequest
	}

	// Data transfer cost based on response body size.
	if len(entry.ResponseBody) > 0 {
		transferKB := float64(len(entry.ResponseBody)) / 1024.0
		cost += transferKB * e.pricing.DataTransfer.PerKB
	}

	return cost
}

// ByService returns costs aggregated by service, sorted by TotalCost descending.
func (e *Engine) ByService(ctx context.Context) ([]ServiceCost, error) {
	entries, err := e.requests.Query(ctx, dataplane.RequestFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}

	type agg struct {
		count int64
		total float64
	}
	m := make(map[string]*agg)

	for _, entry := range entries {
		c := e.RequestCost(entry)
		a, ok := m[entry.Service]
		if !ok {
			a = &agg{}
			m[entry.Service] = a
		}
		a.count++
		a.total += c
	}

	result := make([]ServiceCost, 0, len(m))
	for svc, a := range m {
		result = append(result, ServiceCost{
			Service:      svc,
			RequestCount: a.count,
			TotalCost:    a.total,
			AvgCost:      a.total / float64(a.count),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCost > result[j].TotalCost
	})

	return result, nil
}

// ByRoute returns costs aggregated by service+method+path, sorted by TotalCost descending.
func (e *Engine) ByRoute(ctx context.Context, limit int) ([]RouteCost, error) {
	entries, err := e.requests.Query(ctx, dataplane.RequestFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}

	type routeKey struct {
		service, method, path string
	}
	type agg struct {
		key   routeKey
		count int64
		total float64
	}
	m := make(map[routeKey]*agg)

	for _, entry := range entries {
		c := e.RequestCost(entry)
		k := routeKey{entry.Service, entry.Method, entry.Path}
		a, ok := m[k]
		if !ok {
			a = &agg{key: k}
			m[k] = a
		}
		a.count++
		a.total += c
	}

	result := make([]RouteCost, 0, len(m))
	for _, a := range m {
		result = append(result, RouteCost{
			Service:      a.key.service,
			Method:       a.key.method,
			Path:         a.key.path,
			RequestCount: a.count,
			TotalCost:    a.total,
			AvgCost:      a.total / float64(a.count),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCost > result[j].TotalCost
	})

	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// ByTenant returns costs aggregated by tenant ID, sorted by TotalCost descending.
// Entries with empty TenantID are skipped.
func (e *Engine) ByTenant(ctx context.Context, limit int) ([]TenantCost, error) {
	entries, err := e.requests.Query(ctx, dataplane.RequestFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}

	type agg struct {
		count int64
		total float64
	}
	m := make(map[string]*agg)

	for _, entry := range entries {
		if entry.TenantID == "" {
			continue
		}
		c := e.RequestCost(entry)
		a, ok := m[entry.TenantID]
		if !ok {
			a = &agg{}
			m[entry.TenantID] = a
		}
		a.count++
		a.total += c
	}

	result := make([]TenantCost, 0, len(m))
	for tid, a := range m {
		result = append(result, TenantCost{
			TenantID:     tid,
			RequestCount: a.count,
			TotalCost:    a.total,
			AvgCost:      a.total / float64(a.count),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCost > result[j].TotalCost
	})

	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, nil
}

// Trend returns cost aggregated into time buckets over the given window.
func (e *Engine) Trend(ctx context.Context, window, bucketSize time.Duration) ([]TimeBucket, error) {
	from := time.Now().Add(-window)
	entries, err := e.requests.Query(ctx, dataplane.RequestFilter{
		Limit: 10000,
		From:  from,
	})
	if err != nil {
		return nil, err
	}

	type agg struct {
		cost  float64
		count int64
	}
	m := make(map[time.Time]*agg)

	for _, entry := range entries {
		c := e.RequestCost(entry)
		bucketStart := entry.Timestamp.Truncate(bucketSize)
		a, ok := m[bucketStart]
		if !ok {
			a = &agg{}
			m[bucketStart] = a
		}
		a.cost += c
		a.count++
	}

	result := make([]TimeBucket, 0, len(m))
	for start, a := range m {
		result = append(result, TimeBucket{
			Start:        start,
			End:          start.Add(bucketSize),
			TotalCost:    a.cost,
			RequestCount: a.count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Start.Before(result[j].Start)
	})

	return result, nil
}
