# Phase 4d+e: Smart Alerting & Incident Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-create incidents from regression alerts and SLO breaches, group by cause within a 5-minute window, auto-resolve when source issues recover.

**Architecture:** An `IncidentService` implements the `AlertSink` interface, receives callbacks from the regression engine and SLO engine, groups alerts into incidents by cause + time window, and manages incident lifecycle. PostgreSQL store for persistence, in-memory for local mode.

**Tech Stack:** Go 1.26, PostgreSQL, testcontainers-go

---

## File Structure

```
pkg/incident/
├── types.go          # Incident, AlertSink, IncidentFilter
├── service.go        # Service implementing AlertSink + grouping + auto-resolve
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
- `pkg/regression/engine.go` — add AlertSink + setter + calls
- `pkg/gateway/slo.go` — add SLOAlertFunc + setter + call on breach
- `pkg/admin/api.go` — add 4 incident endpoints
- `cmd/gateway/main.go` — create service, wire sinks
- `pkg/config/config.go` — add IncidentConfig

---

## Task 1: Types & Interfaces

**Files:**
- Create: `pkg/incident/types.go`
- Create: `pkg/incident/store.go`

- [ ] **Step 1:** `mkdir -p pkg/incident/postgres pkg/incident/memory`

- [ ] **Step 2: Write types.go**

```go
package incident

import (
    "context"
    "time"
    "github.com/Viridian-Inc/cloudmock/pkg/regression"
)

type AlertSink interface {
    OnRegression(ctx context.Context, r regression.Regression) error
    OnSLOBreach(ctx context.Context, service, action string, burnRate, budgetUsed float64) error
}

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

- [ ] **Step 3: Write store.go**

```go
type IncidentStore interface {
    Save(ctx context.Context, inc *Incident) error
    Get(ctx context.Context, id string) (*Incident, error)
    List(ctx context.Context, filter IncidentFilter) ([]Incident, error)
    Update(ctx context.Context, inc *Incident) error
    FindActiveByKey(ctx context.Context, service, deployID string, since time.Time) (*Incident, error)
}

var ErrNotFound = errors.New("incident not found")
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/incident/...`

- [ ] **Step 5: Commit**

```bash
git add pkg/incident/
git commit -m "feat(incident): add types, AlertSink, and IncidentStore interfaces"
```

---

## Task 2: In-Memory & PostgreSQL Stores

**Files:**
- Create: `pkg/incident/memory/store.go` + `store_test.go`
- Create: `pkg/incident/postgres/store.go` + `store_test.go`
- Create: `docker/init/postgres/03-incident-schema.sql`

- [ ] **Step 1: Write PostgreSQL schema**

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

- [ ] **Step 2: Write in-memory store test + implementation**

Mutex-protected slice. `FindActiveByKey` matches by: any of `AffectedServices` contains `service`, AND (deployID matches or deployID is empty), AND `LastSeen` after `since`, AND status is "active" or "acknowledged". Test Save/Get/List/Update/FindActiveByKey.

- [ ] **Step 3: Write PostgreSQL store test + implementation**

Same testcontainers pattern as `pkg/regression/postgres/`. Apply schemas 01, 02, 03. Test all IncidentStore methods. Guard with `testing.Short()`.

`FindActiveByKey`:
```sql
SELECT * FROM incidents
WHERE $1 = ANY(affected_services)
AND ($2 = '' OR related_deploy_id::text = $2)
AND last_seen > $3
AND status IN ('active', 'acknowledged')
ORDER BY last_seen DESC
LIMIT 1
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/incident/... -v -short -cover`

- [ ] **Step 5: Commit**

```bash
git add docker/init/postgres/03-incident-schema.sql pkg/incident/memory/ pkg/incident/postgres/
git commit -m "feat(incident): add in-memory and PostgreSQL incident stores"
```

---

## Task 3: Incident Service (Grouping + Auto-Resolution)

**Files:**
- Create: `pkg/incident/service.go`
- Create: `pkg/incident/service_test.go`

- [ ] **Step 1: Write service test**

Use in-memory store. Test:

```go
func TestOnRegression_CreatesIncident(t *testing.T) {
    // Send a regression → verify incident created with correct title, severity, services
}

func TestOnRegression_GroupsByDeploy(t *testing.T) {
    // Send 2 regressions with same deploy_id within 5m → verify 1 incident with AlertCount=2
}

func TestOnRegression_SeparatesUnrelated(t *testing.T) {
    // Send 2 regressions for different services, no deploy → verify 2 incidents
}

func TestOnSLOBreach_CreatesIncident(t *testing.T) {
    // Send SLO breach → verify incident with "SLO" in title
}

func TestAutoResolve_OnRegressionResolved(t *testing.T) {
    // Send regression → incident created
    // Send same regression with Status="resolved" → incident resolved
}

func TestGroupWindow_Expiry(t *testing.T) {
    // Send regression → wait > group window → send another for same service
    // Verify 2 separate incidents
}

func TestSeverityEscalation(t *testing.T) {
    // First alert: warning → incident warning
    // Second alert: critical → incident upgraded to critical
}
```

- [ ] **Step 2: Run tests — verify FAIL**

- [ ] **Step 3: Write service.go**

```go
type Service struct {
    store       IncidentStore
    regStore    regression.RegressionStore  // for checking if all regressions resolved
    groupWindow time.Duration
}

func NewService(store IncidentStore, regStore regression.RegressionStore, groupWindow time.Duration) *Service
```

`OnRegression(ctx, r)`:
1. If r.Status == "resolved": find active incident for this regression's service+deploy, check if all regressions for that deploy are resolved, if so resolve incident
2. If r.Status == "active": build grouping key (r.Service + r.DeployID), call `store.FindActiveByKey(service, deployID, time.Now().Add(-groupWindow))`, merge or create

