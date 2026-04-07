# Core Tier 1 Services Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build 8 core Tier 1 services: STS, KMS, Secrets Manager, SSM Parameter Store, SQS, SNS, S3 object operations, and DynamoDB. Each service uses the existing service framework and registers with the gateway.

**Architecture:** Each service implements the `service.Service` interface from `pkg/service`. Services are registered in-process with the gateway's routing registry. Each has its own in-memory store, handlers, and test suite. AWS SDK integration tests validate API compatibility.

**Tech Stack:** Go 1.26, testify, encoding/xml, encoding/json, crypto (for KMS), github.com/mattn/go-sqlite3 (for DynamoDB)

**Spec:** `docs/superpowers/specs/2026-03-20-cloudmock-design.md`

**Depends on:** Plan 1 (Foundation) — completed

---

## Chunk 1: Auth & Security Services (STS, KMS, Secrets Manager)

### Task 1: STS Service

STS is simple but critical — it's used by every AWS SDK to validate credentials.

**Files:**
- Create: `services/sts/service.go`
- Create: `services/sts/service_test.go`
- Create: `services/sts/handlers.go`
- Create: `services/sts/store.go`

- [ ] **Step 1: Write test for GetCallerIdentity and AssumeRole**

`services/sts/service_test.go`:
```go
package sts_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	stssvc "github.com/Viridian-Inc/cloudmock/services/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGateway(t *testing.T) *gateway.Gateway {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(stssvc.New(cfg.AccountID))
	return gateway.New(cfg, reg)
}

func stsRequest(action string, params url.Values) *http.Request {
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2011-06-15")
	body := params.Encode()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=test/20260320/us-east-1/sts/aws4_request, SignedHeaders=host, Signature=abc")
	return r
}

func TestSTS_GetCallerIdentity(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, stsRequest("GetCallerIdentity", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "<Arn>")
	assert.Contains(t, body, "<Account>000000000000</Account>")
	assert.Contains(t, body, "<UserId>")
}

func TestSTS_AssumeRole(t *testing.T) {
	gw := newTestGateway(t)
	params := url.Values{
		"RoleArn":         {"arn:aws:iam::000000000000:role/TestRole"},
		"RoleSessionName": {"test-session"},
	}
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, stsRequest("AssumeRole", params))

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "<AccessKeyId>")
	assert.Contains(t, body, "<SecretAccessKey>")
	assert.Contains(t, body, "<SessionToken>")
	assert.Contains(t, body, "<AssumedRoleUser>")
}

func TestSTS_GetSessionToken(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, stsRequest("GetSessionToken", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "<AccessKeyId>")
	assert.Contains(t, body, "<SecretAccessKey>")
	assert.Contains(t, body, "<SessionToken>")
}
```

- [ ] **Step 2: Run test, verify fail**

Run: `go test ./services/sts/... -v`
Expected: FAIL

- [ ] **Step 3: Implement STS service**

`services/sts/store.go` — generates temporary credentials (random access key, secret, session token)

`services/sts/handlers.go` — XML response types for GetCallerIdentityResult, AssumeRoleResult, GetSessionTokenResult. Handlers parse form-encoded body, return XML responses.

`services/sts/service.go` — routes by Action query param: GetCallerIdentity, AssumeRole, GetSessionToken, GetFederationToken

