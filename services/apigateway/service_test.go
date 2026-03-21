package apigateway_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	apigwsvc "github.com/neureaux/cloudmock/services/apigateway"
)

// newAPIGatewayHandler builds a full gateway stack with API Gateway registered and IAM disabled.
func newAPIGatewayHandler(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(apigwsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// apigwReq builds an HTTP request targeting the API Gateway service.
func apigwReq(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("apigwReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// Authorization header places "apigateway" as the service in the credential scope.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/apigateway/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// mustCreateAPI creates a REST API and returns its ID.
func mustCreateAPI(t *testing.T, handler http.Handler, name string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost, "/restapis", map[string]string{
		"name":        name,
		"description": "test api",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateRestApi %q: expected 201, got %d\nbody: %s", name, w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatalf("CreateRestApi: id is empty\nbody: %s", w.Body.String())
	}
	return id
}

// rootResourceID returns the root resource ID for an API.
func rootResourceID(t *testing.T, handler http.Handler, apiID string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/"+apiID+"/resources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetResources: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	items, _ := m["items"].([]interface{})
	for _, item := range items {
		r := item.(map[string]interface{})
		if r["path"].(string) == "/" {
			return r["id"].(string)
		}
	}
	t.Fatalf("rootResourceID: root resource not found\nbody: %s", w.Body.String())
	return ""
}

// ---- Test 1: CreateRestApi + GetRestApis ----

func TestAPIGateway_CreateRestApiAndGetRestApis(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	// Create two APIs.
	id1 := mustCreateAPI(t, handler, "my-api")
	id2 := mustCreateAPI(t, handler, "another-api")

	if id1 == "" || id2 == "" {
		t.Fatal("API IDs must not be empty")
	}
	if id1 == id2 {
		t.Fatal("API IDs must be unique")
	}

	// GetRestApis should list both.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRestApis: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	items, ok := m["items"].([]interface{})
	if !ok {
		t.Fatalf("GetRestApis: missing items\nbody: %s", w.Body.String())
	}
	if len(items) < 2 {
		t.Errorf("GetRestApis: expected at least 2 APIs, got %d", len(items))
	}

	// Verify both IDs appear.
	found := make(map[string]bool)
	for _, item := range items {
		api := item.(map[string]interface{})
		found[api["id"].(string)] = true
	}
	for _, id := range []string{id1, id2} {
		if !found[id] {
			t.Errorf("GetRestApis: ID %q not found in list", id)
		}
	}

	// GetRestApi for a single API.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet, "/restapis/"+id1, nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("GetRestApi: expected 200, got %d\nbody: %s", wGet.Code, wGet.Body.String())
	}
	mGet := decodeJSON(t, wGet.Body.String())
	if mGet["id"].(string) != id1 {
		t.Errorf("GetRestApi: expected id=%s, got %s", id1, mGet["id"])
	}
	if mGet["name"].(string) != "my-api" {
		t.Errorf("GetRestApi: expected name=my-api, got %s", mGet["name"])
	}
}

// ---- Test 2: CreateResource + GetResources (verify path computation) ----

func TestAPIGateway_CreateResourceAndGetResources(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "resource-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create a child resource under root: /pets
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "pets"}))
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("CreateResource /pets: expected 201, got %d\nbody: %s", wCreate.Code, wCreate.Body.String())
	}
	mCreate := decodeJSON(t, wCreate.Body.String())
	petsID, _ := mCreate["id"].(string)
	if petsID == "" {
		t.Fatalf("CreateResource: id is empty")
	}
	petsPath, _ := mCreate["path"].(string)
	if petsPath != "/pets" {
		t.Errorf("CreateResource /pets: expected path=/pets, got %q", petsPath)
	}
	if mCreate["parentId"].(string) != rootID {
		t.Errorf("CreateResource /pets: expected parentId=%s, got %s", rootID, mCreate["parentId"])
	}

	// Create a nested resource under /pets: /pets/{petId}
	wNested := httptest.NewRecorder()
	handler.ServeHTTP(wNested, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, petsID),
		map[string]string{"pathPart": "{petId}"}))
	if wNested.Code != http.StatusCreated {
		t.Fatalf("CreateResource /pets/{petId}: expected 201, got %d\nbody: %s", wNested.Code, wNested.Body.String())
	}
	mNested := decodeJSON(t, wNested.Body.String())
	nestedPath, _ := mNested["path"].(string)
	if nestedPath != "/pets/{petId}" {
		t.Errorf("CreateResource /pets/{petId}: expected path=/pets/{petId}, got %q", nestedPath)
	}

	// GetResources should show root + /pets + /pets/{petId}
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, apigwReq(t, http.MethodGet, "/restapis/"+apiID+"/resources", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("GetResources: expected 200, got %d\nbody: %s", wList.Code, wList.Body.String())
	}
	mList := decodeJSON(t, wList.Body.String())
	items, _ := mList["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("GetResources: expected 3 resources (root + /pets + /pets/{petId}), got %d", len(items))
	}

	paths := make(map[string]bool)
	for _, item := range items {
		r := item.(map[string]interface{})
		paths[r["path"].(string)] = true
	}
	for _, want := range []string{"/", "/pets", "/pets/{petId}"} {
		if !paths[want] {
			t.Errorf("GetResources: path %q not found", want)
		}
	}
}

