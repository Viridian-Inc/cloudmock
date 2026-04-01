# Phase 1: Core Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every operation listed in `docs/compatibility-matrix.md` for all 25 Tier 1 AWS services returns correct responses with correct error codes. Comprehensive test suite. `make test-all` passes with zero failures.

**Architecture:** Extend existing per-service `service_test.go` files following established patterns (Go `testing` + `httptest`). Add `make test-all` target. Prioritize services with weakest coverage first. Add integration tests for cross-service workflows. Update compatibility matrix with AppSync promotion.

**Tech Stack:** Go 1.26, standard `testing` package, `httptest`, existing gateway/routing infrastructure

**Spec:** `docs/superpowers/specs/2026-03-31-v1-saas-launch-design.md` Phase 1

**Baseline:** ~1,468 test functions across all service test files. Target: 1,600+ after expansion.

**Note:** All service test files are named `service_test.go` (not `store_test.go` as the spec incorrectly states).

---

## File Structure

```
Makefile                                    # Add test-all target
docs/compatibility-matrix.md                # Promote AppSync to Tier 1
services/s3/service_test.go                 # Expand (currently 3 tests -- weakest)
services/sts/service_test.go                # Expand (currently 6 tests)
services/ses/service_test.go                # Expand (currently 6 tests)
services/route53/service_test.go            # Expand (currently 6 tests)
services/sqs/service_test.go                # Expand (currently 9 tests)
services/sns/service_test.go                # Expand (currently 8 tests)
services/eventbridge/service_test.go        # Expand (currently 6 tests)
services/apigateway/service_test.go         # Expand (currently 6 tests)
services/firehose/service_test.go           # Expand (currently 7 tests)
services/kinesis/service_test.go            # Expand (currently 8 tests)
services/ssm/service_test.go               # Expand (currently 8 tests)
services/secretsmanager/service_test.go     # Expand (currently 9 tests)
services/cloudwatch/service_test.go         # Expand (currently 8 tests)
services/cloudwatchlogs/service_test.go     # Expand (currently 8 tests)
services/cloudformation/service_test.go     # Expand (currently 9 tests)
services/stepfunctions/service_test.go      # Expand (currently 11 tests)
services/cognito/service_test.go            # Expand (currently 8 tests)
services/ecr/service_test.go               # Expand (currently 11 tests)
services/ecs/service_test.go               # Expand (currently 8 tests)
services/rds/service_test.go               # Expand (currently 8 tests)
services/lambda/service_test.go             # Expand (currently 11 tests)
services/kms/service_test.go               # Expand (currently 14 tests)
services/iam/service_test.go               # Expand (currently 10 tests)
services/dynamodb/service_test.go           # Verify coverage (currently 26 tests)
services/appsync/service_test.go            # Normalize to gateway pattern
tests/integration/crossservice_test.go      # Expand integration tests
```

---

## Task 1: Add `make test-all` target and verify baseline

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Read existing Makefile test targets**

Read `Makefile` to understand what `test`, `test-unit`, `test-integration`, and `test-race` do. Note: `test-integration` runs `go test ./pkg/dataplane/...` (dataplane tests), NOT `tests/integration/`. The main `test` target (`go test ./...`) already includes `tests/integration/` since `./...` is recursive.

- [ ] **Step 2: Add test-all target to Makefile**

Add after the existing `test-race` target:

```makefile
.PHONY: test-all
test-all: ## Run all tests: unit + dataplane + integration
	go test -v -cover -count=1 ./...
	go test -v -cover -count=1 ./pkg/dataplane/...
```

This runs the full recursive test suite plus the explicit dataplane integration tests.

- [ ] **Step 3: Run `make test-all` to establish baseline**

