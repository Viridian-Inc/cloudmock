package quicksight

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored resource types ────────────────────────────────────────────────────

// StoredUser is a persisted QuickSight user.
type StoredUser struct {
	UserName                            string
	AwsAccountId                        string
	Namespace                           string
	Arn                                 string
	Email                               string
	Role                                string
	IdentityType                        string
	Active                              bool
	PrincipalId                         string
	CustomPermissionsName               string
	ExternalLoginFederationProviderType string
	ExternalLoginFederationProviderUrl  string
	ExternalLoginId                     string
}

// StoredGroup is a persisted QuickSight group.
type StoredGroup struct {
	GroupName    string
	AwsAccountId string
	Namespace    string
	Arn          string
	Description  string
	PrincipalId  string
	Members      []string // user names
}

// StoredNamespace is a persisted namespace.
type StoredNamespace struct {
	Name             string
	Arn              string
	CapacityRegion   string
	CreationStatus   string
	IdentityStore    string
	NamespaceError   map[string]any
	IamIdentityCenterApplicationArn string
	IamIdentityCenterInstanceArn    string
}

// StoredDataSource is a persisted data source.
type StoredDataSource struct {
	DataSourceId                  string
	AwsAccountId                  string
	Arn                           string
	Name                          string
	Type                          string
	Status                        string
	CreatedTime                   time.Time
	LastUpdatedTime               time.Time
	DataSourceParameters          map[string]any
	AlternateDataSourceParameters []map[string]any
	VpcConnectionProperties       map[string]any
	SslProperties                 map[string]any
	ErrorInfo                     map[string]any
	SecretArn                     string
	Credentials                   map[string]any
}

// StoredDataSet is a persisted data set.
type StoredDataSet struct {
	DataSetId                          string
	AwsAccountId                       string
	Arn                                string
	Name                               string
	CreatedTime                        time.Time
	LastUpdatedTime                    time.Time
	PhysicalTableMap                   map[string]any
	LogicalTableMap                    map[string]any
	OutputColumns                      []map[string]any
	ImportMode                         string
	ConsumedSpiceCapacityInBytes       int64
	ColumnGroups                       []map[string]any
	FieldFolders                       map[string]any
	RowLevelPermissionDataSet          map[string]any
	RowLevelPermissionTagConfiguration map[string]any
	ColumnLevelPermissionRules         []map[string]any
	DataSetUsageConfiguration          map[string]any
	DatasetParameters                  []map[string]any
	RefreshProperties                  map[string]any
	FolderArns                         []string
}

// StoredTemplate is a persisted template.
type StoredTemplate struct {
	TemplateId      string
	AwsAccountId    string
	Arn             string
	Name            string
	CreatedTime     time.Time
	LastUpdatedTime time.Time
	Version         map[string]any
	VersionNumber   int
	Versions        []map[string]any
	Aliases         map[string]map[string]any // alias name -> alias
	Definition      map[string]any
}

// StoredDashboard is a persisted dashboard.
type StoredDashboard struct {
	DashboardId             string
	AwsAccountId            string
	Arn                     string
	Name                    string
	CreatedTime             time.Time
	LastUpdatedTime         time.Time
	LastPublishedTime       time.Time
	Version                 map[string]any
	VersionNumber           int
	Versions                []map[string]any
	LinkEntities            []string
	DashboardPublishOptions map[string]any
	Definition              map[string]any
	ValidationStrategy      map[string]any
	PublishedVersionNumber  int
}

// StoredAnalysis is a persisted analysis.
type StoredAnalysis struct {
	AnalysisId      string
	AwsAccountId    string
	Arn             string
	Name            string
	Status          string
	Errors          []map[string]any
	DataSetArns     []string
	ThemeArn        string
	CreatedTime     time.Time
	LastUpdatedTime time.Time
	Sheets          []map[string]any
	Definition      map[string]any
}

// StoredTheme is a persisted theme.
type StoredTheme struct {
	ThemeId         string
	AwsAccountId    string
	Arn             string
	Name            string
	Type            string
	BaseThemeId     string
	CreatedTime     time.Time
	LastUpdatedTime time.Time
	Version         map[string]any
	VersionNumber   int
	Versions        []map[string]any
	Aliases         map[string]map[string]any
	Configuration   map[string]any
}

// StoredFolder is a persisted folder.
type StoredFolder struct {
	FolderId        string
	AwsAccountId    string
	Arn             string
	Name            string
	FolderType      string
	FolderPath      []string
	CreatedTime     time.Time
	LastUpdatedTime time.Time
	ParentFolderArn string
	SharingModel    string
	Members         []map[string]any // member id -> {MemberId, MemberType}
}

// StoredTopic is a persisted Q topic.
type StoredTopic struct {
	TopicId               string
	AwsAccountId          string
	Arn                   string
	Name                  string
	Description           string
	UserExperienceVersion string
	DataSets              []map[string]any
	ConfigOptions         map[string]any
	RefreshSchedule       map[string]any
	ReviewedAnswers       []map[string]any
	CreatedTime           time.Time
	LastUpdatedTime       time.Time
}

// StoredIAMPolicyAssignment is a persisted IAM policy assignment.
type StoredIAMPolicyAssignment struct {
	AssignmentName   string
	AssignmentId     string
	AssignmentStatus string
	PolicyArn        string
	Identities       map[string][]string // user/group -> names
	AwsAccountId     string
	Namespace        string
}

// StoredIngestion is a persisted ingestion job.
type StoredIngestion struct {
	IngestionId            string
	DataSetId              string
	AwsAccountId           string
	Arn                    string
	IngestionStatus        string
	RowInfo                map[string]any
	QueueInfo              map[string]any
	CreatedTime            time.Time
	IngestionTimeInSeconds int64
	IngestionSizeInBytes   int64
	RequestSource          string
	RequestType            string
	ErrorInfo              map[string]any
}

// StoredRefreshSchedule is a persisted dataset refresh schedule.
type StoredRefreshSchedule struct {
	ScheduleId          string
	DataSetId           string
	AwsAccountId        string
	Arn                 string
	ScheduleFrequency   map[string]any
	StartAfterDateTime  time.Time
	RefreshType         string
}

// StoredVPCConnection is a persisted VPC connection.
type StoredVPCConnection struct {
	VPCConnectionId     string
	AwsAccountId        string
	Arn                 string
	Name                string
	VpcId               string
	SecurityGroupIds    []string
	DnsResolvers        []string
	Status              string
	AvailabilityStatus  string
	NetworkInterfaces   []map[string]any
	RoleArn             string
	CreatedTime         time.Time
	LastUpdatedTime     time.Time
}

// StoredCustomPermissions is a persisted custom permissions object.
type StoredCustomPermissions struct {
	CustomPermissionsName string
	AwsAccountId          string
	Arn                   string
	Capabilities          map[string]any
}

// StoredBrand is a persisted brand.
type StoredBrand struct {
	BrandId         string
	Arn             string
	BrandName       string
	Status          string
	CreatedTime     time.Time
	LastUpdatedTime time.Time
	BrandDetail     map[string]any
	BrandColorPalette map[string]any
	BrandElementStyle map[string]any
	Versions        []map[string]any
	PublishedVersionId string
}

// StoredAssetBundleExportJob is a persisted asset bundle export job.
type StoredAssetBundleExportJob struct {
	JobId                                       string
	Arn                                         string
	Status                                      string
	ResourceArns                                []string
	CloudFormationOverridePropertyConfiguration map[string]any
	ExportFormat                                string
	DownloadUrl                                 string
	IncludePermissions                          bool
	IncludeTags                                 bool
	IncludeAllDependencies                      bool
	ValidationStrategy                          map[string]any
	Warnings                                    []map[string]any
	Errors                                      []map[string]any
	CreatedTime                                 time.Time
	AwsAccountId                                string
	describeCallCount                           int
}

// StoredAssetBundleImportJob is a persisted asset bundle import job.
type StoredAssetBundleImportJob struct {
	JobId                   string
	Arn                     string
	Status                  string
	AssetBundleImportSource map[string]any
	OverrideParameters      map[string]any
	OverridePermissions     map[string]any
	OverrideTags            map[string]any
	OverrideValidationStrategy map[string]any
	FailureAction           string
	RollbackErrors          []map[string]any
	Errors                  []map[string]any
	CreatedTime             time.Time
	AwsAccountId            string
	describeCallCount       int
}

// StoredDashboardSnapshotJob is a persisted dashboard snapshot job.
type StoredDashboardSnapshotJob struct {
	JobId           string
	Arn             string
	DashboardId     string
	AwsAccountId    string
	Status          string
	UserConfiguration map[string]any
	SnapshotConfiguration map[string]any
	JobStatus       string
	Result          map[string]any
	CreatedTime     time.Time
}

// StoredActionConnector is a persisted action connector.
type StoredActionConnector struct {
	ActionConnectorId    string
	Arn                  string
	Name                 string
	Type                 string
	Description          string
	Status               string
	AuthenticationConfig map[string]any
	EnabledActions       []string
	VpcConnectionArn     string
	CreatedTime          time.Time
	LastUpdatedTime      time.Time
}

// StoredAutomationJob is a persisted automation/flow job.
type StoredAutomationJob struct {
	JobId        string
	Status       string
	CreatedTime  time.Time
	Configuration map[string]any
}

// StoredFlow is a persisted flow.
type StoredFlow struct {
	FlowId          string
	Arn             string
	Name            string
	Description     string
	CreatedTime     time.Time
	LastUpdatedTime time.Time
	Permissions     []map[string]any
}

// Store is the in-memory data store for QuickSight resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	// Per-resource sub-stores. Many resources are scoped by account ID
	// in real QuickSight; for the mock we use a flat keyed namespace.
	users                  map[string]map[string]*StoredUser    // namespace -> userName -> user
	groups                 map[string]map[string]*StoredGroup   // namespace -> groupName -> group
	namespaces             map[string]*StoredNamespace
	dataSources            map[string]*StoredDataSource
	dataSets               map[string]*StoredDataSet
	templates              map[string]*StoredTemplate
	dashboards             map[string]*StoredDashboard
	analyses               map[string]*StoredAnalysis
	themes                 map[string]*StoredTheme
	folders                map[string]*StoredFolder
	topics                 map[string]*StoredTopic
	iamPolicyAssignments   map[string]map[string]*StoredIAMPolicyAssignment // namespace -> name -> assignment
	ingestions             map[string]map[string]*StoredIngestion           // dataset -> ingestionId -> ingestion
	refreshSchedules       map[string]map[string]*StoredRefreshSchedule     // dataset -> scheduleId -> schedule
	vpcConnections         map[string]*StoredVPCConnection
	customPermissions      map[string]*StoredCustomPermissions
	brands                 map[string]*StoredBrand
	assetBundleExportJobs  map[string]*StoredAssetBundleExportJob
	assetBundleImportJobs  map[string]*StoredAssetBundleImportJob
	dashboardSnapshotJobs  map[string]*StoredDashboardSnapshotJob
	actionConnectors       map[string]*StoredActionConnector
	automationJobs         map[string]*StoredAutomationJob
	flows                  map[string]*StoredFlow

	// Permissions stores: arn -> []permission
	dataSourcePermissions     map[string][]map[string]any
	dataSetPermissions        map[string][]map[string]any
	templatePermissions       map[string][]map[string]any
	dashboardPermissions      map[string][]map[string]any
	analysisPermissions       map[string][]map[string]any
	themePermissions          map[string][]map[string]any
	folderPermissions         map[string][]map[string]any
	topicPermissions          map[string][]map[string]any
	actionConnectorPermissions map[string][]map[string]any
	flowPermissions           map[string][]map[string]any

	// Tags: arn -> map[key]value
	tags map[string]map[string]string

	// Per-account / per-namespace settings
	accountSettings        *StoredAccountSettings
	accountSubscription    *StoredAccountSubscription
	accountCustomizations  map[string]map[string]any // namespace -> customization
	accountCustomPermission map[string]string         // user arn -> custom permissions name
	roleCustomPermissions  map[string]map[string]string // namespace -> role -> permissions name
	roleMemberships        map[string]map[string]map[string]bool // namespace -> role -> group set
	ipRestriction          map[string]any
	keyRegistration        []map[string]any
	qPersonalizationCfg    string
	qSearchCfg             string
	defaultQBusinessApp    map[string]any
	dashboardsQAConfig     string
	publicSharingEnabled   bool
	identityPropagationConfigs map[string]map[string]any // service -> config
	brandAssignment        string
	selfUpgradeConfig      map[string]any
	dataSetRefreshProperties map[string]map[string]any // dataset arn -> properties
	folderResources        map[string][]string // folder id -> resource arns
}

