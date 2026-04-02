---
title: Athena
description: Amazon Athena emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Athena, supporting workgroup management, named queries, and query execution lifecycle operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateWorkGroup | Supported | Creates a workgroup |
| GetWorkGroup | Supported | Returns workgroup details |
| ListWorkGroups | Supported | Lists all workgroups |
| UpdateWorkGroup | Supported | Updates workgroup configuration |
| DeleteWorkGroup | Supported | Deletes a workgroup |
| CreateNamedQuery | Supported | Saves a named query |
| GetNamedQuery | Supported | Returns a named query |
| ListNamedQueries | Supported | Lists named query IDs |
| DeleteNamedQuery | Supported | Deletes a named query |
| StartQueryExecution | Supported | Starts a query execution |
| GetQueryExecution | Supported | Returns query execution status |
| ListQueryExecutions | Supported | Lists query execution IDs |
| StopQueryExecution | Supported | Cancels a running query |
| GetQueryResults | Supported | Returns stub query results |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |

## Quick Start

### Node.js

```typescript
import { AthenaClient, StartQueryExecutionCommand, GetQueryExecutionCommand } from '@aws-sdk/client-athena';

const client = new AthenaClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { QueryExecutionId } = await client.send(new StartQueryExecutionCommand({
  QueryString: 'SELECT * FROM my_table',
  WorkGroup: 'primary',
}));

const result = await client.send(new GetQueryExecutionCommand({ QueryExecutionId }));
console.log(result.QueryExecution.Status);
```

### Python

```python
import boto3

client = boto3.client('athena',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.start_query_execution(
    QueryString='SELECT * FROM my_table',
    WorkGroup='primary')

result = client.get_query_execution(
    QueryExecutionId=response['QueryExecutionId'])
print(result['QueryExecution']['Status'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  athena:
    enabled: true
```

## Known Differences from AWS

- Queries are not actually executed against any data source
- Query results return stub/empty data
- Query execution status transitions are simulated
