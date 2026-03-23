# Phase 6: Enterprise Features — Design Specification

**Date:** 2026-03-23
**Status:** Approved
**Phase:** 6 of 6 (CloudMock Console)
**Depends on:** Phases 3-5

---

## 6a: Audit Logging

### Data Model

```go
type Entry struct {
    ID        string                 `json:"id"`
    Actor     string                 `json:"actor"`
    Action    string                 `json:"action"`
    Resource  string                 `json:"resource"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

type Logger interface {
    Log(ctx context.Context, entry Entry) error
    Query(ctx context.Context, filter Filter) ([]Entry, error)
}

type Filter struct {
    Actor    string
    Action   string
    Resource string
    Limit    int
}
```

### PostgreSQL table

```sql
CREATE TABLE audit_log (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor     TEXT NOT NULL,
    action    TEXT NOT NULL,
    resource  TEXT NOT NULL,
    details   JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_actor ON audit_log(actor, timestamp DESC);
CREATE INDEX idx_audit_action ON audit_log(action, timestamp DESC);
```

### Auditable actions

| Action | Resource | Triggered by |
|--------|----------|-------------|
| slo.rules.updated | slo:config | PUT /api/slo |
| deploy.created | deploy:{id} | POST /api/deploys |
| view.saved | view:{id} | POST /api/views |
| view.deleted | view:{id} | DELETE /api/views |
| incident.acknowledged | incident:{id} | POST /api/incidents/{id}/acknowledge |
| incident.resolved | incident:{id} | POST /api/incidents/{id}/resolve |
| regression.dismissed | regression:{id} | POST /api/regressions/{id}/dismiss |
| sourcemap.uploaded | sourcemap:{file} | POST /api/sourcemaps |
| chaos.rule.created | chaos:{id} | POST /api/chaos |
| chaos.rule.updated | chaos:{id} | PUT /api/chaos |
| chaos.rule.deleted | chaos:{id} | DELETE /api/chaos |

### API

```
GET /api/audit?actor=X&action=Y&resource=Z&limit=50
```

### File layout

```
pkg/audit/
├── types.go
├── postgres/
│   ├── logger.go
│   └── logger_test.go
└── memory/
    ├── logger.go
    └── logger_test.go
```

---

## 6b: RBAC Auth (JWT)

### Data Model

```go
type User struct {
    ID       string   `json:"id"`
    Email    string   `json:"email"`
    Name     string   `json:"name"`
    Role     string   `json:"role"`     // "admin", "editor", "viewer"
    TenantID string   `json:"tenant_id,omitempty"`
}

type Claims struct {
    jwt.RegisteredClaims
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Role     string `json:"role"`
    TenantID string `json:"tenant_id,omitempty"`
}
```

### Roles

| Role | Can read | Can mutate config | Can manage users |
|------|---------|-------------------|-----------------|
| viewer | all data | no | no |
| editor | all data | yes | no |
| admin | all data | yes | yes |

### Middleware

```go
func AuthMiddleware(secret []byte) func(http.Handler) http.Handler
```

Extracts JWT from `Authorization: Bearer {token}`, validates, injects `User` into request context. Returns 401 if missing/invalid. Skips auth for `GET /api/health`.

### API

```
POST /api/auth/login     — email + password → JWT token
POST /api/auth/register  — create user (admin only)
GET  /api/auth/me        — current user info
GET  /api/users          — list users (admin only)
PUT  /api/users/{id}     — update role (admin only)
```

### User store

PostgreSQL table `users` with bcrypt password hashes. In-memory for local mode.

### Config

```yaml
auth:
  enabled: false  # opt-in for dev tool
  secret: "change-me-in-production"
```

Disabled by default — local dev doesn't need auth. Production enables via config.

---

## 6c: API Rate Limiting

### Design

Token bucket per IP address. Middleware applied before auth.

```go
type RateLimiter struct {
    mu      sync.Mutex
    buckets map[string]*bucket
    rate    float64  // tokens per second
    burst   int      // max burst
}

func NewRateLimiter(rate float64, burst int) *RateLimiter
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler
```

Returns 429 Too Many Requests with `Retry-After` header.

### Config

```yaml
rate_limit:
  enabled: false
  requests_per_second: 100
  burst: 200
```

### File layout

```
pkg/ratelimit/
├── limiter.go
└── limiter_test.go
```

---

## 6d: Tenant Isolation

### Design

When auth is enabled and user has a `tenant_id`, all API queries are automatically scoped:
- RequestReader queries add `TenantID` filter
- TraceReader queries add `TenantID` filter
- Incidents/regressions filtered by `AffectedTenants` containing user's tenant

Implemented as a middleware that wraps the DataPlane with tenant-scoped proxies.

```go
type TenantScopedDataPlane struct {
    inner    *dataplane.DataPlane
    tenantID string
}
```

Admin users bypass tenant scoping.

### File layout

```
pkg/tenantscope/
├── scoped.go      # TenantScoped wrappers for Reader interfaces
└── scoped_test.go
```

---

## 6e: Webhook Integrations

### Design

Fire webhooks on incident creation/resolution.

```go
type WebhookConfig struct {
    URL     string            `json:"url"`
    Type    string            `json:"type"`    // "slack", "pagerduty", "generic"
    Events  []string          `json:"events"`  // "incident.created", "incident.resolved"
    Headers map[string]string `json:"headers,omitempty"`
}
```

Stored in PostgreSQL `webhooks` table. Webhook dispatcher listens to incident events and fires HTTP POST with incident payload formatted for each integration type.

### API

```
GET    /api/webhooks         — list
POST   /api/webhooks         — create (admin only)
DELETE /api/webhooks/{id}    — delete (admin only)
POST   /api/webhooks/{id}/test — send test payload
```

### Formatters

- **Slack:** `{"text": "🚨 Incident: {title}", "blocks": [...]}`
- **PagerDuty:** Events API v2 format
- **Generic:** Raw incident JSON

### File layout

```
pkg/webhook/
├── types.go
├── dispatcher.go
├── dispatcher_test.go
├── formatters.go    # Slack, PagerDuty, generic
├── formatters_test.go
├── postgres/
│   ├── store.go
│   └── store_test.go
└── memory/
    ├── store.go
    └── store_test.go
```

---

## 6f: Exportable Incident Reports

### Design

`GET /api/incidents/{id}/report?format=json|csv|html` generates a downloadable incident report including:
- Incident details (title, severity, timeline)
- Affected services and tenants
- Related regressions
- Top traces
- Root cause analysis

```go
type ReportGenerator struct {
    incidents incident.IncidentStore
    regressions regression.RegressionStore
    traces dataplane.TraceReader
}

func (g *ReportGenerator) Generate(ctx context.Context, id string, format string) ([]byte, string, error)
// returns: content, content-type, error
```

### File layout

```
pkg/report/
├── generator.go
└── generator_test.go
```

One new API endpoint wired into admin API.
