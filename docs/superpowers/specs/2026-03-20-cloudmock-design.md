# cloudmock — Full AWS Emulator

**Date:** 2026-03-20
**Status:** Draft
**License:** Apache 2.0

## Overview

cloudmock is a standalone, open-source local AWS emulator that provides API-compatible emulation of 100 AWS services. It runs as a Docker-based microservice mesh where each AWS service is an isolated container behind a unified gateway. 24 core services receive full functional emulation; the remaining ~74 get auto-generated CRUD-capable stubs that can be deepened over time.

cloudmock is designed to be a complete LocalStack replacement for any AWS user — not tied to any specific project or stack.

## Goals

- Full API compatibility with AWS SDKs and CLI across 100 services
- Real functional emulation for 24 core services (storage, compute, messaging, auth, infrastructure)
- API-compatible stubs for ~74 additional services with realistic responses and CRUD lifecycle
- IAM policy evaluation enforced by default (with configurable modes: enforce, authenticate, none)
- Docker-first distribution with Docker Compose orchestration
- Web dashboard for resource inspection, request logging, and IAM debugging
- CLI for service management, state control, and configuration
- Apache 2.0 open source

## Architecture

### Microservice Mesh

Each AWS service runs as its own Docker container. A gateway container receives all incoming AWS API requests and routes them to the appropriate service container over an internal Docker bridge network.

```
                    ┌─────────────────────────────────┐
                    │         cloudmock-gateway        │
                    │  :4566 (AWS API)                 │
                    │  :4500 (Dashboard)               │
                    │  :4599 (Admin API)               │
                    └──────────────┬──────────────────┘
                                   │
                    ┌──────────────┴──────────────────┐
                    │       cloudmock network          │
                    │         (Docker bridge)          │
                    ├─────────┬─────────┬─────────────┤
                    │         │         │             │
              ┌─────┴──┐ ┌───┴────┐ ┌──┴──────┐  ┌──┴──────┐
              │  iam   │ │   s3   │ │dynamodb │  │ lambda  │
              │ (gRPC+ │ │        │ │         │  │ (docker │
              │  HTTP)  │ │        │ │         │  │  socket)│
              └────────┘ └────────┘ └─────────┘  └─────────┘
                    ... (up to 98 service containers, profile-dependent)
```

### Service Profiles

Running 98 containers simultaneously is impractical on developer machines. cloudmock uses a **profile system** to control which services start:

| Profile | Services | Containers | Use Case |
|---------|----------|------------|----------|
| **minimal** (default) | Gateway, IAM, STS, S3, DynamoDB, SQS, SNS, Lambda, CloudWatch Logs | 9 | Quick local dev |
| **standard** | All Tier 1 services | ~25 | Full integration testing |
| **full** | All Tier 1 + all Tier 2 stubs | ~110 | Complete API compatibility testing |
| **custom** | User-specified via config | Varies | Project-specific needs |

```
cloudmock start                          # Starts 'minimal' profile (default)
cloudmock start --profile standard       # Starts all Tier 1 services
cloudmock start --profile full           # Starts everything
cloudmock start --services s3,lambda     # Custom set
```

Services not running return `503 Service Unavailable` with a helpful error message indicating the service is not enabled and how to enable it.

### Gateway Service

The gateway is a Go HTTP server that:

1. Receives all AWS API requests on port 4566
2. Identifies the target service by inspecting:
   - `Authorization` header (service name from credential scope)
   - `X-Amz-Target` header
   - `Host` header
   - Request path
3. Calls the IAM service (internal gRPC) to authenticate and authorize
4. Proxies authorized requests to the target service container
5. Returns the service response to the caller

The gateway also serves:
- The web dashboard on port 4500
- The admin API on port 4599 (used by the CLI)

### Service Discovery

The gateway discovers available services at startup by querying Docker for containers with a `cloudmock.service` label. Services register their supported API actions via a health check endpoint that returns service metadata.

### Internal Communication

- All containers join a `cloudmock` Docker bridge network
- Gateway routes by container hostname (e.g., `http://cloudmock-s3:8080`)
- All service containers listen on internal port 8080
- Gateway-to-IAM uses gRPC for low-latency auth checks
- Cross-service calls (e.g., S3 event notifications to Lambda) go through the gateway

