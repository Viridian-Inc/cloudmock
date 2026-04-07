package quicksight

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS quicksight service.
type Service struct {
	store *Store
}

// New returns a new quicksight Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "quicksight" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "BatchCreateTopicReviewedAnswer", Method: http.MethodPost, IAMAction: "quicksight:BatchCreateTopicReviewedAnswer"},
		{Name: "BatchDeleteTopicReviewedAnswer", Method: http.MethodPost, IAMAction: "quicksight:BatchDeleteTopicReviewedAnswer"},
		{Name: "CancelIngestion", Method: http.MethodDelete, IAMAction: "quicksight:CancelIngestion"},
		{Name: "CreateAccountCustomization", Method: http.MethodPost, IAMAction: "quicksight:CreateAccountCustomization"},
		{Name: "CreateAccountSubscription", Method: http.MethodPost, IAMAction: "quicksight:CreateAccountSubscription"},
		{Name: "CreateActionConnector", Method: http.MethodPost, IAMAction: "quicksight:CreateActionConnector"},
		{Name: "CreateAnalysis", Method: http.MethodPost, IAMAction: "quicksight:CreateAnalysis"},
		{Name: "CreateBrand", Method: http.MethodPost, IAMAction: "quicksight:CreateBrand"},
		{Name: "CreateCustomPermissions", Method: http.MethodPost, IAMAction: "quicksight:CreateCustomPermissions"},
		{Name: "CreateDashboard", Method: http.MethodPost, IAMAction: "quicksight:CreateDashboard"},
		{Name: "CreateDataSet", Method: http.MethodPost, IAMAction: "quicksight:CreateDataSet"},
		{Name: "CreateDataSource", Method: http.MethodPost, IAMAction: "quicksight:CreateDataSource"},
		{Name: "CreateFolder", Method: http.MethodPost, IAMAction: "quicksight:CreateFolder"},
		{Name: "CreateFolderMembership", Method: http.MethodPut, IAMAction: "quicksight:CreateFolderMembership"},
		{Name: "CreateGroup", Method: http.MethodPost, IAMAction: "quicksight:CreateGroup"},
		{Name: "CreateGroupMembership", Method: http.MethodPut, IAMAction: "quicksight:CreateGroupMembership"},
		{Name: "CreateIAMPolicyAssignment", Method: http.MethodPost, IAMAction: "quicksight:CreateIAMPolicyAssignment"},
		{Name: "CreateIngestion", Method: http.MethodPut, IAMAction: "quicksight:CreateIngestion"},
		{Name: "CreateNamespace", Method: http.MethodPost, IAMAction: "quicksight:CreateNamespace"},
		{Name: "CreateRefreshSchedule", Method: http.MethodPost, IAMAction: "quicksight:CreateRefreshSchedule"},
		{Name: "CreateRoleMembership", Method: http.MethodPost, IAMAction: "quicksight:CreateRoleMembership"},
		{Name: "CreateTemplate", Method: http.MethodPost, IAMAction: "quicksight:CreateTemplate"},
		{Name: "CreateTemplateAlias", Method: http.MethodPost, IAMAction: "quicksight:CreateTemplateAlias"},
		{Name: "CreateTheme", Method: http.MethodPost, IAMAction: "quicksight:CreateTheme"},
		{Name: "CreateThemeAlias", Method: http.MethodPost, IAMAction: "quicksight:CreateThemeAlias"},
		{Name: "CreateTopic", Method: http.MethodPost, IAMAction: "quicksight:CreateTopic"},
		{Name: "CreateTopicRefreshSchedule", Method: http.MethodPost, IAMAction: "quicksight:CreateTopicRefreshSchedule"},
		{Name: "CreateVPCConnection", Method: http.MethodPost, IAMAction: "quicksight:CreateVPCConnection"},
		{Name: "DeleteAccountCustomPermission", Method: http.MethodDelete, IAMAction: "quicksight:DeleteAccountCustomPermission"},
		{Name: "DeleteAccountCustomization", Method: http.MethodDelete, IAMAction: "quicksight:DeleteAccountCustomization"},
		{Name: "DeleteAccountSubscription", Method: http.MethodDelete, IAMAction: "quicksight:DeleteAccountSubscription"},
		{Name: "DeleteActionConnector", Method: http.MethodDelete, IAMAction: "quicksight:DeleteActionConnector"},
		{Name: "DeleteAnalysis", Method: http.MethodDelete, IAMAction: "quicksight:DeleteAnalysis"},
		{Name: "DeleteBrand", Method: http.MethodDelete, IAMAction: "quicksight:DeleteBrand"},
		{Name: "DeleteBrandAssignment", Method: http.MethodDelete, IAMAction: "quicksight:DeleteBrandAssignment"},
		{Name: "DeleteCustomPermissions", Method: http.MethodDelete, IAMAction: "quicksight:DeleteCustomPermissions"},
		{Name: "DeleteDashboard", Method: http.MethodDelete, IAMAction: "quicksight:DeleteDashboard"},
		{Name: "DeleteDataSet", Method: http.MethodDelete, IAMAction: "quicksight:DeleteDataSet"},
		{Name: "DeleteDataSetRefreshProperties", Method: http.MethodDelete, IAMAction: "quicksight:DeleteDataSetRefreshProperties"},
		{Name: "DeleteDataSource", Method: http.MethodDelete, IAMAction: "quicksight:DeleteDataSource"},
		{Name: "DeleteDefaultQBusinessApplication", Method: http.MethodDelete, IAMAction: "quicksight:DeleteDefaultQBusinessApplication"},
		{Name: "DeleteFolder", Method: http.MethodDelete, IAMAction: "quicksight:DeleteFolder"},
		{Name: "DeleteFolderMembership", Method: http.MethodDelete, IAMAction: "quicksight:DeleteFolderMembership"},
		{Name: "DeleteGroup", Method: http.MethodDelete, IAMAction: "quicksight:DeleteGroup"},
		{Name: "DeleteGroupMembership", Method: http.MethodDelete, IAMAction: "quicksight:DeleteGroupMembership"},
		{Name: "DeleteIAMPolicyAssignment", Method: http.MethodDelete, IAMAction: "quicksight:DeleteIAMPolicyAssignment"},
		{Name: "DeleteIdentityPropagationConfig", Method: http.MethodDelete, IAMAction: "quicksight:DeleteIdentityPropagationConfig"},
		{Name: "DeleteNamespace", Method: http.MethodDelete, IAMAction: "quicksight:DeleteNamespace"},
		{Name: "DeleteRefreshSchedule", Method: http.MethodDelete, IAMAction: "quicksight:DeleteRefreshSchedule"},
		{Name: "DeleteRoleCustomPermission", Method: http.MethodDelete, IAMAction: "quicksight:DeleteRoleCustomPermission"},
		{Name: "DeleteRoleMembership", Method: http.MethodDelete, IAMAction: "quicksight:DeleteRoleMembership"},
		{Name: "DeleteTemplate", Method: http.MethodDelete, IAMAction: "quicksight:DeleteTemplate"},
		{Name: "DeleteTemplateAlias", Method: http.MethodDelete, IAMAction: "quicksight:DeleteTemplateAlias"},
		{Name: "DeleteTheme", Method: http.MethodDelete, IAMAction: "quicksight:DeleteTheme"},
		{Name: "DeleteThemeAlias", Method: http.MethodDelete, IAMAction: "quicksight:DeleteThemeAlias"},
		{Name: "DeleteTopic", Method: http.MethodDelete, IAMAction: "quicksight:DeleteTopic"},
		{Name: "DeleteTopicRefreshSchedule", Method: http.MethodDelete, IAMAction: "quicksight:DeleteTopicRefreshSchedule"},
		{Name: "DeleteUser", Method: http.MethodDelete, IAMAction: "quicksight:DeleteUser"},
		{Name: "DeleteUserByPrincipalId", Method: http.MethodDelete, IAMAction: "quicksight:DeleteUserByPrincipalId"},
		{Name: "DeleteUserCustomPermission", Method: http.MethodDelete, IAMAction: "quicksight:DeleteUserCustomPermission"},
		{Name: "DeleteVPCConnection", Method: http.MethodDelete, IAMAction: "quicksight:DeleteVPCConnection"},
		{Name: "DescribeAccountCustomPermission", Method: http.MethodGet, IAMAction: "quicksight:DescribeAccountCustomPermission"},
		{Name: "DescribeAccountCustomization", Method: http.MethodGet, IAMAction: "quicksight:DescribeAccountCustomization"},
		{Name: "DescribeAccountSettings", Method: http.MethodGet, IAMAction: "quicksight:DescribeAccountSettings"},
		{Name: "DescribeAccountSubscription", Method: http.MethodGet, IAMAction: "quicksight:DescribeAccountSubscription"},
		{Name: "DescribeActionConnector", Method: http.MethodGet, IAMAction: "quicksight:DescribeActionConnector"},
		{Name: "DescribeActionConnectorPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeActionConnectorPermissions"},
		{Name: "DescribeAnalysis", Method: http.MethodGet, IAMAction: "quicksight:DescribeAnalysis"},
		{Name: "DescribeAnalysisDefinition", Method: http.MethodGet, IAMAction: "quicksight:DescribeAnalysisDefinition"},
		{Name: "DescribeAnalysisPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeAnalysisPermissions"},
		{Name: "DescribeAssetBundleExportJob", Method: http.MethodGet, IAMAction: "quicksight:DescribeAssetBundleExportJob"},
		{Name: "DescribeAssetBundleImportJob", Method: http.MethodGet, IAMAction: "quicksight:DescribeAssetBundleImportJob"},
		{Name: "DescribeAutomationJob", Method: http.MethodGet, IAMAction: "quicksight:DescribeAutomationJob"},
		{Name: "DescribeBrand", Method: http.MethodGet, IAMAction: "quicksight:DescribeBrand"},
		{Name: "DescribeBrandAssignment", Method: http.MethodGet, IAMAction: "quicksight:DescribeBrandAssignment"},
		{Name: "DescribeBrandPublishedVersion", Method: http.MethodGet, IAMAction: "quicksight:DescribeBrandPublishedVersion"},
		{Name: "DescribeCustomPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeCustomPermissions"},
		{Name: "DescribeDashboard", Method: http.MethodGet, IAMAction: "quicksight:DescribeDashboard"},
		{Name: "DescribeDashboardDefinition", Method: http.MethodGet, IAMAction: "quicksight:DescribeDashboardDefinition"},
		{Name: "DescribeDashboardPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeDashboardPermissions"},
		{Name: "DescribeDashboardSnapshotJob", Method: http.MethodGet, IAMAction: "quicksight:DescribeDashboardSnapshotJob"},
		{Name: "DescribeDashboardSnapshotJobResult", Method: http.MethodGet, IAMAction: "quicksight:DescribeDashboardSnapshotJobResult"},
		{Name: "DescribeDashboardsQAConfiguration", Method: http.MethodGet, IAMAction: "quicksight:DescribeDashboardsQAConfiguration"},
		{Name: "DescribeDataSet", Method: http.MethodGet, IAMAction: "quicksight:DescribeDataSet"},
		{Name: "DescribeDataSetPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeDataSetPermissions"},
		{Name: "DescribeDataSetRefreshProperties", Method: http.MethodGet, IAMAction: "quicksight:DescribeDataSetRefreshProperties"},
		{Name: "DescribeDataSource", Method: http.MethodGet, IAMAction: "quicksight:DescribeDataSource"},
		{Name: "DescribeDataSourcePermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeDataSourcePermissions"},
		{Name: "DescribeDefaultQBusinessApplication", Method: http.MethodGet, IAMAction: "quicksight:DescribeDefaultQBusinessApplication"},
		{Name: "DescribeFolder", Method: http.MethodGet, IAMAction: "quicksight:DescribeFolder"},
		{Name: "DescribeFolderPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeFolderPermissions"},
		{Name: "DescribeFolderResolvedPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeFolderResolvedPermissions"},
		{Name: "DescribeGroup", Method: http.MethodGet, IAMAction: "quicksight:DescribeGroup"},
		{Name: "DescribeGroupMembership", Method: http.MethodGet, IAMAction: "quicksight:DescribeGroupMembership"},
		{Name: "DescribeIAMPolicyAssignment", Method: http.MethodGet, IAMAction: "quicksight:DescribeIAMPolicyAssignment"},
		{Name: "DescribeIngestion", Method: http.MethodGet, IAMAction: "quicksight:DescribeIngestion"},
		{Name: "DescribeIpRestriction", Method: http.MethodGet, IAMAction: "quicksight:DescribeIpRestriction"},
		{Name: "DescribeKeyRegistration", Method: http.MethodGet, IAMAction: "quicksight:DescribeKeyRegistration"},
		{Name: "DescribeNamespace", Method: http.MethodGet, IAMAction: "quicksight:DescribeNamespace"},
		{Name: "DescribeQPersonalizationConfiguration", Method: http.MethodGet, IAMAction: "quicksight:DescribeQPersonalizationConfiguration"},
		{Name: "DescribeQuickSightQSearchConfiguration", Method: http.MethodGet, IAMAction: "quicksight:DescribeQuickSightQSearchConfiguration"},
		{Name: "DescribeRefreshSchedule", Method: http.MethodGet, IAMAction: "quicksight:DescribeRefreshSchedule"},
		{Name: "DescribeRoleCustomPermission", Method: http.MethodGet, IAMAction: "quicksight:DescribeRoleCustomPermission"},
		{Name: "DescribeSelfUpgradeConfiguration", Method: http.MethodGet, IAMAction: "quicksight:DescribeSelfUpgradeConfiguration"},
		{Name: "DescribeTemplate", Method: http.MethodGet, IAMAction: "quicksight:DescribeTemplate"},
		{Name: "DescribeTemplateAlias", Method: http.MethodGet, IAMAction: "quicksight:DescribeTemplateAlias"},
		{Name: "DescribeTemplateDefinition", Method: http.MethodGet, IAMAction: "quicksight:DescribeTemplateDefinition"},
		{Name: "DescribeTemplatePermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeTemplatePermissions"},
		{Name: "DescribeTheme", Method: http.MethodGet, IAMAction: "quicksight:DescribeTheme"},
		{Name: "DescribeThemeAlias", Method: http.MethodGet, IAMAction: "quicksight:DescribeThemeAlias"},
		{Name: "DescribeThemePermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeThemePermissions"},
		{Name: "DescribeTopic", Method: http.MethodGet, IAMAction: "quicksight:DescribeTopic"},
		{Name: "DescribeTopicPermissions", Method: http.MethodGet, IAMAction: "quicksight:DescribeTopicPermissions"},
		{Name: "DescribeTopicRefresh", Method: http.MethodGet, IAMAction: "quicksight:DescribeTopicRefresh"},
		{Name: "DescribeTopicRefreshSchedule", Method: http.MethodGet, IAMAction: "quicksight:DescribeTopicRefreshSchedule"},
		{Name: "DescribeUser", Method: http.MethodGet, IAMAction: "quicksight:DescribeUser"},
		{Name: "DescribeVPCConnection", Method: http.MethodGet, IAMAction: "quicksight:DescribeVPCConnection"},
		{Name: "GenerateEmbedUrlForAnonymousUser", Method: http.MethodPost, IAMAction: "quicksight:GenerateEmbedUrlForAnonymousUser"},
		{Name: "GenerateEmbedUrlForRegisteredUser", Method: http.MethodPost, IAMAction: "quicksight:GenerateEmbedUrlForRegisteredUser"},
		{Name: "GenerateEmbedUrlForRegisteredUserWithIdentity", Method: http.MethodPost, IAMAction: "quicksight:GenerateEmbedUrlForRegisteredUserWithIdentity"},
		{Name: "GetDashboardEmbedUrl", Method: http.MethodGet, IAMAction: "quicksight:GetDashboardEmbedUrl"},
		{Name: "GetFlowMetadata", Method: http.MethodGet, IAMAction: "quicksight:GetFlowMetadata"},
		{Name: "GetFlowPermissions", Method: http.MethodGet, IAMAction: "quicksight:GetFlowPermissions"},
		{Name: "GetIdentityContext", Method: http.MethodPost, IAMAction: "quicksight:GetIdentityContext"},
		{Name: "GetSessionEmbedUrl", Method: http.MethodGet, IAMAction: "quicksight:GetSessionEmbedUrl"},
		{Name: "ListActionConnectors", Method: http.MethodGet, IAMAction: "quicksight:ListActionConnectors"},
		{Name: "ListAnalyses", Method: http.MethodGet, IAMAction: "quicksight:ListAnalyses"},
		{Name: "ListAssetBundleExportJobs", Method: http.MethodGet, IAMAction: "quicksight:ListAssetBundleExportJobs"},
		{Name: "ListAssetBundleImportJobs", Method: http.MethodGet, IAMAction: "quicksight:ListAssetBundleImportJobs"},
		{Name: "ListBrands", Method: http.MethodGet, IAMAction: "quicksight:ListBrands"},
		{Name: "ListCustomPermissions", Method: http.MethodGet, IAMAction: "quicksight:ListCustomPermissions"},
		{Name: "ListDashboardVersions", Method: http.MethodGet, IAMAction: "quicksight:ListDashboardVersions"},
		{Name: "ListDashboards", Method: http.MethodGet, IAMAction: "quicksight:ListDashboards"},
		{Name: "ListDataSets", Method: http.MethodGet, IAMAction: "quicksight:ListDataSets"},
		{Name: "ListDataSources", Method: http.MethodGet, IAMAction: "quicksight:ListDataSources"},
		{Name: "ListFlows", Method: http.MethodGet, IAMAction: "quicksight:ListFlows"},
		{Name: "ListFolderMembers", Method: http.MethodGet, IAMAction: "quicksight:ListFolderMembers"},
		{Name: "ListFolders", Method: http.MethodGet, IAMAction: "quicksight:ListFolders"},
		{Name: "ListFoldersForResource", Method: http.MethodGet, IAMAction: "quicksight:ListFoldersForResource"},
		{Name: "ListGroupMemberships", Method: http.MethodGet, IAMAction: "quicksight:ListGroupMemberships"},
		{Name: "ListGroups", Method: http.MethodGet, IAMAction: "quicksight:ListGroups"},
		{Name: "ListIAMPolicyAssignments", Method: http.MethodGet, IAMAction: "quicksight:ListIAMPolicyAssignments"},
		{Name: "ListIAMPolicyAssignmentsForUser", Method: http.MethodGet, IAMAction: "quicksight:ListIAMPolicyAssignmentsForUser"},
		{Name: "ListIdentityPropagationConfigs", Method: http.MethodGet, IAMAction: "quicksight:ListIdentityPropagationConfigs"},
		{Name: "ListIngestions", Method: http.MethodGet, IAMAction: "quicksight:ListIngestions"},
		{Name: "ListNamespaces", Method: http.MethodGet, IAMAction: "quicksight:ListNamespaces"},
		{Name: "ListRefreshSchedules", Method: http.MethodGet, IAMAction: "quicksight:ListRefreshSchedules"},
		{Name: "ListRoleMemberships", Method: http.MethodGet, IAMAction: "quicksight:ListRoleMemberships"},
		{Name: "ListSelfUpgrades", Method: http.MethodGet, IAMAction: "quicksight:ListSelfUpgrades"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "quicksight:ListTagsForResource"},
		{Name: "ListTemplateAliases", Method: http.MethodGet, IAMAction: "quicksight:ListTemplateAliases"},
		{Name: "ListTemplateVersions", Method: http.MethodGet, IAMAction: "quicksight:ListTemplateVersions"},
		{Name: "ListTemplates", Method: http.MethodGet, IAMAction: "quicksight:ListTemplates"},
		{Name: "ListThemeAliases", Method: http.MethodGet, IAMAction: "quicksight:ListThemeAliases"},
		{Name: "ListThemeVersions", Method: http.MethodGet, IAMAction: "quicksight:ListThemeVersions"},
		{Name: "ListThemes", Method: http.MethodGet, IAMAction: "quicksight:ListThemes"},
		{Name: "ListTopicRefreshSchedules", Method: http.MethodGet, IAMAction: "quicksight:ListTopicRefreshSchedules"},
		{Name: "ListTopicReviewedAnswers", Method: http.MethodGet, IAMAction: "quicksight:ListTopicReviewedAnswers"},
		{Name: "ListTopics", Method: http.MethodGet, IAMAction: "quicksight:ListTopics"},
		{Name: "ListUserGroups", Method: http.MethodGet, IAMAction: "quicksight:ListUserGroups"},
		{Name: "ListUsers", Method: http.MethodGet, IAMAction: "quicksight:ListUsers"},
		{Name: "ListVPCConnections", Method: http.MethodGet, IAMAction: "quicksight:ListVPCConnections"},
		{Name: "PredictQAResults", Method: http.MethodPost, IAMAction: "quicksight:PredictQAResults"},
		{Name: "PutDataSetRefreshProperties", Method: http.MethodPut, IAMAction: "quicksight:PutDataSetRefreshProperties"},
		{Name: "RegisterUser", Method: http.MethodPost, IAMAction: "quicksight:RegisterUser"},
		{Name: "RestoreAnalysis", Method: http.MethodPost, IAMAction: "quicksight:RestoreAnalysis"},
		{Name: "SearchActionConnectors", Method: http.MethodPost, IAMAction: "quicksight:SearchActionConnectors"},
		{Name: "SearchAnalyses", Method: http.MethodPost, IAMAction: "quicksight:SearchAnalyses"},
		{Name: "SearchDashboards", Method: http.MethodPost, IAMAction: "quicksight:SearchDashboards"},
		{Name: "SearchDataSets", Method: http.MethodPost, IAMAction: "quicksight:SearchDataSets"},
		{Name: "SearchDataSources", Method: http.MethodPost, IAMAction: "quicksight:SearchDataSources"},
		{Name: "SearchFlows", Method: http.MethodPost, IAMAction: "quicksight:SearchFlows"},
		{Name: "SearchFolders", Method: http.MethodPost, IAMAction: "quicksight:SearchFolders"},
		{Name: "SearchGroups", Method: http.MethodPost, IAMAction: "quicksight:SearchGroups"},
		{Name: "SearchTopics", Method: http.MethodPost, IAMAction: "quicksight:SearchTopics"},
		{Name: "StartAssetBundleExportJob", Method: http.MethodPost, IAMAction: "quicksight:StartAssetBundleExportJob"},
		{Name: "StartAssetBundleImportJob", Method: http.MethodPost, IAMAction: "quicksight:StartAssetBundleImportJob"},
		{Name: "StartAutomationJob", Method: http.MethodPost, IAMAction: "quicksight:StartAutomationJob"},
		{Name: "StartDashboardSnapshotJob", Method: http.MethodPost, IAMAction: "quicksight:StartDashboardSnapshotJob"},
		{Name: "StartDashboardSnapshotJobSchedule", Method: http.MethodPost, IAMAction: "quicksight:StartDashboardSnapshotJobSchedule"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "quicksight:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "quicksight:UntagResource"},
		{Name: "UpdateAccountCustomPermission", Method: http.MethodPut, IAMAction: "quicksight:UpdateAccountCustomPermission"},
		{Name: "UpdateAccountCustomization", Method: http.MethodPut, IAMAction: "quicksight:UpdateAccountCustomization"},
		{Name: "UpdateAccountSettings", Method: http.MethodPut, IAMAction: "quicksight:UpdateAccountSettings"},
		{Name: "UpdateActionConnector", Method: http.MethodPut, IAMAction: "quicksight:UpdateActionConnector"},
		{Name: "UpdateActionConnectorPermissions", Method: http.MethodPost, IAMAction: "quicksight:UpdateActionConnectorPermissions"},
		{Name: "UpdateAnalysis", Method: http.MethodPut, IAMAction: "quicksight:UpdateAnalysis"},
		{Name: "UpdateAnalysisPermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateAnalysisPermissions"},
		{Name: "UpdateApplicationWithTokenExchangeGrant", Method: http.MethodPut, IAMAction: "quicksight:UpdateApplicationWithTokenExchangeGrant"},
		{Name: "UpdateBrand", Method: http.MethodPut, IAMAction: "quicksight:UpdateBrand"},
		{Name: "UpdateBrandAssignment", Method: http.MethodPut, IAMAction: "quicksight:UpdateBrandAssignment"},
		{Name: "UpdateBrandPublishedVersion", Method: http.MethodPut, IAMAction: "quicksight:UpdateBrandPublishedVersion"},
		{Name: "UpdateCustomPermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateCustomPermissions"},
		{Name: "UpdateDashboard", Method: http.MethodPut, IAMAction: "quicksight:UpdateDashboard"},
		{Name: "UpdateDashboardLinks", Method: http.MethodPut, IAMAction: "quicksight:UpdateDashboardLinks"},
		{Name: "UpdateDashboardPermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateDashboardPermissions"},
		{Name: "UpdateDashboardPublishedVersion", Method: http.MethodPut, IAMAction: "quicksight:UpdateDashboardPublishedVersion"},
		{Name: "UpdateDashboardsQAConfiguration", Method: http.MethodPut, IAMAction: "quicksight:UpdateDashboardsQAConfiguration"},
		{Name: "UpdateDataSet", Method: http.MethodPut, IAMAction: "quicksight:UpdateDataSet"},
		{Name: "UpdateDataSetPermissions", Method: http.MethodPost, IAMAction: "quicksight:UpdateDataSetPermissions"},
		{Name: "UpdateDataSource", Method: http.MethodPut, IAMAction: "quicksight:UpdateDataSource"},
		{Name: "UpdateDataSourcePermissions", Method: http.MethodPost, IAMAction: "quicksight:UpdateDataSourcePermissions"},
		{Name: "UpdateDefaultQBusinessApplication", Method: http.MethodPut, IAMAction: "quicksight:UpdateDefaultQBusinessApplication"},
		{Name: "UpdateFlowPermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateFlowPermissions"},
		{Name: "UpdateFolder", Method: http.MethodPut, IAMAction: "quicksight:UpdateFolder"},
		{Name: "UpdateFolderPermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateFolderPermissions"},
		{Name: "UpdateGroup", Method: http.MethodPut, IAMAction: "quicksight:UpdateGroup"},
		{Name: "UpdateIAMPolicyAssignment", Method: http.MethodPut, IAMAction: "quicksight:UpdateIAMPolicyAssignment"},
		{Name: "UpdateIdentityPropagationConfig", Method: http.MethodPost, IAMAction: "quicksight:UpdateIdentityPropagationConfig"},
		{Name: "UpdateIpRestriction", Method: http.MethodPost, IAMAction: "quicksight:UpdateIpRestriction"},
		{Name: "UpdateKeyRegistration", Method: http.MethodPost, IAMAction: "quicksight:UpdateKeyRegistration"},
		{Name: "UpdatePublicSharingSettings", Method: http.MethodPut, IAMAction: "quicksight:UpdatePublicSharingSettings"},
		{Name: "UpdateQPersonalizationConfiguration", Method: http.MethodPut, IAMAction: "quicksight:UpdateQPersonalizationConfiguration"},
		{Name: "UpdateQuickSightQSearchConfiguration", Method: http.MethodPut, IAMAction: "quicksight:UpdateQuickSightQSearchConfiguration"},
		{Name: "UpdateRefreshSchedule", Method: http.MethodPut, IAMAction: "quicksight:UpdateRefreshSchedule"},
		{Name: "UpdateRoleCustomPermission", Method: http.MethodPut, IAMAction: "quicksight:UpdateRoleCustomPermission"},
		{Name: "UpdateSPICECapacityConfiguration", Method: http.MethodPost, IAMAction: "quicksight:UpdateSPICECapacityConfiguration"},
		{Name: "UpdateSelfUpgrade", Method: http.MethodPost, IAMAction: "quicksight:UpdateSelfUpgrade"},
		{Name: "UpdateSelfUpgradeConfiguration", Method: http.MethodPut, IAMAction: "quicksight:UpdateSelfUpgradeConfiguration"},
		{Name: "UpdateTemplate", Method: http.MethodPut, IAMAction: "quicksight:UpdateTemplate"},
		{Name: "UpdateTemplateAlias", Method: http.MethodPut, IAMAction: "quicksight:UpdateTemplateAlias"},
		{Name: "UpdateTemplatePermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateTemplatePermissions"},
		{Name: "UpdateTheme", Method: http.MethodPut, IAMAction: "quicksight:UpdateTheme"},
		{Name: "UpdateThemeAlias", Method: http.MethodPut, IAMAction: "quicksight:UpdateThemeAlias"},
		{Name: "UpdateThemePermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateThemePermissions"},
		{Name: "UpdateTopic", Method: http.MethodPut, IAMAction: "quicksight:UpdateTopic"},
		{Name: "UpdateTopicPermissions", Method: http.MethodPut, IAMAction: "quicksight:UpdateTopicPermissions"},
		{Name: "UpdateTopicRefreshSchedule", Method: http.MethodPut, IAMAction: "quicksight:UpdateTopicRefreshSchedule"},
		{Name: "UpdateUser", Method: http.MethodPut, IAMAction: "quicksight:UpdateUser"},
		{Name: "UpdateUserCustomPermission", Method: http.MethodPut, IAMAction: "quicksight:UpdateUserCustomPermission"},
		{Name: "UpdateVPCConnection", Method: http.MethodPut, IAMAction: "quicksight:UpdateVPCConnection"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "BatchCreateTopicReviewedAnswer":
		return handleBatchCreateTopicReviewedAnswer(ctx, s.store)
	case "BatchDeleteTopicReviewedAnswer":
		return handleBatchDeleteTopicReviewedAnswer(ctx, s.store)
	case "CancelIngestion":
		return handleCancelIngestion(ctx, s.store)
	case "CreateAccountCustomization":
		return handleCreateAccountCustomization(ctx, s.store)
	case "CreateAccountSubscription":
		return handleCreateAccountSubscription(ctx, s.store)
	case "CreateActionConnector":
		return handleCreateActionConnector(ctx, s.store)
	case "CreateAnalysis":
		return handleCreateAnalysis(ctx, s.store)
	case "CreateBrand":
		return handleCreateBrand(ctx, s.store)
	case "CreateCustomPermissions":
		return handleCreateCustomPermissions(ctx, s.store)
	case "CreateDashboard":
		return handleCreateDashboard(ctx, s.store)
	case "CreateDataSet":
		return handleCreateDataSet(ctx, s.store)
	case "CreateDataSource":
		return handleCreateDataSource(ctx, s.store)
	case "CreateFolder":
		return handleCreateFolder(ctx, s.store)
	case "CreateFolderMembership":
		return handleCreateFolderMembership(ctx, s.store)
	case "CreateGroup":
		return handleCreateGroup(ctx, s.store)
	case "CreateGroupMembership":
		return handleCreateGroupMembership(ctx, s.store)
	case "CreateIAMPolicyAssignment":
		return handleCreateIAMPolicyAssignment(ctx, s.store)
	case "CreateIngestion":
		return handleCreateIngestion(ctx, s.store)
	case "CreateNamespace":
		return handleCreateNamespace(ctx, s.store)
	case "CreateRefreshSchedule":
		return handleCreateRefreshSchedule(ctx, s.store)
	case "CreateRoleMembership":
		return handleCreateRoleMembership(ctx, s.store)
	case "CreateTemplate":
		return handleCreateTemplate(ctx, s.store)
	case "CreateTemplateAlias":
		return handleCreateTemplateAlias(ctx, s.store)
	case "CreateTheme":
		return handleCreateTheme(ctx, s.store)
	case "CreateThemeAlias":
		return handleCreateThemeAlias(ctx, s.store)
	case "CreateTopic":
		return handleCreateTopic(ctx, s.store)
	case "CreateTopicRefreshSchedule":
		return handleCreateTopicRefreshSchedule(ctx, s.store)
	case "CreateVPCConnection":
		return handleCreateVPCConnection(ctx, s.store)
	case "DeleteAccountCustomPermission":
		return handleDeleteAccountCustomPermission(ctx, s.store)
	case "DeleteAccountCustomization":
		return handleDeleteAccountCustomization(ctx, s.store)
	case "DeleteAccountSubscription":
		return handleDeleteAccountSubscription(ctx, s.store)
	case "DeleteActionConnector":
		return handleDeleteActionConnector(ctx, s.store)
	case "DeleteAnalysis":
		return handleDeleteAnalysis(ctx, s.store)
	case "DeleteBrand":
		return handleDeleteBrand(ctx, s.store)
	case "DeleteBrandAssignment":
		return handleDeleteBrandAssignment(ctx, s.store)
	case "DeleteCustomPermissions":
		return handleDeleteCustomPermissions(ctx, s.store)
	case "DeleteDashboard":
		return handleDeleteDashboard(ctx, s.store)
	case "DeleteDataSet":
		return handleDeleteDataSet(ctx, s.store)
	case "DeleteDataSetRefreshProperties":
		return handleDeleteDataSetRefreshProperties(ctx, s.store)
	case "DeleteDataSource":
		return handleDeleteDataSource(ctx, s.store)
	case "DeleteDefaultQBusinessApplication":
		return handleDeleteDefaultQBusinessApplication(ctx, s.store)
	case "DeleteFolder":
		return handleDeleteFolder(ctx, s.store)
	case "DeleteFolderMembership":
		return handleDeleteFolderMembership(ctx, s.store)
	case "DeleteGroup":
		return handleDeleteGroup(ctx, s.store)
	case "DeleteGroupMembership":
		return handleDeleteGroupMembership(ctx, s.store)
	case "DeleteIAMPolicyAssignment":
		return handleDeleteIAMPolicyAssignment(ctx, s.store)
	case "DeleteIdentityPropagationConfig":
		return handleDeleteIdentityPropagationConfig(ctx, s.store)
	case "DeleteNamespace":
		return handleDeleteNamespace(ctx, s.store)
	case "DeleteRefreshSchedule":
		return handleDeleteRefreshSchedule(ctx, s.store)
	case "DeleteRoleCustomPermission":
		return handleDeleteRoleCustomPermission(ctx, s.store)
	case "DeleteRoleMembership":
		return handleDeleteRoleMembership(ctx, s.store)
	case "DeleteTemplate":
		return handleDeleteTemplate(ctx, s.store)
	case "DeleteTemplateAlias":
		return handleDeleteTemplateAlias(ctx, s.store)
	case "DeleteTheme":
		return handleDeleteTheme(ctx, s.store)
	case "DeleteThemeAlias":
		return handleDeleteThemeAlias(ctx, s.store)
	case "DeleteTopic":
		return handleDeleteTopic(ctx, s.store)
	case "DeleteTopicRefreshSchedule":
		return handleDeleteTopicRefreshSchedule(ctx, s.store)
	case "DeleteUser":
		return handleDeleteUser(ctx, s.store)
	case "DeleteUserByPrincipalId":
		return handleDeleteUserByPrincipalId(ctx, s.store)
	case "DeleteUserCustomPermission":
		return handleDeleteUserCustomPermission(ctx, s.store)
	case "DeleteVPCConnection":
		return handleDeleteVPCConnection(ctx, s.store)
	case "DescribeAccountCustomPermission":
		return handleDescribeAccountCustomPermission(ctx, s.store)
	case "DescribeAccountCustomization":
		return handleDescribeAccountCustomization(ctx, s.store)
	case "DescribeAccountSettings":
		return handleDescribeAccountSettings(ctx, s.store)
	case "DescribeAccountSubscription":
		return handleDescribeAccountSubscription(ctx, s.store)
	case "DescribeActionConnector":
		return handleDescribeActionConnector(ctx, s.store)
	case "DescribeActionConnectorPermissions":
		return handleDescribeActionConnectorPermissions(ctx, s.store)
	case "DescribeAnalysis":
		return handleDescribeAnalysis(ctx, s.store)
	case "DescribeAnalysisDefinition":
		return handleDescribeAnalysisDefinition(ctx, s.store)
	case "DescribeAnalysisPermissions":
		return handleDescribeAnalysisPermissions(ctx, s.store)
	case "DescribeAssetBundleExportJob":
		return handleDescribeAssetBundleExportJob(ctx, s.store)
	case "DescribeAssetBundleImportJob":
		return handleDescribeAssetBundleImportJob(ctx, s.store)
	case "DescribeAutomationJob":
		return handleDescribeAutomationJob(ctx, s.store)
	case "DescribeBrand":
		return handleDescribeBrand(ctx, s.store)
	case "DescribeBrandAssignment":
		return handleDescribeBrandAssignment(ctx, s.store)
	case "DescribeBrandPublishedVersion":
		return handleDescribeBrandPublishedVersion(ctx, s.store)
	case "DescribeCustomPermissions":
		return handleDescribeCustomPermissions(ctx, s.store)
	case "DescribeDashboard":
		return handleDescribeDashboard(ctx, s.store)
	case "DescribeDashboardDefinition":
		return handleDescribeDashboardDefinition(ctx, s.store)
	case "DescribeDashboardPermissions":
		return handleDescribeDashboardPermissions(ctx, s.store)
	case "DescribeDashboardSnapshotJob":
		return handleDescribeDashboardSnapshotJob(ctx, s.store)
	case "DescribeDashboardSnapshotJobResult":
		return handleDescribeDashboardSnapshotJobResult(ctx, s.store)
	case "DescribeDashboardsQAConfiguration":
		return handleDescribeDashboardsQAConfiguration(ctx, s.store)
	case "DescribeDataSet":
		return handleDescribeDataSet(ctx, s.store)
	case "DescribeDataSetPermissions":
		return handleDescribeDataSetPermissions(ctx, s.store)
	case "DescribeDataSetRefreshProperties":
		return handleDescribeDataSetRefreshProperties(ctx, s.store)
	case "DescribeDataSource":
		return handleDescribeDataSource(ctx, s.store)
	case "DescribeDataSourcePermissions":
		return handleDescribeDataSourcePermissions(ctx, s.store)
	case "DescribeDefaultQBusinessApplication":
		return handleDescribeDefaultQBusinessApplication(ctx, s.store)
	case "DescribeFolder":
		return handleDescribeFolder(ctx, s.store)
	case "DescribeFolderPermissions":
		return handleDescribeFolderPermissions(ctx, s.store)
	case "DescribeFolderResolvedPermissions":
		return handleDescribeFolderResolvedPermissions(ctx, s.store)
	case "DescribeGroup":
		return handleDescribeGroup(ctx, s.store)
	case "DescribeGroupMembership":
		return handleDescribeGroupMembership(ctx, s.store)
	case "DescribeIAMPolicyAssignment":
		return handleDescribeIAMPolicyAssignment(ctx, s.store)
	case "DescribeIngestion":
		return handleDescribeIngestion(ctx, s.store)
	case "DescribeIpRestriction":
		return handleDescribeIpRestriction(ctx, s.store)
	case "DescribeKeyRegistration":
		return handleDescribeKeyRegistration(ctx, s.store)
	case "DescribeNamespace":
		return handleDescribeNamespace(ctx, s.store)
	case "DescribeQPersonalizationConfiguration":
		return handleDescribeQPersonalizationConfiguration(ctx, s.store)
	case "DescribeQuickSightQSearchConfiguration":
		return handleDescribeQuickSightQSearchConfiguration(ctx, s.store)
	case "DescribeRefreshSchedule":
		return handleDescribeRefreshSchedule(ctx, s.store)
	case "DescribeRoleCustomPermission":
		return handleDescribeRoleCustomPermission(ctx, s.store)
	case "DescribeSelfUpgradeConfiguration":
		return handleDescribeSelfUpgradeConfiguration(ctx, s.store)
	case "DescribeTemplate":
		return handleDescribeTemplate(ctx, s.store)
	case "DescribeTemplateAlias":
		return handleDescribeTemplateAlias(ctx, s.store)
	case "DescribeTemplateDefinition":
		return handleDescribeTemplateDefinition(ctx, s.store)
	case "DescribeTemplatePermissions":
		return handleDescribeTemplatePermissions(ctx, s.store)
	case "DescribeTheme":
		return handleDescribeTheme(ctx, s.store)
	case "DescribeThemeAlias":
		return handleDescribeThemeAlias(ctx, s.store)
	case "DescribeThemePermissions":
		return handleDescribeThemePermissions(ctx, s.store)
	case "DescribeTopic":
		return handleDescribeTopic(ctx, s.store)
	case "DescribeTopicPermissions":
		return handleDescribeTopicPermissions(ctx, s.store)
	case "DescribeTopicRefresh":
		return handleDescribeTopicRefresh(ctx, s.store)
	case "DescribeTopicRefreshSchedule":
		return handleDescribeTopicRefreshSchedule(ctx, s.store)
	case "DescribeUser":
		return handleDescribeUser(ctx, s.store)
	case "DescribeVPCConnection":
		return handleDescribeVPCConnection(ctx, s.store)
	case "GenerateEmbedUrlForAnonymousUser":
		return handleGenerateEmbedUrlForAnonymousUser(ctx, s.store)
	case "GenerateEmbedUrlForRegisteredUser":
		return handleGenerateEmbedUrlForRegisteredUser(ctx, s.store)
	case "GenerateEmbedUrlForRegisteredUserWithIdentity":
		return handleGenerateEmbedUrlForRegisteredUserWithIdentity(ctx, s.store)
	case "GetDashboardEmbedUrl":
		return handleGetDashboardEmbedUrl(ctx, s.store)
	case "GetFlowMetadata":
		return handleGetFlowMetadata(ctx, s.store)
	case "GetFlowPermissions":
		return handleGetFlowPermissions(ctx, s.store)
	case "GetIdentityContext":
		return handleGetIdentityContext(ctx, s.store)
	case "GetSessionEmbedUrl":
		return handleGetSessionEmbedUrl(ctx, s.store)
	case "ListActionConnectors":
		return handleListActionConnectors(ctx, s.store)
	case "ListAnalyses":
		return handleListAnalyses(ctx, s.store)
	case "ListAssetBundleExportJobs":
		return handleListAssetBundleExportJobs(ctx, s.store)
	case "ListAssetBundleImportJobs":
		return handleListAssetBundleImportJobs(ctx, s.store)
	case "ListBrands":
		return handleListBrands(ctx, s.store)
	case "ListCustomPermissions":
		return handleListCustomPermissions(ctx, s.store)
	case "ListDashboardVersions":
		return handleListDashboardVersions(ctx, s.store)
	case "ListDashboards":
		return handleListDashboards(ctx, s.store)
	case "ListDataSets":
		return handleListDataSets(ctx, s.store)
	case "ListDataSources":
		return handleListDataSources(ctx, s.store)
	case "ListFlows":
		return handleListFlows(ctx, s.store)
	case "ListFolderMembers":
		return handleListFolderMembers(ctx, s.store)
	case "ListFolders":
		return handleListFolders(ctx, s.store)
	case "ListFoldersForResource":
		return handleListFoldersForResource(ctx, s.store)
	case "ListGroupMemberships":
		return handleListGroupMemberships(ctx, s.store)
	case "ListGroups":
		return handleListGroups(ctx, s.store)
	case "ListIAMPolicyAssignments":
		return handleListIAMPolicyAssignments(ctx, s.store)
	case "ListIAMPolicyAssignmentsForUser":
		return handleListIAMPolicyAssignmentsForUser(ctx, s.store)
	case "ListIdentityPropagationConfigs":
		return handleListIdentityPropagationConfigs(ctx, s.store)
	case "ListIngestions":
		return handleListIngestions(ctx, s.store)
	case "ListNamespaces":
		return handleListNamespaces(ctx, s.store)
	case "ListRefreshSchedules":
		return handleListRefreshSchedules(ctx, s.store)
	case "ListRoleMemberships":
		return handleListRoleMemberships(ctx, s.store)
	case "ListSelfUpgrades":
		return handleListSelfUpgrades(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ListTemplateAliases":
		return handleListTemplateAliases(ctx, s.store)
	case "ListTemplateVersions":
		return handleListTemplateVersions(ctx, s.store)
	case "ListTemplates":
		return handleListTemplates(ctx, s.store)
	case "ListThemeAliases":
		return handleListThemeAliases(ctx, s.store)
	case "ListThemeVersions":
		return handleListThemeVersions(ctx, s.store)
	case "ListThemes":
		return handleListThemes(ctx, s.store)
	case "ListTopicRefreshSchedules":
		return handleListTopicRefreshSchedules(ctx, s.store)
	case "ListTopicReviewedAnswers":
		return handleListTopicReviewedAnswers(ctx, s.store)
	case "ListTopics":
		return handleListTopics(ctx, s.store)
	case "ListUserGroups":
		return handleListUserGroups(ctx, s.store)
	case "ListUsers":
		return handleListUsers(ctx, s.store)
	case "ListVPCConnections":
		return handleListVPCConnections(ctx, s.store)
	case "PredictQAResults":
		return handlePredictQAResults(ctx, s.store)
	case "PutDataSetRefreshProperties":
		return handlePutDataSetRefreshProperties(ctx, s.store)
	case "RegisterUser":
		return handleRegisterUser(ctx, s.store)
	case "RestoreAnalysis":
		return handleRestoreAnalysis(ctx, s.store)
	case "SearchActionConnectors":
		return handleSearchActionConnectors(ctx, s.store)
	case "SearchAnalyses":
		return handleSearchAnalyses(ctx, s.store)
	case "SearchDashboards":
		return handleSearchDashboards(ctx, s.store)
	case "SearchDataSets":
		return handleSearchDataSets(ctx, s.store)
	case "SearchDataSources":
		return handleSearchDataSources(ctx, s.store)
	case "SearchFlows":
		return handleSearchFlows(ctx, s.store)
	case "SearchFolders":
		return handleSearchFolders(ctx, s.store)
	case "SearchGroups":
		return handleSearchGroups(ctx, s.store)
	case "SearchTopics":
		return handleSearchTopics(ctx, s.store)
	case "StartAssetBundleExportJob":
		return handleStartAssetBundleExportJob(ctx, s.store)
	case "StartAssetBundleImportJob":
		return handleStartAssetBundleImportJob(ctx, s.store)
	case "StartAutomationJob":
		return handleStartAutomationJob(ctx, s.store)
	case "StartDashboardSnapshotJob":
		return handleStartDashboardSnapshotJob(ctx, s.store)
	case "StartDashboardSnapshotJobSchedule":
		return handleStartDashboardSnapshotJobSchedule(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateAccountCustomPermission":
		return handleUpdateAccountCustomPermission(ctx, s.store)
	case "UpdateAccountCustomization":
		return handleUpdateAccountCustomization(ctx, s.store)
	case "UpdateAccountSettings":
		return handleUpdateAccountSettings(ctx, s.store)
	case "UpdateActionConnector":
		return handleUpdateActionConnector(ctx, s.store)
	case "UpdateActionConnectorPermissions":
		return handleUpdateActionConnectorPermissions(ctx, s.store)
	case "UpdateAnalysis":
		return handleUpdateAnalysis(ctx, s.store)
	case "UpdateAnalysisPermissions":
		return handleUpdateAnalysisPermissions(ctx, s.store)
	case "UpdateApplicationWithTokenExchangeGrant":
		return handleUpdateApplicationWithTokenExchangeGrant(ctx, s.store)
	case "UpdateBrand":
		return handleUpdateBrand(ctx, s.store)
	case "UpdateBrandAssignment":
		return handleUpdateBrandAssignment(ctx, s.store)
	case "UpdateBrandPublishedVersion":
		return handleUpdateBrandPublishedVersion(ctx, s.store)
	case "UpdateCustomPermissions":
		return handleUpdateCustomPermissions(ctx, s.store)
	case "UpdateDashboard":
		return handleUpdateDashboard(ctx, s.store)
	case "UpdateDashboardLinks":
		return handleUpdateDashboardLinks(ctx, s.store)
	case "UpdateDashboardPermissions":
		return handleUpdateDashboardPermissions(ctx, s.store)
	case "UpdateDashboardPublishedVersion":
		return handleUpdateDashboardPublishedVersion(ctx, s.store)
	case "UpdateDashboardsQAConfiguration":
		return handleUpdateDashboardsQAConfiguration(ctx, s.store)
	case "UpdateDataSet":
		return handleUpdateDataSet(ctx, s.store)
	case "UpdateDataSetPermissions":
		return handleUpdateDataSetPermissions(ctx, s.store)
	case "UpdateDataSource":
		return handleUpdateDataSource(ctx, s.store)
	case "UpdateDataSourcePermissions":
		return handleUpdateDataSourcePermissions(ctx, s.store)
	case "UpdateDefaultQBusinessApplication":
		return handleUpdateDefaultQBusinessApplication(ctx, s.store)
	case "UpdateFlowPermissions":
		return handleUpdateFlowPermissions(ctx, s.store)
	case "UpdateFolder":
		return handleUpdateFolder(ctx, s.store)
	case "UpdateFolderPermissions":
		return handleUpdateFolderPermissions(ctx, s.store)
	case "UpdateGroup":
		return handleUpdateGroup(ctx, s.store)
	case "UpdateIAMPolicyAssignment":
		return handleUpdateIAMPolicyAssignment(ctx, s.store)
	case "UpdateIdentityPropagationConfig":
		return handleUpdateIdentityPropagationConfig(ctx, s.store)
	case "UpdateIpRestriction":
		return handleUpdateIpRestriction(ctx, s.store)
	case "UpdateKeyRegistration":
		return handleUpdateKeyRegistration(ctx, s.store)
	case "UpdatePublicSharingSettings":
		return handleUpdatePublicSharingSettings(ctx, s.store)
	case "UpdateQPersonalizationConfiguration":
		return handleUpdateQPersonalizationConfiguration(ctx, s.store)
	case "UpdateQuickSightQSearchConfiguration":
		return handleUpdateQuickSightQSearchConfiguration(ctx, s.store)
	case "UpdateRefreshSchedule":
		return handleUpdateRefreshSchedule(ctx, s.store)
	case "UpdateRoleCustomPermission":
		return handleUpdateRoleCustomPermission(ctx, s.store)
	case "UpdateSPICECapacityConfiguration":
		return handleUpdateSPICECapacityConfiguration(ctx, s.store)
	case "UpdateSelfUpgrade":
		return handleUpdateSelfUpgrade(ctx, s.store)
	case "UpdateSelfUpgradeConfiguration":
		return handleUpdateSelfUpgradeConfiguration(ctx, s.store)
	case "UpdateTemplate":
		return handleUpdateTemplate(ctx, s.store)
	case "UpdateTemplateAlias":
		return handleUpdateTemplateAlias(ctx, s.store)
	case "UpdateTemplatePermissions":
		return handleUpdateTemplatePermissions(ctx, s.store)
	case "UpdateTheme":
		return handleUpdateTheme(ctx, s.store)
	case "UpdateThemeAlias":
		return handleUpdateThemeAlias(ctx, s.store)
	case "UpdateThemePermissions":
		return handleUpdateThemePermissions(ctx, s.store)
	case "UpdateTopic":
		return handleUpdateTopic(ctx, s.store)
	case "UpdateTopicPermissions":
		return handleUpdateTopicPermissions(ctx, s.store)
	case "UpdateTopicRefreshSchedule":
		return handleUpdateTopicRefreshSchedule(ctx, s.store)
	case "UpdateUser":
		return handleUpdateUser(ctx, s.store)
	case "UpdateUserCustomPermission":
		return handleUpdateUserCustomPermission(ctx, s.store)
	case "UpdateVPCConnection":
		return handleUpdateVPCConnection(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
