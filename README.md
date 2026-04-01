# CloudMock

**The open-source AWS emulator with built-in observability.**

Local AWS development + distributed tracing + error tracking + alerting -- in one tool. Language-agnostic via OpenTelemetry.

```
  +-----------+        +-------------+        +-----------+
  |  Your App | -----> |  CloudMock  | -----> |  DevTools |
  | (any lang)|  AWS   |  98 services|  OTLP  |  :4500    |
  +-----------+  SDK   +-------------+        +-----------+
                       traces | metrics | logs | topology
```

## Why CloudMock?

- **98 AWS services** emulated locally -- no AWS account needed
- **Full observability** -- traces, metrics, logs, and errors in one dashboard
- **Language-agnostic** -- works with any OpenTelemetry SDK (Go, Python, Java, Node, Rust, ...)
- **Desktop DevTools** -- cross-platform UI for traces, topology, and chaos controls
- **Chaos engineering** -- inject failures to test error handling before production
- **Free forever locally** -- open source, no account required

## Quick Start

```bash
npx cloudmock
# or
brew install cloudmock
# or
docker run -p 4566:4566 -p 4318:4318 -p 4500:4500 neureaux/cloudmock
```

Point your AWS SDK:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
```

Send traces via OpenTelemetry:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

Open DevTools at [http://localhost:4500](http://localhost:4500)

## Usage

### Any AWS SDK

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

### cmk CLI

`cmk` wraps the AWS CLI with the endpoint pre-configured:

```bash
cmk s3 ls
cmk dynamodb list-tables
cmk sqs create-queue --queue-name jobs
```

## Services

**25 Tier 1 services** with full API emulation (429 operations):

S3, DynamoDB, SQS, SNS, Lambda, API Gateway, CloudFormation, CloudWatch, CloudWatch Logs, Cognito, EC2, ECR, ECS, EventBridge, IAM, Kinesis, KMS, Data Firehose, RDS, Route 53, Secrets Manager, SES, SSM, Step Functions, STS

**73 additional services** available as CRUD stubs. See the [compatibility matrix](docs/compatibility-matrix.md) for the full list.

## Configuration

CloudMock reads `cloudmock.yml` from the working directory:

```yaml
profile: standard          # minimal | standard | full
gateway:
  port: 4566
devtools:
  port: 4500
persistence:
  enabled: true
  path: .cloudmock/state
```

See [docs/configuration.md](docs/configuration.md) for the full reference.

## Architecture

CloudMock is a single Go binary that runs an AWS-compatible API gateway, an OpenTelemetry collector, and a web-based DevTools UI. Services are implemented as in-process handlers with a shared plugin framework.

```
cmd/gateway      -> main entry point
gateway/         -> HTTP router, AWS request parsing
services/        -> one package per AWS service
pkg/otel/        -> OpenTelemetry collector and trace storage
pkg/dashboard/   -> embedded DevTools web UI
pkg/admin/       -> admin API for chaos, snapshots, config
```

See [docs/architecture.md](docs/architecture.md) for details.

## Documentation

- [Getting Started](docs/getting-started.md)
- [Configuration](docs/configuration.md)
- [CLI Reference](docs/cli-reference.md)
- [Architecture](docs/architecture.md)
- [Admin API](docs/admin-api.md)
- [Compatibility Matrix](docs/compatibility-matrix.md)

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development environment setup
- Running tests
- Adding new AWS services
- PR guidelines

## Comparison

| Feature | CloudMock | LocalStack (Free) | Moto |
|---|---|---|---|
| AWS services | 98 | ~25 | ~100 |
| Distributed tracing | Built-in | No | No |
| Chaos engineering | Built-in | Pro only | No |
| DevTools UI | Built-in | Pro only | No |
| Language | Go (single binary) | Python | Python |
| License | Apache 2.0 | Apache 2.0 | Apache 2.0 |

## Community

- [GitHub Issues](https://github.com/neureaux/cloudmock/issues) -- bugs and feature requests
- [GitHub Discussions](https://github.com/neureaux/cloudmock/discussions) -- questions and ideas

## License

Apache License 2.0. See [LICENSE](LICENSE).

Copyright 2026 Neureaux.
