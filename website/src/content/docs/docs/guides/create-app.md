---
title: create-cloudmock-app
description: Scaffold a complete project with CloudMock pre-configured for your stack.
---

`create-cloudmock-app` generates a fully working project with CloudMock wired up — tests, CI, docker-compose, and all. No boilerplate required.

## Quick Start

```bash
npx create-cloudmock-app my-app
```

The CLI prompts you to choose a language, AWS services, and test framework, then generates a complete project.

## CLI Options

Pass flags to skip interactive prompts:

```bash
npx create-cloudmock-app my-app --lang node --services dynamodb --test jest
npx create-cloudmock-app my-app --lang python --services s3 --test pytest
npx create-cloudmock-app my-app --lang go --services sqs
```

| Flag | Values |
|------|--------|
| `--lang` | `node`, `python`, `go`, `java`, `rust` |
| `--services` | `s3`, `dynamodb`, `sqs`, `sns`, `lambda` (comma-separated) |
| `--test` | `jest`, `vitest`, `pytest`, `junit5` |

## Templates

### node-dynamodb

Express REST API with full DynamoDB CRUD.

```bash
npx create-cloudmock-app my-app --lang node --services dynamodb
cd my-app && npm install && npm start
```

Endpoints: `POST /items`, `GET /items`, `GET /items/:id`, `DELETE /items/:id`

Tests use `@cloudmock/sdk` — no Docker required for `npm test`.

---

### node-s3

Express file upload/download server backed by S3.

```bash
npx create-cloudmock-app my-app --lang node --services s3
cd my-app && npm install && npm start
```

Endpoints: `POST /upload?key=name`, `GET /files`, `GET /files/:key`

---

### node-sqs

Express producer API + SQS worker consumer.

```bash
npx create-cloudmock-app my-app --lang node --services sqs
cd my-app && npm install
npm start          # HTTP API (terminal 1)
npm run worker     # SQS consumer (terminal 2)
```

Endpoints: `POST /messages`, `GET /messages`

---

### python-dynamodb

FastAPI app with full DynamoDB CRUD using boto3.

```bash
npx create-cloudmock-app my-app --lang python --services dynamodb
cd my-app && pip install -r requirements.txt
uvicorn app:app --reload
```

Interactive docs at http://localhost:8000/docs. Tests via `pytest`.

---

### python-s3

FastAPI app with S3 file upload and download using boto3.

```bash
npx create-cloudmock-app my-app --lang python --services s3
cd my-app && pip install -r requirements.txt
uvicorn app:app --reload
```

Endpoints: `POST /upload?key=name`, `GET /files`, `GET /files/{key}`

---

### go-dynamodb

Go `net/http` server with DynamoDB CRUD using aws-sdk-go-v2.

```bash
npx create-cloudmock-app my-app --lang go --services dynamodb
cd my-app && go mod tidy && go run .
```

Tests use `sdk.New()` for in-process CloudMock (zero network, ~20μs/op):

```bash
go test ./...
```

---

### go-sqs

Go HTTP server with SQS producer/consumer using aws-sdk-go-v2.

```bash
npx create-cloudmock-app my-app --lang go --services sqs
cd my-app && go mod tidy && go run .
```

---

### java-dynamodb

Spring Boot REST controller with DynamoDB.

```bash
npx create-cloudmock-app my-app --lang java --services dynamodb
cd my-app && mvn spring-boot:run
```

Tests use `CloudMock.start()` via JUnit 5:

```bash
mvn test
```

---

### rust-dynamodb

Axum HTTP server with DynamoDB using the official AWS SDK for Rust.

```bash
npx create-cloudmock-app my-app --lang rust --services dynamodb
cd my-app && cargo run
```

Integration tests use the `cloudmock` crate:

```bash
cargo test
```

---

## What Each Template Includes

Every generated project contains:

- **Application code** — a working server with real AWS SDK calls
- **Tests** — full test suite using CloudMock (no Docker needed for tests)
- **docker-compose.yml** — CloudMock container for local development
- **.github/workflows/ci.yml** — CI pipeline using `cloudmock-action`
- **README.md** — getting-started instructions

## Local Development

All templates include a `docker-compose.yml` for running CloudMock locally:

```bash
docker compose up
```

Or use the npm package:

```bash
npx cloudmock
```

Then point your SDK at `http://localhost:4566`.

## CloudMock DevTools

With CloudMock running, open [http://localhost:4500](http://localhost:4500) to explore:

- Distributed trace topology for your service
- Per-service request metrics and latency
- Error tracking and chaos engineering controls
