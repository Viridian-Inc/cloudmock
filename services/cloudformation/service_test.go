package cloudformation_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	cfnsvc "github.com/neureaux/cloudmock/services/cloudformation"
)

// ---- helpers ----

func newCFNGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(cfnsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

func cfnReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2010-05-15")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cloudformation/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// sampleTemplate is a minimal but valid CloudFormation JSON template used in tests.
const sampleTemplate = `{
  "Description": "Test stack for cloudmock",
  "Parameters": {
    "EnvName": {
      "Type": "String",
      "Default": "dev"
    }
  },
  "Resources": {
    "MyBucket": {
      "Type": "AWS::S3::Bucket"
    },
    "MyQueue": {
      "Type": "AWS::SQS::Queue"
    }
  },
  "Outputs": {
    "BucketName": {
      "Value": "my-bucket",
      "Description": "The S3 bucket name",
      "Export": {
        "Name": "MyExportedBucket"
      }
    }
  }
}`

// mustCreateStack creates a stack and returns its StackId ARN.
func mustCreateStack(t *testing.T, handler http.Handler, name string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "CreateStack", url.Values{
		"StackName":    {name},
		"TemplateBody": {sampleTemplate},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateStack %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			StackId string `xml:"StackId"`
		} `xml:"CreateStackResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateStack %s: unmarshal: %v\nbody: %s", name, err, w.Body.String())
	}
	if resp.Result.StackId == "" {
		t.Fatalf("CreateStack %s: StackId is empty\nbody: %s", name, w.Body.String())
	}
	return resp.Result.StackId
}

// ---- Test 1: CreateStack + DescribeStacks ----

func TestCFN_CreateAndDescribeStacks(t *testing.T) {
	handler := newCFNGateway(t)

	stackID := mustCreateStack(t, handler, "my-test-stack")

	if !strings.Contains(stackID, "arn:aws:cloudformation:") {
		t.Errorf("CreateStack: expected ARN stackId, got %s", stackID)
	}
	if !strings.Contains(stackID, "my-test-stack") {
		t.Errorf("CreateStack: ARN should contain stack name, got %s", stackID)
	}

	// DescribeStacks by name.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DescribeStacks", url.Values{"StackName": {"my-test-stack"}}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStacks: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"my-test-stack",
		"CREATE_COMPLETE",
		"Test stack for cloudmock",
		"DescribeStacksResponse",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("DescribeStacks: expected %q in response\nbody: %s", want, body)
		}
	}

	// DescribeStacks — all stacks (no name filter).
	wall := httptest.NewRecorder()
	handler.ServeHTTP(wall, cfnReq(t, "DescribeStacks", nil))
	if wall.Code != http.StatusOK {
		t.Fatalf("DescribeStacks all: expected 200, got %d\nbody: %s", wall.Code, wall.Body.String())
	}
	if !strings.Contains(wall.Body.String(), "my-test-stack") {
		t.Errorf("DescribeStacks all: expected my-test-stack\nbody: %s", wall.Body.String())
	}
}

// ---- Test 2: DescribeStackResources ----

func TestCFN_DescribeStackResources(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "resource-stack")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DescribeStackResources", url.Values{
		"StackName": {"resource-stack"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// Template has two resources: MyBucket and MyQueue.
	for _, want := range []string{
		"DescribeStackResourcesResponse",
		"MyBucket",
		"AWS::S3::Bucket",
		"MyQueue",
		"AWS::SQS::Queue",
		"CREATE_COMPLETE",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("DescribeStackResources: expected %q in response\nbody: %s", want, body)
		}
	}
}

// ---- Test 3: GetTemplate ----

func TestCFN_GetTemplate(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "template-stack")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "GetTemplate", url.Values{
		"StackName": {"template-stack"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "GetTemplateResponse") {
		t.Errorf("GetTemplate: expected GetTemplateResponse\nbody: %s", body)
	}
	// Original template body should be present (XML-encoded).
	if !strings.Contains(body, "AWS::S3::Bucket") {
		t.Errorf("GetTemplate: expected original template content in response\nbody: %s", body)
	}
}

// ---- Test 4: ValidateTemplate ----

func TestCFN_ValidateTemplate(t *testing.T) {
	handler := newCFNGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "ValidateTemplate", url.Values{
		"TemplateBody": {sampleTemplate},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ValidateTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"ValidateTemplateResponse",
		"Test stack for cloudmock",
		"EnvName",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("ValidateTemplate: expected %q in response\nbody: %s", want, body)
		}
	}

	// ValidateTemplate with invalid (non-JSON) template — should still return 200
	// for a v1 implementation that does best-effort parsing.
	winvalid := httptest.NewRecorder()
	handler.ServeHTTP(winvalid, cfnReq(t, "ValidateTemplate", url.Values{
		"TemplateBody": {"not-json"},
	}))
	if winvalid.Code != http.StatusOK {
		t.Fatalf("ValidateTemplate invalid: expected 200, got %d\nbody: %s", winvalid.Code, winvalid.Body.String())
	}
}

// ---- Test 5: DeleteStack + verify status ----

func TestCFN_DeleteStack(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "delete-me")

	// Verify it's visible before deletion.
	wbefore := httptest.NewRecorder()
	handler.ServeHTTP(wbefore, cfnReq(t, "DescribeStacks", url.Values{"StackName": {"delete-me"}}))
	if !strings.Contains(wbefore.Body.String(), "CREATE_COMPLETE") {
		t.Fatal("DescribeStacks before delete: expected CREATE_COMPLETE")
	}

	// DeleteStack.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, cfnReq(t, "DeleteStack", url.Values{"StackName": {"delete-me"}}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteStack: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}
	if !strings.Contains(wdel.Body.String(), "DeleteStackResponse") {
		t.Errorf("DeleteStack: expected DeleteStackResponse\nbody: %s", wdel.Body.String())
	}

	// DescribeStacks after delete — should show DELETE_COMPLETE.
	wafter := httptest.NewRecorder()
	handler.ServeHTTP(wafter, cfnReq(t, "DescribeStacks", url.Values{"StackName": {"delete-me"}}))
	if wafter.Code != http.StatusOK {
		t.Fatalf("DescribeStacks after delete: expected 200, got %d\nbody: %s", wafter.Code, wafter.Body.String())
	}
	if !strings.Contains(wafter.Body.String(), "DELETE_COMPLETE") {
		t.Errorf("DescribeStacks after delete: expected DELETE_COMPLETE\nbody: %s", wafter.Body.String())
	}

	// ListStacks — deleted stack should NOT appear when no filter is given
	// (default listing omits DELETE_COMPLETE for DescribeStacks but ListStacks
	// shows all by default — verify it appears with DELETE_COMPLETE filter).
	wlist := httptest.NewRecorder()
	handler.ServeHTTP(wlist, cfnReq(t, "ListStacks", url.Values{
		"StackStatusFilter.member.1": {"DELETE_COMPLETE"},
	}))
	if wlist.Code != http.StatusOK {
		t.Fatalf("ListStacks filter: expected 200, got %d\nbody: %s", wlist.Code, wlist.Body.String())
	}
	if !strings.Contains(wlist.Body.String(), "delete-me") {
		t.Errorf("ListStacks DELETE_COMPLETE: expected delete-me\nbody: %s", wlist.Body.String())
	}
}

// ---- Test 6: ListStacks with status filter ----

func TestCFN_ListStacksWithFilter(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "live-stack")
	mustCreateStack(t, handler, "to-delete-stack")

	// Delete the second stack.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, cfnReq(t, "DeleteStack", url.Values{"StackName": {"to-delete-stack"}}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteStack: %d %s", wdel.Code, wdel.Body.String())
	}

	// ListStacks with CREATE_COMPLETE filter — should only include live-stack.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "ListStacks", url.Values{
		"StackStatusFilter.member.1": {"CREATE_COMPLETE"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListStacks CREATE_COMPLETE: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "live-stack") {
		t.Errorf("ListStacks CREATE_COMPLETE: expected live-stack\nbody: %s", body)
	}
	if strings.Contains(body, "to-delete-stack") {
		t.Errorf("ListStacks CREATE_COMPLETE: should NOT contain to-delete-stack\nbody: %s", body)
	}

	// ListStacks with no filter — all stacks.
	wall := httptest.NewRecorder()
	handler.ServeHTTP(wall, cfnReq(t, "ListStacks", nil))
	if wall.Code != http.StatusOK {
		t.Fatalf("ListStacks no filter: expected 200, got %d\nbody: %s", wall.Code, wall.Body.String())
	}
	wallBody := wall.Body.String()
	if !strings.Contains(wallBody, "live-stack") {
		t.Errorf("ListStacks no filter: expected live-stack\nbody: %s", wallBody)
	}
	if !strings.Contains(wallBody, "to-delete-stack") {
		t.Errorf("ListStacks no filter: expected to-delete-stack\nbody: %s", wallBody)
	}
}

// ---- Test 7: CreateChangeSet + DescribeChangeSet ----

func TestCFN_ChangeSet(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "changeset-stack")

	// CreateChangeSet.
	wcs := httptest.NewRecorder()
	handler.ServeHTTP(wcs, cfnReq(t, "CreateChangeSet", url.Values{
		"StackName":     {"changeset-stack"},
		"ChangeSetName": {"my-change-set"},
		"Description":   {"add a queue"},
	}))
	if wcs.Code != http.StatusOK {
		t.Fatalf("CreateChangeSet: expected 200, got %d\nbody: %s", wcs.Code, wcs.Body.String())
	}

	csBody := wcs.Body.String()
	if !strings.Contains(csBody, "CreateChangeSetResponse") {
		t.Errorf("CreateChangeSet: expected CreateChangeSetResponse\nbody: %s", csBody)
	}
	if !strings.Contains(csBody, "arn:aws:cloudformation:") {
		t.Errorf("CreateChangeSet: expected ARN in response\nbody: %s", csBody)
	}

	// DescribeChangeSet.
	wdcs := httptest.NewRecorder()
	handler.ServeHTTP(wdcs, cfnReq(t, "DescribeChangeSet", url.Values{
		"StackName":     {"changeset-stack"},
		"ChangeSetName": {"my-change-set"},
	}))
	if wdcs.Code != http.StatusOK {
		t.Fatalf("DescribeChangeSet: expected 200, got %d\nbody: %s", wdcs.Code, wdcs.Body.String())
	}

	dcsBody := wdcs.Body.String()
	for _, want := range []string{
		"DescribeChangeSetResponse",
		"my-change-set",
		"changeset-stack",
		"CREATE_COMPLETE",
		"AVAILABLE",
		"add a queue",
	} {
		if !strings.Contains(dcsBody, want) {
			t.Errorf("DescribeChangeSet: expected %q in response\nbody: %s", want, dcsBody)
		}
	}

	// ExecuteChangeSet.
	wexec := httptest.NewRecorder()
	handler.ServeHTTP(wexec, cfnReq(t, "ExecuteChangeSet", url.Values{
		"StackName":     {"changeset-stack"},
		"ChangeSetName": {"my-change-set"},
	}))
	if wexec.Code != http.StatusOK {
		t.Fatalf("ExecuteChangeSet: expected 200, got %d\nbody: %s", wexec.Code, wexec.Body.String())
	}

	// DescribeChangeSet after execute — ExecutionStatus should be EXECUTE_COMPLETE.
	wdcs2 := httptest.NewRecorder()
	handler.ServeHTTP(wdcs2, cfnReq(t, "DescribeChangeSet", url.Values{
		"StackName":     {"changeset-stack"},
		"ChangeSetName": {"my-change-set"},
	}))
	if !strings.Contains(wdcs2.Body.String(), "EXECUTE_COMPLETE") {
		t.Errorf("DescribeChangeSet after execute: expected EXECUTE_COMPLETE\nbody: %s", wdcs2.Body.String())
	}

	// DeleteChangeSet.
	wdelcs := httptest.NewRecorder()
	handler.ServeHTTP(wdelcs, cfnReq(t, "DeleteChangeSet", url.Values{
		"StackName":     {"changeset-stack"},
		"ChangeSetName": {"my-change-set"},
	}))
	if wdelcs.Code != http.StatusOK {
		t.Fatalf("DeleteChangeSet: expected 200, got %d\nbody: %s", wdelcs.Code, wdelcs.Body.String())
	}

	// DescribeChangeSet after delete — should be not found.
	wdcs3 := httptest.NewRecorder()
	handler.ServeHTTP(wdcs3, cfnReq(t, "DescribeChangeSet", url.Values{
		"StackName":     {"changeset-stack"},
		"ChangeSetName": {"my-change-set"},
	}))
	if wdcs3.Code == http.StatusOK {
		t.Errorf("DescribeChangeSet after delete: expected error, got 200\nbody: %s", wdcs3.Body.String())
	}
}

// ---- Test 8: ListExports ----

func TestCFN_ListExports(t *testing.T) {
	handler := newCFNGateway(t)
	// sampleTemplate has an Output with ExportName "MyExportedBucket".
	mustCreateStack(t, handler, "export-stack")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "ListExports", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListExports: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "ListExportsResponse") {
		t.Errorf("ListExports: expected ListExportsResponse\nbody: %s", body)
	}
	if !strings.Contains(body, "MyExportedBucket") {
		t.Errorf("ListExports: expected MyExportedBucket export\nbody: %s", body)
	}
}

// ---- Test 9: CreateStack AlreadyExistsException ----

func TestCFN_CreateStack_AlreadyExists(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "exists-stack")

	// Try to create again
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "CreateStack", url.Values{
		"StackName":    {"exists-stack"},
		"TemplateBody": {sampleTemplate},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateStack duplicate: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "AlreadyExistsException") {
		t.Errorf("CreateStack duplicate: expected AlreadyExistsException\nbody: %s", body)
	}
}

// ---- Test 10: DescribeStacks nonexistent shows no stacks or error ----

func TestCFN_DescribeStacks_NonExistent(t *testing.T) {
	handler := newCFNGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DescribeStacks", url.Values{
		"StackName": {"nonexistent-stack"},
	}))
	// CloudFormation returns 400 for nonexistent stacks
	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		t.Fatalf("DescribeStacks nonexistent: unexpected status %d\nbody: %s", w.Code, w.Body.String())
	}
	// If 200, there should be no stack members in the response
	if w.Code == http.StatusOK {
		body := w.Body.String()
		if strings.Contains(body, "nonexistent-stack") {
			t.Errorf("DescribeStacks nonexistent: should not find nonexistent-stack\nbody: %s", body)
		}
	}
}

// ---- Test 11: DescribeStackResources for stack with template resources ----

func TestCFN_DescribeStackResources_Detailed(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "detailed-resource-stack")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DescribeStackResources", url.Values{
		"StackName": {"detailed-resource-stack"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStackResources: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// Verify resource status
	if !strings.Contains(body, "CREATE_COMPLETE") {
		t.Errorf("DescribeStackResources: expected CREATE_COMPLETE\nbody: %s", body)
	}

	// Parse and count resources (template has MyBucket + MyQueue)
	var resp struct {
		Result struct {
			Resources []struct {
				LogicalResourceId string `xml:"LogicalResourceId"`
				ResourceType      string `xml:"ResourceType"`
				ResourceStatus    string `xml:"ResourceStatus"`
			} `xml:"StackResources>member"`
		} `xml:"DescribeStackResourcesResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Result.Resources) != 2 {
		t.Errorf("DescribeStackResources: expected 2 resources, got %d", len(resp.Result.Resources))
	}
}

