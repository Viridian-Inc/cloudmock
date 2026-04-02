---
title: CodeCommit
description: AWS CodeCommit emulation in CloudMock
---

## Overview

CloudMock emulates AWS CodeCommit, supporting repository management, branches, pull requests, commits, and diffs.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateRepository | Supported | Creates a repository |
| GetRepository | Supported | Returns repository details |
| ListRepositories | Supported | Lists all repositories |
| DeleteRepository | Supported | Deletes a repository |
| UpdateRepositoryName | Supported | Renames a repository |
| UpdateRepositoryDescription | Supported | Updates repository description |
| CreateBranch | Supported | Creates a branch |
| GetBranch | Supported | Returns branch details |
| ListBranches | Supported | Lists branches in a repository |
| DeleteBranch | Supported | Deletes a branch |
| CreatePullRequest | Supported | Creates a pull request |
| GetPullRequest | Supported | Returns pull request details |
| ListPullRequests | Supported | Lists pull requests |
| UpdatePullRequestStatus | Supported | Updates pull request status |
| MergePullRequestBySquash | Supported | Merges a pull request via squash |
| GetCommit | Supported | Returns commit details |
| GetDifferences | Supported | Returns differences between commits |

## Quick Start

### Node.js

```typescript
import { CodeCommitClient, CreateRepositoryCommand } from '@aws-sdk/client-codecommit';

const client = new CodeCommitClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { repositoryMetadata } = await client.send(new CreateRepositoryCommand({
  repositoryName: 'my-repo',
  repositoryDescription: 'My test repository',
}));
console.log(repositoryMetadata.repositoryId);
```

### Python

```python
import boto3

client = boto3.client('codecommit',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_repository(
    repositoryName='my-repo',
    repositoryDescription='My test repository')
print(response['repositoryMetadata']['repositoryId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  codecommit:
    enabled: true
```

## Known Differences from AWS

- No actual Git repository is created; data is stored in-memory
- Commits and diffs are stubs
- Merge operations update status but do not perform real merges
