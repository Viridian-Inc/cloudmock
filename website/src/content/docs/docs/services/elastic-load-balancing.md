---
title: Elastic Load Balancing
description: Elastic Load Balancing (v2) emulation in CloudMock
---

## Overview

CloudMock emulates Elastic Load Balancing v2 (ALB/NLB), supporting load balancers, target groups, listeners, rules, target health, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateLoadBalancer | Supported | Creates an ALB or NLB |
| DescribeLoadBalancers | Supported | Lists load balancers |
| DeleteLoadBalancer | Supported | Deletes a load balancer |
| ModifyLoadBalancerAttributes | Supported | Modifies LB attributes |
| CreateTargetGroup | Supported | Creates a target group |
| DescribeTargetGroups | Supported | Lists target groups |
| DeleteTargetGroup | Supported | Deletes a target group |
| ModifyTargetGroup | Supported | Modifies target group settings |
| RegisterTargets | Supported | Registers targets |
| DeregisterTargets | Supported | Deregisters targets |
| DescribeTargetHealth | Supported | Returns target health status |
| CreateListener | Supported | Creates a listener |
| DescribeListeners | Supported | Lists listeners |
| DeleteListener | Supported | Deletes a listener |
| ModifyListener | Supported | Modifies a listener |
| CreateRule | Supported | Creates a listener rule |
| DescribeRules | Supported | Lists listener rules |
| DeleteRule | Supported | Deletes a listener rule |
| AddTags | Supported | Adds tags to resources |
| RemoveTags | Supported | Removes tags from resources |
| DescribeTags | Supported | Lists tags for resources |

## Quick Start

### Node.js

```typescript
import { ElasticLoadBalancingV2Client, CreateLoadBalancerCommand } from '@aws-sdk/client-elastic-load-balancing-v2';

const client = new ElasticLoadBalancingV2Client({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { LoadBalancers } = await client.send(new CreateLoadBalancerCommand({
  Name: 'my-alb',
  Subnets: ['subnet-12345', 'subnet-67890'],
  Type: 'application',
}));
console.log(LoadBalancers[0].LoadBalancerArn);
```

### Python

```python
import boto3

client = boto3.client('elbv2',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_load_balancer(
    Name='my-alb',
    Subnets=['subnet-12345', 'subnet-67890'],
    Type='application')
print(response['LoadBalancers'][0]['LoadBalancerArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  elasticloadbalancing:
    enabled: true
```

## Known Differences from AWS

- Load balancers do not route actual traffic
- Target health checks are simulated
- DNS names are generated but do not resolve
