---
title: Contract Testing
description: Validate CloudMock fidelity against real AWS with the dual-mode contract proxy
---

Contract testing sends every request to both real AWS and CloudMock simultaneously, compares the responses, and produces a compatibility report. This lets you prove that CloudMock behaves identically to real AWS for your specific workload -- without maintaining separate test suites.

## Why contract testing

Unit tests verify your application logic. Integration tests verify that your code works with CloudMock. Contract tests close the remaining gap: they verify that **CloudMock itself** returns the same responses as real AWS for the exact API calls your application makes.

This is especially useful when:
- Upgrading CloudMock versions and wanting to confirm nothing regressed
- Onboarding a new AWS service and verifying coverage
- Running in CI to catch fidelity issues before they reach production

## Quick start

Start a CloudMock instance and the contract proxy:

```bash
# Terminal 1: Start CloudMock
npx cloudmock

# Terminal 2: Start the contract proxy
cloudmock contract \
  --cloudmock http://localhost:4566 \
  --port 4577 \
  --output report.json
```

Point your application or tests at the contract proxy:

```bash
export AWS_ENDPOINT_URL=http://localhost:4577
npm test
```

Press `Ctrl+C` to stop the proxy. The compatibility report is written to `report.json`.

## The --run flag

For CI pipelines, use `--run` to execute a command automatically. The proxy starts, runs the command with `AWS_ENDPOINT_URL` set, and exits with code 0 if all responses match or 1 if there are mismatches:

```bash
cloudmock contract \
  --cloudmock http://localhost:4566 \
  --port 4577 \
  --output report.json \
  --run "npm test"
```

The `AWS_ENDPOINT_URL` environment variable is automatically set to the proxy address (`http://localhost:4577`) for the child process.

## Ignoring known differences

Some fields differ between real AWS and CloudMock by design -- for example `RequestId` values and `ResponseMetadata` timestamps. Use `--ignore-paths` to exclude them from comparison:

```bash
cloudmock contract \
  --ignore-paths RequestId,ResponseMetadata,Date \
  --run "npm test"
```

Paths are matched by JSON key name at any nesting depth. For example, `RequestId` ignores both a top-level `RequestId` and `ResponseMetadata.RequestId`.

## Reading the compatibility report

The report is a JSON file with this structure:

```json
{
  "started_at": "2026-04-01T10:00:00Z",
  "duration_sec": 12.5,
  "total_requests": 47,
  "matched": 45,
  "mismatched": 2,
  "compatibility_pct": 95.7,
  "by_service": {
    "dynamodb": { "total": 30, "matched": 30, "pct": 100 },
    "s3": { "total": 17, "matched": 15, "pct": 88.2 }
  },
  "mismatches": [
    {
      "service": "s3",
      "action": "PutObject",
      "aws_status": 200,
      "cloudmock_status": 200,
      "diffs": ["ETag: \"abc123\" -> \"def456\""],
      "severity": "data"
    }
  ]
}
```

**Severity levels:**
- `status` -- HTTP status codes differ (most critical)
- `schema` -- response structure differs (missing or extra keys)
- `data` -- values differ but structure is the same
- `error` -- one side returned an error or was unreachable

## CI integration with GitHub Actions

```yaml
name: Contract Tests
on: [push]
jobs:
  contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1

      - name: Run contract tests
        run: |
          cloudmock contract \
            --cloudmock http://localhost:4566 \
            --port 4577 \
            --output contract-report.json \
            --ignore-paths RequestId,ResponseMetadata \
            --run "npm test"
        env:
          AWS_ACCESS_KEY_ID: test
          AWS_SECRET_ACCESS_KEY: test
          AWS_DEFAULT_REGION: us-east-1

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: contract-report
          path: contract-report.json
```

The job fails if any API responses diverge between real AWS and CloudMock, giving you a clear signal in CI.

## Go SDK contract test helper

You can also use the contract proxy programmatically in Go tests:

```go
package myapp_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/Viridian-Inc/cloudmock/pkg/proxy"
    "github.com/stretchr/testify/assert"
)

func TestAWSCompatibility(t *testing.T) {
    cp := proxy.NewContractProxy("us-east-1", "http://localhost:4566", []string{
        "RequestId", "ResponseMetadata",
    })
    srv := httptest.NewServer(cp)
    defer srv.Close()

    // Point your AWS SDK at srv.URL and run your tests...

    report := cp.Report()
    assert.Equal(t, 100.0, report.CompatibilityPct,
        "CloudMock should match real AWS for all requests")
}
```

## CLI reference

```
cloudmock contract [options]

Options:
  --cloudmock URL       CloudMock endpoint (default: http://localhost:4566)
  --region REGION       AWS region (default: us-east-1)
  --port PORT           Local proxy port (default: 4577)
  --output PATH         Report output path (default: contract-report.json)
  --ignore-paths PATHS  Comma-separated JSON paths to ignore (default: RequestId,ResponseMetadata)
  --run COMMAND         Execute command with proxy, then exit
```
