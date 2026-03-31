package backup

import (
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

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

func strSlice(params map[string]any, key string) []string {
	if v, ok := params[key].([]any); ok {
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func handleCreateBackupPlan(params map[string]any, store *Store) (*service.Response, error) {
	planInput, _ := params["BackupPlan"].(map[string]any)
	name := str(planInput, "BackupPlanName")
	if name == "" {
		return jsonErr(service.ErrValidation("BackupPlan.BackupPlanName is required"))
	}

	var rules []BackupRule
	if rawRules, ok := planInput["Rules"].([]any); ok {
		for _, rr := range rawRules {
			rm, _ := rr.(map[string]any)
			rule := BackupRule{
				RuleName:              str(rm, "RuleName"),
				TargetBackupVaultName: str(rm, "TargetBackupVaultName"),
				ScheduleExpression:    str(rm, "ScheduleExpression"),
			}
			if v, ok := rm["StartWindowMinutes"].(float64); ok {
				rule.StartWindowMinutes = int64(v)
			}
			if v, ok := rm["CompletionWindowMinutes"].(float64); ok {
				rule.CompletionWindowMinutes = int64(v)
			}
			rules = append(rules, rule)
		}
	}

	plan, _ := store.CreateBackupPlan(name, rules)
	return jsonOK(map[string]any{
		"BackupPlanId":  plan.BackupPlanID,
		"BackupPlanArn": plan.BackupPlanArn,
		"VersionId":     plan.VersionID,
		"CreationDate":  plan.CreatedAt.Format(time.RFC3339),
	})
}

func handleGetBackupPlan(params map[string]any, store *Store) (*service.Response, error) {
	planID := str(params, "BackupPlanId")
	if planID == "" {
		return jsonErr(service.ErrValidation("BackupPlanId is required"))
	}
	plan, ok := store.GetBackupPlan(planID)
	if !ok {
		return jsonErr(service.ErrNotFound("BackupPlan", planID))
	}

	rules := make([]map[string]any, 0, len(plan.Rules))
	for _, r := range plan.Rules {
		rules = append(rules, map[string]any{
			"RuleName":              r.RuleName,
			"TargetBackupVaultName": r.TargetBackupVaultName,
			"ScheduleExpression":    r.ScheduleExpression,
		})
	}

	return jsonOK(map[string]any{
		"BackupPlan": map[string]any{
			"BackupPlanName": plan.BackupPlanName,
			"Rules":          rules,
		},
		"BackupPlanId":  plan.BackupPlanID,
		"BackupPlanArn": plan.BackupPlanArn,
		"VersionId":     plan.VersionID,
		"CreationDate":  plan.CreatedAt.Format(time.RFC3339),
	})
}

func handleListBackupPlans(store *Store) (*service.Response, error) {
	plans := store.ListBackupPlans()
	out := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		out = append(out, map[string]any{
			"BackupPlanId":   p.BackupPlanID,
			"BackupPlanArn":  p.BackupPlanArn,
			"BackupPlanName": p.BackupPlanName,
			"VersionId":      p.VersionID,
			"CreationDate":   p.CreatedAt.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"BackupPlansList": out})
}

func handleDeleteBackupPlan(params map[string]any, store *Store) (*service.Response, error) {
	planID := str(params, "BackupPlanId")
	if planID == "" {
		return jsonErr(service.ErrValidation("BackupPlanId is required"))
	}
	if !store.DeleteBackupPlan(planID) {
		return jsonErr(service.ErrNotFound("BackupPlan", planID))
	}
	return jsonOK(map[string]any{"BackupPlanId": planID})
}

func handleCreateBackupVault(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "BackupVaultName")
	if name == "" {
		return jsonErr(service.ErrValidation("BackupVaultName is required"))
	}
	vault, err := store.CreateBackupVault(name, str(params, "EncryptionKeyArn"))
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("BackupVault", name))
	}
	return jsonOK(map[string]any{
		"BackupVaultName": vault.BackupVaultName,
		"BackupVaultArn":  vault.BackupVaultArn,
		"CreationDate":    vault.CreationDate.Format(time.RFC3339),
	})
}