// StoredAccountSettings is account-level settings.
type StoredAccountSettings struct {
	AccountName                  string
	Edition                      string
	DefaultNamespace             string
	NotificationEmail            string
	PublicSharingEnabled         bool
	TerminationProtectionEnabled bool
}

// StoredAccountSubscription is the account subscription.
type StoredAccountSubscription struct {
	Edition                   string
	DirectoryType             string
	AuthenticationType        string
	AccountSubscriptionStatus string
	Notifications             map[string]any
	IAMIdentityCenterInstanceArn string
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	s := &Store{
		accountID: accountID,
		region:    region,
	}
	s.reset()
	return s
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reset()
}

func (s *Store) reset() {
	s.users = make(map[string]map[string]*StoredUser)
	s.groups = make(map[string]map[string]*StoredGroup)
	s.namespaces = make(map[string]*StoredNamespace)
	s.dataSources = make(map[string]*StoredDataSource)
	s.dataSets = make(map[string]*StoredDataSet)
	s.templates = make(map[string]*StoredTemplate)
	s.dashboards = make(map[string]*StoredDashboard)
	s.analyses = make(map[string]*StoredAnalysis)
	s.themes = make(map[string]*StoredTheme)
	s.folders = make(map[string]*StoredFolder)
	s.topics = make(map[string]*StoredTopic)
	s.iamPolicyAssignments = make(map[string]map[string]*StoredIAMPolicyAssignment)
	s.ingestions = make(map[string]map[string]*StoredIngestion)
	s.refreshSchedules = make(map[string]map[string]*StoredRefreshSchedule)
	s.vpcConnections = make(map[string]*StoredVPCConnection)
	s.customPermissions = make(map[string]*StoredCustomPermissions)
	s.brands = make(map[string]*StoredBrand)
	s.assetBundleExportJobs = make(map[string]*StoredAssetBundleExportJob)
	s.assetBundleImportJobs = make(map[string]*StoredAssetBundleImportJob)
	s.dashboardSnapshotJobs = make(map[string]*StoredDashboardSnapshotJob)
	s.actionConnectors = make(map[string]*StoredActionConnector)
	s.automationJobs = make(map[string]*StoredAutomationJob)
	s.flows = make(map[string]*StoredFlow)
	s.dataSourcePermissions = make(map[string][]map[string]any)
	s.dataSetPermissions = make(map[string][]map[string]any)
	s.templatePermissions = make(map[string][]map[string]any)
	s.dashboardPermissions = make(map[string][]map[string]any)
	s.analysisPermissions = make(map[string][]map[string]any)
	s.themePermissions = make(map[string][]map[string]any)
	s.folderPermissions = make(map[string][]map[string]any)
	s.topicPermissions = make(map[string][]map[string]any)
	s.actionConnectorPermissions = make(map[string][]map[string]any)
	s.flowPermissions = make(map[string][]map[string]any)
	s.tags = make(map[string]map[string]string)
	s.accountSettings = &StoredAccountSettings{
		AccountName:      "cloudmock-account",
		Edition:          "ENTERPRISE",
		DefaultNamespace: "default",
		NotificationEmail: "admin@cloudmock.test",
	}
	s.accountSubscription = &StoredAccountSubscription{
		Edition:                   "ENTERPRISE",
		DirectoryType:             "QUICKSIGHT",
		AuthenticationType:        "IAM_AND_QUICKSIGHT",
		AccountSubscriptionStatus: "ACCOUNT_CREATED",
	}
	s.accountCustomizations = make(map[string]map[string]any)
	s.accountCustomPermission = make(map[string]string)
	s.roleCustomPermissions = make(map[string]map[string]string)
	s.roleMemberships = make(map[string]map[string]map[string]bool)
	s.ipRestriction = map[string]any{
		"AwsAccountId":    s.accountID,
		"IpRestrictionRuleMap": map[string]string{},
		"VpcIdRestrictionRuleMap": map[string]string{},
		"VpcEndpointIdRestrictionRuleMap": map[string]string{},
		"Enabled":         false,
	}
	s.keyRegistration = nil
	s.qPersonalizationCfg = "DISABLED"
	s.qSearchCfg = "DISABLED"
	s.defaultQBusinessApp = nil
	s.dashboardsQAConfig = "DISABLED"
	s.publicSharingEnabled = false
	s.identityPropagationConfigs = make(map[string]map[string]any)
	s.brandAssignment = ""
	s.selfUpgradeConfig = nil
	s.dataSetRefreshProperties = make(map[string]map[string]any)
	s.folderResources = make(map[string][]string)
}

// ── ARN builders ─────────────────────────────────────────────────────────────

func (s *Store) arnUser(namespace, userName string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:user/%s/%s", s.region, s.accountID, namespace, userName)
}

func (s *Store) arnGroup(namespace, groupName string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:group/%s/%s", s.region, s.accountID, namespace, groupName)
}

func (s *Store) arnNamespace(name string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:namespace/%s", s.region, s.accountID, name)
}

func (s *Store) arnDataSource(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:datasource/%s", s.region, s.accountID, id)
}

func (s *Store) arnDataSet(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:dataset/%s", s.region, s.accountID, id)
}

func (s *Store) arnTemplate(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:template/%s", s.region, s.accountID, id)
}

func (s *Store) arnTemplateAlias(id, alias string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:template/%s/alias/%s", s.region, s.accountID, id, alias)
}

func (s *Store) arnDashboard(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:dashboard/%s", s.region, s.accountID, id)
}

func (s *Store) arnAnalysis(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:analysis/%s", s.region, s.accountID, id)
}

func (s *Store) arnTheme(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:theme/%s", s.region, s.accountID, id)
}

func (s *Store) arnThemeAlias(id, alias string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:theme/%s/alias/%s", s.region, s.accountID, id, alias)
}

func (s *Store) arnFolder(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:folder/%s", s.region, s.accountID, id)
}

func (s *Store) arnTopic(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:topic/%s", s.region, s.accountID, id)
}

func (s *Store) arnRefreshSchedule(datasetId, scheduleId string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:dataset/%s/refresh-schedule/%s", s.region, s.accountID, datasetId, scheduleId)
}

func (s *Store) arnIngestion(datasetId, ingestionId string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:dataset/%s/ingestion/%s", s.region, s.accountID, datasetId, ingestionId)
}

func (s *Store) arnVPCConnection(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:vpcConnection/%s", s.region, s.accountID, id)
}

func (s *Store) arnCustomPermissions(name string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:custom-permission/%s", s.region, s.accountID, name)
}

func (s *Store) arnBrand(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:brand/%s", s.region, s.accountID, id)
}

func (s *Store) arnAssetBundleExportJob(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:asset-bundle-export-job/%s", s.region, s.accountID, id)
}

func (s *Store) arnAssetBundleImportJob(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:asset-bundle-import-job/%s", s.region, s.accountID, id)
}

func (s *Store) arnDashboardSnapshotJob(id, dashboardId string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:dashboard/%s/snapshot/%s", s.region, s.accountID, dashboardId, id)
}

func (s *Store) arnIAMPolicyAssignment(namespace, name string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:iam-policy-assignment/%s/%s", s.region, s.accountID, namespace, name)
}

func (s *Store) arnActionConnector(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:action-connector/%s", s.region, s.accountID, id)
}

func (s *Store) arnFlow(id string) string {
	return fmt.Sprintf("arn:aws:quicksight:%s:%s:flow/%s", s.region, s.accountID, id)
}

// ── Generic helpers ──────────────────────────────────────────────────────────

func errNotFound(resourceType, id string) *service.AWSError {
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("%s %q not found", resourceType, id), http.StatusNotFound)
}

func errExists(resourceType, id string) *service.AWSError {
	return service.NewAWSError("ResourceExistsException",
		fmt.Sprintf("%s %q already exists", resourceType, id), http.StatusConflict)
}

func errInvalid(message string) *service.AWSError {
	return service.NewAWSError("InvalidParameterValueException", message, http.StatusBadRequest)
}

func errConflict(message string) *service.AWSError {
	return service.NewAWSError("ConflictException", message, http.StatusConflict)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func nowUTC() time.Time { return time.Now().UTC() }

// ── Users ────────────────────────────────────────────────────────────────────

func (s *Store) RegisterUser(namespace, userName, email, role, identityType, customPermissionsName string,
	externalLoginType, externalLoginUrl, externalLoginId string) (*StoredUser, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userName == "" {
		return nil, errInvalid("UserName is required")
	}
	if namespace == "" {
		namespace = "default"
	}
	if s.users[namespace] == nil {
		s.users[namespace] = make(map[string]*StoredUser)
	}
	if _, ok := s.users[namespace][userName]; ok {
		return nil, errExists("User", userName)
	}
	u := &StoredUser{
		UserName:                            userName,
		AwsAccountId:                        s.accountID,
		Namespace:                           namespace,
		Arn:                                 s.arnUser(namespace, userName),
		Email:                               email,
		Role:                                role,
		IdentityType:                        identityType,
		Active:                              true,
		PrincipalId:                         "principal-" + generateID(),
		CustomPermissionsName:               customPermissionsName,
		ExternalLoginFederationProviderType: externalLoginType,
		ExternalLoginFederationProviderUrl:  externalLoginUrl,
		ExternalLoginId:                     externalLoginId,
	}
	s.users[namespace][userName] = u
	return u, nil
}

func (s *Store) GetUser(namespace, userName string) (*StoredUser, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.users[namespace]
	if !ok {
		return nil, errNotFound("User", userName)
	}
	u, ok := ns[userName]
	if !ok {
		return nil, errNotFound("User", userName)
	}
	return u, nil
}

func (s *Store) ListUsers(namespace string) []*StoredUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	out := []*StoredUser{}
	for _, u := range s.users[namespace] {
		out = append(out, u)
	}
	return out
}

