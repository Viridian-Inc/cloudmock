package appconfig

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Application represents an AppConfig application.
type Application struct {
	ID          string
	Name        string
	Description string
	Tags        map[string]string
}

// Environment represents an AppConfig environment.
type Environment struct {
	ApplicationID string
	ID            string
	Name          string
	Description   string
	State         string
	Tags          map[string]string
}

// ConfigurationProfile represents an AppConfig configuration profile.
type ConfigurationProfile struct {
	ApplicationID string
	ID            string
	Name          string
	Description   string
	LocationURI   string
	Type          string
	Tags          map[string]string
}

// DeploymentStrategy represents an AppConfig deployment strategy.
type DeploymentStrategy struct {
	ID                     string
	Name                   string
	Description            string
	DeploymentDurationInMinutes int
	GrowthFactor           float64
	GrowthType             string
	FinalBakeTimeInMinutes int
	ReplicateTo            string
	Tags                   map[string]string
}

// Deployment represents an AppConfig deployment.
type Deployment struct {
	ApplicationID          string
	EnvironmentID          string
	DeploymentStrategyID   string
	ConfigurationProfileID string
	ConfigurationVersion   string
	DeploymentNumber       int
	State                  string
	Description            string
	StartedAt              time.Time
	CompletedAt            *time.Time
	PercentageComplete     float64
	Lifecycle              *lifecycle.Machine
	Tags                   map[string]string
}

// Store manages all AppConfig state in memory.
type Store struct {
	mu                  sync.RWMutex
	applications        map[string]*Application
	environments        map[string]map[string]*Environment // appID -> envID -> env
	configProfiles      map[string]map[string]*ConfigurationProfile // appID -> profileID -> profile
	deploymentStrategies map[string]*DeploymentStrategy
	deployments         map[string]map[string][]*Deployment // appID -> envID -> deployments
	accountID           string
	region              string
	nextAppID           int
	nextEnvID           int
	nextProfileID       int
	nextStrategyID      int
	nextDeployNum       map[string]int // appID:envID -> next deployment number
	lifecycleConfig     *lifecycle.Config
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		applications:         make(map[string]*Application),
		environments:         make(map[string]map[string]*Environment),
		configProfiles:       make(map[string]map[string]*ConfigurationProfile),
		deploymentStrategies: make(map[string]*DeploymentStrategy),
		deployments:          make(map[string]map[string][]*Deployment),
		accountID:            accountID,
		region:               region,
		nextAppID:            1,
		nextEnvID:            1,
		nextProfileID:        1,
		nextStrategyID:       1,
		nextDeployNum:        make(map[string]int),
		lifecycleConfig:      lifecycle.DefaultConfig(),
	}
}

func (s *Store) appARN(appID string) string {
	return fmt.Sprintf("arn:aws:appconfig:%s:%s:application/%s", s.region, s.accountID, appID)
}

func (s *Store) envARN(appID, envID string) string {
	return fmt.Sprintf("arn:aws:appconfig:%s:%s:application/%s/environment/%s", s.region, s.accountID, appID, envID)
}

func (s *Store) profileARN(appID, profileID string) string {
	return fmt.Sprintf("arn:aws:appconfig:%s:%s:application/%s/configurationprofile/%s", s.region, s.accountID, appID, profileID)
}

func (s *Store) strategyARN(strategyID string) string {
	return fmt.Sprintf("arn:aws:appconfig:%s:%s:deploymentstrategy/%s", s.region, s.accountID, strategyID)
}

func (s *Store) generateAppID() string {
	id := fmt.Sprintf("%07d", s.nextAppID)
	s.nextAppID++
	return id
}

func (s *Store) generateEnvID() string {
	id := fmt.Sprintf("%07d", s.nextEnvID)
	s.nextEnvID++
	return id
}

func (s *Store) generateProfileID() string {
	id := fmt.Sprintf("%07d", s.nextProfileID)
	s.nextProfileID++
	return id
}

func (s *Store) generateStrategyID() string {
	id := fmt.Sprintf("%07d", s.nextStrategyID)
	s.nextStrategyID++
	return id
}

// CreateApplication creates a new application.
func (s *Store) CreateApplication(name, description string, tags map[string]string) *Application {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	id := s.generateAppID()
	app := &Application{ID: id, Name: name, Description: description, Tags: tags}
	s.applications[id] = app
	s.environments[id] = make(map[string]*Environment)
	s.configProfiles[id] = make(map[string]*ConfigurationProfile)
	s.deployments[id] = make(map[string][]*Deployment)
	return app
}

// GetApplication returns an application by ID.
func (s *Store) GetApplication(id string) (*Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.applications[id]
	return app, ok
}

// ListApplications returns all applications.
func (s *Store) ListApplications() []*Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Application, 0, len(s.applications))
	for _, app := range s.applications {
		result = append(result, app)
	}
	return result
}

// UpdateApplication updates an application.
func (s *Store) UpdateApplication(id, name, description string) (*Application, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.applications[id]
	if !ok {
		return nil, false
	}
	if name != "" {
		app.Name = name
	}
	if description != "" {
		app.Description = description
	}
	return app, true
}