func handleDescribeBackupVault(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "BackupVaultName")
	if name == "" {
		return jsonErr(service.ErrValidation("BackupVaultName is required"))
	}
	vault, ok := store.GetBackupVault(name)
	if !ok {
		return jsonErr(service.ErrNotFound("BackupVault", name))
	}
	return jsonOK(map[string]any{
		"BackupVaultName":        vault.BackupVaultName,
		"BackupVaultArn":         vault.BackupVaultArn,
		"EncryptionKeyArn":       vault.EncryptionKeyArn,
		"CreationDate":           vault.CreationDate.Format(time.RFC3339),
		"NumberOfRecoveryPoints": vault.NumberOfRecoveryPoints,
	})
}

func handleListBackupVaults(store *Store) (*service.Response, error) {
	vaults := store.ListBackupVaults()
	out := make([]map[string]any, 0, len(vaults))
	for _, v := range vaults {
		out = append(out, map[string]any{
			"BackupVaultName":        v.BackupVaultName,
			"BackupVaultArn":         v.BackupVaultArn,
			"CreationDate":           v.CreationDate.Format(time.RFC3339),
			"NumberOfRecoveryPoints": v.NumberOfRecoveryPoints,
		})
	}
	return jsonOK(map[string]any{"BackupVaultList": out})
}

func handleDeleteBackupVault(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "BackupVaultName")
	if name == "" {
		return jsonErr(service.ErrValidation("BackupVaultName is required"))
	}
	if !store.DeleteBackupVault(name) {
		return jsonErr(service.ErrNotFound("BackupVault", name))
	}
	return jsonOK(map[string]any{})
}

func handleStartBackupJob(params map[string]any, store *Store) (*service.Response, error) {
	vaultName := str(params, "BackupVaultName")
	resourceArn := str(params, "ResourceArn")
	if vaultName == "" || resourceArn == "" {
		return jsonErr(service.ErrValidation("BackupVaultName and ResourceArn are required"))
	}
	job, err := store.StartBackupJob(vaultName, resourceArn,
		str(params, "ResourceType"), str(params, "IamRoleArn"))
	if err != nil {
		return jsonErr(service.ErrNotFound("BackupVault", vaultName))
	}
	return jsonOK(map[string]any{
		"BackupJobId":  job.BackupJobID,
		"CreationDate": job.CreationDate.Format(time.RFC3339),
	})
}

func handleDescribeBackupJob(params map[string]any, store *Store) (*service.Response, error) {
	jobID := str(params, "BackupJobId")
	if jobID == "" {
		return jsonErr(service.ErrValidation("BackupJobId is required"))
	}
	job, ok := store.GetBackupJob(jobID)
	if !ok {
		return jsonErr(service.ErrNotFound("BackupJob", jobID))
	}
	resp := map[string]any{
		"BackupJobId":       job.BackupJobID,
		"BackupVaultName":   job.BackupVaultName,
		"BackupVaultArn":    job.BackupVaultArn,
		"ResourceArn":       job.ResourceArn,
		"ResourceType":      job.ResourceType,
		"State":             job.State,
		"StatusMessage":     job.StatusMessage,
		"CreationDate":      job.CreationDate.Format(time.RFC3339),
		"BackupSizeInBytes": job.BackupSizeInBytes,
	}
	if job.CompletionDate != nil {
		resp["CompletionDate"] = job.CompletionDate.Format(time.RFC3339)
	}
	if job.RecoveryPointArn != "" {
		resp["RecoveryPointArn"] = job.RecoveryPointArn
	}
	return jsonOK(resp)
}

func handleListBackupJobs(store *Store) (*service.Response, error) {
	jobs := store.ListBackupJobs()
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		entry := map[string]any{
			"BackupJobId":     j.BackupJobID,
			"BackupVaultName": j.BackupVaultName,
			"ResourceArn":     j.ResourceArn,
			"State":           j.State,
			"CreationDate":    j.CreationDate.Format(time.RFC3339),
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"BackupJobs": out})
}

func handleListRecoveryPoints(params map[string]any, store *Store) (*service.Response, error) {
	vaultName := str(params, "BackupVaultName")
	rps := store.ListRecoveryPoints(vaultName)
	out := make([]map[string]any, 0, len(rps))
	for _, rp := range rps {
		out = append(out, map[string]any{
			"RecoveryPointArn":  rp.RecoveryPointArn,
			"BackupVaultName":   rp.BackupVaultName,
			"ResourceArn":       rp.ResourceArn,
			"ResourceType":      rp.ResourceType,
			"CreationDate":      rp.CreationDate.Format(time.RFC3339),
			"Status":            rp.Status,
			"BackupSizeInBytes": rp.BackupSizeInBytes,
			"IsEncrypted":       rp.IsEncrypted,
		})
	}
	return jsonOK(map[string]any{"RecoveryPoints": out})
}

