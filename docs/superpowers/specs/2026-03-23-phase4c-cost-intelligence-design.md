# Phase 4c: Cost Intelligence — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 4c of 6 (CloudMock Console — Intelligence Layer, sub-project 3 of 5)
**Depends on:** Phase 3 (Production Data Plane)

---

## Overview

A cost estimation engine that computes per-request costs using configurable AWS pricing rates and provides aggregation by service, route, tenant, and time. Extends the existing `/api/cost` endpoint with route-level, tenant-level, and trend breakdowns.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Architecture | Standalone `pkg/cost/` engine | Separates pricing logic from HTTP handlers; reusable |
| Trend computation | Query-time aggregation | ClickHouse handles GROUP BY time buckets; no precomputed rollups needed |
| Pricing model | Configurable in cloudmock.yml with defaults | Different regions/pricing tiers change rates |

---

## 1. Pricing Config

```go
type PricingConfig struct {
    Lambda       LambdaPricing   `yaml:"lambda" json:"lambda"`
    DynamoDB     DynamoDBPricing `yaml:"dynamodb" json:"dynamodb"`
    S3           S3Pricing       `yaml:"s3" json:"s3"`
    SQS          SQSPricing      `yaml:"sqs" json:"sqs"`
    DataTransfer TransferPricing `yaml:"data_transfer" json:"data_transfer"`
}

type LambdaPricing struct {
    PerGBSecond    float64 `yaml:"per_gb_second" json:"per_gb_second"`
    DefaultMemoryMB int    `yaml:"default_memory_mb" json:"default_memory_mb"`
}

type DynamoDBPricing struct {
    PerRCU float64 `yaml:"per_rcu" json:"per_rcu"`
    PerWCU float64 `yaml:"per_wcu" json:"per_wcu"`
}

type S3Pricing struct {
    PerGET float64 `yaml:"per_get" json:"per_get"`
    PerPUT float64 `yaml:"per_put" json:"per_put"`
}

type SQSPricing struct {
    PerRequest float64 `yaml:"per_request" json:"per_request"`
}

type TransferPricing struct {
    PerKB float64 `yaml:"per_kb" json:"per_kb"`
}
```

Defaults: Lambda $0.0000166667/GB-sec (128MB), DynamoDB $0.00000025/RCU $0.00000125/WCU, S3 $0.0000004/GET $0.000005/PUT, SQS $0.0000004/req, transfer $0.00000009/KB.

---

## 2. Cost Engine

```go
type Engine struct {
    requests dataplane.RequestReader
    pricing  PricingConfig
}

func New(requests dataplane.RequestReader, pricing PricingConfig) *Engine
func (e *Engine) RequestCost(entry dataplane.RequestEntry) float64
func (e *Engine) ByService(ctx context.Context) ([]ServiceCost, error)
func (e *Engine) ByRoute(ctx context.Context, limit int) ([]RouteCost, error)
func (e *Engine) ByTenant(ctx context.Context, limit int) ([]TenantCost, error)
func (e *Engine) Trend(ctx context.Context, window, bucketSize time.Duration) ([]TimeBucket, error)
```

`RequestCost` maps service type to pricing formula:
- Service contains "lambda" → `durationMs / 1000 * memoryMB / 1024 * perGBSecond`
- Service contains "dynamodb" → GET/Query = RCU, PUT/Update = WCU
- Service contains "s3" → GET vs PUT
- Service contains "sqs" → per-request
- All requests → + response body size * transfer rate

Aggregation methods query `RequestReader.Query()` with appropriate filters, compute `RequestCost` per entry, group and sum.

---

## 3. Result Types

```go
type ServiceCost struct {
    Service      string  `json:"service"`
    RequestCount int64   `json:"request_count"`
    TotalCost    float64 `json:"total_cost"`
    AvgCost      float64 `json:"avg_cost"`
}

type RouteCost struct {
    Service      string  `json:"service"`
    Method       string  `json:"method"`
    Path         string  `json:"path"`
    RequestCount int64   `json:"request_count"`
    TotalCost    float64 `json:"total_cost"`
    AvgCost      float64 `json:"avg_cost"`
}

type TenantCost struct {
    TenantID     string  `json:"tenant_id"`
    RequestCount int64   `json:"request_count"`
    TotalCost    float64 `json:"total_cost"`
    AvgCost      float64 `json:"avg_cost"`
}

type TimeBucket struct {
    Start        time.Time `json:"start"`
    End          time.Time `json:"end"`
    TotalCost    float64   `json:"total_cost"`
    RequestCount int64     `json:"request_count"`
}
```

---

## 4. API Endpoints

```
GET /api/cost/routes?limit=20            — top routes by total cost
GET /api/cost/tenants?limit=20           — top tenants by total cost
GET /api/cost/trend?window=7d&bucket=1h  — cost over time buckets
```

Existing `GET /api/cost` stays as-is.

---

## 5. Configuration

```yaml
cost:
  pricing:
    lambda:
      per_gb_second: 0.0000166667
      default_memory_mb: 128
    dynamodb:
      per_rcu: 0.00000025
      per_wcu: 0.00000125
    s3:
      per_get: 0.0000004
      per_put: 0.000005
    sqs:
      per_request: 0.0000004
    data_transfer:
      per_kb: 0.00000009
```

---

## 6. File Layout

```
pkg/cost/
├── types.go        # PricingConfig, ServiceCost, RouteCost, TenantCost, TimeBucket
├── engine.go       # Engine, RequestCost, ByService/ByRoute/ByTenant/Trend
└── engine_test.go  # unit tests with mock RequestReader
```

**Files modified:**
- `pkg/config/config.go` — add `Cost CostConfig` with `Pricing PricingConfig`
- `pkg/admin/api.go` — add 3 cost endpoints, wire engine
- `cmd/gateway/main.go` — create cost engine, pass to admin API
