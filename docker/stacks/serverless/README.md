# Serverless Stack

Express API backed by DynamoDB and SQS — the classic serverless pattern, running locally.

## Services

| Service | Port | Description |
|---------|------|-------------|
| CloudMock | 4566 | AWS API emulation |
| CloudMock DevTools | 4500 | Request inspector and topology view |
| API | 3000 | Express REST API |

## Start

```bash
docker compose up --build
```

The `setup` container creates the DynamoDB table and SQS queue automatically before the API starts.

## Endpoints

```bash
# Health check
curl http://localhost:3000/health

# Create an item (stored in DynamoDB)
curl -X POST http://localhost:3000/items \
  -H "Content-Type: application/json" \
  -d '{"id": "item-1", "data": {"name": "widget", "price": 9.99}}'

# Fetch an item
curl http://localhost:3000/items/item-1

# Enqueue a background job (sent to SQS)
curl -X POST http://localhost:3000/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "send-email", "to": "user@example.com"}'
```

## Customizing

- **Add more DynamoDB tables**: extend the `setup` service entrypoint
- **Add SQS consumers**: create a new service that polls the `jobs` queue
- **Add more routes**: edit `app/index.js` and rebuild
