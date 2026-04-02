---
title: Database Migration Service (DMS)
description: AWS DMS emulation in CloudMock
---

## Overview

CloudMock emulates AWS Database Migration Service, supporting replication instances, endpoints, replication tasks, subnet groups, SSL certificates, and tagging. Replication instance lifecycle transitions through creating → available, and task lifecycle through ready → starting → running → stopping → stopped.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateReplicationInstance | Supported | Lifecycle: creating → available |
| DescribeReplicationInstances | Supported | Lists all instances |
| ModifyReplicationInstance | Supported | Update class and MultiAZ flag |
| DeleteReplicationInstance | Supported | Sets status to deleting |
| CreateEndpoint | Supported | Source or target endpoint |
| DescribeEndpoints | Supported | Lists all endpoints |
| ModifyEndpoint | Supported | Update server, db, username, port |
| DeleteEndpoint | Supported | Deletes an endpoint |
| TestConnection | Supported | Always returns successful |
| CreateReplicationTask | Supported | Links source, target, instance |
| DescribeReplicationTasks | Supported | Lists all tasks |
| StartReplicationTask | Supported | Transitions to starting → running |
| StopReplicationTask | Supported | Transitions to stopping → stopped |
| DeleteReplicationTask | Supported | Deletes a task |
| CreateReplicationSubnetGroup | Supported | Creates subnet group |
| DescribeReplicationSubnetGroups | Supported | Lists subnet groups |
| ModifyReplicationSubnetGroup | Supported | Updates description and subnet IDs |
| DeleteReplicationSubnetGroup | Supported | Deletes subnet group |
| CreateCertificate | Supported | Creates SSL certificate |
| DescribeCertificates | Supported | Lists certificates |
| DeleteCertificate | Supported | Deletes a certificate |
| AddTagsToResource | Supported | Adds tags by ARN |
| RemoveTagsFromResource | Supported | Removes tags by key |
| ListTagsForResource | Supported | Lists tags by ARN |
| CreateEventSubscription | Supported | Creates event subscription |
| DescribeEventSubscriptions | Supported | Lists event subscriptions |
| DeleteEventSubscription | Supported | Deletes event subscription |
| DescribeConnections | Supported | Lists connection test results |

## Quick Start

```python
import boto3

client = boto3.client('dms',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

# Create replication instance
inst = client.create_replication_instance(
    ReplicationInstanceIdentifier='my-rep-inst',
    ReplicationInstanceClass='dms.t3.medium',
)

# Create source and target endpoints
src = client.create_endpoint(
    EndpointIdentifier='source-db',
    EndpointType='source',
    EngineName='mysql',
    ServerName='source.example.com',
    Port=3306,
    DatabaseName='mydb',
    Username='admin',
)
tgt = client.create_endpoint(
    EndpointIdentifier='target-db',
    EndpointType='target',
    EngineName='postgres',
    ServerName='target.example.com',
    Port=5432,
)

# Create and start task
task = client.create_replication_task(
    ReplicationTaskIdentifier='full-load',
    SourceEndpointArn=src['Endpoint']['EndpointArn'],
    TargetEndpointArn=tgt['Endpoint']['EndpointArn'],
    ReplicationInstanceArn=inst['ReplicationInstance']['ReplicationInstanceArn'],
    MigrationType='full-load',
    TableMappings='{}',
)
```

## Configuration

```yaml
services:
  dms:
    enabled: true
```

## Known Differences from AWS

- TestConnection always succeeds regardless of endpoint configuration
- Replication instance lifecycle transitions happen asynchronously via lifecycle package
- No actual data migration occurs
