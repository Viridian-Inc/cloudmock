package servicecatalog

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AcceptPortfolioShareInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	PortfolioShareType *string `json:"PortfolioShareType,omitempty"`
}

type AcceptPortfolioShareOutput struct {
}

type AccessLevelFilter struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AssociateBudgetWithResourceInput struct {
	BudgetName string `json:"BudgetName,omitempty"`
	ResourceId string `json:"ResourceId,omitempty"`
}

type AssociateBudgetWithResourceOutput struct {
}

type AssociatePrincipalWithPortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	PrincipalARN string `json:"PrincipalARN,omitempty"`
	PrincipalType string `json:"PrincipalType,omitempty"`
}

type AssociatePrincipalWithPortfolioOutput struct {
}

type AssociateProductWithPortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	SourcePortfolioId *string `json:"SourcePortfolioId,omitempty"`
}

type AssociateProductWithPortfolioOutput struct {
}

type AssociateServiceActionWithProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken *string `json:"IdempotencyToken,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
	ServiceActionId string `json:"ServiceActionId,omitempty"`
}

type AssociateServiceActionWithProvisioningArtifactOutput struct {
}

type AssociateTagOptionWithResourceInput struct {
	ResourceId string `json:"ResourceId,omitempty"`
	TagOptionId string `json:"TagOptionId,omitempty"`
}

type AssociateTagOptionWithResourceOutput struct {
}

type BatchAssociateServiceActionWithProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	ServiceActionAssociations []ServiceActionAssociation `json:"ServiceActionAssociations,omitempty"`
}

type BatchAssociateServiceActionWithProvisioningArtifactOutput struct {
	FailedServiceActionAssociations []FailedServiceActionAssociation `json:"FailedServiceActionAssociations,omitempty"`
}

type BatchDisassociateServiceActionFromProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	ServiceActionAssociations []ServiceActionAssociation `json:"ServiceActionAssociations,omitempty"`
}

type BatchDisassociateServiceActionFromProvisioningArtifactOutput struct {
	FailedServiceActionAssociations []FailedServiceActionAssociation `json:"FailedServiceActionAssociations,omitempty"`
}

type BudgetDetail struct {
	BudgetName *string `json:"BudgetName,omitempty"`
}

type CloudWatchDashboard struct {
	Name *string `json:"Name,omitempty"`
}

type CodeStarParameters struct {
	ArtifactPath string `json:"ArtifactPath,omitempty"`
	Branch string `json:"Branch,omitempty"`
	ConnectionArn string `json:"ConnectionArn,omitempty"`
	Repository string `json:"Repository,omitempty"`
}

type ConstraintDetail struct {
	ConstraintId *string `json:"ConstraintId,omitempty"`
	Description *string `json:"Description,omitempty"`
	Owner *string `json:"Owner,omitempty"`
	PortfolioId *string `json:"PortfolioId,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ConstraintSummary struct {
	Description *string `json:"Description,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type CopyProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	CopyOptions []string `json:"CopyOptions,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	SourceProductArn string `json:"SourceProductArn,omitempty"`
	SourceProvisioningArtifactIdentifiers []map[string]string `json:"SourceProvisioningArtifactIdentifiers,omitempty"`
	TargetProductId *string `json:"TargetProductId,omitempty"`
	TargetProductName *string `json:"TargetProductName,omitempty"`
}

type CopyProductOutput struct {
	CopyProductToken *string `json:"CopyProductToken,omitempty"`
}

type CreateConstraintInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Description *string `json:"Description,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Parameters string `json:"Parameters,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	Type string `json:"Type,omitempty"`
}

type CreateConstraintOutput struct {
	ConstraintDetail *ConstraintDetail `json:"ConstraintDetail,omitempty"`
	ConstraintParameters *string `json:"ConstraintParameters,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type CreatePortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Description *string `json:"Description,omitempty"`
	DisplayName string `json:"DisplayName,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	ProviderName string `json:"ProviderName,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreatePortfolioOutput struct {
	PortfolioDetail *PortfolioDetail `json:"PortfolioDetail,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreatePortfolioShareInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccountId *string `json:"AccountId,omitempty"`
	OrganizationNode *OrganizationNode `json:"OrganizationNode,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	SharePrincipals bool `json:"SharePrincipals,omitempty"`
	ShareTagOptions bool `json:"ShareTagOptions,omitempty"`
}

type CreatePortfolioShareOutput struct {
	PortfolioShareToken *string `json:"PortfolioShareToken,omitempty"`
}

type CreateProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Description *string `json:"Description,omitempty"`
	Distributor *string `json:"Distributor,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Name string `json:"Name,omitempty"`
	Owner string `json:"Owner,omitempty"`
	ProductType string `json:"ProductType,omitempty"`
	ProvisioningArtifactParameters *ProvisioningArtifactProperties `json:"ProvisioningArtifactParameters,omitempty"`
	SourceConnection *SourceConnection `json:"SourceConnection,omitempty"`
	SupportDescription *string `json:"SupportDescription,omitempty"`
	SupportEmail *string `json:"SupportEmail,omitempty"`
	SupportUrl *string `json:"SupportUrl,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateProductOutput struct {
	ProductViewDetail *ProductViewDetail `json:"ProductViewDetail,omitempty"`
	ProvisioningArtifactDetail *ProvisioningArtifactDetail `json:"ProvisioningArtifactDetail,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateProvisionedProductPlanInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	NotificationArns []string `json:"NotificationArns,omitempty"`
	PathId *string `json:"PathId,omitempty"`
	PlanName string `json:"PlanName,omitempty"`
	PlanType string `json:"PlanType,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisionedProductName string `json:"ProvisionedProductName,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningParameters []UpdateProvisioningParameter `json:"ProvisioningParameters,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateProvisionedProductPlanOutput struct {
	PlanId *string `json:"PlanId,omitempty"`
	PlanName *string `json:"PlanName,omitempty"`
	ProvisionProductId *string `json:"ProvisionProductId,omitempty"`
	ProvisionedProductName *string `json:"ProvisionedProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
}

type CreateProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Parameters ProvisioningArtifactProperties `json:"Parameters,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
}

type CreateProvisioningArtifactOutput struct {
	Info map[string]string `json:"Info,omitempty"`
	ProvisioningArtifactDetail *ProvisioningArtifactDetail `json:"ProvisioningArtifactDetail,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type CreateServiceActionInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Definition map[string]string `json:"Definition,omitempty"`
	DefinitionType string `json:"DefinitionType,omitempty"`
	Description *string `json:"Description,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Name string `json:"Name,omitempty"`
}

type CreateServiceActionOutput struct {
	ServiceActionDetail *ServiceActionDetail `json:"ServiceActionDetail,omitempty"`
}

type CreateTagOptionInput struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type CreateTagOptionOutput struct {
	TagOptionDetail *TagOptionDetail `json:"TagOptionDetail,omitempty"`
}

type DeleteConstraintInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DeleteConstraintOutput struct {
}

type DeletePortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DeletePortfolioOutput struct {
}

type DeletePortfolioShareInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccountId *string `json:"AccountId,omitempty"`
	OrganizationNode *OrganizationNode `json:"OrganizationNode,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
}

