# Admin API Reference

The Admin API runs on port 4599 (default) and provides programmatic access to all CloudMock features.

Base URL: `http://localhost:4599`

All endpoints accept and return JSON unless noted otherwise.

---

## System

### GET /api/version

Returns the CloudMock version.

```bash
curl http://localhost:4599/api/version
```

```json
{"version": "1.0.0", "go": "1.26", "os": "darwin/arm64"}
```

### GET /api/health

Returns health status of all running services.

```bash
curl http://localhost:4599/api/health
```

```json
{
  "status": "ok",
  "services": {
    "s3": true,
    "dynamodb": true,
    "lambda": true,
    "sqs": true
  }
}
```

### GET /api/config

Returns the current runtime configuration.

```bash
curl http://localhost:4599/api/config | jq '.'
```

### GET /api/stats

Returns aggregate statistics (total requests, error count, latency percentiles).

```bash
curl http://localhost:4599/api/stats | jq '.'
```

```json
{
  "total_requests": 1523,
  "error_count": 12,
  "avg_latency_ms": 4.2,
  "p50_ms": 2.1,
  "p95_ms": 15.3,
  "p99_ms": 42.7,
  "services": {"dynamodb": 823, "s3": 412, "lambda": 288}
}
```

### POST /api/reset

Resets all state (resources, requests, traces, errors, logs). Does not stop the server.

```bash
curl -X POST http://localhost:4599/api/reset
```

---

## Services

### GET /api/services

Lists all registered services and their status.

```bash
curl http://localhost:4599/api/services | jq '.'
```

```json
[
  {"name": "s3", "status": "healthy", "request_count": 412},
  {"name": "dynamodb", "status": "healthy", "request_count": 823}
]
```

### GET /api/services/{name}

Get details for a specific service.

```bash
curl http://localhost:4599/api/services/dynamodb | jq '.'
```

---

## Requests

### GET /api/requests

List recent AWS API requests with filtering.

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `service` | string | Filter by service name (e.g., `dynamodb`) |
| `action` | string | Filter by action name (e.g., `PutItem`) |
| `status` | string | Filter by status (`success`, `error`) |
| `limit` | int | Max results (default 100) |
| `offset` | int | Pagination offset |
| `since` | string | ISO timestamp, only requests after this time |

```bash
curl "http://localhost:4599/api/requests?service=dynamodb&limit=10" | jq '.'
```

```json
[
  {
    "id": "req_abc123",
    "service": "dynamodb",
    "action": "PutItem",
    "status": 200,
    "duration_ms": 3.2,
    "timestamp": "2026-03-31T14:23:01.234Z",
    "request_body": {"TableName": "users", "Item": {"userId": {"S": "user-1"}}},
    "response_body": {}
  }
]
```

### GET /api/requests/{id}

Get full details for a single request including headers, body, and IAM evaluation.

```bash
curl http://localhost:4599/api/requests/req_abc123 | jq '.'
```

### GET /api/stream

Server-Sent Events stream of requests in real time.

```bash
curl -N http://localhost:4599/api/stream
```

---

## Traces

### GET /api/traces

List distributed traces.

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `service` | string | Filter by service |
| `limit` | int | Max results |
| `min_duration_ms` | float | Minimum trace duration |
| `has_error` | bool | Only traces with errors |

```bash
curl "http://localhost:4599/api/traces?limit=10" | jq '.'
```

### GET /api/traces/{id}

Get a full trace with all spans.

```bash
curl http://localhost:4599/api/traces/trace_abc123 | jq '.'
```

```json
{
  "trace_id": "trace_abc123",
  "root_span": "api-gateway: POST /orders",
  "duration_ms": 142.5,
  "span_count": 8,
  "services": ["api-gateway", "order-service", "dynamodb"],
  "has_error": false,
  "spans": [
    {
      "span_id": "span_001",
      "parent_id": null,
      "name": "POST /orders",
      "service": "api-gateway",
      "duration_ms": 142.5,
      "status": "OK",
      "attributes": {"http.method": "POST", "http.url": "/orders"}
    }
  ]
}
```

### GET /api/traces/compare

Compare two traces side by side.

```bash
curl "http://localhost:4599/api/traces/compare?a=trace_abc&b=trace_xyz" | jq '.'
```

---

## Metrics

### GET /api/metrics

Aggregate metrics by service.

```bash
curl http://localhost:4599/api/metrics | jq '.'
```

### GET /api/metrics/timeline

