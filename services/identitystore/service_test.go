package identitystore_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/identitystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.IdentityStoreService { return svc.New("123456789012", "us-east-1") }
func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

const storeID = "d-1234567890"

func TestIS_CreateAndDescribeUser(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"IdentityStoreId": storeID, "UserName": "jdoe", "DisplayName": "John Doe",
		"Name": map[string]any{"GivenName": "John", "FamilyName": "Doe"},
		"Emails": []map[string]any{{"Value": "jdoe@example.com", "Type": "work", "Primary": true}},
	}))
	require.NoError(t, err)
	userID := respJSON(t, resp)["UserId"].(string)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeUser", map[string]any{
		"IdentityStoreId": storeID, "UserId": userID,
	}))
	m := respJSON(t, descResp)
	assert.Equal(t, "jdoe", m["UserName"])
	assert.Equal(t, "John", m["Name"].(map[string]any)["GivenName"])
}

func TestIS_ListUsers(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "u1"}))
	s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "u2"}))

	resp, _ := s.HandleRequest(jsonCtx("ListUsers", map[string]any{"IdentityStoreId": storeID}))
	assert.Len(t, respJSON(t, resp)["Users"].([]any), 2)
}

func TestIS_DeleteUser(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "del-u"}))
	userID := respJSON(t, cr)["UserId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteUser", map[string]any{"IdentityStoreId": storeID, "UserId": userID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = s.HandleRequest(jsonCtx("DescribeUser", map[string]any{"IdentityStoreId": storeID, "UserId": userID}))
	require.Error(t, err)
}

func TestIS_GroupCRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{
		"IdentityStoreId": storeID, "DisplayName": "Admins", "Description": "Admin group",
	}))
	require.NoError(t, err)
	groupID := respJSON(t, resp)["GroupId"].(string)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeGroup", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
	}))
	assert.Equal(t, "Admins", respJSON(t, descResp)["DisplayName"])

	listResp, _ := s.HandleRequest(jsonCtx("ListGroups", map[string]any{"IdentityStoreId": storeID}))
	assert.Len(t, respJSON(t, listResp)["Groups"].([]any), 1)

	s.HandleRequest(jsonCtx("DeleteGroup", map[string]any{"IdentityStoreId": storeID, "GroupId": groupID}))
	_, err = s.HandleRequest(jsonCtx("DescribeGroup", map[string]any{"IdentityStoreId": storeID, "GroupId": groupID}))
	require.Error(t, err)
}

func TestIS_GroupMembership(t *testing.T) {
	s := newService()
	ur, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "member-u"}))
	userID := respJSON(t, ur)["UserId"].(string)
	gr, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{"IdentityStoreId": storeID, "DisplayName": "Dev"}))
	groupID := respJSON(t, gr)["GroupId"].(string)

	memResp, err := s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID, "MemberId": map[string]any{"UserId": userID},
	}))
	require.NoError(t, err)
	membershipID := respJSON(t, memResp)["MembershipId"].(string)

	listResp, _ := s.HandleRequest(jsonCtx("ListGroupMemberships", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
	}))
	assert.Len(t, respJSON(t, listResp)["GroupMemberships"].([]any), 1)

	getMResp, _ := s.HandleRequest(jsonCtx("GetGroupMembershipId", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID, "MemberId": map[string]any{"UserId": userID},
	}))
	assert.Equal(t, membershipID, respJSON(t, getMResp)["MembershipId"])

	delResp, err := s.HandleRequest(jsonCtx("DeleteGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "MembershipId": membershipID,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestIS_DuplicateUser(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "dup"}))
	_, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "dup"}))
	require.Error(t, err)
}

func TestIS_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("Bogus", nil))
	require.Error(t, err)
}

func TestIS_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeUser", map[string]any{"IdentityStoreId": storeID, "UserId": "nonexistent"}))
	require.Error(t, err)
}

func TestIS_InvalidEmailFormat(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"IdentityStoreId": storeID, "UserName": "badmail",
		"Emails": []any{map[string]any{"Value": "not-an-email", "Type": "work", "Primary": true}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid email")
}

func TestIS_InvalidPhoneFormat(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"IdentityStoreId": storeID, "UserName": "badphone",
		"PhoneNumbers": []any{map[string]any{"Value": "12345", "Type": "work"}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "phone number")
}

func TestIS_MembershipRequiresExistingGroupAndUser(t *testing.T) {
	s := newService()
	ur, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "exists-u"}))
	userID := respJSON(t, ur)["UserId"].(string)

	// Try with non-existent group
	_, err := s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": "nonexistent-group",
		"MemberId": map[string]any{"UserId": userID},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Group not found")

	// Create group, try with non-existent user
	gr, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{"IdentityStoreId": storeID, "DisplayName": "Test"}))
	groupID := respJSON(t, gr)["GroupId"].(string)

	_, err = s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
		"MemberId": map[string]any{"UserId": "nonexistent-user"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "User not found")
}

