package ram_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/ram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.RAMService {
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

func createShare(t *testing.T, s *svc.RAMService, name string) string {
	t.Helper()
	resp, err := s.HandleRequest(jsonCtx("CreateResourceShare", map[string]any{
		"name": name,
	}))
	require.NoError(t, err)
	share := respBody(t, resp)["resourceShare"].(map[string]any)
	return share["resourceShareArn"].(string)
}

func TestRAM_CreateResourceShare(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateResourceShare", map[string]any{
		"name":                    "my-share",
		"allowExternalPrincipals": false,
		"principals":              []any{"arn:aws:iam::111111111111:root"},
		"resourceArns":            []any{"arn:aws:ec2:us-east-1:123456789012:subnet/subnet-123"},
		"tags":                    []any{map[string]any{"key": "env", "value": "dev"}},
	}))
	require.NoError(t, err)
	share := respBody(t, resp)["resourceShare"].(map[string]any)
	assert.Equal(t, "my-share", share["name"])
	assert.Equal(t, "ACTIVE", share["status"])
	assert.Contains(t, share["resourceShareArn"].(string), "arn:aws:ram:")
}

func TestRAM_CreateResourceShare_MissingName(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateResourceShare", map[string]any{}))
	require.Error(t, err)
}

func TestRAM_GetResourceShares(t *testing.T) {
	s := newService()
	createShare(t, s, "share-1")
	createShare(t, s, "share-2")
	resp, err := s.HandleRequest(jsonCtx("GetResourceShares", map[string]any{
		"resourceOwner": "SELF",
	}))
	require.NoError(t, err)
	shares := respBody(t, resp)["resourceShares"].([]any)
	assert.Len(t, shares, 2)
}

func TestRAM_UpdateResourceShare(t *testing.T) {
	s := newService()
	arn := createShare(t, s, "update-share")
	resp, err := s.HandleRequest(jsonCtx("UpdateResourceShare", map[string]any{
		"resourceShareArn":        arn,
		"name":                    "updated-share",
		"allowExternalPrincipals": false,
	}))
	require.NoError(t, err)
	share := respBody(t, resp)["resourceShare"].(map[string]any)
	assert.Equal(t, "updated-share", share["name"])
	assert.Equal(t, false, share["allowExternalPrincipals"])
}

func TestRAM_UpdateResourceShare_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdateResourceShare", map[string]any{
		"resourceShareArn": "arn:nonexistent",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UnknownResourceException")
}

func TestRAM_DeleteResourceShare(t *testing.T) {
	s := newService()
	arn := createShare(t, s, "del-share")
	resp, err := s.HandleRequest(jsonCtx("DeleteResourceShare", map[string]any{
		"resourceShareArn": arn,
	}))
	require.NoError(t, err)
	m := respBody(t, resp)
	assert.Equal(t, true, m["returnValue"])

	// After deletion, GetResourceShares should not return it (status=DELETED)
	resp, _ = s.HandleRequest(jsonCtx("GetResourceShares", map[string]any{
		"resourceOwner": "SELF",
	}))
	shares := respBody(t, resp)["resourceShares"].([]any)
	assert.Len(t, shares, 0)
}

func TestRAM_AssociateResourceShare(t *testing.T) {
	s := newService()
	arn := createShare(t, s, "assoc-share")
	resp, err := s.HandleRequest(jsonCtx("AssociateResourceShare", map[string]any{
		"resourceShareArn": arn,
		"principals":       []any{"arn:aws:iam::222222222222:root"},
		"resourceArns":     []any{"arn:aws:ec2:us-east-1:123456789012:vpc/vpc-123"},
	}))
	require.NoError(t, err)
	assocs := respBody(t, resp)["resourceShareAssociations"].([]any)
	assert.Len(t, assocs, 2) // 1 principal + 1 resource
}

func TestRAM_DisassociateResourceShare(t *testing.T) {
	s := newService()
	arn := createShare(t, s, "disassoc-share")
	s.HandleRequest(jsonCtx("AssociateResourceShare", map[string]any{
		"resourceShareArn": arn,
		"principals":       []any{"arn:aws:iam::222222222222:root"},
	}))

	resp, err := s.HandleRequest(jsonCtx("DisassociateResourceShare", map[string]any{
		"resourceShareArn": arn,
		"principals":       []any{"arn:aws:iam::222222222222:root"},
	}))
	require.NoError(t, err)
	assocs := respBody(t, resp)["resourceShareAssociations"].([]any)
	assert.Len(t, assocs, 1)
	assert.Equal(t, "DISASSOCIATED", assocs[0].(map[string]any)["status"])
}

func TestRAM_GetResourceShareAssociations(t *testing.T) {
	s := newService()
	arn := createShare(t, s, "assoc-list-share")
	s.HandleRequest(jsonCtx("AssociateResourceShare", map[string]any{
		"resourceShareArn": arn,
		"principals":       []any{"arn:p1", "arn:p2"},
		"resourceArns":     []any{"arn:r1"},
	}))

	resp, _ := s.HandleRequest(jsonCtx("GetResourceShareAssociations", map[string]any{
		"associationType":   "PRINCIPAL",
		"resourceShareArns": []any{arn},
	}))
	assocs := respBody(t, resp)["resourceShareAssociations"].([]any)
	assert.Len(t, assocs, 2)
}

func TestRAM_GetResourceShareInvitations_Empty(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("GetResourceShareInvitations", map[string]any{}))
	require.NoError(t, err)
	invs := respBody(t, resp)["resourceShareInvitations"].([]any)
	assert.Len(t, invs, 0)
}

func TestRAM_AcceptInvitation_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("AcceptResourceShareInvitation", map[string]any{
		"resourceShareInvitationArn": "arn:nonexistent",
	}))
	require.Error(t, err)
}

func TestRAM_RejectInvitation_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("RejectResourceShareInvitation", map[string]any{
		"resourceShareInvitationArn": "arn:nonexistent",
	}))
	require.Error(t, err)
}

func TestRAM_Tagging(t *testing.T) {
	s := newService()
	arn := createShare(t, s, "tag-share")

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"resourceShareArn": arn,
		"tags":             []any{map[string]any{"key": "env", "value": "prod"}, map[string]any{"key": "team", "value": "platform"}},
	}))
	require.NoError(t, err)

	resp, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{
		"resourceArn": arn,
	}))
	tags := respBody(t, resp)["tags"].([]any)
	assert.Len(t, tags, 2)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"resourceShareArn": arn,
		"tagKeys":          []any{"env"},
	}))
	require.NoError(t, err)

	resp, _ = s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{
		"resourceArn": arn,
	}))
	tags = respBody(t, resp)["tags"].([]any)
	assert.Len(t, tags, 1)
}

func TestRAM_InvalidPrincipal(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateResourceShare", map[string]any{
		"name":       "bad-principal-share",
		"principals": []any{"not-a-valid-principal"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MalformedArnException")
}

func TestRAM_ValidAccountIDPrincipal(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateResourceShare", map[string]any{
		"name":       "account-share",
		"principals": []any{"111111111111"},
	}))
	require.NoError(t, err)
	share := respBody(t, resp)["resourceShare"].(map[string]any)
	assert.Equal(t, "ACTIVE", share["status"])
}

func TestRAM_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", map[string]any{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "InvalidAction")
}
