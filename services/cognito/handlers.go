package cognito

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

// UserPool wire types

type createUserPoolRequest struct {
	PoolName               string                   `json:"PoolName"`
	Policies               map[string]any   `json:"Policies"`
	AutoVerifiedAttributes []string                 `json:"AutoVerifiedAttributes"`
	Schema                 []map[string]any `json:"Schema"`
}

type userPoolResponse struct {
	Id           string  `json:"Id"`
	Name         string  `json:"Name"`
	Arn          string  `json:"Arn"`
	CreationDate float64 `json:"CreationDate"`
	Status       string  `json:"Status"`
}

type createUserPoolResponse struct {
	UserPool userPoolResponse `json:"UserPool"`
}

type deleteUserPoolRequest struct {
	UserPoolId string `json:"UserPoolId"`
}

type describeUserPoolRequest struct {
	UserPoolId string `json:"UserPoolId"`
}

type describeUserPoolResponse struct {
	UserPool userPoolResponse `json:"UserPool"`
}

type listUserPoolsRequest struct {
	MaxResults int `json:"MaxResults"`
}

type listUserPoolsResponse struct {
	UserPools []userPoolResponse `json:"UserPools"`
}

// UserPoolClient wire types

type createUserPoolClientRequest struct {
	UserPoolId        string   `json:"UserPoolId"`
	ClientName        string   `json:"ClientName"`
	ExplicitAuthFlows []string `json:"ExplicitAuthFlows"`
	GenerateSecret    bool     `json:"GenerateSecret"`
}

type userPoolClientResponse struct {
	ClientId          string   `json:"ClientId"`
	ClientName        string   `json:"ClientName"`
	UserPoolId        string   `json:"UserPoolId"`
	ClientSecret      string   `json:"ClientSecret,omitempty"`
	ExplicitAuthFlows []string `json:"ExplicitAuthFlows,omitempty"`
}

type createUserPoolClientResponse struct {
	UserPoolClient userPoolClientResponse `json:"UserPoolClient"`
}

type describeUserPoolClientRequest struct {
	UserPoolId string `json:"UserPoolId"`
	ClientId   string `json:"ClientId"`
}

type describeUserPoolClientResponse struct {
	UserPoolClient userPoolClientResponse `json:"UserPoolClient"`
}

type listUserPoolClientsRequest struct {
	UserPoolId string `json:"UserPoolId"`
}

type listUserPoolClientsResponse struct {
	UserPoolClients []userPoolClientResponse `json:"UserPoolClients"`
}

// User wire types

type userAttributeInput struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type adminCreateUserRequest struct {
	UserPoolId        string               `json:"UserPoolId"`
	Username          string               `json:"Username"`
	UserAttributes    []userAttributeInput `json:"UserAttributes"`
	TemporaryPassword string               `json:"TemporaryPassword"`
}

type userResponse struct {
	Username       string               `json:"Username"`
	Attributes     []userAttributeInput `json:"Attributes"`
	UserCreateDate float64              `json:"UserCreateDate"`
	UserStatus     string               `json:"UserStatus"`
	Enabled        bool                 `json:"Enabled"`
}

type adminCreateUserResponse struct {
	User userResponse `json:"User"`
}

type adminGetUserRequest struct {
	UserPoolId string `json:"UserPoolId"`
	Username   string `json:"Username"`
}

type adminGetUserResponse struct {
	Username       string               `json:"Username"`
	UserAttributes []userAttributeInput `json:"UserAttributes"`
	UserCreateDate float64              `json:"UserCreateDate"`
	UserStatus     string               `json:"UserStatus"`
	Enabled        bool                 `json:"Enabled"`
}

type adminDeleteUserRequest struct {
	UserPoolId string `json:"UserPoolId"`
	Username   string `json:"Username"`
}

type adminSetUserPasswordRequest struct {
	UserPoolId string `json:"UserPoolId"`
	Username   string `json:"Username"`
	Password   string `json:"Password"`
	Permanent  bool   `json:"Permanent"`
}

// Auth wire types

