package inspector2

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type Account struct {
	AccountId string `json:"accountId,omitempty"`
	ResourceStatus ResourceStatus `json:"resourceStatus,omitempty"`
	Status string `json:"status,omitempty"`
}

type AccountAggregation struct {
	FindingType *string `json:"findingType,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type AccountAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	ExploitAvailableCount int64 `json:"exploitAvailableCount,omitempty"`
	FixAvailableCount int64 `json:"fixAvailableCount,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type AccountState struct {
	AccountId string `json:"accountId,omitempty"`
	ResourceState ResourceState `json:"resourceState,omitempty"`
	State State `json:"state,omitempty"`
}

type AggregationRequest struct {
	AccountAggregation *AccountAggregation `json:"accountAggregation,omitempty"`
	AmiAggregation *AmiAggregation `json:"amiAggregation,omitempty"`
	AwsEcrContainerAggregation *AwsEcrContainerAggregation `json:"awsEcrContainerAggregation,omitempty"`
	CodeRepositoryAggregation *CodeRepositoryAggregation `json:"codeRepositoryAggregation,omitempty"`
	Ec2InstanceAggregation *Ec2InstanceAggregation `json:"ec2InstanceAggregation,omitempty"`
	FindingTypeAggregation *FindingTypeAggregation `json:"findingTypeAggregation,omitempty"`
	ImageLayerAggregation *ImageLayerAggregation `json:"imageLayerAggregation,omitempty"`
	LambdaFunctionAggregation *LambdaFunctionAggregation `json:"lambdaFunctionAggregation,omitempty"`
	LambdaLayerAggregation *LambdaLayerAggregation `json:"lambdaLayerAggregation,omitempty"`
	PackageAggregation *PackageAggregation `json:"packageAggregation,omitempty"`
	RepositoryAggregation *RepositoryAggregation `json:"repositoryAggregation,omitempty"`
	TitleAggregation *TitleAggregation `json:"titleAggregation,omitempty"`
}

type AggregationResponse struct {
	AccountAggregation *AccountAggregationResponse `json:"accountAggregation,omitempty"`
	AmiAggregation *AmiAggregationResponse `json:"amiAggregation,omitempty"`
	AwsEcrContainerAggregation *AwsEcrContainerAggregationResponse `json:"awsEcrContainerAggregation,omitempty"`
	CodeRepositoryAggregation *CodeRepositoryAggregationResponse `json:"codeRepositoryAggregation,omitempty"`
	Ec2InstanceAggregation *Ec2InstanceAggregationResponse `json:"ec2InstanceAggregation,omitempty"`
	FindingTypeAggregation *FindingTypeAggregationResponse `json:"findingTypeAggregation,omitempty"`
	ImageLayerAggregation *ImageLayerAggregationResponse `json:"imageLayerAggregation,omitempty"`
	LambdaFunctionAggregation *LambdaFunctionAggregationResponse `json:"lambdaFunctionAggregation,omitempty"`
	LambdaLayerAggregation *LambdaLayerAggregationResponse `json:"lambdaLayerAggregation,omitempty"`
	PackageAggregation *PackageAggregationResponse `json:"packageAggregation,omitempty"`
	RepositoryAggregation *RepositoryAggregationResponse `json:"repositoryAggregation,omitempty"`
	TitleAggregation *TitleAggregationResponse `json:"titleAggregation,omitempty"`
}

type AmiAggregation struct {
	Amis []StringFilter `json:"amis,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type AmiAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	AffectedInstances int64 `json:"affectedInstances,omitempty"`
	Ami string `json:"ami,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type AssociateConfigurationRequest struct {
	Resource CodeSecurityResource `json:"resource,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type AssociateMemberRequest struct {
	AccountId string `json:"accountId,omitempty"`
}

type AssociateMemberResponse struct {
	AccountId string `json:"accountId,omitempty"`
}

type AtigData struct {
	FirstSeen *time.Time `json:"firstSeen,omitempty"`
	LastSeen *time.Time `json:"lastSeen,omitempty"`
	Targets []string `json:"targets,omitempty"`
	Ttps []string `json:"ttps,omitempty"`
}

type AutoEnable struct {
	CodeRepository bool `json:"codeRepository,omitempty"`
	Ec2 bool `json:"ec2,omitempty"`
	Ecr bool `json:"ecr,omitempty"`
	Lambda bool `json:"lambda,omitempty"`
	LambdaCode bool `json:"lambdaCode,omitempty"`
}

type AwsEc2InstanceDetails struct {
	IamInstanceProfileArn *string `json:"iamInstanceProfileArn,omitempty"`
	ImageId *string `json:"imageId,omitempty"`
	IpV4Addresses []string `json:"ipV4Addresses,omitempty"`
	IpV6Addresses []string `json:"ipV6Addresses,omitempty"`
	KeyName *string `json:"keyName,omitempty"`
	LaunchedAt *time.Time `json:"launchedAt,omitempty"`
	Platform *string `json:"platform,omitempty"`
	SubnetId *string `json:"subnetId,omitempty"`
	Type *string `json:"type,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
}

type AwsEcrContainerAggregation struct {
	Architectures []StringFilter `json:"architectures,omitempty"`
	ImageShas []StringFilter `json:"imageShas,omitempty"`
	ImageTags []StringFilter `json:"imageTags,omitempty"`
	InUseCount []NumberFilter `json:"inUseCount,omitempty"`
	LastInUseAt []DateFilter `json:"lastInUseAt,omitempty"`
	Repositories []StringFilter `json:"repositories,omitempty"`
	ResourceIds []StringFilter `json:"resourceIds,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type AwsEcrContainerAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	Architecture *string `json:"architecture,omitempty"`
	ImageSha *string `json:"imageSha,omitempty"`
	ImageTags []string `json:"imageTags,omitempty"`
	InUseCount int64 `json:"inUseCount,omitempty"`
	LastInUseAt *time.Time `json:"lastInUseAt,omitempty"`
	Repository *string `json:"repository,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type AwsEcrContainerImageDetails struct {
	Architecture *string `json:"architecture,omitempty"`
	Author *string `json:"author,omitempty"`
	ImageHash string `json:"imageHash,omitempty"`
	ImageTags []string `json:"imageTags,omitempty"`
	InUseCount int64 `json:"inUseCount,omitempty"`
	LastInUseAt *time.Time `json:"lastInUseAt,omitempty"`
	Platform *string `json:"platform,omitempty"`
	PushedAt *time.Time `json:"pushedAt,omitempty"`
	Registry string `json:"registry,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
}

type AwsEcsMetadataDetails struct {
	DetailsGroup string `json:"detailsGroup,omitempty"`
	TaskDefinitionArn string `json:"taskDefinitionArn,omitempty"`
}

type AwsEksMetadataDetails struct {
	Namespace *string `json:"namespace,omitempty"`
	WorkloadInfoList []AwsEksWorkloadInfo `json:"workloadInfoList,omitempty"`
}

type AwsEksWorkloadInfo struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type AwsLambdaFunctionDetails struct {
	Architectures []string `json:"architectures,omitempty"`
	CodeSha256 string `json:"codeSha256,omitempty"`
	ExecutionRoleArn string `json:"executionRoleArn,omitempty"`
	FunctionName string `json:"functionName,omitempty"`
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
	Layers []string `json:"layers,omitempty"`
	PackageType *string `json:"packageType,omitempty"`
	Runtime string `json:"runtime,omitempty"`
	Version string `json:"version,omitempty"`
	VpcConfig *LambdaVpcConfig `json:"vpcConfig,omitempty"`
}

type BatchAssociateCodeSecurityScanConfigurationRequest struct {
	AssociateConfigurationRequests []AssociateConfigurationRequest `json:"associateConfigurationRequests,omitempty"`
}

type BatchAssociateCodeSecurityScanConfigurationResponse struct {
	FailedAssociations []FailedAssociationResult `json:"failedAssociations,omitempty"`
	SuccessfulAssociations []SuccessfulAssociationResult `json:"successfulAssociations,omitempty"`
}

type BatchDisassociateCodeSecurityScanConfigurationRequest struct {
	DisassociateConfigurationRequests []DisassociateConfigurationRequest `json:"disassociateConfigurationRequests,omitempty"`
}

type BatchDisassociateCodeSecurityScanConfigurationResponse struct {
	FailedAssociations []FailedAssociationResult `json:"failedAssociations,omitempty"`
	SuccessfulAssociations []SuccessfulAssociationResult `json:"successfulAssociations,omitempty"`
}

type BatchGetAccountStatusRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
}

type BatchGetAccountStatusResponse struct {
	Accounts []AccountState `json:"accounts,omitempty"`
	FailedAccounts []FailedAccount `json:"failedAccounts,omitempty"`
}

type BatchGetCodeSnippetRequest struct {
	FindingArns []string `json:"findingArns,omitempty"`
}

type BatchGetCodeSnippetResponse struct {
	CodeSnippetResults []CodeSnippetResult `json:"codeSnippetResults,omitempty"`
	Errors []CodeSnippetError `json:"errors,omitempty"`
}

type BatchGetFindingDetailsRequest struct {
	FindingArns []string `json:"findingArns,omitempty"`
}

type BatchGetFindingDetailsResponse struct {
	Errors []FindingDetailsError `json:"errors,omitempty"`
	FindingDetails []FindingDetail `json:"findingDetails,omitempty"`
}

type BatchGetFreeTrialInfoRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
}

type BatchGetFreeTrialInfoResponse struct {
	Accounts []FreeTrialAccountInfo `json:"accounts,omitempty"`
	FailedAccounts []FreeTrialInfoError `json:"failedAccounts,omitempty"`
}

type BatchGetMemberEc2DeepInspectionStatusRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
}

type BatchGetMemberEc2DeepInspectionStatusResponse struct {
	AccountIds []MemberAccountEc2DeepInspectionStatusState `json:"accountIds,omitempty"`
	FailedAccountIds []FailedMemberAccountEc2DeepInspectionStatusState `json:"failedAccountIds,omitempty"`
}

type BatchUpdateMemberEc2DeepInspectionStatusRequest struct {
	AccountIds []MemberAccountEc2DeepInspectionStatus `json:"accountIds,omitempty"`
}

type BatchUpdateMemberEc2DeepInspectionStatusResponse struct {
	AccountIds []MemberAccountEc2DeepInspectionStatusState `json:"accountIds,omitempty"`
	FailedAccountIds []FailedMemberAccountEc2DeepInspectionStatusState `json:"failedAccountIds,omitempty"`
}

type CancelFindingsReportRequest struct {
	ReportId string `json:"reportId,omitempty"`
}

type CancelFindingsReportResponse struct {
	ReportId string `json:"reportId,omitempty"`
}

type CancelSbomExportRequest struct {
	ReportId string `json:"reportId,omitempty"`
}

type CancelSbomExportResponse struct {
	ReportId *string `json:"reportId,omitempty"`
}

type CisCheckAggregation struct {
	AccountId *string `json:"accountId,omitempty"`
	CheckDescription *string `json:"checkDescription,omitempty"`
	CheckId *string `json:"checkId,omitempty"`
	Level *string `json:"level,omitempty"`
	Platform *string `json:"platform,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	StatusCounts *StatusCounts `json:"statusCounts,omitempty"`
	Title *string `json:"title,omitempty"`
}

