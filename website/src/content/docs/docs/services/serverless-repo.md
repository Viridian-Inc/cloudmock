---
title: Serverless Application Repository
description: AWS Serverless Application Repository emulation in CloudMock
---

## Overview

CloudMock emulates the AWS Serverless Application Repository, supporting application management, versioning, and change set creation.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an application |
| GetApplication | Supported | Returns application details |
| ListApplications | Supported | Lists all applications |
| DeleteApplication | Supported | Deletes an application |
| CreateApplicationVersion | Supported | Creates a new version |
| ListApplicationVersions | Supported | Lists application versions |
| CreateChangeSet | Supported | Creates a CloudFormation change set for deployment |

## Quick Start

### Node.js

```typescript
import { ServerlessApplicationRepositoryClient, CreateApplicationCommand } from '@aws-sdk/client-serverlessapplicationrepository';

const client = new ServerlessApplicationRepositoryClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ApplicationId } = await client.send(new CreateApplicationCommand({
  Name: 'my-serverless-app',
  Author: 'test-author',
  Description: 'A test application',
  SemanticVersion: '1.0.0',
  TemplateBody: '{ "AWSTemplateFormatVersion": "2010-09-09" }',
}));
console.log(ApplicationId);
```

### Python

```python
import boto3

client = boto3.client('serverlessrepo',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_application(
    Name='my-serverless-app',
    Author='test-author',
    Description='A test application',
    SemanticVersion='1.0.0',
    TemplateBody='{ "AWSTemplateFormatVersion": "2010-09-09" }')
print(response['ApplicationId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  serverlessrepo:
    enabled: true
```

## Known Differences from AWS

- Change sets are created as stubs and do not deploy actual stacks
- Templates are stored but not validated
- Application sharing and policy management are not supported
