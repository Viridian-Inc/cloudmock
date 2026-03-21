package stepfunctions_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	sfnsvc "github.com/neureaux/cloudmock/services/stepfunctions"
)

// newSFNGateway builds a full gateway stack with the Step Functions service registered and IAM disabled.
func newSFNGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(sfnsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// sfnReq builds a JSON POST request targeting the Step Functions service via X-Amz-Target.
func sfnReq(t *testing.T, action string, body interface{}) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("sfnReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSStepFunctions."+action)
	// Authorization header places "states" as the service in the credential scope
	// so the gateway router can detect "states" as the target service.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/states/aws4_request, SignedHeaders=host, Signature=abc123")
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

// minimalDefinition is a valid (for mocking purposes) ASL JSON definition.
const minimalDefinition = `{"Comment":"test","StartAt":"Pass","States":{"Pass":{"Type":"Pass","End":true}}}`

// ---- Test 1: CreateStateMachine + DescribeStateMachine + ListStateMachines ----

func TestSFN_CreateDescribeList(t *testing.T) {
	handler := newSFNGateway(t)

	// CreateStateMachine
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "my-state-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
		"type":       "STANDARD",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("CreateStateMachine: expected 200, got %d\nbody: %s", wCreate.Code, wCreate.Body.String())
	}

	mCreate := decodeJSON(t, wCreate.Body.String())
	smArn, _ := mCreate["stateMachineArn"].(string)
	if smArn == "" {
		t.Fatal("CreateStateMachine: missing stateMachineArn in response")
	}
	if !strings.Contains(smArn, "my-state-machine") {
		t.Errorf("CreateStateMachine: ARN %q does not contain state machine name", smArn)
	}
	if _, ok := mCreate["creationDate"]; !ok {
		t.Error("CreateStateMachine: missing creationDate in response")
	}

	// DescribeStateMachine
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, sfnReq(t, "DescribeStateMachine", map[string]string{
		"stateMachineArn": smArn,
	}))
	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeStateMachine: expected 200, got %d\nbody: %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	if mDesc["name"] != "my-state-machine" {
		t.Errorf("DescribeStateMachine: expected name=my-state-machine, got %q", mDesc["name"])
	}
	if mDesc["definition"] != minimalDefinition {
		t.Errorf("DescribeStateMachine: definition mismatch")
	}
	if mDesc["status"] != "ACTIVE" {
		t.Errorf("DescribeStateMachine: expected status=ACTIVE, got %q", mDesc["status"])
	}
	if mDesc["type"] != "STANDARD" {
		t.Errorf("DescribeStateMachine: expected type=STANDARD, got %q", mDesc["type"])
	}

	// ListStateMachines
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, sfnReq(t, "ListStateMachines", nil))
	if wList.Code != http.StatusOK {
		t.Fatalf("ListStateMachines: expected 200, got %d\nbody: %s", wList.Code, wList.Body.String())
	}

	mList := decodeJSON(t, wList.Body.String())
	machines, ok := mList["stateMachines"].([]interface{})
	if !ok {
		t.Fatalf("ListStateMachines: missing stateMachines array\nbody: %s", wList.Body.String())
	}
	found := false
	for _, m := range machines {
		entry := m.(map[string]interface{})
		if entry["stateMachineArn"] == smArn {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListStateMachines: ARN %q not found in list", smArn)
	}
}

// ---- Test 2: StartExecution + DescribeExecution ----

func TestSFN_StartAndDescribeExecution(t *testing.T) {
	handler := newSFNGateway(t)

	// Create a state machine first.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "exec-test-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// StartExecution
	inputJSON := `{"hello":"world"}`
	wStart := httptest.NewRecorder()
	handler.ServeHTTP(wStart, sfnReq(t, "StartExecution", map[string]string{
		"stateMachineArn": smArn,
		"name":            "my-execution",
		"input":           inputJSON,
	}))
	if wStart.Code != http.StatusOK {
		t.Fatalf("StartExecution: expected 200, got %d\nbody: %s", wStart.Code, wStart.Body.String())
	}

	mStart := decodeJSON(t, wStart.Body.String())
	execArn, _ := mStart["executionArn"].(string)
	if execArn == "" {
		t.Fatal("StartExecution: missing executionArn in response")
	}
	if !strings.Contains(execArn, "my-execution") {
		t.Errorf("StartExecution: ARN %q does not contain execution name", execArn)
	}
	if _, ok := mStart["startDate"]; !ok {
		t.Error("StartExecution: missing startDate in response")
	}

	// DescribeExecution
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, sfnReq(t, "DescribeExecution", map[string]string{
		"executionArn": execArn,
	}))
	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeExecution: expected 200, got %d\nbody: %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	if mDesc["status"] != "SUCCEEDED" {
		t.Errorf("DescribeExecution: expected status=SUCCEEDED, got %q", mDesc["status"])
	}
	if mDesc["input"] != inputJSON {
		t.Errorf("DescribeExecution: expected input=%q, got %q", inputJSON, mDesc["input"])
	}
	if mDesc["output"] != inputJSON {
		t.Errorf("DescribeExecution: expected output=%q (pass-through), got %q", inputJSON, mDesc["output"])
	}
	if mDesc["stateMachineArn"] != smArn {
		t.Errorf("DescribeExecution: expected stateMachineArn=%q, got %q", smArn, mDesc["stateMachineArn"])
	}
}

