package appconfiguration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestConfigurationStoreLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	storeURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.AppConfiguration/configurationStores/cfgstore?api-version=2024-06-01"
	storePayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard"},
		"tags":{"env":"test"},
		"identity":{"type":"SystemAssigned"},
		"properties":{
			"disableLocalAuth":false,
			"dataPlaneProxy":{"authenticationMode":"Local","privateLinkDelegation":"Disabled"}
		}
	}`)

	createResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, storeURL, storePayload))
	if err != nil {
		t.Fatalf("create store returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create store status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	store := decodeAppConfigResponse(t, createResp)
	if store["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.AppConfiguration/configurationStores/cfgstore" {
		t.Fatalf("unexpected store id: %v", store["id"])
	}
	if store["name"] != "cfgstore" || store["type"] != "Microsoft.AppConfiguration/configurationStores" || store["location"] != "eastus" {
		t.Fatalf("unexpected store identity fields: %v", store)
	}
	props := store["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" || props["endpoint"] != "https://cfgstore.azconfig.io" {
		t.Fatalf("unexpected store properties: %v", props)
	}
	if store["sku"].(map[string]any)["name"] != "Standard" {
		t.Fatalf("unexpected sku: %v", store["sku"])
	}

	getResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, storeURL, nil))
	if err != nil {
		t.Fatalf("get store returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get store status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.AppConfiguration/configurationStores?api-version=2024-06-01", nil))
	if err != nil {
		t.Fatalf("list stores returned error: %v", err)
	}
	listed := decodeAppConfigResponse(t, listResp)
	if len(listed["value"].([]any)) != 1 {
		t.Fatalf("expected one store in list, got %v", listed)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.AppConfiguration/configurationStores",
		"name":     "templated",
		"location": "westus2",
		"sku":      map[string]any{"name": "Free"},
		"properties": map[string]any{
			"disableLocalAuth": true,
		},
	})
	if err != nil {
		t.Fatalf("provision store returned error: %v", err)
	}
	templateStore := templateResult.(map[string]any)
	if templateStore["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.AppConfiguration/configurationStores/templated" {
		t.Fatalf("unexpected provisioned store id: %v", templateStore["id"])
	}
	if templateStore["location"] != "westus2" {
		t.Fatalf("unexpected provisioned store location: %v", templateStore["location"])
	}

	deleteResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodDelete, storeURL, nil))
	if err != nil {
		t.Fatalf("delete store returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete store status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
}

func TestConfigurationStoreListKeysReturnsAccessKeys(t *testing.T) {
	svc := New()

	storeURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.AppConfiguration/configurationStores/cfgstore?api-version=2024-06-01"
	createResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, storeURL, []byte(`{"location":"eastus","sku":{"name":"Standard"}}`)))
	if err != nil {
		t.Fatalf("create store returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create store status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.AppConfiguration/configurationStores/cfgstore/listKeys?api-version=2024-06-01"
	listKeysResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list store keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}

	body := decodeAppConfigResponse(t, listKeysResp)
	keys := body["value"].([]any)
	if len(keys) != 4 {
		t.Fatalf("expected four access keys, got %v", body)
	}
	primary := keys[0].(map[string]any)
	if primary["name"] != "Primary" || primary["readOnly"] != false {
		t.Fatalf("expected first key to be writable Primary, got %v", primary)
	}
	if primary["value"] != "cm-appconfig-cfgstore-primary" {
		t.Fatalf("expected deterministic primary secret, got %v", primary["value"])
	}
	if primary["connectionString"] != "Endpoint=https://cfgstore.azconfig.io;Id=cfgstore-primary;Secret=cm-appconfig-cfgstore-primary" {
		t.Fatalf("unexpected primary connection string: %v", primary["connectionString"])
	}
	readOnly := keys[2].(map[string]any)
	if readOnly["name"] != "Primary Read Only" || readOnly["readOnly"] != true {
		t.Fatalf("expected third key to be read-only Primary, got %v", readOnly)
	}
}

func TestKeyValueDataPlaneLifecycleFilteringAndETags(t *testing.T) {
	svc := New()

	kvURL := "https://cfgstore.azconfig.io/kv/app:message?api-version=2024-09-01&label=prod"
	payload := []byte(`{"value":"hello","content_type":"text/plain","tags":{"env":"prod","tier":"web"}}`)
	putResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, payload))
	if err != nil {
		t.Fatalf("put key-value returned error: %v", err)
	}
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected put status 200, got %d; body=%s", putResp.StatusCode, string(putResp.RawBody))
	}
	created := decodeAppConfigResponse(t, putResp)
	if created["key"] != "app:message" || created["label"] != "prod" || created["value"] != "hello" || created["content_type"] != "text/plain" {
		t.Fatalf("unexpected key-value shape: %v", created)
	}
	if created["etag"] == "" || putResp.Headers["ETag"] == "" || putResp.Headers["Sync-Token"] == "" {
		t.Fatalf("expected ETag and Sync-Token headers, body=%v headers=%v", created, putResp.Headers)
	}
	etag := created["etag"].(string)

	getResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, kvURL, nil))
	if err != nil {
		t.Fatalf("get key-value returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	got := decodeAppConfigResponse(t, getResp)
	if got["etag"] != etag {
		t.Fatalf("expected get to preserve etag %q, got %v", etag, got["etag"])
	}

	updateReq := appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"updated","tags":{"env":"prod","tier":"web"}}`))
	updateReq.RawRequest.Header.Set("If-Match", etag)
	updateResp, err := svc.HandleRequest(updateReq)
	if err != nil {
		t.Fatalf("conditional update returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeAppConfigResponse(t, updateResp)
	if updated["value"] != "updated" || updated["etag"] == etag {
		t.Fatalf("expected updated value and changed etag, got %v", updated)
	}

	staleReq := appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"stale"}`))
	staleReq.RawRequest.Header.Set("If-Match", etag)
	staleResp, err := svc.HandleRequest(staleReq)
	if err != nil {
		t.Fatalf("stale update returned error: %v", err)
	}
	if staleResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stale update status 412, got %d; body=%s", staleResp.StatusCode, string(staleResp.RawBody))
	}

	listURL := "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=app:*&label=prod&tags=env=prod&tags=tier=web"
	listResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list key-values returned error: %v", err)
	}
	listed := decodeAppConfigResponse(t, listResp)
	items := listed["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one filtered key-value, got %v", listed)
	}
	if items[0].(map[string]any)["key"] != "app:message" {
		t.Fatalf("unexpected listed key-value: %v", items[0])
	}

	deleteReq := appConfigCtx(t, http.MethodDelete, kvURL, nil)
	deleteReq.RawRequest.Header.Set("If-Match", updated["etag"].(string))
	deleteResp, err := svc.HandleRequest(deleteReq)
	if err != nil {
		t.Fatalf("delete key-value returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	missingResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, kvURL, nil))
	if err != nil {
		t.Fatalf("get deleted key-value returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected get deleted status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestKeyValueMutationsHonorIfNoneMatchPreconditions(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:conditional?api-version=2024-09-01&label=prod"

	createOnly := appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"first"}`))
	createOnly.RawRequest.Header.Set("If-None-Match", "*")
	createResp, err := svc.HandleRequest(createOnly)
	if err != nil {
		t.Fatalf("create-only key-value returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create-only put status 200, got %d body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeAppConfigResponse(t, createResp)
	createdETag := created["etag"].(string)

	duplicateCreate := appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"duplicate"}`))
	duplicateCreate.RawRequest.Header.Set("If-None-Match", "*")
	duplicateResp, err := svc.HandleRequest(duplicateCreate)
	if err != nil {
		t.Fatalf("duplicate create-only key-value returned error: %v", err)
	}
	if duplicateResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected duplicate create-only put status 412, got %d body=%s", duplicateResp.StatusCode, string(duplicateResp.RawBody))
	}

	matchingNoneMatch := appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"blocked"}`))
	matchingNoneMatch.RawRequest.Header.Set("If-None-Match", quoteETag(createdETag))
	matchingResp, err := svc.HandleRequest(matchingNoneMatch)
	if err != nil {
		t.Fatalf("matching If-None-Match update returned error: %v", err)
	}
	if matchingResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected matching If-None-Match update status 412, got %d body=%s", matchingResp.StatusCode, string(matchingResp.RawBody))
	}

	staleNoneMatch := appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"updated"}`))
	staleNoneMatch.RawRequest.Header.Set("If-None-Match", `"stale"`)
	updateResp, err := svc.HandleRequest(staleNoneMatch)
	if err != nil {
		t.Fatalf("stale If-None-Match update returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stale If-None-Match update status 200, got %d body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeAppConfigResponse(t, updateResp)
	if updated["value"] != "updated" {
		t.Fatalf("expected stale If-None-Match update to change value, got %v", updated)
	}
	updatedETag := updated["etag"].(string)

	blockedDelete := appConfigCtx(t, http.MethodDelete, kvURL, nil)
	blockedDelete.RawRequest.Header.Set("If-None-Match", quoteETag(updatedETag))
	blockedDeleteResp, err := svc.HandleRequest(blockedDelete)
	if err != nil {
		t.Fatalf("matching If-None-Match delete returned error: %v", err)
	}
	if blockedDeleteResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected matching If-None-Match delete status 412, got %d body=%s", blockedDeleteResp.StatusCode, string(blockedDeleteResp.RawBody))
	}

	deleteReq := appConfigCtx(t, http.MethodDelete, kvURL, nil)
	deleteReq.RawRequest.Header.Set("If-None-Match", `"stale"`)
	deleteResp, err := svc.HandleRequest(deleteReq)
	if err != nil {
		t.Fatalf("stale If-None-Match delete returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stale If-None-Match delete status 200, got %d body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	missingDelete, err := svc.HandleRequest(appConfigCtx(t, http.MethodDelete, kvURL, nil))
	if err != nil {
		t.Fatalf("missing key delete returned error: %v", err)
	}
	if missingDelete.StatusCode != http.StatusNoContent || len(missingDelete.RawBody) != 0 {
		t.Fatalf("expected missing key delete to return empty 204, got status=%d body=%s", missingDelete.StatusCode, string(missingDelete.RawBody))
	}
}

func TestKeyValueGetHonorsConditionalHeaders(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:get-conditional?api-version=2024-09-01&label=prod"

	createResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"readable"}`)))
	if err != nil {
		t.Fatalf("create key-value returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create status 200, got %d body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeAppConfigResponse(t, createResp)
	createdETag := created["etag"].(string)

	staleMatch := appConfigCtx(t, http.MethodGet, kvURL, nil)
	staleMatch.RawRequest.Header.Set("If-Match", `"stale"`)
	staleMatchResp, err := svc.HandleRequest(staleMatch)
	if err != nil {
		t.Fatalf("stale If-Match get returned error: %v", err)
	}
	if staleMatchResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stale If-Match get status 412, got %d body=%s", staleMatchResp.StatusCode, string(staleMatchResp.RawBody))
	}

	matchingMatch := appConfigCtx(t, http.MethodGet, kvURL, nil)
	matchingMatch.RawRequest.Header.Set("If-Match", quoteETag(createdETag))
	matchingMatchResp, err := svc.HandleRequest(matchingMatch)
	if err != nil {
		t.Fatalf("matching If-Match get returned error: %v", err)
	}
	if matchingMatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected matching If-Match get status 200, got %d body=%s", matchingMatchResp.StatusCode, string(matchingMatchResp.RawBody))
	}
	if matchingMatchResp.Headers["ETag"] != quoteETag(createdETag) {
		t.Fatalf("expected matching If-Match get ETag %q, got %q", quoteETag(createdETag), matchingMatchResp.Headers["ETag"])
	}
	matchingBody := decodeAppConfigResponse(t, matchingMatchResp)
	if matchingBody["value"] != "readable" {
		t.Fatalf("expected matching If-Match get body to include value, got %v", matchingBody)
	}

	notModified := appConfigCtx(t, http.MethodGet, kvURL, nil)
	notModified.RawRequest.Header.Set("If-None-Match", quoteETag(createdETag))
	notModifiedResp, err := svc.HandleRequest(notModified)
	if err != nil {
		t.Fatalf("matching If-None-Match get returned error: %v", err)
	}
	if notModifiedResp.StatusCode != http.StatusNotModified || len(notModifiedResp.RawBody) != 0 {
		t.Fatalf("expected matching If-None-Match get empty 304, got status=%d body=%s", notModifiedResp.StatusCode, string(notModifiedResp.RawBody))
	}
	if notModifiedResp.Headers["ETag"] != quoteETag(createdETag) {
		t.Fatalf("expected matching If-None-Match get ETag %q, got %q", quoteETag(createdETag), notModifiedResp.Headers["ETag"])
	}
}

