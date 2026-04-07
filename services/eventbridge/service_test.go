package eventbridge_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	ebsvc "github.com/Viridian-Inc/cloudmock/services/eventbridge"
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

// ---- Test 7: PutEvents with detail-type matching ----

func TestEB_PutEventsDetailType(t *testing.T) {
	handler := newEBGateway(t)

	// Publish events with different detail types.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutEvents", map[string]any{
		"Entries": []map[string]any{
			{
				"Source":     "myapp",
				"DetailType": "UserCreated",
				"Detail":     `{"userId":"u1"}`,
			},
			{
				"Source":     "myapp",
				"DetailType": "UserDeleted",
				"Detail":     `{"userId":"u2"}`,
			},
			{
				"Source":     "otherapp",
				"DetailType": "UserCreated",
				"Detail":     `{"userId":"u3"}`,
			},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutEvents: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w.Body.String())
	entries, _ := resp["Entries"].([]any)
	if len(entries) != 3 {
		t.Fatalf("PutEvents: expected 3 entries, got %d", len(entries))
	}

	// Each entry must have a unique EventId.
	ids := make(map[string]bool)
	for i, entry := range entries {
		e, _ := entry.(map[string]any)
		id, _ := e["EventId"].(string)
		if id == "" {
			t.Errorf("PutEvents: entry %d has empty EventId", i)
		}
		if ids[id] {
			t.Errorf("PutEvents: duplicate EventId %s at entry %d", id, i)
		}
		ids[id] = true
	}

	if count, _ := resp["FailedEntryCount"].(float64); count != 0 {
		t.Errorf("PutEvents: expected 0 failures, got %v", count)
	}
}

// ---- Test 8: PutEvents with Resources and Time ----

func TestEB_PutEventsWithResourcesAndTime(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutEvents", map[string]any{
		"Entries": []map[string]any{
			{
				"Source":     "myapp",
				"DetailType": "OrderShipped",
				"Detail":     `{"orderId":"o1"}`,
				"Resources":  []string{"arn:aws:ec2:us-east-1:000000000000:instance/i-12345"},
				"Time":       "2025-01-15T10:30:00Z",
			},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutEvents: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	resp := decodeJSON(t, w.Body.String())
	entries, _ := resp["Entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("PutEvents: expected 1 entry, got %d", len(entries))
	}
	e, _ := entries[0].(map[string]any)
	if id, _ := e["EventId"].(string); id == "" {
		t.Error("PutEvents: EventId is empty")
	}
}

// ---- Test 9: PutEvents to custom event bus ----

func TestEB_PutEventsCustomBus(t *testing.T) {
	handler := newEBGateway(t)
	mustCreateBus(t, handler, "custom-bus")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutEvents", map[string]any{
		"Entries": []map[string]any{
			{
				"Source":       "myapp",
				"DetailType":   "Heartbeat",
				"Detail":       `{"status":"alive"}`,
				"EventBusName": "custom-bus",
			},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutEvents custom bus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.String())
	if count, _ := resp["FailedEntryCount"].(float64); count != 0 {
		t.Errorf("PutEvents custom bus: expected 0 failures, got %v", count)
	}
}

// ---- Test 10: Rule patterns — source + detail-type filter ----

func TestEB_RulePatternSourceAndDetailType(t *testing.T) {
	handler := newEBGateway(t)

	// Create a rule that matches only source=myapp AND detail-type=UserCreated.
	pattern := `{"source":["myapp"],"detail-type":["UserCreated"]}`
	mustPutRule(t, handler, "src-dt-rule", "", pattern)

	// DescribeRule — verify the pattern is stored.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{"Name": "src-dt-rule"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	body := wd.Body.String()
	if !strings.Contains(body, "myapp") {
		t.Errorf("DescribeRule: expected source pattern in response\nbody: %s", body)
	}
	if !strings.Contains(body, "UserCreated") {
		t.Errorf("DescribeRule: expected detail-type pattern in response\nbody: %s", body)
	}
}

// ---- Test 11: Rule with prefix pattern ----

func TestEB_RulePatternPrefix(t *testing.T) {
	handler := newEBGateway(t)

	// Create a rule with a prefix-based pattern.
	pattern := `{"source":[{"prefix":"com.mycompany"}]}`
	ruleARN := mustPutRule(t, handler, "prefix-rule", "", pattern)

	if !strings.Contains(ruleARN, "prefix-rule") {
		t.Errorf("PutRule: expected ARN to contain rule name, got %s", ruleARN)
	}

	// Describe to verify pattern persisted.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{"Name": "prefix-rule"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "com.mycompany") {
		t.Errorf("DescribeRule: expected prefix pattern\nbody: %s", wd.Body.String())
	}
}

