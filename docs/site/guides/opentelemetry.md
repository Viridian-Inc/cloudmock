# OpenTelemetry Integration

CloudMock accepts standard OpenTelemetry data via OTLP. Any language with an OpenTelemetry SDK works -- no CloudMock-specific SDK required.

## How It Works

```
Your App ──► OpenTelemetry SDK ──► OTLP export ──► CloudMock (:4318)
                                                        │
                                                        ▼
                                                   DevTools Dashboard
                                                   ├── Trace viewer
                                                   ├── Metrics
                                                   └── Logs
```

CloudMock runs an OTLP/HTTP receiver on port 4318. Point any OpenTelemetry SDK at it, and traces, metrics, and logs appear in DevTools automatically.

## Universal Setup

Set these environment variables before starting your application:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
export OTEL_SERVICE_NAME=my-service
```

That's it. If your app already uses OpenTelemetry (e.g., with Datadog, New Relic, or Honeycomb), you just change the endpoint URL.

## Language-Specific Setup

### Node.js

```bash
npm install @opentelemetry/sdk-node \
  @opentelemetry/exporter-trace-otlp-http \
  @opentelemetry/exporter-metrics-otlp-http \
  @opentelemetry/auto-instrumentations-node
```

```js
// tracing.js -- require this before your app: node -r ./tracing.js app.js
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-metrics-otlp-http');
const { PeriodicExportingMetricReader } = require('@opentelemetry/sdk-metrics');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');

const sdk = new NodeSDK({
  traceExporter: new OTLPTraceExporter({
    url: 'http://localhost:4318/v1/traces',
  }),
  metricReader: new PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter({
      url: 'http://localhost:4318/v1/metrics',
    }),
  }),
  instrumentations: [getNodeAutoInstrumentations()],
  serviceName: 'my-node-service',
});

sdk.start();
```

Auto-instrumentation captures: HTTP requests, Express/Fastify routes, database queries, AWS SDK calls, and more.

### Python

```bash
pip install opentelemetry-sdk \
  opentelemetry-exporter-otlp-proto-http \
  opentelemetry-instrumentation
```

```python
# tracing.py
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource

resource = Resource.create({"service.name": "my-python-service"})
provider = TracerProvider(resource=resource)

exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(provider)
```

Or use the zero-code approach:

```bash
opentelemetry-instrument \
  --traces_exporter otlp \
  --exporter_otlp_endpoint http://localhost:4318 \
  --exporter_otlp_protocol http/json \
  --service_name my-python-service \
  python app.py
```

### Go

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
```

```go
package main

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initTracer() (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(
        context.Background(),
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("my-go-service"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### Java

Use the OpenTelemetry Java Agent for zero-code instrumentation:

```bash
# Download the agent
curl -L -o opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar

# Run your app with the agent
java -javaagent:opentelemetry-javaagent.jar \
  -Dotel.exporter.otlp.endpoint=http://localhost:4318 \
  -Dotel.exporter.otlp.protocol=http/protobuf \
  -Dotel.service.name=my-java-service \
  -jar my-app.jar
```

The Java agent auto-instruments: Spring Boot, JDBC, HTTP clients, gRPC, Kafka, Redis, and 100+ libraries.

For programmatic setup (Spring Boot):

```java
@Bean
public OpenTelemetry openTelemetry() {
    OtlpHttpSpanExporter exporter = OtlpHttpSpanExporter.builder()
        .setEndpoint("http://localhost:4318/v1/traces")
        .build();

    SdkTracerProvider tracerProvider = SdkTracerProvider.builder()
        .addSpanProcessor(BatchSpanProcessor.builder(exporter).build())
        .setResource(Resource.create(Attributes.of(
            ResourceAttributes.SERVICE_NAME, "my-java-service"
        )))
        .build();

    return OpenTelemetrySdk.builder()
        .setTracerProvider(tracerProvider)
        .buildAndRegisterGlobal();
}
```

### Rust

```toml
# Cargo.toml
[dependencies]
opentelemetry = "0.24"
opentelemetry_sdk = { version = "0.24", features = ["rt-tokio"] }
opentelemetry-otlp = { version = "0.17", features = ["http-json"] }
```

```rust
use opentelemetry::global;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{trace::TracerProvider, Resource};

fn init_tracer() -> TracerProvider {
    let exporter = opentelemetry_otlp::new_exporter()
        .http()
        .with_endpoint("http://localhost:4318");

    opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(exporter)
        .with_trace_config(
            opentelemetry_sdk::trace::Config::default()
                .with_resource(Resource::new(vec![
                    opentelemetry::KeyValue::new("service.name", "my-rust-service"),
                ])),
        )
        .install_batch(opentelemetry_sdk::runtime::Tokio)
        .unwrap()
}
```

## Custom Spans

Add custom spans to trace specific operations in your code:

**Node.js:**
```js
const { trace } = require('@opentelemetry/api');
const tracer = trace.getTracer('my-service');

async function processOrder(order) {
  return tracer.startActiveSpan('process-order', async (span) => {
    span.setAttribute('order.id', order.id);
    span.setAttribute('order.total', order.total);
    try {
      await chargePayment(order);
      await sendConfirmation(order);
    } catch (err) {
      span.recordException(err);
      span.setStatus({ code: 2, message: err.message }); // ERROR
      throw err;
    } finally {
      span.end();
    }
  });
}
```

**Python:**
```python
from opentelemetry import trace

tracer = trace.get_tracer("my-service")

def process_order(order):
    with tracer.start_as_current_span("process-order") as span:
        span.set_attribute("order.id", order.id)
        span.set_attribute("order.total", order.total)
        charge_payment(order)
        send_confirmation(order)
```

**Go:**
```go
tracer := otel.Tracer("my-service")

func processOrder(ctx context.Context, order Order) error {
    ctx, span := tracer.Start(ctx, "process-order",
        trace.WithAttributes(
            attribute.String("order.id", order.ID),
            attribute.Float64("order.total", order.Total),
        ),
    )
    defer span.End()

    if err := chargePayment(ctx, order); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    return sendConfirmation(ctx, order)
}
```

## Verifying Data Flow

After setting up OTel, verify data is reaching CloudMock:

```bash
# Check recent traces via Admin API
curl http://localhost:4599/api/traces | jq '.[0]'

# Check metrics
curl http://localhost:4599/api/metrics | jq '.services'
```

Or open DevTools at `http://localhost:4500` and navigate to the Traces view.

## Combining with AWS Requests

When your app makes both AWS SDK calls and application-level operations, CloudMock correlates them:

1. AWS requests are captured by the gateway (port 4566)
2. Application traces are captured via OTLP (port 4318)
3. DevTools shows both in a unified trace timeline

The trace context propagates through AWS services, so a Lambda invocation triggered by an SQS message shows the full chain.

## OTel Collector

If you already run an OpenTelemetry Collector, add CloudMock as an additional exporter:

```yaml
# otel-collector-config.yaml
exporters:
  otlphttp/cloudmock:
    endpoint: http://localhost:4318

service:
  pipelines:
    traces:
      exporters: [otlphttp/cloudmock, otlphttp/datadog]
    metrics:
      exporters: [otlphttp/cloudmock]
    logs:
      exporters: [otlphttp/cloudmock]
```

This lets you send data to both CloudMock (local) and your production observability platform simultaneously.
