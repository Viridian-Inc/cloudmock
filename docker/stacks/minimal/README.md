# Minimal Stack

Just CloudMock. The simplest possible starting point.

## Start

```bash
docker compose up
```

## Use

Point any AWS SDK at `localhost:4566`:

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

# Create a bucket
aws s3 mb s3://my-bucket

# Put an item in DynamoDB
aws dynamodb create-table \
  --table-name my-table \
  --attribute-definitions AttributeName=id,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST
```

Open DevTools at [http://localhost:4500](http://localhost:4500) to inspect requests, traces, and service topology.
