package cognito

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Group represents a Cognito user pool group.
type Group struct {
	GroupName   string
	UserPoolId  string
	Description string
	RoleArn     string
	Precedence  int
	CreationDate time.Time
	Members     map[string]bool // username -> member
}

// ── Store methods ────────────────────────────────────────────────────────────

func (s *Store) CreateGroup(userPoolID string, req createGroupRequest) (*Group, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return nil, poolNotFound(userPoolID)
	}

	key := userPoolID + "/" + req.GroupName
	if _, ok := s.groups[key]; ok {
		return nil, service.NewAWSError("GroupExistsException",
			fmt.Sprintf("Group %s already exists.", req.GroupName), http.StatusBadRequest)
	}

	group := &Group{
		GroupName:    req.GroupName,
		UserPoolId:   pool.Id,
		Description:  req.Description,
		RoleArn:      req.RoleArn,
		Precedence:   req.Precedence,
		CreationDate: time.Now().UTC(),
		Members:      make(map[string]bool),
	}
	s.groups[key] = group
	return group, nil
}

func (s *Store) GetGroup(userPoolID, groupName string) (*Group, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := userPoolID + "/" + groupName
	g, ok := s.groups[key]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Group %s not found.", groupName), http.StatusBadRequest)
	}
	return g, nil
}

func (s *Store) DeleteGroup(userPoolID, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userPoolID + "/" + groupName
	if _, ok := s.groups[key]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Group %s not found.", groupName), http.StatusBadRequest)
	}
	delete(s.groups, key)
	return nil
}

func (s *Store) ListGroups(userPoolID string) ([]*Group, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.pools[userPoolID]; !ok {
		return nil, poolNotFound(userPoolID)
	}

	prefix := userPoolID + "/"
	var groups []*Group
	for key, g := range s.groups {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			groups = append(groups, g)
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].GroupName < groups[j].GroupName
	})
	return groups, nil
}

func (s *Store) AddUserToGroup(userPoolID, username, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return poolNotFound(userPoolID)
	}
	if _, ok := pool.Users[username]; !ok {
		return service.NewAWSError("UserNotFoundException",
			"User does not exist.", http.StatusBadRequest)
	}
	key := userPoolID + "/" + groupName
	g, ok := s.groups[key]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Group %s not found.", groupName), http.StatusBadRequest)
	}
	g.Members[username] = true
	return nil
}

func (s *Store) RemoveUserFromGroup(userPoolID, username, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userPoolID + "/" + groupName
	g, ok := s.groups[key]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Group %s not found.", groupName), http.StatusBadRequest)
	}
	delete(g.Members, username)
	return nil
}

// ── Password Reset ───────────────────────────────────────────────────────────

func (s *Store) ForgotPassword(userPoolID, username string) *service.AWSError {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return poolNotFound(userPoolID)
	}
	if _, ok := pool.Users[username]; !ok {
		// AWS doesn't reveal if user exists — return success silently
		return nil
	}
	return nil
}

func (s *Store) ConfirmForgotPassword(userPoolID, username, password string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return poolNotFound(userPoolID)
	}
	user, ok := pool.Users[username]
	if !ok {
		return service.NewAWSError("UserNotFoundException",
			"User does not exist.", http.StatusBadRequest)
	}
	hashed, _ := hashPassword(password)
	user.PasswordHash = hashed
	user.UserStatus = userStatusConfirmed
	return nil
}

func (s *Store) ChangePassword(userPoolID, username, oldPassword, newPassword string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	pool, ok := s.pools[userPoolID]
	if !ok {
		return poolNotFound(userPoolID)
	}
	user, ok := pool.Users[username]
	if !ok {
		return service.NewAWSError("UserNotFoundException",
			"User does not exist.", http.StatusBadRequest)
	}
	if !checkPassword(user.PasswordHash, oldPassword) {
		return service.NewAWSError("NotAuthorizedException",
			"Incorrect password.", http.StatusBadRequest)
	}
	hashed, _ := hashPassword(newPassword)
	user.PasswordHash = hashed
	return nil
}

func poolNotFound(id string) *service.AWSError {
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("User pool %s not found.", id), http.StatusBadRequest)
}

// ── Request/Response types ───────────────────────────────────────────────────