type signUpRequest struct {
	ClientId       string               `json:"ClientId"`
	Username       string               `json:"Username"`
	Password       string               `json:"Password"`
	UserAttributes []userAttributeInput `json:"UserAttributes"`
}

type signUpResponse struct {
	UserConfirmed bool   `json:"UserConfirmed"`
	UserSub       string `json:"UserSub"`
}

type initiateAuthRequest struct {
	AuthFlow       string            `json:"AuthFlow"`
	ClientId       string            `json:"ClientId"`
	AuthParameters map[string]string `json:"AuthParameters"`
}

type authenticationResult struct {
	AccessToken  string `json:"AccessToken"`
	IdToken      string `json:"IdToken"`
	RefreshToken string `json:"RefreshToken"`
	ExpiresIn    int    `json:"ExpiresIn"`
	TokenType    string `json:"TokenType"`
}

type initiateAuthResponse struct {
	AuthenticationResult authenticationResult `json:"AuthenticationResult"`
}

type adminConfirmSignUpRequest struct {
	UserPoolId string `json:"UserPoolId"`
	Username   string `json:"Username"`
}

// ---- helpers ----

func poolToResponse(p *UserPool) userPoolResponse {
	return userPoolResponse{
		Id:           p.Id,
		Name:         p.Name,
		Arn:          p.Arn,
		CreationDate: float64(p.CreationDate.Unix()),
		Status:       p.Status,
	}
}

func clientToResponse(c *UserPoolClient) userPoolClientResponse {
	return userPoolClientResponse{
		ClientId:          c.ClientId,
		ClientName:        c.ClientName,
		UserPoolId:        c.UserPoolId,
		ClientSecret:      c.ClientSecret,
		ExplicitAuthFlows: c.ExplicitAuthFlows,
	}
}

func userToResponse(u *User) userResponse {
	attrs := make([]userAttributeInput, 0, len(u.Attributes))
	for k, v := range u.Attributes {
		attrs = append(attrs, userAttributeInput{Name: k, Value: v})
	}
	return userResponse{
		Username:       u.Username,
		Attributes:     attrs,
		UserCreateDate: float64(u.UserCreateDate.Unix()),
		UserStatus:     string(u.UserStatus),
		Enabled:        u.Enabled,
	}
}

func attributeSliceToMap(attrs []userAttributeInput) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name] = a.Value
	}
	return m
}

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonEmpty() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- UserPool handlers ----

func handleCreateUserPool(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createUserPoolRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.PoolName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"PoolName is required.", http.StatusBadRequest))
	}

	pool, awsErr := store.CreateUserPool(req.PoolName, req.Policies, req.AutoVerifiedAttributes, req.Schema)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(createUserPoolResponse{UserPool: poolToResponse(pool)})
}

func handleDeleteUserPool(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteUserPoolRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId is required.", http.StatusBadRequest))
	}
	if awsErr := store.DeleteUserPool(req.UserPoolId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleDescribeUserPool(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeUserPoolRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId is required.", http.StatusBadRequest))
	}

	pool, awsErr := store.GetUserPool(req.UserPoolId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(describeUserPoolResponse{UserPool: poolToResponse(pool)})
}

func handleListUserPools(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listUserPoolsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	pools := store.ListUserPools(req.MaxResults)
	entries := make([]userPoolResponse, 0, len(pools))
	for _, p := range pools {
		entries = append(entries, poolToResponse(p))
	}
	return jsonOK(listUserPoolsResponse{UserPools: entries})
}

// ---- UserPoolClient handlers ----

func handleCreateUserPoolClient(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createUserPoolClientRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId is required.", http.StatusBadRequest))
	}
	if req.ClientName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"ClientName is required.", http.StatusBadRequest))
	}

	client, awsErr := store.CreateUserPoolClient(req.UserPoolId, req.ClientName, req.ExplicitAuthFlows, req.GenerateSecret)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(createUserPoolClientResponse{UserPoolClient: clientToResponse(client)})
}

