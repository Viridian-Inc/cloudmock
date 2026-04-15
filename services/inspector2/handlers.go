package inspector2

import (
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStrPtr(m map[string]any, key string) *string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getStrMap(m map[string]any, key string) map[string]string {
	mm := getMap(m, key)
	if mm == nil {
		return nil
	}
	out := make(map[string]string, len(mm))
	for k, v := range mm {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ── Map conversion helpers ──────────────────────────────────────────────────

func filterToMap(f *StoredFilter) map[string]any {
	out := map[string]any{
		"arn":       f.Arn,
		"name":      f.Name,
		"action":    f.Action,
		"ownerId":   f.OwnerId,
		"createdAt": rfc3339(f.CreatedAt),
		"updatedAt": rfc3339(f.UpdatedAt),
	}
	if f.Description != "" {
		out["description"] = f.Description
	}
	if f.Reason != "" {
		out["reason"] = f.Reason
	}
	if f.Criteria != nil {
		out["criteria"] = f.Criteria
	}
	if len(f.Tags) > 0 {
		out["tags"] = f.Tags
	}
	return out
}

func cisScanConfigToMap(c *StoredCisScanConfiguration) map[string]any {
	out := map[string]any{
		"scanConfigurationArn": c.ScanConfigurationArn,
		"scanName":             c.ScanName,
		"securityLevel":        c.SecurityLevel,
		"ownerId":              c.OwnerId,
	}
	if c.Schedule != nil {
		out["schedule"] = c.Schedule
	}
	if c.Targets != nil {
		out["targets"] = c.Targets
	}
	if len(c.Tags) > 0 {
		out["tags"] = c.Tags
	}
	return out
}

func cisScanToMap(s *StoredCisScan) map[string]any {
	out := map[string]any{
		"scanArn":              s.ScanArn,
		"scanConfigurationArn": s.ScanConfigurationArn,
		"failedChecks":         s.FailedChecks,
		"totalChecks":          s.TotalChecks,
	}
	if s.ScanName != "" {
		out["scanName"] = s.ScanName
	}
	if s.Status != "" {
		out["status"] = s.Status
	}
	if s.SecurityLevel != "" {
		out["securityLevel"] = s.SecurityLevel
	}
	if s.ScheduledBy != "" {
		out["scheduledBy"] = s.ScheduledBy
	}
	if s.Targets != nil {
		out["targets"] = s.Targets
	}
	if !s.ScanDate.IsZero() {
		out["scanDate"] = rfc3339(s.ScanDate)
	}
	return out
}

func codeSecurityIntegrationToMap(i *StoredCodeSecurityIntegration) map[string]any {
	out := map[string]any{
		"integrationArn": i.IntegrationArn,
		"name":           i.Name,
		"type":           i.Type,
		"status":         i.Status,
		"createdOn":      rfc3339(i.CreatedOn),
		"lastUpdateOn":   rfc3339(i.LastUpdateOn),
	}
	if i.StatusReason != "" {
		out["statusReason"] = i.StatusReason
	}
	if len(i.Tags) > 0 {
		out["tags"] = i.Tags
	}
	return out
}

func codeSecurityScanConfigToMap(c *StoredCodeSecurityScanConfiguration) map[string]any {
	out := map[string]any{
		"scanConfigurationArn": c.ScanConfigurationArn,
		"name":                 c.Name,
		"ownerAccountId":       c.OwnerAccountId,
	}
	if c.Level != "" {
		out["level"] = c.Level
	}
	if c.Configuration != nil {
		out["configuration"] = c.Configuration
	}
	if c.ScopeSettings != nil {
		out["scopeSettings"] = c.ScopeSettings
	}
	if len(c.Tags) > 0 {
		out["tags"] = c.Tags
	}
	return out
}

func memberToMap(m *StoredMember) map[string]any {
	return map[string]any{
		"accountId":               m.AccountId,
		"delegatedAdminAccountId": m.DelegatedAdminAccountId,
		"relationshipStatus":      m.RelationshipStatus,
		"updatedAt":               rfc3339(m.UpdatedAt),
	}
}

func findingsReportToMap(r *StoredFindingsReport) map[string]any {
	out := map[string]any{
		"reportId": r.ReportId,
		"status":   r.Status,
	}
	if r.Destination != nil {
		out["destination"] = r.Destination
	}
	if r.FilterCriteria != nil {
		out["filterCriteria"] = r.FilterCriteria
	}
	if r.ErrorCode != "" {
		out["errorCode"] = r.ErrorCode
	}
	if r.ErrorMessage != "" {
		out["errorMessage"] = r.ErrorMessage
	}
	return out
}

func sbomExportToMap(e *StoredSbomExport) map[string]any {
	out := map[string]any{
		"reportId": e.ReportId,
		"status":   e.Status,
	}
	if e.Format != "" {
		out["format"] = e.Format
	}
	if e.Destination != nil {
		out["s3Destination"] = e.Destination
	}
	if e.FilterCriteria != nil {
		out["filterCriteria"] = e.FilterCriteria
	}
	if e.ErrorCode != "" {
		out["errorCode"] = e.ErrorCode
	}
	if e.ErrorMessage != "" {
		out["errorMessage"] = e.ErrorMessage
	}
	return out
}

func accountStatusToMap(st *StoredAccountStatus) map[string]any {
	return map[string]any{
		"accountId": st.AccountId,
		"resourceState": map[string]any{
			"ec2":            map[string]any{"status": st.Ec2},
			"ecr":            map[string]any{"status": st.Ecr},
			"lambda":         map[string]any{"status": st.Lambda},
			"lambdaCode":     map[string]any{"status": st.LambdaCode},
			"codeRepository": map[string]any{"status": st.CodeRepository},
		},
		"state": map[string]any{
			"status": st.Status,
		},
	}
}

func accountToMap(st *StoredAccountStatus) map[string]any {
	return map[string]any{
		"accountId": st.AccountId,
		"resourceStatus": map[string]any{
			"ec2":            st.Ec2,
			"ecr":            st.Ecr,
			"lambda":         st.Lambda,
			"lambdaCode":     st.LambdaCode,
			"codeRepository": st.CodeRepository,
		},
		"status": st.Status,
	}
}

func orgConfigToMap(c *StoredOrganizationConfig) map[string]any {
	return map[string]any{
		"ec2":            c.Ec2,
		"ecr":            c.Ecr,
		"lambda":         c.Lambda,
		"lambdaCode":     c.LambdaCode,
		"codeRepository": c.CodeRepository,
	}
}

// ── Handlers: members ────────────────────────────────────────────────────────

func handleAssociateMember(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountId := getStr(req, "accountId")
	if accountId == "" {
		return jsonErr(service.ErrValidation("accountId is required"))
	}
	m, err := store.AssociateMember(accountId)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"accountId": m.AccountId})
}

func handleDisassociateMember(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountId := getStr(req, "accountId")
	if accountId == "" {
		return jsonErr(service.ErrValidation("accountId is required"))
	}
	if err := store.DisassociateMember(accountId); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"accountId": accountId})
}

