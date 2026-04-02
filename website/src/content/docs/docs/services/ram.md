---
title: RAM
description: AWS Resource Access Manager emulation in CloudMock
---

## Overview

CloudMock emulates AWS Resource Access Manager (RAM), supporting resource share management, associations, invitations, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateResourceShare | Supported | Creates a resource share |
| GetResourceShares | Supported | Lists resource shares |
| UpdateResourceShare | Supported | Updates a resource share |
| DeleteResourceShare | Supported | Deletes a resource share |
| AssociateResourceShare | Supported | Associates resources or principals |
| DisassociateResourceShare | Supported | Disassociates resources or principals |
| GetResourceShareAssociations | Supported | Lists resource share associations |
| GetResourceShareInvitations | Supported | Lists pending invitations |
| AcceptResourceShareInvitation | Supported | Accepts an invitation |
| RejectResourceShareInvitation | Supported | Rejects an invitation |
| ListResources | Supported | Lists resources in resource shares |
| ListPrincipals | Supported | Lists principals in resource shares |
| EnableSharingWithAwsOrganization | Supported | Enables sharing with the AWS organization |
| TagResource | Supported | Adds tags to a resource share |
| UntagResource | Supported | Removes tags from a resource share |
| ListTagsForResource | Supported | Lists tags for a resource share |

## Quick Start

### Node.js

```typescript
import { RAMClient, CreateResourceShareCommand } from '@aws-sdk/client-ram';

const client = new RAMClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { resourceShare } = await client.send(new CreateResourceShareCommand({
  name: 'my-share',
  resourceArns: ['arn:aws:ec2:us-east-1:000000000000:subnet/subnet-12345'],
}));
console.log(resourceShare.resourceShareArn);
```

### Python

```python
import boto3

client = boto3.client('ram',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_resource_share(
    name='my-share',
    resourceArns=['arn:aws:ec2:us-east-1:000000000000:subnet/subnet-12345'])
print(response['resourceShare']['resourceShareArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ram:
    enabled: true
```

## Known Differences from AWS

- Shared resources are not actually accessible from other accounts
- Invitations are tracked but cross-account access is not enforced
- Resource share associations are metadata-only
