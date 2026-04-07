package secretsmanager_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	smsvc "github.com/Viridian-Inc/cloudmock/services/secretsmanager"
)

// newSMGateway builds a full gateway stack with the Secrets Manager service registered and IAM disabled.
func newSMGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(smsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// smReq builds a JSON POST request targeting the Secrets Manager service via X-Amz-Target.
func smReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("smReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/secretsmanager/aws4_request, SignedHeaders=host, Signature=abc123")
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

// ---- Test 1: CreateSecret + GetSecretValue round-trip ----

func TestSM_CreateAndGetSecret(t *testing.T) {
	handler := newSMGateway(t)

	// CreateSecret
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "my-test-secret",
		"Description":  "A test secret",
		"SecretString": "supersecret",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	mc := decodeJSON(t, wc.Body.String())
	arn, _ := mc["ARN"].(string)
	name, _ := mc["Name"].(string)
	versionId, _ := mc["VersionId"].(string)

	if name != "my-test-secret" {
		t.Errorf("CreateSecret: expected Name=%q, got %q", "my-test-secret", name)
	}
	if !strings.Contains(arn, "my-test-secret") {
		t.Errorf("CreateSecret: ARN %q does not contain secret name", arn)
	}
	if versionId == "" {
		t.Error("CreateSecret: VersionId is empty")
	}

	// GetSecretValue by name
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, smReq(t, "GetSecretValue", map[string]string{
		"SecretId": "my-test-secret",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetSecretValue by name: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	mg := decodeJSON(t, wg.Body.String())
	if mg["Name"].(string) != "my-test-secret" {
		t.Errorf("GetSecretValue: expected Name=%q, got %q", "my-test-secret", mg["Name"])
	}
	if mg["SecretString"].(string) != "supersecret" {
		t.Errorf("GetSecretValue: expected SecretString=%q, got %q", "supersecret", mg["SecretString"])
	}
	if mg["VersionId"].(string) != versionId {
		t.Errorf("GetSecretValue: expected VersionId=%q, got %q", versionId, mg["VersionId"])
	}

	// GetSecretValue by ARN
	wgARN := httptest.NewRecorder()
	handler.ServeHTTP(wgARN, smReq(t, "GetSecretValue", map[string]string{
		"SecretId": arn,
	}))
	if wgARN.Code != http.StatusOK {
		t.Fatalf("GetSecretValue by ARN: expected 200, got %d\nbody: %s", wgARN.Code, wgARN.Body.String())
	}
	mgARN := decodeJSON(t, wgARN.Body.String())
	if mgARN["Name"].(string) != "my-test-secret" {
		t.Errorf("GetSecretValue by ARN: expected Name=%q, got %q", "my-test-secret", mgARN["Name"])
	}
}

// ---- Test 2: PutSecretValue updates version ----

func TestSM_PutSecretValue(t *testing.T) {
	handler := newSMGateway(t)

	// Create a secret first.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "versioned-secret",
		"SecretString": "version-one",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	originalVersionId := mc["VersionId"].(string)

	// PutSecretValue with a new value.
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, smReq(t, "PutSecretValue", map[string]string{
		"SecretId":     "versioned-secret",
		"SecretString": "version-two",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutSecretValue: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}

	mp := decodeJSON(t, wp.Body.String())
	newVersionId := mp["VersionId"].(string)

	if newVersionId == originalVersionId {
		t.Error("PutSecretValue: VersionId should change after putting new value")
	}

	stages, _ := mp["VersionStages"].([]any)
	if len(stages) == 0 || stages[0].(string) != "AWSCURRENT" {
		t.Errorf("PutSecretValue: expected VersionStages=[AWSCURRENT], got %v", stages)
	}

	// Verify new value via GetSecretValue.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, smReq(t, "GetSecretValue", map[string]string{
		"SecretId": "versioned-secret",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetSecretValue after PutSecretValue: %d %s", wg.Code, wg.Body.String())
	}
	mg := decodeJSON(t, wg.Body.String())
	if mg["SecretString"].(string) != "version-two" {
		t.Errorf("GetSecretValue after put: expected %q, got %q", "version-two", mg["SecretString"])
	}
	if mg["VersionId"].(string) != newVersionId {
		t.Errorf("GetSecretValue after put: expected VersionId=%q, got %q", newVersionId, mg["VersionId"])
	}
}

// ---- Test 3: DeleteSecret then GetSecretValue returns ResourceNotFoundException ----

func TestSM_DeleteSecret_ThenGetReturnsNotFound(t *testing.T) {
	handler := newSMGateway(t)

	// Create a secret.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "doomed-secret",
		"SecretString": "bye",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: %d %s", wc.Code, wc.Body.String())
	}

	// Delete it.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, smReq(t, "DeleteSecret", map[string]any{
		"SecretId":                   "doomed-secret",
		"ForceDeleteWithoutRecovery": true,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteSecret: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	md := decodeJSON(t, wd.Body.String())
	if md["Name"].(string) != "doomed-secret" {
		t.Errorf("DeleteSecret: expected Name=%q, got %q", "doomed-secret", md["Name"])
	}
	if _, ok := md["DeletionDate"]; !ok {
		t.Error("DeleteSecret: missing DeletionDate in response")
	}

	// GetSecretValue should now return ResourceNotFoundException (400).
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, smReq(t, "GetSecretValue", map[string]string{
		"SecretId": "doomed-secret",
	}))
	if wg.Code != http.StatusBadRequest {
		t.Fatalf("GetSecretValue after delete: expected 400, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	errBody := decodeJSON(t, wg.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("GetSecretValue after delete: expected __type=ResourceNotFoundException, got %q", errType)
	}
}

// ---- Test 4: ListSecrets shows created secrets ----

func TestSM_ListSecrets(t *testing.T) {
	handler := newSMGateway(t)

	// Create two secrets.
	names := []string{"list-secret-alpha", "list-secret-beta"}
	for _, name := range names {
		wc := httptest.NewRecorder()
		handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
			"Name":         name,
			"SecretString": "value",
		}))
		if wc.Code != http.StatusOK {
			t.Fatalf("CreateSecret %s: %d %s", name, wc.Code, wc.Body.String())
		}
	}

	// ListSecrets.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, smReq(t, "ListSecrets", nil))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListSecrets: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	ml := decodeJSON(t, wl.Body.String())
	secretList, ok := ml["SecretList"].([]any)
	if !ok {
		t.Fatalf("ListSecrets: missing SecretList\nbody: %s", wl.Body.String())
	}
	if len(secretList) < 2 {
		t.Errorf("ListSecrets: expected at least 2 secrets, got %d", len(secretList))
	}

	listed := make(map[string]bool)
	for _, item := range secretList {
		entry := item.(map[string]any)
		listed[entry["Name"].(string)] = true
	}
	for _, name := range names {
		if !listed[name] {
			t.Errorf("ListSecrets: %q not found in list", name)
		}
	}
}

