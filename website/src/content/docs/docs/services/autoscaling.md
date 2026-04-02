---
title: Auto Scaling
description: Amazon EC2 Auto Scaling emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EC2 Auto Scaling, supporting launch configurations, auto scaling groups with capacity management, scaling policies, and instance lifecycle operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateLaunchConfiguration | Supported | Creates a launch configuration |
| DescribeLaunchConfigurations | Supported | Lists launch configurations |
| DeleteLaunchConfiguration | Supported | Deletes a launch configuration |
| CreateAutoScalingGroup | Supported | Creates an auto scaling group |
| DescribeAutoScalingGroups | Supported | Lists auto scaling groups |
| UpdateAutoScalingGroup | Supported | Updates group configuration |
| DeleteAutoScalingGroup | Supported | Deletes an auto scaling group |
| SetDesiredCapacity | Supported | Sets the desired instance count |
| DescribeAutoScalingInstances | Supported | Lists instances in ASGs |
| AttachInstances | Supported | Attaches instances to an ASG |
| DetachInstances | Supported | Detaches instances from an ASG |
| PutScalingPolicy | Supported | Creates or updates a scaling policy |
| DescribePolicies | Supported | Lists scaling policies |
| DeletePolicy | Supported | Deletes a scaling policy |
| CreateOrUpdateTags | Supported | Creates or updates ASG tags |
| DescribeTags | Supported | Lists ASG tags |
| DeleteTags | Supported | Deletes ASG tags |

## Quick Start

### Node.js

```typescript
import { AutoScalingClient, CreateAutoScalingGroupCommand } from '@aws-sdk/client-auto-scaling';

const client = new AutoScalingClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateAutoScalingGroupCommand({
  AutoScalingGroupName: 'my-asg',
  LaunchConfigurationName: 'my-lc',
  MinSize: 1,
  MaxSize: 5,
  DesiredCapacity: 2,
  AvailabilityZones: ['us-east-1a'],
}));
```

### Python

```python
import boto3

client = boto3.client('autoscaling',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_auto_scaling_group(
    AutoScalingGroupName='my-asg',
    LaunchConfigurationName='my-lc',
    MinSize=1,
    MaxSize=5,
    DesiredCapacity=2,
    AvailabilityZones=['us-east-1a'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  autoscaling:
    enabled: true
```

## Known Differences from AWS

- Scaling policies are stored but automatic scaling based on metrics is not performed
- Instance reconciliation is simulated but does not launch real EC2 instances
- Health checks are simplified
