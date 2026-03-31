---
title: ECR
description: Amazon ECR (Elastic Container Registry) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon ECR, supporting container image repository management, image manifest storage, authorization token generation, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateRepository | Supported | Creates a container image repository |
| DeleteRepository | Supported | Deletes a repository |
| DescribeRepositories | Supported | Returns repository metadata |
| ListImages | Supported | Returns image tags in a repository |
| BatchGetImage | Supported | Returns image manifests by tag or digest |
| PutImage | Supported | Stores an image manifest |
| BatchDeleteImage | Supported | Deletes images by tag or digest |
| GetAuthorizationToken | Supported | Returns a Docker login token |
| DescribeImageScanFindings | Supported | Returns a stub scan result |
| TagResource | Supported | Adds tags to a repository |
| UntagResource | Supported | Removes tags |
| ListTagsForResource | Supported | Returns tags for a repository |

## Quick Start

### curl

```bash
# Create a repository
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AmazonEC2ContainerRegistry_V20150921.CreateRepository" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"repositoryName": "my-app"}'

# Get authorization token
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AmazonEC2ContainerRegistry_V20150921.GetAuthorizationToken" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{}'
```

### Node.js

```typescript
import { ECRClient, CreateRepositoryCommand, GetAuthorizationTokenCommand } from '@aws-sdk/client-ecr';

const ecr = new ECRClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const repo = await ecr.send(new CreateRepositoryCommand({ repositoryName: 'backend' }));
console.log(repo.repository?.repositoryUri);
// 000000000000.dkr.ecr.us-east-1.localhost:4566/backend

const auth = await ecr.send(new GetAuthorizationTokenCommand({}));
console.log(auth.authorizationData?.[0]?.authorizationToken);
```

### Python

```python
import boto3

ecr = boto3.client('ecr', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

repo = ecr.create_repository(repositoryName='backend')
uri = repo['repository']['repositoryUri']
print(uri)  # 000000000000.dkr.ecr.us-east-1.localhost:4566/backend

token_resp = ecr.get_authorization_token()
token = token_resp['authorizationData'][0]['authorizationToken']

images = ecr.list_images(repositoryName='backend')
for img in images.get('imageIds', []):
    print(img)
```

## Configuration

```yaml
# cloudmock.yml
services:
  ecr:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- `GetAuthorizationToken` returns a **synthetic token**. Docker push/pull to the CloudMock ECR endpoint requires a container registry proxy and is not natively supported.
- **Image layers** are not stored. `PutImage` and `BatchGetImage` operate on manifest metadata only.
- Image vulnerability scanning results from `DescribeImageScanFindings` are **stubs**.
- **Lifecycle policies** are not implemented.
- **Cross-region and cross-account replication** are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| RepositoryNotFoundException | 400 | The specified repository does not exist |
| RepositoryAlreadyExistsException | 400 | A repository with this name already exists |
| ImageNotFoundException | 400 | The specified image does not exist |
| RepositoryNotEmptyException | 400 | The repository contains images and cannot be deleted |
