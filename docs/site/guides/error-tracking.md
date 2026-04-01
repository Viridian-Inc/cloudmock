# Error Tracking

CloudMock captures, groups, and tracks errors from both backend services and browser applications. Errors are first-class entities with stack traces, occurrence counts, and trend analysis.

## How Errors Are Captured

Errors reach CloudMock through three paths:

1. **AWS service errors** -- captured automatically when your AWS SDK calls fail (gateway)
2. **OpenTelemetry spans** -- spans with `ERROR` status are extracted as errors
3. **Direct ingestion** -- send errors via the `/api/errors/ingest` endpoint or the RUM SDK

## Backend Error Ingestion

Send errors from your application code via the Admin API:

```bash
curl -X POST http://localhost:4599/api/errors/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Cannot read property 'id' of undefined",
    "type": "TypeError",
    "stack": "TypeError: Cannot read property '\''id'\'' of undefined\n    at processOrder (/app/src/orders.js:42:18)\n    at handler (/app/src/handler.js:15:5)",
    "service": "order-service",
    "environment": "development",
    "release": "1.2.3",
    "tags": {
      "user_id": "user-123",
      "order_id": "order-456"
    }
  }'
```

## Browser Error Capture with RUM SDK

The `@cloudmock/rum` SDK captures browser errors automatically:

```js
import { CloudMockRUM } from '@cloudmock/rum';

CloudMockRUM.init({
  endpoint: 'http://localhost:4599',
  serviceName: 'my-web-app',
  sampleRate: 1.0,
});
```

This captures:
- Unhandled exceptions (`window.onerror`)
- Unhandled promise rejections
- Console errors
- Network errors (failed fetch/XHR)

## Error Grouping

CloudMock groups identical errors by fingerprint (hash of error type + message + top stack frames). Each group tracks:

| Field | Description |
|-------|-------------|
| `id` | Unique error group ID |
| `message` | Error message |
| `type` | Error type (TypeError, ValueError, etc.) |
| `count` | Total occurrences |
| `first_seen` | Timestamp of first occurrence |
| `last_seen` | Timestamp of most recent occurrence |
| `service` | Originating service name |
| `stack` | Full stack trace |
| `tags` | Custom metadata |

## Viewing Errors

### DevTools Dashboard

Open `http://localhost:4500` and navigate to the Errors tab. You'll see:

- Error inbox with all groups sorted by frequency
- Trend sparklines showing error rate over time
- Filters by service, error type, and time range

Click any error group to see:
- Full stack trace
- All occurrences with timestamps
- Request context (if captured via OTel or gateway)
- Related traces

### Admin API

```bash
# List all error groups
curl http://localhost:4599/api/errors | jq '.'

# Get a specific error group by ID
curl http://localhost:4599/api/errors/err_abc123 | jq '.'
```

Response:

```json
{
  "id": "err_abc123",
  "message": "Cannot read property 'id' of undefined",
  "type": "TypeError",
  "count": 47,
  "first_seen": "2026-03-31T10:00:00Z",
  "last_seen": "2026-03-31T14:23:00Z",
  "service": "order-service",
  "stack": "TypeError: Cannot read property...",
  "tags": {"release": "1.2.3"}
}
```

## Errors from OpenTelemetry

When an OTel span records an exception, CloudMock extracts it as an error:

**Node.js:**
```js
const span = tracer.startSpan('process-order');
try {
  await riskyOperation();
} catch (err) {
  span.recordException(err);          // This becomes a CloudMock error
  span.setStatus({ code: 2 });        // SpanStatusCode.ERROR
  throw err;
} finally {
  span.end();
}
```

**Python:**
```python
with tracer.start_as_current_span("process-order") as span:
    try:
        risky_operation()
    except Exception as e:
        span.record_exception(e)
        span.set_status(StatusCode.ERROR, str(e))
        raise
```

**Go:**
```go
ctx, span := tracer.Start(ctx, "process-order")
defer span.End()

if err := riskyOperation(ctx); err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}
```

## Linking Errors to Traces

When errors come from OTel spans, they're automatically linked to the parent trace. Click "View Trace" on any error to see the full distributed trace, including:

- Which service called which
- Where the error originated
- How it propagated through the call chain

## Alerting on Errors

Set up monitors to alert when error rates spike:

```bash
curl -X POST http://localhost:4599/api/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High error rate - order-service",
    "type": "threshold",
    "query": "errors.count",
    "filters": {"service": "order-service"},
    "threshold": 10,
    "window": "5m",
    "severity": "critical"
  }'
```

See the [Alerting guide](alerting.md) for webhook and Slack integration.

## Source Maps

For minified JavaScript errors, upload source maps to get readable stack traces:

```bash
curl -X POST http://localhost:4599/api/sourcemaps \
  -F "file=@dist/app.js.map" \
  -F "release=1.2.3" \
  -F "url=http://localhost:3000/app.js"
```

The RUM SDK includes the release version in error reports, and CloudMock uses the source map to de-minify stack traces.
