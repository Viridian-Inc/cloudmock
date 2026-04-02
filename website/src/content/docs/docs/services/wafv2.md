---
title: WAFv2
description: AWS WAFv2 emulation in CloudMock
---

## Overview

CloudMock emulates AWS WAFv2, supporting Web ACLs, rule groups, IP sets, regex pattern sets, resource associations, logging, request sampling, and tagging. Includes a basic rule evaluation engine.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateWebACL | Supported | Creates a Web ACL |
| GetWebACL | Supported | Returns Web ACL details |
| ListWebACLs | Supported | Lists Web ACLs |
| UpdateWebACL | Supported | Updates a Web ACL |
| DeleteWebACL | Supported | Deletes a Web ACL |
| CreateRuleGroup | Supported | Creates a rule group |
| GetRuleGroup | Supported | Returns rule group details |
| ListRuleGroups | Supported | Lists rule groups |
| UpdateRuleGroup | Supported | Updates a rule group |
| DeleteRuleGroup | Supported | Deletes a rule group |
| CreateIPSet | Supported | Creates an IP set |
| GetIPSet | Supported | Returns IP set details |
| ListIPSets | Supported | Lists IP sets |
| UpdateIPSet | Supported | Updates an IP set |
| DeleteIPSet | Supported | Deletes an IP set |
| CreateRegexPatternSet | Supported | Creates a regex pattern set |
| GetRegexPatternSet | Supported | Returns regex pattern set details |
| ListRegexPatternSets | Supported | Lists regex pattern sets |
| DeleteRegexPatternSet | Supported | Deletes a regex pattern set |
| GetSampledRequests | Supported | Returns sampled requests |
| AssociateWebACL | Supported | Associates a Web ACL with a resource |
| DisassociateWebACL | Supported | Disassociates a Web ACL |
| GetWebACLForResource | Supported | Returns the Web ACL for a resource |
| PutLoggingConfiguration | Supported | Configures logging for a Web ACL |
| GetLoggingConfiguration | Supported | Returns logging configuration |
| DeleteLoggingConfiguration | Supported | Removes logging configuration |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { WAFV2Client, CreateWebACLCommand } from '@aws-sdk/client-wafv2';

const client = new WAFV2Client({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { Summary } = await client.send(new CreateWebACLCommand({
  Name: 'my-web-acl',
  Scope: 'REGIONAL',
  DefaultAction: { Allow: {} },
  VisibilityConfig: { SampledRequestsEnabled: true, CloudWatchMetricsEnabled: true, MetricName: 'myWebAcl' },
  Rules: [],
}));
console.log(Summary.Id);
```

### Python

```python
import boto3

client = boto3.client('wafv2',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_web_acl(
    Name='my-web-acl',
    Scope='REGIONAL',
    DefaultAction={'Allow': {}},
    VisibilityConfig={'SampledRequestsEnabled': True, 'CloudWatchMetricsEnabled': True, 'MetricName': 'myWebAcl'},
    Rules=[])
print(response['Summary']['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  wafv2:
    enabled: true
```

## Known Differences from AWS

- Rule evaluation is basic and may not match all AWS WAF rule types
- Managed rule groups are not available
- Sampled requests return stub data
- Logging configuration is stored but logs are not delivered
