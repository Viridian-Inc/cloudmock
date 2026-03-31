package swf

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// Domain represents an SWF domain.
type Domain struct {
	Name                                   string
	Description                            string
	Status                                 string // REGISTERED | DEPRECATED
	WorkflowExecutionRetentionPeriodInDays string
	ARN                                    string
	Tags                                   map[string]string
}

// WorkflowType represents an SWF workflow type.
type WorkflowType struct {
	Domain                   string
	Name                     string
	Version                  string
	Description              string
	Status                   string // REGISTERED | DEPRECATED
	DefaultTaskList          string
	DefaultExecutionTimeout  string
	DefaultTaskTimeout       string
	DefaultChildPolicy       string
	CreationDate             time.Time
	DeprecationDate          *time.Time
}

// ActivityType represents an SWF activity type.
type ActivityType struct {
	Domain                  string
	Name                    string
	Version                 string
	Description             string
	Status                  string // REGISTERED | DEPRECATED
	DefaultTaskList         string
	DefaultTaskTimeout      string
	DefaultHeartbeatTimeout string
	DefaultScheduleToStartTimeout string
	DefaultScheduleToCloseTimeout string
	CreationDate            time.Time
	DeprecationDate         *time.Time
}

// WorkflowExecution represents a running workflow execution.
type WorkflowExecution struct {
	Domain       string
	WorkflowID   string
	RunID        string
	WorkflowType *WorkflowType
	TaskList     string
	Input        string
	Status       string // OPEN | CLOSED | COMPLETED | FAILED | CANCELED | TERMINATED | CONTINUED_AS_NEW | TIMED_OUT
	StartTime    time.Time
	CloseTime    *time.Time
	CloseStatus  string
	Tags         map[string]string
	// Decision task management
	PendingDecisionTask bool
	DecisionTaskToken   string
	// Activity task management
	PendingActivities []PendingActivity
}

// PendingActivity tracks a scheduled activity task.
type PendingActivity struct {
	ActivityID    string
	ActivityType  string
	TaskToken     string
	Input         string
	Scheduled     time.Time
}

// Store manages all SWF state in memory.
type Store struct {
	mu          sync.RWMutex
	domains     map[string]*Domain
	workflows   map[string]map[string]*WorkflowType // domain -> name:version -> type
	activities  map[string]map[string]*ActivityType  // domain -> name:version -> type
	executions  map[string]map[string]*WorkflowExecution // domain -> workflowID:runID -> execution
	accountID   string
	region      string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		domains:    make(map[string]*Domain),
		workflows:  make(map[string]map[string]*WorkflowType),
		activities: make(map[string]map[string]*ActivityType),
		executions: make(map[string]map[string]*WorkflowExecution),
		accountID:  accountID,
		region:     region,
	}
}

func newRunID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newTaskToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *Store) domainARN(name string) string {
	return fmt.Sprintf("arn:aws:swf:%s:%s:/domain/%s", s.region, s.accountID, name)
}

func typeKey(name, version string) string {
	return name + ":" + version
}

func execKey(workflowID, runID string) string {
	return workflowID + ":" + runID
}

// RegisterDomain registers a new domain.
func (s *Store) RegisterDomain(name, description, retention string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[name]; ok {
		return false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	s.domains[name] = &Domain{
		Name: name, Description: description, Status: "REGISTERED",
		WorkflowExecutionRetentionPeriodInDays: retention,
		ARN: s.domainARN(name), Tags: tags,
	}
	s.workflows[name] = make(map[string]*WorkflowType)
	s.activities[name] = make(map[string]*ActivityType)
	s.executions[name] = make(map[string]*WorkflowExecution)
	return true
}

// DescribeDomain returns a domain by name.
func (s *Store) DescribeDomain(name string) (*Domain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.domains[name]
	return d, ok
}

// ListDomains returns domains filtered by status.
func (s *Store) ListDomains(status string) []*Domain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Domain, 0)
	for _, d := range s.domains {
		if status != "" && d.Status != status {
			continue
		}
		result = append(result, d)
	}
	return result
}

// DeprecateDomain deprecates a domain.
func (s *Store) DeprecateDomain(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.domains[name]
	if !ok || d.Status == "DEPRECATED" {
		return false
	}
	d.Status = "DEPRECATED"
	return true
}

