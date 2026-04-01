# Quickstart

Get CloudMock running and see your first request in DevTools in under 5 minutes.

## Prerequisites

- Node.js 18+ (for `npx`) or Docker
- An application that uses the AWS SDK (any language)

## Step 1: Install and Start

```bash
npx cloudmock
```

This downloads the CloudMock binary and starts it. You'll see:

```
Starting CloudMock...
  Gateway:   http://localhost:4566
  Admin API: http://localhost:4599
  Dashboard: http://localhost:4500
  Profile:   standard
  Region:    us-east-1

CloudMock started (PID 12345)
```

Alternatively, install the CLI separately:

```bash
brew install cloudmock
cmk start
```

## Step 2: Point Your App at CloudMock

Set one environment variable and your existing AWS SDK calls go through CloudMock:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
```

**Node.js:**

```js
// No code changes needed -- AWS SDK v3 reads AWS_ENDPOINT_URL automatically
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3';

const s3 = new S3Client({ region: 'us-east-1' });
await s3.send(new PutObjectCommand({
  Bucket: 'my-bucket',
  Key: 'hello.txt',
  Body: 'Hello from CloudMock!',
}));
```

**Python:**

```python
import boto3

# boto3 reads AWS_ENDPOINT_URL automatically (v1.28+)
s3 = boto3.client('s3', region_name='us-east-1')
s3.create_bucket(Bucket='my-bucket')
s3.put_object(Bucket='my-bucket', Key='hello.txt', Body=b'Hello from CloudMock!')
```

**Go:**

```go
cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
// AWS SDK Go v2 reads AWS_ENDPOINT_URL automatically
client := s3.NewFromConfig(cfg)
client.PutObject(context.TODO(), &s3.PutObjectInput{
    Bucket: aws.String("my-bucket"),
    Key:    aws.String("hello.txt"),
    Body:   strings.NewReader("Hello from CloudMock!"),
})
```

**Java:**

```java
S3Client s3 = S3Client.builder()
    .region(Region.US_EAST_1)
    .endpointOverride(URI.create("http://localhost:4566"))
    .build();

s3.putObject(
    PutObjectRequest.builder().bucket("my-bucket").key("hello.txt").build(),
    RequestBody.fromString("Hello from CloudMock!")
);
```

## Step 3: Open DevTools

Open your browser to:

```
http://localhost:4500
```

You'll see every AWS request your app makes in real time -- the service, action, duration, status, and full request/response payloads.

## Step 4: Add Observability (Optional)

To see distributed traces from your application code (not just AWS calls), point OpenTelemetry at CloudMock:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/json
```

See the [OpenTelemetry guide](../guides/opentelemetry.md) for language-specific setup.

## Step 5: Explore

With CloudMock running, try:

- **Request Inspector** -- click any request to see full payload, headers, and timing
- **Trace Viewer** -- see distributed traces across services
- **Topology Map** -- visualize your architecture (auto-discovered from IaC)
- **Chaos Controls** -- inject faults to test error handling
- **Cost Dashboard** -- see estimated AWS costs from your usage patterns

## What's Running

After `cmk start`, you have:

| Component | URL | Purpose |
|-----------|-----|---------|
| Gateway | `http://localhost:4566` | AWS API endpoint -- point `AWS_ENDPOINT_URL` here |
| Dashboard | `http://localhost:4500` | DevTools web UI |
| Admin API | `http://localhost:4599` | Programmatic access to all DevTools data |
| OTLP | `http://localhost:4318` | OpenTelemetry receiver for traces, metrics, logs |

## Next Steps

- [Configuration](configuration.md) -- customize ports, profiles, persistence
- [OpenTelemetry](../guides/opentelemetry.md) -- add tracing to your app in any language
- [Error Tracking](../guides/error-tracking.md) -- capture and group errors
- [Service Compatibility](../reference/services.md) -- see all 98 supported AWS services
- [Docker Deployment](../deployment/docker.md) -- run in Docker Compose alongside your app
