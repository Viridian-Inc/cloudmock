# Phase 3: Production Data Plane — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 3 of 6 (CloudMock Console)
**Depends on:** Phase 0-2 (complete)

---

## Overview

Phase 3 replaces CloudMock Console's in-memory storage with a production-grade data plane. The system operates in two modes — `local` (existing in-memory stores, zero config) and `production` (ClickHouse, Prometheus, PostgreSQL, OTel Collector) — with identical APIs and UI.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Deployment target | Docker Compose now, Kubernetes later | Working production-mode setup without K8s overhead; storage interfaces are deployment-agnostic |
| OTel integration | Sidecar replacement | In production mode, LoggingMiddleware stops writing to stores; OTel SDK emits spans; Collector routes to backends |
| PostgreSQL scope | Metadata + state + config | Replaces cloudmock.yml in production; enables multi-user config changes with audit trails |
| Migration story | Config portability only | YAML → PostgreSQL import for SLO rules, views, topology; telemetry data doesn't migrate |
| ClickHouse schema | Spans-first denormalized | Matches OTel Collector's native export; traces reconstructed at query time |
| Storage abstraction | Interface-per-store with DataPlane coordinator | Narrow interfaces for testing, single struct for wiring, mix-and-match during rollout |

---

## 1. Storage Interface Hierarchy

Seven interfaces in `pkg/dataplane/`, organized by domain. Each gets a `memory` implementation (wrapping existing code) and a `persistent` implementation.

### Trace interfaces

```go
// pkg/dataplane/traces.go
type TraceReader interface {
    Get(ctx context.Context, traceID string) (*TraceContext, error)
    Search(ctx context.Context, filter TraceFilter) ([]TraceSummary, error)
    Timeline(ctx context.Context, traceID string) ([]TimelineSpan, error)
}

type TraceWriter interface {
    WriteSpans(ctx context.Context, spans []*Span) error
}
```

### Request interfaces

```go
// pkg/dataplane/requests.go
type RequestReader interface {
    Query(ctx context.Context, filter RequestFilter) ([]RequestEntry, error)
    GetByID(ctx context.Context, id string) (*RequestEntry, error)
}

type RequestWriter interface {
    Write(ctx context.Context, entry RequestEntry) error
}
```

### Metric interfaces

```go
// pkg/dataplane/metrics.go
type MetricReader interface {
    ServiceStats(ctx context.Context, service string, window time.Duration) (*ServiceMetrics, error)
    Percentiles(ctx context.Context, service, action string, window time.Duration) (*LatencyPercentiles, error)
}

type MetricWriter interface {
    Record(ctx context.Context, service, action string, latencyMs float64, statusCode int) error
}
```

### SLO store

```go
// pkg/dataplane/slo.go
type SLOStore interface {
    Rules(ctx context.Context) ([]SLORule, error)
    SetRules(ctx context.Context, rules []SLORule) error
    Status(ctx context.Context) (*SLOStatus, error)
    History(ctx context.Context, limit int) ([]SLORuleChange, error)
}
```

### Config store

```go
// pkg/dataplane/config.go
// Config refers to the existing config.Config type from pkg/config/config.go
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

### Topology store

```go
// pkg/dataplane/topology.go
type TopologyStore interface {
    GetTopology(ctx context.Context) (*TopologyGraph, error)
    RecordEdge(ctx context.Context, edge ObservedEdge) error
    Upstream(ctx context.Context, service string) ([]string, error)
    Downstream(ctx context.Context, service string) ([]string, error)
}
```

### DataPlane coordinator

```go
// pkg/dataplane/dataplane.go
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

Read/write interfaces are split so the middleware only needs writers and the admin API only needs readers. `TraceWriter.WriteSpans` takes a batch of spans (not traces) to match OTel Collector output. `TraceReader.Get` returns an assembled `*TraceContext` with children — the implementation handles reconstruction.

---

## 2. ClickHouse Schema

Spans-first denormalized table matching OTel Collector's native export format.

