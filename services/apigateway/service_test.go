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
func apigwReq(t *testing.T, method, path string, body any) *http.Request {
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
func decodeJSON(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
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
	items, _ := m["items"].([]any)
	for _, item := range items {
		r := item.(map[string]any)
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
	items, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("GetRestApis: missing items\nbody: %s", w.Body.String())
	}
	if len(items) < 2 {
		t.Errorf("GetRestApis: expected at least 2 APIs, got %d", len(items))
	}

	// Verify both IDs appear.
	found := make(map[string]bool)
	for _, item := range items {
		api := item.(map[string]any)
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
	items, _ := mList["items"].([]any)
	if len(items) != 3 {
		t.Errorf("GetResources: expected 3 resources (root + /pets + /pets/{petId}), got %d", len(items))
	}

	paths := make(map[string]bool)
	for _, item := range items {
		r := item.(map[string]any)
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
	integration, ok := mGet["methodIntegration"].(map[string]any)
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
	deployItems, _ := mDeployList["items"].([]any)
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
	stageItems, _ := mStageList["item"].([]any)
	if len(stageItems) != 1 {
		t.Errorf("GetStages: expected 1 stage, got %d", len(stageItems))
	}
	firstStage, _ := stageItems[0].(map[string]any)
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
	items, _ := m["items"].([]any)
	found := false
	for _, item := range items {
		if item.(map[string]any)["id"].(string) == apiID {
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
	items2, _ := m2["items"].([]any)
	for _, item := range items2 {
		if item.(map[string]any)["id"].(string) == apiID {
			t.Errorf("GetRestApis: API %s should not appear after deletion", apiID)
		}
	}
}

// ---- Test 7: GetRestApi (single API with description and createdDate) ----

func TestAPIGateway_GetRestApi(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "single-api")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/"+apiID, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRestApi: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	if m["id"].(string) != apiID {
		t.Errorf("GetRestApi: expected id=%s, got %s", apiID, m["id"])
	}
	if m["name"].(string) != "single-api" {
		t.Errorf("GetRestApi: expected name=single-api, got %s", m["name"])
	}
	if m["description"].(string) != "test api" {
		t.Errorf("GetRestApi: expected description='test api', got %s", m["description"])
	}
	if _, ok := m["createdDate"]; !ok {
		t.Error("GetRestApi: missing createdDate field")
	}
}

// ---- Test 8: GetRestApi NotFoundException ----

func TestAPIGateway_GetRestApi_NotFound(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/nonexistent-id", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GetRestApi (not found): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 9: CreateRestApi BadRequestException (missing name) ----

func TestAPIGateway_CreateRestApi_MissingName(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost, "/restapis", map[string]string{
		"description": "no name provided",
	}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateRestApi (missing name): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 10: CreateRestApi BadRequestException (invalid JSON) ----

func TestAPIGateway_CreateRestApi_InvalidJSON(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/restapis", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/apigateway/aws4_request, SignedHeaders=host, Signature=abc123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateRestApi (invalid JSON): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 11: ListRestApis returns empty when no APIs exist ----

func TestAPIGateway_ListRestApis_Empty(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRestApis (empty): expected 200, got %d", w.Code)
	}

	m := decodeJSON(t, w.Body.String())
	items, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("ListRestApis: missing items array")
	}
	if len(items) != 0 {
		t.Errorf("ListRestApis (empty): expected 0 items, got %d", len(items))
	}
}

// ---- Test 12: GetResource (single resource via GetResources) ----

func TestAPIGateway_GetResource(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "resource-get-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create a child resource.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "users"}))
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wCreate.Code)
	}
	mCreate := decodeJSON(t, wCreate.Body.String())
	usersID := mCreate["id"].(string)

	// GetResources and find the created resource.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/"+apiID+"/resources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetResources: expected 200, got %d", w.Code)
	}
	m := decodeJSON(t, w.Body.String())
	items, _ := m["items"].([]any)

	found := false
	for _, item := range items {
		r := item.(map[string]any)
		if r["id"].(string) == usersID {
			found = true
			if r["path"].(string) != "/users" {
				t.Errorf("GetResource: expected path=/users, got %s", r["path"])
			}
			if r["pathPart"].(string) != "users" {
				t.Errorf("GetResource: expected pathPart=users, got %s", r["pathPart"])
			}
			if r["parentId"].(string) != rootID {
				t.Errorf("GetResource: expected parentId=%s, got %s", rootID, r["parentId"])
			}
		}
	}
	if !found {
		t.Errorf("GetResource: resource %s not found in list", usersID)
	}
}

// ---- Test 13: CreateResource NotFoundException (invalid API) ----

func TestAPIGateway_CreateResource_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		"/restapis/nonexistent/resources/someparent",
		map[string]string{"pathPart": "test"}))
	if w.Code != http.StatusNotFound {
		t.Errorf("CreateResource (invalid API): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 14: CreateResource NotFoundException (invalid parent) ----

func TestAPIGateway_CreateResource_InvalidParent(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "bad-parent-api")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/nonexistent", apiID),
		map[string]string{"pathPart": "test"}))
	if w.Code != http.StatusNotFound {
		t.Errorf("CreateResource (invalid parent): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 15: CreateResource BadRequestException (missing pathPart) ----

func TestAPIGateway_CreateResource_MissingPathPart(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "missing-pathpart-api")
	rootID := rootResourceID(t, handler, apiID)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateResource (missing pathPart): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 16: DeleteResource ----

func TestAPIGateway_DeleteResource(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "delete-resource-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create a resource.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "to-delete"}))
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wCreate.Code)
	}
	mCreate := decodeJSON(t, wCreate.Body.String())
	resID := mCreate["id"].(string)

	// Delete the resource.
	wDel := httptest.NewRecorder()
	handler.ServeHTTP(wDel, apigwReq(t, http.MethodDelete,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, resID), nil))
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("DeleteResource: expected 204, got %d\nbody: %s", wDel.Code, wDel.Body.String())
	}

	// GetResources should only have root.
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, apigwReq(t, http.MethodGet, "/restapis/"+apiID+"/resources", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("GetResources: expected 200, got %d", wList.Code)
	}
	m := decodeJSON(t, wList.Body.String())
	items, _ := m["items"].([]any)
	if len(items) != 1 {
		t.Errorf("GetResources after delete: expected 1 (root only), got %d", len(items))
	}

	// Delete again — should return 404.
	wDel2 := httptest.NewRecorder()
	handler.ServeHTTP(wDel2, apigwReq(t, http.MethodDelete,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, resID), nil))
	if wDel2.Code != http.StatusNotFound {
		t.Errorf("DeleteResource (already deleted): expected 404, got %d", wDel2.Code)
	}
}

