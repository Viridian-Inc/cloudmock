---
title: Batch
description: AWS Batch emulation in CloudMock
---

## Overview

CloudMock emulates AWS Batch, supporting compute environments, job queues, job definitions, and job submission/management.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateComputeEnvironment | Supported | Creates a compute environment |
| DescribeComputeEnvironments | Supported | Lists compute environments |
| DeleteComputeEnvironment | Supported | Deletes a compute environment |
| CreateJobQueue | Supported | Creates a job queue |
| DescribeJobQueues | Supported | Lists job queues |
| DeleteJobQueue | Supported | Deletes a job queue |
| RegisterJobDefinition | Supported | Registers a job definition |
| DescribeJobDefinitions | Supported | Lists job definitions |
| DeregisterJobDefinition | Supported | Deregisters a job definition |
| SubmitJob | Supported | Submits a job for execution |
| DescribeJobs | Supported | Returns job details |
| ListJobs | Supported | Lists jobs in a queue |
| CancelJob | Supported | Cancels a pending job |
| TerminateJob | Supported | Terminates a running job |

## Quick Start

### Node.js

```typescript
import { BatchClient, SubmitJobCommand } from '@aws-sdk/client-batch';

const client = new BatchClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { jobId } = await client.send(new SubmitJobCommand({
  jobName: 'my-job',
  jobQueue: 'my-queue',
  jobDefinition: 'my-job-def',
}));
console.log(jobId);
```

### Python

```python
import boto3

client = boto3.client('batch',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.submit_job(
    jobName='my-job',
    jobQueue='my-queue',
    jobDefinition='my-job-def')
print(response['jobId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  batch:
    enabled: true
```

## Known Differences from AWS

- Jobs are not actually executed on compute resources
- Job status transitions are simulated
- Compute environments do not provision real infrastructure
