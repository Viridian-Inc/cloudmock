---
title: DMS
description: AWS Database Migration Service emulation in CloudMock
---

## Overview

CloudMock emulates AWS Database Migration Service (DMS), supporting replication instances, endpoints, replication tasks, event subscriptions, and connection testing.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateReplicationInstance | Supported | Creates a replication instance |
| DescribeReplicationInstances | Supported | Lists replication instances |
| DeleteReplicationInstance | Supported | Deletes a replication instance |
| CreateEndpoint | Supported | Creates a source or target endpoint |
| DescribeEndpoints | Supported | Lists endpoints |
| DeleteEndpoint | Supported | Deletes an endpoint |
| CreateReplicationTask | Supported | Creates a replication task |
| DescribeReplicationTasks | Supported | Lists replication tasks |
| StartReplicationTask | Supported | Starts a replication task |
| StopReplicationTask | Supported | Stops a replication task |
| DeleteReplicationTask | Supported | Deletes a replication task |
| CreateEventSubscription | Supported | Creates an event subscription |
| DescribeEventSubscriptions | Supported | Lists event subscriptions |
| DeleteEventSubscription | Supported | Deletes an event subscription |
| TestConnection | Supported | Tests endpoint connectivity (stub) |
| DescribeConnections | Supported | Lists connection test results |

## Quick Start

### Node.js

```typescript
import { DatabaseMigrationServiceClient, CreateReplicationInstanceCommand } from '@aws-sdk/client-database-migration-service';

const client = new DatabaseMigrationServiceClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateReplicationInstanceCommand({
  ReplicationInstanceIdentifier: 'my-repl-instance',
  ReplicationInstanceClass: 'dms.t3.medium',
}));
```

### Python

```python
import boto3

client = boto3.client('dms',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_replication_instance(
    ReplicationInstanceIdentifier='my-repl-instance',
    ReplicationInstanceClass='dms.t3.medium')
```

## Configuration

```yaml
# cloudmock.yml
services:
  dms:
    enabled: true
```

## Known Differences from AWS

- No actual data migration occurs
- Connection tests return stub success results
- Replication task status transitions are simulated
