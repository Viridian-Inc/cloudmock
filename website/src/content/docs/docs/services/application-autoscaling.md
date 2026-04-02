---
title: Application Auto Scaling
description: AWS Application Auto Scaling emulation in CloudMock
---

## Overview

CloudMock emulates AWS Application Auto Scaling, supporting scalable target registration, scaling policies, and scheduled actions for various AWS services.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RegisterScalableTarget | Supported | Registers a scalable target |
| DescribeScalableTargets | Supported | Lists registered scalable targets |
| DeregisterScalableTarget | Supported | Removes a scalable target |
| PutScalingPolicy | Supported | Creates or updates a scaling policy |
| DescribeScalingPolicies | Supported | Lists scaling policies |
| DeleteScalingPolicy | Supported | Deletes a scaling policy |
| PutScheduledAction | Supported | Creates or updates a scheduled action |
| DescribeScheduledActions | Supported | Lists scheduled actions |
| DeleteScheduledAction | Supported | Deletes a scheduled action |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { ApplicationAutoScalingClient, RegisterScalableTargetCommand } from '@aws-sdk/client-application-auto-scaling';

const client = new ApplicationAutoScalingClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new RegisterScalableTargetCommand({
  ServiceNamespace: 'ecs',
  ResourceId: 'service/my-cluster/my-service',
  ScalableDimension: 'ecs:service:DesiredCount',
  MinCapacity: 1,
  MaxCapacity: 10,
}));
```

### Python

```python
import boto3

client = boto3.client('application-autoscaling',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.register_scalable_target(
    ServiceNamespace='ecs',
    ResourceId='service/my-cluster/my-service',
    ScalableDimension='ecs:service:DesiredCount',
    MinCapacity=1,
    MaxCapacity=10)
```

## Configuration

```yaml
# cloudmock.yml
services:
  applicationautoscaling:
    enabled: true
```

## Known Differences from AWS

- Scaling policies are stored but do not trigger actual scaling actions
- Scheduled actions are stored but do not execute on schedule
- No integration with target service metrics
