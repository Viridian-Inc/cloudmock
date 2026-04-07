package cognito_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	cognitosvc "github.com/Viridian-Inc/cloudmock/services/cognito"
)

// newCognitoGateway builds a full gateway stack with the Cognito service registered and IAM disabled.
func newCognitoGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(cognitosvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// cognitoReq builds a JSON POST request targeting the Cognito User Pools service via X-Amz-Target.
func cognitoReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("cognitoReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCognitoIdentityProviderService."+action)
	// Authorization header places "cognito-idp" as the service in the credential scope.
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cognito-idp/aws4_request, SignedHeaders=host;x-amz-target, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// ---- Test 1: CreateUserPool + ListUserPools + DescribeUserPool ----

func TestCognito_UserPool_CreateListDescribe(t *testing.T) {
	handler := newCognitoGateway(t)

	// CreateUserPool
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "my-test-pool",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateUserPool: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	mc := decodeJSON(t, wc.Body.String())
	pool, ok := mc["UserPool"].(map[string]any)
	if !ok {
		t.Fatalf("CreateUserPool: missing UserPool in response\nbody: %s", wc.Body.String())
	}
	poolID, _ := pool["Id"].(string)
	if poolID == "" {
		t.Fatal("CreateUserPool: missing UserPool.Id")
	}
	arn, _ := pool["Arn"].(string)
	if !strings.Contains(arn, poolID) {
		t.Errorf("CreateUserPool: Arn %q does not contain pool ID %q", arn, poolID)
	}
	if pool["Name"].(string) != "my-test-pool" {
		t.Errorf("CreateUserPool: expected Name=my-test-pool, got %q", pool["Name"])
	}

	// ListUserPools
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, cognitoReq(t, "ListUserPools", map[string]any{
		"MaxResults": 10,
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListUserPools: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	pools, ok := ml["UserPools"].([]any)
	if !ok || len(pools) == 0 {
		t.Fatalf("ListUserPools: expected non-empty UserPools\nbody: %s", wl.Body.String())
	}
	found := false
	for _, p := range pools {
		entry := p.(map[string]any)
		if entry["Id"].(string) == poolID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListUserPools: pool %q not found in list", poolID)
	}

	// DescribeUserPool
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, cognitoReq(t, "DescribeUserPool", map[string]any{
		"UserPoolId": poolID,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeUserPool: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	md := decodeJSON(t, wd.Body.String())
	poolDesc, ok := md["UserPool"].(map[string]any)
	if !ok {
		t.Fatalf("DescribeUserPool: missing UserPool in response\nbody: %s", wd.Body.String())
	}
	if poolDesc["Id"].(string) != poolID {
		t.Errorf("DescribeUserPool: expected Id=%q, got %q", poolID, poolDesc["Id"])
	}
	if poolDesc["Name"].(string) != "my-test-pool" {
		t.Errorf("DescribeUserPool: expected Name=my-test-pool, got %q", poolDesc["Name"])
	}
}

// ---- Test 2: CreateUserPoolClient + DescribeUserPoolClient ----

func TestCognito_UserPoolClient_CreateDescribe(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "client-test-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	mcp := decodeJSON(t, wcp.Body.String())
	poolID := mcp["UserPool"].(map[string]any)["Id"].(string)

	// CreateUserPoolClient without secret.
	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientName": "my-app-client",
		"ExplicitAuthFlows": []string{"ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH"},
	}))
	if wcc.Code != http.StatusOK {
		t.Fatalf("CreateUserPoolClient: expected 200, got %d\nbody: %s", wcc.Code, wcc.Body.String())
	}
	mcc := decodeJSON(t, wcc.Body.String())
	clientObj, ok := mcc["UserPoolClient"].(map[string]any)
	if !ok {
		t.Fatalf("CreateUserPoolClient: missing UserPoolClient in response\nbody: %s", wcc.Body.String())
	}
	clientID, _ := clientObj["ClientId"].(string)
	if clientID == "" {
		t.Fatal("CreateUserPoolClient: missing ClientId")
	}
	if clientObj["ClientName"].(string) != "my-app-client" {
		t.Errorf("CreateUserPoolClient: expected ClientName=my-app-client, got %q", clientObj["ClientName"])
	}
	if clientObj["ClientSecret"] != nil && clientObj["ClientSecret"].(string) != "" {
		t.Errorf("CreateUserPoolClient: unexpected ClientSecret when GenerateSecret=false")
	}

	// CreateUserPoolClient with secret.
	wccs := httptest.NewRecorder()
	handler.ServeHTTP(wccs, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId":     poolID,
		"ClientName":     "secret-client",
		"GenerateSecret": true,
	}))
	if wccs.Code != http.StatusOK {
		t.Fatalf("CreateUserPoolClient with secret: expected 200, got %d\nbody: %s", wccs.Code, wccs.Body.String())
	}
	mccs := decodeJSON(t, wccs.Body.String())
	secretClient := mccs["UserPoolClient"].(map[string]any)
	secretVal, _ := secretClient["ClientSecret"].(string)
	if secretVal == "" {
		t.Error("CreateUserPoolClient with secret: expected non-empty ClientSecret")
	}

	// DescribeUserPoolClient
	wdc := httptest.NewRecorder()
	handler.ServeHTTP(wdc, cognitoReq(t, "DescribeUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientId":   clientID,
	}))
	if wdc.Code != http.StatusOK {
		t.Fatalf("DescribeUserPoolClient: expected 200, got %d\nbody: %s", wdc.Code, wdc.Body.String())
	}
	mdc := decodeJSON(t, wdc.Body.String())
	clientDesc := mdc["UserPoolClient"].(map[string]any)
	if clientDesc["ClientId"].(string) != clientID {
		t.Errorf("DescribeUserPoolClient: expected ClientId=%q, got %q", clientID, clientDesc["ClientId"])
	}
}