// ---- Test 17: DeleteResource NotFoundException (invalid API) ----

func TestAPIGateway_DeleteResource_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodDelete,
		"/restapis/nonexistent/resources/someid", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("DeleteResource (invalid API): expected 404, got %d", w.Code)
	}
}

// ---- Test 18: GetResources NotFoundException (invalid API) ----

func TestAPIGateway_GetResources_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/nonexistent/resources", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GetResources (invalid API): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 19: PutMethod on non-existent API ----

func TestAPIGateway_PutMethod_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPut,
		"/restapis/nonexistent/resources/someres/methods/GET",
		map[string]string{"authorizationType": "NONE"}))
	if w.Code != http.StatusNotFound {
		t.Errorf("PutMethod (invalid API): expected 404, got %d", w.Code)
	}
}

// ---- Test 20: PutMethod on non-existent resource ----

func TestAPIGateway_PutMethod_InvalidResource(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "method-bad-res-api")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/nonexistent/methods/GET", apiID),
		map[string]string{"authorizationType": "NONE"}))
	if w.Code != http.StatusNotFound {
		t.Errorf("PutMethod (invalid resource): expected 404, got %d", w.Code)
	}
}

// ---- Test 21: PutMethod multiple HTTP methods on same resource ----

func TestAPIGateway_PutMethod_MultipleMethods(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "multi-method-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create a resource.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "orders"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	ordersID := decodeJSON(t, wRes.Body.String())["id"].(string)

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, m := range methods {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, apigwReq(t, http.MethodPut,
			fmt.Sprintf("/restapis/%s/resources/%s/methods/%s", apiID, ordersID, m),
			map[string]string{"authorizationType": "NONE"}))
		if w.Code != http.StatusOK {
			t.Errorf("PutMethod %s: expected 200, got %d", m, w.Code)
		}
		mResp := decodeJSON(t, w.Body.String())
		if mResp["httpMethod"].(string) != m {
			t.Errorf("PutMethod %s: expected httpMethod=%s, got %s", m, m, mResp["httpMethod"])
		}
	}

	// Verify all methods can be retrieved.
	for _, m := range methods {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, apigwReq(t, http.MethodGet,
			fmt.Sprintf("/restapis/%s/resources/%s/methods/%s", apiID, ordersID, m), nil))
		if w.Code != http.StatusOK {
			t.Errorf("GetMethod %s: expected 200, got %d", m, w.Code)
		}
	}
}

