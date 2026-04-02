package shield_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/shield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.ShieldService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestShield_CreateProtection(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name":        "my-protection",
		"ResourceArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/my-alb",
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.NotEmpty(t, m["ProtectionId"])
}

func TestShield_CreateProtection_DuplicateResource(t *testing.T) {
	s := newService()
	resArn := "arn:aws:ec2:us-east-1:123456789012:eip-allocation/eipalloc-123"
	s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "p1", "ResourceArn": resArn,
	}))
	_, err := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "p2", "ResourceArn": resArn,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceAlreadyExistsException")
}

func TestShield_DescribeProtection(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "desc-prot", "ResourceArn": "arn:aws:ec2:us-east-1:123456789012:eip/1",
	}))
	protID := respBody(t, resp)["ProtectionId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DescribeProtection", map[string]any{
		"ProtectionId": protID,
	}))
	require.NoError(t, err)
	prot := respBody(t, resp)["Protection"].(map[string]any)
	assert.Equal(t, "desc-prot", prot["Name"])
	assert.NotEmpty(t, prot["ProtectionArn"])
}

func TestShield_DescribeProtection_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeProtection", map[string]any{
		"ProtectionId": "nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestShield_ListProtections(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "p1", "ResourceArn": "arn:1",
	}))
	s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "p2", "ResourceArn": "arn:2",
	}))
	resp, _ := s.HandleRequest(jsonCtx("ListProtections", map[string]any{}))
	prots := respBody(t, resp)["Protections"].([]any)
	assert.Len(t, prots, 2)
}

func TestShield_DeleteProtection(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "del-prot", "ResourceArn": "arn:del",
	}))
	protID := respBody(t, resp)["ProtectionId"].(string)

	_, err := s.HandleRequest(jsonCtx("DeleteProtection", map[string]any{
		"ProtectionId": protID,
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DescribeProtection", map[string]any{
		"ProtectionId": protID,
	}))
	require.Error(t, err)
}

func TestShield_Subscription(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateSubscription", map[string]any{}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("DescribeSubscription", map[string]any{}))
	require.NoError(t, err)
	sub := respBody(t, resp)["Subscription"].(map[string]any)
	assert.Equal(t, "ACTIVE", sub["SubscriptionState"])
	assert.Equal(t, "ENABLED", sub["AutoRenew"])
	assert.NotEmpty(t, sub["SubscriptionArn"])
}

func TestShield_Subscription_AlreadyExists(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSubscription", map[string]any{}))
	_, err := s.HandleRequest(jsonCtx("CreateSubscription", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceAlreadyExistsException")
}

func TestShield_DescribeSubscription_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeSubscription", map[string]any{}))
	require.Error(t, err)
}

func TestShield_ProtectionGroup_CRUD(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProtectionGroup", map[string]any{
		"ProtectionGroupId": "pg-1",
		"Aggregation":       "MAX",
		"Pattern":           "ALL",
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("DescribeProtectionGroup", map[string]any{
		"ProtectionGroupId": "pg-1",
	}))
	require.NoError(t, err)
	pg := respBody(t, resp)["ProtectionGroup"].(map[string]any)
	assert.Equal(t, "MAX", pg["Aggregation"])

	// List
	resp, _ = s.HandleRequest(jsonCtx("ListProtectionGroups", map[string]any{}))
	pgs := respBody(t, resp)["ProtectionGroups"].([]any)
	assert.Len(t, pgs, 1)

	// Update
	_, err = s.HandleRequest(jsonCtx("UpdateProtectionGroup", map[string]any{
		"ProtectionGroupId": "pg-1",
		"Aggregation":       "SUM",
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("DescribeProtectionGroup", map[string]any{
		"ProtectionGroupId": "pg-1",
	}))
	pg = respBody(t, resp)["ProtectionGroup"].(map[string]any)
	assert.Equal(t, "SUM", pg["Aggregation"])

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteProtectionGroup", map[string]any{
		"ProtectionGroupId": "pg-1",
	}))
	require.NoError(t, err)
}

