---
title: Batch
description: AWS Batch emulation in CloudMock
---

## Overview

CloudMock emulates AWS Batch, supporting compute environments, job queues, job definitions, scheduling policies, job submission/management, and resource tagging. Job state transitions are simulated synchronously in test mode.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateComputeEnvironment | Supported | MANAGED and UNMANAGED types; state machine: CREATING → VALID |
| DescribeComputeEnvironments | Supported | Filter by name or ARN |
| UpdateComputeEnvironment | Supported | Update state or service role |
| DeleteComputeEnvironment | Supported | Stops lifecycle machine |
| CreateJobQueue | Supported | Priority-ordered compute environment list |
| DescribeJobQueues | Supported | Filter by name or ARN |
| UpdateJobQueue | Supported | Update state, priority, or CE order |
| DeleteJobQueue | Supported | |
| RegisterJobDefinition | Supported | Multi-revision support per definition name |
| DescribeJobDefinitions | Supported | Filter by name or status |
| DeregisterJobDefinition | Supported | Marks revision as INACTIVE |
| SubmitJob | Supported | Supports dependsOn chains |
| DescribeJobs | Supported | Returns full job detail |
| ListJobs | Supported | Filter by queue and status |
| CancelJob | Supported | Only cancels non-terminal jobs |
| TerminateJob | Supported | Force-stops running jobs |
| CreateSchedulingPolicy | Supported | Fair-share scheduling policies |
| DescribeSchedulingPolicies | Supported | Filter by ARN |
| UpdateSchedulingPolicy | Supported | Update fair-share parameters |
| DeleteSchedulingPolicy | Supported | |
| TagResource | Supported | Tag any Batch resource by ARN |
| UntagResource | Supported | |
| ListTagsForResource | Supported | |

## Job State Machine

```
SUBMITTED → PENDING → RUNNABLE → STARTING → RUNNING → SUCCEEDED / FAILED
```

Transitions are simulated with configurable delays. In test mode (instant lifecycle), all transitions happen synchronously.

## Quick Start

### Node.js

```typescript
import { BatchClient, CreateComputeEnvironmentCommand, CreateJobQueueCommand, RegisterJobDefinitionCommand, SubmitJobCommand } from '@aws-sdk/client-batch';

const client = new BatchClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateComputeEnvironmentCommand({
  computeEnvironmentName: 'my-ce',
  type: 'MANAGED',
  state: 'ENABLED',
  computeResources: { type: 'EC2', maxvCpus: 256, instanceTypes: ['m5.xlarge'] },
}));

await client.send(new CreateJobQueueCommand({
  jobQueueName: 'my-queue',
  state: 'ENABLED',
  priority: 10,
  computeEnvironmentOrder: [{ computeEnvironment: 'my-ce', order: 1 }],
}));

await client.send(new RegisterJobDefinitionCommand({
  jobDefinitionName: 'my-job-def',
  type: 'container',
  containerProperties: { image: 'alpine:latest', vcpus: 1, memory: 512 },
}));

const { jobId } = await client.send(new SubmitJobCommand({
  jobName: 'my-job',
  jobQueue: 'my-queue',
  jobDefinition: 'my-job-def:1',
}));
console.log('Submitted job:', jobId);
```

### Python

```python
import boto3

client = boto3.client('batch',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_compute_environment(
    computeEnvironmentName='my-ce',
    type='MANAGED',
    state='ENABLED',
    computeResources={'type': 'EC2', 'maxvCpus': 256, 'instanceTypes': ['m5.xlarge']})

response = client.submit_job(
    jobName='my-job',
    jobQueue='my-queue',
    jobDefinition='my-job-def:1')
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
- Job status transitions are simulated (not driven by real compute)
- Compute environments do not provision real EC2 instances
- Fair-share scheduling policies are stored but not enforced during job dispatch
