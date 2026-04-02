---
title: Source Server & SDKs
description: How the CloudMock devtools SDKs capture and forward telemetry from your application
---

The CloudMock devtools SDKs instrument your application code to capture HTTP traffic, console output, and uncaught errors. Captured events are forwarded to the CloudMock admin API, where they appear in the Activity, Traces, and Topology views alongside the AWS API calls you are already making.

This guide covers how the event ingestion pipeline works, how to set up each SDK, and how to configure transport and correlation.

## How it works

Each SDK follows the same pattern:

1. **Initialize** -- Call `init()` at application startup. The SDK connects to the CloudMock admin API and registers itself as a source.
2. **Intercept** -- The SDK monkey-patches HTTP clients, console/logging functions, and error handlers to capture events as they happen.
3. **Forward** -- Captured events are batched and sent to CloudMock over HTTP (primary) or TCP (fallback). Events are buffered locally if CloudMock is not reachable.
4. **Correlate** -- Outgoing HTTP requests are tagged with an `X-CloudMock-Source` header so CloudMock can attribute traffic to the originating service.

The SDKs are designed to be **silent no-ops** when CloudMock is not running. They add no dependencies on CloudMock at runtime in production.

## Event types

| Event type | Description | Captured by |
|-----------|-------------|-------------|
| `source:register` | Registers the application as a source in the topology | All SDKs |
| `http:inbound` | Inbound HTTP request hitting your server | Express/Connect middleware (Node), WrapHandler (Go), WSGI/ASGI middleware (Python) |
| `http:response` | Outbound HTTP response from a request your app made | HTTP client interceptor |
| `http:error` | Outbound HTTP request that errored (connection refused, timeout) | HTTP client interceptor |
| `console` | Console log message (console.log, logging.info, etc.) | Console/logging interceptor |
| `error` | Uncaught exception or unhandled promise rejection | Error handler |

## Transport

The SDKs use a two-tier transport strategy:

### HTTP transport (primary)

Events are sent as JSON batches via `POST /api/source/events` on the admin API port (default `:4599`). This is the preferred transport because it works through proxies, is easy to debug, and supports standard HTTP error handling.

```
POST http://localhost:4599/api/source/events
Content-Type: application/json

[
  {"type": "http:response", "source": "my-api", "runtime": "node", "timestamp": 1711900800000, "data": {...}},
  {"type": "console", "source": "my-api", "runtime": "node", "timestamp": 1711900800001, "data": {...}}
]
```

The SDK probes `GET /api/source/status` on startup to confirm the HTTP endpoint is available.

### TCP transport (fallback)

If the HTTP probe fails (CloudMock is running but the admin API is unreachable, or a firewall blocks the admin port), the SDK falls back to a raw TCP connection on port **4580**. Events are sent as newline-delimited JSON (one JSON object per line).

The TCP transport reconnects automatically with a 5-second backoff. On reconnect, it re-probes HTTP first before falling back to TCP again.

### Batching and buffering

Events are buffered in memory and flushed either every **1 second** or when the buffer reaches **50 events**, whichever comes first. If neither transport is available, events are kept in a ring buffer (max 500 entries) and flushed when a connection is re-established.

## Correlation headers

When the `correlate` option is enabled (default), the SDK injects two headers into every outgoing HTTP request:

| Header | Value | Purpose |
|--------|-------|---------|
| `X-CloudMock-Source` | Application name (e.g., `my-api`) | Lets CloudMock attribute requests to the originating service in the Topology view |
| `X-CloudMock-Request-Id` | Unique request ID | Links outbound requests to inbound request spans for distributed tracing |

These headers are stripped by CloudMock before forwarding to the emulated AWS service, so they never reach your actual request handlers.

## Node.js SDK

The `@cloudmock/node` SDK captures HTTP traffic (both `http`/`https` modules and `globalThis.fetch`), console output, and uncaught errors.

### Install

```bash
npm install @cloudmock/node
```

### Initialize

Call `init()` before creating any HTTP clients or AWS SDK instances. Wrap it in an environment check so it is tree-shaken in production builds:

```typescript
import { init } from '@cloudmock/node';

if (process.env.NODE_ENV !== 'production') {
  init({ appName: 'my-api' });
}
```

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `host` | `string` | `'localhost'` | CloudMock host |
| `port` | `number` | `4580` | TCP port for fallback transport |
| `appName` | `string` | `process.env.npm_package_name` | Application name in the devtools UI |
| `http` | `boolean` | `true` | Intercept `http.request` / `https.request` |
| `fetch` | `boolean` | `true` | Intercept `globalThis.fetch` (Node 18+) |
| `console` | `boolean` | `true` | Intercept `console.log` / `console.error` |
| `errors` | `boolean` | `true` | Capture uncaught exceptions and unhandled rejections |
| `correlate` | `boolean` | `true` | Inject `X-CloudMock-Source` header into outgoing requests |

### Express middleware

