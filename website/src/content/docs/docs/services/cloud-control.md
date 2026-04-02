---
title: Cloud Control API
description: AWS Cloud Control API emulation in CloudMock
---

## Overview

CloudMock emulates the AWS Cloud Control API, providing a uniform CRUD interface for managing AWS resources. It supports routing to underlying service implementations for S3 buckets, DynamoDB tables, SQS queues, and SNS topics.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateResource | Supported | Creates a resource; routes to underlying service for supported types |
| GetResource | Supported | Returns resource details |
| ListResources | Supported | Lists resources of a given type |
| UpdateResource | Supported | Updates a resource |
| DeleteResource | Supported | Deletes a resource |
| GetResourceRequestStatus | Supported | Returns the status of an async request |
| ListResourceRequests | Supported | Lists all resource requests |

### Supported Resource Types

- `AWS::S3::Bucket`
- `AWS::DynamoDB::Table`
- `AWS::SQS::Queue`
- `AWS::SNS::Topic`

## Quick Start

### Node.js

```typescript
import { CloudControlClient, CreateResourceCommand } from '@aws-sdk/client-cloudcontrol';

const client = new CloudControlClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const result = await client.send(new CreateResourceCommand({
  TypeName: 'AWS::S3::Bucket',
  DesiredState: JSON.stringify({ BucketName: 'my-bucket' }),
}));
console.log(result.ProgressEvent);
```

### Python

```python
import boto3
import json

client = boto3.client('cloudcontrol',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_resource(
    TypeName='AWS::S3::Bucket',
    DesiredState=json.dumps({'BucketName': 'my-bucket'}))
print(response['ProgressEvent'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  cloudcontrol:
    enabled: true
```

## Known Differences from AWS

- Only a subset of resource types are supported (S3, DynamoDB, SQS, SNS)
- Unsupported resource types are stored generically without service-specific validation
- Resource type schemas are not enforced