type CisDateFilter struct {
	EarliestScanStartTime *time.Time `json:"earliestScanStartTime,omitempty"`
	LatestScanStartTime *time.Time `json:"latestScanStartTime,omitempty"`
}

type CisFindingStatusFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisNumberFilter struct {
	LowerInclusive int `json:"lowerInclusive,omitempty"`
	UpperInclusive int `json:"upperInclusive,omitempty"`
}

type CisResultStatusFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisScan struct {
	FailedChecks int `json:"failedChecks,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
	ScanDate *time.Time `json:"scanDate,omitempty"`
	ScanName *string `json:"scanName,omitempty"`
	ScheduledBy *string `json:"scheduledBy,omitempty"`
	SecurityLevel *string `json:"securityLevel,omitempty"`
	Status *string `json:"status,omitempty"`
	Targets *CisTargets `json:"targets,omitempty"`
	TotalChecks int `json:"totalChecks,omitempty"`
}

type CisScanConfiguration struct {
	OwnerId *string `json:"ownerId,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
	ScanName *string `json:"scanName,omitempty"`
	Schedule *Schedule `json:"schedule,omitempty"`
	SecurityLevel *string `json:"securityLevel,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	Targets *CisTargets `json:"targets,omitempty"`
}

type CisScanResultDetails struct {
	AccountId *string `json:"accountId,omitempty"`
	CheckDescription *string `json:"checkDescription,omitempty"`
	CheckId *string `json:"checkId,omitempty"`
	FindingArn *string `json:"findingArn,omitempty"`
	Level *string `json:"level,omitempty"`
	Platform *string `json:"platform,omitempty"`
	Remediation *string `json:"remediation,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	Status *string `json:"status,omitempty"`
	StatusReason *string `json:"statusReason,omitempty"`
	TargetResourceId *string `json:"targetResourceId,omitempty"`
	Title *string `json:"title,omitempty"`
}

type CisScanResultDetailsFilterCriteria struct {
	CheckIdFilters []CisStringFilter `json:"checkIdFilters,omitempty"`
	FindingArnFilters []CisStringFilter `json:"findingArnFilters,omitempty"`
	FindingStatusFilters []CisFindingStatusFilter `json:"findingStatusFilters,omitempty"`
	SecurityLevelFilters []CisSecurityLevelFilter `json:"securityLevelFilters,omitempty"`
	TitleFilters []CisStringFilter `json:"titleFilters,omitempty"`
}

type CisScanResultsAggregatedByChecksFilterCriteria struct {
	AccountIdFilters []CisStringFilter `json:"accountIdFilters,omitempty"`
	CheckIdFilters []CisStringFilter `json:"checkIdFilters,omitempty"`
	FailedResourcesFilters []CisNumberFilter `json:"failedResourcesFilters,omitempty"`
	PlatformFilters []CisStringFilter `json:"platformFilters,omitempty"`
	SecurityLevelFilters []CisSecurityLevelFilter `json:"securityLevelFilters,omitempty"`
	TitleFilters []CisStringFilter `json:"titleFilters,omitempty"`
}

type CisScanResultsAggregatedByTargetResourceFilterCriteria struct {
	AccountIdFilters []CisStringFilter `json:"accountIdFilters,omitempty"`
	CheckIdFilters []CisStringFilter `json:"checkIdFilters,omitempty"`
	FailedChecksFilters []CisNumberFilter `json:"failedChecksFilters,omitempty"`
	PlatformFilters []CisStringFilter `json:"platformFilters,omitempty"`
	StatusFilters []CisResultStatusFilter `json:"statusFilters,omitempty"`
	TargetResourceIdFilters []CisStringFilter `json:"targetResourceIdFilters,omitempty"`
	TargetResourceTagFilters []TagFilter `json:"targetResourceTagFilters,omitempty"`
	TargetStatusFilters []CisTargetStatusFilter `json:"targetStatusFilters,omitempty"`
	TargetStatusReasonFilters []CisTargetStatusReasonFilter `json:"targetStatusReasonFilters,omitempty"`
}

type CisScanStatusFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisSecurityLevelFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisSessionMessage struct {
	CisRuleDetails []byte `json:"cisRuleDetails,omitempty"`
	RuleId string `json:"ruleId,omitempty"`
	Status string `json:"status,omitempty"`
}

type CisStringFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisTargetResourceAggregation struct {
	AccountId *string `json:"accountId,omitempty"`
	Platform *string `json:"platform,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	StatusCounts *StatusCounts `json:"statusCounts,omitempty"`
	TargetResourceId *string `json:"targetResourceId,omitempty"`
	TargetResourceTags map[string][]string `json:"targetResourceTags,omitempty"`
	TargetStatus *string `json:"targetStatus,omitempty"`
	TargetStatusReason *string `json:"targetStatusReason,omitempty"`
}

type CisTargetStatusFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisTargetStatusReasonFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CisTargets struct {
	AccountIds []string `json:"accountIds,omitempty"`
	TargetResourceTags map[string][]string `json:"targetResourceTags,omitempty"`
}

type CisaData struct {
	Action *string `json:"action,omitempty"`
	DateAdded *time.Time `json:"dateAdded,omitempty"`
	DateDue *time.Time `json:"dateDue,omitempty"`
}

type ClusterDetails struct {
	ClusterMetadata ClusterMetadata `json:"clusterMetadata,omitempty"`
	LastInUse time.Time `json:"lastInUse,omitempty"`
	RunningUnitCount int64 `json:"runningUnitCount,omitempty"`
	StoppedUnitCount int64 `json:"stoppedUnitCount,omitempty"`
}

type ClusterForImageFilterCriteria struct {
	ResourceId string `json:"resourceId,omitempty"`
}

type ClusterInformation struct {
	ClusterArn string `json:"clusterArn,omitempty"`
	ClusterDetails []ClusterDetails `json:"clusterDetails,omitempty"`
}

type ClusterMetadata struct {
	AwsEcsMetadataDetails *AwsEcsMetadataDetails `json:"awsEcsMetadataDetails,omitempty"`
	AwsEksMetadataDetails *AwsEksMetadataDetails `json:"awsEksMetadataDetails,omitempty"`
}

type CodeFilePath struct {
	EndLine int `json:"endLine,omitempty"`
	FileName string `json:"fileName,omitempty"`
	FilePath string `json:"filePath,omitempty"`
	StartLine int `json:"startLine,omitempty"`
}

type CodeLine struct {
	Content string `json:"content,omitempty"`
	LineNumber int `json:"lineNumber,omitempty"`
}

type CodeRepositoryAggregation struct {
	ProjectNames []StringFilter `json:"projectNames,omitempty"`
	ProviderTypes []StringFilter `json:"providerTypes,omitempty"`
	ResourceIds []StringFilter `json:"resourceIds,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type CodeRepositoryAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	ExploitAvailableActiveFindingsCount int64 `json:"exploitAvailableActiveFindingsCount,omitempty"`
	FixAvailableActiveFindingsCount int64 `json:"fixAvailableActiveFindingsCount,omitempty"`
	ProjectNames string `json:"projectNames,omitempty"`
	ProviderType *string `json:"providerType,omitempty"`
	ResourceId *string `json:"resourceId,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type CodeRepositoryDetails struct {
	IntegrationArn *string `json:"integrationArn,omitempty"`
	ProjectName *string `json:"projectName,omitempty"`
	ProviderType *string `json:"providerType,omitempty"`
}

type CodeRepositoryMetadata struct {
	IntegrationArn *string `json:"integrationArn,omitempty"`
	LastScannedCommitId *string `json:"lastScannedCommitId,omitempty"`
	OnDemandScan *CodeRepositoryOnDemandScan `json:"onDemandScan,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
	ProviderType string `json:"providerType,omitempty"`
	ProviderTypeVisibility string `json:"providerTypeVisibility,omitempty"`
	ScanConfiguration *ProjectCodeSecurityScanConfiguration `json:"scanConfiguration,omitempty"`
}

type CodeRepositoryOnDemandScan struct {
	LastScanAt *time.Time `json:"lastScanAt,omitempty"`
	LastScannedCommitId *string `json:"lastScannedCommitId,omitempty"`
	ScanStatus *ScanStatus `json:"scanStatus,omitempty"`
}

