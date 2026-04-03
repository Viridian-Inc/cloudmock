# Local Development

CloudMock replaces real AWS during local development. Your existing code works unchanged -- just set one environment variable.

## Basic Setup

```bash
# Terminal 1: Start CloudMock
cmk start

# Terminal 2: Run your app
export AWS_ENDPOINT_URL=http://localhost:4566
npm run dev   # or python app.py, go run ., etc.
```

## Project Configuration

Create `.cloudmock.yaml` in your project root for consistent settings across the team:

```yaml
# .cloudmock.yaml
profile: standard
region: us-east-1

gateway:
  port: 4566

dashboard:
  port: 4500

iac:
  path: ./infra
  env: dev
```

Check this into version control. Every developer gets the same setup.

## AWS Credentials

CloudMock accepts any AWS credentials by default. The simplest setup:

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_ENDPOINT_URL=http://localhost:4566
```

For IAM enforcement testing, configure real-looking credentials:

```yaml
# .cloudmock.yaml
iam:
  mode: enforce
  root_access_key: AKIAIOSFODNN7EXAMPLE
  root_secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

## Working with Infrastructure-as-Code

CloudMock auto-discovers Pulumi and Terraform projects to build its topology view.

### Pulumi

```bash
# CloudMock auto-discovers Pulumi.yaml in parent directories
cmk start
# Output: Discovered pulumi project at ./infra

# Deploy your Pulumi stack against CloudMock
export AWS_ENDPOINT_URL=http://localhost:4566
cd infra && pulumi up --stack dev
```

### Terraform

```bash
# Configure Terraform to use CloudMock
export AWS_ENDPOINT_URL=http://localhost:4566

# Or in your provider block:
terraform {
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
}

provider "aws" {
  region     = "us-east-1"
  access_key = "test"
  secret_key = "test"
  endpoints {
    s3       = "http://localhost:4566"
    dynamodb = "http://localhost:4566"
    sqs      = "http://localhost:4566"
    # ... all services use the same endpoint
  }
}
```

## Choosing a Profile

Start with the lightest profile that covers your services:

```bash
# Fast startup, covers most Lambda + DynamoDB + S3 workflows
cmk start --profile minimal

# Includes EC2, ECS, RDS, and other common services
cmk start --profile standard

# Everything -- all 100 services
cmk start --profile full
```

Or pick specific services:

```yaml
# .cloudmock.yaml
profile: custom
services:
  s3:
    enabled: true
  dynamodb:
    enabled: true
  sqs:
    enabled: true
  lambda:
    enabled: true
    runtimes:
      - nodejs20.x
      - python3.12
```

## Persistence

By default, data is in-memory and lost when CloudMock stops. Enable persistence for data that survives restarts:

```yaml
persistence:
  enabled: true
  path: ./.cloudmock-data
```

Add `.cloudmock-data/` to your `.gitignore`.

## Seeding Data

Pre-populate services with data on startup using AWS CLI scripts:

```bash
#!/bin/bash
# seed.sh -- run after cmk start
export AWS_ENDPOINT_URL=http://localhost:4566

# Create S3 bucket
aws s3 mb s3://my-app-uploads

# Create DynamoDB table
aws dynamodb create-table \
  --table-name users \
  --attribute-definitions AttributeName=userId,AttributeType=S \
  --key-schema AttributeName=userId,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

# Create SQS queue
aws sqs create-queue --queue-name order-processing

# Insert seed data
aws dynamodb put-item \
  --table-name users \
  --item '{"userId":{"S":"user-1"},"name":{"S":"Alice"},"email":{"S":"alice@example.com"}}'
```

## Docker Compose

Run CloudMock alongside your application:

```yaml
# docker-compose.yml
services:
  cloudmock:
    image: ghcr.io/neureaux/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
      - "4599:4599"
      - "4318:4318"
    environment:
      CLOUDMOCK_PROFILE: standard

  app:
    build: .
    depends_on:
      - cloudmock
    environment:
      AWS_ENDPOINT_URL: http://cloudmock:4566
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      AWS_REGION: us-east-1
      OTEL_EXPORTER_OTLP_ENDPOINT: http://cloudmock:4318
```

## Resetting State

Clear all data without restarting:

```bash
curl -X POST http://localhost:4599/api/reset
```

Or restart:

```bash
cmk stop && cmk start
```

## Tips

- **Check the Dashboard** (`localhost:4500`) after each API call to see request details, timing, and IAM evaluation results.
- **Use the topology view** to verify your IaC creates the architecture you expect.
- **Enable `debug` logging** for troubleshooting: `CLOUDMOCK_LOG_LEVEL=debug cmk start --fg`
- **Run in foreground** during development for inline logs: `cmk start --fg`
