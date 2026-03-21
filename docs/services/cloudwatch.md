# CloudWatch

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `monitoring` (AWS service name: `cloudwatch`)

## Supported Actions

| Action | Notes |
|--------|-------|
| `PutMetricData` | Stores metric data points |
| `GetMetricData` | Retrieves metric data with a metric query |
| `ListMetrics` | Returns all metrics with optional namespace/name filter |
| `PutMetricAlarm` | Creates or updates an alarm |
| `DescribeAlarms` | Returns alarms with optional name/state filter |
| `DeleteAlarms` | Removes one or more alarms |
| `SetAlarmState` | Manually sets an alarm state (OK, ALARM, INSUFFICIENT_DATA) |
| `DescribeAlarmsForMetric` | Returns alarms linked to a specific metric |
| `TagResource` | Adds tags to an alarm |
| `UntagResource` | Removes tags from an alarm |
| `ListTagsForResource` | Returns tags for an alarm |

## Examples

### AWS CLI

```bash
# Put metric data
aws cloudwatch put-metric-data \
  --namespace "MyApp" \
  --metric-name "RequestCount" \
  --value 42 \
  --unit Count

# List metrics
aws cloudwatch list-metrics --namespace "MyApp"

# Create an alarm
aws cloudwatch put-metric-alarm \
  --alarm-name "HighRequests" \
  --metric-name "RequestCount" \
  --namespace "MyApp" \
  --statistic Sum \
  --period 60 \
  --evaluation-periods 1 \
  --threshold 100 \
  --comparison-operator GreaterThanOrEqualToThreshold

# Check alarm state
aws cloudwatch describe-alarms --alarm-names "HighRequests"

# Set alarm state manually (for testing)
aws cloudwatch set-alarm-state \
  --alarm-name "HighRequests" \
  --state-value ALARM \
  --state-reason "Testing"
```

### Python (boto3)

```python
import boto3
from datetime import datetime, timezone

cw = boto3.client("cloudwatch", endpoint_url="http://localhost:4566",
                  aws_access_key_id="test", aws_secret_access_key="test",
                  region_name="us-east-1")

# Put metric data
cw.put_metric_data(
    Namespace="MyService",
    MetricData=[
        {"MetricName": "Errors", "Value": 5, "Unit": "Count"},
        {"MetricName": "Latency", "Value": 120, "Unit": "Milliseconds"},
    ],
)

# Create alarm
cw.put_metric_alarm(
    AlarmName="HighErrorRate",
    MetricName="Errors",
    Namespace="MyService",
    Statistic="Sum",
    Period=60,
    EvaluationPeriods=1,
    Threshold=10,
    ComparisonOperator="GreaterThanOrEqualToThreshold",
)

# Manually trigger alarm in tests
cw.set_alarm_state(AlarmName="HighErrorRate", StateValue="ALARM", StateReason="Test")
```

## Notes

- Metric data is stored in memory and queryable for the duration of the process.
- Alarm state transitions do not trigger SNS notifications or Auto Scaling actions.
- `GetMetricData` supports standard statistics (Sum, Average, Min, Max, SampleCount).
- Composite alarms are not implemented.
