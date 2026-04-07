package stepfunctions

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// StateMachineStatus is the lifecycle state of a state machine.
type StateMachineStatus string

const (
	stateMachineStatusActive   StateMachineStatus = "ACTIVE"
	stateMachineStatusDeleting StateMachineStatus = "DELETING"
)

// ExecutionStatus is the lifecycle state of an execution.
type ExecutionStatus string

const (
	executionStatusRunning   ExecutionStatus = "RUNNING"
	executionStatusSucceeded ExecutionStatus = "SUCCEEDED"
	executionStatusFailed    ExecutionStatus = "FAILED"
	executionStatusTimedOut  ExecutionStatus = "TIMED_OUT"
	executionStatusAborted   ExecutionStatus = "ABORTED"
)

// StateMachine holds all metadata for an AWS Step Functions state machine.
type StateMachine struct {
	Arn          string
	Name         string
	Definition   string // ASL JSON string
	RoleArn      string
	Type         string // STANDARD | EXPRESS
	Status       StateMachineStatus
	CreationDate time.Time
	Tags         map[string]string
}

// HistoryEvent is a single event in an execution's history.
type HistoryEvent struct {
	Timestamp       time.Time
	Type            string
	Id              int64
	PreviousEventId int64
}

// Execution holds all metadata and history for a state machine execution.
type Execution struct {
	ExecutionArn    string
	StateMachineArn string
	Name            string
	Status          ExecutionStatus
	Input           string
	Output          string
	StartDate       time.Time
	StopDate        *time.Time
	Events          []HistoryEvent
}

// Store is the in-memory store for Step Functions state machines and executions.
type Store struct {
	mu            sync.RWMutex
	stateMachines map[string]*StateMachine // keyed by ARN
	executions    map[string]*Execution    // keyed by ARN
	accountID     string
	region        string
}

// NewStore creates an empty Step Functions Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		stateMachines: make(map[string]*StateMachine),
		executions:    make(map[string]*Execution),
		accountID:     accountID,
		region:        region,
	}
}

// newUUID returns a random UUID-shaped identifier.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// stateMachineARN builds the ARN for a state machine.
func (s *Store) stateMachineARN(name string) string {
	return fmt.Sprintf("arn:aws:states:%s:%s:stateMachine:%s", s.region, s.accountID, name)
}

// executionARN builds the ARN for an execution.
func (s *Store) executionARN(smName, execName string) string {
	return fmt.Sprintf("arn:aws:states:%s:%s:execution:%s:%s", s.region, s.accountID, smName, execName)
}

// CreateStateMachine creates a new state machine and returns it.
func (s *Store) CreateStateMachine(name, definition, roleArn, smType string, tags map[string]string) (*StateMachine, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	arn := s.stateMachineARN(name)
	if _, exists := s.stateMachines[arn]; exists {
		return nil, service.NewAWSError("StateMachineAlreadyExists",
			fmt.Sprintf("State Machine Already Exists: '%s'", arn), http.StatusConflict)
	}

	if smType == "" {
		smType = "STANDARD"
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	sm := &StateMachine{
		Arn:          arn,
		Name:         name,
		Definition:   definition,
		RoleArn:      roleArn,
		Type:         smType,
		Status:       stateMachineStatusActive,
		CreationDate: time.Now().UTC(),
		Tags:         tags,
	}
	s.stateMachines[arn] = sm
	return sm, nil
}

// GetStateMachine retrieves a state machine by ARN.
func (s *Store) GetStateMachine(arn string) (*StateMachine, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sm, ok := s.stateMachines[arn]
	if !ok {
		return nil, service.NewAWSError("StateMachineDoesNotExist",
			fmt.Sprintf("State Machine Does Not Exist: '%s'", arn), http.StatusBadRequest)
	}
	return sm, nil
}

// ListStateMachines returns a snapshot of all state machines.
func (s *Store) ListStateMachines() []*StateMachine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StateMachine, 0, len(s.stateMachines))
	for _, sm := range s.stateMachines {
		out = append(out, sm)
	}
	return out
}

// UpdateStateMachine updates the definition and/or roleArn of a state machine.
func (s *Store) UpdateStateMachine(arn, definition, roleArn string) (*StateMachine, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sm, ok := s.stateMachines[arn]
	if !ok {
		return nil, service.NewAWSError("StateMachineDoesNotExist",
			fmt.Sprintf("State Machine Does Not Exist: '%s'", arn), http.StatusBadRequest)
	}
	if definition != "" {
		sm.Definition = definition
	}
	if roleArn != "" {
		sm.RoleArn = roleArn
	}
	return sm, nil
}

// DeleteStateMachine removes a state machine by ARN.
func (s *Store) DeleteStateMachine(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.stateMachines[arn]; !ok {
		return service.NewAWSError("StateMachineDoesNotExist",
			fmt.Sprintf("State Machine Does Not Exist: '%s'", arn), http.StatusBadRequest)
	}
	delete(s.stateMachines, arn)
	return nil
}

