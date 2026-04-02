---
title: Production Mode
description: Deploy CloudMock with DuckDB, PostgreSQL, Prometheus, and OpenTelemetry for persistent, scalable operation
---

CloudMock ships with two data plane modes. **Local mode** (the default) stores everything in memory and is ideal for development and quick prototyping. **Production mode** adds persistent storage backends for durability, team collaboration, and integration with existing observability infrastructure.

This guide covers how to configure each backend and provides Docker Compose examples for common deployment patterns.

## In-memory mode (local)

The default mode. All state -- AWS resources, request logs, traces, metrics, and preferences -- is held in memory. No external dependencies are required.

```yaml
dataplane:
  mode: local
```

Characteristics:

- **Startup time:** Instant. No migrations, no connections to establish.
- **Request log:** In-memory ring buffer holding the last 10,000 entries.
- **Persistence:** Optional snapshot-on-shutdown via `persistence.enabled: true`. Not suitable for high-reliability scenarios.
- **Multi-user:** Not supported. Each instance is single-tenant.
- **Best for:** Local development, CI pipelines, quick demos.

```bash
# Start in local mode (the default)
npx cloudmock start
```

## DuckDB mode (file-based, persistent)

DuckDB is an embedded columnar database that stores request history, traces, SLO windows, regressions, and incidents in a single file. No separate database server is required.

```yaml
dataplane:
  mode: production
  duckdb_path: cloudmock.duckdb
```

Or via environment variable:

```bash
CLOUDMOCK_DATAPLANE_MODE=production \
CLOUDMOCK_DUCKDB_PATH=./data/cloudmock.duckdb \
npx cloudmock start
```

Characteristics:

- **Storage:** All analytical data is written to a `.duckdb` file on disk. Data survives restarts.
- **Query performance:** DuckDB is optimized for analytical queries (aggregations, time-range scans). The dashboard and metrics APIs are significantly faster with large request volumes.
- **Concurrency:** Single-writer. DuckDB supports one CloudMock instance per file. For multi-instance deployments, use PostgreSQL.
- **Schema migrations:** CloudMock runs migrations automatically on startup.
- **Best for:** Solo or small-team development, staging environments, persistent test environments.

### Docker Compose with DuckDB

```yaml
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
    environment:
      CLOUDMOCK_DATAPLANE_MODE: production
      CLOUDMOCK_DUCKDB_PATH: /data/cloudmock.duckdb
      CLOUDMOCK_PROFILE: standard
    volumes:
      - cloudmock-data:/data

volumes:
  cloudmock-data:
```

## PostgreSQL mode (production, scalable)

PostgreSQL stores configuration and operational data: users, webhooks, saved views, deploy events, preferences, and the audit log. It is designed for team environments where multiple users share a CloudMock instance.

```yaml
dataplane:
  mode: production
  postgresql_url: postgres://cloudmock:secret@localhost:5432/cloudmock
```

Or via environment variable:

```bash
CLOUDMOCK_POSTGRESQL_URL=postgres://cloudmock:secret@localhost:5432/cloudmock
```

Characteristics:

- **Storage:** Relational data for multi-user features (auth, saved views, webhooks, audit trail).
- **Concurrency:** Full multi-writer support. Multiple CloudMock instances can share a PostgreSQL database.
- **Schema migrations:** CloudMock runs migrations automatically on startup.
- **Best for:** Team environments, shared staging, production observability.

PostgreSQL is complementary to DuckDB. For a full production deployment, use both:

- **DuckDB** for request history, traces, and analytical queries.
- **PostgreSQL** for users, webhooks, saved views, and audit logs.

### Docker Compose with PostgreSQL

```yaml
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
    environment:
      CLOUDMOCK_DATAPLANE_MODE: production
      CLOUDMOCK_DUCKDB_PATH: /data/cloudmock.duckdb
      CLOUDMOCK_POSTGRESQL_URL: postgres://cloudmock:secret@postgres:5432/cloudmock
      CLOUDMOCK_PROFILE: standard
    volumes:
      - cloudmock-data:/data
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: cloudmock
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: cloudmock
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cloudmock"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  cloudmock-data:
  postgres-data:
```

## Prometheus integration (metrics export)

CloudMock can read time-series metrics from a Prometheus instance. When configured, the metrics timeline API (`GET /api/metrics/timeline`) queries Prometheus via PromQL instead of computing metrics from the in-memory request log. This provides longer retention and more accurate historical data.

```yaml
dataplane:
  mode: production
  prometheus_url: http://localhost:9090
```

