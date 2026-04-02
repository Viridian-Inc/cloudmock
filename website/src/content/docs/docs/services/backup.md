---
title: Backup
description: AWS Backup emulation in CloudMock
---

## Overview

CloudMock emulates AWS Backup, supporting backup plans, vaults, jobs, recovery points, vault locks, and backup selections.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateBackupPlan | Supported | Creates a backup plan |
| GetBackupPlan | Supported | Returns backup plan details |
| ListBackupPlans | Supported | Lists all backup plans |
| DeleteBackupPlan | Supported | Deletes a backup plan |
| CreateBackupVault | Supported | Creates a backup vault |
| DescribeBackupVault | Supported | Returns vault details |
| ListBackupVaults | Supported | Lists all vaults |
| DeleteBackupVault | Supported | Deletes a vault (must be empty) |
| StartBackupJob | Supported | Starts a backup job |
| DescribeBackupJob | Supported | Returns backup job details |
| ListBackupJobs | Supported | Lists all backup jobs |
| ListRecoveryPoints | Supported | Lists recovery points in a vault |
| DescribeRecoveryPoint | Supported | Returns recovery point details |
| PutBackupVaultLockConfiguration | Supported | Configures vault lock |
| CreateBackupSelection | Supported | Creates a backup selection |
| GetBackupSelection | Supported | Returns backup selection details |
| ListBackupSelections | Supported | Lists backup selections |
| DeleteBackupSelection | Supported | Deletes a backup selection |

## Quick Start

### Node.js

```typescript
import { BackupClient, CreateBackupVaultCommand, CreateBackupPlanCommand } from '@aws-sdk/client-backup';

const client = new BackupClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateBackupVaultCommand({
  BackupVaultName: 'my-vault',
}));

await client.send(new CreateBackupPlanCommand({
  BackupPlan: {
    BackupPlanName: 'my-plan',
    Rules: [{ RuleName: 'daily', TargetBackupVaultName: 'my-vault', ScheduleExpression: 'cron(0 12 * * ? *)' }],
  },
}));
```

### Python

```python
import boto3

client = boto3.client('backup',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_backup_vault(BackupVaultName='my-vault')

client.create_backup_plan(BackupPlan={
    'BackupPlanName': 'my-plan',
    'Rules': [{'RuleName': 'daily', 'TargetBackupVaultName': 'my-vault', 'ScheduleExpression': 'cron(0 12 * * ? *)'}],
})
```

## Configuration

```yaml
# cloudmock.yml
services:
  backup:
    enabled: true
```

## Known Differences from AWS

- Backup jobs do not actually back up any data
- Recovery points are stubs and cannot be used for restore
- Vault lock configuration is stored but not enforced over time
