package inspector2_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/inspector2"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "inspector2."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/inspector2/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestAssociateMember(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociateMember", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociateMember: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchAssociateCodeSecurityScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchAssociateCodeSecurityScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchAssociateCodeSecurityScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDisassociateCodeSecurityScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDisassociateCodeSecurityScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDisassociateCodeSecurityScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetAccountStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetAccountStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetAccountStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetCodeSnippet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetCodeSnippet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetCodeSnippet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetFindingDetails(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetFindingDetails", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetFindingDetails: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetFreeTrialInfo(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetFreeTrialInfo", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetFreeTrialInfo: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetMemberEc2DeepInspectionStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetMemberEc2DeepInspectionStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetMemberEc2DeepInspectionStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchUpdateMemberEc2DeepInspectionStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchUpdateMemberEc2DeepInspectionStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchUpdateMemberEc2DeepInspectionStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCancelFindingsReport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CancelFindingsReport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CancelFindingsReport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCancelSbomExport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CancelSbomExport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CancelSbomExport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCisScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCisScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCisScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCodeSecurityIntegration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCodeSecurityIntegration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCodeSecurityIntegration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCodeSecurityScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCodeSecurityScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCodeSecurityScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFilter(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFilter", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFilter: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFindingsReport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFindingsReport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFindingsReport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateSbomExport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateSbomExport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateSbomExport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCisScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCisScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCisScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCodeSecurityIntegration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCodeSecurityIntegration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCodeSecurityIntegration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCodeSecurityScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCodeSecurityScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCodeSecurityScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFilter(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFilter", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFilter: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeOrganizationConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeOrganizationConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeOrganizationConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "Disable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("Disable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisableDelegatedAdminAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisableDelegatedAdminAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisableDelegatedAdminAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateMember(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateMember", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateMember: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestEnable(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "Enable", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("Enable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestEnableDelegatedAdminAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "EnableDelegatedAdminAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("EnableDelegatedAdminAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCisScanReport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCisScanReport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCisScanReport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCisScanResultDetails(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCisScanResultDetails", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCisScanResultDetails: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetClustersForImage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetClustersForImage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetClustersForImage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCodeSecurityIntegration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCodeSecurityIntegration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCodeSecurityIntegration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCodeSecurityScan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCodeSecurityScan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCodeSecurityScan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCodeSecurityScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCodeSecurityScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCodeSecurityScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetDelegatedAdminAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetDelegatedAdminAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetDelegatedAdminAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetEc2DeepInspectionConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetEc2DeepInspectionConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetEc2DeepInspectionConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetEncryptionKey(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetEncryptionKey", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetEncryptionKey: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindingsReportStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingsReportStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingsReportStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMember(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMember", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMember: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSbomExport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSbomExport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSbomExport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAccountPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAccountPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAccountPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCisScanConfigurations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCisScanConfigurations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCisScanConfigurations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCisScanResultsAggregatedByChecks(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCisScanResultsAggregatedByChecks", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCisScanResultsAggregatedByChecks: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCisScanResultsAggregatedByTargetResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCisScanResultsAggregatedByTargetResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCisScanResultsAggregatedByTargetResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCisScans(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCisScans", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCisScans: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCodeSecurityIntegrations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCodeSecurityIntegrations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCodeSecurityIntegrations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCodeSecurityScanConfigurationAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCodeSecurityScanConfigurationAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCodeSecurityScanConfigurationAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCodeSecurityScanConfigurations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCodeSecurityScanConfigurations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCodeSecurityScanConfigurations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCoverage(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCoverage", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCoverage: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCoverageStatistics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCoverageStatistics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCoverageStatistics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDelegatedAdminAccounts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDelegatedAdminAccounts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDelegatedAdminAccounts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFilters(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFilters", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFilters: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFindingAggregations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFindingAggregations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFindingAggregations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTagsForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTagsForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListUsageTotals(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListUsageTotals", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListUsageTotals: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestResetEncryptionKey(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ResetEncryptionKey", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ResetEncryptionKey: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchVulnerabilities(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchVulnerabilities", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchVulnerabilities: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSendCisSessionHealth(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SendCisSessionHealth", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SendCisSessionHealth: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSendCisSessionTelemetry(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SendCisSessionTelemetry", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SendCisSessionTelemetry: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartCisSession(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartCisSession", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartCisSession: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartCodeSecurityScan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartCodeSecurityScan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartCodeSecurityScan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopCisSession(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopCisSession", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopCisSession: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUntagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UntagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCisScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCisScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCisScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCodeSecurityIntegration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCodeSecurityIntegration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCodeSecurityIntegration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCodeSecurityScanConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCodeSecurityScanConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCodeSecurityScanConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateEc2DeepInspectionConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateEc2DeepInspectionConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateEc2DeepInspectionConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateEncryptionKey(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateEncryptionKey", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateEncryptionKey: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFilter(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFilter", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFilter: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateOrgEc2DeepInspectionConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateOrgEc2DeepInspectionConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateOrgEc2DeepInspectionConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateOrganizationConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateOrganizationConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateOrganizationConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

