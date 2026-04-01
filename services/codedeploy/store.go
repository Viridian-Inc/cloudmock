package codedeploy

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Deployment status constants.
const (
	DeployStatusCreated    = "Created"
	DeployStatusInProgress = "InProgress"
	DeployStatusSucceeded  = "Succeeded"
	DeployStatusFailed     = "Failed"
	DeployStatusStopped    = "Stopped"
)

// Application represents a CodeDeploy application.
type Application struct {
	ID         string
	Name       string
	ARN        string
	ComputePlatform string
	CreatedAt  time.Time
	Tags       map[string]string
}

// DeploymentGroup represents a CodeDeploy deployment group.
type DeploymentGroup struct {
	ID                    string
	Name                  string
	ARN                   string
	ApplicationName       string
	DeploymentConfigName  string
	ServiceRoleARN        string
	AutoScalingGroups     []string
	Ec2TagFilters         []EC2TagFilter
	DeploymentStyle       DeploymentStyle
	CreatedAt             time.Time
	Tags                  map[string]string
}

// EC2TagFilter is a tag filter for EC2 instances.
type EC2TagFilter struct {
	Key   string
	Value string
	Type  string
}

// DeploymentStyle describes the deployment strategy.
type DeploymentStyle struct {
	DeploymentType   string
	DeploymentOption string
}

// Deployment represents a single deployment.
type Deployment struct {
	ID                   string
	ARN                  string
	ApplicationName      string
	DeploymentGroupName  string
	DeploymentConfigName string
	Revision             RevisionLocation
	Status               string
	Description          string
	Creator              string
	CreateTime           time.Time
	StartTime            *time.Time
	CompleteTime         *time.Time
	Targets              []*InstanceTarget
	lifecycle            *lifecycle.Machine
}

// RevisionLocation describes the deployment revision source.
type RevisionLocation struct {
	RevisionType string
	S3Location   *S3Location
	GitHubLocation *GitHubLocation
}

// S3Location describes an S3 revision location.
type S3Location struct {
	Bucket     string
	Key        string
	BundleType string
	Version    string
}

// GitHubLocation describes a GitHub revision location.
type GitHubLocation struct {
	Repository string
	CommitID   string
}

// InstanceDeploymentStatus constants for per-instance lifecycle.
const (
	InstanceStatusPending    = "Pending"
	InstanceStatusInProgress = "InProgress"
	InstanceStatusSucceeded  = "Succeeded"
	InstanceStatusFailed     = "Failed"
	InstanceStatusSkipped    = "Skipped"
)

// DeploymentTarget represents a deployment target instance.
type DeploymentTarget struct {
	DeploymentTargetType string
	InstanceTarget       *InstanceTarget
}

// InstanceTarget describes a deployment to an EC2 instance.
type InstanceTarget struct {
	DeploymentID       string
	TargetID           string
	TargetARN          string
	Status             string
	LastUpdatedAt      time.Time
	LifecycleEvents    []LifecycleEvent
}

// LifecycleEvent represents a deployment lifecycle hook event.
type LifecycleEvent struct {
	LifecycleEventName string
	Status             string
	StartTime          time.Time
	EndTime            *time.Time
}

// Store is the in-memory store for all CodeDeploy resources.
type Store struct {
	mu               sync.RWMutex
	accountID        string
	region           string
	applications     map[string]*Application
	deploymentGroups map[string]map[string]*DeploymentGroup // appName -> groupName -> group
	deployments      map[string]*Deployment                 // deploymentID -> deployment
	tags             map[string]map[string]string
	lcConfig         *lifecycle.Config
}

// NewStore creates an empty CodeDeploy store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:        accountID,
		region:           region,
		applications:     make(map[string]*Application),
		deploymentGroups: make(map[string]map[string]*DeploymentGroup),
		deployments:      make(map[string]*Deployment),
		tags:             make(map[string]map[string]string),
		lcConfig:         lifecycle.DefaultConfig(),
	}
}

// ---- ARN builders ----

func (s *Store) applicationARN(name string) string {
	return fmt.Sprintf("arn:aws:codedeploy:%s:%s:application:%s", s.region, s.accountID, name)
}

func (s *Store) deploymentGroupARN(appName, groupName string) string {
	return fmt.Sprintf("arn:aws:codedeploy:%s:%s:deploymentgroup:%s/%s", s.region, s.accountID, appName, groupName)
}

