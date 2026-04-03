---
title: Switching from LocalStack
description: Migrate from LocalStack to CloudMock in 5 minutes
---

# Switching from LocalStack to CloudMock

CloudMock uses the same port (4566), same endpoint pattern, and same AWS SDK configuration as LocalStack. Most migrations take under 5 minutes.

## Quick Start: One-Line Switch

If you're using Docker Compose, change one line:

```yaml
# Before (LocalStack)
services:
  localstack:
    image: localstack/localstack:latest
    ports: ["4566:4566"]

# After (CloudMock)
services:
  cloudmock:
    image: ghcr.io/viridian-inc/cloudmock:latest
    ports: ["4566:4566"]
```

That's it. Same port, same endpoint. Your app code doesn't change.

## Without Docker

```bash
# Before
pip install localstack && localstack start

# After
npx cloudmock
# or: brew install viridian-inc/tap/cloudmock && cloudmock
```

## Concept Mapping

| LocalStack | CloudMock | Notes |
|------------|-----------|-------|
| `docker run localstack` | `npx cloudmock` | No Docker needed |
| `SERVICES=s3,dynamodb` | `CLOUDMOCK_PROFILE=minimal` | Or `full` for all 99 |
| `localstack status` | `curl localhost:4599/api/services` | Admin API on 4599 |
| `awslocal s3 ls` | `aws --endpoint http://localhost:4566 s3 ls` | Same endpoint |
| `tflocal` | `cloudmock-terraform` | Same pattern |
| `cdklocal` | `cloudmock-cdk` | Same pattern |
| `LOCALSTACK_API_KEY` | Not needed | All features free |
| Cloud Pods (Pro) | `cloudmock --state state.json` | Built-in, free |
| Chaos testing (Pro) | `curl -X POST localhost:4599/api/chaos` | Built-in, free |
| CI Analytics (Pro) | DevTools at localhost:4500 | Built-in, free |

## Environment Variables

| LocalStack | CloudMock |
|------------|-----------|
| `LOCALSTACK_HOSTNAME` | Not needed (always localhost) |
| `EDGE_PORT=4566` | `CLOUDMOCK_PORT=4566` |
| `DEFAULT_REGION=us-east-1` | `CLOUDMOCK_REGION=us-east-1` |
| `SERVICES=s3,sqs` | `CLOUDMOCK_PROFILE=minimal` |
| `DEBUG=1` | Logs always visible |

## Terraform

```bash
# Before
pip install terraform-local && tflocal init && tflocal apply

# After
go install github.com/neureaux/cloudmock/tools/cloudmock-terraform@latest
cloudmock-terraform init && cloudmock-terraform apply
```

## CDK

```bash
# Before
pip install aws-cdk-local && cdklocal deploy

# After
go install github.com/neureaux/cloudmock/tools/cloudmock-cdk@latest
cloudmock-cdk deploy
```

## Python Tests (pytest)

```python
# Before (LocalStack)
import boto3

@pytest.fixture
def s3_client():
    return boto3.client("s3", endpoint_url="http://localhost:4566")

# After (CloudMock)
from cloudmock import CloudMock

@pytest.fixture(scope="session")
def aws():
    with CloudMock() as cm:
        yield cm

@pytest.fixture
def s3_client(aws):
    return aws.boto3_client("s3")
```

## Node.js Tests (Jest)

```javascript
// Before (LocalStack)
const { S3Client } = require("@aws-sdk/client-s3");
const s3 = new S3Client({ endpoint: "http://localhost:4566" });

// After (CloudMock)
const { CloudMock } = require("@cloudmock/sdk");
let cm;
beforeAll(async () => { cm = new CloudMock(); await cm.start(); });
afterAll(async () => { await cm.stop(); });
const s3 = new S3Client(cm.clientConfig());
```

## Go Tests

```go
// Before (LocalStack)
cfg.BaseEndpoint = aws.String("http://localhost:4566")

// After (CloudMock — in-process, 20μs/op)
cm := sdk.New()
defer cm.Close()
s3Client := s3.NewFromConfig(cm.Config())
```

## GitHub Actions

```yaml
# Before
- uses: localstack/setup-localstack@v0.2.3
  with:
    image-tag: latest

# After
- uses: viridian-inc/cloudmock-action@v1
```

## What You Gain

| Feature | LocalStack Free | CloudMock |
|---------|----------------|-----------|
| Services | ~25 | 99 |
| Chaos testing | Pro ($35/mo) | Free |
| State snapshots | Pro ($35/mo) | Free |
| CI analytics | Pro ($35/mo) | Free |
| Traffic replay | No | Free |
| Contract testing | No | Free |
| In-process mode | No | Free (Go) |
| 10 language SDKs | No | Free |
| Memory usage | 583 MB | 67 MB |
| Startup time | 2.1s | 65ms |

## What's Different

- CloudMock doesn't run Lambda code in Docker containers (Lambda handlers are stored but not executed)
- CloudMock's DevTools UI is at port 4500 (not part of the main endpoint)
- CloudMock uses BSL-1.1 license (free for dev/test, commercial use requires license for production hosting)
