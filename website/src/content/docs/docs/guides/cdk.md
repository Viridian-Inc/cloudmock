---
title: AWS CDK
description: Use CloudMock with the AWS CDK for local development and integration testing — no AWS account or CDK bootstrap required
---

CloudMock works with the AWS CDK. The `cloudmock-cdk` wrapper handles credential and endpoint configuration automatically, or you can configure the CDK manually using environment variables.

## Prerequisites

- [Node.js](https://nodejs.org/) 18+
- AWS CDK CLI: `npm install -g aws-cdk`
- CloudMock running on `localhost:4566`

## cloudmock-cdk wrapper

`cloudmock-cdk` is a drop-in replacement for the `cdk` command that pre-configures AWS credentials and endpoint overrides:

```bash
cloudmock-cdk deploy        # deploy all stacks
cloudmock-cdk deploy MyStack  # deploy a specific stack
cloudmock-cdk diff          # show pending changes
cloudmock-cdk destroy       # tear down
```

All arguments are forwarded to `cdk`, so `--require-approval never`, `--outputs-file`, and all other CDK flags work as expected.

`cloudmock-cdk` also skips the CDK bootstrap step — CloudMock provides a built-in bootstrap bucket.

## Manual configuration

Set environment variables so the AWS SDK inside the CDK CLI and your constructs point at CloudMock:

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AWS_ENDPOINT_URL=http://localhost:4566

cdk bootstrap aws://000000000000/us-east-1
cdk deploy
```

`AWS_ENDPOINT_URL` is supported by AWS SDK v2 and v3 and redirects all service calls to the specified base URL. CloudMock uses a single port for all services, so one variable covers everything.

### Account and region

CDK synthesizes templates bound to a specific AWS account and region. When using CloudMock, set the environment on the stack:

```typescript
const app = new cdk.App();
new MyStack(app, "MyStack", {
    env: {
        account: "000000000000",  // CloudMock's default account ID
        region: "us-east-1",
    },
});
```

Or use environment-agnostic stacks (no `env` property) and let CDK resolve the account and region at deploy time from the current credentials.

## Example stack

```typescript
import * as cdk from "aws-cdk-lib";
import * as s3 from "aws-cdk-lib/aws-s3";
import * as dynamodb from "aws-cdk-lib/aws-dynamodb";
import * as sqs from "aws-cdk-lib/aws-sqs";
import { Construct } from "constructs";

export class ExampleStack extends cdk.Stack {
  constructor(scope: Construct, id: string) {
    super(scope, id);

    new s3.Bucket(this, "ExampleBucket");

    new dynamodb.Table(this, "ExampleTable", {
      partitionKey: { name: "pk", type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
    });

    new sqs.Queue(this, "ExampleQueue");
  }
}
```

A complete working example is in [`examples/cdk-typescript/`](https://github.com/Viridian-Inc/cloudmock/tree/main/examples/cdk-typescript) in the repository.

## Supported constructs

Any CDK L1 (Cfn*) or L2 construct that maps to a service CloudMock supports will work. The following construct libraries are tested:

| Library | Example constructs |
|---------|-------------------|
| `aws-cdk-lib/aws-s3` | `Bucket` |
| `aws-cdk-lib/aws-dynamodb` | `Table` |
| `aws-cdk-lib/aws-sqs` | `Queue` |
| `aws-cdk-lib/aws-sns` | `Topic`, `Subscription` |
| `aws-cdk-lib/aws-lambda` | `Function` |
| `aws-cdk-lib/aws-iam` | `Role`, `Policy`, `User` |
| `aws-cdk-lib/aws-ecs` | `Cluster`, `TaskDefinition`, `FargateService` |
| `aws-cdk-lib/aws-eks` | `Cluster`, `Nodegroup` |
| `aws-cdk-lib/aws-rds` | `DatabaseInstance`, `DatabaseCluster` |
| `aws-cdk-lib/aws-route53` | `HostedZone`, `ARecord` |
| `aws-cdk-lib/aws-cloudwatch` | `Alarm`, `Metric` |
| `aws-cdk-lib/aws-logs` | `LogGroup`, `LogStream` |
| `aws-cdk-lib/aws-events` | `EventBus`, `Rule` |
| `aws-cdk-lib/aws-stepfunctions` | `StateMachine` |
| `aws-cdk-lib/aws-kms` | `Key`, `Alias` |
| `aws-cdk-lib/aws-secretsmanager` | `Secret` |
| `aws-cdk-lib/aws-ssm` | `StringParameter` |
| `aws-cdk-lib/aws-cognito` | `UserPool`, `UserPoolClient` |
| `aws-cdk-lib/aws-apigateway` | `RestApi`, `Deployment` |
| `aws-cdk-lib/aws-cloudfront` | `Distribution` |
| `aws-cdk-lib/aws-elasticache` | `CacheCluster`, `ReplicationGroup` |
| `aws-cdk-lib/aws-redshift` | `Cluster` |

## Using CDK in integration tests

CloudMock is useful for integration tests that deploy real CDK stacks:

```typescript
import { App } from "aws-cdk-lib";
import { MyStack } from "../lib/my-stack";

test("stack deploys without errors", async () => {
    process.env.AWS_ENDPOINT_URL = "http://localhost:4566";
    process.env.AWS_ACCESS_KEY_ID = "test";
    process.env.AWS_SECRET_ACCESS_KEY = "test";
    process.env.AWS_DEFAULT_REGION = "us-east-1";

    const app = new App();
    const stack = new MyStack(app, "TestStack");
    // synthesize and deploy stack, then assert on resources
});
```

See the [Testing guide](/docs/guides/testing) for full patterns including CloudMock's in-process Go integration.

## Troubleshooting

**`Unable to resolve AWS account to use`**
Set a hard-coded `env` on the stack, or set `AWS_DEFAULT_ACCOUNT=000000000000`.

**`Stack is in ROLLBACK_COMPLETE state`**
Check the CloudMock server logs (`cloudmock logs`) for the underlying error — CDK surfaces CloudFormation errors which may obscure the root cause.

**CDK bootstrap fails**
Use `cloudmock-cdk` which skips bootstrap, or pre-create the CDK toolkit bucket:

```bash
aws s3 mb s3://cdk-hnb659fds-assets-000000000000-us-east-1 \
  --endpoint-url http://localhost:4566
```
