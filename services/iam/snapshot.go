package iam

import "encoding/json"

// iamState is the serialised form of all IAM state.
type iamState struct {
	Users    []iamUserState    `json:"users"`
	Roles    []iamRoleState    `json:"roles"`
	Policies []iamPolicyState  `json:"policies"`
}

type iamUserState struct {
	UserName string            `json:"user_name"`
	Tags     map[string]string `json:"tags,omitempty"`
}

type iamRoleState struct {
	RoleName                 string `json:"role_name"`
	AssumeRolePolicyDocument string `json:"assume_role_policy_document"`
	Description              string `json:"description,omitempty"`
}

type iamPolicyState struct {
	PolicyName string `json:"policy_name"`
	Document   string `json:"document"`
	Description string `json:"description,omitempty"`
}

// ExportState returns a JSON snapshot of all IAM users, roles, and policies.
func (s *IAMService) ExportState() (json.RawMessage, error) {
	state := iamState{
		Users:    make([]iamUserState, 0),
		Roles:    make([]iamRoleState, 0),
		Policies: make([]iamPolicyState, 0),
	}

	for _, u := range s.store.ListUsers() {
		us := iamUserState{
			UserName: u.UserName,
			Tags:     u.Tags,
		}
		state.Users = append(state.Users, us)
	}

	for _, r := range s.store.ListRoles() {
		state.Roles = append(state.Roles, iamRoleState{
			RoleName:                 r.RoleName,
			AssumeRolePolicyDocument: r.AssumeRolePolicyDocument,
			Description:              r.Description,
		})
	}

	for _, p := range s.store.ListPolicies() {
		state.Policies = append(state.Policies, iamPolicyState{
			PolicyName:  p.PolicyName,
			Document:    p.Document,
			Description: p.Description,
		})
	}

	return json.Marshal(state)
}

// ImportState restores IAM state from a JSON snapshot.
func (s *IAMService) ImportState(data json.RawMessage) error {
	var state iamState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for _, u := range state.Users {
		user, err := s.store.CreateUser(u.UserName)
		if err != nil {
			continue // skip duplicates
		}
		if len(u.Tags) > 0 {
			s.store.TagUser(user.UserName, u.Tags)
		}
	}

	for _, r := range state.Roles {
		s.store.CreateRole(r.RoleName, r.AssumeRolePolicyDocument, r.Description)
	}

	for _, p := range state.Policies {
		s.store.CreatePolicy(p.PolicyName, p.Document, p.Description)
	}

	return nil
}
