package dms

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// ReplicationInstance represents a DMS replication instance.
type ReplicationInstance struct {
	ReplicationInstanceIdentifier string
	ReplicationInstanceArn        string
	ReplicationInstanceClass      string
	AllocatedStorage              int
	EngineVersion                 string
	AutoMinorVersionUpgrade       bool
	AvailabilityZone              string
	MultiAZ                       bool
	PubliclyAccessible            bool
	ReplicationInstanceStatus     string
	InstanceCreateTime            time.Time
	lifecycle                     *lifecycle.Machine
}

// Endpoint represents a DMS endpoint.
type Endpoint struct {
	EndpointIdentifier string
	EndpointArn        string
	EndpointType       string // "source" or "target"
	EngineName         string
	ServerName         string
	Port               int
	DatabaseName       string
	Username           string
	Status             string
	CreatedAt          time.Time
}

// ConnectionTest tracks a test-connection result.
type ConnectionTest struct {
	EndpointArn            string
	ReplicationInstanceArn string
	Status                 string // successful, testing, failed
	FailureMessage         string
	TestedAt               time.Time
}

// ReplicationTask represents a DMS replication task.
type ReplicationTask struct {
	ReplicationTaskIdentifier string
	ReplicationTaskArn        string
	SourceEndpointArn         string
	TargetEndpointArn         string
	ReplicationInstanceArn    string
	MigrationType             string
	TableMappings             string
	Status                    string
	CreatedAt                 time.Time
	StartedAt                 *time.Time
	StoppedAt                 *time.Time
	lifecycle                 *lifecycle.Machine
	// Table statistics tracked during running state
	TablesLoaded  int
	TablesLoading int
	TablesErrored int
}

// EventSubscription represents a DMS event subscription.
type EventSubscription struct {
	CustSubscriptionId string
	SnsTopicArn        string
	SourceType         string
	SourceIds          []string
	EventCategories    []string
	Status             string
	CreatedAt          time.Time
}

// Store manages DMS resources in memory.
type Store struct {
	mu              sync.RWMutex
	instances       map[string]*ReplicationInstance
	endpoints       map[string]*Endpoint
	tasks           map[string]*ReplicationTask
	subscriptions   map[string]*EventSubscription
	connections     []ConnectionTest
	accountID       string
	region          string
	lcConfig        *lifecycle.Config
}

// NewStore returns a new empty DMS Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		instances:     make(map[string]*ReplicationInstance),
		endpoints:     make(map[string]*Endpoint),
		tasks:         make(map[string]*ReplicationTask),
		subscriptions: make(map[string]*EventSubscription),
		accountID:     accountID,
		region:        region,
		lcConfig:      lifecycle.DefaultConfig(),
	}
}

func (s *Store) arnPrefix() string {
	return fmt.Sprintf("arn:aws:dms:%s:%s:", s.region, s.accountID)
}

// CreateReplicationInstance creates a new replication instance.
func (s *Store) CreateReplicationInstance(id, class string, storage int, engineVersion, az string, multiAZ, publicAccess bool) (*ReplicationInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.instances[id]; ok {
		return nil, fmt.Errorf("replication instance already exists: %s", id)
	}

	transitions := []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 3 * time.Second},
	}

	inst := &ReplicationInstance{
		ReplicationInstanceIdentifier: id,
		ReplicationInstanceArn:        s.arnPrefix() + "rep:" + id,
		ReplicationInstanceClass:      class,
		AllocatedStorage:              storage,
		EngineVersion:                 engineVersion,
		AutoMinorVersionUpgrade:       true,
		AvailabilityZone:              az,
		MultiAZ:                       multiAZ,
		PubliclyAccessible:            publicAccess,
		ReplicationInstanceStatus:     "creating",
		InstanceCreateTime:            time.Now().UTC(),
	}
	inst.lifecycle = lifecycle.NewMachine("creating", transitions, s.lcConfig)
	inst.lifecycle.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		inst.ReplicationInstanceStatus = string(to)
	})

	s.instances[id] = inst
	return inst, nil
}

