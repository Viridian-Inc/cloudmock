# Pulumi (TypeScript) + CloudMock Example

This example shows how to use Pulumi with the official AWS provider against CloudMock.

## Prerequisites

- [Pulumi CLI](https://www.pulumi.com/docs/install/) 3.0+
- [Node.js](https://nodejs.org/) 18+
- CloudMock running on `localhost:4566`

Start CloudMock:

```bash
cloudmock start
```

## Option 1: cloudmock-pulumi wrapper (recommended)

The `cloudmock-pulumi` wrapper sets up Pulumi config to point AWS SDK calls at CloudMock and handles the local backend automatically.

```bash
cloudmock-pulumi new typescript
cloudmock-pulumi up
```

## Option 2: Official AWS provider with manual config

Install dependencies and configure Pulumi to target CloudMock endpoints:

```bash
npm install
pulumi login --local
pulumi stack init dev

# Point Pulumi's AWS provider at CloudMock
pulumi config set aws:accessKey test
pulumi config set aws:secretKey test
pulumi config set aws:region us-east-1
pulumi config set aws:skipCredentialsValidation true
pulumi config set aws:skipRequestingAccountId true
pulumi config set aws:endpoints '[{"s3":"http://localhost:4566","dynamodb":"http://localhost:4566","sqs":"http://localhost:4566","sns":"http://localhost:4566","lambda":"http://localhost:4566"}]'

pulumi up
```

## Option 3: Native CloudMock Pulumi provider

CloudMock ships a native Pulumi provider that requires no endpoint configuration:

```bash
pulumi plugin install resource cloudmock
pulumi config set cloudmock:endpoint http://localhost:4566

# Use cloudmock resources directly in your program
import * as cloudmock from "@cloudmock/pulumi";
const bucket = new cloudmock.s3.Bucket("example-bucket");
```

See the [Pulumi guide](/docs/guides/pulumi) for full native provider documentation.

## Viewing created resources

After `pulumi up`, open the CloudMock dashboard at `http://localhost:4566/_cloudmock/` to browse all created resources.

## Teardown

```bash
pulumi destroy
```