// ---- Test 12: Rule with numeric pattern ----

func TestEB_RulePatternNumeric(t *testing.T) {
	handler := newEBGateway(t)

	// Create a rule with a numeric comparison pattern.
	pattern := `{"detail":{"price":[{"numeric":[">",100]}]}}`
	ruleARN := mustPutRule(t, handler, "numeric-rule", "", pattern)

	if !strings.Contains(ruleARN, "numeric-rule") {
		t.Errorf("PutRule: expected ARN to contain rule name, got %s", ruleARN)
	}

	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{"Name": "numeric-rule"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "numeric") {
		t.Errorf("DescribeRule: expected numeric pattern\nbody: %s", wd.Body.String())
	}
}

// ---- Test 13: Rule with exists pattern ----

func TestEB_RulePatternExists(t *testing.T) {
	handler := newEBGateway(t)

	// Create a rule with an exists pattern.
	pattern := `{"detail":{"productName":[{"exists":true}]}}`
	ruleARN := mustPutRule(t, handler, "exists-rule", "", pattern)

	if !strings.Contains(ruleARN, "exists-rule") {
		t.Errorf("PutRule: expected ARN to contain rule name, got %s", ruleARN)
	}

	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{"Name": "exists-rule"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "exists") {
		t.Errorf("DescribeRule: expected exists pattern\nbody: %s", wd.Body.String())
	}
}

// ---- Test 14: ListRules with NamePrefix filtering ----

func TestEB_ListRulesNamePrefix(t *testing.T) {
	handler := newEBGateway(t)

	mustPutRule(t, handler, "order-created", "", `{"source":["orders"]}`)
	mustPutRule(t, handler, "order-deleted", "", `{"source":["orders"]}`)
	mustPutRule(t, handler, "user-created", "", `{"source":["users"]}`)

	// Filter by prefix "order".
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListRules", map[string]any{"NamePrefix": "order"}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "order-created") {
		t.Errorf("ListRules: expected order-created\nbody: %s", body)
	}
	if !strings.Contains(body, "order-deleted") {
		t.Errorf("ListRules: expected order-deleted\nbody: %s", body)
	}
	if strings.Contains(body, "user-created") {
		t.Errorf("ListRules: should not contain user-created with prefix 'order'\nbody: %s", body)
	}

	// No prefix returns all rules.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ebReq(t, "ListRules", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("ListRules no prefix: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}
	body2 := w2.Body.String()
	if !strings.Contains(body2, "order-created") || !strings.Contains(body2, "user-created") {
		t.Errorf("ListRules no prefix: expected all rules\nbody: %s", body2)
	}
}

// ---- Test 15: ListRules on non-existent bus ----

