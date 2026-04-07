package servicecatalog

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS servicecatalog service.
type Service struct {
	store *Store
}

// New returns a new servicecatalog Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "servicecatalog" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "AcceptPortfolioShare", Method: http.MethodPost, IAMAction: "servicecatalog:AcceptPortfolioShare"},
		{Name: "AssociateBudgetWithResource", Method: http.MethodPost, IAMAction: "servicecatalog:AssociateBudgetWithResource"},
		{Name: "AssociatePrincipalWithPortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:AssociatePrincipalWithPortfolio"},
		{Name: "AssociateProductWithPortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:AssociateProductWithPortfolio"},
		{Name: "AssociateServiceActionWithProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:AssociateServiceActionWithProvisioningArtifact"},
		{Name: "AssociateTagOptionWithResource", Method: http.MethodPost, IAMAction: "servicecatalog:AssociateTagOptionWithResource"},
		{Name: "BatchAssociateServiceActionWithProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:BatchAssociateServiceActionWithProvisioningArtifact"},
		{Name: "BatchDisassociateServiceActionFromProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:BatchDisassociateServiceActionFromProvisioningArtifact"},
		{Name: "CopyProduct", Method: http.MethodPost, IAMAction: "servicecatalog:CopyProduct"},
		{Name: "CreateConstraint", Method: http.MethodPost, IAMAction: "servicecatalog:CreateConstraint"},
		{Name: "CreatePortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:CreatePortfolio"},
		{Name: "CreatePortfolioShare", Method: http.MethodPost, IAMAction: "servicecatalog:CreatePortfolioShare"},
		{Name: "CreateProduct", Method: http.MethodPost, IAMAction: "servicecatalog:CreateProduct"},
		{Name: "CreateProvisionedProductPlan", Method: http.MethodPost, IAMAction: "servicecatalog:CreateProvisionedProductPlan"},
		{Name: "CreateProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:CreateProvisioningArtifact"},
		{Name: "CreateServiceAction", Method: http.MethodPost, IAMAction: "servicecatalog:CreateServiceAction"},
		{Name: "CreateTagOption", Method: http.MethodPost, IAMAction: "servicecatalog:CreateTagOption"},
		{Name: "DeleteConstraint", Method: http.MethodPost, IAMAction: "servicecatalog:DeleteConstraint"},
		{Name: "DeletePortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:DeletePortfolio"},
		{Name: "DeletePortfolioShare", Method: http.MethodPost, IAMAction: "servicecatalog:DeletePortfolioShare"},
		{Name: "DeleteProduct", Method: http.MethodPost, IAMAction: "servicecatalog:DeleteProduct"},
		{Name: "DeleteProvisionedProductPlan", Method: http.MethodPost, IAMAction: "servicecatalog:DeleteProvisionedProductPlan"},
		{Name: "DeleteProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:DeleteProvisioningArtifact"},
		{Name: "DeleteServiceAction", Method: http.MethodPost, IAMAction: "servicecatalog:DeleteServiceAction"},
		{Name: "DeleteTagOption", Method: http.MethodPost, IAMAction: "servicecatalog:DeleteTagOption"},
		{Name: "DescribeConstraint", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeConstraint"},
		{Name: "DescribeCopyProductStatus", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeCopyProductStatus"},
		{Name: "DescribePortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:DescribePortfolio"},
		{Name: "DescribePortfolioShareStatus", Method: http.MethodPost, IAMAction: "servicecatalog:DescribePortfolioShareStatus"},
		{Name: "DescribePortfolioShares", Method: http.MethodPost, IAMAction: "servicecatalog:DescribePortfolioShares"},
		{Name: "DescribeProduct", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProduct"},
		{Name: "DescribeProductAsAdmin", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProductAsAdmin"},
		{Name: "DescribeProductView", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProductView"},
		{Name: "DescribeProvisionedProduct", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProvisionedProduct"},
		{Name: "DescribeProvisionedProductPlan", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProvisionedProductPlan"},
		{Name: "DescribeProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProvisioningArtifact"},
		{Name: "DescribeProvisioningParameters", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeProvisioningParameters"},
		{Name: "DescribeRecord", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeRecord"},
		{Name: "DescribeServiceAction", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeServiceAction"},
		{Name: "DescribeServiceActionExecutionParameters", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeServiceActionExecutionParameters"},
		{Name: "DescribeTagOption", Method: http.MethodPost, IAMAction: "servicecatalog:DescribeTagOption"},
		{Name: "DisableAWSOrganizationsAccess", Method: http.MethodPost, IAMAction: "servicecatalog:DisableAWSOrganizationsAccess"},
		{Name: "DisassociateBudgetFromResource", Method: http.MethodPost, IAMAction: "servicecatalog:DisassociateBudgetFromResource"},
		{Name: "DisassociatePrincipalFromPortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:DisassociatePrincipalFromPortfolio"},
		{Name: "DisassociateProductFromPortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:DisassociateProductFromPortfolio"},
		{Name: "DisassociateServiceActionFromProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:DisassociateServiceActionFromProvisioningArtifact"},
		{Name: "DisassociateTagOptionFromResource", Method: http.MethodPost, IAMAction: "servicecatalog:DisassociateTagOptionFromResource"},
		{Name: "EnableAWSOrganizationsAccess", Method: http.MethodPost, IAMAction: "servicecatalog:EnableAWSOrganizationsAccess"},
		{Name: "ExecuteProvisionedProductPlan", Method: http.MethodPost, IAMAction: "servicecatalog:ExecuteProvisionedProductPlan"},
		{Name: "ExecuteProvisionedProductServiceAction", Method: http.MethodPost, IAMAction: "servicecatalog:ExecuteProvisionedProductServiceAction"},
		{Name: "GetAWSOrganizationsAccessStatus", Method: http.MethodPost, IAMAction: "servicecatalog:GetAWSOrganizationsAccessStatus"},
		{Name: "GetProvisionedProductOutputs", Method: http.MethodPost, IAMAction: "servicecatalog:GetProvisionedProductOutputs"},
		{Name: "ImportAsProvisionedProduct", Method: http.MethodPost, IAMAction: "servicecatalog:ImportAsProvisionedProduct"},
		{Name: "ListAcceptedPortfolioShares", Method: http.MethodPost, IAMAction: "servicecatalog:ListAcceptedPortfolioShares"},
		{Name: "ListBudgetsForResource", Method: http.MethodPost, IAMAction: "servicecatalog:ListBudgetsForResource"},
		{Name: "ListConstraintsForPortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:ListConstraintsForPortfolio"},
		{Name: "ListLaunchPaths", Method: http.MethodPost, IAMAction: "servicecatalog:ListLaunchPaths"},
		{Name: "ListOrganizationPortfolioAccess", Method: http.MethodPost, IAMAction: "servicecatalog:ListOrganizationPortfolioAccess"},
		{Name: "ListPortfolioAccess", Method: http.MethodPost, IAMAction: "servicecatalog:ListPortfolioAccess"},
		{Name: "ListPortfolios", Method: http.MethodPost, IAMAction: "servicecatalog:ListPortfolios"},
		{Name: "ListPortfoliosForProduct", Method: http.MethodPost, IAMAction: "servicecatalog:ListPortfoliosForProduct"},
		{Name: "ListPrincipalsForPortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:ListPrincipalsForPortfolio"},
		{Name: "ListProvisionedProductPlans", Method: http.MethodPost, IAMAction: "servicecatalog:ListProvisionedProductPlans"},
		{Name: "ListProvisioningArtifacts", Method: http.MethodPost, IAMAction: "servicecatalog:ListProvisioningArtifacts"},
		{Name: "ListProvisioningArtifactsForServiceAction", Method: http.MethodPost, IAMAction: "servicecatalog:ListProvisioningArtifactsForServiceAction"},
		{Name: "ListRecordHistory", Method: http.MethodPost, IAMAction: "servicecatalog:ListRecordHistory"},
		{Name: "ListResourcesForTagOption", Method: http.MethodPost, IAMAction: "servicecatalog:ListResourcesForTagOption"},
		{Name: "ListServiceActions", Method: http.MethodPost, IAMAction: "servicecatalog:ListServiceActions"},
		{Name: "ListServiceActionsForProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:ListServiceActionsForProvisioningArtifact"},
		{Name: "ListStackInstancesForProvisionedProduct", Method: http.MethodPost, IAMAction: "servicecatalog:ListStackInstancesForProvisionedProduct"},
		{Name: "ListTagOptions", Method: http.MethodPost, IAMAction: "servicecatalog:ListTagOptions"},
		{Name: "NotifyProvisionProductEngineWorkflowResult", Method: http.MethodPost, IAMAction: "servicecatalog:NotifyProvisionProductEngineWorkflowResult"},
		{Name: "NotifyTerminateProvisionedProductEngineWorkflowResult", Method: http.MethodPost, IAMAction: "servicecatalog:NotifyTerminateProvisionedProductEngineWorkflowResult"},
		{Name: "NotifyUpdateProvisionedProductEngineWorkflowResult", Method: http.MethodPost, IAMAction: "servicecatalog:NotifyUpdateProvisionedProductEngineWorkflowResult"},
		{Name: "ProvisionProduct", Method: http.MethodPost, IAMAction: "servicecatalog:ProvisionProduct"},
		{Name: "RejectPortfolioShare", Method: http.MethodPost, IAMAction: "servicecatalog:RejectPortfolioShare"},
		{Name: "ScanProvisionedProducts", Method: http.MethodPost, IAMAction: "servicecatalog:ScanProvisionedProducts"},
		{Name: "SearchProducts", Method: http.MethodPost, IAMAction: "servicecatalog:SearchProducts"},
		{Name: "SearchProductsAsAdmin", Method: http.MethodPost, IAMAction: "servicecatalog:SearchProductsAsAdmin"},
		{Name: "SearchProvisionedProducts", Method: http.MethodPost, IAMAction: "servicecatalog:SearchProvisionedProducts"},
		{Name: "TerminateProvisionedProduct", Method: http.MethodPost, IAMAction: "servicecatalog:TerminateProvisionedProduct"},
		{Name: "UpdateConstraint", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateConstraint"},
		{Name: "UpdatePortfolio", Method: http.MethodPost, IAMAction: "servicecatalog:UpdatePortfolio"},
		{Name: "UpdatePortfolioShare", Method: http.MethodPost, IAMAction: "servicecatalog:UpdatePortfolioShare"},
		{Name: "UpdateProduct", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateProduct"},
		{Name: "UpdateProvisionedProduct", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateProvisionedProduct"},
		{Name: "UpdateProvisionedProductProperties", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateProvisionedProductProperties"},
		{Name: "UpdateProvisioningArtifact", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateProvisioningArtifact"},
		{Name: "UpdateServiceAction", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateServiceAction"},
		{Name: "UpdateTagOption", Method: http.MethodPost, IAMAction: "servicecatalog:UpdateTagOption"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "AcceptPortfolioShare":
		return handleAcceptPortfolioShare(ctx, s.store)
	case "AssociateBudgetWithResource":
		return handleAssociateBudgetWithResource(ctx, s.store)
	case "AssociatePrincipalWithPortfolio":
		return handleAssociatePrincipalWithPortfolio(ctx, s.store)
	case "AssociateProductWithPortfolio":
		return handleAssociateProductWithPortfolio(ctx, s.store)
	case "AssociateServiceActionWithProvisioningArtifact":
		return handleAssociateServiceActionWithProvisioningArtifact(ctx, s.store)
	case "AssociateTagOptionWithResource":
		return handleAssociateTagOptionWithResource(ctx, s.store)
	case "BatchAssociateServiceActionWithProvisioningArtifact":
		return handleBatchAssociateServiceActionWithProvisioningArtifact(ctx, s.store)
	case "BatchDisassociateServiceActionFromProvisioningArtifact":
		return handleBatchDisassociateServiceActionFromProvisioningArtifact(ctx, s.store)
	case "CopyProduct":
		return handleCopyProduct(ctx, s.store)
	case "CreateConstraint":
		return handleCreateConstraint(ctx, s.store)
	case "CreatePortfolio":
		return handleCreatePortfolio(ctx, s.store)
	case "CreatePortfolioShare":
		return handleCreatePortfolioShare(ctx, s.store)
	case "CreateProduct":
		return handleCreateProduct(ctx, s.store)
	case "CreateProvisionedProductPlan":
		return handleCreateProvisionedProductPlan(ctx, s.store)
	case "CreateProvisioningArtifact":
		return handleCreateProvisioningArtifact(ctx, s.store)
	case "CreateServiceAction":
		return handleCreateServiceAction(ctx, s.store)
	case "CreateTagOption":
		return handleCreateTagOption(ctx, s.store)
	case "DeleteConstraint":
		return handleDeleteConstraint(ctx, s.store)
	case "DeletePortfolio":
		return handleDeletePortfolio(ctx, s.store)
	case "DeletePortfolioShare":
		return handleDeletePortfolioShare(ctx, s.store)
	case "DeleteProduct":
		return handleDeleteProduct(ctx, s.store)
	case "DeleteProvisionedProductPlan":
		return handleDeleteProvisionedProductPlan(ctx, s.store)
	case "DeleteProvisioningArtifact":
		return handleDeleteProvisioningArtifact(ctx, s.store)
	case "DeleteServiceAction":
		return handleDeleteServiceAction(ctx, s.store)
	case "DeleteTagOption":
		return handleDeleteTagOption(ctx, s.store)
	case "DescribeConstraint":
		return handleDescribeConstraint(ctx, s.store)
	case "DescribeCopyProductStatus":
		return handleDescribeCopyProductStatus(ctx, s.store)
	case "DescribePortfolio":
		return handleDescribePortfolio(ctx, s.store)
	case "DescribePortfolioShareStatus":
		return handleDescribePortfolioShareStatus(ctx, s.store)
	case "DescribePortfolioShares":
		return handleDescribePortfolioShares(ctx, s.store)
	case "DescribeProduct":
		return handleDescribeProduct(ctx, s.store)
	case "DescribeProductAsAdmin":
		return handleDescribeProductAsAdmin(ctx, s.store)
	case "DescribeProductView":
		return handleDescribeProductView(ctx, s.store)
	case "DescribeProvisionedProduct":
		return handleDescribeProvisionedProduct(ctx, s.store)
	case "DescribeProvisionedProductPlan":
		return handleDescribeProvisionedProductPlan(ctx, s.store)
	case "DescribeProvisioningArtifact":
		return handleDescribeProvisioningArtifact(ctx, s.store)
	case "DescribeProvisioningParameters":
		return handleDescribeProvisioningParameters(ctx, s.store)
	case "DescribeRecord":
		return handleDescribeRecord(ctx, s.store)
	case "DescribeServiceAction":
		return handleDescribeServiceAction(ctx, s.store)
	case "DescribeServiceActionExecutionParameters":
		return handleDescribeServiceActionExecutionParameters(ctx, s.store)
	case "DescribeTagOption":
		return handleDescribeTagOption(ctx, s.store)
	case "DisableAWSOrganizationsAccess":
		return handleDisableAWSOrganizationsAccess(ctx, s.store)
	case "DisassociateBudgetFromResource":
		return handleDisassociateBudgetFromResource(ctx, s.store)
	case "DisassociatePrincipalFromPortfolio":
		return handleDisassociatePrincipalFromPortfolio(ctx, s.store)
	case "DisassociateProductFromPortfolio":
		return handleDisassociateProductFromPortfolio(ctx, s.store)
	case "DisassociateServiceActionFromProvisioningArtifact":
		return handleDisassociateServiceActionFromProvisioningArtifact(ctx, s.store)
	case "DisassociateTagOptionFromResource":
		return handleDisassociateTagOptionFromResource(ctx, s.store)
	case "EnableAWSOrganizationsAccess":
		return handleEnableAWSOrganizationsAccess(ctx, s.store)
	case "ExecuteProvisionedProductPlan":
		return handleExecuteProvisionedProductPlan(ctx, s.store)
	case "ExecuteProvisionedProductServiceAction":
		return handleExecuteProvisionedProductServiceAction(ctx, s.store)
	case "GetAWSOrganizationsAccessStatus":
		return handleGetAWSOrganizationsAccessStatus(ctx, s.store)
	case "GetProvisionedProductOutputs":
		return handleGetProvisionedProductOutputs(ctx, s.store)
	case "ImportAsProvisionedProduct":
		return handleImportAsProvisionedProduct(ctx, s.store)
	case "ListAcceptedPortfolioShares":
		return handleListAcceptedPortfolioShares(ctx, s.store)
	case "ListBudgetsForResource":
		return handleListBudgetsForResource(ctx, s.store)
	case "ListConstraintsForPortfolio":
		return handleListConstraintsForPortfolio(ctx, s.store)
	case "ListLaunchPaths":
		return handleListLaunchPaths(ctx, s.store)
	case "ListOrganizationPortfolioAccess":
		return handleListOrganizationPortfolioAccess(ctx, s.store)
	case "ListPortfolioAccess":
		return handleListPortfolioAccess(ctx, s.store)
	case "ListPortfolios":
		return handleListPortfolios(ctx, s.store)
	case "ListPortfoliosForProduct":
		return handleListPortfoliosForProduct(ctx, s.store)
	case "ListPrincipalsForPortfolio":
		return handleListPrincipalsForPortfolio(ctx, s.store)
	case "ListProvisionedProductPlans":
		return handleListProvisionedProductPlans(ctx, s.store)
	case "ListProvisioningArtifacts":
		return handleListProvisioningArtifacts(ctx, s.store)
	case "ListProvisioningArtifactsForServiceAction":
		return handleListProvisioningArtifactsForServiceAction(ctx, s.store)
	case "ListRecordHistory":
		return handleListRecordHistory(ctx, s.store)
	case "ListResourcesForTagOption":
		return handleListResourcesForTagOption(ctx, s.store)
	case "ListServiceActions":
		return handleListServiceActions(ctx, s.store)
	case "ListServiceActionsForProvisioningArtifact":
		return handleListServiceActionsForProvisioningArtifact(ctx, s.store)
	case "ListStackInstancesForProvisionedProduct":
		return handleListStackInstancesForProvisionedProduct(ctx, s.store)
	case "ListTagOptions":
		return handleListTagOptions(ctx, s.store)
	case "NotifyProvisionProductEngineWorkflowResult":
		return handleNotifyProvisionProductEngineWorkflowResult(ctx, s.store)
	case "NotifyTerminateProvisionedProductEngineWorkflowResult":
		return handleNotifyTerminateProvisionedProductEngineWorkflowResult(ctx, s.store)
	case "NotifyUpdateProvisionedProductEngineWorkflowResult":
		return handleNotifyUpdateProvisionedProductEngineWorkflowResult(ctx, s.store)
	case "ProvisionProduct":
		return handleProvisionProduct(ctx, s.store)
	case "RejectPortfolioShare":
		return handleRejectPortfolioShare(ctx, s.store)
	case "ScanProvisionedProducts":
		return handleScanProvisionedProducts(ctx, s.store)
	case "SearchProducts":
		return handleSearchProducts(ctx, s.store)
	case "SearchProductsAsAdmin":
		return handleSearchProductsAsAdmin(ctx, s.store)
	case "SearchProvisionedProducts":
		return handleSearchProvisionedProducts(ctx, s.store)
	case "TerminateProvisionedProduct":
		return handleTerminateProvisionedProduct(ctx, s.store)
	case "UpdateConstraint":
		return handleUpdateConstraint(ctx, s.store)
	case "UpdatePortfolio":
		return handleUpdatePortfolio(ctx, s.store)
	case "UpdatePortfolioShare":
		return handleUpdatePortfolioShare(ctx, s.store)
	case "UpdateProduct":
		return handleUpdateProduct(ctx, s.store)
	case "UpdateProvisionedProduct":
		return handleUpdateProvisionedProduct(ctx, s.store)
	case "UpdateProvisionedProductProperties":
		return handleUpdateProvisionedProductProperties(ctx, s.store)
	case "UpdateProvisioningArtifact":
		return handleUpdateProvisioningArtifact(ctx, s.store)
	case "UpdateServiceAction":
		return handleUpdateServiceAction(ctx, s.store)
	case "UpdateTagOption":
		return handleUpdateTagOption(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