Run: `make test-all`
Expected: All existing tests pass. Note any failures -- these are pre-existing bugs to fix before proceeding.

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "build: add make test-all target (unit + dataplane + integration)"
```

---

## Task 2: Promote AppSync to Tier 1 in compatibility matrix

**Files:**
- Modify: `docs/compatibility-matrix.md`

- [ ] **Step 1: Read current AppSync implementation**

Read `services/appsync/store.go` to identify all implemented operations. The actual implementation supports ~27 operations (APIs, data sources, resolvers, functions, API keys, tagging) -- far more than the 4 listed in the current Tier 2 entry.

- [ ] **Step 2: Move AppSync from Tier 2 to Tier 1**

Expand the operations list to match what is actually implemented:
- GraphQL APIs: CreateGraphqlApi, GetGraphqlApi, ListGraphqlApis, UpdateGraphqlApi, DeleteGraphqlApi
- Data Sources: CreateDataSource, GetDataSource, ListDataSources, UpdateDataSource, DeleteDataSource
- Resolvers: CreateResolver, GetResolver, ListResolvers, UpdateResolver, DeleteResolver
- Functions: CreateFunction, GetFunction, ListFunctions, UpdateFunction, DeleteFunction
- API Keys: CreateApiKey, ListApiKeys, UpdateApiKey, DeleteApiKey
- Tags: TagResource, UntagResource, ListTagsForResource

- [ ] **Step 3: Update Tier counts**

Change header counts to: "25 Tier 1" and "73 Tier 2" (previously 24 + 74).

- [ ] **Step 4: Commit**

```bash
git add docs/compatibility-matrix.md
git commit -m "docs: promote AppSync to Tier 1 with full operation list (25 Tier 1 + 73 Tier 2)"
```

---

## Task 3: Expand S3 tests (currently 3 tests -- weakest coverage)

**Files:**
- Modify: `services/s3/service_test.go`

- [ ] **Step 1: Read existing S3 tests and store.go**

Read `services/s3/service_test.go` and `services/s3/store.go` to understand current coverage. S3 has the most operations (22) but only 3 tests -- the largest gap of any Tier 1 service. Also check if there are additional test files (e.g., `multipart_test.go`).

- [ ] **Step 2: Write tests for all compatibility matrix operations**

The compatibility matrix lists 10 S3 operations. Cover each with at least one positive and one error test:
- CreateBucket / HeadBucket / ListBuckets / DeleteBucket
- PutObject / GetObject / HeadObject / DeleteObject / ListObjects / ListObjectsV2
- CopyObject
- Multipart: CreateMultipartUpload, UploadPart, CompleteMultipartUpload, AbortMultipartUpload
- Versioning: PutBucketVersioning, GetBucketVersioning, ListObjectVersions
- Error cases: NoSuchBucket, NoSuchKey, BucketAlreadyExists, BucketNotEmpty

Follow the existing `s3Req()` helper pattern for building authenticated requests.

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/s3/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/s3/service_test.go
git commit -m "test(s3): expand coverage for all operations + error codes"
```

---

## Task 4: Expand STS tests (currently 6 tests)

**Files:**
- Modify: `services/sts/service_test.go`

- [ ] **Step 1: Read existing STS tests and store.go**

Read `services/sts/service_test.go` and `services/sts/store.go` to understand current coverage and available operations.

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- GetCallerIdentity -- verify account ID, ARN format
- AssumeRole -- valid role, invalid role, expired session
- GetSessionToken -- token generation, expiry
- Error cases: MalformedPolicyDocument, PackedPolicyTooLarge, RegionDisabledException

Each test follows the existing pattern: build request with helper, send to gateway, verify response status and body.

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/sts/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/sts/service_test.go
git commit -m "test(sts): expand test coverage for all operations + error codes"
```

---

## Task 5: Expand SES tests (currently 6 tests)

**Files:**
- Modify: `services/ses/service_test.go`

- [ ] **Step 1: Read existing SES tests and store.go**

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- SendEmail -- with/without template, HTML + text body
- VerifyEmailIdentity -- verify state transitions
- ListIdentities -- pagination
- CreateTemplate / GetTemplate / DeleteTemplate
- SendTemplatedEmail
- Error cases: MessageRejected, MailFromDomainNotVerified, TemplateDoesNotExist

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/ses/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/ses/service_test.go
git commit -m "test(ses): expand test coverage for all operations + error codes"
```

---

## Task 6: Expand Route 53 tests (currently 6 tests)