Key: STS uses form-encoded POST with ?Action= parameter. Responses are XML with `<GetCallerIdentityResponse>` wrapper and `<RequestId>`.

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./services/sts/... -v`
Expected: PASS

- [ ] **Step 5: Run all tests**

Run: `go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add services/sts/
git commit -m "feat: add STS service — GetCallerIdentity, AssumeRole, GetSessionToken"
```

---

### Task 2: KMS Service

**Files:**
- Create: `services/kms/service.go`
- Create: `services/kms/service_test.go`
- Create: `services/kms/handlers.go`
- Create: `services/kms/store.go`
- Create: `services/kms/crypto.go`

- [ ] **Step 1: Write test for key creation, encrypt, decrypt**

`services/kms/service_test.go` — test through gateway:
1. `TestKMS_CreateKey` — POST with X-Amz-Target: TrentService.CreateKey. Verify 200, response contains KeyId, Arn, KeyState=Enabled
2. `TestKMS_EncryptDecrypt` — Create key, then Encrypt plaintext "hello world" with KeyId, get CiphertextBlob. Then Decrypt CiphertextBlob, verify Plaintext matches original.
3. `TestKMS_CreateAlias` — Create key, then CreateAlias with AliasName=alias/test and TargetKeyId.
4. `TestKMS_DescribeKey` — Create key, then DescribeKey with KeyId.

KMS uses JSON protocol with X-Amz-Target header (e.g., `TrentService.CreateKey`).

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement KMS**

`services/kms/crypto.go` — Software-based encrypt/decrypt using AES-256-GCM. Each key stores a random 32-byte AES key. Encrypt generates a random nonce, encrypts with AES-GCM, returns nonce+ciphertext as base64. Decrypt extracts nonce, decrypts.

`services/kms/store.go` — Key struct (KeyId, Arn, KeyState, Description, CreationDate, AESKey []byte, Aliases []string). Store with CRUD operations.

`services/kms/handlers.go` — JSON request/response types. Handlers for CreateKey, DescribeKey, Encrypt, Decrypt, CreateAlias, ListAliases, ListKeys, EnableKey, DisableKey, ScheduleKeyDeletion.

`services/kms/service.go` — Routes by X-Amz-Target: TrentService.<Action>. Name() returns "kms".

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add services/kms/
git commit -m "feat: add KMS service — key management, AES-GCM encrypt/decrypt, aliases"
```

---

### Task 3: Secrets Manager Service

**Files:**
- Create: `services/secretsmanager/service.go`
- Create: `services/secretsmanager/service_test.go`
- Create: `services/secretsmanager/handlers.go`
- Create: `services/secretsmanager/store.go`

- [ ] **Step 1: Write test for secret CRUD**

Tests through gateway:
1. `TestSecrets_CreateAndGet` — CreateSecret with Name and SecretString. GetSecretValue returns same string.
2. `TestSecrets_PutNewVersion` — CreateSecret, then PutSecretValue with new string. GetSecretValue returns updated.
3. `TestSecrets_Delete` — CreateSecret, DeleteSecret, GetSecretValue returns ResourceNotFoundException.
4. `TestSecrets_ListSecrets` — Create 2 secrets, ListSecrets returns both.
5. `TestSecrets_DescribeSecret` — CreateSecret, DescribeSecret returns metadata.

Secrets Manager uses JSON protocol with X-Amz-Target: secretsmanager.<Action>.

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement Secrets Manager**

`services/secretsmanager/store.go` — Secret struct (Name, ARN, SecretString, SecretBinary, VersionId, VersionStages, CreatedDate, Description, Tags). Store with CRUD + versioning (each PutSecretValue creates a new VersionId).

`services/secretsmanager/handlers.go` — JSON handlers: CreateSecret, GetSecretValue, PutSecretValue, DeleteSecret, RestoreSecret, DescribeSecret, ListSecrets, TagResource, UntagResource, UpdateSecret.

