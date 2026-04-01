# Production Deployment

CloudMock can be deployed as a shared observability backend for staging and production environments. This guide covers deployment beyond local development.

## Architecture

```
                                    ┌─────────────────┐
  App Instance 1 ──────────────────►│                 │
  App Instance 2 ──────────────────►│   CloudMock     │──► PostgreSQL (traces, metrics)
  App Instance 3 ──────────────────►│   Gateway       │──► Prometheus (metrics export)
                                    │   :4566 :4318   │
                                    └────────┬────────┘
                                             │
                                    ┌────────▼────────┐
                                    │  CloudMock       │
                                    │  Admin + Dashboard│
                                    │  :4599 :4500     │
                                    └──────────────────┘
```

## Prerequisites

For production deployments:

- **PostgreSQL 15+** -- for durable trace, metric, and log storage
- **Reverse proxy** (nginx, Caddy, or cloud LB) -- TLS termination and auth
- Minimum 2 vCPU, 2GB RAM (4GB for `full` profile)

## Docker Compose (Staging)

```yaml
# docker-compose.prod.yml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    restart: unless-stopped
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
      - "4318:4318"
    environment:
      CLOUDMOCK_PROFILE: standard
      CLOUDMOCK_DATAPLANE_MODE: postgresql
      CLOUDMOCK_POSTGRESQL_URL: postgresql://cloudmock:secret@postgres:5432/cloudmock
      CLOUDMOCK_IAM_MODE: permissive
      CLOUDMOCK_LOG_LEVEL: info
      CLOUDMOCK_PERSIST: "true"
      CLOUDMOCK_PERSIST_PATH: /data
    volumes:
      - cloudmock-data:/data
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: cloudmock
      POSTGRES_USER: cloudmock
      POSTGRES_PASSWORD: secret
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cloudmock"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  cloudmock-data:
  postgres-data:
```

## Configuration

### Data Plane

For production, use PostgreSQL instead of in-memory storage:

```yaml
# cloudmock.yml
dataplane:
  mode: postgresql
  postgresql:
    url: postgresql://user:password@host:5432/cloudmock
```

### Authentication

Enable authentication for the Admin API:

```yaml
admin_auth:
  enabled: true
  api_key: your-secret-api-key-here

auth:
  enabled: true
  secret: your-jwt-secret-change-this
```

### Rate Limiting

Protect against abuse:

```yaml
rate_limit:
  enabled: true
  requests_per_second: 1000
  burst: 2000
```

### SLO Configuration

Define SLOs for your services:

```yaml
slo:
  enabled: true
  rules:
    - service: dynamodb
      action: "*"
      p50_ms: 10
      p95_ms: 50
      p99_ms: 100
      error_rate: 0.001
    - service: lambda
      action: Invoke
      p50_ms: 100
      p95_ms: 500
      p99_ms: 1000
      error_rate: 0.01
```

## Reverse Proxy

### Nginx

```nginx
server {
    listen 443 ssl;
    server_name cloudmock.example.com;

    ssl_certificate /etc/ssl/certs/cloudmock.crt;
    ssl_certificate_key /etc/ssl/private/cloudmock.key;

    # Dashboard
    location / {
        proxy_pass http://127.0.0.1:4500;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Admin API
    location /api/ {
        proxy_pass http://127.0.0.1:4599;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # SSE endpoints need special handling
    location ~ ^/api/(stream|logs/stream|lambda/logs/stream) {
        proxy_pass http://127.0.0.1:4599;
        proxy_set_header Host $host;
        proxy_buffering off;
        proxy_cache off;
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        chunked_transfer_encoding off;
    }

    # Gateway (AWS endpoint)
    location /aws/ {
        proxy_pass http://127.0.0.1:4566/;
        proxy_set_header Host $host;
    }
}
```

### Caddy

```
cloudmock.example.com {
    handle /api/* {
        reverse_proxy localhost:4599
    }
    handle {
        reverse_proxy localhost:4500
    }
}

gateway.cloudmock.example.com {
    reverse_proxy localhost:4566
}
```

## Monitoring CloudMock Itself

Export CloudMock's own telemetry to an external system:

```yaml
dataplane:
  otel:
    collector_endpoint: https://otel-collector.example.com:4318
    service_name: cloudmock
  prometheus:
    url: http://prometheus:9090/api/v1/write
```

## Fly.io Deployment

```bash
# fly.toml
app = "cloudmock-staging"
primary_region = "iad"

[build]
  image = "ghcr.io/neureaux/cloudmock:latest"

[env]
  CLOUDMOCK_PROFILE = "standard"
  CLOUDMOCK_DATAPLANE_MODE = "postgresql"
  CLOUDMOCK_PERSIST = "true"

[[services]]
  internal_port = 4566
  protocol = "tcp"
  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

[[services]]
  internal_port = 4500
  protocol = "tcp"
  [[services.ports]]
    port = 8443
    handlers = ["tls", "http"]
```

## Resource Requirements

| Profile | vCPU | RAM | Disk |
|---------|------|-----|------|
| `minimal` | 1 | 512MB | 100MB |
| `standard` | 2 | 1GB | 500MB |
| `full` | 2 | 2GB | 1GB |
| `full` + PostgreSQL | 4 | 4GB | 10GB+ |

## Backups

With PostgreSQL data plane:

```bash
# Backup
pg_dump -U cloudmock cloudmock > cloudmock-backup-$(date +%Y%m%d).sql

# Restore
psql -U cloudmock cloudmock < cloudmock-backup-20260331.sql
```

With DuckDB:

```bash
# Just copy the file
cp cloudmock.duckdb cloudmock-backup-$(date +%Y%m%d).duckdb
```

## Security Checklist

- [ ] Enable `admin_auth` with a strong API key
- [ ] Enable `auth` with a strong JWT secret
- [ ] Use TLS via reverse proxy
- [ ] Restrict network access to gateway port (only your apps should reach it)
- [ ] Change default `root_access_key` and `root_secret_key`
- [ ] Set `iam.mode: enforce` if testing IAM policies
- [ ] Enable rate limiting
- [ ] Use PostgreSQL (not in-memory) for data durability
- [ ] Configure log rotation for `/var/log/cloudmock/`
