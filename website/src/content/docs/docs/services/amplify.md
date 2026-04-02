---
title: Amplify
description: AWS Amplify emulation in CloudMock
---

## Overview

CloudMock emulates AWS Amplify, supporting app, branch, domain association, webhook, and job management operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApp | Supported | Creates an Amplify app |
| GetApp | Supported | Returns app details |
| ListApps | Supported | Lists all apps |
| UpdateApp | Supported | Updates app configuration |
| DeleteApp | Supported | Deletes an app |
| CreateBranch | Supported | Creates a branch for an app |
| GetBranch | Supported | Returns branch details |
| ListBranches | Supported | Lists branches for an app |
| UpdateBranch | Supported | Updates branch configuration |
| DeleteBranch | Supported | Deletes a branch |
| CreateDomainAssociation | Supported | Creates a domain association |
| GetDomainAssociation | Supported | Returns domain association details |
| ListDomainAssociations | Supported | Lists domain associations |
| UpdateDomainAssociation | Supported | Updates a domain association |
| DeleteDomainAssociation | Supported | Deletes a domain association |
| CreateWebhook | Supported | Creates a webhook |
| GetWebhook | Supported | Returns webhook details |
| ListWebhooks | Supported | Lists webhooks |
| UpdateWebhook | Supported | Updates a webhook |
| DeleteWebhook | Supported | Deletes a webhook |
| StartJob | Supported | Starts a deployment job |
| GetJob | Supported | Returns job details |
| ListJobs | Supported | Lists jobs |
| StopJob | Supported | Stops a running job |
| CreateBackendEnvironment | Supported | Creates a backend environment for an app |
| GetBackendEnvironment | Supported | Returns backend environment details |
| ListBackendEnvironments | Supported | Lists backend environments for an app |
| DeleteBackendEnvironment | Supported | Deletes a backend environment |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { AmplifyClient, CreateAppCommand } from '@aws-sdk/client-amplify';

const client = new AmplifyClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { app } = await client.send(new CreateAppCommand({
  name: 'my-app',
  repository: 'https://github.com/example/repo',
}));
console.log(app.appId);
```

### Python

```python
import boto3

client = boto3.client('amplify',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_app(
    name='my-app',
    repository='https://github.com/example/repo')
print(response['app']['appId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  amplify:
    enabled: true
```

## Known Differences from AWS

- No actual build or deployment occurs when starting a job
- Domain associations are stored but DNS is not configured
- Webhooks are stored but do not trigger real builds