`services/secretsmanager/service.go` — Routes by X-Amz-Target. Name() returns "secretsmanager".

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add services/secretsmanager/
git commit -m "feat: add Secrets Manager service — secret CRUD, versioning, tags"
```

---

## Chunk 2: SSM Parameter Store & Messaging (SQS, SNS)

### Task 4: SSM Parameter Store

**Files:**
- Create: `services/ssm/service.go`
- Create: `services/ssm/service_test.go`
- Create: `services/ssm/handlers.go`
- Create: `services/ssm/store.go`

- [ ] **Step 1: Write test for parameter CRUD**

Tests:
1. `TestSSM_PutAndGetParameter` — PutParameter with Name=/app/db/host, Value=localhost, Type=String. GetParameter returns value.
2. `TestSSM_SecureString` — PutParameter with Type=SecureString. GetParameter with WithDecryption=true returns value.
3. `TestSSM_GetParametersByPath` — Put /app/db/host and /app/db/port and /app/api/key. GetParametersByPath with Path=/app/db returns 2 params.
4. `TestSSM_DeleteParameter` — PutParameter, DeleteParameter, GetParameter returns ParameterNotFound.
5. `TestSSM_PutParameter_Overwrite` — PutParameter twice with Overwrite=true. GetParameter returns latest.

SSM uses JSON with X-Amz-Target: AmazonSSM.<Action>.

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement SSM**

`services/ssm/store.go` — Parameter struct (Name, Value, Type, Version, ARN, LastModifiedDate, DataType). Store using map keyed by parameter name. Supports hierarchical paths with GetParametersByPath using prefix matching.

`services/ssm/handlers.go` — JSON handlers: PutParameter, GetParameter, GetParameters, GetParametersByPath, DeleteParameter, DeleteParameters, DescribeParameters.

`services/ssm/service.go` — Routes by X-Amz-Target. Name() returns "ssm".

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add services/ssm/
git commit -m "feat: add SSM Parameter Store — String, SecureString, StringList, path queries"
```

---

### Task 5: SQS Service

**Files:**
- Create: `services/sqs/service.go`
- Create: `services/sqs/service_test.go`
- Create: `services/sqs/handlers.go`
- Create: `services/sqs/store.go`
- Create: `services/sqs/queue.go`

- [ ] **Step 1: Write test for queue CRUD and messaging**

Tests:
1. `TestSQS_CreateAndListQueues` — CreateQueue "test-queue". ListQueues contains queue URL.
2. `TestSQS_SendAndReceive` — CreateQueue, SendMessage with body "hello". ReceiveMessage returns message with same body and a ReceiptHandle.
3. `TestSQS_DeleteMessage` — Send, Receive, DeleteMessage with ReceiptHandle. ReceiveMessage returns empty.
4. `TestSQS_VisibilityTimeout` — Send, Receive (message invisible). Immediately ReceiveMessage again returns empty. (Visibility timeout prevents re-delivery.)
5. `TestSQS_DeleteQueue` — CreateQueue, DeleteQueue. ListQueues does not contain it.
6. `TestSQS_GetQueueAttributes` — CreateQueue, GetQueueAttributes returns ApproximateNumberOfMessages.
7. `TestSQS_FIFO` — CreateQueue "test.fifo" with FifoQueue=true. Send with MessageGroupId and MessageDeduplicationId. Receive returns in order.

SQS uses query-string/form-encoded protocol with ?Action= parameter. Responses are XML.

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement SQS**

`services/sqs/queue.go` — Queue struct with name, URL, attributes, messages list, in-flight messages map, FIFO support. Message struct with MessageId, Body, ReceiptHandle, MD5OfBody, SentTimestamp, VisibilityDeadline. Deduplication cache for FIFO.

`services/sqs/store.go` — QueueStore managing all queues. CreateQueue, DeleteQueue, GetQueue, ListQueues.

`services/sqs/handlers.go` — XML response types for SQS. Handlers: CreateQueue, DeleteQueue, ListQueues, GetQueueUrl, GetQueueAttributes, SetQueueAttributes, SendMessage, ReceiveMessage, DeleteMessage, PurgeQueue, ChangeMessageVisibility, SendMessageBatch, DeleteMessageBatch.

