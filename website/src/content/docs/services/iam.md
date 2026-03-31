---
title: IAM
description: AWS IAM (Identity and Access Management) emulation in CloudMock
---

## Overview

CloudMock emulates AWS IAM as an embedded engine within the gateway, managing users, access keys, and policies with full policy evaluation when running in `enforce` mode.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateUser | Supported | Creates an IAM user (via seed file or admin API) |
| GetUser | Supported | Returns user details |
| CreateAccessKey | Supported | Creates an access key pair for a user |
| AttachUserPolicy | Supported | Attaches a policy to a user |
| GetUserPolicies | Supported | Returns policies attached to a user |

## Quick Start

### curl

```bash
# Check caller identity (uses root credentials by default)
curl -X POST "http://localhost:4566/?Action=GetCallerIdentity&Version=2011-06-15" \
  -H "Authorization: AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/sts/aws4_request"
```

### Node.js

```typescript
import { STSClient, GetCallerIdentityCommand } from '@aws-sdk/client-sts';

const sts = new STSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const identity = await sts.send(new GetCallerIdentityCommand({}));
console.log(identity.Arn); // arn:aws:iam::000000000000:root
```

### Python

```python
import boto3

sts = boto3.client('sts', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

identity = sts.get_caller_identity()
print(identity['Arn'])  # arn:aws:iam::000000000000:root
```

## Configuration

```yaml
# cloudmock.yml
iam:
  mode: enforce          # none | authenticate | enforce
  seed_file: ./iam-seed.json
  root_access_key: test
  root_secret_key: test
```

### IAM Modes

| Mode | Behavior |
|------|----------|
| `none` | Skip all authentication and authorization |
| `authenticate` | Verify credentials exist, skip policy evaluation |
| `enforce` | Full policy evaluation on every request |

### IAM Seed File

Bulk-load users, access keys, and policies at startup:

```json
{
  "users": [
    {
      "name": "ci",
      "access_key_id": "AKIAIOSFODNN7EXAMPLE",
      "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "policies": [
        {
          "name": "AllowAll",
          "document": {
            "Version": "2012-10-17",
            "Statement": [
              { "Effect": "Allow", "Action": "*", "Resource": "*" }
            ]
          }
        }
      ]
    }
  ]
}
```

## Known Differences from AWS

- IAM is **embedded in the gateway**, not exposed as a standalone HTTP service.
- The root user (`root_access_key` credential) bypasses all policy checks.
- **Roles, groups, and instance profiles** are not implemented.
- **Managed policies** (AWS-managed policy ARNs) are not available.
- **Policy conditions** (`Condition` block) are not evaluated.
- Wildcard matching supports `*` in Action and Resource fields.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AccessDenied | 403 | The request was denied by policy evaluation |
| InvalidClientTokenId | 403 | The access key ID does not exist |
| SignatureDoesNotMatch | 403 | The secret key does not match |
| IncompleteSignature | 400 | The request signature is incomplete |
