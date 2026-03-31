---
title: Lambda
description: AWS Lambda emulation in CloudMock
---

## Overview

CloudMock emulates AWS Lambda function management (create, update, delete, list, invoke). Function invocation returns a stub 200 response -- actual code execution is not performed.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateFunction | Supported | Stores function configuration and code reference |
| DeleteFunction | Supported | Removes a function |
| GetFunction | Supported | Returns function configuration |
| ListFunctions | Supported | Returns all functions |
| UpdateFunctionCode | Supported | Updates the code reference |
| UpdateFunctionConfiguration | Supported | Updates runtime, handler, environment, etc. |
| InvokeFunction | Supported | Returns a stub 200 response; does not execute code |
| AddPermission | Supported | Stores a resource-based policy statement |
| RemovePermission | Supported | Removes a policy statement |
| CreateEventSourceMapping | Supported | Stores an event source mapping |
| ListEventSourceMappings | Supported | Returns all event source mappings |
| TagResource | Supported | Adds tags to a function |
| UntagResource | Supported | Removes tags from a function |

## Quick Start

### curl

```bash
# Create a function
curl -X POST http://localhost:4566/2015-03-31/functions \
  -H "Content-Type: application/json" \
  -d '{
    "FunctionName": "my-function",
    "Runtime": "nodejs20.x",
    "Role": "arn:aws:iam::000000000000:role/lambda-role",
    "Handler": "index.handler",
    "Code": {"ZipFile": "UEsDBBQAAAAI..."}
  }'

# Invoke (stub response)
curl -X POST http://localhost:4566/2015-03-31/functions/my-function/invocations \
  -d '{"key": "value"}'
```

### Node.js

```typescript
import { LambdaClient, CreateFunctionCommand, InvokeCommand } from '@aws-sdk/client-lambda';

const lambda = new LambdaClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await lambda.send(new CreateFunctionCommand({
  FunctionName: 'hello',
  Runtime: 'nodejs20.x',
  Role: 'arn:aws:iam::000000000000:role/lambda-role',
  Handler: 'index.handler',
  Code: { ZipFile: Buffer.from('placeholder') },
}));

const response = await lambda.send(new InvokeCommand({
  FunctionName: 'hello', Payload: Buffer.from('{}'),
}));
```

### Python

```python
import boto3, zipfile, io

client = boto3.client('lambda', endpoint_url='http://localhost:4566',
                      aws_access_key_id='test', aws_secret_access_key='test',
                      region_name='us-east-1')

buf = io.BytesIO()
with zipfile.ZipFile(buf, 'w') as zf:
    zf.writestr('index.js', 'exports.handler = async () => ({ statusCode: 200 });')
buf.seek(0)

client.create_function(
    FunctionName='hello', Runtime='nodejs20.x',
    Role='arn:aws:iam::000000000000:role/lambda-role',
    Handler='index.handler', Code={'ZipFile': buf.read()},
)

response = client.invoke(FunctionName='hello', Payload=b'{}')
print(response['StatusCode'])  # 200
```

## Configuration

```yaml
# cloudmock.yml
services:
  lambda:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **Invocation is a stub** -- `InvokeFunction` returns an empty 200 response without executing any code.
- **Event source mappings** are stored and returned but no trigger logic runs.
- **Layers, versions, and aliases** are not implemented.
- **Concurrency limits** are not enforced.
- To test Lambda-triggered workflows, invoke your function logic directly in your test code and use CloudMock for the downstream services (SQS, DynamoDB, S3, etc.).

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 404 | The specified function does not exist |
| ResourceConflictException | 409 | A function with this name already exists |
| InvalidParameterValueException | 400 | A parameter value is not valid |
| ServiceException | 500 | Internal service error |
