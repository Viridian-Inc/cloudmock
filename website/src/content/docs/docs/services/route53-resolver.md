---
title: Route 53 Resolver
description: Amazon Route 53 Resolver emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Route 53 Resolver, supporting resolver endpoints, resolver rules, rule associations, and query log configurations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateResolverEndpoint | Supported | Creates a resolver endpoint |
| GetResolverEndpoint | Supported | Returns endpoint details |
| ListResolverEndpoints | Supported | Lists all endpoints |
| DeleteResolverEndpoint | Supported | Deletes an endpoint |
| CreateResolverRule | Supported | Creates a resolver rule |
| GetResolverRule | Supported | Returns rule details |
| ListResolverRules | Supported | Lists all rules |
| DeleteResolverRule | Supported | Deletes a rule |
| AssociateResolverRule | Supported | Associates a rule with a VPC |
| GetResolverRuleAssociation | Supported | Returns association details |
| ListResolverRuleAssociations | Supported | Lists rule associations |
| DisassociateResolverRule | Supported | Disassociates a rule from a VPC |
| CreateQueryLogConfig | Supported | Creates a query log config |
| GetQueryLogConfig | Supported | Returns query log config details |
| ListQueryLogConfigs | Supported | Lists query log configs |
| DeleteQueryLogConfig | Supported | Deletes a query log config |

## Quick Start

### Node.js

```typescript
import { Route53ResolverClient, CreateResolverEndpointCommand } from '@aws-sdk/client-route53resolver';

const client = new Route53ResolverClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ResolverEndpoint } = await client.send(new CreateResolverEndpointCommand({
  CreatorRequestId: 'unique-id',
  Direction: 'INBOUND',
  SecurityGroupIds: ['sg-12345'],
  IpAddresses: [{ SubnetId: 'subnet-12345' }],
}));
console.log(ResolverEndpoint.Id);
```

### Python

```python
import boto3

client = boto3.client('route53resolver',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_resolver_endpoint(
    CreatorRequestId='unique-id',
    Direction='INBOUND',
    SecurityGroupIds=['sg-12345'],
    IpAddresses=[{'SubnetId': 'subnet-12345'}])
print(response['ResolverEndpoint']['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  route53resolver:
    enabled: true
```

## Known Differences from AWS

- Resolver endpoints do not perform actual DNS resolution
- Rule associations are stored but do not affect DNS queries
- Query logging does not capture real DNS queries