**Files:**
- Modify: `services/route53/service_test.go`

- [ ] **Step 1: Read existing tests and store.go**

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- CreateHostedZone / GetHostedZone / ListHostedZones / DeleteHostedZone
- ChangeResourceRecordSets -- A, AAAA, CNAME, MX, TXT record types
- ListResourceRecordSets -- pagination
- Error cases: HostedZoneNotFound, InvalidChangeBatch, NoSuchHostedZone, HostedZoneAlreadyExists

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/route53/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/route53/service_test.go
git commit -m "test(route53): expand test coverage for record operations + error codes"
```

---

## Task 7: Expand SQS tests (currently 9 tests)

**Files:**
- Modify: `services/sqs/service_test.go`

- [ ] **Step 1: Read existing tests and store.go**

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- FIFO queue behavior (MessageDeduplicationId, MessageGroupId)
- Dead-letter queue redrive
- ChangeMessageVisibility / ChangeMessageVisibilityBatch
- Batch operations: SendMessageBatch, DeleteMessageBatch
- Purge queue
- Long polling (WaitTimeSeconds)
- Error cases: QueueDoesNotExist, ReceiptHandleIsInvalid, EmptyBatchRequest, TooManyEntriesInBatchRequest

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/sqs/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/sqs/service_test.go
git commit -m "test(sqs): expand coverage for FIFO, DLQ, batch, error codes"
```

---

## Task 8: Expand SNS tests (currently 8 tests)

**Files:**
- Modify: `services/sns/service_test.go`

- [ ] **Step 1: Read existing tests and store.go**

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- Subscribe with filter policies
- Publish with MessageAttributes
- ListSubscriptionsByTopic -- pagination
- ConfirmSubscription
- SetSubscriptionAttributes / GetSubscriptionAttributes
- Error cases: TopicNotFound, SubscriptionNotFound, InvalidParameter, AuthorizationError

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/sns/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/sns/service_test.go
git commit -m "test(sns): expand coverage for filters, attributes, error codes"
```

---

## Task 9: Expand EventBridge tests (currently 6 tests)

**Files:**
- Modify: `services/eventbridge/service_test.go`

- [ ] **Step 1: Read existing tests and store.go**

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- PutEvents with detail-type matching
- Rule patterns (prefix, numeric, exists)
- ListRules / DescribeRule
- PutTargets / RemoveTargets / ListTargetsByRule
- Enable/Disable rule
- Error cases: ResourceNotFoundException, ManagedRuleException

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/eventbridge/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/eventbridge/service_test.go
git commit -m "test(eventbridge): expand coverage for rules, targets, patterns"
```

---

## Task 10: Expand API Gateway tests (currently 6 tests)

**Files:**
- Modify: `services/apigateway/service_test.go`

- [ ] **Step 1: Read existing tests and store.go**

- [ ] **Step 2: Write tests for missing operations**

Focus on:
- REST API lifecycle: Create/Get/List/Delete
- Resources: CreateResource / GetResource / GetResources
- Methods: PutMethod / GetMethod / PutMethodResponse
- Integrations: PutIntegration / GetIntegration
- Deployments: CreateDeployment / GetDeployment
- Stages: CreateStage / GetStage
- Error cases: NotFoundException, ConflictException, BadRequestException

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/apigateway/`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add services/apigateway/service_test.go
git commit -m "test(apigateway): expand coverage for resources, methods, deployments"
```

---

## Task 11: Expand remaining Tier 1 services (batch, group A)

For each of these 8 services: read existing tests, identify gaps against compatibility matrix, write tests for missing operations and error codes, run, commit separately.

**Per-service minimum:** Each compatibility matrix operation must have at least one positive test and one error test.

- [ ] **Firehose** (`services/firehose/service_test.go`) -- CreateDeliveryStream, PutRecord, PutRecordBatch, DescribeDeliveryStream, ListDeliveryStreams, DeleteDeliveryStream. S3 destination config. Error codes: ResourceNotFoundException, InvalidArgumentException.

