---
title: Testing
description: Use CloudMock for reliable, fast integration tests with no AWS account required — in-process (Go) or HTTP mode (any language)
---

CloudMock replaces AWS in your integration tests. Instead of mocking individual SDK calls or maintaining a shared test account, your tests talk to a local CloudMock instance that behaves like the real AWS API. Tests run faster, cost nothing, and produce deterministic results.

## Two modes

### HTTP mode (any language)

Your tests start CloudMock as a local HTTP server on port 4566 and point AWS SDK clients at it. This works with every AWS SDK — Node.js, Python, Go, Java, Rust, Ruby, Kotlin, Swift, and more. HTTP overhead is minimal (typically 1–5 ms per operation on localhost).

### In-process mode (Go only, ~20 μs/op)

For Go projects, the `github.com/Viridian-Inc/cloudmock/sdk` package embeds the CloudMock engine directly in your test binary. There is no HTTP server, no network round-trip, and no process to start. Operations run at ~20 μs each — over 110x faster than Moto and faster than any HTTP-based alternative. Ideal for large test suites or CI environments where startup time matters.

---

## Go: in-process mode

Add the SDK:

```bash
go get github.com/Viridian-Inc/cloudmock/sdk
```

Use `sdk.New()` in `TestMain` to start a shared instance for the entire test binary, then call `cm.Config()` to get an `aws.Config` pre-configured to talk to the embedded engine.

```go
package myapp_test

import (
    "context"
    "os"
    "strings"
    "testing"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/Viridian-Inc/cloudmock/sdk"
)

var cm *sdk.CloudMock

func TestMain(m *testing.M) {
    cm = sdk.New()
    defer cm.Close()
    os.Exit(m.Run())
}

func TestCreateBucketAndUpload(t *testing.T) {
    client := s3.NewFromConfig(cm.Config(), func(o *s3.Options) {
        o.UsePathStyle = true
    })

    _, err := client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
        Bucket: aws.String("test-bucket"),
    })
    if err != nil {
        t.Fatal(err)
    }

    _, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
        Bucket: aws.String("test-bucket"),
        Key:    aws.String("hello.txt"),
        Body:   strings.NewReader("world"),
    })
    if err != nil {
        t.Fatal(err)
    }
}

func TestDynamoDBCRUD(t *testing.T) {
    client := dynamodb.NewFromConfig(cm.Config())
    ctx := context.TODO()

    // Create table
    client.CreateTable(ctx, &dynamodb.CreateTableInput{
        TableName: aws.String("users"),
        KeySchema: []types.KeySchemaElement{
            {AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash},
        },
        AttributeDefinitions: []types.AttributeDefinition{
            {AttributeName: aws.String("pk"), ScalarAttributeType: types.ScalarAttributeTypeS},
        },
        BillingMode: types.BillingModePayPerRequest,
    })

    // Put item
    client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String("users"),
        Item: map[string]types.AttributeValue{
            "pk":   &types.AttributeValueMemberS{Value: "user-1"},
            "name": &types.AttributeValueMemberS{Value: "Alice"},
        },
    })

    // Get item and assert
    out, _ := client.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String("users"),
        Key: map[string]types.AttributeValue{
            "pk": &types.AttributeValueMemberS{Value: "user-1"},
        },
    })

    name := out.Item["name"].(*types.AttributeValueMemberS).Value
    if name != "Alice" {
        t.Fatalf("expected Alice, got %s", name)
    }
}
```

Run with `go test ./...`. No server startup, no cleanup step.

---

## Go: HTTP mode

For testing HTTP-level behavior or when you need Docker compatibility, run CloudMock as an HTTP server and configure the SDK client to point at it.

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

    _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String("test-bucket"),
    })
    if err != nil {
        t.Fatalf("CreateBucket: %v", err)
    }

    body := []byte("hello from Go test")
    _, err = client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String("test-bucket"),
        Key:    aws.String("greeting.txt"),
        Body:   bytes.NewReader(body),
    })
    if err != nil {
        t.Fatalf("PutObject: %v", err)
    }

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
```

Start CloudMock before running:

```bash
npx cloudmock start &
go test ./... -v
```

---

## Python (pytest)

Install the CloudMock Python SDK:

```bash
pip install cloudmock
```

The `CloudMock` context manager starts an embedded server and configures boto3 automatically. Use `scope="session"` to share one instance across the entire test run.

```python
import pytest
from cloudmock import CloudMock

@pytest.fixture(scope="session")
def aws():
    with CloudMock() as cm:
        yield cm

