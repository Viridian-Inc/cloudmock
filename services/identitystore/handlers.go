package identitystore

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func userResponse(u *User) map[string]any {
	resp := map[string]any{
		"UserId":          u.UserID,
		"IdentityStoreId": u.IdentityStoreID,
		"UserName":        u.UserName,
		"DisplayName":     u.DisplayName,
	}
	if u.Name != nil {
		resp["Name"] = map[string]any{
			"GivenName":  u.Name.GivenName,
			"FamilyName": u.Name.FamilyName,
			"MiddleName": u.Name.MiddleName,
			"Formatted":  u.Name.Formatted,
		}
	}
	if len(u.Emails) > 0 {
		emails := make([]map[string]any, 0, len(u.Emails))
		for _, e := range u.Emails {
			emails = append(emails, map[string]any{
				"Value":   e.Value,
				"Type":    e.Type,
				"Primary": e.Primary,
			})
		}
		resp["Emails"] = emails
	}
	return resp
}

func groupResponse(g *Group) map[string]any {
	return map[string]any{
		"GroupId":         g.GroupID,
		"IdentityStoreId": g.IdentityStoreID,
		"DisplayName":    g.DisplayName,
		"Description":    g.Description,
	}
}

func handleCreateUser(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	userName := str(params, "UserName")
	if identityStoreID == "" || userName == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and UserName are required"))
	}

	var name *Name
	if n, ok := params["Name"].(map[string]any); ok {
		name = &Name{
			GivenName:  str(n, "GivenName"),
			FamilyName: str(n, "FamilyName"),
			MiddleName: str(n, "MiddleName"),
			Formatted:  str(n, "Formatted"),
		}
	}

	var emails []Email
	if es, ok := params["Emails"].([]any); ok {
		for _, e := range es {
			if em, ok := e.(map[string]any); ok {
				value := str(em, "Value")
				if value != "" && !emailRegex.MatchString(value) {
					return jsonErr(service.ErrValidation("Invalid email format: " + value))
				}
				primary, _ := em["Primary"].(bool)
				emails = append(emails, Email{
					Value:   value,
					Type:    str(em, "Type"),
					Primary: primary,
				})
			}
		}
	}

	var phoneNumbers []PhoneNumber
	if pns, ok := params["PhoneNumbers"].([]any); ok {
		for _, p := range pns {
			if pm, ok := p.(map[string]any); ok {
				value := str(pm, "Value")
				if value != "" {
					cleaned := strings.ReplaceAll(value, "-", "")
					if !phoneRegex.MatchString(cleaned) {
						return jsonErr(service.ErrValidation("Invalid phone number format: " + value))
					}
				}
				primary, _ := pm["Primary"].(bool)
				phoneNumbers = append(phoneNumbers, PhoneNumber{
					Value:   value,
					Type:    str(pm, "Type"),
					Primary: primary,
				})
			}
		}
	}

	user, err := store.CreateUser(identityStoreID, userName, str(params, "DisplayName"), name, emails)
	if err != nil {
		return jsonErr(service.NewAWSError("ConflictException", err.Error(), http.StatusConflict))
	}
	return jsonOK(map[string]any{
		"UserId":          user.UserID,
		"IdentityStoreId": identityStoreID,
	})
}

func handleDescribeUser(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	userID := str(params, "UserId")
	if identityStoreID == "" || userID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and UserId are required"))
	}
	user, ok := store.GetUser(identityStoreID, userID)
	if !ok {
		return jsonErr(service.ErrNotFound("User", userID))
	}
	return jsonOK(userResponse(user))
}

func handleListUsers(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	if identityStoreID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId is required"))
	}
	users := store.ListUsers(identityStoreID)
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, userResponse(u))
	}
	return jsonOK(map[string]any{"Users": out})
}

func handleDeleteUser(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	userID := str(params, "UserId")
	if identityStoreID == "" || userID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and UserId are required"))
	}
	if !store.DeleteUser(identityStoreID, userID) {
		return jsonErr(service.ErrNotFound("User", userID))
	}
	return jsonOK(map[string]any{})
}

