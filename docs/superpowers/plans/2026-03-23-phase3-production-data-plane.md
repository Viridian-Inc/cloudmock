# Phase 3: Production Data Plane Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add production-grade storage backends (ClickHouse, PostgreSQL, Prometheus) behind storage interfaces so CloudMock Console runs in both local (in-memory) and production (persistent) modes with identical APIs.

**Architecture:** Seven storage interfaces (`TraceReader/Writer`, `RequestReader/Writer`, `MetricReader/Writer`, `SLOStore`, `ConfigStore`, `TopologyStore`) coordinated by a `DataPlane` struct. Local mode wraps existing in-memory stores. Production mode connects to ClickHouse (traces/requests), PostgreSQL (metadata/config/state), and Prometheus (metrics). OTel SDK replaces direct store writes in production mode.

**Tech Stack:** Go 1.26, ClickHouse 24.3, PostgreSQL 16, Prometheus 2.51, OpenTelemetry Go SDK, OTel Collector Contrib 0.97, Docker Compose, clickhouse-go/v2, pgx/v5, testcontainers-go

---

## File Structure

```
pkg/dataplane/
├── dataplane.go           # DataPlane struct, NewLocalDataPlane(), NewProductionDataPlane()
├── traces.go              # TraceReader, TraceWriter interfaces + shared types (Span)
├── requests.go            # RequestReader, RequestWriter interfaces
├── metrics.go             # MetricReader, MetricWriter interfaces + ServiceMetrics, LatencyPercentiles
├── slo.go                 # SLOStore interface + SLORuleChange type
├── config.go              # ConfigStore interface + DeployFilter, ServiceEntry types
├── topology.go            # TopologyStore interface + ObservedEdge, TopologyGraph types
├── otel.go                # InitTracer(), OTelConfig struct
├── memory/
│   ├── traces.go          # wraps gateway.TraceStore → TraceReader + TraceWriter
│   ├── traces_test.go
│   ├── requests.go        # wraps gateway.RequestLog → RequestReader + RequestWriter
│   ├── requests_test.go
│   ├── metrics.go         # wraps gateway.RequestStats → MetricReader + MetricWriter
│   ├── metrics_test.go
│   ├── slo.go             # wraps gateway.SLOEngine → SLOStore
│   ├── slo_test.go
│   ├── config.go          # wraps config.Config + in-memory slices → ConfigStore
│   ├── config_test.go
│   ├── topology.go        # wraps admin.IaCTopologyConfig → TopologyStore
│   └── topology_test.go
├── clickhouse/
│   ├── client.go          # NewClient(), connection management, health check
│   ├── traces.go          # TraceReader + TraceWriter against spans table
│   ├── traces_test.go     # testcontainers-go with real ClickHouse
│   ├── requests.go        # RequestReader + RequestWriter against spans table
│   └── requests_test.go
├── postgres/
│   ├── client.go          # NewPool(), connection management, health check
│   ├── slo.go             # SLOStore with transactional rule+history writes
│   ├── slo_test.go        # testcontainers-go with real PostgreSQL
│   ├── config.go          # ConfigStore (deploys, views, services, key-value config)
│   ├── config_test.go
│   ├── topology.go        # TopologyStore (edges, blast radius queries)
│   └── topology_test.go
├── prometheus/
│   ├── client.go          # NewClient(), Prometheus HTTP API wrapper
│   ├── metrics.go         # MetricReader (query API) + MetricWriter (no-op in prod)
│   └── metrics_test.go    # mock HTTP responses

cmd/configimport/
└── main.go                # YAML → PostgreSQL config import tool

docker/
├── docker-compose.prod.yml
├── init/
│   ├── clickhouse/01-schema.sql
│   └── postgres/01-schema.sql
└── config/
    ├── otel-collector.yml
    ├── prometheus.yml
    └── recording-rules.yml
```

**Files modified:**
- `pkg/config/config.go:77-90` — add `DataPlane DataPlaneConfig` to Config struct
- `cmd/gateway/main.go:226-266` — mode switch: build local or production DataPlane
- `pkg/admin/api.go:57-76` — replace individual store fields with `*dataplane.DataPlane`
- `pkg/admin/api.go:79-124` — new constructor `NewWithDataPlane()`
- `pkg/gateway/logging.go:252-257` — update `LoggingMiddlewareOpts` to accept `*dataplane.DataPlane`
- `pkg/gateway/logging.go:274-404` — mode-dependent write paths
- `go.mod` — add clickhouse-go/v2, pgx/v5, otel SDK, testcontainers-go, prometheus client

---

## Task 1: Storage Interfaces

**Files:**
- Create: `pkg/dataplane/traces.go`
- Create: `pkg/dataplane/requests.go`
- Create: `pkg/dataplane/metrics.go`
- Create: `pkg/dataplane/slo.go`
- Create: `pkg/dataplane/config.go`
- Create: `pkg/dataplane/topology.go`
- Create: `pkg/dataplane/dataplane.go`

- [ ] **Step 1: Create the dataplane package directory**

Run: `mkdir -p pkg/dataplane`

- [ ] **Step 2: Write trace interfaces**

Create `pkg/dataplane/traces.go`:

```go
package dataplane

import (
    "context"
    "time"
)

type Span struct {
    TraceID      string
    SpanID       string
    ParentSpanID string
    Service      string
    Action       string
    Method       string
    Path         string
    StartTime    time.Time
    EndTime      time.Time
    DurationNs   uint64
    StatusCode   int
    Error        string
    TenantID     string
    OrgID        string
    UserID       string
    MemAllocKB   float64
    Goroutines   uint32
    Metadata     map[string]string
    ReqHeaders   map[string]string
    ReqBody      string
    RespBody     string
}

// TraceContext mirrors gateway.TraceContext — assembled trace with children.
type TraceContext struct {
    TraceID      string
    SpanID       string
    ParentSpanID string
    Service      string
    Action       string
    Method       string
    Path         string
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration
    DurationMs   float64
    StatusCode   int
    Error        string
    Children     []*TraceContext
    Metadata     map[string]string
}

type TraceSummary struct {
    TraceID     string
    RootService string
    RootAction  string
    Method      string
    Path        string
    DurationMs  float64
    StatusCode  int
    SpanCount   int
    HasError    bool
    StartTime   time.Time
}

type TimelineSpan struct {
    SpanID        string
    ParentSpanID  string
    Service       string
    Action        string
    StartOffsetMs float64
    DurationMs    float64
    StatusCode    int
    Error         string
    Depth         int
    Metadata      map[string]string
}

type TraceFilter struct {
    Service    string
    HasError   *bool
    TenantID   string
    Limit      int
}

type TraceReader interface {
    Get(ctx context.Context, traceID string) (*TraceContext, error)
    Search(ctx context.Context, filter TraceFilter) ([]TraceSummary, error)
    Timeline(ctx context.Context, traceID string) ([]TimelineSpan, error)
}

type TraceWriter interface {
    WriteSpans(ctx context.Context, spans []*Span) error
}
```

