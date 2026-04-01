# Docker Deployment

## Quick Start

```bash
docker run -d \
  --name cloudmock \
  -p 4566:4566 \
  -p 4500:4500 \
  -p 4599:4599 \
  -p 4318:4318 \
  ghcr.io/neureaux/cloudmock:latest
```

DevTools: http://localhost:4500
Gateway: http://localhost:4566

## Docker Compose

### Basic: CloudMock + Your App

```yaml
# docker-compose.yml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    ports:
      - "4566:4566"   # Gateway (AWS endpoint)
      - "4500:4500"   # DevTools dashboard
      - "4599:4599"   # Admin API
      - "4318:4318"   # OTLP endpoint
    environment:
      CLOUDMOCK_PROFILE: standard
      CLOUDMOCK_LOG_LEVEL: info

  app:
    build: .
    ports:
      - "3000:3000"
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      AWS_DEFAULT_REGION: us-east-1
      OTEL_EXPORTER_OTLP_ENDPOINT: http://cloudmock:4318
      OTEL_SERVICE_NAME: my-app
```

### With Persistence

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
    volumes:
      - cloudmock-data:/data

volumes:
  cloudmock-data:
```

### With Data Seeding

```yaml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
      - "4318:4318"

  seed:
    image: amazon/aws-cli:latest
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      AWS_DEFAULT_REGION: us-east-1
    entrypoint: /bin/sh
    command:
      - -c
      - |
        # Wait for CloudMock to be ready
        until aws s3 ls 2>/dev/null; do sleep 1; done

        # Create resources
        aws s3 mb s3://uploads
        aws dynamodb create-table \
          --table-name users \
          --attribute-definitions AttributeName=userId,AttributeType=S \
          --key-schema AttributeName=userId,KeyType=HASH \
          --billing-mode PAY_PER_REQUEST
        aws sqs create-queue --queue-name orders
        echo "Seeding complete"
```

### Multi-Service Architecture

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
      CLOUDMOCK_PROFILE: full

  api-gateway:
    build: ./services/api-gateway
    ports:
      - "3000:3000"
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      OTEL_EXPORTER_OTLP_ENDPOINT: http://cloudmock:4318
      OTEL_SERVICE_NAME: api-gateway

  order-service:
    build: ./services/order-service
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      OTEL_EXPORTER_OTLP_ENDPOINT: http://cloudmock:4318
      OTEL_SERVICE_NAME: order-service

  payment-service:
    build: ./services/payment-service
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      OTEL_EXPORTER_OTLP_ENDPOINT: http://cloudmock:4318
      OTEL_SERVICE_NAME: payment-service
```

All three services send telemetry to CloudMock, creating cross-service distributed traces visible in DevTools.

### With DuckDB Persistence

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
      CLOUDMOCK_DATAPLANE_MODE: duckdb
      CLOUDMOCK_DUCKDB_PATH: /data/cloudmock.duckdb
    volumes:
      - cloudmock-db:/data

volumes:
  cloudmock-db:
```

### With Custom Config File

```yaml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
      - "4318:4318"
    volumes:
      - ./cloudmock.yml:/etc/cloudmock/cloudmock.yml:ro
    command: ["--config", "/etc/cloudmock/cloudmock.yml"]
```

## Exposed Ports

| Port | Service | Protocol |
|------|---------|----------|
| 4566 | Gateway | HTTP (AWS API) |
| 4500 | Dashboard | HTTP (Web UI) |
| 4599 | Admin API | HTTP (REST) |
| 4318 | OTLP | HTTP (OpenTelemetry) |

## Health Check

```yaml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:4599/api/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 5s
```

Other services can use `depends_on` with a health condition:

```yaml
  app:
    depends_on:
      cloudmock:
        condition: service_healthy
```

## Building the Image

```bash
# From the cloudmock repository root
docker build -t cloudmock .
```

The Dockerfile is a multi-stage build:

1. **Stage 1:** Builds the DevTools dashboard (Preact + Vite)
2. **Stage 2:** Compiles the Go binary with the dashboard embedded
3. **Stage 3:** Final Alpine image (~25MB)

## Environment Variables

Pass any `CLOUDMOCK_*` environment variable to configure the container. See [Environment Variables](../reference/environment-variables.md) for the full list.

Most common:

```yaml
environment:
  CLOUDMOCK_PROFILE: standard       # Service profile
  CLOUDMOCK_REGION: us-east-1       # AWS region
  CLOUDMOCK_LOG_LEVEL: info         # Log verbosity
  CLOUDMOCK_PERSIST: "true"         # Enable persistence
  CLOUDMOCK_PERSIST_PATH: /data     # Persistence directory
  CLOUDMOCK_IAM_MODE: permissive    # IAM enforcement mode
```