type CodeSecurityIntegrationSummary struct {
	CreatedOn time.Time `json:"createdOn,omitempty"`
	IntegrationArn string `json:"integrationArn,omitempty"`
	LastUpdateOn time.Time `json:"lastUpdateOn,omitempty"`
	Name string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	StatusReason string `json:"statusReason,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	Type string `json:"type,omitempty"`
}

type CodeSecurityResource struct {
	ProjectId *string `json:"projectId,omitempty"`
}

type CodeSecurityScanConfiguration struct {
	ContinuousIntegrationScanConfiguration *ContinuousIntegrationScanConfiguration `json:"continuousIntegrationScanConfiguration,omitempty"`
	PeriodicScanConfiguration *PeriodicScanConfiguration `json:"periodicScanConfiguration,omitempty"`
	RuleSetCategories []string `json:"ruleSetCategories,omitempty"`
}

type CodeSecurityScanConfigurationAssociationSummary struct {
	Resource *CodeSecurityResource `json:"resource,omitempty"`
}

type CodeSecurityScanConfigurationSummary struct {
	ContinuousIntegrationScanSupportedEvents []string `json:"continuousIntegrationScanSupportedEvents,omitempty"`
	FrequencyExpression *string `json:"frequencyExpression,omitempty"`
	Name string `json:"name,omitempty"`
	OwnerAccountId string `json:"ownerAccountId,omitempty"`
	PeriodicScanFrequency *string `json:"periodicScanFrequency,omitempty"`
	RuleSetCategories []string `json:"ruleSetCategories,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
	ScopeSettings *ScopeSettings `json:"scopeSettings,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CodeSnippetError struct {
	ErrorCode string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	FindingArn string `json:"findingArn,omitempty"`
}

type CodeSnippetResult struct {
	CodeSnippet []CodeLine `json:"codeSnippet,omitempty"`
	EndLine int `json:"endLine,omitempty"`
	FindingArn *string `json:"findingArn,omitempty"`
	StartLine int `json:"startLine,omitempty"`
	SuggestedFixes []SuggestedFix `json:"suggestedFixes,omitempty"`
}

type CodeVulnerabilityDetails struct {
	Cwes []string `json:"cwes,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	DetectorName string `json:"detectorName,omitempty"`
	DetectorTags []string `json:"detectorTags,omitempty"`
	FilePath CodeFilePath `json:"filePath,omitempty"`
	ReferenceUrls []string `json:"referenceUrls,omitempty"`
	RuleId *string `json:"ruleId,omitempty"`
	SourceLambdaLayerArn *string `json:"sourceLambdaLayerArn,omitempty"`
}

type ComputePlatform struct {
	Product *string `json:"product,omitempty"`
	Vendor *string `json:"vendor,omitempty"`
	Version *string `json:"version,omitempty"`
}

type ContinuousIntegrationScanConfiguration struct {
	SupportedEvents []string `json:"supportedEvents,omitempty"`
}

type Counts struct {
	Count int64 `json:"count,omitempty"`
	GroupKey *string `json:"groupKey,omitempty"`
}

type CoverageDateFilter struct {
	EndInclusive *time.Time `json:"endInclusive,omitempty"`
	StartInclusive *time.Time `json:"startInclusive,omitempty"`
}

type CoverageFilterCriteria struct {
	AccountId []CoverageStringFilter `json:"accountId,omitempty"`
	CodeRepositoryProjectName []CoverageStringFilter `json:"codeRepositoryProjectName,omitempty"`
	CodeRepositoryProviderType []CoverageStringFilter `json:"codeRepositoryProviderType,omitempty"`
	CodeRepositoryProviderTypeVisibility []CoverageStringFilter `json:"codeRepositoryProviderTypeVisibility,omitempty"`
	Ec2InstanceTags []CoverageMapFilter `json:"ec2InstanceTags,omitempty"`
	EcrImageInUseCount []CoverageNumberFilter `json:"ecrImageInUseCount,omitempty"`
	EcrImageLastInUseAt []CoverageDateFilter `json:"ecrImageLastInUseAt,omitempty"`
	EcrImageTags []CoverageStringFilter `json:"ecrImageTags,omitempty"`
	EcrRepositoryName []CoverageStringFilter `json:"ecrRepositoryName,omitempty"`
	ImagePulledAt []CoverageDateFilter `json:"imagePulledAt,omitempty"`
	LambdaFunctionName []CoverageStringFilter `json:"lambdaFunctionName,omitempty"`
	LambdaFunctionRuntime []CoverageStringFilter `json:"lambdaFunctionRuntime,omitempty"`
	LambdaFunctionTags []CoverageMapFilter `json:"lambdaFunctionTags,omitempty"`
	LastScannedAt []CoverageDateFilter `json:"lastScannedAt,omitempty"`
	LastScannedCommitId []CoverageStringFilter `json:"lastScannedCommitId,omitempty"`
	ResourceId []CoverageStringFilter `json:"resourceId,omitempty"`
	ResourceType []CoverageStringFilter `json:"resourceType,omitempty"`
	ScanMode []CoverageStringFilter `json:"scanMode,omitempty"`
	ScanStatusCode []CoverageStringFilter `json:"scanStatusCode,omitempty"`
	ScanStatusReason []CoverageStringFilter `json:"scanStatusReason,omitempty"`
	ScanType []CoverageStringFilter `json:"scanType,omitempty"`
}

type CoverageMapFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Key string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type CoverageNumberFilter struct {
	LowerInclusive int64 `json:"lowerInclusive,omitempty"`
	UpperInclusive int64 `json:"upperInclusive,omitempty"`
}

type CoverageStringFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type CoveredResource struct {
	AccountId string `json:"accountId,omitempty"`
	LastScannedAt *time.Time `json:"lastScannedAt,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
	ResourceMetadata *ResourceScanMetadata `json:"resourceMetadata,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	ScanMode *string `json:"scanMode,omitempty"`
	ScanStatus *ScanStatus `json:"scanStatus,omitempty"`
	ScanType string `json:"scanType,omitempty"`
}

type CreateCisScanConfigurationRequest struct {
	ScanName string `json:"scanName,omitempty"`
	Schedule Schedule `json:"schedule,omitempty"`
	SecurityLevel string `json:"securityLevel,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	Targets CreateCisTargets `json:"targets,omitempty"`
}

type CreateCisScanConfigurationResponse struct {
	ScanConfigurationArn *string `json:"scanConfigurationArn,omitempty"`
}

type CreateCisTargets struct {
	AccountIds []string `json:"accountIds,omitempty"`
	TargetResourceTags map[string][]string `json:"targetResourceTags,omitempty"`
}

type CreateCodeSecurityIntegrationRequest struct {
	Details *CreateIntegrationDetail `json:"details,omitempty"`
	Name string `json:"name,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	Type string `json:"type,omitempty"`
}

type CreateCodeSecurityIntegrationResponse struct {
	AuthorizationUrl *string `json:"authorizationUrl,omitempty"`
	IntegrationArn string `json:"integrationArn,omitempty"`
	Status string `json:"status,omitempty"`
}

type CreateCodeSecurityScanConfigurationRequest struct {
	Configuration CodeSecurityScanConfiguration `json:"configuration,omitempty"`
	Level string `json:"level,omitempty"`
	Name string `json:"name,omitempty"`
	ScopeSettings *ScopeSettings `json:"scopeSettings,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateCodeSecurityScanConfigurationResponse struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type CreateFilterRequest struct {
	Action string `json:"action,omitempty"`
	Description *string `json:"description,omitempty"`
	FilterCriteria FilterCriteria `json:"filterCriteria,omitempty"`
	Name string `json:"name,omitempty"`
	Reason *string `json:"reason,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateFilterResponse struct {
	Arn string `json:"arn,omitempty"`
}

type CreateFindingsReportRequest struct {
	FilterCriteria *FilterCriteria `json:"filterCriteria,omitempty"`
	ReportFormat string `json:"reportFormat,omitempty"`
	S3Destination Destination `json:"s3Destination,omitempty"`
}

type CreateFindingsReportResponse struct {
	ReportId *string `json:"reportId,omitempty"`
}

type CreateGitLabSelfManagedIntegrationDetail struct {
	AccessToken string `json:"accessToken,omitempty"`
	InstanceUrl string `json:"instanceUrl,omitempty"`
}

type CreateIntegrationDetail struct {
	GitlabSelfManaged *CreateGitLabSelfManagedIntegrationDetail `json:"gitlabSelfManaged,omitempty"`
}

type CreateSbomExportRequest struct {
	ReportFormat string `json:"reportFormat,omitempty"`
	ResourceFilterCriteria *ResourceFilterCriteria `json:"resourceFilterCriteria,omitempty"`
	S3Destination Destination `json:"s3Destination,omitempty"`
}

type CreateSbomExportResponse struct {
	ReportId *string `json:"reportId,omitempty"`
}

type Cvss2 struct {
	BaseScore float64 `json:"baseScore,omitempty"`
	ScoringVector *string `json:"scoringVector,omitempty"`
}

type Cvss3 struct {
	BaseScore float64 `json:"baseScore,omitempty"`
	ScoringVector *string `json:"scoringVector,omitempty"`
}

type Cvss4 struct {
	BaseScore float64 `json:"baseScore,omitempty"`
	ScoringVector *string `json:"scoringVector,omitempty"`
}

type CvssScore struct {
	BaseScore float64 `json:"baseScore,omitempty"`
	ScoringVector string `json:"scoringVector,omitempty"`
	Source string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}

type CvssScoreAdjustment struct {
	Metric string `json:"metric,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type CvssScoreDetails struct {
	Adjustments []CvssScoreAdjustment `json:"adjustments,omitempty"`
	CvssSource *string `json:"cvssSource,omitempty"`
	Score float64 `json:"score,omitempty"`
	ScoreSource string `json:"scoreSource,omitempty"`
	ScoringVector string `json:"scoringVector,omitempty"`
	Version string `json:"version,omitempty"`
}

type DailySchedule struct {
	StartTime Time `json:"startTime,omitempty"`
}

type DateFilter struct {
	EndInclusive *time.Time `json:"endInclusive,omitempty"`
	StartInclusive *time.Time `json:"startInclusive,omitempty"`
}

type DelegatedAdmin struct {
	AccountId *string `json:"accountId,omitempty"`
	RelationshipStatus *string `json:"relationshipStatus,omitempty"`
}

type DelegatedAdminAccount struct {
	AccountId *string `json:"accountId,omitempty"`
	Status *string `json:"status,omitempty"`
}

type DeleteCisScanConfigurationRequest struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type DeleteCisScanConfigurationResponse struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type DeleteCodeSecurityIntegrationRequest struct {
	IntegrationArn string `json:"integrationArn,omitempty"`
}

type DeleteCodeSecurityIntegrationResponse struct {
	IntegrationArn *string `json:"integrationArn,omitempty"`
}

type DeleteCodeSecurityScanConfigurationRequest struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type DeleteCodeSecurityScanConfigurationResponse struct {
	ScanConfigurationArn *string `json:"scanConfigurationArn,omitempty"`
}

type DeleteFilterRequest struct {
	Arn string `json:"arn,omitempty"`
}

type DeleteFilterResponse struct {
	Arn string `json:"arn,omitempty"`
}

type DescribeOrganizationConfigurationRequest struct {
}

type DescribeOrganizationConfigurationResponse struct {
	AutoEnable *AutoEnable `json:"autoEnable,omitempty"`
	MaxAccountLimitReached bool `json:"maxAccountLimitReached,omitempty"`
}

type Destination struct {
	BucketName string `json:"bucketName,omitempty"`
	KeyPrefix *string `json:"keyPrefix,omitempty"`
	KmsKeyArn string `json:"kmsKeyArn,omitempty"`
}

type DisableDelegatedAdminAccountRequest struct {
	DelegatedAdminAccountId string `json:"delegatedAdminAccountId,omitempty"`
}

type DisableDelegatedAdminAccountResponse struct {
	DelegatedAdminAccountId string `json:"delegatedAdminAccountId,omitempty"`
}

type DisableRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	ResourceTypes []string `json:"resourceTypes,omitempty"`
}

type DisableResponse struct {
	Accounts []Account `json:"accounts,omitempty"`
	FailedAccounts []FailedAccount `json:"failedAccounts,omitempty"`
}

type DisassociateConfigurationRequest struct {
	Resource CodeSecurityResource `json:"resource,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type DisassociateMemberRequest struct {
	AccountId string `json:"accountId,omitempty"`
}