// RegisterWorkflowType registers a new workflow type.
func (s *Store) RegisterWorkflowType(domain, name, version, description, defaultTaskList, execTimeout, taskTimeout, childPolicy string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	wfs, ok := s.workflows[domain]
	if !ok {
		return false
	}
	key := typeKey(name, version)
	if _, exists := wfs[key]; exists {
		return false
	}
	wfs[key] = &WorkflowType{
		Domain: domain, Name: name, Version: version,
		Description: description, Status: "REGISTERED",
		DefaultTaskList: defaultTaskList,
		DefaultExecutionTimeout: execTimeout,
		DefaultTaskTimeout: taskTimeout,
		DefaultChildPolicy: childPolicy,
		CreationDate: time.Now().UTC(),
	}
	return true
}

// DescribeWorkflowType returns a workflow type.
func (s *Store) DescribeWorkflowType(domain, name, version string) (*WorkflowType, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wfs, ok := s.workflows[domain]
	if !ok {
		return nil, false
	}
	wt, ok := wfs[typeKey(name, version)]
	return wt, ok
}

// ListWorkflowTypes returns workflow types filtered by status.
func (s *Store) ListWorkflowTypes(domain, status, nameFilter string) []*WorkflowType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wfs, ok := s.workflows[domain]
	if !ok {
		return nil
	}
	result := make([]*WorkflowType, 0)
	for _, wt := range wfs {
		if status != "" && wt.Status != status {
			continue
		}
		if nameFilter != "" && wt.Name != nameFilter {
			continue
		}
		result = append(result, wt)
	}
	return result
}

// DeprecateWorkflowType deprecates a workflow type.
func (s *Store) DeprecateWorkflowType(domain, name, version string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	wfs, ok := s.workflows[domain]
	if !ok {
		return false
	}
	wt, ok := wfs[typeKey(name, version)]
	if !ok || wt.Status == "DEPRECATED" {
		return false
	}
	wt.Status = "DEPRECATED"
	now := time.Now().UTC()
	wt.DeprecationDate = &now
	return true
}

// RegisterActivityType registers a new activity type.
func (s *Store) RegisterActivityType(domain, name, version, description, defaultTaskList, taskTimeout, heartbeatTimeout, scheduleToStartTimeout, scheduleToCloseTimeout string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	acts, ok := s.activities[domain]
	if !ok {
		return false
	}
	key := typeKey(name, version)
	if _, exists := acts[key]; exists {
		return false
	}
	acts[key] = &ActivityType{
		Domain: domain, Name: name, Version: version,
		Description: description, Status: "REGISTERED",
		DefaultTaskList: defaultTaskList,
		DefaultTaskTimeout: taskTimeout,
		DefaultHeartbeatTimeout: heartbeatTimeout,
		DefaultScheduleToStartTimeout: scheduleToStartTimeout,
		DefaultScheduleToCloseTimeout: scheduleToCloseTimeout,
		CreationDate: time.Now().UTC(),
	}
	return true
}

// DescribeActivityType returns an activity type.
func (s *Store) DescribeActivityType(domain, name, version string) (*ActivityType, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acts, ok := s.activities[domain]
	if !ok {
		return nil, false
	}
	at, ok := acts[typeKey(name, version)]
	return at, ok
}

// ListActivityTypes returns activity types filtered by status.
func (s *Store) ListActivityTypes(domain, status, nameFilter string) []*ActivityType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acts, ok := s.activities[domain]
	if !ok {
		return nil
	}
	result := make([]*ActivityType, 0)
	for _, at := range acts {
		if status != "" && at.Status != status {
			continue
		}
		if nameFilter != "" && at.Name != nameFilter {
			continue
		}
		result = append(result, at)
	}
	return result
}

// DeprecateActivityType deprecates an activity type.
func (s *Store) DeprecateActivityType(domain, name, version string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	acts, ok := s.activities[domain]
	if !ok {
		return false
	}
	at, ok := acts[typeKey(name, version)]
	if !ok || at.Status == "DEPRECATED" {
		return false
	}
	at.Status = "DEPRECATED"
	now := time.Now().UTC()
	at.DeprecationDate = &now
	return true
}

