# cloudmock

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26%2B-blue)](go.mod)

A standalone, open-source local AWS emulator written in Go. Run a single binary or Docker container and point your AWS CLI and SDKs at `http://localhost:4566` — no internet required.

## Features

- **98 AWS services** — 24 with full emulation (Tier 1), 74 with CRUD stubs (Tier 2)
- **IAM engine** — full policy evaluation, credential verification, or no-auth mode
- **Service profiles** — start only what you need (`minimal`, `standard`, `full`)
- **Web dashboard** at `http://localhost:4500` for inspecting service state
- **Zero external dependencies** — pure Go, no database required
- **Persistence** — optional state snapshots across restarts

---

## Installation

### Docker (recommended)

```bash
docker run -p 4566:4566 -p 4500:4500 ghcr.io/neureaux/cloudmock:latest
```

### Binary

Download from [GitHub Releases](https://github.com/neureaux/cloudmock/releases) for your platform.

### From source

```bash
go install github.com/neureaux/cloudmock/cmd/gateway@latest
```

---

## Quick Start

### Binary

```bash
# Build from source
git clone https://github.com/neureaux/cloudmock
cd cloudmock
make build

# Start with defaults (profile: minimal, IAM: enforce)
./bin/cloudmock start
```

### Docker Compose

```bash
docker compose up -d
```

### go install

```bash
go install github.com/neureaux/cloudmock/cmd/cloudmock@latest
cloudmock start
```

---

## Usage with AWS CLI

Set the `--endpoint-url` flag or configure it via environment variable:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

# Create an S3 bucket
aws s3 mb s3://my-bucket

# Put an object
echo "hello" | aws s3 cp - s3://my-bucket/hello.txt

# List objects
aws s3 ls s3://my-bucket
```

## Usage with AWS SDKs

### Go

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

cfg, _ := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
    config.WithBaseEndpoint("http://localhost:4566"),
    config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
        return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"}, nil
    })),
)
client := s3.NewFromConfig(cfg)
```

### Python (boto3)

```python
import boto3

s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:4566",
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)
s3.create_bucket(Bucket="my-bucket")
```

### Node.js

```js
const { S3Client, CreateBucketCommand } = require("@aws-sdk/client-s3");

const s3 = new S3Client({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
});

await s3.send(new CreateBucketCommand({ Bucket: "my-bucket" }));
```

---

## Service Profiles

| Profile    | Services enabled |
|------------|-----------------|
| `minimal`  | IAM, STS, S3, DynamoDB, SQS, SNS, Lambda, CloudWatch Logs |
| `standard` | All minimal + RDS, CloudFormation, EC2, ECR, ECS, Secrets Manager, SSM, Kinesis, Firehose, EventBridge, Step Functions, API Gateway |
| `full`     | All 98 supported services |
| `custom`   | Services listed under `services:` in `cloudmock.yml` |

Set the profile at startup:

```bash
cloudmock start --profile standard
# or via environment variable
CLOUDMOCK_PROFILE=standard cloudmock start
```

---

## Configuration

Create or edit `cloudmock.yml` in the working directory:

```yaml
region: us-east-1
account_id: "000000000000"
profile: minimal   # minimal | standard | full | custom

iam:
  mode: enforce    # enforce | authenticate | none
  root_access_key: test
  root_secret_key: test

gateway:
  port: 4566

dashboard:
  enabled: true
  port: 4500

admin:
  port: 4599

logging:
  level: info      # debug | info | warn | error
  format: text     # text | json

persistence:
  enabled: false
  # path: /var/lib/cloudmock/data
```

See [docs/configuration.md](docs/configuration.md) for a full reference.

---

## IAM Modes

| Mode           | Behavior |
|----------------|----------|
| `enforce`      | Full IAM policy evaluation. Requests without valid credentials or without an allow policy are denied. |
| `authenticate` | Credentials are verified but policy checks are skipped — every authenticated call succeeds. |
| `none`         | All auth checks are bypassed. Suitable for quick local development only. |

The default credentials accepted in `enforce` and `authenticate` modes are `test` / `test` (configured via `iam.root_access_key` / `iam.root_secret_key`).

---

## Services

### Tier 1 — Full Emulation (24 services)

| Service | AWS Name |
|---------|----------|
| S3 | `s3` |
| DynamoDB | `dynamodb` |
| SQS | `sqs` |
| SNS | `sns` |
| STS | `sts` |
| KMS | `kms` |
| Secrets Manager | `secretsmanager` |
| SSM Parameter Store | `ssm` |
| CloudWatch | `monitoring` |
| CloudWatch Logs | `logs` |
| EventBridge | `events` |
| Cognito | `cognito-idp` |
| API Gateway | `apigateway` |
| Step Functions | `states` |
| Route 53 | `route53` |
| RDS | `rds` |
| ECR | `ecr` |
| ECS | `ecs` |
| SES | `email` |
| Kinesis | `kinesis` |
| Data Firehose | `firehose` |
| CloudFormation | `cloudformation` |
| IAM | `iam` |
| Lambda | `lambda` |

### Tier 2 — CRUD Stubs (74 services)

Tier 2 services support create, describe/get, list, and delete operations with in-memory resource storage. See [docs/compatibility-matrix.md](docs/compatibility-matrix.md) for the full list.

---

## Dashboard

The web dashboard is available at `http://localhost:4500` when `dashboard.enabled: true`. It shows:

- All registered services and their health status
- Per-service resource counts
- Recent request log
- Current configuration

---

## CLI Commands

```
cloudmock start     Start the cloudmock gateway
cloudmock stop      Stop the cloudmock gateway
cloudmock status    Show health status of all services
cloudmock reset     Reset service state (all or specific service)
cloudmock services  List registered services with action counts
cloudmock config    Show current active configuration
cloudmock version   Print version information
cloudmock help      Show help
```

See [docs/cli-reference.md](docs/cli-reference.md) for flags and examples.

---

## Documentation

- [Getting Started](docs/getting-started.md)
- [Configuration Reference](docs/configuration.md)
- [CLI Reference](docs/cli-reference.md)
- [Architecture](docs/architecture.md)
- [Compatibility Matrix](docs/compatibility-matrix.md)
- [Per-service docs](docs/services/)

---

## Contributing

1. Fork the repository and create a feature branch.
2. Run tests: `make test`
3. Add or update service code under `services/<name>/` (Tier 1) or `services/stubs/catalog.go` (Tier 2).
4. Submit a pull request with a clear description of the change.

See [docs/architecture.md](docs/architecture.md) for how to add a new service.

---

## License

Apache License 2.0. See [LICENSE](LICENSE).
