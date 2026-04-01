# SDK Overview

CloudMock follows an **OpenTelemetry-first** approach. You do not need a CloudMock-specific SDK to use CloudMock. Any application instrumented with standard OpenTelemetry works by pointing the OTLP endpoint to CloudMock.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Required: Works with any language via standard protocols       │
│                                                                 │
│  AWS SDK (any language)  →  Gateway (:4566)  →  AWS emulation  │
│  OpenTelemetry SDK       →  OTLP (:4318)     →  Traces/Metrics │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  Optional: Convenience SDKs for enhanced features               │
│                                                                 │
│  @cloudmock/node   →  Auto-discovery, console capture, source  │
│  @cloudmock/rum    →  Browser vitals, error capture, sessions  │
└─────────────────────────────────────────────────────────────────┘
```

## What You Need

| What You're Doing | What You Need | SDK Required? |
|-------------------|---------------|---------------|
| Making AWS SDK calls | `AWS_ENDPOINT_URL=http://localhost:4566` | No |
| Sending traces/metrics | Any OpenTelemetry SDK | No (use standard OTel) |
| Browser monitoring | `@cloudmock/rum` | Yes (browser SDK) |
| Auto-discovery + console capture (Node.js) | `@cloudmock/node` | Optional convenience |

## Language Support

| Language | AWS SDK | Traces via OTel | CloudMock SDK |
|----------|---------|-----------------|---------------|
| **Node.js** | `@aws-sdk/*` | `@opentelemetry/sdk-node` | `@cloudmock/node` (optional) |
| **Python** | `boto3` | `opentelemetry-sdk` | Use OTel directly |
| **Go** | `aws-sdk-go-v2` | `go.opentelemetry.io/otel` | Use OTel directly |
| **Java** | `aws-sdk-java-v2` | OTel Java Agent | Use OTel directly |
| **Rust** | `aws-sdk-rust` | `opentelemetry` crate | Use OTel directly |
| **C#/.NET** | `AWSSDK.*` | `OpenTelemetry.NET` | Use OTel directly |
| **Ruby** | `aws-sdk-ruby` | `opentelemetry-sdk` | Use OTel directly |
| **PHP** | `aws-sdk-php` | `opentelemetry-php` | Use OTel directly |
| **Browser** | N/A | N/A | `@cloudmock/rum` |

## CloudMock SDKs

### @cloudmock/node

Optional convenience wrapper for Node.js. Adds features beyond what standard OTel provides:

- **Auto-discovery** -- detects running CloudMock instance and configures endpoints automatically
- **Console capture** -- intercepts `console.log/warn/error` and forwards to CloudMock as structured logs
- **Source correlation** -- attaches source file/line to requests for DevTools code links
- **Request enrichment** -- adds `x-cloudmock-source` headers for tracing back to source code

See [@cloudmock/node SDK guide](node.md).

### @cloudmock/rum

Browser SDK for Real User Monitoring. Captures:

- **Web Vitals** -- LCP, FID, CLS, TTFB, FCP
- **JavaScript errors** -- unhandled exceptions and promise rejections
- **Network errors** -- failed fetch/XHR requests
- **User sessions** -- page views, navigation timing
- **Custom events** -- application-specific telemetry

See [@cloudmock/rum SDK guide](rum.md).

## The OTel-First Principle

CloudMock will never require a proprietary SDK. The platform is built on open standards:

1. **AWS API protocol** -- standard AWS SDK calls via `AWS_ENDPOINT_URL`
2. **OpenTelemetry Protocol** -- standard OTLP/HTTP on port 4318
3. **SSE** -- standard Server-Sent Events for real-time streaming

If a team is already using OTel with Datadog, New Relic, or Honeycomb, they can try CloudMock by changing one endpoint URL. Zero migration cost.