func TestShield_ProtectionGroup_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeProtectionGroup", map[string]any{
		"ProtectionGroupId": "nonexistent",
	}))
	require.Error(t, err)
}

func TestShield_Tagging(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "tag-prot", "ResourceArn": "arn:tag",
	}))
	protID := respBody(t, resp)["ProtectionId"].(string)

	// Need the ARN for tagging
	resp, _ = s.HandleRequest(jsonCtx("DescribeProtection", map[string]any{
		"ProtectionId": protID,
	}))
	arn := respBody(t, resp)["Protection"].(map[string]any)["ProtectionArn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": arn,
		"Tags":        []any{map[string]any{"Key": "env", "Value": "dev"}},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{
		"ResourceARN": arn,
	}))
	tags := respBody(t, resp)["Tags"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"ResourceARN": arn,
		"TagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": arn}))
	tags = respBody(t, resp)["Tags"].([]any)
	assert.Len(t, tags, 0)
}

func TestShield_CreateProtection_MissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"ResourceArn": "arn:aws:ec2:us-east-1:123456789012:eip/1",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestShield_CreateProtection_MissingResourceArn(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "test-prot",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestShield_OneSubscriptionPerAccount(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateSubscription", map[string]any{}))
	require.NoError(t, err)
	_, err = s.HandleRequest(jsonCtx("CreateSubscription", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceAlreadyExistsException")
}

func TestShield_DescribeAttack_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeAttack", map[string]any{
		"AttackId": "nonexistent",
	}))
	require.Error(t, err)
}

func TestShield_ListAttacks_Empty(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("ListAttacks", map[string]any{}))
	require.NoError(t, err)
	attacks := respBody(t, resp)["AttackSummaries"].([]any)
	assert.Len(t, attacks, 0)
}

func TestShield_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}