func (s *Store) UpdateUser(namespace, userName, email, role, customPermissionsName string,
	externalLoginType, externalLoginUrl, externalLoginId string, unapply bool) (*StoredUser, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.users[namespace]
	if !ok {
		return nil, errNotFound("User", userName)
	}
	u, ok := ns[userName]
	if !ok {
		return nil, errNotFound("User", userName)
	}
	if email != "" {
		u.Email = email
	}
	if role != "" {
		u.Role = role
	}
	if unapply {
		u.CustomPermissionsName = ""
	} else if customPermissionsName != "" {
		u.CustomPermissionsName = customPermissionsName
	}
	if externalLoginType != "" {
		u.ExternalLoginFederationProviderType = externalLoginType
	}
	if externalLoginUrl != "" {
		u.ExternalLoginFederationProviderUrl = externalLoginUrl
	}
	if externalLoginId != "" {
		u.ExternalLoginId = externalLoginId
	}
	return u, nil
}

func (s *Store) DeleteUser(namespace, userName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.users[namespace]
	if !ok {
		return errNotFound("User", userName)
	}
	if _, ok := ns[userName]; !ok {
		return errNotFound("User", userName)
	}
	delete(ns, userName)
	return nil
}

func (s *Store) DeleteUserByPrincipalId(namespace, principalId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.users[namespace]
	if !ok {
		return errNotFound("User", principalId)
	}
	for k, u := range ns {
		if u.PrincipalId == principalId {
			delete(ns, k)
			return nil
		}
	}
	return errNotFound("User", principalId)
}

// ── Groups ───────────────────────────────────────────────────────────────────

func (s *Store) CreateGroup(namespace, groupName, description string) (*StoredGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if groupName == "" {
		return nil, errInvalid("GroupName is required")
	}
	if namespace == "" {
		namespace = "default"
	}
	if s.groups[namespace] == nil {
		s.groups[namespace] = make(map[string]*StoredGroup)
	}
	if _, ok := s.groups[namespace][groupName]; ok {
		return nil, errExists("Group", groupName)
	}
	g := &StoredGroup{
		GroupName:    groupName,
		AwsAccountId: s.accountID,
		Namespace:    namespace,
		Arn:          s.arnGroup(namespace, groupName),
		Description:  description,
		PrincipalId:  "group-" + generateID(),
	}
	s.groups[namespace][groupName] = g
	return g, nil
}

func (s *Store) GetGroup(namespace, groupName string) (*StoredGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.groups[namespace]
	if !ok {
		return nil, errNotFound("Group", groupName)
	}
	g, ok := ns[groupName]
	if !ok {
		return nil, errNotFound("Group", groupName)
	}
	return g, nil
}

func (s *Store) UpdateGroup(namespace, groupName, description string) (*StoredGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.groups[namespace]
	if !ok {
		return nil, errNotFound("Group", groupName)
	}
	g, ok := ns[groupName]
	if !ok {
		return nil, errNotFound("Group", groupName)
	}
	if description != "" {
		g.Description = description
	}
	return g, nil
}

func (s *Store) DeleteGroup(namespace, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.groups[namespace]
	if !ok {
		return errNotFound("Group", groupName)
	}
	if _, ok := ns[groupName]; !ok {
		return errNotFound("Group", groupName)
	}
	delete(ns, groupName)
	return nil
}

func (s *Store) ListGroups(namespace string) []*StoredGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	out := []*StoredGroup{}
	for _, g := range s.groups[namespace] {
		out = append(out, g)
	}
	return out
}

func (s *Store) AddGroupMember(namespace, groupName, userName string) (*StoredGroup, *StoredUser, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	gns, ok := s.groups[namespace]
	if !ok {
		return nil, nil, errNotFound("Group", groupName)
	}
	g, ok := gns[groupName]
	if !ok {
		return nil, nil, errNotFound("Group", groupName)
	}
	uns, ok := s.users[namespace]
	if !ok {
		return nil, nil, errNotFound("User", userName)
	}
	u, ok := uns[userName]
	if !ok {
		return nil, nil, errNotFound("User", userName)
	}
	for _, m := range g.Members {
		if m == userName {
			return g, u, nil
		}
	}
	g.Members = append(g.Members, userName)
	return g, u, nil
}

func (s *Store) RemoveGroupMember(namespace, groupName, userName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	gns, ok := s.groups[namespace]
	if !ok {
		return errNotFound("Group", groupName)
	}
	g, ok := gns[groupName]
	if !ok {
		return errNotFound("Group", groupName)
	}
	for i, m := range g.Members {
		if m == userName {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			return nil
		}
	}
	return errNotFound("Member", userName)
}

func (s *Store) IsGroupMember(namespace, groupName, userName string) (bool, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	gns, ok := s.groups[namespace]
	if !ok {
		return false, errNotFound("Group", groupName)
	}
	g, ok := gns[groupName]
	if !ok {
		return false, errNotFound("Group", groupName)
	}
	for _, m := range g.Members {
		if m == userName {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) ListGroupsForUser(namespace, userName string) []*StoredGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	out := []*StoredGroup{}
	for _, g := range s.groups[namespace] {
		for _, m := range g.Members {
			if m == userName {
				out = append(out, g)
				break
			}
		}
	}
	return out
}

func (s *Store) ListGroupMembers(namespace, groupName string) ([]*StoredUser, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	gns, ok := s.groups[namespace]
	if !ok {
		return nil, errNotFound("Group", groupName)
	}
	g, ok := gns[groupName]
	if !ok {
		return nil, errNotFound("Group", groupName)
	}
	out := []*StoredUser{}
	uns := s.users[namespace]
	for _, name := range g.Members {
		if u, ok := uns[name]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

// ── Namespaces ───────────────────────────────────────────────────────────────

func (s *Store) CreateNamespace(name, identityStore string) (*StoredNamespace, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, errInvalid("Namespace is required")
	}
	if _, ok := s.namespaces[name]; ok {
		return nil, errExists("Namespace", name)
	}
	if identityStore == "" {
		identityStore = "QUICKSIGHT"
	}
	ns := &StoredNamespace{
		Name:           name,
		Arn:            s.arnNamespace(name),
		CapacityRegion: s.region,
		CreationStatus: "CREATED",
		IdentityStore:  identityStore,
	}
	s.namespaces[name] = ns
	return ns, nil
}

func (s *Store) GetNamespace(name string) (*StoredNamespace, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[name]
	if !ok {
		return nil, errNotFound("Namespace", name)
	}
	return ns, nil
}

func (s *Store) DeleteNamespace(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namespaces[name]; !ok {
		return errNotFound("Namespace", name)
	}
	delete(s.namespaces, name)
	return nil
}

func (s *Store) ListNamespaces() []*StoredNamespace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredNamespace, 0, len(s.namespaces))
	for _, n := range s.namespaces {
		out = append(out, n)
	}
	return out
}

// ── DataSource ───────────────────────────────────────────────────────────────

func (s *Store) CreateDataSource(id, name, dsType string, params, vpcProps, sslProps map[string]any,
	creds map[string]any, alternateParams []map[string]any) (*StoredDataSource, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("DataSourceId is required")
	}
	if _, ok := s.dataSources[id]; ok {
		return nil, errExists("DataSource", id)
	}
	now := nowUTC()
	d := &StoredDataSource{
		DataSourceId:                  id,
		AwsAccountId:                  s.accountID,
		Arn:                           s.arnDataSource(id),
		Name:                          name,
		Type:                          dsType,
		Status:                        "CREATION_SUCCESSFUL",
		CreatedTime:                   now,
		LastUpdatedTime:               now,
		DataSourceParameters:          params,
		AlternateDataSourceParameters: alternateParams,
		VpcConnectionProperties:       vpcProps,
		SslProperties:                 sslProps,
		Credentials:                   creds,
	}
	s.dataSources[id] = d
	return d, nil
}

func (s *Store) GetDataSource(id string) (*StoredDataSource, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.dataSources[id]
	if !ok {
		return nil, errNotFound("DataSource", id)
	}
	return d, nil
}

func (s *Store) UpdateDataSource(id, name string, params, vpcProps, sslProps map[string]any,
	creds map[string]any) (*StoredDataSource, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dataSources[id]
	if !ok {
		return nil, errNotFound("DataSource", id)
	}
	if name != "" {
		d.Name = name
	}
	if params != nil {
		d.DataSourceParameters = params
	}
	if vpcProps != nil {
		d.VpcConnectionProperties = vpcProps
	}
	if sslProps != nil {
		d.SslProperties = sslProps
	}
	if creds != nil {
		d.Credentials = creds
	}
	d.LastUpdatedTime = nowUTC()
	d.Status = "UPDATE_SUCCESSFUL"
	return d, nil
}

func (s *Store) DeleteDataSource(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataSources[id]; !ok {
		return errNotFound("DataSource", id)
	}
	delete(s.dataSources, id)
	return nil
}

func (s *Store) ListDataSources() []*StoredDataSource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredDataSource, 0, len(s.dataSources))
	for _, d := range s.dataSources {
		out = append(out, d)
	}
	return out
}

// ── DataSet ──────────────────────────────────────────────────────────────────

func (s *Store) CreateDataSet(id, name, importMode string, physicalTableMap, logicalTableMap map[string]any,
	columnGroups []map[string]any, fieldFolders map[string]any,
	rowLevelPermDS, rowLevelPermTagCfg map[string]any, columnLevelPermRules []map[string]any,
	dataSetUsageCfg map[string]any, datasetParameters []map[string]any,
	folderArns []string) (*StoredDataSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("DataSetId is required")
	}
	if _, ok := s.dataSets[id]; ok {
		return nil, errExists("DataSet", id)
	}
	now := nowUTC()
	if importMode == "" {
		importMode = "DIRECT_QUERY"
	}
	d := &StoredDataSet{
		DataSetId:                          id,
		AwsAccountId:                       s.accountID,
		Arn:                                s.arnDataSet(id),
		Name:                               name,
		CreatedTime:                        now,
		LastUpdatedTime:                    now,
		PhysicalTableMap:                   physicalTableMap,
		LogicalTableMap:                    logicalTableMap,
		ImportMode:                         importMode,
		ConsumedSpiceCapacityInBytes:       0,
		ColumnGroups:                       columnGroups,
		FieldFolders:                       fieldFolders,
		RowLevelPermissionDataSet:          rowLevelPermDS,
		RowLevelPermissionTagConfiguration: rowLevelPermTagCfg,
		ColumnLevelPermissionRules:         columnLevelPermRules,
		DataSetUsageConfiguration:          dataSetUsageCfg,
		DatasetParameters:                  datasetParameters,
		FolderArns:                         folderArns,
	}
	s.dataSets[id] = d
	return d, nil
}

func (s *Store) GetDataSet(id string) (*StoredDataSet, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.dataSets[id]
	if !ok {
		return nil, errNotFound("DataSet", id)
	}
	return d, nil
}

func (s *Store) UpdateDataSet(id, name, importMode string, physicalTableMap, logicalTableMap map[string]any,
	columnGroups []map[string]any, fieldFolders map[string]any,
	rowLevelPermDS, rowLevelPermTagCfg map[string]any, columnLevelPermRules []map[string]any,
	dataSetUsageCfg map[string]any, datasetParameters []map[string]any) (*StoredDataSet, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dataSets[id]
	if !ok {
		return nil, errNotFound("DataSet", id)
	}
	if name != "" {
		d.Name = name
	}
	if importMode != "" {
		d.ImportMode = importMode
	}
	if physicalTableMap != nil {
		d.PhysicalTableMap = physicalTableMap
	}
	if logicalTableMap != nil {
		d.LogicalTableMap = logicalTableMap
	}
	if columnGroups != nil {
		d.ColumnGroups = columnGroups
	}
	if fieldFolders != nil {
		d.FieldFolders = fieldFolders
	}
	if rowLevelPermDS != nil {
		d.RowLevelPermissionDataSet = rowLevelPermDS
	}
	if rowLevelPermTagCfg != nil {
		d.RowLevelPermissionTagConfiguration = rowLevelPermTagCfg
	}
	if columnLevelPermRules != nil {
		d.ColumnLevelPermissionRules = columnLevelPermRules
	}
	if dataSetUsageCfg != nil {
		d.DataSetUsageConfiguration = dataSetUsageCfg
	}
	if datasetParameters != nil {
		d.DatasetParameters = datasetParameters
	}
	d.LastUpdatedTime = nowUTC()
	return d, nil
}

