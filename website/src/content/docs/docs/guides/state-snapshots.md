---
title: State Snapshots
description: Save and restore CloudMock state across restarts
---

# State Snapshots

Save CloudMock's state to a JSON file and restore it on startup. Teams can commit a `cloudmock-state.json` to version control so everyone starts with the same pre-configured resources.

## Quick Start

### Export state

```bash
# Create some resources
aws --endpoint http://localhost:4566 s3 mb s3://my-bucket
aws --endpoint http://localhost:4566 dynamodb create-table \
  --table-name users \
  --key-schema AttributeName=pk,KeyType=HASH \
  --attribute-definitions AttributeName=pk,AttributeType=S \
  --billing-mode PAY_PER_REQUEST

# Export to file
curl -X POST http://localhost:4599/api/state/export > cloudmock-state.json
```

### Restore state on startup

```bash
cloudmock --state cloudmock-state.json
```

All resources are instantly available — no need to re-create them.

### Auto-save on shutdown

```bash
cloudmock --state cloudmock-state.json --persist
```

State is automatically saved when CloudMock receives SIGTERM (Ctrl+C). Next startup loads the saved state.

## State File Format

```json
{
  "version": 1,
  "exported_at": "2026-04-02T20:00:00Z",
  "services": {
    "s3": {
      "buckets": [
        {
          "name": "my-bucket",
          "objects": [
            {
              "key": "config.json",
              "body_base64": "eyJrZXkiOiJ2YWx1ZSJ9",
              "content_type": "application/json"
            }
          ]
        }
      ]
    },
    "dynamodb": {
      "tables": [
        {
          "name": "users",
          "key_schema": [{ "AttributeName": "pk", "KeyType": "HASH" }],
          "attribute_definitions": [{ "AttributeName": "pk", "AttributeType": "S" }],
          "billing_mode": "PAY_PER_REQUEST",
          "items": [
            { "pk": { "S": "user-1" }, "name": { "S": "Alice" } }
          ]
        }
      ]
    }
  }
}
```

## Supported Services

| Service | What's saved | What's not saved |
|---------|-------------|-----------------|
| S3 | Buckets, objects (key + body + content type) | Multipart uploads in progress |
| DynamoDB | Tables (schema, GSIs, LSIs), items | Streams, TTL config |
| SQS | Queues, attributes | In-flight messages, DLQ config |
| SNS | Topics, subscriptions | Pending confirmations |
| Lambda | Function configs (name, runtime, handler, env) | Invocation logs |
| IAM | Users, roles, policies, attachments | Access keys |
| CloudWatch Logs | Log groups | Log events |
| Route53 | Hosted zones | Records |

## Admin API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/state/export` | POST | Export current state as JSON |
| `/api/state/import` | POST | Import state from JSON body |
| `/api/state/reset` | POST | Clear all service state |

## CI Usage

Commit `cloudmock-state.json` to your repo and load it in CI to skip setup boilerplate:

```yaml
# GitHub Actions
- run: npx cloudmock --state cloudmock-state.json &
- run: sleep 2
- run: npm test
```

## Docker

```bash
docker run -v ./cloudmock-state.json:/state.json \
  -p 4566:4566 ghcr.io/viridian-inc/cloudmock:latest \
  --state /state.json
```

## Go SDK (In-Process)

```go
cm := sdk.New(sdk.WithStateFile("cloudmock-state.json"))
defer cm.Close()
// All resources from the state file are available immediately
```
