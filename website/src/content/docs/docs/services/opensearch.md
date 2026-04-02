---
title: OpenSearch
description: Amazon OpenSearch Service emulation in CloudMock
---

## Overview

CloudMock emulates Amazon OpenSearch Service, supporting domain management, configuration, upgrades, tagging, and basic document indexing and search.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDomain | Supported | Creates an OpenSearch domain |
| DescribeDomain | Supported | Returns domain details |
| ListDomainNames | Supported | Lists all domain names |
| DeleteDomain | Supported | Deletes a domain |
| UpdateDomainConfig | Supported | Updates domain configuration |
| DescribeDomainConfig | Supported | Returns domain configuration |
| AddTags | Supported | Adds tags to a domain |
| RemoveTags | Supported | Removes tags from a domain |
| ListTags | Supported | Lists tags for a domain |
| UpgradeDomain | Supported | Initiates a domain upgrade |
| GetUpgradeStatus | Supported | Returns upgrade status |
| IndexDocument | Supported | Indexes a document |
| Search | Supported | Performs a basic search |
| ClusterHealth | Supported | Returns cluster health status |

## Quick Start

### Node.js

```typescript
import { OpenSearchClient, CreateDomainCommand } from '@aws-sdk/client-opensearch';

const client = new OpenSearchClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateDomainCommand({
  DomainName: 'my-domain',
  EngineVersion: 'OpenSearch_2.11',
}));
```

### Python

```python
import boto3

client = boto3.client('opensearch',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_domain(
    DomainName='my-domain',
    EngineVersion='OpenSearch_2.11')
```

## Configuration

```yaml
# cloudmock.yml
services:
  opensearch:
    enabled: true
```

## Known Differences from AWS

- No actual OpenSearch cluster is provisioned
- Search returns basic in-memory results, not full OpenSearch query DSL
- Cluster health always reports green
- Domain upgrades are simulated status changes
