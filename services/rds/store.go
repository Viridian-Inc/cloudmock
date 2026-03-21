package rds

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// DBInstance represents an RDS DB instance.
type DBInstance struct {
	Identifier    string
	ARN           string
	Class         string
	Engine        string
	EngineVersion string
	Status        string
	MasterUsername string
	AllocatedStorage int
	Endpoint      DBEndpoint
	CreatedTime   time.Time
	Tags          map[string]string
}

// DBEndpoint holds address and port for a DB instance or cluster.
type DBEndpoint struct {
	Address string
	Port    int
}

// DBCluster represents an RDS DB cluster (Aurora).
type DBCluster struct {
	Identifier     string
	ARN            string
	Engine         string
	EngineVersion  string
	Status         string
	MasterUsername  string
	DatabaseName   string
	Endpoint       string
	ReaderEndpoint string
	Port           int
	Members        []string
	Tags           map[string]string
}

// DBSnapshot represents an RDS DB snapshot.
type DBSnapshot struct {
	Identifier         string
	ARN                string
	DBInstanceIdentifier string
	Status             string
	Engine             string
	EngineVersion      string
	AllocatedStorage   int
	SnapshotCreateTime time.Time
	Tags               map[string]string
}

// DBSubnetGroup represents an RDS DB subnet group.
type DBSubnetGroup struct {
	Name        string
	ARN         string
	Description string
	SubnetIds   []string
	Status      string
	Tags        map[string]string
}

// Store manages all RDS resources.
type Store struct {
	mu           sync.RWMutex
	instances    map[string]*DBInstance    // keyed by identifier
	clusters     map[string]*DBCluster     // keyed by identifier
	snapshots    map[string]*DBSnapshot    // keyed by identifier
	subnetGroups map[string]*DBSubnetGroup // keyed by name
	accountID    string
	region       string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		instances:    make(map[string]*DBInstance),
		clusters:     make(map[string]*DBCluster),
		snapshots:    make(map[string]*DBSnapshot),
		subnetGroups: make(map[string]*DBSubnetGroup),
		accountID:    accountID,
		region:       region,
	}
}

// ---- ARN helpers ----

func (s *Store) instanceARN(id string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:db:%s", s.region, s.accountID, id)
}

func (s *Store) clusterARN(id string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:cluster:%s", s.region, s.accountID, id)
}

func (s *Store) snapshotARN(id string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:snapshot:%s", s.region, s.accountID, id)
}

func (s *Store) subnetGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:rds:%s:%s:subgrp:%s", s.region, s.accountID, name)
}

// ---- port helper ----

func defaultPort(engine string) int {
	switch engine {
	case "postgres", "aurora-postgresql":
		return 5432
	default:
		return 3306
	}
}

// ---- endpoint address helper ----

func (s *Store) endpointAddress(identifier string) string {
	suffix := randomHex(8)
	return fmt.Sprintf("%s.%s.%s.rds.amazonaws.com", identifier, suffix, s.region)
}

// ---- DBInstance operations ----

// CreateDBInstance creates a new DB instance. Returns nil, false if identifier already exists.
func (s *Store) CreateDBInstance(id, class, engine, engineVersion, masterUser string, allocStorage int) (*DBInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.instances[id]; exists {
		return nil, false
	}

	inst := &DBInstance{
		Identifier:       id,
		ARN:              s.instanceARN(id),
		Class:            class,
		Engine:           engine,
		EngineVersion:    engineVersion,
		Status:           "available",
		MasterUsername:   masterUser,
		AllocatedStorage: allocStorage,
		Endpoint: DBEndpoint{
			Address: s.endpointAddress(id),
			Port:    defaultPort(engine),
		},
		CreatedTime: time.Now().UTC(),
		Tags:        make(map[string]string),
	}
	s.instances[id] = inst
	return inst, true
}

// GetDBInstance returns a DB instance by identifier.
func (s *Store) GetDBInstance(id string) (*DBInstance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inst, ok := s.instances[id]
	return inst, ok
}

// ListDBInstances returns all instances, optionally filtered by identifier.
func (s *Store) ListDBInstances(filterID string) []*DBInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*DBInstance, 0)
	for _, inst := range s.instances {
		if filterID == "" || inst.Identifier == filterID {
			result = append(result, inst)
		}
	}
	return result
}

// ModifyDBInstance updates mutable fields on an existing instance. Returns false if not found.
func (s *Store) ModifyDBInstance(id, class string, allocStorage int) (*DBInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, ok := s.instances[id]
	if !ok {
		return nil, false
	}
	if class != "" {
		inst.Class = class
	}
	if allocStorage > 0 {
		inst.AllocatedStorage = allocStorage
	}
	return inst, true
}

// DeleteDBInstance removes an instance. Returns false if not found.
func (s *Store) DeleteDBInstance(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.instances[id]; !ok {
		return false
	}
	delete(s.instances, id)
	return true
}

// ---- DBCluster operations ----

// CreateDBCluster creates a new DB cluster. Returns nil, false if identifier already exists.
func (s *Store) CreateDBCluster(id, engine, engineVersion, masterUser, dbName string) (*DBCluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clusters[id]; exists {
		return nil, false
	}

	port := defaultPort(engine)
	endpoint := fmt.Sprintf("%s.cluster-%s.%s.rds.amazonaws.com", id, randomHex(8), s.region)
	readerEndpoint := fmt.Sprintf("%s.cluster-ro-%s.%s.rds.amazonaws.com", id, randomHex(8), s.region)

	cluster := &DBCluster{
		Identifier:     id,
		ARN:            s.clusterARN(id),
		Engine:         engine,
		EngineVersion:  engineVersion,
		Status:         "available",
		MasterUsername:  masterUser,
		DatabaseName:   dbName,
		Endpoint:       endpoint,
		ReaderEndpoint: readerEndpoint,
		Port:           port,
		Members:        make([]string, 0),
		Tags:           make(map[string]string),
	}
	s.clusters[id] = cluster
	return cluster, true
}

