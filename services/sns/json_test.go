package sns_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// snsJSONReq builds a JSON-protocol POST request targeting the SNS service.
func snsJSONReq(t *testing.T, target string, body interface{}) *http.Request {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal JSON body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", target)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sns/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// mustJSONCreateTopic creates a topic via JSON protocol and returns its ARN.
func mustJSONCreateTopic(t *testing.T, handler http.Handler, name string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsJSONReq(t, "SNS.CreateTopic", map[string]interface{}{
		"Name": name,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON CreateTopic %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}

	var resp struct {
		TopicArn string `json:"TopicArn"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON CreateTopic: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.TopicArn == "" {
		t.Fatalf("JSON CreateTopic: TopicArn is empty\nbody: %s", w.Body.String())
	}
	return resp.TopicArn
}

// mustJSONSubscribe subscribes via JSON protocol and returns the SubscriptionArn.
func mustJSONSubscribe(t *testing.T, handler http.Handler, topicArn, protocol, endpoint string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsJSONReq(t, "SNS.Subscribe", map[string]interface{}{
		"TopicArn": topicArn,
		"Protocol": protocol,
		"Endpoint": endpoint,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON Subscribe: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		SubscriptionArn string `json:"SubscriptionArn"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON Subscribe: unmarshal: %v", err)
	}
	if resp.SubscriptionArn == "" {
		t.Fatalf("JSON Subscribe: SubscriptionArn is empty")
	}
	return resp.SubscriptionArn
}

// ---- Test: CreateTopic + ListTopics (JSON) ----

func TestSNSJSON_CreateTopicAndListTopics(t *testing.T) {
	handler := newSNSGateway(t)

	arn1 := mustJSONCreateTopic(t, handler, "json-topic-1")
	arn2 := mustJSONCreateTopic(t, handler, "json-topic-2")

	if !strings.Contains(arn1, "json-topic-1") {
		t.Errorf("CreateTopic JSON: expected ARN to contain topic name, got %s", arn1)
	}
	if !strings.HasPrefix(arn1, "arn:aws:sns:") {
		t.Errorf("CreateTopic JSON: expected ARN prefix, got %s", arn1)
	}
	_ = arn2

	// ListTopics
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsJSONReq(t, "SNS.ListTopics", map[string]interface{}{}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON ListTopics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var listResp struct {
		Topics []struct {
			TopicArn string `json:"TopicArn"`
		} `json:"Topics"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Topics) < 2 {
		t.Fatalf("JSON ListTopics: expected at least 2, got %d", len(listResp.Topics))
	}

	// Idempotency.
	arn1Again := mustJSONCreateTopic(t, handler, "json-topic-1")
	if arn1Again != arn1 {
		t.Errorf("CreateTopic idempotency: expected %s, got %s", arn1, arn1Again)
	}
}

// ---- Test: Subscribe + ListSubscriptions (JSON) ----

func TestSNSJSON_SubscribeAndListSubscriptions(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustJSONCreateTopic(t, handler, "json-sub-topic")

	subArn := mustJSONSubscribe(t, handler, topicArn, "sqs", "arn:aws:sqs:us-east-1:000000000000:my-queue")
	if !strings.Contains(subArn, "json-sub-topic") {
		t.Errorf("Subscribe JSON: expected ARN to contain topic name, got %s", subArn)
	}

	// ListSubscriptions
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsJSONReq(t, "SNS.ListSubscriptions", map[string]interface{}{}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON ListSubscriptions: expected 200, got %d", w.Code)
	}
	var listResp struct {
		Subscriptions []struct {
			SubscriptionArn string `json:"SubscriptionArn"`
			Protocol        string `json:"Protocol"`
		} `json:"Subscriptions"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listResp)
	if len(listResp.Subscriptions) == 0 {
		t.Fatal("JSON ListSubscriptions: expected at least 1")
	}
	if listResp.Subscriptions[0].Protocol != "sqs" {
		t.Errorf("JSON ListSubscriptions: expected protocol sqs, got %s", listResp.Subscriptions[0].Protocol)
	}

	// ListSubscriptionsByTopic
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, snsJSONReq(t, "SNS.ListSubscriptionsByTopic", map[string]interface{}{
		"TopicArn": topicArn,
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("JSON ListSubscriptionsByTopic: expected 200, got %d", wt.Code)
	}
	var byTopicResp struct {
		Subscriptions []struct {
			TopicArn string `json:"TopicArn"`
		} `json:"Subscriptions"`
	}
	_ = json.Unmarshal(wt.Body.Bytes(), &byTopicResp)
	if len(byTopicResp.Subscriptions) == 0 {
		t.Fatal("JSON ListSubscriptionsByTopic: expected at least 1")
	}
}

// ---- Test: Publish (JSON) ----

func TestSNSJSON_Publish(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustJSONCreateTopic(t, handler, "json-publish-topic")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsJSONReq(t, "SNS.Publish", map[string]interface{}{
		"TopicArn": topicArn,
		"Message":  "Hello, JSON SNS!",
		"Subject":  "Test Subject",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON Publish: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		MessageId string `json:"MessageId"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.MessageId == "" {
		t.Error("JSON Publish: MessageId is empty")
	}

	// Publish to non-existent topic.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, snsJSONReq(t, "SNS.Publish", map[string]interface{}{
		"TopicArn": "arn:aws:sns:us-east-1:000000000000:no-such-topic",
		"Message":  "nope",
	}))
	if wf.Code == http.StatusOK {
		t.Error("Publish to non-existent topic: expected error, got 200")
	}
}

// ---- Test: Unsubscribe (JSON) ----

func TestSNSJSON_Unsubscribe(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustJSONCreateTopic(t, handler, "json-unsub-topic")
	subArn := mustJSONSubscribe(t, handler, topicArn, "https", "https://example.com/hook")

	// Unsubscribe.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, snsJSONReq(t, "SNS.Unsubscribe", map[string]interface{}{
		"SubscriptionArn": subArn,
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("JSON Unsubscribe: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// Verify it's gone.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, snsJSONReq(t, "SNS.ListSubscriptions", map[string]interface{}{}))
	body := wl.Body.String()
	if strings.Contains(body, subArn) {
		t.Error("ListSubscriptions: subscription should not appear after unsubscribe")
	}

	// Unsubscribe again — should fail.
	wu2 := httptest.NewRecorder()
	handler.ServeHTTP(wu2, snsJSONReq(t, "SNS.Unsubscribe", map[string]interface{}{
		"SubscriptionArn": subArn,
	}))
	if wu2.Code == http.StatusOK {
		t.Error("Unsubscribe second time: expected error, got 200")
	}
}

// ---- Test: DeleteTopic (JSON) ----

func TestSNSJSON_DeleteTopic(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustJSONCreateTopic(t, handler, "json-deletable-topic")

	// Delete.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, snsJSONReq(t, "SNS.DeleteTopic", map[string]interface{}{
		"TopicArn": topicArn,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("JSON DeleteTopic: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Verify gone.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, snsJSONReq(t, "SNS.ListTopics", map[string]interface{}{}))
	if strings.Contains(wl.Body.String(), "json-deletable-topic") {
		t.Error("ListTopics: topic should not appear after deletion")
	}

	// Delete again — should fail.
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, snsJSONReq(t, "SNS.DeleteTopic", map[string]interface{}{
		"TopicArn": topicArn,
	}))
	if wd2.Code == http.StatusOK {
		t.Error("DeleteTopic second time: expected error, got 200")
	}
}

// ---- Test: GetTopicAttributes + SetTopicAttributes (JSON) ----

func TestSNSJSON_TopicAttributes(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustJSONCreateTopic(t, handler, "json-attr-topic")

	// GetTopicAttributes.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, snsJSONReq(t, "SNS.GetTopicAttributes", map[string]interface{}{
		"TopicArn": topicArn,
	}))
	if wg.Code != http.StatusOK {
		t.Fatalf("JSON GetTopicAttributes: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	var attrResp struct {
		Attributes map[string]string `json:"Attributes"`
	}
	_ = json.Unmarshal(wg.Body.Bytes(), &attrResp)
	if attrResp.Attributes["TopicArn"] != topicArn {
		t.Errorf("GetTopicAttributes: TopicArn mismatch: %s", attrResp.Attributes["TopicArn"])
	}

	// SetTopicAttributes.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, snsJSONReq(t, "SNS.SetTopicAttributes", map[string]interface{}{
		"TopicArn":       topicArn,
		"AttributeName":  "DisplayName",
		"AttributeValue": "JSON Display Name",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("JSON SetTopicAttributes: expected 200, got %d", ws.Code)
	}

	// Verify change.
	wg2 := httptest.NewRecorder()
	handler.ServeHTTP(wg2, snsJSONReq(t, "SNS.GetTopicAttributes", map[string]interface{}{
		"TopicArn": topicArn,
	}))
	var attrResp2 struct {
		Attributes map[string]string `json:"Attributes"`
	}
	_ = json.Unmarshal(wg2.Body.Bytes(), &attrResp2)
	if attrResp2.Attributes["DisplayName"] != "JSON Display Name" {
		t.Errorf("SetTopicAttributes: expected 'JSON Display Name', got %q", attrResp2.Attributes["DisplayName"])
	}
}

// ---- Test: TagResource + UntagResource (JSON) ----

func TestSNSJSON_TagResource(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustJSONCreateTopic(t, handler, "json-tag-topic")

	// TagResource.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, snsJSONReq(t, "SNS.TagResource", map[string]interface{}{
		"ResourceArn": topicArn,
		"Tags": []map[string]string{
			{"Key": "env", "Value": "test"},
			{"Key": "team", "Value": "platform"},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("JSON TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// UntagResource.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, snsJSONReq(t, "SNS.UntagResource", map[string]interface{}{
		"ResourceArn": topicArn,
		"TagKeys":     []string{"env"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("JSON UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}
}

// ---- Test: XML protocol still works (regression) ----

func TestSNSJSON_XMLProtocolStillWorks(t *testing.T) {
	handler := newSNSGateway(t)

	// Use the existing XML helper.
	arn := mustCreateTopic(t, handler, "xml-still-works")
	if !strings.Contains(arn, "xml-still-works") {
		t.Errorf("XML CreateTopic still works: %s", arn)
	}
}
