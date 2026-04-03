# CloudMock: Open-Source AWS Emulator with Built-In Observability

**Draft launch post for Hacker News / Reddit r/golang, r/aws, r/devops**

---

## HN Title Options (pick one)

1. "CloudMock – Open-source AWS emulator with distributed tracing, error tracking, and chaos engineering"
2. "Show HN: I built an open-source alternative to LocalStack with built-in observability"
3. "CloudMock – 97 AWS services emulated locally with OpenTelemetry, error tracking, and a devtools UI"

---

## Post Body

I've been building CloudMock for the past few months — it's an open-source AWS service emulator that goes beyond just mocking API calls. It includes a full observability stack: distributed tracing, error tracking, log management, alerting, and a desktop devtools app.

**The problem I was solving:**

I was building a React Native app backed by AWS (Lambda, DynamoDB, Cognito, SQS, etc.) and found myself juggling:
- LocalStack for AWS emulation (finicky, incomplete)
- Sentry for error tracking
- CloudWatch console for logs
- Postman for API testing
- Flipper for mobile debugging (deprecated)

Five tools, five contexts, and none of them talked to each other.

**What CloudMock does differently:**

1. **100 AWS services emulated** — not just CRUD stubs. Services have behavioral emulation: CodePipeline triggers CodeBuild, AutoScaling creates EC2 instances, Glue crawlers inspect S3 buckets. Cross-service integration works like real AWS.

2. **OpenTelemetry-native** — any language works. Point your OTel SDK at `localhost:4318` and traces, metrics, and logs flow into the dashboard. Go, Python, Java, Node, Rust — no proprietary SDK required.

3. **Error tracking with grouping** — errors are fingerprinted, grouped, and linked to releases. Like a built-in Sentry for your local dev environment.

4. **Chaos engineering** — inject latency, errors, and throttling into any service to test your error handling before production.

5. **Desktop devtools** — a cross-platform app (Tauri + Preact) that shows your entire system: topology graph, request waterfall, traces, metrics, all in one place. It's what Flipper should have been.

6. **IaC-driven topology** — point CloudMock at your Pulumi or Terraform project and it auto-discovers your architecture. DynamoDB tables, Lambda functions, API routes — all extracted from IaC and rendered as an interactive graph.

**Quick start:**

```bash
npx cloudmock
# Point your AWS SDK:
export AWS_ENDPOINT_URL=http://localhost:4566
# Send traces via OpenTelemetry:
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
# Open devtools:
open http://localhost:4500
```

**What's under the hood:**

- Written in Go (241K lines, 2,900+ tests)
- 48 packages covering everything from OTLP ingestion to anomaly detection
- Plugin system supporting Go, Python, TypeScript plugins (Kubernetes + ArgoCD emulation included)
- Multiple storage backends: in-memory (dev), DuckDB (lightweight), PostgreSQL (production)
- 25-page documentation site

**What makes it different from LocalStack:**

| | CloudMock | LocalStack |
|---|---|---|
| AWS services | 97 (25 full + 72 behavioral) | ~60 (varies by tier) |
| Observability | Built-in (traces, metrics, errors, alerts) | None (need separate tools) |
| OpenTelemetry | Native OTLP ingestion | No |
| Chaos engineering | Built-in | No |
| Desktop devtools | Cross-platform app | Web dashboard only |
| IaC topology | Auto-discovers from Pulumi/Terraform | No |
| Error tracking | Sentry-like error inbox | No |
| Pricing | Free forever (open source) | Free tier limited, Pro $35/dev/mo |

**Current status:** Feature-complete for v1. All P0-P3 roadmap items shipped. Looking for early users and feedback.

**Tech stack:**
- Gateway: Go
- DevTools: Tauri v2 + Preact + TypeScript
- Storage: PostgreSQL / DuckDB / in-memory
- License: Apache 2.0

GitHub: [link when public]

I'd love feedback, especially from teams currently using LocalStack, Sentry, or Datadog for local development. What features would make you switch?

---

## Reddit-Specific Versions

### r/golang

**Title:** "I built a 241K-line Go project: CloudMock — AWS emulator with full observability stack"

**Angle:** Focus on the Go architecture, 48 packages, plugin system, test coverage (2,900+ tests), and how Go made the concurrent service emulation possible.

### r/aws

**Title:** "Open-source alternative to LocalStack with 97 AWS services + built-in tracing and error tracking"

**Angle:** Focus on AWS service coverage, IaC integration, and how it compares to LocalStack. Mention specific services: DynamoDB with expression evaluation, Lambda with code execution, S3 with versioning.

### r/reactnative / r/expo

**Title:** "I built a Flipper replacement for React Native — cross-platform devtools with AWS emulation"

**Angle:** Focus on the mobile dev story: replacing Flipper, seeing network requests + server behavior in one place, BLE mesh topology for IoT apps.

### r/devops

**Title:** "CloudMock: Free, open-source AWS emulator with OpenTelemetry, chaos engineering, and SLO tracking"

**Angle:** Focus on the production-readiness story: OTel support, anomaly detection, SLO burn rates, incident management, and how it fits into a CI/CD pipeline.

---

## Timing Notes

- Post HN on a Tuesday or Wednesday around 9-10am ET for best visibility
- Post Reddit subs individually, not cross-posted
- Have the GitHub repo public before posting
- Have the docs site live (even if basic)
- Prepare for questions about: performance vs LocalStack, production readiness, why not just use moto, how it handles Lambda execution
