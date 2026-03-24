package eventbridge_test

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
	ebsvc "github.com/neureaux/cloudmock/services/eventbridge"
)

// newEBGateway builds a full gateway stack with the EventBridge service registered and IAM disabled.
func newEBGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(ebsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// ebReq builds a JSON POST request targeting the EventBridge service via X-Amz-Target.
func ebReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("ebReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSEvents."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/events/aws4_request, SignedHeaders=host, Signature=abc123")
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

// mustCreateBus creates an event bus and returns its ARN.
func mustCreateBus(t *testing.T, handler http.Handler, name string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "CreateEventBus", map[string]any{
		"Name": name,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateEventBus %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	arn, _ := m["EventBusArn"].(string)
	if arn == "" {
		t.Fatalf("CreateEventBus: EventBusArn is empty\nbody: %s", w.Body.String())
	}
	return arn
}

// mustPutRule creates a rule and returns its ARN.
func mustPutRule(t *testing.T, handler http.Handler, name, busName, pattern string) string {
	t.Helper()
	body := map[string]any{
		"Name":         name,
		"EventPattern": pattern,
		"State":        "ENABLED",
	}
	if busName != "" {
		body["EventBusName"] = busName
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutRule", body))
	if w.Code != http.StatusOK {
		t.Fatalf("PutRule %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	arn, _ := m["RuleArn"].(string)
	if arn == "" {
		t.Fatalf("PutRule: RuleArn is empty\nbody: %s", w.Body.String())
	}
	return arn
}

// ---- Test 1: ListEventBuses (verify default exists) + CreateEventBus ----

func TestEB_ListEventBusesAndCreate(t *testing.T) {
	handler := newEBGateway(t)

	// Default bus must always be present.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListEventBuses", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEventBuses: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "default") {
		t.Errorf("ListEventBuses: expected 'default' bus in response\nbody: %s", w.Body.String())
	}

	// Create a custom bus.
	arn := mustCreateBus(t, handler, "my-bus")
	if !strings.Contains(arn, "my-bus") {
		t.Errorf("CreateEventBus: expected ARN to contain bus name, got %s", arn)
	}
	if !strings.HasPrefix(arn, "arn:aws:events:") {
		t.Errorf("CreateEventBus: expected ARN prefix arn:aws:events:, got %s", arn)
	}

	// List again — should include custom bus.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ebReq(t, "ListEventBuses", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("ListEventBuses after create: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "my-bus") {
		t.Errorf("ListEventBuses: expected 'my-bus' in response\nbody: %s", w2.Body.String())
	}

	// DescribeEventBus
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeEventBus", map[string]any{"Name": "my-bus"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeEventBus: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "my-bus") {
		t.Errorf("DescribeEventBus: expected bus name in response\nbody: %s", wd.Body.String())
	}

	// Creating a duplicate bus must fail.
	wdup := httptest.NewRecorder()
	handler.ServeHTTP(wdup, ebReq(t, "CreateEventBus", map[string]any{"Name": "my-bus"}))
	if wdup.Code == http.StatusOK {
		t.Error("CreateEventBus duplicate: expected error, got 200")
	}
}

// ---- Test 2: PutRule + DescribeRule + ListRules ----

func TestEB_Rules(t *testing.T) {
	handler := newEBGateway(t)

	ruleARN := mustPutRule(t, handler, "my-rule", "", `{"source":["myapp"]}`)
	if !strings.Contains(ruleARN, "my-rule") {
		t.Errorf("PutRule: expected ARN to contain rule name, got %s", ruleARN)
	}
	if !strings.HasPrefix(ruleARN, "arn:aws:events:") {
		t.Errorf("PutRule: expected ARN prefix arn:aws:events:, got %s", ruleARN)
	}

	// DescribeRule
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, ebReq(t, "DescribeRule", map[string]any{"Name": "my-rule"}))
	if wr.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}
	body := wr.Body.String()
	for _, want := range []string{"my-rule", "ENABLED", "myapp"} {
		if !strings.Contains(body, want) {
			t.Errorf("DescribeRule: expected %q in response\nbody: %s", want, body)
		}
	}

	// PutRule again (update) — should be idempotent.
	ruleARN2 := mustPutRule(t, handler, "my-rule", "", `{"source":["updated"]}`)
	if ruleARN2 != ruleARN {
		// ARN should stay stable.
		t.Errorf("PutRule update: expected same ARN, got %s vs %s", ruleARN, ruleARN2)
	}

	// Add a second rule with a prefix.
	mustPutRule(t, handler, "my-rule-2", "", `{"source":["other"]}`)

	// ListRules
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListRules", map[string]any{"NamePrefix": "my-rule"}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListRules: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	if !strings.Contains(wl.Body.String(), "my-rule") {
		t.Errorf("ListRules: expected rule in response\nbody: %s", wl.Body.String())
	}
}

