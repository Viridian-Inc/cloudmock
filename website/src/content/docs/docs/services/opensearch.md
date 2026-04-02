---
title: OpenSearch
description: Amazon OpenSearch Service emulation in CloudMock
---

## Overview

CloudMock emulates Amazon OpenSearch Service, supporting domain management, configuration, upgrades, version compatibility, VPC endpoints, tagging, and basic document indexing and search. Domain states transition `Processing -> Active`.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateDomain | Supported | Creates an OpenSearch or Elasticsearch domain |
| DescribeDomain | Supported | Returns domain details |
| DescribeDomains | Supported | Returns details for multiple domains by name |
| ListDomainNames | Supported | Lists all domain names |
| DeleteDomain | Supported | Deletes a domain |
| UpdateDomainConfig | Supported | Updates cluster config or EBS options |
| DescribeDomainConfig | Supported | Returns full domain configuration |
| GetCompatibleVersions | Supported | Returns compatible upgrade versions for a domain or all versions |
| CreateVpcEndpoint | Supported | Creates a VPC endpoint for a domain |
| DescribeVpcEndpoints | Supported | Returns details for VPC endpoints by ID |
| ListVpcEndpoints | Supported | Lists VPC endpoints, optionally filtered by domain ARN |
| DeleteVpcEndpoint | Supported | Deletes a VPC endpoint |
| AddTags | Supported | Adds tags to a domain by ARN |
| RemoveTags | Supported | Removes tags from a domain |
| ListTags | Supported | Lists tags for a domain |
| UpgradeDomain | Supported | Initiates a domain engine version upgrade |
| GetUpgradeStatus | Supported | Returns current upgrade status |
| IndexDocument | Supported | Indexes a document in an in-memory store |
| Search | Supported | Performs a basic match query search |
| ClusterHealth | Supported | Returns cluster health (green/yellow/red) |

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
