package batch

import (
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
)

// ComputeEnvironment represents a Batch compute environment.
type ComputeEnvironment struct {
	ComputeEnvironmentName string
	ComputeEnvironmentArn  string
	Type                   string // MANAGED or UNMANAGED
	State                  string // ENABLED or DISABLED
	Status                 string
	StatusReason           string
	ComputeResources       *ComputeResource
	ServiceRole            string
	CreatedAt              time.Time
	lifecycle              *lifecycle.Machine
}

// ComputeResource describes the compute resources in a managed compute environment.
type ComputeResource struct {
	Type               string
	MinvCpus           int
	MaxvCpus           int
	DesiredvCpus       int
	InstanceTypes      []string
	Subnets            []string
	SecurityGroupIds   []string
	InstanceRole       string
}

// JobQueue represents a Batch job queue.
type JobQueue struct {
	JobQueueName                 string
	JobQueueArn                  string
	State                        string
	Status                       string
	StatusReason                 string
	Priority                     int
	ComputeEnvironmentOrder      []ComputeEnvironmentOrder
	CreatedAt                    time.Time
}

// ComputeEnvironmentOrder defines the order of compute environments in a job queue.
type ComputeEnvironmentOrder struct {
	ComputeEnvironment string
	Order              int
}

// JobDefinition represents a Batch job definition.
type JobDefinition struct {
	JobDefinitionName string
	JobDefinitionArn  string
	Revision          int
	Type              string
	Status            string
	ContainerProperties *ContainerProperties
	RetryStrategy      *RetryStrategy
	Timeout            *JobTimeout
	CreatedAt          time.Time
}

// ContainerProperties describes the container for a job definition.
type ContainerProperties struct {
	Image      string
	Vcpus      int
	Memory     int
	Command    []string
	JobRoleArn string
	Environment []KeyValuePair
}

// KeyValuePair represents a key-value pair.
type KeyValuePair struct {
	Name  string
	Value string
}

// RetryStrategy describes the retry strategy for a job.
type RetryStrategy struct {
	Attempts int
}

// JobTimeout describes the timeout for a job.
type JobTimeout struct {
	AttemptDurationSeconds int
}

// Job represents a Batch job.
type Job struct {
	JobName          string
	JobID            string
	JobArn           string
	JobQueue         string
	JobDefinition    string
	Status           string
	StatusReason     string
	CreatedAt        time.Time
	StartedAt        *time.Time
	StoppedAt        *time.Time
	Container        *ContainerDetail
	RetryStrategy    *RetryStrategy
	Timeout          *JobTimeout
	lifecycle        *lifecycle.Machine
}

// ContainerDetail describes a running container.
type ContainerDetail struct {
	Image       string
	Vcpus       int
	Memory      int
	Command     []string
	ExitCode    *int
	Reason      string
	LogStreamName string
}

// ServiceLocator resolves other services for cross-service integration.
type ServiceLocator interface {
	Lookup(name string) (interface{ HandleRequest(ctx interface{}) (interface{}, error) }, error)
}

// SchedulingPolicy represents a Batch fair-share scheduling policy.
type SchedulingPolicy struct {
	Arn         string
	Name        string
	FairsharePolicy *FairsharePolicy
	Tags        map[string]string
	CreatedAt   time.Time
}

// FairsharePolicy defines fair-share parameters for a scheduling policy.
type FairsharePolicy struct {
	ComputeReservation int
	ShareDecaySeconds  int
	ShareDistributions []ShareDistribution
}

// ShareDistribution defines a share for a single compute identifier.
type ShareDistribution struct {
	ShareIdentifier string
	WeightFactor    float64
}

// Store manages Batch resources in memory.
type Store struct {
	mu                sync.RWMutex
	computeEnvs       map[string]*ComputeEnvironment
	jobQueues         map[string]*JobQueue
	jobDefs           map[string]*JobDefinition
	jobs              map[string]*Job
	schedulingPolicies map[string]*SchedulingPolicy // arn -> policy
	tags              map[string]map[string]string   // arn -> tags
	accountID         string
	region            string
	lcConfig          *lifecycle.Config
	jobSeq            int
	defRevision       map[string]int // jobDefName -> latest revision
}

