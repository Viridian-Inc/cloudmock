# Full-Stack App

React-free notes app — vanilla HTML/JS frontend served by nginx, Express API backed by DynamoDB and S3.

## Services

| Service | Port | Description |
|---------|------|-------------|
| CloudMock | 4566 | AWS API emulation |
| CloudMock DevTools | 4500 | Observability dashboard |
| API | 3000 | Express REST API (DynamoDB + S3) |
| Frontend | 8080 | nginx serving static HTML |

## Start

```bash
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080) — create and view notes, all persisted in DynamoDB.

## API endpoints

```bash
# List all notes
curl http://localhost:3000/notes

# Create a note
curl -X POST http://localhost:3000/notes \
  -H "Content-Type: application/json" \
  -d '{"title": "Hello", "body": "First note"}'

# Upload an attachment (stored in S3)
curl -X POST http://localhost:3000/upload \
  -H "Content-Type: application/json" \
  -d '{"filename": "report.txt", "content": "hello world"}'
```

## Replacing the frontend

The frontend is a single `index.html` with no build step. To use a proper React/Vue/Svelte app:

1. Replace `frontend/index.html` with your static build output
2. Or mount a build directory: `volumes: ["./frontend/dist:/usr/share/nginx/html"]`
3. Add a build step to the `frontend` service if needed
