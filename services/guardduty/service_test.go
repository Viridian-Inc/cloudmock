package guardduty_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/guardduty"
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
	req.Header.Set("X-Amz-Target", "guardduty."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/guardduty/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestAcceptAdministratorInvitation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AcceptAdministratorInvitation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AcceptAdministratorInvitation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAcceptInvitation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AcceptInvitation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AcceptInvitation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestArchiveFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ArchiveFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ArchiveFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateDetector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateDetector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDetector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestCreateIPSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateIPSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateIPSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateMalwareProtectionPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateMalwareProtectionPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMalwareProtectionPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreatePublishingDestination(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreatePublishingDestination", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreatePublishingDestination: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateSampleFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateSampleFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateSampleFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateThreatEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateThreatEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateThreatEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateThreatIntelSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateThreatIntelSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateThreatIntelSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTrustedEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTrustedEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTrustedEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeclineInvitations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeclineInvitations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeclineInvitations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDetector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDetector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDetector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDeleteIPSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteIPSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteIPSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteInvitations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteInvitations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteInvitations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteMalwareProtectionPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteMalwareProtectionPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteMalwareProtectionPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeletePublishingDestination(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeletePublishingDestination", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeletePublishingDestination: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteThreatEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteThreatEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteThreatEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteThreatIntelSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteThreatIntelSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteThreatIntelSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTrustedEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTrustedEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTrustedEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeMalwareScans(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeMalwareScans", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeMalwareScans: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDescribePublishingDestination(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribePublishingDestination", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribePublishingDestination: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisableOrganizationAdminAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisableOrganizationAdminAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisableOrganizationAdminAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateFromAdministratorAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateFromAdministratorAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateFromAdministratorAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateFromMasterAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateFromMasterAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateFromMasterAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestEnableOrganizationAdminAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "EnableOrganizationAdminAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("EnableOrganizationAdminAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetAdministratorAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetAdministratorAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAdministratorAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetCoverageStatistics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetCoverageStatistics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetCoverageStatistics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetDetector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetDetector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetDetector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFilter(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFilter", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFilter: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindingsStatistics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingsStatistics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingsStatistics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetIPSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetIPSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIPSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetInvitationsCount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetInvitationsCount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetInvitationsCount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMalwareProtectionPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMalwareProtectionPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMalwareProtectionPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMalwareScan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMalwareScan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMalwareScan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMalwareScanSettings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMalwareScanSettings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMalwareScanSettings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMasterAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMasterAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMasterAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMemberDetectors(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMemberDetectors", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMemberDetectors: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetOrganizationStatistics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetOrganizationStatistics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetOrganizationStatistics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetRemainingFreeTrialDays(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetRemainingFreeTrialDays", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetRemainingFreeTrialDays: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetThreatEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetThreatEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetThreatEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetThreatIntelSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetThreatIntelSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetThreatIntelSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetTrustedEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetTrustedEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetTrustedEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetUsageStatistics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetUsageStatistics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetUsageStatistics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestInviteMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "InviteMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("InviteMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListDetectors(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDetectors", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDetectors: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListIPSets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListIPSets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIPSets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListInvitations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListInvitations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListInvitations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListMalwareProtectionPlans(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListMalwareProtectionPlans", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListMalwareProtectionPlans: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListMalwareScans(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListMalwareScans", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListMalwareScans: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListOrganizationAdminAccounts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListOrganizationAdminAccounts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListOrganizationAdminAccounts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPublishingDestinations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListPublishingDestinations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListPublishingDestinations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListThreatEntitySets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListThreatEntitySets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListThreatEntitySets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListThreatIntelSets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListThreatIntelSets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListThreatIntelSets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTrustedEntitySets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTrustedEntitySets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTrustedEntitySets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSendObjectMalwareScan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SendObjectMalwareScan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SendObjectMalwareScan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartMalwareScan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartMalwareScan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartMalwareScan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartMonitoringMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartMonitoringMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartMonitoringMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStopMonitoringMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StopMonitoringMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StopMonitoringMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUnarchiveFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UnarchiveFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UnarchiveFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateDetector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDetector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDetector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateFindingsFeedback(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFindingsFeedback", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFindingsFeedback: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateIPSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateIPSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateIPSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMalwareProtectionPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateMalwareProtectionPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMalwareProtectionPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMalwareScanSettings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateMalwareScanSettings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMalwareScanSettings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMemberDetectors(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateMemberDetectors", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMemberDetectors: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdatePublishingDestination(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdatePublishingDestination", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePublishingDestination: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateThreatEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateThreatEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateThreatEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateThreatIntelSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateThreatIntelSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateThreatIntelSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTrustedEntitySet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTrustedEntitySet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTrustedEntitySet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

