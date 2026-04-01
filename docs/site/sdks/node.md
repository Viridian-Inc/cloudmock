# @cloudmock/node SDK

Optional Node.js SDK that wraps OpenTelemetry and adds CloudMock-specific convenience features. Not required -- you can use standard OTel directly.

## Installation

```bash
npm install @cloudmock/node
```

## Quick Start

```js
// Require before your application code
require('@cloudmock/node').init();

// Or with ES modules
import { init } from '@cloudmock/node';
init();
```

That's it. The SDK:

1. Auto-discovers a running CloudMock instance on default ports
2. Configures OpenTelemetry to export traces and metrics to CloudMock
3. Intercepts `console.log/warn/error` and forwards them as structured logs
4. Adds source correlation headers to outgoing requests

## Configuration

```js
import { init } from '@cloudmock/node';

init({
  // CloudMock endpoint (auto-discovered if not set)
  endpoint: 'http://localhost:4599',

  // OTLP endpoint for traces/metrics (auto-discovered if not set)
  otlpEndpoint: 'http://localhost:4318',

  // Service name shown in DevTools
  serviceName: 'my-api-service',

  // Enable/disable features
  captureConsole: true,     // Forward console.log/warn/error
  sourceCorrelation: true,  // Add source file:line to requests
  autoInstrument: true,     // Enable OTel auto-instrumentation

  // Sampling
  tracesSampleRate: 1.0,    // 1.0 = sample everything
});
```

## Features

### Auto-Discovery

The SDK automatically finds CloudMock by checking:

1. `CLOUDMOCK_ADMIN_URL` environment variable
2. `http://localhost:4599/api/health` (default admin port)
3. `http://localhost:4566` (default gateway port)

If CloudMock is not running, the SDK silently disables itself -- no errors, no performance impact.

### Console Capture

With `captureConsole: true` (default), the SDK intercepts console methods:

```js
console.log('Order received', { orderId: 'order-123' });
// → Forwarded to CloudMock as a structured log entry:
// {level: "info", message: "Order received", attributes: {orderId: "order-123"}}

console.error('Payment failed', new Error('insufficient funds'));
// → Forwarded as error-level log with stack trace
```

Logs appear in the DevTools Logs tab with:
- Timestamp
- Log level (info for log, warn for warn, error for error)
- Message
- Structured attributes
- Source file and line number
- Trace ID (if within an active span)

### Source Correlation

With `sourceCorrelation: true` (default), outgoing HTTP requests include an `x-cloudmock-source` header:

```
x-cloudmock-source: /app/src/handlers/orders.js:42
```

This tells DevTools which source file and line initiated each request, enabling "click to source" in the request inspector.

### Auto-Instrumentation

The SDK configures OpenTelemetry auto-instrumentation for common Node.js libraries:

- **HTTP** -- `http`, `https`, `fetch` (Node 18+)
- **Express** -- route handlers, middleware timing
- **Fastify** -- request handlers
- **AWS SDK** -- all AWS service calls
- **Database** -- `pg`, `mysql2`, `mongodb`, `redis`, `ioredis`
- **gRPC** -- client and server calls

### Manual Spans

Create custom spans using the standard OpenTelemetry API:

```js
import { trace } from '@opentelemetry/api';

const tracer = trace.getTracer('my-service');

async function processOrder(order) {
  return tracer.startActiveSpan('process-order', async (span) => {
    span.setAttribute('order.id', order.id);
    span.setAttribute('order.total', order.total);

    try {
      const payment = await chargePayment(order);
      span.setAttribute('payment.id', payment.id);

      await sendConfirmation(order);
      return payment;
    } catch (err) {
      span.recordException(err);
      span.setStatus({ code: 2, message: err.message });
      throw err;
    } finally {
      span.end();
    }
  });
}
```

## Environment Variables

The SDK respects standard OpenTelemetry environment variables plus CloudMock-specific ones:

| Variable | Description |
|----------|-------------|
| `CLOUDMOCK_ADMIN_URL` | CloudMock admin API URL |
| `CLOUDMOCK_ENABLED` | Set to `false` to disable the SDK |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint (overrides auto-discovery) |
| `OTEL_SERVICE_NAME` | Service name for traces |

## Usage with Express

```js
// app.js
import '@cloudmock/node/register';  // One-line setup
import express from 'express';

const app = express();

app.get('/orders/:id', async (req, res) => {
  // This request is automatically traced
  const order = await getOrder(req.params.id);
  res.json(order);
});

app.listen(3000);
```

## Usage with AWS SDK

```js
import '@cloudmock/node/register';
import { DynamoDBClient, PutItemCommand } from '@aws-sdk/client-dynamodb';

// AWS SDK calls are automatically traced and visible in DevTools
const client = new DynamoDBClient({});
await client.send(new PutItemCommand({
  TableName: 'orders',
  Item: { orderId: { S: 'order-123' }, status: { S: 'pending' } },
}));
```

The trace shows both the HTTP request to CloudMock and the DynamoDB operation as linked spans.

## Disabling in Production

The SDK auto-detects whether CloudMock is running. If it's not, the SDK is a no-op. You can also explicitly disable it:

```js
import { init } from '@cloudmock/node';

init({
  enabled: process.env.NODE_ENV !== 'production',
});
```

Or via environment variable:

```bash
CLOUDMOCK_ENABLED=false node app.js
```
