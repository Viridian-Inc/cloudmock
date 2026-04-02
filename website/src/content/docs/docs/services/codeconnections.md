---
title: CodeConnections
description: AWS CodeConnections emulation in CloudMock
---

## Overview

CloudMock emulates AWS CodeConnections (formerly CodeStar Connections), supporting connection and host management with tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateConnection | Supported | Creates a connection |
| GetConnection | Supported | Returns connection details |
| ListConnections | Supported | Lists all connections |
| DeleteConnection | Supported | Deletes a connection |
| UpdateConnectionStatus | Supported | Updates connection status |
| CreateHost | Supported | Creates a host |
| GetHost | Supported | Returns host details |
| ListHosts | Supported | Lists all hosts |
| DeleteHost | Supported | Deletes a host |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { CodeConnectionsClient, CreateConnectionCommand } from '@aws-sdk/client-codeconnections';

const client = new CodeConnectionsClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ConnectionArn } = await client.send(new CreateConnectionCommand({
  ConnectionName: 'my-github-connection',
  ProviderType: 'GitHub',
}));
console.log(ConnectionArn);
```

### Python

```python
import boto3

client = boto3.client('codeconnections',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_connection(
    ConnectionName='my-github-connection',
    ProviderType='GitHub')
print(response['ConnectionArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  codeconnections:
    enabled: true
```

## Known Differences from AWS

- Connections are never actually authenticated with the provider
- Connection status must be manually updated; no OAuth flow is performed
- Hosts are stored but do not connect to real VPC infrastructure