// GetReplicationInstance retrieves a replication instance by identifier.
func (s *Store) GetReplicationInstance(id string) (*ReplicationInstance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inst, ok := s.instances[id]
	return inst, ok
}

// ListReplicationInstances returns all replication instances.
func (s *Store) ListReplicationInstances() []*ReplicationInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ReplicationInstance, 0, len(s.instances))
	for _, inst := range s.instances {
		out = append(out, inst)
	}
	return out
}

// DeleteReplicationInstance removes a replication instance.
func (s *Store) DeleteReplicationInstance(id string) (*ReplicationInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst, ok := s.instances[id]
	if !ok {
		return nil, false
	}
	inst.lifecycle.Stop()
	inst.ReplicationInstanceStatus = "deleting"
	delete(s.instances, id)
	return inst, true
}

// CreateEndpoint creates a new endpoint.
func (s *Store) CreateEndpoint(id, endpointType, engine, server string, port int, db, username string) (*Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.endpoints[id]; ok {
		return nil, fmt.Errorf("endpoint already exists: %s", id)
	}

	ep := &Endpoint{
		EndpointIdentifier: id,
		EndpointArn:        s.arnPrefix() + "endpoint:" + id,
		EndpointType:       endpointType,
		EngineName:         engine,
		ServerName:         server,
		Port:               port,
		DatabaseName:       db,
		Username:           username,
		Status:             "active",
		CreatedAt:          time.Now().UTC(),
	}
	s.endpoints[id] = ep
	return ep, nil
}

// GetEndpoint retrieves an endpoint by identifier.
func (s *Store) GetEndpoint(id string) (*Endpoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep, ok := s.endpoints[id]
	return ep, ok
}

// ListEndpoints returns all endpoints.
func (s *Store) ListEndpoints() []*Endpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Endpoint, 0, len(s.endpoints))
	for _, ep := range s.endpoints {
		out = append(out, ep)
	}
	return out
}

// DeleteEndpoint removes an endpoint.
func (s *Store) DeleteEndpoint(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.endpoints[id]; !ok {
		return false
	}
	delete(s.endpoints, id)
	return true
}

// CreateReplicationTask creates a new replication task.
func (s *Store) CreateReplicationTask(id, sourceArn, targetArn, instanceArn, migrationType, tableMappings string) (*ReplicationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; ok {
		return nil, fmt.Errorf("replication task already exists: %s", id)
	}

	task := &ReplicationTask{
		ReplicationTaskIdentifier: id,
		ReplicationTaskArn:        s.arnPrefix() + "task:" + id,
		SourceEndpointArn:         sourceArn,
		TargetEndpointArn:         targetArn,
		ReplicationInstanceArn:    instanceArn,
		MigrationType:             migrationType,
		TableMappings:             tableMappings,
		Status:                    "ready",
		CreatedAt:                 time.Now().UTC(),
	}
	s.tasks[id] = task
	return task, nil
}

// GetReplicationTask retrieves a replication task by identifier.
func (s *Store) GetReplicationTask(id string) (*ReplicationTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

// ListReplicationTasks returns all replication tasks.
func (s *Store) ListReplicationTasks() []*ReplicationTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ReplicationTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		out = append(out, task)
	}
	return out
}

// StartReplicationTask starts a replication task.
func (s *Store) StartReplicationTask(arn string) (*ReplicationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		if task.ReplicationTaskArn == arn {
			if task.Status != "ready" && task.Status != "stopped" {
				return nil, fmt.Errorf("task %s is not in a startable state: %s", arn, task.Status)
			}
			now := time.Now().UTC()
			task.StartedAt = &now
			task.Status = "starting"

			transitions := []lifecycle.Transition{
				{From: "starting", To: "running", Delay: 2 * time.Second},
			}
			task.lifecycle = lifecycle.NewMachine("starting", transitions, s.lcConfig)
			task.lifecycle.OnTransition(func(from, to lifecycle.State) {
				s.mu.Lock()
				defer s.mu.Unlock()
				task.Status = string(to)
				if string(to) == "running" {
					// Simulate table loading completion.
					task.TablesLoaded = 5
					task.TablesLoading = 0
					task.TablesErrored = 0
				}
			})
			// Handle instant mode transition.
			if string(task.lifecycle.State()) == "running" {
				task.Status = "running"
				task.TablesLoaded = 5
				task.TablesLoading = 0
				task.TablesErrored = 0
			}
			return task, nil
		}
	}
	return nil, fmt.Errorf("replication task not found: %s", arn)
}