func (s *Store) DeleteDataSet(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataSets[id]; !ok {
		return errNotFound("DataSet", id)
	}
	delete(s.dataSets, id)
	delete(s.refreshSchedules, id)
	delete(s.ingestions, id)
	return nil
}

func (s *Store) ListDataSets() []*StoredDataSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredDataSet, 0, len(s.dataSets))
	for _, d := range s.dataSets {
		out = append(out, d)
	}
	return out
}

// PutDataSetRefreshProperties sets properties on a dataset.
func (s *Store) PutDataSetRefreshProperties(datasetId string, props map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dataSets[datasetId]
	if !ok {
		return errNotFound("DataSet", datasetId)
	}
	d.RefreshProperties = props
	return nil
}

func (s *Store) GetDataSetRefreshProperties(datasetId string) (map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.dataSets[datasetId]
	if !ok {
		return nil, errNotFound("DataSet", datasetId)
	}
	return d.RefreshProperties, nil
}

func (s *Store) DeleteDataSetRefreshProperties(datasetId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dataSets[datasetId]
	if !ok {
		return errNotFound("DataSet", datasetId)
	}
	d.RefreshProperties = nil
	return nil
}

// ── Template ─────────────────────────────────────────────────────────────────

func (s *Store) CreateTemplate(id, name string, version map[string]any, definition map[string]any) (*StoredTemplate, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("TemplateId is required")
	}
	if _, ok := s.templates[id]; ok {
		return nil, errExists("Template", id)
	}
	now := nowUTC()
	if version == nil {
		version = map[string]any{}
	}
	version["VersionNumber"] = 1
	version["Status"] = "CREATION_SUCCESSFUL"
	version["CreatedTime"] = now.Format(time.RFC3339)
	t := &StoredTemplate{
		TemplateId:      id,
		AwsAccountId:    s.accountID,
		Arn:             s.arnTemplate(id),
		Name:            name,
		CreatedTime:     now,
		LastUpdatedTime: now,
		Version:         version,
		VersionNumber:   1,
		Versions:        []map[string]any{version},
		Aliases:         make(map[string]map[string]any),
		Definition:      definition,
	}
	s.templates[id] = t
	return t, nil
}

func (s *Store) GetTemplate(id string) (*StoredTemplate, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, errNotFound("Template", id)
	}
	return t, nil
}

func (s *Store) UpdateTemplate(id, name string, version map[string]any, definition map[string]any) (*StoredTemplate, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, errNotFound("Template", id)
	}
	if name != "" {
		t.Name = name
	}
	t.VersionNumber++
	now := nowUTC()
	if version == nil {
		version = map[string]any{}
	}
	version["VersionNumber"] = t.VersionNumber
	version["Status"] = "CREATION_SUCCESSFUL"
	version["CreatedTime"] = now.Format(time.RFC3339)
	t.Version = version
	t.Versions = append(t.Versions, version)
	if definition != nil {
		t.Definition = definition
	}
	t.LastUpdatedTime = now
	return t, nil
}

func (s *Store) DeleteTemplate(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[id]; !ok {
		return errNotFound("Template", id)
	}
	delete(s.templates, id)
	return nil
}

func (s *Store) ListTemplates() []*StoredTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredTemplate, 0, len(s.templates))
	for _, t := range s.templates {
		out = append(out, t)
	}
	return out
}

func (s *Store) CreateTemplateAlias(templateId, aliasName string, versionNumber int) (map[string]any, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.templates[templateId]
	if !ok {
		return nil, errNotFound("Template", templateId)
	}
	if _, exists := t.Aliases[aliasName]; exists {
		return nil, errExists("TemplateAlias", aliasName)
	}
	alias := map[string]any{
		"AliasName":             aliasName,
		"Arn":                   s.arnTemplateAlias(templateId, aliasName),
		"TemplateVersionNumber": versionNumber,
	}
	t.Aliases[aliasName] = alias
	return alias, nil
}

func (s *Store) GetTemplateAlias(templateId, aliasName string) (map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[templateId]
	if !ok {
		return nil, errNotFound("Template", templateId)
	}
	a, ok := t.Aliases[aliasName]
	if !ok {
		return nil, errNotFound("TemplateAlias", aliasName)
	}
	return a, nil
}

func (s *Store) UpdateTemplateAlias(templateId, aliasName string, versionNumber int) (map[string]any, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.templates[templateId]
	if !ok {
		return nil, errNotFound("Template", templateId)
	}
	a, ok := t.Aliases[aliasName]
	if !ok {
		return nil, errNotFound("TemplateAlias", aliasName)
	}
	a["TemplateVersionNumber"] = versionNumber
	return a, nil
}

func (s *Store) DeleteTemplateAlias(templateId, aliasName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.templates[templateId]
	if !ok {
		return errNotFound("Template", templateId)
	}
	if _, ok := t.Aliases[aliasName]; !ok {
		return errNotFound("TemplateAlias", aliasName)
	}
	delete(t.Aliases, aliasName)
	return nil
}

func (s *Store) ListTemplateAliases(templateId string) ([]map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[templateId]
	if !ok {
		return nil, errNotFound("Template", templateId)
	}
	out := make([]map[string]any, 0, len(t.Aliases))
	for _, a := range t.Aliases {
		out = append(out, a)
	}
	return out, nil
}

// ── Dashboard ────────────────────────────────────────────────────────────────

func (s *Store) CreateDashboard(id, name string, version map[string]any, publishOptions map[string]any,
	definition map[string]any, validationStrategy map[string]any, linkEntities []string) (*StoredDashboard, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("DashboardId is required")
	}
	if _, ok := s.dashboards[id]; ok {
		return nil, errExists("Dashboard", id)
	}
	now := nowUTC()
	if version == nil {
		version = map[string]any{}
	}
	version["VersionNumber"] = 1
	version["Status"] = "CREATION_SUCCESSFUL"
	version["CreatedTime"] = now.Format(time.RFC3339)
	d := &StoredDashboard{
		DashboardId:             id,
		AwsAccountId:            s.accountID,
		Arn:                     s.arnDashboard(id),
		Name:                    name,
		CreatedTime:             now,
		LastUpdatedTime:         now,
		LastPublishedTime:       now,
		Version:                 version,
		VersionNumber:           1,
		Versions:                []map[string]any{version},
		LinkEntities:            linkEntities,
		DashboardPublishOptions: publishOptions,
		Definition:              definition,
		ValidationStrategy:      validationStrategy,
		PublishedVersionNumber:  1,
	}
	s.dashboards[id] = d
	return d, nil
}

func (s *Store) GetDashboard(id string) (*StoredDashboard, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.dashboards[id]
	if !ok {
		return nil, errNotFound("Dashboard", id)
	}
	return d, nil
}

func (s *Store) UpdateDashboard(id, name string, version map[string]any, publishOptions, definition,
	validationStrategy map[string]any) (*StoredDashboard, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[id]
	if !ok {
		return nil, errNotFound("Dashboard", id)
	}
	if name != "" {
		d.Name = name
	}
	d.VersionNumber++
	now := nowUTC()
	if version == nil {
		version = map[string]any{}
	}
	version["VersionNumber"] = d.VersionNumber
	version["Status"] = "CREATION_SUCCESSFUL"
	version["CreatedTime"] = now.Format(time.RFC3339)
	d.Version = version
	d.Versions = append(d.Versions, version)
	if publishOptions != nil {
		d.DashboardPublishOptions = publishOptions
	}
	if definition != nil {
		d.Definition = definition
	}
	if validationStrategy != nil {
		d.ValidationStrategy = validationStrategy
	}
	d.LastUpdatedTime = now
	return d, nil
}

func (s *Store) UpdateDashboardLinks(id string, linkEntities []string) (*StoredDashboard, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[id]
	if !ok {
		return nil, errNotFound("Dashboard", id)
	}
	d.LinkEntities = linkEntities
	return d, nil
}

func (s *Store) UpdateDashboardPublishedVersion(id string, versionNumber int) (*StoredDashboard, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.dashboards[id]
	if !ok {
		return nil, errNotFound("Dashboard", id)
	}
	d.PublishedVersionNumber = versionNumber
	d.LastPublishedTime = nowUTC()
	return d, nil
}

func (s *Store) DeleteDashboard(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dashboards[id]; !ok {
		return errNotFound("Dashboard", id)
	}
	delete(s.dashboards, id)
	return nil
}

func (s *Store) ListDashboards() []*StoredDashboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredDashboard, 0, len(s.dashboards))
	for _, d := range s.dashboards {
		out = append(out, d)
	}
	return out
}

// ── Analysis ─────────────────────────────────────────────────────────────────

func (s *Store) CreateAnalysis(id, name string, sourceEntity, definition map[string]any,
	themeArn string, parameters map[string]any) (*StoredAnalysis, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("AnalysisId is required")
	}
	if _, ok := s.analyses[id]; ok {
		return nil, errExists("Analysis", id)
	}
	now := nowUTC()
	a := &StoredAnalysis{
		AnalysisId:      id,
		AwsAccountId:    s.accountID,
		Arn:             s.arnAnalysis(id),
		Name:            name,
		Status:          "CREATION_SUCCESSFUL",
		ThemeArn:        themeArn,
		CreatedTime:     now,
		LastUpdatedTime: now,
		Definition:      definition,
	}
	s.analyses[id] = a
	return a, nil
}

func (s *Store) GetAnalysis(id string) (*StoredAnalysis, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.analyses[id]
	if !ok {
		return nil, errNotFound("Analysis", id)
	}
	return a, nil
}

func (s *Store) UpdateAnalysis(id, name string, sourceEntity, definition map[string]any,
	themeArn string, parameters map[string]any) (*StoredAnalysis, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.analyses[id]
	if !ok {
		return nil, errNotFound("Analysis", id)
	}
	if name != "" {
		a.Name = name
	}
	if definition != nil {
		a.Definition = definition
	}
	if themeArn != "" {
		a.ThemeArn = themeArn
	}
	a.LastUpdatedTime = nowUTC()
	return a, nil
}

func (s *Store) DeleteAnalysis(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.analyses[id]
	if !ok {
		return errNotFound("Analysis", id)
	}
	a.Status = "DELETED"
	return nil
}

func (s *Store) RestoreAnalysis(id string) (*StoredAnalysis, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.analyses[id]
	if !ok {
		return nil, errNotFound("Analysis", id)
	}
	a.Status = "CREATION_SUCCESSFUL"
	return a, nil
}

func (s *Store) ListAnalyses() []*StoredAnalysis {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAnalysis, 0, len(s.analyses))
	for _, a := range s.analyses {
		out = append(out, a)
	}
	return out
}

// ── Theme ────────────────────────────────────────────────────────────────────

