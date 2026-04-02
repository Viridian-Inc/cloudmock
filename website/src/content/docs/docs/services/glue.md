---
title: Glue
description: AWS Glue emulation in CloudMock
---

## Overview

CloudMock emulates AWS Glue, supporting Data Catalog (databases, tables), crawlers, ETL jobs with job runs, connections, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDatabase | Supported | Creates a Glue database |
| GetDatabase | Supported | Returns database details |
| GetDatabases | Supported | Lists all databases |
| DeleteDatabase | Supported | Deletes a database |
| CreateTable | Supported | Creates a table in a database |
| GetTable | Supported | Returns table details |
| GetTables | Supported | Lists tables in a database |
| DeleteTable | Supported | Deletes a table |
| UpdateTable | Supported | Updates table definition |
| CreateCrawler | Supported | Creates a crawler |
| GetCrawler | Supported | Returns crawler details |
| GetCrawlers | Supported | Lists all crawlers |
| DeleteCrawler | Supported | Deletes a crawler |
| StartCrawler | Supported | Starts a crawler run |
| StopCrawler | Supported | Stops a running crawler |
| CreateJob | Supported | Creates an ETL job |
| GetJob | Supported | Returns job details |
| GetJobs | Supported | Lists all jobs |
| DeleteJob | Supported | Deletes a job |
| StartJobRun | Supported | Starts a job run |
| GetJobRun | Supported | Returns job run details |
| GetJobRuns | Supported | Lists job runs |
| BatchStopJobRun | Supported | Stops multiple job runs |
| CreateConnection | Supported | Creates a connection |
| GetConnection | Supported | Returns connection details |
| GetConnections | Supported | Lists all connections |
| DeleteConnection | Supported | Deletes a connection |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| GetTags | Supported | Returns tags for a resource |

## Quick Start

### Node.js

```typescript
import { GlueClient, CreateDatabaseCommand, CreateTableCommand } from '@aws-sdk/client-glue';

const client = new GlueClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateDatabaseCommand({
  DatabaseInput: { Name: 'my_database' },
}));

await client.send(new CreateTableCommand({
  DatabaseName: 'my_database',
  TableInput: { Name: 'my_table', StorageDescriptor: { Columns: [{ Name: 'id', Type: 'int' }], Location: 's3://my-bucket/data/' } },
}));
```

### Python

```python
import boto3

client = boto3.client('glue',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_database(DatabaseInput={'Name': 'my_database'})

client.create_table(
    DatabaseName='my_database',
    TableInput={'Name': 'my_table', 'StorageDescriptor': {'Columns': [{'Name': 'id', 'Type': 'int'}], 'Location': 's3://my-bucket/data/'}})
```

## Configuration

```yaml
# cloudmock.yml
services:
  glue:
    enabled: true
```

## Known Differences from AWS

- Crawlers do not actually discover schemas from data sources
- ETL jobs do not execute real Spark/Python scripts
- Job run status transitions are simulated
