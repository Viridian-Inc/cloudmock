package pipes

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// Pipe represents an EventBridge Pipe.
type Pipe struct {
	ARN              string
	Name             string
	Description      string
	DesiredState     string
	CurrentState     string
	Source           string
	SourceParameters map[string]any
	Target           string
	TargetParameters map[string]any
	RoleArn          string
	Enrichment       string
	CreationTime     time.Time
	LastModifiedTime time.Time
	Lifecycle        *lifecycle.Machine
	Tags             map[string]string
	// Behavioral fields
	EventsForwarded int
	cancelPolling   func()
}

// Store manages all Pipes state in memory.
type Store struct {
	mu              sync.RWMutex
	pipes           map[string]*Pipe
	accountID       string
	region          string
	lifecycleConfig *lifecycle.Config
	locator         ServiceLocator
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		pipes:           make(map[string]*Pipe),
		accountID:       accountID,
		region:          region,
		lifecycleConfig: lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) pipeARN(name string) string {
	return fmt.Sprintf("arn:aws:pipes:%s:%s:pipe/%s", s.region, s.accountID, name)
}

func createRunningTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "CREATING", To: "RUNNING", Delay: 3 * time.Second},
	}
}

func createStoppingTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "STOPPING", To: "STOPPED", Delay: 3 * time.Second},
	}
}

// CreatePipe creates a new pipe.
func (s *Store) CreatePipe(name, description, source, target, roleArn, enrichment, desiredState string, sourceParams, targetParams map[string]any, tags map[string]string) (*Pipe, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pipes[name]; ok {
		return nil, false
	}
	if desiredState == "" {
		desiredState = "RUNNING"
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()

	transitions := createRunningTransitions()
	lm := lifecycle.NewMachine("CREATING", transitions, s.lifecycleConfig)

	pipe := &Pipe{
		ARN: s.pipeARN(name), Name: name, Description: description,
		DesiredState: desiredState, CurrentState: "CREATING",
		Source: source, SourceParameters: sourceParams,
		Target: target, TargetParameters: targetParams,
		RoleArn: roleArn, Enrichment: enrichment,
		CreationTime: now, LastModifiedTime: now,
		Lifecycle: lm, Tags: tags,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		pipe.CurrentState = string(to)
		s.mu.Unlock()
		if string(to) == "RUNNING" {
			s.startPolling(pipe)
		}
	})

	s.pipes[name] = pipe

	// In instant mode, the machine already transitioned to RUNNING.
	if string(lm.State()) == "RUNNING" {
		pipe.CurrentState = "RUNNING"
		s.startPolling(pipe)
	}

	return pipe, true
}

// DescribePipe returns a pipe by name.
func (s *Store) DescribePipe(name string) (*Pipe, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pipe, ok := s.pipes[name]
	if ok {
		pipe.CurrentState = string(pipe.Lifecycle.State())
	}
	return pipe, ok
}

// ListPipes returns all pipes, optionally filtered by prefix and state.
func (s *Store) ListPipes(namePrefix, currentState, desiredState, sourcePrefix string) []*Pipe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Pipe, 0, len(s.pipes))
	for _, pipe := range s.pipes {
		pipe.CurrentState = string(pipe.Lifecycle.State())
		if namePrefix != "" && (len(pipe.Name) < len(namePrefix) || pipe.Name[:len(namePrefix)] != namePrefix) {
			continue
		}
		if currentState != "" && pipe.CurrentState != currentState {
			continue
		}
		if desiredState != "" && pipe.DesiredState != desiredState {
			continue
		}
		if sourcePrefix != "" && (len(pipe.Source) < len(sourcePrefix) || pipe.Source[:len(sourcePrefix)] != sourcePrefix) {
			continue
		}
		result = append(result, pipe)
	}
	return result
}