type DisassociateMemberResponse struct {
	AccountId string `json:"accountId,omitempty"`
}

type Ec2Configuration struct {
	ScanMode string `json:"scanMode,omitempty"`
}

type Ec2ConfigurationState struct {
	ScanModeState *Ec2ScanModeState `json:"scanModeState,omitempty"`
}

type Ec2InstanceAggregation struct {
	Amis []StringFilter `json:"amis,omitempty"`
	InstanceIds []StringFilter `json:"instanceIds,omitempty"`
	InstanceTags []MapFilter `json:"instanceTags,omitempty"`
	OperatingSystems []StringFilter `json:"operatingSystems,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type Ec2InstanceAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	Ami *string `json:"ami,omitempty"`
	InstanceId string `json:"instanceId,omitempty"`
	InstanceTags map[string]string `json:"instanceTags,omitempty"`
	NetworkFindings int64 `json:"networkFindings,omitempty"`
	OperatingSystem *string `json:"operatingSystem,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type Ec2Metadata struct {
	AmiId *string `json:"amiId,omitempty"`
	Platform *string `json:"platform,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type Ec2ScanModeState struct {
	ScanMode *string `json:"scanMode,omitempty"`
	ScanModeStatus *string `json:"scanModeStatus,omitempty"`
}

type EcrConfiguration struct {
	PullDateRescanDuration *string `json:"pullDateRescanDuration,omitempty"`
	PullDateRescanMode *string `json:"pullDateRescanMode,omitempty"`
	RescanDuration string `json:"rescanDuration,omitempty"`
}

type EcrConfigurationState struct {
	RescanDurationState *EcrRescanDurationState `json:"rescanDurationState,omitempty"`
}

type EcrContainerImageMetadata struct {
	ImagePulledAt *time.Time `json:"imagePulledAt,omitempty"`
	InUseCount int64 `json:"inUseCount,omitempty"`
	LastInUseAt *time.Time `json:"lastInUseAt,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type EcrRepositoryMetadata struct {
	Name *string `json:"name,omitempty"`
	ScanFrequency *string `json:"scanFrequency,omitempty"`
}

type EcrRescanDurationState struct {
	PullDateRescanDuration *string `json:"pullDateRescanDuration,omitempty"`
	PullDateRescanMode *string `json:"pullDateRescanMode,omitempty"`
	RescanDuration *string `json:"rescanDuration,omitempty"`
	Status *string `json:"status,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type EnableDelegatedAdminAccountRequest struct {
	ClientToken *string `json:"clientToken,omitempty"`
	DelegatedAdminAccountId string `json:"delegatedAdminAccountId,omitempty"`
}

type EnableDelegatedAdminAccountResponse struct {
	DelegatedAdminAccountId string `json:"delegatedAdminAccountId,omitempty"`
}

type EnableRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	ResourceTypes []string `json:"resourceTypes,omitempty"`
}

type EnableResponse struct {
	Accounts []Account `json:"accounts,omitempty"`
	FailedAccounts []FailedAccount `json:"failedAccounts,omitempty"`
}

type Epss struct {
	Score float64 `json:"score,omitempty"`
}

type EpssDetails struct {
	Score float64 `json:"score,omitempty"`
}

type Evidence struct {
	EvidenceDetail *string `json:"evidenceDetail,omitempty"`
	EvidenceRule *string `json:"evidenceRule,omitempty"`
	Severity *string `json:"severity,omitempty"`
}

type ExploitObserved struct {
	FirstSeen *time.Time `json:"firstSeen,omitempty"`
	LastSeen *time.Time `json:"lastSeen,omitempty"`
}

type ExploitabilityDetails struct {
	LastKnownExploitAt *time.Time `json:"lastKnownExploitAt,omitempty"`
}

type FailedAccount struct {
	AccountId string `json:"accountId,omitempty"`
	ErrorCode string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ResourceStatus *ResourceStatus `json:"resourceStatus,omitempty"`
	Status *string `json:"status,omitempty"`
}

type FailedAssociationResult struct {
	Resource *CodeSecurityResource `json:"resource,omitempty"`
	ScanConfigurationArn *string `json:"scanConfigurationArn,omitempty"`
	StatusCode *string `json:"statusCode,omitempty"`
	StatusMessage *string `json:"statusMessage,omitempty"`
}

type FailedMemberAccountEc2DeepInspectionStatusState struct {
	AccountId string `json:"accountId,omitempty"`
	Ec2ScanStatus *string `json:"ec2ScanStatus,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

type Filter struct {
	Action string `json:"action,omitempty"`
	Arn string `json:"arn,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	Criteria FilterCriteria `json:"criteria,omitempty"`
	Description *string `json:"description,omitempty"`
	Name string `json:"name,omitempty"`
	OwnerId string `json:"ownerId,omitempty"`
	Reason *string `json:"reason,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type FilterCriteria struct {
	AwsAccountId []StringFilter `json:"awsAccountId,omitempty"`
	CodeRepositoryProjectName []StringFilter `json:"codeRepositoryProjectName,omitempty"`
	CodeRepositoryProviderType []StringFilter `json:"codeRepositoryProviderType,omitempty"`
	CodeVulnerabilityDetectorName []StringFilter `json:"codeVulnerabilityDetectorName,omitempty"`
	CodeVulnerabilityDetectorTags []StringFilter `json:"codeVulnerabilityDetectorTags,omitempty"`
	CodeVulnerabilityFilePath []StringFilter `json:"codeVulnerabilityFilePath,omitempty"`
	ComponentId []StringFilter `json:"componentId,omitempty"`
	ComponentType []StringFilter `json:"componentType,omitempty"`
	Ec2InstanceImageId []StringFilter `json:"ec2InstanceImageId,omitempty"`
	Ec2InstanceSubnetId []StringFilter `json:"ec2InstanceSubnetId,omitempty"`
	Ec2InstanceVpcId []StringFilter `json:"ec2InstanceVpcId,omitempty"`
	EcrImageArchitecture []StringFilter `json:"ecrImageArchitecture,omitempty"`
	EcrImageHash []StringFilter `json:"ecrImageHash,omitempty"`
	EcrImageInUseCount []NumberFilter `json:"ecrImageInUseCount,omitempty"`
	EcrImageLastInUseAt []DateFilter `json:"ecrImageLastInUseAt,omitempty"`
	EcrImagePushedAt []DateFilter `json:"ecrImagePushedAt,omitempty"`
	EcrImageRegistry []StringFilter `json:"ecrImageRegistry,omitempty"`
	EcrImageRepositoryName []StringFilter `json:"ecrImageRepositoryName,omitempty"`
	EcrImageTags []StringFilter `json:"ecrImageTags,omitempty"`
	EpssScore []NumberFilter `json:"epssScore,omitempty"`
	ExploitAvailable []StringFilter `json:"exploitAvailable,omitempty"`
	FindingArn []StringFilter `json:"findingArn,omitempty"`
	FindingStatus []StringFilter `json:"findingStatus,omitempty"`
	FindingType []StringFilter `json:"findingType,omitempty"`
	FirstObservedAt []DateFilter `json:"firstObservedAt,omitempty"`
	FixAvailable []StringFilter `json:"fixAvailable,omitempty"`
	InspectorScore []NumberFilter `json:"inspectorScore,omitempty"`
	LambdaFunctionExecutionRoleArn []StringFilter `json:"lambdaFunctionExecutionRoleArn,omitempty"`
	LambdaFunctionLastModifiedAt []DateFilter `json:"lambdaFunctionLastModifiedAt,omitempty"`
	LambdaFunctionLayers []StringFilter `json:"lambdaFunctionLayers,omitempty"`
	LambdaFunctionName []StringFilter `json:"lambdaFunctionName,omitempty"`
	LambdaFunctionRuntime []StringFilter `json:"lambdaFunctionRuntime,omitempty"`
	LastObservedAt []DateFilter `json:"lastObservedAt,omitempty"`
	NetworkProtocol []StringFilter `json:"networkProtocol,omitempty"`
	PortRange []PortRangeFilter `json:"portRange,omitempty"`
	RelatedVulnerabilities []StringFilter `json:"relatedVulnerabilities,omitempty"`
	ResourceId []StringFilter `json:"resourceId,omitempty"`
	ResourceTags []MapFilter `json:"resourceTags,omitempty"`
	ResourceType []StringFilter `json:"resourceType,omitempty"`
	Severity []StringFilter `json:"severity,omitempty"`
	Title []StringFilter `json:"title,omitempty"`
	UpdatedAt []DateFilter `json:"updatedAt,omitempty"`
	VendorSeverity []StringFilter `json:"vendorSeverity,omitempty"`
	VulnerabilityId []StringFilter `json:"vulnerabilityId,omitempty"`
	VulnerabilitySource []StringFilter `json:"vulnerabilitySource,omitempty"`
	VulnerablePackages []PackageFilter `json:"vulnerablePackages,omitempty"`
}

type Finding struct {
	AwsAccountId string `json:"awsAccountId,omitempty"`
	CodeVulnerabilityDetails *CodeVulnerabilityDetails `json:"codeVulnerabilityDetails,omitempty"`
	Description string `json:"description,omitempty"`
	Epss *EpssDetails `json:"epss,omitempty"`
	ExploitAvailable *string `json:"exploitAvailable,omitempty"`
	ExploitabilityDetails *ExploitabilityDetails `json:"exploitabilityDetails,omitempty"`
	FindingArn string `json:"findingArn,omitempty"`
	FirstObservedAt time.Time `json:"firstObservedAt,omitempty"`
	FixAvailable *string `json:"fixAvailable,omitempty"`
	InspectorScore float64 `json:"inspectorScore,omitempty"`
	InspectorScoreDetails *InspectorScoreDetails `json:"inspectorScoreDetails,omitempty"`
	LastObservedAt time.Time `json:"lastObservedAt,omitempty"`
	NetworkReachabilityDetails *NetworkReachabilityDetails `json:"networkReachabilityDetails,omitempty"`
	PackageVulnerabilityDetails *PackageVulnerabilityDetails `json:"packageVulnerabilityDetails,omitempty"`
	Remediation Remediation `json:"remediation,omitempty"`
	Resources []Resource `json:"resources,omitempty"`
	Severity string `json:"severity,omitempty"`
	Status string `json:"status,omitempty"`
	Title *string `json:"title,omitempty"`
	Type string `json:"type,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type FindingDetail struct {
	CisaData *CisaData `json:"cisaData,omitempty"`
	Cwes []string `json:"cwes,omitempty"`
	EpssScore float64 `json:"epssScore,omitempty"`
	Evidences []Evidence `json:"evidences,omitempty"`
	ExploitObserved *ExploitObserved `json:"exploitObserved,omitempty"`
	FindingArn *string `json:"findingArn,omitempty"`
	ReferenceUrls []string `json:"referenceUrls,omitempty"`
	RiskScore int `json:"riskScore,omitempty"`
	Tools []string `json:"tools,omitempty"`
	Ttps []string `json:"ttps,omitempty"`
}

type FindingDetailsError struct {
	ErrorCode string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	FindingArn string `json:"findingArn,omitempty"`
}

type FindingTypeAggregation struct {
	FindingType *string `json:"findingType,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type FindingTypeAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	ExploitAvailableCount int64 `json:"exploitAvailableCount,omitempty"`
	FixAvailableCount int64 `json:"fixAvailableCount,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type FreeTrialAccountInfo struct {
	AccountId string `json:"accountId,omitempty"`
	FreeTrialInfo []FreeTrialInfo `json:"freeTrialInfo,omitempty"`
}

type FreeTrialInfo struct {
	End time.Time `json:"end,omitempty"`
	Start time.Time `json:"start,omitempty"`
	Status string `json:"status,omitempty"`
	Type string `json:"type,omitempty"`
}

type FreeTrialInfoError struct {
	AccountId string `json:"accountId,omitempty"`
	Code string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type GetCisScanReportRequest struct {
	ReportFormat *string `json:"reportFormat,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	TargetAccounts []string `json:"targetAccounts,omitempty"`
}

type GetCisScanReportResponse struct {
	Status *string `json:"status,omitempty"`
	Url *string `json:"url,omitempty"`
}

type GetCisScanResultDetailsRequest struct {
	AccountId string `json:"accountId,omitempty"`
	FilterCriteria *CisScanResultDetailsFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
	TargetResourceId string `json:"targetResourceId,omitempty"`
}

type GetCisScanResultDetailsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	ScanResultDetails []CisScanResultDetails `json:"scanResultDetails,omitempty"`
}

