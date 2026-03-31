package codebuild

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

// Build status constants.
const (
	BuildStatusSubmitted  = "SUBMITTED"
	BuildStatusInProgress = "IN_PROGRESS"
	BuildStatusSucceeded  = "SUCCEEDED"
	BuildStatusFailed     = "FAILED"
	BuildStatusStopped    = "STOPPED"
)

// Project represents a CodeBuild project.
type Project struct {
	Name          string
	ARN           string
	Description   string
	Source        ProjectSource
	Artifacts     ProjectArtifacts
	Environment   ProjectEnvironment
	ServiceRole   string
	TimeoutInMins int
	Tags          map[string]string
	CreatedAt     time.Time
	LastModified  time.Time
}

// ProjectSource describes the build source.
type ProjectSource struct {
	Type     string
	Location string
}

// ProjectArtifacts describes the build output.
type ProjectArtifacts struct {
	Type     string
	Location string
}

// ProjectEnvironment describes the build environment.
type ProjectEnvironment struct {
	Type           string
	Image          string
	ComputeType    string
	EnvironmentVars []EnvironmentVariable
}

// EnvironmentVariable is a name/value pair for build environments.
type EnvironmentVariable struct {
	Name  string
	Value string
	Type  string
}

// Build represents a single build execution.
type Build struct {
	ID              string
	ARN             string
	ProjectName     string
	BuildNumber     int64
	BuildStatus     string
	CurrentPhase    string
	StartTime       time.Time
	EndTime         *time.Time
	Source          ProjectSource
	Artifacts       ProjectArtifacts
	Environment     ProjectEnvironment
	ServiceRole     string
	TimeoutInMins   int
	Logs            BuildLogs
	lifecycle       *lifecycle.Machine
}

// BuildLogs holds build log references.
type BuildLogs struct {
	GroupName  string
	StreamName string
	DeepLink   string
}

// ReportGroup represents a CodeBuild report group.
type ReportGroup struct {
	ARN        string
	Name       string
	Type       string
	ExportConfig ReportExportConfig
	CreatedAt  time.Time
	Tags       map[string]string
}

// ReportExportConfig describes where reports are exported.
type ReportExportConfig struct {
	ExportConfigType string
	S3Destination    *S3ReportExportConfig
}

// S3ReportExportConfig describes S3 export configuration.
type S3ReportExportConfig struct {
	Bucket string
	Path   string
}

// Store is the in-memory store for all CodeBuild resources.
type Store struct {
	mu           sync.RWMutex
	accountID    string
	region       string
	projects     map[string]*Project
	builds       map[string]*Build            // buildID -> Build
	projectBuilds map[string][]string          // projectName -> []buildID
	buildCounter map[string]int64             // projectName -> counter
	reportGroups map[string]*ReportGroup      // name -> ReportGroup
	tags         map[string]map[string]string // resourceARN -> tags
	lcConfig     *lifecycle.Config
}

// NewStore creates an empty CodeBuild store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:     accountID,
		region:        region,
		projects:      make(map[string]*Project),
		builds:        make(map[string]*Build),
		projectBuilds: make(map[string][]string),
		buildCounter:  make(map[string]int64),
		reportGroups:  make(map[string]*ReportGroup),
		tags:          make(map[string]map[string]string),
		lcConfig:      lifecycle.DefaultConfig(),
	}
}

// ---- ARN builders ----

func (s *Store) projectARN(name string) string {
	return fmt.Sprintf("arn:aws:codebuild:%s:%s:project/%s", s.region, s.accountID, name)
}

func (s *Store) buildARN(projectName, buildID string) string {
	return fmt.Sprintf("arn:aws:codebuild:%s:%s:build/%s:%s", s.region, s.accountID, projectName, buildID)
}

func (s *Store) reportGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:codebuild:%s:%s:report-group/%s", s.region, s.accountID, name)
}

// ---- Project operations ----

func (s *Store) CreateProject(p *Project) (*Project, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.Name == "" {
		return nil, service.ErrValidation("Project name is required.")
	}
	if _, exists := s.projects[p.Name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Project already exists: %s", p.Name), http.StatusConflict)
	}

	now := time.Now().UTC()
	p.ARN = s.projectARN(p.Name)
	p.CreatedAt = now
	p.LastModified = now
	if p.TimeoutInMins == 0 {
		p.TimeoutInMins = 60
	}
	if p.Tags == nil {
		p.Tags = make(map[string]string)
	}
	s.projects[p.Name] = p
	return p, nil
}

func (s *Store) GetProject(name string) (*Project, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.projects[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Project not found: %s", name), http.StatusNotFound)
	}
	return p, nil
}

func (s *Store) BatchGetProjects(names []string) ([]*Project, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found []*Project
	var notFound []string
	for _, name := range names {
		if p, ok := s.projects[name]; ok {
			found = append(found, p)
		} else {
			notFound = append(notFound, name)
		}
	}
	return found, notFound
}

func (s *Store) ListProjects() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.projects))
	for name := range s.projects {
		names = append(names, name)
	}
	return names
}

func (s *Store) UpdateProject(name string, updates map[string]any) (*Project, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.projects[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Project not found: %s", name), http.StatusNotFound)
	}

	if desc, ok := updates["description"].(string); ok {
		p.Description = desc
	}
	if role, ok := updates["serviceRole"].(string); ok {
		p.ServiceRole = role
	}
	if timeout, ok := updates["timeoutInMinutes"].(float64); ok {
		p.TimeoutInMins = int(timeout)
	}
	p.LastModified = time.Now().UTC()
	return p, nil
}

func (s *Store) DeleteProject(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.projects[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Project not found: %s", name), http.StatusNotFound)
	}
	delete(s.projects, name)
	return nil
}