Time-series metrics data for charting.

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `service` | string | Filter by service |
| `metric` | string | Metric name (`latency`, `error_rate`, `throughput`) |
| `window` | string | Time window (`1h`, `6h`, `24h`) |
| `interval` | string | Bucket interval (`1m`, `5m`, `15m`) |

```bash
curl "http://localhost:4599/api/metrics/timeline?service=dynamodb&metric=latency&window=1h&interval=1m" | jq '.'
```

### POST /api/metrics/query

Execute a custom metric query.

```bash
curl -X POST http://localhost:4599/api/metrics/query \
  -H "Content-Type: application/json" \
  -d '{"query": "avg(latency_ms) by service where action = '\''PutItem'\''", "window": "1h"}'
```

---

## Errors

### GET /api/errors

List error groups.

```bash
curl http://localhost:4599/api/errors | jq '.'
```

### GET /api/errors/{id}

Get error group details with stack trace and occurrences.

```bash
curl http://localhost:4599/api/errors/err_abc123 | jq '.'
```

### POST /api/errors/ingest

Ingest an error from an external source.

```bash
curl -X POST http://localhost:4599/api/errors/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "message": "TypeError: Cannot read property id of undefined",
    "type": "TypeError",
    "stack": "...",
    "service": "order-service",
    "tags": {"release": "1.2.3"}
  }'
```

---

## Logs

### GET /api/logs

Query logs with filtering.

**Query parameters:** `service`, `level`, `q` (search text), `limit`, `since`

```bash
curl "http://localhost:4599/api/logs?service=order-service&level=error&limit=50" | jq '.'
```

### GET /api/logs/stream

Server-Sent Events stream of logs in real time.

```bash
curl -N http://localhost:4599/api/logs/stream
```

### POST /api/logs/ingest

Ingest log entries.

```bash
curl -X POST http://localhost:4599/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '{"level":"error","message":"Payment failed","service":"payment-service","trace_id":"abc123"}'
```

### GET /api/logs/services

List services that have emitted logs.

### GET /api/logs/levels

List available log levels.

---

## Lambda

### GET /api/lambda/logs

List Lambda function execution logs.

### GET /api/lambda/logs/stream

Server-Sent Events stream of Lambda logs.

---

## Topology

### GET /api/topology

Get the full resource topology graph (nodes and edges).

```bash
curl http://localhost:4599/api/topology | jq '.'
```

### GET /api/topology/config

Get IaC configuration source and discovery status.

### GET /api/resources/{service}

Get resources for a specific service.

```bash
curl http://localhost:4599/api/resources/lambda | jq '.'
```

---

## SLO

### GET /api/slo

Get SLO status for all services.

```bash
curl http://localhost:4599/api/slo | jq '.'
```

```json
{
  "rules": [
    {
      "service": "dynamodb",
      "action": "Query",
      "p50_target": 10,
      "p50_actual": 3.2,
      "p95_target": 50,
      "p95_actual": 12.1,
      "p99_target": 100,
      "p99_actual": 45.3,
      "error_rate_target": 0.001,
      "error_rate_actual": 0.0,
      "status": "healthy"
    }
  ]
}
```

---

## Cost

### GET /api/cost

Get aggregated cost estimates.

```bash
curl http://localhost:4599/api/cost | jq '.'
```

### GET /api/cost/routes

Cost breakdown by API route.

### GET /api/cost/tenants

Cost breakdown by tenant.

### GET /api/cost/trend

Cost trend over time.

---

## Chaos

### GET /api/chaos

List active chaos rules.

### POST /api/chaos

Create a new chaos rule.

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{"service":"dynamodb","action":"PutItem","fault":"error","error_code":"InternalServerError","probability":0.5,"duration":"5m"}'
```

### DELETE /api/chaos/{id}

Delete a chaos rule.

### DELETE /api/chaos

Delete all chaos rules.

---

## Monitors

### GET /api/monitors

List all monitors.

### POST /api/monitors

Create a monitor.

### GET /api/monitors/{id}

Get monitor details.

### PUT /api/monitors/{id}

Update a monitor.

### DELETE /api/monitors/{id}

Delete a monitor.

---

## Alerts

### GET /api/alerts

List alerts. Query params: `severity`, `status`, `limit`.

### GET /api/alerts/{id}

Get alert details.

### PUT /api/alerts/{id}

Update alert status (acknowledge, resolve).

---

## Incidents

### GET /api/incidents

List incidents.

### GET /api/incidents/{id}

Get incident details with timeline.

---

## Webhooks

### GET /api/webhooks

List webhook configurations.

### POST /api/webhooks

Create a webhook.

### PUT /api/webhooks/{id}

Update a webhook.

### DELETE /api/webhooks/{id}

Delete a webhook.

---

## Regressions

### GET /api/regressions

List detected performance regressions.

```bash
curl http://localhost:4599/api/regressions | jq '.'
```

---

## Dashboards

### GET /api/dashboards

List custom dashboards.

### POST /api/dashboards

Create a dashboard.

### GET /api/dashboards/{id}

Get dashboard configuration.

### PUT /api/dashboards/{id}

Update a dashboard.

### DELETE /api/dashboards/{id}

Delete a dashboard.

---

## Deploys

### GET /api/deploys

List deploy events.

### POST /api/deploys

Record a deploy event.

```bash
curl -X POST http://localhost:4599/api/deploys \
  -H "Content-Type: application/json" \
  -d '{"service":"order-service","version":"1.2.3","commit":"abc123","deployer":"ci"}'
