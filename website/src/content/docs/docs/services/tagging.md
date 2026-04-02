---
title: Resource Groups Tagging API
description: AWS Resource Groups Tagging API emulation in CloudMock
---

## Overview

CloudMock emulates the AWS Resource Groups Tagging API, supporting cross-service tag management, querying, and compliance summaries.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetResources | Supported | Finds resources by tag filters |
| GetTagKeys | Supported | Lists all tag keys in use |
| GetTagValues | Supported | Lists values for a tag key |
| TagResources | Supported | Adds tags to multiple resources |
| UntagResources | Supported | Removes tags from multiple resources |
| GetComplianceSummary | Supported | Returns tag compliance summary |

## Quick Start

### Node.js

```typescript
import { ResourceGroupsTaggingAPIClient, GetResourcesCommand, TagResourcesCommand } from '@aws-sdk/client-resource-groups-tagging-api';

const client = new ResourceGroupsTaggingAPIClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new TagResourcesCommand({
  ResourceARNList: ['arn:aws:s3:::my-bucket'],
  Tags: { Environment: 'production' },
}));

const { ResourceTagMappingList } = await client.send(new GetResourcesCommand({
  TagFilters: [{ Key: 'Environment', Values: ['production'] }],
}));
console.log(ResourceTagMappingList);
```

### Python

```python
import boto3

client = boto3.client('resourcegroupstaggingapi',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.tag_resources(
    ResourceARNList=['arn:aws:s3:::my-bucket'],
    Tags={'Environment': 'production'})

response = client.get_resources(
    TagFilters=[{'Key': 'Environment', 'Values': ['production']}])
print(response['ResourceTagMappingList'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  tagging:
    enabled: true
```

## Known Differences from AWS

- Resource discovery is limited to resources registered within CloudMock
- Compliance summary returns simplified results
- Tag policies are not enforced
