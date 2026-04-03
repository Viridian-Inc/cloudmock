# Terraform + CloudMock Example

This example shows how to use Terraform with CloudMock to provision AWS resources locally without a real AWS account.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) 1.0+
- CloudMock running on `localhost:4566`

Start CloudMock:

```bash
cloudmock start
```

## Option 1: cloudmock-terraform wrapper (recommended)

The `cloudmock-terraform` wrapper automatically configures provider endpoints and credentials so you don't need to modify your existing Terraform files.

```bash
cloudmock-terraform init
cloudmock-terraform apply
```

`cloudmock-terraform` is a thin shim that prepends the CloudMock provider configuration and passes all arguments through to `terraform`.

## Option 2: Manual provider configuration

Copy `main.tf` from this directory and run Terraform directly. The provider block configures each AWS service endpoint to point at the local CloudMock server:

```bash
terraform init
terraform apply
```

The key settings in `provider "aws"`:

| Setting | Purpose |
|---------|---------|
| `access_key = "test"` | Any non-empty value works — CloudMock's default root credential |
| `skip_credentials_validation = true` | Prevents real AWS credential checks |
| `skip_metadata_api_check = true` | Skips EC2 instance metadata lookup |
| `endpoints { s3 = "..." }` | Redirects each service to CloudMock |

## Adding more services

Extend the `endpoints` block with any service CloudMock supports:

```hcl
endpoints {
  lambda         = "http://localhost:4566"
  ecs            = "http://localhost:4566"
  eks            = "http://localhost:4566"
  rds            = "http://localhost:4566"
  secretsmanager = "http://localhost:4566"
  ssm            = "http://localhost:4566"
  kms            = "http://localhost:4566"
  cloudwatch     = "http://localhost:4566"
}
```

## Viewing created resources

After `apply`, open the CloudMock dashboard at `http://localhost:4566/_cloudmock/` to browse all created resources.

## Teardown

```bash
terraform destroy
```