@pytest.fixture
def s3_client(aws):
    return aws.boto3_client("s3")

@pytest.fixture
def dynamodb(aws):
    return aws.boto3_client("dynamodb")

def test_s3_upload(s3_client):
    s3_client.create_bucket(Bucket="test")
    s3_client.put_object(Bucket="test", Key="hello.txt", Body=b"world")
    obj = s3_client.get_object(Bucket="test", Key="hello.txt")
    assert obj["Body"].read() == b"world"

def test_dynamodb_crud(dynamodb):
    dynamodb.create_table(
        TableName="users",
        KeySchema=[{"AttributeName": "pk", "KeyType": "HASH"}],
        AttributeDefinitions=[{"AttributeName": "pk", "AttributeType": "S"}],
        BillingMode="PAY_PER_REQUEST",
    )
    dynamodb.put_item(TableName="users", Item={"pk": {"S": "user-1"}, "name": {"S": "Alice"}})
    resp = dynamodb.get_item(TableName="users", Key={"pk": {"S": "user-1"}})
    assert resp["Item"]["name"]["S"] == "Alice"

def test_sqs_send_receive(aws):
    sqs = aws.boto3_client("sqs")
    queue = sqs.create_queue(QueueName="tasks")
    sqs.send_message(QueueUrl=queue["QueueUrl"], MessageBody="do-something")
    msgs = sqs.receive_message(QueueUrl=queue["QueueUrl"])
    assert msgs["Messages"][0]["Body"] == "do-something"
```

To reset state between tests, add an autouse fixture:

```python
@pytest.fixture(autouse=True)
def reset_cloudmock(aws):
    yield
    aws.reset()
```

Run with:

```bash
pytest -v
```

---

## Node.js (Jest)

Install the CloudMock SDK:

```bash
npm install --save-dev @cloudmock/sdk
```

```javascript
const { CloudMock } = require("@cloudmock/sdk");
const { S3Client, CreateBucketCommand, PutObjectCommand, GetObjectCommand } = require("@aws-sdk/client-s3");
const { DynamoDBClient, CreateTableCommand, PutItemCommand, GetItemCommand } = require("@aws-sdk/client-dynamodb");

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
    // GetObject and verify...
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
```

`cm.clientConfig()` returns `{ endpoint, region, credentials, forcePathStyle }` pre-configured for CloudMock. Pass it directly to any AWS SDK v3 client constructor.

---

## Node.js (Vitest)

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

---

## Java (JUnit 5)

Add the CloudMock Java SDK to your `pom.xml`:

```xml
<dependency>
    <groupId>dev.cloudmock</groupId>
    <artifactId>cloudmock-sdk</artifactId>
    <version>1.0.0</version>
    <scope>test</scope>
</dependency>
```

```java
import dev.cloudmock.CloudMock;
import org.junit.jupiter.api.*;
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.dynamodb.DynamoDbClient;
import software.amazon.awssdk.services.dynamodb.model.*;

class AWSTest {
    static CloudMock cm;
    static S3Client s3;
    static DynamoDbClient ddb;

    @BeforeAll
    static void setup() throws Exception {
        cm = CloudMock.start();
        var credentials = StaticCredentialsProvider.create(
            AwsBasicCredentials.create("test", "test"));

        s3 = S3Client.builder()
            .endpointOverride(cm.endpoint())
            .region(Region.US_EAST_1)
            .credentialsProvider(credentials)
            .forcePathStyle(true)
            .build();

        ddb = DynamoDbClient.builder()
            .endpointOverride(cm.endpoint())
            .region(Region.US_EAST_1)
            .credentialsProvider(credentials)
            .build();
    }

    @AfterAll
    static void teardown() {
        cm.close();
    }

    @BeforeEach
    void reset() {
        cm.reset();
    }

    @Test
    void s3CreateBucket() {
        s3.createBucket(b -> b.bucket("test"));
        var buckets = s3.listBuckets().buckets();
        Assertions.assertTrue(buckets.stream().anyMatch(b -> b.name().equals("test")));
    }

