# cloudmock

Local AWS emulation for Python. 98 services. One line of code.

## Install

```bash
pip install cloudmock
```

Requires the CloudMock binary — install with `npm install -g cloudmock` or `brew install viridian-inc/tap/cloudmock`.

## Usage

```python
from cloudmock import mock_aws

with mock_aws() as cm:
    s3 = cm.boto3_client("s3")
    s3.create_bucket(Bucket="my-bucket")
    s3.put_object(Bucket="my-bucket", Key="hello.txt", Body=b"world")

    ddb = cm.boto3_client("dynamodb")
    ddb.create_table(
        TableName="users",
        KeySchema=[{"AttributeName": "pk", "KeyType": "HASH"}],
        AttributeDefinitions=[{"AttributeName": "pk", "AttributeType": "S"}],
        BillingMode="PAY_PER_REQUEST",
    )
```

## pytest fixture

```python
import pytest
from cloudmock import CloudMock

@pytest.fixture(scope="session")
def aws():
    with CloudMock() as cm:
        yield cm

def test_s3(aws):
    s3 = aws.boto3_client("s3")
    s3.create_bucket(Bucket="test")
    buckets = s3.list_buckets()["Buckets"]
    assert any(b["Name"] == "test" for b in buckets)
```
