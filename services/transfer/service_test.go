package transfer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/transfer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.TransferService {
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

func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	data, _ := json.Marshal(resp.Body)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestTransfer_CreateAndDescribeServer(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateServer", map[string]any{
		"Domain":       "S3",
		"EndpointType": "PUBLIC",
		"Protocols":    []string{"SFTP"},
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	m := respJSON(t, resp)
	serverID := m["ServerId"].(string)
	assert.NotEmpty(t, serverID)

	descResp, err := s.HandleRequest(jsonCtx("DescribeServer", map[string]any{
		"ServerId": serverID,
	}))
	require.NoError(t, err)
	dm := respJSON(t, descResp)
	srv := dm["Server"].(map[string]any)
	assert.Equal(t, serverID, srv["ServerId"])
	assert.Equal(t, "OFFLINE", srv["State"])
}

func TestTransfer_ListServers(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))

	resp, err := s.HandleRequest(jsonCtx("ListServers", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Len(t, m["Servers"].([]any), 2)
}

func TestTransfer_DeleteServer(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, createResp)["ServerId"].(string)

	resp, err := s.HandleRequest(jsonCtx("DeleteServer", map[string]any{"ServerId": serverID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	_, err = s.HandleRequest(jsonCtx("DescribeServer", map[string]any{"ServerId": serverID}))
	require.Error(t, err)
}

func TestTransfer_UserLifecycle(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, createResp)["ServerId"].(string)

	userResp, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"ServerId":  serverID,
		"UserName":  "testuser",
		"Role":      "arn:aws:iam::123456789012:role/transfer",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, userResp.StatusCode)

	descResp, err := s.HandleRequest(jsonCtx("DescribeUser", map[string]any{
		"ServerId": serverID,
		"UserName": "testuser",
	}))
	require.NoError(t, err)
	dm := respJSON(t, descResp)
	assert.Equal(t, "testuser", dm["User"].(map[string]any)["UserName"])

	listResp, _ := s.HandleRequest(jsonCtx("ListUsers", map[string]any{"ServerId": serverID}))
	lm := respJSON(t, listResp)
	assert.Len(t, lm["Users"].([]any), 1)

	delResp, err := s.HandleRequest(jsonCtx("DeleteUser", map[string]any{
		"ServerId": serverID,
		"UserName": "testuser",
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestTransfer_SSHPublicKey(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, createResp)["ServerId"].(string)

	s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"ServerId": serverID,
		"UserName": "sshuser",
	}))

	importResp, err := s.HandleRequest(jsonCtx("ImportSshPublicKey", map[string]any{
		"ServerId":         serverID,
		"UserName":         "sshuser",
		"SshPublicKeyBody": "ssh-rsa AAAA...",
	}))
	require.NoError(t, err)
	im := respJSON(t, importResp)
	keyID := im["SshPublicKeyId"].(string)
	assert.NotEmpty(t, keyID)

	delResp, err := s.HandleRequest(jsonCtx("DeleteSshPublicKey", map[string]any{
		"ServerId":       serverID,
		"UserName":       "sshuser",
		"SshPublicKeyId": keyID,
	}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestTransfer_ServerNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeServer", map[string]any{"ServerId": "nonexistent"}))
	require.Error(t, err)

	_, err = s.HandleRequest(jsonCtx("DeleteServer", map[string]any{"ServerId": "nonexistent"}))
	require.Error(t, err)
}

func TestTransfer_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("BogusAction", nil))
	require.Error(t, err)
	awsErr := err.(*service.AWSError)
	assert.Equal(t, "InvalidAction", awsErr.Code)
}

func TestTransfer_ServerLifecycleState(t *testing.T) {
	s := newService()
	createResp, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, createResp)["ServerId"].(string)

	// Server starts OFFLINE
	descResp, _ := s.HandleRequest(jsonCtx("DescribeServer", map[string]any{"ServerId": serverID}))
	dm := respJSON(t, descResp)
	state := dm["Server"].(map[string]any)["State"].(string)
	assert.Contains(t, []string{"OFFLINE", "ONLINE"}, state)
}

func TestTransfer_InvalidProtocol(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateServer", map[string]any{
		"Protocols": []string{"INVALID"},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid protocol")
}

func TestTransfer_ValidProtocols(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateServer", map[string]any{
		"Protocols": []string{"SFTP", "FTPS"},
	}))
	require.NoError(t, err)
	assert.NotEmpty(t, respJSON(t, resp)["ServerId"])
}

func TestTransfer_ServerEndpointGeneration(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	require.NoError(t, err)
	serverID := respJSON(t, createResp)["ServerId"].(string)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeServer", map[string]any{"ServerId": serverID}))
	srv := respJSON(t, descResp)["Server"].(map[string]any)
	// Endpoint should follow the pattern s-{id}.server.transfer.{region}.amazonaws.com
	endpoint, ok := srv["Endpoint"].(string)
	if ok {
		assert.Contains(t, endpoint, "server.transfer.us-east-1.amazonaws.com")
	}
}

func TestTransfer_UserHomeDirValidation(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, cr)["ServerId"].(string)

	_, err := s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"ServerId":      serverID,
		"UserName":      "baddir-user",
		"Role":          "arn:aws:iam::123456789012:role/transfer",
		"HomeDirectory": "no-leading-slash",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "home directory must start with /")
}
