---
title: MWAA (Airflow)
description: Amazon Managed Workflows for Apache Airflow emulation in CloudMock
---

## Overview

CloudMock emulates Amazon MWAA (Managed Workflows for Apache Airflow), supporting environment lifecycle management, CLI/web login token generation, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateEnvironment | Supported | Creates an Airflow environment |
| GetEnvironment | Supported | Returns environment details |
| ListEnvironments | Supported | Lists all environments |
| UpdateEnvironment | Supported | Updates environment configuration |
| DeleteEnvironment | Supported | Deletes an environment |
| CreateCliToken | Supported | Generates a stub CLI token |
| CreateWebLoginToken | Supported | Generates a stub web login token |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { MWAAClient, CreateEnvironmentCommand } from '@aws-sdk/client-mwaa';

const client = new MWAAClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateEnvironmentCommand({
  Name: 'my-airflow-env',
  ExecutionRoleArn: 'arn:aws:iam::000000000000:role/airflow-role',
  SourceBucketArn: 'arn:aws:s3:::my-dags-bucket',
  DagS3Path: 'dags/',
}));
```

### Python

```python
import boto3

client = boto3.client('mwaa',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_environment(
    Name='my-airflow-env',
    ExecutionRoleArn='arn:aws:iam::000000000000:role/airflow-role',
    SourceBucketArn='arn:aws:s3:::my-dags-bucket',
    DagS3Path='dags/')
```

## Configuration

```yaml
# cloudmock.yml
services:
  airflow:
    enabled: true
```

## Known Differences from AWS

- No actual Airflow cluster is provisioned
- CLI and web login tokens are stubs and cannot be used with real Airflow
- Environment status transitions are simulated