// ---- Test 3: PutTargets + ListTargetsByRule + RemoveTargets ----

func TestEB_Targets(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "target-rule", "", `{"source":["app"]}`)

	// PutTargets
	wpt := httptest.NewRecorder()
	handler.ServeHTTP(wpt, ebReq(t, "PutTargets", map[string]any{
		"Rule": "target-rule",
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:lambda:us-east-1:000000000000:function:my-fn"},
			{"Id": "t2", "Arn": "arn:aws:sqs:us-east-1:000000000000:my-queue", "Input": `{"key":"val"}`},
		},
	}))
	if wpt.Code != http.StatusOK {
		t.Fatalf("PutTargets: expected 200, got %d\nbody: %s", wpt.Code, wpt.Body.String())
	}
	ptResp := decodeJSON(t, wpt.Body.String())
	if count, _ := ptResp["FailedEntryCount"].(float64); count != 0 {
		t.Errorf("PutTargets: expected 0 failures, got %v\nbody: %s", count, wpt.Body.String())
	}

	// ListTargetsByRule
	wlt := httptest.NewRecorder()
	handler.ServeHTTP(wlt, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "target-rule"}))
	if wlt.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule: expected 200, got %d\nbody: %s", wlt.Code, wlt.Body.String())
	}
	ltBody := wlt.Body.String()
	if !strings.Contains(ltBody, "t1") || !strings.Contains(ltBody, "t2") {
		t.Errorf("ListTargetsByRule: expected t1 and t2\nbody: %s", ltBody)
	}

	// RemoveTargets
	wrt := httptest.NewRecorder()
	handler.ServeHTTP(wrt, ebReq(t, "RemoveTargets", map[string]any{
		"Rule": "target-rule",
		"Ids":  []string{"t1"},
	}))
	if wrt.Code != http.StatusOK {
		t.Fatalf("RemoveTargets: expected 200, got %d\nbody: %s", wrt.Code, wrt.Body.String())
	}

	// List again — t1 should be gone.
	wlt2 := httptest.NewRecorder()
	handler.ServeHTTP(wlt2, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "target-rule"}))
	if strings.Contains(wlt2.Body.String(), `"t1"`) {
		t.Errorf("ListTargetsByRule after remove: t1 should be gone\nbody: %s", wlt2.Body.String())
	}
	if !strings.Contains(wlt2.Body.String(), "t2") {
		t.Errorf("ListTargetsByRule after remove: t2 should still be present\nbody: %s", wlt2.Body.String())
	}
}

// ---- Test 4: PutEvents ----

func TestEB_PutEvents(t *testing.T) {
	handler := newEBGateway(t)

	wpe := httptest.NewRecorder()
	handler.ServeHTTP(wpe, ebReq(t, "PutEvents", map[string]any{
		"Entries": []map[string]any{
			{
				"Source":       "myapp",
				"DetailType":   "UserCreated",
				"Detail":       `{"userId":"123"}`,
				"EventBusName": "default",
			},
			{
				"Source":     "otherapp",
				"DetailType": "OrderPlaced",
				"Detail":     `{"orderId":"456"}`,
			},
		},
	}))
	if wpe.Code != http.StatusOK {
		t.Fatalf("PutEvents: expected 200, got %d\nbody: %s", wpe.Code, wpe.Body.String())
	}

	resp := decodeJSON(t, wpe.Body.String())
	if count, _ := resp["FailedEntryCount"].(float64); count != 0 {
		t.Errorf("PutEvents: expected 0 failures, got %v", count)
	}

	entries, _ := resp["Entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("PutEvents: expected 2 result entries, got %d\nbody: %s", len(entries), wpe.Body.String())
	}

	for i, entry := range entries {
		e, _ := entry.(map[string]any)
		eventID, _ := e["EventId"].(string)
		if eventID == "" {
			t.Errorf("PutEvents: entry %d has empty EventId\nbody: %s", i, wpe.Body.String())
		}
	}

	// Empty entries must fail.
	wpef := httptest.NewRecorder()
	handler.ServeHTTP(wpef, ebReq(t, "PutEvents", map[string]any{
		"Entries": []any{},
	}))
	if wpef.Code == http.StatusOK {
		t.Error("PutEvents with empty Entries: expected error, got 200")
	}
}

