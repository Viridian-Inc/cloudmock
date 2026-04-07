package quicksight_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/quicksight"
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
	req.Header.Set("X-Amz-Target", "quicksight."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/quicksight/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestBatchCreateTopicReviewedAnswer(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchCreateTopicReviewedAnswer", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchCreateTopicReviewedAnswer: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDeleteTopicReviewedAnswer(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDeleteTopicReviewedAnswer", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDeleteTopicReviewedAnswer: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCancelIngestion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CancelIngestion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CancelIngestion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAccountCustomization(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAccountCustomization", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAccountCustomization: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAccountSubscription(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAccountSubscription", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAccountSubscription: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateActionConnector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateActionConnector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateActionConnector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAnalysis(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAnalysis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAnalysis: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateBrand(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateBrand", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateBrand: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCustomPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCustomPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCustomPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateDashboard(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateDashboard", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDashboard: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateDataSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateDataSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDataSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateDataSource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateDataSource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDataSource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFolder(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFolder", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFolder: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateFolderMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateFolderMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateFolderMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateGroupMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateGroupMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateGroupMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateIAMPolicyAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateIAMPolicyAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateIAMPolicyAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateIngestion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateIngestion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateIngestion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateNamespace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateNamespace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateNamespace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateRoleMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateRoleMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateRoleMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTemplate(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTemplate", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTemplateAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTemplateAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTemplateAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTheme(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTheme", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTheme: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateThemeAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateThemeAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateThemeAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTopic(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTopic", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTopic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTopicRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTopicRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTopicRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateVPCConnection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateVPCConnection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateVPCConnection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAccountCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAccountCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAccountCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAccountCustomization(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAccountCustomization", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAccountCustomization: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAccountSubscription(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAccountSubscription", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAccountSubscription: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteActionConnector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteActionConnector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteActionConnector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAnalysis(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAnalysis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAnalysis: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBrand(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteBrand", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteBrand: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBrandAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteBrandAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteBrandAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCustomPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCustomPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCustomPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDashboard(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDashboard", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDashboard: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDataSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDataSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDataSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDataSetRefreshProperties(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDataSetRefreshProperties", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDataSetRefreshProperties: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDataSource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDataSource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDataSource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteDefaultQBusinessApplication(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteDefaultQBusinessApplication", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteDefaultQBusinessApplication: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFolder(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFolder", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFolder: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFolderMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteFolderMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteFolderMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteGroupMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteGroupMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteGroupMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteIAMPolicyAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteIAMPolicyAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteIAMPolicyAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteIdentityPropagationConfig(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteIdentityPropagationConfig", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteIdentityPropagationConfig: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteNamespace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteNamespace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteNamespace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRoleCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteRoleCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteRoleCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRoleMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteRoleMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteRoleMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTemplate(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTemplate", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTemplateAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTemplateAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTemplateAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTheme(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTheme", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTheme: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteThemeAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteThemeAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteThemeAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTopic(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTopic", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTopic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTopicRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTopicRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTopicRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteUserByPrincipalId(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteUserByPrincipalId", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUserByPrincipalId: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteUserCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteUserCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUserCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteVPCConnection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteVPCConnection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteVPCConnection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccountCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccountCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccountCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccountCustomization(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccountCustomization", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccountCustomization: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccountSettings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccountSettings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccountSettings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccountSubscription(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccountSubscription", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccountSubscription: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeActionConnector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeActionConnector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeActionConnector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeActionConnectorPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeActionConnectorPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeActionConnectorPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAnalysis(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAnalysis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAnalysis: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAnalysisDefinition(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAnalysisDefinition", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAnalysisDefinition: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAnalysisPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAnalysisPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAnalysisPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAssetBundleExportJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAssetBundleExportJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAssetBundleExportJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAssetBundleImportJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAssetBundleImportJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAssetBundleImportJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAutomationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAutomationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAutomationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeBrand(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeBrand", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeBrand: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeBrandAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeBrandAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeBrandAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeBrandPublishedVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeBrandPublishedVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeBrandPublishedVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCustomPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCustomPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCustomPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDashboard(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDashboard", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDashboard: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDashboardDefinition(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDashboardDefinition", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDashboardDefinition: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDashboardPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDashboardPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDashboardPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDashboardSnapshotJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDashboardSnapshotJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDashboardSnapshotJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDashboardSnapshotJobResult(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDashboardSnapshotJobResult", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDashboardSnapshotJobResult: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDashboardsQAConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDashboardsQAConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDashboardsQAConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDataSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDataSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDataSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDataSetPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDataSetPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDataSetPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDataSetRefreshProperties(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDataSetRefreshProperties", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDataSetRefreshProperties: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDataSource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDataSource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDataSource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDataSourcePermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDataSourcePermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDataSourcePermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeDefaultQBusinessApplication(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeDefaultQBusinessApplication", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDefaultQBusinessApplication: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFolder(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFolder", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFolder: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFolderPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFolderPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFolderPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeFolderResolvedPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeFolderResolvedPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeFolderResolvedPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeGroupMembership(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeGroupMembership", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeGroupMembership: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeIAMPolicyAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeIAMPolicyAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeIAMPolicyAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeIngestion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeIngestion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeIngestion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeIpRestriction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeIpRestriction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeIpRestriction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeKeyRegistration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeKeyRegistration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeKeyRegistration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeNamespace(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeNamespace", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeNamespace: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeQPersonalizationConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeQPersonalizationConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeQPersonalizationConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeQuickSightQSearchConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeQuickSightQSearchConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeQuickSightQSearchConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeRoleCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeRoleCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeRoleCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeSelfUpgradeConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeSelfUpgradeConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeSelfUpgradeConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTemplate(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTemplate", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTemplateAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTemplateAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTemplateAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTemplateDefinition(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTemplateDefinition", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTemplateDefinition: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTemplatePermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTemplatePermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTemplatePermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTheme(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTheme", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTheme: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeThemeAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeThemeAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeThemeAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeThemePermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeThemePermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeThemePermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTopic(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTopic", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTopic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTopicPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTopicPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTopicPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTopicRefresh(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTopicRefresh", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTopicRefresh: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTopicRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTopicRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTopicRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeVPCConnection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeVPCConnection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeVPCConnection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGenerateEmbedUrlForAnonymousUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GenerateEmbedUrlForAnonymousUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GenerateEmbedUrlForAnonymousUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGenerateEmbedUrlForRegisteredUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GenerateEmbedUrlForRegisteredUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GenerateEmbedUrlForRegisteredUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGenerateEmbedUrlForRegisteredUserWithIdentity(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GenerateEmbedUrlForRegisteredUserWithIdentity", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GenerateEmbedUrlForRegisteredUserWithIdentity: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetDashboardEmbedUrl(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetDashboardEmbedUrl", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetDashboardEmbedUrl: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFlowMetadata(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFlowMetadata", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFlowMetadata: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetFlowPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetFlowPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetFlowPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetIdentityContext(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetIdentityContext", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIdentityContext: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSessionEmbedUrl(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSessionEmbedUrl", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSessionEmbedUrl: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListActionConnectors(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListActionConnectors", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListActionConnectors: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAnalyses(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAnalyses", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAnalyses: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAssetBundleExportJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAssetBundleExportJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAssetBundleExportJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAssetBundleImportJobs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAssetBundleImportJobs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAssetBundleImportJobs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListBrands(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListBrands", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListBrands: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCustomPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCustomPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCustomPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDashboardVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDashboardVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDashboardVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDashboards(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDashboards", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDashboards: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDataSets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDataSets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDataSets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListDataSources(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListDataSources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListDataSources: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFlows(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFlows", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFlows: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFolderMembers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFolderMembers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFolderMembers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFolders(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFolders", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFolders: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListFoldersForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListFoldersForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListFoldersForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListGroupMemberships(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListGroupMemberships", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListGroupMemberships: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListIAMPolicyAssignments(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListIAMPolicyAssignments", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIAMPolicyAssignments: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListIAMPolicyAssignmentsForUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListIAMPolicyAssignmentsForUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIAMPolicyAssignmentsForUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListIdentityPropagationConfigs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListIdentityPropagationConfigs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIdentityPropagationConfigs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListIngestions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListIngestions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListIngestions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListNamespaces(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListNamespaces", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListNamespaces: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListRefreshSchedules(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListRefreshSchedules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRefreshSchedules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListRoleMemberships(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListRoleMemberships", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRoleMemberships: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListSelfUpgrades(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListSelfUpgrades", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListSelfUpgrades: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestListTemplateAliases(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTemplateAliases", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTemplateAliases: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTemplateVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTemplateVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTemplateVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTemplates(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTemplates", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTemplates: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListThemeAliases(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListThemeAliases", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListThemeAliases: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListThemeVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListThemeVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListThemeVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListThemes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListThemes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListThemes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTopicRefreshSchedules(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTopicRefreshSchedules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTopicRefreshSchedules: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTopicReviewedAnswers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTopicReviewedAnswers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTopicReviewedAnswers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTopics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTopics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTopics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListUserGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListUserGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListUserGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListUsers(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListUsers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListUsers: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListVPCConnections(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListVPCConnections", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListVPCConnections: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPredictQAResults(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PredictQAResults", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PredictQAResults: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutDataSetRefreshProperties(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutDataSetRefreshProperties", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutDataSetRefreshProperties: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRegisterUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RegisterUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RegisterUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRestoreAnalysis(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RestoreAnalysis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RestoreAnalysis: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchActionConnectors(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchActionConnectors", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchActionConnectors: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchAnalyses(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchAnalyses", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchAnalyses: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchDashboards(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchDashboards", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchDashboards: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchDataSets(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchDataSets", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchDataSets: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchDataSources(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchDataSources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchDataSources: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchFlows(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchFlows", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchFlows: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchFolders(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchFolders", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchFolders: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchTopics(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchTopics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchTopics: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartAssetBundleExportJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartAssetBundleExportJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartAssetBundleExportJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartAssetBundleImportJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartAssetBundleImportJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartAssetBundleImportJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartAutomationJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartAutomationJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartAutomationJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartDashboardSnapshotJob(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartDashboardSnapshotJob", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartDashboardSnapshotJob: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartDashboardSnapshotJobSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartDashboardSnapshotJobSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartDashboardSnapshotJobSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
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

func TestUpdateAccountCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAccountCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAccountCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAccountCustomization(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAccountCustomization", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAccountCustomization: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAccountSettings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAccountSettings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAccountSettings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateActionConnector(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateActionConnector", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateActionConnector: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateActionConnectorPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateActionConnectorPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateActionConnectorPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAnalysis(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAnalysis", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAnalysis: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAnalysisPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAnalysisPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAnalysisPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateApplicationWithTokenExchangeGrant(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateApplicationWithTokenExchangeGrant", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateApplicationWithTokenExchangeGrant: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateBrand(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateBrand", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateBrand: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateBrandAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateBrandAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateBrandAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateBrandPublishedVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateBrandPublishedVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateBrandPublishedVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCustomPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCustomPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCustomPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDashboard(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDashboard", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDashboard: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDashboardLinks(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDashboardLinks", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDashboardLinks: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDashboardPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDashboardPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDashboardPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDashboardPublishedVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDashboardPublishedVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDashboardPublishedVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDashboardsQAConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDashboardsQAConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDashboardsQAConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDataSet(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDataSet", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDataSet: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDataSetPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDataSetPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDataSetPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDataSource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDataSource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDataSource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDataSourcePermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDataSourcePermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDataSourcePermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateDefaultQBusinessApplication(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateDefaultQBusinessApplication", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateDefaultQBusinessApplication: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFlowPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFlowPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFlowPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFolder(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFolder", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFolder: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFolderPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateFolderPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFolderPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateIAMPolicyAssignment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateIAMPolicyAssignment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateIAMPolicyAssignment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateIdentityPropagationConfig(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateIdentityPropagationConfig", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateIdentityPropagationConfig: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateIpRestriction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateIpRestriction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateIpRestriction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateKeyRegistration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateKeyRegistration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateKeyRegistration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePublicSharingSettings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdatePublicSharingSettings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePublicSharingSettings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateQPersonalizationConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateQPersonalizationConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateQPersonalizationConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateQuickSightQSearchConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateQuickSightQSearchConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateQuickSightQSearchConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateRoleCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateRoleCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateRoleCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSPICECapacityConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateSPICECapacityConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateSPICECapacityConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSelfUpgrade(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateSelfUpgrade", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateSelfUpgrade: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSelfUpgradeConfiguration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateSelfUpgradeConfiguration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateSelfUpgradeConfiguration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTemplate(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTemplate", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTemplate: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTemplateAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTemplateAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTemplateAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTemplatePermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTemplatePermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTemplatePermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTheme(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTheme", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTheme: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateThemeAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateThemeAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateThemeAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateThemePermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateThemePermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateThemePermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTopic(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTopic", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTopic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTopicPermissions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTopicPermissions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTopicPermissions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTopicRefreshSchedule(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTopicRefreshSchedule", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTopicRefreshSchedule: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateUser(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateUser", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateUser: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateUserCustomPermission(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateUserCustomPermission", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateUserCustomPermission: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateVPCConnection(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateVPCConnection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateVPCConnection: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

