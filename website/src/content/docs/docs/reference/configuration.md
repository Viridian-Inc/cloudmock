---
title: Configuration
description: Complete reference for cloudmock.yml, service profiles, persistence, IAM modes, and environment variables
---

CloudMock is configured via a YAML file (default: `cloudmock.yml` in the working directory) with environment variable overrides applied on top.

## Full cloudmock.yml reference

```yaml
# AWS region to emulate
region: us-east-1

# Simulated AWS account ID (12 digits)
account_id: "000000000000"

# Service profile: minimal | standard | full | custom
# Controls which AWS services are started. See "Service Profiles" below.
profile: minimal

iam:
  # IAM enforcement mode: enforce | authenticate | none
  mode: enforce

  # Root credentials accepted by all modes except "none"
  root_access_key: test
  root_secret_key: test

  # Optional path to a JSON file for seeding IAM users, roles, and policies
  # seed_file: /etc/cloudmock/iam-seed.json

persistence:
  # Persist in-memory state to disk on shutdown and restore on startup
  enabled: false
  # Directory for state snapshots (created if it does not exist)
  # path: /var/lib/cloudmock/data

gateway:
  # Port for the main AWS API endpoint
  port: 4566

dashboard:
  # Web UI for inspecting service state
  enabled: true
  port: 4500

admin:
  # Control-plane REST API (used by the cloudmock CLI and devtools)
  port: 4599

logging:
  # Log level: debug | info | warn | error
  level: info
  # Log format: text (human-readable) | json (structured)
  format: text

# Per-service overrides -- used with profile: custom, or to change defaults
# for a specific service while using another profile.
#
# services:
#   s3:
#     enabled: true
#   lambda:
#     enabled: true
#     runtimes:
#       - nodejs20.x
#       - python3.12
#   dynamodb:
#     enabled: false   # disable one service from a named profile
```

## Service profiles

Profiles control which AWS services start with the gateway. Choose a profile based on how many services your application uses.

### minimal

Starts the smallest useful set of services (8 services):

```
iam, sts, s3, dynamodb, sqs, sns, lambda, cloudwatch-logs
```

Suitable for applications that use only core compute and storage services. This is the default.

### standard

Starts all commonly used production services (20 services):

```
iam, sts, s3, dynamodb, sqs, sns, lambda, cloudwatch-logs,
rds, cloudformation, ec2, ecr, ecs, secretsmanager, ssm,
kinesis, firehose, events, stepfunctions, apigateway
```

### full

Starts all 99 fully emulated AWS services. Use this when your application depends on less common services, or when you want full coverage without listing services individually.

### custom

Only the services explicitly listed under the `services` key are started:

```yaml
profile: custom
services:
  s3:
    enabled: true
  dynamodb:
    enabled: true
  sqs:
    enabled: true
```

You can also use the `CLOUDMOCK_SERVICES` environment variable for a quick override without editing the config file:

```bash
CLOUDMOCK_SERVICES=s3,dynamodb,sqs cloudmock start
```

### Per-service overrides with named profiles

You can override individual services while using a named profile. For example, to use the `standard` profile but disable EC2:

```yaml
profile: standard
services:
  ec2:
    enabled: false
```

Or to add a service not included in the profile:

```yaml
profile: minimal
services:
  cognito-idp:
    enabled: true
```

## Ports

CloudMock uses three ports:

| Port | Config key | Env var | Description |
|------|-----------|---------|-------------|
| 4566 | `gateway.port` | `CLOUDMOCK_GATEWAY_PORT` | Main AWS API endpoint. All AWS SDK/CLI traffic goes here. |
| 4500 | `dashboard.port` | `CLOUDMOCK_DASHBOARD_PORT` | Devtools web UI. Open in a browser to access the dashboard. |
| 4599 | `admin.port` | `CLOUDMOCK_ADMIN_PORT` | Admin/control-plane API. Used by the `cloudmock` CLI and devtools. |

All three ports are configurable. The dashboard can be disabled entirely:

```yaml
dashboard:
  enabled: false
```

## Persistence backends

### In-memory (default)

By default, all state is held in memory. It is fast and requires no setup, but all data is lost when the process exits.

### Snapshot persistence

When `persistence.enabled: true`, CloudMock writes a state snapshot to `persistence.path` on clean shutdown and restores it on startup. The snapshot format is an internal JSON representation of each service's in-memory store.

```yaml
persistence:
  enabled: true
  path: /var/lib/cloudmock/data
```

State is not automatically synced during operation -- only on shutdown. If the process is killed without a clean shutdown, the previous snapshot is loaded.

### DuckDB (production mode)

For durable analytical storage, enable production data plane mode with DuckDB. DuckDB is an embedded columnar database that stores requests, traces, SLO windows, regressions, and incidents in a single file.

```yaml
dataplane:
  mode: production
  duckdb_path: cloudmock.duckdb
```

Or via environment variable:

```bash
CLOUDMOCK_DATAPLANE_MODE=production
CLOUDMOCK_DUCKDB_PATH=./data/cloudmock.duckdb
```

### PostgreSQL (production mode)

For multi-user and team environments, production mode supports PostgreSQL for configuration and operational data (users, webhooks, saved views, deploy events, preferences, audit log).

```yaml
dataplane:
  mode: production
  postgresql_url: postgres://user:pass@localhost:5432/cloudmock
```

Or via environment variable:

```bash
CLOUDMOCK_POSTGRESQL_URL=postgres://user:pass@localhost:5432/cloudmock
```

### Prometheus (production mode)

For time-series metrics, production mode can read from a Prometheus instance for the metrics timeline API.

