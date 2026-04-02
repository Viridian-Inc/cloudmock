---
title: EventBridge Scheduler
description: Amazon EventBridge Scheduler emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EventBridge Scheduler, supporting schedule and schedule group management with tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateSchedule | Supported | Creates a schedule |
| GetSchedule | Supported | Returns schedule details |
| ListSchedules | Supported | Lists all schedules |
| UpdateSchedule | Supported | Updates a schedule |
| DeleteSchedule | Supported | Deletes a schedule |
| CreateScheduleGroup | Supported | Creates a schedule group |
| GetScheduleGroup | Supported | Returns schedule group details |
| ListScheduleGroups | Supported | Lists schedule groups |
| DeleteScheduleGroup | Supported | Deletes a schedule group |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { SchedulerClient, CreateScheduleCommand } from '@aws-sdk/client-scheduler';

const client = new SchedulerClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateScheduleCommand({
  Name: 'my-schedule',
  ScheduleExpression: 'rate(1 hour)',
  FlexibleTimeWindow: { Mode: 'OFF' },
  Target: {
    Arn: 'arn:aws:lambda:us-east-1:000000000000:function:my-function',
    RoleArn: 'arn:aws:iam::000000000000:role/scheduler-role',
  },
}));
```

### Python

```python
import boto3

client = boto3.client('scheduler',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_schedule(
    Name='my-schedule',
    ScheduleExpression='rate(1 hour)',
    FlexibleTimeWindow={'Mode': 'OFF'},
    Target={
        'Arn': 'arn:aws:lambda:us-east-1:000000000000:function:my-function',
        'RoleArn': 'arn:aws:iam::000000000000:role/scheduler-role',
    })
```

## Configuration

```yaml
# cloudmock.yml
services:
  scheduler:
    enabled: true
```

## Known Differences from AWS

- Schedules do not trigger target invocations
- Cron and rate expressions are stored but not evaluated
- Schedule groups are organizational only
