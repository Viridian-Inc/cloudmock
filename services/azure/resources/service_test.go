package resources_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/azure/appconfiguration"
	"github.com/Viridian-Inc/cloudmock/services/azure/containerregistry"
	"github.com/Viridian-Inc/cloudmock/services/azure/eventgrid"
	"github.com/Viridian-Inc/cloudmock/services/azure/eventhub"
	"github.com/Viridian-Inc/cloudmock/services/azure/keyvault"
	"github.com/Viridian-Inc/cloudmock/services/azure/redis"
	"github.com/Viridian-Inc/cloudmock/services/azure/resources"
	"github.com/Viridian-Inc/cloudmock/services/azure/servicebus"
	"github.com/Viridian-Inc/cloudmock/services/azure/storage"
)

func armCtx(t *testing.T, method, targetURL string, body map[string]any) *service.RequestContext {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, targetURL, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	return &service.RequestContext{
		Action:     "",
		AccountID:  "00000000-0000-0000-0000-000000000001",
		RawRequest: req,
		Body:       payload,
	}
}

func decodeARMResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	if resp.RawContentType != "application/json" {
		t.Fatalf("expected application/json response, got %q", resp.RawContentType)
	}
	var out map[string]any
	if err := json.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return out
}

func TestResourceGroupLifecycle(t *testing.T) {
	svc := resources.New()

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location":  "westus2",
		"managedBy": "/subscriptions/sub-1/resourceGroups/rg-admin/providers/Microsoft.ManagedIdentity/userAssignedIdentities/admin",
		"tags": map[string]any{
			"env": "test",
		},
	}))
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createResp.StatusCode)
	}
	created := decodeARMResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a" {
		t.Errorf("unexpected resource group id: %v", created["id"])
	}
	if created["name"] != "rg-a" {
		t.Errorf("unexpected resource group name: %v", created["name"])
	}
	if created["type"] != "Microsoft.Resources/resourceGroups" {
		t.Errorf("unexpected resource group type: %v", created["type"])
	}
	if created["location"] != "westus2" {
		t.Errorf("unexpected location: %v", created["location"])
	}
	if created["managedBy"] != "/subscriptions/sub-1/resourceGroups/rg-admin/providers/Microsoft.ManagedIdentity/userAssignedIdentities/admin" {
		t.Errorf("unexpected managedBy: %v", created["managedBy"])
	}
	properties := created["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" {
		t.Errorf("unexpected provisioning state: %v", properties["provisioningState"])
	}

	getResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getResp.StatusCode)
	}
	got := decodeARMResponse(t, getResp)
	if got["name"] != "rg-a" {
		t.Errorf("unexpected get name: %v", got["name"])
	}
	if got["managedBy"] != "/subscriptions/sub-1/resourceGroups/rg-admin/providers/Microsoft.ManagedIdentity/userAssignedIdentities/admin" {
		t.Errorf("unexpected get managedBy: %v", got["managedBy"])
	}

	listResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	listed := decodeARMResponse(t, listResp)
	value := listed["value"].([]any)
	if len(value) != 1 {
		t.Fatalf("expected one listed resource group, got %d", len(value))
	}

	deleteResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("delete returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete status 202, got %d", deleteResp.StatusCode)
	}
	asyncURL := deleteResp.Headers["Azure-AsyncOperation"]
	if asyncURL == "" {
		t.Fatalf("expected delete response to include Azure-AsyncOperation header, got %v", deleteResp.Headers)
	}
	if deleteResp.Headers["Retry-After"] == "" {
		t.Fatalf("expected delete response to include Retry-After header, got %v", deleteResp.Headers)
	}

	operationResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, asyncURL, nil))
	if err != nil {
		t.Fatalf("delete operation status returned error: %v", err)
	}
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete operation status 200, got %d", operationResp.StatusCode)
	}
	operation := decodeARMResponse(t, operationResp)
	if operation["status"] != "Succeeded" {
		t.Errorf("unexpected delete operation status: %v", operation["status"])
	}

	missingResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("missing get returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing status 404, got %d", missingResp.StatusCode)
	}
	missing := decodeARMResponse(t, missingResp)
	errorBody := missing["error"].(map[string]any)
	if errorBody["code"] != "ResourceGroupNotFound" {
		t.Errorf("unexpected error code: %v", errorBody["code"])
	}
}

func TestResourceGroupsListSupportsTagFilterTopAndNextLink(t *testing.T) {
	svc := resources.New()
	for _, item := range []struct {
		name string
		tags map[string]any
	}{
		{name: "rg-alpha", tags: map[string]any{"env": "prod"}},
		{name: "rg-beta", tags: map[string]any{"env": "dev"}},
		{name: "rg-gamma", tags: map[string]any{"env": "dev", "owner": "prod"}},
	} {
		resp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/"+item.name+"?api-version=2021-04-01", map[string]any{
			"location": "westus2",
			"tags":     item.tags,
		}))
		if err != nil {
			t.Fatalf("create resource group %s returned error: %v", item.name, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create resource group %s status 201, got %d", item.name, resp.StatusCode)
		}
	}

	filterResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups?api-version=2021-04-01&$filter=tagName%20eq%20%27env%27%20and%20tagValue%20eq%20%27prod%27", nil))
	if err != nil {
		t.Fatalf("list resource groups with tag filter returned error: %v", err)
	}
	filtered := decodeARMResponse(t, filterResp)
	filterValues := filtered["value"].([]any)
	if len(filterValues) != 1 || filterValues[0].(map[string]any)["name"] != "rg-alpha" {
		t.Fatalf("unexpected tag filtered resource groups: %v", filterValues)
	}

	pageResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups?api-version=2021-04-01&$top=1", nil))
	if err != nil {
		t.Fatalf("list resource groups with top returned error: %v", err)
	}
	page := decodeARMResponse(t, pageResp)
	pageValues := page["value"].([]any)
	if len(pageValues) != 1 {
		t.Fatalf("expected one resource group in first page, got %v", pageValues)
	}
	nextLink, ok := page["nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected resource group page to include nextLink, got %v", page)
	}

	nextResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, nextLink, nil))
	if err != nil {
		t.Fatalf("list resource groups with nextLink returned error: %v", err)
	}
	nextPage := decodeARMResponse(t, nextResp)
	nextValues := nextPage["value"].([]any)
	if len(nextValues) != 1 {
		t.Fatalf("expected one resource group in next page, got %v", nextValues)
	}
	if nextValues[0].(map[string]any)["id"] == pageValues[0].(map[string]any)["id"] {
		t.Fatalf("expected resource group nextLink to advance, first=%v next=%v", pageValues, nextValues)
	}
}

func TestResourceGroupCheckExistenceUsesHeadStatuses(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	existsResp, err := svc.HandleRequest(armCtx(t, http.MethodHead, "https://management.azure.com/subscriptions/sub-1/resourceGroups/RG-A?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("check existing resource group returned error: %v", err)
	}
	if existsResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected existing resource group check status 204, got %d", existsResp.StatusCode)
	}
	if len(existsResp.RawBody) != 0 {
		t.Fatalf("expected existing resource group check to return no body, got %q", string(existsResp.RawBody))
	}

	missingResp, err := svc.HandleRequest(armCtx(t, http.MethodHead, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-missing?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("check missing resource group returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing resource group check status 404, got %d", missingResp.StatusCode)
	}
	if len(missingResp.RawBody) != 0 {
		t.Fatalf("expected missing resource group check to return no body, got %q", string(missingResp.RawBody))
	}
}

func TestTagsCreateOrUpdateAndGetAtResourceGroupScope(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
		"tags":     map[string]any{"env": "old"},
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	tagsResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/tags/default?api-version=2021-04-01", map[string]any{
		"properties": map[string]any{
			"tags": map[string]any{
				"Environment": "Production",
				"CostCenter":  "ENG",
			},
		},
	}))
	if err != nil {
		t.Fatalf("create or update tags returned error: %v", err)
	}
	if tagsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create or update tags status 200, got %d; body=%s", tagsResp.StatusCode, string(tagsResp.RawBody))
	}
	tagsBody := decodeARMResponse(t, tagsResp)
	if tagsBody["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/tags/default" {
		t.Fatalf("unexpected tags id: %v", tagsBody["id"])
	}
	if tagsBody["name"] != "default" {
		t.Fatalf("unexpected tags name: %v", tagsBody["name"])
	}
	if tagsBody["type"] != "Microsoft.Resources/tags" {
		t.Fatalf("unexpected tags type: %v", tagsBody["type"])
	}
	properties := tagsBody["properties"].(map[string]any)
	tags := properties["tags"].(map[string]any)
	if tags["Environment"] != "Production" || tags["CostCenter"] != "ENG" {
		t.Fatalf("unexpected tags wrapper properties: %v", tags)
	}

	getTagsResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/tags/default?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get tags returned error: %v", err)
	}
	getTags := decodeARMResponse(t, getTagsResp)
	getProperties := getTags["properties"].(map[string]any)
	if getProperties["tags"].(map[string]any)["Environment"] != "Production" {
		t.Fatalf("unexpected get tags body: %v", getTags)
	}

	getResourceGroupResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get resource group returned error: %v", err)
	}
	resourceGroup := decodeARMResponse(t, getResourceGroupResp)
	resourceGroupTags := resourceGroup["tags"].(map[string]any)
	if resourceGroupTags["Environment"] != "Production" {
		t.Fatalf("expected resource group tags to be replaced, got %v", resourceGroupTags)
	}
	if _, ok := resourceGroupTags["env"]; ok {
		t.Fatalf("expected replace semantics to remove old tags, got %v", resourceGroupTags)
	}
}

func TestTagsPatchAndDeleteAtResourceGroupScope(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
		"tags": map[string]any{
			"env":   "dev",
			"owner": "platform",
		},
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	tagsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/tags/default?api-version=2021-04-01"
	mergeResp, err := svc.HandleRequest(armCtx(t, http.MethodPatch, tagsURL, map[string]any{
		"operation": "Merge",
		"properties": map[string]any{
			"tags": map[string]any{
				"env":        "prod",
				"costCenter": "eng",
			},
		},
	}))
	if err != nil {
		t.Fatalf("merge resource group tags returned error: %v", err)
	}
	if mergeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected merge tags status 200, got %d; body=%s", mergeResp.StatusCode, string(mergeResp.RawBody))
	}
	merged := decodeARMResponse(t, mergeResp)
	mergedTags := merged["properties"].(map[string]any)["tags"].(map[string]any)
	if mergedTags["env"] != "prod" || mergedTags["owner"] != "platform" || mergedTags["costCenter"] != "eng" {
		t.Fatalf("unexpected merged tags: %v", mergedTags)
	}

	deleteResp, err := svc.HandleRequest(armCtx(t, http.MethodPatch, tagsURL, map[string]any{
		"operation": "Delete",
		"properties": map[string]any{
			"tags": map[string]any{
				"owner":      "platform",
				"costCenter": "wrong-value",
			},
		},
	}))
	if err != nil {
		t.Fatalf("delete selected resource group tags returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected selected tag delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	selectedDeleted := decodeARMResponse(t, deleteResp)
	selectedDeletedTags := selectedDeleted["properties"].(map[string]any)["tags"].(map[string]any)
	if _, ok := selectedDeletedTags["owner"]; ok {
		t.Fatalf("expected owner tag to be deleted, got %v", selectedDeletedTags)
	}
	if selectedDeletedTags["costCenter"] != "eng" {
		t.Fatalf("expected mismatched costCenter value to remain, got %v", selectedDeletedTags)
	}

	replaceResp, err := svc.HandleRequest(armCtx(t, http.MethodPatch, tagsURL, map[string]any{
		"operation": "Replace",
		"properties": map[string]any{
			"tags": map[string]any{
				"only": "tag",
			},
		},
	}))
	if err != nil {
		t.Fatalf("replace resource group tags returned error: %v", err)
	}
	replaced := decodeARMResponse(t, replaceResp)
	replacedTags := replaced["properties"].(map[string]any)["tags"].(map[string]any)
	if len(replacedTags) != 1 || replacedTags["only"] != "tag" {
		t.Fatalf("unexpected replaced tags: %v", replacedTags)
	}

	clearResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, tagsURL, nil))
	if err != nil {
		t.Fatalf("delete all resource group tags returned error: %v", err)
	}
	if clearResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete all tags status 200, got %d; body=%s", clearResp.StatusCode, string(clearResp.RawBody))
	}
	if len(clearResp.RawBody) != 0 {
		t.Fatalf("expected delete all tags to return no body, got %q", string(clearResp.RawBody))
	}

	getResourceGroupResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get resource group after tag delete returned error: %v", err)
	}
	resourceGroup := decodeARMResponse(t, getResourceGroupResp)
	if _, ok := resourceGroup["tags"]; ok {
		t.Fatalf("expected resource group tags to be removed, got %v", resourceGroup["tags"])
	}
}

