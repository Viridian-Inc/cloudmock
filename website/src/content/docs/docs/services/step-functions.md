---
title: Step Functions
description: AWS Step Functions emulation in CloudMock
---

## Overview

CloudMock emulates AWS Step Functions, supporting state machine lifecycle management, execution tracking, event history retrieval, and tagging. Executions are recorded but the ASL definition is not interpreted.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateStateMachine | Supported | Creates a state machine from an ASL definition |
| DeleteStateMachine | Supported | Deletes a state machine |
| DescribeStateMachine | Supported | Returns state machine definition and metadata |
| ListStateMachines | Supported | Returns all state machines |
| UpdateStateMachine | Supported | Updates the definition or role ARN |
| StartExecution | Supported | Starts an execution and returns its ARN |
| DescribeExecution | Supported | Returns execution status and input/output |
| StopExecution | Supported | Stops a running execution |
| ListExecutions | Supported | Returns executions for a state machine |
| GetExecutionHistory | Supported | Returns the event history of an execution |
| TagResource | Supported | Adds tags to a state machine |
| UntagResource | Supported | Removes tags |
| ListTagsForResource | Supported | Returns tags for a resource |

## Quick Start

### curl

```bash
# Create a state machine
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AWSStepFunctions.CreateStateMachine" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -d '{
    "name": "HelloWorld",
    "definition": "{\"Comment\":\"Test\",\"StartAt\":\"Hello\",\"States\":{\"Hello\":{\"Type\":\"Pass\",\"End\":true}}}",
    "roleArn": "arn:aws:iam::000000000000:role/sfn-role"
  }'

# Start an execution
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AWSStepFunctions.StartExecution" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -d '{"stateMachineArn": "arn:aws:states:us-east-1:000000000000:stateMachine:HelloWorld", "input": "{\"key\":\"value\"}"}'
```

### Node.js

```typescript
import { SFNClient, CreateStateMachineCommand, StartExecutionCommand } from '@aws-sdk/client-sfn';

const sfn = new SFNClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const sm = await sfn.send(new CreateStateMachineCommand({
  name: 'MyFlow',
  definition: JSON.stringify({
    Comment: 'Simple', StartAt: 'Pass', States: { Pass: { Type: 'Pass', End: true } },
  }),
  roleArn: 'arn:aws:iam::000000000000:role/sfn-role',
}));

const exec = await sfn.send(new StartExecutionCommand({
  stateMachineArn: sm.stateMachineArn,
  input: JSON.stringify({ orderId: 'o-123' }),
}));
```

### Python

```python
import boto3, json

sfn = boto3.client('stepfunctions', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

definition = {
    'Comment': 'Simple pass-through',
    'StartAt': 'PassState',
    'States': {'PassState': {'Type': 'Pass', 'End': True}},
}

sm = sfn.create_state_machine(
    name='MyFlow', definition=json.dumps(definition),
    roleArn='arn:aws:iam::000000000000:role/sfn-role',
)

execution = sfn.start_execution(
    stateMachineArn=sm['stateMachineArn'],
    input=json.dumps({'orderId': 'o-123'}),
)
result = sfn.describe_execution(executionArn=execution['executionArn'])
print(result['status'])  # SUCCEEDED
```

## Configuration

```yaml
# cloudmock.yml
services:
  stepfunctions:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Executions are recorded with their input and status but the **state machine definition is not interpreted**. Executions immediately transition to `SUCCEEDED`.
- `GetExecutionHistory` returns a **minimal event list** reflecting only the start and end of the execution.
- **Express workflows** are accepted but behave identically to standard workflows.
- **Activity tasks** and heartbeats are not implemented.
- **Map and Parallel states** are not executed.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| StateMachineDoesNotExist | 400 | The specified state machine does not exist |
| StateMachineAlreadyExists | 400 | A state machine with this name already exists |
| ExecutionDoesNotExist | 400 | The specified execution does not exist |
| ExecutionAlreadyExists | 400 | An execution with this name already exists |
| InvalidDefinition | 400 | The state machine definition is not valid |