// ---- Test 12: DescribeStackEvents returns creation events ----

func TestCFN_DescribeStackEvents_Detailed(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "events-detail-stack")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DescribeStackEvents", url.Values{
		"StackName": {"events-detail-stack"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStackEvents: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// Verify the stack event contains resource types
	if !strings.Contains(body, "AWS::CloudFormation::Stack") {
		t.Errorf("DescribeStackEvents: expected AWS::CloudFormation::Stack\nbody: %s", body)
	}
	// Should have events for the resources too
	if !strings.Contains(body, "AWS::S3::Bucket") && !strings.Contains(body, "AWS::SQS::Queue") {
		t.Errorf("DescribeStackEvents: expected resource events\nbody: %s", body)
	}
}

// ---- Test 13: ValidateTemplate with parameters ----

func TestCFN_ValidateTemplate_Parameters(t *testing.T) {
	handler := newCFNGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "ValidateTemplate", url.Values{
		"TemplateBody": {sampleTemplate},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ValidateTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// Should include EnvName parameter
	if !strings.Contains(body, "EnvName") {
		t.Errorf("ValidateTemplate: expected EnvName parameter\nbody: %s", body)
	}
	// Should include description
	if !strings.Contains(body, "Test stack for cloudmock") {
		t.Errorf("ValidateTemplate: expected description\nbody: %s", body)
	}
}

// ---- Test 14: DeleteStack idempotent (delete nonexistent stack) ----

func TestCFN_DeleteStack_NonExistent(t *testing.T) {
	handler := newCFNGateway(t)

	// Deleting nonexistent stack should still return 200 (CloudFormation behavior)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DeleteStack", url.Values{
		"StackName": {"nonexistent-stack"},
	}))
	// CloudFormation returns 200 for deleting nonexistent stacks
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteStack nonexistent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Original Test 9: DescribeStackEvents ----

func TestCFN_DescribeStackEvents(t *testing.T) {
	handler := newCFNGateway(t)
	mustCreateStack(t, handler, "events-stack")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cfnReq(t, "DescribeStackEvents", url.Values{
		"StackName": {"events-stack"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStackEvents: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"DescribeStackEventsResponse",
		"events-stack",
		"CREATE_COMPLETE",
		"AWS::CloudFormation::Stack",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("DescribeStackEvents: expected %q in response\nbody: %s", want, body)
		}
	}
}
