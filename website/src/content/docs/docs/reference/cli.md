---
title: CLI Reference
description: cmk commands — start, stop, status, bench, and more
---

# CLI Reference

`cmk` is the CloudMock companion CLI. It manages the CloudMock server and provides developer tools.

## Commands

### cmk start

Start the CloudMock gateway, admin API, and DevTools dashboard.

```bash
cmk start                    # default config
cmk start --profile full     # all 99 services
cmk start --port 5566        # custom port
```

### cmk stop

Stop a running CloudMock instance.

```bash
cmk stop
```

### cmk status

Show the running instance's port, service count, and uptime.

```bash
cmk status
```

### cmk bench

Run benchmarks against CloudMock and optionally compare with LocalStack and Moto. Outputs a copy-pasteable markdown table.

```bash
cmk bench                        # 50 connections, 10s per operation
cmk bench -c 200 -d 30s          # heavy: 200 connections, 30s each
cmk bench --skip-localstack       # CloudMock only
cmk bench --skip-moto             # skip Moto comparison
```

**What it does:**
1. Starts CloudMock in test mode on a random port
2. Seeds benchmark data (DynamoDB table + item, SQS queue, S3 bucket, SNS topic, KMS key)
3. Runs 8 operations: DynamoDB GetItem/PutItem, SQS SendMessage, S3 PutObject/GetObject, SNS Publish, STS GetCallerIdentity, KMS Encrypt
4. If LocalStack is installed (Docker image present), starts it and runs the same operations
5. If Moto is installed (`pip install moto[server]`), starts it and runs the same operations
6. Prints a markdown comparison table with req/s per target

**Example output:**

```
| Operation | CloudMock | LocalStack | vs LocalStack |
|---|---|---|---|
| **DynamoDB GetItem** | **210,000** | **791** | 265x |
| **DynamoDB PutItem** | **195,000** | **742** | 263x |
| **SQS SendMessage** | **205,000** | **1,178** | 174x |
| **S3 PutObject (1KB)** | **165,000** | **1,795** | 92x |
| **S3 GetObject (1KB)** | **198,000** | **1,240** | 160x |
| **SNS Publish** | **190,000** | **1,231** | 154x |
| **STS GetCallerIdentity** | **180,000** | **1,229** | 146x |
| **KMS Encrypt** | **195,000** | **1,168** | 167x |
| **Geometric mean** | **192,000** | **1,007** | **191x** |
```

Users trust benchmarks they run on their own hardware. Share the table in issues, Slack, or your team's decision doc.

### cmk logs

Tail CloudMock logs.

```bash
cmk logs           # follow mode
cmk logs --lines 50  # last 50 lines
```

### cmk config

Show the current CloudMock configuration.

```bash
cmk config
```

### cmk version

Print the cmk version.

```bash
cmk version
```

## State Auto-Load

CloudMock automatically loads state from `.cloudmock/state.json` or `.cloudmock/seed-tables.json` in the working directory if present — no `--state` flag needed.

```bash
# Commit shared test fixtures to git
mkdir -p .cloudmock
cloudmock --state .cloudmock/state.json  # first time: creates the file
# Edit your test, create resources...
# On shutdown, state is saved back to .cloudmock/state.json

# Next startup: auto-loaded
cloudmock  # detects .cloudmock/state.json, loads it
```

Teams commit `.cloudmock/state.json` to git so every developer starts with the same baseline.