// ---- Test 3: ListExecutions ----

func TestSFN_ListExecutions(t *testing.T) {
	handler := newSFNGateway(t)

	// Create state machine.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "list-exec-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// Start two executions.
	execNames := []string{"exec-one", "exec-two"}
	var execArns []string
	for _, name := range execNames {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, sfnReq(t, "StartExecution", map[string]string{
			"stateMachineArn": smArn,
			"name":            name,
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("StartExecution %s: %d %s", name, w.Code, w.Body.String())
		}
		execArns = append(execArns, decodeJSON(t, w.Body.String())["executionArn"].(string))
	}

	// ListExecutions
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, sfnReq(t, "ListExecutions", map[string]string{
		"stateMachineArn": smArn,
	}))
	if wList.Code != http.StatusOK {
		t.Fatalf("ListExecutions: expected 200, got %d\nbody: %s", wList.Code, wList.Body.String())
	}

	mList := decodeJSON(t, wList.Body.String())
	execs, ok := mList["executions"].([]interface{})
	if !ok {
		t.Fatalf("ListExecutions: missing executions array\nbody: %s", wList.Body.String())
	}
	if len(execs) < 2 {
		t.Errorf("ListExecutions: expected at least 2, got %d", len(execs))
	}

	listed := make(map[string]bool)
	for _, e := range execs {
		entry := e.(map[string]interface{})
		listed[entry["executionArn"].(string)] = true
	}
	for _, arn := range execArns {
		if !listed[arn] {
			t.Errorf("ListExecutions: executionArn %q not found in list", arn)
		}
	}
}

// ---- Test 4: StopExecution ----

func TestSFN_StopExecution(t *testing.T) {
	handler := newSFNGateway(t)

	// Create state machine.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "stop-exec-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// Start execution.
	wStart := httptest.NewRecorder()
	handler.ServeHTTP(wStart, sfnReq(t, "StartExecution", map[string]string{
		"stateMachineArn": smArn,
		"name":            "to-be-stopped",
	}))
	if wStart.Code != http.StatusOK {
		t.Fatalf("StartExecution: %d %s", wStart.Code, wStart.Body.String())
	}
	execArn := decodeJSON(t, wStart.Body.String())["executionArn"].(string)

	// StopExecution
	wStop := httptest.NewRecorder()
	handler.ServeHTTP(wStop, sfnReq(t, "StopExecution", map[string]string{
		"executionArn": execArn,
		"cause":        "manual stop",
		"error":        "States.ManualStop",
	}))
	if wStop.Code != http.StatusOK {
		t.Fatalf("StopExecution: expected 200, got %d\nbody: %s", wStop.Code, wStop.Body.String())
	}

	mStop := decodeJSON(t, wStop.Body.String())
	if _, ok := mStop["stopDate"]; !ok {
		t.Error("StopExecution: missing stopDate in response")
	}

	// Verify execution is ABORTED via DescribeExecution.
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, sfnReq(t, "DescribeExecution", map[string]string{
		"executionArn": execArn,
	}))
	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeExecution after stop: %d %s", wDesc.Code, wDesc.Body.String())
	}
	mDesc := decodeJSON(t, wDesc.Body.String())
	if mDesc["status"] != "ABORTED" {
		t.Errorf("StopExecution: expected status=ABORTED, got %q", mDesc["status"])
	}
}

// ---- Test 5: GetExecutionHistory ----

