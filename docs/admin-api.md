# Admin API Reference

The CloudMock admin API runs on port **4599** by default (`CLOUDMOCK_ADMIN_PORT`). All endpoints return JSON unless otherwise noted.

Base URL: `http://localhost:4599`

---

## Health & System

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Health check with per-service status and dataplane connectivity |
| `GET` | `/api/version` | Build version, commit SHA, and build time |
| `GET` | `/api/config` | Current active configuration |
| `GET` | `/api/stats` | Aggregate request statistics |

## Services

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/services` | List all registered services with action counts and health |
| `GET` | `/api/services/{name}` | Detail for a single service (actions, health, resource count) |
| `POST` | `/api/services/{name}/reset` | Reset a single service's in-memory state |
| `POST` | `/api/reset` | Reset all services |
| `GET` | `/api/resources/{service}` | List resources for a service (buckets, tables, queues, etc.) |

## Requests

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/requests` | List recent requests. Query params: `limit`, `level` (app/infra/all), `service`, `action`, `status`, `min_latency_ms`, `max_latency_ms`, `caller_id` |
| `GET` | `/api/requests/{id}` | Get a single request by ID |
| `POST` | `/api/requests/{id}/replay` | Replay a captured request against the gateway |
| `GET` | `/api/stream` | SSE stream of requests in real time |
| `GET` | `/api/explain/{requestId}` | AI-ready context for a request: trace, timeline, analysis, narrative |

## Traces

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/traces` | List recent traces. Query params: `service`, `error` (true/false), `limit` |
| `GET` | `/api/traces/{id}` | Get a single trace with spans and timeline |
| `GET` | `/api/traces/compare` | Compare two traces or a trace against its route baseline. Query params: `a` (trace ID), `b` (trace ID), `baseline` (true/false) |

## Topology

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/topology` | Dynamic service topology graph (nodes and edges) |
| `GET` | `/api/topology/config` | Get IaC-derived topology configuration |
| `PUT` | `/api/topology/config` | Set IaC-derived topology configuration |
| `GET` | `/api/blast-radius` | Compute blast radius for a node. Query param: `node` |

## Metrics

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/metrics` | Aggregate metrics (request count, error rate, latency percentiles) |
| `GET` | `/api/metrics/timeline` | Time-series metrics for charting |
| `GET` | `/api/compare` | Before/after comparison of latency and error rate. Query params: `service`, `action` |

## SLO

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/slo` | Current SLO status: windows, health, alerts, rules |
| `PUT` | `/api/slo` | Update SLO rules (body: array of rule objects) |

## Deploys

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/deploys` | List recent deploy events |
| `POST` | `/api/deploys` | Record a deploy event (body: `{service, version, deployed_at}`) |

## Incidents

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/incidents` | List incidents. Query params: `status`, `severity`, `service`, `limit` |
| `GET` | `/api/incidents/{id}` | Get a single incident |
| `GET` | `/api/incidents/{id}/report` | Export incident report. Query param: `format` (json/csv/html) |
| `POST` | `/api/incidents/{id}/acknowledge` | Acknowledge an incident (body: `{owner}`) |
| `POST` | `/api/incidents/{id}/resolve` | Resolve an incident |

## Regressions

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/regressions` | List regressions. Query params: `service`, `deploy_id`, `severity`, `status`, `limit` |
| `GET` | `/api/regressions/{id}` | Get a single regression by ID |
| `POST` | `/api/regressions/{id}/dismiss` | Dismiss a regression |

## Cost

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cost` | Estimated AWS cost breakdown from recent traffic |
| `GET` | `/api/cost/routes` | Cost breakdown by route |
| `GET` | `/api/cost/tenants` | Cost breakdown by tenant |
| `GET` | `/api/cost/trend` | Cost trend over time |

## Tenants

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/tenants` | List all observed tenants with request counts and error rates |
| `GET` | `/api/tenants?id=CALLER_ID` | Detail for a specific tenant |
| `GET` | `/api/tenants/export` | Export tenant report as CSV |

## Profiling

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/profile/{service}` | Capture a profile. Query params: `type` (cpu/heap/goroutine), `duration`, `format` (flamegraph/pprof) |
| `GET` | `/api/profiles` | List captured profiles. Query param: `service` |
| `GET` | `/api/profiles/{id}` | Get a profile by ID. Query param: `format` (flamegraph/pprof) |
| `POST` | `/api/sourcemaps` | Upload a source map for symbolication. Query param: `file` |

## Chaos

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/chaos` | List chaos rules and active status |
| `POST` | `/api/chaos` | Create a chaos rule (body: rule object) |
| `DELETE` | `/api/chaos` | Disable all chaos rules |
| `PUT` | `/api/chaos/{id}` | Update a chaos rule |
| `DELETE` | `/api/chaos/{id}` | Delete a chaos rule |

## Shadow Testing

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/shadow` | Replay recent traffic against a target URL (body: `{target, service, limit}`) |

## Auth

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Authenticate and receive a JWT (body: `{email, password}`) |
| `POST` | `/api/auth/register` | Register a new user (admin only when auth enabled; body: `{email, password, name, role}`) |
| `GET` | `/api/auth/me` | Get the current authenticated user |

## Users

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/users` | List all users (admin only) |
| `PUT` | `/api/users/{id}` | Update a user's role (admin only; body: `{role}`) |

## Audit

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/audit` | Query audit log. Query params: `actor`, `action`, `resource`, `limit` |

## Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/webhooks` | List all webhook configurations |
| `POST` | `/api/webhooks` | Create a webhook (body: webhook config object) |
| `DELETE` | `/api/webhooks/{id}` | Delete a webhook |
| `POST` | `/api/webhooks/{id}/test` | Send a test payload to a webhook |

## Preferences

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/preferences?namespace=X` | List all preferences in a namespace |
| `GET` | `/api/preferences?namespace=X&key=Y` | Get a single preference value |
| `PUT` | `/api/preferences` | Set a preference (body: `{namespace, key, value}`) |
| `DELETE` | `/api/preferences?namespace=X&key=Y` | Delete a preference |

## Views

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/views` | List saved query views |
| `POST` | `/api/views` | Create a saved view (body: view object) |
| `DELETE` | `/api/views?id=VIEW_ID` | Delete a saved view |

## IAM

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/iam/evaluate` | Evaluate an IAM policy (body: `{principal, action, resource}`) |

## SES

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/ses/emails` | List captured SES emails |
| `GET` | `/api/ses/emails/{id}` | Get a single captured email |

## Lambda

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/lambda/logs` | Get Lambda execution logs |
| `GET` | `/api/lambda/logs/stream` | SSE stream of Lambda logs |
