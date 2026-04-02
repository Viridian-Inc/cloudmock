package elasticbeanstalk

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Application represents an Elastic Beanstalk application.
type Application struct {
	ApplicationName string
	ApplicationArn  string
	Description     string
	DateCreated     time.Time
	DateUpdated     time.Time
	Versions        []string
}

// ApplicationVersion represents an application version.
type ApplicationVersion struct {
	ApplicationName string
	VersionLabel    string
	Description     string
	SourceBundle    SourceBundle
	DateCreated     time.Time
	DateUpdated     time.Time
	Status          string
}

// SourceBundle represents the S3 location of an application version.
type SourceBundle struct {
	S3Bucket string
	S3Key    string
}

// Environment represents an Elastic Beanstalk environment.
type Environment struct {
	EnvironmentID    string
	EnvironmentName  string
	ApplicationName  string
	VersionLabel     string
	TemplateName     string
	Description      string
	EndpointURL      string
	CNAME            string
	Status           string
	Health           string
	HealthStatus     string
	SolutionStackName string
	PlatformArn      string
	Tier             EnvironmentTier
	DateCreated      time.Time
	DateUpdated      time.Time
	lifecycle        *lifecycle.Machine
}

// EnvironmentTier represents the tier of an environment.
type EnvironmentTier struct {
	Name    string
	Type    string
	Version string
}

// ConfigurationTemplate represents a configuration template.
type ConfigurationTemplate struct {
	ApplicationName   string
	TemplateName      string
	Description       string
	SolutionStackName string
	PlatformArn       string
	DateCreated       time.Time
	DateUpdated       time.Time
}

// Store manages Elastic Beanstalk resources in memory.
type Store struct {
	mu         sync.RWMutex
	apps       map[string]*Application
	versions   map[string]map[string]*ApplicationVersion // appName -> versionLabel -> version
	envs       map[string]*Environment                   // envName -> env
	templates  map[string]map[string]*ConfigurationTemplate // appName -> templateName -> template
	accountID  string
	region     string
	lcConfig   *lifecycle.Config
	envSeq     int
}

// NewStore returns a new empty Elastic Beanstalk Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		apps:      make(map[string]*Application),
		versions:  make(map[string]map[string]*ApplicationVersion),
		envs:      make(map[string]*Environment),
		templates: make(map[string]map[string]*ConfigurationTemplate),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// CreateApplication creates a new application.
func (s *Store) CreateApplication(name, description string) (*Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[name]; ok {
		return nil, fmt.Errorf("application already exists: %s", name)
	}

	now := time.Now().UTC()
	app := &Application{
		ApplicationName: name,
		ApplicationArn:  fmt.Sprintf("arn:aws:elasticbeanstalk:%s:%s:application/%s", s.region, s.accountID, name),
		Description:     description,
		DateCreated:     now,
		DateUpdated:     now,
	}
	s.apps[name] = app
	s.versions[name] = make(map[string]*ApplicationVersion)
	s.templates[name] = make(map[string]*ConfigurationTemplate)
	return app, nil
}

// GetApplication retrieves an application.
func (s *Store) GetApplication(name string) (*Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[name]
	return app, ok
}

// ListApplications returns all applications.
func (s *Store) ListApplications() []*Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Application, 0, len(s.apps))
	for _, a := range s.apps {
		out = append(out, a)
	}
	return out
}

// DeleteApplication removes an application.
func (s *Store) DeleteApplication(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[name]; !ok {
		return false
	}
	delete(s.apps, name)
	delete(s.versions, name)
	delete(s.templates, name)
	return true
}

