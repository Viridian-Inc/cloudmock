---
title: Testing
description: Use CloudMock for reliable, fast integration tests with no AWS account required
---

CloudMock replaces AWS in your integration tests. Instead of mocking individual SDK calls or maintaining a shared test account, your tests talk to a local CloudMock instance that behaves like the real AWS API. Tests run faster, cost nothing, and produce deterministic results.

## Starting CloudMock in CI

### Docker

The most portable option for CI. Add CloudMock as a service in your CI pipeline:

```yaml
# GitHub Actions
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - 4566:4566
      - 4599:4599
```

```yaml
# GitLab CI
services:
  - name: ghcr.io/Viridian-Inc/cloudmock:latest
    alias: cloudmock
```

### npx

If your CI environment has Node.js, start CloudMock directly:

```bash
npx cloudmock start &
# Wait for CloudMock to be ready
until curl -s http://localhost:4599/api/health > /dev/null 2>&1; do sleep 0.5; done
```

### Docker Compose

For test suites that also need a database or other services:

```yaml
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4599:4599"
    environment:
      CLOUDMOCK_PROFILE: standard
      CLOUDMOCK_IAM_MODE: none
      CLOUDMOCK_LOG_LEVEL: warn

  test-runner:
    build: .
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      AWS_DEFAULT_REGION: us-east-1
```

## Configuring AWS SDKs

Point your AWS SDK at CloudMock using environment variables. This works across all languages without code changes:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

If your test framework sets up clients programmatically, pass the endpoint directly:

```typescript
const client = new DynamoDBClient({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
});
```

For S3, add `forcePathStyle: true` (or `UsePathStyle: true` in Go) to use path-style URLs.

## Resetting state between tests

CloudMock holds all state in memory. Between tests, reset everything to a clean slate:

```bash
curl -X POST http://localhost:4599/api/reset
```

This deletes all resources across all services (buckets, tables, queues, topics, etc.) and returns the list of services that were reset:

```json
{"reset": 5, "services": ["s3", "dynamodb", "sqs", "sns", "sts"]}
```

To reset a single service without touching others:

```bash
curl -X POST http://localhost:4599/api/services/dynamodb/reset
```

### Reset helpers

Create a utility function that runs before each test:

```typescript
// test/helpers.ts
export async function resetCloudMock() {
  await fetch("http://localhost:4599/api/reset", { method: "POST" });
}
```

```python
# tests/conftest.py
import requests
import pytest

@pytest.fixture(autouse=True)
def reset_cloudmock():
    requests.post("http://localhost:4599/api/reset")
    yield
```

```go
// test_helpers.go
func resetCloudMock(t *testing.T) {
    t.Helper()
    resp, err := http.Post("http://localhost:4599/api/reset", "", nil)
    if err != nil {
        t.Fatalf("failed to reset cloudmock: %v", err)
    }
    defer resp.Body.Close()
}
```

## Asserting on traces

After your code runs, query the traces API to verify what happened:

```bash
curl "http://localhost:4599/api/traces?service=dynamodb&limit=10"
```

Each trace contains the service, action, status code, and full request/response payloads. This lets you verify that your application made the expected AWS calls without inspecting return values.

### Trace assertions in tests

```typescript
// Verify that a PutItem was made to the Users table
const res = await fetch("http://localhost:4599/api/requests?service=dynamodb&action=PutItem");
const requests = await res.json();
const userPut = requests.find(
  (r: any) => r.request_body?.TableName === "Users"
);
expect(userPut).toBeDefined();
expect(userPut.status_code).toBe(200);
```

## Example: Jest/Vitest test with DynamoDB

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import {
  DynamoDBClient,
  CreateTableCommand,
  PutItemCommand,
  GetItemCommand,
} from "@aws-sdk/client-dynamodb";

const client = new DynamoDBClient({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
});

beforeEach(async () => {
  await fetch("http://localhost:4599/api/reset", { method: "POST" });
});

