package lambda_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	lambdasvc "github.com/neureaux/cloudmock/services/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newLambdaGateway builds a full gateway stack with the Lambda service registered and IAM disabled.
func newLambdaGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(lambdasvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// lambdaReq builds an HTTP request targeting the Lambda service.
func lambdaReq(t *testing.T, method, path string, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/lambda/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// createZipFile creates a zip archive in memory with the given filename and content.
func createZipFile(t *testing.T, filename, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(filename)
	require.NoError(t, err)
	_, err = f.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// nodeAvailable returns true if node is installed.
func nodeAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

// pythonAvailable returns true if python3 is installed.
func pythonAvailable() bool {
	_, err := exec.LookPath("python3")
	return err == nil
}

func TestCreateFunction_GetFunction_ListFunctions(t *testing.T) {
	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200, body: JSON.stringify(event) });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "test-function",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"},
		"Description": "Test function",
		"Timeout": 10,
		"MemorySize": 256,
		"Environment": {"Variables": {"MY_VAR": "hello"}}
	}`

	// CreateFunction
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	assert.Equal(t, http.StatusCreated, w.Code, "CreateFunction should return 201")

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	assert.Equal(t, "test-function", createResp["FunctionName"])
	assert.Equal(t, "nodejs20.x", createResp["Runtime"])
	assert.Equal(t, "index.handler", createResp["Handler"])
	assert.Equal(t, "$LATEST", createResp["Version"])
	assert.Contains(t, createResp["FunctionArn"].(string), "test-function")
	assert.Equal(t, float64(10), createResp["Timeout"])
	assert.Equal(t, float64(256), createResp["MemorySize"])

	// Duplicate CreateFunction should fail.
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	assert.Equal(t, http.StatusConflict, w.Code, "Duplicate create should return 409")

	// GetFunction
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodGet, "/2015-03-31/functions/test-function", ""))
	assert.Equal(t, http.StatusOK, w.Code, "GetFunction should return 200")

	var getResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &getResp))
	config := getResp["Configuration"].(map[string]any)
	assert.Equal(t, "test-function", config["FunctionName"])
	assert.NotEmpty(t, getResp["Code"])

	// GetFunction not found
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodGet, "/2015-03-31/functions/nonexistent", ""))
	assert.Equal(t, http.StatusNotFound, w.Code, "GetFunction for missing function should return 404")

	// ListFunctions
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodGet, "/2015-03-31/functions", ""))
	assert.Equal(t, http.StatusOK, w.Code, "ListFunctions should return 200")

	var listResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	functions := listResp["Functions"].([]any)
	assert.Len(t, functions, 1)
}

func TestDeleteFunction(t *testing.T) {
	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200 });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "to-delete",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"}
	}`

	// Create
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// Delete
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodDelete, "/2015-03-31/functions/to-delete", ""))
	assert.Equal(t, http.StatusNoContent, w.Code, "DeleteFunction should return 204")

	// Delete again should 404
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodDelete, "/2015-03-31/functions/to-delete", ""))
	assert.Equal(t, http.StatusNotFound, w.Code, "Delete missing function should return 404")

	// Get should 404
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodGet, "/2015-03-31/functions/to-delete", ""))
	assert.Equal(t, http.StatusNotFound, w.Code, "GetFunction after delete should return 404")
}

func TestInvokeDryRun(t *testing.T) {
	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200 });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "dry-run-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"}
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// DryRun invocation
	req := lambdaReq(t, http.MethodPost, "/2015-03-31/functions/dry-run-func/invocations", `{"test": true}`)
	req.Header.Set("X-Amz-Invocation-Type", "DryRun")

	w = httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code, "DryRun invocation should return 204")
}

func TestInvokeNodeJS(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("node not available, skipping Node.js invoke test")
	}

	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200, body: JSON.stringify(event) });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "node-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"}
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// Invoke
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost,
		"/2015-03-31/functions/node-func/invocations", `{"hello": "world"}`))
	assert.Equal(t, http.StatusOK, w.Code, "Invoke should return 200")

	var invokeResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invokeResp))
	assert.Equal(t, float64(200), invokeResp["statusCode"])

	// Verify the body contains the input event.
	bodyStr, ok := invokeResp["body"].(string)
	require.True(t, ok, "body should be a string")
	var bodyParsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodyStr), &bodyParsed))
	assert.Equal(t, "world", bodyParsed["hello"])
}

