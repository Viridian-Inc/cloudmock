# Phase 4a: Regression Detection Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a regression detection engine that identifies performance and reliability degradations through deploy-triggered analysis and continuous periodic scanning, with severity + confidence classification.

**Architecture:** Six pure detection algorithms compare before/after `WindowMetrics` sourced from Prometheus (latency, error rate) and ClickHouse (cache miss, fanout, payload size) via a `MetricSource` interface. Deploy-triggered detection runs on `POST /api/deploys` with re-evaluation at 1m/5m/15m. Continuous scan runs every 5m. Results stored in PostgreSQL, exposed via `/api/regressions`.

**Tech Stack:** Go 1.26, Prometheus recording rules, ClickHouse queries, PostgreSQL, testcontainers-go

---

## File Structure

```
pkg/regression/
├── types.go              # Regression, WindowMetrics, Severity, AlgorithmType, configs
├── algorithms.go         # 6 pure detection functions + confidence/severity helpers
├── algorithms_test.go
├── source.go             # MetricSource interface
├── source_impl.go        # Production MetricSource (Prometheus + ClickHouse)
├── source_impl_test.go
├── engine.go             # Engine struct, deploy-triggered + continuous scan, auto-resolution
├── engine_test.go
├── store.go              # RegressionStore interface + RegressionFilter
├── postgres/
│   ├── store.go          # PostgreSQL RegressionStore
│   └── store_test.go
└── memory/
    ├── store.go          # In-memory RegressionStore
    ├── store_test.go
    └── source.go         # In-memory MetricSource (from RequestLog + TraceStore)
```

**Files modified:**
- `pkg/config/config.go:108-122` — add `Regression RegressionConfig` to Config struct
- `cmd/gateway/main.go:303-312` — create and start regression engine
- `pkg/admin/api.go:58-78` — add regression engine field, new API handlers
- `pkg/admin/api.go:1619-1640` — wire deploy-triggered detection in POST /api/deploys
- `docker/config/recording-rules.yml:16` — add regression detection recording rules
- `docker/init/postgres/` — new `02-regression-schema.sql`

---

## Task 1: Types & Interfaces

**Files:**
- Create: `pkg/regression/types.go`
- Create: `pkg/regression/source.go`
- Create: `pkg/regression/store.go`

- [ ] **Step 1: Create package directory**

Run: `mkdir -p pkg/regression/postgres pkg/regression/memory`

- [ ] **Step 2: Write types.go**

All data types, enums, and config structs from the design spec: `Severity`, `AlgorithmType`, `Regression`, `TimeWindow`, `WindowMetrics`, `AlgorithmConfig` with all 6 sub-configs (`LatencyConfig`, `ErrorConfig`, `OutlierConfig`, `CacheMissConfig`, `FanoutConfig`, `PayloadConfig`). Include `DefaultAlgorithmConfig()` that returns sensible defaults.

- [ ] **Step 3: Write source.go**

`MetricSource` interface:
```go
type MetricSource interface {
    WindowMetrics(ctx context.Context, service, action string, window TimeWindow) (*WindowMetrics, error)
    TenantWindowMetrics(ctx context.Context, service, tenantID string, window TimeWindow) (*WindowMetrics, error)
    FleetWindowMetrics(ctx context.Context, service string, window TimeWindow) (*WindowMetrics, error)
    ListServices(ctx context.Context) ([]string, error)
    ListTenants(ctx context.Context, service string) ([]string, error)
}
```

- [ ] **Step 4: Write store.go**

`RegressionStore` interface + `RegressionFilter`:
```go
type RegressionStore interface {
    Save(ctx context.Context, r *Regression) error
    List(ctx context.Context, filter RegressionFilter) ([]Regression, error)
    Get(ctx context.Context, id string) (*Regression, error)
    UpdateStatus(ctx context.Context, id string, status string) error
    ActiveForDeploy(ctx context.Context, deployID string) ([]Regression, error)
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./pkg/regression/...`

- [ ] **Step 6: Commit**

