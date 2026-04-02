---
title: Neptune
description: Amazon Neptune emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Neptune, supporting cluster and instance lifecycle, snapshots, subnet groups, parameter groups, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDBCluster | Supported | Creates a Neptune cluster |
| DescribeDBClusters | Supported | Lists clusters |
| DeleteDBCluster | Supported | Deletes a cluster |
| ModifyDBCluster | Supported | Modifies cluster configuration |
| CreateDBInstance | Supported | Creates an instance in a cluster |
| DescribeDBInstances | Supported | Lists instances |
| DeleteDBInstance | Supported | Deletes an instance |
| ModifyDBInstance | Supported | Modifies instance configuration |
| CreateDBClusterSnapshot | Supported | Creates a cluster snapshot |
| DescribeDBClusterSnapshots | Supported | Lists cluster snapshots |
| DeleteDBClusterSnapshot | Supported | Deletes a cluster snapshot |
| CreateDBSubnetGroup | Supported | Creates a subnet group |
| DescribeDBSubnetGroups | Supported | Lists subnet groups |
| DeleteDBSubnetGroup | Supported | Deletes a subnet group |
| CreateDBClusterParameterGroup | Supported | Creates a parameter group |
| DescribeDBClusterParameterGroups | Supported | Lists parameter groups |
| DeleteDBClusterParameterGroup | Supported | Deletes a parameter group |
| AddTagsToResource | Supported | Adds tags to a resource |
| RemoveTagsFromResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { NeptuneClient, CreateDBClusterCommand } from '@aws-sdk/client-neptune';

const client = new NeptuneClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateDBClusterCommand({
  DBClusterIdentifier: 'my-neptune-cluster',
  Engine: 'neptune',
}));
```

### Python

```python
import boto3

client = boto3.client('neptune',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_db_cluster(
    DBClusterIdentifier='my-neptune-cluster',
    Engine='neptune')
```

## Configuration

```yaml
# cloudmock.yml
services:
  neptune:
    enabled: true
```

## Known Differences from AWS

- No actual graph database is provisioned
- Gremlin and SPARQL endpoints are not available
- Snapshots are metadata-only