func TestIS_UpdateUser(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"IdentityStoreId": storeID, "UserName": "update-u",
		"Name": map[string]any{"GivenName": "Old", "FamilyName": "Name"},
	}))
	userID := respJSON(t, cr)["UserId"].(string)

	_, err := s.HandleRequest(jsonCtx("UpdateUser", map[string]any{
		"IdentityStoreId": storeID, "UserId": userID,
		"Operations": []any{
			map[string]any{"AttributePath": "displayName", "AttributeValue": "Updated Name"},
		},
	}))
	require.NoError(t, err)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeUser", map[string]any{
		"IdentityStoreId": storeID, "UserId": userID,
	}))
	assert.Equal(t, "Updated Name", respJSON(t, descResp)["DisplayName"])
}

func TestIS_UpdateGroup(t *testing.T) {
	s := newService()
	gr, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{
		"IdentityStoreId": storeID, "DisplayName": "Original",
	}))
	groupID := respJSON(t, gr)["GroupId"].(string)

	_, err := s.HandleRequest(jsonCtx("UpdateGroup", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
		"Operations": []any{
			map[string]any{"AttributePath": "description", "AttributeValue": "Updated description"},
		},
	}))
	require.NoError(t, err)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeGroup", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
	}))
	assert.Equal(t, "Updated description", respJSON(t, descResp)["Description"])
}

func TestIS_GetGroupMembership(t *testing.T) {
	s := newService()
	ur, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "gm-u"}))
	userID := respJSON(t, ur)["UserId"].(string)
	gr, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{"IdentityStoreId": storeID, "DisplayName": "GrpGM"}))
	groupID := respJSON(t, gr)["GroupId"].(string)

	mr, _ := s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
		"MemberId": map[string]any{"UserId": userID},
	}))
	membershipID := respJSON(t, mr)["MembershipId"].(string)

	resp, err := s.HandleRequest(jsonCtx("GetGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "MembershipId": membershipID,
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Equal(t, membershipID, m["MembershipId"])
	assert.Equal(t, groupID, m["GroupId"])
}

func TestIS_IsMemberInGroups(t *testing.T) {
	s := newService()
	ur, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "im-u"}))
	userID := respJSON(t, ur)["UserId"].(string)

	gr1, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{"IdentityStoreId": storeID, "DisplayName": "G1"}))
	group1ID := respJSON(t, gr1)["GroupId"].(string)
	gr2, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{"IdentityStoreId": storeID, "DisplayName": "G2"}))
	group2ID := respJSON(t, gr2)["GroupId"].(string)

	// Add to group1 only
	s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": group1ID,
		"MemberId": map[string]any{"UserId": userID},
	}))

	resp, err := s.HandleRequest(jsonCtx("IsMemberInGroups", map[string]any{
		"IdentityStoreId": storeID,
		"MemberId":        map[string]any{"UserId": userID},
		"GroupIds":        []any{group1ID, group2ID},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	results := m["Results"].([]any)
	assert.Len(t, results, 2)

	for _, r := range results {
		rm := r.(map[string]any)
		if rm["GroupId"] == group1ID {
			assert.True(t, rm["MembershipExists"].(bool))
		} else {
			assert.False(t, rm["MembershipExists"].(bool))
		}
	}
}

func TestIS_DuplicateMembership(t *testing.T) {
	s := newService()
	ur, _ := s.HandleRequest(jsonCtx("CreateUser", map[string]any{"IdentityStoreId": storeID, "UserName": "dup-m"}))
	userID := respJSON(t, ur)["UserId"].(string)
	gr, _ := s.HandleRequest(jsonCtx("CreateGroup", map[string]any{"IdentityStoreId": storeID, "DisplayName": "DupG"}))
	groupID := respJSON(t, gr)["GroupId"].(string)

	_, err := s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
		"MemberId": map[string]any{"UserId": userID},
	}))
	require.NoError(t, err)

	_, err = s.HandleRequest(jsonCtx("CreateGroupMembership", map[string]any{
		"IdentityStoreId": storeID, "GroupId": groupID,
		"MemberId": map[string]any{"UserId": userID},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ConflictException")
}
