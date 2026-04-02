package apprunner

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Service represents an App Runner service.
type Service struct {
	ServiceID   string
	ServiceName string
	ServiceARN  string
	ServiceURL  string
	Status      string // CREATE_IN_PROGRESS, RUNNING, PAUSED, DELETE_IN_PROGRESS, DELETED, OPERATION_IN_PROGRESS
	SourceConfiguration *SourceConfiguration
	InstanceConfiguration *InstanceConfiguration
	AutoScalingConfigurationSummary *AutoScalingConfigSummary
	VpcConnectorSummary *VpcConnectorSummary
	Tags        map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SourceConfiguration describes the service source.
type SourceConfiguration struct {
	RepositoryType string // IMAGE, GITHUB, BITBUCKET
	ImageRepository *ImageRepository
	CodeRepository  *CodeRepository
	AutoDeploymentsEnabled bool
	AuthenticationConfiguration *AuthenticationConfiguration
}

// ImageRepository describes a container image source.
type ImageRepository struct {
	ImageIdentifier     string
	ImageRepositoryType string // ECR, ECR_PUBLIC
}

// CodeRepository describes a source code repository.
type CodeRepository struct {
	RepositoryURL string
	SourceCodeVersion *SourceCodeVersion
	CodeConfiguration *CodeConfiguration
	ConnectionARN string
}

// SourceCodeVersion specifies the version to deploy.
type SourceCodeVersion struct {
	Type  string // BRANCH
	Value string
}

// CodeConfiguration holds build and start settings.
type CodeConfiguration struct {
	ConfigurationSource string
	CodeConfigurationValues *CodeConfigurationValues
}

// CodeConfigurationValues are explicit build settings.
type CodeConfigurationValues struct {
	Runtime       string
	BuildCommand  string
	StartCommand  string
	Port          string
}

// AuthenticationConfiguration holds authentication credentials.
type AuthenticationConfiguration struct {
	AccessRoleARN string
	ConnectionARN string
}

// InstanceConfiguration describes compute resources.
type InstanceConfiguration struct {
	CPU    string
	Memory string
	InstanceRoleARN string
}

// AutoScalingConfigSummary is a reference to an ASC.
type AutoScalingConfigSummary struct {
	AutoScalingConfigurationARN  string
	AutoScalingConfigurationName string
	AutoScalingConfigurationRevision int
}

// VpcConnectorSummary is a reference to a VPC connector.
type VpcConnectorSummary struct {
	VpcConnectorARN  string
	VpcConnectorName string
	VpcConnectorRevision int
}

// Connection represents an App Runner connection (GitHub/Bitbucket OAuth).
type Connection struct {
	ConnectionName string
	ConnectionARN  string
	ProviderType   string // GITHUB, BITBUCKET
	Status         string // PENDING_HANDSHAKE, AVAILABLE, ERROR, DELETED
	Tags           map[string]string
	CreatedAt      time.Time
}

// AutoScalingConfiguration represents an App Runner ASC.
type AutoScalingConfiguration struct {
	AutoScalingConfigurationARN      string
	AutoScalingConfigurationName     string
	AutoScalingConfigurationRevision int
	Latest                           bool
	Status                           string // ACTIVE, INACTIVE
	MinSize                          int
	MaxSize                          int
	MaxConcurrency                   int
	Tags                             map[string]string
	CreatedAt                        time.Time
	DeletedAt                        *time.Time
}

// VpcConnector represents an App Runner VPC connector.
type VpcConnector struct {
	VpcConnectorARN      string
	VpcConnectorName     string
	VpcConnectorRevision int
	Subnets              []string
	SecurityGroups       []string
	Status               string // ACTIVE, INACTIVE
	Tags                 map[string]string
	CreatedAt            time.Time
	DeletedAt            *time.Time
}

// Store manages all App Runner resources in memory.
type Store struct {
	mu               sync.RWMutex
	services         map[string]*Service                    // serviceARN -> service
	servicesByName   map[string]string                      // serviceName -> serviceARN
	connections      map[string]*Connection                 // connectionARN -> connection
	ascConfigs       map[string]*AutoScalingConfiguration   // ascARN -> config
	ascRevisions     map[string]int                         // ascName -> latest revision
	vpcConnectors    map[string]*VpcConnector               // vcARN -> connector
	vcRevisions      map[string]int                         // vcName -> latest revision
	tags             map[string]map[string]string           // resourceARN -> tags
	accountID        string
	region           string
}

// NewStore returns a new empty App Runner Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		services:       make(map[string]*Service),
		servicesByName: make(map[string]string),
		connections:    make(map[string]*Connection),
		ascConfigs:     make(map[string]*AutoScalingConfiguration),
		ascRevisions:   make(map[string]int),
		vpcConnectors:  make(map[string]*VpcConnector),
		vcRevisions:    make(map[string]int),
		tags:           make(map[string]map[string]string),
		accountID:      accountID,
		region:         region,
	}
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%08x%04x%04x%04x%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) serviceARN(name string) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:service/%s/%s", s.region, s.accountID, name, newID()[:8])
}

