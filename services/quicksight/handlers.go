package quicksight

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type APIKeyConnectionMetadata struct {
	ApiKey string `json:"ApiKey,omitempty"`
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	Email *string `json:"Email,omitempty"`
}

type AccountCustomization struct {
	DefaultEmailCustomizationTemplate *string `json:"DefaultEmailCustomizationTemplate,omitempty"`
	DefaultTheme *string `json:"DefaultTheme,omitempty"`
}

type AccountInfo struct {
	AccountName *string `json:"AccountName,omitempty"`
	AccountSubscriptionStatus *string `json:"AccountSubscriptionStatus,omitempty"`
	AuthenticationType *string `json:"AuthenticationType,omitempty"`
	Edition *string `json:"Edition,omitempty"`
	IAMIdentityCenterInstanceArn *string `json:"IAMIdentityCenterInstanceArn,omitempty"`
	NotificationEmail *string `json:"NotificationEmail,omitempty"`
}

type AccountSettings struct {
	AccountName *string `json:"AccountName,omitempty"`
	DefaultNamespace *string `json:"DefaultNamespace,omitempty"`
	Edition *string `json:"Edition,omitempty"`
	NotificationEmail *string `json:"NotificationEmail,omitempty"`
	PublicSharingEnabled bool `json:"PublicSharingEnabled,omitempty"`
	TerminationProtectionEnabled bool `json:"TerminationProtectionEnabled,omitempty"`
}

type ActionConnector struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	Arn string `json:"Arn,omitempty"`
	AuthenticationConfig *ReadAuthConfig `json:"AuthenticationConfig,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	EnabledActions []string `json:"EnabledActions,omitempty"`
	Error *ActionConnectorError `json:"Error,omitempty"`
	LastUpdatedTime time.Time `json:"LastUpdatedTime,omitempty"`
	Name string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
	Type string `json:"Type,omitempty"`
	VpcConnectionArn *string `json:"VpcConnectionArn,omitempty"`
}

type ActionConnectorError struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ActionConnectorSearchFilter struct {
	Name string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value string `json:"Value,omitempty"`
}

type ActionConnectorSummary struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	Arn string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Error *ActionConnectorError `json:"Error,omitempty"`
	LastUpdatedTime time.Time `json:"LastUpdatedTime,omitempty"`
	Name string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
	Type string `json:"Type,omitempty"`
}

type ActiveIAMPolicyAssignment struct {
	AssignmentName *string `json:"AssignmentName,omitempty"`
	PolicyArn *string `json:"PolicyArn,omitempty"`
}

type AdHocFilteringOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type AggFunction struct {
	Aggregation *string `json:"Aggregation,omitempty"`
	AggregationFunctionParameters map[string]string `json:"AggregationFunctionParameters,omitempty"`
	Period *string `json:"Period,omitempty"`
	PeriodField *string `json:"PeriodField,omitempty"`
}

type AggregateOperation struct {
	Aggregations []Aggregation `json:"Aggregations,omitempty"`
	Alias string `json:"Alias,omitempty"`
	GroupByColumnNames []string `json:"GroupByColumnNames,omitempty"`
	Source TransformOperationSource `json:"Source,omitempty"`
}

type Aggregation struct {
	AggregationFunction DataPrepAggregationFunction `json:"AggregationFunction,omitempty"`
	NewColumnId string `json:"NewColumnId,omitempty"`
	NewColumnName string `json:"NewColumnName,omitempty"`
}

type AggregationFunction struct {
	AttributeAggregationFunction *AttributeAggregationFunction `json:"AttributeAggregationFunction,omitempty"`
	CategoricalAggregationFunction *string `json:"CategoricalAggregationFunction,omitempty"`
	DateAggregationFunction *string `json:"DateAggregationFunction,omitempty"`
	NumericalAggregationFunction *NumericalAggregationFunction `json:"NumericalAggregationFunction,omitempty"`
}

type AggregationPartitionBy struct {
	FieldName *string `json:"FieldName,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type AggregationSortConfiguration struct {
	AggregationFunction *AggregationFunction `json:"AggregationFunction,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	SortDirection string `json:"SortDirection,omitempty"`
}

type AllSheetsFilterScopeConfiguration struct {
}

type AmazonElasticsearchParameters struct {
	Domain string `json:"Domain,omitempty"`
}

type AmazonOpenSearchParameters struct {
	Domain string `json:"Domain,omitempty"`
}

type AmazonQInQuickSightConsoleConfigurations struct {
	DataQnA *DataQnAConfigurations `json:"DataQnA,omitempty"`
	DataStories *DataStoriesConfigurations `json:"DataStories,omitempty"`
	ExecutiveSummary *ExecutiveSummaryConfigurations `json:"ExecutiveSummary,omitempty"`
	GenerativeAuthoring *GenerativeAuthoringConfigurations `json:"GenerativeAuthoring,omitempty"`
}

type AmazonQInQuickSightDashboardConfigurations struct {
	ExecutiveSummary *ExecutiveSummaryConfigurations `json:"ExecutiveSummary,omitempty"`
}

type Analysis struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataSetArns []string `json:"DataSetArns,omitempty"`
	Errors []AnalysisError `json:"Errors,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	Sheets []Sheet `json:"Sheets,omitempty"`
	Status *string `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
}

type AnalysisDefaults struct {
	DefaultNewSheetConfiguration DefaultNewSheetConfiguration `json:"DefaultNewSheetConfiguration,omitempty"`
}

type AnalysisDefinition struct {
	AnalysisDefaults *AnalysisDefaults `json:"AnalysisDefaults,omitempty"`
	CalculatedFields []CalculatedField `json:"CalculatedFields,omitempty"`
	ColumnConfigurations []ColumnConfiguration `json:"ColumnConfigurations,omitempty"`
	DataSetIdentifierDeclarations []DataSetIdentifierDeclaration `json:"DataSetIdentifierDeclarations,omitempty"`
	FilterGroups []FilterGroup `json:"FilterGroups,omitempty"`
	Options *AssetOptions `json:"Options,omitempty"`
	ParameterDeclarations []ParameterDeclaration `json:"ParameterDeclarations,omitempty"`
	QueryExecutionOptions *QueryExecutionOptions `json:"QueryExecutionOptions,omitempty"`
	Sheets []SheetDefinition `json:"Sheets,omitempty"`
	StaticFiles []StaticFile `json:"StaticFiles,omitempty"`
	TooltipSheets []TooltipSheetDefinition `json:"TooltipSheets,omitempty"`
}

type AnalysisError struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
	ViolatedEntities []Entity `json:"ViolatedEntities,omitempty"`
}

type AnalysisSearchFilter struct {
	Name *string `json:"Name,omitempty"`
	Operator *string `json:"Operator,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type AnalysisSourceEntity struct {
	SourceTemplate *AnalysisSourceTemplate `json:"SourceTemplate,omitempty"`
}

type AnalysisSourceTemplate struct {
	Arn string `json:"Arn,omitempty"`
	DataSetReferences []DataSetReference `json:"DataSetReferences,omitempty"`
}

type AnalysisSummary struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type Anchor struct {
	AnchorType *string `json:"AnchorType,omitempty"`
	Offset int `json:"Offset,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type AnchorDateConfiguration struct {
	AnchorOption *string `json:"AnchorOption,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
}

type AnonymousUserDashboardEmbeddingConfiguration struct {
	DisabledFeatures []string `json:"DisabledFeatures,omitempty"`
	EnabledFeatures []string `json:"EnabledFeatures,omitempty"`
	FeatureConfigurations *AnonymousUserDashboardFeatureConfigurations `json:"FeatureConfigurations,omitempty"`
	InitialDashboardId string `json:"InitialDashboardId,omitempty"`
}

type AnonymousUserDashboardFeatureConfigurations struct {
	SharedView *SharedViewConfigurations `json:"SharedView,omitempty"`
}

type AnonymousUserDashboardVisualEmbeddingConfiguration struct {
	InitialDashboardVisualId DashboardVisualId `json:"InitialDashboardVisualId,omitempty"`
}

type AnonymousUserEmbeddingExperienceConfiguration struct {
	Dashboard *AnonymousUserDashboardEmbeddingConfiguration `json:"Dashboard,omitempty"`
	DashboardVisual *AnonymousUserDashboardVisualEmbeddingConfiguration `json:"DashboardVisual,omitempty"`
	GenerativeQnA *AnonymousUserGenerativeQnAEmbeddingConfiguration `json:"GenerativeQnA,omitempty"`
	QSearchBar *AnonymousUserQSearchBarEmbeddingConfiguration `json:"QSearchBar,omitempty"`
}

type AnonymousUserGenerativeQnAEmbeddingConfiguration struct {
	InitialTopicId string `json:"InitialTopicId,omitempty"`
}

type AnonymousUserQSearchBarEmbeddingConfiguration struct {
	InitialTopicId string `json:"InitialTopicId,omitempty"`
}

type AnonymousUserSnapshotJobResult struct {
	FileGroups []SnapshotJobResultFileGroup `json:"FileGroups,omitempty"`
}

type AppendOperation struct {
	Alias string `json:"Alias,omitempty"`
	AppendedColumns []AppendedColumn `json:"AppendedColumns,omitempty"`
	FirstSource *TransformOperationSource `json:"FirstSource,omitempty"`
	SecondSource *TransformOperationSource `json:"SecondSource,omitempty"`
}

type AppendedColumn struct {
	ColumnName string `json:"ColumnName,omitempty"`
	NewColumnId string `json:"NewColumnId,omitempty"`
}

type ApplicationTheme struct {
	BrandColorPalette *BrandColorPalette `json:"BrandColorPalette,omitempty"`
	BrandElementStyle *BrandElementStyle `json:"BrandElementStyle,omitempty"`
	ContextualAccentPalette *ContextualAccentPalette `json:"ContextualAccentPalette,omitempty"`
}

type ArcAxisConfiguration struct {
	Range *ArcAxisDisplayRange `json:"Range,omitempty"`
	ReserveRange int `json:"ReserveRange,omitempty"`
}

type ArcAxisDisplayRange struct {
	Max float64 `json:"Max,omitempty"`
	Min float64 `json:"Min,omitempty"`
}

type ArcConfiguration struct {
	ArcAngle float64 `json:"ArcAngle,omitempty"`
	ArcThickness *string `json:"ArcThickness,omitempty"`
}

type ArcOptions struct {
	ArcThickness *string `json:"ArcThickness,omitempty"`
}

type AssetBundleCloudFormationOverridePropertyConfiguration struct {
	Analyses []AssetBundleExportJobAnalysisOverrideProperties `json:"Analyses,omitempty"`
	Dashboards []AssetBundleExportJobDashboardOverrideProperties `json:"Dashboards,omitempty"`
	DataSets []AssetBundleExportJobDataSetOverrideProperties `json:"DataSets,omitempty"`
	DataSources []AssetBundleExportJobDataSourceOverrideProperties `json:"DataSources,omitempty"`
	Folders []AssetBundleExportJobFolderOverrideProperties `json:"Folders,omitempty"`
	RefreshSchedules []AssetBundleExportJobRefreshScheduleOverrideProperties `json:"RefreshSchedules,omitempty"`
	ResourceIdOverrideConfiguration *AssetBundleExportJobResourceIdOverrideConfiguration `json:"ResourceIdOverrideConfiguration,omitempty"`
	Themes []AssetBundleExportJobThemeOverrideProperties `json:"Themes,omitempty"`
	VPCConnections []AssetBundleExportJobVPCConnectionOverrideProperties `json:"VPCConnections,omitempty"`
}

type AssetBundleExportJobAnalysisOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobDashboardOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobDataSetOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobDataSourceOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobError struct {
	Arn *string `json:"Arn,omitempty"`
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AssetBundleExportJobFolderOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobRefreshScheduleOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobResourceIdOverrideConfiguration struct {
	PrefixForAllResources bool `json:"PrefixForAllResources,omitempty"`
}

type AssetBundleExportJobSummary struct {
	Arn *string `json:"Arn,omitempty"`
	AssetBundleExportJobId *string `json:"AssetBundleExportJobId,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	ExportFormat *string `json:"ExportFormat,omitempty"`
	IncludeAllDependencies bool `json:"IncludeAllDependencies,omitempty"`
	IncludePermissions bool `json:"IncludePermissions,omitempty"`
	IncludeTags bool `json:"IncludeTags,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type AssetBundleExportJobThemeOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobVPCConnectionOverrideProperties struct {
	Arn string `json:"Arn,omitempty"`
	Properties []string `json:"Properties,omitempty"`
}

type AssetBundleExportJobValidationStrategy struct {
	StrictModeForAllResources bool `json:"StrictModeForAllResources,omitempty"`
}

type AssetBundleExportJobWarning struct {
	Arn *string `json:"Arn,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type AssetBundleImportJobAnalysisOverrideParameters struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AssetBundleImportJobAnalysisOverridePermissions struct {
	AnalysisIds []string `json:"AnalysisIds,omitempty"`
	Permissions AssetBundleResourcePermissions `json:"Permissions,omitempty"`
}

type AssetBundleImportJobAnalysisOverrideTags struct {
	AnalysisIds []string `json:"AnalysisIds,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type AssetBundleImportJobDashboardOverrideParameters struct {
	DashboardId string `json:"DashboardId,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AssetBundleImportJobDashboardOverridePermissions struct {
	DashboardIds []string `json:"DashboardIds,omitempty"`
	LinkSharingConfiguration *AssetBundleResourceLinkSharingConfiguration `json:"LinkSharingConfiguration,omitempty"`
	Permissions *AssetBundleResourcePermissions `json:"Permissions,omitempty"`
}

type AssetBundleImportJobDashboardOverrideTags struct {
	DashboardIds []string `json:"DashboardIds,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type AssetBundleImportJobDataSetOverrideParameters struct {
	DataSetId string `json:"DataSetId,omitempty"`
	DataSetRefreshProperties *DataSetRefreshProperties `json:"DataSetRefreshProperties,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type AssetBundleImportJobDataSetOverridePermissions struct {
	DataSetIds []string `json:"DataSetIds,omitempty"`
	Permissions AssetBundleResourcePermissions `json:"Permissions,omitempty"`
}

type AssetBundleImportJobDataSetOverrideTags struct {
	DataSetIds []string `json:"DataSetIds,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type AssetBundleImportJobDataSourceCredentialPair struct {
	Password string `json:"Password,omitempty"`
	Username string `json:"Username,omitempty"`
}

type AssetBundleImportJobDataSourceCredentials struct {
	CredentialPair *AssetBundleImportJobDataSourceCredentialPair `json:"CredentialPair,omitempty"`
	SecretArn *string `json:"SecretArn,omitempty"`
}

type AssetBundleImportJobDataSourceOverrideParameters struct {
	Credentials *AssetBundleImportJobDataSourceCredentials `json:"Credentials,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
	DataSourceParameters *DataSourceParameters `json:"DataSourceParameters,omitempty"`
	Name *string `json:"Name,omitempty"`
	SslProperties *SslProperties `json:"SslProperties,omitempty"`
	VpcConnectionProperties *VpcConnectionProperties `json:"VpcConnectionProperties,omitempty"`
}

type AssetBundleImportJobDataSourceOverridePermissions struct {
	DataSourceIds []string `json:"DataSourceIds,omitempty"`
	Permissions AssetBundleResourcePermissions `json:"Permissions,omitempty"`
}

type AssetBundleImportJobDataSourceOverrideTags struct {
	DataSourceIds []string `json:"DataSourceIds,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type AssetBundleImportJobError struct {
	Arn *string `json:"Arn,omitempty"`
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type AssetBundleImportJobFolderOverrideParameters struct {
	FolderId string `json:"FolderId,omitempty"`
	Name *string `json:"Name,omitempty"`
	ParentFolderArn *string `json:"ParentFolderArn,omitempty"`
}

type AssetBundleImportJobFolderOverridePermissions struct {
	FolderIds []string `json:"FolderIds,omitempty"`
	Permissions *AssetBundleResourcePermissions `json:"Permissions,omitempty"`
}

type AssetBundleImportJobFolderOverrideTags struct {
	FolderIds []string `json:"FolderIds,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type AssetBundleImportJobOverrideParameters struct {
	Analyses []AssetBundleImportJobAnalysisOverrideParameters `json:"Analyses,omitempty"`
	Dashboards []AssetBundleImportJobDashboardOverrideParameters `json:"Dashboards,omitempty"`
	DataSets []AssetBundleImportJobDataSetOverrideParameters `json:"DataSets,omitempty"`
	DataSources []AssetBundleImportJobDataSourceOverrideParameters `json:"DataSources,omitempty"`
	Folders []AssetBundleImportJobFolderOverrideParameters `json:"Folders,omitempty"`
	RefreshSchedules []AssetBundleImportJobRefreshScheduleOverrideParameters `json:"RefreshSchedules,omitempty"`
	ResourceIdOverrideConfiguration *AssetBundleImportJobResourceIdOverrideConfiguration `json:"ResourceIdOverrideConfiguration,omitempty"`
	Themes []AssetBundleImportJobThemeOverrideParameters `json:"Themes,omitempty"`
	VPCConnections []AssetBundleImportJobVPCConnectionOverrideParameters `json:"VPCConnections,omitempty"`
}

type AssetBundleImportJobOverridePermissions struct {
	Analyses []AssetBundleImportJobAnalysisOverridePermissions `json:"Analyses,omitempty"`
	Dashboards []AssetBundleImportJobDashboardOverridePermissions `json:"Dashboards,omitempty"`
	DataSets []AssetBundleImportJobDataSetOverridePermissions `json:"DataSets,omitempty"`
	DataSources []AssetBundleImportJobDataSourceOverridePermissions `json:"DataSources,omitempty"`
	Folders []AssetBundleImportJobFolderOverridePermissions `json:"Folders,omitempty"`
	Themes []AssetBundleImportJobThemeOverridePermissions `json:"Themes,omitempty"`
}

type AssetBundleImportJobOverrideTags struct {
	Analyses []AssetBundleImportJobAnalysisOverrideTags `json:"Analyses,omitempty"`
	Dashboards []AssetBundleImportJobDashboardOverrideTags `json:"Dashboards,omitempty"`
	DataSets []AssetBundleImportJobDataSetOverrideTags `json:"DataSets,omitempty"`
	DataSources []AssetBundleImportJobDataSourceOverrideTags `json:"DataSources,omitempty"`
	Folders []AssetBundleImportJobFolderOverrideTags `json:"Folders,omitempty"`
	Themes []AssetBundleImportJobThemeOverrideTags `json:"Themes,omitempty"`
	VPCConnections []AssetBundleImportJobVPCConnectionOverrideTags `json:"VPCConnections,omitempty"`
}

type AssetBundleImportJobOverrideValidationStrategy struct {
	StrictModeForAllResources bool `json:"StrictModeForAllResources,omitempty"`
}

type AssetBundleImportJobRefreshScheduleOverrideParameters struct {
	DataSetId string `json:"DataSetId,omitempty"`
	ScheduleId string `json:"ScheduleId,omitempty"`
	StartAfterDateTime *time.Time `json:"StartAfterDateTime,omitempty"`
}

type AssetBundleImportJobResourceIdOverrideConfiguration struct {
	PrefixForAllResources *string `json:"PrefixForAllResources,omitempty"`
}

type AssetBundleImportJobSummary struct {
	Arn *string `json:"Arn,omitempty"`
	AssetBundleImportJobId *string `json:"AssetBundleImportJobId,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	FailureAction *string `json:"FailureAction,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
}

type AssetBundleImportJobThemeOverrideParameters struct {
	Name *string `json:"Name,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type AssetBundleImportJobThemeOverridePermissions struct {
	Permissions AssetBundleResourcePermissions `json:"Permissions,omitempty"`
	ThemeIds []string `json:"ThemeIds,omitempty"`
}

type AssetBundleImportJobThemeOverrideTags struct {
	Tags []Tag `json:"Tags,omitempty"`
	ThemeIds []string `json:"ThemeIds,omitempty"`
}

type AssetBundleImportJobVPCConnectionOverrideParameters struct {
	DnsResolvers []string `json:"DnsResolvers,omitempty"`
	Name *string `json:"Name,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
	VPCConnectionId string `json:"VPCConnectionId,omitempty"`
}

type AssetBundleImportJobVPCConnectionOverrideTags struct {
	Tags []Tag `json:"Tags,omitempty"`
	VPCConnectionIds []string `json:"VPCConnectionIds,omitempty"`
}

type AssetBundleImportJobWarning struct {
	Arn *string `json:"Arn,omitempty"`
	Message *string `json:"Message,omitempty"`
}

type AssetBundleImportSource struct {
	Body []byte `json:"Body,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
}

type AssetBundleImportSourceDescription struct {
	Body *string `json:"Body,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
}

type AssetBundleResourceLinkSharingConfiguration struct {
	Permissions *AssetBundleResourcePermissions `json:"Permissions,omitempty"`
}

type AssetBundleResourcePermissions struct {
	Actions []string `json:"Actions,omitempty"`
	Principals []string `json:"Principals,omitempty"`
}

type AssetOptions struct {
	CustomActionDefaults *VisualCustomActionDefaults `json:"CustomActionDefaults,omitempty"`
	ExcludedDataSetArns []string `json:"ExcludedDataSetArns,omitempty"`
	QBusinessInsightsStatus *string `json:"QBusinessInsightsStatus,omitempty"`
	Timezone *string `json:"Timezone,omitempty"`
	WeekStart *string `json:"WeekStart,omitempty"`
}

type AthenaParameters struct {
	IdentityCenterConfiguration *IdentityCenterConfiguration `json:"IdentityCenterConfiguration,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	WorkGroup *string `json:"WorkGroup,omitempty"`
}

type AttributeAggregationFunction struct {
	SimpleAttributeAggregation *string `json:"SimpleAttributeAggregation,omitempty"`
	ValueForMultipleValues *string `json:"ValueForMultipleValues,omitempty"`
}

type AuroraParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type AuroraPostgreSqlParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type AuthConfig struct {
	AuthenticationMetadata AuthenticationMetadata `json:"AuthenticationMetadata,omitempty"`
	AuthenticationType string `json:"AuthenticationType,omitempty"`
}

type AuthenticationMetadata struct {
	ApiKeyConnectionMetadata *APIKeyConnectionMetadata `json:"ApiKeyConnectionMetadata,omitempty"`
	AuthorizationCodeGrantMetadata *AuthorizationCodeGrantMetadata `json:"AuthorizationCodeGrantMetadata,omitempty"`
	BasicAuthConnectionMetadata *BasicAuthConnectionMetadata `json:"BasicAuthConnectionMetadata,omitempty"`
	ClientCredentialsGrantMetadata *ClientCredentialsGrantMetadata `json:"ClientCredentialsGrantMetadata,omitempty"`
	IamConnectionMetadata *IAMConnectionMetadata `json:"IamConnectionMetadata,omitempty"`
	NoneConnectionMetadata *NoneConnectionMetadata `json:"NoneConnectionMetadata,omitempty"`
}

type AuthorizationCodeGrantCredentialsDetails struct {
	AuthorizationCodeGrantDetails *AuthorizationCodeGrantDetails `json:"AuthorizationCodeGrantDetails,omitempty"`
}

type AuthorizationCodeGrantDetails struct {
	AuthorizationEndpoint string `json:"AuthorizationEndpoint,omitempty"`
	ClientId string `json:"ClientId,omitempty"`
	ClientSecret string `json:"ClientSecret,omitempty"`
	TokenEndpoint string `json:"TokenEndpoint,omitempty"`
}

type AuthorizationCodeGrantMetadata struct {
	AuthorizationCodeGrantCredentialsDetails *AuthorizationCodeGrantCredentialsDetails `json:"AuthorizationCodeGrantCredentialsDetails,omitempty"`
	AuthorizationCodeGrantCredentialsSource *string `json:"AuthorizationCodeGrantCredentialsSource,omitempty"`
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	RedirectUrl string `json:"RedirectUrl,omitempty"`
}

type AuthorizedTargetsByService struct {
	AuthorizedTargets []string `json:"AuthorizedTargets,omitempty"`
	ServiceModel *string `json:"Service,omitempty"`
}

type AwsIotAnalyticsParameters struct {
	DataSetName string `json:"DataSetName,omitempty"`
}

type AxisDataOptions struct {
	DateAxisOptions *DateAxisOptions `json:"DateAxisOptions,omitempty"`
	NumericAxisOptions *NumericAxisOptions `json:"NumericAxisOptions,omitempty"`
}

type AxisDisplayDataDrivenRange struct {
}

type AxisDisplayMinMaxRange struct {
	Maximum float64 `json:"Maximum,omitempty"`
	Minimum float64 `json:"Minimum,omitempty"`
}

type AxisDisplayOptions struct {
	AxisLineVisibility *string `json:"AxisLineVisibility,omitempty"`
	AxisOffset *string `json:"AxisOffset,omitempty"`
	DataOptions *AxisDataOptions `json:"DataOptions,omitempty"`
	GridLineVisibility *string `json:"GridLineVisibility,omitempty"`
	ScrollbarOptions *ScrollBarOptions `json:"ScrollbarOptions,omitempty"`
	TickLabelOptions *AxisTickLabelOptions `json:"TickLabelOptions,omitempty"`
}

type AxisDisplayRange struct {
	DataDriven *AxisDisplayDataDrivenRange `json:"DataDriven,omitempty"`
	MinMax *AxisDisplayMinMaxRange `json:"MinMax,omitempty"`
}

type AxisLabelOptions struct {
	ApplyTo *AxisLabelReferenceOptions `json:"ApplyTo,omitempty"`
	CustomLabel *string `json:"CustomLabel,omitempty"`
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
}

type AxisLabelReferenceOptions struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
}

type AxisLinearScale struct {
	StepCount int `json:"StepCount,omitempty"`
	StepSize float64 `json:"StepSize,omitempty"`
}

type AxisLogarithmicScale struct {
	Base float64 `json:"Base,omitempty"`
}

type AxisScale struct {
	Linear *AxisLinearScale `json:"Linear,omitempty"`
	Logarithmic *AxisLogarithmicScale `json:"Logarithmic,omitempty"`
}

type AxisTickLabelOptions struct {
	LabelOptions *LabelOptions `json:"LabelOptions,omitempty"`
	RotationAngle float64 `json:"RotationAngle,omitempty"`
}

type BarChartAggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	Colors []DimensionField `json:"Colors,omitempty"`
	SmallMultiples []DimensionField `json:"SmallMultiples,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type BarChartConfiguration struct {
	BarsArrangement *string `json:"BarsArrangement,omitempty"`
	CategoryAxis *AxisDisplayOptions `json:"CategoryAxis,omitempty"`
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	ColorLabelOptions *ChartAxisLabelOptions `json:"ColorLabelOptions,omitempty"`
	ContributionAnalysisDefaults []ContributionAnalysisDefault `json:"ContributionAnalysisDefaults,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	DefaultSeriesSettings *BarChartDefaultSeriesSettings `json:"DefaultSeriesSettings,omitempty"`
	FieldWells *BarChartFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	Orientation *string `json:"Orientation,omitempty"`
	ReferenceLines []ReferenceLine `json:"ReferenceLines,omitempty"`
	Series []BarSeriesItem `json:"Series,omitempty"`
	SmallMultiplesOptions *SmallMultiplesOptions `json:"SmallMultiplesOptions,omitempty"`
	SortConfiguration *BarChartSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	ValueAxis *AxisDisplayOptions `json:"ValueAxis,omitempty"`
	ValueLabelOptions *ChartAxisLabelOptions `json:"ValueLabelOptions,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type BarChartDefaultSeriesSettings struct {
	BorderSettings *BorderSettings `json:"BorderSettings,omitempty"`
	DecalSettings *DecalSettings `json:"DecalSettings,omitempty"`
}

type BarChartFieldWells struct {
	BarChartAggregatedFieldWells *BarChartAggregatedFieldWells `json:"BarChartAggregatedFieldWells,omitempty"`
}

type BarChartSeriesSettings struct {
	BorderSettings *BorderSettings `json:"BorderSettings,omitempty"`
	DecalSettings *DecalSettings `json:"DecalSettings,omitempty"`
}

type BarChartSortConfiguration struct {
	CategoryItemsLimit *ItemsLimitConfiguration `json:"CategoryItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
	ColorItemsLimit *ItemsLimitConfiguration `json:"ColorItemsLimit,omitempty"`
	ColorSort []FieldSortOptions `json:"ColorSort,omitempty"`
	SmallMultiplesLimitConfiguration *ItemsLimitConfiguration `json:"SmallMultiplesLimitConfiguration,omitempty"`
	SmallMultiplesSort []FieldSortOptions `json:"SmallMultiplesSort,omitempty"`
}

type BarChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *BarChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type BarSeriesItem struct {
	DataFieldBarSeriesItem *DataFieldBarSeriesItem `json:"DataFieldBarSeriesItem,omitempty"`
	FieldBarSeriesItem *FieldBarSeriesItem `json:"FieldBarSeriesItem,omitempty"`
}

type BasicAuthConnectionMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	Password string `json:"Password,omitempty"`
	Username string `json:"Username,omitempty"`
}

type BatchCreateTopicReviewedAnswerRequest struct {
	Answers []CreateTopicReviewedAnswer `json:"Answers,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type BatchCreateTopicReviewedAnswerResponse struct {
	InvalidAnswers []InvalidTopicReviewedAnswer `json:"InvalidAnswers,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	SucceededAnswers []SucceededTopicReviewedAnswer `json:"SucceededAnswers,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type BatchDeleteTopicReviewedAnswerRequest struct {
	AnswerIds []string `json:"AnswerIds,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type BatchDeleteTopicReviewedAnswerResponse struct {
	InvalidAnswers []InvalidTopicReviewedAnswer `json:"InvalidAnswers,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	SucceededAnswers []SucceededTopicReviewedAnswer `json:"SucceededAnswers,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type BigQueryParameters struct {
	DataSetRegion *string `json:"DataSetRegion,omitempty"`
	ProjectId string `json:"ProjectId,omitempty"`
}

type BinCountOptions struct {
	Value int `json:"Value,omitempty"`
}

type BinWidthOptions struct {
	BinCountLimit int64 `json:"BinCountLimit,omitempty"`
	Value float64 `json:"Value,omitempty"`
}

type BodySectionConfiguration struct {
	Content BodySectionContent `json:"Content,omitempty"`
	PageBreakConfiguration *SectionPageBreakConfiguration `json:"PageBreakConfiguration,omitempty"`
	RepeatConfiguration *BodySectionRepeatConfiguration `json:"RepeatConfiguration,omitempty"`
	SectionId string `json:"SectionId,omitempty"`
	Style *SectionStyle `json:"Style,omitempty"`
}

type BodySectionContent struct {
	Layout *SectionLayoutConfiguration `json:"Layout,omitempty"`
}

type BodySectionDynamicCategoryDimensionConfiguration struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	Limit int `json:"Limit,omitempty"`
	SortByMetrics []ColumnSort `json:"SortByMetrics,omitempty"`
}

type BodySectionDynamicNumericDimensionConfiguration struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	Limit int `json:"Limit,omitempty"`
	SortByMetrics []ColumnSort `json:"SortByMetrics,omitempty"`
}

type BodySectionRepeatConfiguration struct {
	DimensionConfigurations []BodySectionRepeatDimensionConfiguration `json:"DimensionConfigurations,omitempty"`
	NonRepeatingVisuals []string `json:"NonRepeatingVisuals,omitempty"`
	PageBreakConfiguration *BodySectionRepeatPageBreakConfiguration `json:"PageBreakConfiguration,omitempty"`
}

type BodySectionRepeatDimensionConfiguration struct {
	DynamicCategoryDimensionConfiguration *BodySectionDynamicCategoryDimensionConfiguration `json:"DynamicCategoryDimensionConfiguration,omitempty"`
	DynamicNumericDimensionConfiguration *BodySectionDynamicNumericDimensionConfiguration `json:"DynamicNumericDimensionConfiguration,omitempty"`
}

type BodySectionRepeatPageBreakConfiguration struct {
	After *SectionAfterPageBreak `json:"After,omitempty"`
}

type BookmarksConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type BorderSettings struct {
	BorderColor *string `json:"BorderColor,omitempty"`
	BorderVisibility *string `json:"BorderVisibility,omitempty"`
	BorderWidth *string `json:"BorderWidth,omitempty"`
}

type BorderStyle struct {
	Color *string `json:"Color,omitempty"`
	Show bool `json:"Show,omitempty"`
	Width *string `json:"Width,omitempty"`
}

type BoxPlotAggregatedFieldWells struct {
	GroupBy []DimensionField `json:"GroupBy,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type BoxPlotChartConfiguration struct {
	BoxPlotOptions *BoxPlotOptions `json:"BoxPlotOptions,omitempty"`
	CategoryAxis *AxisDisplayOptions `json:"CategoryAxis,omitempty"`
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	FieldWells *BoxPlotFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	PrimaryYAxisDisplayOptions *AxisDisplayOptions `json:"PrimaryYAxisDisplayOptions,omitempty"`
	PrimaryYAxisLabelOptions *ChartAxisLabelOptions `json:"PrimaryYAxisLabelOptions,omitempty"`
	ReferenceLines []ReferenceLine `json:"ReferenceLines,omitempty"`
	SortConfiguration *BoxPlotSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type BoxPlotFieldWells struct {
	BoxPlotAggregatedFieldWells *BoxPlotAggregatedFieldWells `json:"BoxPlotAggregatedFieldWells,omitempty"`
}

type BoxPlotOptions struct {
	AllDataPointsVisibility *string `json:"AllDataPointsVisibility,omitempty"`
	OutlierVisibility *string `json:"OutlierVisibility,omitempty"`
	StyleOptions *BoxPlotStyleOptions `json:"StyleOptions,omitempty"`
}

type BoxPlotSortConfiguration struct {
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
	PaginationConfiguration *PaginationConfiguration `json:"PaginationConfiguration,omitempty"`
}

type BoxPlotStyleOptions struct {
	FillStyle *string `json:"FillStyle,omitempty"`
}

type BoxPlotVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *BoxPlotChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type BrandColorPalette struct {
	Accent *Palette `json:"Accent,omitempty"`
	Danger *Palette `json:"Danger,omitempty"`
	Dimension *Palette `json:"Dimension,omitempty"`
	Info *Palette `json:"Info,omitempty"`
	Measure *Palette `json:"Measure,omitempty"`
	Primary *Palette `json:"Primary,omitempty"`
	Secondary *Palette `json:"Secondary,omitempty"`
	Success *Palette `json:"Success,omitempty"`
	Warning *Palette `json:"Warning,omitempty"`
}

type BrandDefinition struct {
	ApplicationTheme *ApplicationTheme `json:"ApplicationTheme,omitempty"`
	BrandName string `json:"BrandName,omitempty"`
	Description *string `json:"Description,omitempty"`
	LogoConfiguration *LogoConfiguration `json:"LogoConfiguration,omitempty"`
}

type BrandDetail struct {
	Arn *string `json:"Arn,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
	BrandStatus *string `json:"BrandStatus,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Errors []string `json:"Errors,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Logo *Logo `json:"Logo,omitempty"`
	VersionId *string `json:"VersionId,omitempty"`
	VersionStatus *string `json:"VersionStatus,omitempty"`
}

type BrandElementStyle struct {
	NavbarStyle *NavbarStyle `json:"NavbarStyle,omitempty"`
}

type BrandSummary struct {
	Arn *string `json:"Arn,omitempty"`
	BrandId *string `json:"BrandId,omitempty"`
	BrandName *string `json:"BrandName,omitempty"`
	BrandStatus *string `json:"BrandStatus,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
}

type CalculatedColumn struct {
	ColumnId string `json:"ColumnId,omitempty"`
	ColumnName string `json:"ColumnName,omitempty"`
	Expression string `json:"Expression,omitempty"`
}

type CalculatedField struct {
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
	Expression string `json:"Expression,omitempty"`
	Name string `json:"Name,omitempty"`
}

type CalculatedMeasureField struct {
	Expression string `json:"Expression,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
}

type CancelIngestionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	IngestionId string `json:"IngestionId,omitempty"`
}

type CancelIngestionResponse struct {
	Arn *string `json:"Arn,omitempty"`
	IngestionId *string `json:"IngestionId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type Capabilities struct {
	Action *string `json:"Action,omitempty"`
	AddOrRunAnomalyDetectionForAnalyses *string `json:"AddOrRunAnomalyDetectionForAnalyses,omitempty"`
	AmazonBedrockARSAction *string `json:"AmazonBedrockARSAction,omitempty"`
	AmazonBedrockFSAction *string `json:"AmazonBedrockFSAction,omitempty"`
	AmazonBedrockKRSAction *string `json:"AmazonBedrockKRSAction,omitempty"`
	AmazonSThreeAction *string `json:"AmazonSThreeAction,omitempty"`
	Analysis *string `json:"Analysis,omitempty"`
	ApproveFlowShareRequests *string `json:"ApproveFlowShareRequests,omitempty"`
	AsanaAction *string `json:"AsanaAction,omitempty"`
	Automate *string `json:"Automate,omitempty"`
	BambooHRAction *string `json:"BambooHRAction,omitempty"`
	BoxAgentAction *string `json:"BoxAgentAction,omitempty"`
	BuildCalculatedFieldWithQ *string `json:"BuildCalculatedFieldWithQ,omitempty"`
	CanvaAgentAction *string `json:"CanvaAgentAction,omitempty"`
	ChatAgent *string `json:"ChatAgent,omitempty"`
	ComprehendAction *string `json:"ComprehendAction,omitempty"`
	ComprehendMedicalAction *string `json:"ComprehendMedicalAction,omitempty"`
	ConfluenceAction *string `json:"ConfluenceAction,omitempty"`
	CreateAndUpdateAmazonBedrockARSAction *string `json:"CreateAndUpdateAmazonBedrockARSAction,omitempty"`
	CreateAndUpdateAmazonBedrockFSAction *string `json:"CreateAndUpdateAmazonBedrockFSAction,omitempty"`
	CreateAndUpdateAmazonBedrockKRSAction *string `json:"CreateAndUpdateAmazonBedrockKRSAction,omitempty"`
	CreateAndUpdateAmazonSThreeAction *string `json:"CreateAndUpdateAmazonSThreeAction,omitempty"`
	CreateAndUpdateAsanaAction *string `json:"CreateAndUpdateAsanaAction,omitempty"`
	CreateAndUpdateBambooHRAction *string `json:"CreateAndUpdateBambooHRAction,omitempty"`
	CreateAndUpdateBoxAgentAction *string `json:"CreateAndUpdateBoxAgentAction,omitempty"`
	CreateAndUpdateCanvaAgentAction *string `json:"CreateAndUpdateCanvaAgentAction,omitempty"`
	CreateAndUpdateComprehendAction *string `json:"CreateAndUpdateComprehendAction,omitempty"`
	CreateAndUpdateComprehendMedicalAction *string `json:"CreateAndUpdateComprehendMedicalAction,omitempty"`
	CreateAndUpdateConfluenceAction *string `json:"CreateAndUpdateConfluenceAction,omitempty"`
	CreateAndUpdateDashboardEmailReports *string `json:"CreateAndUpdateDashboardEmailReports,omitempty"`
	CreateAndUpdateDataSources *string `json:"CreateAndUpdateDataSources,omitempty"`
	CreateAndUpdateDatasets *string `json:"CreateAndUpdateDatasets,omitempty"`
	CreateAndUpdateFactSetAction *string `json:"CreateAndUpdateFactSetAction,omitempty"`
	CreateAndUpdateGenericHTTPAction *string `json:"CreateAndUpdateGenericHTTPAction,omitempty"`
	CreateAndUpdateGithubAction *string `json:"CreateAndUpdateGithubAction,omitempty"`
	CreateAndUpdateGoogleCalendarAction *string `json:"CreateAndUpdateGoogleCalendarAction,omitempty"`
	CreateAndUpdateHubspotAction *string `json:"CreateAndUpdateHubspotAction,omitempty"`
	CreateAndUpdateHuggingFaceAction *string `json:"CreateAndUpdateHuggingFaceAction,omitempty"`
	CreateAndUpdateIntercomAction *string `json:"CreateAndUpdateIntercomAction,omitempty"`
	CreateAndUpdateJiraAction *string `json:"CreateAndUpdateJiraAction,omitempty"`
	CreateAndUpdateLinearAction *string `json:"CreateAndUpdateLinearAction,omitempty"`
	CreateAndUpdateMCPAction *string `json:"CreateAndUpdateMCPAction,omitempty"`
	CreateAndUpdateMSExchangeAction *string `json:"CreateAndUpdateMSExchangeAction,omitempty"`
	CreateAndUpdateMSTeamsAction *string `json:"CreateAndUpdateMSTeamsAction,omitempty"`
	CreateAndUpdateMondayAction *string `json:"CreateAndUpdateMondayAction,omitempty"`
	CreateAndUpdateNewRelicAction *string `json:"CreateAndUpdateNewRelicAction,omitempty"`
	CreateAndUpdateNotionAction *string `json:"CreateAndUpdateNotionAction,omitempty"`
	CreateAndUpdateOneDriveAction *string `json:"CreateAndUpdateOneDriveAction,omitempty"`
	CreateAndUpdateOpenAPIAction *string `json:"CreateAndUpdateOpenAPIAction,omitempty"`
	CreateAndUpdatePagerDutyAction *string `json:"CreateAndUpdatePagerDutyAction,omitempty"`
	CreateAndUpdateSAPBillOfMaterialAction *string `json:"CreateAndUpdateSAPBillOfMaterialAction,omitempty"`
	CreateAndUpdateSAPBusinessPartnerAction *string `json:"CreateAndUpdateSAPBusinessPartnerAction,omitempty"`
	CreateAndUpdateSAPMaterialStockAction *string `json:"CreateAndUpdateSAPMaterialStockAction,omitempty"`
	CreateAndUpdateSAPPhysicalInventoryAction *string `json:"CreateAndUpdateSAPPhysicalInventoryAction,omitempty"`
	CreateAndUpdateSAPProductMasterDataAction *string `json:"CreateAndUpdateSAPProductMasterDataAction,omitempty"`
	CreateAndUpdateSalesforceAction *string `json:"CreateAndUpdateSalesforceAction,omitempty"`
	CreateAndUpdateSandPGMIAction *string `json:"CreateAndUpdateSandPGMIAction,omitempty"`
	CreateAndUpdateSandPGlobalEnergyAction *string `json:"CreateAndUpdateSandPGlobalEnergyAction,omitempty"`
	CreateAndUpdateServiceNowAction *string `json:"CreateAndUpdateServiceNowAction,omitempty"`
	CreateAndUpdateSharePointAction *string `json:"CreateAndUpdateSharePointAction,omitempty"`
	CreateAndUpdateSlackAction *string `json:"CreateAndUpdateSlackAction,omitempty"`
	CreateAndUpdateSmartsheetAction *string `json:"CreateAndUpdateSmartsheetAction,omitempty"`
	CreateAndUpdateTextractAction *string `json:"CreateAndUpdateTextractAction,omitempty"`
	CreateAndUpdateThemes *string `json:"CreateAndUpdateThemes,omitempty"`
	CreateAndUpdateThresholdAlerts *string `json:"CreateAndUpdateThresholdAlerts,omitempty"`
	CreateAndUpdateZendeskAction *string `json:"CreateAndUpdateZendeskAction,omitempty"`
	CreateChatAgents *string `json:"CreateChatAgents,omitempty"`
	CreateDashboardExecutiveSummaryWithQ *string `json:"CreateDashboardExecutiveSummaryWithQ,omitempty"`
	CreateSPICEDataset *string `json:"CreateSPICEDataset,omitempty"`
	CreateSharedFolders *string `json:"CreateSharedFolders,omitempty"`
	CreateSpaces *string `json:"CreateSpaces,omitempty"`
	Dashboard *string `json:"Dashboard,omitempty"`
	EditVisualWithQ *string `json:"EditVisualWithQ,omitempty"`
	ExportToCsv *string `json:"ExportToCsv,omitempty"`
	ExportToCsvInScheduledReports *string `json:"ExportToCsvInScheduledReports,omitempty"`
	ExportToExcel *string `json:"ExportToExcel,omitempty"`
	ExportToExcelInScheduledReports *string `json:"ExportToExcelInScheduledReports,omitempty"`
	ExportToPdf *string `json:"ExportToPdf,omitempty"`
	ExportToPdfInScheduledReports *string `json:"ExportToPdfInScheduledReports,omitempty"`
	Extension *string `json:"Extension,omitempty"`
	FactSetAction *string `json:"FactSetAction,omitempty"`
	Flow *string `json:"Flow,omitempty"`
	GenericHTTPAction *string `json:"GenericHTTPAction,omitempty"`
	GithubAction *string `json:"GithubAction,omitempty"`
	GoogleCalendarAction *string `json:"GoogleCalendarAction,omitempty"`
	HubspotAction *string `json:"HubspotAction,omitempty"`
	HuggingFaceAction *string `json:"HuggingFaceAction,omitempty"`
	IncludeContentInScheduledReportsEmail *string `json:"IncludeContentInScheduledReportsEmail,omitempty"`
	IntercomAction *string `json:"IntercomAction,omitempty"`
	JiraAction *string `json:"JiraAction,omitempty"`
	KnowledgeBase *string `json:"KnowledgeBase,omitempty"`
	LinearAction *string `json:"LinearAction,omitempty"`
	MCPAction *string `json:"MCPAction,omitempty"`
	MSExchangeAction *string `json:"MSExchangeAction,omitempty"`
	MSTeamsAction *string `json:"MSTeamsAction,omitempty"`
	ManageSharedFolders *string `json:"ManageSharedFolders,omitempty"`
	MondayAction *string `json:"MondayAction,omitempty"`
	NewRelicAction *string `json:"NewRelicAction,omitempty"`
	NotionAction *string `json:"NotionAction,omitempty"`
	OneDriveAction *string `json:"OneDriveAction,omitempty"`
	OpenAPIAction *string `json:"OpenAPIAction,omitempty"`
	PagerDutyAction *string `json:"PagerDutyAction,omitempty"`
	PerformFlowUiTask *string `json:"PerformFlowUiTask,omitempty"`
	PrintReports *string `json:"PrintReports,omitempty"`
	PublishWithoutApproval *string `json:"PublishWithoutApproval,omitempty"`
	RenameSharedFolders *string `json:"RenameSharedFolders,omitempty"`
	Research *string `json:"Research,omitempty"`
	SAPBillOfMaterialAction *string `json:"SAPBillOfMaterialAction,omitempty"`
	SAPBusinessPartnerAction *string `json:"SAPBusinessPartnerAction,omitempty"`
	SAPMaterialStockAction *string `json:"SAPMaterialStockAction,omitempty"`
	SAPPhysicalInventoryAction *string `json:"SAPPhysicalInventoryAction,omitempty"`
	SAPProductMasterDataAction *string `json:"SAPProductMasterDataAction,omitempty"`
	SalesforceAction *string `json:"SalesforceAction,omitempty"`
	SandPGMIAction *string `json:"SandPGMIAction,omitempty"`
	SandPGlobalEnergyAction *string `json:"SandPGlobalEnergyAction,omitempty"`
	SelfUpgradeUserRole *string `json:"SelfUpgradeUserRole,omitempty"`
	ServiceNowAction *string `json:"ServiceNowAction,omitempty"`
	ShareAmazonBedrockARSAction *string `json:"ShareAmazonBedrockARSAction,omitempty"`
	ShareAmazonBedrockFSAction *string `json:"ShareAmazonBedrockFSAction,omitempty"`
	ShareAmazonBedrockKRSAction *string `json:"ShareAmazonBedrockKRSAction,omitempty"`
	ShareAmazonSThreeAction *string `json:"ShareAmazonSThreeAction,omitempty"`
	ShareAnalyses *string `json:"ShareAnalyses,omitempty"`
	ShareAsanaAction *string `json:"ShareAsanaAction,omitempty"`
	ShareBambooHRAction *string `json:"ShareBambooHRAction,omitempty"`
	ShareBoxAgentAction *string `json:"ShareBoxAgentAction,omitempty"`
	ShareCanvaAgentAction *string `json:"ShareCanvaAgentAction,omitempty"`
	ShareChatAgents *string `json:"ShareChatAgents,omitempty"`
	ShareComprehendAction *string `json:"ShareComprehendAction,omitempty"`
	ShareComprehendMedicalAction *string `json:"ShareComprehendMedicalAction,omitempty"`
	ShareConfluenceAction *string `json:"ShareConfluenceAction,omitempty"`
	ShareDashboards *string `json:"ShareDashboards,omitempty"`
	ShareDataSources *string `json:"ShareDataSources,omitempty"`
	ShareDatasets *string `json:"ShareDatasets,omitempty"`
	ShareFactSetAction *string `json:"ShareFactSetAction,omitempty"`
	ShareGenericHTTPAction *string `json:"ShareGenericHTTPAction,omitempty"`
	ShareGithubAction *string `json:"ShareGithubAction,omitempty"`
	ShareGoogleCalendarAction *string `json:"ShareGoogleCalendarAction,omitempty"`
	ShareHubspotAction *string `json:"ShareHubspotAction,omitempty"`
	ShareHuggingFaceAction *string `json:"ShareHuggingFaceAction,omitempty"`
	ShareIntercomAction *string `json:"ShareIntercomAction,omitempty"`
	ShareJiraAction *string `json:"ShareJiraAction,omitempty"`
	ShareLinearAction *string `json:"ShareLinearAction,omitempty"`
	ShareMCPAction *string `json:"ShareMCPAction,omitempty"`
	ShareMSExchangeAction *string `json:"ShareMSExchangeAction,omitempty"`
	ShareMSTeamsAction *string `json:"ShareMSTeamsAction,omitempty"`
	ShareMondayAction *string `json:"ShareMondayAction,omitempty"`
	ShareNewRelicAction *string `json:"ShareNewRelicAction,omitempty"`
	ShareNotionAction *string `json:"ShareNotionAction,omitempty"`
	ShareOneDriveAction *string `json:"ShareOneDriveAction,omitempty"`
	ShareOpenAPIAction *string `json:"ShareOpenAPIAction,omitempty"`
	SharePagerDutyAction *string `json:"SharePagerDutyAction,omitempty"`
	SharePointAction *string `json:"SharePointAction,omitempty"`
	ShareSAPBillOfMaterialAction *string `json:"ShareSAPBillOfMaterialAction,omitempty"`
	ShareSAPBusinessPartnerAction *string `json:"ShareSAPBusinessPartnerAction,omitempty"`
	ShareSAPMaterialStockAction *string `json:"ShareSAPMaterialStockAction,omitempty"`
	ShareSAPPhysicalInventoryAction *string `json:"ShareSAPPhysicalInventoryAction,omitempty"`
	ShareSAPProductMasterDataAction *string `json:"ShareSAPProductMasterDataAction,omitempty"`
	ShareSalesforceAction *string `json:"ShareSalesforceAction,omitempty"`
	ShareSandPGMIAction *string `json:"ShareSandPGMIAction,omitempty"`
	ShareSandPGlobalEnergyAction *string `json:"ShareSandPGlobalEnergyAction,omitempty"`
	ShareServiceNowAction *string `json:"ShareServiceNowAction,omitempty"`
	ShareSharePointAction *string `json:"ShareSharePointAction,omitempty"`
	ShareSlackAction *string `json:"ShareSlackAction,omitempty"`
	ShareSmartsheetAction *string `json:"ShareSmartsheetAction,omitempty"`
	ShareSpaces *string `json:"ShareSpaces,omitempty"`
	ShareTextractAction *string `json:"ShareTextractAction,omitempty"`
	ShareZendeskAction *string `json:"ShareZendeskAction,omitempty"`
	SlackAction *string `json:"SlackAction,omitempty"`
	SmartsheetAction *string `json:"SmartsheetAction,omitempty"`
	Space *string `json:"Space,omitempty"`
	SubscribeDashboardEmailReports *string `json:"SubscribeDashboardEmailReports,omitempty"`
	TextractAction *string `json:"TextractAction,omitempty"`
	Topic *string `json:"Topic,omitempty"`
	UseAgentWebSearch *string `json:"UseAgentWebSearch,omitempty"`
	UseAmazonBedrockARSAction *string `json:"UseAmazonBedrockARSAction,omitempty"`
	UseAmazonBedrockFSAction *string `json:"UseAmazonBedrockFSAction,omitempty"`
	UseAmazonBedrockKRSAction *string `json:"UseAmazonBedrockKRSAction,omitempty"`
	UseAmazonSThreeAction *string `json:"UseAmazonSThreeAction,omitempty"`
	UseAsanaAction *string `json:"UseAsanaAction,omitempty"`
	UseBambooHRAction *string `json:"UseBambooHRAction,omitempty"`
	UseBedrockModels *string `json:"UseBedrockModels,omitempty"`
	UseBoxAgentAction *string `json:"UseBoxAgentAction,omitempty"`
	UseCanvaAgentAction *string `json:"UseCanvaAgentAction,omitempty"`
	UseComprehendAction *string `json:"UseComprehendAction,omitempty"`
	UseComprehendMedicalAction *string `json:"UseComprehendMedicalAction,omitempty"`
	UseConfluenceAction *string `json:"UseConfluenceAction,omitempty"`
	UseFactSetAction *string `json:"UseFactSetAction,omitempty"`
	UseGenericHTTPAction *string `json:"UseGenericHTTPAction,omitempty"`
	UseGithubAction *string `json:"UseGithubAction,omitempty"`
	UseGoogleCalendarAction *string `json:"UseGoogleCalendarAction,omitempty"`
	UseHubspotAction *string `json:"UseHubspotAction,omitempty"`
	UseHuggingFaceAction *string `json:"UseHuggingFaceAction,omitempty"`
	UseIntercomAction *string `json:"UseIntercomAction,omitempty"`
	UseJiraAction *string `json:"UseJiraAction,omitempty"`
	UseLinearAction *string `json:"UseLinearAction,omitempty"`
	UseMCPAction *string `json:"UseMCPAction,omitempty"`
	UseMSExchangeAction *string `json:"UseMSExchangeAction,omitempty"`
	UseMSTeamsAction *string `json:"UseMSTeamsAction,omitempty"`
	UseMondayAction *string `json:"UseMondayAction,omitempty"`
	UseNewRelicAction *string `json:"UseNewRelicAction,omitempty"`
	UseNotionAction *string `json:"UseNotionAction,omitempty"`
	UseOneDriveAction *string `json:"UseOneDriveAction,omitempty"`
	UseOpenAPIAction *string `json:"UseOpenAPIAction,omitempty"`
	UsePagerDutyAction *string `json:"UsePagerDutyAction,omitempty"`
	UseSAPBillOfMaterialAction *string `json:"UseSAPBillOfMaterialAction,omitempty"`
	UseSAPBusinessPartnerAction *string `json:"UseSAPBusinessPartnerAction,omitempty"`
	UseSAPMaterialStockAction *string `json:"UseSAPMaterialStockAction,omitempty"`
	UseSAPPhysicalInventoryAction *string `json:"UseSAPPhysicalInventoryAction,omitempty"`
	UseSAPProductMasterDataAction *string `json:"UseSAPProductMasterDataAction,omitempty"`
	UseSalesforceAction *string `json:"UseSalesforceAction,omitempty"`
	UseSandPGMIAction *string `json:"UseSandPGMIAction,omitempty"`
	UseSandPGlobalEnergyAction *string `json:"UseSandPGlobalEnergyAction,omitempty"`
	UseServiceNowAction *string `json:"UseServiceNowAction,omitempty"`
	UseSharePointAction *string `json:"UseSharePointAction,omitempty"`
	UseSlackAction *string `json:"UseSlackAction,omitempty"`
	UseSmartsheetAction *string `json:"UseSmartsheetAction,omitempty"`
	UseTextractAction *string `json:"UseTextractAction,omitempty"`
	UseZendeskAction *string `json:"UseZendeskAction,omitempty"`
	ViewAccountSPICECapacity *string `json:"ViewAccountSPICECapacity,omitempty"`
	ZendeskAction *string `json:"ZendeskAction,omitempty"`
}

type CascadingControlConfiguration struct {
	SourceControls []CascadingControlSource `json:"SourceControls,omitempty"`
}

type CascadingControlSource struct {
	ColumnToMatch *ColumnIdentifier `json:"ColumnToMatch,omitempty"`
	SourceSheetControlId *string `json:"SourceSheetControlId,omitempty"`
}

type CastColumnTypeOperation struct {
	ColumnName string `json:"ColumnName,omitempty"`
	Format *string `json:"Format,omitempty"`
	NewColumnType string `json:"NewColumnType,omitempty"`
	SubType *string `json:"SubType,omitempty"`
}

type CastColumnTypesOperation struct {
	Alias string `json:"Alias,omitempty"`
	CastColumnTypeOperations []CastColumnTypeOperation `json:"CastColumnTypeOperations,omitempty"`
	Source TransformOperationSource `json:"Source,omitempty"`
}

type CategoricalDimensionField struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *StringFormatConfiguration `json:"FormatConfiguration,omitempty"`
	HierarchyId *string `json:"HierarchyId,omitempty"`
}

type CategoricalMeasureField struct {
	AggregationFunction *string `json:"AggregationFunction,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *StringFormatConfiguration `json:"FormatConfiguration,omitempty"`
}

type CategoryDrillDownFilter struct {
	CategoryValues []string `json:"CategoryValues,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
}

type CategoryFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	Configuration CategoryFilterConfiguration `json:"Configuration,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
}

type CategoryFilterConfiguration struct {
	CustomFilterConfiguration *CustomFilterConfiguration `json:"CustomFilterConfiguration,omitempty"`
	CustomFilterListConfiguration *CustomFilterListConfiguration `json:"CustomFilterListConfiguration,omitempty"`
	FilterListConfiguration *FilterListConfiguration `json:"FilterListConfiguration,omitempty"`
}

type CategoryInnerFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	Configuration CategoryFilterConfiguration `json:"Configuration,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
}

type CellValueSynonym struct {
	CellValue *string `json:"CellValue,omitempty"`
	Synonyms []string `json:"Synonyms,omitempty"`
}

type ChartAxisLabelOptions struct {
	AxisLabelOptions []AxisLabelOptions `json:"AxisLabelOptions,omitempty"`
	SortIconVisibility *string `json:"SortIconVisibility,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type ClientCredentialsDetails struct {
	ClientCredentialsGrantDetails *ClientCredentialsGrantDetails `json:"ClientCredentialsGrantDetails,omitempty"`
}

type ClientCredentialsGrantDetails struct {
	ClientId string `json:"ClientId,omitempty"`
	ClientSecret string `json:"ClientSecret,omitempty"`
	TokenEndpoint string `json:"TokenEndpoint,omitempty"`
}

type ClientCredentialsGrantMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	ClientCredentialsDetails *ClientCredentialsDetails `json:"ClientCredentialsDetails,omitempty"`
	ClientCredentialsSource *string `json:"ClientCredentialsSource,omitempty"`
}

type ClusterMarker struct {
	SimpleClusterMarker *SimpleClusterMarker `json:"SimpleClusterMarker,omitempty"`
}

type ClusterMarkerConfiguration struct {
	ClusterMarker *ClusterMarker `json:"ClusterMarker,omitempty"`
}

type CollectiveConstant struct {
	ValueList []string `json:"ValueList,omitempty"`
}

type CollectiveConstantEntry struct {
	ConstantType *string `json:"ConstantType,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type ColorScale struct {
	ColorFillType string `json:"ColorFillType,omitempty"`
	Colors []DataColor `json:"Colors,omitempty"`
	NullValueColor *DataColor `json:"NullValueColor,omitempty"`
}

type ColorsConfiguration struct {
	CustomColors []CustomColor `json:"CustomColors,omitempty"`
}

type ColumnConfiguration struct {
	ColorsConfiguration *ColorsConfiguration `json:"ColorsConfiguration,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	DecalSettingsConfiguration *DecalSettingsConfiguration `json:"DecalSettingsConfiguration,omitempty"`
	FormatConfiguration *FormatConfiguration `json:"FormatConfiguration,omitempty"`
	Role *string `json:"Role,omitempty"`
}

type ColumnDescription struct {
	Text *string `json:"Text,omitempty"`
}

type ColumnGroup struct {
	GeoSpatialColumnGroup *GeoSpatialColumnGroup `json:"GeoSpatialColumnGroup,omitempty"`
}

type ColumnGroupColumnSchema struct {
	Name *string `json:"Name,omitempty"`
}

type ColumnGroupSchema struct {
	ColumnGroupColumnSchemaList []ColumnGroupColumnSchema `json:"ColumnGroupColumnSchemaList,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ColumnHierarchy struct {
	DateTimeHierarchy *DateTimeHierarchy `json:"DateTimeHierarchy,omitempty"`
	ExplicitHierarchy *ExplicitHierarchy `json:"ExplicitHierarchy,omitempty"`
	PredefinedHierarchy *PredefinedHierarchy `json:"PredefinedHierarchy,omitempty"`
}

type ColumnIdentifier struct {
	ColumnName string `json:"ColumnName,omitempty"`
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
}

type ColumnLevelPermissionRule struct {
	ColumnNames []string `json:"ColumnNames,omitempty"`
	Principals []string `json:"Principals,omitempty"`
}

type ColumnSchema struct {
	DataType *string `json:"DataType,omitempty"`
	GeographicRole *string `json:"GeographicRole,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type ColumnSort struct {
	AggregationFunction *AggregationFunction `json:"AggregationFunction,omitempty"`
	Direction string `json:"Direction,omitempty"`
	SortBy ColumnIdentifier `json:"SortBy,omitempty"`
}

type ColumnTag struct {
	ColumnDescription *ColumnDescription `json:"ColumnDescription,omitempty"`
	ColumnGeographicRole *string `json:"ColumnGeographicRole,omitempty"`
}

type ColumnToUnpivot struct {
	ColumnName *string `json:"ColumnName,omitempty"`
	NewValue *string `json:"NewValue,omitempty"`
}

type ColumnTooltipItem struct {
	Aggregation *AggregationFunction `json:"Aggregation,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	Label *string `json:"Label,omitempty"`
	TooltipTarget *string `json:"TooltipTarget,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type ComboChartAggregatedFieldWells struct {
	BarValues []MeasureField `json:"BarValues,omitempty"`
	Category []DimensionField `json:"Category,omitempty"`
	Colors []DimensionField `json:"Colors,omitempty"`
	LineValues []MeasureField `json:"LineValues,omitempty"`
}

type ComboChartConfiguration struct {
	BarDataLabels *DataLabelOptions `json:"BarDataLabels,omitempty"`
	BarsArrangement *string `json:"BarsArrangement,omitempty"`
	CategoryAxis *AxisDisplayOptions `json:"CategoryAxis,omitempty"`
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	ColorLabelOptions *ChartAxisLabelOptions `json:"ColorLabelOptions,omitempty"`
	DefaultSeriesSettings *ComboChartDefaultSeriesSettings `json:"DefaultSeriesSettings,omitempty"`
	FieldWells *ComboChartFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	LineDataLabels *DataLabelOptions `json:"LineDataLabels,omitempty"`
	PrimaryYAxisDisplayOptions *AxisDisplayOptions `json:"PrimaryYAxisDisplayOptions,omitempty"`
	PrimaryYAxisLabelOptions *ChartAxisLabelOptions `json:"PrimaryYAxisLabelOptions,omitempty"`
	ReferenceLines []ReferenceLine `json:"ReferenceLines,omitempty"`
	SecondaryYAxisDisplayOptions *AxisDisplayOptions `json:"SecondaryYAxisDisplayOptions,omitempty"`
	SecondaryYAxisLabelOptions *ChartAxisLabelOptions `json:"SecondaryYAxisLabelOptions,omitempty"`
	Series []ComboSeriesItem `json:"Series,omitempty"`
	SingleAxisOptions *SingleAxisOptions `json:"SingleAxisOptions,omitempty"`
	SortConfiguration *ComboChartSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type ComboChartDefaultSeriesSettings struct {
	BorderSettings *BorderSettings `json:"BorderSettings,omitempty"`
	DecalSettings *DecalSettings `json:"DecalSettings,omitempty"`
	LineStyleSettings *LineChartLineStyleSettings `json:"LineStyleSettings,omitempty"`
	MarkerStyleSettings *LineChartMarkerStyleSettings `json:"MarkerStyleSettings,omitempty"`
}

type ComboChartFieldWells struct {
	ComboChartAggregatedFieldWells *ComboChartAggregatedFieldWells `json:"ComboChartAggregatedFieldWells,omitempty"`
}

type ComboChartSeriesSettings struct {
	BorderSettings *BorderSettings `json:"BorderSettings,omitempty"`
	DecalSettings *DecalSettings `json:"DecalSettings,omitempty"`
	LineStyleSettings *LineChartLineStyleSettings `json:"LineStyleSettings,omitempty"`
	MarkerStyleSettings *LineChartMarkerStyleSettings `json:"MarkerStyleSettings,omitempty"`
}

type ComboChartSortConfiguration struct {
	CategoryItemsLimit *ItemsLimitConfiguration `json:"CategoryItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
	ColorItemsLimit *ItemsLimitConfiguration `json:"ColorItemsLimit,omitempty"`
	ColorSort []FieldSortOptions `json:"ColorSort,omitempty"`
}

type ComboChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *ComboChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type ComboSeriesItem struct {
	DataFieldComboSeriesItem *DataFieldComboSeriesItem `json:"DataFieldComboSeriesItem,omitempty"`
	FieldComboSeriesItem *FieldComboSeriesItem `json:"FieldComboSeriesItem,omitempty"`
}

type ComparativeOrder struct {
	SpecifedOrder []string `json:"SpecifedOrder,omitempty"`
	TreatUndefinedSpecifiedValues *string `json:"TreatUndefinedSpecifiedValues,omitempty"`
	UseOrdering *string `json:"UseOrdering,omitempty"`
}

type ComparisonConfiguration struct {
	ComparisonFormat *ComparisonFormatConfiguration `json:"ComparisonFormat,omitempty"`
	ComparisonMethod *string `json:"ComparisonMethod,omitempty"`
}

type ComparisonFormatConfiguration struct {
	NumberDisplayFormatConfiguration *NumberDisplayFormatConfiguration `json:"NumberDisplayFormatConfiguration,omitempty"`
	PercentageDisplayFormatConfiguration *PercentageDisplayFormatConfiguration `json:"PercentageDisplayFormatConfiguration,omitempty"`
}

type Computation struct {
	Forecast *ForecastComputation `json:"Forecast,omitempty"`
	GrowthRate *GrowthRateComputation `json:"GrowthRate,omitempty"`
	MaximumMinimum *MaximumMinimumComputation `json:"MaximumMinimum,omitempty"`
	MetricComparison *MetricComparisonComputation `json:"MetricComparison,omitempty"`
	PeriodOverPeriod *PeriodOverPeriodComputation `json:"PeriodOverPeriod,omitempty"`
	PeriodToDate *PeriodToDateComputation `json:"PeriodToDate,omitempty"`
	TopBottomMovers *TopBottomMoversComputation `json:"TopBottomMovers,omitempty"`
	TopBottomRanked *TopBottomRankedComputation `json:"TopBottomRanked,omitempty"`
	TotalAggregation *TotalAggregationComputation `json:"TotalAggregation,omitempty"`
	UniqueValues *UniqueValuesComputation `json:"UniqueValues,omitempty"`
}

type ConditionalFormattingColor struct {
	Gradient *ConditionalFormattingGradientColor `json:"Gradient,omitempty"`
	Solid *ConditionalFormattingSolidColor `json:"Solid,omitempty"`
}

type ConditionalFormattingCustomIconCondition struct {
	Color *string `json:"Color,omitempty"`
	DisplayConfiguration *ConditionalFormattingIconDisplayConfiguration `json:"DisplayConfiguration,omitempty"`
	Expression string `json:"Expression,omitempty"`
	IconOptions ConditionalFormattingCustomIconOptions `json:"IconOptions,omitempty"`
}

type ConditionalFormattingCustomIconOptions struct {
	Icon *string `json:"Icon,omitempty"`
	UnicodeIcon *string `json:"UnicodeIcon,omitempty"`
}

type ConditionalFormattingGradientColor struct {
	Color GradientColor `json:"Color,omitempty"`
	Expression string `json:"Expression,omitempty"`
}

type ConditionalFormattingIcon struct {
	CustomCondition *ConditionalFormattingCustomIconCondition `json:"CustomCondition,omitempty"`
	IconSet *ConditionalFormattingIconSet `json:"IconSet,omitempty"`
}

type ConditionalFormattingIconDisplayConfiguration struct {
	IconDisplayOption *string `json:"IconDisplayOption,omitempty"`
}

type ConditionalFormattingIconSet struct {
	Expression string `json:"Expression,omitempty"`
	IconSetType *string `json:"IconSetType,omitempty"`
}

type ConditionalFormattingSolidColor struct {
	Color *string `json:"Color,omitempty"`
	Expression string `json:"Expression,omitempty"`
}

type ConfluenceParameters struct {
	ConfluenceUrl string `json:"ConfluenceUrl,omitempty"`
}

type ContextMenuOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type ContextualAccentPalette struct {
	Automation *Palette `json:"Automation,omitempty"`
	Connection *Palette `json:"Connection,omitempty"`
	Insight *Palette `json:"Insight,omitempty"`
	Visualization *Palette `json:"Visualization,omitempty"`
}

type ContributionAnalysisDefault struct {
	ContributorDimensions []ColumnIdentifier `json:"ContributorDimensions,omitempty"`
	MeasureFieldId string `json:"MeasureFieldId,omitempty"`
}

type ContributionAnalysisFactor struct {
	FieldName *string `json:"FieldName,omitempty"`
}

type ContributionAnalysisTimeRanges struct {
	EndRange *TopicIRFilterOption `json:"EndRange,omitempty"`
	StartRange *TopicIRFilterOption `json:"StartRange,omitempty"`
}

type Coordinate struct {
	Latitude float64 `json:"Latitude,omitempty"`
	Longitude float64 `json:"Longitude,omitempty"`
}

type CreateAccountCustomizationRequest struct {
	AccountCustomization AccountCustomization `json:"AccountCustomization,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateAccountCustomizationResponse struct {
	AccountCustomization *AccountCustomization `json:"AccountCustomization,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"Namespace,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateAccountSubscriptionRequest struct {
	AccountName string `json:"AccountName,omitempty"`
	ActiveDirectoryName *string `json:"ActiveDirectoryName,omitempty"`
	AdminGroup []string `json:"AdminGroup,omitempty"`
	AdminProGroup []string `json:"AdminProGroup,omitempty"`
	AuthenticationMethod string `json:"AuthenticationMethod,omitempty"`
	AuthorGroup []string `json:"AuthorGroup,omitempty"`
	AuthorProGroup []string `json:"AuthorProGroup,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ContactNumber *string `json:"ContactNumber,omitempty"`
	DirectoryId *string `json:"DirectoryId,omitempty"`
	Edition *string `json:"Edition,omitempty"`
	EmailAddress *string `json:"EmailAddress,omitempty"`
	FirstName *string `json:"FirstName,omitempty"`
	IAMIdentityCenterInstanceArn *string `json:"IAMIdentityCenterInstanceArn,omitempty"`
	LastName *string `json:"LastName,omitempty"`
	NotificationEmail string `json:"NotificationEmail,omitempty"`
	ReaderGroup []string `json:"ReaderGroup,omitempty"`
	ReaderProGroup []string `json:"ReaderProGroup,omitempty"`
	Realm *string `json:"Realm,omitempty"`
}

type CreateAccountSubscriptionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	SignupResponse *SignupResponse `json:"SignupResponse,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateActionConnectorRequest struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	AuthenticationConfig AuthConfig `json:"AuthenticationConfig,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name string `json:"Name,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	Type string `json:"Type,omitempty"`
	VpcConnectionArn *string `json:"VpcConnectionArn,omitempty"`
}

type CreateActionConnectorResponse struct {
	ActionConnectorId *string `json:"ActionConnectorId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateAnalysisRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Definition *AnalysisDefinition `json:"Definition,omitempty"`
	FolderArns []string `json:"FolderArns,omitempty"`
	Name string `json:"Name,omitempty"`
	Parameters *Parameters `json:"Parameters,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	SourceEntity *AnalysisSourceEntity `json:"SourceEntity,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	ValidationStrategy *ValidationStrategy `json:"ValidationStrategy,omitempty"`
}

type CreateAnalysisResponse struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateBrandRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandDefinition *BrandDefinition `json:"BrandDefinition,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateBrandResponse struct {
	BrandDefinition *BrandDefinition `json:"BrandDefinition,omitempty"`
	BrandDetail *BrandDetail `json:"BrandDetail,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
}

type CreateColumnsOperation struct {
	Alias *string `json:"Alias,omitempty"`
	Columns []CalculatedColumn `json:"Columns,omitempty"`
	Source *TransformOperationSource `json:"Source,omitempty"`
}

type CreateCustomPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Capabilities *Capabilities `json:"Capabilities,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateCustomPermissionsResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateDashboardRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	DashboardPublishOptions *DashboardPublishOptions `json:"DashboardPublishOptions,omitempty"`
	Definition *DashboardVersionDefinition `json:"Definition,omitempty"`
	FolderArns []string `json:"FolderArns,omitempty"`
	LinkEntities []string `json:"LinkEntities,omitempty"`
	LinkSharingConfiguration *LinkSharingConfiguration `json:"LinkSharingConfiguration,omitempty"`
	Name string `json:"Name,omitempty"`
	Parameters *Parameters `json:"Parameters,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	SourceEntity *DashboardSourceEntity `json:"SourceEntity,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	ValidationStrategy *ValidationStrategy `json:"ValidationStrategy,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type CreateDashboardResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VersionArn *string `json:"VersionArn,omitempty"`
}

type CreateDataSetRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ColumnGroups []ColumnGroup `json:"ColumnGroups,omitempty"`
	ColumnLevelPermissionRules []ColumnLevelPermissionRule `json:"ColumnLevelPermissionRules,omitempty"`
	DataPrepConfiguration *DataPrepConfiguration `json:"DataPrepConfiguration,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	DataSetUsageConfiguration *DataSetUsageConfiguration `json:"DataSetUsageConfiguration,omitempty"`
	DatasetParameters []DatasetParameter `json:"DatasetParameters,omitempty"`
	FieldFolders map[string]FieldFolder `json:"FieldFolders,omitempty"`
	FolderArns []string `json:"FolderArns,omitempty"`
	ImportMode string `json:"ImportMode,omitempty"`
	LogicalTableMap map[string]LogicalTable `json:"LogicalTableMap,omitempty"`
	Name string `json:"Name,omitempty"`
	PerformanceConfiguration *PerformanceConfiguration `json:"PerformanceConfiguration,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	PhysicalTableMap map[string]PhysicalTable `json:"PhysicalTableMap,omitempty"`
	RowLevelPermissionDataSet *RowLevelPermissionDataSet `json:"RowLevelPermissionDataSet,omitempty"`
	RowLevelPermissionTagConfiguration *RowLevelPermissionTagConfiguration `json:"RowLevelPermissionTagConfiguration,omitempty"`
	SemanticModelConfiguration *SemanticModelConfiguration `json:"SemanticModelConfiguration,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	UseAs *string `json:"UseAs,omitempty"`
}

type CreateDataSetResponse struct {
	Arn *string `json:"Arn,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	IngestionArn *string `json:"IngestionArn,omitempty"`
	IngestionId *string `json:"IngestionId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateDataSourceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Credentials *DataSourceCredentials `json:"Credentials,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
	DataSourceParameters *DataSourceParameters `json:"DataSourceParameters,omitempty"`
	FolderArns []string `json:"FolderArns,omitempty"`
	Name string `json:"Name,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	SslProperties *SslProperties `json:"SslProperties,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	Type string `json:"Type,omitempty"`
	VpcConnectionProperties *VpcConnectionProperties `json:"VpcConnectionProperties,omitempty"`
}

type CreateDataSourceResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateFolderMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	MemberId string `json:"MemberId,omitempty"`
	MemberType string `json:"MemberType,omitempty"`
}

type CreateFolderMembershipResponse struct {
	FolderMember *FolderMember `json:"FolderMember,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateFolderRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	FolderType *string `json:"FolderType,omitempty"`
	Name *string `json:"Name,omitempty"`
	ParentFolderArn *string `json:"ParentFolderArn,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	SharingModel *string `json:"SharingModel,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateFolderResponse struct {
	Arn *string `json:"Arn,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateGroupMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	MemberName string `json:"MemberName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type CreateGroupMembershipResponse struct {
	GroupMember *GroupMember `json:"GroupMember,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateGroupRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Description *string `json:"Description,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type CreateGroupResponse struct {
	Group *Group `json:"Group,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateIAMPolicyAssignmentRequest struct {
	AssignmentName string `json:"AssignmentName,omitempty"`
	AssignmentStatus string `json:"AssignmentStatus,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Identities map[string][]string `json:"Identities,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	PolicyArn *string `json:"PolicyArn,omitempty"`
}

type CreateIAMPolicyAssignmentResponse struct {
	AssignmentId *string `json:"AssignmentId,omitempty"`
	AssignmentName *string `json:"AssignmentName,omitempty"`
	AssignmentStatus *string `json:"AssignmentStatus,omitempty"`
	Identities map[string][]string `json:"Identities,omitempty"`
	PolicyArn *string `json:"PolicyArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateIngestionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	IngestionId string `json:"IngestionId,omitempty"`
	IngestionType *string `json:"IngestionType,omitempty"`
}

type CreateIngestionResponse struct {
	Arn *string `json:"Arn,omitempty"`
	IngestionId *string `json:"IngestionId,omitempty"`
	IngestionStatus *string `json:"IngestionStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateNamespaceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	IdentityStore string `json:"IdentityStore,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateNamespaceResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CapacityRegion *string `json:"CapacityRegion,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	IdentityStore *string `json:"IdentityStore,omitempty"`
	Name *string `json:"Name,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	Schedule RefreshSchedule `json:"Schedule,omitempty"`
}

type CreateRefreshScheduleResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ScheduleId *string `json:"ScheduleId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateRoleMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MemberName string `json:"MemberName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Role string `json:"Role,omitempty"`
}

type CreateRoleMembershipResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type CreateTemplateAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	TemplateVersionNumber int64 `json:"TemplateVersionNumber,omitempty"`
}

type CreateTemplateAliasResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateAlias *TemplateAlias `json:"TemplateAlias,omitempty"`
}

type CreateTemplateRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Definition *TemplateVersionDefinition `json:"Definition,omitempty"`
	Name *string `json:"Name,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	SourceEntity *TemplateSourceEntity `json:"SourceEntity,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	ValidationStrategy *ValidationStrategy `json:"ValidationStrategy,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type CreateTemplateResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
	VersionArn *string `json:"VersionArn,omitempty"`
}

type CreateThemeAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
	ThemeVersionNumber int64 `json:"ThemeVersionNumber,omitempty"`
}

type CreateThemeAliasResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeAlias *ThemeAlias `json:"ThemeAlias,omitempty"`
}

type CreateThemeRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BaseThemeId string `json:"BaseThemeId,omitempty"`
	Configuration ThemeConfiguration `json:"Configuration,omitempty"`
	Name string `json:"Name,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type CreateThemeResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
	VersionArn *string `json:"VersionArn,omitempty"`
}

type CreateTopicRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DatasetArn string `json:"DatasetArn,omitempty"`
	DatasetName *string `json:"DatasetName,omitempty"`
	RefreshSchedule TopicRefreshSchedule `json:"RefreshSchedule,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type CreateTopicRefreshScheduleResponse struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type CreateTopicRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomInstructions *CustomInstructions `json:"CustomInstructions,omitempty"`
	FolderArns []string `json:"FolderArns,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	Topic TopicDetails `json:"Topic,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type CreateTopicResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RefreshArn *string `json:"RefreshArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type CreateTopicReviewedAnswer struct {
	AnswerId string `json:"AnswerId,omitempty"`
	DatasetArn string `json:"DatasetArn,omitempty"`
	Mir *TopicIR `json:"Mir,omitempty"`
	PrimaryVisual *TopicVisual `json:"PrimaryVisual,omitempty"`
	Question string `json:"Question,omitempty"`
	Template *TopicTemplate `json:"Template,omitempty"`
}

type CreateVPCConnectionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DnsResolvers []string `json:"DnsResolvers,omitempty"`
	Name string `json:"Name,omitempty"`
	RoleArn string `json:"RoleArn,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	VPCConnectionId string `json:"VPCConnectionId,omitempty"`
}

type CreateVPCConnectionResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VPCConnectionId *string `json:"VPCConnectionId,omitempty"`
}

type CredentialPair struct {
	AlternateDataSourceParameters []DataSourceParameters `json:"AlternateDataSourceParameters,omitempty"`
	Password string `json:"Password,omitempty"`
	Username string `json:"Username,omitempty"`
}

type CurrencyDisplayFormatConfiguration struct {
	DecimalPlacesConfiguration *DecimalPlacesConfiguration `json:"DecimalPlacesConfiguration,omitempty"`
	NegativeValueConfiguration *NegativeValueConfiguration `json:"NegativeValueConfiguration,omitempty"`
	NullValueFormatConfiguration *NullValueFormatConfiguration `json:"NullValueFormatConfiguration,omitempty"`
	NumberScale *string `json:"NumberScale,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
	SeparatorConfiguration *NumericSeparatorConfiguration `json:"SeparatorConfiguration,omitempty"`
	Suffix *string `json:"Suffix,omitempty"`
	Symbol *string `json:"Symbol,omitempty"`
}

type CustomActionFilterOperation struct {
	SelectedFieldsConfiguration FilterOperationSelectedFieldsConfiguration `json:"SelectedFieldsConfiguration,omitempty"`
	TargetVisualsConfiguration FilterOperationTargetVisualsConfiguration `json:"TargetVisualsConfiguration,omitempty"`
}

type CustomActionNavigationOperation struct {
	LocalNavigationConfiguration *LocalNavigationConfiguration `json:"LocalNavigationConfiguration,omitempty"`
}

type CustomActionSetParametersOperation struct {
	ParameterValueConfigurations []SetParameterValueConfiguration `json:"ParameterValueConfigurations,omitempty"`
}

type CustomActionURLOperation struct {
	URLTarget string `json:"URLTarget,omitempty"`
	URLTemplate string `json:"URLTemplate,omitempty"`
}

type CustomColor struct {
	Color string `json:"Color,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
	SpecialValue *string `json:"SpecialValue,omitempty"`
}

type CustomConnectionParameters struct {
	ConnectionType *string `json:"ConnectionType,omitempty"`
}

type CustomContentConfiguration struct {
	ContentType *string `json:"ContentType,omitempty"`
	ContentUrl *string `json:"ContentUrl,omitempty"`
	ImageScaling *string `json:"ImageScaling,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
}

type CustomContentVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *CustomContentConfiguration `json:"ChartConfiguration,omitempty"`
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type CustomFilterConfiguration struct {
	CategoryValue *string `json:"CategoryValue,omitempty"`
	MatchOperator string `json:"MatchOperator,omitempty"`
	NullOption string `json:"NullOption,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
	SelectAllOptions *string `json:"SelectAllOptions,omitempty"`
}

type CustomFilterListConfiguration struct {
	CategoryValues []string `json:"CategoryValues,omitempty"`
	MatchOperator string `json:"MatchOperator,omitempty"`
	NullOption string `json:"NullOption,omitempty"`
	SelectAllOptions *string `json:"SelectAllOptions,omitempty"`
}

type CustomInstructions struct {
	CustomInstructionsString string `json:"CustomInstructionsString,omitempty"`
}

type CustomNarrativeOptions struct {
	Narrative string `json:"Narrative,omitempty"`
}

type CustomParameterValues struct {
	DateTimeValues []time.Time `json:"DateTimeValues,omitempty"`
	DecimalValues []float64 `json:"DecimalValues,omitempty"`
	IntegerValues []int64 `json:"IntegerValues,omitempty"`
	StringValues []string `json:"StringValues,omitempty"`
}

type CustomPermissions struct {
	Arn *string `json:"Arn,omitempty"`
	Capabilities *Capabilities `json:"Capabilities,omitempty"`
	CustomPermissionsName *string `json:"CustomPermissionsName,omitempty"`
}

type CustomSql struct {
	Columns []InputColumn `json:"Columns,omitempty"`
	DataSourceArn string `json:"DataSourceArn,omitempty"`
	Name string `json:"Name,omitempty"`
	SqlQuery string `json:"SqlQuery,omitempty"`
}

type CustomValuesConfiguration struct {
	CustomValues CustomParameterValues `json:"CustomValues,omitempty"`
	IncludeNullValue bool `json:"IncludeNullValue,omitempty"`
}

type Dashboard struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	LastPublishedTime *time.Time `json:"LastPublishedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	LinkEntities []string `json:"LinkEntities,omitempty"`
	Name *string `json:"Name,omitempty"`
	Version *DashboardVersion `json:"Version,omitempty"`
}

type DashboardCustomizationVisualOptions struct {
	FieldsConfiguration *VisualCustomizationFieldsConfiguration `json:"FieldsConfiguration,omitempty"`
}

type DashboardError struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
	ViolatedEntities []Entity `json:"ViolatedEntities,omitempty"`
}

type DashboardPublishOptions struct {
	AdHocFilteringOption *AdHocFilteringOption `json:"AdHocFilteringOption,omitempty"`
	DataPointDrillUpDownOption *DataPointDrillUpDownOption `json:"DataPointDrillUpDownOption,omitempty"`
	DataPointMenuLabelOption *DataPointMenuLabelOption `json:"DataPointMenuLabelOption,omitempty"`
	DataPointTooltipOption *DataPointTooltipOption `json:"DataPointTooltipOption,omitempty"`
	DataQAEnabledOption *DataQAEnabledOption `json:"DataQAEnabledOption,omitempty"`
	DataStoriesSharingOption *DataStoriesSharingOption `json:"DataStoriesSharingOption,omitempty"`
	ExecutiveSummaryOption *ExecutiveSummaryOption `json:"ExecutiveSummaryOption,omitempty"`
	ExportToCSVOption *ExportToCSVOption `json:"ExportToCSVOption,omitempty"`
	ExportWithHiddenFieldsOption *ExportWithHiddenFieldsOption `json:"ExportWithHiddenFieldsOption,omitempty"`
	QuickSuiteActionsOption *QuickSuiteActionsOption `json:"QuickSuiteActionsOption,omitempty"`
	SheetControlsOption *SheetControlsOption `json:"SheetControlsOption,omitempty"`
	SheetLayoutElementMaximizationOption *SheetLayoutElementMaximizationOption `json:"SheetLayoutElementMaximizationOption,omitempty"`
	VisualAxisSortOption *VisualAxisSortOption `json:"VisualAxisSortOption,omitempty"`
	VisualMenuOption *VisualMenuOption `json:"VisualMenuOption,omitempty"`
	VisualPublishOptions *DashboardVisualPublishOptions `json:"VisualPublishOptions,omitempty"`
}

type DashboardSearchFilter struct {
	Name *string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type DashboardSourceEntity struct {
	SourceTemplate *DashboardSourceTemplate `json:"SourceTemplate,omitempty"`
}

type DashboardSourceTemplate struct {
	Arn string `json:"Arn,omitempty"`
	DataSetReferences []DataSetReference `json:"DataSetReferences,omitempty"`
}

type DashboardSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	LastPublishedTime *time.Time `json:"LastPublishedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	PublishedVersionNumber int64 `json:"PublishedVersionNumber,omitempty"`
}

type DashboardVersion struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataSetArns []string `json:"DataSetArns,omitempty"`
	Description *string `json:"Description,omitempty"`
	Errors []DashboardError `json:"Errors,omitempty"`
	Sheets []Sheet `json:"Sheets,omitempty"`
	SourceEntityArn *string `json:"SourceEntityArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type DashboardVersionDefinition struct {
	AnalysisDefaults *AnalysisDefaults `json:"AnalysisDefaults,omitempty"`
	CalculatedFields []CalculatedField `json:"CalculatedFields,omitempty"`
	ColumnConfigurations []ColumnConfiguration `json:"ColumnConfigurations,omitempty"`
	DataSetIdentifierDeclarations []DataSetIdentifierDeclaration `json:"DataSetIdentifierDeclarations,omitempty"`
	FilterGroups []FilterGroup `json:"FilterGroups,omitempty"`
	Options *AssetOptions `json:"Options,omitempty"`
	ParameterDeclarations []ParameterDeclaration `json:"ParameterDeclarations,omitempty"`
	Sheets []SheetDefinition `json:"Sheets,omitempty"`
	StaticFiles []StaticFile `json:"StaticFiles,omitempty"`
	TooltipSheets []TooltipSheetDefinition `json:"TooltipSheets,omitempty"`
}

type DashboardVersionSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	SourceEntityArn *string `json:"SourceEntityArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type DashboardVisualId struct {
	DashboardId string `json:"DashboardId,omitempty"`
	SheetId string `json:"SheetId,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type DashboardVisualPublishOptions struct {
	ExportHiddenFieldsOption *ExportHiddenFieldsOption `json:"ExportHiddenFieldsOption,omitempty"`
}

type DashboardVisualResult struct {
	DashboardId *string `json:"DashboardId,omitempty"`
	DashboardName *string `json:"DashboardName,omitempty"`
	DashboardUrl *string `json:"DashboardUrl,omitempty"`
	SheetId *string `json:"SheetId,omitempty"`
	SheetName *string `json:"SheetName,omitempty"`
	VisualId *string `json:"VisualId,omitempty"`
	VisualSubtitle *string `json:"VisualSubtitle,omitempty"`
	VisualTitle *string `json:"VisualTitle,omitempty"`
}

type DataAggregation struct {
	DatasetRowDateGranularity *string `json:"DatasetRowDateGranularity,omitempty"`
	DefaultDateColumnName *string `json:"DefaultDateColumnName,omitempty"`
}

type DataBarsOptions struct {
	FieldId string `json:"FieldId,omitempty"`
	NegativeColor *string `json:"NegativeColor,omitempty"`
	PositiveColor *string `json:"PositiveColor,omitempty"`
}

type DataColor struct {
	Color *string `json:"Color,omitempty"`
	DataValue float64 `json:"DataValue,omitempty"`
}

type DataColorPalette struct {
	Colors []string `json:"Colors,omitempty"`
	EmptyFillColor *string `json:"EmptyFillColor,omitempty"`
	MinMaxGradient []string `json:"MinMaxGradient,omitempty"`
}

type DataFieldBarSeriesItem struct {
	FieldId string `json:"FieldId,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
	Settings *BarChartSeriesSettings `json:"Settings,omitempty"`
}

type DataFieldComboSeriesItem struct {
	FieldId string `json:"FieldId,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
	Settings *ComboChartSeriesSettings `json:"Settings,omitempty"`
}

type DataFieldSeriesItem struct {
	AxisBinding string `json:"AxisBinding,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
	Settings *LineChartSeriesSettings `json:"Settings,omitempty"`
}

type DataLabelOptions struct {
	CategoryLabelVisibility *string `json:"CategoryLabelVisibility,omitempty"`
	DataLabelTypes []DataLabelType `json:"DataLabelTypes,omitempty"`
	LabelColor *string `json:"LabelColor,omitempty"`
	LabelContent *string `json:"LabelContent,omitempty"`
	LabelFontConfiguration *FontConfiguration `json:"LabelFontConfiguration,omitempty"`
	MeasureLabelVisibility *string `json:"MeasureLabelVisibility,omitempty"`
	Overlap *string `json:"Overlap,omitempty"`
	Position *string `json:"Position,omitempty"`
	TotalsVisibility *string `json:"TotalsVisibility,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type DataLabelType struct {
	DataPathLabelType *DataPathLabelType `json:"DataPathLabelType,omitempty"`
	FieldLabelType *FieldLabelType `json:"FieldLabelType,omitempty"`
	MaximumLabelType *MaximumLabelType `json:"MaximumLabelType,omitempty"`
	MinimumLabelType *MinimumLabelType `json:"MinimumLabelType,omitempty"`
	RangeEndsLabelType *RangeEndsLabelType `json:"RangeEndsLabelType,omitempty"`
}

type DataPathColor struct {
	Color string `json:"Color,omitempty"`
	Element DataPathValue `json:"Element,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type DataPathLabelType struct {
	FieldId *string `json:"FieldId,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type DataPathSort struct {
	Direction string `json:"Direction,omitempty"`
	SortPaths []DataPathValue `json:"SortPaths,omitempty"`
}

type DataPathType struct {
	PivotTableDataPathType *string `json:"PivotTableDataPathType,omitempty"`
}

type DataPathValue struct {
	DataPathType *DataPathType `json:"DataPathType,omitempty"`
	FieldId *string `json:"FieldId,omitempty"`
	FieldValue *string `json:"FieldValue,omitempty"`
}

type DataPointDrillUpDownOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type DataPointMenuLabelOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type DataPointTooltipOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type DataPrepAggregationFunction struct {
	ListAggregation *DataPrepListAggregationFunction `json:"ListAggregation,omitempty"`
	SimpleAggregation *DataPrepSimpleAggregationFunction `json:"SimpleAggregation,omitempty"`
}

type DataPrepConfiguration struct {
	DestinationTableMap map[string]DestinationTable `json:"DestinationTableMap,omitempty"`
	SourceTableMap map[string]SourceTable `json:"SourceTableMap,omitempty"`
	TransformStepMap map[string]TransformStep `json:"TransformStepMap,omitempty"`
}

type DataPrepListAggregationFunction struct {
	Distinct bool `json:"Distinct,omitempty"`
	InputColumnName *string `json:"InputColumnName,omitempty"`
	Separator string `json:"Separator,omitempty"`
}

type DataPrepSimpleAggregationFunction struct {
	FunctionType string `json:"FunctionType,omitempty"`
	InputColumnName *string `json:"InputColumnName,omitempty"`
}

type DataQAEnabledOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type DataQnAConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type DataSet struct {
	Arn *string `json:"Arn,omitempty"`
	ColumnGroups []ColumnGroup `json:"ColumnGroups,omitempty"`
	ColumnLevelPermissionRules []ColumnLevelPermissionRule `json:"ColumnLevelPermissionRules,omitempty"`
	ConsumedSpiceCapacityInBytes int64 `json:"ConsumedSpiceCapacityInBytes,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataPrepConfiguration *DataPrepConfiguration `json:"DataPrepConfiguration,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	DataSetUsageConfiguration *DataSetUsageConfiguration `json:"DataSetUsageConfiguration,omitempty"`
	DatasetParameters []DatasetParameter `json:"DatasetParameters,omitempty"`
	FieldFolders map[string]FieldFolder `json:"FieldFolders,omitempty"`
	ImportMode *string `json:"ImportMode,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	LogicalTableMap map[string]LogicalTable `json:"LogicalTableMap,omitempty"`
	Name *string `json:"Name,omitempty"`
	OutputColumns []OutputColumn `json:"OutputColumns,omitempty"`
	PerformanceConfiguration *PerformanceConfiguration `json:"PerformanceConfiguration,omitempty"`
	PhysicalTableMap map[string]PhysicalTable `json:"PhysicalTableMap,omitempty"`
	RowLevelPermissionDataSet *RowLevelPermissionDataSet `json:"RowLevelPermissionDataSet,omitempty"`
	RowLevelPermissionTagConfiguration *RowLevelPermissionTagConfiguration `json:"RowLevelPermissionTagConfiguration,omitempty"`
	SemanticModelConfiguration *SemanticModelConfiguration `json:"SemanticModelConfiguration,omitempty"`
	UseAs *string `json:"UseAs,omitempty"`
}

type DataSetColumnIdMapping struct {
	SourceColumnId string `json:"SourceColumnId,omitempty"`
	TargetColumnId string `json:"TargetColumnId,omitempty"`
}

type DataSetConfiguration struct {
	ColumnGroupSchemaList []ColumnGroupSchema `json:"ColumnGroupSchemaList,omitempty"`
	DataSetSchema *DataSetSchema `json:"DataSetSchema,omitempty"`
	Placeholder *string `json:"Placeholder,omitempty"`
}

type DataSetDateComparisonFilterCondition struct {
	Operator string `json:"Operator,omitempty"`
	Value *DataSetDateFilterValue `json:"Value,omitempty"`
}

type DataSetDateFilterCondition struct {
	ColumnName *string `json:"ColumnName,omitempty"`
	ComparisonFilterCondition *DataSetDateComparisonFilterCondition `json:"ComparisonFilterCondition,omitempty"`
	RangeFilterCondition *DataSetDateRangeFilterCondition `json:"RangeFilterCondition,omitempty"`
}

type DataSetDateFilterValue struct {
	StaticValue *time.Time `json:"StaticValue,omitempty"`
}

type DataSetDateRangeFilterCondition struct {
	IncludeMaximum bool `json:"IncludeMaximum,omitempty"`
	IncludeMinimum bool `json:"IncludeMinimum,omitempty"`
	RangeMaximum *DataSetDateFilterValue `json:"RangeMaximum,omitempty"`
	RangeMinimum *DataSetDateFilterValue `json:"RangeMinimum,omitempty"`
}

type DataSetIdentifierDeclaration struct {
	DataSetArn string `json:"DataSetArn,omitempty"`
	Identifier string `json:"Identifier,omitempty"`
}

type DataSetNumericComparisonFilterCondition struct {
	Operator string `json:"Operator,omitempty"`
	Value *DataSetNumericFilterValue `json:"Value,omitempty"`
}

type DataSetNumericFilterCondition struct {
	ColumnName *string `json:"ColumnName,omitempty"`
	ComparisonFilterCondition *DataSetNumericComparisonFilterCondition `json:"ComparisonFilterCondition,omitempty"`
	RangeFilterCondition *DataSetNumericRangeFilterCondition `json:"RangeFilterCondition,omitempty"`
}

type DataSetNumericFilterValue struct {
	StaticValue float64 `json:"StaticValue,omitempty"`
}

type DataSetNumericRangeFilterCondition struct {
	IncludeMaximum bool `json:"IncludeMaximum,omitempty"`
	IncludeMinimum bool `json:"IncludeMinimum,omitempty"`
	RangeMaximum *DataSetNumericFilterValue `json:"RangeMaximum,omitempty"`
	RangeMinimum *DataSetNumericFilterValue `json:"RangeMinimum,omitempty"`
}

type DataSetReference struct {
	DataSetArn string `json:"DataSetArn,omitempty"`
	DataSetPlaceholder string `json:"DataSetPlaceholder,omitempty"`
}

type DataSetRefreshProperties struct {
	FailureConfiguration *RefreshFailureConfiguration `json:"FailureConfiguration,omitempty"`
	RefreshConfiguration *RefreshConfiguration `json:"RefreshConfiguration,omitempty"`
}

type DataSetSchema struct {
	ColumnSchemaList []ColumnSchema `json:"ColumnSchemaList,omitempty"`
}

type DataSetSearchFilter struct {
	Name string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value string `json:"Value,omitempty"`
}

type DataSetStringComparisonFilterCondition struct {
	Operator string `json:"Operator,omitempty"`
	Value *DataSetStringFilterValue `json:"Value,omitempty"`
}

type DataSetStringFilterCondition struct {
	ColumnName *string `json:"ColumnName,omitempty"`
	ComparisonFilterCondition *DataSetStringComparisonFilterCondition `json:"ComparisonFilterCondition,omitempty"`
	ListFilterCondition *DataSetStringListFilterCondition `json:"ListFilterCondition,omitempty"`
}

type DataSetStringFilterValue struct {
	StaticValue *string `json:"StaticValue,omitempty"`
}

type DataSetStringListFilterCondition struct {
	Operator string `json:"Operator,omitempty"`
	Values *DataSetStringListFilterValue `json:"Values,omitempty"`
}

type DataSetStringListFilterValue struct {
	StaticValues []string `json:"StaticValues,omitempty"`
}

type DataSetSummary struct {
	Arn *string `json:"Arn,omitempty"`
	ColumnLevelPermissionRulesApplied bool `json:"ColumnLevelPermissionRulesApplied,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	ImportMode *string `json:"ImportMode,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	RowLevelPermissionDataSet *RowLevelPermissionDataSet `json:"RowLevelPermissionDataSet,omitempty"`
	RowLevelPermissionDataSetMap map[string]RowLevelPermissionDataSet `json:"RowLevelPermissionDataSetMap,omitempty"`
	RowLevelPermissionTagConfigurationApplied bool `json:"RowLevelPermissionTagConfigurationApplied,omitempty"`
	UseAs *string `json:"UseAs,omitempty"`
}

type DataSetUsageConfiguration struct {
	DisableUseAsDirectQuerySource bool `json:"DisableUseAsDirectQuerySource,omitempty"`
	DisableUseAsImportedSource bool `json:"DisableUseAsImportedSource,omitempty"`
}

type DataSource struct {
	AlternateDataSourceParameters []DataSourceParameters `json:"AlternateDataSourceParameters,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	DataSourceParameters *DataSourceParameters `json:"DataSourceParameters,omitempty"`
	ErrorInfo *DataSourceErrorInfo `json:"ErrorInfo,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	SecretArn *string `json:"SecretArn,omitempty"`
	SslProperties *SslProperties `json:"SslProperties,omitempty"`
	Status *string `json:"Status,omitempty"`
	Type *string `json:"Type,omitempty"`
	VpcConnectionProperties *VpcConnectionProperties `json:"VpcConnectionProperties,omitempty"`
}

type DataSourceCredentials struct {
	CopySourceArn *string `json:"CopySourceArn,omitempty"`
	CredentialPair *CredentialPair `json:"CredentialPair,omitempty"`
	KeyPairCredentials *KeyPairCredentials `json:"KeyPairCredentials,omitempty"`
	OAuthClientCredentials *OAuthClientCredentials `json:"OAuthClientCredentials,omitempty"`
	SecretArn *string `json:"SecretArn,omitempty"`
	WebProxyCredentials *WebProxyCredentials `json:"WebProxyCredentials,omitempty"`
}

type DataSourceErrorInfo struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DataSourceParameters struct {
	AmazonElasticsearchParameters *AmazonElasticsearchParameters `json:"AmazonElasticsearchParameters,omitempty"`
	AmazonOpenSearchParameters *AmazonOpenSearchParameters `json:"AmazonOpenSearchParameters,omitempty"`
	AthenaParameters *AthenaParameters `json:"AthenaParameters,omitempty"`
	AuroraParameters *AuroraParameters `json:"AuroraParameters,omitempty"`
	AuroraPostgreSqlParameters *AuroraPostgreSqlParameters `json:"AuroraPostgreSqlParameters,omitempty"`
	AwsIotAnalyticsParameters *AwsIotAnalyticsParameters `json:"AwsIotAnalyticsParameters,omitempty"`
	BigQueryParameters *BigQueryParameters `json:"BigQueryParameters,omitempty"`
	ConfluenceParameters *ConfluenceParameters `json:"ConfluenceParameters,omitempty"`
	CustomConnectionParameters *CustomConnectionParameters `json:"CustomConnectionParameters,omitempty"`
	DatabricksParameters *DatabricksParameters `json:"DatabricksParameters,omitempty"`
	ExasolParameters *ExasolParameters `json:"ExasolParameters,omitempty"`
	ImpalaParameters *ImpalaParameters `json:"ImpalaParameters,omitempty"`
	JiraParameters *JiraParameters `json:"JiraParameters,omitempty"`
	MariaDbParameters *MariaDbParameters `json:"MariaDbParameters,omitempty"`
	MySqlParameters *MySqlParameters `json:"MySqlParameters,omitempty"`
	OracleParameters *OracleParameters `json:"OracleParameters,omitempty"`
	PostgreSqlParameters *PostgreSqlParameters `json:"PostgreSqlParameters,omitempty"`
	PrestoParameters *PrestoParameters `json:"PrestoParameters,omitempty"`
	QBusinessParameters *QBusinessParameters `json:"QBusinessParameters,omitempty"`
	RdsParameters *RdsParameters `json:"RdsParameters,omitempty"`
	RedshiftParameters *RedshiftParameters `json:"RedshiftParameters,omitempty"`
	S3KnowledgeBaseParameters *S3KnowledgeBaseParameters `json:"S3KnowledgeBaseParameters,omitempty"`
	S3Parameters *S3Parameters `json:"S3Parameters,omitempty"`
	ServiceNowParameters *ServiceNowParameters `json:"ServiceNowParameters,omitempty"`
	SnowflakeParameters *SnowflakeParameters `json:"SnowflakeParameters,omitempty"`
	SparkParameters *SparkParameters `json:"SparkParameters,omitempty"`
	SqlServerParameters *SqlServerParameters `json:"SqlServerParameters,omitempty"`
	StarburstParameters *StarburstParameters `json:"StarburstParameters,omitempty"`
	TeradataParameters *TeradataParameters `json:"TeradataParameters,omitempty"`
	TrinoParameters *TrinoParameters `json:"TrinoParameters,omitempty"`
	TwitterParameters *TwitterParameters `json:"TwitterParameters,omitempty"`
	WebCrawlerParameters *WebCrawlerParameters `json:"WebCrawlerParameters,omitempty"`
}

type DataSourceSearchFilter struct {
	Name string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value string `json:"Value,omitempty"`
}

type DataSourceSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DataStoriesConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type DataStoriesSharingOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type DatabricksParameters struct {
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
	SqlEndpointPath string `json:"SqlEndpointPath,omitempty"`
}

type DatasetMetadata struct {
	CalculatedFields []TopicCalculatedField `json:"CalculatedFields,omitempty"`
	Columns []TopicColumn `json:"Columns,omitempty"`
	DataAggregation *DataAggregation `json:"DataAggregation,omitempty"`
	DatasetArn string `json:"DatasetArn,omitempty"`
	DatasetDescription *string `json:"DatasetDescription,omitempty"`
	DatasetName *string `json:"DatasetName,omitempty"`
	Filters []TopicFilter `json:"Filters,omitempty"`
	NamedEntities []TopicNamedEntity `json:"NamedEntities,omitempty"`
}

type DatasetParameter struct {
	DateTimeDatasetParameter *DateTimeDatasetParameter `json:"DateTimeDatasetParameter,omitempty"`
	DecimalDatasetParameter *DecimalDatasetParameter `json:"DecimalDatasetParameter,omitempty"`
	IntegerDatasetParameter *IntegerDatasetParameter `json:"IntegerDatasetParameter,omitempty"`
	StringDatasetParameter *StringDatasetParameter `json:"StringDatasetParameter,omitempty"`
}

type DateAxisOptions struct {
	MissingDateVisibility *string `json:"MissingDateVisibility,omitempty"`
}

type DateDimensionField struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	DateGranularity *string `json:"DateGranularity,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *DateTimeFormatConfiguration `json:"FormatConfiguration,omitempty"`
	HierarchyId *string `json:"HierarchyId,omitempty"`
}

type DateMeasureField struct {
	AggregationFunction *string `json:"AggregationFunction,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *DateTimeFormatConfiguration `json:"FormatConfiguration,omitempty"`
}

type DateTimeDatasetParameter struct {
	DefaultValues *DateTimeDatasetParameterDefaultValues `json:"DefaultValues,omitempty"`
	Id string `json:"Id,omitempty"`
	Name string `json:"Name,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
	ValueType string `json:"ValueType,omitempty"`
}

type DateTimeDatasetParameterDefaultValues struct {
	StaticValues []time.Time `json:"StaticValues,omitempty"`
}

type DateTimeDefaultValues struct {
	DynamicValue *DynamicDefaultValue `json:"DynamicValue,omitempty"`
	RollingDate *RollingDateConfiguration `json:"RollingDate,omitempty"`
	StaticValues []time.Time `json:"StaticValues,omitempty"`
}

type DateTimeFormatConfiguration struct {
	DateTimeFormat *string `json:"DateTimeFormat,omitempty"`
	NullValueFormatConfiguration *NullValueFormatConfiguration `json:"NullValueFormatConfiguration,omitempty"`
	NumericFormatConfiguration *NumericFormatConfiguration `json:"NumericFormatConfiguration,omitempty"`
}

type DateTimeHierarchy struct {
	DrillDownFilters []DrillDownFilter `json:"DrillDownFilters,omitempty"`
	HierarchyId string `json:"HierarchyId,omitempty"`
}

type DateTimeParameter struct {
	Name string `json:"Name,omitempty"`
	Values []time.Time `json:"Values,omitempty"`
}

type DateTimeParameterDeclaration struct {
	DefaultValues *DateTimeDefaultValues `json:"DefaultValues,omitempty"`
	MappedDataSetParameters []MappedDataSetParameter `json:"MappedDataSetParameters,omitempty"`
	Name string `json:"Name,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
	ValueWhenUnset *DateTimeValueWhenUnsetConfiguration `json:"ValueWhenUnset,omitempty"`
}

type DateTimePickerControlDisplayOptions struct {
	DateIconVisibility *string `json:"DateIconVisibility,omitempty"`
	DateTimeFormat *string `json:"DateTimeFormat,omitempty"`
	HelperTextVisibility *string `json:"HelperTextVisibility,omitempty"`
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type DateTimeValueWhenUnsetConfiguration struct {
	CustomValue *time.Time `json:"CustomValue,omitempty"`
	ValueWhenUnsetOption *string `json:"ValueWhenUnsetOption,omitempty"`
}

type DecalSettings struct {
	DecalColor *string `json:"DecalColor,omitempty"`
	DecalPatternType *string `json:"DecalPatternType,omitempty"`
	DecalStyleType *string `json:"DecalStyleType,omitempty"`
	DecalVisibility *string `json:"DecalVisibility,omitempty"`
	ElementValue *string `json:"ElementValue,omitempty"`
}

type DecalSettingsConfiguration struct {
	CustomDecalSettings []DecalSettings `json:"CustomDecalSettings,omitempty"`
}

type DecimalDatasetParameter struct {
	DefaultValues *DecimalDatasetParameterDefaultValues `json:"DefaultValues,omitempty"`
	Id string `json:"Id,omitempty"`
	Name string `json:"Name,omitempty"`
	ValueType string `json:"ValueType,omitempty"`
}

type DecimalDatasetParameterDefaultValues struct {
	StaticValues []float64 `json:"StaticValues,omitempty"`
}

type DecimalDefaultValues struct {
	DynamicValue *DynamicDefaultValue `json:"DynamicValue,omitempty"`
	StaticValues []float64 `json:"StaticValues,omitempty"`
}

type DecimalParameter struct {
	Name string `json:"Name,omitempty"`
	Values []float64 `json:"Values,omitempty"`
}

type DecimalParameterDeclaration struct {
	DefaultValues *DecimalDefaultValues `json:"DefaultValues,omitempty"`
	MappedDataSetParameters []MappedDataSetParameter `json:"MappedDataSetParameters,omitempty"`
	Name string `json:"Name,omitempty"`
	ParameterValueType string `json:"ParameterValueType,omitempty"`
	ValueWhenUnset *DecimalValueWhenUnsetConfiguration `json:"ValueWhenUnset,omitempty"`
}

type DecimalPlacesConfiguration struct {
	DecimalPlaces int64 `json:"DecimalPlaces,omitempty"`
}

type DecimalValueWhenUnsetConfiguration struct {
	CustomValue float64 `json:"CustomValue,omitempty"`
	ValueWhenUnsetOption *string `json:"ValueWhenUnsetOption,omitempty"`
}

type DefaultDateTimePickerControlOptions struct {
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *DateTimePickerControlDisplayOptions `json:"DisplayOptions,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DefaultFilterControlConfiguration struct {
	ControlOptions DefaultFilterControlOptions `json:"ControlOptions,omitempty"`
	Title string `json:"Title,omitempty"`
}

type DefaultFilterControlOptions struct {
	DefaultDateTimePickerOptions *DefaultDateTimePickerControlOptions `json:"DefaultDateTimePickerOptions,omitempty"`
	DefaultDropdownOptions *DefaultFilterDropDownControlOptions `json:"DefaultDropdownOptions,omitempty"`
	DefaultListOptions *DefaultFilterListControlOptions `json:"DefaultListOptions,omitempty"`
	DefaultRelativeDateTimeOptions *DefaultRelativeDateTimeControlOptions `json:"DefaultRelativeDateTimeOptions,omitempty"`
	DefaultSliderOptions *DefaultSliderControlOptions `json:"DefaultSliderOptions,omitempty"`
	DefaultTextAreaOptions *DefaultTextAreaControlOptions `json:"DefaultTextAreaOptions,omitempty"`
	DefaultTextFieldOptions *DefaultTextFieldControlOptions `json:"DefaultTextFieldOptions,omitempty"`
}

type DefaultFilterDropDownControlOptions struct {
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *DropDownControlDisplayOptions `json:"DisplayOptions,omitempty"`
	SelectableValues *FilterSelectableValues `json:"SelectableValues,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DefaultFilterListControlOptions struct {
	DisplayOptions *ListControlDisplayOptions `json:"DisplayOptions,omitempty"`
	SelectableValues *FilterSelectableValues `json:"SelectableValues,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DefaultFormatting struct {
	DisplayFormat *string `json:"DisplayFormat,omitempty"`
	DisplayFormatOptions *DisplayFormatOptions `json:"DisplayFormatOptions,omitempty"`
}

type DefaultFreeFormLayoutConfiguration struct {
	CanvasSizeOptions FreeFormLayoutCanvasSizeOptions `json:"CanvasSizeOptions,omitempty"`
}

type DefaultGridLayoutConfiguration struct {
	CanvasSizeOptions GridLayoutCanvasSizeOptions `json:"CanvasSizeOptions,omitempty"`
}

type DefaultInteractiveLayoutConfiguration struct {
	FreeForm *DefaultFreeFormLayoutConfiguration `json:"FreeForm,omitempty"`
	Grid *DefaultGridLayoutConfiguration `json:"Grid,omitempty"`
}

type DefaultNewSheetConfiguration struct {
	InteractiveLayoutConfiguration *DefaultInteractiveLayoutConfiguration `json:"InteractiveLayoutConfiguration,omitempty"`
	PaginatedLayoutConfiguration *DefaultPaginatedLayoutConfiguration `json:"PaginatedLayoutConfiguration,omitempty"`
	SheetContentType *string `json:"SheetContentType,omitempty"`
}

type DefaultPaginatedLayoutConfiguration struct {
	SectionBased *DefaultSectionBasedLayoutConfiguration `json:"SectionBased,omitempty"`
}

type DefaultRelativeDateTimeControlOptions struct {
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *RelativeDateTimeControlDisplayOptions `json:"DisplayOptions,omitempty"`
}

type DefaultSectionBasedLayoutConfiguration struct {
	CanvasSizeOptions SectionBasedLayoutCanvasSizeOptions `json:"CanvasSizeOptions,omitempty"`
}

type DefaultSliderControlOptions struct {
	DisplayOptions *SliderControlDisplayOptions `json:"DisplayOptions,omitempty"`
	MaximumValue float64 `json:"MaximumValue,omitempty"`
	MinimumValue float64 `json:"MinimumValue,omitempty"`
	StepSize float64 `json:"StepSize,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type DefaultTextAreaControlOptions struct {
	Delimiter *string `json:"Delimiter,omitempty"`
	DisplayOptions *TextAreaControlDisplayOptions `json:"DisplayOptions,omitempty"`
}

type DefaultTextFieldControlOptions struct {
	DisplayOptions *TextFieldControlDisplayOptions `json:"DisplayOptions,omitempty"`
}

type DeleteAccountCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DeleteAccountCustomPermissionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteAccountCustomizationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type DeleteAccountCustomizationResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteAccountSubscriptionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DeleteAccountSubscriptionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteActionConnectorRequest struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DeleteActionConnectorResponse struct {
	ActionConnectorId *string `json:"ActionConnectorId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteAnalysisRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ForceDeleteWithoutRecovery bool `json:"force-delete-without-recovery,omitempty"`
	RecoveryWindowInDays int64 `json:"recovery-window-in-days,omitempty"`
}

type DeleteAnalysisResponse struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	DeletionTime *time.Time `json:"DeletionTime,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteBrandAssignmentRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DeleteBrandAssignmentResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
}

type DeleteBrandRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
}

type DeleteBrandResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
}

type DeleteCustomPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
}

type DeleteCustomPermissionsResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteDashboardRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DeleteDashboardResponse struct {
	Arn *string `json:"Arn,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteDataSetRefreshPropertiesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
}

type DeleteDataSetRefreshPropertiesResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteDataSetRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
}

type DeleteDataSetResponse struct {
	Arn *string `json:"Arn,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteDataSourceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
}

type DeleteDataSourceResponse struct {
	Arn *string `json:"Arn,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteDefaultQBusinessApplicationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type DeleteDefaultQBusinessApplicationResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteFolderMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	MemberId string `json:"MemberId,omitempty"`
	MemberType string `json:"MemberType,omitempty"`
}

type DeleteFolderMembershipResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteFolderRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
}

type DeleteFolderResponse struct {
	Arn *string `json:"Arn,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteGroupMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	MemberName string `json:"MemberName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DeleteGroupMembershipResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteGroupRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DeleteGroupResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteIAMPolicyAssignmentRequest struct {
	AssignmentName string `json:"AssignmentName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DeleteIAMPolicyAssignmentResponse struct {
	AssignmentName *string `json:"AssignmentName,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteIdentityPropagationConfigRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ServiceModel string `json:"Service,omitempty"`
}

type DeleteIdentityPropagationConfigResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteNamespaceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DeleteNamespaceResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	ScheduleId string `json:"ScheduleId,omitempty"`
}

type DeleteRefreshScheduleResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ScheduleId *string `json:"ScheduleId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteRoleCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Role string `json:"Role,omitempty"`
}

type DeleteRoleCustomPermissionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteRoleMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MemberName string `json:"MemberName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Role string `json:"Role,omitempty"`
}

type DeleteRoleMembershipResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteTemplateAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
}

type DeleteTemplateAliasResponse struct {
	AliasName *string `json:"AliasName,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
}

type DeleteTemplateRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DeleteTemplateResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
}

type DeleteThemeAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type DeleteThemeAliasResponse struct {
	AliasName *string `json:"AliasName,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
}

type DeleteThemeRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DeleteThemeResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
}

type DeleteTopicRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DatasetId string `json:"DatasetId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type DeleteTopicRefreshScheduleResponse struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type DeleteTopicRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type DeleteTopicResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type DeleteUserByPrincipalIdRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	PrincipalId string `json:"PrincipalId,omitempty"`
}

type DeleteUserByPrincipalIdResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteUserCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type DeleteUserCustomPermissionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteUserRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type DeleteUserResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DeleteVPCConnectionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	VPCConnectionId string `json:"VPCConnectionId,omitempty"`
}

type DeleteVPCConnectionResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
	DeletionStatus *string `json:"DeletionStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VPCConnectionId *string `json:"VPCConnectionId,omitempty"`
}

type DescribeAccountCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAccountCustomPermissionResponse struct {
	CustomPermissionsName *string `json:"CustomPermissionsName,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeAccountCustomizationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	Resolved bool `json:"resolved,omitempty"`
}

type DescribeAccountCustomizationResponse struct {
	AccountCustomization *AccountCustomization `json:"AccountCustomization,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"Namespace,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeAccountSettingsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAccountSettingsResponse struct {
	AccountSettings *AccountSettings `json:"AccountSettings,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeAccountSubscriptionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAccountSubscriptionResponse struct {
	AccountInfo *AccountInfo `json:"AccountInfo,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeActionConnectorPermissionsRequest struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeActionConnectorPermissionsResponse struct {
	ActionConnectorId *string `json:"ActionConnectorId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeActionConnectorRequest struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeActionConnectorResponse struct {
	ActionConnector *ActionConnector `json:"ActionConnector,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeAnalysisDefinitionRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAnalysisDefinitionResponse struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Definition *AnalysisDefinition `json:"Definition,omitempty"`
	Errors []AnalysisError `json:"Errors,omitempty"`
	Name *string `json:"Name,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ResourceStatus *string `json:"ResourceStatus,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
}

type DescribeAnalysisPermissionsRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAnalysisPermissionsResponse struct {
	AnalysisArn *string `json:"AnalysisArn,omitempty"`
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeAnalysisRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAnalysisResponse struct {
	Analysis *Analysis `json:"Analysis,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeAssetBundleExportJobRequest struct {
	AssetBundleExportJobId string `json:"AssetBundleExportJobId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAssetBundleExportJobResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AssetBundleExportJobId *string `json:"AssetBundleExportJobId,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	CloudFormationOverridePropertyConfiguration *AssetBundleCloudFormationOverridePropertyConfiguration `json:"CloudFormationOverridePropertyConfiguration,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DownloadUrl *string `json:"DownloadUrl,omitempty"`
	Errors []AssetBundleExportJobError `json:"Errors,omitempty"`
	ExportFormat *string `json:"ExportFormat,omitempty"`
	IncludeAllDependencies bool `json:"IncludeAllDependencies,omitempty"`
	IncludeFolderMembers *string `json:"IncludeFolderMembers,omitempty"`
	IncludeFolderMemberships bool `json:"IncludeFolderMemberships,omitempty"`
	IncludePermissions bool `json:"IncludePermissions,omitempty"`
	IncludeTags bool `json:"IncludeTags,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ResourceArns []string `json:"ResourceArns,omitempty"`
	Status int `json:"Status,omitempty"`
	ValidationStrategy *AssetBundleExportJobValidationStrategy `json:"ValidationStrategy,omitempty"`
	Warnings []AssetBundleExportJobWarning `json:"Warnings,omitempty"`
}

type DescribeAssetBundleImportJobRequest struct {
	AssetBundleImportJobId string `json:"AssetBundleImportJobId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeAssetBundleImportJobResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AssetBundleImportJobId *string `json:"AssetBundleImportJobId,omitempty"`
	AssetBundleImportSource *AssetBundleImportSourceDescription `json:"AssetBundleImportSource,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Errors []AssetBundleImportJobError `json:"Errors,omitempty"`
	FailureAction *string `json:"FailureAction,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	OverrideParameters *AssetBundleImportJobOverrideParameters `json:"OverrideParameters,omitempty"`
	OverridePermissions *AssetBundleImportJobOverridePermissions `json:"OverridePermissions,omitempty"`
	OverrideTags *AssetBundleImportJobOverrideTags `json:"OverrideTags,omitempty"`
	OverrideValidationStrategy *AssetBundleImportJobOverrideValidationStrategy `json:"OverrideValidationStrategy,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	RollbackErrors []AssetBundleImportJobError `json:"RollbackErrors,omitempty"`
	Status int `json:"Status,omitempty"`
	Warnings []AssetBundleImportJobWarning `json:"Warnings,omitempty"`
}

type DescribeAutomationJobRequest struct {
	AutomationGroupId string `json:"AutomationGroupId,omitempty"`
	AutomationId string `json:"AutomationId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	IncludeInputPayload bool `json:"includeInputPayload,omitempty"`
	IncludeOutputPayload bool `json:"includeOutputPayload,omitempty"`
	JobId string `json:"JobId,omitempty"`
}

type DescribeAutomationJobResponse struct {
	Arn string `json:"Arn,omitempty"`
	CreatedAt *time.Time `json:"CreatedAt,omitempty"`
	EndedAt *time.Time `json:"EndedAt,omitempty"`
	InputPayload *string `json:"InputPayload,omitempty"`
	JobStatus string `json:"JobStatus,omitempty"`
	OutputPayload *string `json:"OutputPayload,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	StartedAt *time.Time `json:"StartedAt,omitempty"`
}

type DescribeBrandAssignmentRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeBrandAssignmentResponse struct {
	BrandArn *string `json:"BrandArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
}

type DescribeBrandPublishedVersionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
}

type DescribeBrandPublishedVersionResponse struct {
	BrandDefinition *BrandDefinition `json:"BrandDefinition,omitempty"`
	BrandDetail *BrandDetail `json:"BrandDetail,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
}

type DescribeBrandRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
	VersionId *string `json:"versionId,omitempty"`
}

type DescribeBrandResponse struct {
	BrandDefinition *BrandDefinition `json:"BrandDefinition,omitempty"`
	BrandDetail *BrandDetail `json:"BrandDetail,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
}

type DescribeCustomPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
}

type DescribeCustomPermissionsResponse struct {
	CustomPermissions *CustomPermissions `json:"CustomPermissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDashboardDefinitionRequest struct {
	AliasName *string `json:"alias-name,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DescribeDashboardDefinitionResponse struct {
	DashboardId *string `json:"DashboardId,omitempty"`
	DashboardPublishOptions *DashboardPublishOptions `json:"DashboardPublishOptions,omitempty"`
	Definition *DashboardVersionDefinition `json:"Definition,omitempty"`
	Errors []DashboardError `json:"Errors,omitempty"`
	Name *string `json:"Name,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ResourceStatus *string `json:"ResourceStatus,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
}

type DescribeDashboardPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
}

type DescribeDashboardPermissionsResponse struct {
	DashboardArn *string `json:"DashboardArn,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	LinkSharingConfiguration *LinkSharingConfiguration `json:"LinkSharingConfiguration,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDashboardRequest struct {
	AliasName *string `json:"alias-name,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DescribeDashboardResponse struct {
	Dashboard *Dashboard `json:"Dashboard,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDashboardSnapshotJobRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	SnapshotJobId string `json:"SnapshotJobId,omitempty"`
}

type DescribeDashboardSnapshotJobResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	SnapshotConfiguration *SnapshotConfiguration `json:"SnapshotConfiguration,omitempty"`
	SnapshotJobId *string `json:"SnapshotJobId,omitempty"`
	Status int `json:"Status,omitempty"`
	UserConfiguration *SnapshotUserConfigurationRedacted `json:"UserConfiguration,omitempty"`
}

type DescribeDashboardSnapshotJobResultRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	SnapshotJobId string `json:"SnapshotJobId,omitempty"`
}

type DescribeDashboardSnapshotJobResultResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	ErrorInfo *SnapshotJobErrorInfo `json:"ErrorInfo,omitempty"`
	JobStatus *string `json:"JobStatus,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Result *SnapshotJobResult `json:"Result,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDashboardsQAConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeDashboardsQAConfigurationResponse struct {
	DashboardsQAStatus *string `json:"DashboardsQAStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDataSetPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
}

type DescribeDataSetPermissionsResponse struct {
	DataSetArn *string `json:"DataSetArn,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDataSetRefreshPropertiesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
}

type DescribeDataSetRefreshPropertiesResponse struct {
	DataSetRefreshProperties *DataSetRefreshProperties `json:"DataSetRefreshProperties,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDataSetRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
}

type DescribeDataSetResponse struct {
	DataSet *DataSet `json:"DataSet,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDataSourcePermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
}

type DescribeDataSourcePermissionsResponse struct {
	DataSourceArn *string `json:"DataSourceArn,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDataSourceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
}

type DescribeDataSourceResponse struct {
	DataSource *DataSource `json:"DataSource,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeDefaultQBusinessApplicationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type DescribeDefaultQBusinessApplicationResponse struct {
	ApplicationId *string `json:"ApplicationId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeFolderPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type DescribeFolderPermissionsResponse struct {
	Arn *string `json:"Arn,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeFolderRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
}

type DescribeFolderResolvedPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type DescribeFolderResolvedPermissionsResponse struct {
	Arn *string `json:"Arn,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeFolderResponse struct {
	Folder *Folder `json:"Folder,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeGroupMembershipRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	MemberName string `json:"MemberName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DescribeGroupMembershipResponse struct {
	GroupMember *GroupMember `json:"GroupMember,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeGroupRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DescribeGroupResponse struct {
	Group *Group `json:"Group,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeIAMPolicyAssignmentRequest struct {
	AssignmentName string `json:"AssignmentName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DescribeIAMPolicyAssignmentResponse struct {
	IAMPolicyAssignment *IAMPolicyAssignment `json:"IAMPolicyAssignment,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeIngestionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	IngestionId string `json:"IngestionId,omitempty"`
}

type DescribeIngestionResponse struct {
	Ingestion *Ingestion `json:"Ingestion,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeIpRestrictionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeIpRestrictionResponse struct {
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	IpRestrictionRuleMap map[string]string `json:"IpRestrictionRuleMap,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VpcEndpointIdRestrictionRuleMap map[string]string `json:"VpcEndpointIdRestrictionRuleMap,omitempty"`
	VpcIdRestrictionRuleMap map[string]string `json:"VpcIdRestrictionRuleMap,omitempty"`
}

type DescribeKeyRegistrationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DefaultKeyOnly bool `json:"default-key-only,omitempty"`
}

type DescribeKeyRegistrationResponse struct {
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	KeyRegistration []RegisteredCustomerManagedKey `json:"KeyRegistration,omitempty"`
	QDataKey *QDataKey `json:"QDataKey,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeNamespaceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DescribeNamespaceResponse struct {
	Namespace *NamespaceInfoV2 `json:"Namespace,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeQPersonalizationConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeQPersonalizationConfigurationResponse struct {
	PersonalizationMode *string `json:"PersonalizationMode,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeQuickSightQSearchConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
}

type DescribeQuickSightQSearchConfigurationResponse struct {
	QSearchStatus *string `json:"QSearchStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	ScheduleId string `json:"ScheduleId,omitempty"`
}

type DescribeRefreshScheduleResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RefreshSchedule *RefreshSchedule `json:"RefreshSchedule,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeRoleCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Role string `json:"Role,omitempty"`
}

type DescribeRoleCustomPermissionResponse struct {
	CustomPermissionsName *string `json:"CustomPermissionsName,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeSelfUpgradeConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type DescribeSelfUpgradeConfigurationResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	SelfUpgradeConfiguration *SelfUpgradeConfiguration `json:"SelfUpgradeConfiguration,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeTemplateAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
}

type DescribeTemplateAliasResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateAlias *TemplateAlias `json:"TemplateAlias,omitempty"`
}

type DescribeTemplateDefinitionRequest struct {
	AliasName *string `json:"alias-name,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DescribeTemplateDefinitionResponse struct {
	Definition *TemplateVersionDefinition `json:"Definition,omitempty"`
	Errors []TemplateError `json:"Errors,omitempty"`
	Name *string `json:"Name,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ResourceStatus *string `json:"ResourceStatus,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
}

type DescribeTemplatePermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
}

type DescribeTemplatePermissionsResponse struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateArn *string `json:"TemplateArn,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
}

type DescribeTemplateRequest struct {
	AliasName *string `json:"alias-name,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DescribeTemplateResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	Template *Template `json:"Template,omitempty"`
}

type DescribeThemeAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type DescribeThemeAliasResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeAlias *ThemeAlias `json:"ThemeAlias,omitempty"`
}

type DescribeThemePermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type DescribeThemePermissionsResponse struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
}

type DescribeThemeRequest struct {
	AliasName *string `json:"alias-name,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
	VersionNumber int64 `json:"version-number,omitempty"`
}

type DescribeThemeResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	Theme *Theme `json:"Theme,omitempty"`
}

type DescribeTopicPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type DescribeTopicPermissionsResponse struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type DescribeTopicRefreshRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	RefreshId string `json:"RefreshId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type DescribeTopicRefreshResponse struct {
	RefreshDetails *TopicRefreshDetails `json:"RefreshDetails,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type DescribeTopicRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DatasetId string `json:"DatasetId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type DescribeTopicRefreshScheduleResponse struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
	RefreshSchedule *TopicRefreshSchedule `json:"RefreshSchedule,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type DescribeTopicRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type DescribeTopicResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CustomInstructions *CustomInstructions `json:"CustomInstructions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	Topic *TopicDetails `json:"Topic,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type DescribeUserRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type DescribeUserResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	User *User `json:"User,omitempty"`
}

type DescribeVPCConnectionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	VPCConnectionId string `json:"VPCConnectionId,omitempty"`
}

type DescribeVPCConnectionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VPCConnection *VPCConnection `json:"VPCConnection,omitempty"`
}

type DestinationParameterValueConfiguration struct {
	CustomValuesConfiguration *CustomValuesConfiguration `json:"CustomValuesConfiguration,omitempty"`
	SelectAllValueOptions *string `json:"SelectAllValueOptions,omitempty"`
	SourceColumn *ColumnIdentifier `json:"SourceColumn,omitempty"`
	SourceField *string `json:"SourceField,omitempty"`
	SourceParameterName *string `json:"SourceParameterName,omitempty"`
}

type DestinationTable struct {
	Alias string `json:"Alias,omitempty"`
	Source DestinationTableSource `json:"Source,omitempty"`
}

type DestinationTableSource struct {
	TransformOperationId string `json:"TransformOperationId,omitempty"`
}

type DimensionField struct {
	CategoricalDimensionField *CategoricalDimensionField `json:"CategoricalDimensionField,omitempty"`
	DateDimensionField *DateDimensionField `json:"DateDimensionField,omitempty"`
	NumericalDimensionField *NumericalDimensionField `json:"NumericalDimensionField,omitempty"`
}

type DisplayFormatOptions struct {
	BlankCellFormat *string `json:"BlankCellFormat,omitempty"`
	CurrencySymbol *string `json:"CurrencySymbol,omitempty"`
	DateFormat *string `json:"DateFormat,omitempty"`
	DecimalSeparator *string `json:"DecimalSeparator,omitempty"`
	FractionDigits int `json:"FractionDigits,omitempty"`
	GroupingSeparator *string `json:"GroupingSeparator,omitempty"`
	NegativeFormat *NegativeFormat `json:"NegativeFormat,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
	Suffix *string `json:"Suffix,omitempty"`
	UnitScaler *string `json:"UnitScaler,omitempty"`
	UseBlankCellFormat bool `json:"UseBlankCellFormat,omitempty"`
	UseGrouping bool `json:"UseGrouping,omitempty"`
}

type DonutCenterOptions struct {
	LabelVisibility *string `json:"LabelVisibility,omitempty"`
}

type DonutOptions struct {
	ArcOptions *ArcOptions `json:"ArcOptions,omitempty"`
	DonutCenterOptions *DonutCenterOptions `json:"DonutCenterOptions,omitempty"`
}

type DrillDownFilter struct {
	CategoryFilter *CategoryDrillDownFilter `json:"CategoryFilter,omitempty"`
	NumericEqualityFilter *NumericEqualityDrillDownFilter `json:"NumericEqualityFilter,omitempty"`
	TimeRangeFilter *TimeRangeDrillDownFilter `json:"TimeRangeFilter,omitempty"`
}

type DropDownControlDisplayOptions struct {
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	SelectAllOptions *ListControlSelectAllOptions `json:"SelectAllOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type DynamicDefaultValue struct {
	DefaultValueColumn ColumnIdentifier `json:"DefaultValueColumn,omitempty"`
	GroupNameColumn *ColumnIdentifier `json:"GroupNameColumn,omitempty"`
	UserNameColumn *ColumnIdentifier `json:"UserNameColumn,omitempty"`
}

type EmptyVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type Entity struct {
	Path *string `json:"Path,omitempty"`
}

type ErrorInfo struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ExasolParameters struct {
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type ExcludePeriodConfiguration struct {
	Amount int `json:"Amount,omitempty"`
	Granularity string `json:"Granularity,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type ExecutiveSummaryConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type ExecutiveSummaryOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type ExplicitHierarchy struct {
	Columns []ColumnIdentifier `json:"Columns,omitempty"`
	DrillDownFilters []DrillDownFilter `json:"DrillDownFilters,omitempty"`
	HierarchyId string `json:"HierarchyId,omitempty"`
}

type ExportHiddenFieldsOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type ExportToCSVOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type ExportWithHiddenFieldsOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type FailedKeyRegistrationEntry struct {
	KeyArn *string `json:"KeyArn,omitempty"`
	Message string `json:"Message,omitempty"`
	SenderFault bool `json:"SenderFault,omitempty"`
	StatusCode int `json:"StatusCode,omitempty"`
}

type FieldBarSeriesItem struct {
	FieldId string `json:"FieldId,omitempty"`
	Settings *BarChartSeriesSettings `json:"Settings,omitempty"`
}

type FieldBasedTooltip struct {
	AggregationVisibility *string `json:"AggregationVisibility,omitempty"`
	TooltipFields []TooltipItem `json:"TooltipFields,omitempty"`
	TooltipTitleType *string `json:"TooltipTitleType,omitempty"`
}

type FieldComboSeriesItem struct {
	FieldId string `json:"FieldId,omitempty"`
	Settings *ComboChartSeriesSettings `json:"Settings,omitempty"`
}

type FieldFolder struct {
	Columns []string `json:"columns,omitempty"`
	Description *string `json:"description,omitempty"`
}

type FieldLabelType struct {
	FieldId *string `json:"FieldId,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type FieldSeriesItem struct {
	AxisBinding string `json:"AxisBinding,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	Settings *LineChartSeriesSettings `json:"Settings,omitempty"`
}

type FieldSort struct {
	Direction string `json:"Direction,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
}

type FieldSortOptions struct {
	ColumnSort *ColumnSort `json:"ColumnSort,omitempty"`
	FieldSort *FieldSort `json:"FieldSort,omitempty"`
}

type FieldTooltipItem struct {
	FieldId string `json:"FieldId,omitempty"`
	Label *string `json:"Label,omitempty"`
	TooltipTarget *string `json:"TooltipTarget,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type FilledMapAggregatedFieldWells struct {
	Geospatial []DimensionField `json:"Geospatial,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type FilledMapConditionalFormatting struct {
	ConditionalFormattingOptions []FilledMapConditionalFormattingOption `json:"ConditionalFormattingOptions,omitempty"`
}

type FilledMapConditionalFormattingOption struct {
	Shape FilledMapShapeConditionalFormatting `json:"Shape,omitempty"`
}

type FilledMapConfiguration struct {
	FieldWells *FilledMapFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	MapStyleOptions *GeospatialMapStyleOptions `json:"MapStyleOptions,omitempty"`
	SortConfiguration *FilledMapSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	WindowOptions *GeospatialWindowOptions `json:"WindowOptions,omitempty"`
}

type FilledMapFieldWells struct {
	FilledMapAggregatedFieldWells *FilledMapAggregatedFieldWells `json:"FilledMapAggregatedFieldWells,omitempty"`
}

type FilledMapShapeConditionalFormatting struct {
	FieldId string `json:"FieldId,omitempty"`
	Format *ShapeConditionalFormat `json:"Format,omitempty"`
}

type FilledMapSortConfiguration struct {
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
}

type FilledMapVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *FilledMapConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	ConditionalFormatting *FilledMapConditionalFormatting `json:"ConditionalFormatting,omitempty"`
	GeocodingPreferences []GeocodePreference `json:"GeocodingPreferences,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type Filter struct {
	CategoryFilter *CategoryFilter `json:"CategoryFilter,omitempty"`
	NestedFilter *NestedFilter `json:"NestedFilter,omitempty"`
	NumericEqualityFilter *NumericEqualityFilter `json:"NumericEqualityFilter,omitempty"`
	NumericRangeFilter *NumericRangeFilter `json:"NumericRangeFilter,omitempty"`
	RelativeDatesFilter *RelativeDatesFilter `json:"RelativeDatesFilter,omitempty"`
	TimeEqualityFilter *TimeEqualityFilter `json:"TimeEqualityFilter,omitempty"`
	TimeRangeFilter *TimeRangeFilter `json:"TimeRangeFilter,omitempty"`
	TopBottomFilter *TopBottomFilter `json:"TopBottomFilter,omitempty"`
}

type FilterAggMetrics struct {
	Function *string `json:"Function,omitempty"`
	MetricOperand *Identifier `json:"MetricOperand,omitempty"`
	SortDirection *string `json:"SortDirection,omitempty"`
}

type FilterControl struct {
	CrossSheet *FilterCrossSheetControl `json:"CrossSheet,omitempty"`
	DateTimePicker *FilterDateTimePickerControl `json:"DateTimePicker,omitempty"`
	Dropdown *FilterDropDownControl `json:"Dropdown,omitempty"`
	List *FilterListControl `json:"List,omitempty"`
	RelativeDateTime *FilterRelativeDateTimeControl `json:"RelativeDateTime,omitempty"`
	Slider *FilterSliderControl `json:"Slider,omitempty"`
	TextArea *FilterTextAreaControl `json:"TextArea,omitempty"`
	TextField *FilterTextFieldControl `json:"TextField,omitempty"`
}

type FilterCrossSheetControl struct {
	CascadingControlConfiguration *CascadingControlConfiguration `json:"CascadingControlConfiguration,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
}

type FilterDateTimePickerControl struct {
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *DateTimePickerControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	Title string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type FilterDropDownControl struct {
	CascadingControlConfiguration *CascadingControlConfiguration `json:"CascadingControlConfiguration,omitempty"`
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *DropDownControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SelectableValues *FilterSelectableValues `json:"SelectableValues,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	Title string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type FilterGroup struct {
	CrossDataset string `json:"CrossDataset,omitempty"`
	FilterGroupId string `json:"FilterGroupId,omitempty"`
	Filters []Filter `json:"Filters,omitempty"`
	ScopeConfiguration FilterScopeConfiguration `json:"ScopeConfiguration,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type FilterListConfiguration struct {
	CategoryValues []string `json:"CategoryValues,omitempty"`
	MatchOperator string `json:"MatchOperator,omitempty"`
	NullOption *string `json:"NullOption,omitempty"`
	SelectAllOptions *string `json:"SelectAllOptions,omitempty"`
}

type FilterListControl struct {
	CascadingControlConfiguration *CascadingControlConfiguration `json:"CascadingControlConfiguration,omitempty"`
	DisplayOptions *ListControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SelectableValues *FilterSelectableValues `json:"SelectableValues,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	Title string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type FilterOperation struct {
	ConditionExpression *string `json:"ConditionExpression,omitempty"`
	DateFilterCondition *DataSetDateFilterCondition `json:"DateFilterCondition,omitempty"`
	NumericFilterCondition *DataSetNumericFilterCondition `json:"NumericFilterCondition,omitempty"`
	StringFilterCondition *DataSetStringFilterCondition `json:"StringFilterCondition,omitempty"`
}

type FilterOperationSelectedFieldsConfiguration struct {
	SelectedColumns []ColumnIdentifier `json:"SelectedColumns,omitempty"`
	SelectedFieldOptions *string `json:"SelectedFieldOptions,omitempty"`
	SelectedFields []string `json:"SelectedFields,omitempty"`
}

type FilterOperationTargetVisualsConfiguration struct {
	SameSheetTargetVisualConfiguration *SameSheetTargetVisualConfiguration `json:"SameSheetTargetVisualConfiguration,omitempty"`
}

type FilterRelativeDateTimeControl struct {
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *RelativeDateTimeControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	Title string `json:"Title,omitempty"`
}

type FilterScopeConfiguration struct {
	AllSheets *AllSheetsFilterScopeConfiguration `json:"AllSheets,omitempty"`
	SelectedSheets *SelectedSheetsFilterScopeConfiguration `json:"SelectedSheets,omitempty"`
}

type FilterSelectableValues struct {
	Values []string `json:"Values,omitempty"`
}

type FilterSliderControl struct {
	DisplayOptions *SliderControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	MaximumValue float64 `json:"MaximumValue,omitempty"`
	MinimumValue float64 `json:"MinimumValue,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	StepSize float64 `json:"StepSize,omitempty"`
	Title string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type FilterTextAreaControl struct {
	Delimiter *string `json:"Delimiter,omitempty"`
	DisplayOptions *TextAreaControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	Title string `json:"Title,omitempty"`
}

type FilterTextFieldControl struct {
	DisplayOptions *TextFieldControlDisplayOptions `json:"DisplayOptions,omitempty"`
	FilterControlId string `json:"FilterControlId,omitempty"`
	SourceFilterId string `json:"SourceFilterId,omitempty"`
	Title string `json:"Title,omitempty"`
}

type FiltersOperation struct {
	Alias string `json:"Alias,omitempty"`
	FilterOperations []FilterOperation `json:"FilterOperations,omitempty"`
	Source TransformOperationSource `json:"Source,omitempty"`
}

type FlowSummary struct {
	Arn string `json:"Arn,omitempty"`
	CreatedBy *string `json:"CreatedBy,omitempty"`
	CreatedTime time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
	LastPublishedAt *time.Time `json:"LastPublishedAt,omitempty"`
	LastPublishedBy *string `json:"LastPublishedBy,omitempty"`
	LastUpdatedBy *string `json:"LastUpdatedBy,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name string `json:"Name,omitempty"`
	PublishState *string `json:"PublishState,omitempty"`
	RunCount int `json:"RunCount,omitempty"`
	UserCount int `json:"UserCount,omitempty"`
}

type Folder struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	FolderPath []string `json:"FolderPath,omitempty"`
	FolderType *string `json:"FolderType,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	SharingModel *string `json:"SharingModel,omitempty"`
}

type FolderMember struct {
	MemberId *string `json:"MemberId,omitempty"`
	MemberType *string `json:"MemberType,omitempty"`
}

type FolderSearchFilter struct {
	Name *string `json:"Name,omitempty"`
	Operator *string `json:"Operator,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type FolderSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	FolderType *string `json:"FolderType,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	SharingModel *string `json:"SharingModel,omitempty"`
}

type Font struct {
	FontFamily *string `json:"FontFamily,omitempty"`
}

type FontConfiguration struct {
	FontColor *string `json:"FontColor,omitempty"`
	FontDecoration *string `json:"FontDecoration,omitempty"`
	FontFamily *string `json:"FontFamily,omitempty"`
	FontSize *FontSize `json:"FontSize,omitempty"`
	FontStyle *string `json:"FontStyle,omitempty"`
	FontWeight *FontWeight `json:"FontWeight,omitempty"`
}

type FontSize struct {
	Absolute *string `json:"Absolute,omitempty"`
	Relative *string `json:"Relative,omitempty"`
}

type FontWeight struct {
	Name *string `json:"Name,omitempty"`
}

type ForecastComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	CustomSeasonalityValue int `json:"CustomSeasonalityValue,omitempty"`
	LowerBoundary float64 `json:"LowerBoundary,omitempty"`
	Name *string `json:"Name,omitempty"`
	PeriodsBackward int `json:"PeriodsBackward,omitempty"`
	PeriodsForward int `json:"PeriodsForward,omitempty"`
	PredictionInterval int `json:"PredictionInterval,omitempty"`
	Seasonality *string `json:"Seasonality,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
	UpperBoundary float64 `json:"UpperBoundary,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type ForecastConfiguration struct {
	ForecastProperties *TimeBasedForecastProperties `json:"ForecastProperties,omitempty"`
	Scenario *ForecastScenario `json:"Scenario,omitempty"`
}

type ForecastScenario struct {
	WhatIfPointScenario *WhatIfPointScenario `json:"WhatIfPointScenario,omitempty"`
	WhatIfRangeScenario *WhatIfRangeScenario `json:"WhatIfRangeScenario,omitempty"`
}

type FormatConfiguration struct {
	DateTimeFormatConfiguration *DateTimeFormatConfiguration `json:"DateTimeFormatConfiguration,omitempty"`
	NumberFormatConfiguration *NumberFormatConfiguration `json:"NumberFormatConfiguration,omitempty"`
	StringFormatConfiguration *StringFormatConfiguration `json:"StringFormatConfiguration,omitempty"`
}

type FreeFormLayoutCanvasSizeOptions struct {
	ScreenCanvasSizeOptions *FreeFormLayoutScreenCanvasSizeOptions `json:"ScreenCanvasSizeOptions,omitempty"`
}

type FreeFormLayoutConfiguration struct {
	CanvasSizeOptions *FreeFormLayoutCanvasSizeOptions `json:"CanvasSizeOptions,omitempty"`
	Elements []FreeFormLayoutElement `json:"Elements,omitempty"`
	Groups []SheetLayoutGroup `json:"Groups,omitempty"`
}

type FreeFormLayoutElement struct {
	BackgroundStyle *FreeFormLayoutElementBackgroundStyle `json:"BackgroundStyle,omitempty"`
	BorderRadius *string `json:"BorderRadius,omitempty"`
	BorderStyle *FreeFormLayoutElementBorderStyle `json:"BorderStyle,omitempty"`
	ElementId string `json:"ElementId,omitempty"`
	ElementType string `json:"ElementType,omitempty"`
	Height string `json:"Height,omitempty"`
	LoadingAnimation *LoadingAnimation `json:"LoadingAnimation,omitempty"`
	Padding *string `json:"Padding,omitempty"`
	RenderingRules []SheetElementRenderingRule `json:"RenderingRules,omitempty"`
	SelectedBorderStyle *FreeFormLayoutElementBorderStyle `json:"SelectedBorderStyle,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
	Width string `json:"Width,omitempty"`
	XAxisLocation string `json:"XAxisLocation,omitempty"`
	YAxisLocation string `json:"YAxisLocation,omitempty"`
}

type FreeFormLayoutElementBackgroundStyle struct {
	Color *string `json:"Color,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type FreeFormLayoutElementBorderStyle struct {
	Color *string `json:"Color,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
	Width *string `json:"Width,omitempty"`
}

type FreeFormLayoutScreenCanvasSizeOptions struct {
	OptimizedViewPortWidth string `json:"OptimizedViewPortWidth,omitempty"`
}

type FreeFormSectionLayoutConfiguration struct {
	Elements []FreeFormLayoutElement `json:"Elements,omitempty"`
}

type FunnelChartAggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type FunnelChartConfiguration struct {
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	DataLabelOptions *FunnelChartDataLabelOptions `json:"DataLabelOptions,omitempty"`
	FieldWells *FunnelChartFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	SortConfiguration *FunnelChartSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	ValueLabelOptions *ChartAxisLabelOptions `json:"ValueLabelOptions,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type FunnelChartDataLabelOptions struct {
	CategoryLabelVisibility *string `json:"CategoryLabelVisibility,omitempty"`
	LabelColor *string `json:"LabelColor,omitempty"`
	LabelFontConfiguration *FontConfiguration `json:"LabelFontConfiguration,omitempty"`
	MeasureDataLabelStyle *string `json:"MeasureDataLabelStyle,omitempty"`
	MeasureLabelVisibility *string `json:"MeasureLabelVisibility,omitempty"`
	Position *string `json:"Position,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type FunnelChartFieldWells struct {
	FunnelChartAggregatedFieldWells *FunnelChartAggregatedFieldWells `json:"FunnelChartAggregatedFieldWells,omitempty"`
}

type FunnelChartSortConfiguration struct {
	CategoryItemsLimit *ItemsLimitConfiguration `json:"CategoryItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
}

type FunnelChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *FunnelChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type GaugeChartArcConditionalFormatting struct {
	ForegroundColor *ConditionalFormattingColor `json:"ForegroundColor,omitempty"`
}

type GaugeChartColorConfiguration struct {
	BackgroundColor *string `json:"BackgroundColor,omitempty"`
	ForegroundColor *string `json:"ForegroundColor,omitempty"`
}

type GaugeChartConditionalFormatting struct {
	ConditionalFormattingOptions []GaugeChartConditionalFormattingOption `json:"ConditionalFormattingOptions,omitempty"`
}

type GaugeChartConditionalFormattingOption struct {
	Arc *GaugeChartArcConditionalFormatting `json:"Arc,omitempty"`
	PrimaryValue *GaugeChartPrimaryValueConditionalFormatting `json:"PrimaryValue,omitempty"`
}

type GaugeChartConfiguration struct {
	ColorConfiguration *GaugeChartColorConfiguration `json:"ColorConfiguration,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *GaugeChartFieldWells `json:"FieldWells,omitempty"`
	GaugeChartOptions *GaugeChartOptions `json:"GaugeChartOptions,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	TooltipOptions *TooltipOptions `json:"TooltipOptions,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type GaugeChartFieldWells struct {
	TargetValues []MeasureField `json:"TargetValues,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type GaugeChartOptions struct {
	Arc *ArcConfiguration `json:"Arc,omitempty"`
	ArcAxis *ArcAxisConfiguration `json:"ArcAxis,omitempty"`
	Comparison *ComparisonConfiguration `json:"Comparison,omitempty"`
	PrimaryValueDisplayType *string `json:"PrimaryValueDisplayType,omitempty"`
	PrimaryValueFontConfiguration *FontConfiguration `json:"PrimaryValueFontConfiguration,omitempty"`
}

type GaugeChartPrimaryValueConditionalFormatting struct {
	Icon *ConditionalFormattingIcon `json:"Icon,omitempty"`
	TextColor *ConditionalFormattingColor `json:"TextColor,omitempty"`
}

type GaugeChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *GaugeChartConfiguration `json:"ChartConfiguration,omitempty"`
	ConditionalFormatting *GaugeChartConditionalFormatting `json:"ConditionalFormatting,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type GenerateEmbedUrlForAnonymousUserRequest struct {
	AllowedDomains []string `json:"AllowedDomains,omitempty"`
	AuthorizedResourceArns []string `json:"AuthorizedResourceArns,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ExperienceConfiguration AnonymousUserEmbeddingExperienceConfiguration `json:"ExperienceConfiguration,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	SessionLifetimeInMinutes int64 `json:"SessionLifetimeInMinutes,omitempty"`
	SessionTags []SessionTag `json:"SessionTags,omitempty"`
}

type GenerateEmbedUrlForAnonymousUserResponse struct {
	AnonymousUserArn string `json:"AnonymousUserArn,omitempty"`
	EmbedUrl string `json:"EmbedUrl,omitempty"`
	RequestId string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GenerateEmbedUrlForRegisteredUserRequest struct {
	AllowedDomains []string `json:"AllowedDomains,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ExperienceConfiguration RegisteredUserEmbeddingExperienceConfiguration `json:"ExperienceConfiguration,omitempty"`
	SessionLifetimeInMinutes int64 `json:"SessionLifetimeInMinutes,omitempty"`
	UserArn string `json:"UserArn,omitempty"`
}

type GenerateEmbedUrlForRegisteredUserResponse struct {
	EmbedUrl string `json:"EmbedUrl,omitempty"`
	RequestId string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GenerateEmbedUrlForRegisteredUserWithIdentityRequest struct {
	AllowedDomains []string `json:"AllowedDomains,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ExperienceConfiguration RegisteredUserEmbeddingExperienceConfiguration `json:"ExperienceConfiguration,omitempty"`
	SessionLifetimeInMinutes int64 `json:"SessionLifetimeInMinutes,omitempty"`
}

type GenerateEmbedUrlForRegisteredUserWithIdentityResponse struct {
	EmbedUrl string `json:"EmbedUrl,omitempty"`
	RequestId string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GeneratedAnswerResult struct {
	AnswerId *string `json:"AnswerId,omitempty"`
	AnswerStatus *string `json:"AnswerStatus,omitempty"`
	QuestionId *string `json:"QuestionId,omitempty"`
	QuestionText *string `json:"QuestionText,omitempty"`
	QuestionUrl *string `json:"QuestionUrl,omitempty"`
	Restatement *string `json:"Restatement,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
	TopicName *string `json:"TopicName,omitempty"`
}

type GenerativeAuthoringConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type GeoSpatialColumnGroup struct {
	Columns []string `json:"Columns,omitempty"`
	CountryCode *string `json:"CountryCode,omitempty"`
	Name string `json:"Name,omitempty"`
}

type GeocodePreference struct {
	Preference GeocodePreferenceValue `json:"Preference,omitempty"`
	RequestKey GeocoderHierarchy `json:"RequestKey,omitempty"`
}

type GeocodePreferenceValue struct {
	Coordinate *Coordinate `json:"Coordinate,omitempty"`
	GeocoderHierarchy *GeocoderHierarchy `json:"GeocoderHierarchy,omitempty"`
}

type GeocoderHierarchy struct {
	City *string `json:"City,omitempty"`
	Country *string `json:"Country,omitempty"`
	County *string `json:"County,omitempty"`
	PostCode *string `json:"PostCode,omitempty"`
	State *string `json:"State,omitempty"`
}

type GeospatialCategoricalColor struct {
	CategoryDataColors []GeospatialCategoricalDataColor `json:"CategoryDataColors,omitempty"`
	DefaultOpacity float64 `json:"DefaultOpacity,omitempty"`
	NullDataSettings *GeospatialNullDataSettings `json:"NullDataSettings,omitempty"`
	NullDataVisibility *string `json:"NullDataVisibility,omitempty"`
}

type GeospatialCategoricalDataColor struct {
	Color string `json:"Color,omitempty"`
	DataValue string `json:"DataValue,omitempty"`
}

type GeospatialCircleRadius struct {
	Radius float64 `json:"Radius,omitempty"`
}

type GeospatialCircleSymbolStyle struct {
	CircleRadius *GeospatialCircleRadius `json:"CircleRadius,omitempty"`
	FillColor *GeospatialColor `json:"FillColor,omitempty"`
	StrokeColor *GeospatialColor `json:"StrokeColor,omitempty"`
	StrokeWidth *GeospatialLineWidth `json:"StrokeWidth,omitempty"`
}

type GeospatialColor struct {
	Categorical *GeospatialCategoricalColor `json:"Categorical,omitempty"`
	Gradient *GeospatialGradientColor `json:"Gradient,omitempty"`
	Solid *GeospatialSolidColor `json:"Solid,omitempty"`
}

type GeospatialCoordinateBounds struct {
	East float64 `json:"East,omitempty"`
	North float64 `json:"North,omitempty"`
	South float64 `json:"South,omitempty"`
	West float64 `json:"West,omitempty"`
}

type GeospatialDataSourceItem struct {
	StaticFileDataSource *GeospatialStaticFileSource `json:"StaticFileDataSource,omitempty"`
}

type GeospatialGradientColor struct {
	DefaultOpacity float64 `json:"DefaultOpacity,omitempty"`
	NullDataSettings *GeospatialNullDataSettings `json:"NullDataSettings,omitempty"`
	NullDataVisibility *string `json:"NullDataVisibility,omitempty"`
	StepColors []GeospatialGradientStepColor `json:"StepColors,omitempty"`
}

type GeospatialGradientStepColor struct {
	Color string `json:"Color,omitempty"`
	DataValue float64 `json:"DataValue,omitempty"`
}

type GeospatialHeatmapColorScale struct {
	Colors []GeospatialHeatmapDataColor `json:"Colors,omitempty"`
}

type GeospatialHeatmapConfiguration struct {
	HeatmapColor *GeospatialHeatmapColorScale `json:"HeatmapColor,omitempty"`
}

type GeospatialHeatmapDataColor struct {
	Color string `json:"Color,omitempty"`
}

type GeospatialLayerColorField struct {
	ColorDimensionsFields []DimensionField `json:"ColorDimensionsFields,omitempty"`
	ColorValuesFields []MeasureField `json:"ColorValuesFields,omitempty"`
}

type GeospatialLayerDefinition struct {
	LineLayer *GeospatialLineLayer `json:"LineLayer,omitempty"`
	PointLayer *GeospatialPointLayer `json:"PointLayer,omitempty"`
	PolygonLayer *GeospatialPolygonLayer `json:"PolygonLayer,omitempty"`
}

type GeospatialLayerItem struct {
	Actions []LayerCustomAction `json:"Actions,omitempty"`
	DataSource *GeospatialDataSourceItem `json:"DataSource,omitempty"`
	JoinDefinition *GeospatialLayerJoinDefinition `json:"JoinDefinition,omitempty"`
	Label *string `json:"Label,omitempty"`
	LayerDefinition *GeospatialLayerDefinition `json:"LayerDefinition,omitempty"`
	LayerId string `json:"LayerId,omitempty"`
	LayerType *string `json:"LayerType,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type GeospatialLayerJoinDefinition struct {
	ColorField *GeospatialLayerColorField `json:"ColorField,omitempty"`
	DatasetKeyField *UnaggregatedField `json:"DatasetKeyField,omitempty"`
	ShapeKeyField *string `json:"ShapeKeyField,omitempty"`
}

type GeospatialLayerMapConfiguration struct {
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	MapLayers []GeospatialLayerItem `json:"MapLayers,omitempty"`
	MapState *GeospatialMapState `json:"MapState,omitempty"`
	MapStyle *GeospatialMapStyle `json:"MapStyle,omitempty"`
}

type GeospatialLineLayer struct {
	Style GeospatialLineStyle `json:"Style,omitempty"`
}

type GeospatialLineStyle struct {
	LineSymbolStyle *GeospatialLineSymbolStyle `json:"LineSymbolStyle,omitempty"`
}

type GeospatialLineSymbolStyle struct {
	FillColor *GeospatialColor `json:"FillColor,omitempty"`
	LineWidth *GeospatialLineWidth `json:"LineWidth,omitempty"`
}

type GeospatialLineWidth struct {
	LineWidth float64 `json:"LineWidth,omitempty"`
}

type GeospatialMapAggregatedFieldWells struct {
	Colors []DimensionField `json:"Colors,omitempty"`
	Geospatial []DimensionField `json:"Geospatial,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type GeospatialMapConfiguration struct {
	FieldWells *GeospatialMapFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	MapStyleOptions *GeospatialMapStyleOptions `json:"MapStyleOptions,omitempty"`
	PointStyleOptions *GeospatialPointStyleOptions `json:"PointStyleOptions,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
	WindowOptions *GeospatialWindowOptions `json:"WindowOptions,omitempty"`
}

type GeospatialMapFieldWells struct {
	GeospatialMapAggregatedFieldWells *GeospatialMapAggregatedFieldWells `json:"GeospatialMapAggregatedFieldWells,omitempty"`
}

type GeospatialMapState struct {
	Bounds *GeospatialCoordinateBounds `json:"Bounds,omitempty"`
	MapNavigation *string `json:"MapNavigation,omitempty"`
}

type GeospatialMapStyle struct {
	BackgroundColor *string `json:"BackgroundColor,omitempty"`
	BaseMapStyle *string `json:"BaseMapStyle,omitempty"`
	BaseMapVisibility *string `json:"BaseMapVisibility,omitempty"`
}

type GeospatialMapStyleOptions struct {
	BaseMapStyle *string `json:"BaseMapStyle,omitempty"`
}

type GeospatialMapVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *GeospatialMapConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	GeocodingPreferences []GeocodePreference `json:"GeocodingPreferences,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type GeospatialNullDataSettings struct {
	SymbolStyle GeospatialNullSymbolStyle `json:"SymbolStyle,omitempty"`
}

type GeospatialNullSymbolStyle struct {
	FillColor *string `json:"FillColor,omitempty"`
	StrokeColor *string `json:"StrokeColor,omitempty"`
	StrokeWidth float64 `json:"StrokeWidth,omitempty"`
}

type GeospatialPointLayer struct {
	Style GeospatialPointStyle `json:"Style,omitempty"`
}

type GeospatialPointStyle struct {
	CircleSymbolStyle *GeospatialCircleSymbolStyle `json:"CircleSymbolStyle,omitempty"`
}

type GeospatialPointStyleOptions struct {
	ClusterMarkerConfiguration *ClusterMarkerConfiguration `json:"ClusterMarkerConfiguration,omitempty"`
	HeatmapConfiguration *GeospatialHeatmapConfiguration `json:"HeatmapConfiguration,omitempty"`
	SelectedPointStyle *string `json:"SelectedPointStyle,omitempty"`
}

type GeospatialPolygonLayer struct {
	Style GeospatialPolygonStyle `json:"Style,omitempty"`
}

type GeospatialPolygonStyle struct {
	PolygonSymbolStyle *GeospatialPolygonSymbolStyle `json:"PolygonSymbolStyle,omitempty"`
}

type GeospatialPolygonSymbolStyle struct {
	FillColor *GeospatialColor `json:"FillColor,omitempty"`
	StrokeColor *GeospatialColor `json:"StrokeColor,omitempty"`
	StrokeWidth *GeospatialLineWidth `json:"StrokeWidth,omitempty"`
}

type GeospatialSolidColor struct {
	Color string `json:"Color,omitempty"`
	State *string `json:"State,omitempty"`
}

type GeospatialStaticFileSource struct {
	StaticFileId string `json:"StaticFileId,omitempty"`
}

type GeospatialWindowOptions struct {
	Bounds *GeospatialCoordinateBounds `json:"Bounds,omitempty"`
	MapZoomMode *string `json:"MapZoomMode,omitempty"`
}

type GetDashboardEmbedUrlRequest struct {
	AdditionalDashboardIds []string `json:"additional-dashboard-ids,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	IdentityType string `json:"creds-type,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
	ResetDisabled bool `json:"reset-disabled,omitempty"`
	SessionLifetimeInMinutes int64 `json:"session-lifetime,omitempty"`
	StatePersistenceEnabled bool `json:"state-persistence-enabled,omitempty"`
	UndoRedoDisabled bool `json:"undo-redo-disabled,omitempty"`
	UserArn *string `json:"user-arn,omitempty"`
}

type GetDashboardEmbedUrlResponse struct {
	EmbedUrl *string `json:"EmbedUrl,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GetFlowMetadataInput struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
}

type GetFlowMetadataOutput struct {
	Arn string `json:"Arn,omitempty"`
	CreatedTime time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name string `json:"Name,omitempty"`
	PublishState *string `json:"PublishState,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	RunCount int `json:"RunCount,omitempty"`
	Status int `json:"Status,omitempty"`
	UserCount int `json:"UserCount,omitempty"`
}

type GetFlowPermissionsInput struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
}

type GetFlowPermissionsOutput struct {
	Arn string `json:"Arn,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
	Permissions []Permission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GetIdentityContextRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"Namespace,omitempty"`
	SessionExpiresAt *time.Time `json:"SessionExpiresAt,omitempty"`
	UserIdentifier UserIdentifier `json:"UserIdentifier,omitempty"`
}

type GetIdentityContextResponse struct {
	Context *string `json:"Context,omitempty"`
	RequestId string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GetSessionEmbedUrlRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	EntryPoint *string `json:"entry-point,omitempty"`
	SessionLifetimeInMinutes int64 `json:"session-lifetime,omitempty"`
	UserArn *string `json:"user-arn,omitempty"`
}

type GetSessionEmbedUrlResponse struct {
	EmbedUrl *string `json:"EmbedUrl,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type GlobalTableBorderOptions struct {
	SideSpecificBorder *TableSideBorderOptions `json:"SideSpecificBorder,omitempty"`
	UniformBorder *TableBorderOptions `json:"UniformBorder,omitempty"`
}

type GradientColor struct {
	Stops []GradientStop `json:"Stops,omitempty"`
}

type GradientStop struct {
	Color *string `json:"Color,omitempty"`
	DataValue float64 `json:"DataValue,omitempty"`
	GradientOffset float64 `json:"GradientOffset,omitempty"`
}

type GridLayoutCanvasSizeOptions struct {
	ScreenCanvasSizeOptions *GridLayoutScreenCanvasSizeOptions `json:"ScreenCanvasSizeOptions,omitempty"`
}

type GridLayoutConfiguration struct {
	CanvasSizeOptions *GridLayoutCanvasSizeOptions `json:"CanvasSizeOptions,omitempty"`
	Elements []GridLayoutElement `json:"Elements,omitempty"`
}

type GridLayoutElement struct {
	BackgroundStyle *GridLayoutElementBackgroundStyle `json:"BackgroundStyle,omitempty"`
	BorderRadius *string `json:"BorderRadius,omitempty"`
	BorderStyle *GridLayoutElementBorderStyle `json:"BorderStyle,omitempty"`
	ColumnIndex int `json:"ColumnIndex,omitempty"`
	ColumnSpan int `json:"ColumnSpan,omitempty"`
	ElementId string `json:"ElementId,omitempty"`
	ElementType string `json:"ElementType,omitempty"`
	LoadingAnimation *LoadingAnimation `json:"LoadingAnimation,omitempty"`
	Padding *string `json:"Padding,omitempty"`
	RowIndex int `json:"RowIndex,omitempty"`
	RowSpan int `json:"RowSpan,omitempty"`
	SelectedBorderStyle *GridLayoutElementBorderStyle `json:"SelectedBorderStyle,omitempty"`
}

type GridLayoutElementBackgroundStyle struct {
	Color *string `json:"Color,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type GridLayoutElementBorderStyle struct {
	Color *string `json:"Color,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
	Width *string `json:"Width,omitempty"`
}

type GridLayoutScreenCanvasSizeOptions struct {
	OptimizedViewPortWidth *string `json:"OptimizedViewPortWidth,omitempty"`
	ResizeOption string `json:"ResizeOption,omitempty"`
}

type Group struct {
	Arn *string `json:"Arn,omitempty"`
	Description *string `json:"Description,omitempty"`
	GroupName *string `json:"GroupName,omitempty"`
	PrincipalId *string `json:"PrincipalId,omitempty"`
}

type GroupMember struct {
	Arn *string `json:"Arn,omitempty"`
	MemberName *string `json:"MemberName,omitempty"`
}

type GroupSearchFilter struct {
	Name string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value string `json:"Value,omitempty"`
}

type GrowthRateComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
	PeriodSize int `json:"PeriodSize,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type GutterStyle struct {
	Show bool `json:"Show,omitempty"`
}

type HeaderFooterSectionConfiguration struct {
	Layout SectionLayoutConfiguration `json:"Layout,omitempty"`
	SectionId string `json:"SectionId,omitempty"`
	Style *SectionStyle `json:"Style,omitempty"`
}

type HeatMapAggregatedFieldWells struct {
	Columns []DimensionField `json:"Columns,omitempty"`
	Rows []DimensionField `json:"Rows,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type HeatMapConfiguration struct {
	ColorScale *ColorScale `json:"ColorScale,omitempty"`
	ColumnAxisDisplayOptions *AxisDisplayOptions `json:"ColumnAxisDisplayOptions,omitempty"`
	ColumnLabelOptions *ChartAxisLabelOptions `json:"ColumnLabelOptions,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *HeatMapFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	RowAxisDisplayOptions *AxisDisplayOptions `json:"RowAxisDisplayOptions,omitempty"`
	RowLabelOptions *ChartAxisLabelOptions `json:"RowLabelOptions,omitempty"`
	SortConfiguration *HeatMapSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
}

type HeatMapFieldWells struct {
	HeatMapAggregatedFieldWells *HeatMapAggregatedFieldWells `json:"HeatMapAggregatedFieldWells,omitempty"`
}

type HeatMapSortConfiguration struct {
	HeatMapColumnItemsLimitConfiguration *ItemsLimitConfiguration `json:"HeatMapColumnItemsLimitConfiguration,omitempty"`
	HeatMapColumnSort []FieldSortOptions `json:"HeatMapColumnSort,omitempty"`
	HeatMapRowItemsLimitConfiguration *ItemsLimitConfiguration `json:"HeatMapRowItemsLimitConfiguration,omitempty"`
	HeatMapRowSort []FieldSortOptions `json:"HeatMapRowSort,omitempty"`
}

type HeatMapVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *HeatMapConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type HistogramAggregatedFieldWells struct {
	Values []MeasureField `json:"Values,omitempty"`
}

type HistogramBinOptions struct {
	BinCount *BinCountOptions `json:"BinCount,omitempty"`
	BinWidth *BinWidthOptions `json:"BinWidth,omitempty"`
	SelectedBinType *string `json:"SelectedBinType,omitempty"`
	StartValue float64 `json:"StartValue,omitempty"`
}

type HistogramConfiguration struct {
	BinOptions *HistogramBinOptions `json:"BinOptions,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *HistogramFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
	XAxisDisplayOptions *AxisDisplayOptions `json:"XAxisDisplayOptions,omitempty"`
	XAxisLabelOptions *ChartAxisLabelOptions `json:"XAxisLabelOptions,omitempty"`
	YAxisDisplayOptions *AxisDisplayOptions `json:"YAxisDisplayOptions,omitempty"`
}

type HistogramFieldWells struct {
	HistogramAggregatedFieldWells *HistogramAggregatedFieldWells `json:"HistogramAggregatedFieldWells,omitempty"`
}

type HistogramVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *HistogramConfiguration `json:"ChartConfiguration,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type IAMConnectionMetadata struct {
	RoleArn string `json:"RoleArn,omitempty"`
}

type IAMPolicyAssignment struct {
	AssignmentId *string `json:"AssignmentId,omitempty"`
	AssignmentName *string `json:"AssignmentName,omitempty"`
	AssignmentStatus *string `json:"AssignmentStatus,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	Identities map[string][]string `json:"Identities,omitempty"`
	PolicyArn *string `json:"PolicyArn,omitempty"`
}

type IAMPolicyAssignmentSummary struct {
	AssignmentName *string `json:"AssignmentName,omitempty"`
	AssignmentStatus *string `json:"AssignmentStatus,omitempty"`
}

type Identifier struct {
	Identity string `json:"Identity,omitempty"`
}

type IdentityCenterConfiguration struct {
	EnableIdentityPropagation bool `json:"EnableIdentityPropagation,omitempty"`
}

type Image struct {
	GeneratedImageUrl *string `json:"GeneratedImageUrl,omitempty"`
	Source *ImageSource `json:"Source,omitempty"`
}

type ImageConfiguration struct {
	Source *ImageSource `json:"Source,omitempty"`
}

type ImageCustomAction struct {
	ActionOperations []ImageCustomActionOperation `json:"ActionOperations,omitempty"`
	CustomActionId string `json:"CustomActionId,omitempty"`
	Name string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
	Trigger string `json:"Trigger,omitempty"`
}

type ImageCustomActionOperation struct {
	NavigationOperation *CustomActionNavigationOperation `json:"NavigationOperation,omitempty"`
	SetParametersOperation *CustomActionSetParametersOperation `json:"SetParametersOperation,omitempty"`
	URLOperation *CustomActionURLOperation `json:"URLOperation,omitempty"`
}

type ImageInteractionOptions struct {
	ImageMenuOption *ImageMenuOption `json:"ImageMenuOption,omitempty"`
}

type ImageMenuOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type ImageSet struct {
	Height32 *Image `json:"Height32,omitempty"`
	Height64 *Image `json:"Height64,omitempty"`
	Original Image `json:"Original,omitempty"`
}

type ImageSetConfiguration struct {
	Original ImageConfiguration `json:"Original,omitempty"`
}

type ImageSource struct {
	PublicUrl *string `json:"PublicUrl,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
}

type ImageStaticFile struct {
	Source *StaticFileSource `json:"Source,omitempty"`
	StaticFileId string `json:"StaticFileId,omitempty"`
}

type ImpalaParameters struct {
	Database *string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
	SqlEndpointPath string `json:"SqlEndpointPath,omitempty"`
}

type ImportTableOperation struct {
	Alias string `json:"Alias,omitempty"`
	Source ImportTableOperationSource `json:"Source,omitempty"`
}

type ImportTableOperationSource struct {
	ColumnIdMappings []DataSetColumnIdMapping `json:"ColumnIdMappings,omitempty"`
	SourceTableId string `json:"SourceTableId,omitempty"`
}

type IncrementalRefresh struct {
	LookbackWindow LookbackWindow `json:"LookbackWindow,omitempty"`
}

type Ingestion struct {
	Arn string `json:"Arn,omitempty"`
	CreatedTime time.Time `json:"CreatedTime,omitempty"`
	ErrorInfo *ErrorInfo `json:"ErrorInfo,omitempty"`
	IngestionId *string `json:"IngestionId,omitempty"`
	IngestionSizeInBytes int64 `json:"IngestionSizeInBytes,omitempty"`
	IngestionStatus string `json:"IngestionStatus,omitempty"`
	IngestionTimeInSeconds int64 `json:"IngestionTimeInSeconds,omitempty"`
	QueueInfo *QueueInfo `json:"QueueInfo,omitempty"`
	RequestSource *string `json:"RequestSource,omitempty"`
	RequestType *string `json:"RequestType,omitempty"`
	RowInfo *RowInfo `json:"RowInfo,omitempty"`
}

type InnerFilter struct {
	CategoryInnerFilter *CategoryInnerFilter `json:"CategoryInnerFilter,omitempty"`
}

type InputColumn struct {
	Id *string `json:"Id,omitempty"`
	Name string `json:"Name,omitempty"`
	SubType *string `json:"SubType,omitempty"`
	Type string `json:"Type,omitempty"`
}

type InsightConfiguration struct {
	Computations []Computation `json:"Computations,omitempty"`
	CustomNarrative *CustomNarrativeOptions `json:"CustomNarrative,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
}

type InsightVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
	InsightConfiguration *InsightConfiguration `json:"InsightConfiguration,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type IntegerDatasetParameter struct {
	DefaultValues *IntegerDatasetParameterDefaultValues `json:"DefaultValues,omitempty"`
	Id string `json:"Id,omitempty"`
	Name string `json:"Name,omitempty"`
	ValueType string `json:"ValueType,omitempty"`
}

type IntegerDatasetParameterDefaultValues struct {
	StaticValues []int64 `json:"StaticValues,omitempty"`
}

type IntegerDefaultValues struct {
	DynamicValue *DynamicDefaultValue `json:"DynamicValue,omitempty"`
	StaticValues []int64 `json:"StaticValues,omitempty"`
}

type IntegerParameter struct {
	Name string `json:"Name,omitempty"`
	Values []int64 `json:"Values,omitempty"`
}

type IntegerParameterDeclaration struct {
	DefaultValues *IntegerDefaultValues `json:"DefaultValues,omitempty"`
	MappedDataSetParameters []MappedDataSetParameter `json:"MappedDataSetParameters,omitempty"`
	Name string `json:"Name,omitempty"`
	ParameterValueType string `json:"ParameterValueType,omitempty"`
	ValueWhenUnset *IntegerValueWhenUnsetConfiguration `json:"ValueWhenUnset,omitempty"`
}

type IntegerValueWhenUnsetConfiguration struct {
	CustomValue int64 `json:"CustomValue,omitempty"`
	ValueWhenUnsetOption *string `json:"ValueWhenUnsetOption,omitempty"`
}

type InvalidTopicReviewedAnswer struct {
	AnswerId *string `json:"AnswerId,omitempty"`
	Error *string `json:"Error,omitempty"`
}

type ItemsLimitConfiguration struct {
	ItemsLimit int64 `json:"ItemsLimit,omitempty"`
	OtherCategories *string `json:"OtherCategories,omitempty"`
}

type JiraParameters struct {
	SiteBaseUrl string `json:"SiteBaseUrl,omitempty"`
}

type JoinInstruction struct {
	LeftJoinKeyProperties *JoinKeyProperties `json:"LeftJoinKeyProperties,omitempty"`
	LeftOperand string `json:"LeftOperand,omitempty"`
	OnClause string `json:"OnClause,omitempty"`
	RightJoinKeyProperties *JoinKeyProperties `json:"RightJoinKeyProperties,omitempty"`
	RightOperand string `json:"RightOperand,omitempty"`
	Type string `json:"Type,omitempty"`
}

type JoinKeyProperties struct {
	UniqueKey bool `json:"UniqueKey,omitempty"`
}

type JoinOperandProperties struct {
	OutputColumnNameOverrides []OutputColumnNameOverride `json:"OutputColumnNameOverrides,omitempty"`
}

type JoinOperation struct {
	Alias string `json:"Alias,omitempty"`
	LeftOperand TransformOperationSource `json:"LeftOperand,omitempty"`
	LeftOperandProperties *JoinOperandProperties `json:"LeftOperandProperties,omitempty"`
	OnClause string `json:"OnClause,omitempty"`
	RightOperand TransformOperationSource `json:"RightOperand,omitempty"`
	RightOperandProperties *JoinOperandProperties `json:"RightOperandProperties,omitempty"`
	Type string `json:"Type,omitempty"`
}

type KPIActualValueConditionalFormatting struct {
	Icon *ConditionalFormattingIcon `json:"Icon,omitempty"`
	TextColor *ConditionalFormattingColor `json:"TextColor,omitempty"`
}

type KPIComparisonValueConditionalFormatting struct {
	Icon *ConditionalFormattingIcon `json:"Icon,omitempty"`
	TextColor *ConditionalFormattingColor `json:"TextColor,omitempty"`
}

type KPIConditionalFormatting struct {
	ConditionalFormattingOptions []KPIConditionalFormattingOption `json:"ConditionalFormattingOptions,omitempty"`
}

type KPIConditionalFormattingOption struct {
	ActualValue *KPIActualValueConditionalFormatting `json:"ActualValue,omitempty"`
	ComparisonValue *KPIComparisonValueConditionalFormatting `json:"ComparisonValue,omitempty"`
	PrimaryValue *KPIPrimaryValueConditionalFormatting `json:"PrimaryValue,omitempty"`
	ProgressBar *KPIProgressBarConditionalFormatting `json:"ProgressBar,omitempty"`
}

type KPIConfiguration struct {
	FieldWells *KPIFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	KPIOptions *KPIOptions `json:"KPIOptions,omitempty"`
	SortConfiguration *KPISortConfiguration `json:"SortConfiguration,omitempty"`
}

type KPIFieldWells struct {
	TargetValues []MeasureField `json:"TargetValues,omitempty"`
	TrendGroups []DimensionField `json:"TrendGroups,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type KPIOptions struct {
	Comparison *ComparisonConfiguration `json:"Comparison,omitempty"`
	PrimaryValueDisplayType *string `json:"PrimaryValueDisplayType,omitempty"`
	PrimaryValueFontConfiguration *FontConfiguration `json:"PrimaryValueFontConfiguration,omitempty"`
	ProgressBar *ProgressBarOptions `json:"ProgressBar,omitempty"`
	SecondaryValue *SecondaryValueOptions `json:"SecondaryValue,omitempty"`
	SecondaryValueFontConfiguration *FontConfiguration `json:"SecondaryValueFontConfiguration,omitempty"`
	Sparkline *KPISparklineOptions `json:"Sparkline,omitempty"`
	TrendArrows *TrendArrowOptions `json:"TrendArrows,omitempty"`
	VisualLayoutOptions *KPIVisualLayoutOptions `json:"VisualLayoutOptions,omitempty"`
}

type KPIPrimaryValueConditionalFormatting struct {
	Icon *ConditionalFormattingIcon `json:"Icon,omitempty"`
	TextColor *ConditionalFormattingColor `json:"TextColor,omitempty"`
}

type KPIProgressBarConditionalFormatting struct {
	ForegroundColor *ConditionalFormattingColor `json:"ForegroundColor,omitempty"`
}

type KPISortConfiguration struct {
	TrendGroupSort []FieldSortOptions `json:"TrendGroupSort,omitempty"`
}

type KPISparklineOptions struct {
	Color *string `json:"Color,omitempty"`
	TooltipVisibility *string `json:"TooltipVisibility,omitempty"`
	Type string `json:"Type,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type KPIVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *KPIConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	ConditionalFormatting *KPIConditionalFormatting `json:"ConditionalFormatting,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type KPIVisualLayoutOptions struct {
	StandardLayout *KPIVisualStandardLayout `json:"StandardLayout,omitempty"`
}

type KPIVisualStandardLayout struct {
	Type string `json:"Type,omitempty"`
}

type KeyPairCredentials struct {
	KeyPairUsername string `json:"KeyPairUsername,omitempty"`
	PrivateKey string `json:"PrivateKey,omitempty"`
	PrivateKeyPassphrase *string `json:"PrivateKeyPassphrase,omitempty"`
}

type LabelOptions struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type LayerCustomAction struct {
	ActionOperations []LayerCustomActionOperation `json:"ActionOperations,omitempty"`
	CustomActionId string `json:"CustomActionId,omitempty"`
	Name string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
	Trigger string `json:"Trigger,omitempty"`
}

type LayerCustomActionOperation struct {
	FilterOperation *CustomActionFilterOperation `json:"FilterOperation,omitempty"`
	NavigationOperation *CustomActionNavigationOperation `json:"NavigationOperation,omitempty"`
	SetParametersOperation *CustomActionSetParametersOperation `json:"SetParametersOperation,omitempty"`
	URLOperation *CustomActionURLOperation `json:"URLOperation,omitempty"`
}

type LayerMapVisual struct {
	ChartConfiguration *GeospatialLayerMapConfiguration `json:"ChartConfiguration,omitempty"`
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type Layout struct {
	Configuration LayoutConfiguration `json:"Configuration,omitempty"`
}

type LayoutConfiguration struct {
	FreeFormLayout *FreeFormLayoutConfiguration `json:"FreeFormLayout,omitempty"`
	GridLayout *GridLayoutConfiguration `json:"GridLayout,omitempty"`
	SectionBasedLayout *SectionBasedLayoutConfiguration `json:"SectionBasedLayout,omitempty"`
}

type LegendOptions struct {
	Height *string `json:"Height,omitempty"`
	Position *string `json:"Position,omitempty"`
	Title *LabelOptions `json:"Title,omitempty"`
	ValueFontConfiguration *FontConfiguration `json:"ValueFontConfiguration,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
	Width *string `json:"Width,omitempty"`
}

type LineChartAggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	Colors []DimensionField `json:"Colors,omitempty"`
	SmallMultiples []DimensionField `json:"SmallMultiples,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type LineChartConfiguration struct {
	ContributionAnalysisDefaults []ContributionAnalysisDefault `json:"ContributionAnalysisDefaults,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	DefaultSeriesSettings *LineChartDefaultSeriesSettings `json:"DefaultSeriesSettings,omitempty"`
	FieldWells *LineChartFieldWells `json:"FieldWells,omitempty"`
	ForecastConfigurations []ForecastConfiguration `json:"ForecastConfigurations,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	PrimaryYAxisDisplayOptions *LineSeriesAxisDisplayOptions `json:"PrimaryYAxisDisplayOptions,omitempty"`
	PrimaryYAxisLabelOptions *ChartAxisLabelOptions `json:"PrimaryYAxisLabelOptions,omitempty"`
	ReferenceLines []ReferenceLine `json:"ReferenceLines,omitempty"`
	SecondaryYAxisDisplayOptions *LineSeriesAxisDisplayOptions `json:"SecondaryYAxisDisplayOptions,omitempty"`
	SecondaryYAxisLabelOptions *ChartAxisLabelOptions `json:"SecondaryYAxisLabelOptions,omitempty"`
	Series []SeriesItem `json:"Series,omitempty"`
	SingleAxisOptions *SingleAxisOptions `json:"SingleAxisOptions,omitempty"`
	SmallMultiplesOptions *SmallMultiplesOptions `json:"SmallMultiplesOptions,omitempty"`
	SortConfiguration *LineChartSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	Type *string `json:"Type,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
	XAxisDisplayOptions *AxisDisplayOptions `json:"XAxisDisplayOptions,omitempty"`
	XAxisLabelOptions *ChartAxisLabelOptions `json:"XAxisLabelOptions,omitempty"`
}

type LineChartDefaultSeriesSettings struct {
	AxisBinding *string `json:"AxisBinding,omitempty"`
	DecalSettings *DecalSettings `json:"DecalSettings,omitempty"`
	LineStyleSettings *LineChartLineStyleSettings `json:"LineStyleSettings,omitempty"`
	MarkerStyleSettings *LineChartMarkerStyleSettings `json:"MarkerStyleSettings,omitempty"`
}

type LineChartFieldWells struct {
	LineChartAggregatedFieldWells *LineChartAggregatedFieldWells `json:"LineChartAggregatedFieldWells,omitempty"`
}

type LineChartLineStyleSettings struct {
	LineInterpolation *string `json:"LineInterpolation,omitempty"`
	LineStyle *string `json:"LineStyle,omitempty"`
	LineVisibility *string `json:"LineVisibility,omitempty"`
	LineWidth *string `json:"LineWidth,omitempty"`
}

type LineChartMarkerStyleSettings struct {
	MarkerColor *string `json:"MarkerColor,omitempty"`
	MarkerShape *string `json:"MarkerShape,omitempty"`
	MarkerSize *string `json:"MarkerSize,omitempty"`
	MarkerVisibility *string `json:"MarkerVisibility,omitempty"`
}

type LineChartSeriesSettings struct {
	DecalSettings *DecalSettings `json:"DecalSettings,omitempty"`
	LineStyleSettings *LineChartLineStyleSettings `json:"LineStyleSettings,omitempty"`
	MarkerStyleSettings *LineChartMarkerStyleSettings `json:"MarkerStyleSettings,omitempty"`
}

type LineChartSortConfiguration struct {
	CategoryItemsLimitConfiguration *ItemsLimitConfiguration `json:"CategoryItemsLimitConfiguration,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
	ColorItemsLimitConfiguration *ItemsLimitConfiguration `json:"ColorItemsLimitConfiguration,omitempty"`
	SmallMultiplesLimitConfiguration *ItemsLimitConfiguration `json:"SmallMultiplesLimitConfiguration,omitempty"`
	SmallMultiplesSort []FieldSortOptions `json:"SmallMultiplesSort,omitempty"`
}

type LineChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *LineChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type LineSeriesAxisDisplayOptions struct {
	AxisOptions *AxisDisplayOptions `json:"AxisOptions,omitempty"`
	MissingDataConfigurations []MissingDataConfiguration `json:"MissingDataConfigurations,omitempty"`
}

type LinkSharingConfiguration struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
}

type ListActionConnectorsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListActionConnectorsResponse struct {
	ActionConnectorSummaries []ActionConnectorSummary `json:"ActionConnectorSummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListAnalysesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListAnalysesResponse struct {
	AnalysisSummaryList []AnalysisSummary `json:"AnalysisSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListAssetBundleExportJobsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListAssetBundleExportJobsResponse struct {
	AssetBundleExportJobSummaryList []AssetBundleExportJobSummary `json:"AssetBundleExportJobSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListAssetBundleImportJobsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListAssetBundleImportJobsResponse struct {
	AssetBundleImportJobSummaryList []AssetBundleImportJobSummary `json:"AssetBundleImportJobSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListBrandsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListBrandsResponse struct {
	Brands []BrandSummary `json:"Brands,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListControlDisplayOptions struct {
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	SearchOptions *ListControlSearchOptions `json:"SearchOptions,omitempty"`
	SelectAllOptions *ListControlSelectAllOptions `json:"SelectAllOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type ListControlSearchOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type ListControlSelectAllOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type ListCustomPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListCustomPermissionsResponse struct {
	CustomPermissionsList []CustomPermissions `json:"CustomPermissionsList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListDashboardVersionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListDashboardVersionsResponse struct {
	DashboardVersionSummaryList []DashboardVersionSummary `json:"DashboardVersionSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListDashboardsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListDashboardsResponse struct {
	DashboardSummaryList []DashboardSummary `json:"DashboardSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListDataSetsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListDataSetsResponse struct {
	DataSetSummaries []DataSetSummary `json:"DataSetSummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListDataSourcesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListDataSourcesResponse struct {
	DataSources []DataSource `json:"DataSources,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListFlowsInput struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListFlowsOutput struct {
	FlowSummaryList []FlowSummary `json:"FlowSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListFolderMembersRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListFolderMembersResponse struct {
	FolderMemberList []MemberIdArnPair `json:"FolderMemberList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListFoldersForResourceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListFoldersForResourceResponse struct {
	Folders []string `json:"Folders,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListFoldersRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListFoldersResponse struct {
	FolderSummaryList []FolderSummary `json:"FolderSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListGroupMembershipsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListGroupMembershipsResponse struct {
	GroupMemberList []GroupMember `json:"GroupMemberList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListGroupsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListGroupsResponse struct {
	GroupList []Group `json:"GroupList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListIAMPolicyAssignmentsForUserRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type ListIAMPolicyAssignmentsForUserResponse struct {
	ActiveAssignments []ActiveIAMPolicyAssignment `json:"ActiveAssignments,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListIAMPolicyAssignmentsRequest struct {
	AssignmentStatus *string `json:"assignment-status,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListIAMPolicyAssignmentsResponse struct {
	IAMPolicyAssignments []IAMPolicyAssignmentSummary `json:"IAMPolicyAssignments,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListIdentityPropagationConfigsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListIdentityPropagationConfigsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Services []AuthorizedTargetsByService `json:"Services,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListIngestionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListIngestionsResponse struct {
	Ingestions []Ingestion `json:"Ingestions,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListNamespacesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListNamespacesResponse struct {
	Namespaces []NamespaceInfoV2 `json:"Namespaces,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListRefreshSchedulesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
}

type ListRefreshSchedulesResponse struct {
	RefreshSchedules []RefreshSchedule `json:"RefreshSchedules,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListRoleMembershipsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	Role string `json:"Role,omitempty"`
}

type ListRoleMembershipsResponse struct {
	MembersList []string `json:"MembersList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListSelfUpgradesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListSelfUpgradesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	SelfUpgradeRequestDetails []SelfUpgradeRequestDetail `json:"SelfUpgradeRequestDetails,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type ListTemplateAliasesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-result,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
}

type ListTemplateAliasesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateAliasList []TemplateAlias `json:"TemplateAliasList,omitempty"`
}

type ListTemplateVersionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
}

type ListTemplateVersionsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateVersionSummaryList []TemplateVersionSummary `json:"TemplateVersionSummaryList,omitempty"`
}

type ListTemplatesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-result,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListTemplatesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateSummaryList []TemplateSummary `json:"TemplateSummaryList,omitempty"`
}

type ListThemeAliasesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-result,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type ListThemeAliasesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeAliasList []ThemeAlias `json:"ThemeAliasList,omitempty"`
}

type ListThemeVersionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type ListThemeVersionsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeVersionSummaryList []ThemeVersionSummary `json:"ThemeVersionSummaryList,omitempty"`
}

type ListThemesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	Type *string `json:"type,omitempty"`
}

type ListThemesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeSummaryList []ThemeSummary `json:"ThemeSummaryList,omitempty"`
}

type ListTopicRefreshSchedulesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type ListTopicRefreshSchedulesResponse struct {
	RefreshSchedules []TopicRefreshScheduleSummary `json:"RefreshSchedules,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type ListTopicReviewedAnswersRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type ListTopicReviewedAnswersResponse struct {
	Answers []TopicReviewedAnswer `json:"Answers,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type ListTopicsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListTopicsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicsSummaries []TopicSummary `json:"TopicsSummaries,omitempty"`
}

type ListUserGroupsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type ListUserGroupsResponse struct {
	GroupList []Group `json:"GroupList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type ListUsersRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListUsersResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	UserList []User `json:"UserList,omitempty"`
}

type ListVPCConnectionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type ListVPCConnectionsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VPCConnectionSummaries []VPCConnectionSummary `json:"VPCConnectionSummaries,omitempty"`
}

type LoadingAnimation struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type LocalNavigationConfiguration struct {
	TargetSheetId string `json:"TargetSheetId,omitempty"`
}

type LogicalTable struct {
	Alias string `json:"Alias,omitempty"`
	DataTransforms []TransformOperation `json:"DataTransforms,omitempty"`
	Source LogicalTableSource `json:"Source,omitempty"`
}

type LogicalTableSource struct {
	DataSetArn *string `json:"DataSetArn,omitempty"`
	JoinInstruction *JoinInstruction `json:"JoinInstruction,omitempty"`
	PhysicalTableId *string `json:"PhysicalTableId,omitempty"`
}

type Logo struct {
	AltText string `json:"AltText,omitempty"`
	LogoSet LogoSet `json:"LogoSet,omitempty"`
}

type LogoConfiguration struct {
	AltText string `json:"AltText,omitempty"`
	LogoSet LogoSetConfiguration `json:"LogoSet,omitempty"`
}

type LogoSet struct {
	Favicon *ImageSet `json:"Favicon,omitempty"`
	Primary ImageSet `json:"Primary,omitempty"`
}

type LogoSetConfiguration struct {
	Favicon *ImageSetConfiguration `json:"Favicon,omitempty"`
	Primary ImageSetConfiguration `json:"Primary,omitempty"`
}

type LongFormatText struct {
	PlainText *string `json:"PlainText,omitempty"`
	RichText *string `json:"RichText,omitempty"`
}

type LookbackWindow struct {
	ColumnName string `json:"ColumnName,omitempty"`
	Size int64 `json:"Size,omitempty"`
	SizeUnit string `json:"SizeUnit,omitempty"`
}

type ManifestFileLocation struct {
	Bucket string `json:"Bucket,omitempty"`
	Key string `json:"Key,omitempty"`
}

type MappedDataSetParameter struct {
	DataSetIdentifier string `json:"DataSetIdentifier,omitempty"`
	DataSetParameterName string `json:"DataSetParameterName,omitempty"`
}

type MarginStyle struct {
	Show bool `json:"Show,omitempty"`
}

type MariaDbParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type MaximumLabelType struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type MaximumMinimumComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
	Type string `json:"Type,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type MeasureField struct {
	CalculatedMeasureField *CalculatedMeasureField `json:"CalculatedMeasureField,omitempty"`
	CategoricalMeasureField *CategoricalMeasureField `json:"CategoricalMeasureField,omitempty"`
	DateMeasureField *DateMeasureField `json:"DateMeasureField,omitempty"`
	NumericalMeasureField *NumericalMeasureField `json:"NumericalMeasureField,omitempty"`
}

type MemberIdArnPair struct {
	MemberArn *string `json:"MemberArn,omitempty"`
	MemberId *string `json:"MemberId,omitempty"`
}

type MetricComparisonComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	FromValue *MeasureField `json:"FromValue,omitempty"`
	Name *string `json:"Name,omitempty"`
	TargetValue *MeasureField `json:"TargetValue,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
}

type MinimumLabelType struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type MissingDataConfiguration struct {
	TreatmentOption *string `json:"TreatmentOption,omitempty"`
}

type MySqlParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type NamedEntityDefinition struct {
	FieldName *string `json:"FieldName,omitempty"`
	Metric *NamedEntityDefinitionMetric `json:"Metric,omitempty"`
	PropertyName *string `json:"PropertyName,omitempty"`
	PropertyRole *string `json:"PropertyRole,omitempty"`
	PropertyUsage *string `json:"PropertyUsage,omitempty"`
}

type NamedEntityDefinitionMetric struct {
	Aggregation *string `json:"Aggregation,omitempty"`
	AggregationFunctionParameters map[string]string `json:"AggregationFunctionParameters,omitempty"`
}

type NamedEntityRef struct {
	NamedEntityName *string `json:"NamedEntityName,omitempty"`
}

type NamespaceError struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type NamespaceInfoV2 struct {
	Arn *string `json:"Arn,omitempty"`
	CapacityRegion *string `json:"CapacityRegion,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	IamIdentityCenterApplicationArn *string `json:"IamIdentityCenterApplicationArn,omitempty"`
	IamIdentityCenterInstanceArn *string `json:"IamIdentityCenterInstanceArn,omitempty"`
	IdentityStore *string `json:"IdentityStore,omitempty"`
	Name *string `json:"Name,omitempty"`
	NamespaceError *NamespaceError `json:"NamespaceError,omitempty"`
}

type NavbarStyle struct {
	ContextualNavbar *Palette `json:"ContextualNavbar,omitempty"`
	GlobalNavbar *Palette `json:"GlobalNavbar,omitempty"`
}

type NegativeFormat struct {
	Prefix *string `json:"Prefix,omitempty"`
	Suffix *string `json:"Suffix,omitempty"`
}

type NegativeValueConfiguration struct {
	DisplayMode string `json:"DisplayMode,omitempty"`
}

type NestedFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	IncludeInnerSet bool `json:"IncludeInnerSet,omitempty"`
	InnerFilter InnerFilter `json:"InnerFilter,omitempty"`
}

type NetworkInterface struct {
	AvailabilityZone *string `json:"AvailabilityZone,omitempty"`
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	Status *string `json:"Status,omitempty"`
	SubnetId *string `json:"SubnetId,omitempty"`
}

type NewDefaultValues struct {
	DateTimeStaticValues []time.Time `json:"DateTimeStaticValues,omitempty"`
	DecimalStaticValues []float64 `json:"DecimalStaticValues,omitempty"`
	IntegerStaticValues []int64 `json:"IntegerStaticValues,omitempty"`
	StringStaticValues []string `json:"StringStaticValues,omitempty"`
}

type NoneConnectionMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
}

type NullValueFormatConfiguration struct {
	NullString string `json:"NullString,omitempty"`
}

type NumberDisplayFormatConfiguration struct {
	DecimalPlacesConfiguration *DecimalPlacesConfiguration `json:"DecimalPlacesConfiguration,omitempty"`
	NegativeValueConfiguration *NegativeValueConfiguration `json:"NegativeValueConfiguration,omitempty"`
	NullValueFormatConfiguration *NullValueFormatConfiguration `json:"NullValueFormatConfiguration,omitempty"`
	NumberScale *string `json:"NumberScale,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
	SeparatorConfiguration *NumericSeparatorConfiguration `json:"SeparatorConfiguration,omitempty"`
	Suffix *string `json:"Suffix,omitempty"`
}

type NumberFormatConfiguration struct {
	FormatConfiguration *NumericFormatConfiguration `json:"FormatConfiguration,omitempty"`
}

type NumericAxisOptions struct {
	Range *AxisDisplayRange `json:"Range,omitempty"`
	Scale *AxisScale `json:"Scale,omitempty"`
}

type NumericEqualityDrillDownFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	Value float64 `json:"Value,omitempty"`
}

type NumericEqualityFilter struct {
	AggregationFunction *AggregationFunction `json:"AggregationFunction,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	MatchOperator string `json:"MatchOperator,omitempty"`
	NullOption string `json:"NullOption,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
	SelectAllOptions *string `json:"SelectAllOptions,omitempty"`
	Value float64 `json:"Value,omitempty"`
}

type NumericFormatConfiguration struct {
	CurrencyDisplayFormatConfiguration *CurrencyDisplayFormatConfiguration `json:"CurrencyDisplayFormatConfiguration,omitempty"`
	NumberDisplayFormatConfiguration *NumberDisplayFormatConfiguration `json:"NumberDisplayFormatConfiguration,omitempty"`
	PercentageDisplayFormatConfiguration *PercentageDisplayFormatConfiguration `json:"PercentageDisplayFormatConfiguration,omitempty"`
}

type NumericRangeFilter struct {
	AggregationFunction *AggregationFunction `json:"AggregationFunction,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	IncludeMaximum bool `json:"IncludeMaximum,omitempty"`
	IncludeMinimum bool `json:"IncludeMinimum,omitempty"`
	NullOption string `json:"NullOption,omitempty"`
	RangeMaximum *NumericRangeFilterValue `json:"RangeMaximum,omitempty"`
	RangeMinimum *NumericRangeFilterValue `json:"RangeMinimum,omitempty"`
	SelectAllOptions *string `json:"SelectAllOptions,omitempty"`
}

type NumericRangeFilterValue struct {
	Parameter *string `json:"Parameter,omitempty"`
	StaticValue float64 `json:"StaticValue,omitempty"`
}

type NumericSeparatorConfiguration struct {
	DecimalSeparator *string `json:"DecimalSeparator,omitempty"`
	ThousandsSeparator *ThousandSeparatorOptions `json:"ThousandsSeparator,omitempty"`
}

type NumericalAggregationFunction struct {
	PercentileAggregation *PercentileAggregation `json:"PercentileAggregation,omitempty"`
	SimpleNumericalAggregation *string `json:"SimpleNumericalAggregation,omitempty"`
}

type NumericalDimensionField struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *NumberFormatConfiguration `json:"FormatConfiguration,omitempty"`
	HierarchyId *string `json:"HierarchyId,omitempty"`
}

type NumericalMeasureField struct {
	AggregationFunction *NumericalAggregationFunction `json:"AggregationFunction,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *NumberFormatConfiguration `json:"FormatConfiguration,omitempty"`
}

type OAuthClientCredentials struct {
	ClientId *string `json:"ClientId,omitempty"`
	ClientSecret *string `json:"ClientSecret,omitempty"`
	Username *string `json:"Username,omitempty"`
}

type OAuthParameters struct {
	IdentityProviderResourceUri *string `json:"IdentityProviderResourceUri,omitempty"`
	IdentityProviderVpcConnectionProperties *VpcConnectionProperties `json:"IdentityProviderVpcConnectionProperties,omitempty"`
	OAuthScope *string `json:"OAuthScope,omitempty"`
	TokenProviderUrl string `json:"TokenProviderUrl,omitempty"`
}

type OracleParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
	UseServiceName bool `json:"UseServiceName,omitempty"`
}

type OutputColumn struct {
	Description *string `json:"Description,omitempty"`
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
	SubType *string `json:"SubType,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type OutputColumnNameOverride struct {
	OutputColumnName string `json:"OutputColumnName,omitempty"`
	SourceColumnName *string `json:"SourceColumnName,omitempty"`
}

type OverrideDatasetParameterOperation struct {
	NewDefaultValues *NewDefaultValues `json:"NewDefaultValues,omitempty"`
	NewParameterName *string `json:"NewParameterName,omitempty"`
	ParameterName string `json:"ParameterName,omitempty"`
}

type PaginationConfiguration struct {
	PageNumber int64 `json:"PageNumber,omitempty"`
	PageSize int64 `json:"PageSize,omitempty"`
}

type Palette struct {
	Background *string `json:"Background,omitempty"`
	Foreground *string `json:"Foreground,omitempty"`
}

type PanelConfiguration struct {
	BackgroundColor *string `json:"BackgroundColor,omitempty"`
	BackgroundVisibility *string `json:"BackgroundVisibility,omitempty"`
	BorderColor *string `json:"BorderColor,omitempty"`
	BorderStyle *string `json:"BorderStyle,omitempty"`
	BorderThickness *string `json:"BorderThickness,omitempty"`
	BorderVisibility *string `json:"BorderVisibility,omitempty"`
	GutterSpacing *string `json:"GutterSpacing,omitempty"`
	GutterVisibility *string `json:"GutterVisibility,omitempty"`
	Title *PanelTitleOptions `json:"Title,omitempty"`
}

type PanelTitleOptions struct {
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
	HorizontalTextAlignment *string `json:"HorizontalTextAlignment,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type ParameterControl struct {
	DateTimePicker *ParameterDateTimePickerControl `json:"DateTimePicker,omitempty"`
	Dropdown *ParameterDropDownControl `json:"Dropdown,omitempty"`
	List *ParameterListControl `json:"List,omitempty"`
	Slider *ParameterSliderControl `json:"Slider,omitempty"`
	TextArea *ParameterTextAreaControl `json:"TextArea,omitempty"`
	TextField *ParameterTextFieldControl `json:"TextField,omitempty"`
}

type ParameterDateTimePickerControl struct {
	DisplayOptions *DateTimePickerControlDisplayOptions `json:"DisplayOptions,omitempty"`
	ParameterControlId string `json:"ParameterControlId,omitempty"`
	SourceParameterName string `json:"SourceParameterName,omitempty"`
	Title string `json:"Title,omitempty"`
}

type ParameterDeclaration struct {
	DateTimeParameterDeclaration *DateTimeParameterDeclaration `json:"DateTimeParameterDeclaration,omitempty"`
	DecimalParameterDeclaration *DecimalParameterDeclaration `json:"DecimalParameterDeclaration,omitempty"`
	IntegerParameterDeclaration *IntegerParameterDeclaration `json:"IntegerParameterDeclaration,omitempty"`
	StringParameterDeclaration *StringParameterDeclaration `json:"StringParameterDeclaration,omitempty"`
}

type ParameterDropDownControl struct {
	CascadingControlConfiguration *CascadingControlConfiguration `json:"CascadingControlConfiguration,omitempty"`
	CommitMode *string `json:"CommitMode,omitempty"`
	DisplayOptions *DropDownControlDisplayOptions `json:"DisplayOptions,omitempty"`
	ParameterControlId string `json:"ParameterControlId,omitempty"`
	SelectableValues *ParameterSelectableValues `json:"SelectableValues,omitempty"`
	SourceParameterName string `json:"SourceParameterName,omitempty"`
	Title string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ParameterListControl struct {
	CascadingControlConfiguration *CascadingControlConfiguration `json:"CascadingControlConfiguration,omitempty"`
	DisplayOptions *ListControlDisplayOptions `json:"DisplayOptions,omitempty"`
	ParameterControlId string `json:"ParameterControlId,omitempty"`
	SelectableValues *ParameterSelectableValues `json:"SelectableValues,omitempty"`
	SourceParameterName string `json:"SourceParameterName,omitempty"`
	Title string `json:"Title,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ParameterSelectableValues struct {
	LinkToDataSetColumn *ColumnIdentifier `json:"LinkToDataSetColumn,omitempty"`
	Values []string `json:"Values,omitempty"`
}

type ParameterSliderControl struct {
	DisplayOptions *SliderControlDisplayOptions `json:"DisplayOptions,omitempty"`
	MaximumValue float64 `json:"MaximumValue,omitempty"`
	MinimumValue float64 `json:"MinimumValue,omitempty"`
	ParameterControlId string `json:"ParameterControlId,omitempty"`
	SourceParameterName string `json:"SourceParameterName,omitempty"`
	StepSize float64 `json:"StepSize,omitempty"`
	Title string `json:"Title,omitempty"`
}

type ParameterTextAreaControl struct {
	Delimiter *string `json:"Delimiter,omitempty"`
	DisplayOptions *TextAreaControlDisplayOptions `json:"DisplayOptions,omitempty"`
	ParameterControlId string `json:"ParameterControlId,omitempty"`
	SourceParameterName string `json:"SourceParameterName,omitempty"`
	Title string `json:"Title,omitempty"`
}

type ParameterTextFieldControl struct {
	DisplayOptions *TextFieldControlDisplayOptions `json:"DisplayOptions,omitempty"`
	ParameterControlId string `json:"ParameterControlId,omitempty"`
	SourceParameterName string `json:"SourceParameterName,omitempty"`
	Title string `json:"Title,omitempty"`
}

type Parameters struct {
	DateTimeParameters []DateTimeParameter `json:"DateTimeParameters,omitempty"`
	DecimalParameters []DecimalParameter `json:"DecimalParameters,omitempty"`
	IntegerParameters []IntegerParameter `json:"IntegerParameters,omitempty"`
	StringParameters []StringParameter `json:"StringParameters,omitempty"`
}

type ParentDataSet struct {
	DataSetArn string `json:"DataSetArn,omitempty"`
	InputColumns []InputColumn `json:"InputColumns,omitempty"`
}

type PercentVisibleRange struct {
	From float64 `json:"From,omitempty"`
	To float64 `json:"To,omitempty"`
}

type PercentageDisplayFormatConfiguration struct {
	DecimalPlacesConfiguration *DecimalPlacesConfiguration `json:"DecimalPlacesConfiguration,omitempty"`
	NegativeValueConfiguration *NegativeValueConfiguration `json:"NegativeValueConfiguration,omitempty"`
	NullValueFormatConfiguration *NullValueFormatConfiguration `json:"NullValueFormatConfiguration,omitempty"`
	Prefix *string `json:"Prefix,omitempty"`
	SeparatorConfiguration *NumericSeparatorConfiguration `json:"SeparatorConfiguration,omitempty"`
	Suffix *string `json:"Suffix,omitempty"`
}

type PercentileAggregation struct {
	PercentileValue float64 `json:"PercentileValue,omitempty"`
}

type PerformanceConfiguration struct {
	UniqueKeys []UniqueKey `json:"UniqueKeys,omitempty"`
}

type PeriodOverPeriodComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type PeriodToDateComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
	PeriodTimeGranularity *string `json:"PeriodTimeGranularity,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type Permission struct {
	Actions []string `json:"Actions,omitempty"`
	Principal string `json:"Principal,omitempty"`
}

type PhysicalTable struct {
	CustomSql *CustomSql `json:"CustomSql,omitempty"`
	RelationalTable *RelationalTable `json:"RelationalTable,omitempty"`
	S3Source *S3Source `json:"S3Source,omitempty"`
	SaaSTable *SaaSTable `json:"SaaSTable,omitempty"`
}

type PieChartAggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	SmallMultiples []DimensionField `json:"SmallMultiples,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type PieChartConfiguration struct {
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	ContributionAnalysisDefaults []ContributionAnalysisDefault `json:"ContributionAnalysisDefaults,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	DonutOptions *DonutOptions `json:"DonutOptions,omitempty"`
	FieldWells *PieChartFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	SmallMultiplesOptions *SmallMultiplesOptions `json:"SmallMultiplesOptions,omitempty"`
	SortConfiguration *PieChartSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	ValueLabelOptions *ChartAxisLabelOptions `json:"ValueLabelOptions,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type PieChartFieldWells struct {
	PieChartAggregatedFieldWells *PieChartAggregatedFieldWells `json:"PieChartAggregatedFieldWells,omitempty"`
}

type PieChartSortConfiguration struct {
	CategoryItemsLimit *ItemsLimitConfiguration `json:"CategoryItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
	SmallMultiplesLimitConfiguration *ItemsLimitConfiguration `json:"SmallMultiplesLimitConfiguration,omitempty"`
	SmallMultiplesSort []FieldSortOptions `json:"SmallMultiplesSort,omitempty"`
}

type PieChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *PieChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type PivotConfiguration struct {
	LabelColumnName *string `json:"LabelColumnName,omitempty"`
	PivotedLabels []PivotedLabel `json:"PivotedLabels,omitempty"`
}

type PivotFieldSortOptions struct {
	FieldId string `json:"FieldId,omitempty"`
	SortBy PivotTableSortBy `json:"SortBy,omitempty"`
}

type PivotOperation struct {
	Alias string `json:"Alias,omitempty"`
	GroupByColumnNames []string `json:"GroupByColumnNames,omitempty"`
	PivotConfiguration PivotConfiguration `json:"PivotConfiguration,omitempty"`
	Source TransformOperationSource `json:"Source,omitempty"`
	ValueColumnConfiguration ValueColumnConfiguration `json:"ValueColumnConfiguration,omitempty"`
}

type PivotTableAggregatedFieldWells struct {
	Columns []DimensionField `json:"Columns,omitempty"`
	Rows []DimensionField `json:"Rows,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type PivotTableCellConditionalFormatting struct {
	FieldId string `json:"FieldId,omitempty"`
	Scope *PivotTableConditionalFormattingScope `json:"Scope,omitempty"`
	Scopes []PivotTableConditionalFormattingScope `json:"Scopes,omitempty"`
	TextFormat *TextConditionalFormat `json:"TextFormat,omitempty"`
}

type PivotTableConditionalFormatting struct {
	ConditionalFormattingOptions []PivotTableConditionalFormattingOption `json:"ConditionalFormattingOptions,omitempty"`
}

type PivotTableConditionalFormattingOption struct {
	Cell *PivotTableCellConditionalFormatting `json:"Cell,omitempty"`
}

type PivotTableConditionalFormattingScope struct {
	Role *string `json:"Role,omitempty"`
}

type PivotTableConfiguration struct {
	DashboardCustomizationVisualOptions *DashboardCustomizationVisualOptions `json:"DashboardCustomizationVisualOptions,omitempty"`
	FieldOptions *PivotTableFieldOptions `json:"FieldOptions,omitempty"`
	FieldWells *PivotTableFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	PaginatedReportOptions *PivotTablePaginatedReportOptions `json:"PaginatedReportOptions,omitempty"`
	SortConfiguration *PivotTableSortConfiguration `json:"SortConfiguration,omitempty"`
	TableOptions *PivotTableOptions `json:"TableOptions,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	TotalOptions *PivotTableTotalOptions `json:"TotalOptions,omitempty"`
}

type PivotTableDataPathOption struct {
	DataPathList []DataPathValue `json:"DataPathList,omitempty"`
	Width *string `json:"Width,omitempty"`
}

type PivotTableFieldCollapseStateOption struct {
	State *string `json:"State,omitempty"`
	Target PivotTableFieldCollapseStateTarget `json:"Target,omitempty"`
}

type PivotTableFieldCollapseStateTarget struct {
	FieldDataPathValues []DataPathValue `json:"FieldDataPathValues,omitempty"`
	FieldId *string `json:"FieldId,omitempty"`
}

type PivotTableFieldOption struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type PivotTableFieldOptions struct {
	CollapseStateOptions []PivotTableFieldCollapseStateOption `json:"CollapseStateOptions,omitempty"`
	DataPathOptions []PivotTableDataPathOption `json:"DataPathOptions,omitempty"`
	SelectedFieldOptions []PivotTableFieldOption `json:"SelectedFieldOptions,omitempty"`
}

type PivotTableFieldSubtotalOptions struct {
	FieldId *string `json:"FieldId,omitempty"`
}

type PivotTableFieldWells struct {
	PivotTableAggregatedFieldWells *PivotTableAggregatedFieldWells `json:"PivotTableAggregatedFieldWells,omitempty"`
}

type PivotTableOptions struct {
	CellStyle *TableCellStyle `json:"CellStyle,omitempty"`
	CollapsedRowDimensionsVisibility *string `json:"CollapsedRowDimensionsVisibility,omitempty"`
	ColumnHeaderStyle *TableCellStyle `json:"ColumnHeaderStyle,omitempty"`
	ColumnNamesVisibility *string `json:"ColumnNamesVisibility,omitempty"`
	DefaultCellWidth *string `json:"DefaultCellWidth,omitempty"`
	MetricPlacement *string `json:"MetricPlacement,omitempty"`
	RowAlternateColorOptions *RowAlternateColorOptions `json:"RowAlternateColorOptions,omitempty"`
	RowFieldNamesStyle *TableCellStyle `json:"RowFieldNamesStyle,omitempty"`
	RowHeaderStyle *TableCellStyle `json:"RowHeaderStyle,omitempty"`
	RowsLabelOptions *PivotTableRowsLabelOptions `json:"RowsLabelOptions,omitempty"`
	RowsLayout *string `json:"RowsLayout,omitempty"`
	SingleMetricVisibility *string `json:"SingleMetricVisibility,omitempty"`
	ToggleButtonsVisibility *string `json:"ToggleButtonsVisibility,omitempty"`
}

type PivotTablePaginatedReportOptions struct {
	OverflowColumnHeaderVisibility *string `json:"OverflowColumnHeaderVisibility,omitempty"`
	VerticalOverflowVisibility *string `json:"VerticalOverflowVisibility,omitempty"`
}

type PivotTableRowsLabelOptions struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type PivotTableSortBy struct {
	Column *ColumnSort `json:"Column,omitempty"`
	DataPath *DataPathSort `json:"DataPath,omitempty"`
	Field *FieldSort `json:"Field,omitempty"`
}

type PivotTableSortConfiguration struct {
	FieldSortOptions []PivotFieldSortOptions `json:"FieldSortOptions,omitempty"`
}

type PivotTableTotalOptions struct {
	ColumnSubtotalOptions *SubtotalOptions `json:"ColumnSubtotalOptions,omitempty"`
	ColumnTotalOptions *PivotTotalOptions `json:"ColumnTotalOptions,omitempty"`
	RowSubtotalOptions *SubtotalOptions `json:"RowSubtotalOptions,omitempty"`
	RowTotalOptions *PivotTotalOptions `json:"RowTotalOptions,omitempty"`
}

type PivotTableVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *PivotTableConfiguration `json:"ChartConfiguration,omitempty"`
	ConditionalFormatting *PivotTableConditionalFormatting `json:"ConditionalFormatting,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type PivotTotalOptions struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	MetricHeaderCellStyle *TableCellStyle `json:"MetricHeaderCellStyle,omitempty"`
	Placement *string `json:"Placement,omitempty"`
	ScrollStatus *string `json:"ScrollStatus,omitempty"`
	TotalAggregationOptions []TotalAggregationOption `json:"TotalAggregationOptions,omitempty"`
	TotalCellStyle *TableCellStyle `json:"TotalCellStyle,omitempty"`
	TotalsVisibility *string `json:"TotalsVisibility,omitempty"`
	ValueCellStyle *TableCellStyle `json:"ValueCellStyle,omitempty"`
}

type PivotedLabel struct {
	LabelName string `json:"LabelName,omitempty"`
	NewColumnId string `json:"NewColumnId,omitempty"`
	NewColumnName string `json:"NewColumnName,omitempty"`
}

type PluginVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *PluginVisualConfiguration `json:"ChartConfiguration,omitempty"`
	PluginArn string `json:"PluginArn,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type PluginVisualConfiguration struct {
	FieldWells []PluginVisualFieldWell `json:"FieldWells,omitempty"`
	SortConfiguration *PluginVisualSortConfiguration `json:"SortConfiguration,omitempty"`
	VisualOptions *PluginVisualOptions `json:"VisualOptions,omitempty"`
}

type PluginVisualFieldWell struct {
	AxisName *string `json:"AxisName,omitempty"`
	Dimensions []DimensionField `json:"Dimensions,omitempty"`
	Measures []MeasureField `json:"Measures,omitempty"`
	Unaggregated []UnaggregatedField `json:"Unaggregated,omitempty"`
}

type PluginVisualItemsLimitConfiguration struct {
	ItemsLimit int64 `json:"ItemsLimit,omitempty"`
}

type PluginVisualOptions struct {
	VisualProperties []PluginVisualProperty `json:"VisualProperties,omitempty"`
}

type PluginVisualProperty struct {
	Name *string `json:"Name,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type PluginVisualSortConfiguration struct {
	PluginVisualTableQuerySort *PluginVisualTableQuerySort `json:"PluginVisualTableQuerySort,omitempty"`
}

type PluginVisualTableQuerySort struct {
	ItemsLimitConfiguration *PluginVisualItemsLimitConfiguration `json:"ItemsLimitConfiguration,omitempty"`
	RowSort []FieldSortOptions `json:"RowSort,omitempty"`
}

type PostgreSqlParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type PredefinedHierarchy struct {
	Columns []ColumnIdentifier `json:"Columns,omitempty"`
	DrillDownFilters []DrillDownFilter `json:"DrillDownFilters,omitempty"`
	HierarchyId string `json:"HierarchyId,omitempty"`
}

type PredictQAResultsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	IncludeGeneratedAnswer *string `json:"IncludeGeneratedAnswer,omitempty"`
	IncludeQuickSightQIndex *string `json:"IncludeQuickSightQIndex,omitempty"`
	MaxTopicsToConsider int `json:"MaxTopicsToConsider,omitempty"`
	QueryText string `json:"QueryText,omitempty"`
}

type PredictQAResultsResponse struct {
	AdditionalResults []QAResult `json:"AdditionalResults,omitempty"`
	PrimaryResult *QAResult `json:"PrimaryResult,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type PrestoParameters struct {
	Catalog string `json:"Catalog,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type ProgressBarOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type ProjectOperation struct {
	Alias *string `json:"Alias,omitempty"`
	ProjectedColumns []string `json:"ProjectedColumns,omitempty"`
	Source *TransformOperationSource `json:"Source,omitempty"`
}

type PutDataSetRefreshPropertiesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	DataSetRefreshProperties DataSetRefreshProperties `json:"DataSetRefreshProperties,omitempty"`
}

type PutDataSetRefreshPropertiesResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type QAResult struct {
	DashboardVisual *DashboardVisualResult `json:"DashboardVisual,omitempty"`
	GeneratedAnswer *GeneratedAnswerResult `json:"GeneratedAnswer,omitempty"`
	ResultType *string `json:"ResultType,omitempty"`
}

type QBusinessParameters struct {
	ApplicationArn string `json:"ApplicationArn,omitempty"`
}

type QDataKey struct {
	QDataKeyArn *string `json:"QDataKeyArn,omitempty"`
	QDataKeyType *string `json:"QDataKeyType,omitempty"`
}

type QueryExecutionOptions struct {
	QueryExecutionMode *string `json:"QueryExecutionMode,omitempty"`
}

type QueueInfo struct {
	QueuedIngestion string `json:"QueuedIngestion,omitempty"`
	WaitingOnIngestion string `json:"WaitingOnIngestion,omitempty"`
}

type QuickSuiteActionsOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type RadarChartAggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	Color []DimensionField `json:"Color,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type RadarChartAreaStyleSettings struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type RadarChartConfiguration struct {
	AlternateBandColorsVisibility *string `json:"AlternateBandColorsVisibility,omitempty"`
	AlternateBandEvenColor *string `json:"AlternateBandEvenColor,omitempty"`
	AlternateBandOddColor *string `json:"AlternateBandOddColor,omitempty"`
	AxesRangeScale *string `json:"AxesRangeScale,omitempty"`
	BaseSeriesSettings *RadarChartSeriesSettings `json:"BaseSeriesSettings,omitempty"`
	CategoryAxis *AxisDisplayOptions `json:"CategoryAxis,omitempty"`
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	ColorAxis *AxisDisplayOptions `json:"ColorAxis,omitempty"`
	ColorLabelOptions *ChartAxisLabelOptions `json:"ColorLabelOptions,omitempty"`
	FieldWells *RadarChartFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	Shape *string `json:"Shape,omitempty"`
	SortConfiguration *RadarChartSortConfiguration `json:"SortConfiguration,omitempty"`
	StartAngle float64 `json:"StartAngle,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
}

type RadarChartFieldWells struct {
	RadarChartAggregatedFieldWells *RadarChartAggregatedFieldWells `json:"RadarChartAggregatedFieldWells,omitempty"`
}

type RadarChartSeriesSettings struct {
	AreaStyleSettings *RadarChartAreaStyleSettings `json:"AreaStyleSettings,omitempty"`
}

type RadarChartSortConfiguration struct {
	CategoryItemsLimit *ItemsLimitConfiguration `json:"CategoryItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
	ColorItemsLimit *ItemsLimitConfiguration `json:"ColorItemsLimit,omitempty"`
	ColorSort []FieldSortOptions `json:"ColorSort,omitempty"`
}

type RadarChartVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *RadarChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type RangeConstant struct {
	Maximum *string `json:"Maximum,omitempty"`
	Minimum *string `json:"Minimum,omitempty"`
}

type RangeEndsLabelType struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type RdsParameters struct {
	Database string `json:"Database,omitempty"`
	InstanceId string `json:"InstanceId,omitempty"`
}

type ReadAPIKeyConnectionMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	Email *string `json:"Email,omitempty"`
}

type ReadAuthConfig struct {
	AuthenticationMetadata ReadAuthenticationMetadata `json:"AuthenticationMetadata,omitempty"`
	AuthenticationType string `json:"AuthenticationType,omitempty"`
}

type ReadAuthenticationMetadata struct {
	ApiKeyConnectionMetadata *ReadAPIKeyConnectionMetadata `json:"ApiKeyConnectionMetadata,omitempty"`
	AuthorizationCodeGrantMetadata *ReadAuthorizationCodeGrantMetadata `json:"AuthorizationCodeGrantMetadata,omitempty"`
	BasicAuthConnectionMetadata *ReadBasicAuthConnectionMetadata `json:"BasicAuthConnectionMetadata,omitempty"`
	ClientCredentialsGrantMetadata *ReadClientCredentialsGrantMetadata `json:"ClientCredentialsGrantMetadata,omitempty"`
	IamConnectionMetadata *ReadIamConnectionMetadata `json:"IamConnectionMetadata,omitempty"`
	NoneConnectionMetadata *ReadNoneConnectionMetadata `json:"NoneConnectionMetadata,omitempty"`
}

type ReadAuthorizationCodeGrantCredentialsDetails struct {
	ReadAuthorizationCodeGrantDetails *ReadAuthorizationCodeGrantDetails `json:"ReadAuthorizationCodeGrantDetails,omitempty"`
}

type ReadAuthorizationCodeGrantDetails struct {
	AuthorizationEndpoint string `json:"AuthorizationEndpoint,omitempty"`
	ClientId string `json:"ClientId,omitempty"`
	TokenEndpoint string `json:"TokenEndpoint,omitempty"`
}

type ReadAuthorizationCodeGrantMetadata struct {
	AuthorizationCodeGrantCredentialsSource *string `json:"AuthorizationCodeGrantCredentialsSource,omitempty"`
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	ReadAuthorizationCodeGrantCredentialsDetails *ReadAuthorizationCodeGrantCredentialsDetails `json:"ReadAuthorizationCodeGrantCredentialsDetails,omitempty"`
	RedirectUrl string `json:"RedirectUrl,omitempty"`
}

type ReadBasicAuthConnectionMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	Username string `json:"Username,omitempty"`
}

type ReadClientCredentialsDetails struct {
	ReadClientCredentialsGrantDetails *ReadClientCredentialsGrantDetails `json:"ReadClientCredentialsGrantDetails,omitempty"`
}

type ReadClientCredentialsGrantDetails struct {
	ClientId string `json:"ClientId,omitempty"`
	TokenEndpoint string `json:"TokenEndpoint,omitempty"`
}

type ReadClientCredentialsGrantMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
	ClientCredentialsSource *string `json:"ClientCredentialsSource,omitempty"`
	ReadClientCredentialsDetails *ReadClientCredentialsDetails `json:"ReadClientCredentialsDetails,omitempty"`
}

type ReadIamConnectionMetadata struct {
	RoleArn string `json:"RoleArn,omitempty"`
	SourceArn string `json:"SourceArn,omitempty"`
}

type ReadNoneConnectionMetadata struct {
	BaseEndpoint string `json:"BaseEndpoint,omitempty"`
}

type RecentSnapshotsConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type RedshiftIAMParameters struct {
	AutoCreateDatabaseUser bool `json:"AutoCreateDatabaseUser,omitempty"`
	DatabaseGroups []string `json:"DatabaseGroups,omitempty"`
	DatabaseUser *string `json:"DatabaseUser,omitempty"`
	RoleArn string `json:"RoleArn,omitempty"`
}

type RedshiftParameters struct {
	ClusterId *string `json:"ClusterId,omitempty"`
	Database string `json:"Database,omitempty"`
	Host *string `json:"Host,omitempty"`
	IAMParameters *RedshiftIAMParameters `json:"IAMParameters,omitempty"`
	IdentityCenterConfiguration *IdentityCenterConfiguration `json:"IdentityCenterConfiguration,omitempty"`
	Port int `json:"Port,omitempty"`
}

type ReferenceLine struct {
	DataConfiguration ReferenceLineDataConfiguration `json:"DataConfiguration,omitempty"`
	LabelConfiguration *ReferenceLineLabelConfiguration `json:"LabelConfiguration,omitempty"`
	Status *string `json:"Status,omitempty"`
	StyleConfiguration *ReferenceLineStyleConfiguration `json:"StyleConfiguration,omitempty"`
}

type ReferenceLineCustomLabelConfiguration struct {
	CustomLabel string `json:"CustomLabel,omitempty"`
}

type ReferenceLineDataConfiguration struct {
	AxisBinding *string `json:"AxisBinding,omitempty"`
	DynamicConfiguration *ReferenceLineDynamicDataConfiguration `json:"DynamicConfiguration,omitempty"`
	SeriesType *string `json:"SeriesType,omitempty"`
	StaticConfiguration *ReferenceLineStaticDataConfiguration `json:"StaticConfiguration,omitempty"`
}

type ReferenceLineDynamicDataConfiguration struct {
	Calculation NumericalAggregationFunction `json:"Calculation,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	MeasureAggregationFunction *AggregationFunction `json:"MeasureAggregationFunction,omitempty"`
}

type ReferenceLineLabelConfiguration struct {
	CustomLabelConfiguration *ReferenceLineCustomLabelConfiguration `json:"CustomLabelConfiguration,omitempty"`
	FontColor *string `json:"FontColor,omitempty"`
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
	HorizontalPosition *string `json:"HorizontalPosition,omitempty"`
	ValueLabelConfiguration *ReferenceLineValueLabelConfiguration `json:"ValueLabelConfiguration,omitempty"`
	VerticalPosition *string `json:"VerticalPosition,omitempty"`
}

type ReferenceLineStaticDataConfiguration struct {
	Value float64 `json:"Value,omitempty"`
}

type ReferenceLineStyleConfiguration struct {
	Color *string `json:"Color,omitempty"`
	Pattern *string `json:"Pattern,omitempty"`
}

type ReferenceLineValueLabelConfiguration struct {
	FormatConfiguration *NumericFormatConfiguration `json:"FormatConfiguration,omitempty"`
	RelativePosition *string `json:"RelativePosition,omitempty"`
}

type RefreshConfiguration struct {
	IncrementalRefresh IncrementalRefresh `json:"IncrementalRefresh,omitempty"`
}

type RefreshFailureConfiguration struct {
	EmailAlert *RefreshFailureEmailAlert `json:"EmailAlert,omitempty"`
}

type RefreshFailureEmailAlert struct {
	AlertStatus *string `json:"AlertStatus,omitempty"`
}

type RefreshFrequency struct {
	Interval string `json:"Interval,omitempty"`
	RefreshOnDay *ScheduleRefreshOnEntity `json:"RefreshOnDay,omitempty"`
	TimeOfTheDay *string `json:"TimeOfTheDay,omitempty"`
	Timezone *string `json:"Timezone,omitempty"`
}

type RefreshSchedule struct {
	Arn *string `json:"Arn,omitempty"`
	RefreshType string `json:"RefreshType,omitempty"`
	ScheduleFrequency RefreshFrequency `json:"ScheduleFrequency,omitempty"`
	ScheduleId string `json:"ScheduleId,omitempty"`
	StartAfterDateTime *time.Time `json:"StartAfterDateTime,omitempty"`
}

type RegisterUserRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomFederationProviderUrl *string `json:"CustomFederationProviderUrl,omitempty"`
	CustomPermissionsName *string `json:"CustomPermissionsName,omitempty"`
	Email string `json:"Email,omitempty"`
	ExternalLoginFederationProviderType *string `json:"ExternalLoginFederationProviderType,omitempty"`
	ExternalLoginId *string `json:"ExternalLoginId,omitempty"`
	IamArn *string `json:"IamArn,omitempty"`
	IdentityType string `json:"IdentityType,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	SessionName *string `json:"SessionName,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	UserName *string `json:"UserName,omitempty"`
	UserRole string `json:"UserRole,omitempty"`
}

type RegisterUserResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	User *User `json:"User,omitempty"`
	UserInvitationUrl *string `json:"UserInvitationUrl,omitempty"`
}

type RegisteredCustomerManagedKey struct {
	DefaultKey bool `json:"DefaultKey,omitempty"`
	KeyArn *string `json:"KeyArn,omitempty"`
}

type RegisteredUserConsoleFeatureConfigurations struct {
	AmazonQInQuickSight *AmazonQInQuickSightConsoleConfigurations `json:"AmazonQInQuickSight,omitempty"`
	RecentSnapshots *RecentSnapshotsConfigurations `json:"RecentSnapshots,omitempty"`
	Schedules *SchedulesConfigurations `json:"Schedules,omitempty"`
	SharedView *SharedViewConfigurations `json:"SharedView,omitempty"`
	StatePersistence *StatePersistenceConfigurations `json:"StatePersistence,omitempty"`
	ThresholdAlerts *ThresholdAlertsConfigurations `json:"ThresholdAlerts,omitempty"`
}

type RegisteredUserDashboardEmbeddingConfiguration struct {
	FeatureConfigurations *RegisteredUserDashboardFeatureConfigurations `json:"FeatureConfigurations,omitempty"`
	InitialDashboardId string `json:"InitialDashboardId,omitempty"`
}

type RegisteredUserDashboardFeatureConfigurations struct {
	AmazonQInQuickSight *AmazonQInQuickSightDashboardConfigurations `json:"AmazonQInQuickSight,omitempty"`
	Bookmarks *BookmarksConfigurations `json:"Bookmarks,omitempty"`
	RecentSnapshots *RecentSnapshotsConfigurations `json:"RecentSnapshots,omitempty"`
	Schedules *SchedulesConfigurations `json:"Schedules,omitempty"`
	SharedView *SharedViewConfigurations `json:"SharedView,omitempty"`
	StatePersistence *StatePersistenceConfigurations `json:"StatePersistence,omitempty"`
	ThresholdAlerts *ThresholdAlertsConfigurations `json:"ThresholdAlerts,omitempty"`
}

type RegisteredUserDashboardVisualEmbeddingConfiguration struct {
	InitialDashboardVisualId DashboardVisualId `json:"InitialDashboardVisualId,omitempty"`
}

type RegisteredUserEmbeddingExperienceConfiguration struct {
	Dashboard *RegisteredUserDashboardEmbeddingConfiguration `json:"Dashboard,omitempty"`
	DashboardVisual *RegisteredUserDashboardVisualEmbeddingConfiguration `json:"DashboardVisual,omitempty"`
	GenerativeQnA *RegisteredUserGenerativeQnAEmbeddingConfiguration `json:"GenerativeQnA,omitempty"`
	QSearchBar *RegisteredUserQSearchBarEmbeddingConfiguration `json:"QSearchBar,omitempty"`
	QuickChat *RegisteredUserQuickChatEmbeddingConfiguration `json:"QuickChat,omitempty"`
	QuickSightConsole *RegisteredUserQuickSightConsoleEmbeddingConfiguration `json:"QuickSightConsole,omitempty"`
}

type RegisteredUserGenerativeQnAEmbeddingConfiguration struct {
	InitialTopicId *string `json:"InitialTopicId,omitempty"`
}

type RegisteredUserQSearchBarEmbeddingConfiguration struct {
	InitialTopicId *string `json:"InitialTopicId,omitempty"`
}

type RegisteredUserQuickChatEmbeddingConfiguration struct {
}

type RegisteredUserQuickSightConsoleEmbeddingConfiguration struct {
	FeatureConfigurations *RegisteredUserConsoleFeatureConfigurations `json:"FeatureConfigurations,omitempty"`
	InitialPath *string `json:"InitialPath,omitempty"`
}

type RegisteredUserSnapshotJobResult struct {
	FileGroups []SnapshotJobResultFileGroup `json:"FileGroups,omitempty"`
}

type RelationalTable struct {
	Catalog *string `json:"Catalog,omitempty"`
	DataSourceArn string `json:"DataSourceArn,omitempty"`
	InputColumns []InputColumn `json:"InputColumns,omitempty"`
	Name string `json:"Name,omitempty"`
	Schema *string `json:"Schema,omitempty"`
}

type RelativeDateTimeControlDisplayOptions struct {
	DateTimeFormat *string `json:"DateTimeFormat,omitempty"`
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type RelativeDatesFilter struct {
	AnchorDateConfiguration AnchorDateConfiguration `json:"AnchorDateConfiguration,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	ExcludePeriodConfiguration *ExcludePeriodConfiguration `json:"ExcludePeriodConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	MinimumGranularity *string `json:"MinimumGranularity,omitempty"`
	NullOption string `json:"NullOption,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
	RelativeDateType string `json:"RelativeDateType,omitempty"`
	RelativeDateValue int `json:"RelativeDateValue,omitempty"`
	TimeGranularity string `json:"TimeGranularity,omitempty"`
}

type RenameColumnOperation struct {
	ColumnName string `json:"ColumnName,omitempty"`
	NewColumnName string `json:"NewColumnName,omitempty"`
}

type RenameColumnsOperation struct {
	Alias string `json:"Alias,omitempty"`
	RenameColumnOperations []RenameColumnOperation `json:"RenameColumnOperations,omitempty"`
	Source TransformOperationSource `json:"Source,omitempty"`
}

type ResourcePermission struct {
	Actions []string `json:"Actions,omitempty"`
	Principal string `json:"Principal,omitempty"`
}

type RestoreAnalysisRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	RestoreToFolders bool `json:"restore-to-folders,omitempty"`
}

type RestoreAnalysisResponse struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	RestorationFailedFolderArns []string `json:"RestorationFailedFolderArns,omitempty"`
	Status int `json:"Status,omitempty"`
}

type RollingDateConfiguration struct {
	DataSetIdentifier *string `json:"DataSetIdentifier,omitempty"`
	Expression string `json:"Expression,omitempty"`
}

type RowAlternateColorOptions struct {
	RowAlternateColors []string `json:"RowAlternateColors,omitempty"`
	Status *string `json:"Status,omitempty"`
	UsePrimaryBackgroundColor *string `json:"UsePrimaryBackgroundColor,omitempty"`
}

type RowInfo struct {
	RowsDropped int64 `json:"RowsDropped,omitempty"`
	RowsIngested int64 `json:"RowsIngested,omitempty"`
	TotalRowsInDataset int64 `json:"TotalRowsInDataset,omitempty"`
}

type RowLevelPermissionConfiguration struct {
	RowLevelPermissionDataSet *RowLevelPermissionDataSet `json:"RowLevelPermissionDataSet,omitempty"`
	TagConfiguration *RowLevelPermissionTagConfiguration `json:"TagConfiguration,omitempty"`
}

type RowLevelPermissionDataSet struct {
	Arn string `json:"Arn,omitempty"`
	FormatVersion *string `json:"FormatVersion,omitempty"`
	Namespace *string `json:"Namespace,omitempty"`
	PermissionPolicy string `json:"PermissionPolicy,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type RowLevelPermissionTagConfiguration struct {
	Status *string `json:"Status,omitempty"`
	TagRuleConfigurations [][]string `json:"TagRuleConfigurations,omitempty"`
	TagRules []RowLevelPermissionTagRule `json:"TagRules,omitempty"`
}

type RowLevelPermissionTagRule struct {
	ColumnName string `json:"ColumnName,omitempty"`
	MatchAllValue *string `json:"MatchAllValue,omitempty"`
	TagKey string `json:"TagKey,omitempty"`
	TagMultiValueDelimiter *string `json:"TagMultiValueDelimiter,omitempty"`
}

type S3BucketConfiguration struct {
	BucketName string `json:"BucketName,omitempty"`
	BucketPrefix string `json:"BucketPrefix,omitempty"`
	BucketRegion string `json:"BucketRegion,omitempty"`
}

type S3KnowledgeBaseParameters struct {
	BucketUrl string `json:"BucketUrl,omitempty"`
	MetadataFilesLocation *string `json:"MetadataFilesLocation,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
}

type S3Parameters struct {
	ManifestFileLocation ManifestFileLocation `json:"ManifestFileLocation,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
}

type S3Source struct {
	DataSourceArn string `json:"DataSourceArn,omitempty"`
	InputColumns []InputColumn `json:"InputColumns,omitempty"`
	UploadSettings *UploadSettings `json:"UploadSettings,omitempty"`
}

type SaaSTable struct {
	DataSourceArn string `json:"DataSourceArn,omitempty"`
	InputColumns []InputColumn `json:"InputColumns,omitempty"`
	TablePath []TablePathElement `json:"TablePath,omitempty"`
}

type SameSheetTargetVisualConfiguration struct {
	TargetVisualOptions *string `json:"TargetVisualOptions,omitempty"`
	TargetVisuals []string `json:"TargetVisuals,omitempty"`
}

type SankeyDiagramAggregatedFieldWells struct {
	Destination []DimensionField `json:"Destination,omitempty"`
	Source []DimensionField `json:"Source,omitempty"`
	Weight []MeasureField `json:"Weight,omitempty"`
}

type SankeyDiagramChartConfiguration struct {
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *SankeyDiagramFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	SortConfiguration *SankeyDiagramSortConfiguration `json:"SortConfiguration,omitempty"`
}

type SankeyDiagramFieldWells struct {
	SankeyDiagramAggregatedFieldWells *SankeyDiagramAggregatedFieldWells `json:"SankeyDiagramAggregatedFieldWells,omitempty"`
}

type SankeyDiagramSortConfiguration struct {
	DestinationItemsLimit *ItemsLimitConfiguration `json:"DestinationItemsLimit,omitempty"`
	SourceItemsLimit *ItemsLimitConfiguration `json:"SourceItemsLimit,omitempty"`
	WeightSort []FieldSortOptions `json:"WeightSort,omitempty"`
}

type SankeyDiagramVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *SankeyDiagramChartConfiguration `json:"ChartConfiguration,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type ScatterPlotCategoricallyAggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	Label []DimensionField `json:"Label,omitempty"`
	Size []MeasureField `json:"Size,omitempty"`
	XAxis []MeasureField `json:"XAxis,omitempty"`
	YAxis []MeasureField `json:"YAxis,omitempty"`
}

type ScatterPlotConfiguration struct {
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *ScatterPlotFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	SortConfiguration *ScatterPlotSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
	XAxisDisplayOptions *AxisDisplayOptions `json:"XAxisDisplayOptions,omitempty"`
	XAxisLabelOptions *ChartAxisLabelOptions `json:"XAxisLabelOptions,omitempty"`
	YAxisDisplayOptions *AxisDisplayOptions `json:"YAxisDisplayOptions,omitempty"`
	YAxisLabelOptions *ChartAxisLabelOptions `json:"YAxisLabelOptions,omitempty"`
}

type ScatterPlotFieldWells struct {
	ScatterPlotCategoricallyAggregatedFieldWells *ScatterPlotCategoricallyAggregatedFieldWells `json:"ScatterPlotCategoricallyAggregatedFieldWells,omitempty"`
	ScatterPlotUnaggregatedFieldWells *ScatterPlotUnaggregatedFieldWells `json:"ScatterPlotUnaggregatedFieldWells,omitempty"`
}

type ScatterPlotSortConfiguration struct {
	ScatterPlotLimitConfiguration *ItemsLimitConfiguration `json:"ScatterPlotLimitConfiguration,omitempty"`
}

type ScatterPlotUnaggregatedFieldWells struct {
	Category []DimensionField `json:"Category,omitempty"`
	Label []DimensionField `json:"Label,omitempty"`
	Size []MeasureField `json:"Size,omitempty"`
	XAxis []DimensionField `json:"XAxis,omitempty"`
	YAxis []DimensionField `json:"YAxis,omitempty"`
}

type ScatterPlotVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *ScatterPlotConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type ScheduleRefreshOnEntity struct {
	DayOfMonth *string `json:"DayOfMonth,omitempty"`
	DayOfWeek *string `json:"DayOfWeek,omitempty"`
}

type SchedulesConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type ScrollBarOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
	VisibleRange *VisibleRangeOptions `json:"VisibleRange,omitempty"`
}

type SearchActionConnectorsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []ActionConnectorSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type SearchActionConnectorsResponse struct {
	ActionConnectorSummaries []ActionConnectorSummary `json:"ActionConnectorSummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchAnalysesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []AnalysisSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchAnalysesResponse struct {
	AnalysisSummaryList []AnalysisSummary `json:"AnalysisSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchDashboardsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []DashboardSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchDashboardsResponse struct {
	DashboardSummaryList []DashboardSummary `json:"DashboardSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchDataSetsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []DataSetSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchDataSetsResponse struct {
	DataSetSummaries []DataSetSummary `json:"DataSetSummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchDataSourcesRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []DataSourceSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchDataSourcesResponse struct {
	DataSourceSummaries []DataSourceSummary `json:"DataSourceSummaries,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchFlowsFilter struct {
	Name string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value string `json:"Value,omitempty"`
}

type SearchFlowsInput struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []SearchFlowsFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchFlowsOutput struct {
	FlowSummaryList []FlowSummary `json:"FlowSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchFoldersRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []FolderSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchFoldersResponse struct {
	FolderSummaryList []FolderSummary `json:"FolderSummaryList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchGroupsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []GroupSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"max-results,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	NextToken *string `json:"next-token,omitempty"`
}

type SearchGroupsResponse struct {
	GroupList []Group `json:"GroupList,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type SearchTopicsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Filters []TopicSearchFilter `json:"Filters,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type SearchTopicsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicSummaryList []TopicSummary `json:"TopicSummaryList,omitempty"`
}

type SecondaryValueOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type SectionAfterPageBreak struct {
	Status *string `json:"Status,omitempty"`
}

type SectionBasedLayoutCanvasSizeOptions struct {
	PaperCanvasSizeOptions *SectionBasedLayoutPaperCanvasSizeOptions `json:"PaperCanvasSizeOptions,omitempty"`
}

type SectionBasedLayoutConfiguration struct {
	BodySections []BodySectionConfiguration `json:"BodySections,omitempty"`
	CanvasSizeOptions SectionBasedLayoutCanvasSizeOptions `json:"CanvasSizeOptions,omitempty"`
	FooterSections []HeaderFooterSectionConfiguration `json:"FooterSections,omitempty"`
	HeaderSections []HeaderFooterSectionConfiguration `json:"HeaderSections,omitempty"`
}

type SectionBasedLayoutPaperCanvasSizeOptions struct {
	PaperMargin *Spacing `json:"PaperMargin,omitempty"`
	PaperOrientation *string `json:"PaperOrientation,omitempty"`
	PaperSize *string `json:"PaperSize,omitempty"`
}

type SectionLayoutConfiguration struct {
	FreeFormLayout FreeFormSectionLayoutConfiguration `json:"FreeFormLayout,omitempty"`
}

type SectionPageBreakConfiguration struct {
	After *SectionAfterPageBreak `json:"After,omitempty"`
}

type SectionStyle struct {
	Height *string `json:"Height,omitempty"`
	Padding *Spacing `json:"Padding,omitempty"`
}

type SelectedSheetsFilterScopeConfiguration struct {
	SheetVisualScopingConfigurations []SheetVisualScopingConfiguration `json:"SheetVisualScopingConfigurations,omitempty"`
}

type SelfUpgradeConfiguration struct {
	SelfUpgradeStatus *string `json:"SelfUpgradeStatus,omitempty"`
}

type SelfUpgradeRequestDetail struct {
	CreationTime int64 `json:"CreationTime,omitempty"`
	OriginalRole *string `json:"OriginalRole,omitempty"`
	RequestNote *string `json:"RequestNote,omitempty"`
	RequestStatus *string `json:"RequestStatus,omitempty"`
	RequestedRole *string `json:"RequestedRole,omitempty"`
	UpgradeRequestId *string `json:"UpgradeRequestId,omitempty"`
	UserName *string `json:"UserName,omitempty"`
	LastUpdateAttemptTime int64 `json:"lastUpdateAttemptTime,omitempty"`
	LastUpdateFailureReason *string `json:"lastUpdateFailureReason,omitempty"`
}

type SemanticEntityType struct {
	SubTypeName *string `json:"SubTypeName,omitempty"`
	TypeName *string `json:"TypeName,omitempty"`
	TypeParameters map[string]string `json:"TypeParameters,omitempty"`
}

type SemanticModelConfiguration struct {
	TableMap map[string]SemanticTable `json:"TableMap,omitempty"`
}

type SemanticTable struct {
	Alias string `json:"Alias,omitempty"`
	DestinationTableId string `json:"DestinationTableId,omitempty"`
	RowLevelPermissionConfiguration *RowLevelPermissionConfiguration `json:"RowLevelPermissionConfiguration,omitempty"`
}

type SemanticType struct {
	FalseyCellValue *string `json:"FalseyCellValue,omitempty"`
	FalseyCellValueSynonyms []string `json:"FalseyCellValueSynonyms,omitempty"`
	SubTypeName *string `json:"SubTypeName,omitempty"`
	TruthyCellValue *string `json:"TruthyCellValue,omitempty"`
	TruthyCellValueSynonyms []string `json:"TruthyCellValueSynonyms,omitempty"`
	TypeName *string `json:"TypeName,omitempty"`
	TypeParameters map[string]string `json:"TypeParameters,omitempty"`
}

type SeriesItem struct {
	DataFieldSeriesItem *DataFieldSeriesItem `json:"DataFieldSeriesItem,omitempty"`
	FieldSeriesItem *FieldSeriesItem `json:"FieldSeriesItem,omitempty"`
}

type ServiceNowParameters struct {
	SiteBaseUrl string `json:"SiteBaseUrl,omitempty"`
}

type SessionTag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type SetParameterValueConfiguration struct {
	DestinationParameterName string `json:"DestinationParameterName,omitempty"`
	Value DestinationParameterValueConfiguration `json:"Value,omitempty"`
}

type ShapeConditionalFormat struct {
	BackgroundColor ConditionalFormattingColor `json:"BackgroundColor,omitempty"`
}

type SharedViewConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type Sheet struct {
	Images []SheetImage `json:"Images,omitempty"`
	Name *string `json:"Name,omitempty"`
	SheetId *string `json:"SheetId,omitempty"`
}

type SheetBackgroundStyle struct {
	Color *string `json:"Color,omitempty"`
	Gradient *string `json:"Gradient,omitempty"`
}

type SheetControlInfoIconLabelOptions struct {
	InfoIconText *string `json:"InfoIconText,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type SheetControlLayout struct {
	Configuration SheetControlLayoutConfiguration `json:"Configuration,omitempty"`
}

type SheetControlLayoutConfiguration struct {
	GridLayout *GridLayoutConfiguration `json:"GridLayout,omitempty"`
}

type SheetControlsOption struct {
	VisibilityState *string `json:"VisibilityState,omitempty"`
}

type SheetDefinition struct {
	ContentType *string `json:"ContentType,omitempty"`
	CustomActionDefaults *VisualCustomActionDefaults `json:"CustomActionDefaults,omitempty"`
	Description *string `json:"Description,omitempty"`
	FilterControls []FilterControl `json:"FilterControls,omitempty"`
	Images []SheetImage `json:"Images,omitempty"`
	Layouts []Layout `json:"Layouts,omitempty"`
	Name *string `json:"Name,omitempty"`
	ParameterControls []ParameterControl `json:"ParameterControls,omitempty"`
	SheetControlLayouts []SheetControlLayout `json:"SheetControlLayouts,omitempty"`
	SheetId string `json:"SheetId,omitempty"`
	TextBoxes []SheetTextBox `json:"TextBoxes,omitempty"`
	Title *string `json:"Title,omitempty"`
	Visuals []Visual `json:"Visuals,omitempty"`
}

type SheetElementConfigurationOverrides struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type SheetElementRenderingRule struct {
	ConfigurationOverrides SheetElementConfigurationOverrides `json:"ConfigurationOverrides,omitempty"`
	Expression string `json:"Expression,omitempty"`
}

type SheetImage struct {
	Actions []ImageCustomAction `json:"Actions,omitempty"`
	ImageContentAltText *string `json:"ImageContentAltText,omitempty"`
	Interactions *ImageInteractionOptions `json:"Interactions,omitempty"`
	Scaling *SheetImageScalingConfiguration `json:"Scaling,omitempty"`
	SheetImageId string `json:"SheetImageId,omitempty"`
	Source SheetImageSource `json:"Source,omitempty"`
	Tooltip *SheetImageTooltipConfiguration `json:"Tooltip,omitempty"`
}

type SheetImageScalingConfiguration struct {
	ScalingType *string `json:"ScalingType,omitempty"`
}

type SheetImageSource struct {
	SheetImageStaticFileSource *SheetImageStaticFileSource `json:"SheetImageStaticFileSource,omitempty"`
}

type SheetImageStaticFileSource struct {
	StaticFileId string `json:"StaticFileId,omitempty"`
}

type SheetImageTooltipConfiguration struct {
	TooltipText *SheetImageTooltipText `json:"TooltipText,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type SheetImageTooltipText struct {
	PlainText *string `json:"PlainText,omitempty"`
}

type SheetLayoutElementMaximizationOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type SheetLayoutGroup struct {
	Id string `json:"Id,omitempty"`
	Members []SheetLayoutGroupMember `json:"Members,omitempty"`
}

type SheetLayoutGroupMember struct {
	Id string `json:"Id,omitempty"`
	Type string `json:"Type,omitempty"`
}

type SheetStyle struct {
	Background *SheetBackgroundStyle `json:"Background,omitempty"`
	Tile *TileStyle `json:"Tile,omitempty"`
	TileLayout *TileLayoutStyle `json:"TileLayout,omitempty"`
}

type SheetTextBox struct {
	Content *string `json:"Content,omitempty"`
	Interactions *TextBoxInteractionOptions `json:"Interactions,omitempty"`
	SheetTextBoxId string `json:"SheetTextBoxId,omitempty"`
}

type SheetTooltip struct {
	SheetId *string `json:"SheetId,omitempty"`
}

type SheetVisualScopingConfiguration struct {
	Scope string `json:"Scope,omitempty"`
	SheetId string `json:"SheetId,omitempty"`
	VisualIds []string `json:"VisualIds,omitempty"`
}

type ShortFormatText struct {
	PlainText *string `json:"PlainText,omitempty"`
	RichText *string `json:"RichText,omitempty"`
}

type SignupResponse struct {
	IAMUser bool `json:"IAMUser,omitempty"`
	AccountName *string `json:"accountName,omitempty"`
	DirectoryType *string `json:"directoryType,omitempty"`
	UserLoginName *string `json:"userLoginName,omitempty"`
}

type SimpleClusterMarker struct {
	Color *string `json:"Color,omitempty"`
}

type SingleAxisOptions struct {
	YAxisOptions *YAxisOptions `json:"YAxisOptions,omitempty"`
}

type SliderControlDisplayOptions struct {
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type Slot struct {
	SlotId *string `json:"SlotId,omitempty"`
	VisualId *string `json:"VisualId,omitempty"`
}

type SmallMultiplesAxisProperties struct {
	Placement *string `json:"Placement,omitempty"`
	Scale *string `json:"Scale,omitempty"`
}

type SmallMultiplesOptions struct {
	MaxVisibleColumns int64 `json:"MaxVisibleColumns,omitempty"`
	MaxVisibleRows int64 `json:"MaxVisibleRows,omitempty"`
	PanelConfiguration *PanelConfiguration `json:"PanelConfiguration,omitempty"`
	XAxis *SmallMultiplesAxisProperties `json:"XAxis,omitempty"`
	YAxis *SmallMultiplesAxisProperties `json:"YAxis,omitempty"`
}

type SnapshotAnonymousUser struct {
	RowLevelPermissionTags []SessionTag `json:"RowLevelPermissionTags,omitempty"`
}

type SnapshotAnonymousUserRedacted struct {
	RowLevelPermissionTagKeys []string `json:"RowLevelPermissionTagKeys,omitempty"`
}

type SnapshotConfiguration struct {
	DestinationConfiguration *SnapshotDestinationConfiguration `json:"DestinationConfiguration,omitempty"`
	FileGroups []SnapshotFileGroup `json:"FileGroups,omitempty"`
	Parameters *Parameters `json:"Parameters,omitempty"`
}

type SnapshotDestinationConfiguration struct {
	S3Destinations []SnapshotS3DestinationConfiguration `json:"S3Destinations,omitempty"`
}

type SnapshotFile struct {
	FormatType string `json:"FormatType,omitempty"`
	SheetSelections []SnapshotFileSheetSelection `json:"SheetSelections,omitempty"`
}

type SnapshotFileGroup struct {
	Files []SnapshotFile `json:"Files,omitempty"`
}

type SnapshotFileSheetSelection struct {
	SelectionScope string `json:"SelectionScope,omitempty"`
	SheetId string `json:"SheetId,omitempty"`
	VisualIds []string `json:"VisualIds,omitempty"`
}

type SnapshotJobErrorInfo struct {
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	ErrorType *string `json:"ErrorType,omitempty"`
}

type SnapshotJobResult struct {
	AnonymousUsers []AnonymousUserSnapshotJobResult `json:"AnonymousUsers,omitempty"`
	RegisteredUsers []RegisteredUserSnapshotJobResult `json:"RegisteredUsers,omitempty"`
}

type SnapshotJobResultErrorInfo struct {
	ErrorMessage *string `json:"ErrorMessage,omitempty"`
	ErrorType *string `json:"ErrorType,omitempty"`
}

type SnapshotJobResultFileGroup struct {
	Files []SnapshotFile `json:"Files,omitempty"`
	S3Results []SnapshotJobS3Result `json:"S3Results,omitempty"`
}

type SnapshotJobS3Result struct {
	ErrorInfo []SnapshotJobResultErrorInfo `json:"ErrorInfo,omitempty"`
	S3DestinationConfiguration *SnapshotS3DestinationConfiguration `json:"S3DestinationConfiguration,omitempty"`
	S3Uri *string `json:"S3Uri,omitempty"`
}

type SnapshotS3DestinationConfiguration struct {
	BucketConfiguration S3BucketConfiguration `json:"BucketConfiguration,omitempty"`
}

type SnapshotUserConfiguration struct {
	AnonymousUsers []SnapshotAnonymousUser `json:"AnonymousUsers,omitempty"`
}

type SnapshotUserConfigurationRedacted struct {
	AnonymousUsers []SnapshotAnonymousUserRedacted `json:"AnonymousUsers,omitempty"`
}

type SnowflakeParameters struct {
	AuthenticationType *string `json:"AuthenticationType,omitempty"`
	Database string `json:"Database,omitempty"`
	DatabaseAccessControlRole *string `json:"DatabaseAccessControlRole,omitempty"`
	Host string `json:"Host,omitempty"`
	OAuthParameters *OAuthParameters `json:"OAuthParameters,omitempty"`
	Warehouse string `json:"Warehouse,omitempty"`
}

type SourceTable struct {
	DataSet *ParentDataSet `json:"DataSet,omitempty"`
	PhysicalTableId *string `json:"PhysicalTableId,omitempty"`
}

type Spacing struct {
	Bottom *string `json:"Bottom,omitempty"`
	Left *string `json:"Left,omitempty"`
	Right *string `json:"Right,omitempty"`
	Top *string `json:"Top,omitempty"`
}

type SparkParameters struct {
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type SparklinesOptions struct {
	AllPointsMarker *LineChartMarkerStyleSettings `json:"AllPointsMarker,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	LineColor *string `json:"LineColor,omitempty"`
	LineInterpolation *string `json:"LineInterpolation,omitempty"`
	MaxValueMarker *LineChartMarkerStyleSettings `json:"MaxValueMarker,omitempty"`
	MinValueMarker *LineChartMarkerStyleSettings `json:"MinValueMarker,omitempty"`
	VisualType *string `json:"VisualType,omitempty"`
	XAxisField DimensionField `json:"XAxisField,omitempty"`
	YAxisBehavior *string `json:"YAxisBehavior,omitempty"`
}

type SpatialStaticFile struct {
	Source *StaticFileSource `json:"Source,omitempty"`
	StaticFileId string `json:"StaticFileId,omitempty"`
}

type SqlServerParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type SslProperties struct {
	DisableSsl bool `json:"DisableSsl,omitempty"`
}

type StarburstParameters struct {
	AuthenticationType *string `json:"AuthenticationType,omitempty"`
	Catalog string `json:"Catalog,omitempty"`
	DatabaseAccessControlRole *string `json:"DatabaseAccessControlRole,omitempty"`
	Host string `json:"Host,omitempty"`
	OAuthParameters *OAuthParameters `json:"OAuthParameters,omitempty"`
	Port int `json:"Port,omitempty"`
	ProductType *string `json:"ProductType,omitempty"`
}

type StartAssetBundleExportJobRequest struct {
	AssetBundleExportJobId string `json:"AssetBundleExportJobId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CloudFormationOverridePropertyConfiguration *AssetBundleCloudFormationOverridePropertyConfiguration `json:"CloudFormationOverridePropertyConfiguration,omitempty"`
	ExportFormat string `json:"ExportFormat,omitempty"`
	IncludeAllDependencies bool `json:"IncludeAllDependencies,omitempty"`
	IncludeFolderMembers *string `json:"IncludeFolderMembers,omitempty"`
	IncludeFolderMemberships bool `json:"IncludeFolderMemberships,omitempty"`
	IncludePermissions bool `json:"IncludePermissions,omitempty"`
	IncludeTags bool `json:"IncludeTags,omitempty"`
	ResourceArns []string `json:"ResourceArns,omitempty"`
	ValidationStrategy *AssetBundleExportJobValidationStrategy `json:"ValidationStrategy,omitempty"`
}

type StartAssetBundleExportJobResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AssetBundleExportJobId *string `json:"AssetBundleExportJobId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type StartAssetBundleImportJobRequest struct {
	AssetBundleImportJobId string `json:"AssetBundleImportJobId,omitempty"`
	AssetBundleImportSource AssetBundleImportSource `json:"AssetBundleImportSource,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FailureAction *string `json:"FailureAction,omitempty"`
	OverrideParameters *AssetBundleImportJobOverrideParameters `json:"OverrideParameters,omitempty"`
	OverridePermissions *AssetBundleImportJobOverridePermissions `json:"OverridePermissions,omitempty"`
	OverrideTags *AssetBundleImportJobOverrideTags `json:"OverrideTags,omitempty"`
	OverrideValidationStrategy *AssetBundleImportJobOverrideValidationStrategy `json:"OverrideValidationStrategy,omitempty"`
}

type StartAssetBundleImportJobResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AssetBundleImportJobId *string `json:"AssetBundleImportJobId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type StartAutomationJobRequest struct {
	AutomationGroupId string `json:"AutomationGroupId,omitempty"`
	AutomationId string `json:"AutomationId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	InputPayload *string `json:"InputPayload,omitempty"`
}

type StartAutomationJobResponse struct {
	Arn string `json:"Arn,omitempty"`
	JobId string `json:"JobId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type StartDashboardSnapshotJobRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	SnapshotConfiguration SnapshotConfiguration `json:"SnapshotConfiguration,omitempty"`
	SnapshotJobId string `json:"SnapshotJobId,omitempty"`
	UserConfiguration *SnapshotUserConfiguration `json:"UserConfiguration,omitempty"`
}

type StartDashboardSnapshotJobResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	SnapshotJobId *string `json:"SnapshotJobId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type StartDashboardSnapshotJobScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	ScheduleId string `json:"ScheduleId,omitempty"`
}

type StartDashboardSnapshotJobScheduleResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type StatePersistenceConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type StaticFile struct {
	ImageStaticFile *ImageStaticFile `json:"ImageStaticFile,omitempty"`
	SpatialStaticFile *SpatialStaticFile `json:"SpatialStaticFile,omitempty"`
}

type StaticFileS3SourceOptions struct {
	BucketName string `json:"BucketName,omitempty"`
	ObjectKey string `json:"ObjectKey,omitempty"`
	Region string `json:"Region,omitempty"`
}

type StaticFileSource struct {
	S3Options *StaticFileS3SourceOptions `json:"S3Options,omitempty"`
	UrlOptions *StaticFileUrlSourceOptions `json:"UrlOptions,omitempty"`
}

type StaticFileUrlSourceOptions struct {
	Url string `json:"Url,omitempty"`
}

type StringDatasetParameter struct {
	DefaultValues *StringDatasetParameterDefaultValues `json:"DefaultValues,omitempty"`
	Id string `json:"Id,omitempty"`
	Name string `json:"Name,omitempty"`
	ValueType string `json:"ValueType,omitempty"`
}

type StringDatasetParameterDefaultValues struct {
	StaticValues []string `json:"StaticValues,omitempty"`
}

type StringDefaultValues struct {
	DynamicValue *DynamicDefaultValue `json:"DynamicValue,omitempty"`
	StaticValues []string `json:"StaticValues,omitempty"`
}

type StringFormatConfiguration struct {
	NullValueFormatConfiguration *NullValueFormatConfiguration `json:"NullValueFormatConfiguration,omitempty"`
	NumericFormatConfiguration *NumericFormatConfiguration `json:"NumericFormatConfiguration,omitempty"`
}

type StringParameter struct {
	Name string `json:"Name,omitempty"`
	Values []string `json:"Values,omitempty"`
}

type StringParameterDeclaration struct {
	DefaultValues *StringDefaultValues `json:"DefaultValues,omitempty"`
	MappedDataSetParameters []MappedDataSetParameter `json:"MappedDataSetParameters,omitempty"`
	Name string `json:"Name,omitempty"`
	ParameterValueType string `json:"ParameterValueType,omitempty"`
	ValueWhenUnset *StringValueWhenUnsetConfiguration `json:"ValueWhenUnset,omitempty"`
}

type StringValueWhenUnsetConfiguration struct {
	CustomValue *string `json:"CustomValue,omitempty"`
	ValueWhenUnsetOption *string `json:"ValueWhenUnsetOption,omitempty"`
}

type SubtotalOptions struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	FieldLevel *string `json:"FieldLevel,omitempty"`
	FieldLevelOptions []PivotTableFieldSubtotalOptions `json:"FieldLevelOptions,omitempty"`
	MetricHeaderCellStyle *TableCellStyle `json:"MetricHeaderCellStyle,omitempty"`
	StyleTargets []TableStyleTarget `json:"StyleTargets,omitempty"`
	TotalCellStyle *TableCellStyle `json:"TotalCellStyle,omitempty"`
	TotalsVisibility *string `json:"TotalsVisibility,omitempty"`
	ValueCellStyle *TableCellStyle `json:"ValueCellStyle,omitempty"`
}

type SucceededTopicReviewedAnswer struct {
	AnswerId *string `json:"AnswerId,omitempty"`
}

type SuccessfulKeyRegistrationEntry struct {
	KeyArn string `json:"KeyArn,omitempty"`
	StatusCode int `json:"StatusCode,omitempty"`
}

type TableAggregatedFieldWells struct {
	GroupBy []DimensionField `json:"GroupBy,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type TableBorderOptions struct {
	Color *string `json:"Color,omitempty"`
	Style *string `json:"Style,omitempty"`
	Thickness int `json:"Thickness,omitempty"`
}

type TableCellConditionalFormatting struct {
	FieldId string `json:"FieldId,omitempty"`
	TextFormat *TextConditionalFormat `json:"TextFormat,omitempty"`
}

type TableCellImageSizingConfiguration struct {
	TableCellImageScalingConfiguration *string `json:"TableCellImageScalingConfiguration,omitempty"`
}

type TableCellStyle struct {
	BackgroundColor *string `json:"BackgroundColor,omitempty"`
	Border *GlobalTableBorderOptions `json:"Border,omitempty"`
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
	Height int `json:"Height,omitempty"`
	HorizontalTextAlignment *string `json:"HorizontalTextAlignment,omitempty"`
	TextWrap *string `json:"TextWrap,omitempty"`
	VerticalTextAlignment *string `json:"VerticalTextAlignment,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type TableConditionalFormatting struct {
	ConditionalFormattingOptions []TableConditionalFormattingOption `json:"ConditionalFormattingOptions,omitempty"`
}

type TableConditionalFormattingOption struct {
	Cell *TableCellConditionalFormatting `json:"Cell,omitempty"`
	Row *TableRowConditionalFormatting `json:"Row,omitempty"`
}

type TableConfiguration struct {
	DashboardCustomizationVisualOptions *DashboardCustomizationVisualOptions `json:"DashboardCustomizationVisualOptions,omitempty"`
	FieldOptions *TableFieldOptions `json:"FieldOptions,omitempty"`
	FieldWells *TableFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	PaginatedReportOptions *TablePaginatedReportOptions `json:"PaginatedReportOptions,omitempty"`
	SortConfiguration *TableSortConfiguration `json:"SortConfiguration,omitempty"`
	TableInlineVisualizations []TableInlineVisualization `json:"TableInlineVisualizations,omitempty"`
	TableOptions *TableOptions `json:"TableOptions,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
	TotalOptions *TotalOptions `json:"TotalOptions,omitempty"`
}

type TableFieldCustomIconContent struct {
	Icon *string `json:"Icon,omitempty"`
}

type TableFieldCustomTextContent struct {
	FontConfiguration FontConfiguration `json:"FontConfiguration,omitempty"`
	Value *string `json:"Value,omitempty"`
}

type TableFieldImageConfiguration struct {
	SizingOptions *TableCellImageSizingConfiguration `json:"SizingOptions,omitempty"`
}

type TableFieldLinkConfiguration struct {
	Content TableFieldLinkContentConfiguration `json:"Content,omitempty"`
	Target string `json:"Target,omitempty"`
}

type TableFieldLinkContentConfiguration struct {
	CustomIconContent *TableFieldCustomIconContent `json:"CustomIconContent,omitempty"`
	CustomTextContent *TableFieldCustomTextContent `json:"CustomTextContent,omitempty"`
}

type TableFieldOption struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	URLStyling *TableFieldURLConfiguration `json:"URLStyling,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
	Width *string `json:"Width,omitempty"`
}

type TableFieldOptions struct {
	Order []string `json:"Order,omitempty"`
	PinnedFieldOptions *TablePinnedFieldOptions `json:"PinnedFieldOptions,omitempty"`
	SelectedFieldOptions []TableFieldOption `json:"SelectedFieldOptions,omitempty"`
	TransposedTableOptions []TransposedTableOption `json:"TransposedTableOptions,omitempty"`
}

type TableFieldURLConfiguration struct {
	ImageConfiguration *TableFieldImageConfiguration `json:"ImageConfiguration,omitempty"`
	LinkConfiguration *TableFieldLinkConfiguration `json:"LinkConfiguration,omitempty"`
}

type TableFieldWells struct {
	TableAggregatedFieldWells *TableAggregatedFieldWells `json:"TableAggregatedFieldWells,omitempty"`
	TableUnaggregatedFieldWells *TableUnaggregatedFieldWells `json:"TableUnaggregatedFieldWells,omitempty"`
}

type TableInlineVisualization struct {
	DataBars *DataBarsOptions `json:"DataBars,omitempty"`
	Sparklines *SparklinesOptions `json:"Sparklines,omitempty"`
}

type TableOptions struct {
	CellStyle *TableCellStyle `json:"CellStyle,omitempty"`
	HeaderStyle *TableCellStyle `json:"HeaderStyle,omitempty"`
	Orientation *string `json:"Orientation,omitempty"`
	RowAlternateColorOptions *RowAlternateColorOptions `json:"RowAlternateColorOptions,omitempty"`
}

type TablePaginatedReportOptions struct {
	OverflowColumnHeaderVisibility *string `json:"OverflowColumnHeaderVisibility,omitempty"`
	VerticalOverflowVisibility *string `json:"VerticalOverflowVisibility,omitempty"`
}

type TablePathElement struct {
	Id *string `json:"Id,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type TablePinnedFieldOptions struct {
	PinnedLeftFields []string `json:"PinnedLeftFields,omitempty"`
}

type TableRowConditionalFormatting struct {
	BackgroundColor *ConditionalFormattingColor `json:"BackgroundColor,omitempty"`
	TextColor *ConditionalFormattingColor `json:"TextColor,omitempty"`
}

type TableSideBorderOptions struct {
	Bottom *TableBorderOptions `json:"Bottom,omitempty"`
	InnerHorizontal *TableBorderOptions `json:"InnerHorizontal,omitempty"`
	InnerVertical *TableBorderOptions `json:"InnerVertical,omitempty"`
	Left *TableBorderOptions `json:"Left,omitempty"`
	Right *TableBorderOptions `json:"Right,omitempty"`
	Top *TableBorderOptions `json:"Top,omitempty"`
}

type TableSortConfiguration struct {
	PaginationConfiguration *PaginationConfiguration `json:"PaginationConfiguration,omitempty"`
	RowSort []FieldSortOptions `json:"RowSort,omitempty"`
}

type TableStyleTarget struct {
	CellType string `json:"CellType,omitempty"`
}

type TableUnaggregatedFieldWells struct {
	Values []UnaggregatedField `json:"Values,omitempty"`
}

type TableVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *TableConfiguration `json:"ChartConfiguration,omitempty"`
	ConditionalFormatting *TableConditionalFormatting `json:"ConditionalFormatting,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type Tag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type TagColumnOperation struct {
	ColumnName string `json:"ColumnName,omitempty"`
	Tags []ColumnTag `json:"Tags,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type TagResourceResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type Template struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
	Version *TemplateVersion `json:"Version,omitempty"`
}

type TemplateAlias struct {
	AliasName *string `json:"AliasName,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	TemplateVersionNumber int64 `json:"TemplateVersionNumber,omitempty"`
}

type TemplateError struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
	ViolatedEntities []Entity `json:"ViolatedEntities,omitempty"`
}

type TemplateSourceAnalysis struct {
	Arn string `json:"Arn,omitempty"`
	DataSetReferences []DataSetReference `json:"DataSetReferences,omitempty"`
}

type TemplateSourceEntity struct {
	SourceAnalysis *TemplateSourceAnalysis `json:"SourceAnalysis,omitempty"`
	SourceTemplate *TemplateSourceTemplate `json:"SourceTemplate,omitempty"`
}

type TemplateSourceTemplate struct {
	Arn string `json:"Arn,omitempty"`
}

type TemplateSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	LatestVersionNumber int64 `json:"LatestVersionNumber,omitempty"`
	Name *string `json:"Name,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
}

type TemplateVersion struct {
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DataSetConfigurations []DataSetConfiguration `json:"DataSetConfigurations,omitempty"`
	Description *string `json:"Description,omitempty"`
	Errors []TemplateError `json:"Errors,omitempty"`
	Sheets []Sheet `json:"Sheets,omitempty"`
	SourceEntityArn *string `json:"SourceEntityArn,omitempty"`
	Status *string `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type TemplateVersionDefinition struct {
	AnalysisDefaults *AnalysisDefaults `json:"AnalysisDefaults,omitempty"`
	CalculatedFields []CalculatedField `json:"CalculatedFields,omitempty"`
	ColumnConfigurations []ColumnConfiguration `json:"ColumnConfigurations,omitempty"`
	DataSetConfigurations []DataSetConfiguration `json:"DataSetConfigurations,omitempty"`
	FilterGroups []FilterGroup `json:"FilterGroups,omitempty"`
	Options *AssetOptions `json:"Options,omitempty"`
	ParameterDeclarations []ParameterDeclaration `json:"ParameterDeclarations,omitempty"`
	QueryExecutionOptions *QueryExecutionOptions `json:"QueryExecutionOptions,omitempty"`
	Sheets []SheetDefinition `json:"Sheets,omitempty"`
	StaticFiles []StaticFile `json:"StaticFiles,omitempty"`
	TooltipSheets []TooltipSheetDefinition `json:"TooltipSheets,omitempty"`
}

type TemplateVersionSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Status *string `json:"Status,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type TeradataParameters struct {
	Database string `json:"Database,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type TextAreaControlDisplayOptions struct {
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	PlaceholderOptions *TextControlPlaceholderOptions `json:"PlaceholderOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type TextBoxInteractionOptions struct {
	TextBoxMenuOption *TextBoxMenuOption `json:"TextBoxMenuOption,omitempty"`
}

type TextBoxMenuOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type TextConditionalFormat struct {
	BackgroundColor *ConditionalFormattingColor `json:"BackgroundColor,omitempty"`
	Icon *ConditionalFormattingIcon `json:"Icon,omitempty"`
	TextColor *ConditionalFormattingColor `json:"TextColor,omitempty"`
}

type TextControlPlaceholderOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type TextFieldControlDisplayOptions struct {
	InfoIconLabelOptions *SheetControlInfoIconLabelOptions `json:"InfoIconLabelOptions,omitempty"`
	PlaceholderOptions *TextControlPlaceholderOptions `json:"PlaceholderOptions,omitempty"`
	TitleOptions *LabelOptions `json:"TitleOptions,omitempty"`
}

type Theme struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
	Type *string `json:"Type,omitempty"`
	Version *ThemeVersion `json:"Version,omitempty"`
}

type ThemeAlias struct {
	AliasName *string `json:"AliasName,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	ThemeVersionNumber int64 `json:"ThemeVersionNumber,omitempty"`
}

type ThemeConfiguration struct {
	DataColorPalette *DataColorPalette `json:"DataColorPalette,omitempty"`
	Sheet *SheetStyle `json:"Sheet,omitempty"`
	Typography *Typography `json:"Typography,omitempty"`
	UIColorPalette *UIColorPalette `json:"UIColorPalette,omitempty"`
}

type ThemeError struct {
	Message *string `json:"Message,omitempty"`
	Type *string `json:"Type,omitempty"`
}

type ThemeSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	LatestVersionNumber int64 `json:"LatestVersionNumber,omitempty"`
	Name *string `json:"Name,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
}

type ThemeVersion struct {
	Arn *string `json:"Arn,omitempty"`
	BaseThemeId *string `json:"BaseThemeId,omitempty"`
	Configuration *ThemeConfiguration `json:"Configuration,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Errors []ThemeError `json:"Errors,omitempty"`
	Status *string `json:"Status,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type ThemeVersionSummary struct {
	Arn *string `json:"Arn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	Description *string `json:"Description,omitempty"`
	Status *string `json:"Status,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type ThousandSeparatorOptions struct {
	GroupingStyle *string `json:"GroupingStyle,omitempty"`
	Symbol *string `json:"Symbol,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type ThresholdAlertsConfigurations struct {
	Enabled bool `json:"Enabled,omitempty"`
}

type TileLayoutStyle struct {
	Gutter *GutterStyle `json:"Gutter,omitempty"`
	Margin *MarginStyle `json:"Margin,omitempty"`
}

type TileStyle struct {
	BackgroundColor *string `json:"BackgroundColor,omitempty"`
	Border *BorderStyle `json:"Border,omitempty"`
	BorderRadius *string `json:"BorderRadius,omitempty"`
	Padding *string `json:"Padding,omitempty"`
}

type TimeBasedForecastProperties struct {
	LowerBoundary float64 `json:"LowerBoundary,omitempty"`
	PeriodsBackward int `json:"PeriodsBackward,omitempty"`
	PeriodsForward int `json:"PeriodsForward,omitempty"`
	PredictionInterval int `json:"PredictionInterval,omitempty"`
	Seasonality int `json:"Seasonality,omitempty"`
	UpperBoundary float64 `json:"UpperBoundary,omitempty"`
}

type TimeEqualityFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
	RollingDate *RollingDateConfiguration `json:"RollingDate,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
	Value *time.Time `json:"Value,omitempty"`
}

type TimeRangeDrillDownFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	RangeMaximum time.Time `json:"RangeMaximum,omitempty"`
	RangeMinimum time.Time `json:"RangeMinimum,omitempty"`
	TimeGranularity string `json:"TimeGranularity,omitempty"`
}

type TimeRangeFilter struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	ExcludePeriodConfiguration *ExcludePeriodConfiguration `json:"ExcludePeriodConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	IncludeMaximum bool `json:"IncludeMaximum,omitempty"`
	IncludeMinimum bool `json:"IncludeMinimum,omitempty"`
	NullOption string `json:"NullOption,omitempty"`
	RangeMaximumValue *TimeRangeFilterValue `json:"RangeMaximumValue,omitempty"`
	RangeMinimumValue *TimeRangeFilterValue `json:"RangeMinimumValue,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type TimeRangeFilterValue struct {
	Parameter *string `json:"Parameter,omitempty"`
	RollingDate *RollingDateConfiguration `json:"RollingDate,omitempty"`
	StaticValue *time.Time `json:"StaticValue,omitempty"`
}

type TooltipItem struct {
	ColumnTooltipItem *ColumnTooltipItem `json:"ColumnTooltipItem,omitempty"`
	FieldTooltipItem *FieldTooltipItem `json:"FieldTooltipItem,omitempty"`
}

type TooltipOptions struct {
	FieldBasedTooltip *FieldBasedTooltip `json:"FieldBasedTooltip,omitempty"`
	SelectedTooltipType *string `json:"SelectedTooltipType,omitempty"`
	SheetTooltip *SheetTooltip `json:"SheetTooltip,omitempty"`
	TooltipVisibility *string `json:"TooltipVisibility,omitempty"`
}

type TooltipSheetDefinition struct {
	Images []SheetImage `json:"Images,omitempty"`
	Layouts []Layout `json:"Layouts,omitempty"`
	Name *string `json:"Name,omitempty"`
	SheetId string `json:"SheetId,omitempty"`
	TextBoxes []SheetTextBox `json:"TextBoxes,omitempty"`
	Visuals []Visual `json:"Visuals,omitempty"`
}

type TopBottomFilter struct {
	AggregationSortConfigurations []AggregationSortConfiguration `json:"AggregationSortConfigurations,omitempty"`
	Column ColumnIdentifier `json:"Column,omitempty"`
	DefaultFilterControlConfiguration *DefaultFilterControlConfiguration `json:"DefaultFilterControlConfiguration,omitempty"`
	FilterId string `json:"FilterId,omitempty"`
	Limit int `json:"Limit,omitempty"`
	ParameterName *string `json:"ParameterName,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type TopBottomMoversComputation struct {
	Category *DimensionField `json:"Category,omitempty"`
	ComputationId string `json:"ComputationId,omitempty"`
	MoverSize int `json:"MoverSize,omitempty"`
	Name *string `json:"Name,omitempty"`
	SortOrder *string `json:"SortOrder,omitempty"`
	Time *DimensionField `json:"Time,omitempty"`
	Type string `json:"Type,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type TopBottomRankedComputation struct {
	Category *DimensionField `json:"Category,omitempty"`
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
	ResultSize int `json:"ResultSize,omitempty"`
	Type string `json:"Type,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type TopicCalculatedField struct {
	Aggregation *string `json:"Aggregation,omitempty"`
	AllowedAggregations []string `json:"AllowedAggregations,omitempty"`
	CalculatedFieldDescription *string `json:"CalculatedFieldDescription,omitempty"`
	CalculatedFieldName string `json:"CalculatedFieldName,omitempty"`
	CalculatedFieldSynonyms []string `json:"CalculatedFieldSynonyms,omitempty"`
	CellValueSynonyms []CellValueSynonym `json:"CellValueSynonyms,omitempty"`
	ColumnDataRole *string `json:"ColumnDataRole,omitempty"`
	ComparativeOrder *ComparativeOrder `json:"ComparativeOrder,omitempty"`
	DefaultFormatting *DefaultFormatting `json:"DefaultFormatting,omitempty"`
	DisableIndexing bool `json:"DisableIndexing,omitempty"`
	Expression string `json:"Expression,omitempty"`
	IsIncludedInTopic bool `json:"IsIncludedInTopic,omitempty"`
	NeverAggregateInFilter bool `json:"NeverAggregateInFilter,omitempty"`
	NonAdditive bool `json:"NonAdditive,omitempty"`
	NotAllowedAggregations []string `json:"NotAllowedAggregations,omitempty"`
	SemanticType *SemanticType `json:"SemanticType,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type TopicCategoryFilter struct {
	CategoryFilterFunction *string `json:"CategoryFilterFunction,omitempty"`
	CategoryFilterType *string `json:"CategoryFilterType,omitempty"`
	Constant *TopicCategoryFilterConstant `json:"Constant,omitempty"`
	Inverse bool `json:"Inverse,omitempty"`
}

type TopicCategoryFilterConstant struct {
	CollectiveConstant *CollectiveConstant `json:"CollectiveConstant,omitempty"`
	ConstantType *string `json:"ConstantType,omitempty"`
	SingularConstant *string `json:"SingularConstant,omitempty"`
}

type TopicColumn struct {
	Aggregation *string `json:"Aggregation,omitempty"`
	AllowedAggregations []string `json:"AllowedAggregations,omitempty"`
	CellValueSynonyms []CellValueSynonym `json:"CellValueSynonyms,omitempty"`
	ColumnDataRole *string `json:"ColumnDataRole,omitempty"`
	ColumnDescription *string `json:"ColumnDescription,omitempty"`
	ColumnFriendlyName *string `json:"ColumnFriendlyName,omitempty"`
	ColumnName string `json:"ColumnName,omitempty"`
	ColumnSynonyms []string `json:"ColumnSynonyms,omitempty"`
	ComparativeOrder *ComparativeOrder `json:"ComparativeOrder,omitempty"`
	DefaultFormatting *DefaultFormatting `json:"DefaultFormatting,omitempty"`
	DisableIndexing bool `json:"DisableIndexing,omitempty"`
	IsIncludedInTopic bool `json:"IsIncludedInTopic,omitempty"`
	NeverAggregateInFilter bool `json:"NeverAggregateInFilter,omitempty"`
	NonAdditive bool `json:"NonAdditive,omitempty"`
	NotAllowedAggregations []string `json:"NotAllowedAggregations,omitempty"`
	SemanticType *SemanticType `json:"SemanticType,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type TopicConfigOptions struct {
	QBusinessInsightsEnabled bool `json:"QBusinessInsightsEnabled,omitempty"`
}

type TopicConstantValue struct {
	ConstantType *string `json:"ConstantType,omitempty"`
	Maximum *string `json:"Maximum,omitempty"`
	Minimum *string `json:"Minimum,omitempty"`
	Value *string `json:"Value,omitempty"`
	ValueList []CollectiveConstantEntry `json:"ValueList,omitempty"`
}

type TopicDateRangeFilter struct {
	Constant *TopicRangeFilterConstant `json:"Constant,omitempty"`
	Inclusive bool `json:"Inclusive,omitempty"`
}

type TopicDetails struct {
	ConfigOptions *TopicConfigOptions `json:"ConfigOptions,omitempty"`
	DataSets []DatasetMetadata `json:"DataSets,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name *string `json:"Name,omitempty"`
	UserExperienceVersion *string `json:"UserExperienceVersion,omitempty"`
}

type TopicFilter struct {
	CategoryFilter *TopicCategoryFilter `json:"CategoryFilter,omitempty"`
	DateRangeFilter *TopicDateRangeFilter `json:"DateRangeFilter,omitempty"`
	FilterClass *string `json:"FilterClass,omitempty"`
	FilterDescription *string `json:"FilterDescription,omitempty"`
	FilterName string `json:"FilterName,omitempty"`
	FilterSynonyms []string `json:"FilterSynonyms,omitempty"`
	FilterType *string `json:"FilterType,omitempty"`
	NullFilter *TopicNullFilter `json:"NullFilter,omitempty"`
	NumericEqualityFilter *TopicNumericEqualityFilter `json:"NumericEqualityFilter,omitempty"`
	NumericRangeFilter *TopicNumericRangeFilter `json:"NumericRangeFilter,omitempty"`
	OperandFieldName string `json:"OperandFieldName,omitempty"`
	RelativeDateFilter *TopicRelativeDateFilter `json:"RelativeDateFilter,omitempty"`
}

type TopicIR struct {
	ContributionAnalysis *TopicIRContributionAnalysis `json:"ContributionAnalysis,omitempty"`
	Filters [][]TopicIRFilterOption `json:"Filters,omitempty"`
	GroupByList []TopicIRGroupBy `json:"GroupByList,omitempty"`
	Metrics []TopicIRMetric `json:"Metrics,omitempty"`
	Sort *TopicSortClause `json:"Sort,omitempty"`
	Visual *VisualOptions `json:"Visual,omitempty"`
}

type TopicIRComparisonMethod struct {
	Period *string `json:"Period,omitempty"`
	Type *string `json:"Type,omitempty"`
	WindowSize int `json:"WindowSize,omitempty"`
}

type TopicIRContributionAnalysis struct {
	Direction *string `json:"Direction,omitempty"`
	Factors []ContributionAnalysisFactor `json:"Factors,omitempty"`
	SortType *string `json:"SortType,omitempty"`
	TimeRanges *ContributionAnalysisTimeRanges `json:"TimeRanges,omitempty"`
}

type TopicIRFilterOption struct {
	AggMetrics []FilterAggMetrics `json:"AggMetrics,omitempty"`
	Aggregation *string `json:"Aggregation,omitempty"`
	AggregationFunctionParameters map[string]string `json:"AggregationFunctionParameters,omitempty"`
	AggregationPartitionBy []AggregationPartitionBy `json:"AggregationPartitionBy,omitempty"`
	Anchor *Anchor `json:"Anchor,omitempty"`
	Constant *TopicConstantValue `json:"Constant,omitempty"`
	FilterClass *string `json:"FilterClass,omitempty"`
	FilterType *string `json:"FilterType,omitempty"`
	Function *string `json:"Function,omitempty"`
	Inclusive bool `json:"Inclusive,omitempty"`
	Inverse bool `json:"Inverse,omitempty"`
	LastNextOffset *TopicConstantValue `json:"LastNextOffset,omitempty"`
	NullFilter *string `json:"NullFilter,omitempty"`
	OperandField *Identifier `json:"OperandField,omitempty"`
	Range *TopicConstantValue `json:"Range,omitempty"`
	SortDirection *string `json:"SortDirection,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
	TopBottomLimit *TopicConstantValue `json:"TopBottomLimit,omitempty"`
}

type TopicIRGroupBy struct {
	DisplayFormat *string `json:"DisplayFormat,omitempty"`
	DisplayFormatOptions *DisplayFormatOptions `json:"DisplayFormatOptions,omitempty"`
	FieldName *Identifier `json:"FieldName,omitempty"`
	NamedEntity *NamedEntityRef `json:"NamedEntity,omitempty"`
	Sort *TopicSortClause `json:"Sort,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type TopicIRMetric struct {
	CalculatedFieldReferences []Identifier `json:"CalculatedFieldReferences,omitempty"`
	ComparisonMethod *TopicIRComparisonMethod `json:"ComparisonMethod,omitempty"`
	DisplayFormat *string `json:"DisplayFormat,omitempty"`
	DisplayFormatOptions *DisplayFormatOptions `json:"DisplayFormatOptions,omitempty"`
	Expression *string `json:"Expression,omitempty"`
	Function *AggFunction `json:"Function,omitempty"`
	MetricId *Identifier `json:"MetricId,omitempty"`
	NamedEntity *NamedEntityRef `json:"NamedEntity,omitempty"`
	Operands []Identifier `json:"Operands,omitempty"`
}

type TopicNamedEntity struct {
	Definition []NamedEntityDefinition `json:"Definition,omitempty"`
	EntityDescription *string `json:"EntityDescription,omitempty"`
	EntityName string `json:"EntityName,omitempty"`
	EntitySynonyms []string `json:"EntitySynonyms,omitempty"`
	SemanticEntityType *SemanticEntityType `json:"SemanticEntityType,omitempty"`
}

type TopicNullFilter struct {
	Constant *TopicSingularFilterConstant `json:"Constant,omitempty"`
	Inverse bool `json:"Inverse,omitempty"`
	NullFilterType *string `json:"NullFilterType,omitempty"`
}

type TopicNumericEqualityFilter struct {
	Aggregation *string `json:"Aggregation,omitempty"`
	Constant *TopicSingularFilterConstant `json:"Constant,omitempty"`
}

type TopicNumericRangeFilter struct {
	Aggregation *string `json:"Aggregation,omitempty"`
	Constant *TopicRangeFilterConstant `json:"Constant,omitempty"`
	Inclusive bool `json:"Inclusive,omitempty"`
}

type TopicRangeFilterConstant struct {
	ConstantType *string `json:"ConstantType,omitempty"`
	RangeConstant *RangeConstant `json:"RangeConstant,omitempty"`
}

type TopicRefreshDetails struct {
	RefreshArn *string `json:"RefreshArn,omitempty"`
	RefreshId *string `json:"RefreshId,omitempty"`
	RefreshStatus *string `json:"RefreshStatus,omitempty"`
}

type TopicRefreshSchedule struct {
	BasedOnSpiceSchedule bool `json:"BasedOnSpiceSchedule,omitempty"`
	IsEnabled bool `json:"IsEnabled,omitempty"`
	RepeatAt *string `json:"RepeatAt,omitempty"`
	StartingAt *time.Time `json:"StartingAt,omitempty"`
	Timezone *string `json:"Timezone,omitempty"`
	TopicScheduleType *string `json:"TopicScheduleType,omitempty"`
}

type TopicRefreshScheduleSummary struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
	DatasetId *string `json:"DatasetId,omitempty"`
	DatasetName *string `json:"DatasetName,omitempty"`
	RefreshSchedule *TopicRefreshSchedule `json:"RefreshSchedule,omitempty"`
}

type TopicRelativeDateFilter struct {
	Constant *TopicSingularFilterConstant `json:"Constant,omitempty"`
	RelativeDateFilterFunction *string `json:"RelativeDateFilterFunction,omitempty"`
	TimeGranularity *string `json:"TimeGranularity,omitempty"`
}

type TopicReviewedAnswer struct {
	AnswerId string `json:"AnswerId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	DatasetArn string `json:"DatasetArn,omitempty"`
	Mir *TopicIR `json:"Mir,omitempty"`
	PrimaryVisual *TopicVisual `json:"PrimaryVisual,omitempty"`
	Question string `json:"Question,omitempty"`
	Template *TopicTemplate `json:"Template,omitempty"`
}

type TopicSearchFilter struct {
	Name string `json:"Name,omitempty"`
	Operator string `json:"Operator,omitempty"`
	Value string `json:"Value,omitempty"`
}

type TopicSingularFilterConstant struct {
	ConstantType *string `json:"ConstantType,omitempty"`
	SingularConstant *string `json:"SingularConstant,omitempty"`
}

type TopicSortClause struct {
	Operand *Identifier `json:"Operand,omitempty"`
	SortDirection *string `json:"SortDirection,omitempty"`
}

type TopicSummary struct {
	Arn *string `json:"Arn,omitempty"`
	Name *string `json:"Name,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
	UserExperienceVersion *string `json:"UserExperienceVersion,omitempty"`
}

type TopicTemplate struct {
	Slots []Slot `json:"Slots,omitempty"`
	TemplateType *string `json:"TemplateType,omitempty"`
}

type TopicVisual struct {
	Ir *TopicIR `json:"Ir,omitempty"`
	Role *string `json:"Role,omitempty"`
	SupportingVisuals []TopicVisual `json:"SupportingVisuals,omitempty"`
	VisualId *string `json:"VisualId,omitempty"`
}

type TotalAggregationComputation struct {
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
	Value *MeasureField `json:"Value,omitempty"`
}

type TotalAggregationFunction struct {
	SimpleTotalAggregationFunction *string `json:"SimpleTotalAggregationFunction,omitempty"`
}

type TotalAggregationOption struct {
	FieldId string `json:"FieldId,omitempty"`
	TotalAggregationFunction TotalAggregationFunction `json:"TotalAggregationFunction,omitempty"`
}

type TotalOptions struct {
	CustomLabel *string `json:"CustomLabel,omitempty"`
	Placement *string `json:"Placement,omitempty"`
	ScrollStatus *string `json:"ScrollStatus,omitempty"`
	TotalAggregationOptions []TotalAggregationOption `json:"TotalAggregationOptions,omitempty"`
	TotalCellStyle *TableCellStyle `json:"TotalCellStyle,omitempty"`
	TotalsVisibility *string `json:"TotalsVisibility,omitempty"`
}

type TransformOperation struct {
	CastColumnTypeOperation *CastColumnTypeOperation `json:"CastColumnTypeOperation,omitempty"`
	CreateColumnsOperation *CreateColumnsOperation `json:"CreateColumnsOperation,omitempty"`
	FilterOperation *FilterOperation `json:"FilterOperation,omitempty"`
	OverrideDatasetParameterOperation *OverrideDatasetParameterOperation `json:"OverrideDatasetParameterOperation,omitempty"`
	ProjectOperation *ProjectOperation `json:"ProjectOperation,omitempty"`
	RenameColumnOperation *RenameColumnOperation `json:"RenameColumnOperation,omitempty"`
	TagColumnOperation *TagColumnOperation `json:"TagColumnOperation,omitempty"`
	UntagColumnOperation *UntagColumnOperation `json:"UntagColumnOperation,omitempty"`
}

type TransformOperationSource struct {
	ColumnIdMappings []DataSetColumnIdMapping `json:"ColumnIdMappings,omitempty"`
	TransformOperationId string `json:"TransformOperationId,omitempty"`
}

type TransformStep struct {
	AggregateStep *AggregateOperation `json:"AggregateStep,omitempty"`
	AppendStep *AppendOperation `json:"AppendStep,omitempty"`
	CastColumnTypesStep *CastColumnTypesOperation `json:"CastColumnTypesStep,omitempty"`
	CreateColumnsStep *CreateColumnsOperation `json:"CreateColumnsStep,omitempty"`
	FiltersStep *FiltersOperation `json:"FiltersStep,omitempty"`
	ImportTableStep *ImportTableOperation `json:"ImportTableStep,omitempty"`
	JoinStep *JoinOperation `json:"JoinStep,omitempty"`
	PivotStep *PivotOperation `json:"PivotStep,omitempty"`
	ProjectStep *ProjectOperation `json:"ProjectStep,omitempty"`
	RenameColumnsStep *RenameColumnsOperation `json:"RenameColumnsStep,omitempty"`
	UnpivotStep *UnpivotOperation `json:"UnpivotStep,omitempty"`
}

type TransposedTableOption struct {
	ColumnIndex int `json:"ColumnIndex,omitempty"`
	ColumnType string `json:"ColumnType,omitempty"`
	ColumnWidth *string `json:"ColumnWidth,omitempty"`
}

type TreeMapAggregatedFieldWells struct {
	Colors []MeasureField `json:"Colors,omitempty"`
	Groups []DimensionField `json:"Groups,omitempty"`
	Sizes []MeasureField `json:"Sizes,omitempty"`
}

type TreeMapConfiguration struct {
	ColorLabelOptions *ChartAxisLabelOptions `json:"ColorLabelOptions,omitempty"`
	ColorScale *ColorScale `json:"ColorScale,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *TreeMapFieldWells `json:"FieldWells,omitempty"`
	GroupLabelOptions *ChartAxisLabelOptions `json:"GroupLabelOptions,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	SizeLabelOptions *ChartAxisLabelOptions `json:"SizeLabelOptions,omitempty"`
	SortConfiguration *TreeMapSortConfiguration `json:"SortConfiguration,omitempty"`
	Tooltip *TooltipOptions `json:"Tooltip,omitempty"`
}

type TreeMapFieldWells struct {
	TreeMapAggregatedFieldWells *TreeMapAggregatedFieldWells `json:"TreeMapAggregatedFieldWells,omitempty"`
}

type TreeMapSortConfiguration struct {
	TreeMapGroupItemsLimitConfiguration *ItemsLimitConfiguration `json:"TreeMapGroupItemsLimitConfiguration,omitempty"`
	TreeMapSort []FieldSortOptions `json:"TreeMapSort,omitempty"`
}

type TreeMapVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *TreeMapConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type TrendArrowOptions struct {
	Visibility *string `json:"Visibility,omitempty"`
}

type TrinoParameters struct {
	Catalog string `json:"Catalog,omitempty"`
	Host string `json:"Host,omitempty"`
	Port int `json:"Port,omitempty"`
}

type TwitterParameters struct {
	MaxRows int `json:"MaxRows,omitempty"`
	Query string `json:"Query,omitempty"`
}

type Typography struct {
	AxisLabelFontConfiguration *FontConfiguration `json:"AxisLabelFontConfiguration,omitempty"`
	AxisTitleFontConfiguration *FontConfiguration `json:"AxisTitleFontConfiguration,omitempty"`
	DataLabelFontConfiguration *FontConfiguration `json:"DataLabelFontConfiguration,omitempty"`
	FontFamilies []Font `json:"FontFamilies,omitempty"`
	LegendTitleFontConfiguration *FontConfiguration `json:"LegendTitleFontConfiguration,omitempty"`
	LegendValueFontConfiguration *FontConfiguration `json:"LegendValueFontConfiguration,omitempty"`
	VisualSubtitleFontConfiguration *VisualSubtitleFontConfiguration `json:"VisualSubtitleFontConfiguration,omitempty"`
	VisualTitleFontConfiguration *VisualTitleFontConfiguration `json:"VisualTitleFontConfiguration,omitempty"`
}

type UIColorPalette struct {
	Accent *string `json:"Accent,omitempty"`
	AccentForeground *string `json:"AccentForeground,omitempty"`
	Danger *string `json:"Danger,omitempty"`
	DangerForeground *string `json:"DangerForeground,omitempty"`
	Dimension *string `json:"Dimension,omitempty"`
	DimensionForeground *string `json:"DimensionForeground,omitempty"`
	Measure *string `json:"Measure,omitempty"`
	MeasureForeground *string `json:"MeasureForeground,omitempty"`
	PrimaryBackground *string `json:"PrimaryBackground,omitempty"`
	PrimaryForeground *string `json:"PrimaryForeground,omitempty"`
	SecondaryBackground *string `json:"SecondaryBackground,omitempty"`
	SecondaryForeground *string `json:"SecondaryForeground,omitempty"`
	Success *string `json:"Success,omitempty"`
	SuccessForeground *string `json:"SuccessForeground,omitempty"`
	Warning *string `json:"Warning,omitempty"`
	WarningForeground *string `json:"WarningForeground,omitempty"`
}

type UnaggregatedField struct {
	Column ColumnIdentifier `json:"Column,omitempty"`
	FieldId string `json:"FieldId,omitempty"`
	FormatConfiguration *FormatConfiguration `json:"FormatConfiguration,omitempty"`
}

type UniqueKey struct {
	ColumnNames []string `json:"ColumnNames,omitempty"`
}

type UniqueValuesComputation struct {
	Category *DimensionField `json:"Category,omitempty"`
	ComputationId string `json:"ComputationId,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type UnpivotOperation struct {
	Alias string `json:"Alias,omitempty"`
	ColumnsToUnpivot []ColumnToUnpivot `json:"ColumnsToUnpivot,omitempty"`
	Source TransformOperationSource `json:"Source,omitempty"`
	UnpivotedLabelColumnId string `json:"UnpivotedLabelColumnId,omitempty"`
	UnpivotedLabelColumnName string `json:"UnpivotedLabelColumnName,omitempty"`
	UnpivotedValueColumnId string `json:"UnpivotedValueColumnId,omitempty"`
	UnpivotedValueColumnName string `json:"UnpivotedValueColumnName,omitempty"`
}

type UntagColumnOperation struct {
	ColumnName string `json:"ColumnName,omitempty"`
	TagNames []string `json:"TagNames,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	TagKeys []string `json:"keys,omitempty"`
}

type UntagResourceResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateAccountCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
}

type UpdateAccountCustomPermissionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateAccountCustomizationRequest struct {
	AccountCustomization AccountCustomization `json:"AccountCustomization,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type UpdateAccountCustomizationResponse struct {
	AccountCustomization *AccountCustomization `json:"AccountCustomization,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"Namespace,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateAccountSettingsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DefaultNamespace string `json:"DefaultNamespace,omitempty"`
	NotificationEmail *string `json:"NotificationEmail,omitempty"`
	TerminationProtectionEnabled bool `json:"TerminationProtectionEnabled,omitempty"`
}

type UpdateAccountSettingsResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateActionConnectorPermissionsRequest struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
}

type UpdateActionConnectorPermissionsResponse struct {
	ActionConnectorId *string `json:"ActionConnectorId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateActionConnectorRequest struct {
	ActionConnectorId string `json:"ActionConnectorId,omitempty"`
	AuthenticationConfig AuthConfig `json:"AuthenticationConfig,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Description *string `json:"Description,omitempty"`
	Name string `json:"Name,omitempty"`
	VpcConnectionArn *string `json:"VpcConnectionArn,omitempty"`
}

type UpdateActionConnectorResponse struct {
	ActionConnectorId *string `json:"ActionConnectorId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
}

type UpdateAnalysisPermissionsRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
}

type UpdateAnalysisPermissionsResponse struct {
	AnalysisArn *string `json:"AnalysisArn,omitempty"`
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateAnalysisRequest struct {
	AnalysisId string `json:"AnalysisId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Definition *AnalysisDefinition `json:"Definition,omitempty"`
	Name string `json:"Name,omitempty"`
	Parameters *Parameters `json:"Parameters,omitempty"`
	SourceEntity *AnalysisSourceEntity `json:"SourceEntity,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	ValidationStrategy *ValidationStrategy `json:"ValidationStrategy,omitempty"`
}

type UpdateAnalysisResponse struct {
	AnalysisId *string `json:"AnalysisId,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
}

type UpdateApplicationWithTokenExchangeGrantRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type UpdateApplicationWithTokenExchangeGrantResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateBrandAssignmentRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandArn string `json:"BrandArn,omitempty"`
}

type UpdateBrandAssignmentResponse struct {
	BrandArn *string `json:"BrandArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
}

type UpdateBrandPublishedVersionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
	VersionId string `json:"VersionId,omitempty"`
}

type UpdateBrandPublishedVersionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	VersionId *string `json:"VersionId,omitempty"`
}

type UpdateBrandRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BrandDefinition *BrandDefinition `json:"BrandDefinition,omitempty"`
	BrandId string `json:"BrandId,omitempty"`
}

type UpdateBrandResponse struct {
	BrandDefinition *BrandDefinition `json:"BrandDefinition,omitempty"`
	BrandDetail *BrandDetail `json:"BrandDetail,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
}

type UpdateCustomPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Capabilities *Capabilities `json:"Capabilities,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
}

type UpdateCustomPermissionsResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDashboardLinksRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	LinkEntities []string `json:"LinkEntities,omitempty"`
}

type UpdateDashboardLinksResponse struct {
	DashboardArn *string `json:"DashboardArn,omitempty"`
	LinkEntities []string `json:"LinkEntities,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDashboardPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	GrantLinkPermissions []ResourcePermission `json:"GrantLinkPermissions,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokeLinkPermissions []ResourcePermission `json:"RevokeLinkPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
}

type UpdateDashboardPermissionsResponse struct {
	DashboardArn *string `json:"DashboardArn,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	LinkSharingConfiguration *LinkSharingConfiguration `json:"LinkSharingConfiguration,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDashboardPublishedVersionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	VersionNumber int64 `json:"VersionNumber,omitempty"`
}

type UpdateDashboardPublishedVersionResponse struct {
	DashboardArn *string `json:"DashboardArn,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDashboardRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardId string `json:"DashboardId,omitempty"`
	DashboardPublishOptions *DashboardPublishOptions `json:"DashboardPublishOptions,omitempty"`
	Definition *DashboardVersionDefinition `json:"Definition,omitempty"`
	Name string `json:"Name,omitempty"`
	Parameters *Parameters `json:"Parameters,omitempty"`
	SourceEntity *DashboardSourceEntity `json:"SourceEntity,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	ValidationStrategy *ValidationStrategy `json:"ValidationStrategy,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type UpdateDashboardResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	DashboardId *string `json:"DashboardId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	VersionArn *string `json:"VersionArn,omitempty"`
}

type UpdateDashboardsQAConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DashboardsQAStatus string `json:"DashboardsQAStatus,omitempty"`
}

type UpdateDashboardsQAConfigurationResponse struct {
	DashboardsQAStatus *string `json:"DashboardsQAStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDataSetPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
}

type UpdateDataSetPermissionsResponse struct {
	DataSetArn *string `json:"DataSetArn,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDataSetRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ColumnGroups []ColumnGroup `json:"ColumnGroups,omitempty"`
	ColumnLevelPermissionRules []ColumnLevelPermissionRule `json:"ColumnLevelPermissionRules,omitempty"`
	DataPrepConfiguration *DataPrepConfiguration `json:"DataPrepConfiguration,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	DataSetUsageConfiguration *DataSetUsageConfiguration `json:"DataSetUsageConfiguration,omitempty"`
	DatasetParameters []DatasetParameter `json:"DatasetParameters,omitempty"`
	FieldFolders map[string]FieldFolder `json:"FieldFolders,omitempty"`
	ImportMode string `json:"ImportMode,omitempty"`
	LogicalTableMap map[string]LogicalTable `json:"LogicalTableMap,omitempty"`
	Name string `json:"Name,omitempty"`
	PerformanceConfiguration *PerformanceConfiguration `json:"PerformanceConfiguration,omitempty"`
	PhysicalTableMap map[string]PhysicalTable `json:"PhysicalTableMap,omitempty"`
	RowLevelPermissionDataSet *RowLevelPermissionDataSet `json:"RowLevelPermissionDataSet,omitempty"`
	RowLevelPermissionTagConfiguration *RowLevelPermissionTagConfiguration `json:"RowLevelPermissionTagConfiguration,omitempty"`
	SemanticModelConfiguration *SemanticModelConfiguration `json:"SemanticModelConfiguration,omitempty"`
}

type UpdateDataSetResponse struct {
	Arn *string `json:"Arn,omitempty"`
	DataSetId *string `json:"DataSetId,omitempty"`
	IngestionArn *string `json:"IngestionArn,omitempty"`
	IngestionId *string `json:"IngestionId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDataSourcePermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
}

type UpdateDataSourcePermissionsResponse struct {
	DataSourceArn *string `json:"DataSourceArn,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateDataSourceRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Credentials *DataSourceCredentials `json:"Credentials,omitempty"`
	DataSourceId string `json:"DataSourceId,omitempty"`
	DataSourceParameters *DataSourceParameters `json:"DataSourceParameters,omitempty"`
	Name string `json:"Name,omitempty"`
	SslProperties *SslProperties `json:"SslProperties,omitempty"`
	VpcConnectionProperties *VpcConnectionProperties `json:"VpcConnectionProperties,omitempty"`
}

type UpdateDataSourceResponse struct {
	Arn *string `json:"Arn,omitempty"`
	DataSourceId *string `json:"DataSourceId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
}

type UpdateDefaultQBusinessApplicationRequest struct {
	ApplicationId string `json:"ApplicationId,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

type UpdateDefaultQBusinessApplicationResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateFlowPermissionsInput struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
	GrantPermissions []Permission `json:"GrantPermissions,omitempty"`
	RevokePermissions []Permission `json:"RevokePermissions,omitempty"`
}

type UpdateFlowPermissionsOutput struct {
	Arn string `json:"Arn,omitempty"`
	FlowId string `json:"FlowId,omitempty"`
	Permissions []Permission `json:"Permissions,omitempty"`
	RequestId string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateFolderPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
}

type UpdateFolderPermissionsResponse struct {
	Arn *string `json:"Arn,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateFolderRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	FolderId string `json:"FolderId,omitempty"`
	Name string `json:"Name,omitempty"`
}

type UpdateFolderResponse struct {
	Arn *string `json:"Arn,omitempty"`
	FolderId *string `json:"FolderId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateGroupRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Description *string `json:"Description,omitempty"`
	GroupName string `json:"GroupName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
}

type UpdateGroupResponse struct {
	Group *Group `json:"Group,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateIAMPolicyAssignmentRequest struct {
	AssignmentName string `json:"AssignmentName,omitempty"`
	AssignmentStatus *string `json:"AssignmentStatus,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Identities map[string][]string `json:"Identities,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	PolicyArn *string `json:"PolicyArn,omitempty"`
}

type UpdateIAMPolicyAssignmentResponse struct {
	AssignmentId *string `json:"AssignmentId,omitempty"`
	AssignmentName *string `json:"AssignmentName,omitempty"`
	AssignmentStatus *string `json:"AssignmentStatus,omitempty"`
	Identities map[string][]string `json:"Identities,omitempty"`
	PolicyArn *string `json:"PolicyArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateIdentityPropagationConfigRequest struct {
	AuthorizedTargets []string `json:"AuthorizedTargets,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ServiceModel string `json:"Service,omitempty"`
}

type UpdateIdentityPropagationConfigResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateIpRestrictionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	IpRestrictionRuleMap map[string]string `json:"IpRestrictionRuleMap,omitempty"`
	VpcEndpointIdRestrictionRuleMap map[string]string `json:"VpcEndpointIdRestrictionRuleMap,omitempty"`
	VpcIdRestrictionRuleMap map[string]string `json:"VpcIdRestrictionRuleMap,omitempty"`
}

type UpdateIpRestrictionResponse struct {
	AwsAccountId *string `json:"AwsAccountId,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateKeyRegistrationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	KeyRegistration []RegisteredCustomerManagedKey `json:"KeyRegistration,omitempty"`
}

type UpdateKeyRegistrationResponse struct {
	FailedKeyRegistration []FailedKeyRegistrationEntry `json:"FailedKeyRegistration,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	SuccessfulKeyRegistration []SuccessfulKeyRegistrationEntry `json:"SuccessfulKeyRegistration,omitempty"`
}

type UpdatePublicSharingSettingsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	PublicSharingEnabled bool `json:"PublicSharingEnabled,omitempty"`
}

type UpdatePublicSharingSettingsResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateQPersonalizationConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	PersonalizationMode string `json:"PersonalizationMode,omitempty"`
}

type UpdateQPersonalizationConfigurationResponse struct {
	PersonalizationMode *string `json:"PersonalizationMode,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateQuickSightQSearchConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	QSearchStatus string `json:"QSearchStatus,omitempty"`
}

type UpdateQuickSightQSearchConfigurationResponse struct {
	QSearchStatus *string `json:"QSearchStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DataSetId string `json:"DataSetId,omitempty"`
	Schedule RefreshSchedule `json:"Schedule,omitempty"`
}

type UpdateRefreshScheduleResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	ScheduleId *string `json:"ScheduleId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateRoleCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Role string `json:"Role,omitempty"`
}

type UpdateRoleCustomPermissionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateSPICECapacityConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	PurchaseMode string `json:"PurchaseMode,omitempty"`
}

type UpdateSPICECapacityConfigurationResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateSelfUpgradeConfigurationRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	SelfUpgradeStatus string `json:"SelfUpgradeStatus,omitempty"`
}

type UpdateSelfUpgradeConfigurationResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateSelfUpgradeRequest struct {
	Action string `json:"Action,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	UpgradeRequestId string `json:"UpgradeRequestId,omitempty"`
}

type UpdateSelfUpgradeResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	SelfUpgradeRequestDetail *SelfUpgradeRequestDetail `json:"SelfUpgradeRequestDetail,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateTemplateAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	TemplateVersionNumber int64 `json:"TemplateVersionNumber,omitempty"`
}

type UpdateTemplateAliasResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateAlias *TemplateAlias `json:"TemplateAlias,omitempty"`
}

type UpdateTemplatePermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
}

type UpdateTemplatePermissionsResponse struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateArn *string `json:"TemplateArn,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
}

type UpdateTemplateRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	Definition *TemplateVersionDefinition `json:"Definition,omitempty"`
	Name *string `json:"Name,omitempty"`
	SourceEntity *TemplateSourceEntity `json:"SourceEntity,omitempty"`
	TemplateId string `json:"TemplateId,omitempty"`
	ValidationStrategy *ValidationStrategy `json:"ValidationStrategy,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type UpdateTemplateResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TemplateId *string `json:"TemplateId,omitempty"`
	VersionArn *string `json:"VersionArn,omitempty"`
}

type UpdateThemeAliasRequest struct {
	AliasName string `json:"AliasName,omitempty"`
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
	ThemeVersionNumber int64 `json:"ThemeVersionNumber,omitempty"`
}

type UpdateThemeAliasResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeAlias *ThemeAlias `json:"ThemeAlias,omitempty"`
}

type UpdateThemePermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
}

type UpdateThemePermissionsResponse struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeArn *string `json:"ThemeArn,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
}

type UpdateThemeRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	BaseThemeId string `json:"BaseThemeId,omitempty"`
	Configuration *ThemeConfiguration `json:"Configuration,omitempty"`
	Name *string `json:"Name,omitempty"`
	ThemeId string `json:"ThemeId,omitempty"`
	VersionDescription *string `json:"VersionDescription,omitempty"`
}

type UpdateThemeResponse struct {
	Arn *string `json:"Arn,omitempty"`
	CreationStatus *string `json:"CreationStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	ThemeId *string `json:"ThemeId,omitempty"`
	VersionArn *string `json:"VersionArn,omitempty"`
}

type UpdateTopicPermissionsRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	GrantPermissions []ResourcePermission `json:"GrantPermissions,omitempty"`
	RevokePermissions []ResourcePermission `json:"RevokePermissions,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type UpdateTopicPermissionsResponse struct {
	Permissions []ResourcePermission `json:"Permissions,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type UpdateTopicRefreshScheduleRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DatasetId string `json:"DatasetId,omitempty"`
	RefreshSchedule TopicRefreshSchedule `json:"RefreshSchedule,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type UpdateTopicRefreshScheduleResponse struct {
	DatasetArn *string `json:"DatasetArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicArn *string `json:"TopicArn,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type UpdateTopicRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomInstructions *CustomInstructions `json:"CustomInstructions,omitempty"`
	Topic TopicDetails `json:"Topic,omitempty"`
	TopicId string `json:"TopicId,omitempty"`
}

type UpdateTopicResponse struct {
	Arn *string `json:"Arn,omitempty"`
	RefreshArn *string `json:"RefreshArn,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	TopicId *string `json:"TopicId,omitempty"`
}

type UpdateUserCustomPermissionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomPermissionsName string `json:"CustomPermissionsName,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type UpdateUserCustomPermissionResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
}

type UpdateUserRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	CustomFederationProviderUrl *string `json:"CustomFederationProviderUrl,omitempty"`
	CustomPermissionsName *string `json:"CustomPermissionsName,omitempty"`
	Email string `json:"Email,omitempty"`
	ExternalLoginFederationProviderType *string `json:"ExternalLoginFederationProviderType,omitempty"`
	ExternalLoginId *string `json:"ExternalLoginId,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Role string `json:"Role,omitempty"`
	UnapplyCustomPermissions bool `json:"UnapplyCustomPermissions,omitempty"`
	UserName string `json:"UserName,omitempty"`
}

type UpdateUserResponse struct {
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	User *User `json:"User,omitempty"`
}

type UpdateVPCConnectionRequest struct {
	AwsAccountId string `json:"AwsAccountId,omitempty"`
	DnsResolvers []string `json:"DnsResolvers,omitempty"`
	Name string `json:"Name,omitempty"`
	RoleArn string `json:"RoleArn,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	SubnetIds []string `json:"SubnetIds,omitempty"`
	VPCConnectionId string `json:"VPCConnectionId,omitempty"`
}

type UpdateVPCConnectionResponse struct {
	Arn *string `json:"Arn,omitempty"`
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
	RequestId *string `json:"RequestId,omitempty"`
	Status int `json:"Status,omitempty"`
	UpdateStatus *string `json:"UpdateStatus,omitempty"`
	VPCConnectionId *string `json:"VPCConnectionId,omitempty"`
}

type UploadSettings struct {
	ContainsHeader bool `json:"ContainsHeader,omitempty"`
	CustomCellAddressRange *string `json:"CustomCellAddressRange,omitempty"`
	Delimiter *string `json:"Delimiter,omitempty"`
	Format *string `json:"Format,omitempty"`
	StartFromRow int `json:"StartFromRow,omitempty"`
	TextQualifier *string `json:"TextQualifier,omitempty"`
}

type User struct {
	Active bool `json:"Active,omitempty"`
	Arn *string `json:"Arn,omitempty"`
	CustomPermissionsName *string `json:"CustomPermissionsName,omitempty"`
	Email *string `json:"Email,omitempty"`
	ExternalLoginFederationProviderType *string `json:"ExternalLoginFederationProviderType,omitempty"`
	ExternalLoginFederationProviderUrl *string `json:"ExternalLoginFederationProviderUrl,omitempty"`
	ExternalLoginId *string `json:"ExternalLoginId,omitempty"`
	IdentityType *string `json:"IdentityType,omitempty"`
	PrincipalId *string `json:"PrincipalId,omitempty"`
	Role *string `json:"Role,omitempty"`
	UserName *string `json:"UserName,omitempty"`
}

type UserIdentifier struct {
	Email *string `json:"Email,omitempty"`
	UserArn *string `json:"UserArn,omitempty"`
	UserName *string `json:"UserName,omitempty"`
}

type VPCConnection struct {
	Arn *string `json:"Arn,omitempty"`
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DnsResolvers []string `json:"DnsResolvers,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"NetworkInterfaces,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	Status *string `json:"Status,omitempty"`
	VPCConnectionId *string `json:"VPCConnectionId,omitempty"`
	VPCId *string `json:"VPCId,omitempty"`
}

type VPCConnectionSummary struct {
	Arn *string `json:"Arn,omitempty"`
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DnsResolvers []string `json:"DnsResolvers,omitempty"`
	LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"NetworkInterfaces,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`
	Status *string `json:"Status,omitempty"`
	VPCConnectionId *string `json:"VPCConnectionId,omitempty"`
	VPCId *string `json:"VPCId,omitempty"`
}

type ValidationStrategy struct {
	Mode string `json:"Mode,omitempty"`
}

type ValueColumnConfiguration struct {
	AggregationFunction *DataPrepAggregationFunction `json:"AggregationFunction,omitempty"`
}

type VisibleRangeOptions struct {
	PercentRange *PercentVisibleRange `json:"PercentRange,omitempty"`
}

type Visual struct {
	BarChartVisual *BarChartVisual `json:"BarChartVisual,omitempty"`
	BoxPlotVisual *BoxPlotVisual `json:"BoxPlotVisual,omitempty"`
	ComboChartVisual *ComboChartVisual `json:"ComboChartVisual,omitempty"`
	CustomContentVisual *CustomContentVisual `json:"CustomContentVisual,omitempty"`
	EmptyVisual *EmptyVisual `json:"EmptyVisual,omitempty"`
	FilledMapVisual *FilledMapVisual `json:"FilledMapVisual,omitempty"`
	FunnelChartVisual *FunnelChartVisual `json:"FunnelChartVisual,omitempty"`
	GaugeChartVisual *GaugeChartVisual `json:"GaugeChartVisual,omitempty"`
	GeospatialMapVisual *GeospatialMapVisual `json:"GeospatialMapVisual,omitempty"`
	HeatMapVisual *HeatMapVisual `json:"HeatMapVisual,omitempty"`
	HistogramVisual *HistogramVisual `json:"HistogramVisual,omitempty"`
	InsightVisual *InsightVisual `json:"InsightVisual,omitempty"`
	KPIVisual *KPIVisual `json:"KPIVisual,omitempty"`
	LayerMapVisual *LayerMapVisual `json:"LayerMapVisual,omitempty"`
	LineChartVisual *LineChartVisual `json:"LineChartVisual,omitempty"`
	PieChartVisual *PieChartVisual `json:"PieChartVisual,omitempty"`
	PivotTableVisual *PivotTableVisual `json:"PivotTableVisual,omitempty"`
	PluginVisual *PluginVisual `json:"PluginVisual,omitempty"`
	RadarChartVisual *RadarChartVisual `json:"RadarChartVisual,omitempty"`
	SankeyDiagramVisual *SankeyDiagramVisual `json:"SankeyDiagramVisual,omitempty"`
	ScatterPlotVisual *ScatterPlotVisual `json:"ScatterPlotVisual,omitempty"`
	TableVisual *TableVisual `json:"TableVisual,omitempty"`
	TreeMapVisual *TreeMapVisual `json:"TreeMapVisual,omitempty"`
	WaterfallVisual *WaterfallVisual `json:"WaterfallVisual,omitempty"`
	WordCloudVisual *WordCloudVisual `json:"WordCloudVisual,omitempty"`
}

type VisualAxisSortOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type VisualCustomAction struct {
	ActionOperations []VisualCustomActionOperation `json:"ActionOperations,omitempty"`
	CustomActionId string `json:"CustomActionId,omitempty"`
	Name string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
	Trigger string `json:"Trigger,omitempty"`
}

type VisualCustomActionDefaults struct {
	HighlightOperation *VisualHighlightOperation `json:"highlightOperation,omitempty"`
}

type VisualCustomActionOperation struct {
	FilterOperation *CustomActionFilterOperation `json:"FilterOperation,omitempty"`
	NavigationOperation *CustomActionNavigationOperation `json:"NavigationOperation,omitempty"`
	SetParametersOperation *CustomActionSetParametersOperation `json:"SetParametersOperation,omitempty"`
	URLOperation *CustomActionURLOperation `json:"URLOperation,omitempty"`
}

type VisualCustomizationFieldsConfiguration struct {
	AdditionalFields []ColumnIdentifier `json:"AdditionalFields,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type VisualHighlightOperation struct {
	Trigger string `json:"Trigger,omitempty"`
}

type VisualInteractionOptions struct {
	ContextMenuOption *ContextMenuOption `json:"ContextMenuOption,omitempty"`
	VisualMenuOption *VisualMenuOption `json:"VisualMenuOption,omitempty"`
}

type VisualMenuOption struct {
	AvailabilityStatus *string `json:"AvailabilityStatus,omitempty"`
}

type VisualOptions struct {
	Type *string `json:"type,omitempty"`
}

type VisualPalette struct {
	ChartColor *string `json:"ChartColor,omitempty"`
	ColorMap []DataPathColor `json:"ColorMap,omitempty"`
}

type VisualSubtitleFontConfiguration struct {
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
	TextAlignment *string `json:"TextAlignment,omitempty"`
	TextTransform *string `json:"TextTransform,omitempty"`
}

type VisualSubtitleLabelOptions struct {
	FormatText *LongFormatText `json:"FormatText,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type VisualTitleFontConfiguration struct {
	FontConfiguration *FontConfiguration `json:"FontConfiguration,omitempty"`
	TextAlignment *string `json:"TextAlignment,omitempty"`
	TextTransform *string `json:"TextTransform,omitempty"`
}

type VisualTitleLabelOptions struct {
	FormatText *ShortFormatText `json:"FormatText,omitempty"`
	Visibility *string `json:"Visibility,omitempty"`
}

type VpcConnectionProperties struct {
	VpcConnectionArn string `json:"VpcConnectionArn,omitempty"`
}

type WaterfallChartAggregatedFieldWells struct {
	Breakdowns []DimensionField `json:"Breakdowns,omitempty"`
	Categories []DimensionField `json:"Categories,omitempty"`
	Values []MeasureField `json:"Values,omitempty"`
}

type WaterfallChartColorConfiguration struct {
	GroupColorConfiguration *WaterfallChartGroupColorConfiguration `json:"GroupColorConfiguration,omitempty"`
}

type WaterfallChartConfiguration struct {
	CategoryAxisDisplayOptions *AxisDisplayOptions `json:"CategoryAxisDisplayOptions,omitempty"`
	CategoryAxisLabelOptions *ChartAxisLabelOptions `json:"CategoryAxisLabelOptions,omitempty"`
	ColorConfiguration *WaterfallChartColorConfiguration `json:"ColorConfiguration,omitempty"`
	DataLabels *DataLabelOptions `json:"DataLabels,omitempty"`
	FieldWells *WaterfallChartFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	Legend *LegendOptions `json:"Legend,omitempty"`
	PrimaryYAxisDisplayOptions *AxisDisplayOptions `json:"PrimaryYAxisDisplayOptions,omitempty"`
	PrimaryYAxisLabelOptions *ChartAxisLabelOptions `json:"PrimaryYAxisLabelOptions,omitempty"`
	SortConfiguration *WaterfallChartSortConfiguration `json:"SortConfiguration,omitempty"`
	VisualPalette *VisualPalette `json:"VisualPalette,omitempty"`
	WaterfallChartOptions *WaterfallChartOptions `json:"WaterfallChartOptions,omitempty"`
}

type WaterfallChartFieldWells struct {
	WaterfallChartAggregatedFieldWells *WaterfallChartAggregatedFieldWells `json:"WaterfallChartAggregatedFieldWells,omitempty"`
}

type WaterfallChartGroupColorConfiguration struct {
	NegativeBarColor *string `json:"NegativeBarColor,omitempty"`
	PositiveBarColor *string `json:"PositiveBarColor,omitempty"`
	TotalBarColor *string `json:"TotalBarColor,omitempty"`
}

type WaterfallChartOptions struct {
	TotalBarLabel *string `json:"TotalBarLabel,omitempty"`
}

type WaterfallChartSortConfiguration struct {
	BreakdownItemsLimit *ItemsLimitConfiguration `json:"BreakdownItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
}

type WaterfallVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *WaterfallChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type WebCrawlerParameters struct {
	LoginPageUrl *string `json:"LoginPageUrl,omitempty"`
	PasswordButtonXpath *string `json:"PasswordButtonXpath,omitempty"`
	PasswordFieldXpath *string `json:"PasswordFieldXpath,omitempty"`
	UsernameButtonXpath *string `json:"UsernameButtonXpath,omitempty"`
	UsernameFieldXpath *string `json:"UsernameFieldXpath,omitempty"`
	WebCrawlerAuthType string `json:"WebCrawlerAuthType,omitempty"`
	WebProxyHostName *string `json:"WebProxyHostName,omitempty"`
	WebProxyPortNumber int `json:"WebProxyPortNumber,omitempty"`
}

type WebProxyCredentials struct {
	WebProxyPassword string `json:"WebProxyPassword,omitempty"`
	WebProxyUsername string `json:"WebProxyUsername,omitempty"`
}

type WhatIfPointScenario struct {
	Date time.Time `json:"Date,omitempty"`
	Value float64 `json:"Value,omitempty"`
}

type WhatIfRangeScenario struct {
	EndDate time.Time `json:"EndDate,omitempty"`
	StartDate time.Time `json:"StartDate,omitempty"`
	Value float64 `json:"Value,omitempty"`
}

type WordCloudAggregatedFieldWells struct {
	GroupBy []DimensionField `json:"GroupBy,omitempty"`
	Size []MeasureField `json:"Size,omitempty"`
}

type WordCloudChartConfiguration struct {
	CategoryLabelOptions *ChartAxisLabelOptions `json:"CategoryLabelOptions,omitempty"`
	FieldWells *WordCloudFieldWells `json:"FieldWells,omitempty"`
	Interactions *VisualInteractionOptions `json:"Interactions,omitempty"`
	SortConfiguration *WordCloudSortConfiguration `json:"SortConfiguration,omitempty"`
	WordCloudOptions *WordCloudOptions `json:"WordCloudOptions,omitempty"`
}

type WordCloudFieldWells struct {
	WordCloudAggregatedFieldWells *WordCloudAggregatedFieldWells `json:"WordCloudAggregatedFieldWells,omitempty"`
}

type WordCloudOptions struct {
	CloudLayout *string `json:"CloudLayout,omitempty"`
	MaximumStringLength int `json:"MaximumStringLength,omitempty"`
	WordCasing *string `json:"WordCasing,omitempty"`
	WordOrientation *string `json:"WordOrientation,omitempty"`
	WordPadding *string `json:"WordPadding,omitempty"`
	WordScaling *string `json:"WordScaling,omitempty"`
}

type WordCloudSortConfiguration struct {
	CategoryItemsLimit *ItemsLimitConfiguration `json:"CategoryItemsLimit,omitempty"`
	CategorySort []FieldSortOptions `json:"CategorySort,omitempty"`
}

type WordCloudVisual struct {
	Actions []VisualCustomAction `json:"Actions,omitempty"`
	ChartConfiguration *WordCloudChartConfiguration `json:"ChartConfiguration,omitempty"`
	ColumnHierarchies []ColumnHierarchy `json:"ColumnHierarchies,omitempty"`
	Subtitle *VisualSubtitleLabelOptions `json:"Subtitle,omitempty"`
	Title *VisualTitleLabelOptions `json:"Title,omitempty"`
	VisualContentAltText *string `json:"VisualContentAltText,omitempty"`
	VisualId string `json:"VisualId,omitempty"`
}

type YAxisOptions struct {
	YAxis string `json:"YAxis,omitempty"`
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

func handleBatchCreateTopicReviewedAnswer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchCreateTopicReviewedAnswerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchCreateTopicReviewedAnswer business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchCreateTopicReviewedAnswer"})
}

func handleBatchDeleteTopicReviewedAnswer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req BatchDeleteTopicReviewedAnswerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement BatchDeleteTopicReviewedAnswer business logic
	return jsonOK(map[string]any{"status": "ok", "action": "BatchDeleteTopicReviewedAnswer"})
}

func handleCancelIngestion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CancelIngestionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CancelIngestion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CancelIngestion"})
}

func handleCreateAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAccountCustomizationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAccountCustomization business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAccountCustomization"})
}

func handleCreateAccountSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAccountSubscriptionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAccountSubscription business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAccountSubscription"})
}

func handleCreateActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateActionConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateActionConnector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateActionConnector"})
}

func handleCreateAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAnalysisRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAnalysis business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAnalysis"})
}

func handleCreateBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateBrandRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateBrand business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateBrand"})
}

func handleCreateCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCustomPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCustomPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCustomPermissions"})
}

func handleCreateDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDashboardRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDashboard business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDashboard"})
}

func handleCreateDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDataSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDataSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDataSet"})
}

func handleCreateDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateDataSource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateDataSource"})
}

func handleCreateFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFolderRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFolder business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFolder"})
}

func handleCreateFolderMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFolderMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFolderMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFolderMembership"})
}

func handleCreateGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateGroup"})
}

func handleCreateGroupMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateGroupMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateGroupMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateGroupMembership"})
}

func handleCreateIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateIAMPolicyAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateIAMPolicyAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateIAMPolicyAssignment"})
}

func handleCreateIngestion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateIngestionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateIngestion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateIngestion"})
}

func handleCreateNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateNamespace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateNamespace"})
}

func handleCreateRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateRefreshSchedule"})
}

func handleCreateRoleMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateRoleMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateRoleMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateRoleMembership"})
}

func handleCreateTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTemplateRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTemplate business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTemplate"})
}

func handleCreateTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTemplateAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTemplateAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTemplateAlias"})
}

func handleCreateTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateThemeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTheme business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTheme"})
}

func handleCreateThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateThemeAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateThemeAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateThemeAlias"})
}

func handleCreateTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTopicRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTopic business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTopic"})
}

func handleCreateTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTopicRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTopicRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTopicRefreshSchedule"})
}

func handleCreateVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateVPCConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateVPCConnection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateVPCConnection"})
}

func handleDeleteAccountCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAccountCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAccountCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAccountCustomPermission"})
}

func handleDeleteAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAccountCustomizationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAccountCustomization business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAccountCustomization"})
}

func handleDeleteAccountSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAccountSubscriptionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAccountSubscription business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAccountSubscription"})
}

func handleDeleteActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteActionConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteActionConnector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteActionConnector"})
}

func handleDeleteAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAnalysisRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAnalysis business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAnalysis"})
}

func handleDeleteBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteBrandRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteBrand business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteBrand"})
}

func handleDeleteBrandAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteBrandAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteBrandAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteBrandAssignment"})
}

func handleDeleteCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCustomPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCustomPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCustomPermissions"})
}

func handleDeleteDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDashboardRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDashboard business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDashboard"})
}

func handleDeleteDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDataSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDataSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDataSet"})
}

func handleDeleteDataSetRefreshProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDataSetRefreshPropertiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDataSetRefreshProperties business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDataSetRefreshProperties"})
}

func handleDeleteDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDataSource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDataSource"})
}

func handleDeleteDefaultQBusinessApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteDefaultQBusinessApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteDefaultQBusinessApplication business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteDefaultQBusinessApplication"})
}

func handleDeleteFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFolderRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFolder business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFolder"})
}

func handleDeleteFolderMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFolderMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFolderMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFolderMembership"})
}

func handleDeleteGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteGroup"})
}

func handleDeleteGroupMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteGroupMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteGroupMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteGroupMembership"})
}

func handleDeleteIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteIAMPolicyAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteIAMPolicyAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteIAMPolicyAssignment"})
}

func handleDeleteIdentityPropagationConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteIdentityPropagationConfigRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteIdentityPropagationConfig business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteIdentityPropagationConfig"})
}

func handleDeleteNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteNamespace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteNamespace"})
}

func handleDeleteRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteRefreshSchedule"})
}

func handleDeleteRoleCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteRoleCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteRoleCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteRoleCustomPermission"})
}

func handleDeleteRoleMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteRoleMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteRoleMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteRoleMembership"})
}

func handleDeleteTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTemplateRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTemplate business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTemplate"})
}

func handleDeleteTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTemplateAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTemplateAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTemplateAlias"})
}

func handleDeleteTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteThemeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTheme business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTheme"})
}

func handleDeleteThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteThemeAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteThemeAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteThemeAlias"})
}

func handleDeleteTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTopicRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTopic business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTopic"})
}

func handleDeleteTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTopicRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTopicRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTopicRefreshSchedule"})
}

func handleDeleteUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteUser"})
}

func handleDeleteUserByPrincipalId(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteUserByPrincipalIdRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteUserByPrincipalId business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteUserByPrincipalId"})
}

func handleDeleteUserCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteUserCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteUserCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteUserCustomPermission"})
}

func handleDeleteVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteVPCConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteVPCConnection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteVPCConnection"})
}

func handleDescribeAccountCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAccountCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccountCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccountCustomPermission"})
}

func handleDescribeAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAccountCustomizationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccountCustomization business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccountCustomization"})
}

func handleDescribeAccountSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAccountSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccountSettings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccountSettings"})
}

func handleDescribeAccountSubscription(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAccountSubscriptionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccountSubscription business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccountSubscription"})
}

func handleDescribeActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeActionConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeActionConnector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeActionConnector"})
}

func handleDescribeActionConnectorPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeActionConnectorPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeActionConnectorPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeActionConnectorPermissions"})
}

func handleDescribeAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAnalysisRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAnalysis business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAnalysis"})
}

func handleDescribeAnalysisDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAnalysisDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAnalysisDefinition business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAnalysisDefinition"})
}

func handleDescribeAnalysisPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAnalysisPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAnalysisPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAnalysisPermissions"})
}

func handleDescribeAssetBundleExportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAssetBundleExportJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAssetBundleExportJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAssetBundleExportJob"})
}

func handleDescribeAssetBundleImportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAssetBundleImportJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAssetBundleImportJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAssetBundleImportJob"})
}

func handleDescribeAutomationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAutomationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAutomationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAutomationJob"})
}

func handleDescribeBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeBrandRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeBrand business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeBrand"})
}

func handleDescribeBrandAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeBrandAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeBrandAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeBrandAssignment"})
}

func handleDescribeBrandPublishedVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeBrandPublishedVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeBrandPublishedVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeBrandPublishedVersion"})
}

func handleDescribeCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCustomPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCustomPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCustomPermissions"})
}

func handleDescribeDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDashboardRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDashboard business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDashboard"})
}

func handleDescribeDashboardDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDashboardDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDashboardDefinition business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDashboardDefinition"})
}

func handleDescribeDashboardPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDashboardPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDashboardPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDashboardPermissions"})
}

func handleDescribeDashboardSnapshotJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDashboardSnapshotJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDashboardSnapshotJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDashboardSnapshotJob"})
}

func handleDescribeDashboardSnapshotJobResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDashboardSnapshotJobResultRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDashboardSnapshotJobResult business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDashboardSnapshotJobResult"})
}

func handleDescribeDashboardsQAConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDashboardsQAConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDashboardsQAConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDashboardsQAConfiguration"})
}

func handleDescribeDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDataSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataSet"})
}

func handleDescribeDataSetPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDataSetPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataSetPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataSetPermissions"})
}

func handleDescribeDataSetRefreshProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDataSetRefreshPropertiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataSetRefreshProperties business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataSetRefreshProperties"})
}

func handleDescribeDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataSource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataSource"})
}

func handleDescribeDataSourcePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDataSourcePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDataSourcePermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDataSourcePermissions"})
}

func handleDescribeDefaultQBusinessApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeDefaultQBusinessApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeDefaultQBusinessApplication business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeDefaultQBusinessApplication"})
}

func handleDescribeFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFolderRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFolder business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFolder"})
}

func handleDescribeFolderPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFolderPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFolderPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFolderPermissions"})
}

func handleDescribeFolderResolvedPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFolderResolvedPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFolderResolvedPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFolderResolvedPermissions"})
}

func handleDescribeGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeGroup"})
}

func handleDescribeGroupMembership(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeGroupMembershipRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeGroupMembership business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeGroupMembership"})
}

func handleDescribeIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeIAMPolicyAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeIAMPolicyAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeIAMPolicyAssignment"})
}

func handleDescribeIngestion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeIngestionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeIngestion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeIngestion"})
}

func handleDescribeIpRestriction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeIpRestrictionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeIpRestriction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeIpRestriction"})
}

func handleDescribeKeyRegistration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeKeyRegistrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeKeyRegistration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeKeyRegistration"})
}

func handleDescribeNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeNamespace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeNamespace"})
}

func handleDescribeQPersonalizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeQPersonalizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeQPersonalizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeQPersonalizationConfiguration"})
}

func handleDescribeQuickSightQSearchConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeQuickSightQSearchConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeQuickSightQSearchConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeQuickSightQSearchConfiguration"})
}

func handleDescribeRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeRefreshSchedule"})
}

func handleDescribeRoleCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeRoleCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeRoleCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeRoleCustomPermission"})
}

func handleDescribeSelfUpgradeConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeSelfUpgradeConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeSelfUpgradeConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeSelfUpgradeConfiguration"})
}

func handleDescribeTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTemplateRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTemplate business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTemplate"})
}

func handleDescribeTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTemplateAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTemplateAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTemplateAlias"})
}

func handleDescribeTemplateDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTemplateDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTemplateDefinition business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTemplateDefinition"})
}

func handleDescribeTemplatePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTemplatePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTemplatePermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTemplatePermissions"})
}

func handleDescribeTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeThemeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTheme business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTheme"})
}

func handleDescribeThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeThemeAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeThemeAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeThemeAlias"})
}

func handleDescribeThemePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeThemePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeThemePermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeThemePermissions"})
}

func handleDescribeTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTopicRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTopic business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTopic"})
}

func handleDescribeTopicPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTopicPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTopicPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTopicPermissions"})
}

func handleDescribeTopicRefresh(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTopicRefreshRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTopicRefresh business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTopicRefresh"})
}

func handleDescribeTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTopicRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTopicRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTopicRefreshSchedule"})
}

func handleDescribeUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeUser"})
}

func handleDescribeVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeVPCConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeVPCConnection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeVPCConnection"})
}

func handleGenerateEmbedUrlForAnonymousUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GenerateEmbedUrlForAnonymousUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GenerateEmbedUrlForAnonymousUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GenerateEmbedUrlForAnonymousUser"})
}

func handleGenerateEmbedUrlForRegisteredUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GenerateEmbedUrlForRegisteredUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GenerateEmbedUrlForRegisteredUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GenerateEmbedUrlForRegisteredUser"})
}

func handleGenerateEmbedUrlForRegisteredUserWithIdentity(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GenerateEmbedUrlForRegisteredUserWithIdentityRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GenerateEmbedUrlForRegisteredUserWithIdentity business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GenerateEmbedUrlForRegisteredUserWithIdentity"})
}

func handleGetDashboardEmbedUrl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetDashboardEmbedUrlRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetDashboardEmbedUrl business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetDashboardEmbedUrl"})
}

func handleGetFlowMetadata(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFlowMetadataInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFlowMetadata business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFlowMetadata"})
}

func handleGetFlowPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetFlowPermissionsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetFlowPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetFlowPermissions"})
}

func handleGetIdentityContext(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetIdentityContextRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetIdentityContext business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetIdentityContext"})
}

func handleGetSessionEmbedUrl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetSessionEmbedUrlRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetSessionEmbedUrl business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetSessionEmbedUrl"})
}

func handleListActionConnectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListActionConnectorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListActionConnectors business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListActionConnectors"})
}

func handleListAnalyses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAnalysesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAnalyses business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAnalyses"})
}

func handleListAssetBundleExportJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAssetBundleExportJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAssetBundleExportJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAssetBundleExportJobs"})
}

func handleListAssetBundleImportJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAssetBundleImportJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAssetBundleImportJobs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAssetBundleImportJobs"})
}

func handleListBrands(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListBrandsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListBrands business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListBrands"})
}

func handleListCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCustomPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCustomPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCustomPermissions"})
}

func handleListDashboardVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDashboardVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDashboardVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDashboardVersions"})
}

func handleListDashboards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDashboardsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDashboards business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDashboards"})
}

func handleListDataSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDataSetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDataSets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDataSets"})
}

func handleListDataSources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListDataSourcesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListDataSources business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListDataSources"})
}

func handleListFlows(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFlowsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFlows business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFlows"})
}

func handleListFolderMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFolderMembersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFolderMembers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFolderMembers"})
}

func handleListFolders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFoldersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFolders business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFolders"})
}

func handleListFoldersForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListFoldersForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListFoldersForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListFoldersForResource"})
}

func handleListGroupMemberships(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListGroupMembershipsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListGroupMemberships business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListGroupMemberships"})
}

func handleListGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListGroups"})
}

func handleListIAMPolicyAssignments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListIAMPolicyAssignmentsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListIAMPolicyAssignments business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListIAMPolicyAssignments"})
}

func handleListIAMPolicyAssignmentsForUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListIAMPolicyAssignmentsForUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListIAMPolicyAssignmentsForUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListIAMPolicyAssignmentsForUser"})
}

func handleListIdentityPropagationConfigs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListIdentityPropagationConfigsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListIdentityPropagationConfigs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListIdentityPropagationConfigs"})
}

func handleListIngestions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListIngestionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListIngestions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListIngestions"})
}

func handleListNamespaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListNamespacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListNamespaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListNamespaces"})
}

func handleListRefreshSchedules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListRefreshSchedulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListRefreshSchedules business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListRefreshSchedules"})
}

func handleListRoleMemberships(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListRoleMembershipsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListRoleMemberships business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListRoleMemberships"})
}

func handleListSelfUpgrades(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListSelfUpgradesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListSelfUpgrades business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListSelfUpgrades"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListTemplateAliases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTemplateAliasesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTemplateAliases business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTemplateAliases"})
}

func handleListTemplateVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTemplateVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTemplateVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTemplateVersions"})
}

func handleListTemplates(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTemplatesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTemplates business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTemplates"})
}

func handleListThemeAliases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListThemeAliasesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListThemeAliases business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListThemeAliases"})
}

func handleListThemeVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListThemeVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListThemeVersions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListThemeVersions"})
}

func handleListThemes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListThemesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListThemes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListThemes"})
}

func handleListTopicRefreshSchedules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTopicRefreshSchedulesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTopicRefreshSchedules business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTopicRefreshSchedules"})
}

func handleListTopicReviewedAnswers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTopicReviewedAnswersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTopicReviewedAnswers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTopicReviewedAnswers"})
}

func handleListTopics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTopicsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTopics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTopics"})
}

func handleListUserGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListUserGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListUserGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListUserGroups"})
}

func handleListUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListUsersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListUsers business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListUsers"})
}

func handleListVPCConnections(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListVPCConnectionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListVPCConnections business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListVPCConnections"})
}

func handlePredictQAResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PredictQAResultsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PredictQAResults business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PredictQAResults"})
}

func handlePutDataSetRefreshProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutDataSetRefreshPropertiesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutDataSetRefreshProperties business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutDataSetRefreshProperties"})
}

func handleRegisterUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RegisterUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RegisterUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RegisterUser"})
}

func handleRestoreAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RestoreAnalysisRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RestoreAnalysis business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RestoreAnalysis"})
}

func handleSearchActionConnectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchActionConnectorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchActionConnectors business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchActionConnectors"})
}

func handleSearchAnalyses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchAnalysesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchAnalyses business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchAnalyses"})
}

func handleSearchDashboards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchDashboardsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchDashboards business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchDashboards"})
}

func handleSearchDataSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchDataSetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchDataSets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchDataSets"})
}

func handleSearchDataSources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchDataSourcesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchDataSources business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchDataSources"})
}

func handleSearchFlows(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchFlowsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchFlows business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchFlows"})
}

func handleSearchFolders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchFoldersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchFolders business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchFolders"})
}

func handleSearchGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchGroups"})
}

func handleSearchTopics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req SearchTopicsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement SearchTopics business logic
	return jsonOK(map[string]any{"status": "ok", "action": "SearchTopics"})
}

func handleStartAssetBundleExportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartAssetBundleExportJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartAssetBundleExportJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartAssetBundleExportJob"})
}

func handleStartAssetBundleImportJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartAssetBundleImportJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartAssetBundleImportJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartAssetBundleImportJob"})
}

func handleStartAutomationJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartAutomationJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartAutomationJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartAutomationJob"})
}

func handleStartDashboardSnapshotJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartDashboardSnapshotJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartDashboardSnapshotJob business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartDashboardSnapshotJob"})
}

func handleStartDashboardSnapshotJobSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req StartDashboardSnapshotJobScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement StartDashboardSnapshotJobSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "StartDashboardSnapshotJobSchedule"})
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

func handleUpdateAccountCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAccountCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAccountCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAccountCustomPermission"})
}

func handleUpdateAccountCustomization(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAccountCustomizationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAccountCustomization business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAccountCustomization"})
}

func handleUpdateAccountSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAccountSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAccountSettings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAccountSettings"})
}

func handleUpdateActionConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateActionConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateActionConnector business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateActionConnector"})
}

func handleUpdateActionConnectorPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateActionConnectorPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateActionConnectorPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateActionConnectorPermissions"})
}

func handleUpdateAnalysis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAnalysisRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAnalysis business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAnalysis"})
}

func handleUpdateAnalysisPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAnalysisPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAnalysisPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAnalysisPermissions"})
}

func handleUpdateApplicationWithTokenExchangeGrant(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateApplicationWithTokenExchangeGrantRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateApplicationWithTokenExchangeGrant business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateApplicationWithTokenExchangeGrant"})
}

func handleUpdateBrand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateBrandRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateBrand business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateBrand"})
}

func handleUpdateBrandAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateBrandAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateBrandAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateBrandAssignment"})
}

func handleUpdateBrandPublishedVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateBrandPublishedVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateBrandPublishedVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateBrandPublishedVersion"})
}

func handleUpdateCustomPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCustomPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCustomPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCustomPermissions"})
}

func handleUpdateDashboard(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDashboardRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDashboard business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDashboard"})
}

func handleUpdateDashboardLinks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDashboardLinksRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDashboardLinks business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDashboardLinks"})
}

func handleUpdateDashboardPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDashboardPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDashboardPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDashboardPermissions"})
}

func handleUpdateDashboardPublishedVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDashboardPublishedVersionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDashboardPublishedVersion business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDashboardPublishedVersion"})
}

func handleUpdateDashboardsQAConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDashboardsQAConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDashboardsQAConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDashboardsQAConfiguration"})
}

func handleUpdateDataSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDataSetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDataSet business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDataSet"})
}

func handleUpdateDataSetPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDataSetPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDataSetPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDataSetPermissions"})
}

func handleUpdateDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDataSource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDataSource"})
}

func handleUpdateDataSourcePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDataSourcePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDataSourcePermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDataSourcePermissions"})
}

func handleUpdateDefaultQBusinessApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateDefaultQBusinessApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateDefaultQBusinessApplication business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateDefaultQBusinessApplication"})
}

func handleUpdateFlowPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFlowPermissionsInput
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFlowPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFlowPermissions"})
}

func handleUpdateFolder(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFolderRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFolder business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFolder"})
}

func handleUpdateFolderPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFolderPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFolderPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFolderPermissions"})
}

func handleUpdateGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateGroup"})
}

func handleUpdateIAMPolicyAssignment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateIAMPolicyAssignmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateIAMPolicyAssignment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateIAMPolicyAssignment"})
}

func handleUpdateIdentityPropagationConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateIdentityPropagationConfigRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateIdentityPropagationConfig business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateIdentityPropagationConfig"})
}

func handleUpdateIpRestriction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateIpRestrictionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateIpRestriction business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateIpRestriction"})
}

func handleUpdateKeyRegistration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateKeyRegistrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateKeyRegistration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateKeyRegistration"})
}

func handleUpdatePublicSharingSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdatePublicSharingSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdatePublicSharingSettings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdatePublicSharingSettings"})
}

func handleUpdateQPersonalizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateQPersonalizationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateQPersonalizationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateQPersonalizationConfiguration"})
}

func handleUpdateQuickSightQSearchConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateQuickSightQSearchConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateQuickSightQSearchConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateQuickSightQSearchConfiguration"})
}

func handleUpdateRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateRefreshSchedule"})
}

func handleUpdateRoleCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateRoleCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateRoleCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateRoleCustomPermission"})
}

func handleUpdateSPICECapacityConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateSPICECapacityConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateSPICECapacityConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateSPICECapacityConfiguration"})
}

func handleUpdateSelfUpgrade(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateSelfUpgradeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateSelfUpgrade business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateSelfUpgrade"})
}

func handleUpdateSelfUpgradeConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateSelfUpgradeConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateSelfUpgradeConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateSelfUpgradeConfiguration"})
}

func handleUpdateTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTemplateRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTemplate business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTemplate"})
}

func handleUpdateTemplateAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTemplateAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTemplateAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTemplateAlias"})
}

func handleUpdateTemplatePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTemplatePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTemplatePermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTemplatePermissions"})
}

func handleUpdateTheme(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateThemeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTheme business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTheme"})
}

func handleUpdateThemeAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateThemeAliasRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateThemeAlias business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateThemeAlias"})
}

func handleUpdateThemePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateThemePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateThemePermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateThemePermissions"})
}

func handleUpdateTopic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTopicRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTopic business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTopic"})
}

func handleUpdateTopicPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTopicPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTopicPermissions business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTopicPermissions"})
}

func handleUpdateTopicRefreshSchedule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTopicRefreshScheduleRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTopicRefreshSchedule business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTopicRefreshSchedule"})
}

func handleUpdateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateUser business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateUser"})
}

func handleUpdateUserCustomPermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateUserCustomPermissionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateUserCustomPermission business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateUserCustomPermission"})
}

func handleUpdateVPCConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateVPCConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateVPCConnection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateVPCConnection"})
}

