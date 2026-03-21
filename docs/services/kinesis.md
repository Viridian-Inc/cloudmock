# Kinesis Data Streams

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: Kinesis_20131202.<Action>`)
**Service Name:** `kinesis`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateStream` | Creates a stream with the specified shard count |
| `DeleteStream` | Deletes a stream and all its data |
| `DescribeStream` | Returns stream metadata and shard details |
| `ListStreams` | Returns all stream names |
| `PutRecord` | Writes a single record to a stream |
| `PutRecords` | Writes up to 500 records in one call |
| `GetShardIterator` | Returns a shard iterator for reading |
| `GetRecords` | Returns records from a shard iterator |
| `IncreaseStreamRetentionPeriod` | Sets retention period (hours) |
| `DecreaseStreamRetentionPeriod` | Reduces retention period |
| `AddTagsToStream` | Adds tags to a stream |
| `RemoveTagsFromStream` | Removes tags |
| `ListTagsForStream` | Returns tags for a stream |

## Examples

### AWS CLI

```bash
# Create a stream
aws kinesis create-stream --stream-name events --shard-count 1

# Put a record
aws kinesis put-record \
  --stream-name events \
  --data "$(echo -n '{"event":"click"}' | base64)" \
  --partition-key "user-123"

# Get shard iterator
aws kinesis get-shard-iterator \
  --stream-name events \
  --shard-id shardId-000000000000 \
  --shard-iterator-type TRIM_HORIZON

# Read records
aws kinesis get-records \
  --shard-iterator <ShardIterator>
```

### Python (boto3)

```python
import boto3, base64, json

kinesis = boto3.client("kinesis", endpoint_url="http://localhost:4566",
                       aws_access_key_id="test", aws_secret_access_key="test",
                       region_name="us-east-1")

# Create stream
kinesis.create_stream(StreamName="orders", ShardCount=1)

# Write records
kinesis.put_records(
    StreamName="orders",
    Records=[
        {"Data": json.dumps({"orderId": "o-1"}), "PartitionKey": "p1"},
        {"Data": json.dumps({"orderId": "o-2"}), "PartitionKey": "p2"},
    ],
)

# Read records
stream = kinesis.describe_stream(StreamName="orders")
shard_id = stream["StreamDescription"]["Shards"][0]["ShardId"]

iterator = kinesis.get_shard_iterator(
    StreamName="orders",
    ShardId=shard_id,
    ShardIteratorType="TRIM_HORIZON",
)["ShardIterator"]

records = kinesis.get_records(ShardIterator=iterator)
for r in records["Records"]:
    print(json.loads(r["Data"]))
```

## Notes

- Records are stored in memory per shard. Sequence numbers are monotonically increasing integers.
- `GetRecords` advances the iterator; subsequent calls return newer records.
- Enhanced fan-out (`SubscribeToShard`) is not implemented.
- Stream encryption and server-side encryption are accepted but not enforced.
