---
title: Identity Store
description: AWS Identity Store emulation in CloudMock
---

## Overview

CloudMock emulates AWS Identity Store, supporting user, group, and group membership management for IAM Identity Center.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateUser | Supported | Creates a user |
| DescribeUser | Supported | Returns user details |
| ListUsers | Supported | Lists all users |
| DeleteUser | Supported | Deletes a user |
| CreateGroup | Supported | Creates a group |
| DescribeGroup | Supported | Returns group details |
| ListGroups | Supported | Lists all groups |
| DeleteGroup | Supported | Deletes a group |
| CreateGroupMembership | Supported | Adds a user to a group |
| GetGroupMembershipId | Supported | Returns a membership ID |
| ListGroupMemberships | Supported | Lists group memberships |
| DeleteGroupMembership | Supported | Removes a user from a group |

## Quick Start

### Node.js

```typescript
import { IdentitystoreClient, CreateUserCommand } from '@aws-sdk/client-identitystore';

const client = new IdentitystoreClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { UserId } = await client.send(new CreateUserCommand({
  IdentityStoreId: 'd-1234567890',
  UserName: 'jdoe',
  Name: { GivenName: 'John', FamilyName: 'Doe' },
  DisplayName: 'John Doe',
}));
console.log(UserId);
```

### Python

```python
import boto3

client = boto3.client('identitystore',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_user(
    IdentityStoreId='d-1234567890',
    UserName='jdoe',
    Name={'GivenName': 'John', 'FamilyName': 'Doe'},
    DisplayName='John Doe')
print(response['UserId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  identitystore:
    enabled: true
```

## Known Differences from AWS

- Not connected to a real IAM Identity Center instance
- User authentication is not supported
- Identity store IDs are accepted but not validated