func handleDescribeRecoveryPoint(params map[string]any, store *Store) (*service.Response, error) {
	vaultName := str(params, "BackupVaultName")
	rpArn := str(params, "RecoveryPointArn")
	if vaultName == "" || rpArn == "" {
		return jsonErr(service.ErrValidation("BackupVaultName and RecoveryPointArn are required"))
	}
	rp, ok := store.GetRecoveryPoint(vaultName, rpArn)
	if !ok {
		return jsonErr(service.ErrNotFound("RecoveryPoint", rpArn))
	}
	return jsonOK(map[string]any{
		"RecoveryPointArn":  rp.RecoveryPointArn,
		"BackupVaultName":   rp.BackupVaultName,
		"BackupVaultArn":    rp.BackupVaultArn,
		"ResourceArn":       rp.ResourceArn,
		"ResourceType":      rp.ResourceType,
		"CreationDate":      rp.CreationDate.Format(time.RFC3339),
		"Status":            rp.Status,
		"BackupSizeInBytes": rp.BackupSizeInBytes,
		"IsEncrypted":       rp.IsEncrypted,
	})
}

func handleCreateBackupSelection(params map[string]any, store *Store) (*service.Response, error) {
	planID := str(params, "BackupPlanId")
	selInput, _ := params["BackupSelection"].(map[string]any)
	name := str(selInput, "SelectionName")
	iamRole := str(selInput, "IamRoleArn")
	resources := strSlice(selInput, "Resources")

	if planID == "" || name == "" {
		return jsonErr(service.ErrValidation("BackupPlanId and BackupSelection.SelectionName are required"))
	}

	sel, err := store.CreateBackupSelection(planID, name, iamRole, resources)
	if err != nil {
		return jsonErr(service.ErrNotFound("BackupPlan", planID))
	}
	return jsonOK(map[string]any{
		"SelectionId":  sel.SelectionID,
		"BackupPlanId": planID,
		"CreationDate": sel.CreationDate.Format(time.RFC3339),
	})
}

func handleGetBackupSelection(params map[string]any, store *Store) (*service.Response, error) {
	planID := str(params, "BackupPlanId")
	selID := str(params, "SelectionId")
	if planID == "" || selID == "" {
		return jsonErr(service.ErrValidation("BackupPlanId and SelectionId are required"))
	}
	sel, ok := store.GetBackupSelection(planID, selID)
	if !ok {
		return jsonErr(service.ErrNotFound("BackupSelection", selID))
	}
	return jsonOK(map[string]any{
		"BackupSelection": map[string]any{
			"SelectionName": sel.SelectionName,
			"IamRoleArn":    sel.IamRoleArn,
			"Resources":     sel.Resources,
		},
		"SelectionId":  sel.SelectionID,
		"BackupPlanId": planID,
		"CreationDate": sel.CreationDate.Format(time.RFC3339),
	})
}

func handleListBackupSelections(params map[string]any, store *Store) (*service.Response, error) {
	planID := str(params, "BackupPlanId")
	if planID == "" {
		return jsonErr(service.ErrValidation("BackupPlanId is required"))
	}
	sels := store.ListBackupSelections(planID)
	out := make([]map[string]any, 0, len(sels))
	for _, sel := range sels {
		out = append(out, map[string]any{
			"SelectionId":   sel.SelectionID,
			"SelectionName": sel.SelectionName,
			"BackupPlanId":  sel.BackupPlanID,
			"IamRoleArn":    sel.IamRoleArn,
			"CreationDate":  sel.CreationDate.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"BackupSelectionsList": out})
}

func handleDeleteBackupSelection(params map[string]any, store *Store) (*service.Response, error) {
	planID := str(params, "BackupPlanId")
	selID := str(params, "SelectionId")
	if planID == "" || selID == "" {
		return jsonErr(service.ErrValidation("BackupPlanId and SelectionId are required"))
	}
	if !store.DeleteBackupSelection(planID, selID) {
		return jsonErr(service.ErrNotFound("BackupSelection", selID))
	}
	return jsonOK(map[string]any{})
}