```sql
CREATE TABLE spans (
    -- Identity
    trace_id         FixedString(32),
    span_id          FixedString(16),
    parent_span_id   FixedString(16),

    -- Timing
    start_time       DateTime64(9, 'UTC'),
    end_time         DateTime64(9, 'UTC'),
    duration_ns      UInt64,

    -- Service identity
    service_name     LowCardinality(String),
    action           LowCardinality(String),
    method           LowCardinality(String),
    path             String,
    status_code      UInt16,
    error            String,

    -- Tenant context
    tenant_id        String,
    org_id           String,
    user_id          String,

    -- Resource metrics
    mem_alloc_kb     Float64,
    goroutines       UInt32,

    -- Structured metadata
    metadata         Map(String, String),

    -- Request/response capture
    request_headers  Map(String, String),
    request_body     String,
    response_body    String,

    -- Partition/sort infrastructure
    _date            Date DEFAULT toDate(start_time)
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(_date))
ORDER BY (service_name, action, start_time, trace_id)
TTL _date + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- Secondary indexes
ALTER TABLE spans ADD INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE spans ADD INDEX idx_error error TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 4;
ALTER TABLE spans ADD INDEX idx_status status_code TYPE minmax GRANULARITY 4;
ALTER TABLE spans ADD INDEX idx_user_id user_id TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE spans ADD INDEX idx_org_id org_id TYPE bloom_filter(0.01) GRANULARITY 4;
```

### Query pattern alignment

| Query | Mechanism | Performance |
|-------|-----------|-------------|
| Get trace by ID | bloom_filter on trace_id | O(1) granule skip |
| Recent spans for service | Primary key prefix (service_name, action, start_time) | Primary index |
| Filter by tenant + time | Partition pruning (tenant_id) + ORDER BY (start_time) | Partition prune |
| Error search | tokenbf on error column | Token bloom skip |
| Latency percentiles | `quantile(0.99)(duration_ns) GROUP BY service` | Scan within partition |
| Trace reconstruction | `WHERE trace_id = X`, assemble parent/child in Go | Bloom → few granules |

---

## 3. PostgreSQL Schema

```sql
CREATE TABLE services (
    name             TEXT PRIMARY KEY,
    service_type     TEXT NOT NULL,
    group_name       TEXT,
    description      TEXT,
    owner            TEXT,
    repo_url         TEXT,
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE topology_edges (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_service   TEXT NOT NULL REFERENCES services(name),
    target_service   TEXT NOT NULL REFERENCES services(name),
    edge_type        TEXT NOT NULL,
    first_seen       TIMESTAMPTZ NOT NULL,
    last_seen        TIMESTAMPTZ NOT NULL,
    request_count    BIGINT DEFAULT 0,
    UNIQUE (source_service, target_service, edge_type)
);

CREATE TABLE deploys (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service          TEXT NOT NULL REFERENCES services(name),
    version          TEXT NOT NULL,
    commit_sha       TEXT,
    author           TEXT,
    description      TEXT,
    deployed_at      TIMESTAMPTZ DEFAULT now(),
    metadata         JSONB DEFAULT '{}'
);

CREATE TABLE slo_rules (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service          TEXT NOT NULL,
    action           TEXT NOT NULL DEFAULT '*',
    route            TEXT,
    tenant_tier      TEXT,
    p50_ms           DOUBLE PRECISION,
    p95_ms           DOUBLE PRECISION,
    p99_ms           DOUBLE PRECISION,
    error_rate       DOUBLE PRECISION,
    annotation       TEXT,
    active           BOOLEAN DEFAULT true,
    created_at       TIMESTAMPTZ DEFAULT now(),
    created_by       TEXT
);

CREATE TABLE slo_rule_history (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id          UUID NOT NULL REFERENCES slo_rules(id),
    change_type      TEXT NOT NULL,
    old_values       JSONB,
    new_values       JSONB,
    changed_by       TEXT,
    changed_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE saved_views (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    filters          JSONB NOT NULL,
    created_by       TEXT,
    created_at       TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE config (
    key              TEXT PRIMARY KEY,
    value            JSONB NOT NULL,
    updated_by       TEXT,
    updated_at       TIMESTAMPTZ DEFAULT now()
);

-- Note: incidents table is deferred to Phase 4 (Intelligence Layer).
-- slo_rules.service has no FK to services(name) intentionally — supports wildcard "*" rules.
```

---

## 4. Prometheus Metrics & OTel Collector Pipeline

### Metrics emitted by OTel SDK

```
http_request_duration_seconds{service, action, method, status_code, tenant_id}  histogram
http_requests_total{service, action, method, status_code, tenant_id}            counter
http_request_errors_total{service, action, error_type, tenant_id}               counter
process_memory_alloc_bytes{service}                                             gauge
process_goroutines{service}                                                     gauge

# Computed by Prometheus recording rules
slo_error_budget_remaining{service, action}                                     gauge
slo_burn_rate{service, action, window}                                          gauge
```