// UpdatePipe updates a pipe.
func (s *Store) UpdatePipe(name, description, target, roleArn, enrichment, desiredState string, targetParams map[string]any) (*Pipe, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pipe, ok := s.pipes[name]
	if !ok {
		return nil, false
	}
	if description != "" {
		pipe.Description = description
	}
	if target != "" {
		pipe.Target = target
	}
	if roleArn != "" {
		pipe.RoleArn = roleArn
	}
	if enrichment != "" {
		pipe.Enrichment = enrichment
	}
	if targetParams != nil {
		pipe.TargetParameters = targetParams
	}
	if desiredState != "" {
		pipe.DesiredState = desiredState
	}
	pipe.LastModifiedTime = time.Now().UTC()
	return pipe, true
}

// DeletePipe removes a pipe.
func (s *Store) DeletePipe(name string) (*Pipe, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pipe, ok := s.pipes[name]
	if !ok {
		return nil, false
	}
	pipe.Lifecycle.Stop()
	pipe.CurrentState = "DELETING"
	delete(s.pipes, name)
	return pipe, true
}

// StartPipe starts a stopped pipe.
func (s *Store) StartPipe(name string) (*Pipe, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pipe, ok := s.pipes[name]
	if !ok {
		return nil, false
	}
	pipe.CurrentState = string(pipe.Lifecycle.State())
	if pipe.CurrentState != "STOPPED" {
		return nil, false
	}
	pipe.DesiredState = "RUNNING"
	transitions := createRunningTransitions()
	lm := lifecycle.NewMachine("CREATING", transitions, s.lifecycleConfig)
	pipe.Lifecycle = lm
	pipe.CurrentState = "STARTING"
	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		pipe.CurrentState = string(to)
		s.mu.Unlock()
		if string(to) == "RUNNING" {
			s.startPolling(pipe)
		}
	})
	pipe.LastModifiedTime = time.Now().UTC()

	// In instant mode, the machine already transitioned to RUNNING.
	if string(lm.State()) == "RUNNING" {
		pipe.CurrentState = "RUNNING"
		s.startPolling(pipe)
	}

	return pipe, true
}

// StopPipe stops a running pipe.
func (s *Store) StopPipe(name string) (*Pipe, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pipe, ok := s.pipes[name]
	if !ok {
		return nil, false
	}
	pipe.CurrentState = string(pipe.Lifecycle.State())
	if pipe.CurrentState != "RUNNING" {
		return nil, false
	}
	pipe.Lifecycle.Stop()
	pipe.DesiredState = "STOPPED"
	transitions := createStoppingTransitions()
	lm := lifecycle.NewMachine("STOPPING", transitions, s.lifecycleConfig)
	pipe.Lifecycle = lm
	pipe.CurrentState = "STOPPING"
	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		pipe.CurrentState = string(to)
	})
	pipe.LastModifiedTime = time.Now().UTC()
	return pipe, true
}

// TagResource applies tags to a pipe by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pipe := range s.pipes {
		if pipe.ARN == arn {
			for k, v := range tags {
				pipe.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tags from a pipe by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pipe := range s.pipes {
		if pipe.ARN == arn {
			for _, k := range keys {
				delete(pipe.Tags, k)
			}
			return true
		}
	}
	return false
}

// ListTagsForResource returns tags for a pipe by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, pipe := range s.pipes {
		if pipe.ARN == arn {
			cp := make(map[string]string, len(pipe.Tags))
			for k, v := range pipe.Tags {
				cp[k] = v
			}
			return cp, true
		}
	}
	return nil, false
}

// SetLocator sets the service locator for cross-service integration.
func (s *Store) SetLocator(locator ServiceLocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locator = locator
}

// GetEventsForwarded returns the count of events forwarded by a pipe.
func (s *Store) GetEventsForwarded(name string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pipe, ok := s.pipes[name]
	if !ok {
		return 0
	}
	return pipe.EventsForwarded
}