func TestGenericResourceLifecycleAtResourceGroupScope(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	resourceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2021-04-01"
	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, resourceURL, map[string]any{
		"location": "westus2",
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"addressSpace": map[string]any{
				"addressPrefixes": []any{"10.0.0.0/16"},
			},
		},
	}))
	if err != nil {
		t.Fatalf("create generic resource returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected generic resource create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a" {
		t.Fatalf("unexpected resource id: %v", created["id"])
	}
	if created["name"] != "vnet-a" {
		t.Fatalf("unexpected resource name: %v", created["name"])
	}
	if created["type"] != "Microsoft.Network/virtualNetworks" {
		t.Fatalf("unexpected resource type: %v", created["type"])
	}
	if created["location"] != "westus2" {
		t.Fatalf("unexpected resource location: %v", created["location"])
	}
	properties := created["properties"].(map[string]any)
	addressSpace := properties["addressSpace"].(map[string]any)
	prefixes := addressSpace["addressPrefixes"].([]any)
	if prefixes[0] != "10.0.0.0/16" {
		t.Fatalf("unexpected address prefixes: %v", prefixes)
	}

	getResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, resourceURL, nil))
	if err != nil {
		t.Fatalf("get generic resource returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected generic resource get status 200, got %d", getResp.StatusCode)
	}

	listResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list generic resources returned error: %v", err)
	}
	listed := decodeARMResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one listed generic resource, got %d", len(values))
	}
	listedResource := values[0].(map[string]any)
	if listedResource["type"] != "Microsoft.Network/virtualNetworks" {
		t.Fatalf("unexpected listed resource type: %v", listedResource["type"])
	}

	deleteResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, resourceURL, nil))
	if err != nil {
		t.Fatalf("delete generic resource returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected generic resource delete status 202, got %d", deleteResp.StatusCode)
	}
	asyncURL := deleteResp.Headers["Azure-AsyncOperation"]
	if asyncURL == "" {
		t.Fatalf("expected generic resource delete response to include Azure-AsyncOperation header, got %v", deleteResp.Headers)
	}
	if deleteResp.Headers["Retry-After"] == "" {
		t.Fatalf("expected generic resource delete response to include Retry-After header, got %v", deleteResp.Headers)
	}

	operationResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, asyncURL, nil))
	if err != nil {
		t.Fatalf("generic resource delete operation status returned error: %v", err)
	}
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected generic resource delete operation status 200, got %d", operationResp.StatusCode)
	}
	operation := decodeARMResponse(t, operationResp)
	if operation["status"] != "Succeeded" {
		t.Errorf("unexpected generic resource delete operation status: %v", operation["status"])
	}

	missingResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, resourceURL, nil))
	if err != nil {
		t.Fatalf("missing generic resource get returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing generic resource status 404, got %d", missingResp.StatusCode)
	}
}

func TestGenericResourceCheckExistenceByIDUsesHeadStatuses(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	resourceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2021-04-01"
	_, err = svc.HandleRequest(armCtx(t, http.MethodPut, resourceURL, map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create generic resource returned error: %v", err)
	}

	existsResp, err := svc.HandleRequest(armCtx(t, http.MethodHead, "https://management.azure.com/subscriptions/sub-1/resourceGroups/RG-A/providers/Microsoft.Network/virtualNetworks/VNET-A?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("check existing generic resource returned error: %v", err)
	}
	if existsResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected existing generic resource check status 204, got %d", existsResp.StatusCode)
	}
	if len(existsResp.RawBody) != 0 {
		t.Fatalf("expected existing generic resource check to return no body, got %q", string(existsResp.RawBody))
	}

	missingResp, err := svc.HandleRequest(armCtx(t, http.MethodHead, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-missing?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("check missing generic resource returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing generic resource check status 404, got %d", missingResp.StatusCode)
	}
	if len(missingResp.RawBody) != 0 {
		t.Fatalf("expected missing generic resource check to return no body, got %q", string(missingResp.RawBody))
	}
}

func TestGenericResourceListByResourceGroupSupportsFiltersTopAndNextLink(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	for _, item := range []struct {
		url  string
		body map[string]any
	}{
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/demo-vnet?api-version=2021-04-01",
			body: map[string]any{
				"location": "westus2",
				"tags":     map[string]any{"env": "prod"},
				"plan": map[string]any{
					"name":      "vnet-plan",
					"product":   "networking",
					"publisher": "canonical",
				},
			},
		},
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/demo-ip?api-version=2021-04-01",
			body: map[string]any{
				"location": "westus2",
				"tags":     map[string]any{"env": "prod"},
				"identity": map[string]any{
					"principalId": "principal-ip",
					"type":        "SystemAssigned",
				},
			},
		},
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/datastore?api-version=2021-04-01",
			body: map[string]any{
				"location": "eastus",
				"tags":     map[string]any{"env": "dev"},
			},
		},
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/mixedtags?api-version=2021-04-01",
			body: map[string]any{
				"location": "eastus",
				"tags":     map[string]any{"env": "dev", "owner": "prod"},
			},
		},
	} {
		resp, err := svc.HandleRequest(armCtx(t, http.MethodPut, item.url, item.body))
		if err != nil {
			t.Fatalf("create generic resource %s returned error: %v", item.url, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create status 201 for %s, got %d", item.url, resp.StatusCode)
		}
	}

	typeFilterResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=resourceType%20eq%20%27Microsoft.Network%2FvirtualNetworks%27", nil))
	if err != nil {
		t.Fatalf("list resources with resourceType filter returned error: %v", err)
	}
	typeFilter := decodeARMResponse(t, typeFilterResp)
	typeFilterValues := typeFilter["value"].([]any)
	if len(typeFilterValues) != 1 || typeFilterValues[0].(map[string]any)["name"] != "demo-vnet" {
		t.Fatalf("unexpected resourceType filter result: %v", typeFilterValues)
	}

	notEqualResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=location%20ne%20%27eastus%27", nil))
	if err != nil {
		t.Fatalf("list resources with ne filter returned error: %v", err)
	}
	notEqual := decodeARMResponse(t, notEqualResp)
	notEqualValues := notEqual["value"].([]any)
	if len(notEqualValues) != 2 {
		t.Fatalf("expected two resources outside eastus, got %v", notEqualValues)
	}
	for _, value := range notEqualValues {
		if value.(map[string]any)["location"] == "eastus" {
			t.Fatalf("expected ne filter to exclude eastus resources, got %v", notEqualValues)
		}
	}

	planResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=plan%2Fpublisher%20eq%20%27canonical%27", nil))
	if err != nil {
		t.Fatalf("list resources with plan filter returned error: %v", err)
	}
	plan := decodeARMResponse(t, planResp)
	planValues := plan["value"].([]any)
	if len(planValues) != 1 || planValues[0].(map[string]any)["name"] != "demo-vnet" {
		t.Fatalf("unexpected plan filter resources: %v", planValues)
	}

	identityResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=identity%2FprincipalId%20eq%20%27principal-ip%27", nil))
	if err != nil {
		t.Fatalf("list resources with identity filter returned error: %v", err)
	}
	identity := decodeARMResponse(t, identityResp)
	identityValues := identity["value"].([]any)
	if len(identityValues) != 1 || identityValues[0].(map[string]any)["name"] != "demo-ip" {
		t.Fatalf("unexpected identity filter resources: %v", identityValues)
	}

	substringResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=substringof(%27demo%27%2C%20name)", nil))
	if err != nil {
		t.Fatalf("list resources with substring filter returned error: %v", err)
	}
	substring := decodeARMResponse(t, substringResp)
	substringValues := substring["value"].([]any)
	if len(substringValues) != 2 {
		t.Fatalf("expected two substring filter resources, got %v", substringValues)
	}

	substringOrResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=substringof(%27demo%27%2C%20name)%20or%20substringof(%27store%27%2C%20name)", nil))
	if err != nil {
		t.Fatalf("list resources with substring or filter returned error: %v", err)
	}
	substringOr := decodeARMResponse(t, substringOrResp)
	substringOrValues := substringOr["value"].([]any)
	if len(substringOrValues) != 3 {
		t.Fatalf("expected three substring or filter resources, got %v", substringOrValues)
	}
	for _, value := range substringOrValues {
		name := value.(map[string]any)["name"].(string)
		if name == "mixedtags" {
			t.Fatalf("expected substring or filter to exclude mixedtags, got %v", substringOrValues)
		}
	}

	tagResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$filter=tagName%20eq%20%27env%27%20and%20tagValue%20eq%20%27prod%27", nil))
	if err != nil {
		t.Fatalf("list resources with tag filter returned error: %v", err)
	}
	tagFiltered := decodeARMResponse(t, tagResp)
	tagValues := tagFiltered["value"].([]any)
	if len(tagValues) != 2 {
		t.Fatalf("expected two tag filtered resources, got %v", tagValues)
	}
	for _, value := range tagValues {
		if _, exists := value.(map[string]any)["tags"]; exists {
			t.Fatalf("expected tags to be omitted when filtering by tag name/value, got %v", tagValues)
		}
	}

	pageResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$top=1", nil))
	if err != nil {
		t.Fatalf("list resources with top returned error: %v", err)
	}
	page := decodeARMResponse(t, pageResp)
	pageValues := page["value"].([]any)
	if len(pageValues) != 1 {
		t.Fatalf("expected one resource in first page, got %v", pageValues)
	}
	nextLink, ok := page["nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected nextLink for truncated resource page, got %v", page)
	}

	nextResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, nextLink, nil))
	if err != nil {
		t.Fatalf("list resources with nextLink returned error: %v", err)
	}
	nextPage := decodeARMResponse(t, nextResp)
	nextValues := nextPage["value"].([]any)
	if len(nextValues) != 1 {
		t.Fatalf("expected one resource in next page, got %v", nextValues)
	}
	if nextValues[0].(map[string]any)["id"] == pageValues[0].(map[string]any)["id"] {
		t.Fatalf("expected nextLink to advance to a different resource, first=%v next=%v", pageValues, nextValues)
	}
}

func TestGenericResourceListBySubscriptionSupportsFiltersTopAndNextLink(t *testing.T) {
	svc := resources.New()
	for _, group := range []string{"rg-a", "rg-b"} {
		_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/"+group+"?api-version=2021-04-01", map[string]any{
			"location": "westus2",
		}))
		if err != nil {
			t.Fatalf("create resource group %s returned error: %v", group, err)
		}
	}

	for _, item := range []struct {
		url  string
		body map[string]any
	}{
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/demo-vnet?api-version=2021-04-01",
			body: map[string]any{
				"location": "westus2",
				"tags":     map[string]any{"department": "network", "env": "prod"},
				"plan": map[string]any{
					"name":      "vnet-plan",
					"product":   "networking",
					"publisher": "canonical",
				},
			},
		},
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Network/publicIPAddresses/demo-ip?api-version=2021-04-01",
			body: map[string]any{
				"location": "westus2",
				"tags":     map[string]any{"departureDate": "soon", "env": "prod"},
				"identity": map[string]any{
					"principalId": "principal-ip",
					"type":        "SystemAssigned",
				},
			},
		},
		{
			url: "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Storage/storageAccounts/datastore?api-version=2021-04-01",
			body: map[string]any{
				"location": "eastus",
				"tags":     map[string]any{"env": "dev"},
			},
		},
	} {
		resp, err := svc.HandleRequest(armCtx(t, http.MethodPut, item.url, item.body))
		if err != nil {
			t.Fatalf("create subscription resource %s returned error: %v", item.url, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create status 201 for %s, got %d", item.url, resp.StatusCode)
		}
	}

	filterResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$filter=resourceGroup%20eq%20%27rg-b%27%20and%20resourceType%20eq%20%27Microsoft.Network%2FpublicIPAddresses%27", nil))
	if err != nil {
		t.Fatalf("list subscription resources with filter returned error: %v", err)
	}
	if filterResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription filtered list status 200, got %d; body=%s", filterResp.StatusCode, string(filterResp.RawBody))
	}
	filtered := decodeARMResponse(t, filterResp)
	filteredValues := filtered["value"].([]any)
	if len(filteredValues) != 1 || filteredValues[0].(map[string]any)["name"] != "demo-ip" {
		t.Fatalf("unexpected subscription filtered resources: %v", filteredValues)
	}

	planResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$filter=plan%2Fpublisher%20eq%20%27canonical%27", nil))
	if err != nil {
		t.Fatalf("list subscription resources with plan filter returned error: %v", err)
	}
	plan := decodeARMResponse(t, planResp)
	planValues := plan["value"].([]any)
	if len(planValues) != 1 || planValues[0].(map[string]any)["name"] != "demo-vnet" {
		t.Fatalf("unexpected subscription plan filter resources: %v", planValues)
	}

	identityResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$filter=identity%2FprincipalId%20eq%20%27principal-ip%27", nil))
	if err != nil {
		t.Fatalf("list subscription resources with identity filter returned error: %v", err)
	}
	identity := decodeARMResponse(t, identityResp)
	identityValues := identity["value"].([]any)
	if len(identityValues) != 1 || identityValues[0].(map[string]any)["name"] != "demo-ip" {
		t.Fatalf("unexpected subscription identity filter resources: %v", identityValues)
	}

	startsWithResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$filter=startswith(tagName%2C%20%27depart%27)", nil))
	if err != nil {
		t.Fatalf("list subscription resources with startswith tag filter returned error: %v", err)
	}
	startsWith := decodeARMResponse(t, startsWithResp)
	startsWithValues := startsWith["value"].([]any)
	if len(startsWithValues) != 2 {
		t.Fatalf("expected two resources with tag-name prefix, got %v", startsWithValues)
	}
	for _, value := range startsWithValues {
		tags := value.(map[string]any)["tags"].(map[string]any)
		if _, hasDepartment := tags["department"]; !hasDepartment {
			if _, hasDeparture := tags["departureDate"]; !hasDeparture {
				t.Fatalf("expected tag-name prefix result to include matching tag, got %v", value)
			}
		}
	}

	notEqualResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$filter=resourceGroup%20ne%20%27rg-b%27", nil))
	if err != nil {
		t.Fatalf("list subscription resources with ne filter returned error: %v", err)
	}
	notEqual := decodeARMResponse(t, notEqualResp)
	notEqualValues := notEqual["value"].([]any)
	if len(notEqualValues) != 1 || notEqualValues[0].(map[string]any)["name"] != "demo-vnet" {
		t.Fatalf("unexpected subscription ne filter resources: %v", notEqualValues)
	}

	substringOrResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$filter=substringof(%27vnet%27%2C%20name)%20or%20substringof(%27store%27%2C%20name)", nil))
	if err != nil {
		t.Fatalf("list subscription resources with substring or filter returned error: %v", err)
	}
	substringOr := decodeARMResponse(t, substringOrResp)
	substringOrValues := substringOr["value"].([]any)
	if len(substringOrValues) != 2 {
		t.Fatalf("expected two subscription substring or filter resources, got %v", substringOrValues)
	}
	for _, value := range substringOrValues {
		name := value.(map[string]any)["name"].(string)
		if name == "demo-ip" {
			t.Fatalf("expected subscription substring or filter to exclude demo-ip, got %v", substringOrValues)
		}
	}

	pageResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$top=1", nil))
	if err != nil {
		t.Fatalf("list subscription resources with top returned error: %v", err)
	}
	if pageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription first page status 200, got %d; body=%s", pageResp.StatusCode, string(pageResp.RawBody))
	}
	page := decodeARMResponse(t, pageResp)
	pageValues := page["value"].([]any)
	if len(pageValues) != 1 {
		t.Fatalf("expected one subscription resource in first page, got %v", pageValues)
	}
	nextLink, ok := page["nextLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected subscription resource nextLink, got %v", page)
	}

	nextResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, nextLink, nil))
	if err != nil {
		t.Fatalf("list subscription resources with nextLink returned error: %v", err)
	}
	if nextResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription next page status 200, got %d; body=%s", nextResp.StatusCode, string(nextResp.RawBody))
	}
	nextPage := decodeARMResponse(t, nextResp)
	nextValues := nextPage["value"].([]any)
	if len(nextValues) != 1 {
		t.Fatalf("expected one subscription resource in next page, got %v", nextValues)
	}
	if nextValues[0].(map[string]any)["id"] == pageValues[0].(map[string]any)["id"] {
		t.Fatalf("expected subscription resource nextLink to advance, first=%v next=%v", pageValues, nextValues)
	}
}

