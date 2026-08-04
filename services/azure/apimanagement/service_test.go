package apimanagement

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestAPIManagementServiceLifecycleAndLists(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	createResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{
		"location":"eastus",
		"sku":{"name":"Developer","capacity":1},
		"properties":{"publisherEmail":"admin@example.com","publisherName":"cloudmock"}
	}`)))
	if err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeAPIManagementResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima" {
		t.Fatalf("unexpected APIM service id: %v", created["id"])
	}
	if created["name"] != "apima" || created["type"] != "Microsoft.ApiManagement/service" || created["location"] != "eastus" {
		t.Fatalf("unexpected APIM service identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" || properties["publisherEmail"] != "admin@example.com" || properties["publisherName"] != "cloudmock" {
		t.Fatalf("unexpected APIM service properties: %v", properties)
	}
	if gatewayURL := properties["gatewayUrl"].(string); !strings.Contains(gatewayURL, "/devstoreaccount1-apim/apima") {
		t.Fatalf("expected floci-compatible gateway URL, got %q", gatewayURL)
	}
	sku := created["sku"].(map[string]any)
	if sku["name"] != "Developer" || sku["capacity"].(float64) != 1 {
		t.Fatalf("unexpected APIM sku: %v", sku)
	}

	getResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, serviceURL, nil))
	if err != nil {
		t.Fatalf("get APIM service returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	otherRGURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.ApiManagement/service/apimb?api-version=2024-05-01"
	otherSubURL := "https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.ApiManagement/service/apimz?api-version=2024-05-01"
	for _, rawURL := range []string{otherRGURL, otherSubURL} {
		resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, rawURL, []byte(`{"location":"westus","properties":{}}`)))
		if err != nil {
			t.Fatalf("create APIM service %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create status 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}

	listRGResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service?api-version=2024-05-01", nil))
	if err != nil {
		t.Fatalf("list APIM services by resource group returned error: %v", err)
	}
	listRG := decodeAPIManagementResponse(t, listRGResp)
	if values := listRG["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "apima" {
		t.Fatalf("expected only rg-a service, got %v", listRG)
	}

	listSubResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ApiManagement/service?api-version=2024-05-01", nil))
	if err != nil {
		t.Fatalf("list APIM services by subscription returned error: %v", err)
	}
	listSub := decodeAPIManagementResponse(t, listSubResp)
	subValues := listSub["value"].([]any)
	if len(subValues) != 2 || subValues[0].(map[string]any)["name"] != "apima" || subValues[1].(map[string]any)["name"] != "apimb" {
		t.Fatalf("expected sub-1 services sorted by name, got %v", listSub)
	}

	deleteResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodDelete, serviceURL, nil))
	if err != nil {
		t.Fatalf("delete APIM service returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	getDeletedResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, serviceURL, nil))
	if err != nil {
		t.Fatalf("get deleted APIM service returned error: %v", err)
	}
	if getDeletedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted service status 404, got %d; body=%s", getDeletedResp.StatusCode, string(getDeletedResp.RawBody))
	}
}

func TestAPIManagementServiceKeys(t *testing.T) {
	svc := New()
	want := routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.ApiManagement/service", APIVersion: "2024-05-01"}
	for _, got := range svc.ServiceKeys() {
		if got == want {
			return
		}
	}
	t.Fatalf("expected service key %#v in %#v", want, svc.ServiceKeys())
}

func TestAPIManagementAPIsAndOperationsLifecycle(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	createAPIResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, []byte(`{
		"properties":{"displayName":"Catalog API","path":"catalog","protocols":["https"],"serviceUrl":"https://backend.example.test"}
	}`)))
	if err != nil {
		t.Fatalf("create API returned error: %v", err)
	}
	if createAPIResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", createAPIResp.StatusCode, string(createAPIResp.RawBody))
	}
	api := decodeAPIManagementResponse(t, createAPIResp)
	if api["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api" {
		t.Fatalf("unexpected API id: %v", api["id"])
	}
	if api["type"] != "Microsoft.ApiManagement/service/apis" || api["name"] != "catalog-api" {
		t.Fatalf("unexpected API identity fields: %v", api)
	}
	apiProps := api["properties"].(map[string]any)
	if apiProps["displayName"] != "Catalog API" || apiProps["path"] != "catalog" || apiProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected API properties: %v", apiProps)
	}

	listAPIsResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis?api-version=2024-05-01", nil))
	if err != nil {
		t.Fatalf("list APIs returned error: %v", err)
	}
	listAPIs := decodeAPIManagementResponse(t, listAPIsResp)
	if values := listAPIs["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "catalog-api" {
		t.Fatalf("expected one API in list, got %v", listAPIs)
	}

	operationURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item?api-version=2024-05-01"
	createOperationResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, operationURL, []byte(`{
		"properties":{"displayName":"Get item","method":"GET","urlTemplate":"/items/{id}"}
	}`)))
	if err != nil {
		t.Fatalf("create operation returned error: %v", err)
	}
	if createOperationResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected operation create status 201, got %d; body=%s", createOperationResp.StatusCode, string(createOperationResp.RawBody))
	}
	operation := decodeAPIManagementResponse(t, createOperationResp)
	if operation["type"] != "Microsoft.ApiManagement/service/apis/operations" || operation["name"] != "get-item" {
		t.Fatalf("unexpected operation identity fields: %v", operation)
	}
	operationProps := operation["properties"].(map[string]any)
	if operationProps["method"] != "GET" || operationProps["urlTemplate"] != "/items/{id}" {
		t.Fatalf("unexpected operation properties: %v", operationProps)
	}

	listOperationsResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations?api-version=2024-05-01", nil))
	if err != nil {
		t.Fatalf("list operations returned error: %v", err)
	}
	listOperations := decodeAPIManagementResponse(t, listOperationsResp)
	if values := listOperations["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "get-item" {
		t.Fatalf("expected one operation in list, got %v", listOperations)
	}

	deleteOperationResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodDelete, operationURL, nil))
	if err != nil {
		t.Fatalf("delete operation returned error: %v", err)
	}
	if deleteOperationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected operation delete status 200, got %d; body=%s", deleteOperationResp.StatusCode, string(deleteOperationResp.RawBody))
	}

	deleteAPIResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodDelete, apiURL, nil))
	if err != nil {
		t.Fatalf("delete API returned error: %v", err)
	}
	if deleteAPIResp.StatusCode != http.StatusOK {
		t.Fatalf("expected API delete status 200, got %d; body=%s", deleteAPIResp.StatusCode, string(deleteAPIResp.RawBody))
	}
}

func TestAPIManagementOpenAPIImportCreatesGatewayOperations(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/openapi-api?api-version=2024-05-01"
	operationsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/openapi-api/operations?api-version=2024-05-01"

	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	createAPIResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, apiManagementOpenAPIImportPayload(t, "Orders API", map[string]any{
		"/orders/{orderId}": map[string]any{
			"get": map[string]any{
				"operationId": "getOrder",
				"summary":     "Get order",
			},
		},
		"/orders": map[string]any{
			"post": map[string]any{
				"operationId": "createOrder",
				"summary":     "Create order",
			},
		},
	})))
	if err != nil {
		t.Fatalf("import OpenAPI API returned error: %v", err)
	}
	if createAPIResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected OpenAPI API create status 201, got %d; body=%s", createAPIResp.StatusCode, string(createAPIResp.RawBody))
	}

	assertAPIManagementCollectionNames(t, svc, "imported operations", operationsURL, []string{"createOrder", "getOrder"})

	getOrderResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/openapi/orders/42", nil))
	if err != nil {
		t.Fatalf("gateway get order request returned error: %v", err)
	}
	if getOrderResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get order gateway status 200, got %d; body=%s", getOrderResp.StatusCode, string(getOrderResp.RawBody))
	}
	getOrder := decodeAPIManagementResponse(t, getOrderResp)
	if getOrder["apiId"] != "openapi-api" || getOrder["operationId"] != "getOrder" || getOrder["backendPath"] != "/orders/42" {
		t.Fatalf("unexpected get order gateway response: %v", getOrder)
	}

	createOrderResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPost, "http://localhost:4577/devstoreaccount1-apim/apima/openapi/orders", nil))
	if err != nil {
		t.Fatalf("gateway create order request returned error: %v", err)
	}
	if createOrderResp.StatusCode != http.StatusOK {
		t.Fatalf("expected create order gateway status 200, got %d; body=%s", createOrderResp.StatusCode, string(createOrderResp.RawBody))
	}
	createOrder := decodeAPIManagementResponse(t, createOrderResp)
	if createOrder["apiId"] != "openapi-api" || createOrder["operationId"] != "createOrder" || createOrder["backendPath"] != "/orders" {
		t.Fatalf("unexpected create order gateway response: %v", createOrder)
	}

	updateAPIResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, apiManagementOpenAPIImportPayload(t, "Customers API", map[string]any{
		"/customers/{customerId}": map[string]any{
			"get": map[string]any{
				"operationId": "getCustomer",
				"summary":     "Get customer",
			},
		},
	})))
	if err != nil {
		t.Fatalf("reimport OpenAPI API returned error: %v", err)
	}
	if updateAPIResp.StatusCode != http.StatusOK {
		t.Fatalf("expected OpenAPI API update status 200, got %d; body=%s", updateAPIResp.StatusCode, string(updateAPIResp.RawBody))
	}

	assertAPIManagementCollectionNames(t, svc, "reimported operations", operationsURL, []string{"getCustomer"})

	removedRouteResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/openapi/orders/42", nil))
	if err != nil {
		t.Fatalf("gateway removed order request returned error: %v", err)
	}
	if removedRouteResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected removed route status 404, got %d; body=%s", removedRouteResp.StatusCode, string(removedRouteResp.RawBody))
	}

	getCustomerResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/openapi/customers/7", nil))
	if err != nil {
		t.Fatalf("gateway get customer request returned error: %v", err)
	}
	if getCustomerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get customer gateway status 200, got %d; body=%s", getCustomerResp.StatusCode, string(getCustomerResp.RawBody))
	}
	getCustomer := decodeAPIManagementResponse(t, getCustomerResp)
	if getCustomer["apiId"] != "openapi-api" || getCustomer["operationId"] != "getCustomer" || getCustomer["backendPath"] != "/customers/7" {
		t.Fatalf("unexpected get customer gateway response: %v", getCustomer)
	}
}

func TestAPIManagementPolicyLifecycleAtServiceAPIAndOperationScopes(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	operationURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, []byte(`{"properties":{"path":"catalog"}}`))); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, operationURL, []byte(`{"properties":{"method":"GET","urlTemplate":"/items/{id}"}}`))); err != nil {
		t.Fatalf("create operation returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected operation create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	policyBody := []byte(`{"properties":{"format":"rawxml","value":"<policies><inbound><base /></inbound></policies>"}}`)
	policyURLs := []struct {
		name         string
		rawURL       string
		expectedType string
	}{
		{
			name:         "service",
			rawURL:       "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/policies/policy?api-version=2024-05-01",
			expectedType: "Microsoft.ApiManagement/service/policies",
		},
		{
			name:         "api",
			rawURL:       "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/policies/policy?api-version=2024-05-01",
			expectedType: "Microsoft.ApiManagement/service/apis/policies",
		},
		{
			name:         "operation",
			rawURL:       "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item/policies/policy?api-version=2024-05-01",
			expectedType: "Microsoft.ApiManagement/service/apis/operations/policies",
		},
	}

	for _, policyURL := range policyURLs {
		createResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, policyURL.rawURL, policyBody))
		if err != nil {
			t.Fatalf("create %s policy returned error: %v", policyURL.name, err)
		}
		if createResp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s policy create status 200, got %d; body=%s", policyURL.name, createResp.StatusCode, string(createResp.RawBody))
		}
		created := decodeAPIManagementResponse(t, createResp)
		if created["name"] != "policy" || created["type"] != policyURL.expectedType {
			t.Fatalf("unexpected %s policy identity fields: %v", policyURL.name, created)
		}
		properties := created["properties"].(map[string]any)
		if properties["format"] != "rawxml" || !strings.Contains(properties["value"].(string), "<policies>") {
			t.Fatalf("unexpected %s policy properties: %v", policyURL.name, properties)
		}

		listURL := strings.TrimSuffix(strings.Split(policyURL.rawURL, "/policy?")[0], "") + "?api-version=2024-05-01"
		listResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, listURL, nil))
		if err != nil {
			t.Fatalf("list %s policies returned error: %v", policyURL.name, err)
		}
		listed := decodeAPIManagementResponse(t, listResp)
		if values := listed["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "policy" {
			t.Fatalf("expected one %s policy in list, got %v", policyURL.name, listed)
		}

		deleteResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodDelete, policyURL.rawURL, nil))
		if err != nil {
			t.Fatalf("delete %s policy returned error: %v", policyURL.name, err)
		}
		if deleteResp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s policy delete status 200, got %d; body=%s", policyURL.name, deleteResp.StatusCode, string(deleteResp.RawBody))
		}
	}
}

func TestAPIManagementAuxiliaryResourcesLifecycle(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, []byte(`{"properties":{"path":"catalog","serviceUrl":"https://backend.example.test"}}`))); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	productURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter?api-version=2024-05-01"
	createProductResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, productURL, []byte(`{
		"properties":{"displayName":"Starter"}
	}`)))
	if err != nil {
		t.Fatalf("create product returned error: %v", err)
	}
	if createProductResp.StatusCode != http.StatusOK {
		t.Fatalf("expected product create status 200, got %d; body=%s", createProductResp.StatusCode, string(createProductResp.RawBody))
	}
	product := decodeAPIManagementResponse(t, createProductResp)
	if product["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter" {
		t.Fatalf("unexpected product id: %v", product["id"])
	}
	if product["type"] != "Microsoft.ApiManagement/service/products" || product["name"] != "starter" {
		t.Fatalf("unexpected product identity fields: %v", product)
	}
	productProps := product["properties"].(map[string]any)
	if productProps["displayName"] != "Starter" || productProps["subscriptionRequired"] != true || productProps["approvalRequired"] != false || productProps["state"] != "published" {
		t.Fatalf("unexpected product properties: %v", productProps)
	}
	assertAPIManagementCollectionNames(t, svc, "products", "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products?api-version=2024-05-01", []string{"starter"})

	productAPIURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter/apis/catalog-api?api-version=2024-05-01"
	linkResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, productAPIURL, []byte(`{}`)))
	if err != nil {
		t.Fatalf("link product API returned error: %v", err)
	}
	if linkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected product API link status 200, got %d; body=%s", linkResp.StatusCode, string(linkResp.RawBody))
	}
	linkedAPI := decodeAPIManagementResponse(t, linkResp)
	if linkedAPI["name"] != "catalog-api" || linkedAPI["type"] != "Microsoft.ApiManagement/service/apis" {
		t.Fatalf("expected linked API resource, got %v", linkedAPI)
	}
	getLinkedResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, productAPIURL, nil))
	if err != nil {
		t.Fatalf("get product API returned error: %v", err)
	}
	if getLinkedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected product API get status 200, got %d; body=%s", getLinkedResp.StatusCode, string(getLinkedResp.RawBody))
	}
	assertAPIManagementCollectionNames(t, svc, "product apis", "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter/apis?api-version=2024-05-01", []string{"catalog-api"})

	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/subscriptions/starter-sub?api-version=2024-05-01"
	createSubscriptionResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, subscriptionURL, []byte(`{
		"properties":{
			"displayName":"Starter subscription",
			"scope":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter",
			"primaryKey":"primary-key",
			"secondaryKey":"secondary-key"
		}
	}`)))
	if err != nil {
		t.Fatalf("create subscription returned error: %v", err)
	}
	if createSubscriptionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription create status 200, got %d; body=%s", createSubscriptionResp.StatusCode, string(createSubscriptionResp.RawBody))
	}
	subscription := decodeAPIManagementResponse(t, createSubscriptionResp)
	subscriptionProps := subscription["properties"].(map[string]any)
	if subscription["type"] != "Microsoft.ApiManagement/service/subscriptions" || subscriptionProps["state"] != "active" || subscriptionProps["primaryKey"] != "primary-key" || subscriptionProps["secondaryKey"] != "secondary-key" {
		t.Fatalf("unexpected subscription: %v", subscription)
	}
	assertAPIManagementCollectionNames(t, svc, "subscriptions", "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/subscriptions?api-version=2024-05-01", []string{"starter-sub"})

	namedValueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/namedValues/floci-header?api-version=2024-05-01"
	createNamedValueResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, namedValueURL, []byte(`{
		"properties":{"displayName":"floci-header","value":"named-value-applied","secret":false}
	}`)))
	if err != nil {
		t.Fatalf("create named value returned error: %v", err)
	}
	if createNamedValueResp.StatusCode != http.StatusOK {
		t.Fatalf("expected named value create status 200, got %d; body=%s", createNamedValueResp.StatusCode, string(createNamedValueResp.RawBody))
	}
	namedValue := decodeAPIManagementResponse(t, createNamedValueResp)
	namedValueProps := namedValue["properties"].(map[string]any)
	if namedValue["type"] != "Microsoft.ApiManagement/service/namedValues" || namedValueProps["value"] != "named-value-applied" || namedValueProps["secret"] != false {
		t.Fatalf("unexpected named value: %v", namedValue)
	}

	secretNamedValueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/namedValues/floci-secret?api-version=2024-05-01"
	createSecretResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, secretNamedValueURL, []byte(`{
		"properties":{"displayName":"floci-secret","value":"secret-value-applied","secret":true}
	}`)))
	if err != nil {
		t.Fatalf("create secret named value returned error: %v", err)
	}
	if createSecretResp.StatusCode != http.StatusOK {
		t.Fatalf("expected secret named value create status 200, got %d; body=%s", createSecretResp.StatusCode, string(createSecretResp.RawBody))
	}
	secretNamedValue := decodeAPIManagementResponse(t, createSecretResp)
	secretProps := secretNamedValue["properties"].(map[string]any)
	if secretProps["secret"] != true {
		t.Fatalf("expected secret flag on named value, got %v", secretProps)
	}
	if _, ok := secretProps["value"]; ok {
		t.Fatalf("secret named value should not expose properties.value: %v", secretProps)
	}
	assertAPIManagementCollectionNames(t, svc, "named values", "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/namedValues?api-version=2024-05-01", []string{"floci-header", "floci-secret"})
	getSecretResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, secretNamedValueURL, nil))
	if err != nil {
		t.Fatalf("get secret named value returned error: %v", err)
	}
	getSecret := decodeAPIManagementResponse(t, getSecretResp)
	if _, ok := getSecret["properties"].(map[string]any)["value"]; ok {
		t.Fatalf("secret named value GET should not expose properties.value: %v", getSecret)
	}

	backendURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/backends/catalog-backend?api-version=2024-05-01"
	createBackendResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, backendURL, []byte(`{
		"properties":{"url":"http://127.0.0.1:4577"}
	}`)))
	if err != nil {
		t.Fatalf("create backend returned error: %v", err)
	}
	if createBackendResp.StatusCode != http.StatusOK {
		t.Fatalf("expected backend create status 200, got %d; body=%s", createBackendResp.StatusCode, string(createBackendResp.RawBody))
	}
	backend := decodeAPIManagementResponse(t, createBackendResp)
	backendProps := backend["properties"].(map[string]any)
	if backend["type"] != "Microsoft.ApiManagement/service/backends" || backendProps["title"] != "catalog-backend" || backendProps["protocol"] != "http" || backendProps["url"] != "http://127.0.0.1:4577" {
		t.Fatalf("unexpected backend: %v", backend)
	}
	assertAPIManagementCollectionNames(t, svc, "backends", "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/backends?api-version=2024-05-01", []string{"catalog-backend"})

	deleteURLs := []struct {
		name   string
		rawURL string
	}{
		{name: "product api", rawURL: productAPIURL},
		{name: "subscription", rawURL: subscriptionURL},
		{name: "secret named value", rawURL: secretNamedValueURL},
		{name: "named value", rawURL: namedValueURL},
		{name: "backend", rawURL: backendURL},
		{name: "product", rawURL: productURL},
	}
	for _, deleteURL := range deleteURLs {
		resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodDelete, deleteURL.rawURL, nil))
		if err != nil {
			t.Fatalf("delete %s returned error: %v", deleteURL.name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s delete status 200, got %d; body=%s", deleteURL.name, resp.StatusCode, string(resp.RawBody))
		}
		getResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, deleteURL.rawURL, nil))
		if err != nil {
			t.Fatalf("get deleted %s returned error: %v", deleteURL.name, err)
		}
		if getResp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected deleted %s status 404, got %d; body=%s", deleteURL.name, getResp.StatusCode, string(getResp.RawBody))
		}
	}
}

func TestAPIManagementLocalGatewayRouteAndSubscriptionKeys(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	operationURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, []byte(`{"properties":{"path":"catalog"}}`))); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, operationURL, []byte(`{"properties":{"method":"GET","urlTemplate":"/items/{id}"}}`))); err != nil {
		t.Fatalf("create operation returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected operation create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	gatewayURL := "http://localhost:4577/devstoreaccount1-apim/apima/catalog/items/42?trace=true"
	gatewayResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, gatewayURL, nil))
	if err != nil {
		t.Fatalf("gateway request returned error: %v", err)
	}
	if gatewayResp.StatusCode != http.StatusOK {
		t.Fatalf("expected gateway status 200, got %d; body=%s", gatewayResp.StatusCode, string(gatewayResp.RawBody))
	}
	gateway := decodeAPIManagementResponse(t, gatewayResp)
	if gateway["service"] != "apima" || gateway["apiId"] != "catalog-api" || gateway["operationId"] != "get-item" {
		t.Fatalf("unexpected gateway identity fields: %v", gateway)
	}
	if gateway["path"] != "/catalog/items/42" || gateway["backendPath"] != "/items/42" || gateway["method"] != "GET" {
		t.Fatalf("unexpected gateway path fields: %v", gateway)
	}
	if gateway["queryParams"].(map[string]any)["trace"] != "true" {
		t.Fatalf("expected query params to be echoed, got %v", gateway["queryParams"])
	}

	productURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter?api-version=2024-05-01"
	productAPIURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter/apis/catalog-api?api-version=2024-05-01"
	subscriptionURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/subscriptions/starter-sub?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, productURL, []byte(`{"properties":{"displayName":"Starter"}}`))); err != nil {
		t.Fatalf("create product returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected product create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, productAPIURL, []byte(`{}`))); err != nil {
		t.Fatalf("link product API returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected product API link status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, subscriptionURL, []byte(`{
		"properties":{
			"scope":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter",
			"primaryKey":"primary-key",
			"secondaryKey":"secondary-key"
		}
	}`))); err != nil {
		t.Fatalf("create subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	missingKeyResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, gatewayURL, nil))
	if err != nil {
		t.Fatalf("gateway request without key returned error: %v", err)
	}
	if missingKeyResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected missing subscription key status 401, got %d; body=%s", missingKeyResp.StatusCode, string(missingKeyResp.RawBody))
	}

	withHeaderCtx := apiManagementCtx(t, http.MethodGet, gatewayURL, nil)
	withHeaderCtx.RawRequest.Header.Set("Ocp-Apim-Subscription-Key", "primary-key")
	withHeaderResp, err := svc.HandleRequest(withHeaderCtx)
	if err != nil {
		t.Fatalf("gateway request with header key returned error: %v", err)
	}
	if withHeaderResp.StatusCode != http.StatusOK {
		t.Fatalf("expected gateway status with header key 200, got %d; body=%s", withHeaderResp.StatusCode, string(withHeaderResp.RawBody))
	}

	withQueryResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, gatewayURL+"&subscription-key=secondary-key", nil))
	if err != nil {
		t.Fatalf("gateway request with query key returned error: %v", err)
	}
	if withQueryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected gateway status with query key 200, got %d; body=%s", withQueryResp.StatusCode, string(withQueryResp.RawBody))
	}

	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, subscriptionURL, []byte(`{
		"properties":{
			"scope":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/products/starter",
			"state":"inactive",
			"primaryKey":"primary-key",
			"secondaryKey":"secondary-key"
		}
	}`))); err != nil {
		t.Fatalf("deactivate subscription returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected deactivate subscription status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	inactiveCtx := apiManagementCtx(t, http.MethodGet, gatewayURL, nil)
	inactiveCtx.RawRequest.Header.Set("Ocp-Apim-Subscription-Key", "primary-key")
	inactiveResp, err := svc.HandleRequest(inactiveCtx)
	if err != nil {
		t.Fatalf("gateway request with inactive key returned error: %v", err)
	}
	if inactiveResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected inactive subscription status 401, got %d; body=%s", inactiveResp.StatusCode, string(inactiveResp.RawBody))
	}

	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodDelete, productAPIURL, nil)); err != nil {
		t.Fatalf("unlink product API returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected product API unlink status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	afterUnlinkResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, gatewayURL, nil))
	if err != nil {
		t.Fatalf("gateway request after unlink returned error: %v", err)
	}
	if afterUnlinkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected gateway status after unlink 200, got %d; body=%s", afterUnlinkResp.StatusCode, string(afterUnlinkResp.RawBody))
	}
}

func TestAPIManagementLocalGatewayAppliesPolicyEffects(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	operationURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, []byte(`{"properties":{"path":"catalog"}}`))); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, operationURL, []byte(`{"properties":{"method":"GET","urlTemplate":"/items/{id}"}}`))); err != nil {
		t.Fatalf("create operation returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected operation create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	namedValueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/namedValues/floci-header?api-version=2024-05-01"
	secretNamedValueURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/namedValues/floci-secret?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, namedValueURL, []byte(`{"properties":{"value":"named-value-applied","secret":false}}`))); err != nil {
		t.Fatalf("create named value returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected named value create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, secretNamedValueURL, []byte(`{"properties":{"value":"secret-value-applied","secret":true}}`))); err != nil {
		t.Fatalf("create secret named value returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected secret named value create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	policyURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/policies/policy?api-version=2024-05-01"
	policyXML := `<policies>
		<inbound>
			<base />
			<set-header name="X-Floci-NamedValue" exists-action="override"><value>{{floci-header}}</value></set-header>
			<set-header name="X-Floci-SecretNamedValue" exists-action="override"><value>{{floci-secret}}</value></set-header>
			<set-query-parameter name="debug" exists-action="override"><value>{{floci-header}}</value></set-query-parameter>
			<rewrite-uri template="/health/{{floci-header}}" />
		</inbound>
	</policies>`
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, policyURL, apiManagementPolicyPayload(t, policyXML))); err != nil {
		t.Fatalf("create API policy returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected API policy create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	gatewayResp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/catalog/items/42?trace=true", nil))
	if err != nil {
		t.Fatalf("gateway request returned error: %v", err)
	}
	if gatewayResp.StatusCode != http.StatusOK {
		t.Fatalf("expected gateway status 200, got %d; body=%s", gatewayResp.StatusCode, string(gatewayResp.RawBody))
	}
	gateway := decodeAPIManagementResponse(t, gatewayResp)
	if gateway["backendPath"] != "/health/named-value-applied" {
		t.Fatalf("expected policy rewrite to update backend path, got %v", gateway)
	}
	headers := gateway["headers"].(map[string]any)
	if headers["X-Floci-NamedValue"] != "named-value-applied" || headers["X-Floci-SecretNamedValue"] != "secret-value-applied" {
		t.Fatalf("expected named value headers, got %v", headers)
	}
	queryParams := gateway["queryParams"].(map[string]any)
	if queryParams["trace"] != "true" || queryParams["debug"] != "named-value-applied" {
		t.Fatalf("expected policy query params, got %v", queryParams)
	}
}

func TestAPIManagementLocalGatewayReturnResponsePolicy(t *testing.T) {
	svc := New()

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, []byte(`{"properties":{"path":"catalog"}}`))); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	policyURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/policies/policy?api-version=2024-05-01"
	policyXML := `<policies>
		<inbound>
			<return-response>
				<set-status code="429" reason="Too Many Requests" />
				<set-header name="Retry-After" exists-action="override"><value>5</value></set-header>
				<set-body>{"error":"throttled"}</set-body>
			</return-response>
		</inbound>
	</policies>`
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, policyURL, apiManagementPolicyPayload(t, policyXML))); err != nil {
		t.Fatalf("create return-response policy returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected policy create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/catalog/items/42", nil))
	if err != nil {
		t.Fatalf("gateway return-response request returned error: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected return-response status 429, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp.Headers["Retry-After"] != "5" {
		t.Fatalf("expected Retry-After policy header, got %v", resp.Headers)
	}
	if strings.TrimSpace(string(resp.RawBody)) != `{"error":"throttled"}` {
		t.Fatalf("expected policy body, got %s", string(resp.RawBody))
	}
	if resp.RawContentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", resp.RawContentType)
	}
}

func TestAPIManagementLocalGatewayProxiesAPIServiceURL(t *testing.T) {
	svc := New()

	svc.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Path != "/items/42" || r.URL.Query().Get("trace") != "true" {
			t.Fatalf("unexpected backend request: method=%s path=%s rawQuery=%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Backend":    []string{"api-service-url"},
			},
			Body: io.NopCloser(strings.NewReader(`{"proxied":true,"source":"api"}`)),
		}, nil
	})}

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	operationURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	apiBody := []byte(`{"properties":{"path":"catalog","serviceUrl":"https://api-backend.example.test"}}`)
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, apiBody)); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, operationURL, []byte(`{"properties":{"method":"GET","urlTemplate":"/items/{id}"}}`))); err != nil {
		t.Fatalf("create operation returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected operation create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/catalog/items/42?trace=true", nil))
	if err != nil {
		t.Fatalf("gateway proxy request returned error: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected backend status 202, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp.RawContentType != "application/json" || resp.Headers["X-Backend"] != "api-service-url" {
		t.Fatalf("expected backend content type and header, got contentType=%q headers=%v", resp.RawContentType, resp.Headers)
	}
	if strings.TrimSpace(string(resp.RawBody)) != `{"proxied":true,"source":"api"}` {
		t.Fatalf("expected backend response body, got %s", string(resp.RawBody))
	}
}

func TestAPIManagementLocalGatewaySetBackendServicePolicyOverridesAPIServiceURL(t *testing.T) {
	svc := New()

	svc.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api-backend.example.test" {
			t.Fatalf("API serviceUrl backend should not have been called: %s", r.URL.String())
		}
		if r.URL.Path != "/health" || r.Header.Get("X-Floci-Backend") != "catalog" || r.URL.Query().Get("debug") != "true" {
			t.Fatalf("unexpected policy backend request: path=%s headers=%v rawQuery=%s", r.URL.Path, r.Header, r.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Backend":    []string{"policy-backend"},
			},
			Body: io.NopCloser(strings.NewReader(`{"status":"ok","source":"backend-id"}`)),
		}, nil
	})}

	serviceURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima?api-version=2024-05-01"
	apiURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api?api-version=2024-05-01"
	operationURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/operations/get-item?api-version=2024-05-01"
	backendURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/backends/catalog-backend?api-version=2024-05-01"
	policyURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ApiManagement/service/apima/apis/catalog-api/policies/policy?api-version=2024-05-01"
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, serviceURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create APIM service returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected service create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	apiBody := []byte(`{"properties":{"path":"catalog","serviceUrl":"https://api-backend.example.test"}}`)
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, apiURL, apiBody)); err != nil {
		t.Fatalf("create API returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, operationURL, []byte(`{"properties":{"method":"GET","urlTemplate":"/items/{id}"}}`))); err != nil {
		t.Fatalf("create operation returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected operation create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	backendBody := []byte(`{"properties":{"url":"https://policy-backend.example.test"}}`)
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, backendURL, backendBody)); err != nil {
		t.Fatalf("create backend returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected backend create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	policyXML := `<policies>
		<inbound>
			<set-backend-service backend-id="catalog-backend" />
			<rewrite-uri template="/health" />
			<set-header name="X-Floci-Backend" exists-action="override"><value>catalog</value></set-header>
			<set-query-parameter name="debug" exists-action="override"><value>true</value></set-query-parameter>
		</inbound>
	</policies>`
	if resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodPut, policyURL, apiManagementPolicyPayload(t, policyXML))); err != nil {
		t.Fatalf("create API policy returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected policy create status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-apim/apima/catalog/items/42", nil))
	if err != nil {
		t.Fatalf("gateway policy backend request returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected backend status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp.RawContentType != "application/json" || resp.Headers["X-Backend"] != "policy-backend" {
		t.Fatalf("expected policy backend content type and header, got contentType=%q headers=%v", resp.RawContentType, resp.Headers)
	}
	if strings.TrimSpace(string(resp.RawBody)) != `{"status":"ok","source":"backend-id"}` {
		t.Fatalf("expected policy backend response body, got %s", string(resp.RawBody))
	}
}

func apiManagementCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func assertAPIManagementCollectionNames(t *testing.T, svc *APIManagementService, label, rawURL string, expected []string) {
	t.Helper()
	resp, err := svc.HandleRequest(apiManagementCtx(t, http.MethodGet, rawURL, nil))
	if err != nil {
		t.Fatalf("list %s returned error: %v", label, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected list %s status 200, got %d; body=%s", label, resp.StatusCode, string(resp.RawBody))
	}
	listed := decodeAPIManagementResponse(t, resp)
	values := listed["value"].([]any)
	if len(values) != len(expected) {
		t.Fatalf("expected %d %s, got %v", len(expected), label, listed)
	}
	for i, expectedName := range expected {
		if values[i].(map[string]any)["name"] != expectedName {
			t.Fatalf("expected %s[%d] to be %q, got %v", label, i, expectedName, values[i])
		}
	}
}

func apiManagementPolicyPayload(t *testing.T, xml string) []byte {
	t.Helper()
	data, err := gojson.Marshal(map[string]any{
		"properties": map[string]any{
			"format": "rawxml",
			"value":  xml,
		},
	})
	if err != nil {
		t.Fatalf("marshal policy payload: %v", err)
	}
	return data
}

func apiManagementOpenAPIImportPayload(t *testing.T, title string, paths map[string]any) []byte {
	t.Helper()
	openAPI, err := gojson.Marshal(map[string]any{
		"openapi": "3.0.1",
		"info": map[string]any{
			"title":   title,
			"version": "1.0",
		},
		"paths": paths,
	})
	if err != nil {
		t.Fatalf("marshal OpenAPI document: %v", err)
	}
	data, err := gojson.Marshal(map[string]any{
		"properties": map[string]any{
			"displayName": title,
			"path":        "openapi",
			"protocols":   []string{"https"},
			"format":      "openapi+json",
			"value":       string(openAPI),
		},
	})
	if err != nil {
		t.Fatalf("marshal OpenAPI import payload: %v", err)
	}
	return data
}

func decodeAPIManagementResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