// StopReplicationTask stops a replication task.
func (s *Store) StopReplicationTask(arn string) (*ReplicationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		if task.ReplicationTaskArn == arn {
			if task.Status != "running" {
				return nil, fmt.Errorf("task %s is not running: %s", arn, task.Status)
			}
			if task.lifecycle != nil {
				task.lifecycle.Stop()
			}
			now := time.Now().UTC()
			task.StoppedAt = &now
			task.Status = "stopped"
			return task, nil
		}
	}
	return nil, fmt.Errorf("replication task not found: %s", arn)
}

// DeleteReplicationTask removes a replication task.
func (s *Store) DeleteReplicationTask(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, task := range s.tasks {
		if task.ReplicationTaskArn == arn {
			if task.lifecycle != nil {
				task.lifecycle.Stop()
			}
			delete(s.tasks, id)
			return true
		}
	}
	return false
}

// CreateEventSubscription creates a new event subscription.
func (s *Store) CreateEventSubscription(id, snsTopicArn, sourceType string, sourceIds, eventCategories []string) (*EventSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subscriptions[id]; ok {
		return nil, fmt.Errorf("event subscription already exists: %s", id)
	}

	sub := &EventSubscription{
		CustSubscriptionId: id,
		SnsTopicArn:        snsTopicArn,
		SourceType:         sourceType,
		SourceIds:          sourceIds,
		EventCategories:    eventCategories,
		Status:             "active",
		CreatedAt:          time.Now().UTC(),
	}
	s.subscriptions[id] = sub
	return sub, nil
}

// GetEventSubscription retrieves an event subscription by identifier.
func (s *Store) GetEventSubscription(id string) (*EventSubscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subscriptions[id]
	return sub, ok
}

// ListEventSubscriptions returns all event subscriptions.
func (s *Store) ListEventSubscriptions() []*EventSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*EventSubscription, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		out = append(out, sub)
	}
	return out
}

// DeleteEventSubscription removes an event subscription.
func (s *Store) DeleteEventSubscription(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subscriptions[id]; !ok {
		return false
	}
	delete(s.subscriptions, id)
	return true
}

// TestConnection tests connectivity to an endpoint (always succeeds in mock).
func (s *Store) TestConnection(endpointArn, replicationInstanceArn string) (*ConnectionTest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify endpoint exists.
	var foundEP bool
	for _, ep := range s.endpoints {
		if ep.EndpointArn == endpointArn {
			foundEP = true
			break
		}
	}
	if !foundEP {
		return nil, fmt.Errorf("endpoint not found: %s", endpointArn)
	}

	// Verify replication instance exists.
	var foundInst bool
	for _, inst := range s.instances {
		if inst.ReplicationInstanceArn == replicationInstanceArn {
			foundInst = true
			break
		}
	}
	if !foundInst {
		return nil, fmt.Errorf("replication instance not found: %s", replicationInstanceArn)
	}

	ct := ConnectionTest{
		EndpointArn:            endpointArn,
		ReplicationInstanceArn: replicationInstanceArn,
		Status:                 "successful",
		TestedAt:               time.Now().UTC(),
	}
	s.connections = append(s.connections, ct)
	return &ct, nil
}

// DescribeConnections returns all connection test results.
func (s *Store) DescribeConnections() []ConnectionTest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ConnectionTest, len(s.connections))
	copy(result, s.connections)
	return result
}

// GetReplicationTaskStats returns table statistics for a task.
func (s *Store) GetReplicationTaskStats(arn string) (loaded, loading, errored int, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, task := range s.tasks {
		if task.ReplicationTaskArn == arn {
			if task.lifecycle != nil {
				task.Status = string(task.lifecycle.State())
			}
			return task.TablesLoaded, task.TablesLoading, task.TablesErrored, true
		}
	}
	return 0, 0, 0, false
}
