---
title: Organizations
description: AWS Organizations emulation in CloudMock
---

## Overview

CloudMock emulates AWS Organizations, supporting organization management, organizational units, accounts, policies (including SCPs), and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateOrganization | Supported | Creates an organization |
| DescribeOrganization | Supported | Returns organization details |
| DeleteOrganization | Supported | Deletes the organization |
| ListRoots | Supported | Lists organization roots |
| CreateOrganizationalUnit | Supported | Creates an OU |
| DescribeOrganizationalUnit | Supported | Returns OU details |
| ListOrganizationalUnitsForParent | Supported | Lists OUs under a parent |
| DeleteOrganizationalUnit | Supported | Deletes an OU |
| CreateAccount | Supported | Creates a member account |
| DescribeCreateAccountStatus | Supported | Returns account creation status |
| DescribeAccount | Supported | Returns account details |
| ListAccounts | Supported | Lists all accounts |
| ListAccountsForParent | Supported | Lists accounts under a parent |
| MoveAccount | Supported | Moves an account between OUs |
| CreatePolicy | Supported | Creates a policy (e.g., SCP) |
| DescribePolicy | Supported | Returns policy details |
| ListPolicies | Supported | Lists policies |
| UpdatePolicy | Supported | Updates a policy |
| DeletePolicy | Supported | Deletes a policy |
| AttachPolicy | Supported | Attaches a policy to a target |
| DetachPolicy | Supported | Detaches a policy |
| ListTargetsForPolicy | Supported | Lists targets for a policy |
| EnablePolicyType | Supported | Enables a policy type on a root |
| DisablePolicyType | Supported | Disables a policy type |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { OrganizationsClient, CreateOrganizationCommand, CreateAccountCommand } from '@aws-sdk/client-organizations';

const client = new OrganizationsClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateOrganizationCommand({ FeatureSet: 'ALL' }));

const { CreateAccountStatus } = await client.send(new CreateAccountCommand({
  AccountName: 'dev-account',
  Email: 'dev@example.com',
}));
console.log(CreateAccountStatus.AccountId);
```

### Python

```python
import boto3

client = boto3.client('organizations',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_organization(FeatureSet='ALL')

response = client.create_account(
    AccountName='dev-account',
    Email='dev@example.com')
print(response['CreateAccountStatus']['AccountId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  organizations:
    enabled: true
```

## Known Differences from AWS

- Member accounts are stubs and do not have real AWS resources
- SCPs are stored and attached but not enforced on API calls
- Account creation completes immediately