func handleGetMember(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountId := getStr(req, "accountId")
	if accountId == "" {
		return jsonErr(service.ErrValidation("accountId is required"))
	}
	m, err := store.GetMember(accountId)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"member": memberToMap(m)})
}

func handleListMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	onlyAssociated := getBool(req, "onlyAssociated")
	list := store.ListMembers(onlyAssociated)
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, memberToMap(m))
	}
	return jsonOK(map[string]any{"members": out})
}

// ── Handlers: code security scan configuration associations ─────────────────

func handleBatchAssociateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	successful := make([]map[string]any, 0)
	failed := make([]map[string]any, 0)
	for _, entry := range getMapList(req, "associateConfigurationRequests") {
		scanArn := getStr(entry, "scanConfigurationArn")
		resource := getMap(entry, "resource")
		projectId := getStr(resource, "projectId")
		if err := store.AssociateCodeSecurityScanConfiguration(scanArn, projectId); err != nil {
			failed = append(failed, map[string]any{
				"resource":             resource,
				"scanConfigurationArn": scanArn,
				"statusCode":           err.Code,
				"statusMessage":        err.Message,
			})
			continue
		}
		successful = append(successful, map[string]any{
			"resource":             resource,
			"scanConfigurationArn": scanArn,
		})
	}
	return jsonOK(map[string]any{
		"successfulAssociations": successful,
		"failedAssociations":     failed,
	})
}

func handleBatchDisassociateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	successful := make([]map[string]any, 0)
	failed := make([]map[string]any, 0)
	for _, entry := range getMapList(req, "disassociateConfigurationRequests") {
		scanArn := getStr(entry, "scanConfigurationArn")
		resource := getMap(entry, "resource")
		projectId := getStr(resource, "projectId")
		if err := store.DisassociateCodeSecurityScanConfiguration(scanArn, projectId); err != nil {
			failed = append(failed, map[string]any{
				"resource":             resource,
				"scanConfigurationArn": scanArn,
				"statusCode":           err.Code,
				"statusMessage":        err.Message,
			})
			continue
		}
		successful = append(successful, map[string]any{
			"resource":             resource,
			"scanConfigurationArn": scanArn,
		})
	}
	return jsonOK(map[string]any{
		"successfulAssociations": successful,
		"failedAssociations":     failed,
	})
}

// ── Handlers: batch get account status / details / etc ──────────────────────

func handleBatchGetAccountStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "accountIds")
	if len(ids) == 0 {
		ids = []string{store.AccountID()}
	}
	accounts := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		st := store.GetAccountStatus(id)
		accounts = append(accounts, accountStatusToMap(st))
	}
	return jsonOK(map[string]any{
		"accounts":       accounts,
		"failedAccounts": []map[string]any{},
	})
}

func handleBatchGetCodeSnippet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	results := make([]map[string]any, 0)
	for _, arn := range getStrList(req, "findingArns") {
		results = append(results, map[string]any{
			"findingArn":  arn,
			"codeSnippet": []map[string]any{},
			"startLine":   0,
			"endLine":     0,
		})
	}
	return jsonOK(map[string]any{
		"codeSnippetResults": results,
		"errors":             []map[string]any{},
	})
}

func handleBatchGetFindingDetails(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	details := make([]map[string]any, 0)
	errors := make([]map[string]any, 0)
	for _, arn := range getStrList(req, "findingArns") {
		if _, err := store.GetFinding(arn); err != nil {
			errors = append(errors, map[string]any{
				"findingArn":   arn,
				"errorCode":    err.Code,
				"errorMessage": err.Message,
			})
			continue
		}
		details = append(details, map[string]any{
			"findingArn": arn,
		})
	}
	return jsonOK(map[string]any{
		"findingDetails": details,
		"errors":         errors,
	})
}

func handleBatchGetFreeTrialInfo(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "accountIds")
	if len(ids) == 0 {
		ids = []string{store.AccountID()}
	}
	accounts := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		accounts = append(accounts, map[string]any{
			"accountId":     id,
			"freeTrialInfo": []map[string]any{},
		})
	}
	return jsonOK(map[string]any{
		"accounts":       accounts,
		"failedAccounts": []map[string]any{},
	})
}

func handleBatchGetMemberEc2DeepInspectionStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "accountIds")
	statuses := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		di := store.GetEc2DeepInspection(id)
		statuses = append(statuses, map[string]any{
			"accountId": id,
			"status":    di.Status,
		})
	}
	return jsonOK(map[string]any{
		"accountIds":       statuses,
		"failedAccountIds": []map[string]any{},
	})
}

func handleBatchUpdateMemberEc2DeepInspectionStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	updated := make([]map[string]any, 0)
	for _, entry := range getMapList(req, "accountIds") {
		id := getStr(entry, "accountId")
		activate := getBool(entry, "activateDeepInspection")
		di := store.UpdateEc2DeepInspection(id, nil, activate)
		updated = append(updated, map[string]any{
			"accountId": di.AccountId,
			"status":    di.Status,
		})
	}
	return jsonOK(map[string]any{
		"accountIds":       updated,
		"failedAccountIds": []map[string]any{},
	})
}

// ── Handlers: findings reports / SBOM exports ───────────────────────────────