### Cross-Service Interactions

Many AWS services trigger actions in other services. cloudmock handles this via an internal event bus:

**How it works:**
1. A service emits an internal event (e.g., S3 emits `ObjectCreated` after PutObject)
2. The gateway's event router matches the event to registered integrations (e.g., S3 notification → Lambda, EventBridge rule → SQS)
3. The gateway invokes the target service using a **service-linked identity** — an internal principal with permissions scoped to the specific integration (mirrors AWS service-linked roles)
4. Cross-service calls bypass IAM policy evaluation but are logged with the service-linked identity for auditability
5. Only the gateway's event router can initiate cross-service calls. Direct container-to-container requests on the Docker network are treated as normal API calls and go through full IAM enforcement — there is no "trusted internal" path between service containers.

**Supported cross-service integrations (Tier 1):**
- S3 event notifications → Lambda, SQS, SNS
- DynamoDB Streams → Lambda
- SQS → Lambda (event source mapping)
- SNS → SQS, Lambda, HTTP/S
- EventBridge → Lambda, SQS, SNS, Step Functions
- Kinesis → Lambda (event source mapping)
- CloudWatch Alarms → SNS
- API Gateway → Lambda
- Step Functions → Lambda
- CloudFormation → all Tier 1 services (resource provisioning)

**Tier 2 stubs:** Cannot be targets of cross-service integrations. If a Tier 1 service (e.g., EventBridge) has a rule targeting a Tier 2 stub, the event is logged but silently dropped. The dashboard shows a warning.

## Service Framework

### Common Interface

Every service is built on a shared Go library (`pkg/service`):

```go
type Service interface {
    Name() string
    Actions() []Action
    HandleRequest(ctx *RequestContext) (*Response, error)
}

type Action struct {
    Name      string
    Method    string
    IAMAction string
    Validator RequestValidator
}

type RequestContext struct {
    Action    string
    Region    string
    AccountID string
    Identity  *CallerIdentity
    RawRequest *http.Request
    Body      []byte
    Params    map[string]string
}
```

### Request Lifecycle

1. Gateway receives request, identifies target service and action
2. Gateway calls IAM service: `Authenticate(accessKeyId, signature, signedPayload)` → `CallerIdentity`
3. Gateway calls IAM service: `Authorize(callerIdentity, iamAction, resource, conditions)` → allow/deny
4. If denied → `AccessDeniedException` returned to caller
5. If authorized → request proxied to service container
6. Service parses request using shared library
7. Service executes action logic against local state
8. Service returns AWS-shaped response (XML or JSON per service convention)

### State Storage

- Each service container manages its own state independently
- **Ephemeral mode (default):** All state in-memory (Go maps/structs). Wiped on restart.
- **Persistent mode** (`--persist /path`): State survives restarts via mounted volume.
  - Services with embedded engines (DynamoDB/SQLite, S3/filesystem, RDS/SQLite): engine files stored on the volume. In ephemeral mode, these use temporary directories wiped on shutdown.
  - All other services: state serialized to JSON on the volume using a write-ahead approach (periodic snapshots + append log) to avoid per-write overhead.

### Error Handling

- AWS-standard error codes and shapes (e.g., `NoSuchBucket`, `ResourceNotFoundException`)
- Shared library provides error constructors for common AWS errors
- Correct HTTP status codes per AWS service conventions

## IAM Enforcement

IAM is enforced by default. Every request is authenticated and authorized. For maximum performance or simplified testing, IAM enforcement can be configured:

| Mode | Behavior | Use Case |
|------|----------|----------|
| **enforce** (default) | Full Sig V4 auth + policy evaluation on every request | Realistic AWS behavior, policy testing |
| **authenticate** | Validates credentials, returns CallerIdentity, but skips policy evaluation (all actions allowed) | Identity-aware testing without policy overhead |
| **none** | Accepts any credentials, assigns root identity, skips all checks | Maximum speed, simple integration tests |

Configured via `iam.mode` in `cloudmock.yml` or `CLOUDMOCK_IAM_MODE=enforce|authenticate|none`.

