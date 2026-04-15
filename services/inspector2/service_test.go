package inspector2_test

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
	svc "github.com/Viridian-Inc/cloudmock/services/inspector2"
)

// ── Test helpers ─────────────────────────────────────────────────────────────

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body == nil {
		data = []byte("{}")
	} else {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Inspector2."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/inspector2/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func mustOK(t *testing.T, w *httptest.ResponseRecorder, action string) map[string]any {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d body=%s", action, w.Code, w.Body.String())
	}
	out := map[string]any{}
	if w.Body.Len() == 0 {
		return out
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: decode: %v body=%s", action, err, w.Body.String())
	}
	return out
}

func mustErr(t *testing.T, w *httptest.ResponseRecorder, action string, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("%s: expected %d, got %d body=%s", action, code, w.Code, w.Body.String())
	}
}

// ── Lifecycle tests ──────────────────────────────────────────────────────────

func TestEnableDisableLifecycle(t *testing.T) {
	h := newGateway(t)

	out := mustOK(t, doCall(t, h, "Enable", map[string]any{
		"accountIds":    []string{"111122223333"},
		"resourceTypes": []string{"EC2", "ECR"},
	}), "Enable")
	accounts, _ := out["accounts"].([]any)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %v", out)
	}
	first, _ := accounts[0].(map[string]any)
	rs, _ := first["resourceStatus"].(map[string]any)
	if rs["ec2"] != "ENABLED" || rs["ecr"] != "ENABLED" {
		t.Fatalf("EC2/ECR not enabled: %v", rs)
	}
	if rs["lambda"] != "DISABLED" {
		t.Fatalf("Lambda should still be DISABLED: %v", rs)
	}

	out = mustOK(t, doCall(t, h, "BatchGetAccountStatus", map[string]any{
		"accountIds": []string{"111122223333"},
	}), "BatchGetAccountStatus")
	accs, _ := out["accounts"].([]any)
	if len(accs) != 1 {
		t.Fatalf("BatchGetAccountStatus: expected 1, got %v", out)
	}

	mustOK(t, doCall(t, h, "Disable", map[string]any{
		"accountIds":    []string{"111122223333"},
		"resourceTypes": []string{"EC2"},
	}), "Disable")

	out = mustOK(t, doCall(t, h, "BatchGetAccountStatus", map[string]any{
		"accountIds": []string{"111122223333"},
	}), "BatchGetAccountStatus")
	accs, _ = out["accounts"].([]any)
	first, _ = accs[0].(map[string]any)
	state, _ := first["resourceState"].(map[string]any)
	ec2, _ := state["ec2"].(map[string]any)
	if ec2["status"] != "DISABLED" {
		t.Fatalf("expected EC2 DISABLED after disable, got %v", state)
	}
}

func TestFilterLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateFilter", map[string]any{
		"name":        "myfilter",
		"description": "test filter",
		"action":      "SUPPRESS",
		"reason":      "false-positive",
		"filterCriteria": map[string]any{
			"severity": []map[string]any{
				{"comparison": "EQUALS", "value": "HIGH"},
			},
		},
		"tags": map[string]string{"env": "dev"},
	}), "CreateFilter")

	arn, _ := create["arn"].(string)
	if arn == "" {
		t.Fatalf("CreateFilter: missing arn: %v", create)
	}

	dup := doCall(t, h, "CreateFilter", map[string]any{
		"name":   "myfilter",
		"action": "SUPPRESS",
	})
	if dup.Code != http.StatusConflict {
		t.Fatalf("CreateFilter duplicate: expected 409, got %d body=%s", dup.Code, dup.Body.String())
	}

	list := mustOK(t, doCall(t, h, "ListFilters", nil), "ListFilters")
	filters, _ := list["filters"].([]any)
	if len(filters) != 1 {
		t.Fatalf("ListFilters: expected 1 filter, got %d body=%v", len(filters), list)
	}

	updated := mustOK(t, doCall(t, h, "UpdateFilter", map[string]any{
		"filterArn":   arn,
		"description": "updated desc",
	}), "UpdateFilter")
	if updated["arn"] != arn {
		t.Fatalf("UpdateFilter: arn mismatch: %v", updated)
	}

	delResp := mustOK(t, doCall(t, h, "DeleteFilter", map[string]any{"arn": arn}), "DeleteFilter")
	if delResp["arn"] != arn {
		t.Fatalf("DeleteFilter: arn mismatch: %v", delResp)
	}

	notFound := doCall(t, h, "DeleteFilter", map[string]any{"arn": arn})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("DeleteFilter after delete: expected 404, got %d body=%s", notFound.Code, notFound.Body.String())
	}

	list = mustOK(t, doCall(t, h, "ListFilters", nil), "ListFilters")
	filters, _ = list["filters"].([]any)
	if len(filters) != 0 {
		t.Fatalf("ListFilters after delete: expected 0, got %d", len(filters))
	}
}

