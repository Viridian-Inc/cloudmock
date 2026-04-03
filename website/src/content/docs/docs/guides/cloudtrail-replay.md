---
title: CloudTrail Event Replay
description: Recreate production AWS state in CloudMock by replaying CloudTrail audit logs
---

CloudTrail event replay lets you take real AWS audit logs and replay the write operations against a CloudMock instance. This recreates your production resource topology locally -- tables, queues, buckets, topics, and more -- without manual setup.

## How it works

1. **Export** CloudTrail events from your AWS account
2. **Parse** the JSON log file to extract write operations
3. **Convert** each event into the correct AWS wire-protocol request (JSON, Query, or REST)
4. **Replay** the requests against CloudMock in chronological order

CloudMock handles the protocol conversion automatically. A DynamoDB `CreateTable` event becomes a JSON-protocol POST with the correct `X-Amz-Target` header. An SQS `CreateQueue` becomes a Query-protocol POST with `Action=CreateQueue`. An S3 `CreateBucket` becomes a REST `PUT /bucket-name`.

## Getting CloudTrail logs

Export recent events using the AWS CLI:

```bash
aws cloudtrail lookup-events \
  --start-time 2026-03-01T00:00:00Z \
  --end-time 2026-04-01T00:00:00Z \
  --output json > trail.json
```

Or download a CloudTrail log file from S3 if you have a trail configured:

```bash
aws s3 cp s3://my-trail-bucket/AWSLogs/123456789012/CloudTrail/us-east-1/2026/04/01/trail.json.gz .
gunzip trail.json.gz
```

The file must contain a top-level `Records` array in the standard CloudTrail format.

## Replaying locally

Start CloudMock, then replay:

```bash
# Start CloudMock
npx cloudmock

# Replay CloudTrail events (instant mode)
cloudmock cloudtrail replay --input trail.json --endpoint http://localhost:4566
```

Output:

```
Replaying 847 CloudTrail events against http://localhost:4566

CloudTrail Replay Results
  Total events:  847
  Replayed:      312
  Skipped:       535
  Succeeded:     308
  Failed:        4
  Duration:      1.2s
```

Skipped events are read-only operations (DescribeTable, GetObject, etc.) that do not modify state.

## Filtering by service

Replay only specific services:

```bash
cloudmock cloudtrail replay \
  --input trail.json \
  --services dynamodb,s3,sqs
```

## Speed control

By default, events replay as fast as possible (`--speed 0`). To replay at real-time speed:

```bash
cloudmock cloudtrail replay --input trail.json --speed 1.0
```

Use `--speed 2.0` for double speed, or `--speed 0.5` for half speed.

## Saving results

Write the replay result to a JSON file:

```bash
cloudmock cloudtrail replay --input trail.json --output result.json
```

## Admin API

You can also replay via the admin API:

```bash
curl -X POST http://localhost:4599/api/cloudtrail/replay \
  -H "Content-Type: application/json" \
  -d @trail.json
```

The response is a JSON object with `total_events`, `replayed`, `skipped`, `succeeded`, `failed`, and `errors` fields.

## CI workflow

Use CloudTrail replay in CI to bootstrap a realistic CloudMock environment before running integration tests:

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: viridian-inc/cloudmock-action@v1

      - name: Replay production state
        run: cloudmock cloudtrail replay --input fixtures/trail.json --output replay-result.json

      - name: Run integration tests
        run: npm test
        env:
          AWS_ENDPOINT_URL: http://localhost:4566
```

## Supported services

CloudTrail replay supports the following services and their most common write operations:

| Service | Example events |
|---------|---------------|
| DynamoDB | CreateTable, PutItem, UpdateItem, DeleteTable |
| S3 | CreateBucket, PutObject, DeleteBucket, DeleteObject |
| SQS | CreateQueue, SendMessage, DeleteQueue |
| SNS | CreateTopic, Subscribe, Publish |
| IAM | CreateRole, CreateUser |
| KMS | CreateKey |
| Lambda | CreateFunction, Invoke |
| CloudWatch Logs | CreateLogGroup |
| Kinesis | CreateStream |
| EC2 | RunInstances |
| CloudFormation | CreateStack |
| STS | GetCallerIdentity |

Unsupported events are silently skipped during replay.
