package guardduty

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS guardduty service.
type Service struct {
	store *Store
}

// New returns a new guardduty Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "guardduty" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "AcceptAdministratorInvitation", Method: http.MethodPost, IAMAction: "guardduty:AcceptAdministratorInvitation"},
		{Name: "AcceptInvitation", Method: http.MethodPost, IAMAction: "guardduty:AcceptInvitation"},
		{Name: "ArchiveFindings", Method: http.MethodPost, IAMAction: "guardduty:ArchiveFindings"},
		{Name: "CreateDetector", Method: http.MethodPost, IAMAction: "guardduty:CreateDetector"},
		{Name: "CreateFilter", Method: http.MethodPost, IAMAction: "guardduty:CreateFilter"},
		{Name: "CreateIPSet", Method: http.MethodPost, IAMAction: "guardduty:CreateIPSet"},
		{Name: "CreateMalwareProtectionPlan", Method: http.MethodPost, IAMAction: "guardduty:CreateMalwareProtectionPlan"},
		{Name: "CreateMembers", Method: http.MethodPost, IAMAction: "guardduty:CreateMembers"},
		{Name: "CreatePublishingDestination", Method: http.MethodPost, IAMAction: "guardduty:CreatePublishingDestination"},
		{Name: "CreateSampleFindings", Method: http.MethodPost, IAMAction: "guardduty:CreateSampleFindings"},
		{Name: "CreateThreatEntitySet", Method: http.MethodPost, IAMAction: "guardduty:CreateThreatEntitySet"},
		{Name: "CreateThreatIntelSet", Method: http.MethodPost, IAMAction: "guardduty:CreateThreatIntelSet"},
		{Name: "CreateTrustedEntitySet", Method: http.MethodPost, IAMAction: "guardduty:CreateTrustedEntitySet"},
		{Name: "DeclineInvitations", Method: http.MethodPost, IAMAction: "guardduty:DeclineInvitations"},
		{Name: "DeleteDetector", Method: http.MethodDelete, IAMAction: "guardduty:DeleteDetector"},
		{Name: "DeleteFilter", Method: http.MethodDelete, IAMAction: "guardduty:DeleteFilter"},
		{Name: "DeleteIPSet", Method: http.MethodDelete, IAMAction: "guardduty:DeleteIPSet"},
		{Name: "DeleteInvitations", Method: http.MethodPost, IAMAction: "guardduty:DeleteInvitations"},
		{Name: "DeleteMalwareProtectionPlan", Method: http.MethodDelete, IAMAction: "guardduty:DeleteMalwareProtectionPlan"},
		{Name: "DeleteMembers", Method: http.MethodPost, IAMAction: "guardduty:DeleteMembers"},
		{Name: "DeletePublishingDestination", Method: http.MethodDelete, IAMAction: "guardduty:DeletePublishingDestination"},
		{Name: "DeleteThreatEntitySet", Method: http.MethodDelete, IAMAction: "guardduty:DeleteThreatEntitySet"},
		{Name: "DeleteThreatIntelSet", Method: http.MethodDelete, IAMAction: "guardduty:DeleteThreatIntelSet"},
		{Name: "DeleteTrustedEntitySet", Method: http.MethodDelete, IAMAction: "guardduty:DeleteTrustedEntitySet"},
		{Name: "DescribeMalwareScans", Method: http.MethodPost, IAMAction: "guardduty:DescribeMalwareScans"},
		{Name: "DescribeOrganizationConfiguration", Method: http.MethodGet, IAMAction: "guardduty:DescribeOrganizationConfiguration"},
		{Name: "DescribePublishingDestination", Method: http.MethodGet, IAMAction: "guardduty:DescribePublishingDestination"},
		{Name: "DisableOrganizationAdminAccount", Method: http.MethodPost, IAMAction: "guardduty:DisableOrganizationAdminAccount"},
		{Name: "DisassociateFromAdministratorAccount", Method: http.MethodPost, IAMAction: "guardduty:DisassociateFromAdministratorAccount"},
		{Name: "DisassociateFromMasterAccount", Method: http.MethodPost, IAMAction: "guardduty:DisassociateFromMasterAccount"},
		{Name: "DisassociateMembers", Method: http.MethodPost, IAMAction: "guardduty:DisassociateMembers"},
		{Name: "EnableOrganizationAdminAccount", Method: http.MethodPost, IAMAction: "guardduty:EnableOrganizationAdminAccount"},
		{Name: "GetAdministratorAccount", Method: http.MethodGet, IAMAction: "guardduty:GetAdministratorAccount"},
		{Name: "GetCoverageStatistics", Method: http.MethodPost, IAMAction: "guardduty:GetCoverageStatistics"},
		{Name: "GetDetector", Method: http.MethodGet, IAMAction: "guardduty:GetDetector"},
		{Name: "GetFilter", Method: http.MethodGet, IAMAction: "guardduty:GetFilter"},
		{Name: "GetFindings", Method: http.MethodPost, IAMAction: "guardduty:GetFindings"},
		{Name: "GetFindingsStatistics", Method: http.MethodPost, IAMAction: "guardduty:GetFindingsStatistics"},
		{Name: "GetIPSet", Method: http.MethodGet, IAMAction: "guardduty:GetIPSet"},
		{Name: "GetInvitationsCount", Method: http.MethodGet, IAMAction: "guardduty:GetInvitationsCount"},
		{Name: "GetMalwareProtectionPlan", Method: http.MethodGet, IAMAction: "guardduty:GetMalwareProtectionPlan"},
		{Name: "GetMalwareScan", Method: http.MethodGet, IAMAction: "guardduty:GetMalwareScan"},
		{Name: "GetMalwareScanSettings", Method: http.MethodGet, IAMAction: "guardduty:GetMalwareScanSettings"},
		{Name: "GetMasterAccount", Method: http.MethodGet, IAMAction: "guardduty:GetMasterAccount"},
		{Name: "GetMemberDetectors", Method: http.MethodPost, IAMAction: "guardduty:GetMemberDetectors"},
		{Name: "GetMembers", Method: http.MethodPost, IAMAction: "guardduty:GetMembers"},
		{Name: "GetOrganizationStatistics", Method: http.MethodGet, IAMAction: "guardduty:GetOrganizationStatistics"},
		{Name: "GetRemainingFreeTrialDays", Method: http.MethodPost, IAMAction: "guardduty:GetRemainingFreeTrialDays"},
		{Name: "GetThreatEntitySet", Method: http.MethodGet, IAMAction: "guardduty:GetThreatEntitySet"},
		{Name: "GetThreatIntelSet", Method: http.MethodGet, IAMAction: "guardduty:GetThreatIntelSet"},
		{Name: "GetTrustedEntitySet", Method: http.MethodGet, IAMAction: "guardduty:GetTrustedEntitySet"},
		{Name: "GetUsageStatistics", Method: http.MethodPost, IAMAction: "guardduty:GetUsageStatistics"},
		{Name: "InviteMembers", Method: http.MethodPost, IAMAction: "guardduty:InviteMembers"},
		{Name: "ListCoverage", Method: http.MethodPost, IAMAction: "guardduty:ListCoverage"},
		{Name: "ListDetectors", Method: http.MethodGet, IAMAction: "guardduty:ListDetectors"},
		{Name: "ListFilters", Method: http.MethodGet, IAMAction: "guardduty:ListFilters"},
		{Name: "ListFindings", Method: http.MethodPost, IAMAction: "guardduty:ListFindings"},
		{Name: "ListIPSets", Method: http.MethodGet, IAMAction: "guardduty:ListIPSets"},
		{Name: "ListInvitations", Method: http.MethodGet, IAMAction: "guardduty:ListInvitations"},
		{Name: "ListMalwareProtectionPlans", Method: http.MethodGet, IAMAction: "guardduty:ListMalwareProtectionPlans"},
		{Name: "ListMalwareScans", Method: http.MethodPost, IAMAction: "guardduty:ListMalwareScans"},
		{Name: "ListMembers", Method: http.MethodGet, IAMAction: "guardduty:ListMembers"},
		{Name: "ListOrganizationAdminAccounts", Method: http.MethodGet, IAMAction: "guardduty:ListOrganizationAdminAccounts"},
		{Name: "ListPublishingDestinations", Method: http.MethodGet, IAMAction: "guardduty:ListPublishingDestinations"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "guardduty:ListTagsForResource"},
		{Name: "ListThreatEntitySets", Method: http.MethodGet, IAMAction: "guardduty:ListThreatEntitySets"},
		{Name: "ListThreatIntelSets", Method: http.MethodGet, IAMAction: "guardduty:ListThreatIntelSets"},
		{Name: "ListTrustedEntitySets", Method: http.MethodGet, IAMAction: "guardduty:ListTrustedEntitySets"},
		{Name: "SendObjectMalwareScan", Method: http.MethodPost, IAMAction: "guardduty:SendObjectMalwareScan"},
		{Name: "StartMalwareScan", Method: http.MethodPost, IAMAction: "guardduty:StartMalwareScan"},
		{Name: "StartMonitoringMembers", Method: http.MethodPost, IAMAction: "guardduty:StartMonitoringMembers"},
		{Name: "StopMonitoringMembers", Method: http.MethodPost, IAMAction: "guardduty:StopMonitoringMembers"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "guardduty:TagResource"},
		{Name: "UnarchiveFindings", Method: http.MethodPost, IAMAction: "guardduty:UnarchiveFindings"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "guardduty:UntagResource"},
		{Name: "UpdateDetector", Method: http.MethodPost, IAMAction: "guardduty:UpdateDetector"},
		{Name: "UpdateFilter", Method: http.MethodPost, IAMAction: "guardduty:UpdateFilter"},
		{Name: "UpdateFindingsFeedback", Method: http.MethodPost, IAMAction: "guardduty:UpdateFindingsFeedback"},
		{Name: "UpdateIPSet", Method: http.MethodPost, IAMAction: "guardduty:UpdateIPSet"},
		{Name: "UpdateMalwareProtectionPlan", Method: http.MethodPatch, IAMAction: "guardduty:UpdateMalwareProtectionPlan"},
		{Name: "UpdateMalwareScanSettings", Method: http.MethodPost, IAMAction: "guardduty:UpdateMalwareScanSettings"},
		{Name: "UpdateMemberDetectors", Method: http.MethodPost, IAMAction: "guardduty:UpdateMemberDetectors"},
		{Name: "UpdateOrganizationConfiguration", Method: http.MethodPost, IAMAction: "guardduty:UpdateOrganizationConfiguration"},
		{Name: "UpdatePublishingDestination", Method: http.MethodPost, IAMAction: "guardduty:UpdatePublishingDestination"},
		{Name: "UpdateThreatEntitySet", Method: http.MethodPost, IAMAction: "guardduty:UpdateThreatEntitySet"},
		{Name: "UpdateThreatIntelSet", Method: http.MethodPost, IAMAction: "guardduty:UpdateThreatIntelSet"},
		{Name: "UpdateTrustedEntitySet", Method: http.MethodPost, IAMAction: "guardduty:UpdateTrustedEntitySet"},
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
	case "ArchiveFindings":
		return handleArchiveFindings(ctx, s.store)
	case "CreateDetector":
		return handleCreateDetector(ctx, s.store)
	case "CreateFilter":
		return handleCreateFilter(ctx, s.store)
	case "CreateIPSet":
		return handleCreateIPSet(ctx, s.store)
	case "CreateMalwareProtectionPlan":
		return handleCreateMalwareProtectionPlan(ctx, s.store)
	case "CreateMembers":
		return handleCreateMembers(ctx, s.store)
	case "CreatePublishingDestination":
		return handleCreatePublishingDestination(ctx, s.store)
	case "CreateSampleFindings":
		return handleCreateSampleFindings(ctx, s.store)
	case "CreateThreatEntitySet":
		return handleCreateThreatEntitySet(ctx, s.store)
	case "CreateThreatIntelSet":
		return handleCreateThreatIntelSet(ctx, s.store)
	case "CreateTrustedEntitySet":
		return handleCreateTrustedEntitySet(ctx, s.store)
	case "DeclineInvitations":
		return handleDeclineInvitations(ctx, s.store)
	case "DeleteDetector":
		return handleDeleteDetector(ctx, s.store)
	case "DeleteFilter":
		return handleDeleteFilter(ctx, s.store)
	case "DeleteIPSet":
		return handleDeleteIPSet(ctx, s.store)
	case "DeleteInvitations":
		return handleDeleteInvitations(ctx, s.store)
	case "DeleteMalwareProtectionPlan":
		return handleDeleteMalwareProtectionPlan(ctx, s.store)
	case "DeleteMembers":
		return handleDeleteMembers(ctx, s.store)
	case "DeletePublishingDestination":
		return handleDeletePublishingDestination(ctx, s.store)
	case "DeleteThreatEntitySet":
		return handleDeleteThreatEntitySet(ctx, s.store)
	case "DeleteThreatIntelSet":
		return handleDeleteThreatIntelSet(ctx, s.store)
	case "DeleteTrustedEntitySet":
		return handleDeleteTrustedEntitySet(ctx, s.store)
	case "DescribeMalwareScans":
		return handleDescribeMalwareScans(ctx, s.store)
	case "DescribeOrganizationConfiguration":
		return handleDescribeOrganizationConfiguration(ctx, s.store)
	case "DescribePublishingDestination":
		return handleDescribePublishingDestination(ctx, s.store)
	case "DisableOrganizationAdminAccount":
		return handleDisableOrganizationAdminAccount(ctx, s.store)
	case "DisassociateFromAdministratorAccount":
		return handleDisassociateFromAdministratorAccount(ctx, s.store)
	case "DisassociateFromMasterAccount":
		return handleDisassociateFromMasterAccount(ctx, s.store)
	case "DisassociateMembers":
		return handleDisassociateMembers(ctx, s.store)
	case "EnableOrganizationAdminAccount":
		return handleEnableOrganizationAdminAccount(ctx, s.store)
	case "GetAdministratorAccount":
		return handleGetAdministratorAccount(ctx, s.store)
	case "GetCoverageStatistics":
		return handleGetCoverageStatistics(ctx, s.store)
	case "GetDetector":
		return handleGetDetector(ctx, s.store)
	case "GetFilter":
		return handleGetFilter(ctx, s.store)
	case "GetFindings":
		return handleGetFindings(ctx, s.store)
	case "GetFindingsStatistics":
		return handleGetFindingsStatistics(ctx, s.store)
	case "GetIPSet":
		return handleGetIPSet(ctx, s.store)
	case "GetInvitationsCount":
		return handleGetInvitationsCount(ctx, s.store)
	case "GetMalwareProtectionPlan":
		return handleGetMalwareProtectionPlan(ctx, s.store)
	case "GetMalwareScan":
		return handleGetMalwareScan(ctx, s.store)
	case "GetMalwareScanSettings":
		return handleGetMalwareScanSettings(ctx, s.store)
	case "GetMasterAccount":
		return handleGetMasterAccount(ctx, s.store)
	case "GetMemberDetectors":
		return handleGetMemberDetectors(ctx, s.store)
	case "GetMembers":
		return handleGetMembers(ctx, s.store)
	case "GetOrganizationStatistics":
		return handleGetOrganizationStatistics(ctx, s.store)
	case "GetRemainingFreeTrialDays":
		return handleGetRemainingFreeTrialDays(ctx, s.store)
	case "GetThreatEntitySet":
		return handleGetThreatEntitySet(ctx, s.store)
	case "GetThreatIntelSet":
		return handleGetThreatIntelSet(ctx, s.store)
	case "GetTrustedEntitySet":
		return handleGetTrustedEntitySet(ctx, s.store)
	case "GetUsageStatistics":
		return handleGetUsageStatistics(ctx, s.store)
	case "InviteMembers":
		return handleInviteMembers(ctx, s.store)
	case "ListCoverage":
		return handleListCoverage(ctx, s.store)
	case "ListDetectors":
		return handleListDetectors(ctx, s.store)
	case "ListFilters":
		return handleListFilters(ctx, s.store)
	case "ListFindings":
		return handleListFindings(ctx, s.store)
	case "ListIPSets":
		return handleListIPSets(ctx, s.store)
	case "ListInvitations":
		return handleListInvitations(ctx, s.store)
	case "ListMalwareProtectionPlans":
		return handleListMalwareProtectionPlans(ctx, s.store)
	case "ListMalwareScans":
		return handleListMalwareScans(ctx, s.store)
	case "ListMembers":
		return handleListMembers(ctx, s.store)
	case "ListOrganizationAdminAccounts":
		return handleListOrganizationAdminAccounts(ctx, s.store)
	case "ListPublishingDestinations":
		return handleListPublishingDestinations(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListThreatEntitySets":
		return handleListThreatEntitySets(ctx, s.store)
	case "ListThreatIntelSets":
		return handleListThreatIntelSets(ctx, s.store)
	case "ListTrustedEntitySets":
		return handleListTrustedEntitySets(ctx, s.store)
	case "SendObjectMalwareScan":
		return handleSendObjectMalwareScan(ctx, s.store)
	case "StartMalwareScan":
		return handleStartMalwareScan(ctx, s.store)
	case "StartMonitoringMembers":
		return handleStartMonitoringMembers(ctx, s.store)
	case "StopMonitoringMembers":
		return handleStopMonitoringMembers(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UnarchiveFindings":
		return handleUnarchiveFindings(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateDetector":
		return handleUpdateDetector(ctx, s.store)
	case "UpdateFilter":
		return handleUpdateFilter(ctx, s.store)
	case "UpdateFindingsFeedback":
		return handleUpdateFindingsFeedback(ctx, s.store)
	case "UpdateIPSet":
		return handleUpdateIPSet(ctx, s.store)
	case "UpdateMalwareProtectionPlan":
		return handleUpdateMalwareProtectionPlan(ctx, s.store)
	case "UpdateMalwareScanSettings":
		return handleUpdateMalwareScanSettings(ctx, s.store)
	case "UpdateMemberDetectors":
		return handleUpdateMemberDetectors(ctx, s.store)
	case "UpdateOrganizationConfiguration":
		return handleUpdateOrganizationConfiguration(ctx, s.store)
	case "UpdatePublishingDestination":
		return handleUpdatePublishingDestination(ctx, s.store)
	case "UpdateThreatEntitySet":
		return handleUpdateThreatEntitySet(ctx, s.store)
	case "UpdateThreatIntelSet":
		return handleUpdateThreatIntelSet(ctx, s.store)
	case "UpdateTrustedEntitySet":
		return handleUpdateTrustedEntitySet(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
