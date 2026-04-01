package elasticloadbalancing_test

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	elbsvc "github.com/neureaux/cloudmock/services/elasticloadbalancing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newELBGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(elbsvc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func elbReq(t *testing.T, params map[string]string) *http.Request {
	t.Helper()
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	body := vals.Encode()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/elasticloadbalancing/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// xmlResponse is a generic XML response structure for parsing.
type xmlResponse struct {
	XMLName xml.Name
	Inner   string `xml:",innerxml"`
}

func responseBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return w.Body.String()
}

// ---- CreateLoadBalancer ----

func TestELB_CreateLoadBalancer(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action":            "CreateLoadBalancer",
		"Name":              "test-lb",
		"Subnets.member.1":  "subnet-111",
		"Subnets.member.2":  "subnet-222",
	}))
	require.Equal(t, http.StatusOK, w.Code, responseBody(t, w))
	body := responseBody(t, w)
	assert.Contains(t, body, "test-lb")
	assert.Contains(t, body, "LoadBalancerArn")
	assert.Contains(t, body, "active")
}

func TestELB_CreateLoadBalancer_Duplicate(t *testing.T) {
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dup-lb",
	}))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dup-lb",
	}))
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, responseBody(t, w2), "DuplicateLoadBalancerName")
}

func TestELB_CreateLoadBalancer_MissingName(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
	}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, responseBody(t, w), "Name is required")
}

// ---- DescribeLoadBalancers ----

func TestELB_DescribeLoadBalancers(t *testing.T) {
	h := newELBGateway(t)
	// Create two LBs
	for _, name := range []string{"lb-1", "lb-2"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, elbReq(t, map[string]string{
			"Action": "CreateLoadBalancer",
			"Name":   name,
		}))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action": "DescribeLoadBalancers",
	}))
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "lb-1")
	assert.Contains(t, body, "lb-2")
}

func TestELB_DescribeLoadBalancers_ByName(t *testing.T) {
	h := newELBGateway(t)
	for _, name := range []string{"filter-1", "filter-2"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, elbReq(t, map[string]string{
			"Action": "CreateLoadBalancer",
			"Name":   name,
		}))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action":         "DescribeLoadBalancers",
		"Names.member.1": "filter-1",
	}))
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "filter-1")
	assert.NotContains(t, body, "filter-2")
}

// ---- DeleteLoadBalancer ----

func TestELB_DeleteLoadBalancer(t *testing.T) {
	h := newELBGateway(t)
	// Create
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "del-lb",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	// Extract ARN from response
	body := responseBody(t, w1)
	arn := extractXMLValue(t, body, "LoadBalancerArn")

	// Delete
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":          "DeleteLoadBalancer",
		"LoadBalancerArn": arn,
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify gone by listing
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action": "DescribeLoadBalancers",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	assert.NotContains(t, responseBody(t, w3), "del-lb")
}

func TestELB_DeleteLoadBalancer_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action":          "DeleteLoadBalancer",
		"LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/nonexistent/1234",
	}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_DeleteLoadBalancer_MissingArn(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action": "DeleteLoadBalancer",
	}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- CreateTargetGroup ----

func TestELB_CreateTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "test-tg",
		"Protocol": "HTTP",
		"Port":     "80",
		"VpcId":    "vpc-123",
	}))
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "test-tg")
	assert.Contains(t, body, "TargetGroupArn")
}

func TestELB_CreateTargetGroup_Duplicate(t *testing.T) {
	h := newELBGateway(t)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, elbReq(t, map[string]string{
			"Action":   "CreateTargetGroup",
			"Name":     "dup-tg",
			"Protocol": "HTTP",
			"Port":     "80",
		}))
		if i == 0 {
			require.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, responseBody(t, w), "DuplicateTargetGroupName")
		}
	}
}

// ---- DescribeTargetGroups ----

func TestELB_DescribeTargetGroups_List(t *testing.T) {
	h := newELBGateway(t)
	for _, name := range []string{"tg-a", "tg-b"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, elbReq(t, map[string]string{
			"Action":   "CreateTargetGroup",
			"Name":     name,
			"Protocol": "HTTP",
			"Port":     "80",
		}))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action": "DescribeTargetGroups",
	}))
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "tg-a")
	assert.Contains(t, body, "tg-b")
}

// ---- DeleteTargetGroup ----

func TestELB_DeleteTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "del-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":         "DeleteTargetGroup",
		"TargetGroupArn": arn,
	}))
	require.Equal(t, http.StatusOK, w2.Code)
}

// ---- RegisterTargets / DescribeTargetHealth ----

func TestELB_RegisterTargets_DescribeHealth(t *testing.T) {
	h := newELBGateway(t)
	// Create TG
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "health-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	// Register targets
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":                "RegisterTargets",
		"TargetGroupArn":       tgArn,
		"Targets.member.1.Id":  "i-111",
		"Targets.member.2.Id":  "i-222",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Describe health
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action":          "DescribeTargetHealth",
		"TargetGroupArn":  tgArn,
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "i-111")
	assert.Contains(t, body, "i-222")
	assert.Contains(t, body, "healthy")
}

// ---- Listener lifecycle ----

func TestELB_CreateListener(t *testing.T) {
	h := newELBGateway(t)
	// Create LB first
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "listener-lb",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Create listener
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":                            "CreateListener",
		"LoadBalancerArn":                   lbArn,
		"Protocol":                          "HTTP",
		"Port":                              "80",
		"DefaultActions.member.1.Type":      "forward",
		"DefaultActions.member.1.TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:000:targetgroup/tg/123",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "ListenerArn")
	assert.Contains(t, body, "HTTP")
}