Or via environment variable:

```bash
CLOUDMOCK_PROMETHEUS_URL=http://localhost:9090
```

CloudMock itself does not write to Prometheus. To get metrics into Prometheus, export them via the OpenTelemetry Collector (see below) or scrape the CloudMock admin API's `/metrics` endpoint.

### Docker Compose with Prometheus

```yaml
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
    environment:
      CLOUDMOCK_DATAPLANE_MODE: production
      CLOUDMOCK_DUCKDB_PATH: /data/cloudmock.duckdb
      CLOUDMOCK_PROMETHEUS_URL: http://prometheus:9090
      CLOUDMOCK_OTEL_ENDPOINT: otel-collector:4317
      CLOUDMOCK_PROFILE: standard
    volumes:
      - cloudmock-data:/data

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    ports:
      - "4317:4317"
    volumes:
      - ./otel-config.yml:/etc/otelcol-contrib/config.yaml

volumes:
  cloudmock-data:
  prometheus-data:
```

Example `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "cloudmock"
    static_configs:
      - targets: ["cloudmock:4599"]
```

## OpenTelemetry integration (distributed tracing)

CloudMock can forward traces, metrics, and logs to an OpenTelemetry Collector via OTLP gRPC. This lets you pipe CloudMock telemetry into Jaeger, Grafana Tempo, Honeycomb, Datadog, or any OTLP-compatible backend.

```yaml
dataplane:
  mode: production
  otel_endpoint: localhost:4317
```

Or via environment variable:

```bash
CLOUDMOCK_OTEL_ENDPOINT=localhost:4317
```

The endpoint should point to an OTel Collector's OTLP gRPC receiver (default port 4317). CloudMock sends:

- **Traces** for every AWS API request, with spans for gateway routing, IAM evaluation, and service handling.
- **Metrics** for request counts, latency histograms, and error rates.
- **Logs** for structured request logs in OTLP format.

Example `otel-config.yml` for forwarding to Jaeger:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:

exporters:
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/jaeger]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/jaeger]
```

## Full production Docker Compose

This example combines all backends into a single deployment:

```yaml
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
    environment:
      CLOUDMOCK_DATAPLANE_MODE: production
      CLOUDMOCK_DUCKDB_PATH: /data/cloudmock.duckdb
      CLOUDMOCK_POSTGRESQL_URL: postgres://cloudmock:secret@postgres:5432/cloudmock
      CLOUDMOCK_PROMETHEUS_URL: http://prometheus:9090
      CLOUDMOCK_OTEL_ENDPOINT: otel-collector:4317
      CLOUDMOCK_PROFILE: full
      CLOUDMOCK_IAM_MODE: enforce
      CLOUDMOCK_LOG_LEVEL: info
      CLOUDMOCK_LOG_FORMAT: json
    volumes:
      - cloudmock-data:/data
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: cloudmock
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: cloudmock
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cloudmock"]
      interval: 5s
      timeout: 3s
      retries: 5

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    volumes:
      - ./otel-config.yml:/etc/otelcol-contrib/config.yaml

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
    environment:
      COLLECTOR_OTLP_ENABLED: "true"

volumes:
  cloudmock-data:
  postgres-data:
  prometheus-data:
```

## Environment variable reference

All production mode variables, in one place:

| Variable | Description | Default |
|----------|-------------|---------|
| `CLOUDMOCK_DATAPLANE_MODE` | `local` or `production` | `local` |
| `CLOUDMOCK_DUCKDB_PATH` | Path to the DuckDB file | `cloudmock.duckdb` |
| `CLOUDMOCK_POSTGRESQL_URL` | PostgreSQL connection string | -- (falls back to in-memory) |
| `CLOUDMOCK_PROMETHEUS_URL` | Prometheus base URL for metric queries | -- (metrics computed from traces) |
| `CLOUDMOCK_OTEL_ENDPOINT` | OTel Collector gRPC endpoint (host:port) | -- (telemetry stays local) |

All backends are optional. If a backend is not configured, CloudMock falls back to in-memory storage for that category of data. You can enable production mode with only DuckDB, only PostgreSQL, or any combination.

## Choosing a deployment pattern

| Scenario | Mode | Backends |
|----------|------|----------|
| Local development | `local` | None |
| CI pipeline | `local` | None |
| Persistent dev environment | `production` | DuckDB |
| Team staging environment | `production` | DuckDB + PostgreSQL |
| Production observability | `production` | DuckDB + PostgreSQL + Prometheus + OTel |
