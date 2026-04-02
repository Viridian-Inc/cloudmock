---
title: X-Ray
description: AWS X-Ray emulation in CloudMock
---

## Overview

CloudMock emulates AWS X-Ray, supporting trace segment ingestion, sampling rules management, group management, encryption configuration, and resource tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| PutTraceSegments | Supported | Ingests trace segment documents |
| BatchGetTraces | Supported | Retrieves traces by ID |
| GetTraceSummaries | Supported | Returns summaries of all stored traces |
| GetTraceGraph | Supported | Returns a service graph (empty stub) |
| GetSamplingRules | Supported | Lists all sampling rules including Default |
| CreateSamplingRule | Supported | Creates a new sampling rule |
| UpdateSamplingRule | Supported | Updates an existing sampling rule |
| DeleteSamplingRule | Supported | Deletes a sampling rule (Default cannot be deleted) |
| CreateGroup | Supported | Creates a filter group |
| GetGroup | Supported | Gets a group by name |
| GetGroups | Supported | Lists all groups |
| UpdateGroup | Supported | Updates group filter expression |
| DeleteGroup | Supported | Deletes a group |
| PutEncryptionConfig | Supported | Sets encryption configuration |
| GetEncryptionConfig | Supported | Gets current encryption configuration |
| TagResource | Supported | Adds tags to a resource ARN |
| UntagResource | Supported | Removes tags from a resource ARN |
| ListTagsForResource | Supported | Lists tags for a resource ARN |

## Quick Start

### Node.js

```typescript
import { XRayClient, PutTraceSegmentsCommand, GetSamplingRulesCommand } from '@aws-sdk/client-xray';

const client = new XRayClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// Put trace segments
await client.send(new PutTraceSegmentsCommand({
  TraceSegmentDocuments: [
    JSON.stringify({
      id: 'abc123def456',
      name: 'my-service',
      start_time: Date.now() / 1000,
      end_time: Date.now() / 1000 + 0.5,
      trace_id: '1-' + Math.floor(Date.now() / 1000).toString(16) + '-' + 'abcdef123456789012345678',
    }),
  ],
}));

// List sampling rules
const rules = await client.send(new GetSamplingRulesCommand({}));
console.log(rules.SamplingRuleRecords);
```

### Python

```python
import boto3

client = boto3.client(
    'xray',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test',
)

# Create a sampling rule
client.create_sampling_rule(
    SamplingRule={
        'RuleName': 'my-rule',
        'Priority': 100,
        'FixedRate': 0.05,
        'ReservoirSize': 10,
        'ServiceName': 'my-service',
        'ServiceType': '*',
        'Host': '*',
        'HTTPMethod': '*',
        'URLPath': '*',
        'Version': 1,
    }
)

# Create a group
client.create_group(
    GroupName='my-group',
    FilterExpression='service("my-service")',
)
```

## Notes

- A `Default` sampling rule is pre-seeded with `FixedRate=0.05` and `ReservoirSize=5`. It cannot be deleted.
- Encryption config defaults to `Type=NONE, Status=ACTIVE`. Setting `Type=KMS` requires a `KeyId`.
- Trace segment documents are stored in memory and not persisted between restarts.
