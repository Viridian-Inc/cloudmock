---
title: Comparison
description: CloudMock vs LocalStack vs Moto vs SAM Local -- an honest comparison
---

This page compares CloudMock with other popular AWS emulation and mocking tools. The goal is to help you choose the right tool for your use case. All data reflects the state of each project as of early 2026.

## Feature matrix

| Feature | CloudMock | LocalStack | Moto | SAM Local |
|---------|-----------|------------|------|-----------|
| **Service count** | 99 fully emulated services | 80+ (varies by tier) | 150+ (mock-level) | 4 (Lambda, API GW, DynamoDB, S3) |
| **Protocol fidelity** | High -- implements actual AWS wire protocols (Query, JSON, REST-JSON, REST-XML) | High -- aims for API compatibility | Medium -- Python mocks, some protocol gaps | Medium -- focused on SAM/CloudFormation |
| **Implementation language** | Go | Python | Python | Python / Go (Lambda runtime) |
| **Startup time** | < 1 second | 5-15 seconds | N/A (library) | 3-10 seconds |
| **Memory usage (idle)** | ~30 MB | ~200-500 MB | N/A (in-process) | ~100-200 MB |
| **Memory usage (load)** | ~50-150 MB | ~500 MB - 2 GB | Depends on test suite | ~200-500 MB |
| **Single binary** | Yes | No (Python + Docker) | No (Python library) | No (Python + Docker) |
| **Docker required** | No (optional) | Yes (for many services) | No | Yes (for Lambda) |
| **In-process mode** | Yes (Go only, ~20 μs/op) | No | Yes (Python only) | No |
| **Devtools UI** | Yes -- 12-view desktop app (topology, traces, metrics, chaos, etc.) | Yes -- web dashboard (Pro) | No | No |
| **Distributed tracing** | Built-in always-on (W3C traceparent, waterfall + flamegraph) | Pro tier only | No | Limited (X-Ray local) |
| **Chaos engineering** | Built-in (latency, errors, throttling) | No | No | No |
| **IAM emulation** | Full policy evaluation with enforce/authenticate/none modes | Pro tier only | Partial | No |
| **Pricing** | Free and open source | Free tier + Pro ($35/mo) + Team + Enterprise | Free and open source | Free and open source |
| **CI/CD integration** | `npx cloudmock`, `go install`, Docker | Docker | pip install | SAM CLI |
| **Language SDKs** | Node.js, Go, Python, Java, Rust, Ruby (with trace propagation) | None (use AWS SDKs directly) | Python only (decorator-based) | None (use AWS SDKs directly) |
| **Persistence** | Snapshot, DuckDB, PostgreSQL | Pro tier (persistence) | In-memory only | In-memory only |
| **State snapshots** | Built-in — export/import JSON, commit to git, `--state` flag on startup | Cloud Pods (Pro tier only) | No equivalent | No equivalent |
| **Traffic recording & replay** | Built-in (proxy + Go SDK interceptor, validate subcommand) | No | No | No |
| **CloudTrail event replay** | Built-in (recreate state from audit logs) | No | No | No |
| **Contract testing** | Built-in (dual-mode proxy compares real AWS vs CloudMock live) | No | No | No |
| **GitHub Action** | `viridian-inc/cloudmock-action@v1` — one-line CI setup | Yes (localstack-action) | No | No |
| **Scaffolding CLI** | `create-cloudmock-app` — 9 templates (Node/Python/Go/Java/Rust) | No | No | No |
| **Multi-account** | Single account (configurable ID) | Pro tier | In-process mocking | Single account |

## Traffic recording & replay

| Feature | CloudMock | LocalStack | Moto |
|---|---|---|---|
| Traffic recording & replay | Built-in (proxy + SDK interceptor) | No | No |
| CloudTrail event replay | Built-in (recreate state from audit logs) | No | No |

## IaC tool support

| Feature | CloudMock | LocalStack (Free) | Moto |
|---|---|---|---|
| Terraform wrapper | `cloudmock-terraform` | `tflocal` (Pro) | No |
| CDK support | `cloudmock-cdk` | Yes | No |
| Pulumi support | `cloudmock-pulumi` + native provider | No | No |
| CloudFormation resources | 30 types | Full (Pro) | No |

## Benchmarks

CloudMock wins **31/31 operations** tested against Moto and LocalStack in a standard benchmark covering S3, DynamoDB, SQS, SNS, and IAM operations.

| Mode | CloudMock | Moto | LocalStack |
|------|-----------|------|------------|
| **In-process (Go)** | ~20 μs/op | N/A | N/A |
| **HTTP (localhost)** | ~1-2 ms/op | ~2-3 ms/op | ~5-15 ms/op |
| **Startup time** | < 1 second | N/A (library) | 5-15 seconds |
| **Memory (idle)** | ~30 MB | ~50-100 MB (per test suite) | ~200-500 MB |

CloudMock's in-process Go mode is **over 110x faster** than Moto running equivalent operations in HTTP/server mode. Even in HTTP mode, CloudMock's single Go binary consistently outperforms LocalStack's Python stack.

## Cost comparison

| Scenario | CloudMock | LocalStack | Moto |
|----------|-----------|------------|------|
| Individual developer | $0 | $0 (limited) / $35/mo (Pro) | $0 |
| Team of 5 | $0 | $175/mo (Pro) | $0 |
| Team of 20 | $0 | $700/mo (Pro) | $0 |
| IAM enforcement | $0 | $35+/mo (Pro) | $0 (partial) |
| Devtools / dashboard | $0 | $35+/mo (Pro) | N/A |
| Persistence | $0 | $35+/mo (Pro) | N/A |

