---
title: App Runner
description: AWS App Runner emulation in CloudMock
---

## Overview

CloudMock emulates AWS App Runner, supporting service creation and lifecycle management (pause/resume), GitHub/Bitbucket connections, auto scaling configurations with revision tracking, VPC connectors, and resource tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateService | Supported | Starts in RUNNING state; supports ECR image source |
| DescribeService | Supported | Lookup by ARN |
| ListServices | Supported | Returns all services |
| UpdateService | Supported | Updates source or instance configuration |
| DeleteService | Supported | Removes service from store |
| PauseService | Supported | Sets status to PAUSED |
| ResumeService | Supported | Sets status back to RUNNING |
| CreateConnection | Supported | GitHub and Bitbucket provider types |
| DescribeConnection | Supported | Lookup by ARN |
| CreateAutoScalingConfiguration | Supported | Multi-revision; latest tracking |
| DescribeAutoScalingConfiguration | Supported | Lookup by ARN |
| ListAutoScalingConfigurations | Supported | Filter by name |
| DeleteAutoScalingConfiguration | Supported | Marks as INACTIVE |
| CreateVpcConnector | Supported | Multiple subnets and security groups |
| DescribeVpcConnector | Supported | Lookup by ARN |
| ListVpcConnectors | Supported | Returns all active connectors |
| DeleteVpcConnector | Supported | Marks as INACTIVE |
| TagResource | Supported | Tag any App Runner resource by ARN |
| UntagResource | Supported | |
| ListTagsForResource | Supported | |

## Service State Machine

```
CREATE_IN_PROGRESS → RUNNING → PAUSED
                              ↑
                           ResumeService
```

## Quick Start

### Node.js

```typescript
import {
  AppRunnerClient,
  CreateServiceCommand,
  PauseServiceCommand,
  ResumeServiceCommand,
  CreateAutoScalingConfigurationCommand,
} from '@aws-sdk/client-apprunner';

const client = new AppRunnerClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const asc = await client.send(new CreateAutoScalingConfigurationCommand({
  AutoScalingConfigurationName: 'my-asc',
  MinSize: 1,
  MaxSize: 10,
  MaxConcurrency: 100,
}));

const { Service } = await client.send(new CreateServiceCommand({
  ServiceName: 'my-app',
  SourceConfiguration: {
    ImageRepository: {
      ImageIdentifier: 'public.ecr.aws/nginx/nginx:latest',
      ImageRepositoryType: 'ECR_PUBLIC',
    },
    AutoDeploymentsEnabled: false,
  },
  InstanceConfiguration: {
    Cpu: '1 vCPU',
    Memory: '2 GB',
  },
}));

console.log('Service URL:', Service.ServiceUrl);
console.log('Service ARN:', Service.ServiceArn);

// Pause
await client.send(new PauseServiceCommand({ ServiceArn: Service.ServiceArn }));

// Resume
await client.send(new ResumeServiceCommand({ ServiceArn: Service.ServiceArn }));
```

### Python

```python
import boto3

client = boto3.client('apprunner',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_service(
    ServiceName='my-app',
    SourceConfiguration={
        'ImageRepository': {
            'ImageIdentifier': 'public.ecr.aws/nginx/nginx:latest',
            'ImageRepositoryType': 'ECR_PUBLIC',
        },
    },
    InstanceConfiguration={'Cpu': '1 vCPU', 'Memory': '2 GB'},
    Tags=[{'Key': 'env', 'Value': 'production'}])

print(response['Service']['ServiceUrl'])

# Create a VPC connector
vc = client.create_vpc_connector(
    VpcConnectorName='my-vpc-connector',
    Subnets=['subnet-1', 'subnet-2'],
    SecurityGroups=['sg-1'])
print(vc['VpcConnector']['VpcConnectorArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  apprunner:
    enabled: true
```

## Known Differences from AWS

- Services start immediately in RUNNING state (no actual container image pull)
- Connections are marked AVAILABLE immediately (no OAuth handshake required)
- AutoScaling configurations do not enforce concurrency limits on running services
- VPC connectors are created but do not affect actual network routing
- Service URLs are generated but do not serve actual traffic
