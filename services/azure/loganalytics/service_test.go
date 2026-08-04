package loganalytics

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestWorkspaceLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	workspaceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a?api-version=2025-02-01"
	createResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodPut, workspaceURL, []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"sku":{"name":"PerGB2018"},
			"retentionInDays":30,
			"publicNetworkAccessForIngestion":"Enabled",
			"publicNetworkAccessForQuery":"Enabled",
			"features":{"enableLogAccessUsingOnlyResourcePermissions":true}
		}
	}`)))
	if err != nil {
		t.Fatalf("create workspace returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create workspace status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	workspace := decodeLogAnalyticsResponse(t, createResp)
	if workspace["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-a" {
		t.Fatalf("unexpected workspace id: %v", workspace["id"])
	}
	if workspace["name"] != "law-a" || workspace["type"] != "Microsoft.OperationalInsights/workspaces" || workspace["location"] != "eastus" {
		t.Fatalf("unexpected workspace identity fields: %v", workspace)
	}
	props := workspace["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" || props["customerId"] == "" {
		t.Fatalf("unexpected workspace provisioning fields: %v", props)
	}
	if props["retentionInDays"] != float64(30) || props["publicNetworkAccessForIngestion"] != "Enabled" || props["publicNetworkAccessForQuery"] != "Enabled" {
		t.Fatalf("unexpected workspace retention/network properties: %v", props)
	}
	if props["sku"].(map[string]any)["name"] != "PerGB2018" {
		t.Fatalf("expected sku to be retained in properties, got %v", props["sku"])
	}
	if props["features"].(map[string]any)["enableLogAccessUsingOnlyResourcePermissions"] != true {
		t.Fatalf("expected workspace features to be preserved, got %v", props["features"])
	}
	customerID := props["customerId"]

	updateResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodPut, workspaceURL, []byte(`{
		"location":"westus2",
		"tags":{"env":"prod"},
		"properties":{
			"sku":{"name":"CapacityReservation","capacityReservationLevel":100},
			"retentionInDays":90
		}
	}`)))
	if err != nil {
		t.Fatalf("update workspace returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update workspace status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeLogAnalyticsResponse(t, updateResp)
	updatedProps := updated["properties"].(map[string]any)
	if updated["location"] != "westus2" || updatedProps["retentionInDays"] != float64(90) {
		t.Fatalf("unexpected updated workspace: %v", updated)
	}
	if updatedProps["customerId"] != customerID {
		t.Fatalf("expected customerId to remain stable across updates, got %v then %v", customerID, updatedProps["customerId"])
	}
	if updatedProps["sku"].(map[string]any)["name"] != "CapacityReservation" || updatedProps["sku"].(map[string]any)["capacityReservationLevel"] != float64(100) {
		t.Fatalf("expected updated capacity reservation sku, got %v", updatedProps["sku"])
	}

	getResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodGet, workspaceURL, nil))
	if err != nil {
		t.Fatalf("get workspace returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get workspace status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	createOther := func(rawURL string) {
		t.Helper()
		resp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodPut, rawURL, []byte(`{"location":"eastus","properties":{"sku":{"name":"PerGB2018"}}}`)))
		if err != nil {
			t.Fatalf("create workspace %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create workspace 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}
	createOther("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-b?api-version=2025-02-01")
	createOther("https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.OperationalInsights/workspaces/law-c?api-version=2025-02-01")

	listResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces?api-version=2025-02-01", nil))
	if err != nil {
		t.Fatalf("list workspaces returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list workspaces status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeLogAnalyticsResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "law-a" || values[1].(map[string]any)["name"] != "law-b" {
		t.Fatalf("expected rg-a workspaces sorted by name, got %v", listed)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.OperationalInsights/workspaces",
		"name":     "law-template",
		"location": "eastus",
		"tags":     map[string]any{"env": "template"},
		"properties": map[string]any{
			"sku":             map[string]any{"name": "PerGB2018"},
			"retentionInDays": float64(30),
		},
	})
	if err != nil {
		t.Fatalf("provision workspace returned error: %v", err)
	}
	templateWorkspace := templateResult.(map[string]any)
	if templateWorkspace["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.OperationalInsights/workspaces/law-template" {
		t.Fatalf("unexpected provisioned workspace id: %v", templateWorkspace["id"])
	}

	deleteResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodDelete, workspaceURL, nil))
	if err != nil {
		t.Fatalf("delete workspace returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete workspace status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deleteMissingResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodDelete, workspaceURL, nil))
	if err != nil {
		t.Fatalf("delete missing workspace returned error: %v", err)
	}
	if deleteMissingResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing workspace status 204, got %d; body=%s", deleteMissingResp.StatusCode, string(deleteMissingResp.RawBody))
	}
}

func TestWorkspaceServiceKeys(t *testing.T) {
	svc := New()

	seen := make(map[routing.ServiceKey]bool)
	for _, key := range svc.ServiceKeys() {
		seen[key] = true
	}

	for _, expected := range []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.OperationalInsights/workspaces", APIVersion: "2025-02-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.OperationalInsights/query", APIVersion: "v1"},
	} {
		if !seen[expected] {
			t.Fatalf("expected Log Analytics service key %#v, got %#v", expected, svc.ServiceKeys())
		}
	}
}

func TestWorkspaceQueryPost(t *testing.T) {
	svc := New()

	resp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodPost, "https://api.loganalytics.azure.com/v1/workspaces/ws-123/query", []byte(`{
		"query":"AzureActivity | summarize count() by Category",
		"timespan":"PT12H"
	}`)))
	if err != nil {
		t.Fatalf("workspace query returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected query status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	out := decodeLogAnalyticsResponse(t, resp)
	tables := out["tables"].([]any)
	if len(tables) != 1 {
		t.Fatalf("expected one query result table, got %v", out)
	}
	table := tables[0].(map[string]any)
	if table["name"] != "PrimaryResult" {
		t.Fatalf("expected PrimaryResult table, got %v", table)
	}
	columns := table["columns"].([]any)
	rows := table["rows"].([]any)
	if len(columns) != 2 || len(rows) == 0 {
		t.Fatalf("expected category count columns and rows, got %v", table)
	}
	if columns[0].(map[string]any)["name"] != "Category" || columns[1].(map[string]any)["name"] != "count_" {
		t.Fatalf("unexpected query columns: %v", columns)
	}
	firstRow := rows[0].([]any)
	if firstRow[0] != "Administrative" || firstRow[1] != float64(1) {
		t.Fatalf("unexpected query row: %v", firstRow)
	}

	legacyResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodPost, "https://api.loganalytics.io/v1/workspaces/ws-123/query", []byte(`{"query":"Heartbeat | take 1"}`)))
	if err != nil {
		t.Fatalf("legacy host workspace query returned error: %v", err)
	}
	if legacyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected legacy query status 200, got %d; body=%s", legacyResp.StatusCode, string(legacyResp.RawBody))
	}
	legacyOut := decodeLogAnalyticsResponse(t, legacyResp)
	legacyTable := legacyOut["tables"].([]any)[0].(map[string]any)
	legacyRows := legacyTable["rows"].([]any)
	if len(legacyRows) != 1 || legacyRows[0].([]any)[1] != "ws-123" {
		t.Fatalf("expected deterministic workspace id row for legacy query, got %v", legacyOut)
	}

	missingQueryResp, err := svc.HandleRequest(logAnalyticsCtx(t, http.MethodPost, "https://api.loganalytics.azure.com/v1/workspaces/ws-123/query", []byte(`{"timespan":"PT12H"}`)))
	if err != nil {
		t.Fatalf("missing query request returned error: %v", err)
	}
	if missingQueryResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing query status 400, got %d; body=%s", missingQueryResp.StatusCode, string(missingQueryResp.RawBody))
	}
}

func logAnalyticsCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeLogAnalyticsResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
