# Benchmark: LocalStack vs CloudMock — Design Spec

## Goal

Build a comprehensive benchmark harness that compares CloudMock and LocalStack across all 100 AWS services, measuring performance (latency, startup, resource usage), correctness (response fidelity), and feature coverage. Results serve three purposes:

1. **Marketing** — publishable competitive data for website/README
2. **QA** — validate CloudMock meets its own SLO targets
3. **Feature parity** — map which operations work in each tool

## Scope

- **Targets:** CloudMock, LocalStack Free, LocalStack Pro (where available)
- **Services:** All 98 CloudMock services, presented in two tiers (25 full, 73 stub)
- **Execution modes:** Docker-vs-Docker (fair), native CloudMock binary (advantage showcase)
- **Environments:** Local (macOS ARM), CI (Linux x86_64 via GitHub Actions)
- **Output:** JSON data files + generated markdown summary

## Architecture

```
benchmarks/
├── cmd/bench/main.go          # CLI entrypoint
├── runner/
│   ├── runner.go               # Orchestration: boot targets, run suites, collect results
│   ├── docker.go               # Docker container lifecycle (start, health-check, stop, stats)
│   └── native.go               # Native binary lifecycle (npx cloudmock, process management)
├── harness/
│   ├── harness.go              # AWS SDK call wrapper with timing + error capture
│   ├── metrics.go              # Latency aggregation (P50/P95/P99), CPU/memory sampling
│   └── correctness.go          # Response schema validation, behavioral checks
├── suites/
│   ├── registry.go             # Suite registration + discovery
│   ├── tier1/                  # Full-implementation service suites
│   │   ├── s3.go
│   │   ├── dynamodb.go
│   │   ├── sqs.go
│   │   ├── sns.go
│   │   ├── lambda.go
│   │   ├── apigateway.go
│   │   ├── cloudformation.go
│   │   ├── cognito.go
│   │   ├── eventbridge.go
│   │   ├── ecs.go
│   │   ├── eks.go
│   │   ├── ec2.go
│   │   ├── rds.go
│   │   ├── iam.go
│   │   ├── sts.go
│   │   ├── route53.go
│   │   ├── cloudwatch.go
│   │   ├── cloudwatchlogs.go
│   │   ├── kms.go
│   │   ├── kinesis.go
│   │   ├── firehose.go
│   │   ├── cloudtrail.go
│   │   ├── codebuild.go
│   │   ├── codepipeline.go
│   │   └── config.go
│   └── tier2/                  # Stub service suites (73 services)
│       ├── gen.go              # Generates stub suites from catalog metadata
│       └── ...                 # One file per stub service
├── report/
│   ├── json.go                 # Structured JSON output
│   └── markdown.go             # Summary tables, charts data, analysis
├── results/                    # Generated output (gitignored)
│   ├── YYYY-MM-DD-results.json
│   └── YYYY-MM-DD-summary.md
└── ci/
    └── benchmark.yml           # GitHub Actions workflow
```

## CLI Interface

```bash
# Full benchmark — all services, all targets, Docker mode
go run ./benchmarks/cmd/bench --target=all --mode=docker

# Single target, specific services
go run ./benchmarks/cmd/bench --target=cloudmock --services=s3,dynamodb,sqs

# Native mode (CloudMock only — LocalStack doesn't have a native binary)
go run ./benchmarks/cmd/bench --target=cloudmock --mode=native

# Both modes for CloudMock, Docker for LocalStack
go run ./benchmarks/cmd/bench --target=all --mode=all

# CI mode (extended iterations, Linux, commits results)
go run ./benchmarks/cmd/bench --target=all --mode=all --ci

# Quick smoke run (1 iteration per operation, skip load test)
go run ./benchmarks/cmd/bench --target=all --quick
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | `all` | `cloudmock`, `localstack`, `localstack-pro`, `all` |
| `--mode` | `docker` | `docker`, `native`, `all` |
| `--services` | `*` | Comma-separated service names, or `*` for all |
| `--tier` | `0` | Filter by tier: `1`, `2`, or `0` for both |
| `--iterations` | `100` | Warm-phase iterations per operation |
| `--concurrency` | `10` | Goroutines for load-phase |
| `--ci` | `false` | CI mode: extended iterations, JSON+MD commit |
| `--quick` | `false` | 1 iteration, no load phase, fast feedback |
| `--output` | `results/` | Output directory |

## Suite Interface

```go
type Suite interface {
    Name() string        // AWS service name (e.g. "s3", "dynamodb")
    Tier() int           // 1 = full implementation, 2 = stub
    Operations() []Operation
}

