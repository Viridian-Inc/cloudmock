---
title: Serverless Application Repository
description: AWS Serverless Application Repository emulation in CloudMock
---

## Overview

CloudMock emulates the AWS Serverless Application Repository (SAR), supporting application publishing, versioning, and deployment via CloudFormation change sets.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an application with semantic version (X.Y.Z format required) |
| GetApplication | Supported | Returns application details |
| ListApplications | Supported | Lists all applications |
| UpdateApplication | Supported | Updates description, author, homepage URL, and labels |
| DeleteApplication | Supported | Deletes an application and all its versions |
| CreateApplicationVersion | Supported | Creates a new semantic version for an application |
| ListApplicationVersions | Supported | Lists versions sorted by semantic version |
| CreateCloudFormationChangeSet | Supported | Creates a change set for deploying the application |

## Quick Start

### Node.js

```typescript
import {
  ServerlessApplicationRepositoryClient,
  CreateApplicationCommand,
  CreateApplicationVersionCommand,
} from '@aws-sdk/client-serverlessapplicationrepository';

const client = new ServerlessApplicationRepositoryClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// Create an application
const app = await client.send(new CreateApplicationCommand({
  Name: 'my-serverless-app',
  Author: 'myteam',
  Description: 'A sample serverless application',
  SemanticVersion: '1.0.0',
  SpdxLicenseId: 'MIT',
}));
console.log(app.ApplicationId);

// Create a new version
await client.send(new CreateApplicationVersionCommand({
  ApplicationId: app.ApplicationId,
  SemanticVersion: '1.1.0',
  SourceCodeUrl: 'https://github.com/example/my-app',
}));
```

### Python

```python
import boto3

client = boto3.client('serverlessrepo',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

# Create an application
response = client.create_application(
    Name='my-serverless-app',
    Author='myteam',
    Description='A sample serverless application',
    SemanticVersion='1.0.0')
app_id = response['ApplicationId']

# Update the application
client.update_application(
    ApplicationId=app_id,
    Description='Updated description')

# Deploy via change set
client.create_cloud_formation_change_set(
    ApplicationId=app_id,
    SemanticVersion='1.0.0',
    StackName='my-app-stack')
```

## Configuration

```yaml
# cloudmock.yml
services:
  serverlessrepo:
    enabled: true
```

## Known Differences from AWS

- Semantic versions must match the X.Y.Z format (with optional pre-release and build metadata)
- CloudFormation change sets are created as stubs; no actual CloudFormation deployment occurs
- Template URLs are synthetic S3 URLs and are not accessible
- Application sharing and marketplace features are not emulated
