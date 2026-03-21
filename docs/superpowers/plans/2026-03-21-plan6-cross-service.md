# Cross-Service Integrations Implementation Plan

> **For agentic workers:** This is a retrospective document. All tasks listed below were already executed. Checkboxes are marked completed.

**Goal:** Wire real cross-service behavior into cloudmock so that actions in one service automatically trigger effects in another — matching how AWS services interconnect in production. Specifically: S3 object events fan out to SQS, SNS Publish delivers to subscribed SQS queues, and EventBridge PutEvents delivers to SQS/SNS targets.

**Status:** COMPLETED

---

## Overview

Plan 6 introduced the event bus and a service locator pattern to enable cross-service communication without creating circular dependencies between service packages.

Two integration mechanisms were built:

1. **Event bus** (`pkg/eventbus`) — an in-process pub/sub bus for fire-and-forget cross-service events. S3 publishes object events; the integration layer subscribes and delivers them to SQS using a naming convention.

2. **Service locator** — SNS and EventBridge accept a `ServiceLocator` interface at construction time (or via `SetLocator`). After all services are registered, the gateway calls `SetLocator(registry)` to break the circular dependency. This allows SNS to look up SQS and deliver messages directly, and EventBridge to look up SQS or SNS targets.

---

## Chunk 1: Event Bus

### Task 1: Event Bus Types (`pkg/eventbus/types.go`)

- [x] Define `Event` struct: `Source`, `Type`, `Detail`, `Time`, `Region`, `AccountID`
- [x] Define `Subscription` struct: `ID`, `Source`, `Types`, `Handler`
- [x] Define `EventHandler` type: `func(event *Event) error`
- [x] Event types follow AWS naming convention: e.g., `s3:ObjectCreated:Put`, `s3:ObjectRemoved:Delete`

**Files created:**
- `pkg/eventbus/types.go`

### Task 2: Event Bus Implementation (`pkg/eventbus/bus.go`)

- [x] Implement `Bus` struct with `subscriptions []*Subscription` protected by `sync.RWMutex`
- [x] Implement `NewBus()` constructor
- [x] Implement `Subscribe(sub)` — appends subscription, generates random UUID if `sub.ID` is empty, returns ID
- [x] Implement `Unsubscribe(id)` — removes subscription by ID
- [x] Implement `Publish(event)` — finds matching subscriptions under read lock, invokes each handler in a separate goroutine (fire-and-forget, errors discarded)
- [x] Implement `PublishSync(event)` — synchronous variant for tests, returns `[]error`
- [x] Implement `matches(sub, event)` — filters by `sub.Source` and `sub.Types`
- [x] Implement `matchType(pattern, eventType)` — supports exact match and trailing `*` wildcard (e.g., `s3:*` matches `s3:ObjectCreated:Put`)
- [x] Implement `randomID()` — generates UUID-formatted random ID

**Files created:**
- `pkg/eventbus/bus.go`

---

## Chunk 2: S3 Event Publishing

### Task 3: S3 Service — Event Bus Integration (`services/s3/service.go`)

- [x] Add `bus *eventbus.Bus` field to `S3Service`
- [x] Implement `NewWithBus(bus)` constructor alongside the existing `New()` constructor
- [x] Implement `publishObjectEvent(eventType, bucket, key, size, etag)` — builds an `eventbus.Event` and calls `bus.Publish()` when the bus is non-nil
- [x] Wire event publishing into `PutObject`, `CopyObject`, and `DeleteObject` handlers
  - `PutObject` → publishes `s3:ObjectCreated:Put`
  - `CopyObject` → publishes `s3:ObjectCreated:Copy`
  - `DeleteObject` → publishes `s3:ObjectRemoved:Delete`
- [x] Update gateway registration to use `s3svc.NewWithBus(bus)` instead of `s3svc.New()`

**Files modified:**
- `services/s3/service.go`
- `cmd/gateway/main.go`

---

## Chunk 3: S3 → SQS Integration Wiring

### Task 4: Integration Wiring (`pkg/integration/wiring.go`)

