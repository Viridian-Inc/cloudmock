# Environment Variables

All CloudMock configuration can be overridden via environment variables. Environment variables take precedence over `.cloudmock.yaml` settings.

## CloudMock Variables

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_REGION` | `us-east-1` | AWS region to emulate |
| `CLOUDMOCK_PROFILE` | `standard` | Service profile (`minimal`, `standard`, `full`) |
| `CLOUDMOCK_SERVICES` | | Comma-separated list of services to enable (overrides profile) |

### Ports

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_GATEWAY_PORT` | `4566` | AWS service endpoint port |
| `CLOUDMOCK_ADMIN_PORT` | `4599` | Admin API port |
| `CLOUDMOCK_DASHBOARD_PORT` | `4500` | DevTools dashboard port |
| `CLOUDMOCK_OTLP_PORT` | `4318` | OTLP receiver port |

### Features

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_IAM_MODE` | `enforce` | IAM enforcement: `enforce`, `permissive`, `disabled` |
| `CLOUDMOCK_PERSIST` | `false` | Enable state persistence |
| `CLOUDMOCK_PERSIST_PATH` | | Directory for persistent data |
| `CLOUDMOCK_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `CLOUDMOCK_OTLP_ENABLED` | `true` | Enable OTLP ingestion endpoint |

### Data Plane

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_DATAPLANE_MODE` | `local` | Storage backend: `local`, `duckdb`, `postgresql` |
| `CLOUDMOCK_DUCKDB_PATH` | `cloudmock.duckdb` | DuckDB database file path |
| `CLOUDMOCK_POSTGRESQL_URL` | | PostgreSQL connection URL |
| `CLOUDMOCK_PROMETHEUS_URL` | | Prometheus remote write URL |
| `CLOUDMOCK_OTEL_ENDPOINT` | | Forward telemetry to external OTel collector |

### SaaS Mode

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_SAAS_ENABLED` | `false` | Enable hosted SaaS features |
| `CLERK_SECRET_KEY` | | Clerk authentication secret |
| `CLERK_WEBHOOK_SECRET` | | Clerk webhook verification secret |
| `CLERK_DOMAIN` | | Clerk frontend API domain |
| `STRIPE_SECRET_KEY` | | Stripe billing secret |
| `STRIPE_WEBHOOK_SECRET` | | Stripe webhook verification secret |
| `STRIPE_PRO_PRICE_ID` | | Stripe Pro plan price ID |
| `STRIPE_TEAM_PRICE_ID` | | Stripe Team plan price ID |
| `FLY_API_TOKEN` | | Fly.io API token for instance provisioning |
| `FLY_ORG` | | Fly.io organization |
| `FLY_REGION` | | Fly.io deployment region |
| `CLOUDFLARE_API_TOKEN` | | Cloudflare API token for DNS management |
| `CLOUDFLARE_ZONE_ID` | | Cloudflare DNS zone ID |

## AWS SDK Variables

These standard AWS variables control how your application connects to CloudMock:

| Variable | Recommended Value | Description |
|----------|-------------------|-------------|
| `AWS_ENDPOINT_URL` | `http://localhost:4566` | Directs all AWS SDK calls to CloudMock |
| `AWS_ACCESS_KEY_ID` | `test` | AWS access key (any value works unless IAM is enforced) |
| `AWS_SECRET_ACCESS_KEY` | `test` | AWS secret key |
| `AWS_DEFAULT_REGION` | `us-east-1` | AWS region |

### Per-Service Endpoint Override

AWS SDK v2 (Go, Java) and SDK v3 (Node.js) support per-service endpoint URLs:

```bash
# Override specific services
export AWS_ENDPOINT_URL_S3=http://localhost:4566
export AWS_ENDPOINT_URL_DYNAMODB=http://localhost:4566
export AWS_ENDPOINT_URL_SQS=http://localhost:4566
```

This is useful if you want some services to hit CloudMock and others to hit real AWS.

## OpenTelemetry Variables

Standard OTel environment variables for connecting your application's instrumentation:

| Variable | Recommended Value | Description |
|----------|-------------------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OTLP endpoint for traces, metrics, logs |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/json` | Protocol (`http/json` or `http/protobuf`) |
| `OTEL_SERVICE_NAME` | your service name | Identifies your service in traces |
| `OTEL_TRACES_EXPORTER` | `otlp` | Use OTLP for traces |
| `OTEL_METRICS_EXPORTER` | `otlp` | Use OTLP for metrics |
| `OTEL_LOGS_EXPORTER` | `otlp` | Use OTLP for logs |

## Example: Full Development Environment

```bash
# .env or shell profile
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
export OTEL_SERVICE_NAME=my-service

export CLOUDMOCK_PROFILE=standard
export CLOUDMOCK_LOG_LEVEL=debug
export CLOUDMOCK_PERSIST=true
```

## Example: Docker Compose

```yaml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
      - "4318:4318"
    environment:
      CLOUDMOCK_PROFILE: standard
      CLOUDMOCK_PERSIST: "true"
      CLOUDMOCK_PERSIST_PATH: /data
      CLOUDMOCK_LOG_LEVEL: info
    volumes:
      - cloudmock-data:/data

  app:
    build: .
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      OTEL_EXPORTER_OTLP_ENDPOINT: http://cloudmock:4318
      OTEL_SERVICE_NAME: my-app
    depends_on:
      - cloudmock

volumes:
  cloudmock-data:
```
