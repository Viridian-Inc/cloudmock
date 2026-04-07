package servicecatalog_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/servicecatalog"
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
	req.Header.Set("X-Amz-Target", "AWS242ServiceCatalogService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/servicecatalog/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestAcceptPortfolioShare(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AcceptPortfolioShare", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AcceptPortfolioShare: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAssociateBudgetWithResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociateBudgetWithResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociateBudgetWithResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAssociatePrincipalWithPortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociatePrincipalWithPortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociatePrincipalWithPortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAssociateProductWithPortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociateProductWithPortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociateProductWithPortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAssociateServiceActionWithProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociateServiceActionWithProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociateServiceActionWithProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAssociateTagOptionWithResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AssociateTagOptionWithResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AssociateTagOptionWithResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchAssociateServiceActionWithProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchAssociateServiceActionWithProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchAssociateServiceActionWithProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestBatchDisassociateServiceActionFromProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "BatchDisassociateServiceActionFromProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("BatchDisassociateServiceActionFromProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCopyProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CopyProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CopyProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateConstraint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateConstraint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateConstraint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreatePortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreatePortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreatePortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreatePortfolioShare(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreatePortfolioShare", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreatePortfolioShare: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateProvisionedProductPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateProvisionedProductPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateProvisionedProductPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateServiceAction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateServiceAction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateServiceAction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateTagOption(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateTagOption", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTagOption: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteConstraint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteConstraint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteConstraint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeletePortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeletePortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeletePortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeletePortfolioShare(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeletePortfolioShare", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeletePortfolioShare: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProvisionedProductPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteProvisionedProductPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteProvisionedProductPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteServiceAction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteServiceAction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteServiceAction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTagOption(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteTagOption", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteTagOption: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeConstraint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeConstraint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeConstraint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCopyProductStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCopyProductStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCopyProductStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribePortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribePortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribePortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribePortfolioShareStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribePortfolioShareStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribePortfolioShareStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribePortfolioShares(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribePortfolioShares", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribePortfolioShares: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProductAsAdmin(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProductAsAdmin", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProductAsAdmin: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProductView(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProductView", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProductView: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProvisionedProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProvisionedProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProvisionedProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProvisionedProductPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProvisionedProductPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProvisionedProductPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeProvisioningParameters(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeProvisioningParameters", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeProvisioningParameters: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeRecord(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeRecord", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeRecord: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeServiceAction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeServiceAction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeServiceAction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeServiceActionExecutionParameters(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeServiceActionExecutionParameters", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeServiceActionExecutionParameters: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeTagOption(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeTagOption", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeTagOption: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisableAWSOrganizationsAccess(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisableAWSOrganizationsAccess", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisableAWSOrganizationsAccess: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateBudgetFromResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateBudgetFromResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateBudgetFromResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociatePrincipalFromPortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociatePrincipalFromPortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociatePrincipalFromPortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateProductFromPortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateProductFromPortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateProductFromPortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateServiceActionFromProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateServiceActionFromProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateServiceActionFromProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDisassociateTagOptionFromResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DisassociateTagOptionFromResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DisassociateTagOptionFromResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestEnableAWSOrganizationsAccess(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "EnableAWSOrganizationsAccess", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("EnableAWSOrganizationsAccess: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestExecuteProvisionedProductPlan(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ExecuteProvisionedProductPlan", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteProvisionedProductPlan: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestExecuteProvisionedProductServiceAction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ExecuteProvisionedProductServiceAction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteProvisionedProductServiceAction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetAWSOrganizationsAccessStatus(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetAWSOrganizationsAccessStatus", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetAWSOrganizationsAccessStatus: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetProvisionedProductOutputs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetProvisionedProductOutputs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetProvisionedProductOutputs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestImportAsProvisionedProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ImportAsProvisionedProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ImportAsProvisionedProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAcceptedPortfolioShares(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAcceptedPortfolioShares", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAcceptedPortfolioShares: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListBudgetsForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListBudgetsForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListBudgetsForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListConstraintsForPortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListConstraintsForPortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListConstraintsForPortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListLaunchPaths(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListLaunchPaths", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListLaunchPaths: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListOrganizationPortfolioAccess(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListOrganizationPortfolioAccess", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListOrganizationPortfolioAccess: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPortfolioAccess(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListPortfolioAccess", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListPortfolioAccess: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPortfolios(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListPortfolios", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListPortfolios: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPortfoliosForProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListPortfoliosForProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListPortfoliosForProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPrincipalsForPortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListPrincipalsForPortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListPrincipalsForPortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListProvisionedProductPlans(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListProvisionedProductPlans", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListProvisionedProductPlans: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListProvisioningArtifacts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListProvisioningArtifacts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListProvisioningArtifacts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListProvisioningArtifactsForServiceAction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListProvisioningArtifactsForServiceAction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListProvisioningArtifactsForServiceAction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListRecordHistory(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListRecordHistory", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListRecordHistory: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListResourcesForTagOption(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListResourcesForTagOption", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListResourcesForTagOption: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListServiceActions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListServiceActions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListServiceActions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListServiceActionsForProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListServiceActionsForProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListServiceActionsForProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListStackInstancesForProvisionedProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListStackInstancesForProvisionedProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListStackInstancesForProvisionedProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTagOptions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTagOptions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTagOptions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestNotifyProvisionProductEngineWorkflowResult(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "NotifyProvisionProductEngineWorkflowResult", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("NotifyProvisionProductEngineWorkflowResult: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestNotifyTerminateProvisionedProductEngineWorkflowResult(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "NotifyTerminateProvisionedProductEngineWorkflowResult", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("NotifyTerminateProvisionedProductEngineWorkflowResult: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestNotifyUpdateProvisionedProductEngineWorkflowResult(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "NotifyUpdateProvisionedProductEngineWorkflowResult", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("NotifyUpdateProvisionedProductEngineWorkflowResult: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestProvisionProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ProvisionProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ProvisionProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRejectPortfolioShare(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RejectPortfolioShare", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RejectPortfolioShare: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestScanProvisionedProducts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ScanProvisionedProducts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ScanProvisionedProducts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchProducts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchProducts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchProducts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchProductsAsAdmin(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchProductsAsAdmin", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchProductsAsAdmin: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSearchProvisionedProducts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "SearchProvisionedProducts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("SearchProvisionedProducts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTerminateProvisionedProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TerminateProvisionedProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TerminateProvisionedProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateConstraint(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateConstraint", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateConstraint: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePortfolio(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdatePortfolio", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePortfolio: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePortfolioShare(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdatePortfolioShare", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePortfolioShare: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProvisionedProduct(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateProvisionedProduct", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProvisionedProduct: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProvisionedProductProperties(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateProvisionedProductProperties", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProvisionedProductProperties: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateProvisioningArtifact(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateProvisioningArtifact", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProvisioningArtifact: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateServiceAction(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateServiceAction", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateServiceAction: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTagOption(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateTagOption", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateTagOption: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

