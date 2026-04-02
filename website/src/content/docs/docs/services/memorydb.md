---
title: MemoryDB
description: Amazon MemoryDB for Redis emulation in CloudMock
---

## Overview

CloudMock emulates Amazon MemoryDB for Redis, supporting clusters, ACLs, users, subnet groups, parameter groups, snapshots, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCluster | Supported | Creates a MemoryDB cluster |
| DescribeClusters | Supported | Lists clusters |
| DeleteCluster | Supported | Deletes a cluster |
| UpdateCluster | Supported | Updates cluster configuration |
| CreateACL | Supported | Creates an ACL |
| DescribeACLs | Supported | Lists ACLs |
| DeleteACL | Supported | Deletes an ACL |
| UpdateACL | Supported | Updates an ACL |
| CreateUser | Supported | Creates a user |
| DescribeUsers | Supported | Lists users |
| DeleteUser | Supported | Deletes a user |
| UpdateUser | Supported | Updates a user |
| CreateSubnetGroup | Supported | Creates a subnet group |
| DescribeSubnetGroups | Supported | Lists subnet groups |
| DeleteSubnetGroup | Supported | Deletes a subnet group |
| CreateParameterGroup | Supported | Creates a parameter group |
| DescribeParameterGroups | Supported | Lists parameter groups |
| DeleteParameterGroup | Supported | Deletes a parameter group |
| CreateSnapshot | Supported | Creates a snapshot |
| DescribeSnapshots | Supported | Lists snapshots |
| DeleteSnapshot | Supported | Deletes a snapshot |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTags | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { MemoryDBClient, CreateClusterCommand } from '@aws-sdk/client-memorydb';

const client = new MemoryDBClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateClusterCommand({
  ClusterName: 'my-memorydb',
  NodeType: 'db.r6g.large',
  ACLName: 'open-access',
}));
```

### Python

```python
import boto3

client = boto3.client('memorydb',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_cluster(
    ClusterName='my-memorydb',
    NodeType='db.r6g.large',
    ACLName='open-access')
```

## Configuration

```yaml
# cloudmock.yml
services:
  memorydb:
    enabled: true
```

## Known Differences from AWS

- No actual Redis engine is provisioned
- Cluster endpoints are generated but not functional
- Snapshots are metadata-only
