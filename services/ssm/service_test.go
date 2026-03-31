package ssm_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	ssmsvc "github.com/neureaux/cloudmock/services/ssm"
)

// newSSMGateway builds a full gateway stack with the SSM service registered and IAM disabled.
func newSSMGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(ssmsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// ssmReq builds a JSON POST request targeting the SSM service via X-Amz-Target.
func ssmReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("ssmReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonSSM."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ssm/aws4_request, SignedHeaders=host, Signature=abc123")
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

// ---- Test 1: PutParameter + GetParameter round-trip ----

func TestSSM_PutAndGetParameter(t *testing.T) {
	handler := newSSMGateway(t)

	// PutParameter
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ssmReq(t, "PutParameter", map[string]any{
		"Name":        "/app/config/db-host",
		"Value":       "localhost",
		"Type":        "String",
		"Description": "Database host",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutParameter: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mp := decodeJSON(t, wp.Body.String())
	version, _ := mp["Version"].(float64)
	if version != 1 {
		t.Errorf("PutParameter: expected Version=1, got %v", version)
	}
	tier, _ := mp["Tier"].(string)
	if tier != "Standard" {
		t.Errorf("PutParameter: expected Tier=Standard, got %q", tier)
	}

	// GetParameter
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, ssmReq(t, "GetParameter", map[string]any{
		"Name":           "/app/config/db-host",
		"WithDecryption": false,
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetParameter: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mg := decodeJSON(t, wg.Body.String())
	param, ok := mg["Parameter"].(map[string]any)
	if !ok {
		t.Fatalf("GetParameter: missing Parameter in response\nbody: %s", wg.Body.String())
	}
	if param["Name"].(string) != "/app/config/db-host" {
		t.Errorf("GetParameter: expected Name=%q, got %q", "/app/config/db-host", param["Name"])
	}
	if param["Value"].(string) != "localhost" {
		t.Errorf("GetParameter: expected Value=%q, got %q", "localhost", param["Value"])
	}
	if param["Type"].(string) != "String" {
		t.Errorf("GetParameter: expected Type=String, got %q", param["Type"])
	}
	if param["Version"].(float64) != 1 {
		t.Errorf("GetParameter: expected Version=1, got %v", param["Version"])
	}
	arn, _ := param["ARN"].(string)
	if arn == "" {
		t.Error("GetParameter: ARN is empty")
	}
}

// ---- Test 2: SecureString type ----

func TestSSM_SecureString(t *testing.T) {
	handler := newSSMGateway(t)

	// PutParameter with SecureString
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ssmReq(t, "PutParameter", map[string]any{
		"Name":  "/app/secrets/api-key",
		"Value": "supersecret123",
		"Type":  "SecureString",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutParameter SecureString: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	// GetParameter with WithDecryption=true
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, ssmReq(t, "GetParameter", map[string]any{
		"Name":           "/app/secrets/api-key",
		"WithDecryption": true,
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetParameter SecureString: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mg := decodeJSON(t, wg.Body.String())
	param := mg["Parameter"].(map[string]any)
	if param["Type"].(string) != "SecureString" {
		t.Errorf("SecureString: expected Type=SecureString, got %q", param["Type"])
	}
	if param["Value"].(string) != "supersecret123" {
		t.Errorf("SecureString: expected Value=%q, got %q", "supersecret123", param["Value"])
	}
}

// ---- Test 3: GetParametersByPath with hierarchical names ----

func TestSSM_GetParametersByPath(t *testing.T) {
	handler := newSSMGateway(t)

	// Create a hierarchy of parameters
	params := []struct {
		name  string
		value string
	}{
		{"/app/prod/db-host", "prod-db.example.com"},
		{"/app/prod/db-port", "5432"},
		{"/app/prod/cache/host", "redis.example.com"},
		{"/app/dev/db-host", "localhost"},
		{"/other/param", "other"},
	}

	for _, p := range params {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ssmReq(t, "PutParameter", map[string]any{
			"Name":  p.name,
			"Value": p.value,
			"Type":  "String",
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("PutParameter %s: expected 200, got %d\nbody: %s", p.name, w.Code, w.Body.String())
		}
	}

	// GetParametersByPath for /app/prod, non-recursive (should get db-host, db-port but not cache/host)
	wpath := httptest.NewRecorder()
	handler.ServeHTTP(wpath, ssmReq(t, "GetParametersByPath", map[string]any{
		"Path":      "/app/prod",
		"Recursive": false,
	}))
	if wpath.Code != http.StatusOK {
		t.Fatalf("GetParametersByPath non-recursive: expected 200, got %d\nbody: %s", wpath.Code, wpath.Body.String())
	}

	mpath := decodeJSON(t, wpath.Body.String())
	pathParams, ok := mpath["Parameters"].([]any)
	if !ok {
		t.Fatalf("GetParametersByPath: missing Parameters\nbody: %s", wpath.Body.String())
	}
	if len(pathParams) != 2 {
		t.Errorf("GetParametersByPath non-recursive: expected 2 params, got %d", len(pathParams))
	}

	// Verify the names returned
	names := make(map[string]bool)
	for _, item := range pathParams {
		p := item.(map[string]any)
		names[p["Name"].(string)] = true
	}
	if !names["/app/prod/db-host"] {
		t.Error("GetParametersByPath non-recursive: missing /app/prod/db-host")
	}
	if !names["/app/prod/db-port"] {
		t.Error("GetParametersByPath non-recursive: missing /app/prod/db-port")
	}
	if names["/app/prod/cache/host"] {
		t.Error("GetParametersByPath non-recursive: should NOT include /app/prod/cache/host")
	}

	// GetParametersByPath for /app/prod, recursive (should include cache/host too)
	wrec := httptest.NewRecorder()
	handler.ServeHTTP(wrec, ssmReq(t, "GetParametersByPath", map[string]any{
		"Path":      "/app/prod",
		"Recursive": true,
	}))
	if wrec.Code != http.StatusOK {
		t.Fatalf("GetParametersByPath recursive: expected 200, got %d\nbody: %s", wrec.Code, wrec.Body.String())
	}

	mrec := decodeJSON(t, wrec.Body.String())
	recParams := mrec["Parameters"].([]any)
	if len(recParams) != 3 {
		t.Errorf("GetParametersByPath recursive: expected 3 params, got %d", len(recParams))
	}
}

// ---- Test 4: DeleteParameter then GetParameter returns ParameterNotFound ----

func TestSSM_DeleteParameter_ThenGetReturnsNotFound(t *testing.T) {
	handler := newSSMGateway(t)

	// PutParameter
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ssmReq(t, "PutParameter", map[string]any{
		"Name":  "/doomed/param",
		"Value": "bye",
		"Type":  "String",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutParameter: %d %s", wp.Code, wp.Body.String())
	}

	// DeleteParameter
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ssmReq(t, "DeleteParameter", map[string]any{
		"Name": "/doomed/param",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteParameter: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// GetParameter should return ParameterNotFound (400)
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, ssmReq(t, "GetParameter", map[string]any{
		"Name": "/doomed/param",
	}))
	if wg.Code != http.StatusBadRequest {
		t.Fatalf("GetParameter after delete: expected 400, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	errBody := decodeJSON(t, wg.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ParameterNotFound" {
		t.Errorf("GetParameter after delete: expected __type=ParameterNotFound, got %q", errType)
	}
}

// ---- Test 5: PutParameter overwrite increments version ----

func TestSSM_PutParameter_OverwriteIncrementsVersion(t *testing.T) {
	handler := newSSMGateway(t)

	// First put
	wp1 := httptest.NewRecorder()
	handler.ServeHTTP(wp1, ssmReq(t, "PutParameter", map[string]any{
		"Name":  "/versioned/param",
		"Value": "version-one",
		"Type":  "String",
	}))
	if wp1.Code != http.StatusOK {
		t.Fatalf("PutParameter v1: %d %s", wp1.Code, wp1.Body.String())
	}
	m1 := decodeJSON(t, wp1.Body.String())
	if m1["Version"].(float64) != 1 {
		t.Errorf("PutParameter v1: expected Version=1, got %v", m1["Version"])
	}

	// Second put without overwrite → should fail
	wp2 := httptest.NewRecorder()
	handler.ServeHTTP(wp2, ssmReq(t, "PutParameter", map[string]any{
		"Name":      "/versioned/param",
		"Value":     "version-two",
		"Type":      "String",
		"Overwrite": false,
	}))
	if wp2.Code != http.StatusBadRequest {
		t.Fatalf("PutParameter no-overwrite: expected 400, got %d\nbody: %s", wp2.Code, wp2.Body.String())
	}
	errBody := decodeJSON(t, wp2.Body.String())
	if errBody["__type"].(string) != "ParameterAlreadyExists" {
		t.Errorf("PutParameter no-overwrite: expected ParameterAlreadyExists, got %q", errBody["__type"])
	}

	// Third put with overwrite=true → should succeed and version=2
	wp3 := httptest.NewRecorder()
	handler.ServeHTTP(wp3, ssmReq(t, "PutParameter", map[string]any{
		"Name":      "/versioned/param",
		"Value":     "version-two",
		"Type":      "String",
		"Overwrite": true,
	}))
	if wp3.Code != http.StatusOK {
		t.Fatalf("PutParameter overwrite: expected 200, got %d\nbody: %s", wp3.Code, wp3.Body.String())
	}
	m3 := decodeJSON(t, wp3.Body.String())
	if m3["Version"].(float64) != 2 {
		t.Errorf("PutParameter overwrite: expected Version=2, got %v", m3["Version"])
	}

	// Verify via GetParameter
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, ssmReq(t, "GetParameter", map[string]any{
		"Name": "/versioned/param",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetParameter after overwrite: %d %s", wg.Code, wg.Body.String())
	}
	mg := decodeJSON(t, wg.Body.String())
	param := mg["Parameter"].(map[string]any)
	if param["Value"].(string) != "version-two" {
		t.Errorf("GetParameter after overwrite: expected Value=version-two, got %q", param["Value"])
	}
	if param["Version"].(float64) != 2 {
		t.Errorf("GetParameter after overwrite: expected Version=2, got %v", param["Version"])
	}
}

// ---- Test 6: GetParameters with some invalid names ----

func TestSSM_GetParameters_WithInvalidNames(t *testing.T) {
	handler := newSSMGateway(t)

	// Create some parameters
	for _, name := range []string{"/valid/param1", "/valid/param2"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ssmReq(t, "PutParameter", map[string]any{
			"Name":  name,
			"Value": "value",
			"Type":  "String",
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("PutParameter %s: %d %s", name, w.Code, w.Body.String())
		}
	}

	// GetParameters with some valid and some invalid names
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, ssmReq(t, "GetParameters", map[string]any{
		"Names": []string{"/valid/param1", "/valid/param2", "/does/not/exist", "/also/missing"},
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetParameters: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mg := decodeJSON(t, wg.Body.String())

	// Check found parameters
	params, ok := mg["Parameters"].([]any)
	if !ok {
		t.Fatalf("GetParameters: missing Parameters\nbody: %s", wg.Body.String())
	}
	if len(params) != 2 {
		t.Errorf("GetParameters: expected 2 valid params, got %d", len(params))
	}

	// Check invalid parameters
	invalid, ok := mg["InvalidParameters"].([]any)
	if !ok {
		t.Fatalf("GetParameters: missing InvalidParameters\nbody: %s", wg.Body.String())
	}
	if len(invalid) != 2 {
		t.Errorf("GetParameters: expected 2 invalid params, got %d", len(invalid))
	}

	invalidNames := make(map[string]bool)
	for _, item := range invalid {
		invalidNames[item.(string)] = true
	}
	if !invalidNames["/does/not/exist"] {
		t.Error("GetParameters: expected /does/not/exist in InvalidParameters")
	}
	if !invalidNames["/also/missing"] {
		t.Error("GetParameters: expected /also/missing in InvalidParameters")
	}
}

// ---- Test 7: DescribeParameters returns metadata without revealing values ----

func TestSSM_DescribeParameters(t *testing.T) {
	handler := newSSMGateway(t)

	// Create a parameter with description
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ssmReq(t, "PutParameter", map[string]any{
		"Name":        "/describe/test",
		"Value":       "secret-value",
		"Type":        "SecureString",
		"Description": "A described parameter",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutParameter: %d %s", wp.Code, wp.Body.String())
	}

	// DescribeParameters
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ssmReq(t, "DescribeParameters", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeParameters: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	paramList, ok := md["Parameters"].([]any)
	if !ok {
		t.Fatalf("DescribeParameters: missing Parameters\nbody: %s", wd.Body.String())
	}
	if len(paramList) == 0 {
		t.Fatal("DescribeParameters: expected at least 1 parameter")
	}

	// Find our parameter in the list
	var found map[string]any
	for _, item := range paramList {
		p := item.(map[string]any)
		if p["Name"].(string) == "/describe/test" {
			found = p
			break
		}
	}
	if found == nil {
		t.Fatal("DescribeParameters: /describe/test not found in list")
	}

	// Should have metadata but not value
	if _, hasValue := found["Value"]; hasValue {
		t.Error("DescribeParameters: response should not include Value")
	}
	if found["Type"].(string) != "SecureString" {
		t.Errorf("DescribeParameters: expected Type=SecureString, got %q", found["Type"])
	}
	if found["Description"].(string) != "A described parameter" {
		t.Errorf("DescribeParameters: expected Description=%q, got %q", "A described parameter", found["Description"])
	}
}

// ---- Test 8: StringList type ----

func TestSSM_StringListType(t *testing.T) {
	handler := newSSMGateway(t)

	// PutParameter with StringList type
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ssmReq(t, "PutParameter", map[string]any{
		"Name":  "/app/config/allowed-regions",
		"Value": "us-east-1,us-west-2,eu-west-1",
		"Type":  "StringList",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutParameter StringList: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mp := decodeJSON(t, wp.Body.String())
	if mp["Version"].(float64) != 1 {
		t.Errorf("PutParameter StringList: expected Version=1, got %v", mp["Version"])
	}

	// GetParameter
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, ssmReq(t, "GetParameter", map[string]any{
		"Name": "/app/config/allowed-regions",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetParameter StringList: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mg := decodeJSON(t, wg.Body.String())
	param := mg["Parameter"].(map[string]any)
	if param["Type"].(string) != "StringList" {
		t.Errorf("GetParameter StringList: expected Type=StringList, got %q", param["Type"])
	}
	if param["Value"].(string) != "us-east-1,us-west-2,eu-west-1" {
		t.Errorf("GetParameter StringList: expected Value=%q, got %q",
			"us-east-1,us-west-2,eu-west-1", param["Value"])
	}
}

// ---- Test 9: GetParameter ParameterNotFound error code ----

func TestSSM_GetParameter_ParameterNotFound(t *testing.T) {
	handler := newSSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "GetParameter", map[string]any{
		"Name": "/nonexistent/param",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("GetParameter nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ParameterNotFound" {
		t.Errorf("GetParameter nonexistent: expected __type=ParameterNotFound, got %q", errType)
	}
}

// ---- Test 10: DeleteParameter ParameterNotFound error code ----

func TestSSM_DeleteParameter_ParameterNotFound(t *testing.T) {
	handler := newSSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "DeleteParameter", map[string]any{
		"Name": "/nonexistent/param",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteParameter nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ParameterNotFound" {
		t.Errorf("DeleteParameter nonexistent: expected __type=ParameterNotFound, got %q", errType)
	}
}

// ---- Test 11: PutParameter ParameterAlreadyExists error code ----

func TestSSM_PutParameter_ParameterAlreadyExists(t *testing.T) {
	handler := newSSMGateway(t)

	// Create first
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, ssmReq(t, "PutParameter", map[string]any{
		"Name":  "/exists/param",
		"Value": "v1",
		"Type":  "String",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutParameter first: %d %s", wp.Code, wp.Body.String())
	}

	// Try again without overwrite
	wp2 := httptest.NewRecorder()
	handler.ServeHTTP(wp2, ssmReq(t, "PutParameter", map[string]any{
		"Name":      "/exists/param",
		"Value":     "v2",
		"Type":      "String",
		"Overwrite": false,
	}))
	if wp2.Code != http.StatusBadRequest {
		t.Fatalf("PutParameter duplicate: expected 400, got %d\nbody: %s", wp2.Code, wp2.Body.String())
	}
	errBody := decodeJSON(t, wp2.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ParameterAlreadyExists" {
		t.Errorf("PutParameter duplicate: expected __type=ParameterAlreadyExists, got %q", errType)
	}
}

// ---- Test 12: GetParametersByPath empty result ----

func TestSSM_GetParametersByPath_Empty(t *testing.T) {
	handler := newSSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ssmReq(t, "GetParametersByPath", map[string]any{
		"Path":      "/empty/path",
		"Recursive": true,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetParametersByPath empty: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	params, _ := m["Parameters"].([]any)
	if len(params) != 0 {
		t.Errorf("GetParametersByPath empty: expected 0 params, got %d", len(params))
	}
}

// ---- Test 13: DescribeParameters returns all parameters ----

func TestSSM_DescribeParameters_Multiple(t *testing.T) {
	handler := newSSMGateway(t)

	// Create several parameters
	names := []string{"/desc/param-a", "/desc/param-b", "/desc/param-c"}
	for _, name := range names {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ssmReq(t, "PutParameter", map[string]any{
			"Name":  name,
			"Value": "value",
			"Type":  "String",
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("PutParameter %s: %d %s", name, w.Code, w.Body.String())
		}
	}

	// DescribeParameters
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ssmReq(t, "DescribeParameters", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeParameters: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	paramList, _ := md["Parameters"].([]any)
	if len(paramList) < 3 {
		t.Errorf("DescribeParameters: expected at least 3 parameters, got %d", len(paramList))
	}

	// Verify no values are exposed
	for _, item := range paramList {
		p := item.(map[string]any)
		if _, hasValue := p["Value"]; hasValue {
			t.Errorf("DescribeParameters: should not expose Value for %s", p["Name"])
		}
	}
}

// ---- Test 14: DeleteParameters batch ----

func TestSSM_DeleteParameters(t *testing.T) {
	handler := newSSMGateway(t)

	// Create two parameters
	for _, name := range []string{"/batch/param1", "/batch/param2"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ssmReq(t, "PutParameter", map[string]any{
			"Name":  name,
			"Value": "value",
			"Type":  "String",
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("PutParameter %s: %d %s", name, w.Code, w.Body.String())
		}
	}

	// DeleteParameters with one valid and one invalid
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ssmReq(t, "DeleteParameters", map[string]any{
		"Names": []string{"/batch/param1", "/batch/param2", "/batch/nonexistent"},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteParameters: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	deleted, _ := md["DeletedParameters"].([]any)
	invalid, _ := md["InvalidParameters"].([]any)

	if len(deleted) != 2 {
		t.Errorf("DeleteParameters: expected 2 deleted, got %d", len(deleted))
	}
	if len(invalid) != 1 {
		t.Errorf("DeleteParameters: expected 1 invalid, got %d", len(invalid))
	}
	if invalid[0].(string) != "/batch/nonexistent" {
		t.Errorf("DeleteParameters: expected invalid=/batch/nonexistent, got %q", invalid[0])
	}
}