```bash
git add pkg/regression/
git commit -m "feat(regression): add types, MetricSource, and RegressionStore interfaces"
```

---

## Task 2: Detection Algorithms

**Files:**
- Create: `pkg/regression/algorithms.go`
- Create: `pkg/regression/algorithms_test.go`

- [ ] **Step 1: Write algorithm tests**

Test each of the 6 algorithms as pure functions with crafted `WindowMetrics` pairs:

```go
func TestDetectLatencyRegression(t *testing.T) {
    cfg := LatencyConfig{P99ThresholdPercent: 50, MinSampleSize: 100}

    t.Run("no regression below threshold", func(t *testing.T) {
        before := &WindowMetrics{P99Ms: 100, RequestCount: 200}
        after := &WindowMetrics{P99Ms: 140, RequestCount: 200}  // +40%, below 50%
        r := detectLatencyRegression(before, after, cfg)
        if r != nil { t.Error("expected nil") }
    })

    t.Run("warning at 50% increase", func(t *testing.T) {
        before := &WindowMetrics{P99Ms: 100, RequestCount: 200}
        after := &WindowMetrics{P99Ms: 155, RequestCount: 200}  // +55%
        r := detectLatencyRegression(before, after, cfg)
        if r == nil { t.Fatal("expected regression") }
        if r.Severity != SeverityWarning { t.Errorf("expected warning, got %s", r.Severity) }
    })

    t.Run("critical at 2x increase", func(t *testing.T) {
        before := &WindowMetrics{P99Ms: 100, RequestCount: 500}
        after := &WindowMetrics{P99Ms: 250, RequestCount: 500}  // +150%
        r := detectLatencyRegression(before, after, cfg)
        if r == nil { t.Fatal("expected regression") }
        if r.Severity != SeverityCritical { t.Errorf("expected critical, got %s", r.Severity) }
    })

    t.Run("low confidence with small sample", func(t *testing.T) {
        before := &WindowMetrics{P99Ms: 100, RequestCount: 30}
        after := &WindowMetrics{P99Ms: 200, RequestCount: 30}
        r := detectLatencyRegression(before, after, cfg)
        if r == nil { t.Fatal("expected regression") }
        if r.Confidence > 40 { t.Errorf("expected low confidence, got %d", r.Confidence) }
    })

    t.Run("skip when below min sample size", func(t *testing.T) {
        cfg := LatencyConfig{P99ThresholdPercent: 50, MinSampleSize: 100}
        before := &WindowMetrics{P99Ms: 100, RequestCount: 50}
        after := &WindowMetrics{P99Ms: 200, RequestCount: 50}  // below min 100
        r := detectLatencyRegression(before, after, cfg)
        if r != nil { t.Error("expected nil, below min sample size") }
    })
}
```

Similar tests for: `detectErrorRate`, `detectCacheMiss`, `detectDBFanout`, `detectPayloadGrowth`, `detectTenantOutlier`. Test threshold boundaries, severity classification, confidence scoring, min sample size gating.

Also test helpers: `computeConfidence(sampleSize int64, changePercent, threshold float64, consistent bool) int` and `classifySeverity(changePercent float64) Severity`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/regression/ -v -run TestDetect`
Expected: FAIL — functions don't exist.

- [ ] **Step 3: Write algorithms.go**

6 pure detection functions + `computeConfidence` + `classifySeverity` helpers:

```go
func detectLatencyRegression(before, after *WindowMetrics, cfg LatencyConfig) *Regression {
    if after.RequestCount < int64(cfg.MinSampleSize) { return nil }
    if before.P99Ms == 0 { return nil }
    changePct := (after.P99Ms - before.P99Ms) / before.P99Ms * 100
    sev := classifySeverity(changePct)
    if sev == "" { return nil }
    consistent := (after.P95Ms-before.P95Ms)/before.P95Ms*100 > 20 // both P95 and P99 regressed
    return &Regression{
        Algorithm:     AlgoLatencyRegression,
        Severity:      sev,
        Confidence:    computeConfidence(after.RequestCount, changePct, cfg.P99ThresholdPercent, consistent),
        BeforeValue:   before.P99Ms,
        AfterValue:    after.P99Ms,
        ChangePercent: changePct,
        SampleSize:    after.RequestCount,
        Title:         fmt.Sprintf("P99 latency increased %.0f%% (%s)", changePct, after.Service),
    }
}
```

Each algorithm follows the same pattern: check min sample → compute change → classify severity → compute confidence → return Regression or nil.

- `detectErrorRate` — compares `ErrorRate`, threshold in percentage points (not percent)
- `detectTenantOutlier` — takes tenant + fleet metrics, compares `P99Ms * multiplier`
- `detectCacheMiss` — compares `CacheMissRate`, threshold in percentage points
- `detectDBFanout` — compares `AvgSpanCount` percent change
- `detectPayloadGrowth` — compares `AvgRespSize` percent change

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/regression/ -v -run TestDetect -cover`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/regression/algorithms.go pkg/regression/algorithms_test.go
git commit -m "feat(regression): add 6 detection algorithms with confidence scoring

