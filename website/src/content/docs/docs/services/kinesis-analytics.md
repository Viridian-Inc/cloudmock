---
title: Kinesis Data Analytics
description: Amazon Kinesis Data Analytics emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Kinesis Data Analytics (v2), supporting application lifecycle, input/output management, snapshots, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an analytics application |
| DescribeApplication | Supported | Returns application details |
| ListApplications | Supported | Lists all applications |
| DeleteApplication | Supported | Deletes an application |
| UpdateApplication | Supported | Updates application configuration |
| StartApplication | Supported | Starts an application |
| StopApplication | Supported | Stops an application |
| AddApplicationInput | Supported | Adds an input to an application |
| AddApplicationOutput | Supported | Adds an output to an application |
| DeleteApplicationOutput | Supported | Removes an output |
| CreateApplicationSnapshot | Supported | Creates an application snapshot |
| ListApplicationSnapshots | Supported | Lists application snapshots |
| DeleteApplicationSnapshot | Supported | Deletes a snapshot |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { KinesisAnalyticsV2Client, CreateApplicationCommand } from '@aws-sdk/client-kinesis-analytics-v2';

const client = new KinesisAnalyticsV2Client({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateApplicationCommand({
  ApplicationName: 'my-analytics-app',
  RuntimeEnvironment: 'FLINK-1_15',
  ServiceExecutionRole: 'arn:aws:iam::000000000000:role/analytics-role',
}));
```

### Python

```python
import boto3

client = boto3.client('kinesisanalyticsv2',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_application(
    ApplicationName='my-analytics-app',
    RuntimeEnvironment='FLINK-1_15',
    ServiceExecutionRole='arn:aws:iam::000000000000:role/analytics-role')
```

## Configuration

```yaml
# cloudmock.yml
services:
  kinesisanalytics:
    enabled: true
```

## Known Differences from AWS

- Applications do not actually process streaming data
- Flink/SQL runtime is not provisioned
- Application status transitions are simulated
