---
title: Use with Your Stack
description: Configure AWS SDKs in Node.js, Python, Go, and Java to use CloudMock
---

Every official AWS SDK supports custom endpoints. Point it at `http://localhost:4566` and your application code talks to CloudMock instead of AWS.

## Environment variable (all SDKs)

The simplest approach works across all languages and tools. Set `AWS_ENDPOINT_URL` before running your application:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

AWS CLI v2 and all current AWS SDKs respect `AWS_ENDPOINT_URL`. No code changes required.

This is the recommended approach for local development. Wrap it in a script, add it to your `.env` file, or set it in your IDE's run configuration.

## Node.js (AWS SDK v3)

```typescript
import { S3Client, CreateBucketCommand } from "@aws-sdk/client-s3";

const client = new S3Client({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: {
    accessKeyId: "test",
    secretAccessKey: "test",
  },
  forcePathStyle: true,
});

await client.send(new CreateBucketCommand({ Bucket: "my-bucket" }));
```

The `forcePathStyle: true` option is required for S3. It makes the SDK use `http://localhost:4566/my-bucket` instead of `http://my-bucket.localhost:4566`, which does not resolve locally.

Other services (DynamoDB, SQS, SNS, etc.) only need `endpoint` and `credentials`:

```typescript
import { DynamoDBClient } from "@aws-sdk/client-dynamodb";

const dynamo = new DynamoDBClient({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: {
    accessKeyId: "test",
    secretAccessKey: "test",
  },
});
```

## Python (boto3)

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

# DynamoDB
dynamodb = session.resource("dynamodb", endpoint_url="http://localhost:4566")
table = dynamodb.create_table(
    TableName="Users",
    KeySchema=[{"AttributeName": "UserId", "KeyType": "HASH"}],
    AttributeDefinitions=[{"AttributeName": "UserId", "AttributeType": "S"}],
    BillingMode="PAY_PER_REQUEST",
)
```

Each client or resource call takes its own `endpoint_url`. If you prefer a single configuration point, set `AWS_ENDPOINT_URL` as an environment variable and omit the parameter.

## Go (aws-sdk-go-v2)

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
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider("test", "test", ""),
        ),
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

`config.WithBaseEndpoint` sets the endpoint for all services created from this config. `UsePathStyle` is required for S3 only.

## Java (AWS SDK v2)

```java
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.CreateBucketRequest;

import java.net.URI;

public class CloudMockExample {
    public static void main(String[] args) {
        S3Client s3 = S3Client.builder()
            .endpointOverride(URI.create("http://localhost:4566"))
            .region(Region.US_EAST_1)
            .credentialsProvider(StaticCredentialsProvider.create(
                AwsBasicCredentials.create("test", "test")
            ))
            .forcePathStyle(true)
            .build();

        s3.createBucket(CreateBucketRequest.builder()
            .bucket("my-bucket")
            .build());

        System.out.println("Bucket created");
    }
}
```

Add the S3 dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>software.amazon.awssdk</groupId>
    <artifactId>s3</artifactId>
    <version>2.31.1</version>
</dependency>
```

## Tips

**Keep production code clean.** Rather than hardcoding `localhost:4566`, use the environment variable approach. Your application reads `AWS_ENDPOINT_URL` automatically -- set it in development, leave it unset in production.

**S3 path style.** CloudMock requires path-style S3 URLs (`localhost:4566/bucket`) rather than virtual-hosted-style (`bucket.localhost:4566`). All the examples above set the path style option. This does not affect non-S3 services.

**Same credentials everywhere.** The default root credentials are `test` / `test`. You can change them in `cloudmock.yml` under `iam.root_access_key` and `iam.root_secret_key`, or disable authentication entirely with `iam.mode: none`.

**All 25 Tier 1 services use the same endpoint.** CloudMock routes requests to the correct service based on the AWS service headers. You do not need per-service ports or endpoints.

**See it in practice.** The [todo demo project](https://github.com/Viridian-Inc/cloudmock-todo-demo) shows complete working examples of S3, DynamoDB, SQS, and SNS in Node.js, Python, and Go.