func handleCreateGroup(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	displayName := str(params, "DisplayName")
	if identityStoreID == "" || displayName == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and DisplayName are required"))
	}
	group, _ := store.CreateGroup(identityStoreID, displayName, str(params, "Description"))
	return jsonOK(map[string]any{
		"GroupId":         group.GroupID,
		"IdentityStoreId": identityStoreID,
	})
}

func handleDescribeGroup(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	groupID := str(params, "GroupId")
	if identityStoreID == "" || groupID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and GroupId are required"))
	}
	group, ok := store.GetGroup(identityStoreID, groupID)
	if !ok {
		return jsonErr(service.ErrNotFound("Group", groupID))
	}
	return jsonOK(groupResponse(group))
}

func handleListGroups(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	if identityStoreID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId is required"))
	}
	groups := store.ListGroups(identityStoreID)
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupResponse(g))
	}
	return jsonOK(map[string]any{"Groups": out})
}

func handleDeleteGroup(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	groupID := str(params, "GroupId")
	if identityStoreID == "" || groupID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and GroupId are required"))
	}
	if !store.DeleteGroup(identityStoreID, groupID) {
		return jsonErr(service.ErrNotFound("Group", groupID))
	}
	return jsonOK(map[string]any{})
}

func handleCreateGroupMembership(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	groupID := str(params, "GroupId")

	var memberID string
	if mid, ok := params["MemberId"].(map[string]any); ok {
		memberID = str(mid, "UserId")
	}
	if identityStoreID == "" || groupID == "" || memberID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId, GroupId, and MemberId.UserId are required"))
	}

	// Validate group exists
	if _, ok := store.GetGroup(identityStoreID, groupID); !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Group not found: "+groupID, http.StatusNotFound))
	}
	// Validate user exists
	if _, ok := store.GetUser(identityStoreID, memberID); !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"User not found: "+memberID, http.StatusNotFound))
	}

	membership, err := store.CreateGroupMembership(identityStoreID, groupID, memberID)
	if err != nil {
		return jsonErr(service.NewAWSError("ConflictException", err.Error(), http.StatusConflict))
	}
	return jsonOK(map[string]any{
		"MembershipId":    membership.MembershipID,
		"IdentityStoreId": identityStoreID,
	})
}

func handleGetGroupMembershipID(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	groupID := str(params, "GroupId")
	var memberID string
	if mid, ok := params["MemberId"].(map[string]any); ok {
		memberID = str(mid, "UserId")
	}

	memberships := store.ListGroupMemberships(identityStoreID, groupID)
	for _, m := range memberships {
		if m.MemberID == memberID {
			return jsonOK(map[string]any{
				"MembershipId":    m.MembershipID,
				"IdentityStoreId": identityStoreID,
			})
		}
	}
	return jsonErr(service.ErrNotFound("GroupMembership", groupID+"/"+memberID))
}

func handleListGroupMemberships(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	groupID := str(params, "GroupId")
	if identityStoreID == "" || groupID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and GroupId are required"))
	}
	memberships := store.ListGroupMemberships(identityStoreID, groupID)
	out := make([]map[string]any, 0, len(memberships))
	for _, m := range memberships {
		out = append(out, map[string]any{
			"MembershipId":    m.MembershipID,
			"IdentityStoreId": m.IdentityStoreID,
			"GroupId":         m.GroupID,
			"MemberId":        map[string]any{"UserId": m.MemberID},
		})
	}
	return jsonOK(map[string]any{"GroupMemberships": out})
}

func handleDeleteGroupMembership(params map[string]any, store *Store) (*service.Response, error) {
	identityStoreID := str(params, "IdentityStoreId")
	membershipID := str(params, "MembershipId")
	if identityStoreID == "" || membershipID == "" {
		return jsonErr(service.ErrValidation("IdentityStoreId and MembershipId are required"))
	}
	if !store.DeleteGroupMembership(identityStoreID, membershipID) {
		return jsonErr(service.ErrNotFound("GroupMembership", membershipID))
	}
	return jsonOK(map[string]any{})
}
