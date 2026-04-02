---
title: Auto Scaling
description: Amazon EC2 Auto Scaling emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EC2 Auto Scaling, supporting launch configurations, auto scaling groups with min/max/desired capacity enforcement, scaling policies (Simple, Step, and Target Tracking), scheduled actions, lifecycle hooks, metrics collection, and instance lifecycle operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateLaunchConfiguration | Supported | |
| DescribeLaunchConfigurations | Supported | |
| DeleteLaunchConfiguration | Supported | |
| CreateAutoScalingGroup | Supported | Enforces min/max/desired capacity |
| DescribeAutoScalingGroups | Supported | |
| UpdateAutoScalingGroup | Supported | |
| DeleteAutoScalingGroup | Supported | |
| SetDesiredCapacity | Supported | Clamped to min/max |
| DescribeAutoScalingInstances | Supported | |
| AttachInstances | Supported | |
| DetachInstances | Supported | |
| PutScalingPolicy | Supported | SimpleScaling, StepScaling, TargetTrackingScaling |
| DescribePolicies | Supported | Filter by ASG name or policy names |
| DeletePolicy | Supported | |
| ExecutePolicy | Supported | Applies scaling adjustment; clamps to min/max |
| PutScheduledUpdateGroupAction | Supported | Cron-based scheduled scaling |
| DescribeScheduledActions | Supported | Filter by ASG name |
| DeleteScheduledAction | Supported | |
| EnableMetricsCollection | Supported | No-op acknowledgment in mock |
| DisableMetricsCollection | Supported | No-op acknowledgment in mock |
| PutLifecycleHook | Supported | LAUNCHING and TERMINATING transitions |
| DescribeLifecycleHooks | Supported | Filter by ASG name or hook names |
| DeleteLifecycleHook | Supported | |
| CompleteLifecycleAction | Supported | No-op (lifecycle completion acknowledged) |
| CreateOrUpdateTags | Supported | |
| DescribeTags | Supported | |
| DeleteTags | Supported | |

## Quick Start

### Node.js

```typescript
import {
  AutoScalingClient,
  CreateLaunchConfigurationCommand,
  CreateAutoScalingGroupCommand,
  PutScalingPolicyCommand,
  PutLifecycleHookCommand,
} from '@aws-sdk/client-auto-scaling';

const client = new AutoScalingClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateLaunchConfigurationCommand({
  LaunchConfigurationName: 'my-lc',
  ImageId: 'ami-12345678',
  InstanceType: 't3.micro',
}));

await client.send(new CreateAutoScalingGroupCommand({
  AutoScalingGroupName: 'my-asg',
  LaunchConfigurationName: 'my-lc',
  MinSize: 1,
  MaxSize: 5,
  DesiredCapacity: 2,
  AvailabilityZones: ['us-east-1a'],
}));

await client.send(new PutLifecycleHookCommand({
  AutoScalingGroupName: 'my-asg',
  LifecycleHookName: 'launch-hook',
  LifecycleTransition: 'autoscaling:EC2_INSTANCE_LAUNCHING',
  DefaultResult: 'CONTINUE',
  HeartbeatTimeout: 300,
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

client.create_launch_configuration(
    LaunchConfigurationName='my-lc',
    ImageId='ami-12345678',
    InstanceType='t3.micro')

client.create_auto_scaling_group(
    AutoScalingGroupName='my-asg',
    LaunchConfigurationName='my-lc',
    MinSize=1, MaxSize=5, DesiredCapacity=2,
    AvailabilityZones=['us-east-1a'])

client.put_scheduled_update_group_action(
    AutoScalingGroupName='my-asg',
    ScheduledActionName='nightly-scale-down',
    DesiredCapacity=1,
    Recurrence='0 2 * * *')
```

## Configuration

```yaml
# cloudmock.yml
services:
  autoscaling:
    enabled: true
```

## Known Differences from AWS

- Scaling policies are stored but automatic metric-based scaling is not performed
- ExecutePolicy applies the adjustment once on demand
- Lifecycle hooks are stored but do not intercept actual instance launch/termination
- EnableMetricsCollection/DisableMetricsCollection are no-ops (acknowledged, not enforced)
- Instance reconciliation is simulated and does not launch real EC2 instances
