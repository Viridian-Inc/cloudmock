---
title: Switching from LocalStack
description: Step-by-step guide for migrating an existing LocalStack project to CloudMock, with benchmarks, CI/CD recipes, and common gotchas
---

# Switching from LocalStack to CloudMock

CloudMock is a drop-in replacement for LocalStack. Same port (`4566`), same endpoint pattern, same AWS SDK configuration. Most migrations take under 5 minutes — one Docker Compose line and you're done. This guide walks through a complete migration, including CI/CD changes and the gotchas that trip people up.

## Why migrate?

| Operation | CloudMock req/s | LocalStack req/s | Speedup |
|---|---|---|---|
| DynamoDB GetItem | **188,652** | 791 | **238×** |
| DynamoDB PutItem | **178,858** | 742 | **241×** |
| DynamoDB Query | **179,983** | 780 | **231×** |
| DynamoDB Scan | **43,395** | 472 | **92×** |
| SQS SendMessage | **186,356** | 1,178 | **158×** |
| S3 PutObject | **150,946** | 1,795 | **84×** |
| S3 GetObject | **188,223** | 1,240 | **152×** |
| SNS Publish | **177,929** | 1,231 | **144×** |
| KMS Encrypt | **183,387** | 1,168 | **157×** |
| EC2 DescribeInstances | **176,531** | 1,229 | **144×** |

Geometric mean: CloudMock is **162× faster** than LocalStack across the published benchmark set. Full methodology at [cloudmock.app/docs/reference/benchmarks](/docs/reference/benchmarks/).

Beyond raw throughput:

| | LocalStack free | CloudMock |
|---|---|---|
| Startup time | 2.1 s | **65 ms** |
| Memory footprint | 583 MB | **67 MB** |
| Services | ~25 | **99** |
| Chaos testing | LocalStack Pro ($35/mo) | **Free** |
| State snapshots | LocalStack Pro ($35/mo) | **Free** |
| CI analytics | LocalStack Pro ($35/mo) | **Free** |
| Contract testing | Not available | **Free** |
| In-process Go mode | Not available | **Free (20 μs/op)** |
| Language SDK wrappers | — | **10 languages** |

For a test suite making 20–25 runs per day per developer, CloudMock typically saves **2+ hours of wait time per developer per day**. A team of 10 saves ~5,600 hours per year.

## Step-by-step migration

The migration is five steps. On a typical project, each step is a single-file change.

### Step 1 — Replace the LocalStack image (or binary)

Pick your installation method. The port and endpoint pattern stay the same.

**Docker Compose:**

```yaml
# Before (LocalStack)
services:
  localstack:
    image: localstack/localstack:latest
    ports: ["4566:4566"]
    environment:
      SERVICES: s3,dynamodb,sqs
      DEBUG: 1

# After (CloudMock)
services:
  cloudmock:
    image: ghcr.io/viridian-inc/cloudmock:latest
    ports:
      - "4566:4566"  # AWS gateway
      - "4500:4500"  # DevTools UI (optional)
    environment:
      CLOUDMOCK_PROFILE: minimal
```

**Standalone binary:**

```bash
# Before
pip install localstack
localstack start

# After
npx cloudmock
# or: brew install viridian-inc/tap/cloudmock && cloudmock
# or: curl -fsSL https://cloudmock.app/install.sh | bash
```

**No changes to application code are required.** Your AWS SDK clients keep pointing at `http://localhost:4566`.

### Step 2 — Map environment variables

