---
title: Switching from Moto
description: Migrate from Moto to CloudMock — pytest, unittest, and server mode
---

# Switching from Moto to CloudMock

Moto mocks AWS at the Python SDK level using decorators or a standalone HTTP server. CloudMock is a real HTTP server that any language's AWS SDK can point at — it works with Python, Go, Node, Java, Rust, and more. Migration is straightforward for both pytest and unittest projects.

## Why Switch?

| | Moto | CloudMock |
|---|---|---|
| Works with any language | No (Python only in-process) | Yes — HTTP server for all SDKs |
| In-process speed | Yes (Python) | Yes (Go, ~20 μs/op) |
| DevTools UI | No | Yes (localhost:4500) |
| Chaos engineering | No | Built-in, free |
| State snapshots | No | Built-in, free |
| Traffic replay | No | Built-in, free |
| IaC support (Terraform, CDK) | No | Built-in, free |
| Protocol-level testing | No (intercepts at SDK) | Yes (real HTTP wire protocol) |
| Service count | 150+ (mock depth varies) | 99 (fully emulated) |

## pytest Migration

### Decorator style → fixture style

```python
# Before (Moto)
import boto3
import pytest
from moto import mock_aws

@mock_aws
def test_create_bucket():
    s3 = boto3.client("s3", region_name="us-east-1")
    s3.create_bucket(Bucket="my-bucket")
    response = s3.list_buckets()
    assert len(response["Buckets"]) == 1
```

```python
# After (CloudMock)
import pytest
from cloudmock import CloudMock

@pytest.fixture(scope="session")
def aws():
    with CloudMock() as cm:
        yield cm

def test_create_bucket(aws):
    s3 = aws.boto3_client("s3")
    s3.create_bucket(Bucket="my-bucket")
    response = s3.list_buckets()
    assert len(response["Buckets"]) == 1
```

### Class-based tests

```python
# Before (Moto)
from moto import mock_aws

@mock_aws
class TestS3:
    def test_put_object(self):
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket="my-bucket")
        s3.put_object(Bucket="my-bucket", Key="hello.txt", Body=b"world")
```

```python
# After (CloudMock)
import pytest
from cloudmock import CloudMock

@pytest.fixture(scope="class")
def aws():
    with CloudMock() as cm:
        yield cm

class TestS3:
    def test_put_object(self, aws):
        s3 = aws.boto3_client("s3")
        s3.create_bucket(Bucket="my-bucket")
        s3.put_object(Bucket="my-bucket", Key="hello.txt", Body=b"world")
```

### Shared fixture across test files

```python
# conftest.py
import pytest
from cloudmock import CloudMock

@pytest.fixture(scope="session")
def cloudmock_instance():
    with CloudMock() as cm:
        yield cm

@pytest.fixture
def s3(cloudmock_instance):
    return cloudmock_instance.boto3_client("s3")

@pytest.fixture
def dynamodb(cloudmock_instance):
    return cloudmock_instance.boto3_client("dynamodb")

@pytest.fixture
def sqs(cloudmock_instance):
    return cloudmock_instance.boto3_client("sqs")
```

## unittest Migration

```python
# Before (Moto)
import unittest
import boto3
from moto import mock_aws

class TestDynamoDB(unittest.TestCase):
    @mock_aws
    def test_create_table(self):
        dynamodb = boto3.client("dynamodb", region_name="us-east-1")
        dynamodb.create_table(
            TableName="users",
            KeySchema=[{"AttributeName": "id", "KeyType": "HASH"}],
            AttributeDefinitions=[{"AttributeName": "id", "AttributeType": "S"}],
            BillingMode="PAY_PER_REQUEST",
        )
        tables = dynamodb.list_tables()["TableNames"]
        self.assertIn("users", tables)
```

```python
# After (CloudMock)
import unittest
from cloudmock import CloudMock

class TestDynamoDB(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.cm = CloudMock()
        cls.cm.__enter__()

    @classmethod
    def tearDownClass(cls):
        cls.cm.__exit__(None, None, None)

    def test_create_table(self):
        dynamodb = self.cm.boto3_client("dynamodb")
        dynamodb.create_table(
            TableName="users",
            KeySchema=[{"AttributeName": "id", "KeyType": "HASH"}],
            AttributeDefinitions=[{"AttributeName": "id", "AttributeType": "S"}],
            BillingMode="PAY_PER_REQUEST",
        )
        tables = dynamodb.list_tables()["TableNames"]
        self.assertIn("users", tables)
```