- [ ] **Step 3: Write request interfaces**

Create `pkg/dataplane/requests.go`:

```go
package dataplane

import (
    "context"
    "time"
)

type RequestEntry struct {
    ID             string
    TraceID        string
    SpanID         string
    Timestamp      time.Time
    Service        string
    Action         string
    Method         string
    Path           string
    StatusCode     int
    Latency        time.Duration
    LatencyMs      float64
    CallerID       string
    Error          string
    Level          string
    MemAllocKB     float64
    Goroutines     int
    RequestHeaders map[string]string
    RequestBody    string
    ResponseBody   string
    TenantID       string
    OrgID          string
    UserID         string
}

type RequestFilter struct {
    Service      string
    Path         string
    Method       string
    CallerID     string
    Action       string
    ErrorOnly    bool
    TraceID      string
    Level        string
    Limit        int
    TenantID     string
    OrgID        string
    UserID       string
    MinLatencyMs float64
    MaxLatencyMs float64
    From         time.Time
    To           time.Time
}

type RequestReader interface {
    Query(ctx context.Context, filter RequestFilter) ([]RequestEntry, error)
    GetByID(ctx context.Context, id string) (*RequestEntry, error)
}

type RequestWriter interface {
    Write(ctx context.Context, entry RequestEntry) error
}
```

- [ ] **Step 4: Write metric interfaces**

Create `pkg/dataplane/metrics.go`:

```go
package dataplane

import (
    "context"
    "time"
)

type ServiceMetrics struct {
    Service      string
    RequestCount int64
    ErrorCount   int64
    ErrorRate    float64
    P50Ms        float64
    P95Ms        float64
    P99Ms        float64
}

type LatencyPercentiles struct {
    P50Ms float64
    P95Ms float64
    P99Ms float64
}

type MetricReader interface {
    ServiceStats(ctx context.Context, service string, window time.Duration) (*ServiceMetrics, error)
    Percentiles(ctx context.Context, service, action string, window time.Duration) (*LatencyPercentiles, error)
}

type MetricWriter interface {
    Record(ctx context.Context, service, action string, latencyMs float64, statusCode int) error
}
```

- [ ] **Step 5: Write SLO, config, topology interfaces**

Create `pkg/dataplane/slo.go`:

```go
package dataplane

import (
    "context"
    "time"

    "github.com/neureaux/cloudmock/pkg/config"
)

type SLORuleChange struct {
    RuleID     string
    ChangeType string // "created", "updated", "deleted"
    OldValues  *config.SLORule
    NewValues  *config.SLORule
    ChangedBy  string
    ChangedAt  time.Time
}

type SLOStore interface {
    Rules(ctx context.Context) ([]config.SLORule, error)
    SetRules(ctx context.Context, rules []config.SLORule) error
    Status(ctx context.Context) (*SLOStatus, error)
    History(ctx context.Context, limit int) ([]SLORuleChange, error)
}
```

Note: `SLOStatus` reuses the existing type from `pkg/gateway/slo.go`. Import it or duplicate minimally.

Create `pkg/dataplane/config.go`:

```go
package dataplane

import (
    "context"
    "time"

    "github.com/neureaux/cloudmock/pkg/config"
)

type DeployEvent struct {
    ID          string
    Service     string
    Version     string
    CommitSHA   string
    Author      string
    Description string
    DeployedAt  time.Time
    Metadata    map[string]string
}

type DeployFilter struct {
    Service  string
    Limit    int
}

type SavedView struct {
    ID        string
    Name      string
    Filters   map[string]interface{}
    CreatedBy string
    CreatedAt time.Time
}

type ServiceEntry struct {
    Name        string
    ServiceType string
    GroupName   string
    Description string
    Owner       string
    RepoURL     string
}

type ConfigStore interface {
    GetConfig(ctx context.Context) (*config.Config, error)
    SetConfig(ctx context.Context, cfg *config.Config) error
    ListDeploys(ctx context.Context, filter DeployFilter) ([]DeployEvent, error)
    AddDeploy(ctx context.Context, deploy DeployEvent) error
    ListViews(ctx context.Context) ([]SavedView, error)
    SaveView(ctx context.Context, view SavedView) error
    DeleteView(ctx context.Context, id string) error
    ListServices(ctx context.Context) ([]ServiceEntry, error)
    UpsertService(ctx context.Context, svc ServiceEntry) error
}
```

Create `pkg/dataplane/topology.go`:

```go
package dataplane

import "context"

type ObservedEdge struct {
    Source       string
    Target       string
    EdgeType     string // "iac", "extracted", "traffic"
    RequestCount int64
}

type TopologyNode struct {
    Name        string
    ServiceType string
    Group       string
}

type TopologyGraph struct {
    Nodes []TopologyNode
    Edges []ObservedEdge
}

type TopologyStore interface {
    GetTopology(ctx context.Context) (*TopologyGraph, error)
    RecordEdge(ctx context.Context, edge ObservedEdge) error
    Upstream(ctx context.Context, service string) ([]string, error)
    Downstream(ctx context.Context, service string) ([]string, error)
}
```

- [ ] **Step 6: Write DataPlane coordinator**

Create `pkg/dataplane/dataplane.go`:

```go
package dataplane

import "errors"

var ErrNotFound = errors.New("not found")

type DataPlane struct {
    Traces   TraceReader
    TraceW   TraceWriter
    Requests RequestReader
    RequestW RequestWriter
    Metrics  MetricReader
    MetricW  MetricWriter
    SLO      SLOStore
    Config   ConfigStore
    Topology TopologyStore
    Mode     string // "local" | "production"
}
```