// StartExecution creates a new execution for the given state machine and immediately
// marks it as SUCCEEDED with Output=Input (pass-through; real ASL execution is deferred).
func (s *Store) StartExecution(smArn, execName, input string) (*Execution, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sm, ok := s.stateMachines[smArn]
	if !ok {
		return nil, service.NewAWSError("StateMachineDoesNotExist",
			fmt.Sprintf("State Machine Does Not Exist: '%s'", smArn), http.StatusBadRequest)
	}

	if execName == "" {
		execName = newUUID()
	}

	execArn := s.executionARN(sm.Name, execName)
	if _, exists := s.executions[execArn]; exists {
		return nil, service.NewAWSError("ExecutionAlreadyExists",
			fmt.Sprintf("Execution Already Exists: '%s'", execArn), http.StatusConflict)
	}

	if input == "" {
		input = "{}"
	}

	now := time.Now().UTC()
	stopDate := now

	// Build execution history.
	events := []HistoryEvent{
		{
			Timestamp:       now,
			Type:            "ExecutionStarted",
			Id:              1,
			PreviousEventId: 0,
		},
		{
			Timestamp:       now,
			Type:            "ExecutionSucceeded",
			Id:              2,
			PreviousEventId: 1,
		},
	}

	exec := &Execution{
		ExecutionArn:    execArn,
		StateMachineArn: smArn,
		Name:            execName,
		Status:          executionStatusSucceeded,
		Input:           input,
		Output:          input, // pass-through
		StartDate:       now,
		StopDate:        &stopDate,
		Events:          events,
	}
	s.executions[execArn] = exec
	return exec, nil
}

// GetExecution retrieves an execution by ARN.
func (s *Store) GetExecution(execArn string) (*Execution, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exec, ok := s.executions[execArn]
	if !ok {
		return nil, service.NewAWSError("ExecutionDoesNotExist",
			fmt.Sprintf("Execution Does Not Exist: '%s'", execArn), http.StatusBadRequest)
	}
	return exec, nil
}

// ListExecutions returns all executions for a given state machine ARN.
func (s *Store) ListExecutions(smArn string) ([]*Execution, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.stateMachines[smArn]; !ok {
		return nil, service.NewAWSError("StateMachineDoesNotExist",
			fmt.Sprintf("State Machine Does Not Exist: '%s'", smArn), http.StatusBadRequest)
	}

	out := make([]*Execution, 0)
	for _, exec := range s.executions {
		if exec.StateMachineArn == smArn {
			out = append(out, exec)
		}
	}
	return out, nil
}

// StopExecution aborts an execution by ARN.
func (s *Store) StopExecution(execArn, cause, errCode string) (*Execution, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, ok := s.executions[execArn]
	if !ok {
		return nil, service.NewAWSError("ExecutionDoesNotExist",
			fmt.Sprintf("Execution Does Not Exist: '%s'", execArn), http.StatusBadRequest)
	}

	now := time.Now().UTC()
	exec.Status = executionStatusAborted
	exec.StopDate = &now

	// Append a stop event to history.
	nextID := int64(len(exec.Events) + 1)
	exec.Events = append(exec.Events, HistoryEvent{
		Timestamp:       now,
		Type:            "ExecutionAborted",
		Id:              nextID,
		PreviousEventId: nextID - 1,
	})

	return exec, nil
}

// GetExecutionHistory returns the events for an execution.
func (s *Store) GetExecutionHistory(execArn string) ([]HistoryEvent, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exec, ok := s.executions[execArn]
	if !ok {
		return nil, service.NewAWSError("ExecutionDoesNotExist",
			fmt.Sprintf("Execution Does Not Exist: '%s'", execArn), http.StatusBadRequest)
	}

	events := make([]HistoryEvent, len(exec.Events))
	copy(events, exec.Events)
	return events, nil
}

// TagResource applies tags to a state machine identified by ARN.
func (s *Store) TagResource(resourceArn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	sm, ok := s.stateMachines[resourceArn]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Resource Not Found: '%s'", resourceArn), http.StatusBadRequest)
	}
	for k, v := range tags {
		sm.Tags[k] = v
	}
	return nil
}

// UntagResource removes tags from a state machine identified by ARN.
func (s *Store) UntagResource(resourceArn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	sm, ok := s.stateMachines[resourceArn]
	if !ok {
		return service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Resource Not Found: '%s'", resourceArn), http.StatusBadRequest)
	}
	for _, k := range tagKeys {
		delete(sm.Tags, k)
	}
	return nil
}

// ListTagsForResource returns the tags for a state machine identified by ARN.
func (s *Store) ListTagsForResource(resourceArn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sm, ok := s.stateMachines[resourceArn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Resource Not Found: '%s'", resourceArn), http.StatusBadRequest)
	}
	cp := make(map[string]string, len(sm.Tags))
	for k, v := range sm.Tags {
		cp[k] = v
	}
	return cp, nil
}
