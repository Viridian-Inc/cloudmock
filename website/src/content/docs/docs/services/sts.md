---
title: STS
description: AWS STS (Security Token Service) emulation in CloudMock
---

## Overview

CloudMock emulates AWS STS, enabling caller identity verification, role assumption with temporary credentials, and session token generation.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetCallerIdentity | Supported | Returns the account ID, ARN, and user ID of the caller |
| AssumeRole | Supported | Returns temporary credentials for the specified role ARN |
| GetSessionToken | Supported | Returns temporary credentials for the current user |

## Quick Start

### curl

```bash
# Get caller identity
curl -X POST "http://localhost:4566/?Action=GetCallerIdentity&Version=2011-06-15"

# Assume a role
curl -X POST "http://localhost:4566/?Action=AssumeRole&RoleArn=arn:aws:iam::000000000000:role/my-role&RoleSessionName=test&Version=2011-06-15"
```

### Node.js

```typescript
import { STSClient, GetCallerIdentityCommand, AssumeRoleCommand } from '@aws-sdk/client-sts';

const sts = new STSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const identity = await sts.send(new GetCallerIdentityCommand({}));
console.log(identity.Account); // 000000000000

const assumed = await sts.send(new AssumeRoleCommand({
  RoleArn: 'arn:aws:iam::000000000000:role/my-role',
  RoleSessionName: 'my-session',
}));
console.log(assumed.Credentials.AccessKeyId);
```

### Python

```python
import boto3

sts = boto3.client('sts', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

identity = sts.get_caller_identity()
print(identity['Arn'])  # arn:aws:iam::000000000000:root

response = sts.assume_role(
    RoleArn='arn:aws:iam::000000000000:role/my-role',
    RoleSessionName='test',
)
creds = response['Credentials']
print(creds['AccessKeyId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  sts:
    enabled: true
```

STS behavior is also controlled by the global IAM mode setting (`iam.mode`).

## Known Differences from AWS

- **AssumeRole** returns synthetic temporary credentials with a configurable expiration (default 1 hour). The returned session token is accepted by the IAM middleware for subsequent requests.
- **Cross-account** role assumption is accepted but no cross-account isolation exists.
- **Web identity** and **SAML** federation are not implemented.
- **MFA** is not required or validated for any STS operation.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AccessDenied | 403 | Not authorized to assume this role |
| MalformedPolicyDocument | 400 | The policy document is not valid |
| RegionDisabledException | 403 | STS is disabled in the specified region |
| ExpiredTokenException | 403 | The session token has expired |