// ---- Test 3: AdminCreateUser + AdminGetUser ----

func TestCognito_AdminCreateAndGetUser(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "user-test-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	mcp := decodeJSON(t, wcp.Body.String())
	poolID := mcp["UserPool"].(map[string]any)["Id"].(string)

	// AdminCreateUser
	wcu := httptest.NewRecorder()
	handler.ServeHTTP(wcu, cognitoReq(t, "AdminCreateUser", map[string]any{
		"UserPoolId":        poolID,
		"Username":          "testuser@example.com",
		"TemporaryPassword": "TempPass123!",
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": "testuser@example.com"},
		},
	}))
	if wcu.Code != http.StatusOK {
		t.Fatalf("AdminCreateUser: expected 200, got %d\nbody: %s", wcu.Code, wcu.Body.String())
	}
	mcu := decodeJSON(t, wcu.Body.String())
	user, ok := mcu["User"].(map[string]any)
	if !ok {
		t.Fatalf("AdminCreateUser: missing User in response\nbody: %s", wcu.Body.String())
	}
	if user["Username"].(string) != "testuser@example.com" {
		t.Errorf("AdminCreateUser: expected Username=testuser@example.com, got %q", user["Username"])
	}
	if user["UserStatus"].(string) != "FORCE_CHANGE_PASSWORD" {
		t.Errorf("AdminCreateUser: expected UserStatus=FORCE_CHANGE_PASSWORD, got %q", user["UserStatus"])
	}
	if !user["Enabled"].(bool) {
		t.Error("AdminCreateUser: expected Enabled=true")
	}

	// Check attributes contain sub.
	attrs, _ := user["Attributes"].([]any)
	attrMap := make(map[string]string)
	for _, a := range attrs {
		entry := a.(map[string]any)
		attrMap[entry["Name"].(string)] = entry["Value"].(string)
	}
	if attrMap["sub"] == "" {
		t.Error("AdminCreateUser: missing sub attribute")
	}
	if attrMap["email"] != "testuser@example.com" {
		t.Errorf("AdminCreateUser: expected email attribute, got %q", attrMap["email"])
	}

	// AdminGetUser
	wgu := httptest.NewRecorder()
	handler.ServeHTTP(wgu, cognitoReq(t, "AdminGetUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "testuser@example.com",
	}))
	if wgu.Code != http.StatusOK {
		t.Fatalf("AdminGetUser: expected 200, got %d\nbody: %s", wgu.Code, wgu.Body.String())
	}
	mgu := decodeJSON(t, wgu.Body.String())
	if mgu["Username"].(string) != "testuser@example.com" {
		t.Errorf("AdminGetUser: expected Username=testuser@example.com, got %q", mgu["Username"])
	}

	// AdminGetUser — not found.
	wguf := httptest.NewRecorder()
	handler.ServeHTTP(wguf, cognitoReq(t, "AdminGetUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "nobody@example.com",
	}))
	if wguf.Code != http.StatusBadRequest {
		t.Fatalf("AdminGetUser not found: expected 400, got %d", wguf.Code)
	}
}