type DeletePortfolioShareOutput struct {
	PortfolioShareToken *string `json:"PortfolioShareToken,omitempty"`
}

type DeleteProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DeleteProductOutput struct {
}

type DeleteProvisionedProductPlanInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IgnoreErrors bool `json:"IgnoreErrors,omitempty"`
	PlanId string `json:"PlanId,omitempty"`
}

type DeleteProvisionedProductPlanOutput struct {
}

type DeleteProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
}

type DeleteProvisioningArtifactOutput struct {
}

type DeleteServiceActionInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
	IdempotencyToken *string `json:"IdempotencyToken,omitempty"`
}

type DeleteServiceActionOutput struct {
}

type DeleteTagOptionInput struct {
	Id string `json:"Id,omitempty"`
}

type DeleteTagOptionOutput struct {
}

type DescribeConstraintInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DescribeConstraintOutput struct {
	ConstraintDetail *ConstraintDetail `json:"ConstraintDetail,omitempty"`
	ConstraintParameters *string `json:"ConstraintParameters,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DescribeCopyProductStatusInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	CopyProductToken string `json:"CopyProductToken,omitempty"`
}

type DescribeCopyProductStatusOutput struct {
	CopyProductStatus *string `json:"CopyProductStatus,omitempty"`
	StatusDetail *string `json:"StatusDetail,omitempty"`
	TargetProductId *string `json:"TargetProductId,omitempty"`
}

type DescribePortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DescribePortfolioOutput struct {
	Budgets []BudgetDetail `json:"Budgets,omitempty"`
	PortfolioDetail *PortfolioDetail `json:"PortfolioDetail,omitempty"`
	TagOptions []TagOptionDetail `json:"TagOptions,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type DescribePortfolioShareStatusInput struct {
	PortfolioShareToken string `json:"PortfolioShareToken,omitempty"`
}

type DescribePortfolioShareStatusOutput struct {
	OrganizationNodeValue *string `json:"OrganizationNodeValue,omitempty"`
	PortfolioId *string `json:"PortfolioId,omitempty"`
	PortfolioShareToken *string `json:"PortfolioShareToken,omitempty"`
	ShareDetails *ShareDetails `json:"ShareDetails,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DescribePortfolioSharesInput struct {
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	Type string `json:"Type,omitempty"`
}

type DescribePortfolioSharesOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	PortfolioShareDetails []PortfolioShareDetail `json:"PortfolioShareDetails,omitempty"`
}

type DescribeProductAsAdminInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	SourcePortfolioId *string `json:"SourcePortfolioId,omitempty"`
}

type DescribeProductAsAdminOutput struct {
	Budgets []BudgetDetail `json:"Budgets,omitempty"`
	ProductViewDetail *ProductViewDetail `json:"ProductViewDetail,omitempty"`
	ProvisioningArtifactSummaries []ProvisioningArtifactSummary `json:"ProvisioningArtifactSummaries,omitempty"`
	TagOptions []TagOptionDetail `json:"TagOptions,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type DescribeProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type DescribeProductOutput struct {
	Budgets []BudgetDetail `json:"Budgets,omitempty"`
	LaunchPaths []LaunchPath `json:"LaunchPaths,omitempty"`
	ProductViewSummary *ProductViewSummary `json:"ProductViewSummary,omitempty"`
	ProvisioningArtifacts []ProvisioningArtifact `json:"ProvisioningArtifacts,omitempty"`
}

type DescribeProductViewInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DescribeProductViewOutput struct {
	ProductViewSummary *ProductViewSummary `json:"ProductViewSummary,omitempty"`
	ProvisioningArtifacts []ProvisioningArtifact `json:"ProvisioningArtifacts,omitempty"`
}

type DescribeProvisionedProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type DescribeProvisionedProductOutput struct {
	CloudWatchDashboards []CloudWatchDashboard `json:"CloudWatchDashboards,omitempty"`
	ProvisionedProductDetail *ProvisionedProductDetail `json:"ProvisionedProductDetail,omitempty"`
}

type DescribeProvisionedProductPlanInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PlanId string `json:"PlanId,omitempty"`
}

type DescribeProvisionedProductPlanOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProvisionedProductPlanDetails *ProvisionedProductPlanDetails `json:"ProvisionedProductPlanDetails,omitempty"`
	ResourceChanges []ResourceChange `json:"ResourceChanges,omitempty"`
}

type DescribeProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IncludeProvisioningArtifactParameters bool `json:"IncludeProvisioningArtifactParameters,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningArtifactName *string `json:"ProvisioningArtifactName,omitempty"`
	Verbose bool `json:"Verbose,omitempty"`
}

type DescribeProvisioningArtifactOutput struct {
	Info map[string]string `json:"Info,omitempty"`
	ProvisioningArtifactDetail *ProvisioningArtifactDetail `json:"ProvisioningArtifactDetail,omitempty"`
	ProvisioningArtifactParameters []ProvisioningArtifactParameter `json:"ProvisioningArtifactParameters,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type DescribeProvisioningParametersInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PathId *string `json:"PathId,omitempty"`
	PathName *string `json:"PathName,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningArtifactName *string `json:"ProvisioningArtifactName,omitempty"`
}

type DescribeProvisioningParametersOutput struct {
	ConstraintSummaries []ConstraintSummary `json:"ConstraintSummaries,omitempty"`
	ProvisioningArtifactOutputKeys []ProvisioningArtifactOutput `json:"ProvisioningArtifactOutputKeys,omitempty"`
	ProvisioningArtifactOutputs []ProvisioningArtifactOutput `json:"ProvisioningArtifactOutputs,omitempty"`
	ProvisioningArtifactParameters []ProvisioningArtifactParameter `json:"ProvisioningArtifactParameters,omitempty"`
	ProvisioningArtifactPreferences *ProvisioningArtifactPreferences `json:"ProvisioningArtifactPreferences,omitempty"`
	TagOptions []TagOptionSummary `json:"TagOptions,omitempty"`
	UsageInstructions []UsageInstruction `json:"UsageInstructions,omitempty"`
}

type DescribeRecordInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
}

type DescribeRecordOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
	RecordOutputs []RecordOutput `json:"RecordOutputs,omitempty"`
}

type DescribeServiceActionExecutionParametersInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	ProvisionedProductId string `json:"ProvisionedProductId,omitempty"`
	ServiceActionId string `json:"ServiceActionId,omitempty"`
}

type DescribeServiceActionExecutionParametersOutput struct {
	ServiceActionParameters []ExecutionParameter `json:"ServiceActionParameters,omitempty"`
}

type DescribeServiceActionInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type DescribeServiceActionOutput struct {
	ServiceActionDetail *ServiceActionDetail `json:"ServiceActionDetail,omitempty"`
}

type DescribeTagOptionInput struct {
	Id string `json:"Id,omitempty"`
}

type DescribeTagOptionOutput struct {
	TagOptionDetail *TagOptionDetail `json:"TagOptionDetail,omitempty"`
}

type DisableAWSOrganizationsAccessInput struct {
}

type DisableAWSOrganizationsAccessOutput struct {
}

type DisassociateBudgetFromResourceInput struct {
	BudgetName string `json:"BudgetName,omitempty"`
	ResourceId string `json:"ResourceId,omitempty"`
}

type DisassociateBudgetFromResourceOutput struct {
}

type DisassociatePrincipalFromPortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	PrincipalARN string `json:"PrincipalARN,omitempty"`
	PrincipalType *string `json:"PrincipalType,omitempty"`
}

type DisassociatePrincipalFromPortfolioOutput struct {
}

type DisassociateProductFromPortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
}

type DisassociateProductFromPortfolioOutput struct {
}

type DisassociateServiceActionFromProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken *string `json:"IdempotencyToken,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
	ServiceActionId string `json:"ServiceActionId,omitempty"`
}