// ---- Test 22: GetMethod NotFoundException (invalid API) ----

func TestAPIGateway_GetMethod_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet,
		"/restapis/nonexistent/resources/someres/methods/GET", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GetMethod (invalid API): expected 404, got %d", w.Code)
	}
}

// ---- Test 23: PutIntegration NotFoundException (no method) ----

func TestAPIGateway_PutIntegration_NoMethod(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "int-no-method-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create a resource but do NOT put a method.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "nope"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	resID := decodeJSON(t, wRes.Body.String())["id"].(string)

	// PutIntegration without a method should return 404.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET/integration", apiID, resID),
		map[string]string{"type": "HTTP", "uri": "http://example.com", "httpMethod": "GET"}))
	if w.Code != http.StatusNotFound {
		t.Errorf("PutIntegration (no method): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 24: PutIntegration BadRequestException (missing type) ----

func TestAPIGateway_PutIntegration_MissingType(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "int-no-type-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create resource + method.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "data"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	resID := decodeJSON(t, wRes.Body.String())["id"].(string)

	wMethod := httptest.NewRecorder()
	handler.ServeHTTP(wMethod, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, resID),
		map[string]string{"authorizationType": "NONE"}))
	if wMethod.Code != http.StatusOK {
		t.Fatalf("PutMethod: expected 200, got %d", wMethod.Code)
	}

	// PutIntegration with missing type should return 400.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET/integration", apiID, resID),
		map[string]string{"uri": "http://example.com"}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("PutIntegration (missing type): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 25: PutIntegration with HTTP type ----

func TestAPIGateway_PutIntegration_HTTP(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "int-http-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create resource + method.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "http-target"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	resID := decodeJSON(t, wRes.Body.String())["id"].(string)

	wMethod := httptest.NewRecorder()
	handler.ServeHTTP(wMethod, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, resID),
		map[string]string{"authorizationType": "NONE"}))
	if wMethod.Code != http.StatusOK {
		t.Fatalf("PutMethod: expected 200, got %d", wMethod.Code)
	}

	// PutIntegration with HTTP type.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET/integration", apiID, resID),
		map[string]string{
			"type":       "HTTP",
			"uri":        "http://backend.example.com/api",
			"httpMethod": "GET",
		}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutIntegration (HTTP): expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	if m["type"].(string) != "HTTP" {
		t.Errorf("PutIntegration: expected type=HTTP, got %s", m["type"])
	}
	if m["uri"].(string) != "http://backend.example.com/api" {
		t.Errorf("PutIntegration: expected uri=http://backend.example.com/api, got %s", m["uri"])
	}
}

// ---- Test 26: GetIntegration via GetMethod ----

func TestAPIGateway_GetIntegration(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "get-int-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create resource + method + integration.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "funcs"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	resID := decodeJSON(t, wRes.Body.String())["id"].(string)

	wMethod := httptest.NewRecorder()
	handler.ServeHTTP(wMethod, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST", apiID, resID),
		map[string]string{"authorizationType": "NONE"}))
	if wMethod.Code != http.StatusOK {
		t.Fatalf("PutMethod: expected 200, got %d", wMethod.Code)
	}

	lambdaURI := "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:000000000000:function:handler/invocations"
	wInt := httptest.NewRecorder()
	handler.ServeHTTP(wInt, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST/integration", apiID, resID),
		map[string]string{
			"type":       "AWS_PROXY",
			"uri":        lambdaURI,
			"httpMethod": "POST",
		}))
	if wInt.Code != http.StatusOK {
		t.Fatalf("PutIntegration: expected 200, got %d", wInt.Code)
	}

	// GetMethod should return the integration nested in methodIntegration.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST", apiID, resID), nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("GetMethod: expected 200, got %d", wGet.Code)
	}
	m := decodeJSON(t, wGet.Body.String())
	integration, ok := m["methodIntegration"].(map[string]any)
	if !ok {
		t.Fatalf("GetMethod: missing methodIntegration")
	}
	if integration["type"].(string) != "AWS_PROXY" {
		t.Errorf("GetIntegration: expected type=AWS_PROXY, got %s", integration["type"])
	}
	if integration["uri"].(string) != lambdaURI {
		t.Errorf("GetIntegration: expected uri=%s, got %s", lambdaURI, integration["uri"])
	}
}

