# CloudMock Console — Product Specification

**Date:** 2026-03-22
**Status:** Draft
**Author:** Engineering
**Version:** 1.0

## Executive Summary

CloudMock Console is a tenant-aware observability control plane that answers six questions about every request: what happened, where it happened, why it happened, who was impacted, what changed, and what should be done next.

It combines service topology visualization, distributed tracing, route waterfall analysis, deploy correlation, AI-powered request explanation, function call stack inspection, policy/auth tracing, and SLO intelligence into a single platform.

The platform operates in two modes:
- **Local dev:** In-memory stores, zero config, starts with `pnpm dev:local`
- **Production:** OpenTelemetry ingestion, persistent storage, multi-user auth, multi-region

Same UI, same APIs, different backends.

---

## 1. Product Architecture

### System Diagram

```
                        CloudMock Console
    ┌─────────────────────────────────────────────────┐
    │                  Dashboard SPA                    │
    │  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
    │  │ Topology  │  │ Request  │  │  AI Explain   │   │
    │  │  Canvas   │  │ Explorer │  │    Panel      │   │
    │  └────┬─────┘  └────┬─────┘  └──────┬───────┘   │
    │       │              │               │            │
    │  ┌────▼──────────────▼───────────────▼───────┐   │
    │  │              Admin API Gateway              │   │
    │  │  /topology /requests /traces /slo /explain  │   │
    │  └────┬──────────────┬───────────────┬───────┘   │
    └───────┼──────────────┼───────────────┼───────────┘
            │              │               │
    ┌───────▼──────┐ ┌────▼─────┐ ┌──────▼──────┐
    │ Trace Query  │ │ Topology │ │ AI Explain  │
    │   Service    │ │ Service  │ │   Service   │
    └───────┬──────┘ └────┬─────┘ └──────┬──────┘
            │              │               │
    ┌───────▼──────┐ ┌────▼─────┐ ┌──────▼──────┐
    │  SLO Config  │ │  Deploy  │ │  Incident   │
    │   Service    │ │  Intel   │ │   Service   │
    └───────┬──────┘ └────┬─────┘ └──────┬──────┘
            │              │               │
    ┌───────▼──────────────▼───────────────▼───────┐
    │                 Data Plane                     │
    │  ┌─────────┐  ┌────────┐  ┌────────┐        │
    │  │  Trace   │  │ Metric │  │  Log   │        │
    │  │  Store   │  │ Store  │  │ Store  │        │
    │  └─────────┘  └────────┘  └────────┘        │
    │  ┌─────────┐  ┌────────┐  ┌────────┐        │
    │  │ Profile  │  │Metadata│  │ Event  │        │
    │  │  Store   │  │ Store  │  │ Store  │        │
    │  └─────────┘  └────────┘  └────────┘        │
    └──────────────────────────────────────────────┘
```

### Deployment Modes

| Aspect | Local Dev | Production |
|--------|-----------|------------|
| Trace store | In-memory circular buffer | ClickHouse / Tempo |
| Metric store | In-memory with rolling windows | Prometheus / VictoriaMetrics |
| Log store | In-memory request log | Loki / Elasticsearch |
| Profile store | In-memory snapshots | Pyroscope / pprof bucket |
| Metadata store | In-memory + IaC config | PostgreSQL |
| Auth | Optional API key | JWT + RBAC |
| Ingestion | Direct Go middleware | OTel Collector pipeline |
| Config | cloudmock.yml | Kubernetes ConfigMap / env |

---

## 2. Core Product Areas

### 2.1 Service Topology Screen

The primary systems map. Three-panel layout.

**Left Rail — Request Explorer**

| Filter | Type | Source |
|--------|------|--------|
| Route/path | Text prefix | Request log |
| Method | Dropdown (GET/POST/PUT/DELETE) | Request log |
| Status code | Dropdown (2xx/4xx/5xx) | Request log |
| Service | Dropdown (from registry) | Service registry |
| Tenant ID | Text | Request headers |
| Org ID | Text | Request headers |
| User ID | Text | Request headers |
| Device type | Dropdown (mobile/web/api) | User-Agent |
| Platform | Dropdown (iOS/Android/Web) | User-Agent |
| Release version | Text | Deploy metadata |
| Feature flag | Text | Trace metadata |
| Auth outcome | Dropdown (allow/deny) | Trace metadata |
| Policy decision | Text | Trace metadata |
| Region | Dropdown | Request metadata |
| DB query signature | Text | Request body analysis |
| Trace ID | Text | Trace context |
| Time window | Range picker | Timestamp |
| Deploy version | Dropdown | Deploy events |

