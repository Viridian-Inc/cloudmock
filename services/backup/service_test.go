package backup_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	svc "github.com/Viridian-Inc/cloudmock/services/backup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *svc.BackupService { return svc.New("123456789012", "us-east-1") }
func jsonCtx(action string, body map[string]any) *service.RequestContext {
	bodyBytes, _ := json.Marshal(body)
	return &service.RequestContext{Action: action, Region: "us-east-1", AccountID: "123456789012", Body: bodyBytes,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"}}
}
func respJSON(t *testing.T, resp *service.Response) map[string]any {
	t.Helper(); data, _ := json.Marshal(resp.Body); var m map[string]any; require.NoError(t, json.Unmarshal(data, &m)); return m
}

func TestBackup_CreateAndGetPlan(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateBackupPlan", map[string]any{
		"BackupPlan": map[string]any{"BackupPlanName": "daily-plan", "Rules": []map[string]any{
			{"RuleName": "daily", "TargetBackupVaultName": "Default", "ScheduleExpression": "cron(0 12 * * ? *)"},
		}},
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	planID := m["BackupPlanId"].(string)
	assert.NotEmpty(t, planID)

	getResp, _ := s.HandleRequest(jsonCtx("GetBackupPlan", map[string]any{"BackupPlanId": planID}))
	gm := respJSON(t, getResp)
	assert.Equal(t, "daily-plan", gm["BackupPlan"].(map[string]any)["BackupPlanName"])
}

func TestBackup_ListAndDeletePlan(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateBackupPlan", map[string]any{
		"BackupPlan": map[string]any{"BackupPlanName": "p1"},
	}))
	planID := respJSON(t, cr)["BackupPlanId"].(string)

	listResp, _ := s.HandleRequest(jsonCtx("ListBackupPlans", nil))
	assert.Len(t, respJSON(t, listResp)["BackupPlansList"].([]any), 1)

	delResp, err := s.HandleRequest(jsonCtx("DeleteBackupPlan", map[string]any{"BackupPlanId": planID}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestBackup_VaultCRUD(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("CreateBackupVault", map[string]any{"BackupVaultName": "my-vault"}))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeBackupVault", map[string]any{"BackupVaultName": "my-vault"}))
	assert.Equal(t, "my-vault", respJSON(t, descResp)["BackupVaultName"])

	listResp, _ := s.HandleRequest(jsonCtx("ListBackupVaults", nil))
	// Default vault + my-vault
	assert.GreaterOrEqual(t, len(respJSON(t, listResp)["BackupVaultList"].([]any)), 2)

	s.HandleRequest(jsonCtx("DeleteBackupVault", map[string]any{"BackupVaultName": "my-vault"}))
	_, err = s.HandleRequest(jsonCtx("DescribeBackupVault", map[string]any{"BackupVaultName": "my-vault"}))
	require.Error(t, err)
}

func TestBackup_StartAndDescribeJob(t *testing.T) {
	s := newService()
	resp, err := s.HandleRequest(jsonCtx("StartBackupJob", map[string]any{
		"BackupVaultName": "Default", "ResourceArn": "arn:aws:ec2:us-east-1:123456789012:instance/i-123",
		"ResourceType": "EC2",
	}))
	require.NoError(t, err)
	m := respJSON(t, resp)
	jobID := m["BackupJobId"].(string)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeBackupJob", map[string]any{"BackupJobId": jobID}))
	dm := respJSON(t, descResp)
	// With instant lifecycle transitions, job completes synchronously
	assert.Equal(t, "COMPLETED", dm["State"])
}

func TestBackup_ListJobs(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("StartBackupJob", map[string]any{
		"BackupVaultName": "Default", "ResourceArn": "arn:r1",
	}))
	resp, _ := s.HandleRequest(jsonCtx("ListBackupJobs", nil))
	assert.Len(t, respJSON(t, resp)["BackupJobs"].([]any), 1)
}

func TestBackup_SelectionCRUD(t *testing.T) {
	s := newService()
	cr, _ := s.HandleRequest(jsonCtx("CreateBackupPlan", map[string]any{
		"BackupPlan": map[string]any{"BackupPlanName": "sel-plan"},
	}))
	planID := respJSON(t, cr)["BackupPlanId"].(string)

	selResp, err := s.HandleRequest(jsonCtx("CreateBackupSelection", map[string]any{
		"BackupPlanId": planID, "BackupSelection": map[string]any{
			"SelectionName": "my-sel", "IamRoleArn": "arn:aws:iam::123456789012:role/backup",
			"Resources": []string{"arn:aws:ec2:*:*:instance/*"},
		},
	}))
	require.NoError(t, err)
	selID := respJSON(t, selResp)["SelectionId"].(string)

	getResp, _ := s.HandleRequest(jsonCtx("GetBackupSelection", map[string]any{"BackupPlanId": planID, "SelectionId": selID}))
	assert.Equal(t, "my-sel", respJSON(t, getResp)["BackupSelection"].(map[string]any)["SelectionName"])

	listResp, _ := s.HandleRequest(jsonCtx("ListBackupSelections", map[string]any{"BackupPlanId": planID}))
	assert.Len(t, respJSON(t, listResp)["BackupSelectionsList"].([]any), 1)

	s.HandleRequest(jsonCtx("DeleteBackupSelection", map[string]any{"BackupPlanId": planID, "SelectionId": selID}))
	_, err = s.HandleRequest(jsonCtx("GetBackupSelection", map[string]any{"BackupPlanId": planID, "SelectionId": selID}))
	require.Error(t, err)
}

