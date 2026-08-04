# Architecture

## Overview

cloudmock is a single Go binary that emulates cloud provider APIs locally. AWS remains the default and backward-compatible provider, while the gateway and registry are now provider-aware so Azure and future Google Cloud services can coexist on the same endpoint.

All cloud APIs listen on one gateway port by default (`4566`). The gateway detects a provider-specific route target, then dispatches to the service implementation registered for that provider, logical service, and optional API version.

```
Client (AWS CLI / SDK, Azure SDK / CLI, IaC provider)
        │
        │ HTTP  :4566
        ▼
┌───────────────────┐
│     Gateway       │  pkg/gateway
│  (HTTP mux)       │
└──────┬────────────┘
       │
       ├─ Provider Detection ───────────────────► Routing     pkg/routing
       │  AWS SigV4 / X-Amz-Target                 RouteTarget{Provider, Service,
       │  Azure ARM / data-plane hosts             Action, APIVersion}
       │
       ├─ Auth Layer ───────────────────────────► AWS IAM today, Azure RBAC planned
       │  SigV4 or Bearer material                 provider-specific identities
       │
       ├─ Versioned Registry ───────────────────► provider/service/api-version lookup
       │                                          legacy AWS lookup fallback
       │
       ├─ AWS Tier 1 Service  (e.g. services/s3)
       │   HandleRequest()
       │   in-memory store
       │
       ├─ Azure Service  (e.g. services/azure/storage)
       │   ARM control plane + data plane
       │   provider-specific response helpers
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
  azurearm/             Azure ARM IDs, JSON responses, errors, async/paging helpers
  config/               Configuration loading and defaults
  gateway/              HTTP gateway: provider routing, auth, response encoding
  iam/                  AWS IAM store, engine, auth, policy types
  routing/              Provider, service, action, and API-version detection
  service/              Service interface and shared types
  stub/                 Generic stub engine (Tier 2)
  admin/                Admin REST API
  dashboard/            Web dashboard
services/
  azure/
    authorization/      Microsoft.Authorization role assignment and management lock APIs
    compute/            Microsoft.Compute virtual machine lifecycle APIs
    resources/          Microsoft.Resources ARM resource groups/providers/deployments/resources/tags
    storage/            Microsoft.Storage accounts, keys, Blob range/listing, and Queue peek/receive APIs
    keyvault/           Microsoft.KeyVault vaults, secrets, keys, and crypto APIs
  s3/                   S3 full implementation
  dynamodb/             DynamoDB full implementation
  … (22 more Tier 1 services)
  stubs/
    catalog.go          Tier 2 service model definitions (74 services)
```

---

## Request Lifecycle

1. **Receive** — The gateway's `http.ServeMux` matches all cloud API requests to the cloud API handler.

2. **Detect target** — `routing.DetectTarget(r)` returns a `RouteTarget` with provider, service, action, and API version.
   - AWS uses the Signature V4 credential scope, `X-Amz-Target`, `Action`, and AWS protocol version markers.
   - Azure ARM uses `management.azure.com`, ARM resource paths, and the `api-version` query parameter.
   - Azure data-plane services use provider-specific hosts such as `*.blob.core.windows.net`, `*.queue.core.windows.net`, and `*.vault.azure.net`, with `x-ms-version` or `api-version`.
   - GCP detection is reserved for `*.googleapis.com` and `*.googleapis.cn`.

3. **Authenticate** — AWS requests use the existing IAM middleware and credential store. Azure requests currently accept Bearer-token material in local/test workflows; full Microsoft Entra token validation and Azure RBAC enforcement are planned provider-specific auth layers.

4. **Authorize** — AWS requests in `enforce` mode use the IAM engine. Azure authorization will map role assignments and role definitions to ARM scopes such as subscription, resource group, and resource ID.

5. **Dispatch** — `routing.Registry.LookupTarget` resolves the service by provider, service name, and API version. AWS requests without an explicit version fall back to the legacy `Lookup` path, so existing CloudMock service registration and SDK behavior remain compatible.

6. **Handle** — The service implementation processes the request and returns a `*service.Response`. Azure services use `pkg/azurearm` helpers for ARM JSON responses and error envelopes; Azure Storage data-plane responses use XML or raw bytes where the service protocol requires it, including Blob listing XML with continuation markers and virtual-directory `BlobPrefix` entries.

7. **Encode** — The gateway serializes the response as JSON or XML depending on the service's protocol and writes it to the HTTP response.

---

## Provider And API Versioning