func TestEB_ListRulesNonExistentBus(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListRules", map[string]any{"EventBusName": "no-such-bus"}))
	if w.Code == http.StatusOK {
		t.Error("ListRules on non-existent bus: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("ListRules: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 16: DescribeRule on non-existent rule ----

func TestEB_DescribeRuleNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeRule", map[string]any{"Name": "ghost-rule"}))
	if w.Code == http.StatusOK {
		t.Error("DescribeRule non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("DescribeRule: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 17: DescribeRule on non-existent bus ----

func TestEB_DescribeRuleNonExistentBus(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeRule", map[string]any{
		"Name":         "some-rule",
		"EventBusName": "no-such-bus",
	}))
	if w.Code == http.StatusOK {
		t.Error("DescribeRule on non-existent bus: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("DescribeRule: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 18: DescribeRule with description and schedule expression ----

func TestEB_DescribeRuleWithDescriptionAndSchedule(t *testing.T) {
	handler := newEBGateway(t)

	// Create rule with description and schedule expression.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutRule", map[string]any{
		"Name":               "scheduled-rule",
		"ScheduleExpression": "rate(5 minutes)",
		"Description":        "Runs every five minutes",
		"State":              "ENABLED",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutRule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Describe and verify all fields.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{"Name": "scheduled-rule"}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	body := wd.Body.String()
	for _, want := range []string{"scheduled-rule", "rate(5 minutes)", "Runs every five minutes", "ENABLED"} {
		if !strings.Contains(body, want) {
			t.Errorf("DescribeRule: expected %q in response\nbody: %s", want, body)
		}
	}
}

// ---- Test 19: PutTargets to non-existent rule ----

func TestEB_PutTargetsNonExistentRule(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutTargets", map[string]any{
		"Rule": "no-such-rule",
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:sqs:us-east-1:000000000000:my-queue"},
		},
	}))
	if w.Code == http.StatusOK {
		t.Error("PutTargets to non-existent rule: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("PutTargets: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 20: PutTargets replaces existing target with same ID ----

func TestEB_PutTargetsReplace(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "replace-rule", "", `{"source":["app"]}`)

	// Add a target.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutTargets", map[string]any{
		"Rule": "replace-rule",
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:sqs:us-east-1:000000000000:queue-old"},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutTargets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Replace the same target ID with a new ARN.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ebReq(t, "PutTargets", map[string]any{
		"Rule": "replace-rule",
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:sqs:us-east-1:000000000000:queue-new"},
		},
	}))
	if w2.Code != http.StatusOK {
		t.Fatalf("PutTargets replace: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	// List targets — only one target with the new ARN.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "replace-rule"}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	body := wl.Body.String()
	if !strings.Contains(body, "queue-new") {
		t.Errorf("ListTargetsByRule: expected new ARN\nbody: %s", body)
	}
	if strings.Contains(body, "queue-old") {
		t.Errorf("ListTargetsByRule: old ARN should be replaced\nbody: %s", body)
	}
}

// ---- Test 21: PutTargets with Input override ----

func TestEB_PutTargetsWithInput(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "input-rule", "", `{"source":["app"]}`)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutTargets", map[string]any{
		"Rule": "input-rule",
		"Targets": []map[string]any{
			{
				"Id":    "t1",
				"Arn":   "arn:aws:lambda:us-east-1:000000000000:function:my-fn",
				"Input": `{"constant":"value"}`,
			},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutTargets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// List and verify Input is stored.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "input-rule"}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	if !strings.Contains(wl.Body.String(), "constant") {
		t.Errorf("ListTargetsByRule: expected Input override\nbody: %s", wl.Body.String())
	}
}

// ---- Test 22: RemoveTargets from non-existent rule ----

func TestEB_RemoveTargetsNonExistentRule(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "RemoveTargets", map[string]any{
		"Rule": "no-such-rule",
		"Ids":  []string{"t1"},
	}))
	if w.Code == http.StatusOK {
		t.Error("RemoveTargets from non-existent rule: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("RemoveTargets: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 23: RemoveTargets with non-existent target ID ----

func TestEB_RemoveTargetsNotFoundId(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "rm-rule", "", `{"source":["x"]}`)

	// Add a target.
	wpt := httptest.NewRecorder()
	handler.ServeHTTP(wpt, ebReq(t, "PutTargets", map[string]any{
		"Rule": "rm-rule",
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:sqs:us-east-1:000000000000:q"},
		},
	}))
	if wpt.Code != http.StatusOK {
		t.Fatalf("PutTargets: expected 200, got %d", wpt.Code)
	}

	// Remove a target that does not exist.
	wrt := httptest.NewRecorder()
	handler.ServeHTTP(wrt, ebReq(t, "RemoveTargets", map[string]any{
		"Rule": "rm-rule",
		"Ids":  []string{"nonexistent-id"},
	}))
	if wrt.Code != http.StatusOK {
		t.Fatalf("RemoveTargets: expected 200 with FailedEntries, got %d\nbody: %s", wrt.Code, wrt.Body.String())
	}
	resp := decodeJSON(t, wrt.Body.String())
	failCount, _ := resp["FailedEntryCount"].(float64)
	if failCount != 1 {
		t.Errorf("RemoveTargets: expected 1 failed entry, got %v", failCount)
	}
}

