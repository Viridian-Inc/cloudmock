package neptune

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// DBCluster represents a Neptune DB cluster.
type DBCluster struct {
	Identifier     string
	ARN            string
	Engine         string
	EngineVersion  string
	Status         string
	Endpoint       string
	ReaderEndpoint string
	Port           int
	DatabaseName   string
	Tags           map[string]string
	CreatedTime    time.Time
	Lifecycle      *lifecycle.Machine
}

// DBInstance represents a Neptune DB instance.
type DBInstance struct {
	Identifier      string
	ARN             string
	ClusterID       string
	Class           string
	Engine          string
	EngineVersion   string
	Status          string
	Endpoint        DBEndpoint
	Tags            map[string]string
	CreatedTime     time.Time
	Lifecycle       *lifecycle.Machine
	IsWriter        bool   // true for the primary/writer instance
}

// DBEndpoint holds address and port.
type DBEndpoint struct {
	Address string
	Port    int
}

// DBClusterSnapshot represents a cluster snapshot.
type DBClusterSnapshot struct {
	Identifier        string
	ARN               string
	ClusterIdentifier string
	Status            string
	Engine            string
	EngineVersion     string
	SnapshotCreateTime time.Time
	Tags              map[string]string
}

// DBSubnetGroup represents a Neptune subnet group.
type DBSubnetGroup struct {
	Name        string
	ARN         string
	Description string
	SubnetIds   []string
	Status      string
	Tags        map[string]string
}

// DBClusterParameterGroup represents a parameter group.
type DBClusterParameterGroup struct {
	Name        string
	ARN         string
	Family      string
	Description string
	Tags        map[string]string
}

// Store manages all Neptune resources.
type Store struct {
	mu              sync.RWMutex
	clusters        map[string]*DBCluster
	instances       map[string]*DBInstance
	snapshots       map[string]*DBClusterSnapshot
	subnetGroups    map[string]*DBSubnetGroup
	parameterGroups map[string]*DBClusterParameterGroup
	accountID       string
	region          string
	lcConfig        *lifecycle.Config
}

// NewStore creates a new Neptune store.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:        make(map[string]*DBCluster),
		instances:       make(map[string]*DBInstance),
		snapshots:       make(map[string]*DBClusterSnapshot),
		subnetGroups:    make(map[string]*DBSubnetGroup),
		parameterGroups: make(map[string]*DBClusterParameterGroup),
		accountID:       accountID,
		region:          region,
		lcConfig:        lifecycle.DefaultConfig(),
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) clusterARN(id string) string {
	return fmt.Sprintf("arn:aws:neptune:%s:%s:cluster:%s", s.region, s.accountID, id)
}
func (s *Store) instanceARN(id string) string {
	return fmt.Sprintf("arn:aws:neptune:%s:%s:db:%s", s.region, s.accountID, id)
}
func (s *Store) snapshotARN(id string) string {
	return fmt.Sprintf("arn:aws:neptune:%s:%s:cluster-snapshot:%s", s.region, s.accountID, id)
}
func (s *Store) subnetGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:neptune:%s:%s:subgrp:%s", s.region, s.accountID, name)
}
func (s *Store) parameterGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:neptune:%s:%s:cluster-pg:%s", s.region, s.accountID, name)
}

func clusterTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 5 * time.Second},
		{From: "deleting", To: "deleted", Delay: 3 * time.Second},
	}
}

func instanceTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 5 * time.Second},
		{From: "deleting", To: "deleted", Delay: 3 * time.Second},
	}
}

// ---- Cluster operations ----

