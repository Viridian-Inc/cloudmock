# Go Integration

Go works with CloudMock through the standard AWS SDK for Go v2 and OpenTelemetry. No CloudMock-specific SDK is needed.

## AWS SDK for Go v2

### Setup

```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
```

Point the SDK at CloudMock:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
```

AWS SDK for Go v2 reads `AWS_ENDPOINT_URL` automatically.

### Examples

**S3:**

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    ctx := context.Background()
    cfg, _ := config.LoadDefaultConfig(ctx)
    client := s3.NewFromConfig(cfg)

    // Create bucket
    client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String("my-bucket"),
    })

    // Upload
    client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("hello.txt"),
        Body:   strings.NewReader("Hello from Go!"),
    })

    // Download
    result, _ := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("hello.txt"),
    })
    defer result.Body.Close()
    fmt.Println("Downloaded:", result.ContentLength)
}
```

**DynamoDB:**

```go
package main

import (
    "context"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func main() {
    ctx := context.Background()
    cfg, _ := config.LoadDefaultConfig(ctx)
    client := dynamodb.NewFromConfig(cfg)

    // Create table
    client.CreateTable(ctx, &dynamodb.CreateTableInput{
        TableName: aws.String("users"),
        KeySchema: []types.KeySchemaElement{
            {AttributeName: aws.String("userId"), KeyType: types.KeyTypeHash},
        },
        AttributeDefinitions: []types.AttributeDefinition{
            {AttributeName: aws.String("userId"), AttributeType: types.ScalarAttributeTypeS},
        },
        BillingMode: types.BillingModePayPerRequest,
    })

    // Put item
    client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String("users"),
        Item: map[string]types.AttributeValue{
            "userId": &types.AttributeValueMemberS{Value: "user-1"},
            "name":   &types.AttributeValueMemberS{Value: "Alice"},
        },
    })

    // Get item
    result, _ := client.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String("users"),
        Key: map[string]types.AttributeValue{
            "userId": &types.AttributeValueMemberS{Value: "user-1"},
        },
    })
    fmt.Println("Got:", result.Item)
}
```

## OpenTelemetry

### Setup

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go get go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```

### Initialize Tracer

```go
package tracing

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func InitTracer(ctx context.Context, serviceName string) (func(), error) {
    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )
    otel.SetTracerProvider(tp)

    return func() { tp.Shutdown(ctx) }, nil
}
```

### Instrument AWS SDK

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

cfg, _ := config.LoadDefaultConfig(ctx)
otelaws.AppendMiddlewares(&cfg.APIOptions)  // Add OTel middleware

// Now all AWS SDK calls appear as spans in CloudMock
client := dynamodb.NewFromConfig(cfg)
```

### Instrument HTTP Server

```go
import (
    "net/http"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

mux := http.NewServeMux()
mux.HandleFunc("/orders", handleOrders)

// Wrap with OTel middleware
handler := otelhttp.NewHandler(mux, "my-server")
http.ListenAndServe(":8080", handler)
```

### Custom Spans

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("order-service")

func processOrder(ctx context.Context, order Order) error {
    ctx, span := tracer.Start(ctx, "process-order",
        trace.WithAttributes(
            attribute.String("order.id", order.ID),
            attribute.Float64("order.total", order.Total),
        ),
    )
    defer span.End()

    if err := chargePayment(ctx, order); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("payment failed: %w", err)
    }

    return sendConfirmation(ctx, order)
}
```

## Full Example: HTTP API + DynamoDB

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel"
)

var db *dynamodb.Client
var tracer = otel.Tracer("order-api")

func main() {
    ctx := context.Background()

    // Init tracing
    shutdown, _ := tracing.InitTracer(ctx, "order-api")
    defer shutdown()

    // Init AWS
    cfg, _ := config.LoadDefaultConfig(ctx)
    otelaws.AppendMiddlewares(&cfg.APIOptions)
    db = dynamodb.NewFromConfig(cfg)

    // Routes
    mux := http.NewServeMux()
    mux.HandleFunc("POST /orders", createOrder)
    mux.HandleFunc("GET /orders/{id}", getOrder)

    log.Println("Listening on :8080")
    http.ListenAndServe(":8080", otelhttp.NewHandler(mux, "order-api"))
}

func createOrder(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "create-order")
    defer span.End()

    var order struct {
        ID       string  `json:"id"`
        Customer string  `json:"customer"`
        Total    float64 `json:"total"`
    }
    json.NewDecoder(r.Body).Decode(&order)

    db.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String("orders"),
        Item: map[string]types.AttributeValue{
            "orderId":  &types.AttributeValueMemberS{Value: order.ID},
            "customer": &types.AttributeValueMemberS{Value: order.Customer},
            "status":   &types.AttributeValueMemberS{Value: "pending"},
        },
    })

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"id": order.ID, "status": "pending"})
}

func getOrder(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    result, err := db.GetItem(r.Context(), &dynamodb.GetItemInput{
        TableName: aws.String("orders"),
        Key: map[string]types.AttributeValue{
            "orderId": &types.AttributeValueMemberS{Value: id},
        },
    })
    if err != nil || result.Item == nil {
        http.Error(w, "not found", 404)
        return
    }
    json.NewEncoder(w).Encode(result.Item)
}
```

Run:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
go run .
```

DevTools at `http://localhost:4500` shows HTTP requests, DynamoDB operations, and distributed traces.
