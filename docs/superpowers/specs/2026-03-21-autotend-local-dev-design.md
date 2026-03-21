# Autotend Local Dev with cloudmock

**Date:** 2026-03-21
**Status:** Draft
**Location:** autotend-infra/local/

## Overview

Single-command local development for autotend using cloudmock as the AWS backend. Runs `pulumi up` against cloudmock to provision all infrastructure (40+ DynamoDB tables, Cognito pools, SQS queues, Lambda functions, VPC, etc.), starts local replacements for Tier 2 services (Apollo Server for AppSync, Postgres for Aurora, skip CloudFront/ACM/Backup), seeds test data, and starts all backend services.

## Command

```bash
cd autotend-infra/local && pnpm dev:local
```

## Architecture

```
cloudmock (:4566)  ←── pulumi up (provisions all AWS resources)
     ↑
     ├── BFF service (:8081)
     ├── API services (:3000)
     ├── Local GraphQL (:4000)  ← replaces AppSync
     ├── Local Postgres (:5432) ← replaces Aurora
     └── Mobile app / Web portals → connects to BFF + GQL
```

## AWS Services (20 total)

All hit cloudmock on port 4566:
- DynamoDB (40+ tables) — Tier 1, full emulation
- Lambda (~30 functions) — Tier 1, metadata + local execution
- Cognito (2 user pools) — Tier 1
- API Gateway — Tier 1
- SQS (3 queues) — Tier 1
- SNS — Tier 1
- SES — Tier 1
- S3 (4 buckets) — Tier 1
- EventBridge — Tier 1
- CloudWatch/Logs — Tier 1
- Secrets Manager — Tier 1
- IAM — Tier 1
- VPC/EC2 — Tier 1
- KMS — Tier 1
- RDS — Tier 1 (metadata; actual DB is local Postgres)
- Route 53 — Tier 1

## Local Replacements

| Production Service | Local Replacement | Reason |
|-------------------|-------------------|--------|
| AppSync | Apollo Server (:4000) | Tier 2 stub, need real GraphQL + subscriptions |
| CloudFront | Direct access / dev servers | Not needed locally |
| Aurora PostgreSQL | Local Postgres container (:5432) | Real DB needed for calendar service |
| ACM | Skipped | No HTTPS locally |
| AWS Backup | Skipped | No backups locally |

## Pulumi Local Stack

Uses the real AWS Pulumi provider pointed at cloudmock via endpoint overrides in `Pulumi.local.yaml`. The main infra modules are imported with conditional logic to skip CloudFront/ACM/Backup and replace AppSync.

## Seed Data

| Table | Data |
|-------|------|
| enterprise | 2 test enterprises |
| membership | 5 users per enterprise (admin, teacher, student) |
| resource | 3 classrooms per enterprise |
| session | 10 sample sessions |
| featureFlag | All flags enabled |
| Cognito users | admin/teacher/student with pre-confirmed status |
| Secrets Manager | Dummy LMS credentials |

## CORS

cloudmock gateway gets CORS middleware (enabled via `CLOUDMOCK_CORS=true`):
- Reflects request Origin
- Allows all AWS headers + x-api-key
- Handles OPTIONS preflight
- Max-age 86400

Local GraphQL server uses Express cors() allowing all localhost origins.

## Environment Variables

```
AWS_ENDPOINT_URL=http://localhost:4566
AWS_ACCESS_KEY_ID=test
AWS_SECRET_ACCESS_KEY=test
AWS_DEFAULT_REGION=us-east-1
COGNITO_USER_POOL_ID=<from pulumi output>
COGNITO_CLIENT_ID=<from pulumi output>
GRAPHQL_ENDPOINT=http://localhost:4000/graphql
ENVIRONMENT=local
LOCAL_DB=true
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/calendar_db
```

## File Structure

```
autotend-infra/
  local/
    dev.ts                    # Orchestrator (single command)
    config.ts                 # Shared config: endpoints, ports, env vars
    package.json              # Scripts: dev:local, seed, reset, pulumi:up
    tsconfig.json
    pulumi/
      Pulumi.yaml             # Pulumi project
      Pulumi.local.yaml       # Stack config with cloudmock endpoints
      index.ts                # Main — imports infra modules with overrides
      overrides.ts            # Skip CloudFront/ACM/Backup, replace AppSync
    graphql/
      server.ts               # Apollo Server with CORS, DynamoDB resolvers
      schema.graphql           # Linked from main AppSync schema
    seed/
      seed-data.ts             # DynamoDB test data
      cognito-users.ts         # Cognito test users
      secrets.ts               # Secrets Manager dummy data
```

## Orchestrator Steps

1. Check prerequisites (Docker, node, cloudmock)
2. Start cloudmock gateway (background, port 4566)
3. Wait for health check
4. Start Postgres container (port 5432)
5. Run `pulumi up --stack local --yes`
6. Run seed scripts
7. Start local GraphQL server (port 4000)
8. Start BFF service (port 8081)
9. Start API services (port 3000)
10. Print connection summary

Ctrl+C kills all child processes.
