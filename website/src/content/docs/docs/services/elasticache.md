---
title: ElastiCache
description: Amazon ElastiCache emulation in CloudMock
---

## Overview

CloudMock emulates Amazon ElastiCache, supporting cache clusters, replication groups, subnet groups, parameter groups, snapshots, tagging, and failover testing. Cluster states transition `creating -> available -> modifying -> deleting`.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCacheCluster | Supported | Creates a Redis or Memcached cache cluster |
| DescribeCacheClusters | Supported | Lists cache clusters with optional ID filter |
| ModifyCacheCluster | Supported | Modifies node type, engine version, or node count |
| DeleteCacheCluster | Supported | Deletes a cache cluster |
| CreateReplicationGroup | Supported | Creates a Redis replication group with primary/replica nodes |
| DescribeReplicationGroups | Supported | Lists replication groups |
| ModifyReplicationGroup | Supported | Modifies description, node type, or failover settings |
| DeleteReplicationGroup | Supported | Deletes a replication group |
| CreateCacheSubnetGroup | Supported | Creates a subnet group |
| DescribeCacheSubnetGroups | Supported | Lists subnet groups |
| DeleteCacheSubnetGroup | Supported | Deletes a subnet group |
| CreateCacheParameterGroup | Supported | Creates a parameter group |
| DescribeCacheParameterGroups | Supported | Lists parameter groups |
| DeleteCacheParameterGroup | Supported | Deletes a parameter group |
| CreateSnapshot | Supported | Creates a snapshot from a cluster or replication group |
| DescribeSnapshots | Supported | Lists snapshots with optional filters |
| DeleteSnapshot | Supported | Deletes a snapshot |
| AddTagsToResource | Supported | Adds tags to clusters, replication groups, and snapshots |
| RemoveTagsFromResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |
| TestFailover | Supported | Simulates a failover test for a replication group |

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