// StartWorkflowExecution starts a new workflow execution.
func (s *Store) StartWorkflowExecution(domain, workflowID, workflowName, workflowVersion, taskList, input string, tags map[string]string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	execs, ok := s.executions[domain]
	if !ok {
		return "", "domain"
	}
	wfs, ok := s.workflows[domain]
	if !ok {
		return "", "domain"
	}
	wt, ok := wfs[typeKey(workflowName, workflowVersion)]
	if !ok {
		return "", "workflowType"
	}
	// Check for duplicate open execution
	for _, exec := range execs {
		if exec.WorkflowID == workflowID && exec.Status == "OPEN" {
			return "", "duplicate"
		}
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if taskList == "" {
		taskList = wt.DefaultTaskList
	}
	runID := newRunID()
	exec := &WorkflowExecution{
		Domain: domain, WorkflowID: workflowID, RunID: runID,
		WorkflowType: wt, TaskList: taskList, Input: input,
		Status: "OPEN", StartTime: time.Now().UTC(), Tags: tags,
		PendingDecisionTask: true, DecisionTaskToken: newTaskToken(),
	}
	execs[execKey(workflowID, runID)] = exec
	return runID, ""
}

// DescribeWorkflowExecution returns a workflow execution.
func (s *Store) DescribeWorkflowExecution(domain, workflowID, runID string) (*WorkflowExecution, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execs, ok := s.executions[domain]
	if !ok {
		return nil, false
	}
	exec, ok := execs[execKey(workflowID, runID)]
	return exec, ok
}

// ListOpenWorkflowExecutions returns open executions in a domain.
func (s *Store) ListOpenWorkflowExecutions(domain string) []*WorkflowExecution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execs, ok := s.executions[domain]
	if !ok {
		return nil
	}
	result := make([]*WorkflowExecution, 0)
	for _, exec := range execs {
		if exec.Status == "OPEN" {
			result = append(result, exec)
		}
	}
	return result
}

// ListClosedWorkflowExecutions returns closed executions in a domain.
func (s *Store) ListClosedWorkflowExecutions(domain string) []*WorkflowExecution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execs, ok := s.executions[domain]
	if !ok {
		return nil
	}
	result := make([]*WorkflowExecution, 0)
	for _, exec := range execs {
		if exec.Status != "OPEN" {
			result = append(result, exec)
		}
	}
	return result
}

// TerminateWorkflowExecution terminates a workflow execution.
func (s *Store) TerminateWorkflowExecution(domain, workflowID, runID, reason, details, childPolicy string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	execs, ok := s.executions[domain]
	if !ok {
		return false
	}
	// If runID is empty, find the open execution for the workflowID.
	if runID == "" {
		for _, exec := range execs {
			if exec.WorkflowID == workflowID && exec.Status == "OPEN" {
				runID = exec.RunID
				break
			}
		}
	}
	exec, ok := execs[execKey(workflowID, runID)]
	if !ok || exec.Status != "OPEN" {
		return false
	}
	now := time.Now().UTC()
	exec.Status = "TERMINATED"
	exec.CloseStatus = "TERMINATED"
	exec.CloseTime = &now
	return true
}

// SignalWorkflowExecution sends a signal to a workflow execution.
func (s *Store) SignalWorkflowExecution(domain, workflowID, runID, signalName, input string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	execs, ok := s.executions[domain]
	if !ok {
		return false
	}
	if runID == "" {
		for _, exec := range execs {
			if exec.WorkflowID == workflowID && exec.Status == "OPEN" {
				runID = exec.RunID
				break
			}
		}
	}
	exec, ok := execs[execKey(workflowID, runID)]
	if !ok || exec.Status != "OPEN" {
		return false
	}
	// Schedule a new decision task upon signal.
	exec.PendingDecisionTask = true
	exec.DecisionTaskToken = newTaskToken()
	return true
}

// RequestCancelWorkflowExecution requests cancellation of a workflow execution.
func (s *Store) RequestCancelWorkflowExecution(domain, workflowID, runID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	execs, ok := s.executions[domain]
	if !ok {
		return false
	}
	if runID == "" {
		for _, exec := range execs {
			if exec.WorkflowID == workflowID && exec.Status == "OPEN" {
				runID = exec.RunID
				break
			}
		}
	}
	exec, ok := execs[execKey(workflowID, runID)]
	if !ok || exec.Status != "OPEN" {
		return false
	}
	// Schedule a decision task for the cancel request.
	exec.PendingDecisionTask = true
	exec.DecisionTaskToken = newTaskToken()
	return true
}