func (s *Store) deploymentARN(deploymentID string) string {
	return fmt.Sprintf("arn:aws:codedeploy:%s:%s:deployment/%s", s.region, s.accountID, deploymentID)
}

// ---- Application operations ----

func (s *Store) CreateApplication(name, computePlatform string, tags map[string]string) (*Application, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Application name is required.")
	}
	if _, exists := s.applications[name]; exists {
		return nil, service.NewAWSError("ApplicationAlreadyExistsException",
			fmt.Sprintf("Application already exists: %s", name), http.StatusConflict)
	}

	if computePlatform == "" {
		computePlatform = "Server"
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	app := &Application{
		ID:              newUUID(),
		Name:            name,
		ARN:             s.applicationARN(name),
		ComputePlatform: computePlatform,
		CreatedAt:       time.Now().UTC(),
		Tags:            tags,
	}
	s.applications[name] = app
	return app, nil
}

func (s *Store) GetApplication(name string) (*Application, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.applications[name]
	if !ok {
		return nil, service.NewAWSError("ApplicationDoesNotExistException",
			fmt.Sprintf("Application does not exist: %s", name), http.StatusNotFound)
	}
	return app, nil
}

func (s *Store) ListApplications() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.applications))
	for name := range s.applications {
		names = append(names, name)
	}
	return names
}

func (s *Store) DeleteApplication(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.applications[name]; !ok {
		return service.NewAWSError("ApplicationDoesNotExistException",
			fmt.Sprintf("Application does not exist: %s", name), http.StatusNotFound)
	}
	delete(s.applications, name)
	delete(s.deploymentGroups, name)
	return nil
}

// ---- Deployment Group operations ----

func (s *Store) CreateDeploymentGroup(appName string, group *DeploymentGroup) (*DeploymentGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.applications[appName]; !ok {
		return nil, service.NewAWSError("ApplicationDoesNotExistException",
			fmt.Sprintf("Application does not exist: %s", appName), http.StatusNotFound)
	}

	if group.Name == "" {
		return nil, service.ErrValidation("Deployment group name is required.")
	}

	if s.deploymentGroups[appName] == nil {
		s.deploymentGroups[appName] = make(map[string]*DeploymentGroup)
	}

	if _, exists := s.deploymentGroups[appName][group.Name]; exists {
		return nil, service.NewAWSError("DeploymentGroupAlreadyExistsException",
			fmt.Sprintf("Deployment group already exists: %s", group.Name), http.StatusConflict)
	}

	group.ID = newUUID()
	group.ARN = s.deploymentGroupARN(appName, group.Name)
	group.ApplicationName = appName
	group.CreatedAt = time.Now().UTC()
	if group.Tags == nil {
		group.Tags = make(map[string]string)
	}
	if group.DeploymentConfigName == "" {
		group.DeploymentConfigName = "CodeDeployDefault.OneAtATime"
	}

	s.deploymentGroups[appName][group.Name] = group
	return group, nil
}

func (s *Store) GetDeploymentGroup(appName, groupName string) (*DeploymentGroup, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.applications[appName]; !ok {
		return nil, service.NewAWSError("ApplicationDoesNotExistException",
			fmt.Sprintf("Application does not exist: %s", appName), http.StatusNotFound)
	}

	groups := s.deploymentGroups[appName]
	if groups == nil {
		return nil, service.NewAWSError("DeploymentGroupDoesNotExistException",
			fmt.Sprintf("Deployment group does not exist: %s", groupName), http.StatusNotFound)
	}

	group, ok := groups[groupName]
	if !ok {
		return nil, service.NewAWSError("DeploymentGroupDoesNotExistException",
			fmt.Sprintf("Deployment group does not exist: %s", groupName), http.StatusNotFound)
	}
	return group, nil
}

func (s *Store) ListDeploymentGroups(appName string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.applications[appName]; !ok {
		return nil, service.NewAWSError("ApplicationDoesNotExistException",
			fmt.Sprintf("Application does not exist: %s", appName), http.StatusNotFound)
	}

	groups := s.deploymentGroups[appName]
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	return names, nil
}

func (s *Store) DeleteDeploymentGroup(appName, groupName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.deploymentGroups[appName]
	if groups == nil {
		return nil
	}
	delete(groups, groupName)
	return nil
}

