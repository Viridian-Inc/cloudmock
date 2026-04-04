# CloudMock Cloud -- AWS Observability Product Design Spec

## Overview

CloudMock Cloud is an observability product for real AWS infrastructure that reuses the existing CloudMock devtools UI. Customers install a lightweight agent (Go SDK middleware or reverse proxy sidecar) that captures AWS API calls and exports them as OpenTelemetry spans. Data flows to a hosted ingest service on AWS, stored in Postgres + TimescaleDB. The same 23 devtools views display both local emulation data and production AWS data through a unified query layer with an environment picker.

**Goal:** Let any team see their real AWS topology, traces, metrics, and errors in the same devtools UI they already use for local development -- with one environment toggle.

## Product Family

```
CloudMock CLI (free)         CloudMock Platform ($0.50/10K)    CloudMock Cloud ($10/seat + usage)
─────────────────            ────────────────────────          ──────────────────────────────────
Local binary                 Hosted on Fly                     Hosted on AWS
100 AWS services             Hosted emulation endpoints        Real AWS observability
Devtools at :4500            API keys, teams, audit            Agent + ingest + dashboard
In-memory storage            Fly Postgres                      RDS Postgres + TimescaleDB
```

All three products share the same devtools UI. The only difference is the data source.

## Architecture

### Data Flow

```
Local CloudMock (:4566)  ──┐                              ┌── Devtools UI
                            ├──► Unified Query Layer ──────┤   (23 views)
Cloud Agent (real AWS)   ──┘    (merges by environment)    └── Environment Picker
       │
       │ OTLP/HTTP
       ▼
Ingest Service (ECS Fargate)
       │
       ▼
RDS Postgres + TimescaleDB
```

### Agent

Two deployment modes, both producing the same OTel span format:

**Go SDK Middleware** (wraps AWS SDK transport):
- Package: `github.com/Viridian-Inc/cloudmock-agent/sdk`
- Wraps `http.RoundTripper` to capture AWS API calls
- Zero infrastructure, one import + one line of config
- Based on existing `sdk/interceptor.go` pattern

**Reverse Proxy Sidecar** (language-agnostic):
- Binary: `cloudmock-agent`
- Listens on `:4577`, forwards to real AWS while recording
- Set `AWS_ENDPOINT_URL=http://localhost:4577`
- Works with Python, Node, Java, Rust, any SDK
- Based on existing `pkg/proxy/proxy.go` pattern
- Runs as ECS sidecar, K8s sidecar, or local process

**Span Attributes (both modes):**
- `aws.service` -- s3, dynamodb, sqs, etc.
- `aws.action` -- PutObject, GetItem, SendMessage
- `aws.region`
- `aws.account_id`
- `aws.request_id`
- `aws.error_code` -- if the request failed
- `cloudmock.environment` -- local, staging, production (configurable)
- `cloudmock.source` -- local, agent-sdk, agent-proxy
- `cloudmock.org_id` -- for multi-tenant routing in the ingest service
- `cloudmock.app_id` -- scoped to a Platform app

**Agent authentication:** API key in the `X-Api-Key` header on OTLP export requests. Reuses the same API keys from the Platform product.

### Separation from Core Emulator

The agent is a separate module/repo. It does NOT depend on the CloudMock emulator:

```
cloudmock/                  (emulator -- unchanged)
cloudmock-agent/            (new -- tiny, ~500 lines)
  cmd/agent/main.go         proxy sidecar binary
  sdk/observer.go           Go SDK middleware
  sdk/transport.go          OTel-exporting RoundTripper
  go.mod                    minimal deps (otel SDK only)
cloudmock-cloud/            (new -- ingest service)
  cmd/ingest/main.go        OTLP receiver + TimescaleDB writer
  internal/store/            TimescaleDB query backends
  go.mod
```

**What changes in the main cloudmock repo (minimal):**
- Devtools: environment picker dropdown in the source bar
- Devtools: source badges on traces/requests ("local" / "prod")
- Admin API: `TraceBackend`, `MetricsBackend`, `TopologyBackend` interfaces extracted from existing in-memory stores
- Admin API: optional cloud backend that queries TimescaleDB (behind feature flag, only enabled when `CLOUDMOCK_CLOUD_URL` is set)

No changes to emulation, services, gateway, or routing.

### Ingest Service (AWS)

**Receives OTLP/HTTP spans, validates auth, writes to TimescaleDB.**