func TestInvokePython(t *testing.T) {
	if !pythonAvailable() {
		t.Skip("python3 not available, skipping Python invoke test")
	}

	gw := newLambdaGateway(t)

	pythonCode := createZipFile(t, "lambda_function.py",
		`def handler(event, context):
    return {"statusCode": 200, "body": str(event)}
`)
	b64Code := base64.StdEncoding.EncodeToString(pythonCode)

	createBody := `{
		"FunctionName": "python-func",
		"Runtime": "python3.12",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "lambda_function.handler",
		"Code": {"ZipFile": "` + b64Code + `"}
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// Invoke
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost,
		"/2015-03-31/functions/python-func/invocations", `{"hello": "world"}`))
	assert.Equal(t, http.StatusOK, w.Code, "Invoke should return 200")

	var invokeResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invokeResp))
	assert.Equal(t, float64(200), invokeResp["statusCode"])
}

func TestUpdateFunctionCode(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("node not available, skipping UpdateFunctionCode test")
	}

	gw := newLambdaGateway(t)

	// Create with v1 code.
	v1Code := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200, body: "v1" });`)
	b64V1 := base64.StdEncoding.EncodeToString(v1Code)

	createBody := `{
		"FunctionName": "update-code-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64V1 + `"}
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// Invoke v1.
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost,
		"/2015-03-31/functions/update-code-func/invocations", `{}`))
	require.Equal(t, http.StatusOK, w.Code)
	var v1Resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v1Resp))
	assert.Equal(t, "v1", v1Resp["body"])

	// Update to v2 code.
	v2Code := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200, body: "v2" });`)
	b64V2 := base64.StdEncoding.EncodeToString(v2Code)

	updateBody := `{"ZipFile": "` + b64V2 + `"}`

	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPut,
		"/2015-03-31/functions/update-code-func/code", updateBody))
	assert.Equal(t, http.StatusOK, w.Code, "UpdateFunctionCode should return 200")

	// Invoke v2.
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost,
		"/2015-03-31/functions/update-code-func/invocations", `{}`))
	require.Equal(t, http.StatusOK, w.Code)
	var v2Resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v2Resp))
	assert.Equal(t, "v2", v2Resp["body"])
}

func TestGetFunctionConfiguration_UpdateFunctionConfiguration(t *testing.T) {
	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200 });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "config-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"},
		"Timeout": 3,
		"MemorySize": 128
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// GetFunctionConfiguration
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodGet,
		"/2015-03-31/functions/config-func/configuration", ""))
	assert.Equal(t, http.StatusOK, w.Code)

	var configResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &configResp))
	assert.Equal(t, "config-func", configResp["FunctionName"])
	assert.Equal(t, float64(3), configResp["Timeout"])
	assert.Equal(t, float64(128), configResp["MemorySize"])

	// UpdateFunctionConfiguration
	updateBody := `{"Timeout": 30, "MemorySize": 512, "Description": "updated"}`
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPut,
		"/2015-03-31/functions/config-func/configuration", updateBody))
	assert.Equal(t, http.StatusOK, w.Code)

	var updateResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updateResp))
	assert.Equal(t, float64(30), updateResp["Timeout"])
	assert.Equal(t, float64(512), updateResp["MemorySize"])
	assert.Equal(t, "updated", updateResp["Description"])

	// Verify via GetFunctionConfiguration
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodGet,
		"/2015-03-31/functions/config-func/configuration", ""))
	require.Equal(t, http.StatusOK, w.Code)

	var verifyResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &verifyResp))
	assert.Equal(t, float64(30), verifyResp["Timeout"])
	assert.Equal(t, float64(512), verifyResp["MemorySize"])
	assert.Equal(t, "updated", verifyResp["Description"])
}

func TestInvokeWithEnvironmentVariables(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("node not available, skipping environment variables test")
	}

	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200, body: process.env.MY_VAR || "not set" });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "env-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"},
		"Environment": {"Variables": {"MY_VAR": "hello-from-env"}}
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// Invoke
	w = httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost,
		"/2015-03-31/functions/env-func/invocations", `{}`))
	assert.Equal(t, http.StatusOK, w.Code)

	var invokeResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invokeResp))
	assert.Equal(t, "hello-from-env", invokeResp["body"])
}

func TestInvokeNotFound(t *testing.T) {
	gw := newLambdaGateway(t)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost,
		"/2015-03-31/functions/nonexistent/invocations", `{}`))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInvokeEventType(t *testing.T) {
	if !nodeAvailable() {
		t.Skip("node not available, skipping Event invocation test")
	}

	gw := newLambdaGateway(t)

	nodeCode := createZipFile(t, "index.js",
		`exports.handler = async (event) => ({ statusCode: 200, body: "ok" });`)
	b64Code := base64.StdEncoding.EncodeToString(nodeCode)

	createBody := `{
		"FunctionName": "event-func",
		"Runtime": "nodejs20.x",
		"Role": "arn:aws:iam::000000000000:role/lambda-role",
		"Handler": "index.handler",
		"Code": {"ZipFile": "` + b64Code + `"}
	}`

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", createBody))
	require.Equal(t, http.StatusCreated, w.Code)

	// Invoke with Event type (async, should return 202).
	req := lambdaReq(t, http.MethodPost, "/2015-03-31/functions/event-func/invocations", `{}`)
	req.Header.Set("X-Amz-Invocation-Type", "Event")

	w = httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	assert.Equal(t, http.StatusAccepted, w.Code, "Event invocation should return 202")
}

func TestCreateFunction_ValidationErrors(t *testing.T) {
	gw := newLambdaGateway(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing FunctionName", `{"Runtime":"nodejs20.x","Handler":"index.handler","Role":"arn:aws:iam::000000000000:role/r","Code":{"ZipFile":"UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA=="}}`},
		{"missing Runtime", `{"FunctionName":"f","Handler":"index.handler","Role":"arn:aws:iam::000000000000:role/r","Code":{"ZipFile":"UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA=="}}`},
		{"missing Handler", `{"FunctionName":"f","Runtime":"nodejs20.x","Role":"arn:aws:iam::000000000000:role/r","Code":{"ZipFile":"UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA=="}}`},
		{"missing Role", `{"FunctionName":"f","Runtime":"nodejs20.x","Handler":"index.handler","Code":{"ZipFile":"UEsFBgAAAAAAAAAAAAAAAAAAAAAAAA=="}}`},
		{"missing Code", `{"FunctionName":"f","Runtime":"nodejs20.x","Handler":"index.handler","Role":"arn:aws:iam::000000000000:role/r"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			gw.ServeHTTP(w, lambdaReq(t, http.MethodPost, "/2015-03-31/functions", tt.body))
			assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 for %s", tt.name)
		})
	}
}
