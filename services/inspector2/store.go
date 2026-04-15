package inspector2

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ─────────────────────────────────────────────────────────────

// StoredFilter represents a finding filter.
type StoredFilter struct {
	Arn         string
	Name        string
	Description string
	Action      string
	Reason      string
	Criteria    map[string]any
	OwnerId     string
	Tags        map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StoredFinding represents an Inspector2 finding.
type StoredFinding struct {
	FindingArn      string
	AwsAccountId    string
	Description     string
	Title           string
	FindingType     string
	Severity        string
	Status          string
	InspectorScore  float64
	ExploitAvailable string
	FixAvailable    string
	EpssScore       float64
	Resources       []map[string]any
	Remediation     map[string]any
	FirstObservedAt time.Time
	LastObservedAt  time.Time
	UpdatedAt       time.Time
}

// StoredCisScanConfiguration represents a CIS scan configuration.
type StoredCisScanConfiguration struct {
	ScanConfigurationArn string
	ScanName             string
	SecurityLevel        string
	OwnerId              string
	Schedule             map[string]any
	Targets              map[string]any
	Tags                 map[string]string
	CreatedAt            time.Time
}

// StoredCisScan represents a CIS scan.
type StoredCisScan struct {
	ScanArn              string
	ScanConfigurationArn string
	ScanName             string
	Status               string
	StatusMessage        string
	SecurityLevel        string
	ScheduledBy          string
	FailedChecks         int
	TotalChecks          int
	Targets              map[string]any
	ScanDate             time.Time
}

// StoredCisSession represents an in-progress CIS session.
type StoredCisSession struct {
	SessionToken string
	ScanJobId    string
	StartDate    time.Time
}

// StoredCodeSecurityIntegration represents a code security integration.
type StoredCodeSecurityIntegration struct {
	IntegrationArn string
	Name           string
	Type           string
	Status         string
	StatusReason   string
	Tags           map[string]string
	CreatedOn      time.Time
	LastUpdateOn   time.Time
}

// StoredCodeSecurityScanConfiguration represents a code security scan config.
type StoredCodeSecurityScanConfiguration struct {
	ScanConfigurationArn string
	Name                 string
	Level                string
	Configuration        map[string]any
	ScopeSettings        map[string]any
	Tags                 map[string]string
	OwnerAccountId       string
	CreatedAt            time.Time
	LastUpdatedAt        time.Time
}

// StoredCodeSecurityScan represents a code security scan.
type StoredCodeSecurityScan struct {
	ScanId       string
	AccountId    string
	Resource     map[string]any
	Status       string
	StatusReason string
	LastCommitId string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// StoredMember represents an Inspector2 member account.
type StoredMember struct {
	AccountId               string
	DelegatedAdminAccountId string
	RelationshipStatus      string
	UpdatedAt               time.Time
}

// StoredFindingsReport represents an async findings report job.
type StoredFindingsReport struct {
	ReportId       string
	Status         string
	ReportFormat   string
	Destination    map[string]any
	FilterCriteria map[string]any
	ErrorCode      string
	ErrorMessage   string
	StartedAt      time.Time
	CompletedAt    time.Time
}

// StoredSbomExport represents an async SBOM export job.
type StoredSbomExport struct {
	ReportId       string
	Status         string
	Format         string
	Destination    map[string]any
	FilterCriteria map[string]any
	ErrorCode      string
	ErrorMessage   string
	StartedAt      time.Time
	CompletedAt    time.Time
}

// StoredEc2DeepInspection represents an account's EC2 deep inspection state.
type StoredEc2DeepInspection struct {
	AccountId       string
	PackagePaths    []string
	OrgPackagePaths []string
	Status          string
	ErrorMessage    string
}

// StoredAccountStatus represents per-account scan-type status.
type StoredAccountStatus struct {
	AccountId      string
	Ec2            string
	Ecr            string
	Lambda         string
	LambdaCode     string
	CodeRepository string
	Status         string // overall status
}

// StoredOrganizationConfig represents the org-wide auto-enable settings.
type StoredOrganizationConfig struct {
	Ec2            bool
	Ecr            bool
	Lambda         bool
	LambdaCode     bool
	CodeRepository bool
}

// StoredEncryptionKey holds the kms key for a (scanType, resourceType) pair.
type StoredEncryptionKey struct {
	ResourceType string
	ScanType     string
	KmsKeyId     string
}

// Store is the in-memory data store for Inspector2 resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	filters                 map[string]*StoredFilter // arn -> filter
	findings                map[string]*StoredFinding
	cisScanConfigs          map[string]*StoredCisScanConfiguration // arn -> config
	cisScans                map[string]*StoredCisScan              // scan arn -> scan
	cisSessions             map[string]*StoredCisSession           // sessionToken -> session
	codeSecurityIntegrations map[string]*StoredCodeSecurityIntegration
	codeSecurityScanConfigs  map[string]*StoredCodeSecurityScanConfiguration
	codeSecurityScans        map[string]*StoredCodeSecurityScan
	codeSecurityAssociations map[string]map[string]bool // scanConfigArn -> projectId -> true
	members                  map[string]*StoredMember
	delegatedAdminAccounts   map[string]string // accountId -> status
	findingsReports          map[string]*StoredFindingsReport
	sbomExports              map[string]*StoredSbomExport
	ec2DeepInspections       map[string]*StoredEc2DeepInspection
	orgPackagePaths          []string
	encryptionKeys           map[string]*StoredEncryptionKey // "scanType:resourceType" -> key
	accountStatuses          map[string]*StoredAccountStatus
	orgConfig                *StoredOrganizationConfig
	resourceTags             map[string]map[string]string
	ec2Configuration         map[string]any
	ecrConfiguration         map[string]any
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:                accountID,
		region:                   region,
		filters:                  make(map[string]*StoredFilter),
		findings:                 make(map[string]*StoredFinding),
		cisScanConfigs:           make(map[string]*StoredCisScanConfiguration),
		cisScans:                 make(map[string]*StoredCisScan),
		cisSessions:              make(map[string]*StoredCisSession),
		codeSecurityIntegrations: make(map[string]*StoredCodeSecurityIntegration),
		codeSecurityScanConfigs:  make(map[string]*StoredCodeSecurityScanConfiguration),
		codeSecurityScans:        make(map[string]*StoredCodeSecurityScan),
		codeSecurityAssociations: make(map[string]map[string]bool),
		members:                  make(map[string]*StoredMember),
		delegatedAdminAccounts:   make(map[string]string),
		findingsReports:          make(map[string]*StoredFindingsReport),
		sbomExports:              make(map[string]*StoredSbomExport),
		ec2DeepInspections:       make(map[string]*StoredEc2DeepInspection),
		encryptionKeys:           make(map[string]*StoredEncryptionKey),
		accountStatuses:          make(map[string]*StoredAccountStatus),
		orgConfig:                &StoredOrganizationConfig{},
		resourceTags:             make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filters = make(map[string]*StoredFilter)
	s.findings = make(map[string]*StoredFinding)
	s.cisScanConfigs = make(map[string]*StoredCisScanConfiguration)
	s.cisScans = make(map[string]*StoredCisScan)
	s.cisSessions = make(map[string]*StoredCisSession)
	s.codeSecurityIntegrations = make(map[string]*StoredCodeSecurityIntegration)
	s.codeSecurityScanConfigs = make(map[string]*StoredCodeSecurityScanConfiguration)
	s.codeSecurityScans = make(map[string]*StoredCodeSecurityScan)
	s.codeSecurityAssociations = make(map[string]map[string]bool)
	s.members = make(map[string]*StoredMember)
	s.delegatedAdminAccounts = make(map[string]string)
	s.findingsReports = make(map[string]*StoredFindingsReport)
	s.sbomExports = make(map[string]*StoredSbomExport)
	s.ec2DeepInspections = make(map[string]*StoredEc2DeepInspection)
	s.orgPackagePaths = nil
	s.encryptionKeys = make(map[string]*StoredEncryptionKey)
	s.accountStatuses = make(map[string]*StoredAccountStatus)
	s.orgConfig = &StoredOrganizationConfig{}
	s.resourceTags = make(map[string]map[string]string)
	s.ec2Configuration = nil
	s.ecrConfiguration = nil
}

// AccountID returns the configured account.
func (s *Store) AccountID() string { return s.accountID }

// Region returns the configured region.
func (s *Store) Region() string { return s.region }

// ── Filters ──────────────────────────────────────────────────────────────────

// CreateFilter persists a new filter and returns it.
func (s *Store) CreateFilter(name, description, action, reason string, criteria map[string]any, tags map[string]string) (*StoredFilter, *service.AWSError) {
	if name == "" {
		return nil, service.ErrValidation("name is required")
	}
	if action == "" {
		return nil, service.ErrValidation("action is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := s.buildFilterArn(name)
	if _, ok := s.filters[arn]; ok {
		return nil, service.NewAWSError("ConflictException", "Filter already exists: "+name, 409)
	}
	now := time.Now().UTC()
	f := &StoredFilter{
		Arn:         arn,
		Name:        name,
		Description: description,
		Action:      action,
		Reason:      reason,
		Criteria:    criteria,
		OwnerId:     s.accountID,
		Tags:        copyStrings(tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.filters[arn] = f
	return f, nil
}

// GetFilter returns a filter by ARN.
func (s *Store) GetFilter(arn string) (*StoredFilter, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.filters[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Filter not found: "+arn, 404)
	}
	return f, nil
}

// DeleteFilter removes a filter.
func (s *Store) DeleteFilter(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.filters[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "Filter not found: "+arn, 404)
	}
	delete(s.filters, arn)
	return nil
}

// ListFilters returns all filters, optionally filtered by action and arns.
func (s *Store) ListFilters(action string, arns []string) []*StoredFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wanted := make(map[string]bool, len(arns))
	for _, a := range arns {
		wanted[a] = true
	}
	out := make([]*StoredFilter, 0, len(s.filters))
	for _, f := range s.filters {
		if action != "" && f.Action != action {
			continue
		}
		if len(wanted) > 0 && !wanted[f.Arn] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// UpdateFilter updates an existing filter.
func (s *Store) UpdateFilter(arn string, name, description, action, reason *string, criteria map[string]any) (*StoredFilter, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.filters[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Filter not found: "+arn, 404)
	}
	if name != nil && *name != "" {
		f.Name = *name
	}
	if description != nil {
		f.Description = *description
	}
	if action != nil && *action != "" {
		f.Action = *action
	}
	if reason != nil {
		f.Reason = *reason
	}
	if criteria != nil {
		f.Criteria = criteria
	}
	f.UpdatedAt = time.Now().UTC()
	return f, nil
}

// ── Findings ─────────────────────────────────────────────────────────────────

// PutFinding seeds a finding in the store.
func (s *Store) PutFinding(f *StoredFinding) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.FindingArn == "" {
		f.FindingArn = fmt.Sprintf("arn:aws:inspector2:%s:%s:finding/%s", s.region, s.accountID, generateID())
	}
	s.findings[f.FindingArn] = f
}

// GetFinding returns a finding by ARN.
func (s *Store) GetFinding(arn string) (*StoredFinding, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.findings[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Finding not found: "+arn, 404)
	}
	return f, nil
}

// ListFindings returns findings.
func (s *Store) ListFindings() []*StoredFinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFinding, 0, len(s.findings))
	for _, f := range s.findings {
		out = append(out, f)
	}
	return out
}

// ── CIS scan configurations ──────────────────────────────────────────────────

// CreateCisScanConfiguration persists a new CIS scan configuration.
func (s *Store) CreateCisScanConfiguration(name, securityLevel string, schedule, targets map[string]any, tags map[string]string) (*StoredCisScanConfiguration, *service.AWSError) {
	if name == "" {
		return nil, service.ErrValidation("scanName is required")
	}
	if securityLevel == "" {
		return nil, service.ErrValidation("securityLevel is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := s.buildCisScanConfigArn(generateID())
	cfg := &StoredCisScanConfiguration{
		ScanConfigurationArn: arn,
		ScanName:             name,
		SecurityLevel:        securityLevel,
		OwnerId:              s.accountID,
		Schedule:             schedule,
		Targets:              targets,
		Tags:                 copyStrings(tags),
		CreatedAt:            time.Now().UTC(),
	}
	s.cisScanConfigs[arn] = cfg
	return cfg, nil
}

// GetCisScanConfiguration returns a CIS scan configuration by ARN.
func (s *Store) GetCisScanConfiguration(arn string) (*StoredCisScanConfiguration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.cisScanConfigs[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "CIS scan configuration not found: "+arn, 404)
	}
	return cfg, nil
}

// DeleteCisScanConfiguration removes a CIS scan configuration.
func (s *Store) DeleteCisScanConfiguration(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cisScanConfigs[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "CIS scan configuration not found: "+arn, 404)
	}
	delete(s.cisScanConfigs, arn)
	return nil
}

// UpdateCisScanConfiguration updates a CIS scan configuration.
func (s *Store) UpdateCisScanConfiguration(arn string, name, securityLevel *string, schedule, targets map[string]any) (*StoredCisScanConfiguration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.cisScanConfigs[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "CIS scan configuration not found: "+arn, 404)
	}
	if name != nil && *name != "" {
		cfg.ScanName = *name
	}
	if securityLevel != nil && *securityLevel != "" {
		cfg.SecurityLevel = *securityLevel
	}
	if schedule != nil {
		cfg.Schedule = schedule
	}
	if targets != nil {
		cfg.Targets = targets
	}
	return cfg, nil
}

// ListCisScanConfigurations returns all CIS scan configurations.
func (s *Store) ListCisScanConfigurations() []*StoredCisScanConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCisScanConfiguration, 0, len(s.cisScanConfigs))
	for _, c := range s.cisScanConfigs {
		out = append(out, c)
	}
	return out
}

