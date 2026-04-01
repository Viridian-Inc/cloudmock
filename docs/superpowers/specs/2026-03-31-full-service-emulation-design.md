# Full Behavioral Emulation for 73 CloudMock Services

**Date:** 2026-03-31
**Status:** In Progress (Phases 0-5 executing, Phase 6-7 pending)

## Summary

Upgraded all 73 recently-promoted AWS service mocks from CRUD stubs to full behavioral emulation. Each service now has domain-specific logic, cross-service integration via ServiceLocator, configurable async timing, and realistic error modes.

## Shared Infrastructure (Phase 0)

| Package | Purpose | File |
|---------|---------|------|
| `pkg/worker` | Background worker pool for periodic/deferred work | `pool.go` |
| `pkg/mocklog` | Writes mock execution logs to CloudWatch Logs via ServiceLocator | `writer.go` |
| `pkg/sqlparse` | Lightweight SQL validator for Athena/Redshift | `validator.go` |
| `pkg/testutil` | MockLocator for unit testing cross-service behavior | `mocklocator.go` |
| `pkg/gateway` | Event publisher — publishes `{svc}:ApiCall:{action}` events to bus | Modified `gateway.go` |

## Service Emulation Tiers

### Tier A: Compute/Orchestration (actual execution or orchestration)
- **autoscaling** — EC2 instance creation/termination via locator, capacity reconciliation
- **elasticloadbalancing** — Background health check goroutines, target state machine
- **codebuild** — Build phase progression, mock log generation
- **codepipeline** — Stage/action orchestration triggers CodeBuild/CodeDeploy
- **codedeploy** — Per-instance deployment tracking, rollback
- **eks** — Node groups create EC2 instances, OIDC issuer
- **batch** — Job queue scheduling, compute capacity tracking
- **swf** — Decision/activity task scheduling, workflow history
- **emr** — EC2 fleet creation, step execution with logs
- **glue** — Crawlers generate table schemas from S3, job logs

### Tier B: Data Services (query/data interfaces)
- **athena** — SQL validation against Glue catalog, mock result sets
- **redshift** — Internal schema tracking, SQL validation
- **opensearch/es** — Document indexing, basic search
- **timestreamwrite** — Time-series record storage, time-range queries

### Tier C: Security/Identity (policy evaluation, validation flows)
- **cloudtrail** — Event bus subscription, API call recording, LookupEvents
- **config** — Configuration recorder via event bus, compliance evaluation
- **organizations** — SCP policy evaluation, OU hierarchy enforcement
- **wafv2** — Rule evaluation (IP match, regex, rate-based), sampled requests
- **acm** — DNS validation flow, auto-validate in instant mode
- **acmpca** — CA chain building, certificate issuance with validity

### Tier D: Infrastructure (realistic resource modeling)
- **cloudfront** — Origin validation (S3/ELB must exist)
- **elasticache** — Cluster endpoint generation, node topology, failover
- **neptune/docdb** — Cluster + instance topology, primary/reader endpoints
- **dms** — Endpoint connectivity validation, replication counters

### Tier E: Application Services (event/schedule execution)
- **scheduler** — Schedule firing via worker pool, Lambda/SQS/SNS targets
- **pipes** — Source polling (SQS/DynamoDB), target invocation (Lambda)
- **appconfig** — Deployment strategy execution (linear/exponential rollout)
- **servicediscovery** — Health check integration, DNS management
- **sagemaker** — Training metrics/logs, InvokeEndpoint with mock predictions

### Tier F: Remaining 43 Services (enriched validation + realistic responses)
- Rich field validation, realistic AWS error codes
- Cross-service validation via ServiceLocator where applicable
- Proper constraint enforcement

## Architecture Decisions

1. **Feature-flagged via lifecycle.Config** — all behavioral enhancements gated on `lifecycle.Config.Enabled()`. Disabled by default = instant mode (current behavior). Enabled = full async simulation.

2. **Nil-guarded locator calls** — every cross-service call checks `if s.locator != nil`. Graceful degradation to CRUD stub when locator unavailable.

3. **No big-bang** — each service upgraded independently. Existing 2,147+ tests must pass after each upgrade.

4. **TDD** — new behavioral tests written before implementation.

## Metrics

| Metric | Value |
|--------|-------|
| Services upgraded | 73 |
| New shared packages | 4 |
| New behavioral files | ~30 (reconciler.go, healthcheck.go, phases.go, executor.go, recorder.go, evaluator.go, etc.) |
| Estimated new/modified lines | ~18,850 |
| Cross-service integrations | ~36 services use ServiceLocator |
| Event bus subscribers | CloudTrail, Config |
| Worker pool users | ELB, AutoScaling, Scheduler, Pipes, Config |