func TestGenericResourceListSupportsExpandFields(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	_, err = svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/demo-vnet?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create generic resource returned error: %v", err)
	}

	plainResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list resources without expand returned error: %v", err)
	}
	plain := decodeARMResponse(t, plainResp)
	plainResource := plain["value"].([]any)[0].(map[string]any)
	for _, field := range []string{"createdTime", "changedTime", "provisioningState"} {
		if _, exists := plainResource[field]; exists {
			t.Fatalf("expected %s to be omitted without expand, got %v", field, plainResource)
		}
	}

	expandedURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01&$expand=createdTime%2CchangedTime%2CprovisioningState"
	expandedResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, expandedURL, nil))
	if err != nil {
		t.Fatalf("list resources with expand returned error: %v", err)
	}
	expanded := decodeARMResponse(t, expandedResp)
	expandedResource := expanded["value"].([]any)[0].(map[string]any)
	assertExpandedGenericResource(t, expandedResource)

	subscriptionExpandedResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resources?api-version=2021-04-01&$expand=createdTime%2CchangedTime%2CprovisioningState", nil))
	if err != nil {
		t.Fatalf("list subscription resources with expand returned error: %v", err)
	}
	subscriptionExpanded := decodeARMResponse(t, subscriptionExpandedResp)
	subscriptionResource := subscriptionExpanded["value"].([]any)[0].(map[string]any)
	assertExpandedGenericResource(t, subscriptionResource)
}

func assertExpandedGenericResource(t *testing.T, resource map[string]any) {
	t.Helper()

	createdTime, ok := resource["createdTime"].(string)
	if !ok || createdTime == "" {
		t.Fatalf("expected expanded resource to include createdTime, got %v", resource)
	}
	if _, err := time.Parse(time.RFC3339Nano, createdTime); err != nil {
		t.Fatalf("expected createdTime to be RFC3339Nano, got %q: %v", createdTime, err)
	}

	changedTime, ok := resource["changedTime"].(string)
	if !ok || changedTime == "" {
		t.Fatalf("expected expanded resource to include changedTime, got %v", resource)
	}
	if _, err := time.Parse(time.RFC3339Nano, changedTime); err != nil {
		t.Fatalf("expected changedTime to be RFC3339Nano, got %q: %v", changedTime, err)
	}

	if resource["provisioningState"] != "Succeeded" {
		t.Fatalf("expected expanded resource provisioningState Succeeded, got %v", resource)
	}
}

func TestResourceGroupDeleteCascadesGenericResourcesAndDeployments(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	resourceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2021-04-01"
	_, err = svc.HandleRequest(armCtx(t, http.MethodPut, resourceURL, map[string]any{
		"location": "westus2",
		"properties": map[string]any{
			"addressSpace": map[string]any{
				"addressPrefixes": []any{"10.0.0.0/16"},
			},
		},
	}))
	if err != nil {
		t.Fatalf("create generic resource returned error: %v", err)
	}

	deploymentURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-a?api-version=2021-04-01"
	_, err = svc.HandleRequest(armCtx(t, http.MethodPut, deploymentURL, map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"resources": []any{},
			},
		},
	}))
	if err != nil {
		t.Fatalf("create deployment returned error: %v", err)
	}

	deleteResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("delete resource group returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete resource group status 202, got %d", deleteResp.StatusCode)
	}

	getResourceResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, resourceURL, nil))
	if err != nil {
		t.Fatalf("get generic resource after resource group delete returned error: %v", err)
	}
	if getResourceResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected generic resource to be deleted with resource group, got status %d body=%s", getResourceResp.StatusCode, string(getResourceResp.RawBody))
	}

	listResourcesResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/resources?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list generic resources after resource group delete returned error: %v", err)
	}
	listResources := decodeARMResponse(t, listResourcesResp)
	if got := len(listResources["value"].([]any)); got != 0 {
		t.Fatalf("expected no generic resources after resource group delete, got %d", got)
	}

	getDeploymentResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, deploymentURL, nil))
	if err != nil {
		t.Fatalf("get deployment after resource group delete returned error: %v", err)
	}
	if getDeploymentResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deployment to be deleted with resource group, got status %d body=%s", getDeploymentResp.StatusCode, string(getDeploymentResp.RawBody))
	}

	listDeploymentsResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list deployments after resource group delete returned error: %v", err)
	}
	listDeployments := decodeARMResponse(t, listDeploymentsResp)
	if got := len(listDeployments["value"].([]any)); got != 0 {
		t.Fatalf("expected no deployments after resource group delete, got %d", got)
	}
}

func TestTagsCreateOrUpdateAndGetAtGenericResourceScope(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}
	resourceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2021-04-01"
	_, err = svc.HandleRequest(armCtx(t, http.MethodPut, resourceURL, map[string]any{
		"location": "westus2",
		"tags":     map[string]any{"env": "old"},
	}))
	if err != nil {
		t.Fatalf("create generic resource returned error: %v", err)
	}

	tagsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/providers/Microsoft.Resources/tags/default?api-version=2021-04-01"
	tagsResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, tagsURL, map[string]any{
		"properties": map[string]any{
			"tags": map[string]any{
				"Environment": "Production",
				"Owner":       "Platform",
			},
		},
	}))
	if err != nil {
		t.Fatalf("create or update resource tags returned error: %v", err)
	}
	if tagsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected resource tags status 200, got %d; body=%s", tagsResp.StatusCode, string(tagsResp.RawBody))
	}
	tagsBody := decodeARMResponse(t, tagsResp)
	if tagsBody["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/providers/Microsoft.Resources/tags/default" {
		t.Fatalf("unexpected resource tags id: %v", tagsBody["id"])
	}
	properties := tagsBody["properties"].(map[string]any)
	tags := properties["tags"].(map[string]any)
	if tags["Environment"] != "Production" || tags["Owner"] != "Platform" {
		t.Fatalf("unexpected resource tag wrapper properties: %v", tags)
	}

	getTagsResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, tagsURL, nil))
	if err != nil {
		t.Fatalf("get resource tags returned error: %v", err)
	}
	getTags := decodeARMResponse(t, getTagsResp)
	getProperties := getTags["properties"].(map[string]any)
	if getProperties["tags"].(map[string]any)["Owner"] != "Platform" {
		t.Fatalf("unexpected get resource tags body: %v", getTags)
	}

	getResourceResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, resourceURL, nil))
	if err != nil {
		t.Fatalf("get generic resource returned error: %v", err)
	}
	resource := decodeARMResponse(t, getResourceResp)
	resourceTags := resource["tags"].(map[string]any)
	if resourceTags["Environment"] != "Production" {
		t.Fatalf("expected resource tags to be replaced, got %v", resourceTags)
	}
	if _, ok := resourceTags["env"]; ok {
		t.Fatalf("expected replace semantics to remove old resource tags, got %v", resourceTags)
	}
}

func TestTagsPatchAndDeleteAtGenericResourceScope(t *testing.T) {
	svc := resources.New()
	_, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a?api-version=2021-04-01", map[string]any{
		"location": "westus2",
	}))
	if err != nil {
		t.Fatalf("create resource group returned error: %v", err)
	}

	resourceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2021-04-01"
	_, err = svc.HandleRequest(armCtx(t, http.MethodPut, resourceURL, map[string]any{
		"location": "westus2",
		"tags": map[string]any{
			"env":   "dev",
			"owner": "platform",
		},
	}))
	if err != nil {
		t.Fatalf("create generic resource returned error: %v", err)
	}

	tagsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/providers/Microsoft.Resources/tags/default?api-version=2021-04-01"
	mergeResp, err := svc.HandleRequest(armCtx(t, http.MethodPatch, tagsURL, map[string]any{
		"operation": "Merge",
		"properties": map[string]any{
			"tags": map[string]any{
				"env":        "prod",
				"costCenter": "eng",
			},
		},
	}))
	if err != nil {
		t.Fatalf("merge generic resource tags returned error: %v", err)
	}
	if mergeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected merge generic resource tags status 200, got %d; body=%s", mergeResp.StatusCode, string(mergeResp.RawBody))
	}
	merged := decodeARMResponse(t, mergeResp)
	mergedTags := merged["properties"].(map[string]any)["tags"].(map[string]any)
	if mergedTags["env"] != "prod" || mergedTags["owner"] != "platform" || mergedTags["costCenter"] != "eng" {
		t.Fatalf("unexpected merged generic resource tags: %v", mergedTags)
	}

	clearResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, tagsURL, nil))
	if err != nil {
		t.Fatalf("delete all generic resource tags returned error: %v", err)
	}
	if clearResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete all generic resource tags status 200, got %d; body=%s", clearResp.StatusCode, string(clearResp.RawBody))
	}

	getResourceResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, resourceURL, nil))
	if err != nil {
		t.Fatalf("get generic resource after tag delete returned error: %v", err)
	}
	resource := decodeARMResponse(t, getResourceResp)
	if _, ok := resource["tags"]; ok {
		t.Fatalf("expected generic resource tags to be removed, got %v", resource["tags"])
	}
}

