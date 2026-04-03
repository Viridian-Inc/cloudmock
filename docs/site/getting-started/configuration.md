# Configuration

CloudMock is configured through `.cloudmock.yaml`, environment variables, and CLI flags. Configuration is resolved in this order (later overrides earlier):

1. Built-in defaults
2. `.cloudmock.yaml` (auto-discovered by walking up from current directory)
3. Environment variables
4. CLI flags

## .cloudmock.yaml

Create a `.cloudmock.yaml` in your project root. CloudMock auto-discovers it by walking up from your current working directory.

### Minimal Example

```yaml
# .cloudmock.yaml
profile: standard
region: us-east-1
```

### Full Example

```yaml
# .cloudmock.yaml

# AWS region to emulate
region: us-east-1

# AWS account ID
account_id: "000000000000"

# Service profile: minimal | standard | full | custom
profile: standard

# Gateway (AWS endpoint)
gateway:
  port: 4566

# Admin API
admin:
  port: 4599

# DevTools Dashboard
dashboard:
  port: 4500
  enabled: true

# OpenTelemetry OTLP endpoint
otlp:
  port: 4318
  enabled: true

# Infrastructure-as-Code auto-discovery
iac:
  path: ""       # auto-discovered if empty
  env: "dev"     # Pulumi stack or Terraform workspace

# IAM enforcement
iam:
  mode: enforce  # enforce | permissive | disabled
  root_access_key: test
  root_secret_key: test
  seed_file: ""  # path to IAM seed file (users, roles, policies)

# Persistence
persistence:
  enabled: false
  path: ""       # default: ~/.cloudmock/data

# Logging
logging:
  level: info    # debug | info | warn | error
  format: text   # text | json

# Data plane storage
dataplane:
  mode: local    # local | duckdb | postgresql
  duckdb:
    path: cloudmock.duckdb
  postgresql:
    url: ""
  prometheus:
    url: ""
  otel:
    collector_endpoint: ""
    service_name: cloudmock

# SLO tracking
slo:
  enabled: true
  rules:
    - service: "*"
      action: "*"
      p50_ms: 50
      p95_ms: 200
      p99_ms: 500
      error_rate: 0.01

# Regression detection
regression:
  enabled: true
  scan_interval: 5m
  window: 15m

# Incident management
incidents:
  enabled: true
  group_window: 5m

# Monitor (alerting engine)
monitor:
  enabled: true
  eval_interval: 30s

# Rate limiting
rate_limit:
  enabled: false
  requests_per_second: 100
  burst: 200

# Real User Monitoring
rum:
  enabled: true
  sample_rate: 1.0   # 0.0 to 1.0
  max_events: 10000

# Cost intelligence
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

# Admin API authentication
admin_auth:
  enabled: false
  api_key: ""

# JWT-based RBAC
auth:
  enabled: false
  secret: cloudmock-dev-secret-change-in-production

# Custom service selection (profile: custom)
services:
  s3:
    enabled: true
  dynamodb:
    enabled: true
  lambda:
    enabled: true
    runtimes:
      - nodejs20.x
      - python3.12
```

## Service Profiles

| Profile | Services | Startup Time |
|---------|----------|-------------|
| `minimal` | iam, sts, s3, dynamodb, sqs, sns, lambda, cloudwatch-logs | ~1s |
| `standard` | minimal + rds, cloudformation, ec2, ecr, ecs, secretsmanager, ssm, kinesis, firehose, events, stepfunctions, apigateway | ~2s |
| `full` | All 100 services | ~3s |
| `custom` | Only services listed in `services:` block | varies |

## Environment Variables

Every configuration field can be overridden via environment variable:

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_REGION` | `us-east-1` | AWS region |
| `CLOUDMOCK_GATEWAY_PORT` | `4566` | Gateway port |
| `CLOUDMOCK_ADMIN_PORT` | `4599` | Admin API port |
| `CLOUDMOCK_DASHBOARD_PORT` | `4500` | Dashboard port |
| `CLOUDMOCK_PROFILE` | `standard` | Service profile |
| `CLOUDMOCK_IAM_MODE` | `enforce` | IAM mode |
| `CLOUDMOCK_PERSIST` | `false` | Enable persistence |
| `CLOUDMOCK_PERSIST_PATH` | `""` | Persistence directory |
| `CLOUDMOCK_LOG_LEVEL` | `info` | Log level |
| `CLOUDMOCK_DATAPLANE_MODE` | `local` | Data plane storage mode |
| `CLOUDMOCK_DUCKDB_PATH` | `cloudmock.duckdb` | DuckDB file path |
| `CLOUDMOCK_POSTGRESQL_URL` | `""` | PostgreSQL connection URL |
| `CLOUDMOCK_PROMETHEUS_URL` | `""` | Prometheus URL |
| `CLOUDMOCK_OTEL_ENDPOINT` | `""` | OTel collector endpoint |
| `CLOUDMOCK_OTLP_PORT` | `4318` | OTLP receiver port |
| `CLOUDMOCK_OTLP_ENABLED` | `true` | Enable OTLP ingestion |
| `CLOUDMOCK_SERVICES` | `""` | Comma-separated service list (overrides profile) |

## CLI Flags

```bash
cmk start [flags]

Flags:
  --config <path>    Path to .cloudmock.yaml (auto-discovered if not set)
  --profile <name>   Service profile: minimal | standard | full
  --port <number>    Gateway port (default 4566)
  --fg               Run in foreground instead of daemonizing
```

## IAM Modes

| Mode | Behavior |
|------|----------|
| `enforce` | Validates IAM policies on every request. Rejects unauthorized calls with `AccessDenied`. |
| `permissive` | Logs IAM policy violations but allows all requests through. |
| `disabled` | No IAM checking. All requests are allowed. |

## Data Plane Modes

| Mode | Storage | Use Case |
|------|---------|----------|
| `local` | In-memory | Development. Data lost on restart. |
| `duckdb` | DuckDB file | Local persistence with fast analytical queries. |
| `postgresql` | PostgreSQL | Production / team sharing. |

## Config File Discovery

CloudMock walks up from your current directory looking for `.cloudmock.yaml`:

```
/home/user/projects/my-app/src/  ← you are here
/home/user/projects/my-app/.cloudmock.yaml  ← found!
```

This means you can place one config file in your project root and it works from any subdirectory.

## IaC Auto-Discovery

CloudMock auto-discovers Infrastructure-as-Code projects to build the topology view. It searches for:

1. `Pulumi.yaml` or `Pulumi.*.yaml` in current and parent directories
2. `terraform/` directory with `.tf` files
3. `infra/pulumi/` directory pattern

Override with explicit config:

```yaml
iac:
  path: ./infra/pulumi
  env: dev
```
