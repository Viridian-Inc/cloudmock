---
title: Elasticsearch Service
description: Amazon Elasticsearch Service emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Elasticsearch Service (legacy), supporting domain management, configuration, tagging, and basic document indexing and search operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateElasticsearchDomain | Supported | Creates an Elasticsearch domain |
| DescribeElasticsearchDomain | Supported | Returns domain details |
| ListDomainNames | Supported | Lists all domain names |
| DeleteElasticsearchDomain | Supported | Deletes a domain |
| UpdateElasticsearchDomainConfig | Supported | Updates domain configuration |
| DescribeElasticsearchDomainConfig | Supported | Returns domain configuration |
| AddTags | Supported | Adds tags to a domain |
| RemoveTags | Supported | Removes tags from a domain |
| ListTags | Supported | Lists tags for a domain |
| IndexDocument | Supported | Indexes a document |
| Search | Supported | Performs a basic search |
| ClusterHealth | Supported | Returns cluster health status |

## Quick Start

### Node.js

```typescript
import { ElasticsearchServiceClient, CreateElasticsearchDomainCommand } from '@aws-sdk/client-elasticsearch-service';

const client = new ElasticsearchServiceClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateElasticsearchDomainCommand({
  DomainName: 'my-domain',
  ElasticsearchVersion: '7.10',
}));
```

### Python

```python
import boto3

client = boto3.client('es',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_elasticsearch_domain(
    DomainName='my-domain',
    ElasticsearchVersion='7.10')
```

## Configuration

```yaml
# cloudmock.yml
services:
  es:
    enabled: true
```

## Known Differences from AWS

- No actual Elasticsearch cluster is provisioned
- Search returns basic in-memory results, not full Elasticsearch query DSL
- Cluster health always reports green
- Consider using the OpenSearch service for new projects
