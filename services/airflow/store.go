package airflow

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

type EnvironmentStatus string

const (
	EnvCreating      EnvironmentStatus = "CREATING"
	EnvCreateFailed  EnvironmentStatus = "CREATE_FAILED"
	EnvAvailable     EnvironmentStatus = "AVAILABLE"
	EnvUpdating      EnvironmentStatus = "UPDATING"
	EnvDeleting      EnvironmentStatus = "DELETING"
	EnvDeleted       EnvironmentStatus = "DELETED"
	EnvUpdateFailed  EnvironmentStatus = "UPDATE_FAILED"
)

type Environment struct {
	Name                   string
	Arn                    string
	Status                 EnvironmentStatus
	AirflowVersion         string
	SourceBucketArn        string
	DagS3Path              string
	EnvironmentClass       string
	MaxWorkers             int
	MinWorkers             int
	Schedulers             int
	ExecutionRoleArn       string
	WebserverAccessMode    string
	WebserverUrl           string
	NetworkConfiguration   map[string]any
	LoggingConfiguration   map[string]any
	AirflowConfigurationOptions map[string]string
	WeeklyMaintenanceWindowStart string
	ServiceRoleArn         string
	CreatedAt              time.Time
	LastUpdate             time.Time
	Tags                   map[string]string
	Lifecycle              *lifecycle.Machine
}

type Store struct {
	mu           sync.RWMutex
	environments map[string]*Environment // keyed by name
	tagsByArn    map[string]map[string]string
	accountID    string
	region       string
	lcConfig     *lifecycle.Config
}

func NewStore(accountID, region string) *Store {
	return &Store{
		environments: make(map[string]*Environment),
		tagsByArn:    make(map[string]map[string]string),
		accountID:    accountID,
		region:       region,
		lcConfig:     lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) envARN(name string) string {
	return fmt.Sprintf("arn:aws:airflow:%s:%s:environment/%s", s.region, s.accountID, name)
}

func (s *Store) CreateEnvironment(name, airflowVersion, sourceBucketArn, dagS3Path, envClass string, maxWorkers, minWorkers, schedulers int, executionRoleArn, webserverAccessMode string, networkConfig, loggingConfig map[string]any, airflowConfigOptions map[string]string, maintenanceWindow string, tags map[string]string) (*Environment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.environments[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Environment %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if airflowVersion == "" {
		airflowVersion = "2.8.1"
	}
	if envClass == "" {
		envClass = "mw1.small"
	}
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	if minWorkers <= 0 {
		minWorkers = 1
	}
	if schedulers <= 0 {
		schedulers = 2
	}
	if webserverAccessMode == "" {
		webserverAccessMode = "PRIVATE_ONLY"
	}

	arn := s.envARN(name)
	lc := lifecycle.NewMachine(
		lifecycle.State(EnvCreating),
		[]lifecycle.Transition{
			{From: lifecycle.State(EnvCreating), To: lifecycle.State(EnvAvailable), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)

	now := time.Now().UTC()
	env := &Environment{
		Name:                         name,
		Arn:                          arn,
		Status:                       EnvironmentStatus(lc.State()),
		AirflowVersion:               airflowVersion,
		SourceBucketArn:              sourceBucketArn,
		DagS3Path:                    dagS3Path,
		EnvironmentClass:             envClass,
		MaxWorkers:                   maxWorkers,
		MinWorkers:                   minWorkers,
		Schedulers:                   schedulers,
		ExecutionRoleArn:             executionRoleArn,
		WebserverAccessMode:          webserverAccessMode,
		WebserverUrl:                 fmt.Sprintf("https://%s.%s.airflow.amazonaws.com", newUUID()[:8], s.region),
		NetworkConfiguration:         networkConfig,
		LoggingConfiguration:         loggingConfig,
		AirflowConfigurationOptions: airflowConfigOptions,
		WeeklyMaintenanceWindowStart: maintenanceWindow,
		ServiceRoleArn:               fmt.Sprintf("arn:aws:iam::%s:role/aws-service-role/airflow.amazonaws.com/AWSServiceRoleForAmazonMWAA", s.accountID),
		CreatedAt:                    now,
		LastUpdate:                   now,
		Tags:                         tags,
		Lifecycle:                    lc,
	}
	s.environments[name] = env
	s.tagsByArn[arn] = tags
	return env, nil
}

func (s *Store) GetEnvironment(name string) (*Environment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.environments[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Environment %s not found", name), http.StatusNotFound)
	}
	env.Status = EnvironmentStatus(env.Lifecycle.State())
	return env, nil
}

func (s *Store) ListEnvironments() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.environments))
	for name := range s.environments {
		names = append(names, name)
	}
	return names
}

func (s *Store) UpdateEnvironment(name string, airflowVersion, sourceBucketArn, dagS3Path, envClass string, maxWorkers, minWorkers, schedulers int, executionRoleArn, webserverAccessMode string, networkConfig, loggingConfig map[string]any, airflowConfigOptions map[string]string, maintenanceWindow string) (*Environment, *service.AWSError) {
	s.mu.Lock()
	env, ok := s.environments[name]
	if !ok {
		s.mu.Unlock()
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Environment %s not found", name), http.StatusNotFound)
	}
	if airflowVersion != "" {
		env.AirflowVersion = airflowVersion
	}
	if sourceBucketArn != "" {
		env.SourceBucketArn = sourceBucketArn
	}
	if dagS3Path != "" {
		env.DagS3Path = dagS3Path
	}
	if envClass != "" {
		env.EnvironmentClass = envClass
	}
	if maxWorkers > 0 {
		env.MaxWorkers = maxWorkers
	}
	if minWorkers > 0 {
		env.MinWorkers = minWorkers
	}
	if schedulers > 0 {
		env.Schedulers = schedulers
	}
	if executionRoleArn != "" {
		env.ExecutionRoleArn = executionRoleArn
	}
	if webserverAccessMode != "" {
		env.WebserverAccessMode = webserverAccessMode
	}
	if networkConfig != nil {
		env.NetworkConfiguration = networkConfig
	}
	if loggingConfig != nil {
		env.LoggingConfiguration = loggingConfig
	}
	if airflowConfigOptions != nil {
		env.AirflowConfigurationOptions = airflowConfigOptions
	}
	if maintenanceWindow != "" {
		env.WeeklyMaintenanceWindowStart = maintenanceWindow
	}
	env.LastUpdate = time.Now().UTC()
	lc := env.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(EnvUpdating))
	}
	return env, nil
}

func (s *Store) DeleteEnvironment(name string) *service.AWSError {
	s.mu.Lock()
	env, ok := s.environments[name]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Environment %s not found", name), http.StatusNotFound)
	}
	lc := env.Lifecycle
	delete(s.environments, name)
	delete(s.tagsByArn, env.Arn)
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(EnvDeleting))
	}
	return nil
}

func (s *Store) CreateCliToken(name string) (string, string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.environments[name]
	if !ok {
		return "", "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Environment %s not found", name), http.StatusNotFound)
	}
	return "mock-cli-token-" + newUUID(), env.WebserverUrl, nil
}

func (s *Store) CreateWebLoginToken(name string) (string, string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.environments[name]
	if !ok {
		return "", "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Environment %s not found", name), http.StatusNotFound)
	}
	return "mock-web-login-token-" + newUUID(), env.WebserverUrl, nil
}

// Tags.

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for _, k := range tagKeys {
		delete(existing, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	cp := make(map[string]string, len(existing))
	for k, v := range existing {
		cp[k] = v
	}
	return cp, nil
}