type DisassociateServiceActionFromProvisioningArtifactOutput struct {
}

type DisassociateTagOptionFromResourceInput struct {
	ResourceId string `json:"ResourceId,omitempty"`
	TagOptionId string `json:"TagOptionId,omitempty"`
}

type DisassociateTagOptionFromResourceOutput struct {
}

type EnableAWSOrganizationsAccessInput struct {
}

type EnableAWSOrganizationsAccessOutput struct {
}

type EngineWorkflowResourceIdentifier struct {
	UniqueTag *UniqueTagResourceIdentifier `json:"UniqueTag,omitempty"`
}

type ExecuteProvisionedProductPlanInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	PlanId string `json:"PlanId,omitempty"`
}

type ExecuteProvisionedProductPlanOutput struct {
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
}

type ExecuteProvisionedProductServiceActionInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	ExecuteToken string `json:"ExecuteToken,omitempty"`
	Parameters map[string][]string `json:"Parameters,omitempty"`
	ProvisionedProductId string `json:"ProvisionedProductId,omitempty"`
	ServiceActionId string `json:"ServiceActionId,omitempty"`
}

type ExecuteProvisionedProductServiceActionOutput struct {
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
}

type ExecutionParameter struct {
	DefaultValues []string `json:"DefaultValues,omitempty"`
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type FailedServiceActionAssociation struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ServiceActionId *string `json:"ServiceActionId,omitempty"`
}

type GetAWSOrganizationsAccessStatusInput struct {
}

type GetAWSOrganizationsAccessStatusOutput struct {
	AccessStatus *string `json:"AccessStatus,omitempty"`
}

type GetProvisionedProductOutputsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	OutputKeys []string `json:"OutputKeys,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ProvisionedProductId *string `json:"ProvisionedProductId,omitempty"`
	ProvisionedProductName *string `json:"ProvisionedProductName,omitempty"`
}

type GetProvisionedProductOutputsOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	Outputs []RecordOutput `json:"Outputs,omitempty"`
}

type ImportAsProvisionedProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	PhysicalId string `json:"PhysicalId,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisionedProductName string `json:"ProvisionedProductName,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
}

type ImportAsProvisionedProductOutput struct {
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
}

type LastSync struct {
	LastSuccessfulSyncProvisioningArtifactId *string `json:"LastSuccessfulSyncProvisioningArtifactId,omitempty"`
	LastSuccessfulSyncTime *time.Time `json:"LastSuccessfulSyncTime,omitempty"`
	LastSyncStatus *string `json:"LastSyncStatus,omitempty"`
	LastSyncStatusMessage *string `json:"LastSyncStatusMessage,omitempty"`
	LastSyncTime *time.Time `json:"LastSyncTime,omitempty"`
}

type LaunchPath struct {
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type LaunchPathSummary struct {
	ConstraintSummaries []ConstraintSummary `json:"ConstraintSummaries,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type ListAcceptedPortfolioSharesInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioShareType *string `json:"PortfolioShareType,omitempty"`
}

type ListAcceptedPortfolioSharesOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	PortfolioDetails []PortfolioDetail `json:"PortfolioDetails,omitempty"`
}

type ListBudgetsForResourceInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ResourceId string `json:"ResourceId,omitempty"`
}

type ListBudgetsForResourceOutput struct {
	Budgets []BudgetDetail `json:"Budgets,omitempty"`
	NextPageToken *string `json:"NextPageToken,omitempty"`
}

type ListConstraintsForPortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
}

type ListConstraintsForPortfolioOutput struct {
	ConstraintDetails []ConstraintDetail `json:"ConstraintDetails,omitempty"`
	NextPageToken *string `json:"NextPageToken,omitempty"`
}

type ListLaunchPathsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
}

type ListLaunchPathsOutput struct {
	LaunchPathSummaries []LaunchPathSummary `json:"LaunchPathSummaries,omitempty"`
	NextPageToken *string `json:"NextPageToken,omitempty"`
}

type ListOrganizationPortfolioAccessInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	OrganizationNodeType string `json:"OrganizationNodeType,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
}

type ListOrganizationPortfolioAccessOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	OrganizationNodes []OrganizationNode `json:"OrganizationNodes,omitempty"`
}

type ListPortfolioAccessInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	OrganizationParentId *string `json:"OrganizationParentId,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
}

type ListPortfolioAccessOutput struct {
	AccountIds []string `json:"AccountIds,omitempty"`
	NextPageToken *string `json:"NextPageToken,omitempty"`
}

type ListPortfoliosForProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
}

type ListPortfoliosForProductOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	PortfolioDetails []PortfolioDetail `json:"PortfolioDetails,omitempty"`
}

type ListPortfoliosInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
}

type ListPortfoliosOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	PortfolioDetails []PortfolioDetail `json:"PortfolioDetails,omitempty"`
}

type ListPrincipalsForPortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
}

type ListPrincipalsForPortfolioOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	Principals []Principal `json:"Principals,omitempty"`
}

type ListProvisionedProductPlansInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccessLevelFilter *AccessLevelFilter `json:"AccessLevelFilter,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ProvisionProductId *string `json:"ProvisionProductId,omitempty"`
}

type ListProvisionedProductPlansOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProvisionedProductPlans []ProvisionedProductPlanSummary `json:"ProvisionedProductPlans,omitempty"`
}

type ListProvisioningArtifactsForServiceActionInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ServiceActionId string `json:"ServiceActionId,omitempty"`
}

type ListProvisioningArtifactsForServiceActionOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProvisioningArtifactViews []ProvisioningArtifactView `json:"ProvisioningArtifactViews,omitempty"`
}

type ListProvisioningArtifactsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
}

type ListProvisioningArtifactsOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProvisioningArtifactDetails []ProvisioningArtifactDetail `json:"ProvisioningArtifactDetails,omitempty"`
}

type ListRecordHistoryInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccessLevelFilter *AccessLevelFilter `json:"AccessLevelFilter,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	SearchFilter *ListRecordHistorySearchFilter `json:"SearchFilter,omitempty"`
}

type ListRecordHistoryOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	RecordDetails []RecordDetail `json:"RecordDetails,omitempty"`
}

type ListRecordHistorySearchFilter struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type ListResourcesForTagOptionInput struct {
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ResourceType *string `json:"ResourceType,omitempty"`
	TagOptionId string `json:"TagOptionId,omitempty"`
}

type ListResourcesForTagOptionOutput struct {
	PageToken *string `json:"PageToken,omitempty"`
	ResourceDetails []ResourceDetail `json:"ResourceDetails,omitempty"`
}

type ListServiceActionsForProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
}

type ListServiceActionsForProvisioningArtifactOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ServiceActionSummaries []ServiceActionSummary `json:"ServiceActionSummaries,omitempty"`
}

type ListServiceActionsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
}

type ListServiceActionsOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ServiceActionSummaries []ServiceActionSummary `json:"ServiceActionSummaries,omitempty"`
}

type ListStackInstancesForProvisionedProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	ProvisionedProductId string `json:"ProvisionedProductId,omitempty"`
}

type ListStackInstancesForProvisionedProductOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	StackInstances []StackInstance `json:"StackInstances,omitempty"`
}

type ListTagOptionsFilters struct {
	Active bool `json:"Active,omitempty"`
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type ListTagOptionsInput struct {
	Filters *ListTagOptionsFilters `json:"Filters,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
}

type ListTagOptionsOutput struct {
	PageToken *string `json:"PageToken,omitempty"`
	TagOptionDetails []TagOptionDetail `json:"TagOptionDetails,omitempty"`
}

type NotifyProvisionProductEngineWorkflowResultInput struct {
	FailureReason *string `json:"FailureReason,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Outputs []RecordOutput `json:"Outputs,omitempty"`
	RecordId string `json:"RecordId,omitempty"`
	ResourceIdentifier *EngineWorkflowResourceIdentifier `json:"ResourceIdentifier,omitempty"`
	Status string `json:"Status,omitempty"`
	WorkflowToken string `json:"WorkflowToken,omitempty"`
}

type NotifyProvisionProductEngineWorkflowResultOutput struct {
}

type NotifyTerminateProvisionedProductEngineWorkflowResultInput struct {
	FailureReason *string `json:"FailureReason,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	RecordId string `json:"RecordId,omitempty"`
	Status string `json:"Status,omitempty"`
	WorkflowToken string `json:"WorkflowToken,omitempty"`
}

type NotifyTerminateProvisionedProductEngineWorkflowResultOutput struct {
}

type NotifyUpdateProvisionedProductEngineWorkflowResultInput struct {
	FailureReason *string `json:"FailureReason,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Outputs []RecordOutput `json:"Outputs,omitempty"`
	RecordId string `json:"RecordId,omitempty"`
	Status string `json:"Status,omitempty"`
	WorkflowToken string `json:"WorkflowToken,omitempty"`
}

type NotifyUpdateProvisionedProductEngineWorkflowResultOutput struct {
}

type OrganizationNode struct {
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type ParameterConstraints struct {
	AllowedPattern *string `json:"AllowedPattern,omitempty"`
	AllowedValues []string `json:"AllowedValues,omitempty"`
	ConstraintDescription *string `json:"ConstraintDescription,omitempty"`
	MaxLength *string `json:"MaxLength,omitempty"`
	MaxValue *string `json:"MaxValue,omitempty"`
	MinLength *string `json:"MinLength,omitempty"`
	MinValue *string `json:"MinValue,omitempty"`
}

type PortfolioDetail struct {
	ARN *string `json:"ARN,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	DisplayName *string `json:"DisplayName,omitempty"`
	Id *string `json:"Id,omitempty"`
	ProviderName *string `json:"ProviderName,omitempty"`
}

type PortfolioShareDetail struct {
	Accepted bool `json:"Accepted,omitempty"`
	PrincipalId *string `json:"PrincipalId,omitempty"`
	SharePrincipals bool `json:"SharePrincipals,omitempty"`
	ShareTagOptions bool `json:"ShareTagOptions,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type Principal struct {
	PrincipalARN *string `json:"PrincipalARN,omitempty"`
	PrincipalType *string `json:"PrincipalType,omitempty"`
}

type ProductViewAggregationValue struct {
	ApproximateCount int `json:"ApproximateCount,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type ProductViewDetail struct {
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	ProductARN *string `json:"ProductARN,omitempty"`
	ProductViewSummary *ProductViewSummary `json:"ProductViewSummary,omitempty"`
	SourceConnection *SourceConnectionDetail `json:"SourceConnection,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type ProductViewSummary struct {
	Distributor *string `json:"Distributor,omitempty"`
	HasDefaultPath bool `json:"HasDefaultPath,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	Owner *string `json:"Owner,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ShortDescription *string `json:"ShortDescription,omitempty"`
	SupportDescription *string `json:"SupportDescription,omitempty"`
	SupportEmail *string `json:"SupportEmail,omitempty"`
	SupportUrl *string `json:"SupportUrl,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ProvisionProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	NotificationArns []string `json:"NotificationArns,omitempty"`
	PathId *string `json:"PathId,omitempty"`
	PathName *string `json:"PathName,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	ProvisionToken string `json:"ProvisionToken,omitempty"`
	ProvisionedProductName string `json:"ProvisionedProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningArtifactName *string `json:"ProvisioningArtifactName,omitempty"`
	ProvisioningParameters []ProvisioningParameter `json:"ProvisioningParameters,omitempty"`
	ProvisioningPreferences *ProvisioningPreferences `json:"ProvisioningPreferences,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type ProvisionProductOutput struct {
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
}

type ProvisionedProductAttribute struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Id *string `json:"Id,omitempty"`
	IdempotencyToken *string `json:"IdempotencyToken,omitempty"`
	LastProvisioningRecordId *string `json:"LastProvisioningRecordId,omitempty"`
	LastRecordId *string `json:"LastRecordId,omitempty"`
	LastSuccessfulProvisioningRecordId *string `json:"LastSuccessfulProvisioningRecordId,omitempty"`
	Name *string `json:"Name,omitempty"`
	PhysicalId *string `json:"PhysicalId,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningArtifactName *string `json:"ProvisioningArtifactName,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	Type *string `json:"Type,omitempty"`
	UserArn *string `json:"UserArn,omitempty"`
	UserArnSession *string `json:"UserArnSession,omitempty"`
}