Pure functions comparing before/after WindowMetrics. Configurable
thresholds, min sample size gating, severity + confidence classification."
```

---

## Task 3: PostgreSQL Schema & Store

**Files:**
- Create: `docker/init/postgres/02-regression-schema.sql`
- Create: `pkg/regression/postgres/store.go`
- Create: `pkg/regression/postgres/store_test.go`

- [ ] **Step 1: Write PostgreSQL schema**

Create `docker/init/postgres/02-regression-schema.sql`:
```sql
CREATE TABLE regressions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm       TEXT NOT NULL,
    severity        TEXT NOT NULL,
    confidence      INT NOT NULL,
    service         TEXT NOT NULL,
    action          TEXT,
    deploy_id       UUID REFERENCES deploys(id),
    tenant_id       TEXT,
    title           TEXT NOT NULL,
    before_value    DOUBLE PRECISION NOT NULL,
    after_value     DOUBLE PRECISION NOT NULL,
    change_percent  DOUBLE PRECISION NOT NULL,
    sample_size     BIGINT NOT NULL,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    window_before   TSTZRANGE NOT NULL,
    window_after    TSTZRANGE NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX idx_regressions_service ON regressions(service, detected_at DESC);
CREATE INDEX idx_regressions_deploy ON regressions(deploy_id);
CREATE INDEX idx_regressions_status ON regressions(status) WHERE status = 'active';
```

- [ ] **Step 2: Write store test**

Create `pkg/regression/postgres/store_test.go`. Use testcontainers with PostgreSQL. Apply both `01-schema.sql` and `02-regression-schema.sql`. Test:
- `Save` + `Get` roundtrip
- `List` with filters (service, severity, status, deploy_id)
- `UpdateStatus` (active → dismissed, active → resolved)
- `ActiveForDeploy` returns only active regressions for a deploy

Guard with `if testing.Short() { t.Skip() }`.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/regression/postgres/ -v -short`
Expected: FAIL or skip.

- [ ] **Step 4: Write store implementation**

Create `pkg/regression/postgres/store.go`:
- `Save` — INSERT with all fields, `window_before`/`window_after` as `tstzrange`
- `List` — dynamic WHERE from `RegressionFilter`, ORDER BY `detected_at DESC`
- `Get` — SELECT by id, return `ErrNotFound` if missing
- `UpdateStatus` — UPDATE status (and resolved_at if status is "resolved")
- `ActiveForDeploy` — `WHERE deploy_id = $1 AND status = 'active'`

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/regression/postgres/ -v -cover`
Expected: PASS (or skip in short mode).

- [ ] **Step 6: Commit**

```bash
git add docker/init/postgres/02-regression-schema.sql pkg/regression/postgres/
git commit -m "feat(regression): add PostgreSQL regression store

