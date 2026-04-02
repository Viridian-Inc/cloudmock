package appsync_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	appsvc "github.com/neureaux/cloudmock/services/appsync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAppSyncGateway builds a full gateway stack with the AppSync service registered and IAM disabled.
func newAppSyncGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(appsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// appsyncReq builds an HTTP request targeting the AppSync service via ?Action= query
// param and JSON body. AppSync uses ctx.Action dispatch, so we pass the action as a
// query parameter and the Authorization header routes to the appsync service.
func appsyncReq(t *testing.T, action string, body map[string]any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		require.NoError(t, err)
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/?Action="+action, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/appsync/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// doAppSync sends an AppSync request through the gateway and returns the response recorder.
func doAppSync(t *testing.T, gw http.Handler, action string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, appsyncReq(t, action, body))
	return w
}

// decodeBody unmarshals a JSON response body into a map.
func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

// mustCreateApi creates an API through the gateway and returns its apiId.
func mustCreateApi(t *testing.T, gw http.Handler, name string) string {
	t.Helper()
	w := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{"name": name})
	require.Equal(t, http.StatusOK, w.Code, "CreateGraphqlApi failed: %s", w.Body.String())
	api := decodeBody(t, w)["graphqlApi"].(map[string]any)
	id, ok := api["apiId"].(string)
	require.True(t, ok && id != "")
	return id
}

// ---- GraphQL API tests ----

func TestCreateGraphqlApi(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{
		"name": "my-api", "authenticationType": "AMAZON_COGNITO_USER_POOLS",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api := decodeBody(t, w)["graphqlApi"].(map[string]any)
	assert.Equal(t, "my-api", api["name"])
	assert.Equal(t, "AMAZON_COGNITO_USER_POOLS", api["authenticationType"])
	assert.NotEmpty(t, api["apiId"])
	assert.NotEmpty(t, api["arn"])
	uris := api["uris"].(map[string]any)
	assert.NotEmpty(t, uris["GRAPHQL"])
}

func TestCreateGraphqlApiDefaultAuth(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{"name": "my-api"})
	assert.Equal(t, http.StatusOK, w.Code)
	api := decodeBody(t, w)["graphqlApi"].(map[string]any)
	assert.Equal(t, "API_KEY", api["authenticationType"])
}

func TestCreateGraphqlApiMissingName(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{})
	assert.NotEqual(t, http.StatusOK, w.Code, "expected error for missing name")
}

func TestGetGraphqlApi(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "GetGraphqlApi", map[string]any{"apiId": apiID})
	assert.Equal(t, http.StatusOK, w.Code)
	api := decodeBody(t, w)["graphqlApi"].(map[string]any)
	assert.Equal(t, "my-api", api["name"])
}

func TestGetGraphqlApiNotFound(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "GetGraphqlApi", map[string]any{"apiId": "nonexistent"})
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "NotFoundException")
}

func TestListGraphqlApis(t *testing.T) {
	gw := newAppSyncGateway(t)
	mustCreateApi(t, gw, "api-1")
	mustCreateApi(t, gw, "api-2")

	w := doAppSync(t, gw, "ListGraphqlApis", map[string]any{})
	assert.Equal(t, http.StatusOK, w.Code)
	apis := decodeBody(t, w)["graphqlApis"].([]any)
	assert.Len(t, apis, 2)
}

