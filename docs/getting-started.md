# Getting Started

## Prerequisites

- **Go 1.26+** — required to build from source
- **Docker and Docker Compose** — optional, for container-based deployment
- **AWS CLI v2** — for testing (any version that supports `--endpoint-url`)

---

## Installation

### Option 1: Build from source

```bash
git clone https://github.com/neureaux/cloudmock
cd cloudmock
make build
# Produces: ./bin/cloudmock and ./bin/gateway
```

### Option 2: go install

```bash
go install github.com/neureaux/cloudmock/cmd/cloudmock@latest
```

### Option 3: Docker Compose

```bash
git clone https://github.com/neureaux/cloudmock
cd cloudmock
docker compose up -d
```

This starts the gateway on port `4566`, the dashboard on `4500`, and the admin API on `4599`.

---

## First Run

### 1. Start the gateway

```bash
./bin/cloudmock start
# Starting cloudmock gateway (config=cloudmock.yml)
```

By default, cloudmock uses the `minimal` profile (IAM, STS, S3, DynamoDB, SQS, SNS, Lambda, CloudWatch Logs) with IAM enforcement enabled and root credentials `test` / `test`.

### 2. Configure the AWS CLI

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

Or add a named profile to `~/.aws/config`:

```ini
[profile cloudmock]
region = us-east-1
endpoint_url = http://localhost:4566
```

```ini
# ~/.aws/credentials
[cloudmock]
aws_access_key_id = test
aws_secret_access_key = test
```

Then use `--profile cloudmock` with every CLI command.

### 3. Create an S3 bucket

```bash
aws s3 mb s3://my-first-bucket
# make_bucket: my-first-bucket

aws s3 ls
# 2026-03-21 00:00:00 my-first-bucket
```

### 4. Put and retrieve an object

```bash
echo "Hello, cloudmock!" > hello.txt
aws s3 cp hello.txt s3://my-first-bucket/hello.txt
aws s3 cp s3://my-first-bucket/hello.txt -
# Hello, cloudmock!
```

---

## SDK Examples

### Go (aws-sdk-go-v2)

```go
package main

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
        config.WithBaseEndpoint("http://localhost:4566"),
    )
    if err != nil {
        panic(err)
    }

    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle = true
    })

    _, err = client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
        Bucket: aws.String("my-bucket"),
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("Bucket created")
}
```

### Python (boto3)

```python
import boto3

# Create a session pointing at cloudmock
session = boto3.Session(
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)

s3 = session.client("s3", endpoint_url="http://localhost:4566")
s3.create_bucket(Bucket="my-bucket")

dynamodb = session.resource("dynamodb", endpoint_url="http://localhost:4566")
table = dynamodb.create_table(
    TableName="Users",
    KeySchema=[{"AttributeName": "UserId", "KeyType": "HASH"}],
    AttributeDefinitions=[{"AttributeName": "UserId", "AttributeType": "S"}],
    BillingMode="PAY_PER_REQUEST",
)
print("Table created:", table.table_name)
```

### Node.js (@aws-sdk/client-s3)

```js
const { S3Client, CreateBucketCommand, PutObjectCommand } = require("@aws-sdk/client-s3");

const client = new S3Client({
  region: "us-east-1",
  endpoint: "http://localhost:4566",
  credentials: {
    accessKeyId: "test",
    secretAccessKey: "test",
  },
  forcePathStyle: true,
});

async function main() {
  await client.send(new CreateBucketCommand({ Bucket: "my-bucket" }));
  await client.send(new PutObjectCommand({
    Bucket: "my-bucket",
    Key: "hello.txt",
    Body: "Hello, cloudmock!",
  }));
  console.log("Done");
}

main().catch(console.error);
```

---

## Endpoint Configuration

### Environment variable (recommended for dev)

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
```

This is respected by AWS CLI v2 and all official AWS SDKs.

### Per-service override (CLI)

```bash
aws s3 --endpoint-url http://localhost:4566 ls
aws dynamodb --endpoint-url http://localhost:4566 list-tables
```

### AWS config file

```ini
[profile cloudmock]
region = us-east-1
endpoint_url = http://localhost:4566
```

---

## Checking Status

```bash
cloudmock status
# Status: ok
#
# SERVICE             HEALTHY
# -------             -------
# s3                  yes
# dynamodb            yes
# sqs                 yes
```

Open the dashboard at http://localhost:4500 for a graphical view.