### OTel Collector config

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 1024
  attributes:
    actions:
      - key: tenant_id
        from_attribute: http.request.header.x-tenant-id
        action: upsert
      - key: org_id
        from_attribute: http.request.header.x-enterprise-id
        action: upsert
  filter:
    spans:
      exclude:
        match_type: regexp
        attributes:
          - key: http.target
            value: "^/(favicon|__hmr|assets/)"

exporters:
  clickhouse:
    endpoint: tcp://clickhouse:9000
    database: cloudmock
    traces_table_name: spans
    timeout: 10s
    retry_on_failure:
      enabled: true
  prometheus:
    endpoint: 0.0.0.0:8889
    namespace: cloudmock

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, attributes, filter]
      exporters: [clickhouse]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus]
```

---

## 5. OTel SDK Integration & Mode Switching

### OTel SDK initialization

New `pkg/dataplane/otel.go` provides `InitTracer(ctx, cfg) (shutdown, error)` that configures the OTel trace provider with gRPC exporter to the Collector.

### Mode switching in main.go

```go
mode := cfg.DataPlane.Mode // "local" (default) or "production"

switch mode {
case "production":
    chConn := clickhouse.Open(cfg.DataPlane.ClickHouse)
    pgPool := pgxpool.Connect(ctx, cfg.DataPlane.PostgreSQL)
    promClient := promapi.NewClient(cfg.DataPlane.Prometheus)
    shutdown, _ := dataplane.InitTracer(ctx, cfg.DataPlane.OTel)
    defer shutdown(ctx)

    dp = &dataplane.DataPlane{
        Traces:   clickhouse.NewTraceReader(chConn),
        TraceW:   clickhouse.NewTraceWriter(chConn),
        Requests: clickhouse.NewRequestReader(chConn),
        RequestW: clickhouse.NewRequestWriter(chConn),
        Metrics:  prometheus.NewMetricReader(promClient),
        MetricW:  prometheus.NewMetricWriter(promClient),
        SLO:      postgres.NewSLOStore(pgPool),
        Config:   postgres.NewConfigStore(pgPool),
        Topology: postgres.NewTopologyStore(pgPool),
        Mode:     "production",
    }

case "local":
    dp = &dataplane.DataPlane{
        Traces:   memory.NewTraceReader(traceStore),
        TraceW:   memory.NewTraceWriter(traceStore),
        Requests: memory.NewRequestReader(requestLog),
        RequestW: memory.NewRequestWriter(requestLog),
        Metrics:  memory.NewMetricReader(requestStats),
        MetricW:  memory.NewMetricWriter(requestStats),
        SLO:      memory.NewSLOStore(sloEngine),
        Config:   memory.NewConfigStore(cfg),
        Topology: memory.NewTopologyStore(iacTopology),
        Mode:     "local",
    }
}

adminAPI := admin.NewWithDataPlane(cfg, registry, dp)
```

### LoggingMiddleware changes by mode

| Concern | Local mode | Production mode |
|---------|-----------|----------------|
| Trace creation | `traceStore.Add(trace)` | OTel SDK `tracer.Start()` → Collector → ClickHouse |
| Metrics | `stats.Increment()` + `sloEngine.Record()` | OTel SDK counter/histogram → Collector → Prometheus. Middleware does **not** call `MetricW.Record()` — it is a no-op in production. The mode check (`dp.Mode == "production"`) skips the metric write path entirely. |
| Request logging | `log.Add(entry)` | `dp.RequestW.Write(entry)` → ClickHouse |
| SSE broadcast | `broadcaster.Broadcast()` | `broadcaster.Broadcast()` (unchanged) |

**Out of scope for Phase 3:** Log store (Loki/Elasticsearch). The product spec's Data Plane (Section 4.2) lists a Log Store, but structured request logs are captured as spans in ClickHouse via `request_body`/`response_body` columns. A dedicated log pipeline is deferred until a use case requires log-specific querying beyond what ClickHouse provides.

### Config extension

```yaml
dataplane:
  mode: local  # or "production"
  clickhouse:
    endpoint: clickhouse:9000
    database: cloudmock
  postgresql:
    url: postgres://cloudmock:password@postgres:5432/cloudmock
  prometheus:
    url: http://prometheus:9090
  otel:
    collector_endpoint: otel-collector:4317
    service_name: cloudmock-gateway
