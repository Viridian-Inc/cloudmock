---
title: RDS
description: Amazon RDS (Relational Database Service) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon RDS management operations, supporting DB instance, cluster, snapshot, and subnet group lifecycle. Instances are metadata-only records -- no database engine is started.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDBInstance | Supported | Creates a DB instance record |
| DeleteDBInstance | Supported | Deletes a DB instance |
| DescribeDBInstances | Supported | Returns all DB instances |
| ModifyDBInstance | Supported | Updates instance attributes |
| CreateDBCluster | Supported | Creates an Aurora cluster |
| DeleteDBCluster | Supported | Deletes a cluster |
| DescribeDBClusters | Supported | Returns all clusters |
| CreateDBSnapshot | Supported | Creates a snapshot of a DB instance |
| DeleteDBSnapshot | Supported | Deletes a snapshot |
| DescribeDBSnapshots | Supported | Returns all snapshots |
| CreateDBSubnetGroup | Supported | Creates a subnet group |
| DescribeDBSubnetGroups | Supported | Returns all subnet groups |
| DeleteDBSubnetGroup | Supported | Deletes a subnet group |
| AddTagsToResource | Supported | Adds tags to a resource |
| RemoveTagsFromResource | Supported | Removes tags |
| ListTagsForResource | Supported | Returns tags for a resource |

## Quick Start

### curl

```bash
# Create a DB instance
curl -X POST "http://localhost:4566/?Action=CreateDBInstance&DBInstanceIdentifier=my-db&DBInstanceClass=db.t3.micro&Engine=mysql&MasterUsername=admin&MasterUserPassword=Password123&AllocatedStorage=20"

# Describe instances
curl -X POST "http://localhost:4566/?Action=DescribeDBInstances"
```

### Node.js

```typescript
import { RDSClient, CreateDBInstanceCommand, DescribeDBInstancesCommand } from '@aws-sdk/client-rds';

const rds = new RDSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await rds.send(new CreateDBInstanceCommand({
  DBInstanceIdentifier: 'test-db',
  DBInstanceClass: 'db.t3.micro',
  Engine: 'postgres',
  MasterUsername: 'postgres',
  MasterUserPassword: 'postgres123',
  AllocatedStorage: 20,
}));

const { DBInstances } = await rds.send(new DescribeDBInstancesCommand({}));
DBInstances?.forEach(db => console.log(db.DBInstanceIdentifier, db.DBInstanceStatus));
```

### Python

```python
import boto3

rds = boto3.client('rds', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

rds.create_db_instance(
    DBInstanceIdentifier='test-db',
    DBInstanceClass='db.t3.micro',
    Engine='postgres',
    MasterUsername='postgres',
    MasterUserPassword='postgres123',
    AllocatedStorage=20,
)

response = rds.describe_db_instances()
for db in response['DBInstances']:
    print(db['DBInstanceIdentifier'], db['DBInstanceStatus'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  rds:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- DB instances are **metadata-only records**. No actual database engine is started.
- **Connection strings and endpoints** returned in `DescribeDBInstances` are synthetic and not connectable.
- **Parameter groups**, option groups, and enhanced monitoring are accepted but not enforced.
- **Aurora serverless v2**, Proxy, and Blue/Green deployments are not implemented.
- Instance status immediately transitions to `available` after creation.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| DBInstanceNotFound | 404 | The specified DB instance does not exist |
| DBInstanceAlreadyExists | 400 | A DB instance with this identifier already exists |
| DBSnapshotNotFound | 404 | The specified snapshot does not exist |
| InvalidDBInstanceState | 400 | The DB instance is not in a valid state for this operation |
