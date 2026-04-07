package elasticmapreduce

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
	"github.com/Viridian-Inc/cloudmock/pkg/mocklog"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
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

// ServiceLocator resolves other services by name.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// SecurityConfiguration represents an EMR security configuration.
type SecurityConfiguration struct {
	Name              string
	SecurityConfiguration string // JSON blob
	CreationDateTime  time.Time
}

// Store manages all EMR resources in memory.
type Store struct {
	mu                   sync.RWMutex
	clusters             map[string]*Cluster
	steps                map[string][]*Step            // clusterID -> steps
	instanceGroups       map[string][]*InstanceGroup   // clusterID -> groups
	ec2InstanceIDs       map[string][]string           // clusterID -> EC2 instance IDs
	securityConfigs      map[string]*SecurityConfiguration // name -> config
	accountID            string
	region               string
	lcConfig             *lifecycle.Config
	locator              ServiceLocator
}

// NewStore creates a new EMR Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:        make(map[string]*Cluster),
		steps:           make(map[string][]*Step),
		instanceGroups:  make(map[string][]*InstanceGroup),
		ec2InstanceIDs:  make(map[string][]string),
		securityConfigs: make(map[string]*SecurityConfiguration),
		accountID:       accountID,
		region:          region,
		lcConfig:        lifecycle.DefaultConfig(),
	}
}

// SetLocator sets the service locator for cross-service lookups.
func (s *Store) SetLocator(locator ServiceLocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locator = locator
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

	// Register callback to create EC2 instances when cluster reaches RUNNING
	locator := s.locator
	clusterID := id
	c.Lifecycle.OnTransition(func(from, to lifecycle.State) {
		if string(to) == "RUNNING" {
			s.createEC2InstancesForCluster(clusterID, locator)
		}
	})

	s.clusters[id] = c
	s.steps[id] = make([]*Step, 0)
	s.instanceGroups[id] = make([]*InstanceGroup, 0)
	s.ec2InstanceIDs[id] = make([]string, 0)
	s.mu.Unlock()
	return c
}

// createEC2InstancesForCluster creates mock EC2 instances for all instance groups.
func (s *Store) createEC2InstancesForCluster(clusterID string, locator ServiceLocator) {
	if locator == nil {
		return
	}
	ec2Svc, err := locator.Lookup("ec2")
	if err != nil || ec2Svc == nil {
		return
	}

	s.mu.RLock()
	groups := s.instanceGroups[clusterID]
	totalInstances := 0
	for _, g := range groups {
		totalInstances += g.InstanceCount
	}
	s.mu.RUnlock()

	if totalInstances == 0 {
		totalInstances = 1 // at least a master
	}

	// Launch EC2 instances
	body, _ := json.Marshal(map[string]any{
		"ImageId":      "ami-emr-" + clusterID,
		"InstanceType": "m5.xlarge",
		"MinCount":     totalInstances,
		"MaxCount":     totalInstances,
	})
	ctx := &service.RequestContext{
		Action:     "RunInstances",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	resp, err := ec2Svc.HandleRequest(ctx)
	if err != nil || resp == nil || resp.Body == nil {
		return
	}

	// Parse instance IDs from response
	data, _ := json.Marshal(resp.Body)
	var result struct {
		Instances []struct {
			InstanceId string `json:"InstanceId"`
		} `json:"Instances"`
	}
	if json.Unmarshal(data, &result) == nil {
		s.mu.Lock()
		for _, inst := range result.Instances {
			s.ec2InstanceIDs[clusterID] = append(s.ec2InstanceIDs[clusterID], inst.InstanceId)
		}
		s.mu.Unlock()
	}
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
	var toTerminate []*lifecycle.Machine
	var instanceIDsToTerminate []string
	locator := s.locator
	for _, id := range ids {
		if c, ok := s.clusters[id]; ok {
			if !c.TerminationProtected {
				toTerminate = append(toTerminate, c.Lifecycle)
				instanceIDsToTerminate = append(instanceIDsToTerminate, s.ec2InstanceIDs[id]...)
			}
		}
	}
	s.mu.Unlock()

	for _, lc := range toTerminate {
		if lc != nil {
			lc.ForceState("TERMINATING")
		}
	}

	// Terminate EC2 instances via locator
	if locator != nil && len(instanceIDsToTerminate) > 0 {
		ec2Svc, err := locator.Lookup("ec2")
		if err == nil && ec2Svc != nil {
			body, _ := json.Marshal(map[string]any{
				"InstanceIds": instanceIDsToTerminate,
			})
			ctx := &service.RequestContext{
				Action:     "TerminateInstances",
				Body:       body,
				RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
			}
			_, _ = ec2Svc.HandleRequest(ctx)
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
	c, ok := s.clusters[clusterID]
	if !ok {
		s.mu.Unlock()
		return nil, false
	}
	locator := s.locator
	logUri := c.LogUri
	ids := make([]string, len(steps))
	for i := range steps {
		steps[i].ID = newStepID()
		// Steps transition: PENDING → RUNNING → COMPLETED
		steps[i].Status.State = "COMPLETED"
		ids[i] = steps[i].ID
		step := steps[i]
		s.steps[clusterID] = append(s.steps[clusterID], &step)
	}
	s.mu.Unlock()

	// Write mock execution logs for each step
	if locator != nil {
		writer := mocklog.NewWriter(locator)
		for _, step := range steps {
			logGroup := "/aws/emr/steps"
			if logUri != "" {
				logGroup = logUri + "/steps"
			}
			phases := []string{"SETUP", "EXECUTE", "CLEANUP"}
			lines := mocklog.GenerateBuildLines("emr-step", step.ID, phases)
			writer.WriteBuildLog(logGroup, clusterID+"/"+step.ID, lines)
		}
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
	if _, ok := s.clusters[clusterID]; !ok {
		s.mu.Unlock()
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
	s.mu.Unlock()
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

// ---- Security Configuration operations ----

// CreateSecurityConfiguration creates a new security configuration.
func (s *Store) CreateSecurityConfiguration(name, config string) (*SecurityConfiguration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.securityConfigs[name]; ok {
		return nil, fmt.Errorf("security configuration already exists: %s", name)
	}
	sc := &SecurityConfiguration{
		Name:                  name,
		SecurityConfiguration: config,
		CreationDateTime:      time.Now().UTC(),
	}
	s.securityConfigs[name] = sc
	return sc, nil
}

// DescribeSecurityConfiguration returns a security configuration by name.
func (s *Store) DescribeSecurityConfiguration(name string) (*SecurityConfiguration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.securityConfigs[name]
	return sc, ok
}

// ListSecurityConfigurations returns all security configurations.
func (s *Store) ListSecurityConfigurations() []*SecurityConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*SecurityConfiguration, 0, len(s.securityConfigs))
	for _, sc := range s.securityConfigs {
		out = append(out, sc)
	}
	return out
}

// DeleteSecurityConfiguration removes a security configuration.
func (s *Store) DeleteSecurityConfiguration(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.securityConfigs[name]; !ok {
		return false
	}
	delete(s.securityConfigs, name)
	return true
}