func (s *Store) CreateDBCluster(id, engine, engineVersion, dbName string, tags map[string]string) (*DBCluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clusters[id]; ok {
		return nil, false
	}
	if engine == "" {
		engine = "neptune"
	}
	c := &DBCluster{
		Identifier:     id,
		ARN:            s.clusterARN(id),
		Engine:         engine,
		EngineVersion:  engineVersion,
		Status:         "creating",
		Endpoint:       fmt.Sprintf("%s.cluster-%s.%s.neptune.amazonaws.com", id, randomHex(8), s.region),
		ReaderEndpoint: fmt.Sprintf("%s.cluster-ro-%s.%s.neptune.amazonaws.com", id, randomHex(8), s.region),
		Port:           8182,
		DatabaseName:   dbName,
		Tags:           tags,
		CreatedTime:    time.Now().UTC(),
		Lifecycle:      lifecycle.NewMachine("creating", clusterTransitions(), s.lcConfig),
	}
	s.clusters[id] = c
	return c, true
}

func (s *Store) GetDBCluster(id string) (*DBCluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[id]
	if ok {
		c.Status = string(c.Lifecycle.State())
	}
	return c, ok
}

func (s *Store) ListDBClusters(filterID string) []*DBCluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DBCluster, 0)
	for _, c := range s.clusters {
		c.Status = string(c.Lifecycle.State())
		if filterID == "" || c.Identifier == filterID {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) ModifyDBCluster(id, engineVersion string) (*DBCluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[id]
	if !ok {
		return nil, false
	}
	if engineVersion != "" {
		c.EngineVersion = engineVersion
	}
	return c, true
}

func (s *Store) DeleteDBCluster(id string) (*DBCluster, bool) {
	s.mu.Lock()
	c, ok := s.clusters[id]
	if !ok {
		s.mu.Unlock()
		return nil, false
	}
	lc := c.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState("deleting")
	}
	return c, true
}

// ---- Instance operations ----

func (s *Store) CreateDBInstance(id, clusterID, class, engine, engineVersion string, tags map[string]string) (*DBInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.instances[id]; ok {
		return nil, false
	}
	if engine == "" {
		engine = "neptune"
	}
	// Determine if this is the first (writer) instance for this cluster.
	isWriter := true
	for _, existing := range s.instances {
		if existing.ClusterID == clusterID {
			isWriter = false
			break
		}
	}

	inst := &DBInstance{
		Identifier:    id,
		ARN:           s.instanceARN(id),
		ClusterID:     clusterID,
		Class:         class,
		Engine:        engine,
		EngineVersion: engineVersion,
		Status:        "creating",
		Endpoint: DBEndpoint{
			Address: fmt.Sprintf("%s.%s.%s.neptune.amazonaws.com", id, randomHex(8), s.region),
			Port:    8182,
		},
		Tags:        tags,
		CreatedTime: time.Now().UTC(),
		Lifecycle:   lifecycle.NewMachine("creating", instanceTransitions(), s.lcConfig),
		IsWriter:    isWriter,
	}
	s.instances[id] = inst
	return inst, true
}

func (s *Store) GetDBInstance(id string) (*DBInstance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inst, ok := s.instances[id]
	if ok {
		inst.Status = string(inst.Lifecycle.State())
	}
	return inst, ok
}

func (s *Store) ListDBInstances(filterID string) []*DBInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DBInstance, 0)
	for _, inst := range s.instances {
		inst.Status = string(inst.Lifecycle.State())
		if filterID == "" || inst.Identifier == filterID {
			result = append(result, inst)
		}
	}
	return result
}

func (s *Store) ModifyDBInstance(id, class string) (*DBInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst, ok := s.instances[id]
	if !ok {
		return nil, false
	}
	if class != "" {
		inst.Class = class
	}
	return inst, true
}

func (s *Store) DeleteDBInstance(id string) (*DBInstance, bool) {
	s.mu.Lock()
	inst, ok := s.instances[id]
	if !ok {
		s.mu.Unlock()
		return nil, false
	}
	lc := inst.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState("deleting")
	}
	return inst, true
}

// ---- Snapshot operations ----

