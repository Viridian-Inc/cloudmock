# Multi-Cloud Azure Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor CloudMock from AWS-only routing and registration into provider-aware routing that can support Azure services, Azure API versions, and future Google Cloud support without breaking existing AWS behavior.

**Architecture:** Keep the existing AWS `service.Service` interface and `routing.Registry` methods compatible. Add provider-aware route metadata and versioned registration as opt-in capabilities, then wire Azure ARM and data-plane services through those capabilities service by service. Azure control-plane requests route by ARM resource provider namespace, resource type, and `api-version`; data-plane requests route by provider-specific hostnames such as Blob, Queue, Key Vault, Cosmos DB, and Service Bus endpoints.

**Tech Stack:** Go 1.26, `net/http`, existing `pkg/routing`, `pkg/service`, existing service packages under `services/`, Microsoft Learn REST API documentation, existing `go test` workflow.

---

## Files

- Created: `pkg/routing/target.go` for provider, route target, API version, and Azure ARM route detection.
- Modified: `pkg/routing/router.go` to allow explicit `X-Cloudmock-Action` fixture overrides.
- Modified: `pkg/routing/registry.go` to add provider/version-aware service registration and lookup while preserving existing AWS methods.
- Created: `pkg/routing/target_test.go` for provider/version route detection tests.
- Created: `pkg/routing/versioned_registry_test.go` for coexistence and default-version registry tests.
- Created: `docs/azure-implementation-checklists.md` for Microsoft Learn-backed Azure implementation checklists.
- Next: `pkg/gateway/gateway.go` should call `routing.DetectTarget` and `registry.LookupTarget` after compatibility tests prove AWS behavior remains unchanged.
- Next: `pkg/azurearm/` should hold shared ARM response envelopes, IDs, async operations, `nextLink`, errors, and provider manifests.
- Next: `services/azure/resources/` should implement `Microsoft.Resources` resource groups, providers, deployments, generic resources, tags, and locks.

## Task 1: Provider-Aware Routing And Versioned Registry

**Files:**
- Create: `pkg/routing/target.go`
- Create: `pkg/routing/target_test.go`
- Create: `pkg/routing/versioned_registry_test.go`
- Modify: `pkg/routing/router.go`
- Modify: `pkg/routing/registry.go`

- [x] **Step 1: Write failing tests for provider-aware target detection**

```go
target := routing.DetectTarget(req)
assert.Equal(t, routing.ProviderAzure, target.Provider)
assert.Equal(t, "Microsoft.Storage/storageAccounts", target.Service)
assert.Equal(t, "CreateOrUpdate", target.Action)
assert.Equal(t, "2023-01-01", target.APIVersion)
```

- [x] **Step 2: Run test to verify it fails**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing
```

Expected red result:

```text
undefined: DetectTarget
undefined: ProviderAWS
undefined: ProviderAzure
reg.LookupTarget undefined
undefined: RouteTarget
```

- [x] **Step 3: Implement the smallest provider-aware target and registry API**

Implemented:

```go
type Provider string
type RouteTarget struct {
    Provider   Provider
    Service    string
    Action     string
    APIVersion string
}
type ServiceKey struct {
    Provider   Provider
    Service    string
    APIVersion string
}
```

- [x] **Step 4: Run test to verify it passes**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing
```

Expected green result:

```text
ok github.com/Viridian-Inc/cloudmock/pkg/routing
```

## Task 2: Gateway Compatibility Wiring

**Files:**
- Modify: `pkg/gateway/gateway.go`
- Modify: `pkg/gateway/gateway_test.go`
- Test: `pkg/gateway/gateway_test.go`
- Test: `pkg/routing/target_test.go`

- [ ] **Step 1: Write failing AWS compatibility tests**

Add a gateway test that registers a legacy AWS service with `registry.Register(svc)`, sends an existing SigV4-style request, and asserts the same service handles the request after gateway code switches to `DetectTarget` and `LookupTarget`.

```go
func TestGateway_LegacyAWSRegistryStillRoutesAfterProviderTargetDetection(t *testing.T) {
    handler := newTestGateway(t, "none", &echoService{})
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKID/20240101/us-east-1/s3/aws4_request, SignedHeaders=host, Signature=abc")
    w := httptest.NewRecorder()

    handler.ServeHTTP(w, req)

    require.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run test to verify it fails before gateway wiring**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/gateway -run TestGateway_LegacyAWSRegistryStillRoutesAfterProviderTargetDetection -count=1
```

Expected: fail until `handleAWSRequest` uses `DetectTarget` and `LookupTarget`.

- [ ] **Step 3: Wire route target into the gateway**

Change the service detection section to keep old names while recording provider metadata:

```go
target := routing.DetectTarget(r)
svcName := target.Service
if svcName == "" {
    // existing MissingAuthenticationToken behavior
}
```

Change service lookup to:

```go
svc, err = g.registry.LookupTarget(target)
```

- [ ] **Step 4: Run gateway and routing tests**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/gateway ./pkg/routing
```

Expected: both packages pass with existing AWS tests unchanged.

## Task 3: Azure ARM Shared Package

**Files:**
- Create: `pkg/azurearm/id.go`
- Create: `pkg/azurearm/id_test.go`
- Create: `pkg/azurearm/error.go`
- Create: `pkg/azurearm/error_test.go`
- Create: `pkg/azurearm/async.go`
- Create: `pkg/azurearm/async_test.go`
- Create: `pkg/azurearm/paging.go`
- Create: `pkg/azurearm/paging_test.go`

- [ ] **Step 1: Write failing resource ID parser tests**

Test subscription, resource group, provider namespace, type chain, and resource name extraction from canonical ARM IDs.

```go
id, err := azurearm.ParseID("/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct")
require.NoError(t, err)
assert.Equal(t, "sub-1", id.SubscriptionID)
assert.Equal(t, "rg-a", id.ResourceGroup)
assert.Equal(t, "Microsoft.Storage", id.Provider)
assert.Equal(t, []string{"storageAccounts"}, id.Types)
assert.Equal(t, []string{"acct"}, id.Names)
```

- [ ] **Step 2: Run parser test red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/azurearm -run TestParseID -count=1
```

Expected: package missing or `ParseID` undefined.

- [ ] **Step 3: Implement resource ID parser and formatter**

Implement case-insensitive segment matching for `subscriptions`, `resourceGroups`, and `providers`, while preserving original provider namespace and resource type casing in returned fields.

- [ ] **Step 4: Add Azure error envelope tests**

Use the Microsoft Learn ARM error shape:

```json
{"error":{"code":"ResourceNotFound","message":"The Resource 'x' was not found."}}
```

- [ ] **Step 5: Implement `azurearm.Error` and response writer**

Return JSON with `Content-Type: application/json` and status codes from service handlers.

- [ ] **Step 6: Add async operation tests**

Test `Azure-AsyncOperation`, `Location`, `Retry-After`, `status`, `provisioningState`, and terminal states `Succeeded`, `Failed`, and `Canceled`.

- [ ] **Step 7: Implement async operation store**

Keep operations in memory by operation ID and resource ID, with deterministic operation IDs for tests.

- [ ] **Step 8: Add paging tests**

Test `value` plus `nextLink` with continuation tokens, including final page with no `nextLink`.

- [ ] **Step 9: Run shared package tests**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/azurearm
```

Expected: all `pkg/azurearm` tests pass.

## Task 4: Microsoft.Resources Service

**Files:**
- Create: `services/azure/resources/service.go`
- Create: `services/azure/resources/store.go`
- Create: `services/azure/resources/handlers.go`
- Create: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Test: `services/azure/resources/service_test.go`

- [x] **Step 1: Write failing resource group lifecycle tests**

Exercise `PUT`, `GET`, `LIST`, and `DELETE` for `/subscriptions/{subscriptionId}/resourcegroups/{name}?api-version=2021-04-01`.

- [x] **Step 2: Implement resource group store and handlers**

Return `id`, `name`, `type`, `location`, `tags`, and `properties.provisioningState`.

- [x] **Step 3: Write failing providers list/get/register tests**

Exercise provider metadata for `Microsoft.Resources`, `Microsoft.Storage`, `Microsoft.KeyVault`, and `Microsoft.Network`.

- [x] **Step 4: Implement provider manifest and registration states**

Expose resource types and API versions from static manifests backed by `docs/azure-implementation-checklists.md`.

- [x] **Step 5: Write failing deployment create/get/list tests**

Use an inline ARM template with one storage account resource and verify a deployment record is stored.

- [x] **Step 6: Implement deployment record storage**

Support `Incremental` mode, parameter substitution, outputs, and dependency ordering for first-slice resource types.

- [x] **Step 7: Register Azure resource service lazily**

Add versioned registration for:

```go
routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/resourceGroups", APIVersion: "2021-04-01"}
```

- [x] **Step 8: Run Azure resource tests**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources ./pkg/azurearm ./pkg/routing
```

Expected: all listed packages pass.

## Task 5: First Azure Data Plane Slice

**Files:**
- Create: `services/azure/storage/service.go`
- Create: `services/azure/storage/store.go`
- Create: `services/azure/storage/blob_handlers.go`
- Create: `services/azure/storage/queue_handlers.go`
- Create: `services/azure/storage/service_test.go`
- Modify: `cmd/gateway/main.go`

- [x] **Step 1: Write failing storage account ARM tests**

Use `Microsoft.Storage/storageAccounts` create/get/list/delete with API version `2024-01-01`.

- [x] **Step 2: Implement storage account control plane**

Return account endpoints for blob, queue, table, and file services.

- [x] **Step 3: Write failing Blob data-plane tests**

Exercise create container, put blob, get blob, list blobs, and delete blob using local endpoint host routing.

- [x] **Step 4: Implement Blob data plane**

Preserve content headers, metadata, ETags, last modified time, and continuation tokens.

- [x] **Step 5: Write failing Queue data-plane tests**

Exercise create queue, put message, get messages, delete message, visibility timeout, and pop receipt validation.

- [x] **Step 6: Implement Queue data plane**

Keep message state isolated by storage account and queue name.

- [x] **Step 7: Run storage tests**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/azurearm ./pkg/routing
```

Expected: all listed packages pass.

## Task 6: Azure Key Vault Slice And Multi-Provider ARM Deployment Fan-Out

**Files:**
- Create: `services/azure/keyvault/service.go`
- Create: `services/azure/keyvault/types.go`
- Create: `services/azure/keyvault/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service.go`
- Modify: `services/azure/resources/deployments.go`
- Modify: `cmd/gateway/main.go`

- [x] **Step 1: Write failing Key Vault control-plane tests**

Use `Microsoft.KeyVault/vaults` create/get/list/delete with API versions `2024-11-01` and `2023-07-01`.

- [x] **Step 2: Implement Key Vault ARM control plane**

Return vault resource IDs, `vaultUri`, SKU, access policies, public network access, tags, and `Succeeded` provisioning state.

- [x] **Step 3: Write failing Key Vault secret data-plane tests**

Use `*.vault.azure.net` host routing for set/get/list/delete secret operations with `api-version=2025-07-01`.

- [x] **Step 4: Implement Key Vault secret data plane**

Support secret value storage, versions, attributes, content type, tags, list-without-value behavior, and soft-delete-style deleted secret responses.

- [x] **Step 5: Write failing ARM deployment fan-out tests**

Deploy one ARM template containing `Microsoft.Storage/storageAccounts` and `Microsoft.KeyVault/vaults`, then assert both provider services receive the provisioned resources.

- [x] **Step 6: Implement provider-owned template provisioning hooks**

Add `SupportsTemplateResource` and `ProvisionTemplateResource` to provider services so `Microsoft.Resources` can fan out to each owning provider without importing provider stores directly.

- [x] **Step 7: Register Key Vault and multi-provider provisioning at startup**

Register `Microsoft.KeyVault/vaults` and `Microsoft.KeyVault/secrets` versioned service keys, add Key Vault as an ARM template provisioner, and explicitly set latest Key Vault API versions as defaults.

- [x] **Step 8: Run Key Vault and deployment tests**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault ./services/azure/resources ./pkg/azurearm ./pkg/routing
```

Expected: all listed packages pass.

## Task 7: Azure Authorization Role Assignments

**Files:**
- Create: `services/azure/authorization/service.go`
- Create: `services/azure/authorization/types.go`
- Create: `services/azure/authorization/service_test.go`
- Modify: `pkg/routing/target.go`
- Modify: `pkg/routing/target_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`

- [x] **Step 1: Write failing nested extension-resource routing test**

Use a nested ARM scope such as `/subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{account}/providers/Microsoft.Authorization/roleAssignments/{name}` and assert the detected service is `Microsoft.Authorization/roleAssignments`.

- [x] **Step 2: Implement rightmost-provider Azure routing**

Route Azure extension resources through the rightmost `providers/{namespace}/{type}` segment while keeping resource-provider list/get/register routes under `Microsoft.Resources/providers`.

- [x] **Step 3: Write failing role assignment lifecycle tests**

Use API version `2022-04-01` to create, update, get, list, and delete role assignments at resource group and nested resource scopes.

- [x] **Step 4: Implement role assignment service**

Store assignments by scope and name, return ARM resource shape with `id`, `name`, `type`, and `properties`, support list `value`, and return `204` for idempotent deletes of missing assignments.

- [x] **Step 5: Write failing provider manifest test**

Assert `Microsoft.Authorization` appears in the Microsoft.Resources provider manifest with `roleAssignments` and API version `2022-04-01`.

- [x] **Step 6: Register service at startup**

Register `Microsoft.Authorization/roleAssignments@2022-04-01` with the versioned registry in `cmd/gateway/main.go`.

- [x] **Step 7: Run Authorization tests**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway ./pkg/routing ./services/azure/authorization ./services/azure/resources
```

Expected: all listed packages pass.

## Task 8: Azure Resource Tags At Scope

**Files:**
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn tags contract**

Use `Tags - Create Or Update At Scope` and `Tags - Get At Scope` API version `2021-04-01`. The route shape is `{scope}/providers/Microsoft.Resources/tags/default?api-version=2021-04-01`, and responses use the tags wrapper resource shape with `id`, `name`, `type`, and `properties.tags`.

- [x] **Step 2: Write failing resource-group tags test**

Create a resource group, call `PUT /subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/Microsoft.Resources/tags/default?api-version=2021-04-01`, assert `200 OK`, wrapper fields, and replacement of the resource group's existing tags.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestTagsCreateOrUpdateAndGetAtResourceGroupScope -count=1
```

Expected red result:

```text
expected create or update tags status 200, got 404; body={"error":{"code":"NotFound","message":"The provider route is not implemented."}}
```

- [x] **Step 4: Implement minimal tags-at-scope support**

Add `Microsoft.Resources/tags@2021-04-01` to `ServiceKeys`, add a `TagsResource` response type, and handle `PUT`/`GET` for resource-group scopes by replacing and reading the resource group's stored tags.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestTagsCreateOrUpdateAndGetAtResourceGroupScope -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
```

Expected: both commands pass.

## Task 9: Azure Generic ARM Resources

**Files:**
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `pkg/routing/registry.go`
- Modify: `pkg/routing/versioned_registry_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn generic resources contract**

Use `Resources - Create Or Update`, `Resources - Get By ID`, `Resources - List By Resource Group`, and `Resources - Delete` API version `2021-04-01`. First-slice support is resource-group scoped and stores the generic ARM resource shape for provider resources that do not yet have a first-class CloudMock Azure service.

- [x] **Step 2: Write failing generic resource lifecycle test**

Create a resource group, call `PUT /subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{name}?api-version=2021-04-01`, assert the returned `id`, `name`, `type`, `location`, `tags`, and `properties`, then verify `GET`, `GET /resources`, and `DELETE`.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestGenericResourceLifecycleAtResourceGroupScope -count=1
```

Expected red result:

```text
expected generic resource create status 201, got 404; body={"error":{"code":"NotFound","message":"The provider route is not implemented."}}
```

- [x] **Step 4: Implement minimal generic resource storage**

Add `GenericResource`, store resources by canonical ARM ID, route resource-group-scoped provider paths through Microsoft.Resources, and support create/get/list/delete for unsupported provider resources.

- [x] **Step 5: Write failing registry fallback test**

Register `Microsoft.Resources/resources@2021-04-01`, look up an unregistered Azure provider route such as `Microsoft.Network/virtualNetworks@2021-04-01`, and assert the generic resources service is returned while exact provider services continue to win first.

- [x] **Step 6: Verify registry fallback red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestRegistry_AzureGenericResourceFallbackHandlesUnregisteredProviderResources -count=1
```

Expected red result:

```text
routing: no service registered for provider="azure" service="microsoft.network/virtualnetworks" api-version="2021-04-01"
```

- [x] **Step 7: Implement Azure generic resources fallback**

After exact Azure versioned lookup fails, fall back to `Microsoft.Resources/resources` only for explicit API-version requests and only when the target is not already a `Microsoft.Resources/*` route.

- [x] **Step 8: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestGenericResourceLifecycleAtResourceGroupScope -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestRegistry_AzureGenericResourceFallbackHandlesUnregisteredProviderResources -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing
```

Expected: all commands pass.

## Task 10: Azure Resource Tags At Generic Resource Scope

**Files:**
- Modify: `services/azure/resources/service.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn resource-scope tags contract**

Use `Tags - Create Or Update At Scope` and `Tags - Get At Scope` API version `2021-04-01`. The resource-scope route is `{resourceId}/providers/Microsoft.Resources/tags/default?api-version=2021-04-01`, and the operation replaces the entire tag set for that resource.

- [x] **Step 2: Write failing generic-resource tags test**

Create a resource group and generic `Microsoft.Network/virtualNetworks` resource, call `PUT /subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{name}/providers/Microsoft.Resources/tags/default?api-version=2021-04-01`, assert the tags wrapper response, then verify `GET` and replacement of the generic resource's stored tags.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestTagsCreateOrUpdateAndGetAtGenericResourceScope -count=1
```

Expected red result:

```text
expected resource tags status 200, got 201; body={"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/providers/Microsoft.Resources/tags/default",...}
```

- [x] **Step 4: Implement resource-scope tag routing and storage updates**

Route the rightmost `providers/Microsoft.Resources/tags/default` extension path through the tags handler, resolve the preceding generic resource scope, and replace/read the generic resource's stored `tags` map.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestTagsCreateOrUpdateAndGetAtGenericResourceScope -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
```

Expected: both commands pass.

## Task 11: Azure Management Locks At Resource Group Scope

**Files:**
- Modify: `services/azure/authorization/types.go`
- Modify: `services/azure/authorization/service.go`
- Modify: `services/azure/authorization/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn management locks contract**

Use `Management Locks - Create Or Update At Resource Group Level`, `Get At Resource Group Level`, `List At Resource Group Level`, and `Delete At Resource Group Level` API version `2020-05-01`. The first slice supports resource-group scopes under `Microsoft.Authorization/locks`.

- [x] **Step 2: Write failing management-lock lifecycle test**

Call `PUT /subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/Microsoft.Authorization/locks/{lockName}?api-version=2020-05-01`, assert lock `id`, `name`, `type`, `properties.level`, and `properties.notes`, then verify update, get, list, and delete.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/authorization -run TestManagementLockLifecycleAtResourceGroupScope -count=1
```

Expected red result:

```text
expected create lock status 201, got 404; body={"error":{"code":"NotFound","message":"The role assignment route is not implemented."}}
```

- [x] **Step 4: Implement minimal management lock service support**

Add `ManagementLock`, `ManagementLockProperties`, `Microsoft.Authorization/locks@2020-05-01` service keys, and create/get/list/delete handling scoped by the ARM resource path before the `locks` provider segment.

- [x] **Step 5: Write failing provider manifest test**

Assert the `Microsoft.Authorization` provider manifest includes `locks` with API version `2020-05-01` alongside `roleAssignments`.

- [x] **Step 6: Verify manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected locks in authorization provider manifest, got map[roleAssignments:...]
```

- [x] **Step 7: Add locks to the provider manifest**

Expose `Microsoft.Authorization/locks@2020-05-01` from the built-in provider manifest so Azure SDKs and IaC clients can discover lock support.

- [x] **Step 8: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/authorization -run TestManagementLockLifecycleAtResourceGroupScope -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/authorization ./services/azure/resources
```

Expected: all commands pass.

## Task 12: Azure Storage Account Keys

**Files:**
- Modify: `services/azure/storage/types.go`
- Modify: `services/azure/storage/service.go`
- Modify: `services/azure/storage/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn storage key contracts**

Use `Storage Accounts - List Keys` and `Storage Accounts - Regenerate Key` API version `2024-01-01`. `listKeys` is a POST action under the storage account resource and returns `keys` with `keyName`, `permissions`, and base64 `value`; `regenerateKey` accepts `keyName` and returns the same key-list response shape after rotating the requested key.

- [x] **Step 2: Write failing list/regenerate test**

Create a storage account, call `POST /subscriptions/{subscriptionId}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{account}/listKeys?api-version=2024-01-01`, assert two full-permission keys, then call `POST .../regenerateKey` with `{"keyName":"key2"}` and assert `key1` remains stable while `key2` changes.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestStorageAccountListKeysAndRegenerateKey -count=1
```

Expected red result:

```text
expected list keys status 200, got 404; body={"error":{"code":"NotFound","message":"The storage account route is not implemented."}}
```

- [x] **Step 4: Implement deterministic storage account keys**

Add `StorageAccountListKeysResult`, deterministic `key1`/`key2` state per account, `listKeys` routing, and `regenerateKey` rotation for `key1` or `key2`.

- [x] **Step 5: Write failing provider manifest test**

Assert the `Microsoft.Storage/storageAccounts` provider manifest includes API version `2024-01-01` so clients can discover the same version implemented by the Storage service.

- [x] **Step 6: Verify manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected storageAccounts API versions to include 2024-01-01, got [2023-01-01 2022-09-01]
```

- [x] **Step 7: Add Storage 2024 API metadata**

Add `2024-01-01` to `Microsoft.Storage/storageAccounts` and `storageAccounts/blobServices` provider manifest entries.

- [x] **Step 8: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestStorageAccountListKeysAndRegenerateKey -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./services/azure/resources
```

Expected: all commands pass.

## Task 13: Azure Key Vault Keys And Crypto Operations

**Files:**
- Modify: `services/azure/keyvault/types.go`
- Modify: `services/azure/keyvault/service.go`
- Modify: `services/azure/keyvault/service_test.go`
- Modify: `pkg/routing/target.go`
- Modify: `pkg/routing/target_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Key Vault key contracts**

Use `Create Key`, `Encrypt`, and `Decrypt` API version `2025-07-01`. The first slice supports `POST /keys/{key-name}/create`, `POST /keys/{key-name}/{key-version}/encrypt`, and `POST /keys/{key-name}/{key-version}/decrypt` under the Key Vault data-plane host.

- [x] **Step 2: Write failing key create/encrypt/decrypt test**

Create an RSA key with `key_ops` containing `encrypt` and `decrypt`, assert the returned `key.kid`, `key.kty`, `key.key_ops`, and tags, then encrypt a base64url plaintext and decrypt it back to the original value.

- [x] **Step 3: Verify service red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault -run TestKeyCreateEncryptDecryptRoundTrip -count=1
```

Expected red result:

```text
expected create key status 200, got 404; body={"error":{"code":"NotFound","message":"The key vault secret route is not implemented."}}
```

- [x] **Step 4: Implement first-slice key storage and crypto operations**

Add `JsonWebKey`, `KeyBundle`, and `KeyOperationResult`, store latest key bundles by vault/key name, support create-key versions, and implement deterministic local encrypt/decrypt output that round-trips base64url values for tests.

- [x] **Step 5: Write failing data-plane routing test**

Assert `*.vault.azure.net/keys/...` routes to `Microsoft.KeyVault/keys` with `CreateKey`, `Encrypt`, and `Decrypt` actions instead of defaulting to `Microsoft.KeyVault/secrets`.

- [x] **Step 6: Verify routing red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureKeyVaultKeyDataPlane -count=1
```

Expected red result:

```text
expected: "Microsoft.KeyVault/keys"
actual  : "Microsoft.KeyVault/secrets"
```

- [x] **Step 7: Implement Key Vault keys target detection**

Inspect the first path segment on Key Vault data-plane hosts; route `keys` paths to `Microsoft.KeyVault/keys` and map `create`, `encrypt`, and `decrypt` POST actions.

- [x] **Step 8: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault -run TestKeyCreateEncryptDecryptRoundTrip -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureKeyVaultKeyDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault ./pkg/routing
```

Expected: all commands pass.

## Task 14: Azure Blob Range Reads

**Files:**
- Modify: `services/azure/storage/blob_handlers.go`
- Modify: `services/azure/storage/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Blob range contract**

Use `Get Blob` and `Specify the range header for Blob Storage operations`. A full `Get Blob` returns `200 OK`; a valid byte range returns `206 Partial Content`; `Range` and `x-ms-range` are supported, and `x-ms-range` takes precedence when both headers are present. The response includes `Content-Range`, `Content-Length`, and `Accept-Ranges: bytes`.

- [x] **Step 2: Write failing range-read test**

Create a container and blob, request the blob with both `Range: bytes=6-10` and `x-ms-range: bytes=0-4`, then assert `206`, body `hello`, `Content-Range: bytes 0-4/16`, `Content-Length: 5`, and `Accept-Ranges: bytes`.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobGetSupportsRangeAndXMSRangePrecedence -count=1
```

Expected red result:

```text
expected ranged blob status 206, got 200; body=hello azure blob
```

- [x] **Step 4: Implement minimal byte-range reads**

Parse `x-ms-range` first, fall back to `Range`, support `bytes=start-end`, clamp open-ended ranges to blob size, return `416` for invalid ranges, and preserve blob metadata/content type while returning partial content headers.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobGetSupportsRangeAndXMSRangePrecedence -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage
```

Expected: both commands pass.

## Task 15: Azure Blob Prefix And Delimiter Listing

**Files:**
- Modify: `services/azure/storage/blob_handlers.go`
- Modify: `services/azure/storage/service_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn List Blobs hierarchy contract**

