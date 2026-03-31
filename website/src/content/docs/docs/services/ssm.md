---
title: SSM Parameter Store
description: AWS Systems Manager Parameter Store emulation in CloudMock
---

## Overview

CloudMock emulates AWS Systems Manager Parameter Store, providing hierarchical configuration storage with support for String, StringList, and SecureString parameter types.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| PutParameter | Supported | Creates or updates a parameter (String, StringList, SecureString) |
| GetParameter | Supported | Returns a single parameter by name |
| GetParameters | Supported | Returns multiple parameters by name list |
| GetParametersByPath | Supported | Returns all parameters under a path prefix |
| DeleteParameter | Supported | Deletes a single parameter |
| DeleteParameters | Supported | Deletes multiple parameters |
| DescribeParameters | Supported | Returns parameter metadata (no values) |

## Quick Start

### curl

```bash
# Put a parameter
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AmazonSSM.PutParameter" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"Name": "/app/env", "Value": "production", "Type": "String"}'

# Get a parameter
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AmazonSSM.GetParameter" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"Name": "/app/env"}'
```

### Node.js

```typescript
import { SSMClient, PutParameterCommand, GetParametersByPathCommand } from '@aws-sdk/client-ssm';

const ssm = new SSMClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await ssm.send(new PutParameterCommand({
  Name: '/service/db-host', Value: 'localhost', Type: 'String',
}));
await ssm.send(new PutParameterCommand({
  Name: '/service/db-pass', Value: 'secret', Type: 'SecureString',
}));

const { Parameters } = await ssm.send(new GetParametersByPathCommand({
  Path: '/service', WithDecryption: true,
}));
Parameters?.forEach(p => console.log(p.Name, '=', p.Value));
```

### Python

```python
import boto3

ssm = boto3.client('ssm', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

ssm.put_parameter(Name='/service/db-host', Value='localhost', Type='String')
ssm.put_parameter(Name='/service/db-pass', Value='secret', Type='SecureString')

response = ssm.get_parameters_by_path(Path='/service', WithDecryption=True)
for param in response['Parameters']:
    print(param['Name'], '=', param['Value'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ssm:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **SecureString** type is accepted and stored; encryption is simulated (not cryptographically secure).
- **GetParametersByPath** supports recursive path lookups.
- Parameter **versioning** stores each `PutParameter` call as a new version; the latest version is returned by default.
- **GetParameterHistory** is not implemented.
- **Advanced parameters** (larger size, policies) are not distinguished from standard parameters.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ParameterNotFound | 400 | The specified parameter does not exist |
| ParameterAlreadyExists | 400 | The parameter already exists (when Overwrite is false) |
| InvalidKeyId | 400 | The KMS key ID is not valid |
| TooManyUpdates | 429 | Too many updates for this parameter |