// ListCisScans returns all CIS scans.
func (s *Store) ListCisScans() []*StoredCisScan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCisScan, 0, len(s.cisScans))
	for _, c := range s.cisScans {
		out = append(out, c)
	}
	return out
}

// ── CIS sessions ─────────────────────────────────────────────────────────────

// StartCisSession creates a session entry.
func (s *Store) StartCisSession(scanJobId, sessionToken string) *service.AWSError {
	if scanJobId == "" || sessionToken == "" {
		return service.ErrValidation("scanJobId and sessionToken are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cisSessions[sessionToken] = &StoredCisSession{
		SessionToken: sessionToken,
		ScanJobId:    scanJobId,
		StartDate:    time.Now().UTC(),
	}
	return nil
}

// StopCisSession removes a session.
func (s *Store) StopCisSession(sessionToken string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cisSessions[sessionToken]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "CIS session not found: "+sessionToken, 404)
	}
	delete(s.cisSessions, sessionToken)
	return nil
}

// HasCisSession returns true if a session exists.
func (s *Store) HasCisSession(sessionToken string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cisSessions[sessionToken]
	return ok
}

// ── Code security integrations ──────────────────────────────────────────────

// CreateCodeSecurityIntegration creates a code security integration.
func (s *Store) CreateCodeSecurityIntegration(name, integrationType string, tags map[string]string) (*StoredCodeSecurityIntegration, *service.AWSError) {
	if name == "" {
		return nil, service.ErrValidation("name is required")
	}
	if integrationType == "" {
		return nil, service.ErrValidation("type is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateID()
	arn := fmt.Sprintf("arn:aws:inspector2:%s:%s:code-security-integration/%s", s.region, s.accountID, id)
	now := time.Now().UTC()
	integration := &StoredCodeSecurityIntegration{
		IntegrationArn: arn,
		Name:           name,
		Type:           integrationType,
		Status:         "PENDING",
		Tags:           copyStrings(tags),
		CreatedOn:      now,
		LastUpdateOn:   now,
	}
	s.codeSecurityIntegrations[arn] = integration
	return integration, nil
}

// GetCodeSecurityIntegration returns an integration.
func (s *Store) GetCodeSecurityIntegration(arn string) (*StoredCodeSecurityIntegration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i, ok := s.codeSecurityIntegrations[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Integration not found: "+arn, 404)
	}
	return i, nil
}

// DeleteCodeSecurityIntegration removes an integration.
func (s *Store) DeleteCodeSecurityIntegration(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codeSecurityIntegrations[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "Integration not found: "+arn, 404)
	}
	delete(s.codeSecurityIntegrations, arn)
	return nil
}

// ListCodeSecurityIntegrations returns all integrations.
func (s *Store) ListCodeSecurityIntegrations() []*StoredCodeSecurityIntegration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCodeSecurityIntegration, 0, len(s.codeSecurityIntegrations))
	for _, i := range s.codeSecurityIntegrations {
		out = append(out, i)
	}
	return out
}

