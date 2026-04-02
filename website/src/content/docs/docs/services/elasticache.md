---
title: ElastiCache
description: Amazon ElastiCache emulation in CloudMock
---

## Overview

CloudMock emulates Amazon ElastiCache, supporting cache clusters, replication groups, subnet groups, parameter groups, tagging, and failover testing.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCacheCluster | Supported | Creates a cache cluster |
| DescribeCacheClusters | Supported | Lists cache clusters |
| ModifyCacheCluster | Supported | Modifies a cache cluster |
| DeleteCacheCluster | Supported | Deletes a cache cluster |
| CreateReplicationGroup | Supported | Creates a replication group |
| DescribeReplicationGroups | Supported | Lists replication groups |
| ModifyReplicationGroup | Supported | Modifies a replication group |
| DeleteReplicationGroup | Supported | Deletes a replication group |
| CreateCacheSubnetGroup | Supported | Creates a subnet group |
| DescribeCacheSubnetGroups | Supported | Lists subnet groups |
| DeleteCacheSubnetGroup | Supported | Deletes a subnet group |
| CreateCacheParameterGroup | Supported | Creates a parameter group |
| DescribeCacheParameterGroups | Supported | Lists parameter groups |
| DeleteCacheParameterGroup | Supported | Deletes a parameter group |
| AddTagsToResource | Supported | Adds tags to a resource |
| RemoveTagsFromResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |
| TestFailover | Supported | Simulates a failover test |

## Quick Start

### Node.js

```typescript
import { ElastiCacheClient, CreateCacheClusterCommand } from '@aws-sdk/client-elasticache';

const client = new ElastiCacheClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateCacheClusterCommand({
  CacheClusterId: 'my-cache',
  Engine: 'redis',
  CacheNodeType: 'cache.t3.micro',
  NumCacheNodes: 1,
}));
```

### Python

```python
import boto3

client = boto3.client('elasticache',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_cache_cluster(
    CacheClusterId='my-cache',
    Engine='redis',
    CacheNodeType='cache.t3.micro',
    NumCacheNodes=1)
```

## Configuration

```yaml
# cloudmock.yml
services:
  elasticache:
    enabled: true
```

## Known Differences from AWS

- No actual Redis or Memcached engine is provisioned
- Cache endpoints are generated but not functional
- Failover testing updates status but does not simulate real failover
