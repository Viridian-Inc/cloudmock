---
title: CodeDeploy
description: AWS CodeDeploy emulation in CloudMock
---

## Overview

CloudMock emulates AWS CodeDeploy, supporting application, deployment group, and deployment lifecycle management.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an application |
| GetApplication | Supported | Returns application details |
| ListApplications | Supported | Lists all applications |
| DeleteApplication | Supported | Deletes an application |
| CreateDeploymentGroup | Supported | Creates a deployment group |
| GetDeploymentGroup | Supported | Returns deployment group details |
| ListDeploymentGroups | Supported | Lists deployment groups |
| DeleteDeploymentGroup | Supported | Deletes a deployment group |
| UpdateDeploymentGroup | Supported | Updates deployment group configuration |
| CreateDeployment | Supported | Creates a deployment |
| GetDeployment | Supported | Returns deployment details |
| ListDeployments | Supported | Lists deployments |
| StopDeployment | Supported | Stops a deployment |
| BatchGetDeployments | Supported | Returns details for multiple deployments |
| BatchGetDeploymentTargets | Supported | Returns deployment target details |
| AddTagsToOnPremisesInstances | Supported | Tags on-premises instances |
| RemoveTagsFromOnPremisesInstances | Supported | Untags on-premises instances |

## Quick Start

### Node.js

```typescript
import { CodeDeployClient, CreateApplicationCommand, CreateDeploymentCommand } from '@aws-sdk/client-codedeploy';

const client = new CodeDeployClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateApplicationCommand({
  applicationName: 'my-app',
  computePlatform: 'Server',
}));

const { deploymentId } = await client.send(new CreateDeploymentCommand({
  applicationName: 'my-app',
  deploymentGroupName: 'my-group',
  revision: { revisionType: 'S3', s3Location: { bucket: 'my-bucket', key: 'app.zip', bundleType: 'zip' } },
}));
console.log(deploymentId);
```

### Python

```python
import boto3

client = boto3.client('codedeploy',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_application(applicationName='my-app', computePlatform='Server')

response = client.create_deployment(
    applicationName='my-app',
    deploymentGroupName='my-group',
    revision={'revisionType': 'S3', 's3Location': {'bucket': 'my-bucket', 'key': 'app.zip', 'bundleType': 'zip'}})
print(response['deploymentId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  codedeploy:
    enabled: true
```

## Known Differences from AWS

- Deployments are not actually executed on target instances
- Deployment status transitions are simulated
- On-premises instance registration is tag-only
