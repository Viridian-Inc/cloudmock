---
title: Admin API
description: Complete reference for the CloudMock admin API (46+ endpoints)
---

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

### GET /api/health

Returns the health status of every registered service and the overall system status.

```bash
curl http://localhost:4599/api/health
```

```json
{
  "status": "ok",
  "services": {
    "s3": true,
    "dynamodb": true,
    "sqs": true,
    "sns": true,
    "sts": true
  }
}
```

### GET /api/version

```bash
curl http://localhost:4599/api/version
```

```json
{
  "version": "0.1.0",
  "commit": "abc1234",
  "build_time": "2026-03-21T12:00:00Z"
}
```

---

## Services

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/services` | List all registered services with action counts and health |
| `GET` | `/api/services/{name}` | Detail for a single service (actions, health, resource count) |
| `POST` | `/api/services/{name}/reset` | Reset a single service's in-memory state |
| `POST` | `/api/reset` | Reset all services |
| `GET` | `/api/resources/{service}` | List resources for a service (buckets, tables, queues, etc.) |

### GET /api/services

```bash
curl http://localhost:4599/api/services
```

```json
[
  {"name": "s3", "actions": 10, "healthy": true, "resources": 3},
  {"name": "dynamodb", "actions": 12, "healthy": true, "resources": 5}
]
```

### POST /api/reset

Resets all services to their initial state, deleting all resources.

```bash
curl -X POST http://localhost:4599/api/reset
```

```json
{"reset": 5, "services": ["s3", "dynamodb", "sqs", "sns", "sts"]}
```

### POST /api/services/{name}/reset

Reset a single service:

```bash
curl -X POST http://localhost:4599/api/services/s3/reset
```

---

## Requests

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/requests` | List recent requests with filtering |
| `GET` | `/api/requests/{id}` | Get a single request by ID |
| `POST` | `/api/requests/{id}/replay` | Replay a captured request against the gateway |
| `GET` | `/api/stream` | SSE stream of requests in real time |
| `GET` | `/api/explain/{requestId}` | AI-ready context for a request |

### GET /api/requests

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `limit` | Maximum number of results (default: 100) |
| `level` | Request level filter: `app`, `infra`, or `all` |
| `service` | Filter by AWS service name |
| `action` | Filter by AWS action name |
| `status` | Filter by HTTP status code |
| `min_latency_ms` | Minimum latency in milliseconds |
| `max_latency_ms` | Maximum latency in milliseconds |
| `caller_id` | Filter by caller identity |

```bash
curl "http://localhost:4599/api/requests?service=s3&limit=10"
```

### GET /api/stream

Server-Sent Events stream. Each event is a JSON object:

```bash
curl -N http://localhost:4599/api/stream
```

```
data: {"type":"request","data":{"id":"req-123","service":"s3","action":"PutObject","status_code":200,"latency_ms":2}}
```

### POST /api/requests/{id}/replay

Re-sends a captured request to the gateway:

```bash
curl -X POST http://localhost:4599/api/requests/req-123/replay
```

---

## Traces

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/traces` | List recent traces |
| `GET` | `/api/traces/{id}` | Get a single trace with spans and timeline |
| `GET` | `/api/traces/compare` | Compare two traces or a trace against its baseline |

### GET /api/traces

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `service` | Filter by root service |
| `error` | Filter by error status (`true` / `false`) |
| `limit` | Maximum number of results |

```bash
curl "http://localhost:4599/api/traces?service=s3&limit=20"
```

### GET /api/traces/compare

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `a` | First trace ID |
| `b` | Second trace ID (omit if using baseline) |
| `baseline` | Compare against route baseline (`true` / `false`) |

```bash
curl "http://localhost:4599/api/traces/compare?a=trace-123&b=trace-456"
```

---

## Topology

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/topology` | Dynamic service topology graph (nodes and edges) |
| `GET` | `/api/topology/config` | Get IaC-derived topology configuration |
| `PUT` | `/api/topology/config` | Set IaC-derived topology configuration |
| `GET` | `/api/blast-radius` | Compute blast radius for a node |

### GET /api/topology