`OnSLOBreach(ctx, service, action, burnRate, budgetUsed)`:
1. Find active incident for service (no deploy) within group window
2. Merge or create with title "SLO burn rate alert: {service}/{action}"
3. Severity: budgetUsed > 0.9 → critical, > 0.5 → warning, else info

Title generation:
- Regression: use regression.Title
- SLO: "SLO burn rate alert: {service}/{action} ({budgetUsed*100}% budget consumed)"

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/incident/ -v -run TestOn -cover`

- [ ] **Step 5: Commit**

```bash
git add pkg/incident/service.go pkg/incident/service_test.go
git commit -m "feat(incident): add incident service with grouping and auto-resolution

Groups alerts by cause + 5m window. Auto-resolves when all source
regressions resolve. Severity escalation on merge."
```

---

## Task 4: Wire AlertSink into Regression + SLO Engines

**Files:**
- Modify: `pkg/regression/engine.go` — add AlertSink field + setter + calls
- Modify: `pkg/gateway/slo.go` — add SLOAlertFunc + setter + call

- [ ] **Step 1: Add AlertSink to regression engine**

In `pkg/regression/engine.go`:
- Add `alertSink incident.AlertSink` field (use an interface to avoid tight coupling — or just use the concrete type)
- Actually, to avoid circular imports (incident imports regression for Regression type, regression can't import incident), define a minimal callback interface in the regression package:

```go
// pkg/regression/engine.go
type AlertCallback func(ctx context.Context, r Regression)
```

- Add `alertCallback AlertCallback` field to Engine
- Add `SetAlertCallback(fn AlertCallback)` setter
- In `scanService` and `evaluateDeploy`, after `store.Save()`, call `alertCallback(ctx, reg)` if set
- In `checkResolutions`, after `UpdateStatus("resolved")`, call `alertCallback(ctx, resolvedReg)` if set

- [ ] **Step 2: Add SLOAlertFunc to SLO engine**

In `pkg/gateway/slo.go`:
- Add `type SLOAlertFunc func(service, action string, burnRate, budgetUsed float64)`
- Add `alertFunc SLOAlertFunc` field to SLOEngine
- Add `SetAlertFunc(fn SLOAlertFunc)` setter
- In `Record`, after detecting a breaching window, call `alertFunc(service, action, burnRate, budgetUsed)` if set

- [ ] **Step 3: Run existing tests**

Run: `go test ./pkg/regression/ ./pkg/gateway/ -v -short`
Expected: All PASS (callbacks are optional, nil by default).

- [ ] **Step 4: Commit**

```bash
git add pkg/regression/engine.go pkg/gateway/slo.go
git commit -m "feat(incident): add AlertCallback to regression engine and SLOAlertFunc to SLO engine

Optional callbacks for incident service integration. No behavior
change when callbacks are nil."
```

---

## Task 5: Config, API & Gateway Wiring

**Files:**
- Modify: `pkg/config/config.go` — add IncidentConfig
- Modify: `pkg/admin/api.go` — add 4 incident endpoints
- Modify: `cmd/gateway/main.go` — create service, wire callbacks

- [ ] **Step 1: Add IncidentConfig**

```go
type IncidentConfig struct {
    Enabled     bool   `yaml:"enabled" json:"enabled"`
    GroupWindow string `yaml:"group_window" json:"group_window"` // default "5m"
}
```

Add `Incidents IncidentConfig` to Config struct. Default: enabled=true, group_window="5m".

- [ ] **Step 2: Add incident API endpoints**

- `incidentService *incident.Service` field + `SetIncidentService` setter
- `handleIncidents` handler:
  - `GET /api/incidents` — list with filters (status, severity, service, limit)
  - `GET /api/incidents/{id}` — detail
  - `POST /api/incidents/{id}/acknowledge` — parse `{"owner": "name"}` from body, set status + owner
  - `POST /api/incidents/{id}/resolve` — set status=resolved
- Register `/api/incidents` and `/api/incidents/` in `NewWithDataPlane()`

- [ ] **Step 3: Wire in main.go**

```go
if cfg.Incidents.Enabled {
    var incStore incident.IncidentStore
    switch mode {
    case "local":
        incStore = incmemory.NewStore()
    case "production":
        incStore = incpg.NewStore(pgPool)
    }

    groupWindow := parseDuration(cfg.Incidents.GroupWindow, 5*time.Minute)
    incService := incident.NewService(incStore, regEngine.Store(), groupWindow)

    // Wire callbacks
    regEngine.SetAlertCallback(func(ctx context.Context, r regression.Regression) {
        incService.OnRegression(ctx, r)
    })
    sloEngine.SetAlertFunc(func(service, action string, burnRate, budgetUsed float64) {
        incService.OnSLOBreach(context.Background(), service, action, burnRate, budgetUsed)
    })

    adminAPI.SetIncidentService(incService)
}
```

- [ ] **Step 4: Run all tests**

Run: `go test -short ./pkg/incident/... ./pkg/regression/ ./pkg/gateway/ ./pkg/admin/ ./pkg/config/ -v`

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/admin/api.go cmd/gateway/main.go
git commit -m "feat(incident): wire incident service with API and alert callbacks

4 API endpoints, AlertCallback wired to regression engine,
SLOAlertFunc wired to SLO engine. Configurable group window."
```

---

## Task Summary

| Task | What it builds | Depends on |
|------|---------------|------------|
| 1 | Types + interfaces | — |
| 2 | In-memory + PostgreSQL stores | 1 |
| 3 | Incident service (grouping + auto-resolve) | 1, 2 |
| 4 | AlertSink callbacks in regression + SLO engines | — |
| 5 | Config + API + gateway wiring | 1-4 |