// ---- Test 3: PutMethod + GetMethod ----

func TestAPIGateway_PutMethodAndGetMethod(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "method-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create /items resource.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "items"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource /items: expected 201, got %d", wRes.Code)
	}
	mRes := decodeJSON(t, wRes.Body.String())
	itemsID, _ := mRes["id"].(string)

	// PutMethod: GET on /items
	wPut := httptest.NewRecorder()
	handler.ServeHTTP(wPut, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, itemsID),
		map[string]string{"authorizationType": "NONE"}))
	if wPut.Code != http.StatusOK {
		t.Fatalf("PutMethod GET: expected 200, got %d\nbody: %s", wPut.Code, wPut.Body.String())
	}
	mPut := decodeJSON(t, wPut.Body.String())
	if mPut["httpMethod"].(string) != "GET" {
		t.Errorf("PutMethod: expected httpMethod=GET, got %s", mPut["httpMethod"])
	}
	if mPut["authorizationType"].(string) != "NONE" {
		t.Errorf("PutMethod: expected authorizationType=NONE, got %s", mPut["authorizationType"])
	}

	// GetMethod.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, itemsID), nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("GetMethod: expected 200, got %d\nbody: %s", wGet.Code, wGet.Body.String())
	}
	mGet := decodeJSON(t, wGet.Body.String())
	if mGet["httpMethod"].(string) != "GET" {
		t.Errorf("GetMethod: expected httpMethod=GET, got %s", mGet["httpMethod"])
	}

	// GetMethod for non-existent method should return 404.
	wMissing := httptest.NewRecorder()
	handler.ServeHTTP(wMissing, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/DELETE", apiID, itemsID), nil))
	if wMissing.Code != http.StatusNotFound {
		t.Errorf("GetMethod (not found): expected 404, got %d", wMissing.Code)
	}
}

// ---- Test 4: PutIntegration ----

func TestAPIGateway_PutIntegration(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "integration-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create /proxy resource.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "proxy"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	mRes := decodeJSON(t, wRes.Body.String())
	proxyID, _ := mRes["id"].(string)

	// PutMethod first.
	wMethod := httptest.NewRecorder()
	handler.ServeHTTP(wMethod, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST", apiID, proxyID),
		map[string]string{"authorizationType": "NONE"}))
	if wMethod.Code != http.StatusOK {
		t.Fatalf("PutMethod: expected 200, got %d", wMethod.Code)
	}

	// PutIntegration: AWS_PROXY.
	wInt := httptest.NewRecorder()
	handler.ServeHTTP(wInt, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST/integration", apiID, proxyID),
		map[string]string{
			"type":                  "AWS_PROXY",
			"uri":                   "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:000000000000:function:my-fn/invocations",
			"httpMethod":            "POST",
			"integrationHttpMethod": "POST",
		}))
	if wInt.Code != http.StatusOK {
		t.Fatalf("PutIntegration: expected 200, got %d\nbody: %s", wInt.Code, wInt.Body.String())
	}
	mInt := decodeJSON(t, wInt.Body.String())
	if mInt["type"].(string) != "AWS_PROXY" {
		t.Errorf("PutIntegration: expected type=AWS_PROXY, got %s", mInt["type"])
	}
	if mInt["httpMethod"].(string) != "POST" {
		t.Errorf("PutIntegration: expected httpMethod=POST, got %s", mInt["httpMethod"])
	}

	// GetMethod should now show the integration.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST", apiID, proxyID), nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("GetMethod after integration: expected 200, got %d", wGet.Code)
	}
	mGet := decodeJSON(t, wGet.Body.String())
	integration, ok := mGet["methodIntegration"].(map[string]interface{})
	if !ok {
		t.Fatalf("GetMethod: missing methodIntegration\nbody: %s", wGet.Body.String())
	}
	if integration["type"].(string) != "AWS_PROXY" {
		t.Errorf("GetMethod integration: expected type=AWS_PROXY, got %s", integration["type"])
	}
}

// ---- Test 5: CreateDeployment + CreateStage + GetStages ----