Additional capabilities:
- Save queries as named views (stored in metadata store)
- Compare query A vs query B (side-by-side)
- Live tail mode (SSE streaming with pause/resume)
- Pivot: metric → traces → logs → profile
- Error-only / slow-only toggles
- Trace sampling mode toggle

**Center Canvas — Interactive Service Graph**

| Visual Property | Maps To |
|----------------|---------|
| Node size | Request volume (logarithmic scale) |
| Node color | Health state (green/yellow/red from SLO) |
| Node border | Selected / hovered / alert state |
| Edge thickness | Request throughput |
| Edge color | P99 latency severity or error rate |
| Edge style | Solid (observed), dashed (IaC-only) |

Interactions:
- Click node → opens right panel with service details
- Click edge → shows latency distribution
- Hover → highlights upstream/downstream path
- Zoom, pan, pinch (touch support)
- Cluster collapse/expand by group
- Layout modes: service, domain, flow, hybrid (existing)
- Minimap (existing)

**Right Panel — Service Inspector**

When a node is selected:
- Service identity (name, type, group, icon)
- Health dashboard (request count, error rate, P50/P95/P99)
- Latency percentile bars
- Activity sparkline
- Top slow downstream dependencies (from trace data)
- Recent regressions (from compare endpoint)
- Active incidents (from incident service)
- SLO burn rate (from SLO engine)
- AI summary of current state (from explain service)
- Blast radius visualization

Tabs:
- Overview (stats + health)
- Requests (filtered by service, inline expand with explain)
- Traces (filtered, inline waterfall)
- Connections (inbound/outbound with latency)
- Resource (service-specific details)

### 2.2 Request Explorer (Detailed)

The left panel query model supports all filters in section 2.1. Implementation:

**API:** `GET /api/requests?{filters}`

Current filters (already implemented):
- `service`, `path`, `method`, `caller_id`, `action`, `error`, `trace_id`, `level`, `limit`

New filters to add:
- `tenant_id` — from `X-Tenant-Id` header
- `org_id` — from `X-Enterprise-Id` header
- `user_id` — from `X-User-Id` header
- `feature_flag` — from trace metadata
- `deploy_id` — from `X-Deployment-Id` header
- `min_latency_ms` — latency floor
- `max_latency_ms` — latency ceiling
- `status_min`, `status_max` — status code range
- `from`, `to` — time window (RFC3339)

**Saved Views API:** `GET/POST/DELETE /api/views`

```json
{
  "id": "slow-bff-premium",
  "name": "Slow BFF for Premium Tenants",
  "filters": {
    "service": "bff",
    "min_latency_ms": 500,
    "tenant_tier": "premium"
  },
  "created_by": "megan",
  "created_at": "2026-03-22T..."
}
```

### 2.3 Configurable SLO Policy System

**Configuration:** `cloudmock.yml` or `/api/slo` PUT

```yaml
slo:
  enabled: true
  rules:
    - service: dynamodb
      action: Query
      p50_ms: 10
      p95_ms: 50
      p99_ms: 100
      error_rate: 0.001
    - service: bff
      route: /bff/my-schedule
      p99_ms: 300
      error_rate: 0.01
      tenant_tier: premium   # tighter SLO for premium
    - service: "*"
      action: "*"
      p50_ms: 50
      p95_ms: 200
      p99_ms: 500
      error_rate: 0.01
```

Features:
- Per-route, per-service, per-tenant-tier rules
- Wildcard matching with most-specific-wins
- Error budget calculation (budget = 1 - SLO target)
- Burn rate tracking (consumption rate vs expected)
- Rolling windows (5m, 1h, 24h, 7d)
- Authenticated access required to edit (admin auth)
- Audit log for threshold changes
- Annotations explaining why targets exist
- Weekly SLO summary generation

**API:**
- `GET /api/slo` — current status (already implemented)
- `PUT /api/slo` — update rules (already implemented)
- `GET /api/slo/history` — threshold change audit log
- `GET /api/slo/summary?window=7d` — period summary

### 2.4 Route Waterfall Tracing

