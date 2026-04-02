---
title: Node.js
description: Using CloudMock with Node.js, the @cloudmock/sdk SDK, and AWS SDK v3
---

Node.js is a first-class language for CloudMock. The `@cloudmock/sdk` package provides automatic request interception, trace propagation, and a `CloudMock` class that handles server lifecycle for tests. For AWS-only usage, you can also configure the AWS SDK v3 directly.

## @cloudmock/sdk

### Install

```bash
npm install --save-dev @cloudmock/sdk
```

### Testing with Jest

The `CloudMock` class manages lifecycle and provides `clientConfig()` — a pre-configured options object you pass directly to any AWS SDK v3 client.

```javascript
const { CloudMock } = require("@cloudmock/sdk");
const { S3Client, CreateBucketCommand, PutObjectCommand, GetObjectCommand } = require("@aws-sdk/client-s3");
const { DynamoDBClient, CreateTableCommand, PutItemCommand, GetItemCommand } = require("@aws-sdk/client-dynamodb");
const { SQSClient, CreateQueueCommand, SendMessageCommand, ReceiveMessageCommand } = require("@aws-sdk/client-sqs");

let cm;

beforeAll(async () => {
    cm = new CloudMock();
    await cm.start();
});

afterAll(async () => {
    await cm.stop();
});

beforeEach(async () => {
    await cm.reset();
});

test("S3 upload and download", async () => {
    const s3 = new S3Client(cm.clientConfig());
    await s3.send(new CreateBucketCommand({ Bucket: "test" }));
    await s3.send(new PutObjectCommand({
        Bucket: "test",
        Key: "hello.txt",
        Body: "world",
    }));
    const out = await s3.send(new GetObjectCommand({ Bucket: "test", Key: "hello.txt" }));
    // out.Body is a readable stream -- consume and compare
});

test("DynamoDB CRUD", async () => {
    const ddb = new DynamoDBClient(cm.clientConfig());
    await ddb.send(new CreateTableCommand({
        TableName: "users",
        KeySchema: [{ AttributeName: "pk", KeyType: "HASH" }],
        AttributeDefinitions: [{ AttributeName: "pk", AttributeType: "S" }],
        BillingMode: "PAY_PER_REQUEST",
    }));
    await ddb.send(new PutItemCommand({
        TableName: "users",
        Item: { pk: { S: "user-1" }, name: { S: "Alice" } },
    }));
    const { Item } = await ddb.send(new GetItemCommand({
        TableName: "users",
        Key: { pk: { S: "user-1" } },
    }));
    expect(Item.name.S).toBe("Alice");
});

test("SQS send and receive", async () => {
    const sqs = new SQSClient(cm.clientConfig());
    const { QueueUrl } = await sqs.send(new CreateQueueCommand({ QueueName: "tasks" }));
    await sqs.send(new SendMessageCommand({ QueueUrl, MessageBody: "do-something" }));
    const { Messages } = await sqs.send(new ReceiveMessageCommand({ QueueUrl }));
    expect(Messages[0].Body).toBe("do-something");
});
```

### Testing with Vitest

The same pattern works with Vitest:

```typescript
import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { CloudMock } from "@cloudmock/sdk";
import {
    DynamoDBClient,
    CreateTableCommand,
    PutItemCommand,
    GetItemCommand,
} from "@aws-sdk/client-dynamodb";

let cm: CloudMock;

beforeAll(async () => {
    cm = new CloudMock();
    await cm.start();
});

afterAll(async () => await cm.stop());
beforeEach(async () => await cm.reset());

describe("user repository", () => {
    it("creates and retrieves a user", async () => {
        const client = new DynamoDBClient(cm.clientConfig());

        await client.send(new CreateTableCommand({
            TableName: "Users",
            KeySchema: [{ AttributeName: "UserId", KeyType: "HASH" }],
            AttributeDefinitions: [{ AttributeName: "UserId", AttributeType: "S" }],
            BillingMode: "PAY_PER_REQUEST",
        }));

        await client.send(new PutItemCommand({
            TableName: "Users",
            Item: {
                UserId: { S: "user-1" },
                Name: { S: "Alice" },
                Email: { S: "alice@example.com" },
            },
        }));

        const result = await client.send(new GetItemCommand({
            TableName: "Users",
            Key: { UserId: { S: "user-1" } },
        }));

        expect(result.Item?.Name?.S).toBe("Alice");
        expect(result.Item?.Email?.S).toBe("alice@example.com");
    });
});
```

### Application instrumentation

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

If you do not want to use the `@cloudmock/sdk` package (for example, in a Lambda function or a script that only calls AWS), you can point the AWS SDK v3 directly at CloudMock:

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

Both the `@cloudmock/sdk` package and the AWS SDK v3 are fully typed. The `@cloudmock/sdk` package ships with TypeScript declarations.

## Common issues

### S3 virtual-hosted style URLs

CloudMock does not support virtual-hosted style S3 URLs (`bucket.localhost:4566`). Always set `forcePathStyle: true` on the S3 client.

### CORS in browser-side code

If calling CloudMock from a browser-based application, the gateway returns CORS headers by default. No additional configuration is needed.
