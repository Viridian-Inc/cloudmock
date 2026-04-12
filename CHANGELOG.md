# Changelog

## v1.7.5 (2026-04-12)

### Added
- **IaC support: Terraform HCL parser** — parse `.tf` files for DynamoDB tables, Lambda functions, SQS queues, SNS topics, S3 buckets, IAM roles. Extracts `depends_on` and implicit `aws_TYPE.NAME` references for the dependency graph.
- **IaC support: CDK parser** — parse TypeScript CDK constructs (`new dynamodb.Table`, `new lambda.Function`, etc.) with brace-counting for nested props.
- **IaC support: SAM/CloudFormation parser** — parse `template.yaml` for `AWS::DynamoDB::Table`, `AWS::Serverless::Function`, and other resource types. YAML-based with `DependsOn` support.
- **IaC auto-detection** — `--iac` flag now auto-detects Terraform/CDK/SAM/Pulumi by file presence. Hot-reload watcher works for all four frameworks.
- **IaC vs Runtime diff view** — new DevTools panel comparing IaC-declared resources against running state. Shows synced, missing, orphaned resources. Backend: `GET /api/iac/diff`. Frontend: filterable table with status badges.
- **`cmk bench` command** — self-service benchmarking. Starts CloudMock, optionally LocalStack and Moto, runs 8 key operations, prints a markdown comparison table.
- **State auto-load** — CloudMock now auto-detects `.cloudmock/state.json` or `.cloudmock/seed-tables.json` in the working directory. No `--state` flag needed.
- **`@cloudmock/rum`** added to release workflow. Published alongside the main CLI on every tag push.
- **Expanded npm keywords** for better discoverability (localstack, moto, emulator, devtools, observability).
- **CloudMock Cloud Pulumi stack** — `deploy/cloud/` provisions ECS Fargate + RDS TimescaleDB + ALB + ECR for the ingest service.
- IaC support guide (`docs/guides/iac-support.md`) and CLI reference (`docs/reference/cli.md`).

### Fixed
- **Test-mode XML marshal** — `testmode_fast.go` was JSON-marshaling every response body regardless of `Format`, breaking all XML-protocol services (Route53, EC2, SNS, CloudFormation, S3) in test mode. Fixed to branch on `resp.Format` before marshaling.
- **SQS ListQueueTags/TagQueue/UntagQueue** — implemented for both XML query and JSON protocols. Unblocks Terraform `aws_sqs_queue` resources.
- **SNS ListTagsForResource** — implemented for both protocols. Unblocks Terraform `aws_sns_topic` resources.
- **SNS GetTopicAttributes** — now returns default Policy, EffectiveDeliveryPolicy, Owner, and subscription counters. Fixes Terraform crash on `parsing policy: unexpected end of JSON input`.
- **Homebrew SHA replacement** — release workflow sed patterns now use context-aware matching instead of placeholder strings. SHAs correctly update on every release.
- **Release workflow version sync** — Go binary version injected via `-ldflags`, npm/snap versions bumped from tag. All distribution channels now ship the correct version.
- **Guntest Content-Length** — benchmark harness used hardcoded lengths that drifted from actual body sizes. Computed dynamically now.
- **Compat workflow** — removed dead `tests/compat/` step that silently no-op'd.

## v1.7.4 (2026-04-10)

### Fixed
- SNS ListTagsForResource for Terraform compatibility

## v1.7.3 (2026-04-10)

### Fixed
- SQS JSON-protocol tag handlers (ListQueueTags, TagQueue, UntagQueue)
- SNS default topic attributes (Policy, EffectiveDeliveryPolicy)

## v1.7.2 (2026-04-10)

### Fixed
- Test-mode XML marshal — honor Response.Format in testmode_fast.go
- Route53 benchmark harness nil-deref on CreateHostedZone cleanup

## v1.7.1 (2026-04-10)

### Fixed
- Release workflow: sync version across all distribution channels (npm, snap, Go binary)

## v1.0.0 (2026-03-31)

### Added
- 25 Tier 1 AWS services with comprehensive test coverage (1,876+ tests)
- Built-in browser devtools at localhost:4500 (topology, traces, metrics, chaos, and 8 more views)
- Starlight documentation site (46 pages) at cloudmock.io/docs
- `cmk` CLI wrapper (like awslocal for LocalStack)
- `npx cloudmock` zero-install support
- Homebrew formula for macOS/Linux
- 6 language guides (Node.js, Go, Python, Swift, Kotlin, Dart)
- Node.js, Go, Python SDKs for request capture
- AppSync promoted to Tier 1 (27 operations)
- Source server for SDK-captured HTTP requests (POST /api/source/events)
- Admin API + devtools on single port :4500 (no CORS)
- Startup banner showing ports and service count
- Homepage and pricing page at cloudmock.io

### Changed
- Devtools migrated from Tauri desktop app to browser-only SPA
- Old React dashboard replaced by Preact devtools UI
- README rewritten for 1-minute install experience

### Fixed
- Edge service filtering (split(':')[0] → .pop()!)
- Request trace panel state machine (useReducer replaces 10 useState)
- Replay via admin API (no CORS)
- Requests disappearing from topology panel
- OPTIONS request filtering
