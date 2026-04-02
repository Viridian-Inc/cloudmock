---
title: Support
description: AWS Support emulation in CloudMock
---

## Overview

CloudMock emulates AWS Support, supporting case management, Trusted Advisor checks, service descriptions, and case communications.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCase | Supported | Creates a support case |
| DescribeCases | Supported | Lists support cases |
| ResolveCase | Supported | Resolves a support case |
| DescribeTrustedAdvisorChecks | Supported | Lists Trusted Advisor checks |
| DescribeTrustedAdvisorCheckResult | Supported | Returns check results |
| RefreshTrustedAdvisorCheck | Supported | Refreshes a check |
| DescribeServices | Supported | Lists available service categories |
| DescribeSeverityLevels | Supported | Lists severity levels |
| AddCommunicationToCase | Supported | Adds a communication to a case |
| DescribeCommunications | Supported | Lists communications for a case |

## Quick Start

### Node.js

```typescript
import { SupportClient, CreateCaseCommand } from '@aws-sdk/client-support';

const client = new SupportClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { caseId } = await client.send(new CreateCaseCommand({
  subject: 'Test issue',
  communicationBody: 'This is a test support case.',
  serviceCode: 'amazon-ec2',
  severityCode: 'low',
}));
console.log(caseId);
```

### Python

```python
import boto3

client = boto3.client('support',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_case(
    subject='Test issue',
    communicationBody='This is a test support case.',
    serviceCode='amazon-ec2',
    severityCode='low')
print(response['caseId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  support:
    enabled: true
```

## Known Differences from AWS

- Cases are not routed to actual support engineers
- Trusted Advisor checks return stub results
- Service descriptions and severity levels are predefined sets
