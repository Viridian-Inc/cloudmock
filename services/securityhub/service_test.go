package securityhub_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/securityhub"
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
	req.Header.Set("X-Amz-Target", "securityhub."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/securityhub/aws4_request, SignedHeaders=host, Signature=abc123")
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

func TestBatchDeleteAutomationRules(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDeleteAutomationRules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDeleteAutomationRules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDisableStandards(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDisableStandards", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDisableStandards: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchEnableStandards(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchEnableStandards", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchEnableStandards: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetAutomationRules(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetAutomationRules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetAutomationRules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetConfigurationPolicyAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetConfigurationPolicyAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetConfigurationPolicyAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetSecurityControls(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetSecurityControls", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetSecurityControls: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchGetStandardsControlAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchGetStandardsControlAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchGetStandardsControlAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchImportFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchImportFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchImportFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchUpdateAutomationRules(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchUpdateAutomationRules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchUpdateAutomationRules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchUpdateFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchUpdateFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchUpdateFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchUpdateFindingsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchUpdateFindingsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchUpdateFindingsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchUpdateStandardsControlAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchUpdateStandardsControlAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchUpdateStandardsControlAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateActionTarget(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateActionTarget", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateActionTarget: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAggregatorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAggregatorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAggregatorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAutomationRule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAutomationRule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAutomationRule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAutomationRuleV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAutomationRuleV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAutomationRuleV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateConfigurationPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateConfigurationPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateConfigurationPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateConnectorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateConnectorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateConnectorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFindingAggregator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFindingAggregator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFindingAggregator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateInsight(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateInsight", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateInsight: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestCreateTicketV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTicketV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTicketV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDeleteActionTarget(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteActionTarget", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteActionTarget: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAggregatorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAggregatorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAggregatorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAutomationRuleV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAutomationRuleV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAutomationRuleV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteConfigurationPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteConfigurationPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteConfigurationPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteConnectorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteConnectorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteConnectorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFindingAggregator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFindingAggregator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFindingAggregator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteInsight(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteInsight", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteInsight: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDeleteMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeActionTargets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeActionTargets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeActionTargets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeHub(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeHub", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeHub: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDescribeProducts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProducts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProducts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProductsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProductsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProductsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeSecurityHubV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeSecurityHubV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeSecurityHubV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeStandards(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeStandards", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStandards: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeStandardsControls(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeStandardsControls", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeStandardsControls: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisableImportFindingsForProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisableImportFindingsForProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisableImportFindingsForProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestDisableSecurityHub(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisableSecurityHub", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisableSecurityHub: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisableSecurityHubV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisableSecurityHubV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisableSecurityHubV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestEnableImportFindingsForProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "EnableImportFindingsForProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("EnableImportFindingsForProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestEnableSecurityHub(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "EnableSecurityHub", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("EnableSecurityHub: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestEnableSecurityHubV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "EnableSecurityHubV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("EnableSecurityHubV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestGetAggregatorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetAggregatorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAggregatorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetAutomationRuleV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetAutomationRuleV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAutomationRuleV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetConfigurationPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetConfigurationPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetConfigurationPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetConfigurationPolicyAssociation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetConfigurationPolicyAssociation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetConfigurationPolicyAssociation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetConnectorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetConnectorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetConnectorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetEnabledStandards(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetEnabledStandards", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetEnabledStandards: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindingAggregator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingAggregator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingAggregator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindingHistory(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingHistory", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingHistory: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindingStatisticsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingStatisticsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingStatisticsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestGetFindingsTrendsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingsTrendsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingsTrendsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFindingsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFindingsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFindingsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetInsightResults(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetInsightResults", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetInsightResults: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetInsights(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetInsights", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetInsights: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestGetMasterAccount(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMasterAccount", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMasterAccount: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestGetResourcesStatisticsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetResourcesStatisticsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetResourcesStatisticsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetResourcesTrendsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetResourcesTrendsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetResourcesTrendsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetResourcesV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetResourcesV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetResourcesV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSecurityControlDefinition(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSecurityControlDefinition", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSecurityControlDefinition: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListAggregatorsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAggregatorsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAggregatorsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAutomationRules(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAutomationRules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAutomationRules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAutomationRulesV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAutomationRulesV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAutomationRulesV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListConfigurationPolicies(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListConfigurationPolicies", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListConfigurationPolicies: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListConfigurationPolicyAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListConfigurationPolicyAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListConfigurationPolicyAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListConnectorsV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListConnectorsV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListConnectorsV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEnabledProductsForImport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEnabledProductsForImport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEnabledProductsForImport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFindingAggregators(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFindingAggregators", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFindingAggregators: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListSecurityControlDefinitions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListSecurityControlDefinitions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSecurityControlDefinitions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListStandardsControlAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListStandardsControlAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListStandardsControlAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestRegisterConnectorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RegisterConnectorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RegisterConnectorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartConfigurationPolicyAssociation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartConfigurationPolicyAssociation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartConfigurationPolicyAssociation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartConfigurationPolicyDisassociation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartConfigurationPolicyDisassociation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartConfigurationPolicyDisassociation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateActionTarget(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateActionTarget", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateActionTarget: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAggregatorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAggregatorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAggregatorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAutomationRuleV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAutomationRuleV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAutomationRuleV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateConfigurationPolicy(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateConfigurationPolicy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateConfigurationPolicy: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateConnectorV2(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateConnectorV2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateConnectorV2: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFindingAggregator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFindingAggregator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFindingAggregator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFindings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFindings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFindings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateInsight(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateInsight", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateInsight: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateSecurityControl(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateSecurityControl", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateSecurityControl: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSecurityHubConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateSecurityHubConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateSecurityHubConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateStandardsControl(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateStandardsControl", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateStandardsControl: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