type createGroupRequest struct {
	GroupName   string `json:"GroupName"`
	UserPoolId  string `json:"UserPoolId"`
	Description string `json:"Description"`
	RoleArn     string `json:"RoleArn"`
	Precedence  int    `json:"Precedence"`
}

func groupToJSON(g *Group) map[string]any {
	return map[string]any{
		"GroupName":    g.GroupName,
		"UserPoolId":   g.UserPoolId,
		"Description":  g.Description,
		"RoleArn":      g.RoleArn,
		"Precedence":   g.Precedence,
		"CreationDate": float64(g.CreationDate.Unix()),
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreateGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createGroupRequest
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if req.GroupName == "" || req.UserPoolId == "" {
		return cognitoErr("InvalidParameterException", "GroupName and UserPoolId are required.")
	}
	g, awsErr := store.CreateGroup(req.UserPoolId, req)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(map[string]any{"Group": groupToJSON(g)})
}

func handleDeleteGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		GroupName  string `json:"GroupName"`
		UserPoolId string `json:"UserPoolId"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if awsErr := store.DeleteGroup(req.UserPoolId, req.GroupName); awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(struct{}{})
}

func handleGetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		GroupName  string `json:"GroupName"`
		UserPoolId string `json:"UserPoolId"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	g, awsErr := store.GetGroup(req.UserPoolId, req.GroupName)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(map[string]any{"Group": groupToJSON(g)})
}

func handleListGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		UserPoolId string `json:"UserPoolId"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	groups, awsErr := store.ListGroups(req.UserPoolId)
	if awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	items := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		items = append(items, groupToJSON(g))
	}
	return cognitoOK(map[string]any{"Groups": items})
}

func handleAdminAddUserToGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		UserPoolId string `json:"UserPoolId"`
		Username   string `json:"Username"`
		GroupName  string `json:"GroupName"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if awsErr := store.AddUserToGroup(req.UserPoolId, req.Username, req.GroupName); awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(struct{}{})
}

func handleAdminRemoveUserFromGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		UserPoolId string `json:"UserPoolId"`
		Username   string `json:"Username"`
		GroupName  string `json:"GroupName"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	if awsErr := store.RemoveUserFromGroup(req.UserPoolId, req.Username, req.GroupName); awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(struct{}{})
}

func handleForgotPassword(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		ClientId string `json:"ClientId"`
		Username string `json:"Username"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	// Find pool by client ID
	poolID := store.FindPoolByClientID(req.ClientId)
	if poolID == "" {
		return cognitoErr("ResourceNotFoundException", "Client not found.")
	}
	_ = store.ForgotPassword(poolID, req.Username)
	return cognitoOK(map[string]any{
		"CodeDeliveryDetails": map[string]any{
			"AttributeName":  "email",
			"DeliveryMedium": "EMAIL",
			"Destination":    "***",
		},
	})
}

func handleConfirmForgotPassword(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		ClientId         string `json:"ClientId"`
		Username         string `json:"Username"`
		Password         string `json:"Password"`
		ConfirmationCode string `json:"ConfirmationCode"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	poolID := store.FindPoolByClientID(req.ClientId)
	if poolID == "" {
		return cognitoErr("ResourceNotFoundException", "Client not found.")
	}
	if awsErr := store.ConfirmForgotPassword(poolID, req.Username, req.Password); awsErr != nil {
		return cognitoJsonErr(awsErr)
	}
	return cognitoOK(struct{}{})
}

func handleChangePassword(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req struct {
		PreviousPassword string `json:"PreviousPassword"`
		ProposedPassword string `json:"ProposedPassword"`
		AccessToken      string `json:"AccessToken"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return cognitoErr("InvalidParameterException", "Invalid request body.")
	}
	// In a real implementation we'd decode the access token to get the username.
	// For mock purposes, we accept the change if the token is non-empty.
	if req.AccessToken == "" {
		return cognitoErr("NotAuthorizedException", "Access token is required.")
	}
	return cognitoOK(struct{}{})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func cognitoOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func cognitoJsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func cognitoErr(code, msg string) (*service.Response, error) {
	return cognitoJsonErr(service.NewAWSError(code, msg, http.StatusBadRequest))
}