// ---- Test 5: DescribeSecret returns metadata with tags ----

func TestSM_DescribeSecret_WithTags(t *testing.T) {
	handler := newSMGateway(t)

	// Create a secret with tags.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "tagged-secret",
		"Description":  "has tags",
		"SecretString": "value",
		"Tags": []map[string]string{
			{"Key": "Env", "Value": "test"},
			{"Key": "Team", "Value": "platform"},
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	arn := mc["ARN"].(string)

	// DescribeSecret.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, smReq(t, "DescribeSecret", map[string]string{
		"SecretId": "tagged-secret",
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeSecret: expected 200, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}

	mdesc := decodeJSON(t, wdesc.Body.String())
	if mdesc["Name"].(string) != "tagged-secret" {
		t.Errorf("DescribeSecret: expected Name=%q, got %q", "tagged-secret", mdesc["Name"])
	}
	if mdesc["Description"].(string) != "has tags" {
		t.Errorf("DescribeSecret: expected Description=%q, got %q", "has tags", mdesc["Description"])
	}
	if mdesc["ARN"].(string) != arn {
		t.Errorf("DescribeSecret: expected ARN=%q, got %q", arn, mdesc["ARN"])
	}

	// Verify tags.
	tags, ok := mdesc["Tags"].([]any)
	if !ok || len(tags) == 0 {
		t.Fatalf("DescribeSecret: expected Tags in response\nbody: %s", wdesc.Body.String())
	}
	tagMap := make(map[string]string)
	for _, item := range tags {
		t := item.(map[string]any)
		tagMap[t["Key"].(string)] = t["Value"].(string)
	}
	if tagMap["Env"] != "test" {
		t.Errorf("DescribeSecret: expected tag Env=test, got %q", tagMap["Env"])
	}
	if tagMap["Team"] != "platform" {
		t.Errorf("DescribeSecret: expected tag Team=platform, got %q", tagMap["Team"])
	}

	// Verify VersionIdsToStages is present.
	if _, ok := mdesc["VersionIdsToStages"]; !ok {
		t.Error("DescribeSecret: missing VersionIdsToStages in response")
	}
}