func TestUpdateGraphqlApi(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "UpdateGraphqlApi", map[string]any{
		"apiId": apiID, "name": "updated-api",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	api := decodeBody(t, w)["graphqlApi"].(map[string]any)
	assert.Equal(t, "updated-api", api["name"])
}

func TestDeleteGraphqlApi(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "DeleteGraphqlApi", map[string]any{"apiId": apiID})
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify it's gone.
	w = doAppSync(t, gw, "GetGraphqlApi", map[string]any{"apiId": apiID})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Data Source tests ----

func TestCreateDataSource(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "CreateDataSource", map[string]any{
		"apiId": apiID, "name": "my-ds", "type": "AMAZON_DYNAMODB",
		"description": "test ds",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	ds := decodeBody(t, w)["dataSource"].(map[string]any)
	assert.Equal(t, "my-ds", ds["name"])
	assert.Equal(t, "AMAZON_DYNAMODB", ds["type"])
	assert.NotEmpty(t, ds["dataSourceArn"])
}

func TestGetDataSource(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateDataSource", map[string]any{
		"apiId": apiID, "name": "my-ds", "type": "NONE",
	})

	w := doAppSync(t, gw, "GetDataSource", map[string]any{
		"apiId": apiID, "name": "my-ds",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	ds := decodeBody(t, w)["dataSource"].(map[string]any)
	assert.Equal(t, "my-ds", ds["name"])
}

func TestListDataSources(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateDataSource", map[string]any{"apiId": apiID, "name": "ds-1", "type": "NONE"})
	doAppSync(t, gw, "CreateDataSource", map[string]any{"apiId": apiID, "name": "ds-2", "type": "NONE"})

	w := doAppSync(t, gw, "ListDataSources", map[string]any{"apiId": apiID})
	assert.Equal(t, http.StatusOK, w.Code)
	dss := decodeBody(t, w)["dataSources"].([]any)
	assert.Len(t, dss, 2)
}

func TestDeleteDataSource(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateDataSource", map[string]any{"apiId": apiID, "name": "my-ds", "type": "NONE"})

	w := doAppSync(t, gw, "DeleteDataSource", map[string]any{"apiId": apiID, "name": "my-ds"})
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAppSync(t, gw, "GetDataSource", map[string]any{"apiId": apiID, "name": "my-ds"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Resolver tests ----

func TestCreateResolver(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "CreateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
		"dataSourceName": "my-ds", "kind": "UNIT",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	r := decodeBody(t, w)["resolver"].(map[string]any)
	assert.Equal(t, "Query", r["typeName"])
	assert.Equal(t, "getUser", r["fieldName"])
	assert.Equal(t, "UNIT", r["kind"])
}

func TestGetResolver(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})

	w := doAppSync(t, gw, "GetResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	r := decodeBody(t, w)["resolver"].(map[string]any)
	assert.Equal(t, "getUser", r["fieldName"])
}

func TestListResolvers(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})
	doAppSync(t, gw, "CreateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "listUsers",
	})

	w := doAppSync(t, gw, "ListResolvers", map[string]any{
		"apiId": apiID, "typeName": "Query",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	resolvers := decodeBody(t, w)["resolvers"].([]any)
	assert.Len(t, resolvers, 2)
}

func TestUpdateResolver(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})

	w := doAppSync(t, gw, "UpdateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
		"dataSourceName": "updated-ds",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	r := decodeBody(t, w)["resolver"].(map[string]any)
	assert.Equal(t, "updated-ds", r["dataSourceName"])
}

func TestDeleteResolver(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})

	w := doAppSync(t, gw, "DeleteResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAppSync(t, gw, "GetResolver", map[string]any{
		"apiId": apiID, "typeName": "Query", "fieldName": "getUser",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Function tests ----

func TestCreateFunction(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "CreateFunction", map[string]any{
		"apiId": apiID, "name": "my-func", "dataSourceName": "my-ds",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	f := decodeBody(t, w)["functionConfiguration"].(map[string]any)
	assert.Equal(t, "my-func", f["name"])
	assert.NotEmpty(t, f["functionId"])
}

func TestListFunctions(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateFunction", map[string]any{
		"apiId": apiID, "name": "func-1", "dataSourceName": "ds",
	})
	doAppSync(t, gw, "CreateFunction", map[string]any{
		"apiId": apiID, "name": "func-2", "dataSourceName": "ds",
	})

	w := doAppSync(t, gw, "ListFunctions", map[string]any{"apiId": apiID})
	assert.Equal(t, http.StatusOK, w.Code)
	funcs := decodeBody(t, w)["functions"].([]any)
	assert.Len(t, funcs, 2)
}

// ---- API Key tests ----

func TestCreateApiKey(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	w := doAppSync(t, gw, "CreateApiKey", map[string]any{
		"apiId": apiID, "description": "test key",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	key := decodeBody(t, w)["apiKey"].(map[string]any)
	assert.NotEmpty(t, key["id"])
	assert.True(t, key["expires"].(float64) > 0)
}

func TestListApiKeys(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")
	doAppSync(t, gw, "CreateApiKey", map[string]any{"apiId": apiID})
	doAppSync(t, gw, "CreateApiKey", map[string]any{"apiId": apiID})

	w := doAppSync(t, gw, "ListApiKeys", map[string]any{"apiId": apiID})
	assert.Equal(t, http.StatusOK, w.Code)
	keys := decodeBody(t, w)["apiKeys"].([]any)
	assert.Len(t, keys, 2)
}

func TestDeleteApiKey(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	createW := doAppSync(t, gw, "CreateApiKey", map[string]any{"apiId": apiID})
	keyID := decodeBody(t, createW)["apiKey"].(map[string]any)["id"].(string)

	w := doAppSync(t, gw, "DeleteApiKey", map[string]any{"apiId": apiID, "id": keyID})
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---- Tagging ----

func TestTagApi(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiID := mustCreateApi(t, gw, "my-api")

	getW := doAppSync(t, gw, "GetGraphqlApi", map[string]any{"apiId": apiID})
	arn := decodeBody(t, getW)["graphqlApi"].(map[string]any)["arn"].(string)

	w := doAppSync(t, gw, "TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        map[string]any{"env": "prod"},
	})
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAppSync(t, gw, "ListTagsForResource", map[string]any{"resourceArn": arn})
	assert.Equal(t, http.StatusOK, w.Code)
	tags := decodeBody(t, w)["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
}

func TestUntagApi(t *testing.T) {
	gw := newAppSyncGateway(t)

	createW := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{
		"name": "my-api", "tags": map[string]any{"env": "prod", "team": "alpha"},
	})
	api := decodeBody(t, createW)["graphqlApi"].(map[string]any)
	arn := api["arn"].(string)

	w := doAppSync(t, gw, "UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []string{"team"},
	})
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAppSync(t, gw, "ListTagsForResource", map[string]any{"resourceArn": arn})
	tags := decodeBody(t, w)["tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Nil(t, tags["team"])
}

// ---- Realistic URL tests ----

func TestGraphqlApiURLs(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{"name": "url-api"})
	assert.Equal(t, http.StatusOK, w.Code)
	api := decodeBody(t, w)["graphqlApi"].(map[string]any)
	uris := api["uris"].(map[string]any)

	graphqlURL := uris["GRAPHQL"].(string)
	realtimeURL := uris["REALTIME"].(string)

	assert.Contains(t, graphqlURL, "appsync-api.")
	assert.Contains(t, graphqlURL, ".amazonaws.com/graphql")
	assert.True(t, len(graphqlURL) > 40, "GraphQL URL should be realistic length")

	assert.Contains(t, realtimeURL, "wss://")
	assert.Contains(t, realtimeURL, "appsync-realtime-api.")
	assert.Contains(t, realtimeURL, ".amazonaws.com/graphql")
}

func TestCreateGraphqlApiMissingNameReturnsError(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "CreateGraphqlApi", map[string]any{})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ---- Type tests ----

func TestCreateType(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "type-api")

	w := doAppSync(t, gw, "CreateType", map[string]any{
		"apiId":      apiId,
		"typeName":   "Query",
		"definition": "type Query { hello: String }",
		"format":     "SDL",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	td := decodeBody(t, w)["type"].(map[string]any)
	assert.Equal(t, "Query", td["name"])
	assert.Equal(t, "SDL", td["format"])
}

func TestGetType(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "get-type-api")

	doAppSync(t, gw, "CreateType", map[string]any{
		"apiId": apiId, "typeName": "Query",
		"definition": "type Query { hello: String }", "format": "SDL",
	})

	w := doAppSync(t, gw, "GetType", map[string]any{
		"apiId": apiId, "typeName": "Query", "format": "SDL",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	td := decodeBody(t, w)["type"].(map[string]any)
	assert.Equal(t, "Query", td["name"])
}

func TestGetTypeNotFound(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "no-type-api")

	w := doAppSync(t, gw, "GetType", map[string]any{
		"apiId": apiId, "typeName": "NonExistent", "format": "SDL",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListTypes(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "list-types-api")

	doAppSync(t, gw, "CreateType", map[string]any{
		"apiId": apiId, "typeName": "Query",
		"definition": "type Query { hello: String }", "format": "SDL",
	})
	doAppSync(t, gw, "CreateType", map[string]any{
		"apiId": apiId, "typeName": "Mutation",
		"definition": "type Mutation { createItem: String }", "format": "SDL",
	})

	w := doAppSync(t, gw, "ListTypes", map[string]any{
		"apiId": apiId, "format": "SDL",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	types := decodeBody(t, w)["types"].([]any)
	assert.Len(t, types, 2)
}

func TestUpdateType(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "update-type-api")

	doAppSync(t, gw, "CreateType", map[string]any{
		"apiId": apiId, "typeName": "Query",
		"definition": "type Query { hello: String }", "format": "SDL",
	})

	w := doAppSync(t, gw, "UpdateType", map[string]any{
		"apiId": apiId, "typeName": "Query",
		"definition": "type Query { hello: String world: String }", "format": "SDL",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	td := decodeBody(t, w)["type"].(map[string]any)
	assert.Equal(t, "Query", td["name"])
}

func TestDeleteType(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "delete-type-api")

	doAppSync(t, gw, "CreateType", map[string]any{
		"apiId": apiId, "typeName": "Query",
		"definition": "type Query { hello: String }", "format": "SDL",
	})

	w := doAppSync(t, gw, "DeleteType", map[string]any{
		"apiId": apiId, "typeName": "Query",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deleted
	w2 := doAppSync(t, gw, "GetType", map[string]any{
		"apiId": apiId, "typeName": "Query", "format": "SDL",
	})
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

// ---- Schema tests ----

func TestStartSchemaCreation(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "schema-api")

	w := doAppSync(t, gw, "StartSchemaCreation", map[string]any{
		"apiId":      apiId,
		"definition": "type Query { hello: String }",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "PROCESSING", body["status"])
}

func TestGetSchemaCreationStatus(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "schema-status-api")

	// Before starting: NOT_STARTED
	w := doAppSync(t, gw, "GetSchemaCreationStatus", map[string]any{"apiId": apiId})
	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "NOT_STARTED", body["status"])

	// Start schema creation
	doAppSync(t, gw, "StartSchemaCreation", map[string]any{
		"apiId": apiId, "definition": "type Query { hello: String }",
	})

	// After starting: ACTIVE (instant in mock)
	w2 := doAppSync(t, gw, "GetSchemaCreationStatus", map[string]any{"apiId": apiId})
	assert.Equal(t, http.StatusOK, w2.Code)
	body2 := decodeBody(t, w2)
	assert.Equal(t, "ACTIVE", body2["status"])
}

func TestStartSchemaCreationMissingDefinition(t *testing.T) {
	gw := newAppSyncGateway(t)
	apiId := mustCreateApi(t, gw, "bad-schema-api")

	w := doAppSync(t, gw, "StartSchemaCreation", map[string]any{"apiId": apiId})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ---- Invalid action ----

func TestInvalidAction(t *testing.T) {
	gw := newAppSyncGateway(t)
	w := doAppSync(t, gw, "BogusAction", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "InvalidAction")
}