func TestAPIGateway_DeploymentAndStage(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "deploy-api")

	// CreateDeployment.
	wDeploy := httptest.NewRecorder()
	handler.ServeHTTP(wDeploy, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{
			"stageName":   "prod",
			"description": "initial deployment",
		}))
	if wDeploy.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment: expected 201, got %d\nbody: %s", wDeploy.Code, wDeploy.Body.String())
	}
	mDeploy := decodeJSON(t, wDeploy.Body.String())
	deployID, _ := mDeploy["id"].(string)
	if deployID == "" {
		t.Fatal("CreateDeployment: id is empty")
	}
	if _, ok := mDeploy["createdDate"]; !ok {
		t.Error("CreateDeployment: missing createdDate")
	}

	// GetDeployments.
	wGetDeploys := httptest.NewRecorder()
	handler.ServeHTTP(wGetDeploys, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/deployments", apiID), nil))
	if wGetDeploys.Code != http.StatusOK {
		t.Fatalf("GetDeployments: expected 200, got %d\nbody: %s", wGetDeploys.Code, wGetDeploys.Body.String())
	}
	mDeployList := decodeJSON(t, wGetDeploys.Body.String())
	deployItems, _ := mDeployList["items"].([]interface{})
	if len(deployItems) != 1 {
		t.Errorf("GetDeployments: expected 1 deployment, got %d", len(deployItems))
	}

	// CreateStage.
	wStage := httptest.NewRecorder()
	handler.ServeHTTP(wStage, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/stages", apiID),
		map[string]string{
			"stageName":    "prod",
			"deploymentId": deployID,
			"description":  "production stage",
		}))
	if wStage.Code != http.StatusCreated {
		t.Fatalf("CreateStage: expected 201, got %d\nbody: %s", wStage.Code, wStage.Body.String())
	}
	mStage := decodeJSON(t, wStage.Body.String())
	if mStage["stageName"].(string) != "prod" {
		t.Errorf("CreateStage: expected stageName=prod, got %s", mStage["stageName"])
	}
	if mStage["deploymentId"].(string) != deployID {
		t.Errorf("CreateStage: expected deploymentId=%s, got %s", deployID, mStage["deploymentId"])
	}

	// GetStages.
	wGetStages := httptest.NewRecorder()
	handler.ServeHTTP(wGetStages, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/stages", apiID), nil))
	if wGetStages.Code != http.StatusOK {
		t.Fatalf("GetStages: expected 200, got %d\nbody: %s", wGetStages.Code, wGetStages.Body.String())
	}
	mStageList := decodeJSON(t, wGetStages.Body.String())
	stageItems, _ := mStageList["item"].([]interface{})
	if len(stageItems) != 1 {
		t.Errorf("GetStages: expected 1 stage, got %d", len(stageItems))
	}
	firstStage, _ := stageItems[0].(map[string]interface{})
	if firstStage["stageName"].(string) != "prod" {
		t.Errorf("GetStages: expected stageName=prod, got %s", firstStage["stageName"])
	}
}

// ---- Test 6: DeleteRestApi ----

func TestAPIGateway_DeleteRestApi(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "to-delete-api")

	// Verify it appears in list.
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, apigwReq(t, http.MethodGet, "/restapis", nil))
	m := decodeJSON(t, wList.Body.String())
	items, _ := m["items"].([]interface{})
	found := false
	for _, item := range items {
		if item.(map[string]interface{})["id"].(string) == apiID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("DeleteRestApi: API not found in list before deletion")
	}

	// Delete it.
	wDel := httptest.NewRecorder()
	handler.ServeHTTP(wDel, apigwReq(t, http.MethodDelete, "/restapis/"+apiID, nil))
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("DeleteRestApi: expected 204, got %d\nbody: %s", wDel.Code, wDel.Body.String())
	}

	// GetRestApi should return 404.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet, "/restapis/"+apiID, nil))
	if wGet.Code != http.StatusNotFound {
		t.Errorf("GetRestApi after delete: expected 404, got %d", wGet.Code)
	}

	// Delete again — should return 404.
	wDel2 := httptest.NewRecorder()
	handler.ServeHTTP(wDel2, apigwReq(t, http.MethodDelete, "/restapis/"+apiID, nil))
	if wDel2.Code != http.StatusNotFound {
		t.Errorf("DeleteRestApi (already deleted): expected 404, got %d", wDel2.Code)
	}

	// Verify it's gone from the list.
	wList2 := httptest.NewRecorder()
	handler.ServeHTTP(wList2, apigwReq(t, http.MethodGet, "/restapis", nil))
	m2 := decodeJSON(t, wList2.Body.String())
	items2, _ := m2["items"].([]interface{})
	for _, item := range items2 {
		if item.(map[string]interface{})["id"].(string) == apiID {
			t.Errorf("GetRestApis: API %s should not appear after deletion", apiID)
		}
	}
}
