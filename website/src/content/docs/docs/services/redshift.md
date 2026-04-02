---
title: Redshift
description: Amazon Redshift emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Redshift, supporting cluster lifecycle, snapshots, subnet groups, parameter groups, tagging, pause/resume, and basic SQL statement execution. Cluster states follow `creating -> available -> pausing -> paused -> resuming`.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCluster | Supported | Creates a Redshift cluster |
| DescribeClusters | Supported | Lists clusters with optional identifier filter |
| DeleteCluster | Supported | Deletes a cluster |
| ModifyCluster | Supported | Modifies node type or node count |
| RebootCluster | Supported | Reboots a cluster (no-op in mock) |
| PauseCluster | Supported | Pauses a cluster (status set to `paused`) |
| ResumeCluster | Supported | Resumes a paused cluster |
| CreateClusterSnapshot | Supported | Creates a snapshot including schema metadata |
| DescribeClusterSnapshots | Supported | Lists snapshots |
| DeleteClusterSnapshot | Supported | Deletes a snapshot |
| RestoreFromClusterSnapshot | Supported | Restores a cluster from snapshot (schema is copied) |
| CreateClusterSubnetGroup | Supported | Creates a subnet group |
| DescribeClusterSubnetGroups | Supported | Lists subnet groups |
| DeleteClusterSubnetGroup | Supported | Deletes a subnet group |
| CreateClusterParameterGroup | Supported | Creates a parameter group |
| DescribeClusterParameterGroups | Supported | Lists parameter groups |
| DeleteClusterParameterGroup | Supported | Deletes a parameter group |
| CreateTags | Supported | Adds tags to a resource (by ARN) |
| DeleteTags | Supported | Removes tags from a resource |
| DescribeTags | Supported | Lists tags for a resource |
| AddTagsToResource | Supported | Alias for CreateTags |
| RemoveTagsFromResource | Supported | Alias for DeleteTags |
| ExecuteStatement | Supported | Executes SQL with basic parsing and validation |
| DescribeStatement | Supported | Returns statement execution status |
| GetStatementResult | Supported | Returns stub query results |

## Quick Start

### Node.js

```typescript
import { RedshiftClient, CreateClusterCommand } from '@aws-sdk/client-redshift';

const client = new RedshiftClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateClusterCommand({
  ClusterIdentifier: 'my-cluster',
  NodeType: 'dc2.large',
  MasterUsername: 'admin',
  MasterUserPassword: 'Password123',
  NumberOfNodes: 2,
}));
```

### Python

```python
import boto3

client = boto3.client('redshift',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_cluster(
    ClusterIdentifier='my-cluster',
    NodeType='dc2.large',
    MasterUsername='admin',
    MasterUserPassword='Password123',
    NumberOfNodes=2)
```

## Configuration

```yaml
# cloudmock.yml
services:
  redshift:
    enabled: true
```

## Known Differences from AWS

- No actual data warehouse is provisioned
- SQL statements are not executed; results are stubs
- Snapshots are metadata-only and restore creates a new empty cluster
