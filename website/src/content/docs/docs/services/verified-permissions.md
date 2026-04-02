---
title: Verified Permissions
description: Amazon Verified Permissions emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Verified Permissions, supporting policy stores, policies, schemas, authorization decisions, policy templates, and identity sources.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreatePolicyStore | Supported | Creates a policy store |
| GetPolicyStore | Supported | Returns policy store details |
| ListPolicyStores | Supported | Lists policy stores |
| UpdatePolicyStore | Supported | Updates a policy store |
| DeletePolicyStore | Supported | Deletes a policy store |
| CreatePolicy | Supported | Creates a policy |
| GetPolicy | Supported | Returns policy details |
| ListPolicies | Supported | Lists policies |
| UpdatePolicy | Supported | Updates a policy |
| DeletePolicy | Supported | Deletes a policy |
| PutSchema | Supported | Sets the schema for a policy store |
| GetSchema | Supported | Returns the schema |
| IsAuthorized | Supported | Makes an authorization decision |
| IsAuthorizedWithToken | Supported | Authorization with identity token |
| CreatePolicyTemplate | Supported | Creates a policy template |
| GetPolicyTemplate | Supported | Returns template details |
| ListPolicyTemplates | Supported | Lists policy templates |
| UpdatePolicyTemplate | Supported | Updates a policy template |
| DeletePolicyTemplate | Supported | Deletes a policy template |
| CreateIdentitySource | Supported | Creates an identity source |
| GetIdentitySource | Supported | Returns identity source details |
| ListIdentitySources | Supported | Lists identity sources |
| DeleteIdentitySource | Supported | Deletes an identity source |

## Quick Start

### Node.js

```typescript
import { VerifiedPermissionsClient, CreatePolicyStoreCommand, IsAuthorizedCommand } from '@aws-sdk/client-verifiedpermissions';

const client = new VerifiedPermissionsClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { policyStoreId } = await client.send(new CreatePolicyStoreCommand({
  validationSettings: { mode: 'OFF' },
}));

const authResult = await client.send(new IsAuthorizedCommand({
  policyStoreId,
  principal: { entityType: 'User', entityId: 'alice' },
  action: { actionType: 'Action', actionId: 'view' },
  resource: { entityType: 'Document', entityId: 'doc-123' },
}));
console.log(authResult.decision);
```

### Python

```python
import boto3

client = boto3.client('verifiedpermissions',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_policy_store(
    validationSettings={'mode': 'OFF'})
policy_store_id = response['policyStoreId']

auth_result = client.is_authorized(
    policyStoreId=policy_store_id,
    principal={'entityType': 'User', 'entityId': 'alice'},
    action={'actionType': 'Action', 'actionId': 'view'},
    resource={'entityType': 'Document', 'entityId': 'doc-123'})
print(auth_result['decision'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  verifiedpermissions:
    enabled: true
```

## Known Differences from AWS

- Cedar policy evaluation is simplified
- IsAuthorized returns stub decisions, not full Cedar engine evaluation
- Schema validation is not enforced on policies
- Identity source token verification is not performed