func (s *Store) UpdateDeploymentGroup(appName, groupName string, updates map[string]any) (*DeploymentGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.deploymentGroups[appName]
	if groups == nil {
		return nil, service.NewAWSError("DeploymentGroupDoesNotExistException",
			fmt.Sprintf("Deployment group does not exist: %s", groupName), http.StatusNotFound)
	}

	group, ok := groups[groupName]
	if !ok {
		return nil, service.NewAWSError("DeploymentGroupDoesNotExistException",
			fmt.Sprintf("Deployment group does not exist: %s", groupName), http.StatusNotFound)
	}

	if role, ok := updates["serviceRoleArn"].(string); ok && role != "" {
		group.ServiceRoleARN = role
	}
	if cfg, ok := updates["deploymentConfigName"].(string); ok && cfg != "" {
		group.DeploymentConfigName = cfg
	}
	if newName, ok := updates["newDeploymentGroupName"].(string); ok && newName != "" {
		delete(groups, groupName)
		group.Name = newName
		group.ARN = s.deploymentGroupARN(appName, newName)
		groups[newName] = group
	}

	return group, nil
}

// ---- Deployment operations ----

func (s *Store) CreateDeployment(appName, groupName, configName, description string, revision RevisionLocation) (*Deployment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.applications[appName]; !ok {
		return nil, service.NewAWSError("ApplicationDoesNotExistException",
			fmt.Sprintf("Application does not exist: %s", appName), http.StatusNotFound)
	}

	if groupName != "" {
		groups := s.deploymentGroups[appName]
		if groups == nil || groups[groupName] == nil {
			return nil, service.NewAWSError("DeploymentGroupDoesNotExistException",
				fmt.Sprintf("Deployment group does not exist: %s", groupName), http.StatusNotFound)
		}
	}

	if configName == "" {
		configName = "CodeDeployDefault.OneAtATime"
	}

	deployID := newUUID()
	now := time.Now().UTC()

	// Build initial targets from deployment group info
	var targets []*InstanceTarget
	if groupName != "" {
		if groups, ok := s.deploymentGroups[appName]; ok {
			if group, ok := groups[groupName]; ok {
				// Add auto-scaling group instances as targets
				for _, asgName := range group.AutoScalingGroups {
					targets = append(targets, &InstanceTarget{
						DeploymentID:  deployID,
						TargetID:      asgName,
						TargetARN:     fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", s.region, s.accountID, asgName),
						Status:        InstanceStatusPending,
						LastUpdatedAt: now,
						LifecycleEvents: []LifecycleEvent{
							{LifecycleEventName: "BeforeInstall", Status: InstanceStatusPending, StartTime: now},
							{LifecycleEventName: "AfterInstall", Status: InstanceStatusPending, StartTime: now},
							{LifecycleEventName: "ApplicationStart", Status: InstanceStatusPending, StartTime: now},
							{LifecycleEventName: "ValidateService", Status: InstanceStatusPending, StartTime: now},
						},
					})
				}
			}
		}
	}

	d := &Deployment{
		ID:                   deployID,
		ARN:                  s.deploymentARN(deployID),
		ApplicationName:      appName,
		DeploymentGroupName:  groupName,
		DeploymentConfigName: configName,
		Revision:             revision,
		Status:               DeployStatusCreated,
		Description:          description,
		Creator:              "user",
		CreateTime:           now,
		Targets:              targets,
	}

	// Set up lifecycle: Created -> InProgress -> Succeeded
	transitions := []lifecycle.Transition{
		{From: lifecycle.State(DeployStatusCreated), To: lifecycle.State(DeployStatusInProgress), Delay: 2 * time.Second},
		{From: lifecycle.State(DeployStatusInProgress), To: lifecycle.State(DeployStatusSucceeded), Delay: 5 * time.Second},
	}
	d.lifecycle = lifecycle.NewMachine(lifecycle.State(DeployStatusCreated), transitions, s.lcConfig)
	d.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		d.Status = string(to)
		if to == lifecycle.State(DeployStatusInProgress) {
			now := time.Now().UTC()
			d.StartTime = &now
			// Move targets to InProgress
			for _, t := range d.Targets {
				t.Status = InstanceStatusInProgress
				t.LastUpdatedAt = now
			}
		}
		if to == lifecycle.State(DeployStatusSucceeded) {
			now := time.Now().UTC()
			d.CompleteTime = &now
			for _, t := range d.Targets {
				t.Status = InstanceStatusSucceeded
				t.LastUpdatedAt = now
				for i := range t.LifecycleEvents {
					t.LifecycleEvents[i].Status = InstanceStatusSucceeded
					endTime := now
					t.LifecycleEvents[i].EndTime = &endTime
				}
			}
		}
		if to == lifecycle.State(DeployStatusFailed) {
			now := time.Now().UTC()
			d.CompleteTime = &now
			d.Status = "ROLLED_BACK"
			for _, t := range d.Targets {
				t.Status = InstanceStatusFailed
				t.LastUpdatedAt = now
			}
		}
	})

	s.deployments[deployID] = d
	return d, nil
}

