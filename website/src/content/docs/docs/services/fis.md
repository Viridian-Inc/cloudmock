---
title: FIS
description: AWS Fault Injection Simulator emulation in CloudMock
---

## Overview

CloudMock emulates AWS Fault Injection Simulator (FIS), supporting experiment template management and experiment execution lifecycle.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateExperimentTemplate | Supported | Creates an experiment template |
| GetExperimentTemplate | Supported | Returns template details |
| ListExperimentTemplates | Supported | Lists all templates |
| UpdateExperimentTemplate | Supported | Updates description, roleArn, and tags of a template |
| DeleteExperimentTemplate | Supported | Deletes a template |
| StartExperiment | Supported | Starts an experiment from a template |
| GetExperiment | Supported | Returns experiment details with action/target states |
| ListExperiments | Supported | Lists all experiments |
| StopExperiment | Supported | Stops a running experiment |
| ListTargetResourceTypes | Supported | Returns supported target resource types |
| ListActions | Supported | Returns available FIS actions |
| TagResource | Supported | Adds tags to a template or experiment |
| UntagResource | Supported | Removes tags from a template or experiment |
| ListTagsForResource | Supported | Lists tags for a template or experiment |

## Quick Start

### Node.js

```typescript
import { FisClient, CreateExperimentTemplateCommand, StartExperimentCommand } from '@aws-sdk/client-fis';

const client = new FisClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { experimentTemplate } = await client.send(new CreateExperimentTemplateCommand({
  description: 'Stop instances experiment',
  roleArn: 'arn:aws:iam::000000000000:role/fis-role',
  stopConditions: [{ source: 'none' }],
  actions: { stopInstances: { actionId: 'aws:ec2:stop-instances', targets: { Instances: 'myTargets' } } },
  targets: { myTargets: { resourceType: 'aws:ec2:instance', selectionMode: 'ALL', resourceTags: { env: 'test' } } },
}));

await client.send(new StartExperimentCommand({
  experimentTemplateId: experimentTemplate.id,
}));
```

### Python

```python
import boto3

client = boto3.client('fis',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

template = client.create_experiment_template(
    description='Stop instances experiment',
    roleArn='arn:aws:iam::000000000000:role/fis-role',
    stopConditions=[{'source': 'none'}],
    actions={'stopInstances': {'actionId': 'aws:ec2:stop-instances', 'targets': {'Instances': 'myTargets'}}},
    targets={'myTargets': {'resourceType': 'aws:ec2:instance', 'selectionMode': 'ALL', 'resourceTags': {'env': 'test'}}})

client.start_experiment(
    experimentTemplateId=template['experimentTemplate']['id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  fis:
    enabled: true
```

## Known Differences from AWS

- Experiments do not actually inject faults into resources
- Experiment status transitions are simulated
- Stop conditions are not evaluated
