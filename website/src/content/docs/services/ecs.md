---
title: ECS
description: Amazon ECS (Elastic Container Service) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon ECS, supporting cluster, task definition, service, and task lifecycle management. Tasks and services are metadata records -- no container runtime is involved.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCluster | Supported | Creates an ECS cluster |
| DeleteCluster | Supported | Deletes a cluster |
| DescribeClusters | Supported | Returns cluster details |
| ListClusters | Supported | Returns all cluster ARNs |
| RegisterTaskDefinition | Supported | Registers a task definition revision |
| DeregisterTaskDefinition | Supported | Marks a task definition as INACTIVE |
| DescribeTaskDefinition | Supported | Returns a task definition |
| ListTaskDefinitions | Supported | Returns all task definition ARNs |
| CreateService | Supported | Creates a long-running service |
| DeleteService | Supported | Deletes a service |
| DescribeServices | Supported | Returns service details |
| ListServices | Supported | Returns all service ARNs in a cluster |
| UpdateService | Supported | Updates desired count, task definition, etc. |
| RunTask | Supported | Starts one or more task instances |
| StopTask | Supported | Stops a running task |
| DescribeTasks | Supported | Returns task details |
| ListTasks | Supported | Returns task ARNs in a cluster or service |
| TagResource | Supported | Adds tags |
| UntagResource | Supported | Removes tags |
| ListTagsForResource | Supported | Returns tags for a resource |

## Quick Start

### curl

```bash
# Create a cluster
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"clusterName": "production"}'

# Register a task definition
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"family": "web", "containerDefinitions": [{"name": "nginx", "image": "nginx:latest"}]}'
```

### Node.js

```typescript
import { ECSClient, CreateClusterCommand, RegisterTaskDefinitionCommand, CreateServiceCommand } from '@aws-sdk/client-ecs';

const ecs = new ECSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await ecs.send(new CreateClusterCommand({ clusterName: 'dev' }));
await ecs.send(new RegisterTaskDefinitionCommand({
  family: 'worker',
  containerDefinitions: [{ name: 'app', image: 'myapp:latest', cpu: 256, memory: 512 }],
  requiresCompatibilities: ['FARGATE'],
  networkMode: 'awsvpc', cpu: '256', memory: '512',
}));
await ecs.send(new CreateServiceCommand({
  cluster: 'dev', serviceName: 'workers',
  taskDefinition: 'worker:1', desiredCount: 1, launchType: 'FARGATE',
}));
```

### Python

```python
import boto3

ecs = boto3.client('ecs', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

ecs.create_cluster(clusterName='dev')
ecs.register_task_definition(
    family='worker',
    containerDefinitions=[{'name': 'app', 'image': 'myapp:latest', 'cpu': 256, 'memory': 512}],
    requiresCompatibilities=['FARGATE'],
    networkMode='awsvpc', cpu='256', memory='512',
)
ecs.create_service(
    cluster='dev', serviceName='workers',
    taskDefinition='worker:1', desiredCount=1, launchType='FARGATE',
)

response = ecs.describe_services(cluster='dev', services=['workers'])
svc = response['services'][0]
print(svc['status'], svc['runningCount'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ecs:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Tasks and services are **metadata records**. No container runtime is involved.
- `RunTask` returns a task record with status `RUNNING` immediately. No container is started.
- **Service auto-scaling** and **capacity providers** are not implemented.
- **Container Insights** and **ECS Anywhere** are not implemented.
- **Service Connect** and **service discovery** integration are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ClusterNotFoundException | 400 | The specified cluster does not exist |
| ServiceNotFoundException | 400 | The specified service does not exist |
| ServiceNotActiveException | 400 | The service is not active |
| InvalidParameterException | 400 | An input parameter is not valid |