// ---- Test 4: SignUp + InitiateAuth ----

func TestCognito_SignUpAndInitiateAuth(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "auth-test-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	mcp := decodeJSON(t, wcp.Body.String())
	poolID := mcp["UserPool"].(map[string]any)["Id"].(string)

	// Create client.
	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientName": "auth-client",
	}))
	if wcc.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPoolClient: %d %s", wcc.Code, wcc.Body.String())
	}
	mcc := decodeJSON(t, wcc.Body.String())
	clientID := mcc["UserPoolClient"].(map[string]any)["ClientId"].(string)

	// SignUp
	wsu := httptest.NewRecorder()
	handler.ServeHTTP(wsu, cognitoReq(t, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "newuser@example.com",
		"Password": "NewPass123!",
		"UserAttributes": []map[string]string{
			{"Name": "email", "Value": "newuser@example.com"},
		},
	}))
	if wsu.Code != http.StatusOK {
		t.Fatalf("SignUp: expected 200, got %d\nbody: %s", wsu.Code, wsu.Body.String())
	}
	msu := decodeJSON(t, wsu.Body.String())
	userSub, _ := msu["UserSub"].(string)
	if userSub == "" {
		t.Error("SignUp: missing UserSub in response")
	}
	if msu["UserConfirmed"].(bool) {
		t.Error("SignUp: expected UserConfirmed=false for new signup")
	}

	// Confirm user via AdminConfirmSignUp before auth.
	wconf := httptest.NewRecorder()
	handler.ServeHTTP(wconf, cognitoReq(t, "AdminConfirmSignUp", map[string]any{
		"UserPoolId": poolID,
		"Username":   "newuser@example.com",
	}))
	if wconf.Code != http.StatusOK {
		t.Fatalf("AdminConfirmSignUp: expected 200, got %d\nbody: %s", wconf.Code, wconf.Body.String())
	}

	// InitiateAuth — USER_PASSWORD_AUTH
	wia := httptest.NewRecorder()
	handler.ServeHTTP(wia, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "newuser@example.com",
			"PASSWORD": "NewPass123!",
		},
	}))
	if wia.Code != http.StatusOK {
		t.Fatalf("InitiateAuth: expected 200, got %d\nbody: %s", wia.Code, wia.Body.String())
	}
	mia := decodeJSON(t, wia.Body.String())
	authResult, ok := mia["AuthenticationResult"].(map[string]any)
	if !ok {
		t.Fatalf("InitiateAuth: missing AuthenticationResult\nbody: %s", wia.Body.String())
	}
	accessToken, _ := authResult["AccessToken"].(string)
	if accessToken == "" {
		t.Error("InitiateAuth: missing AccessToken")
	}
	if !strings.Contains(accessToken, ".") {
		t.Error("InitiateAuth: AccessToken should look like a JWT (contain dots)")
	}
	idToken, _ := authResult["IdToken"].(string)
	if idToken == "" {
		t.Error("InitiateAuth: missing IdToken")
	}
	refreshToken, _ := authResult["RefreshToken"].(string)
	if refreshToken == "" {
		t.Error("InitiateAuth: missing RefreshToken")
	}
	if int(authResult["ExpiresIn"].(float64)) != 3600 {
		t.Errorf("InitiateAuth: expected ExpiresIn=3600, got %v", authResult["ExpiresIn"])
	}
	if authResult["TokenType"].(string) != "Bearer" {
		t.Errorf("InitiateAuth: expected TokenType=Bearer, got %q", authResult["TokenType"])
	}

	// InitiateAuth — wrong password
	wiaf := httptest.NewRecorder()
	handler.ServeHTTP(wiaf, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "newuser@example.com",
			"PASSWORD": "WrongPass!",
		},
	}))
	if wiaf.Code != http.StatusBadRequest {
		t.Fatalf("InitiateAuth wrong password: expected 400, got %d", wiaf.Code)
	}

	// InitiateAuth — unsupported auth flow
	wiainv := httptest.NewRecorder()
	handler.ServeHTTP(wiainv, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "CUSTOM_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "newuser@example.com",
			"PASSWORD": "NewPass123!",
		},
	}))
	if wiainv.Code != http.StatusBadRequest {
		t.Fatalf("InitiateAuth unsupported flow: expected 400, got %d", wiainv.Code)
	}
}

