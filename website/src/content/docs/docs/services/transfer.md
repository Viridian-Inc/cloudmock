---
title: Transfer Family
description: AWS Transfer Family emulation in CloudMock
---

## Overview

CloudMock emulates AWS Transfer Family, supporting SFTP/FTPS/FTP server management, user management, and SSH public key operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateServer | Supported | Creates a Transfer server |
| DescribeServer | Supported | Returns server details |
| ListServers | Supported | Lists all servers |
| StartServer | Supported | Starts a server |
| StopServer | Supported | Stops a server |
| DeleteServer | Supported | Deletes a server |
| CreateUser | Supported | Creates a user on a server |
| DescribeUser | Supported | Returns user details |
| ListUsers | Supported | Lists users for a server |
| DeleteUser | Supported | Deletes a user |
| ImportSshPublicKey | Supported | Imports an SSH public key for a user |
| DeleteSshPublicKey | Supported | Deletes an SSH public key |

## Quick Start

### Node.js

```typescript
import { TransferClient, CreateServerCommand, CreateUserCommand } from '@aws-sdk/client-transfer';

const client = new TransferClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ServerId } = await client.send(new CreateServerCommand({
  Protocols: ['SFTP'],
  IdentityProviderType: 'SERVICE_MANAGED',
}));

await client.send(new CreateUserCommand({
  ServerId,
  UserName: 'my-user',
  Role: 'arn:aws:iam::000000000000:role/transfer-role',
  HomeDirectory: '/my-bucket',
}));
```

### Python

```python
import boto3

client = boto3.client('transfer',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_server(
    Protocols=['SFTP'],
    IdentityProviderType='SERVICE_MANAGED')
server_id = response['ServerId']

client.create_user(
    ServerId=server_id,
    UserName='my-user',
    Role='arn:aws:iam::000000000000:role/transfer-role',
    HomeDirectory='/my-bucket')
```

## Configuration

```yaml
# cloudmock.yml
services:
  transfer:
    enabled: true
```

## Known Differences from AWS

- No actual SFTP/FTPS/FTP server is provisioned
- Server endpoints are generated but not functional
- SSH key authentication is not performed
