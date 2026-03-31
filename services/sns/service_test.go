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
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
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

// newSNSGatewayWithIAM builds a gateway with IAM enforce mode for AccessDenied tests.
func newSNSGatewayWithIAM(t *testing.T) (*gateway.Gateway, *iampkg.Store, *iampkg.Engine) {
	t.Helper()

	cfg := config.Default()
	cfg.IAM.Mode = "enforce"

	store := iampkg.NewStore(cfg.AccountID)
	if err := store.InitRoot("ROOTKEY", "ROOTSECRET"); err != nil {
		t.Fatalf("InitRoot: %v", err)
	}

	engine := iampkg.NewEngine()

	reg := routing.NewRegistry()
	reg.Register(snssvc.New(cfg.AccountID, cfg.Region))

	gw := gateway.NewWithIAM(cfg, reg, store, engine)
	return gw, store, engine
}

// snsReqWithKey builds a form-encoded POST request with a specific access key ID.
func snsReqWithKey(t *testing.T, action string, extra url.Values, accessKeyID string) *http.Request {
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
		"AWS4-HMAC-SHA256 Credential="+accessKeyID+"/20240101/us-east-1/sns/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// ---- Test 9: Subscribe with filter policy (attribute-based) ----

func TestSNS_SubscribeWithFilterPolicy(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "filter-topic")

	// Subscribe with a filter policy passed as an Attribute.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "Subscribe", url.Values{
		"TopicArn": {topicArn},
		"Protocol": {"sqs"},
		"Endpoint": {"arn:aws:sqs:us-east-1:000000000000:filtered-queue"},
		"Attributes.entry.1.key":   {"FilterPolicy"},
		"Attributes.entry.1.value": {`{"event":["order_created"]}`},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("Subscribe with filter policy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			SubscriptionArn string `xml:"SubscriptionArn"`
		} `xml:"SubscribeResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Subscribe with filter policy: unmarshal: %v", err)
	}
	if resp.Result.SubscriptionArn == "" {
		t.Error("Subscribe with filter policy: SubscriptionArn is empty")
	}

	// Subscribe to non-existent topic with filter — should fail.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, snsReq(t, "Subscribe", url.Values{
		"TopicArn": {"arn:aws:sns:us-east-1:000000000000:no-such-topic"},
		"Protocol": {"sqs"},
		"Endpoint": {"arn:aws:sqs:us-east-1:000000000000:some-queue"},
		"Attributes.entry.1.key":   {"FilterPolicy"},
		"Attributes.entry.1.value": {`{"event":["x"]}`},
	}))
	if wf.Code == http.StatusOK {
		t.Error("Subscribe with filter to non-existent topic: expected error, got 200")
	}
}

// ---- Test 10: Publish with MessageAttributes ----

func TestSNS_PublishWithMessageAttributes(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "msgattr-topic")

	// Publish with MessageAttributes.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "Publish", url.Values{
		"TopicArn":                                        {topicArn},
		"Message":                                         {"order event"},
		"MessageAttributes.entry.1.Name":                  {"event_type"},
		"MessageAttributes.entry.1.Value.DataType":        {"String"},
		"MessageAttributes.entry.1.Value.StringValue":     {"order_created"},
		"MessageAttributes.entry.2.Name":                  {"priority"},
		"MessageAttributes.entry.2.Value.DataType":        {"String"},
		"MessageAttributes.entry.2.Value.StringValue":     {"high"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("Publish with MessageAttributes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"PublishResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Publish with MessageAttributes: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("Publish with MessageAttributes: MessageId is empty")
	}

	// Publish with TargetArn instead of TopicArn — should also work.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, snsReq(t, "Publish", url.Values{
		"TargetArn": {topicArn},
		"Message":   {"via TargetArn"},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("Publish via TargetArn: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// Publish with empty message — should fail.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, snsReq(t, "Publish", url.Values{
		"TopicArn": {topicArn},
	}))
	if we.Code == http.StatusOK {
		t.Error("Publish with empty message: expected error, got 200")
	}
}

// ---- Test 11: ListSubscriptionsByTopic with multiple subscriptions ----

func TestSNS_ListSubscriptionsByTopicMultiple(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "multi-sub-topic")

	// Create a second topic to verify filtering works.
	otherArn := mustCreateTopic(t, handler, "other-topic")
	mustSubscribe(t, handler, otherArn, "email", "other@example.com")

	// Subscribe 3 endpoints to the main topic.
	sub1 := mustSubscribe(t, handler, topicArn, "sqs", "arn:aws:sqs:us-east-1:000000000000:queue-1")
	sub2 := mustSubscribe(t, handler, topicArn, "https", "https://example.com/hook")
	sub3 := mustSubscribe(t, handler, topicArn, "email", "user@example.com")

	// ListSubscriptionsByTopic.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "ListSubscriptionsByTopic", url.Values{"TopicArn": {topicArn}}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSubscriptionsByTopic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Verify all 3 subscriptions appear.
	for _, arn := range []string{sub1, sub2, sub3} {
		if !strings.Contains(body, arn) {
			t.Errorf("ListSubscriptionsByTopic: expected %s in response\nbody: %s", arn, body)
		}
	}

	// Verify subscription protocols are present.
	for _, proto := range []string{"sqs", "https", "email"} {
		if !strings.Contains(body, proto) {
			t.Errorf("ListSubscriptionsByTopic: expected protocol %s in response\nbody: %s", proto, body)
		}
	}

	// Verify the "other-topic" subscription does NOT appear.
	if strings.Contains(body, "other@example.com") {
		t.Errorf("ListSubscriptionsByTopic: should not contain subscription from other topic\nbody: %s", body)
	}

	// ListSubscriptionsByTopic for non-existent topic — should fail.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, snsReq(t, "ListSubscriptionsByTopic", url.Values{
		"TopicArn": {"arn:aws:sns:us-east-1:000000000000:no-such-topic"},
	}))
	if wf.Code == http.StatusOK {
		t.Error("ListSubscriptionsByTopic for non-existent topic: expected error, got 200")
	}
}

// ---- Test 12: ConfirmSubscription (unimplemented — returns InvalidAction) ----

func TestSNS_ConfirmSubscription(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "confirm-topic")

	// ConfirmSubscription is not implemented — should return 400 InvalidAction.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "ConfirmSubscription", url.Values{
		"TopicArn": {topicArn},
		"Token":    {"fake-confirmation-token"},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ConfirmSubscription: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "InvalidAction") {
		t.Errorf("ConfirmSubscription: expected InvalidAction in response\nbody: %s", body)
	}
}

// ---- Test 13: SetSubscriptionAttributes / GetSubscriptionAttributes (unimplemented) ----

func TestSNS_SubscriptionAttributes(t *testing.T) {
	handler := newSNSGateway(t)

	// SetSubscriptionAttributes is not implemented — should return 400 InvalidAction.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "SetSubscriptionAttributes", url.Values{
		"SubscriptionArn": {"arn:aws:sns:us-east-1:000000000000:topic:sub-id"},
		"AttributeName":   {"FilterPolicy"},
		"AttributeValue":  {`{"event":["x"]}`},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("SetSubscriptionAttributes: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "InvalidAction") {
		t.Errorf("SetSubscriptionAttributes: expected InvalidAction\nbody: %s", w.Body.String())
	}

	// GetSubscriptionAttributes is not implemented — should return 400 InvalidAction.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, snsReq(t, "GetSubscriptionAttributes", url.Values{
		"SubscriptionArn": {"arn:aws:sns:us-east-1:000000000000:topic:sub-id"},
	}))
	if wg.Code != http.StatusBadRequest {
		t.Fatalf("GetSubscriptionAttributes: expected 400, got %d\nbody: %s", wg.Code, wg.Body.String())
	}
	if !strings.Contains(wg.Body.String(), "InvalidAction") {
		t.Errorf("GetSubscriptionAttributes: expected InvalidAction\nbody: %s", wg.Body.String())
	}
}

// ---- Test 14: Error — TopicNotFound for multiple operations ----

func TestSNS_TopicNotFound(t *testing.T) {
	handler := newSNSGateway(t)
	fakeArn := "arn:aws:sns:us-east-1:000000000000:nonexistent-topic"

	tests := []struct {
		action string
		params url.Values
	}{
		{"GetTopicAttributes", url.Values{"TopicArn": {fakeArn}}},
		{"SetTopicAttributes", url.Values{
			"TopicArn":       {fakeArn},
			"AttributeName":  {"DisplayName"},
			"AttributeValue": {"x"},
		}},
		{"Subscribe", url.Values{
			"TopicArn": {fakeArn},
			"Protocol": {"sqs"},
			"Endpoint": {"arn:aws:sqs:us-east-1:000000000000:q"},
		}},
		{"Publish", url.Values{
			"TopicArn": {fakeArn},
			"Message":  {"hello"},
		}},
		{"ListSubscriptionsByTopic", url.Values{"TopicArn": {fakeArn}}},
		{"DeleteTopic", url.Values{"TopicArn": {fakeArn}}},
		{"TagResource", url.Values{
			"ResourceArn":         {fakeArn},
			"Tags.member.1.Key":   {"k"},
			"Tags.member.1.Value": {"v"},
		}},
		{"UntagResource", url.Values{
			"ResourceArn":      {fakeArn},
			"TagKeys.member.1": {"k"},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.action, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, snsReq(t, tc.action, tc.params))
			if w.Code == http.StatusOK {
				t.Errorf("%s on non-existent topic: expected error, got 200\nbody: %s", tc.action, w.Body.String())
			}
			body := w.Body.String()
			if !strings.Contains(body, "NotFound") && !strings.Contains(body, "not exist") {
				t.Errorf("%s: expected NotFound error\nbody: %s", tc.action, body)
			}
		})
	}
}

// ---- Test 15: Error — SubscriptionNotFound ----

func TestSNS_SubscriptionNotFound(t *testing.T) {
	handler := newSNSGateway(t)

	// Unsubscribe with a non-existent subscription ARN.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "Unsubscribe", url.Values{
		"SubscriptionArn": {"arn:aws:sns:us-east-1:000000000000:topic:nonexistent-sub-id"},
	}))
	if w.Code == http.StatusOK {
		t.Error("Unsubscribe non-existent: expected error, got 200")
	}
	body := w.Body.String()
	if !strings.Contains(body, "NotFound") && !strings.Contains(body, "not exist") {
		t.Errorf("Unsubscribe non-existent: expected NotFound in response\nbody: %s", body)
	}
}

// ---- Test 16: Error — InvalidParameter (missing required fields) ----

func TestSNS_InvalidParameter(t *testing.T) {
	handler := newSNSGateway(t)

	tests := []struct {
		name   string
		action string
		params url.Values
	}{
		{"CreateTopic_MissingName", "CreateTopic", url.Values{}},
		{"DeleteTopic_MissingArn", "DeleteTopic", url.Values{}},
		{"GetTopicAttributes_MissingArn", "GetTopicAttributes", url.Values{}},
		{"SetTopicAttributes_MissingArn", "SetTopicAttributes", url.Values{
			"AttributeName": {"DisplayName"},
		}},
		{"SetTopicAttributes_MissingAttrName", "SetTopicAttributes", url.Values{
			"TopicArn": {"arn:aws:sns:us-east-1:000000000000:some-topic"},
		}},
		{"Subscribe_MissingTopicArn", "Subscribe", url.Values{
			"Protocol": {"sqs"},
		}},
		{"Subscribe_MissingProtocol", "Subscribe", url.Values{
			"TopicArn": {"arn:aws:sns:us-east-1:000000000000:some-topic"},
		}},
		{"Unsubscribe_MissingArn", "Unsubscribe", url.Values{}},
		{"Publish_MissingTopicAndTarget", "Publish", url.Values{
			"Message": {"hello"},
		}},
		{"Publish_MissingMessage", "Publish", url.Values{
			"TopicArn": {"arn:aws:sns:us-east-1:000000000000:some-topic"},
		}},
		{"TagResource_MissingArn", "TagResource", url.Values{}},
		{"UntagResource_MissingArn", "UntagResource", url.Values{}},
		{"ListSubscriptionsByTopic_MissingArn", "ListSubscriptionsByTopic", url.Values{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, snsReq(t, tc.action, tc.params))
			if w.Code == http.StatusOK {
				t.Errorf("%s: expected error for missing parameter, got 200\nbody: %s", tc.name, w.Body.String())
			}
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d\nbody: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

// ---- Test 17: Error — AuthorizationError (IAM enforcement) ----

func TestSNS_AccessDenied_WithIAMEnforcement(t *testing.T) {
	gw, store, _ := newSNSGatewayWithIAM(t)

	// Create a user with no policies — should be denied.
	if _, err := store.CreateUser("noperm-sns"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	key, err := store.CreateAccessKey("noperm-sns")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}

	// CreateTopic with unprivileged user — should get 403.
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, snsReqWithKey(t, "CreateTopic", url.Values{"Name": {"denied-topic"}}, key.AccessKeyID))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AccessDenied, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "AccessDenied") {
		t.Errorf("error response should contain 'AccessDenied'\nbody: %s", body)
	}
}

// ---- Test 18: Error — AuthorizationError (invalid credentials) ----

func TestSNS_AccessDenied_InvalidCredential(t *testing.T) {
	gw, _, _ := newSNSGatewayWithIAM(t)

	// Use a completely unknown access key.
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, snsReqWithKey(t, "ListTopics", nil, "AKIAFAKEUNKNOWNKEY"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 19: Subscribe multiple protocols and verify ListSubscriptions ----

func TestSNS_SubscribeMultipleProtocols(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "proto-topic")

	protocols := []struct {
		proto    string
		endpoint string
	}{
		{"sqs", "arn:aws:sqs:us-east-1:000000000000:queue-proto"},
		{"https", "https://hooks.example.com/sns"},
		{"email", "user@example.com"},
		{"lambda", "arn:aws:lambda:us-east-1:000000000000:function:my-func"},
	}

	subArns := make([]string, 0, len(protocols))
	for _, p := range protocols {
		arn := mustSubscribe(t, handler, topicArn, p.proto, p.endpoint)
		subArns = append(subArns, arn)
	}

	// All subscriptions should appear in ListSubscriptions.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "ListSubscriptions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSubscriptions: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, arn := range subArns {
		if !strings.Contains(body, arn) {
			t.Errorf("ListSubscriptions: missing subscription %s\nbody: %s", arn, body)
		}
	}

	// Verify each protocol name is in the response.
	for _, p := range protocols {
		if !strings.Contains(body, p.proto) {
			t.Errorf("ListSubscriptions: missing protocol %s\nbody: %s", p.proto, body)
		}
	}
}

// ---- Test 20: Publish to topic with TargetArn (alternate field) ----

func TestSNS_PublishWithTargetArn(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "target-arn-topic")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "Publish", url.Values{
		"TargetArn": {topicArn},
		"Message":   {"via TargetArn field"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("Publish via TargetArn: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			MessageId string `xml:"MessageId"`
		} `xml:"PublishResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Publish via TargetArn: unmarshal: %v", err)
	}
	if resp.Result.MessageId == "" {
		t.Error("Publish via TargetArn: MessageId is empty")
	}

	// Neither TopicArn nor TargetArn — should fail.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, snsReq(t, "Publish", url.Values{
		"Message": {"no topic or target"},
	}))
	if wf.Code == http.StatusOK {
		t.Error("Publish without TopicArn or TargetArn: expected error, got 200")
	}
}

