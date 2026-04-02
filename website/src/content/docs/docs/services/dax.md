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