- [ ] **Step 7: Verify package compiles**

Run: `cd /Users/megan/work/neureaux/cloudmock && go build ./pkg/dataplane/...`
Expected: Clean compile, no errors.

- [ ] **Step 8: Commit**

```bash
git add pkg/dataplane/
git commit -m "feat(dataplane): add storage interface definitions for Phase 3

Defines TraceReader/Writer, RequestReader/Writer, MetricReader/Writer,
SLOStore, ConfigStore, TopologyStore interfaces and DataPlane coordinator."
```

---

## Task 2: Memory Implementation Wrappers

**Files:**
- Create: `pkg/dataplane/memory/traces.go`
- Create: `pkg/dataplane/memory/requests.go`
- Create: `pkg/dataplane/memory/metrics.go`
- Create: `pkg/dataplane/memory/slo.go`
- Create: `pkg/dataplane/memory/config.go`
- Create: `pkg/dataplane/memory/topology.go`
- Create: `pkg/dataplane/memory/traces_test.go`
- Create: `pkg/dataplane/memory/requests_test.go`
- Create: `pkg/dataplane/memory/metrics_test.go`
- Create: `pkg/dataplane/memory/slo_test.go`
- Create: `pkg/dataplane/memory/config_test.go`
- Create: `pkg/dataplane/memory/topology_test.go`

- [ ] **Step 1: Write trace wrapper test**

Create `pkg/dataplane/memory/traces_test.go`:

```go
package memory_test

import (
    "context"
    "testing"
    "time"

    "github.com/neureaux/cloudmock/pkg/dataplane"
    "github.com/neureaux/cloudmock/pkg/dataplane/memory"
    "github.com/neureaux/cloudmock/pkg/gateway"
)

func TestTraceReaderGet(t *testing.T) {
    store := gateway.NewTraceStore(100)
    store.Add(&gateway.TraceContext{
        TraceID:   "trace-1",
        SpanID:    "span-1",
        Service:   "bff",
        Action:    "GetUser",
        StartTime: time.Now(),
        EndTime:   time.Now().Add(50 * time.Millisecond),
    })

    reader := memory.NewTraceReader(store)
    ctx := context.Background()

    tc, err := reader.Get(ctx, "trace-1")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if tc.Service != "bff" {
        t.Errorf("expected service bff, got %s", tc.Service)
    }

    _, err = reader.Get(ctx, "nonexistent")
    if err != dataplane.ErrNotFound {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}

func TestTraceReaderSearch(t *testing.T) {
    store := gateway.NewTraceStore(100)
    store.Add(&gateway.TraceContext{
        TraceID: "t1", SpanID: "s1", Service: "bff",
        StartTime: time.Now(), EndTime: time.Now(),
    })
    store.Add(&gateway.TraceContext{
        TraceID: "t2", SpanID: "s2", Service: "dynamodb",
        StartTime: time.Now(), EndTime: time.Now(),
    })

    reader := memory.NewTraceReader(store)
    results, err := reader.Search(context.Background(), dataplane.TraceFilter{
        Service: "bff", Limit: 10,
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) != 1 {
        t.Errorf("expected 1 result, got %d", len(results))
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/dataplane/memory/ -v -run TestTraceReader`
Expected: FAIL — `memory` package doesn't exist yet.

- [ ] **Step 3: Write trace wrapper implementation**

Create `pkg/dataplane/memory/traces.go`:

```go
package memory

import (
    "context"

    "github.com/neureaux/cloudmock/pkg/dataplane"
    "github.com/neureaux/cloudmock/pkg/gateway"
)

type traceReader struct {
    store *gateway.TraceStore
}

func NewTraceReader(store *gateway.TraceStore) dataplane.TraceReader {
    return &traceReader{store: store}
}

func (r *traceReader) Get(ctx context.Context, traceID string) (*dataplane.TraceContext, error) {
    t := r.store.Get(traceID)
    if t == nil {
        return nil, dataplane.ErrNotFound
    }
    return convertTraceContext(t), nil
}

func (r *traceReader) Search(ctx context.Context, filter dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
    summaries := r.store.Recent(filter.Service, filter.HasError, filter.Limit)
    result := make([]dataplane.TraceSummary, len(summaries))
    for i, s := range summaries {
        result[i] = dataplane.TraceSummary{
            TraceID:     s.TraceID,
            RootService: s.RootService,
            RootAction:  s.RootAction,
            Method:      s.Method,
            Path:        s.Path,
            DurationMs:  s.DurationMs,
            StatusCode:  s.StatusCode,
            SpanCount:   s.SpanCount,
            HasError:    s.HasError,
            StartTime:   s.StartTime,
        }
    }
    return result, nil
}

func (r *traceReader) Timeline(ctx context.Context, traceID string) ([]dataplane.TimelineSpan, error) {
    spans := r.store.Timeline(traceID)
    if len(spans) == 0 {
        return nil, dataplane.ErrNotFound
    }
    result := make([]dataplane.TimelineSpan, len(spans))
    for i, s := range spans {
        result[i] = dataplane.TimelineSpan{
            SpanID:        s.SpanID,
            ParentSpanID:  s.ParentSpanID,
            Service:       s.Service,
            Action:        s.Action,
            StartOffsetMs: s.StartOffsetMs,
            DurationMs:    s.DurationMs,
            StatusCode:    s.StatusCode,
            Error:         s.Error,
            Depth:         s.Depth,
            Metadata:      s.Metadata,
        }
    }
    return result, nil
}

type traceWriter struct {
    store *gateway.TraceStore
}

func NewTraceWriter(store *gateway.TraceStore) dataplane.TraceWriter {
    return &traceWriter{store: store}
}

func (w *traceWriter) WriteSpans(ctx context.Context, spans []*dataplane.Span) error {
    for _, s := range spans {
        w.store.Add(&gateway.TraceContext{
            TraceID:      s.TraceID,
            SpanID:       s.SpanID,
            ParentSpanID: s.ParentSpanID,
            Service:      s.Service,
            Action:       s.Action,
            Method:       s.Method,
            Path:         s.Path,
            StartTime:    s.StartTime,
            EndTime:      s.EndTime,
            StatusCode:   s.StatusCode,
            Error:        s.Error,
            Metadata:     s.Metadata,
        })
    }
    return nil
}

func convertTraceContext(tc *gateway.TraceContext) *dataplane.TraceContext {
    result := &dataplane.TraceContext{
        TraceID:      tc.TraceID,
        SpanID:       tc.SpanID,
        ParentSpanID: tc.ParentSpanID,
        Service:      tc.Service,
        Action:       tc.Action,
        Method:       tc.Method,
        Path:         tc.Path,
        StartTime:    tc.StartTime,
        EndTime:      tc.EndTime,
        Duration:     tc.Duration,
        DurationMs:   tc.DurationMs,
        StatusCode:   tc.StatusCode,
        Error:        tc.Error,
        Metadata:     tc.Metadata,
    }
    for _, child := range tc.Children {
        result.Children = append(result.Children, convertTraceContext(child))
    }
    return result
}
```