CRUD operations with filtering by service, severity, status, deploy_id.
Integration tests with testcontainers."
```

---

## Task 4: In-Memory Store & MetricSource

**Files:**
- Create: `pkg/regression/memory/store.go`
- Create: `pkg/regression/memory/store_test.go`
- Create: `pkg/regression/memory/source.go`

- [ ] **Step 1: Write in-memory store test**

Test Save/List/Get/UpdateStatus/ActiveForDeploy with in-memory store. Same behavioral tests as PostgreSQL but no testcontainers needed.

- [ ] **Step 2: Write in-memory store implementation**

Mutex-protected slice of `Regression`. `Save` appends with generated UUID. `List` filters in Go. `UpdateStatus` scans for ID.

- [ ] **Step 3: Run tests**

Run: `go test ./pkg/regression/memory/ -v`
Expected: PASS.

- [ ] **Step 4: Write in-memory MetricSource**

Create `pkg/regression/memory/source.go`. Implements `MetricSource` by computing `WindowMetrics` from `gateway.RequestLog` and `gateway.TraceStore`:
- `WindowMetrics` — filter requests by service + time window, compute P50/P95/P99 from latencies, error rate from status codes
- `TenantWindowMetrics` — same but filtered by tenant_id from request headers
- `FleetWindowMetrics` — same without tenant filter
- `ListServices` — unique services from recent requests
- `ListTenants` — unique tenant_ids for a service
- Cache miss rate from trace metadata `x-cache-status`
- Span count from trace children for fanout
- Response body length for payload size

- [ ] **Step 5: Run all memory tests**

Run: `go test ./pkg/regression/memory/ -v -cover`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/regression/memory/
git commit -m "feat(regression): add in-memory store and MetricSource for local mode

MetricSource computes WindowMetrics from RequestLog and TraceStore.
Store uses mutex-protected slice."
```

---

## Task 5: Production MetricSource

**Files:**
- Create: `pkg/regression/source_impl.go`
- Create: `pkg/regression/source_impl_test.go`
- Modify: `docker/config/recording-rules.yml`

- [ ] **Step 1: Add Prometheus recording rules**

Append to `docker/config/recording-rules.yml` after the existing SLO group:

```yaml
  - name: regression_detection
    interval: 30s
    rules:
      - record: cloudmock:service_p50_5m
        expr: histogram_quantile(0.5, sum(rate(cloudmock_http_request_duration_seconds_bucket[5m])) by (service, action, le))
      - record: cloudmock:service_p95_5m
        expr: histogram_quantile(0.95, sum(rate(cloudmock_http_request_duration_seconds_bucket[5m])) by (service, action, le))
      - record: cloudmock:service_p99_5m
        expr: histogram_quantile(0.99, sum(rate(cloudmock_http_request_duration_seconds_bucket[5m])) by (service, action, le))
      - record: cloudmock:service_error_rate_5m
        expr: sum(rate(cloudmock_http_request_errors_total[5m])) by (service, action) / sum(rate(cloudmock_http_requests_total[5m])) by (service, action)
      - record: cloudmock:service_request_count_5m
        expr: sum(increase(cloudmock_http_requests_total[5m])) by (service, action)
      - record: cloudmock:tenant_p99_5m
        expr: histogram_quantile(0.99, sum(rate(cloudmock_http_request_duration_seconds_bucket[5m])) by (service, tenant_id, le))
      - record: cloudmock:fleet_p99_5m
        expr: histogram_quantile(0.99, sum(rate(cloudmock_http_request_duration_seconds_bucket[5m])) by (service, le))
```

- [ ] **Step 2: Write source_impl test**

Mock Prometheus HTTP API (same pattern as `pkg/dataplane/prometheus/metrics_test.go`) and ClickHouse queries. Test:
- `WindowMetrics` returns correct values from Prometheus for latency/error/count
- `WindowMetrics` returns correct cache miss/fanout/payload from ClickHouse
- `ListServices` returns unique services
- `TenantWindowMetrics` queries tenant-scoped recording rules

- [ ] **Step 3: Write source_impl.go**

Production `MetricSource` implementation. Constructor: `NewMetricSource(promAPI promv1.API, chConn driver.Conn)`.