func TestFilterValidation(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateFilter", map[string]any{"action": "NONE"})
	mustErr(t, w, "CreateFilter missing name", http.StatusBadRequest)

	w = doCall(t, h, "CreateFilter", map[string]any{"name": "x"})
	mustErr(t, w, "CreateFilter missing action", http.StatusBadRequest)
}

func TestCisScanConfigurationLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateCisScanConfiguration", map[string]any{
		"scanName":      "weekly-scan",
		"securityLevel": "LEVEL_1",
		"schedule": map[string]any{
			"daily": map[string]any{
				"startTime": map[string]any{"timeOfDay": "04:00", "timezone": "UTC"},
			},
		},
		"targets": map[string]any{
			"accountIds": []string{"111122223333"},
		},
		"tags": map[string]string{"team": "secops"},
	}), "CreateCisScanConfiguration")
	arn, _ := create["scanConfigurationArn"].(string)
	if arn == "" {
		t.Fatalf("CreateCisScanConfiguration: missing arn: %v", create)
	}

	list := mustOK(t, doCall(t, h, "ListCisScanConfigurations", nil), "ListCisScanConfigurations")
	configs, _ := list["scanConfigurations"].([]any)
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	mustOK(t, doCall(t, h, "UpdateCisScanConfiguration", map[string]any{
		"scanConfigurationArn": arn,
		"scanName":             "renamed",
	}), "UpdateCisScanConfiguration")

	mustOK(t, doCall(t, h, "DeleteCisScanConfiguration", map[string]any{
		"scanConfigurationArn": arn,
	}), "DeleteCisScanConfiguration")

	notFound := doCall(t, h, "DeleteCisScanConfiguration", map[string]any{"scanConfigurationArn": arn})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("DeleteCisScanConfiguration after delete: expected 404, got %d", notFound.Code)
	}
}

func TestCisSessionLifecycle(t *testing.T) {
	h := newGateway(t)

	mustOK(t, doCall(t, h, "StartCisSession", map[string]any{
		"scanJobId": "job-1",
		"message": map[string]any{
			"sessionToken": "tok-1",
		},
	}), "StartCisSession")

	mustOK(t, doCall(t, h, "SendCisSessionHealth", map[string]any{
		"sessionToken": "tok-1",
		"scanJobId":    "job-1",
	}), "SendCisSessionHealth")

	mustOK(t, doCall(t, h, "SendCisSessionTelemetry", map[string]any{
		"sessionToken": "tok-1",
		"scanJobId":    "job-1",
		"messages":     []map[string]any{},
	}), "SendCisSessionTelemetry")

	mustOK(t, doCall(t, h, "StopCisSession", map[string]any{
		"sessionToken": "tok-1",
		"scanJobId":    "job-1",
	}), "StopCisSession")

	notFound := doCall(t, h, "SendCisSessionHealth", map[string]any{
		"sessionToken": "tok-1",
		"scanJobId":    "job-1",
	})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("SendCisSessionHealth after stop: expected 404, got %d", notFound.Code)
	}
}

func TestCodeSecurityIntegrationLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateCodeSecurityIntegration", map[string]any{
		"name": "main-gitlab",
		"type": "GITLAB_SELF_MANAGED",
		"tags": map[string]string{"team": "secops"},
	}), "CreateCodeSecurityIntegration")
	arn, _ := create["integrationArn"].(string)
	if arn == "" {
		t.Fatalf("CreateCodeSecurityIntegration: missing arn: %v", create)
	}
	if create["status"] != "PENDING" {
		t.Fatalf("CreateCodeSecurityIntegration: expected PENDING, got %v", create)
	}

	got := mustOK(t, doCall(t, h, "GetCodeSecurityIntegration", map[string]any{
		"integrationArn": arn,
	}), "GetCodeSecurityIntegration")
	if got["integrationArn"] != arn {
		t.Fatalf("GetCodeSecurityIntegration: arn mismatch: %v", got)
	}

	updated := mustOK(t, doCall(t, h, "UpdateCodeSecurityIntegration", map[string]any{
		"integrationArn": arn,
	}), "UpdateCodeSecurityIntegration")
	if updated["status"] != "ACTIVE" {
		t.Fatalf("UpdateCodeSecurityIntegration: expected ACTIVE, got %v", updated)
	}

	list := mustOK(t, doCall(t, h, "ListCodeSecurityIntegrations", nil), "ListCodeSecurityIntegrations")
	if items, _ := list["integrations"].([]any); len(items) != 1 {
		t.Fatalf("ListCodeSecurityIntegrations: expected 1, got %d", len(items))
	}

	mustOK(t, doCall(t, h, "DeleteCodeSecurityIntegration", map[string]any{
		"integrationArn": arn,
	}), "DeleteCodeSecurityIntegration")

	notFound := doCall(t, h, "GetCodeSecurityIntegration", map[string]any{"integrationArn": arn})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("GetCodeSecurityIntegration after delete: expected 404, got %d", notFound.Code)
	}
}

func TestCodeSecurityScanConfigurationLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateCodeSecurityScanConfiguration", map[string]any{
		"name":  "scan-everything",
		"level": "ACCOUNT",
		"configuration": map[string]any{
			"ruleSetCategories": []string{"OWASP"},
		},
	}), "CreateCodeSecurityScanConfiguration")
	arn, _ := create["scanConfigurationArn"].(string)
	if arn == "" {
		t.Fatalf("CreateCodeSecurityScanConfiguration: missing arn: %v", create)
	}

	got := mustOK(t, doCall(t, h, "GetCodeSecurityScanConfiguration", map[string]any{
		"scanConfigurationArn": arn,
	}), "GetCodeSecurityScanConfiguration")
	if got["scanConfigurationArn"] != arn {
		t.Fatalf("GetCodeSecurityScanConfiguration: mismatch: %v", got)
	}

	mustOK(t, doCall(t, h, "UpdateCodeSecurityScanConfiguration", map[string]any{
		"scanConfigurationArn": arn,
		"configuration": map[string]any{
			"ruleSetCategories": []string{"CWE", "OWASP"},
		},
	}), "UpdateCodeSecurityScanConfiguration")

	// Associate / disassociate
	assoc := mustOK(t, doCall(t, h, "BatchAssociateCodeSecurityScanConfiguration", map[string]any{
		"associateConfigurationRequests": []map[string]any{
			{
				"scanConfigurationArn": arn,
				"resource": map[string]any{
					"projectId": "project-A",
				},
			},
		},
	}), "BatchAssociateCodeSecurityScanConfiguration")
	if successful, _ := assoc["successfulAssociations"].([]any); len(successful) != 1 {
		t.Fatalf("BatchAssociateCodeSecurityScanConfiguration: expected 1 successful, got %v", assoc)
	}

	listAssoc := mustOK(t, doCall(t, h, "ListCodeSecurityScanConfigurationAssociations", map[string]any{
		"scanConfigurationArn": arn,
	}), "ListCodeSecurityScanConfigurationAssociations")
	if associations, _ := listAssoc["associations"].([]any); len(associations) != 1 {
		t.Fatalf("ListCodeSecurityScanConfigurationAssociations: expected 1, got %v", listAssoc)
	}

	disassoc := mustOK(t, doCall(t, h, "BatchDisassociateCodeSecurityScanConfiguration", map[string]any{
		"disassociateConfigurationRequests": []map[string]any{
			{
				"scanConfigurationArn": arn,
				"resource": map[string]any{
					"projectId": "project-A",
				},
			},
		},
	}), "BatchDisassociateCodeSecurityScanConfiguration")
	if successful, _ := disassoc["successfulAssociations"].([]any); len(successful) != 1 {
		t.Fatalf("BatchDisassociateCodeSecurityScanConfiguration: expected 1 successful, got %v", disassoc)
	}

	mustOK(t, doCall(t, h, "DeleteCodeSecurityScanConfiguration", map[string]any{
		"scanConfigurationArn": arn,
	}), "DeleteCodeSecurityScanConfiguration")
}