// ---- Test 5: AdminConfirmSignUp ----

func TestCognito_AdminConfirmSignUp(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool and client.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "confirm-test-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	mcp := decodeJSON(t, wcp.Body.String())
	poolID := mcp["UserPool"].(map[string]any)["Id"].(string)

	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientName": "confirm-client",
	}))
	if wcc.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPoolClient: %d %s", wcc.Code, wcc.Body.String())
	}
	mcc := decodeJSON(t, wcc.Body.String())
	clientID := mcc["UserPoolClient"].(map[string]any)["ClientId"].(string)

	// SignUp user.
	wsu := httptest.NewRecorder()
	handler.ServeHTTP(wsu, cognitoReq(t, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "confirm-me@example.com",
		"Password": "Pass123!",
	}))
	if wsu.Code != http.StatusOK {
		t.Fatalf("SignUp: %d %s", wsu.Code, wsu.Body.String())
	}

	// Attempt InitiateAuth before confirmation — should fail with UserNotConfirmedException.
	wiaPre := httptest.NewRecorder()
	handler.ServeHTTP(wiaPre, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "confirm-me@example.com",
			"PASSWORD": "Pass123!",
		},
	}))
	if wiaPre.Code != http.StatusBadRequest {
		t.Fatalf("InitiateAuth unconfirmed: expected 400, got %d", wiaPre.Code)
	}

	// AdminConfirmSignUp.
	wconf := httptest.NewRecorder()
	handler.ServeHTTP(wconf, cognitoReq(t, "AdminConfirmSignUp", map[string]any{
		"UserPoolId": poolID,
		"Username":   "confirm-me@example.com",
	}))
	if wconf.Code != http.StatusOK {
		t.Fatalf("AdminConfirmSignUp: expected 200, got %d\nbody: %s", wconf.Code, wconf.Body.String())
	}

	// Verify status via AdminGetUser.
	wgu := httptest.NewRecorder()
	handler.ServeHTTP(wgu, cognitoReq(t, "AdminGetUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "confirm-me@example.com",
	}))
	if wgu.Code != http.StatusOK {
		t.Fatalf("AdminGetUser after confirm: %d %s", wgu.Code, wgu.Body.String())
	}
	mgu := decodeJSON(t, wgu.Body.String())
	if mgu["UserStatus"].(string) != "CONFIRMED" {
		t.Errorf("AdminConfirmSignUp: expected UserStatus=CONFIRMED, got %q", mgu["UserStatus"])
	}

	// Now InitiateAuth should succeed.
	wia := httptest.NewRecorder()
	handler.ServeHTTP(wia, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "confirm-me@example.com",
			"PASSWORD": "Pass123!",
		},
	}))
	if wia.Code != http.StatusOK {
		t.Fatalf("InitiateAuth after confirm: expected 200, got %d\nbody: %s", wia.Code, wia.Body.String())
	}
}

// ---- Test 6: DeleteUserPool ----