type ProvisionedProductDetail struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Id *string `json:"Id,omitempty"`
	IdempotencyToken *string `json:"IdempotencyToken,omitempty"`
	LastProvisioningRecordId *string `json:"LastProvisioningRecordId,omitempty"`
	LastRecordId *string `json:"LastRecordId,omitempty"`
	LastSuccessfulProvisioningRecordId *string `json:"LastSuccessfulProvisioningRecordId,omitempty"`
	LaunchRoleArn *string `json:"LaunchRoleArn,omitempty"`
	Name *string `json:"Name,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ProvisionedProductPlanDetails struct {
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	NotificationArns []string `json:"NotificationArns,omitempty"`
	PathId *string `json:"PathId,omitempty"`
	PlanId *string `json:"PlanId,omitempty"`
	PlanName *string `json:"PlanName,omitempty"`
	PlanType *string `json:"PlanType,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProvisionProductId *string `json:"ProvisionProductId,omitempty"`
	ProvisionProductName *string `json:"ProvisionProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningParameters []UpdateProvisioningParameter `json:"ProvisioningParameters,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	UpdatedTime *time.Time `json:"UpdatedTime,omitempty"`
}

type ProvisionedProductPlanSummary struct {
	PlanId *string `json:"PlanId,omitempty"`
	PlanName *string `json:"PlanName,omitempty"`
	PlanType *string `json:"PlanType,omitempty"`
	ProvisionProductId *string `json:"ProvisionProductId,omitempty"`
	ProvisionProductName *string `json:"ProvisionProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
}

type ProvisioningArtifact struct {
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Guidance *string `json:"Guidance,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ProvisioningArtifactDetail struct {
	Active bool `json:"Active,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Guidance *string `json:"Guidance,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	SourceRevision *string `json:"SourceRevision,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ProvisioningArtifactOutput struct {
	Description *string `json:"Description,omitempty"`
	Key *string `json:"Key,omitempty"`
}

type ProvisioningArtifactParameter struct {
	DefaultValue *string `json:"DefaultValue,omitempty"`
	Description *string `json:"Description,omitempty"`
	IsNoEcho bool `json:"IsNoEcho,omitempty"`
	ParameterConstraints *ParameterConstraints `json:"ParameterConstraints,omitempty"`
	ParameterKey *string `json:"ParameterKey,omitempty"`
	ParameterType *string `json:"ParameterType,omitempty"`
}

type ProvisioningArtifactPreferences struct {
	StackSetAccounts []string `json:"StackSetAccounts,omitempty"`
	StackSetRegions []string `json:"StackSetRegions,omitempty"`
}

type ProvisioningArtifactProperties struct {
	Description *string `json:"Description,omitempty"`
	DisableTemplateValidation bool `json:"DisableTemplateValidation,omitempty"`
	Info map[string]string `json:"Info,omitempty"`
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ProvisioningArtifactSummary struct {
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	ProvisioningArtifactMetadata map[string]string `json:"ProvisioningArtifactMetadata,omitempty"`
}

type ProvisioningArtifactView struct {
	ProductViewSummary *ProductViewSummary `json:"ProductViewSummary,omitempty"`
	ProvisioningArtifact *ProvisioningArtifact `json:"ProvisioningArtifact,omitempty"`
}

type ProvisioningParameter struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type ProvisioningPreferences struct {
	StackSetAccounts []string `json:"StackSetAccounts,omitempty"`
	StackSetFailureToleranceCount int `json:"StackSetFailureToleranceCount,omitempty"`
	StackSetFailureTolerancePercentage int `json:"StackSetFailureTolerancePercentage,omitempty"`
	StackSetMaxConcurrencyCount int `json:"StackSetMaxConcurrencyCount,omitempty"`
	StackSetMaxConcurrencyPercentage int `json:"StackSetMaxConcurrencyPercentage,omitempty"`
	StackSetRegions []string `json:"StackSetRegions,omitempty"`
}

type RecordDetail struct {
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LaunchRoleArn *string `json:"LaunchRoleArn,omitempty"`
	PathId *string `json:"PathId,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProvisionedProductId *string `json:"ProvisionedProductId,omitempty"`
	ProvisionedProductName *string `json:"ProvisionedProductName,omitempty"`
	ProvisionedProductType *string `json:"ProvisionedProductType,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	RecordErrors []RecordError `json:"RecordErrors,omitempty"`
	RecordId *string `json:"RecordId,omitempty"`
	RecordTags []RecordTag `json:"RecordTags,omitempty"`
	RecordType *string `json:"RecordType,omitempty"`
	Status *string `json:"Status,omitempty"`
	UpdatedTime *time.Time `json:"UpdatedTime,omitempty"`
}

type RecordError struct {
	Code *string `json:"Code,omitempty"`
	Description *string `json:"Description,omitempty"`
}

type RecordOutput struct {
	Description *string `json:"Description,omitempty"`
	OutputKey *string `json:"OutputKey,omitempty"`
	OutputValue *string `json:"OutputValue,omitempty"`
}

type RecordTag struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type RejectPortfolioShareInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	PortfolioShareType *string `json:"PortfolioShareType,omitempty"`
}

type RejectPortfolioShareOutput struct {
}

type ResourceChange struct {
	Action *string `json:"Action,omitempty"`
	Details []ResourceChangeDetail `json:"Details,omitempty"`
	LogicalResourceId *string `json:"LogicalResourceId,omitempty"`
	PhysicalResourceId *string `json:"PhysicalResourceId,omitempty"`
	Replacement *string `json:"Replacement,omitempty"`
	ResourceType *string `json:"ResourceType,omitempty"`
	Scope []string `json:"Scope,omitempty"`
}

type ResourceChangeDetail struct {
	CausingEntity *string `json:"CausingEntity,omitempty"`
	Evaluation *string `json:"Evaluation,omitempty"`
	Target *ResourceTargetDefinition `json:"Target,omitempty"`
}

type ResourceDetail struct {
	ARN *string `json:"ARN,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ResourceTargetDefinition struct {
	Attribute *string `json:"Attribute,omitempty"`
	Name *string `json:"Name,omitempty"`
	RequiresRecreation *string `json:"RequiresRecreation,omitempty"`
}

type ScanProvisionedProductsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccessLevelFilter *AccessLevelFilter `json:"AccessLevelFilter,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
}

type ScanProvisionedProductsOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProvisionedProducts []ProvisionedProductDetail `json:"ProvisionedProducts,omitempty"`
}

type SearchProductsAsAdminInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Filters map[string][]string `json:"Filters,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	PortfolioId *string `json:"PortfolioId,omitempty"`
	ProductSource *string `json:"ProductSource,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
}

type SearchProductsAsAdminOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProductViewDetails []ProductViewDetail `json:"ProductViewDetails,omitempty"`
}

type SearchProductsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Filters map[string][]string `json:"Filters,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
}

type SearchProductsOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProductViewAggregations map[string][]ProductViewAggregationValue `json:"ProductViewAggregations,omitempty"`
	ProductViewSummaries []ProductViewSummary `json:"ProductViewSummaries,omitempty"`
}

type SearchProvisionedProductsInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccessLevelFilter *AccessLevelFilter `json:"AccessLevelFilter,omitempty"`
	Filters map[string][]string `json:"Filters,omitempty"`
	PageSize int `json:"PageSize,omitempty"`
	PageToken *string `json:"PageToken,omitempty"`
	SortBy *string `json:"SortBy,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
}

type SearchProvisionedProductsOutput struct {
	NextPageToken *string `json:"NextPageToken,omitempty"`
	ProvisionedProducts []ProvisionedProductAttribute `json:"ProvisionedProducts,omitempty"`
	TotalResultsCount int `json:"TotalResultsCount,omitempty"`
}

type ServiceActionAssociation struct {
	ProductId string `json:"ProductId,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
	ServiceActionId string `json:"ServiceActionId,omitempty"`
}

type ServiceActionDetail struct {
	Definition map[string]string `json:"Definition,omitempty"`
	ServiceActionSummary *ServiceActionSummary `json:"ServiceActionSummary,omitempty"`
}

type ServiceActionSummary struct {
	DefinitionType *string `json:"DefinitionType,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ShareDetails struct {
	ShareErrors []ShareError `json:"ShareErrors,omitempty"`
	SuccessfulShares []string `json:"SuccessfulShares,omitempty"`
}

type ShareError struct {
	Accounts []string `json:"Accounts,omitempty"`
	Error *string `json:"Error,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type SourceConnection struct {
	ConnectionParameters SourceConnectionParameters `json:"ConnectionParameters,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type SourceConnectionDetail struct {
	ConnectionParameters *SourceConnectionParameters `json:"ConnectionParameters,omitempty"`
	LastSync *LastSync `json:"LastSync,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type SourceConnectionParameters struct {
	CodeStar *CodeStarParameters `json:"CodeStar,omitempty"`
}

type StackInstance struct {
	Account *string `json:"Account,omitempty"`
	Region *string `json:"Region,omitempty"`
	StackInstanceStatus *string `json:"StackInstanceStatus,omitempty"`
}

type Tag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type TagOptionDetail struct {
	Active bool `json:"Active,omitempty"`
	Id *string `json:"Id,omitempty"`
	Key *string `json:"Key,omitempty"`
	Owner *string `json:"Owner,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type TagOptionSummary struct {
	Key *string `json:"Key,omitempty"`
	Values []string `json:"Values,omitempty"`
}

type TerminateProvisionedProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IgnoreErrors bool `json:"IgnoreErrors,omitempty"`
	ProvisionedProductId *string `json:"ProvisionedProductId,omitempty"`
	ProvisionedProductName *string `json:"ProvisionedProductName,omitempty"`
	RetainPhysicalResources bool `json:"RetainPhysicalResources,omitempty"`
	TerminateToken string `json:"TerminateToken,omitempty"`
}

type TerminateProvisionedProductOutput struct {
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
}

type UniqueTagResourceIdentifier struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type UpdateConstraintInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id string `json:"Id,omitempty"`
	Parameters *string `json:"Parameters,omitempty"`
}

type UpdateConstraintOutput struct {
	ConstraintDetail *ConstraintDetail `json:"ConstraintDetail,omitempty"`
	ConstraintParameters *string `json:"ConstraintParameters,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type UpdatePortfolioInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AddTags []Tag `json:"AddTags,omitempty"`
	Description *string `json:"Description,omitempty"`
	DisplayName *string `json:"DisplayName,omitempty"`
	Id string `json:"Id,omitempty"`
	ProviderName *string `json:"ProviderName,omitempty"`
	RemoveTags []string `json:"RemoveTags,omitempty"`
}

type UpdatePortfolioOutput struct {
	PortfolioDetail *PortfolioDetail `json:"PortfolioDetail,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type UpdatePortfolioShareInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AccountId *string `json:"AccountId,omitempty"`
	OrganizationNode *OrganizationNode `json:"OrganizationNode,omitempty"`
	PortfolioId string `json:"PortfolioId,omitempty"`
	SharePrincipals bool `json:"SharePrincipals,omitempty"`
	ShareTagOptions bool `json:"ShareTagOptions,omitempty"`
}

type UpdatePortfolioShareOutput struct {
	PortfolioShareToken *string `json:"PortfolioShareToken,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type UpdateProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	AddTags []Tag `json:"AddTags,omitempty"`
	Description *string `json:"Description,omitempty"`
	Distributor *string `json:"Distributor,omitempty"`
	Id string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	Owner *string `json:"Owner,omitempty"`
	RemoveTags []string `json:"RemoveTags,omitempty"`
	SourceConnection *SourceConnection `json:"SourceConnection,omitempty"`
	SupportDescription *string `json:"SupportDescription,omitempty"`
	SupportEmail *string `json:"SupportEmail,omitempty"`
	SupportUrl *string `json:"SupportUrl,omitempty"`
}

type UpdateProductOutput struct {
	ProductViewDetail *ProductViewDetail `json:"ProductViewDetail,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type UpdateProvisionedProductInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	PathId *string `json:"PathId,omitempty"`
	PathName *string `json:"PathName,omitempty"`
	ProductId *string `json:"ProductId,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	ProvisionedProductId *string `json:"ProvisionedProductId,omitempty"`
	ProvisionedProductName *string `json:"ProvisionedProductName,omitempty"`
	ProvisioningArtifactId *string `json:"ProvisioningArtifactId,omitempty"`
	ProvisioningArtifactName *string `json:"ProvisioningArtifactName,omitempty"`
	ProvisioningParameters []UpdateProvisioningParameter `json:"ProvisioningParameters,omitempty"`
	ProvisioningPreferences *UpdateProvisioningPreferences `json:"ProvisioningPreferences,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	UpdateToken string `json:"UpdateToken,omitempty"`
}

type UpdateProvisionedProductOutput struct {
	RecordDetail *RecordDetail `json:"RecordDetail,omitempty"`
}

type UpdateProvisionedProductPropertiesInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	ProvisionedProductId string `json:"ProvisionedProductId,omitempty"`
	ProvisionedProductProperties map[string]string `json:"ProvisionedProductProperties,omitempty"`
}