type GetClustersForImageRequest struct {
	Filter ClusterForImageFilterCriteria `json:"filter,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetClustersForImageResponse struct {
	Cluster []ClusterInformation `json:"cluster,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetCodeSecurityIntegrationRequest struct {
	IntegrationArn string `json:"integrationArn,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type GetCodeSecurityIntegrationResponse struct {
	AuthorizationUrl *string `json:"authorizationUrl,omitempty"`
	CreatedOn time.Time `json:"createdOn,omitempty"`
	IntegrationArn string `json:"integrationArn,omitempty"`
	LastUpdateOn time.Time `json:"lastUpdateOn,omitempty"`
	Name string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	StatusReason string `json:"statusReason,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	Type string `json:"type,omitempty"`
}

type GetCodeSecurityScanConfigurationRequest struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type GetCodeSecurityScanConfigurationResponse struct {
	Configuration *CodeSecurityScanConfiguration `json:"configuration,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt,omitempty"`
	Level *string `json:"level,omitempty"`
	Name *string `json:"name,omitempty"`
	ScanConfigurationArn *string `json:"scanConfigurationArn,omitempty"`
	ScopeSettings *ScopeSettings `json:"scopeSettings,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type GetCodeSecurityScanRequest struct {
	Resource CodeSecurityResource `json:"resource,omitempty"`
	ScanId string `json:"scanId,omitempty"`
}

type GetCodeSecurityScanResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	LastCommitId *string `json:"lastCommitId,omitempty"`
	Resource *CodeSecurityResource `json:"resource,omitempty"`
	ScanId *string `json:"scanId,omitempty"`
	Status *string `json:"status,omitempty"`
	StatusReason *string `json:"statusReason,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type GetConfigurationRequest struct {
}

type GetConfigurationResponse struct {
	Ec2Configuration *Ec2ConfigurationState `json:"ec2Configuration,omitempty"`
	EcrConfiguration *EcrConfigurationState `json:"ecrConfiguration,omitempty"`
}

type GetDelegatedAdminAccountRequest struct {
}

type GetDelegatedAdminAccountResponse struct {
	DelegatedAdmin *DelegatedAdmin `json:"delegatedAdmin,omitempty"`
}

type GetEc2DeepInspectionConfigurationRequest struct {
}

type GetEc2DeepInspectionConfigurationResponse struct {
	ErrorMessage *string `json:"errorMessage,omitempty"`
	OrgPackagePaths []string `json:"orgPackagePaths,omitempty"`
	PackagePaths []string `json:"packagePaths,omitempty"`
	Status *string `json:"status,omitempty"`
}

type GetEncryptionKeyRequest struct {
	ResourceType string `json:"resourceType,omitempty"`
	ScanType string `json:"scanType,omitempty"`
}

type GetEncryptionKeyResponse struct {
	KmsKeyId string `json:"kmsKeyId,omitempty"`
}

type GetFindingsReportStatusRequest struct {
	ReportId *string `json:"reportId,omitempty"`
}

type GetFindingsReportStatusResponse struct {
	Destination *Destination `json:"destination,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	FilterCriteria *FilterCriteria `json:"filterCriteria,omitempty"`
	ReportId *string `json:"reportId,omitempty"`
	Status *string `json:"status,omitempty"`
}

type GetMemberRequest struct {
	AccountId string `json:"accountId,omitempty"`
}

type GetMemberResponse struct {
	Member *Member `json:"member,omitempty"`
}

type GetSbomExportRequest struct {
	ReportId string `json:"reportId,omitempty"`
}

type GetSbomExportResponse struct {
	ErrorCode *string `json:"errorCode,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	FilterCriteria *ResourceFilterCriteria `json:"filterCriteria,omitempty"`
	Format *string `json:"format,omitempty"`
	ReportId *string `json:"reportId,omitempty"`
	S3Destination *Destination `json:"s3Destination,omitempty"`
	Status *string `json:"status,omitempty"`
}

type ImageLayerAggregation struct {
	LayerHashes []StringFilter `json:"layerHashes,omitempty"`
	Repositories []StringFilter `json:"repositories,omitempty"`
	ResourceIds []StringFilter `json:"resourceIds,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type ImageLayerAggregationResponse struct {
	AccountId string `json:"accountId,omitempty"`
	LayerHash string `json:"layerHash,omitempty"`
	Repository string `json:"repository,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type InspectorScoreDetails struct {
	AdjustedCvss *CvssScoreDetails `json:"adjustedCvss,omitempty"`
}

type LambdaFunctionAggregation struct {
	FunctionNames []StringFilter `json:"functionNames,omitempty"`
	FunctionTags []MapFilter `json:"functionTags,omitempty"`
	ResourceIds []StringFilter `json:"resourceIds,omitempty"`
	Runtimes []StringFilter `json:"runtimes,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type LambdaFunctionAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	FunctionName *string `json:"functionName,omitempty"`
	LambdaTags map[string]string `json:"lambdaTags,omitempty"`
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
	Runtime *string `json:"runtime,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type LambdaFunctionMetadata struct {
	FunctionName *string `json:"functionName,omitempty"`
	FunctionTags map[string]string `json:"functionTags,omitempty"`
	Layers []string `json:"layers,omitempty"`
	Runtime *string `json:"runtime,omitempty"`
}

