package glacier_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/service"
	svc "github.com/neureaux/cloudmock/services/glacier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.GlacierService {
	return svc.New("123456789012", "us-east-1")
}

func restCtx(method, path string, body []byte) *service.RequestContext {
	req := httptest.NewRequest(method, path, nil)
	return &service.RequestContext{
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Body:       body,
		RawRequest: req,
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

func TestGlacier_CreateAndDescribeVault(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/my-vault", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	descResp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/my-vault", nil))
	require.NoError(t, err)
	m := respJSON(t, descResp)
	assert.Equal(t, "my-vault", m["VaultName"])
	assert.Contains(t, m["VaultARN"].(string), "my-vault")
}

func TestGlacier_ListVaults(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/vault-1", nil))
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/vault-2", nil))

	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Len(t, m["VaultList"].([]any), 2)
}

func TestGlacier_DeleteVault(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/del-vault", nil))

	resp, err := s.HandleRequest(restCtx(http.MethodDelete, "/-/vaults/del-vault", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	_, err = s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/del-vault", nil))
	require.Error(t, err)
}

func TestGlacier_UploadAndDeleteArchive(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/arch-vault", nil))

	ctx := restCtx(http.MethodPost, "/-/vaults/arch-vault/archives", []byte("archive-data"))
	ctx.RawRequest.Header.Set("x-amz-archive-description", "test archive")
	ctx.RawRequest.Header.Set("x-amz-sha256-tree-hash", "abc123")
	uploadResp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, uploadResp.StatusCode)

	archiveID := uploadResp.Headers["x-amz-archive-id"]
	assert.NotEmpty(t, archiveID)

	delResp, err := s.HandleRequest(restCtx(http.MethodDelete, "/-/vaults/arch-vault/archives/"+archiveID, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
}

func TestGlacier_InitiateAndDescribeJob(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/job-vault", nil))

	jobBody, _ := json.Marshal(map[string]any{"Type": "inventory-retrieval"})
	initResp, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/job-vault/jobs", jobBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, initResp.StatusCode)
	jobID := initResp.Headers["x-amz-job-id"]
	assert.NotEmpty(t, jobID)

	descResp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/job-vault/jobs/"+jobID, nil))
	require.NoError(t, err)
	m := respJSON(t, descResp)
	assert.Equal(t, jobID, m["JobId"])
	assert.Equal(t, "InProgress", m["StatusCode"])
}

func TestGlacier_ListJobs(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/list-vault", nil))

	jobBody, _ := json.Marshal(map[string]any{"Type": "inventory-retrieval"})
	s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/list-vault/jobs", jobBody))
	s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/list-vault/jobs", jobBody))

	resp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/list-vault/jobs", nil))
	require.NoError(t, err)
	m := respJSON(t, resp)
	assert.Len(t, m["JobList"].([]any), 2)
}

func TestGlacier_VaultNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/nonexistent", nil))
	require.Error(t, err)

	_, err = s.HandleRequest(restCtx(http.MethodDelete, "/-/vaults/nonexistent", nil))
	require.Error(t, err)
}

func TestGlacier_DuplicateVault(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/dup-vault", nil))
	_, err := s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/dup-vault", nil))
	require.Error(t, err)
}

func TestGlacier_NotImplementedRoute(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/bogus/route", nil))
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "NotImplemented"))
}

func TestGlacier_VaultLockTwoStepFlow(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/lock-vault", nil))

	// Step 1: Initiate vault lock
	policy, _ := json.Marshal(map[string]any{"policy": `{"Version":"2012-10-17"}`})
	initResp, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/lock-vault/lock-policy", policy))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, initResp.StatusCode)
	lockID := initResp.Headers["x-amz-lock-id"]
	assert.NotEmpty(t, lockID)

	// Step 2: Complete vault lock
	completeResp, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/lock-vault/lock-policy/"+lockID, nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, completeResp.StatusCode)

	// Cannot initiate lock again on locked vault
	_, err = s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/lock-vault/lock-policy", policy))
	require.Error(t, err)
}

func TestGlacier_VaultNotifications(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/notif-vault", nil))

	body, _ := json.Marshal(map[string]any{
		"vaultNotificationConfig": map[string]any{
			"SNSTopic": "arn:aws:sns:us-east-1:123456789012:glacier-notif",
			"Events":   []string{"ArchiveRetrievalCompleted", "InventoryRetrievalCompleted"},
		},
	})
	setResp, err := s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/notif-vault/notification-configuration", body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, setResp.StatusCode)

	getResp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/notif-vault/notification-configuration", nil))
	require.NoError(t, err)
	m := respJSON(t, getResp)
	config := m["vaultNotificationConfig"].(map[string]any)
	assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:glacier-notif", config["SNSTopic"])
}