**Data model:** `TraceContext` with `Metadata` field (already implemented)

Waterfall shows:
- Request start to finish with time ruler
- Each service hop as a colored bar
- Queue wait time (from SQS spans)
- Auth/policy evaluation time (from metadata)
- Cache lookup (from `x-cache-status` header)
- DB query timings (from DynamoDB/RDS spans)
- External API calls (from plugin spans)
- Cold start penalty (from Lambda init duration)
- Retries and circuit breaker (from retry headers)

UX:
- Click span to expand details (already implemented)
- Highlight critical path automatically (already implemented)
- Show percent contribution to total time (already implemented)
- Compare trace against route baseline (from explain endpoint)
- Collapse unimportant spans (< 1% of total)
- Pin suspect spans for sharing
- Side-by-side trace comparison (`/api/traces/compare?a={id}&b={id}`)

### 2.5 AI "Explain This Request"

**Architecture:** MCP service calling `/api/explain/{id}`

Inputs (already captured):
- Trace graph with all spans
- Span durations and metadata
- Deploy events (from `/api/deploys`)
- Feature flag state (from trace metadata)
- Route baseline (from similar requests)
- Request/response bodies
- Profiling data (mem, goroutines)
- Policy/auth decisions (from trace metadata)

Narrative output structure (already implemented):
1. Opening summary (what happened)
2. Step-by-step execution walkthrough (every span in plain English)
3. Time breakdown by service layer (Data/Compute/Auth/Messaging)
4. Bottleneck analysis (critical path, sequential vs parallel)
5. Latency context (P50/P95/P99 comparison)
6. Request payload analysis (DynamoDB-specific: Scan warnings, FilterExpression notes)
7. Response body
8. Baseline comparison
9. Diagnosis with actionable recommendations
10. Metadata (IDs, timestamps, caller)

Enhanced outputs (to build):
- Confidence score (high/medium/low based on data completeness)
- Evidence list (specific spans/metrics that support the diagnosis)
- Impacted tenants (from tenant_id in trace metadata)
- Suspicious changes (from deploy correlation)
- Suggested next debugging actions
- Suggested mitigations

### 2.6 Function Call Stack Inspection

**New capability.** Requires application-level instrumentation.

**Go services:** Use `runtime.Callers()` to capture stack at key points:
- Handler entry
- DB query execution
- External API call
- Error occurrence

**Node/TypeScript services:** Use `Error().stack` or `--enable-source-maps`:
- Express middleware chain
- Aggregator function calls
- DynamoDB client calls
- External service calls

**Data model extension:**

```go
type SpanStack struct {
    Frames []StackFrame `json:"frames"`
}

type StackFrame struct {
    Function string `json:"function"`
    File     string `json:"file"`
    Line     int    `json:"line"`
    Module   string `json:"module,omitempty"`
}
```

Store in `TraceContext.Metadata["stack"]` as JSON.

**UI:** Collapsible panel in waterfall span detail:
- Collapsed view: top 3 frames
- Expanded view: full stack with source links
- Links to repo/commit/blame (from deploy metadata)

### 2.7 Deploy Correlation and Regression Intelligence

**Already implemented:** `/api/deploys` + explain anomaly detection

**Enhancements:**

Detection algorithms:
- Route latency regression: P99 increased > 50% after deploy
- Error rate regression: error rate increased > 5pp after deploy
- New outlier tenant: one tenant's P99 > 3x fleet average
- Cache miss increase: `x-cache-status: MISS` rate increased
- DB fanout increase: span count per trace increased
- Payload size growth: response body size increased

**API:**
- `GET /api/regressions` — list detected regressions
- `GET /api/regressions?deploy_id=X` — regressions for a deploy
- `GET /api/compare?service=X&deploy=Y` — before/after for deploy (existing enhanced)

**UI:**
- Deploy markers on time-series graphs
- "Before vs After" toggle in topology
- Responsible commit/PR/author display
- Rollback suggestion flag
- Regression confidence score

### 2.8 Tenant-Aware Observability

**Already implemented:** `/api/tenants` + per-tenant stats

**Enhancements:**

Every trace/span carries:
- `tenant_id` (from `X-Tenant-Id`)
- `org_id` (from `X-Enterprise-Id`)
- `plan_tier` (from `X-Plan-Tier` or metadata lookup)
- `environment` (from `X-Environment`)

