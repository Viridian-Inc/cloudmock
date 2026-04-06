# Go Todo Demo

## Quickstart

```bash
# Start CloudMock (in another terminal)
npx cloudmock

# Run the demo
go run quickstart.go
```

## API Server

```bash
go run server/main.go
# Server runs on http://localhost:3000
```

### Try it

```bash
# Create a todo
curl -X POST http://localhost:3000/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy groceries", "description": "Milk, eggs, bread"}'

# Upload an image
curl -X PUT http://localhost:3000/todos/<id>/image \
  -H "Content-Type: image/jpeg" \
  --data-binary @../sample.jpg

# Complete a todo
curl -X PUT http://localhost:3000/todos/<id>/complete

# List todos
curl http://localhost:3000/todos

# Check notifications
curl http://localhost:3000/notifications
```