// UpdateCodeSecurityIntegration updates the status of an integration.
func (s *Store) UpdateCodeSecurityIntegration(arn, status string) (*StoredCodeSecurityIntegration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.codeSecurityIntegrations[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Integration not found: "+arn, 404)
	}
	if status != "" {
		i.Status = status
	} else {
		i.Status = "ACTIVE"
	}
	i.LastUpdateOn = time.Now().UTC()
	return i, nil
}

// ── Code security scan configurations ────────────────────────────────────────

// CreateCodeSecurityScanConfiguration creates a code security scan config.
func (s *Store) CreateCodeSecurityScanConfiguration(name, level string, configuration, scopeSettings map[string]any, tags map[string]string) (*StoredCodeSecurityScanConfiguration, *service.AWSError) {
	if name == "" {
		return nil, service.ErrValidation("name is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := fmt.Sprintf("arn:aws:inspector2:%s:%s:code-security-scan-configuration/%s", s.region, s.accountID, generateID())
	now := time.Now().UTC()
	cfg := &StoredCodeSecurityScanConfiguration{
		ScanConfigurationArn: arn,
		Name:                 name,
		Level:                level,
		Configuration:        configuration,
		ScopeSettings:        scopeSettings,
		Tags:                 copyStrings(tags),
		OwnerAccountId:       s.accountID,
		CreatedAt:            now,
		LastUpdatedAt:        now,
	}
	s.codeSecurityScanConfigs[arn] = cfg
	return cfg, nil
}

// GetCodeSecurityScanConfiguration returns a scan config.
func (s *Store) GetCodeSecurityScanConfiguration(arn string) (*StoredCodeSecurityScanConfiguration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.codeSecurityScanConfigs[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Code security scan configuration not found: "+arn, 404)
	}
	return c, nil
}

// DeleteCodeSecurityScanConfiguration removes a scan config.
func (s *Store) DeleteCodeSecurityScanConfiguration(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codeSecurityScanConfigs[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "Code security scan configuration not found: "+arn, 404)
	}
	delete(s.codeSecurityScanConfigs, arn)
	delete(s.codeSecurityAssociations, arn)
	return nil
}

// UpdateCodeSecurityScanConfiguration updates the configuration.
func (s *Store) UpdateCodeSecurityScanConfiguration(arn string, configuration map[string]any) (*StoredCodeSecurityScanConfiguration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codeSecurityScanConfigs[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Code security scan configuration not found: "+arn, 404)
	}
	if configuration != nil {
		c.Configuration = configuration
	}
	c.LastUpdatedAt = time.Now().UTC()
	return c, nil
}

// ListCodeSecurityScanConfigurations returns all configs.
func (s *Store) ListCodeSecurityScanConfigurations() []*StoredCodeSecurityScanConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredCodeSecurityScanConfiguration, 0, len(s.codeSecurityScanConfigs))
	for _, c := range s.codeSecurityScanConfigs {
		out = append(out, c)
	}
	return out
}

