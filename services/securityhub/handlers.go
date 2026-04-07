package securityhub

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AcceptAdministratorInvitationRequest struct {
	AdministratorId string `json:"AdministratorId,omitempty"`
	InvitationId string `json:"InvitationId,omitempty"`
}

type AcceptAdministratorInvitationResponse struct {
}

type AcceptInvitationRequest struct {
	InvitationId string `json:"InvitationId,omitempty"`
	MasterId string `json:"MasterId,omitempty"`
}

type AcceptInvitationResponse struct {
}

type AccountDetails struct {
	AccountId string `json:"AccountId,omitempty"`
	Email *string `json:"Email,omitempty"`
}

type Action struct {
	ActionType *string `json:"ActionType,omitempty"`
	AwsApiCallAction *AwsApiCallAction `json:"AwsApiCallAction,omitempty"`
	DnsRequestAction *DnsRequestAction `json:"DnsRequestAction,omitempty"`
	NetworkConnectionAction *NetworkConnectionAction `json:"NetworkConnectionAction,omitempty"`
	PortProbeAction *PortProbeAction `json:"PortProbeAction,omitempty"`
}

type ActionLocalIpDetails struct {
	IpAddressV4 *string `json:"IpAddressV4,omitempty"`
}

type ActionLocalPortDetails struct {
	Port int `json:"Port,omitempty"`
	PortName *string `json:"PortName,omitempty"`
}

type ActionRemoteIpDetails struct {
	City *City `json:"City,omitempty"`
	Country *Country `json:"Country,omitempty"`
	GeoLocation *GeoLocation `json:"GeoLocation,omitempty"`
	IpAddressV4 *string `json:"IpAddressV4,omitempty"`
	Organization *IpOrganizationDetails `json:"Organization,omitempty"`
}

type ActionRemotePortDetails struct {
	Port int `json:"Port,omitempty"`
	PortName *string `json:"PortName,omitempty"`
}

type ActionTarget struct {
	ActionTargetArn string `json:"ActionTargetArn,omitempty"`
	Description string `json:"Description,omitempty"`
	Name string `json:"Name,omitempty"`
}

type Actor struct {
	Id *string `json:"Id,omitempty"`
	Session *ActorSession `json:"Session,omitempty"`
	User *ActorUser `json:"User,omitempty"`
}

type ActorSession struct {
	CreatedTime int64 `json:"CreatedTime,omitempty"`
	Issuer *string `json:"Issuer,omitempty"`
	MfaStatus *string `json:"MfaStatus,omitempty"`
	Uid *string `json:"Uid,omitempty"`
}

type ActorUser struct {
	Account *UserAccount `json:"Account,omitempty"`
	CredentialUid *string `json:"CredentialUid,omitempty"`
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
	Uid *string `json:"Uid,omitempty"`
}

type Adjustment struct {
	Metric *string `json:"Metric,omitempty"`
	Reason *string `json:"Reason,omitempty"`
}

type AdminAccount struct {
	AccountId *string `json:"AccountId,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AggregatorV2 struct {
	AggregatorV2Arn *string `json:"AggregatorV2Arn,omitempty"`
}

type AssociatedStandard struct {
	StandardsId *string `json:"StandardsId,omitempty"`
}

type AssociationFilters struct {
	AssociationStatus *string `json:"AssociationStatus,omitempty"`
	AssociationType *string `json:"AssociationType,omitempty"`
	ConfigurationPolicyId *string `json:"ConfigurationPolicyId,omitempty"`
}

type AssociationSetDetails struct {
	AssociationState *AssociationStateDetails `json:"AssociationState,omitempty"`
	GatewayId *string `json:"GatewayId,omitempty"`
	Main bool `json:"Main,omitempty"`
	RouteTableAssociationId *string `json:"RouteTableAssociationId,omitempty"`
	RouteTableId *string `json:"RouteTableId,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
}

type AssociationStateDetails struct {
	State *string `json:"State,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
}

type AutomationRulesAction struct {
	FindingFieldsUpdate *AutomationRulesFindingFieldsUpdate `json:"FindingFieldsUpdate,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AutomationRulesActionTypeObjectV2 struct {
	Type *string `json:"Type,omitempty"`
}

type AutomationRulesActionV2 struct {
	ExternalIntegrationConfiguration *ExternalIntegrationConfiguration `json:"ExternalIntegrationConfiguration,omitempty"`
	FindingFieldsUpdate *AutomationRulesFindingFieldsUpdateV2 `json:"FindingFieldsUpdate,omitempty"`
	Type string `json:"Type,omitempty"`
}

type AutomationRulesConfig struct {
	Actions []AutomationRulesAction `json:"Actions,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	CreatedBy *string `json:"CreatedBy,omitempty"`
	Criteria *AutomationRulesFindingFilters `json:"Criteria,omitempty"`
	Description *string `json:"Description,omitempty"`
	IsTerminal bool `json:"IsTerminal,omitempty"`
	RuleArn *string `json:"RuleArn,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	RuleOrder int `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type AutomationRulesFindingFieldsUpdate struct {
	Confidence int `json:"Confidence,omitempty"`
	Criticality int `json:"Criticality,omitempty"`
	Note *NoteUpdate `json:"Note,omitempty"`
	RelatedFindings []RelatedFinding `json:"RelatedFindings,omitempty"`
	Severity *SeverityUpdate `json:"Severity,omitempty"`
	Types []string `json:"Types,omitempty"`
	UserDefinedFields map[string]string `json:"UserDefinedFields,omitempty"`
	VerificationState *string `json:"VerificationState,omitempty"`
	Workflow *WorkflowUpdate `json:"Workflow,omitempty"`
}

type AutomationRulesFindingFieldsUpdateV2 struct {
	Comment *string `json:"Comment,omitempty"`
	SeverityId int `json:"SeverityId,omitempty"`
	StatusId int `json:"StatusId,omitempty"`
}

type AutomationRulesFindingFilters struct {
	AwsAccountId []StringFilter `json:"AwsAccountId,omitempty"`
	AwsAccountName []StringFilter `json:"AwsAccountName,omitempty"`
	CompanyName []StringFilter `json:"CompanyName,omitempty"`
	ComplianceAssociatedStandardsId []StringFilter `json:"ComplianceAssociatedStandardsId,omitempty"`
	ComplianceSecurityControlId []StringFilter `json:"ComplianceSecurityControlId,omitempty"`
	ComplianceStatus []StringFilter `json:"ComplianceStatus,omitempty"`
	Confidence []NumberFilter `json:"Confidence,omitempty"`
	CreatedAt []DateFilter `json:"CreatedAt,omitempty"`
	Criticality []NumberFilter `json:"Criticality,omitempty"`
	Description []StringFilter `json:"Description,omitempty"`
	FirstObservedAt []DateFilter `json:"FirstObservedAt,omitempty"`
	GeneratorId []StringFilter `json:"GeneratorId,omitempty"`
	Id []StringFilter `json:"Id,omitempty"`
	LastObservedAt []DateFilter `json:"LastObservedAt,omitempty"`
	NoteText []StringFilter `json:"NoteText,omitempty"`
	NoteUpdatedAt []DateFilter `json:"NoteUpdatedAt,omitempty"`
	NoteUpdatedBy []StringFilter `json:"NoteUpdatedBy,omitempty"`
	ProductArn []StringFilter `json:"ProductArn,omitempty"`
	ProductName []StringFilter `json:"ProductName,omitempty"`
	RecordState []StringFilter `json:"RecordState,omitempty"`
	RelatedFindingsId []StringFilter `json:"RelatedFindingsId,omitempty"`
	RelatedFindingsProductArn []StringFilter `json:"RelatedFindingsProductArn,omitempty"`
	ResourceApplicationArn []StringFilter `json:"ResourceApplicationArn,omitempty"`
	ResourceApplicationName []StringFilter `json:"ResourceApplicationName,omitempty"`
	ResourceDetailsOther []MapFilter `json:"ResourceDetailsOther,omitempty"`
	ResourceId []StringFilter `json:"ResourceId,omitempty"`
	ResourcePartition []StringFilter `json:"ResourcePartition,omitempty"`
	ResourceRegion []StringFilter `json:"ResourceRegion,omitempty"`
	ResourceTags []MapFilter `json:"ResourceTags,omitempty"`
	ResourceType []StringFilter `json:"ResourceType,omitempty"`
	SeverityLabel []StringFilter `json:"SeverityLabel,omitempty"`
	SourceUrl []StringFilter `json:"SourceUrl,omitempty"`
	Title []StringFilter `json:"Title,omitempty"`
	Type []StringFilter `json:"Type,omitempty"`
	UpdatedAt []DateFilter `json:"UpdatedAt,omitempty"`
	UserDefinedFields []MapFilter `json:"UserDefinedFields,omitempty"`
	VerificationState []StringFilter `json:"VerificationState,omitempty"`
	WorkflowStatus []StringFilter `json:"WorkflowStatus,omitempty"`
}

type AutomationRulesMetadata struct {
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	CreatedBy *string `json:"CreatedBy,omitempty"`
	Description *string `json:"Description,omitempty"`
	IsTerminal bool `json:"IsTerminal,omitempty"`
	RuleArn *string `json:"RuleArn,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	RuleOrder int `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type AutomationRulesMetadataV2 struct {
	Actions []AutomationRulesActionTypeObjectV2 `json:"Actions,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	RuleArn *string `json:"RuleArn,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	RuleOrder float64 `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type AvailabilityZone struct {
	SubnetId *string `json:"SubnetId,omitempty"`
	ZoneName *string `json:"ZoneName,omitempty"`
}

type AwsAmazonMqBrokerDetails struct {
	AuthenticationStrategy *string `json:"AuthenticationStrategy,omitempty"`
	AutoMinorVersionUpgrade bool `json:"AutoMinorVersionUpgrade,omitempty"`
	BrokerArn *string `json:"BrokerArn,omitempty"`
	BrokerId *string `json:"BrokerId,omitempty"`
	BrokerName *string `json:"BrokerName,omitempty"`
	DeploymentMode *string `json:"DeploymentMode,omitempty"`
	EncryptionOptions *AwsAmazonMqBrokerEncryptionOptionsDetails `json:"EncryptionOptions,omitempty"`
	EngineType *string `json:"EngineType,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	HostInstanceType *string `json:"HostInstanceType,omitempty"`
	LdapServerMetadata *AwsAmazonMqBrokerLdapServerMetadataDetails `json:"LdapServerMetadata,omitempty"`
	Logs *AwsAmazonMqBrokerLogsDetails `json:"Logs,omitempty"`
	MaintenanceWindowStartTime *AwsAmazonMqBrokerMaintenanceWindowStartTimeDetails `json:"MaintenanceWindowStartTime,omitempty"`
	PubliclyAccessible bool `json:"PubliclyAccessible,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	StorageType *string `json:"StorageType,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
	Users []AwsAmazonMqBrokerUsersDetails `json:"Users,omitempty"`
}

type AwsAmazonMqBrokerEncryptionOptionsDetails struct {
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	UseAwsOwnedKey bool `json:"UseAwsOwnedKey,omitempty"`
}

type AwsAmazonMqBrokerLdapServerMetadataDetails struct {
	Hosts []string `json:"Hosts,omitempty"`
	RoleBase *string `json:"RoleBase,omitempty"`
	RoleName *string `json:"RoleName,omitempty"`
	RoleSearchMatching *string `json:"RoleSearchMatching,omitempty"`
	RoleSearchSubtree bool `json:"RoleSearchSubtree,omitempty"`
	ServiceAccountUsername *string `json:"ServiceAccountUsername,omitempty"`
	UserBase *string `json:"UserBase,omitempty"`
	UserRoleName *string `json:"UserRoleName,omitempty"`
	UserSearchMatching *string `json:"UserSearchMatching,omitempty"`
	UserSearchSubtree bool `json:"UserSearchSubtree,omitempty"`
}

type AwsAmazonMqBrokerLogsDetails struct {
	Audit bool `json:"Audit,omitempty"`
	AuditLogGroup *string `json:"AuditLogGroup,omitempty"`
	General bool `json:"General,omitempty"`
	GeneralLogGroup *string `json:"GeneralLogGroup,omitempty"`
	Pending *AwsAmazonMqBrokerLogsPendingDetails `json:"Pending,omitempty"`
}

type AwsAmazonMqBrokerLogsPendingDetails struct {
	Audit bool `json:"Audit,omitempty"`
	General bool `json:"General,omitempty"`
}

type AwsAmazonMqBrokerMaintenanceWindowStartTimeDetails struct {
	DayOfWeek *string `json:"DayOfWeek,omitempty"`
	TimeOfDay *string `json:"TimeOfDay,omitempty"`
	TimeZone *string `json:"TimeZone,omitempty"`
}

type AwsAmazonMqBrokerUsersDetails struct {
	PendingChange *string `json:"PendingChange,omitempty"`
	Username *string `json:"Username,omitempty"`
}

type AwsApiCallAction struct {
	AffectedResources map[string]string `json:"AffectedResources,omitempty"`
	Api *string `json:"Api,omitempty"`
	CallerType *string `json:"CallerType,omitempty"`
	DomainDetails *AwsApiCallActionDomainDetails `json:"DomainDetails,omitempty"`
	FirstSeen *string `json:"FirstSeen,omitempty"`
	LastSeen *string `json:"LastSeen,omitempty"`
	RemoteIpDetails *ActionRemoteIpDetails `json:"RemoteIpDetails,omitempty"`
	ServiceName *string `json:"ServiceName,omitempty"`
}

type AwsApiCallActionDomainDetails struct {
	Domain *string `json:"Domain,omitempty"`
}

type AwsApiGatewayAccessLogSettings struct {
	DestinationArn *string `json:"DestinationArn,omitempty"`
	Format *string `json:"Format,omitempty"`
}

type AwsApiGatewayCanarySettings struct {
	DeploymentId *string `json:"DeploymentId,omitempty"`
	PercentTraffic float64 `json:"PercentTraffic,omitempty"`
	StageVariableOverrides map[string]string `json:"StageVariableOverrides,omitempty"`
	UseStageCache bool `json:"UseStageCache,omitempty"`
}

type AwsApiGatewayEndpointConfiguration struct {
	Types []string `json:"Types,omitempty"`
}

type AwsApiGatewayMethodSettings struct {
	CacheDataEncrypted bool `json:"CacheDataEncrypted,omitempty"`
	CacheTtlInSeconds int `json:"CacheTtlInSeconds,omitempty"`
	CachingEnabled bool `json:"CachingEnabled,omitempty"`
	DataTraceEnabled bool `json:"DataTraceEnabled,omitempty"`
	HttpMethod *string `json:"HttpMethod,omitempty"`
	LoggingLevel *string `json:"LoggingLevel,omitempty"`
	MetricsEnabled bool `json:"MetricsEnabled,omitempty"`
	RequireAuthorizationForCacheControl bool `json:"RequireAuthorizationForCacheControl,omitempty"`
	ResourcePath *string `json:"ResourcePath,omitempty"`
	ThrottlingBurstLimit int `json:"ThrottlingBurstLimit,omitempty"`
	ThrottlingRateLimit float64 `json:"ThrottlingRateLimit,omitempty"`
	UnauthorizedCacheControlHeaderStrategy *string `json:"UnauthorizedCacheControlHeaderStrategy,omitempty"`
}

type AwsApiGatewayRestApiDetails struct {
	ApiKeySource *string `json:"ApiKeySource,omitempty"`
	BinaryMediaTypes []string `json:"BinaryMediaTypes,omitempty"`
	CreatedDate *string `json:"CreatedDate,omitempty"`
	Description *string `json:"Description,omitempty"`
	EndpointConfiguration *AwsApiGatewayEndpointConfiguration `json:"EndpointConfiguration,omitempty"`
	Id *string `json:"Id,omitempty"`
	MinimumCompressionSize int `json:"MinimumCompressionSize,omitempty"`
	Name *string `json:"Name,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type AwsApiGatewayStageDetails struct {
	AccessLogSettings *AwsApiGatewayAccessLogSettings `json:"AccessLogSettings,omitempty"`
	CacheClusterEnabled bool `json:"CacheClusterEnabled,omitempty"`
	CacheClusterSize *string `json:"CacheClusterSize,omitempty"`
	CacheClusterStatus *string `json:"CacheClusterStatus,omitempty"`
	CanarySettings *AwsApiGatewayCanarySettings `json:"CanarySettings,omitempty"`
	ClientCertificateId *string `json:"ClientCertificateId,omitempty"`
	CreatedDate *string `json:"CreatedDate,omitempty"`
	DeploymentId *string `json:"DeploymentId,omitempty"`
	Description *string `json:"Description,omitempty"`
	DocumentationVersion *string `json:"DocumentationVersion,omitempty"`
	LastUpdatedDate *string `json:"LastUpdatedDate,omitempty"`
	MethodSettings []AwsApiGatewayMethodSettings `json:"MethodSettings,omitempty"`
	StageName *string `json:"StageName,omitempty"`
	TracingEnabled bool `json:"TracingEnabled,omitempty"`
	Variables map[string]string `json:"Variables,omitempty"`
	WebAclArn *string `json:"WebAclArn,omitempty"`
}

type AwsApiGatewayV2ApiDetails struct {
	ApiEndpoint *string `json:"ApiEndpoint,omitempty"`
	ApiId *string `json:"ApiId,omitempty"`
	ApiKeySelectionExpression *string `json:"ApiKeySelectionExpression,omitempty"`
	CorsConfiguration *AwsCorsConfiguration `json:"CorsConfiguration,omitempty"`
	CreatedDate *string `json:"CreatedDate,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name *string `json:"Name,omitempty"`
	ProtocolType *string `json:"ProtocolType,omitempty"`
	RouteSelectionExpression *string `json:"RouteSelectionExpression,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type AwsApiGatewayV2RouteSettings struct {
	DataTraceEnabled bool `json:"DataTraceEnabled,omitempty"`
	DetailedMetricsEnabled bool `json:"DetailedMetricsEnabled,omitempty"`
	LoggingLevel *string `json:"LoggingLevel,omitempty"`
	ThrottlingBurstLimit int `json:"ThrottlingBurstLimit,omitempty"`
	ThrottlingRateLimit float64 `json:"ThrottlingRateLimit,omitempty"`
}

type AwsApiGatewayV2StageDetails struct {
	AccessLogSettings *AwsApiGatewayAccessLogSettings `json:"AccessLogSettings,omitempty"`
	ApiGatewayManaged bool `json:"ApiGatewayManaged,omitempty"`
	AutoDeploy bool `json:"AutoDeploy,omitempty"`
	ClientCertificateId *string `json:"ClientCertificateId,omitempty"`
	CreatedDate *string `json:"CreatedDate,omitempty"`
	DefaultRouteSettings *AwsApiGatewayV2RouteSettings `json:"DefaultRouteSettings,omitempty"`
	DeploymentId *string `json:"DeploymentId,omitempty"`
	Description *string `json:"Description,omitempty"`
	LastDeploymentStatusMessage *string `json:"LastDeploymentStatusMessage,omitempty"`
	LastUpdatedDate *string `json:"LastUpdatedDate,omitempty"`
	RouteSettings *AwsApiGatewayV2RouteSettings `json:"RouteSettings,omitempty"`
	StageName *string `json:"StageName,omitempty"`
	StageVariables map[string]string `json:"StageVariables,omitempty"`
}

type AwsAppSyncGraphQlApiAdditionalAuthenticationProvidersDetails struct {
	AuthenticationType *string `json:"AuthenticationType,omitempty"`
	LambdaAuthorizerConfig *AwsAppSyncGraphQlApiLambdaAuthorizerConfigDetails `json:"LambdaAuthorizerConfig,omitempty"`
	OpenIdConnectConfig *AwsAppSyncGraphQlApiOpenIdConnectConfigDetails `json:"OpenIdConnectConfig,omitempty"`
	UserPoolConfig *AwsAppSyncGraphQlApiUserPoolConfigDetails `json:"UserPoolConfig,omitempty"`
}

type AwsAppSyncGraphQlApiDetails struct {
	AdditionalAuthenticationProviders []AwsAppSyncGraphQlApiAdditionalAuthenticationProvidersDetails `json:"AdditionalAuthenticationProviders,omitempty"`
	ApiId *string `json:"ApiId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	AuthenticationType *string `json:"AuthenticationType,omitempty"`
	Id *string `json:"Id,omitempty"`
	LambdaAuthorizerConfig *AwsAppSyncGraphQlApiLambdaAuthorizerConfigDetails `json:"LambdaAuthorizerConfig,omitempty"`
	LogConfig *AwsAppSyncGraphQlApiLogConfigDetails `json:"LogConfig,omitempty"`
	Name *string `json:"Name,omitempty"`
	OpenIdConnectConfig *AwsAppSyncGraphQlApiOpenIdConnectConfigDetails `json:"OpenIdConnectConfig,omitempty"`
	UserPoolConfig *AwsAppSyncGraphQlApiUserPoolConfigDetails `json:"UserPoolConfig,omitempty"`
	WafWebAclArn *string `json:"WafWebAclArn,omitempty"`
	XrayEnabled bool `json:"XrayEnabled,omitempty"`
}

type AwsAppSyncGraphQlApiLambdaAuthorizerConfigDetails struct {
	AuthorizerResultTtlInSeconds int `json:"AuthorizerResultTtlInSeconds,omitempty"`
	AuthorizerUri *string `json:"AuthorizerUri,omitempty"`
	IdentityValidationExpression *string `json:"IdentityValidationExpression,omitempty"`
}

type AwsAppSyncGraphQlApiLogConfigDetails struct {
	CloudWatchLogsRoleArn *string `json:"CloudWatchLogsRoleArn,omitempty"`
	ExcludeVerboseContent bool `json:"ExcludeVerboseContent,omitempty"`
	FieldLogLevel *string `json:"FieldLogLevel,omitempty"`
}

type AwsAppSyncGraphQlApiOpenIdConnectConfigDetails struct {
	AuthTtL int64 `json:"AuthTtL,omitempty"`
	ClientId *string `json:"ClientId,omitempty"`
	IatTtL int64 `json:"IatTtL,omitempty"`
	Issuer *string `json:"Issuer,omitempty"`
}

type AwsAppSyncGraphQlApiUserPoolConfigDetails struct {
	AppIdClientRegex *string `json:"AppIdClientRegex,omitempty"`
	AwsRegion *string `json:"AwsRegion,omitempty"`
	DefaultAction *string `json:"DefaultAction,omitempty"`
	UserPoolId *string `json:"UserPoolId,omitempty"`
}

type AwsAthenaWorkGroupConfigurationDetails struct {
	ResultConfiguration *AwsAthenaWorkGroupConfigurationResultConfigurationDetails `json:"ResultConfiguration,omitempty"`
}

type AwsAthenaWorkGroupConfigurationResultConfigurationDetails struct {
	EncryptionConfiguration *AwsAthenaWorkGroupConfigurationResultConfigurationEncryptionConfigurationDetails `json:"EncryptionConfiguration,omitempty"`
}

type AwsAthenaWorkGroupConfigurationResultConfigurationEncryptionConfigurationDetails struct {
	EncryptionOption *string `json:"EncryptionOption,omitempty"`
	KmsKey *string `json:"KmsKey,omitempty"`
}

type AwsAthenaWorkGroupDetails struct {
	Configuration *AwsAthenaWorkGroupConfigurationDetails `json:"Configuration,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name *string `json:"Name,omitempty"`
	State *string `json:"State,omitempty"`
}

type AwsAutoScalingAutoScalingGroupAvailabilityZonesListDetails struct {
	Value *string `json:"Value,omitempty"`
}

type AwsAutoScalingAutoScalingGroupDetails struct {
	AvailabilityZones []AwsAutoScalingAutoScalingGroupAvailabilityZonesListDetails `json:"AvailabilityZones,omitempty"`
	CapacityRebalance bool `json:"CapacityRebalance,omitempty"`
	CreatedTime *string `json:"CreatedTime,omitempty"`
	HealthCheckGracePeriod int `json:"HealthCheckGracePeriod,omitempty"`
	HealthCheckType *string `json:"HealthCheckType,omitempty"`
	LaunchConfigurationName *string `json:"LaunchConfigurationName,omitempty"`
	LaunchTemplate *AwsAutoScalingAutoScalingGroupLaunchTemplateLaunchTemplateSpecification `json:"LaunchTemplate,omitempty"`
	LoadBalancerNames []string `json:"LoadBalancerNames,omitempty"`
	MixedInstancesPolicy *AwsAutoScalingAutoScalingGroupMixedInstancesPolicyDetails `json:"MixedInstancesPolicy,omitempty"`
}

type AwsAutoScalingAutoScalingGroupLaunchTemplateLaunchTemplateSpecification struct {
	LaunchTemplateId *string `json:"LaunchTemplateId,omitempty"`
	LaunchTemplateName *string `json:"LaunchTemplateName,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type AwsAutoScalingAutoScalingGroupMixedInstancesPolicyDetails struct {
	InstancesDistribution *AwsAutoScalingAutoScalingGroupMixedInstancesPolicyInstancesDistributionDetails `json:"InstancesDistribution,omitempty"`
	LaunchTemplate *AwsAutoScalingAutoScalingGroupMixedInstancesPolicyLaunchTemplateDetails `json:"LaunchTemplate,omitempty"`
}

type AwsAutoScalingAutoScalingGroupMixedInstancesPolicyInstancesDistributionDetails struct {
	OnDemandAllocationStrategy *string `json:"OnDemandAllocationStrategy,omitempty"`
	OnDemandBaseCapacity int `json:"OnDemandBaseCapacity,omitempty"`
	OnDemandPercentageAboveBaseCapacity int `json:"OnDemandPercentageAboveBaseCapacity,omitempty"`
	SpotAllocationStrategy *string `json:"SpotAllocationStrategy,omitempty"`
	SpotInstancePools int `json:"SpotInstancePools,omitempty"`
	SpotMaxPrice *string `json:"SpotMaxPrice,omitempty"`
}

type AwsAutoScalingAutoScalingGroupMixedInstancesPolicyLaunchTemplateDetails struct {
	LaunchTemplateSpecification *AwsAutoScalingAutoScalingGroupMixedInstancesPolicyLaunchTemplateLaunchTemplateSpecification `json:"LaunchTemplateSpecification,omitempty"`
	Overrides []AwsAutoScalingAutoScalingGroupMixedInstancesPolicyLaunchTemplateOverridesListDetails `json:"Overrides,omitempty"`
}

type AwsAutoScalingAutoScalingGroupMixedInstancesPolicyLaunchTemplateLaunchTemplateSpecification struct {
	LaunchTemplateId *string `json:"LaunchTemplateId,omitempty"`
	LaunchTemplateName *string `json:"LaunchTemplateName,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type AwsAutoScalingAutoScalingGroupMixedInstancesPolicyLaunchTemplateOverridesListDetails struct {
	InstanceType *string `json:"InstanceType,omitempty"`
	WeightedCapacity *string `json:"WeightedCapacity,omitempty"`
}

type AwsAutoScalingLaunchConfigurationBlockDeviceMappingsDetails struct {
	DeviceName *string `json:"DeviceName,omitempty"`
	Ebs *AwsAutoScalingLaunchConfigurationBlockDeviceMappingsEbsDetails `json:"Ebs,omitempty"`
	NoDevice bool `json:"NoDevice,omitempty"`
	VirtualName *string `json:"VirtualName,omitempty"`
}

type AwsAutoScalingLaunchConfigurationBlockDeviceMappingsEbsDetails struct {
	DeleteOnTermination bool `json:"DeleteOnTermination,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	Iops int `json:"Iops,omitempty"`
	SnapshotId *string `json:"SnapshotId,omitempty"`
	VolumeSize int `json:"VolumeSize,omitempty"`
	VolumeType *string `json:"VolumeType,omitempty"`
}

type AwsAutoScalingLaunchConfigurationDetails struct {
	AssociatePublicIpAddress bool `json:"AssociatePublicIpAddress,omitempty"`
	BlockDeviceMappings []AwsAutoScalingLaunchConfigurationBlockDeviceMappingsDetails `json:"BlockDeviceMappings,omitempty"`
	ClassicLinkVpcId *string `json:"ClassicLinkVpcId,omitempty"`
	ClassicLinkVpcSecurityGroups []string `json:"ClassicLinkVpcSecurityGroups,omitempty"`
	CreatedTime *string `json:"CreatedTime,omitempty"`
	EbsOptimized bool `json:"EbsOptimized,omitempty"`
	IamInstanceProfile *string `json:"IamInstanceProfile,omitempty"`
	ImageId *string `json:"ImageId,omitempty"`
	InstanceMonitoring *AwsAutoScalingLaunchConfigurationInstanceMonitoringDetails `json:"InstanceMonitoring,omitempty"`
	InstanceType *string `json:"InstanceType,omitempty"`
	KernelId *string `json:"KernelId,omitempty"`
	KeyName *string `json:"KeyName,omitempty"`
	LaunchConfigurationName *string `json:"LaunchConfigurationName,omitempty"`
	MetadataOptions *AwsAutoScalingLaunchConfigurationMetadataOptions `json:"MetadataOptions,omitempty"`
	PlacementTenancy *string `json:"PlacementTenancy,omitempty"`
	RamdiskId *string `json:"RamdiskId,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	SpotPrice *string `json:"SpotPrice,omitempty"`
	UserData *string `json:"UserData,omitempty"`
}

type AwsAutoScalingLaunchConfigurationInstanceMonitoringDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsAutoScalingLaunchConfigurationMetadataOptions struct {
	HttpEndpoint *string `json:"HttpEndpoint,omitempty"`
	HttpPutResponseHopLimit int `json:"HttpPutResponseHopLimit,omitempty"`
	HttpTokens *string `json:"HttpTokens,omitempty"`
}

type AwsBackupBackupPlanAdvancedBackupSettingsDetails struct {
	BackupOptions map[string]string `json:"BackupOptions,omitempty"`
	ResourceType *string `json:"ResourceType,omitempty"`
}

type AwsBackupBackupPlanBackupPlanDetails struct {
	AdvancedBackupSettings []AwsBackupBackupPlanAdvancedBackupSettingsDetails `json:"AdvancedBackupSettings,omitempty"`
	BackupPlanName *string `json:"BackupPlanName,omitempty"`
	BackupPlanRule []AwsBackupBackupPlanRuleDetails `json:"BackupPlanRule,omitempty"`
}

type AwsBackupBackupPlanDetails struct {
	BackupPlan *AwsBackupBackupPlanBackupPlanDetails `json:"BackupPlan,omitempty"`
	BackupPlanArn *string `json:"BackupPlanArn,omitempty"`
	BackupPlanId *string `json:"BackupPlanId,omitempty"`
	VersionId *string `json:"VersionId,omitempty"`
}

type AwsBackupBackupPlanLifecycleDetails struct {
	DeleteAfterDays int64 `json:"DeleteAfterDays,omitempty"`
	MoveToColdStorageAfterDays int64 `json:"MoveToColdStorageAfterDays,omitempty"`
}

type AwsBackupBackupPlanRuleCopyActionsDetails struct {
	DestinationBackupVaultArn *string `json:"DestinationBackupVaultArn,omitempty"`
	Lifecycle *AwsBackupBackupPlanLifecycleDetails `json:"Lifecycle,omitempty"`
}

type AwsBackupBackupPlanRuleDetails struct {
	CompletionWindowMinutes int64 `json:"CompletionWindowMinutes,omitempty"`
	CopyActions []AwsBackupBackupPlanRuleCopyActionsDetails `json:"CopyActions,omitempty"`
	EnableContinuousBackup bool `json:"EnableContinuousBackup,omitempty"`
	Lifecycle *AwsBackupBackupPlanLifecycleDetails `json:"Lifecycle,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	ScheduleExpression *string `json:"ScheduleExpression,omitempty"`
	StartWindowMinutes int64 `json:"StartWindowMinutes,omitempty"`
	TargetBackupVault *string `json:"TargetBackupVault,omitempty"`
}

type AwsBackupBackupVaultDetails struct {
	AccessPolicy *string `json:"AccessPolicy,omitempty"`
	BackupVaultArn *string `json:"BackupVaultArn,omitempty"`
	BackupVaultName *string `json:"BackupVaultName,omitempty"`
	EncryptionKeyArn *string `json:"EncryptionKeyArn,omitempty"`
	Notifications *AwsBackupBackupVaultNotificationsDetails `json:"Notifications,omitempty"`
}

type AwsBackupBackupVaultNotificationsDetails struct {
	BackupVaultEvents []string `json:"BackupVaultEvents,omitempty"`
	SnsTopicArn *string `json:"SnsTopicArn,omitempty"`
}

type AwsBackupRecoveryPointCalculatedLifecycleDetails struct {
	DeleteAt *string `json:"DeleteAt,omitempty"`
	MoveToColdStorageAt *string `json:"MoveToColdStorageAt,omitempty"`
}

type AwsBackupRecoveryPointCreatedByDetails struct {
	BackupPlanArn *string `json:"BackupPlanArn,omitempty"`
	BackupPlanId *string `json:"BackupPlanId,omitempty"`
	BackupPlanVersion *string `json:"BackupPlanVersion,omitempty"`
	BackupRuleId *string `json:"BackupRuleId,omitempty"`
}

type AwsBackupRecoveryPointDetails struct {
	BackupSizeInBytes int64 `json:"BackupSizeInBytes,omitempty"`
	BackupVaultArn *string `json:"BackupVaultArn,omitempty"`
	BackupVaultName *string `json:"BackupVaultName,omitempty"`
	CalculatedLifecycle *AwsBackupRecoveryPointCalculatedLifecycleDetails `json:"CalculatedLifecycle,omitempty"`
	CompletionDate *string `json:"CompletionDate,omitempty"`
	CreatedBy *AwsBackupRecoveryPointCreatedByDetails `json:"CreatedBy,omitempty"`
	CreationDate *string `json:"CreationDate,omitempty"`
	EncryptionKeyArn *string `json:"EncryptionKeyArn,omitempty"`
	IamRoleArn *string `json:"IamRoleArn,omitempty"`
	IsEncrypted bool `json:"IsEncrypted,omitempty"`
	LastRestoreTime *string `json:"LastRestoreTime,omitempty"`
	Lifecycle *AwsBackupRecoveryPointLifecycleDetails `json:"Lifecycle,omitempty"`
	RecoveryPointArn *string `json:"RecoveryPointArn,omitempty"`
	ResourceArn *string `json:"ResourceArn,omitempty"`
	ResourceType *string `json:"ResourceType,omitempty"`
	SourceBackupVaultArn *string `json:"SourceBackupVaultArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
	StorageClass *string `json:"StorageClass,omitempty"`
}

type AwsBackupRecoveryPointLifecycleDetails struct {
	DeleteAfterDays int64 `json:"DeleteAfterDays,omitempty"`
	MoveToColdStorageAfterDays int64 `json:"MoveToColdStorageAfterDays,omitempty"`
}

type AwsCertificateManagerCertificateDetails struct {
	CertificateAuthorityArn *string `json:"CertificateAuthorityArn,omitempty"`
	CreatedAt *string `json:"CreatedAt,omitempty"`
	DomainName *string `json:"DomainName,omitempty"`
	DomainValidationOptions []AwsCertificateManagerCertificateDomainValidationOption `json:"DomainValidationOptions,omitempty"`
	ExtendedKeyUsages []AwsCertificateManagerCertificateExtendedKeyUsage `json:"ExtendedKeyUsages,omitempty"`
	FailureReason *string `json:"FailureReason,omitempty"`
	ImportedAt *string `json:"ImportedAt,omitempty"`
	InUseBy []string `json:"InUseBy,omitempty"`
	IssuedAt *string `json:"IssuedAt,omitempty"`
	Issuer *string `json:"Issuer,omitempty"`
	KeyAlgorithm *string `json:"KeyAlgorithm,omitempty"`
	KeyUsages []AwsCertificateManagerCertificateKeyUsage `json:"KeyUsages,omitempty"`
	NotAfter *string `json:"NotAfter,omitempty"`
	NotBefore *string `json:"NotBefore,omitempty"`
	Options *AwsCertificateManagerCertificateOptions `json:"Options,omitempty"`
	RenewalEligibility *string `json:"RenewalEligibility,omitempty"`
	RenewalSummary *AwsCertificateManagerCertificateRenewalSummary `json:"RenewalSummary,omitempty"`
	Serial *string `json:"Serial,omitempty"`
	SignatureAlgorithm *string `json:"SignatureAlgorithm,omitempty"`
	Status *string `json:"Status,omitempty"`
	Subject *string `json:"Subject,omitempty"`
	SubjectAlternativeNames []string `json:"SubjectAlternativeNames,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsCertificateManagerCertificateDomainValidationOption struct {
	DomainName *string `json:"DomainName,omitempty"`
	ResourceRecord *AwsCertificateManagerCertificateResourceRecord `json:"ResourceRecord,omitempty"`
	ValidationDomain *string `json:"ValidationDomain,omitempty"`
	ValidationEmails []string `json:"ValidationEmails,omitempty"`
	ValidationMethod *string `json:"ValidationMethod,omitempty"`
	ValidationStatus *string `json:"ValidationStatus,omitempty"`
}

type AwsCertificateManagerCertificateExtendedKeyUsage struct {
	Name *string `json:"Name,omitempty"`
	OId *string `json:"OId,omitempty"`
}

type AwsCertificateManagerCertificateKeyUsage struct {
	Name *string `json:"Name,omitempty"`
}

type AwsCertificateManagerCertificateOptions struct {
	CertificateTransparencyLoggingPreference *string `json:"CertificateTransparencyLoggingPreference,omitempty"`
}

type AwsCertificateManagerCertificateRenewalSummary struct {
	DomainValidationOptions []AwsCertificateManagerCertificateDomainValidationOption `json:"DomainValidationOptions,omitempty"`
	RenewalStatus *string `json:"RenewalStatus,omitempty"`
	RenewalStatusReason *string `json:"RenewalStatusReason,omitempty"`
	UpdatedAt *string `json:"UpdatedAt,omitempty"`
}

type AwsCertificateManagerCertificateResourceRecord struct {
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsCloudFormationStackDetails struct {
	Capabilities []string `json:"Capabilities,omitempty"`
	CreationTime *string `json:"CreationTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	DisableRollback bool `json:"DisableRollback,omitempty"`
	DriftInformation *AwsCloudFormationStackDriftInformationDetails `json:"DriftInformation,omitempty"`
	EnableTerminationProtection bool `json:"EnableTerminationProtection,omitempty"`
	LastUpdatedTime *string `json:"LastUpdatedTime,omitempty"`
	NotificationArns []string `json:"NotificationArns,omitempty"`
	Outputs []AwsCloudFormationStackOutputsDetails `json:"Outputs,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	StackId *string `json:"StackId,omitempty"`
	StackName *string `json:"StackName,omitempty"`
	StackStatus *string `json:"StackStatus,omitempty"`
	StackStatusReason *string `json:"StackStatusReason,omitempty"`
	TimeoutInMinutes int `json:"TimeoutInMinutes,omitempty"`
}

type AwsCloudFormationStackDriftInformationDetails struct {
	StackDriftStatus *string `json:"StackDriftStatus,omitempty"`
}

type AwsCloudFormationStackOutputsDetails struct {
	Description *string `json:"Description,omitempty"`
	OutputKey *string `json:"OutputKey,omitempty"`
	OutputValue *string `json:"OutputValue,omitempty"`
}

type AwsCloudFrontDistributionCacheBehavior struct {
	ViewerProtocolPolicy *string `json:"ViewerProtocolPolicy,omitempty"`
}

type AwsCloudFrontDistributionCacheBehaviors struct {
	Items []AwsCloudFrontDistributionCacheBehavior `json:"Items,omitempty"`
}

type AwsCloudFrontDistributionDefaultCacheBehavior struct {
	ViewerProtocolPolicy *string `json:"ViewerProtocolPolicy,omitempty"`
}

type AwsCloudFrontDistributionDetails struct {
	CacheBehaviors *AwsCloudFrontDistributionCacheBehaviors `json:"CacheBehaviors,omitempty"`
	DefaultCacheBehavior *AwsCloudFrontDistributionDefaultCacheBehavior `json:"DefaultCacheBehavior,omitempty"`
	DefaultRootObject *string `json:"DefaultRootObject,omitempty"`
	DomainName *string `json:"DomainName,omitempty"`
	ETag *string `json:"ETag,omitempty"`
	LastModifiedTime *string `json:"LastModifiedTime,omitempty"`
	Logging *AwsCloudFrontDistributionLogging `json:"Logging,omitempty"`
	OriginGroups *AwsCloudFrontDistributionOriginGroups `json:"OriginGroups,omitempty"`
	Origins *AwsCloudFrontDistributionOrigins `json:"Origins,omitempty"`
	Status *string `json:"Status,omitempty"`
	ViewerCertificate *AwsCloudFrontDistributionViewerCertificate `json:"ViewerCertificate,omitempty"`
	WebAclId *string `json:"WebAclId,omitempty"`
}

type AwsCloudFrontDistributionLogging struct {
	Bucket *string `json:"Bucket,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	IncludeCookies bool `json:"IncludeCookies,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
}

type AwsCloudFrontDistributionOriginCustomOriginConfig struct {
	HttpPort int `json:"HttpPort,omitempty"`
	HttpsPort int `json:"HttpsPort,omitempty"`
	OriginKeepaliveTimeout int `json:"OriginKeepaliveTimeout,omitempty"`
	OriginProtocolPolicy *string `json:"OriginProtocolPolicy,omitempty"`
	OriginReadTimeout int `json:"OriginReadTimeout,omitempty"`
	OriginSslProtocols *AwsCloudFrontDistributionOriginSslProtocols `json:"OriginSslProtocols,omitempty"`
}

type AwsCloudFrontDistributionOriginGroup struct {
	FailoverCriteria *AwsCloudFrontDistributionOriginGroupFailover `json:"FailoverCriteria,omitempty"`
}

type AwsCloudFrontDistributionOriginGroupFailover struct {
	StatusCodes *AwsCloudFrontDistributionOriginGroupFailoverStatusCodes `json:"StatusCodes,omitempty"`
}

type AwsCloudFrontDistributionOriginGroupFailoverStatusCodes struct {
	Items []int `json:"Items,omitempty"`
	Quantity int `json:"Quantity,omitempty"`
}

type AwsCloudFrontDistributionOriginGroups struct {
	Items []AwsCloudFrontDistributionOriginGroup `json:"Items,omitempty"`
}

type AwsCloudFrontDistributionOriginItem struct {
	CustomOriginConfig *AwsCloudFrontDistributionOriginCustomOriginConfig `json:"CustomOriginConfig,omitempty"`
	DomainName *string `json:"DomainName,omitempty"`
	Id *string `json:"Id,omitempty"`
	OriginPath *string `json:"OriginPath,omitempty"`
	S3OriginConfig *AwsCloudFrontDistributionOriginS3OriginConfig `json:"S3OriginConfig,omitempty"`
}

type AwsCloudFrontDistributionOriginS3OriginConfig struct {
	OriginAccessIdentity *string `json:"OriginAccessIdentity,omitempty"`
}

type AwsCloudFrontDistributionOriginSslProtocols struct {
	Items []string `json:"Items,omitempty"`
	Quantity int `json:"Quantity,omitempty"`
}

type AwsCloudFrontDistributionOrigins struct {
	Items []AwsCloudFrontDistributionOriginItem `json:"Items,omitempty"`
}

type AwsCloudFrontDistributionViewerCertificate struct {
	AcmCertificateArn *string `json:"AcmCertificateArn,omitempty"`
	Certificate *string `json:"Certificate,omitempty"`
	CertificateSource *string `json:"CertificateSource,omitempty"`
	CloudFrontDefaultCertificate bool `json:"CloudFrontDefaultCertificate,omitempty"`
	IamCertificateId *string `json:"IamCertificateId,omitempty"`
	MinimumProtocolVersion *string `json:"MinimumProtocolVersion,omitempty"`
	SslSupportMethod *string `json:"SslSupportMethod,omitempty"`
}

type AwsCloudTrailTrailDetails struct {
	CloudWatchLogsLogGroupArn *string `json:"CloudWatchLogsLogGroupArn,omitempty"`
	CloudWatchLogsRoleArn *string `json:"CloudWatchLogsRoleArn,omitempty"`
	HasCustomEventSelectors bool `json:"HasCustomEventSelectors,omitempty"`
	HomeRegion *string `json:"HomeRegion,omitempty"`
	IncludeGlobalServiceEvents bool `json:"IncludeGlobalServiceEvents,omitempty"`
	IsMultiRegionTrail bool `json:"IsMultiRegionTrail,omitempty"`
	IsOrganizationTrail bool `json:"IsOrganizationTrail,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LogFileValidationEnabled bool `json:"LogFileValidationEnabled,omitempty"`
	Name *string `json:"Name,omitempty"`
	S3BucketName *string `json:"S3BucketName,omitempty"`
	S3KeyPrefix *string `json:"S3KeyPrefix,omitempty"`
	SnsTopicArn *string `json:"SnsTopicArn,omitempty"`
	SnsTopicName *string `json:"SnsTopicName,omitempty"`
	TrailArn *string `json:"TrailArn,omitempty"`
}

type AwsCloudWatchAlarmDetails struct {
	ActionsEnabled bool `json:"ActionsEnabled,omitempty"`
	AlarmActions []string `json:"AlarmActions,omitempty"`
	AlarmArn *string `json:"AlarmArn,omitempty"`
	AlarmConfigurationUpdatedTimestamp *string `json:"AlarmConfigurationUpdatedTimestamp,omitempty"`
	AlarmDescription *string `json:"AlarmDescription,omitempty"`
	AlarmName *string `json:"AlarmName,omitempty"`
	ComparisonOperator *string `json:"ComparisonOperator,omitempty"`
	DatapointsToAlarm int `json:"DatapointsToAlarm,omitempty"`
	Dimensions []AwsCloudWatchAlarmDimensionsDetails `json:"Dimensions,omitempty"`
	EvaluateLowSampleCountPercentile *string `json:"EvaluateLowSampleCountPercentile,omitempty"`
	EvaluationPeriods int `json:"EvaluationPeriods,omitempty"`
	ExtendedStatistic *string `json:"ExtendedStatistic,omitempty"`
	InsufficientDataActions []string `json:"InsufficientDataActions,omitempty"`
	MetricName *string `json:"MetricName,omitempty"`
	Namespace *string `json:"Namespace,omitempty"`
	OkActions []string `json:"OkActions,omitempty"`
	Period int `json:"Period,omitempty"`
	Statistic *string `json:"Statistic,omitempty"`
	Threshold float64 `json:"Threshold,omitempty"`
	ThresholdMetricId *string `json:"ThresholdMetricId,omitempty"`
	TreatMissingData *string `json:"TreatMissingData,omitempty"`
	Unit *string `json:"Unit,omitempty"`
}

type AwsCloudWatchAlarmDimensionsDetails struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsCodeBuildProjectArtifactsDetails struct {
	ArtifactIdentifier *string `json:"ArtifactIdentifier,omitempty"`
	EncryptionDisabled bool `json:"EncryptionDisabled,omitempty"`
	Location *string `json:"Location,omitempty"`
	Name *string `json:"Name,omitempty"`
	NamespaceType *string `json:"NamespaceType,omitempty"`
	OverrideArtifactName bool `json:"OverrideArtifactName,omitempty"`
	Packaging *string `json:"Packaging,omitempty"`
	Path *string `json:"Path,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsCodeBuildProjectDetails struct {
	Artifacts []AwsCodeBuildProjectArtifactsDetails `json:"Artifacts,omitempty"`
	EncryptionKey *string `json:"EncryptionKey,omitempty"`
	Environment *AwsCodeBuildProjectEnvironment `json:"Environment,omitempty"`
	LogsConfig *AwsCodeBuildProjectLogsConfigDetails `json:"LogsConfig,omitempty"`
	Name *string `json:"Name,omitempty"`
	SecondaryArtifacts []AwsCodeBuildProjectArtifactsDetails `json:"SecondaryArtifacts,omitempty"`
	ServiceRole *string `json:"ServiceRole,omitempty"`
	Source *AwsCodeBuildProjectSource `json:"Source,omitempty"`
	VpcConfig *AwsCodeBuildProjectVpcConfig `json:"VpcConfig,omitempty"`
}

type AwsCodeBuildProjectEnvironment struct {
	Certificate *string `json:"Certificate,omitempty"`
	EnvironmentVariables []AwsCodeBuildProjectEnvironmentEnvironmentVariablesDetails `json:"EnvironmentVariables,omitempty"`
	ImagePullCredentialsType *string `json:"ImagePullCredentialsType,omitempty"`
	PrivilegedMode bool `json:"PrivilegedMode,omitempty"`
	RegistryCredential *AwsCodeBuildProjectEnvironmentRegistryCredential `json:"RegistryCredential,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsCodeBuildProjectEnvironmentEnvironmentVariablesDetails struct {
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsCodeBuildProjectEnvironmentRegistryCredential struct {
	Credential *string `json:"Credential,omitempty"`
	CredentialProvider *string `json:"CredentialProvider,omitempty"`
}

type AwsCodeBuildProjectLogsConfigCloudWatchLogsDetails struct {
	GroupName *string `json:"GroupName,omitempty"`
	Status *string `json:"Status,omitempty"`
	StreamName *string `json:"StreamName,omitempty"`
}

type AwsCodeBuildProjectLogsConfigDetails struct {
	CloudWatchLogs *AwsCodeBuildProjectLogsConfigCloudWatchLogsDetails `json:"CloudWatchLogs,omitempty"`
	S3Logs *AwsCodeBuildProjectLogsConfigS3LogsDetails `json:"S3Logs,omitempty"`
}

type AwsCodeBuildProjectLogsConfigS3LogsDetails struct {
	EncryptionDisabled bool `json:"EncryptionDisabled,omitempty"`
	Location *string `json:"Location,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsCodeBuildProjectSource struct {
	GitCloneDepth int `json:"GitCloneDepth,omitempty"`
	InsecureSsl bool `json:"InsecureSsl,omitempty"`
	Location *string `json:"Location,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsCodeBuildProjectVpcConfig struct {
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	Subnets []string `json:"Subnets,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsCorsConfiguration struct {
	AllowCredentials bool `json:"AllowCredentials,omitempty"`
	AllowHeaders []string `json:"AllowHeaders,omitempty"`
	AllowMethods []string `json:"AllowMethods,omitempty"`
	AllowOrigins []string `json:"AllowOrigins,omitempty"`
	ExposeHeaders []string `json:"ExposeHeaders,omitempty"`
	MaxAge int `json:"MaxAge,omitempty"`
}

type AwsDmsEndpointDetails struct {
	CertificateArn *string `json:"CertificateArn,omitempty"`
	DatabaseName *string `json:"DatabaseName,omitempty"`
	EndpointArn *string `json:"EndpointArn,omitempty"`
	EndpointIdentifier *string `json:"EndpointIdentifier,omitempty"`
	EndpointType *string `json:"EndpointType,omitempty"`
	EngineName *string `json:"EngineName,omitempty"`
	ExternalId *string `json:"ExternalId,omitempty"`
	ExtraConnectionAttributes *string `json:"ExtraConnectionAttributes,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	Port int `json:"Port,omitempty"`
	ServerName *string `json:"ServerName,omitempty"`
	SslMode *string `json:"SslMode,omitempty"`
	Username *string `json:"Username,omitempty"`
}

type AwsDmsReplicationInstanceDetails struct {
	AllocatedStorage int `json:"AllocatedStorage,omitempty"`
	AutoMinorVersionUpgrade bool `json:"AutoMinorVersionUpgrade,omitempty"`
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	MultiAZ bool `json:"MultiAZ,omitempty"`
	PreferredMaintenanceWindow *string `json:"PreferredMaintenanceWindow,omitempty"`
	PubliclyAccessible bool `json:"PubliclyAccessible,omitempty"`
	ReplicationInstanceClass *string `json:"ReplicationInstanceClass,omitempty"`
	ReplicationInstanceIdentifier *string `json:"ReplicationInstanceIdentifier,omitempty"`
	ReplicationSubnetGroup *AwsDmsReplicationInstanceReplicationSubnetGroupDetails `json:"ReplicationSubnetGroup,omitempty"`
	VpcSecurityGroups []AwsDmsReplicationInstanceVpcSecurityGroupsDetails `json:"VpcSecurityGroups,omitempty"`
}

type AwsDmsReplicationInstanceReplicationSubnetGroupDetails struct {
	ReplicationSubnetGroupIdentifier *string `json:"ReplicationSubnetGroupIdentifier,omitempty"`
}

type AwsDmsReplicationInstanceVpcSecurityGroupsDetails struct {
	VpcSecurityGroupId *string `json:"VpcSecurityGroupId,omitempty"`
}

type AwsDmsReplicationTaskDetails struct {
	CdcStartPosition *string `json:"CdcStartPosition,omitempty"`
	CdcStartTime *string `json:"CdcStartTime,omitempty"`
	CdcStopPosition *string `json:"CdcStopPosition,omitempty"`
	Id *string `json:"Id,omitempty"`
	MigrationType *string `json:"MigrationType,omitempty"`
	ReplicationInstanceArn *string `json:"ReplicationInstanceArn,omitempty"`
	ReplicationTaskIdentifier *string `json:"ReplicationTaskIdentifier,omitempty"`
	ReplicationTaskSettings *string `json:"ReplicationTaskSettings,omitempty"`
	ResourceIdentifier *string `json:"ResourceIdentifier,omitempty"`
	SourceEndpointArn *string `json:"SourceEndpointArn,omitempty"`
	TableMappings *string `json:"TableMappings,omitempty"`
	TargetEndpointArn *string `json:"TargetEndpointArn,omitempty"`
	TaskData *string `json:"TaskData,omitempty"`
}

type AwsDynamoDbTableAttributeDefinition struct {
	AttributeName *string `json:"AttributeName,omitempty"`
	AttributeType *string `json:"AttributeType,omitempty"`
}

type AwsDynamoDbTableBillingModeSummary struct {
	BillingMode *string `json:"BillingMode,omitempty"`
	LastUpdateToPayPerRequestDateTime *string `json:"LastUpdateToPayPerRequestDateTime,omitempty"`
}

type AwsDynamoDbTableDetails struct {
	AttributeDefinitions []AwsDynamoDbTableAttributeDefinition `json:"AttributeDefinitions,omitempty"`
	BillingModeSummary *AwsDynamoDbTableBillingModeSummary `json:"BillingModeSummary,omitempty"`
	CreationDateTime *string `json:"CreationDateTime,omitempty"`
	DeletionProtectionEnabled bool `json:"DeletionProtectionEnabled,omitempty"`
	GlobalSecondaryIndexes []AwsDynamoDbTableGlobalSecondaryIndex `json:"GlobalSecondaryIndexes,omitempty"`
	GlobalTableVersion *string `json:"GlobalTableVersion,omitempty"`
	ItemCount int `json:"ItemCount,omitempty"`
	KeySchema []AwsDynamoDbTableKeySchema `json:"KeySchema,omitempty"`
	LatestStreamArn *string `json:"LatestStreamArn,omitempty"`
	LatestStreamLabel *string `json:"LatestStreamLabel,omitempty"`
	LocalSecondaryIndexes []AwsDynamoDbTableLocalSecondaryIndex `json:"LocalSecondaryIndexes,omitempty"`
	ProvisionedThroughput *AwsDynamoDbTableProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
	Replicas []AwsDynamoDbTableReplica `json:"Replicas,omitempty"`
	RestoreSummary *AwsDynamoDbTableRestoreSummary `json:"RestoreSummary,omitempty"`
	SseDescription *AwsDynamoDbTableSseDescription `json:"SseDescription,omitempty"`
	StreamSpecification *AwsDynamoDbTableStreamSpecification `json:"StreamSpecification,omitempty"`
	TableId *string `json:"TableId,omitempty"`
	TableName *string `json:"TableName,omitempty"`
	TableSizeBytes int64 `json:"TableSizeBytes,omitempty"`
	TableStatus *string `json:"TableStatus,omitempty"`
}

type AwsDynamoDbTableGlobalSecondaryIndex struct {
	Backfilling bool `json:"Backfilling,omitempty"`
	IndexArn *string `json:"IndexArn,omitempty"`
	IndexName *string `json:"IndexName,omitempty"`
	IndexSizeBytes int64 `json:"IndexSizeBytes,omitempty"`
	IndexStatus *string `json:"IndexStatus,omitempty"`
	ItemCount int `json:"ItemCount,omitempty"`
	KeySchema []AwsDynamoDbTableKeySchema `json:"KeySchema,omitempty"`
	Projection *AwsDynamoDbTableProjection `json:"Projection,omitempty"`
	ProvisionedThroughput *AwsDynamoDbTableProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
}

type AwsDynamoDbTableKeySchema struct {
	AttributeName *string `json:"AttributeName,omitempty"`
	KeyType *string `json:"KeyType,omitempty"`
}

type AwsDynamoDbTableLocalSecondaryIndex struct {
	IndexArn *string `json:"IndexArn,omitempty"`
	IndexName *string `json:"IndexName,omitempty"`
	KeySchema []AwsDynamoDbTableKeySchema `json:"KeySchema,omitempty"`
	Projection *AwsDynamoDbTableProjection `json:"Projection,omitempty"`
}

type AwsDynamoDbTableProjection struct {
	NonKeyAttributes []string `json:"NonKeyAttributes,omitempty"`
	ProjectionType *string `json:"ProjectionType,omitempty"`
}

type AwsDynamoDbTableProvisionedThroughput struct {
	LastDecreaseDateTime *string `json:"LastDecreaseDateTime,omitempty"`
	LastIncreaseDateTime *string `json:"LastIncreaseDateTime,omitempty"`
	NumberOfDecreasesToday int `json:"NumberOfDecreasesToday,omitempty"`
	ReadCapacityUnits int `json:"ReadCapacityUnits,omitempty"`
	WriteCapacityUnits int `json:"WriteCapacityUnits,omitempty"`
}

type AwsDynamoDbTableProvisionedThroughputOverride struct {
	ReadCapacityUnits int `json:"ReadCapacityUnits,omitempty"`
}

type AwsDynamoDbTableReplica struct {
	GlobalSecondaryIndexes []AwsDynamoDbTableReplicaGlobalSecondaryIndex `json:"GlobalSecondaryIndexes,omitempty"`
	KmsMasterKeyId *string `json:"KmsMasterKeyId,omitempty"`
	ProvisionedThroughputOverride *AwsDynamoDbTableProvisionedThroughputOverride `json:"ProvisionedThroughputOverride,omitempty"`
	RegionName *string `json:"RegionName,omitempty"`
	ReplicaStatus *string `json:"ReplicaStatus,omitempty"`
	ReplicaStatusDescription *string `json:"ReplicaStatusDescription,omitempty"`
}

type AwsDynamoDbTableReplicaGlobalSecondaryIndex struct {
	IndexName *string `json:"IndexName,omitempty"`
	ProvisionedThroughputOverride *AwsDynamoDbTableProvisionedThroughputOverride `json:"ProvisionedThroughputOverride,omitempty"`
}

type AwsDynamoDbTableRestoreSummary struct {
	RestoreDateTime *string `json:"RestoreDateTime,omitempty"`
	RestoreInProgress bool `json:"RestoreInProgress,omitempty"`
	SourceBackupArn *string `json:"SourceBackupArn,omitempty"`
	SourceTableArn *string `json:"SourceTableArn,omitempty"`
}

type AwsDynamoDbTableSseDescription struct {
	InaccessibleEncryptionDateTime *string `json:"InaccessibleEncryptionDateTime,omitempty"`
	KmsMasterKeyArn *string `json:"KmsMasterKeyArn,omitempty"`
	SseType *string `json:"SseType,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsDynamoDbTableStreamSpecification struct {
	StreamEnabled bool `json:"StreamEnabled,omitempty"`
	StreamViewType *string `json:"StreamViewType,omitempty"`
}

type AwsEc2ClientVpnEndpointAuthenticationOptionsActiveDirectoryDetails struct {
	DirectoryId *string `json:"DirectoryId,omitempty"`
}

type AwsEc2ClientVpnEndpointAuthenticationOptionsDetails struct {
	ActiveDirectory *AwsEc2ClientVpnEndpointAuthenticationOptionsActiveDirectoryDetails `json:"ActiveDirectory,omitempty"`
	FederatedAuthentication *AwsEc2ClientVpnEndpointAuthenticationOptionsFederatedAuthenticationDetails `json:"FederatedAuthentication,omitempty"`
	MutualAuthentication *AwsEc2ClientVpnEndpointAuthenticationOptionsMutualAuthenticationDetails `json:"MutualAuthentication,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEc2ClientVpnEndpointAuthenticationOptionsFederatedAuthenticationDetails struct {
	SamlProviderArn *string `json:"SamlProviderArn,omitempty"`
	SelfServiceSamlProviderArn *string `json:"SelfServiceSamlProviderArn,omitempty"`
}

type AwsEc2ClientVpnEndpointAuthenticationOptionsMutualAuthenticationDetails struct {
	ClientRootCertificateChain *string `json:"ClientRootCertificateChain,omitempty"`
}

type AwsEc2ClientVpnEndpointClientConnectOptionsDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
	LambdaFunctionArn *string `json:"LambdaFunctionArn,omitempty"`
	Status *AwsEc2ClientVpnEndpointClientConnectOptionsStatusDetails `json:"Status,omitempty"`
}

type AwsEc2ClientVpnEndpointClientConnectOptionsStatusDetails struct {
	Code *string `json:"Code,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type AwsEc2ClientVpnEndpointClientLoginBannerOptionsDetails struct {
	BannerText *string `json:"BannerText,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsEc2ClientVpnEndpointConnectionLogOptionsDetails struct {
	CloudwatchLogGroup *string `json:"CloudwatchLogGroup,omitempty"`
	CloudwatchLogStream *string `json:"CloudwatchLogStream,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsEc2ClientVpnEndpointDetails struct {
	AuthenticationOptions []AwsEc2ClientVpnEndpointAuthenticationOptionsDetails `json:"AuthenticationOptions,omitempty"`
	ClientCidrBlock *string `json:"ClientCidrBlock,omitempty"`
	ClientConnectOptions *AwsEc2ClientVpnEndpointClientConnectOptionsDetails `json:"ClientConnectOptions,omitempty"`
	ClientLoginBannerOptions *AwsEc2ClientVpnEndpointClientLoginBannerOptionsDetails `json:"ClientLoginBannerOptions,omitempty"`
	ClientVpnEndpointId *string `json:"ClientVpnEndpointId,omitempty"`
	ConnectionLogOptions *AwsEc2ClientVpnEndpointConnectionLogOptionsDetails `json:"ConnectionLogOptions,omitempty"`
	Description *string `json:"Description,omitempty"`
	DnsServer []string `json:"DnsServer,omitempty"`
	SecurityGroupIdSet []string `json:"SecurityGroupIdSet,omitempty"`
	SelfServicePortalUrl *string `json:"SelfServicePortalUrl,omitempty"`
	ServerCertificateArn *string `json:"ServerCertificateArn,omitempty"`
	SessionTimeoutHours int `json:"SessionTimeoutHours,omitempty"`
	SplitTunnel bool `json:"SplitTunnel,omitempty"`
	TransportProtocol *string `json:"TransportProtocol,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
	VpnPort int `json:"VpnPort,omitempty"`
}

type AwsEc2EipDetails struct {
	AllocationId *string `json:"AllocationId,omitempty"`
	AssociationId *string `json:"AssociationId,omitempty"`
	Domain *string `json:"Domain,omitempty"`
	InstanceId *string `json:"InstanceId,omitempty"`
	NetworkBorderGroup *string `json:"NetworkBorderGroup,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	NetworkInterfaceOwnerId *string `json:"NetworkInterfaceOwnerId,omitempty"`
	PrivateIpAddress *string `json:"PrivateIpAddress,omitempty"`
	PublicIp *string `json:"PublicIp,omitempty"`
	PublicIpv4Pool *string `json:"PublicIpv4Pool,omitempty"`
}

type AwsEc2InstanceDetails struct {
	IamInstanceProfileArn *string `json:"IamInstanceProfileArn,omitempty"`
	ImageId *string `json:"ImageId,omitempty"`
	IpV4Addresses []string `json:"IpV4Addresses,omitempty"`
	IpV6Addresses []string `json:"IpV6Addresses,omitempty"`
	KeyName *string `json:"KeyName,omitempty"`
	LaunchedAt *string `json:"LaunchedAt,omitempty"`
	MetadataOptions *AwsEc2InstanceMetadataOptions `json:"MetadataOptions,omitempty"`
	Monitoring *AwsEc2InstanceMonitoringDetails `json:"Monitoring,omitempty"`
	NetworkInterfaces []AwsEc2InstanceNetworkInterfacesDetails `json:"NetworkInterfaces,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
	Type *string `json:"Type,omitempty"`
	VirtualizationType *string `json:"VirtualizationType,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEc2InstanceMetadataOptions struct {
	HttpEndpoint *string `json:"HttpEndpoint,omitempty"`
	HttpProtocolIpv6 *string `json:"HttpProtocolIpv6,omitempty"`
	HttpPutResponseHopLimit int `json:"HttpPutResponseHopLimit,omitempty"`
	HttpTokens *string `json:"HttpTokens,omitempty"`
	InstanceMetadataTags *string `json:"InstanceMetadataTags,omitempty"`
}

type AwsEc2InstanceMonitoringDetails struct {
	State *string `json:"State,omitempty"`
}

type AwsEc2InstanceNetworkInterfacesDetails struct {
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
}

type AwsEc2LaunchTemplateDataBlockDeviceMappingSetDetails struct {
	DeviceName *string `json:"DeviceName,omitempty"`
	Ebs *AwsEc2LaunchTemplateDataBlockDeviceMappingSetEbsDetails `json:"Ebs,omitempty"`
	NoDevice *string `json:"NoDevice,omitempty"`
	VirtualName *string `json:"VirtualName,omitempty"`
}

type AwsEc2LaunchTemplateDataBlockDeviceMappingSetEbsDetails struct {
	DeleteOnTermination bool `json:"DeleteOnTermination,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	Iops int `json:"Iops,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	SnapshotId *string `json:"SnapshotId,omitempty"`
	Throughput int `json:"Throughput,omitempty"`
	VolumeSize int `json:"VolumeSize,omitempty"`
	VolumeType *string `json:"VolumeType,omitempty"`
}

type AwsEc2LaunchTemplateDataCapacityReservationSpecificationCapacityReservationTargetDetails struct {
	CapacityReservationId *string `json:"CapacityReservationId,omitempty"`
	CapacityReservationResourceGroupArn *string `json:"CapacityReservationResourceGroupArn,omitempty"`
}

type AwsEc2LaunchTemplateDataCapacityReservationSpecificationDetails struct {
	CapacityReservationPreference *string `json:"CapacityReservationPreference,omitempty"`
	CapacityReservationTarget *AwsEc2LaunchTemplateDataCapacityReservationSpecificationCapacityReservationTargetDetails `json:"CapacityReservationTarget,omitempty"`
}

type AwsEc2LaunchTemplateDataCpuOptionsDetails struct {
	CoreCount int `json:"CoreCount,omitempty"`
	ThreadsPerCore int `json:"ThreadsPerCore,omitempty"`
}

type AwsEc2LaunchTemplateDataCreditSpecificationDetails struct {
	CpuCredits *string `json:"CpuCredits,omitempty"`
}

type AwsEc2LaunchTemplateDataDetails struct {
	BlockDeviceMappingSet []AwsEc2LaunchTemplateDataBlockDeviceMappingSetDetails `json:"BlockDeviceMappingSet,omitempty"`
	CapacityReservationSpecification *AwsEc2LaunchTemplateDataCapacityReservationSpecificationDetails `json:"CapacityReservationSpecification,omitempty"`
	CpuOptions *AwsEc2LaunchTemplateDataCpuOptionsDetails `json:"CpuOptions,omitempty"`
	CreditSpecification *AwsEc2LaunchTemplateDataCreditSpecificationDetails `json:"CreditSpecification,omitempty"`
	DisableApiStop bool `json:"DisableApiStop,omitempty"`
	DisableApiTermination bool `json:"DisableApiTermination,omitempty"`
	EbsOptimized bool `json:"EbsOptimized,omitempty"`
	ElasticGpuSpecificationSet []AwsEc2LaunchTemplateDataElasticGpuSpecificationSetDetails `json:"ElasticGpuSpecificationSet,omitempty"`
	ElasticInferenceAcceleratorSet []AwsEc2LaunchTemplateDataElasticInferenceAcceleratorSetDetails `json:"ElasticInferenceAcceleratorSet,omitempty"`
	EnclaveOptions *AwsEc2LaunchTemplateDataEnclaveOptionsDetails `json:"EnclaveOptions,omitempty"`
	HibernationOptions *AwsEc2LaunchTemplateDataHibernationOptionsDetails `json:"HibernationOptions,omitempty"`
	IamInstanceProfile *AwsEc2LaunchTemplateDataIamInstanceProfileDetails `json:"IamInstanceProfile,omitempty"`
	ImageId *string `json:"ImageId,omitempty"`
	InstanceInitiatedShutdownBehavior *string `json:"InstanceInitiatedShutdownBehavior,omitempty"`
	InstanceMarketOptions *AwsEc2LaunchTemplateDataInstanceMarketOptionsDetails `json:"InstanceMarketOptions,omitempty"`
	InstanceRequirements *AwsEc2LaunchTemplateDataInstanceRequirementsDetails `json:"InstanceRequirements,omitempty"`
	InstanceType *string `json:"InstanceType,omitempty"`
	KernelId *string `json:"KernelId,omitempty"`
	KeyName *string `json:"KeyName,omitempty"`
	LicenseSet []AwsEc2LaunchTemplateDataLicenseSetDetails `json:"LicenseSet,omitempty"`
	MaintenanceOptions *AwsEc2LaunchTemplateDataMaintenanceOptionsDetails `json:"MaintenanceOptions,omitempty"`
	MetadataOptions *AwsEc2LaunchTemplateDataMetadataOptionsDetails `json:"MetadataOptions,omitempty"`
	Monitoring *AwsEc2LaunchTemplateDataMonitoringDetails `json:"Monitoring,omitempty"`
	NetworkInterfaceSet []AwsEc2LaunchTemplateDataNetworkInterfaceSetDetails `json:"NetworkInterfaceSet,omitempty"`
	Placement *AwsEc2LaunchTemplateDataPlacementDetails `json:"Placement,omitempty"`
	PrivateDnsNameOptions *AwsEc2LaunchTemplateDataPrivateDnsNameOptionsDetails `json:"PrivateDnsNameOptions,omitempty"`
	RamDiskId *string `json:"RamDiskId,omitempty"`
	SecurityGroupIdSet []string `json:"SecurityGroupIdSet,omitempty"`
	SecurityGroupSet []string `json:"SecurityGroupSet,omitempty"`
	UserData *string `json:"UserData,omitempty"`
}

type AwsEc2LaunchTemplateDataElasticGpuSpecificationSetDetails struct {
	Type *string `json:"Type,omitempty"`
}

type AwsEc2LaunchTemplateDataElasticInferenceAcceleratorSetDetails struct {
	Count int `json:"Count,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEc2LaunchTemplateDataEnclaveOptionsDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsEc2LaunchTemplateDataHibernationOptionsDetails struct {
	Configured bool `json:"Configured,omitempty"`
}

type AwsEc2LaunchTemplateDataIamInstanceProfileDetails struct {
	Arn *string `json:"Arn,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceMarketOptionsDetails struct {
	MarketType *string `json:"MarketType,omitempty"`
	SpotOptions *AwsEc2LaunchTemplateDataInstanceMarketOptionsSpotOptionsDetails `json:"SpotOptions,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceMarketOptionsSpotOptionsDetails struct {
	BlockDurationMinutes int `json:"BlockDurationMinutes,omitempty"`
	InstanceInterruptionBehavior *string `json:"InstanceInterruptionBehavior,omitempty"`
	MaxPrice *string `json:"MaxPrice,omitempty"`
	SpotInstanceType *string `json:"SpotInstanceType,omitempty"`
	ValidUntil *string `json:"ValidUntil,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsAcceleratorCountDetails struct {
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsAcceleratorTotalMemoryMiBDetails struct {
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsBaselineEbsBandwidthMbpsDetails struct {
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsDetails struct {
	AcceleratorCount *AwsEc2LaunchTemplateDataInstanceRequirementsAcceleratorCountDetails `json:"AcceleratorCount,omitempty"`
	AcceleratorManufacturers []string `json:"AcceleratorManufacturers,omitempty"`
	AcceleratorNames []string `json:"AcceleratorNames,omitempty"`
	AcceleratorTotalMemoryMiB *AwsEc2LaunchTemplateDataInstanceRequirementsAcceleratorTotalMemoryMiBDetails `json:"AcceleratorTotalMemoryMiB,omitempty"`
	AcceleratorTypes []string `json:"AcceleratorTypes,omitempty"`
	BareMetal *string `json:"BareMetal,omitempty"`
	BaselineEbsBandwidthMbps *AwsEc2LaunchTemplateDataInstanceRequirementsBaselineEbsBandwidthMbpsDetails `json:"BaselineEbsBandwidthMbps,omitempty"`
	BurstablePerformance *string `json:"BurstablePerformance,omitempty"`
	CpuManufacturers []string `json:"CpuManufacturers,omitempty"`
	ExcludedInstanceTypes []string `json:"ExcludedInstanceTypes,omitempty"`
	InstanceGenerations []string `json:"InstanceGenerations,omitempty"`
	LocalStorage *string `json:"LocalStorage,omitempty"`
	LocalStorageTypes []string `json:"LocalStorageTypes,omitempty"`
	MemoryGiBPerVCpu *AwsEc2LaunchTemplateDataInstanceRequirementsMemoryGiBPerVCpuDetails `json:"MemoryGiBPerVCpu,omitempty"`
	MemoryMiB *AwsEc2LaunchTemplateDataInstanceRequirementsMemoryMiBDetails `json:"MemoryMiB,omitempty"`
	NetworkInterfaceCount *AwsEc2LaunchTemplateDataInstanceRequirementsNetworkInterfaceCountDetails `json:"NetworkInterfaceCount,omitempty"`
	OnDemandMaxPricePercentageOverLowestPrice int `json:"OnDemandMaxPricePercentageOverLowestPrice,omitempty"`
	RequireHibernateSupport bool `json:"RequireHibernateSupport,omitempty"`
	SpotMaxPricePercentageOverLowestPrice int `json:"SpotMaxPricePercentageOverLowestPrice,omitempty"`
	TotalLocalStorageGB *AwsEc2LaunchTemplateDataInstanceRequirementsTotalLocalStorageGBDetails `json:"TotalLocalStorageGB,omitempty"`
	VCpuCount *AwsEc2LaunchTemplateDataInstanceRequirementsVCpuCountDetails `json:"VCpuCount,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsMemoryGiBPerVCpuDetails struct {
	Max float64 `json:"Max,omitempty"`
	Min float64 `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsMemoryMiBDetails struct {
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsNetworkInterfaceCountDetails struct {
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsTotalLocalStorageGBDetails struct {
	Max float64 `json:"Max,omitempty"`
	Min float64 `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataInstanceRequirementsVCpuCountDetails struct {
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type AwsEc2LaunchTemplateDataLicenseSetDetails struct {
	LicenseConfigurationArn *string `json:"LicenseConfigurationArn,omitempty"`
}

type AwsEc2LaunchTemplateDataMaintenanceOptionsDetails struct {
	AutoRecovery *string `json:"AutoRecovery,omitempty"`
}

type AwsEc2LaunchTemplateDataMetadataOptionsDetails struct {
	HttpEndpoint *string `json:"HttpEndpoint,omitempty"`
	HttpProtocolIpv6 *string `json:"HttpProtocolIpv6,omitempty"`
	HttpPutResponseHopLimit int `json:"HttpPutResponseHopLimit,omitempty"`
	HttpTokens *string `json:"HttpTokens,omitempty"`
	InstanceMetadataTags *string `json:"InstanceMetadataTags,omitempty"`
}

type AwsEc2LaunchTemplateDataMonitoringDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsEc2LaunchTemplateDataNetworkInterfaceSetDetails struct {
	AssociateCarrierIpAddress bool `json:"AssociateCarrierIpAddress,omitempty"`
	AssociatePublicIpAddress bool `json:"AssociatePublicIpAddress,omitempty"`
	DeleteOnTermination bool `json:"DeleteOnTermination,omitempty"`
	Description *string `json:"Description,omitempty"`
	DeviceIndex int `json:"DeviceIndex,omitempty"`
	Groups []string `json:"Groups,omitempty"`
	InterfaceType *string `json:"InterfaceType,omitempty"`
	Ipv4PrefixCount int `json:"Ipv4PrefixCount,omitempty"`
	Ipv4Prefixes []AwsEc2LaunchTemplateDataNetworkInterfaceSetIpv4PrefixesDetails `json:"Ipv4Prefixes,omitempty"`
	Ipv6AddressCount int `json:"Ipv6AddressCount,omitempty"`
	Ipv6Addresses []AwsEc2LaunchTemplateDataNetworkInterfaceSetIpv6AddressesDetails `json:"Ipv6Addresses,omitempty"`
	Ipv6PrefixCount int `json:"Ipv6PrefixCount,omitempty"`
	Ipv6Prefixes []AwsEc2LaunchTemplateDataNetworkInterfaceSetIpv6PrefixesDetails `json:"Ipv6Prefixes,omitempty"`
	NetworkCardIndex int `json:"NetworkCardIndex,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	PrivateIpAddress *string `json:"PrivateIpAddress,omitempty"`
	PrivateIpAddresses []AwsEc2LaunchTemplateDataNetworkInterfaceSetPrivateIpAddressesDetails `json:"PrivateIpAddresses,omitempty"`
	SecondaryPrivateIpAddressCount int `json:"SecondaryPrivateIpAddressCount,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
}

type AwsEc2LaunchTemplateDataNetworkInterfaceSetIpv4PrefixesDetails struct {
	Ipv4Prefix *string `json:"Ipv4Prefix,omitempty"`
}

type AwsEc2LaunchTemplateDataNetworkInterfaceSetIpv6AddressesDetails struct {
	Ipv6Address *string `json:"Ipv6Address,omitempty"`
}

type AwsEc2LaunchTemplateDataNetworkInterfaceSetIpv6PrefixesDetails struct {
	Ipv6Prefix *string `json:"Ipv6Prefix,omitempty"`
}

type AwsEc2LaunchTemplateDataNetworkInterfaceSetPrivateIpAddressesDetails struct {
	Primary bool `json:"Primary,omitempty"`
	PrivateIpAddress *string `json:"PrivateIpAddress,omitempty"`
}

type AwsEc2LaunchTemplateDataPlacementDetails struct {
	Affinity *string `json:"Affinity,omitempty"`
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	GroupName *string `json:"GroupName,omitempty"`
	HostId *string `json:"HostId,omitempty"`
	HostResourceGroupArn *string `json:"HostResourceGroupArn,omitempty"`
	PartitionNumber int `json:"PartitionNumber,omitempty"`
	SpreadDomain *string `json:"SpreadDomain,omitempty"`
	Tenancy *string `json:"Tenancy,omitempty"`
}

type AwsEc2LaunchTemplateDataPrivateDnsNameOptionsDetails struct {
	EnableResourceNameDnsAAAARecord bool `json:"EnableResourceNameDnsAAAARecord,omitempty"`
	EnableResourceNameDnsARecord bool `json:"EnableResourceNameDnsARecord,omitempty"`
	HostnameType *string `json:"HostnameType,omitempty"`
}

type AwsEc2LaunchTemplateDetails struct {
	DefaultVersionNumber int64 `json:"DefaultVersionNumber,omitempty"`
	Id *string `json:"Id,omitempty"`
	LatestVersionNumber int64 `json:"LatestVersionNumber,omitempty"`
	LaunchTemplateData *AwsEc2LaunchTemplateDataDetails `json:"LaunchTemplateData,omitempty"`
	LaunchTemplateName *string `json:"LaunchTemplateName,omitempty"`
}

type AwsEc2NetworkAclAssociation struct {
	NetworkAclAssociationId *string `json:"NetworkAclAssociationId,omitempty"`
	NetworkAclId *string `json:"NetworkAclId,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
}

type AwsEc2NetworkAclDetails struct {
	Associations []AwsEc2NetworkAclAssociation `json:"Associations,omitempty"`
	Entries []AwsEc2NetworkAclEntry `json:"Entries,omitempty"`
	IsDefault bool `json:"IsDefault,omitempty"`
	NetworkAclId *string `json:"NetworkAclId,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEc2NetworkAclEntry struct {
	CidrBlock *string `json:"CidrBlock,omitempty"`
	Egress bool `json:"Egress,omitempty"`
	IcmpTypeCode *IcmpTypeCode `json:"IcmpTypeCode,omitempty"`
	Ipv6CidrBlock *string `json:"Ipv6CidrBlock,omitempty"`
	PortRange *PortRangeFromTo `json:"PortRange,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	RuleAction *string `json:"RuleAction,omitempty"`
	RuleNumber int `json:"RuleNumber,omitempty"`
}

type AwsEc2NetworkInterfaceAttachment struct {
	AttachTime *string `json:"AttachTime,omitempty"`
	AttachmentId *string `json:"AttachmentId,omitempty"`
	DeleteOnTermination bool `json:"DeleteOnTermination,omitempty"`
	DeviceIndex int `json:"DeviceIndex,omitempty"`
	InstanceId *string `json:"InstanceId,omitempty"`
	InstanceOwnerId *string `json:"InstanceOwnerId,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsEc2NetworkInterfaceDetails struct {
	Attachment *AwsEc2NetworkInterfaceAttachment `json:"Attachment,omitempty"`
	IpV6Addresses []AwsEc2NetworkInterfaceIpV6AddressDetail `json:"IpV6Addresses,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	PrivateIpAddresses []AwsEc2NetworkInterfacePrivateIpAddressDetail `json:"PrivateIpAddresses,omitempty"`
	PublicDnsName *string `json:"PublicDnsName,omitempty"`
	PublicIp *string `json:"PublicIp,omitempty"`
	SecurityGroups []AwsEc2NetworkInterfaceSecurityGroup `json:"SecurityGroups,omitempty"`
	SourceDestCheck bool `json:"SourceDestCheck,omitempty"`
}

type AwsEc2NetworkInterfaceIpV6AddressDetail struct {
	IpV6Address *string `json:"IpV6Address,omitempty"`
}

type AwsEc2NetworkInterfacePrivateIpAddressDetail struct {
	PrivateDnsName *string `json:"PrivateDnsName,omitempty"`
	PrivateIpAddress *string `json:"PrivateIpAddress,omitempty"`
}

type AwsEc2NetworkInterfaceSecurityGroup struct {
	GroupId *string `json:"GroupId,omitempty"`
	GroupName *string `json:"GroupName,omitempty"`
}

type AwsEc2RouteTableDetails struct {
	AssociationSet []AssociationSetDetails `json:"AssociationSet,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	PropagatingVgwSet []PropagatingVgwSetDetails `json:"PropagatingVgwSet,omitempty"`
	RouteSet []RouteSetDetails `json:"RouteSet,omitempty"`
	RouteTableId *string `json:"RouteTableId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEc2SecurityGroupDetails struct {
	GroupId *string `json:"GroupId,omitempty"`
	GroupName *string `json:"GroupName,omitempty"`
	IpPermissions []AwsEc2SecurityGroupIpPermission `json:"IpPermissions,omitempty"`
	IpPermissionsEgress []AwsEc2SecurityGroupIpPermission `json:"IpPermissionsEgress,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEc2SecurityGroupIpPermission struct {
	FromPort int `json:"FromPort,omitempty"`
	IpProtocol *string `json:"IpProtocol,omitempty"`
	IpRanges []AwsEc2SecurityGroupIpRange `json:"IpRanges,omitempty"`
	Ipv6Ranges []AwsEc2SecurityGroupIpv6Range `json:"Ipv6Ranges,omitempty"`
	PrefixListIds []AwsEc2SecurityGroupPrefixListId `json:"PrefixListIds,omitempty"`
	ToPort int `json:"ToPort,omitempty"`
	UserIdGroupPairs []AwsEc2SecurityGroupUserIdGroupPair `json:"UserIdGroupPairs,omitempty"`
}

type AwsEc2SecurityGroupIpRange struct {
	CidrIp *string `json:"CidrIp,omitempty"`
}

type AwsEc2SecurityGroupIpv6Range struct {
	CidrIpv6 *string `json:"CidrIpv6,omitempty"`
}

type AwsEc2SecurityGroupPrefixListId struct {
	PrefixListId *string `json:"PrefixListId,omitempty"`
}

type AwsEc2SecurityGroupUserIdGroupPair struct {
	GroupId *string `json:"GroupId,omitempty"`
	GroupName *string `json:"GroupName,omitempty"`
	PeeringStatus *string `json:"PeeringStatus,omitempty"`
	UserId *string `json:"UserId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
	VpcPeeringConnectionId *string `json:"VpcPeeringConnectionId,omitempty"`
}

type AwsEc2SubnetDetails struct {
	AssignIpv6AddressOnCreation bool `json:"AssignIpv6AddressOnCreation,omitempty"`
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	AvailabilityZoneId *string `json:"AvailabilityZoneId,omitempty"`
	AvailableIpAddressCount int `json:"AvailableIpAddressCount,omitempty"`
	CidrBlock *string `json:"CidrBlock,omitempty"`
	DefaultForAz bool `json:"DefaultForAz,omitempty"`
	Ipv6CidrBlockAssociationSet []Ipv6CidrBlockAssociation `json:"Ipv6CidrBlockAssociationSet,omitempty"`
	MapPublicIpOnLaunch bool `json:"MapPublicIpOnLaunch,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	State *string `json:"State,omitempty"`
	SubnetArn *string `json:"SubnetArn,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEc2TransitGatewayDetails struct {
	AmazonSideAsn int `json:"AmazonSideAsn,omitempty"`
	AssociationDefaultRouteTableId *string `json:"AssociationDefaultRouteTableId,omitempty"`
	AutoAcceptSharedAttachments *string `json:"AutoAcceptSharedAttachments,omitempty"`
	DefaultRouteTableAssociation *string `json:"DefaultRouteTableAssociation,omitempty"`
	DefaultRouteTablePropagation *string `json:"DefaultRouteTablePropagation,omitempty"`
	Description *string `json:"Description,omitempty"`
	DnsSupport *string `json:"DnsSupport,omitempty"`
	Id *string `json:"Id,omitempty"`
	MulticastSupport *string `json:"MulticastSupport,omitempty"`
	PropagationDefaultRouteTableId *string `json:"PropagationDefaultRouteTableId,omitempty"`
	TransitGatewayCidrBlocks []string `json:"TransitGatewayCidrBlocks,omitempty"`
	VpnEcmpSupport *string `json:"VpnEcmpSupport,omitempty"`
}

type AwsEc2VolumeAttachment struct {
	AttachTime *string `json:"AttachTime,omitempty"`
	DeleteOnTermination bool `json:"DeleteOnTermination,omitempty"`
	InstanceId *string `json:"InstanceId,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsEc2VolumeDetails struct {
	Attachments []AwsEc2VolumeAttachment `json:"Attachments,omitempty"`
	CreateTime *string `json:"CreateTime,omitempty"`
	DeviceName *string `json:"DeviceName,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	Size int `json:"Size,omitempty"`
	SnapshotId *string `json:"SnapshotId,omitempty"`
	Status *string `json:"Status,omitempty"`
	VolumeId *string `json:"VolumeId,omitempty"`
	VolumeScanStatus *string `json:"VolumeScanStatus,omitempty"`
	VolumeType *string `json:"VolumeType,omitempty"`
}

type AwsEc2VpcDetails struct {
	CidrBlockAssociationSet []CidrBlockAssociation `json:"CidrBlockAssociationSet,omitempty"`
	DhcpOptionsId *string `json:"DhcpOptionsId,omitempty"`
	Ipv6CidrBlockAssociationSet []Ipv6CidrBlockAssociation `json:"Ipv6CidrBlockAssociationSet,omitempty"`
	State *string `json:"State,omitempty"`
}

type AwsEc2VpcEndpointServiceDetails struct {
	AcceptanceRequired bool `json:"AcceptanceRequired,omitempty"`
	AvailabilityZones []string `json:"AvailabilityZones,omitempty"`
	BaseEndpointDnsNames []string `json:"BaseEndpointDnsNames,omitempty"`
	GatewayLoadBalancerArns []string `json:"GatewayLoadBalancerArns,omitempty"`
	ManagesVpcEndpoints bool `json:"ManagesVpcEndpoints,omitempty"`
	NetworkLoadBalancerArns []string `json:"NetworkLoadBalancerArns,omitempty"`
	PrivateDnsName *string `json:"PrivateDnsName,omitempty"`
	ServiceId *string `json:"ServiceId,omitempty"`
	ServiceName *string `json:"ServiceName,omitempty"`
	ServiceState *string `json:"ServiceState,omitempty"`
	ServiceType []AwsEc2VpcEndpointServiceServiceTypeDetails `json:"ServiceType,omitempty"`
}

type AwsEc2VpcEndpointServiceServiceTypeDetails struct {
	ServiceType *string `json:"ServiceType,omitempty"`
}

type AwsEc2VpcPeeringConnectionDetails struct {
	AccepterVpcInfo *AwsEc2VpcPeeringConnectionVpcInfoDetails `json:"AccepterVpcInfo,omitempty"`
	ExpirationTime *string `json:"ExpirationTime,omitempty"`
	RequesterVpcInfo *AwsEc2VpcPeeringConnectionVpcInfoDetails `json:"RequesterVpcInfo,omitempty"`
	Status *AwsEc2VpcPeeringConnectionStatusDetails `json:"Status,omitempty"`
	VpcPeeringConnectionId *string `json:"VpcPeeringConnectionId,omitempty"`
}

type AwsEc2VpcPeeringConnectionStatusDetails struct {
	Code *string `json:"Code,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type AwsEc2VpcPeeringConnectionVpcInfoDetails struct {
	CidrBlock *string `json:"CidrBlock,omitempty"`
	CidrBlockSet []VpcInfoCidrBlockSetDetails `json:"CidrBlockSet,omitempty"`
	Ipv6CidrBlockSet []VpcInfoIpv6CidrBlockSetDetails `json:"Ipv6CidrBlockSet,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	PeeringOptions *VpcInfoPeeringOptionsDetails `json:"PeeringOptions,omitempty"`
	Region *string `json:"Region,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEc2VpnConnectionDetails struct {
	Category *string `json:"Category,omitempty"`
	CustomerGatewayConfiguration *string `json:"CustomerGatewayConfiguration,omitempty"`
	CustomerGatewayId *string `json:"CustomerGatewayId,omitempty"`
	Options *AwsEc2VpnConnectionOptionsDetails `json:"Options,omitempty"`
	Routes []AwsEc2VpnConnectionRoutesDetails `json:"Routes,omitempty"`
	State *string `json:"State,omitempty"`
	TransitGatewayId *string `json:"TransitGatewayId,omitempty"`
	Type *string `json:"Type,omitempty"`
	VgwTelemetry []AwsEc2VpnConnectionVgwTelemetryDetails `json:"VgwTelemetry,omitempty"`
	VpnConnectionId *string `json:"VpnConnectionId,omitempty"`
	VpnGatewayId *string `json:"VpnGatewayId,omitempty"`
}

type AwsEc2VpnConnectionOptionsDetails struct {
	StaticRoutesOnly bool `json:"StaticRoutesOnly,omitempty"`
	TunnelOptions []AwsEc2VpnConnectionOptionsTunnelOptionsDetails `json:"TunnelOptions,omitempty"`
}

type AwsEc2VpnConnectionOptionsTunnelOptionsDetails struct {
	DpdTimeoutSeconds int `json:"DpdTimeoutSeconds,omitempty"`
	IkeVersions []string `json:"IkeVersions,omitempty"`
	OutsideIpAddress *string `json:"OutsideIpAddress,omitempty"`
	Phase1DhGroupNumbers []int `json:"Phase1DhGroupNumbers,omitempty"`
	Phase1EncryptionAlgorithms []string `json:"Phase1EncryptionAlgorithms,omitempty"`
	Phase1IntegrityAlgorithms []string `json:"Phase1IntegrityAlgorithms,omitempty"`
	Phase1LifetimeSeconds int `json:"Phase1LifetimeSeconds,omitempty"`
	Phase2DhGroupNumbers []int `json:"Phase2DhGroupNumbers,omitempty"`
	Phase2EncryptionAlgorithms []string `json:"Phase2EncryptionAlgorithms,omitempty"`
	Phase2IntegrityAlgorithms []string `json:"Phase2IntegrityAlgorithms,omitempty"`
	Phase2LifetimeSeconds int `json:"Phase2LifetimeSeconds,omitempty"`
	PreSharedKey *string `json:"PreSharedKey,omitempty"`
	RekeyFuzzPercentage int `json:"RekeyFuzzPercentage,omitempty"`
	RekeyMarginTimeSeconds int `json:"RekeyMarginTimeSeconds,omitempty"`
	ReplayWindowSize int `json:"ReplayWindowSize,omitempty"`
	TunnelInsideCidr *string `json:"TunnelInsideCidr,omitempty"`
}

type AwsEc2VpnConnectionRoutesDetails struct {
	DestinationCidrBlock *string `json:"DestinationCidrBlock,omitempty"`
	State *string `json:"State,omitempty"`
}

type AwsEc2VpnConnectionVgwTelemetryDetails struct {
	AcceptedRouteCount int `json:"AcceptedRouteCount,omitempty"`
	CertificateArn *string `json:"CertificateArn,omitempty"`
	LastStatusChange *string `json:"LastStatusChange,omitempty"`
	OutsideIpAddress *string `json:"OutsideIpAddress,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
}

type AwsEcrContainerImageDetails struct {
	Architecture *string `json:"Architecture,omitempty"`
	ImageDigest *string `json:"ImageDigest,omitempty"`
	ImagePublishedAt *string `json:"ImagePublishedAt,omitempty"`
	ImageTags []string `json:"ImageTags,omitempty"`
	RegistryId *string `json:"RegistryId,omitempty"`
	RepositoryName *string `json:"RepositoryName,omitempty"`
}

type AwsEcrRepositoryDetails struct {
	Arn *string `json:"Arn,omitempty"`
	ImageScanningConfiguration *AwsEcrRepositoryImageScanningConfigurationDetails `json:"ImageScanningConfiguration,omitempty"`
	ImageTagMutability *string `json:"ImageTagMutability,omitempty"`
	LifecyclePolicy *AwsEcrRepositoryLifecyclePolicyDetails `json:"LifecyclePolicy,omitempty"`
	RepositoryName *string `json:"RepositoryName,omitempty"`
	RepositoryPolicyText *string `json:"RepositoryPolicyText,omitempty"`
}

type AwsEcrRepositoryImageScanningConfigurationDetails struct {
	ScanOnPush bool `json:"ScanOnPush,omitempty"`
}

type AwsEcrRepositoryLifecyclePolicyDetails struct {
	LifecyclePolicyText *string `json:"LifecyclePolicyText,omitempty"`
	RegistryId *string `json:"RegistryId,omitempty"`
}

type AwsEcsClusterClusterSettingsDetails struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsEcsClusterConfigurationDetails struct {
	ExecuteCommandConfiguration *AwsEcsClusterConfigurationExecuteCommandConfigurationDetails `json:"ExecuteCommandConfiguration,omitempty"`
}

type AwsEcsClusterConfigurationExecuteCommandConfigurationDetails struct {
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LogConfiguration *AwsEcsClusterConfigurationExecuteCommandConfigurationLogConfigurationDetails `json:"LogConfiguration,omitempty"`
	Logging *string `json:"Logging,omitempty"`
}

type AwsEcsClusterConfigurationExecuteCommandConfigurationLogConfigurationDetails struct {
	CloudWatchEncryptionEnabled bool `json:"CloudWatchEncryptionEnabled,omitempty"`
	CloudWatchLogGroupName *string `json:"CloudWatchLogGroupName,omitempty"`
	S3BucketName *string `json:"S3BucketName,omitempty"`
	S3EncryptionEnabled bool `json:"S3EncryptionEnabled,omitempty"`
	S3KeyPrefix *string `json:"S3KeyPrefix,omitempty"`
}

type AwsEcsClusterDefaultCapacityProviderStrategyDetails struct {
	Base int `json:"Base,omitempty"`
	CapacityProvider *string `json:"CapacityProvider,omitempty"`
	Weight int `json:"Weight,omitempty"`
}

type AwsEcsClusterDetails struct {
	ActiveServicesCount int `json:"ActiveServicesCount,omitempty"`
	CapacityProviders []string `json:"CapacityProviders,omitempty"`
	ClusterArn *string `json:"ClusterArn,omitempty"`
	ClusterName *string `json:"ClusterName,omitempty"`
	ClusterSettings []AwsEcsClusterClusterSettingsDetails `json:"ClusterSettings,omitempty"`
	Configuration *AwsEcsClusterConfigurationDetails `json:"Configuration,omitempty"`
	DefaultCapacityProviderStrategy []AwsEcsClusterDefaultCapacityProviderStrategyDetails `json:"DefaultCapacityProviderStrategy,omitempty"`
	RegisteredContainerInstancesCount int `json:"RegisteredContainerInstancesCount,omitempty"`
	RunningTasksCount int `json:"RunningTasksCount,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsEcsContainerDetails struct {
	Image *string `json:"Image,omitempty"`
	MountPoints []AwsMountPoint `json:"MountPoints,omitempty"`
	Name *string `json:"Name,omitempty"`
	Privileged bool `json:"Privileged,omitempty"`
}

type AwsEcsServiceCapacityProviderStrategyDetails struct {
	Base int `json:"Base,omitempty"`
	CapacityProvider *string `json:"CapacityProvider,omitempty"`
	Weight int `json:"Weight,omitempty"`
}

type AwsEcsServiceDeploymentConfigurationDeploymentCircuitBreakerDetails struct {
	Enable bool `json:"Enable,omitempty"`
	Rollback bool `json:"Rollback,omitempty"`
}

type AwsEcsServiceDeploymentConfigurationDetails struct {
	DeploymentCircuitBreaker *AwsEcsServiceDeploymentConfigurationDeploymentCircuitBreakerDetails `json:"DeploymentCircuitBreaker,omitempty"`
	MaximumPercent int `json:"MaximumPercent,omitempty"`
	MinimumHealthyPercent int `json:"MinimumHealthyPercent,omitempty"`
}

type AwsEcsServiceDeploymentControllerDetails struct {
	Type *string `json:"Type,omitempty"`
}

type AwsEcsServiceDetails struct {
	CapacityProviderStrategy []AwsEcsServiceCapacityProviderStrategyDetails `json:"CapacityProviderStrategy,omitempty"`
	Cluster *string `json:"Cluster,omitempty"`
	DeploymentConfiguration *AwsEcsServiceDeploymentConfigurationDetails `json:"DeploymentConfiguration,omitempty"`
	DeploymentController *AwsEcsServiceDeploymentControllerDetails `json:"DeploymentController,omitempty"`
	DesiredCount int `json:"DesiredCount,omitempty"`
	EnableEcsManagedTags bool `json:"EnableEcsManagedTags,omitempty"`
	EnableExecuteCommand bool `json:"EnableExecuteCommand,omitempty"`
	HealthCheckGracePeriodSeconds int `json:"HealthCheckGracePeriodSeconds,omitempty"`
	LaunchType *string `json:"LaunchType,omitempty"`
	LoadBalancers []AwsEcsServiceLoadBalancersDetails `json:"LoadBalancers,omitempty"`
	Name *string `json:"Name,omitempty"`
	NetworkConfiguration *AwsEcsServiceNetworkConfigurationDetails `json:"NetworkConfiguration,omitempty"`
	PlacementConstraints []AwsEcsServicePlacementConstraintsDetails `json:"PlacementConstraints,omitempty"`
	PlacementStrategies []AwsEcsServicePlacementStrategiesDetails `json:"PlacementStrategies,omitempty"`
	PlatformVersion *string `json:"PlatformVersion,omitempty"`
	PropagateTags *string `json:"PropagateTags,omitempty"`
	Role *string `json:"Role,omitempty"`
	SchedulingStrategy *string `json:"SchedulingStrategy,omitempty"`
	ServiceArn *string `json:"ServiceArn,omitempty"`
	ServiceName *string `json:"ServiceName,omitempty"`
	ServiceRegistries []AwsEcsServiceServiceRegistriesDetails `json:"ServiceRegistries,omitempty"`
	TaskDefinition *string `json:"TaskDefinition,omitempty"`
}

type AwsEcsServiceLoadBalancersDetails struct {
	ContainerName *string `json:"ContainerName,omitempty"`
	ContainerPort int `json:"ContainerPort,omitempty"`
	LoadBalancerName *string `json:"LoadBalancerName,omitempty"`
	TargetGroupArn *string `json:"TargetGroupArn,omitempty"`
}

type AwsEcsServiceNetworkConfigurationAwsVpcConfigurationDetails struct {
	AssignPublicIp *string `json:"AssignPublicIp,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	Subnets []string `json:"Subnets,omitempty"`
}

type AwsEcsServiceNetworkConfigurationDetails struct {
	AwsVpcConfiguration *AwsEcsServiceNetworkConfigurationAwsVpcConfigurationDetails `json:"AwsVpcConfiguration,omitempty"`
}

type AwsEcsServicePlacementConstraintsDetails struct {
	Expression *string `json:"Expression,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEcsServicePlacementStrategiesDetails struct {
	Field *string `json:"Field,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEcsServiceServiceRegistriesDetails struct {
	ContainerName *string `json:"ContainerName,omitempty"`
	ContainerPort int `json:"ContainerPort,omitempty"`
	Port int `json:"Port,omitempty"`
	RegistryArn *string `json:"RegistryArn,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsDependsOnDetails struct {
	Condition *string `json:"Condition,omitempty"`
	ContainerName *string `json:"ContainerName,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsDetails struct {
	Command []string `json:"Command,omitempty"`
	Cpu int `json:"Cpu,omitempty"`
	DependsOn []AwsEcsTaskDefinitionContainerDefinitionsDependsOnDetails `json:"DependsOn,omitempty"`
	DisableNetworking bool `json:"DisableNetworking,omitempty"`
	DnsSearchDomains []string `json:"DnsSearchDomains,omitempty"`
	DnsServers []string `json:"DnsServers,omitempty"`
	DockerLabels map[string]string `json:"DockerLabels,omitempty"`
	DockerSecurityOptions []string `json:"DockerSecurityOptions,omitempty"`
	EntryPoint []string `json:"EntryPoint,omitempty"`
	Environment []AwsEcsTaskDefinitionContainerDefinitionsEnvironmentDetails `json:"Environment,omitempty"`
	EnvironmentFiles []AwsEcsTaskDefinitionContainerDefinitionsEnvironmentFilesDetails `json:"EnvironmentFiles,omitempty"`
	Essential bool `json:"Essential,omitempty"`
	ExtraHosts []AwsEcsTaskDefinitionContainerDefinitionsExtraHostsDetails `json:"ExtraHosts,omitempty"`
	FirelensConfiguration *AwsEcsTaskDefinitionContainerDefinitionsFirelensConfigurationDetails `json:"FirelensConfiguration,omitempty"`
	HealthCheck *AwsEcsTaskDefinitionContainerDefinitionsHealthCheckDetails `json:"HealthCheck,omitempty"`
	Hostname *string `json:"Hostname,omitempty"`
	Image *string `json:"Image,omitempty"`
	Interactive bool `json:"Interactive,omitempty"`
	Links []string `json:"Links,omitempty"`
	LinuxParameters *AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersDetails `json:"LinuxParameters,omitempty"`
	LogConfiguration *AwsEcsTaskDefinitionContainerDefinitionsLogConfigurationDetails `json:"LogConfiguration,omitempty"`
	Memory int `json:"Memory,omitempty"`
	MemoryReservation int `json:"MemoryReservation,omitempty"`
	MountPoints []AwsEcsTaskDefinitionContainerDefinitionsMountPointsDetails `json:"MountPoints,omitempty"`
	Name *string `json:"Name,omitempty"`
	PortMappings []AwsEcsTaskDefinitionContainerDefinitionsPortMappingsDetails `json:"PortMappings,omitempty"`
	Privileged bool `json:"Privileged,omitempty"`
	PseudoTerminal bool `json:"PseudoTerminal,omitempty"`
	ReadonlyRootFilesystem bool `json:"ReadonlyRootFilesystem,omitempty"`
	RepositoryCredentials *AwsEcsTaskDefinitionContainerDefinitionsRepositoryCredentialsDetails `json:"RepositoryCredentials,omitempty"`
	ResourceRequirements []AwsEcsTaskDefinitionContainerDefinitionsResourceRequirementsDetails `json:"ResourceRequirements,omitempty"`
	Secrets []AwsEcsTaskDefinitionContainerDefinitionsSecretsDetails `json:"Secrets,omitempty"`
	StartTimeout int `json:"StartTimeout,omitempty"`
	StopTimeout int `json:"StopTimeout,omitempty"`
	SystemControls []AwsEcsTaskDefinitionContainerDefinitionsSystemControlsDetails `json:"SystemControls,omitempty"`
	Ulimits []AwsEcsTaskDefinitionContainerDefinitionsUlimitsDetails `json:"Ulimits,omitempty"`
	User *string `json:"User,omitempty"`
	VolumesFrom []AwsEcsTaskDefinitionContainerDefinitionsVolumesFromDetails `json:"VolumesFrom,omitempty"`
	WorkingDirectory *string `json:"WorkingDirectory,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsEnvironmentDetails struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsEnvironmentFilesDetails struct {
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsExtraHostsDetails struct {
	Hostname *string `json:"Hostname,omitempty"`
	IpAddress *string `json:"IpAddress,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsFirelensConfigurationDetails struct {
	Options map[string]string `json:"Options,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsHealthCheckDetails struct {
	Command []string `json:"Command,omitempty"`
	Interval int `json:"Interval,omitempty"`
	Retries int `json:"Retries,omitempty"`
	StartPeriod int `json:"StartPeriod,omitempty"`
	Timeout int `json:"Timeout,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersCapabilitiesDetails struct {
	Add []string `json:"Add,omitempty"`
	Drop []string `json:"Drop,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersDetails struct {
	Capabilities *AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersCapabilitiesDetails `json:"Capabilities,omitempty"`
	Devices []AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersDevicesDetails `json:"Devices,omitempty"`
	InitProcessEnabled bool `json:"InitProcessEnabled,omitempty"`
	MaxSwap int `json:"MaxSwap,omitempty"`
	SharedMemorySize int `json:"SharedMemorySize,omitempty"`
	Swappiness int `json:"Swappiness,omitempty"`
	Tmpfs []AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersTmpfsDetails `json:"Tmpfs,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersDevicesDetails struct {
	ContainerPath *string `json:"ContainerPath,omitempty"`
	HostPath *string `json:"HostPath,omitempty"`
	Permissions []string `json:"Permissions,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsLinuxParametersTmpfsDetails struct {
	ContainerPath *string `json:"ContainerPath,omitempty"`
	MountOptions []string `json:"MountOptions,omitempty"`
	Size int `json:"Size,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsLogConfigurationDetails struct {
	LogDriver *string `json:"LogDriver,omitempty"`
	Options map[string]string `json:"Options,omitempty"`
	SecretOptions []AwsEcsTaskDefinitionContainerDefinitionsLogConfigurationSecretOptionsDetails `json:"SecretOptions,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsLogConfigurationSecretOptionsDetails struct {
	Name *string `json:"Name,omitempty"`
	ValueFrom *string `json:"ValueFrom,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsMountPointsDetails struct {
	ContainerPath *string `json:"ContainerPath,omitempty"`
	ReadOnly bool `json:"ReadOnly,omitempty"`
	SourceVolume *string `json:"SourceVolume,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsPortMappingsDetails struct {
	ContainerPort int `json:"ContainerPort,omitempty"`
	HostPort int `json:"HostPort,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsRepositoryCredentialsDetails struct {
	CredentialsParameter *string `json:"CredentialsParameter,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsResourceRequirementsDetails struct {
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsSecretsDetails struct {
	Name *string `json:"Name,omitempty"`
	ValueFrom *string `json:"ValueFrom,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsSystemControlsDetails struct {
	Namespace *string `json:"Namespace,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsUlimitsDetails struct {
	HardLimit int `json:"HardLimit,omitempty"`
	Name *string `json:"Name,omitempty"`
	SoftLimit int `json:"SoftLimit,omitempty"`
}

type AwsEcsTaskDefinitionContainerDefinitionsVolumesFromDetails struct {
	ReadOnly bool `json:"ReadOnly,omitempty"`
	SourceContainer *string `json:"SourceContainer,omitempty"`
}

type AwsEcsTaskDefinitionDetails struct {
	ContainerDefinitions []AwsEcsTaskDefinitionContainerDefinitionsDetails `json:"ContainerDefinitions,omitempty"`
	Cpu *string `json:"Cpu,omitempty"`
	ExecutionRoleArn *string `json:"ExecutionRoleArn,omitempty"`
	Family *string `json:"Family,omitempty"`
	InferenceAccelerators []AwsEcsTaskDefinitionInferenceAcceleratorsDetails `json:"InferenceAccelerators,omitempty"`
	IpcMode *string `json:"IpcMode,omitempty"`
	Memory *string `json:"Memory,omitempty"`
	NetworkMode *string `json:"NetworkMode,omitempty"`
	PidMode *string `json:"PidMode,omitempty"`
	PlacementConstraints []AwsEcsTaskDefinitionPlacementConstraintsDetails `json:"PlacementConstraints,omitempty"`
	ProxyConfiguration *AwsEcsTaskDefinitionProxyConfigurationDetails `json:"ProxyConfiguration,omitempty"`
	RequiresCompatibilities []string `json:"RequiresCompatibilities,omitempty"`
	Status *string `json:"Status,omitempty"`
	TaskRoleArn *string `json:"TaskRoleArn,omitempty"`
	Volumes []AwsEcsTaskDefinitionVolumesDetails `json:"Volumes,omitempty"`
}

type AwsEcsTaskDefinitionInferenceAcceleratorsDetails struct {
	DeviceName *string `json:"DeviceName,omitempty"`
	DeviceType *string `json:"DeviceType,omitempty"`
}

type AwsEcsTaskDefinitionPlacementConstraintsDetails struct {
	Expression *string `json:"Expression,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEcsTaskDefinitionProxyConfigurationDetails struct {
	ContainerName *string `json:"ContainerName,omitempty"`
	ProxyConfigurationProperties []AwsEcsTaskDefinitionProxyConfigurationProxyConfigurationPropertiesDetails `json:"ProxyConfigurationProperties,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsEcsTaskDefinitionProxyConfigurationProxyConfigurationPropertiesDetails struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsEcsTaskDefinitionVolumesDetails struct {
	DockerVolumeConfiguration *AwsEcsTaskDefinitionVolumesDockerVolumeConfigurationDetails `json:"DockerVolumeConfiguration,omitempty"`
	EfsVolumeConfiguration *AwsEcsTaskDefinitionVolumesEfsVolumeConfigurationDetails `json:"EfsVolumeConfiguration,omitempty"`
	Host *AwsEcsTaskDefinitionVolumesHostDetails `json:"Host,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AwsEcsTaskDefinitionVolumesDockerVolumeConfigurationDetails struct {
	Autoprovision bool `json:"Autoprovision,omitempty"`
	Driver *string `json:"Driver,omitempty"`
	DriverOpts map[string]string `json:"DriverOpts,omitempty"`
	Labels map[string]string `json:"Labels,omitempty"`
	Scope *string `json:"Scope,omitempty"`
}

type AwsEcsTaskDefinitionVolumesEfsVolumeConfigurationAuthorizationConfigDetails struct {
	AccessPointId *string `json:"AccessPointId,omitempty"`
	Iam *string `json:"Iam,omitempty"`
}

type AwsEcsTaskDefinitionVolumesEfsVolumeConfigurationDetails struct {
	AuthorizationConfig *AwsEcsTaskDefinitionVolumesEfsVolumeConfigurationAuthorizationConfigDetails `json:"AuthorizationConfig,omitempty"`
	FilesystemId *string `json:"FilesystemId,omitempty"`
	RootDirectory *string `json:"RootDirectory,omitempty"`
	TransitEncryption *string `json:"TransitEncryption,omitempty"`
	TransitEncryptionPort int `json:"TransitEncryptionPort,omitempty"`
}

type AwsEcsTaskDefinitionVolumesHostDetails struct {
	SourcePath *string `json:"SourcePath,omitempty"`
}

type AwsEcsTaskDetails struct {
	ClusterArn *string `json:"ClusterArn,omitempty"`
	Containers []AwsEcsContainerDetails `json:"Containers,omitempty"`
	CreatedAt *string `json:"CreatedAt,omitempty"`
	Group *string `json:"Group,omitempty"`
	StartedAt *string `json:"StartedAt,omitempty"`
	StartedBy *string `json:"StartedBy,omitempty"`
	TaskDefinitionArn *string `json:"TaskDefinitionArn,omitempty"`
	Version *string `json:"Version,omitempty"`
	Volumes []AwsEcsTaskVolumeDetails `json:"Volumes,omitempty"`
}

type AwsEcsTaskVolumeDetails struct {
	Host *AwsEcsTaskVolumeHostDetails `json:"Host,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AwsEcsTaskVolumeHostDetails struct {
	SourcePath *string `json:"SourcePath,omitempty"`
}

type AwsEfsAccessPointDetails struct {
	AccessPointId *string `json:"AccessPointId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	ClientToken *string `json:"ClientToken,omitempty"`
	FileSystemId *string `json:"FileSystemId,omitempty"`
	PosixUser *AwsEfsAccessPointPosixUserDetails `json:"PosixUser,omitempty"`
	RootDirectory *AwsEfsAccessPointRootDirectoryDetails `json:"RootDirectory,omitempty"`
}

type AwsEfsAccessPointPosixUserDetails struct {
	Gid *string `json:"Gid,omitempty"`
	SecondaryGids []string `json:"SecondaryGids,omitempty"`
	Uid *string `json:"Uid,omitempty"`
}

type AwsEfsAccessPointRootDirectoryCreationInfoDetails struct {
	OwnerGid *string `json:"OwnerGid,omitempty"`
	OwnerUid *string `json:"OwnerUid,omitempty"`
	Permissions *string `json:"Permissions,omitempty"`
}

type AwsEfsAccessPointRootDirectoryDetails struct {
	CreationInfo *AwsEfsAccessPointRootDirectoryCreationInfoDetails `json:"CreationInfo,omitempty"`
	Path *string `json:"Path,omitempty"`
}

type AwsEksClusterDetails struct {
	Arn *string `json:"Arn,omitempty"`
	CertificateAuthorityData *string `json:"CertificateAuthorityData,omitempty"`
	ClusterStatus *string `json:"ClusterStatus,omitempty"`
	Endpoint *string `json:"Endpoint,omitempty"`
	Logging *AwsEksClusterLoggingDetails `json:"Logging,omitempty"`
	Name *string `json:"Name,omitempty"`
	ResourcesVpcConfig *AwsEksClusterResourcesVpcConfigDetails `json:"ResourcesVpcConfig,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type AwsEksClusterLoggingClusterLoggingDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
	Types []string `json:"Types,omitempty"`
}

type AwsEksClusterLoggingDetails struct {
	ClusterLogging []AwsEksClusterLoggingClusterLoggingDetails `json:"ClusterLogging,omitempty"`
}

type AwsEksClusterResourcesVpcConfigDetails struct {
	EndpointPublicAccess bool `json:"EndpointPublicAccess,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
}

type AwsElasticBeanstalkEnvironmentDetails struct {
	ApplicationName *string `json:"ApplicationName,omitempty"`
	Cname *string `json:"Cname,omitempty"`
	DateCreated *string `json:"DateCreated,omitempty"`
	DateUpdated *string `json:"DateUpdated,omitempty"`
	Description *string `json:"Description,omitempty"`
	EndpointUrl *string `json:"EndpointUrl,omitempty"`
	EnvironmentArn *string `json:"EnvironmentArn,omitempty"`
	EnvironmentId *string `json:"EnvironmentId,omitempty"`
	EnvironmentLinks []AwsElasticBeanstalkEnvironmentEnvironmentLink `json:"EnvironmentLinks,omitempty"`
	EnvironmentName *string `json:"EnvironmentName,omitempty"`
	OptionSettings []AwsElasticBeanstalkEnvironmentOptionSetting `json:"OptionSettings,omitempty"`
	PlatformArn *string `json:"PlatformArn,omitempty"`
	SolutionStackName *string `json:"SolutionStackName,omitempty"`
	Status *string `json:"Status,omitempty"`
	Tier *AwsElasticBeanstalkEnvironmentTier `json:"Tier,omitempty"`
	VersionLabel *string `json:"VersionLabel,omitempty"`
}

type AwsElasticBeanstalkEnvironmentEnvironmentLink struct {
	EnvironmentName *string `json:"EnvironmentName,omitempty"`
	LinkName *string `json:"LinkName,omitempty"`
}

type AwsElasticBeanstalkEnvironmentOptionSetting struct {
	Namespace *string `json:"Namespace,omitempty"`
	OptionName *string `json:"OptionName,omitempty"`
	ResourceName *string `json:"ResourceName,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsElasticBeanstalkEnvironmentTier struct {
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type AwsElasticsearchDomainDetails struct {
	AccessPolicies *string `json:"AccessPolicies,omitempty"`
	DomainEndpointOptions *AwsElasticsearchDomainDomainEndpointOptions `json:"DomainEndpointOptions,omitempty"`
	DomainId *string `json:"DomainId,omitempty"`
	DomainName *string `json:"DomainName,omitempty"`
	ElasticsearchClusterConfig *AwsElasticsearchDomainElasticsearchClusterConfigDetails `json:"ElasticsearchClusterConfig,omitempty"`
	ElasticsearchVersion *string `json:"ElasticsearchVersion,omitempty"`
	EncryptionAtRestOptions *AwsElasticsearchDomainEncryptionAtRestOptions `json:"EncryptionAtRestOptions,omitempty"`
	Endpoint *string `json:"Endpoint,omitempty"`
	Endpoints map[string]string `json:"Endpoints,omitempty"`
	LogPublishingOptions *AwsElasticsearchDomainLogPublishingOptions `json:"LogPublishingOptions,omitempty"`
	NodeToNodeEncryptionOptions *AwsElasticsearchDomainNodeToNodeEncryptionOptions `json:"NodeToNodeEncryptionOptions,omitempty"`
	ServiceSoftwareOptions *AwsElasticsearchDomainServiceSoftwareOptions `json:"ServiceSoftwareOptions,omitempty"`
	VPCOptions *AwsElasticsearchDomainVPCOptions `json:"VPCOptions,omitempty"`
}

type AwsElasticsearchDomainDomainEndpointOptions struct {
	EnforceHTTPS bool `json:"EnforceHTTPS,omitempty"`
	TLSSecurityPolicy *string `json:"TLSSecurityPolicy,omitempty"`
}

type AwsElasticsearchDomainElasticsearchClusterConfigDetails struct {
	DedicatedMasterCount int `json:"DedicatedMasterCount,omitempty"`
	DedicatedMasterEnabled bool `json:"DedicatedMasterEnabled,omitempty"`
	DedicatedMasterType *string `json:"DedicatedMasterType,omitempty"`
	InstanceCount int `json:"InstanceCount,omitempty"`
	InstanceType *string `json:"InstanceType,omitempty"`
	ZoneAwarenessConfig *AwsElasticsearchDomainElasticsearchClusterConfigZoneAwarenessConfigDetails `json:"ZoneAwarenessConfig,omitempty"`
	ZoneAwarenessEnabled bool `json:"ZoneAwarenessEnabled,omitempty"`
}

type AwsElasticsearchDomainElasticsearchClusterConfigZoneAwarenessConfigDetails struct {
	AvailabilityZoneCount int `json:"AvailabilityZoneCount,omitempty"`
}

type AwsElasticsearchDomainEncryptionAtRestOptions struct {
	Enabled bool `json:"Enabled,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
}

type AwsElasticsearchDomainLogPublishingOptions struct {
	AuditLogs *AwsElasticsearchDomainLogPublishingOptionsLogConfig `json:"AuditLogs,omitempty"`
	IndexSlowLogs *AwsElasticsearchDomainLogPublishingOptionsLogConfig `json:"IndexSlowLogs,omitempty"`
	SearchSlowLogs *AwsElasticsearchDomainLogPublishingOptionsLogConfig `json:"SearchSlowLogs,omitempty"`
}

type AwsElasticsearchDomainLogPublishingOptionsLogConfig struct {
	CloudWatchLogsLogGroupArn *string `json:"CloudWatchLogsLogGroupArn,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsElasticsearchDomainNodeToNodeEncryptionOptions struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsElasticsearchDomainServiceSoftwareOptions struct {
	AutomatedUpdateDate *string `json:"AutomatedUpdateDate,omitempty"`
	Cancellable bool `json:"Cancellable,omitempty"`
	CurrentVersion *string `json:"CurrentVersion,omitempty"`
	Description *string `json:"Description,omitempty"`
	NewVersion *string `json:"NewVersion,omitempty"`
	UpdateAvailable bool `json:"UpdateAvailable,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
}

type AwsElasticsearchDomainVPCOptions struct {
	AvailabilityZones []string `json:"AvailabilityZones,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
	VPCId *string `json:"VPCId,omitempty"`
}

type AwsElbAppCookieStickinessPolicy struct {
	CookieName *string `json:"CookieName,omitempty"`
	PolicyName *string `json:"PolicyName,omitempty"`
}

type AwsElbLbCookieStickinessPolicy struct {
	CookieExpirationPeriod int64 `json:"CookieExpirationPeriod,omitempty"`
	PolicyName *string `json:"PolicyName,omitempty"`
}

type AwsElbLoadBalancerAccessLog struct {
	EmitInterval int `json:"EmitInterval,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	S3BucketName *string `json:"S3BucketName,omitempty"`
	S3BucketPrefix *string `json:"S3BucketPrefix,omitempty"`
}

type AwsElbLoadBalancerAdditionalAttribute struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsElbLoadBalancerAttributes struct {
	AccessLog *AwsElbLoadBalancerAccessLog `json:"AccessLog,omitempty"`
	AdditionalAttributes []AwsElbLoadBalancerAdditionalAttribute `json:"AdditionalAttributes,omitempty"`
	ConnectionDraining *AwsElbLoadBalancerConnectionDraining `json:"ConnectionDraining,omitempty"`
	ConnectionSettings *AwsElbLoadBalancerConnectionSettings `json:"ConnectionSettings,omitempty"`
	CrossZoneLoadBalancing *AwsElbLoadBalancerCrossZoneLoadBalancing `json:"CrossZoneLoadBalancing,omitempty"`
}

type AwsElbLoadBalancerBackendServerDescription struct {
	InstancePort int `json:"InstancePort,omitempty"`
	PolicyNames []string `json:"PolicyNames,omitempty"`
}

type AwsElbLoadBalancerConnectionDraining struct {
	Enabled bool `json:"Enabled,omitempty"`
	Timeout int `json:"Timeout,omitempty"`
}

type AwsElbLoadBalancerConnectionSettings struct {
	IdleTimeout int `json:"IdleTimeout,omitempty"`
}

type AwsElbLoadBalancerCrossZoneLoadBalancing struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsElbLoadBalancerDetails struct {
	AvailabilityZones []string `json:"AvailabilityZones,omitempty"`
	BackendServerDescriptions []AwsElbLoadBalancerBackendServerDescription `json:"BackendServerDescriptions,omitempty"`
	CanonicalHostedZoneName *string `json:"CanonicalHostedZoneName,omitempty"`
	CanonicalHostedZoneNameID *string `json:"CanonicalHostedZoneNameID,omitempty"`
	CreatedTime *string `json:"CreatedTime,omitempty"`
	DnsName *string `json:"DnsName,omitempty"`
	HealthCheck *AwsElbLoadBalancerHealthCheck `json:"HealthCheck,omitempty"`
	Instances []AwsElbLoadBalancerInstance `json:"Instances,omitempty"`
	ListenerDescriptions []AwsElbLoadBalancerListenerDescription `json:"ListenerDescriptions,omitempty"`
	LoadBalancerAttributes *AwsElbLoadBalancerAttributes `json:"LoadBalancerAttributes,omitempty"`
	LoadBalancerName *string `json:"LoadBalancerName,omitempty"`
	Policies *AwsElbLoadBalancerPolicies `json:"Policies,omitempty"`
	Scheme *string `json:"Scheme,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	SourceSecurityGroup *AwsElbLoadBalancerSourceSecurityGroup `json:"SourceSecurityGroup,omitempty"`
	Subnets []string `json:"Subnets,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsElbLoadBalancerHealthCheck struct {
	HealthyThreshold int `json:"HealthyThreshold,omitempty"`
	Interval int `json:"Interval,omitempty"`
	Target *string `json:"Target,omitempty"`
	Timeout int `json:"Timeout,omitempty"`
	UnhealthyThreshold int `json:"UnhealthyThreshold,omitempty"`
}

type AwsElbLoadBalancerInstance struct {
	InstanceId *string `json:"InstanceId,omitempty"`
}

type AwsElbLoadBalancerListener struct {
	InstancePort int `json:"InstancePort,omitempty"`
	InstanceProtocol *string `json:"InstanceProtocol,omitempty"`
	LoadBalancerPort int `json:"LoadBalancerPort,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	SslCertificateId *string `json:"SslCertificateId,omitempty"`
}

type AwsElbLoadBalancerListenerDescription struct {
	Listener *AwsElbLoadBalancerListener `json:"Listener,omitempty"`
	PolicyNames []string `json:"PolicyNames,omitempty"`
}

type AwsElbLoadBalancerPolicies struct {
	AppCookieStickinessPolicies []AwsElbAppCookieStickinessPolicy `json:"AppCookieStickinessPolicies,omitempty"`
	LbCookieStickinessPolicies []AwsElbLbCookieStickinessPolicy `json:"LbCookieStickinessPolicies,omitempty"`
	OtherPolicies []string `json:"OtherPolicies,omitempty"`
}

type AwsElbLoadBalancerSourceSecurityGroup struct {
	GroupName *string `json:"GroupName,omitempty"`
	OwnerAlias *string `json:"OwnerAlias,omitempty"`
}

type AwsElbv2LoadBalancerAttribute struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsElbv2LoadBalancerDetails struct {
	AvailabilityZones []AvailabilityZone `json:"AvailabilityZones,omitempty"`
	CanonicalHostedZoneId *string `json:"CanonicalHostedZoneId,omitempty"`
	CreatedTime *string `json:"CreatedTime,omitempty"`
	DNSName *string `json:"DNSName,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	LoadBalancerAttributes []AwsElbv2LoadBalancerAttribute `json:"LoadBalancerAttributes,omitempty"`
	Scheme *string `json:"Scheme,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	State *LoadBalancerState `json:"State,omitempty"`
	Type *string `json:"Type,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsEventSchemasRegistryDetails struct {
	Description *string `json:"Description,omitempty"`
	RegistryArn *string `json:"RegistryArn,omitempty"`
	RegistryName *string `json:"RegistryName,omitempty"`
}

type AwsEventsEndpointDetails struct {
	Arn *string `json:"Arn,omitempty"`
	Description *string `json:"Description,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
	EndpointUrl *string `json:"EndpointUrl,omitempty"`
	EventBuses []AwsEventsEndpointEventBusesDetails `json:"EventBuses,omitempty"`
	Name *string `json:"Name,omitempty"`
	ReplicationConfig *AwsEventsEndpointReplicationConfigDetails `json:"ReplicationConfig,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	RoutingConfig *AwsEventsEndpointRoutingConfigDetails `json:"RoutingConfig,omitempty"`
	State *string `json:"State,omitempty"`
	StateReason *string `json:"StateReason,omitempty"`
}

type AwsEventsEndpointEventBusesDetails struct {
	EventBusArn *string `json:"EventBusArn,omitempty"`
}

type AwsEventsEndpointReplicationConfigDetails struct {
	State *string `json:"State,omitempty"`
}

type AwsEventsEndpointRoutingConfigDetails struct {
	FailoverConfig *AwsEventsEndpointRoutingConfigFailoverConfigDetails `json:"FailoverConfig,omitempty"`
}

type AwsEventsEndpointRoutingConfigFailoverConfigDetails struct {
	Primary *AwsEventsEndpointRoutingConfigFailoverConfigPrimaryDetails `json:"Primary,omitempty"`
	Secondary *AwsEventsEndpointRoutingConfigFailoverConfigSecondaryDetails `json:"Secondary,omitempty"`
}

type AwsEventsEndpointRoutingConfigFailoverConfigPrimaryDetails struct {
	HealthCheck *string `json:"HealthCheck,omitempty"`
}

type AwsEventsEndpointRoutingConfigFailoverConfigSecondaryDetails struct {
	Route *string `json:"Route,omitempty"`
}

type AwsEventsEventbusDetails struct {
	Arn *string `json:"Arn,omitempty"`
	Name *string `json:"Name,omitempty"`
	Policy *string `json:"Policy,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesCloudTrailDetails struct {
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesDetails struct {
	CloudTrail *AwsGuardDutyDetectorDataSourcesCloudTrailDetails `json:"CloudTrail,omitempty"`
	DnsLogs *AwsGuardDutyDetectorDataSourcesDnsLogsDetails `json:"DnsLogs,omitempty"`
	FlowLogs *AwsGuardDutyDetectorDataSourcesFlowLogsDetails `json:"FlowLogs,omitempty"`
	Kubernetes *AwsGuardDutyDetectorDataSourcesKubernetesDetails `json:"Kubernetes,omitempty"`
	MalwareProtection *AwsGuardDutyDetectorDataSourcesMalwareProtectionDetails `json:"MalwareProtection,omitempty"`
	S3Logs *AwsGuardDutyDetectorDataSourcesS3LogsDetails `json:"S3Logs,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesDnsLogsDetails struct {
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesFlowLogsDetails struct {
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesKubernetesAuditLogsDetails struct {
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesKubernetesDetails struct {
	AuditLogs *AwsGuardDutyDetectorDataSourcesKubernetesAuditLogsDetails `json:"AuditLogs,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesMalwareProtectionDetails struct {
	ScanEc2InstanceWithFindings *AwsGuardDutyDetectorDataSourcesMalwareProtectionScanEc2InstanceWithFindingsDetails `json:"ScanEc2InstanceWithFindings,omitempty"`
	ServiceRole *string `json:"ServiceRole,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesMalwareProtectionScanEc2InstanceWithFindingsDetails struct {
	EbsVolumes *AwsGuardDutyDetectorDataSourcesMalwareProtectionScanEc2InstanceWithFindingsEbsVolumesDetails `json:"EbsVolumes,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesMalwareProtectionScanEc2InstanceWithFindingsEbsVolumesDetails struct {
	Reason *string `json:"Reason,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorDataSourcesS3LogsDetails struct {
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorDetails struct {
	DataSources *AwsGuardDutyDetectorDataSourcesDetails `json:"DataSources,omitempty"`
	Features []AwsGuardDutyDetectorFeaturesDetails `json:"Features,omitempty"`
	FindingPublishingFrequency *string `json:"FindingPublishingFrequency,omitempty"`
	ServiceRole *string `json:"ServiceRole,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsGuardDutyDetectorFeaturesDetails struct {
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsIamAccessKeyDetails struct {
	AccessKeyId *string `json:"AccessKeyId,omitempty"`
	AccountId *string `json:"AccountId,omitempty"`
	CreatedAt *string `json:"CreatedAt,omitempty"`
	PrincipalId *string `json:"PrincipalId,omitempty"`
	PrincipalName *string `json:"PrincipalName,omitempty"`
	PrincipalType *string `json:"PrincipalType,omitempty"`
	SessionContext *AwsIamAccessKeySessionContext `json:"SessionContext,omitempty"`
	Status *string `json:"Status,omitempty"`
	UserName *string `json:"UserName,omitempty"`
}

type AwsIamAccessKeySessionContext struct {
	Attributes *AwsIamAccessKeySessionContextAttributes `json:"Attributes,omitempty"`
	SessionIssuer *AwsIamAccessKeySessionContextSessionIssuer `json:"SessionIssuer,omitempty"`
}

type AwsIamAccessKeySessionContextAttributes struct {
	CreationDate *string `json:"CreationDate,omitempty"`
	MfaAuthenticated bool `json:"MfaAuthenticated,omitempty"`
}

type AwsIamAccessKeySessionContextSessionIssuer struct {
	AccountId *string `json:"AccountId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	PrincipalId *string `json:"PrincipalId,omitempty"`
	Type *string `json:"Type,omitempty"`
	UserName *string `json:"UserName,omitempty"`
}

type AwsIamAttachedManagedPolicy struct {
	PolicyArn *string `json:"PolicyArn,omitempty"`
	PolicyName *string `json:"PolicyName,omitempty"`
}

type AwsIamGroupDetails struct {
	AttachedManagedPolicies []AwsIamAttachedManagedPolicy `json:"AttachedManagedPolicies,omitempty"`
	CreateDate *string `json:"CreateDate,omitempty"`
	GroupId *string `json:"GroupId,omitempty"`
	GroupName *string `json:"GroupName,omitempty"`
	GroupPolicyList []AwsIamGroupPolicy `json:"GroupPolicyList,omitempty"`
	Path *string `json:"Path,omitempty"`
}

type AwsIamGroupPolicy struct {
	PolicyName *string `json:"PolicyName,omitempty"`
}

type AwsIamInstanceProfile struct {
	Arn *string `json:"Arn,omitempty"`
	CreateDate *string `json:"CreateDate,omitempty"`
	InstanceProfileId *string `json:"InstanceProfileId,omitempty"`
	InstanceProfileName *string `json:"InstanceProfileName,omitempty"`
	Path *string `json:"Path,omitempty"`
	Roles []AwsIamInstanceProfileRole `json:"Roles,omitempty"`
}

type AwsIamInstanceProfileRole struct {
	Arn *string `json:"Arn,omitempty"`
	AssumeRolePolicyDocument *string `json:"AssumeRolePolicyDocument,omitempty"`
	CreateDate *string `json:"CreateDate,omitempty"`
	Path *string `json:"Path,omitempty"`
	RoleId *string `json:"RoleId,omitempty"`
	RoleName *string `json:"RoleName,omitempty"`
}

type AwsIamPermissionsBoundary struct {
	PermissionsBoundaryArn *string `json:"PermissionsBoundaryArn,omitempty"`
	PermissionsBoundaryType *string `json:"PermissionsBoundaryType,omitempty"`
}

type AwsIamPolicyDetails struct {
	AttachmentCount int `json:"AttachmentCount,omitempty"`
	CreateDate *string `json:"CreateDate,omitempty"`
	DefaultVersionId *string `json:"DefaultVersionId,omitempty"`
	Description *string `json:"Description,omitempty"`
	IsAttachable bool `json:"IsAttachable,omitempty"`
	Path *string `json:"Path,omitempty"`
	PermissionsBoundaryUsageCount int `json:"PermissionsBoundaryUsageCount,omitempty"`
	PolicyId *string `json:"PolicyId,omitempty"`
	PolicyName *string `json:"PolicyName,omitempty"`
	PolicyVersionList []AwsIamPolicyVersion `json:"PolicyVersionList,omitempty"`
	UpdateDate *string `json:"UpdateDate,omitempty"`
}

type AwsIamPolicyVersion struct {
	CreateDate *string `json:"CreateDate,omitempty"`
	IsDefaultVersion bool `json:"IsDefaultVersion,omitempty"`
	VersionId *string `json:"VersionId,omitempty"`
}

type AwsIamRoleDetails struct {
	AssumeRolePolicyDocument *string `json:"AssumeRolePolicyDocument,omitempty"`
	AttachedManagedPolicies []AwsIamAttachedManagedPolicy `json:"AttachedManagedPolicies,omitempty"`
	CreateDate *string `json:"CreateDate,omitempty"`
	InstanceProfileList []AwsIamInstanceProfile `json:"InstanceProfileList,omitempty"`
	MaxSessionDuration int `json:"MaxSessionDuration,omitempty"`
	Path *string `json:"Path,omitempty"`
	PermissionsBoundary *AwsIamPermissionsBoundary `json:"PermissionsBoundary,omitempty"`
	RoleId *string `json:"RoleId,omitempty"`
	RoleName *string `json:"RoleName,omitempty"`
	RolePolicyList []AwsIamRolePolicy `json:"RolePolicyList,omitempty"`
}

type AwsIamRolePolicy struct {
	PolicyName *string `json:"PolicyName,omitempty"`
}

type AwsIamUserDetails struct {
	AttachedManagedPolicies []AwsIamAttachedManagedPolicy `json:"AttachedManagedPolicies,omitempty"`
	CreateDate *string `json:"CreateDate,omitempty"`
	GroupList []string `json:"GroupList,omitempty"`
	Path *string `json:"Path,omitempty"`
	PermissionsBoundary *AwsIamPermissionsBoundary `json:"PermissionsBoundary,omitempty"`
	UserId *string `json:"UserId,omitempty"`
	UserName *string `json:"UserName,omitempty"`
	UserPolicyList []AwsIamUserPolicy `json:"UserPolicyList,omitempty"`
}

type AwsIamUserPolicy struct {
	PolicyName *string `json:"PolicyName,omitempty"`
}

type AwsKinesisStreamDetails struct {
	Arn *string `json:"Arn,omitempty"`
	Name *string `json:"Name,omitempty"`
	RetentionPeriodHours int `json:"RetentionPeriodHours,omitempty"`
	ShardCount int `json:"ShardCount,omitempty"`
	StreamEncryption *AwsKinesisStreamStreamEncryptionDetails `json:"StreamEncryption,omitempty"`
}

type AwsKinesisStreamStreamEncryptionDetails struct {
	EncryptionType *string `json:"EncryptionType,omitempty"`
	KeyId *string `json:"KeyId,omitempty"`
}

type AwsKmsKeyDetails struct {
	AWSAccountId *string `json:"AWSAccountId,omitempty"`
	CreationDate float64 `json:"CreationDate,omitempty"`
	Description *string `json:"Description,omitempty"`
	KeyId *string `json:"KeyId,omitempty"`
	KeyManager *string `json:"KeyManager,omitempty"`
	KeyRotationStatus bool `json:"KeyRotationStatus,omitempty"`
	KeyState *string `json:"KeyState,omitempty"`
	Origin *string `json:"Origin,omitempty"`
}

type AwsLambdaFunctionCode struct {
	S3Bucket *string `json:"S3Bucket,omitempty"`
	S3Key *string `json:"S3Key,omitempty"`
	S3ObjectVersion *string `json:"S3ObjectVersion,omitempty"`
	ZipFile *string `json:"ZipFile,omitempty"`
}

type AwsLambdaFunctionDeadLetterConfig struct {
	TargetArn *string `json:"TargetArn,omitempty"`
}

type AwsLambdaFunctionDetails struct {
	Architectures []string `json:"Architectures,omitempty"`
	Code *AwsLambdaFunctionCode `json:"Code,omitempty"`
	CodeSha256 *string `json:"CodeSha256,omitempty"`
	DeadLetterConfig *AwsLambdaFunctionDeadLetterConfig `json:"DeadLetterConfig,omitempty"`
	Environment *AwsLambdaFunctionEnvironment `json:"Environment,omitempty"`
	FunctionName *string `json:"FunctionName,omitempty"`
	Handler *string `json:"Handler,omitempty"`
	KmsKeyArn *string `json:"KmsKeyArn,omitempty"`
	LastModified *string `json:"LastModified,omitempty"`
	Layers []AwsLambdaFunctionLayer `json:"Layers,omitempty"`
	MasterArn *string `json:"MasterArn,omitempty"`
	MemorySize int `json:"MemorySize,omitempty"`
	PackageType *string `json:"PackageType,omitempty"`
	RevisionId *string `json:"RevisionId,omitempty"`
	Role *string `json:"Role,omitempty"`
	Runtime *string `json:"Runtime,omitempty"`
	Timeout int `json:"Timeout,omitempty"`
	TracingConfig *AwsLambdaFunctionTracingConfig `json:"TracingConfig,omitempty"`
	Version *string `json:"Version,omitempty"`
	VpcConfig *AwsLambdaFunctionVpcConfig `json:"VpcConfig,omitempty"`
}

type AwsLambdaFunctionEnvironment struct {
	Error *AwsLambdaFunctionEnvironmentError `json:"Error,omitempty"`
	Variables map[string]string `json:"Variables,omitempty"`
}

type AwsLambdaFunctionEnvironmentError struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type AwsLambdaFunctionLayer struct {
	Arn *string `json:"Arn,omitempty"`
	CodeSize int `json:"CodeSize,omitempty"`
}

type AwsLambdaFunctionTracingConfig struct {
	Mode *string `json:"Mode,omitempty"`
}

type AwsLambdaFunctionVpcConfig struct {
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsLambdaLayerVersionDetails struct {
	CompatibleRuntimes []string `json:"CompatibleRuntimes,omitempty"`
	CreatedDate *string `json:"CreatedDate,omitempty"`
	Version int64 `json:"Version,omitempty"`
}

type AwsMountPoint struct {
	ContainerPath *string `json:"ContainerPath,omitempty"`
	SourceVolume *string `json:"SourceVolume,omitempty"`
}

type AwsMskClusterClusterInfoClientAuthenticationDetails struct {
	Sasl *AwsMskClusterClusterInfoClientAuthenticationSaslDetails `json:"Sasl,omitempty"`
	Tls *AwsMskClusterClusterInfoClientAuthenticationTlsDetails `json:"Tls,omitempty"`
	Unauthenticated *AwsMskClusterClusterInfoClientAuthenticationUnauthenticatedDetails `json:"Unauthenticated,omitempty"`
}

type AwsMskClusterClusterInfoClientAuthenticationSaslDetails struct {
	Iam *AwsMskClusterClusterInfoClientAuthenticationSaslIamDetails `json:"Iam,omitempty"`
	Scram *AwsMskClusterClusterInfoClientAuthenticationSaslScramDetails `json:"Scram,omitempty"`
}

type AwsMskClusterClusterInfoClientAuthenticationSaslIamDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsMskClusterClusterInfoClientAuthenticationSaslScramDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsMskClusterClusterInfoClientAuthenticationTlsDetails struct {
	CertificateAuthorityArnList []string `json:"CertificateAuthorityArnList,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsMskClusterClusterInfoClientAuthenticationUnauthenticatedDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsMskClusterClusterInfoDetails struct {
	ClientAuthentication *AwsMskClusterClusterInfoClientAuthenticationDetails `json:"ClientAuthentication,omitempty"`
	ClusterName *string `json:"ClusterName,omitempty"`
	CurrentVersion *string `json:"CurrentVersion,omitempty"`
	EncryptionInfo *AwsMskClusterClusterInfoEncryptionInfoDetails `json:"EncryptionInfo,omitempty"`
	EnhancedMonitoring *string `json:"EnhancedMonitoring,omitempty"`
	NumberOfBrokerNodes int `json:"NumberOfBrokerNodes,omitempty"`
}

type AwsMskClusterClusterInfoEncryptionInfoDetails struct {
	EncryptionAtRest *AwsMskClusterClusterInfoEncryptionInfoEncryptionAtRestDetails `json:"EncryptionAtRest,omitempty"`
	EncryptionInTransit *AwsMskClusterClusterInfoEncryptionInfoEncryptionInTransitDetails `json:"EncryptionInTransit,omitempty"`
}

type AwsMskClusterClusterInfoEncryptionInfoEncryptionAtRestDetails struct {
	DataVolumeKMSKeyId *string `json:"DataVolumeKMSKeyId,omitempty"`
}

type AwsMskClusterClusterInfoEncryptionInfoEncryptionInTransitDetails struct {
	ClientBroker *string `json:"ClientBroker,omitempty"`
	InCluster bool `json:"InCluster,omitempty"`
}

type AwsMskClusterDetails struct {
	ClusterInfo *AwsMskClusterClusterInfoDetails `json:"ClusterInfo,omitempty"`
}

type AwsNetworkFirewallFirewallDetails struct {
	DeleteProtection bool `json:"DeleteProtection,omitempty"`
	Description *string `json:"Description,omitempty"`
	FirewallArn *string `json:"FirewallArn,omitempty"`
	FirewallId *string `json:"FirewallId,omitempty"`
	FirewallName *string `json:"FirewallName,omitempty"`
	FirewallPolicyArn *string `json:"FirewallPolicyArn,omitempty"`
	FirewallPolicyChangeProtection bool `json:"FirewallPolicyChangeProtection,omitempty"`
	SubnetChangeProtection bool `json:"SubnetChangeProtection,omitempty"`
	SubnetMappings []AwsNetworkFirewallFirewallSubnetMappingsDetails `json:"SubnetMappings,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsNetworkFirewallFirewallPolicyDetails struct {
	Description *string `json:"Description,omitempty"`
	FirewallPolicy *FirewallPolicyDetails `json:"FirewallPolicy,omitempty"`
	FirewallPolicyArn *string `json:"FirewallPolicyArn,omitempty"`
	FirewallPolicyId *string `json:"FirewallPolicyId,omitempty"`
	FirewallPolicyName *string `json:"FirewallPolicyName,omitempty"`
}

type AwsNetworkFirewallFirewallSubnetMappingsDetails struct {
	SubnetId *string `json:"SubnetId,omitempty"`
}

type AwsNetworkFirewallRuleGroupDetails struct {
	Capacity int `json:"Capacity,omitempty"`
	Description *string `json:"Description,omitempty"`
	RuleGroup *RuleGroupDetails `json:"RuleGroup,omitempty"`
	RuleGroupArn *string `json:"RuleGroupArn,omitempty"`
	RuleGroupId *string `json:"RuleGroupId,omitempty"`
	RuleGroupName *string `json:"RuleGroupName,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsOpenSearchServiceDomainAdvancedSecurityOptionsDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
	InternalUserDatabaseEnabled bool `json:"InternalUserDatabaseEnabled,omitempty"`
	MasterUserOptions *AwsOpenSearchServiceDomainMasterUserOptionsDetails `json:"MasterUserOptions,omitempty"`
}

type AwsOpenSearchServiceDomainClusterConfigDetails struct {
	DedicatedMasterCount int `json:"DedicatedMasterCount,omitempty"`
	DedicatedMasterEnabled bool `json:"DedicatedMasterEnabled,omitempty"`
	DedicatedMasterType *string `json:"DedicatedMasterType,omitempty"`
	InstanceCount int `json:"InstanceCount,omitempty"`
	InstanceType *string `json:"InstanceType,omitempty"`
	WarmCount int `json:"WarmCount,omitempty"`
	WarmEnabled bool `json:"WarmEnabled,omitempty"`
	WarmType *string `json:"WarmType,omitempty"`
	ZoneAwarenessConfig *AwsOpenSearchServiceDomainClusterConfigZoneAwarenessConfigDetails `json:"ZoneAwarenessConfig,omitempty"`
	ZoneAwarenessEnabled bool `json:"ZoneAwarenessEnabled,omitempty"`
}

type AwsOpenSearchServiceDomainClusterConfigZoneAwarenessConfigDetails struct {
	AvailabilityZoneCount int `json:"AvailabilityZoneCount,omitempty"`
}

type AwsOpenSearchServiceDomainDetails struct {
	AccessPolicies *string `json:"AccessPolicies,omitempty"`
	AdvancedSecurityOptions *AwsOpenSearchServiceDomainAdvancedSecurityOptionsDetails `json:"AdvancedSecurityOptions,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	ClusterConfig *AwsOpenSearchServiceDomainClusterConfigDetails `json:"ClusterConfig,omitempty"`
	DomainEndpoint *string `json:"DomainEndpoint,omitempty"`
	DomainEndpointOptions *AwsOpenSearchServiceDomainDomainEndpointOptionsDetails `json:"DomainEndpointOptions,omitempty"`
	DomainEndpoints map[string]string `json:"DomainEndpoints,omitempty"`
	DomainName *string `json:"DomainName,omitempty"`
	EncryptionAtRestOptions *AwsOpenSearchServiceDomainEncryptionAtRestOptionsDetails `json:"EncryptionAtRestOptions,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	Id *string `json:"Id,omitempty"`
	LogPublishingOptions *AwsOpenSearchServiceDomainLogPublishingOptionsDetails `json:"LogPublishingOptions,omitempty"`
	NodeToNodeEncryptionOptions *AwsOpenSearchServiceDomainNodeToNodeEncryptionOptionsDetails `json:"NodeToNodeEncryptionOptions,omitempty"`
	ServiceSoftwareOptions *AwsOpenSearchServiceDomainServiceSoftwareOptionsDetails `json:"ServiceSoftwareOptions,omitempty"`
	VpcOptions *AwsOpenSearchServiceDomainVpcOptionsDetails `json:"VpcOptions,omitempty"`
}

type AwsOpenSearchServiceDomainDomainEndpointOptionsDetails struct {
	CustomEndpoint *string `json:"CustomEndpoint,omitempty"`
	CustomEndpointCertificateArn *string `json:"CustomEndpointCertificateArn,omitempty"`
	CustomEndpointEnabled bool `json:"CustomEndpointEnabled,omitempty"`
	EnforceHTTPS bool `json:"EnforceHTTPS,omitempty"`
	TLSSecurityPolicy *string `json:"TLSSecurityPolicy,omitempty"`
}

type AwsOpenSearchServiceDomainEncryptionAtRestOptionsDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
}

type AwsOpenSearchServiceDomainLogPublishingOption struct {
	CloudWatchLogsLogGroupArn *string `json:"CloudWatchLogsLogGroupArn,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsOpenSearchServiceDomainLogPublishingOptionsDetails struct {
	AuditLogs *AwsOpenSearchServiceDomainLogPublishingOption `json:"AuditLogs,omitempty"`
	IndexSlowLogs *AwsOpenSearchServiceDomainLogPublishingOption `json:"IndexSlowLogs,omitempty"`
	SearchSlowLogs *AwsOpenSearchServiceDomainLogPublishingOption `json:"SearchSlowLogs,omitempty"`
}

type AwsOpenSearchServiceDomainMasterUserOptionsDetails struct {
	MasterUserArn *string `json:"MasterUserArn,omitempty"`
	MasterUserName *string `json:"MasterUserName,omitempty"`
	MasterUserPassword *string `json:"MasterUserPassword,omitempty"`
}

type AwsOpenSearchServiceDomainNodeToNodeEncryptionOptionsDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsOpenSearchServiceDomainServiceSoftwareOptionsDetails struct {
	AutomatedUpdateDate *string `json:"AutomatedUpdateDate,omitempty"`
	Cancellable bool `json:"Cancellable,omitempty"`
	CurrentVersion *string `json:"CurrentVersion,omitempty"`
	Description *string `json:"Description,omitempty"`
	NewVersion *string `json:"NewVersion,omitempty"`
	OptionalDeployment bool `json:"OptionalDeployment,omitempty"`
	UpdateAvailable bool `json:"UpdateAvailable,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
}

type AwsOpenSearchServiceDomainVpcOptionsDetails struct {
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
}

type AwsRdsDbClusterAssociatedRole struct {
	RoleArn *string `json:"RoleArn,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbClusterDetails struct {
	ActivityStreamStatus *string `json:"ActivityStreamStatus,omitempty"`
	AllocatedStorage int `json:"AllocatedStorage,omitempty"`
	AssociatedRoles []AwsRdsDbClusterAssociatedRole `json:"AssociatedRoles,omitempty"`
	AutoMinorVersionUpgrade bool `json:"AutoMinorVersionUpgrade,omitempty"`
	AvailabilityZones []string `json:"AvailabilityZones,omitempty"`
	BackupRetentionPeriod int `json:"BackupRetentionPeriod,omitempty"`
	ClusterCreateTime *string `json:"ClusterCreateTime,omitempty"`
	CopyTagsToSnapshot bool `json:"CopyTagsToSnapshot,omitempty"`
	CrossAccountClone bool `json:"CrossAccountClone,omitempty"`
	CustomEndpoints []string `json:"CustomEndpoints,omitempty"`
	DatabaseName *string `json:"DatabaseName,omitempty"`
	DbClusterIdentifier *string `json:"DbClusterIdentifier,omitempty"`
	DbClusterMembers []AwsRdsDbClusterMember `json:"DbClusterMembers,omitempty"`
	DbClusterOptionGroupMemberships []AwsRdsDbClusterOptionGroupMembership `json:"DbClusterOptionGroupMemberships,omitempty"`
	DbClusterParameterGroup *string `json:"DbClusterParameterGroup,omitempty"`
	DbClusterResourceId *string `json:"DbClusterResourceId,omitempty"`
	DbSubnetGroup *string `json:"DbSubnetGroup,omitempty"`
	DeletionProtection bool `json:"DeletionProtection,omitempty"`
	DomainMemberships []AwsRdsDbDomainMembership `json:"DomainMemberships,omitempty"`
	EnabledCloudWatchLogsExports []string `json:"EnabledCloudWatchLogsExports,omitempty"`
	Endpoint *string `json:"Endpoint,omitempty"`
	Engine *string `json:"Engine,omitempty"`
	EngineMode *string `json:"EngineMode,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	HostedZoneId *string `json:"HostedZoneId,omitempty"`
	HttpEndpointEnabled bool `json:"HttpEndpointEnabled,omitempty"`
	IamDatabaseAuthenticationEnabled bool `json:"IamDatabaseAuthenticationEnabled,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	MasterUsername *string `json:"MasterUsername,omitempty"`
	MultiAz bool `json:"MultiAz,omitempty"`
	Port int `json:"Port,omitempty"`
	PreferredBackupWindow *string `json:"PreferredBackupWindow,omitempty"`
	PreferredMaintenanceWindow *string `json:"PreferredMaintenanceWindow,omitempty"`
	ReadReplicaIdentifiers []string `json:"ReadReplicaIdentifiers,omitempty"`
	ReaderEndpoint *string `json:"ReaderEndpoint,omitempty"`
	Status *string `json:"Status,omitempty"`
	StorageEncrypted bool `json:"StorageEncrypted,omitempty"`
	VpcSecurityGroups []AwsRdsDbInstanceVpcSecurityGroup `json:"VpcSecurityGroups,omitempty"`
}

type AwsRdsDbClusterMember struct {
	DbClusterParameterGroupStatus *string `json:"DbClusterParameterGroupStatus,omitempty"`
	DbInstanceIdentifier *string `json:"DbInstanceIdentifier,omitempty"`
	IsClusterWriter bool `json:"IsClusterWriter,omitempty"`
	PromotionTier int `json:"PromotionTier,omitempty"`
}

type AwsRdsDbClusterOptionGroupMembership struct {
	DbClusterOptionGroupName *string `json:"DbClusterOptionGroupName,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbClusterSnapshotDbClusterSnapshotAttribute struct {
	AttributeName *string `json:"AttributeName,omitempty"`
	AttributeValues []string `json:"AttributeValues,omitempty"`
}

type AwsRdsDbClusterSnapshotDetails struct {
	AllocatedStorage int `json:"AllocatedStorage,omitempty"`
	AvailabilityZones []string `json:"AvailabilityZones,omitempty"`
	ClusterCreateTime *string `json:"ClusterCreateTime,omitempty"`
	DbClusterIdentifier *string `json:"DbClusterIdentifier,omitempty"`
	DbClusterSnapshotAttributes []AwsRdsDbClusterSnapshotDbClusterSnapshotAttribute `json:"DbClusterSnapshotAttributes,omitempty"`
	DbClusterSnapshotIdentifier *string `json:"DbClusterSnapshotIdentifier,omitempty"`
	Engine *string `json:"Engine,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	IamDatabaseAuthenticationEnabled bool `json:"IamDatabaseAuthenticationEnabled,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LicenseModel *string `json:"LicenseModel,omitempty"`
	MasterUsername *string `json:"MasterUsername,omitempty"`
	PercentProgress int `json:"PercentProgress,omitempty"`
	Port int `json:"Port,omitempty"`
	SnapshotCreateTime *string `json:"SnapshotCreateTime,omitempty"`
	SnapshotType *string `json:"SnapshotType,omitempty"`
	Status *string `json:"Status,omitempty"`
	StorageEncrypted bool `json:"StorageEncrypted,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsRdsDbDomainMembership struct {
	Domain *string `json:"Domain,omitempty"`
	Fqdn *string `json:"Fqdn,omitempty"`
	IamRoleName *string `json:"IamRoleName,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbInstanceAssociatedRole struct {
	FeatureName *string `json:"FeatureName,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbInstanceDetails struct {
	AllocatedStorage int `json:"AllocatedStorage,omitempty"`
	AssociatedRoles []AwsRdsDbInstanceAssociatedRole `json:"AssociatedRoles,omitempty"`
	AutoMinorVersionUpgrade bool `json:"AutoMinorVersionUpgrade,omitempty"`
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	BackupRetentionPeriod int `json:"BackupRetentionPeriod,omitempty"`
	CACertificateIdentifier *string `json:"CACertificateIdentifier,omitempty"`
	CharacterSetName *string `json:"CharacterSetName,omitempty"`
	CopyTagsToSnapshot bool `json:"CopyTagsToSnapshot,omitempty"`
	DBClusterIdentifier *string `json:"DBClusterIdentifier,omitempty"`
	DBInstanceClass *string `json:"DBInstanceClass,omitempty"`
	DBInstanceIdentifier *string `json:"DBInstanceIdentifier,omitempty"`
	DBName *string `json:"DBName,omitempty"`
	DbInstancePort int `json:"DbInstancePort,omitempty"`
	DbInstanceStatus *string `json:"DbInstanceStatus,omitempty"`
	DbParameterGroups []AwsRdsDbParameterGroup `json:"DbParameterGroups,omitempty"`
	DbSecurityGroups []string `json:"DbSecurityGroups,omitempty"`
	DbSubnetGroup *AwsRdsDbSubnetGroup `json:"DbSubnetGroup,omitempty"`
	DbiResourceId *string `json:"DbiResourceId,omitempty"`
	DeletionProtection bool `json:"DeletionProtection,omitempty"`
	DomainMemberships []AwsRdsDbDomainMembership `json:"DomainMemberships,omitempty"`
	EnabledCloudWatchLogsExports []string `json:"EnabledCloudWatchLogsExports,omitempty"`
	Endpoint *AwsRdsDbInstanceEndpoint `json:"Endpoint,omitempty"`
	Engine *string `json:"Engine,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	EnhancedMonitoringResourceArn *string `json:"EnhancedMonitoringResourceArn,omitempty"`
	IAMDatabaseAuthenticationEnabled bool `json:"IAMDatabaseAuthenticationEnabled,omitempty"`
	InstanceCreateTime *string `json:"InstanceCreateTime,omitempty"`
	Iops int `json:"Iops,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LatestRestorableTime *string `json:"LatestRestorableTime,omitempty"`
	LicenseModel *string `json:"LicenseModel,omitempty"`
	ListenerEndpoint *AwsRdsDbInstanceEndpoint `json:"ListenerEndpoint,omitempty"`
	MasterUsername *string `json:"MasterUsername,omitempty"`
	MaxAllocatedStorage int `json:"MaxAllocatedStorage,omitempty"`
	MonitoringInterval int `json:"MonitoringInterval,omitempty"`
	MonitoringRoleArn *string `json:"MonitoringRoleArn,omitempty"`
	MultiAz bool `json:"MultiAz,omitempty"`
	OptionGroupMemberships []AwsRdsDbOptionGroupMembership `json:"OptionGroupMemberships,omitempty"`
	PendingModifiedValues *AwsRdsDbPendingModifiedValues `json:"PendingModifiedValues,omitempty"`
	PerformanceInsightsEnabled bool `json:"PerformanceInsightsEnabled,omitempty"`
	PerformanceInsightsKmsKeyId *string `json:"PerformanceInsightsKmsKeyId,omitempty"`
	PerformanceInsightsRetentionPeriod int `json:"PerformanceInsightsRetentionPeriod,omitempty"`
	PreferredBackupWindow *string `json:"PreferredBackupWindow,omitempty"`
	PreferredMaintenanceWindow *string `json:"PreferredMaintenanceWindow,omitempty"`
	ProcessorFeatures []AwsRdsDbProcessorFeature `json:"ProcessorFeatures,omitempty"`
	PromotionTier int `json:"PromotionTier,omitempty"`
	PubliclyAccessible bool `json:"PubliclyAccessible,omitempty"`
	ReadReplicaDBClusterIdentifiers []string `json:"ReadReplicaDBClusterIdentifiers,omitempty"`
	ReadReplicaDBInstanceIdentifiers []string `json:"ReadReplicaDBInstanceIdentifiers,omitempty"`
	ReadReplicaSourceDBInstanceIdentifier *string `json:"ReadReplicaSourceDBInstanceIdentifier,omitempty"`
	SecondaryAvailabilityZone *string `json:"SecondaryAvailabilityZone,omitempty"`
	StatusInfos []AwsRdsDbStatusInfo `json:"StatusInfos,omitempty"`
	StorageEncrypted bool `json:"StorageEncrypted,omitempty"`
	StorageType *string `json:"StorageType,omitempty"`
	TdeCredentialArn *string `json:"TdeCredentialArn,omitempty"`
	Timezone *string `json:"Timezone,omitempty"`
	VpcSecurityGroups []AwsRdsDbInstanceVpcSecurityGroup `json:"VpcSecurityGroups,omitempty"`
}

type AwsRdsDbInstanceEndpoint struct {
	Address *string `json:"Address,omitempty"`
	HostedZoneId *string `json:"HostedZoneId,omitempty"`
	Port int `json:"Port,omitempty"`
}

type AwsRdsDbInstanceVpcSecurityGroup struct {
	Status *string `json:"Status,omitempty"`
	VpcSecurityGroupId *string `json:"VpcSecurityGroupId,omitempty"`
}

type AwsRdsDbOptionGroupMembership struct {
	OptionGroupName *string `json:"OptionGroupName,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbParameterGroup struct {
	DbParameterGroupName *string `json:"DbParameterGroupName,omitempty"`
	ParameterApplyStatus *string `json:"ParameterApplyStatus,omitempty"`
}

type AwsRdsDbPendingModifiedValues struct {
	AllocatedStorage int `json:"AllocatedStorage,omitempty"`
	BackupRetentionPeriod int `json:"BackupRetentionPeriod,omitempty"`
	CaCertificateIdentifier *string `json:"CaCertificateIdentifier,omitempty"`
	DbInstanceClass *string `json:"DbInstanceClass,omitempty"`
	DbInstanceIdentifier *string `json:"DbInstanceIdentifier,omitempty"`
	DbSubnetGroupName *string `json:"DbSubnetGroupName,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	Iops int `json:"Iops,omitempty"`
	LicenseModel *string `json:"LicenseModel,omitempty"`
	MasterUserPassword *string `json:"MasterUserPassword,omitempty"`
	MultiAZ bool `json:"MultiAZ,omitempty"`
	PendingCloudWatchLogsExports *AwsRdsPendingCloudWatchLogsExports `json:"PendingCloudWatchLogsExports,omitempty"`
	Port int `json:"Port,omitempty"`
	ProcessorFeatures []AwsRdsDbProcessorFeature `json:"ProcessorFeatures,omitempty"`
	StorageType *string `json:"StorageType,omitempty"`
}

type AwsRdsDbProcessorFeature struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsRdsDbSecurityGroupDetails struct {
	DbSecurityGroupArn *string `json:"DbSecurityGroupArn,omitempty"`
	DbSecurityGroupDescription *string `json:"DbSecurityGroupDescription,omitempty"`
	DbSecurityGroupName *string `json:"DbSecurityGroupName,omitempty"`
	Ec2SecurityGroups []AwsRdsDbSecurityGroupEc2SecurityGroup `json:"Ec2SecurityGroups,omitempty"`
	IpRanges []AwsRdsDbSecurityGroupIpRange `json:"IpRanges,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsRdsDbSecurityGroupEc2SecurityGroup struct {
	Ec2SecurityGroupId *string `json:"Ec2SecurityGroupId,omitempty"`
	Ec2SecurityGroupName *string `json:"Ec2SecurityGroupName,omitempty"`
	Ec2SecurityGroupOwnerId *string `json:"Ec2SecurityGroupOwnerId,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbSecurityGroupIpRange struct {
	CidrIp *string `json:"CidrIp,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRdsDbSnapshotDetails struct {
	AllocatedStorage int `json:"AllocatedStorage,omitempty"`
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	DbInstanceIdentifier *string `json:"DbInstanceIdentifier,omitempty"`
	DbSnapshotIdentifier *string `json:"DbSnapshotIdentifier,omitempty"`
	DbiResourceId *string `json:"DbiResourceId,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	Engine *string `json:"Engine,omitempty"`
	EngineVersion *string `json:"EngineVersion,omitempty"`
	IamDatabaseAuthenticationEnabled bool `json:"IamDatabaseAuthenticationEnabled,omitempty"`
	InstanceCreateTime *string `json:"InstanceCreateTime,omitempty"`
	Iops int `json:"Iops,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LicenseModel *string `json:"LicenseModel,omitempty"`
	MasterUsername *string `json:"MasterUsername,omitempty"`
	OptionGroupName *string `json:"OptionGroupName,omitempty"`
	PercentProgress int `json:"PercentProgress,omitempty"`
	Port int `json:"Port,omitempty"`
	ProcessorFeatures []AwsRdsDbProcessorFeature `json:"ProcessorFeatures,omitempty"`
	SnapshotCreateTime *string `json:"SnapshotCreateTime,omitempty"`
	SnapshotType *string `json:"SnapshotType,omitempty"`
	SourceDbSnapshotIdentifier *string `json:"SourceDbSnapshotIdentifier,omitempty"`
	SourceRegion *string `json:"SourceRegion,omitempty"`
	Status *string `json:"Status,omitempty"`
	StorageType *string `json:"StorageType,omitempty"`
	TdeCredentialArn *string `json:"TdeCredentialArn,omitempty"`
	Timezone *string `json:"Timezone,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsRdsDbStatusInfo struct {
	Message *string `json:"Message,omitempty"`
	Normal bool `json:"Normal,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusType *string `json:"StatusType,omitempty"`
}

type AwsRdsDbSubnetGroup struct {
	DbSubnetGroupArn *string `json:"DbSubnetGroupArn,omitempty"`
	DbSubnetGroupDescription *string `json:"DbSubnetGroupDescription,omitempty"`
	DbSubnetGroupName *string `json:"DbSubnetGroupName,omitempty"`
	SubnetGroupStatus *string `json:"SubnetGroupStatus,omitempty"`
	Subnets []AwsRdsDbSubnetGroupSubnet `json:"Subnets,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsRdsDbSubnetGroupSubnet struct {
	SubnetAvailabilityZone *AwsRdsDbSubnetGroupSubnetAvailabilityZone `json:"SubnetAvailabilityZone,omitempty"`
	SubnetIdentifier *string `json:"SubnetIdentifier,omitempty"`
	SubnetStatus *string `json:"SubnetStatus,omitempty"`
}

type AwsRdsDbSubnetGroupSubnetAvailabilityZone struct {
	Name *string `json:"Name,omitempty"`
}

type AwsRdsEventSubscriptionDetails struct {
	CustSubscriptionId *string `json:"CustSubscriptionId,omitempty"`
	CustomerAwsId *string `json:"CustomerAwsId,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	EventCategoriesList []string `json:"EventCategoriesList,omitempty"`
	EventSubscriptionArn *string `json:"EventSubscriptionArn,omitempty"`
	SnsTopicArn *string `json:"SnsTopicArn,omitempty"`
	SourceIdsList []string `json:"SourceIdsList,omitempty"`
	SourceType *string `json:"SourceType,omitempty"`
	Status *string `json:"Status,omitempty"`
	SubscriptionCreationTime *string `json:"SubscriptionCreationTime,omitempty"`
}

type AwsRdsPendingCloudWatchLogsExports struct {
	LogTypesToDisable []string `json:"LogTypesToDisable,omitempty"`
	LogTypesToEnable []string `json:"LogTypesToEnable,omitempty"`
}

type AwsRedshiftClusterClusterNode struct {
	NodeRole *string `json:"NodeRole,omitempty"`
	PrivateIpAddress *string `json:"PrivateIpAddress,omitempty"`
	PublicIpAddress *string `json:"PublicIpAddress,omitempty"`
}

type AwsRedshiftClusterClusterParameterGroup struct {
	ClusterParameterStatusList []AwsRedshiftClusterClusterParameterStatus `json:"ClusterParameterStatusList,omitempty"`
	ParameterApplyStatus *string `json:"ParameterApplyStatus,omitempty"`
	ParameterGroupName *string `json:"ParameterGroupName,omitempty"`
}

type AwsRedshiftClusterClusterParameterStatus struct {
	ParameterApplyErrorDescription *string `json:"ParameterApplyErrorDescription,omitempty"`
	ParameterApplyStatus *string `json:"ParameterApplyStatus,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
}

type AwsRedshiftClusterClusterSecurityGroup struct {
	ClusterSecurityGroupName *string `json:"ClusterSecurityGroupName,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRedshiftClusterClusterSnapshotCopyStatus struct {
	DestinationRegion *string `json:"DestinationRegion,omitempty"`
	ManualSnapshotRetentionPeriod int `json:"ManualSnapshotRetentionPeriod,omitempty"`
	RetentionPeriod int `json:"RetentionPeriod,omitempty"`
	SnapshotCopyGrantName *string `json:"SnapshotCopyGrantName,omitempty"`
}

type AwsRedshiftClusterDeferredMaintenanceWindow struct {
	DeferMaintenanceEndTime *string `json:"DeferMaintenanceEndTime,omitempty"`
	DeferMaintenanceIdentifier *string `json:"DeferMaintenanceIdentifier,omitempty"`
	DeferMaintenanceStartTime *string `json:"DeferMaintenanceStartTime,omitempty"`
}

type AwsRedshiftClusterDetails struct {
	AllowVersionUpgrade bool `json:"AllowVersionUpgrade,omitempty"`
	AutomatedSnapshotRetentionPeriod int `json:"AutomatedSnapshotRetentionPeriod,omitempty"`
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	ClusterAvailabilityStatus *string `json:"ClusterAvailabilityStatus,omitempty"`
	ClusterCreateTime *string `json:"ClusterCreateTime,omitempty"`
	ClusterIdentifier *string `json:"ClusterIdentifier,omitempty"`
	ClusterNodes []AwsRedshiftClusterClusterNode `json:"ClusterNodes,omitempty"`
	ClusterParameterGroups []AwsRedshiftClusterClusterParameterGroup `json:"ClusterParameterGroups,omitempty"`
	ClusterPublicKey *string `json:"ClusterPublicKey,omitempty"`
	ClusterRevisionNumber *string `json:"ClusterRevisionNumber,omitempty"`
	ClusterSecurityGroups []AwsRedshiftClusterClusterSecurityGroup `json:"ClusterSecurityGroups,omitempty"`
	ClusterSnapshotCopyStatus *AwsRedshiftClusterClusterSnapshotCopyStatus `json:"ClusterSnapshotCopyStatus,omitempty"`
	ClusterStatus *string `json:"ClusterStatus,omitempty"`
	ClusterSubnetGroupName *string `json:"ClusterSubnetGroupName,omitempty"`
	ClusterVersion *string `json:"ClusterVersion,omitempty"`
	DBName *string `json:"DBName,omitempty"`
	DeferredMaintenanceWindows []AwsRedshiftClusterDeferredMaintenanceWindow `json:"DeferredMaintenanceWindows,omitempty"`
	ElasticIpStatus *AwsRedshiftClusterElasticIpStatus `json:"ElasticIpStatus,omitempty"`
	ElasticResizeNumberOfNodeOptions *string `json:"ElasticResizeNumberOfNodeOptions,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	Endpoint *AwsRedshiftClusterEndpoint `json:"Endpoint,omitempty"`
	EnhancedVpcRouting bool `json:"EnhancedVpcRouting,omitempty"`
	ExpectedNextSnapshotScheduleTime *string `json:"ExpectedNextSnapshotScheduleTime,omitempty"`
	ExpectedNextSnapshotScheduleTimeStatus *string `json:"ExpectedNextSnapshotScheduleTimeStatus,omitempty"`
	HsmStatus *AwsRedshiftClusterHsmStatus `json:"HsmStatus,omitempty"`
	IamRoles []AwsRedshiftClusterIamRole `json:"IamRoles,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LoggingStatus *AwsRedshiftClusterLoggingStatus `json:"LoggingStatus,omitempty"`
	MaintenanceTrackName *string `json:"MaintenanceTrackName,omitempty"`
	ManualSnapshotRetentionPeriod int `json:"ManualSnapshotRetentionPeriod,omitempty"`
	MasterUsername *string `json:"MasterUsername,omitempty"`
	NextMaintenanceWindowStartTime *string `json:"NextMaintenanceWindowStartTime,omitempty"`
	NodeType *string `json:"NodeType,omitempty"`
	NumberOfNodes int `json:"NumberOfNodes,omitempty"`
	PendingActions []string `json:"PendingActions,omitempty"`
	PendingModifiedValues *AwsRedshiftClusterPendingModifiedValues `json:"PendingModifiedValues,omitempty"`
	PreferredMaintenanceWindow *string `json:"PreferredMaintenanceWindow,omitempty"`
	PubliclyAccessible bool `json:"PubliclyAccessible,omitempty"`
	ResizeInfo *AwsRedshiftClusterResizeInfo `json:"ResizeInfo,omitempty"`
	RestoreStatus *AwsRedshiftClusterRestoreStatus `json:"RestoreStatus,omitempty"`
	SnapshotScheduleIdentifier *string `json:"SnapshotScheduleIdentifier,omitempty"`
	SnapshotScheduleState *string `json:"SnapshotScheduleState,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
	VpcSecurityGroups []AwsRedshiftClusterVpcSecurityGroup `json:"VpcSecurityGroups,omitempty"`
}

type AwsRedshiftClusterElasticIpStatus struct {
	ElasticIp *string `json:"ElasticIp,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRedshiftClusterEndpoint struct {
	Address *string `json:"Address,omitempty"`
	Port int `json:"Port,omitempty"`
}

type AwsRedshiftClusterHsmStatus struct {
	HsmClientCertificateIdentifier *string `json:"HsmClientCertificateIdentifier,omitempty"`
	HsmConfigurationIdentifier *string `json:"HsmConfigurationIdentifier,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRedshiftClusterIamRole struct {
	ApplyStatus *string `json:"ApplyStatus,omitempty"`
	IamRoleArn *string `json:"IamRoleArn,omitempty"`
}

type AwsRedshiftClusterLoggingStatus struct {
	BucketName *string `json:"BucketName,omitempty"`
	LastFailureMessage *string `json:"LastFailureMessage,omitempty"`
	LastFailureTime *string `json:"LastFailureTime,omitempty"`
	LastSuccessfulDeliveryTime *string `json:"LastSuccessfulDeliveryTime,omitempty"`
	LoggingEnabled bool `json:"LoggingEnabled,omitempty"`
	S3KeyPrefix *string `json:"S3KeyPrefix,omitempty"`
}

type AwsRedshiftClusterPendingModifiedValues struct {
	AutomatedSnapshotRetentionPeriod int `json:"AutomatedSnapshotRetentionPeriod,omitempty"`
	ClusterIdentifier *string `json:"ClusterIdentifier,omitempty"`
	ClusterType *string `json:"ClusterType,omitempty"`
	ClusterVersion *string `json:"ClusterVersion,omitempty"`
	EncryptionType *string `json:"EncryptionType,omitempty"`
	EnhancedVpcRouting bool `json:"EnhancedVpcRouting,omitempty"`
	MaintenanceTrackName *string `json:"MaintenanceTrackName,omitempty"`
	MasterUserPassword *string `json:"MasterUserPassword,omitempty"`
	NodeType *string `json:"NodeType,omitempty"`
	NumberOfNodes int `json:"NumberOfNodes,omitempty"`
	PubliclyAccessible bool `json:"PubliclyAccessible,omitempty"`
}

type AwsRedshiftClusterResizeInfo struct {
	AllowCancelResize bool `json:"AllowCancelResize,omitempty"`
	ResizeType *string `json:"ResizeType,omitempty"`
}

type AwsRedshiftClusterRestoreStatus struct {
	CurrentRestoreRateInMegaBytesPerSecond float64 `json:"CurrentRestoreRateInMegaBytesPerSecond,omitempty"`
	ElapsedTimeInSeconds int64 `json:"ElapsedTimeInSeconds,omitempty"`
	EstimatedTimeToCompletionInSeconds int64 `json:"EstimatedTimeToCompletionInSeconds,omitempty"`
	ProgressInMegaBytes int64 `json:"ProgressInMegaBytes,omitempty"`
	SnapshotSizeInMegaBytes int64 `json:"SnapshotSizeInMegaBytes,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsRedshiftClusterVpcSecurityGroup struct {
	Status *string `json:"Status,omitempty"`
	VpcSecurityGroupId *string `json:"VpcSecurityGroupId,omitempty"`
}

type AwsRoute53HostedZoneConfigDetails struct {
	Comment *string `json:"Comment,omitempty"`
}

type AwsRoute53HostedZoneDetails struct {
	HostedZone *AwsRoute53HostedZoneObjectDetails `json:"HostedZone,omitempty"`
	NameServers []string `json:"NameServers,omitempty"`
	QueryLoggingConfig *AwsRoute53QueryLoggingConfigDetails `json:"QueryLoggingConfig,omitempty"`
	Vpcs []AwsRoute53HostedZoneVpcDetails `json:"Vpcs,omitempty"`
}

type AwsRoute53HostedZoneObjectDetails struct {
	Config *AwsRoute53HostedZoneConfigDetails `json:"Config,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AwsRoute53HostedZoneVpcDetails struct {
	Id *string `json:"Id,omitempty"`
	Region *string `json:"Region,omitempty"`
}

type AwsRoute53QueryLoggingConfigDetails struct {
	CloudWatchLogsLogGroupArn *CloudWatchLogsLogGroupArnConfigDetails `json:"CloudWatchLogsLogGroupArn,omitempty"`
}

type AwsS3AccessPointDetails struct {
	AccessPointArn *string `json:"AccessPointArn,omitempty"`
	Alias *string `json:"Alias,omitempty"`
	Bucket *string `json:"Bucket,omitempty"`
	BucketAccountId *string `json:"BucketAccountId,omitempty"`
	Name *string `json:"Name,omitempty"`
	NetworkOrigin *string `json:"NetworkOrigin,omitempty"`
	PublicAccessBlockConfiguration *AwsS3AccountPublicAccessBlockDetails `json:"PublicAccessBlockConfiguration,omitempty"`
	VpcConfiguration *AwsS3AccessPointVpcConfigurationDetails `json:"VpcConfiguration,omitempty"`
}

type AwsS3AccessPointVpcConfigurationDetails struct {
	VpcId *string `json:"VpcId,omitempty"`
}

type AwsS3AccountPublicAccessBlockDetails struct {
	BlockPublicAcls bool `json:"BlockPublicAcls,omitempty"`
	BlockPublicPolicy bool `json:"BlockPublicPolicy,omitempty"`
	IgnorePublicAcls bool `json:"IgnorePublicAcls,omitempty"`
	RestrictPublicBuckets bool `json:"RestrictPublicBuckets,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationDetails struct {
	Rules []AwsS3BucketBucketLifecycleConfigurationRulesDetails `json:"Rules,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesAbortIncompleteMultipartUploadDetails struct {
	DaysAfterInitiation int `json:"DaysAfterInitiation,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesDetails struct {
	AbortIncompleteMultipartUpload *AwsS3BucketBucketLifecycleConfigurationRulesAbortIncompleteMultipartUploadDetails `json:"AbortIncompleteMultipartUpload,omitempty"`
	ExpirationDate *string `json:"ExpirationDate,omitempty"`
	ExpirationInDays int `json:"ExpirationInDays,omitempty"`
	ExpiredObjectDeleteMarker bool `json:"ExpiredObjectDeleteMarker,omitempty"`
	Filter *AwsS3BucketBucketLifecycleConfigurationRulesFilterDetails `json:"Filter,omitempty"`
	ID *string `json:"ID,omitempty"`
	NoncurrentVersionExpirationInDays int `json:"NoncurrentVersionExpirationInDays,omitempty"`
	NoncurrentVersionTransitions []AwsS3BucketBucketLifecycleConfigurationRulesNoncurrentVersionTransitionsDetails `json:"NoncurrentVersionTransitions,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
	Status *string `json:"Status,omitempty"`
	Transitions []AwsS3BucketBucketLifecycleConfigurationRulesTransitionsDetails `json:"Transitions,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesFilterDetails struct {
	Predicate *AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateDetails `json:"Predicate,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateDetails struct {
	Operands []AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateOperandsDetails `json:"Operands,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
	Tag *AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateTagDetails `json:"Tag,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateOperandsDetails struct {
	Prefix *string `json:"Prefix,omitempty"`
	Tag *AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateOperandsTagDetails `json:"Tag,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateOperandsTagDetails struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesFilterPredicateTagDetails struct {
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesNoncurrentVersionTransitionsDetails struct {
	Days int `json:"Days,omitempty"`
	StorageClass *string `json:"StorageClass,omitempty"`
}

type AwsS3BucketBucketLifecycleConfigurationRulesTransitionsDetails struct {
	Date *string `json:"Date,omitempty"`
	Days int `json:"Days,omitempty"`
	StorageClass *string `json:"StorageClass,omitempty"`
}

type AwsS3BucketBucketVersioningConfiguration struct {
	IsMfaDeleteEnabled bool `json:"IsMfaDeleteEnabled,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsS3BucketDetails struct {
	AccessControlList *string `json:"AccessControlList,omitempty"`
	BucketLifecycleConfiguration *AwsS3BucketBucketLifecycleConfigurationDetails `json:"BucketLifecycleConfiguration,omitempty"`
	BucketLoggingConfiguration *AwsS3BucketLoggingConfiguration `json:"BucketLoggingConfiguration,omitempty"`
	BucketNotificationConfiguration *AwsS3BucketNotificationConfiguration `json:"BucketNotificationConfiguration,omitempty"`
	BucketVersioningConfiguration *AwsS3BucketBucketVersioningConfiguration `json:"BucketVersioningConfiguration,omitempty"`
	BucketWebsiteConfiguration *AwsS3BucketWebsiteConfiguration `json:"BucketWebsiteConfiguration,omitempty"`
	CreatedAt *string `json:"CreatedAt,omitempty"`
	Name *string `json:"Name,omitempty"`
	ObjectLockConfiguration *AwsS3BucketObjectLockConfiguration `json:"ObjectLockConfiguration,omitempty"`
	OwnerAccountId *string `json:"OwnerAccountId,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	OwnerName *string `json:"OwnerName,omitempty"`
	PublicAccessBlockConfiguration *AwsS3AccountPublicAccessBlockDetails `json:"PublicAccessBlockConfiguration,omitempty"`
	ServerSideEncryptionConfiguration *AwsS3BucketServerSideEncryptionConfiguration `json:"ServerSideEncryptionConfiguration,omitempty"`
}

type AwsS3BucketLoggingConfiguration struct {
	DestinationBucketName *string `json:"DestinationBucketName,omitempty"`
	LogFilePrefix *string `json:"LogFilePrefix,omitempty"`
}

type AwsS3BucketNotificationConfiguration struct {
	Configurations []AwsS3BucketNotificationConfigurationDetail `json:"Configurations,omitempty"`
}

type AwsS3BucketNotificationConfigurationDetail struct {
	Destination *string `json:"Destination,omitempty"`
	Events []string `json:"Events,omitempty"`
	Filter *AwsS3BucketNotificationConfigurationFilter `json:"Filter,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsS3BucketNotificationConfigurationFilter struct {
	S3KeyFilter *AwsS3BucketNotificationConfigurationS3KeyFilter `json:"S3KeyFilter,omitempty"`
}

type AwsS3BucketNotificationConfigurationS3KeyFilter struct {
	FilterRules []AwsS3BucketNotificationConfigurationS3KeyFilterRule `json:"FilterRules,omitempty"`
}

type AwsS3BucketNotificationConfigurationS3KeyFilterRule struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsS3BucketObjectLockConfiguration struct {
	ObjectLockEnabled *string `json:"ObjectLockEnabled,omitempty"`
	Rule *AwsS3BucketObjectLockConfigurationRuleDetails `json:"Rule,omitempty"`
}

type AwsS3BucketObjectLockConfigurationRuleDefaultRetentionDetails struct {
	Days int `json:"Days,omitempty"`
	Mode *string `json:"Mode,omitempty"`
	Years int `json:"Years,omitempty"`
}

type AwsS3BucketObjectLockConfigurationRuleDetails struct {
	DefaultRetention *AwsS3BucketObjectLockConfigurationRuleDefaultRetentionDetails `json:"DefaultRetention,omitempty"`
}

type AwsS3BucketServerSideEncryptionByDefault struct {
	KMSMasterKeyID *string `json:"KMSMasterKeyID,omitempty"`
	SSEAlgorithm *string `json:"SSEAlgorithm,omitempty"`
}

type AwsS3BucketServerSideEncryptionConfiguration struct {
	Rules []AwsS3BucketServerSideEncryptionRule `json:"Rules,omitempty"`
}

type AwsS3BucketServerSideEncryptionRule struct {
	ApplyServerSideEncryptionByDefault *AwsS3BucketServerSideEncryptionByDefault `json:"ApplyServerSideEncryptionByDefault,omitempty"`
}

type AwsS3BucketWebsiteConfiguration struct {
	ErrorDocument *string `json:"ErrorDocument,omitempty"`
	IndexDocumentSuffix *string `json:"IndexDocumentSuffix,omitempty"`
	RedirectAllRequestsTo *AwsS3BucketWebsiteConfigurationRedirectTo `json:"RedirectAllRequestsTo,omitempty"`
	RoutingRules []AwsS3BucketWebsiteConfigurationRoutingRule `json:"RoutingRules,omitempty"`
}

type AwsS3BucketWebsiteConfigurationRedirectTo struct {
	Hostname *string `json:"Hostname,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
}

type AwsS3BucketWebsiteConfigurationRoutingRule struct {
	Condition *AwsS3BucketWebsiteConfigurationRoutingRuleCondition `json:"Condition,omitempty"`
	Redirect *AwsS3BucketWebsiteConfigurationRoutingRuleRedirect `json:"Redirect,omitempty"`
}

type AwsS3BucketWebsiteConfigurationRoutingRuleCondition struct {
	HttpErrorCodeReturnedEquals *string `json:"HttpErrorCodeReturnedEquals,omitempty"`
	KeyPrefixEquals *string `json:"KeyPrefixEquals,omitempty"`
}

type AwsS3BucketWebsiteConfigurationRoutingRuleRedirect struct {
	Hostname *string `json:"Hostname,omitempty"`
	HttpRedirectCode *string `json:"HttpRedirectCode,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	ReplaceKeyPrefixWith *string `json:"ReplaceKeyPrefixWith,omitempty"`
	ReplaceKeyWith *string `json:"ReplaceKeyWith,omitempty"`
}

type AwsS3ObjectDetails struct {
	ContentType *string `json:"ContentType,omitempty"`
	ETag *string `json:"ETag,omitempty"`
	LastModified *string `json:"LastModified,omitempty"`
	SSEKMSKeyId *string `json:"SSEKMSKeyId,omitempty"`
	ServerSideEncryption *string `json:"ServerSideEncryption,omitempty"`
	VersionId *string `json:"VersionId,omitempty"`
}

type AwsSageMakerNotebookInstanceDetails struct {
	AcceleratorTypes []string `json:"AcceleratorTypes,omitempty"`
	AdditionalCodeRepositories []string `json:"AdditionalCodeRepositories,omitempty"`
	DefaultCodeRepository *string `json:"DefaultCodeRepository,omitempty"`
	DirectInternetAccess *string `json:"DirectInternetAccess,omitempty"`
	FailureReason *string `json:"FailureReason,omitempty"`
	InstanceMetadataServiceConfiguration *AwsSageMakerNotebookInstanceMetadataServiceConfigurationDetails `json:"InstanceMetadataServiceConfiguration,omitempty"`
	InstanceType *string `json:"InstanceType,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	NotebookInstanceArn *string `json:"NotebookInstanceArn,omitempty"`
	NotebookInstanceLifecycleConfigName *string `json:"NotebookInstanceLifecycleConfigName,omitempty"`
	NotebookInstanceName *string `json:"NotebookInstanceName,omitempty"`
	NotebookInstanceStatus *string `json:"NotebookInstanceStatus,omitempty"`
	PlatformIdentifier *string `json:"PlatformIdentifier,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	RootAccess *string `json:"RootAccess,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
	Url *string `json:"Url,omitempty"`
	VolumeSizeInGB int `json:"VolumeSizeInGB,omitempty"`
}

type AwsSageMakerNotebookInstanceMetadataServiceConfigurationDetails struct {
	MinimumInstanceMetadataServiceVersion *string `json:"MinimumInstanceMetadataServiceVersion,omitempty"`
}

type AwsSecretsManagerSecretDetails struct {
	Deleted bool `json:"Deleted,omitempty"`
	Description *string `json:"Description,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	Name *string `json:"Name,omitempty"`
	RotationEnabled bool `json:"RotationEnabled,omitempty"`
	RotationLambdaArn *string `json:"RotationLambdaArn,omitempty"`
	RotationOccurredWithinFrequency bool `json:"RotationOccurredWithinFrequency,omitempty"`
	RotationRules *AwsSecretsManagerSecretRotationRules `json:"RotationRules,omitempty"`
}

type AwsSecretsManagerSecretRotationRules struct {
	AutomaticallyAfterDays int `json:"AutomaticallyAfterDays,omitempty"`
}

type AwsSecurityFinding struct {
	Action *Action `json:"Action,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	AwsAccountName *string `json:"AwsAccountName,omitempty"`
	CompanyName *string `json:"CompanyName,omitempty"`
	Compliance *Compliance `json:"Compliance,omitempty"`
	Confidence int `json:"Confidence,omitempty"`
	CreatedAt string `json:"CreatedAt,omitempty"`
	Criticality int `json:"Criticality,omitempty"`
	Description string `json:"Description,omitempty"`
	Detection *Detection `json:"Detection,omitempty"`
	FindingProviderFields *FindingProviderFields `json:"FindingProviderFields,omitempty"`
	FirstObservedAt *string `json:"FirstObservedAt,omitempty"`
	GeneratorDetails *GeneratorDetails `json:"GeneratorDetails,omitempty"`
	GeneratorId string `json:"GeneratorId,omitempty"`
	Id string `json:"Id,omitempty"`
	LastObservedAt *string `json:"LastObservedAt,omitempty"`
	Malware []Malware `json:"Malware,omitempty"`
	Network *Network `json:"Network,omitempty"`
	NetworkPath []NetworkPathComponent `json:"NetworkPath,omitempty"`
	Note *Note `json:"Note,omitempty"`
	PatchSummary *PatchSummary `json:"PatchSummary,omitempty"`
	Process *ProcessDetails `json:"Process,omitempty"`
	ProcessedAt *string `json:"ProcessedAt,omitempty"`
	ProductArn string `json:"ProductArn,omitempty"`
	ProductFields map[string]string `json:"ProductFields,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	RecordState *string `json:"RecordState,omitempty"`
	Region *string `json:"Region,omitempty"`
	RelatedFindings []RelatedFinding `json:"RelatedFindings,omitempty"`
	Remediation *Remediation `json:"Remediation,omitempty"`
	Resources []Resource `json:"Resources,omitempty"`
	Sample bool `json:"Sample,omitempty"`
	SchemaVersion string `json:"SchemaVersion,omitempty"`
	Severity *Severity `json:"Severity,omitempty"`
	SourceUrl *string `json:"SourceUrl,omitempty"`
	ThreatIntelIndicators []ThreatIntelIndicator `json:"ThreatIntelIndicators,omitempty"`
	Threats []Threat `json:"Threats,omitempty"`
	Title string `json:"Title,omitempty"`
	Types []string `json:"Types,omitempty"`
	UpdatedAt string `json:"UpdatedAt,omitempty"`
	UserDefinedFields map[string]string `json:"UserDefinedFields,omitempty"`
	VerificationState *string `json:"VerificationState,omitempty"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities,omitempty"`
	Workflow *Workflow `json:"Workflow,omitempty"`
	WorkflowState *string `json:"WorkflowState,omitempty"`
}

type AwsSecurityFindingFilters struct {
	AwsAccountId []StringFilter `json:"AwsAccountId,omitempty"`
	AwsAccountName []StringFilter `json:"AwsAccountName,omitempty"`
	CompanyName []StringFilter `json:"CompanyName,omitempty"`
	ComplianceAssociatedStandardsId []StringFilter `json:"ComplianceAssociatedStandardsId,omitempty"`
	ComplianceSecurityControlId []StringFilter `json:"ComplianceSecurityControlId,omitempty"`
	ComplianceSecurityControlParametersName []StringFilter `json:"ComplianceSecurityControlParametersName,omitempty"`
	ComplianceSecurityControlParametersValue []StringFilter `json:"ComplianceSecurityControlParametersValue,omitempty"`
	ComplianceStatus []StringFilter `json:"ComplianceStatus,omitempty"`
	Confidence []NumberFilter `json:"Confidence,omitempty"`
	CreatedAt []DateFilter `json:"CreatedAt,omitempty"`
	Criticality []NumberFilter `json:"Criticality,omitempty"`
	Description []StringFilter `json:"Description,omitempty"`
	FindingProviderFieldsConfidence []NumberFilter `json:"FindingProviderFieldsConfidence,omitempty"`
	FindingProviderFieldsCriticality []NumberFilter `json:"FindingProviderFieldsCriticality,omitempty"`
	FindingProviderFieldsRelatedFindingsId []StringFilter `json:"FindingProviderFieldsRelatedFindingsId,omitempty"`
	FindingProviderFieldsRelatedFindingsProductArn []StringFilter `json:"FindingProviderFieldsRelatedFindingsProductArn,omitempty"`
	FindingProviderFieldsSeverityLabel []StringFilter `json:"FindingProviderFieldsSeverityLabel,omitempty"`
	FindingProviderFieldsSeverityOriginal []StringFilter `json:"FindingProviderFieldsSeverityOriginal,omitempty"`
	FindingProviderFieldsTypes []StringFilter `json:"FindingProviderFieldsTypes,omitempty"`
	FirstObservedAt []DateFilter `json:"FirstObservedAt,omitempty"`
	GeneratorId []StringFilter `json:"GeneratorId,omitempty"`
	Id []StringFilter `json:"Id,omitempty"`
	Keyword []KeywordFilter `json:"Keyword,omitempty"`
	LastObservedAt []DateFilter `json:"LastObservedAt,omitempty"`
	MalwareName []StringFilter `json:"MalwareName,omitempty"`
	MalwarePath []StringFilter `json:"MalwarePath,omitempty"`
	MalwareState []StringFilter `json:"MalwareState,omitempty"`
	MalwareType []StringFilter `json:"MalwareType,omitempty"`
	NetworkDestinationDomain []StringFilter `json:"NetworkDestinationDomain,omitempty"`
	NetworkDestinationIpV4 []IpFilter `json:"NetworkDestinationIpV4,omitempty"`
	NetworkDestinationIpV6 []IpFilter `json:"NetworkDestinationIpV6,omitempty"`
	NetworkDestinationPort []NumberFilter `json:"NetworkDestinationPort,omitempty"`
	NetworkDirection []StringFilter `json:"NetworkDirection,omitempty"`
	NetworkProtocol []StringFilter `json:"NetworkProtocol,omitempty"`
	NetworkSourceDomain []StringFilter `json:"NetworkSourceDomain,omitempty"`
	NetworkSourceIpV4 []IpFilter `json:"NetworkSourceIpV4,omitempty"`
	NetworkSourceIpV6 []IpFilter `json:"NetworkSourceIpV6,omitempty"`
	NetworkSourceMac []StringFilter `json:"NetworkSourceMac,omitempty"`
	NetworkSourcePort []NumberFilter `json:"NetworkSourcePort,omitempty"`
	NoteText []StringFilter `json:"NoteText,omitempty"`
	NoteUpdatedAt []DateFilter `json:"NoteUpdatedAt,omitempty"`
	NoteUpdatedBy []StringFilter `json:"NoteUpdatedBy,omitempty"`
	ProcessLaunchedAt []DateFilter `json:"ProcessLaunchedAt,omitempty"`
	ProcessName []StringFilter `json:"ProcessName,omitempty"`
	ProcessParentPid []NumberFilter `json:"ProcessParentPid,omitempty"`
	ProcessPath []StringFilter `json:"ProcessPath,omitempty"`
	ProcessPid []NumberFilter `json:"ProcessPid,omitempty"`
	ProcessTerminatedAt []DateFilter `json:"ProcessTerminatedAt,omitempty"`
	ProductArn []StringFilter `json:"ProductArn,omitempty"`
	ProductFields []MapFilter `json:"ProductFields,omitempty"`
	ProductName []StringFilter `json:"ProductName,omitempty"`
	RecommendationText []StringFilter `json:"RecommendationText,omitempty"`
	RecordState []StringFilter `json:"RecordState,omitempty"`
	Region []StringFilter `json:"Region,omitempty"`
	RelatedFindingsId []StringFilter `json:"RelatedFindingsId,omitempty"`
	RelatedFindingsProductArn []StringFilter `json:"RelatedFindingsProductArn,omitempty"`
	ResourceApplicationArn []StringFilter `json:"ResourceApplicationArn,omitempty"`
	ResourceApplicationName []StringFilter `json:"ResourceApplicationName,omitempty"`
	ResourceAwsEc2InstanceIamInstanceProfileArn []StringFilter `json:"ResourceAwsEc2InstanceIamInstanceProfileArn,omitempty"`
	ResourceAwsEc2InstanceImageId []StringFilter `json:"ResourceAwsEc2InstanceImageId,omitempty"`
	ResourceAwsEc2InstanceIpV4Addresses []IpFilter `json:"ResourceAwsEc2InstanceIpV4Addresses,omitempty"`
	ResourceAwsEc2InstanceIpV6Addresses []IpFilter `json:"ResourceAwsEc2InstanceIpV6Addresses,omitempty"`
	ResourceAwsEc2InstanceKeyName []StringFilter `json:"ResourceAwsEc2InstanceKeyName,omitempty"`
	ResourceAwsEc2InstanceLaunchedAt []DateFilter `json:"ResourceAwsEc2InstanceLaunchedAt,omitempty"`
	ResourceAwsEc2InstanceSubnetId []StringFilter `json:"ResourceAwsEc2InstanceSubnetId,omitempty"`
	ResourceAwsEc2InstanceType []StringFilter `json:"ResourceAwsEc2InstanceType,omitempty"`
	ResourceAwsEc2InstanceVpcId []StringFilter `json:"ResourceAwsEc2InstanceVpcId,omitempty"`
	ResourceAwsIamAccessKeyCreatedAt []DateFilter `json:"ResourceAwsIamAccessKeyCreatedAt,omitempty"`
	ResourceAwsIamAccessKeyPrincipalName []StringFilter `json:"ResourceAwsIamAccessKeyPrincipalName,omitempty"`
	ResourceAwsIamAccessKeyStatus []StringFilter `json:"ResourceAwsIamAccessKeyStatus,omitempty"`
	ResourceAwsIamAccessKeyUserName []StringFilter `json:"ResourceAwsIamAccessKeyUserName,omitempty"`
	ResourceAwsIamUserUserName []StringFilter `json:"ResourceAwsIamUserUserName,omitempty"`
	ResourceAwsS3BucketOwnerId []StringFilter `json:"ResourceAwsS3BucketOwnerId,omitempty"`
	ResourceAwsS3BucketOwnerName []StringFilter `json:"ResourceAwsS3BucketOwnerName,omitempty"`
	ResourceContainerImageId []StringFilter `json:"ResourceContainerImageId,omitempty"`
	ResourceContainerImageName []StringFilter `json:"ResourceContainerImageName,omitempty"`
	ResourceContainerLaunchedAt []DateFilter `json:"ResourceContainerLaunchedAt,omitempty"`
	ResourceContainerName []StringFilter `json:"ResourceContainerName,omitempty"`
	ResourceDetailsOther []MapFilter `json:"ResourceDetailsOther,omitempty"`
	ResourceId []StringFilter `json:"ResourceId,omitempty"`
	ResourcePartition []StringFilter `json:"ResourcePartition,omitempty"`
	ResourceRegion []StringFilter `json:"ResourceRegion,omitempty"`
	ResourceTags []MapFilter `json:"ResourceTags,omitempty"`
	ResourceType []StringFilter `json:"ResourceType,omitempty"`
	Sample []BooleanFilter `json:"Sample,omitempty"`
	SeverityLabel []StringFilter `json:"SeverityLabel,omitempty"`
	SeverityNormalized []NumberFilter `json:"SeverityNormalized,omitempty"`
	SeverityProduct []NumberFilter `json:"SeverityProduct,omitempty"`
	SourceUrl []StringFilter `json:"SourceUrl,omitempty"`
	ThreatIntelIndicatorCategory []StringFilter `json:"ThreatIntelIndicatorCategory,omitempty"`
	ThreatIntelIndicatorLastObservedAt []DateFilter `json:"ThreatIntelIndicatorLastObservedAt,omitempty"`
	ThreatIntelIndicatorSource []StringFilter `json:"ThreatIntelIndicatorSource,omitempty"`
	ThreatIntelIndicatorSourceUrl []StringFilter `json:"ThreatIntelIndicatorSourceUrl,omitempty"`
	ThreatIntelIndicatorType []StringFilter `json:"ThreatIntelIndicatorType,omitempty"`
	ThreatIntelIndicatorValue []StringFilter `json:"ThreatIntelIndicatorValue,omitempty"`
	Title []StringFilter `json:"Title,omitempty"`
	Type []StringFilter `json:"Type,omitempty"`
	UpdatedAt []DateFilter `json:"UpdatedAt,omitempty"`
	UserDefinedFields []MapFilter `json:"UserDefinedFields,omitempty"`
	VerificationState []StringFilter `json:"VerificationState,omitempty"`
	VulnerabilitiesExploitAvailable []StringFilter `json:"VulnerabilitiesExploitAvailable,omitempty"`
	VulnerabilitiesFixAvailable []StringFilter `json:"VulnerabilitiesFixAvailable,omitempty"`
	WorkflowState []StringFilter `json:"WorkflowState,omitempty"`
	WorkflowStatus []StringFilter `json:"WorkflowStatus,omitempty"`
}

type AwsSecurityFindingIdentifier struct {
	Id string `json:"Id,omitempty"`
	ProductArn string `json:"ProductArn,omitempty"`
}

type AwsSnsTopicDetails struct {
	ApplicationSuccessFeedbackRoleArn *string `json:"ApplicationSuccessFeedbackRoleArn,omitempty"`
	FirehoseFailureFeedbackRoleArn *string `json:"FirehoseFailureFeedbackRoleArn,omitempty"`
	FirehoseSuccessFeedbackRoleArn *string `json:"FirehoseSuccessFeedbackRoleArn,omitempty"`
	HttpFailureFeedbackRoleArn *string `json:"HttpFailureFeedbackRoleArn,omitempty"`
	HttpSuccessFeedbackRoleArn *string `json:"HttpSuccessFeedbackRoleArn,omitempty"`
	KmsMasterKeyId *string `json:"KmsMasterKeyId,omitempty"`
	Owner *string `json:"Owner,omitempty"`
	SqsFailureFeedbackRoleArn *string `json:"SqsFailureFeedbackRoleArn,omitempty"`
	SqsSuccessFeedbackRoleArn *string `json:"SqsSuccessFeedbackRoleArn,omitempty"`
	Subscription []AwsSnsTopicSubscription `json:"Subscription,omitempty"`
	TopicName *string `json:"TopicName,omitempty"`
}

type AwsSnsTopicSubscription struct {
	Endpoint *string `json:"Endpoint,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
}

type AwsSqsQueueDetails struct {
	DeadLetterTargetArn *string `json:"DeadLetterTargetArn,omitempty"`
	KmsDataKeyReusePeriodSeconds int `json:"KmsDataKeyReusePeriodSeconds,omitempty"`
	KmsMasterKeyId *string `json:"KmsMasterKeyId,omitempty"`
	QueueName *string `json:"QueueName,omitempty"`
}

type AwsSsmComplianceSummary struct {
	ComplianceType *string `json:"ComplianceType,omitempty"`
	CompliantCriticalCount int `json:"CompliantCriticalCount,omitempty"`
	CompliantHighCount int `json:"CompliantHighCount,omitempty"`
	CompliantInformationalCount int `json:"CompliantInformationalCount,omitempty"`
	CompliantLowCount int `json:"CompliantLowCount,omitempty"`
	CompliantMediumCount int `json:"CompliantMediumCount,omitempty"`
	CompliantUnspecifiedCount int `json:"CompliantUnspecifiedCount,omitempty"`
	ExecutionType *string `json:"ExecutionType,omitempty"`
	NonCompliantCriticalCount int `json:"NonCompliantCriticalCount,omitempty"`
	NonCompliantHighCount int `json:"NonCompliantHighCount,omitempty"`
	NonCompliantInformationalCount int `json:"NonCompliantInformationalCount,omitempty"`
	NonCompliantLowCount int `json:"NonCompliantLowCount,omitempty"`
	NonCompliantMediumCount int `json:"NonCompliantMediumCount,omitempty"`
	NonCompliantUnspecifiedCount int `json:"NonCompliantUnspecifiedCount,omitempty"`
	OverallSeverity *string `json:"OverallSeverity,omitempty"`
	PatchBaselineId *string `json:"PatchBaselineId,omitempty"`
	PatchGroup *string `json:"PatchGroup,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AwsSsmPatch struct {
	ComplianceSummary *AwsSsmComplianceSummary `json:"ComplianceSummary,omitempty"`
}

type AwsSsmPatchComplianceDetails struct {
	Patch *AwsSsmPatch `json:"Patch,omitempty"`
}

type AwsStepFunctionStateMachineDetails struct {
	Label *string `json:"Label,omitempty"`
	LoggingConfiguration *AwsStepFunctionStateMachineLoggingConfigurationDetails `json:"LoggingConfiguration,omitempty"`
	Name *string `json:"Name,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	StateMachineArn *string `json:"StateMachineArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	TracingConfiguration *AwsStepFunctionStateMachineTracingConfigurationDetails `json:"TracingConfiguration,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsStepFunctionStateMachineLoggingConfigurationDestinationsCloudWatchLogsLogGroupDetails struct {
	LogGroupArn *string `json:"LogGroupArn,omitempty"`
}

type AwsStepFunctionStateMachineLoggingConfigurationDestinationsDetails struct {
	CloudWatchLogsLogGroup *AwsStepFunctionStateMachineLoggingConfigurationDestinationsCloudWatchLogsLogGroupDetails `json:"CloudWatchLogsLogGroup,omitempty"`
}

type AwsStepFunctionStateMachineLoggingConfigurationDetails struct {
	Destinations []AwsStepFunctionStateMachineLoggingConfigurationDestinationsDetails `json:"Destinations,omitempty"`
	IncludeExecutionData bool `json:"IncludeExecutionData,omitempty"`
	Level *string `json:"Level,omitempty"`
}

type AwsStepFunctionStateMachineTracingConfigurationDetails struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type AwsWafRateBasedRuleDetails struct {
	MatchPredicates []AwsWafRateBasedRuleMatchPredicate `json:"MatchPredicates,omitempty"`
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	RateKey *string `json:"RateKey,omitempty"`
	RateLimit int64 `json:"RateLimit,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
}

type AwsWafRateBasedRuleMatchPredicate struct {
	DataId *string `json:"DataId,omitempty"`
	Negated bool `json:"Negated,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalRateBasedRuleDetails struct {
	MatchPredicates []AwsWafRegionalRateBasedRuleMatchPredicate `json:"MatchPredicates,omitempty"`
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	RateKey *string `json:"RateKey,omitempty"`
	RateLimit int64 `json:"RateLimit,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
}

type AwsWafRegionalRateBasedRuleMatchPredicate struct {
	DataId *string `json:"DataId,omitempty"`
	Negated bool `json:"Negated,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalRuleDetails struct {
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	PredicateList []AwsWafRegionalRulePredicateListDetails `json:"PredicateList,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
}

type AwsWafRegionalRuleGroupDetails struct {
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	RuleGroupId *string `json:"RuleGroupId,omitempty"`
	Rules []AwsWafRegionalRuleGroupRulesDetails `json:"Rules,omitempty"`
}

type AwsWafRegionalRuleGroupRulesActionDetails struct {
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalRuleGroupRulesDetails struct {
	Action *AwsWafRegionalRuleGroupRulesActionDetails `json:"Action,omitempty"`
	Priority int `json:"Priority,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalRulePredicateListDetails struct {
	DataId *string `json:"DataId,omitempty"`
	Negated bool `json:"Negated,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalWebAclDetails struct {
	DefaultAction *string `json:"DefaultAction,omitempty"`
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	RulesList []AwsWafRegionalWebAclRulesListDetails `json:"RulesList,omitempty"`
	WebAclId *string `json:"WebAclId,omitempty"`
}

type AwsWafRegionalWebAclRulesListActionDetails struct {
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalWebAclRulesListDetails struct {
	Action *AwsWafRegionalWebAclRulesListActionDetails `json:"Action,omitempty"`
	OverrideAction *AwsWafRegionalWebAclRulesListOverrideActionDetails `json:"OverrideAction,omitempty"`
	Priority int `json:"Priority,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafRegionalWebAclRulesListOverrideActionDetails struct {
	Type *string `json:"Type,omitempty"`
}

type AwsWafRuleDetails struct {
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	PredicateList []AwsWafRulePredicateListDetails `json:"PredicateList,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
}

type AwsWafRuleGroupDetails struct {
	MetricName *string `json:"MetricName,omitempty"`
	Name *string `json:"Name,omitempty"`
	RuleGroupId *string `json:"RuleGroupId,omitempty"`
	Rules []AwsWafRuleGroupRulesDetails `json:"Rules,omitempty"`
}

type AwsWafRuleGroupRulesActionDetails struct {
	Type *string `json:"Type,omitempty"`
}

type AwsWafRuleGroupRulesDetails struct {
	Action *AwsWafRuleGroupRulesActionDetails `json:"Action,omitempty"`
	Priority int `json:"Priority,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafRulePredicateListDetails struct {
	DataId *string `json:"DataId,omitempty"`
	Negated bool `json:"Negated,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafWebAclDetails struct {
	DefaultAction *string `json:"DefaultAction,omitempty"`
	Name *string `json:"Name,omitempty"`
	Rules []AwsWafWebAclRule `json:"Rules,omitempty"`
	WebAclId *string `json:"WebAclId,omitempty"`
}

type AwsWafWebAclRule struct {
	Action *WafAction `json:"Action,omitempty"`
	ExcludedRules []WafExcludedRule `json:"ExcludedRules,omitempty"`
	OverrideAction *WafOverrideAction `json:"OverrideAction,omitempty"`
	Priority int `json:"Priority,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AwsWafv2ActionAllowDetails struct {
	CustomRequestHandling *AwsWafv2CustomRequestHandlingDetails `json:"CustomRequestHandling,omitempty"`
}

type AwsWafv2ActionBlockDetails struct {
	CustomResponse *AwsWafv2CustomResponseDetails `json:"CustomResponse,omitempty"`
}

type AwsWafv2CustomHttpHeader struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AwsWafv2CustomRequestHandlingDetails struct {
	InsertHeaders []AwsWafv2CustomHttpHeader `json:"InsertHeaders,omitempty"`
}

type AwsWafv2CustomResponseDetails struct {
	CustomResponseBodyKey *string `json:"CustomResponseBodyKey,omitempty"`
	ResponseCode int `json:"ResponseCode,omitempty"`
	ResponseHeaders []AwsWafv2CustomHttpHeader `json:"ResponseHeaders,omitempty"`
}

type AwsWafv2RuleGroupDetails struct {
	Arn *string `json:"Arn,omitempty"`
	Capacity int64 `json:"Capacity,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	Rules []AwsWafv2RulesDetails `json:"Rules,omitempty"`
	Scope *string `json:"Scope,omitempty"`
	VisibilityConfig *AwsWafv2VisibilityConfigDetails `json:"VisibilityConfig,omitempty"`
}

type AwsWafv2RulesActionCaptchaDetails struct {
	CustomRequestHandling *AwsWafv2CustomRequestHandlingDetails `json:"CustomRequestHandling,omitempty"`
}

type AwsWafv2RulesActionCountDetails struct {
	CustomRequestHandling *AwsWafv2CustomRequestHandlingDetails `json:"CustomRequestHandling,omitempty"`
}

type AwsWafv2RulesActionDetails struct {
	Allow *AwsWafv2ActionAllowDetails `json:"Allow,omitempty"`
	Block *AwsWafv2ActionBlockDetails `json:"Block,omitempty"`
	Captcha *AwsWafv2RulesActionCaptchaDetails `json:"Captcha,omitempty"`
	Count *AwsWafv2RulesActionCountDetails `json:"Count,omitempty"`
}

type AwsWafv2RulesDetails struct {
	Action *AwsWafv2RulesActionDetails `json:"Action,omitempty"`
	Name *string `json:"Name,omitempty"`
	OverrideAction *string `json:"OverrideAction,omitempty"`
	Priority int `json:"Priority,omitempty"`
	VisibilityConfig *AwsWafv2VisibilityConfigDetails `json:"VisibilityConfig,omitempty"`
}

type AwsWafv2VisibilityConfigDetails struct {
	CloudWatchMetricsEnabled bool `json:"CloudWatchMetricsEnabled,omitempty"`
	MetricName *string `json:"MetricName,omitempty"`
	SampledRequestsEnabled bool `json:"SampledRequestsEnabled,omitempty"`
}

type AwsWafv2WebAclActionDetails struct {
	Allow *AwsWafv2ActionAllowDetails `json:"Allow,omitempty"`
	Block *AwsWafv2ActionBlockDetails `json:"Block,omitempty"`
}

type AwsWafv2WebAclCaptchaConfigDetails struct {
	ImmunityTimeProperty *AwsWafv2WebAclCaptchaConfigImmunityTimePropertyDetails `json:"ImmunityTimeProperty,omitempty"`
}

type AwsWafv2WebAclCaptchaConfigImmunityTimePropertyDetails struct {
	ImmunityTime int64 `json:"ImmunityTime,omitempty"`
}

type AwsWafv2WebAclDetails struct {
	Arn *string `json:"Arn,omitempty"`
	Capacity int64 `json:"Capacity,omitempty"`
	CaptchaConfig *AwsWafv2WebAclCaptchaConfigDetails `json:"CaptchaConfig,omitempty"`
	DefaultAction *AwsWafv2WebAclActionDetails `json:"DefaultAction,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	ManagedbyFirewallManager bool `json:"ManagedbyFirewallManager,omitempty"`
	Name *string `json:"Name,omitempty"`
	Rules []AwsWafv2RulesDetails `json:"Rules,omitempty"`
	VisibilityConfig *AwsWafv2VisibilityConfigDetails `json:"VisibilityConfig,omitempty"`
}

type AwsXrayEncryptionConfigDetails struct {
	KeyId *string `json:"KeyId,omitempty"`
	Status *string `json:"Status,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type BatchDeleteAutomationRulesRequest struct {
	AutomationRulesArns []string `json:"AutomationRulesArns,omitempty"`
}

type BatchDeleteAutomationRulesResponse struct {
	ProcessedAutomationRules []string `json:"ProcessedAutomationRules,omitempty"`
	UnprocessedAutomationRules []UnprocessedAutomationRule `json:"UnprocessedAutomationRules,omitempty"`
}

type BatchDisableStandardsRequest struct {
	StandardsSubscriptionArns []string `json:"StandardsSubscriptionArns,omitempty"`
}

type BatchDisableStandardsResponse struct {
	StandardsSubscriptions []StandardsSubscription `json:"StandardsSubscriptions,omitempty"`
}

type BatchEnableStandardsRequest struct {
	StandardsSubscriptionRequests []StandardsSubscriptionRequest `json:"StandardsSubscriptionRequests,omitempty"`
}

type BatchEnableStandardsResponse struct {
	StandardsSubscriptions []StandardsSubscription `json:"StandardsSubscriptions,omitempty"`
}

type BatchGetAutomationRulesRequest struct {
	AutomationRulesArns []string `json:"AutomationRulesArns,omitempty"`
}

type BatchGetAutomationRulesResponse struct {
	Rules []AutomationRulesConfig `json:"Rules,omitempty"`
	UnprocessedAutomationRules []UnprocessedAutomationRule `json:"UnprocessedAutomationRules,omitempty"`
}

type BatchGetConfigurationPolicyAssociationsRequest struct {
	ConfigurationPolicyAssociationIdentifiers []ConfigurationPolicyAssociation `json:"ConfigurationPolicyAssociationIdentifiers,omitempty"`
}

type BatchGetConfigurationPolicyAssociationsResponse struct {
	ConfigurationPolicyAssociations []ConfigurationPolicyAssociationSummary `json:"ConfigurationPolicyAssociations,omitempty"`
	UnprocessedConfigurationPolicyAssociations []UnprocessedConfigurationPolicyAssociation `json:"UnprocessedConfigurationPolicyAssociations,omitempty"`
}

type BatchGetSecurityControlsRequest struct {
	SecurityControlIds []string `json:"SecurityControlIds,omitempty"`
}

type BatchGetSecurityControlsResponse struct {
	SecurityControls []SecurityControl `json:"SecurityControls,omitempty"`
	UnprocessedIds []UnprocessedSecurityControl `json:"UnprocessedIds,omitempty"`
}

type BatchGetStandardsControlAssociationsRequest struct {
	StandardsControlAssociationIds []StandardsControlAssociationId `json:"StandardsControlAssociationIds,omitempty"`
}

type BatchGetStandardsControlAssociationsResponse struct {
	StandardsControlAssociationDetails []StandardsControlAssociationDetail `json:"StandardsControlAssociationDetails,omitempty"`
	UnprocessedAssociations []UnprocessedStandardsControlAssociation `json:"UnprocessedAssociations,omitempty"`
}

type BatchImportFindingsRequest struct {
	Findings []AwsSecurityFinding `json:"Findings,omitempty"`
}

type BatchImportFindingsResponse struct {
	FailedCount int `json:"FailedCount,omitempty"`
	FailedFindings []ImportFindingsError `json:"FailedFindings,omitempty"`
	SuccessCount int `json:"SuccessCount,omitempty"`
}

type BatchUpdateAutomationRulesRequest struct {
	UpdateAutomationRulesRequestItems []UpdateAutomationRulesRequestItem `json:"UpdateAutomationRulesRequestItems,omitempty"`
}

type BatchUpdateAutomationRulesResponse struct {
	ProcessedAutomationRules []string `json:"ProcessedAutomationRules,omitempty"`
	UnprocessedAutomationRules []UnprocessedAutomationRule `json:"UnprocessedAutomationRules,omitempty"`
}

type BatchUpdateFindingsRequest struct {
	Confidence int `json:"Confidence,omitempty"`
	Criticality int `json:"Criticality,omitempty"`
	FindingIdentifiers []AwsSecurityFindingIdentifier `json:"FindingIdentifiers,omitempty"`
	Note *NoteUpdate `json:"Note,omitempty"`
	RelatedFindings []RelatedFinding `json:"RelatedFindings,omitempty"`
	Severity *SeverityUpdate `json:"Severity,omitempty"`
	Types []string `json:"Types,omitempty"`
	UserDefinedFields map[string]string `json:"UserDefinedFields,omitempty"`
	VerificationState *string `json:"VerificationState,omitempty"`
	Workflow *WorkflowUpdate `json:"Workflow,omitempty"`
}

type BatchUpdateFindingsResponse struct {
	ProcessedFindings []AwsSecurityFindingIdentifier `json:"ProcessedFindings,omitempty"`
	UnprocessedFindings []BatchUpdateFindingsUnprocessedFinding `json:"UnprocessedFindings,omitempty"`
}

type BatchUpdateFindingsUnprocessedFinding struct {
	ErrorCode string `json:"ErrorCode,omitempty"`
	ErrorMessage string `json:"ErrorMessage,omitempty"`
	FindingIdentifier AwsSecurityFindingIdentifier `json:"FindingIdentifier,omitempty"`
}

type BatchUpdateFindingsV2ProcessedFinding struct {
	FindingIdentifier *OcsfFindingIdentifier `json:"FindingIdentifier,omitempty"`
	MetadataUid *string `json:"MetadataUid,omitempty"`
}

type BatchUpdateFindingsV2Request struct {
	Comment *string `json:"Comment,omitempty"`
	FindingIdentifiers []OcsfFindingIdentifier `json:"FindingIdentifiers,omitempty"`
	MetadataUids []string `json:"MetadataUids,omitempty"`
	SeverityId int `json:"SeverityId,omitempty"`
	StatusId int `json:"StatusId,omitempty"`
}

type BatchUpdateFindingsV2Response struct {
	ProcessedFindings []BatchUpdateFindingsV2ProcessedFinding `json:"ProcessedFindings,omitempty"`
	UnprocessedFindings []BatchUpdateFindingsV2UnprocessedFinding `json:"UnprocessedFindings,omitempty"`
}

type BatchUpdateFindingsV2UnprocessedFinding struct {
	ErrorCode *string `json:"ErrorCode,omitempty"`
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	FindingIdentifier *OcsfFindingIdentifier `json:"FindingIdentifier,omitempty"`
	MetadataUid *string `json:"MetadataUid,omitempty"`
}

type BatchUpdateStandardsControlAssociationsRequest struct {
	StandardsControlAssociationUpdates []StandardsControlAssociationUpdate `json:"StandardsControlAssociationUpdates,omitempty"`
}

type BatchUpdateStandardsControlAssociationsResponse struct {
	UnprocessedAssociationUpdates []UnprocessedStandardsControlAssociationUpdate `json:"UnprocessedAssociationUpdates,omitempty"`
}

type BooleanConfigurationOptions struct {
	DefaultValue bool `json:"DefaultValue,omitempty"`
}

type BooleanFilter struct {
	Value bool `json:"Value,omitempty"`
}

type Cell struct {
	CellReference *string `json:"CellReference,omitempty"`
	Column int64 `json:"Column,omitempty"`
	ColumnName *string `json:"ColumnName,omitempty"`
	Row int64 `json:"Row,omitempty"`
}

type CidrBlockAssociation struct {
	AssociationId *string `json:"AssociationId,omitempty"`
	CidrBlock *string `json:"CidrBlock,omitempty"`
	CidrBlockState *string `json:"CidrBlockState,omitempty"`
}

type City struct {
	CityName *string `json:"CityName,omitempty"`
}

type ClassificationResult struct {
	AdditionalOccurrences bool `json:"AdditionalOccurrences,omitempty"`
	CustomDataIdentifiers *CustomDataIdentifiersResult `json:"CustomDataIdentifiers,omitempty"`
	MimeType *string `json:"MimeType,omitempty"`
	SensitiveData []SensitiveDataResult `json:"SensitiveData,omitempty"`
	SizeClassified int64 `json:"SizeClassified,omitempty"`
	Status *ClassificationStatus `json:"Status,omitempty"`
}

type ClassificationStatus struct {
	Code *string `json:"Code,omitempty"`
	Reason *string `json:"Reason,omitempty"`
}

type CloudWatchLogsLogGroupArnConfigDetails struct {
	CloudWatchLogsLogGroupArn *string `json:"CloudWatchLogsLogGroupArn,omitempty"`
	HostedZoneId *string `json:"HostedZoneId,omitempty"`
	Id *string `json:"Id,omitempty"`
}

type CodeRepositoryDetails struct {
	CodeSecurityIntegrationArn *string `json:"CodeSecurityIntegrationArn,omitempty"`
	ProjectName *string `json:"ProjectName,omitempty"`
	ProviderType *string `json:"ProviderType,omitempty"`
}

type CodeVulnerabilitiesFilePath struct {
	EndLine int `json:"EndLine,omitempty"`
	FileName *string `json:"FileName,omitempty"`
	FilePath *string `json:"FilePath,omitempty"`
	StartLine int `json:"StartLine,omitempty"`
}

type Compliance struct {
	AssociatedStandards []AssociatedStandard `json:"AssociatedStandards,omitempty"`
	RelatedRequirements []string `json:"RelatedRequirements,omitempty"`
	SecurityControlId *string `json:"SecurityControlId,omitempty"`
	SecurityControlParameters []SecurityControlParameter `json:"SecurityControlParameters,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusReasons []StatusReason `json:"StatusReasons,omitempty"`
}

type CompositeFilter struct {
	BooleanFilters []OcsfBooleanFilter `json:"BooleanFilters,omitempty"`
	DateFilters []OcsfDateFilter `json:"DateFilters,omitempty"`
	IpFilters []OcsfIpFilter `json:"IpFilters,omitempty"`
	MapFilters []OcsfMapFilter `json:"MapFilters,omitempty"`
	NestedCompositeFilters []CompositeFilter `json:"NestedCompositeFilters,omitempty"`
	NumberFilters []OcsfNumberFilter `json:"NumberFilters,omitempty"`
	Operator *string `json:"Operator,omitempty"`
	StringFilters []OcsfStringFilter `json:"StringFilters,omitempty"`
}

type ConfigurationOptions struct {
	Boolean *BooleanConfigurationOptions `json:"Boolean,omitempty"`
	Double *DoubleConfigurationOptions `json:"Double,omitempty"`
	Enum *EnumConfigurationOptions `json:"Enum,omitempty"`
	EnumList *EnumListConfigurationOptions `json:"EnumList,omitempty"`
	Integer *IntegerConfigurationOptions `json:"Integer,omitempty"`
	IntegerList *IntegerListConfigurationOptions `json:"IntegerList,omitempty"`
	String *StringConfigurationOptions `json:"String,omitempty"`
	StringList *StringListConfigurationOptions `json:"StringList,omitempty"`
}

type ConfigurationPolicyAssociation struct {
	Target *Target `json:"Target,omitempty"`
}

type ConfigurationPolicyAssociationSummary struct {
	AssociationStatus *string `json:"AssociationStatus,omitempty"`
	AssociationStatusMessage *string `json:"AssociationStatusMessage,omitempty"`
	AssociationType *string `json:"AssociationType,omitempty"`
	ConfigurationPolicyId *string `json:"ConfigurationPolicyId,omitempty"`
	TargetId *string `json:"TargetId,omitempty"`
	TargetType *string `json:"TargetType,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type ConfigurationPolicySummary struct {
	Arn *string `json:"Arn,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	ServiceEnabled bool `json:"ServiceEnabled,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type ConnectorSummary struct {
	ConnectorArn *string `json:"ConnectorArn,omitempty"`
	ConnectorId string `json:"ConnectorId,omitempty"`
	CreatedAt time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name string `json:"Name,omitempty"`
	ProviderSummary ProviderSummary `json:"ProviderSummary,omitempty"`
}

type ContainerDetails struct {
	ContainerRuntime *string `json:"ContainerRuntime,omitempty"`
	ImageId *string `json:"ImageId,omitempty"`
	ImageName *string `json:"ImageName,omitempty"`
	LaunchedAt *string `json:"LaunchedAt,omitempty"`
	Name *string `json:"Name,omitempty"`
	Privileged bool `json:"Privileged,omitempty"`
	VolumeMounts []VolumeMount `json:"VolumeMounts,omitempty"`
}

type Country struct {
	CountryCode *string `json:"CountryCode,omitempty"`
	CountryName *string `json:"CountryName,omitempty"`
}

type CreateActionTargetRequest struct {
	Description string `json:"Description,omitempty"`
	Id string `json:"Id,omitempty"`
	Name string `json:"Name,omitempty"`
}

type CreateActionTargetResponse struct {
	ActionTargetArn string `json:"ActionTargetArn,omitempty"`
}

type CreateAggregatorV2Request struct {
	ClientToken *string `json:"ClientToken,omitempty"`
	LinkedRegions []string `json:"LinkedRegions,omitempty"`
	RegionLinkingMode string `json:"RegionLinkingMode,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateAggregatorV2Response struct {
	AggregationRegion *string `json:"AggregationRegion,omitempty"`
	AggregatorV2Arn *string `json:"AggregatorV2Arn,omitempty"`
	LinkedRegions []string `json:"LinkedRegions,omitempty"`
	RegionLinkingMode *string `json:"RegionLinkingMode,omitempty"`
}

type CreateAutomationRuleRequest struct {
	Actions []AutomationRulesAction `json:"Actions,omitempty"`
	Criteria AutomationRulesFindingFilters `json:"Criteria,omitempty"`
	Description string `json:"Description,omitempty"`
	IsTerminal bool `json:"IsTerminal,omitempty"`
	RuleName string `json:"RuleName,omitempty"`
	RuleOrder int `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateAutomationRuleResponse struct {
	RuleArn *string `json:"RuleArn,omitempty"`
}

type CreateAutomationRuleV2Request struct {
	Actions []AutomationRulesActionV2 `json:"Actions,omitempty"`
	ClientToken *string `json:"ClientToken,omitempty"`
	Criteria Criteria `json:"Criteria,omitempty"`
	Description string `json:"Description,omitempty"`
	RuleName string `json:"RuleName,omitempty"`
	RuleOrder float64 `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateAutomationRuleV2Response struct {
	RuleArn *string `json:"RuleArn,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
}

type CreateConfigurationPolicyRequest struct {
	ConfigurationPolicy Policy `json:"ConfigurationPolicy,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name string `json:"Name,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateConfigurationPolicyResponse struct {
	Arn *string `json:"Arn,omitempty"`
	ConfigurationPolicy *Policy `json:"ConfigurationPolicy,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type CreateConnectorV2Request struct {
	ClientToken *string `json:"ClientToken,omitempty"`
	Description *string `json:"Description,omitempty"`
	KmsKeyArn *string `json:"KmsKeyArn,omitempty"`
	Name string `json:"Name,omitempty"`
	Provider ProviderConfiguration `json:"Provider,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type CreateConnectorV2Response struct {
	AuthUrl *string `json:"AuthUrl,omitempty"`
	ConnectorArn string `json:"ConnectorArn,omitempty"`
	ConnectorId string `json:"ConnectorId,omitempty"`
	ConnectorStatus *string `json:"ConnectorStatus,omitempty"`
}

type CreateFindingAggregatorRequest struct {
	RegionLinkingMode string `json:"RegionLinkingMode,omitempty"`
	Regions []string `json:"Regions,omitempty"`
}

type CreateFindingAggregatorResponse struct {
	FindingAggregationRegion *string `json:"FindingAggregationRegion,omitempty"`
	FindingAggregatorArn *string `json:"FindingAggregatorArn,omitempty"`
	RegionLinkingMode *string `json:"RegionLinkingMode,omitempty"`
	Regions []string `json:"Regions,omitempty"`
}

type CreateInsightRequest struct {
	Filters AwsSecurityFindingFilters `json:"Filters,omitempty"`
	GroupByAttribute string `json:"GroupByAttribute,omitempty"`
	Name string `json:"Name,omitempty"`
}

type CreateInsightResponse struct {
	InsightArn string `json:"InsightArn,omitempty"`
}

type CreateMembersRequest struct {
	AccountDetails []AccountDetails `json:"AccountDetails,omitempty"`
}

type CreateMembersResponse struct {
	UnprocessedAccounts []Result `json:"UnprocessedAccounts,omitempty"`
}

type CreateTicketV2Request struct {
	ClientToken *string `json:"ClientToken,omitempty"`
	ConnectorId string `json:"ConnectorId,omitempty"`
	FindingMetadataUid string `json:"FindingMetadataUid,omitempty"`
	Mode *string `json:"Mode,omitempty"`
}

type CreateTicketV2Response struct {
	TicketId string `json:"TicketId,omitempty"`
	TicketSrcUrl *string `json:"TicketSrcUrl,omitempty"`
}

type Criteria struct {
	OcsfFindingCriteria *OcsfFindingFilters `json:"OcsfFindingCriteria,omitempty"`
}

type CustomDataIdentifiersDetections struct {
	Arn *string `json:"Arn,omitempty"`
	Count int64 `json:"Count,omitempty"`
	Name *string `json:"Name,omitempty"`
	Occurrences *Occurrences `json:"Occurrences,omitempty"`
}

type CustomDataIdentifiersResult struct {
	Detections []CustomDataIdentifiersDetections `json:"Detections,omitempty"`
	TotalCount int64 `json:"TotalCount,omitempty"`
}

type Cvss struct {
	Adjustments []Adjustment `json:"Adjustments,omitempty"`
	BaseScore float64 `json:"BaseScore,omitempty"`
	BaseVector *string `json:"BaseVector,omitempty"`
	Source *string `json:"Source,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type DataClassificationDetails struct {
	DetailedResultsLocation *string `json:"DetailedResultsLocation,omitempty"`
	Result *ClassificationResult `json:"Result,omitempty"`
}

type DateFilter struct {
	DateRange *DateRange `json:"DateRange,omitempty"`
	End *string `json:"End,omitempty"`
	Start *string `json:"Start,omitempty"`
}

type DateRange struct {
	Unit *string `json:"Unit,omitempty"`
	Value int `json:"Value,omitempty"`
}

type DeclineInvitationsRequest struct {
	AccountIds []string `json:"AccountIds,omitempty"`
}

type DeclineInvitationsResponse struct {
	UnprocessedAccounts []Result `json:"UnprocessedAccounts,omitempty"`
}

type DeleteActionTargetRequest struct {
	ActionTargetArn string `json:"ActionTargetArn,omitempty"`
}

type DeleteActionTargetResponse struct {
	ActionTargetArn string `json:"ActionTargetArn,omitempty"`
}

type DeleteAggregatorV2Request struct {
	AggregatorV2Arn string `json:"AggregatorV2Arn,omitempty"`
}

type DeleteAggregatorV2Response struct {
}

type DeleteAutomationRuleV2Request struct {
	Identifier string `json:"Identifier,omitempty"`
}

type DeleteAutomationRuleV2Response struct {
}

type DeleteConfigurationPolicyRequest struct {
	Identifier string `json:"Identifier,omitempty"`
}

type DeleteConfigurationPolicyResponse struct {
}

type DeleteConnectorV2Request struct {
	ConnectorId string `json:"ConnectorId,omitempty"`
}

type DeleteConnectorV2Response struct {
}

type DeleteFindingAggregatorRequest struct {
	FindingAggregatorArn string `json:"FindingAggregatorArn,omitempty"`
}

type DeleteFindingAggregatorResponse struct {
}

type DeleteInsightRequest struct {
	InsightArn string `json:"InsightArn,omitempty"`
}

type DeleteInsightResponse struct {
	InsightArn string `json:"InsightArn,omitempty"`
}

type DeleteInvitationsRequest struct {
	AccountIds []string `json:"AccountIds,omitempty"`
}

type DeleteInvitationsResponse struct {
	UnprocessedAccounts []Result `json:"UnprocessedAccounts,omitempty"`
}

type DeleteMembersRequest struct {
	AccountIds []string `json:"AccountIds,omitempty"`
}

type DeleteMembersResponse struct {
	UnprocessedAccounts []Result `json:"UnprocessedAccounts,omitempty"`
}

type DescribeActionTargetsRequest struct {
	ActionTargetArns []string `json:"ActionTargetArns,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeActionTargetsResponse struct {
	ActionTargets []ActionTarget `json:"ActionTargets,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeHubRequest struct {
	HubArn *string `json:"HubArn,omitempty"`
}

type DescribeHubResponse struct {
	AutoEnableControls bool `json:"AutoEnableControls,omitempty"`
	ControlFindingGenerator *string `json:"ControlFindingGenerator,omitempty"`
	HubArn *string `json:"HubArn,omitempty"`
	SubscribedAt *string `json:"SubscribedAt,omitempty"`
}

type DescribeOrganizationConfigurationRequest struct {
}

type DescribeOrganizationConfigurationResponse struct {
	AutoEnable bool `json:"AutoEnable,omitempty"`
	AutoEnableStandards *string `json:"AutoEnableStandards,omitempty"`
	MemberAccountLimitReached bool `json:"MemberAccountLimitReached,omitempty"`
	OrganizationConfiguration *OrganizationConfiguration `json:"OrganizationConfiguration,omitempty"`
}

type DescribeProductsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ProductArn *string `json:"ProductArn,omitempty"`
}

type DescribeProductsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	Products []Product `json:"Products,omitempty"`
}

type DescribeProductsV2Request struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeProductsV2Response struct {
	NextToken *string `json:"NextToken,omitempty"`
	ProductsV2 []ProductV2 `json:"ProductsV2,omitempty"`
}

type DescribeSecurityHubV2Request struct {
}

type DescribeSecurityHubV2Response struct {
	HubV2Arn *string `json:"HubV2Arn,omitempty"`
	SubscribedAt *string `json:"SubscribedAt,omitempty"`
}

type DescribeStandardsControlsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StandardsSubscriptionArn string `json:"StandardsSubscriptionArn,omitempty"`
}

type DescribeStandardsControlsResponse struct {
	Controls []StandardsControl `json:"Controls,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeStandardsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeStandardsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	Standards []Standard `json:"Standards,omitempty"`
}

type Detection struct {
	Sequence *Sequence `json:"Sequence,omitempty"`
}

type DisableImportFindingsForProductRequest struct {
	ProductSubscriptionArn string `json:"ProductSubscriptionArn,omitempty"`
}

type DisableImportFindingsForProductResponse struct {
}

type DisableOrganizationAdminAccountRequest struct {
	AdminAccountId string `json:"AdminAccountId,omitempty"`
	Feature *string `json:"Feature,omitempty"`
}

type DisableOrganizationAdminAccountResponse struct {
}

type DisableSecurityHubRequest struct {
}

type DisableSecurityHubResponse struct {
}

type DisableSecurityHubV2Request struct {
}

type DisableSecurityHubV2Response struct {
}

type DisassociateFromAdministratorAccountRequest struct {
}

type DisassociateFromAdministratorAccountResponse struct {
}

type DisassociateFromMasterAccountRequest struct {
}

type DisassociateFromMasterAccountResponse struct {
}

type DisassociateMembersRequest struct {
	AccountIds []string `json:"AccountIds,omitempty"`
}

type DisassociateMembersResponse struct {
}

type DnsRequestAction struct {
	Blocked bool `json:"Blocked,omitempty"`
	Domain *string `json:"Domain,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
}

type DoubleConfigurationOptions struct {
	DefaultValue float64 `json:"DefaultValue,omitempty"`
	Max float64 `json:"Max,omitempty"`
	Min float64 `json:"Min,omitempty"`
}

type EnableImportFindingsForProductRequest struct {
	ProductArn string `json:"ProductArn,omitempty"`
}

type EnableImportFindingsForProductResponse struct {
	ProductSubscriptionArn *string `json:"ProductSubscriptionArn,omitempty"`
}

type EnableOrganizationAdminAccountRequest struct {
	AdminAccountId string `json:"AdminAccountId,omitempty"`
	Feature *string `json:"Feature,omitempty"`
}

type EnableOrganizationAdminAccountResponse struct {
	AdminAccountId *string `json:"AdminAccountId,omitempty"`
	Feature *string `json:"Feature,omitempty"`
}

type EnableSecurityHubRequest struct {
	ControlFindingGenerator *string `json:"ControlFindingGenerator,omitempty"`
	EnableDefaultStandards bool `json:"EnableDefaultStandards,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type EnableSecurityHubResponse struct {
}

type EnableSecurityHubV2Request struct {
	Tags map[string]string `json:"Tags,omitempty"`
}

type EnableSecurityHubV2Response struct {
	HubV2Arn *string `json:"HubV2Arn,omitempty"`
}

type EnumConfigurationOptions struct {
	AllowedValues []string `json:"AllowedValues,omitempty"`
	DefaultValue *string `json:"DefaultValue,omitempty"`
}

type EnumListConfigurationOptions struct {
	AllowedValues []string `json:"AllowedValues,omitempty"`
	DefaultValue []string `json:"DefaultValue,omitempty"`
	MaxItems int `json:"MaxItems,omitempty"`
}

type ExternalIntegrationConfiguration struct {
	ConnectorArn *string `json:"ConnectorArn,omitempty"`
}

type FilePaths struct {
	FileName *string `json:"FileName,omitempty"`
	FilePath *string `json:"FilePath,omitempty"`
	Hash *string `json:"Hash,omitempty"`
	ResourceId *string `json:"ResourceId,omitempty"`
}

type FindingAggregator struct {
	FindingAggregatorArn *string `json:"FindingAggregatorArn,omitempty"`
}

type FindingHistoryRecord struct {
	FindingCreated bool `json:"FindingCreated,omitempty"`
	FindingIdentifier *AwsSecurityFindingIdentifier `json:"FindingIdentifier,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	UpdateSource *FindingHistoryUpdateSource `json:"UpdateSource,omitempty"`
	UpdateTime *time.Time `json:"UpdateTime,omitempty"`
	Updates []FindingHistoryUpdate `json:"Updates,omitempty"`
}

type FindingHistoryUpdate struct {
	NewValue *string `json:"NewValue,omitempty"`
	OldValue *string `json:"OldValue,omitempty"`
	UpdatedField *string `json:"UpdatedField,omitempty"`
}

type FindingHistoryUpdateSource struct {
	Identity *string `json:"Identity,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type FindingProviderFields struct {
	Confidence int `json:"Confidence,omitempty"`
	Criticality int `json:"Criticality,omitempty"`
	RelatedFindings []RelatedFinding `json:"RelatedFindings,omitempty"`
	Severity *FindingProviderSeverity `json:"Severity,omitempty"`
	Types []string `json:"Types,omitempty"`
}

type FindingProviderSeverity struct {
	Label *string `json:"Label,omitempty"`
	Original *string `json:"Original,omitempty"`
}

type FindingsTrendsCompositeFilter struct {
	NestedCompositeFilters []FindingsTrendsCompositeFilter `json:"NestedCompositeFilters,omitempty"`
	Operator *string `json:"Operator,omitempty"`
	StringFilters []FindingsTrendsStringFilter `json:"StringFilters,omitempty"`
}

type FindingsTrendsFilters struct {
	CompositeFilters []FindingsTrendsCompositeFilter `json:"CompositeFilters,omitempty"`
	CompositeOperator *string `json:"CompositeOperator,omitempty"`
}

type FindingsTrendsStringFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *StringFilter `json:"Filter,omitempty"`
}

type FirewallPolicyDetails struct {
	StatefulRuleGroupReferences []FirewallPolicyStatefulRuleGroupReferencesDetails `json:"StatefulRuleGroupReferences,omitempty"`
	StatelessCustomActions []FirewallPolicyStatelessCustomActionsDetails `json:"StatelessCustomActions,omitempty"`
	StatelessDefaultActions []string `json:"StatelessDefaultActions,omitempty"`
	StatelessFragmentDefaultActions []string `json:"StatelessFragmentDefaultActions,omitempty"`
	StatelessRuleGroupReferences []FirewallPolicyStatelessRuleGroupReferencesDetails `json:"StatelessRuleGroupReferences,omitempty"`
}

type FirewallPolicyStatefulRuleGroupReferencesDetails struct {
	ResourceArn *string `json:"ResourceArn,omitempty"`
}

type FirewallPolicyStatelessCustomActionsDetails struct {
	ActionDefinition *StatelessCustomActionDefinition `json:"ActionDefinition,omitempty"`
	ActionName *string `json:"ActionName,omitempty"`
}

type FirewallPolicyStatelessRuleGroupReferencesDetails struct {
	Priority int `json:"Priority,omitempty"`
	ResourceArn *string `json:"ResourceArn,omitempty"`
}

type GeneratorDetails struct {
	Description *string `json:"Description,omitempty"`
	Labels []string `json:"Labels,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type GeoLocation struct {
	Lat float64 `json:"Lat,omitempty"`
	Lon float64 `json:"Lon,omitempty"`
}

type GetAdministratorAccountRequest struct {
}

type GetAdministratorAccountResponse struct {
	Administrator *Invitation `json:"Administrator,omitempty"`
}

type GetAggregatorV2Request struct {
	AggregatorV2Arn string `json:"AggregatorV2Arn,omitempty"`
}

type GetAggregatorV2Response struct {
	AggregationRegion *string `json:"AggregationRegion,omitempty"`
	AggregatorV2Arn *string `json:"AggregatorV2Arn,omitempty"`
	LinkedRegions []string `json:"LinkedRegions,omitempty"`
	RegionLinkingMode *string `json:"RegionLinkingMode,omitempty"`
}

type GetAutomationRuleV2Request struct {
	Identifier string `json:"Identifier,omitempty"`
}

type GetAutomationRuleV2Response struct {
	Actions []AutomationRulesActionV2 `json:"Actions,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Criteria *Criteria `json:"Criteria,omitempty"`
	Description *string `json:"Description,omitempty"`
	RuleArn *string `json:"RuleArn,omitempty"`
	RuleId *string `json:"RuleId,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	RuleOrder float64 `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type GetConfigurationPolicyAssociationRequest struct {
	Target Target `json:"Target,omitempty"`
}

type GetConfigurationPolicyAssociationResponse struct {
	AssociationStatus *string `json:"AssociationStatus,omitempty"`
	AssociationStatusMessage *string `json:"AssociationStatusMessage,omitempty"`
	AssociationType *string `json:"AssociationType,omitempty"`
	ConfigurationPolicyId *string `json:"ConfigurationPolicyId,omitempty"`
	TargetId *string `json:"TargetId,omitempty"`
	TargetType *string `json:"TargetType,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type GetConfigurationPolicyRequest struct {
	Identifier string `json:"Identifier,omitempty"`
}

type GetConfigurationPolicyResponse struct {
	Arn *string `json:"Arn,omitempty"`
	ConfigurationPolicy *Policy `json:"ConfigurationPolicy,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type GetConnectorV2Request struct {
	ConnectorId string `json:"ConnectorId,omitempty"`
}

type GetConnectorV2Response struct {
	ConnectorArn *string `json:"ConnectorArn,omitempty"`
	ConnectorId string `json:"ConnectorId,omitempty"`
	CreatedAt time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	Health HealthCheck `json:"Health,omitempty"`
	KmsKeyArn *string `json:"KmsKeyArn,omitempty"`
	LastUpdatedAt time.Time `json:"LastUpdatedAt,omitempty"`
	Name string `json:"Name,omitempty"`
	ProviderDetail ProviderDetail `json:"ProviderDetail,omitempty"`
}

type GetEnabledStandardsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StandardsSubscriptionArns []string `json:"StandardsSubscriptionArns,omitempty"`
}

type GetEnabledStandardsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	StandardsSubscriptions []StandardsSubscription `json:"StandardsSubscriptions,omitempty"`
}

type GetFindingAggregatorRequest struct {
	FindingAggregatorArn string `json:"FindingAggregatorArn,omitempty"`
}

type GetFindingAggregatorResponse struct {
	FindingAggregationRegion *string `json:"FindingAggregationRegion,omitempty"`
	FindingAggregatorArn *string `json:"FindingAggregatorArn,omitempty"`
	RegionLinkingMode *string `json:"RegionLinkingMode,omitempty"`
	Regions []string `json:"Regions,omitempty"`
}

type GetFindingHistoryRequest struct {
	EndTime *time.Time `json:"EndTime,omitempty"`
	FindingIdentifier AwsSecurityFindingIdentifier `json:"FindingIdentifier,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StartTime *time.Time `json:"StartTime,omitempty"`
}

type GetFindingHistoryResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	Records []FindingHistoryRecord `json:"Records,omitempty"`
}

type GetFindingStatisticsV2Request struct {
	GroupByRules []GroupByRule `json:"GroupByRules,omitempty"`
	MaxStatisticResults int `json:"MaxStatisticResults,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
}

type GetFindingStatisticsV2Response struct {
	GroupByResults []GroupByResult `json:"GroupByResults,omitempty"`
}

type GetFindingsRequest struct {
	Filters *AwsSecurityFindingFilters `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortCriteria []SortCriterion `json:"SortCriteria,omitempty"`
}

type GetFindingsResponse struct {
	Findings []AwsSecurityFinding `json:"Findings,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetFindingsTrendsV2Request struct {
	EndTime time.Time `json:"EndTime,omitempty"`
	Filters *FindingsTrendsFilters `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StartTime time.Time `json:"StartTime,omitempty"`
}

type GetFindingsTrendsV2Response struct {
	Granularity string `json:"Granularity,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	TrendsMetrics []TrendsMetricsResult `json:"TrendsMetrics,omitempty"`
}

type GetFindingsV2Request struct {
	Filters *OcsfFindingFilters `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortCriteria []SortCriterion `json:"SortCriteria,omitempty"`
}

type GetFindingsV2Response struct {
	Findings []OcsfFinding `json:"Findings,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetInsightResultsRequest struct {
	InsightArn string `json:"InsightArn,omitempty"`
}

type GetInsightResultsResponse struct {
	InsightResults InsightResults `json:"InsightResults,omitempty"`
}

type GetInsightsRequest struct {
	InsightArns []string `json:"InsightArns,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetInsightsResponse struct {
	Insights []Insight `json:"Insights,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type GetInvitationsCountRequest struct {
}

type GetInvitationsCountResponse struct {
	InvitationsCount int `json:"InvitationsCount,omitempty"`
}

type GetMasterAccountRequest struct {
}

type GetMasterAccountResponse struct {
	Master *Invitation `json:"Master,omitempty"`
}

type GetMembersRequest struct {
	AccountIds []string `json:"AccountIds,omitempty"`
}

type GetMembersResponse struct {
	Members []Member `json:"Members,omitempty"`
	UnprocessedAccounts []Result `json:"UnprocessedAccounts,omitempty"`
}

type GetResourcesStatisticsV2Request struct {
	GroupByRules []ResourceGroupByRule `json:"GroupByRules,omitempty"`
	MaxStatisticResults int `json:"MaxStatisticResults,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
}

type GetResourcesStatisticsV2Response struct {
	GroupByResults []GroupByResult `json:"GroupByResults,omitempty"`
}

type GetResourcesTrendsV2Request struct {
	EndTime time.Time `json:"EndTime,omitempty"`
	Filters *ResourcesTrendsFilters `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StartTime time.Time `json:"StartTime,omitempty"`
}

type GetResourcesTrendsV2Response struct {
	Granularity string `json:"Granularity,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	TrendsMetrics []ResourcesTrendsMetricsResult `json:"TrendsMetrics,omitempty"`
}

type GetResourcesV2Request struct {
	Filters *ResourcesFilters `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SortCriteria []SortCriterion `json:"SortCriteria,omitempty"`
}

type GetResourcesV2Response struct {
	NextToken *string `json:"NextToken,omitempty"`
	Resources []ResourceResult `json:"Resources,omitempty"`
}

type GetSecurityControlDefinitionRequest struct {
	SecurityControlId string `json:"SecurityControlId,omitempty"`
}

type GetSecurityControlDefinitionResponse struct {
	SecurityControlDefinition SecurityControlDefinition `json:"SecurityControlDefinition,omitempty"`
}

type GroupByResult struct {
	GroupByField *string `json:"GroupByField,omitempty"`
	GroupByValues []GroupByValue `json:"GroupByValues,omitempty"`
}

type GroupByRule struct {
	Filters *OcsfFindingFilters `json:"Filters,omitempty"`
	GroupByField string `json:"GroupByField,omitempty"`
}

type GroupByValue struct {
	Count int `json:"Count,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
}

type HealthCheck struct {
	ConnectorStatus string `json:"ConnectorStatus,omitempty"`
	LastCheckedAt time.Time `json:"LastCheckedAt,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type IcmpTypeCode struct {
	Code int `json:"Code,omitempty"`
	Type int `json:"Type,omitempty"`
}

type ImportFindingsError struct {
	ErrorCode string `json:"ErrorCode,omitempty"`
	ErrorMessage string `json:"ErrorMessage,omitempty"`
	Id string `json:"Id,omitempty"`
}

type Indicator struct {
	Key *string `json:"Key,omitempty"`
	Title *string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
	Values []string `json:"Values,omitempty"`
}

type Insight struct {
	Filters AwsSecurityFindingFilters `json:"Filters,omitempty"`
	GroupByAttribute string `json:"GroupByAttribute,omitempty"`
	InsightArn string `json:"InsightArn,omitempty"`
	Name string `json:"Name,omitempty"`
}

type InsightResultValue struct {
	Count int `json:"Count,omitempty"`
	GroupByAttributeValue string `json:"GroupByAttributeValue,omitempty"`
}

type InsightResults struct {
	GroupByAttribute string `json:"GroupByAttribute,omitempty"`
	InsightArn string `json:"InsightArn,omitempty"`
	ResultValues []InsightResultValue `json:"ResultValues,omitempty"`
}

type IntegerConfigurationOptions struct {
	DefaultValue int `json:"DefaultValue,omitempty"`
	Max int `json:"Max,omitempty"`
	Min int `json:"Min,omitempty"`
}

type IntegerListConfigurationOptions struct {
	DefaultValue []int `json:"DefaultValue,omitempty"`
	Max int `json:"Max,omitempty"`
	MaxItems int `json:"MaxItems,omitempty"`
	Min int `json:"Min,omitempty"`
}

type Invitation struct {
	AccountId *string `json:"AccountId,omitempty"`
	InvitationId *string `json:"InvitationId,omitempty"`
	InvitedAt *time.Time `json:"InvitedAt,omitempty"`
	MemberStatus *string `json:"MemberStatus,omitempty"`
}

type InviteMembersRequest struct {
	AccountIds []string `json:"AccountIds,omitempty"`
}

type InviteMembersResponse struct {
	UnprocessedAccounts []Result `json:"UnprocessedAccounts,omitempty"`
}

type IpFilter struct {
	Cidr *string `json:"Cidr,omitempty"`
}

type IpOrganizationDetails struct {
	Asn int `json:"Asn,omitempty"`
	AsnOrg *string `json:"AsnOrg,omitempty"`
	Isp *string `json:"Isp,omitempty"`
	Org *string `json:"Org,omitempty"`
}

type Ipv6CidrBlockAssociation struct {
	AssociationId *string `json:"AssociationId,omitempty"`
	CidrBlockState *string `json:"CidrBlockState,omitempty"`
	Ipv6CidrBlock *string `json:"Ipv6CidrBlock,omitempty"`
}

type JiraCloudDetail struct {
	AuthStatus *string `json:"AuthStatus,omitempty"`
	AuthUrl *string `json:"AuthUrl,omitempty"`
	CloudId *string `json:"CloudId,omitempty"`
	Domain *string `json:"Domain,omitempty"`
	ProjectKey *string `json:"ProjectKey,omitempty"`
}

type JiraCloudProviderConfiguration struct {
	ProjectKey *string `json:"ProjectKey,omitempty"`
}

type JiraCloudUpdateConfiguration struct {
	ProjectKey *string `json:"ProjectKey,omitempty"`
}

type KeywordFilter struct {
	Value *string `json:"Value,omitempty"`
}

type ListAggregatorsV2Request struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListAggregatorsV2Response struct {
	AggregatorsV2 []AggregatorV2 `json:"AggregatorsV2,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListAutomationRulesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListAutomationRulesResponse struct {
	AutomationRulesMetadata []AutomationRulesMetadata `json:"AutomationRulesMetadata,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListAutomationRulesV2Request struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListAutomationRulesV2Response struct {
	NextToken *string `json:"NextToken,omitempty"`
	Rules []AutomationRulesMetadataV2 `json:"Rules,omitempty"`
}

type ListConfigurationPoliciesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListConfigurationPoliciesResponse struct {
	ConfigurationPolicySummaries []ConfigurationPolicySummary `json:"ConfigurationPolicySummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListConfigurationPolicyAssociationsRequest struct {
	Filters *AssociationFilters `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListConfigurationPolicyAssociationsResponse struct {
	ConfigurationPolicyAssociationSummaries []ConfigurationPolicyAssociationSummary `json:"ConfigurationPolicyAssociationSummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListConnectorsV2Request struct {
	ConnectorStatus *string `json:"ConnectorStatus,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ProviderName *string `json:"ProviderName,omitempty"`
}

type ListConnectorsV2Response struct {
	Connectors []ConnectorSummary `json:"Connectors,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEnabledProductsForImportRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEnabledProductsForImportResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	ProductSubscriptions []string `json:"ProductSubscriptions,omitempty"`
}

type ListFindingAggregatorsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListFindingAggregatorsResponse struct {
	FindingAggregators []FindingAggregator `json:"FindingAggregators,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListInvitationsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListInvitationsResponse struct {
	Invitations []Invitation `json:"Invitations,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListMembersRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	OnlyAssociated bool `json:"OnlyAssociated,omitempty"`
}

type ListMembersResponse struct {
	Members []Member `json:"Members,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListOrganizationAdminAccountsRequest struct {
	Feature *string `json:"Feature,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListOrganizationAdminAccountsResponse struct {
	AdminAccounts []AdminAccount `json:"AdminAccounts,omitempty"`
	Feature *string `json:"Feature,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListSecurityControlDefinitionsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	StandardsArn *string `json:"StandardsArn,omitempty"`
}

type ListSecurityControlDefinitionsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	SecurityControlDefinitions []SecurityControlDefinition `json:"SecurityControlDefinitions,omitempty"`
}

type ListStandardsControlAssociationsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
}

type ListStandardsControlAssociationsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	StandardsControlAssociationSummaries []StandardsControlAssociationSummary `json:"StandardsControlAssociationSummaries,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags map[string]string `json:"Tags,omitempty"`
}

type LoadBalancerState struct {
	Code *string `json:"Code,omitempty"`
	Reason *string `json:"Reason,omitempty"`
}

type Malware struct {
	Name string `json:"Name,omitempty"`
	Path *string `json:"Path,omitempty"`
	State *string `json:"State,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type MapFilter struct {
	Comparison *string `json:"Comparison,omitempty"`
	Key *string `json:"Key,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type Member struct {
	AccountId *string `json:"AccountId,omitempty"`
	AdministratorId *string `json:"AdministratorId,omitempty"`
	Email *string `json:"Email,omitempty"`
	InvitedAt *time.Time `json:"InvitedAt,omitempty"`
	MasterId *string `json:"MasterId,omitempty"`
	MemberStatus *string `json:"MemberStatus,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type Network struct {
	DestinationDomain *string `json:"DestinationDomain,omitempty"`
	DestinationIpV4 *string `json:"DestinationIpV4,omitempty"`
	DestinationIpV6 *string `json:"DestinationIpV6,omitempty"`
	DestinationPort int `json:"DestinationPort,omitempty"`
	Direction *string `json:"Direction,omitempty"`
	OpenPortRange *PortRange `json:"OpenPortRange,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	SourceDomain *string `json:"SourceDomain,omitempty"`
	SourceIpV4 *string `json:"SourceIpV4,omitempty"`
	SourceIpV6 *string `json:"SourceIpV6,omitempty"`
	SourceMac *string `json:"SourceMac,omitempty"`
	SourcePort int `json:"SourcePort,omitempty"`
}

type NetworkAutonomousSystem struct {
	Name *string `json:"Name,omitempty"`
	Number int `json:"Number,omitempty"`
}

type NetworkConnection struct {
	Direction *string `json:"Direction,omitempty"`
}

type NetworkConnectionAction struct {
	Blocked bool `json:"Blocked,omitempty"`
	ConnectionDirection *string `json:"ConnectionDirection,omitempty"`
	LocalPortDetails *ActionLocalPortDetails `json:"LocalPortDetails,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	RemoteIpDetails *ActionRemoteIpDetails `json:"RemoteIpDetails,omitempty"`
	RemotePortDetails *ActionRemotePortDetails `json:"RemotePortDetails,omitempty"`
}

type NetworkEndpoint struct {
	AutonomousSystem *NetworkAutonomousSystem `json:"AutonomousSystem,omitempty"`
	Connection *NetworkConnection `json:"Connection,omitempty"`
	Domain *string `json:"Domain,omitempty"`
	Id *string `json:"Id,omitempty"`
	Ip *string `json:"Ip,omitempty"`
	Location *NetworkGeoLocation `json:"Location,omitempty"`
	Port int `json:"Port,omitempty"`
}

type NetworkGeoLocation struct {
	City *string `json:"City,omitempty"`
	Country *string `json:"Country,omitempty"`
	Lat float64 `json:"Lat,omitempty"`
	Lon float64 `json:"Lon,omitempty"`
}

type NetworkHeader struct {
	Destination *NetworkPathComponentDetails `json:"Destination,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	Source *NetworkPathComponentDetails `json:"Source,omitempty"`
}

type NetworkPathComponent struct {
	ComponentId *string `json:"ComponentId,omitempty"`
	ComponentType *string `json:"ComponentType,omitempty"`
	Egress *NetworkHeader `json:"Egress,omitempty"`
	Ingress *NetworkHeader `json:"Ingress,omitempty"`
}

type NetworkPathComponentDetails struct {
	Address []string `json:"Address,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
}

type Note struct {
	Text string `json:"Text,omitempty"`
	UpdatedAt string `json:"UpdatedAt,omitempty"`
	UpdatedBy string `json:"UpdatedBy,omitempty"`
}

type NoteUpdate struct {
	Text string `json:"Text,omitempty"`
	UpdatedBy string `json:"UpdatedBy,omitempty"`
}

type NumberFilter struct {
	Eq float64 `json:"Eq,omitempty"`
	Gt float64 `json:"Gt,omitempty"`
	Gte float64 `json:"Gte,omitempty"`
	Lt float64 `json:"Lt,omitempty"`
	Lte float64 `json:"Lte,omitempty"`
}

type Occurrences struct {
	Cells []Cell `json:"Cells,omitempty"`
	LineRanges []Range `json:"LineRanges,omitempty"`
	OffsetRanges []Range `json:"OffsetRanges,omitempty"`
	Pages []Page `json:"Pages,omitempty"`
	Records []Record `json:"Records,omitempty"`
}

type OcsfBooleanFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *BooleanFilter `json:"Filter,omitempty"`
}

type OcsfDateFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *DateFilter `json:"Filter,omitempty"`
}

type OcsfFinding struct {
}

type OcsfFindingFilters struct {
	CompositeFilters []CompositeFilter `json:"CompositeFilters,omitempty"`
	CompositeOperator *string `json:"CompositeOperator,omitempty"`
}

type OcsfFindingIdentifier struct {
	CloudAccountUid string `json:"CloudAccountUid,omitempty"`
	FindingInfoUid string `json:"FindingInfoUid,omitempty"`
	MetadataProductUid string `json:"MetadataProductUid,omitempty"`
}

type OcsfIpFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *IpFilter `json:"Filter,omitempty"`
}

type OcsfMapFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *MapFilter `json:"Filter,omitempty"`
}

type OcsfNumberFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *NumberFilter `json:"Filter,omitempty"`
}

type OcsfStringFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *StringFilter `json:"Filter,omitempty"`
}

type OrganizationConfiguration struct {
	ConfigurationType *string `json:"ConfigurationType,omitempty"`
	Status *string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
}

type Page struct {
	LineRange *Range `json:"LineRange,omitempty"`
	OffsetRange *Range `json:"OffsetRange,omitempty"`
	PageNumber int64 `json:"PageNumber,omitempty"`
}

type ParameterConfiguration struct {
	Value *ParameterValue `json:"Value,omitempty"`
	ValueType string `json:"ValueType,omitempty"`
}

type ParameterDefinition struct {
	ConfigurationOptions ConfigurationOptions `json:"ConfigurationOptions,omitempty"`
	Description string `json:"Description,omitempty"`
}

type ParameterValue struct {
	Boolean bool `json:"Boolean,omitempty"`
	Double float64 `json:"Double,omitempty"`
	Enum *string `json:"Enum,omitempty"`
	EnumList []string `json:"EnumList,omitempty"`
	Integer int `json:"Integer,omitempty"`
	IntegerList []int `json:"IntegerList,omitempty"`
	String *string `json:"String,omitempty"`
	StringList []string `json:"StringList,omitempty"`
}

type PatchSummary struct {
	FailedCount int `json:"FailedCount,omitempty"`
	Id string `json:"Id,omitempty"`
	InstalledCount int `json:"InstalledCount,omitempty"`
	InstalledOtherCount int `json:"InstalledOtherCount,omitempty"`
	InstalledPendingReboot int `json:"InstalledPendingReboot,omitempty"`
	InstalledRejectedCount int `json:"InstalledRejectedCount,omitempty"`
	MissingCount int `json:"MissingCount,omitempty"`
	Operation *string `json:"Operation,omitempty"`
	OperationEndTime *string `json:"OperationEndTime,omitempty"`
	OperationStartTime *string `json:"OperationStartTime,omitempty"`
	RebootOption *string `json:"RebootOption,omitempty"`
}

type Policy struct {
	SecurityHub *SecurityHubPolicy `json:"SecurityHub,omitempty"`
}

type PortProbeAction struct {
	Blocked bool `json:"Blocked,omitempty"`
	PortProbeDetails []PortProbeDetail `json:"PortProbeDetails,omitempty"`
}

type PortProbeDetail struct {
	LocalIpDetails *ActionLocalIpDetails `json:"LocalIpDetails,omitempty"`
	LocalPortDetails *ActionLocalPortDetails `json:"LocalPortDetails,omitempty"`
	RemoteIpDetails *ActionRemoteIpDetails `json:"RemoteIpDetails,omitempty"`
}

type PortRange struct {
	Begin int `json:"Begin,omitempty"`
	End int `json:"End,omitempty"`
}

type PortRangeFromTo struct {
	From int `json:"From,omitempty"`
	To int `json:"To,omitempty"`
}

type ProcessDetails struct {
	LaunchedAt *string `json:"LaunchedAt,omitempty"`
	Name *string `json:"Name,omitempty"`
	ParentPid int `json:"ParentPid,omitempty"`
	Path *string `json:"Path,omitempty"`
	Pid int `json:"Pid,omitempty"`
	TerminatedAt *string `json:"TerminatedAt,omitempty"`
}

type Product struct {
	ActivationUrl *string `json:"ActivationUrl,omitempty"`
	Categories []string `json:"Categories,omitempty"`
	CompanyName *string `json:"CompanyName,omitempty"`
	Description *string `json:"Description,omitempty"`
	IntegrationTypes []string `json:"IntegrationTypes,omitempty"`
	MarketplaceUrl *string `json:"MarketplaceUrl,omitempty"`
	ProductArn string `json:"ProductArn,omitempty"`
	ProductName *string `json:"ProductName,omitempty"`
	ProductSubscriptionResourcePolicy *string `json:"ProductSubscriptionResourcePolicy,omitempty"`
}

type ProductV2 struct {
	ActivationUrl *string `json:"ActivationUrl,omitempty"`
	Categories []string `json:"Categories,omitempty"`
	CompanyName *string `json:"CompanyName,omitempty"`
	Description *string `json:"Description,omitempty"`
	IntegrationV2Types []string `json:"IntegrationV2Types,omitempty"`
	MarketplaceProductId *string `json:"MarketplaceProductId,omitempty"`
	MarketplaceUrl *string `json:"MarketplaceUrl,omitempty"`
	ProductV2Name *string `json:"ProductV2Name,omitempty"`
}

type PropagatingVgwSetDetails struct {
	GatewayId *string `json:"GatewayId,omitempty"`
}

type ProviderConfiguration struct {
	JiraCloud *JiraCloudProviderConfiguration `json:"JiraCloud,omitempty"`
	ServiceNow *ServiceNowProviderConfiguration `json:"ServiceNow,omitempty"`
}

type ProviderDetail struct {
	JiraCloud *JiraCloudDetail `json:"JiraCloud,omitempty"`
	ServiceNow *ServiceNowDetail `json:"ServiceNow,omitempty"`
}

type ProviderSummary struct {
	ConnectorStatus *string `json:"ConnectorStatus,omitempty"`
	ProviderName *string `json:"ProviderName,omitempty"`
}

type ProviderUpdateConfiguration struct {
	JiraCloud *JiraCloudUpdateConfiguration `json:"JiraCloud,omitempty"`
	ServiceNow *ServiceNowUpdateConfiguration `json:"ServiceNow,omitempty"`
}

type Range struct {
	End int64 `json:"End,omitempty"`
	Start int64 `json:"Start,omitempty"`
	StartColumn int64 `json:"StartColumn,omitempty"`
}

type Recommendation struct {
	Text *string `json:"Text,omitempty"`
	Url *string `json:"Url,omitempty"`
}

type Record struct {
	JsonPath *string `json:"JsonPath,omitempty"`
	RecordIndex int64 `json:"RecordIndex,omitempty"`
}

type RegisterConnectorV2Request struct {
	AuthCode string `json:"AuthCode,omitempty"`
	AuthState string `json:"AuthState,omitempty"`
}

type RegisterConnectorV2Response struct {
	ConnectorArn *string `json:"ConnectorArn,omitempty"`
	ConnectorId string `json:"ConnectorId,omitempty"`
}

type RelatedFinding struct {
	Id string `json:"Id,omitempty"`
	ProductArn string `json:"ProductArn,omitempty"`
}

type Remediation struct {
	Recommendation *Recommendation `json:"Recommendation,omitempty"`
}

type Resource struct {
	ApplicationArn *string `json:"ApplicationArn,omitempty"`
	ApplicationName *string `json:"ApplicationName,omitempty"`
	DataClassification *DataClassificationDetails `json:"DataClassification,omitempty"`
	Details *ResourceDetails `json:"Details,omitempty"`
	Id string `json:"Id,omitempty"`
	Partition *string `json:"Partition,omitempty"`
	Region *string `json:"Region,omitempty"`
	ResourceRole *string `json:"ResourceRole,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
	Type string `json:"Type,omitempty"`
}

type ResourceConfig struct {
}

type ResourceDetails struct {
	AwsAmazonMqBroker *AwsAmazonMqBrokerDetails `json:"AwsAmazonMqBroker,omitempty"`
	AwsApiGatewayRestApi *AwsApiGatewayRestApiDetails `json:"AwsApiGatewayRestApi,omitempty"`
	AwsApiGatewayStage *AwsApiGatewayStageDetails `json:"AwsApiGatewayStage,omitempty"`
	AwsApiGatewayV2Api *AwsApiGatewayV2ApiDetails `json:"AwsApiGatewayV2Api,omitempty"`
	AwsApiGatewayV2Stage *AwsApiGatewayV2StageDetails `json:"AwsApiGatewayV2Stage,omitempty"`
	AwsAppSyncGraphQlApi *AwsAppSyncGraphQlApiDetails `json:"AwsAppSyncGraphQlApi,omitempty"`
	AwsAthenaWorkGroup *AwsAthenaWorkGroupDetails `json:"AwsAthenaWorkGroup,omitempty"`
	AwsAutoScalingAutoScalingGroup *AwsAutoScalingAutoScalingGroupDetails `json:"AwsAutoScalingAutoScalingGroup,omitempty"`
	AwsAutoScalingLaunchConfiguration *AwsAutoScalingLaunchConfigurationDetails `json:"AwsAutoScalingLaunchConfiguration,omitempty"`
	AwsBackupBackupPlan *AwsBackupBackupPlanDetails `json:"AwsBackupBackupPlan,omitempty"`
	AwsBackupBackupVault *AwsBackupBackupVaultDetails `json:"AwsBackupBackupVault,omitempty"`
	AwsBackupRecoveryPoint *AwsBackupRecoveryPointDetails `json:"AwsBackupRecoveryPoint,omitempty"`
	AwsCertificateManagerCertificate *AwsCertificateManagerCertificateDetails `json:"AwsCertificateManagerCertificate,omitempty"`
	AwsCloudFormationStack *AwsCloudFormationStackDetails `json:"AwsCloudFormationStack,omitempty"`
	AwsCloudFrontDistribution *AwsCloudFrontDistributionDetails `json:"AwsCloudFrontDistribution,omitempty"`
	AwsCloudTrailTrail *AwsCloudTrailTrailDetails `json:"AwsCloudTrailTrail,omitempty"`
	AwsCloudWatchAlarm *AwsCloudWatchAlarmDetails `json:"AwsCloudWatchAlarm,omitempty"`
	AwsCodeBuildProject *AwsCodeBuildProjectDetails `json:"AwsCodeBuildProject,omitempty"`
	AwsDmsEndpoint *AwsDmsEndpointDetails `json:"AwsDmsEndpoint,omitempty"`
	AwsDmsReplicationInstance *AwsDmsReplicationInstanceDetails `json:"AwsDmsReplicationInstance,omitempty"`
	AwsDmsReplicationTask *AwsDmsReplicationTaskDetails `json:"AwsDmsReplicationTask,omitempty"`
	AwsDynamoDbTable *AwsDynamoDbTableDetails `json:"AwsDynamoDbTable,omitempty"`
	AwsEc2ClientVpnEndpoint *AwsEc2ClientVpnEndpointDetails `json:"AwsEc2ClientVpnEndpoint,omitempty"`
	AwsEc2Eip *AwsEc2EipDetails `json:"AwsEc2Eip,omitempty"`
	AwsEc2Instance *AwsEc2InstanceDetails `json:"AwsEc2Instance,omitempty"`
	AwsEc2LaunchTemplate *AwsEc2LaunchTemplateDetails `json:"AwsEc2LaunchTemplate,omitempty"`
	AwsEc2NetworkAcl *AwsEc2NetworkAclDetails `json:"AwsEc2NetworkAcl,omitempty"`
	AwsEc2NetworkInterface *AwsEc2NetworkInterfaceDetails `json:"AwsEc2NetworkInterface,omitempty"`
	AwsEc2RouteTable *AwsEc2RouteTableDetails `json:"AwsEc2RouteTable,omitempty"`
	AwsEc2SecurityGroup *AwsEc2SecurityGroupDetails `json:"AwsEc2SecurityGroup,omitempty"`
	AwsEc2Subnet *AwsEc2SubnetDetails `json:"AwsEc2Subnet,omitempty"`
	AwsEc2TransitGateway *AwsEc2TransitGatewayDetails `json:"AwsEc2TransitGateway,omitempty"`
	AwsEc2Volume *AwsEc2VolumeDetails `json:"AwsEc2Volume,omitempty"`
	AwsEc2Vpc *AwsEc2VpcDetails `json:"AwsEc2Vpc,omitempty"`
	AwsEc2VpcEndpointService *AwsEc2VpcEndpointServiceDetails `json:"AwsEc2VpcEndpointService,omitempty"`
	AwsEc2VpcPeeringConnection *AwsEc2VpcPeeringConnectionDetails `json:"AwsEc2VpcPeeringConnection,omitempty"`
	AwsEc2VpnConnection *AwsEc2VpnConnectionDetails `json:"AwsEc2VpnConnection,omitempty"`
	AwsEcrContainerImage *AwsEcrContainerImageDetails `json:"AwsEcrContainerImage,omitempty"`
	AwsEcrRepository *AwsEcrRepositoryDetails `json:"AwsEcrRepository,omitempty"`
	AwsEcsCluster *AwsEcsClusterDetails `json:"AwsEcsCluster,omitempty"`
	AwsEcsContainer *AwsEcsContainerDetails `json:"AwsEcsContainer,omitempty"`
	AwsEcsService *AwsEcsServiceDetails `json:"AwsEcsService,omitempty"`
	AwsEcsTask *AwsEcsTaskDetails `json:"AwsEcsTask,omitempty"`
	AwsEcsTaskDefinition *AwsEcsTaskDefinitionDetails `json:"AwsEcsTaskDefinition,omitempty"`
	AwsEfsAccessPoint *AwsEfsAccessPointDetails `json:"AwsEfsAccessPoint,omitempty"`
	AwsEksCluster *AwsEksClusterDetails `json:"AwsEksCluster,omitempty"`
	AwsElasticBeanstalkEnvironment *AwsElasticBeanstalkEnvironmentDetails `json:"AwsElasticBeanstalkEnvironment,omitempty"`
	AwsElasticsearchDomain *AwsElasticsearchDomainDetails `json:"AwsElasticsearchDomain,omitempty"`
	AwsElbLoadBalancer *AwsElbLoadBalancerDetails `json:"AwsElbLoadBalancer,omitempty"`
	AwsElbv2LoadBalancer *AwsElbv2LoadBalancerDetails `json:"AwsElbv2LoadBalancer,omitempty"`
	AwsEventSchemasRegistry *AwsEventSchemasRegistryDetails `json:"AwsEventSchemasRegistry,omitempty"`
	AwsEventsEndpoint *AwsEventsEndpointDetails `json:"AwsEventsEndpoint,omitempty"`
	AwsEventsEventbus *AwsEventsEventbusDetails `json:"AwsEventsEventbus,omitempty"`
	AwsGuardDutyDetector *AwsGuardDutyDetectorDetails `json:"AwsGuardDutyDetector,omitempty"`
	AwsIamAccessKey *AwsIamAccessKeyDetails `json:"AwsIamAccessKey,omitempty"`
	AwsIamGroup *AwsIamGroupDetails `json:"AwsIamGroup,omitempty"`
	AwsIamPolicy *AwsIamPolicyDetails `json:"AwsIamPolicy,omitempty"`
	AwsIamRole *AwsIamRoleDetails `json:"AwsIamRole,omitempty"`
	AwsIamUser *AwsIamUserDetails `json:"AwsIamUser,omitempty"`
	AwsKinesisStream *AwsKinesisStreamDetails `json:"AwsKinesisStream,omitempty"`
	AwsKmsKey *AwsKmsKeyDetails `json:"AwsKmsKey,omitempty"`
	AwsLambdaFunction *AwsLambdaFunctionDetails `json:"AwsLambdaFunction,omitempty"`
	AwsLambdaLayerVersion *AwsLambdaLayerVersionDetails `json:"AwsLambdaLayerVersion,omitempty"`
	AwsMskCluster *AwsMskClusterDetails `json:"AwsMskCluster,omitempty"`
	AwsNetworkFirewallFirewall *AwsNetworkFirewallFirewallDetails `json:"AwsNetworkFirewallFirewall,omitempty"`
	AwsNetworkFirewallFirewallPolicy *AwsNetworkFirewallFirewallPolicyDetails `json:"AwsNetworkFirewallFirewallPolicy,omitempty"`
	AwsNetworkFirewallRuleGroup *AwsNetworkFirewallRuleGroupDetails `json:"AwsNetworkFirewallRuleGroup,omitempty"`
	AwsOpenSearchServiceDomain *AwsOpenSearchServiceDomainDetails `json:"AwsOpenSearchServiceDomain,omitempty"`
	AwsRdsDbCluster *AwsRdsDbClusterDetails `json:"AwsRdsDbCluster,omitempty"`
	AwsRdsDbClusterSnapshot *AwsRdsDbClusterSnapshotDetails `json:"AwsRdsDbClusterSnapshot,omitempty"`
	AwsRdsDbInstance *AwsRdsDbInstanceDetails `json:"AwsRdsDbInstance,omitempty"`
	AwsRdsDbSecurityGroup *AwsRdsDbSecurityGroupDetails `json:"AwsRdsDbSecurityGroup,omitempty"`
	AwsRdsDbSnapshot *AwsRdsDbSnapshotDetails `json:"AwsRdsDbSnapshot,omitempty"`
	AwsRdsEventSubscription *AwsRdsEventSubscriptionDetails `json:"AwsRdsEventSubscription,omitempty"`
	AwsRedshiftCluster *AwsRedshiftClusterDetails `json:"AwsRedshiftCluster,omitempty"`
	AwsRoute53HostedZone *AwsRoute53HostedZoneDetails `json:"AwsRoute53HostedZone,omitempty"`
	AwsS3AccessPoint *AwsS3AccessPointDetails `json:"AwsS3AccessPoint,omitempty"`
	AwsS3AccountPublicAccessBlock *AwsS3AccountPublicAccessBlockDetails `json:"AwsS3AccountPublicAccessBlock,omitempty"`
	AwsS3Bucket *AwsS3BucketDetails `json:"AwsS3Bucket,omitempty"`
	AwsS3Object *AwsS3ObjectDetails `json:"AwsS3Object,omitempty"`
	AwsSageMakerNotebookInstance *AwsSageMakerNotebookInstanceDetails `json:"AwsSageMakerNotebookInstance,omitempty"`
	AwsSecretsManagerSecret *AwsSecretsManagerSecretDetails `json:"AwsSecretsManagerSecret,omitempty"`
	AwsSnsTopic *AwsSnsTopicDetails `json:"AwsSnsTopic,omitempty"`
	AwsSqsQueue *AwsSqsQueueDetails `json:"AwsSqsQueue,omitempty"`
	AwsSsmPatchCompliance *AwsSsmPatchComplianceDetails `json:"AwsSsmPatchCompliance,omitempty"`
	AwsStepFunctionStateMachine *AwsStepFunctionStateMachineDetails `json:"AwsStepFunctionStateMachine,omitempty"`
	AwsWafRateBasedRule *AwsWafRateBasedRuleDetails `json:"AwsWafRateBasedRule,omitempty"`
	AwsWafRegionalRateBasedRule *AwsWafRegionalRateBasedRuleDetails `json:"AwsWafRegionalRateBasedRule,omitempty"`
	AwsWafRegionalRule *AwsWafRegionalRuleDetails `json:"AwsWafRegionalRule,omitempty"`
	AwsWafRegionalRuleGroup *AwsWafRegionalRuleGroupDetails `json:"AwsWafRegionalRuleGroup,omitempty"`
	AwsWafRegionalWebAcl *AwsWafRegionalWebAclDetails `json:"AwsWafRegionalWebAcl,omitempty"`
	AwsWafRule *AwsWafRuleDetails `json:"AwsWafRule,omitempty"`
	AwsWafRuleGroup *AwsWafRuleGroupDetails `json:"AwsWafRuleGroup,omitempty"`
	AwsWafWebAcl *AwsWafWebAclDetails `json:"AwsWafWebAcl,omitempty"`
	AwsWafv2RuleGroup *AwsWafv2RuleGroupDetails `json:"AwsWafv2RuleGroup,omitempty"`
	AwsWafv2WebAcl *AwsWafv2WebAclDetails `json:"AwsWafv2WebAcl,omitempty"`
	AwsXrayEncryptionConfig *AwsXrayEncryptionConfigDetails `json:"AwsXrayEncryptionConfig,omitempty"`
	CodeRepository *CodeRepositoryDetails `json:"CodeRepository,omitempty"`
	Container *ContainerDetails `json:"Container,omitempty"`
	Other map[string]string `json:"Other,omitempty"`
}

type ResourceFindingsSummary struct {
	FindingType string `json:"FindingType,omitempty"`
	ProductName string `json:"ProductName,omitempty"`
	Severities *ResourceSeverityBreakdown `json:"Severities,omitempty"`
	TotalFindings int `json:"TotalFindings,omitempty"`
}

type ResourceGroupByRule struct {
	Filters *ResourcesFilters `json:"Filters,omitempty"`
	GroupByField string `json:"GroupByField,omitempty"`
}

type ResourceResult struct {
	AccountId string `json:"AccountId,omitempty"`
	FindingsSummary []ResourceFindingsSummary `json:"FindingsSummary,omitempty"`
	Region string `json:"Region,omitempty"`
	ResourceCategory *string `json:"ResourceCategory,omitempty"`
	ResourceConfig ResourceConfig `json:"ResourceConfig,omitempty"`
	ResourceCreationTimeDt *string `json:"ResourceCreationTimeDt,omitempty"`
	ResourceDetailCaptureTimeDt string `json:"ResourceDetailCaptureTimeDt,omitempty"`
	ResourceGuid *string `json:"ResourceGuid,omitempty"`
	ResourceId string `json:"ResourceId,omitempty"`
	ResourceName *string `json:"ResourceName,omitempty"`
	ResourceTags []ResourceTag `json:"ResourceTags,omitempty"`
	ResourceType *string `json:"ResourceType,omitempty"`
}

type ResourceSeverityBreakdown struct {
	Critical int `json:"Critical,omitempty"`
	Fatal int `json:"Fatal,omitempty"`
	High int `json:"High,omitempty"`
	Informational int `json:"Informational,omitempty"`
	Low int `json:"Low,omitempty"`
	Medium int `json:"Medium,omitempty"`
	Other int `json:"Other,omitempty"`
	Unknown int `json:"Unknown,omitempty"`
}

type ResourceTag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type ResourcesCompositeFilter struct {
	DateFilters []ResourcesDateFilter `json:"DateFilters,omitempty"`
	MapFilters []ResourcesMapFilter `json:"MapFilters,omitempty"`
	NestedCompositeFilters []ResourcesCompositeFilter `json:"NestedCompositeFilters,omitempty"`
	NumberFilters []ResourcesNumberFilter `json:"NumberFilters,omitempty"`
	Operator *string `json:"Operator,omitempty"`
	StringFilters []ResourcesStringFilter `json:"StringFilters,omitempty"`
}

type ResourcesCount struct {
	AllResources int64 `json:"AllResources,omitempty"`
}

type ResourcesDateFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *DateFilter `json:"Filter,omitempty"`
}

type ResourcesFilters struct {
	CompositeFilters []ResourcesCompositeFilter `json:"CompositeFilters,omitempty"`
	CompositeOperator *string `json:"CompositeOperator,omitempty"`
}

type ResourcesMapFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *MapFilter `json:"Filter,omitempty"`
}

type ResourcesNumberFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *NumberFilter `json:"Filter,omitempty"`
}

type ResourcesStringFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *StringFilter `json:"Filter,omitempty"`
}

type ResourcesTrendsCompositeFilter struct {
	NestedCompositeFilters []ResourcesTrendsCompositeFilter `json:"NestedCompositeFilters,omitempty"`
	Operator *string `json:"Operator,omitempty"`
	StringFilters []ResourcesTrendsStringFilter `json:"StringFilters,omitempty"`
}

type ResourcesTrendsFilters struct {
	CompositeFilters []ResourcesTrendsCompositeFilter `json:"CompositeFilters,omitempty"`
	CompositeOperator *string `json:"CompositeOperator,omitempty"`
}

type ResourcesTrendsMetricsResult struct {
	Timestamp time.Time `json:"Timestamp,omitempty"`
	TrendsValues ResourcesTrendsValues `json:"TrendsValues,omitempty"`
}

type ResourcesTrendsStringFilter struct {
	FieldName *string `json:"FieldName,omitempty"`
	Filter *StringFilter `json:"Filter,omitempty"`
}

type ResourcesTrendsValues struct {
	ResourcesCount ResourcesCount `json:"ResourcesCount,omitempty"`
}

type Result struct {
	AccountId *string `json:"AccountId,omitempty"`
	ProcessingResult *string `json:"ProcessingResult,omitempty"`
}

type RouteSetDetails struct {
	CarrierGatewayId *string `json:"CarrierGatewayId,omitempty"`
	CoreNetworkArn *string `json:"CoreNetworkArn,omitempty"`
	DestinationCidrBlock *string `json:"DestinationCidrBlock,omitempty"`
	DestinationIpv6CidrBlock *string `json:"DestinationIpv6CidrBlock,omitempty"`
	DestinationPrefixListId *string `json:"DestinationPrefixListId,omitempty"`
	EgressOnlyInternetGatewayId *string `json:"EgressOnlyInternetGatewayId,omitempty"`
	GatewayId *string `json:"GatewayId,omitempty"`
	InstanceId *string `json:"InstanceId,omitempty"`
	InstanceOwnerId *string `json:"InstanceOwnerId,omitempty"`
	LocalGatewayId *string `json:"LocalGatewayId,omitempty"`
	NatGatewayId *string `json:"NatGatewayId,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	Origin *string `json:"Origin,omitempty"`
	State *string `json:"State,omitempty"`
	TransitGatewayId *string `json:"TransitGatewayId,omitempty"`
	VpcPeeringConnectionId *string `json:"VpcPeeringConnectionId,omitempty"`
}

type RuleGroupDetails struct {
	RuleVariables *RuleGroupVariables `json:"RuleVariables,omitempty"`
	RulesSource *RuleGroupSource `json:"RulesSource,omitempty"`
}

type RuleGroupSource struct {
	RulesSourceList *RuleGroupSourceListDetails `json:"RulesSourceList,omitempty"`
	RulesString *string `json:"RulesString,omitempty"`
	StatefulRules []RuleGroupSourceStatefulRulesDetails `json:"StatefulRules,omitempty"`
	StatelessRulesAndCustomActions *RuleGroupSourceStatelessRulesAndCustomActionsDetails `json:"StatelessRulesAndCustomActions,omitempty"`
}

type RuleGroupSourceCustomActionsDetails struct {
	ActionDefinition *StatelessCustomActionDefinition `json:"ActionDefinition,omitempty"`
	ActionName *string `json:"ActionName,omitempty"`
}

type RuleGroupSourceListDetails struct {
	GeneratedRulesType *string `json:"GeneratedRulesType,omitempty"`
	TargetTypes []string `json:"TargetTypes,omitempty"`
	Targets []string `json:"Targets,omitempty"`
}

type RuleGroupSourceStatefulRulesDetails struct {
	Action *string `json:"Action,omitempty"`
	Header *RuleGroupSourceStatefulRulesHeaderDetails `json:"Header,omitempty"`
	RuleOptions []RuleGroupSourceStatefulRulesOptionsDetails `json:"RuleOptions,omitempty"`
}

type RuleGroupSourceStatefulRulesHeaderDetails struct {
	Destination *string `json:"Destination,omitempty"`
	DestinationPort *string `json:"DestinationPort,omitempty"`
	Direction *string `json:"Direction,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
	Source *string `json:"Source,omitempty"`
	SourcePort *string `json:"SourcePort,omitempty"`
}

type RuleGroupSourceStatefulRulesOptionsDetails struct {
	Keyword *string `json:"Keyword,omitempty"`
	Settings []string `json:"Settings,omitempty"`
}

type RuleGroupSourceStatelessRuleDefinition struct {
	Actions []string `json:"Actions,omitempty"`
	MatchAttributes *RuleGroupSourceStatelessRuleMatchAttributes `json:"MatchAttributes,omitempty"`
}

type RuleGroupSourceStatelessRuleMatchAttributes struct {
	DestinationPorts []RuleGroupSourceStatelessRuleMatchAttributesDestinationPorts `json:"DestinationPorts,omitempty"`
	Destinations []RuleGroupSourceStatelessRuleMatchAttributesDestinations `json:"Destinations,omitempty"`
	Protocols []int `json:"Protocols,omitempty"`
	SourcePorts []RuleGroupSourceStatelessRuleMatchAttributesSourcePorts `json:"SourcePorts,omitempty"`
	Sources []RuleGroupSourceStatelessRuleMatchAttributesSources `json:"Sources,omitempty"`
	TcpFlags []RuleGroupSourceStatelessRuleMatchAttributesTcpFlags `json:"TcpFlags,omitempty"`
}

type RuleGroupSourceStatelessRuleMatchAttributesDestinationPorts struct {
	FromPort int `json:"FromPort,omitempty"`
	ToPort int `json:"ToPort,omitempty"`
}

type RuleGroupSourceStatelessRuleMatchAttributesDestinations struct {
	AddressDefinition *string `json:"AddressDefinition,omitempty"`
}

type RuleGroupSourceStatelessRuleMatchAttributesSourcePorts struct {
	FromPort int `json:"FromPort,omitempty"`
	ToPort int `json:"ToPort,omitempty"`
}

type RuleGroupSourceStatelessRuleMatchAttributesSources struct {
	AddressDefinition *string `json:"AddressDefinition,omitempty"`
}

type RuleGroupSourceStatelessRuleMatchAttributesTcpFlags struct {
	Flags []string `json:"Flags,omitempty"`
	Masks []string `json:"Masks,omitempty"`
}

type RuleGroupSourceStatelessRulesAndCustomActionsDetails struct {
	CustomActions []RuleGroupSourceCustomActionsDetails `json:"CustomActions,omitempty"`
	StatelessRules []RuleGroupSourceStatelessRulesDetails `json:"StatelessRules,omitempty"`
}

type RuleGroupSourceStatelessRulesDetails struct {
	Priority int `json:"Priority,omitempty"`
	RuleDefinition *RuleGroupSourceStatelessRuleDefinition `json:"RuleDefinition,omitempty"`
}

type RuleGroupVariables struct {
	IpSets *RuleGroupVariablesIpSetsDetails `json:"IpSets,omitempty"`
	PortSets *RuleGroupVariablesPortSetsDetails `json:"PortSets,omitempty"`
}

type RuleGroupVariablesIpSetsDetails struct {
	Definition []string `json:"Definition,omitempty"`
}

type RuleGroupVariablesPortSetsDetails struct {
	Definition []string `json:"Definition,omitempty"`
}

type SecurityControl struct {
	Description string `json:"Description,omitempty"`
	LastUpdateReason *string `json:"LastUpdateReason,omitempty"`
	Parameters map[string]ParameterConfiguration `json:"Parameters,omitempty"`
	RemediationUrl string `json:"RemediationUrl,omitempty"`
	SecurityControlArn string `json:"SecurityControlArn,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
	SecurityControlStatus string `json:"SecurityControlStatus,omitempty"`
	SeverityRating string `json:"SeverityRating,omitempty"`
	Title string `json:"Title,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
}

type SecurityControlCustomParameter struct {
	Parameters map[string]ParameterConfiguration `json:"Parameters,omitempty"`
	SecurityControlId *string `json:"SecurityControlId,omitempty"`
}

type SecurityControlDefinition struct {
	CurrentRegionAvailability string `json:"CurrentRegionAvailability,omitempty"`
	CustomizableProperties []string `json:"CustomizableProperties,omitempty"`
	Description string `json:"Description,omitempty"`
	ParameterDefinitions map[string]ParameterDefinition `json:"ParameterDefinitions,omitempty"`
	RemediationUrl string `json:"RemediationUrl,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
	SeverityRating string `json:"SeverityRating,omitempty"`
	Title string `json:"Title,omitempty"`
}

type SecurityControlParameter struct {
	Name *string `json:"Name,omitempty"`
	Value []string `json:"Value,omitempty"`
}

type SecurityControlsConfiguration struct {
	DisabledSecurityControlIdentifiers []string `json:"DisabledSecurityControlIdentifiers,omitempty"`
	EnabledSecurityControlIdentifiers []string `json:"EnabledSecurityControlIdentifiers,omitempty"`
	SecurityControlCustomParameters []SecurityControlCustomParameter `json:"SecurityControlCustomParameters,omitempty"`
}

type SecurityHubPolicy struct {
	EnabledStandardIdentifiers []string `json:"EnabledStandardIdentifiers,omitempty"`
	SecurityControlsConfiguration *SecurityControlsConfiguration `json:"SecurityControlsConfiguration,omitempty"`
	ServiceEnabled bool `json:"ServiceEnabled,omitempty"`
}

type SensitiveDataDetections struct {
	Count int64 `json:"Count,omitempty"`
	Occurrences *Occurrences `json:"Occurrences,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type SensitiveDataResult struct {
	Category *string `json:"Category,omitempty"`
	Detections []SensitiveDataDetections `json:"Detections,omitempty"`
	TotalCount int64 `json:"TotalCount,omitempty"`
}

type Sequence struct {
	Actors []Actor `json:"Actors,omitempty"`
	Endpoints []NetworkEndpoint `json:"Endpoints,omitempty"`
	SequenceIndicators []Indicator `json:"SequenceIndicators,omitempty"`
	Signals []Signal `json:"Signals,omitempty"`
	Uid *string `json:"Uid,omitempty"`
}

type ServiceNowDetail struct {
	AuthStatus string `json:"AuthStatus,omitempty"`
	InstanceName *string `json:"InstanceName,omitempty"`
	SecretArn string `json:"SecretArn,omitempty"`
}

type ServiceNowProviderConfiguration struct {
	InstanceName string `json:"InstanceName,omitempty"`
	SecretArn string `json:"SecretArn,omitempty"`
}

type ServiceNowUpdateConfiguration struct {
	SecretArn *string `json:"SecretArn,omitempty"`
}

type Severity struct {
	Label *string `json:"Label,omitempty"`
	Normalized int `json:"Normalized,omitempty"`
	Original *string `json:"Original,omitempty"`
	Product float64 `json:"Product,omitempty"`
}

type SeverityTrendsCount struct {
	Critical int64 `json:"Critical,omitempty"`
	Fatal int64 `json:"Fatal,omitempty"`
	High int64 `json:"High,omitempty"`
	Informational int64 `json:"Informational,omitempty"`
	Low int64 `json:"Low,omitempty"`
	Medium int64 `json:"Medium,omitempty"`
	Other int64 `json:"Other,omitempty"`
	Unknown int64 `json:"Unknown,omitempty"`
}

type SeverityUpdate struct {
	Label *string `json:"Label,omitempty"`
	Normalized int `json:"Normalized,omitempty"`
	Product float64 `json:"Product,omitempty"`
}

type Signal struct {
	ActorIds []string `json:"ActorIds,omitempty"`
	Count int `json:"Count,omitempty"`
	CreatedAt int64 `json:"CreatedAt,omitempty"`
	EndpointIds []string `json:"EndpointIds,omitempty"`
	FirstSeenAt int64 `json:"FirstSeenAt,omitempty"`
	Id *string `json:"Id,omitempty"`
	LastSeenAt int64 `json:"LastSeenAt,omitempty"`
	Name *string `json:"Name,omitempty"`
	ProductArn *string `json:"ProductArn,omitempty"`
	ResourceIds []string `json:"ResourceIds,omitempty"`
	Severity float64 `json:"Severity,omitempty"`
	SignalIndicators []Indicator `json:"SignalIndicators,omitempty"`
	Title *string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
	UpdatedAt int64 `json:"UpdatedAt,omitempty"`
}

type SoftwarePackage struct {
	Architecture *string `json:"Architecture,omitempty"`
	Epoch *string `json:"Epoch,omitempty"`
	FilePath *string `json:"FilePath,omitempty"`
	FixedInVersion *string `json:"FixedInVersion,omitempty"`
	Name *string `json:"Name,omitempty"`
	PackageManager *string `json:"PackageManager,omitempty"`
	Release *string `json:"Release,omitempty"`
	Remediation *string `json:"Remediation,omitempty"`
	SourceLayerArn *string `json:"SourceLayerArn,omitempty"`
	SourceLayerHash *string `json:"SourceLayerHash,omitempty"`
	Version *string `json:"Version,omitempty"`
}

type SortCriterion struct {
	Field *string `json:"Field,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
}

type Standard struct {
	Description *string `json:"Description,omitempty"`
	EnabledByDefault bool `json:"EnabledByDefault,omitempty"`
	Name *string `json:"Name,omitempty"`
	StandardsArn *string `json:"StandardsArn,omitempty"`
	StandardsManagedBy *StandardsManagedBy `json:"StandardsManagedBy,omitempty"`
}

type StandardsControl struct {
	ControlId *string `json:"ControlId,omitempty"`
	ControlStatus *string `json:"ControlStatus,omitempty"`
	ControlStatusUpdatedAt *time.Time `json:"ControlStatusUpdatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	DisabledReason *string `json:"DisabledReason,omitempty"`
	RelatedRequirements []string `json:"RelatedRequirements,omitempty"`
	RemediationUrl *string `json:"RemediationUrl,omitempty"`
	SeverityRating *string `json:"SeverityRating,omitempty"`
	StandardsControlArn *string `json:"StandardsControlArn,omitempty"`
	Title *string `json:"Title,omitempty"`
}

type StandardsControlAssociationDetail struct {
	AssociationStatus string `json:"AssociationStatus,omitempty"`
	RelatedRequirements []string `json:"RelatedRequirements,omitempty"`
	SecurityControlArn string `json:"SecurityControlArn,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
	StandardsArn string `json:"StandardsArn,omitempty"`
	StandardsControlArns []string `json:"StandardsControlArns,omitempty"`
	StandardsControlDescription *string `json:"StandardsControlDescription,omitempty"`
	StandardsControlTitle *string `json:"StandardsControlTitle,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
	UpdatedReason *string `json:"UpdatedReason,omitempty"`
}

type StandardsControlAssociationId struct {
	SecurityControlId string `json:"SecurityControlId,omitempty"`
	StandardsArn string `json:"StandardsArn,omitempty"`
}

type StandardsControlAssociationSummary struct {
	AssociationStatus string `json:"AssociationStatus,omitempty"`
	RelatedRequirements []string `json:"RelatedRequirements,omitempty"`
	SecurityControlArn string `json:"SecurityControlArn,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
	StandardsArn string `json:"StandardsArn,omitempty"`
	StandardsControlDescription *string `json:"StandardsControlDescription,omitempty"`
	StandardsControlTitle *string `json:"StandardsControlTitle,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
	UpdatedReason *string `json:"UpdatedReason,omitempty"`
}

type StandardsControlAssociationUpdate struct {
	AssociationStatus string `json:"AssociationStatus,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
	StandardsArn string `json:"StandardsArn,omitempty"`
	UpdatedReason *string `json:"UpdatedReason,omitempty"`
}

type StandardsManagedBy struct {
	Company *string `json:"Company,omitempty"`
	Product *string `json:"Product,omitempty"`
}

type StandardsStatusReason struct {
	StatusReasonCode string `json:"StatusReasonCode,omitempty"`
}

type StandardsSubscription struct {
	StandardsArn string `json:"StandardsArn,omitempty"`
	StandardsControlsUpdatable *string `json:"StandardsControlsUpdatable,omitempty"`
	StandardsInput map[string]string `json:"StandardsInput,omitempty"`
	StandardsStatus string `json:"StandardsStatus,omitempty"`
	StandardsStatusReason *StandardsStatusReason `json:"StandardsStatusReason,omitempty"`
	StandardsSubscriptionArn string `json:"StandardsSubscriptionArn,omitempty"`
}

type StandardsSubscriptionRequest struct {
	StandardsArn string `json:"StandardsArn,omitempty"`
	StandardsInput map[string]string `json:"StandardsInput,omitempty"`
}

type StartConfigurationPolicyAssociationRequest struct {
	ConfigurationPolicyIdentifier string `json:"ConfigurationPolicyIdentifier,omitempty"`
	Target Target `json:"Target,omitempty"`
}

type StartConfigurationPolicyAssociationResponse struct {
	AssociationStatus *string `json:"AssociationStatus,omitempty"`
	AssociationStatusMessage *string `json:"AssociationStatusMessage,omitempty"`
	AssociationType *string `json:"AssociationType,omitempty"`
	ConfigurationPolicyId *string `json:"ConfigurationPolicyId,omitempty"`
	TargetId *string `json:"TargetId,omitempty"`
	TargetType *string `json:"TargetType,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type StartConfigurationPolicyDisassociationRequest struct {
	ConfigurationPolicyIdentifier string `json:"ConfigurationPolicyIdentifier,omitempty"`
	Target *Target `json:"Target,omitempty"`
}

type StartConfigurationPolicyDisassociationResponse struct {
}

type StatelessCustomActionDefinition struct {
	PublishMetricAction *StatelessCustomPublishMetricAction `json:"PublishMetricAction,omitempty"`
}

type StatelessCustomPublishMetricAction struct {
	Dimensions []StatelessCustomPublishMetricActionDimension `json:"Dimensions,omitempty"`
}

type StatelessCustomPublishMetricActionDimension struct {
	Value *string `json:"Value,omitempty"`
}

type StatusReason struct {
	Description *string `json:"Description,omitempty"`
	ReasonCode string `json:"ReasonCode,omitempty"`
}

type StringConfigurationOptions struct {
	DefaultValue *string `json:"DefaultValue,omitempty"`
	ExpressionDescription *string `json:"ExpressionDescription,omitempty"`
	Re2Expression *string `json:"Re2Expression,omitempty"`
}

type StringFilter struct {
	Comparison *string `json:"Comparison,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type StringListConfigurationOptions struct {
	DefaultValue []string `json:"DefaultValue,omitempty"`
	ExpressionDescription *string `json:"ExpressionDescription,omitempty"`
	MaxItems int `json:"MaxItems,omitempty"`
	Re2Expression *string `json:"Re2Expression,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	Tags map[string]string `json:"Tags,omitempty"`
}

type TagResourceResponse struct {
}

type Target struct {
	AccountId *string `json:"AccountId,omitempty"`
	OrganizationalUnitId *string `json:"OrganizationalUnitId,omitempty"`
	RootId *string `json:"RootId,omitempty"`
}

type Threat struct {
	FilePaths []FilePaths `json:"FilePaths,omitempty"`
	ItemCount int `json:"ItemCount,omitempty"`
	Name *string `json:"Name,omitempty"`
	Severity *string `json:"Severity,omitempty"`
}

type ThreatIntelIndicator struct {
	Category *string `json:"Category,omitempty"`
	LastObservedAt *string `json:"LastObservedAt,omitempty"`
	Source *string `json:"Source,omitempty"`
	SourceUrl *string `json:"SourceUrl,omitempty"`
	Type *string `json:"Type,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type TrendsMetricsResult struct {
	Timestamp time.Time `json:"Timestamp,omitempty"`
	TrendsValues TrendsValues `json:"TrendsValues,omitempty"`
}

type TrendsValues struct {
	SeverityTrends SeverityTrendsCount `json:"SeverityTrends,omitempty"`
}

type UnprocessedAutomationRule struct {
	ErrorCode int `json:"ErrorCode,omitempty"`
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	RuleArn *string `json:"RuleArn,omitempty"`
}

type UnprocessedConfigurationPolicyAssociation struct {
	ConfigurationPolicyAssociationIdentifiers *ConfigurationPolicyAssociation `json:"ConfigurationPolicyAssociationIdentifiers,omitempty"`
	ErrorCode *string `json:"ErrorCode,omitempty"`
	ErrorReason *string `json:"ErrorReason,omitempty"`
}

type UnprocessedSecurityControl struct {
	ErrorCode string `json:"ErrorCode,omitempty"`
	ErrorReason *string `json:"ErrorReason,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
}

type UnprocessedStandardsControlAssociation struct {
	ErrorCode string `json:"ErrorCode,omitempty"`
	ErrorReason *string `json:"ErrorReason,omitempty"`
	StandardsControlAssociationId StandardsControlAssociationId `json:"StandardsControlAssociationId,omitempty"`
}

type UnprocessedStandardsControlAssociationUpdate struct {
	ErrorCode string `json:"ErrorCode,omitempty"`
	ErrorReason *string `json:"ErrorReason,omitempty"`
	StandardsControlAssociationUpdate StandardsControlAssociationUpdate `json:"StandardsControlAssociationUpdate,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	TagKeys []string `json:"tagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateActionTargetRequest struct {
	ActionTargetArn string `json:"ActionTargetArn,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type UpdateActionTargetResponse struct {
}

type UpdateAggregatorV2Request struct {
	AggregatorV2Arn string `json:"AggregatorV2Arn,omitempty"`
	LinkedRegions []string `json:"LinkedRegions,omitempty"`
	RegionLinkingMode string `json:"RegionLinkingMode,omitempty"`
}

type UpdateAggregatorV2Response struct {
	AggregationRegion *string `json:"AggregationRegion,omitempty"`
	AggregatorV2Arn *string `json:"AggregatorV2Arn,omitempty"`
	LinkedRegions []string `json:"LinkedRegions,omitempty"`
	RegionLinkingMode *string `json:"RegionLinkingMode,omitempty"`
}

type UpdateAutomationRuleV2Request struct {
	Actions []AutomationRulesActionV2 `json:"Actions,omitempty"`
	Criteria *Criteria `json:"Criteria,omitempty"`
	Description *string `json:"Description,omitempty"`
	Identifier string `json:"Identifier,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	RuleOrder float64 `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
}

type UpdateAutomationRuleV2Response struct {
}

type UpdateAutomationRulesRequestItem struct {
	Actions []AutomationRulesAction `json:"Actions,omitempty"`
	Criteria *AutomationRulesFindingFilters `json:"Criteria,omitempty"`
	Description *string `json:"Description,omitempty"`
	IsTerminal bool `json:"IsTerminal,omitempty"`
	RuleArn string `json:"RuleArn,omitempty"`
	RuleName *string `json:"RuleName,omitempty"`
	RuleOrder int `json:"RuleOrder,omitempty"`
	RuleStatus *string `json:"RuleStatus,omitempty"`
}

type UpdateConfigurationPolicyRequest struct {
	ConfigurationPolicy *Policy `json:"ConfigurationPolicy,omitempty"`
	Description *string `json:"Description,omitempty"`
	Identifier string `json:"Identifier,omitempty"`
	Name *string `json:"Name,omitempty"`
	UpdatedReason *string `json:"UpdatedReason,omitempty"`
}

type UpdateConfigurationPolicyResponse struct {
	Arn *string `json:"Arn,omitempty"`
	ConfigurationPolicy *Policy `json:"ConfigurationPolicy,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
}

type UpdateConnectorV2Request struct {
	ConnectorId string `json:"ConnectorId,omitempty"`
	Description *string `json:"Description,omitempty"`
	Provider *ProviderUpdateConfiguration `json:"Provider,omitempty"`
}

type UpdateConnectorV2Response struct {
}

type UpdateFindingAggregatorRequest struct {
	FindingAggregatorArn string `json:"FindingAggregatorArn,omitempty"`
	RegionLinkingMode string `json:"RegionLinkingMode,omitempty"`
	Regions []string `json:"Regions,omitempty"`
}

type UpdateFindingAggregatorResponse struct {
	FindingAggregationRegion *string `json:"FindingAggregationRegion,omitempty"`
	FindingAggregatorArn *string `json:"FindingAggregatorArn,omitempty"`
	RegionLinkingMode *string `json:"RegionLinkingMode,omitempty"`
	Regions []string `json:"Regions,omitempty"`
}

type UpdateFindingsRequest struct {
	Filters AwsSecurityFindingFilters `json:"Filters,omitempty"`
	Note *NoteUpdate `json:"Note,omitempty"`
	RecordState *string `json:"RecordState,omitempty"`
}

type UpdateFindingsResponse struct {
}

type UpdateInsightRequest struct {
	Filters *AwsSecurityFindingFilters `json:"Filters,omitempty"`
	GroupByAttribute *string `json:"GroupByAttribute,omitempty"`
	InsightArn string `json:"InsightArn,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type UpdateInsightResponse struct {
}

type UpdateOrganizationConfigurationRequest struct {
	AutoEnable bool `json:"AutoEnable,omitempty"`
	AutoEnableStandards *string `json:"AutoEnableStandards,omitempty"`
	OrganizationConfiguration *OrganizationConfiguration `json:"OrganizationConfiguration,omitempty"`
}

type UpdateOrganizationConfigurationResponse struct {
}

type UpdateSecurityControlRequest struct {
	LastUpdateReason *string `json:"LastUpdateReason,omitempty"`
	Parameters map[string]ParameterConfiguration `json:"Parameters,omitempty"`
	SecurityControlId string `json:"SecurityControlId,omitempty"`
}

type UpdateSecurityControlResponse struct {
}

type UpdateSecurityHubConfigurationRequest struct {
	AutoEnableControls bool `json:"AutoEnableControls,omitempty"`
	ControlFindingGenerator *string `json:"ControlFindingGenerator,omitempty"`
}

type UpdateSecurityHubConfigurationResponse struct {
}

type UpdateStandardsControlRequest struct {
	ControlStatus *string `json:"ControlStatus,omitempty"`
	DisabledReason *string `json:"DisabledReason,omitempty"`
	StandardsControlArn string `json:"StandardsControlArn,omitempty"`
}

type UpdateStandardsControlResponse struct {
}

type UserAccount struct {
	Name *string `json:"Name,omitempty"`
	Uid *string `json:"Uid,omitempty"`
}

type VolumeMount struct {
	MountPath *string `json:"MountPath,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type VpcInfoCidrBlockSetDetails struct {
	CidrBlock *string `json:"CidrBlock,omitempty"`
}

type VpcInfoIpv6CidrBlockSetDetails struct {
	Ipv6CidrBlock *string `json:"Ipv6CidrBlock,omitempty"`
}

type VpcInfoPeeringOptionsDetails struct {
	AllowDnsResolutionFromRemoteVpc bool `json:"AllowDnsResolutionFromRemoteVpc,omitempty"`
	AllowEgressFromLocalClassicLinkToRemoteVpc bool `json:"AllowEgressFromLocalClassicLinkToRemoteVpc,omitempty"`
	AllowEgressFromLocalVpcToRemoteClassicLink bool `json:"AllowEgressFromLocalVpcToRemoteClassicLink,omitempty"`
}

type Vulnerability struct {
	CodeVulnerabilities []VulnerabilityCodeVulnerabilities `json:"CodeVulnerabilities,omitempty"`
	Cvss []Cvss `json:"Cvss,omitempty"`
	EpssScore float64 `json:"EpssScore,omitempty"`
	ExploitAvailable *string `json:"ExploitAvailable,omitempty"`
	FixAvailable *string `json:"FixAvailable,omitempty"`
	Id string `json:"Id,omitempty"`
	LastKnownExploitAt *string `json:"LastKnownExploitAt,omitempty"`
	ReferenceUrls []string `json:"ReferenceUrls,omitempty"`
	RelatedVulnerabilities []string `json:"RelatedVulnerabilities,omitempty"`
	Vendor *VulnerabilityVendor `json:"Vendor,omitempty"`
	VulnerablePackages []SoftwarePackage `json:"VulnerablePackages,omitempty"`
}

type VulnerabilityCodeVulnerabilities struct {
	Cwes []string `json:"Cwes,omitempty"`
	FilePath *CodeVulnerabilitiesFilePath `json:"FilePath,omitempty"`
	SourceArn *string `json:"SourceArn,omitempty"`
}

type VulnerabilityVendor struct {
	Name string `json:"Name,omitempty"`
	Url *string `json:"Url,omitempty"`
	VendorCreatedAt *string `json:"VendorCreatedAt,omitempty"`
	VendorSeverity *string `json:"VendorSeverity,omitempty"`
	VendorUpdatedAt *string `json:"VendorUpdatedAt,omitempty"`
}

type WafAction struct {
	Type *string `json:"Type,omitempty"`
}

type WafExcludedRule struct {
	RuleId *string `json:"RuleId,omitempty"`
}

type WafOverrideAction struct {
	Type *string `json:"Type,omitempty"`
}

type Workflow struct {
	Status *string `json:"Status,omitempty"`
}

type WorkflowUpdate struct {
	Status *string `json:"Status,omitempty"`
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

func handleBatchDeleteAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDeleteAutomationRulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDeleteAutomationRules business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDeleteAutomationRules"})
}

func handleBatchDisableStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDisableStandardsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDisableStandards business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDisableStandards"})
}

func handleBatchEnableStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchEnableStandardsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchEnableStandards business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchEnableStandards"})
}

func handleBatchGetAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetAutomationRulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetAutomationRules business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetAutomationRules"})
}

func handleBatchGetConfigurationPolicyAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetConfigurationPolicyAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetConfigurationPolicyAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetConfigurationPolicyAssociations"})
}

func handleBatchGetSecurityControls(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetSecurityControlsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetSecurityControls business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetSecurityControls"})
}

func handleBatchGetStandardsControlAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchGetStandardsControlAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchGetStandardsControlAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchGetStandardsControlAssociations"})
}

func handleBatchImportFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchImportFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchImportFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchImportFindings"})
}

func handleBatchUpdateAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchUpdateAutomationRulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchUpdateAutomationRules business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchUpdateAutomationRules"})
}

func handleBatchUpdateFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchUpdateFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchUpdateFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchUpdateFindings"})
}

func handleBatchUpdateFindingsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchUpdateFindingsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchUpdateFindingsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchUpdateFindingsV2"})
}

func handleBatchUpdateStandardsControlAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchUpdateStandardsControlAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchUpdateStandardsControlAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchUpdateStandardsControlAssociations"})
}

func handleCreateActionTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateActionTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateActionTarget business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateActionTarget"})
}

func handleCreateAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAggregatorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAggregatorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAggregatorV2"})
}

func handleCreateAutomationRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAutomationRuleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAutomationRule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAutomationRule"})
}

func handleCreateAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAutomationRuleV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAutomationRuleV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAutomationRuleV2"})
}

func handleCreateConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateConfigurationPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateConfigurationPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateConfigurationPolicy"})
}

func handleCreateConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateConnectorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateConnectorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateConnectorV2"})
}

func handleCreateFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFindingAggregatorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFindingAggregator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFindingAggregator"})
}

func handleCreateInsight(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateInsightRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateInsight business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateInsight"})
}

func handleCreateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateMembers"})
}

func handleCreateTicketV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTicketV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTicketV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTicketV2"})
}

func handleDeclineInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeclineInvitationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeclineInvitations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeclineInvitations"})
}

func handleDeleteActionTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteActionTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteActionTarget business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteActionTarget"})
}

func handleDeleteAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAggregatorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAggregatorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAggregatorV2"})
}

func handleDeleteAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAutomationRuleV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAutomationRuleV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAutomationRuleV2"})
}

func handleDeleteConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteConfigurationPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteConfigurationPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteConfigurationPolicy"})
}

func handleDeleteConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteConnectorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteConnectorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteConnectorV2"})
}

func handleDeleteFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFindingAggregatorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFindingAggregator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFindingAggregator"})
}

func handleDeleteInsight(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteInsightRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteInsight business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteInsight"})
}

func handleDeleteInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteInvitationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteInvitations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteInvitations"})
}

func handleDeleteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteMembers"})
}

func handleDescribeActionTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeActionTargetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeActionTargets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeActionTargets"})
}

func handleDescribeHub(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeHubRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeHub business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeHub"})
}

func handleDescribeOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeOrganizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeOrganizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeOrganizationConfiguration"})
}

func handleDescribeProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProductsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProducts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProducts"})
}

func handleDescribeProductsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeProductsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeProductsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeProductsV2"})
}

func handleDescribeSecurityHubV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeSecurityHubV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeSecurityHubV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeSecurityHubV2"})
}

func handleDescribeStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeStandardsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeStandards business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeStandards"})
}

func handleDescribeStandardsControls(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeStandardsControlsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeStandardsControls business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeStandardsControls"})
}

func handleDisableImportFindingsForProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableImportFindingsForProductRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableImportFindingsForProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableImportFindingsForProduct"})
}

func handleDisableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableOrganizationAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableOrganizationAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableOrganizationAdminAccount"})
}

func handleDisableSecurityHub(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableSecurityHubRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableSecurityHub business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableSecurityHub"})
}

func handleDisableSecurityHubV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DisableSecurityHubV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DisableSecurityHubV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DisableSecurityHubV2"})
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

func handleEnableImportFindingsForProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableImportFindingsForProductRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableImportFindingsForProduct business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableImportFindingsForProduct"})
}

func handleEnableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableOrganizationAdminAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableOrganizationAdminAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableOrganizationAdminAccount"})
}

func handleEnableSecurityHub(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableSecurityHubRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableSecurityHub business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableSecurityHub"})
}

func handleEnableSecurityHubV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req EnableSecurityHubV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement EnableSecurityHubV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "EnableSecurityHubV2"})
}

func handleGetAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetAdministratorAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetAdministratorAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetAdministratorAccount"})
}

func handleGetAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetAggregatorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetAggregatorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetAggregatorV2"})
}

func handleGetAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetAutomationRuleV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetAutomationRuleV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetAutomationRuleV2"})
}

func handleGetConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetConfigurationPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetConfigurationPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetConfigurationPolicy"})
}

func handleGetConfigurationPolicyAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetConfigurationPolicyAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetConfigurationPolicyAssociation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetConfigurationPolicyAssociation"})
}

func handleGetConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetConnectorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetConnectorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetConnectorV2"})
}

func handleGetEnabledStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetEnabledStandardsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetEnabledStandards business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetEnabledStandards"})
}

func handleGetFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingAggregatorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingAggregator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingAggregator"})
}

func handleGetFindingHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingHistoryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingHistory business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingHistory"})
}

func handleGetFindingStatisticsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingStatisticsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingStatisticsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingStatisticsV2"})
}

func handleGetFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindings"})
}

func handleGetFindingsTrendsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingsTrendsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingsTrendsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingsTrendsV2"})
}

func handleGetFindingsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFindingsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFindingsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFindingsV2"})
}

func handleGetInsightResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetInsightResultsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetInsightResults business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetInsightResults"})
}

func handleGetInsights(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetInsightsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetInsights business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetInsights"})
}

func handleGetInvitationsCount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetInvitationsCountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetInvitationsCount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetInvitationsCount"})
}

func handleGetMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMasterAccountRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMasterAccount business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMasterAccount"})
}

func handleGetMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetMembers"})
}

func handleGetResourcesStatisticsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetResourcesStatisticsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetResourcesStatisticsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetResourcesStatisticsV2"})
}

func handleGetResourcesTrendsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetResourcesTrendsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetResourcesTrendsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetResourcesTrendsV2"})
}

func handleGetResourcesV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetResourcesV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetResourcesV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetResourcesV2"})
}

func handleGetSecurityControlDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSecurityControlDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSecurityControlDefinition business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSecurityControlDefinition"})
}

func handleInviteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req InviteMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement InviteMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "InviteMembers"})
}

func handleListAggregatorsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAggregatorsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAggregatorsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAggregatorsV2"})
}

func handleListAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAutomationRulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAutomationRules business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAutomationRules"})
}

func handleListAutomationRulesV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAutomationRulesV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAutomationRulesV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAutomationRulesV2"})
}

func handleListConfigurationPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListConfigurationPoliciesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListConfigurationPolicies business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListConfigurationPolicies"})
}

func handleListConfigurationPolicyAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListConfigurationPolicyAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListConfigurationPolicyAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListConfigurationPolicyAssociations"})
}

func handleListConnectorsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListConnectorsV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListConnectorsV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListConnectorsV2"})
}

func handleListEnabledProductsForImport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEnabledProductsForImportRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEnabledProductsForImport business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEnabledProductsForImport"})
}

func handleListFindingAggregators(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFindingAggregatorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFindingAggregators business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFindingAggregators"})
}

func handleListInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListInvitationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListInvitations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListInvitations"})
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

func handleListSecurityControlDefinitions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListSecurityControlDefinitionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListSecurityControlDefinitions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListSecurityControlDefinitions"})
}

func handleListStandardsControlAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListStandardsControlAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListStandardsControlAssociations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListStandardsControlAssociations"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleRegisterConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RegisterConnectorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RegisterConnectorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RegisterConnectorV2"})
}

func handleStartConfigurationPolicyAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartConfigurationPolicyAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartConfigurationPolicyAssociation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartConfigurationPolicyAssociation"})
}

func handleStartConfigurationPolicyDisassociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartConfigurationPolicyDisassociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartConfigurationPolicyDisassociation business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartConfigurationPolicyDisassociation"})
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

func handleUpdateActionTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateActionTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateActionTarget business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateActionTarget"})
}

func handleUpdateAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAggregatorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAggregatorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAggregatorV2"})
}

func handleUpdateAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAutomationRuleV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAutomationRuleV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAutomationRuleV2"})
}

func handleUpdateConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateConfigurationPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateConfigurationPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateConfigurationPolicy"})
}

func handleUpdateConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateConnectorV2Request
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateConnectorV2 business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateConnectorV2"})
}

func handleUpdateFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFindingAggregatorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFindingAggregator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFindingAggregator"})
}

func handleUpdateFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFindingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFindings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFindings"})
}

func handleUpdateInsight(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateInsightRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateInsight business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateInsight"})
}

func handleUpdateOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateOrganizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateOrganizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateOrganizationConfiguration"})
}

func handleUpdateSecurityControl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateSecurityControlRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateSecurityControl business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateSecurityControl"})
}

func handleUpdateSecurityHubConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateSecurityHubConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateSecurityHubConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateSecurityHubConfiguration"})
}

func handleUpdateStandardsControl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateStandardsControlRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateStandardsControl business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateStandardsControl"})
}

