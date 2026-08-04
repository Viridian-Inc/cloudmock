package cdn

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestProfileLifecycleListAndTemplateProvisioning(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15"
	createResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, profileURL, []byte(`{
		"location":"global",
		"sku":{"name":"Premium_AzureFrontDoor"},
		"tags":{"env":"test"},
		"identity":{"type":"SystemAssigned"},
		"properties":{
			"originResponseTimeoutSeconds":45,
			"logScrubbing":{"state":"Enabled","scrubbingRules":[{"matchVariable":"RequestIPAddress","selectorMatchOperator":"EqualsAny","state":"Enabled"}]}
		}
	}`)))
	if err != nil {
		t.Fatalf("create CDN profile returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create profile status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeCDNResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a" {
		t.Fatalf("unexpected profile id: %v", created["id"])
	}
	if created["name"] != "frontdoor-a" || created["type"] != "Microsoft.Cdn/profiles" || created["kind"] != "frontdoor" || created["location"] != "global" {
		t.Fatalf("unexpected profile identity fields: %v", created)
	}
	if created["sku"].(map[string]any)["name"] != "Premium_AzureFrontDoor" {
		t.Fatalf("unexpected profile sku: %v", created["sku"])
	}
	properties := created["properties"].(map[string]any)
	if properties["originResponseTimeoutSeconds"] != float64(45) || properties["provisioningState"] != "Succeeded" || properties["resourceState"] != "Active" || properties["frontDoorId"] == "" {
		t.Fatalf("unexpected profile properties: %v", properties)
	}

	getResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, profileURL, nil))
	if err != nil {
		t.Fatalf("get CDN profile returned error: %v", err)
	}
	got := decodeCDNResponse(t, getResp)
	if got["properties"].(map[string]any)["frontDoorId"] != properties["frontDoorId"] {
		t.Fatalf("expected frontDoorId to be stable, got %v then %v", properties["frontDoorId"], got["properties"].(map[string]any)["frontDoorId"])
	}

	_, err = svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.Cdn/profiles/frontdoor-b?api-version=2025-04-15", []byte(`{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`)))
	if err != nil {
		t.Fatalf("create second CDN profile returned error: %v", err)
	}
	_, err = svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-c?api-version=2025-04-15", []byte(`{"location":"global","sku":{"name":"Premium_AzureFrontDoor"}}`)))
	if err != nil {
		t.Fatalf("create other subscription CDN profile returned error: %v", err)
	}

	rgListResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles?api-version=2025-04-15", nil))
	if err != nil {
		t.Fatalf("list CDN profiles by resource group returned error: %v", err)
	}
	rgList := decodeCDNResponse(t, rgListResp)
	rgValues := rgList["value"].([]any)
	if len(rgValues) != 1 || rgValues[0].(map[string]any)["name"] != "frontdoor-a" {
		t.Fatalf("expected only rg-a profile in resource group list, got %v", rgList)
	}

	subListResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Cdn/profiles?api-version=2025-04-15", nil))
	if err != nil {
		t.Fatalf("list CDN profiles by subscription returned error: %v", err)
	}
	subList := decodeCDNResponse(t, subListResp)
	subValues := subList["value"].([]any)
	if len(subValues) != 2 || subValues[0].(map[string]any)["name"] != "frontdoor-a" || subValues[1].(map[string]any)["name"] != "frontdoor-b" {
		t.Fatalf("expected stable subscription profile list, got %v", subList)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Cdn/profiles",
		"name":     "frontdoor-template",
		"location": "global",
		"sku":      map[string]any{"name": "Standard_AzureFrontDoor"},
		"tags":     map[string]any{"env": "template"},
		"properties": map[string]any{
			"originResponseTimeoutSeconds": float64(60),
		},
	})
	if err != nil {
		t.Fatalf("provision CDN profile returned error: %v", err)
	}
	templateProfile := templateResult.(map[string]any)
	if templateProfile["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-template" {
		t.Fatalf("unexpected template profile id: %v", templateProfile["id"])
	}

	deleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, profileURL, nil))
	if err != nil {
		t.Fatalf("delete CDN profile returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete profile status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	missingDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, profileURL, nil))
	if err != nil {
		t.Fatalf("delete missing CDN profile returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing delete status 204, got %d; body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}
}

func TestEndpointLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, profileURL, []byte(`{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`))); err != nil {
		t.Fatalf("create parent CDN profile returned error: %v", err)
	}

	endpointURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15"
	createResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, endpointURL, []byte(`{
		"location":"westus",
		"tags":{"env":"test"},
		"properties":{
			"originHostHeader":"www.example.com",
			"originPath":"/photos",
			"contentTypesToCompress":["text/html","application/octet-stream"],
			"isCompressionEnabled":true,
			"isHttpAllowed":true,
			"isHttpsAllowed":true,
			"queryStringCachingBehavior":"BypassCaching",
			"defaultOriginGroup":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/originGroup1"},
			"origins":[{"name":"origin1","properties":{"hostName":"www.example.com","httpPort":80,"httpsPort":443,"originHostHeader":"www.example.com","priority":1,"weight":50,"enabled":true}}],
			"originGroups":[{"name":"originGroup1","properties":{"origins":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin1"}]}}]
		}
	}`)))
	if err != nil {
		t.Fatalf("create CDN endpoint returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create endpoint status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeCDNResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a" {
		t.Fatalf("unexpected endpoint id: %v", created["id"])
	}
	if created["name"] != "endpoint-a" || created["type"] != "Microsoft.Cdn/profiles/endpoints" || created["location"] != "westus" {
		t.Fatalf("unexpected endpoint identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["hostName"] != "endpoint-a.azureedge.net" || properties["provisioningState"] != "Succeeded" || properties["resourceState"] != "Running" {
		t.Fatalf("unexpected endpoint projected properties: %v", properties)
	}
	if properties["originHostHeader"] != "www.example.com" || properties["originPath"] != "/photos" || properties["queryStringCachingBehavior"] != "BypassCaching" {
		t.Fatalf("expected endpoint request properties to be preserved, got %v", properties)
	}
	if len(properties["origins"].([]any)) != 1 || len(properties["originGroups"].([]any)) != 1 {
		t.Fatalf("expected nested origins and origin groups to be preserved, got %v", properties)
	}

	getResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, endpointURL, nil))
	if err != nil {
		t.Fatalf("get CDN endpoint returned error: %v", err)
	}
	got := decodeCDNResponse(t, getResp)
	if got["properties"].(map[string]any)["hostName"] != properties["hostName"] {
		t.Fatalf("expected endpoint hostName to be stable, got %v then %v", properties["hostName"], got["properties"].(map[string]any)["hostName"])
	}

	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-b?api-version=2025-04-15", []byte(`{"location":"westus","properties":{"origins":[{"name":"origin1","properties":{"hostName":"b.example.com"}}]}}`))); err != nil {
		t.Fatalf("create second CDN endpoint returned error: %v", err)
	}

	listResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints?api-version=2025-04-15", nil))
	if err != nil {
		t.Fatalf("list CDN endpoints returned error: %v", err)
	}
	list := decodeCDNResponse(t, listResp)
	values := list["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "endpoint-a" || values[1].(map[string]any)["name"] != "endpoint-b" {
		t.Fatalf("expected stable endpoint list, got %v", list)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.Cdn/profiles/endpoints",
		"name":     "frontdoor-a/endpoint-template",
		"location": "westus",
		"tags":     map[string]any{"env": "template"},
		"properties": map[string]any{
			"origins": []any{map[string]any{
				"name":       "origin1",
				"properties": map[string]any{"hostName": "template.example.com"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("provision CDN endpoint returned error: %v", err)
	}
	templateEndpoint := templateResult.(map[string]any)
	if templateEndpoint["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-template" {
		t.Fatalf("unexpected template endpoint id: %v", templateEndpoint["id"])
	}

	missingProfileResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/missing/endpoints/orphan?api-version=2025-04-15", []byte(`{"location":"westus","properties":{"origins":[{"name":"origin1","properties":{"hostName":"www.example.com"}}]}}`)))
	if err != nil {
		t.Fatalf("create CDN endpoint under missing profile returned error: %v", err)
	}
	if missingProfileResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing profile status 404, got %d; body=%s", missingProfileResp.StatusCode, string(missingProfileResp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, endpointURL, nil))
	if err != nil {
		t.Fatalf("delete CDN endpoint returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete endpoint status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	missingDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, endpointURL, nil))
	if err != nil {
		t.Fatalf("delete missing CDN endpoint returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing endpoint delete status 204, got %d; body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}
}

func TestOriginGroupLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, profileURL, []byte(`{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`))); err != nil {
		t.Fatalf("create parent CDN profile returned error: %v", err)
	}
	endpointURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, endpointURL, []byte(`{"location":"westus","properties":{"origins":[{"name":"origin1","properties":{"hostName":"www.example.com"}}]}}`))); err != nil {
		t.Fatalf("create parent CDN endpoint returned error: %v", err)
	}

	originGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-a?api-version=2025-04-15"
	createResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, originGroupURL, []byte(`{
		"properties":{
			"healthProbeSettings":{
				"probePath":"/health.aspx",
				"probeRequestType":"GET",
				"probeProtocol":"Http",
				"probeIntervalInSeconds":120
			},
			"origins":[
				{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin1"}
			],
			"responseBasedOriginErrorDetectionSettings":{
				"responseBasedDetectedErrorTypes":"TcpErrorsOnly",
				"responseBasedFailoverThresholdPercentage":10
			},
			"trafficRestorationTimeToHealedOrNewEndpointsInMinutes":15
		}
	}`)))
	if err != nil {
		t.Fatalf("create CDN origin group returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create origin group status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeCDNResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-a" {
		t.Fatalf("unexpected origin group id: %v", created["id"])
	}
	if created["name"] != "origin-group-a" || created["type"] != "Microsoft.Cdn/profiles/endpoints/originGroups" {
		t.Fatalf("unexpected origin group identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" || properties["resourceState"] != "Active" {
		t.Fatalf("unexpected origin group projected properties: %v", properties)
	}
	if properties["trafficRestorationTimeToHealedOrNewEndpointsInMinutes"] != float64(15) {
		t.Fatalf("expected traffic restoration setting to be preserved, got %v", properties)
	}
	if len(properties["origins"].([]any)) != 1 {
		t.Fatalf("expected origin references to be preserved, got %v", properties["origins"])
	}

	parentEndpointResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, endpointURL, nil))
	if err != nil {
		t.Fatalf("get parent endpoint returned error: %v", err)
	}
	parentEndpoint := decodeCDNResponse(t, parentEndpointResp)
	parentOriginGroups := parentEndpoint["properties"].(map[string]any)["originGroups"].([]any)
	if len(parentOriginGroups) != 1 || parentOriginGroups[0].(map[string]any)["name"] != "origin-group-a" {
		t.Fatalf("expected parent endpoint to project origin group, got %v", parentEndpoint)
	}

	getResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, originGroupURL, nil))
	if err != nil {
		t.Fatalf("get CDN origin group returned error: %v", err)
	}
	got := decodeCDNResponse(t, getResp)
	if got["properties"].(map[string]any)["resourceState"] != "Active" {
		t.Fatalf("expected origin group resource state to be stable, got %v", got)
	}

	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-b?api-version=2025-04-15", []byte(`{"properties":{"origins":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin2"}]}}`))); err != nil {
		t.Fatalf("create second CDN origin group returned error: %v", err)
	}

	listResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups?api-version=2025-04-15", nil))
	if err != nil {
		t.Fatalf("list CDN origin groups returned error: %v", err)
	}
	list := decodeCDNResponse(t, listResp)
	values := list["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "origin-group-a" || values[1].(map[string]any)["name"] != "origin-group-b" {
		t.Fatalf("expected stable origin group list, got %v", list)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.Cdn/profiles/endpoints/originGroups",
		"name": "frontdoor-a/endpoint-a/origin-group-template",
		"properties": map[string]any{
			"origins": []any{map[string]any{
				"id": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-template",
			}},
		},
	})
	if err != nil {
		t.Fatalf("provision CDN origin group returned error: %v", err)
	}
	templateOriginGroup := templateResult.(map[string]any)
	if templateOriginGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/originGroups/origin-group-template" {
		t.Fatalf("unexpected template origin group id: %v", templateOriginGroup["id"])
	}

	missingEndpointResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/missing/originGroups/orphan?api-version=2025-04-15", []byte(`{"properties":{"origins":[{"id":"origin1"}]}}`)))
	if err != nil {
		t.Fatalf("create CDN origin group under missing endpoint returned error: %v", err)
	}
	if missingEndpointResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing endpoint status 404, got %d; body=%s", missingEndpointResp.StatusCode, string(missingEndpointResp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, originGroupURL, nil))
	if err != nil {
		t.Fatalf("delete CDN origin group returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete origin group status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	missingDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, originGroupURL, nil))
	if err != nil {
		t.Fatalf("delete missing CDN origin group returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing origin group delete status 204, got %d; body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}
}

func TestOriginLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, profileURL, []byte(`{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`))); err != nil {
		t.Fatalf("create parent CDN profile returned error: %v", err)
	}
	endpointURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, endpointURL, []byte(`{"location":"westus","properties":{}}`))); err != nil {
		t.Fatalf("create parent CDN endpoint returned error: %v", err)
	}

	originURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-a?api-version=2025-04-15"
	createResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, originURL, []byte(`{
		"properties":{
			"hostName":"www.example.com",
			"httpPort":80,
			"httpsPort":443,
			"originHostHeader":"www.example.com",
			"priority":1,
			"weight":50,
			"enabled":true,
			"privateLinkAlias":"alias-a",
			"privateLinkLocation":"eastus",
			"privateEndpointStatus":"Pending",
			"privateLinkApprovalMessage":"approve me"
		}
	}`)))
	if err != nil {
		t.Fatalf("create CDN origin returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create origin status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeCDNResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-a" {
		t.Fatalf("unexpected origin id: %v", created["id"])
	}
	if created["name"] != "origin-a" || created["type"] != "Microsoft.Cdn/profiles/endpoints/origins" {
		t.Fatalf("unexpected origin identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["hostName"] != "www.example.com" || properties["originHostHeader"] != "www.example.com" {
		t.Fatalf("expected origin host properties to be preserved, got %v", properties)
	}
	if properties["httpPort"] != float64(80) || properties["httpsPort"] != float64(443) || properties["priority"] != float64(1) || properties["weight"] != float64(50) || properties["enabled"] != true {
		t.Fatalf("expected origin routing properties to be preserved, got %v", properties)
	}
	if properties["privateLinkAlias"] != "alias-a" || properties["privateLinkLocation"] != "eastus" || properties["privateEndpointStatus"] != "Pending" || properties["privateLinkApprovalMessage"] != "approve me" {
		t.Fatalf("expected private link properties to be preserved, got %v", properties)
	}
	if properties["provisioningState"] != "Succeeded" || properties["resourceState"] != "Active" {
		t.Fatalf("unexpected origin projected properties: %v", properties)
	}

	parentEndpointResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, endpointURL, nil))
	if err != nil {
		t.Fatalf("get parent endpoint returned error: %v", err)
	}
	parentEndpoint := decodeCDNResponse(t, parentEndpointResp)
	parentOrigins := parentEndpoint["properties"].(map[string]any)["origins"].([]any)
	if len(parentOrigins) != 1 || parentOrigins[0].(map[string]any)["name"] != "origin-a" {
		t.Fatalf("expected parent endpoint to project origin, got %v", parentEndpoint)
	}

	getResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, originURL, nil))
	if err != nil {
		t.Fatalf("get CDN origin returned error: %v", err)
	}
	got := decodeCDNResponse(t, getResp)
	if got["properties"].(map[string]any)["resourceState"] != "Active" {
		t.Fatalf("expected origin resource state to be stable, got %v", got)
	}

	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-b?api-version=2025-04-15", []byte(`{"properties":{"hostName":"b.example.com","priority":2,"weight":25}}`))); err != nil {
		t.Fatalf("create second CDN origin returned error: %v", err)
	}

	listResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins?api-version=2025-04-15", nil))
	if err != nil {
		t.Fatalf("list CDN origins returned error: %v", err)
	}
	list := decodeCDNResponse(t, listResp)
	values := list["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "origin-a" || values[1].(map[string]any)["name"] != "origin-b" {
		t.Fatalf("expected stable origin list, got %v", list)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.Cdn/profiles/endpoints/origins",
		"name": "frontdoor-a/endpoint-a/origin-template",
		"properties": map[string]any{
			"hostName":         "template.example.com",
			"originHostHeader": "template.example.com",
			"priority":         float64(3),
			"weight":           float64(10),
			"enabled":          true,
		},
	})
	if err != nil {
		t.Fatalf("provision CDN origin returned error: %v", err)
	}
	templateOrigin := templateResult.(map[string]any)
	if templateOrigin["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/origins/origin-template" {
		t.Fatalf("unexpected template origin id: %v", templateOrigin["id"])
	}

	missingEndpointResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/missing/origins/orphan?api-version=2025-04-15", []byte(`{"properties":{"hostName":"orphan.example.com"}}`)))
	if err != nil {
		t.Fatalf("create CDN origin under missing endpoint returned error: %v", err)
	}
	if missingEndpointResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing endpoint status 404, got %d; body=%s", missingEndpointResp.StatusCode, string(missingEndpointResp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, originURL, nil))
	if err != nil {
		t.Fatalf("delete CDN origin returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete origin status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	parentAfterDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, endpointURL, nil))
	if err != nil {
		t.Fatalf("get parent endpoint after origin delete returned error: %v", err)
	}
	parentAfterDelete := decodeCDNResponse(t, parentAfterDeleteResp)
	for _, item := range parentAfterDelete["properties"].(map[string]any)["origins"].([]any) {
		if item.(map[string]any)["name"] == "origin-a" {
			t.Fatalf("expected parent endpoint projection to remove deleted origin, got %v", parentAfterDelete)
		}
	}
	missingDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, originURL, nil))
	if err != nil {
		t.Fatalf("delete missing CDN origin returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing origin delete status 204, got %d; body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}
}

func TestCustomDomainLifecycleAndTemplateProvisioning(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, profileURL, []byte(`{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`))); err != nil {
		t.Fatalf("create parent CDN profile returned error: %v", err)
	}
	endpointURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, endpointURL, []byte(`{"location":"westus","properties":{}}`))); err != nil {
		t.Fatalf("create parent CDN endpoint returned error: %v", err)
	}

	customDomainURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com?api-version=2025-04-15"
	createResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, customDomainURL, []byte(`{
		"properties":{
			"hostName":"www.contoso.com",
			"validationData":"icp-license-123",
			"customHttpsParameters":{
				"certificateSource":"Cdn",
				"certificateSourceParameters":{"certificateType":"Dedicated"},
				"minimumTlsVersion":"TLS12",
				"protocolType":"ServerNameIndication"
			}
		}
	}`)))
	if err != nil {
		t.Fatalf("create CDN custom domain returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create custom domain status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeCDNResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com" {
		t.Fatalf("unexpected custom domain id: %v", created["id"])
	}
	if created["name"] != "www-contoso-com" || created["type"] != "Microsoft.Cdn/profiles/endpoints/customDomains" {
		t.Fatalf("unexpected custom domain identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["hostName"] != "www.contoso.com" || properties["validationData"] != "icp-license-123" {
		t.Fatalf("expected custom domain request properties to be preserved, got %v", properties)
	}
	if properties["provisioningState"] != "Succeeded" || properties["resourceState"] != "Active" {
		t.Fatalf("unexpected custom domain projected state: %v", properties)
	}
	if properties["customHttpsProvisioningState"] != "Enabling" || properties["customHttpsProvisioningSubstate"] != "PendingDomainControlValidationREquestApproval" {
		t.Fatalf("unexpected custom HTTPS provisioning state: %v", properties)
	}
	httpsParams := properties["customHttpsParameters"].(map[string]any)
	if httpsParams["certificateSource"] != "Cdn" || httpsParams["minimumTlsVersion"] != "TLS12" || httpsParams["protocolType"] != "ServerNameIndication" {
		t.Fatalf("expected custom HTTPS parameters to be preserved, got %v", httpsParams)
	}

	parentEndpointResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, endpointURL, nil))
	if err != nil {
		t.Fatalf("get parent endpoint returned error: %v", err)
	}
	parentEndpoint := decodeCDNResponse(t, parentEndpointResp)
	parentCustomDomains := parentEndpoint["properties"].(map[string]any)["customDomains"].([]any)
	if len(parentCustomDomains) != 1 || parentCustomDomains[0].(map[string]any)["name"] != "www-contoso-com" {
		t.Fatalf("expected parent endpoint to project custom domain, got %v", parentEndpoint)
	}

	getResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, customDomainURL, nil))
	if err != nil {
		t.Fatalf("get CDN custom domain returned error: %v", err)
	}
	got := decodeCDNResponse(t, getResp)
	if got["properties"].(map[string]any)["hostName"] != "www.contoso.com" {
		t.Fatalf("expected custom domain hostName to be stable, got %v", got)
	}

	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/api-contoso-com?api-version=2025-04-15", []byte(`{"properties":{"hostName":"api.contoso.com"}}`))); err != nil {
		t.Fatalf("create second CDN custom domain returned error: %v", err)
	}

	listResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains?api-version=2025-04-15", nil))
	if err != nil {
		t.Fatalf("list CDN custom domains returned error: %v", err)
	}
	list := decodeCDNResponse(t, listResp)
	values := list["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "api-contoso-com" || values[1].(map[string]any)["name"] != "www-contoso-com" {
		t.Fatalf("expected stable custom domain list, got %v", list)
	}

	templateResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type": "Microsoft.Cdn/profiles/endpoints/customDomains",
		"name": "frontdoor-a/endpoint-a/template-contoso-com",
		"properties": map[string]any{
			"hostName": "template.contoso.com",
		},
	})
	if err != nil {
		t.Fatalf("provision CDN custom domain returned error: %v", err)
	}
	templateCustomDomain := templateResult.(map[string]any)
	if templateCustomDomain["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/template-contoso-com" {
		t.Fatalf("unexpected template custom domain id: %v", templateCustomDomain["id"])
	}

	missingEndpointResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/missing/customDomains/orphan?api-version=2025-04-15", []byte(`{"properties":{"hostName":"orphan.contoso.com"}}`)))
	if err != nil {
		t.Fatalf("create CDN custom domain under missing endpoint returned error: %v", err)
	}
	if missingEndpointResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing endpoint status 404, got %d; body=%s", missingEndpointResp.StatusCode, string(missingEndpointResp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, customDomainURL, nil))
	if err != nil {
		t.Fatalf("delete CDN custom domain returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete custom domain status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	parentAfterDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, endpointURL, nil))
	if err != nil {
		t.Fatalf("get parent endpoint after custom domain delete returned error: %v", err)
	}
	parentAfterDelete := decodeCDNResponse(t, parentAfterDeleteResp)
	for _, item := range parentAfterDelete["properties"].(map[string]any)["customDomains"].([]any) {
		if item.(map[string]any)["name"] == "www-contoso-com" {
			t.Fatalf("expected parent endpoint projection to remove deleted custom domain, got %v", parentAfterDelete)
		}
	}
	missingDeleteResp, err := svc.HandleRequest(cdnCtx(t, http.MethodDelete, customDomainURL, nil))
	if err != nil {
		t.Fatalf("delete missing CDN custom domain returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing custom domain delete status 204, got %d; body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}
}

func TestCustomDomainCustomHTTPSActions(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, profileURL, []byte(`{"location":"global","sku":{"name":"Standard_AzureFrontDoor"}}`))); err != nil {
		t.Fatalf("create parent CDN profile returned error: %v", err)
	}
	endpointURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, endpointURL, []byte(`{"location":"westus","properties":{}}`))); err != nil {
		t.Fatalf("create parent CDN endpoint returned error: %v", err)
	}
	customDomainURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com?api-version=2025-04-15"
	if _, err := svc.HandleRequest(cdnCtx(t, http.MethodPut, customDomainURL, []byte(`{"properties":{"hostName":"www.contoso.com","validationData":"validationdata"}}`))); err != nil {
		t.Fatalf("create CDN custom domain returned error: %v", err)
	}

	enableURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com/enableCustomHttps?api-version=2025-04-15"
	enableResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPost, enableURL, []byte(`{
		"certificateSource":"Cdn",
		"certificateSourceParameters":{"certificateType":"Dedicated"},
		"minimumTlsVersion":"TLS12",
		"protocolType":"ServerNameIndication"
	}`)))
	if err != nil {
		t.Fatalf("enable custom HTTPS returned error: %v", err)
	}
	if enableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected enable custom HTTPS status 200, got %d; body=%s", enableResp.StatusCode, string(enableResp.RawBody))
	}
	enabled := decodeCDNResponse(t, enableResp)
	enabledProperties := enabled["properties"].(map[string]any)
	if enabledProperties["customHttpsProvisioningState"] != "Enabled" || enabledProperties["customHttpsProvisioningSubstate"] != "CertificateDeployed" {
		t.Fatalf("expected custom HTTPS to be enabled, got %v", enabledProperties)
	}
	enabledHTTPSParams := enabledProperties["customHttpsParameters"].(map[string]any)
	if enabledHTTPSParams["certificateSource"] != "Cdn" || enabledHTTPSParams["minimumTlsVersion"] != "TLS12" || enabledHTTPSParams["protocolType"] != "ServerNameIndication" {
		t.Fatalf("expected custom HTTPS parameters to be stored, got %v", enabledHTTPSParams)
	}

	getAfterEnableResp, err := svc.HandleRequest(cdnCtx(t, http.MethodGet, customDomainURL, nil))
	if err != nil {
		t.Fatalf("get custom domain after enable returned error: %v", err)
	}
	gotAfterEnable := decodeCDNResponse(t, getAfterEnableResp)
	if gotAfterEnable["properties"].(map[string]any)["customHttpsProvisioningState"] != "Enabled" {
		t.Fatalf("expected enabled state to persist, got %v", gotAfterEnable)
	}

	disableURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/www-contoso-com/disableCustomHttps?api-version=2025-04-15"
	disableResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPost, disableURL, nil))
	if err != nil {
		t.Fatalf("disable custom HTTPS returned error: %v", err)
	}
	if disableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected disable custom HTTPS status 200, got %d; body=%s", disableResp.StatusCode, string(disableResp.RawBody))
	}
	disabled := decodeCDNResponse(t, disableResp)
	disabledProperties := disabled["properties"].(map[string]any)
	if disabledProperties["customHttpsProvisioningState"] != "Disabled" || disabledProperties["customHttpsProvisioningSubstate"] != "CertificateDeleted" {
		t.Fatalf("expected custom HTTPS to be disabled, got %v", disabledProperties)
	}

	missingResp, err := svc.HandleRequest(cdnCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cdn/profiles/frontdoor-a/endpoints/endpoint-a/customDomains/missing/enableCustomHttps?api-version=2025-04-15", []byte(`{"certificateSource":"Cdn","certificateSourceParameters":{"certificateType":"Dedicated"},"protocolType":"ServerNameIndication"}`)))
	if err != nil {
		t.Fatalf("enable missing custom domain returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing custom domain status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func cdnCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeCDNResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
