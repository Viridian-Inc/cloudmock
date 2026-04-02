---
title: Application Auto Scaling
description: AWS Application Auto Scaling emulation in CloudMock
---

## Overview

CloudMock emulates AWS Application Auto Scaling, supporting scalable target registration, TargetTracking and StepScaling policies, scheduled actions, and resource tagging across multiple service namespaces (ECS, DynamoDB, RDS, Lambda, SageMaker, and more).

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RegisterScalableTarget | Supported | Register or update; idempotent |
| DescribeScalableTargets | Supported | Filter by namespace, resource IDs, dimension |
| DeregisterScalableTarget | Supported | Removes target |
| PutScalingPolicy | Supported | TargetTrackingScaling and StepScaling types |
| DescribeScalingPolicies | Supported | Filter by namespace, resource, dimension |
| DeleteScalingPolicy | Supported | |
| PutScheduledAction | Supported | Upsert; start/end times supported |
| DescribeScheduledActions | Supported | Filter by namespace and resource |
| DeleteScheduledAction | Supported | |
| TagResource | Supported | Tags policies and scheduled actions by ARN |
| UntagResource | Supported | |
| ListTagsForResource | Supported | |

## Supported Service Namespaces

`ecs`, `dynamodb`, `ec2`, `rds`, `sagemaker`, `custom-resource`, `comprehend`, `lambda`, `cassandra`, `kafka`, `elasticache`, `neptune`

## Quick Start

### Node.js

```typescript
import {
  ApplicationAutoScalingClient,
  RegisterScalableTargetCommand,
  PutScalingPolicyCommand,
  PutScheduledActionCommand,
} from '@aws-sdk/client-application-auto-scaling';

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

await client.send(new PutScalingPolicyCommand({
  PolicyName: 'cpu-tracking',
  ServiceNamespace: 'ecs',
  ResourceId: 'service/my-cluster/my-service',
  ScalableDimension: 'ecs:service:DesiredCount',
  PolicyType: 'TargetTrackingScaling',
  TargetTrackingScalingPolicyConfiguration: { TargetValue: 70.0 },
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
    ServiceNamespace='dynamodb',
    ResourceId='table/my-table',
    ScalableDimension='dynamodb:table:ReadCapacityUnits',
    MinCapacity=5,
    MaxCapacity=1000)

client.put_scaling_policy(
    PolicyName='read-capacity-tracking',
    ServiceNamespace='dynamodb',
    ResourceId='table/my-table',
    ScalableDimension='dynamodb:table:ReadCapacityUnits',
    PolicyType='TargetTrackingScaling',
    TargetTrackingScalingPolicyConfiguration={'TargetValue': 70.0})
```

## Configuration

```yaml
# cloudmock.yml
services:
  applicationautoscaling:
    enabled: true
```

## Known Differences from AWS

- Scaling policies are stored but do not trigger actual scaling actions automatically
- Scheduled actions are stored but do not execute on their cron schedule
- No integration with CloudWatch metrics for target tracking
