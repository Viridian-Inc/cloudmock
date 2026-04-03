---
title: Docker Compose Stacks
description: Ready-to-run Docker Compose templates for common architectures with CloudMock.
---

Eight production-pattern stacks you can clone and adapt. Each stack wires CloudMock to real application code so you can see the full integration in minutes.

## Stacks overview

| Stack | Services | Best for |
|-------|----------|----------|
| [minimal](#minimal) | CloudMock | Exploring the API, one-off testing |
| [serverless](#serverless) | CloudMock + Node API | REST APIs backed by DynamoDB + SQS |
| [microservices](#microservices) | CloudMock + 3 services (Node, Python, Go) | SNS fan-out, polyglot architectures |
| [data-pipeline](#data-pipeline) | CloudMock + uploader + worker | S3 ingest → SQS → DynamoDB |
| [webapp-postgres](#webapp-postgres) | CloudMock + Node API + Postgres | Hybrid: relational DB + AWS services |
| [fullstack](#fullstack) | CloudMock + Node API + nginx | Full-stack apps with a frontend |
| [terraform](#terraform) | CloudMock (Terraform runs on host) | IaC validation before deploying to AWS |
| [monitoring](#monitoring) | CloudMock + Prometheus + Grafana | Observability, metrics dashboards |

All stacks live in `docker/stacks/` in the CloudMock repo.

---

## minimal

Just CloudMock. The fastest way to start.

```bash
cd docker/stacks/minimal
docker compose up
```

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

aws s3 mb s3://my-bucket
aws dynamodb list-tables
```

Open DevTools at [http://localhost:4500](http://localhost:4500).

---

## serverless

Express API with DynamoDB storage and SQS job queue.

```bash
cd docker/stacks/serverless
docker compose up --build
```

```bash
# Create an item
curl -X POST http://localhost:3000/items \
  -H "Content-Type: application/json" \
  -d '{"id": "item-1", "data": {"name": "widget"}}'

# Fetch it back
curl http://localhost:3000/items/item-1

# Queue a job
curl -X POST http://localhost:3000/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "send-email", "to": "user@example.com"}'
```

The `setup` container auto-creates the DynamoDB table and SQS queue before the API starts.

---

## microservices

Three services in three languages communicating via SNS → SQS fan-out.

```bash
cd docker/stacks/microservices
docker compose up --build
```

```bash
# Create an order — triggers payment + notification pipeline
curl -X POST http://localhost:3001/orders \
  -H "Content-Type: application/json" \
  -d '{"customerId": "cust-1", "amount": 49.99}'

# Check payment status
curl http://localhost:3002/payments/<order-id>
```

**Architecture**: `order-service` (Node.js, port 3001) → SNS → two SQS queues → `payment-service` (Python/FastAPI, port 3002) + `notification-service` (Go, port 3003).

---

## data-pipeline

File ingestion pipeline: S3 upload → SQS notification → worker → DynamoDB.

```bash
cd docker/stacks/data-pipeline
docker compose up --build
```

The `uploader` generates 5 sample records, uploads them to S3, and sends SQS notifications. The `worker` picks them up, transforms the data, and writes results to DynamoDB.

```bash
# View processed results
aws dynamodb scan --table-name processed-records \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 --no-sign-request
```

---

## webapp-postgres

Hybrid stack: structured data in Postgres, files in S3, async work in SQS.

```bash
cd docker/stacks/webapp-postgres
docker compose up --build
```

```bash
# Create a user (Postgres)
curl -X POST http://localhost:3000/users \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "name": "Alice"}'

# Upload a file (S3)
curl -X POST http://localhost:3000/users/<id>/files \
  -H "Content-Type: application/json" \
  -d '{"report": "Q1", "data": [1, 2, 3]}'

# Queue a job (SQS)
curl -X POST http://localhost:3000/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "generate-report", "userId": "<id>"}'
```

---

## fullstack

Notes app: vanilla HTML frontend (no build step) + Express API backed by DynamoDB.

```bash
cd docker/stacks/fullstack
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080) — create and browse notes in the browser. The API runs at port 3000.

To replace the frontend with a React/Vue/Svelte app, swap out `frontend/index.html` for your static build output and adjust the nginx volume mount.

---

## terraform

Validate Terraform configs against CloudMock before deploying to real AWS.

```bash
# Start CloudMock
cd docker/stacks/terraform
docker compose up -d

# Run Terraform on your host
cd infra
terraform init
terraform apply -auto-approve
```

The `infra/provider.tf` points all AWS provider endpoints at `localhost:4566`. The `infra/main.tf` provisions an S3 bucket, DynamoDB table, and SQS queue with a dead-letter queue.

```bash
terraform destroy -auto-approve
docker compose down
```

---

## monitoring

Prometheus scrapes CloudMock's admin metrics API; Grafana visualizes them.

```bash
cd docker/stacks/monitoring
docker compose up
```

| URL | Service |
|-----|---------|
| http://localhost:4500 | CloudMock DevTools |
| http://localhost:9090 | Prometheus |
| http://localhost:3000 | Grafana (admin / admin) |

In Grafana, add a Prometheus data source pointing at `http://prometheus:9090`, then build dashboards using metrics like `cloudmock_requests_total` and `cloudmock_request_duration_seconds`.

---

## How to customize

### Add your own service

Add a new entry under `services:` in any `docker-compose.yml`:

```yaml
services:
  my-worker:
    build: ./my-worker
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
    depends_on:
      setup:
        condition: service_completed_successfully
```

### Add more AWS resources

Extend the `setup` service entrypoint to create additional tables, queues, buckets, or topics:

```yaml
setup:
  entrypoint: >
    /bin/sh -c "
      aws dynamodb create-table --table-name my-table ...
      aws sqs create-queue --queue-name my-queue
      aws s3 mb s3://my-bucket
      aws sns create-topic --name my-topic
    "
```

### Use a state snapshot

Pre-populate CloudMock with a saved state instead of running setup commands:

```yaml
cloudmock:
  image: ghcr.io/viridian-inc/cloudmock:latest
  volumes:
    - ./seed-state.json:/cloudmock/state.json:ro
  environment:
    CLOUDMOCK_STATE_FILE: /cloudmock/state.json
```

See the [state snapshots guide](/docs/guides/state-snapshots) for details.

### Point a real app at CloudMock

Any app that reads `AWS_ENDPOINT_URL` from the environment will automatically use CloudMock:

```yaml
my-app:
  image: my-app:latest
  environment:
    AWS_ENDPOINT_URL: http://cloudmock:4566
    AWS_ACCESS_KEY_ID: test
    AWS_SECRET_ACCESS_KEY: test
    AWS_DEFAULT_REGION: us-east-1
```
