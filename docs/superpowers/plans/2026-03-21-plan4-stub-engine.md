# Stub Engine + Tier 2 Services Implementation Plan

> **For agentic workers:** This is a retrospective document. All tasks listed below were already executed. Checkboxes are marked completed.

**Goal:** Build a generic stub engine that can emulate any AWS service from a declarative model definition, then register 74 Tier 2 AWS services using that engine. This expands cloudmock's service coverage from 24 Tier 1 services to 98 total services.

**Status:** COMPLETED

---

## Overview

Plan 4 introduced the stub engine — a runtime interpreter that turns a `ServiceModel` struct into a fully functional AWS mock service. Instead of writing service-specific Go code for every one of the 74 Tier 2 services, each service is expressed as a data model and the engine handles CRUD routing, ID generation, ARN construction, and tagging automatically.

The 74 Tier 2 services support basic CRUD operations (create, describe/list, delete, update where applicable) with in-memory storage. No business logic is executed — these services exist to accept SDK calls without errors, enabling infrastructure tests that don't depend on specific service behavior.

---

## Chunk 1: Stub Engine Core

### Task 1: Service Model Types (`pkg/stub/model.go`)

- [x] Define `ServiceModel` struct with fields: `ServiceName`, `Protocol`, `TargetPrefix`, `Actions`, `ResourceTypes`
- [x] Define `Action` struct with fields: `Name`, `Type`, `ResourceType`, `InputFields`, `OutputFields`, `IdField`
- [x] Define `Field` struct with fields: `Name`, `Type`, `Required`
- [x] Define `ResourceType` struct with fields: `Name`, `IdField`, `ArnPattern`, `Fields`
- [x] Support action types: `create`, `describe`, `list`, `delete`, `update`, `tag`, `untag`, `listTags`, `other`
- [x] Support protocols: `json`, `rest-json`, `query`, `rest-xml`

**Files created:**
- `pkg/stub/model.go`

### Task 2: Thread-Safe Resource Store (`pkg/stub/resource.go`)

- [x] Implement `ResourceStore` with three-level map: `resourceType -> id -> fields`
- [x] Implement `tags` map: `resourceArn -> map[string]string`
- [x] Implement `counters` map for sequential ID generation
- [x] Implement `nextID(prefix)` generating deterministic hex IDs (e.g., `vpc-00000001`)
- [x] Implement `Create(resourceType, idPrefix, fields)` — stores resource, returns generated ID
- [x] Implement `Get(resourceType, id)` — returns a copy of resource fields
- [x] Implement `Delete(resourceType, id)` — returns error if not found
- [x] Implement `List(resourceType)` — returns all resources of a type
- [x] Implement `Update(resourceType, id, updates)` — merges fields into existing resource
- [x] Implement `Tag(arn, tags)`, `Untag(arn, keys)`, `ListTags(arn)` — ARN-keyed tag store
- [x] Implement `BuildARN(pattern, region, account, id)` — substitutes `{region}`, `{account}`, `{id}` placeholders
- [x] All methods use `sync.RWMutex` for thread safety

**Files created:**
- `pkg/stub/resource.go`

### Task 3: Stub Engine (`pkg/stub/engine.go`)

- [x] Implement `StubService` struct holding `model`, `store`, `accountID`, `region`
- [x] Implement `NewStubService(model, accountID, region)` constructor
- [x] Implement `service.Service` interface: `Name()`, `Actions()`, `HealthCheck()`, `HandleRequest()`
- [x] Implement `HandleRequest()` — dispatches to handler by `action.Type`
- [x] Implement `handleCreate()` — generates ID and ARN, stores resource, echoes output fields
- [x] Implement `handleDescribe()` — looks up resource by ID field
- [x] Implement `handleList()` — returns all resources under a pluralized key
- [x] Implement `handleDelete()` — removes resource, returns 404 if missing
- [x] Implement `handleUpdate()` — merges fields, returns updated resource
- [x] Implement `handleTag()`, `handleUntag()`, `handleListTags()` — ARN-based tag operations
- [x] Implement `parseInput()` — supports both JSON and form-encoded (query/XML protocol) request bodies
- [x] Implement `extractID()` — pulls resource identifier from params using `action.IdField`
- [x] Validate required input fields before dispatching
- [x] Return correct `ResponseFormat` (JSON vs XML) based on `model.Protocol`
- [x] Implement `extractTags()` and `extractTagKeys()` helpers for both list and map tag formats

