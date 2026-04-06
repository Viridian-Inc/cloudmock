# CloudMock Todo Demo

A todo app with image attachments and push notifications, powered by [CloudMock](https://cloudmock.dev) — local AWS emulation.

Demonstrates **S3**, **DynamoDB**, **SQS**, and **SNS** in three languages.

## Prerequisites

- [CloudMock](https://cloudmock.dev/docs/getting-started/installation/) (or Docker)
- One of: Node.js 18+, Python 3.10+, Go 1.21+

## Quick Start

**Option A — npx (fastest):**

```bash
npx cloudmock
```

**Option B — Docker:**

```bash
docker compose up -d
```

Then pick your language:

| Language | Quickstart | API Server |
|----------|-----------|------------|
| **Node.js** | `cd node && npm install && npx tsx quickstart.ts` | `cd node && npm install && npx tsx server/index.ts` |
| **Python** | `cd python && pip install -r requirements.txt && python quickstart.py` | `cd python && pip install -r requirements.txt && python server/app.py` |
| **Go** | `cd go && go run quickstart.go` | `cd go && go run server/main.go` |

Open **http://localhost:3000** for the web UI, or **http://localhost:4500** for CloudMock DevTools.

## What It Does

1. **Create a todo** — stores it in DynamoDB
2. **Attach an image** — uploads to S3, links to the todo
3. **Complete a todo** — publishes a notification via SNS
4. **Receive notifications** — polls SQS (subscribed to the SNS topic)

## API Endpoints (server mode)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/todos` | Create a todo (`title`, `description` in JSON body) |
| `GET` | `/todos` | List all todos |
| `GET` | `/todos/:id` | Get a single todo |
| `PUT` | `/todos/:id/image` | Upload image (binary body, `Content-Type: image/*`) |
| `PUT` | `/todos/:id/complete` | Mark as complete, sends SNS notification |
| `GET` | `/notifications` | Poll SQS for completion notifications |

## Learn More

- [CloudMock Docs](https://cloudmock.dev/docs/)
- [GitHub](https://github.com/Viridian-Inc/cloudmock)