Returns the full topology graph:

```bash
curl http://localhost:4599/api/topology
```

```json
{
  "nodes": [
    {"id": "external:bff-service", "label": "BFF", "service": "bff", "type": "external", "group": "API"},
    {"id": "svc:dynamodb", "label": "DynamoDB", "service": "dynamodb", "type": "aws-service", "group": "Database"}
  ],
  "edges": [
    {"source": "external:bff-service", "target": "svc:dynamodb", "type": "invoke"}
  ]
}
```

### GET /api/blast-radius

```bash
curl "http://localhost:4599/api/blast-radius?node=svc:dynamodb"
```

---

## Metrics

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/metrics` | Aggregate metrics (request count, error rate, latency percentiles) |
| `GET` | `/api/metrics/timeline` | Time-series metrics for charting |
| `GET` | `/api/compare` | Before/after comparison of latency and error rate |

### GET /api/metrics

```bash
curl http://localhost:4599/api/metrics
```

### GET /api/compare

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `service` | Service to compare |
| `action` | Action to compare |

```bash
curl "http://localhost:4599/api/compare?service=s3&action=PutObject"
```

---

## SLO

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/slo` | Current SLO status: windows, health, alerts, rules |
| `PUT` | `/api/slo` | Update SLO rules |

### PUT /api/slo

```bash
curl -X PUT http://localhost:4599/api/slo \
  -H "Content-Type: application/json" \
  -d '[
    {"service": "s3", "latency_p99_ms": 100, "error_rate_threshold": 0.01},
    {"service": "dynamodb", "latency_p99_ms": 50, "error_rate_threshold": 0.005}
  ]'
```

---

## Deploys

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/deploys` | List recent deploy events |
| `POST` | `/api/deploys` | Record a deploy event |

### POST /api/deploys

```bash
curl -X POST http://localhost:4599/api/deploys \
  -H "Content-Type: application/json" \
  -d '{"service": "my-api", "version": "1.2.3", "deployed_at": "2026-03-21T12:00:00Z"}'
```

---

## Incidents

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/incidents` | List incidents |
| `GET` | `/api/incidents/{id}` | Get a single incident |
| `GET` | `/api/incidents/{id}/report` | Export incident report |
| `POST` | `/api/incidents/{id}/acknowledge` | Acknowledge an incident |
| `POST` | `/api/incidents/{id}/resolve` | Resolve an incident |

### GET /api/incidents

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `status` | Filter by status (active, acknowledged, resolved) |
| `severity` | Filter by severity |
| `service` | Filter by service |
| `limit` | Maximum results |

### GET /api/incidents/{id}/report

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `format` | Export format: `json`, `csv`, or `html` |

```bash
curl "http://localhost:4599/api/incidents/inc-123/report?format=html" > report.html
```

---

## Regressions

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/regressions` | List regressions |
| `GET` | `/api/regressions/{id}` | Get a single regression |
| `POST` | `/api/regressions/{id}/dismiss` | Dismiss a regression |

### GET /api/regressions

Query parameters: `service`, `deploy_id`, `severity`, `status`, `limit`.

---

## Cost

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cost` | Estimated AWS cost breakdown from recent traffic |
| `GET` | `/api/cost/routes` | Cost breakdown by route |
| `GET` | `/api/cost/tenants` | Cost breakdown by tenant |
| `GET` | `/api/cost/trend` | Cost trend over time |

### GET /api/cost

```bash
curl http://localhost:4599/api/cost
```

Returns estimated AWS costs based on the pricing model applied to observed traffic patterns.

---

## Tenants

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/tenants` | List all observed tenants with request counts and error rates |
| `GET` | `/api/tenants?id=CALLER_ID` | Detail for a specific tenant |
| `GET` | `/api/tenants/export` | Export tenant report as CSV |

---

## Profiling

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/profile/{service}` | Capture a profile |
| `GET` | `/api/profiles` | List captured profiles |
| `GET` | `/api/profiles/{id}` | Get a profile by ID |
| `POST` | `/api/sourcemaps` | Upload a source map for symbolication |

### GET /api/profile/{service}

Query parameters:

