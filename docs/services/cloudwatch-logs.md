# CloudWatch Logs

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: Logs.<Action>`)
**Service Name:** `logs`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateLogGroup` | Creates a log group |
| `DeleteLogGroup` | Deletes a log group and all its streams |
| `DescribeLogGroups` | Lists log groups with optional prefix filter |
| `CreateLogStream` | Creates a log stream within a group |
| `DeleteLogStream` | Deletes a log stream |
| `DescribeLogStreams` | Lists log streams within a group |
| `PutLogEvents` | Appends log events to a stream; requires sequence token after first call |
| `GetLogEvents` | Returns log events from a stream |
| `FilterLogEvents` | Searches log events across streams with a filter pattern |
| `PutRetentionPolicy` | Sets the retention period for a log group |
| `DeleteRetentionPolicy` | Removes the retention policy (logs kept indefinitely) |
| `TagLogGroup` | Adds tags to a log group |
| `UntagLogGroup` | Removes tags from a log group |
| `ListTagsLogGroup` | Returns tags for a log group |

## Examples

### AWS CLI

```bash
# Create log group and stream
aws logs create-log-group --log-group-name /app/server
aws logs create-log-stream \
  --log-group-name /app/server \
  --log-stream-name app-1

# Put log events
aws logs put-log-events \
  --log-group-name /app/server \
  --log-stream-name app-1 \
  --log-events '[{"timestamp":1700000000000,"message":"Server started"}]'

# Retrieve log events
aws logs get-log-events \
  --log-group-name /app/server \
  --log-stream-name app-1

# Filter logs
aws logs filter-log-events \
  --log-group-name /app/server \
  --filter-pattern "ERROR"

# Set retention
aws logs put-retention-policy \
  --log-group-name /app/server \
  --retention-in-days 7
```

### Python (boto3)

```python
import boto3, time

logs = boto3.client("logs", endpoint_url="http://localhost:4566",
                    aws_access_key_id="test", aws_secret_access_key="test",
                    region_name="us-east-1")

# Setup
logs.create_log_group(logGroupName="/app/api")
logs.create_log_stream(logGroupName="/app/api", logStreamName="instance-1")

# Write events
now_ms = int(time.time() * 1000)
response = logs.put_log_events(
    logGroupName="/app/api",
    logStreamName="instance-1",
    logEvents=[
        {"timestamp": now_ms, "message": "Request received"},
        {"timestamp": now_ms + 1, "message": "Response sent 200"},
    ],
)

# Read events
events = logs.get_log_events(
    logGroupName="/app/api",
    logStreamName="instance-1",
)
for event in events["events"]:
    print(event["message"])
```

## Notes

- `PutLogEvents` requires a `sequenceToken` after the first call to a stream. The token from each response must be passed in the next call.
- `FilterLogEvents` performs substring matching; CloudWatch Insights syntax is not supported.
- Log retention enforcement (automatic deletion of old events) is not implemented.