// ---- Test 24: ListTargetsByRule on non-existent rule ----

func TestEB_ListTargetsByRuleNonExistentRule(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "no-such-rule"}))
	if w.Code == http.StatusOK {
		t.Error("ListTargetsByRule non-existent rule: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("ListTargetsByRule: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 25: ListTargetsByRule with empty targets ----

func TestEB_ListTargetsByRuleEmpty(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "empty-targets-rule", "", `{"source":["app"]}`)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "empty-targets-rule"}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON(t, w.Body.String())
	targets, _ := resp["Targets"].([]any)
	if len(targets) != 0 {
		t.Errorf("ListTargetsByRule: expected 0 targets, got %d", len(targets))
	}
}

// ---- Test 26: Enable/Disable rule on custom bus ----

func TestEB_EnableDisableRuleCustomBus(t *testing.T) {
	handler := newEBGateway(t)
	mustCreateBus(t, handler, "toggle-bus")
	mustPutRule(t, handler, "bus-rule", "toggle-bus", `{"source":["app"]}`)

	// Disable.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DisableRule", map[string]any{
		"Name":         "bus-rule",
		"EventBusName": "toggle-bus",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DisableRule custom bus: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Verify disabled.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ebReq(t, "DescribeRule", map[string]any{
		"Name":         "bus-rule",
		"EventBusName": "toggle-bus",
	}))
	if !strings.Contains(wdesc.Body.String(), "DISABLED") {
		t.Errorf("DescribeRule: expected DISABLED\nbody: %s", wdesc.Body.String())
	}

	// Enable.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, ebReq(t, "EnableRule", map[string]any{
		"Name":         "bus-rule",
		"EventBusName": "toggle-bus",
	}))
	if we.Code != http.StatusOK {
		t.Fatalf("EnableRule custom bus: expected 200, got %d\nbody: %s", we.Code, we.Body.String())
	}

	// Verify enabled.
	wdesc2 := httptest.NewRecorder()
	handler.ServeHTTP(wdesc2, ebReq(t, "DescribeRule", map[string]any{
		"Name":         "bus-rule",
		"EventBusName": "toggle-bus",
	}))
	if !strings.Contains(wdesc2.Body.String(), "ENABLED") {
		t.Errorf("DescribeRule: expected ENABLED\nbody: %s", wdesc2.Body.String())
	}
}

// ---- Test 27: DisableRule on non-existent rule ----