func TestKeyValueLockMutationsHonorIfNoneMatchPreconditions(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:lock-if-none-match?api-version=2024-09-01&label=prod"
	lockURL := "https://cfgstore.azconfig.io/locks/app:lock-if-none-match?api-version=2024-09-01&label=prod"

	createResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"lockable"}`)))
	if err != nil {
		t.Fatalf("create key-value returned error: %v", err)
	}
	created := decodeAppConfigResponse(t, createResp)
	createdETag := created["etag"].(string)

	blockedLock := appConfigCtx(t, http.MethodPut, lockURL, nil)
	blockedLock.RawRequest.Header.Set("If-None-Match", quoteETag(createdETag))
	blockedLockResp, err := svc.HandleRequest(blockedLock)
	if err != nil {
		t.Fatalf("matching If-None-Match lock returned error: %v", err)
	}
	if blockedLockResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected matching If-None-Match lock status 412, got %d body=%s", blockedLockResp.StatusCode, string(blockedLockResp.RawBody))
	}

	lockReq := appConfigCtx(t, http.MethodPut, lockURL, nil)
	lockReq.RawRequest.Header.Set("If-None-Match", `"stale"`)
	lockResp, err := svc.HandleRequest(lockReq)
	if err != nil {
		t.Fatalf("stale If-None-Match lock returned error: %v", err)
	}
	if lockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stale If-None-Match lock status 200, got %d body=%s", lockResp.StatusCode, string(lockResp.RawBody))
	}
	locked := decodeAppConfigResponse(t, lockResp)
	if locked["locked"] != true {
		t.Fatalf("expected stale If-None-Match lock to set locked=true, got %v", locked)
	}
	lockedETag := locked["etag"].(string)

	blockedUnlock := appConfigCtx(t, http.MethodDelete, lockURL, nil)
	blockedUnlock.RawRequest.Header.Set("If-None-Match", quoteETag(lockedETag))
	blockedUnlockResp, err := svc.HandleRequest(blockedUnlock)
	if err != nil {
		t.Fatalf("matching If-None-Match unlock returned error: %v", err)
	}
	if blockedUnlockResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected matching If-None-Match unlock status 412, got %d body=%s", blockedUnlockResp.StatusCode, string(blockedUnlockResp.RawBody))
	}

	unlockReq := appConfigCtx(t, http.MethodDelete, lockURL, nil)
	unlockReq.RawRequest.Header.Set("If-None-Match", `"stale"`)
	unlockResp, err := svc.HandleRequest(unlockReq)
	if err != nil {
		t.Fatalf("stale If-None-Match unlock returned error: %v", err)
	}
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stale If-None-Match unlock status 200, got %d body=%s", unlockResp.StatusCode, string(unlockResp.RawBody))
	}
	unlocked := decodeAppConfigResponse(t, unlockResp)
	if unlocked["locked"] != false {
		t.Fatalf("expected stale If-None-Match unlock to set locked=false, got %v", unlocked)
	}
}

func TestKeyValueListPaginatesWithNextLink(t *testing.T) {
	svc := New()

	for i := 0; i < 101; i++ {
		key := fmt.Sprintf("page:%03d", i)
		_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/"+url.PathEscape(key)+"?api-version=2024-09-01", []byte(`{"value":"v"}`)))
		if err != nil {
			t.Fatalf("set key-value %s returned error: %v", key, err)
		}
	}

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=page:*", nil))
	if err != nil {
		t.Fatalf("list first key-value page returned error: %v", err)
	}
	firstBody := decodeAppConfigResponse(t, firstResp)
	firstItems := firstBody["items"].([]any)
	if len(firstItems) != 100 {
		t.Fatalf("expected first key-value page to contain 100 items, got %d", len(firstItems))
	}
	nextLink, ok := firstBody["@nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected first key-value page to include @nextLink, got %v", firstBody)
	}
	if firstResp.Headers["Link"] != "<"+nextLink+">; rel=\"next\"" {
		t.Fatalf("expected Link header to point at next page %q, got %q", nextLink, firstResp.Headers["Link"])
	}
	if firstItems[0].(map[string]any)["key"] != "page:000" || firstItems[99].(map[string]any)["key"] != "page:099" {
		t.Fatalf("unexpected first key-value page bounds: first=%v last=%v", firstItems[0], firstItems[99])
	}

	secondResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+nextLink, nil))
	if err != nil {
		t.Fatalf("list second key-value page returned error: %v", err)
	}
	secondBody := decodeAppConfigResponse(t, secondResp)
	secondItems := secondBody["items"].([]any)
	if len(secondItems) != 1 || secondItems[0].(map[string]any)["key"] != "page:100" {
		t.Fatalf("unexpected second key-value page: %v", secondItems)
	}
	if _, ok := secondBody["@nextLink"]; ok {
		t.Fatalf("did not expect @nextLink on final key-value page: %v", secondBody)
	}
	if secondResp.Headers["Link"] != "" {
		t.Fatalf("did not expect Link header on final key-value page, got %q", secondResp.Headers["Link"])
	}
}

func TestKeyValueListHonorsConditionalHeaders(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01&label=prod", []byte(`{"value":"one"}`)))
	if err != nil {
		t.Fatalf("put first key-value returned error: %v", err)
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:two?api-version=2024-09-01&label=prod", []byte(`{"value":"two"}`)))
	if err != nil {
		t.Fatalf("put second key-value returned error: %v", err)
	}

	listURL := "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=app:*&label=prod"
	listResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list key-values returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected initial list status 200, got %d body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listETag := listResp.Headers["ETag"]
	if listETag == "" {
		t.Fatalf("expected list response to include ETag header, got headers=%v", listResp.Headers)
	}

	notModifiedReq := appConfigCtx(t, http.MethodGet, listURL, nil)
	notModifiedReq.RawRequest.Header.Set("If-None-Match", listETag)
	notModifiedResp, err := svc.HandleRequest(notModifiedReq)
	if err != nil {
		t.Fatalf("conditional list If-None-Match returned error: %v", err)
	}
	if notModifiedResp.StatusCode != http.StatusNotModified || len(notModifiedResp.RawBody) != 0 {
		t.Fatalf("expected matching If-None-Match list to return empty 304, got status=%d body=%s", notModifiedResp.StatusCode, string(notModifiedResp.RawBody))
	}
	if notModifiedResp.Headers["ETag"] != listETag {
		t.Fatalf("expected 304 list to echo ETag %q, got headers=%v", listETag, notModifiedResp.Headers)
	}

	staleMatchReq := appConfigCtx(t, http.MethodGet, listURL, nil)
	staleMatchReq.RawRequest.Header.Set("If-Match", `"stale"`)
	staleMatchResp, err := svc.HandleRequest(staleMatchReq)
	if err != nil {
		t.Fatalf("conditional list stale If-Match returned error: %v", err)
	}
	if staleMatchResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stale If-Match list status 412, got %d body=%s", staleMatchResp.StatusCode, string(staleMatchResp.RawBody))
	}
}

func TestKeyValueListAcceptDatetimeReturnsHistoricalRepresentation(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:time-travel?api-version=2024-09-01&label=prod"

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"first","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("put first key-value returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first put status 200, got %d body=%s", firstResp.StatusCode, string(firstResp.RawBody))
	}
	first := decodeAppConfigResponse(t, firstResp)
	firstModified, err := time.Parse(time.RFC3339Nano, first["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse first last_modified: %v", err)
	}

	updateResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"second","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("put second key-value returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected second put status 200, got %d body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}

	listReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=app:*&label=prod&tags=env=prod", nil)
	listReq.RawRequest.Header.Set("Accept-Datetime", firstModified.Format(http.TimeFormat))
	listResp, err := svc.HandleRequest(listReq)
	if err != nil {
		t.Fatalf("historical key-value list returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected historical list status 200, got %d body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	if listResp.Headers["Memento-Datetime"] != firstModified.Format(http.TimeFormat) {
		t.Fatalf("expected Memento-Datetime %q, got headers=%v", firstModified.Format(http.TimeFormat), listResp.Headers)
	}
	if !strings.HasPrefix(listResp.Headers["Link"], "</kv?") || !strings.Contains(listResp.Headers["Link"], ">; rel=\"original\"") {
		t.Fatalf("expected original Link header for historical list, got %q", listResp.Headers["Link"])
	}
	items := decodeAppConfigResponse(t, listResp)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one historical key-value, got %v", items)
	}
	item := items[0].(map[string]any)
	if item["value"] != "first" || item["etag"] != first["etag"] {
		t.Fatalf("expected historical list to return first revision, got %v first=%v", item, first)
	}
}

func TestKeyValueResponsesUseHTTPDateLastModifiedHeaders(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:last-modified?api-version=2024-09-01&label=prod"
	lockURL := "https://cfgstore.azconfig.io/locks/app:last-modified?api-version=2024-09-01&label=prod"

	putResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"created"}`)))
	if err != nil {
		t.Fatalf("put key-value returned error: %v", err)
	}
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected put status 200, got %d body=%s", putResp.StatusCode, string(putResp.RawBody))
	}
	putBody := decodeAppConfigResponse(t, putResp)
	putModified, err := time.Parse(time.RFC3339Nano, putBody["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse put last_modified: %v", err)
	}
	if putResp.Headers["Last-Modified"] != putModified.UTC().Format(http.TimeFormat) {
		t.Fatalf("expected put Last-Modified %q, got headers=%v", putModified.UTC().Format(http.TimeFormat), putResp.Headers)
	}

	getResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, kvURL, nil))
	if err != nil {
		t.Fatalf("get key-value returned error: %v", err)
	}
	if getResp.Headers["Last-Modified"] != putModified.UTC().Format(http.TimeFormat) {
		t.Fatalf("expected get Last-Modified %q, got headers=%v", putModified.UTC().Format(http.TimeFormat), getResp.Headers)
	}

	lockResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, lockURL, nil))
	if err != nil {
		t.Fatalf("lock key-value returned error: %v", err)
	}
	lockBody := decodeAppConfigResponse(t, lockResp)
	lockModified, err := time.Parse(time.RFC3339Nano, lockBody["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse lock last_modified: %v", err)
	}
	if lockResp.Headers["Last-Modified"] != lockModified.UTC().Format(http.TimeFormat) {
		t.Fatalf("expected lock Last-Modified %q, got headers=%v", lockModified.UTC().Format(http.TimeFormat), lockResp.Headers)
	}

	unlockResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodDelete, lockURL, nil))
	if err != nil {
		t.Fatalf("unlock key-value returned error: %v", err)
	}
	unlockBody := decodeAppConfigResponse(t, unlockResp)
	unlockModified, err := time.Parse(time.RFC3339Nano, unlockBody["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse unlock last_modified: %v", err)
	}
	if unlockResp.Headers["Last-Modified"] != unlockModified.UTC().Format(http.TimeFormat) {
		t.Fatalf("expected unlock Last-Modified %q, got headers=%v", unlockModified.UTC().Format(http.TimeFormat), unlockResp.Headers)
	}

	deleteResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodDelete, kvURL, nil))
	if err != nil {
		t.Fatalf("delete key-value returned error: %v", err)
	}
	if deleteResp.Headers["Last-Modified"] != unlockModified.UTC().Format(http.TimeFormat) {
		t.Fatalf("expected delete Last-Modified %q, got headers=%v", unlockModified.UTC().Format(http.TimeFormat), deleteResp.Headers)
	}
}