func TestCognito_DeleteUserPool(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "delete-test-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	mcp := decodeJSON(t, wcp.Body.String())
	poolID := mcp["UserPool"].(map[string]any)["Id"].(string)

	// Verify it exists.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, cognitoReq(t, "DescribeUserPool", map[string]any{
		"UserPoolId": poolID,
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeUserPool before delete: %d %s", wdesc.Code, wdesc.Body.String())
	}

	// Delete pool.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, cognitoReq(t, "DeleteUserPool", map[string]any{
		"UserPoolId": poolID,
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteUserPool: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// Verify it's gone.
	wafter := httptest.NewRecorder()
	handler.ServeHTTP(wafter, cognitoReq(t, "DescribeUserPool", map[string]any{
		"UserPoolId": poolID,
	}))
	if wafter.Code != http.StatusBadRequest {
		t.Fatalf("DescribeUserPool after delete: expected 400, got %d", wafter.Code)
	}

	// Delete again — should fail.
	wdel2 := httptest.NewRecorder()
	handler.ServeHTTP(wdel2, cognitoReq(t, "DeleteUserPool", map[string]any{
		"UserPoolId": poolID,
	}))
	if wdel2.Code != http.StatusBadRequest {
		t.Fatalf("DeleteUserPool nonexistent: expected 400, got %d", wdel2.Code)
	}
}

// ---- Additional edge case: AdminDeleteUser ----

func TestCognito_AdminDeleteUser(t *testing.T) {
	handler := newCognitoGateway(t)

	// Setup pool + user.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "del-user-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	poolID := decodeJSON(t, wcp.Body.String())["UserPool"].(map[string]any)["Id"].(string)

	wcu := httptest.NewRecorder()
	handler.ServeHTTP(wcu, cognitoReq(t, "AdminCreateUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "to-delete@example.com",
	}))
	if wcu.Code != http.StatusOK {
		t.Fatalf("setup AdminCreateUser: %d %s", wcu.Code, wcu.Body.String())
	}

	// Delete user.
	wdu := httptest.NewRecorder()
	handler.ServeHTTP(wdu, cognitoReq(t, "AdminDeleteUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "to-delete@example.com",
	}))
	if wdu.Code != http.StatusOK {
		t.Fatalf("AdminDeleteUser: expected 200, got %d\nbody: %s", wdu.Code, wdu.Body.String())
	}

	// Verify user is gone.
	wgu := httptest.NewRecorder()
	handler.ServeHTTP(wgu, cognitoReq(t, "AdminGetUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "to-delete@example.com",
	}))
	if wgu.Code != http.StatusBadRequest {
		t.Fatalf("AdminGetUser after delete: expected 400, got %d", wgu.Code)
	}
}

// ---- Test: DescribeUserPool — ResourceNotFoundException ----

func TestCognito_DescribeUserPool_NotFound(t *testing.T) {
	handler := newCognitoGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "DescribeUserPool", map[string]any{
		"UserPoolId": "us-east-1_NONEXISTENT",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DescribeUserPool not found: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "ResourceNotFoundException") {
		t.Errorf("DescribeUserPool not found: expected ResourceNotFoundException in body\nbody: %s", body)
	}
}

// ---- Test: ListUserPools — empty and populated ----

func TestCognito_ListUserPools_Empty(t *testing.T) {
	handler := newCognitoGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "ListUserPools", map[string]any{
		"MaxResults": 10,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListUserPools empty: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	pools, _ := m["UserPools"].([]any)
	if len(pools) != 0 {
		t.Errorf("ListUserPools empty: expected 0 pools, got %d", len(pools))
	}
}

// ---- Test: SignUp — UsernameExistsException ----

func TestCognito_SignUp_UsernameExistsException(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool + client.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "dup-signup-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	poolID := decodeJSON(t, wcp.Body.String())["UserPool"].(map[string]any)["Id"].(string)

	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientName": "dup-client",
	}))
	if wcc.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPoolClient: %d %s", wcc.Code, wcc.Body.String())
	}
	clientID := decodeJSON(t, wcc.Body.String())["UserPoolClient"].(map[string]any)["ClientId"].(string)

	// First sign-up.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, cognitoReq(t, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "duplicate@example.com",
		"Password": "Pass123!",
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("first SignUp: %d %s", w1.Code, w1.Body.String())
	}

	// Second sign-up — same username.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, cognitoReq(t, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "duplicate@example.com",
		"Password": "Pass456!",
	}))
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("duplicate SignUp: expected 400, got %d\nbody: %s", w2.Code, w2.Body.String())
	}
	body := w2.Body.String()
	if !strings.Contains(body, "UsernameExistsException") {
		t.Errorf("duplicate SignUp: expected UsernameExistsException in body\nbody: %s", body)
	}
}

