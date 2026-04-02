# Operational Service Map — Design Spec

**Date:** 2026-03-30
**Status:** Design
**Location:** `neureaux-devtools/src/views/topology/`

## Problem

During incident response, engineers jump between Datadog, Grafana, CloudWatch, Slack, and PagerDuty to understand what's broken, what's impacted, and what changed. There's no single screen that shows live service health, blast radius, recent deploys, and lets you drill into metrics, endpoints, and incidents — all correlated in one place.

The topology view currently shows a static architecture diagram. It should be the **first screen you open after getting paged** — a live operational service map that replaces the frantic tab-switching.

## Goals

1. Live health states on every service node (green/yellow/red)
2. User-facing vs internal impact distinction (path-based inference + manual override)
3. Edge metrics (req/s, p99 latency) updated in real-time
4. Click any service → deep-dive inspector (metrics, endpoints, deploys, incidents, connections)
5. Time travel: scrub through history for PIR/postmortem analysis
6. Service browsing by explicitly configured domain groupings
7. Deploy correlation: visualize when deploys happened relative to metric changes
8. Blast radius: highlight downstream services affected by a failing node
9. Local/cloud routing toggle per service
10. OpenAPI/manifest endpoint browsing per service

## Non-Goals (This Spec)

- Log ingestion from external sources (future — configurable log sources)
- Real-time alerting/paging (use existing PagerDuty/OpsGenie integration)
- Multi-tenant data isolation (cloudmock.io Phase 4)

## Data Sources

All data comes from existing cloudmock admin API endpoints. No new backend endpoints needed.

| Data | Endpoint | Update Frequency |
|------|----------|-----------------|
| Service metrics (p50/p95/p99, error rate, call count) | `GET /api/metrics?minutes=N` | Poll every 10s |
| Time-bucketed metrics | `GET /api/metrics/timeline?minutes=N&bucket=1m` | Poll every 30s |
| Topology (nodes + edges with callCount, avgLatencyMs) | `GET /api/topology` | Poll every 30s |
| Deployments | `GET /api/deploys` | Poll every 30s |
| Incidents | `GET /api/incidents` | Poll every 10s |
| SLO status | `GET /api/slo` | Poll every 30s |
| Live request stream | `GET /api/stream` (SSE) | Real-time |
| Service endpoints | Service manifest JSON / OpenAPI spec files | On load |

### Local Persistence (SQLite via Tauri)