- `WindowMetrics` — queries Prometheus recording rules for P50/P95/P99, error rate, request count at a specific timestamp. Queries ClickHouse for cache miss, fanout, payload size within the time window.
- `TenantWindowMetrics` — queries `cloudmock:tenant_p99_5m{tenant_id="X"}` from Prometheus
- `FleetWindowMetrics` — queries `cloudmock:fleet_p99_5m` from Prometheus
- `ListServices` — `SELECT DISTINCT service_name FROM spans WHERE start_time > now() - interval`
- `ListTenants` — `SELECT DISTINCT tenant_id FROM spans WHERE service_name = ? ORDER BY count(*) DESC LIMIT ?`

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/regression/ -v -run TestMetricSource -cover`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/regression/source_impl.go pkg/regression/source_impl_test.go docker/config/recording-rules.yml
git commit -m "feat(regression): add production MetricSource (Prometheus + ClickHouse)

Queries Prometheus recording rules for latency/error aggregates.
Queries ClickHouse for cache miss, DB fanout, payload size.
Extends recording rules with regression detection metrics."
```

---

## Task 6: Regression Engine

**Files:**
- Create: `pkg/regression/engine.go`
- Create: `pkg/regression/engine_test.go`

- [ ] **Step 1: Write engine test**

Mock `MetricSource` and `RegressionStore`. Test:

```go
func TestEngine_DeployTriggered(t *testing.T) {
    // Create engine with mock source that returns before/after WindowMetrics
    // showing a P99 regression
    // Call engine.OnDeploy(deployEvent)
    // Verify store.Save was called with a latency regression
}

func TestEngine_ContinuousScan(t *testing.T) {
    // Create engine with mock source returning degraded metrics
    // Call engine.Scan(ctx)
    // Verify regressions detected and saved
}

func TestEngine_AutoResolution(t *testing.T) {
    // Create engine with active regression in store
    // Mock source returns recovered metrics (within 10% of before)
    // Call engine.Scan(ctx)
    // Verify regression status updated to "resolved"
}

func TestEngine_DeployReEvaluation(t *testing.T) {
    // Verify that OnDeploy schedules re-evaluation at 1m, 5m, 15m
    // Use short timers for testing (1ms, 5ms, 15ms)
    // Verify store.Save called multiple times with updated confidence
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/regression/ -v -run TestEngine`
Expected: FAIL.

- [ ] **Step 3: Write engine.go**

```go
type Engine struct {
    source     MetricSource
    store      RegressionStore
    config     AlgorithmConfig
    deploys    dataplane.ConfigStore
    interval   time.Duration
    window     time.Duration
    pending    chan pendingDeploy
    stop       chan struct{}
}

func New(source MetricSource, store RegressionStore, deploys dataplane.ConfigStore, cfg AlgorithmConfig, interval, window time.Duration) *Engine

// Start begins the continuous scan ticker and deploy re-evaluation consumer
func (e *Engine) Start(ctx context.Context)

// Stop signals the engine to shut down
func (e *Engine) Stop()

// OnDeploy is called when a new deploy is recorded — queues deploy for evaluation
func (e *Engine) OnDeploy(deploy dataplane.DeployEvent)

// Scan runs one cycle of continuous detection across all services
func (e *Engine) Scan(ctx context.Context) error

// scanService runs all 6 algorithms for one service
func (e *Engine) scanService(ctx context.Context, service string) ([]Regression, error)

// checkResolutions checks active regressions and resolves recovered ones
func (e *Engine) checkResolutions(ctx context.Context) error

// evaluateDeploy runs all 6 algorithms comparing before/after a deploy
func (e *Engine) evaluateDeploy(ctx context.Context, deploy pendingDeploy) error
```

The `Start` method launches two goroutines:
1. Ticker goroutine: calls `Scan()` every `interval`, then `checkResolutions()`
2. Pending deploy consumer: reads from `pending` channel, schedules `time.AfterFunc` callbacks at 1m, 5m, 15m post-deploy

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/regression/ -v -run TestEngine -cover`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/regression/engine.go pkg/regression/engine_test.go
git commit -m "feat(regression): add detection engine with deploy-triggered and continuous scan

Periodic scan every 5m, deploy re-evaluation at 1m/5m/15m,
auto-resolution when metrics recover."
```