// NewStore returns a new empty Batch Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		computeEnvs:        make(map[string]*ComputeEnvironment),
		jobQueues:          make(map[string]*JobQueue),
		jobDefs:            make(map[string]*JobDefinition),
		jobs:               make(map[string]*Job),
		schedulingPolicies: make(map[string]*SchedulingPolicy),
		tags:               make(map[string]map[string]string),
		accountID:          accountID,
		region:             region,
		lcConfig:           lifecycle.DefaultConfig(),
		defRevision:        make(map[string]int),
	}
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:batch:%s:%s:", s.region, s.accountID)
}

// CreateComputeEnvironment creates a compute environment.
func (s *Store) CreateComputeEnvironment(name, ceType, state, serviceRole string, resources *ComputeResource) (*ComputeEnvironment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.computeEnvs[name]; ok {
		return nil, fmt.Errorf("compute environment already exists: %s", name)
	}

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "VALID", Delay: 2 * time.Second},
	}

	ce := &ComputeEnvironment{
		ComputeEnvironmentName: name,
		ComputeEnvironmentArn:  s.arnPrefix() + "compute-environment/" + name,
		Type:                   ceType,
		State:                  state,
		Status:                 "CREATING",
		StatusReason:           "Creating compute environment",
		ComputeResources:       resources,
		ServiceRole:            serviceRole,
		CreatedAt:              time.Now().UTC(),
	}
	ce.lifecycle = lifecycle.NewMachine("CREATING", transitions, s.lcConfig)
	ce.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		ce.Status = string(to)
		if to == "VALID" {
			ce.StatusReason = "ComputeEnvironment is VALID"
		}
	})

	s.computeEnvs[name] = ce
	return ce, nil
}

// GetComputeEnvironment retrieves a compute environment.
func (s *Store) GetComputeEnvironment(name string) (*ComputeEnvironment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ce, ok := s.computeEnvs[name]
	return ce, ok
}

// ListComputeEnvironments returns all compute environments.
func (s *Store) ListComputeEnvironments() []*ComputeEnvironment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ComputeEnvironment, 0, len(s.computeEnvs))
	for _, ce := range s.computeEnvs {
		out = append(out, ce)
	}
	return out
}

// DeleteComputeEnvironment removes a compute environment.
func (s *Store) DeleteComputeEnvironment(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ce, ok := s.computeEnvs[name]
	if !ok {
		return false
	}
	if ce.lifecycle != nil {
		ce.lifecycle.Stop()
	}
	delete(s.computeEnvs, name)
	return true
}

// CreateJobQueue creates a job queue.
func (s *Store) CreateJobQueue(name, state string, priority int, computeEnvOrder []ComputeEnvironmentOrder) (*JobQueue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobQueues[name]; ok {
		return nil, fmt.Errorf("job queue already exists: %s", name)
	}

	jq := &JobQueue{
		JobQueueName:            name,
		JobQueueArn:             s.arnPrefix() + "job-queue/" + name,
		State:                   state,
		Status:                  "VALID",
		Priority:                priority,
		ComputeEnvironmentOrder: computeEnvOrder,
		CreatedAt:               time.Now().UTC(),
	}
	s.jobQueues[name] = jq
	return jq, nil
}

// GetJobQueue retrieves a job queue.
func (s *Store) GetJobQueue(name string) (*JobQueue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jq, ok := s.jobQueues[name]
	return jq, ok
}

// ListJobQueues returns all job queues.
func (s *Store) ListJobQueues() []*JobQueue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*JobQueue, 0, len(s.jobQueues))
	for _, jq := range s.jobQueues {
		out = append(out, jq)
	}
	return out
}

// DeleteJobQueue removes a job queue.
func (s *Store) DeleteJobQueue(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobQueues[name]; !ok {
		return false
	}
	delete(s.jobQueues, name)
	return true
}

// RegisterJobDefinition registers a new job definition.
func (s *Store) RegisterJobDefinition(name, jobType string, container *ContainerProperties, retry *RetryStrategy, timeout *JobTimeout) (*JobDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.defRevision[name]++
	revision := s.defRevision[name]

	defKey := fmt.Sprintf("%s:%d", name, revision)
	jd := &JobDefinition{
		JobDefinitionName: name,
		JobDefinitionArn:  s.arnPrefix() + "job-definition/" + defKey,
		Revision:          revision,
		Type:              jobType,
		Status:            "ACTIVE",
		ContainerProperties: container,
		RetryStrategy:      retry,
		Timeout:            timeout,
		CreatedAt:          time.Now().UTC(),
	}
	s.jobDefs[defKey] = jd
	return jd, nil
}