func handleDescribeUserPoolClient(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeUserPoolClientRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" || req.ClientId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId and ClientId are required.", http.StatusBadRequest))
	}

	client, awsErr := store.GetUserPoolClient(req.UserPoolId, req.ClientId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(describeUserPoolClientResponse{UserPoolClient: clientToResponse(client)})
}

func handleListUserPoolClients(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listUserPoolClientsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId is required.", http.StatusBadRequest))
	}

	clients, awsErr := store.ListUserPoolClients(req.UserPoolId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	entries := make([]userPoolClientResponse, 0, len(clients))
	for _, c := range clients {
		entries = append(entries, clientToResponse(c))
	}
	return jsonOK(listUserPoolClientsResponse{UserPoolClients: entries})
}

// ---- User handlers ----

func handleAdminCreateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req adminCreateUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId is required.", http.StatusBadRequest))
	}
	if req.Username == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Username is required.", http.StatusBadRequest))
	}

	attrs := attributeSliceToMap(req.UserAttributes)
	user, awsErr := store.AdminCreateUser(req.UserPoolId, req.Username, req.TemporaryPassword, attrs)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(adminCreateUserResponse{User: userToResponse(user)})
}

func handleAdminGetUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req adminGetUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" || req.Username == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId and Username are required.", http.StatusBadRequest))
	}

	user, awsErr := store.GetUser(req.UserPoolId, req.Username)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	attrs := make([]userAttributeInput, 0, len(user.Attributes))
	for k, v := range user.Attributes {
		attrs = append(attrs, userAttributeInput{Name: k, Value: v})
	}
	return jsonOK(adminGetUserResponse{
		Username:       user.Username,
		UserAttributes: attrs,
		UserCreateDate: float64(user.UserCreateDate.Unix()),
		UserStatus:     string(user.UserStatus),
		Enabled:        user.Enabled,
	})
}

func handleAdminDeleteUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req adminDeleteUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" || req.Username == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId and Username are required.", http.StatusBadRequest))
	}

	if awsErr := store.DeleteUser(req.UserPoolId, req.Username); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleAdminSetUserPassword(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req adminSetUserPasswordRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" || req.Username == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId and Username are required.", http.StatusBadRequest))
	}
	if req.Password == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Password is required.", http.StatusBadRequest))
	}

	if awsErr := store.SetUserPassword(req.UserPoolId, req.Username, req.Password, req.Permanent); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

// ---- Auth handlers ----

func handleSignUp(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req signUpRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClientId == "" || req.Username == "" || req.Password == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"ClientId, Username, and Password are required.", http.StatusBadRequest))
	}

	attrs := attributeSliceToMap(req.UserAttributes)
	user, awsErr := store.SignUp(req.ClientId, req.Username, req.Password, attrs)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(signUpResponse{
		UserConfirmed: user.UserStatus == userStatusConfirmed,
		UserSub:       user.Sub,
	})
}

func handleInitiateAuth(ctx *service.RequestContext, store *Store, keys *KeyStore) (*service.Response, error) {
	var req initiateAuthRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClientId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"ClientId is required.", http.StatusBadRequest))
	}
	if req.AuthFlow != "USER_PASSWORD_AUTH" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Only USER_PASSWORD_AUTH AuthFlow is supported.", http.StatusBadRequest))
	}

	username := req.AuthParameters["USERNAME"]
	password := req.AuthParameters["PASSWORD"]
	if username == "" || password == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"AuthParameters must include USERNAME and PASSWORD.", http.StatusBadRequest))
	}

	result, awsErr := store.InitiateAuth(req.ClientId, username, password, keys)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(initiateAuthResponse{
		AuthenticationResult: authenticationResult{
			AccessToken:  result.AccessToken,
			IdToken:      result.IdToken,
			RefreshToken: result.RefreshToken,
			ExpiresIn:    result.ExpiresIn,
			TokenType:    result.TokenType,
		},
	})
}

func handleAdminConfirmSignUp(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req adminConfirmSignUpRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.UserPoolId == "" || req.Username == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"UserPoolId and Username are required.", http.StatusBadRequest))
	}

	if awsErr := store.ConfirmUser(req.UserPoolId, req.Username); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