### IAM Service

The IAM service container stores users, roles, policies, groups, and identity providers. It exposes:
- Standard IAM HTTP API (for `aws iam` commands)
- Internal gRPC API (for gateway auth checks)

### Auth Flow

1. Gateway extracts AWS Signature V4 from `Authorization` header
2. `Authenticate(accessKeyId, signature, signedPayload)` → `CallerIdentity`
3. `Authorize(callerIdentity, iamAction, resource, conditions)` → allow/deny
4. Denied requests receive `AccessDeniedException`

### Policy Evaluation Engine

- Identity-based policies, resource-based policies, permission boundaries, SCPs (via Organizations)
- Policy variables: `${aws:username}`, `${aws:SourceIp}`, etc.
- Condition operators: StringEquals, ArnLike, IpAddress, etc.
- Wildcard matching on actions and resources
- Full AWS policy evaluation logic (explicit deny > explicit allow > implicit deny)

### Bootstrap

- On first start, creates root account with configurable access keys (default: `test`/`test`)
- Seed mechanism: load predefined users/roles/policies from YAML/JSON config file

### Performance

- Policy evaluation cache with short TTL, invalidated on policy mutations
- gRPC for gateway-to-IAM communication
- Connection pooling

## Tier 1 Services — Full Emulation

24 services with complete functional emulation (DynamoDB Streams is a feature of DynamoDB, not a separate service):

### Storage

| Service | Backend | Key Features |
|---------|---------|-------------|
| **S3** | Filesystem | Buckets, objects, multipart upload, versioning, lifecycle rules, presigned URLs, bucket policies, CORS, event notifications (SNS/SQS/Lambda). Supports both path-style and virtual-hosted-style URLs. |
| **DynamoDB** | SQLite | Full query/scan, key conditions, filter expressions, GSIs, LSIs, projections, TTL, streams. DynamoDB Streams is included as a feature (not a separate container) — captures item-level changes, delivers to Lambda triggers. |

### Compute

| Service | Backend | Key Features |
|---------|---------|-------------|
| **Lambda** | Docker containers | Function execution in isolated containers. Node.js, Python, Go, Java runtimes. Event sources: SQS, SNS, S3, EventBridge, API Gateway, DynamoDB Streams, Kinesis |
| **ECS** | Docker host | Task definitions, services, tasks. Runs containers on host Docker |
| **ECR** | Local registry | Docker image push/pull, repository management |

### Messaging

| Service | Backend | Key Features |
|---------|---------|-------------|
| **SQS** | In-memory | Standard and FIFO queues, visibility timeout, dead-letter queues, delay queues, long polling |
| **SNS** | In-memory | Topics, subscriptions (SQS, Lambda, HTTP/S), message filtering, fan-out |
| **EventBridge** | In-memory | Event buses, rules, pattern matching, targets (Lambda, SQS, Step Functions) |
| **Kinesis** | In-memory | Streams, shards, shard iterators, consumer groups |
| **Data Firehose** | In-memory + S3 | Delivery streams to S3 |

### Auth & Security

| Service | Backend | Key Features |
|---------|---------|-------------|
| **IAM** | In-memory | Full policy engine, users, roles, groups, policies, instance profiles |
| **STS** | In-memory | AssumeRole, GetCallerIdentity, federation tokens, session tokens |
| **Cognito** | In-memory | User pools, app clients, sign-up/sign-in, JWT tokens, hosted UI stub |
| **KMS** | In-memory | Key creation, encrypt/decrypt (software-based), key rotation, aliases |
| **Secrets Manager** | In-memory | Store/retrieve secrets, versioning, rotation scheduling |

### Infrastructure