## Server Mode Migration

Moto has a standalone server mode (`moto_server`) that behaves like CloudMock's HTTP mode. Switching is a direct replacement:

```bash
# Before (Moto server mode)
pip install moto[server]
moto_server -p 5000
# then point boto3 at http://localhost:5000

# After (CloudMock)
npx cloudmock
# or: brew install viridian-inc/tap/cloudmock && cloudmock
# point boto3 at http://localhost:4566
```

Update your boto3 clients:

```python
# Before (Moto server mode)
import boto3
s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:5000",
    region_name="us-east-1",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)

# After (CloudMock HTTP mode)
import boto3
s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:4566",
    region_name="us-east-1",
    aws_access_key_id="test",
    aws_secret_access_key="test",
)
```

Or use the CloudMock SDK to skip manual configuration entirely:

```python
from cloudmock import CloudMock

with CloudMock() as cm:
    s3 = cm.boto3_client("s3")   # endpoint, credentials, region pre-configured
    sqs = cm.boto3_client("sqs")
```

## Concept Mapping

| Moto | CloudMock | Notes |
|------|-----------|-------|
| `@mock_aws` decorator | `CloudMock()` context manager | Wraps entire test or fixture scope |
| `@mock_s3`, `@mock_dynamodb` | Same `CloudMock()` — all services active | No per-service decorators needed |
| `moto_server -p 5000` | `npx cloudmock` (port 4566) | Real HTTP server, same pattern |
| In-process Python mocking | HTTP server (Python) or in-process (Go) | Python tests use HTTP; Go gets 20 μs/op |
| Per-decorator isolation | Fixture scope controls isolation | Use `scope="function"` for per-test |
| `boto3.client(endpoint_url=...)` | `cm.boto3_client("s3")` | Auto-configured credentials |

## Test Isolation

Moto resets state between each `@mock_aws` decorated test automatically. CloudMock resets when you restart or re-enter the context manager. For per-test isolation, use a function-scoped fixture:

```python
# Per-test isolation (equivalent to @mock_aws on each test)
@pytest.fixture
def aws():
    with CloudMock() as cm:
        yield cm
# CloudMock starts fresh for every test function
```

For faster tests that share state across a suite:

```python
# Shared state (faster — one CloudMock instance for the whole session)
@pytest.fixture(scope="session")
def aws():
    with CloudMock() as cm:
        yield cm
```

## GitHub Actions

```yaml
# Before (Moto — no special setup needed, just pip install)
- run: pip install moto pytest boto3
- run: pytest

# After (CloudMock)
- uses: viridian-inc/cloudmock-action@v1
- run: pip install cloudmock pytest boto3
- run: pytest
```

## What You Gain

| Feature | Moto | CloudMock |
|---------|------|-----------|
| Language support | Python only (in-process) | Any language via HTTP |
| Protocol-level testing | No (SDK intercept) | Yes (real HTTP wire protocol) |
| DevTools UI | No | Yes (topology, traces, chaos) |
| Chaos engineering | No | Built-in, free |
| State snapshots | No | Built-in, free |
| Traffic replay | No | Built-in, free |
| IaC support | No | Terraform, CDK, Pulumi |
| GitHub Action | No | `viridian-inc/cloudmock-action@v1` |
| Multi-language team support | No | Yes |
| Contract testing | No | Built-in, free |

## What's Different

- **HTTP overhead**: Moto's in-process Python mode has zero network overhead; CloudMock's Python SDK uses HTTP (still fast — ~1ms/op). If you need zero-network speed, use CloudMock's Go SDK (~20 μs/op).
- **Test isolation model**: Moto resets state per decorator; CloudMock resets per context manager scope. Use `scope="function"` in pytest for equivalent per-test isolation.
- **Service decorator specificity**: Moto lets you use `@mock_s3` to mock only S3. CloudMock runs all 99 services always — there's no per-service activation.
- **Lambda execution**: Moto can execute simple Lambda handlers in-process. CloudMock stores Lambda configuration but does not execute handlers.
- **License**: CloudMock uses BSL-1.1 (free for dev/test, commercial production hosting requires a license). Moto uses Apache 2.0.
