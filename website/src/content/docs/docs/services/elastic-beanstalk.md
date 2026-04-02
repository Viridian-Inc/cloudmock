---
title: Elastic Beanstalk
description: AWS Elastic Beanstalk emulation in CloudMock
---

## Overview

CloudMock emulates AWS Elastic Beanstalk, supporting the full application/version/environment hierarchy, configuration templates, platform version listing, and configuration settings validation.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApplication | Supported | Creates an application |
| DescribeApplications | Supported | Lists applications |
| UpdateApplication | Supported | Updates application description |
| DeleteApplication | Supported | Cascades to versions and templates |
| CreateApplicationVersion | Supported | Multi-version support per application |
| DescribeApplicationVersions | Supported | Filter by application name |
| DeleteApplicationVersion | Supported | Removes a specific version |
| CreateEnvironment | Supported | State machine: Launching → Ready |
| DescribeEnvironments | Supported | Filter by application name |
| UpdateEnvironment | Supported | Update version label or description |
| TerminateEnvironment | Supported | State machine: Terminating → Terminated |
| CreateConfigurationTemplate | Supported | |
| DescribeConfigurationSettings | Supported | |
| ValidateConfigurationSettings | Supported | Returns empty messages (all valid in mock) |
| DeleteConfigurationTemplate | Supported | |
| ListPlatformVersions | Supported | Returns mock platform list (Docker, Node.js, Python, Java) |

## Environment State Machine

```
Launching → Ready → Updating → Terminating → Terminated
```

## Quick Start

### Node.js

```typescript
import { ElasticBeanstalkClient, CreateApplicationCommand, CreateApplicationVersionCommand, CreateEnvironmentCommand } from '@aws-sdk/client-elastic-beanstalk';

const client = new ElasticBeanstalkClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateApplicationCommand({ ApplicationName: 'my-app' }));

await client.send(new CreateApplicationVersionCommand({
  ApplicationName: 'my-app',
  VersionLabel: 'v1.0.0',
  SourceBundle: { S3Bucket: 'my-bucket', S3Key: 'app-v1.zip' },
}));

await client.send(new CreateEnvironmentCommand({
  ApplicationName: 'my-app',
  EnvironmentName: 'my-env',
  VersionLabel: 'v1.0.0',
  SolutionStackName: '64bit Amazon Linux 2023 v6.0.0 running Node.js 18',
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
client.create_application_version(
    ApplicationName='my-app',
    VersionLabel='v1.0.0',
    SourceBundle={'S3Bucket': 'my-bucket', 'S3Key': 'app-v1.zip'})
client.create_environment(
    ApplicationName='my-app',
    EnvironmentName='my-env',
    VersionLabel='v1.0.0')
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
- Environment health transitions are simulated
- Application deployments are not executed
- ValidateConfigurationSettings always returns zero messages (all valid)
- ListPlatformVersions returns a static list of common platforms