type LambdaLayerAggregation struct {
	FunctionNames []StringFilter `json:"functionNames,omitempty"`
	LayerArns []StringFilter `json:"layerArns,omitempty"`
	ResourceIds []StringFilter `json:"resourceIds,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type LambdaLayerAggregationResponse struct {
	AccountId string `json:"accountId,omitempty"`
	FunctionName string `json:"functionName,omitempty"`
	LayerArn string `json:"layerArn,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type LambdaVpcConfig struct {
	SecurityGroupIds []string `json:"securityGroupIds,omitempty"`
	SubnetIds []string `json:"subnetIds,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
}

type ListAccountPermissionsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	ServiceModel *string `json:"service,omitempty"`
}

type ListAccountPermissionsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type ListCisScanConfigurationsFilterCriteria struct {
	ScanConfigurationArnFilters []CisStringFilter `json:"scanConfigurationArnFilters,omitempty"`
	ScanNameFilters []CisStringFilter `json:"scanNameFilters,omitempty"`
	TargetResourceTagFilters []TagFilter `json:"targetResourceTagFilters,omitempty"`
}

type ListCisScanConfigurationsRequest struct {
	FilterCriteria *ListCisScanConfigurationsFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type ListCisScanConfigurationsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	ScanConfigurations []CisScanConfiguration `json:"scanConfigurations,omitempty"`
}

type ListCisScanResultsAggregatedByChecksRequest struct {
	FilterCriteria *CisScanResultsAggregatedByChecksFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type ListCisScanResultsAggregatedByChecksResponse struct {
	CheckAggregations []CisCheckAggregation `json:"checkAggregations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCisScanResultsAggregatedByTargetResourceRequest struct {
	FilterCriteria *CisScanResultsAggregatedByTargetResourceFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	ScanArn string `json:"scanArn,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type ListCisScanResultsAggregatedByTargetResourceResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	TargetResourceAggregations []CisTargetResourceAggregation `json:"targetResourceAggregations,omitempty"`
}

type ListCisScansFilterCriteria struct {
	FailedChecksFilters []CisNumberFilter `json:"failedChecksFilters,omitempty"`
	ScanArnFilters []CisStringFilter `json:"scanArnFilters,omitempty"`
	ScanAtFilters []CisDateFilter `json:"scanAtFilters,omitempty"`
	ScanConfigurationArnFilters []CisStringFilter `json:"scanConfigurationArnFilters,omitempty"`
	ScanNameFilters []CisStringFilter `json:"scanNameFilters,omitempty"`
	ScanStatusFilters []CisScanStatusFilter `json:"scanStatusFilters,omitempty"`
	ScheduledByFilters []CisStringFilter `json:"scheduledByFilters,omitempty"`
	TargetAccountIdFilters []CisStringFilter `json:"targetAccountIdFilters,omitempty"`
	TargetResourceIdFilters []CisStringFilter `json:"targetResourceIdFilters,omitempty"`
	TargetResourceTagFilters []TagFilter `json:"targetResourceTagFilters,omitempty"`
}

type ListCisScansRequest struct {
	DetailLevel *string `json:"detailLevel,omitempty"`
	FilterCriteria *ListCisScansFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type ListCisScansResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Scans []CisScan `json:"scans,omitempty"`
}

type ListCodeSecurityIntegrationsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCodeSecurityIntegrationsResponse struct {
	Integrations []CodeSecurityIntegrationSummary `json:"integrations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCodeSecurityScanConfigurationAssociationsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type ListCodeSecurityScanConfigurationAssociationsResponse struct {
	Associations []CodeSecurityScanConfigurationAssociationSummary `json:"associations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCodeSecurityScanConfigurationsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCodeSecurityScanConfigurationsResponse struct {
	Configurations []CodeSecurityScanConfigurationSummary `json:"configurations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCoverageRequest struct {
	FilterCriteria *CoverageFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCoverageResponse struct {
	CoveredResources []CoveredResource `json:"coveredResources,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCoverageStatisticsRequest struct {
	FilterCriteria *CoverageFilterCriteria `json:"filterCriteria,omitempty"`
	GroupBy *string `json:"groupBy,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListCoverageStatisticsResponse struct {
	CountsByGroup []Counts `json:"countsByGroup,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	TotalCounts int64 `json:"totalCounts,omitempty"`
}

type ListDelegatedAdminAccountsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListDelegatedAdminAccountsResponse struct {
	DelegatedAdminAccounts []DelegatedAdminAccount `json:"delegatedAdminAccounts,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFiltersRequest struct {
	Action *string `json:"action,omitempty"`
	Arns []string `json:"arns,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFiltersResponse struct {
	Filters []Filter `json:"filters,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFindingAggregationsRequest struct {
	AccountIds []StringFilter `json:"accountIds,omitempty"`
	AggregationRequest *AggregationRequest `json:"aggregationRequest,omitempty"`
	AggregationType string `json:"aggregationType,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFindingAggregationsResponse struct {
	AggregationType string `json:"aggregationType,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	Responses []AggregationResponse `json:"responses,omitempty"`
}

type ListFindingsRequest struct {
	FilterCriteria *FilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortCriteria *SortCriteria `json:"sortCriteria,omitempty"`
}

type ListFindingsResponse struct {
	Findings []Finding `json:"findings,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListMembersRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	OnlyAssociated bool `json:"onlyAssociated,omitempty"`
}

type ListMembersResponse struct {
	Members []Member `json:"members,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags map[string]string `json:"tags,omitempty"`
}

type ListUsageTotalsRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListUsageTotalsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Totals []UsageTotal `json:"totals,omitempty"`
}

type MapFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Key string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type Member struct {
	AccountId *string `json:"accountId,omitempty"`
	DelegatedAdminAccountId *string `json:"delegatedAdminAccountId,omitempty"`
	RelationshipStatus *string `json:"relationshipStatus,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type MemberAccountEc2DeepInspectionStatus struct {
	AccountId string `json:"accountId,omitempty"`
	ActivateDeepInspection bool `json:"activateDeepInspection,omitempty"`
}

type MemberAccountEc2DeepInspectionStatusState struct {
	AccountId string `json:"accountId,omitempty"`
	ErrorMessage *string `json:"errorMessage,omitempty"`
	Status *string `json:"status,omitempty"`
}

type MonthlySchedule struct {
	Day string `json:"day,omitempty"`
	StartTime Time `json:"startTime,omitempty"`
}

type NetworkPath struct {
	Steps []Step `json:"steps,omitempty"`
}

type NetworkReachabilityDetails struct {
	NetworkPath NetworkPath `json:"networkPath,omitempty"`
	OpenPortRange PortRange `json:"openPortRange,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type NumberFilter struct {
	LowerInclusive float64 `json:"lowerInclusive,omitempty"`
	UpperInclusive float64 `json:"upperInclusive,omitempty"`
}

type OneTimeSchedule struct {
}

type PackageAggregation struct {
	PackageNames []StringFilter `json:"packageNames,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type PackageAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	PackageName string `json:"packageName,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type PackageFilter struct {
	Architecture *StringFilter `json:"architecture,omitempty"`
	Epoch *NumberFilter `json:"epoch,omitempty"`
	FilePath *StringFilter `json:"filePath,omitempty"`
	Name *StringFilter `json:"name,omitempty"`
	Release *StringFilter `json:"release,omitempty"`
	SourceLambdaLayerArn *StringFilter `json:"sourceLambdaLayerArn,omitempty"`
	SourceLayerHash *StringFilter `json:"sourceLayerHash,omitempty"`
	Version *StringFilter `json:"version,omitempty"`
}

type PackageVulnerabilityDetails struct {
	Cvss []CvssScore `json:"cvss,omitempty"`
	ReferenceUrls []string `json:"referenceUrls,omitempty"`
	RelatedVulnerabilities []string `json:"relatedVulnerabilities,omitempty"`
	Source string `json:"source,omitempty"`
	SourceUrl *string `json:"sourceUrl,omitempty"`
	VendorCreatedAt *time.Time `json:"vendorCreatedAt,omitempty"`
	VendorSeverity *string `json:"vendorSeverity,omitempty"`
	VendorUpdatedAt *time.Time `json:"vendorUpdatedAt,omitempty"`
	VulnerabilityId string `json:"vulnerabilityId,omitempty"`
	VulnerablePackages []VulnerablePackage `json:"vulnerablePackages,omitempty"`
}

type PeriodicScanConfiguration struct {
	Frequency *string `json:"frequency,omitempty"`
	FrequencyExpression *string `json:"frequencyExpression,omitempty"`
}

type Permission struct {
	Operation string `json:"operation,omitempty"`
	ServiceModel string `json:"service,omitempty"`
}

type PortRange struct {
	Begin int `json:"begin,omitempty"`
	End int `json:"end,omitempty"`
}

type PortRangeFilter struct {
	BeginInclusive int `json:"beginInclusive,omitempty"`
	EndInclusive int `json:"endInclusive,omitempty"`
}

type ProjectCodeSecurityScanConfiguration struct {
	ContinuousIntegrationScanConfigurations []ProjectContinuousIntegrationScanConfiguration `json:"continuousIntegrationScanConfigurations,omitempty"`
	PeriodicScanConfigurations []ProjectPeriodicScanConfiguration `json:"periodicScanConfigurations,omitempty"`
}

type ProjectContinuousIntegrationScanConfiguration struct {
	RuleSetCategories []string `json:"ruleSetCategories,omitempty"`
	SupportedEvent *string `json:"supportedEvent,omitempty"`
}

type ProjectPeriodicScanConfiguration struct {
	FrequencyExpression *string `json:"frequencyExpression,omitempty"`
	RuleSetCategories []string `json:"ruleSetCategories,omitempty"`
}

type Recommendation struct {
	Url *string `json:"Url,omitempty"`
	Text *string `json:"text,omitempty"`
}

type Remediation struct {
	Recommendation *Recommendation `json:"recommendation,omitempty"`
}

type RepositoryAggregation struct {
	Repositories []StringFilter `json:"repositories,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
}

type RepositoryAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	AffectedImages int64 `json:"affectedImages,omitempty"`
	Repository string `json:"repository,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
}

type ResetEncryptionKeyRequest struct {
	ResourceType string `json:"resourceType,omitempty"`
	ScanType string `json:"scanType,omitempty"`
}

type ResetEncryptionKeyResponse struct {
}