- [ ] **Kinesis** (`services/kinesis/service_test.go`) -- CreateStream, PutRecord, PutRecords, GetShardIterator, GetRecords, DescribeStream, ListStreams, MergeShards, SplitShard. Error codes: ResourceNotFoundException, LimitExceededException.

- [ ] **SSM** (`services/ssm/service_test.go`) -- GetParameter, PutParameter (String/SecureString/StringList types), GetParametersByPath, DeleteParameter, DescribeParameters. Error codes: ParameterNotFound, ParameterAlreadyExists.

- [ ] **Secrets Manager** (`services/secretsmanager/service_test.go`) -- CreateSecret, GetSecretValue (by version/stage), PutSecretValue, UpdateSecret, DeleteSecret, RotateSecret config, ListSecrets. Error codes: ResourceNotFoundException, ResourceExistsException.

- [ ] **CloudWatch** (`services/cloudwatch/service_test.go`) -- PutMetricData, GetMetricStatistics with periods/statistics, ListMetrics, PutMetricAlarm, DescribeAlarms, DeleteAlarms. Error codes: ResourceNotFound, InvalidParameterValue.

- [ ] **CloudWatch Logs** (`services/cloudwatchlogs/service_test.go`) -- CreateLogGroup/Stream, PutLogEvents sequencing (sequenceToken), GetLogEvents, FilterLogEvents patterns, DescribeLogGroups, DeleteLogGroup. Error codes: ResourceAlreadyExistsException, ResourceNotFoundException.

- [ ] **CloudFormation** (`services/cloudformation/service_test.go`) -- CreateStack, DescribeStacks, ListStacks, DeleteStack, ValidateTemplate, DescribeStackEvents, DescribeStackResources. Error codes: AlreadyExistsException, InsufficientCapabilitiesException.

- [ ] **Step Functions** (`services/stepfunctions/service_test.go`) -- CreateStateMachine, DescribeStateMachine, ListStateMachines, StartExecution, DescribeExecution, ListExecutions, StopExecution, DeleteStateMachine. Error codes: StateMachineDoesNotExist, ExecutionDoesNotExist.

---

## Task 12: Expand remaining Tier 1 services (batch, group B)

- [ ] **Cognito** (`services/cognito/service_test.go`) -- CreateUserPool, DescribeUserPool, ListUserPools, SignUp, InitiateAuth (USER_PASSWORD_AUTH), AdminCreateUser, AdminGetUser. Error codes: ResourceNotFoundException, UsernameExistsException, NotAuthorizedException.

- [ ] **ECR** (`services/ecr/service_test.go`) -- CreateRepository, DescribeRepositories, ListImages, PutImage, BatchGetImage, DeleteRepository, GetAuthorizationToken. Error codes: RepositoryNotFoundException, ImageAlreadyExistsException.

- [ ] **ECS** (`services/ecs/service_test.go`) -- CreateCluster, DescribeClusters, ListClusters, RegisterTaskDefinition, CreateService, DescribeServices, UpdateService, RunTask, DescribeTasks, DeleteService, DeleteCluster. Error codes: ClusterNotFoundException, ServiceNotFoundException.

- [ ] **RDS** (`services/rds/service_test.go`) -- CreateDBInstance, DescribeDBInstances, ModifyDBInstance, DeleteDBInstance, CreateDBSnapshot, DescribeDBSnapshots, CreateDBCluster, DescribeDBClusters. Error codes: DBInstanceNotFound, DBInstanceAlreadyExists.

- [ ] **Lambda** (`services/lambda/service_test.go`) -- CreateFunction, GetFunction, ListFunctions, UpdateFunctionCode, UpdateFunctionConfiguration, Invoke, CreateAlias, GetAlias, PublishLayerVersion. Error codes: ResourceNotFoundException, ResourceConflictException, InvalidParameterValueException.

- [ ] **KMS** (`services/kms/service_test.go`) -- CreateKey, DescribeKey, ListKeys, Encrypt/Decrypt roundtrip, GenerateDataKey, CreateAlias, ListAliases, ScheduleKeyDeletion, EnableKeyRotation. Error codes: NotFoundException, DisabledException, InvalidCiphertextException.

