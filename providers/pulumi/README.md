# Pulumi Provider for cloudmock

A native Pulumi provider for cloudmock that implements the gRPC ResourceProvider
protocol directly, without requiring `pulumi-terraform-bridge`.

## Architecture

```
providers/pulumi/
├── cmd/
│   └── pulumi-resource-cloudmock/
│       └── main.go                  # Provider binary entrypoint
├── internal/
│   ├── provider.go                  # Pulumi provider protocol implementation
│   ├── pulumirpc.go                 # Hand-written gRPC service definitions
│   ├── schema_gen.go                # Pulumi schema generation from registry
│   ├── crud.go                      # HTTP client for cloudmock gateway
│   ├── codec.go                     # Hybrid proto/JSON codec
│   └── provider_test.go             # Tests (22 tests)
├── provider.go                      # Resource token mappings (reference)
├── schema.json                      # Generated Pulumi package schema
└── README.md
```

The provider communicates with cloudmock's gateway using the same HTTP client
approach as the Terraform provider, supporting JSON, query, and REST protocols.

## Building

```bash
go build -o bin/pulumi-resource-cloudmock ./providers/pulumi/cmd/pulumi-resource-cloudmock/
```

## Installing

```bash
cp bin/pulumi-resource-cloudmock ~/.pulumi/bin/
```

## Usage

### TypeScript

```typescript
import * as cloudmock from "@neureaux/pulumi-cloudmock";

const bucket = new cloudmock.s3.Bucket("my-bucket", {
    bucket: "my-test-bucket",
});

const table = new cloudmock.dynamodb.Table("my-table", {
    tableName: "users",
    hashKey: "id",
    attribute: ["id"],
});
```

### Python

```python
import neureaux_pulumi_cloudmock as cloudmock

bucket = cloudmock.s3.Bucket("my-bucket", bucket="my-test-bucket")
table = cloudmock.dynamodb.Table("my-table", table_name="users", hash_key="id")
```

### Go

```go
import "github.com/Viridian-Inc/cloudmock/providers/pulumi/sdk/go/cloudmock/s3"

bucket, _ := s3.NewBucket(ctx, "my-bucket", &s3.BucketArgs{
    Bucket: pulumi.String("my-test-bucket"),
})
```

## Configuration

| Property     | Env Var              | Default                 |
|-------------|----------------------|-------------------------|
| `endpoint`  | `CLOUDMOCK_ENDPOINT` | `http://localhost:4566` |
| `region`    | `AWS_REGION`         | `us-east-1`             |
| `accessKey` | --                   | `test`                  |
| `secretKey` | --                   | `test`                  |

```bash
pulumi config set cloudmock:endpoint http://localhost:4566
```

## Resource Mapping

Resources are organized by AWS service module using the token format
`cloudmock:<service>:<Resource>`:

| Terraform Type              | Pulumi Type                |
|----------------------------|----------------------------|
| `cloudmock_s3_bucket`      | `cloudmock:s3:Bucket`      |
| `cloudmock_dynamodb_table` | `cloudmock:dynamodb:Table`  |
| `cloudmock_ec2_vpc`        | `cloudmock:ec2:Vpc`        |
| `cloudmock_ec2_subnet`     | `cloudmock:ec2:Subnet`     |
| `cloudmock_lambda_function`| `cloudmock:lambda:Function` |
| `cloudmock_sqs_queue`      | `cloudmock:sqs:Queue`      |

New resources added to the schema registry are automatically available.

## Generating SDKs

Once the provider binary is built, generate language-specific SDKs:

```bash
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language nodejs
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language python
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language go
pulumi package gen-sdk bin/pulumi-resource-cloudmock --language dotnet
```

## Regenerating schema.json

```bash
WRITE_SCHEMA=1 go test ./providers/pulumi/internal/ -run TestGenerateSchemaJSON_WriteFile -v
```

## Testing

```bash
go test ./providers/pulumi/internal/ -v
```

## Design Notes

This provider implements Pulumi's gRPC `ResourceProvider` protocol natively
rather than using `pulumi-terraform-bridge`. This avoids the bridge's heavy
dependency tree while still providing full CRUD resource management.

Key design decisions:
- **Hand-written gRPC types**: The `pulumirpc.go` file contains Go struct
  definitions that match Pulumi's protobuf service, avoiding the need for
  protoc code generation or the full Pulumi SDK.
- **Schema generation**: The Pulumi package schema is generated dynamically
  from the cloudmock schema registry, so new resources are automatically
  picked up.
- **Shared CRUD layer**: The HTTP client in `crud.go` uses the same protocol
  dispatch (JSON/query/REST) as the Terraform provider.
