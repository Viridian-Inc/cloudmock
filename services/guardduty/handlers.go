package guardduty

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AcceptAdministratorInvitationRequest struct {
	AdministratorId string `json:"administratorId,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	InvitationId string `json:"invitationId,omitempty"`
}

type AcceptAdministratorInvitationResponse struct {
}

type AcceptInvitationRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	InvitationId string `json:"invitationId,omitempty"`
	MasterId string `json:"masterId,omitempty"`
}

type AcceptInvitationResponse struct {
}

type AccessControlList struct {
	AllowsPublicReadAccess bool `json:"allowsPublicReadAccess,omitempty"`
	AllowsPublicWriteAccess bool `json:"allowsPublicWriteAccess,omitempty"`
}

type AccessKey struct {
	PrincipalId *string `json:"principalId,omitempty"`
	UserName *string `json:"userName,omitempty"`
	UserType *string `json:"userType,omitempty"`
}

type AccessKeyDetails struct {
	AccessKeyId *string `json:"accessKeyId,omitempty"`
	PrincipalId *string `json:"principalId,omitempty"`
	UserName *string `json:"userName,omitempty"`
	UserType *string `json:"userType,omitempty"`
}

type Account struct {
	Name *string `json:"account,omitempty"`
	Uid string `json:"uid,omitempty"`
}

type AccountDetail struct {
	AccountId string `json:"accountId,omitempty"`
	Email string `json:"email,omitempty"`
}

type AccountFreeTrialInfo struct {
	AccountId *string `json:"accountId,omitempty"`
	DataSources *DataSourcesFreeTrial `json:"dataSources,omitempty"`
	Features []FreeTrialFeatureConfigurationResult `json:"features,omitempty"`
}

type AccountLevelPermissions struct {
	BlockPublicAccess *BlockPublicAccess `json:"blockPublicAccess,omitempty"`
}

type AccountStatistics struct {
	AccountId *string `json:"accountId,omitempty"`
	LastGeneratedAt *time.Time `json:"lastGeneratedAt,omitempty"`
	TotalFindings int `json:"totalFindings,omitempty"`
}

type Action struct {
	ActionType *string `json:"actionType,omitempty"`
	AwsApiCallAction *AwsApiCallAction `json:"awsApiCallAction,omitempty"`
	DnsRequestAction *DnsRequestAction `json:"dnsRequestAction,omitempty"`
	KubernetesApiCallAction *KubernetesApiCallAction `json:"kubernetesApiCallAction,omitempty"`
	KubernetesPermissionCheckedDetails *KubernetesPermissionCheckedDetails `json:"kubernetesPermissionCheckedDetails,omitempty"`
	KubernetesRoleBindingDetails *KubernetesRoleBindingDetails `json:"kubernetesRoleBindingDetails,omitempty"`
	KubernetesRoleDetails *KubernetesRoleDetails `json:"kubernetesRoleDetails,omitempty"`
	NetworkConnectionAction *NetworkConnectionAction `json:"networkConnectionAction,omitempty"`
	PortProbeAction *PortProbeAction `json:"portProbeAction,omitempty"`
	RdsLoginAttemptAction *RdsLoginAttemptAction `json:"rdsLoginAttemptAction,omitempty"`
}

type Actor struct {
	Id string `json:"id,omitempty"`
	Process *ActorProcess `json:"process,omitempty"`
	Session *Session `json:"session,omitempty"`
	User *User `json:"user,omitempty"`
}

type ActorProcess struct {
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
	Sha256 *string `json:"sha256,omitempty"`
}

type AdditionalInfo struct {
	DeviceName *string `json:"deviceName,omitempty"`
	VersionId *string `json:"versionId,omitempty"`
}

type AddonDetails struct {
	AddonStatus *string `json:"addonStatus,omitempty"`
	AddonVersion *string `json:"addonVersion,omitempty"`
}

type AdminAccount struct {
	AdminAccountId *string `json:"adminAccountId,omitempty"`
	AdminStatus *string `json:"adminStatus,omitempty"`
}

type Administrator struct {
	AccountId *string `json:"accountId,omitempty"`
	InvitationId *string `json:"invitationId,omitempty"`
	InvitedAt *string `json:"invitedAt,omitempty"`
	RelationshipStatus *string `json:"relationshipStatus,omitempty"`
}

type AgentDetails struct {
	Version *string `json:"version,omitempty"`
}

type Anomaly struct {
	Profiles map[string]map[string][]AnomalyObject `json:"profiles,omitempty"`
	Unusual *AnomalyUnusual `json:"unusual,omitempty"`
}

type AnomalyObject struct {
	Observations *Observations `json:"observations,omitempty"`
	ProfileSubtype *string `json:"profileSubtype,omitempty"`
	ProfileType *string `json:"profileType,omitempty"`
}

type AnomalyUnusual struct {
	Behavior map[string]map[string]AnomalyObject `json:"behavior,omitempty"`
}

type ArchiveFindingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FindingIds []string `json:"findingIds,omitempty"`
}

type ArchiveFindingsResponse struct {
}

type AutonomousSystem struct {
	Name string `json:"name,omitempty"`
	Number int `json:"number,omitempty"`
}

type AutoscalingAutoScalingGroup struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
}

type AwsApiCallAction struct {
	AffectedResources map[string]string `json:"affectedResources,omitempty"`
	Api *string `json:"api,omitempty"`
	CallerType *string `json:"callerType,omitempty"`
	DomainDetails *DomainDetails `json:"domainDetails,omitempty"`
	ErrorCode *string `json:"errorCode,omitempty"`
	RemoteAccountDetails *RemoteAccountDetails `json:"remoteAccountDetails,omitempty"`
	RemoteIpDetails *RemoteIpDetails `json:"remoteIpDetails,omitempty"`
	ServiceName *string `json:"serviceName,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
}

type BlockPublicAccess struct {
	BlockPublicAcls bool `json:"blockPublicAcls,omitempty"`
	BlockPublicPolicy bool `json:"blockPublicPolicy,omitempty"`
	IgnorePublicAcls bool `json:"ignorePublicAcls,omitempty"`
	RestrictPublicBuckets bool `json:"restrictPublicBuckets,omitempty"`
}

type BucketLevelPermissions struct {
	AccessControlList *AccessControlList `json:"accessControlList,omitempty"`
	BlockPublicAccess *BlockPublicAccess `json:"blockPublicAccess,omitempty"`
	BucketPolicy *BucketPolicy `json:"bucketPolicy,omitempty"`
}

type BucketPolicy struct {
	AllowsPublicReadAccess bool `json:"allowsPublicReadAccess,omitempty"`
	AllowsPublicWriteAccess bool `json:"allowsPublicWriteAccess,omitempty"`
}

type City struct {
	CityName *string `json:"cityName,omitempty"`
}

type CloudTrailConfigurationResult struct {
	Status string `json:"status,omitempty"`
}

type CloudformationStack struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
}

type Condition struct {
	Eq []string `json:"eq,omitempty"`
	Equals []string `json:"equals,omitempty"`
	GreaterThan int64 `json:"greaterThan,omitempty"`
	GreaterThanOrEqual int64 `json:"greaterThanOrEqual,omitempty"`
	Gt int `json:"gt,omitempty"`
	Gte int `json:"gte,omitempty"`
	LessThan int64 `json:"lessThan,omitempty"`
	LessThanOrEqual int64 `json:"lessThanOrEqual,omitempty"`
	Lt int `json:"lt,omitempty"`
	Lte int `json:"lte,omitempty"`
	Matches []string `json:"matches,omitempty"`
	Neq []string `json:"neq,omitempty"`
	NotEquals []string `json:"notEquals,omitempty"`
	NotMatches []string `json:"notMatches,omitempty"`
}

type Container struct {
	ContainerRuntime *string `json:"containerRuntime,omitempty"`
	Id *string `json:"id,omitempty"`
	Image *string `json:"image,omitempty"`
	ImagePrefix *string `json:"imagePrefix,omitempty"`
	Name *string `json:"name,omitempty"`
	SecurityContext *SecurityContext `json:"securityContext,omitempty"`
	VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`
}

type ContainerFindingResource struct {
	Image string `json:"image,omitempty"`
	ImageUid *string `json:"imageUid,omitempty"`
}

type ContainerInstanceDetails struct {
	CompatibleContainerInstances int64 `json:"compatibleContainerInstances,omitempty"`
	CoveredContainerInstances int64 `json:"coveredContainerInstances,omitempty"`
}

type Country struct {
	CountryCode *string `json:"countryCode,omitempty"`
	CountryName *string `json:"countryName,omitempty"`
}

type CoverageEc2InstanceDetails struct {
	AgentDetails *AgentDetails `json:"agentDetails,omitempty"`
	ClusterArn *string `json:"clusterArn,omitempty"`
	InstanceId *string `json:"instanceId,omitempty"`
	InstanceType *string `json:"instanceType,omitempty"`
	ManagementType *string `json:"managementType,omitempty"`
}

type CoverageEcsClusterDetails struct {
	ClusterName *string `json:"clusterName,omitempty"`
	ContainerInstanceDetails *ContainerInstanceDetails `json:"containerInstanceDetails,omitempty"`
	FargateDetails *FargateDetails `json:"fargateDetails,omitempty"`
}

type CoverageEksClusterDetails struct {
	AddonDetails *AddonDetails `json:"addonDetails,omitempty"`
	ClusterName *string `json:"clusterName,omitempty"`
	CompatibleNodes int64 `json:"compatibleNodes,omitempty"`
	CoveredNodes int64 `json:"coveredNodes,omitempty"`
	ManagementType *string `json:"managementType,omitempty"`
}

type CoverageFilterCondition struct {
	Equals []string `json:"equals,omitempty"`
	NotEquals []string `json:"notEquals,omitempty"`
}

type CoverageFilterCriteria struct {
	FilterCriterion []CoverageFilterCriterion `json:"filterCriterion,omitempty"`
}

type CoverageFilterCriterion struct {
	CriterionKey *string `json:"criterionKey,omitempty"`
	FilterCondition *CoverageFilterCondition `json:"filterCondition,omitempty"`
}

type CoverageResource struct {
	AccountId *string `json:"accountId,omitempty"`
	CoverageStatus *string `json:"coverageStatus,omitempty"`
	DetectorId *string `json:"detectorId,omitempty"`
	Issue *string `json:"issue,omitempty"`
	ResourceDetails *CoverageResourceDetails `json:"resourceDetails,omitempty"`
	ResourceId *string `json:"resourceId,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type CoverageResourceDetails struct {
	Ec2InstanceDetails *CoverageEc2InstanceDetails `json:"ec2InstanceDetails,omitempty"`
	EcsClusterDetails *CoverageEcsClusterDetails `json:"ecsClusterDetails,omitempty"`
	EksClusterDetails *CoverageEksClusterDetails `json:"eksClusterDetails,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
}

type CoverageSortCriteria struct {
	AttributeName *string `json:"attributeName,omitempty"`
	OrderBy *string `json:"orderBy,omitempty"`
}

type CoverageStatistics struct {
	CountByCoverageStatus map[string]int64 `json:"countByCoverageStatus,omitempty"`
	CountByResourceType map[string]int64 `json:"countByResourceType,omitempty"`
}

type CreateDetectorRequest struct {
	ClientToken *string `json:"clientToken,omitempty"`
	DataSources *DataSourceConfigurations `json:"dataSources,omitempty"`
	Enable bool `json:"enable,omitempty"`
	Features []DetectorFeatureConfiguration `json:"features,omitempty"`
	FindingPublishingFrequency *string `json:"findingPublishingFrequency,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateDetectorResponse struct {
	DetectorId *string `json:"detectorId,omitempty"`
	UnprocessedDataSources *UnprocessedDataSourcesResult `json:"unprocessedDataSources,omitempty"`
}