// GetJobDefinition retrieves a job definition.
func (s *Store) GetJobDefinition(nameOrArn string) (*JobDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if jd, ok := s.jobDefs[nameOrArn]; ok {
		return jd, true
	}
	// Search by ARN
	for _, jd := range s.jobDefs {
		if jd.JobDefinitionArn == nameOrArn {
			return jd, true
		}
	}
	return nil, false
}

// ListJobDefinitions returns all job definitions.
func (s *Store) ListJobDefinitions() []*JobDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*JobDefinition, 0, len(s.jobDefs))
	for _, jd := range s.jobDefs {
		out = append(out, jd)
	}
	return out
}

// DeregisterJobDefinition deactivates a job definition.
func (s *Store) DeregisterJobDefinition(defArn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, jd := range s.jobDefs {
		if jd.JobDefinitionArn == defArn {
			jd.Status = "INACTIVE"
			return true
		}
	}
	return false
}

// SubmitJob submits a new job.
func (s *Store) SubmitJob(name, queue, jobDef string, container *ContainerDetail, retry *RetryStrategy, timeout *JobTimeout) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobSeq++
	jobID := fmt.Sprintf("job-%012d", s.jobSeq)

	transitions := []lifecycle.Transition{
		{From: "SUBMITTED", To: "PENDING", Delay: 500 * time.Millisecond},
		{From: "PENDING", To: "RUNNABLE", Delay: 1 * time.Second},
		{From: "RUNNABLE", To: "STARTING", Delay: 1 * time.Second},
		{From: "STARTING", To: "RUNNING", Delay: 1 * time.Second},
		{From: "RUNNING", To: "SUCCEEDED", Delay: 5 * time.Second},
	}

	now := time.Now().UTC()

	// Generate log stream name for the job.
	if container != nil && container.LogStreamName == "" {
		container.LogStreamName = fmt.Sprintf("batch/job/%s/%s", name, jobID)
	}

	job := &Job{
		JobName:       name,
		JobID:         jobID,
		JobArn:        s.arnPrefix() + "job/" + jobID,
		JobQueue:      queue,
		JobDefinition: jobDef,
		Status:        "SUBMITTED",
		StatusReason:  "Job submitted",
		CreatedAt:     now,
		Container:     container,
		RetryStrategy: retry,
		Timeout:       timeout,
	}
	job.lifecycle = lifecycle.NewMachine("SUBMITTED", transitions, s.lcConfig)
	job.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		job.Status = string(to)
		now := time.Now().UTC()
		if to == "RUNNING" {
			job.StartedAt = &now
			job.StatusReason = "Essential container in task exited"
		}
		if to == "SUCCEEDED" {
			job.StoppedAt = &now
			job.StatusReason = "Essential container in task exited"
			if job.Container != nil {
				exitCode := 0
				job.Container.ExitCode = &exitCode
			}
		}
	})

	s.jobs[jobID] = job
	return job, nil
}

// GetJob retrieves a job.
func (s *Store) GetJob(jobID string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[jobID]
	if ok && job.lifecycle != nil {
		job.Status = string(job.lifecycle.State())
	}
	return job, ok
}

// ListJobs returns jobs, optionally filtered by queue and status.
func (s *Store) ListJobs(queue, status string) []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0)
	for _, job := range s.jobs {
		if job.lifecycle != nil {
			job.Status = string(job.lifecycle.State())
		}
		if queue != "" && job.JobQueue != queue {
			continue
		}
		if status != "" && job.Status != status {
			continue
		}
		out = append(out, job)
	}
	return out
}

// CancelJob cancels a job.
func (s *Store) CancelJob(jobID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}
	if job.Status == "SUCCEEDED" || job.Status == "FAILED" {
		return fmt.Errorf("job %s is already in terminal state: %s", jobID, job.Status)
	}
	if job.lifecycle != nil {
		job.lifecycle.Stop()
	}
	job.Status = "FAILED"
	job.StatusReason = reason
	now := time.Now().UTC()
	job.StoppedAt = &now
	return nil
}

// TerminateJob terminates a running job.
func (s *Store) TerminateJob(jobID, reason string) error {
	return s.CancelJob(jobID, reason)
}

