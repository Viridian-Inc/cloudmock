---
title: Cloud Map (Service Discovery)
description: AWS Cloud Map emulation in CloudMock
---

## Overview

CloudMock emulates AWS Cloud Map (Service Discovery), supporting namespace management, service registration, instance management, and instance discovery.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateHttpNamespace | Supported | Creates an HTTP namespace |
| CreatePrivateDnsNamespace | Supported | Creates a private DNS namespace |
| CreatePublicDnsNamespace | Supported | Creates a public DNS namespace |
| GetNamespace | Supported | Returns namespace details |
| ListNamespaces | Supported | Lists all namespaces |
| DeleteNamespace | Supported | Deletes a namespace |
| CreateService | Supported | Creates a service |
| GetService | Supported | Returns service details |
| ListServices | Supported | Lists services |
| UpdateService | Supported | Updates a service |
| DeleteService | Supported | Deletes a service |
| RegisterInstance | Supported | Registers a service instance |
| DeregisterInstance | Supported | Deregisters an instance |
| GetInstance | Supported | Returns instance details |
| ListInstances | Supported | Lists instances for a service |
| DiscoverInstances | Supported | Discovers healthy instances |
| UpdateInstanceCustomHealthStatus | Supported | Updates custom health status |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { ServiceDiscoveryClient, CreateHttpNamespaceCommand, CreateServiceCommand } from '@aws-sdk/client-servicediscovery';

const client = new ServiceDiscoveryClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { OperationId } = await client.send(new CreateHttpNamespaceCommand({
  Name: 'my-namespace',
}));

await client.send(new CreateServiceCommand({
  Name: 'my-service',
  NamespaceId: 'ns-12345',
}));
```

### Python

```python
import boto3

client = boto3.client('servicediscovery',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_http_namespace(Name='my-namespace')
client.create_service(Name='my-service', NamespaceId='ns-12345')
```

## Configuration

```yaml
# cloudmock.yml
services:
  servicediscovery:
    enabled: true
```

## Known Differences from AWS

- DNS namespaces do not create actual Route 53 hosted zones
- Instance discovery returns registered instances without health filtering
- Custom health status updates are stored but not used for filtering
