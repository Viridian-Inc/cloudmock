package securityhub

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS securityhub service.
type Service struct {
	store *Store
}

// New returns a new securityhub Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "securityhub" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "AcceptAdministratorInvitation", Method: http.MethodPost, IAMAction: "securityhub:AcceptAdministratorInvitation"},
		{Name: "AcceptInvitation", Method: http.MethodPost, IAMAction: "securityhub:AcceptInvitation"},
		{Name: "BatchDeleteAutomationRules", Method: http.MethodPost, IAMAction: "securityhub:BatchDeleteAutomationRules"},
		{Name: "BatchDisableStandards", Method: http.MethodPost, IAMAction: "securityhub:BatchDisableStandards"},
		{Name: "BatchEnableStandards", Method: http.MethodPost, IAMAction: "securityhub:BatchEnableStandards"},
		{Name: "BatchGetAutomationRules", Method: http.MethodPost, IAMAction: "securityhub:BatchGetAutomationRules"},
		{Name: "BatchGetConfigurationPolicyAssociations", Method: http.MethodPost, IAMAction: "securityhub:BatchGetConfigurationPolicyAssociations"},
		{Name: "BatchGetSecurityControls", Method: http.MethodPost, IAMAction: "securityhub:BatchGetSecurityControls"},
		{Name: "BatchGetStandardsControlAssociations", Method: http.MethodPost, IAMAction: "securityhub:BatchGetStandardsControlAssociations"},
		{Name: "BatchImportFindings", Method: http.MethodPost, IAMAction: "securityhub:BatchImportFindings"},
		{Name: "BatchUpdateAutomationRules", Method: http.MethodPatch, IAMAction: "securityhub:BatchUpdateAutomationRules"},
		{Name: "BatchUpdateFindings", Method: http.MethodPatch, IAMAction: "securityhub:BatchUpdateFindings"},
		{Name: "BatchUpdateFindingsV2", Method: http.MethodPatch, IAMAction: "securityhub:BatchUpdateFindingsV2"},
		{Name: "BatchUpdateStandardsControlAssociations", Method: http.MethodPatch, IAMAction: "securityhub:BatchUpdateStandardsControlAssociations"},
		{Name: "CreateActionTarget", Method: http.MethodPost, IAMAction: "securityhub:CreateActionTarget"},
		{Name: "CreateAggregatorV2", Method: http.MethodPost, IAMAction: "securityhub:CreateAggregatorV2"},
		{Name: "CreateAutomationRule", Method: http.MethodPost, IAMAction: "securityhub:CreateAutomationRule"},
		{Name: "CreateAutomationRuleV2", Method: http.MethodPost, IAMAction: "securityhub:CreateAutomationRuleV2"},
		{Name: "CreateConfigurationPolicy", Method: http.MethodPost, IAMAction: "securityhub:CreateConfigurationPolicy"},
		{Name: "CreateConnectorV2", Method: http.MethodPost, IAMAction: "securityhub:CreateConnectorV2"},
		{Name: "CreateFindingAggregator", Method: http.MethodPost, IAMAction: "securityhub:CreateFindingAggregator"},
		{Name: "CreateInsight", Method: http.MethodPost, IAMAction: "securityhub:CreateInsight"},
		{Name: "CreateMembers", Method: http.MethodPost, IAMAction: "securityhub:CreateMembers"},
		{Name: "CreateTicketV2", Method: http.MethodPost, IAMAction: "securityhub:CreateTicketV2"},
		{Name: "DeclineInvitations", Method: http.MethodPost, IAMAction: "securityhub:DeclineInvitations"},
		{Name: "DeleteActionTarget", Method: http.MethodDelete, IAMAction: "securityhub:DeleteActionTarget"},
		{Name: "DeleteAggregatorV2", Method: http.MethodDelete, IAMAction: "securityhub:DeleteAggregatorV2"},
		{Name: "DeleteAutomationRuleV2", Method: http.MethodDelete, IAMAction: "securityhub:DeleteAutomationRuleV2"},
		{Name: "DeleteConfigurationPolicy", Method: http.MethodDelete, IAMAction: "securityhub:DeleteConfigurationPolicy"},
		{Name: "DeleteConnectorV2", Method: http.MethodDelete, IAMAction: "securityhub:DeleteConnectorV2"},
		{Name: "DeleteFindingAggregator", Method: http.MethodDelete, IAMAction: "securityhub:DeleteFindingAggregator"},
		{Name: "DeleteInsight", Method: http.MethodDelete, IAMAction: "securityhub:DeleteInsight"},
		{Name: "DeleteInvitations", Method: http.MethodPost, IAMAction: "securityhub:DeleteInvitations"},
		{Name: "DeleteMembers", Method: http.MethodPost, IAMAction: "securityhub:DeleteMembers"},
		{Name: "DescribeActionTargets", Method: http.MethodPost, IAMAction: "securityhub:DescribeActionTargets"},
		{Name: "DescribeHub", Method: http.MethodGet, IAMAction: "securityhub:DescribeHub"},
		{Name: "DescribeOrganizationConfiguration", Method: http.MethodGet, IAMAction: "securityhub:DescribeOrganizationConfiguration"},
		{Name: "DescribeProducts", Method: http.MethodGet, IAMAction: "securityhub:DescribeProducts"},
		{Name: "DescribeProductsV2", Method: http.MethodGet, IAMAction: "securityhub:DescribeProductsV2"},
		{Name: "DescribeSecurityHubV2", Method: http.MethodGet, IAMAction: "securityhub:DescribeSecurityHubV2"},
		{Name: "DescribeStandards", Method: http.MethodGet, IAMAction: "securityhub:DescribeStandards"},
		{Name: "DescribeStandardsControls", Method: http.MethodGet, IAMAction: "securityhub:DescribeStandardsControls"},
		{Name: "DisableImportFindingsForProduct", Method: http.MethodDelete, IAMAction: "securityhub:DisableImportFindingsForProduct"},
		{Name: "DisableOrganizationAdminAccount", Method: http.MethodPost, IAMAction: "securityhub:DisableOrganizationAdminAccount"},
		{Name: "DisableSecurityHub", Method: http.MethodDelete, IAMAction: "securityhub:DisableSecurityHub"},
		{Name: "DisableSecurityHubV2", Method: http.MethodDelete, IAMAction: "securityhub:DisableSecurityHubV2"},
		{Name: "DisassociateFromAdministratorAccount", Method: http.MethodPost, IAMAction: "securityhub:DisassociateFromAdministratorAccount"},
		{Name: "DisassociateFromMasterAccount", Method: http.MethodPost, IAMAction: "securityhub:DisassociateFromMasterAccount"},
		{Name: "DisassociateMembers", Method: http.MethodPost, IAMAction: "securityhub:DisassociateMembers"},
		{Name: "EnableImportFindingsForProduct", Method: http.MethodPost, IAMAction: "securityhub:EnableImportFindingsForProduct"},
		{Name: "EnableOrganizationAdminAccount", Method: http.MethodPost, IAMAction: "securityhub:EnableOrganizationAdminAccount"},
		{Name: "EnableSecurityHub", Method: http.MethodPost, IAMAction: "securityhub:EnableSecurityHub"},
		{Name: "EnableSecurityHubV2", Method: http.MethodPost, IAMAction: "securityhub:EnableSecurityHubV2"},
		{Name: "GetAdministratorAccount", Method: http.MethodGet, IAMAction: "securityhub:GetAdministratorAccount"},
		{Name: "GetAggregatorV2", Method: http.MethodGet, IAMAction: "securityhub:GetAggregatorV2"},
		{Name: "GetAutomationRuleV2", Method: http.MethodGet, IAMAction: "securityhub:GetAutomationRuleV2"},
		{Name: "GetConfigurationPolicy", Method: http.MethodGet, IAMAction: "securityhub:GetConfigurationPolicy"},
		{Name: "GetConfigurationPolicyAssociation", Method: http.MethodPost, IAMAction: "securityhub:GetConfigurationPolicyAssociation"},
		{Name: "GetConnectorV2", Method: http.MethodGet, IAMAction: "securityhub:GetConnectorV2"},
		{Name: "GetEnabledStandards", Method: http.MethodPost, IAMAction: "securityhub:GetEnabledStandards"},
		{Name: "GetFindingAggregator", Method: http.MethodGet, IAMAction: "securityhub:GetFindingAggregator"},
		{Name: "GetFindingHistory", Method: http.MethodPost, IAMAction: "securityhub:GetFindingHistory"},
		{Name: "GetFindingStatisticsV2", Method: http.MethodPost, IAMAction: "securityhub:GetFindingStatisticsV2"},
		{Name: "GetFindings", Method: http.MethodPost, IAMAction: "securityhub:GetFindings"},
		{Name: "GetFindingsTrendsV2", Method: http.MethodPost, IAMAction: "securityhub:GetFindingsTrendsV2"},
		{Name: "GetFindingsV2", Method: http.MethodPost, IAMAction: "securityhub:GetFindingsV2"},
		{Name: "GetInsightResults", Method: http.MethodGet, IAMAction: "securityhub:GetInsightResults"},
		{Name: "GetInsights", Method: http.MethodPost, IAMAction: "securityhub:GetInsights"},
		{Name: "GetInvitationsCount", Method: http.MethodGet, IAMAction: "securityhub:GetInvitationsCount"},
		{Name: "GetMasterAccount", Method: http.MethodGet, IAMAction: "securityhub:GetMasterAccount"},
		{Name: "GetMembers", Method: http.MethodPost, IAMAction: "securityhub:GetMembers"},
		{Name: "GetResourcesStatisticsV2", Method: http.MethodPost, IAMAction: "securityhub:GetResourcesStatisticsV2"},
		{Name: "GetResourcesTrendsV2", Method: http.MethodPost, IAMAction: "securityhub:GetResourcesTrendsV2"},
		{Name: "GetResourcesV2", Method: http.MethodPost, IAMAction: "securityhub:GetResourcesV2"},
		{Name: "GetSecurityControlDefinition", Method: http.MethodGet, IAMAction: "securityhub:GetSecurityControlDefinition"},
		{Name: "InviteMembers", Method: http.MethodPost, IAMAction: "securityhub:InviteMembers"},
		{Name: "ListAggregatorsV2", Method: http.MethodGet, IAMAction: "securityhub:ListAggregatorsV2"},
		{Name: "ListAutomationRules", Method: http.MethodGet, IAMAction: "securityhub:ListAutomationRules"},
		{Name: "ListAutomationRulesV2", Method: http.MethodGet, IAMAction: "securityhub:ListAutomationRulesV2"},
		{Name: "ListConfigurationPolicies", Method: http.MethodGet, IAMAction: "securityhub:ListConfigurationPolicies"},
		{Name: "ListConfigurationPolicyAssociations", Method: http.MethodPost, IAMAction: "securityhub:ListConfigurationPolicyAssociations"},
		{Name: "ListConnectorsV2", Method: http.MethodGet, IAMAction: "securityhub:ListConnectorsV2"},
		{Name: "ListEnabledProductsForImport", Method: http.MethodGet, IAMAction: "securityhub:ListEnabledProductsForImport"},
		{Name: "ListFindingAggregators", Method: http.MethodGet, IAMAction: "securityhub:ListFindingAggregators"},
		{Name: "ListInvitations", Method: http.MethodGet, IAMAction: "securityhub:ListInvitations"},
		{Name: "ListMembers", Method: http.MethodGet, IAMAction: "securityhub:ListMembers"},
		{Name: "ListOrganizationAdminAccounts", Method: http.MethodGet, IAMAction: "securityhub:ListOrganizationAdminAccounts"},
		{Name: "ListSecurityControlDefinitions", Method: http.MethodGet, IAMAction: "securityhub:ListSecurityControlDefinitions"},
		{Name: "ListStandardsControlAssociations", Method: http.MethodGet, IAMAction: "securityhub:ListStandardsControlAssociations"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "securityhub:ListTagsForResource"},
		{Name: "RegisterConnectorV2", Method: http.MethodPost, IAMAction: "securityhub:RegisterConnectorV2"},
		{Name: "StartConfigurationPolicyAssociation", Method: http.MethodPost, IAMAction: "securityhub:StartConfigurationPolicyAssociation"},
		{Name: "StartConfigurationPolicyDisassociation", Method: http.MethodPost, IAMAction: "securityhub:StartConfigurationPolicyDisassociation"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "securityhub:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "securityhub:UntagResource"},
		{Name: "UpdateActionTarget", Method: http.MethodPatch, IAMAction: "securityhub:UpdateActionTarget"},
		{Name: "UpdateAggregatorV2", Method: http.MethodPatch, IAMAction: "securityhub:UpdateAggregatorV2"},
		{Name: "UpdateAutomationRuleV2", Method: http.MethodPatch, IAMAction: "securityhub:UpdateAutomationRuleV2"},
		{Name: "UpdateConfigurationPolicy", Method: http.MethodPatch, IAMAction: "securityhub:UpdateConfigurationPolicy"},
		{Name: "UpdateConnectorV2", Method: http.MethodPatch, IAMAction: "securityhub:UpdateConnectorV2"},
		{Name: "UpdateFindingAggregator", Method: http.MethodPatch, IAMAction: "securityhub:UpdateFindingAggregator"},
		{Name: "UpdateFindings", Method: http.MethodPatch, IAMAction: "securityhub:UpdateFindings"},
		{Name: "UpdateInsight", Method: http.MethodPatch, IAMAction: "securityhub:UpdateInsight"},
		{Name: "UpdateOrganizationConfiguration", Method: http.MethodPost, IAMAction: "securityhub:UpdateOrganizationConfiguration"},
		{Name: "UpdateSecurityControl", Method: http.MethodPatch, IAMAction: "securityhub:UpdateSecurityControl"},
		{Name: "UpdateSecurityHubConfiguration", Method: http.MethodPatch, IAMAction: "securityhub:UpdateSecurityHubConfiguration"},
		{Name: "UpdateStandardsControl", Method: http.MethodPatch, IAMAction: "securityhub:UpdateStandardsControl"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "AcceptAdministratorInvitation":
		return handleAcceptAdministratorInvitation(ctx, s.store)
	case "AcceptInvitation":
		return handleAcceptInvitation(ctx, s.store)
	case "BatchDeleteAutomationRules":
		return handleBatchDeleteAutomationRules(ctx, s.store)
	case "BatchDisableStandards":
		return handleBatchDisableStandards(ctx, s.store)
	case "BatchEnableStandards":
		return handleBatchEnableStandards(ctx, s.store)
	case "BatchGetAutomationRules":
		return handleBatchGetAutomationRules(ctx, s.store)
	case "BatchGetConfigurationPolicyAssociations":
		return handleBatchGetConfigurationPolicyAssociations(ctx, s.store)
	case "BatchGetSecurityControls":
		return handleBatchGetSecurityControls(ctx, s.store)
	case "BatchGetStandardsControlAssociations":
		return handleBatchGetStandardsControlAssociations(ctx, s.store)
	case "BatchImportFindings":
		return handleBatchImportFindings(ctx, s.store)
	case "BatchUpdateAutomationRules":
		return handleBatchUpdateAutomationRules(ctx, s.store)
	case "BatchUpdateFindings":
		return handleBatchUpdateFindings(ctx, s.store)
	case "BatchUpdateFindingsV2":
		return handleBatchUpdateFindingsV2(ctx, s.store)
	case "BatchUpdateStandardsControlAssociations":
		return handleBatchUpdateStandardsControlAssociations(ctx, s.store)
	case "CreateActionTarget":
		return handleCreateActionTarget(ctx, s.store)
	case "CreateAggregatorV2":
		return handleCreateAggregatorV2(ctx, s.store)
	case "CreateAutomationRule":
		return handleCreateAutomationRule(ctx, s.store)
	case "CreateAutomationRuleV2":
		return handleCreateAutomationRuleV2(ctx, s.store)
	case "CreateConfigurationPolicy":
		return handleCreateConfigurationPolicy(ctx, s.store)
	case "CreateConnectorV2":
		return handleCreateConnectorV2(ctx, s.store)
	case "CreateFindingAggregator":
		return handleCreateFindingAggregator(ctx, s.store)
	case "CreateInsight":
		return handleCreateInsight(ctx, s.store)
	case "CreateMembers":
		return handleCreateMembers(ctx, s.store)
	case "CreateTicketV2":
		return handleCreateTicketV2(ctx, s.store)
	case "DeclineInvitations":
		return handleDeclineInvitations(ctx, s.store)
	case "DeleteActionTarget":
		return handleDeleteActionTarget(ctx, s.store)
	case "DeleteAggregatorV2":
		return handleDeleteAggregatorV2(ctx, s.store)
	case "DeleteAutomationRuleV2":
		return handleDeleteAutomationRuleV2(ctx, s.store)
	case "DeleteConfigurationPolicy":
		return handleDeleteConfigurationPolicy(ctx, s.store)
	case "DeleteConnectorV2":
		return handleDeleteConnectorV2(ctx, s.store)
	case "DeleteFindingAggregator":
		return handleDeleteFindingAggregator(ctx, s.store)
	case "DeleteInsight":
		return handleDeleteInsight(ctx, s.store)
	case "DeleteInvitations":
		return handleDeleteInvitations(ctx, s.store)
	case "DeleteMembers":
		return handleDeleteMembers(ctx, s.store)
	case "DescribeActionTargets":
		return handleDescribeActionTargets(ctx, s.store)
	case "DescribeHub":
		return handleDescribeHub(ctx, s.store)
	case "DescribeOrganizationConfiguration":
		return handleDescribeOrganizationConfiguration(ctx, s.store)
	case "DescribeProducts":
		return handleDescribeProducts(ctx, s.store)
	case "DescribeProductsV2":
		return handleDescribeProductsV2(ctx, s.store)
	case "DescribeSecurityHubV2":
		return handleDescribeSecurityHubV2(ctx, s.store)
	case "DescribeStandards":
		return handleDescribeStandards(ctx, s.store)
	case "DescribeStandardsControls":
		return handleDescribeStandardsControls(ctx, s.store)
	case "DisableImportFindingsForProduct":
		return handleDisableImportFindingsForProduct(ctx, s.store)
	case "DisableOrganizationAdminAccount":
		return handleDisableOrganizationAdminAccount(ctx, s.store)
	case "DisableSecurityHub":
		return handleDisableSecurityHub(ctx, s.store)
	case "DisableSecurityHubV2":
		return handleDisableSecurityHubV2(ctx, s.store)
	case "DisassociateFromAdministratorAccount":
		return handleDisassociateFromAdministratorAccount(ctx, s.store)
	case "DisassociateFromMasterAccount":
		return handleDisassociateFromMasterAccount(ctx, s.store)
	case "DisassociateMembers":
		return handleDisassociateMembers(ctx, s.store)
	case "EnableImportFindingsForProduct":
		return handleEnableImportFindingsForProduct(ctx, s.store)
	case "EnableOrganizationAdminAccount":
		return handleEnableOrganizationAdminAccount(ctx, s.store)
	case "EnableSecurityHub":
		return handleEnableSecurityHub(ctx, s.store)
	case "EnableSecurityHubV2":
		return handleEnableSecurityHubV2(ctx, s.store)
	case "GetAdministratorAccount":
		return handleGetAdministratorAccount(ctx, s.store)
	case "GetAggregatorV2":
		return handleGetAggregatorV2(ctx, s.store)
	case "GetAutomationRuleV2":
		return handleGetAutomationRuleV2(ctx, s.store)
	case "GetConfigurationPolicy":
		return handleGetConfigurationPolicy(ctx, s.store)
	case "GetConfigurationPolicyAssociation":
		return handleGetConfigurationPolicyAssociation(ctx, s.store)
	case "GetConnectorV2":
		return handleGetConnectorV2(ctx, s.store)
	case "GetEnabledStandards":
		return handleGetEnabledStandards(ctx, s.store)
	case "GetFindingAggregator":
		return handleGetFindingAggregator(ctx, s.store)
	case "GetFindingHistory":
		return handleGetFindingHistory(ctx, s.store)
	case "GetFindingStatisticsV2":
		return handleGetFindingStatisticsV2(ctx, s.store)
	case "GetFindings":
		return handleGetFindings(ctx, s.store)
	case "GetFindingsTrendsV2":
		return handleGetFindingsTrendsV2(ctx, s.store)
	case "GetFindingsV2":
		return handleGetFindingsV2(ctx, s.store)
	case "GetInsightResults":
		return handleGetInsightResults(ctx, s.store)
	case "GetInsights":
		return handleGetInsights(ctx, s.store)
	case "GetInvitationsCount":
		return handleGetInvitationsCount(ctx, s.store)
	case "GetMasterAccount":
		return handleGetMasterAccount(ctx, s.store)
	case "GetMembers":
		return handleGetMembers(ctx, s.store)
	case "GetResourcesStatisticsV2":
		return handleGetResourcesStatisticsV2(ctx, s.store)
	case "GetResourcesTrendsV2":
		return handleGetResourcesTrendsV2(ctx, s.store)
	case "GetResourcesV2":
		return handleGetResourcesV2(ctx, s.store)
	case "GetSecurityControlDefinition":
		return handleGetSecurityControlDefinition(ctx, s.store)
	case "InviteMembers":
		return handleInviteMembers(ctx, s.store)
	case "ListAggregatorsV2":
		return handleListAggregatorsV2(ctx, s.store)
	case "ListAutomationRules":
		return handleListAutomationRules(ctx, s.store)
	case "ListAutomationRulesV2":
		return handleListAutomationRulesV2(ctx, s.store)
	case "ListConfigurationPolicies":
		return handleListConfigurationPolicies(ctx, s.store)
	case "ListConfigurationPolicyAssociations":
		return handleListConfigurationPolicyAssociations(ctx, s.store)
	case "ListConnectorsV2":
		return handleListConnectorsV2(ctx, s.store)
	case "ListEnabledProductsForImport":
		return handleListEnabledProductsForImport(ctx, s.store)
	case "ListFindingAggregators":
		return handleListFindingAggregators(ctx, s.store)
	case "ListInvitations":
		return handleListInvitations(ctx, s.store)
	case "ListMembers":
		return handleListMembers(ctx, s.store)
	case "ListOrganizationAdminAccounts":
		return handleListOrganizationAdminAccounts(ctx, s.store)
	case "ListSecurityControlDefinitions":
		return handleListSecurityControlDefinitions(ctx, s.store)
	case "ListStandardsControlAssociations":
		return handleListStandardsControlAssociations(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "RegisterConnectorV2":
		return handleRegisterConnectorV2(ctx, s.store)
	case "StartConfigurationPolicyAssociation":
		return handleStartConfigurationPolicyAssociation(ctx, s.store)
	case "StartConfigurationPolicyDisassociation":
		return handleStartConfigurationPolicyDisassociation(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateActionTarget":
		return handleUpdateActionTarget(ctx, s.store)
	case "UpdateAggregatorV2":
		return handleUpdateAggregatorV2(ctx, s.store)
	case "UpdateAutomationRuleV2":
		return handleUpdateAutomationRuleV2(ctx, s.store)
	case "UpdateConfigurationPolicy":
		return handleUpdateConfigurationPolicy(ctx, s.store)
	case "UpdateConnectorV2":
		return handleUpdateConnectorV2(ctx, s.store)
	case "UpdateFindingAggregator":
		return handleUpdateFindingAggregator(ctx, s.store)
	case "UpdateFindings":
		return handleUpdateFindings(ctx, s.store)
	case "UpdateInsight":
		return handleUpdateInsight(ctx, s.store)
	case "UpdateOrganizationConfiguration":
		return handleUpdateOrganizationConfiguration(ctx, s.store)
	case "UpdateSecurityControl":
		return handleUpdateSecurityControl(ctx, s.store)
	case "UpdateSecurityHubConfiguration":
		return handleUpdateSecurityHubConfiguration(ctx, s.store)
	case "UpdateStandardsControl":
		return handleUpdateStandardsControl(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