func (s *Store) connectionARN(name string) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:connection/%s/%s", s.region, s.accountID, name, newID()[:8])
}

func (s *Store) ascARN(name string, revision int) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:autoscalingconfiguration/%s/%d/%s", s.region, s.accountID, name, revision, newID()[:8])
}

func (s *Store) vcARN(name string, revision int) string {
	return fmt.Sprintf("arn:aws:apprunner:%s:%s:vpcconnector/%s/%d/%s", s.region, s.accountID, name, revision, newID()[:8])
}

// ---- Service operations ----

// CreateService creates a new App Runner service.
func (s *Store) CreateService(name string, sourceCfg *SourceConfiguration, instanceCfg *InstanceConfiguration, tags map[string]string) (*Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.servicesByName[name]; ok {
		return nil, fmt.Errorf("service already exists: %s", name)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	id := newID()[:8]
	arn := fmt.Sprintf("arn:aws:apprunner:%s:%s:service/%s/%s", s.region, s.accountID, name, id)
	now := time.Now().UTC()
	svc := &Service{
		ServiceID:             id,
		ServiceName:           name,
		ServiceARN:            arn,
		ServiceURL:            fmt.Sprintf("%s.%s.awsapprunner.com", id, s.region),
		Status:                "RUNNING",
		SourceConfiguration:   sourceCfg,
		InstanceConfiguration: instanceCfg,
		Tags:                  tags,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.services[arn] = svc
	s.servicesByName[name] = arn
	s.tags[arn] = tags
	return svc, nil
}

// GetService retrieves a service by ARN or name.
func (s *Store) GetService(nameOrARN string) (*Service, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if svc, ok := s.services[nameOrARN]; ok {
		return svc, true
	}
	if arn, ok := s.servicesByName[nameOrARN]; ok {
		return s.services[arn], true
	}
	return nil, false
}

// ListServices returns all services.
func (s *Store) ListServices() []*Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Service, 0, len(s.services))
	for _, svc := range s.services {
		out = append(out, svc)
	}
	return out
}

// UpdateService updates service configuration.
func (s *Store) UpdateService(nameOrARN string, sourceCfg *SourceConfiguration, instanceCfg *InstanceConfiguration) (*Service, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := nameOrARN
	if _, ok := s.services[arn]; !ok {
		if a, ok := s.servicesByName[nameOrARN]; ok {
			arn = a
		} else {
			return nil, false
		}
	}
	svc := s.services[arn]
	if sourceCfg != nil {
		svc.SourceConfiguration = sourceCfg
	}
	if instanceCfg != nil {
		svc.InstanceConfiguration = instanceCfg
	}
	svc.UpdatedAt = time.Now().UTC()
	svc.Status = "OPERATION_IN_PROGRESS"
	// Simulate immediate completion.
	go func() {
		time.Sleep(10 * time.Millisecond)
		s.mu.Lock()
		svc.Status = "RUNNING"
		s.mu.Unlock()
	}()
	return svc, true
}

// DeleteService marks a service as deleted.
func (s *Store) DeleteService(nameOrARN string) (*Service, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := nameOrARN
	if _, ok := s.services[arn]; !ok {
		if a, ok := s.servicesByName[nameOrARN]; ok {
			arn = a
		} else {
			return nil, false
		}
	}
	svc := s.services[arn]
	svc.Status = "DELETED"
	svc.UpdatedAt = time.Now().UTC()
	delete(s.servicesByName, svc.ServiceName)
	delete(s.services, arn)
	return svc, true
}

// PauseService pauses a running service.
func (s *Store) PauseService(nameOrARN string) (*Service, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := nameOrARN
	if _, ok := s.services[arn]; !ok {
		if a, ok := s.servicesByName[nameOrARN]; ok {
			arn = a
		} else {
			return nil, false
		}
	}
	svc := s.services[arn]
	svc.Status = "PAUSED"
	svc.UpdatedAt = time.Now().UTC()
	return svc, true
}

// ResumeService resumes a paused service.
func (s *Store) ResumeService(nameOrARN string) (*Service, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := nameOrARN
	if _, ok := s.services[arn]; !ok {
		if a, ok := s.servicesByName[nameOrARN]; ok {
			arn = a
		} else {
			return nil, false
		}
	}
	svc := s.services[arn]
	svc.Status = "RUNNING"
	svc.UpdatedAt = time.Now().UTC()
	return svc, true
}

// ---- Connection operations ----