type Resource struct {
	Details *ResourceDetails `json:"details,omitempty"`
	Id string `json:"id,omitempty"`
	Partition *string `json:"partition,omitempty"`
	Region *string `json:"region,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	Type string `json:"type,omitempty"`
}

type ResourceDetails struct {
	AwsEc2Instance *AwsEc2InstanceDetails `json:"awsEc2Instance,omitempty"`
	AwsEcrContainerImage *AwsEcrContainerImageDetails `json:"awsEcrContainerImage,omitempty"`
	AwsLambdaFunction *AwsLambdaFunctionDetails `json:"awsLambdaFunction,omitempty"`
	CodeRepository *CodeRepositoryDetails `json:"codeRepository,omitempty"`
}

type ResourceFilterCriteria struct {
	AccountId []ResourceStringFilter `json:"accountId,omitempty"`
	Ec2InstanceTags []ResourceMapFilter `json:"ec2InstanceTags,omitempty"`
	EcrImageTags []ResourceStringFilter `json:"ecrImageTags,omitempty"`
	EcrRepositoryName []ResourceStringFilter `json:"ecrRepositoryName,omitempty"`
	LambdaFunctionName []ResourceStringFilter `json:"lambdaFunctionName,omitempty"`
	LambdaFunctionTags []ResourceMapFilter `json:"lambdaFunctionTags,omitempty"`
	ResourceId []ResourceStringFilter `json:"resourceId,omitempty"`
	ResourceType []ResourceStringFilter `json:"resourceType,omitempty"`
}

type ResourceMapFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Key string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type ResourceScanMetadata struct {
	CodeRepository *CodeRepositoryMetadata `json:"codeRepository,omitempty"`
	Ec2 *Ec2Metadata `json:"ec2,omitempty"`
	EcrImage *EcrContainerImageMetadata `json:"ecrImage,omitempty"`
	EcrRepository *EcrRepositoryMetadata `json:"ecrRepository,omitempty"`
	LambdaFunction *LambdaFunctionMetadata `json:"lambdaFunction,omitempty"`
}

type ResourceState struct {
	CodeRepository *State `json:"codeRepository,omitempty"`
	Ec2 State `json:"ec2,omitempty"`
	Ecr State `json:"ecr,omitempty"`
	Lambda *State `json:"lambda,omitempty"`
	LambdaCode *State `json:"lambdaCode,omitempty"`
}

type ResourceStatus struct {
	CodeRepository *string `json:"codeRepository,omitempty"`
	Ec2 string `json:"ec2,omitempty"`
	Ecr string `json:"ecr,omitempty"`
	Lambda *string `json:"lambda,omitempty"`
	LambdaCode *string `json:"lambdaCode,omitempty"`
}

type ResourceStringFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type ScanStatus struct {
	Reason string `json:"reason,omitempty"`
	StatusCode string `json:"statusCode,omitempty"`
}

type Schedule struct {
	Daily *DailySchedule `json:"daily,omitempty"`
	Monthly *MonthlySchedule `json:"monthly,omitempty"`
	OneTime *OneTimeSchedule `json:"oneTime,omitempty"`
	Weekly *WeeklySchedule `json:"weekly,omitempty"`
}

type ScopeSettings struct {
	ProjectSelectionScope *string `json:"projectSelectionScope,omitempty"`
}

type SearchVulnerabilitiesFilterCriteria struct {
	VulnerabilityIds []string `json:"vulnerabilityIds,omitempty"`
}

type SearchVulnerabilitiesRequest struct {
	FilterCriteria SearchVulnerabilitiesFilterCriteria `json:"filterCriteria,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type SearchVulnerabilitiesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

type SendCisSessionHealthRequest struct {
	ScanJobId string `json:"scanJobId,omitempty"`
	SessionToken string `json:"sessionToken,omitempty"`
}

type SendCisSessionHealthResponse struct {
}

type SendCisSessionTelemetryRequest struct {
	Messages []CisSessionMessage `json:"messages,omitempty"`
	ScanJobId string `json:"scanJobId,omitempty"`
	SessionToken string `json:"sessionToken,omitempty"`
}

type SendCisSessionTelemetryResponse struct {
}

type SeverityCounts struct {
	All int64 `json:"all,omitempty"`
	Critical int64 `json:"critical,omitempty"`
	High int64 `json:"high,omitempty"`
	Medium int64 `json:"medium,omitempty"`
}

type SortCriteria struct {
	Field string `json:"field,omitempty"`
	SortOrder string `json:"sortOrder,omitempty"`
}

type StartCisSessionMessage struct {
	SessionToken string `json:"sessionToken,omitempty"`
}

type StartCisSessionRequest struct {
	Message StartCisSessionMessage `json:"message,omitempty"`
	ScanJobId string `json:"scanJobId,omitempty"`
}

type StartCisSessionResponse struct {
}

type StartCodeSecurityScanRequest struct {
	ClientToken *string `json:"clientToken,omitempty"`
	Resource CodeSecurityResource `json:"resource,omitempty"`
}

type StartCodeSecurityScanResponse struct {
	ScanId *string `json:"scanId,omitempty"`
	Status *string `json:"status,omitempty"`
}

type State struct {
	ErrorCode string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Status string `json:"status,omitempty"`
}

type StatusCounts struct {
	Failed int `json:"failed,omitempty"`
	Passed int `json:"passed,omitempty"`
	Skipped int `json:"skipped,omitempty"`
}

type Step struct {
	ComponentArn *string `json:"componentArn,omitempty"`
	ComponentId string `json:"componentId,omitempty"`
	ComponentType string `json:"componentType,omitempty"`
}

type StopCisMessageProgress struct {
	ErrorChecks int `json:"errorChecks,omitempty"`
	FailedChecks int `json:"failedChecks,omitempty"`
	InformationalChecks int `json:"informationalChecks,omitempty"`
	NotApplicableChecks int `json:"notApplicableChecks,omitempty"`
	NotEvaluatedChecks int `json:"notEvaluatedChecks,omitempty"`
	SuccessfulChecks int `json:"successfulChecks,omitempty"`
	TotalChecks int `json:"totalChecks,omitempty"`
	UnknownChecks int `json:"unknownChecks,omitempty"`
}

type StopCisSessionMessage struct {
	BenchmarkProfile *string `json:"benchmarkProfile,omitempty"`
	BenchmarkVersion *string `json:"benchmarkVersion,omitempty"`
	ComputePlatform *ComputePlatform `json:"computePlatform,omitempty"`
	Progress StopCisMessageProgress `json:"progress,omitempty"`
	Reason *string `json:"reason,omitempty"`
	Status string `json:"status,omitempty"`
}

type StopCisSessionRequest struct {
	Message StopCisSessionMessage `json:"message,omitempty"`
	ScanJobId string `json:"scanJobId,omitempty"`
	SessionToken string `json:"sessionToken,omitempty"`
}

type StopCisSessionResponse struct {
}

type StringFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Value string `json:"value,omitempty"`
}

type SuccessfulAssociationResult struct {
	Resource *CodeSecurityResource `json:"resource,omitempty"`
	ScanConfigurationArn *string `json:"scanConfigurationArn,omitempty"`
}

type SuggestedFix struct {
	Code *string `json:"code,omitempty"`
	Description *string `json:"description,omitempty"`
}

type TagFilter struct {
	Comparison string `json:"comparison,omitempty"`
	Key string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type TagResourceResponse struct {
}

type Time struct {
	TimeOfDay string `json:"timeOfDay,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type TitleAggregation struct {
	FindingType *string `json:"findingType,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	SortBy *string `json:"sortBy,omitempty"`
	SortOrder *string `json:"sortOrder,omitempty"`
	Titles []StringFilter `json:"titles,omitempty"`
	VulnerabilityIds []StringFilter `json:"vulnerabilityIds,omitempty"`
}

type TitleAggregationResponse struct {
	AccountId *string `json:"accountId,omitempty"`
	SeverityCounts *SeverityCounts `json:"severityCounts,omitempty"`
	Title string `json:"title,omitempty"`
	VulnerabilityId *string `json:"vulnerabilityId,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	TagKeys []string `json:"tagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateCisScanConfigurationRequest struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
	ScanName *string `json:"scanName,omitempty"`
	Schedule *Schedule `json:"schedule,omitempty"`
	SecurityLevel *string `json:"securityLevel,omitempty"`
	Targets *UpdateCisTargets `json:"targets,omitempty"`
}

type UpdateCisScanConfigurationResponse struct {
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type UpdateCisTargets struct {
	AccountIds []string `json:"accountIds,omitempty"`
	TargetResourceTags map[string][]string `json:"targetResourceTags,omitempty"`
}

type UpdateCodeSecurityIntegrationRequest struct {
	Details UpdateIntegrationDetails `json:"details,omitempty"`
	IntegrationArn string `json:"integrationArn,omitempty"`
}

type UpdateCodeSecurityIntegrationResponse struct {
	IntegrationArn string `json:"integrationArn,omitempty"`
	Status string `json:"status,omitempty"`
}

type UpdateCodeSecurityScanConfigurationRequest struct {
	Configuration CodeSecurityScanConfiguration `json:"configuration,omitempty"`
	ScanConfigurationArn string `json:"scanConfigurationArn,omitempty"`
}

type UpdateCodeSecurityScanConfigurationResponse struct {
	ScanConfigurationArn *string `json:"scanConfigurationArn,omitempty"`
}

type UpdateConfigurationRequest struct {
	Ec2Configuration *Ec2Configuration `json:"ec2Configuration,omitempty"`
	EcrConfiguration *EcrConfiguration `json:"ecrConfiguration,omitempty"`
}

type UpdateConfigurationResponse struct {
}

type UpdateEc2DeepInspectionConfigurationRequest struct {
	ActivateDeepInspection bool `json:"activateDeepInspection,omitempty"`
	PackagePaths []string `json:"packagePaths,omitempty"`
}

type UpdateEc2DeepInspectionConfigurationResponse struct {
	ErrorMessage *string `json:"errorMessage,omitempty"`
	OrgPackagePaths []string `json:"orgPackagePaths,omitempty"`
	PackagePaths []string `json:"packagePaths,omitempty"`
	Status *string `json:"status,omitempty"`
}

type UpdateEncryptionKeyRequest struct {
	KmsKeyId string `json:"kmsKeyId,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	ScanType string `json:"scanType,omitempty"`
}

type UpdateEncryptionKeyResponse struct {
}

type UpdateFilterRequest struct {
	Action *string `json:"action,omitempty"`
	Description *string `json:"description,omitempty"`
	FilterArn string `json:"filterArn,omitempty"`
	FilterCriteria *FilterCriteria `json:"filterCriteria,omitempty"`
	Name *string `json:"name,omitempty"`
	Reason *string `json:"reason,omitempty"`
}

type UpdateFilterResponse struct {
	Arn string `json:"arn,omitempty"`
}

type UpdateGitHubIntegrationDetail struct {
	Code string `json:"code,omitempty"`
	InstallationId string `json:"installationId,omitempty"`
}

type UpdateGitLabSelfManagedIntegrationDetail struct {
	AuthCode string `json:"authCode,omitempty"`
}

type UpdateIntegrationDetails struct {
	Github *UpdateGitHubIntegrationDetail `json:"github,omitempty"`
	GitlabSelfManaged *UpdateGitLabSelfManagedIntegrationDetail `json:"gitlabSelfManaged,omitempty"`
}

type UpdateOrgEc2DeepInspectionConfigurationRequest struct {
	OrgPackagePaths []string `json:"orgPackagePaths,omitempty"`
}

type UpdateOrgEc2DeepInspectionConfigurationResponse struct {
}

type UpdateOrganizationConfigurationRequest struct {
	AutoEnable AutoEnable `json:"autoEnable,omitempty"`
}

type UpdateOrganizationConfigurationResponse struct {
	AutoEnable AutoEnable `json:"autoEnable,omitempty"`
}

type Usage struct {
	Currency *string `json:"currency,omitempty"`
	EstimatedMonthlyCost float64 `json:"estimatedMonthlyCost,omitempty"`
	Total float64 `json:"total,omitempty"`
	Type *string `json:"type,omitempty"`
}

type UsageTotal struct {
	AccountId *string `json:"accountId,omitempty"`
	Usage []Usage `json:"usage,omitempty"`
}

type Vulnerability struct {
	AtigData *AtigData `json:"atigData,omitempty"`
	CisaData *CisaData `json:"cisaData,omitempty"`
	Cvss2 *Cvss2 `json:"cvss2,omitempty"`
	Cvss3 *Cvss3 `json:"cvss3,omitempty"`
	Cvss4 *Cvss4 `json:"cvss4,omitempty"`
	Cwes []string `json:"cwes,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectionPlatforms []string `json:"detectionPlatforms,omitempty"`
	Epss *Epss `json:"epss,omitempty"`
	ExploitObserved *ExploitObserved `json:"exploitObserved,omitempty"`
	Id string `json:"id,omitempty"`
	ReferenceUrls []string `json:"referenceUrls,omitempty"`
	RelatedVulnerabilities []string `json:"relatedVulnerabilities,omitempty"`
	Source *string `json:"source,omitempty"`
	SourceUrl *string `json:"sourceUrl,omitempty"`
	VendorCreatedAt *time.Time `json:"vendorCreatedAt,omitempty"`
	VendorSeverity *string `json:"vendorSeverity,omitempty"`
	VendorUpdatedAt *time.Time `json:"vendorUpdatedAt,omitempty"`
}

type VulnerablePackage struct {
	Arch *string `json:"arch,omitempty"`
	Epoch int `json:"epoch,omitempty"`
	FilePath *string `json:"filePath,omitempty"`
	FixedInVersion *string `json:"fixedInVersion,omitempty"`
	Name string `json:"name,omitempty"`
	PackageManager *string `json:"packageManager,omitempty"`
	Release *string `json:"release,omitempty"`
	Remediation *string `json:"remediation,omitempty"`
	SourceLambdaLayerArn *string `json:"sourceLambdaLayerArn,omitempty"`
	SourceLayerHash *string `json:"sourceLayerHash,omitempty"`
	Version string `json:"version,omitempty"`
}

type WeeklySchedule struct {
	Days []string `json:"days,omitempty"`
	StartTime Time `json:"startTime,omitempty"`
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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleAssociateMember(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AssociateMemberRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AssociateMember business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AssociateMember"})
}

func handleBatchAssociateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchAssociateCodeSecurityScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchAssociateCodeSecurityScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchAssociateCodeSecurityScanConfiguration"})
}

func handleBatchDisassociateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDisassociateCodeSecurityScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDisassociateCodeSecurityScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDisassociateCodeSecurityScanConfiguration"})
}

func handleBatchGetAccountStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetAccountStatusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetAccountStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetAccountStatus"})
}