type CreateFilterRequest struct {
	Action *string `json:"action,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	FindingCriteria FindingCriteria `json:"findingCriteria,omitempty"`
	Name string `json:"name,omitempty"`
	Rank int `json:"rank,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateFilterResponse struct {
	Name string `json:"name,omitempty"`
}

type CreateIPSetRequest struct {
	Activate bool `json:"activate,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateIPSetResponse struct {
	IpSetId string `json:"ipSetId,omitempty"`
}

type CreateMalwareProtectionPlanRequest struct {
	Actions *MalwareProtectionPlanActions `json:"actions,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	ProtectedResource CreateProtectedResource `json:"protectedResource,omitempty"`
	Role string `json:"role,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateMalwareProtectionPlanResponse struct {
	MalwareProtectionPlanId *string `json:"malwareProtectionPlanId,omitempty"`
}

type CreateMembersRequest struct {
	AccountDetails []AccountDetail `json:"accountDetails,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type CreateMembersResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type CreateProtectedResource struct {
	S3Bucket *CreateS3BucketResource `json:"s3Bucket,omitempty"`
}

type CreatePublishingDestinationRequest struct {
	ClientToken *string `json:"clientToken,omitempty"`
	DestinationProperties DestinationProperties `json:"destinationProperties,omitempty"`
	DestinationType string `json:"destinationType,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreatePublishingDestinationResponse struct {
	DestinationId string `json:"destinationId,omitempty"`
}

type CreateS3BucketResource struct {
	BucketName *string `json:"bucketName,omitempty"`
	ObjectPrefixes []string `json:"objectPrefixes,omitempty"`
}

type CreateSampleFindingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FindingTypes []string `json:"findingTypes,omitempty"`
}

type CreateSampleFindingsResponse struct {
}

type CreateThreatEntitySetRequest struct {
	Activate bool `json:"activate,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateThreatEntitySetResponse struct {
	ThreatEntitySetId string `json:"threatEntitySetId,omitempty"`
}

type CreateThreatIntelSetRequest struct {
	Activate bool `json:"activate,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateThreatIntelSetResponse struct {
	ThreatIntelSetId string `json:"threatIntelSetId,omitempty"`
}

type CreateTrustedEntitySetRequest struct {
	Activate bool `json:"activate,omitempty"`
	ClientToken *string `json:"clientToken,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type CreateTrustedEntitySetResponse struct {
	TrustedEntitySetId string `json:"trustedEntitySetId,omitempty"`
}

type DNSLogsConfigurationResult struct {
	Status string `json:"status,omitempty"`
}

type DataSourceConfigurations struct {
	Kubernetes *KubernetesConfiguration `json:"kubernetes,omitempty"`
	MalwareProtection *MalwareProtectionConfiguration `json:"malwareProtection,omitempty"`
	S3Logs *S3LogsConfiguration `json:"s3Logs,omitempty"`
}

type DataSourceConfigurationsResult struct {
	CloudTrail CloudTrailConfigurationResult `json:"cloudTrail,omitempty"`
	DNSLogs DNSLogsConfigurationResult `json:"dnsLogs,omitempty"`
	FlowLogs FlowLogsConfigurationResult `json:"flowLogs,omitempty"`
	Kubernetes *KubernetesConfigurationResult `json:"kubernetes,omitempty"`
	MalwareProtection *MalwareProtectionConfigurationResult `json:"malwareProtection,omitempty"`
	S3Logs S3LogsConfigurationResult `json:"s3Logs,omitempty"`
}

type DataSourceFreeTrial struct {
	FreeTrialDaysRemaining int `json:"freeTrialDaysRemaining,omitempty"`
}

type DataSourcesFreeTrial struct {
	CloudTrail *DataSourceFreeTrial `json:"cloudTrail,omitempty"`
	DnsLogs *DataSourceFreeTrial `json:"dnsLogs,omitempty"`
	FlowLogs *DataSourceFreeTrial `json:"flowLogs,omitempty"`
	Kubernetes *KubernetesDataSourceFreeTrial `json:"kubernetes,omitempty"`
	MalwareProtection *MalwareProtectionDataSourceFreeTrial `json:"malwareProtection,omitempty"`
	S3Logs *DataSourceFreeTrial `json:"s3Logs,omitempty"`
}

type DateStatistics struct {
	Date *time.Time `json:"date,omitempty"`
	LastGeneratedAt *time.Time `json:"lastGeneratedAt,omitempty"`
	Severity float64 `json:"severity,omitempty"`
	TotalFindings int `json:"totalFindings,omitempty"`
}

type DeclineInvitationsRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
}

type DeclineInvitationsResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type DefaultServerSideEncryption struct {
	EncryptionType *string `json:"encryptionType,omitempty"`
	KmsMasterKeyArn *string `json:"kmsMasterKeyArn,omitempty"`
}

type DeleteDetectorRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type DeleteDetectorResponse struct {
}

type DeleteFilterRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FilterName string `json:"filterName,omitempty"`
}

type DeleteFilterResponse struct {
}

type DeleteIPSetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	IpSetId string `json:"ipSetId,omitempty"`
}

type DeleteIPSetResponse struct {
}

type DeleteInvitationsRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
}

type DeleteInvitationsResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type DeleteMalwareProtectionPlanRequest struct {
	MalwareProtectionPlanId string `json:"malwareProtectionPlanId,omitempty"`
}

type DeleteMembersRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type DeleteMembersResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type DeletePublishingDestinationRequest struct {
	DestinationId string `json:"destinationId,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type DeletePublishingDestinationResponse struct {
}

type DeleteThreatEntitySetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	ThreatEntitySetId string `json:"threatEntitySetId,omitempty"`
}

type DeleteThreatEntitySetResponse struct {
}

type DeleteThreatIntelSetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	ThreatIntelSetId string `json:"threatIntelSetId,omitempty"`
}

type DeleteThreatIntelSetResponse struct {
}

type DeleteTrustedEntitySetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	TrustedEntitySetId string `json:"trustedEntitySetId,omitempty"`
}

type DeleteTrustedEntitySetResponse struct {
}

type DescribeMalwareScansRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FilterCriteria *FilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortCriteria *SortCriteria `json:"sortCriteria,omitempty"`
}

type DescribeMalwareScansResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Scans []Scan `json:"scans,omitempty"`
}

type DescribeOrganizationConfigurationRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type DescribeOrganizationConfigurationResponse struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
	AutoEnableOrganizationMembers *string `json:"autoEnableOrganizationMembers,omitempty"`
	DataSources *OrganizationDataSourceConfigurationsResult `json:"dataSources,omitempty"`
	Features []OrganizationFeatureConfigurationResult `json:"features,omitempty"`
	MemberAccountLimitReached bool `json:"memberAccountLimitReached,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type DescribePublishingDestinationRequest struct {
	DestinationId string `json:"destinationId,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type DescribePublishingDestinationResponse struct {
	DestinationId string `json:"destinationId,omitempty"`
	DestinationProperties DestinationProperties `json:"destinationProperties,omitempty"`
	DestinationType string `json:"destinationType,omitempty"`
	PublishingFailureStartTimestamp int64 `json:"publishingFailureStartTimestamp,omitempty"`
	Status string `json:"status,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type Destination struct {
	DestinationId string `json:"destinationId,omitempty"`
	DestinationType string `json:"destinationType,omitempty"`
	Status string `json:"status,omitempty"`
}

type DestinationProperties struct {
	DestinationArn *string `json:"destinationArn,omitempty"`
	KmsKeyArn *string `json:"kmsKeyArn,omitempty"`
}

type Detection struct {
	Anomaly *Anomaly `json:"anomaly,omitempty"`
	Sequence *Sequence `json:"sequence,omitempty"`
}

type DetectorAdditionalConfiguration struct {
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

type DetectorAdditionalConfigurationResult struct {
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type DetectorFeatureConfiguration struct {
	AdditionalConfiguration []DetectorAdditionalConfiguration `json:"additionalConfiguration,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

type DetectorFeatureConfigurationResult struct {
	AdditionalConfiguration []DetectorAdditionalConfigurationResult `json:"additionalConfiguration,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type DisableOrganizationAdminAccountRequest struct {
	AdminAccountId string `json:"adminAccountId,omitempty"`
}

type DisableOrganizationAdminAccountResponse struct {
}

type DisassociateFromAdministratorAccountRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type DisassociateFromAdministratorAccountResponse struct {
}

type DisassociateFromMasterAccountRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type DisassociateFromMasterAccountResponse struct {
}

type DisassociateMembersRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type DisassociateMembersResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type DnsRequestAction struct {
	Blocked bool `json:"blocked,omitempty"`
	Domain *string `json:"domain,omitempty"`
	DomainWithSuffix *string `json:"domainWithSuffix,omitempty"`
	Protocol *string `json:"protocol,omitempty"`
	VpcOwnerAccountId *string `json:"vpcOwnerAccountId,omitempty"`
}

type DomainDetails struct {
	Domain *string `json:"domain,omitempty"`
}

type EbsSnapshot struct {
	DeviceName *string `json:"deviceName,omitempty"`
}

type EbsSnapshotDetails struct {
	SnapshotArn *string `json:"snapshotArn,omitempty"`
}

type EbsVolumeDetails struct {
	ScannedVolumeDetails []VolumeDetail `json:"scannedVolumeDetails,omitempty"`
	SkippedVolumeDetails []VolumeDetail `json:"skippedVolumeDetails,omitempty"`
}

type EbsVolumeScanDetails struct {
	ScanCompletedAt *time.Time `json:"scanCompletedAt,omitempty"`
	ScanDetections *ScanDetections `json:"scanDetections,omitempty"`
	ScanId *string `json:"scanId,omitempty"`
	ScanStartedAt *time.Time `json:"scanStartedAt,omitempty"`
	ScanType *string `json:"scanType,omitempty"`
	Sources []string `json:"sources,omitempty"`
	TriggerFindingId *string `json:"triggerFindingId,omitempty"`
}

type EbsVolumesResult struct {
	Reason *string `json:"reason,omitempty"`
	Status *string `json:"status,omitempty"`
}

type Ec2Image struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
}

type Ec2ImageDetails struct {
	ImageArn *string `json:"imageArn,omitempty"`
}

type Ec2Instance struct {
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
	Ec2NetworkInterfaceUids []string `json:"ec2NetworkInterfaceUids,omitempty"`
	IamInstanceProfile *IamInstanceProfile `json:"IamInstanceProfile,omitempty"`
	ImageDescription *string `json:"imageDescription,omitempty"`
	InstanceState *string `json:"instanceState,omitempty"`
	InstanceType *string `json:"instanceType,omitempty"`
	OutpostArn *string `json:"outpostArn,omitempty"`
	Platform *string `json:"platform,omitempty"`
	ProductCodes []ProductCode `json:"productCodes,omitempty"`
}

type Ec2LaunchTemplate struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
	Version *string `json:"version,omitempty"`
}

type Ec2NetworkInterface struct {
	Ipv6Addresses []string `json:"ipv6Addresses,omitempty"`
	PrivateIpAddresses []PrivateIpAddressDetails `json:"privateIpAddresses,omitempty"`
	PublicIp *string `json:"publicIp,omitempty"`
	SecurityGroups []SecurityGroup `json:"securityGroups,omitempty"`
	SubNetId *string `json:"subNetId,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
}

type Ec2Vpc struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
}

type EcsCluster struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
	Status *string `json:"status,omitempty"`
}

type EcsClusterDetails struct {
	ActiveServicesCount int `json:"activeServicesCount,omitempty"`
	Arn *string `json:"arn,omitempty"`
	Name *string `json:"name,omitempty"`
	RegisteredContainerInstancesCount int `json:"registeredContainerInstancesCount,omitempty"`
	RunningTasksCount int `json:"runningTasksCount,omitempty"`
	Status *string `json:"status,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	TaskDetails *EcsTaskDetails `json:"taskDetails,omitempty"`
}