// AssociateCodeSecurityScanConfiguration adds an association.
func (s *Store) AssociateCodeSecurityScanConfiguration(scanConfigArn, projectId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codeSecurityScanConfigs[scanConfigArn]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "Code security scan configuration not found: "+scanConfigArn, 404)
	}
	if s.codeSecurityAssociations[scanConfigArn] == nil {
		s.codeSecurityAssociations[scanConfigArn] = make(map[string]bool)
	}
	s.codeSecurityAssociations[scanConfigArn][projectId] = true
	return nil
}

// DisassociateCodeSecurityScanConfiguration removes an association.
func (s *Store) DisassociateCodeSecurityScanConfiguration(scanConfigArn, projectId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codeSecurityScanConfigs[scanConfigArn]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "Code security scan configuration not found: "+scanConfigArn, 404)
	}
	if m, ok := s.codeSecurityAssociations[scanConfigArn]; ok {
		delete(m, projectId)
	}
	return nil
}

// ListCodeSecurityScanConfigurationAssociations returns the project ids associated with a config.
func (s *Store) ListCodeSecurityScanConfigurationAssociations(scanConfigArn string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0)
	if m, ok := s.codeSecurityAssociations[scanConfigArn]; ok {
		for projectId := range m {
			out = append(out, projectId)
		}
	}
	return out
}

