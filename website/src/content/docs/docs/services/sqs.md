---
title: SQS
description: Amazon SQS (Simple Queue Service) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon SQS, a fully managed message queuing service, supporting queue lifecycle, message send/receive/delete, visibility timeouts, and batch operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateQueue | Supported | Creates a standard queue |
| DeleteQueue | Supported | Deletes a queue and all its messages |
| ListQueues | Supported | Returns queue URLs with optional prefix filter |
| GetQueueUrl | Supported | Returns the URL of a named queue |
| GetQueueAttributes | Supported | Returns queue attributes (ApproximateNumberOfMessages, etc.) |
| SetQueueAttributes | Supported | Sets queue attributes (VisibilityTimeout, etc.) |
| SendMessage | Supported | Sends a single message |
| ReceiveMessage | Supported | Returns up to 10 messages; marks them invisible |
| DeleteMessage | Supported | Permanently removes a received message using its receipt handle |
| PurgeQueue | Supported | Deletes all messages in the queue |
| ChangeMessageVisibility | Supported | Extends or resets visibility timeout |
| SendMessageBatch | Supported | Sends up to 10 messages in one call |
| DeleteMessageBatch | Supported | Deletes up to 10 messages in one call |

## Quick Start

### curl

```bash
# Create a queue
curl -X POST "http://localhost:4566/?Action=CreateQueue&QueueName=my-queue"

# Send a message
curl -X POST "http://localhost:4566/000000000000/my-queue?Action=SendMessage&MessageBody=Hello"

# Receive messages
curl -X POST "http://localhost:4566/000000000000/my-queue?Action=ReceiveMessage&MaxNumberOfMessages=5"
```

### Node.js

```typescript
import { SQSClient, CreateQueueCommand, SendMessageCommand, ReceiveMessageCommand } from '@aws-sdk/client-sqs';

const sqs = new SQSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { QueueUrl } = await sqs.send(new CreateQueueCommand({ QueueName: 'jobs' }));
await sqs.send(new SendMessageCommand({ QueueUrl, MessageBody: '{"job":"process"}' }));
const { Messages } = await sqs.send(new ReceiveMessageCommand({ QueueUrl, MaxNumberOfMessages: 10 }));
```

### Python

```python
import boto3

sqs = boto3.client('sqs', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

response = sqs.create_queue(QueueName='jobs')
url = response['QueueUrl']

sqs.send_message(QueueUrl=url, MessageBody='{"job": "process-image"}')
messages = sqs.receive_message(QueueUrl=url, MaxNumberOfMessages=10).get('Messages', [])
for msg in messages:
    print(msg['Body'])
    sqs.delete_message(QueueUrl=url, ReceiptHandle=msg['ReceiptHandle'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  sqs:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Queue URLs follow the pattern `http://localhost:4566/{AccountId}/{QueueName}`.
- **VisibilityTimeout** is enforced -- messages that are not deleted within the timeout reappear in the queue.
- **FIFO queues** are accepted at creation but do not enforce ordering or deduplication.
- **Dead-letter queue** redrive is not implemented.
- **Message attributes** are stored and returned but not used for filtering.
- **Long polling** (`WaitTimeSeconds`) is accepted but returns immediately.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AWS.SimpleQueueService.NonExistentQueue | 400 | The specified queue does not exist |
| QueueAlreadyExists | 400 | A queue with this name already exists |
| ReceiptHandleIsInvalid | 400 | The receipt handle provided is not valid |
| EmptyBatchRequest | 400 | The batch request contains no entries |