type EcsTask struct {
	ContainerUids []string `json:"containerUids,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	LaunchType *string `json:"launchType,omitempty"`
	TaskDefinitionArn *string `json:"taskDefinitionArn,omitempty"`
}

type EcsTaskDetails struct {
	Arn *string `json:"arn,omitempty"`
	Containers []Container `json:"containers,omitempty"`
	DefinitionArn *string `json:"definitionArn,omitempty"`
	Group *string `json:"group,omitempty"`
	LaunchType *string `json:"launchType,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	StartedBy *string `json:"startedBy,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	TaskCreatedAt *time.Time `json:"createdAt,omitempty"`
	Version *string `json:"version,omitempty"`
	Volumes []Volume `json:"volumes,omitempty"`
}

type EksCluster struct {
	Arn *string `json:"arn,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
	Status *string `json:"status,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
}

type EksClusterDetails struct {
	Arn *string `json:"arn,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
}

type EnableOrganizationAdminAccountRequest struct {
	AdminAccountId string `json:"adminAccountId,omitempty"`
}

type EnableOrganizationAdminAccountResponse struct {
}

type Evidence struct {
	ThreatIntelligenceDetails []ThreatIntelligenceDetail `json:"threatIntelligenceDetails,omitempty"`
}

type FargateDetails struct {
	Issues []string `json:"issues,omitempty"`
	ManagementType *string `json:"managementType,omitempty"`
}

type FilterCondition struct {
	EqualsValue *string `json:"equalsValue,omitempty"`
	GreaterThan int64 `json:"greaterThan,omitempty"`
	LessThan int64 `json:"lessThan,omitempty"`
}

type FilterCriteria struct {
	FilterCriterion []FilterCriterion `json:"filterCriterion,omitempty"`
}

type FilterCriterion struct {
	CriterionKey *string `json:"criterionKey,omitempty"`
	FilterCondition *FilterCondition `json:"filterCondition,omitempty"`
}

type Finding struct {
	AccountId string `json:"accountId,omitempty"`
	Arn string `json:"arn,omitempty"`
	AssociatedAttackSequenceArn *string `json:"associatedAttackSequenceArn,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	Description *string `json:"description,omitempty"`
	Id string `json:"id,omitempty"`
	Partition *string `json:"partition,omitempty"`
	Region string `json:"region,omitempty"`
	Resource Resource `json:"resource,omitempty"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
	ServiceModel *ServiceModel `json:"service,omitempty"`
	Severity float64 `json:"severity,omitempty"`
	Title *string `json:"title,omitempty"`
	Type string `json:"type,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type FindingCriteria struct {
	Criterion map[string]Condition `json:"criterion,omitempty"`
}

type FindingStatistics struct {
	CountBySeverity map[string]int `json:"countBySeverity,omitempty"`
	GroupedByAccount []AccountStatistics `json:"groupedByAccount,omitempty"`
	GroupedByDate []DateStatistics `json:"groupedByDate,omitempty"`
	GroupedByFindingType []FindingTypeStatistics `json:"groupedByFindingType,omitempty"`
	GroupedByResource []ResourceStatistics `json:"groupedByResource,omitempty"`
	GroupedBySeverity []SeverityStatistics `json:"groupedBySeverity,omitempty"`
}

type FindingTypeStatistics struct {
	FindingType *string `json:"findingType,omitempty"`
	LastGeneratedAt *time.Time `json:"lastGeneratedAt,omitempty"`
	TotalFindings int `json:"totalFindings,omitempty"`
}

type FlowLogsConfigurationResult struct {
	Status string `json:"status,omitempty"`
}

type FreeTrialFeatureConfigurationResult struct {
	FreeTrialDaysRemaining int `json:"freeTrialDaysRemaining,omitempty"`
	Name *string `json:"name,omitempty"`
}

type GeoLocation struct {
	Lat float64 `json:"lat,omitempty"`
	Lon float64 `json:"lon,omitempty"`
}

type GetAdministratorAccountRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type GetAdministratorAccountResponse struct {
	Administrator Administrator `json:"administrator,omitempty"`
}

type GetCoverageStatisticsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FilterCriteria *CoverageFilterCriteria `json:"filterCriteria,omitempty"`
	StatisticsType []string `json:"statisticsType,omitempty"`
}

type GetCoverageStatisticsResponse struct {
	CoverageStatistics *CoverageStatistics `json:"coverageStatistics,omitempty"`
}

type GetDetectorRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type GetDetectorResponse struct {
	CreatedAt *string `json:"createdAt,omitempty"`
	DataSources *DataSourceConfigurationsResult `json:"dataSources,omitempty"`
	Features []DetectorFeatureConfigurationResult `json:"features,omitempty"`
	FindingPublishingFrequency *string `json:"findingPublishingFrequency,omitempty"`
	ServiceRole string `json:"serviceRole,omitempty"`
	Status string `json:"status,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	UpdatedAt *string `json:"updatedAt,omitempty"`
}

type GetFilterRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FilterName string `json:"filterName,omitempty"`
}

type GetFilterResponse struct {
	Action string `json:"action,omitempty"`
	Description *string `json:"description,omitempty"`
	FindingCriteria FindingCriteria `json:"findingCriteria,omitempty"`
	Name string `json:"name,omitempty"`
	Rank int `json:"rank,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type GetFindingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FindingIds []string `json:"findingIds,omitempty"`
	SortCriteria *SortCriteria `json:"sortCriteria,omitempty"`
}

type GetFindingsResponse struct {
	Findings []Finding `json:"findings,omitempty"`
}

type GetFindingsStatisticsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FindingCriteria *FindingCriteria `json:"findingCriteria,omitempty"`
	FindingStatisticTypes []string `json:"findingStatisticTypes,omitempty"`
	GroupBy *string `json:"groupBy,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	OrderBy *string `json:"orderBy,omitempty"`
}

type GetFindingsStatisticsResponse struct {
	FindingStatistics FindingStatistics `json:"findingStatistics,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type GetIPSetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	IpSetId string `json:"ipSetId,omitempty"`
}

type GetIPSetResponse struct {
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type GetInvitationsCountRequest struct {
}

type GetInvitationsCountResponse struct {
	InvitationsCount int `json:"invitationsCount,omitempty"`
}

type GetMalwareProtectionPlanRequest struct {
	MalwareProtectionPlanId string `json:"malwareProtectionPlanId,omitempty"`
}

type GetMalwareProtectionPlanResponse struct {
	Actions *MalwareProtectionPlanActions `json:"actions,omitempty"`
	Arn *string `json:"arn,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	ProtectedResource *CreateProtectedResource `json:"protectedResource,omitempty"`
	Role *string `json:"role,omitempty"`
	Status *string `json:"status,omitempty"`
	StatusReasons []MalwareProtectionPlanStatusReason `json:"statusReasons,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type GetMalwareScanRequest struct {
	ScanId string `json:"scanId,omitempty"`
}

type GetMalwareScanResponse struct {
	AdminDetectorId *string `json:"adminDetectorId,omitempty"`
	DetectorId *string `json:"detectorId,omitempty"`
	FailedResourcesCount int `json:"failedResourcesCount,omitempty"`
	ResourceArn *string `json:"resourceArn,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	ScanCategory *string `json:"scanCategory,omitempty"`
	ScanCompletedAt *time.Time `json:"scanCompletedAt,omitempty"`
	ScanConfiguration *ScanConfiguration `json:"scanConfiguration,omitempty"`
	ScanId *string `json:"scanId,omitempty"`
	ScanResultDetails *GetMalwareScanResultDetails `json:"scanResultDetails,omitempty"`
	ScanStartedAt *time.Time `json:"scanStartedAt,omitempty"`
	ScanStatus *string `json:"scanStatus,omitempty"`
	ScanStatusReason *string `json:"scanStatusReason,omitempty"`
	ScanType *string `json:"scanType,omitempty"`
	ScannedResources []ScannedResource `json:"scannedResources,omitempty"`
	ScannedResourcesCount int `json:"scannedResourcesCount,omitempty"`
	SkippedResourcesCount int `json:"skippedResourcesCount,omitempty"`
}

type GetMalwareScanResultDetails struct {
	FailedFileCount int64 `json:"failedFileCount,omitempty"`
	ScanResultStatus *string `json:"scanResultStatus,omitempty"`
	SkippedFileCount int64 `json:"skippedFileCount,omitempty"`
	ThreatFoundFileCount int64 `json:"threatFoundFileCount,omitempty"`
	Threats []ScanResultThreat `json:"threats,omitempty"`
	TotalBytes int64 `json:"totalBytes,omitempty"`
	TotalFileCount int64 `json:"totalFileCount,omitempty"`
	UniqueThreatCount int64 `json:"uniqueThreatCount,omitempty"`
}

type GetMalwareScanSettingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type GetMalwareScanSettingsResponse struct {
	EbsSnapshotPreservation *string `json:"ebsSnapshotPreservation,omitempty"`
	ScanResourceCriteria *ScanResourceCriteria `json:"scanResourceCriteria,omitempty"`
}

type GetMasterAccountRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
}

type GetMasterAccountResponse struct {
	Master Master `json:"master,omitempty"`
}

type GetMemberDetectorsRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type GetMemberDetectorsResponse struct {
	MemberDataSourceConfigurations []MemberDataSourceConfiguration `json:"members,omitempty"`
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type GetMembersRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type GetMembersResponse struct {
	Members []Member `json:"members,omitempty"`
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type GetOrganizationStatisticsResponse struct {
	OrganizationDetails *OrganizationDetails `json:"organizationDetails,omitempty"`
}

type GetRemainingFreeTrialDaysRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type GetRemainingFreeTrialDaysResponse struct {
	Accounts []AccountFreeTrialInfo `json:"accounts,omitempty"`
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type GetThreatEntitySetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	ThreatEntitySetId string `json:"threatEntitySetId,omitempty"`
}

type GetThreatEntitySetResponse struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	ErrorDetails *string `json:"errorDetails,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type GetThreatIntelSetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	ThreatIntelSetId string `json:"threatIntelSetId,omitempty"`
}

type GetThreatIntelSetResponse struct {
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type GetTrustedEntitySetRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	TrustedEntitySetId string `json:"trustedEntitySetId,omitempty"`
}

type GetTrustedEntitySetResponse struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	ErrorDetails *string `json:"errorDetails,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Format string `json:"format,omitempty"`
	Location string `json:"location,omitempty"`
	Name string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type GetUsageStatisticsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	Unit *string `json:"unit,omitempty"`
	UsageCriteria UsageCriteria `json:"usageCriteria,omitempty"`
	UsageStatisticType string `json:"usageStatisticsType,omitempty"`
}

type GetUsageStatisticsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	UsageStatistics *UsageStatistics `json:"usageStatistics,omitempty"`
}

type HighestSeverityThreatDetails struct {
	Count int `json:"count,omitempty"`
	Severity *string `json:"severity,omitempty"`
	ThreatName *string `json:"threatName,omitempty"`
}

type HostPath struct {
	Path *string `json:"path,omitempty"`
}

type IamInstanceProfile struct {
	Arn *string `json:"arn,omitempty"`
	Id *string `json:"id,omitempty"`
}

type IamInstanceProfileV2 struct {
	Ec2InstanceUids []string `json:"ec2InstanceUids,omitempty"`
}

type ImpersonatedUser struct {
	Groups []string `json:"groups,omitempty"`
	Username *string `json:"username,omitempty"`
}

type IncrementalScanDetails struct {
	BaselineResourceArn string `json:"baselineResourceArn,omitempty"`
}

type Indicator struct {
	Key string `json:"key,omitempty"`
	Title *string `json:"title,omitempty"`
	Values []string `json:"values,omitempty"`
}

type InstanceDetails struct {
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
	IamInstanceProfile *IamInstanceProfile `json:"iamInstanceProfile,omitempty"`
	ImageDescription *string `json:"imageDescription,omitempty"`
	ImageId *string `json:"imageId,omitempty"`
	InstanceId *string `json:"instanceId,omitempty"`
	InstanceState *string `json:"instanceState,omitempty"`
	InstanceType *string `json:"instanceType,omitempty"`
	LaunchTime *string `json:"launchTime,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces,omitempty"`
	OutpostArn *string `json:"outpostArn,omitempty"`
	Platform *string `json:"platform,omitempty"`
	ProductCodes []ProductCode `json:"productCodes,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type Invitation struct {
	AccountId *string `json:"accountId,omitempty"`
	InvitationId *string `json:"invitationId,omitempty"`
	InvitedAt *string `json:"invitedAt,omitempty"`
	RelationshipStatus *string `json:"relationshipStatus,omitempty"`
}

type InviteMembersRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	DisableEmailNotification bool `json:"disableEmailNotification,omitempty"`
	Message *string `json:"message,omitempty"`
}

type InviteMembersResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type ItemDetails struct {
	AdditionalInfo *AdditionalInfo `json:"additionalInfo,omitempty"`
	Hash *string `json:"hash,omitempty"`
	ItemPath *string `json:"itemPath,omitempty"`
	ResourceArn *string `json:"resourceArn,omitempty"`
}

type ItemPath struct {
	Hash *string `json:"hash,omitempty"`
	NestedItemPath *string `json:"nestedItemPath,omitempty"`
}

type KubernetesApiCallAction struct {
	Namespace *string `json:"namespace,omitempty"`
	Parameters *string `json:"parameters,omitempty"`
	RemoteIpDetails *RemoteIpDetails `json:"remoteIpDetails,omitempty"`
	RequestUri *string `json:"requestUri,omitempty"`
	Resource *string `json:"resource,omitempty"`
	ResourceName *string `json:"resourceName,omitempty"`
	SourceIps []string `json:"sourceIPs,omitempty"`
	StatusCode int `json:"statusCode,omitempty"`
	Subresource *string `json:"subresource,omitempty"`
	UserAgent *string `json:"userAgent,omitempty"`
	Verb *string `json:"verb,omitempty"`
}

type KubernetesAuditLogsConfiguration struct {
	Enable bool `json:"enable,omitempty"`
}

type KubernetesAuditLogsConfigurationResult struct {
	Status string `json:"status,omitempty"`
}

type KubernetesConfiguration struct {
	AuditLogs KubernetesAuditLogsConfiguration `json:"auditLogs,omitempty"`
}

type KubernetesConfigurationResult struct {
	AuditLogs KubernetesAuditLogsConfigurationResult `json:"auditLogs,omitempty"`
}

type KubernetesDataSourceFreeTrial struct {
	AuditLogs *DataSourceFreeTrial `json:"auditLogs,omitempty"`
}

type KubernetesDetails struct {
	KubernetesUserDetails *KubernetesUserDetails `json:"kubernetesUserDetails,omitempty"`
	KubernetesWorkloadDetails *KubernetesWorkloadDetails `json:"kubernetesWorkloadDetails,omitempty"`
}

type KubernetesPermissionCheckedDetails struct {
	Allowed bool `json:"allowed,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Resource *string `json:"resource,omitempty"`
	Verb *string `json:"verb,omitempty"`
}

type KubernetesRoleBindingDetails struct {
	Kind *string `json:"kind,omitempty"`
	Name *string `json:"name,omitempty"`
	RoleRefKind *string `json:"roleRefKind,omitempty"`
	RoleRefName *string `json:"roleRefName,omitempty"`
	Uid *string `json:"uid,omitempty"`
}

type KubernetesRoleDetails struct {
	Kind *string `json:"kind,omitempty"`
	Name *string `json:"name,omitempty"`
	Uid *string `json:"uid,omitempty"`
}

type KubernetesUserDetails struct {
	Groups []string `json:"groups,omitempty"`
	ImpersonatedUser *ImpersonatedUser `json:"impersonatedUser,omitempty"`
	SessionName []string `json:"sessionName,omitempty"`
	Uid *string `json:"uid,omitempty"`
	Username *string `json:"username,omitempty"`
}

type KubernetesWorkload struct {
	ContainerUids []string `json:"containerUids,omitempty"`
	KubernetesResourcesTypes *string `json:"type,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type KubernetesWorkloadDetails struct {
	Containers []Container `json:"containers,omitempty"`
	HostIPC bool `json:"hostIPC,omitempty"`
	HostNetwork bool `json:"hostNetwork,omitempty"`
	HostPID bool `json:"hostPID,omitempty"`
	Name *string `json:"name,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`
	Type *string `json:"type,omitempty"`
	Uid *string `json:"uid,omitempty"`
	Volumes []Volume `json:"volumes,omitempty"`
}

type LambdaDetails struct {
	Description *string `json:"description,omitempty"`
	FunctionArn *string `json:"functionArn,omitempty"`
	FunctionName *string `json:"functionName,omitempty"`
	FunctionVersion *string `json:"functionVersion,omitempty"`
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
	RevisionId *string `json:"revisionId,omitempty"`
	Role *string `json:"role,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	VpcConfig *VpcConfig `json:"vpcConfig,omitempty"`
}

type LineageObject struct {
	Euid int `json:"euid,omitempty"`
	ExecutablePath *string `json:"executablePath,omitempty"`
	Name *string `json:"name,omitempty"`
	NamespacePid int `json:"namespacePid,omitempty"`
	ParentUuid *string `json:"parentUuid,omitempty"`
	Pid int `json:"pid,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	UserId int `json:"userId,omitempty"`
	Uuid *string `json:"uuid,omitempty"`
}

type ListCoverageRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FilterCriteria *CoverageFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortCriteria *CoverageSortCriteria `json:"sortCriteria,omitempty"`
}

type ListCoverageResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Resources []CoverageResource `json:"resources,omitempty"`
}

type ListDetectorsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListDetectorsResponse struct {
	DetectorIds []string `json:"detectorIds,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFiltersRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFiltersResponse struct {
	FilterNames []string `json:"filterNames,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListFindingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FindingCriteria *FindingCriteria `json:"findingCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortCriteria *SortCriteria `json:"sortCriteria,omitempty"`
}

type ListFindingsResponse struct {
	FindingIds []string `json:"findingIds,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListIPSetsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListIPSetsResponse struct {
	IpSetIds []string `json:"ipSetIds,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListInvitationsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListInvitationsResponse struct {
	Invitations []Invitation `json:"invitations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListMalwareProtectionPlansRequest struct {
	NextToken *string `json:"nextToken,omitempty"`
}

type ListMalwareProtectionPlansResponse struct {
	MalwareProtectionPlans []MalwareProtectionPlanSummary `json:"malwareProtectionPlans,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListMalwareScansFilterCriteria struct {
	ListMalwareScansFilterCriterion []ListMalwareScansFilterCriterion `json:"filterCriterion,omitempty"`
}

type ListMalwareScansFilterCriterion struct {
	FilterCondition *FilterCondition `json:"filterCondition,omitempty"`
	ListMalwareScansCriterionKey *string `json:"criterionKey,omitempty"`
}

type ListMalwareScansRequest struct {
	FilterCriteria *ListMalwareScansFilterCriteria `json:"filterCriteria,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	SortCriteria *SortCriteria `json:"sortCriteria,omitempty"`
}

type ListMalwareScansResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Scans []MalwareScan `json:"scans,omitempty"`
}

type ListMembersRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	OnlyAssociated *string `json:"onlyAssociated,omitempty"`
}

type ListMembersResponse struct {
	Members []Member `json:"members,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListOrganizationAdminAccountsRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListOrganizationAdminAccountsResponse struct {
	AdminAccounts []AdminAccount `json:"adminAccounts,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListPublishingDestinationsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListPublishingDestinationsResponse struct {
	Destinations []Destination `json:"destinations,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags map[string]string `json:"tags,omitempty"`
}

type ListThreatEntitySetsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListThreatEntitySetsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	ThreatEntitySetIds []string `json:"threatEntitySetIds,omitempty"`
}

type ListThreatIntelSetsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListThreatIntelSetsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	ThreatIntelSetIds []string `json:"threatIntelSetIds,omitempty"`
}

type ListTrustedEntitySetsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListTrustedEntitySetsResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	TrustedEntitySetIds []string `json:"trustedEntitySetIds,omitempty"`
}

type LocalIpDetails struct {
	IpAddressV4 *string `json:"ipAddressV4,omitempty"`
	IpAddressV6 *string `json:"ipAddressV6,omitempty"`
}

type LocalPortDetails struct {
	Port int `json:"port,omitempty"`
	PortName *string `json:"portName,omitempty"`
}

type LoginAttribute struct {
	Application *string `json:"application,omitempty"`
	FailedLoginAttempts int `json:"failedLoginAttempts,omitempty"`
	SuccessfulLoginAttempts int `json:"successfulLoginAttempts,omitempty"`
	User *string `json:"user,omitempty"`
}

type MalwareProtectionConfiguration struct {
	ScanEc2InstanceWithFindings *ScanEc2InstanceWithFindings `json:"scanEc2InstanceWithFindings,omitempty"`
}

type MalwareProtectionConfigurationResult struct {
	ScanEc2InstanceWithFindings *ScanEc2InstanceWithFindingsResult `json:"scanEc2InstanceWithFindings,omitempty"`
	ServiceRole *string `json:"serviceRole,omitempty"`
}

type MalwareProtectionDataSourceFreeTrial struct {
	ScanEc2InstanceWithFindings *DataSourceFreeTrial `json:"scanEc2InstanceWithFindings,omitempty"`
}

type MalwareProtectionFindingsScanConfiguration struct {
	IncrementalScanDetails *IncrementalScanDetails `json:"incrementalScanDetails,omitempty"`
	TriggerType *string `json:"triggerType,omitempty"`
}

type MalwareProtectionPlanActions struct {
	Tagging *MalwareProtectionPlanTaggingAction `json:"tagging,omitempty"`
}

type MalwareProtectionPlanStatusReason struct {
	Code *string `json:"code,omitempty"`
	Message *string `json:"message,omitempty"`
}

type MalwareProtectionPlanSummary struct {
	MalwareProtectionPlanId *string `json:"malwareProtectionPlanId,omitempty"`
}

type MalwareProtectionPlanTaggingAction struct {
	Status *string `json:"status,omitempty"`
}