| Service | Backend | Key Features |
|---------|---------|-------------|
| **CloudFormation** | In-memory | Template parser and resource provisioner (see CloudFormation Scope below) |
| **API Gateway** | In-memory | REST and HTTP APIs, Lambda/HTTP integration, authorizers, stages |
| **Route 53** | In-memory | Hosted zones, record sets, DNS resolution within cloudmock network |
| **RDS** | SQLite / Postgres sidecar | Instances, clusters, snapshots (metadata). Embedded SQLite or optional Postgres |
| **CloudWatch** | In-memory | Metrics ingestion, basic queries, alarms triggering SNS |
| **CloudWatch Logs** | In-memory | Log groups, streams, ingestion, filter patterns |
| **SSM** | In-memory | Parameter store (String, SecureString, StringList), basic documents |
| **SES** | In-memory | Email capture to local mailbox (viewable in dashboard), verification stubs |
| **Step Functions** | In-memory | ASL parser, state machine execution, Task/Wait/Choice/Parallel/Map states |

### CloudFormation Scope

CloudFormation is the most complex Tier 1 service. It requires understanding of every other service's resource schemas. The initial scope is deliberately limited:

**Supported in v1:**
- Intrinsic functions: Ref, Fn::Sub, Fn::Join, Fn::GetAtt, Fn::Select, Fn::Split, Fn::If, Fn::Equals, Fn::And, Fn::Or, Fn::Not
- Conditions, Mappings, Parameters, Outputs
- Dependency graph resolution (DependsOn + implicit dependencies from Ref/GetAtt)
- Create and delete stack operations
- Resource types for all Tier 1 services (e.g., `AWS::S3::Bucket`, `AWS::DynamoDB::Table`, `AWS::Lambda::Function`)

**Deferred to later versions:**
- UpdateStack (change sets, drift detection)
- Nested stacks
- Stack policies
- Custom resources (Lambda-backed)
- Rollback on failure
- Resource types for Tier 2 services

**How it discovers resource schemas:**
Each service exposes a `/internal/cloudformation-resources` endpoint that returns its supported resource types, their properties, and the API calls needed to create/delete them. This allows CloudFormation to provision resources without hardcoded knowledge of every service.

### Lambda Execution Model

Lambda executes functions in isolated Docker containers with the following behavior:

- **Warm pooling:** After a function executes, the container is kept warm for 5 minutes (configurable). Subsequent invocations reuse the warm container, mimicking AWS cold/warm start behavior.
- **Concurrency:** Default limit of 10 concurrent executions per function (configurable). Excess invocations are throttled.
- **Runtimes:** `nodejs20.x`, `python3.12`, `provided.al2023` (Go, Rust, custom). Each runtime has a corresponding `cloudmock/lambda-runtime-<name>` Docker image.
- **Function code:** Zip uploads stored in the Lambda service's state. Container image functions pull from cloudmock ECR.
- **Layers:** Supported. Layer content is mounted into the execution container at `/opt`.
- **Environment variables:** Injected into the execution container, including `AWS_LAMBDA_FUNCTION_NAME`, `AWS_REGION`, `AWS_ENDPOINT_URL` (pointing back to cloudmock gateway).
- **Timeout:** Enforced via container deadline. Default 3s, max 900s.
- **Memory:** Configurable per function. Mapped to Docker container memory limits.
- **Runtime API:** Implements the Lambda Runtime API (`/2018-06-01/runtime/`) for custom runtimes.

## Tier 2 Services — API-Compatible Stubs

~74 services with auto-generated stubs.

### Stub Generation

Stubs are auto-generated from AWS API model definitions (botocore JSON service models):
- Accept any valid API call for the service
- Return realistic success responses with generated resource IDs/ARNs
- CRUD lifecycle: create returns ID, describe returns stored resource, delete removes it, list returns all
- No business logic beyond CRUD

### Stub Capabilities

- Correct request validation (required fields, enum values, patterns)
- Correct response shapes with realistic fake data (ARNs, timestamps, UUIDs)
- Resource lifecycle (create → describe → update → delete → 404)
- Tagging support (TagResource/UntagResource/ListTagsForResource)
- Pagination (NextToken)
- Correct error codes (ResourceNotFoundException, ResourceAlreadyExistsException)

### Stub Limitations

- No business logic (Glue doesn't run ETL, Athena doesn't query)
- No cross-service integrations
- No async workflows

### Deepening Path

Stubs are generated with a clean interface. To deepen a service:
1. Replace the generated handler with custom logic
2. The request/response parsing layer stays unchanged
3. Add service-specific state management
4. Add integration tests

### Stub Image