**Files created:**
- `pkg/stub/engine.go`

### Task 4: Stub Registry (`pkg/stub/registry.go`)

- [x] Implement `StubRegistry` with `models` map keyed by `ServiceName`
- [x] Implement `Register(model)` — adds model to registry
- [x] Implement `CreateService(serviceName, accountID, region)` — instantiates `StubService` from registered model
- [x] Implement `ListServices()` — returns names of all registered models

**Files created:**
- `pkg/stub/registry.go`

---

## Chunk 2: Tier 2 Service Catalog

### Task 5: Service Catalog (`services/stubs/catalog.go`)

The catalog defines all 74 Tier 2 service models using constructor helpers that minimize repetition.

**Helper functions:**
- [x] `f(name, typ, required)` — creates a `Field`
- [x] `reqStr(name)` / `optStr(name)` — required/optional string fields
- [x] `createAction(name, resType, idField, in, out)` — creates a create-type action
- [x] `describeAction(name, resType, idField)` — creates a describe-type action
- [x] `listAction(name, resType)` — creates a list-type action
- [x] `deleteAction(name, resType, idField)` — creates a delete-type action
- [x] `updateAction(name, resType, idField, extra)` — creates an update-type action
- [x] `otherAction(name, resType)` — creates an other-type action (returns empty success)
- [x] `rt(name, idField, arnPattern, fields)` — creates a `ResourceType`

**Query protocol services (11 services):**
- [x] Auto Scaling (`autoscaling`)
- [x] ELB/ALB (`elasticloadbalancing`)
- [x] Elastic Beanstalk (`elasticbeanstalk`)
- [x] ElastiCache (`elasticache`)
- [x] Redshift (`redshift`)
- [x] Neptune (`neptune`)
- [x] Elasticsearch (`es`)
- [x] EMR (`elasticmapreduce`)
- [x] EC2 (`ec2`)
- [x] Shield (`shield`)
- [x] WAF Regional (`waf-regional`)
- [x] DocumentDB (`docdb`)

**JSON protocol services (40+ services):**
- [x] ACM, ACM PCA, AppConfig, Application Auto Scaling, Athena, Backup, CodeBuild, CodeCommit, CodeDeploy, CodePipeline, CodeConnections, Config, Cost Explorer, DMS, Glue, Identity Store, Lake Formation, Lex (v2), Location, Macie, MediaConvert, MediaLive, MQ, MSK, OpenSearch, Organizations, RAM, Rekognition, Robomaker, SageMaker, SecurityHub, Service Catalog, ServiceDiscovery, SFN (aliases), SNS Platform, SNS Subscriptions, SSO, Textract, Transfer Family, WorkSpaces, X-Ray

**REST-JSON protocol services:**
- [x] Amplify, AppMesh, AppRunner, AppStream, Batch, Cloud9, CloudTrail, CodeArtifact, DataSync, Device Farm, Elastic Inference, Forecast, FSx, GameLift, Global Accelerator, Greengrass, GuardDuty, Inspector, IoT, IoT Analytics, IoT Events, Lex (v1), Lightsail, Network Manager, OpsWorks, Pinpoint, Polly, Resource Groups, Sagemaker (aliases), Service Quotas, Transcribe, Translate, VPC Lattice, WAFv2, Well-Architected Tool, WorkMail

**REST-XML protocol services:**
- [x] CloudFront

**Files created:**
- `services/stubs/catalog.go`

### Task 6: Service Registration (`services/stubs/register.go`)

- [x] Implement `RegisterAll(registry, accountID, region)` — iterates all models from `AllModels()`, registers each as a `StubService` with the gateway registry
- [x] Wire `RegisterAll` into gateway startup (`cmd/gateway/main.go`)

**Files created:**
- `services/stubs/register.go`

---

## Verification

- [x] All 74 stub services load at gateway startup without error
- [x] Total service count reaches 98 (24 Tier 1 + 74 Tier 2)
- [x] Stub service `Create` → `Describe` → `Delete` lifecycle works end-to-end
- [x] Tag operations (tag, untag, listTags) work via ARN lookup
- [x] JSON and form-encoded request bodies both parse correctly
- [x] Engine tests pass (`pkg/stub/engine_test.go`)
- [x] Catalog tests pass (`services/stubs/catalog_test.go`)