- [ ] **Step 4: Run trace tests**

Run: `go test ./pkg/dataplane/memory/ -v -run TestTraceReader`
Expected: PASS

- [ ] **Step 5: Write request wrapper test**

Create `pkg/dataplane/memory/requests_test.go` — test `NewRequestReader` delegates `Query()` and `GetByID()` to `gateway.RequestLog`.

- [ ] **Step 6: Write request wrapper implementation**

Create `pkg/dataplane/memory/requests.go` — thin wrapper converting `gateway.RequestEntry` ↔ `dataplane.RequestEntry`, delegating to `RequestLog.RecentFiltered()` and `RequestLog.GetByID()`.

- [ ] **Step 7: Run request tests**

Run: `go test ./pkg/dataplane/memory/ -v -run TestRequestReader`
Expected: PASS

- [ ] **Step 8: Write metrics wrapper (test + impl)**

Create `pkg/dataplane/memory/metrics_test.go` and `pkg/dataplane/memory/metrics.go`. `MetricReader` computes percentiles from `RequestLog` entries (same logic as existing `handleMetrics` in admin API). `MetricWriter` delegates to `RequestStats.Increment()`.

- [ ] **Step 9: Write SLO wrapper (test + impl)**

Create `pkg/dataplane/memory/slo_test.go` and `pkg/dataplane/memory/slo.go`. Delegates to `SLOEngine.Rules()`, `SetRules()`, `Status()`. `History()` returns empty slice (in-memory has no history).

- [ ] **Step 10: Write config wrapper (test + impl)**

Create `pkg/dataplane/memory/config_test.go` and `pkg/dataplane/memory/config.go`. Wraps `config.Config` for `GetConfig/SetConfig`. Deploys and views stored in in-memory slices (mirrors current `admin.API` behavior). Services derived from IaC topology.

- [ ] **Step 11: Write topology wrapper (test + impl)**

Create `pkg/dataplane/memory/topology_test.go` and `pkg/dataplane/memory/topology.go`. Wraps existing IaC topology config. `RecordEdge` appends to in-memory slice. `Upstream/Downstream` walks edges.

- [ ] **Step 12: Run all memory tests**

Run: `go test ./pkg/dataplane/memory/ -v -cover`
Expected: All PASS, reasonable coverage.

- [ ] **Step 13: Commit**

```bash
git add pkg/dataplane/memory/
git commit -m "feat(dataplane): add memory implementation wrappers

Thin adapters around existing TraceStore, RequestLog, RequestStats,
SLOEngine to satisfy dataplane interfaces. Zero logic duplication."
```

---

## Task 3: Config Extension & DataPlane Wiring

**Files:**
- Modify: `pkg/config/config.go:77-90` — add DataPlaneConfig
- Modify: `cmd/gateway/main.go:226-266` — mode switch
- Modify: `pkg/admin/api.go:57-124` — accept DataPlane
- Modify: `pkg/gateway/logging.go:252-404` — mode-dependent paths

- [ ] **Step 1: Add DataPlaneConfig to config struct**

Modify `pkg/config/config.go`. Add after line 90:

```go
type DataPlaneConfig struct {
    Mode       string           `yaml:"mode" json:"mode"`             // "local" (default) or "production"
    ClickHouse ClickHouseConfig `yaml:"clickhouse" json:"clickhouse"`
    PostgreSQL PostgreSQLConfig `yaml:"postgresql" json:"postgresql"`
    Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
    OTel       OTelConfig       `yaml:"otel" json:"otel"`
}

type ClickHouseConfig struct {
    Endpoint string `yaml:"endpoint" json:"endpoint"`
    Database string `yaml:"database" json:"database"`
}

type PostgreSQLConfig struct {
    URL string `yaml:"url" json:"url"`
}

type PrometheusConfig struct {
    URL string `yaml:"url" json:"url"`
}

type OTelConfig struct {
    CollectorEndpoint string `yaml:"collector_endpoint" json:"collector_endpoint"`
    ServiceName       string `yaml:"service_name" json:"service_name"`
}
```

Add `DataPlane DataPlaneConfig` to the `Config` struct. Default `Mode` to `"local"` in `Load()` if empty.

- [ ] **Step 2: Verify config loads**

Run: `go test ./pkg/config/ -v`
Expected: PASS (existing tests still work, new field is optional with zero value).

- [ ] **Step 3: Add NewWithDataPlane to admin API**

Modify `pkg/admin/api.go`. Add a new constructor that accepts `*dataplane.DataPlane` instead of individual stores. Keep the old `New()` constructor working (it will create a local DataPlane internally). Update handler methods to read from `a.dp.Traces`, `a.dp.Requests`, etc. instead of `a.traceStore`, `a.log`, etc.

Key changes:
- Add `dp *dataplane.DataPlane` field to API struct
- `NewWithDataPlane(cfg, registry, dp)` sets `a.dp = dp`
- `handleTraces` uses `a.dp.Traces.Search()` instead of `a.traceStore.Recent()`
- `handleTraceByID` uses `a.dp.Traces.Get()` instead of `a.traceStore.Get()`
- `handleRequests` uses `a.dp.Requests.Query()` instead of `a.log.RecentFiltered()`
- `handleSLO` GET uses `a.dp.SLO.Status()` and `a.dp.SLO.Rules()`
- `handleSLO` PUT uses `a.dp.SLO.SetRules()`
- Deploy handlers use `a.dp.Config.ListDeploys()` / `AddDeploy()`
- View handlers use `a.dp.Config.ListViews()` / `SaveView()` / `DeleteView()`