func (s *Store) CreateTheme(id, name, baseThemeId string, configuration map[string]any, version map[string]any) (*StoredTheme, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("ThemeId is required")
	}
	if _, ok := s.themes[id]; ok {
		return nil, errExists("Theme", id)
	}
	now := nowUTC()
	if version == nil {
		version = map[string]any{}
	}
	version["VersionNumber"] = 1
	version["Status"] = "CREATION_SUCCESSFUL"
	version["CreatedTime"] = now.Format(time.RFC3339)
	version["BaseThemeId"] = baseThemeId
	version["Configuration"] = configuration
	t := &StoredTheme{
		ThemeId:         id,
		AwsAccountId:    s.accountID,
		Arn:             s.arnTheme(id),
		Name:            name,
		Type:            "CUSTOM",
		BaseThemeId:     baseThemeId,
		CreatedTime:     now,
		LastUpdatedTime: now,
		Version:         version,
		VersionNumber:   1,
		Versions:        []map[string]any{version},
		Aliases:         make(map[string]map[string]any),
		Configuration:   configuration,
	}
	s.themes[id] = t
	return t, nil
}

func (s *Store) GetTheme(id string) (*StoredTheme, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.themes[id]
	if !ok {
		return nil, errNotFound("Theme", id)
	}
	return t, nil
}

func (s *Store) UpdateTheme(id, name, baseThemeId string, configuration map[string]any) (*StoredTheme, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.themes[id]
	if !ok {
		return nil, errNotFound("Theme", id)
	}
	if name != "" {
		t.Name = name
	}
	if baseThemeId != "" {
		t.BaseThemeId = baseThemeId
	}
	if configuration != nil {
		t.Configuration = configuration
	}
	t.VersionNumber++
	now := nowUTC()
	v := map[string]any{
		"VersionNumber": t.VersionNumber,
		"Status":        "CREATION_SUCCESSFUL",
		"CreatedTime":   now.Format(time.RFC3339),
		"BaseThemeId":   t.BaseThemeId,
		"Configuration": t.Configuration,
	}
	t.Version = v
	t.Versions = append(t.Versions, v)
	t.LastUpdatedTime = now
	return t, nil
}

func (s *Store) DeleteTheme(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.themes[id]; !ok {
		return errNotFound("Theme", id)
	}
	delete(s.themes, id)
	return nil
}

func (s *Store) ListThemes() []*StoredTheme {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredTheme, 0, len(s.themes))
	for _, t := range s.themes {
		out = append(out, t)
	}
	return out
}

func (s *Store) CreateThemeAlias(themeId, aliasName string, versionNumber int) (map[string]any, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.themes[themeId]
	if !ok {
		return nil, errNotFound("Theme", themeId)
	}
	if _, exists := t.Aliases[aliasName]; exists {
		return nil, errExists("ThemeAlias", aliasName)
	}
	alias := map[string]any{
		"AliasName":          aliasName,
		"Arn":                s.arnThemeAlias(themeId, aliasName),
		"ThemeVersionNumber": versionNumber,
	}
	t.Aliases[aliasName] = alias
	return alias, nil
}

func (s *Store) GetThemeAlias(themeId, aliasName string) (map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.themes[themeId]
	if !ok {
		return nil, errNotFound("Theme", themeId)
	}
	a, ok := t.Aliases[aliasName]
	if !ok {
		return nil, errNotFound("ThemeAlias", aliasName)
	}
	return a, nil
}

func (s *Store) UpdateThemeAlias(themeId, aliasName string, versionNumber int) (map[string]any, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.themes[themeId]
	if !ok {
		return nil, errNotFound("Theme", themeId)
	}
	a, ok := t.Aliases[aliasName]
	if !ok {
		return nil, errNotFound("ThemeAlias", aliasName)
	}
	a["ThemeVersionNumber"] = versionNumber
	return a, nil
}

func (s *Store) DeleteThemeAlias(themeId, aliasName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.themes[themeId]
	if !ok {
		return errNotFound("Theme", themeId)
	}
	if _, ok := t.Aliases[aliasName]; !ok {
		return errNotFound("ThemeAlias", aliasName)
	}
	delete(t.Aliases, aliasName)
	return nil
}

func (s *Store) ListThemeAliases(themeId string) ([]map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.themes[themeId]
	if !ok {
		return nil, errNotFound("Theme", themeId)
	}
	out := make([]map[string]any, 0, len(t.Aliases))
	for _, a := range t.Aliases {
		out = append(out, a)
	}
	return out, nil
}

// ── Folder ───────────────────────────────────────────────────────────────────

func (s *Store) CreateFolder(id, name, folderType, parentFolderArn, sharingModel string) (*StoredFolder, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("FolderId is required")
	}
	if _, ok := s.folders[id]; ok {
		return nil, errExists("Folder", id)
	}
	now := nowUTC()
	if folderType == "" {
		folderType = "SHARED"
	}
	if sharingModel == "" {
		sharingModel = "ACCOUNT"
	}
	f := &StoredFolder{
		FolderId:        id,
		AwsAccountId:    s.accountID,
		Arn:             s.arnFolder(id),
		Name:            name,
		FolderType:      folderType,
		FolderPath:      []string{id},
		CreatedTime:     now,
		LastUpdatedTime: now,
		ParentFolderArn: parentFolderArn,
		SharingModel:    sharingModel,
		Members:         []map[string]any{},
	}
	s.folders[id] = f
	return f, nil
}

func (s *Store) GetFolder(id string) (*StoredFolder, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.folders[id]
	if !ok {
		return nil, errNotFound("Folder", id)
	}
	return f, nil
}

func (s *Store) UpdateFolder(id, name string) (*StoredFolder, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.folders[id]
	if !ok {
		return nil, errNotFound("Folder", id)
	}
	if name != "" {
		f.Name = name
	}
	f.LastUpdatedTime = nowUTC()
	return f, nil
}

func (s *Store) DeleteFolder(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.folders[id]; !ok {
		return errNotFound("Folder", id)
	}
	delete(s.folders, id)
	return nil
}

func (s *Store) ListFolders() []*StoredFolder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFolder, 0, len(s.folders))
	for _, f := range s.folders {
		out = append(out, f)
	}
	return out
}

func (s *Store) CreateFolderMembership(folderId, memberId, memberType string) (map[string]any, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.folders[folderId]
	if !ok {
		return nil, errNotFound("Folder", folderId)
	}
	for _, m := range f.Members {
		if m["MemberId"] == memberId {
			return m, nil
		}
	}
	m := map[string]any{
		"MemberId":   memberId,
		"MemberType": memberType,
	}
	f.Members = append(f.Members, m)
	return m, nil
}

func (s *Store) DeleteFolderMembership(folderId, memberId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.folders[folderId]
	if !ok {
		return errNotFound("Folder", folderId)
	}
	for i, m := range f.Members {
		if m["MemberId"] == memberId {
			f.Members = append(f.Members[:i], f.Members[i+1:]...)
			return nil
		}
	}
	return errNotFound("FolderMember", memberId)
}

func (s *Store) ListFolderMembers(folderId string) ([]map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.folders[folderId]
	if !ok {
		return nil, errNotFound("Folder", folderId)
	}
	return f.Members, nil
}

// ── Topic ────────────────────────────────────────────────────────────────────

func (s *Store) CreateTopic(id, name, description, userExperienceVersion string,
	dataSets []map[string]any, configOptions map[string]any) (*StoredTopic, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("TopicId is required")
	}
	if _, ok := s.topics[id]; ok {
		return nil, errExists("Topic", id)
	}
	now := nowUTC()
	if userExperienceVersion == "" {
		userExperienceVersion = "NEW_READER_EXPERIENCE"
	}
	t := &StoredTopic{
		TopicId:               id,
		AwsAccountId:          s.accountID,
		Arn:                   s.arnTopic(id),
		Name:                  name,
		Description:           description,
		UserExperienceVersion: userExperienceVersion,
		DataSets:              dataSets,
		ConfigOptions:         configOptions,
		CreatedTime:           now,
		LastUpdatedTime:       now,
	}
	s.topics[id] = t
	return t, nil
}

func (s *Store) GetTopic(id string) (*StoredTopic, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.topics[id]
	if !ok {
		return nil, errNotFound("Topic", id)
	}
	return t, nil
}

func (s *Store) UpdateTopic(id, name, description string, dataSets []map[string]any) (*StoredTopic, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.topics[id]
	if !ok {
		return nil, errNotFound("Topic", id)
	}
	if name != "" {
		t.Name = name
	}
	if description != "" {
		t.Description = description
	}
	if dataSets != nil {
		t.DataSets = dataSets
	}
	t.LastUpdatedTime = nowUTC()
	return t, nil
}

func (s *Store) DeleteTopic(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.topics[id]; !ok {
		return errNotFound("Topic", id)
	}
	delete(s.topics, id)
	return nil
}

func (s *Store) ListTopics() []*StoredTopic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredTopic, 0, len(s.topics))
	for _, t := range s.topics {
		out = append(out, t)
	}
	return out
}

func (s *Store) SetTopicRefreshSchedule(topicId string, schedule map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.topics[topicId]
	if !ok {
		return errNotFound("Topic", topicId)
	}
	t.RefreshSchedule = schedule
	return nil
}

func (s *Store) GetTopicRefreshSchedule(topicId string) (map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.topics[topicId]
	if !ok {
		return nil, errNotFound("Topic", topicId)
	}
	return t.RefreshSchedule, nil
}

func (s *Store) DeleteTopicRefreshSchedule(topicId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.topics[topicId]
	if !ok {
		return errNotFound("Topic", topicId)
	}
	t.RefreshSchedule = nil
	return nil
}

func (s *Store) AddTopicReviewedAnswer(topicId string, answers []map[string]any) ([]map[string]any, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.topics[topicId]
	if !ok {
		return nil, errNotFound("Topic", topicId)
	}
	t.ReviewedAnswers = append(t.ReviewedAnswers, answers...)
	return t.ReviewedAnswers, nil
}

func (s *Store) DeleteTopicReviewedAnswer(topicId string, answerIds []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.topics[topicId]
	if !ok {
		return errNotFound("Topic", topicId)
	}
	keep := []map[string]any{}
	for _, ans := range t.ReviewedAnswers {
		id, _ := ans["AnswerId"].(string)
		matched := false
		for _, aid := range answerIds {
			if aid == id {
				matched = true
				break
			}
		}
		if !matched {
			keep = append(keep, ans)
		}
	}
	t.ReviewedAnswers = keep
	return nil
}

func (s *Store) ListTopicReviewedAnswers(topicId string) ([]map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.topics[topicId]
	if !ok {
		return nil, errNotFound("Topic", topicId)
	}
	return t.ReviewedAnswers, nil
}

// ── IAMPolicyAssignment ──────────────────────────────────────────────────────

func (s *Store) CreateIAMPolicyAssignment(namespace, name, status, policyArn string,
	identities map[string][]string) (*StoredIAMPolicyAssignment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, errInvalid("AssignmentName is required")
	}
	if namespace == "" {
		namespace = "default"
	}
	if s.iamPolicyAssignments[namespace] == nil {
		s.iamPolicyAssignments[namespace] = make(map[string]*StoredIAMPolicyAssignment)
	}
	if _, ok := s.iamPolicyAssignments[namespace][name]; ok {
		return nil, errExists("IAMPolicyAssignment", name)
	}
	a := &StoredIAMPolicyAssignment{
		AssignmentName:   name,
		AssignmentId:     "assignment-" + generateID(),
		AssignmentStatus: status,
		PolicyArn:        policyArn,
		Identities:       identities,
		AwsAccountId:     s.accountID,
		Namespace:        namespace,
	}
	s.iamPolicyAssignments[namespace][name] = a
	return a, nil
}

