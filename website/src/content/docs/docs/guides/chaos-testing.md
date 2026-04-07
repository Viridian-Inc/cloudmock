---
title: Chaos Testing
description: Inject faults into CloudMock services to test retries, circuit breakers, and timeout handling
---

Chaos testing verifies that your application handles AWS failures gracefully — throttling, transient errors, network partitions, and latency spikes. CloudMock lets you inject these faults deterministically in unit and integration tests, without flaky real-AWS behavior.

## Fault types

| Type | Effect |
|------|--------|
| `error` | Returns a configurable HTTP error status code |
| `latency` | Adds artificial delay before responding |
| `timeout` | Holds the connection for 30 seconds then returns 504 |
| `blackhole` | Closes the connection without sending any response |
| `throttle` | Returns HTTP 429 with a `ThrottlingException` body |

---

## Go: in-process SDK

The Go SDK injects faults directly into the embedded gateway engine — no HTTP round-trips needed.

### Basic usage

```go
import "github.com/Viridian-Inc/cloudmock/sdk"

func TestRetryOnThrottle(t *testing.T) {
    cm := sdk.New()
    defer cm.Close()

    // Inject a DynamoDB throttle on all actions.
    cm.InjectFault("dynamodb", "*", "throttle")

    client := dynamodb.NewFromConfig(cm.Config())
    _, err := client.GetItem(ctx, &dynamodb.GetItemInput{...})
    // err will be a ThrottlingException (HTTP 429)
}
```

### Available options

```go
cm.InjectFault("s3", "GetObject", "error",
    sdk.WithStatusCode(503),           // HTTP status code (default: 500)
    sdk.WithMessage("Service Unavailable"),
    sdk.WithPercentage(50),            // Fire on 50% of matching requests
)

cm.InjectFault("dynamodb", "*", "latency",
    sdk.WithLatency(2000),             // Add 2 seconds of latency
)
```

### Testing retry logic

```go
func TestRetryExhaustion(t *testing.T) {
    cm := sdk.New()
    defer cm.Close()

    // Fail all S3 calls with 503 Service Unavailable.
    cm.InjectFault("s3", "*", "error",
        sdk.WithStatusCode(503),
        sdk.WithMessage("Service Unavailable"),
    )

    client := s3.NewFromConfig(cm.Config(), func(o *s3.Options) {
        o.UsePathStyle = true
    })

    _, err := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("my-key"),
    })
    require.Error(t, err)
    // Verify your retry logic exhausted the configured max attempts.
}
```

### Testing circuit breaker behavior

```go
func TestCircuitBreaker(t *testing.T) {
    cm := sdk.New()
    defer cm.Close()

    // Inject errors on 80% of DynamoDB calls to simulate degraded service.
    cm.InjectFault("dynamodb", "*", "error",
        sdk.WithStatusCode(500),
        sdk.WithPercentage(80),
    )

    // Run your service logic — circuit breaker should open after enough failures.
    err := yourService.DoSomething(ctx, cm.Config())
    assert.True(t, yourService.CircuitIsOpen())

    // Clear faults — circuit breaker should close after recovery.
    cm.ClearFaults()
    err = yourService.DoSomething(ctx, cm.Config())
    assert.NoError(t, err)
    assert.False(t, yourService.CircuitIsOpen())
}
```

### Testing timeout handling

```go
func TestTimeoutHandling(t *testing.T) {
    cm := sdk.New()
    defer cm.Close()

    // Simulate a hung Lambda service.
    cm.InjectFault("lambda", "*", "timeout")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    client := lambda.NewFromConfig(cm.Config())
    _, err := client.Invoke(ctx, &lambda.InvokeInput{
        FunctionName: aws.String("my-function"),
    })

    // The context deadline should fire before the 30-second chaos timeout.
    require.ErrorIs(t, err, context.DeadlineExceeded)
}
```

---

## Python: pytest

Install the Python SDK:

```bash
pip install cloudmock
```

Use `inject_fault()` and `clear_faults()` on a running instance:

```python
import pytest
from cloudmock import CloudMock

@pytest.fixture(scope="module")
def cm():
    with CloudMock() as instance:
        yield instance

def test_retry_on_throttle(cm):
    cm.inject_fault("dynamodb", "*", "throttle",
        message="Rate exceeded", percentage=100)

    ddb = cm.boto3_client("dynamodb")
    with pytest.raises(Exception) as exc_info:
        ddb.list_tables()
    assert "ThrottlingException" in str(exc_info.value)

def test_recovery_after_clear(cm):
    cm.inject_fault("s3", "*", "error", status_code=503)

    s3 = cm.boto3_client("s3")
    with pytest.raises(Exception):
        s3.list_buckets()

    cm.clear_faults()
    # Now requests should succeed.
    response = s3.list_buckets()
    assert "Buckets" in response
```

