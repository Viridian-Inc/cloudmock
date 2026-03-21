# Data Firehose

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: Firehose_20150804.<Action>`)
**Service Name:** `firehose`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateDeliveryStream` | Creates a delivery stream with S3, Redshift, or OpenSearch destination |
| `DeleteDeliveryStream` | Deletes a delivery stream |
| `DescribeDeliveryStream` | Returns stream metadata and configuration |
| `ListDeliveryStreams` | Returns all delivery stream names |
| `PutRecord` | Writes a single record to a delivery stream |
| `PutRecordBatch` | Writes up to 500 records in one call |
| `UpdateDestination` | Updates the destination configuration |
| `TagDeliveryStream` | Adds tags to a stream |
| `UntagDeliveryStream` | Removes tags |
| `ListTagsForDeliveryStream` | Returns tags for a stream |

## Examples

### AWS CLI

```bash
# Create a delivery stream with S3 destination
aws firehose create-delivery-stream \
  --delivery-stream-name logs-to-s3 \
  --s3-destination-configuration '{
    "RoleARN": "arn:aws:iam::000000000000:role/firehose-role",
    "BucketARN": "arn:aws:s3:::my-logs-bucket",
    "Prefix": "firehose/"
  }'

# Put a record
aws firehose put-record \
  --delivery-stream-name logs-to-s3 \
  --record '{"Data":"eyJldmVudCI6InBhZ2V2aWV3In0="}'

# Describe stream
aws firehose describe-delivery-stream \
  --delivery-stream-name logs-to-s3

# List streams
aws firehose list-delivery-streams
```

### Python (boto3)

```python
import boto3, base64, json

firehose = boto3.client("firehose", endpoint_url="http://localhost:4566",
                        aws_access_key_id="test", aws_secret_access_key="test",
                        region_name="us-east-1")

# Create stream
firehose.create_delivery_stream(
    DeliveryStreamName="events",
    S3DestinationConfiguration={
        "RoleARN": "arn:aws:iam::000000000000:role/fh-role",
        "BucketARN": "arn:aws:s3:::events-bucket",
    },
)

# Write records
records = [
    {"Data": json.dumps({"event": "signup", "userId": "u-1"})},
    {"Data": json.dumps({"event": "login",  "userId": "u-2"})},
]
firehose.put_record_batch(DeliveryStreamName="events", Records=records)

# Describe
stream = firehose.describe_delivery_stream(DeliveryStreamName="events")
print(stream["DeliveryStreamDescription"]["DeliveryStreamStatus"])
```

## Notes

- Records are stored in memory. Delivery to S3 or other destinations is not performed.
- Buffering and compression settings are accepted but have no effect.
- Dynamic partitioning and data transformation via Lambda are not implemented.
- `PutRecordBatch` returns all records as successfully written.
