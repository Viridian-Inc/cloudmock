package sns_test

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
	snssvc "github.com/neureaux/cloudmock/services/sns"
)

// newSNSGateway builds a full gateway stack with the SNS service registered and IAM disabled.
func newSNSGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(snssvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// snsReq builds a form-encoded POST request targeting the SNS service.
func snsReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2010-03-31")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sns/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// mustCreateTopic is a test helper that creates a topic and returns its ARN.
func mustCreateTopic(t *testing.T, handler http.Handler, name string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "CreateTopic", url.Values{"Name": {name}}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTopic %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			TopicArn string `xml:"TopicArn"`
		} `xml:"CreateTopicResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateTopic: unmarshal response: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Result.TopicArn == "" {
		t.Fatalf("CreateTopic: TopicArn is empty\nbody: %s", w.Body.String())
	}
	return resp.Result.TopicArn
}

// mustSubscribe is a test helper that subscribes to a topic and returns the SubscriptionArn.
func mustSubscribe(t *testing.T, handler http.Handler, topicArn, protocol, endpoint string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "Subscribe", url.Values{
		"TopicArn": {topicArn},
		"Protocol": {protocol},
		"Endpoint": {endpoint},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("Subscribe: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			SubscriptionArn string `xml:"SubscriptionArn"`
		} `xml:"SubscribeResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Subscribe: unmarshal response: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Result.SubscriptionArn == "" {
		t.Fatalf("Subscribe: SubscriptionArn is empty\nbody: %s", w.Body.String())
	}
	return resp.Result.SubscriptionArn
}

// ---- Test 1: CreateTopic + ListTopics ----

func TestSNS_CreateTopicAndListTopics(t *testing.T) {
	handler := newSNSGateway(t)

	arn1 := mustCreateTopic(t, handler, "test-topic-1")
	arn2 := mustCreateTopic(t, handler, "test-topic-2")

	if !strings.Contains(arn1, "test-topic-1") {
		t.Errorf("CreateTopic: expected ARN to contain topic name, got %s", arn1)
	}
	if !strings.Contains(arn2, "test-topic-2") {
		t.Errorf("CreateTopic: expected ARN to contain topic name, got %s", arn2)
	}

	// ARN format: arn:aws:sns:{region}:{accountId}:{topicName}
	if !strings.HasPrefix(arn1, "arn:aws:sns:") {
		t.Errorf("CreateTopic: expected ARN prefix arn:aws:sns:, got %s", arn1)
	}

	// ListTopics
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "ListTopics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTopics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "test-topic-1") {
		t.Errorf("ListTopics: expected test-topic-1 in response\nbody: %s", body)
	}
	if !strings.Contains(body, "test-topic-2") {
		t.Errorf("ListTopics: expected test-topic-2 in response\nbody: %s", body)
	}

	// CreateTopic is idempotent — same ARN returned.
	arn1Again := mustCreateTopic(t, handler, "test-topic-1")
	if arn1Again != arn1 {
		t.Errorf("CreateTopic idempotency: expected %s, got %s", arn1, arn1Again)
	}
}

// ---- Test 2: Subscribe + ListSubscriptions ----

func TestSNS_SubscribeAndListSubscriptions(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "sub-topic")

	subArn := mustSubscribe(t, handler, topicArn, "sqs", "arn:aws:sqs:us-east-1:000000000000:my-queue")

	if !strings.Contains(subArn, "sub-topic") {
		t.Errorf("Subscribe: expected subscription ARN to contain topic name, got %s", subArn)
	}
	if !strings.HasPrefix(subArn, "arn:aws:sns:") {
		t.Errorf("Subscribe: expected subscription ARN prefix arn:aws:sns:, got %s", subArn)
	}

	// ListSubscriptions
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "ListSubscriptions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSubscriptions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "sub-topic") {
		t.Errorf("ListSubscriptions: expected sub-topic in response\nbody: %s", body)
	}
	if !strings.Contains(body, "sqs") {
		t.Errorf("ListSubscriptions: expected protocol sqs in response\nbody: %s", body)
	}

	// ListSubscriptionsByTopic
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, snsReq(t, "ListSubscriptionsByTopic", url.Values{"TopicArn": {topicArn}}))
	if wt.Code != http.StatusOK {
		t.Fatalf("ListSubscriptionsByTopic: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}
	bodyT := wt.Body.String()
	if !strings.Contains(bodyT, "sub-topic") {
		t.Errorf("ListSubscriptionsByTopic: expected sub-topic in response\nbody: %s", bodyT)
	}
}

// ---- Test 3: Publish — verify MessageId returned ----

func TestSNS_Publish(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "publish-topic")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "Publish", url.Values{
		"TopicArn": {topicArn},
		"Message":  {"Hello, SNS!"},
		"Subject":  {"Test Subject"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("Publish: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{"PublishResponse", "PublishResult", "MessageId"} {
		if !strings.Contains(body, want) {
			t.Errorf("Publish: expected %q in response\nbody: %s", want, body)
		}
	}

	// Extract MessageId and verify it is non-empty.
	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"PublishResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Publish: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Errorf("Publish: MessageId is empty\nbody: %s", body)
	}

	// Publish to non-existent topic — should fail.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, snsReq(t, "Publish", url.Values{
		"TopicArn": {"arn:aws:sns:us-east-1:000000000000:no-such-topic"},
		"Message":  {"nope"},
	}))
	if wf.Code == http.StatusOK {
		t.Error("Publish to non-existent topic: expected error, got 200")
	}
}

