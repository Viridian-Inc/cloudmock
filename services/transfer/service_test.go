package transfer_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/transfer"
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

// ---- Test: UpdateServer ----

func TestTransfer_UpdateServer(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{
		"Protocols": []any{"SFTP"},
	}))
	serverID := respJSON(t, cr)["ServerId"].(string)

	resp, err := s.HandleRequest(jsonCtx("UpdateServer", map[string]any{
		"ServerId":  serverID,
		"Protocols": []any{"SFTP", "FTP"},
	}))
	require.NoError(t, err)
	body := respJSON(t, resp)
	assert.Equal(t, serverID, body["ServerId"])

	// Verify update
	descResp, _ := s.HandleRequest(jsonCtx("DescribeServer", map[string]any{"ServerId": serverID}))
	srv := respJSON(t, descResp)["Server"].(map[string]any)
	protocols := srv["Protocols"].([]any)
	assert.Len(t, protocols, 2)
}

func TestTransfer_UpdateServerNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("UpdateServer", map[string]any{
		"ServerId": "s-ghost",
	}))
	require.Error(t, err)
}

// ---- Test: UpdateUser ----

func TestTransfer_UpdateUser(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, cr)["ServerId"].(string)

	_, _ = s.HandleRequest(jsonCtx("CreateUser", map[string]any{
		"ServerId":      serverID,
		"UserName":      "upd-user",
		"Role":          "arn:aws:iam::123456789012:role/transfer",
		"HomeDirectory": "/home/orig",
	}))

	resp, err := s.HandleRequest(jsonCtx("UpdateUser", map[string]any{
		"ServerId":      serverID,
		"UserName":      "upd-user",
		"HomeDirectory": "/home/updated",
	}))
	require.NoError(t, err)
	body := respJSON(t, resp)
	assert.Equal(t, "upd-user", body["UserName"])

	// Verify
	descResp, _ := s.HandleRequest(jsonCtx("DescribeUser", map[string]any{
		"ServerId": serverID, "UserName": "upd-user",
	}))
	user := respJSON(t, descResp)["User"].(map[string]any)
	assert.Equal(t, "/home/updated", user["HomeDirectory"])
}

// ---- Test: Workflows ----

func TestTransfer_WorkflowCRUD(t *testing.T) {
	s := newService()
	createResp, err := s.HandleRequest(jsonCtx("CreateWorkflow", map[string]any{
		"Description": "Test workflow",
		"Steps": []any{
			map[string]any{"Type": "COPY", "CopyStepDetails": map[string]any{}},
		},
	}))
	require.NoError(t, err)
	wfID := respJSON(t, createResp)["WorkflowId"].(string)
	assert.NotEmpty(t, wfID)

	// Describe
	descResp, err := s.HandleRequest(jsonCtx("DescribeWorkflow", map[string]any{
		"WorkflowId": wfID,
	}))
	require.NoError(t, err)
	wf := respJSON(t, descResp)["Workflow"].(map[string]any)
	assert.Equal(t, wfID, wf["WorkflowId"])
	assert.Equal(t, "Test workflow", wf["Description"])

	// List
	listResp, err := s.HandleRequest(jsonCtx("ListWorkflows", map[string]any{}))
	require.NoError(t, err)
	workflows := respJSON(t, listResp)["Workflows"].([]any)
	assert.Len(t, workflows, 1)

	// Delete
	_, err = s.HandleRequest(jsonCtx("DeleteWorkflow", map[string]any{"WorkflowId": wfID}))
	require.NoError(t, err)

	listResp2, _ := s.HandleRequest(jsonCtx("ListWorkflows", map[string]any{}))
	workflows2 := respJSON(t, listResp2)["Workflows"].([]any)
	assert.Len(t, workflows2, 0)
}

func TestTransfer_WorkflowNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("DescribeWorkflow", map[string]any{"WorkflowId": "w-ghost"}))
	require.Error(t, err)
}

// ---- Test: Tagging ----

func TestTransfer_Tagging(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateServer", map[string]any{}))
	serverID := respJSON(t, cr)["ServerId"].(string)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeServer", map[string]any{"ServerId": serverID}))
	arn := respJSON(t, descResp)["Server"].(map[string]any)["Arn"].(string)

	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"Arn":  arn,
		"Tags": []any{map[string]any{"Key": "env", "Value": "test"}},
	}))
	require.NoError(t, err)

	listResp, err := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"Arn": arn}))
	require.NoError(t, err)
	tags := respJSON(t, listResp)["Tags"].([]any)
	assert.Len(t, tags, 1)

	_, err = s.HandleRequest(jsonCtx("UntagResource", map[string]any{
		"Arn":     arn,
		"TagKeys": []any{"env"},
	}))
	require.NoError(t, err)

	listResp2, _ := s.HandleRequest(jsonCtx("ListTagsForResource", map[string]any{"Arn": arn}))
	tags2 := respJSON(t, listResp2)["Tags"].([]any)
	assert.Len(t, tags2, 0)
}

func TestTransfer_TaggingNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("TagResource", map[string]any{
		"Arn":  "arn:aws:transfer:us-east-1:123456789012:server/ghost",
		"Tags": []any{},
	}))
	require.Error(t, err)
}
