# Chaos Engineering

CloudMock includes built-in fault injection for testing how your application handles AWS service failures. Inject errors, latency, and throttling without modifying your application code.

## Concepts

A **chaos rule** defines a fault to inject on matching requests. Rules specify:

- **Target** -- which service and action to affect
- **Fault type** -- error, latency, or throttling
- **Probability** -- percentage of requests to affect (0.0 to 1.0)
- **Duration** -- how long the rule stays active

## Creating Chaos Rules

### Inject Errors

Make DynamoDB return `InternalServerError` on 50% of PutItem calls:

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "dynamodb",
    "action": "PutItem",
    "fault": "error",
    "error_code": "InternalServerError",
    "error_message": "Simulated internal error",
    "probability": 0.5,
    "duration": "5m"
  }'
```

### Inject Latency

Add 2 seconds of latency to all S3 GetObject calls:

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "s3",
    "action": "GetObject",
    "fault": "latency",
    "latency_ms": 2000,
    "probability": 1.0,
    "duration": "10m"
  }'
```

### Inject Throttling

Simulate SQS throttling at 30% of requests:

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "sqs",
    "action": "*",
    "fault": "throttle",
    "probability": 0.3,
    "duration": "5m"
  }'
```

### Target All Actions on a Service

Use `"action": "*"` to affect every action on a service:

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "lambda",
    "action": "*",
    "fault": "latency",
    "latency_ms": 5000,
    "probability": 0.2,
    "duration": "5m"
  }'
```

## Managing Rules

```bash
# List active chaos rules
curl http://localhost:4599/api/chaos | jq '.'

# Delete a specific rule
curl -X DELETE http://localhost:4599/api/chaos/rule_abc123

# Delete all rules (stop all chaos)
curl -X DELETE http://localhost:4599/api/chaos
```

## Fault Types

| Fault | Description | Parameters |
|-------|-------------|------------|
| `error` | Return an AWS error response | `error_code`, `error_message` |
| `latency` | Add delay before responding | `latency_ms` |
| `throttle` | Return `ThrottlingException` | (none -- uses standard throttling response) |

### Common Error Codes

| Service | Error Code | Meaning |
|---------|-----------|---------|
| DynamoDB | `InternalServerError` | Server-side error |
| DynamoDB | `ProvisionedThroughputExceededException` | Capacity exceeded |
| DynamoDB | `ConditionalCheckFailedException` | Condition expression failed |
| S3 | `InternalError` | Server-side error |
| S3 | `SlowDown` | Rate limiting |
| S3 | `ServiceUnavailable` | Temporary unavailability |
| Lambda | `ServiceException` | Internal error |
| Lambda | `TooManyRequestsException` | Concurrent execution limit |
| SQS | `InternalError` | Server-side error |
| SQS | `OverLimit` | Queue limit exceeded |

## Use Cases

### Test Retry Logic

Inject intermittent errors and verify your SDK client retries correctly:

```bash
# 30% of DynamoDB writes fail
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "dynamodb",
    "action": "PutItem",
    "fault": "error",
    "error_code": "ProvisionedThroughputExceededException",
    "probability": 0.3,
    "duration": "2m"
  }'

# Run your tests, then check: did all items eventually get written?
```

### Test Timeout Handling

Add latency beyond your client's timeout:

```bash
# S3 uploads take 30 seconds
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "s3",
    "action": "PutObject",
    "fault": "latency",
    "latency_ms": 30000,
    "probability": 1.0,
    "duration": "2m"
  }'
```

### Test Circuit Breaker

Cause a complete outage and verify your circuit breaker opens:

```bash
# DynamoDB is completely down
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "dynamodb",
    "action": "*",
    "fault": "error",
    "error_code": "InternalServerError",
    "probability": 1.0,
    "duration": "5m"
  }'
```

### Test Graceful Degradation

Simulate a Lambda cold start with high latency:

```bash
curl -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "lambda",
    "action": "Invoke",
    "fault": "latency",
    "latency_ms": 10000,
    "probability": 0.1,
    "duration": "5m"
  }'
```

## DevTools Integration

Active chaos rules appear in the DevTools dashboard:

- **Chaos indicator** -- a warning badge when chaos rules are active
- **Request tagging** -- requests affected by chaos are marked in the request list
- **Impact metrics** -- see how fault injection affects error rates and latency in real time

## Automation

Run chaos tests as part of your CI pipeline:

```bash
#!/bin/bash
# chaos-test.sh

# Start CloudMock
cmk start
sleep 2

# Inject faults
RULE_ID=$(curl -s -X POST http://localhost:4599/api/chaos \
  -H "Content-Type: application/json" \
  -d '{
    "service": "dynamodb",
    "action": "GetItem",
    "fault": "error",
    "error_code": "InternalServerError",
    "probability": 0.5,
    "duration": "5m"
  }' | jq -r '.id')

# Run your test suite
npm test

# Clean up
curl -X DELETE "http://localhost:4599/api/chaos/$RULE_ID"
cmk stop
```
