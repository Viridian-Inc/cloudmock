---
title: SageMaker
description: Amazon SageMaker emulation in CloudMock
---

## Overview

CloudMock emulates Amazon SageMaker, supporting notebook instances, training jobs, models, endpoint configurations, endpoints, processing jobs, transform jobs, inference invocation, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateNotebookInstance | Supported | Creates a notebook instance |
| DescribeNotebookInstance | Supported | Returns notebook instance details |
| ListNotebookInstances | Supported | Lists notebook instances |
| DeleteNotebookInstance | Supported | Deletes a notebook instance |
| StartNotebookInstance | Supported | Starts a notebook instance |
| StopNotebookInstance | Supported | Stops a notebook instance |
| UpdateNotebookInstance | Supported | Updates a notebook instance |
| CreateTrainingJob | Supported | Creates a training job |
| DescribeTrainingJob | Supported | Returns training job details |
| ListTrainingJobs | Supported | Lists training jobs |
| StopTrainingJob | Supported | Stops a training job |
| CreateModel | Supported | Creates a model |
| DescribeModel | Supported | Returns model details |
| ListModels | Supported | Lists models |
| DeleteModel | Supported | Deletes a model |
| CreateEndpointConfig | Supported | Creates an endpoint configuration |
| DescribeEndpointConfig | Supported | Returns endpoint config details |
| ListEndpointConfigs | Supported | Lists endpoint configurations |
| DeleteEndpointConfig | Supported | Deletes an endpoint configuration |
| CreateEndpoint | Supported | Creates an endpoint |
| DescribeEndpoint | Supported | Returns endpoint details |
| ListEndpoints | Supported | Lists endpoints |
| DeleteEndpoint | Supported | Deletes an endpoint |
| UpdateEndpoint | Supported | Updates an endpoint |
| CreateProcessingJob | Supported | Creates a processing job |
| DescribeProcessingJob | Supported | Returns processing job details |
| ListProcessingJobs | Supported | Lists processing jobs |
| StopProcessingJob | Supported | Stops a processing job |
| CreateTransformJob | Supported | Creates a batch transform job |
| DescribeTransformJob | Supported | Returns transform job details |
| ListTransformJobs | Supported | Lists transform jobs |
| StopTransformJob | Supported | Stops a transform job |
| InvokeEndpoint | Supported | Returns a stub inference response |
| AddTags | Supported | Adds tags to a resource |
| DeleteTags | Supported | Removes tags from a resource |
| ListTags | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { SageMakerClient, CreateNotebookInstanceCommand } from '@aws-sdk/client-sagemaker';

const client = new SageMakerClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateNotebookInstanceCommand({
  NotebookInstanceName: 'my-notebook',
  InstanceType: 'ml.t3.medium',
  RoleArn: 'arn:aws:iam::000000000000:role/sagemaker-role',
}));
```

### Python

```python
import boto3

client = boto3.client('sagemaker',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_notebook_instance(
    NotebookInstanceName='my-notebook',
    InstanceType='ml.t3.medium',
    RoleArn='arn:aws:iam::000000000000:role/sagemaker-role')
```

## Configuration

```yaml
# cloudmock.yml
services:
  sagemaker:
    enabled: true
```

## Known Differences from AWS

- No actual ML infrastructure is provisioned
- Training and processing jobs do not execute real workloads
- InvokeEndpoint returns stub responses, not model inference
- Notebook instances are metadata-only