type MalwareScan struct {
	ResourceArn *string `json:"resourceArn,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	ScanCompletedAt *time.Time `json:"scanCompletedAt,omitempty"`
	ScanId *string `json:"scanId,omitempty"`
	ScanResultStatus *string `json:"scanResultStatus,omitempty"`
	ScanStartedAt *time.Time `json:"scanStartedAt,omitempty"`
	ScanStatus *string `json:"scanStatus,omitempty"`
	ScanType *string `json:"scanType,omitempty"`
}

type MalwareScanDetails struct {
	ScanCategory *string `json:"scanCategory,omitempty"`
	ScanConfiguration *MalwareProtectionFindingsScanConfiguration `json:"scanConfiguration,omitempty"`
	ScanId *string `json:"scanId,omitempty"`
	ScanType *string `json:"scanType,omitempty"`
	Threats []Threat `json:"threats,omitempty"`
	UniqueThreatCount int `json:"uniqueThreatCount,omitempty"`
}

type Master struct {
	AccountId *string `json:"accountId,omitempty"`
	InvitationId *string `json:"invitationId,omitempty"`
	InvitedAt *string `json:"invitedAt,omitempty"`
	RelationshipStatus *string `json:"relationshipStatus,omitempty"`
}

type Member struct {
	AccountId string `json:"accountId,omitempty"`
	AdministratorId *string `json:"administratorId,omitempty"`
	DetectorId *string `json:"detectorId,omitempty"`
	Email string `json:"email,omitempty"`
	InvitedAt *string `json:"invitedAt,omitempty"`
	MasterId string `json:"masterId,omitempty"`
	RelationshipStatus string `json:"relationshipStatus,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type MemberAdditionalConfiguration struct {
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

type MemberAdditionalConfigurationResult struct {
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type MemberDataSourceConfiguration struct {
	AccountId string `json:"accountId,omitempty"`
	DataSources *DataSourceConfigurationsResult `json:"dataSources,omitempty"`
	Features []MemberFeaturesConfigurationResult `json:"features,omitempty"`
}

type MemberFeaturesConfiguration struct {
	AdditionalConfiguration []MemberAdditionalConfiguration `json:"additionalConfiguration,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

type MemberFeaturesConfigurationResult struct {
	AdditionalConfiguration []MemberAdditionalConfigurationResult `json:"additionalConfiguration,omitempty"`
	Name *string `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type NetworkConnection struct {
	Direction string `json:"direction,omitempty"`
}

type NetworkConnectionAction struct {
	Blocked bool `json:"blocked,omitempty"`
	ConnectionDirection *string `json:"connectionDirection,omitempty"`
	LocalIpDetails *LocalIpDetails `json:"localIpDetails,omitempty"`
	LocalNetworkInterface *string `json:"localNetworkInterface,omitempty"`
	LocalPortDetails *LocalPortDetails `json:"localPortDetails,omitempty"`
	Protocol *string `json:"protocol,omitempty"`
	RemoteIpDetails *RemoteIpDetails `json:"remoteIpDetails,omitempty"`
	RemotePortDetails *RemotePortDetails `json:"remotePortDetails,omitempty"`
}

type NetworkEndpoint struct {
	AutonomousSystem *AutonomousSystem `json:"autonomousSystem,omitempty"`
	Connection *NetworkConnection `json:"connection,omitempty"`
	Domain *string `json:"domain,omitempty"`
	Id string `json:"id,omitempty"`
	Ip *string `json:"ip,omitempty"`
	Location *NetworkGeoLocation `json:"location,omitempty"`
	Port int `json:"port,omitempty"`
}

type NetworkGeoLocation struct {
	City string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	Latitude float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lon,omitempty"`
}

type NetworkInterface struct {
	Ipv6Addresses []string `json:"ipv6Addresses,omitempty"`
	NetworkInterfaceId *string `json:"networkInterfaceId,omitempty"`
	PrivateDnsName *string `json:"privateDnsName,omitempty"`
	PrivateIpAddress *string `json:"privateIpAddress,omitempty"`
	PrivateIpAddresses []PrivateIpAddressDetails `json:"privateIpAddresses,omitempty"`
	PublicDnsName *string `json:"publicDnsName,omitempty"`
	PublicIp *string `json:"publicIp,omitempty"`
	SecurityGroups []SecurityGroup `json:"securityGroups,omitempty"`
	SubnetId *string `json:"subnetId,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
}

type Observations struct {
	Text []string `json:"text,omitempty"`
}

type Organization struct {
	Asn *string `json:"asn,omitempty"`
	AsnOrg *string `json:"asnOrg,omitempty"`
	Isp *string `json:"isp,omitempty"`
	Org *string `json:"org,omitempty"`
}

type OrganizationAdditionalConfiguration struct {
	AutoEnable *string `json:"autoEnable,omitempty"`
	Name *string `json:"name,omitempty"`
}

type OrganizationAdditionalConfigurationResult struct {
	AutoEnable *string `json:"autoEnable,omitempty"`
	Name *string `json:"name,omitempty"`
}

type OrganizationDataSourceConfigurations struct {
	Kubernetes *OrganizationKubernetesConfiguration `json:"kubernetes,omitempty"`
	MalwareProtection *OrganizationMalwareProtectionConfiguration `json:"malwareProtection,omitempty"`
	S3Logs *OrganizationS3LogsConfiguration `json:"s3Logs,omitempty"`
}

type OrganizationDataSourceConfigurationsResult struct {
	Kubernetes *OrganizationKubernetesConfigurationResult `json:"kubernetes,omitempty"`
	MalwareProtection *OrganizationMalwareProtectionConfigurationResult `json:"malwareProtection,omitempty"`
	S3Logs OrganizationS3LogsConfigurationResult `json:"s3Logs,omitempty"`
}

type OrganizationDetails struct {
	OrganizationStatistics *OrganizationStatistics `json:"organizationStatistics,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type OrganizationEbsVolumes struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
}

type OrganizationEbsVolumesResult struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
}

type OrganizationFeatureConfiguration struct {
	AdditionalConfiguration []OrganizationAdditionalConfiguration `json:"additionalConfiguration,omitempty"`
	AutoEnable *string `json:"autoEnable,omitempty"`
	Name *string `json:"name,omitempty"`
}

type OrganizationFeatureConfigurationResult struct {
	AdditionalConfiguration []OrganizationAdditionalConfigurationResult `json:"additionalConfiguration,omitempty"`
	AutoEnable *string `json:"autoEnable,omitempty"`
	Name *string `json:"name,omitempty"`
}

type OrganizationFeatureStatistics struct {
	AdditionalConfiguration []OrganizationFeatureStatisticsAdditionalConfiguration `json:"additionalConfiguration,omitempty"`
	EnabledAccountsCount int `json:"enabledAccountsCount,omitempty"`
	Name *string `json:"name,omitempty"`
}

type OrganizationFeatureStatisticsAdditionalConfiguration struct {
	EnabledAccountsCount int `json:"enabledAccountsCount,omitempty"`
	Name *string `json:"name,omitempty"`
}

type OrganizationKubernetesAuditLogsConfiguration struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
}

type OrganizationKubernetesAuditLogsConfigurationResult struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
}

type OrganizationKubernetesConfiguration struct {
	AuditLogs OrganizationKubernetesAuditLogsConfiguration `json:"auditLogs,omitempty"`
}

type OrganizationKubernetesConfigurationResult struct {
	AuditLogs OrganizationKubernetesAuditLogsConfigurationResult `json:"auditLogs,omitempty"`
}

type OrganizationMalwareProtectionConfiguration struct {
	ScanEc2InstanceWithFindings *OrganizationScanEc2InstanceWithFindings `json:"scanEc2InstanceWithFindings,omitempty"`
}

type OrganizationMalwareProtectionConfigurationResult struct {
	ScanEc2InstanceWithFindings *OrganizationScanEc2InstanceWithFindingsResult `json:"scanEc2InstanceWithFindings,omitempty"`
}

type OrganizationS3LogsConfiguration struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
}

type OrganizationS3LogsConfigurationResult struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
}

type OrganizationScanEc2InstanceWithFindings struct {
	EbsVolumes *OrganizationEbsVolumes `json:"ebsVolumes,omitempty"`
}

type OrganizationScanEc2InstanceWithFindingsResult struct {
	EbsVolumes *OrganizationEbsVolumesResult `json:"ebsVolumes,omitempty"`
}

type OrganizationStatistics struct {
	ActiveAccountsCount int `json:"activeAccountsCount,omitempty"`
	CountByFeature []OrganizationFeatureStatistics `json:"countByFeature,omitempty"`
	EnabledAccountsCount int `json:"enabledAccountsCount,omitempty"`
	MemberAccountsCount int `json:"memberAccountsCount,omitempty"`
	TotalAccountsCount int `json:"totalAccountsCount,omitempty"`
}

type Owner struct {
	Id *string `json:"id,omitempty"`
}

type PermissionConfiguration struct {
	AccountLevelPermissions *AccountLevelPermissions `json:"accountLevelPermissions,omitempty"`
	BucketLevelPermissions *BucketLevelPermissions `json:"bucketLevelPermissions,omitempty"`
}

type PortProbeAction struct {
	Blocked bool `json:"blocked,omitempty"`
	PortProbeDetails []PortProbeDetail `json:"portProbeDetails,omitempty"`
}

type PortProbeDetail struct {
	LocalIpDetails *LocalIpDetails `json:"localIpDetails,omitempty"`
	LocalPortDetails *LocalPortDetails `json:"localPortDetails,omitempty"`
	RemoteIpDetails *RemoteIpDetails `json:"remoteIpDetails,omitempty"`
}

type PrivateIpAddressDetails struct {
	PrivateDnsName *string `json:"privateDnsName,omitempty"`
	PrivateIpAddress *string `json:"privateIpAddress,omitempty"`
}

type ProcessDetails struct {
	Euid int `json:"euid,omitempty"`
	ExecutablePath *string `json:"executablePath,omitempty"`
	ExecutableSha256 *string `json:"executableSha256,omitempty"`
	Lineage []LineageObject `json:"lineage,omitempty"`
	Name *string `json:"name,omitempty"`
	NamespacePid int `json:"namespacePid,omitempty"`
	ParentUuid *string `json:"parentUuid,omitempty"`
	Pid int `json:"pid,omitempty"`
	Pwd *string `json:"pwd,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	User *string `json:"user,omitempty"`
	UserId int `json:"userId,omitempty"`
	Uuid *string `json:"uuid,omitempty"`
}

type ProductCode struct {
	Code *string `json:"productCodeId,omitempty"`
	ProductType *string `json:"productCodeType,omitempty"`
}

type PublicAccess struct {
	EffectivePermission *string `json:"effectivePermission,omitempty"`
	PermissionConfiguration *PermissionConfiguration `json:"permissionConfiguration,omitempty"`
}

type PublicAccessConfiguration struct {
	PublicAclAccess *string `json:"publicAclAccess,omitempty"`
	PublicAclIgnoreBehavior *string `json:"publicAclIgnoreBehavior,omitempty"`
	PublicBucketRestrictBehavior *string `json:"publicBucketRestrictBehavior,omitempty"`
	PublicPolicyAccess *string `json:"publicPolicyAccess,omitempty"`
}

type RdsDbInstanceDetails struct {
	DbClusterIdentifier *string `json:"dbClusterIdentifier,omitempty"`
	DbInstanceArn *string `json:"dbInstanceArn,omitempty"`
	DbInstanceIdentifier *string `json:"dbInstanceIdentifier,omitempty"`
	DbiResourceId *string `json:"dbiResourceId,omitempty"`
	Engine *string `json:"engine,omitempty"`
	EngineVersion *string `json:"engineVersion,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type RdsDbUserDetails struct {
	Application *string `json:"application,omitempty"`
	AuthMethod *string `json:"authMethod,omitempty"`
	Database *string `json:"database,omitempty"`
	Ssl *string `json:"ssl,omitempty"`
	User *string `json:"user,omitempty"`
}

