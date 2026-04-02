---
title: S3 Tables
description: Amazon S3 Tables emulation in CloudMock
---

## Overview

CloudMock emulates Amazon S3 Tables, supporting table bucket management, table CRUD operations, and table policy management.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateTableBucket | Supported | Creates a table bucket |
| GetTableBucket | Supported | Returns table bucket details |
| ListTableBuckets | Supported | Lists all table buckets |
| DeleteTableBucket | Supported | Deletes a table bucket |
| CreateTable | Supported | Creates a table in a bucket |
| GetTable | Supported | Returns table details |
| ListTables | Supported | Lists tables in a bucket |
| DeleteTable | Supported | Deletes a table |
| PutTablePolicy | Supported | Sets a table policy |
| GetTablePolicy | Supported | Returns table policy |
| DeleteTablePolicy | Supported | Removes a table policy |

## Quick Start

### Node.js

```typescript
import { S3TablesClient, CreateTableBucketCommand, CreateTableCommand } from '@aws-sdk/client-s3tables';

const client = new S3TablesClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { arn } = await client.send(new CreateTableBucketCommand({
  name: 'my-table-bucket',
}));

await client.send(new CreateTableCommand({
  tableBucketARN: arn,
  namespace: 'default',
  name: 'my-table',
  format: 'ICEBERG',
}));
```

### Python

```python
import boto3

client = boto3.client('s3tables',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_table_bucket(name='my-table-bucket')
arn = response['arn']

client.create_table(
    tableBucketARN=arn,
    namespace='default',
    name='my-table',
    format='ICEBERG')
```

## Configuration

```yaml
# cloudmock.yml
services:
  s3tables:
    enabled: true
```

## Known Differences from AWS

- Tables do not store actual Iceberg data
- Table policies are stored but not enforced
- No integration with analytics engines
