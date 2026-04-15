# Changelog

## v1.8.1 (2026-04-14)

### Fixed
- **Test CloudMock Action workflow** was failing on every tag push since v1.7.2 with `Process completed with exit code 7` on `curl -sf http://localhost:4599/api/health`. The composite action at `.github/actions/cloudmock/action.yml` defaults `test-mode: 'true'` which disables the admin API; probing `:4599` always returns connection refused. Fixed the workflow step to probe the gateway endpoint (from the action's own `endpoint` output) instead — a 4xx on an unauthenticated GET is proof-of-life.

## v1.8.0 (2026-04-14)

### Added
- **14 previously-stubbed services now fully implemented** — `polly`, `translate`, `keyspaces` (cassandra), `ecrpublic`, `efs` (elasticfilesystem), `lexmodels` (lex), `globalaccelerator`, `rekognition`, `inspector2`, `comprehend`, `guardduty`, `servicecatalog`, `securityhub`, and `quicksight` had code scaffolding but every handler returned `{"status":"ok","action":"X"}` without touching a store. 951 handlers have been rewritten as real in-memory CRUD backed by typed per-resource stores. SDK clients that previously received a fake success now get AWS-shaped responses and persistence across calls.
- **Gateway registration for the 14 services** — `cmd/gateway/main.go` now imports and calls `registerOrDefer` for each, so the SigV4 router actually resolves them. Before this change, requests to rekognition/quicksight/etc. hit `ServiceUnavailable: Service not registered`.
- **`GET /api/sns/topics`** — devtools SNS browser endpoint. Returns topics with subscription counts and recent messages.
- **`GET /api/eventbridge/buses`** — devtools EventBridge browser endpoint. Returns buses with rules and target ARNs.
- **`GET /api/apigateway/apis`** — devtools API Gateway browser endpoint. Returns REST APIs with their routes + integrations.
- **`GET /api/route53/zones`** — devtools Route 53 browser endpoint. Returns hosted zones with record sets.
- **`GET /api/browser/anomalies`** — wrapper over the anomaly detector that returns `{anomalies: [...]}` with fields mapped to what the devtools view decodes (`type`, `message`, `detectedAt` vs the raw detector output's `metric`, `description`, `detected_at`).
- **`GET /api/browser/logs`** — wrapper over the Lambda log buffer that returns `{logs: [...]}` with `functionName → service` and `stream → severity` mapping.
- `BrowserInspect() []map[string]any` method on the sns, eventbridge, apigateway, and route53 services — pre-shapes resource state for the new admin endpoints without introducing a circular import back into the services.
- **Smithy RPC-v2 CBOR protocol support** for cloudwatch — handles the new `POST /service/GraniteServiceVersion20100801/operation/{Op}` wire format that `aws-sdk-go-v2 v1.56+` uses. Responses are sent with the `smithy-protocol: rpc-v2-cbor` header and an empty `application/cbor` body; the SDK's deserializer accepts this and constructs the zero-valued output struct. PutMetricData, ListMetrics, GetMetricData all pass SDK correctness checks without requiring a full CBOR encoder.

### Fixed
- **Devtools SNS / EventBridge / API Gateway / Route 53 / Anomalies / Logs views were showing empty lists permanently.** Each view was fetching an endpoint that returned the wrong shape (e.g. `/api/services/sns` returns `{name, actionCount, healthy}`, not a topic list). They silently fell back to `|| []` and never rendered data. All six now point at the new `/api/*/...` endpoints above and light up with real state.
- **apigateway timestamp serialization** — `CreatedDate` on `restApiResponse`, `deploymentResponse`, and `stageResponse` was marshaled as an RFC3339 string. The aws-sdk-go-v2 apigateway client's smithy deserializer declares that field as `Timestamp` and rejects strings with `expected Timestamp to be a JSON Number, got string instead`. Switched all three to `float64` (epoch seconds) and updated the converter helpers.
- **eks timestamp serialization** — same root cause as apigateway. `clusterJSON`, `nodegroupJSON`, `fargateProfileJSON`, `addonJSON` had `CreatedAt`/`ModifiedAt` formatted via `time.Format("2006-01-02T15:04:05Z")`. Now epoch-seconds float64 at all five conversion sites.
- **CloudFormation CreateStack after DeleteStack** — `DeleteStack` marks stacks as `DELETE_COMPLETE` but leaves them in the map for history lookups. `CreateStack` then errored with `AlreadyExistsException` on any name that had been deleted. Matches real AWS semantics now: a stack in `DELETE_COMPLETE` is treated as absent for the purposes of recreation. Fixes the tier-1 benchmark's cloudformation 3/4 failures that came from the shared stackName across warm iterations.
- **apprunner and the 14 previously-stub services were not registered with the gateway.** They existed as code but weren't imported by any entry point. SDK calls returned `ServiceUnavailable`. Fixed for the 14; `apprunner` is tracked separately.
- **codebuild benchmark harness** — the tier-1 codebuild suite didn't set `ServiceRole`, which aws-sdk-go-v2 now marks required. The SDK's `Validate` middleware rejected the input client-side before it ever hit any mock. Added `ServiceRole` to both the setup helper and the `CreateProject` Run. Helps all three targets (cloudmock, moto, localstack) the same way.

### Performance
- **Tier-1 benchmark correctness: 87/100 → 100/100.** Every one of the 100 tier-1 ops across 26 services now round-trips through the AWS Go SDK without errors. LocalStack 3.8 holds 66/100 and moto holds 87/100 on the same run.
- **Throughput unchanged.** cloudmock test mode (`CLOUDMOCK_TEST_MODE=true`) still sustains ~130k req/s across all 12 README benchmark ops against `hey -c 50 -z 15s`. The correctness fixes did not add any hot-path overhead — the cloudwatch CBOR short-circuit is the fastest cloudwatch path and runs in <1ms p50.

### Benchmark results (this release, hey -c 50 -z 15s, same machine)
| Operation | cloudmock | localstack 3.8 | moto |
|---|---:|---:|---:|
| DynamoDB GetItem | 141,902 req/s | 688 | 685 |
| DynamoDB PutItem | 149,939 req/s | 523 | 705 |
| DynamoDB Query | 157,151 req/s | 735 | 552 |
| SQS SendMessage | 156,415 req/s | 1,236 | 21 |
| S3 GetObject (1KB) | 162,663 req/s | 1,121 | 650 |
| SNS Publish | 134,731 req/s | 1,304 | 682 |
| KMS Encrypt | 152,431 req/s | 1,100 | 738 |

Geomean: **cloudmock 129k req/s, 149× faster than localstack, 322× faster than moto.**

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