// CreateApplicationVersion creates a new application version.
func (s *Store) CreateApplicationVersion(appName, versionLabel, description, s3Bucket, s3Key string) (*ApplicationVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appName]; !ok {
		return nil, fmt.Errorf("application not found: %s", appName)
	}

	// Enforce version label uniqueness per application
	if _, exists := s.versions[appName][versionLabel]; exists {
		return nil, fmt.Errorf("version label already exists for application %s: %s", appName, versionLabel)
	}

	now := time.Now().UTC()
	ver := &ApplicationVersion{
		ApplicationName: appName,
		VersionLabel:    versionLabel,
		Description:     description,
		SourceBundle:    SourceBundle{S3Bucket: s3Bucket, S3Key: s3Key},
		DateCreated:     now,
		DateUpdated:     now,
		Status:          "PROCESSED",
	}
	s.versions[appName][versionLabel] = ver
	s.apps[appName].Versions = append(s.apps[appName].Versions, versionLabel)
	return ver, nil
}

// ListApplicationVersions returns all versions for an app.
func (s *Store) ListApplicationVersions(appName string) []*ApplicationVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	verMap := s.versions[appName]
	out := make([]*ApplicationVersion, 0, len(verMap))
	for _, v := range verMap {
		out = append(out, v)
	}
	return out
}

// CreateEnvironment creates a new environment.
func (s *Store) CreateEnvironment(appName, envName, versionLabel, description, solutionStack, templateName string, tier EnvironmentTier) (*Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appName]; !ok {
		return nil, fmt.Errorf("application not found: %s", appName)
	}
	if _, ok := s.envs[envName]; ok {
		return nil, fmt.Errorf("environment already exists: %s", envName)
	}

	s.envSeq++
	envID := fmt.Sprintf("e-%s", newUUID()[:8])

	if tier.Name == "" {
		tier = EnvironmentTier{Name: "WebServer", Type: "Standard", Version: "1.0"}
	}

	transitions := []lifecycle.Transition{
		{From: "Launching", To: "Ready", Delay: 3 * time.Second},
	}

	now := time.Now().UTC()
	env := &Environment{
		EnvironmentID:    envID,
		EnvironmentName:  envName,
		ApplicationName:  appName,
		VersionLabel:     versionLabel,
		TemplateName:     templateName,
		Description:      description,
		EndpointURL:      fmt.Sprintf("%s.%s.elasticbeanstalk.com", envName, s.region),
		CNAME:            fmt.Sprintf("%s.%s.elasticbeanstalk.com", envName, s.region),
		Status:           "Launching",
		Health:           "Grey",
		HealthStatus:     "Unknown",
		SolutionStackName: solutionStack,
		Tier:             tier,
		DateCreated:      now,
		DateUpdated:      now,
	}
	env.lifecycle = lifecycle.NewMachine("Launching", transitions, s.lcConfig)
	env.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		env.Status = string(to)
		env.DateUpdated = time.Now().UTC()
		if to == "Ready" {
			env.Health = "Green"
			env.HealthStatus = "Ok"
		}
	})

	s.envs[envName] = env
	return env, nil
}

// GetEnvironment retrieves an environment.
func (s *Store) GetEnvironment(name string) (*Environment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	env, ok := s.envs[name]
	if ok && env.lifecycle != nil {
		env.Status = string(env.lifecycle.State())
	}
	return env, ok
}

// ListEnvironments returns all environments, optionally filtered by app.
func (s *Store) ListEnvironments(appName string) []*Environment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Environment, 0)
	for _, env := range s.envs {
		if appName == "" || env.ApplicationName == appName {
			out = append(out, env)
		}
	}
	return out
}

// TerminateEnvironment terminates an environment.
func (s *Store) TerminateEnvironment(name string) (*Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	env, ok := s.envs[name]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", name)
	}
	if env.lifecycle != nil {
		env.lifecycle.Stop()
	}

	transitions := []lifecycle.Transition{
		{From: "Terminating", To: "Terminated", Delay: 3 * time.Second},
	}
	env.Status = "Terminating"
	env.DateUpdated = time.Now().UTC()
	env.lifecycle = lifecycle.NewMachine("Terminating", transitions, s.lcConfig)
	env.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		env.Status = string(to)
		env.DateUpdated = time.Now().UTC()
		if to == "Terminated" {
			env.Health = "Grey"
			env.HealthStatus = "Unknown"
		}
	})
	return env, nil
}