CloudMock is fully open source with no paid tiers. All features — including IAM enforcement, distributed tracing, chaos engineering, and persistence — are available at no cost.

## Detailed comparison

### CloudMock vs LocalStack

**LocalStack** is the most established AWS emulator. It has broad service coverage and a large community.

Where CloudMock is stronger:
- **Startup speed**: CloudMock starts in under 1 second (single Go binary). LocalStack takes 5-15 seconds to initialize its Python/Docker stack.
- **Resource usage**: CloudMock idles at ~30 MB of memory. LocalStack typically uses 200-500 MB, and more under load.
- **In-process testing**: CloudMock embeds directly into Go test binaries for ~20 μs/op operation speed. LocalStack has no equivalent.
- **Devtools**: CloudMock includes a 12-view desktop observability console with topology maps, distributed tracing, chaos engineering, and incident tracking. LocalStack's dashboard is available only in the Pro tier.
- **IAM enforcement**: CloudMock includes full IAM policy evaluation in the free tier. LocalStack restricts IAM to Pro.
- **No Docker dependency**: CloudMock runs as a single binary. LocalStack requires Docker for many services.
- **Chaos engineering**: Built into CloudMock. Not available in LocalStack.
- **Cost**: CloudMock is fully open source with no paid tiers. LocalStack's free tier has limitations; full features require Pro ($35/month per developer).
- **Multi-language SDKs**: CloudMock provides SDK adapters for Go, Node.js, Python, Java, Rust, and Ruby with trace propagation. LocalStack provides none.

Where LocalStack is stronger:
- **Service count**: LocalStack supports more services at deeper implementation levels, especially for complex services like CloudFormation, ECS, and EKS.
- **Community and ecosystem**: LocalStack has a larger user base, more Stack Overflow answers, and more third-party integrations.
- **CloudFormation**: LocalStack has more complete CloudFormation support, including more resource types and intrinsic functions.
- **Maturity**: LocalStack has been in development since 2017 and has addressed more edge cases.

### CloudMock vs Moto

**Moto** is a Python library that mocks AWS services at the SDK level using decorators or a standalone server mode.

Where CloudMock is stronger:
- **Language independence**: CloudMock is an HTTP server that works with any AWS SDK (Node.js, Go, Python, Java, Kotlin, Swift, Dart, Rust, Ruby). Moto is primarily a Python library.
- **Performance**: CloudMock's in-process Go mode runs at ~20 μs/op -- over 110x faster than Moto's server mode. Even CloudMock's HTTP mode outperforms Moto.
- **Benchmark results**: CloudMock wins 31/31 operations tested.
- **Devtools**: CloudMock includes observability tools. Moto has no UI.
- **Protocol fidelity**: CloudMock implements actual AWS HTTP protocols. Moto intercepts at the Python SDK level, which can miss protocol-level bugs.
- **Integration testing**: CloudMock runs as a real HTTP server, testing the full SDK-to-server path. Moto intercepts requests before they leave the process.

Where Moto is stronger:
- **Service breadth**: Moto supports 150+ services with deep mock coverage.
- **Test isolation**: As a Python library with decorators, Moto provides excellent per-test isolation without starting/stopping servers.
- **No server process**: Moto runs in-process, so there is no server to manage.
- **Python ecosystem**: Moto integrates natively with pytest, unittest, and other Python test frameworks.

### CloudMock vs SAM Local

**SAM Local** (`sam local`) is AWS's official tool for running Lambda functions and API Gateway locally.

Where CloudMock is stronger:
- **Service coverage**: CloudMock emulates 99 services. SAM Local supports Lambda, API Gateway, DynamoDB (via DynamoDB Local), and S3.
- **Devtools**: CloudMock includes topology maps, tracing, metrics, and chaos engineering. SAM Local has no equivalent.
- **Startup speed**: CloudMock starts in under 1 second. SAM Local takes 3-10 seconds, longer when building containers.
- **Protocol coverage**: CloudMock covers all 4 AWS wire protocols. SAM Local focuses on Lambda invocation and API Gateway routing.

Where SAM Local is stronger:
- **Lambda execution**: SAM Local actually runs Lambda functions in Docker containers with the correct runtime. CloudMock stubs Lambda invocations.
- **CloudFormation integration**: SAM Local reads SAM templates directly and provisions resources. CloudMock requires separate configuration.
- **Official AWS support**: SAM Local is maintained by AWS and tracks service changes closely.
- **Step Functions Local**: SAM integrates with Step Functions Local for workflow testing.

## When to use each tool

| Use case | Recommended tool |
|----------|-----------------|
| Fast feedback loop during development | CloudMock |
| CI/CD integration tests (all languages) | CloudMock |
| Go tests with maximum speed (~20 μs/op) | CloudMock (in-process) |
| Python unit tests with fine-grained mocking | Moto |
| Testing Lambda function execution | SAM Local |
| Deep CloudFormation testing | LocalStack Pro |
| Observability and chaos engineering during dev | CloudMock |
| Team environment with shared state | CloudMock (production mode) or LocalStack Pro |
| Budget-conscious teams | CloudMock or Moto (both fully free) |
| Maximum service coverage regardless of cost | LocalStack Enterprise |

## Migration from LocalStack

If you are currently using LocalStack, migrating to CloudMock is straightforward because both tools serve the AWS API on a configurable port:

1. Change the endpoint URL from LocalStack's port (default 4566) to CloudMock's port (also 4566 by default, so this may be a no-op).
2. Verify that the services your application uses are in CloudMock's [compatibility matrix](/docs/services/).
3. Replace any LocalStack-specific admin API calls with the equivalent CloudMock admin API calls.
4. Update CI/CD scripts to install and start CloudMock instead of LocalStack.