func handleCreateFindingsReport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	format := getStr(req, "reportFormat")
	if format == "" {
		return jsonErr(service.ErrValidation("reportFormat is required"))
	}
	dest := getMap(req, "s3Destination")
	if dest == nil {
		return jsonErr(service.ErrValidation("s3Destination is required"))
	}
	report := store.CreateFindingsReport(format, dest, getMap(req, "filterCriteria"))
	return jsonOK(map[string]any{"reportId": report.ReportId})
}

func handleCancelFindingsReport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "reportId")
	if id == "" {
		return jsonErr(service.ErrValidation("reportId is required"))
	}
	if err := store.CancelFindingsReport(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"reportId": id})
}

func handleGetFindingsReportStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "reportId")
	if id == "" {
		return jsonErr(service.ErrValidation("reportId is required"))
	}
	r, err := store.GetFindingsReport(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(findingsReportToMap(r))
}

func handleCreateSbomExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	format := getStr(req, "reportFormat")
	if format == "" {
		return jsonErr(service.ErrValidation("reportFormat is required"))
	}
	dest := getMap(req, "s3Destination")
	if dest == nil {
		return jsonErr(service.ErrValidation("s3Destination is required"))
	}
	exp := store.CreateSbomExport(format, dest, getMap(req, "resourceFilterCriteria"))
	return jsonOK(map[string]any{"reportId": exp.ReportId})
}

func handleCancelSbomExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "reportId")
	if id == "" {
		return jsonErr(service.ErrValidation("reportId is required"))
	}
	if err := store.CancelSbomExport(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"reportId": id})
}

func handleGetSbomExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "reportId")
	if id == "" {
		return jsonErr(service.ErrValidation("reportId is required"))
	}
	e, err := store.GetSbomExport(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(sbomExportToMap(e))
}

// ── Handlers: CIS scan configurations ───────────────────────────────────────

func handleCreateCisScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	cfg, err := store.CreateCisScanConfiguration(
		getStr(req, "scanName"),
		getStr(req, "securityLevel"),
		getMap(req, "schedule"),
		getMap(req, "targets"),
		getStrMap(req, "tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanConfigurationArn": cfg.ScanConfigurationArn})
}

func handleDeleteCisScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "scanConfigurationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("scanConfigurationArn is required"))
	}
	if err := store.DeleteCisScanConfiguration(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanConfigurationArn": arn})
}

func handleUpdateCisScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "scanConfigurationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("scanConfigurationArn is required"))
	}
	cfg, err := store.UpdateCisScanConfiguration(
		arn,
		getStrPtr(req, "scanName"),
		getStrPtr(req, "securityLevel"),
		getMap(req, "schedule"),
		getMap(req, "targets"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanConfigurationArn": cfg.ScanConfigurationArn})
}

func handleListCisScanConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListCisScanConfigurations()
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, cisScanConfigToMap(c))
	}
	return jsonOK(map[string]any{"scanConfigurations": out})
}

func handleListCisScans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListCisScans()
	out := make([]map[string]any, 0, len(list))
	for _, s := range list {
		out = append(out, cisScanToMap(s))
	}
	return jsonOK(map[string]any{"scans": out})
}

func handleListCisScanResultsAggregatedByChecks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"checkAggregations": []map[string]any{}})
}

func handleListCisScanResultsAggregatedByTargetResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"targetResourceAggregations": []map[string]any{}})
}

func handleGetCisScanReport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "scanArn") == "" {
		return jsonErr(service.ErrValidation("scanArn is required"))
	}
	return jsonOK(map[string]any{
		"status": "SUCCEEDED",
		"url":    "",
	})
}

func handleGetCisScanResultDetails(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "scanArn") == "" {
		return jsonErr(service.ErrValidation("scanArn is required"))
	}
	return jsonOK(map[string]any{"scanResultDetails": []map[string]any{}})
}

// ── Handlers: CIS sessions ──────────────────────────────────────────────────

func handleStartCisSession(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	scanJobId := getStr(req, "scanJobId")
	msg := getMap(req, "message")
	sessionToken := getStr(msg, "sessionToken")
	if err := store.StartCisSession(scanJobId, sessionToken); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleStopCisSession(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sessionToken := getStr(req, "sessionToken")
	if sessionToken == "" {
		return jsonErr(service.ErrValidation("sessionToken is required"))
	}
	if err := store.StopCisSession(sessionToken); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleSendCisSessionHealth(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sessionToken := getStr(req, "sessionToken")
	if sessionToken == "" {
		return jsonErr(service.ErrValidation("sessionToken is required"))
	}
	if !store.HasCisSession(sessionToken) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"CIS session not found: "+sessionToken, http.StatusNotFound))
	}
	return jsonOK(map[string]any{})
}

