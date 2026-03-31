---
title: Kinesis
description: Amazon Kinesis Data Streams emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Kinesis Data Streams, supporting stream lifecycle, record ingestion (single and batch), shard iteration with sequential reads, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateStream | Supported | Creates a stream with the specified shard count |
| DeleteStream | Supported | Deletes a stream and all its data |
| DescribeStream | Supported | Returns stream metadata and shard details |
| ListStreams | Supported | Returns all stream names |
| PutRecord | Supported | Writes a single record to a stream |
| PutRecords | Supported | Writes up to 500 records in one call |
| GetShardIterator | Supported | Returns a shard iterator for reading |
| GetRecords | Supported | Returns records from a shard iterator |
| IncreaseStreamRetentionPeriod | Supported | Sets retention period (hours) |
| DecreaseStreamRetentionPeriod | Supported | Reduces retention period |
| AddTagsToStream | Supported | Adds tags to a stream |
| RemoveTagsFromStream | Supported | Removes tags |
| ListTagsForStream | Supported | Returns tags for a stream |

## Quick Start

### curl

```bash
# Create a stream
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: Kinesis_20131202.CreateStream" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"StreamName": "events", "ShardCount": 1}'

# Put a record
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: Kinesis_20131202.PutRecord" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"StreamName": "events", "Data": "eyJldmVudCI6ImNsaWNrIn0=", "PartitionKey": "user-123"}'
```

### Node.js

```typescript
import { KinesisClient, CreateStreamCommand, PutRecordsCommand, GetShardIteratorCommand, GetRecordsCommand } from '@aws-sdk/client-kinesis';

const kinesis = new KinesisClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await kinesis.send(new CreateStreamCommand({ StreamName: 'orders', ShardCount: 1 }));
await kinesis.send(new PutRecordsCommand({
  StreamName: 'orders',
  Records: [
    { Data: Buffer.from(JSON.stringify({ orderId: 'o-1' })), PartitionKey: 'p1' },
    { Data: Buffer.from(JSON.stringify({ orderId: 'o-2' })), PartitionKey: 'p2' },
  ],
}));
```

### Python

```python
import boto3, json

kinesis = boto3.client('kinesis', endpoint_url='http://localhost:4566',
                       aws_access_key_id='test', aws_secret_access_key='test',
                       region_name='us-east-1')

kinesis.create_stream(StreamName='orders', ShardCount=1)
kinesis.put_records(
    StreamName='orders',
    Records=[
        {'Data': json.dumps({'orderId': 'o-1'}), 'PartitionKey': 'p1'},
        {'Data': json.dumps({'orderId': 'o-2'}), 'PartitionKey': 'p2'},
    ],
)

stream = kinesis.describe_stream(StreamName='orders')
shard_id = stream['StreamDescription']['Shards'][0]['ShardId']
iterator = kinesis.get_shard_iterator(
    StreamName='orders', ShardId=shard_id, ShardIteratorType='TRIM_HORIZON',
)['ShardIterator']

records = kinesis.get_records(ShardIterator=iterator)
for r in records['Records']:
    print(json.loads(r['Data']))
```

## Configuration

```yaml
# cloudmock.yml
services:
  kinesis:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Records are stored **in memory** per shard. Sequence numbers are monotonically increasing integers.
- `GetRecords` advances the iterator; subsequent calls return newer records.
- **Enhanced fan-out** (`SubscribeToShard`) is not implemented.
- **Stream encryption** and server-side encryption are accepted but not enforced.
- **Shard splitting and merging** are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified stream does not exist |
| ResourceInUseException | 400 | The stream is being created or deleted |
| InvalidArgumentException | 400 | An argument is not valid |
| ProvisionedThroughputExceededException | 400 | The request rate exceeds the shard throughput |