    @Test
    void dynamoDBCrud() {
        ddb.createTable(CreateTableRequest.builder()
            .tableName("users")
            .keySchema(KeySchemaElement.builder()
                .attributeName("pk").keyType(KeyType.HASH).build())
            .attributeDefinitions(AttributeDefinition.builder()
                .attributeName("pk").attributeType(ScalarAttributeType.S).build())
            .billingMode(BillingMode.PAY_PER_REQUEST)
            .build());

        ddb.putItem(PutItemRequest.builder()
            .tableName("users")
            .item(Map.of(
                "pk", AttributeValue.builder().s("user-1").build(),
                "name", AttributeValue.builder().s("Alice").build()))
            .build());

        var resp = ddb.getItem(GetItemRequest.builder()
            .tableName("users")
            .key(Map.of("pk", AttributeValue.builder().s("user-1").build()))
            .build());

        Assertions.assertEquals("Alice", resp.item().get("name").s());
    }
}
```

---

## Rust

Add the CloudMock Rust crate:

```toml
# Cargo.toml
[dev-dependencies]
cloudmock = "0.1"
tokio = { version = "1", features = ["full"] }
aws-config = "1"
aws-sdk-s3 = "1"
aws-sdk-dynamodb = "1"
aws-credential-types = "1"
```

```rust
use cloudmock::CloudMock;

#[tokio::test]
async fn test_s3() {
    let cm = CloudMock::start().await.unwrap();
    let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
        .endpoint_url(cm.endpoint())
        .credentials_provider(aws_credential_types::Credentials::new(
            "test", "test", None, None, "cloudmock",
        ))
        .region(aws_config::Region::new("us-east-1"))
        .load()
        .await;

    let s3 = aws_sdk_s3::Client::new(&config);
    s3.create_bucket().bucket("test").send().await.unwrap();

    let buckets = s3.list_buckets().send().await.unwrap();
    assert!(buckets.buckets().iter().any(|b| b.name() == Some("test")));

    cm.stop().await;
}

#[tokio::test]
async fn test_dynamodb() {
    let cm = CloudMock::start().await.unwrap();
    let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
        .endpoint_url(cm.endpoint())
        .credentials_provider(aws_credential_types::Credentials::new(
            "test", "test", None, None, "cloudmock",
        ))
        .region(aws_config::Region::new("us-east-1"))
        .load()
        .await;

    let ddb = aws_sdk_dynamodb::Client::new(&config);

    ddb.create_table()
        .table_name("users")
        .key_schema(
            aws_sdk_dynamodb::types::KeySchemaElement::builder()
                .attribute_name("pk")
                .key_type(aws_sdk_dynamodb::types::KeyType::Hash)
                .build()
                .unwrap(),
        )
        .attribute_definitions(
            aws_sdk_dynamodb::types::AttributeDefinition::builder()
                .attribute_name("pk")
                .attribute_type(aws_sdk_dynamodb::types::ScalarAttributeType::S)
                .build()
                .unwrap(),
        )
        .billing_mode(aws_sdk_dynamodb::types::BillingMode::PayPerRequest)
        .send()
        .await
        .unwrap();

    cm.stop().await;
}
```

Run with:

```bash
cargo test
```

---

## Ruby (Minitest)

Install the CloudMock Ruby gem:

```bash
gem install cloudmock
```

Or in your `Gemfile`:

```ruby
gem "cloudmock", group: :test
```

```ruby
require "minitest/autorun"
require "cloudmock"
require "aws-sdk-s3"
require "aws-sdk-dynamodb"

class AWSTest < Minitest::Test
    def setup
        @cm = CloudMock.start
        @s3 = Aws::S3::Client.new(@cm.aws_config)
        @ddb = Aws::DynamoDB::Client.new(@cm.aws_config)
    end

    def teardown
        @cm.stop
    end

    def test_create_bucket
        @s3.create_bucket(bucket: "test")
        buckets = @s3.list_buckets.buckets.map(&:name)
        assert_includes buckets, "test"
    end

    def test_dynamodb_crud
        @ddb.create_table(
            table_name: "users",
            key_schema: [{ attribute_name: "pk", key_type: "HASH" }],
            attribute_definitions: [{ attribute_name: "pk", attribute_type: "S" }],
            billing_mode: "PAY_PER_REQUEST",
        )
        @ddb.put_item(
            table_name: "users",
            item: { "pk" => "user-1", "name" => "Alice" },
        )
        resp = @ddb.get_item(
            table_name: "users",
            key: { "pk" => "user-1" },
        )
        assert_equal "Alice", resp.item["name"]
    end

    def test_sqs_round_trip
        sqs = Aws::SQS::Client.new(@cm.aws_config)
        queue = sqs.create_queue(queue_name: "tasks")
        sqs.send_message(queue_url: queue.queue_url, message_body: "do-something")
        msgs = sqs.receive_message(queue_url: queue.queue_url)
        assert_equal "do-something", msgs.messages.first.body
    end
