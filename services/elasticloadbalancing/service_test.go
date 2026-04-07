package elasticloadbalancing_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	elbsvc "github.com/Viridian-Inc/cloudmock/services/elasticloadbalancing"
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

func responseBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return w.Body.String()
}

func doReq(t *testing.T, h http.Handler, params map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, elbReq(t, params))
	return w
}

// ============================================================
// Load Balancer Tests
// ============================================================

func TestELB_CreateLoadBalancer(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":           "CreateLoadBalancer",
		"Name":             "test-lb",
		"Subnets.member.1": "subnet-111",
		"Subnets.member.2": "subnet-222",
	})
	require.Equal(t, http.StatusOK, w.Code, responseBody(t, w))
	body := responseBody(t, w)
	assert.Contains(t, body, "test-lb")
	assert.Contains(t, body, "LoadBalancerArn")
	assert.Contains(t, body, "active")
	assert.Contains(t, body, "application")
	assert.Contains(t, body, "internet-facing")
}

func TestELB_CreateLoadBalancer_NLB(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":           "CreateLoadBalancer",
		"Name":             "nlb-test",
		"Type":             "network",
		"Scheme":           "internal",
		"Subnets.member.1": "subnet-333",
	})
	require.Equal(t, http.StatusOK, w.Code, responseBody(t, w))
	body := responseBody(t, w)
	assert.Contains(t, body, "nlb-test")
	assert.Contains(t, body, "network")
	assert.Contains(t, body, "internal")
	assert.Contains(t, body, "loadbalancer/net/")
}

func TestELB_CreateLoadBalancer_GWLB(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":           "CreateLoadBalancer",
		"Name":             "gwlb-test",
		"Type":             "gateway",
		"Subnets.member.1": "subnet-444",
	})
	require.Equal(t, http.StatusOK, w.Code, responseBody(t, w))
	body := responseBody(t, w)
	assert.Contains(t, body, "gwlb-test")
	assert.Contains(t, body, "gateway")
	assert.Contains(t, body, "loadbalancer/gwy/")
}

func TestELB_CreateLoadBalancer_WithTags(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":              "CreateLoadBalancer",
		"Name":                "tagged-lb",
		"Tags.member.1.Key":   "env",
		"Tags.member.1.Value": "prod",
	})
	require.Equal(t, http.StatusOK, w.Code)
	lbArn := extractXMLValue(t, responseBody(t, w), "LoadBalancerArn")

	// Verify tags
	w2 := doReq(t, h, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": lbArn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "env")
	assert.Contains(t, body, "prod")
}

func TestELB_CreateLoadBalancer_Duplicate(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dup-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dup-lb",
	})
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, responseBody(t, w2), "DuplicateLoadBalancerName")
}

func TestELB_CreateLoadBalancer_MissingName(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, responseBody(t, w), "Name is required")
}

// ---- DescribeLoadBalancers ----

func TestELB_DescribeLoadBalancers(t *testing.T) {
	h := newELBGateway(t)
	for _, name := range []string{"lb-1", "lb-2"} {
		w := doReq(t, h, map[string]string{
			"Action": "CreateLoadBalancer",
			"Name":   name,
		})
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := doReq(t, h, map[string]string{
		"Action": "DescribeLoadBalancers",
	})
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "lb-1")
	assert.Contains(t, body, "lb-2")
}

func TestELB_DescribeLoadBalancers_ByName(t *testing.T) {
	h := newELBGateway(t)
	for _, name := range []string{"filter-1", "filter-2"} {
		w := doReq(t, h, map[string]string{
			"Action": "CreateLoadBalancer",
			"Name":   name,
		})
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := doReq(t, h, map[string]string{
		"Action":         "DescribeLoadBalancers",
		"Names.member.1": "filter-1",
	})
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "filter-1")
	assert.NotContains(t, body, "filter-2")
}

func TestELB_DescribeLoadBalancers_ByARN(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "arn-filter-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                    "DescribeLoadBalancers",
		"LoadBalancerArns.member.1": arn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, responseBody(t, w2), "arn-filter-lb")
}