describe("user repository", () => {
  it("creates and retrieves a user", async () => {
    // Create table
    await client.send(
      new CreateTableCommand({
        TableName: "Users",
        KeySchema: [{ AttributeName: "UserId", KeyType: "HASH" }],
        AttributeDefinitions: [
          { AttributeName: "UserId", AttributeType: "S" },
        ],
        BillingMode: "PAY_PER_REQUEST",
      })
    );

    // Insert user
    await client.send(
      new PutItemCommand({
        TableName: "Users",
        Item: {
          UserId: { S: "user-1" },
          Name: { S: "Alice" },
          Email: { S: "alice@example.com" },
        },
      })
    );

    // Retrieve user
    const result = await client.send(
      new GetItemCommand({
        TableName: "Users",
        Key: { UserId: { S: "user-1" } },
      })
    );

    expect(result.Item).toBeDefined();
    expect(result.Item!.Name.S).toBe("Alice");
    expect(result.Item!.Email.S).toBe("alice@example.com");
  });

  it("returns empty for non-existent user", async () => {
    await client.send(
      new CreateTableCommand({
        TableName: "Users",
        KeySchema: [{ AttributeName: "UserId", KeyType: "HASH" }],
        AttributeDefinitions: [
          { AttributeName: "UserId", AttributeType: "S" },
        ],
        BillingMode: "PAY_PER_REQUEST",
      })
    );

    const result = await client.send(
      new GetItemCommand({
        TableName: "Users",
        Key: { UserId: { S: "does-not-exist" } },
      })
    );

    expect(result.Item).toBeUndefined();
  });
});
```

Run with:

```bash
# Start CloudMock in the background
npx cloudmock start &

# Run tests
npx vitest run

# Or with Docker
docker compose up -d cloudmock
npx vitest run
```

## Example: Go test with S3

```go
package storage_test

import (
    "bytes"
    "context"
    "io"
    "net/http"
    "testing"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func newS3Client(t *testing.T) *s3.Client {
    t.Helper()
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider("test", "test", ""),
        ),
        config.WithBaseEndpoint("http://localhost:4566"),
    )
    if err != nil {
        t.Fatalf("failed to load config: %v", err)
    }
    return s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle = true
    })
}

func resetCloudMock(t *testing.T) {
    t.Helper()
    resp, err := http.Post("http://localhost:4599/api/reset", "", nil)
    if err != nil {
        t.Fatalf("failed to reset cloudmock: %v", err)
    }
    defer resp.Body.Close()
}

func TestS3PutAndGet(t *testing.T) {
    resetCloudMock(t)
    client := newS3Client(t)
    ctx := context.TODO()

    // Create bucket
    _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String("test-bucket"),
    })
    if err != nil {
        t.Fatalf("CreateBucket: %v", err)
    }

    // Upload object
    body := []byte("hello from Go test")
    _, err = client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String("test-bucket"),
        Key:    aws.String("greeting.txt"),
        Body:   bytes.NewReader(body),
    })
    if err != nil {
        t.Fatalf("PutObject: %v", err)
    }

    // Download object
    out, err := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String("test-bucket"),
        Key:    aws.String("greeting.txt"),
    })
    if err != nil {
        t.Fatalf("GetObject: %v", err)
    }
    defer out.Body.Close()

    data, err := io.ReadAll(out.Body)
    if err != nil {
        t.Fatalf("ReadAll: %v", err)
    }

    if string(data) != "hello from Go test" {
        t.Errorf("got %q, want %q", string(data), "hello from Go test")
    }
}

func TestS3ListObjects(t *testing.T) {
    resetCloudMock(t)
    client := newS3Client(t)
    ctx := context.TODO()

    _, _ = client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String("list-bucket"),
    })

    // Upload three objects
    for _, key := range []string{"a.txt", "b.txt", "c.txt"} {
        _, err := client.PutObject(ctx, &s3.PutObjectInput{
            Bucket: aws.String("list-bucket"),
            Key:    aws.String(key),
            Body:   bytes.NewReader([]byte("content")),
        })
        if err != nil {
            t.Fatalf("PutObject %s: %v", key, err)
        }
    }

    // List objects
    out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
        Bucket: aws.String("list-bucket"),
    })
    if err != nil {
        t.Fatalf("ListObjectsV2: %v", err)
    }

    if len(out.Contents) != 3 {
        t.Errorf("got %d objects, want 3", len(out.Contents))
    }
}
```

Run with:

```bash
# Start CloudMock
npx cloudmock start &

# Run tests
go test ./... -v
```

## Tips for integration testing

**Disable IAM in tests.** Set `CLOUDMOCK_IAM_MODE=none` to skip authentication checks. This avoids the need to configure credentials in every test client.

**Use the `minimal` profile.** If your tests only use S3 and DynamoDB, the default `minimal` profile starts fast and uses less memory. Switch to `standard` or `full` only if your tests need additional services.

**Reset aggressively.** Call `POST /api/reset` in `beforeEach` / `setUp`, not `afterEach` / `tearDown`. This way each test starts clean even if the previous test crashed.

**Run CloudMock once per test suite.** Starting a new CloudMock process for every test file is slow. Start it once before the suite and reset state between tests.

**Check traces for debugging.** When a test fails, query `GET /api/requests?level=all` to see exactly what AWS API calls were made and what CloudMock returned. This is often more informative than the test assertion message.

**Seed data with the AWS SDK.** Rather than using fixtures or SQL inserts, create your test data using the same AWS SDK calls your application uses. This keeps test setup and production code aligned.