func (s *Store) GetDeployment(deploymentID string) (*Deployment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.deployments[deploymentID]
	if !ok {
		return nil, service.NewAWSError("DeploymentDoesNotExistException",
			fmt.Sprintf("Deployment does not exist: %s", deploymentID), http.StatusNotFound)
	}
	if d.lifecycle != nil {
		d.Status = string(d.lifecycle.State())
	}
	return d, nil
}

func (s *Store) ListDeployments(appName, groupName, status string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ids []string
	for id, d := range s.deployments {
		if d.lifecycle != nil {
			d.Status = string(d.lifecycle.State())
		}
		if appName != "" && d.ApplicationName != appName {
			continue
		}
		if groupName != "" && d.DeploymentGroupName != groupName {
			continue
		}
		if status != "" && d.Status != status {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (s *Store) BatchGetDeployments(ids []string) ([]*Deployment, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Deployment
	for _, id := range ids {
		if d, ok := s.deployments[id]; ok {
			if d.lifecycle != nil {
				d.Status = string(d.lifecycle.State())
			}
			result = append(result, d)
		}
	}
	return result, nil
}

func (s *Store) StopDeployment(deploymentID string) (*Deployment, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.deployments[deploymentID]
	if !ok {
		return nil, service.NewAWSError("DeploymentDoesNotExistException",
			fmt.Sprintf("Deployment does not exist: %s", deploymentID), http.StatusNotFound)
	}

	if d.lifecycle != nil {
		d.Status = string(d.lifecycle.State())
	}

	if d.Status == DeployStatusSucceeded || d.Status == DeployStatusFailed || d.Status == DeployStatusStopped {
		return nil, service.NewAWSError("DeploymentAlreadyCompletedException",
			"Deployment is already complete.", http.StatusBadRequest)
	}

	if d.lifecycle != nil {
		d.lifecycle.Stop()
	}
	d.Status = DeployStatusStopped
	now := time.Now().UTC()
	d.CompleteTime = &now
	return d, nil
}

func (s *Store) BatchGetDeploymentTargets(deploymentID string, targetIDs []string) ([]*DeploymentTarget, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.deployments[deploymentID]
	if !ok {
		return nil, service.NewAWSError("DeploymentDoesNotExistException",
			fmt.Sprintf("Deployment does not exist: %s", deploymentID), http.StatusNotFound)
	}

	if d.lifecycle != nil {
		d.Status = string(d.lifecycle.State())
	}

	// Build a map of tracked targets for quick lookup
	trackedTargets := make(map[string]*InstanceTarget, len(d.Targets))
	for _, t := range d.Targets {
		trackedTargets[t.TargetID] = t
	}

	var targets []*DeploymentTarget
	for _, tid := range targetIDs {
		if tracked, ok := trackedTargets[tid]; ok {
			targets = append(targets, &DeploymentTarget{
				DeploymentTargetType: "InstanceTarget",
				InstanceTarget:       tracked,
			})
		} else {
			// Fallback: synthetic target using deployment status
			targets = append(targets, &DeploymentTarget{
				DeploymentTargetType: "InstanceTarget",
				InstanceTarget: &InstanceTarget{
					DeploymentID:  d.ID,
					TargetID:      tid,
					TargetARN:     fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", s.region, s.accountID, tid),
					Status:        d.Status,
					LastUpdatedAt: time.Now().UTC(),
				},
			})
		}
	}
	return targets, nil
}

// ---- Tag operations ----

func (s *Store) AddTagsToOnPremisesInstances(instanceNames []string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, name := range instanceNames {
		key := "onprem:" + name
		if s.tags[key] == nil {
			s.tags[key] = make(map[string]string)
		}
		for k, v := range tags {
			s.tags[key][k] = v
		}
	}
	return nil
}

func (s *Store) RemoveTagsFromOnPremisesInstances(instanceNames []string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, name := range instanceNames {
		key := "onprem:" + name
		m := s.tags[key]
		if m == nil {
			continue
		}
		for _, k := range tagKeys {
			delete(m, k)
		}
	}
	return nil
}
