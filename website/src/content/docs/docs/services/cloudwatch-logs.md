---
title: CloudWatch Logs
description: Amazon CloudWatch Logs emulation in CloudMock
---

## Overview

CloudMock emulates Amazon CloudWatch Logs, supporting log group and stream management, log event ingestion and retrieval, filtering, retention policies, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateLogGroup | Supported | Creates a log group |
| DeleteLogGroup | Supported | Deletes a log group and all its streams |
| DescribeLogGroups | Supported | Lists log groups with optional prefix filter |
| CreateLogStream | Supported | Creates a log stream within a group |
| DeleteLogStream | Supported | Deletes a log stream |
| DescribeLogStreams | Supported | Lists log streams within a group |
| PutLogEvents | Supported | Appends log events to a stream; requires sequence token after first call |
| GetLogEvents | Supported | Returns log events from a stream |
| FilterLogEvents | Supported | Searches log events across streams with a filter pattern |
| PutRetentionPolicy | Supported | Sets the retention period for a log group |
| DeleteRetentionPolicy | Supported | Removes the retention policy |
| TagLogGroup | Supported | Adds tags to a log group |
| UntagLogGroup | Supported | Removes tags from a log group |
| ListTagsLogGroup | Supported | Returns tags for a log group |

## Quick Start

### curl

```bash
# Create a log group
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: Logs_20140328.CreateLogGroup" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"logGroupName": "/app/server"}'

# Create a log stream
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: Logs_20140328.CreateLogStream" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"logGroupName": "/app/server", "logStreamName": "instance-1"}'
```

### Node.js

```typescript
import { CloudWatchLogsClient, CreateLogGroupCommand, CreateLogStreamCommand, PutLogEventsCommand } from '@aws-sdk/client-cloudwatch-logs';

const logs = new CloudWatchLogsClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await logs.send(new CreateLogGroupCommand({ logGroupName: '/app/api' }));
await logs.send(new CreateLogStreamCommand({ logGroupName: '/app/api', logStreamName: 'instance-1' }));
await logs.send(new PutLogEventsCommand({
  logGroupName: '/app/api', logStreamName: 'instance-1',
  logEvents: [
    { timestamp: Date.now(), message: 'Request received' },
    { timestamp: Date.now() + 1, message: 'Response sent 200' },
  ],
}));
```

### Python

```python
import boto3, time

logs = boto3.client('logs', endpoint_url='http://localhost:4566',
                    aws_access_key_id='test', aws_secret_access_key='test',
                    region_name='us-east-1')

logs.create_log_group(logGroupName='/app/api')
logs.create_log_stream(logGroupName='/app/api', logStreamName='instance-1')

now_ms = int(time.time() * 1000)
logs.put_log_events(
    logGroupName='/app/api', logStreamName='instance-1',
    logEvents=[
        {'timestamp': now_ms, 'message': 'Request received'},
        {'timestamp': now_ms + 1, 'message': 'Response sent 200'},
    ],
)

events = logs.get_log_events(logGroupName='/app/api', logStreamName='instance-1')
for event in events['events']:
    print(event['message'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  cloudwatch-logs:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- `PutLogEvents` requires a **sequenceToken** after the first call to a stream. The token from each response must be passed in the next call.
- `FilterLogEvents` performs **substring matching**; CloudWatch Insights syntax is not supported.
- Log **retention enforcement** (automatic deletion of old events) is not implemented.
- **Subscription filters** and cross-account log delivery are not implemented.
- **Metric filters** are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified log group or stream does not exist |
| ResourceAlreadyExistsException | 400 | A log group or stream with this name already exists |
| InvalidSequenceTokenException | 400 | The sequence token is not valid |
| DataAlreadyAcceptedException | 400 | The log events were already accepted |
