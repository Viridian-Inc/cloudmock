package backup

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// BackupService is the cloudmock implementation of the AWS Backup API.
type BackupService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new BackupService for the given AWS account ID and region.
func New(accountID, region string) *BackupService {
	return &BackupService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *BackupService) Name() string { return "backup" }

// Actions returns the list of Backup API actions supported by this service.
func (s *BackupService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateBackupPlan", Method: http.MethodPost, IAMAction: "backup:CreateBackupPlan"},
		{Name: "GetBackupPlan", Method: http.MethodPost, IAMAction: "backup:GetBackupPlan"},
		{Name: "ListBackupPlans", Method: http.MethodPost, IAMAction: "backup:ListBackupPlans"},
		{Name: "DeleteBackupPlan", Method: http.MethodPost, IAMAction: "backup:DeleteBackupPlan"},
		{Name: "CreateBackupVault", Method: http.MethodPost, IAMAction: "backup:CreateBackupVault"},
		{Name: "DescribeBackupVault", Method: http.MethodPost, IAMAction: "backup:DescribeBackupVault"},
		{Name: "ListBackupVaults", Method: http.MethodPost, IAMAction: "backup:ListBackupVaults"},
		{Name: "DeleteBackupVault", Method: http.MethodPost, IAMAction: "backup:DeleteBackupVault"},
		{Name: "StartBackupJob", Method: http.MethodPost, IAMAction: "backup:StartBackupJob"},
		{Name: "DescribeBackupJob", Method: http.MethodPost, IAMAction: "backup:DescribeBackupJob"},
		{Name: "ListBackupJobs", Method: http.MethodPost, IAMAction: "backup:ListBackupJobs"},
		{Name: "ListRecoveryPointsByBackupVault", Method: http.MethodPost, IAMAction: "backup:ListRecoveryPointsByBackupVault"},
		{Name: "DescribeRecoveryPoint", Method: http.MethodPost, IAMAction: "backup:DescribeRecoveryPoint"},
		{Name: "CreateBackupSelection", Method: http.MethodPost, IAMAction: "backup:CreateBackupSelection"},
		{Name: "GetBackupSelection", Method: http.MethodPost, IAMAction: "backup:GetBackupSelection"},
		{Name: "ListBackupSelections", Method: http.MethodPost, IAMAction: "backup:ListBackupSelections"},
		{Name: "DeleteBackupSelection", Method: http.MethodPost, IAMAction: "backup:DeleteBackupSelection"},
		{Name: "PutBackupVaultLockConfiguration", Method: http.MethodPost, IAMAction: "backup:PutBackupVaultLockConfiguration"},
	}
}

// HealthCheck always returns nil.
func (s *BackupService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Backup request to the appropriate handler.
func (s *BackupService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	switch ctx.Action {
	case "CreateBackupPlan":
		return handleCreateBackupPlan(params, s.store)
	case "GetBackupPlan":
		return handleGetBackupPlan(params, s.store)
	case "ListBackupPlans":
		return handleListBackupPlans(s.store)
	case "DeleteBackupPlan":
		return handleDeleteBackupPlan(params, s.store)
	case "CreateBackupVault":
		return handleCreateBackupVault(params, s.store)
	case "DescribeBackupVault":
		return handleDescribeBackupVault(params, s.store)
	case "ListBackupVaults":
		return handleListBackupVaults(s.store)
	case "DeleteBackupVault":
		return handleDeleteBackupVault(params, s.store)
	case "StartBackupJob":
		return handleStartBackupJob(params, s.store)
	case "DescribeBackupJob":
		return handleDescribeBackupJob(params, s.store)
	case "ListBackupJobs":
		return handleListBackupJobs(s.store)
	case "ListRecoveryPointsByBackupVault":
		return handleListRecoveryPoints(params, s.store)
	case "DescribeRecoveryPoint":
		return handleDescribeRecoveryPoint(params, s.store)
	case "CreateBackupSelection":
		return handleCreateBackupSelection(params, s.store)
	case "GetBackupSelection":
		return handleGetBackupSelection(params, s.store)
	case "ListBackupSelections":
		return handleListBackupSelections(params, s.store)
	case "DeleteBackupSelection":
		return handleDeleteBackupSelection(params, s.store)
	case "PutBackupVaultLockConfiguration":
		return handlePutBackupVaultLockConfiguration(params, s.store)
	default:
		return jsonErr(service.NewAWSError("InvalidAction",
			"The action "+ctx.Action+" is not valid for this web service.",
			http.StatusBadRequest))
	}
}