### pytest fixture with automatic cleanup

```python
@pytest.fixture
def chaos_cm(cm):
    """Yields the CloudMock instance and automatically clears faults after each test."""
    yield cm
    cm.clear_faults()

def test_s3_503(chaos_cm):
    chaos_cm.inject_fault("s3", "*", "error", status_code=503)
    s3 = chaos_cm.boto3_client("s3")
    with pytest.raises(Exception):
        s3.list_buckets()
    # clear_faults() called automatically by fixture teardown
```

---

## Node.js: Jest

Install the Node SDK:

```bash
npm install @cloudmock/sdk
```

```javascript
const { mockAWS } = require("@cloudmock/sdk");
const { S3Client, ListBucketsCommand } = require("@aws-sdk/client-s3");
const { DynamoDBClient, ListTablesCommand } = require("@aws-sdk/client-dynamodb");

let cm;

beforeAll(async () => {
    cm = await mockAWS();
});

afterAll(async () => {
    await cm.stop();
});

afterEach(async () => {
    await cm.clearFaults();
});

test("S3 returns 503 when error fault is active", async () => {
    await cm.injectFault("s3", "*", "error", { statusCode: 503 });
    const s3 = new S3Client(cm.clientConfig());
    await expect(s3.send(new ListBucketsCommand({}))).rejects.toThrow();
});

test("DynamoDB returns 429 when throttle fault is active", async () => {
    await cm.injectFault("dynamodb", "*", "throttle", {
        message: "Rate exceeded",
    });
    const ddb = new DynamoDBClient(cm.clientConfig());
    await expect(ddb.send(new ListTablesCommand({}))).rejects.toThrow(/429/);
});

test("normal operation resumes after clearing faults", async () => {
    await cm.injectFault("s3", "*", "error", { statusCode: 500 });
    await cm.clearFaults();
    const s3 = new S3Client(cm.clientConfig());
    // Should not throw.
    const response = await s3.send(new ListBucketsCommand({}));
    expect(response.Buckets).toBeDefined();
});
```

---

## Config file: static fault injection

You can define chaos rules in your `cloudmock.yaml` so they are active at startup. This is useful for integration test environments where you always want certain fault conditions present.

```yaml
# cloudmock.yaml
chaos:
  rules:
    - service: dynamodb
      action: "*"
      type: latency
      latency_ms: 100
      percentage: 100

    - service: s3
      action: GetObject
      type: error
      error_code: 503
      error_msg: "Injected S3 read failure"
      percentage: 25
```

Config-file rules are loaded once at startup and immediately active. They supplement any rules managed through the admin API or devtools UI.

### Config rule fields

| Field | Type | Description |
|-------|------|-------------|
| `service` | string | Target service (`"s3"`, `"dynamodb"`, `"*"` for all) |
| `action` | string | Target API action or `"*"` for all |
| `type` | string | `error`, `latency`, `timeout`, `blackhole`, or `throttle` |
| `error_code` | int | HTTP status code for `error` faults |
| `error_msg` | string | Error message returned with the fault |
| `latency_ms` | int | Milliseconds of added latency for `latency` faults |
| `percentage` | int | 0–100 probability the fault fires per request |

---

## Common patterns

### Retry testing

Verify that your retry configuration actually retries on transient errors:

```go
// Use 50% failure rate to test that retries succeed when the service recovers.
cm.InjectFault("s3", "*", "error",
    sdk.WithStatusCode(503),
    sdk.WithPercentage(50),
)
```

### Circuit breaker testing

Inject 100% failures to trigger circuit opening, then `ClearFaults()` to simulate recovery.

### Timeout handling

Use `"timeout"` fault type to simulate hung connections. Pair with a short `context.WithTimeout` to verify your application's timeout handling fires before the chaos timeout.

### Partial degradation

Use `WithPercentage` to simulate a partially degraded service. For example, 10% error rate to test that your application handles occasional failures without completely failing.

### Throttle handling

Use `"throttle"` to verify that your application respects AWS rate limits and implements exponential backoff correctly.