type RdsLimitlessDbDetails struct {
	DbClusterIdentifier *string `json:"dbClusterIdentifier,omitempty"`
	DbShardGroupArn *string `json:"dbShardGroupArn,omitempty"`
	DbShardGroupIdentifier *string `json:"dbShardGroupIdentifier,omitempty"`
	DbShardGroupResourceId *string `json:"dbShardGroupResourceId,omitempty"`
	Engine *string `json:"engine,omitempty"`
	EngineVersion *string `json:"engineVersion,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type RdsLoginAttemptAction struct {
	LoginAttributes []LoginAttribute `json:"LoginAttributes,omitempty"`
	RemoteIpDetails *RemoteIpDetails `json:"remoteIpDetails,omitempty"`
}

type RecoveryPoint struct {
	BackupVaultName string `json:"backupVaultName,omitempty"`
}

type RecoveryPointDetails struct {
	BackupVaultName *string `json:"backupVaultName,omitempty"`
	RecoveryPointArn *string `json:"recoveryPointArn,omitempty"`
}

type RemoteAccountDetails struct {
	AccountId *string `json:"accountId,omitempty"`
	Affiliated bool `json:"affiliated,omitempty"`
}

type RemoteIpDetails struct {
	City *City `json:"city,omitempty"`
	Country *Country `json:"country,omitempty"`
	GeoLocation *GeoLocation `json:"geoLocation,omitempty"`
	IpAddressV4 *string `json:"ipAddressV4,omitempty"`
	IpAddressV6 *string `json:"ipAddressV6,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

type RemotePortDetails struct {
	Port int `json:"port,omitempty"`
	PortName *string `json:"portName,omitempty"`
}

type Resource struct {
	AccessKeyDetails *AccessKeyDetails `json:"accessKeyDetails,omitempty"`
	ContainerDetails *Container `json:"containerDetails,omitempty"`
	EbsSnapshotDetails *EbsSnapshotDetails `json:"ebsSnapshotDetails,omitempty"`
	EbsVolumeDetails *EbsVolumeDetails `json:"ebsVolumeDetails,omitempty"`
	Ec2ImageDetails *Ec2ImageDetails `json:"ec2ImageDetails,omitempty"`
	EcsClusterDetails *EcsClusterDetails `json:"ecsClusterDetails,omitempty"`
	EksClusterDetails *EksClusterDetails `json:"eksClusterDetails,omitempty"`
	InstanceDetails *InstanceDetails `json:"instanceDetails,omitempty"`
	KubernetesDetails *KubernetesDetails `json:"kubernetesDetails,omitempty"`
	LambdaDetails *LambdaDetails `json:"lambdaDetails,omitempty"`
	RdsDbInstanceDetails *RdsDbInstanceDetails `json:"rdsDbInstanceDetails,omitempty"`
	RdsDbUserDetails *RdsDbUserDetails `json:"rdsDbUserDetails,omitempty"`
	RdsLimitlessDbDetails *RdsLimitlessDbDetails `json:"rdsLimitlessDbDetails,omitempty"`
	RecoveryPointDetails *RecoveryPointDetails `json:"recoveryPointDetails,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	S3BucketDetails []S3BucketDetail `json:"s3BucketDetails,omitempty"`
}

type ResourceData struct {
	AccessKey *AccessKey `json:"accessKey,omitempty"`
	AutoscalingAutoScalingGroup *AutoscalingAutoScalingGroup `json:"autoscalingAutoScalingGroup,omitempty"`
	CloudformationStack *CloudformationStack `json:"cloudformationStack,omitempty"`
	Container *ContainerFindingResource `json:"container,omitempty"`
	Ec2Image *Ec2Image `json:"ec2Image,omitempty"`
	Ec2Instance *Ec2Instance `json:"ec2Instance,omitempty"`
	Ec2LaunchTemplate *Ec2LaunchTemplate `json:"ec2LaunchTemplate,omitempty"`
	Ec2NetworkInterface *Ec2NetworkInterface `json:"ec2NetworkInterface,omitempty"`
	Ec2Vpc *Ec2Vpc `json:"ec2Vpc,omitempty"`
	EcsCluster *EcsCluster `json:"ecsCluster,omitempty"`
	EcsTask *EcsTask `json:"ecsTask,omitempty"`
	EksCluster *EksCluster `json:"eksCluster,omitempty"`
	IamInstanceProfile *IamInstanceProfileV2 `json:"iamInstanceProfile,omitempty"`
	KubernetesWorkload *KubernetesWorkload `json:"kubernetesWorkload,omitempty"`
	S3Bucket *S3Bucket `json:"s3Bucket,omitempty"`
	S3Object *S3Object `json:"s3Object,omitempty"`
}

type ResourceDetails struct {
	InstanceArn *string `json:"instanceArn,omitempty"`
}

type ResourceStatistics struct {
	AccountId *string `json:"accountId,omitempty"`
	LastGeneratedAt *time.Time `json:"lastGeneratedAt,omitempty"`
	ResourceId *string `json:"resourceId,omitempty"`
	ResourceType *string `json:"resourceType,omitempty"`
	TotalFindings int `json:"totalFindings,omitempty"`
}

type ResourceV2 struct {
	AccountId *string `json:"accountId,omitempty"`
	CloudPartition *string `json:"cloudPartition,omitempty"`
	Data *ResourceData `json:"data,omitempty"`
	Name *string `json:"name,omitempty"`
	Region *string `json:"region,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	ServiceModel *string `json:"service,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	Uid string `json:"uid,omitempty"`
}

type RuntimeContext struct {
	AddressFamily *string `json:"addressFamily,omitempty"`
	CommandLineExample *string `json:"commandLineExample,omitempty"`
	FileSystemType *string `json:"fileSystemType,omitempty"`
	Flags []string `json:"flags,omitempty"`
	IanaProtocolNumber int `json:"ianaProtocolNumber,omitempty"`
	LdPreloadValue *string `json:"ldPreloadValue,omitempty"`
	LibraryPath *string `json:"libraryPath,omitempty"`
	MemoryRegions []string `json:"memoryRegions,omitempty"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
	ModifyingProcess *ProcessDetails `json:"modifyingProcess,omitempty"`
	ModuleFilePath *string `json:"moduleFilePath,omitempty"`
	ModuleName *string `json:"moduleName,omitempty"`
	ModuleSha256 *string `json:"moduleSha256,omitempty"`
	MountSource *string `json:"mountSource,omitempty"`
	MountTarget *string `json:"mountTarget,omitempty"`
	ReleaseAgentPath *string `json:"releaseAgentPath,omitempty"`
	RuncBinaryPath *string `json:"runcBinaryPath,omitempty"`
	ScriptPath *string `json:"scriptPath,omitempty"`
	ServiceName *string `json:"serviceName,omitempty"`
	ShellHistoryFilePath *string `json:"shellHistoryFilePath,omitempty"`
	SocketPath *string `json:"socketPath,omitempty"`
	TargetProcess *ProcessDetails `json:"targetProcess,omitempty"`
	ThreatFilePath *string `json:"threatFilePath,omitempty"`
	ToolCategory *string `json:"toolCategory,omitempty"`
	ToolName *string `json:"toolName,omitempty"`
}

type RuntimeDetails struct {
	Context *RuntimeContext `json:"context,omitempty"`
	Process *ProcessDetails `json:"process,omitempty"`
}

type S3Bucket struct {
	AccountPublicAccess *PublicAccessConfiguration `json:"accountPublicAccess,omitempty"`
	BucketPublicAccess *PublicAccessConfiguration `json:"bucketPublicAccess,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	EffectivePermission *string `json:"effectivePermission,omitempty"`
	EncryptionKeyArn *string `json:"encryptionKeyArn,omitempty"`
	EncryptionType *string `json:"encryptionType,omitempty"`
	OwnerId *string `json:"ownerId,omitempty"`
	PublicReadAccess *string `json:"publicReadAccess,omitempty"`
	PublicWriteAccess *string `json:"publicWriteAccess,omitempty"`
	S3ObjectUids []string `json:"s3ObjectUids,omitempty"`
}

type S3BucketDetail struct {
	Arn *string `json:"arn,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	DefaultServerSideEncryption *DefaultServerSideEncryption `json:"defaultServerSideEncryption,omitempty"`
	Name *string `json:"name,omitempty"`
	Owner *Owner `json:"owner,omitempty"`
	PublicAccess *PublicAccess `json:"publicAccess,omitempty"`
	S3ObjectDetails []S3ObjectDetail `json:"s3ObjectDetails,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	Type *string `json:"type,omitempty"`
}

type S3LogsConfiguration struct {
	Enable bool `json:"enable,omitempty"`
}

type S3LogsConfigurationResult struct {
	Status string `json:"status,omitempty"`
}

type S3Object struct {
	ETag *string `json:"eTag,omitempty"`
	Key *string `json:"key,omitempty"`
	VersionId *string `json:"versionId,omitempty"`
}

type S3ObjectDetail struct {
	ETag *string `json:"eTag,omitempty"`
	Hash *string `json:"hash,omitempty"`
	Key *string `json:"key,omitempty"`
	ObjectArn *string `json:"objectArn,omitempty"`
	VersionId *string `json:"versionId,omitempty"`
}

type S3ObjectForSendObjectMalwareScan struct {
	Bucket *string `json:"bucket,omitempty"`
	Key *string `json:"key,omitempty"`
	VersionId *string `json:"versionId,omitempty"`
}

type Scan struct {
	AccountId *string `json:"accountId,omitempty"`
	AdminDetectorId *string `json:"adminDetectorId,omitempty"`
	AttachedVolumes []VolumeDetail `json:"attachedVolumes,omitempty"`
	DetectorId *string `json:"detectorId,omitempty"`
	FailureReason *string `json:"failureReason,omitempty"`
	FileCount int64 `json:"fileCount,omitempty"`
	ResourceDetails *ResourceDetails `json:"resourceDetails,omitempty"`
	ScanEndTime *time.Time `json:"scanEndTime,omitempty"`
	ScanId *string `json:"scanId,omitempty"`
	ScanResultDetails *ScanResultDetails `json:"scanResultDetails,omitempty"`
	ScanStartTime *time.Time `json:"scanStartTime,omitempty"`
	ScanStatus *string `json:"scanStatus,omitempty"`
	ScanType *string `json:"scanType,omitempty"`
	TotalBytes int64 `json:"totalBytes,omitempty"`
	TriggerDetails *TriggerDetails `json:"triggerDetails,omitempty"`
}

type ScanCondition struct {
	MapEquals []ScanConditionPair `json:"mapEquals,omitempty"`
}

type ScanConditionPair struct {
	Key string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type ScanConfiguration struct {
	IncrementalScanDetails *IncrementalScanDetails `json:"incrementalScanDetails,omitempty"`
	RecoveryPoint *ScanConfigurationRecoveryPoint `json:"recoveryPoint,omitempty"`
	Role *string `json:"role,omitempty"`
	TriggerDetails *TriggerDetails `json:"triggerDetails,omitempty"`
}

type ScanConfigurationRecoveryPoint struct {
	BackupVaultName *string `json:"backupVaultName,omitempty"`
}

type ScanDetections struct {
	HighestSeverityThreatDetails *HighestSeverityThreatDetails `json:"highestSeverityThreatDetails,omitempty"`
	ScannedItemCount *ScannedItemCount `json:"scannedItemCount,omitempty"`
	ThreatDetectedByName *ThreatDetectedByName `json:"threatDetectedByName,omitempty"`
	ThreatsDetectedItemCount *ThreatsDetectedItemCount `json:"threatsDetectedItemCount,omitempty"`
}

type ScanEc2InstanceWithFindings struct {
	EbsVolumes bool `json:"ebsVolumes,omitempty"`
}

type ScanEc2InstanceWithFindingsResult struct {
	EbsVolumes *EbsVolumesResult `json:"ebsVolumes,omitempty"`
}

type ScanFilePath struct {
	FileName *string `json:"fileName,omitempty"`
	FilePath *string `json:"filePath,omitempty"`
	Hash *string `json:"hash,omitempty"`
	VolumeArn *string `json:"volumeArn,omitempty"`
}

type ScanResourceCriteria struct {
	Exclude map[string]ScanCondition `json:"exclude,omitempty"`
	Include map[string]ScanCondition `json:"include,omitempty"`
}

type ScanResultDetails struct {
	ScanResult *string `json:"scanResult,omitempty"`
}

type ScanResultThreat struct {
	Count int64 `json:"count,omitempty"`
	Hash *string `json:"hash,omitempty"`
	ItemDetails []ItemDetails `json:"itemDetails,omitempty"`
	Name *string `json:"name,omitempty"`
	Source *string `json:"source,omitempty"`
}

type ScanThreatName struct {
	FilePaths []ScanFilePath `json:"filePaths,omitempty"`
	ItemCount int `json:"itemCount,omitempty"`
	Name *string `json:"name,omitempty"`
	Severity *string `json:"severity,omitempty"`
}

type ScannedItemCount struct {
	Files int `json:"files,omitempty"`
	TotalGb int `json:"totalGb,omitempty"`
	Volumes int `json:"volumes,omitempty"`
}

type ScannedResource struct {
	ResourceDetails *ScannedResourceDetails `json:"resourceDetails,omitempty"`
	ScanStatusReason *string `json:"scanStatusReason,omitempty"`
	ScannedResourceArn *string `json:"scannedResourceArn,omitempty"`
	ScannedResourceStatus *string `json:"scannedResourceStatus,omitempty"`
	ScannedResourceType *string `json:"scannedResourceType,omitempty"`
}

type ScannedResourceDetails struct {
	EbsSnapshot *EbsSnapshot `json:"ebsSnapshot,omitempty"`
	EbsVolume *VolumeDetail `json:"ebsVolume,omitempty"`
}

type SecurityContext struct {
	AllowPrivilegeEscalation bool `json:"allowPrivilegeEscalation,omitempty"`
	Privileged bool `json:"privileged,omitempty"`
}

type SecurityGroup struct {
	GroupId *string `json:"groupId,omitempty"`
	GroupName *string `json:"groupName,omitempty"`
}

type SendObjectMalwareScanRequest struct {
	S3Object *S3ObjectForSendObjectMalwareScan `json:"s3Object,omitempty"`
}

type SendObjectMalwareScanResponse struct {
}

type Sequence struct {
	Actors []Actor `json:"actors,omitempty"`
	AdditionalSequenceTypes []string `json:"additionalSequenceTypes,omitempty"`
	Description string `json:"description,omitempty"`
	Endpoints []NetworkEndpoint `json:"endpoints,omitempty"`
	Resources []ResourceV2 `json:"resources,omitempty"`
	SequenceIndicators []Indicator `json:"sequenceIndicators,omitempty"`
	Signals []Signal `json:"signals,omitempty"`
	Uid string `json:"uid,omitempty"`
}

type ServiceModel struct {
	Action *Action `json:"action,omitempty"`
	AdditionalInfo *ServiceAdditionalInfo `json:"additionalInfo,omitempty"`
	Archived bool `json:"archived,omitempty"`
	Count int `json:"count,omitempty"`
	Detection *Detection `json:"detection,omitempty"`
	DetectorId *string `json:"detectorId,omitempty"`
	EbsVolumeScanDetails *EbsVolumeScanDetails `json:"ebsVolumeScanDetails,omitempty"`
	EventFirstSeen *string `json:"eventFirstSeen,omitempty"`
	EventLastSeen *string `json:"eventLastSeen,omitempty"`
	Evidence *Evidence `json:"evidence,omitempty"`
	FeatureName *string `json:"featureName,omitempty"`
	MalwareScanDetails *MalwareScanDetails `json:"malwareScanDetails,omitempty"`
	ResourceRole *string `json:"resourceRole,omitempty"`
	RuntimeDetails *RuntimeDetails `json:"runtimeDetails,omitempty"`
	ServiceName *string `json:"serviceName,omitempty"`
	UserFeedback *string `json:"userFeedback,omitempty"`
}

type ServiceAdditionalInfo struct {
	Type *string `json:"type,omitempty"`
	Value *string `json:"value,omitempty"`
}

type Session struct {
	CreatedTime *time.Time `json:"createdTime,omitempty"`
	Issuer *string `json:"issuer,omitempty"`
	MfaStatus *string `json:"mfaStatus,omitempty"`
	Uid *string `json:"uid,omitempty"`
}

type SeverityStatistics struct {
	LastGeneratedAt *time.Time `json:"lastGeneratedAt,omitempty"`
	Severity float64 `json:"severity,omitempty"`
	TotalFindings int `json:"totalFindings,omitempty"`
}

type Signal struct {
	ActorIds []string `json:"actorIds,omitempty"`
	Count int `json:"count,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	Description *string `json:"description,omitempty"`
	EndpointIds []string `json:"endpointIds,omitempty"`
	FirstSeenAt time.Time `json:"firstSeenAt,omitempty"`
	LastSeenAt time.Time `json:"lastSeenAt,omitempty"`
	Name string `json:"name,omitempty"`
	ResourceUids []string `json:"resourceUids,omitempty"`
	Severity float64 `json:"severity,omitempty"`
	SignalIndicators []Indicator `json:"signalIndicators,omitempty"`
	Type string `json:"type,omitempty"`
	Uid string `json:"uid,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type SortCriteria struct {
	AttributeName *string `json:"attributeName,omitempty"`
	OrderBy *string `json:"orderBy,omitempty"`
}

type StartMalwareScanConfiguration struct {
	IncrementalScanDetails *IncrementalScanDetails `json:"incrementalScanDetails,omitempty"`
	RecoveryPoint *RecoveryPoint `json:"recoveryPoint,omitempty"`
	Role string `json:"role,omitempty"`
}

type StartMalwareScanRequest struct {
	ClientToken *string `json:"clientToken,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
	ScanConfiguration *StartMalwareScanConfiguration `json:"scanConfiguration,omitempty"`
}

type StartMalwareScanResponse struct {
	ScanId *string `json:"scanId,omitempty"`
}

type StartMonitoringMembersRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type StartMonitoringMembersResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type StopMonitoringMembersRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type StopMonitoringMembersResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type Tag struct {
	Key *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	Tags map[string]string `json:"tags,omitempty"`
}

type TagResourceResponse struct {
}

type Threat struct {
	Count int64 `json:"count,omitempty"`
	Hash *string `json:"hash,omitempty"`
	ItemDetails []ItemDetails `json:"itemDetails,omitempty"`
	ItemPaths []ItemPath `json:"itemPaths,omitempty"`
	Name *string `json:"name,omitempty"`
	Source *string `json:"source,omitempty"`
}

type ThreatDetectedByName struct {
	ItemCount int `json:"itemCount,omitempty"`
	Shortened bool `json:"shortened,omitempty"`
	ThreatNames []ScanThreatName `json:"threatNames,omitempty"`
	UniqueThreatNameCount int `json:"uniqueThreatNameCount,omitempty"`
}

type ThreatIntelligenceDetail struct {
	ThreatFileSha256 *string `json:"threatFileSha256,omitempty"`
	ThreatListName *string `json:"threatListName,omitempty"`
	ThreatNames []string `json:"threatNames,omitempty"`
}

type ThreatsDetectedItemCount struct {
	Files int `json:"files,omitempty"`
}

type Total struct {
	Amount *string `json:"amount,omitempty"`
	Unit *string `json:"unit,omitempty"`
}

type TriggerDetails struct {
	Description *string `json:"description,omitempty"`
	GuardDutyFindingId *string `json:"guardDutyFindingId,omitempty"`
	TriggerType *string `json:"triggerType,omitempty"`
}

type UnarchiveFindingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	FindingIds []string `json:"findingIds,omitempty"`
}

type UnarchiveFindingsResponse struct {
}

type UnprocessedAccount struct {
	AccountId string `json:"accountId,omitempty"`
	Result string `json:"result,omitempty"`
}

type UnprocessedDataSourcesResult struct {
	MalwareProtection *MalwareProtectionConfigurationResult `json:"malwareProtection,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	TagKeys []string `json:"tagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateDetectorRequest struct {
	DataSources *DataSourceConfigurations `json:"dataSources,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	Enable bool `json:"enable,omitempty"`
	Features []DetectorFeatureConfiguration `json:"features,omitempty"`
	FindingPublishingFrequency *string `json:"findingPublishingFrequency,omitempty"`
}

type UpdateDetectorResponse struct {
}

type UpdateFilterRequest struct {
	Action *string `json:"action,omitempty"`
	Description *string `json:"description,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	FilterName string `json:"filterName,omitempty"`
	FindingCriteria *FindingCriteria `json:"findingCriteria,omitempty"`
	Rank int `json:"rank,omitempty"`
}

type UpdateFilterResponse struct {
	Name string `json:"name,omitempty"`
}

type UpdateFindingsFeedbackRequest struct {
	Comments *string `json:"comments,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	Feedback string `json:"feedback,omitempty"`
	FindingIds []string `json:"findingIds,omitempty"`
}

type UpdateFindingsFeedbackResponse struct {
}

type UpdateIPSetRequest struct {
	Activate bool `json:"activate,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	IpSetId string `json:"ipSetId,omitempty"`
	Location *string `json:"location,omitempty"`
	Name *string `json:"name,omitempty"`
}

type UpdateIPSetResponse struct {
}

type UpdateMalwareProtectionPlanRequest struct {
	Actions *MalwareProtectionPlanActions `json:"actions,omitempty"`
	MalwareProtectionPlanId string `json:"malwareProtectionPlanId,omitempty"`
	ProtectedResource *UpdateProtectedResource `json:"protectedResource,omitempty"`
	Role *string `json:"role,omitempty"`
}

type UpdateMalwareScanSettingsRequest struct {
	DetectorId string `json:"detectorId,omitempty"`
	EbsSnapshotPreservation *string `json:"ebsSnapshotPreservation,omitempty"`
	ScanResourceCriteria *ScanResourceCriteria `json:"scanResourceCriteria,omitempty"`
}

type UpdateMalwareScanSettingsResponse struct {
}

type UpdateMemberDetectorsRequest struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DataSources *DataSourceConfigurations `json:"dataSources,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	Features []MemberFeaturesConfiguration `json:"features,omitempty"`
}

type UpdateMemberDetectorsResponse struct {
	UnprocessedAccounts []UnprocessedAccount `json:"unprocessedAccounts,omitempty"`
}

type UpdateOrganizationConfigurationRequest struct {
	AutoEnable bool `json:"autoEnable,omitempty"`
	AutoEnableOrganizationMembers *string `json:"autoEnableOrganizationMembers,omitempty"`
	DataSources *OrganizationDataSourceConfigurations `json:"dataSources,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	Features []OrganizationFeatureConfiguration `json:"features,omitempty"`
}

type UpdateOrganizationConfigurationResponse struct {
}

type UpdateProtectedResource struct {
	S3Bucket *UpdateS3BucketResource `json:"s3Bucket,omitempty"`
}

type UpdatePublishingDestinationRequest struct {
	DestinationId string `json:"destinationId,omitempty"`
	DestinationProperties *DestinationProperties `json:"destinationProperties,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
}

type UpdatePublishingDestinationResponse struct {
}

type UpdateS3BucketResource struct {
	ObjectPrefixes []string `json:"objectPrefixes,omitempty"`
}

type UpdateThreatEntitySetRequest struct {
	Activate bool `json:"activate,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Location *string `json:"location,omitempty"`
	Name *string `json:"name,omitempty"`
	ThreatEntitySetId string `json:"threatEntitySetId,omitempty"`
}

type UpdateThreatEntitySetResponse struct {
}

type UpdateThreatIntelSetRequest struct {
	Activate bool `json:"activate,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Location *string `json:"location,omitempty"`
	Name *string `json:"name,omitempty"`
	ThreatIntelSetId string `json:"threatIntelSetId,omitempty"`
}

type UpdateThreatIntelSetResponse struct {
}

type UpdateTrustedEntitySetRequest struct {
	Activate bool `json:"activate,omitempty"`
	DetectorId string `json:"detectorId,omitempty"`
	ExpectedBucketOwner *string `json:"expectedBucketOwner,omitempty"`
	Location *string `json:"location,omitempty"`
	Name *string `json:"name,omitempty"`
	TrustedEntitySetId string `json:"trustedEntitySetId,omitempty"`
}

type UpdateTrustedEntitySetResponse struct {
}

type UsageAccountResult struct {
	AccountId *string `json:"accountId,omitempty"`
	Total *Total `json:"total,omitempty"`
}

type UsageCriteria struct {
	AccountIds []string `json:"accountIds,omitempty"`
	DataSources []string `json:"dataSources,omitempty"`
	Features []string `json:"features,omitempty"`
	Resources []string `json:"resources,omitempty"`
}

type UsageDataSourceResult struct {
	DataSource *string `json:"dataSource,omitempty"`
	Total *Total `json:"total,omitempty"`
}

type UsageFeatureResult struct {
	Feature *string `json:"feature,omitempty"`
	Total *Total `json:"total,omitempty"`
}

type UsageResourceResult struct {
	Resource *string `json:"resource,omitempty"`
	Total *Total `json:"total,omitempty"`
}

type UsageStatistics struct {
	SumByAccount []UsageAccountResult `json:"sumByAccount,omitempty"`
	SumByDataSource []UsageDataSourceResult `json:"sumByDataSource,omitempty"`
	SumByFeature []UsageFeatureResult `json:"sumByFeature,omitempty"`
	SumByResource []UsageResourceResult `json:"sumByResource,omitempty"`
	TopAccountsByFeature []UsageTopAccountsResult `json:"topAccountsByFeature,omitempty"`
	TopResources []UsageResourceResult `json:"topResources,omitempty"`
}

type UsageTopAccountResult struct {
	AccountId *string `json:"accountId,omitempty"`
	Total *Total `json:"total,omitempty"`
}

type UsageTopAccountsResult struct {
	Accounts []UsageTopAccountResult `json:"accounts,omitempty"`
	Feature *string `json:"feature,omitempty"`
}

type User struct {
	Account *Account `json:"account,omitempty"`
	CredentialUid *string `json:"credentialUid,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Uid string `json:"uid,omitempty"`
}

type Volume struct {
	HostPath *HostPath `json:"hostPath,omitempty"`
	Name *string `json:"name,omitempty"`
}

type VolumeDetail struct {
	DeviceName *string `json:"deviceName,omitempty"`
	EncryptionType *string `json:"encryptionType,omitempty"`
	KmsKeyArn *string `json:"kmsKeyArn,omitempty"`
	SnapshotArn *string `json:"snapshotArn,omitempty"`
	VolumeArn *string `json:"volumeArn,omitempty"`
	VolumeSizeInGB int `json:"volumeSizeInGB,omitempty"`
	VolumeType *string `json:"volumeType,omitempty"`
}

type VolumeMount struct {
	MountPath *string `json:"mountPath,omitempty"`
	Name *string `json:"name,omitempty"`
}

type VpcConfig struct {
	SecurityGroups []SecurityGroup `json:"securityGroups,omitempty"`
	SubnetIds []string `json:"subnetIds,omitempty"`
	VpcId *string `json:"vpcId,omitempty"`
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

func handleAcceptAdministratorInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AcceptAdministratorInvitationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AcceptAdministratorInvitation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AcceptAdministratorInvitation"})
}

func handleAcceptInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AcceptInvitationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AcceptInvitation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AcceptInvitation"})
}

func handleArchiveFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ArchiveFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ArchiveFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ArchiveFindings"})
}

func handleCreateDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDetectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDetector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDetector"})
}

func handleCreateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFilter"})
}

func handleCreateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateIPSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateIPSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateIPSet"})
}

func handleCreateMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateMalwareProtectionPlanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateMalwareProtectionPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateMalwareProtectionPlan"})
}

func handleCreateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateMembers"})
}

func handleCreatePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreatePublishingDestinationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreatePublishingDestination business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreatePublishingDestination"})
}

func handleCreateSampleFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateSampleFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateSampleFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateSampleFindings"})
}

func handleCreateThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateThreatEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateThreatEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateThreatEntitySet"})
}

func handleCreateThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateThreatIntelSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateThreatIntelSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateThreatIntelSet"})
}

func handleCreateTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTrustedEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTrustedEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTrustedEntitySet"})
}

func handleDeclineInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeclineInvitationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeclineInvitations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeclineInvitations"})
}

func handleDeleteDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDetectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDetector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDetector"})
}

func handleDeleteFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFilter"})
}

func handleDeleteIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteIPSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteIPSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteIPSet"})
}

func handleDeleteInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteInvitationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteInvitations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteInvitations"})
}

func handleDeleteMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteMalwareProtectionPlanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteMalwareProtectionPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteMalwareProtectionPlan"})
}

func handleDeleteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteMembers"})
}

func handleDeletePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeletePublishingDestinationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeletePublishingDestination business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeletePublishingDestination"})
}

func handleDeleteThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteThreatEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteThreatEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteThreatEntitySet"})
}

func handleDeleteThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteThreatIntelSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteThreatIntelSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteThreatIntelSet"})
}

func handleDeleteTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTrustedEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTrustedEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTrustedEntitySet"})
}

func handleDescribeMalwareScans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeMalwareScansRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeMalwareScans business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeMalwareScans"})
}

func handleDescribeOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeOrganizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeOrganizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeOrganizationConfiguration"})
}

func handleDescribePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribePublishingDestinationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribePublishingDestination business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribePublishingDestination"})
}

func handleDisableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableOrganizationAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableOrganizationAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableOrganizationAdminAccount"})
}

func handleDisassociateFromAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateFromAdministratorAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateFromAdministratorAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateFromAdministratorAccount"})
}

func handleDisassociateFromMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateFromMasterAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateFromMasterAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateFromMasterAccount"})
}

func handleDisassociateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisassociateMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisassociateMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisassociateMembers"})
}

func handleEnableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableOrganizationAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableOrganizationAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableOrganizationAdminAccount"})
}

func handleGetAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetAdministratorAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetAdministratorAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetAdministratorAccount"})
}

func handleGetCoverageStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetCoverageStatisticsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetCoverageStatistics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetCoverageStatistics"})
}

func handleGetDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetDetectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetDetector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetDetector"})
}

func handleGetFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFilter"})
}

func handleGetFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindings"})
}

func handleGetFindingsStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingsStatisticsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingsStatistics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingsStatistics"})
}

func handleGetIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetIPSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetIPSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetIPSet"})
}

func handleGetInvitationsCount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetInvitationsCountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetInvitationsCount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetInvitationsCount"})
}

func handleGetMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMalwareProtectionPlanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMalwareProtectionPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMalwareProtectionPlan"})
}

func handleGetMalwareScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMalwareScanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMalwareScan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMalwareScan"})
}

func handleGetMalwareScanSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMalwareScanSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMalwareScanSettings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMalwareScanSettings"})
}

func handleGetMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMasterAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMasterAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMasterAccount"})
}

func handleGetMemberDetectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMemberDetectorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMemberDetectors business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMemberDetectors"})
}

func handleGetMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMembers"})
}

func handleGetOrganizationStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	// TODO: implement GetOrganizationStatistics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetOrganizationStatistics"})
}

func handleGetRemainingFreeTrialDays(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetRemainingFreeTrialDaysRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetRemainingFreeTrialDays business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetRemainingFreeTrialDays"})
}

func handleGetThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetThreatEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetThreatEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetThreatEntitySet"})
}

func handleGetThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetThreatIntelSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetThreatIntelSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetThreatIntelSet"})
}

func handleGetTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetTrustedEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetTrustedEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetTrustedEntitySet"})
}

func handleGetUsageStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetUsageStatisticsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetUsageStatistics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetUsageStatistics"})
}

func handleInviteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req InviteMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement InviteMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "InviteMembers"})
}

func handleListCoverage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCoverageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCoverage business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCoverage"})
}

func handleListDetectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDetectorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDetectors business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDetectors"})
}

func handleListFilters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFiltersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFilters business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFilters"})
}

func handleListFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFindings"})
}

func handleListIPSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListIPSetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListIPSets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListIPSets"})
}

func handleListInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListInvitationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListInvitations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListInvitations"})
}

func handleListMalwareProtectionPlans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListMalwareProtectionPlansRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListMalwareProtectionPlans business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListMalwareProtectionPlans"})
}

func handleListMalwareScans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListMalwareScansRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListMalwareScans business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListMalwareScans"})
}

func handleListMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListMembers"})
}

func handleListOrganizationAdminAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListOrganizationAdminAccountsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListOrganizationAdminAccounts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListOrganizationAdminAccounts"})
}

func handleListPublishingDestinations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListPublishingDestinationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListPublishingDestinations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListPublishingDestinations"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListThreatEntitySets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListThreatEntitySetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListThreatEntitySets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListThreatEntitySets"})
}

func handleListThreatIntelSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListThreatIntelSetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListThreatIntelSets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListThreatIntelSets"})
}

func handleListTrustedEntitySets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTrustedEntitySetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTrustedEntitySets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTrustedEntitySets"})
}

func handleSendObjectMalwareScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SendObjectMalwareScanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SendObjectMalwareScan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SendObjectMalwareScan"})
}

func handleStartMalwareScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartMalwareScanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartMalwareScan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartMalwareScan"})
}

func handleStartMonitoringMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartMonitoringMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartMonitoringMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartMonitoringMembers"})
}

func handleStopMonitoringMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StopMonitoringMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StopMonitoringMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StopMonitoringMembers"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUnarchiveFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UnarchiveFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UnarchiveFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UnarchiveFindings"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUpdateDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDetectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDetector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDetector"})
}

func handleUpdateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFilterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFilter business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFilter"})
}

func handleUpdateFindingsFeedback(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFindingsFeedbackRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFindingsFeedback business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFindingsFeedback"})
}

func handleUpdateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateIPSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateIPSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateIPSet"})
}

func handleUpdateMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateMalwareProtectionPlanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateMalwareProtectionPlan business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateMalwareProtectionPlan"})
}

func handleUpdateMalwareScanSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateMalwareScanSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateMalwareScanSettings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateMalwareScanSettings"})
}

func handleUpdateMemberDetectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateMemberDetectorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateMemberDetectors business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateMemberDetectors"})
}

func handleUpdateOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateOrganizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateOrganizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateOrganizationConfiguration"})
}

func handleUpdatePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdatePublishingDestinationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdatePublishingDestination business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdatePublishingDestination"})
}

func handleUpdateThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateThreatEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateThreatEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateThreatEntitySet"})
}

func handleUpdateThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateThreatIntelSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateThreatIntelSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateThreatIntelSet"})
}

func handleUpdateTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTrustedEntitySetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTrustedEntitySet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTrustedEntitySet"})
}

