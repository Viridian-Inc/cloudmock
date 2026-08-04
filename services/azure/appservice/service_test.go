package appservice

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestFunctionAppPlanSiteAndFunctionMetadataLifecycle(t *testing.T) {
	svc := New()

	planURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a?api-version=2024-04-01"
	planPayload := []byte(`{
		"location":"eastus",
		"kind":"functionapp",
		"sku":{"name":"Y1","tier":"Dynamic"},
		"tags":{"env":"test"},
		"properties":{"reserved":true}
	}`)

	createPlanResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, planURL, planPayload))
	if err != nil {
		t.Fatalf("create app service plan returned error: %v", err)
	}
	if createPlanResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create app service plan status 201, got %d; body=%s", createPlanResp.StatusCode, string(createPlanResp.RawBody))
	}
	createdPlan := decodeAppServiceResponse(t, createPlanResp)
	if createdPlan["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a" {
		t.Fatalf("unexpected app service plan id: %v", createdPlan["id"])
	}
	if createdPlan["name"] != "plan-a" || createdPlan["type"] != "Microsoft.Web/serverfarms" || createdPlan["location"] != "eastus" {
		t.Fatalf("unexpected app service plan identity fields: %v", createdPlan)
	}
	planSku := createdPlan["sku"].(map[string]any)
	if planSku["name"] != "Y1" || planSku["tier"] != "Dynamic" {
		t.Fatalf("unexpected app service plan sku: %v", planSku)
	}
	planProps := createdPlan["properties"].(map[string]any)
	if planProps["provisioningState"] != "Succeeded" || planProps["status"] != "Ready" {
		t.Fatalf("unexpected app service plan state: %v", planProps)
	}

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-a?api-version=2024-04-01"
	sitePayload := []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"test"},
		"properties":{
			"serverFarmId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a",
			"httpsOnly":true,
			"siteConfig":{"appSettings":[{"name":"FUNCTIONS_WORKER_RUNTIME","value":"node"}]},
			"functions":[{"name":"HttpTrigger","config":{"bindings":[{"type":"httpTrigger","direction":"in","name":"req"}]}}]
		}
	}`)

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, sitePayload))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create function app status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}
	createdSite := decodeAppServiceResponse(t, createSiteResp)
	if createdSite["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-a" {
		t.Fatalf("unexpected function app id: %v", createdSite["id"])
	}
	if createdSite["name"] != "func-a" || createdSite["type"] != "Microsoft.Web/sites" || createdSite["kind"] != "functionapp,linux" {
		t.Fatalf("unexpected function app identity fields: %v", createdSite)
	}
	siteProps := createdSite["properties"].(map[string]any)
	if siteProps["state"] != "Running" || siteProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected function app state: %v", siteProps)
	}
	if siteProps["defaultHostName"] != "func-a.azurewebsites.net" {
		t.Fatalf("unexpected function app default host name: %v", siteProps["defaultHostName"])
	}

	getSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, siteURL, nil))
	if err != nil {
		t.Fatalf("get function app returned error: %v", err)
	}
	if getSiteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get function app status 200, got %d; body=%s", getSiteResp.StatusCode, string(getSiteResp.RawBody))
	}

	listSitesResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list function apps returned error: %v", err)
	}
	listedSites := decodeAppServiceResponse(t, listSitesResp)
	siteValues := listedSites["value"].([]any)
	if len(siteValues) != 1 {
		t.Fatalf("expected one function app in list, got %d in %v", len(siteValues), listedSites)
	}

	listFunctionsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-a/functions?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list functions returned error: %v", err)
	}
	if listFunctionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list functions status 200, got %d; body=%s", listFunctionsResp.StatusCode, string(listFunctionsResp.RawBody))
	}
	listedFunctions := decodeAppServiceResponse(t, listFunctionsResp)
	functionValues := listedFunctions["value"].([]any)
	if len(functionValues) != 1 {
		t.Fatalf("expected one function metadata record, got %d in %v", len(functionValues), listedFunctions)
	}
	function := functionValues[0].(map[string]any)
	if function["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-a/functions/HttpTrigger" {
		t.Fatalf("unexpected function id: %v", function["id"])
	}
	if function["name"] != "func-a/HttpTrigger" || function["type"] != "Microsoft.Web/sites/functions" {
		t.Fatalf("unexpected function identity fields: %v", function)
	}
	functionProps := function["properties"].(map[string]any)
	if functionProps["name"] != "HttpTrigger" {
		t.Fatalf("unexpected function properties: %v", functionProps)
	}

	deleteSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, siteURL, nil))
	if err != nil {
		t.Fatalf("delete function app returned error: %v", err)
	}
	if deleteSiteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete function app status 202, got %d; body=%s", deleteSiteResp.StatusCode, string(deleteSiteResp.RawBody))
	}

	getDeletedSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, siteURL, nil))
	if err != nil {
		t.Fatalf("get deleted function app returned error: %v", err)
	}
	if getDeletedSiteResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted function app status 404, got %d; body=%s", getDeletedSiteResp.StatusCode, string(getDeletedSiteResp.RawBody))
	}
}

func TestAppServicePlanPatchUpdatesSkuTagsKindAndProperties(t *testing.T) {
	svc := New()
	planURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-patch?api-version=2024-04-01"

	createPlanResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, planURL, []byte(`{
		"location":"eastus",
		"kind":"linux",
		"sku":{"name":"B1","tier":"Basic"},
		"tags":{"env":"dev"},
		"properties":{
			"reserved":true,
			"perSiteScaling":false
		}
	}`)))
	if err != nil {
		t.Fatalf("create app service plan returned error: %v", err)
	}
	if createPlanResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected app service plan create status 201, got %d; body=%s", createPlanResp.StatusCode, string(createPlanResp.RawBody))
	}

	patchPlanResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, planURL, []byte(`{
		"kind":"app,linux",
		"sku":{"name":"P1v3","tier":"PremiumV3","capacity":2},
		"tags":{"env":"prod","owner":"platform"},
		"properties":{
			"perSiteScaling":true,
			"zoneRedundant":true,
			"targetWorkerCount":2
		}
	}`)))
	if err != nil {
		t.Fatalf("patch app service plan returned error: %v", err)
	}
	if patchPlanResp.StatusCode != http.StatusOK {
		t.Fatalf("expected app service plan patch status 200, got %d; body=%s", patchPlanResp.StatusCode, string(patchPlanResp.RawBody))
	}

	updated := decodeAppServiceResponse(t, patchPlanResp)
	if updated["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-patch" || updated["name"] != "plan-patch" || updated["type"] != "Microsoft.Web/serverfarms" {
		t.Fatalf("unexpected patched app service plan identity fields: %v", updated)
	}
	if updated["location"] != "eastus" || updated["kind"] != "app,linux" {
		t.Fatalf("expected patch to preserve location and update kind, got %v", updated)
	}
	sku := updated["sku"].(map[string]any)
	if sku["name"] != "P1v3" || sku["tier"] != "PremiumV3" || sku["capacity"].(float64) != 2 {
		t.Fatalf("expected patched sku, got %v", sku)
	}
	tags := updated["tags"].(map[string]any)
	if tags["env"] != "prod" || tags["owner"] != "platform" {
		t.Fatalf("expected replaced tags, got %v", tags)
	}
	props := updated["properties"].(map[string]any)
	if props["reserved"] != true || props["perSiteScaling"] != true || props["zoneRedundant"] != true || props["targetWorkerCount"].(float64) != 2 {
		t.Fatalf("expected patched and preserved properties, got %v", props)
	}
	if props["provisioningState"] != "Succeeded" || props["status"] != "Ready" {
		t.Fatalf("expected deterministic plan state to persist, got %v", props)
	}

	persistedResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, planURL, nil))
	if err != nil {
		t.Fatalf("get patched app service plan returned error: %v", err)
	}
	persisted := decodeAppServiceResponse(t, persistedResp)
	persistedProps := persisted["properties"].(map[string]any)
	if persistedProps["perSiteScaling"] != true || persistedProps["reserved"] != true {
		t.Fatalf("expected patched app service plan properties to persist, got %v", persistedProps)
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/missing?api-version=2024-04-01", []byte(`{"properties":{"perSiteScaling":true}}`)))
	if err != nil {
		t.Fatalf("patch missing app service plan returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing plan patch status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestAppServiceSlotLifecycle(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot?api-version=2024-04-01"
	slotURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot/slots/staging?api-version=2024-04-01"
	slotsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot/slots?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"properties":{
			"serverFarmId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a",
			"siteConfig":{"linuxFxVersion":"Node|20"}
		}
	}`)))
	if err != nil {
		t.Fatalf("create web app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected web app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	createSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, slotURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"tags":{"stage":"staging"},
		"properties":{
			"httpsOnly":true,
			"siteConfig":{"linuxFxVersion":"Node|20"}
		}
	}`)))
	if err != nil {
		t.Fatalf("create web app slot returned error: %v", err)
	}
	if createSlotResp.StatusCode != http.StatusOK {
		t.Fatalf("expected web app slot create status 200, got %d; body=%s", createSlotResp.StatusCode, string(createSlotResp.RawBody))
	}
	createdSlot := decodeAppServiceResponse(t, createSlotResp)
	if createdSlot["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot/slots/staging" {
		t.Fatalf("unexpected slot id: %v", createdSlot["id"])
	}
	if createdSlot["name"] != "site-slot/staging" || createdSlot["type"] != "Microsoft.Web/sites/slots" || createdSlot["kind"] != "app,linux" {
		t.Fatalf("unexpected slot identity fields: %v", createdSlot)
	}
	slotProps := createdSlot["properties"].(map[string]any)
	if slotProps["state"] != "Running" || slotProps["provisioningState"] != "Succeeded" || slotProps["repositorySiteName"] != "site-slot" {
		t.Fatalf("unexpected slot state fields: %v", slotProps)
	}
	if slotProps["defaultHostName"] != "site-slot-staging.azurewebsites.net" {
		t.Fatalf("unexpected slot default host name: %v", slotProps["defaultHostName"])
	}
	enabledHostNames := slotProps["enabledHostNames"].([]any)
	if len(enabledHostNames) != 2 || enabledHostNames[0] != "site-slot-staging.azurewebsites.net" || enabledHostNames[1] != "site-slot-staging.scm.azurewebsites.net" {
		t.Fatalf("unexpected slot enabled host names: %v", enabledHostNames)
	}

	getSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotURL, nil))
	if err != nil {
		t.Fatalf("get web app slot returned error: %v", err)
	}
	if getSlotResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get web app slot status 200, got %d; body=%s", getSlotResp.StatusCode, string(getSlotResp.RawBody))
	}

	listSlotsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotsURL, nil))
	if err != nil {
		t.Fatalf("list web app slots returned error: %v", err)
	}
	if listSlotsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list web app slots status 200, got %d; body=%s", listSlotsResp.StatusCode, string(listSlotsResp.RawBody))
	}
	listed := decodeAppServiceResponse(t, listSlotsResp)
	values := listed["value"].([]any)
	if len(values) != 1 || values[0].(map[string]any)["name"] != "site-slot/staging" {
		t.Fatalf("expected staging slot in list, got %v", listed)
	}

	deleteSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, slotURL, nil))
	if err != nil {
		t.Fatalf("delete web app slot returned error: %v", err)
	}
	if deleteSlotResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete web app slot status 204, got %d; body=%s", deleteSlotResp.StatusCode, string(deleteSlotResp.RawBody))
	}
	getDeletedSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotURL, nil))
	if err != nil {
		t.Fatalf("get deleted web app slot returned error: %v", err)
	}
	if getDeletedSlotResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted slot status 404, got %d; body=%s", getDeletedSlotResp.StatusCode, string(getDeletedSlotResp.RawBody))
	}

	missingParentResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/slots/staging?api-version=2024-04-01", []byte(`{"location":"eastus"}`)))
	if err != nil {
		t.Fatalf("create slot under missing site returned error: %v", err)
	}
	if missingParentResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing parent slot create status 404, got %d; body=%s", missingParentResp.StatusCode, string(missingParentResp.RawBody))
	}
}

func TestAppServiceSlotPatchUpdatesIdentityTagsKindAndProperties(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-patch?api-version=2024-04-01"
	slotURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-patch/slots/staging?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"properties":{
			"serverFarmId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a",
			"siteConfig":{"linuxFxVersion":"Node|18"}
		}
	}`)))
	if err != nil {
		t.Fatalf("create parent web app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected parent web app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	createSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, slotURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"tags":{"stage":"staging"},
		"properties":{
			"enabled":true,
			"httpsOnly":false,
			"customSetting":"preserved",
			"siteConfig":{"linuxFxVersion":"Node|18","alwaysOn":false}
		}
	}`)))
	if err != nil {
		t.Fatalf("create web app slot returned error: %v", err)
	}
	if createSlotResp.StatusCode != http.StatusOK {
		t.Fatalf("expected web app slot create status 200, got %d; body=%s", createSlotResp.StatusCode, string(createSlotResp.RawBody))
	}

	patchSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, slotURL, []byte(`{
		"kind":"app,linux,container",
		"identity":{"type":"SystemAssigned"},
		"tags":{"stage":"prod","owner":"platform"},
		"properties":{
			"httpsOnly":true,
			"clientAffinityEnabled":false,
			"siteConfig":{"linuxFxVersion":"Node|20","alwaysOn":true}
		}
	}`)))
	if err != nil {
		t.Fatalf("patch web app slot returned error: %v", err)
	}
	if patchSlotResp.StatusCode != http.StatusOK {
		t.Fatalf("expected web app slot patch status 200, got %d; body=%s", patchSlotResp.StatusCode, string(patchSlotResp.RawBody))
	}

	updatedSlot := decodeAppServiceResponse(t, patchSlotResp)
	if updatedSlot["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-patch/slots/staging" {
		t.Fatalf("unexpected patched slot id: %v", updatedSlot["id"])
	}
	if updatedSlot["name"] != "site-slot-patch/staging" || updatedSlot["type"] != "Microsoft.Web/sites/slots" || updatedSlot["kind"] != "app,linux,container" {
		t.Fatalf("unexpected patched slot identity fields: %v", updatedSlot)
	}
	identity := updatedSlot["identity"].(map[string]any)
	if identity["type"] != "SystemAssigned" {
		t.Fatalf("expected patched slot identity, got %v", identity)
	}
	tags := updatedSlot["tags"].(map[string]any)
	if tags["stage"] != "prod" || tags["owner"] != "platform" {
		t.Fatalf("expected replaced slot tags, got %v", tags)
	}
	props := updatedSlot["properties"].(map[string]any)
	if props["httpsOnly"] != true || props["clientAffinityEnabled"] != false || props["enabled"] != true || props["customSetting"] != "preserved" {
		t.Fatalf("expected patched and preserved slot properties, got %v", props)
	}
	if props["provisioningState"] != "Succeeded" || props["state"] != "Running" || props["repositorySiteName"] != "site-slot-patch" {
		t.Fatalf("expected deterministic slot state fields to persist, got %v", props)
	}
	if props["defaultHostName"] != "site-slot-patch-staging.azurewebsites.net" {
		t.Fatalf("expected deterministic slot default host name, got %v", props["defaultHostName"])
	}
	siteConfig := props["siteConfig"].(map[string]any)
	if siteConfig["linuxFxVersion"] != "Node|20" || siteConfig["alwaysOn"] != true {
		t.Fatalf("expected patched slot siteConfig, got %v", siteConfig)
	}

	persistedResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotURL, nil))
	if err != nil {
		t.Fatalf("get patched web app slot returned error: %v", err)
	}
	persisted := decodeAppServiceResponse(t, persistedResp)
	persistedProps := persisted["properties"].(map[string]any)
	if persisted["kind"] != "app,linux,container" || persistedProps["httpsOnly"] != true || persistedProps["customSetting"] != "preserved" {
		t.Fatalf("expected patched web app slot to persist, got %v", persisted)
	}

	missingParentResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/slots/staging?api-version=2024-04-01", []byte(`{"properties":{"httpsOnly":true}}`)))
	if err != nil {
		t.Fatalf("patch slot under missing web app returned error: %v", err)
	}
	if missingParentResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing parent slot patch status 404, got %d; body=%s", missingParentResp.StatusCode, string(missingParentResp.RawBody))
	}

	missingSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-patch/slots/missing?api-version=2024-04-01", []byte(`{"properties":{"httpsOnly":true}}`)))
	if err != nil {
		t.Fatalf("patch missing web app slot returned error: %v", err)
	}
	if missingSlotResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing slot patch status 404, got %d; body=%s", missingSlotResp.StatusCode, string(missingSlotResp.RawBody))
	}
}

func TestAppServiceSlotConfigurationResourceLifecycle(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config-web?api-version=2024-04-01"
	slotURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config-web/slots/staging?api-version=2024-04-01"
	slotConfigURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config-web/slots/staging/config/web?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"properties":{
			"serverFarmId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a",
			"siteConfig":{"linuxFxVersion":"Node|18"}
		}
	}`)))
	if err != nil {
		t.Fatalf("create parent web app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected parent web app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	createSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, slotURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"properties":{
			"siteConfig":{
				"linuxFxVersion":"Node|18",
				"alwaysOn":false,
				"appCommandLine":"node server.js"
			}
		}
	}`)))
	if err != nil {
		t.Fatalf("create web app slot returned error: %v", err)
	}
	if createSlotResp.StatusCode != http.StatusOK {
		t.Fatalf("expected web app slot create status 200, got %d; body=%s", createSlotResp.StatusCode, string(createSlotResp.RawBody))
	}

	getConfigResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotConfigURL, nil))
	if err != nil {
		t.Fatalf("get web app slot config returned error: %v", err)
	}
	if getConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get web app slot config status 200, got %d; body=%s", getConfigResp.StatusCode, string(getConfigResp.RawBody))
	}
	config := decodeAppServiceResponse(t, getConfigResp)
	if config["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config-web/slots/staging/config/web" ||
		config["name"] != "web" ||
		config["type"] != "Microsoft.Web/sites/slots/config" {
		t.Fatalf("unexpected web app slot config identity fields: %v", config)
	}
	configProps := config["properties"].(map[string]any)
	if configProps["linuxFxVersion"] != "Node|18" || configProps["alwaysOn"] != false || configProps["appCommandLine"] != "node server.js" {
		t.Fatalf("unexpected initial web app slot config properties: %v", configProps)
	}

	patchConfigResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, slotConfigURL, []byte(`{
		"properties":{
			"linuxFxVersion":"Node|20",
			"alwaysOn":true,
			"http20Enabled":true
		}
	}`)))
	if err != nil {
		t.Fatalf("patch web app slot config returned error: %v", err)
	}
	if patchConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected patch web app slot config status 200, got %d; body=%s", patchConfigResp.StatusCode, string(patchConfigResp.RawBody))
	}
	patchedProps := decodeAppServiceResponse(t, patchConfigResp)["properties"].(map[string]any)
	if patchedProps["linuxFxVersion"] != "Node|20" || patchedProps["alwaysOn"] != true || patchedProps["http20Enabled"] != true || patchedProps["appCommandLine"] != "node server.js" {
		t.Fatalf("expected patched slot config to merge with existing properties, got %v", patchedProps)
	}

	slotAfterPatchResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotURL, nil))
	if err != nil {
		t.Fatalf("get slot after config patch returned error: %v", err)
	}
	slotAfterPatch := decodeAppServiceResponse(t, slotAfterPatchResp)
	slotConfigAfterPatch := slotAfterPatch["properties"].(map[string]any)["siteConfig"].(map[string]any)
	if slotConfigAfterPatch["linuxFxVersion"] != "Node|20" || slotConfigAfterPatch["appCommandLine"] != "node server.js" {
		t.Fatalf("expected patched slot config to persist on slot resource, got %v", slotConfigAfterPatch)
	}

	replaceConfigResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, slotConfigURL, []byte(`{
		"properties":{
			"linuxFxVersion":"Python|3.12",
			"alwaysOn":false
		}
	}`)))
	if err != nil {
		t.Fatalf("replace web app slot config returned error: %v", err)
	}
	if replaceConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected replace web app slot config status 200, got %d; body=%s", replaceConfigResp.StatusCode, string(replaceConfigResp.RawBody))
	}
	replacedProps := decodeAppServiceResponse(t, replaceConfigResp)["properties"].(map[string]any)
	if replacedProps["linuxFxVersion"] != "Python|3.12" || replacedProps["alwaysOn"] != false {
		t.Fatalf("expected replaced slot config properties, got %v", replacedProps)
	}
	if _, ok := replacedProps["appCommandLine"]; ok {
		t.Fatalf("expected PUT slot config to replace appCommandLine, got %v", replacedProps)
	}

	missingSlotResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config-web/slots/missing/config/web?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("get missing web app slot config returned error: %v", err)
	}
	if missingSlotResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing slot config status 404, got %d; body=%s", missingSlotResp.StatusCode, string(missingSlotResp.RawBody))
	}
}

func TestAppServiceSlotARMTemplateProvisioning(t *testing.T) {
	svc := New()

	parent, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Web/sites",
		"name":     "site-template",
		"location": "eastus",
		"kind":     "app,linux",
		"properties": map[string]any{
			"siteConfig": map[string]any{"linuxFxVersion": "Node|20"},
		},
	})
	if err != nil {
		t.Fatalf("provision parent web app returned error: %v", err)
	}
	parentMap := parent.(map[string]any)
	if parentMap["name"] != "site-template" {
		t.Fatalf("unexpected parent web app from template: %v", parentMap)
	}

	provisioned, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Web/sites/slots",
		"name":     "site-template/staging",
		"location": "eastus",
		"kind":     "app,linux",
		"tags":     map[string]any{"stage": "staging"},
		"properties": map[string]any{
			"siteConfig": map[string]any{"linuxFxVersion": "Node|20"},
		},
	})
	if err != nil {
		t.Fatalf("provision web app slot returned error: %v", err)
	}
	slot := provisioned.(map[string]any)
	if slot["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-template/slots/staging" {
		t.Fatalf("unexpected template slot id: %v", slot["id"])
	}
	if slot["name"] != "site-template/staging" || slot["type"] != "Microsoft.Web/sites/slots" {
		t.Fatalf("unexpected template slot identity fields: %v", slot)
	}
	props := slot["properties"].(map[string]any)
	if props["repositorySiteName"] != "site-template" || props["defaultHostName"] != "site-template-staging.azurewebsites.net" {
		t.Fatalf("unexpected template slot properties: %v", props)
	}
}

func TestFunctionAppARMFunctionChildLifecycle(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-arm?api-version=2024-04-01"
	functionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-arm/functions/HttpTrigger?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	createFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, functionURL, []byte(`{
		"kind":"function",
		"properties":{
			"language":"javascript",
			"isDisabled":false,
			"config":{"bindings":[{"type":"httpTrigger","direction":"in","name":"req"}]},
			"files":{"index.js":"module.exports = async function(context, req) {}"},
			"test_data":"{\"name\":\"Azure\"}"
		}
	}`)))
	if err != nil {
		t.Fatalf("create ARM function returned error: %v", err)
	}
	if createFunctionResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create ARM function status 201, got %d; body=%s", createFunctionResp.StatusCode, string(createFunctionResp.RawBody))
	}
	created := decodeAppServiceResponse(t, createFunctionResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-arm/functions/HttpTrigger" {
		t.Fatalf("unexpected ARM function id: %v", created["id"])
	}
	if created["name"] != "func-arm/HttpTrigger" || created["type"] != "Microsoft.Web/sites/functions" || created["kind"] != "function" {
		t.Fatalf("unexpected ARM function identity fields: %v", created)
	}
	createdProps := created["properties"].(map[string]any)
	if createdProps["function_app_id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-arm" {
		t.Fatalf("expected function_app_id to reference parent site, got %v", createdProps)
	}
	if !strings.Contains(createdProps["invoke_url_template"].(string), "/api/HttpTrigger") || !strings.Contains(createdProps["href"].(string), "/functions/HttpTrigger") {
		t.Fatalf("expected function href and invoke URL defaults, got %v", createdProps)
	}
	if createdProps["language"] != "javascript" || createdProps["isDisabled"] != false {
		t.Fatalf("expected function properties to be preserved, got %v", createdProps)
	}

	getFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, functionURL, nil))
	if err != nil {
		t.Fatalf("get ARM function returned error: %v", err)
	}
	if getFunctionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get ARM function status 200, got %d; body=%s", getFunctionResp.StatusCode, string(getFunctionResp.RawBody))
	}
	got := decodeAppServiceResponse(t, getFunctionResp)
	if got["name"] != "func-arm/HttpTrigger" || got["properties"].(map[string]any)["test_data"] != `{"name":"Azure"}` {
		t.Fatalf("unexpected get ARM function response: %v", got)
	}

	listFunctionsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-arm/functions?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list ARM functions returned error: %v", err)
	}
	listed := decodeAppServiceResponse(t, listFunctionsResp)
	if values := listed["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "func-arm/HttpTrigger" {
		t.Fatalf("expected created ARM function in list, got %v", listed)
	}

	deleteFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, functionURL, nil))
	if err != nil {
		t.Fatalf("delete ARM function returned error: %v", err)
	}
	if deleteFunctionResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete ARM function status 204, got %d; body=%s", deleteFunctionResp.StatusCode, string(deleteFunctionResp.RawBody))
	}

	getDeletedFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, functionURL, nil))
	if err != nil {
		t.Fatalf("get deleted ARM function returned error: %v", err)
	}
	if getDeletedFunctionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted ARM function status 404, got %d; body=%s", getDeletedFunctionResp.StatusCode, string(getDeletedFunctionResp.RawBody))
	}
}

func TestFunctionAppARMFunctionKeysLifecycle(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-keys?api-version=2024-04-01"
	functionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-keys/functions/HttpTrigger?api-version=2024-04-01"
	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-keys/functions/HttpTrigger/listkeys?api-version=2024-04-01"
	customKeyURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-keys/functions/HttpTrigger/keys/client?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	createFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, functionURL, []byte(`{
		"kind":"function",
		"properties":{"language":"javascript"}
	}`)))
	if err != nil {
		t.Fatalf("create ARM function returned error: %v", err)
	}
	if createFunctionResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create ARM function status 201, got %d; body=%s", createFunctionResp.StatusCode, string(createFunctionResp.RawBody))
	}

	listKeysResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list function keys returned error: %v", err)
	}
	if listKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list function keys status 200, got %d; body=%s", listKeysResp.StatusCode, string(listKeysResp.RawBody))
	}
	listedKeys := decodeAppServiceResponse(t, listKeysResp)
	if listedKeys["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-keys/functions/HttpTrigger/listkeys" {
		t.Fatalf("unexpected function keys id: %v", listedKeys["id"])
	}
	if listedKeys["name"] != "func-keys/HttpTrigger/listkeys" || listedKeys["type"] != "Microsoft.Web/sites/functions/listkeys" {
		t.Fatalf("unexpected function keys identity fields: %v", listedKeys)
	}
	keyProps := listedKeys["properties"].(map[string]any)
	if keyProps["default"] != "cloudmock-func-keys-httptrigger-default-key" {
		t.Fatalf("expected deterministic default function key, got %v", keyProps)
	}

	createKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, customKeyURL, []byte(`{"name":"client","value":"secret-1"}`)))
	if err != nil {
		t.Fatalf("create function secret returned error: %v", err)
	}
	if createKeyResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create function secret status 201, got %d; body=%s", createKeyResp.StatusCode, string(createKeyResp.RawBody))
	}
	createdKey := decodeAppServiceResponse(t, createKeyResp)
	if createdKey["name"] != "client" || createdKey["value"] != "secret-1" {
		t.Fatalf("unexpected created function secret response: %v", createdKey)
	}

	updateKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, customKeyURL, []byte(`{"name":"client","value":"secret-2"}`)))
	if err != nil {
		t.Fatalf("update function secret returned error: %v", err)
	}
	if updateKeyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update function secret status 200, got %d; body=%s", updateKeyResp.StatusCode, string(updateKeyResp.RawBody))
	}

	listUpdatedKeysResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listKeysURL, nil))
	if err != nil {
		t.Fatalf("list updated function keys returned error: %v", err)
	}
	updatedKeys := decodeAppServiceResponse(t, listUpdatedKeysResp)
	updatedProps := updatedKeys["properties"].(map[string]any)
	if updatedProps["default"] != "cloudmock-func-keys-httptrigger-default-key" || updatedProps["client"] != "secret-2" {
		t.Fatalf("expected default and updated client function keys, got %v", updatedProps)
	}

	deleteKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, customKeyURL, nil))
	if err != nil {
		t.Fatalf("delete function secret returned error: %v", err)
	}
	if deleteKeyResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete function secret status 204, got %d; body=%s", deleteKeyResp.StatusCode, string(deleteKeyResp.RawBody))
	}

	deleteMissingKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, customKeyURL, nil))
	if err != nil {
		t.Fatalf("delete missing function secret returned error: %v", err)
	}
	if deleteMissingKeyResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing function secret status 404, got %d; body=%s", deleteMissingKeyResp.StatusCode, string(deleteMissingKeyResp.RawBody))
	}
}

func TestFunctionAppARMHostKeysLifecycle(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-host-keys?api-version=2024-04-01"
	listHostKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-host-keys/host/default/listkeys?api-version=2024-04-01"
	functionKeyURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-host-keys/host/default/functionkeys/client?api-version=2024-04-01"
	systemKeyURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-host-keys/host/default/systemkeys/durabletask?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	listHostKeysResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listHostKeysURL, nil))
	if err != nil {
		t.Fatalf("list host keys returned error: %v", err)
	}
	if listHostKeysResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list host keys status 200, got %d; body=%s", listHostKeysResp.StatusCode, string(listHostKeysResp.RawBody))
	}
	hostKeys := decodeAppServiceResponse(t, listHostKeysResp)
	if hostKeys["masterKey"] != "cloudmock-func-host-keys-master-key" {
		t.Fatalf("expected deterministic master key, got %v", hostKeys)
	}
	functionKeys := hostKeys["functionKeys"].(map[string]any)
	if functionKeys["default"] != "cloudmock-func-host-keys-host-default-key" {
		t.Fatalf("expected deterministic default host function key, got %v", functionKeys)
	}
	systemKeys := hostKeys["systemKeys"].(map[string]any)
	if len(systemKeys) != 0 {
		t.Fatalf("expected no default system keys, got %v", systemKeys)
	}

	createFunctionKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, functionKeyURL, []byte(`{"name":"client","value":"host-secret-1"}`)))
	if err != nil {
		t.Fatalf("create host function secret returned error: %v", err)
	}
	if createFunctionKeyResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create host function secret status 201, got %d; body=%s", createFunctionKeyResp.StatusCode, string(createFunctionKeyResp.RawBody))
	}
	createdFunctionKey := decodeAppServiceResponse(t, createFunctionKeyResp)
	if createdFunctionKey["name"] != "client" || createdFunctionKey["value"] != "host-secret-1" {
		t.Fatalf("unexpected host function secret response: %v", createdFunctionKey)
	}

	updateFunctionKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, functionKeyURL, []byte(`{"name":"client","value":"host-secret-2"}`)))
	if err != nil {
		t.Fatalf("update host function secret returned error: %v", err)
	}
	if updateFunctionKeyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update host function secret status 200, got %d; body=%s", updateFunctionKeyResp.StatusCode, string(updateFunctionKeyResp.RawBody))
	}

	createSystemKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, systemKeyURL, []byte(`{"name":"durabletask","value":"system-secret"}`)))
	if err != nil {
		t.Fatalf("create host system secret returned error: %v", err)
	}
	if createSystemKeyResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create host system secret status 201, got %d; body=%s", createSystemKeyResp.StatusCode, string(createSystemKeyResp.RawBody))
	}

	listUpdatedHostKeysResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listHostKeysURL, nil))
	if err != nil {
		t.Fatalf("list updated host keys returned error: %v", err)
	}
	updatedHostKeys := decodeAppServiceResponse(t, listUpdatedHostKeysResp)
	updatedFunctionKeys := updatedHostKeys["functionKeys"].(map[string]any)
	if updatedFunctionKeys["default"] != "cloudmock-func-host-keys-host-default-key" || updatedFunctionKeys["client"] != "host-secret-2" {
		t.Fatalf("expected default and custom host function keys, got %v", updatedFunctionKeys)
	}
	updatedSystemKeys := updatedHostKeys["systemKeys"].(map[string]any)
	if updatedSystemKeys["durabletask"] != "system-secret" {
		t.Fatalf("expected custom host system key, got %v", updatedSystemKeys)
	}

	deleteFunctionKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, functionKeyURL, nil))
	if err != nil {
		t.Fatalf("delete host function secret returned error: %v", err)
	}
	if deleteFunctionKeyResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete host function secret status 204, got %d; body=%s", deleteFunctionKeyResp.StatusCode, string(deleteFunctionKeyResp.RawBody))
	}

	deleteMissingFunctionKeyResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, functionKeyURL, nil))
	if err != nil {
		t.Fatalf("delete missing host function secret returned error: %v", err)
	}
	if deleteMissingFunctionKeyResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing host function secret status 404, got %d; body=%s", deleteMissingFunctionKeyResp.StatusCode, string(deleteMissingFunctionKeyResp.RawBody))
	}
}

func TestFunctionAppPublishingCredentialsAndProfile(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-publish?api-version=2024-04-01"
	credentialsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-publish/config/publishingcredentials/list?api-version=2024-04-01"
	publishXMLURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-publish/publishxml?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	credentialsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, credentialsURL, nil))
	if err != nil {
		t.Fatalf("list publishing credentials returned error: %v", err)
	}
	if credentialsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list publishing credentials status 200, got %d; body=%s", credentialsResp.StatusCode, string(credentialsResp.RawBody))
	}
	credentials := decodeAppServiceResponse(t, credentialsResp)
	if credentials["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-publish/config/publishingcredentials" {
		t.Fatalf("unexpected publishing credentials id: %v", credentials["id"])
	}
	if credentials["name"] != "publishingcredentials" || credentials["type"] != "Microsoft.Web/sites/config" {
		t.Fatalf("unexpected publishing credentials identity fields: %v", credentials)
	}
	credentialProps := credentials["properties"].(map[string]any)
	if credentialProps["publishingUserName"] != "$func-publish" {
		t.Fatalf("unexpected publishing user name: %v", credentialProps)
	}
	if credentialProps["publishingPassword"] != "cloudmock-func-publish-publishing-password" {
		t.Fatalf("unexpected publishing password: %v", credentialProps)
	}
	if credentialProps["scmUri"] != "https://$func-publish:cloudmock-func-publish-publishing-password@func-publish.scm.azurewebsites.net" {
		t.Fatalf("unexpected publishing scmUri: %v", credentialProps)
	}

	profileResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, publishXMLURL, []byte(`{"format":"WebDeploy"}`)))
	if err != nil {
		t.Fatalf("list publishing profile returned error: %v", err)
	}
	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list publishing profile status 200, got %d; body=%s", profileResp.StatusCode, string(profileResp.RawBody))
	}
	if profileResp.RawContentType != "application/xml" {
		t.Fatalf("expected application/xml publishing profile content type, got %q", profileResp.RawContentType)
	}
	profileXML := string(profileResp.RawBody)
	for _, want := range []string{
		`<publishData>`,
		`publishMethod="MSDeploy"`,
		`publishUrl="func-publish.scm.azurewebsites.net:443"`,
		`userName="$func-publish"`,
		`userPWD="cloudmock-func-publish-publishing-password"`,
		`destinationAppUrl="https://func-publish.azurewebsites.net"`,
	} {
		if !strings.Contains(profileXML, want) {
			t.Fatalf("expected publishing profile XML to contain %q, got %s", want, profileXML)
		}
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/config/publishingcredentials/list?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list missing publishing credentials returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing publishing credentials status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestAppServiceSubscriptionScopedLists(t *testing.T) {
	svc := New()

	createPlan := func(rawURL, location string) {
		t.Helper()
		resp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, rawURL, []byte(`{
			"location":"`+location+`",
			"sku":{"name":"B1","tier":"Basic"},
			"properties":{"reserved":true}
		}`)))
		if err != nil {
			t.Fatalf("create plan %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create plan 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}
	createSite := func(rawURL, planID string) {
		t.Helper()
		resp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, rawURL, []byte(`{
			"location":"eastus",
			"kind":"app,linux",
			"properties":{
				"serverFarmId":"`+planID+`",
				"httpsOnly":true
			}
		}`)))
		if err != nil {
			t.Fatalf("create site %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create site 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}

	planAURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a?api-version=2024-04-01"
	planBURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Web/serverfarms/plan-b?api-version=2024-04-01"
	planOtherURL := "https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.Web/serverfarms/plan-z?api-version=2024-04-01"
	createPlan(planBURL, "westus2")
	createPlan(planOtherURL, "centralus")
	createPlan(planAURL, "eastus")

	createSite("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a?api-version=2024-04-01", "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a")
	createSite("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Web/sites/site-b?api-version=2024-04-01", "/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Web/serverfarms/plan-b")
	createSite("https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.Web/sites/site-z?api-version=2024-04-01", "/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.Web/serverfarms/plan-z")

	plansResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Web/serverfarms?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list subscription plans returned error: %v", err)
	}
	if plansResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription plan list status 200, got %d; body=%s", plansResp.StatusCode, string(plansResp.RawBody))
	}
	plans := decodeAppServiceResponse(t, plansResp)
	planValues := plans["value"].([]any)
	if len(planValues) != 2 || planValues[0].(map[string]any)["name"] != "plan-a" || planValues[1].(map[string]any)["name"] != "plan-b" {
		t.Fatalf("expected sub-1 plans sorted by name, got %v", plans)
	}

	sitesResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Web/sites?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list subscription sites returned error: %v", err)
	}
	if sitesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription site list status 200, got %d; body=%s", sitesResp.StatusCode, string(sitesResp.RawBody))
	}
	sites := decodeAppServiceResponse(t, sitesResp)
	siteValues := sites["value"].([]any)
	if len(siteValues) != 2 || siteValues[0].(map[string]any)["name"] != "site-a" || siteValues[1].(map[string]any)["name"] != "site-b" {
		t.Fatalf("expected sub-1 sites sorted by name, got %v", sites)
	}
}

func TestAzureFunctionsLocalAdminAndInvokeLifecycle(t *testing.T) {
	svc := New()

	baseURL := "http://localhost:4577/devstoreaccount1-functions"
	appURL := baseURL + "/admin/apps/app-a"
	functionURL := appURL + "/functions/hello"

	createAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, appURL, []byte(`{
		"runtime":"node",
		"environment":{"APP_SETTING":"from-app"}
	}`)))
	if err != nil {
		t.Fatalf("create local function app returned error: %v", err)
	}
	if createAppResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create local function app status 201, got %d; body=%s", createAppResp.StatusCode, string(createAppResp.RawBody))
	}
	app := decodeAppServiceResponse(t, createAppResp)
	if app["name"] != "app-a" || app["runtime"] != "node" || app["status"] != "Running" {
		t.Fatalf("unexpected local function app response: %v", app)
	}

	getAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, appURL, nil))
	if err != nil {
		t.Fatalf("get local function app returned error: %v", err)
	}
	if getAppResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get local function app status 200, got %d; body=%s", getAppResp.StatusCode, string(getAppResp.RawBody))
	}

	listAppsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, baseURL+"/admin/apps", nil))
	if err != nil {
		t.Fatalf("list local function apps returned error: %v", err)
	}
	listApps := decodeAppServiceResponse(t, listAppsResp)
	if values := listApps["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "app-a" {
		t.Fatalf("expected app-a in local function app list, got %v", listApps)
	}

	deployResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, functionURL, []byte(`{
		"handler":"index.handler",
		"timeoutSeconds":60,
		"zipBase64":"ZmFrZS16aXA=",
		"environment":{"FUNC_SETTING":"from-function"}
	}`)))
	if err != nil {
		t.Fatalf("deploy local function returned error: %v", err)
	}
	if deployResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected deploy local function status 201, got %d; body=%s", deployResp.StatusCode, string(deployResp.RawBody))
	}
	deployed := decodeAppServiceResponse(t, deployResp)
	if deployed["name"] != "hello" || deployed["appName"] != "app-a" || deployed["runtime"] != "node" || deployed["status"] != "Ready" {
		t.Fatalf("unexpected deployed local function response: %v", deployed)
	}
	if !strings.Contains(deployed["invokeUrl"].(string), "/devstoreaccount1-functions/api/app-a/hello") {
		t.Fatalf("unexpected local function invoke URL: %v", deployed["invokeUrl"])
	}

	listFunctionsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, appURL+"/functions", nil))
	if err != nil {
		t.Fatalf("list local functions returned error: %v", err)
	}
	listFunctions := decodeAppServiceResponse(t, listFunctionsResp)
	if values := listFunctions["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "hello" {
		t.Fatalf("expected hello in local function list, got %v", listFunctions)
	}

	invokeResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, baseURL+"/api/app-a/hello?msg=world", []byte(`{"name":"Azure"}`)))
	if err != nil {
		t.Fatalf("invoke local function returned error: %v", err)
	}
	if invokeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected invoke local function status 200, got %d; body=%s", invokeResp.StatusCode, string(invokeResp.RawBody))
	}
	invoked := decodeAppServiceResponse(t, invokeResp)
	if invoked["appName"] != "app-a" || invoked["functionName"] != "hello" || invoked["method"] != "POST" {
		t.Fatalf("unexpected local function invocation response: %v", invoked)
	}
	if invoked["queryParams"].(map[string]any)["msg"] != "world" || invoked["body"] != `{"name":"Azure"}` {
		t.Fatalf("expected invocation query/body echo, got %v", invoked)
	}

	deleteFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, functionURL, nil))
	if err != nil {
		t.Fatalf("delete local function returned error: %v", err)
	}
	if deleteFunctionResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete local function status 204, got %d; body=%s", deleteFunctionResp.StatusCode, string(deleteFunctionResp.RawBody))
	}
	getDeletedFunctionResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, functionURL, nil))
	if err != nil {
		t.Fatalf("get deleted local function returned error: %v", err)
	}
	if getDeletedFunctionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted local function status 404, got %d; body=%s", getDeletedFunctionResp.StatusCode, string(getDeletedFunctionResp.RawBody))
	}

	deleteAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodDelete, appURL, nil))
	if err != nil {
		t.Fatalf("delete local function app returned error: %v", err)
	}
	if deleteAppResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete local function app status 204, got %d; body=%s", deleteAppResp.StatusCode, string(deleteAppResp.RawBody))
	}
	getDeletedAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, appURL, nil))
	if err != nil {
		t.Fatalf("get deleted local function app returned error: %v", err)
	}
	if getDeletedAppResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted local function app status 404, got %d; body=%s", getDeletedAppResp.StatusCode, string(getDeletedAppResp.RawBody))
	}
}

func TestFunctionAppSiteConfigLinuxFxVersionBridgesLocalFunctionApp(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-linux?api-version=2024-04-01"
	configURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-linux/config/web?api-version=2024-04-01"
	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Python|3.12"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app with linuxFxVersion returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	configResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, configURL, nil))
	if err != nil {
		t.Fatalf("get function app config returned error: %v", err)
	}
	if configResp.StatusCode != http.StatusOK {
		t.Fatalf("expected function app config status 200, got %d; body=%s", configResp.StatusCode, string(configResp.RawBody))
	}
	config := decodeAppServiceResponse(t, configResp)
	if config["name"] != "web" || config["type"] != "Microsoft.Web/sites/config" {
		t.Fatalf("unexpected config identity fields: %v", config)
	}
	configProps := config["properties"].(map[string]any)
	if configProps["linuxFxVersion"] != "Python|3.12" {
		t.Fatalf("expected Python linuxFxVersion, got %v", configProps)
	}

	localAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-functions/admin/apps/func-linux", nil))
	if err != nil {
		t.Fatalf("get bridged local function app returned error: %v", err)
	}
	if localAppResp.StatusCode != http.StatusOK {
		t.Fatalf("expected bridged local function app status 200, got %d; body=%s", localAppResp.StatusCode, string(localAppResp.RawBody))
	}
	localApp := decodeAppServiceResponse(t, localAppResp)
	if localApp["runtime"] != "python" || localApp["linuxFxVersion"] != "Python|3.12" {
		t.Fatalf("unexpected bridged local function app: %v", localApp)
	}

	updateConfigResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, configURL, []byte(`{"properties":{"linuxFxVersion":"Node|20"}}`)))
	if err != nil {
		t.Fatalf("update function app config returned error: %v", err)
	}
	if updateConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected config update status 200, got %d; body=%s", updateConfigResp.StatusCode, string(updateConfigResp.RawBody))
	}

	updatedLocalAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-functions/admin/apps/func-linux", nil))
	if err != nil {
		t.Fatalf("get updated bridged local function app returned error: %v", err)
	}
	updatedLocalApp := decodeAppServiceResponse(t, updatedLocalAppResp)
	if updatedLocalApp["runtime"] != "node" || updatedLocalApp["linuxFxVersion"] != "Node|20" {
		t.Fatalf("unexpected updated bridged local function app: %v", updatedLocalApp)
	}
}

func TestFunctionAppPatchSiteConfigMergesPropertiesAndBridgesRuntime(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-patch?api-version=2024-04-01"
	configURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-patch/config/web?api-version=2024-04-01"
	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{
			"siteConfig":{
				"linuxFxVersion":"Python|3.12",
				"alwaysOn":false,
				"http20Enabled":false,
				"appCommandLine":"python app.py"
			}
		}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	patchConfigResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, configURL, []byte(`{
		"properties":{
			"linuxFxVersion":"Node|20",
			"alwaysOn":true,
			"http20Enabled":true,
			"webSocketsEnabled":true
		}
	}`)))
	if err != nil {
		t.Fatalf("patch function app config returned error: %v", err)
	}
	if patchConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected config patch status 200, got %d; body=%s", patchConfigResp.StatusCode, string(patchConfigResp.RawBody))
	}
	patchedConfig := decodeAppServiceResponse(t, patchConfigResp)
	if patchedConfig["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-patch/config/web" || patchedConfig["name"] != "web" || patchedConfig["type"] != "Microsoft.Web/sites/config" {
		t.Fatalf("unexpected patched config identity fields: %v", patchedConfig)
	}
	configProps := patchedConfig["properties"].(map[string]any)
	if configProps["linuxFxVersion"] != "Node|20" || configProps["alwaysOn"] != true || configProps["http20Enabled"] != true || configProps["webSocketsEnabled"] != true {
		t.Fatalf("expected patched config properties, got %v", configProps)
	}
	if configProps["appCommandLine"] != "python app.py" {
		t.Fatalf("expected config patch to preserve existing appCommandLine, got %v", configProps["appCommandLine"])
	}

	getConfigResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, configURL, nil))
	if err != nil {
		t.Fatalf("get patched function app config returned error: %v", err)
	}
	persistedProps := decodeAppServiceResponse(t, getConfigResp)["properties"].(map[string]any)
	if persistedProps["linuxFxVersion"] != "Node|20" || persistedProps["appCommandLine"] != "python app.py" {
		t.Fatalf("expected patched config to persist with merged properties, got %v", persistedProps)
	}

	localAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-functions/admin/apps/func-config-patch", nil))
	if err != nil {
		t.Fatalf("get patched local function app returned error: %v", err)
	}
	localApp := decodeAppServiceResponse(t, localAppResp)
	if localApp["runtime"] != "node" || localApp["linuxFxVersion"] != "Node|20" {
		t.Fatalf("expected patched config to refresh local function app runtime, got %v", localApp)
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/config/web?api-version=2024-04-01", []byte(`{"properties":{"alwaysOn":true}}`)))
	if err != nil {
		t.Fatalf("patch missing function app config returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing config patch status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestAppServiceSlotConfigurationNamesLifecycle(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config?api-version=2024-04-01"
	slotConfigNamesURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config/config/slotConfigNames?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"app,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create web app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected web app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	initialResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotConfigNamesURL, nil))
	if err != nil {
		t.Fatalf("list empty slot configuration names returned error: %v", err)
	}
	if initialResp.StatusCode != http.StatusOK {
		t.Fatalf("expected empty slot configuration names status 200, got %d; body=%s", initialResp.StatusCode, string(initialResp.RawBody))
	}
	initial := decodeAppServiceResponse(t, initialResp)
	if initial["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-slot-config/config/slotConfigNames" ||
		initial["name"] != "slotConfigNames" ||
		initial["type"] != "Microsoft.Web/sites/config" ||
		initial["kind"] != "app,linux" {
		t.Fatalf("unexpected empty slot configuration names identity fields: %v", initial)
	}
	initialProps := initial["properties"].(map[string]any)
	for _, key := range []string{"appSettingNames", "connectionStringNames", "azureStorageConfigNames"} {
		values, ok := initialProps[key].([]any)
		if !ok || len(values) != 0 {
			t.Fatalf("expected empty %s list, got %v", key, initialProps[key])
		}
	}

	updateResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, slotConfigNamesURL, []byte(`{
		"kind":"app,linux",
		"properties":{
			"appSettingNames":["APP_SLOT","FeatureFlag"],
			"connectionStringNames":["MainDb"],
			"azureStorageConfigNames":["assets"]
		}
	}`)))
	if err != nil {
		t.Fatalf("update slot configuration names returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected slot configuration names update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeAppServiceResponse(t, updateResp)
	updatedProps := updated["properties"].(map[string]any)
	if got := updatedProps["appSettingNames"].([]any); len(got) != 2 || got[0] != "APP_SLOT" || got[1] != "FeatureFlag" {
		t.Fatalf("unexpected sticky app setting names: %v", got)
	}
	if got := updatedProps["connectionStringNames"].([]any); len(got) != 1 || got[0] != "MainDb" {
		t.Fatalf("unexpected sticky connection string names: %v", got)
	}
	if got := updatedProps["azureStorageConfigNames"].([]any); len(got) != 1 || got[0] != "assets" {
		t.Fatalf("unexpected sticky Azure storage config names: %v", got)
	}

	persistedResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, slotConfigNamesURL, nil))
	if err != nil {
		t.Fatalf("list updated slot configuration names returned error: %v", err)
	}
	persistedProps := decodeAppServiceResponse(t, persistedResp)["properties"].(map[string]any)
	if got := persistedProps["appSettingNames"].([]any); len(got) != 2 || got[0] != "APP_SLOT" || got[1] != "FeatureFlag" {
		t.Fatalf("expected slot configuration names to persist, got %v", persistedProps)
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/config/slotConfigNames?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("list missing slot configuration names returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing slot configuration names status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestFunctionAppPatchSiteConfigSyncsNamedConfigProjections(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-named?api-version=2024-04-01"
	configURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-named/config/web?api-version=2024-04-01"
	appSettingsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-named/config/appsettings/list?api-version=2024-04-01"
	connectionStringsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-config-named/config/connectionstrings/list?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	patchResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, configURL, []byte(`{
		"properties":{
			"appSettings":[
				{"name":"FROM_PATCH","value":"yes"},
				{"name":"SLOT_SETTING","value":"sticky","slotSetting":true}
			],
			"connectionStrings":[
				{"name":"DefaultConnection","connectionString":"Server=tcp:db;","type":"SQLAzure","slotSetting":true}
			]
		}
	}`)))
	if err != nil {
		t.Fatalf("patch function app config returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected config patch status 200, got %d; body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}

	appSettingsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, appSettingsURL, nil))
	if err != nil {
		t.Fatalf("list patched app settings returned error: %v", err)
	}
	appSettings := decodeAppServiceResponse(t, appSettingsResp)["properties"].(map[string]any)
	if appSettings["FROM_PATCH"] != "yes" || appSettings["SLOT_SETTING"] != "sticky" {
		t.Fatalf("expected config patch app settings to sync named config projection, got %v", appSettings)
	}

	connectionStringsResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, connectionStringsURL, nil))
	if err != nil {
		t.Fatalf("list patched connection strings returned error: %v", err)
	}
	connectionStrings := decodeAppServiceResponse(t, connectionStringsResp)["properties"].(map[string]any)
	defaultConnection := connectionStrings["DefaultConnection"].(map[string]any)
	if defaultConnection["value"] != "Server=tcp:db;" || defaultConnection["type"] != "SQLAzure" || defaultConnection["slotSetting"] != true {
		t.Fatalf("expected config patch connection strings to sync named config projection, got %v", connectionStrings)
	}
}

func TestFunctionAppSyncFunctionTriggers(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-sync?api-version=2024-04-01"
	syncURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-sync/syncfunctiontriggers?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"Node|20"}}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	syncResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, syncURL, nil))
	if err != nil {
		t.Fatalf("sync function triggers returned error: %v", err)
	}
	if syncResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected sync function triggers status 204, got %d; body=%s", syncResp.StatusCode, string(syncResp.RawBody))
	}
	if len(syncResp.RawBody) != 0 {
		t.Fatalf("expected sync function triggers to return no body, got %s", string(syncResp.RawBody))
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/syncfunctiontriggers?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("sync missing function triggers returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing sync function triggers status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestFunctionAppPatchSiteUpdatesTagsIdentityAndProperties(t *testing.T) {
	svc := New()
	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-patch?api-version=2024-04-01"

	createSiteResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"tags":{"env":"dev"},
		"properties":{
			"serverFarmId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a",
			"httpsOnly":false,
			"siteConfig":{"linuxFxVersion":"Python|3.12"}
		}
	}`)))
	if err != nil {
		t.Fatalf("create function app returned error: %v", err)
	}
	if createSiteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected function app create status 201, got %d; body=%s", createSiteResp.StatusCode, string(createSiteResp.RawBody))
	}

	patchResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, siteURL, []byte(`{
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"prod","owner":"platform"},
		"properties":{
			"httpsOnly":true,
			"clientAffinityEnabled":false,
			"siteConfig":{"linuxFxVersion":"Node|20"}
		}
	}`)))
	if err != nil {
		t.Fatalf("patch function app returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected function app patch status 200, got %d; body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}

	updated := decodeAppServiceResponse(t, patchResp)
	if updated["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-patch" || updated["name"] != "func-patch" || updated["type"] != "Microsoft.Web/sites" {
		t.Fatalf("unexpected patched function app identity fields: %v", updated)
	}
	if updated["kind"] != "functionapp,linux" || updated["location"] != "eastus" {
		t.Fatalf("expected patch to preserve kind/location, got %v", updated)
	}
	identity := updated["identity"].(map[string]any)
	if identity["type"] != "SystemAssigned" {
		t.Fatalf("expected patched identity, got %v", identity)
	}
	tags := updated["tags"].(map[string]any)
	if tags["env"] != "prod" || tags["owner"] != "platform" {
		t.Fatalf("expected replaced tags, got %v", tags)
	}
	props := updated["properties"].(map[string]any)
	if props["httpsOnly"] != true || props["clientAffinityEnabled"] != false {
		t.Fatalf("expected patched properties, got %v", props)
	}
	if props["serverFarmId"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/serverfarms/plan-a" {
		t.Fatalf("expected patch to preserve serverFarmId, got %v", props["serverFarmId"])
	}
	if props["state"] != "Running" || props["provisioningState"] != "Succeeded" || props["defaultHostName"] != "func-patch.azurewebsites.net" {
		t.Fatalf("expected patch to preserve deterministic state fields, got %v", props)
	}
	siteConfig := props["siteConfig"].(map[string]any)
	if siteConfig["linuxFxVersion"] != "Node|20" {
		t.Fatalf("expected patched linuxFxVersion, got %v", siteConfig)
	}

	persisted := getAppServiceSite(t, svc, siteURL)
	persistedProps := persisted["properties"].(map[string]any)
	if persistedProps["httpsOnly"] != true || persistedProps["clientAffinityEnabled"] != false {
		t.Fatalf("expected patched properties to persist, got %v", persistedProps)
	}

	localAppResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-functions/admin/apps/func-patch", nil))
	if err != nil {
		t.Fatalf("get patched local function app returned error: %v", err)
	}
	localApp := decodeAppServiceResponse(t, localAppResp)
	if localApp["runtime"] != "node" || localApp["linuxFxVersion"] != "Node|20" {
		t.Fatalf("expected patched siteConfig to refresh local function app, got %v", localApp)
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPatch, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing?api-version=2024-04-01", []byte(`{"properties":{"httpsOnly":true}}`)))
	if err != nil {
		t.Fatalf("patch missing function app returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing patch status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestFunctionAppSiteRejectsMalformedLinuxFxVersion(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/func-bad?api-version=2024-04-01"
	resp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"kind":"functionapp,linux",
		"properties":{"siteConfig":{"linuxFxVersion":"python-3.12"}}
	}`)))
	if err != nil {
		t.Fatalf("create malformed linuxFxVersion function app returned error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected malformed linuxFxVersion status 400, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
}

func TestAppServiceSiteAppSettingsConfigLifecycle(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-settings?api-version=2024-04-01"
	appSettingsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-settings/config/appsettings?api-version=2024-04-01"
	listAppSettingsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-settings/config/appsettings/list?api-version=2024-04-01"

	createResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"properties":{
			"siteConfig":{
				"appSettings":[
					{"name":"FROM_CREATE","value":"yes"},
					{"name":"SECRET_SETTING","value":"initial","slotSetting":true}
				]
			}
		}
	}`)))
	if err != nil {
		t.Fatalf("create web app with app settings returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected web app create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	seededResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listAppSettingsURL, nil))
	if err != nil {
		t.Fatalf("list seeded app settings returned error: %v", err)
	}
	if seededResp.StatusCode != http.StatusOK {
		t.Fatalf("expected seeded app settings status 200, got %d; body=%s", seededResp.StatusCode, string(seededResp.RawBody))
	}
	seededProps := decodeAppServiceResponse(t, seededResp)["properties"].(map[string]any)
	if seededProps["FROM_CREATE"] != "yes" || seededProps["SECRET_SETTING"] != "initial" {
		t.Fatalf("expected seeded app settings from siteConfig, got %v", seededProps)
	}

	updateResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, appSettingsURL, []byte(`{
		"properties":{"APPSETTING_A":"one","SECRET_SETTING":"updated"}
	}`)))
	if err != nil {
		t.Fatalf("update app settings returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected app settings update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeAppServiceResponse(t, updateResp)
	if updated["name"] != "appsettings" || updated["type"] != "Microsoft.Web/sites/config" {
		t.Fatalf("unexpected app settings identity fields: %v", updated)
	}
	if updated["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-settings/config/appsettings" {
		t.Fatalf("unexpected app settings id: %v", updated["id"])
	}
	updatedProps := updated["properties"].(map[string]any)
	if updatedProps["APPSETTING_A"] != "one" || updatedProps["SECRET_SETTING"] != "updated" {
		t.Fatalf("expected updated app settings, got %v", updatedProps)
	}
	if _, ok := updatedProps["FROM_CREATE"]; ok {
		t.Fatalf("expected app settings update to replace the prior set, got %v", updatedProps)
	}

	listResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listAppSettingsURL, nil))
	if err != nil {
		t.Fatalf("list updated app settings returned error: %v", err)
	}
	listProps := decodeAppServiceResponse(t, listResp)["properties"].(map[string]any)
	if listProps["APPSETTING_A"] != "one" || listProps["SECRET_SETTING"] != "updated" {
		t.Fatalf("expected updated app settings from list endpoint, got %v", listProps)
	}
	if _, ok := listProps["FROM_CREATE"]; ok {
		t.Fatalf("expected list endpoint to reflect replacement semantics, got %v", listProps)
	}
}

func TestAppServiceSiteConnectionStringsConfigLifecycle(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-connections?api-version=2024-04-01"
	connectionStringsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-connections/config/connectionstrings?api-version=2024-04-01"
	listConnectionStringsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-connections/config/connectionstrings/list?api-version=2024-04-01"

	createResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{
		"location":"eastus",
		"properties":{
			"siteConfig":{
				"connectionStrings":[
					{"name":"seededDb","value":"Server=tcp:seeded.database.windows.net;Database=app;","type":"SQLAzure","slotSetting":true}
				]
			}
		}
	}`)))
	if err != nil {
		t.Fatalf("create web app with connection strings returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected web app create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	seededResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listConnectionStringsURL, nil))
	if err != nil {
		t.Fatalf("list seeded connection strings returned error: %v", err)
	}
	if seededResp.StatusCode != http.StatusOK {
		t.Fatalf("expected seeded connection strings status 200, got %d; body=%s", seededResp.StatusCode, string(seededResp.RawBody))
	}
	seededProps := decodeAppServiceResponse(t, seededResp)["properties"].(map[string]any)
	seededDB := seededProps["seededDb"].(map[string]any)
	if seededDB["value"] != "Server=tcp:seeded.database.windows.net;Database=app;" || seededDB["type"] != "SQLAzure" || seededDB["slotSetting"] != true {
		t.Fatalf("expected seeded connection string from siteConfig, got %v", seededProps)
	}

	updateResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, connectionStringsURL, []byte(`{
		"properties":{
			"db":{"value":"Server=tcp:sql.database.windows.net;Database=app;","type":"SQLAzure","slotSetting":true},
			"cache":{"value":"redis://cache","type":"Custom"}
		}
	}`)))
	if err != nil {
		t.Fatalf("update connection strings returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected connection strings update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeAppServiceResponse(t, updateResp)
	if updated["name"] != "connectionstrings" || updated["type"] != "Microsoft.Web/sites/config" {
		t.Fatalf("unexpected connection strings identity fields: %v", updated)
	}
	if updated["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-connections/config/connectionstrings" {
		t.Fatalf("unexpected connection strings id: %v", updated["id"])
	}
	updatedProps := updated["properties"].(map[string]any)
	db := updatedProps["db"].(map[string]any)
	if db["value"] != "Server=tcp:sql.database.windows.net;Database=app;" || db["type"] != "SQLAzure" || db["slotSetting"] != true {
		t.Fatalf("expected SQLAzure connection string, got %v", updatedProps)
	}
	cache := updatedProps["cache"].(map[string]any)
	if cache["value"] != "redis://cache" || cache["type"] != "Custom" {
		t.Fatalf("expected custom connection string, got %v", updatedProps)
	}
	if _, ok := updatedProps["seededDb"]; ok {
		t.Fatalf("expected connection string update to replace the prior set, got %v", updatedProps)
	}

	listResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, listConnectionStringsURL, nil))
	if err != nil {
		t.Fatalf("list updated connection strings returned error: %v", err)
	}
	listProps := decodeAppServiceResponse(t, listResp)["properties"].(map[string]any)
	if listProps["db"].(map[string]any)["type"] != "SQLAzure" || listProps["cache"].(map[string]any)["value"] != "redis://cache" {
		t.Fatalf("expected updated connection strings from list endpoint, got %v", listProps)
	}
	if _, ok := listProps["seededDb"]; ok {
		t.Fatalf("expected list endpoint to reflect replacement semantics, got %v", listProps)
	}
}

func TestAppServiceSiteStartStopAndRestartUpdateState(t *testing.T) {
	svc := New()

	siteURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a?api-version=2024-04-01"
	stopURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/stop?api-version=2024-04-01"
	startURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/start?api-version=2024-04-01"
	restartURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/site-a/restart?softRestart=true&synchronous=true&api-version=2024-04-01"

	if resp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPut, siteURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create web app returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected web app create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	stopResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, stopURL, nil))
	if err != nil {
		t.Fatalf("stop web app returned error: %v", err)
	}
	if stopResp.StatusCode != http.StatusOK {
		t.Fatalf("expected stop web app status 200, got %d; body=%s", stopResp.StatusCode, string(stopResp.RawBody))
	}
	stoppedSite := getAppServiceSite(t, svc, siteURL)
	if stoppedSite["properties"].(map[string]any)["state"] != "Stopped" {
		t.Fatalf("expected stopped site state, got %v", stoppedSite)
	}

	startResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, startURL, nil))
	if err != nil {
		t.Fatalf("start web app returned error: %v", err)
	}
	if startResp.StatusCode != http.StatusOK {
		t.Fatalf("expected start web app status 200, got %d; body=%s", startResp.StatusCode, string(startResp.RawBody))
	}
	startedSite := getAppServiceSite(t, svc, siteURL)
	if startedSite["properties"].(map[string]any)["state"] != "Running" {
		t.Fatalf("expected started site state, got %v", startedSite)
	}

	restartResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, restartURL, nil))
	if err != nil {
		t.Fatalf("restart web app returned error: %v", err)
	}
	if restartResp.StatusCode != http.StatusOK {
		t.Fatalf("expected restart web app status 200, got %d; body=%s", restartResp.StatusCode, string(restartResp.RawBody))
	}
	restartedSite := getAppServiceSite(t, svc, siteURL)
	restartedProps := restartedSite["properties"].(map[string]any)
	if restartedProps["state"] != "Running" || restartedProps["restartCount"].(float64) != 1 || restartedProps["lastRestartedAt"] == "" {
		t.Fatalf("expected restarted site state and metadata, got %v", restartedSite)
	}

	missingResp, err := svc.HandleRequest(appServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Web/sites/missing/stop?api-version=2024-04-01", nil))
	if err != nil {
		t.Fatalf("stop missing web app returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing stop status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func appServiceCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func getAppServiceSite(t *testing.T, svc *AppService, rawURL string) map[string]any {
	t.Helper()
	resp, err := svc.HandleRequest(appServiceCtx(t, http.MethodGet, rawURL, nil))
	if err != nil {
		t.Fatalf("get web app returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected get web app status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	return decodeAppServiceResponse(t, resp)
}

func decodeAppServiceResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
