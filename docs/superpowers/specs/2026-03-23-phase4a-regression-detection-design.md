# Phase 4a: Regression Detection Engine — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 4a of 6 (CloudMock Console — Intelligence Layer, sub-project 1 of 5)
**Depends on:** Phase 3 (Production Data Plane)

---

## Overview

A regression detection engine that identifies performance and reliability degradations through two paths: deploy-triggered analysis (comparing before/after windows when a deploy is recorded) and continuous periodic scanning (detecting gradual degradation and tenant outliers). Results are classified by severity (impact magnitude) and confidence (statistical certainty), stored in PostgreSQL, and exposed via API.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Detection trigger | Continuous + event-driven | Deploy-triggered catches deploy-correlated regressions; periodic 5m scan catches gradual degradation and tenant outliers |
| Storage | PostgreSQL for results, Prometheus for detection | Prometheus precomputes aggregates via recording rules; Go algorithms compare windows and compute confidence; PostgreSQL stores results |
| Classification | Severity + confidence | Severity measures impact magnitude (how bad); confidence measures certainty (how sure based on sample size and consistency) |
| Configuration | Per-algorithm config with sensible defaults | Matches SLO rule pattern; each algorithm has different sensitivity needs |
| Architecture | Hybrid Prometheus + Go | Prometheus precomputes heavy aggregations; Go implements comparison logic, deploy correlation, and confidence scoring |

---

## 1. Data Model

### Regression struct

```go
// pkg/regression/types.go

type Severity string
const (
    SeverityCritical Severity = "critical"  // >2x baseline
    SeverityWarning  Severity = "warning"   // >50% baseline
    SeverityInfo     Severity = "info"      // >20% baseline
)

type AlgorithmType string
const (
    AlgoLatencyRegression AlgorithmType = "latency_regression"
    AlgoErrorRate         AlgorithmType = "error_rate"
    AlgoTenantOutlier     AlgorithmType = "tenant_outlier"
    AlgoCacheMiss         AlgorithmType = "cache_miss"
    AlgoDBFanout          AlgorithmType = "db_fanout"
    AlgoPayloadGrowth     AlgorithmType = "payload_growth"
)

type Regression struct {
    ID             string
    Algorithm      AlgorithmType
    Severity       Severity
    Confidence     int            // 0-100
    Service        string
    Action         string
    DeployID       string         // empty for non-deploy-correlated
    TenantID       string         // for tenant outlier only
    Title          string         // auto-generated summary
    BeforeValue    float64        // e.g., P99 was 120ms
    AfterValue     float64        // e.g., P99 is now 350ms
    ChangePercent  float64        // e.g., 191.7%
    SampleSize     int64          // requests in the after window
    DetectedAt     time.Time
    WindowBefore   TimeWindow
    WindowAfter    TimeWindow
    Status         string         // "active", "resolved", "dismissed"
    ResolvedAt     *time.Time
}

type TimeWindow struct {
    Start time.Time
    End   time.Time
}

type WindowMetrics struct {
    Service       string
    Action        string
    P50Ms         float64
    P95Ms         float64
    P99Ms         float64
    ErrorRate     float64
    RequestCount  int64
    CacheMissRate float64
    AvgSpanCount  float64     // spans per trace (fanout)
    AvgRespSize   float64     // response body size bytes
}
```

### PostgreSQL table

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

---

## 2. Detection Algorithms

Six pure functions, each taking before/after `WindowMetrics` and returning `*Regression` (or nil if no regression detected).

```go
// pkg/regression/algorithms.go

func detectLatencyRegression(before, after *WindowMetrics, cfg LatencyConfig) *Regression
func detectErrorRate(before, after *WindowMetrics, cfg ErrorConfig) *Regression
func detectCacheMiss(before, after *WindowMetrics, cfg CacheMissConfig) *Regression
func detectDBFanout(before, after *WindowMetrics, cfg FanoutConfig) *Regression
func detectPayloadGrowth(before, after *WindowMetrics, cfg PayloadConfig) *Regression

// Tenant outlier compares one tenant vs fleet:
func detectTenantOutlier(tenant *WindowMetrics, fleet *WindowMetrics, cfg OutlierConfig) *Regression
```

### Confidence scoring

Each algorithm computes confidence from:
- **Sample size**: <50 requests → 30, 50-500 → 60, >500 → 85
- **Magnitude bonus**: change exceeds 2x threshold → +10
- **Consistency**: both P95 and P99 regressed (not just one) → +5
- Capped at 100

### Severity classification

- `>2x baseline` → critical
- `>50% baseline` → warning
- `>20% baseline` → info
- `<20%` → no regression (return nil)

### Default thresholds

| Algorithm | Threshold | Min Samples |
|-----------|-----------|-------------|
| Latency regression | P99 +50% | 100 |
| Error rate | +5 percentage points | 50 |
| Tenant outlier | P99 > 3x fleet avg | 200 |
| Cache miss | +20 percentage points | 100 |
| DB fanout | +50% avg spans/trace | 50 |
| Payload growth | +100% avg response size | 50 |