- [ ] **IAM** (`services/iam/service_test.go`) -- CreateUser, GetUser, ListUsers, CreateRole, AttachRolePolicy, DetachRolePolicy, CreatePolicy, GetPolicy. Error codes: EntityAlreadyExists, NoSuchEntity, MalformedPolicyDocument.

- [ ] **DynamoDB** (`services/dynamodb/service_test.go`) -- Verify all 12 compatibility matrix operations have positive + error tests. Current 26 tests is strong but verify: CreateTable, DeleteTable, PutItem, GetItem, UpdateItem, DeleteItem, Query, Scan, BatchGetItem, BatchWriteItem, TransactWriteItems, TransactGetItems. Error codes: ResourceNotFoundException, ConditionalCheckFailedException, ValidationException.

---

## Task 13: Normalize AppSync to gateway test pattern

**Files:**
- Modify: `services/appsync/service_test.go`

- [ ] **Step 1: Read current AppSync test pattern**

AppSync tests currently call `s.HandleRequest()` directly, bypassing the HTTP gateway (routing + middleware). All other Tier 1 services test through the gateway with `httptest`. The assertion library (testify vs raw Go) is a separate concern -- the key issue is that direct HandleRequest testing skips routing, auth middleware, and CORS handling.

- [ ] **Step 2: Rewrite tests to use gateway pattern**

Convert to build a gateway with `gateway.New(cfg, reg)` and send HTTP requests with proper REST URL paths:
- `POST /v1/apis` for CreateGraphqlApi
- `GET /v1/apis/{apiId}` for GetGraphqlApi
- etc.

Keep all existing test coverage, just change the harness from `s.HandleRequest()` to `httptest.NewRecorder()` + gateway.

- [ ] **Step 3: Run tests**

Run: `go test -v ./services/appsync/`
Expected: All tests pass with the new gateway-based pattern

- [ ] **Step 4: Commit**

```bash
git add services/appsync/service_test.go
git commit -m "test(appsync): normalize to HTTP gateway test pattern"
```

---

## Task 14: Expand cross-service integration tests

**Files:**
- Modify: `tests/integration/crossservice_test.go`

- [ ] **Step 1: Read existing integration tests**

Understand current S3->SQS, SNS->SQS, EventBridge->SQS patterns.

- [ ] **Step 2: Add Lambda invocation integration test**

Test: API Gateway -> Lambda invocation -> DynamoDB write. Verify the full chain works.

- [ ] **Step 3: Add CloudFormation provisioning test**

Test: CreateStack with a template that creates an S3 bucket + DynamoDB table. Verify resources exist after stack creation.

- [ ] **Step 4: Run integration tests**

Run: `go test -v ./tests/integration/`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add tests/integration/crossservice_test.go
git commit -m "test(integration): add Lambda invocation + CloudFormation provisioning chains"
```

---

## Task 15: Final verification

- [ ] **Step 1: Run full test suite**

Run: `make test-all`
Expected: Zero failures across all services

- [ ] **Step 2: Verify test count increased**

Run: `go test -v ./services/... 2>&1 | grep -c "^--- PASS"`
Expected: 1,600+ tests (up from ~1,468 baseline)

- [ ] **Step 3: Verify no ignored errors in Go code**

Run: `golangci-lint run --enable errcheck ./services/... ./pkg/...` (if golangci-lint is installed)
Or: `go vet ./services/... ./pkg/...`
Expected: No critical issues

- [ ] **Step 4: Run race detector**

Run: `make test-race`
Expected: Zero race conditions

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "test: Phase 1 complete — all 25 Tier 1 services verified (1,600+ tests)"
```

---

## Verification

1. `make test-all` -- zero failures
2. `make test-race` -- zero race conditions
3. Every operation in `docs/compatibility-matrix.md` for Tier 1 services has corresponding test coverage
4. AppSync listed as Tier 1 in compatibility matrix with full operation list
5. Test count: 1,600+ (up from ~1,468 baseline)
6. `go vet ./...` -- zero issues
