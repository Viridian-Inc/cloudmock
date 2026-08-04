package managedidentity

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestUserAssignedIdentityLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	identityURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-a?api-version=2023-01-31"
	identityPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"}
	}`)
	createResp, err := svc.HandleRequest(identityCtx(t, http.MethodPut, identityURL, identityPayload))
	if err != nil {
		t.Fatalf("create identity returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create identity status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	identity := decodeIdentityResponse(t, createResp)
	if identity["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-a" {
		t.Fatalf("unexpected identity id: %v", identity["id"])
	}
	if identity["name"] != "id-a" || identity["type"] != "Microsoft.ManagedIdentity/userAssignedIdentities" || identity["location"] != "eastus" {
		t.Fatalf("unexpected identity fields: %v", identity)
	}
	if identity["clientId"] != "client-sub-1-rg-a-id-a" || identity["principalId"] != "principal-sub-1-rg-a-id-a" || identity["tenantId"] != "tenant-sub-1" {
		t.Fatalf("unexpected generated identity IDs: %v", identity)
	}

	listResp, err := svc.HandleRequest(identityCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ManagedIdentity/userAssignedIdentities?api-version=2023-01-31", nil))
	if err != nil {
		t.Fatalf("list identities returned error: %v", err)
	}
	listed := decodeIdentityResponse(t, listResp)
	if len(listed["value"].([]any)) != 1 {
		t.Fatalf("expected one identity in list, got %v", listed)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.ManagedIdentity/userAssignedIdentities",
		"name":     "id-b",
		"location": "westus2",
		"tags":     map[string]any{"env": "template"},
	})
	if err != nil {
		t.Fatalf("provision identity returned error: %v", err)
	}
	templateIdentity := templateResult.(map[string]any)
	if templateIdentity["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id-b" {
		t.Fatalf("unexpected template identity id: %v", templateIdentity["id"])
	}
	if templateIdentity["clientId"] != "client-sub-1-rg-a-id-b" {
		t.Fatalf("unexpected template client id: %v", templateIdentity)
	}

	deleteResp, err := svc.HandleRequest(identityCtx(t, http.MethodDelete, identityURL, nil))
	if err != nil {
		t.Fatalf("delete identity returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete identity status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
}

func identityCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeIdentityResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