```

---

## 6. Docker Compose

`docker/docker-compose.prod.yml` runs ClickHouse, PostgreSQL, Prometheus, OTel Collector, and CloudMock.

Init scripts in `docker/init/`:
- `clickhouse/01-schema.sql` — spans table DDL
- `postgres/01-schema.sql` — all PostgreSQL tables

### Config import tool

`cmd/configimport/main.go` reads `cloudmock.yml` and imports SLO rules, services, saved views, and config key-value pairs into PostgreSQL. One-time use when migrating from local to production mode.

### Startup flow

1. `docker compose -f docker/docker-compose.prod.yml up -d` — starts backing stores
2. `go run cmd/configimport/main.go --config cloudmock.yml` — seeds PostgreSQL (optional)
3. CloudMock starts with `CLOUDMOCK_DATAPLANE_MODE=production`

---

## 7. Memory Implementation Wrappers

Package `pkg/dataplane/memory/` provides thin adapters around existing concrete types (`TraceStore`, `RequestLog`, `RequestStats`, `SLOEngine`) to satisfy the new interfaces. Each wrapper:

1. Adds `context.Context` parameter (ignored in local mode)
2. Adds `error` return (always nil in local mode)
3. Delegates to the existing method

Zero logic duplication. Existing circular buffer code stays untouched.

---

## 8. Persistent Implementations

**`pkg/dataplane/clickhouse/`** — `TraceReader`, `TraceWriter`, `RequestReader`, `RequestWriter`
- Uses `clickhouse-go/v2` driver
- `TraceReader.Get()` queries `SELECT * FROM spans WHERE trace_id = ?` and assembles parent/child tree in Go
- `TraceWriter.WriteSpans()` uses batch insert
- `RequestReader.Query()` translates `RequestFilter` into parameterized `WHERE` clauses

**`pkg/dataplane/postgres/`** — `SLOStore`, `ConfigStore`, `TopologyStore`
- Uses `pgx/v5` with connection pooling
- `SLOStore.SetRules()` wraps rule insert + history insert in a transaction
- `ConfigStore.GetConfig()` assembles `Config` from `SELECT * FROM config`
- `TopologyStore.RecordEdge()` does `INSERT ... ON CONFLICT DO UPDATE`

**`pkg/dataplane/prometheus/`** — `MetricReader`, `MetricWriter`
- `MetricReader` wraps Prometheus HTTP API for queries
- `MetricWriter` is a no-op in production — OTel SDK emits metrics directly to Collector

---

## 9. Testing Strategy

**Unit tests per implementation:**
- `memory/*_test.go` — verify wrappers delegate correctly
- `clickhouse/*_test.go` — `testcontainers-go` with real ClickHouse; span insert → trace reconstruction roundtrip
- `postgres/*_test.go` — `testcontainers-go` with PostgreSQL; SLO CRUD + history, config import, edge upsert
- `prometheus/*_test.go` — mock HTTP API responses

**Integration test (`pkg/dataplane/dataplane_test.go`):**
- Build both `local` and `production` DataPlane instances
- Run identical test suite against both — verify same behavior for all Reader interface methods

**Admin API tests:**
- Refactor existing `api_test.go` to accept interfaces instead of concrete stores
- Run against both memory and persistent backends

---

## File Layout

```
pkg/dataplane/
├── dataplane.go        # DataPlane struct, factory functions
├── traces.go           # TraceReader, TraceWriter interfaces
├── requests.go         # RequestReader, RequestWriter interfaces
├── metrics.go          # MetricReader, MetricWriter interfaces
├── slo.go              # SLOStore interface
├── config.go           # ConfigStore interface
├── topology.go         # TopologyStore interface
├── otel.go             # OTel SDK initialization
├── memory/
│   ├── traces.go       # wraps gateway.TraceStore
│   ├── requests.go     # wraps gateway.RequestLog
│   ├── metrics.go      # wraps gateway.RequestStats
│   ├── slo.go          # wraps gateway.SLOEngine
│   ├── config.go       # wraps config.Config
│   └── topology.go     # wraps admin.IaCTopologyConfig
├── clickhouse/
│   ├── traces.go       # TraceReader + TraceWriter
│   ├── requests.go     # RequestReader + RequestWriter
│   └── client.go       # connection management
├── postgres/
│   ├── slo.go          # SLOStore
│   ├── config.go       # ConfigStore
│   ├── topology.go     # TopologyStore
│   └── client.go       # connection pool
└── prometheus/
    ├── metrics.go       # MetricReader + MetricWriter
    └── client.go        # HTTP API wrapper

cmd/configimport/
└── main.go             # YAML → PostgreSQL import tool

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