```

---

## Tenants

### GET /api/tenants

List tracked tenants (multi-tenant observability).

### GET /api/tenants/export

Export tenant data.

---

## Views

### GET /api/views

List saved views.

### POST /api/views

Create a saved view (filtered request/trace query).

---

## Blast Radius

### POST /api/blast-radius

Analyze blast radius for a resource failure.

```bash
curl -X POST http://localhost:4599/api/blast-radius \
  -H "Content-Type: application/json" \
  -d '{"resource": "dynamodb:orders"}'
```

---

## Shadow Testing

### POST /api/shadow

Run a shadow test comparing traffic against a different endpoint.

---

## Compare

### POST /api/compare

Compare two time windows of metrics for before/after analysis.

---

## Explain

### GET /api/explain/{request_id}

Get an AI-generated debug explanation for a request or error.

```bash
curl http://localhost:4599/api/explain/req_abc123 | jq '.'
```

---

## Profiling

### GET /api/profile/{service}

Get profiling data for a service.

### GET /api/profiles

List available profiles.

---

## RUM (Real User Monitoring)

### POST /api/rum/events

Ingest RUM events from the browser SDK.

### GET /api/rum/vitals

Get Web Vitals metrics (LCP, FID, CLS, TTFB, FCP).

### GET /api/rum/pages

Get per-page performance breakdown.

### GET /api/rum/errors

Get browser-side errors captured by RUM.

### GET /api/rum/sessions

List user sessions.

---

## SES (Email)

### GET /api/ses/emails

List emails sent via the SES emulation.

### GET /api/ses/emails/{id}

Get email content for a specific sent email.

---

## IAM

### POST /api/iam/evaluate

Evaluate an IAM policy against an action.

```bash
curl -X POST http://localhost:4599/api/iam/evaluate \
  -H "Content-Type: application/json" \
  -d '{"principal":"arn:aws:iam::000000000000:user/alice","action":"s3:GetObject","resource":"arn:aws:s3:::my-bucket/*"}'
```

---

## Source SDK

### POST /api/source/events

Ingest events from source SDKs (@cloudmock/node).

### GET /api/source/status

Get source SDK connection status.

---

## Plugins

### GET /api/plugins

List installed plugins.

### GET /api/plugins/{name}

Get plugin details.

---

## Traffic Recording & Replay

### GET /api/traffic/recordings

List traffic recordings.

### POST /api/traffic/record

Start recording traffic.

### POST /api/traffic/record/stop

Stop recording.

### POST /api/traffic/replay

Start replaying a recording.

### GET /api/traffic/replay/{id}

Get replay status.

### GET /api/traffic/runs

List replay runs.

### POST /api/traffic/synthetic

Generate synthetic traffic.

### POST /api/traffic/compare

Compare two traffic recordings.

---

## Source Maps

### POST /api/sourcemaps

Upload a source map for JavaScript error de-minification.

---

## Audit

### GET /api/audit

Get audit log of admin actions.

---

## Auth

### POST /api/auth/login

Authenticate and receive a JWT token. Only active when `auth.enabled: true`.

```bash
curl -X POST http://localhost:4599/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "..."}'
```

### POST /api/auth/register

Register a new user.

### GET /api/auth/me

Get the authenticated user's profile.

---

## Users

### GET /api/users

List users (admin only).

### GET /api/users/{id}

Get user details.

---

## Preferences

### GET /api/preferences

Get user preferences.

### PUT /api/preferences

Update user preferences.

---

## Usage & Subscription (SaaS)

These endpoints are only active when `saas.enabled: true`.

### GET /api/usage

Get usage metrics for the current tenant.

### GET /api/subscription

Get subscription details.