func TestKeysListAcceptDatetimeReturnsHistoricalNames(t *testing.T) {
	svc := New()

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:historical-key?api-version=2024-09-01&label=prod", []byte(`{"value":"first"}`)))
	if err != nil {
		t.Fatalf("put first key returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first put status 200, got %d body=%s", firstResp.StatusCode, string(firstResp.RawBody))
	}
	first := decodeAppConfigResponse(t, firstResp)
	firstModified, err := time.Parse(time.RFC3339Nano, first["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse first last_modified: %v", err)
	}

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/other:current-key?api-version=2024-09-01&label=prod", []byte(`{"value":"later"}`))); err != nil {
		t.Fatalf("put later key returned error: %v", err)
	}

	listReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/keys?api-version=2024-09-01&name=app:*", nil)
	listReq.RawRequest.Header.Set("Accept-Datetime", firstModified.Format(http.TimeFormat))
	listResp, err := svc.HandleRequest(listReq)
	if err != nil {
		t.Fatalf("historical keys list returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected historical keys status 200, got %d body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	if listResp.Headers["Memento-Datetime"] != firstModified.Format(http.TimeFormat) {
		t.Fatalf("expected Memento-Datetime %q, got headers=%v", firstModified.Format(http.TimeFormat), listResp.Headers)
	}
	if !strings.Contains(listResp.Headers["Link"], ">; rel=\"original\"") {
		t.Fatalf("expected original Link header for historical keys, got %q", listResp.Headers["Link"])
	}
	items := decodeAppConfigResponse(t, listResp)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["name"] != "app:historical-key" {
		t.Fatalf("expected only historical app key, got %v", items)
	}
}

func TestLabelsListAcceptDatetimeReturnsHistoricalNames(t *testing.T) {
	svc := New()

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:historical-label?api-version=2024-09-01&label=prod", []byte(`{"value":"first"}`)))
	if err != nil {
		t.Fatalf("put first label returned error: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first put status 200, got %d body=%s", firstResp.StatusCode, string(firstResp.RawBody))
	}
	first := decodeAppConfigResponse(t, firstResp)
	firstModified, err := time.Parse(time.RFC3339Nano, first["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse first last_modified: %v", err)
	}

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:current-label?api-version=2024-09-01&label=preview", []byte(`{"value":"later"}`))); err != nil {
		t.Fatalf("put later label returned error: %v", err)
	}

	listReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/labels?api-version=2024-09-01&name=pr*", nil)
	listReq.RawRequest.Header.Set("Accept-Datetime", firstModified.Format(http.TimeFormat))
	listResp, err := svc.HandleRequest(listReq)
	if err != nil {
		t.Fatalf("historical labels list returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected historical labels status 200, got %d body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	if listResp.Headers["Memento-Datetime"] != firstModified.Format(http.TimeFormat) {
		t.Fatalf("expected Memento-Datetime %q, got headers=%v", firstModified.Format(http.TimeFormat), listResp.Headers)
	}
	if !strings.Contains(listResp.Headers["Link"], ">; rel=\"original\"") {
		t.Fatalf("expected original Link header for historical labels, got %q", listResp.Headers["Link"])
	}
	items := decodeAppConfigResponse(t, listResp)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["name"] != "prod" {
		t.Fatalf("expected only historical prod label, got %v", items)
	}
}

func TestKeysAndLabelsListPaginationReturnsNextLink(t *testing.T) {
	svc := New()

	for i := 0; i < 101; i++ {
		targetURL := fmt.Sprintf("https://cfgstore.azconfig.io/kv/page:key:%03d?api-version=2024-09-01&label=label-%03d", i, i)
		resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, targetURL, []byte(`{"value":"v"}`)))
		if err != nil {
			t.Fatalf("put paged key %d returned error: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected put paged key %d status 200, got %d body=%s", i, resp.StatusCode, string(resp.RawBody))
		}
	}

	keyFirst, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/keys?api-version=2024-09-01&name=page:*", nil))
	if err != nil {
		t.Fatalf("list first keys page returned error: %v", err)
	}
	keyFirstBody := decodeAppConfigResponse(t, keyFirst)
	keyFirstItems := keyFirstBody["items"].([]any)
	if len(keyFirstItems) != 100 {
		t.Fatalf("expected first keys page size 100, got %d", len(keyFirstItems))
	}
	keyNextLink, ok := keyFirstBody["@nextLink"].(string)
	if !ok || keyNextLink == "" || !strings.Contains(keyFirst.Headers["Link"], `rel="next"`) {
		t.Fatalf("expected keys nextLink body and Link header, body=%v headers=%v", keyFirstBody, keyFirst.Headers)
	}
	keySecond, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+keyNextLink, nil))
	if err != nil {
		t.Fatalf("list second keys page returned error: %v", err)
	}
	keySecondItems := decodeAppConfigResponse(t, keySecond)["items"].([]any)
	if len(keySecondItems) != 1 || keySecondItems[0].(map[string]any)["name"] != "page:key:100" {
		t.Fatalf("unexpected second keys page: %v", keySecondItems)
	}

	labelFirst, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/labels?api-version=2024-09-01&name=label-*", nil))
	if err != nil {
		t.Fatalf("list first labels page returned error: %v", err)
	}
	labelFirstBody := decodeAppConfigResponse(t, labelFirst)
	labelFirstItems := labelFirstBody["items"].([]any)
	if len(labelFirstItems) != 100 {
		t.Fatalf("expected first labels page size 100, got %d", len(labelFirstItems))
	}
	labelNextLink, ok := labelFirstBody["@nextLink"].(string)
	if !ok || labelNextLink == "" || !strings.Contains(labelFirst.Headers["Link"], `rel="next"`) {
		t.Fatalf("expected labels nextLink body and Link header, body=%v headers=%v", labelFirstBody, labelFirst.Headers)
	}
	labelSecond, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+labelNextLink, nil))
	if err != nil {
		t.Fatalf("list second labels page returned error: %v", err)
	}
	labelSecondItems := decodeAppConfigResponse(t, labelSecond)["items"].([]any)
	if len(labelSecondItems) != 1 || labelSecondItems[0].(map[string]any)["name"] != "label-100" {
		t.Fatalf("unexpected second labels page: %v", labelSecondItems)
	}
}