Most LocalStack environment variables either have no equivalent (because CloudMock doesn't need them) or map 1:1:

| LocalStack | CloudMock | Notes |
|---|---|---|
| `LOCALSTACK_HOSTNAME` | — | Always `localhost` |
| `EDGE_PORT=4566` | `CLOUDMOCK_PORT=4566` | Same default |
| `DEFAULT_REGION=us-east-1` | `CLOUDMOCK_REGION=us-east-1` | Same default |
| `SERVICES=s3,sqs,dynamodb` | `CLOUDMOCK_PROFILE=minimal` | Or `full` for all 99 |
| `DEBUG=1` | — | Logs always visible |
| `LOCALSTACK_API_KEY` | — | CloudMock has no paid tier |
| `PERSISTENCE=1` | `--state state.json` flag | See Step 5 below |
| `LAMBDA_EXECUTOR=docker` | — | CloudMock stores Lambda handlers but does not execute their code |

### Step 3 — Verify your SDK configuration still works

CloudMock uses the exact same endpoint URL and path-style addressing as LocalStack, so existing client configs keep working unchanged:

```python
# Python (boto3) — no change required
import boto3
s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:4566",
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)
```

```javascript
// Node.js — no change required
import { S3Client } from "@aws-sdk/client-s3";
const s3 = new S3Client({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
});
```

```go
// Go — no change required
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithBaseEndpoint("http://localhost:4566"))
```

If your project currently uses `awslocal`, `tflocal`, or `cdklocal`, swap them for the CloudMock equivalents (all of which follow the same CLI pattern):

```bash
# Before                          # After
awslocal s3 ls                    aws --endpoint http://localhost:4566 s3 ls
tflocal init                      cloudmock-terraform init
cdklocal deploy                   cloudmock-cdk deploy
```

The CloudMock variants are installed via `go install github.com/Viridian-Inc/cloudmock/tools/cloudmock-terraform@latest` and `...@latest/tools/cloudmock-cdk`. If you prefer to avoid installing extra binaries, `aws --endpoint http://localhost:4566` works identically to `awslocal`.

### Step 4 — Update CI/CD

CloudMock ships a first-party GitHub Action that replaces LocalStack's setup action with one line:

```yaml
# Before (LocalStack)
- uses: localstack/setup-localstack@v0.2.3
  with:
    image-tag: latest
    install-awslocal: "true"

# After (CloudMock)
- uses: viridian-inc/cloudmock-action@v1
```

The action installs CloudMock, starts it in test mode (135× faster than LocalStack), runs a health check, and sets `AWS_ENDPOINT_URL` for all subsequent steps.

**GitLab CI:**

```yaml
# Before (LocalStack)
test:
  image: python:3.12
  services:
    - name: localstack/localstack:latest
      alias: localstack
  variables:
    AWS_ENDPOINT_URL: http://localstack:4566
    AWS_ACCESS_KEY_ID: test
    AWS_SECRET_ACCESS_KEY: test
  script:
    - pip install -r requirements.txt
    - pytest

# After (CloudMock)
test:
  image: python:3.12
  services:
    - name: ghcr.io/viridian-inc/cloudmock:latest
      alias: cloudmock
  variables:
    AWS_ENDPOINT_URL: http://cloudmock:4566
    AWS_ACCESS_KEY_ID: test
    AWS_SECRET_ACCESS_KEY: test
    CLOUDMOCK_TEST_MODE: "true"  # strips observability for max throughput
  script:
    - pip install -r requirements.txt
    - pytest
```

`CLOUDMOCK_TEST_MODE=true` turns off observability (traces, metrics, dashboard) for ~2× additional throughput — ideal for CI where you only need the fast HTTP responses.

### Step 5 — Port state persistence (if you use it)

If your LocalStack setup relies on `PERSISTENCE=1` or Cloud Pods for shared developer state, CloudMock has a first-class equivalent that works on free tier:

```bash
# Export current state to a file
curl -X POST http://localhost:4599/api/snapshot/export > state.json

# Commit state.json to git so every developer has the same baseline
git add state.json && git commit -m "dev: update shared test fixtures"

# Load at startup
cloudmock --state state.json
```

The state file is plain JSON and diff-friendly. Teams typically commit it to a `fixtures/` directory and refresh it whenever the shared dev baseline changes.

## Per-language test setup

CloudMock ships native language SDKs that auto-start the binary and clean up on exit. If your existing test setup just points boto3/aws-sdk-js at `http://localhost:4566`, **no changes are needed** — keep using what you have. The native SDKs are optional convenience wrappers for test suites that want lifecycle management.

### Python (pytest)

```python
# Before (LocalStack) — still works with CloudMock unchanged
@pytest.fixture
def s3_client():
    return boto3.client("s3", endpoint_url="http://localhost:4566")

# After (CloudMock native SDK) — auto-starts, auto-cleans
from cloudmock import CloudMock

@pytest.fixture(scope="session")
def aws():
    with CloudMock() as cm:
        yield cm

@pytest.fixture
def s3_client(aws):
    return aws.boto3_client("s3")
```

### Node.js (Jest / Vitest)

```javascript
// Before (LocalStack) — still works with CloudMock unchanged
const s3 = new S3Client({ endpoint: "http://localhost:4566" });

// After (CloudMock native SDK)
import { CloudMock } from "@cloudmock/sdk";

let cm;
beforeAll(async () => { cm = new CloudMock(); await cm.start(); });
afterAll(async () => { await cm.stop(); });
const s3 = new S3Client(cm.clientConfig());
```

### Go

Go gets a bonus: in-process mode eliminates the HTTP layer entirely (~20 μs/op, 7,366× faster than the fastest HTTP path).

```go
// Before (LocalStack)
cfg.BaseEndpoint = aws.String("http://localhost:4566")

// After (CloudMock in-process)
import "github.com/Viridian-Inc/cloudmock/sdk"

cm := sdk.New()
defer cm.Close()
s3Client := s3.NewFromConfig(cm.Config())
```

See the [language guides](/docs/language-guides/) for Java, Kotlin, Rust, Ruby, Swift, C#, and C/C++.

## Common gotchas

These are the issues people most often hit during migration. Most are one-line fixes once you know what to look for.

### Port 4566 vs Port 4500

LocalStack uses a single port (`4566`) for the AWS gateway. CloudMock splits responsibilities: `4566` is the AWS gateway (SDK traffic), and `4500` is the DevTools dashboard UI. If you only care about AWS emulation, expose `4566` and skip `4500`. If you want the observability dashboard, expose both.

### `AWS_ENDPOINT_URL` vs per-service endpoint config

Recent AWS SDKs honor the `AWS_ENDPOINT_URL` environment variable for all services at once. If you're migrating from a LocalStack config that listed endpoints per service (`S3_ENDPOINT`, `DYNAMODB_ENDPOINT`, etc.), you can collapse all of them into a single `AWS_ENDPOINT_URL=http://localhost:4566`. CloudMock serves every service on the same port.

### Lambda handler code is stored but not executed

CloudMock stores Lambda function definitions and metadata, and SDK calls that list/describe/update Lambdas work exactly as expected. What it does **not** do is spin up Docker containers to run your handler code on invocation. If your tests depend on Lambda execution (e.g., invoking a function and asserting on its side effects), you'll need to either restructure the test to call the handler directly, or wait for container-mode (tracked in the roadmap).

This trade-off is intentional: container-based Lambda execution is the single biggest source of LocalStack's startup cost, memory usage, and slow test feedback.

### Service coverage gaps

CloudMock covers 99 services vs LocalStack's ~25 on free tier. If you're migrating off LocalStack Pro, the LocalStack-exclusive services (AppSync, MWAA, etc.) may not be on the CloudMock roadmap yet. Check [cloudmock.app/docs](/docs/) for the current service list, and file an issue if you need one that isn't listed.

### Profile selection

LocalStack boots with `SERVICES=s3,dynamodb,sqs` to keep startup fast. CloudMock's equivalent is `CLOUDMOCK_PROFILE=minimal`, which pre-loads a curated subset. For tests that touch the full surface, use `CLOUDMOCK_PROFILE=full`. Profiles don't affect routing (every service is still reachable), only which services are eagerly initialized at startup.

### License

CloudMock is BSL-1.1: **free for local development, testing, and internal use**. Commercial use that involves hosting CloudMock as a service to third parties requires a license. Most teams (using CloudMock as a dev/CI dependency) are in the free tier with no action required. See [cloudmock.app/license](/license/) for details.

## Reference — concept mapping

| LocalStack | CloudMock |
|---|---|
| `docker run localstack` | `npx cloudmock` |
| `SERVICES=s3,dynamodb` | `CLOUDMOCK_PROFILE=minimal` |
| `localstack status` | `curl localhost:4599/api/services` |
| `awslocal s3 ls` | `aws --endpoint http://localhost:4566 s3 ls` |
| `tflocal` | `cloudmock-terraform` |
| `cdklocal` | `cloudmock-cdk` |
| `LOCALSTACK_API_KEY` | Not needed — all features free |
| Cloud Pods (Pro) | `--state state.json` |
| Chaos testing (Pro) | `curl -X POST localhost:4599/api/chaos` |
| CI Analytics (Pro) | DevTools at `localhost:4500` |

## Next steps

- [Verify your stack works end-to-end](/docs/getting-started/with-your-stack/)
- [Install the GitHub Action for CI](/docs/guides/ci-cd-github-actions/)
- [Explore the DevTools dashboard](/docs/devtools/overview/) — traces, metrics, chaos, profiler
- [Set up state snapshots](/docs/guides/state-snapshots/) for shared team baselines
- [Migrating from moto?](/docs/guides/migrate-from-moto/) — similar process, different starting point
