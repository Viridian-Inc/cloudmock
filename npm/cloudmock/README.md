# cloudmock

Local AWS emulation. 99 services. One command.

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

## Links

- [Documentation](https://cloudmock.io/docs)
- [GitHub](https://github.com/neureaux/cloudmock)
- [License](https://github.com/neureaux/cloudmock/blob/main/LICENSE) (Apache-2.0)