// StartCodeSecurityScan creates a scan record.
func (s *Store) StartCodeSecurityScan(resource map[string]any) *StoredCodeSecurityScan {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	scan := &StoredCodeSecurityScan{
		ScanId:    generateID(),
		AccountId: s.accountID,
		Resource:  resource,
		Status:    "IN_PROGRESS",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.codeSecurityScans[scan.ScanId] = scan
	return scan
}

// GetCodeSecurityScan returns a scan record.
func (s *Store) GetCodeSecurityScan(scanId string) (*StoredCodeSecurityScan, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	scan, ok := s.codeSecurityScans[scanId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Code security scan not found: "+scanId, 404)
	}
	return scan, nil
}

// ── Members ──────────────────────────────────────────────────────────────────

// AssociateMember adds a member account.
func (s *Store) AssociateMember(accountId string) (*StoredMember, *service.AWSError) {
	if accountId == "" {
		return nil, service.ErrValidation("accountId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m := &StoredMember{
		AccountId:               accountId,
		DelegatedAdminAccountId: s.accountID,
		RelationshipStatus:      "INVITED",
		UpdatedAt:               time.Now().UTC(),
	}
	s.members[accountId] = m
	return m, nil
}

// DisassociateMember removes a member account.
func (s *Store) DisassociateMember(accountId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.members[accountId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException", "Member not found: "+accountId, 404)
	}
	m.RelationshipStatus = "REMOVED"
	m.UpdatedAt = time.Now().UTC()
	return nil
}

// GetMember returns a member.
func (s *Store) GetMember(accountId string) (*StoredMember, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.members[accountId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Member not found: "+accountId, 404)
	}
	return m, nil
}

// ListMembers returns all members.
func (s *Store) ListMembers(onlyAssociated bool) []*StoredMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMember, 0, len(s.members))
	for _, m := range s.members {
		if onlyAssociated && m.RelationshipStatus == "REMOVED" {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ── Delegated admin accounts ─────────────────────────────────────────────────

// EnableDelegatedAdminAccount marks an account as the delegated admin.
func (s *Store) EnableDelegatedAdminAccount(accountId string) *service.AWSError {
	if accountId == "" {
		return service.ErrValidation("delegatedAdminAccountId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delegatedAdminAccounts[accountId] = "ENABLED"
	return nil
}

// DisableDelegatedAdminAccount removes a delegated admin account.
func (s *Store) DisableDelegatedAdminAccount(accountId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.delegatedAdminAccounts[accountId]; !ok {
		return service.NewAWSError("ResourceNotFoundException", "Delegated admin account not found: "+accountId, 404)
	}
	delete(s.delegatedAdminAccounts, accountId)
	return nil
}

// GetDelegatedAdminAccount returns the first delegated admin account, if any.
func (s *Store) GetDelegatedAdminAccount() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, status := range s.delegatedAdminAccounts {
		return id, status
	}
	return "", ""
}

// ListDelegatedAdminAccounts returns all delegated admin accounts.
func (s *Store) ListDelegatedAdminAccounts() []map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]string, 0, len(s.delegatedAdminAccounts))
	for id, status := range s.delegatedAdminAccounts {
		out = append(out, map[string]string{
			"accountId": id,
			"status":    status,
		})
	}
	return out
}

// ── Findings reports / SBOM exports ──────────────────────────────────────────

// CreateFindingsReport starts a findings report job.
func (s *Store) CreateFindingsReport(format string, destination, filterCriteria map[string]any) *StoredFindingsReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	report := &StoredFindingsReport{
		ReportId:       generateID(),
		Status:         "IN_PROGRESS",
		ReportFormat:   format,
		Destination:    destination,
		FilterCriteria: filterCriteria,
		StartedAt:      time.Now().UTC(),
	}
	s.findingsReports[report.ReportId] = report
	return report
}

