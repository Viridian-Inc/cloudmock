---
title: Account
description: AWS Account Management emulation in CloudMock
---

## Overview

CloudMock emulates AWS Account Management, supporting contact information management, alternate contacts, and region opt-in controls.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetContactInformation | Supported | Returns account contact information |
| PutContactInformation | Supported | Updates account contact information |
| GetAlternateContact | Supported | Returns an alternate contact |
| PutAlternateContact | Supported | Creates or updates an alternate contact |
| DeleteAlternateContact | Supported | Removes an alternate contact |
| GetRegionOptStatus | Supported | Returns region opt-in status |
| ListRegions | Supported | Lists available regions and their status |
| EnableRegion | Supported | Enables an opt-in region |
| DisableRegion | Supported | Disables an opt-in region |

## Quick Start

### Node.js

```typescript
import { AccountClient, GetContactInformationCommand } from '@aws-sdk/client-account';

const client = new AccountClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const result = await client.send(new GetContactInformationCommand({}));
console.log(result.ContactInformation);
```

### Python

```python
import boto3

client = boto3.client('account',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.get_contact_information()
print(response['ContactInformation'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  account:
    enabled: true
```

## Known Differences from AWS

- Region opt-in status changes are stored in-memory only
- No validation of contact information fields beyond basic presence checks
