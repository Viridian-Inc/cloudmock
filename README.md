# CloudMock

**Local AWS emulation with built-in observability.**

98 AWS services + distributed tracing + error tracking + alerting — in one binary. Language-agnostic via OpenTelemetry.

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

- **98 AWS services** emulated locally — no AWS account needed
- **Full observability** — traces, metrics, logs, and errors in one dashboard
- **Language-agnostic** — works with any OpenTelemetry SDK (Go, Python, Java, Node, Rust, ...)
- **Built-in DevTools** — topology maps, request tracing, chaos engineering
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
| **apt/deb** | `curl -LO https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.4/cloudmock_1.0.4_amd64.deb && sudo apt install cloudmock_1.0.4_amd64.deb` |
| **Shell** | `curl -fsSL https://cloudmock.pages.dev/install.sh \| bash` |

## Services

98 AWS services including S3, DynamoDB, SQS, SNS, Lambda, API Gateway, Cognito, EC2, ECS, EKS, EventBridge, IAM, KMS, RDS, Route 53, Step Functions, and many more.

See the full list at [cloudmock.pages.dev/docs/services](https://cloudmock.pages.dev/docs/).

## Performance

CloudMock is the fastest AWS mock available. Two modes:

**In-Process (Go)** — zero network overhead, sub-millisecond everything:
```go
cm := sdk.New()
cfg := cm.Config()
client := dynamodb.NewFromConfig(cfg) // 20μs per GetItem
```

**HTTP (any language)** — native binary or Docker:
```bash
npx cloudmock  # 65ms startup, <1ms per operation
```

### Benchmarks (P50 latency)

| Service | CloudMock In-Process | CloudMock HTTP | Moto | LocalStack |
|---------|---------------------|---------------|------|------------|
| **DynamoDB GetItem** | **0.020ms** | 0.44ms | 2.41ms | 5.21ms |
| **S3 PutObject** | **0.030ms** | 1.01ms | 2.20ms | 1.57ms |
| **SQS SendMessage** | **0.015ms** | 0.73ms | 2.58ms | 2.60ms |
| **SNS Publish** | — | 1.56ms | 4.00ms | 3.18ms |
| **IAM CreateUser** | — | 0.98ms | 3.10ms | 2.70ms |
| **Startup** | ~1ms | 65ms | 764ms | 2,094ms |

### Cost at 1,000 CI Builds/Day

| | CloudMock (In-Process) | CloudMock (HTTP) | Moto | LocalStack |
|---|---|---|---|---|
| **Annual cost** | **$4.32** | $191 | $1,065 | $17,159 |
| **vs In-Process** | — | 44x more | 247x more | 3,973x more |

[Full benchmark details and methodology](https://cloudmock.pages.dev/docs/reference/benchmarks/)

## SDKs

CloudMock provides native SDK adapters for every major language:

| Language | Package | Install |
|----------|---------|---------|
| **Go** | `github.com/neureaux/cloudmock/sdk` | `go get` (in-process, 20μs/op) |
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

## Comparison

| Feature | CloudMock | LocalStack (Free) | Moto |
|---|---|---|---|
| AWS services | 98 | ~25 | ~100 |
| **Speed vs Moto** | **110x faster** | 0.9x | 1x |
| Distributed tracing | Built-in | No | No |
| Chaos engineering | Built-in | Pro only | No |
| DevTools UI | Built-in | Pro only | No |
| In-process mode | Go SDK | No | Python only |
| Language | Go (single binary) | Python | Python |
| License | BSL 1.1 | Apache 2.0 | Apache 2.0 |

## Documentation

Full docs at **[cloudmock.pages.dev](https://cloudmock.pages.dev)**

## Community

- [GitHub Issues](https://github.com/Viridian-Inc/cloudmock/issues) — bugs and feature requests
- [GitHub Discussions](https://github.com/Viridian-Inc/cloudmock/discussions) — questions and ideas

## License

Business Source License 1.1. Free for local development and internal use. See [LICENSE](LICENSE).

Copyright 2026 Viridian Inc.