// ---- Test: AdminCreateUser — UsernameExistsException ----

func TestCognito_AdminCreateUser_UsernameExistsException(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "dup-admin-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	poolID := decodeJSON(t, wcp.Body.String())["UserPool"].(map[string]any)["Id"].(string)

	// First admin create.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, cognitoReq(t, "AdminCreateUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "admin-dup@example.com",
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("first AdminCreateUser: %d %s", w1.Code, w1.Body.String())
	}

	// Second admin create — same username.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, cognitoReq(t, "AdminCreateUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "admin-dup@example.com",
	}))
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("duplicate AdminCreateUser: expected 400, got %d\nbody: %s", w2.Code, w2.Body.String())
	}
	body := w2.Body.String()
	if !strings.Contains(body, "UsernameExistsException") {
		t.Errorf("duplicate AdminCreateUser: expected UsernameExistsException in body\nbody: %s", body)
	}
}

// ---- Test: InitiateAuth — NotAuthorizedException (explicit error code check) ----

func TestCognito_InitiateAuth_NotAuthorizedException(t *testing.T) {
	handler := newCognitoGateway(t)

	// Setup: pool, client, user.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "notauth-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", wcp.Code, wcp.Body.String())
	}
	poolID := decodeJSON(t, wcp.Body.String())["UserPool"].(map[string]any)["Id"].(string)

	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientName": "notauth-client",
	}))
	if wcc.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", wcc.Code, wcc.Body.String())
	}
	clientID := decodeJSON(t, wcc.Body.String())["UserPoolClient"].(map[string]any)["ClientId"].(string)

	wsu := httptest.NewRecorder()
	handler.ServeHTTP(wsu, cognitoReq(t, "SignUp", map[string]any{
		"ClientId": clientID,
		"Username": "authuser@example.com",
		"Password": "CorrectPass123!",
	}))
	if wsu.Code != http.StatusOK {
		t.Fatalf("setup SignUp: %d %s", wsu.Code, wsu.Body.String())
	}

	// Confirm the user.
	wconf := httptest.NewRecorder()
	handler.ServeHTTP(wconf, cognitoReq(t, "AdminConfirmSignUp", map[string]any{
		"UserPoolId": poolID,
		"Username":   "authuser@example.com",
	}))
	if wconf.Code != http.StatusOK {
		t.Fatalf("setup AdminConfirmSignUp: %d %s", wconf.Code, wconf.Body.String())
	}

	// Wrong password should return NotAuthorizedException.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "authuser@example.com",
			"PASSWORD": "WrongPass!",
		},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("InitiateAuth wrong password: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "NotAuthorizedException") {
		t.Errorf("InitiateAuth wrong password: expected NotAuthorizedException in body\nbody: %s", body)
	}
}

// ---- Test: ListUserPoolClients ----

