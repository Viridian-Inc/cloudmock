# Architecture

## Overview

cloudmock is a single Go binary that emulates the AWS API surface. All AWS services listen on a single port (default `4566`). An HTTP gateway dispatches each request to the appropriate service implementation based on the AWS Signature V4 credential scope and `X-Amz-Target` header.

```
Client (AWS CLI / SDK)
        │
        │ HTTP  :4566
        ▼
┌───────────────────┐
│     Gateway       │  pkg/gateway
│  (HTTP mux)       │
└──────┬────────────┘
       │
       ├─ IAM Middleware  ──────────────────────► IAM Engine  pkg/iam
       │  (auth + authz)                          (policy eval)
       │
       ├─ Service Router  ──────────────────────► Routing     pkg/routing
       │  (detect svc + action)                   (credential scope / X-Amz-Target)
       │
       ├─ Tier 1 Service  (e.g. services/s3)
       │   HandleRequest()
       │   in-memory store
       │
       └─ Tier 2 Stub     pkg/stub
           StubService.HandleRequest()
           ResourceStore (generic CRUD)

Admin API  :4599        pkg/admin
Dashboard  :4500        pkg/dashboard
```

---

## Directory Structure

```
cmd/cloudmock/          CLI binary (start, stop, status, reset, …)
gateway/                Gateway binary entry point
pkg/
  config/               Configuration loading and defaults
  gateway/              HTTP gateway: routing, IAM middleware, response encoding
  iam/                  IAM store, engine, auth, policy types
  routing/              Service and action detection from request headers
  service/              Service interface and shared types
  stub/                 Generic stub engine (Tier 2)
  admin/                Admin REST API
  dashboard/            Web dashboard
services/
  s3/                   S3 full implementation
  dynamodb/             DynamoDB full implementation
  … (22 more Tier 1 services)
  stubs/
    catalog.go          Tier 2 service model definitions (74 services)
```

---

## Request Lifecycle

1. **Receive** — The gateway's `http.ServeMux` matches all requests to `handleAWSRequest`.

2. **Detect service** — `routing.DetectService(r)` reads the AWS Signature V4 `Authorization` header to extract the service name from the credential scope (`AKID/date/region/SERVICE/aws4_request`). If absent, it falls back to the `X-Amz-Target` header.

3. **Authenticate** — If IAM mode is `enforce` or `authenticate`, the IAM middleware validates the request signature and looks up the access key in the credential store.

4. **Authorize** — In `enforce` mode, the IAM engine evaluates the caller's attached policies against the requested action and resource ARN. Explicit Deny wins; Allow is required; all else is implicit Deny.

5. **Dispatch** — The service registry looks up the named service and calls `service.HandleRequest(ctx)`.

6. **Handle** — The service implementation processes the request and returns a `*service.Response`.

7. **Encode** — The gateway serializes the response as JSON or XML depending on the service's protocol and writes it to the HTTP response.

---

## Service Framework

Every service implements the `service.Service` interface (`pkg/service/service.go`):

```go
type Service interface {
    Name() string
    Actions() []Action
    HandleRequest(ctx *RequestContext) (*Response, error)
    HealthCheck() error
}
```

- `Name()` returns the lowercase AWS service identifier (e.g. `"s3"`, `"dynamodb"`).
- `Actions()` declares every supported API action, its HTTP method, and the IAM action string used for policy evaluation.
- `HandleRequest()` receives a `*RequestContext` (action name, region, account ID, caller identity, raw HTTP request, decoded body) and returns a `*Response`.
- `HealthCheck()` is called by the admin API; Tier 1 services always return `nil`.

Services are registered with the routing registry at startup. The registry is populated in `services/register.go` based on the active profile.

---

## IAM Engine

The IAM engine (`pkg/iam/engine.go`) implements AWS IAM evaluation semantics:

1. Root callers (identified by `IsRoot: true` on the `AccessKey`) are always allowed.
2. All policies attached to the caller principal are collected.
3. Every statement whose `Action` and `Resource` match the request is evaluated:
   - Any **Deny** statement results in an explicit Deny (returned immediately).
   - Any **Allow** statement is noted.
4. If at least one Allow was found and no Deny, the request is allowed.
5. Otherwise it is an implicit Deny.

Action matching supports wildcards (`s3:*`, `s3:Get*`). Resource matching supports `*` and ARN prefix matching.

---

## Stub Engine

Tier 2 services are driven by the stub engine (`pkg/stub/engine.go`). Each service is described by a `ServiceModel` that lists:

- `ServiceName` — the AWS service identifier
- `Protocol` — `query`, `json`, `rest-json`, or `rest-xml`
- `Actions` — map of action name → `Action` (type: create/describe/list/delete/update/other)
- `ResourceTypes` — map of resource type key → `ResourceType` (ID field, ARN pattern, fields)

At runtime `StubService.HandleRequest()` parses the request body (JSON or form-encoded), validates required fields, and delegates to a generic handler:

| Action type | Behaviour |
|-------------|-----------|
| `create` | Generates a random ID, stores the resource, returns ID and ARN |
| `describe` | Looks up a resource by its ID field |
| `list` | Returns all resources of that type |
| `delete` | Removes a resource by ID |
| `update` | Merges fields into an existing resource |
| `other` | Returns an empty success response |

Resource state is held in `pkg/stub/resource.go` — an in-memory map keyed by resource type and ID.

---

## How to Add a New Tier 1 Service

1. Create `services/<name>/` with at minimum `service.go` and `store.go`.

2. Implement the `service.Service` interface. The `Actions()` method must list every supported action:

```go
func (s *MyService) Actions() []service.Action {
    return []service.Action{
        {Name: "CreateFoo", Method: http.MethodPost, IAMAction: "myservice:CreateFoo"},
        {Name: "DeleteFoo", Method: http.MethodDelete, IAMAction: "myservice:DeleteFoo"},
    }
}
```

3. Implement `HandleRequest`. Use a switch on `ctx.Action`:

```go
func (s *MyService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
    switch ctx.Action {
    case "CreateFoo":
        return handleCreateFoo(s.store, ctx)
    case "DeleteFoo":
        return handleDeleteFoo(s.store, ctx)
    default:
        return nil, service.NewAWSError("InvalidAction", "unknown action", http.StatusBadRequest)
    }
}
```

4. Register the service in `services/register.go` inside the appropriate profile block.

## How to Add a New Tier 2 Service

Add an entry to the appropriate function in `services/stubs/catalog.go`:

```go
{
    ServiceName:  "mynewservice",
    Protocol:     "json",
    TargetPrefix: "MyNewService_20240101",
    Actions: map[string]stub.Action{
        "CreateWidget": createAction("CreateWidget", "widget", "WidgetId",
            []stub.Field{reqStr("WidgetName")},
            []stub.Field{optStr("WidgetName")}),
        "DescribeWidget": describeAction("DescribeWidget", "widget", "WidgetId"),
        "ListWidgets":    listAction("ListWidgets", "widget"),
        "DeleteWidget":   deleteAction("DeleteWidget", "widget", "WidgetId"),
    },
    ResourceTypes: map[string]stub.ResourceType{
        "widget": rt("Widget", "WidgetId",
            "arn:aws:mynewservice:{region}:{account}:widget/{id}",
            []stub.Field{optStr("WidgetName")}),
    },
},
```

No other changes are needed — the stub engine handles routing and CRUD automatically.
