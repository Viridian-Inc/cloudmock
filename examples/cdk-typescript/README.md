# AWS CDK (TypeScript) + CloudMock Example

This example shows how to use the AWS CDK against CloudMock for local development and testing.

## Prerequisites

- [Node.js](https://nodejs.org/) 18+
- [AWS CDK CLI](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html): `npm install -g aws-cdk`
- CloudMock running on `localhost:4566`

Start CloudMock:

```bash
cloudmock start
```

## Option 1: cloudmock-cdk wrapper (recommended)

The `cloudmock-cdk` wrapper configures CDK's AWS credentials and endpoint overrides automatically:

```bash
cloudmock-cdk deploy
```

This is equivalent to running `cdk deploy` with the correct environment variables set for CloudMock.

## Option 2: Manual CDK configuration

Set environment variables so the AWS SDK inside CDK points at CloudMock, then deploy normally:

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AWS_ENDPOINT_URL=http://localhost:4566

npm install
cdk bootstrap aws://000000000000/us-east-1 --endpoint-url http://localhost:4566
cdk deploy
```

The `AWS_ENDPOINT_URL` variable is supported by AWS SDK v2 and v3 and redirects all service calls to the specified base URL. CloudMock uses a single port for all services.

## Stack contents

`lib/stack.ts` defines an example stack with:

- An S3 bucket
- A DynamoDB table (on-demand billing, string partition key `pk`)
- An SQS queue

Extend it with any CDK constructs backed by services CloudMock supports.

## Viewing created resources

After deployment, open the CloudMock dashboard at `http://localhost:4566/_cloudmock/` to browse all created resources.

## Teardown

```bash
cloudmock-cdk destroy
# or
cdk destroy
```