// PollForDecisionTask returns a pending decision task.
func (s *Store) PollForDecisionTask(domain, taskList string) (*WorkflowExecution, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	execs, ok := s.executions[domain]
	if !ok {
		return nil, ""
	}
	for _, exec := range execs {
		if exec.Status == "OPEN" && exec.PendingDecisionTask && exec.TaskList == taskList {
			token := exec.DecisionTaskToken
			exec.PendingDecisionTask = false
			return exec, token
		}
	}
	return nil, ""
}

// RespondDecisionTaskCompleted processes a decision task completion.
func (s *Store) RespondDecisionTaskCompleted(taskToken string, decisions []map[string]any) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, execs := range s.executions {
		for _, exec := range execs {
			if exec.DecisionTaskToken == taskToken {
				// Process decisions
				for _, decision := range decisions {
					decisionType, _ := decision["decisionType"].(string)
					switch decisionType {
					case "CompleteWorkflowExecution":
						now := time.Now().UTC()
						exec.Status = "COMPLETED"
						exec.CloseStatus = "COMPLETED"
						exec.CloseTime = &now
					case "FailWorkflowExecution":
						now := time.Now().UTC()
						exec.Status = "FAILED"
						exec.CloseStatus = "FAILED"
						exec.CloseTime = &now
					case "CancelWorkflowExecution":
						now := time.Now().UTC()
						exec.Status = "CANCELED"
						exec.CloseStatus = "CANCELED"
						exec.CloseTime = &now
					case "ScheduleActivityTask":
						attrs, _ := decision["scheduleActivityTaskDecisionAttributes"].(map[string]any)
						activityID, _ := attrs["activityId"].(string)
						activityType, _ := attrs["activityType"].(map[string]any)
						typeName, _ := activityType["name"].(string)
						inputStr, _ := attrs["input"].(string)
						exec.PendingActivities = append(exec.PendingActivities, PendingActivity{
							ActivityID: activityID, ActivityType: typeName,
							TaskToken: newTaskToken(), Input: inputStr,
							Scheduled: time.Now().UTC(),
						})
					}
				}
				return true
			}
		}
	}
	return false
}

// PollForActivityTask returns a pending activity task.
func (s *Store) PollForActivityTask(domain, taskList string) (*WorkflowExecution, *PendingActivity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	execs, ok := s.executions[domain]
	if !ok {
		return nil, nil
	}
	for _, exec := range execs {
		if exec.Status != "OPEN" || exec.TaskList != taskList {
			continue
		}
		for i := range exec.PendingActivities {
			activity := &exec.PendingActivities[i]
			if activity.TaskToken != "" {
				token := activity.TaskToken
				_ = token
				return exec, activity
			}
		}
	}
	return nil, nil
}

// RespondActivityTaskCompleted completes an activity task.
func (s *Store) RespondActivityTaskCompleted(taskToken, result string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, execs := range s.executions {
		for _, exec := range execs {
			for i, act := range exec.PendingActivities {
				if act.TaskToken == taskToken {
					exec.PendingActivities = append(exec.PendingActivities[:i], exec.PendingActivities[i+1:]...)
					exec.PendingDecisionTask = true
					exec.DecisionTaskToken = newTaskToken()
					return true
				}
			}
		}
	}
	return false
}

// RespondActivityTaskFailed fails an activity task.
func (s *Store) RespondActivityTaskFailed(taskToken, reason, details string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, execs := range s.executions {
		for _, exec := range execs {
			for i, act := range exec.PendingActivities {
				if act.TaskToken == taskToken {
					exec.PendingActivities = append(exec.PendingActivities[:i], exec.PendingActivities[i+1:]...)
					exec.PendingDecisionTask = true
					exec.DecisionTaskToken = newTaskToken()
					return true
				}
			}
		}
	}
	return false
}

// TagResource applies tags to a domain by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.domains {
		if d.ARN == arn {
			for k, v := range tags {
				d.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tags from a domain by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.domains {
		if d.ARN == arn {
			for _, k := range keys {
				delete(d.Tags, k)
			}
			return true
		}
	}
	return false
}

// ListTagsForResource returns tags for a domain by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.domains {
		if d.ARN == arn {
			cp := make(map[string]string, len(d.Tags))
			for k, v := range d.Tags {
				cp[k] = v
			}
			return cp, true
		}
	}
	return nil, false
}