**API:**
- `GET /api/tenants` — list with stats (existing)
- `GET /api/tenants?id=X` — tenant detail (existing)
- `GET /api/tenants/export` — CSV export (existing)
- `GET /api/tenants/{id}/compare` — tenant vs fleet baseline
- `GET /api/tenants/{id}/incidents` — tenant-specific incidents

**UI:**
- Isolate one tenant in topology (filter everything)
- Compare tenant vs fleet baseline
- Top impacted tenants panel
- Exportable incident report per tenant

### 2.9 Policy-Aware Tracing

**Context propagation (already implemented):** `extractTraceMetadata` captures:
- `x-policy-decision`
- `x-authz-result`
- `x-cache-status`
- `x-feature-flag`

**Enhancements:**

Trace the full auth/policy path:
- Authentication service (JWT validation)
- Identity resolution (user lookup)
- Policy engine evaluation (PEP/PDP)
- Role lookup (membership check)
- Contextual grants
- Allow/deny result
- Fallback logic
- Cache behavior
- Policy version used

**Data model:** Extend `TraceContext.Metadata` with structured policy fields:
```json
{
  "auth.method": "cognito-jwt",
  "auth.user_id": "user-123",
  "authz.policy_version": "v2.3",
  "authz.decision": "allow",
  "authz.evaluation_ms": "12.5",
  "authz.cache_hit": "false",
  "authz.roles": "admin,teacher"
}
```

### 2.10 Continuous Profiling

**Already implemented:** `MemAllocKB` and `Goroutines` per request.

**Enhancements:**

Per-service profiling endpoint:
- `GET /api/profile/{service}?type=cpu` — CPU flame graph data
- `GET /api/profile/{service}?type=heap` — heap allocation profile
- `GET /api/profile/{service}?type=goroutine` — goroutine dump

For Go services: use `runtime/pprof` programmatically.
For Node services: use `--inspect` + Chrome DevTools protocol.

**Trace-to-profile linking:**
When a span is slow, show:
- Top hot functions (from CPU profile)
- Allocation spikes (from heap profile)
- Lock contention (from block profile)
- GC pressure (from `runtime.MemStats`)

### 2.11 Smart Alerting and Incident Intelligence

**New service.** Build on SLO engine.

Alert on:
- Anomaly from baseline (statistical, not threshold)
- SLO burn-rate exhaustion
- New error pattern (error message clustering)
- Deploy-correlated spike
- Tenant-specific degradation
- Cascading downstream failure

**Incident data model:**
```go
type Incident struct {
    ID           string
    Status       string    // "active", "resolved", "investigating"
    Severity     string    // "critical", "warning", "info"
    Title        string    // auto-generated summary
    AffectedRoutes []string
    AffectedServices []string
    AffectedTenants  []string
    FirstSeen    time.Time
    LastSeen     time.Time
    RootCause    string    // from AI explain
    RelatedDeploy string
    TopTraces    []string  // trace IDs
    Owner        string
}
```

**API:**
- `GET /api/incidents` — list active incidents
- `GET /api/incidents/{id}` — incident detail
- `POST /api/incidents/{id}/acknowledge` — claim ownership
- `POST /api/incidents/{id}/resolve` — mark resolved

### 2.12 Cost-Aware Traces

**Already implemented:** `/api/cost` with per-service estimates.

**Enhancements:**

Per-request cost calculation:
- Lambda: `duration_ms * memory_mb * $0.0000166667 / 1000`
- DynamoDB: `read_units * $0.00000025 + write_units * $0.00000125`
- S3: `$0.0000004` per GET, `$0.000005` per PUT
- SQS: `$0.0000004` per request
- Data transfer: `response_size_kb * $0.00000009`

**API:**
- `GET /api/cost` — aggregate cost (existing)
- `GET /api/cost/routes` — cost per route
- `GET /api/cost/tenants` — cost per tenant
- `GET /api/cost/trend?window=7d` — cost trend
- `GET /api/cost/regressions` — cost regressions after deploys

---

## 3. Backend Services

### 3.1 Trace Query Service

Responsible for search, filtering, trace retrieval, compare mode, tenant scoping.

**Local mode:** Uses in-memory `TraceStore` (existing, 500-trace buffer).
**Production mode:** Queries ClickHouse/Tempo via gRPC.