func handleBatchGetCodeSnippet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetCodeSnippetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetCodeSnippet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetCodeSnippet"})
}

func handleBatchGetFindingDetails(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetFindingDetailsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetFindingDetails business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetFindingDetails"})
}

func handleBatchGetFreeTrialInfo(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetFreeTrialInfoRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetFreeTrialInfo business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetFreeTrialInfo"})
}

func handleBatchGetMemberEc2DeepInspectionStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetMemberEc2DeepInspectionStatusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetMemberEc2DeepInspectionStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetMemberEc2DeepInspectionStatus"})
}

func handleBatchUpdateMemberEc2DeepInspectionStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchUpdateMemberEc2DeepInspectionStatusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchUpdateMemberEc2DeepInspectionStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchUpdateMemberEc2DeepInspectionStatus"})
}

func handleCancelFindingsReport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CancelFindingsReportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CancelFindingsReport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CancelFindingsReport"})
}

func handleCancelSbomExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CancelSbomExportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CancelSbomExport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CancelSbomExport"})
}

func handleCreateCisScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCisScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCisScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCisScanConfiguration"})
}

func handleCreateCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCodeSecurityIntegrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCodeSecurityIntegration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCodeSecurityIntegration"})
}

func handleCreateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCodeSecurityScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCodeSecurityScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCodeSecurityScanConfiguration"})
}

func handleCreateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFilter"})
}

func handleCreateFindingsReport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFindingsReportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFindingsReport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFindingsReport"})
}

func handleCreateSbomExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateSbomExportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateSbomExport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateSbomExport"})
}

func handleDeleteCisScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCisScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCisScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCisScanConfiguration"})
}

func handleDeleteCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCodeSecurityIntegrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCodeSecurityIntegration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCodeSecurityIntegration"})
}

func handleDeleteCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCodeSecurityScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCodeSecurityScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCodeSecurityScanConfiguration"})
}

func handleDeleteFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFilter"})
}

func handleDescribeOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeOrganizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeOrganizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeOrganizationConfiguration"})
}

func handleDisable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement Disable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "Disable"})
}

func handleDisableDelegatedAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableDelegatedAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableDelegatedAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableDelegatedAdminAccount"})
}

func handleDisassociateMember(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateMemberRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateMember business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateMember"})
}

func handleEnable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement Enable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "Enable"})
}

func handleEnableDelegatedAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableDelegatedAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableDelegatedAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableDelegatedAdminAccount"})
}

func handleGetCisScanReport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCisScanReportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCisScanReport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCisScanReport"})
}

func handleGetCisScanResultDetails(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCisScanResultDetailsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCisScanResultDetails business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCisScanResultDetails"})
}

func handleGetClustersForImage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetClustersForImageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetClustersForImage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetClustersForImage"})
}

func handleGetCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCodeSecurityIntegrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCodeSecurityIntegration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCodeSecurityIntegration"})
}

func handleGetCodeSecurityScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCodeSecurityScanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCodeSecurityScan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCodeSecurityScan"})
}

func handleGetCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCodeSecurityScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCodeSecurityScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCodeSecurityScanConfiguration"})
}

func handleGetConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetConfiguration"})
}

func handleGetDelegatedAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetDelegatedAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetDelegatedAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetDelegatedAdminAccount"})
}

func handleGetEc2DeepInspectionConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetEc2DeepInspectionConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetEc2DeepInspectionConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetEc2DeepInspectionConfiguration"})
}

func handleGetEncryptionKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetEncryptionKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetEncryptionKey business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetEncryptionKey"})
}

func handleGetFindingsReportStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingsReportStatusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingsReportStatus business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingsReportStatus"})
}

func handleGetMember(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMemberRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMember business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMember"})
}

func handleGetSbomExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSbomExportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSbomExport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSbomExport"})
}

func handleListAccountPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAccountPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAccountPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAccountPermissions"})
}

func handleListCisScanConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCisScanConfigurationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCisScanConfigurations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCisScanConfigurations"})
}

func handleListCisScanResultsAggregatedByChecks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCisScanResultsAggregatedByChecksRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCisScanResultsAggregatedByChecks business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCisScanResultsAggregatedByChecks"})
}

func handleListCisScanResultsAggregatedByTargetResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCisScanResultsAggregatedByTargetResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCisScanResultsAggregatedByTargetResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCisScanResultsAggregatedByTargetResource"})
}

func handleListCisScans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCisScansRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCisScans business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCisScans"})
}

func handleListCodeSecurityIntegrations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCodeSecurityIntegrationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCodeSecurityIntegrations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCodeSecurityIntegrations"})
}

func handleListCodeSecurityScanConfigurationAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCodeSecurityScanConfigurationAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCodeSecurityScanConfigurationAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCodeSecurityScanConfigurationAssociations"})
}

func handleListCodeSecurityScanConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCodeSecurityScanConfigurationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCodeSecurityScanConfigurations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCodeSecurityScanConfigurations"})
}

func handleListCoverage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCoverageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCoverage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCoverage"})
}

func handleListCoverageStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCoverageStatisticsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCoverageStatistics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCoverageStatistics"})
}

func handleListDelegatedAdminAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDelegatedAdminAccountsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDelegatedAdminAccounts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDelegatedAdminAccounts"})
}

func handleListFilters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFiltersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFilters business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFilters"})
}

func handleListFindingAggregations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFindingAggregationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFindingAggregations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFindingAggregations"})
}

func handleListFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFindings"})
}

func handleListMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListMembers"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListUsageTotals(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListUsageTotalsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListUsageTotals business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListUsageTotals"})
}

func handleResetEncryptionKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ResetEncryptionKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ResetEncryptionKey business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ResetEncryptionKey"})
}

func handleSearchVulnerabilities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchVulnerabilitiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchVulnerabilities business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchVulnerabilities"})
}

func handleSendCisSessionHealth(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SendCisSessionHealthRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SendCisSessionHealth business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SendCisSessionHealth"})
}

func handleSendCisSessionTelemetry(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SendCisSessionTelemetryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SendCisSessionTelemetry business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SendCisSessionTelemetry"})
}

func handleStartCisSession(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartCisSessionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartCisSession business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartCisSession"})
}

func handleStartCodeSecurityScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartCodeSecurityScanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartCodeSecurityScan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartCodeSecurityScan"})
}

func handleStopCisSession(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopCisSessionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopCisSession business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopCisSession"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUpdateCisScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCisScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCisScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCisScanConfiguration"})
}

func handleUpdateCodeSecurityIntegration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCodeSecurityIntegrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCodeSecurityIntegration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCodeSecurityIntegration"})
}

func handleUpdateCodeSecurityScanConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCodeSecurityScanConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCodeSecurityScanConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCodeSecurityScanConfiguration"})
}

func handleUpdateConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateConfiguration"})
}

func handleUpdateEc2DeepInspectionConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateEc2DeepInspectionConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateEc2DeepInspectionConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateEc2DeepInspectionConfiguration"})
}

func handleUpdateEncryptionKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateEncryptionKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateEncryptionKey business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateEncryptionKey"})
}

func handleUpdateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFilter"})
}

func handleUpdateOrgEc2DeepInspectionConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateOrgEc2DeepInspectionConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateOrgEc2DeepInspectionConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateOrgEc2DeepInspectionConfiguration"})
}

func handleUpdateOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateOrganizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateOrganizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateOrganizationConfiguration"})
}

