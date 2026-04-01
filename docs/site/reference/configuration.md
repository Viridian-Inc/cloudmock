# Full Configuration Reference

Complete reference for the `.cloudmock.yaml` configuration file and the internal `Config` struct.

## Top-Level Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `region` | string | `us-east-1` | AWS region to emulate |
| `account_id` | string | `000000000000` | AWS account ID |
| `profile` | string | `minimal` | Service profile: `minimal`, `standard`, `full`, `custom` |

## Gateway

```yaml
gateway:
  port: 4566
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `gateway.port` | int | `4566` | Port for AWS service endpoint |

## Admin API

```yaml
admin:
  port: 4599
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `admin.port` | int | `4599` | Port for the Admin REST API |

## Dashboard

```yaml
dashboard:
  enabled: true
  port: 4500
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dashboard.enabled` | bool | `true` | Enable/disable the DevTools dashboard |
| `dashboard.port` | int | `4500` | Port for the dashboard web UI |

## OTLP

```yaml
otlp:
  enabled: true
  port: 4318
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `otlp.enabled` | bool | `true` | Enable/disable the OTLP receiver |
| `otlp.port` | int | `4318` | Port for OTLP/HTTP endpoint |

## IAM

```yaml
iam:
  mode: enforce
  root_access_key: test
  root_secret_key: test
  seed_file: ""
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `iam.mode` | string | `enforce` | `enforce` (reject unauthorized), `permissive` (log only), `disabled` |
| `iam.root_access_key` | string | `test` | Root access key for initial authentication |
| `iam.root_secret_key` | string | `test` | Root secret key for initial authentication |
| `iam.seed_file` | string | `""` | Path to YAML file with pre-created users, roles, and policies |

## Persistence

```yaml
persistence:
  enabled: false
  path: ""
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `persistence.enabled` | bool | `false` | Enable state persistence across restarts |
| `persistence.path` | string | `""` | Directory for persistent data (defaults to `~/.cloudmock/data`) |

## Logging

```yaml
logging:
  level: info
  format: text
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `logging.level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `logging.format` | string | `text` | Log format: `text`, `json` |

## Data Plane

```yaml
dataplane:
  mode: local
  duckdb:
    path: cloudmock.duckdb
  postgresql:
    url: ""
  prometheus:
    url: ""
  otel:
    collector_endpoint: ""
    service_name: cloudmock
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dataplane.mode` | string | `local` | Storage backend: `local` (in-memory), `duckdb`, `postgresql` |
| `dataplane.duckdb.path` | string | `cloudmock.duckdb` | DuckDB database file path |
| `dataplane.postgresql.url` | string | `""` | PostgreSQL connection URL |
| `dataplane.prometheus.url` | string | `""` | Prometheus remote write URL |
| `dataplane.otel.collector_endpoint` | string | `""` | Forward telemetry to external OTel collector |
| `dataplane.otel.service_name` | string | `cloudmock` | Service name for exported telemetry |

## SLO

```yaml
slo:
  enabled: true
  rules:
    - service: "*"
      action: "*"
      p50_ms: 50
      p95_ms: 200
      p99_ms: 500
      error_rate: 0.01
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `slo.enabled` | bool | `true` | Enable SLO tracking |
| `slo.rules[].service` | string | | Service name or `*` for all |
| `slo.rules[].action` | string | | Action name or `*` for all |
| `slo.rules[].p50_ms` | float | | Target P50 latency in ms |
| `slo.rules[].p95_ms` | float | | Target P95 latency in ms |
| `slo.rules[].p99_ms` | float | | Target P99 latency in ms |
| `slo.rules[].error_rate` | float | | Max acceptable error rate (0.01 = 1%) |

## Regression Detection

```yaml
regression:
  enabled: true
  scan_interval: 5m
  window: 15m
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `regression.enabled` | bool | `true` | Enable automatic regression detection |
| `regression.scan_interval` | string | `5m` | How often to scan for regressions (Go duration) |
| `regression.window` | string | `15m` | Time window to analyze (Go duration) |

## Incidents

```yaml
incidents:
  enabled: true
  group_window: 5m
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `incidents.enabled` | bool | `true` | Enable incident management |
| `incidents.group_window` | string | `5m` | Time window for grouping alerts into incidents |

## Monitor

```yaml
monitor:
  enabled: true
  eval_interval: 30s
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `monitor.enabled` | bool | `true` | Enable the alerting engine |
| `monitor.eval_interval` | string | `30s` | How often to evaluate monitor rules |

## Rate Limiting

```yaml
rate_limit:
  enabled: false
  requests_per_second: 100
  burst: 200
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rate_limit.enabled` | bool | `false` | Enable rate limiting on the gateway |
| `rate_limit.requests_per_second` | float | `100` | Sustained request rate |
| `rate_limit.burst` | int | `200` | Maximum burst size |

## RUM

```yaml
rum:
  enabled: true
  sample_rate: 1.0
  max_events: 10000
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rum.enabled` | bool | `true` | Enable Real User Monitoring endpoints |
| `rum.sample_rate` | float | `1.0` | Sampling rate for RUM events (0.0 to 1.0) |
| `rum.max_events` | int | `10000` | Max events in the circular buffer |

## Cost

```yaml
cost:
  pricing:
    lambda:
      perGBSecond: 0.0000166667
      defaultMemoryMB: 128
    dynamodb:
      perRCU: 0.00000025
      perWCU: 0.00000125
    s3:
      perGET: 0.0000004
      perPUT: 0.000005
    sqs:
      perRequest: 0.0000004
    dataTransfer:
      perKB: 0.00000009
```

All pricing fields are floats representing USD per unit.

## Admin Auth

```yaml
admin_auth:
  enabled: false
  api_key: ""
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `admin_auth.enabled` | bool | `false` | Require API key for admin endpoints |
| `admin_auth.api_key` | string | `""` | The API key value |

## Auth (RBAC)

```yaml
auth:
  enabled: false
  secret: cloudmock-dev-secret-change-in-production
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `auth.enabled` | bool | `false` | Enable JWT-based authentication |
| `auth.secret` | string | `cloudmock-dev-...` | JWT signing secret |

## Services

```yaml
services:
  s3:
    enabled: true
  lambda:
    enabled: true
    port: 0
    runtimes:
      - nodejs20.x
      - python3.12
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `services.<name>.enabled` | bool | `true` | Enable/disable this service |
| `services.<name>.port` | int | `0` | Custom port (0 = use gateway) |
| `services.<name>.runtimes` | []string | `[]` | Lambda-specific: allowed runtimes |
