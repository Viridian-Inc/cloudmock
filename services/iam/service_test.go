package iam_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	iamsvc "github.com/Viridian-Inc/cloudmock/services/iam"
)

// newIAMGateway builds a full gateway stack with the IAM service registered and IAM auth disabled.
func newIAMGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	engine := iampkg.NewEngine()
	pkgStore := iampkg.NewStore(cfg.AccountID)

	reg := routing.NewRegistry()
	reg.Register(iamsvc.New(cfg.AccountID, engine, pkgStore))

	return gateway.New(cfg, reg)
}

// iamReq builds a form-encoded POST request targeting the IAM service.
func iamReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2010-05-08")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/iam/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func doIAM(t *testing.T, handler http.Handler, action string, extra url.Values) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, iamReq(t, action, extra))
	return w
}

func requireOK(t *testing.T, w *httptest.ResponseRecorder, action string) string {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d\nbody: %s", action, w.Code, w.Body.String())
	}
	return w.Body.String()
}

func requireContains(t *testing.T, body, action string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("%s: expected body to contain %q\nbody: %s", action, want, body)
		}
	}
}

// ---- Test 1: CreateUser + GetUser + ListUsers ----

func TestIAM_CreateUser_GetUser_ListUsers(t *testing.T) {
	handler := newIAMGateway(t)

	// CreateUser
	extra := url.Values{}
	extra.Set("UserName", "testuser")
	w := doIAM(t, handler, "CreateUser", extra)
	body := requireOK(t, w, "CreateUser")
	requireContains(t, body, "CreateUser",
		"CreateUserResponse",
		"<UserName>testuser</UserName>",
		"<Arn>arn:aws:iam::000000000000:user/testuser</Arn>",
		"<UserId>AIDA",
		"RequestId",
	)

	// GetUser
	extra = url.Values{}
	extra.Set("UserName", "testuser")
	w = doIAM(t, handler, "GetUser", extra)
	body = requireOK(t, w, "GetUser")
	requireContains(t, body, "GetUser",
		"GetUserResponse",
		"<UserName>testuser</UserName>",
	)

	// ListUsers
	w = doIAM(t, handler, "ListUsers", nil)
	body = requireOK(t, w, "ListUsers")
	requireContains(t, body, "ListUsers",
		"ListUsersResponse",
		"<UserName>testuser</UserName>",
	)

	// GetUser - not found
	extra = url.Values{}
	extra.Set("UserName", "nonexistent")
	w = doIAM(t, handler, "GetUser", extra)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetUser (nonexistent): expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// CreateUser - duplicate
	extra = url.Values{}
	extra.Set("UserName", "testuser")
	w = doIAM(t, handler, "CreateUser", extra)
	if w.Code != http.StatusConflict {
		t.Fatalf("CreateUser (duplicate): expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test 2: CreateRole + GetRole ----

func TestIAM_CreateRole_GetRole(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("RoleName", "MyRole")
	extra.Set("AssumeRolePolicyDocument", `{"Version":"2012-10-17","Statement":[]}`)
	extra.Set("Description", "test role")
	w := doIAM(t, handler, "CreateRole", extra)
	body := requireOK(t, w, "CreateRole")
	requireContains(t, body, "CreateRole",
		"CreateRoleResponse",
		"<RoleName>MyRole</RoleName>",
		"<Arn>arn:aws:iam::000000000000:role/MyRole</Arn>",
		"<RoleId>AROA",
	)

	// GetRole
	extra = url.Values{}
	extra.Set("RoleName", "MyRole")
	w = doIAM(t, handler, "GetRole", extra)
	body = requireOK(t, w, "GetRole")
	requireContains(t, body, "GetRole",
		"GetRoleResponse",
		"<RoleName>MyRole</RoleName>",
		"<Description>test role</Description>",
	)

	// ListRoles
	w = doIAM(t, handler, "ListRoles", nil)
	body = requireOK(t, w, "ListRoles")
	requireContains(t, body, "ListRoles",
		"ListRolesResponse",
		"<RoleName>MyRole</RoleName>",
	)
}

// ---- Test 3: CreatePolicy + AttachUserPolicy + ListAttachedUserPolicies ----

func TestIAM_CreatePolicy_AttachUser_ListAttached(t *testing.T) {
	handler := newIAMGateway(t)

	// Create user
	extra := url.Values{}
	extra.Set("UserName", "policyuser")
	doIAM(t, handler, "CreateUser", extra)

	// Create policy
	extra = url.Values{}
	extra.Set("PolicyName", "ReadOnly")
	extra.Set("PolicyDocument", `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:Get*","Resource":"*"}]}`)
	w := doIAM(t, handler, "CreatePolicy", extra)
	body := requireOK(t, w, "CreatePolicy")
	requireContains(t, body, "CreatePolicy",
		"CreatePolicyResponse",
		"<PolicyName>ReadOnly</PolicyName>",
		"<Arn>arn:aws:iam::000000000000:policy/ReadOnly</Arn>",
		"<PolicyId>ANPA",
	)

	// Attach policy to user
	extra = url.Values{}
	extra.Set("UserName", "policyuser")
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/ReadOnly")
	w = doIAM(t, handler, "AttachUserPolicy", extra)
	requireOK(t, w, "AttachUserPolicy")

	// List attached user policies
	extra = url.Values{}
	extra.Set("UserName", "policyuser")
	w = doIAM(t, handler, "ListAttachedUserPolicies", extra)
	body = requireOK(t, w, "ListAttachedUserPolicies")
	requireContains(t, body, "ListAttachedUserPolicies",
		"ListAttachedUserPoliciesResponse",
		"<PolicyName>ReadOnly</PolicyName>",
		"<PolicyArn>arn:aws:iam::000000000000:policy/ReadOnly</PolicyArn>",
	)

	// Detach policy
	extra = url.Values{}
	extra.Set("UserName", "policyuser")
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/ReadOnly")
	w = doIAM(t, handler, "DetachUserPolicy", extra)
	requireOK(t, w, "DetachUserPolicy")

	// Verify detached
	extra = url.Values{}
	extra.Set("UserName", "policyuser")
	w = doIAM(t, handler, "ListAttachedUserPolicies", extra)
	body = requireOK(t, w, "ListAttachedUserPolicies")
	if strings.Contains(body, "ReadOnly") {
		t.Error("ListAttachedUserPolicies: policy should be detached")
	}
}

// ---- Test 4: CreateGroup + AddUserToGroup + GetGroup ----

func TestIAM_CreateGroup_AddUser_GetGroup(t *testing.T) {
	handler := newIAMGateway(t)

	// Create user
	extra := url.Values{}
	extra.Set("UserName", "groupuser")
	doIAM(t, handler, "CreateUser", extra)

	// Create group
	extra = url.Values{}
	extra.Set("GroupName", "Developers")
	w := doIAM(t, handler, "CreateGroup", extra)
	body := requireOK(t, w, "CreateGroup")
	requireContains(t, body, "CreateGroup",
		"CreateGroupResponse",
		"<GroupName>Developers</GroupName>",
		"<Arn>arn:aws:iam::000000000000:group/Developers</Arn>",
		"<GroupId>AGPA",
	)

	// Add user to group
	extra = url.Values{}
	extra.Set("GroupName", "Developers")
	extra.Set("UserName", "groupuser")
	w = doIAM(t, handler, "AddUserToGroup", extra)
	requireOK(t, w, "AddUserToGroup")

	// GetGroup should include the user
	extra = url.Values{}
	extra.Set("GroupName", "Developers")
	w = doIAM(t, handler, "GetGroup", extra)
	body = requireOK(t, w, "GetGroup")
	requireContains(t, body, "GetGroup",
		"GetGroupResponse",
		"<GroupName>Developers</GroupName>",
		"<UserName>groupuser</UserName>",
	)

	// RemoveUserFromGroup
	extra = url.Values{}
	extra.Set("GroupName", "Developers")
	extra.Set("UserName", "groupuser")
	w = doIAM(t, handler, "RemoveUserFromGroup", extra)
	requireOK(t, w, "RemoveUserFromGroup")

	// ListGroups
	w = doIAM(t, handler, "ListGroups", nil)
	body = requireOK(t, w, "ListGroups")
	requireContains(t, body, "ListGroups",
		"ListGroupsResponse",
		"<GroupName>Developers</GroupName>",
	)
}

// ---- Test 5: CreateAccessKey + ListAccessKeys ----

func TestIAM_CreateAccessKey_ListAccessKeys(t *testing.T) {
	handler := newIAMGateway(t)

	// Create user
	extra := url.Values{}
	extra.Set("UserName", "keyuser")
	doIAM(t, handler, "CreateUser", extra)

	// Create access key
	extra = url.Values{}
	extra.Set("UserName", "keyuser")
	w := doIAM(t, handler, "CreateAccessKey", extra)
	body := requireOK(t, w, "CreateAccessKey")
	requireContains(t, body, "CreateAccessKey",
		"CreateAccessKeyResponse",
		"<UserName>keyuser</UserName>",
		"<AccessKeyId>AKIA",
		"<SecretAccessKey>",
		"<Status>Active</Status>",
	)

	// List access keys
	extra = url.Values{}
	extra.Set("UserName", "keyuser")
	w = doIAM(t, handler, "ListAccessKeys", extra)
	body = requireOK(t, w, "ListAccessKeys")
	requireContains(t, body, "ListAccessKeys",
		"ListAccessKeysResponse",
		"<UserName>keyuser</UserName>",
		"<AccessKeyId>AKIA",
	)
}

// ---- Test 6: CreateInstanceProfile + AddRoleToInstanceProfile ----

func TestIAM_CreateInstanceProfile_AddRole(t *testing.T) {
	handler := newIAMGateway(t)

	// Create role
	extra := url.Values{}
	extra.Set("RoleName", "EC2Role")
	extra.Set("AssumeRolePolicyDocument", `{}`)
	doIAM(t, handler, "CreateRole", extra)

	// Create instance profile
	extra = url.Values{}
	extra.Set("InstanceProfileName", "EC2Profile")
	w := doIAM(t, handler, "CreateInstanceProfile", extra)
	body := requireOK(t, w, "CreateInstanceProfile")
	requireContains(t, body, "CreateInstanceProfile",
		"CreateInstanceProfileResponse",
		"<InstanceProfileName>EC2Profile</InstanceProfileName>",
		"<Arn>arn:aws:iam::000000000000:instance-profile/EC2Profile</Arn>",
	)

	// Add role to instance profile
	extra = url.Values{}
	extra.Set("InstanceProfileName", "EC2Profile")
	extra.Set("RoleName", "EC2Role")
	w = doIAM(t, handler, "AddRoleToInstanceProfile", extra)
	requireOK(t, w, "AddRoleToInstanceProfile")

	// GetInstanceProfile should include the role
	extra = url.Values{}
	extra.Set("InstanceProfileName", "EC2Profile")
	w = doIAM(t, handler, "GetInstanceProfile", extra)
	body = requireOK(t, w, "GetInstanceProfile")
	requireContains(t, body, "GetInstanceProfile",
		"GetInstanceProfileResponse",
		"<InstanceProfileName>EC2Profile</InstanceProfileName>",
		"<RoleName>EC2Role</RoleName>",
	)

	// ListInstanceProfiles
	w = doIAM(t, handler, "ListInstanceProfiles", nil)
	body = requireOK(t, w, "ListInstanceProfiles")
	requireContains(t, body, "ListInstanceProfiles",
		"ListInstanceProfilesResponse",
		"<InstanceProfileName>EC2Profile</InstanceProfileName>",
	)

	// RemoveRoleFromInstanceProfile
	extra = url.Values{}
	extra.Set("InstanceProfileName", "EC2Profile")
	extra.Set("RoleName", "EC2Role")
	w = doIAM(t, handler, "RemoveRoleFromInstanceProfile", extra)
	requireOK(t, w, "RemoveRoleFromInstanceProfile")
}

// ---- Test 7: Delete cascading ----

func TestIAM_Delete_Cascading(t *testing.T) {
	handler := newIAMGateway(t)

	// Create user with access key and attached policy
	extra := url.Values{}
	extra.Set("UserName", "cascadeuser")
	doIAM(t, handler, "CreateUser", extra)

	extra = url.Values{}
	extra.Set("UserName", "cascadeuser")
	doIAM(t, handler, "CreateAccessKey", extra)

	extra = url.Values{}
	extra.Set("PolicyName", "CascadePolicy")
	extra.Set("PolicyDocument", `{"Version":"2012-10-17","Statement":[]}`)
	doIAM(t, handler, "CreatePolicy", extra)

	extra = url.Values{}
	extra.Set("UserName", "cascadeuser")
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/CascadePolicy")
	doIAM(t, handler, "AttachUserPolicy", extra)

	// Delete user should clean up access keys and policy attachments
	extra = url.Values{}
	extra.Set("UserName", "cascadeuser")
	w := doIAM(t, handler, "DeleteUser", extra)
	requireOK(t, w, "DeleteUser")

	// User should be gone
	extra = url.Values{}
	extra.Set("UserName", "cascadeuser")
	w = doIAM(t, handler, "GetUser", extra)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetUser after delete: expected 404, got %d", w.Code)
	}

	// Policy should now be deletable (attach count back to 0)
	extra = url.Values{}
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/CascadePolicy")
	w = doIAM(t, handler, "DeletePolicy", extra)
	requireOK(t, w, "DeletePolicy")

	// Delete role with instance profile reference
	extra = url.Values{}
	extra.Set("RoleName", "CascadeRole")
	extra.Set("AssumeRolePolicyDocument", `{}`)
	doIAM(t, handler, "CreateRole", extra)

	extra = url.Values{}
	extra.Set("InstanceProfileName", "CascadeProfile")
	doIAM(t, handler, "CreateInstanceProfile", extra)

	extra = url.Values{}
	extra.Set("InstanceProfileName", "CascadeProfile")
	extra.Set("RoleName", "CascadeRole")
	doIAM(t, handler, "AddRoleToInstanceProfile", extra)

	// Delete role should remove it from instance profiles
	extra = url.Values{}
	extra.Set("RoleName", "CascadeRole")
	w = doIAM(t, handler, "DeleteRole", extra)
	requireOK(t, w, "DeleteRole")

	// Instance profile should still exist but without the role
	extra = url.Values{}
	extra.Set("InstanceProfileName", "CascadeProfile")
	w = doIAM(t, handler, "GetInstanceProfile", extra)
	body := requireOK(t, w, "GetInstanceProfile")
	if strings.Contains(body, "CascadeRole") {
		t.Error("GetInstanceProfile: role should be removed after role deletion")
	}
}

// ---- Test: Unknown action ----

func TestIAM_UnknownAction(t *testing.T) {
	handler := newIAMGateway(t)

	w := doIAM(t, handler, "NonExistentAction", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test: UpdateUser ----

func TestIAM_UpdateUser(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("UserName", "oldname")
	doIAM(t, handler, "CreateUser", extra)

	extra = url.Values{}
	extra.Set("UserName", "oldname")
	extra.Set("NewUserName", "newname")
	w := doIAM(t, handler, "UpdateUser", extra)
	requireOK(t, w, "UpdateUser")

	// Old name should be gone
	extra = url.Values{}
	extra.Set("UserName", "oldname")
	w = doIAM(t, handler, "GetUser", extra)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetUser (old name): expected 404, got %d", w.Code)
	}

	// New name should work
	extra = url.Values{}
	extra.Set("UserName", "newname")
	w = doIAM(t, handler, "GetUser", extra)
	body := requireOK(t, w, "GetUser")
	requireContains(t, body, "GetUser",
		"<UserName>newname</UserName>",
	)
}

// ---- Test: EntityAlreadyExists — CreateRole duplicate ----

func TestIAM_CreateRole_AlreadyExists(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("RoleName", "DupRole")
	extra.Set("AssumeRolePolicyDocument", `{}`)
	w := doIAM(t, handler, "CreateRole", extra)
	requireOK(t, w, "first CreateRole")

	// Duplicate.
	w = doIAM(t, handler, "CreateRole", extra)
	if w.Code != http.StatusConflict {
		t.Fatalf("CreateRole duplicate: expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "EntityAlreadyExists") {
		t.Errorf("CreateRole duplicate: expected EntityAlreadyExists\nbody: %s", body)
	}
}

// ---- Test: NoSuchEntity — GetUser not found (explicit error code) ----

func TestIAM_GetUser_NoSuchEntity(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("UserName", "ghost-user")
	w := doIAM(t, handler, "GetUser", extra)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetUser not found: expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "NoSuchEntity") {
		t.Errorf("GetUser not found: expected NoSuchEntity\nbody: %s", body)
	}
}

// ---- Test: NoSuchEntity — GetRole not found ----

func TestIAM_GetRole_NoSuchEntity(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("RoleName", "GhostRole")
	w := doIAM(t, handler, "GetRole", extra)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetRole not found: expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "NoSuchEntity") {
		t.Errorf("GetRole not found: expected NoSuchEntity\nbody: %s", body)
	}
}

// ---- Test: GetPolicy + NoSuchEntity ----

func TestIAM_GetPolicy(t *testing.T) {
	handler := newIAMGateway(t)

	// Create a policy.
	extra := url.Values{}
	extra.Set("PolicyName", "GetTestPolicy")
	extra.Set("PolicyDocument", `{"Version":"2012-10-17","Statement":[]}`)
	w := doIAM(t, handler, "CreatePolicy", extra)
	requireOK(t, w, "CreatePolicy")

	// GetPolicy.
	extra = url.Values{}
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/GetTestPolicy")
	w = doIAM(t, handler, "GetPolicy", extra)
	body := requireOK(t, w, "GetPolicy")
	requireContains(t, body, "GetPolicy",
		"GetPolicyResponse",
		"<PolicyName>GetTestPolicy</PolicyName>",
	)

	// GetPolicy — not found.
	extra = url.Values{}
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/NonExistentPolicy")
	w = doIAM(t, handler, "GetPolicy", extra)
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetPolicy not found: expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ---- Test: DetachRolePolicy + NoSuchEntity ----

func TestIAM_DetachRolePolicy_NotAttached(t *testing.T) {
	handler := newIAMGateway(t)

	// Create role and policy.
	extra := url.Values{}
	extra.Set("RoleName", "DetachRole")
	extra.Set("AssumeRolePolicyDocument", `{}`)
	doIAM(t, handler, "CreateRole", extra)

	extra = url.Values{}
	extra.Set("PolicyName", "DetachPol")
	extra.Set("PolicyDocument", `{}`)
	doIAM(t, handler, "CreatePolicy", extra)

	// Detach without attaching first — should fail.
	extra = url.Values{}
	extra.Set("RoleName", "DetachRole")
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/DetachPol")
	w := doIAM(t, handler, "DetachRolePolicy", extra)
	if w.Code == http.StatusOK {
		t.Fatal("DetachRolePolicy not attached: expected error, got 200")
	}
}

// ---- Test: CreatePolicy — EntityAlreadyExists ----

func TestIAM_CreatePolicy_AlreadyExists(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("PolicyName", "DupPolicy")
	extra.Set("PolicyDocument", `{}`)
	w := doIAM(t, handler, "CreatePolicy", extra)
	requireOK(t, w, "first CreatePolicy")

	// Duplicate.
	w = doIAM(t, handler, "CreatePolicy", extra)
	if w.Code != http.StatusConflict {
		t.Fatalf("CreatePolicy duplicate: expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "EntityAlreadyExists") {
		t.Errorf("CreatePolicy duplicate: expected EntityAlreadyExists\nbody: %s", body)
	}
}

// ---- Test: AttachRolePolicy + ListAttachedRolePolicies ----

func TestIAM_AttachRolePolicy_ListAttached(t *testing.T) {
	handler := newIAMGateway(t)

	extra := url.Values{}
	extra.Set("RoleName", "PolRole")
	extra.Set("AssumeRolePolicyDocument", `{}`)
	doIAM(t, handler, "CreateRole", extra)

	extra = url.Values{}
	extra.Set("PolicyName", "RolePol")
	extra.Set("PolicyDocument", `{}`)
	doIAM(t, handler, "CreatePolicy", extra)

	extra = url.Values{}
	extra.Set("RoleName", "PolRole")
	extra.Set("PolicyArn", "arn:aws:iam::000000000000:policy/RolePol")
	w := doIAM(t, handler, "AttachRolePolicy", extra)
	requireOK(t, w, "AttachRolePolicy")

	extra = url.Values{}
	extra.Set("RoleName", "PolRole")
	w = doIAM(t, handler, "ListAttachedRolePolicies", extra)
	body := requireOK(t, w, "ListAttachedRolePolicies")
	requireContains(t, body, "ListAttachedRolePolicies",
		"<PolicyName>RolePol</PolicyName>",
	)
}
