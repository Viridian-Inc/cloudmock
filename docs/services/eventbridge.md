# EventBridge

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: AWSEvents.<Action>`)
**Service Name:** `events`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateEventBus` | Creates a custom event bus |
| `DeleteEventBus` | Deletes a custom event bus |
| `DescribeEventBus` | Returns event bus details |
| `ListEventBuses` | Returns all event buses including the default |
| `PutRule` | Creates or updates a rule on an event bus |
| `DeleteRule` | Deletes a rule |
| `DescribeRule` | Returns rule details |
| `ListRules` | Returns rules on an event bus |
| `PutTargets` | Associates targets (SQS, Lambda, etc.) with a rule |
| `RemoveTargets` | Removes targets from a rule |
| `ListTargetsByRule` | Returns all targets for a rule |
| `PutEvents` | Publishes custom events to event buses |
| `EnableRule` | Enables a disabled rule |
| `DisableRule` | Disables a rule without deleting it |
| `TagResource` | Adds tags to a rule or event bus |
| `UntagResource` | Removes tags |
| `ListTagsForResource` | Returns tags for a resource |

## Examples

### AWS CLI

```bash
# Create a custom event bus
aws events create-event-bus --name my-app-bus

# Create a rule
aws events put-rule \
  --name OrderCreated \
  --event-bus-name my-app-bus \
  --event-pattern '{"source":["com.myapp.orders"]}' \
  --state ENABLED

# Add an SQS target
aws events put-targets \
  --rule OrderCreated \
  --event-bus-name my-app-bus \
  --targets '[{"Id":"1","Arn":"arn:aws:sqs:us-east-1:000000000000:orders-queue"}]'

# Publish an event
aws events put-events \
  --entries '[{
    "EventBusName": "my-app-bus",
    "Source": "com.myapp.orders",
    "DetailType": "OrderCreated",
    "Detail": "{\"orderId\": \"o-123\"}"
  }]'
```

### Python (boto3)

```python
import boto3, json

events = boto3.client("events", endpoint_url="http://localhost:4566",
                      aws_access_key_id="test", aws_secret_access_key="test",
                      region_name="us-east-1")

# Create bus and rule
events.create_event_bus(Name="app-bus")
events.put_rule(
    Name="UserSignedUp",
    EventBusName="app-bus",
    EventPattern=json.dumps({"source": ["com.myapp.auth"]}),
    State="ENABLED",
)

# Send event
events.put_events(Entries=[{
    "EventBusName": "app-bus",
    "Source": "com.myapp.auth",
    "DetailType": "UserSignedUp",
    "Detail": json.dumps({"userId": "u-456"}),
}])
```

## Notes

- `PutEvents` stores events in memory and fans out to SQS targets within cloudmock.
- Scheduled rules (cron/rate expressions) are stored but do not trigger on a schedule.
- Lambda, SNS, and other target types are stored but delivery is not implemented beyond SQS.
- The default event bus (`default`) is always available.