end
```

`@cm.aws_config` returns an `Aws::Config` hash with `endpoint`, `region`, and `credentials` set for CloudMock.

---

## Resetting state between tests

CloudMock holds all state in memory. Reset to a clean slate between tests using the admin API:

```bash
curl -X POST http://localhost:4599/api/reset
```

Response:

```json
{"reset": 5, "services": ["s3", "dynamodb", "sqs", "sns", "sts"]}
```

Reset a single service without touching others:

```bash
curl -X POST http://localhost:4599/api/services/dynamodb/reset
```

### Reset helpers

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

---

## CI integration

### GitHub Actions — cloudmock-action (recommended)

The official `cloudmock-action` handles install, startup, health-check, and environment variable setup in one step:

```yaml
name: Tests
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1
      - run: npm test  # AWS_ENDPOINT_URL is already set
```

The action automatically sets `AWS_ENDPOINT_URL`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_DEFAULT_REGION` for all subsequent steps.

#### With options

```yaml
- uses: viridian-inc/cloudmock-action@v1
  with:
    profile: full            # minimal (default), standard, or full (all 100 services)
    state: fixtures/state.json  # pre-load a state snapshot
    iam-mode: enforce        # none (default), log, or enforce
```

#### Outputs

```yaml
- uses: viridian-inc/cloudmock-action@v1
  id: cloudmock
- run: |
    echo "Endpoint: ${{ steps.cloudmock.outputs.endpoint }}"
    aws --endpoint ${{ steps.cloudmock.outputs.endpoint }} s3 ls
```

| Output | Description |
|--------|-------------|
| `endpoint` | Gateway URL (e.g., `http://localhost:4566`) |
| `admin-url` | Admin API URL (e.g., `http://localhost:4599`) |
| `version` | Installed CloudMock version |

### GitHub Actions — npx

```yaml
name: Tests
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm install -g cloudmock
      - run: cloudmock &
      - run: sleep 2
      - run: npm test
```

### GitHub Actions — Docker service

```yaml
name: Tests
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      cloudmock:
        image: ghcr.io/viridian-inc/cloudmock:latest
        ports: ["4566:4566", "4599:4599"]
    steps:
      - uses: actions/checkout@v4
      - run: npm test
```

### GitLab CI — Docker service

```yaml
test:
  services:
    - name: ghcr.io/viridian-inc/cloudmock:latest
      alias: cloudmock
  variables:
    AWS_ENDPOINT_URL: http://cloudmock:4566
    AWS_ACCESS_KEY_ID: test
    AWS_SECRET_ACCESS_KEY: test
    AWS_DEFAULT_REGION: us-east-1
  script:
    - pytest -v
```

### Docker Compose (multi-service)

```yaml
services:
  cloudmock:
    image: ghcr.io/viridian-inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4599:4599"
    environment:
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

---

## Asserting on traces

After your code runs, query the traces API to verify what happened:

```bash
curl "http://localhost:4599/api/traces?service=dynamodb&limit=10"
```

Each trace contains the service, action, status code, and full request/response payloads. This lets you verify that your application made the expected AWS calls without inspecting return values.

```typescript
// Verify that a PutItem was made to the users table
const res = await fetch("http://localhost:4599/api/requests?service=dynamodb&action=PutItem");
const requests = await res.json();
const userPut = requests.find(
    (r: any) => r.request_body?.TableName === "users"
);
expect(userPut).toBeDefined();
expect(userPut.status_code).toBe(200);
```

---

## Tips

**Prefer in-process mode for Go.** At ~20 μs/op, in-process mode eliminates HTTP overhead and makes large test suites dramatically faster. Use HTTP mode only when you need to test actual network behavior or run in a Docker environment.

**Disable IAM in tests.** Set `CLOUDMOCK_IAM_MODE=none` to skip authentication checks. This avoids the need to configure credentials in every test client.

**Reset aggressively.** Call reset in `beforeEach` / `setUp`, not `afterEach` / `tearDown`. Each test starts clean even if the previous test crashed.

**Run CloudMock once per test suite.** Starting a new CloudMock process for every test file is slow. Start it once before the suite and reset state between tests.

**Seed data with the AWS SDK.** Rather than using fixtures or SQL inserts, create your test data using the same AWS SDK calls your application uses. This keeps test setup and production code aligned.

**Check traces for debugging.** When a test fails, query `GET /api/requests?level=all` to see exactly what AWS API calls were made and what CloudMock returned. This is often more informative than the test assertion message.