// DeleteApplication removes an application.
func (s *Store) DeleteApplication(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[id]; !ok {
		return false
	}
	delete(s.applications, id)
	delete(s.environments, id)
	delete(s.configProfiles, id)
	delete(s.deployments, id)
	return true
}

// CreateEnvironment creates a new environment.
func (s *Store) CreateEnvironment(appID, name, description string, tags map[string]string) (*Environment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[appID]; !ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	id := s.generateEnvID()
	env := &Environment{
		ApplicationID: appID, ID: id, Name: name,
		Description: description, State: "READY_FOR_DEPLOYMENT", Tags: tags,
	}
	s.environments[appID][id] = env
	return env, true
}

// GetEnvironment returns an environment by app and env ID.
func (s *Store) GetEnvironment(appID, envID string) (*Environment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	envs, ok := s.environments[appID]
	if !ok {
		return nil, false
	}
	env, ok := envs[envID]
	return env, ok
}

// ListEnvironments returns all environments for an application.
func (s *Store) ListEnvironments(appID string) ([]*Environment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	envs, ok := s.environments[appID]
	if !ok {
		return nil, false
	}
	result := make([]*Environment, 0, len(envs))
	for _, env := range envs {
		result = append(result, env)
	}
	return result, true
}

// UpdateEnvironment updates an environment.
func (s *Store) UpdateEnvironment(appID, envID, name, description string) (*Environment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	envs, ok := s.environments[appID]
	if !ok {
		return nil, false
	}
	env, ok := envs[envID]
	if !ok {
		return nil, false
	}
	if name != "" {
		env.Name = name
	}
	if description != "" {
		env.Description = description
	}
	return env, true
}

// DeleteEnvironment removes an environment.
func (s *Store) DeleteEnvironment(appID, envID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	envs, ok := s.environments[appID]
	if !ok {
		return false
	}
	if _, ok := envs[envID]; !ok {
		return false
	}
	delete(envs, envID)
	return true
}

// CreateConfigurationProfile creates a new configuration profile.
func (s *Store) CreateConfigurationProfile(appID, name, description, locationURI, profileType string, tags map[string]string) (*ConfigurationProfile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[appID]; !ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if profileType == "" {
		profileType = "AWS.Freeform"
	}
	id := s.generateProfileID()
	profile := &ConfigurationProfile{
		ApplicationID: appID, ID: id, Name: name,
		Description: description, LocationURI: locationURI,
		Type: profileType, Tags: tags,
	}
	s.configProfiles[appID][id] = profile
	return profile, true
}

// GetConfigurationProfile returns a configuration profile.
func (s *Store) GetConfigurationProfile(appID, profileID string) (*ConfigurationProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles, ok := s.configProfiles[appID]
	if !ok {
		return nil, false
	}
	profile, ok := profiles[profileID]
	return profile, ok
}

// ListConfigurationProfiles returns all profiles for an application.
func (s *Store) ListConfigurationProfiles(appID string) ([]*ConfigurationProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles, ok := s.configProfiles[appID]
	if !ok {
		return nil, false
	}
	result := make([]*ConfigurationProfile, 0, len(profiles))
	for _, p := range profiles {
		result = append(result, p)
	}
	return result, true
}

// DeleteConfigurationProfile removes a configuration profile.
func (s *Store) DeleteConfigurationProfile(appID, profileID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, ok := s.configProfiles[appID]
	if !ok {
		return false
	}
	if _, ok := profiles[profileID]; !ok {
		return false
	}
	delete(profiles, profileID)
	return true
}

// CreateDeploymentStrategy creates a new deployment strategy.
func (s *Store) CreateDeploymentStrategy(name, description string, durationMinutes int, growthFactor float64, growthType string, bakeTimeMinutes int, replicateTo string, tags map[string]string) *DeploymentStrategy {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	if growthType == "" {
		growthType = "LINEAR"
	}
	if replicateTo == "" {
		replicateTo = "NONE"
	}
	id := s.generateStrategyID()
	strategy := &DeploymentStrategy{
		ID: id, Name: name, Description: description,
		DeploymentDurationInMinutes: durationMinutes,
		GrowthFactor: growthFactor, GrowthType: growthType,
		FinalBakeTimeInMinutes: bakeTimeMinutes,
		ReplicateTo: replicateTo, Tags: tags,
	}
	s.deploymentStrategies[id] = strategy
	return strategy
}

// GetDeploymentStrategy returns a deployment strategy by ID.
func (s *Store) GetDeploymentStrategy(id string) (*DeploymentStrategy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	strategy, ok := s.deploymentStrategies[id]
	return strategy, ok
}

// ListDeploymentStrategies returns all deployment strategies.
func (s *Store) ListDeploymentStrategies() []*DeploymentStrategy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DeploymentStrategy, 0, len(s.deploymentStrategies))
	for _, ds := range s.deploymentStrategies {
		result = append(result, ds)
	}
	return result
}

