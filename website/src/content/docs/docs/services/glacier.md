---
title: Glacier
description: Amazon S3 Glacier emulation in CloudMock
---

## Overview

CloudMock emulates Amazon S3 Glacier, supporting vault management, archive operations, retrieval jobs, vault locks, and notifications.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateVault | Supported | Creates a vault |
| DescribeVault | Supported | Returns vault details |
| ListVaults | Supported | Lists all vaults |
| DeleteVault | Supported | Deletes an empty vault |
| UploadArchive | Supported | Uploads an archive to a vault |
| DeleteArchive | Supported | Deletes an archive |
| InitiateJob | Supported | Initiates a retrieval or inventory job |
| DescribeJob | Supported | Returns job details |
| ListJobs | Supported | Lists jobs for a vault |
| InitiateVaultLock | Supported | Initiates a vault lock |
| CompleteVaultLock | Supported | Completes a vault lock |
| SetVaultNotifications | Supported | Configures vault notifications |
| GetVaultNotifications | Supported | Returns vault notification config |

## Quick Start

### Node.js

```typescript
import { GlacierClient, CreateVaultCommand, UploadArchiveCommand } from '@aws-sdk/client-glacier';

const client = new GlacierClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateVaultCommand({ vaultName: 'my-vault', accountId: '-' }));

const { archiveId } = await client.send(new UploadArchiveCommand({
  vaultName: 'my-vault',
  accountId: '-',
  body: Buffer.from('archive data'),
}));
console.log(archiveId);
```

### Python

```python
import boto3

client = boto3.client('glacier',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_vault(vaultName='my-vault', accountId='-')

response = client.upload_archive(
    vaultName='my-vault',
    accountId='-',
    body=b'archive data')
print(response['archiveId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  glacier:
    enabled: true
```

## Known Differences from AWS

- Archives are stored in-memory, not in cold storage
- Retrieval jobs complete immediately rather than taking hours
- Vault lock policy enforcement is simplified