All stub services share one Docker image: `cloudmock/stub`. The service name is passed via environment variable, and the stub engine loads the appropriate AWS API model at startup.

### Stub Service List (74 services)

Account Management, Amplify, AppConfig, Application Auto Scaling, AppSync, Athena, Auto Scaling, Backup, Batch, Bedrock, ACM, Cloud Control, CloudFront, CloudTrail, CodeArtifact, CodeBuild, CodeCommit, CodeConnections, CodeDeploy, CodePipeline, Config, Cost Explorer, DMS, DocumentDB, Elastic Beanstalk, EC2, EFS, EKS, ELB, EMR, ElastiCache, Elasticsearch, MediaConvert, EventBridge Pipes, EventBridge Scheduler, FIS, Glacier, Glue, Identity Store, IoT, IoT Data, IoT Wireless, Lake Formation, Managed Blockchain, Apache Flink, MSK, MWAA, MemoryDB, MQ, Neptune, OpenSearch, Organizations, Pinpoint, ACM PCA, Redshift, RAM, Resource Groups, Resource Groups Tagging API, Route 53 Resolver, S3 Tables, SageMaker, Serverless Application Repository, Service Discovery, Shield, SWF, SSO Admin, Support, Textract, Timestream, Transcribe, Transfer, Verified Permissions, WAF, X-Ray

Note: Organizations is a stub but the IAM policy engine supports basic SCP evaluation against a single account. Full multi-account support (multiple account IDs, cross-account assume role) is deferred.

## Docker Architecture

### Container Topology

```
cloudmock-gateway     :4566 (AWS API), :4500 (dashboard), :4599 (admin API)
cloudmock-iam         internal gRPC + HTTP
cloudmock-s3          internal HTTP, optional external port
cloudmock-dynamodb    internal HTTP, optional external port
cloudmock-lambda      internal HTTP, Docker socket mount
cloudmock-ecs         internal HTTP, Docker socket mount
cloudmock-ecr         internal HTTP, local registry
cloudmock-sqs         internal HTTP
cloudmock-sns         internal HTTP
cloudmock-eventbridge internal HTTP
cloudmock-kinesis     internal HTTP
cloudmock-firehose    internal HTTP
cloudmock-sts         internal HTTP
cloudmock-cognito     internal HTTP
cloudmock-kms         internal HTTP
cloudmock-secrets     internal HTTP
cloudmock-cfn         internal HTTP
cloudmock-apigateway  internal HTTP
cloudmock-route53     internal HTTP
cloudmock-rds         internal HTTP
cloudmock-cloudwatch  internal HTTP
cloudmock-cwlogs      internal HTTP
cloudmock-ssm         internal HTTP
cloudmock-ses         internal HTTP
cloudmock-stepfn      internal HTTP
cloudmock-rds-postgres internal HTTP (optional Postgres sidecar, started when RDS is enabled with postgres engine)
cloudmock-stub-*      internal HTTP (one per stub service, shared image)
```

### Image Strategy

- **`cloudmock/base`** — Go runtime, shared service framework, health check endpoint
- **`cloudmock/<service>`** — Extends base, service-specific binary (one per Tier 1 service)
- **`cloudmock/stub`** — Generic stub engine, parameterized by service name env var. One image, many containers.
- **`cloudmock/gateway`** — Gateway + dashboard static assets
- **`cloudmock/lambda-runtime-<name>`** — Execution images for Lambda runtimes

### Docker Socket Access

Only Lambda and ECS require `/var/run/docker.sock` mounted to spawn execution containers.

### Networking

- All containers join `cloudmock` Docker bridge network
- Internal port: 8080 for all services
- Optional per-service external ports via configuration

### Startup Sequence

1. Gateway starts, initializes admin API
2. IAM starts, creates root account, signals ready via health check
3. All other services start in parallel
4. Each service registers with gateway via health check (returns service metadata)
5. Gateway marks itself ready when IAM and all profile-required services are healthy
6. Services that fail to start are retried 3 times with exponential backoff (fixed policy), then marked degraded
7. Requests to degraded services return `503 Service Unavailable` with the error reason
8. Healthy services remain fully operational regardless of degraded services