// ---- Test 27: CreateDeployment NotFoundException (invalid API) ----

func TestAPIGateway_CreateDeployment_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		"/restapis/nonexistent/deployments",
		map[string]string{"description": "deploy"}))
	if w.Code != http.StatusNotFound {
		t.Errorf("CreateDeployment (invalid API): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 28: GetDeployments NotFoundException (invalid API) ----

func TestAPIGateway_GetDeployments_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/nonexistent/deployments", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GetDeployments (invalid API): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 29: Multiple deployments for same API ----

func TestAPIGateway_MultipleDeployments(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "multi-deploy-api")

	// Create two deployments.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "deploy-1"}))
	if w1.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment 1: expected 201, got %d", w1.Code)
	}
	d1 := decodeJSON(t, w1.Body.String())
	id1, _ := d1["id"].(string)

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "deploy-2"}))
	if w2.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment 2: expected 201, got %d", w2.Code)
	}
	d2 := decodeJSON(t, w2.Body.String())
	id2, _ := d2["id"].(string)

	if id1 == id2 {
		t.Error("Deployment IDs must be unique")
	}

	// GetDeployments should list both.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/deployments", apiID), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetDeployments: expected 200, got %d", w.Code)
	}
	m := decodeJSON(t, w.Body.String())
	items, _ := m["items"].([]any)
	if len(items) != 2 {
		t.Errorf("GetDeployments: expected 2, got %d", len(items))
	}

	found := make(map[string]bool)
	for _, item := range items {
		dep := item.(map[string]any)
		found[dep["id"].(string)] = true
		if _, ok := dep["createdDate"]; !ok {
			t.Error("Deployment missing createdDate")
		}
	}
	if !found[id1] || !found[id2] {
		t.Errorf("GetDeployments: missing one of the deployment IDs")
	}
}

// ---- Test 30: CreateStage BadRequestException (missing stageName) ----

func TestAPIGateway_CreateStage_MissingStageName(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "stage-no-name-api")

	// Create a deployment first.
	wDep := httptest.NewRecorder()
	handler.ServeHTTP(wDep, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "dep"}))
	if wDep.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment: expected 201, got %d", wDep.Code)
	}
	depID := decodeJSON(t, wDep.Body.String())["id"].(string)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/stages", apiID),
		map[string]string{"deploymentId": depID}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateStage (missing stageName): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 31: CreateStage BadRequestException (missing deploymentId) ----

func TestAPIGateway_CreateStage_MissingDeploymentId(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "stage-no-dep-api")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/stages", apiID),
		map[string]string{"stageName": "prod"}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateStage (missing deploymentId): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 32: CreateStage NotFoundException (invalid deployment) ----