func TestEB_DisableRuleNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DisableRule", map[string]any{"Name": "no-such-rule"}))
	if w.Code == http.StatusOK {
		t.Error("DisableRule non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("DisableRule: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 28: DescribeEventBus non-existent ----

func TestEB_DescribeEventBusNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeEventBus", map[string]any{"Name": "no-such-bus"}))
	if w.Code == http.StatusOK {
		t.Error("DescribeEventBus non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("DescribeEventBus: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 29: DescribeEventBus default (empty name) ----

func TestEB_DescribeEventBusDefault(t *testing.T) {
	handler := newEBGateway(t)

	// Empty name should default to "default" bus.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeEventBus", map[string]any{}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeEventBus default: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "default") {
		t.Errorf("DescribeEventBus: expected 'default' in response\nbody: %s", w.Body.String())
	}
}

// ---- Test 30: TagResource / UntagResource / ListTagsForResource on event bus ----

func TestEB_TagResourceEventBus(t *testing.T) {
	handler := newEBGateway(t)
	busARN := mustCreateBus(t, handler, "tagged-bus")

	// Tag the bus.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ebReq(t, "TagResource", map[string]any{
		"ResourceARN": busARN,
		"Tags":        map[string]string{"env": "prod", "team": "backend"},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// List tags.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTagsForResource", map[string]any{"ResourceARN": busARN}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	body := wl.Body.String()
	if !strings.Contains(body, "prod") || !strings.Contains(body, "backend") {
		t.Errorf("ListTagsForResource: expected tags in response\nbody: %s", body)
	}

	// Untag.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, ebReq(t, "UntagResource", map[string]any{
		"ResourceARN": busARN,
		"TagKeys":     []string{"team"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// List again — team should be gone.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, ebReq(t, "ListTagsForResource", map[string]any{"ResourceARN": busARN}))
	body2 := wl2.Body.String()
	if !strings.Contains(body2, "prod") {
		t.Errorf("ListTagsForResource after untag: env=prod should remain\nbody: %s", body2)
	}
	if strings.Contains(body2, "backend") {
		t.Errorf("ListTagsForResource after untag: team=backend should be removed\nbody: %s", body2)
	}
}

// ---- Test 31: TagResource / ListTagsForResource on rule ----

func TestEB_TagResourceRule(t *testing.T) {
	handler := newEBGateway(t)
	ruleARN := mustPutRule(t, handler, "tagged-rule", "", `{"source":["app"]}`)

	// Tag the rule.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ebReq(t, "TagResource", map[string]any{
		"ResourceARN": ruleARN,
		"Tags":        map[string]string{"cost-center": "engineering"},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource rule: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// List tags.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTagsForResource", map[string]any{"ResourceARN": ruleARN}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource rule: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	if !strings.Contains(wl.Body.String(), "engineering") {
		t.Errorf("ListTagsForResource: expected tag value\nbody: %s", wl.Body.String())
	}
}

// ---- Test 32: TagResource on non-existent resource ----

func TestEB_TagResourceNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "TagResource", map[string]any{
		"ResourceARN": "arn:aws:events:us-east-1:000000000000:event-bus/no-such-bus",
		"Tags":        map[string]string{"key": "val"},
	}))
	if w.Code == http.StatusOK {
		t.Error("TagResource non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("TagResource: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 33: ListTagsForResource on non-existent resource ----

func TestEB_ListTagsForResourceNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListTagsForResource", map[string]any{
		"ResourceARN": "arn:aws:events:us-east-1:000000000000:event-bus/ghost",
	}))
	if w.Code == http.StatusOK {
		t.Error("ListTagsForResource non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("ListTagsForResource: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 34: DeleteRule on non-existent rule ----

func TestEB_DeleteRuleNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DeleteRule", map[string]any{"Name": "no-such-rule"}))
	if w.Code == http.StatusOK {
		t.Error("DeleteRule non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("DeleteRule: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 35: PutRule on non-existent bus ----

func TestEB_PutRuleNonExistentBus(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutRule", map[string]any{
		"Name":         "orphan-rule",
		"EventBusName": "no-such-bus",
		"EventPattern": `{"source":["app"]}`,
	}))
	if w.Code == http.StatusOK {
		t.Error("PutRule on non-existent bus: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("PutRule: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 36: PutRule without name (validation) ----

func TestEB_PutRuleNoName(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutRule", map[string]any{
		"EventPattern": `{"source":["app"]}`,
	}))
	if w.Code == http.StatusOK {
		t.Error("PutRule without name: expected error, got 200")
	}
}

// ---- Test 37: DeleteEventBus non-existent ----

func TestEB_DeleteEventBusNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DeleteEventBus", map[string]any{"Name": "no-such-bus"}))
	if w.Code == http.StatusOK {
		t.Error("DeleteEventBus non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("DeleteEventBus: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 38: CreateEventBus without name (validation) ----

func TestEB_CreateEventBusNoName(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "CreateEventBus", map[string]any{}))
	if w.Code == http.StatusOK {
		t.Error("CreateEventBus without name: expected error, got 200")
	}
}

// ---- Test 39: PutTargets without Rule (validation) ----

func TestEB_PutTargetsNoRule(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutTargets", map[string]any{
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:sqs:us-east-1:000000000000:q"},
		},
	}))
	if w.Code == http.StatusOK {
		t.Error("PutTargets without Rule: expected error, got 200")
	}
}

