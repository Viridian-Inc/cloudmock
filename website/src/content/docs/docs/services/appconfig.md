---
title: AppConfig
description: AWS AppConfig emulation in CloudMock
---

## Overview

CloudMock emulates AWS AppConfig, supporting application, environment, configuration profile, deployment strategy, and deployment management.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an application |
| GetApplication | Supported | Returns application details |
| ListApplications | Supported | Lists all applications |
| UpdateApplication | Supported | Updates an application |
| DeleteApplication | Supported | Deletes an application |
| CreateEnvironment | Supported | Creates an environment |
| GetEnvironment | Supported | Returns environment details |
| ListEnvironments | Supported | Lists environments for an app |
| UpdateEnvironment | Supported | Updates an environment |
| DeleteEnvironment | Supported | Deletes an environment |
| CreateConfigurationProfile | Supported | Creates a configuration profile |
| GetConfigurationProfile | Supported | Returns configuration profile details |
| ListConfigurationProfiles | Supported | Lists configuration profiles |
| DeleteConfigurationProfile | Supported | Deletes a configuration profile |
| CreateDeploymentStrategy | Supported | Creates a deployment strategy |
| GetDeploymentStrategy | Supported | Returns deployment strategy details |
| ListDeploymentStrategies | Supported | Lists deployment strategies |
| DeleteDeploymentStrategy | Supported | Deletes a deployment strategy |
| StartDeployment | Supported | Starts a configuration deployment |
| GetDeployment | Supported | Returns deployment details |
| ListDeployments | Supported | Lists deployments |
| StopDeployment | Supported | Stops a running deployment |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { AppConfigClient, CreateApplicationCommand } from '@aws-sdk/client-appconfig';

const client = new AppConfigClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const app = await client.send(new CreateApplicationCommand({
  Name: 'my-app',
}));
console.log(app.Id);
```

### Python

```python
import boto3

client = boto3.client('appconfig',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_application(Name='my-app')
print(response['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  appconfig:
    enabled: true
```

## Known Differences from AWS

- Deployments complete immediately rather than following the deployment strategy timing
- Configuration validation is not performed
- No actual configuration delivery occurs
