---
title: MediaConvert
description: AWS Elemental MediaConvert emulation in CloudMock
---

## Overview

CloudMock emulates AWS Elemental MediaConvert, supporting transcoding jobs, job templates, presets, and queues.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateJob | Supported | Creates a transcoding job |
| GetJob | Supported | Returns job details |
| ListJobs | Supported | Lists all jobs |
| CancelJob | Supported | Cancels a pending job |
| CreateJobTemplate | Supported | Creates a job template |
| GetJobTemplate | Supported | Returns template details |
| ListJobTemplates | Supported | Lists all job templates |
| DeleteJobTemplate | Supported | Deletes a job template |
| CreatePreset | Supported | Creates an output preset |
| GetPreset | Supported | Returns preset details |
| ListPresets | Supported | Lists all presets |
| DeletePreset | Supported | Deletes a preset |
| CreateQueue | Supported | Creates a queue |
| GetQueue | Supported | Returns queue details |
| ListQueues | Supported | Lists all queues |
| DeleteQueue | Supported | Deletes a queue |

## Quick Start

### Node.js

```typescript
import { MediaConvertClient, CreateJobCommand } from '@aws-sdk/client-mediaconvert';

const client = new MediaConvertClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { Job } = await client.send(new CreateJobCommand({
  Role: 'arn:aws:iam::000000000000:role/mediaconvert-role',
  Settings: {
    Inputs: [{ FileInput: 's3://my-bucket/input.mp4' }],
    OutputGroups: [{ OutputGroupSettings: { Type: 'FILE_GROUP_SETTINGS', FileGroupSettings: { Destination: 's3://my-bucket/output/' } }, Outputs: [{ ContainerSettings: { Container: 'MP4' } }] }],
  },
}));
console.log(Job.Id);
```

### Python

```python
import boto3

client = boto3.client('mediaconvert',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_job(
    Role='arn:aws:iam::000000000000:role/mediaconvert-role',
    Settings={
        'Inputs': [{'FileInput': 's3://my-bucket/input.mp4'}],
        'OutputGroups': [{'OutputGroupSettings': {'Type': 'FILE_GROUP_SETTINGS', 'FileGroupSettings': {'Destination': 's3://my-bucket/output/'}}, 'Outputs': [{'ContainerSettings': {'Container': 'MP4'}}]}],
    })
print(response['Job']['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  mediaconvert:
    enabled: true
```

## Known Differences from AWS

- Jobs do not perform actual media transcoding
- Job status transitions are simulated
- No output files are generated
