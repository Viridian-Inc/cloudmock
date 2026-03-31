---
title: Go
description: Using CloudMock with Go, the cloudmock Go SDK, and aws-sdk-go-v2
---

Go is a first-class language for CloudMock (CloudMock itself is written in Go). The `cloudmock` Go SDK provides HTTP interceptor middleware and trace propagation. For AWS-only usage, configure `aws-sdk-go-v2` to point at the CloudMock gateway.

## cloudmock Go SDK

### Install

```bash
go get github.com/neureaux/cloudmock/sdk/go
```

### Initialize

Call `cloudmock.Init()` early in your application startup. This registers an HTTP transport interceptor that captures outgoing AWS API calls and forwards telemetry to the CloudMock admin API.

```go
package main

import (
    "log"
    cloudmock "github.com/neureaux/cloudmock/sdk/go"
)

func main() {
    err := cloudmock.Init(cloudmock.Config{
        AdminURL:    "http://localhost:4599",
        ServiceName: "my-api",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cloudmock.Shutdown()

    // Your application code here
}
```

### HTTP interceptor

The SDK wraps Go's `http.DefaultTransport` to intercept outgoing requests to the CloudMock gateway. All AWS SDK calls that use the default HTTP client are automatically captured.

If you use a custom `http.Client`, wrap its transport:

```go
client := &http.Client{
    Transport: cloudmock.WrapTransport(http.DefaultTransport),
}
```

### net/http middleware

For HTTP servers, the SDK provides middleware that traces inbound requests:

```go
import "github.com/neureaux/cloudmock/sdk/go"

mux := http.NewServeMux()
mux.HandleFunc("/users", handleUsers)

handler := cloudmock.Middleware(mux)
http.ListenAndServe(":8080", handler)
```

### What gets captured

- **Inbound HTTP requests** -- Method, path, status code, duration.
- **Outbound AWS calls** -- Service, action, latency, status code. Automatically detected from request headers.
- **Trace context** -- A trace ID is propagated from inbound requests to outbound AWS calls.
- **Service identity** -- The `ServiceName` appears as a node in the Topology view.

## aws-sdk-go-v2 endpoint configuration

If you do not need the CloudMock Go SDK (for example, in a CLI tool or Lambda function), configure `aws-sdk-go-v2` directly:

### Environment variable

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

### Programmatic configuration

```go
package main

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

    // S3 -- must use path style
    s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle = true
    })

    _, err = s3Client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
        Bucket: aws.String("my-bucket"),
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("S3 bucket created")

    // DynamoDB
    ddbClient := dynamodb.NewFromConfig(cfg)
    _, err = ddbClient.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
        TableName: aws.String("Users"),
        KeySchema: []types.KeySchemaElement{
            {AttributeName: aws.String("UserId"), KeyType: types.KeyTypeHash},
        },
        AttributeDefinitions: []types.AttributeDefinition{
            {AttributeName: aws.String("UserId"), AttributeType: types.ScalarAttributeTypeS},
        },
        BillingMode: types.BillingModePayPerRequest,
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("DynamoDB table created")
}
```

### Conditional endpoint

```go
func loadConfig() (aws.Config, error) {
    opts := []func(*config.LoadOptions) error{
        config.WithRegion("us-east-1"),
    }

    if endpoint := os.Getenv("CLOUDMOCK_ENDPOINT"); endpoint != "" {
        opts = append(opts,
            config.WithBaseEndpoint(endpoint),
            config.WithCredentialsProvider(
                credentials.NewStaticCredentialsProvider("test", "test", ""),
            ),
        )
    }

    return config.LoadDefaultConfig(context.TODO(), opts...)
}
```

## Common issues

### S3 path style

CloudMock requires path-style S3 access. Always set `UsePathStyle: true` on the S3 client options. Virtual-hosted style URLs (e.g., `bucket.localhost`) are not supported.

### Module version

The `config.WithBaseEndpoint` function requires `aws-sdk-go-v2` v1.25.0 or later. Earlier versions use per-service endpoint resolvers, which are more verbose to configure.
