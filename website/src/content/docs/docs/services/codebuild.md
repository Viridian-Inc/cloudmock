---
title: CodeBuild
description: AWS CodeBuild emulation in CloudMock
---

## Overview

CloudMock emulates AWS CodeBuild, supporting project management, build execution lifecycle, and report groups.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateProject | Supported | Creates a build project |
| BatchGetProjects | Supported | Returns details for multiple projects |
| ListProjects | Supported | Lists all project names |
| UpdateProject | Supported | Updates project configuration |
| DeleteProject | Supported | Deletes a project |
| StartBuild | Supported | Starts a build with simulated phases |
| BatchGetBuilds | Supported | Returns details for multiple builds |
| ListBuilds | Supported | Lists all build IDs across all projects |
| ListBuildsForProject | Supported | Lists builds for a project |
| StopBuild | Supported | Stops a running build |
| CreateReportGroup | Supported | Creates a report group |
| BatchGetReportGroups | Supported | Returns details for report groups |
| ListReportGroups | Supported | Lists all report groups |
| DeleteReportGroup | Supported | Deletes a report group |

## Quick Start

### Node.js

```typescript
import { CodeBuildClient, CreateProjectCommand, StartBuildCommand } from '@aws-sdk/client-codebuild';

const client = new CodeBuildClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateProjectCommand({
  name: 'my-project',
  source: { type: 'GITHUB', location: 'https://github.com/example/repo' },
  artifacts: { type: 'NO_ARTIFACTS' },
  environment: { type: 'LINUX_CONTAINER', computeType: 'BUILD_GENERAL1_SMALL', image: 'aws/codebuild/standard:7.0' },
  serviceRole: 'arn:aws:iam::000000000000:role/codebuild-role',
}));

const { build } = await client.send(new StartBuildCommand({ projectName: 'my-project' }));
console.log(build.id);
```

### Python

```python
import boto3

client = boto3.client('codebuild',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_project(
    name='my-project',
    source={'type': 'GITHUB', 'location': 'https://github.com/example/repo'},
    artifacts={'type': 'NO_ARTIFACTS'},
    environment={'type': 'LINUX_CONTAINER', 'computeType': 'BUILD_GENERAL1_SMALL', 'image': 'aws/codebuild/standard:7.0'},
    serviceRole='arn:aws:iam::000000000000:role/codebuild-role')

response = client.start_build(projectName='my-project')
print(response['build']['id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  codebuild:
    enabled: true
```

## Known Differences from AWS

- Builds are not actually executed; phases are simulated
- Build logs are not generated
- Source code is not pulled from repositories