func TestSFN_GetExecutionHistory(t *testing.T) {
	handler := newSFNGateway(t)

	// Create state machine.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "history-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// Start execution.
	wStart := httptest.NewRecorder()
	handler.ServeHTTP(wStart, sfnReq(t, "StartExecution", map[string]string{
		"stateMachineArn": smArn,
		"name":            "history-exec",
	}))
	if wStart.Code != http.StatusOK {
		t.Fatalf("StartExecution: %d %s", wStart.Code, wStart.Body.String())
	}
	execArn := decodeJSON(t, wStart.Body.String())["executionArn"].(string)

	// GetExecutionHistory
	wHist := httptest.NewRecorder()
	handler.ServeHTTP(wHist, sfnReq(t, "GetExecutionHistory", map[string]string{
		"executionArn": execArn,
	}))
	if wHist.Code != http.StatusOK {
		t.Fatalf("GetExecutionHistory: expected 200, got %d\nbody: %s", wHist.Code, wHist.Body.String())
	}

	mHist := decodeJSON(t, wHist.Body.String())
	events, ok := mHist["events"].([]interface{})
	if !ok {
		t.Fatalf("GetExecutionHistory: missing events array\nbody: %s", wHist.Body.String())
	}
	if len(events) < 2 {
		t.Errorf("GetExecutionHistory: expected at least 2 events, got %d", len(events))
	}

	// Check that the first event is ExecutionStarted.
	firstEvent := events[0].(map[string]interface{})
	if firstEvent["type"] != "ExecutionStarted" {
		t.Errorf("GetExecutionHistory: expected first event type=ExecutionStarted, got %q", firstEvent["type"])
	}

	// Check that there's an ExecutionSucceeded event.
	foundSucceeded := false
	for _, e := range events {
		entry := e.(map[string]interface{})
		if entry["type"] == "ExecutionSucceeded" {
			foundSucceeded = true
			break
		}
	}
	if !foundSucceeded {
		t.Error("GetExecutionHistory: no ExecutionSucceeded event found")
	}
}

// ---- Test 6: DeleteStateMachine ----

func TestSFN_DeleteStateMachine(t *testing.T) {
	handler := newSFNGateway(t)

	// Create state machine.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "to-delete-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// DeleteStateMachine
	wDel := httptest.NewRecorder()
	handler.ServeHTTP(wDel, sfnReq(t, "DeleteStateMachine", map[string]string{
		"stateMachineArn": smArn,
	}))
	if wDel.Code != http.StatusOK {
		t.Fatalf("DeleteStateMachine: expected 200, got %d\nbody: %s", wDel.Code, wDel.Body.String())
	}

	// DescribeStateMachine should return an error now.
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, sfnReq(t, "DescribeStateMachine", map[string]string{
		"stateMachineArn": smArn,
	}))
	if wDesc.Code == http.StatusOK {
		t.Fatal("DescribeStateMachine after delete: expected non-200, got 200")
	}
}

// ---- Test 7: UpdateStateMachine ----

func TestSFN_UpdateStateMachine(t *testing.T) {
	handler := newSFNGateway(t)

	// Create state machine.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "updatable-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/OldRole",
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// UpdateStateMachine with a new definition and roleArn.
	newDef := `{"Comment":"updated","StartAt":"Pass","States":{"Pass":{"Type":"Pass","End":true}}}`
	wUpdate := httptest.NewRecorder()
	handler.ServeHTTP(wUpdate, sfnReq(t, "UpdateStateMachine", map[string]string{
		"stateMachineArn": smArn,
		"definition":      newDef,
		"roleArn":         "arn:aws:iam::123456789012:role/NewRole",
	}))
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("UpdateStateMachine: expected 200, got %d\nbody: %s", wUpdate.Code, wUpdate.Body.String())
	}

	mUpdate := decodeJSON(t, wUpdate.Body.String())
	if _, ok := mUpdate["updateDate"]; !ok {
		t.Error("UpdateStateMachine: missing updateDate in response")
	}

	// DescribeStateMachine to verify updates.
	wDesc := httptest.NewRecorder()
	handler.ServeHTTP(wDesc, sfnReq(t, "DescribeStateMachine", map[string]string{
		"stateMachineArn": smArn,
	}))
	if wDesc.Code != http.StatusOK {
		t.Fatalf("DescribeStateMachine after update: %d %s", wDesc.Code, wDesc.Body.String())
	}

	mDesc := decodeJSON(t, wDesc.Body.String())
	if mDesc["definition"] != newDef {
		t.Errorf("UpdateStateMachine: expected definition updated, got %q", mDesc["definition"])
	}
	if mDesc["roleArn"] != "arn:aws:iam::123456789012:role/NewRole" {
		t.Errorf("UpdateStateMachine: expected roleArn=NewRole, got %q", mDesc["roleArn"])
	}
}