## Configuration

### cloudmock.yml

```yaml
region: us-east-1
account_id: "000000000000"

persistence:
  enabled: false
  path: /data

iam:
  mode: enforce                 # enforce | authenticate | none
  root_access_key: test
  root_secret_key: test
  seed_file: ./iam-seed.yaml   # optional

services:
  s3:
    enabled: true
    port: 4572              # optional external port
  dynamodb:
    enabled: true
  lambda:
    enabled: true
    runtimes:
      - nodejs20.x
      - python3.12
      - provided.al2023
  # ... all services configurable
  # services not listed default to enabled

dashboard:
  enabled: true
  port: 4500

logging:
  level: info               # debug, info, warn, error
  format: json              # json, text
```

### Environment Variables

All config options can also be set via environment variables:
- `CLOUDMOCK_REGION=us-east-1`
- `CLOUDMOCK_SERVICES=s3,dynamodb,lambda` (selective enable)
- `CLOUDMOCK_PERSIST=true`
- `CLOUDMOCK_PERSIST_PATH=/data`
- `CLOUDMOCK_LOG_LEVEL=debug`
- `CLOUDMOCK_IAM_MODE=enforce` (enforce|authenticate|none)

### SDK Endpoint Configuration

Users point their AWS SDKs/CLI at cloudmock using:
- `AWS_ENDPOINT_URL=http://localhost:4566` (unified env var, supported by all AWS SDKs since late 2023)
- `--endpoint-url http://localhost:4566` (AWS CLI flag)
- Per-service endpoint overrides: `AWS_ENDPOINT_URL_S3=http://localhost:4572`

S3 supports both path-style (`http://localhost:4566/bucket/key`) and virtual-hosted-style (`http://bucket.s3.localhost:4566/key`) URLs. Path-style is the default for local development.

### Region Handling

cloudmock is single-region (configurable, default `us-east-1`). Requests arriving with a different region in the credential scope are **accepted and served** — the region in the response reflects the configured region, not the requested one. This matches LocalStack behavior and avoids SDK configuration friction. A warning is logged when a region mismatch is detected.

### Admin API Security

The admin API (port 4599) is exposed via Docker Compose port mapping as `127.0.0.1:4599:4599`, restricting host-side access to localhost only. Inside the container, the gateway binds to `0.0.0.0:4599`. For shared environments, an optional bearer token can be configured via `CLOUDMOCK_ADMIN_TOKEN`. The dashboard uses the same admin API and respects the same authentication.

## CLI

The `cloudmock` CLI is a separate Go binary that communicates with the gateway's admin API on port 4599.

```
cloudmock start                              # Start minimal profile (default)
cloudmock start --services s3,lambda,dynamodb # Start specific services
cloudmock start --config ./cloudmock.yml     # Start with config file
cloudmock stop                               # Stop all containers
cloudmock status                             # Show running services, ports, health
cloudmock reset                              # Wipe all state, restart fresh
cloudmock reset --service s3                 # Reset single service
cloudmock seed --file seed.yaml              # Load predefined resources
cloudmock logs                               # Tail all service logs
cloudmock logs --service lambda              # Tail specific service
cloudmock config                             # Show current configuration
cloudmock config set region us-west-2        # Update configuration
cloudmock version                            # Version info
```

## Dashboard

Web UI served on port 4500 by the gateway container.

### Pages

- **Service Overview** — Grid of all services with health status, request count, uptime, tier indicator
- **Resource Explorer** — Browse resources per service (S3 buckets, DynamoDB tables, Lambda functions, etc.) with detail views and JSON inspection
- **Request Log** — Real-time feed of all API calls: service, action, caller identity, latency, status code. Filterable by service, action, status, time range. Searchable.
- **IAM Debugger** — Input a principal + action + resource, see allow/deny with explanation of which policy statement matched
- **SES Mailbox** — View captured emails with sender, recipient, subject, body
- **Lambda Logs** — View function execution output, invocation history
- **CloudWatch Viewer** — Browse metrics and log groups
- **State Management** — Export/import state snapshots as JSON, reset individual or all services

### Tech Stack

