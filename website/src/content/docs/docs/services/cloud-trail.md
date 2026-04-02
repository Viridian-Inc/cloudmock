---
title: CloudTrail
description: AWS CloudTrail emulation in CloudMock
---

## Overview

CloudMock emulates AWS CloudTrail, supporting trail management, logging control, event selectors, insight selectors, event lookup, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateTrail | Supported | Creates a trail |
| GetTrail | Supported | Returns trail details |
| DescribeTrails | Supported | Lists all trails |
| DeleteTrail | Supported | Deletes a trail |
| UpdateTrail | Supported | Updates trail configuration |
| StartLogging | Supported | Starts logging for a trail |
| StopLogging | Supported | Stops logging for a trail |
| GetTrailStatus | Supported | Returns trail logging status |
| PutEventSelectors | Supported | Configures event selectors |
| GetEventSelectors | Supported | Returns event selectors |
| PutInsightSelectors | Supported | Configures insight selectors |
| GetInsightSelectors | Supported | Returns insight selectors |
| LookupEvents | Supported | Returns recorded events |
| AddTags | Supported | Adds tags to a trail |
| RemoveTags | Supported | Removes tags from a trail |
| ListTags | Supported | Lists tags for a trail |

## Quick Start

### Node.js

```typescript
import { CloudTrailClient, CreateTrailCommand } from '@aws-sdk/client-cloudtrail';

const client = new CloudTrailClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateTrailCommand({
  Name: 'my-trail',
  S3BucketName: 'my-trail-bucket',
}));
```

### Python

```python
import boto3

client = boto3.client('cloudtrail',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_trail(
    Name='my-trail',
    S3BucketName='my-trail-bucket')
```

## Configuration

```yaml
# cloudmock.yml
services:
  cloudtrail:
    enabled: true
```

## Known Differences from AWS

- Events are recorded in-memory and do not persist to S3
- LookupEvents returns internally recorded API calls, not comprehensive AWS event history
- Insight events are not generated