func TestKeysAndLabelsListNameFilterSupportsEscapedReservedCharacters(t *testing.T) {
	svc := New()

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/"+url.PathEscape("app,key")+"?api-version=2024-09-01&label="+url.QueryEscape("prod*"), []byte(`{"value":"escaped"}`))); err != nil {
		t.Fatalf("put escaped key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app-other?api-version=2024-09-01&label=prod-a", []byte(`{"value":"other"}`))); err != nil {
		t.Fatalf("put other key-value returned error: %v", err)
	}

	keysResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/keys?api-version=2024-09-01&name="+url.QueryEscape(`app\,key`), nil))
	if err != nil {
		t.Fatalf("list keys with escaped comma returned error: %v", err)
	}
	keys := decodeAppConfigResponse(t, keysResp)["items"].([]any)
	if len(keys) != 1 || keys[0].(map[string]any)["name"] != "app,key" {
		t.Fatalf("expected escaped comma key filter to match only app,key, got %v", keys)
	}

	labelsResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/labels?api-version=2024-09-01&name="+url.QueryEscape(`prod\*`), nil))
	if err != nil {
		t.Fatalf("list labels with escaped star returned error: %v", err)
	}
	labels := decodeAppConfigResponse(t, labelsResp)["items"].([]any)
	if len(labels) != 1 || labels[0].(map[string]any)["name"] != "prod*" {
		t.Fatalf("expected escaped star label filter to match only prod*, got %v", labels)
	}
}

func TestKeyValueListFiltersSupportEscapedReservedCharactersAndLabelWildcards(t *testing.T) {
	svc := New()

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/"+url.PathEscape("app,key")+"?api-version=2024-09-01&label="+url.QueryEscape("prod*"), []byte(`{"value":"literal"}`))); err != nil {
		t.Fatalf("put literal-filter key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app-other?api-version=2024-09-01&label=prod-a", []byte(`{"value":"wildcard-label"}`))); err != nil {
		t.Fatalf("put wildcard label key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app-key?api-version=2024-09-01&label=dev", []byte(`{"value":"excluded"}`))); err != nil {
		t.Fatalf("put excluded key-value returned error: %v", err)
	}

	escapedResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key="+url.QueryEscape(`app\,key`)+"&label="+url.QueryEscape(`prod\*`), nil))
	if err != nil {
		t.Fatalf("list key-values with escaped key and label filters returned error: %v", err)
	}
	escapedItems := decodeAppConfigResponse(t, escapedResp)["items"].([]any)
	if len(escapedItems) != 1 ||
		escapedItems[0].(map[string]any)["key"] != "app,key" ||
		escapedItems[0].(map[string]any)["label"] != "prod*" {
		t.Fatalf("expected escaped key and label filters to match only literal values, got %v", escapedItems)
	}

	wildcardResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=app-other&label=prod*", nil))
	if err != nil {
		t.Fatalf("list key-values with label wildcard returned error: %v", err)
	}
	wildcardItems := decodeAppConfigResponse(t, wildcardResp)["items"].([]any)
	if len(wildcardItems) != 1 ||
		wildcardItems[0].(map[string]any)["key"] != "app-other" ||
		wildcardItems[0].(map[string]any)["label"] != "prod-a" {
		t.Fatalf("expected label wildcard filter to match prod-a only, got %v", wildcardItems)
	}
}

func TestKeyValueListTagFiltersDistinguishNullAndEmptyValues(t *testing.T) {
	svc := New()

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:null?api-version=2024-09-01", []byte(`{"value":"null","tags":{"optional":null}}`))); err != nil {
		t.Fatalf("put null-tag key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:empty?api-version=2024-09-01", []byte(`{"value":"empty","tags":{"optional":""}}`))); err != nil {
		t.Fatalf("put empty-tag key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:value?api-version=2024-09-01", []byte(`{"value":"value","tags":{"optional":"set"}}`))); err != nil {
		t.Fatalf("put value-tag key-value returned error: %v", err)
	}

	nullResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=tag:*&tags=optional=%00", nil))
	if err != nil {
		t.Fatalf("list null-tag key-values returned error: %v", err)
	}
	nullItems := decodeAppConfigResponse(t, nullResp)["items"].([]any)
	if len(nullItems) != 1 || nullItems[0].(map[string]any)["key"] != "tag:null" {
		t.Fatalf("expected null tag filter to match only tag:null, got %v", nullItems)
	}
	nullTags := nullItems[0].(map[string]any)["tags"].(map[string]any)
	if value, ok := nullTags["optional"]; !ok || value != nil {
		t.Fatalf("expected null tag value to remain null in response, got %v", nullTags)
	}

	emptyResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=tag:*&tags=optional=", nil))
	if err != nil {
		t.Fatalf("list empty-tag key-values returned error: %v", err)
	}
	emptyItems := decodeAppConfigResponse(t, emptyResp)["items"].([]any)
	if len(emptyItems) != 1 || emptyItems[0].(map[string]any)["key"] != "tag:empty" {
		t.Fatalf("expected empty tag filter to match only tag:empty, got %v", emptyItems)
	}
	emptyTags := emptyItems[0].(map[string]any)["tags"].(map[string]any)
	if emptyTags["optional"] != "" {
		t.Fatalf("expected empty tag value to remain empty string in response, got %v", emptyTags)
	}
}

func TestKeyValueListTagFiltersSupportEscapedReservedCharacters(t *testing.T) {
	svc := New()

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:escaped?api-version=2024-09-01", []byte(`{"value":"escaped","tags":{"segment":"blue,green","marker":"literal*","path":"root\\child"}}`))); err != nil {
		t.Fatalf("put escaped-tag key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:plain?api-version=2024-09-01", []byte(`{"value":"plain","tags":{"segment":"blue","marker":"literal","path":"root"}}`))); err != nil {
		t.Fatalf("put plain-tag key-value returned error: %v", err)
	}

	filterQuery := "tags=" + url.QueryEscape(`segment=blue\,green`) +
		"&tags=" + url.QueryEscape(`marker=literal\*`) +
		"&tags=" + url.QueryEscape(`path=root\\child`)
	resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=tag:*&"+filterQuery, nil))
	if err != nil {
		t.Fatalf("list key-values with escaped tag filters returned error: %v", err)
	}
	items := decodeAppConfigResponse(t, resp)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["key"] != "tag:escaped" {
		t.Fatalf("expected escaped tag filters to match only tag:escaped, got %v", items)
	}
}

func TestKeyValueListEmptyTagFilterMatchesAnyTags(t *testing.T) {
	svc := New()

	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:any-one?api-version=2024-09-01", []byte(`{"value":"one","tags":{"env":"prod"}}`))); err != nil {
		t.Fatalf("put first tagged key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/tag:any-two?api-version=2024-09-01", []byte(`{"value":"two","tags":{"tier":"web"}}`))); err != nil {
		t.Fatalf("put second tagged key-value returned error: %v", err)
	}
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/other:any?api-version=2024-09-01", []byte(`{"value":"other","tags":{"env":"prod"}}`))); err != nil {
		t.Fatalf("put other key-value returned error: %v", err)
	}

	resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&key=tag:*&tags=", nil))
	if err != nil {
		t.Fatalf("list key-values with empty tag filter returned error: %v", err)
	}
	items := decodeAppConfigResponse(t, resp)["items"].([]any)
	if len(items) != 2 ||
		items[0].(map[string]any)["key"] != "tag:any-one" ||
		items[1].(map[string]any)["key"] != "tag:any-two" {
		t.Fatalf("expected empty tag filter to match all tag:* key-values with any tags, got %v", items)
	}
}

func TestKeyValueListRejectsUnescapedReservedCharactersInTagFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name      string
		tagFilter string
	}{
		{name: "comma", tagFilter: "segment=blue,green"},
		{name: "star", tagFilter: "marker=literal*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&tags="+url.QueryEscape(tt.tagFilter), nil))
			if err != nil {
				t.Fatalf("list key-values with invalid tag filter returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid tag filter status 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" ||
				body["name"] != "tags" ||
				body["status"].(float64) != http.StatusBadRequest {
				t.Fatalf("unexpected invalid tag filter problem body: %v", body)
			}
		})
	}
}

func TestKeyValueListRejectsUnescapedMiddleStarInKeyAndLabelFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name  string
		query string
		field string
	}{
		{name: "key", query: "key=app*bad", field: "key"},
		{name: "label", query: "label=prod*bad", field: "label"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&"+tt.query, nil))
			if err != nil {
				t.Fatalf("list key-values with invalid %s filter returned error: %v", tt.field, err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid %s filter status 400, got %d body=%s", tt.field, resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" ||
				body["name"] != tt.field ||
				body["status"].(float64) != http.StatusBadRequest {
				t.Fatalf("unexpected invalid %s filter problem body: %v", tt.field, body)
			}
		})
	}
}

func TestKeysAndLabelsListRejectTooManyNameFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name string
		path string
	}{
		{name: "keys", path: "/keys"},
		{name: "labels", path: "/labels"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+tt.path+"?api-version=2024-09-01&name=a,b,c,d,e,f", nil))
			if err != nil {
				t.Fatalf("list %s returned error: %v", tt.name, err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected too many name filters status 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" || body["name"] != "name" {
				t.Fatalf("unexpected too many name filters problem body: %v", body)
			}
		})
	}
}