func TestCognito_ListUserPoolClients(t *testing.T) {
	handler := newCognitoGateway(t)

	// Create pool.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "list-clients-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup CreateUserPool: %d %s", wcp.Code, wcp.Body.String())
	}
	poolID := decodeJSON(t, wcp.Body.String())["UserPool"].(map[string]any)["Id"].(string)

	// Create two clients.
	for _, name := range []string{"client-a", "client-b"} {
		wcc := httptest.NewRecorder()
		handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
			"UserPoolId": poolID,
			"ClientName": name,
		}))
		if wcc.Code != http.StatusOK {
			t.Fatalf("setup CreateUserPoolClient %s: %d %s", name, wcc.Code, wcc.Body.String())
		}
	}

	// ListUserPoolClients.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "ListUserPoolClients", map[string]any{
		"UserPoolId": poolID,
		"MaxResults": 10,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ListUserPoolClients: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	m := decodeJSON(t, w.Body.String())
	clients, ok := m["UserPoolClients"].([]any)
	if !ok || len(clients) < 2 {
		t.Fatalf("ListUserPoolClients: expected 2+ clients, got %v\nbody: %s", clients, w.Body.String())
	}

	// ListUserPoolClients on non-existent pool.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, cognitoReq(t, "ListUserPoolClients", map[string]any{
		"UserPoolId": "us-east-1_NONEXISTENT",
		"MaxResults": 10,
	}))
	if wne.Code != http.StatusBadRequest {
		t.Fatalf("ListUserPoolClients nonexistent pool: expected 400, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test: AdminSetUserPassword ----

func TestCognito_AdminSetUserPassword(t *testing.T) {
	handler := newCognitoGateway(t)

	// Setup pool + user.
	wcp := httptest.NewRecorder()
	handler.ServeHTTP(wcp, cognitoReq(t, "CreateUserPool", map[string]any{
		"PoolName": "set-pw-pool",
	}))
	if wcp.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", wcp.Code, wcp.Body.String())
	}
	poolID := decodeJSON(t, wcp.Body.String())["UserPool"].(map[string]any)["Id"].(string)

	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, cognitoReq(t, "CreateUserPoolClient", map[string]any{
		"UserPoolId": poolID,
		"ClientName": "pw-client",
	}))
	if wcc.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", wcc.Code, wcc.Body.String())
	}
	clientID := decodeJSON(t, wcc.Body.String())["UserPoolClient"].(map[string]any)["ClientId"].(string)

	wcu := httptest.NewRecorder()
	handler.ServeHTTP(wcu, cognitoReq(t, "AdminCreateUser", map[string]any{
		"UserPoolId":        poolID,
		"Username":          "pw-user@example.com",
		"TemporaryPassword": "TempPass123!",
	}))
	if wcu.Code != http.StatusOK {
		t.Fatalf("setup: %d %s", wcu.Code, wcu.Body.String())
	}

	// Set permanent password.
	wsp := httptest.NewRecorder()
	handler.ServeHTTP(wsp, cognitoReq(t, "AdminSetUserPassword", map[string]any{
		"UserPoolId": poolID,
		"Username":   "pw-user@example.com",
		"Password":   "NewPerm123!",
		"Permanent":  true,
	}))
	if wsp.Code != http.StatusOK {
		t.Fatalf("AdminSetUserPassword: expected 200, got %d\nbody: %s", wsp.Code, wsp.Body.String())
	}

	// Verify user status is CONFIRMED after permanent password.
	wgu := httptest.NewRecorder()
	handler.ServeHTTP(wgu, cognitoReq(t, "AdminGetUser", map[string]any{
		"UserPoolId": poolID,
		"Username":   "pw-user@example.com",
	}))
	if wgu.Code != http.StatusOK {
		t.Fatalf("AdminGetUser: %d %s", wgu.Code, wgu.Body.String())
	}
	mgu := decodeJSON(t, wgu.Body.String())
	if mgu["UserStatus"].(string) != "CONFIRMED" {
		t.Errorf("AdminSetUserPassword permanent: expected UserStatus=CONFIRMED, got %q", mgu["UserStatus"])
	}

	// Now auth should work with the new password.
	wia := httptest.NewRecorder()
	handler.ServeHTTP(wia, cognitoReq(t, "InitiateAuth", map[string]any{
		"AuthFlow": "USER_PASSWORD_AUTH",
		"ClientId": clientID,
		"AuthParameters": map[string]string{
			"USERNAME": "pw-user@example.com",
			"PASSWORD": "NewPerm123!",
		},
	}))
	if wia.Code != http.StatusOK {
		t.Fatalf("InitiateAuth after password set: expected 200, got %d\nbody: %s", wia.Code, wia.Body.String())
	}

	// AdminSetUserPassword on non-existent user.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, cognitoReq(t, "AdminSetUserPassword", map[string]any{
		"UserPoolId": poolID,
		"Username":   "nonexistent@example.com",
		"Password":   "Pass123!",
		"Permanent":  true,
	}))
	if wne.Code != http.StatusBadRequest {
		t.Fatalf("AdminSetUserPassword nonexistent: expected 400, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Unknown action ----

func TestCognito_UnknownAction(t *testing.T) {
	handler := newCognitoGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cognitoReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