// ---- Test 40: RemoveTargets without Rule (validation) ----

func TestEB_RemoveTargetsNoRule(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "RemoveTargets", map[string]any{
		"Ids": []string{"t1"},
	}))
	if w.Code == http.StatusOK {
		t.Error("RemoveTargets without Rule: expected error, got 200")
	}
}

// ---- Test 41: ListTargetsByRule without Rule (validation) ----

func TestEB_ListTargetsByRuleNoRule(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "ListTargetsByRule", map[string]any{}))
	if w.Code == http.StatusOK {
		t.Error("ListTargetsByRule without Rule: expected error, got 200")
	}
}

// ---- Test 42: EnableRule / DisableRule without Name (validation) ----

func TestEB_EnableDisableRuleNoName(t *testing.T) {
	handler := newEBGateway(t)

	// EnableRule without name.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "EnableRule", map[string]any{}))
	if w.Code == http.StatusOK {
		t.Error("EnableRule without name: expected error, got 200")
	}

	// DisableRule without name.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ebReq(t, "DisableRule", map[string]any{}))
	if w2.Code == http.StatusOK {
		t.Error("DisableRule without name: expected error, got 200")
	}
}

// ---- Test 43: PutRule update preserves targets ----

func TestEB_PutRuleUpdatePreservesTargets(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "preserve-rule", "", `{"source":["v1"]}`)

	// Add targets.
	wpt := httptest.NewRecorder()
	handler.ServeHTTP(wpt, ebReq(t, "PutTargets", map[string]any{
		"Rule": "preserve-rule",
		"Targets": []map[string]any{
			{"Id": "t1", "Arn": "arn:aws:sqs:us-east-1:000000000000:q"},
		},
	}))
	if wpt.Code != http.StatusOK {
		t.Fatalf("PutTargets: expected 200, got %d", wpt.Code)
	}

	// Update the rule pattern.
	mustPutRule(t, handler, "preserve-rule", "", `{"source":["v2"]}`)

	// Targets should still be there.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "preserve-rule"}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule: expected 200, got %d", wl.Code)
	}
	if !strings.Contains(wl.Body.String(), "t1") {
		t.Errorf("PutRule update: targets should be preserved\nbody: %s", wl.Body.String())
	}

	// Verify pattern was updated.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{"Name": "preserve-rule"}))
	if !strings.Contains(wd.Body.String(), "v2") {
		t.Errorf("PutRule update: pattern should be updated\nbody: %s", wd.Body.String())
	}
}

// ---- Test 44: Invalid action returns error ----

func TestEB_InvalidAction(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "BogusAction", nil))
	if w.Code == http.StatusOK {
		t.Error("Invalid action: expected error, got 200")
	}
}

// ---- Test 45: Multiple targets on one rule ----

