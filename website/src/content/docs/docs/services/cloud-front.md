---
title: CloudFront
description: Amazon CloudFront emulation in CloudMock
---

## Overview

CloudMock emulates Amazon CloudFront, supporting distribution management, cache invalidation, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDistribution | Supported | Creates a CloudFront distribution |
| GetDistribution | Supported | Returns distribution details |
| ListDistributions | Supported | Lists all distributions |
| UpdateDistribution | Supported | Updates distribution configuration |
| DeleteDistribution | Supported | Deletes a distribution |
| CreateInvalidation | Supported | Creates a cache invalidation |
| GetInvalidation | Supported | Returns invalidation details |
| ListInvalidations | Supported | Lists invalidations for a distribution |
| TagResource | Supported | Adds tags to a distribution |
| UntagResource | Supported | Removes tags from a distribution |
| ListTagsForResource | Supported | Lists tags for a distribution |

## Quick Start

### Node.js

```typescript
import { CloudFrontClient, CreateDistributionCommand } from '@aws-sdk/client-cloudfront';

const client = new CloudFrontClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { Distribution } = await client.send(new CreateDistributionCommand({
  DistributionConfig: {
    CallerReference: 'unique-ref',
    Origins: { Quantity: 1, Items: [{ Id: 'myS3Origin', DomainName: 'my-bucket.s3.amazonaws.com' }] },
    DefaultCacheBehavior: { TargetOriginId: 'myS3Origin', ViewerProtocolPolicy: 'redirect-to-https' },
    Enabled: true,
    Comment: 'My distribution',
  },
}));
console.log(Distribution.Id);
```

### Python

```python
import boto3

client = boto3.client('cloudfront',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_distribution(DistributionConfig={
    'CallerReference': 'unique-ref',
    'Origins': {'Quantity': 1, 'Items': [{'Id': 'myS3Origin', 'DomainName': 'my-bucket.s3.amazonaws.com'}]},
    'DefaultCacheBehavior': {'TargetOriginId': 'myS3Origin', 'ViewerProtocolPolicy': 'redirect-to-https'},
    'Enabled': True,
    'Comment': 'My distribution',
})
print(response['Distribution']['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  cloudfront:
    enabled: true
```

## Known Differences from AWS

- Distributions do not serve actual content
- Invalidations are recorded but do not affect any cache
- Domain names are generated but do not resolve
