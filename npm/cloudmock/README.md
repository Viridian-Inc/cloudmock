# cloudmock

Local AWS emulation. 100 services. One command.

## Install

```bash
npx cloudmock
```

Or install globally:

```bash
npm install -g cloudmock
cloudmock
```

## What this does

This npm package downloads the pre-built CloudMock binary for your platform
(macOS, Linux, Windows on arm64/x64) and runs it. The binary is cached at
`~/.cloudmock/bin/` so subsequent runs start instantly.

## Usage

```bash
# Start CloudMock
npx cloudmock

# Point your AWS SDK at it
aws --endpoint-url=http://localhost:4566 s3 ls
```

## Distributed Tracing

Every request is automatically traced with W3C-compatible trace and span IDs. Incoming `traceparent` headers are propagated, and responses include `traceparent`, `X-Cloudmock-Trace-Id`, and `X-Cloudmock-Span-Id` headers. View traces in the DevTools UI at `http://localhost:4500`.

## SDKs

Native SDK adapters for Go, Node.js, Python, Java, Rust, and Ruby with trace propagation and devtools integration. Any language works via HTTP.

## Infrastructure as Code

CloudMock works with your existing IaC tools — no code changes needed.

### Terraform

```bash
# Install the wrapper
go install github.com/neureaux/cloudmock/tools/cloudmock-terraform@latest

# Use your existing .tf files — they just work
cloudmock-terraform init
cloudmock-terraform plan
cloudmock-terraform apply
```

Or configure the official AWS provider manually:
```hcl
provider "aws" {
  endpoints {
    s3       = "http://localhost:4566"
    dynamodb = "http://localhost:4566"
    # ... all services use the same endpoint
  }
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
}
```

### CDK

```bash
cloudmock-cdk deploy --all
cloudmock-cdk destroy --all
```

30 CloudFormation resource types fully provisioned (S3, DynamoDB, Lambda, IAM, EC2, SQS, SNS, RDS, ECS, Route53, KMS, and more).

### Pulumi

```bash
cloudmock-pulumi up
cloudmock-pulumi destroy
```

Works with the official `@pulumi/aws` provider. Also ships a native CloudMock Pulumi provider with 44 resource types.

## Multi-Account Support

Simulate multiple AWS accounts with per-account resource isolation and cross-account STS AssumeRole:

```yaml
# cloudmock.yml
accounts:
  - id: "222222222222"
    name: "Development"
  - id: "333333333333"
    name: "Production"
```

Each account gets isolated service instances. Cross-account `sts:AssumeRole` returns credentials bound to the target account. Organizations `CreateAccount` automatically provisions new isolated accounts.

## Traffic Recording & Replay

Record real AWS traffic and replay it against CloudMock to prove your mock matches production.

```bash
# Record: proxy mode captures real AWS calls
cloudmock record --output prod-traffic.json
# (run your test suite — hits real AWS via proxy, all recorded)

# Replay: send recorded traffic to CloudMock
cloudmock replay --input prod-traffic.json

# Validate: CI-friendly comparison (exit code 0 = all match)
cloudmock validate --input prod-traffic.json
```

Go SDK interceptor for zero-config recording:
```go
recorder := sdk.NewRecorder()
cfg.HTTPClient = &http.Client{Transport: recorder.Wrap(http.DefaultTransport)}
// Use AWS SDK normally — all calls recorded
recorder.SaveToFile("recording.json")
```

## Contract Testing

Verify CloudMock fidelity against real AWS in real time. The dual-mode proxy sends every request to both endpoints, diffs the responses, and produces a compatibility report.

```bash
# Run your test suite through the contract proxy
cloudmock contract \
  --cloudmock http://localhost:4566 \
  --port 4577 \
  --ignore-paths RequestId,ResponseMetadata \
  --run "npm test"

# Outputs contract-report.json with per-service compatibility %
# Exit code 0 = 100% match, 1 = mismatches found
```

## Links

- [Documentation](https://cloudmock.io/docs)
- [GitHub](https://github.com/neureaux/cloudmock)
- [License](https://github.com/neureaux/cloudmock/blob/main/LICENSE) (Apache-2.0)
