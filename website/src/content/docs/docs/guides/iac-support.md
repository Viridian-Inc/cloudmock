---
title: IaC Support
description: Auto-provision resources from Terraform, CDK, SAM, and Pulumi projects
---

# IaC Support

CloudMock can auto-provision DynamoDB tables, Lambda functions, SQS queues, SNS topics, and S3 buckets directly from your Infrastructure-as-Code source — no seed scripts needed.

## Quick Start

```bash
cloudmock --iac path/to/your/project
```

CloudMock auto-detects the IaC framework:

| Framework | Detection | File patterns |
|---|---|---|
| **Terraform** | `.tf` files present | `*.tf` |
| **SAM / CloudFormation** | `template.yaml` present | `template.yaml`, `template.yml` |
| **CDK** | `cdk.json` present | `lib/*.ts` |
| **Pulumi** | `Pulumi.yaml` present | `*.ts` |

## What gets provisioned

CloudMock parses your IaC source and creates the declared resources at startup:

- **DynamoDB tables** — full schema: hash key, range key, attributes, GSIs, LSIs
- **Lambda functions** — function name, runtime, handler, timeout, memory
- **SQS queues** — queue name, FIFO detection
- **SNS topics** — topic name
- **S3 buckets** — bucket name

Resources are re-scanned on file changes (hot reload) so your mock environment stays in sync with your IaC as you develop.

## Terraform

CloudMock parses `.tf` files for `resource` blocks:

```hcl
resource "aws_dynamodb_table" "users" {
  name         = "users-table"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"
  range_key    = "sk"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }

  global_secondary_index {
    name            = "email-index"
    hash_key        = "email"
    projection_type = "ALL"
  }
}
```

Dependency extraction:
- **Explicit**: `depends_on = [aws_dynamodb_table.users]`
- **Implicit**: references like `aws_dynamodb_table.users.arn` in other resource blocks

## CDK (TypeScript)

CloudMock parses TypeScript files importing `aws-cdk-lib`:

```typescript
const table = new dynamodb.Table(this, 'UsersTable', {
  tableName: 'users',
  partitionKey: { name: 'pk', type: dynamodb.AttributeType.STRING },
  sortKey: { name: 'sk', type: dynamodb.AttributeType.STRING },
});

const handler = new lambda.Function(this, 'ApiHandler', {
  functionName: 'api-handler',
  runtime: lambda.Runtime.NODEJS_20_X,
  handler: 'index.handler',
});
```

## SAM / CloudFormation

CloudMock parses `template.yaml`:

```yaml
Resources:
  UsersTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: users
      KeySchema:
        - AttributeName: pk
          KeyType: HASH
      AttributeDefinitions:
        - AttributeName: pk
          AttributeType: S

  ApiHandler:
    Type: AWS::Serverless::Function
    Properties:
      FunctionName: api-handler
      Runtime: nodejs20.x
      Handler: index.handler
    DependsOn: UsersTable
```

## Pulumi (TypeScript)

The original supported framework:

```typescript
const table = new aws.dynamodb.Table("users", {
  name: `users-${stack}`,
  hashKey: "pk",
  rangeKey: "sk",
  attributes: [
    { name: "pk", type: "S" },
    { name: "sk", type: "S" },
  ],
});
```

## IaC Diff View

The DevTools dashboard includes an **IaC vs Runtime** panel that compares your declared resources against what's actually provisioned:

- **Synced** — resource exists and matches IaC declaration
- **Missing** — declared in IaC but not yet provisioned
- **Orphaned** — provisioned but not in IaC (may have been created manually or via SDK)

Access it via the Tools section in the DevTools sidebar, or query the API directly:

```bash
curl http://localhost:4599/api/iac/diff | jq
```

## Environment-aware naming

Use the `--iac-env` flag to resolve environment-specific resource names:

```bash
cloudmock --iac ./infra --iac-env staging
```

This resolves template variables like `${environment}` or `${stack}` to the given value, so `users-${environment}` becomes `users-staging`.