// ---- Test 21: DeleteTopic cascades subscription removal ----

func TestSNS_DeleteTopicCascadesSubscriptions(t *testing.T) {
	handler := newSNSGateway(t)
	topicArn := mustCreateTopic(t, handler, "cascade-topic")

	// Add multiple subscriptions.
	mustSubscribe(t, handler, topicArn, "sqs", "arn:aws:sqs:us-east-1:000000000000:q1")
	mustSubscribe(t, handler, topicArn, "sqs", "arn:aws:sqs:us-east-1:000000000000:q2")
	mustSubscribe(t, handler, topicArn, "email", "cascade@example.com")

	// Verify subscriptions exist.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, snsReq(t, "ListSubscriptions", nil))
	if !strings.Contains(wl.Body.String(), "cascade-topic") {
		t.Fatal("subscriptions should exist before delete")
	}

	// Delete topic.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, snsReq(t, "DeleteTopic", url.Values{"TopicArn": {topicArn}}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteTopic: expected 200, got %d", wd.Code)
	}

	// All subscriptions should be gone.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, snsReq(t, "ListSubscriptions", nil))
	if strings.Contains(wl2.Body.String(), "cascade-topic") {
		t.Error("ListSubscriptions: subscriptions should be removed after topic deletion")
	}
}

// ---- Test 22: CreateTopic with attributes and tags ----

func TestSNS_CreateTopicWithAttributesAndTags(t *testing.T) {
	handler := newSNSGateway(t)

	// Create topic with attributes and tags.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, snsReq(t, "CreateTopic", url.Values{
		"Name":                         {"full-topic"},
		"Attributes.entry.1.key":       {"DisplayName"},
		"Attributes.entry.1.value":     {"Full Display"},
		"Tags.member.1.Key":            {"env"},
		"Tags.member.1.Value":          {"production"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTopic with attrs/tags: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			TopicArn string `xml:"TopicArn"`
		} `xml:"CreateTopicResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateTopic: unmarshal: %v", err)
	}
	topicArn := resp.Result.TopicArn

	// Verify attribute persisted.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, snsReq(t, "GetTopicAttributes", url.Values{"TopicArn": {topicArn}}))
	if wg.Code != http.StatusOK {
		t.Fatalf("GetTopicAttributes: expected 200, got %d", wg.Code)
	}
	if !strings.Contains(wg.Body.String(), "Full Display") {
		t.Errorf("GetTopicAttributes: expected DisplayName 'Full Display'\nbody: %s", wg.Body.String())
	}
}
