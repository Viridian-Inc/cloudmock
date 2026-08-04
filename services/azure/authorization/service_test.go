package authorization_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/azure/authorization"
)

func authorizationCtx(t *testing.T, method, targetURL string, body map[string]any) *service.RequestContext {
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
		AccountID:  "00000000-0000-0000-0000-000000000001",
		RawRequest: req,
		Body:       payload,
	}
}

func decodeAuthorizationJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	if resp.RawContentType != "application/json" {
		t.Fatalf("expected application/json, got %q", resp.RawContentType)
	}
	var out map[string]any
	if err := json.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	return out
}

func TestRoleAssignmentLifecycleAtResourceGroupScope(t *testing.T) {
	svc := authorization.New()
	const assignmentID = "11111111-1111-1111-1111-111111111111"
	const principalID = "22222222-2222-2222-2222-222222222222"
	const roleDefinitionID = "/subscriptions/sub-1/providers/Microsoft.Authorization/roleDefinitions/33333333-3333-3333-3333-333333333333"
	const roleAssignmentURL = "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Authorization/roleAssignments/" + assignmentID + "?api-version=2022-04-01"

	createResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodPut, roleAssignmentURL, map[string]any{
		"properties": map[string]any{
			"principalId":      principalID,
			"principalType":    "ServicePrincipal",
			"roleDefinitionId": roleDefinitionID,
			"description":      "grant test access",
		},
	}))
	if err != nil {
		t.Fatalf("create role assignment returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createResp.StatusCode)
	}
	created := decodeAuthorizationJSON(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Authorization/roleAssignments/"+assignmentID {
		t.Fatalf("unexpected role assignment id: %v", created["id"])
	}
	if created["name"] != assignmentID {
		t.Fatalf("unexpected role assignment name: %v", created["name"])
	}
	if created["type"] != "Microsoft.Authorization/roleAssignments" {
		t.Fatalf("unexpected role assignment type: %v", created["type"])
	}
	properties := created["properties"].(map[string]any)
	if properties["scope"] != "/subscriptions/sub-1/resourceGroups/rg-a" {
		t.Fatalf("unexpected scope: %v", properties["scope"])
	}
	if properties["principalId"] != principalID {
		t.Fatalf("unexpected principal id: %v", properties["principalId"])
	}
	if properties["principalType"] != "ServicePrincipal" {
		t.Fatalf("unexpected principal type: %v", properties["principalType"])
	}
	if properties["roleDefinitionId"] != roleDefinitionID {
		t.Fatalf("unexpected role definition id: %v", properties["roleDefinitionId"])
	}

	updateResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodPut, roleAssignmentURL, map[string]any{
		"properties": map[string]any{
			"principalId":      principalID,
			"principalType":    "ServicePrincipal",
			"roleDefinitionId": roleDefinitionID,
			"description":      "updated grant",
		},
	}))
	if err != nil {
		t.Fatalf("update role assignment returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update status 200, got %d", updateResp.StatusCode)
	}

	getResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodGet, roleAssignmentURL, nil))
	if err != nil {
		t.Fatalf("get role assignment returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getResp.StatusCode)
	}

	listResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Authorization/roleAssignments?api-version=2022-04-01", nil))
	if err != nil {
		t.Fatalf("list role assignments returned error: %v", err)
	}
	listed := decodeAuthorizationJSON(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one listed role assignment, got %d", len(values))
	}

	deleteResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodDelete, roleAssignmentURL, nil))
	if err != nil {
		t.Fatalf("delete role assignment returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d", deleteResp.StatusCode)
	}
	deleted := decodeAuthorizationJSON(t, deleteResp)
	if deleted["name"] != assignmentID {
		t.Fatalf("unexpected deleted role assignment name: %v", deleted["name"])
	}

	deleteMissingResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodDelete, roleAssignmentURL, nil))
	if err != nil {
		t.Fatalf("delete missing role assignment returned error: %v", err)
	}
	if deleteMissingResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete missing status 204, got %d", deleteMissingResp.StatusCode)
	}
}

func TestRoleAssignmentLifecycleAtNestedResourceScope(t *testing.T) {
	svc := authorization.New()
	const assignmentID = "44444444-4444-4444-4444-444444444444"
	const targetURL = "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct/providers/Microsoft.Authorization/roleAssignments/" + assignmentID + "?api-version=2022-04-01"

	createResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodPut, targetURL, map[string]any{
		"properties": map[string]any{
			"principalId":      "55555555-5555-5555-5555-555555555555",
			"principalType":    "User",
			"roleDefinitionId": "/subscriptions/sub-1/providers/Microsoft.Authorization/roleDefinitions/66666666-6666-6666-6666-666666666666",
		},
	}))
	if err != nil {
		t.Fatalf("create nested role assignment returned error: %v", err)
	}
	created := decodeAuthorizationJSON(t, createResp)
	properties := created["properties"].(map[string]any)
	if properties["scope"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct" {
		t.Fatalf("unexpected nested scope: %v", properties["scope"])
	}
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acct/providers/Microsoft.Authorization/roleAssignments/"+assignmentID {
		t.Fatalf("unexpected nested role assignment id: %v", created["id"])
	}
}

func TestManagementLockLifecycleAtResourceGroupScope(t *testing.T) {
	svc := authorization.New()
	const lockURL = "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Authorization/locks/protect-rg?api-version=2020-05-01"

	createResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodPut, lockURL, map[string]any{
		"properties": map[string]any{
			"level": "CanNotDelete",
			"notes": "protect test resource group",
		},
	}))
	if err != nil {
		t.Fatalf("create management lock returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create lock status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeAuthorizationJSON(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Authorization/locks/protect-rg" {
		t.Fatalf("unexpected lock id: %v", created["id"])
	}
	if created["name"] != "protect-rg" {
		t.Fatalf("unexpected lock name: %v", created["name"])
	}
	if created["type"] != "Microsoft.Authorization/locks" {
		t.Fatalf("unexpected lock type: %v", created["type"])
	}
	properties := created["properties"].(map[string]any)
	if properties["level"] != "CanNotDelete" {
		t.Fatalf("unexpected lock level: %v", properties["level"])
	}
	if properties["notes"] != "protect test resource group" {
		t.Fatalf("unexpected lock notes: %v", properties["notes"])
	}

	updateResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodPut, lockURL, map[string]any{
		"properties": map[string]any{
			"level": "ReadOnly",
			"notes": "read only test resource group",
		},
	}))
	if err != nil {
		t.Fatalf("update management lock returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update lock status 200, got %d", updateResp.StatusCode)
	}

	getResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodGet, lockURL, nil))
	if err != nil {
		t.Fatalf("get management lock returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get lock status 200, got %d", getResp.StatusCode)
	}

	listResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Authorization/locks?api-version=2020-05-01", nil))
	if err != nil {
		t.Fatalf("list management locks returned error: %v", err)
	}
	listed := decodeAuthorizationJSON(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one listed management lock, got %d", len(values))
	}

	deleteResp, err := svc.HandleRequest(authorizationCtx(t, http.MethodDelete, lockURL, nil))
	if err != nil {
		t.Fatalf("delete management lock returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete lock status 200, got %d", deleteResp.StatusCode)
	}
	deleted := decodeAuthorizationJSON(t, deleteResp)
	if deleted["name"] != "protect-rg" {
		t.Fatalf("unexpected deleted lock name: %v", deleted["name"])
	}
}