// GetFindingsReport returns a findings report by id.
func (s *Store) GetFindingsReport(reportId string) (*StoredFindingsReport, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.findingsReports[reportId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "Findings report not found: "+reportId, 404)
	}
	return r, nil
}

// CancelFindingsReport sets a findings report to CANCELLED.
func (s *Store) CancelFindingsReport(reportId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.findingsReports[reportId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException", "Findings report not found: "+reportId, 404)
	}
	r.Status = "CANCELLED"
	r.CompletedAt = time.Now().UTC()
	return nil
}

// CreateSbomExport starts an SBOM export job.
func (s *Store) CreateSbomExport(format string, destination, filterCriteria map[string]any) *StoredSbomExport {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp := &StoredSbomExport{
		ReportId:       generateID(),
		Status:         "IN_PROGRESS",
		Format:         format,
		Destination:    destination,
		FilterCriteria: filterCriteria,
		StartedAt:      time.Now().UTC(),
	}
	s.sbomExports[exp.ReportId] = exp
	return exp
}

// GetSbomExport returns an SBOM export.
func (s *Store) GetSbomExport(reportId string) (*StoredSbomExport, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.sbomExports[reportId]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException", "SBOM export not found: "+reportId, 404)
	}
	return e, nil
}

// CancelSbomExport sets an SBOM export to CANCELLED.
func (s *Store) CancelSbomExport(reportId string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sbomExports[reportId]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException", "SBOM export not found: "+reportId, 404)
	}
	e.Status = "CANCELLED"
	e.CompletedAt = time.Now().UTC()
	return nil
}