// ---- Test 5: DeleteRule + DeleteEventBus ----

func TestEB_DeleteRuleAndBus(t *testing.T) {
	handler := newEBGateway(t)
	mustCreateBus(t, handler, "deletable-bus")
	mustPutRule(t, handler, "deletable-rule", "deletable-bus", `{"source":["x"]}`)

	// DeleteRule
	wdr := httptest.NewRecorder()
	handler.ServeHTTP(wdr, ebReq(t, "DeleteRule", map[string]any{
		"Name":         "deletable-rule",
		"EventBusName": "deletable-bus",
	}))
	if wdr.Code != http.StatusOK {
		t.Fatalf("DeleteRule: expected 200, got %d\nbody: %s", wdr.Code, wdr.Body.String())
	}

	// DescribeRule after deletion must fail.
	wdr2 := httptest.NewRecorder()
	handler.ServeHTTP(wdr2, ebReq(t, "DescribeRule", map[string]any{
		"Name":         "deletable-rule",
		"EventBusName": "deletable-bus",
	}))
	if wdr2.Code == http.StatusOK {
		t.Error("DescribeRule after delete: expected error, got 200")
	}

	// Deleting the default bus must fail.
	wddb := httptest.NewRecorder()
	handler.ServeHTTP(wddb, ebReq(t, "DeleteEventBus", map[string]any{"Name": "default"}))
	if wddb.Code == http.StatusOK {
		t.Error("DeleteEventBus default: expected error, got 200")
	}

	// DeleteEventBus
	wdb := httptest.NewRecorder()
	handler.ServeHTTP(wdb, ebReq(t, "DeleteEventBus", map[string]any{"Name": "deletable-bus"}))
	if wdb.Code != http.StatusOK {
		t.Fatalf("DeleteEventBus: expected 200, got %d\nbody: %s", wdb.Code, wdb.Body.String())
	}

	// List buses — deletable-bus must be gone.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListEventBuses", nil))
	if strings.Contains(wl.Body.String(), "deletable-bus") {
		t.Errorf("ListEventBuses: deletable-bus should be gone\nbody: %s", wl.Body.String())
	}

	// Delete again — must fail.
	wdb2 := httptest.NewRecorder()
	handler.ServeHTTP(wdb2, ebReq(t, "DeleteEventBus", map[string]any{"Name": "deletable-bus"}))
	if wdb2.Code == http.StatusOK {
		t.Error("DeleteEventBus second time: expected error, got 200")
	}
}

// ---- Test 6: EnableRule / DisableRule ----

func TestEB_EnableDisableRule(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "toggle-rule", "", `{"source":["app"]}`)

	// Disable the rule.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DisableRule", map[string]any{"Name": "toggle-rule"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DisableRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Describe — state must be DISABLED.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ebReq(t, "DescribeRule", map[string]any{"Name": "toggle-rule"}))
	if !strings.Contains(wdesc.Body.String(), "DISABLED") {
		t.Errorf("DisableRule: expected state DISABLED\nbody: %s", wdesc.Body.String())
	}

	// Enable the rule.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, ebReq(t, "EnableRule", map[string]any{"Name": "toggle-rule"}))
	if we.Code != http.StatusOK {
		t.Fatalf("EnableRule: expected 200, got %d\nbody: %s", we.Code, we.Body.String())
	}

	// Describe — state must be ENABLED.
	wdesc2 := httptest.NewRecorder()
	handler.ServeHTTP(wdesc2, ebReq(t, "DescribeRule", map[string]any{"Name": "toggle-rule"}))
	if !strings.Contains(wdesc2.Body.String(), "ENABLED") {
		t.Errorf("EnableRule: expected state ENABLED\nbody: %s", wdesc2.Body.String())
	}

	// EnableRule on non-existent rule — must fail.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, ebReq(t, "EnableRule", map[string]any{"Name": "no-such-rule"}))
	if wne.Code == http.StatusOK {
		t.Error("EnableRule on non-existent rule: expected error, got 200")
	}
}