Components:
- OTLP/HTTP receiver (port 4318, same protocol as local CloudMock)
- API key validation (hash lookup against Platform's api_keys table)
- Span enrichment: org_id, app_id from API key; environment from span attribute
- TimescaleDB writer: batch inserts every 1 second or 1000 spans

**Storage Schema (TimescaleDB):**

```sql
-- Raw spans (hypertable, partitioned by time)
CREATE TABLE spans (
    time           TIMESTAMPTZ NOT NULL,
    trace_id       TEXT NOT NULL,
    span_id        TEXT NOT NULL,
    parent_span_id TEXT,
    org_id         UUID NOT NULL,
    app_id         UUID,
    environment    TEXT NOT NULL,
    source         TEXT NOT NULL,
    service        TEXT NOT NULL,
    action         TEXT NOT NULL,
    region         TEXT,
    account_id     TEXT,
    request_id     TEXT,
    duration_ms    DOUBLE PRECISION NOT NULL,
    status_code    INT,
    error_code     TEXT,
    attributes     JSONB
);
SELECT create_hypertable('spans', 'time');

-- Continuous aggregates for metrics (auto-computed)
CREATE MATERIALIZED VIEW service_metrics_1m
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', time) AS bucket,
    org_id,
    environment,
    service,
    action,
    COUNT(*) AS request_count,
    AVG(duration_ms) AS avg_ms,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_ms) AS p50_ms,
    percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) AS p95_ms,
    percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms) AS p99_ms,
    COUNT(*) FILTER (WHERE error_code IS NOT NULL) AS error_count
FROM spans
GROUP BY bucket, org_id, environment, service, action;

-- Topology edges (materialized view, refreshed every 60s)
CREATE MATERIALIZED VIEW topology_edges AS
SELECT DISTINCT ON (org_id, environment, parent_service, child_service)
    org_id,
    environment,
    s1.service AS parent_service,
    s2.service AS child_service,
    COUNT(*) AS call_count,
    AVG(s2.duration_ms) AS avg_latency_ms
FROM spans s2
JOIN spans s1 ON s2.parent_span_id = s1.span_id AND s2.trace_id = s1.trace_id
WHERE s2.time > now() - interval '1 hour'
GROUP BY org_id, environment, s1.service, s2.service;
```

**Data retention:** Configurable per org via the Platform settings page. Defaults: 30 days for raw spans, 90 days for aggregated metrics. TimescaleDB compression policy applied after 7 days. Drop chunks policy for expired data.

### AWS Infrastructure

```
us-east-1
├── ALB (TLS termination)
│   └── ingest.cloudmock.app
├── ECS Fargate
│   ├── ingest service (2 tasks, auto-scaling)
│   └── query service (2 tasks, serves admin API for cloud data)
├── RDS Postgres 16 + TimescaleDB
│   └── db.t4g.medium (upgradeable)
├── S3 (optional long-term trace archive for enterprise)
└── CloudWatch (monitoring the monitoring)
```

**Cost baseline:**
- RDS db.t4g.medium: ~$50/mo
- ECS Fargate (4 tasks): ~$60/mo
- ALB: ~$20/mo
- Total: ~$130/mo baseline

### Unified Query Layer

The admin API gets backend interfaces that abstract the data source:

```go
type TraceBackend interface {
    Query(ctx context.Context, filter TraceFilter) ([]Trace, error)
    Get(ctx context.Context, traceID string) (*Trace, error)
}

type MetricsBackend interface {
    Query(ctx context.Context, filter MetricsFilter) ([]MetricPoint, error)
    Timeline(ctx context.Context, service, action string, window time.Duration) ([]TimelinePoint, error)
}

type TopologyBackend interface {
    Get(ctx context.Context, window time.Duration) (*TopologyGraph, error)
}
```

Three implementations:
- `MemoryBackend` -- existing in-memory stores (local mode, no change)
- `CloudBackend` -- queries TimescaleDB via the cloud query service
- `MergedBackend` -- queries both, merges results, tags each with source/environment

The admin API adds `?env=` query parameter to all endpoints. When set, it filters to that environment. When "All", the MergedBackend combines everything.

### Devtools UI Changes

Three small additions to the existing Preact SPA:

1. **Environment picker** -- dropdown in the source bar. Options: "Local" (default), plus any cloud environments detected. Shows a colored dot (green=local, blue=staging, orange=production).

2. **Source badges** -- small tag on each trace/request row showing "local" or "prod" with appropriate color. Appears in Activity, Traces, and Metrics views.

3. **Topology comparison overlay** -- when viewing "All" environments, the topology map shows edges from both local and production. Each edge displays both latencies so you can compare emulation vs. real AWS performance at a glance.

No new views. No changes to the 23 existing view components beyond adding the environment/source filtering.

## Pricing

| Product | Price | Free Tier |
|---------|-------|-----------|
| CLI | Free forever | Unlimited local |
| Platform | $0.50 per 10K requests | 1K requests/mo |
| Cloud | $10/seat/mo + $0.50 per million events | 100K events/mo |
| Enterprise | Custom | Dedicated infra, BAA, SLA |

**Cloud pricing example:**
- Team of 5 engineers, 20M events/month
- Base: 5 x $10 = $50/mo
- Events: (20M - 0.1M free) / 1M x $0.50 = $9.95/mo
- Total: ~$60/mo

**Compared to Datadog:**
- 5 hosts x $23/mo (infra) + $31/mo (APM) = $270/mo
- CloudMock Cloud: $60/mo (78% cheaper)

## Implementation Phases

### Phase 1: Agent (cloudmock-agent repo)
- Go SDK middleware (observer RoundTripper)
- Reverse proxy sidecar binary
- OTLP export with AWS-specific span attributes
- API key authentication on export
- Tests with mock OTLP collector

### Phase 2: Ingest + Storage (cloudmock-cloud repo)
- OTLP/HTTP receiver with auth validation
- TimescaleDB schema (spans hypertable, continuous aggregates, topology view)
- Batch writer (1s or 1000 spans)
- Query service implementing TraceBackend, MetricsBackend, TopologyBackend
- ECS Fargate deployment on AWS
- Terraform for AWS infrastructure

### Phase 3: Unified Devtools (cloudmock repo, minimal changes)
- Extract backend interfaces from existing admin API
- CloudBackend implementation calling the query service
- MergedBackend combining local + cloud
- Environment picker in devtools source bar
- Source badges on traces/requests
- Topology comparison overlay

### Phase 4: Polish + Enterprise
- Data retention controls in Platform settings (applies to Cloud storage)
- S3 archive for long-term trace storage
- VPC peering for enterprise customers
- Multi-region ingest (eu-west-1, ap-southeast-1)
- SOC2 / HIPAA BAA