// GetDBCluster returns a DB cluster by identifier.
func (s *Store) GetDBCluster(id string) (*DBCluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[id]
	return c, ok
}

// ListDBClusters returns all clusters, optionally filtered by identifier.
func (s *Store) ListDBClusters(filterID string) []*DBCluster {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*DBCluster, 0)
	for _, c := range s.clusters {
		if filterID == "" || c.Identifier == filterID {
			result = append(result, c)
		}
	}
	return result
}

// DeleteDBCluster removes a cluster. Returns false if not found.
func (s *Store) DeleteDBCluster(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[id]; !ok {
		return false
	}
	delete(s.clusters, id)
	return true
}

// ---- DBSnapshot operations ----

// CreateDBSnapshot creates a snapshot for the given instance. Returns nil, false if snapshot ID already
// exists or the instance does not exist.
func (s *Store) CreateDBSnapshot(snapshotID, instanceID string) (*DBSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.snapshots[snapshotID]; exists {
		return nil, false
	}

	inst, ok := s.instances[instanceID]
	if !ok {
		return nil, false
	}

	snap := &DBSnapshot{
		Identifier:           snapshotID,
		ARN:                  s.snapshotARN(snapshotID),
		DBInstanceIdentifier: instanceID,
		Status:               "available",
		Engine:               inst.Engine,
		EngineVersion:        inst.EngineVersion,
		AllocatedStorage:     inst.AllocatedStorage,
		SnapshotCreateTime:   time.Now().UTC(),
		Tags:                 make(map[string]string),
	}
	s.snapshots[snapshotID] = snap
	return snap, true
}

// GetDBSnapshot returns a snapshot by identifier.
func (s *Store) GetDBSnapshot(id string) (*DBSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.snapshots[id]
	return snap, ok
}

// ListDBSnapshots returns snapshots, optionally filtered by instance ID or snapshot ID.
func (s *Store) ListDBSnapshots(instanceID, snapshotID string) []*DBSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*DBSnapshot, 0)
	for _, snap := range s.snapshots {
		if snapshotID != "" && snap.Identifier != snapshotID {
			continue
		}
		if instanceID != "" && snap.DBInstanceIdentifier != instanceID {
			continue
		}
		result = append(result, snap)
	}
	return result
}

// DeleteDBSnapshot removes a snapshot. Returns false if not found.
func (s *Store) DeleteDBSnapshot(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.snapshots[id]; !ok {
		return false
	}
	delete(s.snapshots, id)
	return true
}

// ---- DBSubnetGroup operations ----

// CreateDBSubnetGroup creates a new subnet group. Returns nil, false if name already exists.
func (s *Store) CreateDBSubnetGroup(name, description string, subnetIDs []string) (*DBSubnetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subnetGroups[name]; exists {
		return nil, false
	}

	sg := &DBSubnetGroup{
		Name:        name,
		ARN:         s.subnetGroupARN(name),
		Description: description,
		SubnetIds:   subnetIDs,
		Status:      "Complete",
		Tags:        make(map[string]string),
	}
	s.subnetGroups[name] = sg
	return sg, true
}

// GetDBSubnetGroup returns a subnet group by name.
func (s *Store) GetDBSubnetGroup(name string) (*DBSubnetGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sg, ok := s.subnetGroups[name]
	return sg, ok
}

// ListDBSubnetGroups returns all subnet groups, optionally filtered by name.
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

// DeleteDBSubnetGroup removes a subnet group. Returns false if not found.
func (s *Store) DeleteDBSubnetGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subnetGroups[name]; !ok {
		return false
	}
	delete(s.subnetGroups, name)
	return true
}

// ---- Tag operations ----

// AddTagsToResource merges tags onto the resource identified by ARN.
func (s *Store) AddTagsToResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.tagMapByARN(arn)
	if target == nil {
		return false
	}
	for k, v := range tags {
		target[k] = v
	}
	return true
}

// RemoveTagsFromResource removes tag keys from a resource.
func (s *Store) RemoveTagsFromResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.tagMapByARN(arn)
	if target == nil {
		return false
	}
	for _, k := range keys {
		delete(target, k)
	}
	return true
}

// ListTagsForResource returns the tag map for a resource by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := s.tagMapByARN(arn)
	if target == nil {
		return nil, false
	}
	// Return a copy.
	result := make(map[string]string, len(target))
	for k, v := range target {
		result[k] = v
	}
	return result, true
}

// tagMapByARN locates the tag map for a resource by ARN.
// Must be called with s.mu held.
func (s *Store) tagMapByARN(arn string) map[string]string {
	for _, inst := range s.instances {
		if inst.ARN == arn {
			return inst.Tags
		}
	}
	for _, c := range s.clusters {
		if c.ARN == arn {
			return c.Tags
		}
	}
	for _, snap := range s.snapshots {
		if snap.ARN == arn {
			return snap.Tags
		}
	}
	for _, sg := range s.subnetGroups {
		if sg.ARN == arn {
			return sg.Tags
		}
	}
	return nil
}

// ---- utility ----

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