// GetComputeEnvironmentVCPUs returns available and desired vCPUs for a compute environment.
func (s *Store) GetComputeEnvironmentVCPUs(name string) (available, desired int, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ce, exists := s.computeEnvs[name]
	if !exists || ce.ComputeResources == nil {
		return 0, 0, false
	}
	// Count running jobs that use this CE's queues.
	usedVCPUs := 0
	for _, job := range s.jobs {
		if job.Status == "RUNNING" && job.Container != nil {
			usedVCPUs += job.Container.Vcpus
		}
	}
	return ce.ComputeResources.MaxvCpus - usedVCPUs, ce.ComputeResources.DesiredvCpus, true
}

// GetJobDefRevisions returns all revisions for a given job definition name.
func (s *Store) GetJobDefRevisions(name string) []*JobDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var revisions []*JobDefinition
	for _, jd := range s.jobDefs {
		if jd.JobDefinitionName == name {
			revisions = append(revisions, jd)
		}
	}
	return revisions
}

// UpdateComputeEnvironment updates a compute environment's state or service role.
func (s *Store) UpdateComputeEnvironment(name, state, serviceRole string) (*ComputeEnvironment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ce, ok := s.computeEnvs[name]
	if !ok {
		return nil, false
	}
	if state != "" {
		ce.State = state
	}
	if serviceRole != "" {
		ce.ServiceRole = serviceRole
	}
	return ce, true
}

// UpdateJobQueue updates a job queue's state, priority, or compute environment order.
func (s *Store) UpdateJobQueue(name, state string, priority int, ceOrder []ComputeEnvironmentOrder) (*JobQueue, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jq, ok := s.jobQueues[name]
	if !ok {
		return nil, false
	}
	if state != "" {
		jq.State = state
	}
	if priority > 0 {
		jq.Priority = priority
	}
	if len(ceOrder) > 0 {
		jq.ComputeEnvironmentOrder = ceOrder
	}
	return jq, true
}

// CreateSchedulingPolicy creates a new scheduling policy.
func (s *Store) CreateSchedulingPolicy(name string, fsp *FairsharePolicy, tags map[string]string) (*SchedulingPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arn := fmt.Sprintf("arn:aws:batch:%s:%s:scheduling-policy/%s", s.region, s.accountID, name)
	if _, ok := s.schedulingPolicies[arn]; ok {
		return nil, fmt.Errorf("scheduling policy already exists: %s", name)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	sp := &SchedulingPolicy{
		Arn:             arn,
		Name:            name,
		FairsharePolicy: fsp,
		Tags:            tags,
		CreatedAt:       time.Now().UTC(),
	}
	s.schedulingPolicies[arn] = sp
	s.tags[arn] = tags
	return sp, nil
}

// DescribeSchedulingPolicies returns scheduling policies by ARN.
func (s *Store) DescribeSchedulingPolicies(arns []string) []*SchedulingPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(arns) == 0 {
		out := make([]*SchedulingPolicy, 0, len(s.schedulingPolicies))
		for _, sp := range s.schedulingPolicies {
			out = append(out, sp)
		}
		return out
	}
	out := make([]*SchedulingPolicy, 0)
	for _, arn := range arns {
		if sp, ok := s.schedulingPolicies[arn]; ok {
			out = append(out, sp)
		}
	}
	return out
}

// UpdateSchedulingPolicy updates a scheduling policy's fair-share config.
func (s *Store) UpdateSchedulingPolicy(arn string, fsp *FairsharePolicy) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.schedulingPolicies[arn]
	if !ok {
		return false
	}
	if fsp != nil {
		sp.FairsharePolicy = fsp
	}
	return true
}

// DeleteSchedulingPolicy removes a scheduling policy.
func (s *Store) DeleteSchedulingPolicy(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.schedulingPolicies[arn]; !ok {
		return false
	}
	delete(s.schedulingPolicies, arn)
	delete(s.tags, arn)
	return true
}

// TagResource applies tags to any resource by ARN.
func (s *Store) TagResource(resourceARN string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tags[resourceARN]; !ok {
		s.tags[resourceARN] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[resourceARN][k] = v
	}
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(resourceARN string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tags[resourceARN]; ok {
		for _, k := range keys {
			delete(t, k)
		}
	}
}

// ListTagsForResource returns all tags for a resource by ARN.
func (s *Store) ListTagsForResource(resourceARN string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range s.tags[resourceARN] {
		result[k] = v
	}
	return result
}