func TestEB_MultipleTargets(t *testing.T) {
	handler := newEBGateway(t)
	mustPutRule(t, handler, "multi-target-rule", "", `{"source":["app"]}`)

	// Add 5 targets at once.
	targets := make([]map[string]any, 5)
	for i := range targets {
		targets[i] = map[string]any{
			"Id":  fmt.Sprintf("t%d", i+1),
			"Arn": fmt.Sprintf("arn:aws:sqs:us-east-1:000000000000:queue-%d", i+1),
		}
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutTargets", map[string]any{
		"Rule":    "multi-target-rule",
		"Targets": targets,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutTargets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// List — all 5 should be present.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "multi-target-rule"}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	resp := decodeJSON(t, wl.Body.String())
	tgts, _ := resp["Targets"].([]any)
	if len(tgts) != 5 {
		t.Errorf("ListTargetsByRule: expected 5 targets, got %d\nbody: %s", len(tgts), wl.Body.String())
	}

	// Remove 2 targets.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, ebReq(t, "RemoveTargets", map[string]any{
		"Rule": "multi-target-rule",
		"Ids":  []string{"t2", "t4"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("RemoveTargets: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}

	// List — 3 should remain.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, ebReq(t, "ListTargetsByRule", map[string]any{"Rule": "multi-target-rule"}))
	resp2 := decodeJSON(t, wl2.Body.String())
	tgts2, _ := resp2["Targets"].([]any)
	if len(tgts2) != 3 {
		t.Errorf("ListTargetsByRule after remove: expected 3 targets, got %d\nbody: %s", len(tgts2), wl2.Body.String())
	}
}

// ---- Test 46: UntagResource on non-existent resource ----

func TestEB_UntagResourceNotFound(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "UntagResource", map[string]any{
		"ResourceARN": "arn:aws:events:us-east-1:000000000000:event-bus/ghost",
		"TagKeys":     []string{"key"},
	}))
	if w.Code == http.StatusOK {
		t.Error("UntagResource non-existent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ResourceNotFoundException") {
		t.Errorf("UntagResource: expected ResourceNotFoundException\nbody: %s", w.Body.String())
	}
}

// ---- Test 47: DescribeRule without Name (validation) ----

func TestEB_DescribeRuleNoName(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DescribeRule", map[string]any{}))
	if w.Code == http.StatusOK {
		t.Error("DescribeRule without name: expected error, got 200")
	}
}

// ---- Test 48: DeleteRule without Name (validation) ----

func TestEB_DeleteRuleNoName(t *testing.T) {
	handler := newEBGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "DeleteRule", map[string]any{}))
	if w.Code == http.StatusOK {
		t.Error("DeleteRule without name: expected error, got 200")
	}
}

// ---- Test 49: Rules on custom bus ----

func TestEB_RulesOnCustomBus(t *testing.T) {
	handler := newEBGateway(t)
	mustCreateBus(t, handler, "custom-bus")

	// Create a rule on the custom bus.
	ruleARN := mustPutRule(t, handler, "custom-rule", "custom-bus", `{"source":["app"]}`)
	if !strings.Contains(ruleARN, "custom-bus") {
		t.Errorf("PutRule: expected ARN to contain bus name, got %s", ruleARN)
	}

	// Create a rule on default bus with the same name — should not collide.
	defaultRuleARN := mustPutRule(t, handler, "custom-rule", "", `{"source":["other"]}`)
	if defaultRuleARN == ruleARN {
		t.Error("PutRule: rules on different buses should have different ARNs")
	}

	// ListRules on custom bus should only show the custom bus rule.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListRules", map[string]any{"EventBusName": "custom-bus"}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListRules: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	body := wl.Body.String()
	if !strings.Contains(body, "custom-rule") {
		t.Errorf("ListRules: expected custom-rule\nbody: %s", body)
	}

	// DescribeRule on custom bus.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ebReq(t, "DescribeRule", map[string]any{
		"Name":         "custom-rule",
		"EventBusName": "custom-bus",
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeRule: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "app") {
		t.Errorf("DescribeRule: expected pattern from custom bus rule\nbody: %s", wd.Body.String())
	}
}

// ---- Test 50: PutTargets on custom bus ----

func TestEB_PutTargetsCustomBus(t *testing.T) {
	handler := newEBGateway(t)
	mustCreateBus(t, handler, "target-bus")
	mustPutRule(t, handler, "bus-target-rule", "target-bus", `{"source":["app"]}`)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ebReq(t, "PutTargets", map[string]any{
		"Rule":         "bus-target-rule",
		"EventBusName": "target-bus",
		"Targets": []map[string]any{
			{"Id": "bt1", "Arn": "arn:aws:sqs:us-east-1:000000000000:q"},
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("PutTargets custom bus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// List targets on the custom bus rule.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ebReq(t, "ListTargetsByRule", map[string]any{
		"Rule":         "bus-target-rule",
		"EventBusName": "target-bus",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTargetsByRule custom bus: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	if !strings.Contains(wl.Body.String(), "bt1") {
		t.Errorf("ListTargetsByRule: expected bt1\nbody: %s", wl.Body.String())
	}
}
