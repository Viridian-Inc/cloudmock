package codepipeline

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Pipeline execution status constants.
const (
	ExecStatusInProgress = "InProgress"
	ExecStatusSucceeded  = "Succeeded"
	ExecStatusFailed     = "Failed"
	ExecStatusStopping   = "Stopping"
	ExecStatusStopped    = "Stopped"
	ExecStatusSuperseded = "Superseded"
)

// Stage action status constants.
const (
	ActionStatusInProgress = "InProgress"
	ActionStatusSucceeded  = "Succeeded"
	ActionStatusFailed     = "Failed"
)

// Pipeline represents a CodePipeline pipeline.
type Pipeline struct {
	Name            string
	ARN             string
	RoleARN         string
	Version         int
	Stages          []StageDeclaration
	Tags            map[string]string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// StageDeclaration describes a pipeline stage.
type StageDeclaration struct {
	Name    string
	Actions []ActionDeclaration
}

// ActionDeclaration describes an action within a stage.
type ActionDeclaration struct {
	Name            string
	ActionTypeID    ActionTypeID
	Configuration   map[string]string
	InputArtifacts  []ArtifactRef
	OutputArtifacts []ArtifactRef
	RunOrder        int
}

// ActionTypeID identifies the type of an action.
type ActionTypeID struct {
	Category string
	Owner    string
	Provider string
	Version  string
}

// ArtifactRef is a named artifact reference.
type ArtifactRef struct {
	Name string
}

// PipelineExecution represents a single execution of a pipeline.
type PipelineExecution struct {
	ID             string
	PipelineName   string
	PipelineVersion int
	Status         string
	StatusSummary  string
	StartTime      time.Time
	EndTime        *time.Time
	StageStates    []StageState
	lifecycle      *lifecycle.Machine
}

// StageState tracks the state of a stage in an execution.
type StageState struct {
	StageName    string
	ActionStates []ActionState
	Status       string
}

// ActionState tracks the state of an action in a stage.
type ActionState struct {
	ActionName string
	Status     string
	Summary    string
	LastUpdate time.Time
}

// ApprovalResult holds the result of a manual approval action.
type ApprovalResult struct {
	Summary string
	Status  string // Approved | Rejected
}

// Webhook represents a CodePipeline webhook.
type Webhook struct {
	Name           string
	ARN            string
	PipelineName   string
	TargetAction   string
	TargetPipeline string
	Authentication string
	Filters        []WebhookFilter
	URL            string
	Tags           map[string]string
	CreatedAt      time.Time
}

// WebhookFilter defines a filter condition for a webhook.
type WebhookFilter struct {
	JSONPath    string
	MatchEquals string
}

// Store is the in-memory store for all CodePipeline resources.
type Store struct {
	mu         sync.RWMutex
	accountID  string
	region     string
	pipelines  map[string]*Pipeline
	executions map[string][]*PipelineExecution // pipelineName -> executions
	webhooks   map[string]*Webhook             // webhookName -> webhook
	tags       map[string]map[string]string
	lcConfig   *lifecycle.Config
}

// NewStore creates an empty CodePipeline store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:  accountID,
		region:     region,
		pipelines:  make(map[string]*Pipeline),
		executions: make(map[string][]*PipelineExecution),
		webhooks:   make(map[string]*Webhook),
		tags:       make(map[string]map[string]string),
		lcConfig:   lifecycle.DefaultConfig(),
	}
}

// ---- ARN builders ----

func (s *Store) pipelineARN(name string) string {
	return fmt.Sprintf("arn:aws:codepipeline:%s:%s:%s", s.region, s.accountID, name)
}

// ---- Pipeline operations ----

func (s *Store) CreatePipeline(p *Pipeline) (*Pipeline, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.Name == "" {
		return nil, service.ErrValidation("Pipeline name is required.")
	}
	if _, exists := s.pipelines[p.Name]; exists {
		return nil, service.NewAWSError("PipelineNameInUseException",
			fmt.Sprintf("Pipeline already exists: %s", p.Name), http.StatusConflict)
	}

	now := time.Now().UTC()
	p.ARN = s.pipelineARN(p.Name)
	p.Version = 1
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Tags == nil {
		p.Tags = make(map[string]string)
	}
	s.pipelines[p.Name] = p
	return p, nil
}

func (s *Store) GetPipeline(name string) (*Pipeline, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.pipelines[name]
	if !ok {
		return nil, service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", name), http.StatusNotFound)
	}
	return p, nil
}

func (s *Store) ListPipelines() []*Pipeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Pipeline, 0, len(s.pipelines))
	for _, p := range s.pipelines {
		result = append(result, p)
	}
	return result
}

