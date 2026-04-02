---
title: Resource Groups
description: AWS Resource Groups emulation in CloudMock
---

## Overview

CloudMock emulates AWS Resource Groups, supporting group management, resource grouping, search, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateGroup | Supported | Creates a resource group |
| GetGroup | Supported | Returns group details |
| ListGroups | Supported | Lists all groups |
| UpdateGroup | Supported | Updates a group |
| DeleteGroup | Supported | Deletes a group |
| GroupResources | Supported | Adds resources to a group |
| UngroupResources | Supported | Removes resources from a group |
| ListGroupResources | Supported | Lists resources in a group |
| SearchResources | Supported | Searches for resources |
| GetTags | Supported | Returns tags for a group |
| TagResource | Supported | Adds tags to a group |
| UntagResource | Supported | Removes tags from a group |

## Quick Start

### Node.js

```typescript
import { ResourceGroupsClient, CreateGroupCommand } from '@aws-sdk/client-resource-groups';

const client = new ResourceGroupsClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateGroupCommand({
  Name: 'my-group',
  ResourceQuery: {
    Type: 'TAG_FILTERS_1_0',
    Query: JSON.stringify({ ResourceTypeFilters: ['AWS::AllSupported'], TagFilters: [{ Key: 'env', Values: ['prod'] }] }),
  },
}));
```

### Python

```python
import boto3
import json

client = boto3.client('resource-groups',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_group(
    Name='my-group',
    ResourceQuery={
        'Type': 'TAG_FILTERS_1_0',
        'Query': json.dumps({'ResourceTypeFilters': ['AWS::AllSupported'], 'TagFilters': [{'Key': 'env', 'Values': ['prod']}]}),
    })
```

## Configuration

```yaml
# cloudmock.yml
services:
  resourcegroups:
    enabled: true
```

## Known Differences from AWS

- Resource queries are stored but dynamic tag-based discovery is limited
- SearchResources returns manually grouped resources only
- Cross-service resource discovery is not implemented