// ---- Build operations ----

func (s *Store) StartBuild(projectName string, envOverrides *ProjectEnvironment) (*Build, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.projects[projectName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Project not found: %s", projectName), http.StatusNotFound)
	}

	s.buildCounter[projectName]++
	buildNum := s.buildCounter[projectName]
	buildID := newUUID()
	now := time.Now().UTC()

	env := p.Environment
	if envOverrides != nil {
		if envOverrides.Image != "" {
			env.Image = envOverrides.Image
		}
		if envOverrides.ComputeType != "" {
			env.ComputeType = envOverrides.ComputeType
		}
	}

	b := &Build{
		ID:            buildID,
		ARN:           s.buildARN(projectName, buildID),
		ProjectName:   projectName,
		BuildNumber:   buildNum,
		BuildStatus:   BuildStatusSubmitted,
		CurrentPhase:  "SUBMITTED",
		StartTime:     now,
		Source:        p.Source,
		Artifacts:     p.Artifacts,
		Environment:   env,
		ServiceRole:   p.ServiceRole,
		TimeoutInMins: p.TimeoutInMins,
		Logs: BuildLogs{
			GroupName:  fmt.Sprintf("/aws/codebuild/%s", projectName),
			StreamName: buildID,
		},
	}

	// Set up lifecycle: SUBMITTED -> IN_PROGRESS -> SUCCEEDED
	transitions := []lifecycle.Transition{
		{From: lifecycle.State(BuildStatusSubmitted), To: lifecycle.State(BuildStatusInProgress), Delay: 2 * time.Second},
		{From: lifecycle.State(BuildStatusInProgress), To: lifecycle.State(BuildStatusSucceeded), Delay: 5 * time.Second},
	}
	b.lifecycle = lifecycle.NewMachine(lifecycle.State(BuildStatusSubmitted), transitions, s.lcConfig)
	b.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		b.BuildStatus = string(to)
		if to == lifecycle.State(BuildStatusInProgress) {
			b.CurrentPhase = "BUILD"
		}
		if to == lifecycle.State(BuildStatusSucceeded) || to == lifecycle.State(BuildStatusFailed) {
			b.CurrentPhase = "COMPLETED"
			now := time.Now().UTC()
			b.EndTime = &now
		}
	})

	s.builds[buildID] = b
	s.projectBuilds[projectName] = append(s.projectBuilds[projectName], buildID)
	return b, nil
}

func (s *Store) GetBuild(buildID string) (*Build, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.builds[buildID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Build not found: %s", buildID), http.StatusNotFound)
	}
	// Refresh status from lifecycle
	if b.lifecycle != nil {
		b.BuildStatus = string(b.lifecycle.State())
	}
	return b, nil
}

func (s *Store) BatchGetBuilds(ids []string) ([]*Build, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found []*Build
	var notFound []string
	for _, id := range ids {
		if b, ok := s.builds[id]; ok {
			if b.lifecycle != nil {
				b.BuildStatus = string(b.lifecycle.State())
			}
			found = append(found, b)
		} else {
			notFound = append(notFound, id)
		}
	}
	return found, notFound
}

func (s *Store) ListBuildsForProject(projectName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.projectBuilds[projectName]
	if ids == nil {
		return []string{}
	}
	// Return in reverse order (newest first)
	result := make([]string, len(ids))
	for i, id := range ids {
		result[len(ids)-1-i] = id
	}
	return result
}

func (s *Store) StopBuild(buildID string) (*Build, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.builds[buildID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Build not found: %s", buildID), http.StatusNotFound)
	}

	if b.BuildStatus == BuildStatusSucceeded || b.BuildStatus == BuildStatusFailed || b.BuildStatus == BuildStatusStopped {
		return nil, service.NewAWSError("InvalidInputException",
			"Build is already complete.", http.StatusBadRequest)
	}

	if b.lifecycle != nil {
		b.lifecycle.Stop()
	}
	b.BuildStatus = BuildStatusStopped
	b.CurrentPhase = "COMPLETED"
	now := time.Now().UTC()
	b.EndTime = &now
	return b, nil
}

// ---- Report Group operations ----

func (s *Store) CreateReportGroup(name, reportType string, exportConfig ReportExportConfig, tags map[string]string) (*ReportGroup, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Report group name is required.")
	}
	if _, exists := s.reportGroups[name]; exists {
		return nil, service.NewAWSError("ResourceAlreadyExistsException",
			fmt.Sprintf("Report group already exists: %s", name), http.StatusConflict)
	}

	if tags == nil {
		tags = make(map[string]string)
	}
	rg := &ReportGroup{
		ARN:          s.reportGroupARN(name),
		Name:         name,
		Type:         reportType,
		ExportConfig: exportConfig,
		CreatedAt:    time.Now().UTC(),
		Tags:         tags,
	}
	s.reportGroups[name] = rg
	return rg, nil
}

func (s *Store) BatchGetReportGroups(names []string) ([]*ReportGroup, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found []*ReportGroup
	var notFound []string
	for _, name := range names {
		if rg, ok := s.reportGroups[name]; ok {
			found = append(found, rg)
		} else {
			notFound = append(notFound, name)
		}
	}
	return found, notFound
}

func (s *Store) ListReportGroups() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	arns := make([]string, 0, len(s.reportGroups))
	for _, rg := range s.reportGroups {
		arns = append(arns, rg.ARN)
	}
	return arns
}

func (s *Store) DeleteReportGroup(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.reportGroups[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Report group not found: %s", name), http.StatusNotFound)
	}
	delete(s.reportGroups, name)
	return nil
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, keys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.tags[arn]
	if m == nil {
		return nil
	}
	for _, k := range keys {
		delete(m, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m := s.tags[arn]
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
