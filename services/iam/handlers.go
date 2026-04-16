package iam

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const iamXmlns = "https://iam.amazonaws.com/doc/2010-05-08/"

// ---- XML response types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// --- User types ---

type xmlUser struct {
	Path       string `xml:"Path"`
	UserName   string `xml:"UserName"`
	UserID     string `xml:"UserId"`
	Arn        string `xml:"Arn"`
	CreateDate string `xml:"CreateDate"`
}

type xmlCreateUserResponse struct {
	XMLName xml.Name            `xml:"CreateUserResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlCreateUserResult `xml:"CreateUserResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlCreateUserResult struct {
	User xmlUser `xml:"User"`
}

type xmlGetUserResponse struct {
	XMLName xml.Name            `xml:"GetUserResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlGetUserResult    `xml:"GetUserResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlGetUserResult struct {
	User xmlUser `xml:"User"`
}

type xmlListUsersResponse struct {
	XMLName xml.Name             `xml:"ListUsersResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlListUsersResult   `xml:"ListUsersResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlListUsersResult struct {
	Users       []xmlUser `xml:"Users>member"`
	IsTruncated bool      `xml:"IsTruncated"`
}

type xmlUpdateUserResponse struct {
	XMLName xml.Name            `xml:"UpdateUserResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlDeleteUserResponse struct {
	XMLName xml.Name            `xml:"DeleteUserResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// --- Role types ---

type xmlRole struct {
	Path                     string `xml:"Path"`
	RoleName                 string `xml:"RoleName"`
	RoleID                   string `xml:"RoleId"`
	Arn                      string `xml:"Arn"`
	CreateDate               string `xml:"CreateDate"`
	AssumeRolePolicyDocument string `xml:"AssumeRolePolicyDocument,omitempty"`
	Description              string `xml:"Description,omitempty"`
}

type xmlCreateRoleResponse struct {
	XMLName xml.Name            `xml:"CreateRoleResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlCreateRoleResult `xml:"CreateRoleResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlCreateRoleResult struct {
	Role xmlRole `xml:"Role"`
}

type xmlGetRoleResponse struct {
	XMLName xml.Name            `xml:"GetRoleResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlGetRoleResult    `xml:"GetRoleResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlGetRoleResult struct {
	Role xmlRole `xml:"Role"`
}

type xmlListRolesResponse struct {
	XMLName xml.Name             `xml:"ListRolesResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlListRolesResult   `xml:"ListRolesResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlListRolesResult struct {
	Roles       []xmlRole `xml:"Roles>member"`
	IsTruncated bool      `xml:"IsTruncated"`
}

type xmlDeleteRoleResponse struct {
	XMLName xml.Name            `xml:"DeleteRoleResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// --- Policy types ---

type xmlPolicy struct {
	PolicyName      string `xml:"PolicyName"`
	PolicyID        string `xml:"PolicyId"`
	Arn             string `xml:"Arn"`
	Path            string `xml:"Path"`
	Description     string `xml:"Description,omitempty"`
	CreateDate      string `xml:"CreateDate"`
	AttachmentCount int    `xml:"AttachmentCount"`
}

type xmlCreatePolicyResponse struct {
	XMLName xml.Name              `xml:"CreatePolicyResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlCreatePolicyResult `xml:"CreatePolicyResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlCreatePolicyResult struct {
	Policy xmlPolicy `xml:"Policy"`
}

type xmlGetPolicyResponse struct {
	XMLName xml.Name            `xml:"GetPolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlGetPolicyResult  `xml:"GetPolicyResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlGetPolicyResult struct {
	Policy xmlPolicy `xml:"Policy"`
}

type xmlListPoliciesResponse struct {
	XMLName xml.Name              `xml:"ListPoliciesResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlListPoliciesResult `xml:"ListPoliciesResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlListPoliciesResult struct {
	Policies    []xmlPolicy `xml:"Policies>member"`
	IsTruncated bool        `xml:"IsTruncated"`
}

type xmlDeletePolicyResponse struct {
	XMLName xml.Name            `xml:"DeletePolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// --- Attached policy types ---

type xmlAttachedPolicy struct {
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

type xmlAttachUserPolicyResponse struct {
	XMLName xml.Name            `xml:"AttachUserPolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlDetachUserPolicyResponse struct {
	XMLName xml.Name            `xml:"DetachUserPolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlAttachRolePolicyResponse struct {
	XMLName xml.Name            `xml:"AttachRolePolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlDetachRolePolicyResponse struct {
	XMLName xml.Name            `xml:"DetachRolePolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlListAttachedUserPoliciesResponse struct {
	XMLName xml.Name                          `xml:"ListAttachedUserPoliciesResponse"`
	Xmlns   string                            `xml:"xmlns,attr"`
	Result  xmlListAttachedUserPoliciesResult `xml:"ListAttachedUserPoliciesResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlListAttachedUserPoliciesResult struct {
	AttachedPolicies []xmlAttachedPolicy `xml:"AttachedPolicies>member"`
	IsTruncated      bool                `xml:"IsTruncated"`
}

type xmlListAttachedRolePoliciesResponse struct {
	XMLName xml.Name                          `xml:"ListAttachedRolePoliciesResponse"`
	Xmlns   string                            `xml:"xmlns,attr"`
	Result  xmlListAttachedRolePoliciesResult `xml:"ListAttachedRolePoliciesResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlListAttachedRolePoliciesResult struct {
	AttachedPolicies []xmlAttachedPolicy `xml:"AttachedPolicies>member"`
	IsTruncated      bool                `xml:"IsTruncated"`
}

// --- Group types ---

type xmlGroup struct {
	Path       string `xml:"Path"`
	GroupName  string `xml:"GroupName"`
	GroupID    string `xml:"GroupId"`
	Arn        string `xml:"Arn"`
	CreateDate string `xml:"CreateDate"`
}

type xmlCreateGroupResponse struct {
	XMLName xml.Name             `xml:"CreateGroupResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlCreateGroupResult `xml:"CreateGroupResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlCreateGroupResult struct {
	Group xmlGroup `xml:"Group"`
}

type xmlGetGroupResponse struct {
	XMLName xml.Name            `xml:"GetGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlGetGroupResult   `xml:"GetGroupResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlGetGroupResult struct {
	Group       xmlGroup  `xml:"Group"`
	Users       []xmlUser `xml:"Users>member"`
	IsTruncated bool      `xml:"IsTruncated"`
}

type xmlListGroupsResponse struct {
	XMLName xml.Name             `xml:"ListGroupsResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlListGroupsResult  `xml:"ListGroupsResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlListGroupsResult struct {
	Groups      []xmlGroup `xml:"Groups>member"`
	IsTruncated bool       `xml:"IsTruncated"`
}

type xmlDeleteGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlAddUserToGroupResponse struct {
	XMLName xml.Name            `xml:"AddUserToGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlRemoveUserFromGroupResponse struct {
	XMLName xml.Name            `xml:"RemoveUserFromGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// --- Access Key types ---

type xmlAccessKey struct {
	UserName    string `xml:"UserName"`
	AccessKeyID string `xml:"AccessKeyId"`
	Status      string `xml:"Status"`
	CreateDate  string `xml:"CreateDate"`
}

type xmlAccessKeyWithSecret struct {
	UserName       string `xml:"UserName"`
	AccessKeyID    string `xml:"AccessKeyId"`
	Status         string `xml:"Status"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	CreateDate     string `xml:"CreateDate"`
}

type xmlCreateAccessKeyResponse struct {
	XMLName xml.Name                 `xml:"CreateAccessKeyResponse"`
	Xmlns   string                   `xml:"xmlns,attr"`
	Result  xmlCreateAccessKeyResult `xml:"CreateAccessKeyResult"`
	Meta    xmlResponseMetadata      `xml:"ResponseMetadata"`
}

type xmlCreateAccessKeyResult struct {
	AccessKey xmlAccessKeyWithSecret `xml:"AccessKey"`
}

type xmlListAccessKeysResponse struct {
	XMLName xml.Name                `xml:"ListAccessKeysResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlListAccessKeysResult `xml:"ListAccessKeysResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlListAccessKeysResult struct {
	AccessKeyMetadata []xmlAccessKey `xml:"AccessKeyMetadata>member"`
	IsTruncated       bool           `xml:"IsTruncated"`
}

type xmlDeleteAccessKeyResponse struct {
	XMLName xml.Name            `xml:"DeleteAccessKeyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// --- Instance Profile types ---

type xmlInstanceProfile struct {
	InstanceProfileName string    `xml:"InstanceProfileName"`
	InstanceProfileID   string    `xml:"InstanceProfileId"`
	Arn                 string    `xml:"Arn"`
	Path                string    `xml:"Path"`
	Roles               []xmlRole `xml:"Roles>member"`
	CreateDate          string    `xml:"CreateDate"`
}

type xmlCreateInstanceProfileResponse struct {
	XMLName xml.Name                       `xml:"CreateInstanceProfileResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlCreateInstanceProfileResult `xml:"CreateInstanceProfileResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlCreateInstanceProfileResult struct {
	InstanceProfile xmlInstanceProfile `xml:"InstanceProfile"`
}

type xmlGetInstanceProfileResponse struct {
	XMLName xml.Name                    `xml:"GetInstanceProfileResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlGetInstanceProfileResult `xml:"GetInstanceProfileResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
}

type xmlGetInstanceProfileResult struct {
	InstanceProfile xmlInstanceProfile `xml:"InstanceProfile"`
}

type xmlListInstanceProfilesResponse struct {
	XMLName xml.Name                      `xml:"ListInstanceProfilesResponse"`
	Xmlns   string                        `xml:"xmlns,attr"`
	Result  xmlListInstanceProfilesResult `xml:"ListInstanceProfilesResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlListInstanceProfilesResult struct {
	InstanceProfiles []xmlInstanceProfile `xml:"InstanceProfiles>member"`
	IsTruncated      bool                 `xml:"IsTruncated"`
}

type xmlDeleteInstanceProfileResponse struct {
	XMLName xml.Name            `xml:"DeleteInstanceProfileResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlAddRoleToInstanceProfileResponse struct {
	XMLName xml.Name            `xml:"AddRoleToInstanceProfileResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlRemoveRoleFromInstanceProfileResponse struct {
	XMLName xml.Name            `xml:"RemoveRoleFromInstanceProfileResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// --- Tag types ---

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type xmlTagUserResponse struct {
	XMLName xml.Name            `xml:"TagUserResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlUntagUserResponse struct {
	XMLName xml.Name            `xml:"UntagUserResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlListUserTagsResponse struct {
	XMLName xml.Name              `xml:"ListUserTagsResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlListUserTagsResult `xml:"ListUserTagsResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlListUserTagsResult struct {
	Tags        []xmlTag `xml:"Tags>member"`
	IsTruncated bool     `xml:"IsTruncated"`
}

// ---- helpers ----

const timeFmt = "2006-01-02T15:04:05Z"

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func xmlOK(body any) (*service.Response, error) {
	data, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
	}, nil
}

func iamErr(code, msg string, status int) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML},
		service.NewAWSError(code, msg, status)
}

func errFromStore(err error) (*service.Response, error) {
	msg := err.Error()
	code := "InternalFailure"
	status := http.StatusInternalServerError

	if strings.HasPrefix(msg, "EntityAlreadyExists:") {
		code = "EntityAlreadyExists"
		status = http.StatusConflict
	} else if strings.HasPrefix(msg, "NoSuchEntity:") {
		code = "NoSuchEntity"
		status = http.StatusNotFound
	} else if strings.HasPrefix(msg, "DeleteConflict:") {
		code = "DeleteConflict"
		status = http.StatusConflict
	} else if strings.HasPrefix(msg, "LimitExceeded:") {
		code = "LimitExceeded"
		status = http.StatusConflict
	}

	return iamErr(code, msg, status)
}

func toXMLUser(u *IAMUser) xmlUser {
	return xmlUser{
		Path:       u.Path,
		UserName:   u.UserName,
		UserID:     u.UserID,
		Arn:        u.Arn,
		CreateDate: u.CreateDate.Format(timeFmt),
	}
}

func toXMLRole(r *IAMRole) xmlRole {
	return xmlRole{
		Path:                     r.Path,
		RoleName:                 r.RoleName,
		RoleID:                   r.RoleID,
		Arn:                      r.Arn,
		CreateDate:               r.CreateDate.Format(timeFmt),
		AssumeRolePolicyDocument: r.AssumeRolePolicyDocument,
		Description:              r.Description,
	}
}

func toXMLPolicy(p *IAMPolicy) xmlPolicy {
	return xmlPolicy{
		PolicyName:      p.PolicyName,
		PolicyID:        p.PolicyID,
		Arn:             p.Arn,
		Path:            p.Path,
		Description:     p.Description,
		CreateDate:      p.CreateDate.Format(timeFmt),
		AttachmentCount: p.AttachCount,
	}
}

func toXMLGroup(g *IAMGroup) xmlGroup {
	return xmlGroup{
		Path:       g.Path,
		GroupName:  g.GroupName,
		GroupID:    g.GroupID,
		Arn:        g.Arn,
		CreateDate: g.CreateDate.Format(timeFmt),
	}
}

// ---- User handlers ----

func handleCreateUser(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	user, err := store.CreateUser(userName)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlCreateUserResponse{
		Xmlns:  iamXmlns,
		Result: xmlCreateUserResult{User: toXMLUser(user)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleGetUser(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	user, err := store.GetUser(userName)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlGetUserResponse{
		Xmlns:  iamXmlns,
		Result: xmlGetUserResult{User: toXMLUser(user)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListUsers(store *Store) (*service.Response, error) {
	users := store.ListUsers()
	xmlUsers := make([]xmlUser, len(users))
	for i, u := range users {
		xmlUsers[i] = toXMLUser(u)
	}

	return xmlOK(&xmlListUsersResponse{
		Xmlns:  iamXmlns,
		Result: xmlListUsersResult{Users: xmlUsers},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDeleteUser(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	if err := store.DeleteUser(userName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDeleteUserResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleUpdateUser(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	newUserName := form.Get("NewUserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}
	if newUserName == "" {
		return iamErr("ValidationError", "NewUserName is required.", http.StatusBadRequest)
	}

	if _, err := store.UpdateUser(userName, newUserName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlUpdateUserResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Tag handlers ----

func handleTagUser(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	tags := parseTags(form)
	if err := store.TagUser(userName, tags); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlTagUserResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleUntagUser(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	tagKeys := parseTagKeys(form)
	if err := store.UntagUser(userName, tagKeys); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlUntagUserResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListUserTags(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	tags, err := store.ListUserTags(userName)
	if err != nil {
		return errFromStore(err)
	}

	xmlTags := make([]xmlTag, 0, len(tags))
	for k, v := range tags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}

	return xmlOK(&xmlListUserTagsResponse{
		Xmlns:  iamXmlns,
		Result: xmlListUserTagsResult{Tags: xmlTags},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// parseTags extracts Tags.member.N.Key / Tags.member.N.Value from form values.
func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		value := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		if key == "" {
			break
		}
		tags[key] = value
	}
	return tags
}

// parseTagKeys extracts TagKeys.member.N from form values.
func parseTagKeys(form url.Values) []string {
	var keys []string
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if key == "" {
			break
		}
		keys = append(keys, key)
	}
	return keys
}

// ---- Role handlers ----

func handleCreateRole(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	if roleName == "" {
		return iamErr("ValidationError", "RoleName is required.", http.StatusBadRequest)
	}

	assumeDoc := form.Get("AssumeRolePolicyDocument")
	description := form.Get("Description")

	role, err := store.CreateRole(roleName, assumeDoc, description)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlCreateRoleResponse{
		Xmlns:  iamXmlns,
		Result: xmlCreateRoleResult{Role: toXMLRole(role)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleGetRole(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	if roleName == "" {
		return iamErr("ValidationError", "RoleName is required.", http.StatusBadRequest)
	}

	role, err := store.GetRole(roleName)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlGetRoleResponse{
		Xmlns:  iamXmlns,
		Result: xmlGetRoleResult{Role: toXMLRole(role)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListRoles(store *Store) (*service.Response, error) {
	roles := store.ListRoles()
	xmlRoles := make([]xmlRole, len(roles))
	for i, r := range roles {
		xmlRoles[i] = toXMLRole(r)
	}

	return xmlOK(&xmlListRolesResponse{
		Xmlns:  iamXmlns,
		Result: xmlListRolesResult{Roles: xmlRoles},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDeleteRole(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	if roleName == "" {
		return iamErr("ValidationError", "RoleName is required.", http.StatusBadRequest)
	}

	if err := store.DeleteRole(roleName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDeleteRoleResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Policy handlers ----

func handleCreatePolicy(store *Store, form url.Values) (*service.Response, error) {
	policyName := form.Get("PolicyName")
	if policyName == "" {
		return iamErr("ValidationError", "PolicyName is required.", http.StatusBadRequest)
	}

	document := form.Get("PolicyDocument")
	description := form.Get("Description")

	policy, err := store.CreatePolicy(policyName, document, description)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlCreatePolicyResponse{
		Xmlns:  iamXmlns,
		Result: xmlCreatePolicyResult{Policy: toXMLPolicy(policy)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleGetPolicy(store *Store, form url.Values) (*service.Response, error) {
	policyArn := form.Get("PolicyArn")
	if policyArn == "" {
		return iamErr("ValidationError", "PolicyArn is required.", http.StatusBadRequest)
	}

	policy, err := store.GetPolicy(policyArn)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlGetPolicyResponse{
		Xmlns:  iamXmlns,
		Result: xmlGetPolicyResult{Policy: toXMLPolicy(policy)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListPolicies(store *Store) (*service.Response, error) {
	policies := store.ListPolicies()
	xmlPolicies := make([]xmlPolicy, len(policies))
	for i, p := range policies {
		xmlPolicies[i] = toXMLPolicy(p)
	}

	return xmlOK(&xmlListPoliciesResponse{
		Xmlns:  iamXmlns,
		Result: xmlListPoliciesResult{Policies: xmlPolicies},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDeletePolicy(store *Store, form url.Values) (*service.Response, error) {
	policyArn := form.Get("PolicyArn")
	if policyArn == "" {
		return iamErr("ValidationError", "PolicyArn is required.", http.StatusBadRequest)
	}

	if err := store.DeletePolicy(policyArn); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDeletePolicyResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleAttachUserPolicy(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	policyArn := form.Get("PolicyArn")
	if userName == "" || policyArn == "" {
		return iamErr("ValidationError", "UserName and PolicyArn are required.", http.StatusBadRequest)
	}

	if err := store.AttachUserPolicy(userName, policyArn); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlAttachUserPolicyResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDetachUserPolicy(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	policyArn := form.Get("PolicyArn")
	if userName == "" || policyArn == "" {
		return iamErr("ValidationError", "UserName and PolicyArn are required.", http.StatusBadRequest)
	}

	if err := store.DetachUserPolicy(userName, policyArn); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDetachUserPolicyResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleAttachRolePolicy(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	policyArn := form.Get("PolicyArn")
	if roleName == "" || policyArn == "" {
		return iamErr("ValidationError", "RoleName and PolicyArn are required.", http.StatusBadRequest)
	}

	if err := store.AttachRolePolicy(roleName, policyArn); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlAttachRolePolicyResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDetachRolePolicy(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	policyArn := form.Get("PolicyArn")
	if roleName == "" || policyArn == "" {
		return iamErr("ValidationError", "RoleName and PolicyArn are required.", http.StatusBadRequest)
	}

	if err := store.DetachRolePolicy(roleName, policyArn); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDetachRolePolicyResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListAttachedUserPolicies(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	attached, err := store.ListAttachedUserPolicies(userName)
	if err != nil {
		return errFromStore(err)
	}

	xmlAttached := make([]xmlAttachedPolicy, len(attached))
	for i, a := range attached {
		xmlAttached[i] = xmlAttachedPolicy{PolicyName: a.PolicyName, PolicyArn: a.PolicyArn}
	}

	return xmlOK(&xmlListAttachedUserPoliciesResponse{
		Xmlns:  iamXmlns,
		Result: xmlListAttachedUserPoliciesResult{AttachedPolicies: xmlAttached},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListAttachedRolePolicies(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	if roleName == "" {
		return iamErr("ValidationError", "RoleName is required.", http.StatusBadRequest)
	}

	attached, err := store.ListAttachedRolePolicies(roleName)
	if err != nil {
		return errFromStore(err)
	}

	xmlAttached := make([]xmlAttachedPolicy, len(attached))
	for i, a := range attached {
		xmlAttached[i] = xmlAttachedPolicy{PolicyName: a.PolicyName, PolicyArn: a.PolicyArn}
	}

	return xmlOK(&xmlListAttachedRolePoliciesResponse{
		Xmlns:  iamXmlns,
		Result: xmlListAttachedRolePoliciesResult{AttachedPolicies: xmlAttached},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Inline role policies ----

type xmlListRolePoliciesResponse struct {
	XMLName xml.Name                  `xml:"ListRolePoliciesResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlListRolePoliciesResult `xml:"ListRolePoliciesResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlListRolePoliciesResult struct {
	PolicyNames []string `xml:"PolicyNames>member"`
	IsTruncated bool     `xml:"IsTruncated"`
}

// handleListRolePolicies returns names of inline policies embedded on a role.
// Cloudmock doesn't model inline role policies yet (only managed-policy
// attachments via Attach/DetachRolePolicy), so this reports an empty list
// for any existing role. Pulumi's post-create refresh needs this call to
// succeed; without it, every IAM role resource fails after a successful
// CreateRole.
func handleListRolePolicies(store *Store, form url.Values) (*service.Response, error) {
	roleName := form.Get("RoleName")
	if roleName == "" {
		return iamErr("ValidationError", "RoleName is required.", http.StatusBadRequest)
	}
	if _, err := store.GetRole(roleName); err != nil {
		return errFromStore(err)
	}
	return xmlOK(&xmlListRolePoliciesResponse{
		Xmlns:  iamXmlns,
		Result: xmlListRolePoliciesResult{PolicyNames: nil},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Policy versions ----

type xmlPolicyVersion struct {
	Document         string `xml:"Document,omitempty"`
	VersionId        string `xml:"VersionId"`
	IsDefaultVersion bool   `xml:"IsDefaultVersion"`
	CreateDate       string `xml:"CreateDate"`
}

type xmlGetPolicyVersionResponse struct {
	XMLName xml.Name                  `xml:"GetPolicyVersionResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlGetPolicyVersionResult `xml:"GetPolicyVersionResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlGetPolicyVersionResult struct {
	PolicyVersion xmlPolicyVersion `xml:"PolicyVersion"`
}

type xmlListPolicyVersionsResponse struct {
	XMLName xml.Name                    `xml:"ListPolicyVersionsResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlListPolicyVersionsResult `xml:"ListPolicyVersionsResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
}

type xmlListPolicyVersionsResult struct {
	Versions    []xmlPolicyVersion `xml:"Versions>member"`
	IsTruncated bool               `xml:"IsTruncated"`
}

// handleGetPolicyVersion returns the policy document for a specific version of
// a managed policy. Cloudmock stores only the latest document and treats every
// policy as having a single version ("v1"). The document is URL-encoded in the
// response, matching the real IAM API contract — AWS SDKs URL-decode the
// Document field before surfacing it.
func handleGetPolicyVersion(store *Store, form url.Values) (*service.Response, error) {
	policyArn := form.Get("PolicyArn")
	if policyArn == "" {
		return iamErr("ValidationError", "PolicyArn is required.", http.StatusBadRequest)
	}
	versionId := form.Get("VersionId")
	if versionId == "" {
		versionId = "v1"
	}

	policy, err := store.GetPolicy(policyArn)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlGetPolicyVersionResponse{
		Xmlns: iamXmlns,
		Result: xmlGetPolicyVersionResult{
			PolicyVersion: xmlPolicyVersion{
				Document:         url.QueryEscape(policy.Document),
				VersionId:        versionId,
				IsDefaultVersion: versionId == "v1",
				CreateDate:       policy.CreateDate.Format(timeFmt),
			},
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// handleListPolicyVersions reports all versions of a managed policy. Cloudmock
// models only a single version per policy, so this always returns a one-entry
// list marked as the default version.
func handleListPolicyVersions(store *Store, form url.Values) (*service.Response, error) {
	policyArn := form.Get("PolicyArn")
	if policyArn == "" {
		return iamErr("ValidationError", "PolicyArn is required.", http.StatusBadRequest)
	}

	policy, err := store.GetPolicy(policyArn)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlListPolicyVersionsResponse{
		Xmlns: iamXmlns,
		Result: xmlListPolicyVersionsResult{
			Versions: []xmlPolicyVersion{{
				VersionId:        "v1",
				IsDefaultVersion: true,
				CreateDate:       policy.CreateDate.Format(timeFmt),
			}},
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Group handlers ----

func handleCreateGroup(store *Store, form url.Values) (*service.Response, error) {
	groupName := form.Get("GroupName")
	if groupName == "" {
		return iamErr("ValidationError", "GroupName is required.", http.StatusBadRequest)
	}

	group, err := store.CreateGroup(groupName)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlCreateGroupResponse{
		Xmlns:  iamXmlns,
		Result: xmlCreateGroupResult{Group: toXMLGroup(group)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleGetGroup(store *Store, form url.Values) (*service.Response, error) {
	groupName := form.Get("GroupName")
	if groupName == "" {
		return iamErr("ValidationError", "GroupName is required.", http.StatusBadRequest)
	}

	group, users, err := store.GetGroup(groupName)
	if err != nil {
		return errFromStore(err)
	}

	xmlUsers := make([]xmlUser, len(users))
	for i, u := range users {
		xmlUsers[i] = toXMLUser(u)
	}

	return xmlOK(&xmlGetGroupResponse{
		Xmlns: iamXmlns,
		Result: xmlGetGroupResult{
			Group: toXMLGroup(group),
			Users: xmlUsers,
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListGroups(store *Store) (*service.Response, error) {
	groups := store.ListGroups()
	xmlGroups := make([]xmlGroup, len(groups))
	for i, g := range groups {
		xmlGroups[i] = toXMLGroup(g)
	}

	return xmlOK(&xmlListGroupsResponse{
		Xmlns:  iamXmlns,
		Result: xmlListGroupsResult{Groups: xmlGroups},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDeleteGroup(store *Store, form url.Values) (*service.Response, error) {
	groupName := form.Get("GroupName")
	if groupName == "" {
		return iamErr("ValidationError", "GroupName is required.", http.StatusBadRequest)
	}

	if err := store.DeleteGroup(groupName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDeleteGroupResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleAddUserToGroup(store *Store, form url.Values) (*service.Response, error) {
	groupName := form.Get("GroupName")
	userName := form.Get("UserName")
	if groupName == "" || userName == "" {
		return iamErr("ValidationError", "GroupName and UserName are required.", http.StatusBadRequest)
	}

	if err := store.AddUserToGroup(groupName, userName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlAddUserToGroupResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleRemoveUserFromGroup(store *Store, form url.Values) (*service.Response, error) {
	groupName := form.Get("GroupName")
	userName := form.Get("UserName")
	if groupName == "" || userName == "" {
		return iamErr("ValidationError", "GroupName and UserName are required.", http.StatusBadRequest)
	}

	if err := store.RemoveUserFromGroup(groupName, userName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlRemoveUserFromGroupResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Access Key handlers ----

func handleCreateAccessKey(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	key, err := store.CreateAccessKey(userName)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlCreateAccessKeyResponse{
		Xmlns: iamXmlns,
		Result: xmlCreateAccessKeyResult{
			AccessKey: xmlAccessKeyWithSecret{
				UserName:       key.UserName,
				AccessKeyID:    key.AccessKeyID,
				Status:         key.Status,
				SecretAccessKey: key.SecretAccessKey,
				CreateDate:     key.CreateDate.Format(timeFmt),
			},
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListAccessKeys(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	if userName == "" {
		return iamErr("ValidationError", "UserName is required.", http.StatusBadRequest)
	}

	keys, err := store.ListAccessKeys(userName)
	if err != nil {
		return errFromStore(err)
	}

	xmlKeys := make([]xmlAccessKey, len(keys))
	for i, k := range keys {
		xmlKeys[i] = xmlAccessKey{
			UserName:    k.UserName,
			AccessKeyID: k.AccessKeyID,
			Status:      k.Status,
			CreateDate:  k.CreateDate.Format(timeFmt),
		}
	}

	return xmlOK(&xmlListAccessKeysResponse{
		Xmlns:  iamXmlns,
		Result: xmlListAccessKeysResult{AccessKeyMetadata: xmlKeys},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDeleteAccessKey(store *Store, form url.Values) (*service.Response, error) {
	userName := form.Get("UserName")
	accessKeyID := form.Get("AccessKeyId")
	if userName == "" || accessKeyID == "" {
		return iamErr("ValidationError", "UserName and AccessKeyId are required.", http.StatusBadRequest)
	}

	if err := store.DeleteAccessKey(userName, accessKeyID); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDeleteAccessKeyResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Instance Profile handlers ----

func handleCreateInstanceProfile(store *Store, form url.Values) (*service.Response, error) {
	name := form.Get("InstanceProfileName")
	if name == "" {
		return iamErr("ValidationError", "InstanceProfileName is required.", http.StatusBadRequest)
	}

	ip, err := store.CreateInstanceProfile(name)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlCreateInstanceProfileResponse{
		Xmlns: iamXmlns,
		Result: xmlCreateInstanceProfileResult{
			InstanceProfile: toXMLInstanceProfile(store, ip),
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleGetInstanceProfile(store *Store, form url.Values) (*service.Response, error) {
	name := form.Get("InstanceProfileName")
	if name == "" {
		return iamErr("ValidationError", "InstanceProfileName is required.", http.StatusBadRequest)
	}

	ip, err := store.GetInstanceProfile(name)
	if err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlGetInstanceProfileResponse{
		Xmlns: iamXmlns,
		Result: xmlGetInstanceProfileResult{
			InstanceProfile: toXMLInstanceProfile(store, ip),
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleListInstanceProfiles(store *Store) (*service.Response, error) {
	ips := store.ListInstanceProfiles()
	xmlIPs := make([]xmlInstanceProfile, len(ips))
	for i, ip := range ips {
		xmlIPs[i] = toXMLInstanceProfile(store, ip)
	}

	return xmlOK(&xmlListInstanceProfilesResponse{
		Xmlns:  iamXmlns,
		Result: xmlListInstanceProfilesResult{InstanceProfiles: xmlIPs},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleDeleteInstanceProfile(store *Store, form url.Values) (*service.Response, error) {
	name := form.Get("InstanceProfileName")
	if name == "" {
		return iamErr("ValidationError", "InstanceProfileName is required.", http.StatusBadRequest)
	}

	if err := store.DeleteInstanceProfile(name); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlDeleteInstanceProfileResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleAddRoleToInstanceProfile(store *Store, form url.Values) (*service.Response, error) {
	profileName := form.Get("InstanceProfileName")
	roleName := form.Get("RoleName")
	if profileName == "" || roleName == "" {
		return iamErr("ValidationError", "InstanceProfileName and RoleName are required.", http.StatusBadRequest)
	}

	if err := store.AddRoleToInstanceProfile(profileName, roleName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlAddRoleToInstanceProfileResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

func handleRemoveRoleFromInstanceProfile(store *Store, form url.Values) (*service.Response, error) {
	profileName := form.Get("InstanceProfileName")
	roleName := form.Get("RoleName")
	if profileName == "" || roleName == "" {
		return iamErr("ValidationError", "InstanceProfileName and RoleName are required.", http.StatusBadRequest)
	}

	if err := store.RemoveRoleFromInstanceProfile(profileName, roleName); err != nil {
		return errFromStore(err)
	}

	return xmlOK(&xmlRemoveRoleFromInstanceProfileResponse{
		Xmlns: iamXmlns,
		Meta:  xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// toXMLInstanceProfile converts an IAMInstanceProfile to its XML representation,
// resolving role names to full role details.
func toXMLInstanceProfile(store *Store, ip *IAMInstanceProfile) xmlInstanceProfile {
	var roles []xmlRole
	for _, roleName := range ip.Roles {
		if r, err := store.GetRole(roleName); err == nil {
			roles = append(roles, toXMLRole(r))
		}
	}

	return xmlInstanceProfile{
		InstanceProfileName: ip.InstanceProfileName,
		InstanceProfileID:   ip.InstanceProfileID,
		Arn:                 ip.Arn,
		Path:                ip.Path,
		Roles:               roles,
		CreateDate:          ip.CreateDate.Format(timeFmt),
	}
}