```bash
CLOUDMOCK_PROMETHEUS_URL=http://localhost:9090
```

### OpenTelemetry Collector (production mode)

For exporting telemetry, production mode can forward traces, metrics, and logs to an OTel Collector.

```bash
CLOUDMOCK_OTEL_ENDPOINT=localhost:4317
```

### Backend summary

| Backend | Stores | Required |
|---------|--------|----------|
| **DuckDB** | Requests, traces, SLO windows, regressions, incidents | No -- falls back to in-memory |
| **PostgreSQL** | Users, webhooks, saved views, deploy events, preferences, audit log | No -- falls back to in-memory |
| **Prometheus** | Metrics time series | No -- metrics computed from traces |
| **OTel Collector** | Trace/metric/log export | No -- telemetry stays local |

You can enable production mode with any subset of backends. Unconfigured backends fall back to in-memory storage.

## IAM modes

### enforce (default)

Requests must include valid AWS Signature V4 credentials. The IAM engine evaluates every request against attached policies. Requests without an explicit Allow are denied.

This mode is suitable for testing IAM policies and reproducing permission errors locally.

### authenticate

Credentials are validated (the access key must exist in the store) but policy evaluation is skipped. All authenticated requests succeed.

This mode is useful when you want to verify that your application sends valid credentials without dealing with policy configuration.

### none

All authentication and authorization checks are bypassed. Any request is accepted regardless of credentials. Useful for rapid prototyping, but not safe for multi-user environments.

### Root credentials

The `root_access_key` and `root_secret_key` values define a superuser credential that bypasses all policy checks (in `enforce` and `authenticate` modes). The defaults are both `test`, matching the convention used by other AWS emulators.

### IAM seed file

If `iam.seed_file` is set, CloudMock loads users, access keys, and policies from a JSON file at startup:

```json
{
  "users": [
    {
      "name": "ci-user",
      "access_key_id": "AKIAIOSFODNN7EXAMPLE",
      "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "policies": [
        {
          "name": "AllowS3",
          "document": {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": ["s3:*"],
                "Resource": "*"
              }
            ]
          }
        }
      ]
    }
  ]
}
```

## Environment variable reference

All environment variables override the corresponding value in `cloudmock.yml`.

| Variable | Description | Default |
|----------|-------------|---------|
| `CLOUDMOCK_GATEWAY_PORT` | Gateway HTTP port | `4566` |
| `CLOUDMOCK_ADMIN_PORT` | Admin API port | `4599` |
| `CLOUDMOCK_DASHBOARD_PORT` | Dashboard port | `4500` |
| `CLOUDMOCK_DATAPLANE_MODE` | Storage mode (`local` / `production`) | `local` |
| `CLOUDMOCK_DUCKDB_PATH` | DuckDB file path | `cloudmock.duckdb` |
| `CLOUDMOCK_POSTGRESQL_URL` | PostgreSQL connection URL | -- |
| `CLOUDMOCK_PROMETHEUS_URL` | Prometheus URL | -- |
| `CLOUDMOCK_OTEL_ENDPOINT` | OTel Collector endpoint | -- |
| `CLOUDMOCK_LOG_FORMAT` | Log format (`text` / `json`) | `text` |
| `CLOUDMOCK_LOG_LEVEL` | Log level (`debug` / `info` / `warn` / `error`) | `info` |
| `CLOUDMOCK_REGION` | AWS region to emulate | `us-east-1` |
| `CLOUDMOCK_IAM_MODE` | IAM mode (`enforce` / `authenticate` / `none`) | `none` |
| `CLOUDMOCK_PERSIST` | Enable persistence (`true` / `false`) | `false` |
| `CLOUDMOCK_PERSIST_PATH` | Directory for state snapshots | -- |
| `CLOUDMOCK_SERVICES` | Comma-separated list of services to enable | -- |
| `CLOUDMOCK_PROFILE` | Service profile (overrides config file) | -- |
| `CLOUDMOCK_ADMIN_ADDR` | Address the CLI uses to reach the admin API | `http://localhost:4599` |

### Example

```bash
CLOUDMOCK_REGION=eu-west-1 \
CLOUDMOCK_IAM_MODE=none \
CLOUDMOCK_LOG_LEVEL=debug \
./bin/cloudmock start
```

## Logging

### Text format (default)

Human-readable output for terminal use:

```
2026-03-21 12:00:00 INFO  s3 PutObject 200 1ms
```

### JSON format

Structured output for log aggregation systems:

```json
{"time":"2026-03-21T12:00:00Z","level":"INFO","msg":"request","service":"s3","action":"PutObject","status":200,"duration_ms":1}
```

Configure with:

```yaml
logging:
  level: debug
  format: json
```

## State Persistence

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--state` | `CLOUDMOCK_STATE` | (none) | Path to state file. Loaded on startup, saved on shutdown. |
| `--persist` | `CLOUDMOCK_PERSIST` | `false` | Auto-save state on shutdown (SIGTERM/SIGINT) |

### Example

```bash
# Start with pre-configured state
cloudmock --state cloudmock-state.json

# Auto-save state on shutdown
cloudmock --state cloudmock-state.json --persist
```

### YAML Configuration

```yaml
persistence:
  enabled: true
  path: ./cloudmock-state.json
```

## Config file location

CloudMock looks for `cloudmock.yml` in the following order:

1. Path specified by `-config` flag: `cloudmock start -config /etc/cloudmock/prod.yml`
2. `cloudmock.yml` in the current working directory
3. Built-in defaults (minimal profile, enforce IAM, ports 4566/4500/4599)