func TestProviderManifestListGetAndRegister(t *testing.T) {
	svc := resources.New()

	listResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list providers returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list providers status 200, got %d", listResp.StatusCode)
	}
	listed := decodeARMResponse(t, listResp)
	providers := listed["value"].([]any)
	if len(providers) < 4 {
		t.Fatalf("expected at least four provider manifests, got %d", len(providers))
	}
	var listedResourcesProvider map[string]any
	for _, item := range providers {
		provider := item.(map[string]any)
		if provider["namespace"] == "Microsoft.Resources" {
			listedResourcesProvider = provider
			break
		}
	}
	if listedResourcesProvider == nil {
		t.Fatalf("expected Microsoft.Resources in provider list, got %v", providers)
	}
	if listedResourcesProvider["id"] != "/subscriptions/sub-1/providers/Microsoft.Resources" {
		t.Fatalf("unexpected listed Microsoft.Resources provider id: %v", listedResourcesProvider["id"])
	}
	if listedResourcesProvider["registrationPolicy"] != "RegistrationFree" {
		t.Fatalf("unexpected listed Microsoft.Resources registrationPolicy: %v", listedResourcesProvider["registrationPolicy"])
	}

	getResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Storage?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get provider returned error: %v", err)
	}
	got := decodeARMResponse(t, getResp)
	if got["namespace"] != "Microsoft.Storage" {
		t.Errorf("unexpected provider namespace: %v", got["namespace"])
	}
	if got["id"] != "/subscriptions/sub-1/providers/Microsoft.Storage" {
		t.Fatalf("unexpected storage provider id: %v", got["id"])
	}
	if got["registrationPolicy"] != "RegistrationRequired" {
		t.Fatalf("unexpected storage registrationPolicy: %v", got["registrationPolicy"])
	}
	storageResourceTypes := got["resourceTypes"].([]any)
	var storageAccountType map[string]any
	for _, item := range storageResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "storageAccounts" {
			storageAccountType = resourceType
			break
		}
	}
	if storageAccountType == nil {
		t.Fatalf("expected storageAccounts in storage provider manifest, got %v", storageResourceTypes)
	}
	storageAPIVersions := storageAccountType["apiVersions"].([]any)
	hasStorage2024 := false
	for _, version := range storageAPIVersions {
		if version == "2024-01-01" {
			hasStorage2024 = true
			break
		}
	}
	if !hasStorage2024 {
		t.Fatalf("expected storageAccounts API versions to include 2024-01-01, got %v", storageAPIVersions)
	}

	getComputeResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Compute?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get compute provider returned error: %v", err)
	}
	if getComputeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get compute provider status 200, got %d; body=%s", getComputeResp.StatusCode, string(getComputeResp.RawBody))
	}
	computeProvider := decodeARMResponse(t, getComputeResp)
	if computeProvider["namespace"] != "Microsoft.Compute" {
		t.Fatalf("unexpected compute provider namespace: %v", computeProvider["namespace"])
	}
	computeResourceTypes := computeProvider["resourceTypes"].([]any)
	var virtualMachineType map[string]any
	for _, item := range computeResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "virtualMachines" {
			virtualMachineType = resourceType
			break
		}
	}
	if virtualMachineType == nil {
		t.Fatalf("expected virtualMachines in compute provider manifest, got %v", computeResourceTypes)
	}
	var diskType map[string]any
	for _, item := range computeResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "disks" {
			diskType = resourceType
			break
		}
	}
	if diskType == nil {
		t.Fatalf("expected disks in compute provider manifest, got %v", computeResourceTypes)
	}
	computeAPIVersions := virtualMachineType["apiVersions"].([]any)
	hasCompute2025 := false
	for _, version := range computeAPIVersions {
		if version == "2025-11-01" {
			hasCompute2025 = true
			break
		}
	}
	if !hasCompute2025 {
		t.Fatalf("expected virtualMachines API versions to include 2025-11-01, got %v", computeAPIVersions)
	}
	diskAPIVersions := diskType["apiVersions"].([]any)
	hasDisk2025 := false
	for _, version := range diskAPIVersions {
		if version == "2025-01-02" {
			hasDisk2025 = true
			break
		}
	}
	if !hasDisk2025 {
		t.Fatalf("expected disks API versions to include 2025-01-02, got %v", diskAPIVersions)
	}

	getWebResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Web?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get web provider returned error: %v", err)
	}
	if getWebResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get web provider status 200, got %d; body=%s", getWebResp.StatusCode, string(getWebResp.RawBody))
	}
	webProvider := decodeARMResponse(t, getWebResp)
	if webProvider["namespace"] != "Microsoft.Web" {
		t.Fatalf("unexpected web provider namespace: %v", webProvider["namespace"])
	}
	webResourceTypes := webProvider["resourceTypes"].([]any)
	webTypes := make(map[string]map[string]any, len(webResourceTypes))
	for _, item := range webResourceTypes {
		resourceType := item.(map[string]any)
		webTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"serverfarms", "sites", "sites/slots"} {
		foundType := webTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in web provider manifest, got %v", resourceType, webResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		hasAppService2024 := false
		for _, version := range versions {
			if version == "2024-04-01" {
				hasAppService2024 = true
				break
			}
		}
		if !hasAppService2024 {
			t.Fatalf("expected %s API versions to include 2024-04-01, got %v", resourceType, versions)
		}
	}

	getNetworkResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Network?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get network provider returned error: %v", err)
	}
	if getNetworkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get network provider status 200, got %d; body=%s", getNetworkResp.StatusCode, string(getNetworkResp.RawBody))
	}
	networkProvider := decodeARMResponse(t, getNetworkResp)
	if networkProvider["namespace"] != "Microsoft.Network" {
		t.Fatalf("unexpected network provider namespace: %v", networkProvider["namespace"])
	}
	networkResourceTypes := networkProvider["resourceTypes"].([]any)
	networkTypes := make(map[string]map[string]any, len(networkResourceTypes))
	for _, item := range networkResourceTypes {
		resourceType := item.(map[string]any)
		networkTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"virtualNetworks", "virtualNetworks/subnets", "networkSecurityGroups", "networkSecurityGroups/securityRules", "publicIPAddresses", "networkInterfaces", "loadBalancers", "applicationGateways", "privateEndpoints", "privateEndpoints/privateDnsZoneGroups"} {
		foundType := networkTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in network provider manifest, got %v", resourceType, networkResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		for _, expectedVersion := range []string{"2025-05-01", "2023-09-01"} {
			foundVersion := false
			for _, version := range versions {
				if version == expectedVersion {
					foundVersion = true
					break
				}
			}
			if !foundVersion {
				t.Fatalf("expected %s API versions to include %s, got %v", resourceType, expectedVersion, versions)
			}
		}
	}
	for _, resourceType := range []string{"dnsZones", "dnsZones/A", "dnsZones/CNAME", "dnsZones/MX", "dnsZones/TXT"} {
		foundType := networkTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in network provider manifest, got %v", resourceType, networkResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		hasDNS2018 := false
		for _, version := range versions {
			if version == "2018-05-01" {
				hasDNS2018 = true
				break
			}
		}
		if !hasDNS2018 {
			t.Fatalf("expected %s API versions to include 2018-05-01, got %v", resourceType, versions)
		}
	}

	getInsightsResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Insights?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get insights provider returned error: %v", err)
	}
	if getInsightsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get insights provider status 200, got %d; body=%s", getInsightsResp.StatusCode, string(getInsightsResp.RawBody))
	}
	insightsProvider := decodeARMResponse(t, getInsightsResp)
	if insightsProvider["namespace"] != "Microsoft.Insights" {
		t.Fatalf("unexpected insights provider namespace: %v", insightsProvider["namespace"])
	}
	insightsResourceTypes := insightsProvider["resourceTypes"].([]any)
	insightsTypes := make(map[string]map[string]any, len(insightsResourceTypes))
	for _, item := range insightsResourceTypes {
		resourceType := item.(map[string]any)
		insightsTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for resourceType, expectedVersions := range map[string][]string{
		"actionGroups":       {"2021-09-01", "2019-06-01"},
		"metricAlerts":       {"2024-03-01-preview", "2018-03-01"},
		"diagnosticSettings": {"2021-05-01-preview"},
		"metrics":            {"2023-10-01"},
		"metricDefinitions":  {"2023-10-01"},
		"components":         {"2015-05-01"},
		"eventtypes":         {"2015-04-01"},
	} {
		foundType := insightsTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in insights provider manifest, got %v", resourceType, insightsResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		for _, expectedVersion := range expectedVersions {
			foundVersion := false
			for _, version := range versions {
				if version == expectedVersion {
					foundVersion = true
					break
				}
			}
			if !foundVersion {
				t.Fatalf("expected %s API versions to include %s, got %v", resourceType, expectedVersion, versions)
			}
		}
	}

	getAppConfigResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.AppConfiguration?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get app configuration provider returned error: %v", err)
	}
	if getAppConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get app configuration provider status 200, got %d; body=%s", getAppConfigResp.StatusCode, string(getAppConfigResp.RawBody))
	}
	appConfigProvider := decodeARMResponse(t, getAppConfigResp)
	if appConfigProvider["namespace"] != "Microsoft.AppConfiguration" {
		t.Fatalf("unexpected app configuration provider namespace: %v", appConfigProvider["namespace"])
	}
	appConfigResourceTypes := appConfigProvider["resourceTypes"].([]any)
	appConfigTypes := make(map[string]map[string]any, len(appConfigResourceTypes))
	for _, item := range appConfigResourceTypes {
		resourceType := item.(map[string]any)
		appConfigTypes[resourceType["resourceType"].(string)] = resourceType
	}
	storeType := appConfigTypes["configurationStores"]
	if storeType == nil {
		t.Fatalf("expected configurationStores in app configuration provider manifest, got %v", appConfigResourceTypes)
	}
	storeVersions := storeType["apiVersions"].([]any)
	hasStore2024 := false
	for _, version := range storeVersions {
		if version == "2024-06-01" {
			hasStore2024 = true
			break
		}
	}
	if !hasStore2024 {
		t.Fatalf("expected configurationStores API versions to include 2024-06-01, got %v", storeVersions)
	}

	getOperationalInsightsResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.OperationalInsights?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get operational insights provider returned error: %v", err)
	}
	if getOperationalInsightsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get operational insights provider status 200, got %d; body=%s", getOperationalInsightsResp.StatusCode, string(getOperationalInsightsResp.RawBody))
	}
	operationalInsightsProvider := decodeARMResponse(t, getOperationalInsightsResp)
	if operationalInsightsProvider["namespace"] != "Microsoft.OperationalInsights" {
		t.Fatalf("unexpected operational insights provider namespace: %v", operationalInsightsProvider["namespace"])
	}
	operationalInsightsResourceTypes := operationalInsightsProvider["resourceTypes"].([]any)
	operationalInsightsTypes := make(map[string]map[string]any, len(operationalInsightsResourceTypes))
	for _, item := range operationalInsightsResourceTypes {
		resourceType := item.(map[string]any)
		operationalInsightsTypes[resourceType["resourceType"].(string)] = resourceType
	}
	workspaceType := operationalInsightsTypes["workspaces"]
	if workspaceType == nil {
		t.Fatalf("expected workspaces in operational insights provider manifest, got %v", operationalInsightsResourceTypes)
	}
	workspaceVersions := workspaceType["apiVersions"].([]any)
	hasWorkspace2025 := false
	for _, version := range workspaceVersions {
		if version == "2025-02-01" {
			hasWorkspace2025 = true
			break
		}
	}
	if !hasWorkspace2025 {
		t.Fatalf("expected workspaces API versions to include 2025-02-01, got %v", workspaceVersions)
	}

	getServiceBusResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ServiceBus?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get service bus provider returned error: %v", err)
	}
	if getServiceBusResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get service bus provider status 200, got %d; body=%s", getServiceBusResp.StatusCode, string(getServiceBusResp.RawBody))
	}
	serviceBusProvider := decodeARMResponse(t, getServiceBusResp)
	if serviceBusProvider["namespace"] != "Microsoft.ServiceBus" {
		t.Fatalf("unexpected service bus provider namespace: %v", serviceBusProvider["namespace"])
	}
	serviceBusResourceTypes := serviceBusProvider["resourceTypes"].([]any)
	serviceBusTypes := make(map[string]map[string]any, len(serviceBusResourceTypes))
	for _, item := range serviceBusResourceTypes {
		resourceType := item.(map[string]any)
		serviceBusTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"namespaces", "namespaces/authorizationRules", "namespaces/queues", "namespaces/queues/authorizationRules", "namespaces/topics", "namespaces/topics/authorizationRules", "namespaces/topics/subscriptions"} {
		foundType := serviceBusTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in service bus provider manifest, got %v", resourceType, serviceBusResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		hasServiceBus2024 := false
		for _, version := range versions {
			if version == "2024-01-01" {
				hasServiceBus2024 = true
				break
			}
		}
		if !hasServiceBus2024 {
			t.Fatalf("expected %s API versions to include 2024-01-01, got %v", resourceType, versions)
		}
	}

	getEventGridResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.EventGrid?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get event grid provider returned error: %v", err)
	}
	if getEventGridResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get event grid provider status 200, got %d; body=%s", getEventGridResp.StatusCode, string(getEventGridResp.RawBody))
	}
	eventGridProvider := decodeARMResponse(t, getEventGridResp)
	if eventGridProvider["namespace"] != "Microsoft.EventGrid" {
		t.Fatalf("unexpected event grid provider namespace: %v", eventGridProvider["namespace"])
	}
	eventGridResourceTypes := eventGridProvider["resourceTypes"].([]any)
	eventGridTypes := make(map[string]map[string]any, len(eventGridResourceTypes))
	for _, item := range eventGridResourceTypes {
		resourceType := item.(map[string]any)
		eventGridTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"topics", "topics/eventSubscriptions"} {
		foundType := eventGridTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in event grid provider manifest, got %v", resourceType, eventGridResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		hasEventGrid2025 := false
		for _, version := range versions {
			if version == "2025-02-15" {
				hasEventGrid2025 = true
				break
			}
		}
		if !hasEventGrid2025 {
			t.Fatalf("expected %s API versions to include 2025-02-15, got %v", resourceType, versions)
		}
	}

	getEventHubResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.EventHub?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get event hubs provider returned error: %v", err)
	}
	if getEventHubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get event hubs provider status 200, got %d; body=%s", getEventHubResp.StatusCode, string(getEventHubResp.RawBody))
	}
	eventHubProvider := decodeARMResponse(t, getEventHubResp)
	if eventHubProvider["namespace"] != "Microsoft.EventHub" {
		t.Fatalf("unexpected event hubs provider namespace: %v", eventHubProvider["namespace"])
	}
	eventHubResourceTypes := eventHubProvider["resourceTypes"].([]any)
	eventHubTypes := make(map[string]map[string]any, len(eventHubResourceTypes))
	for _, item := range eventHubResourceTypes {
		resourceType := item.(map[string]any)
		eventHubTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{
		"namespaces",
		"namespaces/authorizationRules",
		"namespaces/eventhubs",
		"namespaces/eventhubs/authorizationRules",
		"namespaces/eventhubs/consumergroups",
	} {
		foundType := eventHubTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in event hubs provider manifest, got %v", resourceType, eventHubResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		for _, expectedVersion := range []string{"2026-01-01", "2024-01-01"} {
			foundVersion := false
			for _, version := range versions {
				if version == expectedVersion {
					foundVersion = true
					break
				}
			}
			if !foundVersion {
				t.Fatalf("expected %s API versions to include %s, got %v", resourceType, expectedVersion, versions)
			}
		}
	}

	getContainerRegistryResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerRegistry?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get container registry provider returned error: %v", err)
	}
	if getContainerRegistryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get container registry provider status 200, got %d; body=%s", getContainerRegistryResp.StatusCode, string(getContainerRegistryResp.RawBody))
	}
	containerRegistryProvider := decodeARMResponse(t, getContainerRegistryResp)
	if containerRegistryProvider["namespace"] != "Microsoft.ContainerRegistry" {
		t.Fatalf("unexpected container registry provider namespace: %v", containerRegistryProvider["namespace"])
	}
	containerRegistryResourceTypes := containerRegistryProvider["resourceTypes"].([]any)
	var registryType map[string]any
	for _, item := range containerRegistryResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "registries" {
			registryType = resourceType
			break
		}
	}
	if registryType == nil {
		t.Fatalf("expected registries in container registry provider manifest, got %v", containerRegistryResourceTypes)
	}
	registryAPIVersions := registryType["apiVersions"].([]any)
	for _, expectedVersion := range []string{"2025-11-01", "2023-07-01"} {
		foundVersion := false
		for _, version := range registryAPIVersions {
			if version == expectedVersion {
				foundVersion = true
				break
			}
		}
		if !foundVersion {
			t.Fatalf("expected registries API versions to include %s, got %v", expectedVersion, registryAPIVersions)
		}
	}

	getContainerInstanceResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get container instance provider returned error: %v", err)
	}
	if getContainerInstanceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get container instance provider status 200, got %d; body=%s", getContainerInstanceResp.StatusCode, string(getContainerInstanceResp.RawBody))
	}
	containerInstanceProvider := decodeARMResponse(t, getContainerInstanceResp)
	if containerInstanceProvider["namespace"] != "Microsoft.ContainerInstance" {
		t.Fatalf("unexpected container instance provider namespace: %v", containerInstanceProvider["namespace"])
	}
	containerInstanceResourceTypes := containerInstanceProvider["resourceTypes"].([]any)
	var containerGroupType map[string]any
	for _, item := range containerInstanceResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "containerGroups" {
			containerGroupType = resourceType
			break
		}
	}
	if containerGroupType == nil {
		t.Fatalf("expected containerGroups in container instance provider manifest, got %v", containerInstanceResourceTypes)
	}
	containerGroupAPIVersions := containerGroupType["apiVersions"].([]any)
	hasContainerGroups2025 := false
	for _, version := range containerGroupAPIVersions {
		if version == "2025-09-01" {
			hasContainerGroups2025 = true
			break
		}
	}
	if !hasContainerGroups2025 {
		t.Fatalf("expected containerGroups API versions to include 2025-09-01, got %v", containerGroupAPIVersions)
	}

	getContainerServiceResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerService?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get container service provider returned error: %v", err)
	}
	if getContainerServiceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get container service provider status 200, got %d; body=%s", getContainerServiceResp.StatusCode, string(getContainerServiceResp.RawBody))
	}
	containerServiceProvider := decodeARMResponse(t, getContainerServiceResp)
	if containerServiceProvider["namespace"] != "Microsoft.ContainerService" {
		t.Fatalf("unexpected container service provider namespace: %v", containerServiceProvider["namespace"])
	}
	containerServiceResourceTypes := containerServiceProvider["resourceTypes"].([]any)
	containerServiceTypes := make(map[string]map[string]any, len(containerServiceResourceTypes))
	for _, item := range containerServiceResourceTypes {
		resourceType := item.(map[string]any)
		containerServiceTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"managedClusters", "locations/kubernetesVersions"} {
		foundType := containerServiceTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in container service provider manifest, got %v", resourceType, containerServiceResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		foundVersion := false
		for _, version := range versions {
			if version == "2026-03-01" {
				foundVersion = true
				break
			}
		}
		if !foundVersion {
			t.Fatalf("expected %s API versions to include 2026-03-01, got %v", resourceType, versions)
		}
	}

	getContainerAppsResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.App?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get container apps provider returned error: %v", err)
	}
	if getContainerAppsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get container apps provider status 200, got %d; body=%s", getContainerAppsResp.StatusCode, string(getContainerAppsResp.RawBody))
	}
	containerAppsProvider := decodeARMResponse(t, getContainerAppsResp)
	if containerAppsProvider["namespace"] != "Microsoft.App" {
		t.Fatalf("unexpected container apps provider namespace: %v", containerAppsProvider["namespace"])
	}
	containerAppsResourceTypes := containerAppsProvider["resourceTypes"].([]any)
	containerAppTypes := make(map[string]map[string]any, len(containerAppsResourceTypes))
	for _, item := range containerAppsResourceTypes {
		resourceType := item.(map[string]any)
		containerAppTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"managedEnvironments", "containerApps"} {
		foundType := containerAppTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in container apps provider manifest, got %v", resourceType, containerAppsResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		foundVersion := false
		for _, version := range versions {
			if version == "2025-07-01" {
				foundVersion = true
				break
			}
		}
		if !foundVersion {
			t.Fatalf("expected %s API versions to include 2025-07-01, got %v", resourceType, versions)
		}
	}

	getDocumentDBResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.DocumentDB?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get document db provider returned error: %v", err)
	}
	if getDocumentDBResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get document db provider status 200, got %d; body=%s", getDocumentDBResp.StatusCode, string(getDocumentDBResp.RawBody))
	}
	documentDBProvider := decodeARMResponse(t, getDocumentDBResp)
	if documentDBProvider["namespace"] != "Microsoft.DocumentDB" {
		t.Fatalf("unexpected document db provider namespace: %v", documentDBProvider["namespace"])
	}
	documentDBResourceTypes := documentDBProvider["resourceTypes"].([]any)
	documentDBTypes := make(map[string]map[string]any, len(documentDBResourceTypes))
	for _, item := range documentDBResourceTypes {
		resourceType := item.(map[string]any)
		documentDBTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"databaseAccounts", "databaseAccounts/sqlDatabases", "databaseAccounts/sqlDatabases/containers"} {
		foundType := documentDBTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in document db provider manifest, got %v", resourceType, documentDBResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		for _, expectedVersion := range []string{"2025-05-01", "2024-05-15"} {
			foundVersion := false
			for _, version := range versions {
				if version == expectedVersion {
					foundVersion = true
					break
				}
			}
			if !foundVersion {
				t.Fatalf("expected %s API versions to include %s, got %v", resourceType, expectedVersion, versions)
			}
		}
	}

	getSQLResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Sql?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get sql provider returned error: %v", err)
	}
	if getSQLResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get sql provider status 200, got %d; body=%s", getSQLResp.StatusCode, string(getSQLResp.RawBody))
	}
	sqlProvider := decodeARMResponse(t, getSQLResp)
	if sqlProvider["namespace"] != "Microsoft.Sql" {
		t.Fatalf("unexpected sql provider namespace: %v", sqlProvider["namespace"])
	}
	sqlResourceTypes := sqlProvider["resourceTypes"].([]any)
	sqlTypes := make(map[string]map[string]any, len(sqlResourceTypes))
	for _, item := range sqlResourceTypes {
		resourceType := item.(map[string]any)
		sqlTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"servers", "servers/databases", "servers/firewallRules"} {
		foundType := sqlTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in sql provider manifest, got %v", resourceType, sqlResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		for _, expectedVersion := range []string{"2025-01-01", "2023-08-01"} {
			foundVersion := false
			for _, version := range versions {
				if version == expectedVersion {
					foundVersion = true
					break
				}
			}
			if !foundVersion {
				t.Fatalf("expected %s API versions to include %s, got %v", resourceType, expectedVersion, versions)
			}
		}
	}

	getPostgreSQLResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.DBforPostgreSQL?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get postgresql provider returned error: %v", err)
	}
	if getPostgreSQLResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get postgresql provider status 200, got %d; body=%s", getPostgreSQLResp.StatusCode, string(getPostgreSQLResp.RawBody))
	}
	postgresqlProvider := decodeARMResponse(t, getPostgreSQLResp)
	if postgresqlProvider["namespace"] != "Microsoft.DBforPostgreSQL" {
		t.Fatalf("unexpected postgresql provider namespace: %v", postgresqlProvider["namespace"])
	}
	postgresqlResourceTypes := postgresqlProvider["resourceTypes"].([]any)
	postgresqlTypes := make(map[string]map[string]any, len(postgresqlResourceTypes))
	for _, item := range postgresqlResourceTypes {
		resourceType := item.(map[string]any)
		postgresqlTypes[resourceType["resourceType"].(string)] = resourceType
	}
	for _, resourceType := range []string{"flexibleServers", "flexibleServers/databases", "flexibleServers/firewallRules"} {
		foundType := postgresqlTypes[resourceType]
		if foundType == nil {
			t.Fatalf("expected %s in postgresql provider manifest, got %v", resourceType, postgresqlResourceTypes)
		}
		versions := foundType["apiVersions"].([]any)
		for _, expectedVersion := range []string{"2025-08-01", "2024-08-01"} {
			foundVersion := false
			for _, version := range versions {
				if version == expectedVersion {
					foundVersion = true
					break
				}
			}
			if !foundVersion {
				t.Fatalf("expected %s API versions to include %s, got %v", resourceType, expectedVersion, versions)
			}
		}
	}

	getCacheResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Cache?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get cache provider returned error: %v", err)
	}
	if getCacheResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get cache provider status 200, got %d; body=%s", getCacheResp.StatusCode, string(getCacheResp.RawBody))
	}
	cacheProvider := decodeARMResponse(t, getCacheResp)
	if cacheProvider["namespace"] != "Microsoft.Cache" {
		t.Fatalf("unexpected cache provider namespace: %v", cacheProvider["namespace"])
	}
	cacheResourceTypes := cacheProvider["resourceTypes"].([]any)
	var redisType map[string]any
	for _, item := range cacheResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "Redis" {
			redisType = resourceType
			break
		}
	}
	if redisType == nil {
		t.Fatalf("expected Redis in cache provider manifest, got %v", cacheResourceTypes)
	}
	redisAPIVersions := redisType["apiVersions"].([]any)
	for _, expectedVersion := range []string{"2024-11-01", "2023-08-01"} {
		foundVersion := false
		for _, version := range redisAPIVersions {
			if version == expectedVersion {
				foundVersion = true
				break
			}
		}
		if !foundVersion {
			t.Fatalf("expected Redis API versions to include %s, got %v", expectedVersion, redisAPIVersions)
		}
	}

	getManagedIdentityResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ManagedIdentity?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get managed identity provider returned error: %v", err)
	}
	if getManagedIdentityResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get managed identity provider status 200, got %d; body=%s", getManagedIdentityResp.StatusCode, string(getManagedIdentityResp.RawBody))
	}
	managedIdentityProvider := decodeARMResponse(t, getManagedIdentityResp)
	if managedIdentityProvider["namespace"] != "Microsoft.ManagedIdentity" {
		t.Fatalf("unexpected managed identity provider namespace: %v", managedIdentityProvider["namespace"])
	}
	managedIdentityResourceTypes := managedIdentityProvider["resourceTypes"].([]any)
	var userAssignedIdentityType map[string]any
	for _, item := range managedIdentityResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "userAssignedIdentities" {
			userAssignedIdentityType = resourceType
			break
		}
	}
	if userAssignedIdentityType == nil {
		t.Fatalf("expected userAssignedIdentities in managed identity provider manifest, got %v", managedIdentityResourceTypes)
	}
	userAssignedIdentityAPIVersions := userAssignedIdentityType["apiVersions"].([]any)
	for _, expectedVersion := range []string{"2023-01-31", "2018-11-30"} {
		foundVersion := false
		for _, version := range userAssignedIdentityAPIVersions {
			if version == expectedVersion {
				foundVersion = true
				break
			}
		}
		if !foundVersion {
			t.Fatalf("expected userAssignedIdentities API versions to include %s, got %v", expectedVersion, userAssignedIdentityAPIVersions)
		}
	}

	getCDNResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Cdn?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get CDN provider returned error: %v", err)
	}
	if getCDNResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get CDN provider status 200, got %d; body=%s", getCDNResp.StatusCode, string(getCDNResp.RawBody))
	}
	cdnProvider := decodeARMResponse(t, getCDNResp)
	if cdnProvider["namespace"] != "Microsoft.Cdn" {
		t.Fatalf("unexpected CDN provider namespace: %v", cdnProvider["namespace"])
	}
	cdnResourceTypes := cdnProvider["resourceTypes"].([]any)
	var profileType map[string]any
	var endpointType map[string]any
	var originGroupType map[string]any
	var originType map[string]any
	var customDomainType map[string]any
	for _, item := range cdnResourceTypes {
		resourceType := item.(map[string]any)
		if resourceType["resourceType"] == "profiles" {
			profileType = resourceType
			continue
		}
		if resourceType["resourceType"] == "profiles/endpoints" {
			endpointType = resourceType
			continue
		}
		if resourceType["resourceType"] == "profiles/endpoints/originGroups" {
			originGroupType = resourceType
			continue
		}
		if resourceType["resourceType"] == "profiles/endpoints/origins" {
			originType = resourceType
			continue
		}
		if resourceType["resourceType"] == "profiles/endpoints/customDomains" {
			customDomainType = resourceType
		}
	}
	if profileType == nil {
		t.Fatalf("expected profiles in CDN provider manifest, got %v", cdnResourceTypes)
	}
	profileAPIVersions := profileType["apiVersions"].([]any)
	if len(profileAPIVersions) == 0 || profileAPIVersions[0] != "2025-04-15" {
		t.Fatalf("unexpected CDN profile API versions: %v", profileAPIVersions)
	}
	if endpointType == nil {
		t.Fatalf("expected profiles/endpoints in CDN provider manifest, got %v", cdnResourceTypes)
	}
	endpointAPIVersions := endpointType["apiVersions"].([]any)
	if len(endpointAPIVersions) == 0 || endpointAPIVersions[0] != "2025-04-15" {
		t.Fatalf("unexpected CDN endpoint API versions: %v", endpointAPIVersions)
	}
	if originGroupType == nil {
		t.Fatalf("expected profiles/endpoints/originGroups in CDN provider manifest, got %v", cdnResourceTypes)
	}
	originGroupAPIVersions := originGroupType["apiVersions"].([]any)
	if len(originGroupAPIVersions) == 0 || originGroupAPIVersions[0] != "2025-04-15" {
		t.Fatalf("unexpected CDN origin group API versions: %v", originGroupAPIVersions)
	}
	if originType == nil {
		t.Fatalf("expected profiles/endpoints/origins in CDN provider manifest, got %v", cdnResourceTypes)
	}
	originAPIVersions := originType["apiVersions"].([]any)
	if len(originAPIVersions) == 0 || originAPIVersions[0] != "2025-04-15" {
		t.Fatalf("unexpected CDN origin API versions: %v", originAPIVersions)
	}
	if customDomainType == nil {
		t.Fatalf("expected profiles/endpoints/customDomains in CDN provider manifest, got %v", cdnResourceTypes)
	}
	customDomainAPIVersions := customDomainType["apiVersions"].([]any)
	if len(customDomainAPIVersions) == 0 || customDomainAPIVersions[0] != "2025-04-15" {
		t.Fatalf("unexpected CDN custom domain API versions: %v", customDomainAPIVersions)
	}

	getAuthorizationResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Authorization?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get authorization provider returned error: %v", err)
	}
	if getAuthorizationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get authorization provider status 200, got %d; body=%s", getAuthorizationResp.StatusCode, string(getAuthorizationResp.RawBody))
	}
	authorizationProvider := decodeARMResponse(t, getAuthorizationResp)
	if authorizationProvider["namespace"] != "Microsoft.Authorization" {
		t.Errorf("unexpected authorization provider namespace: %v", authorizationProvider["namespace"])
	}
	authResourceTypes := authorizationProvider["resourceTypes"].([]any)
	authorizationTypes := make(map[string]map[string]any, len(authResourceTypes))
	for _, item := range authResourceTypes {
		resourceType := item.(map[string]any)
		authorizationTypes[resourceType["resourceType"].(string)] = resourceType
	}
	if _, ok := authorizationTypes["roleAssignments"]; !ok {
		t.Fatalf("expected roleAssignments in authorization provider manifest, got %v", authorizationTypes)
	}
	locks, ok := authorizationTypes["locks"]
	if !ok {
		t.Fatalf("expected locks in authorization provider manifest, got %v", authorizationTypes)
	}
	lockAPIVersions := locks["apiVersions"].([]any)
	if len(lockAPIVersions) == 0 || lockAPIVersions[0] != "2020-05-01" {
		t.Fatalf("unexpected locks API versions: %v", lockAPIVersions)
	}

	registerResp, err := svc.HandleRequest(armCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Storage/register?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("register provider returned error: %v", err)
	}
	registered := decodeARMResponse(t, registerResp)
	if registered["registrationState"] != "Registered" {
		t.Errorf("unexpected registration state: %v", registered["registrationState"])
	}
	if registered["id"] != "/subscriptions/sub-1/providers/Microsoft.Storage" {
		t.Fatalf("unexpected registered provider id: %v", registered["id"])
	}
	if registered["registrationPolicy"] != "RegistrationRequired" {
		t.Fatalf("unexpected registered provider registrationPolicy: %v", registered["registrationPolicy"])
	}
}

