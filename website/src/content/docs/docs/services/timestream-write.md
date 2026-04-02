---
title: Timestream Write
description: Amazon Timestream Write emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Timestream Write, supporting database and table management, record writing, querying, endpoint discovery, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDatabase | Supported | Creates a database |
| DescribeDatabase | Supported | Returns database details |
| ListDatabases | Supported | Lists all databases |
| UpdateDatabase | Supported | Updates database configuration |
| DeleteDatabase | Supported | Deletes a database |
| CreateTable | Supported | Creates a table |
| DescribeTable | Supported | Returns table details |
| ListTables | Supported | Lists tables in a database |
| UpdateTable | Supported | Updates table configuration |
| DeleteTable | Supported | Deletes a table |
| WriteRecords | Supported | Writes time-series records |
| Query | Supported | Executes a query (stub results) |
| DescribeEndpoints | Supported | Returns service endpoints |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { TimestreamWriteClient, CreateDatabaseCommand, CreateTableCommand, WriteRecordsCommand } from '@aws-sdk/client-timestream-write';

const client = new TimestreamWriteClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateDatabaseCommand({ DatabaseName: 'my-db' }));
await client.send(new CreateTableCommand({ DatabaseName: 'my-db', TableName: 'my-table' }));
await client.send(new WriteRecordsCommand({
  DatabaseName: 'my-db',
  TableName: 'my-table',
  Records: [{ Dimensions: [{ Name: 'host', Value: 'server-1' }], MeasureName: 'cpu', MeasureValue: '85.5', MeasureValueType: 'DOUBLE', Time: String(Date.now()) }],
}));
```

### Python

```python
import boto3
import time

client = boto3.client('timestream-write',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_database(DatabaseName='my-db')
client.create_table(DatabaseName='my-db', TableName='my-table')
client.write_records(
    DatabaseName='my-db',
    TableName='my-table',
    Records=[{'Dimensions': [{'Name': 'host', 'Value': 'server-1'}], 'MeasureName': 'cpu', 'MeasureValue': '85.5', 'MeasureValueType': 'DOUBLE', 'Time': str(int(time.time() * 1000))}])
```

## Configuration

```yaml
# cloudmock.yml
services:
  timestreamwrite:
    enabled: true
```

## Known Differences from AWS

- Records are stored in-memory only
- Query returns stub results, not actual time-series aggregations
- Retention policies are stored but not enforced