// ---- Test 4: Unsubscribe ----

func TestSNS_Unsubscribe(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "unsub-topic")
	subArn := mustSubscribe(t, handler, topicArn, "https", "https://example.com/endpoint")

	// Verify subscription appears in list.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, snsReq(t, "ListSubscriptions", nil))
	if !strings.Contains(wl.Body.String(), subArn) {
		t.Fatalf("ListSubscriptions: subscription not found before unsubscribe\nbody: %s", wl.Body.String())
	}

	// Unsubscribe.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, snsReq(t, "Unsubscribe", url.Values{"SubscriptionArn": {subArn}}))
	if wu.Code != http.StatusOK {
		t.Fatalf("Unsubscribe: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// Verify it's gone.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, snsReq(t, "ListSubscriptions", nil))
	if strings.Contains(wl2.Body.String(), subArn) {
		t.Errorf("ListSubscriptions: subscription should not appear after unsubscribe\nbody: %s", wl2.Body.String())
	}

	// Unsubscribe again — should fail.
	wu2 := httptest.NewRecorder()
	handler.ServeHTTP(wu2, snsReq(t, "Unsubscribe", url.Values{"SubscriptionArn": {subArn}}))
	if wu2.Code == http.StatusOK {
		t.Error("Unsubscribe second time: expected error, got 200")
	}
}

// ---- Test 5: DeleteTopic ----

func TestSNS_DeleteTopic(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "deletable-topic")

	// Subscribe before deleting so we can check subscriptions are cleaned up.
	subArn := mustSubscribe(t, handler, topicArn, "email", "test@example.com")
	_ = subArn

	// Verify topic in list.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, snsReq(t, "ListTopics", nil))
	if !strings.Contains(wl.Body.String(), "deletable-topic") {
		t.Fatal("ListTopics: topic not found before deletion")
	}

	// DeleteTopic.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, snsReq(t, "DeleteTopic", url.Values{"TopicArn": {topicArn}}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteTopic: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Verify topic is gone.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, snsReq(t, "ListTopics", nil))
	if strings.Contains(wl2.Body.String(), "deletable-topic") {
		t.Error("ListTopics: topic should not appear after deletion")
	}

	// Delete again — should fail.
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, snsReq(t, "DeleteTopic", url.Values{"TopicArn": {topicArn}}))
	if wd2.Code == http.StatusOK {
		t.Error("DeleteTopic second time: expected error, got 200")
	}

	// Publishing to a deleted topic should fail.
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, snsReq(t, "Publish", url.Values{
		"TopicArn": {topicArn},
		"Message":  {"ghost"},
	}))
	if wp.Code == http.StatusOK {
		t.Error("Publish to deleted topic: expected error, got 200")
	}
}

// ---- Test 6: GetTopicAttributes / SetTopicAttributes ----

func TestSNS_TopicAttributes(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "attr-topic")

	// GetTopicAttributes
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, snsReq(t, "GetTopicAttributes", url.Values{"TopicArn": {topicArn}}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetTopicAttributes: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	body := wg.Body.String()
	for _, want := range []string{
		"GetTopicAttributesResponse",
		"GetTopicAttributesResult",
		"TopicArn",
		"attr-topic",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GetTopicAttributes: expected %q in response\nbody: %s", want, body)
		}
	}

	// SetTopicAttributes
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, snsReq(t, "SetTopicAttributes", url.Values{
		"TopicArn":       {topicArn},
		"AttributeName":  {"DisplayName"},
		"AttributeValue": {"My Display Name"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SetTopicAttributes: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}

	// Verify attribute persisted via GetTopicAttributes.
	wg2 := httptest.NewRecorder()
	handler.ServeHTTP(wg2, snsReq(t, "GetTopicAttributes", url.Values{"TopicArn": {topicArn}}))
	if wg2.Code != http.StatusOK {
		t.Fatalf("GetTopicAttributes after set: expected 200, got %d\nbody: %s", wg2.Code, wg2.Body.String())
	}
	if !strings.Contains(wg2.Body.String(), "My Display Name") {
		t.Errorf("GetTopicAttributes: expected DisplayName value in response\nbody: %s", wg2.Body.String())
	}

	// SetTopicAttributes on non-existent topic.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, snsReq(t, "SetTopicAttributes", url.Values{
		"TopicArn":       {"arn:aws:sns:us-east-1:000000000000:no-topic"},
		"AttributeName":  {"DisplayName"},
		"AttributeValue": {"x"},
	}))
	if we.Code == http.StatusOK {
		t.Error("SetTopicAttributes on non-existent topic: expected error, got 200")
	}
}

// ---- Test 7: Unknown action ----

func TestSNS_UnknownAction(t *testing.T) {
	handler := newSNSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 8: TagResource / UntagResource ----

func TestSNS_TagResource(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "tag-topic")

	// TagResource
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, snsReq(t, "TagResource", url.Values{
		"ResourceArn":           {topicArn},
		"Tags.member.1.Key":     {"env"},
		"Tags.member.1.Value":   {"test"},
		"Tags.member.2.Key":     {"team"},
		"Tags.member.2.Value":   {"platform"},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// GetTopicAttributes verifies topic still exists with ARN.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, snsReq(t, "GetTopicAttributes", url.Values{"TopicArn": {topicArn}}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetTopicAttributes after tag: expected 200, got %d\nbody: %s", wg.Code, wg.Body.String())
	}

	// UntagResource
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, snsReq(t, "UntagResource", url.Values{
		"ResourceArn":       {topicArn},
		"TagKeys.member.1":  {"env"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}
}