func TestELB_CreateListener_LBNotFound(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action":          "CreateListener",
		"LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/nope/1234",
		"Protocol":        "HTTP",
		"Port":            "80",
	}))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Tags ----

func TestELB_AddTags_DescribeTags(t *testing.T) {
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "tag-lb",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Add tags
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":                "AddTags",
		"ResourceArns.member.1": lbArn,
		"Tags.member.1.Key":     "env",
		"Tags.member.1.Value":   "test",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Describe tags
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": lbArn,
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "env")
	assert.Contains(t, body, "test")
}

func TestELB_RemoveTags(t *testing.T) {
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "untag-lb",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Add then remove
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":                "AddTags",
		"ResourceArns.member.1": lbArn,
		"Tags.member.1.Key":     "remove-me",
		"Tags.member.1.Value":   "val",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action":                "RemoveTags",
		"ResourceArns.member.1": lbArn,
		"TagKeys.member.1":      "remove-me",
	}))
	require.Equal(t, http.StatusOK, w3.Code)

	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, elbReq(t, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": lbArn,
	}))
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, responseBody(t, w4), "remove-me")
}

// ---- InvalidAction ----

func TestELB_InvalidAction(t *testing.T) {
	h := newELBGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, map[string]string{
		"Action": "NonExistentAction",
	}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, responseBody(t, w), "InvalidAction")
}

// ---- ModifyTargetGroup ----

func TestELB_ModifyTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "mod-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":                 "ModifyTargetGroup",
		"TargetGroupArn":        arn,
		"HealthCheckPath":       "/health",
		"HealthyThresholdCount": "3",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "/health")
}

// ---- Rule lifecycle ----

func TestELB_CreateRule_DescribeRules(t *testing.T) {
	h := newELBGateway(t)
	// Setup: LB -> Listener
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "rule-lb",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	}))
	require.Equal(t, http.StatusOK, w2.Code)
	listenerArn := extractXMLValue(t, responseBody(t, w2), "ListenerArn")

	// Create rule
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action":                            "CreateRule",
		"ListenerArn":                       listenerArn,
		"Priority":                          "10",
		"Conditions.member.1.Field":         "path-pattern",
		"Conditions.member.1.Values.member.1": "/api/*",
		"Actions.member.1.Type":             "forward",
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, responseBody(t, w3), "RuleArn")

	// Describe rules should include default + our rule
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, elbReq(t, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
	}))
	require.Equal(t, http.StatusOK, w4.Code)
	body := responseBody(t, w4)
	assert.Contains(t, body, "default")
	assert.Contains(t, body, "10")
}

// ---- DeregisterTargets ----

func TestELB_DeregisterTargets(t *testing.T) {
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "dereg-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":               "RegisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-aaa",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action":               "DeregisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-aaa",
	}))
	require.Equal(t, http.StatusOK, w3.Code)

	// Verify target gone
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, elbReq(t, map[string]string{
		"Action":         "DescribeTargetHealth",
		"TargetGroupArn": tgArn,
	}))
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, responseBody(t, w4), "i-aaa")
}

// ---- Behavioral: Health Check ----

func TestELB_TargetHealth_InitialToHealthy(t *testing.T) {
	// Without a health checker, targets should auto-promote from initial to healthy on read
	h := newELBGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, elbReq(t, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "hc-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	}))
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	// Register targets
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, elbReq(t, map[string]string{
		"Action":               "RegisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-hc1",
		"Targets.member.2.Id": "i-hc2",
	}))
	require.Equal(t, http.StatusOK, w2.Code)

	// Health should be "healthy" on describe (auto-promote from initial)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, elbReq(t, map[string]string{
		"Action":         "DescribeTargetHealth",
		"TargetGroupArn": tgArn,
	}))
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "healthy")
	assert.Contains(t, body, "i-hc1")
	assert.Contains(t, body, "i-hc2")
}

func TestELB_ConnectionDraining_DeregisterTargets(t *testing.T) {
	// Test that draining state is available in store
	svc := elbsvc.New("123456789012", "us-east-1")
	store := svc.GetStore()

	store.CreateTargetGroup("drain-tg", "HTTP", 80, "vpc-123", "instance", "/", "HTTP", "traffic-port")
	tgs := store.ListTargetGroups(nil, nil, "")
	require.Len(t, tgs, 1)
	tgARN := tgs[0].ARN

	store.RegisterTargets(tgARN, []elbsvc.Target{
		{ID: "i-drain1", Port: 80},
	})

	// Deregister with draining
	ok := store.DeregisterTargetsWithDraining(tgARN, []string{"i-drain1"})
	assert.True(t, ok)

	// Describe should show draining
	targets, ok := store.DescribeTargetHealth(tgARN)
	require.True(t, ok)
	require.Len(t, targets, 1)
	assert.Equal(t, "draining", targets[0].Health)
}

// ---- helpers ----

func extractXMLValue(t *testing.T, xmlBody, tag string) string {
	t.Helper()
	start := strings.Index(xmlBody, "<"+tag+">")
	if start == -1 {
		t.Fatalf("tag <%s> not found in XML body:\n%s", tag, xmlBody)
	}
	start += len("<" + tag + ">")
	end := strings.Index(xmlBody[start:], "</"+tag+">")
	if end == -1 {
		t.Fatalf("closing tag </%s> not found in XML body:\n%s", tag, xmlBody)
	}
	return xmlBody[start : start+end]
}