func TestELB_DescribeLoadBalancers_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":         "DescribeLoadBalancers",
		"Names.member.1": "nonexistent",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, responseBody(t, w), "LoadBalancerNotFound")
}

func TestELB_DescribeLoadBalancers_Pagination(t *testing.T) {
	h := newELBGateway(t)
	// Create 5 LBs
	for i := 0; i < 5; i++ {
		w := doReq(t, h, map[string]string{
			"Action": "CreateLoadBalancer",
			"Name":   "page-lb-" + string(rune('a'+i)),
		})
		require.Equal(t, http.StatusOK, w.Code)
	}

	// Page 1: 2 items
	w1 := doReq(t, h, map[string]string{
		"Action":   "DescribeLoadBalancers",
		"PageSize": "2",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	body1 := responseBody(t, w1)
	marker := extractXMLValue(t, body1, "NextMarker")
	assert.NotEmpty(t, marker)

	// Page 2: next 2 items
	w2 := doReq(t, h, map[string]string{
		"Action":   "DescribeLoadBalancers",
		"PageSize": "2",
		"Marker":   marker,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body2 := responseBody(t, w2)
	marker2 := extractXMLValue(t, body2, "NextMarker")
	assert.NotEmpty(t, marker2)

	// Page 3: last item, no next marker
	w3 := doReq(t, h, map[string]string{
		"Action":   "DescribeLoadBalancers",
		"PageSize": "2",
		"Marker":   marker2,
	})
	require.Equal(t, http.StatusOK, w3.Code)
	body3 := responseBody(t, w3)
	assert.NotContains(t, body3, "<NextMarker>")
}

// ---- DeleteLoadBalancer ----

func TestELB_DeleteLoadBalancer(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "del-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":          "DeleteLoadBalancer",
		"LoadBalancerArn": arn,
	})
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify gone
	w3 := doReq(t, h, map[string]string{
		"Action": "DescribeLoadBalancers",
	})
	require.Equal(t, http.StatusOK, w3.Code)
	assert.NotContains(t, responseBody(t, w3), "del-lb")
}

func TestELB_DeleteLoadBalancer_CascadesListeners(t *testing.T) {
	h := newELBGateway(t)
	// Create LB
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "cascade-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Create listener
	w2 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	require.Equal(t, http.StatusOK, w2.Code)

	// Delete LB
	w3 := doReq(t, h, map[string]string{
		"Action":          "DeleteLoadBalancer",
		"LoadBalancerArn": lbArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)

	// Listeners for this LB should be empty
	w4 := doReq(t, h, map[string]string{
		"Action":          "DescribeListeners",
		"LoadBalancerArn": lbArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, responseBody(t, w4), "ListenerArn")
}

func TestELB_DeleteLoadBalancer_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":          "DeleteLoadBalancer",
		"LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/nonexistent/1234",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_DeleteLoadBalancer_MissingArn(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action": "DeleteLoadBalancer",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- DescribeLoadBalancerAttributes ----

func TestELB_DescribeLoadBalancerAttributes(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "attrs-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":          "DescribeLoadBalancerAttributes",
		"LoadBalancerArn": arn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "deletion_protection.enabled")
	assert.Contains(t, body, "idle_timeout.timeout_seconds")
}

func TestELB_DescribeLoadBalancerAttributes_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":          "DescribeLoadBalancerAttributes",
		"LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/nope/1234",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- ModifyLoadBalancerAttributes ----

func TestELB_ModifyLoadBalancerAttributes(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "mod-attrs-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                   "ModifyLoadBalancerAttributes",
		"LoadBalancerArn":          arn,
		"Attributes.member.1.Key":   "idle_timeout.timeout_seconds",
		"Attributes.member.1.Value": "120",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "idle_timeout.timeout_seconds")
	assert.Contains(t, body, "120")
}

// ---- SetSecurityGroups ----

func TestELB_SetSecurityGroups(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":                   "CreateLoadBalancer",
		"Name":                     "sg-lb",
		"SecurityGroups.member.1":  "sg-old",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                  "SetSecurityGroups",
		"LoadBalancerArn":         arn,
		"SecurityGroups.member.1": "sg-new1",
		"SecurityGroups.member.2": "sg-new2",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "sg-new1")
	assert.Contains(t, body, "sg-new2")
}

func TestELB_SetSecurityGroups_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":                  "SetSecurityGroups",
		"LoadBalancerArn":         "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/nope/1234",
		"SecurityGroups.member.1": "sg-1",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- SetSubnets ----

func TestELB_SetSubnets(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":           "CreateLoadBalancer",
		"Name":             "subnets-lb",
		"Subnets.member.1": "subnet-old",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":           "SetSubnets",
		"LoadBalancerArn":  arn,
		"Subnets.member.1": "subnet-new1",
		"Subnets.member.2": "subnet-new2",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "subnet-new1")
	assert.Contains(t, body, "subnet-new2")
}

// ============================================================
// Target Group Tests
// ============================================================

func TestELB_CreateTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "test-tg",
		"Protocol": "HTTP",
		"Port":     "80",
		"VpcId":    "vpc-123",
	})
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "test-tg")
	assert.Contains(t, body, "TargetGroupArn")
	assert.Contains(t, body, "instance") // default target type
}

