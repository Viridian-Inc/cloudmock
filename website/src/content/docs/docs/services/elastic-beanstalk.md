---
title: Elastic Beanstalk
description: AWS Elastic Beanstalk emulation in CloudMock
---

## Overview

CloudMock emulates AWS Elastic Beanstalk, supporting application and environment lifecycle, application versions, and configuration templates.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an application |
| DescribeApplications | Supported | Lists applications |
| DeleteApplication | Supported | Deletes an application |
| CreateApplicationVersion | Supported | Creates an application version |
| DescribeApplicationVersions | Supported | Lists application versions |
| CreateEnvironment | Supported | Creates an environment |
| DescribeEnvironments | Supported | Lists environments |
| TerminateEnvironment | Supported | Terminates an environment |
| CreateConfigurationTemplate | Supported | Creates a configuration template |
| DescribeConfigurationSettings | Supported | Returns configuration settings |
| DeleteConfigurationTemplate | Supported | Deletes a configuration template |

## Quick Start

### Node.js

```typescript
import { ElasticBeanstalkClient, CreateApplicationCommand, CreateEnvironmentCommand } from '@aws-sdk/client-elastic-beanstalk';

const client = new ElasticBeanstalkClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateApplicationCommand({
  ApplicationName: 'my-app',
}));

await client.send(new CreateEnvironmentCommand({
  ApplicationName: 'my-app',
  EnvironmentName: 'my-env',
  SolutionStackName: '64bit Amazon Linux 2 v5.8.0 running Node.js 18',
}));
```

### Python

```python
import boto3

client = boto3.client('elasticbeanstalk',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_application(ApplicationName='my-app')
client.create_environment(
    ApplicationName='my-app',
    EnvironmentName='my-env',
    SolutionStackName='64bit Amazon Linux 2 v5.8.0 running Node.js 18')
```

## Configuration

```yaml
# cloudmock.yml
services:
  elasticbeanstalk:
    enabled: true
```

## Known Differences from AWS

- No actual infrastructure is provisioned for environments
- Environment health is always reported as healthy
- Application deployments are not executed