func (s *Store) GetIAMPolicyAssignment(namespace, name string) (*StoredIAMPolicyAssignment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.iamPolicyAssignments[namespace]
	if !ok {
		return nil, errNotFound("IAMPolicyAssignment", name)
	}
	a, ok := ns[name]
	if !ok {
		return nil, errNotFound("IAMPolicyAssignment", name)
	}
	return a, nil
}

func (s *Store) UpdateIAMPolicyAssignment(namespace, name, status, policyArn string,
	identities map[string][]string) (*StoredIAMPolicyAssignment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.iamPolicyAssignments[namespace]
	if !ok {
		return nil, errNotFound("IAMPolicyAssignment", name)
	}
	a, ok := ns[name]
	if !ok {
		return nil, errNotFound("IAMPolicyAssignment", name)
	}
	if status != "" {
		a.AssignmentStatus = status
	}
	if policyArn != "" {
		a.PolicyArn = policyArn
	}
	if identities != nil {
		a.Identities = identities
	}
	return a, nil
}

func (s *Store) DeleteIAMPolicyAssignment(namespace, name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	ns, ok := s.iamPolicyAssignments[namespace]
	if !ok {
		return errNotFound("IAMPolicyAssignment", name)
	}
	if _, ok := ns[name]; !ok {
		return errNotFound("IAMPolicyAssignment", name)
	}
	delete(ns, name)
	return nil
}

func (s *Store) ListIAMPolicyAssignments(namespace string) []*StoredIAMPolicyAssignment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	out := []*StoredIAMPolicyAssignment{}
	for _, a := range s.iamPolicyAssignments[namespace] {
		out = append(out, a)
	}
	return out
}

// ── Ingestion ────────────────────────────────────────────────────────────────

func (s *Store) CreateIngestion(datasetId, ingestionId, ingestionType string) (*StoredIngestion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if datasetId == "" || ingestionId == "" {
		return nil, errInvalid("DataSetId and IngestionId are required")
	}
	if _, ok := s.dataSets[datasetId]; !ok {
		return nil, errNotFound("DataSet", datasetId)
	}
	if s.ingestions[datasetId] == nil {
		s.ingestions[datasetId] = make(map[string]*StoredIngestion)
	}
	if _, ok := s.ingestions[datasetId][ingestionId]; ok {
		return nil, errExists("Ingestion", ingestionId)
	}
	if ingestionType == "" {
		ingestionType = "FULL_REFRESH"
	}
	i := &StoredIngestion{
		IngestionId:     ingestionId,
		DataSetId:       datasetId,
		AwsAccountId:    s.accountID,
		Arn:             s.arnIngestion(datasetId, ingestionId),
		IngestionStatus: "INITIALIZED",
		CreatedTime:     nowUTC(),
		RequestSource:   "MANUAL",
		RequestType:     ingestionType,
	}
	s.ingestions[datasetId][ingestionId] = i
	return i, nil
}

func (s *Store) DescribeIngestion(datasetId, ingestionId string) (*StoredIngestion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ds, ok := s.ingestions[datasetId]
	if !ok {
		return nil, errNotFound("Ingestion", ingestionId)
	}
	i, ok := ds[ingestionId]
	if !ok {
		return nil, errNotFound("Ingestion", ingestionId)
	}
	// Advance state on each describe to simulate progress.
	switch i.IngestionStatus {
	case "INITIALIZED":
		i.IngestionStatus = "QUEUED"
	case "QUEUED":
		i.IngestionStatus = "RUNNING"
	case "RUNNING":
		i.IngestionStatus = "COMPLETED"
		i.IngestionTimeInSeconds = 12
		i.IngestionSizeInBytes = 1024
		i.RowInfo = map[string]any{"RowsIngested": 100, "RowsDropped": 0}
	}
	return i, nil
}

func (s *Store) CancelIngestion(datasetId, ingestionId string) (*StoredIngestion, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ds, ok := s.ingestions[datasetId]
	if !ok {
		return nil, errNotFound("Ingestion", ingestionId)
	}
	i, ok := ds[ingestionId]
	if !ok {
		return nil, errNotFound("Ingestion", ingestionId)
	}
	i.IngestionStatus = "CANCELLED"
	return i, nil
}

func (s *Store) ListIngestions(datasetId string) []*StoredIngestion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*StoredIngestion{}
	for _, i := range s.ingestions[datasetId] {
		out = append(out, i)
	}
	return out
}

// ── RefreshSchedule ──────────────────────────────────────────────────────────

func (s *Store) CreateRefreshSchedule(datasetId string, schedule map[string]any) (*StoredRefreshSchedule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if datasetId == "" || schedule == nil {
		return nil, errInvalid("DataSetId and Schedule are required")
	}
	scheduleId, _ := schedule["ScheduleId"].(string)
	if scheduleId == "" {
		return nil, errInvalid("ScheduleId is required")
	}
	if _, ok := s.dataSets[datasetId]; !ok {
		return nil, errNotFound("DataSet", datasetId)
	}
	if s.refreshSchedules[datasetId] == nil {
		s.refreshSchedules[datasetId] = make(map[string]*StoredRefreshSchedule)
	}
	if _, ok := s.refreshSchedules[datasetId][scheduleId]; ok {
		return nil, errExists("RefreshSchedule", scheduleId)
	}
	freq, _ := schedule["ScheduleFrequency"].(map[string]any)
	refreshType, _ := schedule["RefreshType"].(string)
	r := &StoredRefreshSchedule{
		ScheduleId:        scheduleId,
		DataSetId:         datasetId,
		AwsAccountId:      s.accountID,
		Arn:               s.arnRefreshSchedule(datasetId, scheduleId),
		ScheduleFrequency: freq,
		RefreshType:       refreshType,
	}
	s.refreshSchedules[datasetId][scheduleId] = r
	return r, nil
}

func (s *Store) GetRefreshSchedule(datasetId, scheduleId string) (*StoredRefreshSchedule, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ds, ok := s.refreshSchedules[datasetId]
	if !ok {
		return nil, errNotFound("RefreshSchedule", scheduleId)
	}
	r, ok := ds[scheduleId]
	if !ok {
		return nil, errNotFound("RefreshSchedule", scheduleId)
	}
	return r, nil
}

func (s *Store) UpdateRefreshSchedule(datasetId string, schedule map[string]any) (*StoredRefreshSchedule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	scheduleId, _ := schedule["ScheduleId"].(string)
	ds, ok := s.refreshSchedules[datasetId]
	if !ok {
		return nil, errNotFound("RefreshSchedule", scheduleId)
	}
	r, ok := ds[scheduleId]
	if !ok {
		return nil, errNotFound("RefreshSchedule", scheduleId)
	}
	if freq, ok := schedule["ScheduleFrequency"].(map[string]any); ok {
		r.ScheduleFrequency = freq
	}
	if refreshType, ok := schedule["RefreshType"].(string); ok && refreshType != "" {
		r.RefreshType = refreshType
	}
	return r, nil
}

func (s *Store) DeleteRefreshSchedule(datasetId, scheduleId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ds, ok := s.refreshSchedules[datasetId]
	if !ok {
		return errNotFound("RefreshSchedule", scheduleId)
	}
	if _, ok := ds[scheduleId]; !ok {
		return errNotFound("RefreshSchedule", scheduleId)
	}
	delete(ds, scheduleId)
	return nil
}

func (s *Store) ListRefreshSchedules(datasetId string) []*StoredRefreshSchedule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*StoredRefreshSchedule{}
	for _, r := range s.refreshSchedules[datasetId] {
		out = append(out, r)
	}
	return out
}

// ── VPC Connection ───────────────────────────────────────────────────────────

func (s *Store) CreateVPCConnection(id, name, vpcId string, securityGroupIds, dnsResolvers []string,
	roleArn string) (*StoredVPCConnection, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("VPCConnectionId is required")
	}
	if _, ok := s.vpcConnections[id]; ok {
		return nil, errExists("VPCConnection", id)
	}
	now := nowUTC()
	v := &StoredVPCConnection{
		VPCConnectionId:    id,
		AwsAccountId:       s.accountID,
		Arn:                s.arnVPCConnection(id),
		Name:               name,
		VpcId:              vpcId,
		SecurityGroupIds:   securityGroupIds,
		DnsResolvers:       dnsResolvers,
		Status:             "AVAILABLE",
		AvailabilityStatus: "AVAILABLE",
		RoleArn:            roleArn,
		CreatedTime:        now,
		LastUpdatedTime:    now,
		NetworkInterfaces:  []map[string]any{},
	}
	s.vpcConnections[id] = v
	return v, nil
}

func (s *Store) GetVPCConnection(id string) (*StoredVPCConnection, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.vpcConnections[id]
	if !ok {
		return nil, errNotFound("VPCConnection", id)
	}
	return v, nil
}

func (s *Store) UpdateVPCConnection(id, name string, securityGroupIds, dnsResolvers []string,
	roleArn string) (*StoredVPCConnection, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.vpcConnections[id]
	if !ok {
		return nil, errNotFound("VPCConnection", id)
	}
	if name != "" {
		v.Name = name
	}
	if securityGroupIds != nil {
		v.SecurityGroupIds = securityGroupIds
	}
	if dnsResolvers != nil {
		v.DnsResolvers = dnsResolvers
	}
	if roleArn != "" {
		v.RoleArn = roleArn
	}
	v.LastUpdatedTime = nowUTC()
	return v, nil
}

func (s *Store) DeleteVPCConnection(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.vpcConnections[id]
	if !ok {
		return errNotFound("VPCConnection", id)
	}
	v.Status = "DELETED"
	delete(s.vpcConnections, id)
	return nil
}

func (s *Store) ListVPCConnections() []*StoredVPCConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredVPCConnection, 0, len(s.vpcConnections))
	for _, v := range s.vpcConnections {
		out = append(out, v)
	}
	return out
}

// ── Custom permissions ───────────────────────────────────────────────────────

func (s *Store) CreateCustomPermissions(name string, capabilities map[string]any) (*StoredCustomPermissions, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, errInvalid("CustomPermissionsName is required")
	}
	if _, ok := s.customPermissions[name]; ok {
		return nil, errExists("CustomPermissions", name)
	}
	cp := &StoredCustomPermissions{
		CustomPermissionsName: name,
		AwsAccountId:          s.accountID,
		Arn:                   s.arnCustomPermissions(name),
		Capabilities:          capabilities,
	}
	s.customPermissions[name] = cp
	return cp, nil
}

func (s *Store) GetCustomPermissions(name string) (*StoredCustomPermissions, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp, ok := s.customPermissions[name]
	if !ok {
		return nil, errNotFound("CustomPermissions", name)
	}
	return cp, nil
}

func (s *Store) UpdateCustomPermissions(name string, capabilities map[string]any) (*StoredCustomPermissions, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.customPermissions[name]
	if !ok {
		return nil, errNotFound("CustomPermissions", name)
	}
	if capabilities != nil {
		cp.Capabilities = capabilities
	}
	return cp, nil
}

func (s *Store) DeleteCustomPermissions(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.customPermissions[name]; !ok {
		return errNotFound("CustomPermissions", name)
	}
	delete(s.customPermissions, name)
	return nil
}

