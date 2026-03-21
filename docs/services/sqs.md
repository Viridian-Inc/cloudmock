# SQS — Simple Queue Service

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `sqs`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateQueue` | Creates a standard queue |
| `DeleteQueue` | Deletes a queue and all its messages |
| `ListQueues` | Returns queue URLs with optional prefix filter |
| `GetQueueUrl` | Returns the URL of a named queue |
| `GetQueueAttributes` | Returns queue attributes (ApproximateNumberOfMessages, etc.) |
| `SetQueueAttributes` | Sets queue attributes (VisibilityTimeout, etc.) |
| `SendMessage` | Sends a single message |
| `ReceiveMessage` | Returns up to 10 messages; marks them invisible |
| `DeleteMessage` | Permanently removes a received message using its receipt handle |
| `PurgeQueue` | Deletes all messages in the queue |
| `ChangeMessageVisibility` | Extends or resets visibility timeout |
| `SendMessageBatch` | Sends up to 10 messages in one call |
| `DeleteMessageBatch` | Deletes up to 10 messages in one call |

## Examples

### AWS CLI

```bash
# Create a queue
aws sqs create-queue --queue-name my-queue

# Send a message
aws sqs send-message \
  --queue-url http://localhost:4566/000000000000/my-queue \
  --message-body "Hello from cloudmock"

# Receive messages
aws sqs receive-message \
  --queue-url http://localhost:4566/000000000000/my-queue \
  --max-number-of-messages 5

# Delete a message
aws sqs delete-message \
  --queue-url http://localhost:4566/000000000000/my-queue \
  --receipt-handle <ReceiptHandle>

# Purge the queue
aws sqs purge-queue \
  --queue-url http://localhost:4566/000000000000/my-queue
```

### Python (boto3)

```python
import boto3

sqs = boto3.client("sqs", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create queue
response = sqs.create_queue(QueueName="jobs")
url = response["QueueUrl"]

# Send
sqs.send_message(QueueUrl=url, MessageBody='{"job": "process-image"}')

# Receive and delete
messages = sqs.receive_message(QueueUrl=url, MaxNumberOfMessages=10).get("Messages", [])
for msg in messages:
    print(msg["Body"])
    sqs.delete_message(QueueUrl=url, ReceiptHandle=msg["ReceiptHandle"])
```

## Notes

- Queue URLs follow the pattern `http://localhost:4566/{AccountId}/{QueueName}`.
- `VisibilityTimeout` is enforced — messages that are not deleted within the timeout reappear in the queue.
- FIFO queues are accepted at creation but do not enforce ordering or deduplication.
- Dead-letter queue redrive is not implemented.