// ---- Tagging tests ----

func TestSFN_TaggingRoundTrip(t *testing.T) {
	handler := newSFNGateway(t)

	// Create state machine.
	wCreate := httptest.NewRecorder()
	handler.ServeHTTP(wCreate, sfnReq(t, "CreateStateMachine", map[string]interface{}{
		"name":       "tagging-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
		"tags": []map[string]string{
			{"key": "env", "value": "test"},
		},
	}))
	if wCreate.Code != http.StatusOK {
		t.Fatalf("setup CreateStateMachine: %d %s", wCreate.Code, wCreate.Body.String())
	}
	smArn := decodeJSON(t, wCreate.Body.String())["stateMachineArn"].(string)

	// TagResource
	wTag := httptest.NewRecorder()
	handler.ServeHTTP(wTag, sfnReq(t, "TagResource", map[string]interface{}{
		"resourceArn": smArn,
		"tags": []map[string]string{
			{"key": "owner", "value": "alice"},
		},
	}))
	if wTag.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wTag.Code, wTag.Body.String())
	}

	// ListTagsForResource
	wList := httptest.NewRecorder()
	handler.ServeHTTP(wList, sfnReq(t, "ListTagsForResource", map[string]string{
		"resourceArn": smArn,
	}))
	if wList.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wList.Code, wList.Body.String())
	}

	mList := decodeJSON(t, wList.Body.String())
	tags, ok := mList["tags"].([]interface{})
	if !ok {
		t.Fatalf("ListTagsForResource: missing tags array\nbody: %s", wList.Body.String())
	}

	tagMap := make(map[string]string)
	for _, tagRaw := range tags {
		tag := tagRaw.(map[string]interface{})
		tagMap[tag["key"].(string)] = tag["value"].(string)
	}
	if tagMap["owner"] != "alice" {
		t.Errorf("ListTagsForResource: expected owner=alice, got %q", tagMap["owner"])
	}
	if tagMap["env"] != "test" {
		t.Errorf("ListTagsForResource: expected env=test, got %q", tagMap["env"])
	}

	// UntagResource
	wUntag := httptest.NewRecorder()
	handler.ServeHTTP(wUntag, sfnReq(t, "UntagResource", map[string]interface{}{
		"resourceArn": smArn,
		"tagKeys":     []string{"owner"},
	}))
	if wUntag.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wUntag.Code, wUntag.Body.String())
	}

	// ListTagsForResource again — "owner" should be gone.
	wList2 := httptest.NewRecorder()
	handler.ServeHTTP(wList2, sfnReq(t, "ListTagsForResource", map[string]string{
		"resourceArn": smArn,
	}))
	mList2 := decodeJSON(t, wList2.Body.String())
	tags2 := mList2["tags"].([]interface{})
	for _, tagRaw := range tags2 {
		tag := tagRaw.(map[string]interface{})
		if tag["key"] == "owner" {
			t.Error("UntagResource: tag 'owner' still present after untagging")
		}
	}
}

// ---- Error cases ----

func TestSFN_CreateStateMachine_AlreadyExists(t *testing.T) {
	handler := newSFNGateway(t)

	body := map[string]interface{}{
		"name":       "duplicate-machine",
		"definition": minimalDefinition,
		"roleArn":    "arn:aws:iam::123456789012:role/StepFunctionsRole",
	}

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, sfnReq(t, "CreateStateMachine", body))
	if w1.Code != http.StatusOK {
		t.Fatalf("first CreateStateMachine: %d %s", w1.Code, w1.Body.String())
	}

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, sfnReq(t, "CreateStateMachine", body))
	if w2.Code != http.StatusConflict {
		t.Fatalf("duplicate CreateStateMachine: expected 409, got %d\nbody: %s", w2.Code, w2.Body.String())
	}
}

func TestSFN_DescribeStateMachine_NotFound(t *testing.T) {
	handler := newSFNGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sfnReq(t, "DescribeStateMachine", map[string]string{
		"stateMachineArn": "arn:aws:states:us-east-1:123456789012:stateMachine:does-not-exist",
	}))
	if w.Code == http.StatusOK {
		t.Fatalf("DescribeStateMachine not found: expected non-200, got 200")
	}
}

func TestSFN_UnknownAction(t *testing.T) {
	handler := newSFNGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sfnReq(t, "UnknownAction", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