func TestELB_CreateTargetGroup_Duplicate(t *testing.T) {
	h := newELBGateway(t)
	for i := 0; i < 2; i++ {
		w := doReq(t, h, map[string]string{
			"Action":   "CreateTargetGroup",
			"Name":     "dup-tg",
			"Protocol": "HTTP",
			"Port":     "80",
		})
		if i == 0 {
			require.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, responseBody(t, w), "DuplicateTargetGroupName")
		}
	}
}

func TestELB_CreateTargetGroup_MissingName(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- DescribeTargetGroups ----

func TestELB_DescribeTargetGroups_List(t *testing.T) {
	h := newELBGateway(t)
	for _, name := range []string{"tg-a", "tg-b"} {
		w := doReq(t, h, map[string]string{
			"Action":   "CreateTargetGroup",
			"Name":     name,
			"Protocol": "HTTP",
			"Port":     "80",
		})
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := doReq(t, h, map[string]string{
		"Action": "DescribeTargetGroups",
	})
	require.Equal(t, http.StatusOK, w.Code)
	body := responseBody(t, w)
	assert.Contains(t, body, "tg-a")
	assert.Contains(t, body, "tg-b")
}

func TestELB_DescribeTargetGroups_ByLB(t *testing.T) {
	h := newELBGateway(t)
	// Create LB
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "tg-lb-filter",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Create TGs
	w2 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "tg-linked",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	tgArn := extractXMLValue(t, responseBody(t, w2), "TargetGroupArn")

	w3 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "tg-unlinked",
		"Protocol": "HTTP",
		"Port":     "8080",
	})
	require.Equal(t, http.StatusOK, w3.Code)

	// Create listener referencing first TG
	doReq(t, h, map[string]string{
		"Action":                              "CreateListener",
		"LoadBalancerArn":                     lbArn,
		"Protocol":                            "HTTP",
		"Port":                                "80",
		"DefaultActions.member.1.Type":        "forward",
		"DefaultActions.member.1.TargetGroupArn": tgArn,
	})

	// Filter by LB
	w4 := doReq(t, h, map[string]string{
		"Action":          "DescribeTargetGroups",
		"LoadBalancerArn": lbArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	body := responseBody(t, w4)
	assert.Contains(t, body, "tg-linked")
	assert.NotContains(t, body, "tg-unlinked")
}

func TestELB_DescribeTargetGroups_Pagination(t *testing.T) {
	h := newELBGateway(t)
	for i := 0; i < 5; i++ {
		w := doReq(t, h, map[string]string{
			"Action":   "CreateTargetGroup",
			"Name":     "ptg-" + string(rune('a'+i)),
			"Protocol": "HTTP",
			"Port":     "80",
		})
		require.Equal(t, http.StatusOK, w.Code)
	}

	w1 := doReq(t, h, map[string]string{
		"Action":   "DescribeTargetGroups",
		"PageSize": "3",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	body1 := responseBody(t, w1)
	marker := extractXMLValue(t, body1, "NextMarker")
	assert.NotEmpty(t, marker)

	w2 := doReq(t, h, map[string]string{
		"Action":   "DescribeTargetGroups",
		"PageSize": "3",
		"Marker":   marker,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	assert.NotContains(t, responseBody(t, w2), "<NextMarker>")
}

func TestELB_DescribeTargetGroups_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":         "DescribeTargetGroups",
		"Names.member.1": "nonexistent",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, responseBody(t, w), "TargetGroupNotFound")
}

// ---- DeleteTargetGroup ----

func TestELB_DeleteTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "del-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := doReq(t, h, map[string]string{
		"Action":         "DeleteTargetGroup",
		"TargetGroupArn": arn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
}

func TestELB_DeleteTargetGroup_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":         "DeleteTargetGroup",
		"TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:000:targetgroup/nope/1234",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_DeleteTargetGroup_InUse(t *testing.T) {
	h := newELBGateway(t)
	// Create LB
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "inuse-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Create TG
	w2 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "inuse-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	tgArn := extractXMLValue(t, responseBody(t, w2), "TargetGroupArn")

	// Create listener referencing TG
	doReq(t, h, map[string]string{
		"Action":                                  "CreateListener",
		"LoadBalancerArn":                         lbArn,
		"Protocol":                                "HTTP",
		"Port":                                    "80",
		"DefaultActions.member.1.Type":            "forward",
		"DefaultActions.member.1.TargetGroupArn":  tgArn,
	})

	// Try to delete TG - should fail
	w3 := doReq(t, h, map[string]string{
		"Action":         "DeleteTargetGroup",
		"TargetGroupArn": tgArn,
	})
	assert.Equal(t, http.StatusBadRequest, w3.Code)
	assert.Contains(t, responseBody(t, w3), "ResourceInUse")
}

// ---- ModifyTargetGroup ----

func TestELB_ModifyTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "mod-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                 "ModifyTargetGroup",
		"TargetGroupArn":        arn,
		"HealthCheckPath":       "/health",
		"HealthyThresholdCount": "3",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "/health")
}

func TestELB_ModifyTargetGroup_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":          "ModifyTargetGroup",
		"TargetGroupArn":  "arn:aws:elasticloadbalancing:us-east-1:000:targetgroup/nope/1234",
		"HealthCheckPath": "/foo",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- DescribeTargetGroupAttributes ----

func TestELB_DescribeTargetGroupAttributes(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "tg-attrs",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := doReq(t, h, map[string]string{
		"Action":         "DescribeTargetGroupAttributes",
		"TargetGroupArn": arn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "deregistration_delay.timeout_seconds")
	assert.Contains(t, body, "300")
}

// ---- ModifyTargetGroupAttributes ----

func TestELB_ModifyTargetGroupAttributes(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "tg-mod-attrs",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	arn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                   "ModifyTargetGroupAttributes",
		"TargetGroupArn":           arn,
		"Attributes.member.1.Key":   "deregistration_delay.timeout_seconds",
		"Attributes.member.1.Value": "60",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "deregistration_delay.timeout_seconds")
	assert.Contains(t, body, "60")
}

// ============================================================
// Target Registration / Health Tests
// ============================================================

func TestELB_RegisterTargets_DescribeHealth(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "health-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	w2 := doReq(t, h, map[string]string{
		"Action":               "RegisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-111",
		"Targets.member.2.Id": "i-222",
	})
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := doReq(t, h, map[string]string{
		"Action":         "DescribeTargetHealth",
		"TargetGroupArn": tgArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "i-111")
	assert.Contains(t, body, "i-222")
	assert.Contains(t, body, "healthy")
}