type Operation struct {
    Name     string
    Setup    func(ctx context.Context, client any) error
    Run      func(ctx context.Context, client any) (any, error)
    Teardown func(ctx context.Context, client any) error
    Validate func(resp any) []CorrectnessFinding
}

type CorrectnessFinding struct {
    Field    string  // e.g. "ResponseMetadata.RequestId"
    Expected string
    Actual   string
    Grade    Grade   // Pass, Partial, Fail
}

type Grade string
const (
    Pass        Grade = "pass"
    Partial     Grade = "partial"
    Fail        Grade = "fail"
    Unsupported Grade = "unsupported"
)
```

## Service Coverage

### Tier 1 — Full Implementation Suites (25 services)

Each suite tests 5-15 operations covering:

- **CRUD lifecycle:** create, read/describe, list, update, delete
- **Edge cases:** duplicate creates, not-found errors, pagination tokens
- **Service-specific behavior:**
  - S3: PutObject, GetObject, DeleteObject, ListObjects, multipart upload, presigned URLs, CopyObject
  - DynamoDB: CreateTable, PutItem, GetItem, Query, Scan, UpdateItem, DeleteItem, conditional writes, BatchWriteItem
  - SQS: CreateQueue, SendMessage, ReceiveMessage, DeleteMessage, visibility timeout, batch operations
  - SNS: CreateTopic, Subscribe, Publish, Unsubscribe, ListSubscriptions
  - Lambda: CreateFunction, Invoke (sync), ListFunctions, GetFunction, DeleteFunction
  - IAM: CreateUser, CreateRole, PutRolePolicy, AttachRolePolicy, GetUser, ListUsers
  - STS: GetCallerIdentity, AssumeRole
  - And so on for all 25 services

### Tier 2 — Stub Suites (73 services)

Each suite tests 3-5 operations:

- **Create** a primary resource
- **Describe/Get** the resource
- **List** resources
- **Delete** the resource
- **Error handling:** describe non-existent resource (expect appropriate error)

Stub suites are generated from CloudMock's `services/stubs/catalog.go` metadata, which already defines the resource types and operations per service. A code generator (`suites/tier2/gen.go`) reads the catalog and produces test files.

## Measurement Design

### Startup Time

Measured from container/process start to first successful `sts:GetCallerIdentity` response.

```
1. Record timestamp T0
2. Start target (docker run / npx cloudmock)
3. Poll sts:GetCallerIdentity every 100ms
4. Record timestamp T1 on first 200 response
5. Startup time = T1 - T0
```

Run 5 times, report median.

### Latency

Each operation runs in three phases:

| Phase | Iterations | Concurrency | Purpose |
|-------|-----------|-------------|---------|
| **Cold** | 1 | 1 | First request after boot |
| **Warm** | 100 (configurable) | 1 | Steady-state sequential latency |
| **Load** | 50 per goroutine | 10 goroutines (configurable) | Throughput + latency under contention |

Latency captured via `time.Now()` around each SDK call. Aggregated into:
- Min, Max, Mean
- P50, P95, P99 (using a histogram)
- Throughput (ops/sec) for load phase

### Resource Usage

Sampled every 500ms during the full benchmark run via Docker API (`/containers/{id}/stats`):

- **Memory:** RSS peak, RSS average (MB)
- **CPU:** Peak %, average % (normalized to 1 core)
- **Disk:** Container filesystem size after all operations

For native mode, sampled via `/proc/{pid}/status` (Linux) or `ps` (macOS).

### Correctness

Each operation's `Validate` function checks:

1. **HTTP status code** — matches expected (e.g. 200 for success, 404 for not-found)
2. **Required fields** — response contains expected top-level and nested fields
3. **Error conventions** — error responses use correct AWS error codes (e.g. `ResourceNotFoundException`, `BucketAlreadyOwnedByYou`)
4. **Behavioral correctness** — side effects actually happened:
   - SQS: message invisible after ReceiveMessage, reappears after visibility timeout
   - S3: object retrievable after PutObject
   - DynamoDB: conditional write fails with ConditionalCheckFailedException

Overall correctness score per service:
```
score = (pass_count * 1.0 + partial_count * 0.5) / total_operations
```

## Report Format

### JSON Schema (`results/YYYY-MM-DD-results.json`)

```json
{
  "meta": {
    "date": "2026-04-02",
    "platform": "darwin/arm64",
    "go_version": "1.26",
    "cloudmock_version": "1.0.3",
    "localstack_version": "3.x.x",
    "mode": "docker",
    "iterations": 100,
    "concurrency": 10
  },
  "startup": {
    "cloudmock_docker": { "median_ms": 850, "runs": [820, 850, 870, 840, 860] },
    "cloudmock_native": { "median_ms": 200, "runs": [190, 200, 210, 195, 205] },
    "localstack_free": { "median_ms": 4200, "runs": [4100, 4200, 4300, 4150, 4250] },
    "localstack_pro": { "median_ms": 5800, "runs": [5700, 5800, 5900, 5750, 5850] }
  },
  "resources": {
    "cloudmock_docker": { "peak_memory_mb": 45, "avg_memory_mb": 32, "peak_cpu_pct": 12, "avg_cpu_pct": 5 },
    "localstack_free": { "peak_memory_mb": 380, "avg_memory_mb": 290, "peak_cpu_pct": 65, "avg_cpu_pct": 30 }
  },
  "services": {
    "s3": {
      "tier": 1,
      "operations": {
        "PutObject": {
          "cloudmock_docker": {
            "cold_ms": 12,
            "warm": { "p50": 2.1, "p95": 4.5, "p99": 8.0, "min": 1.5, "max": 12.0, "mean": 2.8 },
            "load": { "p50": 3.5, "p95": 8.0, "p99": 15.0, "throughput_ops_sec": 2800 },
            "correctness": "pass"
          },
          "localstack_free": {
            "cold_ms": 85,
            "warm": { "p50": 15.3, "p95": 42.0, "p99": 78.0, "min": 10.0, "max": 95.0, "mean": 20.1 },
            "load": { "p50": 25.0, "p95": 65.0, "p99": 120.0, "throughput_ops_sec": 380 },
            "correctness": "pass"
          }
        }
      }
    }
  },
  "summary": {
    "total_services": 98,
    "cloudmock_supported": 98,
    "localstack_free_supported": 24,
    "localstack_pro_supported": 55,
    "avg_latency_ratio": 7.2,
    "correctness_scores": {
      "cloudmock": 0.94,
      "localstack_free": 0.87,
      "localstack_pro": 0.91
    }
  }
}
```

### Markdown Report (`results/YYYY-MM-DD-summary.md`)

Sections:

1. **Executive Summary** — headline numbers: startup time, avg latency ratio, service coverage, memory footprint
2. **Methodology** — how the benchmark was run, hardware, iterations, what was measured
3. **Startup & Resources** — table comparing boot time, memory, CPU across all targets/modes
4. **Tier 1 Service Results** — per-service table with latency (P50/P95/P99), correctness grade, winner column
5. **Tier 2 Service Results** — coverage matrix (supported/unsupported per target), latency for supported operations
6. **Correctness Matrix** — full grid of service x operation with pass/partial/fail/unsupported per target
7. **Latency Distribution** — percentile comparisons for headline services (S3, DynamoDB, SQS, Lambda)
8. **Native vs Docker** — CloudMock native binary advantage over its own Docker image
9. **Conclusions** — category winners, key takeaways

## CI Integration

### GitHub Actions Workflow (`benchmarks/ci/benchmark.yml`)

```yaml
name: Benchmark
on:
  workflow_dispatch:
    inputs:
      services:
        description: 'Services to benchmark (comma-separated or * for all)'
        default: '*'
      mode:
        description: 'docker, native, or all'
        default: 'all'
  push:
    tags: ['v*']