Use [List Blobs](https://learn.microsoft.com/en-us/rest/api/storageservices/list-blobs). The first hierarchical listing slice supports `prefix`, `delimiter`, `NextMarker`, and `BlobPrefix` response entries for collapsed virtual directories.

- [x] **Step 2: Write failing prefix/delimiter test**

Create blobs named `docs/a.txt`, `docs/archive/b.txt`, and `images/c.png`, then call `GET ?restype=container&comp=list&prefix=docs/&delimiter=/`. Assert the response echoes `Prefix` and `Delimiter`, includes `docs/a.txt`, includes a `BlobPrefix` for `docs/archive/`, and excludes the unrelated and nested blobs.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobListSupportsPrefixAndDelimiter -count=1
```

Expected red result:

```text
expected prefix echo in response, got: <EnumerationResults ...><Name>docs/archive/b.txt</Name>...<Name>images/c.png</Name>...
```

- [x] **Step 4: Implement minimal hierarchical listing**

Filter list results by `prefix`, collapse the first `delimiter` boundary after the prefix into a unique `BlobPrefix`, sort blobs and prefixes together by name, preserve existing marker paging, and echo `Prefix`, `Marker`, `MaxResults`, and `Delimiter` where present.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobListSupportsPrefixAndDelimiter -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage
```

Expected: both commands pass.

## Task 16: Azure Queue Peek Messages

**Files:**
- Modify: `services/azure/storage/queue_handlers.go`
- Modify: `services/azure/storage/service_test.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Queue peek contract**

Use [Peek Messages](https://learn.microsoft.com/en-us/rest/api/storageservices/peek-messages) and [Get Messages](https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages). `GET /{queue}/messages?peekonly=true` returns visible messages without changing their visibility, without returning `PopReceipt`, and without returning `TimeNextVisible`.

- [x] **Step 2: Write failing peek-only test**

Create a queue, enqueue a visible message, call `GET /messages?peekonly=true&numofmessages=1`, assert the response includes the message text and omits `PopReceipt` and `TimeNextVisible`, then call normal `GET /messages` and assert the message is still receivable with a pop receipt.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestQueuePeekMessagesDoesNotHideOrLockMessage -count=1
```

Expected red result:

```text
peek response must not include PopReceipt, got: <QueueMessagesList>...
```

- [x] **Step 4: Implement non-mutating peek handling**

Route `peekonly=true` to a read-only Queue handler, filter to visible non-expired messages, cap `numofmessages` at 32, and emit the Azure peek response XML shape without `PopReceipt` or `TimeNextVisible`.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestQueuePeekMessagesDoesNotHideOrLockMessage -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage
```

Expected: both commands pass.

## Task 17: Azure Compute Virtual Machine Lifecycle

**Files:**
- Create: `services/azure/compute/service.go`
- Create: `services/azure/compute/types.go`
- Create: `services/azure/compute/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/architecture.md`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Compute VM contracts**

Use [Virtual Machines - Create Or Update](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/create-or-update?view=rest-compute-2025-11-01), [Virtual Machines - Get](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get?view=rest-compute-2025-11-01), [Virtual Machines - List](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/list?view=rest-compute-2025-11-01), and [Virtual Machines - Delete](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/delete?view=rest-compute-2025-11-01). The first slice supports resource-group-scoped VM create/get/list/delete for API version `2025-11-01`.

- [x] **Step 2: Write failing VM lifecycle test**

Create a VM with location, tags, hardware profile, OS profile, storage image reference, and network interface reference. Assert ARM identity fields, preserved hardware profile, `properties.provisioningState == "Succeeded"`, get/list behavior, accepted delete, and 404 after delete.

- [x] **Step 3: Verify service red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestVirtualMachineLifecycle -count=1
```

Expected red result:

```text
github.com/Viridian-Inc/cloudmock/services/azure/compute: no non-test Go files
```

- [x] **Step 4: Implement VM lifecycle service**

Add `ComputeService`, `VirtualMachine`, versioned service keys, ARM route parsing, in-memory VM storage, create/get/list/delete handlers, deterministic provisioning state, and ARM template provisioner support for `Microsoft.Compute/virtualMachines`.

- [x] **Step 5: Verify service green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestVirtualMachineLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute
```

Expected: both commands pass.

- [x] **Step 6: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Compute` exists and includes `virtualMachines` with API version `2025-11-01`.

- [x] **Step 7: Verify manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected get compute provider status 200, got 404; body={"error":{"code":"ProviderNotFound",...}}
```

- [x] **Step 8: Add Compute provider metadata and gateway wiring**

Add `Microsoft.Compute/virtualMachines@2025-11-01` to the provider manifest, include Compute in namespace normalization, register the Compute service in `cmd/gateway`, and attach it to Microsoft.Resources as a template provisioner.

- [x] **Step 9: Verify package wiring**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway
```

Expected: all commands pass.

## Task 18: Azure Compute Virtual Machine Instance View

**Files:**
- Modify: `services/azure/compute/service.go`
- Modify: `services/azure/compute/types.go`
- Modify: `services/azure/compute/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn VM instance-view contract**

Use [Virtual Machines - Get](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get?view=rest-compute-2025-11-01). The first slice supports `$expand=instanceView`, which retrieves runtime VM status in the returned VM model.

- [x] **Step 2: Write failing instance-view expansion test**

Create a VM, call `GET .../virtualMachines/{vmName}?api-version=2025-11-01&$expand=instanceView`, and assert `properties.instanceView.statuses` includes deterministic provisioning and power-state status entries.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestVirtualMachineGetExpandsInstanceView -count=1
```

Expected red result:

```text
expected properties.instanceView in expanded response
```

- [x] **Step 4: Implement deterministic instance-view projection**

Detect `$expand=instanceView` on VM get, clone the VM properties for the response, and add `ProvisioningState/succeeded` plus `PowerState/running` status entries without mutating the stored base model.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestVirtualMachineGetExpandsInstanceView -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute
```

Expected: both commands pass.

## Task 19: Azure Compute Virtual Machine Start And Deallocate

**Files:**
- Modify: `services/azure/compute/service.go`
- Modify: `services/azure/compute/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn VM operation contracts**

Use [Virtual Machines - Start](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/start?view=rest-compute-2025-11-01) and [Virtual Machines - Deallocate](https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/deallocate?view=rest-compute-2025-11-01). The first slice accepts the documented `POST` operation routes and updates deterministic local power state for instance-view compatibility.

- [x] **Step 2: Write failing state-transition test**

Create a VM, call `POST .../deallocate`, assert `202 Accepted`, verify `$expand=instanceView` returns `PowerState/deallocated`, call `POST .../start`, assert `202 Accepted`, and verify instance view returns `PowerState/running`.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestVirtualMachineStartAndDeallocateUpdateInstanceViewPowerState -count=1
```

Expected red result:

```text
expected deallocate status 202, got 404
```

- [x] **Step 4: Implement VM operation routes and power state**

Allow operation path segments after VM name, route `start` and `deallocate`, maintain a per-VM power-state store, return `202 Accepted` with operation headers, and project that state into `properties.instanceView.statuses`.

- [x] **Step 5: Verify green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestVirtualMachineStartAndDeallocateUpdateInstanceViewPowerState -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute
```

Expected: both commands pass.

## Task 20: Azure Compute Managed Disks

**Files:**
- Modify: `services/azure/compute/service.go`
- Modify: `services/azure/compute/types.go`
- Modify: `services/azure/compute/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn managed disk contracts**

Use [Disks - Create Or Update](https://learn.microsoft.com/en-us/rest/api/compute/disks/create-or-update?view=rest-compute-2025-01-02), [Disks - Get](https://learn.microsoft.com/en-us/rest/api/compute/disks/get?view=rest-compute-2025-01-02), [Disks - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/compute/disks/list-by-resource-group?view=rest-compute-2025-01-02), and [Disks - Delete](https://learn.microsoft.com/en-us/rest/api/compute/disks/delete?view=rest-compute-2025-01-02). The first slice supports synchronous in-memory disk CRUD while preserving request SKU, zones, tags, creation data, disk size, and OS type.

- [x] **Step 2: Write failing managed disk lifecycle test**

Create `PUT .../providers/Microsoft.Compute/disks/{diskName}?api-version=2025-01-02`, assert Azure resource identity fields, preserved SKU and properties, deterministic `Succeeded`/`Unattached` state, `GET`, `LIST`, `DELETE`, and deleted-resource `404`.

- [x] **Step 3: Verify red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestManagedDiskLifecycle -count=1
```

Expected red result:

```text
expected create disk status 201, got 404
```

- [x] **Step 4: Implement disk routes and in-memory lifecycle**

Add `Disk`, disk route parsing, create/get/list/delete handlers, a versioned `Microsoft.Compute/disks@2025-01-02` service key, and ARM template provisioning support for `Microsoft.Compute/disks`.

- [x] **Step 5: Verify compute green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute -run TestManagedDiskLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/compute
```

Expected: both commands pass.

- [x] **Step 6: Write failing provider manifest assertion**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Compute` exposes `disks` with API version `2025-01-02`.

- [x] **Step 7: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected disks in compute provider manifest
```

- [x] **Step 8: Add disk provider metadata**

Add `Microsoft.Compute/disks@2025-01-02` to the provider manifest with the same first-slice locations and tag capability as virtual machines.

- [x] **Step 9: Verify resources green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
```

Expected: both commands pass.

## Task 21: Azure Functions App Service Control Plane

**Files:**
- Add: `services/azure/appservice/service.go`
- Add: `services/azure/appservice/types.go`
- Add: `services/azure/appservice/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn App Service contracts**

Use [Web Apps - Create Or Update](https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/create-or-update?view=rest-appservice-2024-04-01), [Web Apps - Get](https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/get?view=rest-appservice-2024-04-01), [Web Apps - Delete](https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/delete?view=rest-appservice-2024-04-01), [Web Apps - List Functions](https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/list-functions?view=rest-appservice-2024-04-01), and [App Service Plans - Create Or Update](https://learn.microsoft.com/en-us/rest/api/appservice/app-service-plans/create-or-update?view=rest-appservice-2024-04-01). The first slice models Azure Functions through `Microsoft.Web/serverfarms` and `Microsoft.Web/sites` ARM resources.

- [x] **Step 2: Write failing Function App lifecycle test**

Create an App Service plan, create a Function App site with `kind=functionapp,linux`, preserve identity/tags/site config/server farm references, list the site, list seeded function metadata, delete the site, and assert a deleted-resource `404`.

- [x] **Step 3: Verify initial red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestFunctionAppPlanSiteAndFunctionMetadataLifecycle -count=1
```

Expected first red result:

```text
undefined: New
```

- [x] **Step 4: Add minimal scaffold and verify behavioral red**

Add a minimal `Microsoft.Web` service scaffold that compiles but returns an Azure ARM `404` for every route.

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestFunctionAppPlanSiteAndFunctionMetadataLifecycle -count=1
```

Expected behavioral red result:

```text
expected create app service plan status 201, got 404
```

- [x] **Step 5: Implement App Service plan, Function App, and function metadata lifecycle**

Add `AppServicePlan`, `Site`, and `Function` resource shapes; implement `Microsoft.Web/serverfarms` and `Microsoft.Web/sites` route parsing; support plan/site create/get/list/delete; support `GET sites/{name}/functions`; return deterministic `Ready`, `Running`, and `Succeeded` states; add versioned service keys and ARM template provisioning hooks.

- [x] **Step 6: Verify App Service green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestFunctionAppPlanSiteAndFunctionMetadataLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice
```

Expected: both commands pass.

- [x] **Step 7: Write failing provider manifest assertion**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Web` exposes `serverfarms` and `sites` with API version `2024-04-01`.

- [x] **Step 8: Verify manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.Web" could not be found.
```

- [x] **Step 9: Add provider metadata and gateway wiring**

Add `Microsoft.Web/serverfarms@2024-04-01` and `Microsoft.Web/sites@2024-04-01` to the provider manifest, normalize `Microsoft.Web`, instantiate/register the App Service package in `cmd/gateway`, and register it as a Microsoft.Resources ARM template provisioner.

- [x] **Step 10: Verify package and gateway green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice ./cmd/gateway -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing ./pkg/gateway -count=1
```

Expected: all commands pass.

## Task 22: Azure Network Virtual Networks And NSGs

**Files:**
- Add: `services/azure/network/service.go`
- Add: `services/azure/network/types.go`
- Add: `services/azure/network/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Network contracts**

Use [Virtual Networks - Create Or Update](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/create-or-update?view=rest-virtualnetwork-2025-05-01), [Virtual Networks - Get](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/get?view=rest-virtualnetwork-2025-05-01), [Network Security Groups - Create Or Update](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/create-or-update?view=rest-virtualnetwork-2025-05-01), and [Network Security Groups - Get](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/get?view=rest-virtualnetwork-2025-05-01). Microsoft Learn redirects the older `rest-virtualnetwork-2023-09-01` moniker to `2025-05-01`, so the service registers both versions for compatibility.

- [x] **Step 2: Write failing VNet and NSG lifecycle test**

Create a virtual network with address space and subnet metadata, assert the ARM identity fields, deterministic `Succeeded` state, projected subnet IDs, `GET`, and `LIST`; create a network security group with a rule, assert rule ID projection, `GET`, `LIST`, and delete both resources.

- [x] **Step 3: Verify initial red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestVirtualNetworkAndNetworkSecurityGroupLifecycle -count=1
```

Expected first red result:

```text
undefined: New
```

- [x] **Step 4: Add minimal scaffold and verify behavioral red**

Add a minimal `Microsoft.Network` service scaffold that compiles but returns an Azure ARM `404` for every route.

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestVirtualNetworkAndNetworkSecurityGroupLifecycle -count=1
```

Expected behavioral red result:

```text
expected create virtual network status 201, got 404
```

- [x] **Step 5: Implement Network lifecycle behavior**

Add `VirtualNetwork` and `NetworkSecurityGroup` resource shapes; implement route parsing for `virtualNetworks` and `networkSecurityGroups`; support create/get/list/delete; preserve request properties and tags; project subnet/security-rule child IDs; return deterministic `Succeeded` state; register `2025-05-01` and `2023-09-01`; add ARM template provisioning hooks.

- [x] **Step 6: Verify Network green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestVirtualNetworkAndNetworkSecurityGroupLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/network
```

Expected: both commands pass.

- [x] **Step 7: Write failing provider manifest version assertion**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Network` exposes `virtualNetworks` and `networkSecurityGroups` with both `2025-05-01` and `2023-09-01`.

- [x] **Step 8: Verify manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected virtualNetworks API versions to include 2025-05-01
```

- [x] **Step 9: Add provider metadata and gateway wiring**

Add `Microsoft.Network/virtualNetworks` and `Microsoft.Network/networkSecurityGroups` provider metadata for both API versions, instantiate/register the Network service in `cmd/gateway`, and register it as a Microsoft.Resources ARM template provisioner.

- [x] **Step 10: Verify package and gateway green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway
```

Expected: all commands pass.

## Task 23: Azure Service Bus Namespace And Queue Management

**Files:**
- Add: `services/azure/servicebus/service.go`
- Add: `services/azure/servicebus/types.go`
- Add: `services/azure/servicebus/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Service Bus control-plane contracts**

Use [Namespaces - Create Or Update](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/namespaces/create-or-update?view=rest-servicebus-controlplane-2024-01-01), [Queues - Create Or Update](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/queues/create-or-update?view=rest-servicebus-controlplane-2024-01-01), [Topics - Create Or Update](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/topics/create-or-update?view=rest-servicebus-controlplane-2024-01-01), and [Subscriptions - Create Or Update](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/subscriptions/create-or-update?view=rest-servicebus-controlplane-2024-01-01). The first slice implements namespaces and queues while advertising topic/subscription provider metadata for the next messaging slice.

- [x] **Step 2: Write failing namespace and queue lifecycle test**

Create a namespace with SKU, tags, and zone-redundant properties; assert ARM identity, service bus endpoint, `Active` status, and `Succeeded` provisioning state; create a namespace-scoped queue with delivery properties; assert queue identity, preserved properties, list behavior, and delete behavior.

- [x] **Step 3: Verify initial red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestNamespaceAndQueueLifecycle -count=1
```

Expected first red result:

```text
undefined: New
```

- [x] **Step 4: Add minimal scaffold and verify behavioral red**

Add a minimal `Microsoft.ServiceBus` service scaffold that compiles but returns an Azure ARM `404` for every route.

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestNamespaceAndQueueLifecycle -count=1
```

Expected behavioral red result:

```text
expected create namespace status 201, got 404
```

- [x] **Step 5: Implement namespace and queue management**

Add `Namespace` and `Queue` resource shapes; implement `namespaces` and `namespaces/{namespace}/queues` route parsing; support create/get/list/delete; preserve request properties and tags; project service bus endpoint/status/provisioning fields; register `Microsoft.ServiceBus/namespaces@2024-01-01`; add ARM template provisioning hooks for namespaces and queues.

- [x] **Step 6: Verify Service Bus green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestNamespaceAndQueueLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus
```

Expected: both commands pass.

- [x] **Step 7: Write failing provider manifest assertion**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.ServiceBus` exposes `namespaces`, `namespaces/queues`, `namespaces/topics`, and `namespaces/topics/subscriptions` with API version `2024-01-01`.

- [x] **Step 8: Verify manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.ServiceBus" could not be found.
```

- [x] **Step 9: Add provider metadata and gateway wiring**

Add `Microsoft.ServiceBus` provider metadata for namespaces, queues, topics, and subscriptions; normalize the namespace; instantiate/register the Service Bus service in `cmd/gateway`; and register it as a Microsoft.Resources ARM template provisioner.

- [x] **Step 10: Verify package and gateway green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway
```

Expected: all commands pass.

## Task 24: Azure Service Bus Topic And Subscription Management

**Files:**
- Modify: `services/azure/servicebus/service.go`
- Modify: `services/azure/servicebus/types.go`
- Modify: `services/azure/servicebus/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Service Bus topic and subscription contracts**

Use [Topics - Create Or Update](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/topics/create-or-update?view=rest-servicebus-controlplane-2024-01-01), [Topics - List By Namespace](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/topics/list-by-namespace?view=rest-servicebus-controlplane-2024-01-01), [Topics - Get](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/topics/get?view=rest-servicebus-controlplane-2024-01-01), [Topics - Delete](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/topics/delete?view=rest-servicebus-controlplane-2024-01-01), [Subscriptions - Create Or Update](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/subscriptions/create-or-update?view=rest-servicebus-controlplane-2024-01-01), [Subscriptions - List By Topic](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/subscriptions/list-by-topic?view=rest-servicebus-controlplane-2024-01-01), [Subscriptions - Get](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/subscriptions/get?view=rest-servicebus-controlplane-2024-01-01), and [Subscriptions - Delete](https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane/subscriptions/delete?view=rest-servicebus-controlplane-2024-01-01). This slice implements ARM management lifecycle for topics and subscriptions; runtime message send/receive and subscription rules remain separate checklist items.

- [x] **Step 2: Write failing topic/subscription lifecycle test**

Create a namespace, create a namespace-scoped topic, assert ARM ID/name/type, preserved properties, `Active` status, and `Succeeded` provisioning state; list topics; create a topic-scoped subscription, assert ARM ID/name/type and preserved properties; list subscriptions; delete the subscription and topic.

- [x] **Step 3: Verify lifecycle red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestTopicSubscriptionLifecycle -count=1
```

Expected red result:

```text
expected create topic status 201, got 200
```

The failure proves `topics/{name}` was incorrectly routed through namespace update behavior before this task.

- [x] **Step 4: Implement topic and subscription control-plane routes**

Add `Topic` and `Subscription` resource shapes; extend route parsing to support `namespaces/{namespace}/topics/{topic}` and `namespaces/{namespace}/topics/{topic}/subscriptions/{subscription}`; implement create/get/list/delete; preserve request properties; return deterministic `Active` and `Succeeded` fields; cascade subscription deletion on topic and namespace deletion.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestTopicSubscriptionLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus
```

Expected: both commands pass.

- [x] **Step 6: Write failing ARM template provisioning test**

Provision a namespace, topic, and subscription through `ProvisionTemplateResource`; assert topic/subscription ARM IDs and resource types.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestTopicSubscriptionTemplateProvisioning -count=1
```

Expected red result:

```text
unsupported Service Bus template resource type "Microsoft.ServiceBus/namespaces/topics"
```

- [x] **Step 8: Add ARM template support for topic and subscription resources**

Extend `SupportsTemplateResource` and `ProvisionTemplateResource` for `Microsoft.ServiceBus/namespaces/topics` with `{namespace}/{topic}` names and `Microsoft.ServiceBus/namespaces/topics/subscriptions` with `{namespace}/{topic}/{subscription}` names.

- [x] **Step 9: Verify Service Bus package green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestTopicSubscriptionTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus
```

Expected: both commands pass.

## Task 25: Azure Network Subnet And Security Rule Management

**Files:**
- Modify: `services/azure/network/service.go`
- Modify: `services/azure/network/types.go`
- Modify: `services/azure/network/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Network child-resource contracts**

Use [Subnets - Create Or Update](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/create-or-update?view=rest-virtualnetwork-2025-05-01), [Subnets - Get](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get?view=rest-virtualnetwork-2025-05-01), [Subnets - List](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/list?view=rest-virtualnetwork-2025-05-01), [Subnets - Delete](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/delete?view=rest-virtualnetwork-2025-05-01), [Security Rules - Create Or Update](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/security-rules/create-or-update?view=rest-virtualnetwork-2025-05-01), [Security Rules - Get](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/security-rules/get?view=rest-virtualnetwork-2025-05-01), [Security Rules - List](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/security-rules/list?view=rest-virtualnetwork-2025-05-01), and [Security Rules - Delete](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/security-rules/delete?view=rest-virtualnetwork-2025-05-01). This slice implements direct child-resource lifecycle for the existing Network service.

- [x] **Step 2: Write failing subnet and security-rule lifecycle tests**

Create parent VNet and NSG resources, then exercise `virtualNetworks/{vnet}/subnets/{subnet}` and `networkSecurityGroups/{nsg}/securityRules/{rule}` with create/list/delete assertions, stored properties, deterministic `Succeeded` state, and parent-resource synchronization.

- [x] **Step 3: Verify child-resource route red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestSubnetLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestSecurityRuleLifecycle -count=1
```

Expected red result:

```text
expected create subnet status 201, got 404
expected create security rule status 201, got 404
```

- [x] **Step 4: Implement subnet and security-rule routes**

Add `Subnet` and `SecurityRule` resource shapes; extend Network route parsing to support direct child resources; store child resources independently; synchronize parent VNet/NSG child collections on create and delete; cascade child deletion when parents are deleted.

- [x] **Step 5: Verify child-resource lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestSubnetLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestSecurityRuleLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/network
```

Expected: all commands pass.

- [x] **Step 6: Write failing template and provider manifest tests**

Add `TestNetworkChildTemplateProvisioning` for subnet and security-rule ARM template resources, and extend `TestProviderManifestListGetAndRegister` to assert `virtualNetworks/subnets` and `networkSecurityGroups/securityRules`.

- [x] **Step 7: Verify template and manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestNetworkChildTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
unsupported Network template resource type "Microsoft.Network/virtualNetworks/subnets"
expected virtualNetworks/subnets in network provider manifest
```

- [x] **Step 8: Add template and provider manifest support**

Extend Network ARM template provisioning for `Microsoft.Network/virtualNetworks/subnets` and `Microsoft.Network/networkSecurityGroups/securityRules`; add provider manifest entries for both child types with API versions `2025-05-01` and `2023-09-01`.

- [x] **Step 9: Verify Network and Resources green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestNetworkChildTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/network
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
```

Expected: all commands pass.

## Task 26: Azure Event Grid Topic And Event Subscription Management

**Files:**
- Add: `services/azure/eventgrid/service.go`
- Add: `services/azure/eventgrid/types.go`
- Add: `services/azure/eventgrid/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Event Grid control-plane contracts**

Use [Topics - Create Or Update](https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/create-or-update?view=rest-eventgrid-controlplane-2025-02-15), [Topics - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15), [Topic Event Subscriptions - Create Or Update](https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15), and [Topic Event Subscriptions - List](https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/list?view=rest-eventgrid-controlplane-2025-02-15). The Event Grid REST TOC currently exposes stable control-plane docs under `rest-eventgrid-controlplane-2025-02-15`.

- [x] **Step 2: Write failing topic/event-subscription lifecycle test**

Create a custom topic with tags and schema properties; assert ARM identity, deterministic endpoint projection, and `Succeeded` state; list topics; create a topic-scoped event subscription with destination, filter, labels, and retry policy; list and delete the event subscription; delete the topic.

- [x] **Step 3: Verify compile and behavioral red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run TestTopicAndEventSubscriptionLifecycle -count=1
```

Expected red results:

```text
undefined: New
expected create topic status 201, got 404
```

- [x] **Step 4: Implement Event Grid topic and topic event-subscription lifecycle**

Add `Topic` and `EventSubscription` shapes; implement route parsing for `Microsoft.EventGrid/topics` and `topics/{topic}/eventSubscriptions`; preserve request properties and tags; project deterministic topic endpoints and provisioning state; cascade event subscription deletion on topic delete.

- [x] **Step 5: Verify Event Grid lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run TestTopicAndEventSubscriptionLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid
```

Expected: both commands pass.

- [x] **Step 6: Write failing template and provider manifest tests**

Add `TestTopicEventSubscriptionTemplateProvisioning`; extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.EventGrid/topics` and `Microsoft.EventGrid/topics/eventSubscriptions` expose API version `2025-02-15`.

- [x] **Step 7: Verify template and manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run TestTopicEventSubscriptionTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
unsupported Event Grid template resource type
The resource provider "Microsoft.EventGrid" could not be found.
```

- [x] **Step 8: Add template, provider manifest, and gateway support**

Extend Event Grid ARM template provisioning for `Microsoft.EventGrid/topics` and `Microsoft.EventGrid/topics/eventSubscriptions`; add provider manifest entries; normalize `Microsoft.EventGrid`; instantiate and register Event Grid in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 9: Verify Event Grid wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run TestTopicEventSubscriptionTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway
```

Expected: all commands pass.

## Task 27: Azure Event Hubs Namespace And Event Hub Management

**Files:**
- Add: `services/azure/eventhub/service.go`
- Add: `services/azure/eventhub/types.go`
- Add: `services/azure/eventhub/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Event Hubs control-plane contracts**

Use [Namespaces - Create Or Update](https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/create-or-update?view=rest-eventhub-2026-01-01), [Namespaces - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-by-resource-group?view=rest-eventhub-2026-01-01), [Event Hubs - Create Or Update](https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/create-or-update?view=rest-eventhub-2026-01-01), and [Event Hubs - List By Namespace](https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/list-by-namespace?view=rest-eventhub-2026-01-01). The Event Hubs REST TOC currently exposes stable management-plane docs under `rest-eventhub-2026-01-01`, with `2024-01-01` kept as a compatibility API version.

- [x] **Step 2: Write failing namespace/event hub lifecycle test**

Create an Event Hubs namespace with SKU, tags, Kafka, zone redundancy, and public network access properties; assert ARM identity, deterministic service bus endpoint projection, `Active`/`Succeeded` state, list behavior, namespace-scoped event hub create/get/list/delete, property preservation, and namespace deletion.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestNamespaceAndEventHubLifecycle -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement Event Hubs namespace and event hub lifecycle**

Add `Namespace` and `EventHub` shapes; implement route parsing for `Microsoft.EventHub/namespaces` and `namespaces/{namespace}/eventhubs`; preserve request properties, SKU, and tags; project deterministic service bus endpoint, status, and provisioning state; cascade event hub deletion on namespace delete; register service keys for `2026-01-01` and `2024-01-01`.

- [x] **Step 5: Verify Event Hubs lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestNamespaceAndEventHubLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub
```

Expected: both commands pass.

- [x] **Step 6: Add template and provider manifest tests**

Add `TestNamespaceEventHubTemplateProvisioning`; extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.EventHub/namespaces` and `Microsoft.EventHub/namespaces/eventhubs` expose API versions `2026-01-01` and `2024-01-01`.

- [x] **Step 7: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.EventHub" could not be found.
```

- [x] **Step 8: Add template, provider manifest, and gateway support**

Extend Event Hubs ARM template provisioning for `Microsoft.EventHub/namespaces` and `Microsoft.EventHub/namespaces/eventhubs`; add provider manifest entries; normalize `Microsoft.EventHub`; instantiate and register Event Hubs in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 9: Verify Event Hubs wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 28: Azure Container Registry Registry Management

**Files:**
- Add: `services/azure/containerregistry/service.go`
- Add: `services/azure/containerregistry/types.go`
- Add: `services/azure/containerregistry/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Container Registry control-plane contracts**

Use [Registries - Create](https://learn.microsoft.com/en-us/rest/api/container-registry/registries/create?view=rest-container-registry-2025-11-01), [Registries - Get](https://learn.microsoft.com/en-us/rest/api/container-registry/registries/get?view=rest-container-registry-2025-11-01), [Registries - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/container-registry/registries/list-by-resource-group?view=rest-container-registry-2025-11-01), [Registries - List Credentials](https://learn.microsoft.com/en-us/rest/api/container-registry/registries/list-credentials?view=rest-container-registry-2025-11-01), and [Registries - Regenerate Credential](https://learn.microsoft.com/en-us/rest/api/container-registry/registries/regenerate-credential?view=rest-container-registry-2025-11-01). The Azure REST TOC currently exposes the latest stable ACR docs under `rest-container-registry-2025-11-01`; `2023-07-01` remains registered as a compatibility API version.

- [x] **Step 2: Write failing registry lifecycle and credential tests**

Create a registry with SKU, tags, admin user, and public network access properties; assert ARM identity, deterministic login server projection, `Succeeded` state, list behavior, deterministic `listCredentials`, `regenerateCredential` rotation, and deletion.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestRegistryLifecycleAndCredentials -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement registry lifecycle and credentials**

Add `Registry` shape; implement route parsing for `Microsoft.ContainerRegistry/registries`; preserve request properties, SKU, identity, and tags; project deterministic login server and provisioning state; implement deterministic `listCredentials` and named credential rotation for `password` and `password2`; register service keys for `2025-11-01` and `2023-07-01`.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestRegistryLifecycleAndCredentials -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Add `TestRegistryTemplateProvisioning` for `Microsoft.ContainerRegistry/registries`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestRegistryTemplateProvisioning -count=1
```

Expected red result:

```text
svc.ProvisionTemplateResource undefined
```

- [x] **Step 8: Add template provisioning**

Extend ACR service with `SupportsTemplateResource` and `ProvisionTemplateResource` for `Microsoft.ContainerRegistry/registries`.

- [x] **Step 9: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.ContainerRegistry/registries` exposes API versions `2025-11-01` and `2023-07-01`.

- [x] **Step 10: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.ContainerRegistry" could not be found.
```

- [x] **Step 11: Add provider manifest and gateway support**

Add provider manifest entries; normalize `Microsoft.ContainerRegistry`; instantiate and register ACR in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 12: Verify ACR wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 29: Azure DNS Zone And Record Set Management

**Files:**
- Add: `services/azure/dns/service.go`
- Add: `services/azure/dns/types.go`
- Add: `services/azure/dns/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn DNS contracts**

Use [Zones - Create Or Update](https://learn.microsoft.com/en-us/rest/api/dns/zones/create-or-update?view=rest-dns-2018-05-01), [Zones - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/dns/zones/list-by-resource-group?view=rest-dns-2018-05-01), [Record Sets - Create Or Update](https://learn.microsoft.com/en-us/rest/api/dns/record-sets/create-or-update?view=rest-dns-2018-05-01), and [Record Sets - List By DNS Zone](https://learn.microsoft.com/en-us/rest/api/dns/record-sets/list-by-dns-zone?view=rest-dns-2018-05-01). Azure public DNS management is exposed under `Microsoft.Network/dnsZones` with API version `2018-05-01`.

- [x] **Step 2: Write failing zone and record-set lifecycle test**

Create a DNS zone with tags and `Public` zone type; assert ARM identity and deterministic `Succeeded` state; list zones; create an A record set with TTL, records, and metadata; list and delete record sets; delete the zone.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/dns -run TestZoneAndRecordSetLifecycle -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement DNS zone and record-set lifecycle**

Add `Zone` and `RecordSet` shapes; implement route parsing for `Microsoft.Network/dnsZones`, `/recordsets`, and record-type child routes; preserve request properties and tags; project deterministic provisioning state; cascade record-set deletion on zone delete.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/dns -run TestZoneAndRecordSetLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Add `TestZoneRecordSetTemplateProvisioning` for `Microsoft.Network/dnsZones` and `Microsoft.Network/dnsZones/A`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/dns -run TestZoneRecordSetTemplateProvisioning -count=1
```

Expected red result:

```text
svc.ProvisionTemplateResource undefined
```

- [x] **Step 8: Add template support**

Extend DNS service with `SupportsTemplateResource` and `ProvisionTemplateResource` for zones and record-set resource types.

- [x] **Step 9: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Network/dnsZones` and common record-set types expose API version `2018-05-01`.

- [x] **Step 10: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected dnsZones in network provider manifest
```

- [x] **Step 11: Add provider manifest and gateway support**

Add provider manifest entries for DNS zones and common record-set child types; instantiate and register Azure DNS in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 12: Verify DNS wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/dns ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 30: Azure Cosmos DB Account, SQL Database, And SQL Container Management

**Files:**
- Add: `services/azure/cosmosdb/service.go`
- Add: `services/azure/cosmosdb/types.go`
- Add: `services/azure/cosmosdb/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Cosmos DB resource-provider contracts**

Use [Database Accounts - Create Or Update](https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts/create-or-update?view=rest-cosmos-db-resource-provider-2025-05-01), [Database Accounts - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts/list-by-resource-group?view=rest-cosmos-db-resource-provider-2025-05-01), [SQL Resources - Create Update SQL Database](https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/sql-resources/create-update-sql-database?view=rest-cosmos-db-resource-provider-2025-05-01), and [SQL Resources - Create Update SQL Container](https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/sql-resources/create-update-sql-container?view=rest-cosmos-db-resource-provider-2025-05-01). The first slice targets `2025-05-01` and registers `2024-05-15` as a compatibility version.

- [x] **Step 2: Write failing account/database/container lifecycle test**

Create a database account with locations and consistency policy; assert ARM identity, deterministic document endpoint, and `Succeeded` provisioning state; list accounts; create SQL database and SQL container resources; list containers; delete a container.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestAccountSQLDatabaseAndContainerLifecycle -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement Cosmos DB lifecycle**

Add account, SQL database, and SQL container shapes; implement route parsing for `Microsoft.DocumentDB/databaseAccounts`, `sqlDatabases`, and `containers`; preserve request properties, options, and tags; project deterministic document endpoint and provisioning state; cascade child deletion.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestAccountSQLDatabaseAndContainerLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Add `TestSQLDatabaseAndContainerTemplateProvisioning` for `Microsoft.DocumentDB/databaseAccounts`, `Microsoft.DocumentDB/databaseAccounts/sqlDatabases`, and `Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestSQLDatabaseAndContainerTemplateProvisioning -count=1
```

Expected red result:

```text
svc.ProvisionTemplateResource undefined
```

- [x] **Step 8: Add template support**

Extend Cosmos DB service with `SupportsTemplateResource` and `ProvisionTemplateResource` for account, database, and container resources.

- [x] **Step 9: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.DocumentDB` exposes `databaseAccounts`, `databaseAccounts/sqlDatabases`, and `databaseAccounts/sqlDatabases/containers` for API versions `2025-05-01` and `2024-05-15`.

- [x] **Step 10: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.DocumentDB" could not be found.
```

- [x] **Step 11: Add provider manifest and gateway support**

Add provider manifest entries; normalize `Microsoft.DocumentDB`; instantiate and register Cosmos DB in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 12: Verify Cosmos wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 31: Azure SQL Server, Database, And Firewall Rule Management

**Files:**
- Add: `services/azure/sql/service.go`
- Add: `services/azure/sql/types.go`
- Add: `services/azure/sql/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn SQL resource-provider contracts**

Use [Servers - Create Or Update](https://learn.microsoft.com/en-us/rest/api/sql/servers/create-or-update?view=rest-sql-2025-01-01), [Servers - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/sql/servers/list-by-resource-group?view=rest-sql-2025-01-01), [Databases - Create Or Update](https://learn.microsoft.com/en-us/rest/api/sql/databases/create-or-update?view=rest-sql-2025-01-01), [Databases - List By Server](https://learn.microsoft.com/en-us/rest/api/sql/databases/list-by-server?view=rest-sql-2025-01-01), [Firewall Rules - Create Or Update](https://learn.microsoft.com/en-us/rest/api/sql/firewall-rules/create-or-update?view=rest-sql-2025-01-01), and [Firewall Rules - List By Server](https://learn.microsoft.com/en-us/rest/api/sql/firewall-rules/list-by-server?view=rest-sql-2025-01-01). The first slice targets `2025-01-01` and registers `2023-08-01` as a compatibility version.

- [x] **Step 2: Write failing server/database/firewall lifecycle test**

Create a logical server with admin and TLS properties; assert ARM identity, deterministic fully qualified domain name, `Ready` state, and `Succeeded` provisioning state; list servers; create a database with SKU and collation metadata; list databases; create and list a firewall rule; delete the child resources.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/sql -run TestServerDatabaseAndFirewallRuleLifecycle -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement SQL lifecycle**

Add server, database, and firewall rule shapes; implement route parsing for `Microsoft.Sql/servers`, `databases`, and `firewallRules`; preserve request properties, SKU, tags, and location; project deterministic fully qualified domain name, status, and provisioning state; cascade child deletion when a server is deleted.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/sql -run TestServerDatabaseAndFirewallRuleLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Add `TestServerDatabaseAndFirewallRuleTemplateProvisioning` for `Microsoft.Sql/servers`, `Microsoft.Sql/servers/databases`, and `Microsoft.Sql/servers/firewallRules`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/sql -run TestServerDatabaseAndFirewallRuleTemplateProvisioning -count=1
```

Expected red result:

```text
svc.ProvisionTemplateResource undefined
```

- [x] **Step 8: Add template support**

Extend SQL service with `SupportsTemplateResource` and `ProvisionTemplateResource` for server, database, and firewall-rule resources.

- [x] **Step 9: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Sql` exposes `servers`, `servers/databases`, and `servers/firewallRules` for API versions `2025-01-01` and `2023-08-01`.

- [x] **Step 10: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.Sql" could not be found.
```

- [x] **Step 11: Add provider manifest and gateway support**

Add provider manifest entries; normalize `Microsoft.Sql`; instantiate and register SQL in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 12: Verify SQL wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/sql ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 32: Azure PostgreSQL Flexible Server, Database, And Firewall Rule Management

**Files:**
- Add: `services/azure/postgresql/service.go`
- Add: `services/azure/postgresql/types.go`
- Add: `services/azure/postgresql/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn PostgreSQL resource-provider contracts**

Use [Flexible Servers - Create](https://learn.microsoft.com/en-us/rest/api/postgresql/servers/create?view=rest-postgresql-2025-08-01), [Flexible Servers - Get](https://learn.microsoft.com/en-us/rest/api/postgresql/servers/get?view=rest-postgresql-2025-08-01), [Flexible Servers - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/postgresql/servers/list-by-resource-group?view=rest-postgresql-2025-08-01), [Databases - Create](https://learn.microsoft.com/en-us/rest/api/postgresql/databases/create?view=rest-postgresql-2025-08-01), [Databases - List By Server](https://learn.microsoft.com/en-us/rest/api/postgresql/databases/list-by-server?view=rest-postgresql-2025-08-01), [Firewall Rules - Create Or Update](https://learn.microsoft.com/en-us/rest/api/postgresql/firewall-rules/create-or-update?view=rest-postgresql-2025-08-01), and [Firewall Rules - List By Server](https://learn.microsoft.com/en-us/rest/api/postgresql/firewall-rules/list-by-server?view=rest-postgresql-2025-08-01). The first slice targets `2025-08-01` and registers `2024-08-01` as a compatibility version.

- [x] **Step 2: Write failing Flexible Server/database/firewall lifecycle test**

Create a Flexible Server with SKU, storage, backup, version, network, admin, and tags; assert ARM identity, deterministic fully qualified domain name, `Ready` state, and `Succeeded` provisioning state; list servers; create/list/delete a database; create/list/delete a firewall rule.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/postgresql -run TestFlexibleServerDatabaseAndFirewallRuleLifecycle -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement PostgreSQL lifecycle**

Add Flexible Server, database, and firewall rule shapes; implement route parsing for `Microsoft.DBforPostgreSQL/flexibleServers`, `databases`, and `firewallRules`; preserve request properties, SKU, identity, tags, and location; project deterministic fully qualified domain name, state, and provisioning state; cascade child deletion when a server is deleted.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/postgresql -run TestFlexibleServerDatabaseAndFirewallRuleLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Add `TestFlexibleServerDatabaseAndFirewallRuleTemplateProvisioning` for `Microsoft.DBforPostgreSQL/flexibleServers`, `Microsoft.DBforPostgreSQL/flexibleServers/databases`, and `Microsoft.DBforPostgreSQL/flexibleServers/firewallRules`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/postgresql -run TestFlexibleServerDatabaseAndFirewallRuleTemplateProvisioning -count=1
```

Expected red result:

```text
svc.ProvisionTemplateResource undefined
```

- [x] **Step 8: Add template support**

Extend PostgreSQL service with `SupportsTemplateResource` and `ProvisionTemplateResource` for Flexible Server, database, and firewall-rule resources.

- [x] **Step 9: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.DBforPostgreSQL` exposes `flexibleServers`, `flexibleServers/databases`, and `flexibleServers/firewallRules` for API versions `2025-08-01` and `2024-08-01`.

- [x] **Step 10: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.DBforPostgreSQL" could not be found.
```

- [x] **Step 11: Add provider manifest and gateway support**

Add provider manifest entries; normalize `Microsoft.DBforPostgreSQL`; instantiate and register PostgreSQL in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 12: Verify PostgreSQL wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/postgresql ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 33: Azure Cache For Redis Lifecycle And Access Keys

**Files:**
- Add: `services/azure/redis/service.go`
- Add: `services/azure/redis/types.go`
- Add: `services/azure/redis/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Redis contracts**

Use [Redis - Create](https://learn.microsoft.com/en-us/rest/api/redis/redis/create?view=rest-redis-2024-11-01), [Redis - Get](https://learn.microsoft.com/en-us/rest/api/redis/redis/get?view=rest-redis-2024-11-01), [Redis - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/redis/redis/list-by-resource-group?view=rest-redis-2024-11-01), [Redis - List Keys](https://learn.microsoft.com/en-us/rest/api/redis/redis/list-keys?view=rest-redis-2024-11-01), and [Redis - Regenerate Key](https://learn.microsoft.com/en-us/rest/api/redis/redis/regenerate-key?view=rest-redis-2024-11-01). The first slice targets `2024-11-01` and registers `2023-08-01` as a compatibility version.

- [x] **Step 2: Write failing lifecycle/key test**

Create a Redis cache with SKU, tags, and TLS/configuration properties; assert ARM identity, deterministic host name, SSL port, and `Succeeded` provisioning state; list caches; list deterministic keys; regenerate the primary key; delete the cache.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/redis -run TestRedisLifecycleKeysAndRegenerateKey -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement Redis lifecycle and key operations**

Add cache and access-key shapes; implement route parsing for `Microsoft.Cache/Redis`, `listKeys`, and `regenerateKey`; preserve request properties, SKU, identity, tags, and location; project deterministic host/port/provisioning fields; store deterministic per-cache key state and rotate requested key material.

- [x] **Step 5: Verify lifecycle/key green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/redis -run TestRedisLifecycleKeysAndRegenerateKey -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Add `TestRedisTemplateProvisioning` for `Microsoft.Cache/Redis`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/redis -run TestRedisTemplateProvisioning -count=1
```

Expected red result:

```text
svc.ProvisionTemplateResource undefined
```

- [x] **Step 8: Add template support**

Extend Redis service with `SupportsTemplateResource` and `ProvisionTemplateResource` for cache resources.

- [x] **Step 9: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Cache` exposes `Redis` for API versions `2024-11-01` and `2023-08-01`.

- [x] **Step 10: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.Cache" could not be found.
```

- [x] **Step 11: Add provider manifest and gateway support**

Add provider manifest entries; normalize `Microsoft.Cache`; instantiate and register Redis in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 12: Verify Redis wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/redis ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 34: Azure Managed Identity User-Assigned Identity Management

**Files:**
- Add: `services/azure/managedidentity/service.go`
- Add: `services/azure/managedidentity/types.go`
- Add: `services/azure/managedidentity/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `cmd/gateway/main.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Managed Identity contracts**

Use [User Assigned Identities - Create Or Update](https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/create-or-update?view=rest-managedidentity-2023-01-31), [User Assigned Identities - Get](https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2023-01-31), [User Assigned Identities - List By Resource Group](https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/list-by-resource-group?view=rest-managedidentity-2023-01-31), and [User Assigned Identities - Delete](https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/delete?view=rest-managedidentity-2023-01-31). The first slice targets `2023-01-31` and registers `2018-11-30` as a compatibility version.

- [x] **Step 2: Write failing identity lifecycle/template test**

Create a user-assigned identity; assert ARM identity, location, tags, deterministic `clientId`, `principalId`, and `tenantId`; list identities; provision a second identity through ARM template support; delete the first identity.

- [x] **Step 3: Verify compile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/managedidentity -run TestUserAssignedIdentityLifecycleAndTemplateProvisioning -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Implement identity lifecycle and template support**

Add user-assigned identity shape; implement route parsing for `Microsoft.ManagedIdentity/userAssignedIdentities`; preserve request location and tags; project deterministic tenant/client/principal IDs; add `SupportsTemplateResource` and `ProvisionTemplateResource`.

- [x] **Step 5: Verify identity package green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/managedidentity -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.ManagedIdentity` exposes `userAssignedIdentities` for API versions `2023-01-31` and `2018-11-30`.

- [x] **Step 7: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
The resource provider "Microsoft.ManagedIdentity" could not be found.
```

- [x] **Step 8: Add provider manifest and gateway support**

Add provider manifest entries; normalize `Microsoft.ManagedIdentity`; instantiate and register Managed Identity in `cmd/gateway`; register it as a Microsoft.Resources template provisioner.

- [x] **Step 9: Verify Managed Identity wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/managedidentity ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 35: Azure Network Public IP Address And Network Interface Management

**Files:**
- Modify: `services/azure/network/service.go`
- Modify: `services/azure/network/types.go`
- Modify: `services/azure/network/service_test.go`
- Modify: `services/azure/resources/types.go`
- Modify: `services/azure/resources/service_test.go`
- Modify: `docs/azure-implementation-checklists.md`

- [x] **Step 1: Review Microsoft Learn Network contracts**

Use [Public IP Addresses - Create Or Update](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/create-or-update?view=rest-virtualnetwork-2025-05-01), [Public IP Addresses - Get](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/get?view=rest-virtualnetwork-2025-05-01), [Public IP Addresses - List](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/list?view=rest-virtualnetwork-2025-05-01), [Network Interfaces - Create Or Update](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/create-or-update?view=rest-virtualnetwork-2025-05-01), [Network Interfaces - Get](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/get?view=rest-virtualnetwork-2025-05-01), and [Network Interfaces - List](https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/list?view=rest-virtualnetwork-2025-05-01). The slice uses the existing Network versions `2025-05-01` and `2023-09-01`.

- [x] **Step 2: Write failing lifecycle test**

Add `TestPublicIPAddressAndNetworkInterfaceLifecycle` to create prerequisite VNet, subnet, and NSG resources; create/list/delete a public IP; create/list/delete a NIC referencing subnet, NSG, and public IP; assert deterministic public IP address/FQDN, NIC MAC address, IP configuration IDs, and `Succeeded` provisioning state.

- [x] **Step 3: Verify lifecycle red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestPublicIPAddressAndNetworkInterfaceLifecycle -count=1
```

Expected red result:

```text
The Network route is not implemented.
```

- [x] **Step 4: Implement lifecycle**

Add Public IP and NIC resource shapes; extend Network service state, dispatch, CRUD/list/delete handlers, deterministic projected fields, and top-level IDs.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestPublicIPAddressAndNetworkInterfaceLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Extend `TestNetworkChildTemplateProvisioning` to provision `Microsoft.Network/publicIPAddresses` and `Microsoft.Network/networkInterfaces`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestNetworkChildTemplateProvisioning -count=1
```

Expected red result:

```text
unsupported Network template resource type "Microsoft.Network/publicIPAddresses"
```

- [x] **Step 8: Add template support**

Extend Network `SupportsTemplateResource` and `ProvisionTemplateResource` for public IP and NIC resources, preserving SKU and zones.

- [x] **Step 9: Write failing ServiceKeys test**

Add `TestServiceKeysIncludeTopLevelNetworkResourceTypes` to assert versioned service keys for `virtualNetworks`, `networkSecurityGroups`, `publicIPAddresses`, and `networkInterfaces` across `2025-05-01` and `2023-09-01`.

- [x] **Step 10: Verify ServiceKeys red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestServiceKeysIncludeTopLevelNetworkResourceTypes -count=1
```

Expected red result:

```text
expected service key azure|Microsoft.Network/publicIPAddresses|2025-05-01
```

- [x] **Step 11: Register versioned service keys**

Extend `NetworkService.ServiceKeys` with `Microsoft.Network/publicIPAddresses` and `Microsoft.Network/networkInterfaces`.

- [x] **Step 12: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `publicIPAddresses` and `networkInterfaces` in the `Microsoft.Network` provider manifest for `2025-05-01` and `2023-09-01`.

- [x] **Step 13: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected publicIPAddresses in network provider manifest
```

- [x] **Step 14: Add provider manifest support**

Add `publicIPAddresses` and `networkInterfaces` provider resource types to `Microsoft.Network`.

- [x] **Step 15: Verify Network wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 36: Azure Network Load Balancer Management

**Docs verified:** Microsoft Learn Load Balancer REST docs:
- `https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/create-or-update?view=rest-load-balancer-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/get?view=rest-load-balancer-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/list?view=rest-load-balancer-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/delete?view=rest-load-balancer-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/get?view=rest-load-balancer-2023-09-01`

- [x] **Step 1: Verify Microsoft Learn docs**

Confirm the current `2025-05-01` Load Balancer create/get/list/delete docs return 200, and the older `2023-09-01` view resolves after redirect.

- [x] **Step 2: Write failing lifecycle test**

Add `TestLoadBalancerLifecycle` covering:

- Public IP dependency setup.
- `PUT Microsoft.Network/loadBalancers`.
- Stored identity, location, SKU, tags, and `Succeeded` provisioning state.
- Nested child ID projection for frontend IP configurations, backend address pools, probes, load balancing rules, inbound NAT rules, and outbound rules.
- `GET`, resource-group `LIST`, and `DELETE`.

- [x] **Step 3: Verify lifecycle red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestLoadBalancerLifecycle -count=1
```

Expected red result:

```text
The Network route is not implemented.
```

- [x] **Step 4: Implement lifecycle**

Add the `LoadBalancer` resource shape; extend Network service state, actions, dispatch, CRUD/list/delete handlers, nested child ID projection, and top-level resource IDs.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestLoadBalancerLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Extend `TestNetworkChildTemplateProvisioning` to provision `Microsoft.Network/loadBalancers`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestNetworkChildTemplateProvisioning -count=1
```

Expected red result:

```text
unsupported Network template resource type "Microsoft.Network/loadBalancers"
```

- [x] **Step 8: Add template support**

Extend Network `SupportsTemplateResource` and `ProvisionTemplateResource` for Load Balancers, preserving SKU, tags, and properties.

- [x] **Step 9: Write failing ServiceKeys test**

Extend `TestServiceKeysIncludeTopLevelNetworkResourceTypes` to assert versioned service keys for `Microsoft.Network/loadBalancers` across `2025-05-01` and `2023-09-01`.

- [x] **Step 10: Verify ServiceKeys red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestServiceKeysIncludeTopLevelNetworkResourceTypes -count=1
```

Expected red result:

```text
expected service key azure|Microsoft.Network/loadBalancers|2025-05-01
```

- [x] **Step 11: Register versioned service keys**

Extend `NetworkService.ServiceKeys` with `Microsoft.Network/loadBalancers`.

- [x] **Step 12: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `loadBalancers` in the `Microsoft.Network` provider manifest for `2025-05-01` and `2023-09-01`.

- [x] **Step 13: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected loadBalancers in network provider manifest
```

- [x] **Step 14: Add provider manifest support**

Add `loadBalancers` provider resource type to `Microsoft.Network`.

- [x] **Step 15: Verify Load Balancer wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 37: Azure Network Application Gateway Management

**Docs verified:** Microsoft Learn Application Gateway REST docs:
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/create-or-update?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/get?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/delete?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/get?view=rest-application-gateway-2023-09-01`

- [x] **Step 1: Verify Microsoft Learn docs**

Confirm the current `2025-05-01` Application Gateway create/get/list/delete docs return 200, and the older `2023-09-01` view resolves.

- [x] **Step 2: Write failing lifecycle test**

Add `TestApplicationGatewayLifecycle` covering:

- VNet, subnet, and public IP dependency setup.
- `PUT Microsoft.Network/applicationGateways`.
- Stored identity, location, SKU, tags, and `Succeeded` provisioning state.
- Nested child ID projection for gateway IP configurations, frontend IP configurations, frontend ports, backend address pools, backend HTTP settings, HTTP listeners, request routing rules, probes, redirect configurations, SSL certificates, trusted root certificates, and URL path maps.
- `GET`, resource-group `LIST`, and `DELETE`.

- [x] **Step 3: Verify lifecycle red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestApplicationGatewayLifecycle -count=1
```

Expected red result:

```text
The Network route is not implemented.
```

- [x] **Step 4: Implement lifecycle**

Add the `ApplicationGateway` resource shape; extend Network service state, actions, dispatch, CRUD/list/delete handlers, broad nested child ID projection, and top-level resource IDs.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestApplicationGatewayLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Extend `TestNetworkChildTemplateProvisioning` to provision `Microsoft.Network/applicationGateways`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestNetworkChildTemplateProvisioning -count=1
```

Expected red result:

```text
unsupported Network template resource type "Microsoft.Network/applicationGateways"
```

- [x] **Step 8: Add template support**

Extend Network `SupportsTemplateResource` and `ProvisionTemplateResource` for Application Gateways, preserving SKU, tags, and properties.

- [x] **Step 9: Write failing ServiceKeys test**

Extend `TestServiceKeysIncludeTopLevelNetworkResourceTypes` to assert versioned service keys for `Microsoft.Network/applicationGateways` across `2025-05-01` and `2023-09-01`.

- [x] **Step 10: Verify ServiceKeys red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestServiceKeysIncludeTopLevelNetworkResourceTypes -count=1
```

Expected red result:

```text
expected service key azure|Microsoft.Network/applicationGateways|2025-05-01
```

- [x] **Step 11: Register versioned service keys**

Extend `NetworkService.ServiceKeys` with `Microsoft.Network/applicationGateways`.

- [x] **Step 12: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `applicationGateways` in the `Microsoft.Network` provider manifest for `2025-05-01` and `2023-09-01`.

- [x] **Step 13: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected applicationGateways in network provider manifest
```

- [x] **Step 14: Add provider manifest support**

Add `applicationGateways` provider resource type to `Microsoft.Network`.

- [x] **Step 15: Verify Application Gateway wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 38: Azure Network Private Link Management

**Docs verified:** Microsoft Learn Private Endpoint and Private DNS Zone Group REST docs:
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/create-or-update?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/list?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/delete?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get?view=rest-virtualnetwork-2023-09-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-dns-zone-groups/create-or-update?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-dns-zone-groups/get?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-dns-zone-groups/list?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-dns-zone-groups/delete?view=rest-virtualnetwork-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-dns-zone-groups/get?view=rest-virtualnetwork-2023-09-01`

- [x] **Step 1: Verify Microsoft Learn docs**

Confirm current `2025-05-01` Private Endpoint and Private DNS Zone Group create/get/list/delete docs return 200, and the older `2023-09-01` views resolve.

- [x] **Step 2: Write failing lifecycle test**

Add `TestPrivateEndpointAndPrivateDNSZoneGroupLifecycle` covering:

- VNet and subnet dependency setup.
- `PUT Microsoft.Network/privateEndpoints`.
- Stored identity, location, tags, subnet references, and `Succeeded` provisioning state.
- Nested private link service connection and IP configuration ID projection.
- `PUT Microsoft.Network/privateEndpoints/privateDnsZoneGroups`.
- Private DNS zone config child ID projection.
- Parent private endpoint synchronization with zone groups.
- Resource-group private endpoint `LIST`, parent-scoped zone group `LIST`, and `DELETE` for child and parent.

- [x] **Step 3: Verify lifecycle red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestPrivateEndpointAndPrivateDNSZoneGroupLifecycle -count=1
```

Expected red result:

```text
The Network route is not implemented.
```

- [x] **Step 4: Implement lifecycle**

Add `PrivateEndpoint` and `PrivateDNSZoneGroup` resource shapes; extend Network service state, actions, dispatch, CRUD/list/delete handlers, nested child ID projection, parent synchronization, cascade deletion, and canonical IDs.

- [x] **Step 5: Verify lifecycle green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestPrivateEndpointAndPrivateDNSZoneGroupLifecycle -count=1
```

Expected: command passes.

- [x] **Step 6: Write failing template test**

Extend `TestNetworkChildTemplateProvisioning` to provision `Microsoft.Network/privateEndpoints` and `Microsoft.Network/privateEndpoints/privateDnsZoneGroups`.

- [x] **Step 7: Verify template red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestNetworkChildTemplateProvisioning -count=1
```

Expected red result:

```text
unsupported Network template resource type "Microsoft.Network/privateEndpoints"
```

- [x] **Step 8: Add template support**

Extend Network `SupportsTemplateResource` and `ProvisionTemplateResource` for private endpoints and private DNS zone groups, including `{privateEndpoint}/{privateDnsZoneGroup}` template naming.

- [x] **Step 9: Write failing ServiceKeys test**

Extend `TestServiceKeysIncludeTopLevelNetworkResourceTypes` to assert versioned service keys for `Microsoft.Network/privateEndpoints` across `2025-05-01` and `2023-09-01`.

- [x] **Step 10: Verify ServiceKeys red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network -run TestServiceKeysIncludeTopLevelNetworkResourceTypes -count=1
```

Expected red result:

```text
expected service key azure|Microsoft.Network/privateEndpoints|2025-05-01
```

- [x] **Step 11: Register versioned service keys**

Extend `NetworkService.ServiceKeys` with `Microsoft.Network/privateEndpoints`.

- [x] **Step 12: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `privateEndpoints` and `privateEndpoints/privateDnsZoneGroups` in the `Microsoft.Network` provider manifest for `2025-05-01` and `2023-09-01`.

- [x] **Step 13: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
expected privateEndpoints in network provider manifest
```

- [x] **Step 14: Add provider manifest support**

Add `privateEndpoints` and `privateEndpoints/privateDnsZoneGroups` provider resource types to `Microsoft.Network`.

- [x] **Step 15: Verify Private Link wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/network ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 39: Azure Monitor Action Group And Metric Alert Management

**Docs verified:** Microsoft Learn Monitor REST docs:
- `https://learn.microsoft.com/en-us/rest/api/monitor/action-groups/create-or-update?view=rest-monitor-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/monitor/action-groups/get?view=rest-monitor-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/monitor/action-groups/list-by-resource-group?view=rest-monitor-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/monitor/action-groups/delete?view=rest-monitor-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/monitor/action-groups/get?view=rest-monitor-2019-06-01`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-alerts/create-or-update?view=rest-monitor-2024-03-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-alerts/get?view=rest-monitor-2024-03-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-alerts/list-by-resource-group?view=rest-monitor-2024-03-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-alerts/delete?view=rest-monitor-2024-03-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-alerts/get?view=rest-monitor-2018-03-01`

- [x] **Step 1: Verify Microsoft Learn docs**

Confirm current Action Group `2021-09-01`, compatibility Action Group `2019-06-01`, current Metric Alert `2024-03-01-preview`, and stable Metric Alert `2018-03-01` docs resolve.

- [x] **Step 2: Write failing Monitor service tests**

Add `services/azure/monitor/service_test.go` covering:

- Action Group create/get/list/delete with tags and receiver collection preservation.
- Metric Alert create/get/list/delete with tags, scopes, criteria, actions, severity, evaluation frequency, and window size.
- ARM template provisioning for `Microsoft.Insights/actionGroups` and `Microsoft.Insights/metricAlerts`.
- Versioned service keys for current and compatibility API versions.

- [x] **Step 3: Verify Monitor service red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -count=1
```

Expected red result:

```text
undefined: New
```

- [x] **Step 4: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Insights` exposes `actionGroups` and `metricAlerts` with current and compatibility API versions.

- [x] **Step 5: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result:

```text
ProviderNotFound
```

- [x] **Step 6: Implement Monitor service**

Add `services/azure/monitor` with resource types, service keys, template provisioning, route parsing, action group CRUD, metric alert CRUD, stable list ordering, Azure JSON response envelopes, and deterministic `Succeeded` provisioning states.

- [x] **Step 7: Add provider manifest support**

Add `Microsoft.Insights` provider metadata for action groups and metric alerts.

- [x] **Step 8: Register gateway and template provisioner**

Register `azmonitor.New()` in `cmd/gateway/main.go`, attach it to the ARM deployment provisioner, and register all versioned service keys.

- [x] **Step 9: Verify Monitor wiring green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor ./services/azure/resources ./cmd/gateway
```

Expected: all commands pass.

## Task 40: Azure Monitor Diagnostic Settings And Metric Read APIs

**Docs verified:** Microsoft Learn Monitor REST docs:
- `https://learn.microsoft.com/en-us/rest/api/monitor/diagnostic-settings/create-or-update?view=rest-monitor-2021-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/diagnostic-settings/get?view=rest-monitor-2021-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/diagnostic-settings/list?view=rest-monitor-2021-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/diagnostic-settings/delete?view=rest-monitor-2021-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metrics/list?view=rest-monitor-2023-10-01`
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-definitions/list?view=rest-monitor-2023-10-01`

- [x] **Step 1: Verify Microsoft Learn docs**

Confirm resource-scoped diagnostic settings use `2021-05-01-preview` and metrics/metric definitions use `2023-10-01`.

- [x] **Step 2: Write failing Monitor extension-resource tests**

Add `services/azure/monitor/service_test.go` coverage for:

- Diagnostic setting create/get/list/delete on a nested resource scope.
- ARM template provisioning for `Microsoft.Insights/diagnosticSettings` with explicit `scope`.
- Resource-scoped metric definitions list.
- Resource-scoped metrics list with `timespan`, `interval`, and `metricnames`.
- Versioned service keys for `diagnosticSettings`, `metrics`, and `metricDefinitions`.

- [x] **Step 3: Verify Monitor red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run TestDiagnosticSettingLifecycleAndTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run TestMetricDefinitionsAndMetricsList -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run TestMonitorServiceKeysIncludeVersionedResources -count=1
```

Expected red results: route not implemented and missing service key failures.

- [x] **Step 4: Write failing provider manifest test**

Extend `TestProviderManifestListGetAndRegister` to assert `Microsoft.Insights` exposes `diagnosticSettings`, `metrics`, and `metricDefinitions` with verified API versions.

- [x] **Step 5: Verify provider manifest red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red result: missing `diagnosticSettings` provider metadata.

- [x] **Step 6: Implement Monitor extension-resource support**

Extend Monitor route parsing to find the final `providers/Microsoft.Insights` segment, add diagnostic setting state and CRUD, add deterministic metric definitions and metrics responses, and add service keys/template support.

- [x] **Step 7: Verify Monitor diagnostics and metrics green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run TestDiagnosticSettingLifecycleAndTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run TestMetricDefinitionsAndMetricsList -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run TestMonitorServiceKeysIncludeVersionedResources -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected: all commands pass.

## Task 41: Azure App Configuration Store And Key-Value Data Plane

**Docs verified:** Microsoft Learn App Configuration REST docs:
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/create?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/get?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/list-by-resource-group?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-key-value`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/README.md`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-node/tests/appconfig.test.ts`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/appconfig/test_keyvalues.py`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/appconfig/test_etags.py`
- `/private/tmp/floci-az/src/test/java/io/floci/az/services/appconfig/AppConfigHandlerTest.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/AppConfigHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/AppConfigModels.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/KvFilters.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/SyncTokens.java`

- [x] **Step 1: Clone and inspect floci-az**

Clone `https://github.com/floci-io/floci-az` into `/private/tmp/floci-az`, confirm MIT license from GitHub, and identify App Configuration as a high-value missing CloudMock service.

- [x] **Step 2: Write failing App Configuration tests**

Add `services/azure/appconfiguration/service_test.go` covering:

- Configuration store create/get/list/delete.
- ARM template provisioning for `Microsoft.AppConfiguration/configurationStores`.
- Versioned service keys for control-plane and key-value data-plane APIs.
- Data-plane key-value set/get/list/delete with labels, tags, ETags, stale `If-Match` failure, Sync-Token, and Azure media types.

- [x] **Step 3: Write failing routing and provider manifest tests**

Extend `pkg/routing/target_test.go` for `*.azconfig.io` and floci-style local `/{account}-appconfig` data-plane routing. Extend `TestProviderManifestListGetAndRegister` for `Microsoft.AppConfiguration/configurationStores`.

- [x] **Step 4: Verify App Configuration red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestConfigurationStoreLifecycleAndTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestKeyValueDataPlaneLifecycleFilteringAndETags -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppConfigurationDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected red results: undefined `New`, empty route service, and missing provider metadata.

- [x] **Step 5: Implement App Configuration service**

Add `services/azure/appconfiguration` with control-plane store state, template provisioning, provider service keys, key-value data-plane storage, quoted ETags, conditional `If-Match`, labels, tags, simple wildcard key filtering, Sync-Token headers, and Azure App Configuration media types.

- [x] **Step 6: Register routing, manifest, and gateway wiring**

Detect App Configuration data-plane requests, register `Microsoft.AppConfiguration` provider metadata, add `azappconfiguration.New()` to gateway bootstrap, and attach it as an ARM template provisioner.

- [x] **Step 7: Verify App Configuration green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestConfigurationStoreLifecycleAndTemplateProvisioning -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestKeyValueDataPlaneLifecycleFilteringAndETags -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppConfigurationDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration ./services/azure/resources ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 42: Azure Table Storage Data Plane

**Docs verified:** Microsoft Learn Table Storage REST docs:
- `https://learn.microsoft.com/en-us/rest/api/storageservices/table-service-rest-api`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/create-table`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/query-tables`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-table`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/insert-entity`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/query-entities`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/update-entity2`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/merge-entity`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-entity1`
- `https://learn.microsoft.com/en-us/rest/api/storageservices/performing-entity-group-transactions`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/compatibility-tests/sdk-test-node/tests/table.test.ts`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/test_table.py`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/table/TableServiceHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/table/ODataFilter.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/table/TableModel.java`

- [x] **Step 1: Verify Microsoft Learn and floci table behavior**

Confirm Table Storage data-plane routing, table lifecycle operations, JSON entity payload shape, simple OData query support, and ETag concurrency behavior.

- [x] **Step 2: Write failing Table Storage tests**

Extend `pkg/routing/target_test.go` and `services/azure/storage/service_test.go` to cover:

- `*.table.core.windows.net` and floci-style local `/{account}-table` routing to `Microsoft.Storage/tableServices`.
- Versioned `Microsoft.Storage/tableServices` service key registration for `2023-11-03`.
- Table create/list/delete and duplicate conflict behavior.
- Entity insert/get/query/update/delete with `$filter`, `$select`, OData ETags, and stale `If-Match` failure.

- [x] **Step 3: Verify Table Storage red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureStorageDataPlaneHosts -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestStorageServiceKeysIncludeTableDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneLifecycleFilteringAndETags -count=1
```

Expected red results: table requests route as AWS/S3 or fall through the storage account route, and the table service key is missing.

- [x] **Step 4: Implement Table Storage service**

Extend Azure Storage with `Microsoft.Storage/tableServices` service keys, host and local-path data-plane detection, in-memory table/entity state, table CRUD, entity CRUD, simple OData filtering/projection, ETag matching, and Azure table response headers.

- [x] **Step 5: Verify Table Storage green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureStorageDataPlaneHosts -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestStorageServiceKeysIncludeTableDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneLifecycleFilteringAndETags -count=1
```

Expected: all commands pass.

## Task 43: Azure Cosmos DB SQL API Data Plane

**Docs verified:** Microsoft Learn Cosmos DB SQL API REST docs:
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/create-a-database`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/list-databases`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/create-a-collection`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/list-collections`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/create-a-document`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/read-a-document`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/replace-a-document`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/delete-a-document`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/patch-a-document`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db/query-documents`
- `https://learn.microsoft.com/en-us/azure/cosmos-db/nosql/query/overview`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/compatibility-tests/sdk-test-node/tests/cosmos.test.ts`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/test_cosmos.py`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/cosmos/CosmosHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/cosmos/CosmosQueryEngine.java`

- [x] **Step 1: Verify Microsoft Learn and floci SQL API behavior**

Confirm Cosmos DB SQL API data-plane versioning, database/collection/document REST paths, query request headers, response envelopes, ETags, pagination headers, and floci-style local `/{account}-cosmos` route behavior.

- [x] **Step 2: Write failing Cosmos SQL API tests**

Extend `pkg/routing/target_test.go` and `services/azure/cosmosdb/service_test.go` to cover:

- `*.documents.azure.com` and floci-style local `/{account}-cosmos` routing to `Microsoft.DocumentDB/sqlApi`.
- Versioned SQL API service key registration for `2018-12-31`.
- Data-plane database and collection create/list/delete behavior.
- Document create/read/replace/upsert/list/delete behavior with partition-key and ETag headers.
- Query POST with named parameters, `ORDER BY`, pagination, `COUNT`, and patch updates.

- [x] **Step 3: Verify Cosmos SQL API red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureCosmosDBDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestCosmosServiceKeysIncludeSQLDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestCosmosSQLDataPlaneDatabaseContainerAndDocumentLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestCosmosSQLDataPlaneQueryPaginationAndPatch -count=1
```

Expected red results: requests are not identified as Azure/Cosmos data-plane, service key is missing, and direct SQL API requests return route-not-implemented responses.

- [x] **Step 4: Implement Cosmos SQL API data plane**

Add `Microsoft.DocumentDB/sqlApi` service keys, route detection for `documents.azure.com` and local `-cosmos` paths, in-memory SQL API database/collection/document state, system metadata, ETag headers, partition-key lookup, document CRUD/upsert, simple SQL query evaluation, continuation tokens, and patch operations.

- [x] **Step 5: Verify Cosmos SQL API green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureCosmosDBDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestCosmosServiceKeysIncludeSQLDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestCosmosSQLDataPlaneDatabaseContainerAndDocumentLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb -run TestCosmosSQLDataPlaneQueryPaginationAndPatch -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/cosmosdb ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 44: Azure App Configuration Keys, Labels, Locks, And Revisions

**Docs verified:** Microsoft Learn App Configuration data-plane docs:
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-keys`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-labels`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-locks`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-revisions`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-snapshots`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/appconfig/test_feature_flags.py`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/appconfig/test_labels.py`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/appconfig/test_snapshots.py`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/AppConfigHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/AppConfigModels.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/Paginator.java`

- [x] **Step 1: Verify Microsoft Learn and floci App Configuration behavior**

Confirm collection routes for `/keys`, `/labels`, `/locks/{key}`, and `/revisions`, including distinct key/label listings, lock/unlock semantics, and revision history filtering.

- [x] **Step 2: Write failing App Configuration tests**

Extend `pkg/routing/target_test.go` and `services/azure/appconfiguration/service_test.go` to cover:

- Route actions for `ListKeys`, `ListLabels`, `LockKeyValue`, `UnlockKeyValue`, and `ListRevisions`.
- Distinct key and label listing with filters and Azure App Configuration media types.
- Revision history after multiple updates.
- Locking a key-value, rejecting updates while locked, unlocking, and updating again.

- [x] **Step 3: Verify App Configuration red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppConfigurationDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestKeysLabelsLocksAndRevisionsDataPlane -count=1
```

Expected red results: route actions are empty and `/keys` returns not implemented.

- [x] **Step 4: Implement App Configuration expanded data-plane support**

Add routing actions, keyset/labelset media types, revision state, `/keys`, `/labels`, `/revisions`, and `/locks/{key}` handlers, plus locked-key update rejection.

- [x] **Step 5: Verify App Configuration expanded data plane green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppConfigurationDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestKeysLabelsLocksAndRevisionsDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 45: Azure App Configuration Snapshots

**Docs verified:** Microsoft Learn App Configuration snapshot docs:
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/concept-snapshots`
- `https://learn.microsoft.com/en-us/azure/azure-app-configuration/rest-api-snapshots`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/appconfig/test_snapshots.py`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/AppConfigHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/appconfig/AppConfigModels.java`

- [x] **Step 1: Verify Microsoft Learn and floci snapshot behavior**

Confirm snapshot create/get/list/archive/recover routes, frozen snapshot item capture, composition by key or key/label, name/status filters, snapshot media types, and SDK-facing `list_configuration_settings(snapshot_name=...)` behavior.

- [x] **Step 2: Write failing App Configuration snapshot tests**

Extend `pkg/routing/target_test.go` and `services/azure/appconfiguration/service_test.go` to cover:

- Route actions for `CreateSnapshot` and `ListSnapshots`.
- Snapshot creation with filters, `key_label` composition, retention, tags, ETag, and Azure snapshot media type.
- Frozen snapshot-scoped key-value reads via `/kv?snapshot={name}`.
- Snapshot get/list, archive, and recover.

- [x] **Step 3: Verify App Configuration snapshot red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppConfigurationDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestSnapshotsDataPlaneFreezeArchiveRecoverAndList -count=1
```

Expected red result before implementation: routing action detection is missing or `/snapshots/{name}` returns the App Configuration route-not-implemented problem response.

- [x] **Step 4: Implement App Configuration snapshot support**

Add snapshot route dispatch, snapshot media types, in-memory snapshot state, create/get/list/update handlers, frozen item capture, `key` and `key_label` composition, name/status filtering, snapshot metadata projection, ETag headers, and `/kv?snapshot={name}` reads.

- [x] **Step 5: Verify App Configuration snapshot green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppConfigurationDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration -run TestSnapshotsDataPlaneFreezeArchiveRecoverAndList -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appconfiguration ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 46: Azure Key Vault Secret Versions, Properties, And Backup

**Docs verified:** Microsoft Learn Key Vault secret docs:
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/set-secret/set-secret?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/get-secret/get-secret?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/get-secret-versions/get-secret-versions?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/update-secret/update-secret?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/backup-secret/backup-secret?view=rest-keyvault-secrets-2025-07-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/keyvault/test_versions.py`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/keyvault/test_secrets.py`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/keyvault/KeyVaultHandler.java`

- [x] **Step 1: Verify Microsoft Learn and floci Key Vault secret behavior**

Confirm that each secret set creates a new version, latest reads return the newest version, specific-version reads remain available, version lists omit values, property updates do not change values, disabled secrets reject reads, and backup returns opaque secret data.

- [x] **Step 2: Write failing Key Vault tests**

Extend `pkg/routing/target_test.go` and `services/azure/keyvault/service_test.go` to cover:

- `ListSecretVersions`, `UpdateSecret`, and `BackupSecret` routing actions.
- 32-character lowercase hex version IDs.
- Latest and specific-version secret reads.
- Version metadata listing without secret values.
- Property update preserving the stored value.
- Disabled latest-version reads returning forbidden.
- Backup returning non-empty opaque data.

- [x] **Step 3: Verify Key Vault red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureKeyVaultSecretDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault -run TestSecretVersionsPropertiesDisableAndBackup -count=1
```

Expected red result before implementation: route action detection is missing and secret versions are non-Azure-shaped while only the latest secret value is retained.

- [x] **Step 4: Implement Key Vault secret versions**

Add per-vault/per-secret version maps, keep the existing latest-secret map as the compatibility pointer, generate 32-character lowercase hex versions, implement version listing, specific-version lookups, property patching, disabled-secret rejection, backup responses, and the corresponding service actions.

- [x] **Step 5: Verify Key Vault secret versions green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureKeyVaultSecretDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault -run TestSecretVersionsPropertiesDisableAndBackup -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 47: Azure Key Vault Deleted Secret Lifecycle

**Docs verified:** Microsoft Learn Key Vault deleted-secret docs:
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/delete-secret/delete-secret?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/get-deleted-secret/get-deleted-secret?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/get-deleted-secrets/get-deleted-secrets?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/recover-deleted-secret/recover-deleted-secret?view=rest-keyvault-secrets-2025-07-01`
- `https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/purge-deleted-secret/purge-deleted-secret?view=rest-keyvault-secrets-2025-07-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/keyvault/test_secrets.py`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/keyvault/KeyVaultHandler.java`

- [x] **Step 1: Verify Microsoft Learn and floci deleted-secret behavior**

Confirm deleted-secret routes, delete metadata fields, active-read removal after delete, deleted-secret list/get responses, recover restoring the latest secret, retained version behavior, and purge returning `204 No Content`.

- [x] **Step 2: Write failing deleted-secret tests**

Extend `pkg/routing/target_test.go` and `services/azure/keyvault/service_test.go` to cover:

- `ListDeletedSecrets`, `GetDeletedSecret`, `RecoverDeletedSecret`, and `PurgeDeletedSecret` routing actions.
- Delete moving active versions into deleted-secret state.
- Active latest read returning not found after delete.
- Deleted get/list metadata with `recoveryId`, `deletedDate`, and `scheduledPurgeDate`.
- Recover restoring latest and older versions.
- Purge removing deleted-secret state and returning `204`.

- [x] **Step 3: Verify deleted-secret red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureKeyVaultSecretDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault -run TestSecretSoftDeleteRecoverAndPurge -count=1
```

Expected red result before implementation: deleted-secret route actions are empty and `/deletedsecrets` returns the Key Vault route-not-implemented response.

- [x] **Step 4: Implement deleted-secret lifecycle**

Add deleted-secret state, move latest plus all active versions into deleted state on delete, implement `/deletedsecrets`, `/deletedsecrets/{name}`, `/deletedsecrets/{name}/recover`, and deleted-secret purge, and preserve existing active secret behavior.

- [x] **Step 5: Verify deleted-secret green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureKeyVaultSecretDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault -run TestSecretSoftDeleteRecoverAndPurge -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/keyvault ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 48: Azure Service Bus Queue Runtime REST Slice

**Docs verified:** Microsoft Learn Service Bus runtime REST docs:
- `https://learn.microsoft.com/en-us/rest/api/servicebus/service-bus-runtime-rest`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/send-message-to-queue`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/peek-lock-message-non-destructive-read`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/delete-message`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/receive-and-delete-message-destructive-read`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/servicebus/ServiceBusHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/servicebus/ServiceBusNamespaceManager.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/servicebus/ServiceBusModels.java`

- [x] **Step 1: Verify Microsoft Learn and floci Service Bus behavior**

Confirm runtime host routing for `*.servicebus.windows.net`, queue send path `/messages`, peek-lock and receive-delete path `/messages/head`, completion path `/messages/{messageId}/{lockToken}`, response statuses, `Location`, and `BrokerProperties` headers.

- [x] **Step 2: Write failing Service Bus runtime tests**

Extend `pkg/routing/target_test.go` and `services/azure/servicebus/service_test.go` to cover:

- Versionless `Microsoft.ServiceBus/runtime` provider detection.
- Runtime actions `SendMessage`, `PeekLockMessage`, `ReceiveAndDeleteMessage`, and `CompleteMessage`.
- ARM-created queue initialization of runtime queue state.
- Send returning `201`.
- Peek-lock returning body, `Location`, and `BrokerProperties`.
- Complete deleting a locked message.
- Receive-delete returning body and `204` when empty.

- [x] **Step 3: Verify Service Bus runtime red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureServiceBusRuntimeDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestQueueRuntimeSendPeekLockCompleteAndReceiveDelete -count=1
```

Expected red result before implementation: `servicebus.windows.net` requests are not identified as Azure runtime traffic and direct runtime requests return the Service Bus route-not-implemented response.

- [x] **Step 4: Implement Service Bus queue runtime**

Add Service Bus runtime host detection, versionless service key registration, queue runtime state attached to ARM-created queues, send, peek-lock, complete, receive-delete, broker property headers, and runtime queue cleanup during queue/namespace deletion.

- [x] **Step 5: Verify Service Bus runtime green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureServiceBusRuntimeDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestQueueRuntimeSendPeekLockCompleteAndReceiveDelete -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 49: Azure Service Bus Topic Runtime Fan-Out

**Docs verified:** Microsoft Learn Service Bus runtime REST docs:
- `https://learn.microsoft.com/en-us/rest/api/servicebus/send-message-to-queue`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/peek-lock-message-non-destructive-read`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/delete-message`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/servicebus/ServiceBusHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/servicebus/ServiceBusNamespaceManager.java`

- [x] **Step 1: Verify topic and subscription runtime behavior**

Confirm that Service Bus runtime send accepts queue or topic paths, subscription receive uses `/topics/{topic}/subscriptions/{subscription}/messages/head`, and locked subscription completion uses the `Location` path returned from peek-lock.

- [x] **Step 2: Write failing topic runtime test**

Extend `services/azure/servicebus/service_test.go` to cover:

- ARM-created topic and subscription initialize runtime subscription state.
- Sending to `/topic-a/messages` returns `201`.
- Topic send fans out the payload to `/topic-a/subscriptions/sub-a/messages/head`.
- Subscription peek-lock returns body and a subscription-scoped `Location`.
- Deleting the `Location` completes the subscription message.

- [x] **Step 3: Verify topic runtime red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run TestTopicRuntimeFansOutToSubscription -count=1
```

Expected red result before implementation: topic runtime send is treated as a queue send and returns `410`.

- [x] **Step 4: Implement topic runtime fan-out**

Add runtime topic and subscription maps, initialize them during ARM topic/subscription lifecycle, fan out topic sends into subscription runtime queues, route subscription peek-lock/complete paths through a shared runtime target parser, and clean up runtime state on topic/subscription deletion.

- [x] **Step 5: Verify topic runtime green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus -run 'Test(QueueRuntimeSendPeekLockCompleteAndReceiveDelete|TopicRuntimeFansOutToSubscription)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/servicebus ./pkg/routing ./cmd/gateway -count=1
```

Expected: all commands pass.

## Task 50: Azure Event Grid Custom Topic Publish Data Plane

**Docs verified:** Microsoft Learn Event Grid custom topic publish docs:
- `https://learn.microsoft.com/en-us/azure/event-grid/post-to-custom-topic`

**floci-az reference files inspected:**
- No Event Grid publish implementation was present under `/private/tmp/floci-az/src/main/java/io/floci/az/services`; this slice follows Microsoft Learn directly.

- [x] **Step 1: Verify Microsoft Learn publish contract**

Confirm that custom topic publishing uses `POST https://<topic-endpoint>?api-version=2018-01-01`, with topic endpoints like `https://{topic}.{region}-1.eventgrid.azure.net/api/events`, `aeg-sas-key` publisher clients, Event Grid schema event arrays, `200 OK` success, `400 Bad Request` invalid event data, and `404 Not Found` incorrect endpoints.

- [x] **Step 2: Write failing Event Grid publish tests**

Extend `pkg/routing/target_test.go` and `services/azure/eventgrid/service_test.go` to cover:

- Data-plane host routing for `*.eventgrid.azure.net/api/events`.
- Versioned `Microsoft.EventGrid/publish|2018-01-01` service registration.
- Publishing a valid Event Grid event array to an ARM-created custom topic endpoint.
- Retaining published events per topic for local assertions.
- Rejecting malformed non-array payloads with `400`.

- [x] **Step 3: Verify Event Grid publish red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventGridPublishDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run 'Test(PublishEventsToCustomTopicDataPlane|PublishEventsRejectsInvalidCustomTopicPayload|ServiceKeysIncludePublishDataPlane)' -count=1
```

Expected red results before implementation: routing does not identify Event Grid publish as Azure data-plane traffic, and the Event Grid service has no publish-state field or data-plane service key.

- [x] **Step 4: Implement Event Grid publish data plane**

Add Event Grid data-plane host detection, `PublishEvents` action mapping, versioned publish service key registration, publish request dispatch before ARM route parsing, endpoint matching against created topics, Event Grid schema validation, documented status responses, and retained per-topic published events.

- [x] **Step 5: Verify Event Grid publish green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventGridPublishDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run 'Test(PublishEventsToCustomTopicDataPlane|PublishEventsRejectsInvalidCustomTopicPayload|ServiceKeysIncludePublishDataPlane|TopicAndEventSubscriptionLifecycle|TopicEventSubscriptionTemplateProvisioning)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 51: Azure Event Grid Topic Keys And Publish Authorization

**Docs verified:** Microsoft Learn Event Grid topic key docs:
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/list-shared-access-keys?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/regenerate-key?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/azure/event-grid/post-to-custom-topic#header`

**floci-az reference files inspected:**
- No Event Grid key implementation was present under `/private/tmp/floci-az/src/main/java/io/floci/az/services`; this slice follows Microsoft Learn directly.

- [x] **Step 1: Verify Microsoft Learn key contracts**

Confirm that topic keys are listed with `POST .../topics/{topicName}/listKeys?api-version=2025-02-15`, regenerated with `POST .../topics/{topicName}/regenerateKey?api-version=2025-02-15` and body `{"keyName":"key1"}` or `{"keyName":"key2"}`, both return `key1` and `key2`, and custom topic publishing uses `aeg-sas-key`.

- [x] **Step 2: Write failing key-management tests**

Extend routing and Event Grid tests to cover:

- `ListSharedAccessKeys` and `RegenerateKey` action detection.
- `listKeys` returns two distinct keys for an ARM-created topic.
- `regenerateKey` rotates only the requested key.
- Wrong or stale keys are rejected by custom topic publish with `401`.
- New rotated keys authorize publish.
- Invalid `keyName` is rejected with `400`.

- [x] **Step 3: Verify key-management red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventGridTopicKeyActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run 'Test(PublishEventsToCustomTopicDataPlane|TopicSharedAccessKeysRegenerateAndAuthorizePublish|PublishEventsRejectsInvalidCustomTopicPayload|RegenerateTopicSharedAccessKeyRejectsInvalidKeyName)' -count=1
```

Expected red results before implementation: routing returns raw `listKeys`/`regenerateKey` action names and Event Grid returns the route-not-implemented response for topic key operations.

- [x] **Step 4: Implement topic keys and publish authorization**

Add topic key state initialized during topic creation, `listKeys`, `regenerateKey`, independent key rotation, topic-key cleanup on topic delete, key-management action registration, route dispatch for topic operations, and `aeg-sas-key` validation during custom topic publish.

- [x] **Step 5: Verify key-management green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureEventGrid(TopicKeyActions|PublishDataPlane)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid -run 'Test(PublishEventsToCustomTopicDataPlane|TopicSharedAccessKeysRegenerateAndAuthorizePublish|PublishEventsRejectsInvalidCustomTopicPayload|RegenerateTopicSharedAccessKeyRejectsInvalidKeyName|ServiceKeysIncludePublishDataPlane)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventgrid ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 52: Azure Event Hubs Consumer Group Lifecycle

**Docs verified:** Microsoft Learn Event Hubs consumer group docs:
- `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/create-or-update?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/get?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/list-by-event-hub?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/delete?view=rest-eventhub-2026-01-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/eventhub/EventHubHandler.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/EventHubNamespaceManagementTest.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-node/tests/eventhub.test.ts`
- `/private/tmp/floci-az/docs/services/event-hub.md`

- [x] **Step 1: Verify Microsoft Learn and floci consumer-group behavior**

Confirm that Event Hubs consumer groups are nested below `namespaces/{namespace}/eventhubs/{eventHub}/consumergroups`, that create/update returns a `ConsumerGroup` resource with `userMetadata`, `createdAt`, and `updatedAt`, list returns a `value` envelope, delete returns `200` or `204`, and floci provisions `$Default` consumer groups for event hubs in its sidecar topology.

- [x] **Step 2: Write failing consumer group tests**

Extend routing and Event Hubs tests to cover:

- Nested `consumergroups` action detection for create/get/list/delete.
- Automatic `$Default` consumer group creation when an event hub is created.
- Consumer group create/get/list/update/delete lifecycle.
- Stable `createdAt` across update and refreshed `updatedAt`.
- Delete idempotency returning `204` for a missing group.

- [x] **Step 3: Verify consumer group red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventHubConsumerGroupActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestConsumerGroupLifecycle -count=1
```

Expected red results before implementation: routing returns generic ARM actions and Event Hubs returns route-not-implemented for `consumergroups`.

- [x] **Step 4: Implement consumer group lifecycle**

Add `ConsumerGroup` resource state, nested route parsing for `eventhubs/{hub}/consumergroups/{group}`, action registration, default `$Default` group creation on event hub create, create/get/list/delete handlers, timestamp fields, update behavior, and namespace/event hub cleanup for child consumer groups.

- [x] **Step 5: Verify consumer group green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventHubConsumerGroupActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestConsumerGroupLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 53: Azure Event Hubs Namespace Authorization Rules And Keys

**Docs verified:** Microsoft Learn Event Hubs namespace authorization-rule docs:
- `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/create-or-update-authorization-rule?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-authorization-rules?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-keys?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/regenerate-keys?view=rest-eventhub-2026-01-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/event-hub.md`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/eventhub/EventHubHandler.java`

- [x] **Step 1: Verify Microsoft Learn and floci key behavior**

Confirm namespace authorization rules live under `namespaces/{namespace}/authorizationRules`, list returns a `value` envelope, `listKeys` returns primary/secondary keys and connection strings, `regenerateKeys` accepts `PrimaryKey` or `SecondaryKey`, and floci's Event Hubs connection strings use `RootManageSharedAccessKey`.

- [x] **Step 2: Write failing authorization-rule tests**

Extend routing and Event Hubs tests to cover:

- Namespace authorization-rule action detection for create/get/list/delete/listKeys/regenerateKeys.
- Default `RootManageSharedAccessKey` rule creation with `Listen`, `Manage`, and `Send` rights.
- Custom authorization rule create/get/list/delete lifecycle.
- Key listing with primary/secondary keys and connection strings.
- Primary-key rotation preserving secondary key.
- Explicit secondary-key replacement.
- Delete idempotency returning `204` for a missing rule.

- [x] **Step 3: Verify authorization-rule red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventHubNamespaceAuthorizationRuleActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestNamespaceAuthorizationRulesKeysAndRegeneration -count=1
```

Expected red results before implementation: routing returns generic ARM action names and Event Hubs returns route-not-implemented for `authorizationRules`.

- [x] **Step 4: Implement namespace authorization rules and keys**

Add `AuthorizationRule` and `AccessKeys` state, seed `RootManageSharedAccessKey` during namespace creation, implement namespace rule create/get/list/delete, deterministic primary/secondary key generation, namespace connection string projection, independent key regeneration, explicit key replacement, cleanup on namespace delete, and action registration.

- [x] **Step 5: Verify authorization-rule green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventHubNamespaceAuthorizationRuleActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestNamespaceAuthorizationRulesKeysAndRegeneration -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 54: Azure Event Hubs Event-Hub Authorization Rules And Keys

**Docs verified:** Microsoft Learn Event Hubs event-hub authorization-rule docs:
- `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/create-or-update-authorization-rule?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/list-keys?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/regenerate-keys?view=rest-eventhub-2026-01-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/event-hub.md`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-node/tests/eventhub.test.ts`

- [x] **Step 1: Verify Microsoft Learn event-hub auth contracts**

Confirm event-hub-scoped authorization rules live under `namespaces/{namespace}/eventhubs/{eventHub}/authorizationRules`, create/update returns `Microsoft.EventHub/Namespaces/EventHubs/AuthorizationRules`, listKeys returns connection strings with `EntityPath`, and regenerateKeys rotates primary or secondary keys.

- [x] **Step 2: Write failing event-hub auth tests**

Extend routing and Event Hubs tests to cover:

- Event-hub authorization-rule action detection for create/get/list/delete/listKeys/regenerateKeys.
- Event-hub authorization rule create/list/delete lifecycle.
- Event-hub scoped key listing with `EntityPath`.
- Event-hub primary key regeneration.

- [x] **Step 3: Verify event-hub auth red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureEventHubAuthorizationRuleActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run TestEventHubAuthorizationRulesKeysAndRegeneration -count=1
```

Expected red results before implementation: event-hub auth routes are detected as namespace auth actions and Event Hubs returns route-not-implemented for event-hub-scoped authorization rules.

- [x] **Step 4: Implement event-hub authorization rules and keys**

Add event-hub authorization-rule route parsing, scoped action names, create/get/list/delete handlers, event-hub scoped access key generation, connection string projection with `EntityPath`, key regeneration, and event hub cleanup of child auth rules.

- [x] **Step 5: Verify event-hub auth green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureEventHub(AuthorizationRuleActions|NamespaceAuthorizationRuleActions)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub -run 'Test(EventHubAuthorizationRulesKeysAndRegeneration|NamespaceAuthorizationRulesKeysAndRegeneration)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/eventhub ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 55: Azure Container Registry Name Availability And Replications

**Docs verified:** Microsoft Learn ACR docs:
- `https://learn.microsoft.com/en-us/rest/api/container-registry/registries/check-name-availability?view=rest-container-registry-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/container-registry/replications/list?view=rest-container-registry-2025-11-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/test/java/io/floci/az/services/acr/AcrHandlerTest.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-python/tests/test_acr.py`
- `/private/tmp/floci-az/docs/services/acr.md`

- [x] **Step 1: Verify Microsoft Learn and floci ACR contracts**

Confirm `checkNameAvailability` is subscription-scoped under `Microsoft.ContainerRegistry`, returns `nameAvailable`, `reason`, and `message`, and that `replications` is a registry child collection returned as a `value` envelope. Confirm floci exercises name availability against existing registries and exposes empty replication lists for compatibility.

- [x] **Step 2: Write failing ACR routing and service tests**

Extend routing and ACR tests to cover:

- Subscription-scoped `checkNameAvailability` action detection.
- Registry child `replications` list action detection.
- Existing registry names returning `AlreadyExists`.
- Free registry names returning available.
- Invalid registry names returning `Invalid`.
- Existing registries returning an empty replication list.

- [x] **Step 3: Verify ACR red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestCheckNameAvailabilityAndReplications -count=1
```

Expected red results before implementation: routing returns generic/raw action names, and the ACR service returns route-not-implemented for `checkNameAvailability`.

- [x] **Step 4: Implement ACR name availability and replications**

Add ACR-specific routing for subscription-scoped actions and registry child collections, support `POST /subscriptions/{subscriptionId}/providers/Microsoft.ContainerRegistry/checkNameAvailability`, validate Azure registry naming rules, detect existing registry names per subscription, and add `GET .../registries/{name}/replications` returning an empty `value` envelope for existing registries.

- [x] **Step 5: Verify ACR green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestCheckNameAvailabilityAndReplications -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 56: Azure Container Registry Registry Update

**Docs verified:** Microsoft Learn ACR registry update docs:
- `https://learn.microsoft.com/en-us/rest/api/container-registry/registries/update?view=rest-container-registry-2025-11-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/acr/AcrHandler.java`
- `/private/tmp/floci-az/docs/services/acr.md`

- [x] **Step 1: Verify Microsoft Learn and floci update behavior**

Confirm `PATCH .../registries/{registryName}` accepts tags, SKU, identity, and mutable properties such as `adminUserEnabled`, `publicNetworkAccess`, and `roleAssignmentMode`, returning a registry resource. Confirm floci updates tags, SKU name, and `adminUserEnabled` in place.

- [x] **Step 2: Write failing ACR update tests**

Extend routing and ACR tests to cover:

- ACR `PATCH` action detection as `UpdateRegistry`.
- Tag replacement.
- SKU name update with tier projection.
- Mutable property merge.
- Preservation of `loginServer` and `provisioningState`.
- Persistence visible through subsequent `GET`.

- [x] **Step 3: Verify ACR update red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestRegistryPatchUpdatesMutableFields -count=1
```

Expected red results before implementation: routing returns generic `Update`, and the ACR service rejects `PATCH` with `405 MethodNotAllowed`.

- [x] **Step 4: Implement registry update**

Add the `UpdateRegistry` action, route `PATCH` registry resources to an update handler, replace tags when provided, normalize SKU tier from SKU name, merge provided mutable properties into the stored registry, preserve required defaults, support identity updates, and return the updated registry with `200 OK`.

- [x] **Step 5: Verify ACR update green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestRegistryPatchUpdatesMutableFields -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 57: Azure Container Registry Subscription List And Usages

**Docs verified:** Microsoft Learn ACR registry list and usage docs:
- `https://learn.microsoft.com/en-us/rest/api/container-registry/registries/list?view=rest-container-registry-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/container-registry/registries/list-usages?view=rest-container-registry-2025-11-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/acr/AcrHandler.java`
- `/private/tmp/floci-az/docs/services/acr.md`

- [x] **Step 1: Verify Microsoft Learn and floci list/read contracts**

Confirm subscription-scoped registry list lives at `/subscriptions/{subscriptionId}/providers/Microsoft.ContainerRegistry/registries`, returns a `value` envelope, and that `listUsages` returns quota usage entries for an existing registry. Confirm floci supports subscription list and a static `Size` quota report.

- [x] **Step 2: Write failing ACR list and usage tests**

Extend routing and ACR tests to cover:

- Subscription-scoped list action detection as `ListRegistries`.
- Registry `listUsages` action detection.
- Subscription list filtering across resource groups while excluding other subscriptions.
- Stable name ordering in subscription list results.
- Static registry usage response with `Size`, positive byte limit, and zero current usage.

- [x] **Step 3: Verify list and usage red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestListSubscriptionRegistriesAndUsages -count=1
```

Expected red results before implementation: routing returns generic `List`/`Get`, and the ACR service returns `404 NotFound` for subscription-scoped registry listing.

- [x] **Step 4: Implement subscription list and usages**

Add ACR-specific routing actions for subscription list and `listUsages`, support subscription-scoped parse routes, extend registry listing to handle optional resource groups, and return deterministic static usage quota metadata for existing registries.

- [x] **Step 5: Verify list and usage green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestListSubscriptionRegistriesAndUsages -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 58: Azure Container Registry Import Image No-Op

**Docs verified:** Microsoft Learn ACR import image docs:
- `https://learn.microsoft.com/en-us/rest/api/container-registry/registries/import-image?view=rest-container-registry-2025-11-01`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/acr/AcrHandler.java`
- `/private/tmp/floci-az/docs/services/acr.md`

- [x] **Step 1: Verify Microsoft Learn and floci import behavior**

Confirm `POST .../registries/{registryName}/importImage` accepts source, target tags, untagged target repositories, and import mode, and may return `200 OK` or `202 Accepted`. Confirm floci accepts the operation as a no-op.

- [x] **Step 2: Write failing ACR import tests**

Extend routing and ACR tests to cover:

- `importImage` action detection as `ImportImage`.
- Existing registry import request returning `202 Accepted`.
- Empty response body for the no-op operation.

- [x] **Step 3: Verify import red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestImportImageAcceptedNoop -count=1
```

Expected red results before implementation: routing returns lowercase generic `importImage`, and the ACR service returns route-not-implemented.

- [x] **Step 4: Implement import no-op**

Add the `ImportImage` action, route `POST importImage` requests, verify the destination registry exists, and return `202 Accepted` with an empty body.

- [x] **Step 5: Verify import green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run TestImportImageAcceptedNoop -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 59: Azure Kubernetes Service Mocked Management Plane

**Docs verified:** Microsoft Learn AKS docs:
- `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/aks/agent-pools/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/aks/agent-pools/get`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/aks.md`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/aks/AksHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/aks/AksModels.java`
- `/private/tmp/floci-az/src/test/java/io/floci/az/services/aks/AksHandlerTest.java`

- [x] **Step 1: Verify Microsoft Learn and floci AKS contracts**

Confirm managed clusters use `Microsoft.ContainerService/managedClusters`, create/update returns `200` or `201`, agent pools are nested under `managedClusters/{cluster}/agentPools`, and floci mocked mode returns immediate `Succeeded` state plus a synthetic base64 kubeconfig.

- [x] **Step 2: Write failing AKS routing and service tests**

Add routing and service tests to cover:

- Managed cluster create/get/list by resource group/list by subscription/patch tags/delete.
- Admin and user credential action detection and kubeconfig response shape.
- Default agent pool projection from cluster creation.
- Agent pool create/get/list/update/delete.
- Versioned service keys for `2026-02-01`, `2026-04-01`, and `2024-04-01`.

- [x] **Step 3: Verify AKS red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -count=1
```

Expected red results before implementation: routing falls back to generic ARM actions, and the new `services/azure/containerservice` package fails to compile because `New` is undefined.

- [x] **Step 4: Implement mocked AKS service**

Add the `containerservice` Azure service package, modeled on floci mocked mode, with managed cluster state, deterministic ARM resource IDs, default agent pools, synthetic base64 kubeconfig, cluster and agent-pool route parsing, template provisioning support, versioned service keys, precise routing actions, and gateway registration.

- [x] **Step 5: Verify AKS green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 60: Azure API Management Service Lifecycle

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/azure/api-management/`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/apim.md`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementService.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify Microsoft Learn and floci APIM service contracts**

Confirm APIM service resources use `Microsoft.ApiManagement/service`, expose service lifecycle operations through ARM, and floci returns `Succeeded` provisioning state plus deterministic gateway and management API URLs.

- [x] **Step 2: Write failing APIM service tests**

Add routing and service tests to cover:

- APIM service create/get/list by resource group/list by subscription/delete action detection.
- Service resource identity, location, SKU, publisher fields, and provisioning state.
- floci-compatible `gatewayUrl` containing `/devstoreaccount1-apim/{service}`.
- Stable subscription-list ordering and filtering.
- Versioned service key for `2024-05-01`.

- [x] **Step 3: Verify APIM service red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementServiceActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
```

Expected red results before implementation: routing falls back to generic ARM actions, and the new `services/azure/apimanagement` package fails to compile because `New` is undefined.

- [x] **Step 4: Implement APIM service lifecycle**

Add the APIM service package with service-resource state, create/get/list/delete handlers, deterministic `gatewayUrl` and `managementApiUrl`, publisher and SKU defaults, ARM template provisioning support, versioned service keys, precise routing actions, and gateway registration.

- [x] **Step 5: Verify APIM service green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementServiceActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 61: Azure API Management APIs And Operations

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-operation/create-or-update`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/apim.md`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementService.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify Microsoft Learn and floci APIM child-resource contracts**

Confirm APIs live under `service/{serviceName}/apis/{apiId}`, operations live under `apis/{apiId}/operations/{operationId}`, list responses use a `value` envelope, and floci defaults API display name/path/protocols plus operation display name/method/url template.

- [x] **Step 2: Write failing APIM API/operation tests**

Add routing and service tests to cover:

- API create/get/list/delete action detection and ARM resource shape.
- Operation create/get/list/delete action detection and ARM resource shape.
- API and operation property preservation/defaults.
- Service/API existence validation.
- Operation cleanup when an API is deleted.

- [x] **Step 3: Verify APIM API/operation red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementAPIAndOperationActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementAPIsAndOperationsLifecycle -count=1
```

Expected red results before implementation: routing falls back to generic ARM actions, and the APIM service returns route-not-implemented for nested API routes.

- [x] **Step 4: Implement API and operation resources**

Add APIM API and operation state maps, nested route parsing, create/get/list/delete handlers, property defaults, cascade deletion from service to APIs and from APIs to operations, and precise routing actions.

- [x] **Step 5: Verify APIM API/operation green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementAPIAndOperationActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementAPIsAndOperationsLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 62: Azure API Management Policy Resources

**Docs verified:** Microsoft Learn API Management policy docs:
- `https://learn.microsoft.com/en-us/azure/api-management/api-management-howto-policies`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/policy/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-policy/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-operation-policy/create-or-update`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/apim.md`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementService.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify floci APIM policy resource scopes**

Confirm floci stores policies under service, API, and operation parent keys, returns `rawxml` format by default, lists policies by parent scope, and removes child policies when parent resources are deleted.

- [x] **Step 2: Write failing APIM policy tests**

Add routing and service tests to cover:

- Policy create/list/delete action detection.
- Service-scoped policy resource shape and storage.
- API-scoped policy resource shape and storage.
- Operation-scoped policy resource shape and storage.
- Policy list isolation by parent scope.

- [x] **Step 3: Verify APIM policy red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementPolicyActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementPolicyLifecycleAtServiceAPIAndOperationScopes -count=1
```

Expected red results before implementation: routing falls back to generic ARM actions, and the APIM service returns route-not-implemented for policy routes.

- [x] **Step 4: Implement APIM policy resources**

Add policy state keyed by parent resource, service/API/operation scoped route parsing, create/get/list/delete handlers, `rawxml`/empty-value defaults, internal parent-key stripping from ARM responses, and parent deletion cleanup.

- [x] **Step 5: Verify APIM policy green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementPolicyActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementPolicyLifecycleAtServiceAPIAndOperationScopes -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement ./pkg/routing ./cmd/gateway ./services/azure/resources -count=1
```

Expected: all commands pass.

## Task 63: Azure API Management Products, Subscriptions, Named Values, And Backends

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/azure/api-management/`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/product/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/product-api/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/subscription/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/named-value/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/backend/create-or-update`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/docs/services/apim.md`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementService.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify floci APIM auxiliary resource contracts**

Confirm floci supports APIM products, product API links, subscriptions, named values, and backends under `service/{serviceName}`, returns `200 OK` for these create/update handlers, defaults product/subscription/named-value/backend properties, hides `properties.value` for secret named values, and returns linked API resources for product API links.

- [x] **Step 2: Write failing APIM auxiliary resource tests**

Add routing and service tests to cover:

- Product create/get/list/delete action detection and ARM resource shape.
- Product API link create/get/list/delete action detection and linked API resource responses.
- Subscription create/get/list/delete action detection, default active state, scope, and key preservation.
- Named value create/get/list/delete action detection and secret value response redaction.
- Backend create/get/list/delete action detection and default title/protocol/url behavior.
- Cascade cleanup from service/API/product deletes.

- [x] **Step 3: Verify APIM auxiliary resource red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementAuxiliaryResourceActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementAuxiliaryResourcesLifecycle -count=1
```

Observed red results before implementation: routing fell back to generic ARM actions for products, subscriptions, named values, and backends; product API links were misdetected as generic API actions; and the APIM service returned `404` with `The API Management child route is not implemented.` for product creation.

- [x] **Step 4: Implement APIM auxiliary resources**

Add product, product API link, subscription, named value, and backend state maps; nested request handlers; create/get/list/delete methods; floci-compatible property defaults; deterministic subscription key generation; secret named value response filtering; service/API/product cascade cleanup; and precise APIM routing actions. Fix APIM subscription action detection so the helper skips the leading ARM `/subscriptions/{subscriptionId}` segment and matches the service child `subscriptions` collection instead.

- [x] **Step 5: Verify APIM auxiliary resource green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementAuxiliaryResourceActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementAuxiliaryResourcesLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 64: Azure API Management Local Gateway Runtime And Subscription Enforcement

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/azure/api-management/`
- `https://learn.microsoft.com/en-us/azure/api-management/api-management-gateways-overview`
- `https://learn.microsoft.com/en-us/azure/api-management/api-management-subscriptions`
- `https://learn.microsoft.com/en-us/azure/api-management/api-management-howto-policies`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementGateway.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementProxy.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify floci APIM runtime contracts**

Confirm local APIM gateway requests use `/devstoreaccount1-apim/{service}/{apiPath...}`, match the most-specific API path, match operations by method and URL template, return mock proxy JSON when no backend URL is configured, and enforce subscription keys only when products are linked to the matched API.

- [x] **Step 2: Write failing APIM runtime tests**

Add routing and service tests to cover:

- Local `/devstoreaccount1-apim/{service}` request detection as Azure `Microsoft.ApiManagement/service`.
- Runtime action detection as `GatewayProxy` with omitted API version so the registry default version is used.
- API path and operation URL-template matching.
- Mock proxy JSON body with service, API ID, operation ID, method, gateway path, backend path, headers, and query params.
- Product API link subscription enforcement for missing, header, query-string, inactive, and unlinked cases.

- [x] **Step 3: Verify APIM runtime red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementLocalGatewayAction -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementLocalGatewayRouteAndSubscriptionKeys -count=1
```

Observed red results before implementation: routing classified the local APIM path as AWS S3 with no action, and the APIM service returned ARM route `404` with `The API Management route is not implemented.`

- [x] **Step 4: Implement APIM runtime gateway slice**

Add local `-apim` provider/service detection, a `GatewayProxy` action, APIM service runtime dispatch before ARM parsing, service/API/operation match helpers, floci-compatible mock JSON response projection, single-value query echoing, and product subscription-key validation against active subscriptions scoped to linked products.

- [x] **Step 5: Verify APIM runtime green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAPIManagementLocalGatewayAction -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementLocalGatewayRouteAndSubscriptionKeys -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 65: Azure API Management Gateway Policy Effects

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/azure/api-management/api-management-howto-policies`
- `https://learn.microsoft.com/en-us/azure/api-management/api-management-policy-expressions`
- `https://learn.microsoft.com/en-us/azure/api-management/set-header-policy`
- `https://learn.microsoft.com/en-us/azure/api-management/rewrite-uri-policy`
- `https://learn.microsoft.com/en-us/azure/api-management/return-response-policy`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementGateway.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementProxy.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify floci APIM policy runtime contracts**

Confirm floci applies service, API, and operation policies in gateway routing; expands named values in policy values; supports `set-header`, `set-query-parameter`, `rewrite-uri`, and `return-response`; and emits return-response status, headers, body, and JSON content type when appropriate.

- [x] **Step 2: Write failing APIM policy-effect tests**

Add APIM service tests to cover:

- API policy `set-header` with regular and secret named value expansion.
- API policy `set-query-parameter` while preserving original query params.
- API policy `rewrite-uri` updating the mock proxy `backendPath`.
- API policy `return-response` overriding status, headers, body, and content type.

- [x] **Step 3: Verify APIM policy-effect red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run 'TestAPIManagementLocalGateway(AppliesPolicyEffects|ReturnResponsePolicy)' -count=1
```

Observed red results before implementation: gateway responses ignored stored policy XML, left headers empty, left `backendPath` unchanged, and returned normal `200` mock proxy JSON instead of the `return-response` status/body.

- [x] **Step 4: Implement APIM policy-effect runtime**

Add a small XML-decoder based policy evaluator for stored APIM policy XML, service/API/operation policy lookup and ordered application, named value expansion, header/query mutation semantics for override/skip/delete/append where applicable, URI rewrite, and return-response projection.

- [x] **Step 5: Verify APIM policy-effect green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run 'TestAPIManagementLocalGateway(AppliesPolicyEffects|ReturnResponsePolicy)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 66: Azure API Management Backend Selection And HTTP Forwarding

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/azure/api-management/set-backend-service-policy`
- `https://learn.microsoft.com/en-us/azure/api-management/forward-request-policy`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/backend/create-or-update`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementGateway.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementProxy.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify APIM backend forwarding contracts**

Confirm Microsoft Learn documents `set-backend-service` as changing the backend base URL via `base-url` or `backend-id`, and `forward-request` as forwarding to the API backend URL after policy changes. Confirm floci proxies when a backend URL is available and otherwise returns a mock proxy JSON body.

- [x] **Step 2: Write failing backend-proxy tests**

Add APIM service tests to cover:

- API `properties.serviceUrl` forwarding to a backend HTTP client with matched operation suffix and original query parameters.
- Backend response status, content type, headers, and body passthrough.
- `set-backend-service backend-id` resolving an APIM backend resource and overriding the API `serviceUrl`.
- Policy-transformed headers/query parameters and `rewrite-uri` output being sent to the backend.

- [x] **Step 3: Verify backend-proxy red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run 'TestAPIManagementLocalGateway(ProxiesAPIServiceURL|SetBackendServicePolicyOverridesAPIServiceURL)' -count=1
```

Observed red results before implementation: the first draft attempted to use `httptest.NewServer`, which the sandbox cannot bind; after replacing it with an in-memory `RoundTripper`, the expected red result was `svc.httpClient undefined`, proving there was no backend proxy injection/path yet.

- [x] **Step 4: Implement backend forwarding**

Add an injectable APIM HTTP client, resolve API `serviceUrl`, implement policy `set-backend-service` with `backend-id` and `base-url`, release the APIM state lock before outbound calls, build backend URLs from suffix/rewrite and query params, forward method/body/content type/policy headers, and pass through backend status, content type, selected headers, and body.

- [x] **Step 5: Verify backend-proxy green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run 'TestAPIManagementLocalGateway(ProxiesAPIServiceURL|SetBackendServicePolicyOverridesAPIServiceURL)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 67: Azure API Management OpenAPI Import Parity

**Docs verified:** Microsoft Learn API Management docs:
- `https://learn.microsoft.com/en-us/azure/api-management/import-api-from-oas`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-operation/list-by-api`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/apim/ApiManagementService.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/ApiManagementCompatibilityTest.java`

- [x] **Step 1: Verify APIM OpenAPI import contracts**

Confirm API create/update payloads with `properties.format` containing `openapi` and JSON `properties.value` create APIM operation resources from OpenAPI `paths`, preserve `operationId`, `summary`, method, and path template, and replace stale imported operations when the same API is re-imported.

- [x] **Step 2: Write failing OpenAPI import test**

Add APIM service coverage for importing an Orders OpenAPI document, listing generated `createOrder` and `getOrder` operations, routing local gateway requests through those generated operation templates, then re-importing a Customers document and confirming stale order operations/routes are removed.

- [x] **Step 3: Verify OpenAPI import red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementOpenAPIImportCreatesGatewayOperations -count=1
```

Observed red result before implementation: the operations collection for the imported API was empty.

- [x] **Step 4: Implement OpenAPI operation import**

Wire API create/update to parse OpenAPI import payloads, decode embedded JSON OpenAPI documents, replace existing operations for the API, and create operation resources for supported HTTP methods in `paths`.

- [x] **Step 5: Verify OpenAPI import green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -run TestAPIManagementOpenAPIImportCreatesGatewayOperations -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/apimanagement -count=1
```

Expected: all commands pass.

## Task 68: Azure Functions Local Admin And HTTP Invoke Slice

**Docs verified:** Microsoft Learn Azure Functions docs:
- `https://learn.microsoft.com/en-us/azure/azure-functions/functions-overview`
- `https://learn.microsoft.com/en-us/azure/azure-functions/functions-bindings-http-webhook-trigger`
- `https://learn.microsoft.com/en-us/azure/azure-functions/functions-deployment-technologies`
- `https://learn.microsoft.com/en-us/azure/azure-functions/functions-develop-local`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/functions/FunctionsServiceHandler.java`
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/functions/FunctionModels.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/FunctionsCompatibilityTest.java`

- [x] **Step 1: Verify floci local Functions contracts**

Confirm floci routes `/{account}-functions/admin/apps/...` for app and function metadata management and `/{account}-functions/api/{app}/{function}` for HTTP trigger invocation. Management tests run without Docker; execution-backed invocation is Docker-dependent in floci.

- [x] **Step 2: Write failing local Functions tests**

Add routing coverage for `-functions` local paths and App Service coverage for function app create/get/list/delete, function deploy/get/list/delete, generated invoke URL, and deterministic local HTTP invocation JSON.

- [x] **Step 3: Verify local Functions red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureFunctionsLocalDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestAzureFunctionsLocalAdminAndInvokeLifecycle -count=1
```

Observed red results before implementation: routing classified `-functions` local requests as AWS/S3, and App Service returned `404 NotFound` for the local Functions route.

- [x] **Step 4: Implement local Functions support**

Add `Microsoft.Web/functions` service-key registration, Azure route detection/action mapping for `-functions` paths, in-memory local function app/function state, floci-compatible admin JSON shapes, `Ready` versus `AwaitingDeploy` deploy status, app/function environment merge, delete cleanup, and a deterministic mock HTTP invocation response. This intentionally does not execute Docker containers yet.

- [x] **Step 5: Verify local Functions green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureFunctionsLocalDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestAzureFunctionsLocalAdminAndInvokeLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 69: App Service Site Config Linux FX Version Bridge

**Docs verified:** Microsoft Learn App Service and Azure Functions docs:
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/get-configuration`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/update-configuration`
- `https://learn.microsoft.com/en-us/azure/azure-functions/functions-develop-local`

**floci-az reference files inspected:**
- `/private/tmp/floci-az/src/main/java/io/floci/az/services/arm/ArmHandler.java`
- `/private/tmp/floci-az/compatibility-tests/sdk-test-java/src/test/java/io/floci/az/compat/FunctionsLinuxFxVersionCompatibilityTest.java`

- [x] **Step 1: Verify App Service config bridge contract**

Confirm floci persists `properties.siteConfig.linuxFxVersion` on ARM `Microsoft.Web/sites` creation, returns it from `sites/{name}/config/web`, rejects malformed values, and bridges valid Functions runtime metadata into the local Functions admin app.

- [x] **Step 2: Write failing App Service config tests**

Add App Service tests for `linuxFxVersion` persistence, `GET config/web`, local Functions admin bridge to runtime `python`, `PUT config/web` updating the bridged runtime to `node`, and malformed `python-3.12` rejection.

- [x] **Step 3: Verify App Service config red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run 'TestFunctionAppSite(ConfigLinuxFxVersionBridgesLocalFunctionApp|RejectsMalformedLinuxFxVersion)' -count=1
```

Observed red results before implementation: `GET config/web` returned `404 NotFound`, and malformed `linuxFxVersion` was accepted with `201 Created`.

- [x] **Step 4: Implement App Service config bridge**

Add `sites/{name}/config/web` route parsing, config get/update handlers, Linux FX version validation for supported Functions runtimes, config update projection, and sync valid site config runtime metadata to `devstoreaccount1-functions/admin/apps/{app}`.

- [x] **Step 5: Verify App Service config green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run 'TestFunctionAppSite(ConfigLinuxFxVersionBridgesLocalFunctionApp|RejectsMalformedLinuxFxVersion)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -count=1
```

Expected: all commands pass.

## Task 70: App Service Site Start Stop Restart Actions

**Docs verified:** Microsoft Learn App Service docs:
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/start?view=rest-appservice-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/stop?view=rest-appservice-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/restart?view=rest-appservice-2024-04-01`

- [x] **Step 1: Verify App Service action contracts**

Confirm Microsoft Learn documents `POST sites/{name}/start`, `POST sites/{name}/stop`, and `POST sites/{name}/restart`, each returning `200` on success. Restart accepts optional `softRestart` and `synchronous` query parameters.

- [x] **Step 2: Write failing App Service action tests**

Add routing coverage for normalized `StartSite`, `StopSite`, and `RestartSite` action names and service coverage for site state transitions from Running to Stopped to Running, restart metadata, and missing-site `404`.

- [x] **Step 3: Verify App Service action red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppServiceSiteActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestAppServiceSiteStartStopAndRestartUpdateState -count=1
```

Observed red results before implementation: routing returned raw lowercase path segments (`start`, `stop`, `restart`) and App Service returned `405 MethodNotAllowed` for site action routes.

- [x] **Step 4: Implement App Service site actions**

Add explicit route action detection, service actions for start/stop/restart, site action handlers that mutate stored site state, restart timestamp/count metadata, missing-resource handling, and bridged local Function App status synchronization.

- [x] **Step 5: Verify App Service action green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppServiceSiteActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestAppServiceSiteStartStopAndRestartUpdateState -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 71: App Service App Settings And Connection Strings Config

**Docs verified:** Microsoft Learn App Service docs:
- `https://learn.microsoft.com/en-us/azure/app-service/configure-common`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/list-application-settings?view=rest-appservice-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/update-application-settings?view=rest-appservice-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/list-connection-strings?view=rest-appservice-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/update-connection-strings?view=rest-appservice-2024-04-01`

- [x] **Step 1: Verify App Service config resource contracts**

Confirm Microsoft Learn documents App Service app settings and connection strings as app configuration, with REST operations for listing and updating application settings and connection strings under `Microsoft.Web/sites/config`.

- [x] **Step 2: Write failing App Service config tests**

Add App Service tests for seeded `siteConfig.appSettings` and `siteConfig.connectionStrings` projection into dedicated config resources, `POST config/appsettings/list`, `PUT config/appsettings`, `POST config/connectionstrings/list`, and `PUT config/connectionstrings` replacement behavior. Add routing tests for the corresponding normalized action names.

- [x] **Step 3: Verify App Service config red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run 'TestAppServiceSite(AppSettings|ConnectionStrings)ConfigLifecycle' -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppServiceConfigActions -count=1
```

Observed red results before implementation: both new list endpoints returned `404 NotFound` with `The App Service route is not implemented.` Routing returned generic `CreateOrUpdate` for config `PUT` calls and raw `list` for config list calls.

- [x] **Step 4: Implement App Service app settings and connection strings config resources**

Add `config/appsettings` and `config/connectionstrings` route parsing, list/update handlers, dedicated in-memory stores, site-create seeding from `siteConfig`, site-delete cleanup, deterministic `Microsoft.Web/sites/config` resource envelopes, provider action declarations, and explicit routing target detection while projecting updates back into stored `siteConfig`.

- [x] **Step 5: Verify App Service config green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run 'TestAppServiceSite(AppSettings|ConnectionStrings)ConfigLifecycle' -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureAppService(Config|Site)Actions' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -count=1
```

Expected: all commands pass.

## Task 72: Table Storage Explicit MERGE Entity Updates

**Docs and local reference verified:**
- Microsoft Learn `Merge Entity`: `https://learn.microsoft.com/en-us/rest/api/storageservices/merge-entity`
- floci-az table handlers: `/private/tmp/floci-az/src/main/java/io/floci/az/services/table/TableServiceHandler.java` and `/private/tmp/floci-az/src/main/java/io/floci/az/services/cosmos/table/CosmosTableApiHandler.java`, both treating `PATCH` and `MERGE` as partial entity updates.

- [x] **Step 1: Verify Table MERGE compatibility target**

Confirm Azure Table Storage exposes a REST `MERGE` operation and floci-az accepts both `PATCH` and `MERGE` for partial entity updates on table routes.

- [x] **Step 2: Write failing Table MERGE test**

Add a storage data-plane test on the floci-style local `/{account}-table` route that creates a table, inserts an entity, sends a literal `MERGE` request with only the changed field and `If-Match: *`, and verifies existing properties are preserved. Add routing coverage for detecting `MERGE` table requests as the stable `Merge` action.

- [x] **Step 3: Verify Table MERGE red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneMergeVerbPartiallyUpdatesEntity -count=1
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureTableDataPlaneMerge -count=1
```

Observed red results before implementation: the handler first returned `405 MethodNotAllowed` for `MERGE`; after accepting the verb, the same test exposed `400 PropertiesNeedValue` because updates required `PartitionKey` and `RowKey` in the body even when the URL carried the entity key. Routing returned an empty action for the literal `MERGE` method.

- [x] **Step 4: Implement explicit MERGE support**

Accept `MERGE` in the table entity update route, map it to the existing partial-merge path, let entity update parsing derive missing `PartitionKey`/`RowKey` values from the URL while keeping inserts strict, and normalize `MERGE` target actions as `Merge`.

- [x] **Step 5: Verify Table MERGE green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(TableDataPlaneMergeVerbPartiallyUpdatesEntity|DetectTarget_AzureTableDataPlaneMerge)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 73: Blob Storage ETag Conditional Reads And Deletes

**Docs and local reference verified:**
- Microsoft Learn `Get Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob`
- Microsoft Learn `Delete Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-blob`
- Microsoft Learn conditional headers: `https://learn.microsoft.com/en-us/rest/api/storageservices/specifying-conditional-headers-for-blob-service-operations`
- floci-az blob compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/BlobServiceTest.java`, covering `If-Match` GET and `If-None-Match` DELETE.

- [x] **Step 1: Verify blob conditional contracts**

Confirm Azure Blob Storage documents conditional headers for service operations and floci-az expects stale ETag conditions to return `412` with `x-ms-error-code: ConditionNotMet`.

- [x] **Step 2: Write failing blob conditional test**

Add a blob data-plane test that uploads a blob, verifies matching `If-Match` GET succeeds, verifies stale `If-Match` GET returns `412 ConditionNotMet`, verifies matching `If-None-Match` DELETE returns `412 ConditionNotMet`, and verifies the blob still exists after the blocked delete.

- [x] **Step 3: Verify blob conditional red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneConditionalGetAndDeleteHonorETags -count=1
```

Observed red results before implementation: stale `If-Match` GET ignored the condition and returned `200` with the blob body.

- [x] **Step 4: Implement blob ETag conditions**

Add shared blob conditional-header checks for `If-Match` and `If-None-Match`, enforce them before GET range processing and before DELETE mutation, and return Azure Storage-compatible `412 ConditionNotMet` responses with `x-ms-error-code`.

- [x] **Step 5: Verify blob conditional green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneConditionalGetAndDeleteHonorETags -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 74: Blob Storage List Include Metadata

**Docs and local reference verified:**
- Microsoft Learn `List Blobs`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-blobs`
- floci-az blob compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/BlobServiceTest.java`, covering `comp=list&include=metadata` XML output.

- [x] **Step 1: Verify blob listing metadata target**

Confirm Azure Blob Storage `List Blobs` supports the `include` query parameter and floci-az expects metadata to appear in XML listings when `include=metadata`.

- [x] **Step 2: Write failing blob metadata listing test**

Add a blob data-plane test that uploads a blob with `x-ms-meta-owner`, lists the container with `restype=container&comp=list&include=metadata`, and asserts the XML contains `<Metadata>` and the metadata key/value element.

- [x] **Step 3: Verify blob metadata listing red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobListCanIncludeMetadata -count=1
```

Observed red results before implementation: `List Blobs` returned XML with blob properties but no `<Metadata>` block.

- [x] **Step 4: Implement blob listing metadata**

Parse the `include` query parameter, carry blob metadata into list items only when `metadata` is requested, and emit stable sorted XML metadata elements.

- [x] **Step 5: Verify blob metadata listing green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobListCanIncludeMetadata -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 75: Blob Storage Get And Set Metadata Routes

**Docs and local reference verified:**
- Microsoft Learn `Set Blob Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-blob-metadata`
- Microsoft Learn `Get Blob Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-metadata`
- floci-az blob compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/BlobServiceTest.java`, covering `PUT` and `GET` `?comp=metadata`.

- [x] **Step 1: Verify blob metadata route target**

Confirm Azure Blob Storage exposes dedicated `comp=metadata` operations and floci-az expects metadata replacement/get through `PUT` and `GET` on that route.

- [x] **Step 2: Write failing blob metadata route test**

Add a blob data-plane test that uploads a blob with initial metadata, sends `PUT ?comp=metadata` with replacement `x-ms-meta-*` headers, verifies `200` and changed ETag, sends `GET ?comp=metadata`, verifies metadata headers and no body, and verifies a normal blob GET still returns the original body/content type with updated metadata headers.

- [x] **Step 3: Verify blob metadata route red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSetAndGetMetadata -count=1
```

Observed red results before implementation: `PUT ?comp=metadata` fell through to blob upload and returned `201 Created`, replacing the blob content path instead of metadata-only state.

- [x] **Step 4: Implement blob metadata routes**

Dispatch `comp=metadata` before normal blob upload/download, implement `PUT` metadata replacement with ETag/last-modified updates and condition checks, and implement `GET` metadata responses with storage headers and `x-ms-meta-*` values but no body.

- [x] **Step 5: Verify blob metadata route green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSetAndGetMetadata -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 76: Queue Storage Local Route Metadata And Idempotent Create

**Docs and local reference verified:**
- Microsoft Learn `Create Queue`: `https://learn.microsoft.com/en-us/rest/api/storageservices/create-queue4`
- Microsoft Learn `Get Queue Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-queue-metadata`
- Microsoft Learn `Set Queue Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-queue-metadata`
- floci-az queue compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/QueueServiceTest.java`, covering existing queue create behavior and `PUT`/`GET` `?comp=metadata`.

- [x] **Step 1: Verify queue metadata and local route target**

Confirm Azure Queue Storage exposes queue create metadata headers, existing-queue status semantics, and dedicated `comp=metadata` operations. Confirm floci-az uses local `/{account}-queue/{queue}` routes for SDK compatibility.

- [x] **Step 2: Write failing queue local route and metadata tests**

Add a queue data-plane service test that creates `http://localhost:4577/devstoreaccount1-queue/work` with metadata, recreates the same queue, verifies preserved metadata through `GET ?comp=metadata`, replaces metadata with `PUT ?comp=metadata`, and verifies the updated metadata plus `x-ms-approximate-messages-count`. Add a routing test proving local `-queue` paths target `Microsoft.Storage/queueServices`.

- [x] **Step 3: Verify queue local route and metadata red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(QueueDataPlaneLocalRouteMetadataAndIdempotentCreate|DetectTarget_AzureStorageDataPlaneHosts)' -count=1
```

Observed red results before implementation: the local `-queue` storage request returned the ARM-route `404` path, and target detection classified the local route as the AWS S3 fallback instead of Azure Queue Storage.

- [x] **Step 4: Implement queue local route, idempotent create, and metadata**

Recognize local `/{account}-queue` paths in storage data-plane account detection and routing, strip the account segment before queue dispatch, preserve metadata when an existing queue is recreated, and implement `PUT`/`GET` `?comp=metadata` with Azure Queue Storage headers.

- [x] **Step 5: Verify queue local route and metadata green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(QueueDataPlaneLocalRouteMetadataAndIdempotentCreate|DetectTarget_AzureStorageDataPlaneHosts)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 77: Queue Storage Delete And List Queues With Metadata

**Docs and local reference verified:**
- Microsoft Learn `Delete Queue`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-queue3`
- Microsoft Learn `List Queues`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-queues1`
- floci-az queue compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/QueueServiceTest.java`, covering `createAndDeleteQueue` and `listQueuesIncludesMetadataWhenRequested`.

- [x] **Step 1: Verify queue delete and list targets**

Confirm Azure Queue Storage exposes `DELETE /{queue}` with `204 No Content`, and account-level `GET ?comp=list` with stable alphabetical queue XML plus optional `include=metadata` blocks.

- [x] **Step 2: Write failing queue delete and list tests**

Add a queue data-plane test that creates a local floci-style queue with metadata and a message, deletes the queue, and verifies subsequent metadata reads return `404`. Add a separate queue list test that creates multiple local queues, calls `GET /{account}-queue?comp=list&include=metadata`, and verifies alphabetical names plus metadata XML.

- [x] **Step 3: Verify queue delete and list red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueueDataPlane(DeleteQueueRemovesQueueState|ListQueuesCanIncludeMetadata)' -count=1
```

Observed red results before implementation: `DELETE /{queue}` returned the queue route `404`, and account-level `GET ?comp=list` returned `400 InvalidUri`.

- [x] **Step 4: Implement queue delete and list**

Add account-level queue list dispatch before invalid URI handling, delete queue state on `DELETE /{queue}`, emit Azure `EnumerationResults` XML with stable ordering, `ServiceEndpoint`, optional metadata, and basic `prefix`, `marker`, and `maxresults` support.

- [x] **Step 5: Verify queue delete and list green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueueDataPlane(DeleteQueueRemovesQueueState|ListQueuesCanIncludeMetadata)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 78: Queue Storage Message Clear Update And Count Validation

**Docs and local reference verified:**
- Microsoft Learn `Get Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages`
- Microsoft Learn `Clear Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/clear-messages`
- Microsoft Learn `Update Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/update-message`
- floci-az queue compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/QueueServiceTest.java`, covering `numOfMessagesValidation`, `clearMessages`, and `updateMessageChangesTextAndPopReceipt`.

- [x] **Step 1: Verify queue message operation targets**

Confirm Azure Queue Storage requires `numofmessages` to be a nonzero integer no greater than 32, supports `DELETE /{queue}/messages` for clearing a queue, and supports `PUT /{queue}/messages/{id}` with required `popreceipt` and `visibilitytimeout` that returns a new pop receipt.

- [x] **Step 2: Write failing queue message tests**

Add tests for invalid `numofmessages=0` and `33`, clearing all messages from a local floci-style queue, and updating a dequeued message so its text changes, stale pop receipts are rejected, and the returned pop receipt can delete the message.

- [x] **Step 3: Verify queue message red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueue(GetMessagesRejectsInvalidNumOfMessages|ClearMessagesDeletesAllMessages|UpdateMessageChangesTextAndPopReceipt)' -count=1
```

Observed red results before implementation: invalid `numofmessages=0` returned `200`, `DELETE /messages` returned the queue route `404`, and `PUT /messages/{id}` returned the queue route `404`.

- [x] **Step 4: Implement queue message clear, update, and validation**

Validate `numofmessages` in receive and peek paths, route `DELETE /messages` to clear queue message state, and route `PUT /messages/{id}` to update message text, validate the current pop receipt, rotate the pop receipt, and emit `x-ms-time-next-visible`.

- [x] **Step 5: Verify queue message green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueue(GetMessagesRejectsInvalidNumOfMessages|ClearMessagesDeletesAllMessages|UpdateMessageChangesTextAndPopReceipt)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 79: Blob Storage Local Route Container Conflict And Delete

**Docs and local reference verified:**
- Microsoft Learn `Blob Service REST API`: `https://learn.microsoft.com/en-us/rest/api/storageservices/blob-service-rest-api`
- Microsoft Learn `Create Container`: `https://learn.microsoft.com/en-us/rest/api/storageservices/create-container`
- Microsoft Learn `Delete Container`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-container`
- floci-az blob compatibility tests: `/private/tmp/floci-az/src/test/java/io/floci/az/services/BlobServiceTest.java`, covering `/{account}/{container}?restype=container`, duplicate container create conflict, blob put/get, and container delete.

- [x] **Step 1: Verify local Blob route and container lifecycle target**

Confirm floci-az uses Azurite-style localhost paths without a `-blob` suffix, while Microsoft Learn defines container create conflict and delete container status semantics for Blob Storage REST.

- [x] **Step 2: Write failing local Blob service and routing tests**

Add a storage test that creates `http://localhost:4577/devstoreaccount1/docs?restype=container`, verifies duplicate create returns `409`, writes and reads a local blob, deletes the container with `202`, and verifies the child blob is gone. Add a routing test proving the same local Blob path targets `Microsoft.Storage/blobServices`.

- [x] **Step 3: Verify local Blob red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(BlobDataPlaneLocalRouteContainerConflictAndDelete|DetectTarget_AzureStorageDataPlaneHosts)' -count=1
```

Observed red results before implementation: the local Blob service request fell through to the storage account control-plane `404`, and routing classified the path as AWS S3 instead of Azure Blob Storage.

- [x] **Step 4: Implement local Blob routing and container lifecycle**

Detect localhost `devstoreaccount1` and `-blob` path prefixes as Azure Blob data-plane requests, strip the local account segment before Blob dispatch, route them to `Microsoft.Storage/blobServices`, return `409 ContainerAlreadyExists` on duplicate creates, and delete container state with `202 Accepted`.

- [x] **Step 5: Verify local Blob green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(BlobDataPlaneLocalRouteContainerConflictAndDelete|DetectTarget_AzureStorageDataPlaneHosts)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 80: Blob Storage Container Metadata Routes

**Docs and local reference verified:**
- Microsoft Learn `Set Container Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-container-metadata`
- Microsoft Learn `Get Container Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-container-metadata`

- [x] **Step 1: Verify container metadata target**

Confirm Azure Blob Storage exposes `PUT` and `GET/HEAD` on `?restype=container&comp=metadata`, including the local emulator URI shape, and that setting metadata replaces all existing container metadata while updating the container ETag.

- [x] **Step 2: Write failing container metadata test**

Add a local Blob data-plane test that creates a container with metadata, reads metadata through `GET ?restype=container&comp=metadata`, replaces metadata through `PUT ?restype=container&comp=metadata`, verifies the ETag changes, and verifies the old metadata key is removed.

- [x] **Step 3: Verify container metadata red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobContainerDataPlaneSetAndGetMetadata -count=1
```

Observed red result before implementation: `GET ?restype=container&comp=metadata` fell through to blob lookup and returned `404 BlobNotFound`.

- [x] **Step 4: Implement container metadata routes**

Dispatch `comp=metadata` before normal container create/list/delete behavior, implement metadata replacement with ETag/last-modified updates, return metadata headers without a body for `GET/HEAD`, and reuse Blob condition checks for ETag preconditions.

- [x] **Step 5: Verify container metadata green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobContainerDataPlaneSetAndGetMetadata -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -count=1
```

Expected: all commands pass.

## Task 81: Blob Storage List Containers And Container Properties

**Docs and local reference verified:**
- Microsoft Learn `List Containers`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-containers2`
- Microsoft Learn `Get Container Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-container-properties`

- [x] **Step 1: Verify list containers and properties target**

Confirm Azure Blob Storage exposes account-root `GET ?comp=list`, including the local emulator URI `/{account}?comp=list`, and `GET/HEAD ?restype=container` for container properties. Confirm list responses are alphabetical XML and can include container metadata plus continuation markers.

- [x] **Step 2: Write failing list containers and properties tests**

Add a routing test that `http://localhost:4577/devstoreaccount1?comp=list` targets Azure Blob Storage as a `List` action. Add storage tests that read `GET ?restype=container` properties without a body, and list local account containers with `include=metadata`, `maxresults=1`, `NextMarker`, and a second marker page.

- [x] **Step 3: Verify list containers and properties red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(BlobContainerDataPlaneGetProperties|BlobDataPlaneListContainersCanIncludeMetadataAndPage|DetectTarget_AzureStorageLocalBlobListContainers)' -count=1
```

Observed red results before implementation: container properties returned `404 BlobNotFound`, account-root list containers returned storage account control-plane `404`, and local account-root routing classified the request as AWS S3.

- [x] **Step 4: Implement list containers and properties**

Detect local account-root Blob list requests, strip normalized localhost account segments correctly, add `GET/HEAD ?restype=container` properties handling, and add account-level container listing XML with metadata, stable sorting, `ServiceEndpoint`, and marker/maxresults pagination.

- [x] **Step 5: Verify list containers and properties green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -run 'Test(BlobContainerDataPlaneGetProperties|BlobDataPlaneListContainersCanIncludeMetadataAndPage|DetectTarget_AzureStorageLocalBlobListContainers)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 82: Blob Storage List MaxResults Validation

**Docs and local reference verified:**
- Microsoft Learn `List Containers`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-containers2`
- Microsoft Learn `List Blobs`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-blobs`

- [x] **Step 1: Verify list maxresults validation target**

Confirm both Blob Storage list operations document `maxresults` values less than or equal to zero as `400 Bad Request`.

- [x] **Step 2: Write failing list maxresults tests**

Add tests for local account-level `GET /{account}?comp=list&maxresults=0` and container-level `GET ?restype=container&comp=list&maxresults=0`, expecting `400 Bad Request`.

- [x] **Step 3: Verify list maxresults red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestBlob(ListContainersRejectsInvalidMaxResults|ListRejectsInvalidMaxResults)' -count=1
```

Observed red results before implementation: both invalid list requests returned `200 OK` and normal XML listing bodies.

- [x] **Step 4: Implement list maxresults validation**

Add shared Blob list parsing for `maxresults`, returning `400 InvalidQueryParameterValue` for nonnumeric or nonpositive values before list XML generation.

- [x] **Step 5: Verify list maxresults green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestBlob(ListContainersRejectsInvalidMaxResults|ListRejectsInvalidMaxResults)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 83: Blob Storage Get Blob Properties

**Docs and local reference verified:**
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify blob properties target**

Confirm Azure Blob Storage defines `HEAD` on a blob URI as `Get Blob Properties`, returning metadata, HTTP properties, system properties, and no blob content body.

- [x] **Step 2: Write failing blob properties test**

Add a local Blob data-plane test that uploads a text blob with metadata, sends `HEAD /{account}/{container}/{blob}`, and verifies status `200`, no body, content length/type, blob type, metadata, ETag, last-modified, and `Accept-Ranges`.

- [x] **Step 3: Verify blob properties red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneHeadReturnsPropertiesWithoutBody -count=1
```

Observed red result before implementation: `HEAD` returned `405 MethodNotAllowed` for blob routes.

- [x] **Step 4: Implement blob properties**

Route `HEAD` blob requests to a properties-only handler, build shared Blob property headers for GET/HEAD paths, and return no body for HEAD while preserving ETag condition checks.

- [x] **Step 5: Verify blob properties green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneHeadReturnsPropertiesWithoutBody -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 84: Blob Storage Set Blob Properties

**Docs and local reference verified:**
- Microsoft Learn `Set Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-blob-properties`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify blob property update target**

Confirm Azure Blob Storage exposes `PUT ?comp=properties`, updates blob HTTP/system properties, returns `200 OK` with new ETag/last-modified headers, preserves blob content, and reflects updated properties through `HEAD`/Get Blob Properties and normal `GET Blob`.

- [x] **Step 2: Write failing blob property update test**

Add a local Blob data-plane test that uploads a text blob with metadata, calls `PUT ?comp=properties` with content type, cache control, content language, content encoding, and content disposition headers, verifies a changed ETag and no body, then verifies `HEAD` and `GET` preserve bytes and metadata while reflecting new properties.

- [x] **Step 3: Verify blob property update red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSetPropertiesUpdatesHeadersAndPreservesContent -count=1
```

Observed red result before implementation: `PUT ?comp=properties` fell through to normal blob upload and returned `201 Created`.

- [x] **Step 4: Implement blob property update route**

Dispatch `comp=properties` before normal upload, persist Azure blob HTTP property fields separately from content and metadata, update blob/container ETags and last-modified timestamps, return `200 OK`, and reuse the stored fields in Get Blob and Get Blob Properties responses.

- [x] **Step 5: Verify blob property update green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSetPropertiesUpdatesHeadersAndPreservesContent -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 85: Blob Storage Put Blob Content Property Headers

**Docs and local reference verified:**
- Microsoft Learn `Put Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-blob`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify Put Blob property header target**

Confirm `Put Blob` accepts standard content headers and Azure `x-ms-blob-content-*` headers for content type, cache control, encoding, language, MD5, and content disposition, and that these values are returned by `Get Blob Properties`.

- [x] **Step 2: Write failing Put Blob property header test**

Add a local Blob data-plane test that uploads a blob with `x-ms-blob-content-type`, `x-ms-blob-cache-control`, `x-ms-blob-content-language`, `x-ms-blob-content-encoding`, `x-ms-blob-content-md5`, and `x-ms-blob-content-disposition`, then verifies `HEAD` returns those properties.

- [x] **Step 3: Verify Put Blob property header red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlanePutBlobStoresAzureContentPropertyHeaders -count=1
```

Observed red result before implementation: only `x-ms-blob-content-type` was reflected; cache control, language, encoding, MD5, and disposition headers were missing from `HEAD`.

- [x] **Step 4: Implement Put Blob property header capture**

Capture the Azure `x-ms-blob-content-*` property headers on `Put Blob` with standard HTTP header fallbacks, store them on blob state, and return them from `Get Blob` and `Get Blob Properties`.

- [x] **Step 5: Verify Put Blob property header green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlanePutBlobStoresAzureContentPropertyHeaders -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 86: Blob Storage Put Block And Put Block List

**Docs and local reference verified:**
- Microsoft Learn `Put Block`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-block`
- Microsoft Learn `Put Block List`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-block-list`
- Microsoft Learn `Get Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify staged block target**

Confirm `Put Block` stages bytes under a URL-encoded Base64 `blockid`, `Put Block List` commits ordered XML `Committed`, `Uncommitted`, or `Latest` block IDs into a block blob, successful operations return `201 Created`, and committed blobs expose content properties and metadata through read/property responses.

- [x] **Step 2: Write failing staged block commit test**

Add a local Blob data-plane test that stages two blocks with `?comp=block&blockid=...`, commits them with `?comp=blocklist`, and verifies `GET Blob` returns concatenated content, `text/plain` content type, metadata, content length, and block blob system headers.

- [x] **Step 3: Verify staged block red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlanePutBlockListCommitsStagedBlocks -count=1
```

Observed red result before implementation: `PUT ?comp=blocklist` fell through to normal `Put Blob` handling and stored the XML block list body as blob content.

- [x] **Step 4: Implement staged block commit behavior**

Dispatch `comp=block` and `comp=blocklist` before normal blob uploads, store uncommitted blocks per container/blob/block ID, parse ordered XML block lists, concatenate referenced blocks, persist committed blob properties and metadata, clear staged block state after commit or direct `Put Blob`, and return Azure Storage headers.

- [x] **Step 5: Verify staged block green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlanePutBlockListCommitsStagedBlocks -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 87: Blob Storage Get Block List

**Docs and local reference verified:**
- Microsoft Learn `Get Block List`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-block-list`
- Microsoft Learn `Put Block`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-block`
- Microsoft Learn `Put Block List`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-block-list`

- [x] **Step 1: Verify block-list read target**

Confirm `Get Block List` accepts `blocklisttype=committed`, `uncommitted`, or `all`, defaults to committed blocks when omitted, returns `200 OK`, uses Azure XML `BlockList` with `CommittedBlocks` and/or `UncommittedBlocks`, preserves committed order from `Put Block List`, sorts uncommitted block IDs alphabetically, and includes committed blob headers such as ETag, last-modified, and `x-ms-blob-content-length`.

- [x] **Step 2: Write failing Get Block List test**

Add a local Blob data-plane test that commits two staged blocks, stages a later uncommitted block for the same blob, calls `GET ?comp=blocklist&blocklisttype=all`, and verifies committed and uncommitted XML entries, block sizes, response content type, committed blob length, ETag, and last-modified headers.

- [x] **Step 3: Verify Get Block List red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneGetBlockListReturnsCommittedAndUncommittedBlocks -count=1
```

Observed red result before implementation: `GET ?comp=blocklist` returned `405 MethodNotAllowed` because only block-list commits were routed.

- [x] **Step 4: Implement Get Block List response**

Store committed block IDs and byte sizes with committed blob state, dispatch `GET ?comp=blocklist`, support `blocklisttype` filtering, render Azure XML block-list groups, return committed blob headers, and keep uncommitted block staging independent from the committed blob content.

- [x] **Step 5: Verify Get Block List green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneGetBlockListReturnsCommittedAndUncommittedBlocks -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 88: Blob Storage Set And Get Blob Tags

**Docs and local reference verified:**
- Microsoft Learn `Set Blob Tags`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-blob-tags`
- Microsoft Learn `Get Blob Tags`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-tags`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify blob tag target**

Confirm `Set Blob Tags` uses `PUT ?comp=tags`, accepts Azure XML `<Tags><TagSet><Tag><Key>...</Key><Value>...</Value></Tag></TagSet></Tags>`, returns `204 No Content`, replaces the full tag set, and does not change the blob ETag or last-modified time. Confirm `Get Blob Tags` uses `GET ?comp=tags`, returns `200 OK`, and emits an XML tag document with `Content-Length`.

- [x] **Step 2: Write failing Set/Get Blob Tags test**

Add a local Blob data-plane test that uploads a blob, captures `HEAD` properties, sets two tags with Azure XML, verifies `204 No Content`, verifies `HEAD` still has the same ETag and last-modified time, and verifies `GET ?comp=tags` returns the tag XML.

- [x] **Step 3: Verify blob tag red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSetAndGetBlobTags -count=1
```

Observed red result before implementation: `PUT ?comp=tags` fell through to normal `Put Blob` handling and returned `201 Created`.

- [x] **Step 4: Implement blob tag storage and routes**

Dispatch `comp=tags` before normal blob upload, parse Azure XML tag documents, store tags separately from blob metadata, return `204 No Content` for updates without changing blob ETag or last-modified time, and render deterministic Azure XML for tag reads.

- [x] **Step 5: Verify blob tag green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSetAndGetBlobTags -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 89: Blob Storage Find Blobs By Tags

**Docs and local reference verified:**
- Microsoft Learn `Find Blobs by Tags`: `https://learn.microsoft.com/en-us/rest/api/storageservices/find-blobs-by-tags`
- Microsoft Learn `Set Blob Tags`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-blob-tags`
- Microsoft Learn `Get Blob Tags`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-tags`

- [x] **Step 1: Verify account-level tag query target**

Confirm `Find Blobs by Tags` uses account-root `GET ?comp=blobs&where=<expression>`, returns `200 OK`, emits XML `EnumerationResults` with `Where`, `Blobs`, `Blob`, `Name`, `ContainerName`, matching `Tags`, and `NextMarker`, supports `maxresults`/`marker`, rejects nonpositive `maxresults`, and supports the documented expression shape for `@container`, quoted tag names, string comparison operators, and `AND`.

- [x] **Step 2: Write failing Find Blobs by Tags test**

Add a local Blob data-plane test that creates two containers, uploads three blobs, sets blob tags, queries the local account root with `@container='docs' AND "env"='test'`, and verifies only the matching blob is returned with XML tags and response headers.

- [x] **Step 3: Verify Find Blobs by Tags red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneFindBlobsByTags -count=1
```

Observed red result before implementation: local account-root `?comp=blobs` was classified as an unimplemented storage-account route and returned `404 NotFound`.

- [x] **Step 4: Implement account-root tag query**

Classify local account-root `comp=blobs` requests as Blob data-plane traffic, dispatch account-root Blob requests to `findBlobsByTags`, parse the supported tag where-expression subset, filter stored blob tag maps across containers, sort results deterministically, apply `marker` and `maxresults`, and render Azure XML with `Content-Length` and `x-ms-version` headers.

- [x] **Step 5: Verify Find Blobs by Tags green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneFindBlobsByTags -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 90: Blob Storage Lease Blob Acquire And Release

**Docs and local reference verified:**
- Microsoft Learn `Lease Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/lease-blob`
- Microsoft Learn `Put Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-blob`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify lease target**

Confirm `Lease Blob` uses `PUT ?comp=lease`, supports `x-ms-lease-action: acquire` and `release`, returns `201 Created` for acquire and `200 OK` for release, returns `x-ms-lease-id` on acquire, does not change ETag or last-modified values, reports lease status/state through blob properties, and requires the active lease ID for blob write and delete operations.

- [x] **Step 2: Write failing lease test**

Add a local Blob data-plane test that uploads a blob, captures `HEAD` properties, acquires an infinite lease with a proposed lease ID, verifies lease headers and unchanged ETag/last-modified values, verifies `HEAD` reports `locked/leased`, verifies `Put Blob` without the lease fails with `412`, verifies `Put Blob` with the lease succeeds, releases the lease, and verifies properties return to `unlocked/available`.

- [x] **Step 3: Verify lease red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneLeaseAcquireReleaseAndWriteEnforcement -count=1
```

Observed red result before implementation: `PUT ?comp=lease` fell through to normal `Put Blob` handling, returned `201 Created` without `x-ms-lease-id`, and changed the blob ETag.

- [x] **Step 4: Implement lease state and enforcement**

Dispatch `comp=lease` before normal blob writes, store active lease IDs on blob objects, support acquire and release actions, preserve blob ETag/last-modified values during lease operations, report lease state/status in property headers, preserve active leases across authorized overwrites, and reject write/delete operations that omit or mismatch the active lease ID.

- [x] **Step 5: Verify lease green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneLeaseAcquireReleaseAndWriteEnforcement -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 91: Blob Storage Lease Blob Renew Change And Break

**Docs and local reference verified:**
- Microsoft Learn `Lease Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/lease-blob`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`
- Microsoft Learn `Put Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-blob`

- [x] **Step 1: Verify remaining lease-action target**

Confirm `Lease Blob` supports `x-ms-lease-action: renew`, `change`, and `break`; `renew` and `change` return `200 OK`, `break` returns `202 Accepted`, `renew` returns the active lease ID, `change` switches to `x-ms-proposed-lease-id`, immediate break returns `x-ms-lease-time: 0`, lease actions do not mutate ETag or last-modified values, and broken leases report `x-ms-lease-status: unlocked` with `x-ms-lease-state: broken`.

- [x] **Step 2: Write failing renew/change/break test**

Add a local Blob data-plane test that acquires a lease, renews it, changes it to a new lease ID, verifies the old lease ID no longer authorizes writes, breaks the lease immediately, verifies property headers report `broken`, and verifies a fresh lease can be acquired after break.

- [x] **Step 3: Verify renew/change/break red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneLeaseRenewChangeAndBreak -count=1
```

Observed red result before implementation: `x-ms-lease-action: renew` returned `400 InvalidHeaderValue` because only acquire and release were implemented.

- [x] **Step 4: Implement remaining lease actions**

Track explicit lease state on blobs, implement `renew`, `change`, and immediate `break`, preserve ETag and last-modified values, return lease headers required by each action, update property headers for `leased` and `broken` states, allow re-acquire after break, and keep write enforcement limited to active locked lease states.

- [x] **Step 5: Verify renew/change/break green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneLeaseRenewChangeAndBreak -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 92: Blob Storage Snapshot Blob Point-In-Time Reads

**Docs and local reference verified:**
- Microsoft Learn `Snapshot Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/snapshot-blob`
- Microsoft Learn `Get Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob`
- Microsoft Learn `Get Blob Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob-properties`

- [x] **Step 1: Verify snapshot target**

Confirm `Snapshot Blob` uses `PUT ?comp=snapshot`, returns `201 Created` with an opaque `x-ms-snapshot` value, copies the base blob's content, metadata, tags, committed block list, HTTP properties, ETag, and last-modified value when no snapshot metadata override is provided, does not copy the base blob lease, and lets later `GET Blob` and `Get Blob Properties` requests retrieve the snapshot using `snapshot=<DateTime>`.

- [x] **Step 2: Write failing snapshot test**

Add a local Blob data-plane test that uploads a text blob with metadata, snapshots it, overwrites the base blob with different content/properties/metadata, verifies `GET` with the snapshot value returns the original body/content type/metadata and `x-ms-snapshot`, verifies a normal base `GET` returns the updated blob, and verifies `HEAD` with the snapshot value returns the original length/content type/metadata.

- [x] **Step 3: Verify snapshot red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSnapshotBlobPreservesPointInTimeContent -count=1
```

Observed red result before implementation: `PUT ?comp=snapshot` fell through to normal `Put Blob` handling, returned `201 Created` without `x-ms-snapshot`, and overwrote the blob with an empty body.

- [x] **Step 4: Implement snapshot state and reads**

Dispatch `comp=snapshot` before normal blob writes, store snapshots per container and blob name, deep-copy blob content, metadata, tags, committed blocks, and HTTP properties, preserve snapshot ETag/last-modified values unless new snapshot metadata is supplied, clear lease state on snapshot objects, and resolve `GET`/`HEAD` requests through snapshot state when the `snapshot` query parameter is present.

- [x] **Step 5: Verify snapshot green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneSnapshotBlobPreservesPointInTimeContent -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 93: Blob Storage Delete Blob Snapshot Options

**Docs and local reference verified:**
- Microsoft Learn `Delete Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-blob`
- Microsoft Learn `Snapshot Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/snapshot-blob`
- Microsoft Learn `Get Blob`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-blob`

- [x] **Step 1: Verify snapshot delete target**

Confirm `Delete Blob` deletes either a base blob or a snapshot, requires `x-ms-delete-snapshots: include` or `only` when deleting a base blob with snapshots, returns `409 Conflict` when that header is missing, deletes only snapshots when the header value is `only`, deletes the base blob and snapshots when the header value is `include`, deletes a single snapshot with `snapshot=<DateTime>`, and treats `x-ms-delete-snapshots` on an individual snapshot delete as invalid.

- [x] **Step 2: Write failing snapshot delete test**

Add a local Blob data-plane test that creates a base blob and snapshot, verifies deleting the base blob without `x-ms-delete-snapshots` returns `409 SnapshotsPresent` and preserves both resources, verifies `x-ms-delete-snapshots: only` removes the snapshot but preserves the base blob, verifies deleting `?snapshot=<DateTime>` removes only that snapshot, and verifies `x-ms-delete-snapshots: include` removes the base blob and remaining snapshots.

- [x] **Step 3: Verify snapshot delete red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneDeleteBlobHonorsSnapshotOptions -count=1
```

Observed red result before implementation: deleting the base blob with snapshots returned `202 Accepted` and removed the base blob instead of returning `409 SnapshotsPresent`.

- [x] **Step 4: Implement snapshot-aware delete**

Route `DELETE` with query parameters into the delete handler, check `snapshot=<DateTime>` before base deletion, reject `x-ms-delete-snapshots` on individual snapshot deletes, delete snapshot map entries independently from base blobs, return `409 SnapshotsPresent` for base deletes without the required header, support `only` and `include`, and preserve existing lease and ETag condition enforcement for base deletes.

- [x] **Step 5: Verify snapshot delete green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestBlobDataPlaneDeleteBlobHonorsSnapshotOptions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 94: Queue Storage Put Message Validation

**Docs and local reference verified:**
- Microsoft Learn `Put Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-message`
- Microsoft Learn `Get Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages`
- Microsoft Learn `Peek Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/peek-messages`

- [x] **Step 1: Verify put-message validation target**

Confirm `Put Message` accepts local and hosted queue routes, validates `visibilitytimeout` as a nonnegative value no greater than seven days, validates `messagettl` as positive or `-1`, requires a positive TTL to be greater than the initial visibility timeout, rejects message text larger than 64 KiB, and does not enqueue invalid messages.

- [x] **Step 2: Write failing validation test**

Add a local Queue data-plane test that creates a queue, attempts invalid `Put Message` requests for negative visibility timeout, visibility timeout above seven days, zero TTL, visibility timeout equal to TTL, and an oversized message body, verifies each request returns `400 Bad Request`, and verifies the queue remains empty afterward.

- [x] **Step 3: Verify validation red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestQueuePutMessageValidatesVisibilityTTLAndSize -count=1
```

Observed red result before implementation: every invalid `Put Message` request returned `201 Created` and enqueued a message.

- [x] **Step 4: Implement validation helpers**

Validate the parsed XML message before locking/enqueueing, enforce the 64 KiB message-text limit, add queue-specific integer parsing that reports invalid query parameters instead of silently falling back, enforce `visibilitytimeout` and `messagettl` bounds, reject positive TTL values that are not greater than the visibility timeout, and represent `messagettl=-1` with a far-future expiration timestamp.

- [x] **Step 5: Verify validation green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestQueuePutMessageValidatesVisibilityTTLAndSize -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 95: Queue Storage Get Messages Receive Semantics

**Docs and local reference verified:**
- Microsoft Learn `Get Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages`
- Microsoft Learn `Delete Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-message2`
- Microsoft Learn `Update Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/update-message`

- [x] **Step 1: Verify get-message receive target**

Confirm `Get Messages` validates `numofmessages` and receive `visibilitytimeout`, returns `400 Bad Request` for out-of-range visibility timeout values, does not hide or remove messages when validation fails, rotates pop receipts on each receive, returns `DequeueCount` in received message XML, and increments `DequeueCount` each time the message is dequeued.

- [x] **Step 2: Write failing receive tests**

Add local Queue data-plane tests that reject `visibilitytimeout=0` and `visibilitytimeout=604801` without hiding/removing the message, and verify a message reports `DequeueCount=1` on first receive, can be made visible again through `Update Message`, then reports `DequeueCount=2` with a new pop receipt on the next receive.

- [x] **Step 3: Verify receive red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueueGetMessages(ValidatesVisibilityTimeout|IncrementsDequeueCount)' -count=1
```

Observed red result before implementation: `visibilitytimeout=0` returned `200 OK` and hid the message, and received messages omitted `DequeueCount` so the decoded count was `0`.

- [x] **Step 4: Implement receive validation and dequeue count**

Add queue message dequeue-count state, increment it only when `Get Messages` returns the message, include `DequeueCount` in `QueueMessage` XML, validate receive `visibilitytimeout` with a queue-specific parser instead of silent fallback parsing, and preserve existing pop-receipt rotation behavior.

- [x] **Step 5: Verify receive green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueueGetMessages(ValidatesVisibilityTimeout|IncrementsDequeueCount)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 96: Queue Storage Delete Message Pop Receipt Semantics

**Docs and local reference verified:**
- Microsoft Learn `Delete Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-message2`
- Microsoft Learn `Get Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages`
- Microsoft Learn `Update Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/update-message`

- [x] **Step 1: Verify delete-message target**

Confirm `Delete Message` requires the `popreceipt` query parameter, deletes only when the supplied receipt is the latest receipt returned by `Get Messages` or `Update Message`, returns `204 No Content` on success, and returns `404 Not Found` when a message with the supplied pop receipt is not found because the receipt is stale, the message was updated, the message was dequeued again, or the message has expired.

- [x] **Step 2: Write failing pop-receipt test**

Add a local Queue data-plane test that dequeues a message, verifies missing `popreceipt` is rejected with `400`, updates the message to rotate its pop receipt, verifies deleting with the old receipt returns `404`, and verifies deleting with the new current receipt succeeds with `204`.

- [x] **Step 3: Verify delete-message red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueue(DataPlaneLifecycle|DeleteMessageRequiresCurrentPopReceipt)' -count=1
```

Observed red result before implementation: wrong and stale pop receipts returned `400 PopReceiptMismatch` instead of `404 MessageNotFound`.

- [x] **Step 4: Implement current-receipt delete behavior**

Validate missing `popreceipt` before acquiring the queue lock, preserve successful deletes with the current receipt, return `404 MessageNotFound` when a message ID exists but the receipt is stale or non-current, and keep expired-message deletes as `404`.

- [x] **Step 5: Verify delete-message green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestQueue(DataPlaneLifecycle|DeleteMessageRequiresCurrentPopReceipt)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 97: Queue Storage Update Message Validation

**Docs and local reference verified:**
- Microsoft Learn `Update Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/update-message`
- Microsoft Learn `Put Message`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-message`
- Microsoft Learn `Get Messages`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-messages`

- [x] **Step 1: Verify update-message target**

Confirm `Update Message` requires a valid current pop receipt, accepts message text up to the documented 64 KiB encoded size, rejects oversized messages with `400 Bad Request`, returns a fresh `x-ms-popreceipt` and `x-ms-time-next-visible` on success, rejects visibility timeouts that would extend past the message expiration time, and treats stale or non-current receipts as message-not-found conditions.

- [x] **Step 2: Write failing update-message validation test**

Add a local Queue data-plane test that enqueues a short-lived message, dequeues it to obtain a pop receipt, verifies an oversized update returns `400` without mutating state, verifies a visibility timeout beyond the remaining TTL returns `400`, verifies a valid update rotates the receipt and changes the text, verifies an update using the stale receipt returns `404`, and peeks the queue to prove failed updates did not overwrite the committed message text.

- [x] **Step 3: Verify update-message red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestQueueUpdateMessageValidatesCurrentReceiptSizeAndExpiry -count=1
```

Observed red result before implementation: an oversized update returned `204 No Content` instead of `400 Bad Request`.

- [x] **Step 4: Implement update-message validation and current-receipt behavior**

Validate the update body against the existing 64 KiB Queue message limit before acquiring the queue lock, keep missing `popreceipt` or `visibilitytimeout` as `400 InvalidQueryParameterValue`, keep `visibilitytimeout` bounded to `0..604800`, return `404 MessageNotFound` when the message ID exists but the supplied pop receipt is stale or non-current, reject next-visible timestamps later than message expiration, and only mutate text or rotate `x-ms-popreceipt` after all validations succeed.

- [x] **Step 5: Verify update-message green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestQueueUpdateMessageValidatesCurrentReceiptSizeAndExpiry -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 98: Table Storage OData Batch Insert Changesets

**Docs and local reference verified:**
- Microsoft Learn `Performing entity group transactions`: `https://learn.microsoft.com/en-us/rest/api/storageservices/performing-entity-group-transactions`
- Microsoft Learn `Table Service REST API`: `https://learn.microsoft.com/en-us/rest/api/storageservices/table-service-rest-api`
- Microsoft Learn `Insert Entity`: `https://learn.microsoft.com/en-us/rest/api/storageservices/insert-entity`
- Microsoft Learn `Update Entity`: `https://learn.microsoft.com/en-us/rest/api/storageservices/update-entity2`

- [x] **Step 1: Verify batch transaction target**

Confirm Table Storage sends batch requests to `/$batch`, uses a multipart MIME batch body with a single changeset for insert, update, merge, and delete operations, limits changesets to 100 operations and 4 MiB payloads, returns an overall `202 Accepted` response when a well-formed batch is received, and reports individual operation HTTP statuses inside the multipart response.

- [x] **Step 2: Write failing batch insert test**

Add a local floci-style Table data-plane test that creates a table, posts a JSON changeset to `/{account}-table/$batch` with an embedded `POST` entity insert and `Prefer: return-no-content`, expects `202 Accepted` with a multipart batch response containing `HTTP/1.1 204 No Content` and an `ETag`, then reads the inserted entity through the normal Table entity route.

- [x] **Step 3: Verify batch red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneBatchInsertChangeset -count=1
```

Observed red result before implementation: `$batch` was treated as a normal entity collection route and returned `400 PropertiesNeedValue`.

- [x] **Step 4: Implement first-slice multipart batch handling**

Route `POST $batch` before normal table/entity dispatch, parse multipart batch and changeset boundaries, parse embedded `application/http` requests, delegate each embedded request to the existing Table entity handlers, cap payloads at 4 MiB and operation count at 100, return deterministic multipart batch responses with per-operation status and headers, and restore the account table snapshot when an embedded operation fails.

- [x] **Step 5: Verify batch green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneBatchInsertChangeset -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 99: Table Storage Batch Partition Validation

**Docs and local reference verified:**
- Microsoft Learn `Performing entity group transactions`: `https://learn.microsoft.com/en-us/rest/api/storageservices/performing-entity-group-transactions`
- Microsoft Learn `Insert Entity`: `https://learn.microsoft.com/en-us/rest/api/storageservices/insert-entity`

- [x] **Step 1: Verify validation target**

Confirm an entity group transaction must operate on entities with the same `PartitionKey`, and that an entity can appear only once in a transaction. Invalid changesets should be rejected before committing partial state.

- [x] **Step 2: Write failing mixed-partition test**

Add a local Table `$batch` test that creates a table, submits one changeset with two embedded insert operations using different partition keys, expects `400 Bad Request`, and verifies neither entity can be read afterward.

- [x] **Step 3: Verify validation red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneBatchRejectsMixedPartitionsAtomically -count=1
```

Observed red result before implementation: the mixed-partition batch returned `202 Accepted` and both embedded inserts succeeded with `204 No Content`.

- [x] **Step 4: Implement pre-execution batch validation**

Derive each embedded operation's table, `PartitionKey`, and `RowKey` from its OData URL or JSON body, reject changesets that mix partition keys, reject duplicate entity operations in the same changeset, and run this validation before executing any embedded operation.

- [x] **Step 5: Verify validation green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestTableDataPlaneBatchRejectsMixedPartitionsAtomically -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 100: Azure Files FileREST Share Lifecycle

**Docs and local reference verified:**
- Microsoft Learn `Azure Files REST API`: `https://learn.microsoft.com/en-us/rest/api/storageservices/file-service-rest-api`
- Microsoft Learn `Create Share`: `https://learn.microsoft.com/en-us/rest/api/storageservices/create-share`
- Microsoft Learn `List Shares`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-shares`
- Microsoft Learn `Get Share Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-share-properties`
- Microsoft Learn `Delete Share`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-share`

- [x] **Step 1: Verify FileREST target**

Confirm Azure Files data-plane requests use `*.file.core.windows.net`, local floci compatibility should use `/{account}-file`, share lifecycle requests use `?restype=share`, account-level share listing uses `?comp=list`, successful share creation returns `201 Created`, share property reads return `200 OK` with metadata/property headers and no body for `HEAD`, and delete returns `202 Accepted`.

- [x] **Step 2: Write failing routing and share lifecycle tests**

Add routing tests for hosted `acct.file.core.windows.net` and local `devstoreaccount1-file` paths resolving to `Microsoft.Storage/fileServices`, and add a Storage service test that creates a share with metadata/quota/tier/protocol headers, rejects duplicate creates, reads properties with `HEAD`, lists shares with `include=metadata`, and deletes the share.

- [x] **Step 3: Verify FileREST red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureStorageDataPlaneHosts -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileShareDataPlaneLifecycle -count=1
```

Observed red result before implementation: FileREST routes fell through to AWS S3 in routing, and storage requests returned the generic storage-account control-plane `404 NotFound`.

- [x] **Step 4: Implement FileREST first slice**

Register `Microsoft.Storage/fileServices` for the data-plane API version, detect hosted and local file endpoints, add file share state to `StorageService`, implement `handleFile`, create/list/get/head/delete share operations, return Azure Storage ETag/last-modified/version headers, preserve share metadata/properties, and emit Azure XML list responses.

- [x] **Step 5: Verify FileREST green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureStorageDataPlaneHosts -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileShareDataPlaneLifecycle -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 101: Azure Files Directory, File, Range, and Listing Operations

**Docs and local reference verified:**
- Microsoft Learn `Create Directory`: `https://learn.microsoft.com/en-us/rest/api/storageservices/create-directory`
- Microsoft Learn `Create File`: `https://learn.microsoft.com/en-us/rest/api/storageservices/create-file`
- Microsoft Learn `Put Range`: `https://learn.microsoft.com/en-us/rest/api/storageservices/put-range`
- Microsoft Learn `Get File`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-file`
- Microsoft Learn `List Directories and Files`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-directories-and-files`

- [x] **Step 1: Verify FileREST file target**

Confirm directories are created with `PUT ...?restype=directory`, file creation initializes a file with `x-ms-content-length` and `x-ms-type:file`, file content is written separately with `PUT ...?comp=range`, reads return content/metadata/properties and support byte ranges, and `GET ...?restype=directory&comp=list` returns only one directory level.

- [x] **Step 2: Write failing file operation test**

Add a local floci-style FileREST test that creates a share, creates `logs/`, creates `logs/today.txt` with file length/content type/metadata, writes `bytes=0-10` with `Put Range`, reads the whole file, reads `bytes=6-10` with `206 Partial Content`, and lists `logs` to verify the file entry and content length.

- [x] **Step 3: Verify file operation red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneDirectoryFileRangeAndList -count=1
```

Observed red result before implementation: directory creation returned `404 ShareNotFound` because FileREST only routed share-level operations.

- [x] **Step 4: Implement FileREST directory/file/range/list support**

Extend file share state with directory and file maps, route directory paths through `restype=directory`, route file creation through `x-ms-type:file`, implement range update/clear with `x-ms-range` or `Range`, implement full and partial file reads with metadata/content headers, and emit Azure XML one-level directory listings.

- [x] **Step 5: Verify file operation green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneDirectoryFileRangeAndList -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage ./pkg/routing -count=1
```

Expected: all commands pass.

## Task 102: Azure Files File Properties and Delete Semantics

**Docs and local reference verified:**
- Microsoft Learn `Get File Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-file-properties`
- Microsoft Learn `Delete File`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-file2`
- Microsoft Learn `Delete Directory`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-directory`

- [x] **Step 1: Verify FileREST property and delete target**

Confirm `HEAD` file requests return file properties and metadata without content, including `x-ms-type: File`; `DELETE` file requests immediately remove the file and return `202 Accepted`; and `DELETE ...?restype=directory` only removes empty directories, returning `409 DirectoryNotEmpty` while children remain.

- [x] **Step 2: Write failing property/delete test**

Add a local floci-style FileREST test that creates a share, directory, and file, writes file content, checks file properties through `HEAD`, verifies a non-empty directory delete conflict, deletes the file, confirms the file is no longer readable, and then deletes the now-empty directory.

- [x] **Step 3: Verify property/delete red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlanePropertiesAndDeletes -count=1
```

Observed red result before implementation: `HEAD` file properties returned status `200` with content length and metadata, but omitted `x-ms-type: File`.

- [x] **Step 4: Implement FileREST property and delete support**

Add Azure file and directory resource type headers, route file `DELETE` requests, route directory `DELETE` requests through `restype=directory`, remove files from share state, reject non-empty directory deletes with `409 DirectoryNotEmpty`, and delete empty directories with `202 Accepted`.

- [x] **Step 5: Verify property/delete green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlanePropertiesAndDeletes -count=1
```

Expected: the focused property/delete test passes.

## Task 103: Azure Files Directory Properties and Root Listings

**Docs and local reference verified:**
- Microsoft Learn `Get Directory Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-directory-properties`
- Microsoft Learn `List Directories and Files`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-directories-and-files`

- [x] **Step 1: Verify FileREST directory property and root-list targets**

Confirm `GET` and `HEAD ...?restype=directory` return directory properties and metadata without listing children, and confirm `GET /{share}?restype=directory&comp=list` lists only direct root children under the share.

- [x] **Step 2: Write failing directory properties test**

Add a local floci-style FileREST test that creates a share and metadata-bearing directory, then verifies `HEAD` and `GET` directory property requests return `200 OK`, no response body, directory attributes, and metadata headers.

- [x] **Step 3: Verify directory properties red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneDirectoryProperties -count=1
```

Observed red result before implementation: `HEAD ...?restype=directory` fell through to generic file lookup and returned `404 ResourceNotFound`.

- [x] **Step 4: Implement directory properties**

Route `GET` and `HEAD` directory requests without `comp=list` through a directory properties handler, look up the directory in share state, and return Azure directory headers without a body.

- [x] **Step 5: Verify directory properties green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneDirectoryProperties -count=1
```

Expected: the focused directory properties test passes.

- [x] **Step 6: Write failing root directory listing test**

Add a local floci-style FileREST test that creates a root directory, a root file, and a nested file, then verifies `GET /{share}?restype=directory&comp=list` returns only the direct root children.

- [x] **Step 7: Verify root listing red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneRootDirectoryList -count=1
```

Observed red result before implementation: share-level `?restype=directory&comp=list` returned `404 ShareNotFound` because only nested directory list routes existed.

- [x] **Step 8: Implement root directory listing**

Route share-level `?restype=directory&comp=list` requests to the existing directory listing logic with an empty directory path.

- [x] **Step 9: Verify root listing green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneRootDirectoryList -count=1
```

Expected: the focused root directory listing test passes.

## Task 104: Azure Files File and Directory Metadata Replacement

**Docs and local reference verified:**
- Microsoft Learn `Set File Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-file-metadata`
- Microsoft Learn `Set Directory Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-directory-metadata`

- [x] **Step 1: Verify FileREST metadata targets**

Confirm `PUT ?comp=metadata` on a file and `PUT ?restype=directory&comp=metadata` on a directory replace all existing user metadata, return `200 OK` with no response body, and rotate the resource ETag.

- [x] **Step 2: Write failing file metadata test**

Add a local floci-style FileREST test that creates a file with two metadata keys, sets metadata with only one replacement key, verifies a new ETag and no body, and confirms file properties expose only the replacement metadata.

- [x] **Step 3: Verify file metadata red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneSetFileMetadataReplacesExistingMetadata -count=1
```

Observed red result before implementation: `PUT ?comp=metadata` returned `404 ShareNotFound` because FileREST only routed range writes, file creation, reads, and deletes.

- [x] **Step 4: Implement file metadata replacement**

Route file `PUT ?comp=metadata` requests, replace the stored file metadata map from `x-ms-meta-*` headers, rotate ETag/last-modified, and return `200 OK` with storage headers.

- [x] **Step 5: Verify file metadata green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneSetFileMetadataReplacesExistingMetadata -count=1
```

Expected: the focused file metadata test passes.

- [x] **Step 6: Write failing directory metadata test**

Add a local floci-style FileREST test that creates a directory with two metadata keys, sets metadata with only one replacement key, verifies a new ETag and no body, and confirms directory properties expose only the replacement metadata.

- [x] **Step 7: Verify directory metadata red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneSetDirectoryMetadataReplacesExistingMetadata -count=1
```

Observed red result before implementation: `PUT ?restype=directory&comp=metadata` was treated as duplicate directory creation and returned `409 ResourceAlreadyExists`.

- [x] **Step 8: Implement directory metadata replacement**

Route directory metadata updates before create-directory, replace the stored directory metadata map from `x-ms-meta-*` headers, rotate ETag/last-modified, and return `200 OK` with storage headers.

- [x] **Step 9: Verify directory metadata green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneSetDirectoryMetadataReplacesExistingMetadata -count=1
```

Expected: the focused directory metadata test passes.

## Task 105: Azure Files File and Directory System Properties

**Docs and local reference verified:**
- Microsoft Learn `Set File Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-file-properties`
- Microsoft Learn `Set Directory Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-directory-properties`

- [x] **Step 1: Verify FileREST property targets**

Confirm `PUT ?comp=properties` on files updates HTTP properties as a replacement group, clears omitted HTTP properties when any property in that group is supplied, optionally resizes files through `x-ms-content-length`, and returns `200 OK` without a body. Confirm `PUT ?restype=directory&comp=properties` updates directory attributes and file-time headers independently from directory metadata.

- [x] **Step 2: Write failing file properties test**

Add a local floci-style FileREST test that creates and writes an 11-byte file, calls `Set File Properties` with `x-ms-content-length: 5` plus selected HTTP property headers, verifies a new ETag/no body response, confirms reads are truncated to five bytes, confirms selected HTTP property headers are returned, and confirms omitted HTTP properties are cleared.

- [x] **Step 3: Verify file properties red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneSetFilePropertiesUpdatesHeadersAndResizes -count=1
```

Observed red result before implementation: `PUT ?comp=properties` returned `404 ShareNotFound` because FileREST did not route file property updates.

- [x] **Step 4: Implement file properties**

Persist file HTTP property fields, route file `PUT ?comp=properties`, resize file content when `x-ms-content-length` is supplied, replace the documented HTTP property group when any group member is supplied, rotate ETag/last-modified, and surface persisted properties through file reads and property reads.

- [x] **Step 5: Write failing directory properties test**

Add a local floci-style FileREST test that creates a directory, calls `Set Directory Properties` with file attributes and explicit creation/last-write/change times, verifies a new ETag/no body response, and confirms later directory property reads include those system headers.

- [x] **Step 6: Verify directory properties red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneSetDirectoryPropertiesUpdatesSystemHeaders -count=1
```

Observed red result before implementation: `PUT ?restype=directory&comp=properties` was treated as duplicate directory creation and returned `409 ResourceAlreadyExists`.

- [x] **Step 7: Implement directory properties**

Persist directory attributes and file-time fields, route directory `PUT ?restype=directory&comp=properties` before create-directory, preserve existing values when property headers are omitted or `preserve`, rotate ETag/last-modified, and surface persisted properties through directory property reads.

- [x] **Step 8: Verify properties green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneSet(FilePropertiesUpdatesHeadersAndResizes|DirectoryPropertiesUpdatesSystemHeaders)' -count=1
```

Expected: both focused property tests pass.

## Task 106: Azure Files File and Directory Metadata Reads

**Docs and local reference verified:**
- Microsoft Learn `Get File Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-file-metadata`
- Microsoft Learn `Get Directory Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-directory-metadata`

- [x] **Step 1: Verify FileREST metadata read targets**

Confirm dedicated metadata read routes return user-defined metadata and `x-ms-type` headers without returning file content, content headers, or directory listing payloads. Use `200 OK` to match the Microsoft Learn sample responses for these read operations.

- [x] **Step 2: Write failing file metadata read test**

Add a local floci-style FileREST test that creates a metadata-bearing non-empty file, calls `GET ?comp=metadata`, verifies `200 OK`, no response body, `x-ms-type: File`, metadata headers, and absence of file content headers.

- [x] **Step 3: Verify file metadata read red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneGetFileMetadataReturnsMetadataHeadersOnly -count=1
```

Observed red result before implementation: `GET ?comp=metadata` returned full file content and content headers because it fell through to generic file reads.

- [x] **Step 4: Write failing directory metadata read test**

Add a local floci-style FileREST test that creates a metadata-bearing directory, calls `GET ?restype=directory&comp=metadata`, verifies `200 OK`, no response body, `x-ms-type: Directory`, metadata headers, and also verifies `HEAD` metadata behavior.

- [x] **Step 5: Verify directory metadata read red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneGetDirectoryMetadataReturnsMetadataHeadersOnly -count=1
```

Observed red result before implementation: `GET ?restype=directory&comp=metadata` fell through to generic file lookup and returned `404 ResourceNotFound`.

- [x] **Step 6: Implement metadata read routes**

Route file and directory `GET`/`HEAD ?comp=metadata` requests to dedicated metadata handlers, look up the stored resource, and return only storage headers, `x-ms-type`, and `x-ms-meta-*` headers with no response body.

- [x] **Step 7: Verify metadata reads green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneGet(FileMetadataReturnsMetadataHeadersOnly|DirectoryMetadataReturnsMetadataHeadersOnly)' -count=1
```

Expected: both focused metadata read tests pass.

## Task 107: Azure Files Range List Tracking

**Docs and local reference verified:**
- Microsoft Learn `List Ranges`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-ranges`

- [x] **Step 1: Verify FileREST range-list target**

Confirm `GET ?comp=rangelist` returns `200 OK`, XML `<Ranges>` with sorted non-overlapping valid ranges, `x-ms-content-length`, ETag, last-modified, and no file content. Confirm `x-ms-range`/`Range` filters the returned valid ranges, with `x-ms-range` taking precedence.

- [x] **Step 2: Write failing range-list test**

Add a local floci-style FileREST test that creates a 1024-byte file, writes ranges `0-127` and `512-767`, clears `64-127`, verifies List Ranges returns `0-63` and `512-767`, and verifies `x-ms-range: bytes=600-900` returns only the clipped `600-767` range.

- [x] **Step 3: Verify range-list red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneListRangesTracksUpdatesClearsAndFilters -count=1
```

Observed red result before implementation: `GET ?comp=rangelist` fell through to generic file reads and returned the full file body with `application/octet-stream` instead of Azure range-list XML.

- [x] **Step 4: Implement range tracking and List Ranges**

Persist valid file ranges separately from file content, merge ranges on update writes, split/remove ranges on clear writes, trim tracked ranges when files are truncated, route `GET ?comp=rangelist`, filter ranges by `x-ms-range` or `Range`, and return Azure XML plus file length/storage headers.

- [x] **Step 5: Verify range-list green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneListRangesTracksUpdatesClearsAndFilters -count=1
```

Expected: the focused range-list test passes.

## Task 108: Azure Files File and Directory Rename

**Docs and local reference verified:**
- Microsoft Learn `Rename File`: `https://learn.microsoft.com/en-us/rest/api/storageservices/rename-file`
- Microsoft Learn `Rename Directory`: `https://learn.microsoft.com/en-us/rest/api/storageservices/rename-directory`

- [x] **Step 1: Verify FileREST rename targets**

Confirm file rename uses `PUT ?comp=rename` with `x-ms-file-rename-source`, returns `200 OK` without a response body, preserves file content/properties/metadata/tracked ranges unless headers override metadata/properties, and removes the source file. Confirm directory rename uses `PUT ?restype=directory&comp=rename`, moves the directory subtree, preserves root directory metadata/properties unless overridden, and removes the source paths.

- [x] **Step 2: Write failing file rename test**

Add a local floci-style FileREST test that creates source and destination parent directories, creates and writes a metadata-bearing source file, renames it to the destination parent, verifies `200 OK` and response headers, confirms the destination file content/metadata/properties/ranges are preserved, and confirms the source file returns `404`.

- [x] **Step 3: Verify file rename red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneRenameFilePreservesContentMetadataAndRanges -count=1
```

Observed red result before implementation: `PUT ?comp=rename` returned `404 ShareNotFound` because FileREST did not route file rename requests.

- [x] **Step 4: Implement file rename**

Route file `PUT ?comp=rename`, parse same-share `x-ms-file-rename-source`, validate destination parent directories and destination conflicts, move the stored file object to the destination path, preserve content/properties/metadata/ranges, optionally replace metadata/content type from headers, rotate ETag/last-modified, and delete the source path.

- [x] **Step 5: Write failing directory rename test**

Add a local floci-style FileREST test that creates a metadata-bearing source directory with a child file, renames the directory, verifies `200 OK`, confirms the destination directory metadata is preserved, confirms the child file moved with content/metadata intact, and confirms the source directory and child file return `404`.

- [x] **Step 6: Verify directory rename red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneRenameDirectoryMovesChildPaths -count=1
```

Observed red result before implementation: `PUT ?restype=directory&comp=rename` was treated as create-directory and returned `201 Created`.

- [x] **Step 7: Implement directory rename**

Route directory `PUT ?restype=directory&comp=rename` before create-directory, parse same-share source headers, validate destination parent and subtree conflicts, move the source directory plus descendant file and directory paths under the destination, preserve child data, optionally replace root metadata/properties from headers, rotate the root directory ETag/last-modified, and remove source paths.

- [x] **Step 8: Verify rename green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneRename(FilePreservesContentMetadataAndRanges|DirectoryMovesChildPaths)' -count=1
```

Expected: both focused rename tests pass.

## Task 109: Azure Files File Lease Acquire, State, and Enforcement

**Docs and local reference verified:**
- Microsoft Learn `Lease File`: `https://learn.microsoft.com/en-us/rest/api/storageservices/lease-file`

- [x] **Step 1: Verify FileREST lease targets**

Confirm `PUT ?comp=lease` supports Azure Files lease actions for file write/delete locks, returns `201 Created` for acquire, `200 OK` for renew/change/release, `202 Accepted` for break, preserves ETag and `Last-Modified` during lease operations, and requires active lease IDs for mutating file operations.

- [x] **Step 2: Write failing acquire/release enforcement test**

Add a local floci-style FileREST test that creates a share, directory, and file, acquires a proposed infinite lease, verifies ETag and `Last-Modified` are unchanged, verifies file property lease headers, rejects `Put Range` and `Delete File` without the lease ID, accepts `Put Range` with the lease ID, releases the lease, and then allows delete.

- [x] **Step 3: Verify acquire/release red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneLeaseFileAcquireReleaseAndWriteEnforcement -count=1
```

Observed red result before implementation: `PUT ?comp=lease` returned `404 ShareNotFound` because FileREST did not route file lease requests.

- [x] **Step 4: Implement file lease actions and enforcement**

Persist per-file lease ID/state, route file `PUT ?comp=lease`, implement acquire/renew/change/release/immediate break without changing file ETag or `Last-Modified`, expose `x-ms-lease-status` and `x-ms-lease-state` on file property/read headers, and enforce active lease IDs for create-file overwrites, range writes, property updates, metadata updates, renames, and deletes.

- [x] **Step 5: Write failing available/broken write outcome test**

Add a local FileREST test for the documented lease-state outcome table: writes that carry a lease ID against an available file fail with `412`, writes that carry the old lease ID against a broken lease fail with `412`, and a successful no-lease `Put Range` on a broken lease transitions the lease state back to available.

- [x] **Step 6: Verify available/broken write red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneLeaseFileAvailableAndBrokenWriteOutcomes -count=1
```

Observed red result before implementation: writes carrying a lease ID against an available file returned `409 LeaseIdMismatchWithFileOperation` instead of the documented `412`.

- [x] **Step 7: Implement available/broken write outcomes**

Return `412 LeaseNotPresentWithFileOperation` when a write supplies a lease ID but the file is available or broken, keep active mismatched lease IDs as `409`, and make successful no-lease `Put Range` operations on broken leases transition the file lease state to available.

- [x] **Step 8: Write failing broken release test**

Add a local FileREST test that acquires a lease, breaks it, releases it with the matching original lease ID, and verifies the file returns to unlocked/available.

- [x] **Step 9: Verify broken release red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneLeaseFileReleaseBrokenLeaseWithMatchingID -count=1
```

Observed red result before implementation: release after break returned `409 LeaseIdMismatchWithLeaseOperation` because the break path cleared the lease ID too early.

- [x] **Step 10: Implement broken release**

Retain the lease ID while a file lease is broken, clear it only on release, reacquire, or a successful no-lease `Put Range`, and keep broken leases reported as unlocked with `x-ms-lease-state: broken`.

- [x] **Step 11: Verify file lease green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneLeaseFile(AcquireReleaseAndWriteEnforcement|AvailableAndBrokenWriteOutcomes|ReleaseBrokenLeaseWithMatchingID)' -count=1
```

Expected: all focused file lease tests pass.

## Task 110: Azure Files Copy File

**Docs and local reference verified:**
- Microsoft Learn `Copy File`: `https://learn.microsoft.com/en-us/rest/api/storageservices/copy-file`

- [x] **Step 1: Verify FileREST copy target**

Confirm `Copy File` uses `PUT` with required `x-ms-copy-source`, returns `202 Accepted`, returns `x-ms-copy-id` and `x-ms-copy-status`, copies the full source file/blob to the destination file, copies source metadata when destination metadata headers are omitted, applies destination `x-ms-meta-*` headers when present, overwrites destination files, and requires a matching destination lease ID when the destination file has an active lease.

- [x] **Step 2: Write failing content/properties/metadata copy test**

Add a local floci-style FileREST test that creates source and destination-parent directories, creates a metadata-bearing source file, sets file HTTP properties, writes source content, copies it to a new destination with `x-ms-copy-source`, verifies `202 Accepted`, copy ID/status headers, copied content, copied metadata, copied content length, and copied HTTP properties.

- [x] **Step 3: Write failing destination lease enforcement test**

Add a local FileREST test that creates a source file and an existing leased destination file, verifies copy without the destination lease ID fails with `412 Precondition Failed`, then verifies copy with the matching lease ID succeeds and preserves the active destination lease while replacing destination content length.

- [x] **Step 4: Verify copy red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneCopyFile(CopiesContentPropertiesAndMetadata|RequiresDestinationLeaseID)' -count=1
```

Observed red result before implementation: both tests fell through to missing FileREST routing; copy returned `404 ShareNotFound` instead of `202 Accepted` or destination-lease `412`.

- [x] **Step 5: Implement synchronous local copy**

Route file `PUT` requests with `x-ms-copy-source`, parse hosted and floci-style local source URLs for file and blob sources, copy source content/properties/metadata/ranges into the destination, overwrite existing destination files, preserve destination lease state when the matching lease ID is supplied, return synchronous success copy headers, and expose copy status headers on later file reads/properties.

- [x] **Step 6: Verify copy green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneCopyFile(CopiesContentPropertiesAndMetadata|RequiresDestinationLeaseID)' -count=1
```

Expected: both focused copy tests pass.

## Task 111: Azure Files Share Snapshots

**Docs and local reference verified:**
- Microsoft Learn `Snapshot Share`: `https://learn.microsoft.com/en-us/rest/api/storageservices/snapshot-share`
- Microsoft Learn `List Shares`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-shares`
- Microsoft Learn `Delete Share`: `https://learn.microsoft.com/en-us/rest/api/storageservices/delete-share`

- [x] **Step 1: Verify FileREST share snapshot targets**

Confirm `PUT ?restype=share&comp=snapshot` creates read-only point-in-time snapshots, returns `201 Created` with `x-ms-snapshot`, snapshots can be addressed through `sharesnapshot=<DateTime>`, `List Shares` includes snapshots only when `include=snapshots` is supplied, and base-share deletes fail with `409` when snapshots exist unless `x-ms-delete-snapshots: include` is supplied.

- [x] **Step 2: Write failing snapshot create/read/list test**

Add a local floci-style FileREST test that creates a share, directory, and file, writes initial file content, snapshots the share, mutates the base file, verifies base reads return the later content, verifies `GET` with `sharesnapshot=<DateTime>` returns the original content and `x-ms-snapshot` header, verifies writes against the snapshot are rejected, and verifies account `List Shares` with `include=metadata,snapshots` includes the snapshot entry.

- [x] **Step 3: Write failing snapshot delete test**

Add a local FileREST test that creates a share snapshot, verifies deleting the base share without `x-ms-delete-snapshots` returns `409`, deletes an individual snapshot through `sharesnapshot=<DateTime>`, creates another snapshot, and verifies deleting the base share with `x-ms-delete-snapshots: include` succeeds.

- [x] **Step 4: Verify share snapshot red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(SnapshotSharePreservesPointInTimeFilesAndListsSnapshots|DeleteShareHonorsSnapshotOptions)' -count=1
```

Observed red result before implementation: `PUT ?restype=share&comp=snapshot` was treated as duplicate share creation and returned `409 ShareAlreadyExists`; deleting a base share with snapshots returned `202 Accepted` instead of the documented `409`.

- [x] **Step 5: Implement share snapshots**

Persist per-share snapshot maps, clone share metadata/directories/files/ranges/properties at snapshot time, return `x-ms-snapshot`, resolve snapshot reads through `sharesnapshot=<DateTime>`, reject mutating operations against snapshot resources, list snapshots when requested, reject base share deletes while snapshots exist unless `x-ms-delete-snapshots` includes them, and support individual snapshot deletion.

- [x] **Step 6: Verify share snapshot green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(SnapshotSharePreservesPointInTimeFilesAndListsSnapshots|DeleteShareHonorsSnapshotOptions)' -count=1
```

Expected: both focused share snapshot tests pass.

## Task 112: Azure Files Share Metadata and Properties

**Docs and local reference verified:**
- Microsoft Learn `Set Share Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-share-metadata`
- Microsoft Learn `Get Share Metadata`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-share-metadata`
- Microsoft Learn `Set Share Properties`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-share-properties`

- [x] **Step 1: Verify share metadata/property targets**

Confirm share metadata uses `PUT/GET/HEAD ?restype=share&comp=metadata`, `Set Share Metadata` replaces all existing metadata and updates ETag/last-modified, `Get Share Metadata` returns only metadata plus storage headers, snapshot metadata can be read but not modified, and share properties use `PUT ?restype=share&comp=properties` to update quota/tier/root-squash style properties while rejecting snapshot property updates.

- [x] **Step 2: Write failing share metadata test**

Add a local floci-style FileREST test that creates a metadata-bearing share, snapshots it, replaces base share metadata with one key, verifies a new ETag, verifies `GET ?comp=metadata` returns only replacement metadata with no body, verifies snapshot metadata still returns the point-in-time metadata plus `x-ms-snapshot`, and verifies setting metadata on the snapshot fails with `400`.

- [x] **Step 3: Write failing share properties test**

Add a local FileREST test that creates a share with initial quota/tier, snapshots it, calls `PUT ?comp=properties` with a new quota, access tier, root squash, and snapshot virtual-directory-access setting, verifies `200 OK` and ETag rotation, confirms later `HEAD ?restype=share` returns those property headers, and verifies setting properties on the snapshot fails with `400`.

- [x] **Step 4: Verify metadata/properties red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(SetAndGetShareMetadataReplacesExistingMetadata|SetSharePropertiesUpdatesShareHeaders)' -count=1
```

Observed red result before implementation: both `PUT ?restype=share&comp=metadata` and `PUT ?restype=share&comp=properties` were treated as duplicate share creation and returned `409 ShareAlreadyExists`.

- [x] **Step 5: Implement share metadata/properties**

Route share metadata and property operations before create-share, replace metadata maps from `x-ms-meta-*`, rotate share ETags/last-modified timestamps, return metadata-only headers for dedicated metadata reads, update quota/access-tier/root-squash/snapshot virtual-directory-access properties, expose those properties through share headers/listings, and reject share snapshot mutations.

- [x] **Step 6: Verify metadata/properties green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(SetAndGetShareMetadataReplacesExistingMetadata|SetSharePropertiesUpdatesShareHeaders)' -count=1
```

Expected: both focused share metadata/property tests pass.

## Task 113: Azure Files Share Lease

**Docs and local reference verified:**
- Microsoft Learn `Lease Share`: `https://learn.microsoft.com/en-us/rest/api/storageservices/lease-share`

- [x] **Step 1: Verify Lease Share targets**

Confirm `PUT ?restype=share&comp=lease` supports file share and share snapshot leases in API version `2020-02-10` and later, accepts acquire, renew, change, release, and break actions, returns `201 Created` for acquire, `200 OK` for renew/change/release, `202 Accepted` for break, returns `x-ms-lease-id` or `x-ms-lease-time` where documented, does not update share `Last-Modified`, requires the active lease ID for delete and set-share operations, and keeps snapshot lease state separate from base-share lease state.

- [x] **Step 2: Write failing base share lease enforcement tests**

Add local floci-style FileREST tests that create a share, acquire a proposed infinite share lease, verify lease ID plus unchanged ETag/last-modified headers, verify `HEAD ?restype=share` reports locked/leased, reject share metadata/property updates and base delete without the active lease ID, allow metadata/property updates with the lease ID, release the lease, and then allow delete. Add a second test for renew, change, old-lease `409`, immediate break, broken lease headers, and reacquire after break.

- [x] **Step 3: Verify base share lease red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneLeaseShare(AcquireReleaseAndMutationEnforcement|RenewChangeAndBreak)' -count=1
```

Observed red result before implementation: share lease `PUT` requests were treated as duplicate share creation and returned `409 ShareAlreadyExists` with no lease headers.

- [x] **Step 4: Implement base share lease actions and enforcement**

Persist per-share lease ID/state, route share `PUT ?restype=share&comp=lease` before create-share, implement acquire/renew/change/release/immediate break without changing share ETag or last-modified values, project lease status/state through share property headers, enforce active lease IDs for share metadata, share properties, share snapshot creation, and base share deletion, and return documented `412`/`409` lease operation failures for missing, absent, or mismatched lease IDs.

- [x] **Step 5: Write failing share snapshot lease test**

Add a local FileREST test that creates a share snapshot, acquires a proposed lease against `sharesnapshot=<DateTime>`, verifies lease ID plus unchanged snapshot ETag/last-modified headers, verifies snapshot property reads report locked/leased with `x-ms-snapshot`, verifies the base share remains unlocked/available, rejects deleting the leased snapshot without the lease ID, and allows deleting it with the matching lease ID.

- [x] **Step 6: Verify share snapshot lease red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneLeaseShareSnapshotDeleteEnforcement -count=1
```

Observed red result before implementation: share snapshot lease acquire returned `400 InvalidQueryParameterValue` because snapshot mutation guards rejected `PUT ?comp=lease&sharesnapshot=...`.

- [x] **Step 7: Implement share snapshot leases**

Allow `PUT ?restype=share&comp=lease&sharesnapshot=<DateTime>` through the snapshot guard, reuse the share lease state machine against the stored snapshot share copy, persist snapshot lease changes independently from the base share, expose snapshot lease headers on snapshot share property reads, and enforce snapshot lease IDs when deleting individual snapshots.

- [x] **Step 8: Verify Lease Share green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlaneLeaseShare(AcquireReleaseAndMutationEnforcement|RenewChangeAndBreak|SnapshotDeleteEnforcement)' -count=1
```

Expected: all focused Lease Share tests pass.

## Task 114: Azure Files Get Share Stats

**Docs and local reference verified:**
- Microsoft Learn `Get Share Stats`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-share-stats`

- [x] **Step 1: Verify Get Share Stats target**

Confirm `GET ?restype=share&comp=stats` returns `200 OK` with Azure XML `<ShareStats><ShareUsageBytes>...</ShareUsageBytes></ShareStats>`, returns the share ETag and last-modified headers, does not update share last-modified when files are created or resized below the share, accepts stats reads on leased shares without a lease ID, validates a supplied `x-ms-lease-id` against the active lease with `412` on mismatch, and rejects share snapshot stats with `400 InvalidQueryParameterValue`.

- [x] **Step 2: Write failing share stats test**

Add a local floci-style FileREST test that creates a share, records share headers, creates files under the share, calls `GET ?restype=share&comp=stats`, verifies XML `ShareUsageBytes`, verifies share ETag/last-modified match the pre-file share headers, acquires a share lease, verifies stats without a lease ID still succeeds, verifies a wrong lease ID fails with `412`, verifies the matching lease ID succeeds, snapshots the leased share with the matching lease ID, and verifies snapshot stats fail with `400`.

- [x] **Step 3: Verify share stats red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneGetShareStatsReturnsUsageBytesAndRejectsSnapshots -count=1
```

Observed red result before implementation: `GET ?restype=share&comp=stats` fell through to share properties, returned `200 OK` with no XML content type and an empty body.

- [x] **Step 4: Implement share stats**

Route `GET ?restype=share&comp=stats` before normal share property reads, reject snapshot stats requests, calculate local `ShareUsageBytes` from stored file content lengths, return Azure XML with the share storage headers, and validate an optional `x-ms-lease-id` against active share leases with `412 LeaseIdMismatchWithShareOperation` when it does not match.

- [x] **Step 5: Verify share stats green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run TestFileDataPlaneGetShareStatsReturnsUsageBytesAndRejectsSnapshots -count=1
```

Expected: focused share stats test passes.

## Task 115: Azure Files Share ACL

**Docs and local reference verified:**
- Microsoft Learn `Get Share ACL`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-share-acl`
- Microsoft Learn `Set Share ACL`: `https://learn.microsoft.com/en-us/rest/api/storageservices/set-share-acl`

- [x] **Step 1: Verify share ACL targets**

Confirm `GET/HEAD ?restype=share&comp=acl` returns stored access policies as Azure `SignedIdentifiers` XML for `GET` and headers only for `HEAD`, `PUT ?restype=share&comp=acl` replaces the full stored policy set, policy IDs are capped at 64 characters, at most five access policies are allowed, ACL operations update share ETags/last-modified headers on successful `PUT`, optional lease ID validation applies to `GET`, active lease IDs are required for `PUT` when a share is leased, a supplied lease ID on an available share fails with `412`, invalid ACL updates preserve existing policies, and share snapshot ACL reads/writes return `400 InvalidQueryParameterValue`.

- [x] **Step 2: Write failing ACL replacement test**

Add a local floci-style FileREST test that creates a share, sets two signed identifiers through `PUT ?comp=acl`, verifies `200 OK` and ETag rotation, retrieves the policies through `GET ?comp=acl`, verifies policy XML and current share headers, verifies `HEAD ?comp=acl` returns headers without a body, replaces the ACL with one policy, and verifies omitted policies are removed.

- [x] **Step 3: Write failing lease, limit, and snapshot validation test**

Add a local FileREST test that rejects setting ACL with a lease ID on an available share, acquires a share lease, rejects setting ACL without the active lease ID, accepts setting ACL with the matching lease ID, rejects `GET ?comp=acl` with a mismatched lease ID, rejects more than five policies while preserving the previous ACL, and rejects `GET`/`PUT ?comp=acl&sharesnapshot=<DateTime>`.

- [x] **Step 4: Verify share ACL red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(SetAndGetShareACLReplacesPolicies|ShareACLLeaseLimitAndSnapshotValidation)' -count=1
```

Observed red result before implementation: `PUT ?restype=share&comp=acl` fell through to duplicate share creation and returned `409 ShareAlreadyExists`.

- [x] **Step 5: Implement share ACL**

Route share `comp=acl` before create/properties handling, parse Azure `SignedIdentifiers` XML into ordered share access policies, replace the full policy set on successful `PUT`, enforce the five-policy and 64-character ID limits before mutation, rotate share ETags and last-modified timestamps, emit stored policies through Azure XML on `GET`, return storage headers only on `HEAD`, enforce lease checks for ACL reads/writes, and reject snapshot ACL requests.

- [x] **Step 6: Verify share ACL green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(SetAndGetShareACLReplacesPolicies|ShareACLLeaseLimitAndSnapshotValidation)' -count=1
```

Expected: both focused share ACL tests pass.

## Task 116: Azure Files List and Force Close Handles

**Docs and local reference verified:**
- Microsoft Learn `List Handles`: `https://learn.microsoft.com/en-us/rest/api/storageservices/list-handles`
- Microsoft Learn `Force Close Handles`: `https://learn.microsoft.com/en-us/rest/api/storageservices/force-close-handles`

- [x] **Step 1: Verify handle-operation targets**

Confirm `GET ?comp=listhandles` returns `200 OK` Azure XML `EnumerationResults` with a `HandleList`, supports files, directories, optional recursive directory enumeration, positive `maxresults`, continuation markers, and share snapshots. Confirm `PUT ?comp=forceclosehandles` requires `x-ms-handle-id`, supports wildcard `*`, returns `200 OK` with `x-ms-number-of-handles-closed` and `x-ms-number-of-handles-failed`, and rejects `x-ms-recursive` on file targets with `400 Bad Request`.

- [x] **Step 2: Write failing list handles test**

Add a local floci-style FileREST test that creates a share, directory, and file, verifies file `GET ?comp=listhandles` returns Azure XML with an empty `HandleList` and storage version headers, verifies directory list handles echoes `marker` and `maxresults`, rejects nonpositive `maxresults`, snapshots the share, and verifies list handles against the snapshot file succeeds and echoes `ShareSnapshot`.

- [x] **Step 3: Write failing force-close handles test**

Add a local FileREST test that creates a share, directory, and file, rejects `PUT ?comp=forceclosehandles` without `x-ms-handle-id`, rejects recursive force close on a file target, verifies wildcard force close on a file returns zero closed/failed handle counts and no body, and verifies recursive wildcard force close on a directory also returns zero closed/failed counts with Azure version headers.

- [x] **Step 4: Verify handle-operation red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(ListHandlesReturnsAzureXMLForFilesDirectoriesAndSnapshots|ForceCloseHandlesValidatesHeadersAndReturnsCounts)' -count=1
```

Observed red result before implementation: `GET ?comp=listhandles` fell through to normal file content reads and returned `application/octet-stream`, while `PUT ?comp=forceclosehandles` returned the fallback `404 ShareNotFound`.

- [x] **Step 5: Implement first-slice handle routes**

Route file, directory, and root-directory `comp=listhandles` and `comp=forceclosehandles`, validate target existence and positive `maxresults`, return Azure XML `EnumerationResults` with an empty `HandleList` until CloudMock has an SMB/open-handle source, echo marker/maxresults/share-snapshot fields, require `x-ms-handle-id`, reject recursive force close on file targets, and return documented zero-count force-close headers.

- [x] **Step 6: Verify handle-operation green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(ListHandlesReturnsAzureXMLForFilesDirectoriesAndSnapshots|ForceCloseHandlesValidatesHeadersAndReturnsCounts)' -count=1
```

Expected: both focused handle-operation tests pass.

## Task 117: Azure Files Create and Get Permission

**Docs and local reference verified:**
- Microsoft Learn `Create Permission`: `https://learn.microsoft.com/en-us/rest/api/storageservices/create-permission`
- Microsoft Learn `Get Permission`: `https://learn.microsoft.com/en-us/rest/api/storageservices/get-permission`

- [x] **Step 1: Verify file permission targets**

Confirm `PUT ?restype=share&comp=filepermission` creates a share-level security descriptor from JSON `permission`, defaults omitted `format` to `sddl`, supports explicit `format: sddl` and version `2024-11-04` `format: binary`, returns `201 Created` with `x-ms-file-permission-key`, and no body. Confirm `GET ?restype=share&comp=filepermission` requires `x-ms-file-permission-key`, supports optional `x-ms-file-permission-format`, returns legacy `{permission}` JSON for pre-`2024-11-04` SDDL reads, returns `{format, permission}` for `2024-11-04` or binary descriptors, and returns `404` for unknown permission keys.

- [x] **Step 2: Write failing SDDL and binary round-trip test**

Add a local floci-style FileREST test that creates a share, creates an SDDL permission, verifies `201 Created`, `x-ms-file-permission-key`, and no body, retrieves it by key as legacy JSON without `format`, creates a binary permission with `format: binary` under `2024-11-04`, verifies it receives a distinct permission key, and retrieves it with `x-ms-file-permission-format: binary` as JSON containing both `format` and `permission`.

- [x] **Step 3: Write failing validation test**

Add a local FileREST test that rejects create-permission requests without `permission`, rejects unsupported `format` values, rejects get-permission requests without `x-ms-file-permission-key`, and returns `404 PermissionNotFound` for unknown keys.

- [x] **Step 4: Verify file permission red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(CreateAndGetFilePermissionStoresSDDLAndBinaryFormats|FilePermissionValidationAndMissingKeys)' -count=1
```

Observed red result before implementation: `PUT ?restype=share&comp=filepermission` fell through to duplicate share creation and returned `409 ShareAlreadyExists`.

- [x] **Step 5: Implement file permission storage and retrieval**

Route share `comp=filepermission` before create/properties handling, parse JSON permission bodies, default omitted `format` to `sddl`, validate supported `sddl` and base64 `binary` formats, store descriptors under deterministic SHA-256-derived permission keys on the share, return created keys through `x-ms-file-permission-key`, retrieve descriptors by key, honor pre-`2024-11-04` and `2024-11-04` response shapes, validate requested formats, and return Azure-style missing-key and unknown-key errors.

- [x] **Step 6: Verify file permission green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./services/azure/storage -run 'TestFileDataPlane(CreateAndGetFilePermissionStoresSDDLAndBinaryFormats|FilePermissionValidationAndMissingKeys)' -count=1
```

Expected: both focused file permission tests pass.

## Task 118: Azure Container Instances Container Groups

**Docs and local reference verified:**
- Microsoft Learn `Container Groups - Create Or Update`: `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/create-or-update?view=rest-container-instances-2025-09-01`
- Microsoft Learn `Container Groups - Get`: `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/get?view=rest-container-instances-2025-09-01`
- Microsoft Learn `Container Groups - List By Resource Group`: `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/list-by-resource-group?view=rest-container-instances-2025-09-01`
- Microsoft Learn `Container Groups - List`: `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/list?view=rest-container-instances-2025-09-01`
- Microsoft Learn `Container Groups - Delete`: `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/delete?view=rest-container-instances-2025-09-01`
- Microsoft Learn `Container Groups - Restart`: `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/restart?view=rest-container-instances-2025-09-01`

- [x] **Step 1: Verify container group targets**

Confirm `Microsoft.ContainerInstance/containerGroups` routes for `2025-09-01` support create/update, get, resource-group list, subscription list, delete, and restart. Confirm create/update accepts ARM container group request shape with `properties.containers`, `imageRegistryCredentials`, `ipAddress`, `osType`, `restartPolicy`, and volumes, returns `201 Created` for new groups and `200 OK` for updates, returns Azure `value` list envelopes, returns `200 OK` plus deleted resource body for delete, and exposes provider manifest metadata and ARM template provisioning.

- [x] **Step 2: Write failing routing and service tests**

Add routing tests for `CreateOrUpdateContainerGroup`, `GetContainerGroup`, `ListContainerGroups`, `DeleteContainerGroup`, and `RestartContainerGroup`. Add service tests for create/update/get/list/delete/restart, property preservation, deterministic provisioning and instance-view fields, missing container validation, ARM template provisioning, and versioned service keys.

- [x] **Step 3: Verify container group red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerInstanceContainerGroupActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerinstance -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed red results before implementation: routing fell back to generic ARM actions such as `CreateOrUpdate`, `Get`, `List`, and `Delete`; the new `services/azure/containerinstance` package failed to compile because `New` was undefined; and the provider manifest returned `404 ProviderNotFound` for `Microsoft.ContainerInstance`.

- [x] **Step 4: Implement container group first slice**

Add `services/azure/containerinstance`, route `Microsoft.ContainerInstance/containerGroups`, store resources in memory by subscription/resource group/name, preserve request metadata and properties, inject deterministic `Succeeded` provisioning state and running group/container instance views, support create/update/get/list/delete/restart, validate missing `properties.containers`, expose `ServiceKeys()` for `2025-09-01`, wire ARM template provisioning, add provider manifest metadata, and register the service in the gateway bootstrap.

- [x] **Step 5: Verify container group green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerInstanceContainerGroupActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerinstance -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway -count=1
```

Expected: focused routing, Container Instance, Resources provider manifest, and gateway compile tests pass.

## Task 119: Azure Container Apps and Managed Environments

**Docs and local reference verified:**
- Microsoft Learn `Azure Container Apps Resource Manager REST APIs`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/operation-groups`
- Microsoft Learn `Container Apps - Create Or Update`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/create-or-update?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Container Apps - Get`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/get?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Container Apps - List By Resource Group`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/list-by-resource-group?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Container Apps - Delete`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/delete?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Managed Environments - Create Or Update`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/managed-environments/create-or-update?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Managed Environments - Get`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/managed-environments/get?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Managed Environments - List By Resource Group`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/managed-environments/list-by-resource-group?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Managed Environments - Delete`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/managed-environments/delete?view=rest-resource-manager-containerapps-2025-07-01`

- [x] **Step 1: Verify Container Apps targets**

Confirm `Microsoft.App/managedEnvironments` and `Microsoft.App/containerApps` routes for `2025-07-01` support create/update, get, resource-group list, and delete. Confirm container app create/update preserves managed environment references, configuration, ingress, secrets, templates, containers, scale metadata, identity, kind, and tags; injects deterministic `Succeeded` provisioning state, `Running` status, latest revision names, and latest revision FQDNs; supports ARM template provisioning; and exposes provider manifest metadata.

- [x] **Step 2: Write failing routing, service, and manifest tests**

Add routing tests for `CreateOrUpdateManagedEnvironment`, `ListManagedEnvironments`, `DeleteManagedEnvironment`, `CreateOrUpdateContainerApp`, `GetContainerApp`, `ListContainerApps`, and `DeleteContainerApp`. Add service tests for managed environment lifecycle, container app create/update/list/delete, property preservation, deterministic defaults, missing template validation, ARM template provisioning, and versioned service keys. Extend the `Microsoft.Resources` provider manifest test for `Microsoft.App`.

- [x] **Step 3: Verify Container Apps red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerAppsActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed red results before implementation: routing fell back to generic ARM actions, the new `services/azure/containerapps` package failed to compile because `New` was undefined, and the provider manifest returned `404 ProviderNotFound` for `Microsoft.App`.

- [x] **Step 4: Implement Container Apps first slice**

Add `services/azure/containerapps`, route `Microsoft.App/containerApps` and `Microsoft.App/managedEnvironments`, store resources in memory by subscription/resource group/name, preserve request metadata and properties, inject deterministic managed-environment domains/static IPs, deterministic app provisioning/running/revision projections, support create/update/get/list/delete, validate missing `properties.template.containers`, expose `ServiceKeys()` for `2025-07-01`, wire ARM template provisioning, add provider manifest metadata, and register the service in the gateway bootstrap.

- [x] **Step 5: Verify Container Apps green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerAppsActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway -count=1
```

Expected: focused routing, Container Apps, Resources provider manifest, and gateway compile tests pass.

## Task 120: Azure Container Registry Docker Registry Data Plane

**Docs verified:**
- Microsoft Learn `Azure Container Registry REST API reference`: `https://learn.microsoft.com/en-us/rest/api/containerregistry/`
- Microsoft Learn `View container registry repositories`: `https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repositories`
- Microsoft Learn `Authenticate with Azure Container Registry`: `https://learn.microsoft.com/en-us/azure/container-registry/container-registry-authentication`
- Microsoft Learn `Push your first image to your Azure container registry by using the Docker CLI`: `https://learn.microsoft.com/en-us/azure/container-registry/container-registry-get-started-docker-cli`

- [x] **Step 1: Verify ACR data-plane target surface**

Confirm ACR is a managed Docker registry for private images and artifacts, exposes repositories and tags after images are pushed, supports Docker CLI login/push/pull flows through `*.azurecr.io`, and must remain routed separately from ARM `Microsoft.ContainerRegistry/registries` management APIs.

- [x] **Step 2: Write failing routing and service tests**

Add routing tests for hosted `https://{registry}.azurecr.io/v2/...` and local floci-style `/{registry}-acr/v2/...` requests, covering registry ping, catalog, tag list, manifest put/get/delete, and default-version lookup with no ARM `api-version`. Add service tests for registry ping headers, empty/non-empty catalogs, manifest push digest projection, tag and digest reads, HEAD manifest reads, tag deletion, local route compatibility, missing registry, and missing manifest responses.

- [x] **Step 3: Verify ACR data-plane red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryDataPlaneActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run 'TestRegistryDataPlane(ManifestTagsAndCatalog|LocalRouteAndMissingResources)' -count=1
```

Observed red results before implementation: routing misclassified ACR `/v2/...` requests as AWS/S3, and the ACR service returned the existing `404 NotFound` route-not-implemented response for data-plane paths.

- [x] **Step 4: Implement ACR data-plane first slice**

Add a separate versioned service key for `Microsoft.ContainerRegistry/registry@2021-07-01`, detect hosted ACR and local `-acr` data-plane routes before AWS/S3 routing, implement Registry v2 ping, catalog, tag list, manifest put/get/head/delete, deterministic `sha256` digests, Docker distribution headers, media-type preservation, registry-existence validation against ARM-created registries, and registry cleanup on ARM delete.

- [x] **Step 5: Verify ACR data-plane green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryDataPlaneActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run 'TestRegistryDataPlane(ManifestTagsAndCatalog|LocalRouteAndMissingResources)' -count=1
```

Expected: focused routing and ACR data-plane tests pass.

## Task 121: Azure Container Registry Blob Upload And Pull

**Docs verified:**
- Microsoft Learn `Azure Container Registry REST API reference`: `https://learn.microsoft.com/en-us/rest/api/containerregistry/`
- Microsoft Learn `Push your first image to your Azure container registry by using the Docker CLI`: `https://learn.microsoft.com/en-us/azure/container-registry/container-registry-get-started-docker-cli`
- Microsoft Learn `View container registry repositories`: `https://learn.microsoft.com/en-us/azure/container-registry/container-registry-repositories`
- CNCF Distribution `HTTP API V2`: `https://distribution.github.io/distribution/spec/api/`

- [x] **Step 1: Verify ACR blob gap**

Confirm Microsoft Learn documents ACR as a Docker-compatible registry used by Docker CLI `push` and `pull`, and that Registry v2 clients push image layers as blobs before manifest upload. Confirm the current ACR slice handled manifests, tags, and catalogs but did not store blobs or upload sessions.

- [x] **Step 2: Write failing blob routing and service tests**

Extend routing tests for `StartBlobUpload`, `UploadBlobChunk`, `CompleteBlobUpload`, `GetBlob`, `DeleteBlob`, and `CancelBlobUpload` on hosted ACR paths. Add service tests for upload start headers, chunk append range headers, digest-checked upload completion, blob `HEAD` and `GET`, blob delete, digest mismatch rejection with Docker-style `DIGEST_INVALID`, missing blob `404`, cancel upload, and canceled upload completion failure.

- [x] **Step 3: Verify blob red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryDataPlaneActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run 'TestRegistryDataPlaneBlob(UploadReadAndDelete|UploadDigestValidationAndCancel)' -count=1
```

Observed red results before implementation: routing returned generic actions such as `uploads`, `Update`, `CreateOrUpdate`, `Get`, and `Delete`; the service returned `404 NAME_UNKNOWN` route-not-implemented responses for blob upload paths.

- [x] **Step 4: Implement blob upload and pull behavior**

Add in-memory blob storage per registry/repository, upload session tracking with deterministic upload IDs, Docker upload response headers (`Location`, `Range`, `Docker-Upload-UUID`, `Content-Length`), `POST` start, `PATCH` chunk append, `PUT` digest-checked completion, `DELETE` cancel, blob `HEAD`/`GET`, blob delete, upload cleanup on registry delete, and Docker-style `errors` JSON for data-plane failures.

- [x] **Step 5: Verify blob green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerRegistryDataPlaneActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerregistry -run 'TestRegistryDataPlaneBlob(UploadReadAndDelete|UploadDigestValidationAndCancel)' -count=1
```

Expected: focused routing and ACR blob data-plane tests pass.

## Task 122: Azure Kubernetes Service Managed Cluster Upgrade Profile

**Docs verified:**
- Microsoft Learn `Managed Clusters - Get Upgrade Profile`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/get-upgrade-profile`
- Microsoft Learn `Managed Clusters - Create Or Update`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/create-or-update`

- [x] **Step 1: Verify AKS upgrade profile contract**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/upgradeProfiles/default?api-version=...` and a `ManagedClusterUpgradeProfile` response containing `properties.controlPlaneProfile` and `properties.agentPoolProfiles` upgrade entries.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `GetManagedClusterUpgradeProfile` and service coverage for the `upgradeProfiles/default` child resource, including deterministic control-plane and agent-pool profile shape plus missing-cluster and non-default-profile `404` behavior.

- [x] **Step 3: Verify upgrade profile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterUpgradeProfile -count=1
```

Observed red results before implementation: routing returned generic `Get` for `upgradeProfiles/default`, and the ContainerService handler returned `404 NotFound` with `The Container Service child route is not implemented.`

- [x] **Step 4: Implement mocked upgrade profile behavior**

Add `GetManagedClusterUpgradeProfile`, route detection for `managedClusters/{cluster}/upgradeProfiles/default`, child-route dispatch, deterministic `ManagedClusterUpgradeProfile` response generation, next-minor upgrade candidates derived from the stored cluster Kubernetes version, and `404 ResourceNotFound` handling for missing clusters or non-default profile names.

- [x] **Step 5: Verify upgrade profile green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterUpgradeProfile -count=1
```

Expected: focused routing and AKS upgrade-profile tests pass.

## Task 123: Azure Kubernetes Service Managed Cluster Start And Stop

**Docs verified:**
- Microsoft Learn `Managed Clusters - Start`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/start`
- Microsoft Learn `Managed Clusters - Stop`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/stop`

- [x] **Step 1: Verify AKS start/stop contract**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/start?api-version=...` and `POST .../stop?api-version=...`, with accepted/no-content long-running-operation-compatible responses and `Location` plus `Retry-After` headers on accepted responses.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `StartManagedCluster` and `StopManagedCluster`. Add service coverage that creates a managed cluster, stops it, verifies accepted operation headers, reads back `properties.powerState.code=Stopped`, starts it again, verifies `Running`, and checks missing-cluster `404` behavior.

- [x] **Step 3: Verify start/stop red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterStartStopActions -count=1
```

Observed red results before implementation: routing returned raw `stop` and `start` action names, and the ContainerService handler returned `404 NotFound` with `The Container Service action is not implemented.` for `stop`.

- [x] **Step 4: Implement mocked start/stop behavior**

Add `StartManagedCluster` and `StopManagedCluster` actions, route detection, action dispatch, immediate in-memory `PowerState` transitions, `properties.powerState.code` response projection, deterministic operation `Location`, and `Retry-After` headers.

- [x] **Step 5: Verify start/stop green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterStartStopActions -count=1
```

Expected: focused routing and AKS start/stop tests pass.

## Task 124: Azure Kubernetes Service List Kubernetes Versions

**Docs verified:**
- Microsoft Learn `Managed Clusters - List Kubernetes Versions`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/list-kubernetes-versions`

- [x] **Step 1: Verify AKS Kubernetes version discovery contract**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/providers/Microsoft.ContainerService/locations/{location}/kubernetesVersions?api-version=2026-03-01` and a `KubernetesVersionListResult` response containing a top-level `values` array with `version`, `isDefault`, `isPreview`, `patchVersions`, `upgrades`, and `capabilities.supportPlan` metadata.

- [x] **Step 2: Write failing routing, service, service-key, and provider-manifest tests**

Add routing coverage for `Microsoft.ContainerService/locations` as `ListKubernetesVersions`. Add service coverage for the `values` envelope, deterministic default and preview versions, patch upgrade paths, and `KubernetesOfficial` support-plan metadata. Extend ContainerService service-key coverage for `managedClusters@2026-03-01` and `locations@2026-03-01`. Extend Microsoft.Resources provider-manifest coverage for `Microsoft.ContainerService` resource types.

- [x] **Step 3: Verify Kubernetes versions red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSLocationActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run 'Test(ListKubernetesVersions|ServiceKeysIncludeAKSAPIVersions)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed red results before implementation: routing returned generic `Get`, ContainerService returned `404 NotFound` with `The Container Service route is not implemented.`, service keys omitted `2026-03-01` and the `locations` service key, and Microsoft.Resources returned `ProviderNotFound` for `Microsoft.ContainerService`.

- [x] **Step 4: Implement mocked Kubernetes version discovery**

Register `Microsoft.ContainerService/managedClusters@2026-03-01` and `Microsoft.ContainerService/locations@2026-03-01`, add location-route parsing, implement `ListKubernetesVersions`, return deterministic Kubernetes version profiles with patch upgrade metadata, add route action detection, and add `Microsoft.ContainerService` provider manifest entries for `managedClusters` and `locations/kubernetesVersions`.

- [x] **Step 5: Verify Kubernetes versions green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSLocationActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run 'Test(ListKubernetesVersions|ServiceKeysIncludeAKSAPIVersions)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected: focused routing, ContainerService, and provider-manifest tests pass.

## Task 125: Azure Kubernetes Service Agent Pool Upgrade Profile

**Docs verified:**
- Microsoft Learn `Agent Pools - Get Upgrade Profile`: `https://learn.microsoft.com/en-us/rest/api/aks/agent-pools/get-upgrade-profile?view=rest-aks-2026-03-01`

- [x] **Step 1: Verify AKS agent-pool upgrade profile contract**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/agentPools/{agentPoolName}/upgradeProfiles/default?api-version=2026-03-01` and an `AgentPoolUpgradeProfile` response containing `properties.kubernetesVersion`, `properties.latestNodeImageVersion`, `properties.osType`, and `properties.upgrades`.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `GetAgentPoolUpgradeProfile`. Add service coverage that creates a cluster with Linux and Windows pools, reads the Windows pool's `upgradeProfiles/default` child resource, verifies resource identity, node image metadata, Kubernetes upgrade candidates, and checks missing-pool and non-default-profile `404` behavior.

- [x] **Step 3: Verify agent-pool upgrade profile red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/get_agent_pool_upgrade_profile -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestAgentPoolUpgradeProfile -count=1
```

Observed red results before implementation: routing returned `GetManagedClusterUpgradeProfile` for the deeper agent-pool profile path, and the ContainerService handler returned `404 NotFound` with `The Container Service route is not implemented.`

- [x] **Step 4: Implement mocked agent-pool upgrade profile behavior**

Add `GetAgentPoolUpgradeProfile` route detection, extend the ContainerService route parser for nested `agentPools/{pool}/upgradeProfiles/{profile}` child paths, dispatch nested child requests, and return deterministic `AgentPoolUpgradeProfile` responses derived from stored cluster and pool state.

- [x] **Step 5: Verify agent-pool upgrade profile green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/get_agent_pool_upgrade_profile -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestAgentPoolUpgradeProfile -count=1
```

Expected: focused routing and ContainerService tests pass.

## Task 126: Azure Kubernetes Service Run Command And Get Command Result

**Docs verified:**
- Microsoft Learn `Managed Clusters - Run Command`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/run-command?view=rest-aks-2026-03-01`
- Microsoft Learn `Managed Clusters - Get Command Result`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/get-command-result?view=rest-aks-2026-03-01`

- [x] **Step 1: Verify AKS command contracts**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/runCommand?api-version=2026-03-01`, a request body with `command`, optional `clusterToken`, and optional base64 `context`, and a `RunCommandResult` response containing `id`, `properties.exitCode`, `properties.logs`, `properties.provisioningState`, `properties.startedAt`, and `properties.finishedAt`. Confirm `GET .../managedClusters/{resourceName}/commandResults/{commandId}?api-version=2026-03-01` returns the same result envelope or pending long-running-operation headers.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `RunCommand` and `GetCommandResult`. Add service coverage that creates a cluster, submits a mocked command, verifies the Azure `RunCommandResult` envelope, retrieves the stored result by command ID, checks missing-command `404`, and rejects empty command bodies with `400`.

- [x] **Step 3: Verify command red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureAKSManagedClusterActions/(run_command|get_command_result)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterRunCommandAndGetResult -count=1
```

Observed red results before implementation: routing returned raw `runCommand` and generic `Get`, and the ContainerService handler returned `404 NotFound` with `The Container Service action is not implemented.`

- [x] **Step 4: Implement mocked command behavior**

Add AKS-specific action detection for `runCommand` and `commandResults/{commandId}`, register service actions, add in-memory deterministic command-result storage, validate non-empty commands, return immediate mocked `RunCommandResult` envelopes, and support result lookup by cluster and command ID.

- [x] **Step 5: Verify command green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureAKSManagedClusterActions/(run_command|get_command_result)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterRunCommandAndGetResult -count=1
```

Expected: focused routing and ContainerService tests pass.

## Task 127: Azure Kubernetes Service Cluster Monitoring User Credentials

**Docs verified:**
- Microsoft Learn `Managed Clusters - List Cluster Monitoring User Credentials`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/list-cluster-monitoring-user-credentials?view=rest-aks-2026-03-01`

- [x] **Step 1: Verify AKS monitoring credential contract**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/listClusterMonitoringUserCredential?api-version=2026-03-01`, optional `server-fqdn`, and a `CredentialResults` response with a top-level `kubeconfigs` array containing `name` and base64 `value`.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `ListClusterMonitoringUserCredentials`. Add service coverage that creates a cluster, requests monitoring user credentials with `server-fqdn=private`, verifies the `clusterMonitoringUser` kubeconfig name, decodes the base64 kubeconfig, checks credential/server FQDN projection, and checks missing-cluster `404`.

- [x] **Step 3: Verify monitoring credential red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/list_monitoring_user_credentials -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterMonitoringUserCredentials -count=1
```

Observed red results before implementation: routing returned raw `listClusterMonitoringUserCredential`, and the ContainerService handler returned `404 NotFound` with `The Container Service action is not implemented.`

- [x] **Step 4: Implement mocked monitoring credential behavior**

Add AKS-specific action detection for `listClusterMonitoringUserCredential`, register the service action, dispatch to the shared credential envelope, generate per-credential kubeconfigs, and project the optional `server-fqdn` query into deterministic mocked server URLs.

- [x] **Step 5: Verify monitoring credential green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/list_monitoring_user_credentials -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterMonitoringUserCredentials -count=1
```

Expected: focused routing and ContainerService tests pass.

## Task 128: Azure Kubernetes Service Agent Pool Upgrade Node Image Version

**Docs verified:**
- Microsoft Learn `Agent Pools - Upgrade Node Image Version`: `https://learn.microsoft.com/en-us/rest/api/aks/agent-pools/upgrade-node-image-version?view=rest-aks-2026-03-01`

- [x] **Step 1: Verify AKS agent-pool node-image upgrade contract**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/agentPools/{agentPoolName}/upgradeNodeImageVersion?api-version=2026-03-01`, `200 OK` or `202 Accepted`, and long-running-operation headers `Azure-AsyncOperation`, `Location`, and `Retry-After` on accepted responses. Confirm the accepted response body is an `Agent Pool` resource with `properties.nodeImageVersion` and `properties.provisioningState`.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `UpgradeAgentPoolNodeImageVersion`. Add service coverage that creates a cluster with a Linux agent pool, posts `upgradeNodeImageVersion`, verifies `202 Accepted` operation headers, validates the projected latest node image version and `UpgradingNodeImageVersion` state, reads back the stored pool, and checks missing-pool `404`.

- [x] **Step 3: Verify node-image upgrade red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/upgrade_agent_pool_node_image_version -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestAgentPoolUpgradeNodeImageVersion -count=1
```

Observed red results before implementation: routing returned raw `upgradeNodeImageVersion`, and the ContainerService handler returned `404 NotFound` with `The Container Service child route is not implemented.`

- [x] **Step 4: Implement mocked node-image upgrade behavior**

Add AKS-specific action detection for `upgradeNodeImageVersion`, register the service action, preserve deterministic initial node-image versions on agent pools, update the target pool to the latest mocked node-image version, project `UpgradingNodeImageVersion`, and return accepted long-running-operation-compatible headers.

- [x] **Step 5: Verify node-image upgrade green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/upgrade_agent_pool_node_image_version -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestAgentPoolUpgradeNodeImageVersion -count=1
```

Expected: focused routing and ContainerService tests pass.

## Task 129: Azure Container Apps Subscription Lists

**Docs verified:**
- Microsoft Learn `Container Apps - List By Subscription`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/list-by-subscription?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Managed Environments - List By Subscription`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/managed-environments/list-by-subscription?view=rest-resource-manager-containerapps-2025-07-01`

- [x] **Step 1: Verify Container Apps subscription-list contracts**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/providers/Microsoft.App/containerApps?api-version=2025-07-01` returning a `Container App Collection` with top-level `value`, and `GET /subscriptions/{subscriptionId}/providers/Microsoft.App/managedEnvironments?api-version=2025-07-01` returning a managed environment collection.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for subscription-scoped container app and managed environment list paths. Add service coverage that creates environments and apps across two resource groups and two subscriptions, then verifies subscription-scoped list responses include only the requested subscription and are stable name-ordered Azure `value` envelopes.

- [x] **Step 3: Verify subscription-list red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureContainerAppsActions/(list_container_apps_by_subscription|list_managed_environments_by_subscription)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppsSubscriptionScopedLists -count=1
```

Observed red results before implementation: routing already returned the expected list actions, while the ContainerApps service returned `404 NotFound` with `The Container Apps route is not implemented.` for subscription-scoped list paths.

- [x] **Step 4: Implement subscription-scoped list behavior**

Extend the Container Apps route parser to accept `/subscriptions/{subscriptionId}/providers/Microsoft.App/{resourceType}` paths and update shared list filtering so an empty resource group means subscription-wide enumeration while preserving resource-group list behavior.

- [x] **Step 5: Verify subscription-list green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureContainerAppsActions/(list_container_apps_by_subscription|list_managed_environments_by_subscription)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppsSubscriptionScopedLists -count=1
```

Expected: focused routing and ContainerApps tests pass.

## Task 130: Azure Kubernetes Service Agent Pool Abort Latest Operation

**Docs verified:**
- Microsoft Learn `Agent Pools - Abort Latest Operation`: `https://learn.microsoft.com/en-us/rest/api/aks/agent-pools/abort-latest-operation?view=rest-aks-2026-03-01`

- [x] **Step 1: Verify AKS agent-pool abort contract**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/agentPools/{agentPoolName}/abort?api-version=2026-03-01`, `202 Accepted`, and long-running-operation headers `Azure-AsyncOperation`, `Location`, and `Retry-After`. Confirm the operation moves the agent pool toward a canceled state and may return conflict when no current operation can be canceled.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `AbortAgentPoolLatestOperation`. Add service coverage that creates a cluster, starts a mocked node-image upgrade, posts `agentPools/{pool}/abort`, verifies `202 Accepted` operation headers, reads back `properties.provisioningState=Canceled`, verifies a second idle abort returns `409 Conflict`, and checks missing-pool `404`.

- [x] **Step 3: Verify agent-pool abort red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/abort_agent_pool_latest_operation -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestAgentPoolAbortLatestOperation -count=1
```

Observed red results before implementation: routing returned raw `abort`, and the ContainerService handler returned `404 NotFound` with `The Container Service child route is not implemented.`

- [x] **Step 4: Implement mocked agent-pool abort behavior**

Add AKS-specific action detection for `agentPools/{pool}/abort`, register the service action, dispatch the nested child route, treat known in-progress agent-pool provisioning states as abortable, move aborted pools to `Canceled`, return accepted long-running-operation-compatible headers, and return `409 OperationNotAllowed` when no current operation is abortable.

- [x] **Step 5: Verify agent-pool abort green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAKSManagedClusterActions/abort_agent_pool_latest_operation -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestAgentPoolAbortLatestOperation -count=1
```

Expected: focused routing and ContainerService tests pass.

## Task 131: Azure Kubernetes Service Managed Cluster Rotation Actions

**Docs verified:**
- Microsoft Learn `Managed Clusters - Rotate Cluster Certificates`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/rotate-cluster-certificates?view=rest-aks-2026-03-01`
- Microsoft Learn `Managed Clusters - Rotate Service Account Signing Keys`: `https://learn.microsoft.com/en-us/rest/api/aks/managed-clusters/rotate-service-account-signing-keys?view=rest-aks-2026-03-01`

- [x] **Step 1: Verify AKS managed-cluster rotation contracts**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/rotateClusterCertificates?api-version=2026-03-01` and `POST .../rotateServiceAccountSigningKeys?api-version=2026-03-01`. Both operations return `202 Accepted` with a `Location` operation URL.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `RotateClusterCertificates` and `RotateServiceAccountSigningKeys`. Add service coverage that creates a cluster, posts both rotation actions, verifies `202 Accepted` and deterministic operation `Location` headers, and checks missing-cluster `404`.

- [x] **Step 3: Verify rotation red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureAKSManagedClusterActions/(rotate_cluster_certificates|rotate_service_account_signing_keys)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterRotationActions -count=1
```

Observed red results before implementation: routing returned raw `rotateClusterCertificates` and `rotateServiceAccountSigningKeys`, and the ContainerService handler returned `404 NotFound` with `The Container Service action is not implemented.`

- [x] **Step 4: Implement mocked rotation behavior**

Add AKS-specific action detection, register service actions, dispatch both managed-cluster rotation actions through a shared accepted-operation helper, verify cluster existence, and return long-running-operation-compatible `202 Accepted` responses with deterministic `Location` and `Retry-After` headers.

- [x] **Step 5: Verify rotation green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureAKSManagedClusterActions/(rotate_cluster_certificates|rotate_service_account_signing_keys)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerservice -run TestManagedClusterRotationActions -count=1
```

Expected: focused routing and ContainerService tests pass.

## Task 132: Azure Container Apps List Secrets

**Docs verified:**
- Microsoft Learn `Container Apps - List Secrets`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/list-secrets?view=rest-resource-manager-containerapps-2025-07-01`

- [x] **Step 1: Verify Container Apps list secrets contract**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/containerApps/{containerAppName}/listSecrets?api-version=2025-07-01` and a `200 OK` `SecretsCollection` response with top-level `value` containing `ContainerAppSecret` entries.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `ListContainerAppSecrets`. Add service coverage that creates a container app with `properties.configuration.secrets`, posts `listSecrets`, verifies the Azure `value` envelope preserves stored secret names and values with stable ordering, and checks missing-app `404`.

- [x] **Step 3: Verify list secrets red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerAppsActions/list_container_app_secrets -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppListSecrets -count=1
```

Observed red results before implementation: routing returned raw `listSecrets`, and the ContainerApps service returned `404 NotFound` with `The Container Apps route is not implemented.`

- [x] **Step 4: Implement list secrets behavior**

Add Container Apps-specific action detection for `listSecrets`, register the service action, extend the route parser for container app action paths, dispatch `POST listSecrets`, and return stored `properties.configuration.secrets` as a copied, stable name-ordered `value` envelope.

- [x] **Step 5: Verify list secrets green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerAppsActions/list_container_app_secrets -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppListSecrets -count=1
```

Expected: focused routing and ContainerApps tests pass.

## Task 133: Azure Container Apps Revisions List And Get

**Docs verified:**
- Microsoft Learn `Container Apps Revisions - List Revisions`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps-revisions/list-revisions?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Container Apps Revisions - Get Revision`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps-revisions/get-revision?view=rest-resource-manager-containerapps-2025-07-01`

- [x] **Step 1: Verify Container Apps revision contracts**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/containerApps/{containerAppName}/revisions?api-version=2025-07-01` returning a `RevisionCollection` with top-level `value`, and `GET .../revisions/{revisionName}?api-version=2025-07-01` returning a `Revision` resource.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `ListContainerAppRevisions` and `GetContainerAppRevision`. Add service coverage that creates a container app, lists revisions, verifies the deterministic latest revision identity, active/running/provisioned state, stored template projection, gets the revision by name, and checks missing-revision plus missing-app `404`.

- [x] **Step 3: Verify revision red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureContainerAppsActions/(list_container_app_revisions|get_container_app_revision)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppRevisionsListAndGet -count=1
```

Observed red results before implementation: routing returned generic `Get`, and the ContainerApps service returned `404 NotFound` with `The Container Apps action is not implemented.`

- [x] **Step 4: Implement revision list/get behavior**

Add Container Apps-specific action detection for `revisions`, register revision actions, extend the route parser for child revision paths, dispatch list/get requests, and project a deterministic active revision from the stored app's latest revision and template state.

- [x] **Step 5: Verify revision green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run 'TestDetectTarget_AzureContainerAppsActions/(list_container_app_revisions|get_container_app_revision)' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppRevisionsListAndGet -count=1
```

Expected: focused routing and ContainerApps tests pass.

## Task 134: Azure Container Apps Revision Actions

**Docs verified:**
- Microsoft Learn `Container Apps Revisions - Activate Revision`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps-revisions/activate-revision?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Container Apps Revisions - Deactivate Revision`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps-revisions/deactivate-revision?view=rest-resource-manager-containerapps-2025-07-01`
- Microsoft Learn `Container Apps Revisions - Restart Revision`: `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps-revisions/restart-revision?view=rest-resource-manager-containerapps-2025-07-01`

- [x] **Step 1: Verify Container Apps revision action contracts**

Confirm Microsoft Learn documents `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/containerApps/{containerAppName}/revisions/{revisionName}/activate?api-version=2025-07-01`, `POST .../deactivate?api-version=2025-07-01`, and `POST .../restart?api-version=2025-07-01`, each returning `200 OK`.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for `ActivateContainerAppRevision`, `DeactivateContainerAppRevision`, and `RestartContainerAppRevision`. Add service coverage that creates a container app, deactivates the deterministic latest revision, verifies subsequent get returns inactive/stopped state, activates the revision back to active/running state, restarts it with a `200 OK` response, and checks missing-revision `404`.

- [x] **Step 3: Verify revision action red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerAppsActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppRevisionActions -count=1
```

Observed red results before implementation: routing returned raw action names `activate`, `deactivate`, and `restart`, and the ContainerApps service returned `404 NotFound` with `The Container Apps route is not implemented.` for the revision action route.

- [x] **Step 4: Implement revision action behavior**

Add Container Apps-specific action detection for revision `activate`, `deactivate`, and `restart`, register service actions with Azure action IAM names, extend the route parser for `containerApps/{app}/revisions/{revision}/{action}`, dispatch the three `POST` actions, persist mutable projected revision active/running state, remove revision state when the owning app is deleted, and return `200 OK` projected revision resources.

- [x] **Step 5: Verify revision action green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureContainerAppsActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/containerapps -run TestContainerAppRevisionActions -count=1
```

Expected: focused routing and ContainerApps tests pass.

## Task 135: Azure App Service Subscription Lists

**Docs verified:**
- Microsoft Learn `Web Apps - List`: `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/list?view=rest-appservice-2024-04-01`
- Microsoft Learn `App Service Plans - List`: `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-plans/list?view=rest-appservice-2024-04-01`

- [x] **Step 1: Verify App Service subscription-list contracts**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/providers/Microsoft.Web/sites?api-version=2024-04-01` and `GET /subscriptions/{subscriptionId}/providers/Microsoft.Web/serverfarms?api-version=2024-04-01`, both returning `200 OK` collection responses with top-level `value`.

- [x] **Step 2: Write failing routing and service tests**

Add routing coverage for named `ListSites` and `ListPlans` actions on subscription-scoped Microsoft.Web collection routes. Add service coverage that creates App Service plans and Web Apps across two resource groups in one subscription plus another subscription, lists `/subscriptions/sub-1/providers/Microsoft.Web/serverfarms` and `/subscriptions/sub-1/providers/Microsoft.Web/sites`, verifies stable name ordering, and verifies resources from other subscriptions are excluded.

- [x] **Step 3: Verify App Service subscription-list red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppServiceSubscriptionLists -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestAppServiceSubscriptionScopedLists -count=1
```

Observed red results before implementation: routing returned generic `List` for both subscription collection paths, and the AppService handler returned `404 NotFound` with `The App Service route is not implemented.` for the subscription plan list path.

- [x] **Step 4: Implement App Service subscription-list behavior**

Add Microsoft.Web plan and site collection action detection for resource-group and subscription-scoped provider paths, extend App Service route parsing for `/subscriptions/{subscriptionId}/providers/Microsoft.Web/{resourceType}`, and update plan/site list filtering so an empty resource group means subscription-wide enumeration while preserving existing resource-group list behavior.

- [x] **Step 5: Verify App Service subscription-list green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureAppServiceSubscriptionLists -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/appservice -run TestAppServiceSubscriptionScopedLists -count=1
```

Expected: focused routing and AppService tests pass.

## Task 136: Azure Application Insights Components

**Docs verified:**
- Microsoft Learn `Components - Create Or Update`: `https://learn.microsoft.com/en-us/rest/api/application-insights/components/create-or-update?view=rest-application-insights-2015-05-01`
- Microsoft Learn `Components - Get`: `https://learn.microsoft.com/en-us/rest/api/application-insights/components/get?view=rest-application-insights-2015-05-01`
- Microsoft Learn `Components - List`: `https://learn.microsoft.com/en-us/rest/api/application-insights/components/list?view=rest-application-insights-2015-05-01`
- Microsoft Learn `Components - Delete`: `https://learn.microsoft.com/en-us/rest/api/application-insights/components/delete?view=rest-application-insights-2015-05-01`

- [x] **Step 1: Verify Application Insights component contracts**

Confirm Microsoft Learn documents `PUT`, `GET`, and `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Insights/components/{resourceName}?api-version=2015-05-01`, plus `GET /subscriptions/{subscriptionId}/providers/Microsoft.Insights/components?api-version=2015-05-01` for subscription-wide listing. Create/update, get, and list return `200 OK`; delete returns `200 OK` or `204 No Content`.

- [x] **Step 2: Write failing routing, service, template, and provider-manifest tests**

Add routing coverage for `CreateOrUpdateComponent`, `GetComponent`, `ListComponents`, and `DeleteComponent`. Add monitor service coverage for create/update/get/subscription-list/delete, deterministic read-only IDs and connection string projection, update stability, ARM template provisioning, and the `azure|Microsoft.Insights/components|2015-05-01` service key. Extend the provider manifest test to require `Microsoft.Insights/components` with API version `2015-05-01`.

- [x] **Step 3: Verify Application Insights component red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureApplicationInsightsComponentActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run 'TestApplicationInsightsComponentLifecycleAndTemplateProvisioning|TestMonitorServiceKeysIncludeVersionedResources' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed red results before implementation: routing returned generic `CreateOrUpdate`, `Get`, `List`, and `Delete`; the Monitor handler returned `404 NotFound` for component routes and lacked the component service key; and the provider manifest lacked `components`.

- [x] **Step 4: Implement component lifecycle behavior**

Add `Microsoft.Insights/components` service keys and actions, route parsing for resource-group component routes and subscription-scoped component lists, component create/update/get/list/delete state, deterministic `ApplicationId`, `InstrumentationKey`, `AppId`, `TenantId`, connection-string defaults, idempotent missing-resource delete, ARM template provisioning, provider manifest metadata, and component-specific routing action detection.

- [x] **Step 5: Verify Application Insights component green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureApplicationInsightsComponentActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run 'TestApplicationInsightsComponentLifecycleAndTemplateProvisioning|TestMonitorServiceKeysIncludeVersionedResources' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Expected: focused routing, Monitor, and provider manifest tests pass.

## Task 137: Azure Application Insights Query GET

**Docs verified:**
- Microsoft Learn `Query - Get`: `https://learn.microsoft.com/en-us/rest/api/application-insights/query/get?view=rest-application-insights-v1`

- [x] **Step 1: Verify Application Insights query contract**

Confirm Microsoft Learn documents `GET https://api.applicationinsights.io/v1/apps/{appId}/query?query={query}` with optional `timespan`, API version `v1`, OAuth security, and `200 OK` responses using a top-level `tables` array with column and row data.

- [x] **Step 2: Write failing routing and service-key tests**

Add routing coverage that classifies `api.applicationinsights.io` as Azure, extracts `v1` from the path, routes to `Microsoft.Insights/query`, and emits `QueryGet`. Extend Monitor service-key coverage to require `azure|Microsoft.Insights/query|v1`.

- [x] **Step 3: Write failing query response tests**

Add Monitor service coverage that calls `/v1/apps/{appId}/query?query=requests | take 1`, expects `200 OK`, verifies the Azure `tables` envelope, column metadata, deterministic row width, app ID, and item type, and verifies missing `query` returns `400 BadRequest`.

- [x] **Step 4: Verify query red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureApplicationInsightsQueryDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run 'TestApplicationInsightsQueryGet|TestMonitorServiceKeysIncludeVersionedResources' -count=1
```

Observed red results before implementation: routing treated the host as AWS/S3 with no API version, and the Monitor handler returned `404 NotFound` while service keys lacked `Microsoft.Insights/query|v1`.

- [x] **Step 5: Implement query GET behavior**

Add Application Insights query host detection, `v1` path API-version extraction, query-specific action detection, a versioned Monitor service key, direct Monitor parsing for `/v1/apps/{appId}/query`, required `query` validation, deterministic `tables` results for request/trace/count-shaped query inputs, and Azure-compatible JSON error responses.

- [x] **Step 6: Verify query green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureApplicationInsightsQueryDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run 'TestApplicationInsightsQueryGet|TestMonitorServiceKeysIncludeVersionedResources' -count=1
```

Expected: focused routing and Monitor query tests pass.

## Task 138: Azure Log Analytics Workspaces

**Docs verified:**
- Microsoft Learn `Workspaces - Create Or Update`: `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/create-or-update?view=rest-loganalytics-2025-02-01`
- Microsoft Learn `Workspaces - Get`: `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/get?view=rest-loganalytics-2025-02-01`
- Microsoft Learn `Workspaces - List By Resource Group`: `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/list-by-resource-group?view=rest-loganalytics-2025-02-01`
- Microsoft Learn `Workspaces - Delete`: `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/delete?view=rest-loganalytics-2025-02-01`

- [x] **Step 1: Verify Log Analytics workspace contracts**

Confirm Microsoft Learn documents `PUT`, `GET`, and `DELETE /subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/Microsoft.OperationalInsights/workspaces/{workspaceName}?api-version=2025-02-01`, plus `GET /subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/Microsoft.OperationalInsights/workspaces?api-version=2025-02-01` for resource-group listing. Create/update returns `200 OK`, `201 Created`, or `202 Accepted`; get and list return `200 OK`; delete returns `200 OK`, `202 Accepted`, or `204 No Content`.

- [x] **Step 2: Write failing routing, service, template, and provider-manifest tests**

Add routing coverage for `CreateOrUpdateWorkspace`, `GetWorkspace`, `ListWorkspacesByResourceGroup`, and `DeleteWorkspace`. Add Log Analytics service coverage for create/update/get/list/delete, stable read-only `customerId`, SKU and retention preservation, feature preservation, ARM template provisioning, and the `azure|Microsoft.OperationalInsights/workspaces|2025-02-01` service key. Extend the provider manifest test to require `Microsoft.OperationalInsights/workspaces` with API version `2025-02-01`.

- [x] **Step 3: Verify Log Analytics workspace red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureLogAnalyticsWorkspaceActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/loganalytics -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed red results before implementation: routing returned generic `CreateOrUpdate`, `Get`, `List`, and `Delete`; the Log Analytics package had no `New` implementation; and the provider manifest returned `ProviderNotFound` for `Microsoft.OperationalInsights`.

- [x] **Step 4: Implement workspace lifecycle behavior**

Add the `services/azure/loganalytics` package, versioned service key, action metadata, workspace create/update/get/list/delete state, deterministic `customerId`, default SKU, retention, public network access, workspace capping, timestamps, idempotent missing-resource delete, ARM template provisioning, gateway registration, provider manifest metadata, and workspace-specific routing action detection.

- [x] **Step 5: Verify Log Analytics workspace green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureLogAnalyticsWorkspaceActions -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/loganalytics -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed green results: focused routing, Log Analytics, and provider manifest tests pass.

## Task 139: Azure Log Analytics Workspace Query POST

**Docs verified:**
- Microsoft Learn `Azure Monitor Logs query API overview`: `https://learn.microsoft.com/en-us/azure/azure-monitor/logs/api/overview`

Note: the older checklist link `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspace-query/query` currently returns `404 Content not found`; the Learn overview updated on May 27, 2026 documents the public endpoint, request body, and response envelope.

- [x] **Step 1: Verify Log Analytics query contract**

Confirm Microsoft Learn documents `POST https://api.loganalytics.azure.com/v1/workspaces/{workspaceId}/query` with `Authorization: Bearer`, JSON request bodies containing `query` and optional `timespan`, and `200 OK` JSON responses containing a top-level `tables` array with `columns` and `rows`. Also keep `api.loganalytics.io` as a compatibility alias for older clients.

- [x] **Step 2: Write failing routing and service-key tests**

Add routing coverage that classifies both `api.loganalytics.azure.com` and `api.loganalytics.io` as Azure, extracts `v1` from the path, routes to `Microsoft.OperationalInsights/query`, and emits `QueryWorkspace`. Extend Log Analytics service-key coverage to require `azure|Microsoft.OperationalInsights/query|v1`.

- [x] **Step 3: Write failing query response tests**

Add Log Analytics service coverage for `POST /v1/workspaces/{workspaceId}/query` with body `{"query":"AzureActivity | summarize count() by Category","timespan":"PT12H"}`, expecting `200 OK`, the Azure `tables` envelope, `Category` and `count_` columns, deterministic rows, legacy host support for `Heartbeat | take 1`, and `400 BadRequest` for missing query text.

- [x] **Step 4: Verify query red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureLogAnalyticsQueryDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/loganalytics -run 'TestWorkspace(QueryPost|ServiceKeys)' -count=1
```

Observed red results before implementation: routing returned an empty provider, service, action, and API version for both Log Analytics query hosts; the Log Analytics service lacked the `Microsoft.OperationalInsights/query|v1` key; and the service returned `404 NotFound` for query POST requests.

- [x] **Step 5: Implement query POST behavior**

Add Log Analytics query host detection, `v1` path API-version extraction, query-specific action detection, a versioned Log Analytics query service key, direct Log Analytics parsing for `/v1/workspaces/{workspaceId}/query`, required query validation, deterministic `tables` results for category count, generic count, and table-take shaped query inputs, and Azure-compatible JSON error responses.

- [x] **Step 6: Verify query green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureLogAnalyticsQueryDataPlane -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/loganalytics -run 'TestWorkspace(QueryPost|ServiceKeys)' -count=1
```

Observed green results: focused routing and Log Analytics query tests pass.

## Task 140: Azure Monitor Activity Logs List

**Docs verified:**
- Microsoft Learn `Activity Logs - List`: `https://learn.microsoft.com/en-us/rest/api/monitor/activity-logs/list?view=rest-monitor-2015-04-01`

- [x] **Step 1: Verify Activity Logs List contract**

Confirm Microsoft Learn documents `GET /subscriptions/{subscriptionId}/providers/Microsoft.Insights/eventtypes/management/values?api-version=2015-04-01&$filter={filter}` with optional `$select`, `200 OK` responses using `EventDataCollection`, and documented fields such as `value`, `nextLink`, `authorization`, `eventDataId`, `eventTimestamp`, `operationName`, `resourceGroupName`, `resourceProviderName`, `status`, `subStatus`, `submissionTimestamp`, and `subscriptionId`. The `$filter` argument is required and must include at least `eventTimestamp ge`.

- [x] **Step 2: Write failing routing, service, service-key, and provider-manifest tests**

Add routing coverage for `Microsoft.Insights/eventtypes` returning `ListActivityLogs` on the documented subscription-scoped path. Add Monitor service coverage for required `$filter`, deterministic `EventDataCollection.value`, localized operation/status/provider fields, authorization details, `$select` projection, and the `azure|Microsoft.Insights/eventtypes|2015-04-01` service key. Extend the provider manifest test to require `Microsoft.Insights/eventtypes` with API version `2015-04-01`.

- [x] **Step 3: Verify Activity Logs red**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureActivityLogsList -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run 'TestActivityLogsListAndSelect|TestMonitorServiceKeysIncludeVersionedResources' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed red results before implementation: routing returned generic `Get`, the Monitor handler returned `404 NotFound`, the Monitor service key list lacked `Microsoft.Insights/eventtypes|2015-04-01`, and the provider manifest lacked `eventtypes`.

- [x] **Step 4: Implement Activity Logs List behavior**

Add activity log action detection, `2015-04-01` versioned service key, direct Monitor parsing for `/subscriptions/{subscriptionId}/providers/Microsoft.Insights/eventtypes/management/values`, required `$filter` validation, deterministic `EventDataCollection.value` output, resource group/resource/provider/correlation filter extraction, `$select` projection, localized string helpers, and provider manifest metadata.

- [x] **Step 5: Verify Activity Logs green**

Run:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing -run TestDetectTarget_AzureActivityLogsList -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/monitor -run 'TestActivityLogsListAndSelect|TestMonitorServiceKeysIncludeVersionedResources' -count=1
env GOCACHE=/private/tmp/go-build go test ./services/azure/resources -run TestProviderManifestListGetAndRegister -count=1
```

Observed green results: focused routing, Monitor, and provider manifest tests pass.

## Verification Gates

- [x] Before any completion claim, run:

```bash
env GOCACHE=/private/tmp/go-build go test ./cmd/gateway ./pkg/routing ./pkg/gateway ./pkg/azurearm ./services/azure/resources ./services/azure/storage ./services/azure/keyvault ./services/azure/authorization ./services/azure/compute ./services/azure/appservice ./services/azure/appconfiguration ./services/azure/network ./services/azure/monitor ./services/azure/loganalytics ./services/azure/servicebus ./services/azure/eventgrid ./services/azure/eventhub ./services/azure/containerregistry ./services/azure/containerinstance ./services/azure/containerapps ./services/azure/containerservice ./services/azure/apimanagement ./services/azure/dns ./services/azure/cosmosdb ./services/azure/sql ./services/azure/postgresql ./services/azure/redis ./services/azure/managedidentity
```

- [x] Before merging any Azure service slice, run the existing AWS compatibility subset:

```bash
env GOCACHE=/private/tmp/go-build go test ./pkg/routing ./pkg/gateway ./services/s3 ./services/dynamodb ./services/sqs ./services/lambda
```

- [ ] Before merging the umbrella branch, run the repo-wide test target when practical:

```bash
env GOCACHE=/private/tmp/go-build go test ./...
```