// CreateConfigurationTemplate creates a configuration template.
func (s *Store) CreateConfigurationTemplate(appName, templateName, description, solutionStack, platformArn string) (*ConfigurationTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[appName]; !ok {
		return nil, fmt.Errorf("application not found: %s", appName)
	}

	now := time.Now().UTC()
	tmpl := &ConfigurationTemplate{
		ApplicationName:   appName,
		TemplateName:      templateName,
		Description:       description,
		SolutionStackName: solutionStack,
		PlatformArn:       platformArn,
		DateCreated:       now,
		DateUpdated:       now,
	}
	s.templates[appName][templateName] = tmpl
	return tmpl, nil
}

// GetConfigurationTemplate retrieves a configuration template.
func (s *Store) GetConfigurationTemplate(appName, templateName string) (*ConfigurationTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tmplMap, ok := s.templates[appName]
	if !ok {
		return nil, false
	}
	tmpl, ok := tmplMap[templateName]
	return tmpl, ok
}

// DeleteConfigurationTemplate removes a configuration template.
func (s *Store) DeleteConfigurationTemplate(appName, templateName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmplMap, ok := s.templates[appName]
	if !ok {
		return false
	}
	if _, ok := tmplMap[templateName]; !ok {
		return false
	}
	delete(tmplMap, templateName)
	return true
}

// UpdateApplication updates an application's description.
func (s *Store) UpdateApplication(name, description string) (*Application, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.apps[name]
	if !ok {
		return nil, false
	}
	app.Description = description
	app.DateUpdated = time.Now().UTC()
	return app, true
}

// UpdateEnvironment updates an environment's version or description.
func (s *Store) UpdateEnvironment(envName, versionLabel, description string) (*Environment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	env, ok := s.envs[envName]
	if !ok {
		return nil, false
	}
	if versionLabel != "" {
		env.VersionLabel = versionLabel
	}
	if description != "" {
		env.Description = description
	}
	env.DateUpdated = time.Now().UTC()
	return env, true
}

// DeleteApplicationVersion removes an application version.
func (s *Store) DeleteApplicationVersion(appName, versionLabel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	verMap, ok := s.versions[appName]
	if !ok {
		return false
	}
	if _, ok := verMap[versionLabel]; !ok {
		return false
	}
	delete(verMap, versionLabel)
	// Remove from app's Versions list.
	if app, ok := s.apps[appName]; ok {
		newVersions := make([]string, 0)
		for _, v := range app.Versions {
			if v != versionLabel {
				newVersions = append(newVersions, v)
			}
		}
		app.Versions = newVersions
	}
	return true
}

// ListPlatformVersions returns a mock list of supported platform versions.
func (s *Store) ListPlatformVersions() []map[string]string {
	return []map[string]string{
		{"PlatformArn": fmt.Sprintf("arn:aws:elasticbeanstalk:%s::platform/Docker running on 64bit Amazon Linux 2023/4.0.0", s.region), "PlatformStatus": "Ready", "PlatformCategory": "Docker", "OperatingSystemName": "Amazon Linux 2023"},
		{"PlatformArn": fmt.Sprintf("arn:aws:elasticbeanstalk:%s::platform/Node.js 18 running on 64bit Amazon Linux 2023/6.0.0", s.region), "PlatformStatus": "Ready", "PlatformCategory": "Node.js", "OperatingSystemName": "Amazon Linux 2023"},
		{"PlatformArn": fmt.Sprintf("arn:aws:elasticbeanstalk:%s::platform/Python 3.11 running on 64bit Amazon Linux 2023/4.0.0", s.region), "PlatformStatus": "Ready", "PlatformCategory": "Python", "OperatingSystemName": "Amazon Linux 2023"},
		{"PlatformArn": fmt.Sprintf("arn:aws:elasticbeanstalk:%s::platform/Java 17 running on 64bit Amazon Linux 2023/4.0.0", s.region), "PlatformStatus": "Ready", "PlatformCategory": "Java", "OperatingSystemName": "Amazon Linux 2023"},
	}
}