func TestDataPlaneRejectsInvalidAPIVersionRequests(t *testing.T) {
	svc := New()

	tests := []struct {
		name   string
		url    string
		title  string
		detail string
	}{
		{
			name:   "missing",
			url:    "https://cfgstore.azconfig.io/keys",
			title:  "API version is not specified",
			detail: "An API version is required, but was not specified.",
		},
		{
			name:   "ambiguous",
			url:    "https://cfgstore.azconfig.io/keys?api-version=2024-09-01&api-version=1.0",
			title:  "Ambiguous API version",
			detail: "The following API versions were requested: 2024-09-01, 1.0. At most, only a single API version may be specified. Please update the intended API version and retry the request.",
		},
		{
			name:   "invalid",
			url:    "https://cfgstore.azconfig.io/keys?api-version=not-a-version",
			title:  "Invalid API version",
			detail: "The HTTP resource that matches the request URI '/keys' does not support the API version 'not-a-version'.",
		},
		{
			name:   "unsupported",
			url:    "https://cfgstore.azconfig.io/keys?api-version=2023-01-01",
			title:  "Unsupported API version",
			detail: "The HTTP resource that matches the request URI '/keys' does not support the API version '2023-01-01'.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, tt.url, nil))
			if err != nil {
				t.Fatalf("versioned request returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected versioning status 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" ||
				body["title"] != tt.title ||
				body["name"] != "api-version" ||
				body["detail"] != tt.detail ||
				body["status"].(float64) != http.StatusBadRequest {
				t.Fatalf("unexpected versioning problem body: %v", body)
			}
		})
	}
}

func TestDataPlaneEchoesClientRequestIDWhenRequested(t *testing.T) {
	svc := New()

	successReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/keys?api-version=2024-09-01", nil)
	successReq.RawRequest.Header.Set("x-ms-client-request-id", "11111111-1111-1111-1111-111111111111")
	successReq.RawRequest.Header.Set("x-ms-return-client-request-id", "true")
	successResp, err := svc.HandleRequest(successReq)
	if err != nil {
		t.Fatalf("client request id success request returned error: %v", err)
	}
	if successResp.StatusCode != http.StatusOK {
		t.Fatalf("expected success status 200, got %d body=%s", successResp.StatusCode, string(successResp.RawBody))
	}
	if successResp.Headers["x-ms-client-request-id"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected echoed client request id on success response, got headers=%v", successResp.Headers)
	}

	problemReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/keys", nil)
	problemReq.RawRequest.Header.Set("x-ms-client-request-id", "22222222-2222-2222-2222-222222222222")
	problemReq.RawRequest.Header.Set("x-ms-return-client-request-id", "true")
	problemResp, err := svc.HandleRequest(problemReq)
	if err != nil {
		t.Fatalf("client request id problem request returned error: %v", err)
	}
	if problemResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected problem status 400, got %d body=%s", problemResp.StatusCode, string(problemResp.RawBody))
	}
	if problemResp.Headers["x-ms-client-request-id"] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected echoed client request id on problem response, got headers=%v", problemResp.Headers)
	}

	notRequestedReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/labels?api-version=2024-09-01", nil)
	notRequestedReq.RawRequest.Header.Set("x-ms-client-request-id", "33333333-3333-3333-3333-333333333333")
	notRequestedResp, err := svc.HandleRequest(notRequestedReq)
	if err != nil {
		t.Fatalf("non-return client request id request returned error: %v", err)
	}
	if notRequestedResp.Headers["x-ms-client-request-id"] != "" {
		t.Fatalf("did not expect client request id echo without return header, got headers=%v", notRequestedResp.Headers)
	}
}

func TestKeyValueListRejectsTooManyFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name  string
		query string
		field string
	}{
		{
			name:  "key",
			query: "key=a,b,c,d,e,f",
			field: "key",
		},
		{
			name:  "label",
			query: "label=a,b,c,d,e,f",
			field: "label",
		},
		{
			name:  "tags",
			query: "tags=a=1&tags=b=2&tags=c=3&tags=d=4&tags=e=5&tags=f=6",
			field: "tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&"+tt.query, nil))
			if err != nil {
				t.Fatalf("list key-values returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected too many %s filters status 400, got %d; body=%s", tt.field, resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" || body["name"] != tt.field {
				t.Fatalf("unexpected too many filters problem body: %v", body)
			}
		})
	}
}

func TestKeysLabelsLocksAndRevisionsDataPlane(t *testing.T) {
	svc := New()

	put := func(targetURL, payload string) map[string]any {
		t.Helper()
		resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, targetURL, []byte(payload)))
		if err != nil {
			t.Fatalf("put %s returned error: %v", targetURL, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected put %s status 200, got %d; body=%s", targetURL, resp.StatusCode, string(resp.RawBody))
		}
		return decodeAppConfigResponse(t, resp)
	}

	keyProdURL := "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01&label=prod"
	put(keyProdURL, `{"value":"v1","content_type":"text/plain","tags":{"env":"prod"}}`)
	put(keyProdURL, `{"value":"v2","content_type":"text/plain","tags":{"env":"prod"}}`)
	put("https://cfgstore.azconfig.io/kv/app:two?api-version=2024-09-01&label=dev", `{"value":"dev","tags":{"env":"dev"}}`)
	put("https://cfgstore.azconfig.io/kv/other?api-version=2024-09-01", `{"value":"default"}`)

	keysResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/keys?api-version=2024-09-01&name=app:*", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if keysResp.RawContentType != "application/vnd.microsoft.appconfig.keyset+json;charset=utf-8" {
		t.Fatalf("unexpected keys content type: %q", keysResp.RawContentType)
	}
	keys := decodeAppConfigResponse(t, keysResp)["items"].([]any)
	if len(keys) != 2 || keys[0].(map[string]any)["name"] != "app:one" || keys[1].(map[string]any)["name"] != "app:two" {
		t.Fatalf("unexpected key list: %v", keys)
	}

	labelsResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/labels?api-version=2024-09-01&name=pr*", nil))
	if err != nil {
		t.Fatalf("list labels returned error: %v", err)
	}
	if labelsResp.RawContentType != "application/vnd.microsoft.appconfig.labelset+json;charset=utf-8" {
		t.Fatalf("unexpected labels content type: %q", labelsResp.RawContentType)
	}
	labels := decodeAppConfigResponse(t, labelsResp)["items"].([]any)
	if len(labels) != 1 || labels[0].(map[string]any)["name"] != "prod" {
		t.Fatalf("unexpected label list: %v", labels)
	}

	revisionsResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=app:one&label=prod", nil))
	if err != nil {
		t.Fatalf("list revisions returned error: %v", err)
	}
	revisions := decodeAppConfigResponse(t, revisionsResp)["items"].([]any)
	if len(revisions) != 2 {
		t.Fatalf("expected two revisions, got %v", revisions)
	}
	values := map[string]bool{}
	for _, item := range revisions {
		values[item.(map[string]any)["value"].(string)] = true
	}
	if !values["v1"] || !values["v2"] {
		t.Fatalf("expected both revision values, got %v", revisions)
	}

	lockResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/locks/app:one?api-version=2024-09-01&label=prod", nil))
	if err != nil {
		t.Fatalf("lock key-value returned error: %v", err)
	}
	if lockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected lock status 200, got %d; body=%s", lockResp.StatusCode, string(lockResp.RawBody))
	}
	locked := decodeAppConfigResponse(t, lockResp)
	if locked["locked"] != true || locked["etag"] == "" {
		t.Fatalf("unexpected locked key-value: %v", locked)
	}

	lockedUpdate, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, keyProdURL, []byte(`{"value":"blocked"}`)))
	if err != nil {
		t.Fatalf("locked update returned error: %v", err)
	}
	if lockedUpdate.StatusCode != http.StatusConflict {
		t.Fatalf("expected locked update status 409, got %d; body=%s", lockedUpdate.StatusCode, string(lockedUpdate.RawBody))
	}
	lockedProblem := decodeAppConfigResponse(t, lockedUpdate)
	if lockedProblem["type"] != "https://azconfig.io/errors/key-locked" ||
		lockedProblem["title"] != "Modifying key 'app:one' is not allowed" ||
		lockedProblem["name"] != "app:one" ||
		lockedProblem["detail"] != "The key is read-only. To allow modification unlock it first." ||
		lockedProblem["status"].(float64) != http.StatusConflict {
		t.Fatalf("unexpected locked update problem body: %v", lockedProblem)
	}

	unlockResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodDelete, "https://cfgstore.azconfig.io/locks/app:one?api-version=2024-09-01&label=prod", nil))
	if err != nil {
		t.Fatalf("unlock key-value returned error: %v", err)
	}
	if unlockResp.StatusCode != http.StatusOK {
		t.Fatalf("expected unlock status 200, got %d; body=%s", unlockResp.StatusCode, string(unlockResp.RawBody))
	}
	unlocked := decodeAppConfigResponse(t, unlockResp)
	if unlocked["locked"] != false {
		t.Fatalf("expected unlocked response, got %v", unlocked)
	}
	put(keyProdURL, `{"value":"v3","content_type":"text/plain","tags":{"env":"prod"}}`)
}