Add the middleware to capture inbound HTTP requests. This records the method, URL, status code, duration, request headers, and the first 4 KB of the response body:

```typescript
import express from 'express';
import { init, getMiddleware } from '@cloudmock/node';

if (process.env.NODE_ENV !== 'production') {
  init({ appName: 'my-api' });
}

const app = express();
app.use(getMiddleware());

app.get('/users', async (req, res) => {
  const result = await dynamodb.send(new ScanCommand({ TableName: 'Users' }));
  res.json(result.Items);
});

app.listen(3000);
```

### Teardown

Call `teardown()` during graceful shutdown to restore all monkey-patched functions and flush remaining events:

```typescript
import { teardown } from '@cloudmock/node';

process.on('SIGTERM', () => {
  teardown();
  process.exit(0);
});
```

## Go SDK

The Go SDK wraps `http.RoundTripper` for outbound traffic and `http.Handler` for inbound traffic.

### Install

```bash
go get github.com/Viridian-Inc/cloudmock-sdk-go
```

### Initialize

```go
package main

import (
    "net/http"
    cloudmock "github.com/Viridian-Inc/cloudmock-sdk-go"
)

func main() {
    cloudmock.Init(cloudmock.Options{AppName: "my-service"})
    defer cloudmock.Close()

    // Wrap outbound HTTP client
    client := &http.Client{
        Transport: cloudmock.WrapTransport(http.DefaultTransport),
    }

    // Wrap inbound HTTP handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api/health", healthHandler)
    http.ListenAndServe(":8080", cloudmock.WrapHandler(mux))
}
```

### Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `AppName` | `string` | `"go-app"` | Application name in the devtools UI |
| `Host` | `string` | `"localhost"` | CloudMock host |
| `Port` | `int` | `4580` | TCP port for the source server |

### Logging

Send structured log messages to the devtools:

```go
cloudmock.Log("info", "processing order #1234")
cloudmock.Log("error", "payment gateway timeout")
```

### Panic recovery

The SDK includes a panic recovery middleware. When a handler panics, the SDK captures the stack trace and sends it to the devtools before re-panicking:

```go
// WrapHandler already includes panic recovery
http.ListenAndServe(":8080", cloudmock.WrapHandler(mux))
```

## Python SDK

The Python SDK supports WSGI (Flask, Django), ASGI (FastAPI, Starlette), and the `requests` / `urllib3` HTTP libraries.

### Install

```bash
pip install cloudmock
```

### Initialize

```python
import cloudmock

cloudmock.init("my-service")
```

### Options

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `app_name` | `str` | -- (required) | Application name in the devtools UI |
| `host` | `str` | `"localhost"` | CloudMock host |
| `port` | `int` | `4580` | TCP port for the source server |
| `capture_http` | `bool` | `True` | Monkey-patch `requests` / `urllib3` for outbound HTTP |
| `capture_errors` | `bool` | `True` | Install `sys.excepthook` for uncaught exceptions |
| `capture_logging` | `bool` | `True` | Install a `logging.Handler` that forwards log records |

### Flask middleware

```python
from flask import Flask
import cloudmock

app = Flask(__name__)
cloudmock.init("my-flask-app")
app.wsgi_app = cloudmock.get_middleware()(app.wsgi_app)

@app.route("/users")
def list_users():
    table = dynamodb.Table("Users")
    return table.scan()["Items"]
```

### FastAPI middleware

```python
from fastapi import FastAPI
import cloudmock

app = FastAPI()
cloudmock.init("my-fastapi-app")
app.add_middleware(cloudmock.get_asgi_middleware())

@app.get("/users")
async def list_users():
    table = dynamodb.Table("Users")
    return table.scan()["Items"]
```

### Teardown

```python
import atexit
import cloudmock

cloudmock.init("my-service")
atexit.register(cloudmock.close)
```

### Logging integration

When `capture_logging=True` (default), the SDK installs a `logging.Handler` on the root logger. Any log record at any level is forwarded to the devtools. You do not need to change your existing logging configuration:

```python
import logging
logger = logging.getLogger(__name__)
logger.info("Processing order #1234")  # appears in devtools
```

## Lambda attribution

When your application runs inside AWS Lambda (or a Lambda emulated by CloudMock), the SDK automatically reads the `AWS_LAMBDA_FUNCTION_NAME` environment variable and includes it in the `source:register` event. This lets CloudMock link the source to the corresponding Lambda function in the Topology view.

If you are running outside Lambda but want to attribute traffic to a specific Lambda function, set the `X-CloudMock-Source` header on inbound requests to match the function name:

```bash
curl -H "X-CloudMock-Source: my-function" http://localhost:3000/api/invoke
```

## Verifying the connection

After starting your application with the SDK initialized, check the admin API to confirm the source is registered:

```bash
curl http://localhost:4599/api/source/status
```

You should see your application listed as a connected source. Events will begin appearing in the Activity view within 1 second of the first captured event.
