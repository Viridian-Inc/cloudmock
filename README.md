# CloudMock

![AWS Compatibility](https://img.shields.io/badge/AWS_Compatibility-100%25-brightgreen)
![Compat Tests](https://github.com/Viridian-Inc/cloudmock/actions/workflows/compat.yml/badge.svg)
![CI](https://github.com/Viridian-Inc/cloudmock/actions/workflows/ci.yml/badge.svg)

**Local AWS emulation with built-in observability.**

100 AWS services + distributed tracing + error tracking + alerting — in one binary. Language-agnostic via OpenTelemetry.

![CloudMock DevTools — Topology View](docs/devtools-topology.jpg)

## Quick Start

```bash
npx cloudmock
# or
brew install viridian-inc/tap/cloudmock
# or
sudo snap install cloudmock
# or
docker run -p 4566:4566 -p 4500:4500 ghcr.io/viridian-inc/cloudmock:latest
```

Point your AWS SDK:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
```

Open DevTools at [http://localhost:4500](http://localhost:4500)

## Why CloudMock?

- **100 AWS services** emulated locally — no AWS account needed
- **Full observability** — traces, metrics, logs, and errors in one dashboard
- **Language-agnostic** — works with any OpenTelemetry SDK (Go, Python, Java, Node, Rust, ...)
- **Built-in DevTools** — topology maps, request tracing, chaos engineering
- **State snapshots** — export state to JSON, commit to git, restore on startup — everyone shares the same baseline
- **Free for local dev and internal use** — source-available, no account required

## Usage

CloudMock is a drop-in replacement for AWS. Point any SDK at `localhost:4566`:

```js
// Node.js
const client = new S3Client({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
});
```

```python
# Python
s3 = boto3.client("s3", endpoint_url="http://localhost:4566",
    aws_access_key_id="test", aws_secret_access_key="test")
```

```go
// Go
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithBaseEndpoint("http://localhost:4566"))
```

## Install

| Method | Command |
|--------|---------|
| **npm** | `npx cloudmock` |
| **Homebrew** | `brew install viridian-inc/tap/cloudmock` |
| **Snap** | `sudo snap install cloudmock` |
| **Docker** | `docker run -p 4566:4566 -p 4500:4500 ghcr.io/viridian-inc/cloudmock:latest` |
| **apt/deb** | `curl -LO https://github.com/Viridian-Inc/cloudmock/releases/download/v1.5.1/cloudmock_1.5.1_amd64.deb && sudo apt install cloudmock_1.4.0_amd64.deb` |
| **Shell** | `curl -fsSL https://cloudmock.app/install.sh \| bash` |

## Services

100 AWS services including S3, DynamoDB, SQS, SNS, Lambda, API Gateway, Cognito, EC2, ECS, EKS, EventBridge, IAM, KMS, RDS, Route 53, Step Functions, and many more.

See the full list at [cloudmock.app/docs/services](https://cloudmock.app/docs/).

## Performance

CloudMock is the fastest AWS mock available — **249x faster than LocalStack**, **143x faster than Moto**.

### Test Mode

For CI and test suites, test mode strips all observability overhead and uses a Rust-accelerated DynamoDB store. Only the gateway runs — no dashboard, no admin API, no tracing:

```bash
CLOUDMOCK_TEST_MODE=true npx cloudmock  # 0.1s startup, 0.009ms per operation
```

### Benchmarks (requests/sec — 200 concurrent, 30s sustained)

Gatling gun tests using [hey](https://github.com/rakyll/hey). All three running on the same machine.

| Operation | CloudMock | Moto | LocalStack | vs Moto | vs LS |
|---|---|---|---|---|---|
| **DynamoDB GetItem** | **188,652** | 849 | 791 | 222x | 238x |
| **DynamoDB PutItem** | **178,858** | 940 | 742 | 190x | 241x |
| **DynamoDB Query** | **179,983** | 721 | 780 | 250x | 231x |
| **DynamoDB Scan (100 items)** | **43,395** | 98 | 472 | 442x | 92x |
| **SQS SendMessage** | **186,356** | 759 | 1,178 | 246x | 158x |
| **S3 PutObject (1KB)** | **150,946** | 806 | 1,795 | 187x | 84x |
| **S3 GetObject (1KB)** | **188,223** | 834 | 1,240 | 226x | 152x |
| **SNS Publish** | **177,929** | 458 | 1,231 | 388x | 144x |
| **STS GetCallerIdentity** | **167,015** | 741 | 1,229 | 225x | 136x |
| **IAM ListUsers** | **177,417** | 717 | 1,234 | 247x | 144x |
| **EC2 DescribeInstances** | **176,531** | 506 | 1,229 | 349x | 144x |
| **KMS Encrypt** | **183,387** | 812 | 1,168 | 226x | 157x |

**Geometric mean: CloudMock 163,224 req/s — 250x faster than Moto (654 req/s), 162x faster than LocalStack (1,007 req/s).**

<details>
<summary>In-process mode (Go) — 7,366x faster</summary>

Zero network overhead. DynamoDB GetItem at 43ns with zero allocations (frozen JSON cache):

```go
cm := sdk.New()
cfg := cm.Config()
client := dynamodb.NewFromConfig(cfg) // 43ns per GetItem, 0 allocs
```
</details>

### Developer Cost Savings

A team of 10 developers running 20-25 test runs/day saves **6 minutes per run** vs LocalStack.

| Team Size | Wait Time Eliminated | Developer Cost Saved | With Context Switching |
|---|---|---|---|
| 10 devs | 5,625 hrs/yr | **$422K**/yr | **$1.3M**/yr |
| 50 devs | 28,125 hrs/yr | **$2.1M**/yr | **$6.3M**/yr |
| 200 devs | 112,500 hrs/yr | **$8.4M**/yr | **$25.3M**/yr |

*Based on $75/hr loaded cost, 250 working days/year. Context switching multiplier based on Microsoft/Google research showing 15-25 min productivity loss per interruption. [Full methodology](https://cloudmock.app/docs/reference/benchmarks/).*

### What makes CloudMock fast

- **Frozen JSON cache** — items pre-serialized to JSON at write time; reads return cached bytes with zero marshaling (43ns, 0 allocs)
- **Go + fasthttp** — native binary, zero-copy request handling, no interpreter overhead
- **Rust-accelerated DynamoDB** — hot-path PutItem/GetItem via Rust shared library with serde_json + DashMap
- **Direct partition lookup** — O(1) hash key resolution from KeyConditionExpression, limit pushdown to B-tree
- **String-interned keys** — sync.Map intern pool eliminates repeat allocations for partition key lookups
- **Pre-serialized XML** — all 19 XML services serialize to RawBody at handler level, bypassing gateway marshal
- **goccy/go-json everywhere** — 2-3x faster than encoding/json across all 73 JSON services
- **Lock-free SQS** — atomic counter UUID/receipt generation, no crypto/rand syscall per message
- **Test mode** — fasthttp server, all observability stripped, 100 services pre-resolved into plain map

[Full benchmark details and methodology](https://cloudmock.app/docs/reference/benchmarks/)

## SDKs

CloudMock provides native SDK adapters for every major language:

| Language | Package | Install |
|----------|---------|---------|
| **Go** | `github.com/Viridian-Inc/cloudmock/sdk` | `go get` (in-process, 20μs/op) |
| **Python** | `cloudmock` | `pip install cloudmock` |
| **Node.js** | `@cloudmock/sdk` | `npm install @cloudmock/sdk` |
| **Java** | `dev.cloudmock:cloudmock-sdk` | Maven Central |
| **Kotlin** | `dev.cloudmock:cloudmock-sdk` | Gradle |
| **Rust** | `cloudmock` | `cargo add cloudmock` |
| **C/C++** | `libcloudmock` | `make` (static library) |
| **Ruby** | `cloudmock` | `gem install cloudmock` |
| **C#/.NET** | `CloudMock` | `dotnet add package CloudMock` |
| **Swift** | `CloudMock` | Swift Package Manager |

Every SDK auto-starts the CloudMock binary, returns pre-configured AWS clients, and cleans up on exit. One line of code to start testing:

```python
# Python
with mock_aws() as cm:
    s3 = cm.boto3_client("s3")
```

```typescript
// Node.js
const cm = await mockAWS();
const s3 = new S3Client(cm.clientConfig());
```

```java
// Java
try (var cm = CloudMock.start()) {
    var s3 = S3Client.builder().endpointOverride(cm.endpoint()).build();
}
```

```go
// Go (in-process — zero network, 20μs/op)
cm := sdk.New()
s3Client := s3.NewFromConfig(cm.Config())
```

## GitHub Action

One line to add CloudMock to your CI:

```yaml
- uses: viridian-inc/cloudmock-action@v1
```

Auto-installs, starts in test mode (135x faster than LocalStack), health-checks, and sets `AWS_ENDPOINT_URL` for all subsequent steps. Works with Node.js, Python, Go, Java, Rust, and any language with an AWS SDK.

To disable test mode and get full observability in CI:

```yaml
- uses: viridian-inc/cloudmock-action@v1
  with:
    test-mode: 'false'
```

## Create a Project

```bash
npx create-cloudmock-app my-app
```

Generates a complete project with CloudMock pre-configured for your stack. Supports Node.js, Python, Go, Java, and Rust with S3, DynamoDB, and SQS templates.

## Docker Compose Stacks

Eight ready-to-run stacks in [`docker/stacks/`](docker/stacks/):

| Stack | What it includes |
|-------|-----------------|
| `minimal/` | CloudMock only — point any SDK at localhost:4566 |
| `serverless/` | Express API + DynamoDB + SQS |
| `microservices/` | Node.js + Python + Go services via SNS fan-out |
| `data-pipeline/` | S3 ingest → SQS → worker → DynamoDB |
| `webapp-postgres/` | Node API + Postgres + S3 + SQS |
| `fullstack/` | nginx frontend + Node API + DynamoDB |
| `terraform/` | CloudMock + Terraform IaC validation |
| `monitoring/` | CloudMock + Prometheus + Grafana |

```bash
cd docker/stacks/minimal
docker compose up
```

See the [Docker Compose guide](https://cloudmock.app/docs/guides/docker-compose) for quick starts, customization, and how to add your own services.

## Switching from LocalStack or Moto?

- [Migrate from LocalStack](https://cloudmock.app/docs/guides/migrate-from-localstack/) — 5-minute step-by-step
- [Migrate from Moto](https://cloudmock.app/docs/guides/migrate-from-moto/) — Python pytest/unittest migration

## Traffic Recording & Replay

Record real AWS traffic and replay against CloudMock to validate compatibility:

```bash
cloudmock record --output prod-traffic.json    # proxy mode: captures real AWS calls
cloudmock validate --input prod-traffic.json   # replay + compare, exit 0 = all match
```

## CloudTrail Event Replay

Recreate production AWS state from CloudTrail audit logs:

```bash
# Export CloudTrail events from AWS
aws cloudtrail lookup-events --start-time 2026-03-01 --output json > trail.json

# Replay write operations against CloudMock
cloudmock cloudtrail replay --input trail.json --endpoint http://localhost:4566
```

Filter by service, control replay speed, or use the admin API:

```bash
cloudmock cloudtrail replay --input trail.json --services dynamodb,s3 --speed 0
curl -X POST http://localhost:4599/api/cloudtrail/replay -d @trail.json
```

## Comparison

| Feature | CloudMock | LocalStack (Free) | Moto |
|---|---|---|---|
| AWS services | 100 | ~25 | ~100 |
| **Throughput** | **163,224 req/s** | 1,007 req/s | 654 req/s |
| **Speed multiplier** | **baseline** | 162x slower | 250x slower |
| **Avg latency** | **1.1ms** | 170ms | 264ms |
| Test mode (CI) | Built-in | No | No |
| Distributed tracing | Built-in | No | No |
| Chaos engineering | Built-in | Pro only | No |
| DevTools UI | Built-in | Pro only | No |
| In-process mode | Go SDK | No | Python only |
| Language | Go (single binary) | Python | Python |
| License | BSL 1.1 | Apache 2.0 | Apache 2.0 |

## Documentation

Full docs at **[cloudmock.app](https://cloudmock.app)**

## Community

- [GitHub Issues](https://github.com/Viridian-Inc/cloudmock/issues) — bugs and feature requests
- [GitHub Discussions](https://github.com/Viridian-Inc/cloudmock/discussions) — questions and ideas

## License

Business Source License 1.1. Free for local development and internal use. See [LICENSE](LICENSE).

Copyright 2026 Viridian Inc.
