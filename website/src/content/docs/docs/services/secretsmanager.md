---
title: Secrets Manager
description: AWS Secrets Manager emulation in CloudMock
---

## Overview

CloudMock emulates AWS Secrets Manager, providing secret lifecycle management including creation, retrieval, versioning, deletion with restore capability, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateSecret | Supported | Creates a secret with string or binary value |
| GetSecretValue | Supported | Returns the current secret value |
| PutSecretValue | Supported | Adds a new version of the secret |
| UpdateSecret | Supported | Updates secret metadata (description, KMS key) |
| DeleteSecret | Supported | Marks the secret for deletion (immediate in emulator) |
| RestoreSecret | Supported | Cancels a pending deletion |
| DescribeSecret | Supported | Returns secret metadata without the value |
| ListSecrets | Supported | Returns all secrets |
| TagResource | Supported | Adds tags to a secret |
| UntagResource | Supported | Removes tags from a secret |

## Quick Start

### curl

```bash
# Create a secret
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: secretsmanager.CreateSecret" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"Name": "/app/db-password", "SecretString": "supersecret"}'

# Get secret value
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: secretsmanager.GetSecretValue" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"SecretId": "/app/db-password"}'
```

### Node.js

```typescript
import { SecretsManagerClient, CreateSecretCommand, GetSecretValueCommand } from '@aws-sdk/client-secrets-manager';

const sm = new SecretsManagerClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await sm.send(new CreateSecretCommand({
  Name: '/app/db-password', SecretString: 'supersecret',
}));

const { SecretString } = await sm.send(new GetSecretValueCommand({
  SecretId: '/app/db-password',
}));
console.log(SecretString); // supersecret
```

### Python

```python
import boto3, json

sm = boto3.client('secretsmanager', endpoint_url='http://localhost:4566',
                  aws_access_key_id='test', aws_secret_access_key='test',
                  region_name='us-east-1')

sm.create_secret(
    Name='/app/config',
    SecretString=json.dumps({'host': 'db.local', 'password': 's3cr3t'}),
)

response = sm.get_secret_value(SecretId='/app/config')
config = json.loads(response['SecretString'])
print(config['host'])  # db.local
```

## Configuration

```yaml
# cloudmock.yml
services:
  secretsmanager:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Secret versioning is tracked via version IDs but only the latest version is accessible without specifying a version ID.
- **Automatic rotation** is not implemented.
- **Binary secrets** (`SecretBinary`) are stored but returned as-is without base64 processing.
- **Resource policies** on secrets are not supported.
- **Replication** to other regions is not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified secret does not exist |
| ResourceExistsException | 400 | A secret with this name already exists |
| InvalidParameterException | 400 | An input parameter is invalid |
| InvalidRequestException | 400 | The request is not valid (e.g., deleting an already-deleted secret) |
