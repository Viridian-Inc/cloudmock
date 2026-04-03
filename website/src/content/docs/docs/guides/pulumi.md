---
title: Pulumi
description: Use CloudMock with Pulumi — via the official AWS provider, the cloudmock-pulumi wrapper, or the native CloudMock provider
---

CloudMock works with Pulumi in three ways: the `cloudmock-pulumi` wrapper (simplest), the official Pulumi AWS provider with endpoint overrides (most compatible), or the native CloudMock Pulumi provider (deepest integration).

## Prerequisites

- [Pulumi CLI](https://www.pulumi.com/docs/install/) 3.0+
- CloudMock running on `localhost:4566`

## Option 1: cloudmock-pulumi wrapper

`cloudmock-pulumi` is a drop-in replacement for the `pulumi` CLI that automatically configures the AWS provider to target CloudMock and uses a local state backend:

```bash
cloudmock-pulumi new typescript    # scaffold a new TypeScript project
cloudmock-pulumi up                # deploy against CloudMock
cloudmock-pulumi preview           # preview changes
cloudmock-pulumi destroy           # tear down
```

All arguments are forwarded to `pulumi`, so `-s`, `--diff`, `--yes`, and other flags work as expected.

## Option 2: Official AWS provider with endpoint overrides

Use the `@pulumi/aws` package and configure the provider to point at CloudMock.

### TypeScript / JavaScript

```typescript
import * as aws from "@pulumi/aws";

const provider = new aws.Provider("cloudmock", {
    accessKey: "test",
    secretKey: "test",
    region: "us-east-1",
    skipCredentialsValidation: true,
    skipRequestingAccountId: true,
    endpoints: [{
        s3:             "http://localhost:4566",
        dynamodb:       "http://localhost:4566",
        sqs:            "http://localhost:4566",
        sns:            "http://localhost:4566",
        lambda:         "http://localhost:4566",
        iam:            "http://localhost:4566",
        ecs:            "http://localhost:4566",
        eks:            "http://localhost:4566",
        rds:            "http://localhost:4566",
        kms:            "http://localhost:4566",
        secretsmanager: "http://localhost:4566",
        ssm:            "http://localhost:4566",
        cloudwatch:     "http://localhost:4566",
        cloudwatchlogs: "http://localhost:4566",
    }],
});

const bucket = new aws.s3.Bucket("example-bucket", {}, { provider });
const table = new aws.dynamodb.Table("example-table", {
    attributes: [{ name: "pk", type: "S" }],
    hashKey: "pk",
    billingMode: "PAY_PER_REQUEST",
}, { provider });
const queue = new aws.sqs.Queue("example-queue", {}, { provider });

export const bucketName = bucket.id;
export const tableName = table.name;
export const queueUrl = queue.url;
```

### Using Pulumi config instead of a provider object

For projects that use the default provider, configure endpoints via `pulumi config`:

```bash
pulumi config set aws:accessKey test
pulumi config set aws:secretKey test
pulumi config set aws:region us-east-1
pulumi config set aws:skipCredentialsValidation true
pulumi config set aws:skipRequestingAccountId true
pulumi config set aws:endpoints \
  '[{"s3":"http://localhost:4566","dynamodb":"http://localhost:4566","sqs":"http://localhost:4566"}]'
```

### Python

```python
import pulumi_aws as aws

provider = aws.Provider("cloudmock",
    access_key="test",
    secret_key="test",
    region="us-east-1",
    skip_credentials_validation=True,
    skip_requesting_account_id=True,
    endpoints=[aws.ProviderEndpointArgs(
        s3="http://localhost:4566",
        dynamodb="http://localhost:4566",
        sqs="http://localhost:4566",
    )],
)

bucket = aws.s3.Bucket("example-bucket", opts=pulumi.ResourceOptions(provider=provider))
```

### Go

```go
provider, err := aws.NewProvider(ctx, "cloudmock", &aws.ProviderArgs{
    AccessKey:                  pulumi.String("test"),
    SecretKey:                  pulumi.String("test"),
    Region:                     pulumi.String("us-east-1"),
    SkipCredentialsValidation:  pulumi.Bool(true),
    SkipRequestingAccountId:    pulumi.Bool(true),
    Endpoints: aws.ProviderEndpointArray{
        &aws.ProviderEndpointArgs{
            S3:       pulumi.String("http://localhost:4566"),
            Dynamodb: pulumi.String("http://localhost:4566"),
            Sqs:      pulumi.String("http://localhost:4566"),
        },
    },
})
```

## Option 3: Native CloudMock Pulumi provider

The native CloudMock provider is pre-configured for CloudMock and does not require endpoint configuration. It exposes CloudMock-specific features like chaos controls, metric injection, and topology queries as first-class Pulumi resources.

### Install

```bash
pulumi plugin install resource cloudmock
```

Or add to your project:

```bash
npm install @cloudmock/pulumi    # TypeScript / JavaScript
pip install pulumi-cloudmock     # Python
go get github.com/neureaux/cloudmock/sdk/go/cloudmock  # Go
```

### Usage (TypeScript)

```typescript
import * as cloudmock from "@cloudmock/pulumi";

const provider = new cloudmock.Provider("local", {
    endpoint: "http://localhost:4566",
});

const bucket = new cloudmock.s3.Bucket("example-bucket", {}, { provider });
const table = new cloudmock.dynamodb.Table("example-table", {
    hashKey: "pk",
    attributes: [{ name: "pk", type: "S" }],
    billingMode: "PAY_PER_REQUEST",
}, { provider });
```

## State backends

Use a local state file for development:

```bash
pulumi login --local
```

Or use Pulumi Cloud (the default) — it works with CloudMock because the state is stored separately from AWS.

## Example project

A complete working example is in [`examples/pulumi-typescript/`](https://github.com/neureaux/cloudmock/tree/main/examples/pulumi-typescript) in the repository.

## Troubleshooting

**`error: aws:index/provider:Provider resource ... No valid credential sources found`**
Set `skipCredentialsValidation: true` on the provider.

**Resources deploy but changes are not visible**
Confirm that CloudMock is running (`curl http://localhost:4566/_cloudmock/health`) and that the endpoint for the resource's service is in the `endpoints` list.

**`pulumi up` hangs**
This is usually a network issue. Verify CloudMock is reachable at `localhost:4566`.