- [ ] **Step 4: Update LoggingMiddleware for mode awareness**

Modify `pkg/gateway/logging.go`. Update `LoggingMiddlewareOpts` to include `*dataplane.DataPlane`:

```go
type LoggingMiddlewareOpts struct {
    Broadcaster RequestBroadcaster
    DataPlane   *dataplane.DataPlane  // if set, used for writes
    // Legacy fields — used when DataPlane is nil (backwards compat during migration)
    TraceStore  *TraceStore
    SLOEngine   *SLOEngine
}
```

In `LoggingMiddlewareWithOpts`, after capturing the request:
- If `opts.DataPlane != nil && opts.DataPlane.Mode == "production"`: skip `TraceStore.Add()` and `SLOEngine.Record()` (OTel SDK handles these). Call `opts.DataPlane.RequestW.Write(entry)` for request logging.
- Otherwise: existing behavior (write to in-memory stores via legacy fields or DataPlane memory wrappers).
- `Broadcaster.Broadcast()` always runs regardless of mode.

- [ ] **Step 5: Update main.go to build DataPlane**

Modify `cmd/gateway/main.go:226-266`. Replace direct store creation + setter injection with:

```go
mode := cfg.DataPlane.Mode
if mode == "" {
    mode = "local"
}

var dp *dataplane.DataPlane

switch mode {
case "local":
    requestLog := gateway.NewRequestLog(1000)
    requestStats := gateway.NewRequestStats()
    traceStore := gateway.NewTraceStore(500)
    sloEngine := gateway.NewSLOEngine(cfg.SLO.Rules)

    dp = &dataplane.DataPlane{
        Traces:   memory.NewTraceReader(traceStore),
        TraceW:   memory.NewTraceWriter(traceStore),
        Requests: memory.NewRequestReader(requestLog),
        RequestW: memory.NewRequestWriter(requestLog),
        Metrics:  memory.NewMetricReader(requestStats),
        MetricW:  memory.NewMetricWriter(requestStats),
        SLO:      memory.NewSLOStore(sloEngine),
        Config:   memory.NewConfigStore(cfg),
        Topology: memory.NewTopologyStore(nil),
        Mode:     "local",
    }

case "production":
    // Production wiring — implemented in Task 6
    log.Fatal("production mode not yet implemented")
}

adminAPI := admin.NewWithDataPlane(cfg, registry, dp)
```

- [ ] **Step 6: Run existing tests**

Run: `go test ./pkg/admin/ ./pkg/gateway/ ./cmd/gateway/ -v`
Expected: All existing tests PASS. The refactor preserves behavior.

- [ ] **Step 7: Commit**

```bash
git add pkg/config/config.go cmd/gateway/main.go pkg/admin/api.go pkg/gateway/logging.go
git commit -m "feat(dataplane): wire DataPlane into admin API and middleware

Config extension with DataPlane mode. Admin API accepts DataPlane struct.
LoggingMiddleware supports mode-dependent write paths. Local mode
unchanged in behavior."
```

---

## Task 4: Docker Compose & Database Schemas

**Files:**
- Create: `docker/docker-compose.prod.yml`
- Create: `docker/init/clickhouse/01-schema.sql`
- Create: `docker/init/postgres/01-schema.sql`
- Create: `docker/config/otel-collector.yml`
- Create: `docker/config/prometheus.yml`
- Create: `docker/config/recording-rules.yml`

- [ ] **Step 1: Create directory structure**

Run: `mkdir -p docker/init/clickhouse docker/init/postgres docker/config`

- [ ] **Step 2: Write ClickHouse schema**

Create `docker/init/clickhouse/01-schema.sql` with the `spans` table DDL from the design spec (Section 2) — MergeTree engine, partitioned by `(tenant_id, toYYYYMM(_date))`, ORDER BY `(service_name, action, start_time, trace_id)`, 30-day TTL, bloom/minmax/tokenbf indexes.

- [ ] **Step 3: Write PostgreSQL schema**

Create `docker/init/postgres/01-schema.sql` with all tables from design spec Section 3: `services`, `topology_edges`, `deploys`, `slo_rules`, `slo_rule_history`, `saved_views`, `config`.

- [ ] **Step 4: Write OTel Collector config**

Create `docker/config/otel-collector.yml` with OTLP receiver (gRPC 4317 + HTTP 4318), batch/attributes/filter processors, ClickHouse trace exporter, Prometheus metric exporter. Filter excludes favicon/HMR/assets. Attributes processor extracts tenant_id and org_id from request headers.

- [ ] **Step 5: Write Prometheus config**

Create `docker/config/prometheus.yml` — scrape OTel Collector's Prometheus exporter at `otel-collector:8889`.

Create `docker/config/recording-rules.yml` — SLO burn rate and error budget recording rules.

- [ ] **Step 6: Write docker-compose.prod.yml**

Create `docker/docker-compose.prod.yml` with services: clickhouse (24.3), postgres (16-alpine), prometheus (2.51), otel-collector (contrib 0.97), cloudmock (builds from parent). Init scripts mounted. Named volumes for persistence. CloudMock depends_on all backing stores. Environment variables set `CLOUDMOCK_DATAPLANE_MODE=production` and connection strings.

- [ ] **Step 7: Verify compose config is valid**

Run: `cd /Users/megan/work/neureaux/cloudmock && docker compose -f docker/docker-compose.prod.yml config`
Expected: Valid YAML, no errors.

- [ ] **Step 8: Commit**

```bash
git add docker/
git commit -m "feat(dataplane): add Docker Compose and database schemas

ClickHouse spans table, PostgreSQL metadata schema, OTel Collector
pipeline config, Prometheus with SLO recording rules."
```

---

## Task 5: ClickHouse Implementation

**Files:**
- Modify: `go.mod` — add `github.com/ClickHouse/clickhouse-go/v2`, `github.com/testcontainers/testcontainers-go`
- Create: `pkg/dataplane/clickhouse/client.go`
- Create: `pkg/dataplane/clickhouse/traces.go`
- Create: `pkg/dataplane/clickhouse/traces_test.go`
- Create: `pkg/dataplane/clickhouse/requests.go`
- Create: `pkg/dataplane/clickhouse/requests_test.go`