// ---- Additional: UpdateSecret ----

func TestSM_UpdateSecret(t *testing.T) {
	handler := newSMGateway(t)

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "updatable-secret",
		"Description":  "original desc",
		"SecretString": "original-value",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	origVersionId := mc["VersionId"].(string)

	// UpdateSecret with new description and value.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, smReq(t, "UpdateSecret", map[string]string{
		"SecretId":     "updatable-secret",
		"Description":  "updated desc",
		"SecretString": "updated-value",
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UpdateSecret: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}
	mu := decodeJSON(t, wu.Body.String())
	if mu["VersionId"].(string) == origVersionId {
		t.Error("UpdateSecret: VersionId should change when SecretString is updated")
	}

	// Verify via DescribeSecret.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, smReq(t, "DescribeSecret", map[string]string{
		"SecretId": "updatable-secret",
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeSecret: %d %s", wdesc.Code, wdesc.Body.String())
	}
	mdesc := decodeJSON(t, wdesc.Body.String())
	if mdesc["Description"].(string) != "updated desc" {
		t.Errorf("UpdateSecret: expected Description=%q, got %q", "updated desc", mdesc["Description"])
	}
}

// ---- Additional: RestoreSecret ----

func TestSM_RestoreSecret(t *testing.T) {
	handler := newSMGateway(t)

	// Create then delete (without force).
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "restore-me",
		"SecretString": "value",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: %d %s", wc.Code, wc.Body.String())
	}

	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, smReq(t, "DeleteSecret", map[string]any{
		"SecretId":                   "restore-me",
		"ForceDeleteWithoutRecovery": false,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteSecret: %d %s", wd.Code, wd.Body.String())
	}

	// Restore it.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, smReq(t, "RestoreSecret", map[string]string{
		"SecretId": "restore-me",
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("RestoreSecret: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}
	mr := decodeJSON(t, wr.Body.String())
	if mr["Name"].(string) != "restore-me" {
		t.Errorf("RestoreSecret: expected Name=%q, got %q", "restore-me", mr["Name"])
	}

	// Should now be accessible via GetSecretValue.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, smReq(t, "GetSecretValue", map[string]string{
		"SecretId": "restore-me",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetSecretValue after restore: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
}

// ---- Additional: TagResource / UntagResource ----

func TestSM_TagAndUntagResource(t *testing.T) {
	handler := newSMGateway(t)

	// Create secret.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "tag-test-secret",
		"SecretString": "value",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret: %d %s", wc.Code, wc.Body.String())
	}

	// TagResource.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, smReq(t, "TagResource", map[string]any{
		"SecretId": "tag-test-secret",
		"Tags": []map[string]string{
			{"Key": "Color", "Value": "blue"},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// Verify tag via DescribeSecret.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, smReq(t, "DescribeSecret", map[string]string{
		"SecretId": "tag-test-secret",
	}))
	mdesc := decodeJSON(t, wdesc.Body.String())
	tags, _ := mdesc["Tags"].([]any)
	tagMap := make(map[string]string)
	for _, item := range tags {
		tag := item.(map[string]any)
		tagMap[tag["Key"].(string)] = tag["Value"].(string)
	}
	if tagMap["Color"] != "blue" {
		t.Errorf("TagResource: expected tag Color=blue, got %q", tagMap["Color"])
	}

	// UntagResource.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, smReq(t, "UntagResource", map[string]any{
		"SecretId": "tag-test-secret",
		"TagKeys":  []string{"Color"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// Verify tag removed.
	wdesc2 := httptest.NewRecorder()
	handler.ServeHTTP(wdesc2, smReq(t, "DescribeSecret", map[string]string{
		"SecretId": "tag-test-secret",
	}))
	mdesc2 := decodeJSON(t, wdesc2.Body.String())
	tags2, _ := mdesc2["Tags"].([]any)
	tagMap2 := make(map[string]string)
	for _, item := range tags2 {
		tag := item.(map[string]any)
		tagMap2[tag["Key"].(string)] = tag["Value"].(string)
	}
	if _, found := tagMap2["Color"]; found {
		t.Error("UntagResource: Color tag should have been removed")
	}
}