func TestDeploymentCreateGetListAndProvisionStorageAccount(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"parameters": map[string]any{
				"storageAccountName": map[string]any{"value": "depstoreacct"},
			},
			"template": map[string]any{
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[parameters('storageAccountName')]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"tags":       map[string]any{"env": "deploy"},
					},
				},
				"outputs": map[string]any{
					"accountName": map[string]any{
						"type":  "string",
						"value": "[parameters('storageAccountName')]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-a?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected deployment create status 201, got %d", createResp.StatusCode)
	}
	created := decodeARMResponse(t, createResp)
	if created["name"] != "dep-a" {
		t.Fatalf("unexpected deployment name: %v", created["name"])
	}
	props := created["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected provisioning state: %v", props["provisioningState"])
	}
	outputs := props["outputs"].(map[string]any)
	accountName := outputs["accountName"].(map[string]any)
	if accountName["value"] != "depstoreacct" {
		t.Fatalf("unexpected output value: %v", accountName["value"])
	}

	getResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get deployment returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get deployment status 200, got %d", getResp.StatusCode)
	}

	listResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list deployments returned error: %v", err)
	}
	listed := decodeARMResponse(t, listResp)
	if got := len(listed["value"].([]any)); got != 1 {
		t.Fatalf("expected one deployment, got %d", got)
	}

	deleteResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("delete deployment returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected deployment delete status 202, got %d", deleteResp.StatusCode)
	}
	if deleteResp.Headers["Location"] == "" {
		t.Fatalf("expected deployment delete Location header, got %v", deleteResp.Headers)
	}
	if deleteResp.Headers["Retry-After"] == "" {
		t.Fatalf("expected deployment delete Retry-After header, got %v", deleteResp.Headers)
	}
	deleteStatusResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, deleteResp.Headers["Location"], nil))
	if err != nil {
		t.Fatalf("get deployment delete status returned error: %v", err)
	}
	if deleteStatusResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected deployment delete status polling to return 202 while running, got %d body=%s", deleteStatusResp.StatusCode, string(deleteStatusResp.RawBody))
	}
	deleteCompleteResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, deleteResp.Headers["Location"], nil))
	if err != nil {
		t.Fatalf("get deployment delete completion status returned error: %v", err)
	}
	if deleteCompleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected deployment delete completion polling to return 204, got %d body=%s", deleteCompleteResp.StatusCode, string(deleteCompleteResp.RawBody))
	}

	deletedGetResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("get deleted deployment returned error: %v", err)
	}
	if deletedGetResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted deployment get status 404, got %d body=%s", deletedGetResp.StatusCode, string(deletedGetResp.RawBody))
	}

	deletedListResp, err := svc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("list deployments after delete returned error: %v", err)
	}
	deletedList := decodeARMResponse(t, deletedListResp)
	if got := len(deletedList["value"].([]any)); got != 0 {
		t.Fatalf("expected no deployment history after delete, got %d", got)
	}

	missingDeleteResp, err := svc.HandleRequest(armCtx(t, http.MethodDelete, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-a?api-version=2021-04-01", nil))
	if err != nil {
		t.Fatalf("delete missing deployment returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing deployment delete status 204, got %d body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}

	storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/depstoreacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get provisioned storage account returned error: %v", err)
	}
	storageAccount := decodeARMResponse(t, storageResp)
	if storageAccount["name"] != "depstoreacct" {
		t.Fatalf("expected provisioned storage account, got %v", storageAccount["name"])
	}
}

func TestDeploymentCanProvisionResourcesAcrossProviders(t *testing.T) {
	storageSvc := storage.New()
	keyVaultSvc := keyvault.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)
	svc.SetTemplateProvisioner(keyVaultSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "multistoreacct",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
					},
					map[string]any{
						"type":       "Microsoft.KeyVault/vaults",
						"apiVersion": "2024-11-01",
						"name":       "multivault",
						"location":   "westus2",
						"properties": map[string]any{
							"tenantId":       "tenant-1",
							"sku":            map[string]any{"family": "A", "name": "standard"},
							"accessPolicies": []any{},
						},
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-multi?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected deployment create status 201, got %d", createResp.StatusCode)
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	if got := len(props["outputResources"].([]any)); got != 2 {
		t.Fatalf("expected two output resources, got %d", got)
	}

	storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/multistoreacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get provisioned storage account returned error: %v", err)
	}
	if storageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected provisioned storage account status 200, got %d", storageResp.StatusCode)
	}

	vaultResp, err := keyVaultSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.KeyVault/vaults/multivault?api-version=2024-11-01", nil))
	if err != nil {
		t.Fatalf("get provisioned key vault returned error: %v", err)
	}
	vault := decodeARMResponse(t, vaultResp)
	if vault["name"] != "multivault" {
		t.Fatalf("expected provisioned key vault, got %v", vault["name"])
	}
}

