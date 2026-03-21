# Crossplane Provider for cloudmock

This directory contains the scaffold for generating a Crossplane provider from
`terraform-provider-cloudmock` using [Upjet](https://github.com/crossplane/upjet).

## Architecture

There are two approaches for using Crossplane with cloudmock:

### Option A: Generate a native `provider-cloudmock` (this directory)

Uses Upjet to generate a full Crossplane provider from the Terraform provider.
This gives you `Managed Resources` with types like `Bucket`, `Table`, `Vpc`.

### Option B: Use `provider-aws` with cloudmock endpoint (crossplane-aws-config/)

Points the official Crossplane AWS provider at a cloudmock gateway. This is
simpler and works today without any code generation. See
`providers/crossplane-aws-config/` for the Helm chart and ProviderConfig.

## Generating with Upjet (Option A)

### Prerequisites

- Go 1.22+
- [Upjet](https://github.com/crossplane/upjet) CLI
- Docker (for building provider images)

### Steps

1. **Initialize the Upjet project:**
   ```bash
   upjet generate \
     --terraform-provider-source neureaux/cloudmock \
     --terraform-provider-version 0.1.0 \
     --root-group cloudmock.neureaux.io \
     --output ./generated
   ```

2. **Apply the provider metadata** from `config/provider-metadata.yaml`.

3. **Build the provider:**
   ```bash
   cd generated
   make generate
   make build
   ```

4. **Build the Docker image:**
   ```bash
   docker build -t ghcr.io/neureaux/provider-cloudmock:v0.1.0 .
   ```

5. **Install in a Crossplane cluster:**
   ```yaml
   apiVersion: pkg.crossplane.io/v1
   kind: Provider
   metadata:
     name: provider-cloudmock
   spec:
     package: ghcr.io/neureaux/provider-cloudmock:v0.1.0
   ```

## Resource Groups

The generated provider will organize resources under these API groups:

| Terraform Type              | Crossplane Group                    | Kind           |
|----------------------------|-------------------------------------|----------------|
| `cloudmock_s3_bucket`      | `s3.cloudmock.neureaux.io`          | `Bucket`       |
| `cloudmock_dynamodb_table` | `dynamodb.cloudmock.neureaux.io`    | `Table`        |
| `cloudmock_vpc`            | `ec2.cloudmock.neureaux.io`         | `VPC`          |
| `cloudmock_subnet`         | `ec2.cloudmock.neureaux.io`         | `Subnet`       |
| `cloudmock_security_group` | `ec2.cloudmock.neureaux.io`         | `SecurityGroup`|
| `cloudmock_sqs_queue`      | `sqs.cloudmock.neureaux.io`         | `Queue`        |
| `cloudmock_lambda_function`| `lambda.cloudmock.neureaux.io`      | `Function`     |

## Usage Example

```yaml
apiVersion: s3.cloudmock.neureaux.io/v1alpha1
kind: Bucket
metadata:
  name: my-test-bucket
spec:
  forProvider:
    bucket: my-test-bucket
    region: us-east-1
  providerConfigRef:
    name: cloudmock
```
