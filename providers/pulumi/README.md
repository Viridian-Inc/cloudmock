# Pulumi Provider for cloudmock

This directory contains the scaffold for a Pulumi provider that bridges
`terraform-provider-cloudmock` to Pulumi using
[pulumi-terraform-bridge](https://github.com/pulumi/pulumi-terraform-bridge).

## Architecture

```
providers/pulumi/
├── cmd/
│   └── pulumi-resource-cloudmock/
│       └── main.go              # Provider binary entrypoint
├── provider.go                  # tfbridge configuration (resource mappings)
└── README.md
```

The provider works by wrapping the existing Terraform provider
(`providers/terraform/`) with Pulumi's tfbridge SDK. This gives you native
Pulumi resources for every Terraform resource that cloudmock supports.

## Prerequisites

- Go 1.22+
- Pulumi CLI (`brew install pulumi`)
- The following Go modules (add to `go.mod`):

```
go get github.com/pulumi/pulumi-terraform-bridge/v3@latest
go get github.com/pulumi/pulumi/sdk/v3@latest
```

## Building

1. **Uncomment the code** in `provider.go` and `cmd/pulumi-resource-cloudmock/main.go`.

2. **Add dependencies:**
   ```bash
   cd /path/to/cloudmock
   go get github.com/pulumi/pulumi-terraform-bridge/v3@latest
   go get github.com/pulumi/pulumi/sdk/v3@latest
   go mod tidy
   ```

3. **Build the provider binary:**
   ```bash
   go build -o bin/pulumi-resource-cloudmock ./providers/pulumi/cmd/pulumi-resource-cloudmock/
   ```

4. **Install locally:**
   ```bash
   cp bin/pulumi-resource-cloudmock ~/.pulumi/bin/
   ```

## Generating SDKs

Once the provider binary is built, generate language-specific SDKs:

```bash
# Generate Node.js SDK
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language nodejs

# Generate Python SDK
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language python

# Generate Go SDK
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language go

# Generate .NET SDK
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language dotnet
```

## Usage

### TypeScript

```typescript
import * as cloudmock from "@neureaux/pulumi-cloudmock";

const bucket = new cloudmock.s3.Bucket("my-bucket", {
    bucket: "my-test-bucket",
});

const table = new cloudmock.dynamodb.Table("my-table", {
    name: "users",
    hashKey: "id",
    attributes: [{ name: "id", type: "S" }],
});
```

### Python

```python
import neureaux_pulumi_cloudmock as cloudmock

bucket = cloudmock.s3.Bucket("my-bucket", bucket="my-test-bucket")
table = cloudmock.dynamodb.Table("my-table", name="users", hash_key="id")
```

### Go

```go
import "github.com/neureaux/cloudmock/providers/pulumi/sdk/go/cloudmock/s3"

bucket, _ := s3.NewBucket(ctx, "my-bucket", &s3.BucketArgs{
    Bucket: pulumi.String("my-test-bucket"),
})
```

## Configuration

| Property     | Env Var              | Default                 |
|-------------|----------------------|-------------------------|
| `endpoint`  | `CLOUDMOCK_ENDPOINT` | `http://localhost:4566` |
| `region`    | `AWS_REGION`         | `us-east-1`             |
| `accessKey` | —                    | `test`                  |
| `secretKey` | —                    | `test`                  |

```bash
pulumi config set cloudmock:endpoint http://localhost:4566
```

## Resource Mapping

Resources are organized by AWS service module:

| Terraform Type              | Pulumi Type                |
|----------------------------|----------------------------|
| `cloudmock_s3_bucket`      | `cloudmock:s3:Bucket`      |
| `cloudmock_dynamodb_table` | `cloudmock:dynamodb:Table`  |
| `cloudmock_vpc`            | `cloudmock:ec2:Vpc`        |
| `cloudmock_lambda_function`| `cloudmock:lambda:Function` |
| `cloudmock_sqs_queue`      | `cloudmock:sqs:Queue`      |

Any resources added to the Terraform provider are automatically available
in Pulumi through the dynamic mapping in `provider.go`.