func TestELB_RegisterTargets_TGNotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":               "RegisterTargets",
		"TargetGroupArn":      "arn:aws:elasticloadbalancing:us-east-1:000:targetgroup/nope/1234",
		"Targets.member.1.Id": "i-111",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_DeregisterTargets(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "dereg-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	doReq(t, h, map[string]string{
		"Action":               "RegisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-aaa",
	})

	w3 := doReq(t, h, map[string]string{
		"Action":               "DeregisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-aaa",
	})
	require.Equal(t, http.StatusOK, w3.Code)

	w4 := doReq(t, h, map[string]string{
		"Action":         "DescribeTargetHealth",
		"TargetGroupArn": tgArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, responseBody(t, w4), "i-aaa")
}

func TestELB_TargetHealth_InitialToHealthy(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "hc-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	doReq(t, h, map[string]string{
		"Action":               "RegisterTargets",
		"TargetGroupArn":      tgArn,
		"Targets.member.1.Id": "i-hc1",
		"Targets.member.2.Id": "i-hc2",
	})

	w3 := doReq(t, h, map[string]string{
		"Action":         "DescribeTargetHealth",
		"TargetGroupArn": tgArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "healthy")
	assert.Contains(t, body, "i-hc1")
	assert.Contains(t, body, "i-hc2")
}

