---
title: Node.js
description: Using CloudMock with Node.js, the @cloudmock/node SDK, and AWS SDK v3
---

Node.js is a first-class language for CloudMock. The `@cloudmock/node` SDK provides automatic request interception, trace propagation, and devtools integration. For AWS-only usage, you can also configure the AWS SDK v3 directly.

## @cloudmock/node SDK

### Install

```bash
npm install @cloudmock/node
```

### Initialize

Call `init()` early in your application startup, before creating AWS SDK clients. This configures the SDK to intercept outgoing AWS API calls and forward telemetry to the CloudMock admin API.

```typescript
import { init } from '@cloudmock/node';

init({
  adminUrl: 'http://localhost:4599',  // CloudMock admin API
  serviceName: 'my-api',              // Identifies this service in the topology
});
```

### Express middleware

The SDK provides Express middleware that automatically traces inbound HTTP requests and correlates them with outbound AWS calls:

```typescript
import express from 'express';
import { init, expressMiddleware } from '@cloudmock/node';

init({ adminUrl: 'http://localhost:4599', serviceName: 'my-api' });

const app = express();
app.use(expressMiddleware());

app.get('/users', async (req, res) => {
  // AWS calls made here are automatically traced and visible in the devtools
  const result = await dynamodb.send(new ScanCommand({ TableName: 'Users' }));
  res.json(result.Items);
});
```

### What gets captured

When the SDK is active, the following data is sent to the CloudMock admin API:

- **Inbound requests** -- Method, path, headers, status code, and response time for every HTTP request your service handles.
- **Outbound AWS calls** -- Service, action, latency, status code, and request/response bodies for every AWS SDK call.
- **Trace context** -- A trace ID is generated for each inbound request and propagated to all outbound calls, enabling end-to-end tracing in the devtools.
- **Service identity** -- The `serviceName` you provide appears as a node in the Topology view.

## AWS SDK v3 endpoint configuration

If you do not want to use the `@cloudmock/node` SDK (for example, in a Lambda function or a script that only calls AWS), you can point the AWS SDK v3 directly at CloudMock:

### Environment variable (recommended)

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

The `AWS_ENDPOINT_URL` environment variable is supported by AWS SDK v3 since version 3.400+.

### Per-client configuration

```typescript
import { S3Client, CreateBucketCommand } from '@aws-sdk/client-s3';
import { DynamoDBClient, PutItemCommand } from '@aws-sdk/client-dynamodb';

const s3 = new S3Client({
  region: 'us-east-1',
  endpoint: 'http://localhost:4566',
  credentials: {
    accessKeyId: 'test',
    secretAccessKey: 'test',
  },
  forcePathStyle: true,  // Required for S3 path-style access
});

const dynamodb = new DynamoDBClient({
  region: 'us-east-1',
  endpoint: 'http://localhost:4566',
  credentials: {
    accessKeyId: 'test',
    secretAccessKey: 'test',
  },
});

// Use as normal
await s3.send(new CreateBucketCommand({ Bucket: 'my-bucket' }));
await dynamodb.send(new PutItemCommand({
  TableName: 'Users',
  Item: { UserId: { S: 'user-1' }, Name: { S: 'Alice' } },
}));
```

### Conditional endpoint (dev vs. production)

A common pattern is to set the endpoint only in development:

```typescript
const clientConfig = {
  region: process.env.AWS_REGION || 'us-east-1',
  ...(process.env.CLOUDMOCK_ENDPOINT && {
    endpoint: process.env.CLOUDMOCK_ENDPOINT,
    credentials: {
      accessKeyId: 'test',
      secretAccessKey: 'test',
    },
  }),
};

const s3 = new S3Client({ ...clientConfig, forcePathStyle: true });
const dynamodb = new DynamoDBClient(clientConfig);
```

Then in your `.env`:

```bash
CLOUDMOCK_ENDPOINT=http://localhost:4566
```

## TypeScript support

Both the `@cloudmock/node` SDK and the AWS SDK v3 are fully typed. The `@cloudmock/node` package ships with TypeScript declarations.

## Common issues

### S3 virtual-hosted style URLs

CloudMock does not support virtual-hosted style S3 URLs (`bucket.localhost:4566`). Always set `forcePathStyle: true` on the S3 client.

### CORS in browser-side code

If calling CloudMock from a browser-based application, the gateway returns CORS headers by default. No additional configuration is needed.
