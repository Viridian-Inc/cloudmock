# Webapp + Postgres Stack

Hybrid stack: relational data in Postgres, files in S3, background jobs in SQS.

## Services

| Service | Port | Description |
|---------|------|-------------|
| CloudMock | 4566 | AWS API emulation |
| CloudMock DevTools | 4500 | Observability dashboard |
| Postgres | 5432 | Relational user data |
| API | 3000 | Express REST API |

## Start

```bash
docker compose up --build
```

## Endpoints

```bash
# Health check (confirms Postgres connection)
curl http://localhost:3000/health

# Create a user (stored in Postgres)
curl -X POST http://localhost:3000/users \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "name": "Alice"}'
# Returns: {"id": "...", "email": "...", "name": "...", "created_at": "..."}

# Fetch a user
curl http://localhost:3000/users/<id>

# Upload a file for a user (stored in S3)
curl -X POST http://localhost:3000/users/<id>/files \
  -H "Content-Type: application/json" \
  -d '{"report": "Q1", "data": [1, 2, 3]}'

# Queue a background job (sent to SQS)
curl -X POST http://localhost:3000/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "generate-report", "userId": "<id>"}'
```

## When to use this pattern

- You need structured queries, JOINs, or transactions — use Postgres
- You need to store large files or blobs — use S3
- You need to offload slow work (emails, reports, exports) — use SQS
- This combination covers ~80% of web application needs
