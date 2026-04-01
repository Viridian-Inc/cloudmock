# Log Management

CloudMock provides a unified log viewer that collects application logs, Lambda execution logs, and CloudWatch Logs in one searchable timeline.

## How Logs Are Collected

Logs reach CloudMock through multiple paths:

1. **CloudWatch Logs service** -- Lambda functions and services that write to CloudWatch Logs
2. **OpenTelemetry logs** -- OTLP log signals sent to port 4318
3. **Direct ingestion** -- POST to `/api/logs/ingest`
4. **Source SDK capture** -- `@cloudmock/node` auto-captures `console.log/warn/error`

## Viewing Logs

### DevTools Dashboard

Open `http://localhost:4500` and navigate to the Logs tab. Features:

- **Live tail** -- logs stream in real time via SSE
- **Full-text search** -- search across all log sources
- **Level filtering** -- info, warn, error, debug
- **Service filtering** -- filter by originating service
- **Time range** -- select a time window
- **Trace correlation** -- click a log entry to see the related trace

### Admin API

```bash
# List recent logs
curl "http://localhost:4599/api/logs?limit=50" | jq '.'

# Filter by service and level
curl "http://localhost:4599/api/logs?service=order-service&level=error&limit=20" | jq '.'

# Full-text search
curl "http://localhost:4599/api/logs?q=payment+failed&limit=20" | jq '.'

# Stream logs (Server-Sent Events)
curl -N http://localhost:4599/api/logs/stream

# List available services
curl http://localhost:4599/api/logs/services | jq '.'

# List available log levels
curl http://localhost:4599/api/logs/levels | jq '.'
```

Response:

```json
[
  {
    "timestamp": "2026-03-31T14:23:01.234Z",
    "level": "error",
    "message": "Payment processing failed: insufficient funds",
    "service": "order-service",
    "trace_id": "abc123def456",
    "span_id": "789ghi",
    "attributes": {
      "order_id": "order-456",
      "amount": 99.99
    }
  }
]
```

## Sending Logs via OTLP

Any OpenTelemetry log SDK can send logs to CloudMock:

**Node.js:**

```js
const { logs } = require('@opentelemetry/api-logs');
const { LoggerProvider, SimpleLogRecordProcessor } = require('@opentelemetry/sdk-logs');
const { OTLPLogExporter } = require('@opentelemetry/exporter-logs-otlp-http');

const loggerProvider = new LoggerProvider();
loggerProvider.addLogRecordProcessor(
  new SimpleLogRecordProcessor(
    new OTLPLogExporter({ url: 'http://localhost:4318/v1/logs' })
  )
);

const logger = loggerProvider.getLogger('my-service');
logger.emit({
  severityText: 'ERROR',
  body: 'Payment processing failed',
  attributes: { 'order.id': 'order-456' },
});
```

**Python:**

```python
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
import logging

logger_provider = LoggerProvider()
logger_provider.add_log_record_processor(
    BatchLogRecordProcessor(OTLPLogExporter(endpoint="http://localhost:4318/v1/logs"))
)

handler = LoggingHandler(logger_provider=logger_provider)
logging.getLogger().addHandler(handler)

# Now standard logging calls are exported to CloudMock
logging.error("Payment processing failed", extra={"order_id": "order-456"})
```

## Direct Ingestion

Send logs directly to the CloudMock API:

```bash
curl -X POST http://localhost:4599/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "timestamp": "2026-03-31T14:23:01.234Z",
    "level": "error",
    "message": "Payment processing failed: insufficient funds",
    "service": "order-service",
    "trace_id": "abc123def456",
    "attributes": {
      "order_id": "order-456",
      "amount": 99.99,
      "currency": "USD"
    }
  }'
```

## Batch Ingestion

Send multiple log entries at once:

```bash
curl -X POST http://localhost:4599/api/logs/ingest \
  -H "Content-Type: application/json" \
  -d '[
    {"level": "info", "message": "Order received", "service": "api-gateway", "trace_id": "abc123"},
    {"level": "info", "message": "Processing payment", "service": "payment-service", "trace_id": "abc123"},
    {"level": "error", "message": "Payment declined", "service": "payment-service", "trace_id": "abc123"}
  ]'
```

## Log-to-Trace Correlation

When logs include a `trace_id`, CloudMock links them to the corresponding distributed trace. In DevTools:

1. Click any log entry with a trace ID
2. "View Trace" takes you to the full trace timeline
3. The trace view shows log entries inline alongside spans

To include trace context in your logs automatically, use OpenTelemetry's log bridge or include the trace ID manually:

```js
const { trace } = require('@opentelemetry/api');

function log(level, message, extra = {}) {
  const span = trace.getActiveSpan();
  const traceId = span?.spanContext().traceId || '';
  const spanId = span?.spanContext().spanId || '';

  // Send to CloudMock with trace context
  fetch('http://localhost:4599/api/logs/ingest', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ level, message, trace_id: traceId, span_id: spanId, ...extra }),
  });
}
```

## Lambda Execution Logs

Lambda functions running in CloudMock automatically have their execution logs captured. View them:

```bash
# All Lambda logs
curl http://localhost:4599/api/lambda/logs | jq '.'

# Stream Lambda logs in real time
curl -N http://localhost:4599/api/lambda/logs/stream
```

These logs include:
- Function start/end markers
- `console.log` / `print` output
- Runtime errors and timeouts
- Memory usage and duration

## Log Levels

CloudMock normalizes log levels across sources:

| Level | Sources |
|-------|---------|
| `debug` | OTel TRACE/DEBUG, console.debug |
| `info` | OTel INFO, console.log, console.info |
| `warn` | OTel WARN, console.warn |
| `error` | OTel ERROR/FATAL, console.error, uncaught exceptions |
