---
title: Pinpoint
description: Amazon Pinpoint emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Pinpoint, supporting application management, segments, campaigns, journeys, and endpoints.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateApp | Supported | Creates a Pinpoint application |
| GetApp | Supported | Returns application details |
| GetApps | Supported | Lists all applications |
| DeleteApp | Supported | Deletes an application |
| CreateSegment | Supported | Creates a segment |
| GetSegment | Supported | Returns segment details |
| GetSegments | Supported | Lists segments for an app |
| DeleteSegment | Supported | Deletes a segment |
| CreateCampaign | Supported | Creates a campaign |
| GetCampaign | Supported | Returns campaign details |
| GetCampaigns | Supported | Lists campaigns for an app |
| DeleteCampaign | Supported | Deletes a campaign |
| CreateJourney | Supported | Creates a journey |
| GetJourney | Supported | Returns journey details |
| ListJourneys | Supported | Lists journeys for an app |
| UpdateEndpoint | Supported | Creates or updates an endpoint |
| GetEndpoint | Supported | Returns endpoint details |

## Quick Start

### Node.js

```typescript
import { PinpointClient, CreateAppCommand } from '@aws-sdk/client-pinpoint';

const client = new PinpointClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ApplicationResponse } = await client.send(new CreateAppCommand({
  CreateApplicationRequest: { Name: 'my-app' },
}));
console.log(ApplicationResponse.Id);
```

### Python

```python
import boto3

client = boto3.client('pinpoint',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_app(
    CreateApplicationRequest={'Name': 'my-app'})
print(response['ApplicationResponse']['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  pinpoint:
    enabled: true
```

## Known Differences from AWS

- Campaigns and journeys do not send actual messages
- Segments do not evaluate against real endpoint data dynamically
- Analytics and metrics are not collected
