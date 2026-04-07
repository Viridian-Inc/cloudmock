---
title: Traffic Recording and Replay
description: Record real AWS traffic, replay it against CloudMock, and validate compatibility in CI
---

CloudMock's traffic replay engine lets you capture real AWS API calls, replay them against a CloudMock instance, and compare results for compatibility validation. This is useful for verifying that CloudMock behaves the same as real AWS for your specific workload.

## Overview

The traffic replay workflow has three stages:

1. **Record** -- Capture live AWS traffic using the recording proxy or Go SDK interceptor
2. **Replay** -- Send the recorded requests to CloudMock
3. **Validate** -- Compare responses and generate a compatibility report

---

## Record mode (proxy)

The recording proxy sits between your application and AWS. It forwards every request to the real AWS endpoint, captures the request and response, and returns the real response to your application.

```bash
# Start the recording proxy on port 4577
cloudmock record --output recording.json --region us-east-1 --port 4577
```

Then point your AWS SDK at the proxy:

```bash
export AWS_ENDPOINT_URL=http://localhost:4577
# Run your application normally -- all traffic is recorded
./my-app
# Press Ctrl+C to stop and save
```

The proxy detects the target AWS service from the SigV4 Authorization header and routes to the correct endpoint (e.g., `s3.us-east-1.amazonaws.com`, `dynamodb.us-east-1.amazonaws.com`).

### Output format

The recording is saved as a JSON file containing a `Recording` object with an array of `CapturedEntry` values. Each entry includes:

- Request method, path, headers, and body
- Response status code and body
- Service name and action
- Latency and timing offset

---

## Replay mode

Replay sends each recorded request to a CloudMock instance:

```bash
cloudmock replay --input recording.json --endpoint http://localhost:4566
```

The replay prints a summary of matched vs mismatched status codes.

---

## Validate mode (CI)

The validate command replays traffic and compares responses, returning a non-zero exit code if there are mismatches:

```bash
cloudmock validate --input recording.json --endpoint http://localhost:4566
```

Use `--strict` to enable deep JSON body comparison (not just status codes):

```bash
cloudmock validate --input recording.json --endpoint http://localhost:4566 --strict
```

Fields like `RequestId` and `ResponseMetadata` are automatically ignored since they vary between calls.

### Exit codes

- `0` -- All requests matched
- `1` -- One or more mismatches or errors

---

## Go SDK interceptor

For Go applications, you can record traffic at the SDK level without a proxy by wrapping the HTTP transport:

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    cmsdk "github.com/Viridian-Inc/cloudmock/sdk"
)

func main() {
    recorder := cmsdk.NewRecorder()

    cfg, _ := config.LoadDefaultConfig(context.TODO(),
        config.WithHTTPClient(&http.Client{
            Transport: recorder,
        }),
    )

    client := s3.NewFromConfig(cfg)
    client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})

    // Save the recording
    recorder.SaveToFile("recording.json")
    fmt.Printf("Captured %d entries\n", len(recorder.Entries()))
}
```

The interceptor captures every request and response transparently. Your application receives the original AWS response unchanged.

---

## Compatibility report format

The comparison engine produces a `ComparisonReport`:

```json
{
  "total_requests": 50,
  "matched": 48,
  "mismatched": 2,
  "errors": 0,
  "compatibility_pct": 96.0,
  "mismatches": [
    {
      "entry_id": "proxy-12",
      "service": "s3",
      "action": "GetObject",
      "original_status": 200,
      "replay_status": 404,
      "diffs": ["status: 200 -> 404"],
      "severity": "status"
    }
  ]
}
```

Severity levels:
- **status** -- HTTP status code differs
- **data** -- Response body values differ
- **schema** -- Response body has missing or extra fields

---

## Entry injection API

You can inject entries directly into CloudMock via the admin API:

```bash
curl -X POST http://localhost:4599/api/traffic/entries \
  -H 'Content-Type: application/json' \
  -d '[{"id":"test-1","service":"s3","action":"GetObject","method":"GET","path":"/my-bucket/key","status_code":200}]'
```

This creates an ad-hoc recording from the provided entries.

---

## Example CI workflow

```yaml
name: CloudMock Compatibility
on: [push]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Start CloudMock
        run: |
          docker run -d --name cloudmock -p 4566:4566 ghcr.io/neureaux/cloudmock:latest
          sleep 2

      - name: Validate recording
        run: |
          cloudmock validate \
            --input tests/fixtures/recording.json \
            --endpoint http://localhost:4566 \
            --strict

      - name: Stop CloudMock
        if: always()
        run: docker rm -f cloudmock
```

This fails the build if CloudMock's responses diverge from the recorded AWS behavior.
