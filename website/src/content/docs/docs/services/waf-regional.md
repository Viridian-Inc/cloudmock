---
title: WAF Regional
description: AWS WAF Regional (v1) emulation in CloudMock
---

## Overview

CloudMock emulates AWS WAF Regional (v1), supporting Web ACLs, rules, IP sets, byte match sets, and resource associations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateWebACL | Supported | Creates a Web ACL |
| GetWebACL | Supported | Returns Web ACL details |
| ListWebACLs | Supported | Lists Web ACLs |
| UpdateWebACL | Supported | Updates a Web ACL |
| DeleteWebACL | Supported | Deletes a Web ACL |
| CreateRule | Supported | Creates a rule |
| GetRule | Supported | Returns rule details |
| ListRules | Supported | Lists rules |
| UpdateRule | Supported | Updates a rule |
| DeleteRule | Supported | Deletes a rule |
| CreateIPSet | Supported | Creates an IP set |
| GetIPSet | Supported | Returns IP set details |
| ListIPSets | Supported | Lists IP sets |
| UpdateIPSet | Supported | Updates an IP set |
| DeleteIPSet | Supported | Deletes an IP set |
| CreateByteMatchSet | Supported | Creates a byte match set |
| GetByteMatchSet | Supported | Returns byte match set details |
| ListByteMatchSets | Supported | Lists byte match sets |
| DeleteByteMatchSet | Supported | Deletes a byte match set |
| AssociateWebACL | Supported | Associates a Web ACL with a resource |
| DisassociateWebACL | Supported | Disassociates a Web ACL |
| GetWebACLForResource | Supported | Returns the Web ACL for a resource |

## Quick Start

### Node.js

```typescript
import { WAFRegionalClient, CreateWebACLCommand } from '@aws-sdk/client-waf-regional';

const client = new WAFRegionalClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { WebACL } = await client.send(new CreateWebACLCommand({
  Name: 'my-web-acl',
  MetricName: 'myWebAcl',
  DefaultAction: { Type: 'ALLOW' },
  ChangeToken: 'change-token',
}));
console.log(WebACL.WebACLId);
```

### Python

```python
import boto3

client = boto3.client('waf-regional',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_web_acl(
    Name='my-web-acl',
    MetricName='myWebAcl',
    DefaultAction={'Type': 'ALLOW'},
    ChangeToken='change-token')
print(response['WebACL']['WebACLId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  wafregional:
    enabled: true
```

## Known Differences from AWS

- Rules do not evaluate actual HTTP requests
- Change tokens are accepted but not validated
- Consider using WAFv2 for new projects