type UpdateProvisionedProductPropertiesOutput struct {
	ProvisionedProductId *string `json:"ProvisionedProductId,omitempty"`
	ProvisionedProductProperties map[string]string `json:"ProvisionedProductProperties,omitempty"`
	RecordId *string `json:"RecordId,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type UpdateProvisioningArtifactInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Active bool `json:"Active,omitempty"`
	Description *string `json:"Description,omitempty"`
	Guidance *string `json:"Guidance,omitempty"`
	Name *string `json:"Name,omitempty"`
	ProductId string `json:"ProductId,omitempty"`
	ProvisioningArtifactId string `json:"ProvisioningArtifactId,omitempty"`
}

type UpdateProvisioningArtifactOutput struct {
	Info map[string]string `json:"Info,omitempty"`
	ProvisioningArtifactDetail *ProvisioningArtifactDetail `json:"ProvisioningArtifactDetail,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type UpdateProvisioningParameter struct {
	Key *string `json:"Key,omitempty"`
	UsePreviousValue bool `json:"UsePreviousValue,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type UpdateProvisioningPreferences struct {
	StackSetAccounts []string `json:"StackSetAccounts,omitempty"`
	StackSetFailureToleranceCount int `json:"StackSetFailureToleranceCount,omitempty"`
	StackSetFailureTolerancePercentage int `json:"StackSetFailureTolerancePercentage,omitempty"`
	StackSetMaxConcurrencyCount int `json:"StackSetMaxConcurrencyCount,omitempty"`
	StackSetMaxConcurrencyPercentage int `json:"StackSetMaxConcurrencyPercentage,omitempty"`
	StackSetOperationType *string `json:"StackSetOperationType,omitempty"`
	StackSetRegions []string `json:"StackSetRegions,omitempty"`
}

type UpdateServiceActionInput struct {
	AcceptLanguage *string `json:"AcceptLanguage,omitempty"`
	Definition map[string]string `json:"Definition,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type UpdateServiceActionOutput struct {
	ServiceActionDetail *ServiceActionDetail `json:"ServiceActionDetail,omitempty"`
}