Provider routing is represented by `routing.RouteTarget`:

```go
type RouteTarget struct {
    Provider   Provider
    Service    string
    Action     string
    APIVersion string
}
```

Services can still register with the legacy AWS-only API:

```go
registry.Register(s3Service)
```

Provider-aware services register exact version keys:

```go
registry.RegisterVersioned(routing.ServiceKey{
    Provider:   routing.ProviderAzure,
    Service:    "Microsoft.Storage/storageAccounts",
    APIVersion: "2024-01-01",
}, storageService)
```

This keeps versioned APIs isolated:

- AWS v1/v2 or protocol-specific services can register separate versioned implementations while old AWS services keep the existing `Register` and `Lookup` behavior.
- Azure resource provider versions such as `Microsoft.KeyVault/vaults@2024-11-01` and `Microsoft.KeyVault/vaults@2023-07-01` can coexist in the same registry.
- Tests can call `LookupVersioned` or send HTTP fixtures with explicit API versions to validate each version independently.
- `SetDefaultVersion(provider, service, version)` is explicit and only applies when a provider request omits its version. Azure SDK and ARM requests normally include `api-version`, so defaulting is a compatibility fallback rather than the primary route.
- Service packages should keep version differences close to the provider package. If one version requires different behavior, create a distinct implementation or version-specific handler table rather than branching in the gateway.

---

## Service Framework

Every cloud service still implements the `service.Service` interface (`pkg/service/service.go`):

```go
type Service interface {
    Name() string
    Actions() []Action
    HandleRequest(ctx *RequestContext) (*Response, error)
    HealthCheck() error
}
```

- `Name()` returns the legacy AWS service identifier for AWS services (for example `"s3"`, `"dynamodb"`) or the provider namespace for Azure services (for example `"Microsoft.Storage"`).
- `Actions()` declares every supported API action, its HTTP method, and the provider-specific authorization action string used for policy evaluation.
- `HandleRequest()` receives a `*RequestContext` containing action name, region, account ID, caller identity, raw HTTP request, decoded body, query params, and the detected service.
- `HealthCheck()` is called by the admin API; Tier 1 services always return `nil`.

AWS services are registered with the routing registry at startup through the existing `Register` and lazy registration paths. Provider-aware services register one or more `routing.ServiceKey` values with `RegisterVersioned`.

Azure service packages expose a `ServiceKeys()` helper when they own multiple ARM or data-plane route keys. For example, `services/azure/keyvault` registers separate keys for `Microsoft.KeyVault/vaults` control-plane versions and `Microsoft.KeyVault/secrets` data-plane versions.

ARM template deployments are provider fan-out operations. `services/azure/resources` keeps template parsing and dependency ordering in the Microsoft.Resources service, while individual Azure services opt in through:

```go
type TemplateProvisioner interface {
    SupportsTemplateResource(resource map[string]any) bool
    ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error)
}
```

This allows one deployment to provision a Storage account and a Key Vault without making Microsoft.Resources import service-specific state stores directly.

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

## How to Add a Provider-Aware Service

1. Add the provider package under `services/<provider>/<service>/`. Keep control-plane and data-plane handlers in the same package only when they share state, such as Azure Storage accounts and Blob/Queue data-plane resources.

2. Implement `service.Service` without changing the interface. Put provider-specific parsing in the service package or a provider helper package such as `pkg/azurearm`, not in the gateway.

3. Expose exact route keys for every API version the service supports:

```go
func (s *MyAzureService) ServiceKeys() []routing.ServiceKey {
    return []routing.ServiceKey{
        {Provider: routing.ProviderAzure, Service: "Microsoft.Example/widgets", APIVersion: "2026-01-01"},
        {Provider: routing.ProviderAzure, Service: "Microsoft.Example/widgets", APIVersion: "2025-01-01"},
    }
}
```

4. Register every key at startup with `registry.RegisterVersioned`. Set a default version only when clients can legitimately omit a version; do not rely on defaults for Azure ARM compatibility because ARM requests are expected to send `api-version`.

5. Write tests at three levels: target detection in `pkg/routing`, versioned registry lookup in `pkg/routing`, and HTTP/service behavior in the provider package. When a new version differs materially, add version-specific fixtures instead of broadening one test case.

6. For Azure ARM template resources, implement `TemplateProvisioner` only when the service can create resources from resolved ARM template resource objects. Unsupported resources should remain generic deployment output until a first-class provider service owns them.

---

## How to Add a New AWS Tier 1 Service

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

## How to Add a New AWS Tier 2 Service

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