func (s *Store) ListCustomPermissions() []*StoredCustomPermissions {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCustomPermissions, 0, len(s.customPermissions))
	for _, cp := range s.customPermissions {
		out = append(out, cp)
	}
	return out
}

// ── Brand ────────────────────────────────────────────────────────────────────

func (s *Store) CreateBrand(id, name string, brandDetail, brandColorPalette, brandElementStyle map[string]any) (*StoredBrand, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("BrandId is required")
	}
	if _, ok := s.brands[id]; ok {
		return nil, errExists("Brand", id)
	}
	now := nowUTC()
	b := &StoredBrand{
		BrandId:           id,
		Arn:               s.arnBrand(id),
		BrandName:         name,
		Status:            "CREATE_SUCCEEDED",
		CreatedTime:       now,
		LastUpdatedTime:   now,
		BrandDetail:       brandDetail,
		BrandColorPalette: brandColorPalette,
		BrandElementStyle: brandElementStyle,
		Versions:          []map[string]any{{"VersionId": "1", "Status": "CREATE_SUCCEEDED"}},
		PublishedVersionId: "1",
	}
	s.brands[id] = b
	return b, nil
}

func (s *Store) GetBrand(id string) (*StoredBrand, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.brands[id]
	if !ok {
		return nil, errNotFound("Brand", id)
	}
	return b, nil
}

func (s *Store) UpdateBrand(id, name string, brandDetail, brandColorPalette, brandElementStyle map[string]any) (*StoredBrand, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.brands[id]
	if !ok {
		return nil, errNotFound("Brand", id)
	}
	if name != "" {
		b.BrandName = name
	}
	if brandDetail != nil {
		b.BrandDetail = brandDetail
	}
	if brandColorPalette != nil {
		b.BrandColorPalette = brandColorPalette
	}
	if brandElementStyle != nil {
		b.BrandElementStyle = brandElementStyle
	}
	b.LastUpdatedTime = nowUTC()
	return b, nil
}

func (s *Store) UpdateBrandPublishedVersion(id, versionId string) (*StoredBrand, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.brands[id]
	if !ok {
		return nil, errNotFound("Brand", id)
	}
	b.PublishedVersionId = versionId
	return b, nil
}

func (s *Store) DeleteBrand(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.brands[id]; !ok {
		return errNotFound("Brand", id)
	}
	delete(s.brands, id)
	return nil
}

func (s *Store) ListBrands() []*StoredBrand {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredBrand, 0, len(s.brands))
	for _, b := range s.brands {
		out = append(out, b)
	}
	return out
}

// ── Asset bundle export job ──────────────────────────────────────────────────

func (s *Store) StartAssetBundleExportJob(id string, resourceArns []string, exportFormat string,
	cfgOverride map[string]any, includePerms, includeTags bool) (*StoredAssetBundleExportJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("AssetBundleExportJobId is required")
	}
	if _, ok := s.assetBundleExportJobs[id]; ok {
		return nil, errExists("AssetBundleExportJob", id)
	}
	j := &StoredAssetBundleExportJob{
		JobId:        id,
		Arn:          s.arnAssetBundleExportJob(id),
		Status:       "QUEUED_FOR_IMMEDIATE_EXECUTION",
		ResourceArns: resourceArns,
		ExportFormat: exportFormat,
		CloudFormationOverridePropertyConfiguration: cfgOverride,
		IncludePermissions: includePerms,
		IncludeTags:        includeTags,
		CreatedTime:        nowUTC(),
		AwsAccountId:       s.accountID,
	}
	s.assetBundleExportJobs[id] = j
	return j, nil
}

func (s *Store) DescribeAssetBundleExportJob(id string) (*StoredAssetBundleExportJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.assetBundleExportJobs[id]
	if !ok {
		return nil, errNotFound("AssetBundleExportJob", id)
	}
	j.describeCallCount++
	if j.describeCallCount >= 1 && j.Status != "SUCCESSFUL" {
		j.Status = "SUCCESSFUL"
		j.DownloadUrl = fmt.Sprintf("https://cloudmock.test/asset-bundle/exports/%s.qs", id)
	}
	return j, nil
}

func (s *Store) ListAssetBundleExportJobs() []*StoredAssetBundleExportJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAssetBundleExportJob, 0, len(s.assetBundleExportJobs))
	for _, j := range s.assetBundleExportJobs {
		out = append(out, j)
	}
	return out
}

// ── Asset bundle import job ──────────────────────────────────────────────────

func (s *Store) StartAssetBundleImportJob(id string, source map[string]any, overrideParams, overridePerms,
	overrideTags, overrideValStrategy map[string]any, failureAction string) (*StoredAssetBundleImportJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("AssetBundleImportJobId is required")
	}
	if _, ok := s.assetBundleImportJobs[id]; ok {
		return nil, errExists("AssetBundleImportJob", id)
	}
	j := &StoredAssetBundleImportJob{
		JobId:                      id,
		Arn:                        s.arnAssetBundleImportJob(id),
		Status:                     "QUEUED_FOR_IMMEDIATE_EXECUTION",
		AssetBundleImportSource:    source,
		OverrideParameters:         overrideParams,
		OverridePermissions:        overridePerms,
		OverrideTags:               overrideTags,
		OverrideValidationStrategy: overrideValStrategy,
		FailureAction:              failureAction,
		CreatedTime:                nowUTC(),
		AwsAccountId:               s.accountID,
	}
	s.assetBundleImportJobs[id] = j
	return j, nil
}

func (s *Store) DescribeAssetBundleImportJob(id string) (*StoredAssetBundleImportJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.assetBundleImportJobs[id]
	if !ok {
		return nil, errNotFound("AssetBundleImportJob", id)
	}
	j.describeCallCount++
	if j.describeCallCount >= 1 && j.Status != "SUCCESSFUL" {
		j.Status = "SUCCESSFUL"
	}
	return j, nil
}

func (s *Store) ListAssetBundleImportJobs() []*StoredAssetBundleImportJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAssetBundleImportJob, 0, len(s.assetBundleImportJobs))
	for _, j := range s.assetBundleImportJobs {
		out = append(out, j)
	}
	return out
}

// ── Dashboard snapshot job ───────────────────────────────────────────────────

func (s *Store) StartDashboardSnapshotJob(jobId, dashboardId string, userCfg, snapshotCfg map[string]any) (*StoredDashboardSnapshotJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if jobId == "" || dashboardId == "" {
		return nil, errInvalid("SnapshotJobId and DashboardId are required")
	}
	if _, ok := s.dashboards[dashboardId]; !ok {
		return nil, errNotFound("Dashboard", dashboardId)
	}
	if _, ok := s.dashboardSnapshotJobs[jobId]; ok {
		return nil, errExists("DashboardSnapshotJob", jobId)
	}
	j := &StoredDashboardSnapshotJob{
		JobId:                 jobId,
		Arn:                   s.arnDashboardSnapshotJob(jobId, dashboardId),
		DashboardId:           dashboardId,
		AwsAccountId:          s.accountID,
		Status:                "QUEUED",
		JobStatus:             "QUEUED",
		UserConfiguration:     userCfg,
		SnapshotConfiguration: snapshotCfg,
		CreatedTime:           nowUTC(),
	}
	s.dashboardSnapshotJobs[jobId] = j
	return j, nil
}

func (s *Store) GetDashboardSnapshotJob(jobId string) (*StoredDashboardSnapshotJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.dashboardSnapshotJobs[jobId]
	if !ok {
		return nil, errNotFound("DashboardSnapshotJob", jobId)
	}
	if j.Status == "QUEUED" {
		j.Status = "RUNNING"
		j.JobStatus = "RUNNING"
	} else if j.Status == "RUNNING" {
		j.Status = "COMPLETED"
		j.JobStatus = "COMPLETED"
		j.Result = map[string]any{
			"AnonymousUserSnapshotJobResult": []map[string]any{
				{
					"FileGroups": []map[string]any{
						{
							"Files": []map[string]any{
								{
									"FormatType": "CSV",
									"S3DestinationConfiguration": map[string]any{
										"BucketConfiguration": map[string]any{
											"BucketName":   "cloudmock-snapshots",
											"BucketPrefix": "snapshots/" + j.JobId,
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}
	return j, nil
}

// ── Action Connector ─────────────────────────────────────────────────────────

func (s *Store) CreateActionConnector(id, name, connectorType, description string, authCfg map[string]any,
	enabledActions []string, vpcConnectionArn string) (*StoredActionConnector, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return nil, errInvalid("ActionConnectorId is required")
	}
	if _, ok := s.actionConnectors[id]; ok {
		return nil, errExists("ActionConnector", id)
	}
	now := nowUTC()
	a := &StoredActionConnector{
		ActionConnectorId:    id,
		Arn:                  s.arnActionConnector(id),
		Name:                 name,
		Type:                 connectorType,
		Description:          description,
		Status:               "ACTIVE",
		AuthenticationConfig: authCfg,
		EnabledActions:       enabledActions,
		VpcConnectionArn:     vpcConnectionArn,
		CreatedTime:          now,
		LastUpdatedTime:      now,
	}
	s.actionConnectors[id] = a
	return a, nil
}

func (s *Store) GetActionConnector(id string) (*StoredActionConnector, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.actionConnectors[id]
	if !ok {
		return nil, errNotFound("ActionConnector", id)
	}
	return a, nil
}

func (s *Store) UpdateActionConnector(id, name, description string, authCfg map[string]any,
	enabledActions []string) (*StoredActionConnector, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.actionConnectors[id]
	if !ok {
		return nil, errNotFound("ActionConnector", id)
	}
	if name != "" {
		a.Name = name
	}
	if description != "" {
		a.Description = description
	}
	if authCfg != nil {
		a.AuthenticationConfig = authCfg
	}
	if enabledActions != nil {
		a.EnabledActions = enabledActions
	}
	a.LastUpdatedTime = nowUTC()
	return a, nil
}

func (s *Store) DeleteActionConnector(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.actionConnectors[id]; !ok {
		return errNotFound("ActionConnector", id)
	}
	delete(s.actionConnectors, id)
	return nil
}

func (s *Store) ListActionConnectors() []*StoredActionConnector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredActionConnector, 0, len(s.actionConnectors))
	for _, a := range s.actionConnectors {
		out = append(out, a)
	}
	return out
}

// ── Permissions ──────────────────────────────────────────────────────────────

func (s *Store) UpdatePermissions(store map[string][]map[string]any, arn string,
	grant, revoke []map[string]any) []map[string]any {
	current := store[arn]
	// Apply revokes first.
	for _, r := range revoke {
		principal, _ := r["Principal"].(string)
		current = removePrincipal(current, principal)
	}
	// Apply grants.
	for _, g := range grant {
		principal, _ := g["Principal"].(string)
		current = removePrincipal(current, principal)
		current = append(current, g)
	}
	store[arn] = current
	return current
}

func removePrincipal(perms []map[string]any, principal string) []map[string]any {
	keep := []map[string]any{}
	for _, p := range perms {
		if pp, _ := p["Principal"].(string); pp != principal {
			keep = append(keep, p)
		}
	}
	return keep
}

func (s *Store) GetPermissions(store map[string][]map[string]any, arn string) []map[string]any {
	if perms, ok := store[arn]; ok {
		return perms
	}
	return []map[string]any{}
}

// UpdatePermissionsHelper is a thread-safe wrapper.
func (s *Store) UpdateDataSourcePermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.dataSourcePermissions, arn, grant, revoke)
}

func (s *Store) DescribeDataSourcePermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.dataSourcePermissions, arn)
}

func (s *Store) UpdateDataSetPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.dataSetPermissions, arn, grant, revoke)
}

