---
title: EventBridge Pipes
description: Amazon EventBridge Pipes emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EventBridge Pipes, supporting pipe creation, lifecycle management, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreatePipe | Supported | Creates a pipe |
| DescribePipe | Supported | Returns pipe details |
| ListPipes | Supported | Lists all pipes |
| UpdatePipe | Supported | Updates pipe configuration |
| DeletePipe | Supported | Deletes a pipe |
| StartPipe | Supported | Starts a stopped pipe |
| StopPipe | Supported | Stops a running pipe |
| TagResource | Supported | Adds tags to a pipe |
| UntagResource | Supported | Removes tags from a pipe |
| ListTagsForResource | Supported | Lists tags for a pipe |

## Quick Start

### Node.js

```typescript
import { PipesClient, CreatePipeCommand } from '@aws-sdk/client-pipes';

const client = new PipesClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreatePipeCommand({
  Name: 'my-pipe',
  Source: 'arn:aws:sqs:us-east-1:000000000000:my-queue',
  Target: 'arn:aws:lambda:us-east-1:000000000000:function:my-function',
  RoleArn: 'arn:aws:iam::000000000000:role/pipe-role',
}));
```

### Python

```python
import boto3

client = boto3.client('pipes',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_pipe(
    Name='my-pipe',
    Source='arn:aws:sqs:us-east-1:000000000000:my-queue',
    Target='arn:aws:lambda:us-east-1:000000000000:function:my-function',
    RoleArn='arn:aws:iam::000000000000:role/pipe-role')
```

## Configuration

```yaml
# cloudmock.yml
services:
  pipes:
    enabled: true
```

## Known Differences from AWS

- Pipes do not actually poll sources or invoke targets
- Enrichment and filtering are stored but not executed
- Pipe status transitions are simulated
