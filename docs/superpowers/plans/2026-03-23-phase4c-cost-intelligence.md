# Phase 4c: Cost Intelligence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-route, per-tenant, and trend cost breakdowns with configurable AWS pricing rates.

**Architecture:** A `CostEngine` in `pkg/cost/` computes per-request costs using configurable pricing, then aggregates by service, route, tenant, or time bucket via the `RequestReader` interface.

**Tech Stack:** Go 1.26, existing `dataplane.RequestReader` interface

---

## File Structure

```
pkg/cost/
├── types.go        # PricingConfig (all sub-configs), result types
├── engine.go       # Engine, RequestCost, ByService/ByRoute/ByTenant/Trend
└── engine_test.go  # unit tests with mock RequestReader
```

**Files modified:**
- `pkg/config/config.go` — add CostConfig
- `pkg/admin/api.go` — add 3 cost endpoints + SetCostEngine
- `cmd/gateway/main.go` — create engine, wire

---

## Task 1: Types & Cost Engine

**Files:**
- Create: `pkg/cost/types.go`
- Create: `pkg/cost/engine.go`
- Create: `pkg/cost/engine_test.go`

- [ ] **Step 1:** `mkdir -p pkg/cost`

- [ ] **Step 2: Write types.go** — PricingConfig with all sub-configs (LambdaPricing, DynamoDBPricing, S3Pricing, SQSPricing, TransferPricing), result types (ServiceCost, RouteCost, TenantCost, TimeBucket), `DefaultPricingConfig()` returning sensible defaults.

- [ ] **Step 3: Write engine tests**

Test with a mock RequestReader returning canned requests for different services:

```go
func TestRequestCost_Lambda(t *testing.T) {
    // 100ms Lambda at 128MB → specific cost
}
func TestRequestCost_DynamoDB_Read(t *testing.T) {
    // GET request to dynamodb → RCU pricing
}
func TestRequestCost_DynamoDB_Write(t *testing.T) {
    // PUT request to dynamodb → WCU pricing
}
func TestRequestCost_S3(t *testing.T) {
    // GET and PUT to s3 → different rates
}
func TestRequestCost_SQS(t *testing.T) {
    // SQS request → per-request pricing
}
func TestRequestCost_DataTransfer(t *testing.T) {
    // Any request with response body → transfer cost added
}
func TestRequestCost_CustomPricing(t *testing.T) {
    // Override pricing config → different costs
}
func TestByRoute(t *testing.T) {
    // Multiple requests to different routes → grouped, sorted by total cost
}
func TestByTenant(t *testing.T) {
    // Requests with different tenant_ids → grouped by tenant
}
func TestTrend(t *testing.T) {
    // Requests across time → correct hourly buckets
}
func TestByService(t *testing.T) {
    // Multiple services → grouped, totaled
}
```

- [ ] **Step 4: Run tests — verify FAIL**

Run: `go test ./pkg/cost/ -v`

- [ ] **Step 5: Write engine.go**

`RequestCost(entry)` — match service name (case-insensitive contains) to pricing formula. Add data transfer cost from `len(entry.ResponseBody)`. Return total.

`ByService(ctx)` — `RequestReader.Query(ctx, RequestFilter{Limit: 10000})`, group by Service, sum costs, sort by TotalCost desc.

`ByRoute(ctx, limit)` — same query, group by `Service:Method:Path`, sort by TotalCost desc, apply limit.

`ByTenant(ctx, limit)` — same query, group by TenantID (from entry.TenantID or request headers), sort by TotalCost desc, apply limit. Skip entries with empty tenant.

`Trend(ctx, window, bucketSize)` — query with `From: now-window`, group by time bucket (truncate timestamp to bucketSize), sum costs per bucket.

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/cost/ -v -cover`
Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/cost/
git commit -m "feat(cost): add cost engine with configurable pricing and aggregations

Per-request cost estimation for Lambda/DynamoDB/S3/SQS with
configurable rates. Aggregation by service, route, tenant, and
time bucket via RequestReader interface."
```

---

## Task 2: Config & API Wiring

**Files:**
- Modify: `pkg/config/config.go` — add CostConfig
- Modify: `pkg/admin/api.go` — add endpoints + setter
- Modify: `cmd/gateway/main.go` — wire engine

- [ ] **Step 1: Add CostConfig to config**

```go
type CostConfig struct {
    Pricing cost.PricingConfig `yaml:"pricing" json:"pricing"`
}
```

Add `Cost CostConfig` to Config struct. In `Default()`, set `Pricing: cost.DefaultPricingConfig()`.

Note: Import `pkg/cost` from config. If circular import, define PricingConfig in config package instead.

- [ ] **Step 2: Add cost API endpoints**

In `pkg/admin/api.go`:
- Add `costEngine *cost.Engine` field
- Add `SetCostEngine(engine *cost.Engine)` setter
- Add `handleCostRoutes` — `GET /api/cost/routes?limit=20`
- Add `handleCostTenants` — `GET /api/cost/tenants?limit=20`
- Add `handleCostTrend` — `GET /api/cost/trend?window=7d&bucket=1h`
- Register routes in `NewWithDataPlane()`
- Parse `window` and `bucket` as duration strings (e.g., "7d" → 7*24h, "1h" → 1h)

- [ ] **Step 3: Wire in main.go**

```go
costEngine := cost.New(dp.Requests, cfg.Cost.Pricing)
adminAPI.SetCostEngine(costEngine)
```

- [ ] **Step 4: Run all tests**

Run: `go test -short ./pkg/cost/ ./pkg/admin/ ./pkg/config/ -v`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/admin/api.go cmd/gateway/main.go
git commit -m "feat(cost): wire cost engine with 3 new API endpoints

/api/cost/routes, /api/cost/tenants, /api/cost/trend with configurable
pricing in cloudmock.yml."
```

---

## Task Summary

| Task | What it builds | Depends on |
|------|---------------|------------|
| 1 | Types + Engine + tests | — |
| 2 | Config + API + wiring | 1 |
