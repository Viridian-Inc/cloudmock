package fis

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// ExperimentTemplate represents an FIS experiment template.
type ExperimentTemplate struct {
	ID          string
	Description string
	Targets     map[string]ExperimentTarget
	Actions     map[string]ExperimentAction
	StopConditions []StopCondition
	RoleArn     string
	Tags        map[string]string
	CreationTime time.Time
	LastUpdateTime time.Time
}

// ExperimentTarget defines targets for an experiment.
type ExperimentTarget struct {
	ResourceType  string
	ResourceArns  []string
	SelectionMode string
}

// ExperimentAction defines an action in an experiment.
type ExperimentAction struct {
	ActionID    string
	Description string
	Parameters  map[string]string
	Targets     map[string]string
	StartAfter  []string
}

// StopCondition defines when to stop an experiment.
type StopCondition struct {
	Source string
	Value  string
}

// Experiment represents a running or completed experiment.
type Experiment struct {
	ID             string
	TemplateID     string
	RoleArn        string
	State          string
	StateReason    string
	Tags           map[string]string
	CreationTime   time.Time
	StartTime      *time.Time
	EndTime        *time.Time
	lifecycle      *lifecycle.Machine
}

// TargetAccountConfiguration represents a target account config.
type TargetAccountConfiguration struct {
	AccountID   string
	RoleArn     string
	Description string
}

// Store manages FIS resources in memory.
type Store struct {
	mu               sync.RWMutex
	templates        map[string]*ExperimentTemplate
	experiments      map[string]*Experiment
	targetAccounts   map[string]*TargetAccountConfiguration
	accountID        string
	region           string
	lcConfig         *lifecycle.Config
	templateSeq      int
	experimentSeq    int
}

// NewStore returns a new empty FIS Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		templates:      make(map[string]*ExperimentTemplate),
		experiments:    make(map[string]*Experiment),
		targetAccounts: make(map[string]*TargetAccountConfiguration),
		accountID:      accountID,
		region:         region,
		lcConfig:       lifecycle.DefaultConfig(),
	}
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:fis:%s:%s:", s.region, s.accountID)
}

// CreateExperimentTemplate creates a new experiment template.
func (s *Store) CreateExperimentTemplate(desc, roleArn string, targets map[string]ExperimentTarget, actions map[string]ExperimentAction, stopConditions []StopCondition, tags map[string]string) (*ExperimentTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.templateSeq++
	id := fmt.Sprintf("EXT%010d", s.templateSeq)
	now := time.Now().UTC()

	tmpl := &ExperimentTemplate{
		ID:             id,
		Description:    desc,
		Targets:        targets,
		Actions:        actions,
		StopConditions: stopConditions,
		RoleArn:        roleArn,
		Tags:           tags,
		CreationTime:   now,
		LastUpdateTime: now,
	}
	s.templates[id] = tmpl
	return tmpl, nil
}

// GetExperimentTemplate retrieves a template by ID.
func (s *Store) GetExperimentTemplate(id string) (*ExperimentTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[id]
	return t, ok
}

// ListExperimentTemplates returns all templates.
func (s *Store) ListExperimentTemplates() []*ExperimentTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ExperimentTemplate, 0, len(s.templates))
	for _, t := range s.templates {
		out = append(out, t)
	}
	return out
}

// DeleteExperimentTemplate removes a template.
func (s *Store) DeleteExperimentTemplate(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[id]; !ok {
		return false
	}
	delete(s.templates, id)
	return true
}

// StartExperiment starts a new experiment from a template.
func (s *Store) StartExperiment(templateID string, tags map[string]string) (*Experiment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, ok := s.templates[templateID]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	s.experimentSeq++
	id := fmt.Sprintf("EXP%010d", s.experimentSeq)

	transitions := []lifecycle.Transition{
		{From: "initiating", To: "running", Delay: 1 * time.Second},
		{From: "running", To: "completed", Delay: 5 * time.Second},
	}

	now := time.Now().UTC()
	exp := &Experiment{
		ID:           id,
		TemplateID:   templateID,
		RoleArn:      tmpl.RoleArn,
		State:        "initiating",
		StateReason:  "Experiment is initiating",
		Tags:         tags,
		CreationTime: now,
	}
	exp.lifecycle = lifecycle.NewMachine("initiating", transitions, s.lcConfig)
	exp.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		exp.State = string(to)
		now := time.Now().UTC()
		if to == "running" {
			exp.StartTime = &now
			exp.StateReason = "Experiment is running"
		} else if to == "completed" {
			exp.EndTime = &now
			exp.StateReason = "Experiment completed"
		}
	})

	s.experiments[id] = exp
	return exp, nil
}

// GetExperiment retrieves an experiment by ID.
func (s *Store) GetExperiment(id string) (*Experiment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.experiments[id]
	return e, ok
}

// ListExperiments returns all experiments.
func (s *Store) ListExperiments() []*Experiment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Experiment, 0, len(s.experiments))
	for _, e := range s.experiments {
		out = append(out, e)
	}
	return out
}

// StopExperiment stops a running experiment.
func (s *Store) StopExperiment(id string) (*Experiment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.experiments[id]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", id)
	}
	if exp.lifecycle != nil {
		exp.lifecycle.Stop()
	}
	exp.State = "stopped"
	exp.StateReason = "Experiment was stopped"
	now := time.Now().UTC()
	exp.EndTime = &now
	return exp, nil
}

// CreateTargetAccountConfiguration creates or updates a target account config.
func (s *Store) CreateTargetAccountConfiguration(accountID, roleArn, description string) *TargetAccountConfiguration {
	s.mu.Lock()
	defer s.mu.Unlock()
	tac := &TargetAccountConfiguration{
		AccountID:   accountID,
		RoleArn:     roleArn,
		Description: description,
	}
	s.targetAccounts[accountID] = tac
	return tac
}

// GetTargetAccountConfiguration retrieves a target account config.
func (s *Store) GetTargetAccountConfiguration(accountID string) (*TargetAccountConfiguration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tac, ok := s.targetAccounts[accountID]
	return tac, ok
}

// ListTargetAccountConfigurations returns all target account configs.
func (s *Store) ListTargetAccountConfigurations() []*TargetAccountConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TargetAccountConfiguration, 0, len(s.targetAccounts))
	for _, tac := range s.targetAccounts {
		out = append(out, tac)
	}
	return out
}

// DeleteTargetAccountConfiguration removes a target account config.
func (s *Store) DeleteTargetAccountConfiguration(accountID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.targetAccounts[accountID]; !ok {
		return false
	}
	delete(s.targetAccounts, accountID)
	return true
}
