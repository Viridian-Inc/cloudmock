package elasticmapreduce

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Cluster represents an EMR cluster.
type Cluster struct {
	ID                    string
	Name                  string
	ARN                   string
	Status                ClusterStatus
	ReleaseLabel          string
	Applications          []Application
	LogUri                string
	ServiceRole           string
	JobFlowRole           string
	MasterPublicDnsName   string
	NormalizedInstanceHours int
	VisibleToAllUsers     bool
	TerminationProtected  bool
	AutoTerminate         bool
	Tags                  map[string]string
	Lifecycle             *lifecycle.Machine
}

// ClusterStatus holds cluster state info.
type ClusterStatus struct {
	State string
}

// Application represents an installed application.
type Application struct {
	Name    string
	Version string
}

// Step represents an EMR step.
type Step struct {
	ID      string
	Name    string
	Status  StepStatus
	Config  HadoopJarStepConfig
}

// StepStatus holds step state.
type StepStatus struct {
	State string
}

// HadoopJarStepConfig holds step configuration.
type HadoopJarStepConfig struct {
	Jar        string
	MainClass  string
	Args       []string
	Properties map[string]string
}

// InstanceGroup represents an EMR instance group.
type InstanceGroup struct {
	ID            string
	Name          string
	ClusterID     string
	InstanceRole  string // MASTER, CORE, TASK
	InstanceType  string
	InstanceCount int
	Market        string
	Status        InstanceGroupStatus
}

// InstanceGroupStatus holds instance group state.
type InstanceGroupStatus struct {
	State string
}

// Store manages all EMR resources in memory.
type Store struct {
	mu             sync.RWMutex
	clusters       map[string]*Cluster
	steps          map[string][]*Step          // clusterID -> steps
	instanceGroups map[string][]*InstanceGroup // clusterID -> groups
	accountID      string
	region         string
	lcConfig       *lifecycle.Config
}

// NewStore creates a new EMR Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:       make(map[string]*Cluster),
		steps:          make(map[string][]*Step),
		instanceGroups: make(map[string][]*InstanceGroup),
		accountID:      accountID,
		region:         region,
		lcConfig:       lifecycle.DefaultConfig(),
	}
}

func newID() string {
	b := make([]byte, 13)
	_, _ = rand.Read(b)
	return fmt.Sprintf("j-%026x", b)[:15]
}

func newStepID() string {
	b := make([]byte, 13)
	_, _ = rand.Read(b)
	return fmt.Sprintf("s-%026x", b)[:15]
}

func newIGID() string {
	b := make([]byte, 13)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ig-%026x", b)[:16]
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) clusterARN(id string) string {
	return fmt.Sprintf("arn:aws:elasticmapreduce:%s:%s:cluster/%s", s.region, s.accountID, id)
}

func clusterTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "STARTING", To: "BOOTSTRAPPING", Delay: 2 * time.Second},
		{From: "BOOTSTRAPPING", To: "RUNNING", Delay: 3 * time.Second},
		{From: "TERMINATING", To: "TERMINATED", Delay: 2 * time.Second},
	}
}

func (s *Store) RunJobFlow(name, releaseLabel, logUri, serviceRole, jobFlowRole string, apps []Application, tags map[string]string) *Cluster {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	c := &Cluster{
		ID:                  id,
		Name:                name,
		ARN:                 s.clusterARN(id),
		ReleaseLabel:        releaseLabel,
		Applications:        apps,
		LogUri:              logUri,
		ServiceRole:         serviceRole,
		JobFlowRole:         jobFlowRole,
		MasterPublicDnsName: fmt.Sprintf("ec2-%s.compute-1.amazonaws.com", id),
		VisibleToAllUsers:   true,
		Tags:                tags,
		Lifecycle:           lifecycle.NewMachine("STARTING", clusterTransitions(), s.lcConfig),
		Status:              ClusterStatus{State: "STARTING"},
	}
	s.clusters[id] = c
	s.steps[id] = make([]*Step, 0)
	s.instanceGroups[id] = make([]*InstanceGroup, 0)
	return c
}

func (s *Store) GetCluster(id string) (*Cluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[id]
	if ok {
		c.Status.State = string(c.Lifecycle.State())
	}
	return c, ok
}

func (s *Store) ListClusters(states []string) []*Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Cluster, 0)
	stateSet := make(map[string]bool)
	for _, st := range states {
		stateSet[st] = true
	}
	for _, c := range s.clusters {
		c.Status.State = string(c.Lifecycle.State())
		if len(stateSet) == 0 || stateSet[c.Status.State] {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) TerminateJobFlows(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if c, ok := s.clusters[id]; ok {
			if !c.TerminationProtected {
				c.Lifecycle.ForceState("TERMINATING")
			}
		}
	}
}

func (s *Store) SetTerminationProtection(ids []string, protected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if c, ok := s.clusters[id]; ok {
			c.TerminationProtected = protected
		}
	}
}

func (s *Store) SetVisibleToAllUsers(ids []string, visible bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if c, ok := s.clusters[id]; ok {
			c.VisibleToAllUsers = visible
		}
	}
}

// ---- Step operations ----

func (s *Store) AddSteps(clusterID string, steps []Step) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clusters[clusterID]; !ok {
		return nil, false
	}
	ids := make([]string, len(steps))
	for i := range steps {
		steps[i].ID = newStepID()
		steps[i].Status.State = "COMPLETED"
		ids[i] = steps[i].ID
		step := steps[i]
		s.steps[clusterID] = append(s.steps[clusterID], &step)
	}
	return ids, true
}

func (s *Store) ListSteps(clusterID string) []*Step {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.steps[clusterID]
}

func (s *Store) GetStep(clusterID, stepID string) (*Step, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, step := range s.steps[clusterID] {
		if step.ID == stepID {
			return step, true
		}
	}
	return nil, false
}

// ---- InstanceGroup operations ----

func (s *Store) AddInstanceGroups(clusterID string, groups []InstanceGroup) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clusters[clusterID]; !ok {
		return nil, false
	}
	ids := make([]string, len(groups))
	for i := range groups {
		groups[i].ID = newIGID()
		groups[i].ClusterID = clusterID
		groups[i].Status.State = "RUNNING"
		ids[i] = groups[i].ID
		g := groups[i]
		s.instanceGroups[clusterID] = append(s.instanceGroups[clusterID], &g)
	}
	return ids, true
}

func (s *Store) ListInstanceGroups(clusterID string) []*InstanceGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.instanceGroups[clusterID]
}

func (s *Store) ModifyInstanceGroups(clusterID string, modifications map[string]int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups := s.instanceGroups[clusterID]
	for _, g := range groups {
		if count, ok := modifications[g.ID]; ok {
			g.InstanceCount = count
		}
	}
	return true
}

// ---- Tag operations ----

func (s *Store) AddTags(resourceID string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[resourceID]
	if !ok {
		return false
	}
	for k, v := range tags {
		c.Tags[k] = v
	}
	return true
}

func (s *Store) RemoveTags(resourceID string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[resourceID]
	if !ok {
		return false
	}
	for _, k := range keys {
		delete(c.Tags, k)
	}
	return true
}