func handleSendCisSessionTelemetry(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sessionToken := getStr(req, "sessionToken")
	if sessionToken == "" {
		return jsonErr(service.ErrValidation("sessionToken is required"))
	}
	if !store.HasCisSession(sessionToken) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"CIS session not found: "+sessionToken, http.StatusNotFound))
	}
	return jsonOK(map[string]any{})
}

// ── Handlers: code security integrations ────────────────────────────────────

func handleCreateCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	i, err := store.CreateCodeSecurityIntegration(
		getStr(req, "name"),
		getStr(req, "type"),
		getStrMap(req, "tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"integrationArn":    i.IntegrationArn,
		"status":            i.Status,
		"authorizationUrl":  "",
	})
}

func handleDeleteCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "integrationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("integrationArn is required"))
	}
	if err := store.DeleteCodeSecurityIntegration(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"integrationArn": arn})
}

func handleGetCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "integrationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("integrationArn is required"))
	}
	i, err := store.GetCodeSecurityIntegration(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(codeSecurityIntegrationToMap(i))
}

func handleListCodeSecurityIntegrations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListCodeSecurityIntegrations()
	out := make([]map[string]any, 0, len(list))
	for _, i := range list {
		out = append(out, codeSecurityIntegrationToMap(i))
	}
	return jsonOK(map[string]any{"integrations": out})
}

func handleUpdateCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "integrationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("integrationArn is required"))
	}
	i, err := store.UpdateCodeSecurityIntegration(arn, "ACTIVE")
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"integrationArn": i.IntegrationArn,
		"status":         i.Status,
	})
}

// ── Handlers: code security scan configurations ─────────────────────────────

func handleCreateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	cfg, err := store.CreateCodeSecurityScanConfiguration(
		getStr(req, "name"),
		getStr(req, "level"),
		getMap(req, "configuration"),
		getMap(req, "scopeSettings"),
		getStrMap(req, "tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanConfigurationArn": cfg.ScanConfigurationArn})
}

func handleDeleteCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "scanConfigurationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("scanConfigurationArn is required"))
	}
	if err := store.DeleteCodeSecurityScanConfiguration(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanConfigurationArn": arn})
}

func handleGetCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "scanConfigurationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("scanConfigurationArn is required"))
	}
	cfg, err := store.GetCodeSecurityScanConfiguration(arn)
	if err != nil {
		return jsonErr(err)
	}
	out := codeSecurityScanConfigToMap(cfg)
	out["createdAt"] = rfc3339(cfg.CreatedAt)
	out["lastUpdatedAt"] = rfc3339(cfg.LastUpdatedAt)
	return jsonOK(out)
}

func handleListCodeSecurityScanConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListCodeSecurityScanConfigurations()
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, codeSecurityScanConfigToMap(c))
	}
	return jsonOK(map[string]any{"configurations": out})
}

func handleUpdateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "scanConfigurationArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("scanConfigurationArn is required"))
	}
	cfg, err := store.UpdateCodeSecurityScanConfiguration(arn, getMap(req, "configuration"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanConfigurationArn": cfg.ScanConfigurationArn})
}

func handleListCodeSecurityScanConfigurationAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "scanConfigurationArn")
	projects := store.ListCodeSecurityScanConfigurationAssociations(arn)
	associations := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		associations = append(associations, map[string]any{
			"resource": map[string]any{
				"projectId": p,
			},
		})
	}
	return jsonOK(map[string]any{"associations": associations})
}

func handleStartCodeSecurityScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resource := getMap(req, "resource")
	if resource == nil {
		return jsonErr(service.ErrValidation("resource is required"))
	}
	scan := store.StartCodeSecurityScan(resource)
	return jsonOK(map[string]any{
		"scanId": scan.ScanId,
		"status": scan.Status,
	})
}

func handleGetCodeSecurityScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	scanId := getStr(req, "scanId")
	if scanId == "" {
		return jsonErr(service.ErrValidation("scanId is required"))
	}
	scan, err := store.GetCodeSecurityScan(scanId)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"scanId":    scan.ScanId,
		"accountId": scan.AccountId,
		"resource":  scan.Resource,
		"status":    scan.Status,
		"createdAt": rfc3339(scan.CreatedAt),
		"updatedAt": rfc3339(scan.UpdatedAt),
	})
}

// ── Handlers: filters ───────────────────────────────────────────────────────

func handleCreateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	desc := getStr(req, "description")
	reason := getStr(req, "reason")
	f, err := store.CreateFilter(
		getStr(req, "name"),
		desc,
		getStr(req, "action"),
		reason,
		getMap(req, "filterCriteria"),
		getStrMap(req, "tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"arn": f.Arn})
}

func handleDeleteFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("arn is required"))
	}
	if err := store.DeleteFilter(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"arn": arn})
}

func handleUpdateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "filterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("filterArn is required"))
	}
	f, err := store.UpdateFilter(
		arn,
		getStrPtr(req, "name"),
		getStrPtr(req, "description"),
		getStrPtr(req, "action"),
		getStrPtr(req, "reason"),
		getMap(req, "filterCriteria"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"arn": f.Arn})
}

func handleListFilters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListFilters(getStr(req, "action"), getStrList(req, "arns"))
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		out = append(out, filterToMap(f))
	}
	return jsonOK(map[string]any{"filters": out})
}

// ── Handlers: organization configuration ────────────────────────────────────

func handleDescribeOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	cfg := store.GetOrganizationConfiguration()
	return jsonOK(map[string]any{
		"autoEnable":             orgConfigToMap(cfg),
		"maxAccountLimitReached": false,
	})
}

func handleUpdateOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	auto := getMap(req, "autoEnable")
	if auto == nil {
		return jsonErr(service.ErrValidation("autoEnable is required"))
	}
	cfg := &StoredOrganizationConfig{
		Ec2:            getBool(auto, "ec2"),
		Ecr:            getBool(auto, "ecr"),
		Lambda:         getBool(auto, "lambda"),
		LambdaCode:     getBool(auto, "lambdaCode"),
		CodeRepository: getBool(auto, "codeRepository"),
	}
	store.SetOrganizationConfiguration(cfg)
	return jsonOK(map[string]any{"autoEnable": orgConfigToMap(cfg)})
}

// ── Handlers: enable / disable ──────────────────────────────────────────────

func handleEnable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	statuses := store.EnableAccount(getStrList(req, "accountIds"), getStrList(req, "resourceTypes"))
	accounts := make([]map[string]any, 0, len(statuses))
	for _, st := range statuses {
		accounts = append(accounts, accountToMap(st))
	}
	return jsonOK(map[string]any{
		"accounts":       accounts,
		"failedAccounts": []map[string]any{},
	})
}

func handleDisable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	statuses := store.DisableAccount(getStrList(req, "accountIds"), getStrList(req, "resourceTypes"))
	accounts := make([]map[string]any, 0, len(statuses))
	for _, st := range statuses {
		accounts = append(accounts, accountToMap(st))
	}
	return jsonOK(map[string]any{
		"accounts":       accounts,
		"failedAccounts": []map[string]any{},
	})
}

// ── Handlers: delegated admin ───────────────────────────────────────────────

func handleEnableDelegatedAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "delegatedAdminAccountId")
	if err := store.EnableDelegatedAdminAccount(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"delegatedAdminAccountId": id})
}

func handleDisableDelegatedAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "delegatedAdminAccountId")
	if id == "" {
		return jsonErr(service.ErrValidation("delegatedAdminAccountId is required"))
	}
	if err := store.DisableDelegatedAdminAccount(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"delegatedAdminAccountId": id})
}

func handleGetDelegatedAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	id, status := store.GetDelegatedAdminAccount()
	if id == "" {
		return jsonOK(map[string]any{})
	}
	return jsonOK(map[string]any{
		"delegatedAdmin": map[string]any{
			"accountId":          id,
			"relationshipStatus": status,
		},
	})
}

func handleListDelegatedAdminAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListDelegatedAdminAccounts()
	out := make([]map[string]any, 0, len(list))
	for _, e := range list {
		out = append(out, map[string]any{
			"accountId": e["accountId"],
			"status":    e["status"],
		})
	}
	return jsonOK(map[string]any{"delegatedAdminAccounts": out})
}

// ── Handlers: configuration (EC2/ECR) ───────────────────────────────────────

func handleGetConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	ec2 := store.GetEc2Configuration()
	ecr := store.GetEcrConfiguration()
	return jsonOK(map[string]any{
		"ec2Configuration": map[string]any{
			"scanModeState": map[string]any{
				"scanMode":       getStr(ec2, "scanMode"),
				"scanModeStatus": "SUCCESS",
			},
		},
		"ecrConfiguration": map[string]any{
			"rescanDurationState": map[string]any{
				"rescanDuration":         getStr(ecr, "rescanDuration"),
				"pullDateRescanDuration": getStr(ecr, "pullDateRescanDuration"),
				"pullDateRescanMode":     getStr(ecr, "pullDateRescanMode"),
				"status":                 "SUCCESS",
			},
		},
	})
}

func handleUpdateConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if cfg := getMap(req, "ec2Configuration"); cfg != nil {
		store.SetEc2Configuration(cfg)
	}
	if cfg := getMap(req, "ecrConfiguration"); cfg != nil {
		store.SetEcrConfiguration(cfg)
	}
	return jsonOK(map[string]any{})
}

// ── Handlers: EC2 deep inspection ───────────────────────────────────────────

func handleGetEc2DeepInspectionConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	di := store.GetEc2DeepInspection(store.AccountID())
	out := map[string]any{
		"status":          di.Status,
		"packagePaths":    di.PackagePaths,
		"orgPackagePaths": di.OrgPackagePaths,
	}
	if di.PackagePaths == nil {
		out["packagePaths"] = []string{}
	}
	if di.OrgPackagePaths == nil {
		out["orgPackagePaths"] = []string{}
	}
	return jsonOK(out)
}

func handleUpdateEc2DeepInspectionConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	activate := getBool(req, "activateDeepInspection")
	paths := getStrList(req, "packagePaths")
	di := store.UpdateEc2DeepInspection(store.AccountID(), paths, activate)
	out := map[string]any{
		"status":          di.Status,
		"packagePaths":    di.PackagePaths,
		"orgPackagePaths": store.OrgEc2DeepInspectionPackagePaths(),
	}
	if di.PackagePaths == nil {
		out["packagePaths"] = []string{}
	}
	if out["orgPackagePaths"] == nil {
		out["orgPackagePaths"] = []string{}
	}
	return jsonOK(out)
}

func handleUpdateOrgEc2DeepInspectionConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.SetOrgEc2DeepInspectionPackagePaths(getStrList(req, "orgPackagePaths"))
	return jsonOK(map[string]any{})
}

// ── Handlers: encryption keys ───────────────────────────────────────────────

func handleGetEncryptionKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	resourceType := ctx.Params["resourceType"]
	scanType := ctx.Params["scanType"]
	if resourceType == "" || scanType == "" {
		var req map[string]any
		_ = parseJSON(ctx.Body, &req)
		if resourceType == "" {
			resourceType = getStr(req, "resourceType")
		}
		if scanType == "" {
			scanType = getStr(req, "scanType")
		}
	}
	if resourceType == "" || scanType == "" {
		return jsonErr(service.ErrValidation("resourceType and scanType are required"))
	}
	key := store.GetEncryptionKey(scanType, resourceType)
	if key == "" {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No encryption key configured for "+scanType+"/"+resourceType, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"kmsKeyId": key})
}

func handleUpdateEncryptionKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.SetEncryptionKey(
		getStr(req, "scanType"),
		getStr(req, "resourceType"),
		getStr(req, "kmsKeyId"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleResetEncryptionKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.ResetEncryptionKey(getStr(req, "scanType"), getStr(req, "resourceType")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Handlers: list / search read-only ───────────────────────────────────────

func handleGetClustersForImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if filter := getMap(req, "filter"); filter == nil || getStr(filter, "resourceId") == "" {
		return jsonErr(service.ErrValidation("filter.resourceId is required"))
	}
	return jsonOK(map[string]any{"cluster": []map[string]any{}})
}

func handleListAccountPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	svcModel := getStr(req, "service")
	perms := []map[string]any{
		{"operation": "ENABLE_SCANNING", "service": "EC2"},
		{"operation": "ENABLE_SCANNING", "service": "ECR"},
		{"operation": "ENABLE_SCANNING", "service": "LAMBDA"},
	}
	if svcModel != "" {
		filtered := make([]map[string]any, 0, len(perms))
		for _, p := range perms {
			if p["service"] == svcModel {
				filtered = append(filtered, p)
			}
		}
		perms = filtered
	}
	return jsonOK(map[string]any{"permissions": perms})
}

func handleListCoverage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"coveredResources": []map[string]any{}})
}

func handleListCoverageStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"countsByGroup": []map[string]any{},
		"totalCounts":   0,
	})
}

func handleListFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListFindings()
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		entry := map[string]any{
			"findingArn":      f.FindingArn,
			"awsAccountId":    f.AwsAccountId,
			"description":     f.Description,
			"severity":        f.Severity,
			"status":          f.Status,
			"type":            f.FindingType,
			"firstObservedAt": rfc3339(f.FirstObservedAt),
			"lastObservedAt":  rfc3339(f.LastObservedAt),
		}
		if f.Title != "" {
			entry["title"] = f.Title
		}
		if f.InspectorScore > 0 {
			entry["inspectorScore"] = f.InspectorScore
		}
		if f.ExploitAvailable != "" {
			entry["exploitAvailable"] = f.ExploitAvailable
		}
		if f.FixAvailable != "" {
			entry["fixAvailable"] = f.FixAvailable
		}
		if len(f.Resources) > 0 {
			entry["resources"] = f.Resources
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"findings": out})
}

func handleListFindingAggregations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	aggType := getStr(req, "aggregationType")
	if aggType == "" {
		return jsonErr(service.ErrValidation("aggregationType is required"))
	}
	return jsonOK(map[string]any{
		"aggregationType": aggType,
		"responses":       []map[string]any{},
	})
}

func handleListUsageTotals(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "accountIds")
	if len(ids) == 0 {
		ids = []string{store.AccountID()}
	}
	totals := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		totals = append(totals, map[string]any{
			"accountId": id,
			"usage": []map[string]any{
				{
					"type":                 "EC2_INSTANCE_HOURS",
					"total":                0.0,
					"estimatedMonthlyCost": 0.0,
					"currency":             "USD",
				},
			},
		})
	}
	return jsonOK(map[string]any{"totals": totals})
}

func handleSearchVulnerabilities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	criteria := getMap(req, "filterCriteria")
	if criteria == nil {
		return jsonErr(service.ErrValidation("filterCriteria is required"))
	}
	ids := getStrList(criteria, "vulnerabilityIds")
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]any{
			"id":          id,
			"description": "Mock vulnerability " + id,
			"source":      "NVD",
			"cwes":        []string{},
			"referenceUrls": []string{
				"https://nvd.nist.gov/vuln/detail/" + id,
			},
		})
	}
	return jsonOK(map[string]any{"vulnerabilities": out})
}

// ── Handlers: tags ──────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required"))
	}
	store.TagResource(arn, getStrMap(req, "tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	arn := ctx.Params["resourceArn"]
	keys := []string{}
	if v, ok := ctx.Params["tagKeys"]; ok && v != "" {
		keys = append(keys, v)
	}
	if len(ctx.Body) > 0 {
		var req map[string]any
		_ = parseJSON(ctx.Body, &req)
		if arn == "" {
			arn = getStr(req, "resourceArn")
		}
		if k := getStrList(req, "tagKeys"); len(k) > 0 {
			keys = k
		}
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required"))
	}
	store.UntagResource(arn, keys)
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	arn := ctx.Params["resourceArn"]
	if arn == "" {
		var req map[string]any
		_ = parseJSON(ctx.Body, &req)
		arn = getStr(req, "resourceArn")
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required"))
	}
	tags := store.ListTags(arn)
	return jsonOK(map[string]any{"tags": tags})
}