func TestCodeSecurityScanLifecycle(t *testing.T) {
	h := newGateway(t)

	start := mustOK(t, doCall(t, h, "StartCodeSecurityScan", map[string]any{
		"resource": map[string]any{
			"projectId": "project-1",
		},
	}), "StartCodeSecurityScan")
	scanId, _ := start["scanId"].(string)
	if scanId == "" {
		t.Fatalf("StartCodeSecurityScan: missing scanId: %v", start)
	}

	got := mustOK(t, doCall(t, h, "GetCodeSecurityScan", map[string]any{
		"scanId": scanId,
	}), "GetCodeSecurityScan")
	if got["scanId"] != scanId {
		t.Fatalf("GetCodeSecurityScan: mismatch: %v", got)
	}
}

func TestMemberLifecycle(t *testing.T) {
	h := newGateway(t)

	mustOK(t, doCall(t, h, "AssociateMember", map[string]any{
		"accountId": "111122223333",
	}), "AssociateMember")

	got := mustOK(t, doCall(t, h, "GetMember", map[string]any{
		"accountId": "111122223333",
	}), "GetMember")
	member, _ := got["member"].(map[string]any)
	if member["accountId"] != "111122223333" {
		t.Fatalf("GetMember: mismatch: %v", got)
	}
	if member["relationshipStatus"] != "INVITED" {
		t.Fatalf("GetMember: expected INVITED, got %v", member)
	}

	list := mustOK(t, doCall(t, h, "ListMembers", nil), "ListMembers")
	if members, _ := list["members"].([]any); len(members) != 1 {
		t.Fatalf("ListMembers: expected 1, got %d", len(members))
	}

	mustOK(t, doCall(t, h, "DisassociateMember", map[string]any{
		"accountId": "111122223333",
	}), "DisassociateMember")

	listAssoc := mustOK(t, doCall(t, h, "ListMembers", map[string]any{
		"onlyAssociated": true,
	}), "ListMembers (onlyAssociated)")
	if members, _ := listAssoc["members"].([]any); len(members) != 0 {
		t.Fatalf("ListMembers onlyAssociated after disassociate: expected 0, got %d", len(members))
	}
}

func TestDelegatedAdminLifecycle(t *testing.T) {
	h := newGateway(t)

	mustOK(t, doCall(t, h, "EnableDelegatedAdminAccount", map[string]any{
		"delegatedAdminAccountId": "111122223333",
	}), "EnableDelegatedAdminAccount")

	got := mustOK(t, doCall(t, h, "GetDelegatedAdminAccount", nil), "GetDelegatedAdminAccount")
	admin, _ := got["delegatedAdmin"].(map[string]any)
	if admin["accountId"] != "111122223333" {
		t.Fatalf("GetDelegatedAdminAccount: mismatch: %v", got)
	}

	list := mustOK(t, doCall(t, h, "ListDelegatedAdminAccounts", nil), "ListDelegatedAdminAccounts")
	if items, _ := list["delegatedAdminAccounts"].([]any); len(items) != 1 {
		t.Fatalf("ListDelegatedAdminAccounts: expected 1, got %d", len(items))
	}

	mustOK(t, doCall(t, h, "DisableDelegatedAdminAccount", map[string]any{
		"delegatedAdminAccountId": "111122223333",
	}), "DisableDelegatedAdminAccount")

	notFound := doCall(t, h, "DisableDelegatedAdminAccount", map[string]any{
		"delegatedAdminAccountId": "111122223333",
	})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("DisableDelegatedAdminAccount after delete: expected 404, got %d", notFound.Code)
	}
}

func TestFindingsReportLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateFindingsReport", map[string]any{
		"reportFormat": "CSV",
		"s3Destination": map[string]any{
			"bucketName": "my-bucket",
			"keyPrefix":  "reports/",
			"kmsKeyArn":  "arn:aws:kms:us-east-1:111122223333:key/abc",
		},
	}), "CreateFindingsReport")
	id, _ := create["reportId"].(string)
	if id == "" {
		t.Fatalf("CreateFindingsReport: missing reportId: %v", create)
	}

	status := mustOK(t, doCall(t, h, "GetFindingsReportStatus", map[string]any{
		"reportId": id,
	}), "GetFindingsReportStatus")
	if status["status"] != "IN_PROGRESS" {
		t.Fatalf("GetFindingsReportStatus: expected IN_PROGRESS, got %v", status)
	}

	mustOK(t, doCall(t, h, "CancelFindingsReport", map[string]any{
		"reportId": id,
	}), "CancelFindingsReport")

	status = mustOK(t, doCall(t, h, "GetFindingsReportStatus", map[string]any{
		"reportId": id,
	}), "GetFindingsReportStatus after cancel")
	if status["status"] != "CANCELLED" {
		t.Fatalf("GetFindingsReportStatus: expected CANCELLED, got %v", status)
	}
}

func TestSbomExportLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateSbomExport", map[string]any{
		"reportFormat": "CYCLONEDX_1_4",
		"s3Destination": map[string]any{
			"bucketName": "my-bucket",
			"kmsKeyArn":  "arn:aws:kms:us-east-1:111122223333:key/abc",
		},
	}), "CreateSbomExport")
	id, _ := create["reportId"].(string)
	if id == "" {
		t.Fatalf("CreateSbomExport: missing reportId: %v", create)
	}

	got := mustOK(t, doCall(t, h, "GetSbomExport", map[string]any{
		"reportId": id,
	}), "GetSbomExport")
	if got["status"] != "IN_PROGRESS" {
		t.Fatalf("GetSbomExport: expected IN_PROGRESS, got %v", got)
	}

	mustOK(t, doCall(t, h, "CancelSbomExport", map[string]any{
		"reportId": id,
	}), "CancelSbomExport")

	got = mustOK(t, doCall(t, h, "GetSbomExport", map[string]any{
		"reportId": id,
	}), "GetSbomExport after cancel")
	if got["status"] != "CANCELLED" {
		t.Fatalf("GetSbomExport: expected CANCELLED, got %v", got)
	}
}

func TestEncryptionKeyLifecycle(t *testing.T) {
	h := newGateway(t)

	notFound := doCall(t, h, "GetEncryptionKey", map[string]any{
		"scanType":     "CODE",
		"resourceType": "AWS_LAMBDA_FUNCTION",
	})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("GetEncryptionKey before set: expected 404, got %d", notFound.Code)
	}

	mustOK(t, doCall(t, h, "UpdateEncryptionKey", map[string]any{
		"scanType":     "CODE",
		"resourceType": "AWS_LAMBDA_FUNCTION",
		"kmsKeyId":     "arn:aws:kms:us-east-1:111122223333:key/test-key",
	}), "UpdateEncryptionKey")

	got := mustOK(t, doCall(t, h, "GetEncryptionKey", map[string]any{
		"scanType":     "CODE",
		"resourceType": "AWS_LAMBDA_FUNCTION",
	}), "GetEncryptionKey")
	if got["kmsKeyId"] != "arn:aws:kms:us-east-1:111122223333:key/test-key" {
		t.Fatalf("GetEncryptionKey: mismatch: %v", got)
	}

	mustOK(t, doCall(t, h, "ResetEncryptionKey", map[string]any{
		"scanType":     "CODE",
		"resourceType": "AWS_LAMBDA_FUNCTION",
	}), "ResetEncryptionKey")

	notFound = doCall(t, h, "GetEncryptionKey", map[string]any{
		"scanType":     "CODE",
		"resourceType": "AWS_LAMBDA_FUNCTION",
	})
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("GetEncryptionKey after reset: expected 404, got %d", notFound.Code)
	}
}