- [ ] **Step 1: Add ClickHouse dependency**

Run: `go get github.com/ClickHouse/clickhouse-go/v2 github.com/testcontainers/testcontainers-go`

- [ ] **Step 2: Write ClickHouse client**

Create `pkg/dataplane/clickhouse/client.go`:

```go
package clickhouse

import (
    "context"
    "fmt"

    "github.com/ClickHouse/clickhouse-go/v2"
    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
    "github.com/neureaux/cloudmock/pkg/config"
)

type Client struct {
    conn driver.Conn
}

func NewClient(ctx context.Context, cfg config.ClickHouseConfig) (*Client, error) {
    conn, err := clickhouse.Open(&clickhouse.Options{
        Addr: []string{cfg.Endpoint},
        Auth: clickhouse.Auth{Database: cfg.Database},
    })
    if err != nil {
        return nil, fmt.Errorf("clickhouse connect: %w", err)
    }
    if err := conn.Ping(ctx); err != nil {
        return nil, fmt.Errorf("clickhouse ping: %w", err)
    }
    return &Client{conn: conn}, nil
}

func (c *Client) Conn() driver.Conn { return c.conn }

func (c *Client) Close() error { return c.conn.Close() }
```

- [ ] **Step 3: Write trace writer test**

Create `pkg/dataplane/clickhouse/traces_test.go`. Use `testcontainers-go` to start a ClickHouse container. Run the schema init SQL. Write spans via `TraceWriter.WriteSpans()`, then read back via `TraceReader.Get()` and verify the parent/child tree is correctly reconstructed.

```go
func TestTraceWriteAndRead(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    ctx := context.Background()
    // Start ClickHouse container via testcontainers
    // Apply 01-schema.sql
    // Write 3 spans: root + 2 children
    // Read back via TraceReader.Get(traceID)
    // Verify: root has 2 children, correct services
}
```

- [ ] **Step 4: Run trace writer test to verify it fails**

Run: `go test ./pkg/dataplane/clickhouse/ -v -run TestTraceWriteAndRead`
Expected: FAIL — implementation doesn't exist.

- [ ] **Step 5: Write trace reader/writer implementation**

Create `pkg/dataplane/clickhouse/traces.go`:
- `TraceWriter.WriteSpans()` — batch insert into `spans` table
- `TraceReader.Get()` — `SELECT * FROM spans WHERE trace_id = ? ORDER BY start_time`, then assemble parent/child tree using `parent_span_id` linkage
- `TraceReader.Search()` — build `WHERE` clause from `TraceFilter` fields, return `TraceSummary` list
- `TraceReader.Timeline()` — query spans for trace, compute `start_offset_ms` relative to earliest span, sort by start time

- [ ] **Step 6: Run trace tests**

Run: `go test ./pkg/dataplane/clickhouse/ -v -run TestTraceWriteAndRead`
Expected: PASS

- [ ] **Step 7: Write request reader/writer (test + impl)**

Create `pkg/dataplane/clickhouse/requests_test.go` and `pkg/dataplane/clickhouse/requests.go`. `RequestWriter.Write()` inserts a span row (request entries are stored as spans). `RequestReader.Query()` translates `RequestFilter` into parameterized WHERE clauses. `RequestReader.GetByID()` queries by span_id.

- [ ] **Step 8: Run all ClickHouse tests**

Run: `go test ./pkg/dataplane/clickhouse/ -v -cover`
Expected: All PASS.

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum pkg/dataplane/clickhouse/
git commit -m "feat(dataplane): add ClickHouse trace and request storage

Implements TraceReader/Writer and RequestReader/Writer against spans
table. Trace reconstruction from denormalized spans. Integration tests
with testcontainers."
```

---

## Task 6: PostgreSQL Implementation

**Files:**
- Modify: `go.mod` — add `github.com/jackc/pgx/v5`
- Create: `pkg/dataplane/postgres/client.go`
- Create: `pkg/dataplane/postgres/slo.go`
- Create: `pkg/dataplane/postgres/slo_test.go`
- Create: `pkg/dataplane/postgres/config.go`
- Create: `pkg/dataplane/postgres/config_test.go`
- Create: `pkg/dataplane/postgres/topology.go`
- Create: `pkg/dataplane/postgres/topology_test.go`

- [ ] **Step 1: Add pgx dependency**

Run: `go get github.com/jackc/pgx/v5`

- [ ] **Step 2: Write PostgreSQL client**

Create `pkg/dataplane/postgres/client.go` with `NewPool(ctx, cfg) (*pgxpool.Pool, error)`.

- [ ] **Step 3: Write SLO store test**

Create `pkg/dataplane/postgres/slo_test.go`. Use testcontainers for PostgreSQL. Test:
- `SetRules()` inserts rules and creates history entries
- `Rules()` returns active rules
- `History()` returns change log ordered by time
- `SetRules()` again — verify old rules deactivated, new ones active, history has two entries

- [ ] **Step 4: Run SLO test to verify it fails**

Run: `go test ./pkg/dataplane/postgres/ -v -run TestSLOStore`
Expected: FAIL

- [ ] **Step 5: Write SLO store implementation**

Create `pkg/dataplane/postgres/slo.go`:
- `SetRules()` — in a transaction: deactivate existing rules (`UPDATE slo_rules SET active=false`), insert new rules, insert history entries for each change
- `Rules()` — `SELECT * FROM slo_rules WHERE active = true`
- `Status()` — reads rules, queries Prometheus for current metrics (or returns empty status if Prometheus unavailable)
- `History()` — `SELECT * FROM slo_rule_history ORDER BY changed_at DESC LIMIT $1`

- [ ] **Step 6: Run SLO tests**

Run: `go test ./pkg/dataplane/postgres/ -v -run TestSLOStore`
Expected: PASS

- [ ] **Step 7: Write config store (test + impl)**

Create `pkg/dataplane/postgres/config_test.go` and `config.go`. Test CRUD for deploys, views, services, and key-value config. Implementation uses standard pgx queries with `ON CONFLICT` for upserts.

- [ ] **Step 8: Write topology store (test + impl)**

Create `pkg/dataplane/postgres/topology_test.go` and `topology.go`. Test edge recording (upsert with count increment), upstream/downstream queries. Implementation uses recursive CTE for multi-hop blast radius if needed, or simple single-hop joins.

- [ ] **Step 9: Run all PostgreSQL tests**

Run: `go test ./pkg/dataplane/postgres/ -v -cover`
Expected: All PASS.

- [ ] **Step 10: Commit**

```bash
git add go.mod go.sum pkg/dataplane/postgres/
git commit -m "feat(dataplane): add PostgreSQL metadata, SLO, and topology storage