Memory-based for immediate window (cloudmock's existing buffers). SQLite for multi-day history:

- `metrics_snapshots`: 5-minute rollups of per-service metrics. 7-day retention.
- `deploy_events`: deploy history. 30-day retention.
- `incident_history`: incident lifecycle events. 30-day retention.

Eventually backed by SaaS database (cloudmock.io Phase 4).

## UI Layout

```
┌─ Toolbar ────────────────────────────────────────────────────────┐
│ [Time: Live ▾] [15m | 1h | 6h | 24h | 3d | 7d | Custom]       │
│ 🔴 2 active incidents │ Last deploy: 3m ago (auth fix)          │
├─ Service Map (center, scrollable) ──────┬─ Inspector (right) ───┤
│                                         │                       │
│  Nodes with health states:              │ Tabbed detail panel:  │
│  🟢 green = healthy                     │ [Metrics] [Endpoints] │
│  🟡 yellow = degraded                   │ [Deploys] [Incidents] │
│  🔴 red = critical                      │ [Connections]         │
│                                         │                       │
│  Edges with: req/s + p99               │ Sparkline charts,     │
│  User-facing impact banner              │ route lists,          │
│  Incident badges on affected nodes      │ deploy history,       │
│  Deploy markers (recent)                │ blast radius          │
│                                         │                       │
├─ Timeline (bottom, collapsible) ────────┴───────────────────────┤
│ ──●─────────●──────────────●───────────────●──────────── now    │
│   deploy    incident       deploy          alert                │
└─────────────────────────────────────────────────────────────────┘
```

### Service Map (Center)

ELK.js layered DAG layout (existing). Enhanced with:

**Node rendering:**
- Pill-shaped nodes with health-state border color (green/yellow/red)
- Pulsing red border + "⚠ User Impact" badge when user-facing path is degraded
- Incident count badge (red circle with number) on affected nodes
- Deploy marker (blue dot) on recently-deployed nodes
- Resource count badge for collapsed AWS services

**Health state computation:**
- 🟢 `error_rate < 1% AND p99 < slo_threshold (or < 200ms default)`
- 🟡 `error_rate 1-5% OR p99 within 80% of SLO threshold`
- 🔴 `error_rate > 5% OR p99 > SLO threshold OR has active incident`

**User-facing impact detection:**
- **Path-based inference (default):** BFS from Client nodes through synchronous edges. If a red/yellow node is reachable from a Client node via synchronous (non-stream, non-async) edges, it's user-facing.
- **Manual override:** per-service `userFacing: true/false` tag in domain config. Overrides inference.
- Visual: user-facing issues get a distinct treatment (pulsing border, banner at top of map).

**Edge rendering:**
- Label: `{req/s} · {p99}ms` (from topology edge `callCount` and `avgLatencyMs`)
- Thickness proportional to req/s
- Color: green when healthy, yellow/red when latency exceeds thresholds

**Blast radius:**
- When a node is selected, highlight all downstream nodes (BFS along outgoing edges)
- Dim non-affected nodes
- Inspector "Connections" tab shows full dependency tree

### Inspector Panel (Right)

**Metrics tab** (default when clicking a node):
- Sparkline charts: req/s, p99 latency, error rate
- Time range synced with global selector
- SLO burn rate bar (if SLO rules exist)
- Before/after deploy comparison when deploy is selected on timeline

**Endpoints tab:**
- Routes from service manifest (existing `service-manifest.json`)
- Fallback: OpenAPI spec files (scanned from configurable paths)
- Grouped by resource prefix (e.g., `/attendance/check-in`, `/attendance/report`)
- Per-endpoint: request count, avg latency, error count
- Click → filters Activity view to that route

**Deploys tab:**
- Chronological list from `/api/deploys`
- Fields: commit, author, branch, PR, timestamp
- Metric change indicator: shows if p99 or error rate changed after deploy

**Incidents tab:**
- Active + recent incidents affecting this service
- Severity badge, status, first/last seen, alert count
- Acknowledge/Resolve buttons (existing API)

**Connections tab:**
- Inbound + outbound services with edge metrics
- Blast radius count: "N downstream services affected"
- Click connection → selects that node

### Timeline (Bottom)

Horizontal timeline bar showing events:
- 🔵 Deploys (service name, commit message)
- 🔴 Incidents (severity, title, affected services)
- 🟡 Alerts (SLO breaches, error rate spikes)

**Interaction:**
- Drag to scrub — service map updates health states to that point in time
- Click event → inspector shows details
- "Live" button snaps back to real-time

**Data source:** Merged from `/api/deploys`, `/api/incidents`, `/api/slo` alerts. Historical data from SQLite when available.

### Service Browser (Domain-Grouped)

Accessible from a sidebar toggle or the Services view. Services grouped by explicitly configured domains:

```json
{
  "domains": [
    {
      "name": "Core Platform",
      "services": ["identity", "organizations", "membership"]
    },
    {
      "name": "Scheduling",
      "services": ["attendance", "calendar", "compliance"]
    },
    {
      "name": "Commerce",
      "services": ["billing", "orders"]
    },
    {
      "name": "Engagement",
      "services": ["notifications", "integrations", "invitation"]
    }
  ]
}
```

Config file: `service-domains.json` in project root or `~/.config/neureaux-devtools/`. Editable from Settings view.

Each service row shows: icon, name, health dot, route count (from manifest), active incident count.

### Local/Cloud Routing

Per-service toggle in the inspector panel:

```
Routing:  ● Local    ○ Cloud (dev)
Local:    http://localhost:3202
Cloud:    https://bff.dev.autotend.io
```

- Persisted to localStorage per connection profile
- Extends cloudmock's existing `ProxyRoute` infrastructure
- Cloud endpoints configured in `service-domains.json` or per-service in settings
- Use case: test local BFF against real cloud Cognito

### OpenAPI / Service Manifest Integration

**Primary source:** `service-manifest.json` (already extracted from source code by `extract-contracts.ts`)
- Routes per service with method + path
- Tables per service with access patterns
- SDK clients per service

**Secondary source:** OpenAPI spec files
- Configurable scan paths (e.g., `["services/*/openapi.yaml", "api-specs/*.json"]`)
- Parsed on load, mapped to topology nodes by service name
- Provides: parameter schemas, response types, descriptions
- Displayed in Endpoints tab

## Implementation Phases

### Phase A: Live Service Map (Core)
- Health state computation + node coloring
- Edge metrics display (req/s, p99)
- Polling hooks for metrics, incidents, deploys
- User-facing impact detection (path-based)
- Incident badges on nodes
- Deploy markers

### Phase B: Inspector Tabs
- Metrics tab with sparkline charts
- Endpoints tab (service manifest)
- Deploys tab
- Incidents tab
- Connections tab with blast radius

### Phase C: Timeline + Time Travel
- Timeline bar with deploy/incident/alert events
- Time range selector (presets + custom)
- Historical scrubbing (map updates to past state)
- SQLite persistence layer in Rust backend

### Phase D: Service Browser + Routing
- Domain-grouped service list
- Domain config file support
- Local/cloud routing toggle
- OpenAPI spec file scanning

## Key Files

| File | Purpose |
|------|---------|
| `src/views/topology/index.tsx` | Orchestrates data fetching, collapse logic |
| `src/views/topology/topology-canvas.tsx` | ELK layout + node/edge rendering |
| `src/views/topology/node-inspector.tsx` | Right panel with tabs |
| `src/views/topology/timeline.tsx` | Bottom timeline bar (new) |
| `src/views/topology/service-browser.tsx` | Domain-grouped list (new) |
| `src/hooks/use-topology-metrics.ts` | Polling for live metrics (new) |
| `src/hooks/use-timeline.ts` | Merged deploy/incident/alert events (new) |
| `src/lib/health.ts` | Health state computation logic (new) |
| `src-tauri/src/persistence.rs` | SQLite read/write for history (new) |

## Verification

1. Generate traffic → nodes turn green with req/s on edges
2. Inject errors (chaos view) → node turns red, user-facing impact banner appears
3. POST a deploy → deploy marker appears on timeline + node
4. Create incident → badge appears on affected node
5. Scrub timeline back → map shows past health states
6. Click service → inspector shows metrics sparklines, endpoints, deploys
7. Toggle routing → requests route to cloud endpoint