func TestDeploymentResolvesTemplateVariablesAndConcatExpressions(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"parameters": map[string]any{
				"suffix": map[string]any{"value": "acct"},
			},
			"template": map[string]any{
				"variables": map[string]any{
					"namePrefix":  "varstore",
					"accountName": "[concat(variables('namePrefix'), parameters('suffix'))]",
					"tagBlock": map[string]any{
						"source": "[concat('arm-', variables('namePrefix'))]",
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[variables('accountName')]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"tags":       "[variables('tagBlock')]",
					},
				},
				"outputs": map[string]any{
					"accountName": map[string]any{
						"type":  "string",
						"value": "[variables('accountName')]",
					},
					"prefixedName": map[string]any{
						"type":  "string",
						"value": "[concat('deployed-', variables('accountName'))]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-vars?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create variable deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected variable deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["accountName"].(map[string]any)["value"] != "varstoreacct" {
		t.Fatalf("expected variable output accountName to resolve, got %v", outputs["accountName"])
	}
	if outputs["prefixedName"].(map[string]any)["value"] != "deployed-varstoreacct" {
		t.Fatalf("expected concat output to resolve, got %v", outputs["prefixedName"])
	}

	storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/varstoreacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get variable-provisioned storage account returned error: %v", err)
	}
	if storageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected variable-provisioned storage account status 200, got %d body=%s", storageResp.StatusCode, string(storageResp.RawBody))
	}
	account := decodeARMResponse(t, storageResp)
	tags := account["tags"].(map[string]any)
	if tags["source"] != "arm-varstore" {
		t.Fatalf("expected variable tag block to resolve, got %v", tags)
	}
}

func TestDeploymentExpandsVariableCopyLoop(t *testing.T) {
	svc := resources.New()

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"itemCount": map[string]any{
						"type":         "int",
						"defaultValue": 3,
					},
				},
				"variables": map[string]any{
					"copy": []any{
						map[string]any{
							"name":  "objectArray",
							"count": "[parameters('itemCount')]",
							"input": map[string]any{
								"name":       "[concat('myDataDisk', copyIndex('objectArray', 1))]",
								"diskSizeGB": "1",
								"diskIndex":  "[copyIndex('objectArray')]",
							},
						},
					},
				},
				"resources": []any{},
				"outputs": map[string]any{
					"arrayResult": map[string]any{
						"type":  "array",
						"value": "[variables('objectArray')]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-variable-copy?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create variable-copy deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected variable-copy deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	arrayResult := outputs["arrayResult"].(map[string]any)
	values, ok := arrayResult["value"].([]any)
	if !ok {
		t.Fatalf("expected variable copy output array, got %v", arrayResult["value"])
	}
	if len(values) != 3 {
		t.Fatalf("expected three variable copy entries, got %v", values)
	}
	for index, value := range values {
		entry := value.(map[string]any)
		expectedName := "myDataDisk" + strconv.Itoa(index+1)
		if entry["name"] != expectedName || entry["diskSizeGB"] != "1" || entry["diskIndex"] != float64(index) {
			t.Fatalf("expected variable copy entry %d to match generated disk object, got %v", index, entry)
		}
	}
}

func TestDeploymentReferenceReturnsRuntimeStateInOutputs(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"accountName": "refstoreacct",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[variables('accountName')]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
					},
				},
				"outputs": map[string]any{
					"blobEndpoint": map[string]any{
						"type":  "string",
						"value": "[reference(variables('accountName')).primaryEndpoints.blob]",
					},
					"fullLocation": map[string]any{
						"type":  "string",
						"value": "[reference(resourceId('Microsoft.Storage/storageAccounts', variables('accountName')), '2024-01-01', 'Full').location]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-reference?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create reference deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected reference deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["blobEndpoint"].(map[string]any)["value"] != "https://refstoreacct.blob.core.windows.net/" {
		t.Fatalf("expected reference output to return blob endpoint, got %v", outputs["blobEndpoint"])
	}
	if outputs["fullLocation"].(map[string]any)["value"] != "westus2" {
		t.Fatalf("expected full reference output to return location, got %v", outputs["fullLocation"])
	}
}

func TestDeploymentListKeysReturnsStorageAccountKeysInOutputs(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"accountName": "listkeystoreacct",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[variables('accountName')]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
					},
				},
				"outputs": map[string]any{
					"keyName": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('accountName')), '2024-01-01').keys[0].keyName]",
					},
					"secondKeyName": map[string]any{
						"type":  "string",
						"value": "[listKeys(variables('accountName'), '2024-01-01').keys[1].keyName]",
					},
					"storageKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.Storage/storageAccounts', variables('accountName')), '2024-01-01').keys[0].value]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["keyName"].(map[string]any)["value"] != "key1" {
		t.Fatalf("expected listKeys keyName output to return key1, got %v", outputs["keyName"])
	}
	if outputs["secondKeyName"].(map[string]any)["value"] != "key2" {
		t.Fatalf("expected listKeys secondKeyName output to return key2 from resource name lookup, got %v", outputs["secondKeyName"])
	}
	expectedKey := base64.StdEncoding.EncodeToString([]byte("cloudmock:sub-1/rg-a/listkeystoreacct:key1:0"))
	if outputs["storageKey"].(map[string]any)["value"] != expectedKey {
		t.Fatalf("expected listKeys storageKey output to return deterministic storage key, got %v", outputs["storageKey"])
	}
}

func TestDeploymentListKeysReturnsRedisKeysInOutputs(t *testing.T) {
	redisSvc := redis.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(redisSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"cacheName": "cache-a",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Cache/Redis",
						"apiVersion": "2024-11-01",
						"name":       "[variables('cacheName')]",
						"location":   "eastus",
						"sku":        map[string]any{"name": "Basic", "family": "C", "capacity": 1},
						"properties": map[string]any{"minimumTlsVersion": "1.2"},
					},
				},
				"outputs": map[string]any{
					"primaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.Cache/Redis', variables('cacheName')), '2024-11-01').primaryKey]",
					},
					"secondaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(variables('cacheName'), '2024-11-01').secondaryKey]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-redis-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create Redis listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected Redis listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["primaryKey"].(map[string]any)["value"] != "cloudmock-cache-a-primary" {
		t.Fatalf("expected Redis listKeys primary output to return deterministic primary key, got %v", outputs["primaryKey"])
	}
	if outputs["secondaryKey"].(map[string]any)["value"] != "cloudmock-cache-a-secondary" {
		t.Fatalf("expected Redis listKeys secondary output to return deterministic secondary key, got %v", outputs["secondaryKey"])
	}
}

func TestDeploymentListKeysReturnsEventHubAuthorizationRuleKeysInOutputs(t *testing.T) {
	eventHubSvc := eventhub.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(eventHubSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"namespaceName": "namespace-a",
					"hubName":       "hub-a",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.EventHub/namespaces",
						"apiVersion": "2026-01-01",
						"name":       "[variables('namespaceName')]",
						"location":   "eastus",
						"sku":        map[string]any{"name": "Standard", "tier": "Standard", "capacity": 1},
					},
					map[string]any{
						"type":       "Microsoft.EventHub/namespaces/authorizationRules",
						"apiVersion": "2026-01-01",
						"name":       "[format('{0}/ns-rule-a', variables('namespaceName'))]",
						"properties": map[string]any{"rights": []any{"Listen", "Send"}},
					},
					map[string]any{
						"type":       "Microsoft.EventHub/namespaces/eventhubs",
						"apiVersion": "2026-01-01",
						"name":       "[format('{0}/{1}', variables('namespaceName'), variables('hubName'))]",
						"properties": map[string]any{"partitionCount": 2},
					},
					map[string]any{
						"type":       "Microsoft.EventHub/namespaces/eventhubs/authorizationRules",
						"apiVersion": "2026-01-01",
						"name":       "[format('{0}/{1}/hub-rule-a', variables('namespaceName'), variables('hubName'))]",
						"properties": map[string]any{"rights": []any{"Listen", "Send"}},
					},
				},
				"outputs": map[string]any{
					"namespaceKeyName": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.EventHub/namespaces/authorizationRules', variables('namespaceName'), 'ns-rule-a'), '2026-01-01').keyName]",
					},
					"namespacePrimaryConnectionString": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.EventHub/namespaces/authorizationRules', variables('namespaceName'), 'ns-rule-a'), '2026-01-01').primaryConnectionString]",
					},
					"namespacePrimaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.EventHub/namespaces/authorizationRules', variables('namespaceName'), 'ns-rule-a'), '2026-01-01').primaryKey]",
					},
					"hubPrimaryConnectionString": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.EventHub/namespaces/eventhubs/authorizationRules', variables('namespaceName'), variables('hubName'), 'hub-rule-a'), '2026-01-01').primaryConnectionString]",
					},
					"hubSecondaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.EventHub/namespaces/eventhubs/authorizationRules', variables('namespaceName'), variables('hubName'), 'hub-rule-a'), '2026-01-01').secondaryKey]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-eventhub-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create Event Hubs listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected Event Hubs listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	namespaceKey, ok := outputs["namespacePrimaryKey"].(map[string]any)["value"].(string)
	if outputs["namespaceKeyName"].(map[string]any)["value"] != "ns-rule-a" {
		t.Fatalf("expected namespace listKeys keyName output to return rule name, got %v", outputs["namespaceKeyName"])
	}
	if !ok || namespaceKey == "" {
		t.Fatalf("expected namespace listKeys primary key output to be populated, got %v", outputs["namespacePrimaryKey"])
	}
	expectedNamespaceConnection := "Endpoint=sb://namespace-a.servicebus.windows.net/;SharedAccessKeyName=ns-rule-a;SharedAccessKey=" + namespaceKey
	if outputs["namespacePrimaryConnectionString"].(map[string]any)["value"] != expectedNamespaceConnection {
		t.Fatalf("expected namespace listKeys connection string to match key output, got %v", outputs["namespacePrimaryConnectionString"])
	}
	hubSecondaryKey, ok := outputs["hubSecondaryKey"].(map[string]any)["value"].(string)
	if !ok || hubSecondaryKey == "" {
		t.Fatalf("expected event hub listKeys secondary key output to be populated, got %v", outputs["hubSecondaryKey"])
	}
	hubConnection, ok := outputs["hubPrimaryConnectionString"].(map[string]any)["value"].(string)
	if !ok {
		t.Fatalf("expected event hub listKeys connection string output to be a string, got %v", outputs["hubPrimaryConnectionString"])
	}
	if !strings.Contains(hubConnection, "Endpoint=sb://namespace-a.servicebus.windows.net/;SharedAccessKeyName=hub-rule-a;SharedAccessKey=") ||
		!strings.HasSuffix(hubConnection, ";EntityPath=hub-a") {
		t.Fatalf("expected event hub listKeys connection string to include EntityPath, got %v", outputs["hubPrimaryConnectionString"])
	}
}