// ── EC2 deep inspection ──────────────────────────────────────────────────────

// UpdateEc2DeepInspection updates an account's deep inspection state.
func (s *Store) UpdateEc2DeepInspection(accountId string, packagePaths []string, activate bool) *StoredEc2DeepInspection {
	s.mu.Lock()
	defer s.mu.Unlock()
	di, ok := s.ec2DeepInspections[accountId]
	if !ok {
		di = &StoredEc2DeepInspection{AccountId: accountId}
		s.ec2DeepInspections[accountId] = di
	}
	if packagePaths != nil {
		di.PackagePaths = packagePaths
	}
	if activate {
		di.Status = "ACTIVATED"
	} else {
		di.Status = "DEACTIVATED"
	}
	return di
}

// GetEc2DeepInspection returns an account's deep inspection state.
func (s *Store) GetEc2DeepInspection(accountId string) *StoredEc2DeepInspection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	di, ok := s.ec2DeepInspections[accountId]
	if !ok {
		return &StoredEc2DeepInspection{AccountId: accountId, Status: "DEACTIVATED", OrgPackagePaths: s.orgPackagePaths}
	}
	out := *di
	out.OrgPackagePaths = s.orgPackagePaths
	return &out
}

// SetOrgEc2DeepInspectionPackagePaths sets the org-wide package paths.
func (s *Store) SetOrgEc2DeepInspectionPackagePaths(paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orgPackagePaths = paths
}

// OrgEc2DeepInspectionPackagePaths returns the org-wide package paths.
func (s *Store) OrgEc2DeepInspectionPackagePaths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.orgPackagePaths))
	copy(out, s.orgPackagePaths)
	return out
}

// ── Encryption keys ──────────────────────────────────────────────────────────