func (s *Store) UpdatePipeline(p *Pipeline) (*Pipeline, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.pipelines[p.Name]
	if !ok {
		return nil, service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", p.Name), http.StatusNotFound)
	}

	existing.Stages = p.Stages
	if p.RoleARN != "" {
		existing.RoleARN = p.RoleARN
	}
	existing.Version++
	existing.UpdatedAt = time.Now().UTC()
	return existing, nil
}

func (s *Store) DeletePipeline(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pipelines[name]; !ok {
		return service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", name), http.StatusNotFound)
	}
	delete(s.pipelines, name)
	delete(s.executions, name)
	return nil
}

// ---- Execution operations ----

func (s *Store) StartPipelineExecution(pipelineName string) (*PipelineExecution, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pipelines[pipelineName]
	if !ok {
		return nil, service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", pipelineName), http.StatusNotFound)
	}

	execID := newUUID()
	now := time.Now().UTC()

	// Build stage states from pipeline definition
	stageStates := make([]StageState, len(p.Stages))
	for i, stage := range p.Stages {
		actionStates := make([]ActionState, len(stage.Actions))
		for j, action := range stage.Actions {
			actionStates[j] = ActionState{
				ActionName: action.Name,
				Status:     ActionStatusInProgress,
				LastUpdate: now,
			}
		}
		status := ""
		if i == 0 {
			status = ActionStatusInProgress
		}
		stageStates[i] = StageState{
			StageName:    stage.Name,
			ActionStates: actionStates,
			Status:       status,
		}
	}

	exec := &PipelineExecution{
		ID:              execID,
		PipelineName:    pipelineName,
		PipelineVersion: p.Version,
		Status:          ExecStatusInProgress,
		StartTime:       now,
		StageStates:     stageStates,
	}

	// Set up lifecycle: InProgress -> Succeeded
	transitions := []lifecycle.Transition{
		{From: lifecycle.State(ExecStatusInProgress), To: lifecycle.State(ExecStatusSucceeded), Delay: 10 * time.Second},
	}
	exec.lifecycle = lifecycle.NewMachine(lifecycle.State(ExecStatusInProgress), transitions, s.lcConfig)
	exec.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		exec.Status = string(to)
		if to == lifecycle.State(ExecStatusSucceeded) || to == lifecycle.State(ExecStatusFailed) {
			now := time.Now().UTC()
			exec.EndTime = &now
			for i := range exec.StageStates {
				exec.StageStates[i].Status = string(to)
				for j := range exec.StageStates[i].ActionStates {
					exec.StageStates[i].ActionStates[j].Status = string(to)
					exec.StageStates[i].ActionStates[j].LastUpdate = now
				}
			}
		}
	})

	s.executions[pipelineName] = append(s.executions[pipelineName], exec)
	return exec, nil
}

func (s *Store) GetPipelineState(pipelineName string) (*Pipeline, []*PipelineExecution, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.pipelines[pipelineName]
	if !ok {
		return nil, nil, service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", pipelineName), http.StatusNotFound)
	}

	execs := s.executions[pipelineName]
	// Refresh statuses
	for _, e := range execs {
		if e.lifecycle != nil {
			e.Status = string(e.lifecycle.State())
		}
	}
	return p, execs, nil
}

func (s *Store) GetPipelineExecution(pipelineName, executionID string) (*PipelineExecution, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	execs := s.executions[pipelineName]
	for _, e := range execs {
		if e.ID == executionID {
			if e.lifecycle != nil {
				e.Status = string(e.lifecycle.State())
			}
			return e, nil
		}
	}
	return nil, service.NewAWSError("PipelineExecutionNotFoundException",
		fmt.Sprintf("Execution not found: %s", executionID), http.StatusNotFound)
}

func (s *Store) ListPipelineExecutions(pipelineName string) ([]*PipelineExecution, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.pipelines[pipelineName]; !ok {
		return nil, service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", pipelineName), http.StatusNotFound)
	}

	execs := s.executions[pipelineName]
	for _, e := range execs {
		if e.lifecycle != nil {
			e.Status = string(e.lifecycle.State())
		}
	}
	return execs, nil
}