---

## Task 7: Config Extension & API Wiring

**Files:**
- Modify: `pkg/config/config.go:108-122`
- Modify: `pkg/admin/api.go:58-78, 131-175, 1619-1640`
- Modify: `cmd/gateway/main.go:303-312`

- [ ] **Step 1: Add RegressionConfig to config**

In `pkg/config/config.go`, add to the Config struct:

```go
type RegressionConfig struct {
    Enabled      bool                          `yaml:"enabled" json:"enabled"`
    ScanInterval string                        `yaml:"scan_interval" json:"scan_interval"` // e.g., "5m"
    Window       string                        `yaml:"window" json:"window"`               // e.g., "15m"
    Algorithms   regression.AlgorithmConfig    `yaml:"algorithms" json:"algorithms"`
}
```

Add `Regression RegressionConfig` to the Config struct. Set defaults in `Default()`: enabled=true, scan_interval="5m", window="15m", algorithms from `DefaultAlgorithmConfig()`.

Note: To avoid a circular import (config → regression), define the algorithm config types in the config package or keep them in regression and have config reference regression. Follow whichever pattern works — check existing patterns first.

- [ ] **Step 2: Add regression API endpoints to admin API**

Add to `pkg/admin/api.go`:
- `regressionEngine *regression.Engine` field on API struct
- `SetRegressionEngine(engine *regression.Engine)` setter
- `handleRegressions` handler:
  - `GET /api/regressions` — list with query param filters (service, deploy_id, severity, status, limit)
  - `GET /api/regressions/{id}` — detail by ID
  - `POST /api/regressions/{id}/dismiss` — set status to "dismissed"
- Register routes in `NewWithDataPlane()`
- In `handleDeploys` POST handler (around line 1636), after `AddDeploy()`, call `engine.OnDeploy(deploy)`

- [ ] **Step 3: Wire regression engine in main.go**

In `cmd/gateway/main.go`, after DataPlane construction:

```go
// Create regression engine
if cfg.Regression.Enabled {
    var regStore regression.RegressionStore
    var regSource regression.MetricSource

    switch mode {
    case "local":
        regStore = regmemory.NewStore()
        regSource = regmemory.NewMetricSource(requestLog, traceStore)
    case "production":
        regStore = regpg.NewStore(pgPool)
        regSource = regression.NewMetricSource(promClient.API(), chClient.Conn())
    }

    scanInterval, _ := time.ParseDuration(cfg.Regression.ScanInterval)
    window, _ := time.ParseDuration(cfg.Regression.Window)
    regEngine := regression.New(regSource, regStore, dp.Config, cfg.Regression.Algorithms, scanInterval, window)
    regEngine.Start(ctx)
    defer regEngine.Stop()

    adminAPI.SetRegressionEngine(regEngine)
}
```

- [ ] **Step 4: Run all tests**

Run: `go test ./pkg/admin/ ./pkg/gateway/ ./pkg/config/ ./pkg/regression/... -v -short`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/admin/api.go cmd/gateway/main.go
git commit -m "feat(regression): wire engine into admin API and gateway

Config extension with regression settings. Three new API endpoints.
Deploy-triggered detection on POST /api/deploys. Engine started
in both local and production modes."
```

---

## Task Summary

| Task | What it builds | Depends on |
|------|---------------|------------|
| 1 | Types, MetricSource, RegressionStore interfaces | — |
| 2 | 6 detection algorithms (pure functions) | 1 |
| 3 | PostgreSQL schema + store implementation | 1 |
| 4 | In-memory store + MetricSource (local mode) | 1 |
| 5 | Production MetricSource (Prometheus + ClickHouse) | 1 |
| 6 | Engine (scan loop, deploy trigger, auto-resolution) | 1, 2 |
| 7 | Config extension + API wiring + main.go | 1-6 |