`services/sqs/service.go` — Routes by Action query param. Name() returns "sqs". Generates queue URLs as `http://sqs.us-east-1.localhost:4566/000000000000/queue-name`.

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add services/sqs/
git commit -m "feat: add SQS service — standard and FIFO queues, send/receive/delete, visibility timeout"
```

---

### Task 6: SNS Service

**Files:**
- Create: `services/sns/service.go`
- Create: `services/sns/service_test.go`
- Create: `services/sns/handlers.go`
- Create: `services/sns/store.go`

- [ ] **Step 1: Write test for topics and publishing**

Tests:
1. `TestSNS_CreateAndListTopics` — CreateTopic "test-topic". ListTopics contains topic ARN.
2. `TestSNS_Subscribe` — CreateTopic, Subscribe with Protocol=sqs, Endpoint=arn:aws:sqs:.... Verify SubscriptionArn returned.
3. `TestSNS_Publish` — CreateTopic, Publish message. Verify MessageId returned. (Messages are stored in-memory for later retrieval via dashboard/admin API.)
4. `TestSNS_Unsubscribe` — Subscribe, Unsubscribe. ListSubscriptions does not contain it.
5. `TestSNS_DeleteTopic` — CreateTopic, DeleteTopic. ListTopics empty.
6. `TestSNS_TopicAttributes` — CreateTopic, GetTopicAttributes, SetTopicAttributes.

SNS uses query-string/form-encoded with ?Action=. XML responses.

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement SNS**

`services/sns/store.go` — Topic struct (ARN, Name, Attributes, Subscriptions). Subscription struct (ARN, Protocol, Endpoint, TopicArn, FilterPolicy). PublishedMessage struct for message log.

`services/sns/handlers.go` — XML response types. Handlers: CreateTopic, DeleteTopic, ListTopics, GetTopicAttributes, SetTopicAttributes, Subscribe, Unsubscribe, ListSubscriptions, ListSubscriptionsByTopic, Publish, TagResource, UntagResource.

`services/sns/service.go` — Routes by Action. Name() returns "sns".

Note: Cross-service delivery (SNS → SQS, SNS → Lambda) is deferred to the cross-service integrations plan. For now, published messages are stored in an in-memory log accessible via the admin API.

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add services/sns/
git commit -m "feat: add SNS service — topics, subscriptions, publish, message log"
```

---

## Chunk 3: S3 Object Operations

### Task 7: S3 Object CRUD

Extend the existing S3 service with object operations.

**Files:**
- Create: `services/s3/object_store.go`
- Create: `services/s3/object_handlers.go`
- Create: `services/s3/object_test.go`
- Modify: `services/s3/service.go` — add object routes
- Modify: `services/s3/store.go` — integrate object store

- [ ] **Step 1: Write test for object CRUD**

`services/s3/object_test.go`:
1. `TestS3_PutAndGetObject` — Create bucket, PUT /bucket/key.txt with body "file content". GET /bucket/key.txt returns same body with correct Content-Type.
2. `TestS3_DeleteObject` — Put object, DELETE /bucket/key.txt → 204. GET returns 404.
3. `TestS3_HeadObject` — Put object, HEAD /bucket/key.txt returns 200 with Content-Length and ETag headers. HEAD nonexistent returns 404.
4. `TestS3_ListObjects` — Put 3 objects. GET /bucket?list-type=2 returns ListBucketResult with all 3 keys.
5. `TestS3_ListObjectsWithPrefix` — Put /bucket/a/1.txt, /bucket/a/2.txt, /bucket/b/1.txt. List with prefix=a/ returns 2 objects.
6. `TestS3_CopyObject` — Put object, PUT /bucket/copy.txt with x-amz-copy-source header. GET copy returns same content.
7. `TestS3_ObjectInNonexistentBucket` — PUT object in nonexistent bucket → NoSuchBucket error.

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement object store**

`services/s3/object_store.go` — Object struct (Key, Body []byte, ContentType, ETag, Size, LastModified, Metadata map). ObjectStore per bucket, stores objects in map. Methods: PutObject, GetObject, DeleteObject, HeadObject, ListObjects (with prefix, delimiter, max-keys, continuation-token), CopyObject.