func TestAPIGateway_CreateStage_InvalidDeployment(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "stage-bad-dep-api")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/stages", apiID),
		map[string]string{
			"stageName":    "prod",
			"deploymentId": "nonexistent-deploy-id",
		}))
	if w.Code != http.StatusNotFound {
		t.Errorf("CreateStage (invalid deployment): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 33: CreateStage NotFoundException (invalid API) ----

func TestAPIGateway_CreateStage_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		"/restapis/nonexistent/stages",
		map[string]string{
			"stageName":    "prod",
			"deploymentId": "some-deploy-id",
		}))
	if w.Code != http.StatusNotFound {
		t.Errorf("CreateStage (invalid API): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 34: GetStages NotFoundException (invalid API) ----

func TestAPIGateway_GetStages_InvalidAPI(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/nonexistent/stages", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GetStages (invalid API): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 35: Multiple stages for same API ----

func TestAPIGateway_MultipleStages(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "multi-stage-api")

	// Create a deployment.
	wDep := httptest.NewRecorder()
	handler.ServeHTTP(wDep, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "dep"}))
	if wDep.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment: expected 201, got %d", wDep.Code)
	}
	depID := decodeJSON(t, wDep.Body.String())["id"].(string)

	// Create two stages.
	for _, stage := range []string{"dev", "prod"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
			fmt.Sprintf("/restapis/%s/stages", apiID),
			map[string]string{
				"stageName":    stage,
				"deploymentId": depID,
				"description":  stage + " stage",
			}))
		if w.Code != http.StatusCreated {
			t.Fatalf("CreateStage %s: expected 201, got %d\nbody: %s", stage, w.Code, w.Body.String())
		}
	}

	// GetStages should return both.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/stages", apiID), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetStages: expected 200, got %d", w.Code)
	}
	m := decodeJSON(t, w.Body.String())
	stageItems, _ := m["item"].([]any)
	if len(stageItems) != 2 {
		t.Errorf("GetStages: expected 2 stages, got %d", len(stageItems))
	}

	names := make(map[string]bool)
	for _, item := range stageItems {
		st := item.(map[string]any)
		names[st["stageName"].(string)] = true
		if st["deploymentId"].(string) != depID {
			t.Errorf("Stage: expected deploymentId=%s, got %s", depID, st["deploymentId"])
		}
	}
	if !names["dev"] || !names["prod"] {
		t.Errorf("GetStages: expected dev and prod, got %v", names)
	}
}

// ---- Test 36: Full lifecycle (create API, resource, method, integration, deploy, stage, delete) ----

func TestAPIGateway_FullLifecycle(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	// Step 1: Create API.
	apiID := mustCreateAPI(t, handler, "lifecycle-api")

	// Step 2: Create resource /v1.
	rootID := rootResourceID(t, handler, apiID)
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "v1"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource /v1: expected 201, got %d", wRes.Code)
	}
	v1ID := decodeJSON(t, wRes.Body.String())["id"].(string)

	// Step 3: Create nested resource /v1/health.
	wNested := httptest.NewRecorder()
	handler.ServeHTTP(wNested, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, v1ID),
		map[string]string{"pathPart": "health"}))
	if wNested.Code != http.StatusCreated {
		t.Fatalf("CreateResource /v1/health: expected 201, got %d", wNested.Code)
	}
	healthID := decodeJSON(t, wNested.Body.String())["id"].(string)
	healthPath := decodeJSON(t, wNested.Body.String())["path"].(string)
	if healthPath != "/v1/health" {
		t.Errorf("CreateResource: expected path=/v1/health, got %s", healthPath)
	}

	// Step 4: Put GET method on /v1/health.
	wMethod := httptest.NewRecorder()
	handler.ServeHTTP(wMethod, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, healthID),
		map[string]string{"authorizationType": "NONE"}))
	if wMethod.Code != http.StatusOK {
		t.Fatalf("PutMethod: expected 200, got %d", wMethod.Code)
	}

	// Step 5: Put integration on the method.
	wInt := httptest.NewRecorder()
	handler.ServeHTTP(wInt, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET/integration", apiID, healthID),
		map[string]string{
			"type":       "MOCK",
			"httpMethod": "GET",
		}))
	if wInt.Code != http.StatusOK {
		t.Fatalf("PutIntegration: expected 200, got %d", wInt.Code)
	}

	// Step 6: Create deployment.
	wDep := httptest.NewRecorder()
	handler.ServeHTTP(wDep, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "v1 deployment"}))
	if wDep.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment: expected 201, got %d", wDep.Code)
	}
	depID := decodeJSON(t, wDep.Body.String())["id"].(string)

	// Step 7: Create stage.
	wStage := httptest.NewRecorder()
	handler.ServeHTTP(wStage, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/stages", apiID),
		map[string]string{
			"stageName":    "v1",
			"deploymentId": depID,
			"description":  "version 1",
		}))
	if wStage.Code != http.StatusCreated {
		t.Fatalf("CreateStage: expected 201, got %d", wStage.Code)
	}

	// Step 8: Verify full state.
	// Verify 3 resources: root + /v1 + /v1/health.
	wResList := httptest.NewRecorder()
	handler.ServeHTTP(wResList, apigwReq(t, http.MethodGet,
		"/restapis/"+apiID+"/resources", nil))
	if wResList.Code != http.StatusOK {
		t.Fatalf("GetResources: expected 200, got %d", wResList.Code)
	}
	resItems, _ := decodeJSON(t, wResList.Body.String())["items"].([]any)
	if len(resItems) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(resItems))
	}

	// Verify 1 deployment.
	wDepList := httptest.NewRecorder()
	handler.ServeHTTP(wDepList, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/deployments", apiID), nil))
	depItems, _ := decodeJSON(t, wDepList.Body.String())["items"].([]any)
	if len(depItems) != 1 {
		t.Errorf("Expected 1 deployment, got %d", len(depItems))
	}

	// Verify 1 stage.
	wStageList := httptest.NewRecorder()
	handler.ServeHTTP(wStageList, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/stages", apiID), nil))
	stageItems, _ := decodeJSON(t, wStageList.Body.String())["item"].([]any)
	if len(stageItems) != 1 {
		t.Errorf("Expected 1 stage, got %d", len(stageItems))
	}

	// Step 9: Delete API.
	wDel := httptest.NewRecorder()
	handler.ServeHTTP(wDel, apigwReq(t, http.MethodDelete, "/restapis/"+apiID, nil))
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("DeleteRestApi: expected 204, got %d", wDel.Code)
	}

	// Verify everything is gone.
	wVerify := httptest.NewRecorder()
	handler.ServeHTTP(wVerify, apigwReq(t, http.MethodGet, "/restapis/"+apiID, nil))
	if wVerify.Code != http.StatusNotFound {
		t.Errorf("GetRestApi after delete: expected 404, got %d", wVerify.Code)
	}
}

