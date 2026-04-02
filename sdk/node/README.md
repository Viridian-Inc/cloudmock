# @cloudmock/sdk

Local AWS emulation for Node.js. 98 services. One line of code.

## Install

```bash
npm install @cloudmock/sdk
```

> CloudMock is run via `npx cloudmock` — no separate global install required.

## Quick start

```ts
import { mockAWS } from "@cloudmock/sdk";
import { S3Client, CreateBucketCommand, PutObjectCommand, GetObjectCommand } from "@aws-sdk/client-s3";
import { DynamoDBClient, CreateTableCommand, PutItemCommand } from "@aws-sdk/client-dynamodb";

// Start CloudMock on a random free port
const cm = await mockAWS();

// --- S3 ---
const s3 = new S3Client(cm.clientConfig());

await s3.send(new CreateBucketCommand({ Bucket: "my-bucket" }));
await s3.send(new PutObjectCommand({
  Bucket: "my-bucket",
  Key: "hello.txt",
  Body: "world",
}));

// --- DynamoDB ---
const ddb = new DynamoDBClient(cm.clientConfig());

await ddb.send(new CreateTableCommand({
  TableName: "users",
  KeySchema: [{ AttributeName: "pk", KeyType: "HASH" }],
  AttributeDefinitions: [{ AttributeName: "pk", ScalarAttributeType: "S" }],
  BillingMode: "PAY_PER_REQUEST",
}));

await ddb.send(new PutItemCommand({
  TableName: "users",
  Item: { pk: { S: "user-1" } },
}));

// Shut down when done
await cm.stop();
```

## API

### `mockAWS(options?)` — convenience function

Starts CloudMock and returns a configured instance. Equivalent to `new CloudMock(options); await cm.start(); return cm`.

```ts
const cm = await mockAWS({ region: "eu-west-1" });
```

### `new CloudMock(options?)`

| Option | Type | Default | Description |
|---|---|---|---|
| `port` | `number` | random free port | Port for the CloudMock process |
| `region` | `string` | `"us-east-1"` | AWS region reported to SDK clients |
| `profile` | `string` | `"minimal"` | CloudMock service profile |

#### `cm.start()` → `Promise<void>`

Spawns `npx cloudmock --port <port>` and waits up to 30 s for the health endpoint to respond.

#### `cm.stop()` → `Promise<void>`

Kills the CloudMock process. Safe to call multiple times.

#### `cm.endpoint` → `string`

Base URL of the running instance, e.g. `http://localhost:4566`.

#### `cm.clientConfig()` → `AWSClientConfig`

Returns a configuration object ready to pass into any AWS SDK v3 client:

```ts
const config = cm.clientConfig();
// {
//   endpoint: "http://localhost:<port>",
//   region: "us-east-1",
//   credentials: { accessKeyId: "test", secretAccessKey: "test" },
//   forcePathStyle: true,
// }
```

## Jest / Vitest setup

Use `beforeAll` / `afterAll` to share one CloudMock instance across your entire test suite:

```ts
import { CloudMock } from "@cloudmock/sdk";
import { S3Client, CreateBucketCommand, ListBucketsCommand } from "@aws-sdk/client-s3";

let cm: CloudMock;
let s3: S3Client;

beforeAll(async () => {
  cm = new CloudMock();
  await cm.start();
  s3 = new S3Client(cm.clientConfig());
});

afterAll(async () => {
  await cm.stop();
});

test("creates a bucket", async () => {
  await s3.send(new CreateBucketCommand({ Bucket: "test-bucket" }));
  const { Buckets } = await s3.send(new ListBucketsCommand({}));
  expect(Buckets?.map((b) => b.Name)).toContain("test-bucket");
});
```

### Per-test isolation (Vitest)

```ts
import { mockAWS } from "@cloudmock/sdk";
import { DynamoDBClient, CreateTableCommand, PutItemCommand, GetItemCommand } from "@aws-sdk/client-dynamodb";

describe("DynamoDB", () => {
  let cm: Awaited<ReturnType<typeof mockAWS>>;

  beforeEach(async () => { cm = await mockAWS(); });
  afterEach(async () => { await cm.stop(); });

  it("reads back a written item", async () => {
    const ddb = new DynamoDBClient(cm.clientConfig());

    await ddb.send(new CreateTableCommand({
      TableName: "sessions",
      KeySchema: [{ AttributeName: "id", KeyType: "HASH" }],
      AttributeDefinitions: [{ AttributeName: "id", ScalarAttributeType: "S" }],
      BillingMode: "PAY_PER_REQUEST",
    }));

    await ddb.send(new PutItemCommand({
      TableName: "sessions",
      Item: { id: { S: "abc" }, ttl: { N: "9999" } },
    }));

    const { Item } = await ddb.send(new GetItemCommand({
      TableName: "sessions",
      Key: { id: { S: "abc" } },
    }));

    expect(Item?.ttl?.N).toBe("9999");
  });
});
```

## CommonJS usage

```js
const { mockAWS } = require("@cloudmock/sdk");
const { S3Client, CreateBucketCommand } = require("@aws-sdk/client-s3");

async function main() {
  const cm = await mockAWS();
  const s3 = new S3Client(cm.clientConfig());
  await s3.send(new CreateBucketCommand({ Bucket: "demo" }));
  console.log("bucket created at", cm.endpoint);
  await cm.stop();
}

main().catch(console.error);
```

## License

[BSL-1.1](../../LICENSE)