func TestDeploymentListKeysReturnsServiceBusAuthorizationRuleKeysInOutputs(t *testing.T) {
	serviceBusSvc := servicebus.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(serviceBusSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"namespaceName": "sb-namespace-a",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces",
						"apiVersion": "2024-01-01",
						"name":       "[variables('namespaceName')]",
						"location":   "eastus",
						"sku":        map[string]any{"name": "Standard", "tier": "Standard"},
					},
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces/AuthorizationRules",
						"apiVersion": "2024-01-01",
						"name":       "[format('{0}/send-listen-rule', variables('namespaceName'))]",
						"properties": map[string]any{"rights": []any{"Listen", "Send"}},
					},
				},
				"outputs": map[string]any{
					"keyName": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/AuthorizationRules', variables('namespaceName'), 'send-listen-rule'), '2024-01-01').keyName]",
					},
					"primaryConnectionString": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/AuthorizationRules', variables('namespaceName'), 'send-listen-rule'), '2024-01-01').primaryConnectionString]",
					},
					"primaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/AuthorizationRules', variables('namespaceName'), 'send-listen-rule'), '2024-01-01').primaryKey]",
					},
					"secondaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/AuthorizationRules', variables('namespaceName'), 'send-listen-rule'), '2024-01-01').secondaryKey]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-servicebus-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create Service Bus listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected Service Bus listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	primaryKey, ok := outputs["primaryKey"].(map[string]any)["value"].(string)
	if outputs["keyName"].(map[string]any)["value"] != "send-listen-rule" {
		t.Fatalf("expected Service Bus listKeys keyName output to return rule name, got %v", outputs["keyName"])
	}
	if !ok || primaryKey == "" {
		t.Fatalf("expected Service Bus listKeys primary key output to be populated, got %v", outputs["primaryKey"])
	}
	secondaryKey, ok := outputs["secondaryKey"].(map[string]any)["value"].(string)
	if !ok || secondaryKey == "" || secondaryKey == primaryKey {
		t.Fatalf("expected Service Bus listKeys secondary key output to be populated and distinct, got %v", outputs["secondaryKey"])
	}
	expectedConnection := "Endpoint=sb://sb-namespace-a.servicebus.windows.net/;SharedAccessKeyName=send-listen-rule;SharedAccessKey=" + primaryKey
	if outputs["primaryConnectionString"].(map[string]any)["value"] != expectedConnection {
		t.Fatalf("expected Service Bus listKeys connection string to match key output, got %v", outputs["primaryConnectionString"])
	}
}

func TestDeploymentListKeysReturnsServiceBusEntityAuthorizationRuleKeysInOutputs(t *testing.T) {
	serviceBusSvc := servicebus.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(serviceBusSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"namespaceName": "sb-entity-ns",
					"queueName":     "queue-a",
					"topicName":     "topic-a",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces",
						"apiVersion": "2024-01-01",
						"name":       "[variables('namespaceName')]",
						"location":   "eastus",
						"sku":        map[string]any{"name": "Standard", "tier": "Standard"},
					},
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces/queues",
						"apiVersion": "2024-01-01",
						"name":       "[format('{0}/{1}', variables('namespaceName'), variables('queueName'))]",
						"properties": map[string]any{},
					},
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces/queues/AuthorizationRules",
						"apiVersion": "2024-01-01",
						"name":       "[format('{0}/{1}/queue-send', variables('namespaceName'), variables('queueName'))]",
						"properties": map[string]any{"rights": []any{"Send"}},
					},
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces/topics",
						"apiVersion": "2024-01-01",
						"name":       "[format('{0}/{1}', variables('namespaceName'), variables('topicName'))]",
						"properties": map[string]any{},
					},
					map[string]any{
						"type":       "Microsoft.ServiceBus/namespaces/topics/AuthorizationRules",
						"apiVersion": "2024-01-01",
						"name":       "[format('{0}/{1}/topic-listen', variables('namespaceName'), variables('topicName'))]",
						"properties": map[string]any{"rights": []any{"Listen"}},
					},
				},
				"outputs": map[string]any{
					"queueKeyName": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/queues/AuthorizationRules', variables('namespaceName'), variables('queueName'), 'queue-send'), '2024-01-01').keyName]",
					},
					"queuePrimaryConnectionString": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/queues/AuthorizationRules', variables('namespaceName'), variables('queueName'), 'queue-send'), '2024-01-01').primaryConnectionString]",
					},
					"queuePrimaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/queues/AuthorizationRules', variables('namespaceName'), variables('queueName'), 'queue-send'), '2024-01-01').primaryKey]",
					},
					"topicKeyName": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/topics/AuthorizationRules', variables('namespaceName'), variables('topicName'), 'topic-listen'), '2024-01-01').keyName]",
					},
					"topicPrimaryConnectionString": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/topics/AuthorizationRules', variables('namespaceName'), variables('topicName'), 'topic-listen'), '2024-01-01').primaryConnectionString]",
					},
					"topicSecondaryKey": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.ServiceBus/namespaces/topics/AuthorizationRules', variables('namespaceName'), variables('topicName'), 'topic-listen'), '2024-01-01').secondaryKey]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-servicebus-entity-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create Service Bus entity listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected Service Bus entity listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	queuePrimaryKey, ok := outputs["queuePrimaryKey"].(map[string]any)["value"].(string)
	if outputs["queueKeyName"].(map[string]any)["value"] != "queue-send" {
		t.Fatalf("expected queue listKeys keyName output to return rule name, got %v", outputs["queueKeyName"])
	}
	if !ok || queuePrimaryKey == "" {
		t.Fatalf("expected queue listKeys primary key output to be populated, got %v", outputs["queuePrimaryKey"])
	}
	expectedQueueConnection := "Endpoint=sb://sb-entity-ns.servicebus.windows.net/;SharedAccessKeyName=queue-send;SharedAccessKey=" + queuePrimaryKey + ";EntityPath=queue-a"
	if outputs["queuePrimaryConnectionString"].(map[string]any)["value"] != expectedQueueConnection {
		t.Fatalf("expected queue listKeys connection string to include EntityPath, got %v", outputs["queuePrimaryConnectionString"])
	}
	topicSecondaryKey, ok := outputs["topicSecondaryKey"].(map[string]any)["value"].(string)
	if outputs["topicKeyName"].(map[string]any)["value"] != "topic-listen" {
		t.Fatalf("expected topic listKeys keyName output to return rule name, got %v", outputs["topicKeyName"])
	}
	if !ok || topicSecondaryKey == "" {
		t.Fatalf("expected topic listKeys secondary key output to be populated, got %v", outputs["topicSecondaryKey"])
	}
	topicConnection, ok := outputs["topicPrimaryConnectionString"].(map[string]any)["value"].(string)
	if !ok || !strings.Contains(topicConnection, "Endpoint=sb://sb-entity-ns.servicebus.windows.net/;SharedAccessKeyName=topic-listen;SharedAccessKey=") || !strings.HasSuffix(topicConnection, ";EntityPath=topic-a") {
		t.Fatalf("expected topic listKeys connection string to include EntityPath, got %v", outputs["topicPrimaryConnectionString"])
	}
}

func TestDeploymentListKeysReturnsEventGridTopicKeysInOutputs(t *testing.T) {
	eventGridSvc := eventgrid.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(eventGridSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"topicName": "topic-a",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.EventGrid/topics",
						"apiVersion": "2025-02-15",
						"name":       "[variables('topicName')]",
						"location":   "eastus",
						"properties": map[string]any{"inputSchema": "EventGridSchema"},
					},
				},
				"outputs": map[string]any{
					"key1": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.EventGrid/topics', variables('topicName')), '2025-02-15').key1]",
					},
					"key2": map[string]any{
						"type":  "string",
						"value": "[listKeys(variables('topicName'), '2025-02-15').key2]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-eventgrid-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create Event Grid listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected Event Grid listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	key1, ok := outputs["key1"].(map[string]any)["value"].(string)
	if !ok || key1 == "" {
		t.Fatalf("expected Event Grid listKeys key1 output to be populated, got %v", outputs["key1"])
	}
	key2, ok := outputs["key2"].(map[string]any)["value"].(string)
	if !ok || key2 == "" {
		t.Fatalf("expected Event Grid listKeys key2 output to be populated, got %v", outputs["key2"])
	}
	if key1 == key2 {
		t.Fatalf("expected Event Grid listKeys outputs to return distinct keys, got key1=%q key2=%q", key1, key2)
	}
}

func TestDeploymentListCredentialsReturnsContainerRegistryCredentialsInOutputs(t *testing.T) {
	registrySvc := containerregistry.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(registrySvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"registryName": "acra",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.ContainerRegistry/registries",
						"apiVersion": "2025-11-01",
						"name":       "[variables('registryName')]",
						"location":   "eastus",
						"sku":        map[string]any{"name": "Basic"},
						"properties": map[string]any{"adminUserEnabled": true},
					},
				},
				"outputs": map[string]any{
					"username": map[string]any{
						"type":  "string",
						"value": "[listCredentials(resourceId('Microsoft.ContainerRegistry/registries', variables('registryName')), '2025-11-01').username]",
					},
					"passwordName": map[string]any{
						"type":  "string",
						"value": "[listCredentials(variables('registryName'), '2025-11-01').passwords[0].name]",
					},
					"passwordValue": map[string]any{
						"type":  "string",
						"value": "[listCredentials(resourceId('Microsoft.ContainerRegistry/registries', variables('registryName')), '2025-11-01').passwords[0].value]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-acr-list-credentials?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create Container Registry listCredentials deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected Container Registry listCredentials deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["username"].(map[string]any)["value"] != "acra" {
		t.Fatalf("expected listCredentials username output to return registry name, got %v", outputs["username"])
	}
	if outputs["passwordName"].(map[string]any)["value"] != "password" {
		t.Fatalf("expected listCredentials password name output to return password, got %v", outputs["passwordName"])
	}
	if outputs["passwordValue"].(map[string]any)["value"] != "cm-acr-acra-password-1" {
		t.Fatalf("expected listCredentials password value output to return deterministic first credential, got %v", outputs["passwordValue"])
	}
}

func TestDeploymentListKeysReturnsAppConfigurationStoreKeysInOutputs(t *testing.T) {
	appConfigSvc := appconfiguration.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(appConfigSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"storeName": "cfgstore",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.AppConfiguration/configurationStores",
						"apiVersion": "2024-06-01",
						"name":       "[variables('storeName')]",
						"location":   "eastus",
						"sku":        map[string]any{"name": "Standard"},
						"properties": map[string]any{"disableLocalAuth": false},
					},
				},
				"outputs": map[string]any{
					"primaryName": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.AppConfiguration/configurationStores', variables('storeName')), '2024-06-01').value[0].name]",
					},
					"primaryConnectionString": map[string]any{
						"type":  "string",
						"value": "[listKeys(resourceId('Microsoft.AppConfiguration/configurationStores', variables('storeName')), '2024-06-01').value[0].connectionString]",
					},
					"readOnly": map[string]any{
						"type":  "bool",
						"value": "[listKeys(variables('storeName'), '2024-06-01').value[2].readOnly]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-appconfig-list-keys?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create App Configuration listKeys deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected App Configuration listKeys deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["primaryName"].(map[string]any)["value"] != "Primary" {
		t.Fatalf("expected App Configuration primary key output to return Primary, got %v", outputs["primaryName"])
	}
	expectedConnection := "Endpoint=https://cfgstore.azconfig.io;Id=cfgstore-primary;Secret=cm-appconfig-cfgstore-primary"
	if outputs["primaryConnectionString"].(map[string]any)["value"] != expectedConnection {
		t.Fatalf("expected App Configuration connection string output %q, got %v", expectedConnection, outputs["primaryConnectionString"])
	}
	if outputs["readOnly"].(map[string]any)["value"] != true {
		t.Fatalf("expected App Configuration read-only key output to return true, got %v", outputs["readOnly"])
	}
}

func TestDeploymentUsesTemplateParameterDefaultValues(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"namePrefix": map[string]any{
						"type":         "string",
						"defaultValue": "defaultstore",
					},
					"accountName": map[string]any{
						"type":         "string",
						"defaultValue": "[concat(parameters('namePrefix'), 'acct')]",
					},
					"tags": map[string]any{
						"type": "object",
						"defaultValue": map[string]any{
							"env": "defaulted",
						},
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[parameters('accountName')]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"tags":       "[parameters('tags')]",
					},
				},
				"outputs": map[string]any{
					"accountName": map[string]any{
						"type":  "string",
						"value": "[parameters('accountName')]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-param-defaults?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create parameter-default deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected parameter-default deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["accountName"].(map[string]any)["value"] != "defaultstoreacct" {
		t.Fatalf("expected defaulted accountName output, got %v", outputs["accountName"])
	}

	storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/defaultstoreacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get default-parameter storage account returned error: %v", err)
	}
	if storageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected default-parameter storage account status 200, got %d body=%s", storageResp.StatusCode, string(storageResp.RawBody))
	}
	account := decodeARMResponse(t, storageResp)
	tags := account["tags"].(map[string]any)
	if tags["env"] != "defaulted" {
		t.Fatalf("expected default object parameter tags, got %v", tags)
	}
}