**Interface:**
```go
type TraceQueryService interface {
    Search(filter TraceFilter) ([]TraceSummary, error)
    Get(traceID string) (*TraceContext, error)
    Timeline(traceID string) ([]TimelineSpan, error)
    Compare(traceA, traceB string) (*TraceComparison, error)
}
```

### 3.2 Topology Service

Responsible for dependency graph generation, health computation, edge metrics, blast radius.

**Local mode:** Uses in-memory topology from IaC config + traffic edges (existing).
**Production mode:** Persists to metadata store, updates from OTel spans.

### 3.3 SLO Config Service

Responsible for route thresholds, burn-rate calculations, config history, RBAC.

**Local mode:** Uses `cloudmock.yml` + in-memory engine (existing).
**Production mode:** PostgreSQL-backed with audit log and RBAC.

### 3.4 Deploy Intelligence Service

Responsible for ingesting deploy/config/flag changes, correlating regressions.

**Local mode:** In-memory deploy events (existing `/api/deploys`).
**Production mode:** Persistent store with webhook ingestion from CI/CD.

### 3.5 AI Explain Service (MCP)

Responsible for trace context retrieval, grounded prompts, structured explanations.

**Local mode:** `/api/explain/{id}` with Go-generated narrative (existing).
**Production mode:** Same endpoint + optional LLM enhancement via MCP.

### 3.6 Profile Analysis Service

Responsible for CPU/memory profile indexing, trace-to-profile linking.

**Local mode:** `runtime.MemStats` per request (existing).
**Production mode:** Pyroscope integration with continuous profiling.

### 3.7 Incident Service

Responsible for alert grouping, incident creation, ownership routing.

**Local mode:** In-memory incident list.
**Production mode:** PagerDuty/Slack integration, persistent storage.

---

## 4. Data Plane

### 4.1 Telemetry Ingestion

**Local mode:** Direct Go middleware (existing `LoggingMiddleware`).

**Production mode:** OpenTelemetry pipeline:

```
App Services
    │ OTel SDK
    ▼
OTel Collector
    │ processors: batch, filter, enrich
    ▼
Fanout Exporters
    ├── Trace Store (ClickHouse / Tempo)
    ├── Metric Store (Prometheus)
    ├── Log Store (Loki)
    └── Profile Store (Pyroscope)
```

### 4.2 Storage Architecture

| Store | Local | Production | Schema |
|-------|-------|------------|--------|
| Traces | `TraceStore` (500 circular) | ClickHouse `traces` table | TraceContext + spans |
| Metrics | `RequestStats` + `SLOEngine` | Prometheus TSDB | RED + USE + histograms |
| Logs | `RequestLog` (1000 circular) | Loki | Structured JSON, indexed by trace_id |
| Profiles | `runtime.MemStats` snapshots | Pyroscope | pprof format |
| Metadata | IaC config + in-memory | PostgreSQL | Services, deploys, SLOs, views |
| Events | SSE broadcaster | Kafka/NATS | Deploy, config, flag changes |

### 4.3 High-Cardinality Indexing

Production trace store must support:
- `tenant_id` partition (millions of unique values)
- `trace_id` point lookup (billions)
- `service + route + time_range` aggregation
- `deploy_id` correlation
- Full-text search on error messages

ClickHouse with `MergeTree` engine and `tenant_id` as partition key.

---

## 5. API Contract Summary

### Existing APIs (already implemented)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/topology` | GET | Service graph with nodes + edges |
| `/api/topology/config` | GET/PUT | IaC topology config |
| `/api/requests` | GET | Filtered request log (level=app default) |
| `/api/requests/{id}` | GET | Request detail |
| `/api/requests/{id}/replay` | POST | Replay captured request |
| `/api/traces` | GET | Trace list with filtering |
| `/api/traces/{id}` | GET | Full trace tree |
| `/api/traces/{id}/timeline` | GET | Waterfall timeline spans |
| `/api/explain/{id}` | GET | AI narrative + analysis |
| `/api/slo` | GET/PUT | SLO status + rule updates |
| `/api/blast-radius` | GET | Upstream/downstream impact |
| `/api/tenants` | GET | Per-tenant stats |
| `/api/tenants/export` | GET | CSV export |
| `/api/cost` | GET | Cost breakdown by service |
| `/api/compare` | GET | Before/after comparison |
| `/api/deploys` | GET/POST | Deploy event tracking |
| `/api/shadow` | POST | Shadow traffic testing |
| `/api/metrics` | GET | Per-service latency percentiles |
| `/api/stream` | GET | SSE live event stream |
| `/api/chaos` | GET/POST/PUT/DELETE | Fault injection rules |
| `/api/health` | GET | Service health |