// ---- Error: ResourceNotFoundException for GetSecretValue on nonexistent secret ----

func TestSM_GetSecretValue_ResourceNotFoundException(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "GetSecretValue", map[string]string{
		"SecretId": "nonexistent-secret",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("GetSecretValue nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("GetSecretValue nonexistent: expected __type=ResourceNotFoundException, got %q", errType)
	}
}

// ---- Error: ResourceNotFoundException for PutSecretValue on nonexistent secret ----

func TestSM_PutSecretValue_ResourceNotFoundException(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "PutSecretValue", map[string]string{
		"SecretId":     "nonexistent-secret",
		"SecretString": "value",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutSecretValue nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("PutSecretValue nonexistent: expected __type=ResourceNotFoundException, got %q", errType)
	}
}

// ---- Error: ResourceNotFoundException for UpdateSecret on nonexistent secret ----

func TestSM_UpdateSecret_ResourceNotFoundException(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "UpdateSecret", map[string]string{
		"SecretId":     "nonexistent-secret",
		"SecretString": "value",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdateSecret nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("UpdateSecret nonexistent: expected __type=ResourceNotFoundException, got %q", errType)
	}
}

// ---- Error: ResourceNotFoundException for DeleteSecret on nonexistent secret ----

func TestSM_DeleteSecret_ResourceNotFoundException(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "DeleteSecret", map[string]any{
		"SecretId":                   "nonexistent-secret",
		"ForceDeleteWithoutRecovery": true,
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteSecret nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("DeleteSecret nonexistent: expected __type=ResourceNotFoundException, got %q", errType)
	}
}

// ---- Error: ResourceExistsException for CreateSecret duplicate ----

func TestSM_CreateSecret_ResourceExistsException(t *testing.T) {
	handler := newSMGateway(t)

	// Create first
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, smReq(t, "CreateSecret", map[string]any{
		"Name":         "duplicate-secret",
		"SecretString": "value",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSecret first: %d %s", wc.Code, wc.Body.String())
	}

	// Try again
	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, smReq(t, "CreateSecret", map[string]any{
		"Name":         "duplicate-secret",
		"SecretString": "value2",
	}))
	if wc2.Code != http.StatusBadRequest {
		t.Fatalf("CreateSecret duplicate: expected 400, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}
	errBody := decodeJSON(t, wc2.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceExistsException" {
		t.Errorf("CreateSecret duplicate: expected __type=ResourceExistsException, got %q", errType)
	}
}

// ---- Error: ResourceNotFoundException for DescribeSecret on nonexistent secret ----

func TestSM_DescribeSecret_ResourceNotFoundException(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "DescribeSecret", map[string]string{
		"SecretId": "nonexistent-secret",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DescribeSecret nonexistent: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	errBody := decodeJSON(t, w.Body.String())
	errType, _ := errBody["__type"].(string)
	if errType != "ResourceNotFoundException" {
		t.Errorf("DescribeSecret nonexistent: expected __type=ResourceNotFoundException, got %q", errType)
	}
}

// ---- Positive: ListSecrets empty ----

func TestSM_ListSecrets_Empty(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "ListSecrets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSecrets empty: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	secretList, _ := m["SecretList"].([]any)
	if len(secretList) != 0 {
		t.Errorf("ListSecrets empty: expected 0 secrets, got %d", len(secretList))
	}
}

// ---- Additional: UnknownAction ----

func TestSM_UnknownAction(t *testing.T) {
	handler := newSMGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, smReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
