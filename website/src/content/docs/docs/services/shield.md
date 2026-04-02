---
title: Shield
description: AWS Shield emulation in CloudMock
---

## Overview

CloudMock emulates AWS Shield (Advanced), supporting protection management, subscriptions, attack tracking, protection groups, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateProtection | Supported | Creates a protection for a resource |
| DescribeProtection | Supported | Returns protection details |
| ListProtections | Supported | Lists all protections |
| DeleteProtection | Supported | Deletes a protection |
| CreateSubscription | Supported | Creates a Shield Advanced subscription |
| DescribeSubscription | Supported | Returns subscription details |
| DescribeAttack | Supported | Returns attack details |
| ListAttacks | Supported | Lists detected attacks |
| CreateProtectionGroup | Supported | Creates a protection group |
| DescribeProtectionGroup | Supported | Returns protection group details |
| ListProtectionGroups | Supported | Lists protection groups |
| UpdateProtectionGroup | Supported | Updates a protection group |
| DeleteProtectionGroup | Supported | Deletes a protection group |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { ShieldClient, CreateProtectionCommand } from '@aws-sdk/client-shield';

const client = new ShieldClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ProtectionId } = await client.send(new CreateProtectionCommand({
  Name: 'my-protection',
  ResourceArn: 'arn:aws:elasticloadbalancing:us-east-1:000000000000:loadbalancer/app/my-alb/1234567890',
}));
console.log(ProtectionId);
```

### Python

```python
import boto3

client = boto3.client('shield',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_protection(
    Name='my-protection',
    ResourceArn='arn:aws:elasticloadbalancing:us-east-1:000000000000:loadbalancer/app/my-alb/1234567890')
print(response['ProtectionId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  shield:
    enabled: true
```

## Known Differences from AWS

- No actual DDoS protection is provided
- Attacks are stubs and not based on real traffic analysis
- Subscription does not incur costs or provide real SRT access