func TestOrganizationConfiguration(t *testing.T) {
	h := newGateway(t)

	got := mustOK(t, doCall(t, h, "DescribeOrganizationConfiguration", nil), "DescribeOrganizationConfiguration")
	auto, _ := got["autoEnable"].(map[string]any)
	if auto == nil {
		t.Fatalf("DescribeOrganizationConfiguration: missing autoEnable: %v", got)
	}

	mustOK(t, doCall(t, h, "UpdateOrganizationConfiguration", map[string]any{
		"autoEnable": map[string]any{
			"ec2":            true,
			"ecr":            true,
			"lambda":         false,
			"lambdaCode":     false,
			"codeRepository": false,
		},
	}), "UpdateOrganizationConfiguration")

	got = mustOK(t, doCall(t, h, "DescribeOrganizationConfiguration", nil), "DescribeOrganizationConfiguration")
	auto, _ = got["autoEnable"].(map[string]any)
	if auto["ec2"] != true || auto["ecr"] != true || auto["lambda"] != false {
		t.Fatalf("DescribeOrganizationConfiguration: %v", auto)
	}
}

func TestEc2DeepInspectionLifecycle(t *testing.T) {
	h := newGateway(t)

	got := mustOK(t, doCall(t, h, "GetEc2DeepInspectionConfiguration", nil), "GetEc2DeepInspectionConfiguration")
	if got["status"] != "DEACTIVATED" {
		t.Fatalf("GetEc2DeepInspectionConfiguration: expected DEACTIVATED, got %v", got)
	}

	updated := mustOK(t, doCall(t, h, "UpdateEc2DeepInspectionConfiguration", map[string]any{
		"activateDeepInspection": true,
		"packagePaths":           []string{"/usr/local/bin"},
	}), "UpdateEc2DeepInspectionConfiguration")
	if updated["status"] != "ACTIVATED" {
		t.Fatalf("UpdateEc2DeepInspectionConfiguration: expected ACTIVATED, got %v", updated)
	}

	mustOK(t, doCall(t, h, "UpdateOrgEc2DeepInspectionConfiguration", map[string]any{
		"orgPackagePaths": []string{"/opt"},
	}), "UpdateOrgEc2DeepInspectionConfiguration")

	got = mustOK(t, doCall(t, h, "GetEc2DeepInspectionConfiguration", nil), "GetEc2DeepInspectionConfiguration")
	orgPaths, _ := got["orgPackagePaths"].([]any)
	if len(orgPaths) != 1 || orgPaths[0] != "/opt" {
		t.Fatalf("GetEc2DeepInspectionConfiguration: orgPackagePaths mismatch: %v", got)
	}
}

func TestConfigurationLifecycle(t *testing.T) {
	h := newGateway(t)

	mustOK(t, doCall(t, h, "UpdateConfiguration", map[string]any{
		"ec2Configuration": map[string]any{"scanMode": "EC2_HYBRID"},
		"ecrConfiguration": map[string]any{"rescanDuration": "DAYS_30"},
	}), "UpdateConfiguration")

	got := mustOK(t, doCall(t, h, "GetConfiguration", nil), "GetConfiguration")
	ec2cfg, _ := got["ec2Configuration"].(map[string]any)
	scanMode, _ := ec2cfg["scanModeState"].(map[string]any)
	if scanMode["scanMode"] != "EC2_HYBRID" {
		t.Fatalf("GetConfiguration: ec2 scanMode mismatch: %v", got)
	}
}