func TestELB_ConnectionDraining_DeregisterTargets(t *testing.T) {
	svc := elbsvc.New("123456789012", "us-east-1")
	store := svc.GetStore()

	store.CreateTargetGroup("drain-tg", "HTTP", 80, "vpc-123", "instance", "/", "HTTP", "traffic-port", nil)
	tgs := store.ListTargetGroups(nil, nil, "")
	require.Len(t, tgs, 1)
	tgARN := tgs[0].ARN

	store.RegisterTargets(tgARN, []elbsvc.Target{
		{ID: "i-drain1", Port: 80},
	})

	ok := store.DeregisterTargetsWithDraining(tgARN, []string{"i-drain1"})
	assert.True(t, ok)

	targets, ok := store.DescribeTargetHealth(tgARN)
	require.True(t, ok)
	require.Len(t, targets, 1)
	assert.Equal(t, "draining", targets[0].Health)
}

// ============================================================
// Listener Tests
// ============================================================

func TestELB_CreateListener(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "listener-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                                  "CreateListener",
		"LoadBalancerArn":                         lbArn,
		"Protocol":                                "HTTP",
		"Port":                                    "80",
		"DefaultActions.member.1.Type":            "forward",
		"DefaultActions.member.1.TargetGroupArn":  "arn:aws:elasticloadbalancing:us-east-1:000:targetgroup/tg/123",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	assert.Contains(t, body, "ListenerArn")
	assert.Contains(t, body, "HTTP")
}

func TestELB_CreateListener_DuplicatePort(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dup-port-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// First listener on port 80
	w2 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	require.Equal(t, http.StatusOK, w2.Code)

	// Second listener on same port 80 should fail
	w3 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	assert.Equal(t, http.StatusBadRequest, w3.Code)
	assert.Contains(t, responseBody(t, w3), "DuplicateListener")
}