- [x] Define `SQSEnqueuer` interface: `EnqueueDirect(queueName, messageBody string) bool`
- [x] Implement `WireIntegrations(bus, registry, accountID, region)` — subscribes to all `s3:*` events on the bus and routes them to SQS
- [x] Implement `handleS3Event(registry, event, accountID, region)`:
  - Looks up `"sqs"` in the registry
  - Casts to `SQSEnqueuer` (no-op if SQS is not registered or does not implement the interface)
  - Extracts `bucket`, `key`, `size`, `etag` from `event.Detail`
  - Calls `buildS3EventNotification(...)` to produce AWS-compatible JSON
  - Delivers to queue named `s3-events-{bucket}` via `EnqueueDirect`
- [x] Implement `buildS3EventNotification(...)` — produces a standard AWS S3 event notification envelope with `Records[0]` containing `eventVersion`, `eventSource`, `awsRegion`, `eventTime`, `eventName`, `s3.bucket`, `s3.object`
- [x] Wire `WireIntegrations` into `cmd/gateway/main.go` after all services are registered

**Files created/modified:**
- `pkg/integration/wiring.go`
- `cmd/gateway/main.go`

### Task 5: SQS `EnqueueDirect` (`services/sqs/service.go`)

- [x] Implement `EnqueueDirect(queueName, messageBody string) bool` — adds a message directly to the named queue without an HTTP request, returns false if queue does not exist
- [x] SQS service automatically satisfies the `SQSEnqueuer` interface

**Files modified:**
- `services/sqs/service.go`

---

## Chunk 4: SNS → SQS Fan-out

### Task 6: SNS Service Locator Pattern (`services/sns/service.go`)

- [x] Define `ServiceLocator` interface: `Lookup(name string) (service.Service, error)`
- [x] Add `locator ServiceLocator` field to `SNSService`
- [x] Implement `NewWithLocator(accountID, region, locator)` constructor
- [x] Implement `SetLocator(locator)` — allows post-construction injection to break circular dependency
- [x] Implement `PublishDirect(topicName, message, subject string) bool` — publishes to a topic by name for internal use
- [x] Wire SNS `Publish` action to look up subscribed SQS queues via the service locator and deliver messages via `SQSEnqueuer.EnqueueDirect`
- [x] Update gateway startup to call `snsService.SetLocator(registry)` after all services are registered

**Files modified:**
- `services/sns/service.go`
- `cmd/gateway/main.go`

---

## Chunk 5: EventBridge → SQS/SNS Target Delivery

### Task 7: EventBridge Service Locator Pattern (`services/eventbridge/service.go`)

- [x] Define `ServiceLocator` interface (same shape as SNS): `Lookup(name string) (service.Service, error)`
- [x] Add `locator ServiceLocator` field to `EventBridgeService`
- [x] Implement `NewWithLocator(accountID, region, locator)` constructor
- [x] Implement `SetLocator(locator)` — post-construction injection
- [x] Wire `PutEvents` to evaluate rules against registered event buses, match targets, and deliver to SQS/SNS targets via the service locator
- [x] Update gateway startup to call `ebService.SetLocator(registry)` after all services are registered

**Files modified:**
- `services/eventbridge/service.go`
- `cmd/gateway/main.go`

---

## Architecture Summary

```
S3 PutObject
    │
    ▼  bus.Publish("s3:ObjectCreated:Put")
EventBus
    │
    ▼  handleS3Event (pkg/integration)
SQS.EnqueueDirect("s3-events-{bucket}", body)

SNS Publish → topic subscribers → SQS.EnqueueDirect(queueName, body)
                                    (via ServiceLocator)

EventBridge PutEvents → rule matching → SQS/SNS targets
                                         (via ServiceLocator)
```

---

## Verification

- [x] `eventbus.Bus` fan-out delivers to all matching subscriptions
- [x] Wildcard type matching (`s3:*`) works correctly
- [x] `PublishSync` variant used in tests for deterministic delivery
- [x] S3 `PutObject` triggers SQS delivery when queue `s3-events-{bucket}` exists
- [x] SNS `Publish` delivers to subscribed SQS queue via `EnqueueDirect`
- [x] EventBridge `PutEvents` delivers to registered targets
- [x] `SetLocator` pattern correctly breaks circular dependency at startup
- [x] Event bus tests pass (`pkg/eventbus/bus_test.go`)