### New APIs (to implement)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/requests` | GET | Extended filters (tenant, org, time range) |
| `/api/views` | GET/POST/DELETE | Saved query views |
| `/api/traces/compare` | GET | Side-by-side trace comparison |
| `/api/slo/history` | GET | SLO threshold change audit log |
| `/api/slo/summary` | GET | Period SLO summary |
| `/api/regressions` | GET | Detected regressions |
| `/api/tenants/{id}/compare` | GET | Tenant vs fleet baseline |
| `/api/incidents` | GET/POST | Incident management |
| `/api/incidents/{id}` | GET/PUT | Incident detail + updates |
| `/api/profile/{service}` | GET | CPU/heap/goroutine profile |
| `/api/cost/routes` | GET | Per-route cost breakdown |
| `/api/cost/tenants` | GET | Per-tenant cost breakdown |
| `/api/cost/trend` | GET | Cost trend over time |

---

## 6. Phased Roadmap

### Phase 0: Foundation (DONE)
- Reverse proxy with HTTPS + DNS
- Dynamic domain config from Pulumi IaC
- Auto-extracted Lambda contracts
- DynamoDB table definitions with GSIs
- File watcher for auto-updates

### Phase 1: Core Observability (DONE)
- Request panel with live tail + filtering
- Waterfall tracing with span merging
- AI explain with narrative debug reports
- SLO engine with burn rate
- Request replay
- Blast radius analysis
- Tenant filtering + cost estimation
- Deploy correlation
- Time-travel comparison

### Phase 2: Dashboard UI Overhaul (NEXT)
- 3-panel topology layout (left rail, center canvas, right inspector)
- Health-colored nodes and edges
- Node size from traffic volume
- Edge thickness from throughput
- Interactive service inspector
- Saved query views
- Advanced filter UI

### Phase 3: Production Data Plane
- OpenTelemetry SDK integration (Go + Node)
- OTel Collector pipeline
- ClickHouse trace storage
- Prometheus metrics storage
- Persistent metadata store (PostgreSQL)

### Phase 4: Intelligence Layer
- Smart alerting (anomaly-based)
- Incident service with grouping
- Regression detection engine
- Side-by-side trace comparison
- Cost regression detection

### Phase 5: Advanced Profiling
- Continuous CPU/heap profiling
- Flame graph rendering
- Trace-to-profile linking
- Function call stack capture
- Source map symbolication

### Phase 6: Enterprise Features
- Multi-user RBAC auth
- Tenant isolation (hard visibility boundaries)
- Exportable incident reports
- Webhook integrations (Slack, PagerDuty)
- API rate limiting
- Audit logging

---

## 7. Technical Decisions

### Why Go for the backend
- Already the cloudmock language
- Excellent concurrency for fan-out queries
- Low memory overhead for in-memory stores
- Single binary deployment
- Native pprof for profiling

### Why Preact for the dashboard
- Already in use (< 10KB framework)
- Fast rendering for topology SVG
- No migration needed

### Why ClickHouse for production traces
- Columnar storage optimized for time-series queries
- High-cardinality indexing (tenant_id partitioning)
- SQL interface for ad-hoc queries
- MergeTree engine for efficient aggregation
- Open source, well-documented

### Why OpenTelemetry for production ingestion
- Industry standard (vendor-neutral)
- SDK available for Go, Node, Python
- Collector handles batching, sampling, enrichment
- Compatible with ClickHouse, Prometheus, Loki exporters

---

## 8. Success Criteria

The platform succeeds when a developer can:

1. Open the topology screen and immediately see which services are healthy
2. Click a service and see its requests, traces, and connections
3. Click a request and get a full narrative explanation of what happened
4. Filter by tenant and see if an issue is global or customer-specific
5. See a deploy marker and compare before/after performance
6. Get an alert when SLO budget is at risk
7. Replay a failing request to reproduce the issue
8. See the estimated cost of a request path
9. All of this in < 2 seconds page load, with live updates via SSE
10. Zero configuration required for local dev (`pnpm dev:local` and it works)