func TestDeploymentResolvesParameterObjectPropertyAccess(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"account": map[string]any{
						"type": "object",
						"defaultValue": map[string]any{
							"name": "objectstoreacct",
							"tags": map[string]any{
								"env": "object-default",
							},
						},
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[parameters('account').name]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"tags":       "[parameters('account').tags]",
					},
				},
				"outputs": map[string]any{
					"accountName": map[string]any{
						"type":  "string",
						"value": "[parameters('account').name]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-param-object?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create parameter-object deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected parameter-object deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	if outputs["accountName"].(map[string]any)["value"] != "objectstoreacct" {
		t.Fatalf("expected object parameter property output, got %v", outputs["accountName"])
	}

	storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/objectstoreacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get object-parameter storage account returned error: %v", err)
	}
	if storageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected object-parameter storage account status 200, got %d body=%s", storageResp.StatusCode, string(storageResp.RawBody))
	}
	account := decodeARMResponse(t, storageResp)
	tags := account["tags"].(map[string]any)
	if tags["env"] != "object-default" {
		t.Fatalf("expected object parameter tags to resolve, got %v", tags)
	}
}

func TestDeploymentExpandsResourceCopyLoop(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"storageCount": map[string]any{
						"type":         "int",
						"defaultValue": 2,
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[concat('copyacct', copyIndex())]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"copy": map[string]any{
							"name":  "storagecopy",
							"count": "[parameters('storageCount')]",
						},
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-copy?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create copy deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected copy deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputResources := props["outputResources"].([]any)
	if len(outputResources) != 2 {
		t.Fatalf("expected two copied output resources, got %v", outputResources)
	}
	for index, value := range outputResources {
		resource := value.(map[string]any)
		expectedName := "copyacct" + strconv.Itoa(index)
		if resource["name"] != expectedName {
			t.Fatalf("expected copied resource %d name %s, got %v", index, expectedName, resource)
		}
		storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/"+expectedName+"?api-version=2024-01-01", nil))
		if err != nil {
			t.Fatalf("get copied storage account %s returned error: %v", expectedName, err)
		}
		if storageResp.StatusCode != http.StatusOK {
			t.Fatalf("expected copied storage account %s status 200, got %d body=%s", expectedName, storageResp.StatusCode, string(storageResp.RawBody))
		}
	}
}

func TestDeploymentExpandsArrayDrivenResourceCopyLoop(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"storageNames": map[string]any{
						"type": "array",
						"defaultValue": []any{
							"arraycopya",
							"arraycopyb",
						},
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[format('{0}acct', parameters('storageNames')[copyIndex()])]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"copy": map[string]any{
							"name":  "storagecopy",
							"count": "[length(parameters('storageNames'))]",
						},
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-array-copy?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create array-copy deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected array-copy deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputResources, ok := props["outputResources"].([]any)
	if !ok {
		t.Fatalf("expected array-copy deployment to include outputResources, got %v", props["outputResources"])
	}
	if len(outputResources) != 2 {
		t.Fatalf("expected two array-copied output resources, got %v", outputResources)
	}
	for index, expectedName := range []string{"arraycopyaacct", "arraycopybacct"} {
		resource := outputResources[index].(map[string]any)
		if resource["name"] != expectedName {
			t.Fatalf("expected array-copied resource %d name %s, got %v", index, expectedName, resource)
		}
		storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/"+expectedName+"?api-version=2024-01-01", nil))
		if err != nil {
			t.Fatalf("get array-copied storage account %s returned error: %v", expectedName, err)
		}
		if storageResp.StatusCode != http.StatusOK {
			t.Fatalf("expected array-copied storage account %s status 200, got %d body=%s", expectedName, storageResp.StatusCode, string(storageResp.RawBody))
		}
	}
}

func TestDeploymentExpandsRangeDrivenResourceCopyLoop(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"storageCount": map[string]any{
						"type":         "int",
						"defaultValue": 2,
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[format('rangecopy{0}{1}', range(0, parameters('storageCount'))[copyIndex()], uniqueString(resourceGroup().id))]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"copy": map[string]any{
							"name":  "storagecopy",
							"count": "[length(range(0, parameters('storageCount')))]",
						},
					},
				},
				"outputs": map[string]any{
					"suffix": map[string]any{
						"type":  "string",
						"value": "[uniqueString(resourceGroup().id)]",
					},
					"indexes": map[string]any{
						"type":  "array",
						"value": "[range(0, parameters('storageCount'))]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-range-copy?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create range-copy deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected range-copy deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	suffix := outputs["suffix"].(map[string]any)["value"].(string)
	if len(suffix) != 13 {
		t.Fatalf("expected uniqueString suffix length 13, got %q", suffix)
	}
	indexes := outputs["indexes"].(map[string]any)["value"].([]any)
	if len(indexes) != 2 || indexes[0] != float64(0) || indexes[1] != float64(1) {
		t.Fatalf("expected range output [0, 1], got %v", indexes)
	}

	outputResources := props["outputResources"].([]any)
	if len(outputResources) != 2 {
		t.Fatalf("expected two range-copied output resources, got %v", outputResources)
	}
	for index, expectedPrefix := range []string{"rangecopy0", "rangecopy1"} {
		resource := outputResources[index].(map[string]any)
		name := resource["name"].(string)
		if !strings.HasPrefix(name, expectedPrefix) || !strings.HasSuffix(name, suffix) {
			t.Fatalf("expected range-copied resource %d name with prefix %s and suffix %s, got %v", index, expectedPrefix, suffix, resource)
		}
		storageResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/"+name+"?api-version=2024-01-01", nil))
		if err != nil {
			t.Fatalf("get range-copied storage account %s returned error: %v", name, err)
		}
		if storageResp.StatusCode != http.StatusOK {
			t.Fatalf("expected range-copied storage account %s status 200, got %d body=%s", name, storageResp.StatusCode, string(storageResp.RawBody))
		}
	}
}

func TestDeploymentExpandsOutputCopyLoop(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"storageCount": map[string]any{
						"type":         "int",
						"defaultValue": 2,
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[format('outcopy{0}', copyIndex())]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
						"copy": map[string]any{
							"name":  "storagecopy",
							"count": "[parameters('storageCount')]",
						},
					},
				},
				"outputs": map[string]any{
					"storageInfo": map[string]any{
						"type": "array",
						"copy": map[string]any{
							"count": "[parameters('storageCount')]",
							"input": map[string]any{
								"name": "[format('outcopy{0}', copyIndex())]",
								"id":   "[resourceId('Microsoft.Storage/storageAccounts', format('outcopy{0}', copyIndex()))]",
							},
						},
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-output-copy?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create output-copy deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected output-copy deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	storageInfo := outputs["storageInfo"].(map[string]any)
	if storageInfo["type"] != "array" {
		t.Fatalf("expected output copy type array, got %v", storageInfo)
	}
	values, ok := storageInfo["value"].([]any)
	if !ok {
		t.Fatalf("expected output copy value array, got %v", storageInfo["value"])
	}
	if len(values) != 2 {
		t.Fatalf("expected two output copy entries, got %v", values)
	}
	for index, value := range values {
		entry := value.(map[string]any)
		expectedName := "outcopy" + strconv.Itoa(index)
		expectedID := "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/" + expectedName
		if entry["name"] != expectedName || entry["id"] != expectedID {
			t.Fatalf("expected output copy entry %d name/id %s/%s, got %v", index, expectedName, expectedID, entry)
		}
	}
}

func TestDeploymentExpandsPropertyCopyLoop(t *testing.T) {
	svc := resources.New()

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"subnetNames": map[string]any{
						"type": "array",
						"defaultValue": []any{
							"subnet-a",
							"subnet-b",
						},
					},
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Network/virtualNetworks",
						"apiVersion": "2025-01-01",
						"name":       "copyvnet",
						"location":   "westus2",
						"properties": map[string]any{
							"addressSpace": map[string]any{
								"addressPrefixes": []any{"10.0.0.0/16"},
							},
							"copy": []any{
								map[string]any{
									"name":  "subnets",
									"count": "[length(parameters('subnetNames'))]",
									"input": map[string]any{
										"name": "[parameters('subnetNames')[copyIndex('subnets')]]",
										"properties": map[string]any{
											"addressPrefix": "[format('10.0.{0}.0/24', copyIndex('subnets'))]",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-property-copy?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create property-copy deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected property-copy deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputResources := props["outputResources"].([]any)
	if len(outputResources) != 1 {
		t.Fatalf("expected one property-copy output resource, got %v", outputResources)
	}
	resource := outputResources[0].(map[string]any)
	resourceProps := resource["properties"].(map[string]any)
	if _, hasCopy := resourceProps["copy"]; hasCopy {
		t.Fatalf("expected property copy block to be removed from resolved properties, got %v", resourceProps)
	}
	subnets, ok := resourceProps["subnets"].([]any)
	if !ok {
		t.Fatalf("expected property copy to create subnets array, got %v", resourceProps["subnets"])
	}
	if len(subnets) != 2 {
		t.Fatalf("expected two property-copied subnets, got %v", subnets)
	}
	for index, expectedName := range []string{"subnet-a", "subnet-b"} {
		subnet := subnets[index].(map[string]any)
		subnetProps := subnet["properties"].(map[string]any)
		expectedPrefix := "10.0." + strconv.Itoa(index) + ".0/24"
		if subnet["name"] != expectedName || subnetProps["addressPrefix"] != expectedPrefix {
			t.Fatalf("expected property-copied subnet %d name/prefix %s/%s, got %v", index, expectedName, expectedPrefix, subnet)
		}
	}
}

func TestDeploymentHonorsResourceConditions(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"parameters": map[string]any{
					"deploySkipped": map[string]any{
						"type":         "bool",
						"defaultValue": false,
					},
					"newOrExisting": map[string]any{
						"type":         "string",
						"defaultValue": "new",
					},
				},
				"resources": []any{
					map[string]any{
						"condition":  "[parameters('deploySkipped')]",
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "skippedconditionacct",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
					},
					map[string]any{
						"condition":  "[equals(parameters('newOrExisting'), 'new')]",
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "createdconditionacct",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-conditions?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create conditional deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected conditional deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputResources := props["outputResources"].([]any)
	if len(outputResources) != 1 || outputResources[0].(map[string]any)["name"] != "createdconditionacct" {
		t.Fatalf("expected only condition-true resource in outputResources, got %v", outputResources)
	}

	skippedResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/skippedconditionacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get skipped conditional storage account returned error: %v", err)
	}
	if skippedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected skipped conditional storage account status 404, got %d body=%s", skippedResp.StatusCode, string(skippedResp.RawBody))
	}

	createdResp, err := storageSvc.HandleRequest(armCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/createdconditionacct?api-version=2024-01-01", nil))
	if err != nil {
		t.Fatalf("get created conditional storage account returned error: %v", err)
	}
	if createdResp.StatusCode != http.StatusOK {
		t.Fatalf("expected created conditional storage account status 200, got %d body=%s", createdResp.StatusCode, string(createdResp.RawBody))
	}
}

func TestDeploymentResolvesResourceIDExpressions(t *testing.T) {
	storageSvc := storage.New()
	svc := resources.New()
	svc.SetTemplateProvisioner(storageSvc)

	deployment := map[string]any{
		"properties": map[string]any{
			"mode": "Incremental",
			"template": map[string]any{
				"variables": map[string]any{
					"storageName": "depresourceidstore",
					"vaultName":   "depresourceidvault",
				},
				"resources": []any{
					map[string]any{
						"type":       "Microsoft.Mock/widgets",
						"apiVersion": "2024-01-01",
						"name":       "[variables('vaultName')]",
						"location":   "westus2",
						"dependsOn": []any{
							"[resourceId('Microsoft.Storage/storageAccounts', variables('storageName'))]",
						},
						"properties": map[string]any{
							"storageId": "[resourceId('Microsoft.Storage/storageAccounts', variables('storageName'))]",
						},
					},
					map[string]any{
						"type":       "Microsoft.Storage/storageAccounts",
						"apiVersion": "2024-01-01",
						"name":       "[variables('storageName')]",
						"location":   "westus2",
						"kind":       "StorageV2",
						"sku":        map[string]any{"name": "Standard_LRS"},
					},
				},
				"outputs": map[string]any{
					"storageId": map[string]any{
						"type":  "string",
						"value": "[resourceId('Microsoft.Storage/storageAccounts', variables('storageName'))]",
					},
				},
			},
		},
	}

	createResp, err := svc.HandleRequest(armCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Resources/deployments/dep-resource-id?api-version=2021-04-01", deployment))
	if err != nil {
		t.Fatalf("create resourceId deployment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected resourceId deployment create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeARMResponse(t, createResp)
	props := created["properties"].(map[string]any)
	outputs := props["outputs"].(map[string]any)
	expectedStorageID := "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/depresourceidstore"
	if outputs["storageId"].(map[string]any)["value"] != expectedStorageID {
		t.Fatalf("expected resourceId output to resolve to %s, got %v", expectedStorageID, outputs["storageId"])
	}
	outputResources := props["outputResources"].([]any)
	if len(outputResources) != 2 {
		t.Fatalf("expected two output resources, got %v", outputResources)
	}
	if outputResources[0].(map[string]any)["type"] != "Microsoft.Storage/storageAccounts" {
		t.Fatalf("expected resourceId dependsOn to order storage before vault, got %v", outputResources)
	}
	mockResource := outputResources[1].(map[string]any)
	if mockResource["type"] != "Microsoft.Mock/widgets" {
		t.Fatalf("expected unresolved mock resource as second output resource, got %v", outputResources)
	}
	properties := mockResource["properties"].(map[string]any)
	if properties["storageId"] != expectedStorageID {
		t.Fatalf("expected nested resourceId property to resolve, got %v", properties)
	}
}
