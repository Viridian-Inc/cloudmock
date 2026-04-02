---
title: SSO Admin
description: AWS IAM Identity Center (SSO) Admin emulation in CloudMock
---

## Overview

CloudMock emulates AWS IAM Identity Center (SSO) Admin, supporting instances, permission sets, account assignments, managed policies, inline policies, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| ListInstances | Supported | Lists SSO instances |
| DescribeInstance | Supported | Returns instance details |
| CreatePermissionSet | Supported | Creates a permission set |
| DescribePermissionSet | Supported | Returns permission set details |
| ListPermissionSets | Supported | Lists permission sets |
| UpdatePermissionSet | Supported | Updates a permission set |
| DeletePermissionSet | Supported | Deletes a permission set |
| CreateAccountAssignment | Supported | Creates an account assignment |
| ListAccountAssignments | Supported | Lists account assignments |
| DeleteAccountAssignment | Supported | Deletes an account assignment |
| AttachManagedPolicyToPermissionSet | Supported | Attaches a managed policy |
| DetachManagedPolicyFromPermissionSet | Supported | Detaches a managed policy |
| ListManagedPoliciesInPermissionSet | Supported | Lists managed policies |
| PutInlinePolicyToPermissionSet | Supported | Sets an inline policy |
| GetInlinePolicyForPermissionSet | Supported | Returns the inline policy |
| DeleteInlinePolicyFromPermissionSet | Supported | Removes the inline policy |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { SSOAdminClient, CreatePermissionSetCommand } from '@aws-sdk/client-sso-admin';

const client = new SSOAdminClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { PermissionSet } = await client.send(new CreatePermissionSetCommand({
  InstanceArn: 'arn:aws:sso:::instance/ssoins-1234567890',
  Name: 'AdminAccess',
  SessionDuration: 'PT8H',
}));
console.log(PermissionSet.PermissionSetArn);
```

### Python

```python
import boto3

client = boto3.client('sso-admin',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_permission_set(
    InstanceArn='arn:aws:sso:::instance/ssoins-1234567890',
    Name='AdminAccess',
    SessionDuration='PT8H')
print(response['PermissionSet']['PermissionSetArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ssoadmin:
    enabled: true
```

## Known Differences from AWS

- SSO instances are stubs and do not provide actual SSO functionality
- Account assignments are stored but do not grant real access
- Permission sets are not provisioned to accounts
