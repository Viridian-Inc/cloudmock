---
title: Managed Blockchain
description: Amazon Managed Blockchain emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Managed Blockchain, supporting network creation, member and node management, and proposal voting.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateNetwork | Supported | Creates a blockchain network |
| GetNetwork | Supported | Returns network details |
| ListNetworks | Supported | Lists all networks |
| GetMember | Supported | Returns member details |
| ListMembers | Supported | Lists members of a network |
| CreateNode | Supported | Creates a peer node |
| GetNode | Supported | Returns node details |
| ListNodes | Supported | Lists nodes in a network |
| DeleteNode | Supported | Deletes a node |
| CreateProposal | Supported | Creates a governance proposal |
| GetProposal | Supported | Returns proposal details |
| VoteOnProposal | Supported | Casts a vote on a proposal |
| ListProposals | Supported | Lists proposals for a network |

## Quick Start

### Node.js

```typescript
import { ManagedBlockchainClient, CreateNetworkCommand } from '@aws-sdk/client-managedblockchain';

const client = new ManagedBlockchainClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { NetworkId, MemberId } = await client.send(new CreateNetworkCommand({
  Name: 'my-network',
  Framework: 'HYPERLEDGER_FABRIC',
  FrameworkVersion: '2.2',
  VotingPolicy: { ApprovalThresholdPolicy: { ThresholdPercentage: 50, ProposalDurationInHours: 24, ThresholdComparator: 'GREATER_THAN' } },
  MemberConfiguration: { Name: 'my-member', FrameworkConfiguration: { Fabric: { AdminUsername: 'admin', AdminPassword: 'Password123' } } },
}));
console.log(NetworkId);
```

### Python

```python
import boto3

client = boto3.client('managedblockchain',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_network(
    Name='my-network',
    Framework='HYPERLEDGER_FABRIC',
    FrameworkVersion='2.2',
    VotingPolicy={'ApprovalThresholdPolicy': {'ThresholdPercentage': 50, 'ProposalDurationInHours': 24, 'ThresholdComparator': 'GREATER_THAN'}},
    MemberConfiguration={'Name': 'my-member', 'FrameworkConfiguration': {'Fabric': {'AdminUsername': 'admin', 'AdminPassword': 'Password123'}}})
print(response['NetworkId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  managedblockchain:
    enabled: true
```

## Known Differences from AWS

- No actual blockchain network is provisioned
- Nodes do not run Hyperledger Fabric or Ethereum
- Proposal voting is tracked but does not trigger real governance actions