// ---- Test 37: NotFoundException for non-existent route ----

func TestAPIGateway_NonExistentRoute(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/something-else", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("Non-existent route: expected 404, got %d", w.Code)
	}
}

// ---- Test 38: Deeply nested resource path computation ----

func TestAPIGateway_DeeplyNestedResourcePaths(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "deep-path-api")
	parentID := rootResourceID(t, handler, apiID)

	// Build /api/v2/users/{userId}/posts
	segments := []struct {
		pathPart     string
		expectedPath string
	}{
		{"api", "/api"},
		{"v2", "/api/v2"},
		{"users", "/api/v2/users"},
		{"{userId}", "/api/v2/users/{userId}"},
		{"posts", "/api/v2/users/{userId}/posts"},
	}

	for _, seg := range segments {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
			fmt.Sprintf("/restapis/%s/resources/%s", apiID, parentID),
			map[string]string{"pathPart": seg.pathPart}))
		if w.Code != http.StatusCreated {
			t.Fatalf("CreateResource %s: expected 201, got %d\nbody: %s", seg.pathPart, w.Code, w.Body.String())
		}
		m := decodeJSON(t, w.Body.String())
		gotPath := m["path"].(string)
		if gotPath != seg.expectedPath {
			t.Errorf("CreateResource %s: expected path=%s, got %s", seg.pathPart, seg.expectedPath, gotPath)
		}
		parentID = m["id"].(string)
	}

	// Verify total resources: root + 5 segments = 6
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodGet, "/restapis/"+apiID+"/resources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetResources: expected 200, got %d", w.Code)
	}
	items, _ := decodeJSON(t, w.Body.String())["items"].([]any)
	if len(items) != 6 {
		t.Errorf("GetResources: expected 6 resources (root + 5 nested), got %d", len(items))
	}
}

// ---- Test 39: PutMethod replaces existing method ----

func TestAPIGateway_PutMethod_ReplacesExisting(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "replace-method-api")
	rootID := rootResourceID(t, handler, apiID)

	// Create resource.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "auth"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	resID := decodeJSON(t, wRes.Body.String())["id"].(string)

	// PutMethod with NONE auth.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, resID),
		map[string]string{"authorizationType": "NONE"}))
	if w1.Code != http.StatusOK {
		t.Fatalf("PutMethod (first): expected 200, got %d", w1.Code)
	}

	// Replace with AWS_IAM auth.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, resID),
		map[string]string{"authorizationType": "AWS_IAM"}))
	if w2.Code != http.StatusOK {
		t.Fatalf("PutMethod (replace): expected 200, got %d", w2.Code)
	}

	// Verify the replacement took effect.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/GET", apiID, resID), nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("GetMethod: expected 200, got %d", wGet.Code)
	}
	m := decodeJSON(t, wGet.Body.String())
	if m["authorizationType"].(string) != "AWS_IAM" {
		t.Errorf("GetMethod after replace: expected authorizationType=AWS_IAM, got %s", m["authorizationType"])
	}
}

