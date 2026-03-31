---
title: API Gateway
description: Amazon API Gateway (REST APIs) emulation in CloudMock
---

## Overview

CloudMock emulates the Amazon API Gateway management API for REST APIs, supporting API creation, resource/method/integration configuration, deployments, and stage management.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateRestApi | Supported | Creates a REST API and returns its ID |
| GetRestApis | Supported | Returns all REST APIs |
| GetRestApi | Supported | Returns a specific REST API |
| DeleteRestApi | Supported | Deletes a REST API |
| CreateResource | Supported | Creates a resource (path part) under a parent |
| GetResources | Supported | Returns all resources in an API |
| DeleteResource | Supported | Deletes a resource |
| PutMethod | Supported | Adds an HTTP method to a resource |
| GetMethod | Supported | Returns method configuration |
| PutIntegration | Supported | Associates an integration (Lambda, HTTP, Mock) with a method |
| CreateDeployment | Supported | Deploys an API to a stage |
| GetDeployments | Supported | Returns all deployments |
| CreateStage | Supported | Creates a stage pointing at a deployment |
| GetStages | Supported | Returns all stages for an API |

## Quick Start

### curl

```bash
# Create an API
curl -X POST http://localhost:4566/restapis \
  -H "Content-Type: application/json" \
  -d '{"name": "MyAPI"}'

# Get resources (root resource)
curl http://localhost:4566/restapis/<api-id>/resources
```

### Node.js

```typescript
import { APIGatewayClient, CreateRestApiCommand, CreateResourceCommand, CreateDeploymentCommand } from '@aws-sdk/client-api-gateway';

const apigw = new APIGatewayClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const api = await apigw.send(new CreateRestApiCommand({ name: 'TestAPI' }));
const resources = await apigw.send(new GetResourcesCommand({ restApiId: api.id }));
const rootId = resources.items?.find(r => r.path === '/')?.id;

await apigw.send(new CreateResourceCommand({
  restApiId: api.id!, parentId: rootId!, pathPart: 'hello',
}));
await apigw.send(new CreateDeploymentCommand({
  restApiId: api.id!, stageName: 'dev',
}));
```

### Python

```python
import boto3

apigw = boto3.client('apigateway', endpoint_url='http://localhost:4566',
                     aws_access_key_id='test', aws_secret_access_key='test',
                     region_name='us-east-1')

api = apigw.create_rest_api(name='TestAPI')
api_id = api['id']

resources = apigw.get_resources(restApiId=api_id)
root_id = next(r['id'] for r in resources['items'] if r['path'] == '/')

resource = apigw.create_resource(restApiId=api_id, parentId=root_id, pathPart='hello')
apigw.put_method(restApiId=api_id, resourceId=resource['id'],
                 httpMethod='GET', authorizationType='NONE')
apigw.put_integration(restApiId=api_id, resourceId=resource['id'],
                      httpMethod='GET', type='MOCK')
apigw.create_deployment(restApiId=api_id, stageName='dev')
```

## Configuration

```yaml
# cloudmock.yml
services:
  apigateway:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **API invocation** through the emulated endpoint is not supported -- only the management API is emulated.
- HTTP and Lambda **proxy integrations** are recorded but not executed.
- **API Gateway v2** (HTTP APIs and WebSocket APIs) is not implemented.
- **Usage plans**, **API keys**, and **authorizers** are not implemented.
- **Request/response models** and **validators** are not enforced.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| NotFoundException | 404 | The specified API or resource does not exist |
| ConflictException | 409 | A resource with this path already exists |
| BadRequestException | 400 | The request is not valid |
| LimitExceededException | 429 | A service limit has been exceeded |
