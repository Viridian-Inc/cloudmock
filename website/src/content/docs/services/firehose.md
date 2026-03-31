---
title: Data Firehose
description: Amazon Data Firehose emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Data Firehose, supporting delivery stream management, record ingestion (single and batch), destination configuration, and tagging. Records are stored in memory without delivery to destinations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDeliveryStream | Supported | Creates a delivery stream with S3, Redshift, or OpenSearch destination |
| DeleteDeliveryStream | Supported | Deletes a delivery stream |
| DescribeDeliveryStream | Supported | Returns stream metadata and configuration |
| ListDeliveryStreams | Supported | Returns all delivery stream names |
| PutRecord | Supported | Writes a single record to a delivery stream |
| PutRecordBatch | Supported | Writes up to 500 records in one call |
| UpdateDestination | Supported | Updates the destination configuration |
| TagDeliveryStream | Supported | Adds tags to a stream |
| UntagDeliveryStream | Supported | Removes tags |
| ListTagsForDeliveryStream | Supported | Returns tags for a stream |

## Quick Start

### curl

```bash
# Create a delivery stream
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: Firehose_20150804.CreateDeliveryStream" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{
    "DeliveryStreamName": "logs-to-s3",
    "S3DestinationConfiguration": {
      "RoleARN": "arn:aws:iam::000000000000:role/firehose-role",
      "BucketARN": "arn:aws:s3:::my-logs-bucket"
    }
  }'

# Put a record
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: Firehose_20150804.PutRecord" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"DeliveryStreamName": "logs-to-s3", "Record": {"Data": "eyJldmVudCI6InBhZ2V2aWV3In0="}}'
```

### Node.js

```typescript
import { FirehoseClient, CreateDeliveryStreamCommand, PutRecordBatchCommand } from '@aws-sdk/client-firehose';

const firehose = new FirehoseClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await firehose.send(new CreateDeliveryStreamCommand({
  DeliveryStreamName: 'events',
  S3DestinationConfiguration: {
    RoleARN: 'arn:aws:iam::000000000000:role/fh-role',
    BucketARN: 'arn:aws:s3:::events-bucket',
  },
}));

await firehose.send(new PutRecordBatchCommand({
  DeliveryStreamName: 'events',
  Records: [
    { Data: Buffer.from(JSON.stringify({ event: 'signup' })) },
    { Data: Buffer.from(JSON.stringify({ event: 'login' })) },
  ],
}));
```

### Python

```python
import boto3, json

firehose = boto3.client('firehose', endpoint_url='http://localhost:4566',
                        aws_access_key_id='test', aws_secret_access_key='test',
                        region_name='us-east-1')

firehose.create_delivery_stream(
    DeliveryStreamName='events',
    S3DestinationConfiguration={
        'RoleARN': 'arn:aws:iam::000000000000:role/fh-role',
        'BucketARN': 'arn:aws:s3:::events-bucket',
    },
)

firehose.put_record_batch(
    DeliveryStreamName='events',
    Records=[
        {'Data': json.dumps({'event': 'signup', 'userId': 'u-1'})},
        {'Data': json.dumps({'event': 'login', 'userId': 'u-2'})},
    ],
)

stream = firehose.describe_delivery_stream(DeliveryStreamName='events')
print(stream['DeliveryStreamDescription']['DeliveryStreamStatus'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  firehose:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Records are stored **in memory**. Delivery to S3 or other destinations is not performed.
- **Buffering and compression** settings are accepted but have no effect.
- **Dynamic partitioning** and data transformation via Lambda are not implemented.
- `PutRecordBatch` returns all records as successfully written.
- **Server-side encryption** is accepted but not enforced.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified delivery stream does not exist |
| ResourceInUseException | 400 | The delivery stream is being created or deleted |
| InvalidArgumentException | 400 | An argument is not valid |
| ServiceUnavailableException | 503 | The service is temporarily unavailable |