// SetEncryptionKey sets a kms key for a (scanType, resourceType) pair.
func (s *Store) SetEncryptionKey(scanType, resourceType, kmsKeyId string) *service.AWSError {
	if scanType == "" || resourceType == "" {
		return service.ErrValidation("scanType and resourceType are required")
	}
	if kmsKeyId == "" {
		return service.ErrValidation("kmsKeyId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scanType + ":" + resourceType
	s.encryptionKeys[key] = &StoredEncryptionKey{
		ScanType:     scanType,
		ResourceType: resourceType,
		KmsKeyId:     kmsKeyId,
	}
	return nil
}

// GetEncryptionKey returns the kms key for a (scanType, resourceType) pair.
func (s *Store) GetEncryptionKey(scanType, resourceType string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := scanType + ":" + resourceType
	if k, ok := s.encryptionKeys[key]; ok {
		return k.KmsKeyId
	}
	return ""
}

// ResetEncryptionKey removes the kms key for a (scanType, resourceType) pair.
func (s *Store) ResetEncryptionKey(scanType, resourceType string) *service.AWSError {
	if scanType == "" || resourceType == "" {
		return service.ErrValidation("scanType and resourceType are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scanType + ":" + resourceType
	delete(s.encryptionKeys, key)
	return nil
}

// ── Account statuses & enable/disable ────────────────────────────────────────

// EnableAccount sets the status for the given accounts and resource types.
func (s *Store) EnableAccount(accountIds []string, resourceTypes []string) []*StoredAccountStatus {
	if len(accountIds) == 0 {
		accountIds = []string{s.accountID}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"EC2", "ECR", "LAMBDA", "LAMBDA_CODE", "CODE_REPOSITORY"}
	}
	out := make([]*StoredAccountStatus, 0, len(accountIds))
	for _, id := range accountIds {
		st, ok := s.accountStatuses[id]
		if !ok {
			st = &StoredAccountStatus{
				AccountId:      id,
				Ec2:            "DISABLED",
				Ecr:            "DISABLED",
				Lambda:         "DISABLED",
				LambdaCode:     "DISABLED",
				CodeRepository: "DISABLED",
				Status:         "DISABLED",
			}
			s.accountStatuses[id] = st
		}
		for _, rt := range resourceTypes {
			switch rt {
			case "EC2":
				st.Ec2 = "ENABLED"
			case "ECR":
				st.Ecr = "ENABLED"
			case "LAMBDA":
				st.Lambda = "ENABLED"
			case "LAMBDA_CODE":
				st.LambdaCode = "ENABLED"
			case "CODE_REPOSITORY":
				st.CodeRepository = "ENABLED"
			}
		}
		st.Status = "ENABLED"
		out = append(out, st)
	}
	return out
}

// DisableAccount disables resource types for the given accounts.
func (s *Store) DisableAccount(accountIds []string, resourceTypes []string) []*StoredAccountStatus {
	if len(accountIds) == 0 {
		accountIds = []string{s.accountID}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"EC2", "ECR", "LAMBDA", "LAMBDA_CODE", "CODE_REPOSITORY"}
	}
	out := make([]*StoredAccountStatus, 0, len(accountIds))
	for _, id := range accountIds {
		st, ok := s.accountStatuses[id]
		if !ok {
			st = &StoredAccountStatus{
				AccountId:      id,
				Ec2:            "DISABLED",
				Ecr:            "DISABLED",
				Lambda:         "DISABLED",
				LambdaCode:     "DISABLED",
				CodeRepository: "DISABLED",
				Status:         "DISABLED",
			}
			s.accountStatuses[id] = st
		}
		for _, rt := range resourceTypes {
			switch rt {
			case "EC2":
				st.Ec2 = "DISABLED"
			case "ECR":
				st.Ecr = "DISABLED"
			case "LAMBDA":
				st.Lambda = "DISABLED"
			case "LAMBDA_CODE":
				st.LambdaCode = "DISABLED"
			case "CODE_REPOSITORY":
				st.CodeRepository = "DISABLED"
			}
		}
		if st.Ec2 == "DISABLED" && st.Ecr == "DISABLED" && st.Lambda == "DISABLED" && st.LambdaCode == "DISABLED" && st.CodeRepository == "DISABLED" {
			st.Status = "DISABLED"
		}
		out = append(out, st)
	}
	return out
}

// GetAccountStatus returns a single account's status.
func (s *Store) GetAccountStatus(accountId string) *StoredAccountStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.accountStatuses[accountId]
	if !ok {
		return &StoredAccountStatus{
			AccountId:      accountId,
			Ec2:            "DISABLED",
			Ecr:            "DISABLED",
			Lambda:         "DISABLED",
			LambdaCode:     "DISABLED",
			CodeRepository: "DISABLED",
			Status:         "DISABLED",
		}
	}
	cp := *st
	return &cp
}

// ── Organization configuration ───────────────────────────────────────────────

// SetOrganizationConfiguration replaces the org auto-enable settings.
func (s *Store) SetOrganizationConfiguration(cfg *StoredOrganizationConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orgConfig = cfg
}

// GetOrganizationConfiguration returns the org auto-enable settings.
func (s *Store) GetOrganizationConfiguration() *StoredOrganizationConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := *s.orgConfig
	return &cp
}

// ── Configuration (EC2/ECR) ──────────────────────────────────────────────────

// SetEc2Configuration stores the EC2 scan configuration.
func (s *Store) SetEc2Configuration(cfg map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ec2Configuration = cfg
}

// SetEcrConfiguration stores the ECR scan configuration.
func (s *Store) SetEcrConfiguration(cfg map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ecrConfiguration = cfg
}

// GetEc2Configuration returns the stored EC2 configuration.
func (s *Store) GetEc2Configuration() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyAnyMap(s.ec2Configuration)
}

// GetEcrConfiguration returns the stored ECR configuration.
func (s *Store) GetEcrConfiguration() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyAnyMap(s.ecrConfiguration)
}

// ── Tags ─────────────────────────────────────────────────────────────────────

// TagResource sets tags on a resource ARN.
func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.resourceTags[arn][k] = v
	}
}

// UntagResource removes tag keys from a resource ARN.
func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.resourceTags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

// ListTags returns the tags for a resource ARN.
func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.resourceTags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func (s *Store) buildFilterArn(name string) string {
	return fmt.Sprintf("arn:aws:inspector2:%s:%s:owner/%s/filter/%s", s.region, s.accountID, s.accountID, name)
}

func (s *Store) buildCisScanConfigArn(id string) string {
	return fmt.Sprintf("arn:aws:inspector2:%s:%s:owner/%s/cis-configuration/%s", s.region, s.accountID, s.accountID, id)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func copyStrings(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyAnyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
