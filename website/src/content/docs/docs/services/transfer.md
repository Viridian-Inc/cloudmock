---
title: Transfer Family
description: AWS Transfer Family emulation in CloudMock
---

## Overview

CloudMock emulates AWS Transfer Family, supporting managed SFTP/FTPS/FTP/AS2 servers, users with SSH public key management, workflows, and tagging. Server lifecycle transitions through OFFLINE → STARTING → ONLINE → STOPPING.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateServer | Supported | Validates protocol types: SFTP, FTPS, FTP, AS2 |
| DescribeServer | Supported | Returns server details and endpoint URL |
| ListServers | Supported | Lists all servers |
| UpdateServer | Supported | Updates protocols and logging role |
| StartServer | Supported | Transitions OFFLINE → STARTING → ONLINE |
| StopServer | Supported | Transitions ONLINE → STOPPING → OFFLINE |
| DeleteServer | Supported | Deletes a server |
| CreateUser | Supported | Validates home directory starts with / |
| DescribeUser | Supported | Returns user details and SSH keys |
| ListUsers | Supported | Lists users for a server |
| UpdateUser | Supported | Updates home directory and IAM role |
| DeleteUser | Supported | Deletes a user |
| ImportSshPublicKey | Supported | Adds SSH public key to user |
| DeleteSshPublicKey | Supported | Removes SSH public key |
| CreateWorkflow | Supported | Creates a transfer workflow |
| DescribeWorkflow | Supported | Returns workflow steps |
| ListWorkflows | Supported | Lists all workflows |
| DeleteWorkflow | Supported | Deletes a workflow |
| TagResource | Supported | Adds tags by ARN |
| UntagResource | Supported | Removes tags by key |
| ListTagsForResource | Supported | Lists tags by ARN |

## Quick Start

```python
import boto3

client = boto3.client('transfer',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

# Create SFTP server
server = client.create_server(Protocols=['SFTP'])
server_id = server['ServerId']

# Create user
client.create_user(
    ServerId=server_id,
    UserName='alice',
    Role='arn:aws:iam::123456789012:role/transfer-role',
    HomeDirectory='/home/alice',
)

# Import SSH public key
client.import_ssh_public_key(
    ServerId=server_id,
    UserName='alice',
    SshPublicKeyBody='ssh-rsa AAAA...',
)
```

## Configuration

```yaml
services:
  transfer:
    enabled: true
```

## Known Differences from AWS

- No actual SFTP/FTP connections are accepted
- Server endpoints are generated deterministically from the server ID
- Workflow steps are stored but not executed on file uploads