Single-page application built with React. Built in its own Docker build stage (`dashboard/Dockerfile`), then static assets are copied into the `cloudmock/gateway` image via multi-stage build. No separate dashboard container — the gateway serves the assets directly. Communicates with the admin API.

## Testing

### Unit Tests

Per-service tests for individual action handlers. Standard Go `testing` package. Mocked state stores.

### Integration Tests

Use official AWS SDKs (Go, Python, Node.js) pointed at cloudmock (`--endpoint-url http://localhost:4566`). Each Tier 1 service gets a test suite exercising core CRUD and business logic operations.

### Compatibility Tests

Run subsets of AWS SDK test suites against cloudmock to validate request/response shapes match real AWS behavior.

### Smoke Tests

Docker Compose brings up all services. CLI runs `cloudmock status` to verify health. Basic CRUD operations run against each Tier 1 service.

### CI Pipeline

GitHub Actions:
1. Build all Docker images
2. Run unit tests per service (parallel)
3. Start full Docker Compose stack
4. Run smoke tests
5. Run integration tests per service (parallel)
6. Build and test CLI

## Documentation

Static site generated from markdown, served by the dashboard or built with Hugo.

### Structure

```
docs/
  getting-started.md          # Install, quickstart, basic usage
  configuration.md            # cloudmock.yml reference, env vars, CLI flags
  cli-reference.md            # All CLI commands and flags
  dashboard-guide.md          # Web UI usage guide
  architecture.md             # Internal architecture, contributing
  compatibility-matrix.md     # All services × all actions status table
  contributing.md             # How to add services, deepen stubs, run tests
  services/
    s3.md                     # Per-service: supported actions, limitations, examples
    dynamodb.md
    lambda.md
    ... (one per service)
```

### Per-Service Page Template

```markdown
# Service Name

**Tier:** 1 (Full Emulation) / 2 (Stub)
**Emulation Depth:** Full / CRUD Stub

## Supported Actions

| Action | Status | Notes |
|--------|--------|-------|
| CreateBucket | Implemented | Full support |
| PutObject | Implemented | Max 5GB |
| ... | ... | ... |

## Known Limitations

- List of differences from real AWS

## Examples

### AWS CLI
(examples)

### Python (boto3)
(examples)

### Go (aws-sdk-go-v2)
(examples)
```

## Project Structure

```
cloudmock/
  cmd/
    cloudmock/              # CLI binary
      main.go
    gateway/                # Gateway binary
      main.go
  pkg/
    service/                # Shared service framework
      service.go            # Service interface
      request.go            # Request parsing
      response.go           # Response formatting
      errors.go             # AWS error constructors
      validation.go         # Request validation
    iam/
      engine.go             # Policy evaluation engine
      types.go              # IAM types (policies, roles, etc.)
    routing/
      router.go             # AWS service detection and routing
    admin/
      api.go                # Admin API handlers
    config/
      config.go             # Configuration loading
    stub/
      generator.go          # Stub generation from API models
      engine.go             # Generic stub request handler
  services/
    s3/                     # Tier 1 service implementations
      service.go
      handlers.go
      storage.go
      Dockerfile
    dynamodb/
      service.go
      handlers.go
      store.go
      Dockerfile
    lambda/
      service.go
      handlers.go
      executor.go
      Dockerfile
    iam/
      service.go
      handlers.go
      grpc.go
      Dockerfile
    ... (one directory per Tier 1 service)
  stub/
    models/                 # AWS API model definitions (botocore JSON)
    Dockerfile              # Shared stub image
  dashboard/
    src/                    # React SPA
    public/
    package.json
    Dockerfile              # Build stage only — assets copied into gateway image
  docker-compose.yml        # Full stack composition
  cloudmock.yml             # Default configuration
  Makefile                  # Build targets
  docs/                     # Documentation
  LICENSE                   # Apache 2.0
  README.md
```

## Non-Goals

- HSM-backed encryption (KMS uses software crypto)
- Real DNS resolution outside the cloudmock network
- Multi-region (single configurable region)
- AWS billing emulation beyond Cost Explorer stubs
- Real VPC networking / security groups enforcement on containers
- Performance matching real AWS (optimized for correctness, not throughput)