ETag is MD5 hex of body. Content-Type defaults to application/octet-stream.

- [ ] **Step 4: Implement object handlers**

`services/s3/object_handlers.go` — handlers for PutObject, GetObject, DeleteObject, HeadObject, ListObjectsV2, CopyObject.

GetObject writes raw body bytes with Content-Type, Content-Length, ETag headers (does NOT use WriteXMLResponse).
PutObject reads body bytes, stores in object store.
ListObjectsV2 returns XML ListBucketResult.

- [ ] **Step 5: Update service.go routing**

Add object routes to HandleRequest:
- GET /bucket (no key) → ListObjectsV2 (if ?list-type=2 or default)
- GET /bucket/key → GetObject
- PUT /bucket/key → if x-amz-copy-source header present → CopyObject, else PutObject
- DELETE /bucket/key → DeleteObject
- HEAD /bucket/key → HeadObject

Helper: `hasObjectKey(ctx)` checks if path has more than 1 segment.

- [ ] **Step 6: Run tests, verify pass**

- [ ] **Step 7: Run all tests including existing bucket tests**

Run: `go test ./services/s3/... -v`
Expected: ALL PASS (bucket + object tests)

- [ ] **Step 8: Commit**

```bash
git add services/s3/
git commit -m "feat: add S3 object operations — put, get, delete, head, list, copy"
```

---

## Chunk 4: DynamoDB

### Task 8: DynamoDB Core — Table CRUD and Item Operations

This is the most complex service. We'll use an in-memory store first (SQLite deferred).

**Files:**
- Create: `services/dynamodb/service.go`
- Create: `services/dynamodb/service_test.go`
- Create: `services/dynamodb/handlers.go`
- Create: `services/dynamodb/store.go`
- Create: `services/dynamodb/table.go`
- Create: `services/dynamodb/expression.go`

- [ ] **Step 1: Write test for table and item operations**

`services/dynamodb/service_test.go`:
1. `TestDynamoDB_CreateAndDescribeTable` — CreateTable with TableName, KeySchema (HASH), AttributeDefinitions. DescribeTable returns matching schema.
2. `TestDynamoDB_PutAndGetItem` — CreateTable, PutItem with {pk: "user1", name: "Alice"}. GetItem with key {pk: "user1"} returns full item.
3. `TestDynamoDB_DeleteItem` — PutItem, DeleteItem, GetItem returns empty.
4. `TestDynamoDB_UpdateItem` — PutItem, UpdateItem with SET name = :val. GetItem returns updated.
5. `TestDynamoDB_Query` — Put 3 items with same partition key, different sort keys. Query with KeyConditionExpression returns matching items in order.
6. `TestDynamoDB_Scan` — Put 5 items. Scan returns all 5.
7. `TestDynamoDB_DeleteTable` — CreateTable, DeleteTable. DescribeTable returns ResourceNotFoundException.
8. `TestDynamoDB_ListTables` — Create 2 tables. ListTables returns both names.

DynamoDB uses JSON with X-Amz-Target: DynamoDB_20120810.<Action>.

- [ ] **Step 2: Run test, verify fail**

- [ ] **Step 3: Implement DynamoDB table and store**

`services/dynamodb/table.go` — Table struct (Name, KeySchema, AttributeDefinitions, Items map, GSIs, LSIs, CreationDateTime, Status). Item is map[string]AttributeValue. AttributeValue supports S, N, B, BOOL, NULL, L, M, SS, NS, BS types.

`services/dynamodb/store.go` — TableStore with CreateTable, DeleteTable, DescribeTable, ListTables, GetTable. GetItem, PutItem, DeleteItem, UpdateItem, Query, Scan on a table.

`services/dynamodb/expression.go` — Basic expression parser for:
- KeyConditionExpression: `pk = :pk AND sk BEGINS_WITH :prefix`
- FilterExpression: basic comparisons
- UpdateExpression: `SET #name = :val, REMOVE #attr`
- ProjectionExpression: `attr1, attr2`
- ExpressionAttributeNames: `{"#name": "actualName"}`
- ExpressionAttributeValues: `{":val": {"S": "value"}}`