func TestSnapshotsDataPlaneFreezeArchiveRecoverAndList(t *testing.T) {
	svc := New()

	settingURL := "https://cfgstore.azconfig.io/kv/snap:message?api-version=2024-09-01&label=prod"
	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, settingURL, []byte(`{"value":"before","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/snap:other?api-version=2024-09-01&label=dev", []byte(`{"value":"excluded","tags":{"env":"dev"}}`)))
	if err != nil {
		t.Fatalf("set excluded source returned error: %v", err)
	}

	createSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/snap-a?api-version=2024-09-01", []byte(`{"filters":[{"key":"snap:*","label":"prod"}],"composition_type":"key_label","retention_period":3600,"tags":{"release":"blue"}}`)))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}
	if createSnapshot.StatusCode != http.StatusCreated {
		t.Fatalf("expected create snapshot status 201, got %d; body=%s", createSnapshot.StatusCode, string(createSnapshot.RawBody))
	}
	if createSnapshot.RawContentType != "application/vnd.microsoft.appconfig.snapshot+json;charset=utf-8" {
		t.Fatalf("unexpected snapshot content type: %q", createSnapshot.RawContentType)
	}
	created := decodeAppConfigResponse(t, createSnapshot)
	if created["name"] != "snap-a" || created["status"] != "provisioning" || created["items_count"] != float64(0) || created["etag"] == "" {
		t.Fatalf("unexpected created snapshot: %v", created)
	}
	if createSnapshot.Headers["Operation-Location"] != "/operations?snapshot=snap-a&api-version=2024-09-01" {
		t.Fatalf("expected documented Operation-Location header, got %q", createSnapshot.Headers["Operation-Location"])
	}

	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, settingURL, []byte(`{"value":"after","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("update source after snapshot returned error: %v", err)
	}

	listFromSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&snapshot=snap-a", nil))
	if err != nil {
		t.Fatalf("list snapshot key-values returned error: %v", err)
	}
	snapshotItems := decodeAppConfigResponse(t, listFromSnapshot)["items"].([]any)
	if len(snapshotItems) != 1 || snapshotItems[0].(map[string]any)["value"] != "before" {
		t.Fatalf("expected frozen snapshot value, got %v", snapshotItems)
	}

	getSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots/snap-a?api-version=2024-09-01", nil))
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	if getSnapshot.Headers["Link"] != "</kv?snapshot=snap-a&api-version=2024-09-01>; rel=\"items\"" {
		t.Fatalf("expected snapshot item Link header, got %q", getSnapshot.Headers["Link"])
	}
	if got := decodeAppConfigResponse(t, getSnapshot); got["status"] != "ready" || got["name"] != "snap-a" {
		t.Fatalf("unexpected get snapshot response: %v", got)
	}

	listSnapshots, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&name=snap*", nil))
	if err != nil {
		t.Fatalf("list snapshots returned error: %v", err)
	}
	snapshots := decodeAppConfigResponse(t, listSnapshots)["items"].([]any)
	if len(snapshots) != 1 || snapshots[0].(map[string]any)["name"] != "snap-a" {
		t.Fatalf("unexpected snapshot list: %v", snapshots)
	}

	archive, err := svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/snap-a?api-version=2024-09-01", []byte(`{"status":"archived"}`)))
	if err != nil {
		t.Fatalf("archive snapshot returned error: %v", err)
	}
	archived := decodeAppConfigResponse(t, archive)
	if archived["status"] != "archived" || archived["expires"] == nil {
		t.Fatalf("unexpected archived snapshot: %v", archived)
	}

	recover, err := svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/snap-a?api-version=2024-09-01", []byte(`{"status":"ready"}`)))
	if err != nil {
		t.Fatalf("recover snapshot returned error: %v", err)
	}
	recovered := decodeAppConfigResponse(t, recover)
	if recovered["status"] != "ready" || recovered["expires"] != nil {
		t.Fatalf("unexpected recovered snapshot: %v", recovered)
	}
}

func TestSnapshotGetHonorsConditionalHeaders(t *testing.T) {
	svc := New()
	snapshotURL := "https://cfgstore.azconfig.io/snapshots/conditional-get?api-version=2024-09-01"

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:snapshot-get?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	createSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, snapshotURL, []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}
	if createSnapshot.StatusCode != http.StatusCreated {
		t.Fatalf("expected create snapshot status 201, got %d; body=%s", createSnapshot.StatusCode, string(createSnapshot.RawBody))
	}
	etag := decodeAppConfigResponse(t, createSnapshot)["etag"].(string)

	staleMatch := appConfigCtx(t, http.MethodGet, snapshotURL, nil)
	staleMatch.RawRequest.Header.Set("If-Match", `"stale"`)
	staleMatchResp, err := svc.HandleRequest(staleMatch)
	if err != nil {
		t.Fatalf("snapshot get with stale If-Match returned error: %v", err)
	}
	if staleMatchResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stale If-Match snapshot get status 412, got %d body=%s", staleMatchResp.StatusCode, string(staleMatchResp.RawBody))
	}

	matchingMatch := appConfigCtx(t, http.MethodGet, snapshotURL, nil)
	matchingMatch.RawRequest.Header.Set("If-Match", quoteETag(etag))
	matchingMatchResp, err := svc.HandleRequest(matchingMatch)
	if err != nil {
		t.Fatalf("snapshot get with matching If-Match returned error: %v", err)
	}
	if matchingMatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected matching If-Match snapshot get status 200, got %d body=%s", matchingMatchResp.StatusCode, string(matchingMatchResp.RawBody))
	}
	if matchingMatchResp.Headers["Link"] != "</kv?snapshot=conditional-get&api-version=2024-09-01>; rel=\"items\"" {
		t.Fatalf("expected snapshot get Link header, got %q", matchingMatchResp.Headers["Link"])
	}

	wildcardNoneMatch := appConfigCtx(t, http.MethodGet, snapshotURL, nil)
	wildcardNoneMatch.RawRequest.Header.Set("If-None-Match", "*")
	wildcardNoneMatchResp, err := svc.HandleRequest(wildcardNoneMatch)
	if err != nil {
		t.Fatalf("snapshot get with wildcard If-None-Match returned error: %v", err)
	}
	if wildcardNoneMatchResp.StatusCode != http.StatusNotModified || len(wildcardNoneMatchResp.RawBody) != 0 {
		t.Fatalf("expected wildcard If-None-Match snapshot get empty 304, got status=%d body=%s", wildcardNoneMatchResp.StatusCode, string(wildcardNoneMatchResp.RawBody))
	}
	if wildcardNoneMatchResp.Headers["ETag"] != quoteETag(etag) {
		t.Fatalf("expected wildcard If-None-Match snapshot get ETag %q, got %q", quoteETag(etag), wildcardNoneMatchResp.Headers["ETag"])
	}

	matchingNoneMatch := appConfigCtx(t, http.MethodGet, snapshotURL, nil)
	matchingNoneMatch.RawRequest.Header.Set("If-None-Match", quoteETag(etag))
	matchingNoneMatchResp, err := svc.HandleRequest(matchingNoneMatch)
	if err != nil {
		t.Fatalf("snapshot get with matching If-None-Match returned error: %v", err)
	}
	if matchingNoneMatchResp.StatusCode != http.StatusNotModified || len(matchingNoneMatchResp.RawBody) != 0 {
		t.Fatalf("expected matching If-None-Match snapshot get empty 304, got status=%d body=%s", matchingNoneMatchResp.StatusCode, string(matchingNoneMatchResp.RawBody))
	}
}

func TestSnapshotCreateReturnsProvisioningOperationLocationAndPoll(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:message?api-version=2024-09-01&label=prod", []byte(`{"value":"before","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}

	createSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/snap-op?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*","label":"prod"}],"composition_type":"key_label"}`)))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}
	if createSnapshot.StatusCode != http.StatusCreated {
		t.Fatalf("expected create snapshot status 201, got %d; body=%s", createSnapshot.StatusCode, string(createSnapshot.RawBody))
	}
	created := decodeAppConfigResponse(t, createSnapshot)
	if created["status"] != "provisioning" || created["items_count"] != float64(0) || created["size"] != float64(0) {
		t.Fatalf("expected create response to expose a provisioning snapshot, got %v", created)
	}
	operationLocation := createSnapshot.Headers["Operation-Location"]
	if operationLocation != "/operations?snapshot=snap-op&api-version=2024-09-01" {
		t.Fatalf("unexpected Operation-Location header: %q", operationLocation)
	}

	operationResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+operationLocation, nil))
	if err != nil {
		t.Fatalf("poll snapshot operation returned error: %v", err)
	}
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected operation poll status 200, got %d; body=%s", operationResp.StatusCode, string(operationResp.RawBody))
	}
	operation := decodeAppConfigResponse(t, operationResp)
	if operation["id"] != "snap-op" || operation["status"] != "Succeeded" || operation["error"] != nil {
		t.Fatalf("unexpected snapshot operation response: %v", operation)
	}

	getSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots/snap-op?api-version=2024-09-01", nil))
	if err != nil {
		t.Fatalf("get snapshot after operation returned error: %v", err)
	}
	got := decodeAppConfigResponse(t, getSnapshot)
	if got["status"] != "ready" || got["items_count"] != float64(1) {
		t.Fatalf("expected stored snapshot to be ready after operation, got %v", got)
	}
}

func TestSnapshotCreateRejectsInvalidFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name    string
		payload string
		field   string
	}{
		{
			name:    "missing filters",
			payload: `{}`,
			field:   "filters",
		},
		{
			name:    "too many filters",
			payload: `{"filters":[{"key":"a:*"},{"key":"b:*"},{"key":"c:*"},{"key":"d:*"}]}`,
			field:   "filters",
		},
		{
			name:    "missing filter key",
			payload: `{"filters":[{"label":"prod"}]}`,
			field:   "filters[0].key",
		},
		{
			name:    "multi label with key composition",
			payload: `{"composition_type":"key","filters":[{"key":"app:*","label":"prod,dev"}]}`,
			field:   "filters[0].label",
		},
		{
			name:    "too many tag filters",
			payload: `{"filters":[{"key":"app:*","tags":["a=1","b=2","c=3","d=4","e=5","f=6"]}],"composition_type":"key_label"}`,
			field:   "filters[0].tags",
		},
		{
			name:    "invalid composition type",
			payload: `{"filters":[{"key":"app:*"}],"composition_type":"key_value"}`,
			field:   "composition_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/"+url.PathEscape(tt.name)+"?api-version=2024-09-01", []byte(tt.payload)))
			if err != nil {
				t.Fatalf("create invalid snapshot returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid snapshot status 400, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" || body["name"] != tt.field {
				t.Fatalf("unexpected invalid snapshot problem body: %v", body)
			}
		})
	}
}

func TestSnapshotCreateFiltersItemsByTags(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01&label=prod", []byte(`{"value":"prod","tags":{"env":"prod","tier":"web"}}`)))
	if err != nil {
		t.Fatalf("set prod key-value returned error: %v", err)
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:two?api-version=2024-09-01&label=prod", []byte(`{"value":"dev","tags":{"env":"dev","tier":"web"}}`)))
	if err != nil {
		t.Fatalf("set dev key-value returned error: %v", err)
	}

	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/tagged?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*","label":"prod","tags":["env=prod","tier=web"]}],"composition_type":"key_label"}`)))
	if err != nil {
		t.Fatalf("create tagged snapshot returned error: %v", err)
	}

	listFromSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/kv?api-version=2024-09-01&snapshot=tagged", nil))
	if err != nil {
		t.Fatalf("list tagged snapshot key-values returned error: %v", err)
	}
	items := decodeAppConfigResponse(t, listFromSnapshot)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["key"] != "app:one" {
		t.Fatalf("expected snapshot tag filters to retain only app:one, got %v", items)
	}
}

func TestRevisionsListSupportsLeadingAndContainsWildcards(t *testing.T) {
	svc := New()

	putRevision := func(targetURL, value string) {
		t.Helper()
		resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, targetURL, []byte(fmt.Sprintf(`{"value":%q}`, value))))
		if err != nil {
			t.Fatalf("put revision %s returned error: %v", value, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected put revision %s status 200, got %d body=%s", value, resp.StatusCode, string(resp.RawBody))
		}
	}

	putRevision("https://cfgstore.azconfig.io/kv/prod:feature:blue?api-version=2024-09-01&label=west-prod-blue", "blue")
	putRevision("https://cfgstore.azconfig.io/kv/dev:feature:red?api-version=2024-09-01&label=east-dev", "red")
	putRevision("https://cfgstore.azconfig.io/kv/other:flag?api-version=2024-09-01&label=qa", "qa")

	assertRevisionFilter := func(name, rawKeyFilter, rawLabelFilter string) {
		t.Helper()
		listURL := "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=" + url.QueryEscape(rawKeyFilter) + "&label=" + url.QueryEscape(rawLabelFilter)
		resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, listURL, nil))
		if err != nil {
			t.Fatalf("%s revision list returned error: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s revision list status 200, got %d body=%s", name, resp.StatusCode, string(resp.RawBody))
		}
		items := decodeAppConfigResponse(t, resp)["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected %s filter to return one revision, got %v", name, items)
		}
		item := items[0].(map[string]any)
		if item["key"] != "prod:feature:blue" || item["label"] != "west-prod-blue" || item["value"] != "blue" {
			t.Fatalf("unexpected %s revision filter result: %v", name, item)
		}
	}

	assertRevisionFilter("contains", "*feature*", "*prod*")
	assertRevisionFilter("suffix", "*blue", "*prod-blue")
}

func TestRevisionsListRejectsInvalidFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name  string
		query string
		field string
	}{
		{
			name:  "too many key filters",
			query: "key=a,b,c,d,e,f",
			field: "key",
		},
		{
			name:  "too many label filters",
			query: "label=a,b,c,d,e,f",
			field: "label",
		},
		{
			name:  "too many tag filters",
			query: "tags=a=1&tags=b=2&tags=c=3&tags=d=4&tags=e=5&tags=f=6",
			field: "tags",
		},
		{
			name:  "invalid key wildcard",
			query: "key=app*bad",
			field: "key",
		},
		{
			name:  "invalid label wildcard",
			query: "label=prod*bad",
			field: "label",
		},
		{
			name:  "invalid tag comma",
			query: "tags=" + url.QueryEscape("segment=blue,green"),
			field: "tags",
		},
		{
			name:  "invalid tag star",
			query: "tags=" + url.QueryEscape("marker=literal*"),
			field: "tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&"+tt.query, nil))
			if err != nil {
				t.Fatalf("list revisions with invalid filters returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid %s filter status 400, got %d body=%s", tt.field, resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" ||
				body["name"] != tt.field ||
				body["status"].(float64) != http.StatusBadRequest {
				t.Fatalf("unexpected invalid %s filter problem body: %v", tt.field, body)
			}
		})
	}
}

func TestRevisionsListHonorsRangeHeader(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:range?api-version=2024-09-01&label=prod"

	for _, value := range []string{"oldest", "middle", "newest"} {
		resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(fmt.Sprintf(`{"value":%q}`, value))))
		if err != nil {
			t.Fatalf("put revision %s returned error: %v", value, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected put revision %s status 200, got %d body=%s", value, resp.StatusCode, string(resp.RawBody))
		}
	}

	rangeReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=app:range&label=prod", nil)
	rangeReq.RawRequest.Header.Set("Range", "items=1-2")
	rangeResp, err := svc.HandleRequest(rangeReq)
	if err != nil {
		t.Fatalf("range revision list returned error: %v", err)
	}
	if rangeResp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected range revision list status 206, got %d body=%s", rangeResp.StatusCode, string(rangeResp.RawBody))
	}
	if rangeResp.Headers["Accept-Ranges"] != "items" || rangeResp.Headers["Content-Range"] != "items 1-2/3" {
		t.Fatalf("unexpected revision range headers: %v", rangeResp.Headers)
	}
	revisions := decodeAppConfigResponse(t, rangeResp)["items"].([]any)
	if len(revisions) != 2 {
		t.Fatalf("expected two ranged revisions, got %v", revisions)
	}
	if revisions[0].(map[string]any)["value"] != "middle" || revisions[1].(map[string]any)["value"] != "oldest" {
		t.Fatalf("expected middle and oldest revisions for range 1-2, got %v", revisions)
	}

	unsatisfiableReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=app:range&label=prod", nil)
	unsatisfiableReq.RawRequest.Header.Set("Range", "items=3-4")
	unsatisfiableResp, err := svc.HandleRequest(unsatisfiableReq)
	if err != nil {
		t.Fatalf("unsatisfiable range revision list returned error: %v", err)
	}
	if unsatisfiableResp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("expected unsatisfiable range status 416, got %d body=%s", unsatisfiableResp.StatusCode, string(unsatisfiableResp.RawBody))
	}
	if unsatisfiableResp.Headers["Accept-Ranges"] != "items" || unsatisfiableResp.Headers["Content-Range"] != "items */3" {
		t.Fatalf("unexpected unsatisfiable revision range headers: %v", unsatisfiableResp.Headers)
	}
}

func TestRevisionsListPaginatesWithNextLink(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:revision-page?api-version=2024-09-01&label=prod"

	for i := 0; i < 101; i++ {
		value := fmt.Sprintf("v%03d", i)
		resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(fmt.Sprintf(`{"value":%q}`, value))))
		if err != nil {
			t.Fatalf("put paged revision %s returned error: %v", value, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected put paged revision %s status 200, got %d body=%s", value, resp.StatusCode, string(resp.RawBody))
		}
	}

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=app:revision-page&label=prod", nil))
	if err != nil {
		t.Fatalf("list first revision page returned error: %v", err)
	}
	if firstResp.Headers["Accept-Ranges"] != "items" {
		t.Fatalf("expected first revision page to include Accept-Ranges, got headers=%v", firstResp.Headers)
	}
	firstBody := decodeAppConfigResponse(t, firstResp)
	firstItems := firstBody["items"].([]any)
	if len(firstItems) != 100 {
		t.Fatalf("expected first revision page to contain 100 items, got %d", len(firstItems))
	}
	nextLink, ok := firstBody["@nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected first revision page to include @nextLink, got %v", firstBody)
	}
	if firstResp.Headers["Link"] != "<"+nextLink+">; rel=\"next\"" {
		t.Fatalf("expected first revision page Link header to point at %q, got %q", nextLink, firstResp.Headers["Link"])
	}
	if firstItems[0].(map[string]any)["value"] != "v100" || firstItems[99].(map[string]any)["value"] != "v001" {
		t.Fatalf("unexpected first revision page bounds: first=%v last=%v", firstItems[0], firstItems[99])
	}

	secondResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+nextLink, nil))
	if err != nil {
		t.Fatalf("list second revision page returned error: %v", err)
	}
	secondBody := decodeAppConfigResponse(t, secondResp)
	secondItems := secondBody["items"].([]any)
	if len(secondItems) != 1 || secondItems[0].(map[string]any)["value"] != "v000" {
		t.Fatalf("unexpected second revision page: %v", secondItems)
	}
	if _, ok := secondBody["@nextLink"]; ok {
		t.Fatalf("did not expect @nextLink on final revision page: %v", secondBody)
	}
	if secondResp.Headers["Link"] != "" {
		t.Fatalf("did not expect Link header on final revision page, got %q", secondResp.Headers["Link"])
	}
}

func TestRevisionsListAcceptDatetimeReturnsHistoricalRepresentation(t *testing.T) {
	svc := New()
	kvURL := "https://cfgstore.azconfig.io/kv/app:revision-history?api-version=2024-09-01&label=prod"

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"first","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("put first revision returned error: %v", err)
	}
	first := decodeAppConfigResponse(t, firstResp)
	secondResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"second","tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("put second revision returned error: %v", err)
	}
	second := decodeAppConfigResponse(t, secondResp)
	if _, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, kvURL, []byte(`{"value":"third","tags":{"env":"prod"}}`))); err != nil {
		t.Fatalf("put third revision returned error: %v", err)
	}
	secondModified, err := time.Parse(time.RFC3339Nano, second["last_modified"].(string))
	if err != nil {
		t.Fatalf("parse second last_modified: %v", err)
	}

	historicalReq := appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/revisions?api-version=2024-09-01&key=app:revision-history&label=prod&tags=env=prod", nil)
	historicalReq.RawRequest.Header.Set("Accept-Datetime", secondModified.Format(http.TimeFormat))
	historicalResp, err := svc.HandleRequest(historicalReq)
	if err != nil {
		t.Fatalf("historical revision list returned error: %v", err)
	}
	if historicalResp.StatusCode != http.StatusOK {
		t.Fatalf("expected historical revision list status 200, got %d body=%s", historicalResp.StatusCode, string(historicalResp.RawBody))
	}
	if historicalResp.Headers["Memento-Datetime"] != secondModified.Format(http.TimeFormat) {
		t.Fatalf("expected Memento-Datetime %q, got headers=%v", secondModified.Format(http.TimeFormat), historicalResp.Headers)
	}
	if !strings.Contains(historicalResp.Headers["Link"], ">; rel=\"original\"") {
		t.Fatalf("expected original Link header for historical revisions, got %q", historicalResp.Headers["Link"])
	}
	revisions := decodeAppConfigResponse(t, historicalResp)["items"].([]any)
	if len(revisions) != 2 {
		t.Fatalf("expected two historical revisions, got %v", revisions)
	}
	if revisions[0].(map[string]any)["value"] != "second" || revisions[0].(map[string]any)["etag"] != second["etag"] {
		t.Fatalf("expected newest historical revision to be second, got %v second=%v", revisions[0], second)
	}
	if revisions[1].(map[string]any)["value"] != "first" || revisions[1].(map[string]any)["etag"] != first["etag"] {
		t.Fatalf("expected oldest historical revision to be first, got %v first=%v", revisions[1], first)
	}
}

func TestSnapshotListNameFilterSupportsEscapedComma(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	snapshotName := "snap,comma"
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/"+url.PathEscape(snapshotName)+"?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create comma snapshot returned error: %v", err)
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/snap-other?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create other snapshot returned error: %v", err)
	}

	escapedNameFilter := url.QueryEscape(`snap\,comma`)
	listResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&name="+escapedNameFilter, nil))
	if err != nil {
		t.Fatalf("list snapshots with escaped comma returned error: %v", err)
	}
	items := decodeAppConfigResponse(t, listResp)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["name"] != snapshotName {
		t.Fatalf("expected escaped comma name filter to match only %q, got %v", snapshotName, items)
	}
}

