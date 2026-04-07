---
title: Terraform
description: Use CloudMock with Terraform to develop and test infrastructure locally — no AWS account required
---

CloudMock works with Terraform's official AWS provider. You can point any Terraform configuration at CloudMock by overriding the provider endpoints, or use the `cloudmock-terraform` wrapper that handles this automatically.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) 1.0+
- CloudMock running on `localhost:4566`

## cloudmock-terraform wrapper

The `cloudmock-terraform` wrapper is the quickest way to use an existing Terraform project with CloudMock. It injects the endpoint overrides and dummy credentials transparently:

```bash
cloudmock-terraform init
cloudmock-terraform plan
cloudmock-terraform apply
```

All arguments are forwarded to `terraform`, so you can pass `-var`, `-target`, `-auto-approve`, and any other flags you normally use.

## Manual provider configuration

If you prefer to configure the provider yourself, add the following block to your Terraform configuration:

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  access_key                  = "test"
  secret_key                  = "test"
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    s3             = "http://localhost:4566"
    dynamodb       = "http://localhost:4566"
    sqs            = "http://localhost:4566"
    sns            = "http://localhost:4566"
    lambda         = "http://localhost:4566"
    iam            = "http://localhost:4566"
    ecs            = "http://localhost:4566"
    eks            = "http://localhost:4566"
    rds            = "http://localhost:4566"
    secretsmanager = "http://localhost:4566"
    ssm            = "http://localhost:4566"
    kms            = "http://localhost:4566"
    cloudwatch     = "http://localhost:4566"
    cloudwatchlogs = "http://localhost:4566"
    route53        = "http://localhost:4566"
    apigateway     = "http://localhost:4566"
    cloudfront     = "http://localhost:4566"
    elasticache    = "http://localhost:4566"
    redshift       = "http://localhost:4566"
  }
}
```

All services share the same port. Add only the services your configuration uses.

### Key settings

| Setting | Why it's needed |
|---------|----------------|
| `access_key = "test"` | CloudMock's default root credential; any non-empty value works |
| `skip_credentials_validation` | Prevents the provider from calling real AWS to validate credentials |
| `skip_metadata_api_check` | Skips EC2 instance metadata lookup at provider initialization |
| `skip_requesting_account_id` | Prevents an STS `GetCallerIdentity` call during initialization |
| `endpoints { ... }` | Redirects each service client to CloudMock |

## Example

```hcl
resource "aws_s3_bucket" "example" {
  bucket = "my-example-bucket"
}

resource "aws_dynamodb_table" "example" {
  name         = "my-example-table"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"

  attribute {
    name = "pk"
    type = "S"
  }
}

resource "aws_sqs_queue" "example" {
  name = "my-example-queue"
}
```

A complete working example is in [`examples/terraform/`](https://github.com/Viridian-Inc/cloudmock/tree/main/examples/terraform) in the repository.

## State management

Terraform state works normally — use local state for quick experiments or any remote backend that does not require AWS credentials (e.g., Terraform Cloud, a local file, or an S3 backend pointed at CloudMock itself).

To use CloudMock's S3 as a Terraform backend:

```hcl
terraform {
  backend "s3" {
    bucket   = "terraform-state"
    key      = "my-project/terraform.tfstate"
    region   = "us-east-1"
    endpoint = "http://localhost:4566"

    access_key                  = "test"
    secret_key                  = "test"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    force_path_style            = true
  }
}
```

## Supported resources

CloudMock supports the most commonly used resource types. The following Terraform resource types are tested and known to work:

- `aws_s3_bucket`, `aws_s3_object`
- `aws_dynamodb_table`
- `aws_sqs_queue`
- `aws_sns_topic`, `aws_sns_topic_subscription`
- `aws_lambda_function`, `aws_lambda_event_source_mapping`
- `aws_iam_role`, `aws_iam_policy`, `aws_iam_user`
- `aws_ecs_cluster`, `aws_ecs_task_definition`, `aws_ecs_service`
- `aws_eks_cluster`, `aws_eks_node_group`
- `aws_db_instance`, `aws_rds_cluster`
- `aws_route53_zone`, `aws_route53_record`
- `aws_cloudwatch_metric_alarm`
- `aws_cloudwatch_log_group`, `aws_cloudwatch_log_stream`
- `aws_cloudwatch_event_bus`, `aws_cloudwatch_event_rule`
- `aws_sfn_state_machine`
- `aws_kms_key`, `aws_kms_alias`
- `aws_secretsmanager_secret`, `aws_secretsmanager_secret_version`
- `aws_ssm_parameter`
- `aws_cognito_user_pool`, `aws_cognito_user_pool_client`
- `aws_api_gateway_rest_api`, `aws_api_gateway_deployment`
- `aws_cloudfront_distribution`
- `aws_elasticache_cluster`, `aws_elasticache_replication_group`
- `aws_redshift_cluster`

## Troubleshooting

**`Error: error configuring Terraform AWS Provider: no valid credential sources found`**
Make sure `skip_credentials_validation = true` is set in the provider block.

**`Error: Post "https://dynamodb.us-east-1.amazonaws.com/": dial tcp`**
The `endpoints` block for that service is missing or CloudMock is not running. Verify with `curl http://localhost:4566/_cloudmock/health`.

**Resources appear to apply but do not show in CloudMock**
Check that the endpoint for the specific resource type is listed in the `endpoints` block.
