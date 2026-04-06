package inspector2

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS inspector2 service.
type Service struct {
	store *Store
}

// New returns a new inspector2 Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "inspector2" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "AssociateMember", Method: http.MethodPost, IAMAction: "inspector2:AssociateMember"},
		{Name: "BatchAssociateCodeSecurityScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:BatchAssociateCodeSecurityScanConfiguration"},
		{Name: "BatchDisassociateCodeSecurityScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:BatchDisassociateCodeSecurityScanConfiguration"},
		{Name: "BatchGetAccountStatus", Method: http.MethodPost, IAMAction: "inspector2:BatchGetAccountStatus"},
		{Name: "BatchGetCodeSnippet", Method: http.MethodPost, IAMAction: "inspector2:BatchGetCodeSnippet"},
		{Name: "BatchGetFindingDetails", Method: http.MethodPost, IAMAction: "inspector2:BatchGetFindingDetails"},
		{Name: "BatchGetFreeTrialInfo", Method: http.MethodPost, IAMAction: "inspector2:BatchGetFreeTrialInfo"},
		{Name: "BatchGetMemberEc2DeepInspectionStatus", Method: http.MethodPost, IAMAction: "inspector2:BatchGetMemberEc2DeepInspectionStatus"},
		{Name: "BatchUpdateMemberEc2DeepInspectionStatus", Method: http.MethodPost, IAMAction: "inspector2:BatchUpdateMemberEc2DeepInspectionStatus"},
		{Name: "CancelFindingsReport", Method: http.MethodPost, IAMAction: "inspector2:CancelFindingsReport"},
		{Name: "CancelSbomExport", Method: http.MethodPost, IAMAction: "inspector2:CancelSbomExport"},
		{Name: "CreateCisScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:CreateCisScanConfiguration"},
		{Name: "CreateCodeSecurityIntegration", Method: http.MethodPost, IAMAction: "inspector2:CreateCodeSecurityIntegration"},
		{Name: "CreateCodeSecurityScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:CreateCodeSecurityScanConfiguration"},
		{Name: "CreateFilter", Method: http.MethodPost, IAMAction: "inspector2:CreateFilter"},
		{Name: "CreateFindingsReport", Method: http.MethodPost, IAMAction: "inspector2:CreateFindingsReport"},
		{Name: "CreateSbomExport", Method: http.MethodPost, IAMAction: "inspector2:CreateSbomExport"},
		{Name: "DeleteCisScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:DeleteCisScanConfiguration"},
		{Name: "DeleteCodeSecurityIntegration", Method: http.MethodPost, IAMAction: "inspector2:DeleteCodeSecurityIntegration"},
		{Name: "DeleteCodeSecurityScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:DeleteCodeSecurityScanConfiguration"},
		{Name: "DeleteFilter", Method: http.MethodPost, IAMAction: "inspector2:DeleteFilter"},
		{Name: "DescribeOrganizationConfiguration", Method: http.MethodPost, IAMAction: "inspector2:DescribeOrganizationConfiguration"},
		{Name: "Disable", Method: http.MethodPost, IAMAction: "inspector2:Disable"},
		{Name: "DisableDelegatedAdminAccount", Method: http.MethodPost, IAMAction: "inspector2:DisableDelegatedAdminAccount"},
		{Name: "DisassociateMember", Method: http.MethodPost, IAMAction: "inspector2:DisassociateMember"},
		{Name: "Enable", Method: http.MethodPost, IAMAction: "inspector2:Enable"},
		{Name: "EnableDelegatedAdminAccount", Method: http.MethodPost, IAMAction: "inspector2:EnableDelegatedAdminAccount"},
		{Name: "GetCisScanReport", Method: http.MethodPost, IAMAction: "inspector2:GetCisScanReport"},
		{Name: "GetCisScanResultDetails", Method: http.MethodPost, IAMAction: "inspector2:GetCisScanResultDetails"},
		{Name: "GetClustersForImage", Method: http.MethodPost, IAMAction: "inspector2:GetClustersForImage"},
		{Name: "GetCodeSecurityIntegration", Method: http.MethodPost, IAMAction: "inspector2:GetCodeSecurityIntegration"},
		{Name: "GetCodeSecurityScan", Method: http.MethodPost, IAMAction: "inspector2:GetCodeSecurityScan"},
		{Name: "GetCodeSecurityScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:GetCodeSecurityScanConfiguration"},
		{Name: "GetConfiguration", Method: http.MethodPost, IAMAction: "inspector2:GetConfiguration"},
		{Name: "GetDelegatedAdminAccount", Method: http.MethodPost, IAMAction: "inspector2:GetDelegatedAdminAccount"},
		{Name: "GetEc2DeepInspectionConfiguration", Method: http.MethodPost, IAMAction: "inspector2:GetEc2DeepInspectionConfiguration"},
		{Name: "GetEncryptionKey", Method: http.MethodGet, IAMAction: "inspector2:GetEncryptionKey"},
		{Name: "GetFindingsReportStatus", Method: http.MethodPost, IAMAction: "inspector2:GetFindingsReportStatus"},
		{Name: "GetMember", Method: http.MethodPost, IAMAction: "inspector2:GetMember"},
		{Name: "GetSbomExport", Method: http.MethodPost, IAMAction: "inspector2:GetSbomExport"},
		{Name: "ListAccountPermissions", Method: http.MethodPost, IAMAction: "inspector2:ListAccountPermissions"},
		{Name: "ListCisScanConfigurations", Method: http.MethodPost, IAMAction: "inspector2:ListCisScanConfigurations"},
		{Name: "ListCisScanResultsAggregatedByChecks", Method: http.MethodPost, IAMAction: "inspector2:ListCisScanResultsAggregatedByChecks"},
		{Name: "ListCisScanResultsAggregatedByTargetResource", Method: http.MethodPost, IAMAction: "inspector2:ListCisScanResultsAggregatedByTargetResource"},
		{Name: "ListCisScans", Method: http.MethodPost, IAMAction: "inspector2:ListCisScans"},
		{Name: "ListCodeSecurityIntegrations", Method: http.MethodPost, IAMAction: "inspector2:ListCodeSecurityIntegrations"},
		{Name: "ListCodeSecurityScanConfigurationAssociations", Method: http.MethodPost, IAMAction: "inspector2:ListCodeSecurityScanConfigurationAssociations"},
		{Name: "ListCodeSecurityScanConfigurations", Method: http.MethodPost, IAMAction: "inspector2:ListCodeSecurityScanConfigurations"},
		{Name: "ListCoverage", Method: http.MethodPost, IAMAction: "inspector2:ListCoverage"},
		{Name: "ListCoverageStatistics", Method: http.MethodPost, IAMAction: "inspector2:ListCoverageStatistics"},
		{Name: "ListDelegatedAdminAccounts", Method: http.MethodPost, IAMAction: "inspector2:ListDelegatedAdminAccounts"},
		{Name: "ListFilters", Method: http.MethodPost, IAMAction: "inspector2:ListFilters"},
		{Name: "ListFindingAggregations", Method: http.MethodPost, IAMAction: "inspector2:ListFindingAggregations"},
		{Name: "ListFindings", Method: http.MethodPost, IAMAction: "inspector2:ListFindings"},
		{Name: "ListMembers", Method: http.MethodPost, IAMAction: "inspector2:ListMembers"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "inspector2:ListTagsForResource"},
		{Name: "ListUsageTotals", Method: http.MethodPost, IAMAction: "inspector2:ListUsageTotals"},
		{Name: "ResetEncryptionKey", Method: http.MethodPut, IAMAction: "inspector2:ResetEncryptionKey"},
		{Name: "SearchVulnerabilities", Method: http.MethodPost, IAMAction: "inspector2:SearchVulnerabilities"},
		{Name: "SendCisSessionHealth", Method: http.MethodPut, IAMAction: "inspector2:SendCisSessionHealth"},
		{Name: "SendCisSessionTelemetry", Method: http.MethodPut, IAMAction: "inspector2:SendCisSessionTelemetry"},
		{Name: "StartCisSession", Method: http.MethodPut, IAMAction: "inspector2:StartCisSession"},
		{Name: "StartCodeSecurityScan", Method: http.MethodPost, IAMAction: "inspector2:StartCodeSecurityScan"},
		{Name: "StopCisSession", Method: http.MethodPut, IAMAction: "inspector2:StopCisSession"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "inspector2:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "inspector2:UntagResource"},
		{Name: "UpdateCisScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:UpdateCisScanConfiguration"},
		{Name: "UpdateCodeSecurityIntegration", Method: http.MethodPost, IAMAction: "inspector2:UpdateCodeSecurityIntegration"},
		{Name: "UpdateCodeSecurityScanConfiguration", Method: http.MethodPost, IAMAction: "inspector2:UpdateCodeSecurityScanConfiguration"},
		{Name: "UpdateConfiguration", Method: http.MethodPost, IAMAction: "inspector2:UpdateConfiguration"},
		{Name: "UpdateEc2DeepInspectionConfiguration", Method: http.MethodPost, IAMAction: "inspector2:UpdateEc2DeepInspectionConfiguration"},
		{Name: "UpdateEncryptionKey", Method: http.MethodPut, IAMAction: "inspector2:UpdateEncryptionKey"},
		{Name: "UpdateFilter", Method: http.MethodPost, IAMAction: "inspector2:UpdateFilter"},
		{Name: "UpdateOrgEc2DeepInspectionConfiguration", Method: http.MethodPost, IAMAction: "inspector2:UpdateOrgEc2DeepInspectionConfiguration"},
		{Name: "UpdateOrganizationConfiguration", Method: http.MethodPost, IAMAction: "inspector2:UpdateOrganizationConfiguration"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "AssociateMember":
		return handleAssociateMember(ctx, s.store)
	case "BatchAssociateCodeSecurityScanConfiguration":
		return handleBatchAssociateCodeSecurityScanConfiguration(ctx, s.store)
	case "BatchDisassociateCodeSecurityScanConfiguration":
		return handleBatchDisassociateCodeSecurityScanConfiguration(ctx, s.store)
	case "BatchGetAccountStatus":
		return handleBatchGetAccountStatus(ctx, s.store)
	case "BatchGetCodeSnippet":
		return handleBatchGetCodeSnippet(ctx, s.store)
	case "BatchGetFindingDetails":
		return handleBatchGetFindingDetails(ctx, s.store)
	case "BatchGetFreeTrialInfo":
		return handleBatchGetFreeTrialInfo(ctx, s.store)
	case "BatchGetMemberEc2DeepInspectionStatus":
		return handleBatchGetMemberEc2DeepInspectionStatus(ctx, s.store)
	case "BatchUpdateMemberEc2DeepInspectionStatus":
		return handleBatchUpdateMemberEc2DeepInspectionStatus(ctx, s.store)
	case "CancelFindingsReport":
		return handleCancelFindingsReport(ctx, s.store)
	case "CancelSbomExport":
		return handleCancelSbomExport(ctx, s.store)
	case "CreateCisScanConfiguration":
		return handleCreateCisScanConfiguration(ctx, s.store)
	case "CreateCodeSecurityIntegration":
		return handleCreateCodeSecurityIntegration(ctx, s.store)
	case "CreateCodeSecurityScanConfiguration":
		return handleCreateCodeSecurityScanConfiguration(ctx, s.store)
	case "CreateFilter":
		return handleCreateFilter(ctx, s.store)
	case "CreateFindingsReport":
		return handleCreateFindingsReport(ctx, s.store)
	case "CreateSbomExport":
		return handleCreateSbomExport(ctx, s.store)
	case "DeleteCisScanConfiguration":
		return handleDeleteCisScanConfiguration(ctx, s.store)
	case "DeleteCodeSecurityIntegration":
		return handleDeleteCodeSecurityIntegration(ctx, s.store)
	case "DeleteCodeSecurityScanConfiguration":
		return handleDeleteCodeSecurityScanConfiguration(ctx, s.store)
	case "DeleteFilter":
		return handleDeleteFilter(ctx, s.store)
	case "DescribeOrganizationConfiguration":
		return handleDescribeOrganizationConfiguration(ctx, s.store)
	case "Disable":
		return handleDisable(ctx, s.store)
	case "DisableDelegatedAdminAccount":
		return handleDisableDelegatedAdminAccount(ctx, s.store)
	case "DisassociateMember":
		return handleDisassociateMember(ctx, s.store)
	case "Enable":
		return handleEnable(ctx, s.store)
	case "EnableDelegatedAdminAccount":
		return handleEnableDelegatedAdminAccount(ctx, s.store)
	case "GetCisScanReport":
		return handleGetCisScanReport(ctx, s.store)
	case "GetCisScanResultDetails":
		return handleGetCisScanResultDetails(ctx, s.store)
	case "GetClustersForImage":
		return handleGetClustersForImage(ctx, s.store)
	case "GetCodeSecurityIntegration":
		return handleGetCodeSecurityIntegration(ctx, s.store)
	case "GetCodeSecurityScan":
		return handleGetCodeSecurityScan(ctx, s.store)
	case "GetCodeSecurityScanConfiguration":
		return handleGetCodeSecurityScanConfiguration(ctx, s.store)
	case "GetConfiguration":
		return handleGetConfiguration(ctx, s.store)
	case "GetDelegatedAdminAccount":
		return handleGetDelegatedAdminAccount(ctx, s.store)
	case "GetEc2DeepInspectionConfiguration":
		return handleGetEc2DeepInspectionConfiguration(ctx, s.store)
	case "GetEncryptionKey":
		return handleGetEncryptionKey(ctx, s.store)
	case "GetFindingsReportStatus":
		return handleGetFindingsReportStatus(ctx, s.store)
	case "GetMember":
		return handleGetMember(ctx, s.store)
	case "GetSbomExport":
		return handleGetSbomExport(ctx, s.store)
	case "ListAccountPermissions":
		return handleListAccountPermissions(ctx, s.store)
	case "ListCisScanConfigurations":
		return handleListCisScanConfigurations(ctx, s.store)
	case "ListCisScanResultsAggregatedByChecks":
		return handleListCisScanResultsAggregatedByChecks(ctx, s.store)
	case "ListCisScanResultsAggregatedByTargetResource":
		return handleListCisScanResultsAggregatedByTargetResource(ctx, s.store)
	case "ListCisScans":
		return handleListCisScans(ctx, s.store)
	case "ListCodeSecurityIntegrations":
		return handleListCodeSecurityIntegrations(ctx, s.store)
	case "ListCodeSecurityScanConfigurationAssociations":
		return handleListCodeSecurityScanConfigurationAssociations(ctx, s.store)
	case "ListCodeSecurityScanConfigurations":
		return handleListCodeSecurityScanConfigurations(ctx, s.store)
	case "ListCoverage":
		return handleListCoverage(ctx, s.store)
	case "ListCoverageStatistics":
		return handleListCoverageStatistics(ctx, s.store)
	case "ListDelegatedAdminAccounts":
		return handleListDelegatedAdminAccounts(ctx, s.store)
	case "ListFilters":
		return handleListFilters(ctx, s.store)
	case "ListFindingAggregations":
		return handleListFindingAggregations(ctx, s.store)
	case "ListFindings":
		return handleListFindings(ctx, s.store)
	case "ListMembers":
		return handleListMembers(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListUsageTotals":
		return handleListUsageTotals(ctx, s.store)
	case "ResetEncryptionKey":
		return handleResetEncryptionKey(ctx, s.store)
	case "SearchVulnerabilities":
		return handleSearchVulnerabilities(ctx, s.store)
	case "SendCisSessionHealth":
		return handleSendCisSessionHealth(ctx, s.store)
	case "SendCisSessionTelemetry":
		return handleSendCisSessionTelemetry(ctx, s.store)
	case "StartCisSession":
		return handleStartCisSession(ctx, s.store)
	case "StartCodeSecurityScan":
		return handleStartCodeSecurityScan(ctx, s.store)
	case "StopCisSession":
		return handleStopCisSession(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateCisScanConfiguration":
		return handleUpdateCisScanConfiguration(ctx, s.store)
	case "UpdateCodeSecurityIntegration":
		return handleUpdateCodeSecurityIntegration(ctx, s.store)
	case "UpdateCodeSecurityScanConfiguration":
		return handleUpdateCodeSecurityScanConfiguration(ctx, s.store)
	case "UpdateConfiguration":
		return handleUpdateConfiguration(ctx, s.store)
	case "UpdateEc2DeepInspectionConfiguration":
		return handleUpdateEc2DeepInspectionConfiguration(ctx, s.store)
	case "UpdateEncryptionKey":
		return handleUpdateEncryptionKey(ctx, s.store)
	case "UpdateFilter":
		return handleUpdateFilter(ctx, s.store)
	case "UpdateOrgEc2DeepInspectionConfiguration":
		return handleUpdateOrgEc2DeepInspectionConfiguration(ctx, s.store)
	case "UpdateOrganizationConfiguration":
		return handleUpdateOrganizationConfiguration(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