### Algorithm configuration

```go
type AlgorithmConfig struct {
    LatencyRegression LatencyConfig   `yaml:"latency_regression"`
    ErrorRate         ErrorConfig     `yaml:"error_rate"`
    TenantOutlier     OutlierConfig   `yaml:"tenant_outlier"`
    CacheMiss         CacheMissConfig `yaml:"cache_miss"`
    DBFanout          FanoutConfig    `yaml:"db_fanout"`
    PayloadGrowth     PayloadConfig   `yaml:"payload_growth"`
}

type LatencyConfig struct {
    P99ThresholdPercent float64 `yaml:"p99_threshold_percent"` // default: 50
    MinSampleSize       int     `yaml:"min_sample_size"`       // default: 100
}

type ErrorConfig struct {
    ThresholdPP   float64 `yaml:"threshold_pp"`    // default: 5
    MinSampleSize int     `yaml:"min_sample_size"` // default: 50
}

type OutlierConfig struct {
    Multiplier    float64 `yaml:"multiplier"`      // default: 3.0
    MinSampleSize int     `yaml:"min_sample_size"` // default: 200
    MaxTenants    int     `yaml:"max_tenants"`     // default: 100, top by request volume
}

type CacheMissConfig struct {
    ThresholdPP   float64 `yaml:"threshold_pp"`    // default: 20
    MinSampleSize int     `yaml:"min_sample_size"` // default: 100
}

type FanoutConfig struct {
    ThresholdPercent float64 `yaml:"threshold_percent"` // default: 50
    MinSampleSize    int     `yaml:"min_sample_size"`   // default: 50
}

type PayloadConfig struct {
    ThresholdPercent float64 `yaml:"threshold_percent"` // default: 100
    MinSampleSize    int     `yaml:"min_sample_size"`   // default: 50
}
```

---

## 3. Engine Architecture

```go
// pkg/regression/engine.go

// MetricSource composes Prometheus and ClickHouse queries to build WindowMetrics.
// This is NOT an extension of dataplane.MetricReader — it's a regression-specific
// interface that reads from both Prometheus (latency, error rate, request count)
// and ClickHouse (cache miss, fanout, payload size).
type MetricSource interface {
    WindowMetrics(ctx context.Context, service, action string, window TimeWindow) (*WindowMetrics, error)
    TenantWindowMetrics(ctx context.Context, service, tenantID string, window TimeWindow) (*WindowMetrics, error)
    FleetWindowMetrics(ctx context.Context, service string, window TimeWindow) (*WindowMetrics, error)
    ListServices(ctx context.Context) ([]string, error)
    ListTenants(ctx context.Context, service string) ([]string, error)
}

type Engine struct {
    source     MetricSource              // composes Prometheus + ClickHouse
    config     dataplane.ConfigStore
    store      RegressionStore
    algorithms AlgorithmConfig
    interval   time.Duration             // default 5m
    window     time.Duration             // default 15m
    stop       chan struct{}
    pending    chan pendingDeploy         // deploy re-evaluation queue
}

// pendingDeploy tracks deploys needing re-evaluation at 1m, 5m, 15m post-deploy
type pendingDeploy struct {
    DeployID  string
    Service   string
    DeployAt  time.Time
    EvalTimes []time.Duration // remaining: e.g., [1m, 5m, 15m]
}

type RegressionStore interface {
    Save(ctx context.Context, r *Regression) error
    List(ctx context.Context, filter RegressionFilter) ([]Regression, error)
    Get(ctx context.Context, id string) (*Regression, error)
    UpdateStatus(ctx context.Context, id string, status string) error
    ActiveForDeploy(ctx context.Context, deployID string) ([]Regression, error)
}

type RegressionFilter struct {
    Service   string
    DeployID  string
    Algorithm AlgorithmType
    Severity  Severity
    Status    string
    Limit     int
}
```

### Two detection paths

**Deploy-triggered:** Called from the admin API's deploy handler. When `POST /api/deploys` is received, the deploy is sent to the `pending` channel. A dedicated goroutine consumes from this channel and schedules evaluations at 1m, 5m, and 15m post-deploy using `time.AfterFunc`. Each evaluation runs all 6 algorithms comparing the `window` before the deploy vs current. Results from later evaluations replace earlier ones (keyed by deploy_id + algorithm + service). If the 1m evaluation finds a regression, subsequent evaluations still run to refine confidence as more data accumulates.

**Continuous scan:** Ticker every `interval`. For each service, queries Prometheus for the current window vs the previous window. Detects gradual degradation not tied to deploys. Tenant outlier detection runs here — queries `MetricSource.ListTenants()` which returns tenants active in the current window (bounded by Prometheus cardinality, typically <1000). For services with >100 active tenants, the scan samples the top 100 by request volume to keep query cost bounded. The `OutlierConfig` has a `MaxTenants` field (default: 100) controlling this limit.

### Auto-resolution

