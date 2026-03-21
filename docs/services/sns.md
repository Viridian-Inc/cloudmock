# SNS — Simple Notification Service

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `sns`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateTopic` | Creates a topic and returns its ARN |
| `DeleteTopic` | Deletes a topic and all its subscriptions |
| `ListTopics` | Returns all topic ARNs |
| `GetTopicAttributes` | Returns topic attributes |
| `SetTopicAttributes` | Sets topic attributes |
| `Subscribe` | Creates a subscription (SQS, HTTP, email, lambda) |
| `Unsubscribe` | Removes a subscription |
| `ListSubscriptions` | Returns all subscriptions |
| `ListSubscriptionsByTopic` | Returns subscriptions for a specific topic |
| `Publish` | Publishes a message; fans out to SQS subscribers |
| `TagResource` | Adds tags to a topic |
| `UntagResource` | Removes tags from a topic |

## Examples

### AWS CLI

```bash
# Create a topic
aws sns create-topic --name notifications

# Subscribe an SQS queue
aws sns subscribe \
  --topic-arn arn:aws:sns:us-east-1:000000000000:notifications \
  --protocol sqs \
  --notification-endpoint arn:aws:sqs:us-east-1:000000000000:my-queue

# Publish a message
aws sns publish \
  --topic-arn arn:aws:sns:us-east-1:000000000000:notifications \
  --message "Hello subscribers"

# List subscriptions
aws sns list-subscriptions-by-topic \
  --topic-arn arn:aws:sns:us-east-1:000000000000:notifications
```

### Python (boto3)

```python
import boto3

sns = boto3.client("sns", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create topic
topic = sns.create_topic(Name="alerts")
topic_arn = topic["TopicArn"]

# Subscribe SQS queue
sns.subscribe(
    TopicArn=topic_arn,
    Protocol="sqs",
    Endpoint="arn:aws:sqs:us-east-1:000000000000:my-queue",
)

# Publish
sns.publish(TopicArn=topic_arn, Message="Something happened", Subject="Alert")
```

## Notes

- Fan-out to SQS subscribers is implemented: publishing to a topic delivers the message to all subscribed SQS queues within cloudmock.
- HTTP/HTTPS and email protocol subscriptions are stored but delivery is not attempted.
- Lambda subscriptions record the ARN but do not invoke the function.
- Message filtering by subscription attributes is not implemented.