// DeleteDeploymentStrategy removes a deployment strategy.
func (s *Store) DeleteDeploymentStrategy(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.deploymentStrategies[id]; !ok {
		return false
	}
	delete(s.deploymentStrategies, id)
	return true
}

// StartDeployment starts a new deployment.
func (s *Store) StartDeployment(appID, envID, strategyID, profileID, configVersion, description string, tags map[string]string) (*Deployment, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.applications[appID]; !ok {
		return nil, "application"
	}
	envs, ok := s.environments[appID]
	if !ok {
		return nil, "environment"
	}
	if _, ok := envs[envID]; !ok {
		return nil, "environment"
	}
	if _, ok := s.deploymentStrategies[strategyID]; !ok {
		return nil, "strategy"
	}
	profiles, ok := s.configProfiles[appID]
	if !ok {
		return nil, "profile"
	}
	if _, ok := profiles[profileID]; !ok {
		return nil, "profile"
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	key := appID + ":" + envID
	s.nextDeployNum[key]++
	num := s.nextDeployNum[key]

	transitions := []lifecycle.Transition{
		{From: "BAKING", To: "COMPLETE", Delay: 5 * time.Second},
	}
	lm := lifecycle.NewMachine("BAKING", transitions, s.lifecycleConfig)

	now := time.Now().UTC()
	dep := &Deployment{
		ApplicationID:          appID,
		EnvironmentID:          envID,
		DeploymentStrategyID:   strategyID,
		ConfigurationProfileID: profileID,
		ConfigurationVersion:   configVersion,
		DeploymentNumber:       num,
		State:                  "BAKING",
		Description:            description,
		StartedAt:              now,
		PercentageComplete:     100.0,
		Lifecycle:              lm,
		Tags:                   tags,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		dep.State = string(to)
		if string(to) == "COMPLETE" {
			t := time.Now().UTC()
			dep.CompletedAt = &t
		}
	})

	if s.deployments[appID] == nil {
		s.deployments[appID] = make(map[string][]*Deployment)
	}
	s.deployments[appID][envID] = append(s.deployments[appID][envID], dep)
	return dep, ""
}

// GetDeployment returns a deployment by number.
func (s *Store) GetDeployment(appID, envID string, deploymentNumber int) (*Deployment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deps, ok := s.deployments[appID]
	if !ok {
		return nil, false
	}
	envDeps, ok := deps[envID]
	if !ok {
		return nil, false
	}
	for _, dep := range envDeps {
		if dep.DeploymentNumber == deploymentNumber {
			dep.State = string(dep.Lifecycle.State())
			return dep, true
		}
	}
	return nil, false
}

// ListDeployments returns all deployments for an app/env.
func (s *Store) ListDeployments(appID, envID string) ([]*Deployment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deps, ok := s.deployments[appID]
	if !ok {
		return nil, false
	}
	envDeps, ok := deps[envID]
	if !ok {
		return nil, false
	}
	for _, dep := range envDeps {
		dep.State = string(dep.Lifecycle.State())
	}
	result := make([]*Deployment, len(envDeps))
	copy(result, envDeps)
	return result, true
}

// StopDeployment stops a deployment.
func (s *Store) StopDeployment(appID, envID string, deploymentNumber int) (*Deployment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deps, ok := s.deployments[appID]
	if !ok {
		return nil, false
	}
	envDeps, ok := deps[envID]
	if !ok {
		return nil, false
	}
	for _, dep := range envDeps {
		if dep.DeploymentNumber == deploymentNumber {
			dep.Lifecycle.Stop()
			dep.Lifecycle.ForceState("ROLLED_BACK")
			dep.State = "ROLLED_BACK"
			t := time.Now().UTC()
			dep.CompletedAt = &t
			return dep, true
		}
	}
	return nil, false
}

// TagResource applies tags to a resource by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findResourceByARN(arn); r != nil {
		for k, v := range tags {
			r[k] = v
		}
		return true
	}
	return false
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findResourceByARN(arn); r != nil {
		for _, k := range keys {
			delete(r, k)
		}
		return true
	}
	return false
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r := s.findResourceByARN(arn); r != nil {
		cp := make(map[string]string, len(r))
		for k, v := range r {
			cp[k] = v
		}
		return cp, true
	}
	return nil, false
}

// findResourceByARN returns a pointer to the tags map for a resource. Caller must hold lock.
func (s *Store) findResourceByARN(arn string) map[string]string {
	for _, app := range s.applications {
		if s.appARN(app.ID) == arn {
			return app.Tags
		}
	}
	for appID, envs := range s.environments {
		for _, env := range envs {
			if s.envARN(appID, env.ID) == arn {
				return env.Tags
			}
		}
	}
	for appID, profiles := range s.configProfiles {
		for _, p := range profiles {
			if s.profileARN(appID, p.ID) == arn {
				return p.Tags
			}
		}
	}
	for _, ds := range s.deploymentStrategies {
		if s.strategyARN(ds.ID) == arn {
			return ds.Tags
		}
	}
	return nil
}
