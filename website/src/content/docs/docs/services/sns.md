---
title: SNS
description: Amazon SNS (Simple Notification Service) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon SNS, a pub/sub messaging service, supporting topic management, subscriptions, message publishing with fan-out to SQS subscribers, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateTopic | Supported | Creates a topic and returns its ARN |
| DeleteTopic | Supported | Deletes a topic and all its subscriptions |
| ListTopics | Supported | Returns all topic ARNs |
| GetTopicAttributes | Supported | Returns topic attributes |
| SetTopicAttributes | Supported | Sets topic attributes |
| Subscribe | Supported | Creates a subscription (SQS, HTTP, email, Lambda) |
| Unsubscribe | Supported | Removes a subscription |
| ListSubscriptions | Supported | Returns all subscriptions |
| ListSubscriptionsByTopic | Supported | Returns subscriptions for a specific topic |
| Publish | Supported | Publishes a message; fans out to SQS subscribers |
| TagResource | Supported | Adds tags to a topic |
| UntagResource | Supported | Removes tags from a topic |

## Quick Start

### curl

```bash
# Create a topic
curl -X POST "http://localhost:4566/?Action=CreateTopic&Name=notifications"

# Subscribe an SQS queue
curl -X POST "http://localhost:4566/?Action=Subscribe&TopicArn=arn:aws:sns:us-east-1:000000000000:notifications&Protocol=sqs&Endpoint=arn:aws:sqs:us-east-1:000000000000:my-queue"

# Publish a message
curl -X POST "http://localhost:4566/?Action=Publish&TopicArn=arn:aws:sns:us-east-1:000000000000:notifications&Message=Hello"
```

### Node.js

```typescript
import { SNSClient, CreateTopicCommand, PublishCommand, SubscribeCommand } from '@aws-sdk/client-sns';

const sns = new SNSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { TopicArn } = await sns.send(new CreateTopicCommand({ Name: 'alerts' }));
await sns.send(new SubscribeCommand({
  TopicArn, Protocol: 'sqs',
  Endpoint: 'arn:aws:sqs:us-east-1:000000000000:my-queue',
}));
await sns.send(new PublishCommand({ TopicArn, Message: 'Something happened' }));
```

### Python

```python
import boto3

sns = boto3.client('sns', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

topic = sns.create_topic(Name='alerts')
topic_arn = topic['TopicArn']

sns.subscribe(TopicArn=topic_arn, Protocol='sqs',
              Endpoint='arn:aws:sqs:us-east-1:000000000000:my-queue')
sns.publish(TopicArn=topic_arn, Message='Something happened', Subject='Alert')
```

## Configuration

```yaml
# cloudmock.yml
services:
  sns:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **Fan-out to SQS** is implemented: publishing to a topic delivers the message to all subscribed SQS queues within CloudMock.
- **HTTP/HTTPS** and **email** protocol subscriptions are stored but delivery is not attempted.
- **Lambda** subscriptions record the ARN but do not invoke the function.
- **Message filtering** by subscription filter policies is not implemented.
- **FIFO topics** are not implemented.
- **Platform applications** (mobile push) are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| NotFound | 404 | The specified topic does not exist |
| InvalidParameter | 400 | An input parameter is invalid |
| SubscriptionLimitExceeded | 400 | Too many subscriptions on this topic |
| AuthorizationError | 403 | Not authorized to perform this action |