Start with simple expression support — exact match, begins_with, between for key conditions. SET and REMOVE for updates.

- [ ] **Step 4: Implement handlers**

`services/dynamodb/handlers.go` — JSON request/response types matching AWS DynamoDB API. Handlers: CreateTable, DeleteTable, DescribeTable, ListTables, PutItem, GetItem, DeleteItem, UpdateItem, Query, Scan, BatchGetItem, BatchWriteItem.

- [ ] **Step 5: Implement service**

`services/dynamodb/service.go` — Routes by X-Amz-Target: DynamoDB_20120810.<Action>. Name() returns "dynamodb".

- [ ] **Step 6: Run tests, verify pass**

- [ ] **Step 7: Commit**

```bash
git add services/dynamodb/
git commit -m "feat: add DynamoDB service — tables, items, query, scan, expressions"
```

---

## Chunk 5: Service Registration and Integration

### Task 9: Register All New Services in Gateway

**Files:**
- Modify: `cmd/gateway/main.go`
- Create: `tests/integration/services_integration_test.go`

- [ ] **Step 1: Update gateway main.go**

Register all new services in the gateway startup:
```go
import (
    stssvc "github.com/Viridian-Inc/cloudmock/services/sts"
    kmssvc "github.com/Viridian-Inc/cloudmock/services/kms"
    secretssvc "github.com/Viridian-Inc/cloudmock/services/secretsmanager"
    ssmsvc "github.com/Viridian-Inc/cloudmock/services/ssm"
    sqssvc "github.com/Viridian-Inc/cloudmock/services/sqs"
    snssvc "github.com/Viridian-Inc/cloudmock/services/sns"
    dynamodbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
    s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
)

// Register services
registry.Register(s3svc.New())
registry.Register(stssvc.New(cfg.AccountID))
registry.Register(kmssvc.New(cfg.AccountID, cfg.Region))
registry.Register(secretssvc.New(cfg.AccountID, cfg.Region))
registry.Register(ssmsvc.New(cfg.AccountID, cfg.Region))
registry.Register(sqssvc.New(cfg.AccountID, cfg.Region))
registry.Register(snssvc.New(cfg.AccountID, cfg.Region))
registry.Register(dynamodbsvc.New())
```

- [ ] **Step 2: Write integration test hitting all services**

`tests/integration/services_integration_test.go`:
- Test each service through the gateway with IAM mode "none"
- Verify basic CRUD for each service
- Verify all services show up in /_cloudmock/services endpoint

- [ ] **Step 3: Run all tests**

Run: `go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/gateway/ tests/integration/
git commit -m "feat: register all core services in gateway, add multi-service integration tests"
```

---

## Summary

This plan produces 8 working services:

| Service | Protocol | Key Operations |
|---------|----------|---------------|
| STS | Form/XML | GetCallerIdentity, AssumeRole, GetSessionToken |
| KMS | JSON | CreateKey, Encrypt, Decrypt, Aliases |
| Secrets Manager | JSON | CreateSecret, GetSecretValue, PutSecretValue, Delete |
| SSM | JSON | PutParameter, GetParameter, GetParametersByPath |
| SQS | Form/XML | CreateQueue, Send/Receive/Delete Message, FIFO |
| SNS | Form/XML | CreateTopic, Subscribe, Publish |
| S3 (objects) | REST | PutObject, GetObject, DeleteObject, ListObjects, CopyObject |
| DynamoDB | JSON | CreateTable, PutItem, GetItem, Query, Scan, UpdateItem |

**Next plan:** Plan 3 — Infrastructure services (CloudWatch, CloudWatch Logs, EventBridge, Cognito, API Gateway, Step Functions, Route 53, RDS, ECR, ECS, SES, Kinesis, Data Firehose)
