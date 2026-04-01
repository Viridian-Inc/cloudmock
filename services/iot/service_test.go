package iot_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/iot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.IoTService {
	return svc.New("123456789012", "us-east-1")
}

func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{
		Action:     action,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       bodyBytes,
		Params:     make(map[string]string),
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
}

func respBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// ---- Test 1: CreateThing ----

func TestCreateThing(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{
		"thingName": "sensor-1",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body := respBody(t, resp)
	assert.Equal(t, "sensor-1", body["thingName"])
	assert.Contains(t, body["thingArn"].(string), "sensor-1")
}

// ---- Test 2: DescribeThing ----

func TestDescribeThing(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "sensor-desc"}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("DescribeThing", map[string]any{"thingName": "sensor-desc"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, "sensor-desc", body["thingName"])
	assert.Equal(t, float64(1), body["version"])
}

// ---- Test 3: ListThings ----

func TestListThings(t *testing.T) {
	s := newService()
	for _, name := range []string{"t-1", "t-2", "t-3"} {
		_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": name}))
		require.NoError(t, err)
	}

	resp, err := s.HandleRequest(jsonCtx("ListThings", map[string]any{}))
	require.NoError(t, err)
	body := respBody(t, resp)
	things := body["things"].([]any)
	assert.Len(t, things, 3)
}

// ---- Test 4: DeleteThing ----

func TestDeleteThing(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "t-del"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteThing", map[string]any{"thingName": "t-del"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DescribeThing", map[string]any{"thingName": "t-del"}))
	require.Error(t, err)
}

// ---- Test 5: UpdateThing ----

func TestUpdateThing(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "t-upd"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("UpdateThing", map[string]any{
		"thingName":        "t-upd",
		"attributePayload": map[string]any{"location": "floor-2"},
	}))
	require.NoError(t, err)

	resp, err := s.HandleRequest(jsonCtx("DescribeThing", map[string]any{"thingName": "t-upd"}))
	require.NoError(t, err)
	body := respBody(t, resp)
	assert.Equal(t, float64(2), body["version"])
}

// ---- Test 6: Thing NotFound ----

func TestThingNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeThing", map[string]any{"thingName": "nonexistent"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceNotFoundException", awsErr.Code)
}

// ---- Test 7: InvalidAction ----

func TestInvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("FakeAction", map[string]any{}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

// ---- Test 8: Policy CRUD + Attach/Detach ----

func TestPolicyCRUDAndAttach(t *testing.T) {
	s := newService()
	// Create
	createResp, err := s.HandleRequest(jsonCtx("CreatePolicy", map[string]any{
		"policyName":     "test-policy",
		"policyDocument": `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"iot:*","Resource":"*"}]}`,
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	assert.Equal(t, "test-policy", body["policyName"])

	// Get
	getResp, err := s.HandleRequest(jsonCtx("GetPolicy", map[string]any{"policyName": "test-policy"}))
	require.NoError(t, err)
	getBody := respBody(t, getResp)
	assert.Equal(t, "test-policy", getBody["policyName"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListPolicies", map[string]any{}))
	require.NoError(t, err)
	policies := respBody(t, listResp)["policies"].([]any)
	assert.Len(t, policies, 1)

	// Attach
	_, err = s.HandleRequest(jsonCtx("AttachPolicy", map[string]any{
		"policyName": "test-policy",
		"target":     "arn:aws:iot:us-east-1:123456789012:cert/abc123",
	}))
	require.NoError(t, err)

	// List attached
	attachedResp, err := s.HandleRequest(jsonCtx("ListAttachedPolicies", map[string]any{
		"target": "arn:aws:iot:us-east-1:123456789012:cert/abc123",
	}))
	require.NoError(t, err)
	attached := respBody(t, attachedResp)["policies"].([]any)
	assert.Len(t, attached, 1)

	// Detach
	_, err = s.HandleRequest(jsonCtx("DetachPolicy", map[string]any{
		"policyName": "test-policy",
		"target":     "arn:aws:iot:us-east-1:123456789012:cert/abc123",
	}))
	require.NoError(t, err)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeletePolicy", map[string]any{"policyName": "test-policy"}))
	require.NoError(t, err)
}

// ---- Test 9: Certificates ----

func TestCertificateCRUD(t *testing.T) {
	s := newService()
	// Create
	createResp, err := s.HandleRequest(jsonCtx("CreateKeysAndCertificate", map[string]any{
		"setAsActive": true,
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	certId := body["certificateId"].(string)
	assert.NotEmpty(t, certId)
	assert.NotEmpty(t, body["certificatePem"])
	assert.NotEmpty(t, body["keyPair"])

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeCertificate", map[string]any{"certificateId": certId}))
	require.NoError(t, err)
	certDesc := respBody(t, descResp)["certificateDescription"].(map[string]any)
	assert.Equal(t, "ACTIVE", certDesc["status"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListCertificates", map[string]any{}))
	require.NoError(t, err)
	certs := respBody(t, listResp)["certificates"].([]any)
	assert.Len(t, certs, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteCertificate", map[string]any{"certificateId": certId}))
	require.NoError(t, err)
}

// ---- Test 10: AttachThingPrincipal / DetachThingPrincipal ----

func TestThingPrincipal(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "sensor-p"}))
	require.NoError(t, err)

	certResp, err := s.HandleRequest(jsonCtx("CreateKeysAndCertificate", map[string]any{"setAsActive": true}))
	require.NoError(t, err)
	certArn := respBody(t, certResp)["certificateArn"].(string)

	_, err = s.HandleRequest(jsonCtx("AttachThingPrincipal", map[string]any{
		"thingName": "sensor-p",
		"principal": certArn,
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("DetachThingPrincipal", map[string]any{
		"thingName": "sensor-p",
		"principal": certArn,
	}))
	require.NoError(t, err)
}

// ---- Test 11: ThingType CRUD ----

func TestThingTypeCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateThingType", map[string]any{
		"thingTypeName": "sensor-type",
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	assert.Equal(t, "sensor-type", body["thingTypeName"])

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeThingType", map[string]any{"thingTypeName": "sensor-type"}))
	require.NoError(t, err)
	assert.Equal(t, "sensor-type", respBody(t, descResp)["thingTypeName"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListThingTypes", map[string]any{}))
	require.NoError(t, err)
	types := respBody(t, listResp)["thingTypes"].([]any)
	assert.Len(t, types, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteThingType", map[string]any{"thingTypeName": "sensor-type"}))
	require.NoError(t, err)
}

// ---- Test 12: ThingGroup + AddThing/RemoveThing ----

func TestThingGroupAndMembership(t *testing.T) {
	s := newService()
	// Create group
	createResp, err := s.HandleRequest(jsonCtx("CreateThingGroup", map[string]any{
		"thingGroupName": "floor-sensors",
	}))
	require.NoError(t, err)
	body := respBody(t, createResp)
	assert.Equal(t, "floor-sensors", body["thingGroupName"])

	// Create thing
	_, err = s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "sensor-grp"}))
	require.NoError(t, err)

	// Add to group
	_, err = s.HandleRequest(jsonCtx("AddThingToThingGroup", map[string]any{
		"thingName":      "sensor-grp",
		"thingGroupName": "floor-sensors",
	}))
	require.NoError(t, err)

	// Remove from group
	_, err = s.HandleRequest(jsonCtx("RemoveThingFromThingGroup", map[string]any{
		"thingName":      "sensor-grp",
		"thingGroupName": "floor-sensors",
	}))
	require.NoError(t, err)

	// List groups
	listResp, err := s.HandleRequest(jsonCtx("ListThingGroups", map[string]any{}))
	require.NoError(t, err)
	groups := respBody(t, listResp)["thingGroups"].([]any)
	assert.Len(t, groups, 1)

	// Delete group
	_, err = s.HandleRequest(jsonCtx("DeleteThingGroup", map[string]any{"thingGroupName": "floor-sensors"}))
	require.NoError(t, err)
}

// ---- Test 13: TopicRule CRUD ----

func TestTopicRuleCRUD(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateTopicRule", map[string]any{
		"ruleName": "temp-rule",
		"topicRulePayload": map[string]any{
			"sql":         "SELECT * FROM 'sensor/temp'",
			"description": "Temperature sensor rule",
			"actions":     []any{map[string]any{"lambda": map[string]any{"functionArn": "arn:aws:lambda:us-east-1:123456789012:function:process"}}},
		},
	}))
	require.NoError(t, err)

	// Get
	getResp, err := s.HandleRequest(jsonCtx("GetTopicRule", map[string]any{"ruleName": "temp-rule"}))
	require.NoError(t, err)
	body := respBody(t, getResp)
	rule := body["rule"].(map[string]any)
	assert.Equal(t, "temp-rule", rule["ruleName"])
	assert.Equal(t, "SELECT * FROM 'sensor/temp'", rule["sql"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListTopicRules", map[string]any{}))
	require.NoError(t, err)
	rules := respBody(t, listResp)["rules"].([]any)
	assert.Len(t, rules, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteTopicRule", map[string]any{"ruleName": "temp-rule"}))
	require.NoError(t, err)
}

// ---- Test 14: Tagging ----

func TestTagging(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "tag-thing"}))
	require.NoError(t, err)

	arn := "arn:aws:iot:us-east-1:123456789012:thing/tag-thing"

	_, err = s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        []any{map[string]any{"Key": "env", "Value": "prod"}},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	require.NoError(t, err)
	tags := respBody(t, listResp)["tags"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"resourceArn": arn}))
	require.NoError(t, err)
	tags2 := respBody(t, listResp2)["tags"].([]any)
	assert.Len(t, tags2, 0)
}

// ---- Test 15: Service Name and HealthCheck ----

func TestServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "iot", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test 16: Duplicate Thing ----

func TestDuplicateThing(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "dup-thing"}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "dup-thing"}))
	require.Error(t, err)
	awsErr, ok := err.(*service.AWSError)
	require.True(t, ok)
	assert.Equal(t, "ResourceAlreadyExistsException", awsErr.Code)
}

// ---- Enrichment tests ----

func TestIoT_ListThingPrincipals(t *testing.T) {
	s := newService()
	// Create a thing
	s.HandleRequest(jsonCtx("CreateThing", map[string]any{"thingName": "sensor-1"}))

	// Create certificate
	certResp, _ := s.HandleRequest(jsonCtx("CreateKeysAndCertificate", map[string]any{"setAsActive": true}))
	cm := respBody(t, certResp)
	certArn := cm["certificateArn"].(string)

	// Attach
	_, err := s.HandleRequest(jsonCtx("AttachThingPrincipal", map[string]any{
		"thingName": "sensor-1", "principal": certArn,
	}))
	require.NoError(t, err)

	// List principals
	resp, err := s.HandleRequest(jsonCtx("ListThingPrincipals", map[string]any{"thingName": "sensor-1"}))
	require.NoError(t, err)
	m := respBody(t, resp)
	principals := m["principals"].([]any)
	assert.Len(t, principals, 1)
	assert.Equal(t, certArn, principals[0])
}

func TestIoT_TopicRuleInvalidSQL(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateTopicRule", map[string]any{
		"ruleName": "bad-sql-rule",
		"topicRulePayload": map[string]any{
			"sql":         "INSERT INTO table VALUES (1)",
			"description": "bad SQL",
			"actions":     []map[string]any{},
		},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SELECT")
}

func TestIoT_TopicRuleValidSQL(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateTopicRule", map[string]any{
		"ruleName": "good-sql-rule",
		"topicRulePayload": map[string]any{
			"sql":         "SELECT * FROM 'iot/topic'",
			"description": "valid rule",
			"actions":     []map[string]any{},
		},
	}))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestIoT_ListThingPrincipals_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("ListThingPrincipals", map[string]any{"thingName": "nonexistent"}))
	require.Error(t, err)
}