func TestShield_UpdateSubscription(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateSubscription", map[string]any{}))

	_, err := s.HandleRequest(jsonCtx("UpdateSubscription", map[string]any{
		"AutoRenew": "DISABLED",
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeSubscription", map[string]any{}))
	sub := respBody(t, resp)["Subscription"].(map[string]any)
	assert.Equal(t, "DISABLED", sub["AutoRenew"])
}

func TestShield_UpdateSubscription_NoSubscription(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdateSubscription", map[string]any{
		"AutoRenew": "DISABLED",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestShield_DescribeAttackStatistics(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeAttackStatistics", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Contains(t, m, "TimeRange")
	assert.Contains(t, m, "DataItems")
	tr := m["TimeRange"].(map[string]any)
	assert.Contains(t, tr, "FromInclusive")
	assert.Contains(t, tr, "ToExclusive")
}

func TestShield_DescribeDRTAccess(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("DescribeDRTAccess", map[string]any{}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Contains(t, m, "RoleArn")
	assert.Contains(t, m, "LogBucketList")
}

func TestShield_ApplicationLayerAutoResponse(t *testing.T) {
	s := newService()
	resourceArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/abc"

	// Create protection
	s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "app-prot", "ResourceArn": resourceArn,
	}))

	// Enable
	_, err := s.HandleRequest(jsonCtx("EnableApplicationLayerAutomaticResponse", map[string]any{
		"ResourceArn": resourceArn,
		"Action":      map[string]any{"Block": map[string]any{}},
	}))
	require.NoError(t, err)

	// Disable
	_, err = s.HandleRequest(jsonCtx("DisableApplicationLayerAutomaticResponse", map[string]any{
		"ResourceArn": resourceArn,
	}))
	require.NoError(t, err)
}

func TestShield_ApplicationLayerAutoResponse_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("EnableApplicationLayerAutomaticResponse", map[string]any{
		"ResourceArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/no-such",
		"Action":      map[string]any{"Count": map[string]any{}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestShield_EnableApplicationLayerAutoResponse_MissingArn(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("EnableApplicationLayerAutomaticResponse", map[string]any{
		"Action": map[string]any{"Block": map[string]any{}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ValidationError")
}

func TestShield_AssociateHealthCheck(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "hc-prot", "ResourceArn": "arn:aws:ec2:us-east-1:123456789012:eip/hc1",
	}))
	protID := respBody(t, resp)["ProtectionId"].(string)
	hcArn := "arn:aws:route53:::healthcheck/abc-123"

	_, err := s.HandleRequest(jsonCtx("AssociateHealthCheck", map[string]any{
		"ProtectionId":   protID,
		"HealthCheckArn": hcArn,
	}))
	require.NoError(t, err)

	// Disassociate
	_, err = s.HandleRequest(jsonCtx("DisassociateHealthCheck", map[string]any{
		"ProtectionId":   protID,
		"HealthCheckArn": hcArn,
	}))
	require.NoError(t, err)
}

func TestShield_AssociateHealthCheck_Duplicate(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "hc-dup", "ResourceArn": "arn:aws:ec2:us-east-1:123456789012:eip/hc2",
	}))
	protID := respBody(t, resp)["ProtectionId"].(string)
	hcArn := "arn:aws:route53:::healthcheck/dup-123"

	s.HandleRequest(jsonCtx("AssociateHealthCheck", map[string]any{
		"ProtectionId": protID, "HealthCheckArn": hcArn,
	}))
	_, err := s.HandleRequest(jsonCtx("AssociateHealthCheck", map[string]any{
		"ProtectionId": protID, "HealthCheckArn": hcArn,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceAlreadyExistsException")
}

func TestShield_AssociateHealthCheck_ProtectionNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("AssociateHealthCheck", map[string]any{
		"ProtectionId":   "nonexistent",
		"HealthCheckArn": "arn:aws:route53:::healthcheck/xyz",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestShield_DisassociateHealthCheck_NotAssociated(t *testing.T) {
	s := newService()
	resp, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "hc-noassoc", "ResourceArn": "arn:aws:ec2:us-east-1:123456789012:eip/hc3",
	}))
	protID := respBody(t, resp)["ProtectionId"].(string)

	_, err := s.HandleRequest(jsonCtx("DisassociateHealthCheck", map[string]any{
		"ProtectionId":   protID,
		"HealthCheckArn": "arn:aws:route53:::healthcheck/nope",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ResourceNotFoundException")
}

func TestShield_ProtectionGroup_ARBITRARY_Members(t *testing.T) {
	s := newService()
	r1, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "pg-p1", "ResourceArn": "arn:pg:1",
	}))
	r2, _ := s.HandleRequest(jsonCtx("CreateProtection", map[string]any{
		"Name": "pg-p2", "ResourceArn": "arn:pg:2",
	}))
	id1 := respBody(t, r1)["ProtectionId"].(string)
	id2 := respBody(t, r2)["ProtectionId"].(string)

	_, err := s.HandleRequest(jsonCtx("CreateProtectionGroup", map[string]any{
		"ProtectionGroupId": "arb-pg",
		"Aggregation":       "MAX",
		"Pattern":           "ARBITRARY",
		"Members":           []any{id1, id2},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeProtectionGroup", map[string]any{
		"ProtectionGroupId": "arb-pg",
	}))
	pg := respBody(t, resp)["ProtectionGroup"].(map[string]any)
	assert.Equal(t, "ARBITRARY", pg["Pattern"])
	members := pg["Members"].([]any)
	assert.Len(t, members, 2)
}

func TestShield_TagResource_ProtectionGroup(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateProtectionGroup", map[string]any{
		"ProtectionGroupId": "tag-pg",
		"Aggregation":       "SUM",
		"Pattern":           "ALL",
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("DescribeProtectionGroup", map[string]any{
		"ProtectionGroupId": "tag-pg",
	}))
	pgArn := respBody(t, resp)["ProtectionGroup"].(map[string]any)["ProtectionGroupArn"].(string)

	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"ResourceARN": pgArn,
		"Tags":        []any{map[string]any{"Key": "project", "Value": "shield-test"}},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"ResourceARN": pgArn}))
	tags := respBody(t, resp)["Tags"].([]any)
	assert.Len(t, tags, 1)
}