func TestSnapshotListNameFilterSupportsEscapedStar(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	snapshotName := "snap*"
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/"+url.PathEscape(snapshotName)+"?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create star snapshot returned error: %v", err)
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/snap-other?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create other snapshot returned error: %v", err)
	}

	escapedNameFilter := url.QueryEscape(`snap\*`)
	listResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&name="+escapedNameFilter, nil))
	if err != nil {
		t.Fatalf("list snapshots with escaped star returned error: %v", err)
	}
	items := decodeAppConfigResponse(t, listResp)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["name"] != snapshotName {
		t.Fatalf("expected escaped star name filter to match only %q, got %v", snapshotName, items)
	}
}

func TestSnapshotListStatusWildcardMatchesAnyStatus(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	for _, name := range []string{"snap-ready", "snap-archived"} {
		_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/"+name+"?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
		if err != nil {
			t.Fatalf("create snapshot %s returned error: %v", name, err)
		}
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/snap-archived?api-version=2024-09-01", []byte(`{"status":"archived"}`)))
	if err != nil {
		t.Fatalf("archive snapshot returned error: %v", err)
	}

	listResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&status=*", nil))
	if err != nil {
		t.Fatalf("list snapshots with status wildcard returned error: %v", err)
	}
	items := decodeAppConfigResponse(t, listResp)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected status=* to match ready and archived snapshots, got %v", items)
	}
	if items[0].(map[string]any)["name"] != "snap-archived" || items[1].(map[string]any)["name"] != "snap-ready" {
		t.Fatalf("unexpected snapshot ordering for status=* list: %v", items)
	}
}

func TestSnapshotListRejectsTooManyNameOrStatusFilters(t *testing.T) {
	svc := New()

	tests := []struct {
		name  string
		query string
		field string
	}{
		{
			name:  "name",
			query: "name=a,b,c,d,e,f",
			field: "name",
		},
		{
			name:  "status",
			query: "status=ready,archived,failed,provisioning,unknown,extra",
			field: "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&"+tt.query, nil))
			if err != nil {
				t.Fatalf("list snapshots returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected too many %s filters status 400, got %d; body=%s", tt.field, resp.StatusCode, string(resp.RawBody))
			}
			body := decodeAppConfigResponse(t, resp)
			if body["type"] != "https://azconfig.io/errors/invalid-argument" || body["name"] != tt.field {
				t.Fatalf("unexpected too many filters problem body: %v", body)
			}
		})
	}
}

func TestSnapshotListPaginatesWithNextLink(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	for i := 0; i < 101; i++ {
		name := fmt.Sprintf("page-%03d", i)
		_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/"+name+"?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
		if err != nil {
			t.Fatalf("create snapshot %s returned error: %v", name, err)
		}
	}

	firstResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io/snapshots?api-version=2024-09-01&name=page-*&status=ready", nil))
	if err != nil {
		t.Fatalf("list first snapshot page returned error: %v", err)
	}
	firstBody := decodeAppConfigResponse(t, firstResp)
	firstItems := firstBody["items"].([]any)
	if len(firstItems) != 100 {
		t.Fatalf("expected first snapshot page to contain 100 items, got %d", len(firstItems))
	}
	nextLink, ok := firstBody["@nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected first snapshot page to include @nextLink, got %v", firstBody)
	}
	if firstResp.Headers["Link"] != "<"+nextLink+">; rel=\"next\"" {
		t.Fatalf("expected Link header to point at next page %q, got %q", nextLink, firstResp.Headers["Link"])
	}
	if firstItems[0].(map[string]any)["name"] != "page-000" || firstItems[99].(map[string]any)["name"] != "page-099" {
		t.Fatalf("unexpected first page bounds: first=%v last=%v", firstItems[0], firstItems[99])
	}

	secondResp, err := svc.HandleRequest(appConfigCtx(t, http.MethodGet, "https://cfgstore.azconfig.io"+nextLink, nil))
	if err != nil {
		t.Fatalf("list second snapshot page returned error: %v", err)
	}
	secondBody := decodeAppConfigResponse(t, secondResp)
	secondItems := secondBody["items"].([]any)
	if len(secondItems) != 1 || secondItems[0].(map[string]any)["name"] != "page-100" {
		t.Fatalf("unexpected second snapshot page: %v", secondItems)
	}
	if _, ok := secondBody["@nextLink"]; ok {
		t.Fatalf("did not expect @nextLink on final snapshot page: %v", secondBody)
	}
	if secondResp.Headers["Link"] != "" {
		t.Fatalf("did not expect Link header on final snapshot page, got %q", secondResp.Headers["Link"])
	}
}

func TestSnapshotArchiveRecoverHonorsConditionalHeaders(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	createSnapshot, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/conditional?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}
	etag := decodeAppConfigResponse(t, createSnapshot)["etag"].(string)

	staleMatch := appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/conditional?api-version=2024-09-01", []byte(`{"status":"archived"}`))
	staleMatch.RawRequest.Header.Set("If-Match", `"stale"`)
	staleMatchResp, err := svc.HandleRequest(staleMatch)
	if err != nil {
		t.Fatalf("archive snapshot with stale If-Match returned error: %v", err)
	}
	if staleMatchResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stale If-Match status 412, got %d; body=%s", staleMatchResp.StatusCode, string(staleMatchResp.RawBody))
	}

	matchingNoneMatch := appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/conditional?api-version=2024-09-01", []byte(`{"status":"archived"}`))
	matchingNoneMatch.RawRequest.Header.Set("If-None-Match", quoteETag(etag))
	matchingNoneMatchResp, err := svc.HandleRequest(matchingNoneMatch)
	if err != nil {
		t.Fatalf("archive snapshot with matching If-None-Match returned error: %v", err)
	}
	if matchingNoneMatchResp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected matching If-None-Match status 412, got %d; body=%s", matchingNoneMatchResp.StatusCode, string(matchingNoneMatchResp.RawBody))
	}

	currentMatch := appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/conditional?api-version=2024-09-01", []byte(`{"status":"archived"}`))
	currentMatch.RawRequest.Header.Set("If-Match", quoteETag(etag))
	currentMatchResp, err := svc.HandleRequest(currentMatch)
	if err != nil {
		t.Fatalf("archive snapshot with current If-Match returned error: %v", err)
	}
	if currentMatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected current If-Match archive status 200, got %d; body=%s", currentMatchResp.StatusCode, string(currentMatchResp.RawBody))
	}
	if archived := decodeAppConfigResponse(t, currentMatchResp); archived["status"] != "archived" {
		t.Fatalf("expected archive to succeed after current If-Match, got %v", archived)
	}
}

func TestSnapshotArchiveRecoverAreIdempotentForCurrentState(t *testing.T) {
	svc := New()

	_, err := svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/kv/app:one?api-version=2024-09-01", []byte(`{"value":"v1"}`)))
	if err != nil {
		t.Fatalf("set snapshot source returned error: %v", err)
	}
	_, err = svc.HandleRequest(appConfigCtx(t, http.MethodPut, "https://cfgstore.azconfig.io/snapshots/idempotent?api-version=2024-09-01", []byte(`{"filters":[{"key":"app:*"}]}`)))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}

	archiveOnce, err := svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/idempotent?api-version=2024-09-01", []byte(`{"status":"archived"}`)))
	if err != nil {
		t.Fatalf("archive snapshot returned error: %v", err)
	}
	if archiveOnce.StatusCode != http.StatusOK {
		t.Fatalf("expected initial archive status 200, got %d; body=%s", archiveOnce.StatusCode, string(archiveOnce.RawBody))
	}
	archiveTwice, err := svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/idempotent?api-version=2024-09-01", []byte(`{"status":"archived"}`)))
	if err != nil {
		t.Fatalf("idempotent archive returned error: %v", err)
	}
	if archiveTwice.StatusCode != http.StatusOK {
		t.Fatalf("expected idempotent archive status 200, got %d; body=%s", archiveTwice.StatusCode, string(archiveTwice.RawBody))
	}
	if archived := decodeAppConfigResponse(t, archiveTwice); archived["status"] != "archived" || archived["expires"] == nil {
		t.Fatalf("unexpected idempotent archive response: %v", archived)
	}

	recoverOnce, err := svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/idempotent?api-version=2024-09-01", []byte(`{"status":"ready"}`)))
	if err != nil {
		t.Fatalf("recover snapshot returned error: %v", err)
	}
	if recoverOnce.StatusCode != http.StatusOK {
		t.Fatalf("expected recover status 200, got %d; body=%s", recoverOnce.StatusCode, string(recoverOnce.RawBody))
	}
	recoverTwice, err := svc.HandleRequest(appConfigCtx(t, http.MethodPatch, "https://cfgstore.azconfig.io/snapshots/idempotent?api-version=2024-09-01", []byte(`{"status":"ready"}`)))
	if err != nil {
		t.Fatalf("idempotent recover returned error: %v", err)
	}
	if recoverTwice.StatusCode != http.StatusOK {
		t.Fatalf("expected idempotent recover status 200, got %d; body=%s", recoverTwice.StatusCode, string(recoverTwice.RawBody))
	}
	if recovered := decodeAppConfigResponse(t, recoverTwice); recovered["status"] != "ready" || recovered["expires"] != nil {
		t.Fatalf("unexpected idempotent recover response: %v", recovered)
	}
}

func TestAppConfigurationServiceKeysIncludeControlAndDataPlaneVersions(t *testing.T) {
	svc := New()

	seen := make(map[string]bool)
	for _, key := range svc.ServiceKeys() {
		seen[string(key.Provider)+"|"+key.Service+"|"+key.APIVersion] = true
	}

	for _, expected := range []string{
		"azure|Microsoft.AppConfiguration/configurationStores|2024-06-01",
		"azure|Microsoft.AppConfiguration/kv|2024-09-01",
	} {
		if !seen[expected] {
			t.Fatalf("expected service key %s", expected)
		}
	}
}

func appConfigCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: method, URL: u, Host: u.Host}
	req.Header = http.Header{"Authorization": []string{"Bearer azure-token"}}
	return &service.RequestContext{
		Region:     "eastus",
		AccountID:  "sub-1",
		Action:     method,
		RawRequest: req,
		Body:       body,
	}
}

func decodeAppConfigResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