func (s *Store) CreateDBClusterSnapshot(snapID, clusterID string, tags map[string]string) (*DBClusterSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[snapID]; ok {
		return nil, false
	}
	c, ok := s.clusters[clusterID]
	if !ok {
		return nil, false
	}
	snap := &DBClusterSnapshot{
		Identifier: snapID, ARN: s.snapshotARN(snapID), ClusterIdentifier: clusterID,
		Status: "available", Engine: c.Engine, EngineVersion: c.EngineVersion,
		SnapshotCreateTime: time.Now().UTC(), Tags: tags,
	}
	s.snapshots[snapID] = snap
	return snap, true
}

func (s *Store) ListDBClusterSnapshots(clusterID, snapID string) []*DBClusterSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DBClusterSnapshot, 0)
	for _, snap := range s.snapshots {
		if snapID != "" && snap.Identifier != snapID {
			continue
		}
		if clusterID != "" && snap.ClusterIdentifier != clusterID {
			continue
		}
		result = append(result, snap)
	}
	return result
}

func (s *Store) DeleteDBClusterSnapshot(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[id]; !ok {
		return false
	}
	delete(s.snapshots, id)
	return true
}

// ---- SubnetGroup operations ----

func (s *Store) CreateDBSubnetGroup(name, desc string, subnetIDs []string, tags map[string]string) (*DBSubnetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; ok {
		return nil, false
	}
	sg := &DBSubnetGroup{Name: name, ARN: s.subnetGroupARN(name), Description: desc, SubnetIds: subnetIDs, Status: "Complete", Tags: tags}
	s.subnetGroups[name] = sg
	return sg, true
}

func (s *Store) ListDBSubnetGroups(filterName string) []*DBSubnetGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DBSubnetGroup, 0)
	for _, sg := range s.subnetGroups {
		if filterName == "" || sg.Name == filterName {
			result = append(result, sg)
		}
	}
	return result
}

func (s *Store) DeleteDBSubnetGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; !ok {
		return false
	}
	delete(s.subnetGroups, name)
	return true
}

// ---- ParameterGroup operations ----

func (s *Store) CreateDBClusterParameterGroup(name, family, desc string, tags map[string]string) (*DBClusterParameterGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; ok {
		return nil, false
	}
	pg := &DBClusterParameterGroup{Name: name, ARN: s.parameterGroupARN(name), Family: family, Description: desc, Tags: tags}
	s.parameterGroups[name] = pg
	return pg, true
}

func (s *Store) ListDBClusterParameterGroups(filterName string) []*DBClusterParameterGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DBClusterParameterGroup, 0)
	for _, pg := range s.parameterGroups {
		if filterName == "" || pg.Name == filterName {
			result = append(result, pg)
		}
	}
	return result
}

func (s *Store) DeleteDBClusterParameterGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; !ok {
		return false
	}
	delete(s.parameterGroups, name)
	return true
}

// ---- Tag operations ----

func (s *Store) tagMapByARN(arn string) map[string]string {
	for _, c := range s.clusters {
		if c.ARN == arn { return c.Tags }
	}
	for _, i := range s.instances {
		if i.ARN == arn { return i.Tags }
	}
	for _, snap := range s.snapshots {
		if snap.ARN == arn { return snap.Tags }
	}
	for _, sg := range s.subnetGroups {
		if sg.ARN == arn { return sg.Tags }
	}
	for _, pg := range s.parameterGroups {
		if pg.ARN == arn { return pg.Tags }
	}
	return nil
}

func (s *Store) AddTags(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.tagMapByARN(arn)
	if target == nil { return false }
	for k, v := range tags { target[k] = v }
	return true
}

func (s *Store) RemoveTags(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.tagMapByARN(arn)
	if target == nil { return false }
	for _, k := range keys { delete(target, k) }
	return true
}

func (s *Store) ListTags(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target := s.tagMapByARN(arn)
	if target == nil { return nil, false }
	result := make(map[string]string, len(target))
	for k, v := range target { result[k] = v }
	return result, true
}