// CreateConnection creates a new connection.
func (s *Store) CreateConnection(name, providerType string, tags map[string]string) (*Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.connections {
		if c.ConnectionName == name {
			return nil, fmt.Errorf("connection already exists: %s", name)
		}
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	arn := s.connectionARN(name)
	conn := &Connection{
		ConnectionName: name,
		ConnectionARN:  arn,
		ProviderType:   providerType,
		Status:         "AVAILABLE",
		Tags:           tags,
		CreatedAt:      time.Now().UTC(),
	}
	s.connections[arn] = conn
	s.tags[arn] = tags
	return conn, nil
}

// GetConnection retrieves a connection by ARN.
func (s *Store) GetConnection(arn string) (*Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.connections[arn]
	return c, ok
}

// ---- AutoScalingConfiguration operations ----

// CreateAutoScalingConfiguration creates a new ASC revision.
func (s *Store) CreateAutoScalingConfiguration(name string, minSize, maxSize, maxConcurrency int, tags map[string]string) (*AutoScalingConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	s.ascRevisions[name]++
	revision := s.ascRevisions[name]
	arn := s.ascARN(name, revision)
	asc := &AutoScalingConfiguration{
		AutoScalingConfigurationARN:      arn,
		AutoScalingConfigurationName:     name,
		AutoScalingConfigurationRevision: revision,
		Latest:                           true,
		Status:                           "ACTIVE",
		MinSize:                          minSize,
		MaxSize:                          maxSize,
		MaxConcurrency:                   maxConcurrency,
		Tags:                             tags,
		CreatedAt:                        time.Now().UTC(),
	}
	// Mark older revisions as not latest.
	for _, existing := range s.ascConfigs {
		if existing.AutoScalingConfigurationName == name {
			existing.Latest = false
		}
	}
	s.ascConfigs[arn] = asc
	s.tags[arn] = tags
	return asc, nil
}

// DescribeAutoScalingConfiguration retrieves an ASC by ARN.
func (s *Store) DescribeAutoScalingConfiguration(arn string) (*AutoScalingConfiguration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asc, ok := s.ascConfigs[arn]
	return asc, ok
}

// ListAutoScalingConfigurations returns all ASC configurations.
func (s *Store) ListAutoScalingConfigurations(name string) []*AutoScalingConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*AutoScalingConfiguration, 0)
	for _, asc := range s.ascConfigs {
		if name != "" && asc.AutoScalingConfigurationName != name {
			continue
		}
		out = append(out, asc)
	}
	return out
}

// DeleteAutoScalingConfiguration marks an ASC as deleted.
func (s *Store) DeleteAutoScalingConfiguration(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	asc, ok := s.ascConfigs[arn]
	if !ok {
		return false
	}
	now := time.Now().UTC()
	asc.Status = "INACTIVE"
	asc.DeletedAt = &now
	delete(s.ascConfigs, arn)
	return true
}

// ---- VpcConnector operations ----

// CreateVpcConnector creates a new VPC connector.
func (s *Store) CreateVpcConnector(name string, subnets, securityGroups []string, tags map[string]string) (*VpcConnector, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	s.vcRevisions[name]++
	revision := s.vcRevisions[name]
	arn := s.vcARN(name, revision)
	vc := &VpcConnector{
		VpcConnectorARN:      arn,
		VpcConnectorName:     name,
		VpcConnectorRevision: revision,
		Subnets:              subnets,
		SecurityGroups:       securityGroups,
		Status:               "ACTIVE",
		Tags:                 tags,
		CreatedAt:            time.Now().UTC(),
	}
	s.vpcConnectors[arn] = vc
	s.tags[arn] = tags
	return vc, nil
}

// DescribeVpcConnector retrieves a VPC connector by ARN.
func (s *Store) DescribeVpcConnector(arn string) (*VpcConnector, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vc, ok := s.vpcConnectors[arn]
	return vc, ok
}

// ListVpcConnectors returns all VPC connectors.
func (s *Store) ListVpcConnectors() []*VpcConnector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*VpcConnector, 0, len(s.vpcConnectors))
	for _, vc := range s.vpcConnectors {
		out = append(out, vc)
	}
	return out
}

// DeleteVpcConnector marks a VPC connector as deleted.
func (s *Store) DeleteVpcConnector(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	vc, ok := s.vpcConnectors[arn]
	if !ok {
		return false
	}
	now := time.Now().UTC()
	vc.Status = "INACTIVE"
	vc.DeletedAt = &now
	delete(s.vpcConnectors, arn)
	return true
}

// ---- Tag operations ----

// TagResource applies tags to any resource by ARN.
func (s *Store) TagResource(resourceARN string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tags[resourceARN]; !ok {
		s.tags[resourceARN] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[resourceARN][k] = v
	}
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(resourceARN string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tags[resourceARN]; ok {
		for _, k := range keys {
			delete(t, k)
		}
	}
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(resourceARN string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range s.tags[resourceARN] {
		result[k] = v
	}
	return result
}