// ---- Test 40: Integration replaces existing integration ----

func TestAPIGateway_PutIntegration_ReplacesExisting(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "replace-int-api")
	rootID := rootResourceID(t, handler, apiID)

	// Setup: resource + method + initial integration.
	wRes := httptest.NewRecorder()
	handler.ServeHTTP(wRes, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/resources/%s", apiID, rootID),
		map[string]string{"pathPart": "swap"}))
	if wRes.Code != http.StatusCreated {
		t.Fatalf("CreateResource: expected 201, got %d", wRes.Code)
	}
	resID := decodeJSON(t, wRes.Body.String())["id"].(string)

	wMethod := httptest.NewRecorder()
	handler.ServeHTTP(wMethod, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST", apiID, resID),
		map[string]string{"authorizationType": "NONE"}))
	if wMethod.Code != http.StatusOK {
		t.Fatalf("PutMethod: expected 200, got %d", wMethod.Code)
	}

	// First integration: MOCK.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST/integration", apiID, resID),
		map[string]string{"type": "MOCK", "httpMethod": "POST"}))
	if w1.Code != http.StatusOK {
		t.Fatalf("PutIntegration (first): expected 200, got %d", w1.Code)
	}

	// Replace with AWS_PROXY.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, apigwReq(t, http.MethodPut,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST/integration", apiID, resID),
		map[string]string{
			"type":       "AWS_PROXY",
			"uri":        "arn:aws:apigateway:us-east-1:lambda:path/fn",
			"httpMethod": "POST",
		}))
	if w2.Code != http.StatusOK {
		t.Fatalf("PutIntegration (replace): expected 200, got %d", w2.Code)
	}

	// Verify the replacement.
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, apigwReq(t, http.MethodGet,
		fmt.Sprintf("/restapis/%s/resources/%s/methods/POST", apiID, resID), nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("GetMethod: expected 200, got %d", wGet.Code)
	}
	integration := decodeJSON(t, wGet.Body.String())["methodIntegration"].(map[string]any)
	if integration["type"].(string) != "AWS_PROXY" {
		t.Errorf("Integration after replace: expected type=AWS_PROXY, got %s", integration["type"])
	}
}

// ---- Test 41: Deployment has unique ID and createdDate ----

func TestAPIGateway_Deployment_Fields(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "dep-fields-api")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "test deployment fields"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment: expected 201, got %d", w.Code)
	}

	m := decodeJSON(t, w.Body.String())
	if m["id"].(string) == "" {
		t.Error("Deployment id must not be empty")
	}
	if m["description"].(string) != "test deployment fields" {
		t.Errorf("Deployment description: expected 'test deployment fields', got %s", m["description"])
	}
	if _, ok := m["createdDate"]; !ok {
		t.Error("Deployment missing createdDate")
	}
}

// ---- Test 42: Stage has correct fields ----

func TestAPIGateway_Stage_Fields(t *testing.T) {
	handler := newAPIGatewayHandler(t)

	apiID := mustCreateAPI(t, handler, "stage-fields-api")

	// Create deployment.
	wDep := httptest.NewRecorder()
	handler.ServeHTTP(wDep, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/deployments", apiID),
		map[string]string{"description": "dep"}))
	if wDep.Code != http.StatusCreated {
		t.Fatalf("CreateDeployment: expected 201, got %d", wDep.Code)
	}
	depID := decodeJSON(t, wDep.Body.String())["id"].(string)

	// Create stage.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, apigwReq(t, http.MethodPost,
		fmt.Sprintf("/restapis/%s/stages", apiID),
		map[string]string{
			"stageName":    "staging",
			"deploymentId": depID,
			"description":  "staging environment",
		}))
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateStage: expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}

	m := decodeJSON(t, w.Body.String())
	if m["stageName"].(string) != "staging" {
		t.Errorf("Stage stageName: expected staging, got %s", m["stageName"])
	}
	if m["deploymentId"].(string) != depID {
		t.Errorf("Stage deploymentId: expected %s, got %s", depID, m["deploymentId"])
	}
	if m["description"].(string) != "staging environment" {
		t.Errorf("Stage description: expected 'staging environment', got %s", m["description"])
	}
	if _, ok := m["createdDate"]; !ok {
		t.Error("Stage missing createdDate")
	}
}