func (s *Store) StopPipelineExecution(pipelineName, executionID, reason string, abandon bool) (*PipelineExecution, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	execs := s.executions[pipelineName]
	for _, e := range execs {
		if e.ID == executionID {
			if e.lifecycle != nil {
				e.Status = string(e.lifecycle.State())
			}

			if e.Status != ExecStatusInProgress {
				return nil, service.NewAWSError("PipelineExecutionNotStoppableException",
					"Execution is not in a stoppable state.", http.StatusBadRequest)
			}

			if e.lifecycle != nil {
				e.lifecycle.Stop()
			}
			if abandon {
				e.Status = ExecStatusStopped
			} else {
				e.Status = ExecStatusStopping
			}
			e.StatusSummary = reason
			now := time.Now().UTC()
			e.EndTime = &now
			return e, nil
		}
	}
	return nil, service.NewAWSError("PipelineExecutionNotFoundException",
		fmt.Sprintf("Execution not found: %s", executionID), http.StatusNotFound)
}

func (s *Store) PutApprovalResult(pipelineName, stageName, actionName string, result ApprovalResult) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	execs := s.executions[pipelineName]
	if len(execs) == 0 {
		return service.NewAWSError("PipelineNotFoundException",
			fmt.Sprintf("Pipeline not found: %s", pipelineName), http.StatusNotFound)
	}

	// Apply to latest execution
	exec := execs[len(execs)-1]
	for i := range exec.StageStates {
		if exec.StageStates[i].StageName == stageName {
			for j := range exec.StageStates[i].ActionStates {
				if exec.StageStates[i].ActionStates[j].ActionName == actionName {
					if result.Status == "Approved" {
						exec.StageStates[i].ActionStates[j].Status = ActionStatusSucceeded
					} else {
						exec.StageStates[i].ActionStates[j].Status = ActionStatusFailed
					}
					exec.StageStates[i].ActionStates[j].Summary = result.Summary
					exec.StageStates[i].ActionStates[j].LastUpdate = time.Now().UTC()
					return nil
				}
			}
		}
	}
	return service.NewAWSError("ActionNotFoundException",
		fmt.Sprintf("Action %s not found in stage %s", actionName, stageName), http.StatusNotFound)
}

func (s *Store) RetryStageExecution(pipelineName, stageName, executionID string) (*PipelineExecution, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	execs := s.executions[pipelineName]
	for _, e := range execs {
		if e.ID == executionID {
			for i := range e.StageStates {
				if e.StageStates[i].StageName == stageName {
					e.StageStates[i].Status = ActionStatusInProgress
					for j := range e.StageStates[i].ActionStates {
						e.StageStates[i].ActionStates[j].Status = ActionStatusInProgress
						e.StageStates[i].ActionStates[j].LastUpdate = time.Now().UTC()
					}
					e.Status = ExecStatusInProgress
					return e, nil
				}
			}
			return nil, service.NewAWSError("StageNotFoundException",
				fmt.Sprintf("Stage not found: %s", stageName), http.StatusNotFound)
		}
	}
	return nil, service.NewAWSError("PipelineExecutionNotFoundException",
		fmt.Sprintf("Execution not found: %s", executionID), http.StatusNotFound)
}

// ---- Webhook operations ----

func (s *Store) webhookARN(name string) string {
	return fmt.Sprintf("arn:aws:codepipeline:%s:%s:webhook:%s", s.region, s.accountID, name)
}

func (s *Store) PutWebhook(name, pipelineName, targetAction, authentication string, filters []WebhookFilter, tags map[string]string) (*Webhook, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		return nil, service.ErrValidation("Webhook name is required.")
	}
	if pipelineName == "" {
		return nil, service.ErrValidation("Target pipeline name is required.")
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	wh := s.webhooks[name]
	if wh == nil {
		wh = &Webhook{
			Name:           name,
			ARN:            s.webhookARN(name),
			PipelineName:   pipelineName,
			TargetAction:   targetAction,
			TargetPipeline: pipelineName,
			Authentication: authentication,
			Filters:        filters,
			URL:            fmt.Sprintf("https://webhooks.codepipeline.%s.amazonaws.com/webhooks/%s/%s", s.region, s.accountID, name),
			Tags:           tags,
			CreatedAt:      time.Now().UTC(),
		}
	} else {
		wh.PipelineName = pipelineName
		wh.TargetAction = targetAction
		wh.Authentication = authentication
		wh.Filters = filters
		for k, v := range tags {
			wh.Tags[k] = v
		}
	}

	s.webhooks[name] = wh
	return wh, nil
}

func (s *Store) ListWebhooks() []*Webhook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Webhook, 0, len(s.webhooks))
	for _, wh := range s.webhooks {
		result = append(result, wh)
	}
	return result
}

func (s *Store) DeleteWebhook(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.webhooks[name]; !ok {
		return service.NewAWSError("WebhookNotFoundException",
			fmt.Sprintf("Webhook not found: %s", name), http.StatusNotFound)
	}
	delete(s.webhooks, name)
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