type UpdateTagOptionInput struct {
	Active bool `json:"Active,omitempty"`
	Id string `json:"Id,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type UpdateTagOptionOutput struct {
	TagOptionDetail *TagOptionDetail `json:"TagOptionDetail,omitempty"`
}

type UsageInstruction struct {
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}



// ── Handler helpers ──────────────────────────────────────────────────────────

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
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleAcceptPortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AcceptPortfolioShareInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AcceptPortfolioShare business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AcceptPortfolioShare"})
}

func handleAssociateBudgetWithResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociateBudgetWithResourceInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociateBudgetWithResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociateBudgetWithResource"})
}

func handleAssociatePrincipalWithPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociatePrincipalWithPortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociatePrincipalWithPortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociatePrincipalWithPortfolio"})
}

func handleAssociateProductWithPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociateProductWithPortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociateProductWithPortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociateProductWithPortfolio"})
}

func handleAssociateServiceActionWithProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociateServiceActionWithProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociateServiceActionWithProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociateServiceActionWithProvisioningArtifact"})
}

func handleAssociateTagOptionWithResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociateTagOptionWithResourceInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociateTagOptionWithResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociateTagOptionWithResource"})
}

func handleBatchAssociateServiceActionWithProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchAssociateServiceActionWithProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchAssociateServiceActionWithProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchAssociateServiceActionWithProvisioningArtifact"})
}

func handleBatchDisassociateServiceActionFromProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDisassociateServiceActionFromProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDisassociateServiceActionFromProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDisassociateServiceActionFromProvisioningArtifact"})
}

func handleCopyProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CopyProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CopyProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CopyProduct"})
}

func handleCreateConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateConstraintInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateConstraint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateConstraint"})
}

func handleCreatePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreatePortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreatePortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreatePortfolio"})
}

func handleCreatePortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreatePortfolioShareInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreatePortfolioShare business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreatePortfolioShare"})
}

func handleCreateProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateProduct"})
}

func handleCreateProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateProvisionedProductPlanInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateProvisionedProductPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateProvisionedProductPlan"})
}

func handleCreateProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateProvisioningArtifact"})
}

func handleCreateServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateServiceActionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateServiceAction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateServiceAction"})
}

func handleCreateTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTagOptionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTagOption business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTagOption"})
}

func handleDeleteConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteConstraintInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteConstraint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteConstraint"})
}

func handleDeletePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeletePortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeletePortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeletePortfolio"})
}

func handleDeletePortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeletePortfolioShareInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeletePortfolioShare business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeletePortfolioShare"})
}

func handleDeleteProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteProduct"})
}

func handleDeleteProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteProvisionedProductPlanInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteProvisionedProductPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteProvisionedProductPlan"})
}

func handleDeleteProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteProvisioningArtifact"})
}

func handleDeleteServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteServiceActionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteServiceAction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteServiceAction"})
}

func handleDeleteTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTagOptionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTagOption business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTagOption"})
}

func handleDescribeConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeConstraintInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeConstraint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeConstraint"})
}

func handleDescribeCopyProductStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCopyProductStatusInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCopyProductStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCopyProductStatus"})
}

func handleDescribePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribePortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribePortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribePortfolio"})
}

func handleDescribePortfolioShareStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribePortfolioShareStatusInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribePortfolioShareStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribePortfolioShareStatus"})
}

func handleDescribePortfolioShares(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribePortfolioSharesInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribePortfolioShares business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribePortfolioShares"})
}

func handleDescribeProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProduct"})
}

func handleDescribeProductAsAdmin(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProductAsAdminInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProductAsAdmin business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProductAsAdmin"})
}

func handleDescribeProductView(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProductViewInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProductView business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProductView"})
}

func handleDescribeProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProvisionedProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProvisionedProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProvisionedProduct"})
}

func handleDescribeProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProvisionedProductPlanInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProvisionedProductPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProvisionedProductPlan"})
}

func handleDescribeProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProvisioningArtifact"})
}

func handleDescribeProvisioningParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProvisioningParametersInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProvisioningParameters business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProvisioningParameters"})
}

func handleDescribeRecord(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeRecordInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeRecord business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeRecord"})
}

func handleDescribeServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeServiceActionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeServiceAction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeServiceAction"})
}

func handleDescribeServiceActionExecutionParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeServiceActionExecutionParametersInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeServiceActionExecutionParameters business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeServiceActionExecutionParameters"})
}

func handleDescribeTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTagOptionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTagOption business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTagOption"})
}

func handleDisableAWSOrganizationsAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableAWSOrganizationsAccessInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableAWSOrganizationsAccess business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableAWSOrganizationsAccess"})
}

func handleDisassociateBudgetFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateBudgetFromResourceInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateBudgetFromResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateBudgetFromResource"})
}

func handleDisassociatePrincipalFromPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociatePrincipalFromPortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociatePrincipalFromPortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociatePrincipalFromPortfolio"})
}

func handleDisassociateProductFromPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateProductFromPortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateProductFromPortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateProductFromPortfolio"})
}

func handleDisassociateServiceActionFromProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateServiceActionFromProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateServiceActionFromProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateServiceActionFromProvisioningArtifact"})
}

func handleDisassociateTagOptionFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateTagOptionFromResourceInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateTagOptionFromResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateTagOptionFromResource"})
}

func handleEnableAWSOrganizationsAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableAWSOrganizationsAccessInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableAWSOrganizationsAccess business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableAWSOrganizationsAccess"})
}

func handleExecuteProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ExecuteProvisionedProductPlanInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ExecuteProvisionedProductPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ExecuteProvisionedProductPlan"})
}

func handleExecuteProvisionedProductServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ExecuteProvisionedProductServiceActionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ExecuteProvisionedProductServiceAction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ExecuteProvisionedProductServiceAction"})
}

func handleGetAWSOrganizationsAccessStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetAWSOrganizationsAccessStatusInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetAWSOrganizationsAccessStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetAWSOrganizationsAccessStatus"})
}

func handleGetProvisionedProductOutputs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetProvisionedProductOutputsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetProvisionedProductOutputs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetProvisionedProductOutputs"})
}

func handleImportAsProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ImportAsProvisionedProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ImportAsProvisionedProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ImportAsProvisionedProduct"})
}

func handleListAcceptedPortfolioShares(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAcceptedPortfolioSharesInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAcceptedPortfolioShares business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAcceptedPortfolioShares"})
}

func handleListBudgetsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListBudgetsForResourceInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListBudgetsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListBudgetsForResource"})
}

func handleListConstraintsForPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListConstraintsForPortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListConstraintsForPortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListConstraintsForPortfolio"})
}

func handleListLaunchPaths(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListLaunchPathsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListLaunchPaths business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListLaunchPaths"})
}

func handleListOrganizationPortfolioAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListOrganizationPortfolioAccessInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListOrganizationPortfolioAccess business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListOrganizationPortfolioAccess"})
}

func handleListPortfolioAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListPortfolioAccessInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListPortfolioAccess business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListPortfolioAccess"})
}

func handleListPortfolios(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListPortfoliosInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListPortfolios business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListPortfolios"})
}

func handleListPortfoliosForProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListPortfoliosForProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListPortfoliosForProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListPortfoliosForProduct"})
}

func handleListPrincipalsForPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListPrincipalsForPortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListPrincipalsForPortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListPrincipalsForPortfolio"})
}

func handleListProvisionedProductPlans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListProvisionedProductPlansInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListProvisionedProductPlans business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListProvisionedProductPlans"})
}

func handleListProvisioningArtifacts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListProvisioningArtifactsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListProvisioningArtifacts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListProvisioningArtifacts"})
}

func handleListProvisioningArtifactsForServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListProvisioningArtifactsForServiceActionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListProvisioningArtifactsForServiceAction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListProvisioningArtifactsForServiceAction"})
}

func handleListRecordHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListRecordHistoryInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListRecordHistory business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListRecordHistory"})
}

func handleListResourcesForTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListResourcesForTagOptionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListResourcesForTagOption business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListResourcesForTagOption"})
}

func handleListServiceActions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListServiceActionsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListServiceActions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListServiceActions"})
}

func handleListServiceActionsForProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListServiceActionsForProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListServiceActionsForProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListServiceActionsForProvisioningArtifact"})
}

func handleListStackInstancesForProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListStackInstancesForProvisionedProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListStackInstancesForProvisionedProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListStackInstancesForProvisionedProduct"})
}

func handleListTagOptions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagOptionsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagOptions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagOptions"})
}

func handleNotifyProvisionProductEngineWorkflowResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req NotifyProvisionProductEngineWorkflowResultInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement NotifyProvisionProductEngineWorkflowResult business logic
	return jsonOK(map[string]any{"status": "ok", "action": "NotifyProvisionProductEngineWorkflowResult"})
}

func handleNotifyTerminateProvisionedProductEngineWorkflowResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req NotifyTerminateProvisionedProductEngineWorkflowResultInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement NotifyTerminateProvisionedProductEngineWorkflowResult business logic
	return jsonOK(map[string]any{"status": "ok", "action": "NotifyTerminateProvisionedProductEngineWorkflowResult"})
}

func handleNotifyUpdateProvisionedProductEngineWorkflowResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req NotifyUpdateProvisionedProductEngineWorkflowResultInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement NotifyUpdateProvisionedProductEngineWorkflowResult business logic
	return jsonOK(map[string]any{"status": "ok", "action": "NotifyUpdateProvisionedProductEngineWorkflowResult"})
}

func handleProvisionProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ProvisionProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ProvisionProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ProvisionProduct"})
}

func handleRejectPortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RejectPortfolioShareInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RejectPortfolioShare business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RejectPortfolioShare"})
}

func handleScanProvisionedProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ScanProvisionedProductsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ScanProvisionedProducts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ScanProvisionedProducts"})
}

func handleSearchProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchProductsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchProducts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchProducts"})
}

func handleSearchProductsAsAdmin(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchProductsAsAdminInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchProductsAsAdmin business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchProductsAsAdmin"})
}

func handleSearchProvisionedProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchProvisionedProductsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchProvisionedProducts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchProvisionedProducts"})
}

func handleTerminateProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TerminateProvisionedProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TerminateProvisionedProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TerminateProvisionedProduct"})
}

func handleUpdateConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateConstraintInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateConstraint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateConstraint"})
}

func handleUpdatePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdatePortfolioInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdatePortfolio business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdatePortfolio"})
}

func handleUpdatePortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdatePortfolioShareInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdatePortfolioShare business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdatePortfolioShare"})
}

func handleUpdateProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateProduct"})
}

func handleUpdateProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateProvisionedProductInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateProvisionedProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateProvisionedProduct"})
}

func handleUpdateProvisionedProductProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateProvisionedProductPropertiesInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateProvisionedProductProperties business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateProvisionedProductProperties"})
}

func handleUpdateProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateProvisioningArtifactInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateProvisioningArtifact business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateProvisioningArtifact"})
}

func handleUpdateServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateServiceActionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateServiceAction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateServiceAction"})
}

func handleUpdateTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTagOptionInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTagOption business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTagOption"})
}

