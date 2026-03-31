---
title: EventBridge
description: Amazon EventBridge emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EventBridge, supporting event bus management, rule creation with event patterns, target association with fan-out to SQS, and custom event publishing.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateEventBus | Supported | Creates a custom event bus |
| DeleteEventBus | Supported | Deletes a custom event bus |
| DescribeEventBus | Supported | Returns event bus details |
| ListEventBuses | Supported | Returns all event buses including the default |
| PutRule | Supported | Creates or updates a rule on an event bus |
| DeleteRule | Supported | Deletes a rule |
| DescribeRule | Supported | Returns rule details |
| ListRules | Supported | Returns rules on an event bus |
| PutTargets | Supported | Associates targets (SQS, Lambda, etc.) with a rule |
| RemoveTargets | Supported | Removes targets from a rule |
| ListTargetsByRule | Supported | Returns all targets for a rule |
| PutEvents | Supported | Publishes custom events to event buses |
| EnableRule | Supported | Enables a disabled rule |
| DisableRule | Supported | Disables a rule without deleting it |
| TagResource | Supported | Adds tags to a rule or event bus |
| UntagResource | Supported | Removes tags |
| ListTagsForResource | Supported | Returns tags for a resource |

## Quick Start

### curl

```bash
# Create an event bus
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AWSEvents.CreateEventBus" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"Name": "my-app-bus"}'

# Put events
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AWSEvents.PutEvents" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"Entries": [{"EventBusName": "my-app-bus", "Source": "com.myapp.orders", "DetailType": "OrderCreated", "Detail": "{\"orderId\": \"o-123\"}"}]}'
```

### Node.js

```typescript
import { EventBridgeClient, CreateEventBusCommand, PutRuleCommand, PutEventsCommand } from '@aws-sdk/client-eventbridge';

const eb = new EventBridgeClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await eb.send(new CreateEventBusCommand({ Name: 'app-bus' }));
await eb.send(new PutRuleCommand({
  Name: 'OrderCreated', EventBusName: 'app-bus',
  EventPattern: JSON.stringify({ source: ['com.myapp.orders'] }),
  State: 'ENABLED',
}));
await eb.send(new PutEventsCommand({
  Entries: [{
    EventBusName: 'app-bus', Source: 'com.myapp.orders',
    DetailType: 'OrderCreated', Detail: JSON.stringify({ orderId: 'o-123' }),
  }],
}));
```

### Python

```python
import boto3, json

events = boto3.client('events', endpoint_url='http://localhost:4566',
                      aws_access_key_id='test', aws_secret_access_key='test',
                      region_name='us-east-1')

events.create_event_bus(Name='app-bus')
events.put_rule(
    Name='UserSignedUp', EventBusName='app-bus',
    EventPattern=json.dumps({'source': ['com.myapp.auth']}),
    State='ENABLED',
)
events.put_events(Entries=[{
    'EventBusName': 'app-bus', 'Source': 'com.myapp.auth',
    'DetailType': 'UserSignedUp',
    'Detail': json.dumps({'userId': 'u-456'}),
}])
```

## Configuration

```yaml
# cloudmock.yml
services:
  eventbridge:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- `PutEvents` stores events in memory and **fans out to SQS targets** within CloudMock.
- **Scheduled rules** (cron/rate expressions) are stored but do not trigger on a schedule.
- **Lambda, SNS, and other target types** are stored but delivery is not implemented beyond SQS.
- The **default event bus** (`default`) is always available.
- **Schema discovery** and **event replay** are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified event bus or rule does not exist |
| ResourceAlreadyExistsException | 400 | An event bus with this name already exists |
| InvalidEventPatternException | 400 | The event pattern is not valid JSON |
| ConcurrentModificationException | 400 | The rule was modified by another request |
