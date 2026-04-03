# Data Pipeline Stack

S3 file ingestion → SQS notification → worker processing → DynamoDB results.

## Architecture

```
[uploader] --> S3 (data-ingestion bucket)
           --> SQS (ingest queue)
                    |
              [worker]
                    |
              DynamoDB (processed-records table)
```

## Services

| Service | Description |
|---------|-------------|
| CloudMock | AWS API emulation (port 4566) |
| CloudMock DevTools | Observability dashboard (port 4500) |
| uploader | Generates 5 sample records, uploads to S3, sends SQS notification |
| worker | Polls SQS, fetches each file from S3, transforms data, writes to DynamoDB |

## Start

```bash
docker compose up --build
```

Watch the logs — the uploader sends 5 records, the worker picks them up immediately and writes results to DynamoDB.

## Verify results

```bash
# See processed records in DynamoDB
aws dynamodb scan --table-name processed-records \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 \
  --no-sign-request

# Browse uploaded files in S3
aws s3 ls s3://data-ingestion/uploads/ --recursive \
  --endpoint-url http://localhost:4566 \
  --region us-east-1 \
  --no-sign-request
```

## Customizing

- **Real file ingestion**: mount a local directory into the uploader container and read from it
- **Different formats**: CSV, Parquet — extend the worker's `processMessage` function
- **Scale workers**: add `replicas: 3` under the `worker` service deploy config
- **Dead-letter queue**: add a DLQ in the `setup` step and configure it on the main queue