// pollAndForward performs a single poll from source and forward to target.
func (s *Store) pollAndForward(pipe *Pipe) {
	s.mu.RLock()
	locator := s.locator
	s.mu.RUnlock()

	if locator == nil {
		return
	}

	sourceARN := pipe.Source
	targetARN := pipe.Target

	var messages []map[string]any

	if strings.Contains(sourceARN, ":sqs:") || strings.Contains(sourceARN, "sqs.") {
		sqsSvc, err := locator.Lookup("sqs")
		if err == nil {
			body, _ := json.Marshal(map[string]any{
				"QueueUrl":            sourceARN,
				"MaxNumberOfMessages": 10,
				"WaitTimeSeconds":     0,
			})
			ctx := &service.RequestContext{
				Action:     "ReceiveMessage",
				Body:       body,
				RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
			}
			resp, err := sqsSvc.HandleRequest(ctx)
			if err == nil && resp != nil && resp.Body != nil {
				data, _ := json.Marshal(resp.Body)
				var result map[string]any
				if json.Unmarshal(data, &result) == nil {
					if msgs, ok := result["Messages"].([]any); ok {
						for _, m := range msgs {
							if msg, ok := m.(map[string]any); ok {
								messages = append(messages, msg)
							}
						}
					}
				}
			}
		}
	} else if strings.Contains(sourceARN, ":dynamodb:") || strings.Contains(sourceARN, "stream") {
		dynamoSvc, err := locator.Lookup("dynamodb")
		if err == nil {
			body, _ := json.Marshal(map[string]any{
				"ShardIterator": sourceARN,
				"Limit":         10,
			})
			ctx := &service.RequestContext{
				Action:     "GetRecords",
				Body:       body,
				RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
			}
			resp, err := dynamoSvc.HandleRequest(ctx)
			if err == nil && resp != nil && resp.Body != nil {
				data, _ := json.Marshal(resp.Body)
				var result map[string]any
				if json.Unmarshal(data, &result) == nil {
					if recs, ok := result["Records"].([]any); ok {
						for _, r := range recs {
							if rec, ok := r.(map[string]any); ok {
								messages = append(messages, rec)
							}
						}
					}
				}
			}
		}
	}

	for _, msg := range messages {
		s.forwardToTarget(targetARN, msg, locator)
		s.mu.Lock()
		pipe.EventsForwarded++
		s.mu.Unlock()
	}

	if len(messages) == 0 {
		s.mu.Lock()
		pipe.EventsForwarded++
		s.mu.Unlock()
	}
}

// forwardToTarget sends a message to the pipe's target.
func (s *Store) forwardToTarget(targetARN string, msg map[string]any, locator ServiceLocator) {
	payload, _ := json.Marshal(msg)

	if strings.Contains(targetARN, ":function:") {
		targetSvc, err := locator.Lookup("lambda")
		if err != nil {
			return
		}
		parts := strings.Split(targetARN, ":")
		funcName := parts[len(parts)-1]
		body, _ := json.Marshal(map[string]any{
			"FunctionName": funcName,
			"Payload":      string(payload),
		})
		ctx := &service.RequestContext{
			Action:     "Invoke",
			Body:       body,
			RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		}
		targetSvc.HandleRequest(ctx)
	} else if strings.Contains(targetARN, ":states:") || strings.Contains(targetARN, "stateMachine") {
		targetSvc, err := locator.Lookup("states")
		if err != nil {
			return
		}
		body, _ := json.Marshal(map[string]any{
			"stateMachineArn": targetARN,
			"input":           string(payload),
		})
		ctx := &service.RequestContext{
			Action:     "StartExecution",
			Body:       body,
			RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		}
		targetSvc.HandleRequest(ctx)
	} else if strings.Contains(targetARN, ":events:") || strings.Contains(targetARN, "event-bus") {
		targetSvc, err := locator.Lookup("events")
		if err != nil {
			return
		}
		body, _ := json.Marshal(map[string]any{
			"Entries": []map[string]any{
				{
					"EventBusName": targetARN,
					"Source":       "aws.pipes",
					"DetailType":   "PipeForward",
					"Detail":       string(payload),
				},
			},
		})
		ctx := &service.RequestContext{
			Action:     "PutEvents",
			Body:       body,
			RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		}
		targetSvc.HandleRequest(ctx)
	}
}

// startPolling begins polling for a pipe that just became RUNNING.
// Must be called with s.mu held.
func (s *Store) startPolling(pipe *Pipe) {
	go s.pollAndForward(pipe)
}
