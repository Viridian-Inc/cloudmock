---
title: Bedrock
description: Amazon Bedrock emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Bedrock, supporting foundation model listing, provisioned throughput management, model customization jobs, guardrails, model invocation, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetFoundationModel | Supported | Returns foundation model details |
| ListFoundationModels | Supported | Lists available foundation models |
| CreateProvisionedModelThroughput | Supported | Creates provisioned throughput |
| GetProvisionedModelThroughput | Supported | Returns provisioned throughput details |
| ListProvisionedModelThroughputs | Supported | Lists provisioned throughputs |
| UpdateProvisionedModelThroughput | Supported | Updates provisioned throughput |
| DeleteProvisionedModelThroughput | Supported | Deletes provisioned throughput |
| CreateModelCustomizationJob | Supported | Creates a model customization job |
| GetModelCustomizationJob | Supported | Returns customization job details |
| ListModelCustomizationJobs | Supported | Lists customization jobs |
| StopModelCustomizationJob | Supported | Stops a customization job |
| InvokeModel | Supported | Returns a stub model response |
| CreateGuardrail | Supported | Creates a guardrail |
| ApplyGuardrail | Supported | Applies a guardrail to content |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { BedrockClient, ListFoundationModelsCommand } from '@aws-sdk/client-bedrock';

const client = new BedrockClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { modelSummaries } = await client.send(new ListFoundationModelsCommand({}));
console.log(modelSummaries);
```

### Python

```python
import boto3

client = boto3.client('bedrock',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.list_foundation_models()
print(response['modelSummaries'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  bedrock:
    enabled: true
```

## Known Differences from AWS

- InvokeModel returns stub responses, not actual model inference results
- Foundation model list contains a predefined set of models
- Model customization jobs do not perform actual training
- Guardrail evaluation is simplified
