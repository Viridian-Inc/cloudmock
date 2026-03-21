# c7n-cloudmock

Cloud Custodian plugin for the cloudmock AWS emulator. Auto-configures boto3 endpoints and adds cloudmock-specific resources, actions, and filters.

## Installation

```bash
cd custodian/plugin
pip install -e .
```

This registers the plugin via the `custodian.plugins` entry point. Cloud Custodian loads it automatically on startup.

## Configuration

| Environment Variable        | Default                  | Description                          |
|-----------------------------|--------------------------|--------------------------------------|
| `CLOUDMOCK_ENDPOINT`        | `http://localhost:4566`  | AWS service endpoint (overrides boto3 default) |
| `CLOUDMOCK_ADMIN_ENDPOINT`  | `http://localhost:4599`  | cloudmock admin API for management   |
| `AWS_ENDPOINT_URL`          | set by plugin            | boto3 endpoint; plugin sets this if unset |
| `AWS_ACCESS_KEY_ID`         | `test`                   | set by plugin if unset               |
| `AWS_SECRET_ACCESS_KEY`     | `test`                   | set by plugin if unset               |
| `AWS_DEFAULT_REGION`        | `us-east-1`              | set by plugin if unset               |

## Resources

### `cloudmock-service`

Queries the cloudmock admin API for service status.

```yaml
policies:
  - name: list-services
    resource: cloudmock-service
```

### `cloudmock-request`

Queries the cloudmock request log (up to 1000 entries).

```yaml
policies:
  - name: list-requests
    resource: cloudmock-request
```

## Actions

### `cloudmock-reset`

Resets cloudmock state. Optionally scoped to a single service.

```yaml
actions:
  - type: cloudmock-reset
    service: s3       # omit to reset all services
```

### `cloudmock-seed`

Seeds cloudmock with data from a JSON file.

```yaml
actions:
  - type: cloudmock-seed
    file: fixtures/seed.json
```

### `cloudmock-snapshot`

Exports the current cloudmock state to a JSON file.

```yaml
actions:
  - type: cloudmock-snapshot
    output: cloudmock-state.json
```

## Filters

### `cloudmock-tier`

Filters `cloudmock-service` resources by emulation tier.

- **Tier 1** — fully emulated services (s3, dynamodb, sqs, sns, lambda, iam, sts, cognito, apigateway, cloudformation, cloudwatch, logs, events, states, secretsmanager, kms, ssm, route53, ecr, ecs, rds, ses, kinesis, firehose, ec2, monitoring)
- **Tier 2** — all other (partially emulated) services

```yaml
policies:
  - name: tier1-services
    resource: cloudmock-service
    filters:
      - type: cloudmock-tier
        tier: 1
```

## Running Tests

```bash
cd custodian/plugin
pip install -e .
python -m pytest tests/
```