On each scan, the engine checks active regressions. If the metric has returned to within 10% of the before-value, the regression is marked as `resolved`.

### API endpoints

```
GET  /api/regressions                 — list with filters (service, deploy_id, severity, status)
GET  /api/regressions/{id}            — detail
POST /api/regressions/{id}/dismiss    — manually dismiss a false positive
```

---

## 4. Prometheus Recording Rules

New recording rules precompute the aggregates the engine needs:

```yaml
groups:
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

### Algorithm → data source mapping

| Algorithm | Data Source | Query |
|-----------|-------------|-------|
| Latency regression | Prometheus | `cloudmock:service_p99_5m` at time T vs T-window |
| Error rate | Prometheus | `cloudmock:service_error_rate_5m` |
| Tenant outlier | Prometheus | `cloudmock:tenant_p99_5m` vs `cloudmock:fleet_p99_5m` |
| Cache miss | ClickHouse | `SELECT countIf(metadata['x-cache-status']='MISS') / count(*)` |
| DB fanout | ClickHouse | `SELECT avg(span_count) FROM (SELECT trace_id, count(*) as span_count ... GROUP BY trace_id)` |
| Payload growth | ClickHouse | `SELECT avg(length(response_body))` |

---

## 5. Configuration

```yaml
# cloudmock.yml extension
regression:
  enabled: true
  scan_interval: 5m
  window: 15m
  algorithms:
    latency_regression:
      p99_threshold_percent: 50
      min_sample_size: 100
    error_rate:
      threshold_pp: 5
      min_sample_size: 50
    tenant_outlier:
      multiplier: 3.0
      min_sample_size: 200
      max_tenants: 100
    cache_miss:
      threshold_pp: 20
      min_sample_size: 100
    db_fanout:
      threshold_percent: 50
      min_sample_size: 50
    payload_growth:
      threshold_percent: 100
      min_sample_size: 50
```

In production mode, regression config is stored in PostgreSQL's `config` table (key: `"regression"`) and can be updated via the config API, matching the Phase 3 pattern.

---

## 6. Local Mode

The engine works identically in local mode. The `MetricSource` interface abstracts the backend — same algorithms, different data sources. A `memory.MetricSource` implementation computes `WindowMetrics` from the in-memory `RequestLog` (percentiles from latency samples, error rate from status codes) and `TraceStore` (cache miss from metadata, fanout from child count, payload size from response body). All 6 algorithms run against the same `MetricSource` interface regardless of mode.

---

## 7. Testing Strategy

**Unit tests (`pkg/regression/algorithms_test.go`):**
- Each algorithm as pure function — crafted `WindowMetrics` pairs
- Threshold boundaries (49% → nil, 51% → warning)
- Confidence scoring (low/medium/high sample sizes)
- Severity classification (20%/50%/200% changes)
- Tenant outlier with fleet baseline
- Auto-resolution logic

**Engine tests (`pkg/regression/engine_test.go`):**
- Mock `MetricReader`, `TraceReader`, `ConfigStore`, `RegressionStore`
- Deploy-triggered path: simulate deploy → verify algorithms ran → check results stored
- Continuous scan: tick → verify services scanned → check results
- Auto-resolution: active regression → metric recovers → verify resolved

**Integration tests (`pkg/regression/store_test.go`):**
- Testcontainers with PostgreSQL for `RegressionStore`
- Full CRUD: Save → List → Get → UpdateStatus
- Filters: by service, deploy_id, severity, status

**Admin API tests:**
- `GET /api/regressions` returns stored regressions with filters
- `GET /api/regressions/{id}` returns detail
- `POST /api/regressions/{id}/dismiss` updates status

---

## File Layout

```
pkg/regression/
├── types.go              # Regression, WindowMetrics, Severity, AlgorithmType, config types
├── algorithms.go         # 6 pure detection functions + confidence/severity helpers
├── algorithms_test.go
├── engine.go             # Engine struct, deploy-triggered + continuous scan, auto-resolution
├── engine_test.go
├── store.go              # RegressionStore interface + RegressionFilter
├── source.go             # MetricSource interface
├── source_impl.go        # MetricSource implementation composing Prometheus + ClickHouse
├── source_impl_test.go
├── postgres/
│   ├── store.go          # PostgreSQL RegressionStore implementation
│   └── store_test.go     # testcontainers integration test
└── memory/
    ├── store.go          # In-memory RegressionStore for local mode
    ├── store_test.go
    └── source.go         # In-memory MetricSource for local mode

docker/init/postgres/02-regression-schema.sql  # regressions table DDL
docker/config/recording-rules.yml              # updated with regression detection rules
```

**Files modified:**
- `pkg/config/config.go` — add `Regression RegressionConfig` to Config struct
- `pkg/admin/api.go` — add regression API handlers, wire deploy-triggered detection
- `cmd/gateway/main.go` — create and start regression engine
- `docker/config/recording-rules.yml` — add regression detection recording rules
- `docker/init/postgres/01-schema.sql` or new `02-regression-schema.sql` — regressions table
