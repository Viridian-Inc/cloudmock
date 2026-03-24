# Changelog

All notable changes to CloudMock will be documented in this file.

## [1.0.0] - 2026-03-23

### Added

#### AWS Services
- 24 fully-implemented Tier 1 services: S3, DynamoDB, Lambda, SQS, SNS, STS, KMS, Secrets Manager, SSM, CloudWatch, CloudWatch Logs, EventBridge, Cognito, API Gateway, Step Functions, Route 53, RDS, ECR, ECS, SES, Kinesis, Firehose, CloudFormation, IAM
- 74 Tier 2 CRUD stub services via generic stub engine
- Cross-service integrations: SNS→SQS, SNS→Lambda, EventBridge→Lambda, S3→EventBridge, DynamoDB Streams
- Full IAM policy evaluation engine with enforce/authenticate/none modes
- OAuth2/OIDC endpoints for Cognito user pools

#### Observability Console
- 3-panel topology layout (RequestExplorer, TopologyCanvas, ServiceInspector)
- Distributed tracing with span merging and waterfall visualization
- AI-powered request explanation with narrative debug reports
- SLO engine with configurable thresholds and burn rate tracking
- Regression detection (6 algorithms: latency, error rate, tenant outlier, cache miss, DB fanout, payload growth)
- Side-by-side trace comparison with route baseline synthesis
- Cost intelligence with configurable AWS pricing and per-route/tenant/trend breakdowns
- Incident service with alert grouping, auto-create, auto-resolve
- Advanced profiling: CPU/heap/goroutine capture, flame graphs, source map symbolication

#### Enterprise Features
- JWT authentication with RBAC (admin/editor/viewer roles)
- Tenant isolation with scoped data visibility
- Audit logging for all mutating API actions
- Webhook integrations (Slack, PagerDuty, generic)
- Exportable incident reports (JSON, CSV, HTML)
- Token bucket rate limiting with proxy header support
- Backend preference store (replaces browser localStorage)

#### Infrastructure
- Production data plane: DuckDB (embedded columnar) + PostgreSQL + Prometheus
- OpenTelemetry SDK integration with OTel Collector pipeline
- Docker Compose deployment with multi-stage Dockerfile
- Saved query views API
- Graceful shutdown with SIGTERM/SIGINT handling
- Structured logging with log/slog (JSON or text format)
- Environment variable overrides for all configuration
- Health endpoint with dataplane connectivity check
- Version endpoint with build-time injection

#### Dashboard
- Autotend design system: Figtree font, brand navy/blue/teal palette
- Grouped sidebar navigation (Observe, Respond, Resources, Settings)
- Incidents and Regressions pages with summary cards and filterable tables
- Settings page (webhooks, users, audit tabs)
- Cost tab in Metrics page
- Profile tab in ServiceInspector with flame graph rendering
- Saved views picker for filter presets
- Trace comparison view with span alignment

#### Developer Experience
- Zero-config local development (`go run cmd/gateway/main.go`)
- AWS CLI, SDK Go v2, SDK JS v3, and boto3 compatibility
- Request replay for debugging
- Blast radius analysis
- Command palette (Cmd+K)
- DynamoDB browser with query builder, PartiQL, and terminal REPL
- S3, SQS, Cognito, Lambda, IAM resource browsers
