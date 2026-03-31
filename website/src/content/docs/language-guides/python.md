---
title: Python
description: Using CloudMock with Python, the cloudmock Python SDK, and boto3
---

Python is a first-class language for CloudMock. The `cloudmock` Python SDK provides automatic request interception and trace propagation. For AWS-only usage, configure `boto3` to point at the CloudMock gateway.

## cloudmock Python SDK

### Install

```bash
pip install cloudmock
```

### Initialize

Call `cloudmock.init()` early in your application startup, before creating boto3 clients or sessions. This patches the `requests` library to intercept outgoing HTTP calls to the CloudMock gateway and forward telemetry to the admin API.

```python
import cloudmock

cloudmock.init(
    admin_url="http://localhost:4599",
    service_name="my-api",
)
```

### requests interceptor

The SDK monkey-patches `requests.Session.send` to intercept outgoing HTTP requests. All boto3 calls (which use `requests` internally) are automatically captured. If you use a custom HTTP client, the SDK also provides an explicit interceptor:

```python
import cloudmock
from cloudmock import intercept_session

cloudmock.init(admin_url="http://localhost:4599", service_name="my-api")

# Wrap a custom requests session
import requests
session = requests.Session()
intercept_session(session)
```

### WSGI/ASGI middleware

For web applications, the SDK provides middleware that traces inbound HTTP requests:

```python
# Flask
from flask import Flask
from cloudmock.flask import CloudMockMiddleware

app = Flask(__name__)
app.wsgi_app = CloudMockMiddleware(app.wsgi_app)

# FastAPI
from fastapi import FastAPI
from cloudmock.fastapi import CloudMockMiddleware

app = FastAPI()
app.add_middleware(CloudMockMiddleware)
```

### What gets captured

- **Inbound HTTP requests** -- Method, path, status code, duration.
- **Outbound AWS calls** -- Service, action, latency, status code via the requests interceptor.
- **Trace context** -- A trace ID is generated per inbound request and propagated to all outbound calls.
- **Service identity** -- The `service_name` appears as a node in the Topology view.

## boto3 endpoint configuration

If you do not need the CloudMock Python SDK, configure boto3 directly:

### Environment variable

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

The `AWS_ENDPOINT_URL` environment variable is supported by boto3 since version 1.29.0.

### Programmatic configuration

```python
import boto3

session = boto3.Session(
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)

# S3
s3 = session.client("s3", endpoint_url="http://localhost:4566")
s3.create_bucket(Bucket="my-bucket")
s3.put_object(Bucket="my-bucket", Key="hello.txt", Body=b"Hello, CloudMock!")

# DynamoDB
dynamodb = session.resource("dynamodb", endpoint_url="http://localhost:4566")
table = dynamodb.create_table(
    TableName="Users",
    KeySchema=[{"AttributeName": "UserId", "KeyType": "HASH"}],
    AttributeDefinitions=[{"AttributeName": "UserId", "AttributeType": "S"}],
    BillingMode="PAY_PER_REQUEST",
)
print(f"Table created: {table.table_name}")

# SQS
sqs = session.client("sqs", endpoint_url="http://localhost:4566")
queue = sqs.create_queue(QueueName="my-queue")
sqs.send_message(QueueUrl=queue["QueueUrl"], MessageBody="test message")
```

### Conditional endpoint

```python
import os
import boto3

def get_session():
    endpoint = os.environ.get("CLOUDMOCK_ENDPOINT")
    if endpoint:
        return boto3.Session(
            aws_access_key_id="test",
            aws_secret_access_key="test",
            region_name="us-east-1",
        )
    return boto3.Session()

def get_client(service_name):
    session = get_session()
    endpoint = os.environ.get("CLOUDMOCK_ENDPOINT")
    kwargs = {}
    if endpoint:
        kwargs["endpoint_url"] = endpoint
    return session.client(service_name, **kwargs)

# Usage
s3 = get_client("s3")
dynamodb = get_client("dynamodb")
```

### pytest fixture

A common pattern for tests:

```python
import pytest
import boto3

@pytest.fixture
def aws_session():
    return boto3.Session(
        aws_access_key_id="test",
        aws_secret_access_key="test",
        region_name="us-east-1",
    )

@pytest.fixture
def s3_client(aws_session):
    return aws_session.client("s3", endpoint_url="http://localhost:4566")

@pytest.fixture
def dynamodb_resource(aws_session):
    return aws_session.resource("dynamodb", endpoint_url="http://localhost:4566")
```

## Common issues

### endpoint_url must be set per client

Unlike AWS SDK v3 for JavaScript, boto3 does not support a single global endpoint override via the client constructor. You must pass `endpoint_url` to each `client()` or `resource()` call. The `AWS_ENDPOINT_URL` environment variable avoids this, but requires boto3 1.29.0+.

### S3 path style

boto3 uses path-style S3 access by default when an explicit `endpoint_url` is set, so no additional configuration is needed.
