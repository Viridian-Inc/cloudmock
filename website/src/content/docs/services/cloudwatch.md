---
title: CloudWatch
description: Amazon CloudWatch emulation in CloudMock
---

## Overview

CloudMock emulates Amazon CloudWatch metrics and alarms, supporting metric data ingestion, querying, alarm creation with manual state management, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| PutMetricData | Supported | Stores metric data points |
| GetMetricData | Supported | Retrieves metric data with a metric query |
| ListMetrics | Supported | Returns all metrics with optional namespace/name filter |
| PutMetricAlarm | Supported | Creates or updates an alarm |
| DescribeAlarms | Supported | Returns alarms with optional name/state filter |
| DeleteAlarms | Supported | Removes one or more alarms |
| SetAlarmState | Supported | Manually sets an alarm state (OK, ALARM, INSUFFICIENT_DATA) |
| DescribeAlarmsForMetric | Supported | Returns alarms linked to a specific metric |
| TagResource | Supported | Adds tags to an alarm |
| UntagResource | Supported | Removes tags from an alarm |
| ListTagsForResource | Supported | Returns tags for an alarm |

## Quick Start

### curl

```bash
# Put metric data
curl -X POST "http://localhost:4566/?Action=PutMetricData&Namespace=MyApp&MetricData.member.1.MetricName=RequestCount&MetricData.member.1.Value=42&MetricData.member.1.Unit=Count"

# List metrics
curl -X POST "http://localhost:4566/?Action=ListMetrics&Namespace=MyApp"
```

### Node.js

```typescript
import { CloudWatchClient, PutMetricDataCommand, PutMetricAlarmCommand } from '@aws-sdk/client-cloudwatch';

const cw = new CloudWatchClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await cw.send(new PutMetricDataCommand({
  Namespace: 'MyService',
  MetricData: [
    { MetricName: 'Errors', Value: 5, Unit: 'Count' },
    { MetricName: 'Latency', Value: 120, Unit: 'Milliseconds' },
  ],
}));

await cw.send(new PutMetricAlarmCommand({
  AlarmName: 'HighErrorRate', MetricName: 'Errors', Namespace: 'MyService',
  Statistic: 'Sum', Period: 60, EvaluationPeriods: 1,
  Threshold: 10, ComparisonOperator: 'GreaterThanOrEqualToThreshold',
}));
```

### Python

```python
import boto3

cw = boto3.client('cloudwatch', endpoint_url='http://localhost:4566',
                  aws_access_key_id='test', aws_secret_access_key='test',
                  region_name='us-east-1')

cw.put_metric_data(
    Namespace='MyService',
    MetricData=[
        {'MetricName': 'Errors', 'Value': 5, 'Unit': 'Count'},
        {'MetricName': 'Latency', 'Value': 120, 'Unit': 'Milliseconds'},
    ],
)

cw.put_metric_alarm(
    AlarmName='HighErrorRate', MetricName='Errors', Namespace='MyService',
    Statistic='Sum', Period=60, EvaluationPeriods=1,
    Threshold=10, ComparisonOperator='GreaterThanOrEqualToThreshold',
)

# Manually trigger alarm for testing
cw.set_alarm_state(AlarmName='HighErrorRate', StateValue='ALARM', StateReason='Test')
```

## Configuration

```yaml
# cloudmock.yml
services:
  cloudwatch:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Metric data is stored **in memory** and queryable for the duration of the process.
- Alarm **state transitions** do not trigger SNS notifications or Auto Scaling actions.
- `GetMetricData` supports standard statistics (Sum, Average, Min, Max, SampleCount).
- **Composite alarms** are not implemented.
- **Metric math expressions** are not supported.
- **Dashboards** are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFound | 404 | The specified alarm does not exist |
| InvalidParameterValue | 400 | An input parameter is invalid |
| LimitExceededFault | 400 | The request exceeds a service limit |
| InvalidParameterCombination | 400 | Parameters cannot be used together |