jobs:
  benchmark:
    runs-on: ubuntu-latest-16core  # need resources for fair comparison
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - name: Run benchmark
        run: go run ./benchmarks/cmd/bench --target=all --mode=${{ inputs.mode || 'all' }} --services=${{ inputs.services || '*' }} --ci
      - name: Upload results
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-results
          path: benchmarks/results/
      - name: Commit results
        if: github.ref_type == 'tag'
        run: |
          git add benchmarks/results/
          git commit -m "benchmark: ${{ github.ref_name }} results"
          git push
```

### Local Development

```bash
# Quick feedback loop during development
go run ./benchmarks/cmd/bench --target=cloudmock --services=s3 --quick

# Full local run before pushing
go run ./benchmarks/cmd/bench --target=all --mode=docker
```

## Constraints & Assumptions

- **LocalStack Pro** requires a license key. The harness accepts `LOCALSTACK_API_KEY` env var; if absent, Pro benchmarks are skipped.
- **ARM vs x86:** Local runs on macOS ARM use Rosetta for LocalStack's x86 image. CI runs on native x86. Results are labeled by platform.
- **Network:** All targets run on localhost. No network latency variance.
- **Idempotency:** Each operation's Setup/Teardown ensures clean state. No cross-operation interference.
- **Timeouts:** 30s per operation. Operations exceeding timeout are graded `fail`.

## Build Order

1. Harness infrastructure (metrics, correctness, runner)
2. Tier 1 suites for core services (S3, DynamoDB, SQS — validate the harness works)
3. Report generation (JSON + markdown)
4. Remaining Tier 1 suites
5. Tier 2 stub suite generator
6. CI workflow
7. Native mode support
