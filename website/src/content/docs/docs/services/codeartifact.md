---
title: CodeArtifact
description: AWS CodeArtifact emulation in CloudMock
---

## Overview

CloudMock emulates AWS CodeArtifact, supporting domain and repository management, package listing, authorization tokens, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDomain | Supported | Creates a domain |
| DescribeDomain | Supported | Returns domain details |
| ListDomains | Supported | Lists all domains |
| DeleteDomain | Supported | Deletes a domain |
| CreateRepository | Supported | Creates a repository |
| DescribeRepository | Supported | Returns repository details |
| ListRepositories | Supported | Lists all repositories |
| UpdateRepository | Supported | Updates repository description and upstreams |
| DeleteRepository | Supported | Deletes a repository |
| DescribePackage | Supported | Returns package details |
| ListPackages | Supported | Lists packages in a repository |
| ListPackageVersions | Supported | Lists versions of a package |
| DescribePackageVersion | Supported | Returns package version details |
| GetPackageVersionReadme | Supported | Returns a generated README for a package version |
| PutDomainPermissionsPolicy | Supported | Sets the resource policy for a domain |
| GetDomainPermissionsPolicy | Supported | Returns the resource policy for a domain |
| DeleteDomainPermissionsPolicy | Supported | Deletes the resource policy for a domain |
| GetRepositoryEndpoint | Supported | Returns the repository endpoint URL |
| GetAuthorizationToken | Supported | Returns a stub authorization token |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { CodeartifactClient, CreateDomainCommand, CreateRepositoryCommand } from '@aws-sdk/client-codeartifact';

const client = new CodeartifactClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateDomainCommand({ domain: 'my-domain' }));
await client.send(new CreateRepositoryCommand({
  domain: 'my-domain',
  repository: 'my-repo',
}));
```

### Python

```python
import boto3

client = boto3.client('codeartifact',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_domain(domain='my-domain')
client.create_repository(domain='my-domain', repository='my-repo')
```

## Configuration

```yaml
# cloudmock.yml
services:
  codeartifact:
    enabled: true
```

## Known Differences from AWS

- Authorization tokens are stubs and cannot be used with real package managers
- Package publishing is not supported; packages must be pre-seeded
- Repository endpoints are placeholders
