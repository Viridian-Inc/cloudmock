---
title: SWF
description: Amazon Simple Workflow Service emulation in CloudMock
---

## Overview

CloudMock emulates Amazon SWF, supporting domains, workflow and activity types, workflow execution lifecycle, decision and activity task polling, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RegisterDomain | Supported | Registers a domain |
| DescribeDomain | Supported | Returns domain details |
| ListDomains | Supported | Lists domains |
| DeprecateDomain | Supported | Deprecates a domain |
| RegisterWorkflowType | Supported | Registers a workflow type |
| DescribeWorkflowType | Supported | Returns workflow type details |
| ListWorkflowTypes | Supported | Lists workflow types |
| DeprecateWorkflowType | Supported | Deprecates a workflow type |
| RegisterActivityType | Supported | Registers an activity type |
| DescribeActivityType | Supported | Returns activity type details |
| ListActivityTypes | Supported | Lists activity types |
| DeprecateActivityType | Supported | Deprecates an activity type |
| StartWorkflowExecution | Supported | Starts a workflow execution |
| DescribeWorkflowExecution | Supported | Returns execution details |
| ListOpenWorkflowExecutions | Supported | Lists open executions |
| ListClosedWorkflowExecutions | Supported | Lists closed executions |
| TerminateWorkflowExecution | Supported | Terminates an execution |
| SignalWorkflowExecution | Supported | Sends a signal to an execution |
| RequestCancelWorkflowExecution | Supported | Requests cancellation |
| PollForDecisionTask | Supported | Polls for a decision task |
| RespondDecisionTaskCompleted | Supported | Completes a decision task |
| PollForActivityTask | Supported | Polls for an activity task |
| RespondActivityTaskCompleted | Supported | Completes an activity task |
| RespondActivityTaskFailed | Supported | Fails an activity task |
| GetWorkflowExecutionHistory | Supported | Returns execution history |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { SWFClient, RegisterDomainCommand, StartWorkflowExecutionCommand } from '@aws-sdk/client-swf';

const client = new SWFClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new RegisterDomainCommand({
  name: 'my-domain',
  workflowExecutionRetentionPeriodInDays: '30',
}));

const { runId } = await client.send(new StartWorkflowExecutionCommand({
  domain: 'my-domain',
  workflowId: 'my-workflow-1',
  workflowType: { name: 'my-workflow', version: '1.0' },
}));
console.log(runId);
```

### Python

```python
import boto3

client = boto3.client('swf',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.register_domain(
    name='my-domain',
    workflowExecutionRetentionPeriodInDays='30')

response = client.start_workflow_execution(
    domain='my-domain',
    workflowId='my-workflow-1',
    workflowType={'name': 'my-workflow', 'version': '1.0'})
print(response['runId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  swf:
    enabled: true
```

## Known Differences from AWS

- Task polling returns stubs; actual distributed task coordination is simplified
- Workflow execution history is basic
- Long polling is not implemented; responses are immediate