func TestBackup_NotFound(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("GetBackupPlan", map[string]any{"BackupPlanId": "nonexistent"}))
	require.Error(t, err)
	_, err = s.HandleRequest(jsonCtx("DescribeBackupJob", map[string]any{"BackupJobId": "nonexistent"}))
	require.Error(t, err)
}

func TestBackup_InvalidAction(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("Bogus", nil))
	require.Error(t, err)
}

func TestBackup_VaultLockPreventsDelete(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateBackupVault", map[string]any{"BackupVaultName": "locked-vault"}))

	// Lock the vault
	_, err := s.HandleRequest(jsonCtx("PutBackupVaultLockConfiguration", map[string]any{
		"BackupVaultName":  "locked-vault",
		"MinRetentionDays": float64(7),
		"MaxRetentionDays": float64(365),
	}))
	require.NoError(t, err)

	// Try to delete - should fail
	_, err = s.HandleRequest(jsonCtx("DeleteBackupVault", map[string]any{"BackupVaultName": "locked-vault"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestBackup_VaultLockValidation(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateBackupVault", map[string]any{"BackupVaultName": "v1"}))

	// MinRetentionDays < 1
	_, err := s.HandleRequest(jsonCtx("PutBackupVaultLockConfiguration", map[string]any{
		"BackupVaultName":  "v1",
		"MinRetentionDays": float64(0),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MinRetentionDays")

	// MaxRetention < MinRetention
	_, err = s.HandleRequest(jsonCtx("PutBackupVaultLockConfiguration", map[string]any{
		"BackupVaultName":  "v1",
		"MinRetentionDays": float64(30),
		"MaxRetentionDays": float64(7),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MaxRetentionDays")
}

func TestBackup_StartJobRequiresVault(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("StartBackupJob", map[string]any{
		"BackupVaultName": "nonexistent-vault",
		"ResourceArn":     "arn:aws:ec2:us-east-1:123456789012:instance/i-999",
	}))
	require.Error(t, err)
}

func TestBackup_RecoveryPointLifecycle(t *testing.T) {
	s := newService()
	// Start a backup job on the Default vault
	resp, err := s.HandleRequest(jsonCtx("StartBackupJob", map[string]any{
		"BackupVaultName": "Default",
		"ResourceArn":     "arn:aws:ec2:us-east-1:123456789012:instance/i-abc",
		"ResourceType":    "EC2",
	}))
	require.NoError(t, err)
	jobID := respJSON(t, resp)["BackupJobId"].(string)

	// With default lifecycle (instant transitions via goroutines), wait briefly
	time.Sleep(50 * time.Millisecond)

	descResp, _ := s.HandleRequest(jsonCtx("DescribeBackupJob", map[string]any{"BackupJobId": jobID}))
	dm := respJSON(t, descResp)
	assert.Equal(t, "COMPLETED", dm["State"])
	assert.NotEmpty(t, dm["RecoveryPointArn"])

	// List recovery points on Default vault
	rpResp, _ := s.HandleRequest(jsonCtx("ListRecoveryPointsByBackupVault", map[string]any{"BackupVaultName": "Default"}))
	rps := respJSON(t, rpResp)["RecoveryPoints"].([]any)
	assert.GreaterOrEqual(t, len(rps), 1)
}

func TestBackup_DuplicateVaultCreate(t *testing.T) {
	s := newService()
	_, err := s.HandleRequest(jsonCtx("CreateBackupVault", map[string]any{"BackupVaultName": "dup-vault"}))
	require.NoError(t, err)

	// Second create of the same vault should fail
	_, err = s.HandleRequest(jsonCtx("CreateBackupVault", map[string]any{"BackupVaultName": "dup-vault"}))
	require.Error(t, err)
}

func TestBackup_DuplicatePlanCreate(t *testing.T) {
	s := newService()
	body := map[string]any{
		"BackupPlan": map[string]any{"BackupPlanName": "dup-plan"},
	}
	cr, err := s.HandleRequest(jsonCtx("CreateBackupPlan", body))
	require.NoError(t, err)
	_ = respJSON(t, cr)["BackupPlanId"]

	// Second plan with the same name is allowed (AWS allows it; each gets a unique ID).
	// So instead test that describing a plan with a bogus ID returns an error.
	_, err = s.HandleRequest(jsonCtx("GetBackupPlan", map[string]any{"BackupPlanId": "bogus-plan-id"}))
	require.Error(t, err)
}

func TestBackup_VaultNotEmptyCannotDelete(t *testing.T) {
	s := newService()
	s.HandleRequest(jsonCtx("CreateBackupVault", map[string]any{"BackupVaultName": "notempty-vault"}))
	// Start a job to create a recovery point (async lifecycle)
	s.HandleRequest(jsonCtx("StartBackupJob", map[string]any{
		"BackupVaultName": "notempty-vault",
		"ResourceArn":     "arn:aws:ec2:us-east-1:123456789012:instance/i-xyz",
	}))
	// Wait for lifecycle transition callbacks to complete
	time.Sleep(50 * time.Millisecond)

	// Try to delete - should fail because it has recovery points
	_, err := s.HandleRequest(jsonCtx("DeleteBackupVault", map[string]any{"BackupVaultName": "notempty-vault"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovery points")
}
