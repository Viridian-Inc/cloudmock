# Configuration Reference

cloudmock is configured via a YAML file (default: `cloudmock.yml` in the working directory) with environment variable overrides applied on top.

---

## Full `cloudmock.yml` Reference

```yaml
# AWS region to emulate
region: us-east-1

# Simulated AWS account ID (12 digits)
account_id: "000000000000"

# Service profile: minimal | standard | full | custom
# - minimal:  iam, sts, s3, dynamodb, sqs, sns, lambda, cloudwatch-logs
# - standard: all minimal services + rds, cloudformation, ec2, ecr, ecs,
#             secretsmanager, ssm, kinesis, firehose, events, stepfunctions, apigateway
# - full:     all supported services (98 total)
# - custom:   only the services listed under the `services` key below
profile: minimal

iam:
  # IAM enforcement mode
  # - enforce:      full IAM policy evaluation (default)
  # - authenticate: verify credentials only, skip policy checks
  # - none:         skip all auth (development only)
  mode: enforce

  # Root credentials accepted by all modes except `none`
  root_access_key: test
  root_secret_key: test

  # Optional path to a JSON file for seeding IAM users, roles, and policies
  # seed_file: /etc/cloudmock/iam-seed.json

persistence:
  # Persist in-memory state to disk on shutdown and restore on startup
  enabled: false
  # Directory for state snapshots (created if it does not exist)
  # path: /var/lib/cloudmock/data

gateway:
  # Port for the main AWS API endpoint
  port: 4566

dashboard:
  # Web UI for inspecting service state
  enabled: true
  port: 4500

admin:
  # Control-plane REST API (used by the `cloudmock` CLI)
  port: 4599

logging:
  # Log level: debug | info | warn | error
  level: info
  # Log format: text (human-readable) | json (structured)
  format: text

# Per-service overrides — used when profile: custom, or to change defaults
# for a specific service while using another profile.
#
# services:
#   s3:
#     enabled: true
#   lambda:
#     enabled: true
#     runtimes:
#       - nodejs20.x
#       - python3.12
#   dynamodb:
#     enabled: false   # disable one service from a named profile
```

---

## Environment Variables

All environment variables override the corresponding value in `cloudmock.yml`.

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_REGION` | `us-east-1` | AWS region to emulate |
| `CLOUDMOCK_IAM_MODE` | `enforce` | IAM mode: `enforce`, `authenticate`, or `none` |
| `CLOUDMOCK_PERSIST` | `false` | Enable persistence (`true`/`false`) |
| `CLOUDMOCK_PERSIST_PATH` | — | Directory for state snapshots |
| `CLOUDMOCK_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `CLOUDMOCK_SERVICES` | — | Comma-separated list of services to enable (overrides profile) |
| `CLOUDMOCK_PROFILE` | — | Service profile (set by `cloudmock start --profile`) |
| `CLOUDMOCK_ADMIN_ADDR` | `http://localhost:4599` | Address the CLI uses to reach the admin API |

### Example

```bash
CLOUDMOCK_REGION=eu-west-1 \
CLOUDMOCK_IAM_MODE=none \
CLOUDMOCK_LOG_LEVEL=debug \
./bin/cloudmock start
```

---

## Service Profiles

### `minimal`

Starts the smallest useful set of services:

```
iam, sts, s3, dynamodb, sqs, sns, lambda, cloudwatch-logs
```

Suitable for applications that use only core compute and storage services.

### `standard`

Starts all Tier 1 services that are commonly used in production stacks:

```
iam, sts, s3, dynamodb, sqs, sns, lambda, cloudwatch-logs,
rds, cloudformation, ec2, ecr, ecs, secretsmanager, ssm,
kinesis, firehose, events, stepfunctions, apigateway
```

### `full`

Starts all 98 supported services including all Tier 2 stubs.

### `custom`

Only the services explicitly listed under the `services` key are started:

```yaml
profile: custom
services:
  s3:
    enabled: true
  dynamodb:
    enabled: true
  sqs:
    enabled: true
```

You can also use `CLOUDMOCK_SERVICES` for a quick override without editing the config file:

```bash
CLOUDMOCK_SERVICES=s3,dynamodb,sqs cloudmock start
```

---

## IAM Configuration

### Modes

**`enforce`** (default) — Requests must include valid AWS Signature V4 credentials. The IAM engine evaluates every request against attached policies. Requests without an explicit Allow are denied.

**`authenticate`** — Credentials are validated (the access key must exist in the store) but policy evaluation is skipped. All authenticated requests succeed.

**`none`** — All authentication and authorization checks are bypassed. Useful for rapid prototyping but not safe for multi-user environments.

### Root credentials

The `root_access_key` and `root_secret_key` values define a superuser credential that bypasses all policy checks. The defaults are both `test`, matching the convention used by LocalStack and other emulators.

### IAM seed file

If `iam.seed_file` is set, cloudmock loads users, access keys, and policies from a JSON file at startup. Format:

```json
{
  "users": [
    {
      "name": "ci-user",
      "access_key_id": "AKIAIOSFODNN7EXAMPLE",
      "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "policies": [
        {
          "name": "AllowS3",
          "document": {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": ["s3:*"],
                "Resource": "*"
              }
            ]
          }
        }
      ]
    }
  ]
}
```

---

## Persistence

When `persistence.enabled: true`, cloudmock writes a state snapshot to `persistence.path` on clean shutdown and restores it on startup. The snapshot format is an internal JSON representation of each service's in-memory store.

```yaml
persistence:
  enabled: true
  path: /var/lib/cloudmock/data
```

State is not automatically synced during operation — only on shutdown. If the process is killed without a clean shutdown, the previous snapshot is loaded.

---

## Logging

```yaml
logging:
  level: debug   # debug | info | warn | error
  format: json   # text | json
```

JSON format is recommended when shipping logs to a centralized system:

```json
{"time":"2026-03-21T12:00:00Z","level":"INFO","msg":"request","service":"s3","action":"PutObject","status":200,"duration_ms":1}
```

Text format is easier to read in a terminal:

```
2026-03-21 12:00:00 INFO  s3 PutObject 200 1ms
```