func TestELB_CreateListener_LBNotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":          "CreateListener",
		"LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/nope/1234",
		"Protocol":        "HTTP",
		"Port":            "80",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_DescribeListeners(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dl-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	// Create two listeners
	for _, port := range []string{"80", "443"} {
		doReq(t, h, map[string]string{
			"Action":                       "CreateListener",
			"LoadBalancerArn":              lbArn,
			"Protocol":                     "HTTP",
			"Port":                         port,
			"DefaultActions.member.1.Type": "forward",
		})
	}

	w2 := doReq(t, h, map[string]string{
		"Action":          "DescribeListeners",
		"LoadBalancerArn": lbArn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	body := responseBody(t, w2)
	// Should contain both listeners
	assert.Contains(t, body, "<Port>80</Port>")
	assert.Contains(t, body, "<Port>443</Port>")
}

func TestELB_DescribeListeners_Pagination(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "dl-page-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	for i := 0; i < 5; i++ {
		doReq(t, h, map[string]string{
			"Action":                       "CreateListener",
			"LoadBalancerArn":              lbArn,
			"Protocol":                     "HTTP",
			"Port":                         string(rune('1'+i)) + "000",
			"DefaultActions.member.1.Type": "forward",
		})
	}

	w2 := doReq(t, h, map[string]string{
		"Action":          "DescribeListeners",
		"LoadBalancerArn": lbArn,
		"PageSize":        "2",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	marker := extractXMLValue(t, responseBody(t, w2), "NextMarker")
	assert.NotEmpty(t, marker)
}

func TestELB_DeleteListener(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "del-l-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	lArn := extractXMLValue(t, responseBody(t, w2), "ListenerArn")

	w3 := doReq(t, h, map[string]string{
		"Action":      "DeleteListener",
		"ListenerArn": lArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)

	// Verify gone
	w4 := doReq(t, h, map[string]string{
		"Action":          "DescribeListeners",
		"LoadBalancerArn": lbArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, responseBody(t, w4), lArn)
}

func TestELB_DeleteListener_NotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":      "DeleteListener",
		"ListenerArn": "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/x/1/listener/abc",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_ModifyListener(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "ml-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	lArn := extractXMLValue(t, responseBody(t, w2), "ListenerArn")

	w3 := doReq(t, h, map[string]string{
		"Action":      "ModifyListener",
		"ListenerArn": lArn,
		"Port":        "8080",
	})
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, responseBody(t, w3), "<Port>8080</Port>")
}

// ============================================================
// Rule Tests
// ============================================================

func createListenerForRules(t *testing.T, h http.Handler) (string, string) {
	t.Helper()
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "rule-lb-" + randomTestID(),
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	listenerArn := extractXMLValue(t, responseBody(t, w2), "ListenerArn")
	return lbArn, listenerArn
}

func TestELB_CreateRule_DescribeRules(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	w3 := doReq(t, h, map[string]string{
		"Action":                                "CreateRule",
		"ListenerArn":                           listenerArn,
		"Priority":                              "10",
		"Conditions.member.1.Field":             "path-pattern",
		"Conditions.member.1.Values.member.1":   "/api/*",
		"Actions.member.1.Type":                 "forward",
	})
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, responseBody(t, w3), "RuleArn")

	// Describe rules should include default + our rule
	w4 := doReq(t, h, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	body := responseBody(t, w4)
	assert.Contains(t, body, "default")
	assert.Contains(t, body, "10")
}

func TestELB_CreateRule_DuplicatePriority(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	for i := 0; i < 2; i++ {
		w := doReq(t, h, map[string]string{
			"Action":                              "CreateRule",
			"ListenerArn":                         listenerArn,
			"Priority":                            "5",
			"Conditions.member.1.Field":           "path-pattern",
			"Conditions.member.1.Values.member.1": "/test",
			"Actions.member.1.Type":               "forward",
		})
		if i == 0 {
			require.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, responseBody(t, w), "PriorityInUse")
		}
	}
}

func TestELB_CreateRule_ListenerNotFound(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":                    "CreateRule",
		"ListenerArn":              "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/x/1/listener/nope",
		"Priority":                 "1",
		"Actions.member.1.Type":    "forward",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestELB_DeleteRule(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	w1 := doReq(t, h, map[string]string{
		"Action":                              "CreateRule",
		"ListenerArn":                         listenerArn,
		"Priority":                            "20",
		"Conditions.member.1.Field":           "host-header",
		"Conditions.member.1.Values.member.1": "example.com",
		"Actions.member.1.Type":               "forward",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	ruleArn := extractXMLValue(t, responseBody(t, w1), "RuleArn")

	w2 := doReq(t, h, map[string]string{
		"Action":  "DeleteRule",
		"RuleArn": ruleArn,
	})
	require.Equal(t, http.StatusOK, w2.Code)
}

func TestELB_DeleteRule_DefaultNotAllowed(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	// Get the default rule ARN
	w1 := doReq(t, h, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
	})
	require.Equal(t, http.StatusOK, w1.Code)
	// Find default rule ARN
	body := responseBody(t, w1)
	defaultRuleArn := extractXMLValue(t, body, "RuleArn")

	w2 := doReq(t, h, map[string]string{
		"Action":  "DeleteRule",
		"RuleArn": defaultRuleArn,
	})
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, responseBody(t, w2), "OperationNotPermitted")
}

func TestELB_ModifyRule(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	w1 := doReq(t, h, map[string]string{
		"Action":                              "CreateRule",
		"ListenerArn":                         listenerArn,
		"Priority":                            "30",
		"Conditions.member.1.Field":           "path-pattern",
		"Conditions.member.1.Values.member.1": "/old/*",
		"Actions.member.1.Type":               "forward",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	ruleArn := extractXMLValue(t, responseBody(t, w1), "RuleArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                              "ModifyRule",
		"RuleArn":                             ruleArn,
		"Conditions.member.1.Field":           "path-pattern",
		"Conditions.member.1.Values.member.1": "/new/*",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, responseBody(t, w2), "/new/*")
}

func TestELB_SetRulePriorities(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	w1 := doReq(t, h, map[string]string{
		"Action":                              "CreateRule",
		"ListenerArn":                         listenerArn,
		"Priority":                            "10",
		"Conditions.member.1.Field":           "path-pattern",
		"Conditions.member.1.Values.member.1": "/a",
		"Actions.member.1.Type":               "forward",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	ruleArn := extractXMLValue(t, responseBody(t, w1), "RuleArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                              "SetRulePriorities",
		"RulePriorities.member.1.RuleArn":     ruleArn,
		"RulePriorities.member.1.Priority":    "99",
	})
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify priority changed
	w3 := doReq(t, h, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, responseBody(t, w3), "99")
}

func TestELB_DescribeRules_Pagination(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	for i := 1; i <= 5; i++ {
		doReq(t, h, map[string]string{
			"Action":                              "CreateRule",
			"ListenerArn":                         listenerArn,
			"Priority":                            string(rune('0'+i)) + "0",
			"Conditions.member.1.Field":           "path-pattern",
			"Conditions.member.1.Values.member.1": "/" + string(rune('a'+i)),
			"Actions.member.1.Type":               "forward",
		})
	}

	// 5 rules + 1 default = 6 total. Page size 3.
	w1 := doReq(t, h, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
		"PageSize":    "3",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	marker := extractXMLValue(t, responseBody(t, w1), "NextMarker")
	assert.NotEmpty(t, marker)

	w2 := doReq(t, h, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
		"PageSize":    "3",
		"Marker":      marker,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	assert.NotContains(t, responseBody(t, w2), "<NextMarker>")
}

// ============================================================
// Tag Tests
// ============================================================

func TestELB_AddTags_DescribeTags(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "tag-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                "AddTags",
		"ResourceArns.member.1": lbArn,
		"Tags.member.1.Key":     "env",
		"Tags.member.1.Value":   "test",
	})
	require.Equal(t, http.StatusOK, w2.Code)

	w3 := doReq(t, h, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": lbArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "env")
	assert.Contains(t, body, "test")
}

func TestELB_RemoveTags(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "untag-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	doReq(t, h, map[string]string{
		"Action":                "AddTags",
		"ResourceArns.member.1": lbArn,
		"Tags.member.1.Key":     "remove-me",
		"Tags.member.1.Value":   "val",
	})

	doReq(t, h, map[string]string{
		"Action":                "RemoveTags",
		"ResourceArns.member.1": lbArn,
		"TagKeys.member.1":      "remove-me",
	})

	w4 := doReq(t, h, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": lbArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, responseBody(t, w4), "remove-me")
}

func TestELB_Tags_OnTargetGroup(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action":   "CreateTargetGroup",
		"Name":     "tag-tg",
		"Protocol": "HTTP",
		"Port":     "80",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	tgArn := extractXMLValue(t, responseBody(t, w1), "TargetGroupArn")

	doReq(t, h, map[string]string{
		"Action":                "AddTags",
		"ResourceArns.member.1": tgArn,
		"Tags.member.1.Key":     "team",
		"Tags.member.1.Value":   "platform",
	})

	w3 := doReq(t, h, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": tgArn,
	})
	require.Equal(t, http.StatusOK, w3.Code)
	body := responseBody(t, w3)
	assert.Contains(t, body, "team")
	assert.Contains(t, body, "platform")
}

func TestELB_Tags_OnListener(t *testing.T) {
	h := newELBGateway(t)
	w1 := doReq(t, h, map[string]string{
		"Action": "CreateLoadBalancer",
		"Name":   "ltag-lb",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	lbArn := extractXMLValue(t, responseBody(t, w1), "LoadBalancerArn")

	w2 := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              lbArn,
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	lArn := extractXMLValue(t, responseBody(t, w2), "ListenerArn")

	doReq(t, h, map[string]string{
		"Action":                "AddTags",
		"ResourceArns.member.1": lArn,
		"Tags.member.1.Key":     "cost-center",
		"Tags.member.1.Value":   "123",
	})

	w4 := doReq(t, h, map[string]string{
		"Action":                "DescribeTags",
		"ResourceArns.member.1": lArn,
	})
	require.Equal(t, http.StatusOK, w4.Code)
	assert.Contains(t, responseBody(t, w4), "cost-center")
}

// ---- InvalidAction ----

func TestELB_InvalidAction(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action": "NonExistentAction",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, responseBody(t, w), "InvalidAction")
}

// ============================================================
// Cross-resource relationship tests
// ============================================================

func TestELB_ListenerRequiresValidLB(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":                       "CreateListener",
		"LoadBalancerArn":              "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/fake/1234",
		"Protocol":                     "HTTP",
		"Port":                         "80",
		"DefaultActions.member.1.Type": "forward",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, responseBody(t, w), "LoadBalancerNotFound")
}

func TestELB_RuleRequiresValidListener(t *testing.T) {
	h := newELBGateway(t)
	w := doReq(t, h, map[string]string{
		"Action":                 "CreateRule",
		"ListenerArn":           "arn:aws:elasticloadbalancing:us-east-1:000:loadbalancer/app/fake/1234/listener/abc",
		"Priority":              "1",
		"Actions.member.1.Type": "forward",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, responseBody(t, w), "ListenerNotFound")
}

func TestELB_DeleteListener_CascadesRules(t *testing.T) {
	h := newELBGateway(t)
	_, listenerArn := createListenerForRules(t, h)

	// Create a rule
	doReq(t, h, map[string]string{
		"Action":                              "CreateRule",
		"ListenerArn":                         listenerArn,
		"Priority":                            "50",
		"Conditions.member.1.Field":           "path-pattern",
		"Conditions.member.1.Values.member.1": "/cascade",
		"Actions.member.1.Type":               "forward",
	})

	// Delete listener
	doReq(t, h, map[string]string{
		"Action":      "DeleteListener",
		"ListenerArn": listenerArn,
	})

	// Rules for this listener should be empty
	w := doReq(t, h, map[string]string{
		"Action":      "DescribeRules",
		"ListenerArn": listenerArn,
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, responseBody(t, w), "RuleArn")
}

// ============================================================
// Store-level tests
// ============================================================

func TestELB_Store_LoadBalancerStateMachine(t *testing.T) {
	svc := elbsvc.New("123456789012", "us-east-1")
	store := svc.GetStore()

	lb, err := store.CreateLoadBalancer("state-lb", "application", "", "", "", nil, nil, nil)
	require.NoError(t, err)
	// State should be active (auto-promoted from provisioning)
	assert.Equal(t, "active", lb.State)
}

func TestELB_Store_ARNFormat(t *testing.T) {
	svc := elbsvc.New("123456789012", "us-east-1")
	store := svc.GetStore()

	// ALB
	alb, _ := store.CreateLoadBalancer("alb-arn", "application", "", "", "", nil, nil, nil)
	assert.Contains(t, alb.ARN, "loadbalancer/app/alb-arn/")

	// NLB
	nlb, _ := store.CreateLoadBalancer("nlb-arn", "network", "", "", "", nil, nil, nil)
	assert.Contains(t, nlb.ARN, "loadbalancer/net/nlb-arn/")

	// GWLB
	gwlb, _ := store.CreateLoadBalancer("gwlb-arn", "gateway", "", "", "", nil, nil, nil)
	assert.Contains(t, gwlb.ARN, "loadbalancer/gwy/gwlb-arn/")
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

var testSeq int

func randomTestID() string {
	testSeq++
	return fmt.Sprintf("%d", testSeq)
}