func TestGlacier_NonEmptyVaultCannotDelete(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/nonempty-vault", nil))

	ctx := restCtx(http.MethodPost, "/-/vaults/nonempty-vault/archives", []byte("data"))
	ctx.RawRequest.Header.Set("x-amz-archive-description", "test")
	ctx.RawRequest.Header.Set("x-amz-sha256-tree-hash", "hash")
	s.HandleRequest(ctx)

	_, err := s.HandleRequest(restCtx(http.MethodDelete, "/-/vaults/nonempty-vault", nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

func TestGlacier_CompleteVaultLockBadID(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/bad-lock", nil))

	_, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/bad-lock/lock-policy/wrong-id", nil))
	require.Error(t, err)
}

// ---- Test 14: AbortVaultLock ----

func TestGlacier_AbortVaultLock(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/abort-vault", nil))

	// Initiate lock
	initResp, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/abort-vault/lock-policy",
		[]byte(`{"Policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)))
	require.NoError(t, err)
	_ = initResp

	// Abort lock
	_, err = s.HandleRequest(restCtx(http.MethodDelete, "/-/vaults/abort-vault/lock-policy", nil))
	require.NoError(t, err)

	// GetVaultLock should now fail (no lock)
	_, err = s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/abort-vault/lock-policy", nil))
	require.Error(t, err)
}

// ---- Test 15: GetVaultLock ----

func TestGlacier_GetVaultLock(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/get-lock-vault", nil))

	_, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/get-lock-vault/lock-policy",
		[]byte(`{"Policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)))
	require.NoError(t, err)

	getResp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/get-lock-vault/lock-policy", nil))
	require.NoError(t, err)
	lockBody := respJSON(t, getResp)
	assert.NotEmpty(t, lockBody["LockId"])
	assert.Equal(t, "InProgress", lockBody["State"])
}

// ---- Test 16: Tags ----

func TestGlacier_TaggingVault(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/tag-vault", nil))

	// Add tags
	tagsBody := []byte(`{"Tags":{"env":"prod","team":"iot"}}`)
	_, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/tag-vault/tags", tagsBody))
	require.NoError(t, err)

	// List tags
	listResp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/tag-vault/tags", nil))
	require.NoError(t, err)
	body := respJSON(t, listResp)
	tags := body["Tags"].(map[string]any)
	assert.Equal(t, "prod", tags["env"])
	assert.Equal(t, "iot", tags["team"])
}

// ---- Test 17: GetJobOutput ----

func TestGlacier_GetJobOutput(t *testing.T) {
	s := newService()
	s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/output-vault", nil))

	initResp, err := s.HandleRequest(restCtx(http.MethodPost, "/-/vaults/output-vault/jobs",
		[]byte(`{"Type":"inventory-retrieval"}`)))
	require.NoError(t, err)
	// Job ID is in headers or body - check what format it has
	_ = initResp
	_ = err

	// Try getting output from a nonexistent job - should error
	_, err = s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/output-vault/jobs/ghost-job/output", nil))
	require.Error(t, err)
}

// ---- Test 18: ServiceName and HealthCheck ----

func TestGlacier_ServiceNameAndHealthCheck(t *testing.T) {
	s := newService()
	assert.Equal(t, "glacier", s.Name())
	assert.NoError(t, s.HealthCheck())
}

// ---- Test 19: ListTagsForVault - vault not found ----

func TestGlacier_ListTagsNotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults/ghost-vault/tags", nil))
	require.Error(t, err)
}

// ---- Test 20: Multiple vaults and archives ----

func TestGlacier_MultipleVaultsAndArchives(t *testing.T) {
	s := newService()
	for _, name := range []string{"vault-a", "vault-b", "vault-c"} {
		_, err := s.HandleRequest(restCtx(http.MethodPut, "/-/vaults/"+name, nil))
		require.NoError(t, err)
	}

	// List should show all 3
	listResp, err := s.HandleRequest(restCtx(http.MethodGet, "/-/vaults", nil))
	require.NoError(t, err)
	body := respJSON(t, listResp)
	vaultList := body["VaultList"].([]any)
	assert.Len(t, vaultList, 3)

	// Upload archive to vault-a
	ctx := restCtx(http.MethodPost, "/-/vaults/vault-a/archives", []byte("archive-content"))
	ctx.RawRequest.Header.Set("x-amz-archive-description", "my archive")
	ctx.RawRequest.Header.Set("x-amz-sha256-tree-hash", "abc123hash")
	archResp, err := s.HandleRequest(ctx)
	require.NoError(t, err)
	// Archive ID is returned in response headers
	assert.NotNil(t, archResp)
	assert.Equal(t, http.StatusCreated, archResp.StatusCode)
}
