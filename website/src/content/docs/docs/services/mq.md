---
title: Amazon MQ
description: Amazon MQ emulation in CloudMock
---

## Overview

CloudMock emulates Amazon MQ, supporting broker management, configurations, users, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateBroker | Supported | Creates a message broker |
| DescribeBroker | Supported | Returns broker details |
| ListBrokers | Supported | Lists all brokers |
| DeleteBroker | Supported | Deletes a broker |
| UpdateBroker | Supported | Updates broker configuration |
| RebootBroker | Supported | Reboots a broker |
| CreateConfiguration | Supported | Creates a configuration |
| DescribeConfiguration | Supported | Returns configuration details |
| ListConfigurations | Supported | Lists configurations |
| UpdateConfiguration | Supported | Updates a configuration |
| CreateUser | Supported | Creates a broker user |
| DescribeUser | Supported | Returns user details |
| ListUsers | Supported | Lists users for a broker |
| UpdateUser | Supported | Updates a user |
| DeleteUser | Supported | Deletes a user |
| CreateTags | Supported | Adds tags to a broker |
| DeleteTags | Supported | Removes tags from a broker |
| ListTags | Supported | Lists tags for a broker |

## Quick Start

### Node.js

```typescript
import { MqClient, CreateBrokerCommand } from '@aws-sdk/client-mq';

const client = new MqClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { BrokerId } = await client.send(new CreateBrokerCommand({
  BrokerName: 'my-broker',
  EngineType: 'ACTIVEMQ',
  EngineVersion: '5.17.6',
  HostInstanceType: 'mq.m5.large',
  DeploymentMode: 'SINGLE_INSTANCE',
  Users: [{ Username: 'admin', Password: 'Password123' }],
}));
console.log(BrokerId);
```

### Python

```python
import boto3

client = boto3.client('mq',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_broker(
    BrokerName='my-broker',
    EngineType='ACTIVEMQ',
    EngineVersion='5.17.6',
    HostInstanceType='mq.m5.large',
    DeploymentMode='SINGLE_INSTANCE',
    Users=[{'Username': 'admin', 'Password': 'Password123'}])
print(response['BrokerId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  mq:
    enabled: true
```

## Known Differences from AWS

- No actual ActiveMQ or RabbitMQ engine is provisioned
- Broker endpoints are generated but not functional for messaging
- Reboots update status but do not affect any running process
