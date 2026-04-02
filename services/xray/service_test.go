package xray_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	xraysvc "github.com/neureaux/cloudmock/services/xray"
)

// newXRayGateway builds a full gateway stack with the XRay service registered and IAM disabled.
func newXRayGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(xraysvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// xrayReq builds a JSON POST request targeting the XRay service via X-Amz-Target.
func xrayReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("xrayReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSXRay."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/xray/aws4_request, SignedHeaders=host, Signature=abc123")
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

// ---- Test 1: PutTraceSegments ----

func TestXRay_PutTraceSegments(t *testing.T) {
	handler := newXRayGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, xrayReq(t, "PutTraceSegments", map[string]any{
		"TraceSegmentDocuments": []string{
			`{"id":"abc123","name":"my-service","start_time":1234567890,"end_time":1234567891}`,
			`{"id":"def456","name":"my-service","start_time":1234567892,"end_time":1234567893}`,
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutTraceSegments: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	if _, ok := m["UnprocessedTraceSegments"]; !ok {
		t.Errorf("PutTraceSegments: missing UnprocessedTraceSegments\nbody: %s", w.Body.String())
	}
}

// ---- Test 2: GetTraceSummaries ----

func TestXRay_GetTraceSummaries(t *testing.T) {
	handler := newXRayGateway(t)

	// Put some segments first
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, xrayReq(t, "PutTraceSegments", map[string]any{
		"TraceSegmentDocuments": []string{
			`{"id":"seg1","name":"svc","start_time":1234567890,"end_time":1234567891}`,
		},
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutTraceSegments: %d %s", wp.Code, wp.Body.String())
	}

	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, xrayReq(t, "GetTraceSummaries", map[string]any{
		"StartTime": 0,
		"EndTime":   9999999999,
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetTraceSummaries: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	m := decodeJSON(t, wg.Body.String())
	summaries, ok := m["TraceSummaries"].([]any)
	if !ok {
		t.Fatalf("GetTraceSummaries: missing TraceSummaries\nbody: %s", wg.Body.String())
	}
	if len(summaries) < 1 {
		t.Errorf("GetTraceSummaries: expected at least 1 summary, got %d", len(summaries))
	}
}

// ---- Test 3: BatchGetTraces ----

func TestXRay_BatchGetTraces(t *testing.T) {
	handler := newXRayGateway(t)

	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, xrayReq(t, "BatchGetTraces", map[string]any{
		"TraceIds": []string{"1-abc-def123456789012"},
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("BatchGetTraces: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	m := decodeJSON(t, wg.Body.String())
	if _, ok := m["Traces"]; !ok {
		t.Errorf("BatchGetTraces: missing Traces\nbody: %s", wg.Body.String())
	}
}

// ---- Test 4: GetTraceGraph ----

func TestXRay_GetTraceGraph(t *testing.T) {
	handler := newXRayGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, xrayReq(t, "GetTraceGraph", map[string]any{
		"TraceIds": []string{"1-abc-def123456789012"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTraceGraph: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	if _, ok := m["Services"]; !ok {
		t.Errorf("GetTraceGraph: missing Services\nbody: %s", w.Body.String())
	}
}

// ---- Test 5: GetSamplingRules (default rule present) ----

func TestXRay_GetSamplingRules_DefaultPresent(t *testing.T) {
	handler := newXRayGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, xrayReq(t, "GetSamplingRules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSamplingRules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	records, ok := m["SamplingRuleRecords"].([]any)
	if !ok {
		t.Fatalf("GetSamplingRules: missing SamplingRuleRecords\nbody: %s", w.Body.String())
	}
	if len(records) < 1 {
		t.Error("GetSamplingRules: expected at least 1 rule (Default)")
	}
	// Verify the Default rule exists
	found := false
	for _, rec := range records {
		r := rec.(map[string]any)
		rule, _ := r["SamplingRule"].(map[string]any)
		if rule["RuleName"] == "Default" {
			found = true
		}
	}
	if !found {
		t.Error("GetSamplingRules: Default rule not found")
	}
}

// ---- Test 6: CreateSamplingRule and DeleteSamplingRule ----

func TestXRay_SamplingRule_CreateAndDelete(t *testing.T) {
	handler := newXRayGateway(t)

	// Create
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, xrayReq(t, "CreateSamplingRule", map[string]any{
		"SamplingRule": map[string]any{
			"RuleName":      "my-rule",
			"Priority":      100,
			"FixedRate":     0.10,
			"ReservoirSize": 10,
			"ServiceName":   "my-service",
			"ServiceType":   "AWS::ECS::Container",
			"Host":          "*",
			"HTTPMethod":    "GET",
			"URLPath":       "/api/*",
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSamplingRule: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	record, ok := mc["SamplingRuleRecord"].(map[string]any)
	if !ok {
		t.Fatalf("CreateSamplingRule: missing SamplingRuleRecord\nbody: %s", wc.Body.String())
	}
	rule, _ := record["SamplingRule"].(map[string]any)
	if rule["RuleName"] != "my-rule" {
		t.Errorf("CreateSamplingRule: expected RuleName=my-rule, got %v", rule["RuleName"])
	}
	if rule["RuleARN"] == "" {
		t.Error("CreateSamplingRule: RuleARN is empty")
	}

	// Verify it appears in GetSamplingRules
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, xrayReq(t, "GetSamplingRules", nil))
	if wl.Code != http.StatusOK {
		t.Fatalf("GetSamplingRules: %d %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	records, _ := ml["SamplingRuleRecords"].([]any)
	foundRule := false
	for _, rec := range records {
		r := rec.(map[string]any)
		ruleMap, _ := r["SamplingRule"].(map[string]any)
		if ruleMap["RuleName"] == "my-rule" {
			foundRule = true
		}
	}
	if !foundRule {
		t.Error("GetSamplingRules: my-rule not found after create")
	}

	// Delete
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, xrayReq(t, "DeleteSamplingRule", map[string]any{
		"RuleName": "my-rule",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteSamplingRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
}

// ---- Test 7: CreateSamplingRule duplicate returns error ----

func TestXRay_CreateSamplingRule_Duplicate(t *testing.T) {
	handler := newXRayGateway(t)

	body := map[string]any{
		"SamplingRule": map[string]any{
			"RuleName":      "dup-rule",
			"Priority":      200,
			"FixedRate":     0.05,
			"ReservoirSize": 5,
		},
	}

	wc1 := httptest.NewRecorder()
	handler.ServeHTTP(wc1, xrayReq(t, "CreateSamplingRule", body))
	if wc1.Code != http.StatusOK {
		t.Fatalf("CreateSamplingRule first: %d %s", wc1.Code, wc1.Body.String())
	}

	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, xrayReq(t, "CreateSamplingRule", body))
	if wc2.Code != http.StatusBadRequest {
		t.Fatalf("CreateSamplingRule duplicate: expected 400, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}
}

// ---- Test 8: UpdateSamplingRule ----

func TestXRay_UpdateSamplingRule(t *testing.T) {
	handler := newXRayGateway(t)

	// Create
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, xrayReq(t, "CreateSamplingRule", map[string]any{
		"SamplingRule": map[string]any{
			"RuleName":      "upd-rule",
			"Priority":      300,
			"FixedRate":     0.01,
			"ReservoirSize": 1,
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSamplingRule: %d %s", wc.Code, wc.Body.String())
	}

	// Update
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, xrayReq(t, "UpdateSamplingRule", map[string]any{
		"SamplingRuleUpdate": map[string]any{
			"RuleName":      "upd-rule",
			"FixedRate":     0.50,
			"ReservoirSize": 50,
		},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UpdateSamplingRule: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}
	mu := decodeJSON(t, wu.Body.String())
	record, _ := mu["SamplingRuleRecord"].(map[string]any)
	rule, _ := record["SamplingRule"].(map[string]any)
	if rule["FixedRate"].(float64) != 0.50 {
		t.Errorf("UpdateSamplingRule: expected FixedRate=0.50, got %v", rule["FixedRate"])
	}
}

// ---- Test 9: DeleteSamplingRule not found returns error ----

func TestXRay_DeleteSamplingRule_NotFound(t *testing.T) {
	handler := newXRayGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, xrayReq(t, "DeleteSamplingRule", map[string]any{
		"RuleName": "nonexistent-rule",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteSamplingRule not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 10: CreateGroup and GetGroup ----

func TestXRay_Group_CreateAndGet(t *testing.T) {
	handler := newXRayGateway(t)

	// Create
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, xrayReq(t, "CreateGroup", map[string]any{
		"GroupName":        "my-group",
		"FilterExpression": "service(\"my-service\")",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateGroup: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	g, ok := mc["Group"].(map[string]any)
	if !ok {
		t.Fatalf("CreateGroup: missing Group\nbody: %s", wc.Body.String())
	}
	if g["GroupName"] != "my-group" {
		t.Errorf("CreateGroup: expected GroupName=my-group, got %v", g["GroupName"])
	}
	if g["GroupARN"] == "" {
		t.Error("CreateGroup: GroupARN is empty")
	}

	// GetGroup
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, xrayReq(t, "GetGroup", map[string]any{
		"GroupName": "my-group",
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetGroup: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	mg := decodeJSON(t, wg.Body.String())
	grp, _ := mg["Group"].(map[string]any)
	if grp["FilterExpression"] != "service(\"my-service\")" {
		t.Errorf("GetGroup: unexpected FilterExpression: %v", grp["FilterExpression"])
	}
}

// ---- Test 11: GetGroups lists all groups ----

func TestXRay_GetGroups(t *testing.T) {
	handler := newXRayGateway(t)

	for _, name := range []string{"group-a", "group-b", "group-c"} {
		wc := httptest.NewRecorder()
		handler.ServeHTTP(wc, xrayReq(t, "CreateGroup", map[string]any{
			"GroupName": name,
		}))
		if wc.Code != http.StatusOK {
			t.Fatalf("CreateGroup %s: %d %s", name, wc.Code, wc.Body.String())
		}
	}

	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, xrayReq(t, "GetGroups", nil))
	if wl.Code != http.StatusOK {
		t.Fatalf("GetGroups: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	groups, _ := ml["Groups"].([]any)
	if len(groups) != 3 {
		t.Errorf("GetGroups: expected 3 groups, got %d", len(groups))
	}
}

// ---- Test 12: UpdateGroup and DeleteGroup ----

func TestXRay_Group_UpdateAndDelete(t *testing.T) {
	handler := newXRayGateway(t)

	// Create
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, xrayReq(t, "CreateGroup", map[string]any{
		"GroupName":        "upd-group",
		"FilterExpression": "service(\"old\")",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateGroup: %d %s", wc.Code, wc.Body.String())
	}

	// Update
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, xrayReq(t, "UpdateGroup", map[string]any{
		"GroupName":        "upd-group",
		"FilterExpression": "service(\"new\")",
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UpdateGroup: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}
	mu := decodeJSON(t, wu.Body.String())
	grp, _ := mu["Group"].(map[string]any)
	if grp["FilterExpression"] != "service(\"new\")" {
		t.Errorf("UpdateGroup: expected updated filter, got %v", grp["FilterExpression"])
	}

	// Delete
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, xrayReq(t, "DeleteGroup", map[string]any{
		"GroupName": "upd-group",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteGroup: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// GetGroup after delete should fail
	wge := httptest.NewRecorder()
	handler.ServeHTTP(wge, xrayReq(t, "GetGroup", map[string]any{
		"GroupName": "upd-group",
	}))
	if wge.Code != http.StatusBadRequest {
		t.Fatalf("GetGroup after delete: expected 400, got %d\nbody: %s", wge.Code, wge.Body.String())
	}
}

// ---- Test 13: CreateGroup duplicate returns error ----

func TestXRay_CreateGroup_Duplicate(t *testing.T) {
	handler := newXRayGateway(t)

	body := map[string]any{"GroupName": "dup-group"}

	wc1 := httptest.NewRecorder()
	handler.ServeHTTP(wc1, xrayReq(t, "CreateGroup", body))
	if wc1.Code != http.StatusOK {
		t.Fatalf("CreateGroup first: %d %s", wc1.Code, wc1.Body.String())
	}

	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, xrayReq(t, "CreateGroup", body))
	if wc2.Code != http.StatusBadRequest {
		t.Fatalf("CreateGroup duplicate: expected 400, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}
}

// ---- Test 14: PutEncryptionConfig and GetEncryptionConfig ----

func TestXRay_EncryptionConfig(t *testing.T) {
	handler := newXRayGateway(t)

	// Default encryption config
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, xrayReq(t, "GetEncryptionConfig", nil))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetEncryptionConfig: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	mg := decodeJSON(t, wg.Body.String())
	cfg, ok := mg["EncryptionConfig"].(map[string]any)
	if !ok {
		t.Fatalf("GetEncryptionConfig: missing EncryptionConfig\nbody: %s", wg.Body.String())
	}
	if cfg["Type"] != "NONE" {
		t.Errorf("GetEncryptionConfig default: expected Type=NONE, got %v", cfg["Type"])
	}
	if cfg["Status"] != "ACTIVE" {
		t.Errorf("GetEncryptionConfig default: expected Status=ACTIVE, got %v", cfg["Status"])
	}

	// Put KMS encryption config
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, xrayReq(t, "PutEncryptionConfig", map[string]any{
		"KeyId": "arn:aws:kms:us-east-1:123456789012:key/test-key",
		"Type":  "KMS",
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PutEncryptionConfig: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}
	mp := decodeJSON(t, wp.Body.String())
	newCfg, _ := mp["EncryptionConfig"].(map[string]any)
	if newCfg["Type"] != "KMS" {
		t.Errorf("PutEncryptionConfig: expected Type=KMS, got %v", newCfg["Type"])
	}

	// Verify via GetEncryptionConfig
	wg2 := httptest.NewRecorder()
	handler.ServeHTTP(wg2, xrayReq(t, "GetEncryptionConfig", nil))
	if wg2.Code != http.StatusOK {
		t.Fatalf("GetEncryptionConfig after put: %d %s", wg2.Code, wg2.Body.String())
	}
	mg2 := decodeJSON(t, wg2.Body.String())
	cfg2, _ := mg2["EncryptionConfig"].(map[string]any)
	if cfg2["Type"] != "KMS" {
		t.Errorf("GetEncryptionConfig after put: expected Type=KMS, got %v", cfg2["Type"])
	}
}

// ---- Test 15: TagResource / UntagResource / ListTagsForResource ----

func TestXRay_TagOperations(t *testing.T) {
	handler := newXRayGateway(t)

	// Create a sampling rule to get an ARN
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, xrayReq(t, "CreateSamplingRule", map[string]any{
		"SamplingRule": map[string]any{
			"RuleName":      "tag-rule",
			"Priority":      500,
			"FixedRate":     0.02,
			"ReservoirSize": 2,
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateSamplingRule: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	record, _ := mc["SamplingRuleRecord"].(map[string]any)
	ruleMap, _ := record["SamplingRule"].(map[string]any)
	arn, _ := ruleMap["RuleARN"].(string)
	if arn == "" {
		t.Fatal("CreateSamplingRule: RuleARN is empty")
	}

	// TagResource
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, xrayReq(t, "TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags": map[string]any{
			"env":  "test",
			"team": "platform",
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// ListTagsForResource
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, xrayReq(t, "ListTagsForResource", map[string]any{
		"ResourceARN": arn,
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	tags, _ := ml["Tags"].([]any)
	if len(tags) != 2 {
		t.Errorf("ListTagsForResource: expected 2 tags, got %d\nbody: %s", len(tags), wl.Body.String())
	}

	// UntagResource
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, xrayReq(t, "UntagResource", map[string]any{
		"ResourceARN": arn,
		"TagKeys":     []string{"env"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// ListTagsForResource after untag — only "team" should remain
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, xrayReq(t, "ListTagsForResource", map[string]any{
		"ResourceARN": arn,
	}))
	if wl2.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource after untag: %d %s", wl2.Code, wl2.Body.String())
	}
	ml2 := decodeJSON(t, wl2.Body.String())
	tags2, _ := ml2["Tags"].([]any)
	if len(tags2) != 1 {
		t.Errorf("ListTagsForResource after untag: expected 1 tag, got %d", len(tags2))
	}
}

// ---- Test 16: InvalidAction ----

func TestXRay_InvalidAction(t *testing.T) {
	handler := newXRayGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, xrayReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 17: ServiceName and HealthCheck ----

func TestXRay_ServiceNameAndHealthCheck(t *testing.T) {
	svc := xraysvc.New("123456789012", "us-east-1")
	if svc.Name() != "xray" {
		t.Errorf("Name: expected xray, got %q", svc.Name())
	}
	if err := svc.HealthCheck(); err != nil {
		t.Errorf("HealthCheck: expected nil, got %v", err)
	}
}