func TestTagsLifecycle(t *testing.T) {
	h := newGateway(t)

	create := mustOK(t, doCall(t, h, "CreateFilter", map[string]any{
		"name":   "tagged",
		"action": "NONE",
	}), "CreateFilter")
	arn, _ := create["arn"].(string)

	mustOK(t, doCall(t, h, "TagResource", map[string]any{
		"resourceArn": arn,
		"tags":        map[string]string{"team": "secops", "env": "prod"},
	}), "TagResource")

	got := mustOK(t, doCall(t, h, "ListTagsForResource", map[string]any{
		"resourceArn": arn,
	}), "ListTagsForResource")
	tags, _ := got["tags"].(map[string]any)
	if tags["team"] != "secops" || tags["env"] != "prod" {
		t.Fatalf("ListTagsForResource: mismatch: %v", got)
	}

	mustOK(t, doCall(t, h, "UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []string{"env"},
	}), "UntagResource")

	got = mustOK(t, doCall(t, h, "ListTagsForResource", map[string]any{
		"resourceArn": arn,
	}), "ListTagsForResource")
	tags, _ = got["tags"].(map[string]any)
	if _, ok := tags["env"]; ok {
		t.Fatalf("UntagResource: env tag still present: %v", tags)
	}
	if tags["team"] != "secops" {
		t.Fatalf("UntagResource: team tag removed: %v", tags)
	}
}

func TestSearchVulnerabilities(t *testing.T) {
	h := newGateway(t)

	got := mustOK(t, doCall(t, h, "SearchVulnerabilities", map[string]any{
		"filterCriteria": map[string]any{
			"vulnerabilityIds": []string{"CVE-2024-12345"},
		},
	}), "SearchVulnerabilities")
	vulns, _ := got["vulnerabilities"].([]any)
	if len(vulns) != 1 {
		t.Fatalf("SearchVulnerabilities: expected 1, got %v", got)
	}
	first, _ := vulns[0].(map[string]any)
	if first["id"] != "CVE-2024-12345" {
		t.Fatalf("SearchVulnerabilities: id mismatch: %v", first)
	}
}

func TestEmptyListsReturnEmpty(t *testing.T) {
	h := newGateway(t)

	out := mustOK(t, doCall(t, h, "ListCoverage", nil), "ListCoverage")
	if cov, _ := out["coveredResources"].([]any); len(cov) != 0 {
		t.Fatalf("ListCoverage: expected empty, got %v", out)
	}

	out = mustOK(t, doCall(t, h, "ListCoverageStatistics", nil), "ListCoverageStatistics")
	if c, _ := out["countsByGroup"].([]any); len(c) != 0 {
		t.Fatalf("ListCoverageStatistics: expected empty, got %v", out)
	}

	out = mustOK(t, doCall(t, h, "ListFindings", nil), "ListFindings")
	if f, _ := out["findings"].([]any); len(f) != 0 {
		t.Fatalf("ListFindings: expected empty, got %v", out)
	}

	out = mustOK(t, doCall(t, h, "ListCisScans", nil), "ListCisScans")
	if s, _ := out["scans"].([]any); len(s) != 0 {
		t.Fatalf("ListCisScans: expected empty, got %v", out)
	}

	out = mustOK(t, doCall(t, h, "ListFindingAggregations", map[string]any{
		"aggregationType": "TITLE",
	}), "ListFindingAggregations")
	if r, _ := out["responses"].([]any); len(r) != 0 {
		t.Fatalf("ListFindingAggregations: expected empty, got %v", out)
	}
}

func TestUsageTotalsAndPermissions(t *testing.T) {
	h := newGateway(t)

	out := mustOK(t, doCall(t, h, "ListUsageTotals", map[string]any{
		"accountIds": []string{"111122223333", "444455556666"},
	}), "ListUsageTotals")
	totals, _ := out["totals"].([]any)
	if len(totals) != 2 {
		t.Fatalf("ListUsageTotals: expected 2, got %d", len(totals))
	}

	out = mustOK(t, doCall(t, h, "ListAccountPermissions", nil), "ListAccountPermissions")
	if perms, _ := out["permissions"].([]any); len(perms) == 0 {
		t.Fatalf("ListAccountPermissions: expected at least one permission, got 0")
	}

	out = mustOK(t, doCall(t, h, "ListAccountPermissions", map[string]any{
		"service": "EC2",
	}), "ListAccountPermissions filtered")
	perms, _ := out["permissions"].([]any)
	if len(perms) != 1 {
		t.Fatalf("ListAccountPermissions filtered: expected 1, got %d", len(perms))
	}
}