| Parameter | Description |
|-----------|-------------|
| `type` | Profile type: `cpu`, `heap`, or `goroutine` |
| `duration` | Duration to profile (e.g., `30s`) |
| `format` | Output format: `flamegraph` or `pprof` |

```bash
curl "http://localhost:4599/api/profile/s3?type=cpu&duration=30s&format=flamegraph"
```

---

## Chaos

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/chaos` | List chaos rules and active status |
| `POST` | `/api/chaos` | Create a chaos rule |
| `DELETE` | `/api/chaos` | Disable all chaos rules |
| `PUT` | `/api/chaos/{id}` | Update a chaos rule |
| `DELETE` | `/api/chaos/{id}` | Delete a chaos rule |

### POST /api/chaos

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "active": true,
    "rules": [
      {"service": "dynamodb", "type": "latency", "value": 2000},
      {"service": "s3", "action": "GetObject", "type": "error", "value": 500}
    ]
  }'
```

---

## Shadow Testing

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/shadow` | Replay recent traffic against a target URL |

### POST /api/shadow

```bash
curl -X POST http://localhost:4599/api/shadow \
  -H "Content-Type: application/json" \
  -d '{"target": "http://staging:4566", "service": "s3", "limit": 100}'
```

---

## Auth

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Authenticate and receive a JWT |
| `POST` | `/api/auth/register` | Register a new user (admin only) |
| `GET` | `/api/auth/me` | Get the current authenticated user |

### POST /api/auth/login

```bash
curl -X POST http://localhost:4599/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "secret"}'
```

---

## Users

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/users` | List all users (admin only) |
| `PUT` | `/api/users/{id}` | Update a user's role (admin only) |

---

## Audit

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/audit` | Query audit log |

Query parameters: `actor`, `action`, `resource`, `limit`.

---

## Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/webhooks` | List all webhook configurations |
| `POST` | `/api/webhooks` | Create a webhook |
| `DELETE` | `/api/webhooks/{id}` | Delete a webhook |
| `POST` | `/api/webhooks/{id}/test` | Send a test payload to a webhook |

### POST /api/webhooks

```bash
curl -X POST http://localhost:4599/api/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://hooks.slack.com/services/...",
    "events": ["incident.created", "regression.detected"],
    "active": true
  }'
```

---

## Preferences

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/preferences?namespace=X` | List all preferences in a namespace |
| `GET` | `/api/preferences?namespace=X&key=Y` | Get a single preference value |
| `PUT` | `/api/preferences` | Set a preference |
| `DELETE` | `/api/preferences?namespace=X&key=Y` | Delete a preference |

---

## Views

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/views` | List saved query views |
| `POST` | `/api/views` | Create a saved view |
| `DELETE` | `/api/views?id=VIEW_ID` | Delete a saved view |

---

## IAM

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/iam/evaluate` | Evaluate an IAM policy |

### POST /api/iam/evaluate

Test whether a principal would be allowed to perform an action:

```bash
curl -X POST http://localhost:4599/api/iam/evaluate \
  -H "Content-Type: application/json" \
  -d '{"principal": "arn:aws:iam::000000000000:user/ci-user", "action": "s3:PutObject", "resource": "arn:aws:s3:::my-bucket/*"}'
```

---

## SES

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/ses/emails` | List captured SES emails |
| `GET` | `/api/ses/emails/{id}` | Get a single captured email |

---

## Lambda

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/lambda/logs` | Get Lambda execution logs |
| `GET` | `/api/lambda/logs/stream` | SSE stream of Lambda logs |

---

## State Management

### Export State

`POST /api/state/export`

Export the current state of all services as a JSON file.

**Response:** JSON state file containing all S3 buckets, DynamoDB tables, SQS queues, SNS topics, Lambda functions, IAM resources, CloudWatch Logs groups, and Route53 zones.

### Import State

`POST /api/state/import`

Import a previously exported state file. The request body should be a JSON state file.

**Request Body:** JSON state file (same format as export response)

### Reset State

`POST /api/state/reset`

Clear all service state. All buckets, tables, queues, topics, functions, and other resources are deleted.
