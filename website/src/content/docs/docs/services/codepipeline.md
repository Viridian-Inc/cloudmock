---
title: CodePipeline
description: AWS CodePipeline emulation in CloudMock
---

## Overview

CloudMock emulates AWS CodePipeline, supporting pipeline CRUD, execution management, approval actions, stage retries, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreatePipeline | Supported | Creates a pipeline |
| GetPipeline | Supported | Returns pipeline definition |
| ListPipelines | Supported | Lists all pipelines |
| UpdatePipeline | Supported | Updates pipeline definition |
| DeletePipeline | Supported | Deletes a pipeline |
| GetPipelineState | Supported | Returns current pipeline state |
| GetPipelineExecution | Supported | Returns execution details |
| ListPipelineExecutions | Supported | Lists pipeline executions |
| StartPipelineExecution | Supported | Starts a pipeline execution |
| StopPipelineExecution | Supported | Stops a pipeline execution |
| PutApprovalResult | Supported | Approves or rejects a manual approval |
| RetryStageExecution | Supported | Retries a failed stage |
| TagResource | Supported | Adds tags to a pipeline |
| UntagResource | Supported | Removes tags from a pipeline |
| ListTagsForResource | Supported | Lists tags for a pipeline |

## Quick Start

### Node.js

```typescript
import { CodePipelineClient, CreatePipelineCommand } from '@aws-sdk/client-codepipeline';

const client = new CodePipelineClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreatePipelineCommand({
  pipeline: {
    name: 'my-pipeline',
    roleArn: 'arn:aws:iam::000000000000:role/pipeline-role',
    stages: [
      { name: 'Source', actions: [{ name: 'Source', actionTypeId: { category: 'Source', owner: 'AWS', provider: 'S3', version: '1' }, configuration: { S3Bucket: 'my-bucket', S3ObjectKey: 'source.zip' }, outputArtifacts: [{ name: 'SourceOutput' }] }] },
      { name: 'Deploy', actions: [{ name: 'Deploy', actionTypeId: { category: 'Deploy', owner: 'AWS', provider: 'S3', version: '1' }, inputArtifacts: [{ name: 'SourceOutput' }], configuration: { BucketName: 'deploy-bucket', Extract: 'true' } }] },
    ],
    artifactStore: { type: 'S3', location: 'artifact-bucket' },
  },
}));
```

### Python

```python
import boto3

client = boto3.client('codepipeline',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_pipeline(pipeline={
    'name': 'my-pipeline',
    'roleArn': 'arn:aws:iam::000000000000:role/pipeline-role',
    'stages': [
        {'name': 'Source', 'actions': [{'name': 'Source', 'actionTypeId': {'category': 'Source', 'owner': 'AWS', 'provider': 'S3', 'version': '1'}, 'configuration': {'S3Bucket': 'my-bucket', 'S3ObjectKey': 'source.zip'}, 'outputArtifacts': [{'name': 'SourceOutput'}]}]},
        {'name': 'Deploy', 'actions': [{'name': 'Deploy', 'actionTypeId': {'category': 'Deploy', 'owner': 'AWS', 'provider': 'S3', 'version': '1'}, 'inputArtifacts': [{'name': 'SourceOutput'}], 'configuration': {'BucketName': 'deploy-bucket', 'Extract': 'true'}}]},
    ],
    'artifactStore': {'type': 'S3', 'location': 'artifact-bucket'},
})
```

## Configuration

```yaml
# cloudmock.yml
services:
  codepipeline:
    enabled: true
```

## Known Differences from AWS

- Pipeline executions do not actually run actions against real services
- Stage transitions are simulated
- Artifact stores are referenced but artifacts are not transferred