func TestBatchGetCodeSnippetAndFreeTrial(t *testing.T) {
	h := newGateway(t)

	out := mustOK(t, doCall(t, h, "BatchGetCodeSnippet", map[string]any{
		"findingArns": []string{
			"arn:aws:inspector2:us-east-1:111122223333:finding/abc",
		},
	}), "BatchGetCodeSnippet")
	if results, _ := out["codeSnippetResults"].([]any); len(results) != 1 {
		t.Fatalf("BatchGetCodeSnippet: expected 1 result, got %v", out)
	}

	out = mustOK(t, doCall(t, h, "BatchGetFreeTrialInfo", map[string]any{
		"accountIds": []string{"111122223333"},
	}), "BatchGetFreeTrialInfo")
	if accs, _ := out["accounts"].([]any); len(accs) != 1 {
		t.Fatalf("BatchGetFreeTrialInfo: expected 1 account, got %v", out)
	}
}

func TestBatchGetMemberEc2DeepInspectionStatus(t *testing.T) {
	h := newGateway(t)

	mustOK(t, doCall(t, h, "BatchUpdateMemberEc2DeepInspectionStatus", map[string]any{
		"accountIds": []map[string]any{
			{"accountId": "111122223333", "activateDeepInspection": true},
			{"accountId": "444455556666", "activateDeepInspection": false},
		},
	}), "BatchUpdateMemberEc2DeepInspectionStatus")

	out := mustOK(t, doCall(t, h, "BatchGetMemberEc2DeepInspectionStatus", map[string]any{
		"accountIds": []string{"111122223333", "444455556666"},
	}), "BatchGetMemberEc2DeepInspectionStatus")
	statuses, _ := out["accountIds"].([]any)
	if len(statuses) != 2 {
		t.Fatalf("BatchGetMemberEc2DeepInspectionStatus: expected 2, got %d", len(statuses))
	}
	first, _ := statuses[0].(map[string]any)
	second, _ := statuses[1].(map[string]any)
	if first["status"] == second["status"] {
		t.Fatalf("expected differing statuses, got %v / %v", first, second)
	}
}

func TestGetClustersForImage(t *testing.T) {
	h := newGateway(t)

	out := mustOK(t, doCall(t, h, "GetClustersForImage", map[string]any{
		"filter": map[string]any{
			"resourceId": "sha256:abc",
		},
	}), "GetClustersForImage")
	if c, _ := out["cluster"].([]any); len(c) != 0 {
		t.Fatalf("GetClustersForImage: expected empty cluster slice, got %v", out)
	}

	missing := doCall(t, h, "GetClustersForImage", map[string]any{"filter": map[string]any{}})
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("GetClustersForImage missing resourceId: expected 400, got %d", missing.Code)
	}
}

func TestCisScanReportRequiresArn(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "GetCisScanReport", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("GetCisScanReport without arn: expected 400, got %d", w.Code)
	}

	mustOK(t, doCall(t, h, "GetCisScanReport", map[string]any{
		"scanArn": "arn:aws:inspector2:us-east-1:111122223333:cis-scan/abc",
	}), "GetCisScanReport")

	mustOK(t, doCall(t, h, "GetCisScanResultDetails", map[string]any{
		"scanArn": "arn:aws:inspector2:us-east-1:111122223333:cis-scan/abc",
	}), "GetCisScanResultDetails")
}

// ── Smoke test for unknown action ────────────────────────────────────────────

func TestUnknownAction(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "BogusAction", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("BogusAction: expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "InvalidAction") {
		t.Fatalf("BogusAction: expected InvalidAction error, got %s", w.Body.String())
	}
}
