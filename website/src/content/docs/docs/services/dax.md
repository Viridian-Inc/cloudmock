---
title: DAX
description: Amazon DynamoDB Accelerator (DAX) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon DynamoDB Accelerator (DAX), providing a fully in-memory mock of the DAX control plane API. DAX is an in-memory cache for DynamoDB that delivers up to 10x performance improvement. CloudMock supports cluster management, subnet groups, parameter groups, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCluster | Supported | Creates a DAX cluster with proper ARN (arn:aws:dax:{region}:{account}:cache/{name}) |
| DescribeClusters | Supported | Lists all clusters or filters by name |
| UpdateCluster | Supported | Updates description, maintenance window, notification topic |
| DeleteCluster | Supported | Deletes a cluster (returns cluster in deleting state) |
| IncreaseReplicationFactor | Supported | Adds nodes to an existing cluster |
| DecreaseReplicationFactor | Supported | Removes nodes from an existing cluster |
| CreateSubnetGroup | Supported | Creates a subnet group |
| DescribeSubnetGroups | Supported | Lists subnet groups |
| DeleteSubnetGroup | Supported | Deletes a subnet group |
| CreateParameterGroup | Supported | Creates a parameter group |
| DescribeParameterGroups | Supported | Lists parameter groups |
| UpdateParameterGroup | Supported | Updates parameters in a parameter group |
| DeleteParameterGroup | Supported | Deletes a parameter group |
| DescribeParameters | Supported | Returns parameters for a given group |
| DescribeDefaultParameters | Supported | Returns default DAX parameters |
| TagResource | Supported | Adds tags to a cluster by ARN |
| UntagResource | Supported | Removes tags from a cluster |
| ListTags | Supported | Lists tags for a cluster |

## Cluster States

DAX clusters progress through the following states:

| State | Description |
|-------|-------------|
| creating | Cluster is being provisioned |
| available | Cluster is ready for use |
| modifying | Cluster configuration is being updated |
| deleting | Cluster is being deleted |

In CloudMock, clusters transition from `creating` to `available` nearly instantly (lifecycle machine with delays disabled in test mode).

## Quick Start

### Node.js

```typescript
import { DAXClient, CreateClusterCommand, DescribeClustersCommand } from '@aws-sdk/client-dax';

const client = new DAXClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// Create a cluster
const { Cluster } = await client.send(new CreateClusterCommand({
  ClusterName: 'my-dax-cluster',
  NodeType: 'dax.r4.large',
  ReplicationFactor: 3,
  IamRoleArn: 'arn:aws:iam::123456789012:role/AmazonDAXFullAccess',
  SubnetGroupName: 'my-subnet-group',
}));
console.log(Cluster.ClusterArn);
// arn:aws:dax:us-east-1:123456789012:cache/my-dax-cluster

// List clusters
const { Clusters } = await client.send(new DescribeClustersCommand({}));
console.log(Clusters.length);
```

### Python

```python
import boto3

client = boto3.client('dax',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

# Create a cluster
response = client.create_cluster(
    ClusterName='my-dax-cluster',
    NodeType='dax.r4.large',
    ReplicationFactor=3,
    IamRoleArn='arn:aws:iam::123456789012:role/AmazonDAXFullAccess',
    SubnetGroupName='my-subnet-group')
print(response['Cluster']['ClusterArn'])

# Create a parameter group
pg = client.create_parameter_group(
    ParameterGroupName='my-params',
    Description='Custom DAX parameters')

# Update a parameter
client.update_parameter_group(
    ParameterGroupName='my-params',
    ParameterNameValues=[
        {'ParameterName': 'query-ttl-millis', 'ParameterValue': '600000'},
        {'ParameterName': 'record-ttl-millis', 'ParameterValue': '600000'},
    ])
```

## Configuration

```yaml
# cloudmock.yml
services:
  dax:
    enabled: true
```

## ARN Format

DAX cluster ARNs follow this format:

```
arn:aws:dax:{region}:{account-id}:cache/{cluster-name}
```

## Default Parameters

CloudMock provides the following default DAX parameters:

| Parameter | Default Value | Description |
|-----------|---------------|-------------|
| query-ttl-millis | 300000 | TTL for cached query results (5 minutes) |
| record-ttl-millis | 300000 | TTL for cached item records (5 minutes) |

## Known Differences from AWS

- Cluster endpoints are synthetic DNS names and do not accept real DAX protocol connections
- Node types are not validated against available DAX node type catalog
- Availability zone placement is simulated
- Cluster encryption (SSE) is tracked but not enforced
- No VPC subnet validation is performed

## Data-Plane (Caching Proxy)

CloudMock includes a DAX data-plane HTTP server on port `8111` that acts as a caching proxy for DynamoDB operations. Unlike real DAX (which uses a proprietary binary protocol), CloudMock's data-plane accepts standard DynamoDB JSON requests over HTTP.

### Supported Data-Plane Operations

| Operation | Caching Behavior |
|-----------|-----------------|
| GetItem | Read-through: cache miss forwards to DynamoDB, result cached |
| Query | Read-through: full result set cached by request hash |
| Scan | Read-through: full result set cached by request hash |
| PutItem | Write-through: writes to DynamoDB, then invalidates/updates cache |
| UpdateItem | Write-through: writes to DynamoDB, then invalidates cache |
| DeleteItem | Write-through: writes to DynamoDB, then invalidates cache |

### Quick Start

Point your standard DynamoDB SDK at port `8111` instead of `4566`:

```typescript
import { DynamoDBClient, GetItemCommand } from '@aws-sdk/client-dynamodb';

const client = new DynamoDBClient({
  endpoint: 'http://localhost:8111',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// First call: cache miss → reads from DynamoDB → caches result
// Second call: cache hit → returns from cache
const result = await client.send(new GetItemCommand({
  TableName: 'users',
  Key: { pk: { S: 'user1' } },
}));
```

### Cluster Selection

Use the `X-Dax-Cluster` header to route to a specific cluster's cache:

```bash
curl -X POST http://localhost:8111 \
  -H 'X-Amz-Target: DynamoDB_20120810.GetItem' \
  -H 'X-Dax-Cluster: my-cluster' \
  -d '{"TableName":"users","Key":{"pk":{"S":"user1"}}}'
```

If omitted, a default cache with 5-minute TTLs is used.

### Cache Configuration

Controlled through DAX parameter groups (via control-plane API on port `4566`):

| Parameter | Default | Description |
|-----------|---------|-------------|
| `record-ttl-millis` | 300000 | TTL for cached items (5 minutes) |
| `query-ttl-millis` | 300000 | TTL for cached query results |
| `write-strategy` | `invalidate` | `invalidate` (matches AWS) or `update-cache` |

### Cache Statistics

```bash
curl http://localhost:8111/stats/my-cluster
```

Returns:
```json
{"itemHits":1520,"itemMisses":340,"queryHits":200,"queryMisses":80,"itemSize":890,"querySize":45,"evictions":12,"writeThroughs":150,"invalidations":150}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_DAX_PORT` | `8111` | Port for the DAX data-plane server |