Implements SLOStore with transactional rule+history writes, ConfigStore
for deploys/views/services/config, TopologyStore for edge tracking.
Integration tests with testcontainers."
```

---

## Task 7: Prometheus Implementation

**Files:**
- Modify: `go.mod` — add `github.com/prometheus/client_golang`, `github.com/prometheus/common`
- Create: `pkg/dataplane/prometheus/client.go`
- Create: `pkg/dataplane/prometheus/metrics.go`
- Create: `pkg/dataplane/prometheus/metrics_test.go`

- [ ] **Step 1: Add Prometheus client dependency**

Run: `go get github.com/prometheus/client_golang github.com/prometheus/common`

- [ ] **Step 2: Write metrics test**

Create `pkg/dataplane/prometheus/metrics_test.go`. Mock the Prometheus HTTP API responses for `ServiceStats()` and `Percentiles()`. Test that PromQL queries are correctly constructed and results parsed.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/dataplane/prometheus/ -v`
Expected: FAIL

- [ ] **Step 4: Write Prometheus client and metrics implementation**

Create `pkg/dataplane/prometheus/client.go` — wraps `prometheus/client_golang/api/v1` for query execution.

Create `pkg/dataplane/prometheus/metrics.go`:
- `MetricReader.ServiceStats()` — runs `rate(http_requests_total{service="X"}[Ym])` for count, `rate(http_request_errors_total{...}[Ym])` for errors, `histogram_quantile()` for percentiles
- `MetricReader.Percentiles()` — runs `histogram_quantile(0.5|0.95|0.99, rate(http_request_duration_seconds{service="X",action="Y"}[Zm]))`
- `MetricWriter.Record()` — no-op in production mode (OTel SDK handles emission). Returns nil.

- [ ] **Step 5: Run metrics tests**

Run: `go test ./pkg/dataplane/prometheus/ -v -cover`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum pkg/dataplane/prometheus/
git commit -m "feat(dataplane): add Prometheus metrics reader

Implements MetricReader against Prometheus HTTP API. MetricWriter is
no-op in production (OTel SDK emits directly). Unit tests with mocked
HTTP responses."
```

---

## Task 8: OTel SDK Integration

**Files:**
- Modify: `go.mod` — add `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`
- Create: `pkg/dataplane/otel.go`

- [ ] **Step 1: Add OTel dependencies**

Run: `go get go.opentelemetry.io/otel go.opentelemetry.io/otel/sdk go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc`

- [ ] **Step 2: Write OTel initialization**

Create `pkg/dataplane/otel.go`:

```go
package dataplane

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
    "github.com/neureaux/cloudmock/pkg/config"
)

func InitTracer(ctx context.Context, cfg config.OTelConfig) (func(context.Context) error, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("otel exporter: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(cfg.ServiceName),
        )),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})

    return tp.Shutdown, nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./pkg/dataplane/...`
Expected: Clean compile.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum pkg/dataplane/otel.go
git commit -m "feat(dataplane): add OTel SDK trace provider initialization

Configures OTLP gRPC exporter to OTel Collector with batched span
export and W3C trace context propagation."
```

---

## Task 9: Production Mode Wiring

**Files:**
- Modify: `cmd/gateway/main.go` — complete the `"production"` case
- Modify: `pkg/gateway/logging.go` — add OTel span creation in production mode

- [ ] **Step 1: Wire production DataPlane in main.go**

Replace the `log.Fatal("production mode not yet implemented")` in main.go with:

```go
case "production":
    chClient, err := clickhouse.NewClient(ctx, cfg.DataPlane.ClickHouse)
    if err != nil {
        log.Fatalf("clickhouse: %v", err)
    }
    defer chClient.Close()

    pgPool, err := postgres.NewPool(ctx, cfg.DataPlane.PostgreSQL)
    if err != nil {
        log.Fatalf("postgres: %v", err)
    }
    defer pgPool.Close()

    promClient, err := prometheus.NewClient(cfg.DataPlane.Prometheus)
    if err != nil {
        log.Fatalf("prometheus: %v", err)
    }

    shutdown, err := dataplane.InitTracer(ctx, cfg.DataPlane.OTel)
    if err != nil {
        log.Fatalf("otel: %v", err)
    }
    defer shutdown(ctx)

    dp = &dataplane.DataPlane{
        Traces:   chImpl.NewTraceReader(chClient),
        TraceW:   chImpl.NewTraceWriter(chClient),
        Requests: chImpl.NewRequestReader(chClient),
        RequestW: chImpl.NewRequestWriter(chClient),
        Metrics:  promImpl.NewMetricReader(promClient),
        MetricW:  promImpl.NewMetricWriter(),
        SLO:      pgImpl.NewSLOStore(pgPool),
        Config:   pgImpl.NewConfigStore(pgPool),
        Topology: pgImpl.NewTopologyStore(pgPool),
        Mode:     "production",
    }
```

- [ ] **Step 2: Add OTel span creation to LoggingMiddleware**

In production mode, the middleware creates OTel spans instead of writing to TraceStore:

```go
if opts.DataPlane != nil && opts.DataPlane.Mode == "production" {
    tracer := otel.Tracer("cloudmock-gateway")
    ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, svcName+"/"+action))
    defer span.End()
    // Set span attributes: service, action, tenant_id, status_code, etc.
    span.SetAttributes(
        attribute.String("service.name", svcName),
        attribute.String("action", action),
        attribute.String("tenant_id", tenantID),
        attribute.Int("http.status_code", rec.statusCode),
    )
    r = r.WithContext(ctx)
}
```

- [ ] **Step 3: Verify local mode still works**

