---
title: CloudFormation
description: AWS CloudFormation emulation in CloudMock
---

## Overview

CloudMock emulates AWS CloudFormation stack management, supporting stack lifecycle, template validation, resource listing, change sets, and exports. Stack resources are stored as metadata -- CloudFormation does not create actual resources in the emulator.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateStack | Supported | Creates a stack from a template |
| DeleteStack | Supported | Deletes a stack |
| DescribeStacks | Supported | Returns stack metadata and outputs |
| ListStacks | Supported | Returns stack summaries with optional status filter |
| DescribeStackResources | Supported | Returns the resources in a stack |
| DescribeStackEvents | Supported | Returns the event history for a stack |
| GetTemplate | Supported | Returns the template body for a stack |
| ValidateTemplate | Supported | Validates a template and returns parameter names |
| ListExports | Supported | Returns all stack exports |
| CreateChangeSet | Supported | Creates a change set for a stack |
| DescribeChangeSet | Supported | Returns change set details |
| ExecuteChangeSet | Supported | Applies a change set to a stack |
| DeleteChangeSet | Supported | Discards a change set |

## Quick Start

### curl

```bash
# Validate a template
curl -X POST "http://localhost:4566/?Action=ValidateTemplate&TemplateBody=%7B%22AWSTemplateFormatVersion%22%3A%222010-09-09%22%7D"

# Create a stack
curl -X POST "http://localhost:4566/?Action=CreateStack&StackName=my-stack&TemplateBody=%7B%22AWSTemplateFormatVersion%22%3A%222010-09-09%22%7D"
```

### Node.js

```typescript
import { CloudFormationClient, CreateStackCommand, DescribeStacksCommand } from '@aws-sdk/client-cloudformation';

const cf = new CloudFormationClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await cf.send(new CreateStackCommand({
  StackName: 'infra',
  TemplateBody: JSON.stringify({
    AWSTemplateFormatVersion: '2010-09-09',
    Resources: {
      MyBucket: { Type: 'AWS::S3::Bucket', Properties: { BucketName: 'my-cf-bucket' } },
    },
  }),
}));

const { Stacks } = await cf.send(new DescribeStacksCommand({ StackName: 'infra' }));
console.log(Stacks?.[0]?.StackStatus); // CREATE_COMPLETE
```

### Python

```python
import boto3

cf = boto3.client('cloudformation', endpoint_url='http://localhost:4566',
                  aws_access_key_id='test', aws_secret_access_key='test',
                  region_name='us-east-1')

template = """
AWSTemplateFormatVersion: '2010-09-09'
Parameters:
  BucketName:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref BucketName
Outputs:
  BucketArn:
    Value: !GetAtt MyBucket.Arn
"""

cf.create_stack(
    StackName='infra', TemplateBody=template,
    Parameters=[{'ParameterKey': 'BucketName', 'ParameterValue': 'my-cf-bucket'}],
)

response = cf.describe_stacks(StackName='infra')
print(response['Stacks'][0]['StackStatus'])  # CREATE_COMPLETE
```

## Configuration

```yaml
# cloudmock.yml
services:
  cloudformation:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Stack resources listed in the template are stored as **metadata only**. CloudFormation does not create actual resources (e.g., an `AWS::S3::Bucket` in the template does not create a real S3 bucket).
- Stacks immediately transition to `CREATE_COMPLETE` after `CreateStack`.
- Change sets transition to `CREATE_COMPLETE` immediately and can be executed without waits.
- **Nested stacks**, **stack sets**, and **drift detection** are not implemented.
- **Intrinsic functions** (`!Ref`, `!GetAtt`, etc.) are not evaluated.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ValidationError | 400 | The template or parameters are not valid |
| AlreadyExistsException | 400 | A stack with this name already exists |
| ChangeSetNotFound | 404 | The specified change set does not exist |
| InsufficientCapabilitiesException | 400 | Required capabilities were not acknowledged |
