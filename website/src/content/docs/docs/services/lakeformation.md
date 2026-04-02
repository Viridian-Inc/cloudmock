---
title: Lake Formation
description: AWS Lake Formation emulation in CloudMock
---

## Overview

CloudMock emulates AWS Lake Formation, supporting resource registration, permissions management, data lake settings, and LF-Tags for fine-grained access control.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RegisterResource | Supported | Registers a resource with Lake Formation |
| DeregisterResource | Supported | Deregisters a resource |
| ListResources | Supported | Lists registered resources |
| GrantPermissions | Supported | Grants permissions on a resource |
| RevokePermissions | Supported | Revokes permissions |
| GetEffectivePermissionsForPath | Supported | Returns effective permissions for a path |
| ListPermissions | Supported | Lists all permissions |
| GetDataLakeSettings | Supported | Returns data lake settings |
| PutDataLakeSettings | Supported | Updates data lake settings |
| CreateLFTag | Supported | Creates an LF-Tag |
| GetLFTag | Supported | Returns LF-Tag details |
| ListLFTags | Supported | Lists all LF-Tags |
| DeleteLFTag | Supported | Deletes an LF-Tag |
| UpdateLFTag | Supported | Updates an LF-Tag |
| AddLFTagsToResource | Supported | Assigns LF-Tags to a resource |
| RemoveLFTagsFromResource | Supported | Removes LF-Tags from a resource |
| GetResourceLFTags | Supported | Returns LF-Tags for a resource |

## Quick Start

### Node.js

```typescript
import { LakeFormationClient, RegisterResourceCommand } from '@aws-sdk/client-lakeformation';

const client = new LakeFormationClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new RegisterResourceCommand({
  ResourceArn: 'arn:aws:s3:::my-data-lake',
  RoleArn: 'arn:aws:iam::000000000000:role/lf-role',
}));
```

### Python

```python
import boto3

client = boto3.client('lakeformation',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.register_resource(
    ResourceArn='arn:aws:s3:::my-data-lake',
    RoleArn='arn:aws:iam::000000000000:role/lf-role')
```

## Configuration

```yaml
# cloudmock.yml
services:
  lakeformation:
    enabled: true
```

## Known Differences from AWS

- Permissions are stored but not enforced on actual data access
- LF-Tags are managed independently of Glue Data Catalog integration
- Data lake settings do not affect other service behavior
