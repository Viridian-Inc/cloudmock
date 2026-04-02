---
title: Amazon Glacier
description: AWS S3 Glacier emulation in CloudMock
---

## Overview

CloudMock emulates Amazon S3 Glacier, supporting vault management, archive upload/deletion, job initiation and retrieval, vault lock (two-step InProgress → Locked), and tagging. Job lifecycle transitions from InProgress → Succeeded.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateVault | Supported | Creates a new vault |
| DescribeVault | Supported | Returns vault details and inventory info |
| ListVaults | Supported | Lists all vaults |
| DeleteVault | Supported | Fails if vault is non-empty |
| UploadArchive | Supported | Stores archive, returns ID in x-amz-archive-id header |
| DeleteArchive | Supported | Deletes archive by ID |
| InitiateJob | Supported | archive-retrieval or inventory-retrieval |
| DescribeJob | Supported | Returns job status and details |
| ListJobs | Supported | Lists jobs for a vault |
| GetJobOutput | Supported | Returns mock output for completed jobs |
| InitiateVaultLock | Supported | Creates InProgress lock, returns lockId in header |
| CompleteVaultLock | Supported | Transitions lock to Locked state |
| AbortVaultLock | Supported | Cancels InProgress vault lock |
| GetVaultLock | Supported | Returns lock policy and state |
| AddTagsToVault | Supported | Adds tags (POST /-/vaults/{name}/tags) |
| RemoveTagsFromVault | Supported | Removes tags by key |
| ListTagsForVault | Supported | Lists vault tags |

## REST API Routing

Glacier uses REST path-based routing, not JSON action names:

| Method | Path | Operation |
|--------|------|-----------|
| PUT | `/-/vaults/{name}` | CreateVault |
| GET | `/-/vaults/{name}` | DescribeVault |
| DELETE | `/-/vaults/{name}` | DeleteVault |
| GET | `/-/vaults` | ListVaults |
| POST | `/-/vaults/{name}/archives` | UploadArchive |
| DELETE | `/-/vaults/{name}/archives/{id}` | DeleteArchive |
| POST | `/-/vaults/{name}/jobs` | InitiateJob |
| GET | `/-/vaults/{name}/jobs/{id}` | DescribeJob |
| GET | `/-/vaults/{name}/jobs/{id}/output` | GetJobOutput |
| POST | `/-/vaults/{name}/lock-policy` | InitiateVaultLock |
| POST | `/-/vaults/{name}/lock-policy/{id}` | CompleteVaultLock |
| DELETE | `/-/vaults/{name}/lock-policy` | AbortVaultLock |
| GET | `/-/vaults/{name}/lock-policy` | GetVaultLock |
| GET | `/-/vaults/{name}/tags` | ListTagsForVault |
| POST | `/-/vaults/{name}/tags` | AddTagsToVault |

## Quick Start

```python
import boto3

client = boto3.client('glacier',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_vault(vaultName='my-archives')

# Upload archive
with open('backup.tar.gz', 'rb') as f:
    response = client.upload_archive(
        vaultName='my-archives',
        body=f,
        archiveDescription='Monthly backup',
    )
archive_id = response['archiveId']

# Initiate inventory job
job = client.initiate_job(
    vaultName='my-archives',
    jobParameters={'Type': 'inventory-retrieval'},
)
print(job['jobId'])
```

## Configuration

```yaml
services:
  glacier:
    enabled: true
```

## Known Differences from AWS

- Archives are stored in memory (not persisted)
- Job output is mock data, not real archive bytes
- Inventory dates and archive counts are simulated
