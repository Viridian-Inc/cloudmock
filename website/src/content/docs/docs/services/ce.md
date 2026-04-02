---
title: Cost Explorer
description: AWS Cost Explorer emulation in CloudMock
---

## Overview

CloudMock emulates AWS Cost Explorer (CE), supporting cost and usage queries, forecasts, dimension values, tags, and utilization reports.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetCostAndUsage | Supported | Returns stub cost and usage data |
| GetCostForecast | Supported | Returns stub cost forecast |
| GetDimensionValues | Supported | Returns available dimension values |
| GetTags | Supported | Returns available cost allocation tags |
| GetReservationUtilization | Supported | Returns stub reservation utilization |
| GetSavingsPlansUtilization | Supported | Returns stub savings plans utilization |

## Quick Start

### Node.js

```typescript
import { CostExplorerClient, GetCostAndUsageCommand } from '@aws-sdk/client-cost-explorer';

const client = new CostExplorerClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const result = await client.send(new GetCostAndUsageCommand({
  TimePeriod: { Start: '2024-01-01', End: '2024-02-01' },
  Granularity: 'MONTHLY',
  Metrics: ['UnblendedCost'],
}));
console.log(result.ResultsByTime);
```

### Python

```python
import boto3

client = boto3.client('ce',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.get_cost_and_usage(
    TimePeriod={'Start': '2024-01-01', 'End': '2024-02-01'},
    Granularity='MONTHLY',
    Metrics=['UnblendedCost'])
print(response['ResultsByTime'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ce:
    enabled: true
```

## Known Differences from AWS

- All cost data is synthetic/stub data, not based on actual resource usage
- Forecasts return placeholder values
- Dimension values and tags are predefined sets
