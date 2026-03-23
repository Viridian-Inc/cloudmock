# Phase 4d+e: Smart Alerting & Incident Service — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 4d+e of 6 (CloudMock Console — Intelligence Layer, sub-projects 4+5 of 5)
**Depends on:** Phase 3 (Data Plane), Phase 4a (Regression Engine)

---

## Overview

An incident service that auto-creates incidents from regression alerts and SLO breaches, groups related alerts by cause within a 5-minute window, and auto-resolves when source issues recover. Integrates with existing regression engine and SLO engine via an `AlertSink` callback pattern.

### Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Combined alerting + incidents | Tightly coupled — alerts without incidents incomplete |
| Grouping | By cause + 5-minute window | One bad deploy = one incident, not N regressions |
| Integration | AlertSink callback | No polling, no tight coupling |
| Lifecycle | Auto-create + auto-resolve + manual API | Self-healing; defer escalation to Phase 6 |

---

## 1. AlertSink Interface

```go
// pkg/incident/types.go

type AlertSink interface {
    OnRegression(ctx context.Context, r regression.Regression) error
    OnSLOBreach(ctx context.Context, service, action string, burnRate, budgetUsed float64) error
    OnErrorPattern(ctx context.Context, service, pattern string, count int) error
}
```

The regression engine and SLO engine accept an optional `AlertSink`. When set, they call it alongside their normal behavior.

---

## 2. Incident Data Model

```go
type Incident struct {
    ID               string
    Status           string     // "active", "acknowledged", "resolved"
    Severity         string     // "critical", "warning", "info"
    Title            string
    AffectedServices []string
    AffectedTenants  []string
    AlertCount       int
    RootCause        string
    RelatedDeployID  string
    FirstSeen        time.Time
    LastSeen         time.Time
    ResolvedAt       *time.Time
    Owner            string
}

type IncidentFilter struct {
    Status   string
    Severity string
    Service  string
    Limit    int
}
```

### PostgreSQL table

```sql
CREATE TABLE incidents (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status            TEXT NOT NULL DEFAULT 'active',
    severity          TEXT NOT NULL,
    title             TEXT NOT NULL,
    affected_services TEXT[] DEFAULT '{}',
    affected_tenants  TEXT[] DEFAULT '{}',
    alert_count       INT NOT NULL DEFAULT 1,
    root_cause        TEXT,
    related_deploy_id UUID REFERENCES deploys(id),
    first_seen        TIMESTAMPTZ NOT NULL,
    last_seen         TIMESTAMPTZ NOT NULL,
    resolved_at       TIMESTAMPTZ,
    owner             TEXT
);

CREATE INDEX idx_incidents_status ON incidents(status) WHERE status = 'active';
CREATE INDEX idx_incidents_severity ON incidents(severity, first_seen DESC);
```

---

## 3. Incident Service

```go
type Service struct {
    store        IncidentStore
    groupWindow  time.Duration  // default 5m
}

func NewService(store IncidentStore, groupWindow time.Duration) *Service
```

Implements `AlertSink`. On each alert:
1. Build a grouping key from service + deploy_id (if present) + alert type
2. Query store for active incident matching that key with `last_seen` within `groupWindow`
3. If found: update `LastSeen`, increment `AlertCount`, merge `AffectedServices`/`AffectedTenants`, upgrade severity if new alert is more severe
4. If not found: create new incident with title, severity, affected services from the alert

### Auto-resolution

`OnRegression` is called for both new and resolved regressions (regression.Status == "resolved"). When a resolved regression arrives:
1. Find active incidents with matching `RelatedDeployID` or `AffectedServices`
2. Check if all regressions for that incident are now resolved (query regression store)
3. If all resolved → auto-resolve the incident

`CheckSLORecovery(ctx)` — called periodically. For active incidents sourced from SLO breaches, check if the SLO window is no longer breaching. If recovered → auto-resolve.

### IncidentStore interface

```go
type IncidentStore interface {
    Save(ctx context.Context, inc *Incident) error
    Get(ctx context.Context, id string) (*Incident, error)
    List(ctx context.Context, filter IncidentFilter) ([]Incident, error)
    Update(ctx context.Context, inc *Incident) error
    FindActiveByKey(ctx context.Context, service, deployID string, since time.Time) (*Incident, error)
}
```

---

## 4. Integration Points

### Regression engine (`pkg/regression/engine.go`)
- Add `alertSink AlertSink` field (optional)
- Add `SetAlertSink(sink AlertSink)` setter
- In `scanService` and `evaluateDeploy`, after saving a regression, call `sink.OnRegression(ctx, reg)` if sink != nil
- In `checkResolutions`, after resolving a regression, call `sink.OnRegression(ctx, resolvedReg)` if sink != nil

### SLO engine (`pkg/gateway/slo.go`)
- Add `alertSink` field with a general interface to avoid importing incident package
- Define callback type: `type SLOAlertFunc func(service, action string, burnRate, budgetUsed float64)`
- Add `SetAlertFunc(fn SLOAlertFunc)` setter
- In `Record`, after detecting a breach, call the function if set
- The incident service adapts this to its `AlertSink` interface in main.go wiring

---

## 5. API

```
GET  /api/incidents                    — list (query: status, severity, service, limit)
GET  /api/incidents/{id}               — detail
POST /api/incidents/{id}/acknowledge   — set status=acknowledged, owner from request body
POST /api/incidents/{id}/resolve       — set status=resolved
```

---

## 6. Configuration

```yaml
incidents:
  enabled: true
  group_window: 5m
```

---

## 7. File Layout

```
pkg/incident/
├── types.go          # Incident, AlertSink, IncidentFilter, AlertType
├── service.go        # Service implementing AlertSink, grouping, auto-resolve
├── service_test.go
├── store.go          # IncidentStore interface
├── postgres/
│   ├── store.go
│   └── store_test.go
└── memory/
    ├── store.go
    └── store_test.go

docker/init/postgres/03-incident-schema.sql
```

**Files modified:**
- `pkg/regression/engine.go` — add AlertSink field + setter + calls
- `pkg/gateway/slo.go` — add SLOAlertFunc field + setter + calls
- `pkg/admin/api.go` — add incident API endpoints
- `cmd/gateway/main.go` — create incident service, wire sinks
- `pkg/config/config.go` — add IncidentConfig
