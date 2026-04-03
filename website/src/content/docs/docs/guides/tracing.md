---
title: Distributed Tracing
description: Auto-instrumented tracing with W3C traceparent propagation
---

CloudMock automatically generates a trace span for every request. No configuration is required -- tracing is always on.

## How it works

Every request that passes through the CloudMock gateway receives:

- A **Trace ID** (W3C-compatible, 32-character hex string)
- A **Span ID** (W3C-compatible, 16-character hex string)
- Response headers: `X-Cloudmock-Trace-Id`, `X-Cloudmock-Span-Id`, and `traceparent`

Traces are stored in an in-memory trace store and displayed in the DevTools Traces view.

## Viewing traces in DevTools

Open the DevTools UI at [http://localhost:4500](http://localhost:4500) and navigate to the **Traces** tab. You will see a list of all recent traces with waterfall and flamegraph views. See the [Traces View](/docs/devtools/traces/) documentation for details.

## W3C traceparent propagation

CloudMock implements the [W3C Trace Context](https://www.w3.org/TR/trace-context/) specification. If your application sends a `traceparent` header, CloudMock will:

1. **Parse** the incoming trace ID and parent span ID
2. **Preserve** the trace ID across the request
3. **Generate** a new span ID for the CloudMock span
4. **Return** an updated `traceparent` header in the response

### traceparent format

```
traceparent: 00-{32-char-hex-trace-id}-{16-char-hex-span-id}-01
```

### Example

Send a request with a traceparent header:

```bash
curl -H "traceparent: 00-abcdef1234567890abcdef1234567890-1234567890abcdef-01" \
     http://localhost:4566/my-bucket
```

The response will include:

```
traceparent: 00-abcdef1234567890abcdef1234567890-<new-span-id>-01
X-Cloudmock-Trace-Id: abcdef1234567890abcdef1234567890
X-Cloudmock-Span-Id: <new-span-id>
```

The trace ID is preserved from the incoming request. The span ID is always new, representing CloudMock's span in the trace.

## Connecting to Jaeger or Zipkin

In production mode, CloudMock emits OpenTelemetry spans via the OTel SDK. Configure the OTLP exporter to send spans to your collector:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
cloudmock --production
```

This sends spans to any OpenTelemetry-compatible backend (Jaeger, Zipkin, Datadog, Honeycomb, etc.).

## Go SDK tracing

The Go SDK supports tracing via the `WithTracing()` option:

```go
import "github.com/neureaux/cloudmock/sdk"

cm := sdk.New(sdk.WithTracing())
defer cm.Close()

// All AWS SDK calls through cm.Config() are now traced.
client := s3.NewFromConfig(cm.Config(), func(o *s3.Options) {
    o.UsePathStyle = true
})

client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})

// Access the trace store programmatically.
traces := cm.TraceStore().Recent("", nil, 10)
fmt.Printf("Captured %d traces\n", len(traces))
```

When tracing is enabled, the SDK:

- Creates an in-memory TraceStore and wires it to the gateway
- Propagates `traceparent` headers from the Go context (if an OTel span is active)
- Stores all spans for programmatic inspection in tests

### Example: trace a request through your app and CloudMock

```go
func TestOrderWorkflow(t *testing.T) {
    cm := sdk.New(sdk.WithTracing())
    defer cm.Close()

    ddb := dynamodb.NewFromConfig(cm.Config())
    s3c := s3.NewFromConfig(cm.Config(), func(o *s3.Options) { o.UsePathStyle = true })

    // Your application code makes AWS calls...
    s3c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("orders")})
    ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
        TableName: aws.String("orders"),
        // ...
    })

    // Inspect traces to verify your app's AWS call patterns.
    traces := cm.TraceStore().Recent("", nil, 50)
    for _, tr := range traces {
        t.Logf("trace=%s service=%s action=%s duration=%.2fms",
            tr.TraceID, tr.RootService, tr.RootAction, tr.DurationMs)
    }
}
```