Run: `go test ./pkg/admin/ ./pkg/gateway/ ./cmd/gateway/ -v`
Expected: All PASS. Production mode is only activated by config.

- [ ] **Step 4: Commit**

```bash
git add cmd/gateway/main.go pkg/gateway/logging.go
git commit -m "feat(dataplane): wire production mode with ClickHouse, PostgreSQL, Prometheus

Production DataPlane connects to all backing stores. LoggingMiddleware
emits OTel spans in production mode. Local mode unchanged."
```

---

## Task 10: Config Import Tool

**Files:**
- Create: `cmd/configimport/main.go`

- [ ] **Step 1: Write config import tool**

Create `cmd/configimport/main.go`:

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"

    "github.com/neureaux/cloudmock/pkg/config"
    "github.com/neureaux/cloudmock/pkg/dataplane"
    "github.com/neureaux/cloudmock/pkg/dataplane/postgres"
)

func main() {
    configPath := flag.String("config", "cloudmock.yml", "path to cloudmock.yml")
    pgURL := flag.String("pg-url", "", "PostgreSQL connection URL (required)")
    flag.Parse()

    if *pgURL == "" {
        log.Fatal("--pg-url is required")
    }

    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    ctx := context.Background()
    pool, err := postgres.NewPool(ctx, config.PostgreSQLConfig{URL: *pgURL})
    if err != nil {
        log.Fatalf("postgres: %v", err)
    }
    defer pool.Close()

    configStore := postgres.NewConfigStore(pool)
    sloStore := postgres.NewSLOStore(pool)

    // Import SLO rules
    if len(cfg.SLO.Rules) > 0 {
        if err := sloStore.SetRules(ctx, cfg.SLO.Rules); err != nil {
            log.Fatalf("import SLO rules: %v", err)
        }
        fmt.Printf("Imported %d SLO rules\n", len(cfg.SLO.Rules))
    }

    // Import full config as key-value
    if err := configStore.SetConfig(ctx, cfg); err != nil {
        log.Fatalf("import config: %v", err)
    }
    fmt.Println("Config imported successfully")
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/configimport/`
Expected: Clean compile.

- [ ] **Step 3: Commit**

```bash
git add cmd/configimport/
git commit -m "feat(dataplane): add config import tool for YAML-to-PostgreSQL migration

Reads cloudmock.yml and imports SLO rules and config into PostgreSQL.
One-time use when migrating from local to production mode."
```

---

## Task 11: Integration Test — Local vs Production Parity

**Files:**
- Create: `pkg/dataplane/dataplane_test.go`

- [ ] **Step 1: Write parity test suite**

Create `pkg/dataplane/dataplane_test.go`. Define a test suite that runs against any `DataPlane` instance:

```go
func runParityTests(t *testing.T, dp *dataplane.DataPlane) {
    t.Run("TraceWriteAndRead", func(t *testing.T) { ... })
    t.Run("RequestWriteAndQuery", func(t *testing.T) { ... })
    t.Run("SLORulesRoundtrip", func(t *testing.T) { ... })
    t.Run("ConfigStoreRoundtrip", func(t *testing.T) { ... })
    t.Run("TopologyEdgeUpsert", func(t *testing.T) { ... })
    t.Run("DeploysCRUD", func(t *testing.T) { ... })
    t.Run("ViewsCRUD", func(t *testing.T) { ... })
}

func TestLocalDataPlane(t *testing.T) {
    dp := buildLocalDataPlane()
    runParityTests(t, dp)
}

func TestProductionDataPlane(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    dp := buildProductionDataPlane(t) // testcontainers for CH + PG + mock Prom
    runParityTests(t, dp)
}
```

- [ ] **Step 2: Run parity tests — local**

Run: `go test ./pkg/dataplane/ -v -run TestLocalDataPlane`
Expected: PASS

- [ ] **Step 3: Run parity tests — production**

Run: `go test ./pkg/dataplane/ -v -run TestProductionDataPlane`
Expected: PASS (requires Docker for testcontainers)

- [ ] **Step 4: Commit**

```bash
git add pkg/dataplane/dataplane_test.go
git commit -m "test(dataplane): add parity test suite for local vs production mode

Same test suite runs against both in-memory and persistent backends.
Verifies identical behavior for all storage interfaces."
```

---

## Task 12: Makefile & Documentation

**Files:**
- Modify: `Makefile`
- Modify: `docker/gateway.Dockerfile` (if production mode needs env vars)

- [ ] **Step 1: Add Makefile targets**

Add to `Makefile`:

```makefile
.PHONY: dev-prod
dev-prod: ## Start production data plane (Docker Compose)
	docker compose -f docker/docker-compose.prod.yml up -d

.PHONY: dev-prod-down
dev-prod-down: ## Stop production data plane
	docker compose -f docker/docker-compose.prod.yml down

.PHONY: config-import
config-import: ## Import cloudmock.yml into PostgreSQL
	go run cmd/configimport/main.go --config cloudmock.yml --pg-url "postgres://cloudmock:cloudmock@localhost:5432/cloudmock"

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	go test -v -cover ./pkg/dataplane/... -count=1

.PHONY: test-unit
test-unit: ## Run unit tests only (no Docker)
	go test -v -short -cover ./...
```

- [ ] **Step 2: Run make test-unit**

Run: `make test-unit`
Expected: All unit tests PASS (integration tests skipped via `-short`).

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(dataplane): add Makefile targets for production mode

dev-prod, dev-prod-down, config-import, test-integration, test-unit."
```

---

## Task Summary

| Task | What it builds | Depends on |
|------|---------------|------------|
| 1 | Storage interfaces + DataPlane struct | — |
| 2 | Memory wrappers (existing stores → interfaces) | 1 |
| 3 | Config extension + wiring (admin API, middleware, main.go) | 1, 2 |
| 4 | Docker Compose + database schemas | — |
| 5 | ClickHouse implementation | 1, 4 |
| 6 | PostgreSQL implementation | 1, 4 |
| 7 | Prometheus implementation | 1 |
| 8 | OTel SDK integration | — |
| 9 | Production mode wiring | 3, 5, 6, 7, 8 |
| 10 | Config import tool | 6 |
| 11 | Parity test suite | 2, 5, 6, 7 |
| 12 | Makefile + docs | 4, 9 |
