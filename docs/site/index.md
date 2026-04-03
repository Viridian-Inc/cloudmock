# CloudMock

**Local AWS emulation with built-in observability.** One command gives you 100 AWS services, distributed tracing, error tracking, log management, and a real-time DevTools dashboard.

## Why CloudMock?

- **100 AWS services** running locally -- 25 with full implementation, 73 with behavioral emulation
- **Zero-config observability** -- traces, metrics, logs, and errors captured automatically
- **OpenTelemetry native** -- any language works by pointing OTLP to `localhost:4318`
- **Real-time DevTools** -- see every request, trace, and error as it happens
- **IaC-aware topology** -- auto-discovers Pulumi and Terraform to map your architecture
- **Chaos engineering** -- inject faults, latency, and errors to test resilience
- **Cost intelligence** -- estimate AWS costs from your local usage patterns

## Quick Start

```bash
# Install
npx cloudmock

# Or via Homebrew
brew install cloudmock

# Start CloudMock
cmk start

# Point your app at CloudMock
export AWS_ENDPOINT_URL=http://localhost:4566

# Open DevTools
open http://localhost:4500
```

That's it. Your AWS SDK calls now go through CloudMock, and every request appears in the DevTools dashboard at `localhost:4500`.

## Architecture

```
Your App (any language)
    │
    ├── AWS SDK calls ──────────► Gateway (:4566) ──► 100 AWS services
    │
    ├── OpenTelemetry ──────────► OTLP endpoint (:4318) ──► Traces & metrics
    │
    └── @cloudmock/rum ─────────► RUM endpoint ──► Browser performance
                                       │
                                       ▼
                                  Admin API (:4599)
                                       │
                                       ▼
                                  DevTools Dashboard (:4500)
                                  ├── Request inspector
                                  ├── Trace viewer
                                  ├── Error inbox
                                  ├── Log viewer
                                  ├── Metrics & SLOs
                                  ├── Topology map
                                  ├── Cost dashboard
                                  └── Chaos controls
```

## Default Ports

| Port | Service | Description |
|------|---------|-------------|
| 4566 | Gateway | AWS service endpoint (set as `AWS_ENDPOINT_URL`) |
| 4500 | Dashboard | DevTools web UI |
| 4599 | Admin API | REST API for programmatic access |
| 4318 | OTLP | OpenTelemetry HTTP endpoint for traces, metrics, logs |

## Documentation

- **[Quickstart](getting-started/quickstart.md)** -- zero to DevTools in 5 minutes
- **[Installation](getting-started/installation.md)** -- npm, Homebrew, Docker, binary
- **[Configuration](getting-started/configuration.md)** -- `.cloudmock.yaml` reference
- **[OpenTelemetry](guides/opentelemetry.md)** -- instrument any language
- **[API Reference](reference/api.md)** -- all Admin API endpoints
- **[CLI Reference](reference/cli.md)** -- `cmk` command reference
- **[Service Compatibility](reference/services.md)** -- 100 AWS services matrix
- **[SDKs](sdks/overview.md)** -- optional convenience SDKs

## Service Profiles

CloudMock ships with three profiles that control which AWS services are loaded:

| Profile | Services | Use Case |
|---------|----------|----------|
| `minimal` | 8 core (IAM, STS, S3, DynamoDB, SQS, SNS, Lambda, CloudWatch Logs) | Fast startup, basic development |
| `standard` | 20 tier-1 (adds RDS, EC2, ECS, ECR, etc.) | Most development workflows |
| `full` | All 100 services | Complete AWS emulation |

## Comparison

| Feature | CloudMock | LocalStack | Moto |
|---------|-----------|------------|------|
| AWS services | 98 | 80+ | 150+ (stubs) |
| Built-in DevTools | Yes | No | No |
| Distributed tracing | Yes | No | No |
| Error tracking | Yes | No | No |
| Log management | Yes | No | No |
| OpenTelemetry ingestion | Yes | No | No |
| Chaos engineering | Yes | No | No |
| Cost estimation | Yes | No | No |
| IaC topology | Yes | No | No |
| RUM / browser monitoring | Yes | No | No |
| SLO tracking | Yes | No | No |
| Free | Yes | Partial | Yes |