func (s *Store) DescribeDataSetPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.dataSetPermissions, arn)
}

func (s *Store) UpdateTemplatePermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.templatePermissions, arn, grant, revoke)
}

func (s *Store) DescribeTemplatePermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.templatePermissions, arn)
}

func (s *Store) UpdateDashboardPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.dashboardPermissions, arn, grant, revoke)
}

func (s *Store) DescribeDashboardPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.dashboardPermissions, arn)
}

func (s *Store) UpdateAnalysisPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.analysisPermissions, arn, grant, revoke)
}

func (s *Store) DescribeAnalysisPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.analysisPermissions, arn)
}

func (s *Store) UpdateThemePermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.themePermissions, arn, grant, revoke)
}

func (s *Store) DescribeThemePermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.themePermissions, arn)
}

func (s *Store) UpdateFolderPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.folderPermissions, arn, grant, revoke)
}

func (s *Store) DescribeFolderPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.folderPermissions, arn)
}

func (s *Store) UpdateTopicPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.topicPermissions, arn, grant, revoke)
}

func (s *Store) DescribeTopicPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.topicPermissions, arn)
}

func (s *Store) UpdateActionConnectorPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.actionConnectorPermissions, arn, grant, revoke)
}

func (s *Store) DescribeActionConnectorPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.actionConnectorPermissions, arn)
}

func (s *Store) UpdateFlowPermissions(arn string, grant, revoke []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.UpdatePermissions(s.flowPermissions, arn, grant, revoke)
}

func (s *Store) GetFlowPermissions(arn string) []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.GetPermissions(s.flowPermissions, arn)
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(existing, k)
		}
	}
}

func (s *Store) ListTagsForResource(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if existing, ok := s.tags[arn]; ok {
		out := make(map[string]string, len(existing))
		for k, v := range existing {
			out[k] = v
		}
		return out
	}
	return map[string]string{}
}

// ── Account settings & customizations ────────────────────────────────────────

func (s *Store) GetAccountSettings() *StoredAccountSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountSettings
}

func (s *Store) UpdateAccountSettings(defaultNamespace, notificationEmail string,
	terminationProtection *bool) *StoredAccountSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	if defaultNamespace != "" {
		s.accountSettings.DefaultNamespace = defaultNamespace
	}
	if notificationEmail != "" {
		s.accountSettings.NotificationEmail = notificationEmail
	}
	if terminationProtection != nil {
		s.accountSettings.TerminationProtectionEnabled = *terminationProtection
	}
	return s.accountSettings
}

func (s *Store) CreateAccountSubscription(edition, authType, accountName, notificationEmail string) *StoredAccountSubscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	if edition != "" {
		s.accountSubscription.Edition = edition
	}
	if authType != "" {
		s.accountSubscription.AuthenticationType = authType
	}
	if accountName != "" {
		s.accountSettings.AccountName = accountName
	}
	if notificationEmail != "" {
		s.accountSettings.NotificationEmail = notificationEmail
	}
	s.accountSubscription.AccountSubscriptionStatus = "ACCOUNT_CREATED"
	return s.accountSubscription
}

func (s *Store) GetAccountSubscription() *StoredAccountSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountSubscription
}

func (s *Store) DeleteAccountSubscription() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountSubscription.AccountSubscriptionStatus = "TERMINATED"
}

func (s *Store) PutAccountCustomization(namespace string, customization map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	s.accountCustomizations[namespace] = customization
}

func (s *Store) GetAccountCustomization(namespace string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	if c, ok := s.accountCustomizations[namespace]; ok {
		return c
	}
	return nil
}

func (s *Store) DeleteAccountCustomization(namespace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if namespace == "" {
		namespace = "default"
	}
	delete(s.accountCustomizations, namespace)
}

func (s *Store) UpdatePublicSharingSettings(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publicSharingEnabled = enabled
	s.accountSettings.PublicSharingEnabled = enabled
}

// ── IP restriction & key registration ────────────────────────────────────────

func (s *Store) UpdateIpRestriction(restriction map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range restriction {
		s.ipRestriction[k] = v
	}
	return s.ipRestriction
}

func (s *Store) GetIpRestriction() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ipRestriction
}

func (s *Store) UpdateKeyRegistration(keys []map[string]any) []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyRegistration = keys
	return s.keyRegistration
}

func (s *Store) GetKeyRegistration() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.keyRegistration == nil {
		return []map[string]any{}
	}
	return s.keyRegistration
}

// ── Q personalization & search config ────────────────────────────────────────

func (s *Store) UpdateQPersonalizationConfiguration(status string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status != "" {
		s.qPersonalizationCfg = status
	}
	return s.qPersonalizationCfg
}

func (s *Store) GetQPersonalizationConfiguration() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.qPersonalizationCfg
}

func (s *Store) UpdateQuickSightQSearchConfiguration(status string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status != "" {
		s.qSearchCfg = status
	}
	return s.qSearchCfg
}

func (s *Store) GetQuickSightQSearchConfiguration() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.qSearchCfg
}

func (s *Store) UpdateDashboardsQAConfiguration(status string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status != "" {
		s.dashboardsQAConfig = status
	}
	return s.dashboardsQAConfig
}

func (s *Store) GetDashboardsQAConfiguration() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dashboardsQAConfig
}

// ── Default Q business app ───────────────────────────────────────────────────

func (s *Store) SetDefaultQBusinessApplication(app map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultQBusinessApp = app
}

func (s *Store) GetDefaultQBusinessApplication() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultQBusinessApp
}

func (s *Store) DeleteDefaultQBusinessApplication() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultQBusinessApp = nil
}

// ── Identity propagation ─────────────────────────────────────────────────────

func (s *Store) PutIdentityPropagationConfig(svcName string, config map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identityPropagationConfigs[svcName] = config
}

func (s *Store) DeleteIdentityPropagationConfig(svcName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.identityPropagationConfigs, svcName)
}

func (s *Store) ListIdentityPropagationConfigs() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []map[string]any{}
	for svcName, cfg := range s.identityPropagationConfigs {
		entry := map[string]any{"Service": svcName}
		for k, v := range cfg {
			entry[k] = v
		}
		out = append(out, entry)
	}
	return out
}

// ── Account custom permission, role custom permission, role membership ──────

func (s *Store) PutAccountCustomPermission(arn, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountCustomPermission[arn] = name
}

func (s *Store) GetAccountCustomPermission(arn string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accountCustomPermission[arn]
}

func (s *Store) DeleteAccountCustomPermission(arn string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.accountCustomPermission, arn)
}

func (s *Store) PutRoleCustomPermission(namespace, role, permissionsName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roleCustomPermissions[namespace] == nil {
		s.roleCustomPermissions[namespace] = make(map[string]string)
	}
	s.roleCustomPermissions[namespace][role] = permissionsName
}

func (s *Store) GetRoleCustomPermission(namespace, role string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ns, ok := s.roleCustomPermissions[namespace]; ok {
		return ns[role]
	}
	return ""
}

func (s *Store) DeleteRoleCustomPermission(namespace, role string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ns, ok := s.roleCustomPermissions[namespace]; ok {
		delete(ns, role)
	}
}

func (s *Store) AddRoleMembership(namespace, role, memberName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roleMemberships[namespace] == nil {
		s.roleMemberships[namespace] = make(map[string]map[string]bool)
	}
	if s.roleMemberships[namespace][role] == nil {
		s.roleMemberships[namespace][role] = make(map[string]bool)
	}
	s.roleMemberships[namespace][role][memberName] = true
}

func (s *Store) RemoveRoleMembership(namespace, role, memberName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ns, ok := s.roleMemberships[namespace]; ok {
		if r, ok := ns[role]; ok {
			delete(r, memberName)
		}
	}
}

func (s *Store) ListRoleMemberships(namespace, role string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []string{}
	if ns, ok := s.roleMemberships[namespace]; ok {
		if r, ok := ns[role]; ok {
			for n := range r {
				out = append(out, n)
			}
		}
	}
	return out
}

// ── Brand assignment ─────────────────────────────────────────────────────────

func (s *Store) UpdateBrandAssignment(brandArn string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.brandAssignment = brandArn
	return s.brandAssignment
}

func (s *Store) GetBrandAssignment() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brandAssignment
}

func (s *Store) DeleteBrandAssignment() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.brandAssignment = ""
}

// ── Self upgrade ─────────────────────────────────────────────────────────────

func (s *Store) PutSelfUpgradeConfiguration(cfg map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selfUpgradeConfig = cfg
}

func (s *Store) GetSelfUpgradeConfiguration() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.selfUpgradeConfig == nil {
		return map[string]any{"Status": "DISABLED"}
	}
	return s.selfUpgradeConfig
}

// ── Folder resource lookup ───────────────────────────────────────────────────

func (s *Store) AddResourceToFolder(folderId, resourceArn string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.folderResources[folderId] = append(s.folderResources[folderId], resourceArn)
}

func (s *Store) RemoveResourceFromFolder(folderId, resourceArn string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arns := s.folderResources[folderId]
	keep := []string{}
	for _, a := range arns {
		if a != resourceArn {
			keep = append(keep, a)
		}
	}
	s.folderResources[folderId] = keep
}

func (s *Store) ListFoldersForResource(resourceArn string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []string{}
	for folderId, arns := range s.folderResources {
		for _, a := range arns {
			if a == resourceArn {
				out = append(out, folderId)
				break
			}
		}
	}
	return out
}

// ── Flow ─────────────────────────────────────────────────────────────────────

func (s *Store) CreateFlow(id, name, description string) *StoredFlow {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := nowUTC()
	f := &StoredFlow{
		FlowId:          id,
		Arn:             s.arnFlow(id),
		Name:            name,
		Description:     description,
		CreatedTime:     now,
		LastUpdatedTime: now,
	}
	s.flows[id] = f
	return f
}

func (s *Store) GetFlow(id string) (*StoredFlow, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flows[id]
	if !ok {
		return nil, errNotFound("Flow", id)
	}
	return f, nil
}

func (s *Store) ListFlows() []*StoredFlow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFlow, 0, len(s.flows))
	for _, f := range s.flows {
		out = append(out, f)
	}
	return out
}

// ── Automation jobs ──────────────────────────────────────────────────────────

func (s *Store) StartAutomationJob(jobId string, cfg map[string]any) *StoredAutomationJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := &StoredAutomationJob{
		JobId:         jobId,
		Status:        "QUEUED",
		CreatedTime:   nowUTC(),
		Configuration: cfg,
	}
	s.automationJobs[jobId] = j
	return j
}

func (s *Store) GetAutomationJob(jobId string) (*StoredAutomationJob, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.automationJobs[jobId]
	if !ok {
		return nil, errNotFound("AutomationJob", jobId)
	}
	if j.Status == "QUEUED" {
		j.Status = "RUNNING"
	} else if j.Status == "RUNNING" {
		j.Status = "COMPLETED"
	}
	return j, nil
}

// ── Self upgrade list ────────────────────────────────────────────────────────

func (s *Store) ListSelfUpgrades() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return []map[string]any{}
}
